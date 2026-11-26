package animation

// Port of packages/flutter/lib/src/animation/animations.dart.

import (
	"math"
	"reflect"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// _AlwaysCompleteAnimation is kAlwaysCompleteAnimation's type.
type _AlwaysCompleteAnimation struct{ AnimationBase[float64] }

func (*_AlwaysCompleteAnimation) AddListener(func())                           {}
func (*_AlwaysCompleteAnimation) RemoveListener(func())                        {}
func (*_AlwaysCompleteAnimation) AddStatusListener(AnimationStatusListener)    {}
func (*_AlwaysCompleteAnimation) RemoveStatusListener(AnimationStatusListener) {}
func (*_AlwaysCompleteAnimation) Status() AnimationStatus                      { return AnimationStatusCompleted }
func (*_AlwaysCompleteAnimation) Value() float64                               { return 1.0 }
func (*_AlwaysCompleteAnimation) String() string                               { return "kAlwaysCompleteAnimation" }

// KAlwaysCompleteAnimation is an animation that is always complete (value
// 1.0).
var KAlwaysCompleteAnimation Animation[float64] = func() Animation[float64] {
	a := &_AlwaysCompleteAnimation{}
	a.SetSelf(a)
	return a
}()

// _AlwaysDismissedAnimation is kAlwaysDismissedAnimation's type.
type _AlwaysDismissedAnimation struct{ AnimationBase[float64] }

func (*_AlwaysDismissedAnimation) AddListener(func())                           {}
func (*_AlwaysDismissedAnimation) RemoveListener(func())                        {}
func (*_AlwaysDismissedAnimation) AddStatusListener(AnimationStatusListener)    {}
func (*_AlwaysDismissedAnimation) RemoveStatusListener(AnimationStatusListener) {}
func (*_AlwaysDismissedAnimation) Status() AnimationStatus                      { return AnimationStatusDismissed }
func (*_AlwaysDismissedAnimation) Value() float64                               { return 0.0 }
func (*_AlwaysDismissedAnimation) String() string                               { return "kAlwaysDismissedAnimation" }

// KAlwaysDismissedAnimation is an animation that is always dismissed
// (value 0.0).
var KAlwaysDismissedAnimation Animation[float64] = func() Animation[float64] {
	a := &_AlwaysDismissedAnimation{}
	a.SetSelf(a)
	return a
}()

// AlwaysStoppedAnimation is an animation that is always stopped at a given
// value (with status forward).
type AlwaysStoppedAnimation[T any] struct {
	AnimationBase[T]
	value T
}

// NewAlwaysStoppedAnimation creates an animation that is always stopped at
// value.
func NewAlwaysStoppedAnimation[T any](value T) *AlwaysStoppedAnimation[T] {
	a := &AlwaysStoppedAnimation[T]{value: value}
	a.SetSelf(a)
	return a
}

func (a *AlwaysStoppedAnimation[T]) Value() T                                   { return a.value }
func (*AlwaysStoppedAnimation[T]) AddListener(func())                           {}
func (*AlwaysStoppedAnimation[T]) RemoveListener(func())                        {}
func (*AlwaysStoppedAnimation[T]) AddStatusListener(AnimationStatusListener)    {}
func (*AlwaysStoppedAnimation[T]) RemoveStatusListener(AnimationStatusListener) {}
func (*AlwaysStoppedAnimation[T]) Status() AnimationStatus                      { return AnimationStatusForward }

func (a *AlwaysStoppedAnimation[T]) ToStringDetails() string {
	return a.AnimationBase.ToStringDetails() + " " + dartString(a.value) + "; paused"
}

// ParentSelf is what AnimationWithParentMixin calls on the object it is
// mixed into.
type ParentSelf[T any] interface {
	// Parent is the animation whose value this animation will proxy.
	Parent() Animation[T]
}

// AnimationWithParentMixin implements most of the Animation interface by
// deferring its behavior to a given Parent animation.
type AnimationWithParentMixin[T any] struct{ self ParentSelf[T] }

// SetSelf points the mixin at the object it is mixed into.
func (m *AnimationWithParentMixin[T]) SetSelf(self ParentSelf[T]) { m.self = self }

func (m *AnimationWithParentMixin[T]) parent() Animation[T] {
	if m.self == nil {
		panic(unbound("AnimationWithParentMixin"))
	}
	return m.self.Parent()
}

func (m *AnimationWithParentMixin[T]) AddListener(listener func()) {
	m.parent().AddListener(listener)
}

func (m *AnimationWithParentMixin[T]) RemoveListener(listener func()) {
	m.parent().RemoveListener(listener)
}

func (m *AnimationWithParentMixin[T]) AddStatusListener(listener AnimationStatusListener) {
	m.parent().AddStatusListener(listener)
}

func (m *AnimationWithParentMixin[T]) RemoveStatusListener(listener AnimationStatusListener) {
	m.parent().RemoveStatusListener(listener)
}

func (m *AnimationWithParentMixin[T]) Status() AnimationStatus { return m.parent().Status() }

// ProxyAnimation is an animation that is a proxy for another animation
// (its Parent), which can be changed: useful when the animation driving a
// value changes over time.
type ProxyAnimation struct {
	AnimationBase[float64]
	AnimationLazyListenerMixin
	AnimationLocalListenersMixin
	AnimationLocalStatusListenersMixin

	status *AnimationStatus
	value  *float64
	parent Animation[float64]

	notifyListeners       func()
	notifyStatusListeners AnimationStatusListener
}

// NewProxyAnimation creates a proxy animation. Without an animation (nil),
// it starts dismissed with value 0.0.
func NewProxyAnimation(animation ...Animation[float64]) *ProxyAnimation {
	p := &ProxyAnimation{}
	p.SetSelf(p)
	if len(animation) > 0 && !isNull(animation[0]) {
		p.parent = animation[0]
	}
	if p.parent == nil {
		status, value := AnimationStatusDismissed, 0.0
		p.status, p.value = &status, &value
	}
	return p
}

// SetSelf points the embedded bases at the object embedding the proxy.
func (p *ProxyAnimation) SetSelf(self interface {
	Animation[float64]
	LazyListenerSelf
}) {
	p.AnimationBase.SetSelf(self)
	p.AnimationLazyListenerMixin.SetSelf(self)
	p.AnimationLocalListenersMixin.SetSelf(registrationSelf(self, p))
	p.AnimationLocalStatusListenersMixin.SetSelf(registrationSelf(self, p))
	// Dart's tear-offs notifyListeners/notifyStatusListeners, kept so the
	// same values are removed from the parent.
	p.notifyListeners = p.NotifyListeners
	p.notifyStatusListeners = p.NotifyStatusListeners
}

// Parent is the animation whose value this animation proxies (nil: none).
func (p *ProxyAnimation) Parent() Animation[float64] { return p.parent }

// SetParent changes the proxied animation; listeners are notified if the
// value or status changes.
func (p *ProxyAnimation) SetParent(value Animation[float64]) {
	if isNull(value) {
		value = nil
	}
	if animationsIdentical(value, p.parent) {
		return
	}
	if p.parent != nil {
		status, v := p.parent.Status(), p.parent.Value()
		p.status, p.value = &status, &v
		if p.IsListening() {
			p.DidStopListening()
		}
	}
	p.parent = value
	if p.parent != nil {
		if p.IsListening() {
			p.DidStartListening()
		}
		if *p.value != p.parent.Value() {
			p.NotifyListeners()
		}
		if *p.status != p.parent.Status() {
			p.NotifyStatusListeners(p.parent.Status())
		}
		p.status, p.value = nil, nil
	}
}

// DidStartListening forwards the parent's notifications.
func (p *ProxyAnimation) DidStartListening() {
	if p.parent != nil {
		p.parent.AddListener(p.notifyListeners)
		p.parent.AddStatusListener(p.notifyStatusListeners)
	}
}

// DidStopListening stops forwarding the parent's notifications.
func (p *ProxyAnimation) DidStopListening() {
	if p.parent != nil {
		p.parent.RemoveListener(p.notifyListeners)
		p.parent.RemoveStatusListener(p.notifyStatusListeners)
	}
}

func (p *ProxyAnimation) Status() AnimationStatus {
	if p.parent != nil {
		return p.parent.Status()
	}
	return *p.status
}

func (p *ProxyAnimation) Value() float64 {
	if p.parent != nil {
		return p.parent.Value()
	}
	return *p.value
}

func (p *ProxyAnimation) String() string {
	if p.parent == nil {
		return foundation.ObjectRuntimeType(p.outer(), "ProxyAnimation") + "(null; " +
			p.AnimationBase.ToStringDetails() + " " + dart.ToStringAsFixed(p.Value(), 3) + ")"
	}
	return p.parent.String() + "➩" + foundation.ObjectRuntimeType(p.outer(), "ProxyAnimation")
}

// ReverseAnimation is an animation that is the reverse of another
// animation: value 1.0 - parent's value, and the opposite status.
type ReverseAnimation struct {
	AnimationBase[float64]
	AnimationLazyListenerMixin
	AnimationLocalStatusListenersMixin

	// Parent is the animation whose value and direction this animation is
	// reversing.
	Parent Animation[float64]

	statusChangeHandler AnimationStatusListener
}

// NewReverseAnimation creates a reverse animation of parent.
func NewReverseAnimation(parent Animation[float64]) *ReverseAnimation {
	r := &ReverseAnimation{Parent: parent}
	r.SetSelf(r)
	return r
}

// SetSelf points the embedded bases at the object embedding the animation.
func (r *ReverseAnimation) SetSelf(self interface {
	Animation[float64]
	LazyListenerSelf
}) {
	r.AnimationBase.SetSelf(self)
	r.AnimationLazyListenerMixin.SetSelf(self)
	r.AnimationLocalStatusListenersMixin.SetSelf(registrationSelf(self, r))
	r.statusChangeHandler = r.handleStatusChange
}

func (r *ReverseAnimation) AddListener(listener func()) {
	r.DidRegisterListener()
	r.Parent.AddListener(listener)
}

func (r *ReverseAnimation) RemoveListener(listener func()) {
	r.Parent.RemoveListener(listener)
	r.DidUnregisterListener()
}

// DidStartListening listens to the parent's status.
func (r *ReverseAnimation) DidStartListening() { r.Parent.AddStatusListener(r.statusChangeHandler) }

// DidStopListening stops listening to the parent's status.
func (r *ReverseAnimation) DidStopListening() { r.Parent.RemoveStatusListener(r.statusChangeHandler) }

func (r *ReverseAnimation) handleStatusChange(status AnimationStatus) {
	r.NotifyStatusListeners(reverseStatus(status))
}

func (r *ReverseAnimation) Status() AnimationStatus { return reverseStatus(r.Parent.Status()) }

func (r *ReverseAnimation) Value() float64 { return 1.0 - r.Parent.Value() }

func reverseStatus(status AnimationStatus) AnimationStatus {
	switch status {
	case AnimationStatusForward:
		return AnimationStatusReverse
	case AnimationStatusReverse:
		return AnimationStatusForward
	case AnimationStatusCompleted:
		return AnimationStatusDismissed
	}
	return AnimationStatusCompleted
}

func (r *ReverseAnimation) String() string {
	return r.Parent.String() + "➪" + foundation.ObjectRuntimeType(r.outer(), "ReverseAnimation")
}

// CurvedAnimation is an animation that applies a curve to another
// animation (with ReverseCurve, when set, while the parent runs in
// reverse).
type CurvedAnimation struct {
	AnimationBase[float64]
	AnimationWithParentMixin[float64]

	parent Animation[float64]
	// Curve is the curve to use in the forward direction.
	Curve Curve
	// ReverseCurve is the curve to use in the reverse direction (nil:
	// Curve).
	ReverseCurve Curve
	// IsDisposed reports whether Dispose was called.
	IsDisposed bool

	curveDirection       *AnimationStatus
	updateCurveDirection AnimationStatusListener
}

// CurvedAnimationOptions are CurvedAnimation's optional named parameters
// (parent and curve, required, are NewCurvedAnimation's arguments).
type CurvedAnimationOptions struct {
	ReverseCurve Curve
}

// NewCurvedAnimation creates a curved animation of parent.
func NewCurvedAnimation(parent Animation[float64], curve Curve, opts ...CurvedAnimationOptions) *CurvedAnimation {
	c := &CurvedAnimation{parent: parent, Curve: curve}
	if len(opts) > 0 {
		c.ReverseCurve = opts[0].ReverseCurve
	}
	c.SetSelf(c)
	debugAssert(func() bool { return foundation.DebugMaybeDispatchCreated("animation", "CurvedAnimation", c) })
	c.handleCurveDirection(parent.Status())
	parent.AddStatusListener(c.updateCurveDirection)
	return c
}

// SetSelf points the embedded bases at the object embedding the animation.
func (c *CurvedAnimation) SetSelf(self interface {
	Animation[float64]
	ParentSelf[float64]
}) {
	c.AnimationBase.SetSelf(self)
	c.AnimationWithParentMixin.SetSelf(self)
	c.updateCurveDirection = c.handleCurveDirection
}

// Parent is the animation to which this animation applies a curve.
func (c *CurvedAnimation) Parent() Animation[float64] { return c.parent }

func (c *CurvedAnimation) handleCurveDirection(status AnimationStatus) {
	if status.IsAnimating() {
		if c.curveDirection == nil {
			c.curveDirection = &status
		}
	} else {
		c.curveDirection = nil
	}
}

func (c *CurvedAnimation) useForwardCurve() bool {
	direction := c.parent.Status()
	if c.curveDirection != nil {
		direction = *c.curveDirection
	}
	return c.ReverseCurve == nil || direction != AnimationStatusReverse
}

// Dispose stops listening to the parent's status.
func (c *CurvedAnimation) Dispose() {
	debugAssert(func() bool { return foundation.DebugMaybeDispatchDisposed(c) })
	c.IsDisposed = true
	c.parent.RemoveStatusListener(c.updateCurveDirection)
}

func (c *CurvedAnimation) Value() float64 {
	activeCurve := c.Curve
	if !c.useForwardCurve() {
		activeCurve = c.ReverseCurve
	}
	t := c.parent.Value()
	if activeCurve == nil {
		return t
	}
	if t == 0.0 || t == 1.0 {
		if foundation.KDebugMode {
			transformedValue := activeCurve.Transform(t)
			roundedTransformedValue := float64(dartRound(transformedValue))
			if roundedTransformedValue != t {
				panic(foundation.NewFlutterError(
					"Invalid curve endpoint at " + dart.DoubleToString(t) + ".\n" +
						"Curves must map 0.0 to near zero and 1.0 to near one but " +
						runtimeType(activeCurve) + " mapped " + dart.DoubleToString(t) + " to " +
						dart.DoubleToString(transformedValue) + ", which is near " +
						dart.DoubleToString(roundedTransformedValue) + "."))
			}
		}
		return t
	}
	return activeCurve.Transform(t)
}

func (c *CurvedAnimation) String() string {
	if c.ReverseCurve == nil {
		return c.parent.String() + "➩" + c.Curve.String()
	}
	if c.useForwardCurve() {
		return c.parent.String() + "➩" + c.Curve.String() + "ₒₙ/" + c.ReverseCurve.String()
	}
	return c.parent.String() + "➩" + c.Curve.String() + "/" + c.ReverseCurve.String() + "ₒₙ"
}

type _TrainHoppingMode int

const (
	trainHoppingModeMinimize _TrainHoppingMode = iota
	trainHoppingModeMaximize
)

// TrainHoppingAnimation follows an animation (the current train) until a
// second one (the next train) crosses it, then switches to that one.
type TrainHoppingAnimation struct {
	AnimationBase[float64]
	AnimationEagerListenerMixin
	AnimationLocalListenersMixin
	AnimationLocalStatusListenersMixin

	currentTrain Animation[float64]
	nextTrain    Animation[float64]
	mode         *_TrainHoppingMode
	// OnSwitchedTrain is called when this animation switches to the next
	// train.
	OnSwitchedTrain func()

	lastStatus *AnimationStatus
	lastValue  *float64

	statusChangeHandler AnimationStatusListener
	valueChangeHandler  func()
}

// TrainHoppingAnimationOptions are TrainHoppingAnimation's named
// parameters.
type TrainHoppingAnimationOptions struct {
	OnSwitchedTrain func()
}

// NewTrainHoppingAnimation creates a train-hopping animation: it follows
// currentTrain until nextTrain (which may be nil) crosses it.
func NewTrainHoppingAnimation(currentTrain, nextTrain Animation[float64], opts ...TrainHoppingAnimationOptions) *TrainHoppingAnimation {
	a := &TrainHoppingAnimation{currentTrain: currentTrain}
	if !isNull(nextTrain) {
		a.nextTrain = nextTrain
	}
	if len(opts) > 0 {
		a.OnSwitchedTrain = opts[0].OnSwitchedTrain
	}
	a.SetSelf(a)
	debugAssert(func() bool { return foundation.DebugMaybeDispatchCreated("animation", "TrainHoppingAnimation", a) })
	if a.nextTrain != nil {
		switch {
		case a.currentTrain.Value() == a.nextTrain.Value():
			a.currentTrain = a.nextTrain
			a.nextTrain = nil
		case a.currentTrain.Value() > a.nextTrain.Value():
			mode := trainHoppingModeMaximize
			a.mode = &mode
		default:
			assert(a.currentTrain.Value() < a.nextTrain.Value(), nil)
			mode := trainHoppingModeMinimize
			a.mode = &mode
		}
	}
	a.currentTrain.AddStatusListener(a.statusChangeHandler)
	a.currentTrain.AddListener(a.valueChangeHandler)
	if a.nextTrain != nil {
		a.nextTrain.AddListener(a.valueChangeHandler)
	}
	assert(a.mode != nil || a.nextTrain == nil, nil)
	return a
}

// SetSelf points the embedded bases at the object embedding the animation.
func (a *TrainHoppingAnimation) SetSelf(self Animation[float64]) {
	a.AnimationBase.SetSelf(self)
	a.AnimationLocalListenersMixin.SetSelf(registrationSelf(self, a))
	a.AnimationLocalStatusListenersMixin.SetSelf(registrationSelf(self, a))
	a.statusChangeHandler = a.handleStatusChange
	a.valueChangeHandler = a.handleValueChange
}

// CurrentTrain is the animation that is currently driving this animation
// (nil after Dispose).
func (a *TrainHoppingAnimation) CurrentTrain() Animation[float64] { return a.currentTrain }

func (a *TrainHoppingAnimation) handleStatusChange(status AnimationStatus) {
	assert(a.currentTrain != nil, nil)
	if a.lastStatus == nil || status != *a.lastStatus {
		a.NotifyStatusListeners(status)
		a.lastStatus = &status
	}
	assert(a.lastStatus != nil, nil)
}

func (a *TrainHoppingAnimation) Status() AnimationStatus { return a.currentTrain.Status() }

func (a *TrainHoppingAnimation) handleValueChange() {
	assert(a.currentTrain != nil, nil)
	hop := false
	if a.nextTrain != nil {
		assert(a.mode != nil, nil)
		switch *a.mode {
		case trainHoppingModeMinimize:
			hop = a.nextTrain.Value() <= a.currentTrain.Value()
		case trainHoppingModeMaximize:
			hop = a.nextTrain.Value() >= a.currentTrain.Value()
		}
		if hop {
			a.currentTrain.RemoveStatusListener(a.statusChangeHandler)
			a.currentTrain.RemoveListener(a.valueChangeHandler)
			a.currentTrain = a.nextTrain
			a.nextTrain = nil
			a.currentTrain.AddStatusListener(a.statusChangeHandler)
			a.handleStatusChange(a.currentTrain.Status())
		}
	}
	newValue := a.Value()
	if a.lastValue == nil || newValue != *a.lastValue {
		a.NotifyListeners()
		a.lastValue = &newValue
	}
	assert(a.lastValue != nil, nil)
	if hop && a.OnSwitchedTrain != nil {
		a.OnSwitchedTrain()
	}
}

func (a *TrainHoppingAnimation) Value() float64 { return a.currentTrain.Value() }

// Dispose stops listening to the trains and clears the listeners.
func (a *TrainHoppingAnimation) Dispose() {
	debugAssert(func() bool { return foundation.DebugMaybeDispatchDisposed(a) })
	assert(a.currentTrain != nil, nil)
	a.currentTrain.RemoveStatusListener(a.statusChangeHandler)
	a.currentTrain.RemoveListener(a.valueChangeHandler)
	a.currentTrain = nil
	if a.nextTrain != nil {
		a.nextTrain.RemoveListener(a.valueChangeHandler)
	}
	a.nextTrain = nil
	a.ClearListeners()
	a.ClearStatusListeners()
	a.AnimationEagerListenerMixin.Dispose()
}

func (a *TrainHoppingAnimation) String() string {
	current := "null"
	if a.currentTrain != nil {
		current = a.currentTrain.String()
	}
	if a.nextTrain != nil {
		return current + "➩" + foundation.ObjectRuntimeType(a.outer(), "TrainHoppingAnimation") +
			"(next: " + a.nextTrain.String() + ")"
	}
	return current + "➩" + foundation.ObjectRuntimeType(a.outer(), "TrainHoppingAnimation") + "(no next)"
}

// CompoundAnimation is an interface for combining multiple Animations
// (Dart's abstract class): the embedding type defines Value and calls
// Construct. Its status is Next's while Next is animating, First's
// otherwise.
type CompoundAnimation[T any] struct {
	AnimationBase[T]
	AnimationLazyListenerMixin
	AnimationLocalListenersMixin
	AnimationLocalStatusListenersMixin

	// First is the first of the combined animations.
	First Animation[T]
	// Next is the second of the combined animations.
	Next Animation[T]

	lastStatus *AnimationStatus
	lastValue  *T

	maybeNotifyListeners       func()
	maybeNotifyStatusListeners AnimationStatusListener
}

// Construct is Dart's CompoundAnimation({required first, required next})
// constructor, run by the embedding type (self) once.
func (c *CompoundAnimation[T]) Construct(self Animation[T], first, next Animation[T]) {
	c.First, c.Next = first, next
	c.AnimationBase.SetSelf(self)
	if lazy, ok := self.(LazyListenerSelf); ok {
		c.AnimationLazyListenerMixin.SetSelf(lazy)
	} else {
		c.AnimationLazyListenerMixin.SetSelf(c)
	}
	c.AnimationLocalListenersMixin.SetSelf(registrationSelf(self, c))
	c.AnimationLocalStatusListenersMixin.SetSelf(registrationSelf(self, c))
	c.maybeNotifyListeners = c.handleMaybeNotifyListeners
	c.maybeNotifyStatusListeners = c.handleMaybeNotifyStatusListeners
}

// DidStartListening listens to both animations.
func (c *CompoundAnimation[T]) DidStartListening() {
	c.First.AddListener(c.maybeNotifyListeners)
	c.First.AddStatusListener(c.maybeNotifyStatusListeners)
	c.Next.AddListener(c.maybeNotifyListeners)
	c.Next.AddStatusListener(c.maybeNotifyStatusListeners)
}

// DidStopListening stops listening to both animations.
func (c *CompoundAnimation[T]) DidStopListening() {
	c.First.RemoveListener(c.maybeNotifyListeners)
	c.First.RemoveStatusListener(c.maybeNotifyStatusListeners)
	c.Next.RemoveListener(c.maybeNotifyListeners)
	c.Next.RemoveStatusListener(c.maybeNotifyStatusListeners)
}

// Status is Next's status while Next is animating, First's otherwise.
func (c *CompoundAnimation[T]) Status() AnimationStatus {
	if c.Next.Status().IsAnimating() {
		return c.Next.Status()
	}
	return c.First.Status()
}

func (c *CompoundAnimation[T]) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "CompoundAnimation") + "(" + c.First.String() + ", " +
		c.Next.String() + ")"
}

