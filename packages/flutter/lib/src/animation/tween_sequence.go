package animation

// Port of packages/flutter/lib/src/animation/tween_sequence.dart.

import (
	"strconv"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// TweenSequence is an Animatable built from a sequence of tweens, each
// over its share (weight) of the [0, 1] interval.
type TweenSequence[T any] struct {
	AnimatableBase[T]
	items     []TweenSequenceItem[T]
	intervals []_Interval
}

// NewTweenSequence creates a sequence of items (at least one, with total
// weight > 0).
func NewTweenSequence[T any](items []TweenSequenceItem[T]) *TweenSequence[T] {
	s := &TweenSequence[T]{}
	s.init(items)
	s.SetSelf(s)
	return s
}

func (s *TweenSequence[T]) init(items []TweenSequenceItem[T]) {
	assert(len(items) > 0, nil)
	s.items = append(s.items, items...)
	totalWeight := 0.0
	for _, item := range s.items {
		totalWeight += item.Weight
	}
	assert(totalWeight > 0.0, nil)
	start := 0.0
	for i := range s.items {
		end := 1.0
		if i != len(s.items)-1 {
			end = start + s.items[i].Weight/totalWeight
		}
		s.intervals = append(s.intervals, newInterval(start, end))
		start = end
	}
}

func (s *TweenSequence[T]) evaluateAt(t float64, index int) T {
	element := s.items[index]
	tInterval := s.intervals[index].value(t)
	return element.Tween.Transform(tInterval)
}

func (s *TweenSequence[T]) Transform(t float64) T {
	assert(t >= 0.0 && t <= 1.0, nil)
	if t == 1.0 {
		return s.evaluateAt(t, len(s.items)-1)
	}
	for index := range s.items {
		if s.intervals[index].contains(t) {
			return s.evaluateAt(t, index)
		}
	}
	// Should be unreachable.
	panic(dart.StateError{Message: "TweenSequence.evaluate() could not find an interval for " + dart.DoubleToString(t)})
}

func (s *TweenSequence[T]) String() string {
	return "TweenSequence(" + strconv.Itoa(len(s.items)) + " items)"
}

// FlippedTweenSequence is a TweenSequence evaluated flipped: Transform(t)
// is 1 - TweenSequence.Transform(1 - t). For curves of TweenSequences.
type FlippedTweenSequence struct{ TweenSequence[float64] }

// NewFlippedTweenSequence creates a flipped sequence of items.
func NewFlippedTweenSequence(items []TweenSequenceItem[float64]) *FlippedTweenSequence {
	s := &FlippedTweenSequence{}
	s.init(items)
	s.SetSelf(s)
	return s
}

func (s *FlippedTweenSequence) Transform(t float64) float64 {
	return 1 - s.TweenSequence.Transform(1-t)
}

// TweenSequenceItem is a simple holder for one element of a TweenSequence.
type TweenSequenceItem[T any] struct {
	// Tween defines the value of the sequence over its interval.
	Tween Animatable[T]
	// Weight is this item's share of the sequence's duration (> 0).
	Weight float64
}

// NewTweenSequenceItem creates an item (tween and weight are Dart's
// required named parameters), asserting weight > 0.
func NewTweenSequenceItem[T any](tween Animatable[T], weight float64) TweenSequenceItem[T] {
	assert(weight > 0.0, nil)
	return TweenSequenceItem[T]{Tween: tween, Weight: weight}
}

type _Interval struct{ start, end float64 }

func newInterval(start, end float64) _Interval {
	assert(end > start, nil)
	return _Interval{start, end}
}

func (i _Interval) contains(t float64) bool { return t >= i.start && t < i.end }

func (i _Interval) value(t float64) float64 { return (t - i.start) / (i.end - i.start) }

func (i _Interval) String() string {
	return "<" + dart.DoubleToString(i.start) + ", " + dart.DoubleToString(i.end) + ">"
}
