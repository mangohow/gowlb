package collection

import "reflect"

type queue[T any] struct {
	elems     []T
	isPointer bool
}

func NewQueue[T any]() Queue[T] {
	var temp T
	return &queue[T]{
		isPointer: reflect.TypeOf(temp).Kind() == reflect.Ptr,
	}
}

func (q *queue[T]) Push(e T) {
	q.elems = append(q.elems, e)
}

func (q *queue[T]) Pop() T {
	if len(q.elems) == 0 {
		return *new(T)
	}

	e := q.elems[0]
	if q.isPointer {
		q.elems[0] = *new(T)
	}
	q.elems = q.elems[1:]

	return e
}

func (q *queue[T]) Peek() T {
	if len(q.elems) == 0 {
		return *new(T)
	}

	return q.elems[0]
}

func (q *queue[T]) Size() int {
	return len(q.elems)
}

func (q *queue[T]) Empty() bool {
	return len(q.elems) == 0
}

func (q *queue[T]) Clear() {
	q.elems = nil
}
