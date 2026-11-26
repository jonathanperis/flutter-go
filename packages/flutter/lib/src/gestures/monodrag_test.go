// Port of packages/flutter/test/gestures/monodrag_test.dart (Flutter
// 3.50.1): 5 of 6 tests.
//
// Skipped: "DragGestureRecognizer can be subclassed to beat a
// CustomScrollView in the arena" needs widgets (CustomScrollView,
// RawGestureDetector not ported); its subclassing is checked by
// TestMonodragSubclassOverridesHasSufficientDistance below.
package gestures_test

import (
	"math"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func TestMonodragAcceptGestureToleratesNullLastPendingEventTimestamp(t *testing.T) {
	flutter_test.New(t)
	var recognizer DragGestureRecognizer = NewVerticalDragGestureRecognizer()
	r := recognizer.(*VerticalDragGestureRecognizer)
	event := NewPointerDownEvent(PointerDownEventOptions{TimeStamp: 10 * 24 * time.Hour})

	if r.DebugLastPendingEventTimestamp() != nil {
		t.Fatal("not null")
	}
	recognizer.AddAllowedPointer(event)
	if ts := r.DebugLastPendingEventTimestamp(); ts == nil || *ts != event.TimeStamp() {
		t.Fatalf("timestamp %v", ts)
	}
	// Normal case: acceptGesture called and we have a last timestamp set.
	recognizer.AcceptGesture(event.Pointer())
	if r.DebugLastPendingEventTimestamp() != nil {
		t.Fatal("not null after accept")
	}
	// Reject the gesture to reset state and allow accepting it again.
	recognizer.RejectGesture(event.Pointer())
	if r.DebugLastPendingEventTimestamp() != nil {
		t.Fatal("not null after reject")
	}
	recognizer.AcceptGesture(event.Pointer())
	if r.DebugLastPendingEventTimestamp() != nil {
		t.Fatal("not null after second accept")
	}
	recognizer.Dispose()
}

func TestMonodragUpForPendingPointerAfterWinningArenaForAnother(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		v := NewVerticalDragGestureRecognizer()
		v.OnStart = func(*DragStartDetails) {}
		t.Cleanup(v.Dispose)
		h := NewHorizontalDragGestureRecognizer()
		h.OnStart = func(*DragStartDetails) {}
		t.Cleanup(h.Dispose)

		down90 := NewPointerDownEvent(PointerDownEventOptions{Pointer: 90, Position: off(10, 10)})
		up90 := NewPointerUpEvent(PointerUpEventOptions{Pointer: 90, Position: off(10, 10)})
		down91 := NewPointerDownEvent(PointerDownEventOptions{Pointer: 91, Position: off(20, 20)})
		up91 := NewPointerUpEvent(PointerUpEventOptions{Pointer: 91, Position: off(20, 20)})

		v.AddPointer(down90)
		GestureBindingInstance().GestureArena().Close(90)
		h.AddPointer(down91)
		v.AddPointer(down91)
		GestureBindingInstance().GestureArena().Close(91)
		tester.Async.FlushMicrotasks()

		GestureBindingInstance().HandleEvent(up90, NewHitTestEntry(mockHitTestTarget{}))
		GestureBindingInstance().HandleEvent(up91, NewHitTestEntry(mockHitTestTarget{}))
	})
}

func onlyAcceptDragOnThresholdScript(t *testing.T, only bool) []string {
	var dragCallbacks []string
	testGesture(t, func(tester *flutter_test.GestureTester) {
		verticalDrag := NewVerticalDragGestureRecognizer()
		verticalDrag.OnlyAcceptDragOnThreshold = only
		verticalDrag.OnStart = func(*DragStartDetails) { dragCallbacks = append(dragCallbacks, "onStart") }
		verticalDrag.OnUpdate = func(*DragUpdateDetails) { dragCallbacks = append(dragCallbacks, "onUpdate") }
		verticalDrag.OnEnd = func(*DragEndDetails) { dragCallbacks = append(dragCallbacks, "onEnd") }

		down1 := NewPointerDownEvent(PointerDownEventOptions{Pointer: 6, Position: off(10, 10)})
		up1 := NewPointerUpEvent(PointerUpEventOptions{Pointer: 6, Position: off(10, 10)})
		verticalDrag.AddPointer(down1)
		tester.CloseArena(down1.Pointer())
		tester.Route(down1)
		tester.Route(up1)
		verticalDrag.Dispose()
	})
	return dragCallbacks
}

