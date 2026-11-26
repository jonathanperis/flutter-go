// Port of packages/flutter/test/gestures/events_test.dart (Flutter
// 3.50.1): all 25 tests. testGesture (flutter_test's GestureTester) is not
// needed by the four tests that use it (they only call pure functions), so
// they are plain tests. hasOneLineDescription is checked as Dart's matcher
// does (non-empty, no newline, no "Instance of", not starting/ending with
// whitespace). "Ensure certain event types are allowed" passes if it
// compiles, as in Dart.
package gestures_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

func f(v float64) *float64 { return &v }
func i(v int) *int         { return &v }

func expectOneLine(t *testing.T, s string) {
	t.Helper()
	if s == "" || strings.Contains(s, "\n") || strings.Contains(s, "Instance of") ||
		strings.TrimSpace(s) != s {
		t.Errorf("not a one-line description: %q", s)
	}
}

func TestEventsToStringControl(t *testing.T) {
	expectOneLine(t, NewPointerDownEvent().String())
	expectOneLine(t, NewPointerDownEvent().ToStringFull())
}

func TestNthMouseButton(t *testing.T) {
	if NthMouseButton(2) != KSecondaryMouseButton || NthStylusButton(2) != KSecondaryStylusButton {
		t.Fatal("nth button")
	}
}

func TestSmallestButton(t *testing.T) {
	for in, want := range map[int]int{0x0: 0x0, 0x1: 0x1, 0x200: 0x200, 0x220: 0x20} {
		if got := SmallestButton(in); got != want {
			t.Errorf("SmallestButton(%#x) = %#x", in, got)
		}
	}
}

func TestIsSingleButton(t *testing.T) {
	for in, want := range map[int]bool{0x0: false, 0x1: true, 0x200: true, 0x220: false} {
		if got := IsSingleButton(in); got != want {
			t.Errorf("IsSingleButton(%#x) = %v", in, got)
		}
	}
}

var nonMouseKinds = []ui.PointerDeviceKind{ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus,
	ui.PointerDeviceKindTouch, ui.PointerDeviceKindUnknown}

func TestComputedSlopsByKind(t *testing.T) {
	if ComputeHitSlop(ui.PointerDeviceKindMouse, nil) != KPrecisePointerHitSlop ||
		ComputePanSlop(ui.PointerDeviceKindMouse, nil) != KPrecisePointerPanSlop ||
		ComputeScaleSlop(ui.PointerDeviceKindMouse) != KPrecisePointerScaleSlop {
		t.Fatal("mouse slops")
	}
	for _, k := range nonMouseKinds {
		if ComputeHitSlop(k, nil) != KTouchSlop || ComputePanSlop(k, nil) != KPanSlop || ComputeScaleSlop(k) != KScaleSlop {
			t.Errorf("%v slops", k)
		}
	}
}

func TestComputedSlopsDeferToDevice(t *testing.T) {
	settings := &DeviceGestureSettings{TouchSlop: f(1)}
	if ComputeHitSlop(ui.PointerDeviceKindMouse, settings) != KPrecisePointerHitSlop ||
		ComputePanSlop(ui.PointerDeviceKindMouse, settings) != KPrecisePointerPanSlop {
		t.Fatal("mouse slops")
	}
	for _, k := range nonMouseKinds {
		if ComputeHitSlop(k, settings) != 1 || ComputePanSlop(k, settings) != 2 {
			t.Errorf("%v slops", k)
		}
	}
}

func fromMouseHover() PointerEvent {
	return NewPointerHoverEvent(PointerHoverEventOptions{TimeStamp: 24 * time.Hour, Kind: ui.PointerDeviceKindUnknown,
		Device: 10, Position: ui.NewOffset(101.0, 202.0), Buttons: 7, Obscured: true, PressureMax: f(2.1),
		PressureMin: f(1.1), Distance: 11, DistanceMax: 110, Size: 11, RadiusMajor: 11, RadiusMinor: 9,
		RadiusMin: 1.1, RadiusMax: 22, Orientation: 1.1, Tilt: 1.1, Synthesized: true})
}

func fromMouseMove() PointerEvent {
	return NewPointerMoveEvent(PointerMoveEventOptions{TimeStamp: 24 * time.Hour, Kind: ui.PointerDeviceKindUnknown,
		Device: 10, Position: ui.NewOffset(101.0, 202.0), Buttons: i(7), Obscured: true, PressureMax: f(2.1),
		PressureMin: f(1.1), DistanceMax: 110, Size: 11, RadiusMajor: 11, RadiusMinor: 9,
		RadiusMin: 1.1, RadiusMax: 22, Orientation: 1.1, Tilt: 1.1, Synthesized: true})
}

