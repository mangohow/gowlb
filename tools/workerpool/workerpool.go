package workerpool

import (
	"errors"
	"fmt"
	"github.com/mangohow/gowlb/tools/sync"
	"os"
	"runtime"
	stdsync "sync"
	"sync/atomic"
	"time"
)

type WorkerPool interface {
	Submit(task func()) error
	WorkerCount() int
	QueueSize() int
	Start() error
	Shutdown(drain bool)
	ShutdownWait(drain bool)
}

type RejectPolicy func(pool WorkerPool, task func(), submit func() error) error

var (
	// 创建新的goroutine处理
	NewProcRunsPolicy = func() RejectPolicy {
		return func(pool WorkerPool, task func(), submit func() error) error {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						if p, ok := pool.(*workerPool); ok {
							p.handleCrash(r)
						}
					}
				}()

				task()
			}()
			return nil
		}
	}

	// 调用方处理
	CallerRunsPolicy = func() RejectPolicy {
		return func(pool WorkerPool, task func(), submit func() error) error {
			defer func() {
				if r := recover(); r != nil {
					if p, ok := pool.(*workerPool); ok {
						p.handleCrash(r)
					}
				}
			}()

			task()
			return nil
		}
	}

	// 返回错误
	AbortPolicy = func() RejectPolicy {
		return func(pool WorkerPool, task func(), submit func() error) error {
			return TaskQueueFullErr
		}
	}

	// 直接丢弃任务, 不返回错误
	DiscardPolicy = func() RejectPolicy {
		return func(pool WorkerPool, task func(), submit func() error) error {
			return nil
		}
	}

	// 休眠一段时间重试几次, 如果还不行, 则使用传入的拒绝策略
	SubmitAfterwardsPolicy = func(reties int, wait time.Duration, rejectPolicy RejectPolicy) RejectPolicy {
		return func(pool WorkerPool, task func(), submit func() error) error {
			var err error
			for i := 0; i < reties; i++ {
				time.Sleep(wait)
				err = submit()
				if err == TaskQueueFullErr {
					continue
				}

				return err
			}

			if rejectPolicy != nil {
				return rejectPolicy(pool, task, submit)
			}

			return err
		}
	}
)

const (
	stateInit = iota
	stateRunning
	stateDrain
	stateShutdown
)

type workerPool struct {
	workerCount   atomic.Int32
	state         atomic.Int32
	growingMux    stdsync.Mutex
	minWorker     int
	maxWorker     int
	aliveDuration time.Duration
	taskChan      chan func()
	wg            sync.WaitGroup
	rejectPolicy  RejectPolicy
	panicHandler  func(r any, stack []byte)
}

type Option func(*workerPool)

func WithAliveDuration(duration time.Duration) Option {
	return func(pool *workerPool) {
		pool.aliveDuration = duration
	}
}

func WithRejectPolicy(rejectPolicy RejectPolicy) Option {
	return func(pool *workerPool) {
		pool.rejectPolicy = rejectPolicy
	}
}

func WithPanicHandler(panicHandler func(any, []byte)) Option {
	return func(pool *workerPool) {
		pool.panicHandler = panicHandler
	}
}

func NewWorkerPool(minWorker, maxWorker, chanSize int, opts ...Option) WorkerPool {
	if minWorker < 0 || maxWorker < 0 || minWorker > maxWorker || chanSize < 0 {
		panic("invalid parameter")
	}

	if chanSize < 64 {
		chanSize = 64
	}

	pool := &workerPool{
		minWorker: minWorker,
		maxWorker: maxWorker,
		taskChan:  make(chan func(), chanSize),
	}

	for _, opt := range opts {
		opt(pool)
	}

	if pool.rejectPolicy == nil {
		pool.rejectPolicy = AbortPolicy()
	}

	if pool.aliveDuration < 0 {
		panic("invalid aliveDuration")
	} else if pool.aliveDuration == 0 {
		pool.aliveDuration = time.Minute * 5
	}

	return pool
}

var (
	SubmitShutdownWorkerPoolErr = errors.New("work pool has been shutdown")
	TaskQueueFullErr            = errors.New("task queue is full")
	WorkerPoolStateInvalidError = errors.New("work pool state is invalid")
)

func (w *workerPool) Submit(task func()) error {
	if w.state.Load() != stateRunning {
		return SubmitShutdownWorkerPoolErr
	}

	// 如果队列中任务数量大于3, 则创建worker
	if len(w.taskChan) > 3 && int(w.workerCount.Load()) < w.maxWorker {
		w.growingMux.Lock()
		if int(w.workerCount.Load()) < w.maxWorker {
			w.workerCount.Add(1)
			w.wg.Go(w.worker)
		}
		w.growingMux.Unlock()
	}

	select {
	case w.taskChan <- task:
		return nil
	default:
	}

	return w.rejectPolicy(w, task, func() error {
		if w.state.Load() != stateRunning {
			return SubmitShutdownWorkerPoolErr
		}
		select {
		case w.taskChan <- task:
			return nil
		default:
			return TaskQueueFullErr
		}
	})
}

func (w *workerPool) WorkerCount() int {
	return int(w.workerCount.Load())
}

func (w *workerPool) QueueSize() int {
	return len(w.taskChan)
}

func (w *workerPool) Start() error {
	if w.state.Load() == stateRunning {
		return nil
	}

	if w.state.Load() != stateInit {
		return WorkerPoolStateInvalidError
	}

	w.state.Store(stateRunning)
	for i := 0; i < w.minWorker; i++ {
		w.workerCount.Add(1)
		w.wg.Go(w.worker)
	}

	return nil
}

func (w *workerPool) Shutdown(drain bool) {
	if w.state.Load() != stateRunning {
		return
	}

	newState := stateShutdown
	if drain {
		newState = stateDrain
	}

	w.state.Store(int32(newState))
	close(w.taskChan)
}

func (w *workerPool) ShutdownWait(drain bool) {
	if w.state.Load() != stateRunning {
		return
	}

	w.Shutdown(drain)
	w.wg.Wait()
}

func (w *workerPool) worker() {
	defer w.workerCount.Add(-1)
	timer := time.NewTimer(w.aliveDuration)
	defer timer.Stop()

	for {
		if w.state.Load() == stateShutdown {
			return
		}

		select {
		case task, valid := <-w.taskChan:
			// 通道被关闭
			if !valid {
				w.state.Store(stateShutdown)
				return
			}
			w.executeTask(task)
		case <-timer.C:
			if int(w.workerCount.Load()) > w.minWorker {
				return
			}
		}

		timer.Reset(w.aliveDuration)
	}

}

func (w *workerPool) executeTask(task func()) {
	defer func() {
		if r := recover(); r != nil {
			w.handleCrash(r)
		}
	}()

	task()
}

func (w *workerPool) handleCrash(r any) {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]

	if handler := w.panicHandler; handler != nil {
		handler(r, buf)
		return
	}

	// 打印到 stderr
	_, _ = fmt.Fprintf(os.Stderr, "worker: panic recovered: %v\n%s\n", r, buf)
}
