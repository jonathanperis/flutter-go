package animation

// Port of packages/flutter/lib/src/animation/animation_controller.dart.
//
// AnimationController is a struct (Flutter extends it). Its own algorithms
// call its members directly; String/ToStringDetails and the Animation
// getters go through AnimationBase's self, so a type embedding the
// controller calls SetSelf once.
//
// Floating point: the arithmetic is Dart's, operation for operation;
// products that Go could fuse into a multiply-add are rounded explicitly
// (see src/cmd/fmacheck).

import (
	"math"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// _AnimationDirection is the direction in which an animation is running.
type _AnimationDirection int

const (
	animationDirectionForward _AnimationDirection = iota
	animationDirectionReverse
)

var kFlingSpringDescription = physics.NewSpringDescriptionWithDampingRatio(1.0, 500.0, 1.0)

var kFlingTolerance = physics.Tolerance{Velocity: math.Inf(1), Distance: 0.01, Time: physics.DefaultTolerance.Time}

// SemanticsDisableAnimations stands for SemanticsBinding.instance
// .disableAnimations (the platform's reduce-motion setting, or
// debugSemanticsDisableAnimations), which AnimationBehavior.normal
// consults. The semantics library is not ported yet: until it is, this
// hook answers (false by default; tests replace it).
var SemanticsDisableAnimations = func() bool { return false }

// AnimationBehavior configures how an AnimationController behaves when
// animations are disabled (reduced motion).
type AnimationBehavior int

const (
	// AnimationBehaviorNormal: the controller reduces its duration when
	// animations are turned off (to 5%, or a 200x fling velocity).
	AnimationBehaviorNormal AnimationBehavior = iota
	// AnimationBehaviorPreserve: the controller keeps its behavior; for
	// repeating animations and animations that must not be skipped.
	AnimationBehaviorPreserve
)

// Name is the Dart enum value's name ("normal").
func (b AnimationBehavior) Name() string { return [...]string{"normal", "preserve"}[b] }

// String is Dart's toString ("AnimationBehavior.normal").
func (b AnimationBehavior) String() string { return "AnimationBehavior." + b.Name() }

func (b AnimationBehavior) enableAnimations() bool {
	if b == AnimationBehaviorNormal {
		return !SemanticsDisableAnimations()
	}
	return true
}

// AnimationController is a controller for an animation: it produces values
// from LowerBound to UpperBound (0.0 to 1.0 by default) over Duration,
// one per frame while running, from a Ticker of its TickerProvider.
type AnimationController struct {
	AnimationBase[float64]
	AnimationEagerListenerMixin
	AnimationLocalListenersMixin
	AnimationLocalStatusListenersMixin

	// LowerBound is the value at which this animation is dismissed.
	LowerBound float64
	// UpperBound is the value at which this animation is completed.
	UpperBound float64
	// DebugLabel identifies the controller in debug output ("" is Dart's
	// null).
	DebugLabel string
	// AnimationBehavior is the behavior when animations are disabled.
	AnimationBehavior AnimationBehavior
	// Duration is the length of time this animation should last (nil:
	// none). For reverse runs, ReverseDuration takes precedence when set.
	Duration *time.Duration
	// ReverseDuration is the length of time this animation should last
	// when going in reverse (nil: Duration).
	ReverseDuration *time.Duration

	ticker              *scheduler.Ticker
	simulation          physics.Simulation
	value               float64
	lastElapsedDuration *time.Duration
	direction           _AnimationDirection
	status              AnimationStatus
	lastReportedStatus  AnimationStatus
}

// AnimationControllerOptions are AnimationController's named parameters
// (vsync, required, is NewAnimationController's first argument).
type AnimationControllerOptions struct {
	// Value is the initial value (nil: LowerBound).
	Value             *float64
	Duration          *time.Duration
	ReverseDuration   *time.Duration
	DebugLabel        string
	LowerBound        *float64 // nil: 0.0
	UpperBound        *float64 // nil: 1.0
	AnimationBehavior AnimationBehavior
}

// NewAnimationController creates an animation controller whose ticker
// comes from vsync.
func NewAnimationController(vsync scheduler.TickerProvider, opts ...AnimationControllerOptions) *AnimationController {
	c := &AnimationController{}
	c.Construct(c, vsync, opts...)
	return c
}

// Construct is Dart's AnimationController({...}) constructor, run once by
// a type extending AnimationController (self is the outer object):
//
//	type MyController struct{ animation.AnimationController }
//
//	c := &MyController{}
//	c.Construct(c, vsync, animation.AnimationControllerOptions{...})
func (c *AnimationController) Construct(self Animation[float64], vsync scheduler.TickerProvider, opts ...AnimationControllerOptions) {
	var o AnimationControllerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	c.LowerBound, c.UpperBound = 0.0, 1.0
	c.Duration, c.ReverseDuration, c.DebugLabel, c.AnimationBehavior = o.Duration, o.ReverseDuration, o.DebugLabel, o.AnimationBehavior
	if o.LowerBound != nil {
		c.LowerBound = *o.LowerBound
	}
	if o.UpperBound != nil {
		c.UpperBound = *o.UpperBound
	}
	assert(c.UpperBound >= c.LowerBound, nil)
	c.SetSelf(self)
	debugAssert(func() bool { return foundation.DebugMaybeDispatchCreated("animation", "AnimationController", self) })
	c.ticker = vsync.CreateTicker(c.tick)
	if o.Value != nil {
		c.internalSetValue(*o.Value)
	} else {
		c.internalSetValue(c.LowerBound)
	}
}

// AnimationControllerUnboundedOptions are AnimationController.unbounded's
// named parameters (vsync is the first argument).
type AnimationControllerUnboundedOptions struct {
	Value           float64
	Duration        *time.Duration
	ReverseDuration *time.Duration
	DebugLabel      string
	// AnimationBehavior: nil is AnimationBehaviorPreserve.
	AnimationBehavior *AnimationBehavior
}

// NewAnimationControllerUnbounded is AnimationController.unbounded: a
// controller without bounds (from -∞ to +∞), for physics simulations such
// as scrolling. Its default behavior is AnimationBehaviorPreserve.
func NewAnimationControllerUnbounded(vsync scheduler.TickerProvider, opts ...AnimationControllerUnboundedOptions) *AnimationController {
	c := &AnimationController{}
	c.ConstructUnbounded(c, vsync, opts...)
	return c
}

// ConstructUnbounded is Dart's AnimationController.unbounded constructor,
// run once by a type extending AnimationController (see Construct).
func (c *AnimationController) ConstructUnbounded(self Animation[float64], vsync scheduler.TickerProvider, opts ...AnimationControllerUnboundedOptions) {
	var o AnimationControllerUnboundedOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	c.LowerBound, c.UpperBound = math.Inf(-1), math.Inf(1)
	c.Duration, c.ReverseDuration, c.DebugLabel = o.Duration, o.ReverseDuration, o.DebugLabel
	c.AnimationBehavior = AnimationBehaviorPreserve
	if o.AnimationBehavior != nil {
		c.AnimationBehavior = *o.AnimationBehavior
	}
	c.SetSelf(self)
	debugAssert(func() bool { return foundation.DebugMaybeDispatchCreated("animation", "AnimationController", self) })
	c.ticker = vsync.CreateTicker(c.tick)
	c.internalSetValue(o.Value)
}

// SetSelf points the controller's embedded bases at the object embedding
// it (Construct calls it).
func (c *AnimationController) SetSelf(self Animation[float64]) {
	c.AnimationBase.SetSelf(self)
	c.AnimationLocalListenersMixin.SetSelf(registrationSelf(self, c))
	c.AnimationLocalStatusListenersMixin.SetSelf(registrationSelf(self, c))
}

// View is this controller as a read-only Animation.
func (c *AnimationController) View() Animation[float64] { return c.outer() }

// Resync recreates the Ticker with the new TickerProvider.
func (c *AnimationController) Resync(vsync scheduler.TickerProvider) {
	oldTicker := c.ticker
	c.ticker = vsync.CreateTicker(c.tick)
	c.ticker.AbsorbTicker(oldTicker)
}

// Value is the current value of the animation.
func (c *AnimationController) Value() float64 { return c.value }

// SetValue stops the animation and sets the value (clamped to the bounds),
// notifying listeners and, if it changed, status listeners.
func (c *AnimationController) SetValue(newValue float64) {
	c.Stop()
	c.internalSetValue(newValue)
	c.NotifyListeners()
	c.checkStatusChanged()
}

// Reset sets the value to LowerBound, stopping the animation.
func (c *AnimationController) Reset() { c.SetValue(c.LowerBound) }

// Velocity is the rate of change of the value in units per second (0.0
// when not animating).
func (c *AnimationController) Velocity() float64 {
	if !c.IsAnimating() {
		return 0.0
	}
	return c.simulation.Dx(float64(c.lastElapsedDuration.Microseconds()) / 1e6)
}

func (c *AnimationController) internalSetValue(newValue float64) {
	c.value = ui.ClampDouble(newValue, c.LowerBound, c.UpperBound)
	switch {
	case c.value == c.LowerBound:
		c.status = AnimationStatusDismissed
	case c.value == c.UpperBound:
		c.status = AnimationStatusCompleted
	default:
		if c.direction == animationDirectionForward {
			c.status = AnimationStatusForward
		} else {
			c.status = AnimationStatusReverse
		}
	}
}

// LastElapsedDuration is the amount of time that has passed between the
// time the animation started and the most recent tick (nil when not
// animating).
func (c *AnimationController) LastElapsedDuration() *time.Duration { return c.lastElapsedDuration }

// IsAnimating reports whether the ticker is running (the animation may be
// silenced).
func (c *AnimationController) IsAnimating() bool { return c.ticker != nil && c.ticker.IsActive() }

// Status is the current status.
func (c *AnimationController) Status() AnimationStatus { return c.status }

// ForwardOptions are Forward's named parameters.
type ForwardOptions struct {
	// From, when set, is the value to start from.
	From *float64
}

func (c *AnimationController) assertNotDisposed(method string) {
	assert(c.ticker != nil, "AnimationController."+method+"() called after AnimationController.dispose()\n"+
		"AnimationController methods should not be used after calling dispose.")
}

// Forward starts running this animation forwards (towards the end).
func (c *AnimationController) Forward(opts ...ForwardOptions) *scheduler.TickerFuture {
	if foundation.KDebugMode && c.Duration == nil {
		panic(foundation.NewFlutterError(
			"AnimationController.forward() called with no default duration.\n" +
				"The \"duration\" property should be set, either in the constructor or later, before " +
				"calling the forward() function."))
	}
	c.assertNotDisposed("forward")
	c.direction = animationDirectionForward
	if len(opts) > 0 && opts[0].From != nil {
		c.SetValue(*opts[0].From)
	}
	return c.animateToInternal(c.UpperBound, nil, Curves.Linear)
}

// ReverseOptions are Reverse's named parameters.
type ReverseOptions struct {
	// From, when set, is the value to start from.
	From *float64
}

// Reverse starts running this animation in reverse (towards the
// beginning).
func (c *AnimationController) Reverse(opts ...ReverseOptions) *scheduler.TickerFuture {
	if foundation.KDebugMode && c.Duration == nil && c.ReverseDuration == nil {
		panic(foundation.NewFlutterError(
			"AnimationController.reverse() called with no default duration or reverseDuration.\n" +
				"The \"duration\" or \"reverseDuration\" property should be set, either in the constructor or later, before " +
				"calling the reverse() function."))
	}
	c.assertNotDisposed("reverse")
	c.direction = animationDirectionReverse
	if len(opts) > 0 && opts[0].From != nil {
		c.SetValue(*opts[0].From)
	}
	return c.animateToInternal(c.LowerBound, nil, Curves.Linear)
}

// ToggleOptions are Toggle's named parameters.
type ToggleOptions struct {
	// From, when set, is the value to start from.
	From *float64
}

// Toggle runs the animation forward if it is dismissed or reversing, and
// in reverse otherwise.
func (c *AnimationController) Toggle(opts ...ToggleOptions) *scheduler.TickerFuture {
	if foundation.KDebugMode {
		duration := c.Duration
		if c.IsForwardOrCompleted() && duration == nil {
			duration = c.ReverseDuration
		}
		if duration == nil {
			panic(foundation.NewFlutterError(
				"AnimationController.toggle() called with no default duration.\n" +
					"The \"duration\" property should be set, either in the constructor or later, before " +
					"calling the toggle() function."))
		}
	}
	c.assertNotDisposed("toggle")
	if c.IsForwardOrCompleted() {
		c.direction = animationDirectionReverse
	} else {
		c.direction = animationDirectionForward
	}
	if len(opts) > 0 && opts[0].From != nil {
		c.SetValue(*opts[0].From)
	}
	target := c.UpperBound
	if c.direction == animationDirectionReverse {
		target = c.LowerBound
	}
	return c.animateToInternal(target, nil, Curves.Linear)
}

// AnimateToOptions are AnimateTo's named parameters.
type AnimateToOptions struct {
	// Duration overrides the controller's (scaled by the distance to go).
	Duration *time.Duration
	// Curve is the curve to follow (nil: Curves.Linear).
	Curve Curve
}

// AnimateTo drives the animation from its current value to target. The
// status is forward (then completed) whatever the direction of travel.
func (c *AnimationController) AnimateTo(target float64, opts ...AnimateToOptions) *scheduler.TickerFuture {
	var o AnimateToOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if foundation.KDebugMode && c.Duration == nil && o.Duration == nil {
		panic(foundation.NewFlutterError(
			"AnimationController.animateTo() called with no explicit duration and no default duration.\n" +
				"Either the \"duration\" argument to the animateTo() method should be provided, or the " +
				"\"duration\" property should be set, either in the constructor or later, before " +
				"calling the animateTo() function."))
	}
	c.assertNotDisposed("animateTo")
	c.direction = animationDirectionForward
	return c.animateToInternal(target, o.Duration, curveOrLinear(o.Curve))
}

// AnimateBackOptions are AnimateBack's named parameters.
type AnimateBackOptions struct {
	// Duration overrides the controller's (scaled by the distance to go).
	Duration *time.Duration
	// Curve is the curve to follow (nil: Curves.Linear).
	Curve Curve
}

// AnimateBack drives the animation from its current value to target, in
// reverse: the status is reverse (then dismissed).
func (c *AnimationController) AnimateBack(target float64, opts ...AnimateBackOptions) *scheduler.TickerFuture {
	var o AnimateBackOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if foundation.KDebugMode && c.Duration == nil && c.ReverseDuration == nil && o.Duration == nil {
		panic(foundation.NewFlutterError(
			"AnimationController.animateBack() called with no explicit duration and no default duration or reverseDuration.\n" +
				"Either the \"duration\" argument to the animateBack() method should be provided, or the " +
				"\"duration\" or \"reverseDuration\" property should be set, either in the constructor or later, before " +
				"calling the animateBack() function."))
	}
	c.assertNotDisposed("animateBack")
	c.direction = animationDirectionReverse
	return c.animateToInternal(target, o.Duration, curveOrLinear(o.Curve))
}

func curveOrLinear(c Curve) Curve {
	if c == nil {
		return Curves.Linear
	}
	return c
}

// scaleDuration is Dart's `Duration * num`: microseconds times factor,
// rounded (half away from zero).
func scaleDuration(d time.Duration, factor float64) time.Duration {
	return time.Duration(dartRound(float64(d.Microseconds())*factor)) * time.Microsecond
}

func (c *AnimationController) animateToInternal(target float64, duration *time.Duration, curve Curve) *scheduler.TickerFuture {
	// Ideally, the framework would be able to handle zero duration
	// animations; however, the common pattern of an eternally repeating
	// animation might cause an endless loop if it weren't delayed for at
	// least one frame. Instead, it's run at 5% of the normal duration to
	// limit most animations to a single frame.
	scale := 0.05
	if c.AnimationBehavior.enableAnimations() {
		scale = 1.0
	}
	var simulationDuration time.Duration
	if duration == nil {
		assert(!(c.Duration == nil && c.direction == animationDirectionForward), nil)
		assert(!(c.Duration == nil && c.direction == animationDirectionReverse && c.ReverseDuration == nil), nil)
		rng := c.UpperBound - c.LowerBound
		remainingFraction := 1.0
		if !math.IsInf(rng, 0) && !math.IsNaN(rng) {
			remainingFraction = math.Abs(target-c.value) / rng
		}
		var directionDuration time.Duration
		if c.direction == animationDirectionReverse && c.ReverseDuration != nil {
			directionDuration = *c.ReverseDuration
		} else {
			directionDuration = *c.Duration
		}
		simulationDuration = scaleDuration(directionDuration, remainingFraction)
	} else if target == c.Value() {
		// Already at target, don't animate.
		simulationDuration = 0
	} else {
		simulationDuration = *duration
	}
	c.Stop()
	if simulationDuration == 0 {
		if c.Value() != target {
			c.value = ui.ClampDouble(target, c.LowerBound, c.UpperBound)
			c.NotifyListeners()
		}
		if c.direction == animationDirectionForward {
			c.status = AnimationStatusCompleted
		} else {
			c.status = AnimationStatusDismissed
		}
		c.checkStatusChanged()
		return scheduler.NewTickerFutureComplete()
	}
	assert(simulationDuration > 0, nil)
	assert(!c.IsAnimating(), nil)
	return c.startSimulation(newInterpolationSimulation(c.value, target, simulationDuration, curve, scale))
}

// RepeatOptions are Repeat's named parameters.
type RepeatOptions struct {
	// Min and Max bound the repetition (nil: LowerBound and UpperBound).
	Min, Max *float64
	// Reverse makes every other run go from Max to Min.
	Reverse bool
	// Period is the length of one run (nil: Duration).
	Period *time.Duration
	// Count, when set, is the number of runs (> 0) before stopping.
	Count *int
}

// Repeat starts running this animation in the forward direction, and
// restarts it when it completes (or alternates direction with Reverse).
// The returned future never completes unless Count is set.
func (c *AnimationController) Repeat(opts ...RepeatOptions) *scheduler.TickerFuture {
	var o RepeatOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	min, max, period := c.LowerBound, c.UpperBound, c.Duration
	if o.Min != nil {
		min = *o.Min
	}
	if o.Max != nil {
		max = *o.Max
	}
	if o.Period != nil {
		period = o.Period
	}
	if foundation.KDebugMode && period == nil {
		panic(foundation.NewFlutterError(
			"AnimationController.repeat() called without an explicit period and with no default Duration.\n" +
				"Either the \"period\" argument to the repeat() method should be provided, or the " +
				"\"duration\" property should be set, either in the constructor or later, before " +
				"calling the repeat() function."))
	}
	assert(max >= min, nil)
	assert(max <= c.UpperBound && min >= c.LowerBound, nil)
	assert(o.Count == nil || *o.Count > 0, "Count shall be greater than zero if not null")
	c.Stop()
	return c.startSimulation(newRepeatingSimulation(c.value, min, max, o.Reverse, *period, c.directionSetter, o.Count))
}

func (c *AnimationController) directionSetter(direction _AnimationDirection) {
	c.direction = direction
	if c.direction == animationDirectionForward {
		c.status = AnimationStatusForward
	} else {
		c.status = AnimationStatusReverse
	}
	c.checkStatusChanged()
}

// FlingOptions are Fling's named parameters.
type FlingOptions struct {
	// Velocity is the initial velocity (nil: 1.0); negative flings toward
	// LowerBound.
	Velocity *float64
	// SpringDescription is the spring (nil: a critically damped spring of
	// mass 1 and stiffness 500).
	SpringDescription *physics.SpringDescription
	// AnimationBehavior overrides the controller's.
	AnimationBehavior *AnimationBehavior
}

// Fling drives the animation with a spring (within LowerBound and
// UpperBound) and initial velocity.
func (c *AnimationController) Fling(opts ...FlingOptions) *scheduler.TickerFuture {
	var o FlingOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	velocity := 1.0
	if o.Velocity != nil {
		velocity = *o.Velocity
	}
	springDescription := kFlingSpringDescription
	if o.SpringDescription != nil {
		springDescription = *o.SpringDescription
	}
	var target float64
	if velocity < 0.0 {
		c.direction = animationDirectionReverse
		target = c.LowerBound - kFlingTolerance.Distance
	} else {
		c.direction = animationDirectionForward
		target = c.UpperBound + kFlingTolerance.Distance
	}
	behavior := c.AnimationBehavior
	if o.AnimationBehavior != nil {
		behavior = *o.AnimationBehavior
	}
	// The "200.0" value is arbitrary (it was chosen because it worked for
	// the drawer widget).
	scale := 200.0
	if behavior.enableAnimations() {
		scale = 1.0
	}
	simulation := physics.NewSpringSimulation(springDescription, c.Value(), target, velocity*scale)
	simulation.SetTolerance(kFlingTolerance)
	assert(simulation.Type() != physics.SpringTypeUnderDamped,
		"The specified spring simulation is of type SpringType.underDamped.\n"+
			"An underdamped spring results in oscillation rather than a fling. "+
			"Consider specifying a different springDescription, or use animateWith() "+
			"with an explicit SpringSimulation if an underdamped spring is intentional.")
	c.Stop()
	return c.startSimulation(simulation)
}

// AnimateWith drives the animation according to simulation (forward).
func (c *AnimationController) AnimateWith(simulation physics.Simulation) *scheduler.TickerFuture {
	c.assertNotDisposed("animateWith")
	c.Stop()
	c.direction = animationDirectionForward
	return c.startSimulation(simulation)
}

// AnimateBackWith drives the animation according to simulation, in
// reverse.
func (c *AnimationController) AnimateBackWith(simulation physics.Simulation) *scheduler.TickerFuture {
	c.assertNotDisposed("animateWith") // sic: Flutter's message names animateWith
	c.Stop()
	c.direction = animationDirectionReverse
	return c.startSimulation(simulation)
}

func (c *AnimationController) startSimulation(simulation physics.Simulation) *scheduler.TickerFuture {
	assert(!c.IsAnimating(), nil)
	c.simulation = simulation
	zero := time.Duration(0)
	c.lastElapsedDuration = &zero
	c.value = ui.ClampDouble(simulation.X(0.0), c.LowerBound, c.UpperBound)
	result := c.ticker.Start()
	if c.direction == animationDirectionForward {
		c.status = AnimationStatusForward
	} else {
		c.status = AnimationStatusReverse
	}
	c.checkStatusChanged()
	return result
}

// StopOptions are Stop's named parameters.
type StopOptions struct {
	// Canceled (nil: true) cancels the TickerFuture; false completes it.
	Canceled *bool
}

// Stop stops running this animation, without notifying listeners or
// changing the status.
func (c *AnimationController) Stop(opts ...StopOptions) {
	c.assertNotDisposed("stop")
	canceled := true
	if len(opts) > 0 && opts[0].Canceled != nil {
		canceled = *opts[0].Canceled
	}
	c.simulation = nil
	c.lastElapsedDuration = nil
	c.ticker.Stop(scheduler.TickerStopOptions{Canceled: canceled})
}

// Dispose releases the resources used by this object; the controller must
// not be used afterwards.
func (c *AnimationController) Dispose() {
	if foundation.KDebugMode && c.ticker == nil {
		style := foundation.DiagnosticsTreeStyleErrorProperty
		rt := runtimeType(c.outer())
		panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
			foundation.NewErrorSummary("AnimationController.dispose() called more than once."),
			foundation.NewErrorDescription("A given " + rt + " cannot be disposed more than once.\n"),
			foundation.NewDiagnosticsProperty[any]("The following "+rt+" object was disposed multiple times", c.outer(),
				foundation.DiagnosticsPropertyOptions{Style: &style}),
		}))
	}
	debugAssert(func() bool { return foundation.DebugMaybeDispatchDisposed(c.outer()) })
	c.ticker.Dispose()
	c.ticker = nil
	c.ClearStatusListeners()
	c.ClearListeners()
	c.AnimationEagerListenerMixin.Dispose()
}

