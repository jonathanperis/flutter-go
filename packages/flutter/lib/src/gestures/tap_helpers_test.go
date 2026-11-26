package gestures_test

// Helpers for the ports of the tap, double tap, multitap, serial tap, long
// press and force press tests: Dart's const pointer events (pe* builders
// with Dart's defaults), testGesture (flutter_test.RunTestGesture), and
// expectations.

import (
	"reflect"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// peOpt are the optional fields of the pe* event builders (zero: Dart's
// default; Buttons 0 is kPrimaryButton for down/move events).
type peOpt struct {
	Buttons     int
	Kind        ui.PointerDeviceKind
	TimeStamp   time.Duration
	Pressure    *float64
	PressureMin *float64
	PressureMax *float64
	Delta       ui.Offset
	Device      int
}

func peOne(o []peOpt) peOpt {
	if len(o) > 0 {
		return o[0]
	}
	return peOpt{}
}

func (o peOpt) buttons() *int {
	if o.Buttons == 0 {
		return nil
	}
	b := o.Buttons
	return &b
}

func peDown(pointer int, x, y float64, opts ...peOpt) *PointerDownEvent {
	o := peOne(opts)
	return NewPointerDownEvent(PointerDownEventOptions{Pointer: pointer, Position: ui.NewOffset(x, y), Buttons: o.buttons(),
		Kind: o.Kind, TimeStamp: o.TimeStamp, Pressure: o.Pressure, PressureMin: o.PressureMin, PressureMax: o.PressureMax,
		Device: o.Device})
}

func peMove(pointer int, x, y float64, opts ...peOpt) *PointerMoveEvent {
	o := peOne(opts)
	return NewPointerMoveEvent(PointerMoveEventOptions{Pointer: pointer, Position: ui.NewOffset(x, y), Buttons: o.buttons(),
		Kind: o.Kind, TimeStamp: o.TimeStamp, Pressure: o.Pressure, PressureMin: o.PressureMin, PressureMax: o.PressureMax,
		Delta: o.Delta, Device: o.Device})
}

func peUp(pointer int, x, y float64, opts ...peOpt) *PointerUpEvent {
	o := peOne(opts)
	return NewPointerUpEvent(PointerUpEventOptions{Pointer: pointer, Position: ui.NewOffset(x, y), Kind: o.Kind,
		TimeStamp: o.TimeStamp, Device: o.Device, Buttons: o.Buttons})
}

func peCancel(pointer int, x, y float64, opts ...peOpt) *PointerCancelEvent {
	o := peOne(opts)
	return NewPointerCancelEvent(PointerCancelEventOptions{Pointer: pointer, Position: ui.NewOffset(x, y), Kind: o.Kind,
		TimeStamp: o.TimeStamp, Device: o.Device})
}

func tapF(v float64) *float64 { return &v }

// testGesture is gesture_tester.dart's testGesture.
func tapTestGesture(t *testing.T, body func(tester *flutter_test.GestureTester)) {
	t.Helper()
	flutter_test.RunTestGesture(t, body)
}

func tapArena() *GestureArenaManager { return GestureBindingInstance().GestureArena() }

func expectDeep[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

// strs is a recorded log (Dart's List<String>); its zero value compares
// equal to an empty list.
type strs []string

func (s *strs) add(v string) { *s = append(*s, v) }

func expectLog(t *testing.T, got strs, want ...string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual([]string(got), want) {
		t.Fatalf("log %q, want %q", []string(got), want)
	}
}

// nopArenaMember is the tap tests' TestGestureArenaMember.
type nopArenaMember struct{}

func (nopArenaMember) AcceptGesture(int) {}
func (nopArenaMember) RejectGesture(int) {}

func tapMs(n int) time.Duration { return time.Duration(n) * time.Millisecond }
