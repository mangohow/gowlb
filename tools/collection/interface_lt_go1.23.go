//go:build !go1.23

package collection

type Set[V comparable] interface {
	Add(v V)
	Adds(v ...V)
	AddSet(s Set[V])
	Has(v V) bool
	Delete(v V)
	Values() []V
	Any([]V) bool
	Every([]V) bool
	ForEach(func(v V))
	ForEachP(func(v *V))
	Len() int
}
