//go:build go1.23

package collection

import "iter"

func (s *set[V]) Iterator() iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range s.set {
			if !yield(v) {
				break
			}
		}
	}
}
