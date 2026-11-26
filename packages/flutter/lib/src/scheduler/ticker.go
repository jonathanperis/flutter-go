package scheduler

// Port of ticker.dart (Flutter 3.50.1).
//
// Ticker is a concrete class Flutter extends (widgets' _WidgetTicker), so,
// per API_MAPPING, it keeps a pointer to the outer object (self) and calls
// its overridable members through it; a type embedding Ticker calls
// SetSelf once when created. TickerFuture implements async.Future (a
// user-defined future, as Dart's TickerFuture implements Future<void>).

import (
	"slices"
	"strings"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// TickerCallback is the signature of the callback passed to NewTicker: the
// time elapsed from the frame time stamp when the ticker was last started
// to the current frame time stamp.
type TickerCallback func(elapsed time.Duration)

// TickerProvider is implemented by objects that vend Tickers (in Flutter,
// a State with TickerProviderStateMixin, or the WidgetTester).
// AnimationControllers need one to obtain their Ticker.
type TickerProvider interface {
	// CreateTicker creates a ticker with the given callback.
	CreateTicker(onTick TickerCallback) *Ticker
}

// tickerSelf is the set of Ticker members a type embedding Ticker may
// override; Ticker calls them through self.
type tickerSelf interface {
	Muted() bool
	SetMuted(value bool)
	IsTicking() bool
	IsActive() bool
	Scheduled() bool
	ShouldScheduleTick() bool
	ScheduleTick(opts ...TickerScheduleTickOptions)
	UnscheduleTick()
	Dispose()
	ToString(opts ...TickerToStringOptions) string
}

// Ticker calls its callback once per animation frame, when enabled.
//
// When created, a ticker is disabled: call Start. A ticker can be silenced
// by setting Muted: time still elapses and Start/Stop still work, but no
// callbacks are called. By convention Start and Stop are used by the
// ticker's consumer (an AnimationController), and Muted by the
// TickerProvider that created it.
type Ticker struct {
	self tickerSelf

	future *TickerFuture

	// ForceFrames makes this ticker request frames with
	// SchedulerBinding.ScheduleForcedFrame instead of ScheduleFrame, so it
	// advances even when frames are disabled (e.g. in the background).
	ForceFrames bool

	muted       bool
	startTime   *time.Duration
	onTick      TickerCallback
	animationID *int

	// DebugLabel appears in String() in debug builds ("" stands for Dart's
	// null label: both print nothing).
	DebugLabel string

	debugCreationStack dart.StackTrace
}

// TickerOptions are NewTicker's named parameters.
type TickerOptions struct {
	// DebugLabel appears in String() in debug builds.
	DebugLabel string
}

// NewTicker creates a ticker that will call onTick once per frame while
// running.
func NewTicker(onTick TickerCallback, opts ...TickerOptions) *Ticker {
	t := &Ticker{onTick: onTick}
	if len(opts) > 0 {
		t.DebugLabel = opts[0].DebugLabel
	}
	t.self = t
	if foundation.KDebugMode {
		t.debugCreationStack = dart.StackTraceCurrent()
	}
	dartAssert(foundation.DebugMaybeDispatchCreated("scheduler", "Ticker", t), nil)
	return t
}

// SetSelf points the ticker at the object embedding it, whose overrides
// (Dispose, ScheduleTick...) the ticker calls. Call it once when creating
// such an object.
func (t *Ticker) SetSelf(self tickerSelf) { t.self = self }

func (t *Ticker) outer() tickerSelf {
	if t.self == nil {
		panic("scheduler: Ticker used before SetSelf; create it with NewTicker, or call SetSelf when embedding it")
	}
	return t.self
}

// Muted reports whether this ticker has been silenced.
func (t *Ticker) Muted() bool { return t.muted }

// SetMuted silences (true) or unsilences (false) the ticker: muting
// unschedules a scheduled tick (not the frame); unmuting may schedule one.
func (t *Ticker) SetMuted(value bool) {
	self := t.outer()
	if value == self.Muted() {
		return
	}
	t.muted = value
	if value {
		self.UnscheduleTick()
	} else if self.ShouldScheduleTick() {
		self.ScheduleTick()
	}
}

// IsTicking reports whether the ticker will call its callback on the next
// frame: it is active, not muted, and frames are enabled (or a frame is in
// progress, e.g. a warm-up or forced frame).
func (t *Ticker) IsTicking() bool {
	if t.future == nil {
		return false
	}
	if t.outer().Muted() {
		return false
	}
	if SchedulerBindingInstance().FramesEnabled() {
		return true
	}
	if SchedulerBindingInstance().SchedulerPhase() != SchedulerPhaseIdle {
		return true
	} // for example, we might be in a warm-up frame or forced frame
	return false
}

// IsActive reports whether time is elapsing for this ticker: true from
// Start until Stop.
func (t *Ticker) IsActive() bool { return t.future != nil }

// Start starts the clock (and, unless muted, the callbacks once per
// animation frame). The returned future completes when the ticker stops;
// if it is disposed or canceled, use TickerFuture.OrCancel. The ticker
// must not be active.
//
// Started during a frame (before the post-frame callbacks), the ticker
// counts from that frame's time stamp; otherwise from its first tick.
func (t *Ticker) Start() *TickerFuture {
	self := t.outer()
	if foundation.KDebugMode && self.IsActive() {
		panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
			foundation.NewErrorSummary("A ticker was started twice."),
			foundation.NewErrorDescription("A ticker that is already active cannot be started again without first stopping it."),
			t.DescribeForError("The affected ticker was"),
		}))
	}
	dartAssert(t.startTime == nil, nil)
	t.future = newTickerFuture()
	if self.ShouldScheduleTick() {
		self.ScheduleTick()
	}
	binding := SchedulerBindingInstance()
	if binding.SchedulerPhase() > SchedulerPhaseIdle && binding.SchedulerPhase() < SchedulerPhasePostFrameCallbacks {
		startTime := binding.CurrentFrameTimeStamp()
		t.startTime = &startTime
	}
	return t.future
}

