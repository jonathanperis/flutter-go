// Package animation is Flutter's animation library (Flutter 3.50.1):
// Animation and AnimationStatus, AnimationController, the animation
// combinators (ProxyAnimation, ReverseAnimation, CurvedAnimation,
// TrainHoppingAnimation, CompoundAnimation...), Animatables and Tweens,
// TweenSequence, AnimationStyle and the curves. Controllers are driven by
// the scheduler library's Tickers; values match Flutter's frame by frame
// (see the animation_controller and tween difftests).
package animation

// Port of packages/flutter/lib/src/animation/animation.dart.
//
// Animation is Dart's abstract class Animation<T> (which extends Listenable
// and implements ValueListenable<T>): an interface, plus the embeddable
// AnimationBase[T] holding the concrete members (isDismissed..., toString,
// toStringDetails). As everywhere (API_MAPPING "Mixins and overridable
// methods"), AnimationBase calls the overridable members (Status,
// ToStringDetails...) through self: a type embedding it calls SetSelf once.

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// AnimationStatus is the status of an animation.
type AnimationStatus int

const (
	// AnimationStatusDismissed: the animation is stopped at the beginning.
	AnimationStatusDismissed AnimationStatus = iota
	// AnimationStatusForward: the animation is running from beginning to end.
	AnimationStatusForward
	// AnimationStatusReverse: the animation is running backwards, from end
	// to beginning.
	AnimationStatusReverse
	// AnimationStatusCompleted: the animation is stopped at the end.
	AnimationStatusCompleted
)

var animationStatusNames = [...]string{"dismissed", "forward", "reverse", "completed"}

// Name is the Dart enum value's name ("forward").
func (s AnimationStatus) Name() string { return animationStatusNames[s] }

// String is Dart's toString ("AnimationStatus.forward").
func (s AnimationStatus) String() string { return "AnimationStatus." + s.Name() }

// IsDismissed reports whether the animation is stopped at the beginning.
func (s AnimationStatus) IsDismissed() bool { return s == AnimationStatusDismissed }

// IsCompleted reports whether the animation is stopped at the end.
func (s AnimationStatus) IsCompleted() bool { return s == AnimationStatusCompleted }

// IsAnimating reports whether the animation is running (forward or
// reverse).
func (s AnimationStatus) IsAnimating() bool {
	switch s {
	case AnimationStatusForward, AnimationStatusReverse:
		return true
	}
	return false
}

// IsForwardOrCompleted reports whether the current aim of the animation is
// toward completion (forward or completed).
func (s AnimationStatus) IsForwardOrCompleted() bool {
	switch s {
	case AnimationStatusForward, AnimationStatusCompleted:
		return true
	}
	return false
}

// AnimationStatusListener is the signature of listeners that receive
// AnimationStatus updates.
type AnimationStatusListener = func(status AnimationStatus)

// ValueListenableTransformer is the signature of a function that receives
// a value of type T and returns a value of type T (see
// NewAnimationFromValueListenable).
type ValueListenableTransformer[T any] func(value T) T

// Animation is a value which might change over time, moving forward or
// backward, with a status. Implementations embed AnimationBase.
type Animation[T any] interface {
	foundation.ValueListenable[T]
	// AddStatusListener calls listener every time the status changes.
	AddStatusListener(listener AnimationStatusListener)
	// RemoveStatusListener stops calling listener on status changes.
	RemoveStatusListener(listener AnimationStatusListener)
	// Status is the current status of the animation.
	Status() AnimationStatus
	// IsDismissed reports whether the animation is stopped at the beginning.
	IsDismissed() bool
	// IsCompleted reports whether the animation is stopped at the end.
	IsCompleted() bool
	// IsAnimating reports whether the animation is running.
	IsAnimating() bool
	// IsForwardOrCompleted reports whether the animation is aimed at
	// completion.
	IsForwardOrCompleted() bool
	// ToStringDetails is the details of String (by default an arrow for the
	// status).
	ToStringDetails() string
	String() string
}

// AnimationBase is the embeddable implementation of Animation's concrete
// members. The embedding type implements Value, Status and the listener
// methods, and calls SetSelf once when created.
type AnimationBase[T any] struct{ self Animation[T] }