func (c *AnimationController) checkStatusChanged() {
	newStatus := c.Status()
	if c.lastReportedStatus != newStatus {
		c.lastReportedStatus = newStatus
		c.NotifyStatusListeners(newStatus)
	}
}

func (c *AnimationController) tick(elapsed time.Duration) {
	c.lastElapsedDuration = &elapsed
	elapsedInSeconds := float64(elapsed.Microseconds()) / 1e6
	assert(elapsedInSeconds >= 0.0, nil)
	c.value = ui.ClampDouble(c.simulation.X(elapsedInSeconds), c.LowerBound, c.UpperBound)
	if c.simulation.IsDone(elapsedInSeconds) {
		if c.direction == animationDirectionForward {
			c.status = AnimationStatusCompleted
		} else {
			c.status = AnimationStatusDismissed
		}
		canceled := false
		c.Stop(StopOptions{Canceled: &canceled})
	}
	c.NotifyListeners()
	c.checkStatusChanged()
}

// ToStringDetails is the status arrow, the value to 3 decimals, and
// "paused", "DISPOSED", "silenced" and the debug label as they apply.
func (c *AnimationController) ToStringDetails() string {
	paused := ""
	if !c.IsAnimating() {
		paused = "; paused"
	}
	ticker := ""
	if c.ticker == nil {
		ticker = "; DISPOSED"
	} else if c.ticker.Muted() {
		ticker = "; silenced"
	}
	label := ""
	if foundation.KDebugMode && c.DebugLabel != "" {
		label = "; for " + c.DebugLabel
	}
	more := c.AnimationBase.ToStringDetails() + " " + dart.ToStringAsFixed(c.Value(), 3)
	return more + paused + ticker + label
}

