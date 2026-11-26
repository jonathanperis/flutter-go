package flutter_test

// A partial port of flutter_test's binding.dart (Flutter 3.50.1):
// AutomatedTestWidgetsFlutterBinding, the binding `testWidgets` runs in,
// composed as TestWidgetsFlutterBinding is (BindingBase with
// SchedulerBinding, ServicesBinding, GestureBinding, SemanticsBinding,
// PaintingBinding, TestDefaultBinaryMessengerBinding; RendererBinding and
// WidgetsBinding are not in it yet). The persistent
// frame callback draws the current Tester's BuildOwner, as
// RendererBinding/WidgetsBinding's drawFrame would.
//
// As in flutter_test, the binding is a process-wide singleton reused by
// every test; each test (Tester) gets its own fake event loop (Dart: its
// own FakeAsync zone), whose clock is the frame clock.

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AutomatedTestWidgetsFlutterBinding is the binding headless widget tests
// run in (partial, see the file comment). It controls time: frames are
// produced only by Tester.Pump, from the test's fake clock.
type AutomatedTestWidgetsFlutterBinding struct {
	foundation.BindingBaseBase
	scheduler.SchedulerBindingMixin
	services.ServicesBindingMixin
	gestures.GestureBindingMixin
	semantics.SemanticsBindingMixin
	painting.PaintingBindingMixin
	TestDefaultBinaryMessengerBindingMixin

	tester *Tester // the test in progress (flutter_test: inTest)

	testRestorationManager *TestRestorationManager
	testTextInput          *TestTextInput
}

var automatedTestBinding *AutomatedTestWidgetsFlutterBinding

// AutomatedTestWidgetsFlutterBindingEnsureInitialized returns the binding,
// creating it the first time (AutomatedTestWidgetsFlutterBinding
// .ensureInitialized()).
func AutomatedTestWidgetsFlutterBindingEnsureInitialized() *AutomatedTestWidgetsFlutterBinding {
	if automatedTestBinding == nil {
		b := &AutomatedTestWidgetsFlutterBinding{}
		// TestWidgetsFlutterBinding(): debugDisableShadows = disableShadows.
		painting.DebugDisableShadows = b.DisableShadows()
		b.SchedulerBindingMixin.SetSelf(b)
		b.ServicesBindingMixin.SetSelf(b)
		b.GestureBindingMixin.SetSelf(b)
		b.SemanticsBindingMixin.SetSelf(b)
		b.PaintingBindingMixin.SetSelf(b)
		b.TestDefaultBinaryMessengerBindingMixin.SetSelf(b)
		loop := async.Current()
		b.Construct(b, &b.SchedulerBindingMixin, &b.ServicesBindingMixin, &b.GestureBindingMixin, &b.SemanticsBindingMixin,
			&b.PaintingBindingMixin, &b.TestDefaultBinaryMessengerBindingMixin)
		// ServicesBinding's channel buffers drain in a microtask, and the
		// engine answers initializationComplete with a posted event: run
		// them, as Dart's event loop would before the first test.
		loop.FlushMicrotasks()
		loop.RunPosted()
		loop.FlushMicrotasks()
		automatedTestBinding = b
	}
	return automatedTestBinding
}

// DisableShadows is the value debugDisableShadows has while tests run:
// true in AutomatedTestWidgetsFlutterBinding, as in Flutter (shadows paint
// as solid outlines, which are stable across platforms).
func (b *AutomatedTestWidgetsFlutterBinding) DisableShadows() bool { return true }

// InitInstances also resets timeDilation (TestWidgetsFlutterBinding) and
// adds the persistent frame callback that draws the tree
// (RendererBinding's _handlePersistentFrameCallback).
func (b *AutomatedTestWidgetsFlutterBinding) InitInstances() {
	b.TestDefaultBinaryMessengerBindingMixin.InitInstances()
	b.testTextInput = NewTestTextInput() // onCleared: _resetFocusedEditable (no EditableText yet)
	scheduler.SetTimeDilation(1.0)       // just in case the developer has artificially changed it for development
	b.AddPersistentFrameCallback(b.handlePersistentFrameCallback)
}

// The methods several mixins define (Go can't promote them) are the last
// mixin's in `with` order, as in Dart.

func (b *AutomatedTestWidgetsFlutterBinding) InitServiceExtensions() {
	b.ServicesBindingMixin.InitServiceExtensions()
}
func (b *AutomatedTestWidgetsFlutterBinding) Unlocked()          { b.GestureBindingMixin.Unlocked() }
func (b *AutomatedTestWidgetsFlutterBinding) Evict(asset string) { b.PaintingBindingMixin.Evict(asset) }
func (b *AutomatedTestWidgetsFlutterBinding) HandleMemoryPressure() {
	b.PaintingBindingMixin.HandleMemoryPressure()
}
func (b *AutomatedTestWidgetsFlutterBinding) HandleSystemMessage(systemMessage any) async.Future[async.Void] {
	return b.PaintingBindingMixin.HandleSystemMessage(systemMessage)
}

