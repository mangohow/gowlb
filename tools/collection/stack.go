package collection

type stack[T any] struct {
	elems     []T
	isPointer bool
}

func (s *stack[T]) Push(e T) {
	s.elems = append(s.elems, e)
}

func (s *stack[T]) Pop() T {
	if len(s.elems) == 0 {
		return *new(T)
	}

	n := len(s.elems) - 1
	e := s.elems[n]
	if s.isPointer {
		s.elems[n] = *new(T)
	}
	s.elems = s.elems[:n]

	return e
}

func (s *stack[T]) Peek() T {
	if len(s.elems) == 0 {
		return *new(T)
	}

	return s.elems[len(s.elems)-1]
}

func (s *stack[T]) Size() int {
	return len(s.elems)
}

func (s *stack[T]) Empty() bool {
	return len(s.elems) == 0
}

func (s *stack[T]) Clear() {
	s.elems = nil
}