// _InterpolationSimulation follows curve from begin to end over duration
// (scaled by scale).
type _InterpolationSimulation struct {
	physics.SimulationBase
	durationInSeconds float64
	begin, end        float64
	curve             Curve
}

func newInterpolationSimulation(begin, end float64, duration time.Duration, curve Curve, scale float64) *_InterpolationSimulation {
	assert(duration.Microseconds() > 0, nil)
	return &_InterpolationSimulation{SimulationBase: physics.NewSimulationBase(), begin: begin, end: end, curve: curve,
		durationInSeconds: float64(float64(duration.Microseconds())*scale) / 1e6}
}

func (s *_InterpolationSimulation) X(timeInSeconds float64) float64 {
	t := ui.ClampDouble(timeInSeconds/s.durationInSeconds, 0.0, 1.0)
	switch t {
	case 0.0:
		return s.begin
	case 1.0:
		return s.end
	}
	return s.begin + float64((s.end-s.begin)*s.curve.Transform(t))
}

func (s *_InterpolationSimulation) Dx(timeInSeconds float64) float64 {
	epsilon := s.Tolerance().Time
	return (s.X(timeInSeconds+epsilon) - s.X(timeInSeconds-epsilon)) / float64(2*epsilon)
}

func (s *_InterpolationSimulation) IsDone(timeInSeconds float64) bool {
	return timeInSeconds > s.durationInSeconds
}