func (c *CompoundAnimation[T]) handleMaybeNotifyStatusListeners(AnimationStatus) {
	status := c.outer().Status()
	if c.lastStatus == nil || status != *c.lastStatus {
		c.lastStatus = &status
		c.NotifyStatusListeners(status)
	}
}

func (c *CompoundAnimation[T]) handleMaybeNotifyListeners() {
	value := c.outer().Value()
	if c.lastValue == nil || !valuesEqual(value, *c.lastValue) {
		c.lastValue = &value
		c.NotifyListeners()
	}
}

// AnimationMean is an animation of two animations' mean value.
type AnimationMean struct{ CompoundAnimation[float64] }

// NewAnimationMean creates an animation of the mean of left and right.
func NewAnimationMean(left, right Animation[float64]) *AnimationMean {
	a := &AnimationMean{}
	a.Construct(a, left, right)
	return a
}

func (a *AnimationMean) Value() float64 { return (a.First.Value() + a.Next.Value()) / 2.0 }

// num is Dart's num as a type bound.
type num interface{ ~int | ~float64 }

// AnimationMax is an animation of two animations' maximum value.
type AnimationMax[T num] struct{ CompoundAnimation[T] }

// NewAnimationMax creates an animation of the maximum of first and next.
func NewAnimationMax[T num](first, next Animation[T]) *AnimationMax[T] {
	a := &AnimationMax[T]{}
	a.Construct(a, first, next)
	return a
}

