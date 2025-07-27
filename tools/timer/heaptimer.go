package timer

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/mangohow/gowlb/tools/collection"
)

type HeapTimer struct {
	timers    collection.Queue[*timer]
	addCh     chan *timer
	modCh     chan modTimer
	removeCh  chan *timer
	triggerFn func(t *timer)
}

type modTimer struct {
	t *timer
	d time.Duration
}

// 执行任务时使用协程池
func NewExecutePoolHeapTimer(poolSize int) TimerScheduler {
	t := &HeapTimer{
		addCh:    make(chan *timer, 1024),
		modCh:    make(chan modTimer, 1024),
		removeCh: make(chan *timer, 1024),
		timers: collection.NewPriorityQueue[*timer](func(a, b *timer) bool {
			return a.trigger.Before(b.trigger)
		}),
	}

	once := sync.Once{}
	ch := make(chan *timer, 1024)
	// 触发回调函数时，通过协程池去处理
	var workerFunc func()
	workerFunc = func() {
		defer func() {
			if r := recover(); r != nil {
				go workerFunc()
			}
		}()

		for {
			t := <-ch
			if t.fn != nil {
				t.fn()
			}
		}
	}
	t.triggerFn = func(t *timer) {
		once.Do(func() {
			for i := 0; i < poolSize; i++ {
				go workerFunc()
			}
		})

		select {
		case ch <- t:
		default:
			go func() {
				ch <- t
			}()
		}
	}

	go t.tick()

	return t
}

// 执行任务时启动一个goroutine
func NewAsyncHeapTimer() TimerScheduler {
	t := &HeapTimer{
		addCh:    make(chan *timer, 1024),
		modCh:    make(chan modTimer, 1024),
		removeCh: make(chan *timer, 1024),
		timers: collection.NewPriorityQueue[*timer](func(a, b *timer) bool {
			return a.trigger.Before(b.trigger)
		}),
	}

	// 触发回调函数时，直接启动一个Goroutine去处理
	t.triggerFn = func(t *timer) {
		if t.fn != nil {
			go func() {
				defer func() {
					recover()
				}()
				t.fn()
			}()
		}
	}

	go t.tick()

	return t
}

// 执行任务时同步调用, 适用于立刻就能执行完成的任务, 不适合耗时任务, 会阻塞其他任务
func NewSyncHeapTimer() TimerScheduler {
	t := &HeapTimer{
		addCh:    make(chan *timer, 1024),
		modCh:    make(chan modTimer, 1024),
		removeCh: make(chan *timer, 1024),
		timers: collection.NewPriorityQueue[*timer](func(a, b *timer) bool {
			return a.trigger.Before(b.trigger)
		}),
	}

	// 触发回调函数时，直接同步处理
	t.triggerFn = func(t *timer) {
		defer func() {
			recover()
		}()

		if t.fn != nil {
			t.fn()
		}
	}

	go t.tick()

	return t
}

// 通过一个goroutine来处理所有定时器的添加、重置、删除以及定时器的触发
// 采取无锁化的方法
func (h *HeapTimer) tick() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "tick panic: %v\n%s", r, string(debug.Stack()))
			go h.tick()
		}
	}()

	var (
		never            = make(<-chan time.Time)
		nextTriggerTimer *time.Timer

		// 使用非阻塞的方式添加，防止阻塞该goroutine
		addFn = func(t *timer) {
			select {
			case h.addCh <- t:
			default:
				go func() {
					h.addCh <- t
				}()
			}
		}
	)

	for {
		now := time.Now()
		for h.timers.Size() > 0 {
			m := h.timers.Peek()
			// 如果堆顶的定时器没有触发, 设置最小触发时间，等下次触发
			if m.trigger.After(now) {
				break
			}

			// 定时器触发了，从堆中删除
			h.timers.Pop()
			// 定时器触发，检查是否被删除了
			// 如果被删除了，从堆中移除，并且再次检查堆顶
			if m.removed {
				continue
			}

			// 执行回调函数
			h.triggerFn(m)

			// 如果是ticker，则重新添加到定时器集合中
			if m.ticker {
				m.trigger = now.Add(m.duration)
				addFn(m)
			}
		}
		nextTrigger := never
		if h.timers.Size() > 0 {
			if nextTriggerTimer != nil {
				nextTriggerTimer.Stop()
			}
			top := h.timers.Peek()
			nextTriggerTimer = time.NewTimer(top.duration)
			nextTrigger = nextTriggerTimer.C
		}

		select {
		// 向堆中添加一个定时器
		case t := <-h.addCh:
			// 1. 如果添加的是第一个定时器，那么它肯定是最早触发的，需要重置定时器
			// 2. 和堆顶的最小触发定时器进行比较，如果刚添加的更早触发，则需要重置系统定时器
			if t.trigger.Before(now) {
				h.triggerFn(t)
				break
			}
			h.timers.Push(t)
			drained := false
			for !drained {
				select {
				case t := <-h.addCh:
					if t.trigger.Before(time.Now()) {
						h.triggerFn(t)
					} else {
						h.timers.Push(t)
					}
				default:
					drained = true
				}
			}
		// 删除原来的定时器，重新添加一个
		case t := <-h.modCh:
			tt := *t.t
			tt.duration = t.d
			tt.trigger = time.Now().Add(t.d)
			t.t.removed = true
			addFn(&tt)
		// 删除时，只设置删除标志
		case t := <-h.removeCh:
			t.removed = true
		case <-nextTrigger:
		}
	}

}

func (h *HeapTimer) add(d time.Duration, fn func(), ticker bool) *timer {
	if fn == nil {
		panic("nil function")
	}
	if d < 0 {
		panic("negative duration")
	}

	tm := &timer{
		h:        h,
		fn:       fn,
		ticker:   ticker,
		trigger:  time.Now().Add(d),
		duration: d,
	}

	h.addCh <- tm

	return tm
}

func (h *HeapTimer) remove(t *timer) {
	h.removeCh <- t
}

func (h *HeapTimer) reset(tm *timer, duration time.Duration) {
	if duration < 0 {
		panic("negative duration")
	}
	h.modCh <- modTimer{
		t: tm,
		d: duration,
	}
}

func (h *HeapTimer) SetTimer(duration time.Duration, fn func()) Timer {
	return h.add(duration, fn, false)
}

func (h *HeapTimer) SetTicker(duration time.Duration, fn func()) Ticker {
	return h.add(duration, fn, true)
}

type timer struct {
	h        *HeapTimer
	fn       func()
	ticker   bool
	trigger  time.Time
	duration time.Duration
	removed  bool
}

func (t *timer) Stop() {
	t.h.remove(t)
}

func (t *timer) Reset(duration time.Duration) {
	t.h.reset(t, duration)
}
