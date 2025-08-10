package collection

import (
	"time"
)

type TimerScheduler interface {
	// SetTimer 设置一次性执行的定时器
	SetTimer(duration time.Duration, fn func()) Timer
	Shutdown()
	IsShutdown() bool
}

type Timer interface {
	Stop()
	Reset(duration time.Duration)
}

type delayingQueue[T any] struct {
	BlockingQueue[T]
	scheduler TimerScheduler
}

func NewDelayingQueue[T any](blocking BlockingQueue[T], scheduler TimerScheduler) DelayingQueue[T] {
	return &delayingQueue[T]{
		BlockingQueue: blocking,
		scheduler:     scheduler,
	}
}

func (d *delayingQueue[T]) PushAfter(e T, duration time.Duration) {
	if duration < 0 {
		panic("negative duration")
	}
	if duration == 0 {
		d.Push(e)
		return
	}

	if d.scheduler.IsShutdown() {
		return
	}

	d.scheduler.SetTimer(duration, func() {
		d.Push(e)
	})
}

func (d *delayingQueue[T]) Shutdown() {
	d.BlockingQueue.Shutdown()
	d.scheduler.Shutdown()
}