// DescribeForError is a debug representation of the ticker for error
// messages.
func (t *Ticker) DescribeForError(name string) foundation.DiagnosticsNode {
	description := t.outer().ToString(TickerToStringOptions{DebugIncludeStack: true})
	return foundation.NewDiagnosticsProperty(name, t, foundation.DiagnosticsPropertyOptions{Description: &description})
}

// TickerStopOptions are Stop's named parameters.
type TickerStopOptions struct {
	// Canceled: the future returned by Start does not complete, and
	// OrCancel's fails with TickerCanceled.
	Canceled bool
}

// Stop stops calling the callback and completes the future Start returned
// (or, when canceled, fails its OrCancel future). It does nothing if the
// ticker is not active.
func (t *Ticker) Stop(opts ...TickerStopOptions) {
	self := t.outer()
	if !self.IsActive() {
		return
	}
	canceled := len(opts) > 0 && opts[0].Canceled

	// We take the _future into a local variable so that isTicking is false
	// when we actually complete the future (isTicking uses _future to
	// determine its state).
	localFuture := t.future
	t.future = nil
	t.startTime = nil
	dartAssert(!self.IsActive(), nil)

	self.UnscheduleTick()
	if canceled {
		localFuture.cancel(t)
	} else {
		localFuture.complete()
	}
}

// Scheduled reports whether a frame callback is scheduled (protected).
func (t *Ticker) Scheduled() bool { return t.animationID != nil }

// ShouldScheduleTick reports whether a tick should be scheduled: not muted,
// active, and not already scheduled (protected).
func (t *Ticker) ShouldScheduleTick() bool {
	self := t.outer()
	return !self.Muted() && self.IsActive() && !self.Scheduled()
}

func (t *Ticker) tick(timeStamp time.Duration) {
	self := t.outer()
	dartAssert(self.IsTicking(), nil)
	dartAssert(self.Scheduled(), nil)
	t.animationID = nil

	if t.startTime == nil {
		t.startTime = &timeStamp
	}
	t.onTick(timeStamp - *t.startTime)

	// The onTick callback may have scheduled another tick already, for
	// example by calling stop then start again.
	if self.ShouldScheduleTick() {
		self.ScheduleTick(TickerScheduleTickOptions{Rescheduling: true})
	}
}

// TickerScheduleTickOptions are ScheduleTick's named parameters.
type TickerScheduleTickOptions struct {
	// Rescheduling: the tick is scheduled from the previous tick.
	Rescheduling bool
}

