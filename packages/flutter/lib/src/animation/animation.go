// Package animation is flutter-go's animation layer: AnimationController
// (Controller, not yet ported in full) and curves, driven by the scheduler
// library's Tickers. Values match Flutter's frame by frame (see difftest).
package animation

import (
	"math"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
)

// ---------------------------------------------------------------------------
// AnimationController

type Status int

const (
	Dismissed Status = iota
	Forward
	Reverse
	Completed
)

// Controller drives a value over time. Bounded controllers go from 0 to 1;
// use NewUnboundedController for arbitrary values (e.g. scroll offsets).
type Controller struct {
	Duration  time.Duration
	value     float64
	lower     float64
	upper     float64
	ticker    *scheduler.Ticker
	from, to  float64
	simDur    time.Duration
	curve     Curve
	sim       physics.Simulation // set by AnimateWith; nil for interpolation
	status    Status
	listeners []func()
	statusFns []func(Status)
}

// NewController creates a controller from 0 to 1 whose ticker comes from
// vsync (Flutter's AnimationController(vsync:, duration:)).
func NewController(vsync scheduler.TickerProvider, d time.Duration) *Controller {
	c := &Controller{Duration: d, lower: 0, upper: 1}
	c.ticker = vsync.CreateTicker(c.tick)
	return c
}

// NewUnboundedController creates a controller without bounds starting at
// value (Flutter's AnimationController.unbounded).
func NewUnboundedController(vsync scheduler.TickerProvider, value float64) *Controller {
	c := &Controller{value: value, lower: math.Inf(-1), upper: math.Inf(1)}
	c.ticker = vsync.CreateTicker(c.tick)
	return c
}

func (c *Controller) Value() float64       { return c.value }
func (c *Controller) Status() Status       { return c.status }
func (c *Controller) IsAnimating() bool    { return c.ticker.IsActive() }
func (c *Controller) AddListener(f func()) { c.listeners = append(c.listeners, f) }
func (c *Controller) AddStatusListener(f func(Status)) {
	c.statusFns = append(c.statusFns, f)
}

// SetValue stops any animation and jumps to v.
func (c *Controller) SetValue(v float64) {
	c.stopTicker(true)
	c.value = math.Max(c.lower, math.Min(c.upper, v))
	c.notify()
}

func (c *Controller) Forward() { c.animate(c.upper, c.scaledDuration(c.upper), Curves.Linear, Forward) }
func (c *Controller) Reverse() { c.animate(c.lower, c.scaledDuration(c.lower), Curves.Linear, Reverse) }

// AnimateTo animates to target over d (not scaled) with curve.
func (c *Controller) AnimateTo(target float64, d time.Duration, curve Curve) {
	st := Forward
	if target < c.value {
		st = Reverse
	}
	c.animate(target, d, curve, st)
}

// scaledDuration is Flutter's: the full duration scaled by the fraction of
// the range left to travel, rounded to whole microseconds.
func (c *Controller) scaledDuration(target float64) time.Duration {
	frac := math.Abs(target-c.value) / (c.upper - c.lower)
	return time.Duration(math.Round(float64(c.Duration.Microseconds())*frac)) * time.Microsecond
}

func (c *Controller) animate(target float64, d time.Duration, curve Curve, st Status) {
	c.stopTicker(true)
	c.sim = nil
	c.from, c.to, c.simDur, c.curve = c.value, target, d, curve
	c.setStatus(st)
	if d == 0 || c.from == c.to {
		c.value = target
		c.notify()
		c.finish()
		return
	}
	c.ticker.Start()
}

func (c *Controller) tick(elapsed time.Duration) {
	if c.sim != nil {
		secs := float64(elapsed.Microseconds()) / 1e6
		c.value = math.Max(c.lower, math.Min(c.upper, c.sim.X(secs)))
		done := c.sim.IsDone(secs)
		if done {
			c.stopTicker(false)
			c.sim = nil
		}
		// Flutter notifies value listeners before the status change (and
		// the TickerFuture completion that scroll activities wait for).
		c.notify()
		if done {
			c.setStatus(Completed)
		}
		return
	}
	// Same arithmetic as Flutter's _InterpolationSimulation.
	t := float64(elapsed.Microseconds()) / 1e6 / (float64(c.simDur.Microseconds()) / 1e6)
	t = math.Max(0, math.Min(1, t))
	switch t {
	case 0:
		c.value = c.from
	case 1:
		c.value = c.to
	default:
		c.value = c.from + float64((c.to-c.from)*c.curve.Transform(t))
	}
	c.notify()
	if t == 1 {
		c.stopTicker(false)
		c.finish()
	}
}

func (c *Controller) finish() {
	switch {
	case c.value == c.upper:
		c.setStatus(Completed)
	case c.value == c.lower:
		c.setStatus(Dismissed)
	default:
		c.setStatus(Completed)
	}
}

func (c *Controller) setStatus(s Status) {
	if s == c.status {
		return
	}
	c.status = s
	for _, f := range c.statusFns {
		f(s)
	}
}

func (c *Controller) notify() {
	for _, f := range c.listeners {
		f()
	}
}

// stopTicker is AnimationController.stop(canceled:): a completed animation
// stops with canceled false, an interrupted one with canceled true.
func (c *Controller) stopTicker(canceled bool) {
	c.ticker.Stop(scheduler.TickerStopOptions{Canceled: canceled})
}

// Dispose releases the ticker; the controller can't be used afterwards.
func (c *Controller) Dispose() { c.ticker.Dispose(); c.listeners, c.statusFns = nil, nil }

// LerpDouble is dart:ui's lerpDouble, a*(1-t) + b*t. Flutter uses this form
// for sizes and colors; the rounding differs slightly from a + (b-a)*t.
func LerpDouble(a, b, t float64) float64 { return float64(a*(1-t)) + float64(b*t) }
