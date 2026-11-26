// Package async is the subset of dart:async that Flutter's API is built on:
// an event loop with microtasks and timers, Future/Completer, and
// coroutines (Async/Await) that suspend the way Dart's async functions do.
//
// The semantics follow the Dart 3 VM and package:fake_async (which
// flutter_test runs widget tests in), read from the SDK that ships with
// Flutter 3.50.1:
//   - microtasks run FIFO, and the queue is drained after every event;
//   - an asynchronous Completer (and Future.value) delivers its result in a
//     microtask; a sync Completer delivers immediately;
//   - listeners run in registration order, synchronously while the future
//     completes; a listener added to an already-completed future runs in a
//     microtask;
//   - an async function runs synchronously until its first Await;
//     afterwards its result is delivered synchronously when it returns.
//
// Everything runs on one logical thread (Dart's isolate): coroutines are
// goroutines, but exactly one of them, or the loop's caller, runs at a time.
package async

import (
	"fmt"
	"sync"
	"time"
)

// Loop is an isolate's event loop with a controllable clock, like
// package:fake_async: time moves only through Elapse/ElapseTo, so tests are
// deterministic and a real app advances it to the wall clock.
type Loop struct {
	elapsed    time.Duration
	elapsingTo *time.Duration
	microtasks []func()
	timers     []*Timer // creation order; ties on deadline fire in this order

	// UncaughtError receives errors no one handles: a panic in a microtask
	// or timer, or a failed Future without listeners. The default panics.
	UncaughtError func(err error)

	nullFut *future[Void]

	postMu   sync.Mutex
	posted   []func()
	postedCh chan struct{}
}

func NewLoop() *Loop { return &Loop{} }

var current = NewLoop()

// Current is the loop that package-level functions and new futures use.
func Current() *Loop { return current }

// SetCurrent makes l the current loop and returns a function restoring the
// previous one (as flutter_test gives each widget test its own FakeAsync).
func SetCurrent(l *Loop) (restore func()) {
	prev := current
	current = l
	return func() { current = prev }
}

// Elapsed is the loop's clock: the time elapsed since it was created.
func (l *Loop) Elapsed() time.Duration { return l.elapsed }

// MicrotaskCount is the number of queued microtasks.
func (l *Loop) MicrotaskCount() int { return len(l.microtasks) }

// PendingTimers returns the active timers in creation order.
func (l *Loop) PendingTimers() []*Timer { return append([]*Timer(nil), l.timers...) }

// ScheduleMicrotask queues f on the current loop (dart:async's
// scheduleMicrotask).
func ScheduleMicrotask(f func()) { current.ScheduleMicrotask(f) }

func (l *Loop) ScheduleMicrotask(f func()) { l.microtasks = append(l.microtasks, f) }

// FlushMicrotasks runs microtasks until the queue is empty, including those
// scheduled meanwhile.
func (l *Loop) FlushMicrotasks() {
	for len(l.microtasks) > 0 {
		f := l.microtasks[0]
		l.microtasks = l.microtasks[1:]
		l.guard(f)
	}
}

func (l *Loop) guard(f func()) {
	defer func() {
		if r := recover(); r != nil {
			l.reportUncaught(asError(r))
		}
	}()
	f()
}

func (l *Loop) reportUncaught(err error) {
	if l.UncaughtError != nil {
		l.UncaughtError(err)
		return
	}
	panic(fmt.Errorf("async: uncaught error: %w", err))
}

// Elapse advances the clock by d, firing due timers in deadline order and
// draining microtasks before and after each (FakeAsync.elapse).
func (l *Loop) Elapse(d time.Duration) {
	if d < 0 {
		panic("async: Elapse duration may not be negative")
	}
	l.ElapseTo(l.elapsed + d)
}

// ElapseTo advances the clock to t (no-op for times in the past, apart from
// running due timers and microtasks).
func (l *Loop) ElapseTo(t time.Duration) {
	if l.elapsingTo != nil {
		panic("async: cannot elapse until the previous elapse is complete")
	}
	if t < l.elapsed {
		t = l.elapsed
	}
	l.elapsingTo = &t
	defer func() { l.elapsingTo = nil }()
	l.fireTimersWhile(func(next *Timer) bool { return next.nextCall <= *l.elapsingTo })
	l.elapseTo(t)
}

// ElapseBlocking advances the clock by d without running timers or
// microtasks, as if a callback blocked for d (FakeAsync.elapseBlocking).
func (l *Loop) ElapseBlocking(d time.Duration) {
	if d < 0 {
		panic("async: ElapseBlocking duration may not be negative")
	}
	l.elapsed += d
	if l.elapsingTo != nil && l.elapsed > *l.elapsingTo {
		*l.elapsingTo = l.elapsed
	}
}

// FlushTimers runs timers until none are left (or only periodic ones, when
// flushPeriodic is false), panicking if that would take longer than timeout
// (FakeAsync.flushTimers).
func (l *Loop) FlushTimers(timeout time.Duration, flushPeriodic bool) {
	limit := l.elapsed + timeout
	l.fireTimersWhile(func(t *Timer) bool {
		if t.nextCall > limit {
			panic(fmt.Sprintf("async: exceeded timeout %v while flushing timers", timeout))
		}
		if t.nextCall <= l.elapsed {
			return true
		}
		if flushPeriodic {
			return true
		}
		for _, x := range l.timers {
			if !x.periodic {
				return true
			}
		}
		return false
	})
}