// SetSelf points the base at the animation that embeds it.
func (a *AnimationBase[T]) SetSelf(self Animation[T]) { a.self = self }

func (a *AnimationBase[T]) outer() Animation[T] {
	if a.self == nil {
		panic("animation: AnimationBase used before SetSelf: build animations with their constructors, " +
			"or call SetSelf(outer) once on a type that embeds it")
	}
	return a.self
}

// IsDismissed is Status().IsDismissed().
func (a *AnimationBase[T]) IsDismissed() bool { return a.outer().Status().IsDismissed() }

// IsCompleted is Status().IsCompleted().
func (a *AnimationBase[T]) IsCompleted() bool { return a.outer().Status().IsCompleted() }

// IsAnimating is Status().IsAnimating().
func (a *AnimationBase[T]) IsAnimating() bool { return a.outer().Status().IsAnimating() }

// IsForwardOrCompleted is Status().IsForwardOrCompleted().
func (a *AnimationBase[T]) IsForwardOrCompleted() bool {
	return a.outer().Status().IsForwardOrCompleted()
}

// String is "Type#hash(details)".
func (a *AnimationBase[T]) String() string {
	self := a.outer()
	return foundation.DescribeIdentity(self) + "(" + self.ToStringDetails() + ")"
}

// ToStringDetails is an arrow for the status: ▶ forward, ◀ reverse,
// ⏭ completed, ⏮ dismissed.
func (a *AnimationBase[T]) ToStringDetails() string {
	switch a.outer().Status() {
	case AnimationStatusForward:
		return "▶" // >
	case AnimationStatusReverse:
		return "◀" // <
	case AnimationStatusCompleted:
		return "⏭" // >>|
	}
	return "⏮" // |<<
}

// Drive chains a Tween (or CurveTween) to this animation: Dart's
// animation.drive(child), which is child.animate(animation). (Go methods
// can't take type parameters, so the generic method is a function.)
func Drive[U any](animation Animation[float64], child Animatable[U]) Animation[U] {
	return child.Animate(animation)
}

// AnimationFromValueListenableOptions are
// NewAnimationFromValueListenable's named parameters.
type AnimationFromValueListenableOptions[T any] struct {
	// Transformer, when set, is applied to the listenable's value.
	Transformer ValueListenableTransformer[T]
}

// NewAnimationFromValueListenable is Animation.fromValueListenable: an
// animation that delegates to listenable, with a fixed status (forward).
// The value is Transformer(listenable.Value()) when a transformer is given
// (and its result is not null).
func NewAnimationFromValueListenable[T any](listenable foundation.ValueListenable[T], opts ...AnimationFromValueListenableOptions[T]) Animation[T] {
	a := &_ValueListenableDelegateAnimation[T]{listenable: listenable}
	if len(opts) > 0 {
		a.transformer = opts[0].Transformer
	}
	a.SetSelf(a)
	return a
}

// _ValueListenableDelegateAnimation is an animation that delegates to a
// value listenable, with a fixed direction.
type _ValueListenableDelegateAnimation[T any] struct {
	AnimationBase[T]
	listenable  foundation.ValueListenable[T]
	transformer ValueListenableTransformer[T]
}

func (a *_ValueListenableDelegateAnimation[T]) AddListener(listener func()) {
	a.listenable.AddListener(listener)
}

func (a *_ValueListenableDelegateAnimation[T]) RemoveListener(listener func()) {
	a.listenable.RemoveListener(listener)
}

// AddStatusListener does nothing: the status never changes.
func (a *_ValueListenableDelegateAnimation[T]) AddStatusListener(AnimationStatusListener) {}

// RemoveStatusListener does nothing: the status never changes.
func (a *_ValueListenableDelegateAnimation[T]) RemoveStatusListener(AnimationStatusListener) {}

func (a *_ValueListenableDelegateAnimation[T]) Status() AnimationStatus {
	return AnimationStatusForward
}

func (a *_ValueListenableDelegateAnimation[T]) Value() T {
	if a.transformer != nil {
		if v := a.transformer(a.listenable.Value()); !isNull(v) {
			return v
		}
	}
	return a.listenable.Value()
}