// expectFields compares the getters of got with want's, except those named
// in fromEmpty, which are compared with empty's.
func expectFields(t *testing.T, got, want, empty PointerEvent, fromEmpty ...string) {
	t.Helper()
	fields := map[string]func(PointerEvent) any{
		"timeStamp": func(e PointerEvent) any { return e.TimeStamp() }, "pointer": func(e PointerEvent) any { return e.Pointer() },
		"kind": func(e PointerEvent) any { return e.Kind() }, "device": func(e PointerEvent) any { return e.Device() },
		"position": func(e PointerEvent) any { return e.Position() }, "delta": func(e PointerEvent) any { return e.Delta() },
		"buttons": func(e PointerEvent) any { return e.Buttons() }, "down": func(e PointerEvent) any { return e.Down() },
		"obscured": func(e PointerEvent) any { return e.Obscured() }, "pressure": func(e PointerEvent) any { return e.Pressure() },
		"pressureMin": func(e PointerEvent) any { return e.PressureMin() }, "pressureMax": func(e PointerEvent) any { return e.PressureMax() },
		"distance": func(e PointerEvent) any { return e.Distance() }, "distanceMax": func(e PointerEvent) any { return e.DistanceMax() },
		"size": func(e PointerEvent) any { return e.Size() }, "radiusMajor": func(e PointerEvent) any { return e.RadiusMajor() },
		"radiusMinor": func(e PointerEvent) any { return e.RadiusMinor() }, "radiusMin": func(e PointerEvent) any { return e.RadiusMin() },
		"radiusMax": func(e PointerEvent) any { return e.RadiusMax() }, "orientation": func(e PointerEvent) any { return e.Orientation() },
		"tilt": func(e PointerEvent) any { return e.Tilt() }, "synthesized": func(e PointerEvent) any { return e.Synthesized() },
	}
	isEmpty := map[string]bool{}
	for _, n := range fromEmpty {
		isEmpty[n] = true
	}
	for name, get := range fields {
		ref := want
		if isEmpty[name] {
			ref = empty
		}
		if get(got) != get(ref) {
			t.Errorf("%s = %v, want %v", name, get(got), get(ref))
		}
	}
}

func TestFromMouseEvent(t *testing.T) {
	hover, move := fromMouseHover(), fromMouseMove()
	emptyEnter, emptyExit := NewPointerEnterEvent(), NewPointerExitEvent()
	// position and delta: the tests set no delta.
	expectFields(t, NewPointerEnterEventFromMouseEvent(hover), hover, emptyEnter, "pointer", "down", "pressure")
	expectFields(t, NewPointerExitEventFromMouseEvent(hover), hover, emptyExit, "pointer", "down", "pressure")
	expectFields(t, NewPointerEnterEventFromMouseEvent(move), move, emptyEnter, "pointer", "pressure")
	expectFields(t, NewPointerExitEventFromMouseEvent(move), move, emptyExit, "pointer", "pressure")
	if _, ok := NewPointerEnterEventFromMouseEvent(hover).(*PointerEnterEvent); !ok {
		t.Fatal("type")
	}
}

func TestDefaultValuesOfPointerEvents(t *testing.T) {
	if NewPointerDownEvent().Buttons() != KPrimaryButton || NewPointerMoveEvent().Buttons() != KPrimaryButton {
		t.Fatal("default buttons")
	}
}

func TestPaintTransformToPointerEventTransform(t *testing.T) {
	original := vm.NewMatrix4Identity()
	changed := PointerEventRemovePerspectiveTransform(original)
	if !changed.Equal(original) {
		t.Fatal("identity changed")
	}
	original = vm.NewMatrix4Identity()
	original.ScaleByDouble(3.0, 3.0, 3.0, 1.0)
	changed = PointerEventRemovePerspectiveTransform(original)
	if changed.Equal(original) {
		t.Fatal("scale unchanged")
	}
	original.SetColumn(2, vm.NewVector4(0, 0, 1, 0))
	original.SetRow(2, vm.NewVector4(0, 0, 1, 0))
	if !changed.Equal(original) {
		t.Fatal("perspective not removed")
	}
}