// ScheduleTick schedules a tick for the next frame; only call it when
// ShouldScheduleTick is true (protected).
func (t *Ticker) ScheduleTick(opts ...TickerScheduleTickOptions) {
	self := t.outer()
	rescheduling := len(opts) > 0 && opts[0].Rescheduling
	dartAssert(!self.Scheduled(), nil)
	dartAssert(self.ShouldScheduleTick(), nil)
	binding := SchedulerBindingInstance()
	if t.ForceFrames {
		binding.ScheduleForcedFrame()
	} else {
		binding.ScheduleFrame()
	}
	scheduleNewFrame := false
	id := binding.ScheduleFrameCallback(t.tick, ScheduleFrameCallbackOptions{
		Rescheduling:     rescheduling,
		ScheduleNewFrame: &scheduleNewFrame,
	})
	t.animationID = &id
}

// UnscheduleTick cancels the frame callback ScheduleTick requested, if any
// (protected).
func (t *Ticker) UnscheduleTick() {
	self := t.outer()
	if self.Scheduled() {
		SchedulerBindingInstance().CancelFrameCallbackWithId(*t.animationID)
		t.animationID = nil
	}
	dartAssert(!self.ShouldScheduleTick(), nil)
}

// AbsorbTicker makes this ticker take the state of originalTicker (keeping
// the identity of its TickerFuture when it is active) and disposes
// originalTicker. This ticker must not be active.
func (t *Ticker) AbsorbTicker(originalTicker *Ticker) {
	self := t.outer()
	dartAssert(!self.IsActive(), nil)
	dartAssert(t.future == nil, nil)
	dartAssert(t.startTime == nil, nil)
	dartAssert(t.animationID == nil, nil)
	dartAssert(originalTicker.future != nil || originalTicker.startTime == nil,
		"Cannot absorb Ticker after it has been disposed.")
	if originalTicker.future != nil {
		t.future = originalTicker.future
		t.startTime = originalTicker.startTime
		if self.ShouldScheduleTick() {
			self.ScheduleTick()
		}
		originalTicker.future = nil // so that it doesn't get disposed when we dispose of originalTicker
		originalTicker.outer().UnscheduleTick()
	}
	originalTicker.outer().Dispose()
}

// Dispose releases the ticker. If it is active, its frame callback is
// canceled, the future Start returned never completes, and its OrCancel
// future fails with TickerCanceled.
func (t *Ticker) Dispose() {
	self := t.outer()
	dartAssert(foundation.DebugMaybeDispatchDisposed(t), nil)
	if t.future != nil {
		localFuture := t.future
		t.future = nil
		dartAssert(!self.IsActive(), nil)
		self.UnscheduleTick()
		localFuture.cancel(t)
	}
	if foundation.KDebugMode {
		// We intentionally don't null out _startTime. This means that if start()
		// was ever called, the object is now in a bogus state. This weakly helps
		// catch cases of use-after-dispose.
		var zero time.Duration
		t.startTime = &zero
	}
}

// TickerToStringOptions are ToString's named parameters.
type TickerToStringOptions struct {
	// DebugIncludeStack adds the stack where the ticker was created (debug
	// builds).
	DebugIncludeStack bool
}

func (t *Ticker) String() string { return t.outer().ToString() }

// ToString is Dart's toString({debugIncludeStack}).
func (t *Ticker) ToString(opts ...TickerToStringOptions) string {
	debugIncludeStack := len(opts) > 0 && opts[0].DebugIncludeStack
	var buffer strings.Builder
	buffer.WriteString(foundation.ObjectRuntimeType(t.self, "Ticker") + "(")
	if foundation.KDebugMode {
		buffer.WriteString(t.DebugLabel)
	}
	buffer.WriteString(")")
	if foundation.KDebugMode && debugIncludeStack {
		buffer.WriteString("\n")
		buffer.WriteString("The stack trace when the " + foundation.ObjectRuntimeType(t.self, "Ticker") + " was actually created was:\n")
		stack := "null"
		if t.debugCreationStack != nil {
			stack = t.debugCreationStack.String()
		}
		lines := strings.Split(strings.TrimRight(stack, " \t\n\r"), "\n")
		for line := range foundation.FlutterErrorDefaultStackFilter(slices.Values(lines)) {
			buffer.WriteString(line + "\n")
		}
	}
	return buffer.String()
}

