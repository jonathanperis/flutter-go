package foundation

import (
	"iter"
	"slices"
)

// ObserverList is a list optimized for the observer pattern: fast
// contains, and iteration over a stable snapshot order.
type ObserverList[T comparable] struct {
	list  []T
	dirty bool
	set   map[T]struct{}
}

func (l *ObserverList[T]) Add(item T) {
	l.dirty = true
	l.list = append(l.list, item)
}

// Remove removes the first occurrence of item and reports whether it was
// present.
func (l *ObserverList[T]) Remove(item T) bool {
	i := slices.Index(l.list, item)
	if i < 0 {
		return false
	}
	l.list = slices.Delete(l.list, i, i+1)
	l.dirty = true
	clear(l.set) // so we don't leak items
	return true
}

func (l *ObserverList[T]) Clear() {
	l.dirty = false
	l.list = nil
	clear(l.set)
}

func (l *ObserverList[T]) Contains(element T) bool {
	if len(l.list) < 3 {
		return slices.Contains(l.list, element)
	}
	if l.dirty {
		if l.set == nil {
			l.set = map[T]struct{}{}
		}
		for _, v := range l.list {
			l.set[v] = struct{}{}
		}
		l.dirty = false
	}
	_, ok := l.set[element]
	return ok
}

func (l *ObserverList[T]) All() iter.Seq[T] { return slices.Values(l.list) }
func (l *ObserverList[T]) IsEmpty() bool    { return len(l.list) == 0 }
func (l *ObserverList[T]) IsNotEmpty() bool { return len(l.list) != 0 }
func (l *ObserverList[T]) ToList() []T      { return slices.Clone(l.list) }

// HashedObserverList is an ObserverList with constant-time add and remove;
// an item added n times must be removed n times. Iteration follows first
// insertion order, like Dart's LinkedHashMap.
type HashedObserverList[T comparable] struct {
	counts map[T]int
	order  []T
}

func (l *HashedObserverList[T]) Add(item T) {
	if l.counts == nil {
		l.counts = map[T]int{}
	}
	if l.counts[item] == 0 {
		l.order = append(l.order, item)
	}
	l.counts[item]++
}

func (l *HashedObserverList[T]) Remove(item T) bool {
	n := l.counts[item]
	switch n {
	case 0:
		return false
	case 1:
		delete(l.counts, item)
		l.order = slices.DeleteFunc(l.order, func(v T) bool { return v == item })
	default:
		l.counts[item] = n - 1
	}
	return true
}

func (l *HashedObserverList[T]) Clear()                  { clear(l.counts); l.order = nil }
func (l *HashedObserverList[T]) Contains(element T) bool { return l.counts[element] > 0 }
func (l *HashedObserverList[T]) All() iter.Seq[T]        { return slices.Values(slices.Clone(l.order)) }
func (l *HashedObserverList[T]) IsEmpty() bool           { return len(l.order) == 0 }
func (l *HashedObserverList[T]) IsNotEmpty() bool        { return len(l.order) != 0 }
func (l *HashedObserverList[T]) ToList() []T             { return slices.Clone(l.order) }
