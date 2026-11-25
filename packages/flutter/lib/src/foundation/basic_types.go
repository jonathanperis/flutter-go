package foundation

import (
	"iter"
	"math"
	"reflect"
	"time"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
)

// Callback signatures (basic_types.dart). Dart's Iterable is Go's iter.Seq
// and Future is async.Future.
type (
	ValueChanged[T any]     func(value T)
	ValueSetter[T any]      func(value T)
	ValueGetter[T any]      func() T
	IterableFilter[T any]   func(input iter.Seq[T]) iter.Seq[T]
	AsyncCallback           func() async.Future[async.Void]
	AsyncValueSetter[T any] func(value T) async.Future[async.Void]
	AsyncValueGetter[T any] func() async.Future[T]
)

// CachingIterable lazily pulls from an iterator once and caches the results,
// so iterating it again doesn't re-run the source.
type CachingIterable[E any] struct {
	next    func() (E, bool)
	stop    func()
	results []E
}

// NewCachingIterable wraps a sequence (Dart passes an Iterator).
func NewCachingIterable[E any](seq iter.Seq[E]) *CachingIterable[E] {
	next, stop := iter.Pull(seq)
	return &CachingIterable[E]{next: next, stop: stop}
}

func (c *CachingIterable[E]) fillNext() bool {
	if c.next == nil {
		return false
	}
	v, ok := c.next()
	if !ok {
		c.stop()
		c.next = nil
		return false
	}
	c.results = append(c.results, v)
	return true
}

// All iterates the cached elements, pulling more as needed.
func (c *CachingIterable[E]) All() iter.Seq[E] {
	return func(yield func(E) bool) {
		for i := 0; ; i++ {
			if i == len(c.results) && !c.fillNext() {
				return
			}
			if !yield(c.results[i]) {
				return
			}
		}
	}
}

// Length forces the whole source and returns its size.
func (c *CachingIterable[E]) Length() int {
	for c.fillNext() {
	}
	return len(c.results)
}

// ElementAt returns the element at index, pulling as needed; it panics like
// Dart's RangeError/IndexError when out of range.
func (c *CachingIterable[E]) ElementAt(index int) E {
	if index < 0 {
		panic("foundation: index must not be negative")
	}
	for len(c.results) <= index {
		if !c.fillNext() {
			panic("foundation: index out of range")
		}
	}
	return c.results[index]
}

// ToList forces the whole source and returns a copy of the elements.
func (c *CachingIterable[E]) ToList() []E {
	for c.fillNext() {
	}
	return append([]E(nil), c.results...)
}

// Factory holds a constructor for T (Flutter uses it for gesture
// recognizers).
type Factory[T any] struct {
	Constructor ValueGetter[T]
}

func NewFactory[T any](constructor ValueGetter[T]) Factory[T] { return Factory[T]{constructor} }

// Type is the type the factory builds.
func (f Factory[T]) Type() reflect.Type { return reflect.TypeFor[T]() }

func (f Factory[T]) String() string { return "Factory(type: " + typeName(f.Type()) + ")" }

func typeName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		return typeName(t.Elem())
	}
	return t.Name()
}

// LerpDuration is lerpDuration: linear interpolation in microseconds,
// rounded like Dart's double.round().
func LerpDuration(a, b time.Duration, t float64) time.Duration {
	am, bm := a.Microseconds(), b.Microseconds()
	v := float64(am) + float64(float64(bm-am)*t) // no FMA, as in Dart
	return time.Duration(math.Round(v)) * time.Microsecond
}