// String is Simulation's toString: the runtime type.
func (s *_InterpolationSimulation) String() string {
	return foundation.ObjectRuntimeType(s, "Simulation")
}

// _RepeatingSimulation repeats min → max (alternating with reverse) with
// the given period, count times (forever when count is nil).
type _RepeatingSimulation struct {
	physics.SimulationBase
	min, max        float64
	reverse         bool
	count           *int
	directionSetter func(_AnimationDirection)
	periodInSeconds float64
	initialT        float64
}

func newRepeatingSimulation(initialValue, min, max float64, reverse bool, period time.Duration,
	directionSetter func(_AnimationDirection), count *int) *_RepeatingSimulation {
	assert(count == nil || *count > 0, "Count shall be greater than zero if not null")
	s := &_RepeatingSimulation{SimulationBase: physics.NewSimulationBase(), min: min, max: max, reverse: reverse,
		count: count, directionSetter: directionSetter,
		periodInSeconds: float64(period.Microseconds()) / 1e6}
	if max != min {
		s.initialT = float64(((ui.ClampDouble(initialValue, min, max) - min) / (max - min)) *
			(float64(period.Microseconds()) / 1e6))
	}
	assert(s.periodInSeconds > 0.0, nil)
	assert(s.initialT >= 0.0, nil)
	return s
}

