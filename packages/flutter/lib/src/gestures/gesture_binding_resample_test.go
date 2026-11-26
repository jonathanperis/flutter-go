package gestures_test

// Port of packages/flutter/test/gestures/gesture_binding_resample_event_test.dart
// (Flutter 3.50.1): its one test. gesture_binding_resample_event_on_widget_test.dart
// is skipped: it needs widgets (Listener) and the widget tester's
// resampling.

import (
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// resampleBinding is TestResampleEventFlutterBinding (BindingBase with
// GestureBinding, SchedulerBinding).
type resampleBinding struct {
	foundation.BindingBaseBase
	GestureBindingMixin
	scheduler.SchedulerBindingMixin

	callback          func(PointerEvent)
	postFrameCallback scheduler.FrameCallback
	frameTime         *time.Duration
	loop              *async.Loop
}

func (b *resampleBinding) InitInstances()         { b.SchedulerBindingMixin.InitInstances() }
func (b *resampleBinding) InitServiceExtensions() { b.SchedulerBindingMixin.InitServiceExtensions() }
func (b *resampleBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }

func (b *resampleBinding) HandleEvent(event PointerEvent, entry HitTestEntry) {
	b.GestureBindingMixin.HandleEvent(event, entry)
	if b.callback != nil {
		b.callback(event)
	}
}

func (b *resampleBinding) CurrentSystemFrameTimeStamp() time.Duration {
	if b.frameTime == nil {
		panic(dart.NewAssertionError(nil))
	}
	return *b.frameTime
}

func (b *resampleBinding) AddPostFrameCallback(callback scheduler.FrameCallback, opts ...scheduler.AddPostFrameCallbackOptions) {
	b.postFrameCallback = callback
}

func (b *resampleBinding) DebugSamplingClock() SamplingClock { return fakeClock{b.loop} }

// fakeClock is TestSamplingClock on fake_async's clock, which starts at
// DateTime.utc(2015).
type fakeClock struct{ loop *async.Loop }

var utc2015 = time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

func (c fakeClock) Now() time.Time { return utc2015.Add(c.loop.Elapsed()) }
func (c fakeClock) Stopwatch() *dart.Stopwatch {
	return dart.NewStopwatchWithClock(c.loop.Elapsed)
}

func TestGestureBindingResampleEventFile(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	binding := &resampleBinding{loop: loop}
	binding.GestureBindingMixin.SetSelf(binding)
	binding.SchedulerBindingMixin.SetSelf(binding)
	binding.Construct(binding, &binding.GestureBindingMixin, &binding.SchedulerBindingMixin)

	t.Run("Pointer event resampling", func(t *testing.T) {
		fake := fakeClock{loop}
		currentTime := func() time.Duration { return time.Duration(fake.Now().UnixMilli()) * time.Millisecond }
		epoch := currentTime()
		ms := time.Millisecond
		packet := ui.PointerDataPacket{Data: []ui.PointerData{
			{Change: ui.PointerChangeAdd, TimeStamp: epoch},
			{Change: ui.PointerChangeDown, TimeStamp: epoch + 10*ms},
			{Change: ui.PointerChangeMove, PhysicalX: 10, TimeStamp: epoch + 20*ms},
			{Change: ui.PointerChangeMove, PhysicalX: 20, TimeStamp: epoch + 30*ms},
			{Change: ui.PointerChangeMove, PhysicalX: 30, TimeStamp: epoch + 40*ms},
			{Change: ui.PointerChangeMove, PhysicalX: 40, TimeStamp: epoch + 50*ms},
			{Change: ui.PointerChangeMove, PhysicalX: 50, TimeStamp: epoch + 60*ms},
			{Change: ui.PointerChangeUp, PhysicalX: 50, TimeStamp: epoch + 70*ms},
			{Change: ui.PointerChangeRemove, PhysicalX: 50, TimeStamp: epoch + 70*ms},
		}}
		const samplingOffset = -5 * time.Millisecond
		const frameInterval = 16667 * time.Microsecond

		GestureBindingInstance().SetResamplingEnabled(true)
		GestureBindingInstance().SetSamplingOffset(samplingOffset)
		var events []PointerEvent
		binding.callback = func(e PointerEvent) { events = append(events, e) }

		GestureBindingInstance().PlatformDispatcher().OnPointerDataPacket()(packet)
		if len(events) != 0 {
			t.Fatalf("%d events", len(events))
		}
		callback := binding.postFrameCallback
		binding.postFrameCallback = nil
		if callback == nil {
			t.Fatal("no frame callback")
		}
		ft := epoch + 15*ms
		binding.frameTime = &ft
		callback(0)

		dpr := GestureBindingInstance().PlatformDispatcher().ImplicitView().DevicePixelRatio()
		expectEvent := func(i int, kind string, ts time.Duration) {
			t.Helper()
			if len(events) <= i {
				t.Fatalf("%d events, want event %d", len(events), i)
			}
			if got := kindsOf(events[i : i+1]); got != kind {
				t.Errorf("event %d is %s, want %s", i, got, kind)
			}
			if events[i].TimeStamp() != ts {
				t.Errorf("event %d timeStamp %v, want %v", i, events[i].TimeStamp(), ts)
			}
		}
		if len(events) != 1 {
			t.Fatalf("%d events", len(events))
		}
		expectEvent(0, "PointerDownEvent", *binding.frameTime+samplingOffset)
		if events[0].Position() != ui.NewOffset(0.0/dpr, 0) {
			t.Error(events[0].Position())
		}

		callback = binding.postFrameCallback
		binding.postFrameCallback = nil
		if callback == nil {
			t.Fatal("no second frame callback")
		}
		frameTime := epoch + 25*ms
		binding.frameTime = &frameTime
		callback(0)
		if len(events) != 2 {
			t.Fatalf("%d events", len(events))
		}
		expectEvent(1, "PointerMoveEvent", frameTime+samplingOffset)
		if events[1].Position() != ui.NewOffset(10.0/dpr, 0) || events[1].Delta() != ui.NewOffset(10.0/dpr, 0) {
			t.Error(events[1].Position(), events[1].Delta())
		}

		// Dart's frameInterval * 1.5 rounds 25000.5 µs to 25001.
		loop.Elapse(25001 * time.Microsecond)
		if len(events) != 3 {
			t.Fatalf("%d events", len(events))
		}
		expectEvent(2, "PointerMoveEvent", frameTime+frameInterval+samplingOffset)

		loop.Elapse(frameInterval)
		if len(events) != 5 {
			t.Fatalf("%d events", len(events))
		}
		expectEvent(3, "PointerMoveEvent", frameTime+2*frameInterval+samplingOffset)
		expectEvent(4, "PointerUpEvent", frameTime+2*frameInterval+samplingOffset)

		loop.Elapse(frameInterval)
		if len(events) != 5 {
			t.Fatalf("%d events", len(events))
		}
		GestureBindingInstance().SetResamplingEnabled(false)
	})
}
