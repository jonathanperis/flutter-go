package gestures_test

// Port of packages/flutter/test/gestures/recognizer_test.dart (Flutter
// 3.50.1): all 7 tests. "dispatches memory events" listens to
// FlutterMemoryAllocations directly (leak_tracker's memoryEvents is not
// ported); hasAGoodToStringDeep is checked on the text it requires.

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

var (
	recDown  = NewPointerDownEvent(PointerDownEventOptions{Pointer: 5, Position: ui.NewOffset(10, 10)})
	recMove  = NewPointerMoveEvent(PointerMoveEventOptions{Pointer: 5, Position: ui.NewOffset(15, 15)})
	recUp    = NewPointerUpEvent(PointerUpEventOptions{Pointer: 5, Position: ui.NewOffset(15, 15)})
	recDown2 = NewPointerDownEvent(PointerDownEventOptions{Pointer: 6, Position: ui.NewOffset(10, 10)})
	recMove2 = NewPointerMoveEvent(PointerMoveEventOptions{Pointer: 6, Position: ui.NewOffset(100, 200)})
	recUp2   = NewPointerUpEvent(PointerUpEventOptions{Pointer: 6, Position: ui.NewOffset(100, 200)})
)

// TestGestureRecognizer is the Dart test's TestGestureRecognizer.
type TestGestureRecognizer struct{ GestureRecognizerBase }

func newTestGestureRecognizer(debugOwner any) *TestGestureRecognizer {
	r := &TestGestureRecognizer{}
	r.Construct(r, GestureRecognizerOptions{DebugOwner: debugOwner})
	return r
}

func (r *TestGestureRecognizer) DebugDescription() string     { return "debugDescription content" }
func (r *TestGestureRecognizer) AddPointer(*PointerDownEvent) {}
func (r *TestGestureRecognizer) AcceptGesture(int)            {}
func (r *TestGestureRecognizer) RejectGesture(int)            {}

// indefiniteGestureRecognizer adds itself to the arena but never resolves.
type indefiniteGestureRecognizer struct{ GestureRecognizerBase }

func newIndefiniteGestureRecognizer() *indefiniteGestureRecognizer {
	r := &indefiniteGestureRecognizer{}
	r.Construct(r, GestureRecognizerOptions{})
	return r
}

func (r *indefiniteGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	GestureBindingInstance().GestureArena().Add(event.Pointer(), r)
}
func (r *indefiniteGestureRecognizer) AcceptGesture(int)        {}
func (r *indefiniteGestureRecognizer) RejectGesture(int)        {}
func (r *indefiniteGestureRecognizer) DebugDescription() string { return "Unresolving" }

// testPrimaryPointerGestureRecognizer is TestPrimaryPointerGestureRecognizer<T>:
// it resolves with resolution on an event of the primary pointer matching
// isT.
type testPrimaryPointerGestureRecognizer struct {
	PrimaryPointerGestureRecognizerBase
	resolution      GestureDisposition
	isT             func(PointerEvent) bool
	onAcceptGesture func()
	onRejectGesture func()
}

func newTestPrimaryPointer(isT func(PointerEvent) bool, resolution GestureDisposition, opts PrimaryPointerGestureRecognizerOptions,
	onAccept, onReject func()) *testPrimaryPointerGestureRecognizer {
	r := &testPrimaryPointerGestureRecognizer{resolution: resolution, isT: isT, onAcceptGesture: onAccept, onRejectGesture: onReject}
	r.Construct(r, opts)
	return r
}

func (r *testPrimaryPointerGestureRecognizer) AcceptGesture(pointer int) {
	r.PrimaryPointerGestureRecognizerBase.AcceptGesture(pointer)
	if r.onAcceptGesture != nil {
		r.onAcceptGesture()
	}
}

func (r *testPrimaryPointerGestureRecognizer) RejectGesture(pointer int) {
	r.PrimaryPointerGestureRecognizerBase.RejectGesture(pointer)
	if r.onRejectGesture != nil {
		r.onRejectGesture()
	}
}

func (r *testPrimaryPointerGestureRecognizer) HandlePrimaryPointer(event PointerEvent) {
	if r.isT(event) {
		r.Resolve(r.resolution)
	}
}

func (r *testPrimaryPointerGestureRecognizer) DebugDescription() string { return "TestPrimaryPointer" }

func isUp(e PointerEvent) bool   { _, ok := e.(*PointerUpEvent); return ok }
func isMove(e PointerEvent) bool { _, ok := e.(*PointerMoveEvent); return ok }

