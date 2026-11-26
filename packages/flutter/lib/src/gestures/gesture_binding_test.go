package gestures_test

// Port of packages/flutter/test/gestures/gesture_binding_test.dart and
// locking_test.dart (Flutter 3.50.1): all 21 + 1 tests. Each Dart file
// has its own isolate and binding; here each file is one top-level test
// run in a child process (isolated), whose subtests are the Dart tests.
// The platform_views mock handler of the Dart file is not installed (no
// test sends on that channel).

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

const isolatedEnv = "FLUTTER_GO_GESTURES_TEST_ISOLATED"

// isolated re-runs the calling top-level test in its own process (a
// binding per process); it returns true in the child.
func isolated(t *testing.T) bool {
	t.Helper()
	if os.Getenv(isolatedEnv) == t.Name() {
		return true
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), isolatedEnv+"="+t.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s in its own process: %v\n%s", t.Name(), err, out)
	}
	return false
}

// testGestureFlutterBinding is gesture_binding_test.dart's
// TestGestureFlutterBinding: BindingBase with GestureBinding,
// SchedulerBinding, ServicesBinding, TestDefaultBinaryMessengerBinding.
type testGestureFlutterBinding struct {
	foundation.BindingBaseBase
	GestureBindingMixin
	scheduler.SchedulerBindingMixin
	services.ServicesBindingMixin
	flutter_test.TestDefaultBinaryMessengerBindingMixin

	onHandlePointerEvent func(PointerEvent)
	onHandleEvent        func(PointerEvent)
	onHitTestInView      func(result *HitTestResult, position ui.Offset, viewID int)
}

func newTestGestureFlutterBinding() *testGestureFlutterBinding {
	loop := async.Current()
	b := &testGestureFlutterBinding{}
	b.GestureBindingMixin.SetSelf(b)
	b.SchedulerBindingMixin.SetSelf(b)
	b.ServicesBindingMixin.SetSelf(b)
	b.TestDefaultBinaryMessengerBindingMixin.SetSelf(b)
	b.Construct(b, &b.GestureBindingMixin, &b.SchedulerBindingMixin, &b.ServicesBindingMixin,
		&b.TestDefaultBinaryMessengerBindingMixin)
	loop.FlushMicrotasks()
	loop.RunPosted()
	return b
}

func (b *testGestureFlutterBinding) InitInstances() {
	b.TestDefaultBinaryMessengerBindingMixin.InitInstances()
}
func (b *testGestureFlutterBinding) InitServiceExtensions() {
	b.ServicesBindingMixin.InitServiceExtensions()
}
func (b *testGestureFlutterBinding) Unlocked() { b.SchedulerBindingMixin.Unlocked() }
func (b *testGestureFlutterBinding) CreateBinaryMessenger() services.BinaryMessenger {
	return b.TestDefaultBinaryMessengerBindingMixin.CreateBinaryMessenger()
}

func (b *testGestureFlutterBinding) HandlePointerEvent(event PointerEvent) {
	if b.onHandlePointerEvent != nil {
		b.onHandlePointerEvent(event)
	}
	b.GestureBindingMixin.HandlePointerEvent(event)
}

func (b *testGestureFlutterBinding) HandleEvent(event PointerEvent, entry HitTestEntry) {
	b.GestureBindingMixin.HandleEvent(event, entry)
	if b.onHandleEvent != nil {
		b.onHandleEvent(event)
	}
}

func (b *testGestureFlutterBinding) HitTestInView(result *HitTestResult, position ui.Offset, viewID int) {
	if b.onHitTestInView != nil {
		b.onHitTestInView(result, position, viewID)
		return
	}
	b.GestureBindingMixin.HitTestInView(result, position, viewID)
}

type dummyNativeHitTestTarget struct{ NativeHitTestTargetMixin }

func (*dummyNativeHitTestTarget) HandleEvent(PointerEvent, HitTestEntry) {}

func kindsOf(events []PointerEvent) string {
	var s []string
	for _, e := range events {
		s = append(s, strings.TrimPrefix(fmt.Sprintf("%T", e), "*gestures.")) // Dart's isA (transformed events too)
	}
	return strings.Join(s, ",")
}

func expectKinds(t *testing.T, events []PointerEvent, want string) {
	t.Helper()
	if got := kindsOf(events); got != want {
		t.Errorf("events %s, want %s", got, want)
	}
}

func pdp(data ...ui.PointerData) ui.PointerDataPacket { return ui.PointerDataPacket{Data: data} }

func change(c ui.PointerChange) ui.PointerData { return ui.PointerData{Change: c} }