func (s *_RepeatingSimulation) exitTimeInSeconds() float64 {
	return float64(float64(*s.count)*s.periodInSeconds) - s.initialT
}

func (s *_RepeatingSimulation) X(timeInSeconds float64) float64 {
	assert(timeInSeconds >= 0.0, nil)
	totalTimeInSeconds := timeInSeconds + s.initialT
	t := dart.Mod(totalTimeInSeconds/s.periodInSeconds, 1.0)
	isPlayingReverse := int64(dart.TruncDiv(totalTimeInSeconds, s.periodInSeconds))&1 == 1
	if s.reverse && isPlayingReverse {
		s.directionSetter(animationDirectionReverse)
		return lerpDouble(s.max, s.min, t)
	}
	s.directionSetter(animationDirectionForward)
	return lerpDouble(s.min, s.max, t)
}

func (s *_RepeatingSimulation) Dx(float64) float64 { return (s.max - s.min) / s.periodInSeconds }

func (s *_RepeatingSimulation) IsDone(timeInSeconds float64) bool {
	// if timeInSeconds elapsed the exit time && count is not null,
	// consider marking the simulation as "DONE"
	return s.count != nil && timeInSeconds >= s.exitTimeInSeconds()
}

// String is Simulation's toString: the runtime type.
func (s *_RepeatingSimulation) String() string { return foundation.ObjectRuntimeType(s, "Simulation") }

// debugAssert is assert(() {...}()) (here, the memory-allocation
// dispatch): it runs in debug builds only.
func debugAssert(check func() bool) {
	if foundation.KDebugMode {
		assert(check(), nil)
	}
}

// runtimeType is Dart's `runtimeType.toString()` for an object (its Go
// type in Dart's spelling).
func runtimeType(v any) string { return foundation.ObjectRuntimeType(v, "Object") }
