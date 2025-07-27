package collection

import (
	"container/heap"
	"reflect"
)

type priorityQueue[T any] struct {
	que heapQueue[T]
}

func NewPriorityQueue[T any](less func(a, b T) bool) Queue[T] {
	var temp T
	return &priorityQueue[T]{
		que: heapQueue[T]{
			less:      less,
			isPointer: reflect.TypeOf(temp).Kind() == reflect.Ptr,
		},
	}
}

func (p *priorityQueue[T]) Push(e T) {
	heap.Push(&p.que, e)
}

func (p *priorityQueue[T]) Pop() T {
	if p.que.Len() == 0 {
		return *new(T)
	}
	return heap.Pop(&p.que).(T)
}

func (p *priorityQueue[T]) Peek() T {
	if p.que.Len() == 0 {
		return *new(T)
	}

	return p.que.arr[0]
}

func (p *priorityQueue[T]) Size() int {
	return len(p.que.arr)
}

func (p *priorityQueue[T]) Empty() bool {
	return len(p.que.arr) == 0
}

func (p *priorityQueue[T]) Clear() {
	p.que.arr = nil
}

type heapQueue[T any] struct {
	arr       []T
	less      func(a, b T) bool
	isPointer bool
}

func (h *heapQueue[T]) Len() int {
	return len(h.arr)
}

func (h *heapQueue[T]) Less(i, j int) bool {
	return h.less(h.arr[i], h.arr[j])
}

func (h *heapQueue[T]) Swap(i, j int) {
	h.arr[i], h.arr[j] = h.arr[j], h.arr[i]
}

func (h *heapQueue[T]) Push(x any) {
	h.arr = append(h.arr, x.(T))
}

func (h *heapQueue[T]) Pop() any {
	n := len(h.arr) - 1
	t := h.arr[n]
	// 置为0值, 如果是指针对垃圾回收友好
	if h.isPointer {
		h.arr[n] = *new(T)
	}
	h.arr = h.arr[:n]
	return t
}
