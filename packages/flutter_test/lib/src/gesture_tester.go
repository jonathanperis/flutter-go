package flutter_test

// Port of packages/flutter/test/gestures/gesture_tester.dart (Flutter
// 3.50.1), the helper of Flutter's gesture tests: testGesture runs a test
// in the test binding (testWidgets) inside its own FakeAsync, and
// GestureTester routes events straight to GestureBinding's pointer router.

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/async"
)

// GestureTester routes pointer events to the gesture binding's router.
type GestureTester struct {
	// Async is the test's fake event loop (Dart's FakeAsync).
	Async *async.Loop
}

// CloseArena closes the gesture arena of pointer.
func (g *GestureTester) CloseArena(pointer int) {
	gestures.GestureBindingInstance().GestureArena().Close(pointer)
}

// Route routes event through GestureBinding.instance.pointerRouter and
// flushes microtasks.
func (g *GestureTester) Route(event gestures.PointerEvent) {
	gestures.GestureBindingInstance().PointerRouter().Route(event)
	g.Async.FlushMicrotasks()
}

// TestGestureFunc is the signature of a testGesture callback (Dart's
// GestureTest).
type TestGestureFunc func(tester *GestureTester)

// RunTestGesture is gesture_tester.dart's testGesture: it runs callback as
// a widget test (the test binding, flutter_test's post-test checks) in a
// FakeAsync of its own.
func RunTestGesture(t *testing.T, callback TestGestureFunc) {
	t.Helper()
	New(t)
	RunTestGestureInCurrentTest(callback)
}

// RunTestGestureInCurrentTest runs callback in a FakeAsync of its own
// inside the widget test in progress (`FakeAsync().run` inside
// testWidgets), with a GestureTester on it.
func RunTestGestureInCurrentTest(callback TestGestureFunc) {
	loop := async.NewLoop()
	restore := async.SetCurrent(loop)
	defer restore()
	callback(&GestureTester{Async: loop})
}
