package animation

// Test helpers shared by the ports of packages/flutter/test/animation:
// scheduler_tester.dart's tick, flutter_test's TestVSync, the setUp of
// those files (WidgetsFlutterBinding.ensureInitialized + resetEpoch), a
// pump for the testWidgets ones, and takeException.
//
// Dart runs each test file in its own isolate with its own binding; here
// every file shares one plain SchedulerBinding (one per process), and
// setUp cancels the transient callbacks earlier tests left behind (the
// Dart tests often don't dispose their controllers).

import (
	"math"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
)

const ms = time.Millisecond

// testBinding is the scheduler part of WidgetsFlutterBinding.
type testBinding struct {
	foundation.BindingBaseBase
	scheduler.SchedulerBindingMixin
}

func (b *testBinding) InitInstances()         { b.SchedulerBindingMixin.InitInstances() }
func (b *testBinding) InitServiceExtensions() { b.SchedulerBindingMixin.InitServiceExtensions() }
func (b *testBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }

var binding *testBinding

// ensureBinding returns the binding, created once per process.
func ensureBinding() *testBinding {
	if binding == nil {
		b := &testBinding{}
		b.SchedulerBindingMixin.SetSelf(b)
		b.Construct(b, &b.SchedulerBindingMixin)
		binding = b
	}
	return binding
}

// testVSync is flutter_test's TestVSync: a TickerProvider of plain
// Tickers.
type testVSync struct{}

func (testVSync) CreateTicker(onTick scheduler.TickerCallback) *scheduler.Ticker {
	return scheduler.NewTicker(onTick)
}

var vsync = testVSync{}

// testState is what setUp installs for one test.
type testState struct {
	loop       *async.Loop
	exceptions []any
}

// setUp is the Dart files' setUp (ensureInitialized, resetEpoch), plus a
// fake event loop for the test, the cancellation of transient callbacks
// left by earlier tests, and FlutterError.onError collecting reported
// errors (flutter_test's takeException).
func setUp(t *testing.T) *testState {
	t.Helper()
	b := ensureBinding()
	st := &testState{loop: async.NewLoop()}
	t.Cleanup(async.SetCurrent(st.loop))
	noFrame := false
	last := b.ScheduleFrameCallback(func(time.Duration) {}, scheduler.ScheduleFrameCallbackOptions{ScheduleNewFrame: &noFrame})
	for id := 1; id <= last; id++ {
		b.CancelFrameCallbackWithId(id)
	}
	b.ResetEpoch()
	previous := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		st.exceptions = append(st.exceptions, details.Exception)
	}
	semantics := SemanticsDisableAnimations
	t.Cleanup(func() {
		foundation.FlutterErrorOnError = previous
		SemanticsDisableAnimations = semantics
		if len(st.exceptions) > 0 {
			t.Errorf("exceptions reported and not taken: %v", st.exceptions)
		}
	})
	return st
}

// takeException returns the first exception reported since the last call.
func (st *testState) takeException() any {
	if len(st.exceptions) == 0 {
		return nil
	}
	e := st.exceptions[0]
	st.exceptions = nil
	return e
}

// tick is scheduler_tester.dart's tick: a frame at time stamp duration.
func tick(duration time.Duration) {
	ensureBinding().HandleBeginFrame(&duration)
	ensureBinding().HandleDrawFrame()
}

// pump is flutter_test's pump in the test's fake event loop: elapse, then
// a frame if one is scheduled (with microtasks flushed around its phases).
func (st *testState) pump(duration ...time.Duration) {
	if len(duration) > 0 {
		st.loop.Elapse(duration[0])
	}
	b := ensureBinding()
	if b.HasScheduledFrame() {
		st.loop.FlushMicrotasks()
		now := st.loop.Elapsed()
		b.HandleBeginFrame(&now)
		st.loop.FlushMicrotasks()
		b.HandleDrawFrame()
	}
	st.loop.FlushMicrotasks()
}

func dur(d time.Duration) *time.Duration { return &d }
func f64(v float64) *float64             { return &v }
func intp(v int) *int                    { return &v }

// near is flutter_test's moreOrLessEquals (epsilon 1e-10).
func near(a, b float64) bool { return math.Abs(a-b) <= 1e-10 }

// panics runs f and returns what it panicked with (nil if it didn't).
func panics(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return nil
}

func expect(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func expectStatuses(t *testing.T, got []AnimationStatus, want ...AnimationStatus) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("statuses %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("statuses %v, want %v", got, want)
		}
	}
}