func TestTransformPosition(t *testing.T) {
	position := ui.NewOffset(20, 30)
	if PointerEventTransformPosition(nil, position) != position ||
		PointerEventTransformPosition(vm.NewMatrix4Identity(), position) != position ||
		PointerEventTransformPosition(vm.NewMatrix4TranslationValues(10, 20, 0), position) != ui.NewOffset(30, 50) {
		t.Fatal("transformPosition")
	}
}

func scale2() *vm.Matrix4 {
	m := vm.NewMatrix4Identity()
	m.ScaleByDouble(2.0, 2.0, 1.0, 1.0)
	return m
}

func TestTransformDeltaViaPositions(t *testing.T) {
	end, delta := ui.NewOffset(20, 30), ui.NewOffset(5, 5)
	if got := PointerEventTransformDeltaViaPositions(end, nil, delta, scale2()); got != ui.NewOffset(10, 10) {
		t.Errorf("got %v", got)
	}
	transformedEnd := ui.NewOffset(40, 60)
	if got := PointerEventTransformDeltaViaPositions(end, &transformedEnd, delta, scale2()); got != ui.NewOffset(10, 10) {
		t.Errorf("got %v", got)
	}
	if got := PointerEventTransformDeltaViaPositions(end, &transformedEnd, delta, nil); got != delta {
		t.Errorf("got %v", got)
	}
}

func expectTransformedEvent(t *testing.T, original PointerEvent, transform *vm.Matrix4, localDelta, localPosition *ui.Offset) {
	t.Helper()
	if original.Position() != original.LocalPosition() || original.Delta() != original.LocalDelta() ||
		original.Original() != nil || original.Transform() != nil {
		t.Fatalf("original %T", original)
	}
	transformed := original.Transformed(transform)
	if transformed.Original() != original || !transformed.Transform().Equal(transform) {
		t.Fatalf("%T: original/transform", original)
	}
	wantDelta, wantPosition := original.LocalDelta(), original.LocalPosition()
	if localDelta != nil {
		wantDelta = *localDelta
	}
	if localPosition != nil {
		wantPosition = *localPosition
	}
	if transformed.LocalDelta() != wantDelta || transformed.LocalPosition() != wantPosition {
		t.Errorf("%T: local %v %v", original, transformed.LocalPosition(), transformed.LocalDelta())
	}
	expectFields(t, transformed, original, nil)
	if transformed.DistanceMin() != original.DistanceMin() || transformed.PlatformData() != original.PlatformData() {
		t.Errorf("%T: distanceMin/platformData", original)
	}
	// Dart: `transformed is PointerXEvent`.
	if reflect.TypeOf(transformed) != reflect.TypeOf(original) {
		t.Errorf("transformed %T is not a %T", transformed, original)
	}
}