// NextDeadline is the earliest pending timer's deadline.
func (l *Loop) NextDeadline() (time.Duration, bool) {
	if t := l.nextTimer(); t != nil {
		return t.nextCall, true
	}
	return 0, false
}

func (l *Loop) nextTimer() *Timer {
	var best *Timer
	for _, t := range l.timers {
		if best == nil || t.nextCall < best.nextCall {
			best = t
		}
	}
	return best
}

func (l *Loop) fireTimersWhile(pred func(*Timer) bool) {
	l.FlushMicrotasks()
	for len(l.timers) > 0 {
		t := l.nextTimer()
		if !pred(t) {
			break
		}
		l.elapseTo(t.nextCall)
		t.fire()
		l.FlushMicrotasks()
	}
}

func (l *Loop) elapseTo(t time.Duration) {
	if t > l.elapsed {
		l.elapsed = t
	}
}

// Timer is dart:async's Timer.
type Timer struct {
	loop     *Loop
	duration time.Duration
	nextCall time.Duration
	periodic bool
	tick     int
	callback func(*Timer)
}

// NewTimer calls callback once after d (Dart's Timer(d, callback)).
func NewTimer(d time.Duration, callback func()) *Timer {
	return current.newTimer(d, false, func(*Timer) { callback() })
}

// NewPeriodicTimer calls callback every d (Timer.periodic).
func NewPeriodicTimer(d time.Duration, callback func(*Timer)) *Timer {
	return current.newTimer(d, true, callback)
}

// TimerRun runs callback as soon as possible, after pending microtasks
// (Timer.run).
func TimerRun(callback func()) { NewTimer(0, callback) }

func (l *Loop) newTimer(d time.Duration, periodic bool, cb func(*Timer)) *Timer {
	if d < 0 {
		d = 0
	}
	t := &Timer{loop: l, duration: d, nextCall: l.elapsed + d, periodic: periodic, callback: cb}
	l.timers = append(l.timers, t)
	return t
}

// Tick is the number of times the timer has fired (missed periodic ticks
// included).
func (t *Timer) Tick() int { return t.tick }

// IsActive reports whether the timer will still fire.
func (t *Timer) IsActive() bool {
	for _, x := range t.loop.timers {
		if x == t {
			return true
		}
	}
	return false
}

// IsPeriodic reports whether the timer repeats.
func (t *Timer) IsPeriodic() bool { return t.periodic }

// Cancel stops the timer.
func (t *Timer) Cancel() {
	for i, x := range t.loop.timers {
		if x == t {
			t.loop.timers = append(t.loop.timers[:i:i], t.loop.timers[i+1:]...)
			return
		}
	}
}

func (t *Timer) fire() {
	t.tick++
	if t.periodic {
		t.nextCall += t.duration
		for t.nextCall < t.loop.elapsed {
			t.tick++
			t.nextCall += t.duration
		}
	} else {
		t.Cancel()
	}
	t.loop.guard(func() { t.callback(t) })
}

// PanicError wraps a non-error value passed to panic, which Dart would have
// thrown.
type PanicError struct{ Value any }

func (e PanicError) Error() string { return fmt.Sprint(e.Value) }

func asError(r any) error {
	if err, ok := r.(error); ok {
		return err
	}
	return PanicError{r}
}

// ---- events from other goroutines ----

// Post queues f to run on the loop as an event from outside the isolate,
// like a message arriving on a ReceivePort (an isolate's result, I/O). It
// is the one Loop method that is safe to call from any goroutine. Posted
// events run only when the loop's owner calls RunPosted or WaitPosted (the
// fake clock never runs them), each as its own event: microtasks are
// drained after it.
func (l *Loop) Post(f func()) {
	l.postMu.Lock()
	l.posted = append(l.posted, f)
	ch := l.postedChannel()
	l.postMu.Unlock()
	select {
	case ch <- struct{}{}:
	default:
	}
}

// RunPosted runs the events posted so far, in arrival order, and returns
// how many ran.
func (l *Loop) RunPosted() int {
	l.postMu.Lock()
	events := l.posted
	l.posted = nil
	l.postMu.Unlock()
	for _, f := range events {
		l.guard(f)
		l.FlushMicrotasks()
	}
	return len(events)
}

// WaitPosted blocks until at least one event has been posted, then runs
// the posted events (RunPosted).
func (l *Loop) WaitPosted() {
	l.postMu.Lock()
	ch := l.postedChannel()
	l.postMu.Unlock()
	for l.RunPosted() == 0 {
		<-ch
	}
}

// postedChannel signals posts; call with postMu held.
func (l *Loop) postedChannel() chan struct{} {
	if l.postedCh == nil {
		l.postedCh = make(chan struct{}, 1)
	}
	return l.postedCh
}