func expand(data []ui.PointerData, dpr func(int) *float64) []PointerEvent {
	return slices.Collect(PointerEventConverter.Expand(slices.Values(data), dpr))
}

func TestGestureBindingFile(t *testing.T) {
	if !isolated(t) {
		return
	}
	defer async.SetCurrent(async.NewLoop())()
	binding := newTestGestureFlutterBinding()
	pd := GestureBindingInstance().PlatformDispatcher()
	view := pd.ImplicitView()
	hitTest := func() ui.HitTestResponse {
		if cb := pd.OnHitTest(); cb != nil {
			return cb(ui.HitTestRequest{View: view, Offset: ui.NewOffset(1, 1)})
		}
		return ui.HitTestResponseEmpty
	}
	run := func(name string, f func(t *testing.T)) {
		t.Run(name, func(t *testing.T) {
			defer func() { binding.onHitTestInView = nil }()
			f(t)
		})
	}

	run("Platform view hit test should not accept gesture if no hit", func(t *testing.T) {
		binding.onHitTestInView = func(*HitTestResult, ui.Offset, int) {}
		if hitTest().HasPlatformView {
			t.Error("hasPlatformView")
		}
	})
	run("Platform view hit test should not accept gesture if no platform view", func(t *testing.T) {
		binding.onHitTestInView = func(r *HitTestResult, _ ui.Offset, _ int) { r.Add(NewHitTestEntry(&dummyHitTestTarget{})) }
		if hitTest().HasPlatformView {
			t.Error("hasPlatformView")
		}
	})
	run("Platform view hit test should accept gesture if a platform view is not the first hit", func(t *testing.T) {
		binding.onHitTestInView = func(r *HitTestResult, _ ui.Offset, _ int) {
			r.Add(NewHitTestEntry(&dummyHitTestTarget{}))
			r.Add(NewHitTestEntry(&dummyNativeHitTestTarget{}))
		}
		if !hitTest().HasPlatformView {
			t.Error("!hasPlatformView")
		}
	})
	run("Platform view hit test should accept gesture if a platform view is the first hit", func(t *testing.T) {
		binding.onHitTestInView = func(r *HitTestResult, _ ui.Offset, _ int) {
			r.Add(NewHitTestEntry(&dummyNativeHitTestTarget{}))
			r.Add(NewHitTestEntry(&dummyHitTestTarget{}))
		}
		if !hitTest().HasPlatformView {
			t.Error("!hasPlatformView")
		}
	})

	dispatch := func(packet ui.PointerDataPacket) []PointerEvent {
		var events []PointerEvent
		binding.onHandleEvent = func(e PointerEvent) { events = append(events, e) }
		pd.OnPointerDataPacket()(packet)
		return events
	}
	run("Pointer tap events", func(t *testing.T) {
		expectKinds(t, dispatch(pdp(change(ui.PointerChangeDown), change(ui.PointerChangeUp))), "PointerDownEvent,PointerUpEvent")
	})
	run("Pointer move events", func(t *testing.T) {
		expectKinds(t, dispatch(pdp(change(ui.PointerChangeDown), change(ui.PointerChangeMove), change(ui.PointerChangeUp))),
			"PointerDownEvent,PointerMoveEvent,PointerUpEvent")
	})
	run("Pointer hover events", func(t *testing.T) {
		var routed []PointerEvent
		route := func(e PointerEvent) { routed = append(routed, e) }
		GestureBindingInstance().PointerRouter().AddGlobalRoute(route)
		defer GestureBindingInstance().PointerRouter().RemoveGlobalRoute(route)
		events := dispatch(pdp(change(ui.PointerChangeAdd), change(ui.PointerChangeHover), change(ui.PointerChangeHover),
			change(ui.PointerChangeRemove), change(ui.PointerChangeAdd), change(ui.PointerChangeHover)))
		expectKinds(t, events, "PointerHoverEvent,PointerHoverEvent,PointerHoverEvent")
		expectKinds(t, routed, "PointerAddedEvent,PointerHoverEvent,PointerHoverEvent,PointerRemovedEvent,PointerAddedEvent,PointerHoverEvent")
	})
	run("Pointer cancel events", func(t *testing.T) {
		expectKinds(t, dispatch(pdp(change(ui.PointerChangeDown), ui.PointerData{})), "PointerDownEvent,PointerCancelEvent")
	})
	run("Can cancel pointers", func(t *testing.T) {
		var events []PointerEvent
		binding.onHandleEvent = func(e PointerEvent) {
			events = append(events, e)
			if _, ok := e.(*PointerDownEvent); ok {
				binding.CancelPointer(e.Pointer())
			}
		}
		pd.OnPointerDataPacket()(pdp(change(ui.PointerChangeDown), change(ui.PointerChangeUp)))
		expectKinds(t, events, "PointerDownEvent,PointerCancelEvent")
	})

	const devicePixelRatio = 2.5
	dpr := func(int) *float64 { v := devicePixelRatio; return &v }
	run("Can expand add and hover pointers", func(t *testing.T) {
		d := func(c ui.PointerChange) ui.PointerData { return ui.PointerData{Change: c, Device: 24} }
		expectKinds(t, expand([]ui.PointerData{d(ui.PointerChangeAdd), d(ui.PointerChangeHover), d(ui.PointerChangeRemove),
			d(ui.PointerChangeAdd), d(ui.PointerChangeHover)}, dpr),
			"PointerAddedEvent,PointerHoverEvent,PointerRemovedEvent,PointerAddedEvent,PointerHoverEvent")
	})
	run("Can handle malformed scrolling event.", func(t *testing.T) {
		expectKinds(t, expand([]ui.PointerData{{Change: ui.PointerChangeAdd, Device: 24}}, dpr), "PointerAddedEvent")
		scroll := ui.PointerSignalKindScroll
		s := func(x, y float64) ui.PointerData {
			return ui.PointerData{SignalKind: &scroll, Device: 24, ScrollDeltaX: x, ScrollDeltaY: y}
		}
		inf, nan := math.Inf(1), math.NaN()
		if n := len(expand([]ui.PointerData{s(inf, 10), s(nan, 10), s(-inf, 10), s(10, inf), s(10, nan), s(10, -inf)}, dpr)); n != 0 {
			t.Errorf("%d events", n)
		}
		expectKinds(t, expand([]ui.PointerData{s(10, 10)}, dpr), "PointerScrollEvent")
		zero := func(int) *float64 { v := 0.0; return &v }
		if n := len(expand([]ui.PointerData{s(10, 10)}, zero)); n != 0 {
			t.Errorf("%d events with dpr 0", n)
		}
	})
	run("Can expand pointer scroll events", func(t *testing.T) {
		scroll := ui.PointerSignalKindScroll
		expectKinds(t, expand([]ui.PointerData{{Change: ui.PointerChangeAdd}, {Change: ui.PointerChangeHover, SignalKind: &scroll}}, dpr),
			"PointerAddedEvent,PointerScrollEvent")
	})

	loc := ui.NewOffset(10, 10).Mul(devicePixelRatio)
	seq := func(kind ui.PointerDeviceKind, downButtons, moveButtons int) []ui.PointerData {
		p := func(c ui.PointerChange, buttons int) ui.PointerData {
			return ui.PointerData{Change: c, Kind: kind, Buttons: buttons, PhysicalX: loc.Dx, PhysicalY: loc.Dy}
		}
		return []ui.PointerData{p(ui.PointerChangeAdd, 0), p(ui.PointerChangeHover, 0), p(ui.PointerChangeDown, downButtons),
			p(ui.PointerChangeMove, moveButtons), p(ui.PointerChangeUp, 0)}
	}
	expectButtons := func(t *testing.T, events []PointerEvent, down, move int) {
		t.Helper()
		expectKinds(t, events, "PointerAddedEvent,PointerHoverEvent,PointerDownEvent,PointerMoveEvent,PointerUpEvent")
		want := []int{0, 0, down, move, 0}
		for i, e := range events {
			if e.Buttons() != want[i] {
				t.Errorf("event %d buttons %d, want %d", i, e.Buttons(), want[i])
			}
		}
	}
	run("Should synthesize kPrimaryButton for touch when no button is set", func(t *testing.T) {
		expectButtons(t, expand(seq(ui.PointerDeviceKindTouch, 0, 0), dpr), KPrimaryButton, KPrimaryButton)
	})
	run("Should not synthesize kPrimaryButton for touch when a button is set", func(t *testing.T) {
		expectButtons(t, expand(seq(ui.PointerDeviceKindTouch, KSecondaryButton, KSecondaryButton), dpr), KSecondaryButton, KSecondaryButton)
	})
	run("Should synthesize kPrimaryButton for stylus when no button is set", func(t *testing.T) {
		for _, kind := range []ui.PointerDeviceKind{ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus} {
			expectButtons(t, expand(seq(kind, 0, KSecondaryStylusButton), dpr), KPrimaryButton, KSecondaryStylusButton)
		}
	})
	run("Should synthesize kPrimaryButton for unknown devices when no button is set", func(t *testing.T) {
		expectButtons(t, expand(seq(ui.PointerDeviceKindUnknown, 0, KSecondaryButton), dpr), KPrimaryButton, KSecondaryButton)
	})
	run("Should not synthesize kPrimaryButton for mouse", func(t *testing.T) {
		expectButtons(t, expand(seq(ui.PointerDeviceKindMouse, KMiddleMouseButton, KMiddleMouseButton|KSecondaryMouseButton), dpr),
			KMiddleMouseButton, KMiddleMouseButton|KSecondaryMouseButton)
	})
	run("Pointer pan/zoom events", func(t *testing.T) {
		expectKinds(t, dispatch(pdp(change(ui.PointerChangePanZoomStart), change(ui.PointerChangePanZoomUpdate), change(ui.PointerChangePanZoomEnd))),
			"PointerPanZoomStartEvent,PointerPanZoomUpdateEvent,PointerPanZoomEndEvent")
	})
	run("Error handling", func(t *testing.T) {
		var events []string
		binding.onHandlePointerEvent = func(event PointerEvent) {
			panic(dart.NewException("zipzapzooey " + event.String()))
		}
		previous := foundation.FlutterErrorOnError
		foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) { events = append(events, details.String()) }
		defer func() {
			binding.onHandlePointerEvent = nil
			foundation.FlutterErrorOnError = previous
		}()
		pd.OnPointerDataPacket()(pdp(change(ui.PointerChangeDown), change(ui.PointerChangeUp)))
		if len(events) != 1 {
			t.Fatalf("%d errors", len(events))
		}
		for _, s := range []string{"while handling a pointer data\npacket", "zipzapzooey", "PointerDownEvent"} {
			if !strings.Contains(events[0], s) {
				t.Errorf("missing %q in %s", s, events[0])
			}
		}
		if strings.Contains(events[0], "PointerUpEvent") {
			t.Errorf("PointerUpEvent processed: %s", events[0])
		}
	})
	run("PointerEventConverter processes view IDs", func(t *testing.T) {
		const startID = 987654
		changes := []ui.PointerChange{ui.PointerChangeCancel, ui.PointerChangeAdd, ui.PointerChangeRemove, ui.PointerChangeHover,
			ui.PointerChangeDown, ui.PointerChangeMove, ui.PointerChangeUp, ui.PointerChangePanZoomStart,
			ui.PointerChangePanZoomUpdate, ui.PointerChangePanZoomEnd}
		var data []ui.PointerData
		for i, c := range changes {
			data = append(data, ui.PointerData{ViewID: startID + i, Change: c})
		}
		var viewIDs []int
		events := expand(data, func(viewID int) *float64 {
			viewIDs = append(viewIDs, viewID)
			v := float64(viewID) / 10.0
			return &v
		})
		if len(viewIDs) != 10 || len(events) != 10 {
			t.Fatalf("%d view ids, %d events", len(viewIDs), len(events))
		}
		for i := range 10 {
			if viewIDs[i] != startID+i || events[i].ViewID() != startID+i {
				t.Errorf("%d: %d %d", i, viewIDs[i], events[i].ViewID())
			}
		}
	})
}