func expectStrings(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGestureRecognizerSmoketest(t *testing.T) {
	flutter_test.New(t)
	recognizer := newTestGestureRecognizer(0)
	deep := recognizer.ToStringDeep()
	if !strings.HasPrefix(deep, "TestGestureRecognizer#") || !strings.Contains(deep, "debugOwner: 0") || !strings.HasSuffix(deep, "\n") {
		t.Errorf("toStringDeep %q", deep)
	}
}

func TestGestureRecognizerDispatchesMemoryEvents(t *testing.T) {
	var events []foundation.ObjectEvent
	listener := func(e foundation.ObjectEvent) {
		if _, ok := e.Object().(*TestGestureRecognizer); ok {
			events = append(events, e)
		}
	}
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	newTestGestureRecognizer(nil).Dispose()
	if len(events) != 2 {
		t.Fatalf("%d events", len(events))
	}
	created, ok := events[0].(*foundation.ObjectCreated)
	_, disposed := events[1].(*foundation.ObjectDisposed)
	if !ok || !disposed || created.ClassName != "GestureRecognizer" || created.Library != "package:flutter/gestures.dart" {
		t.Errorf("events %#v", events)
	}
}

func TestOffsetPair(t *testing.T) {
	offset1 := OffsetPair{Local: ui.NewOffset(10, 20), Global: ui.NewOffset(30, 40)}
	if offset1.Local != ui.NewOffset(10, 20) || offset1.Global != ui.NewOffset(30, 40) {
		t.Error(offset1)
	}
	offset2 := OffsetPair{Local: ui.NewOffset(50, 60), Global: ui.NewOffset(70, 80)}
	sum := offset2.Add(offset1)
	if sum.Local != ui.NewOffset(60, 80) || sum.Global != ui.NewOffset(100, 120) {
		t.Error(sum)
	}
	difference := offset2.Sub(offset1)
	if difference.Local != ui.NewOffset(40, 40) || difference.Global != ui.NewOffset(40, 40) {
		t.Error(difference)
	}
	if s := offset1.String(); s != "OffsetPair(local: Offset(10.0, 20.0), global: Offset(30.0, 40.0))" {
		t.Error(s)
	}
}

func expectPrimary(t *testing.T, r *testPrimaryPointerGestureRecognizer, state GestureRecognizerState, pointer *int, initial *OffsetPair) {
	t.Helper()
	if r.State() != state {
		t.Errorf("state %v, want %v", r.State(), state)
	}
	if (r.PrimaryPointer() == nil) != (pointer == nil) || (pointer != nil && *r.PrimaryPointer() != *pointer) {
		t.Errorf("primaryPointer %v, want %v", r.PrimaryPointer(), pointer)
	}
	if (r.InitialPosition() == nil) != (initial == nil) || (initial != nil && *r.InitialPosition() != *initial) {
		t.Errorf("initialPosition %v, want %v", r.InitialPosition(), initial)
	}
}

func TestPrimaryPointerCleansUpStateAfterWinningArena(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		var resolutions []string
		indefinite := newIndefiniteGestureRecognizer()
		defer indefinite.Dispose()
		accepting := newTestPrimaryPointer(isUp, GestureDispositionAccepted, PrimaryPointerGestureRecognizerOptions{},
			func() { resolutions = append(resolutions, "accepted") }, func() { resolutions = append(resolutions, "rejected") })
		defer accepting.Dispose()
		expectPrimary(t, accepting, GestureRecognizerStateReady, nil, nil)
		expectStrings(t, resolutions)

		indefinite.AddPointer(recDown)
		accepting.AddPointer(recDown)
		five := 5
		initial := OffsetPair{Global: recDown.Position(), Local: recDown.LocalPosition()}
		expectPrimary(t, accepting, GestureRecognizerStatePossible, &five, &initial)
		expectStrings(t, resolutions)

		tester.CloseArena(5)
		tester.Async.FlushMicrotasks()
		tester.Route(recDown)
		tester.Route(recUp)
		expectPrimary(t, accepting, GestureRecognizerStateReady, &five, nil)
		expectStrings(t, resolutions, "accepted")
	})
}

