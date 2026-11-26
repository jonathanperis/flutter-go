// Package animation is flutter-go's scheduler and animation layer: frame
// callbacks, tickers, AnimationController and curves. It mirrors Flutter's
// scheduler and animation libraries closely enough that values match
// Flutter's to within floating-point error (see difftest).
package animation

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"

	"math"
	"time"
)

// Scheduler runs transient frame callbacks (ticks) at the start of each frame,
// before build/layout/paint, as Flutter's SchedulerBinding does.
type Scheduler struct {
	now       time.Duration // timestamp of the current or last frame
	inFrame   bool
	callbacks map[int]func(time.Duration)
	nextID    int
	// RequestFrame is called when a callback is scheduled.
	RequestFrame func()
	// Clock returns the current time on the frame timeline (defaults to
	// the last frame's timestamp). Timers are measured against it.
	Clock  func() time.Duration
	timers []timer
}

type timer struct {
	at time.Duration
	id int
	fn func()
}

func (s *Scheduler) clock() time.Duration {
	if s.Clock != nil {
		return s.Clock()
	}
	return s.now
}

// AfterFunc runs fn once d has passed (Flutter uses dart:async Timers for
// things like the cursor blink, so idle apps don't draw frames).
func (s *Scheduler) AfterFunc(d time.Duration, fn func()) int {
	s.nextID++
	s.timers = append(s.timers, timer{s.clock() + d, s.nextID, fn})
	return s.nextID
}

func (s *Scheduler) CancelTimer(id int) {
	for i, t := range s.timers {
		if t.id == id {
			s.timers = append(s.timers[:i], s.timers[i+1:]...)
			return
		}
	}
}

// RunTimers runs every timer due at or before now, in deadline order.
func (s *Scheduler) RunTimers(now time.Duration) {
	for {
		best := -1
		for i, t := range s.timers {
			if t.at <= now && (best < 0 || t.at < s.timers[best].at) {
				best = i
			}
		}
		if best < 0 {
			return
		}
		t := s.timers[best]
		s.timers = append(s.timers[:best], s.timers[best+1:]...)
		t.fn()
	}
}

// NextTimer returns the earliest pending deadline.
func (s *Scheduler) NextTimer() (time.Duration, bool) {
	if len(s.timers) == 0 {
		return 0, false
	}
	next := s.timers[0].at
	for _, t := range s.timers[1:] {
		next = min(next, t.at)
	}
	return next, true
}

func NewScheduler() *Scheduler { return &Scheduler{callbacks: map[int]func(time.Duration){}} }

// Now returns the current frame's timestamp.
func (s *Scheduler) Now() time.Duration { return s.now }

// HasScheduledFrame reports whether any callback is waiting for a frame.
func (s *Scheduler) HasScheduledFrame() bool { return len(s.callbacks) > 0 }

func (s *Scheduler) schedule(cb func(time.Duration)) int {
	s.nextID++
	s.callbacks[s.nextID] = cb
	if s.RequestFrame != nil {
		s.RequestFrame()
	}
	return s.nextID
}

func (s *Scheduler) cancel(id int) { delete(s.callbacks, id) }

// BeginFrame runs the callbacks scheduled for this frame. Callbacks they
// schedule run in the next frame.
func (s *Scheduler) BeginFrame(ts time.Duration) {
	s.now, s.inFrame = ts, true
	cbs := s.callbacks
	s.callbacks = map[int]func(time.Duration){}
	ids := make([]int, 0, len(cbs))
	for id := range cbs {
		ids = append(ids, id)
	}
	sortInts(ids) // registration order, like Flutter
	for _, id := range ids {
		cbs[id](ts)
	}
}

// EndFrame marks the end of build/layout/paint.
func (s *Scheduler) EndFrame() { s.inFrame = false }

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

// Ticker calls onTick once per frame with the time elapsed since it started.
type Ticker struct {
	s        *Scheduler
	onTick   func(elapsed time.Duration)
	active   bool
	hasStart bool
	start    time.Duration
	id       int
}

func NewTicker(s *Scheduler, onTick func(elapsed time.Duration)) *Ticker {
	return &Ticker{s: s, onTick: onTick}
}

func (t *Ticker) IsActive() bool { return t.active }

// Start begins ticking. As in Flutter, a ticker started during a frame uses
// that frame's timestamp as its start; otherwise the first tick's timestamp.
func (t *Ticker) Start() {
	t.active, t.hasStart = true, false
	if t.s.inFrame {
		t.start, t.hasStart = t.s.now, true
	}
	t.id = t.s.schedule(t.tick)
}

func (t *Ticker) Stop() {
	t.active = false
	t.s.cancel(t.id)
}

func (t *Ticker) tick(ts time.Duration) {
	if !t.active {
		return
	}
	if !t.hasStart {
		t.start, t.hasStart = ts, true
	}
	t.id = t.s.schedule(t.tick) // keep ticking unless onTick stops us
	t.onTick(ts - t.start)
}

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
	ticker    *Ticker
	from, to  float64
	simDur    time.Duration
	curve     Curve
	sim       physics.Simulation // set by AnimateWith; nil for interpolation
	status    Status
	listeners []func()
	statusFns []func(Status)
}

func NewController(s *Scheduler, d time.Duration) *Controller {
	c := &Controller{Duration: d, lower: 0, upper: 1}
	c.ticker = NewTicker(s, c.tick)
	return c
}

func NewUnboundedController(s *Scheduler, value float64) *Controller {
	c := &Controller{value: value, lower: math.Inf(-1), upper: math.Inf(1)}
	c.ticker = NewTicker(s, c.tick)
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
	c.ticker.Stop()
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
	c.ticker.Stop()
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
			c.ticker.Stop()
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
		c.ticker.Stop()
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

func (c *Controller) Dispose() { c.ticker.Stop(); c.listeners, c.statusFns = nil, nil }

// LerpDouble is dart:ui's lerpDouble, a*(1-t) + b*t. Flutter uses this form
// for sizes and colors; the rounding differs slightly from a + (b-a)*t.
func LerpDouble(a, b, t float64) float64 { return float64(a*(1-t)) + float64(b*t) }
