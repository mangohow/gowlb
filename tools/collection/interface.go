package collection

import (
	"sync"
	"time"
)

type ConcurrentMap[K comparable, V any] interface {
	Set(key K, v V)
	Get(key K) (V, bool)
	GetBatch(keys []K) []V
	Delete(key K)
	Has(key K) bool
	Keys() []K
	KeysSet() Set[K]
	Values() []V
	Merge(ConcurrentMap[K, V])
	Clone() ConcurrentMap[K, V]
	ToMap() map[K]V
	MergeMap(map[K]V)
	Len() int
}

type PLM[K comparable, V any] interface {
	// Pointer 获取地址
	Pointer() uintptr
	// RWLock 获取内部的RWMutex
	RWLock() *sync.RWMutex
	// InnerMap 获取内部map
	InnerMap() map[K]V
}

type Queue[T any] interface {
	Push(T)
	Pop() T
	Peek() T
	Size() int
	Empty() bool
	Clear()
}

type Stack[T any] interface {
	Queue[T]
}

type BlockingQueue[T any] interface {
	Push(T)
	Pop() (item T, shutdown bool)
	Size() int
	Empty() bool
	Shutdown()
	ShutdownWithDrained()
}

type DelayingQueue[T any] interface {
	BlockingQueue[T]
	PushAfter(T, time.Duration)
	Shutdown()
}

type ExpirationMap[K comparable, V any] interface {
	ConcurrentMap[K, V]
	SetExpired(key K, val V, duration time.Duration)
	Destroy()
}