func (a *AnimationMax[T]) Value() T { return dartMax(a.First.Value(), a.Next.Value()) }

// AnimationMin is an animation of two animations' minimum value.
type AnimationMin[T num] struct{ CompoundAnimation[T] }

// NewAnimationMin creates an animation of the minimum of first and next.
func NewAnimationMin[T num](first, next Animation[T]) *AnimationMin[T] {
	a := &AnimationMin[T]{}
	a.Construct(a, first, next)
	return a
}

func (a *AnimationMin[T]) Value() T { return dartMin(a.First.Value(), a.Next.Value()) }

func isDouble[T num]() bool { return reflect.TypeFor[T]().Kind() == reflect.Float64 }

// dartMax is dart:math's max (math_patch.dart): NaN if either is NaN, and
// 0.0 over -0.0.
func dartMax[T num](a, b T) T {
	if a > b {
		return a
	}
	if a < b {
		return b
	}
	if isDouble[T]() {
		fa, fb := float64(a), float64(b)
		if fa == 0.0 {
			// a is either 0.0 or -0.0; b is either 0.0, -0.0 or NaN. This is
			// 0.0 if either is 0.0, and NaN if b is NaN.
			return T(fa + fb)
		}
		if math.IsNaN(fb) {
			return b
		}
		return a
	}
	return a // ints: max(-0.0, 0) can't arise
}