func TestMonodragOnlyAcceptDragOnThresholdTrue(t *testing.T) {
	if got := onlyAcceptDragOnThresholdScript(t, true); len(got) != 0 {
		t.Errorf("callbacks %v, want none", got)
	}
}

func TestMonodragOnlyAcceptDragOnThresholdFalse(t *testing.T) {
	dexpect(t, "callbacks", onlyAcceptDragOnThresholdScript(t, false), []string{"onStart", "onEnd"})
}

func TestMonodragRecognizersOnDifferentButtonFilters(t *testing.T) {
	setUp := func() (recognized *[]string, primary, secondary *HorizontalDragGestureRecognizer) {
		rec := []string{}
		recognized = &rec
		primary = NewHorizontalDragGestureRecognizer(GestureRecognizerOptions{
			AllowedButtonsFilter: func(buttons int) bool { return KPrimaryButton == buttons }})
		primary.OnStart = func(*DragStartDetails) { *recognized = append(*recognized, "onStartPrimary") }
		secondary = NewHorizontalDragGestureRecognizer(GestureRecognizerOptions{
			AllowedButtonsFilter: func(buttons int) bool { return KSecondaryButton == buttons }})
		secondary.OnStart = func(*DragStartDetails) { *recognized = append(*recognized, "onStartSecondary") }
		return
	}
	t.Run("Primary button works", func(t *testing.T) {
		recognized, primary, secondary := setUp()
		testGesture(t, func(tester *flutter_test.GestureTester) {
			down1 := NewPointerDownEvent(PointerDownEventOptions{Pointer: 6, Position: off(10, 10)})
			primary.AddPointer(down1)
			secondary.AddPointer(down1)
			tester.CloseArena(down1.Pointer())
			tester.Route(down1)
			dexpect(t, "recognized", *recognized, []string{"onStartPrimary"})
			primary.Dispose()
			secondary.Dispose()
		})
	})
	t.Run("Secondary button works", func(t *testing.T) {
		recognized, primary, secondary := setUp()
		testGesture(t, func(tester *flutter_test.GestureTester) {
			down1 := NewPointerDownEvent(PointerDownEventOptions{Pointer: 6, Position: off(10, 10),
				Buttons: intp(KSecondaryMouseButton)})
			primary.AddPointer(down1)
			secondary.AddPointer(down1)
			tester.CloseArena(down1.Pointer())
			tester.Route(down1)
			dexpect(t, "recognized", *recognized, []string{"onStartSecondary"})
			primary.Dispose()
			secondary.Dispose()
		})
	})
}

// eagerPanGestureRecognizer is the skipped test's _EagerPanGestureRecognizer:
// a PanGestureRecognizer subclass (from another package) overriding the
// acceptance threshold and the fling decision.
type eagerPanGestureRecognizer struct{ PanGestureRecognizer }

func newEagerPanGestureRecognizer() *eagerPanGestureRecognizer {
	r := &eagerPanGestureRecognizer{}
	r.Construct(r, DragGestureRecognizerOptions{})
	return r
}

func (r *eagerPanGestureRecognizer) HasSufficientGlobalDistanceToAccept(kind ui.PointerDeviceKind, deviceTouchSlop *float64) bool {
	return math.Abs(r.GlobalDistanceMoved()) > ComputeHitSlop(kind, r.GestureSettings)
}

func TestMonodragSubclassOverridesHasSufficientDistance(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		eager := newEagerPanGestureRecognizer()
		pan := NewPanGestureRecognizer()
		var log []string
		eager.OnStart = func(*DragStartDetails) { log = append(log, "eager start") }
		pan.OnStart = func(*DragStartDetails) { log = append(log, "pan start") }
		p := tp(5, ui.PointerDeviceKindTouch)
		down := p.Down(off(10, 10))
		pan.AddPointer(down)
		eager.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		// More than the hit slop (18), less than the pan slop (36).
		tester.Route(p.Move(off(10, 10+KTouchSlop+1)))
		dexpect(t, "log", log, []string{"eager start"})
		tester.Route(p.Up())
		pan.Dispose()
		eager.Dispose()
	})
}