// DebugSamplingClock is the test clock (_TestSamplingClock): fake time on
// the test's own event loop (testWidgets' FakeAsync clock, even inside a
// nested FakeAsync such as testGesture's), from 2015-01-01 as
// flutter_test's clock; outside a test, the current loop's.
func (b *AutomatedTestWidgetsFlutterBinding) DebugSamplingClock() gestures.SamplingClock {
	if b.tester != nil {
		return testSamplingClock{loop: b.tester.loop}
	}
	return testSamplingClock{loop: async.Current()}
}

type testSamplingClock struct{ loop *async.Loop }

func (c testSamplingClock) Now() time.Time {
	return time.Unix(0, 0).UTC().Add(testClockEpoch + c.loop.Elapsed())
}

func (c testSamplingClock) Stopwatch() *dart.Stopwatch {
	return dart.NewStopwatchWithClock(c.loop.Elapsed)
}
func (b *AutomatedTestWidgetsFlutterBinding) CreateBinaryMessenger() services.BinaryMessenger {
	return b.TestDefaultBinaryMessengerBindingMixin.CreateBinaryMessenger()
}

// HitTestInView is RendererBinding's: the test's tree (its gesture
// targets, see widgets.BuildOwner.HitTestGestures), then GestureBinding's.
func (b *AutomatedTestWidgetsFlutterBinding) HitTestInView(result *gestures.HitTestResult, position ui.Offset, viewID int) {
	if b.tester != nil {
		b.tester.Owner.HitTestGestures(result, position)
	}
	b.GestureBindingMixin.HitTestInView(result, position, viewID)
}

// PerformSemanticsAction is RendererBinding's (not ported): actions are
// ignored.
func (b *AutomatedTestWidgetsFlutterBinding) PerformSemanticsAction(action ui.SemanticsActionEvent) {}

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
	// TestWidgetsFlutterBinding.reset().
	if b.testRestorationManager != nil {
		b.testRestorationManager.Dispose()
		b.testRestorationManager = nil
	}
	// registerTestTextInput is true: the text input mock is reset and
	// registered for each test.
	b.testTextInput.Reset()
	b.testTextInput.Register()
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
		b.testTextInput.Unregister()
		b.ResetGestureBinding()
		// TestWidgetsFlutterBinding's per-test keyboard reset.
		b.KeyEventManager().KeyMessageHandler = nil
		services.RawKeyboardInstance.ClearKeysPressed()
		services.HardwareKeyboardInstance().ClearState()
		b.KeyEventManager().ClearState()
		t.restoreLoop()
		foundation.FlutterErrorOnError = t.previousOnError
		b.ResetInternalState()
		if b.testRestorationManager != nil {
			b.testRestorationManager.Dispose()
			b.testRestorationManager = nil
		}
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
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.t.Errorf("%v", r)
			}
		}()
		painting.DebugAssertAllPaintingVarsUnset("The value of a painting debug variable was changed by the test.",
			painting.DebugAssertAllPaintingVarsUnsetOptions{DebugDisableShadowsOverride: b.DisableShadows()})
	}()
}

// TestRestorationManager is TestWidgetsFlutterBinding's covariant
// restorationManager: the test's TestRestorationManager, created on first
// use and disposed at the end of each test.
func (b *AutomatedTestWidgetsFlutterBinding) TestRestorationManager() *TestRestorationManager {
	if b.testRestorationManager == nil {
		b.testRestorationManager = NewTestRestorationManager()
	}
	return b.testRestorationManager
}

// RestorationManager is the test's TestRestorationManager (its embedded
// RestorationManager; TestRestorationManager returns the whole).
func (b *AutomatedTestWidgetsFlutterBinding) RestorationManager() *services.RestorationManager {
	return &b.TestRestorationManager().RestorationManager
}

// CreateRestorationManager creates a TestRestorationManager.
func (b *AutomatedTestWidgetsFlutterBinding) CreateRestorationManager() *services.RestorationManager {
	return &NewTestRestorationManager().RestorationManager
}

// TestTextInput is the binding's mock of the platform's text input
// (testTextInput): reset and registered for every test.
func (b *AutomatedTestWidgetsFlutterBinding) TestTextInput() *TestTextInput { return b.testTextInput }