func TestTransformingEvents(t *testing.T) {
	transform := scale2().Multiplied(vm.NewMatrix4TranslationValues(10.0, 20.0, 0.0))
	localPosition := ui.NewOffset(60, 100)
	localDelta := ui.NewOffset(10, 10)
	ts := 2 * time.Second
	mouse := ui.PointerDeviceKindMouse
	pos := ui.NewOffset(20, 30)
	d55 := ui.NewOffset(5, 5)

	expectTransformedEvent(t, NewPointerAddedEvent(PointerAddedEventOptions{TimeStamp: ts, Kind: mouse, Device: 1,
		Position: pos, Obscured: true, PressureMin: f(10), PressureMax: f(60), Distance: 12, DistanceMax: 24,
		RadiusMin: 10, RadiusMax: 50, Orientation: 2, Tilt: 4}), transform, nil, &localPosition)
	expectTransformedEvent(t, NewPointerCancelEvent(PointerCancelEventOptions{TimeStamp: ts, Pointer: 45, Kind: mouse,
		Device: 1, Position: pos, Buttons: 4, Obscured: true, PressureMin: f(10), PressureMax: f(60), Distance: 12,
		DistanceMax: 24, Size: 10, RadiusMajor: 33, RadiusMinor: 44, RadiusMin: 10, RadiusMax: 50, Orientation: 2,
		Tilt: 4}), transform, nil, &localPosition)
	expectTransformedEvent(t, NewPointerDownEvent(PointerDownEventOptions{TimeStamp: ts, Pointer: 45, Kind: mouse,
		Device: 1, Position: pos, Buttons: i(4), Obscured: true, Pressure: f(34), PressureMin: f(10), PressureMax: f(60),
		DistanceMax: 24, Size: 10, RadiusMajor: 33, RadiusMinor: 44, RadiusMin: 10, RadiusMax: 50, Orientation: 2,
		Tilt: 4}), transform, nil, &localPosition)
	hoverLike := PointerHoverEventOptions{TimeStamp: ts, Kind: mouse, Device: 1, Position: pos, Delta: d55,
		Buttons: 4, Obscured: true, PressureMin: f(10), PressureMax: f(60), Distance: 12, DistanceMax: 24, Size: 10,
		RadiusMajor: 33, RadiusMinor: 44, RadiusMin: 10, RadiusMax: 50, Orientation: 2, Tilt: 4, Synthesized: true}
	expectTransformedEvent(t, NewPointerEnterEvent(PointerEnterEventOptions{PointerHoverEventOptions: hoverLike}),
		transform, &localDelta, &localPosition)
	expectTransformedEvent(t, NewPointerExitEvent(PointerEnterEventOptions{PointerHoverEventOptions: hoverLike}),
		transform, &localDelta, &localPosition)
	expectTransformedEvent(t, NewPointerHoverEvent(hoverLike), transform, &localDelta, &localPosition)
	expectTransformedEvent(t, NewPointerMoveEvent(PointerMoveEventOptions{TimeStamp: ts, Pointer: 45, Kind: mouse,
		Device: 1, Position: pos, Delta: d55, Buttons: i(4), Obscured: true, Pressure: f(34), PressureMin: f(10),
		PressureMax: f(60), DistanceMax: 24, Size: 10, RadiusMajor: 33, RadiusMinor: 44, RadiusMin: 10, RadiusMax: 50,
		Orientation: 2, Tilt: 4, PlatformData: 10, Synthesized: true}), transform, &localDelta, &localPosition)
	expectTransformedEvent(t, NewPointerRemovedEvent(PointerRemovedEventOptions{TimeStamp: ts, Kind: mouse, Device: 1,
		Position: pos, Obscured: true, PressureMin: f(10), PressureMax: f(60), DistanceMax: 24, RadiusMin: 10,
		RadiusMax: 50}), transform, nil, &localPosition)
	expectTransformedEvent(t, NewPointerScrollEvent(PointerScrollEventOptions{PointerSignalEventOptions: PointerSignalEventOptions{
		TimeStamp: ts, Device: 1, Position: pos}}), transform, nil, &localPosition)
	panZoom := PointerPanZoomEventOptions{TimeStamp: ts, Device: 1, Position: pos}
	expectTransformedEvent(t, NewPointerPanZoomStartEvent(panZoom), transform, nil, &localPosition)
	expectTransformedEvent(t, NewPointerPanZoomUpdateEvent(PointerPanZoomUpdateEventOptions{PointerPanZoomEventOptions: panZoom}),
		transform, nil, &localPosition)
	expectTransformedEvent(t, NewPointerPanZoomEndEvent(panZoom), transform, nil, &localPosition)
	expectTransformedEvent(t, NewPointerUpEvent(PointerUpEventOptions{TimeStamp: ts, Pointer: 45, Kind: mouse,
		Device: 1, Position: pos, Buttons: 4, Obscured: true, Pressure: 34, PressureMin: f(10), PressureMax: f(60),
		Distance: 12, DistanceMax: 24, Size: 10, RadiusMajor: 33, RadiusMinor: 44, RadiusMin: 10, RadiusMax: 50,
		Orientation: 2, Tilt: 4}), transform, nil, &localPosition)
}

