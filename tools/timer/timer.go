package timer

import "time"

type TimerScheduler interface {
	// SetTimer 设置一次性执行的定时器
	SetTimer(duration time.Duration, task func()) Timer
	// SetTicker 设置定期执行的定时器
	SetTicker(duration time.Duration, task func()) Ticker
	// Shutdown 关闭定时器调度器, 释放goroutine等资源
	Shutdown()
	IsShutdown() bool
}

// Timer 对Timer来说, Stop不是必要的, 因为当定时器到期会自动解除引用, 以进行垃圾回收
// 如果需要停止回调函数的调用，则调用Stop是必要的
type Timer interface {
	Stop()
	Reset(duration time.Duration)
}

// Ticker 对Ticker来说, 如果需要停止Ticker, 则调用Stop是必要的, 不然该定时器会一直重复执行
type Ticker interface {
	Stop()
}
