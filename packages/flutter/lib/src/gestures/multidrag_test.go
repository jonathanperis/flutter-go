// Port of packages/flutter/test/gestures/multidrag_test.dart (Flutter
// 3.50.1): all 5 tests. No skips.
package gestures_test

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

type testDrag struct{ DragBase }

func delayedScript(t *testing.T, opts DelayedMultiDragGestureRecognizerOptions, kind ui.PointerDeviceKind,
	firstMove, secondMove ui.Offset, startedAfterDelay bool) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewDelayedMultiDragGestureRecognizer(opts)
		didStartDrag := false
		drag.OnStart = func(ui.Offset) Drag { didStartDrag = true; return &testDrag{} }
		pointer := tp(5, kind)
		down := pointer.Down(off(10, 10))
		drag.AddPointer(down)
		tester.CloseArena(5)
		expectTrue(t, "not started", !didStartDrag)
		tester.Async.FlushMicrotasks()
		expectTrue(t, "not started after microtasks", !didStartDrag)
		tester.Route(pointer.Move(firstMove))
		expectTrue(t, "not started after move", !didStartDrag)
		tester.Async.Elapse(KLongPressTimeout * 2)
		dexpect(t, "started after delay", didStartDrag, startedAfterDelay)
		tester.Route(pointer.Move(secondMove))
		dexpect(t, "started after second move", didStartDrag, startedAfterDelay)
		drag.Dispose()
	})
}

func TestMultiDragMovingBeforeDelayRejects(t *testing.T) {
	delayedScript(t, DelayedMultiDragGestureRecognizerOptions{}, ui.PointerDeviceKindTouch, off(20, 60), off(30, 120), false)
}

func TestMultiDragDelayTriggers(t *testing.T) {
	delayedScript(t, DelayedMultiDragGestureRecognizerOptions{}, ui.PointerDeviceKindTouch, off(20, 20), off(30, 70), true)
}

func TestMultiDragCanFilterBasedOnDeviceKind(t *testing.T) {
	delayedScript(t, DelayedMultiDragGestureRecognizerOptions{GestureRecognizerOptions: GestureRecognizerOptions{
		SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindTouch: {}}}},
		ui.PointerDeviceKindMouse, off(20, 20), off(30, 70), false)
}

func TestMultiDragAllowedButtonsFilterNullOrUnspecified(t *testing.T) {
	recognizer1 := NewImmediateMultiDragGestureRecognizer()
	recognizer2 := NewImmediateMultiDragGestureRecognizer(GestureRecognizerOptions{AllowedButtonsFilter: nil})
	allowed := NewPointerDownEvent(PointerDownEventOptions{TimeStamp: 240 * 3600 * 1e9})
	expectTrue(t, "1 allowed", recognizer1.IsPointerAllowed(allowed))
	expectTrue(t, "2 allowed", recognizer2.IsPointerAllowed(allowed))
	rejected := NewPointerDownEvent(PointerDownEventOptions{TimeStamp: 240 * 3600 * 1e9, Buttons: intp(KMiddleMouseButton)})
	expectTrue(t, "1 rejected", !recognizer1.IsPointerAllowed(rejected))
	expectTrue(t, "2 rejected", !recognizer2.IsPointerAllowed(rejected))
}

// testMultiDragPointerState is the test's _MultiDragPointerState.
type testMultiDragPointerState struct{ MultiDragPointerStateBase }

func (s *testMultiDragPointerState) Accepted(GestureMultiDragStartCallback) {}

func TestMultiDragPointerStateDispatchesMemoryEvents(t *testing.T) {
	var events []foundation.ObjectEvent
	listener := func(event foundation.ObjectEvent) {
		if _, ok := event.Object().(*testMultiDragPointerState); ok {
			events = append(events, event)
		}
	}
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	s := &testMultiDragPointerState{}
	s.Construct(s, ui.Offset{}, ui.PointerDeviceKindTouch, nil)
	s.Dispose()
	if len(events) != 2 {
		t.Fatalf("%d events, want 2", len(events))
	}
	if _, ok := events[0].(*foundation.ObjectCreated); !ok {
		t.Fatalf("first %T", events[0])
	}
	if _, ok := events[1].(*foundation.ObjectDisposed); !ok {
		t.Fatalf("second %T", events[1])
	}
	if events[0].Object() != events[1].Object() {
		t.Fatal("different objects")
	}
}