// TickerFuture represents an ongoing Ticker sequence: Ticker.Start returns
// one. It completes when the ticker is stopped (not canceled); if the
// ticker is disposed or stopped with Canceled, it never completes, so
// TickerFutures should generally not be awaited. OrCancel is a derived
// future that fails with TickerCanceled in that case;
// WhenCompleteOrCancel runs a callback either way.
type TickerFuture struct {
	primaryCompleter   *async.Completer[async.Void]
	secondaryCompleter *async.Completer[async.Void]
	completed          *bool // nil means unresolved, true means complete, false means canceled
}

var _ async.Future[async.Void] = (*TickerFuture)(nil)

func newTickerFuture() *TickerFuture {
	return &TickerFuture{primaryCompleter: async.NewCompleter[async.Void]()}
}

// NewTickerFutureComplete is TickerFuture.complete(): an already-complete
// ticker sequence (for animations that skip their ticker).
func NewTickerFutureComplete() *TickerFuture {
	f := newTickerFuture()
	f.complete()
	return f
}

func (f *TickerFuture) complete() {
	dartAssert(f.completed == nil, nil)
	completed := true
	f.completed = &completed
	f.primaryCompleter.Complete(async.Void{})
	if f.secondaryCompleter != nil {
		f.secondaryCompleter.Complete(async.Void{})
	}
}

func (f *TickerFuture) cancel(ticker *Ticker) {
	dartAssert(f.completed == nil, nil)
	completed := false
	f.completed = &completed
	if f.secondaryCompleter != nil {
		f.secondaryCompleter.CompleteError(TickerCanceled{Ticker: ticker})
	}
}

// WhenCompleteOrCancel calls callback when this future completes or when
// the ticker is canceled. It handles OrCancel's error, so canceling causes
// no uncaught error.
func (f *TickerFuture) WhenCompleteOrCancel(callback func()) {
	async.OnComplete(f.OrCancel(), func(async.Void) { callback() }, func(error) { callback() })
}

// OrCancel is a future that completes when this one does, or fails with
// TickerCanceled when the ticker is canceled. Once it has been read, an
// uncaught cancellation is an uncaught error.
func (f *TickerFuture) OrCancel() async.Future[async.Void] {
	if f.secondaryCompleter == nil {
		f.secondaryCompleter = async.NewCompleter[async.Void]()
		if f.completed != nil {
			if *f.completed {
				f.secondaryCompleter.Complete(async.Void{})
			} else {
				f.secondaryCompleter.CompleteError(TickerCanceled{})
			}
		}
	}
	return f.secondaryCompleter.Future()
}

// ThenAny is then (see async.Future): it delegates to the primary future.
func (f *TickerFuture) ThenAny(onValue func(async.Void) any, onError func(error) any) async.Future[any] {
	return f.primaryCompleter.Future().ThenAny(onValue, onError)
}

// CatchError delegates to the primary future.
func (f *TickerFuture) CatchError(onError func(error) async.Void, test func(error) bool) async.Future[async.Void] {
	return f.primaryCompleter.Future().CatchError(onError, test)
}

// Timeout delegates to the primary future.
func (f *TickerFuture) Timeout(timeLimit time.Duration, onTimeout func() async.Void) async.Future[async.Void] {
	return f.primaryCompleter.Future().Timeout(timeLimit, onTimeout)
}

// WhenComplete delegates to the primary future.
func (f *TickerFuture) WhenComplete(action func()) async.Future[async.Void] {
	return f.primaryCompleter.Future().WhenComplete(action)
}

// Ignore is Dart's ignore() on a user-defined future.
func (f *TickerFuture) Ignore() { async.IgnoreUserDefined[async.Void](f) }

// Await is Dart's await (through then, as for any user-defined future).
func (f *TickerFuture) Await() async.Void { return async.AwaitUserDefined[async.Void](f) }

func (f *TickerFuture) String() string {
	state := "active"
	if f.completed != nil {
		if *f.completed {
			state = "complete"
		} else {
			state = "canceled"
		}
	}
	return foundation.DescribeIdentity(f) + "(" + state + ")"
}

// TickerCanceled is the error of TickerFuture.OrCancel when the ticker is
// canceled.
type TickerCanceled struct {
	// Ticker is the ticker that was canceled; nil if the OrCancel future
	// was created after the ticker was canceled.
	Ticker *Ticker
}

func (e TickerCanceled) Error() string {
	if e.Ticker != nil {
		return "This ticker was canceled: " + e.Ticker.String()
	}
	return `The ticker was canceled before the "orCancel" property was first used.`
}
