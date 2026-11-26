package gestures_test

// Helpers of the drag tests (drag_test, monodrag_test, multidrag_test,
// transformed_monodrag_test).

import (
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func off(x, y float64) ui.Offset { return ui.NewOffset(x, y) }

func dexpect[T any](t *testing.T, name string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func expectTrue(t *testing.T, name string, v bool) {
	t.Helper()
	if !v {
		t.Errorf("%s: want true", name)
	}
}

// testGesture is gesture_tester.dart's testGesture.
func testGesture(t *testing.T, callback func(tester *flutter_test.GestureTester)) {
	t.Helper()
	flutter_test.RunTestGesture(t, callback)
}

func intp(v int) *int { return &v }

// tp is flutter_test's TestPointer(pointer, kind, device, buttons).
func tp(pointer int, kind ui.PointerDeviceKind, buttons ...int) *flutter_test.TestPointer {
	o := flutter_test.TestPointerOptions{Pointer: &pointer, Kind: kind}
	if len(buttons) > 0 {
		o.Buttons = &buttons[0]
	}
	return flutter_test.NewTestPointer(o)
}

type mockHitTestTarget struct{}

func (mockHitTestTarget) HandleEvent(PointerEvent, HitTestEntry) {}

func f64(v float64) *float64 { return &v }