func TestCopyWith(t *testing.T) {
	day := 24 * time.Hour
	unknown := ui.PointerDeviceKindUnknown
	pos := ui.NewOffset(101.0, 202.0)
	added := NewPointerAddedEvent(PointerAddedEventOptions{TimeStamp: day, Kind: unknown, Device: 10, Position: pos,
		Obscured: true, PressureMax: f(2.1), PressureMin: f(1.1), Distance: 11, DistanceMax: 110, RadiusMin: 1.1,
		RadiusMax: 22, Orientation: 1.1, Tilt: 1.1})
	hover := fromMouseHover()
	down := NewPointerDownEvent(PointerDownEventOptions{TimeStamp: day, Pointer: 1, Kind: unknown, Device: 10,
		Position: pos, Buttons: i(7), Obscured: true, PressureMax: f(2.1), PressureMin: f(1.1), DistanceMax: 110,
		Size: 11, RadiusMajor: 11, RadiusMinor: 9, RadiusMin: 1.1, RadiusMax: 22, Orientation: 1.1, Tilt: 1.1})
	move := NewPointerMoveEvent(PointerMoveEventOptions{TimeStamp: day, Pointer: 1, Kind: unknown, Device: 10,
		Position: pos, Delta: ui.NewOffset(1.0, 2.0), Buttons: i(7), Obscured: true, PressureMax: f(2.1),
		PressureMin: f(1.1), DistanceMax: 110, Size: 11, RadiusMajor: 11, RadiusMinor: 9, RadiusMin: 1.1,
		RadiusMax: 22, Orientation: 1.1, Tilt: 1.1, Synthesized: true})
	up := NewPointerUpEvent(PointerUpEventOptions{TimeStamp: day, Pointer: 1, Kind: unknown, Device: 10,
		Position: pos, Buttons: 7, Obscured: true, PressureMax: f(2.1), PressureMin: f(1.1), DistanceMax: 110,
		Size: 11, RadiusMajor: 11, RadiusMinor: 9, RadiusMin: 1.1, RadiusMax: 22, Orientation: 1.1, Tilt: 1.1})
	removed := NewPointerRemovedEvent(PointerRemovedEventOptions{TimeStamp: day, Kind: unknown, Device: 10,
		Position: pos, Obscured: true, PressureMax: f(2.1), PressureMin: f(1.1), DistanceMax: 110, RadiusMin: 1.1,
		RadiusMax: 22})

	position := ui.OffsetZero
	timeStamp := 2 * day
	copyOf := func(e PointerEvent) PointerEvent {
		c := e.CopyWith(PointerEventCopyWithOptions{Position: &position, TimeStamp: &timeStamp})
		if c.TimeStamp() != timeStamp || c.Position() != position {
			t.Fatalf("%T: copied timeStamp/position", e)
		}
		return c
	}
	// The fields whose value comes from an empty event (besides the replaced
	// position and time stamp, compared above).
	expectFields(t, withPosTime(copyOf(added), added), added, NewPointerAddedEvent(),
		"pointer", "delta", "down", "pressure", "size", "radiusMajor", "radiusMinor", "synthesized")
	expectFields(t, withPosTime(copyOf(hover), hover), hover, NewPointerHoverEvent(), "pointer", "delta", "down", "pressure")
	expectFields(t, withPosTime(copyOf(down), down), down, NewPointerDownEvent(), "delta", "down", "pressure", "synthesized")
	expectFields(t, withPosTime(copyOf(move), move), move, NewPointerMoveEvent(), "pressure")
	expectFields(t, withPosTime(copyOf(up), up), up, NewPointerUpEvent(), "down", "pressure", "synthesized")
	expectFields(t, withPosTime(copyOf(removed), removed), removed, NewPointerRemovedEvent(),
		"pointer", "delta", "down", "pressure", "distance", "size", "radiusMajor", "radiusMinor", "orientation", "tilt", "synthesized")
}

// withPosTime returns a copy of c with e's position and time stamp, so
// expectFields can compare the remaining fields.
func withPosTime(c, e PointerEvent) PointerEvent {
	p, ts := e.Position(), e.TimeStamp()
	return c.CopyWith(PointerEventCopyWithOptions{Position: &p, TimeStamp: &ts})
}

func TestCertainEventTypesAllowed(t *testing.T) {
	trackpad := ui.PointerDeviceKindTrackpad
	_ = NewPointerHoverEvent(PointerHoverEventOptions{Kind: trackpad})
	_ = NewPointerScrollInertiaCancelEvent(PointerSignalEventOptions{Kind: &trackpad})
	_ = NewPointerScrollEvent(PointerScrollEventOptions{PointerSignalEventOptions: PointerSignalEventOptions{Kind: &trackpad}})
}

func TestCertainEventTypesNotAllowed(t *testing.T) {
	trackpad := ui.PointerDeviceKindTrackpad
	for name, build := range map[string]func(){
		"down": func() { NewPointerDownEvent(PointerDownEventOptions{Kind: trackpad}) },
		"move": func() { NewPointerMoveEvent(PointerMoveEventOptions{Kind: trackpad}) },
		"up":   func() { NewPointerUpEvent(PointerUpEventOptions{Kind: trackpad}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s with a trackpad kind didn't panic", name)
				}
			}()
			build()
		}()
	}
}
