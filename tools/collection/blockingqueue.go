package collection

import (
	"sync"
)

type blockingQueue[T any] struct {
	queue    Queue[T]
	cond     *sync.Cond
	shutdown bool
}

func NewBlockingQueue[T any]() BlockingQueue[T] {
	return &blockingQueue[T]{
		queue: NewQueue[T](),
		cond:  sync.NewCond(&sync.Mutex{}),
	}
}

func NewBlockingQueueWithConfig[T any](queue Queue[T]) BlockingQueue[T] {
	return &blockingQueue[T]{
		queue: queue,
		cond:  sync.NewCond(&sync.Mutex{}),
	}
}

func (b *blockingQueue[T]) Push(e T) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	if b.shutdown {
		return
	}

	b.queue.Push(e)
	b.cond.Signal()
}

func (b *blockingQueue[T]) Pop() (T, bool) {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	for b.queue.Empty() && !b.shutdown {
		b.cond.Wait()
	}

	if b.queue.Empty() {
		return *new(T), true
	}

	item := b.queue.Pop()
	if b.queue.Empty() {
		b.cond.Signal()
	}

	return item, b.shutdown
}

func (b *blockingQueue[T]) Size() int {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	return b.queue.Size()
}

func (b *blockingQueue[T]) Empty() bool {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	return b.queue.Empty()
}

func (b *blockingQueue[T]) Shutdown() {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	b.shutdown = true
	b.cond.Broadcast()
}

func (b *blockingQueue[T]) ShutdownWithDrained() {
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	b.shutdown = true
	b.cond.Broadcast()
	for !b.queue.Empty() {
		b.cond.Wait()
	}
}
