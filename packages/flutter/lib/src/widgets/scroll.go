package widgets

import (
	"github.com/jonathanperis/flutter-go/src/dart"
	"math"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/animation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// ScrollPhysics selects how a list responds at its edges and after a drag.
type ScrollPhysics int

const (
	// PlatformPhysics picks by platform, as Flutter's ScrollBehavior does:
	// clamping on Android/Linux/Windows, bouncing on iOS, bouncing with
	// "fast" deceleration on macOS.
	PlatformPhysics ScrollPhysics = iota
	ClampingPhysics
	BouncingPhysics
	BouncingFastPhysics
)

// Constants from Flutter's scroll_physics.dart / scroll_activity.dart.
var (
	defaultSpring         = physics.NewSpringDescriptionWithDampingRatio(0.5, 100.0, 1.1)
	fastSpring            = physics.NewSpringDescriptionWithDampingRatio(0.3, 75.0, 1.3) // BouncingScrollPhysics.spring, fast deceleration
	rubberBandLambda      = dart.Log(2) / 0.07
	rubberBandSpring      = physics.SpringDescription{Mass: 1.0, Stiffness: 1e5 * rubberBandLambda * rubberBandLambda, Damping: (1e5 + 1) * rubberBandLambda}
	motionStartThreshold  = 3.5   // BouncingScrollPhysics.dragStartDistanceMotionThreshold
	motionStoppedDuration = 50 * time.Millisecond // ScrollDragController.motionStoppedDurationThreshold
	bigThresholdBreak     = 24.0  // ScrollDragController._bigThresholdBreakDistance
)

// ScrollController owns a list's scroll position and its physics (Flutter's
// ScrollController + ScrollPositionWithSingleContext).
type ScrollController struct {
	list    *r.RenderFixedExtentList
	initial float64
	vsync   scheduler.TickerProvider // set when attached
	anim    *animation.AnimationController
	// DevicePixelRatio sets physics tolerances (Flutter's toleranceFor).
	// flutter_test uses 3; 0 means 1.
	DevicePixelRatio float64
	// Physics overrides the platform's default.
	Physics  ScrollPhysics
	platform services.Platform

	// drag state (ScrollDragController)
	offsetSinceLastStop *float64
	lastNonStationary   time.Duration
}

// NewScrollController starts at initialOffset.
func NewScrollController(initialOffset float64) *ScrollController {
	return &ScrollController{initial: initialOffset}
}

func (c *ScrollController) attach(l *r.RenderFixedExtentList) {
	if c.list == nil {
		l.SetScrollOffset(c.initial)
	} else {
		l.SetScrollOffset(c.list.ScrollOffset())
	}
	c.list = l
}

func (c *ScrollController) Offset() float64 {
	if c.list == nil {
		return c.initial
	}
	return c.list.ScrollOffset()
}

func (c *ScrollController) MaxScrollExtent() float64 {
	if c.list == nil {
		return 0
	}
	return c.list.MaxScrollExtent()
}

func (c *ScrollController) physics() ScrollPhysics {
	if c.Physics != PlatformPhysics {
		return c.Physics
	}
	switch c.platform {
	case services.MacOS:
		return BouncingFastPhysics
	case services.IOS:
		return BouncingPhysics
	}
	return ClampingPhysics
}

func (c *ScrollController) bouncing() bool { return c.physics() != ClampingPhysics }

func (c *ScrollController) dpr() float64 {
	if c.DevicePixelRatio > 0 {
		return c.DevicePixelRatio
	}
	return 1
}

// tolerance is ScrollPhysics.toleranceFor: Flutter's time tolerance stays
// at the default.
func (c *ScrollController) tolerance() physics.Tolerance {
	return physics.NewTolerance(physics.ToleranceOptions{Velocity: F(1.0 / (0.050 * c.dpr())), Distance: F(1.0 / c.dpr())})
}

func (c *ScrollController) outOfRange() bool {
	p := c.list.ScrollOffset()
	return p < 0 || p > c.list.MaxScrollExtent()
}

// applyBoundaryConditions is ClampingScrollPhysics's (bouncing allows all).
func (c *ScrollController) applyBoundaryConditions(value float64) float64 {
	if c.bouncing() {
		return 0
	}
	pixels, lo, hi := c.list.ScrollOffset(), 0.0, c.list.MaxScrollExtent()
	switch {
	case value < pixels && pixels <= lo:
		return value - pixels
	case hi <= pixels && pixels < value:
		return value - pixels
	case value < lo && lo < pixels:
		return value - lo
	case pixels < hi && hi < value:
		return value - hi
	}
	return 0
}

// setPixels moves to value minus whatever the physics refuse; it returns
// that overscroll (Flutter's ScrollPosition.setPixels).
func (c *ScrollController) setPixels(value float64) float64 {
	over := c.applyBoundaryConditions(value)
	c.list.SetScrollOffset(value - over)
	return over
}

// applyPhysicsToUserOffset is BouncingScrollPhysics's rubber-band friction
// while overscrolled (identity for clamping).
func (c *ScrollController) applyPhysicsToUserOffset(offset float64) float64 {
	if !c.bouncing() || !c.outOfRange() {
		return offset
	}
	pixels, max := c.list.ScrollOffset(), c.list.MaxScrollExtent()
	pastStart, pastEnd := math.Max(-pixels, 0), math.Max(pixels-max, 0)
	past := math.Max(pastStart, pastEnd)
	easing := (pastStart > 0 && offset < 0) || (pastEnd > 0 && offset > 0)
	factor := 0.52
	if c.physics() == BouncingFastPhysics {
		factor = 0.26
	}
	friction := func(frac float64) float64 { return dart.Pow(1-frac, 2) * factor }
	viewport := c.list.ViewportExtent()
	var gamma float64
	if easing {
		gamma = friction((past - math.Abs(offset)) / viewport)
	} else {
		gamma = friction(past / viewport)
	}
	dir := sign(offset)
	if easing && c.physics() == BouncingFastPhysics {
		return dir * math.Abs(offset)
	}
	abs := math.Abs(offset)
	total := 0.0
	if past > 0 {
		toLimit := past / gamma
		if abs < toLimit {
			return dir * abs * gamma
		}
		total += past
		abs -= toLimit
	}
	return dir * (total + abs)
}

func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// ---------------------------------------------------------------------------
// Activities

func (c *ScrollController) goIdle() {
	if c.anim != nil {
		c.anim.Dispose()
		c.anim = nil
	}
}

// Hold stops any scroll animation (a pointer went down: ScrollPosition.hold).
func (c *ScrollController) Hold() { c.goIdle() }

// IsAnimating reports whether a scroll animation is running.
func (c *ScrollController) IsAnimating() bool { return c.anim != nil && c.anim.IsAnimating() }

// createBallisticSimulation is the physics' (nil means: go idle).
func (c *ScrollController) createBallisticSimulation(v float64) physics.Simulation {
	tol := c.tolerance()
	pixels, max := c.list.ScrollOffset(), c.list.MaxScrollExtent()
	if !c.bouncing() {
		if c.outOfRange() {
			end := 0.0
			if pixels > max {
				end = max
			}
			return physics.NewScrollSpringSimulation(defaultSpring, pixels, end, math.Min(0, v), physics.ScrollSpringSimulationOptions{Tolerance: &tol})
		}
		if math.Abs(v) < tol.Velocity || (v > 0 && pixels >= max) || (v < 0 && pixels <= 0) {
			return nil
		}
		return NewClampingScrollSimulation(ClampingScrollSimulationOptions{Position: pixels, Velocity: v, Tolerance: &tol})
	}
	stationary := math.Abs(v) < tol.Velocity
	if stationary && !c.outOfRange() {
		return nil
	}
	spring := defaultSpring
	if c.physics() == BouncingFastPhysics {
		spring = fastSpring
	}
	if stationary { // and out of range: a rubber band back into range
		spring = rubberBandSpring
	}
	decel := 0.0
	if c.physics() == BouncingFastPhysics {
		decel = 1400
	}
	return NewBouncingScrollSimulation(BouncingScrollSimulationOptions{Spring: spring, Position: pixels, Velocity: v,
		LeadingExtent: 0, TrailingExtent: max, ConstantDeceleration: decel, Tolerance: &tol})
}

// goBallistic starts a ballistic activity at velocity v (px/s, positive
// scrolls content up), or goes idle if the physics say nothing to do.
func (c *ScrollController) goBallistic(v float64) {
	if c.list == nil || c.vsync == nil {
		return
	}
	c.goIdle()
	sim := c.createBallisticSimulation(v)
	if sim == nil {
		return
	}
	a := animation.NewAnimationControllerUnbounded(c.vsync, animation.AnimationControllerUnboundedOptions{Value: c.list.ScrollOffset()})
	c.anim = a
	a.AddListener(func() {
		// BallisticScrollActivity.applyMoveTo: stop if the physics refuse.
		if math.Abs(c.setPixels(a.Value())) >= 1e-10 {
			c.goIdle()
		}
	})
	a.AddStatusListener(func(s animation.AnimationStatus) {
		if s == animation.AnimationStatusCompleted && c.anim == a {
			c.goBallistic(0) // _end: settle (e.g. spring back into range)
		}
	})
	a.AnimateWith(sim)
}

// Fling starts a ballistic scroll at velocity (logical px/s, positive
// scrolls content up), as ScrollPosition.goBallistic does.
func (c *ScrollController) Fling(velocity float64) { c.goBallistic(velocity) }

// JumpTo moves to offset immediately, then lets the physics settle it, as
// Flutter's jumpTo does (an out-of-range jump animates back into range).
func (c *ScrollController) JumpTo(offset float64) {
	if c.list == nil {
		c.initial = offset
		return
	}
	c.goIdle()
	c.list.SetScrollOffset(offset)
	c.goBallistic(0)
}

// AnimateTo scrolls to offset over d with curve (a DrivenScrollActivity):
// it stops if the physics refuse a position, then settles.
func (c *ScrollController) AnimateTo(offset float64, d time.Duration, curve animation.Curve) {
	if c.list == nil || c.vsync == nil {
		c.JumpTo(offset)
		return
	}
	c.goIdle()
	a := animation.NewAnimationControllerUnbounded(c.vsync, animation.AnimationControllerUnboundedOptions{Value: c.list.ScrollOffset()})
	c.anim = a
	a.AddListener(func() {
		if c.setPixels(a.Value()) != 0 {
			c.goIdle()
		}
	})
	a.AddStatusListener(func(s animation.AnimationStatus) {
		if s == animation.AnimationStatusCompleted && c.anim == a {
			c.goBallistic(0)
		}
	})
	a.AnimateTo(offset, animation.AnimateToOptions{Duration: &d, Curve: curve})
}

// ---------------------------------------------------------------------------
// Drag (ScrollDragController) and pointer scroll

func (c *ScrollController) dragStart(ts time.Duration) {
	c.goIdle()
	c.lastNonStationary = ts
	c.offsetSinceLastStop = nil
	if c.bouncing() {
		zero := 0.0
		c.offsetSinceLastStop = &zero
	}
}

// adjustForScrollStartThreshold is ScrollDragController's: on bouncing
// physics, the first 3.5 px after a stop are swallowed, then motion eases in.
func (c *ScrollController) adjustForScrollStartThreshold(offset float64, ts time.Duration) float64 {
	if offset == 0 {
		if c.bouncing() && c.offsetSinceLastStop == nil && ts-c.lastNonStationary > motionStoppedDuration {
			zero := 0.0
			c.offsetSinceLastStop = &zero
		}
		return 0
	}
	if c.offsetSinceLastStop == nil {
		return offset
	}
	*c.offsetSinceLastStop += offset
	if math.Abs(*c.offsetSinceLastStop) > motionStartThreshold {
		c.offsetSinceLastStop = nil
		if math.Abs(offset) > bigThresholdBreak {
			return offset
		}
		return math.Min(motionStartThreshold/3.0, math.Abs(offset)) * sign(offset)
	}
	return 0
}

// dragUpdate applies a finger delta (positive = finger moved down).
func (c *ScrollController) dragUpdate(delta float64, ts time.Duration) {
	if delta != 0 {
		c.lastNonStationary = ts
	}
	delta = c.adjustForScrollStartThreshold(delta, ts)
	if delta == 0 {
		return
	}
	c.setPixels(c.list.ScrollOffset() - c.applyPhysicsToUserOffset(delta))
}

// flingLimits are the physics' minFlingDistance, minFlingVelocity and
// maxFlingVelocity, which ScrollableState gives its drag recognizer.
func (c *ScrollController) flingLimits() (minDistance, minVelocity, maxVelocity float64) {
	minVelocity, maxVelocity = MinFlingVelocity, MaxFlingVelocity
	if c.bouncing() {
		minVelocity = MinFlingVelocity * 2
	}
	if c.physics() == BouncingFastPhysics {
		maxVelocity = MaxFlingVelocity * 8
	}
	return TouchSlop, minVelocity, maxVelocity
}

// dragEnd is ScrollDragController.end: the drag recognizer's primary
// velocity (already the fling decision), negated, goes ballistic.
func (c *ScrollController) dragEnd(velocity float64) { c.goBallistic(-velocity) }

// dragCancel is ScrollHoldController/ScrollDragController.cancel.
func (c *ScrollController) dragCancel() { c.goBallistic(0) }

// pointerScroll is ScrollPosition.pointerScroll: a wheel delta, clamped to
// the scroll range on every platform.
func (c *ScrollController) pointerScroll(delta float64) {
	if delta == 0 {
		c.goBallistic(0)
		return
	}
	target := math.Max(0, math.Min(c.list.MaxScrollExtent(), c.list.ScrollOffset()+delta))
	if target != c.list.ScrollOffset() {
		c.goIdle()
		c.list.SetScrollOffset(target)
	}
	c.goBallistic(0)
}