// lockingBinding is locking_test.dart's TestGestureFlutterBinding
// (BindingBase with GestureBinding).
type lockingBinding struct {
	foundation.BindingBaseBase
	GestureBindingMixin
	callback func(PointerEvent)
}

func (b *lockingBinding) InitInstances() { b.GestureBindingMixin.InitInstances() }
func (b *lockingBinding) Unlocked()      { b.GestureBindingMixin.Unlocked() }
func (b *lockingBinding) HandleEvent(event PointerEvent, entry HitTestEntry) {
	if b.callback != nil {
		b.callback(event)
	}
	b.GestureBindingMixin.HandleEvent(event, entry)
}

func TestLockingFile(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	b := &lockingBinding{}
	b.GestureBindingMixin.SetSelf(b)
	b.Construct(b, &b.GestureBindingMixin)

	t.Run("Pointer events are locked during reassemble", func(t *testing.T) {
		var events []PointerEvent
		b.callback = func(e PointerEvent) { events = append(events, e) }
		tested := false
		done := false
		async.OnComplete(b.LockEvents(func() async.Future[async.Void] {
			return async.AsyncVoid(func() {
				GestureBindingInstance().PlatformDispatcher().OnPointerDataPacket()(
					pdp(change(ui.PointerChangeDown), change(ui.PointerChangeUp)))
				if len(events) != 0 {
					t.Errorf("%d events while locked", len(events))
				}
				tested = true
			})
		}), func(async.Void) { done = true }, nil)
		loop.FlushMicrotasks()
		if !done || !tested {
			t.Fatal("not run")
		}
		expectKinds(t, events, "PointerDownEvent,PointerUpEvent")
	})
}