func TestPrimaryPointerCleansUpStateAfterLosingArena(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		var resolutions []string
		indefinite := newIndefiniteGestureRecognizer()
		defer indefinite.Dispose()
		rejecting := newTestPrimaryPointer(isMove, GestureDispositionRejected, PrimaryPointerGestureRecognizerOptions{},
			func() { resolutions = append(resolutions, "accepted") }, func() { resolutions = append(resolutions, "rejected") })
		defer rejecting.Dispose()
		expectPrimary(t, rejecting, GestureRecognizerStateReady, nil, nil)

		indefinite.AddPointer(recDown)
		rejecting.AddPointer(recDown)
		five := 5
		initial := OffsetPair{Global: recDown.Position(), Local: recDown.LocalPosition()}
		expectPrimary(t, rejecting, GestureRecognizerStatePossible, &five, &initial)
		expectStrings(t, resolutions)

		tester.CloseArena(5)
		tester.Async.FlushMicrotasks()
		tester.Route(recDown)
		tester.Route(recMove)
		expectPrimary(t, rejecting, GestureRecognizerStateDefunct, &five, &initial)
		expectStrings(t, resolutions, "rejected")

		tester.Route(recUp)
		expectPrimary(t, rejecting, GestureRecognizerStateReady, &five, nil)
		expectStrings(t, resolutions, "rejected")
	})
}

func TestPrimaryPointerWorksProperlyWhenRecycled(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		var resolutions []string
		indefinite := newIndefiniteGestureRecognizer()
		defer indefinite.Dispose()
		accepting := newTestPrimaryPointer(isUp, GestureDispositionAccepted, PrimaryPointerGestureRecognizerOptions{
			PreAcceptSlopTolerance: f64(15), PostAcceptSlopTolerance: f64(1000)},
			func() { resolutions = append(resolutions, "accepted") }, func() { resolutions = append(resolutions, "rejected") })
		defer accepting.Dispose()

		indefinite.AddPointer(recDown)
		accepting.AddPointer(recDown)
		tester.CloseArena(5)
		tester.Async.FlushMicrotasks()
		tester.Route(recDown)
		tester.Route(recUp)
		expectStrings(t, resolutions, "accepted")
		resolutions = nil

		indefinite.AddPointer(recDown2)
		accepting.AddPointer(recDown2)
		tester.CloseArena(6)
		tester.Async.FlushMicrotasks()
		tester.Route(recDown2)
		tester.Route(recMove2)
		expectStrings(t, resolutions, "rejected")
		tester.Route(recUp2)
		expectStrings(t, resolutions, "rejected")
	})
}

func TestPrimaryPointerUsesExpectedPreAcceptSlopTolerance(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		var resolutions []string
		accept := func() { resolutions = append(resolutions, "accepted") }
		reject := func() { resolutions = append(resolutions, "rejected") }
		indefinite := newIndefiniteGestureRecognizer()
		defer indefinite.Dispose()
		defaultSlop := newTestPrimaryPointer(isUp, GestureDispositionAccepted,
			PrimaryPointerGestureRecognizerOptions{NoPostAcceptSlopTolerance: true}, accept, reject)
		defer defaultSlop.Dispose()
		setSlop := newTestPrimaryPointer(isUp, GestureDispositionAccepted,
			PrimaryPointerGestureRecognizerOptions{PreAcceptSlopTolerance: f64(5), NoPostAcceptSlopTolerance: true}, accept, reject)
		defer setSlop.Dispose()
		nullSlop := newTestPrimaryPointer(isUp, GestureDispositionAccepted,
			PrimaryPointerGestureRecognizerOptions{NoPreAcceptSlopTolerance: true, NoPostAcceptSlopTolerance: true}, accept, reject)
		defer nullSlop.Dispose()

		if v := defaultSlop.PreAcceptSlopTolerance(); v == nil || *v != KTouchSlop {
			t.Errorf("default slop %v", v)
		}
		if v := setSlop.PreAcceptSlopTolerance(); v == nil || *v != 5 {
			t.Errorf("set slop %v", v)
		}
		if v := nullSlop.PreAcceptSlopTolerance(); v != nil {
			t.Errorf("null slop %v", *v)
		}

		run := func(r GestureRecognizer, down *PointerDownEvent, move *PointerMoveEvent, up *PointerUpEvent, pointer int) {
			indefinite.AddPointer(down)
			r.AddPointer(down)
			tester.CloseArena(pointer)
			tester.Async.FlushMicrotasks()
			tester.Route(down)
			tester.Route(move)
			tester.Route(up)
		}
		run(defaultSlop, recDown, recMove, recUp, 5)
		expectStrings(t, resolutions, "accepted")
		resolutions = nil

		defaultSlop.GestureSettings = &DeviceGestureSettings{TouchSlop: f64(5)}
		run(defaultSlop, recDown, recMove, recUp, 5)
		expectStrings(t, resolutions, "rejected")
		resolutions = nil

		run(setSlop, recDown, recMove, recUp, 5)
		expectStrings(t, resolutions, "rejected")
		resolutions = nil

		run(nullSlop, recDown2, recMove2, recUp2, 6)
		expectStrings(t, resolutions, "accepted")
	})
}