// dartMin is dart:math's min (math_patch.dart): NaN if either is NaN, and
// -0.0 over 0.0.
func dartMin[T num](a, b T) T {
	if a > b {
		return b
	}
	if a < b {
		return a
	}
	if isDouble[T]() {
		fa, fb := float64(a), float64(b)
		if fa == 0.0 {
			// -0.0 if either a or b is -0.0, NaN if b is NaN.
			return T(float64(float64(fa+fb)*fa) * fb)
		}
		if (fa == 0 && math.Signbit(fb)) || math.IsNaN(fb) {
			return b
		}
		return a
	}
	return a
}

// animationsIdentical is Dart's == on animations (identity, as none
// overrides it).
func animationsIdentical(a, b Animation[float64]) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Type() != vb.Type() {
		return false
	}
	if va.Kind() == reflect.Pointer {
		return va.Pointer() == vb.Pointer()
	}
	return a == b
}

// valuesEqual is Dart's == on two values: the Equal method when the type
// has one, Go's == for comparable values (NaN is unequal), false
// otherwise.
func valuesEqual[T any](a, b T) bool {
	if e, ok := any(a).(interface{ Equal(T) bool }); ok {
		return e.Equal(b)
	}
	va := reflect.ValueOf(any(a))
	if !va.IsValid() {
		return !reflect.ValueOf(any(b)).IsValid()
	}
	if va.Comparable() {
		return any(a) == any(b)
	}
	return false
}
