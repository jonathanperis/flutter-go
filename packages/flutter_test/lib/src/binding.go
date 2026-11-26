package flutter_test

// A partial port of flutter_test's binding.dart (Flutter 3.50.1): the
// scheduler side of AutomatedTestWidgetsFlutterBinding, the binding
// `testWidgets` runs in. The rest of WidgetsFlutterBinding (gestures,
// services, rendering, widgets bindings) is not ported: the persistent
// frame callback draws the current Tester's BuildOwner, as
// RendererBinding/WidgetsBinding's drawFrame would.
//
// As in flutter_test, the binding is a process-wide singleton reused by
// every test; each test (Tester) gets its own fake event loop (Dart: its
// own FakeAsync zone), whose clock is the frame clock.

import (
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AutomatedTestWidgetsFlutterBinding is the binding headless widget tests
// run in (partial: BindingBase and SchedulerBinding). It controls time:
// frames are produced only by Tester.Pump, from the test's fake clock.
type AutomatedTestWidgetsFlutterBinding struct {
	scheduler.SchedulerBindingMixin
	tester *Tester // the test in progress (flutter_test: inTest)
}

var automatedTestBinding *AutomatedTestWidgetsFlutterBinding

// AutomatedTestWidgetsFlutterBindingEnsureInitialized returns the binding,
// creating it the first time (AutomatedTestWidgetsFlutterBinding
// .ensureInitialized()).
func AutomatedTestWidgetsFlutterBindingEnsureInitialized() *AutomatedTestWidgetsFlutterBinding {
	if automatedTestBinding == nil {
		b := &AutomatedTestWidgetsFlutterBinding{}
		base := &foundation.BindingBaseBase{}
		b.SchedulerBindingMixin.BindingBase = base
		b.SetSelf(b)
		base.Construct(b)
		automatedTestBinding = b
	}
	return automatedTestBinding
}

// InitInstances also resets timeDilation (TestWidgetsFlutterBinding) and
// adds the persistent frame callback that draws the tree
// (RendererBinding's _handlePersistentFrameCallback).
func (b *AutomatedTestWidgetsFlutterBinding) InitInstances() {
	b.SchedulerBindingMixin.InitInstances()
	scheduler.SetTimeDilation(1.0) // just in case the developer has artificially changed it for development
	b.AddPersistentFrameCallback(b.handlePersistentFrameCallback)
}

func (b *AutomatedTestWidgetsFlutterBinding) handlePersistentFrameCallback(time.Duration) {
	t := b.tester
	if t == nil || t.Owner.Root() == nil { // no tree yet (pumpWidget not called)
		return
	}
	t.Canvas = &r.Recorder{}
	t.Owner.DrawFrame(t.Size, t.Canvas)
}

// EnsureFrameCallbacksRegistered leaves the PlatformDispatcher alone: frames
// come from Pump, never from the engine.
func (b *AutomatedTestWidgetsFlutterBinding) EnsureFrameCallbacksRegistered() {
	pd := b.PlatformDispatcher()
	if pd.OnDrawFrame() != nil || pd.OnBeginFrame() != nil {
		panic(dart.NewAssertionError(nil))
	}
}

// ScheduleWarmUpFrame runs the frame at once (no timers), so that the
// startup warm-up frame doesn't leave timers behind.
func (b *AutomatedTestWidgetsFlutterBinding) ScheduleWarmUpFrame() {
	t := b.inTest()
	b.HandleBeginFrame(nil)
	t.loop.FlushMicrotasks()
	b.HandleDrawFrame()
	t.loop.FlushMicrotasks()
}

func (b *AutomatedTestWidgetsFlutterBinding) inTest() *Tester {
	if b.tester == nil {
		panic(dart.NewAssertionError("not in a test"))
	}
	return b.tester
}

// testClockEpoch is DateTime.utc(2015): flutter_test's fake clock starts
// there (fakeAsync.getClock(DateTime.utc(2015))), and frame time stamps
// are its microsecondsSinceEpoch.
const testClockEpoch = 1420070400 * time.Second

// Pump is AutomatedTestWidgetsFlutterBinding.pump: advance the fake clock by
// duration (running due timers and microtasks), then, if a frame is
// scheduled, flush microtasks, HandleBeginFrame at the clock's time, flush
// microtasks (the engine's mid-frame drain), HandleDrawFrame; finally flush
// microtasks.
//
// An exception reported to FlutterErrorOnError meanwhile (the scheduler
// catches what frame callbacks throw) is re-raised as a panic once the pump
// is done (flutter_test records it and fails the test; TakeException
// retrieves it instead).
//
// Changing Tester.Size is a metrics change: the next pump forces a frame,
// as WidgetsBinding.handleMetricsChanged does.
func (b *AutomatedTestWidgetsFlutterBinding) Pump(duration ...time.Duration) {
	t := b.inTest()
	if len(duration) > 0 {
		t.loop.Elapse(duration[0])
	}
	if t.Size != t.lastSize {
		t.lastSize = t.Size
		b.ScheduleForcedFrame()
	}
	if b.HasScheduledFrame() {
		t.loop.FlushMicrotasks()
		rawTimeStamp := testClockEpoch + t.loop.Elapsed()
		b.HandleBeginFrame(&rawTimeStamp)
		t.loop.FlushMicrotasks()
		b.HandleDrawFrame()
	}
	t.loop.FlushMicrotasks()
	if e := t.TakeException(); e != nil {
		panic(e)
	}
}

// startTest makes t the test in progress, on its own fake event loop.
func (b *AutomatedTestWidgetsFlutterBinding) startTest(t *Tester) {
	if b.tester != nil {
		panic(dart.NewAssertionError("a test is already running"))
	}
	b.tester = t
	t.restoreLoop = async.SetCurrent(t.loop)
	t.lastSize = t.Size
	t.previousOnError = foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		if t.pendingException == nil {
			t.pendingException = details
		}
	}
}

// postTest unmounts the tree (flutter_test's post-test "runApp(Container)"
// and pump), checks the invariants flutter_test checks after every test
// (no transient callbacks left, no pending timers, timeDilation reset), and
// resets the binding's per-test state.
func (b *AutomatedTestWidgetsFlutterBinding) postTest(t *Tester) {
	defer func() {
		b.tester = nil
		t.restoreLoop()
		foundation.FlutterErrorOnError = t.previousOnError
		b.ResetInternalState()
	}()
	if e := t.TakeException(); e != nil {
		t.t.Errorf("An exception was reported and not taken: %v", e)
	}
	// runApp(Container(...)) (whose scheduleWarmUpFrame runs a frame at
	// once in this binding), then pump().
	t.Owner.RunApp(widgets.SizedBox{})
	b.ScheduleWarmUpFrame()
	b.Pump()
	if n := b.TransientCallbackCount(); n > 0 {
		t.t.Errorf("An animation is still running even after the widget tree was disposed (%d transient callbacks).", n)
		// Cancel them so they don't run in the next test's frames (ids are
		// handed out in sequence; the next one bounds those in use).
		noFrame := false
		last := b.ScheduleFrameCallback(func(time.Duration) {}, scheduler.ScheduleFrameCallbackOptions{ScheduleNewFrame: &noFrame})
		for id := 1; id <= last; id++ {
			b.CancelFrameCallbackWithId(id)
		}
	}
	if timers := t.loop.PendingTimers(); len(timers) > 0 {
		t.t.Errorf("A Timer is still pending even after the widget tree was disposed (%d timers).", len(timers))
	}
	if scheduler.TimeDilation() != 1.0 {
		t.t.Errorf("The timeDilation was changed and not reset by the test.")
	}
}
