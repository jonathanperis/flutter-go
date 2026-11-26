package gestures_test

// Port of packages/flutter/test/gestures/scale_test.dart and
// transformed_scale_test.dart (Flutter 3.50.1): all 15 tests of
// scale_test.dart, and transformed_scale_test's "gets local coordinates",
// adapted: it pumps a GestureDetector in a Center; here the down events
// carry the hit test's transform (translate(-350, -250), the 100x100 box
// centered in 800x600) directly.

import (
	"math"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// newTapStub is the tests' TapGestureRecognizer whose onTap sets didTap:
// every test expects didTap to stay false, so a tap fails the test.
func newTapStub(t *testing.T) *TapGestureRecognizer {
	tap := NewTapGestureRecognizer()
	tap.OnTap = func() { t.Error("didTap") }
	return tap
}

func sOff(x, y float64) ui.Offset { return ui.NewOffset(x, y) }

func sIP(i int) *int { return &i }

type scaleProbe struct {
	didStart, didEnd                bool
	focal, delta                    *ui.Offset
	scale, hScale, vScale, rotation *float64
	pointerCount                    *int
	startTimestamp, updateTimestamp *time.Duration
	endVelocity                     *float64
	kind                            *ui.PointerDeviceKind
}

func sProbe(s *ScaleGestureRecognizer) *scaleProbe {
	p := &scaleProbe{}
	s.OnStart = func(d *ScaleStartDetails) {
		p.didStart = true
		f := d.FocalPoint
		p.focal = &f
		p.pointerCount = sIP(d.PointerCount)
		p.startTimestamp = d.SourceTimeStamp
		p.kind = d.Kind
	}
	s.OnUpdate = func(d *ScaleUpdateDetails) {
		sc, h, v, r := d.Scale, d.HorizontalScale, d.VerticalScale, d.Rotation
		p.scale, p.hScale, p.vScale, p.rotation = &sc, &h, &v, &r
		f, dl := d.FocalPoint, d.FocalPointDelta
		p.focal, p.delta = &f, &dl
		p.pointerCount = sIP(d.PointerCount)
		p.updateTimestamp = d.SourceTimeStamp
	}
	s.OnEnd = func(d *ScaleEndDetails) {
		p.didEnd = true
		v := d.ScaleVelocity
		p.endVelocity = &v
	}
	return p
}

func (p *scaleProbe) clearUpdate() {
	p.focal, p.delta, p.scale, p.hScale, p.vScale, p.rotation, p.pointerCount = nil, nil, nil, nil, nil, nil, nil
	p.updateTimestamp = nil
}

func sNil[T any](t *testing.T, what string, v *T) {
	t.Helper()
	if v != nil {
		t.Errorf("%s = %v, want null", what, *v)
	}
}

func sEq[T comparable](t *testing.T, what string, v *T, want T) {
	t.Helper()
	if v == nil {
		t.Errorf("%s = null, want %v", what, want)
	} else if *v != want {
		t.Errorf("%s = %v, want %v", what, *v, want)
	}
}

func sBool(t *testing.T, what string, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func sClose(t *testing.T, what string, v *float64, want, eps float64) {
	t.Helper()
	if v == nil {
		t.Errorf("%s = null, want %v", what, want)
	} else if math.Abs(*v-want) > eps {
		t.Errorf("%s = %v, want %v ± %v", what, *v, want, eps)
	}
}

func (p *scaleProbe) expectNoUpdate(t *testing.T) {
	t.Helper()
	sNil(t, "updatedScale", p.scale)
	sNil(t, "updatedFocalPoint", p.focal)
	sNil(t, "updatedDelta", p.delta)
	sNil(t, "updatedPointerCount", p.pointerCount)
}

func sTP(pointer int, kind ...ui.PointerDeviceKind) *flutter_test.TestPointer {
	o := flutter_test.TestPointerOptions{Pointer: sIP(pointer)}
	if len(kind) > 0 {
		o.Kind = kind[0]
	}
	return flutter_test.NewTestPointer(o)
}

func sTS(ms int) flutter_test.TestPointerEventOptions {
	return flutter_test.TestPointerEventOptions{TimeStamp: time.Duration(ms) * time.Millisecond}
}

func TestScaleShouldRecognizeScaleGestures(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		tap := newTapStub(t)
		p := sProbe(scale)

		pointer1 := sTP(1)
		down := pointer1.Down(sOff(0, 0))
		scale.AddPointer(down)
		tap.AddPointer(down)
		tester.CloseArena(1)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		sBool(t, "didEndScale", p.didEnd, false)

		// One-finger panning
		tester.Route(down)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)

		tester.Route(pointer1.Move(sOff(20, 30)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(20, 30))
		sEq(t, "pointerCount", p.pointerCount, 1)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		// Two-finger scaling
		pointer2 := sTP(2)
		down2 := pointer2.Down(sOff(10, 20))
		scale.AddPointer(down2)
		tap.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		p.expectNoUpdate(t)
		sBool(t, "didStartScale", p.didStart, false)

		// Zoom in
		tester.Route(pointer2.Move(sOff(0, 10)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(10, 20))
		sEq(t, "scale", p.scale, 2.0)
		sEq(t, "hScale", p.hScale, 2.0)
		sEq(t, "vScale", p.vScale, 2.0)
		sEq(t, "delta", p.delta, sOff(-5, -5))
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		// Zoom out
		tester.Route(pointer2.Move(sOff(15, 25)))
		sEq(t, "focal", p.focal, sOff(17.5, 27.5))
		sEq(t, "scale", p.scale, 0.5)
		sEq(t, "hScale", p.hScale, 0.5)
		sEq(t, "vScale", p.vScale, 0.5)
		sEq(t, "delta", p.delta, sOff(7.5, 7.5))
		sEq(t, "pointerCount", p.pointerCount, 2)

		// Horizontal scaling
		tester.Route(pointer2.Move(sOff(0, 20)))
		sEq(t, "hScale", p.hScale, 2.0)
		sEq(t, "vScale", p.vScale, 1.0)
		sEq(t, "pointerCount", p.pointerCount, 2)

		// Vertical scaling
		tester.Route(pointer2.Move(sOff(10, 10)))
		sEq(t, "hScale", p.hScale, 1.0)
		sEq(t, "vScale", p.vScale, 2.0)
		sEq(t, "delta", p.delta, sOff(5, -5))
		sEq(t, "pointerCount", p.pointerCount, 2)
		tester.Route(pointer2.Move(sOff(15, 25)))
		p.clearUpdate()

		// Three-finger scaling
		pointer3 := sTP(3)
		down3 := pointer3.Down(sOff(25, 35))
		scale.AddPointer(down3)
		tap.AddPointer(down3)
		tester.CloseArena(3)
		tester.Route(down3)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		p.expectNoUpdate(t)
		sBool(t, "didStartScale", p.didStart, false)

		// Zoom in
		tester.Route(pointer3.Move(sOff(55, 65)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(30, 40))
		sEq(t, "scale", p.scale, 5.0)
		sEq(t, "delta", p.delta, sOff(10, 10))
		sEq(t, "pointerCount", p.pointerCount, 3)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		// Return to original positions but with different fingers
		tester.Route(pointer1.Move(sOff(25, 35)))
		tester.Route(pointer2.Move(sOff(20, 30)))
		tester.Route(pointer3.Move(sOff(15, 25)))
		sBool(t, "didStartScale", p.didStart, false)
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 1.0)
		dx, dy := p.delta.Dx, p.delta.Dy
		sClose(t, "delta.dx", &dx, -13.3, 0.1)
		sClose(t, "delta.dy", &dy, -13.3, 0.1)
		sEq(t, "pointerCount", p.pointerCount, 3)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		tester.Route(pointer1.Up())
		sBool(t, "didStartScale", p.didStart, false)
		sNil(t, "focal", p.focal)
		sNil(t, "scale", p.scale)
		sNil(t, "delta", p.delta)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false

		// Continue scaling with two fingers
		tester.Route(pointer3.Move(sOff(10, 20)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(15, 25))
		sEq(t, "scale", p.scale, 2.0)
		sEq(t, "delta", p.delta, sOff(-2.5, -2.5))
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()

		// Continue rotating with two fingers
		tester.Route(pointer3.Move(sOff(30, 40)))
		sEq(t, "focal", p.focal, sOff(25, 35))
		sEq(t, "scale", p.scale, 2.0)
		sEq(t, "delta", p.delta, sOff(10, 10))
		p.clearUpdate()
		tester.Route(pointer3.Move(sOff(10, 20)))
		sEq(t, "focal", p.focal, sOff(15, 25))
		sEq(t, "scale", p.scale, 2.0)
		sEq(t, "delta", p.delta, sOff(-10, -10))
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()

		tester.Route(pointer2.Up())
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false

		// Continue panning with one finger
		tester.Route(pointer3.Move(sOff(0, 0)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(0, 0))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(-10, -20))
		sEq(t, "pointerCount", p.pointerCount, 1)
		p.clearUpdate()

		// We are done
		tester.Route(pointer3.Up())
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		sBool(t, "didEndScale", p.didEnd, true)

		scale.Dispose()
		tap.Dispose()
	})
}

func TestScaleRejectsUnallowedDeviceKinds(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer(ScaleGestureRecognizerOptions{GestureRecognizerOptions: GestureRecognizerOptions{
			SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindTouch: {}}}})
		p := sProbe(scale)
		mousePointer := sTP(1, ui.PointerDeviceKindMouse)
		down := mousePointer.Down(sOff(0, 0))
		scale.AddPointer(down)
		tester.CloseArena(1)

		tester.Route(down)
		sBool(t, "didStartScale", p.didStart, false)
		sNil(t, "scale", p.scale)

		tester.Route(mousePointer.Move(sOff(20, 30)))
		sBool(t, "didStartScale", p.didStart, false)
		sNil(t, "scale", p.scale)
		scale.Dispose()
	})
}

func TestScaleFromAllowedDevicesCannotBeEndedFromUnallowed(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer(ScaleGestureRecognizerOptions{GestureRecognizerOptions: GestureRecognizerOptions{
			SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindTouch: {}}}})
		p := sProbe(scale)
		touchPointer := sTP(1)
		down := touchPointer.Down(sOff(0, 0))
		scale.AddPointer(down)
		tester.CloseArena(1)

		tester.Route(down)
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sNil(t, "scale", p.scale)
		sEq(t, "focal", p.focal, sOff(0, 0))
		sBool(t, "didEndScale", p.didEnd, false)

		tester.Route(touchPointer.Move(sOff(20, 30)))
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 1.0)
		p.focal, p.scale = nil, nil
		sBool(t, "didEndScale", p.didEnd, false)

		mousePointer := sTP(2, ui.PointerDeviceKindMouse)
		down2 := mousePointer.Down(sOff(10, 20))
		scale.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		sBool(t, "didEndScale", p.didEnd, false)
		sNil(t, "scale", p.scale)
		sBool(t, "didStartScale", p.didStart, false)

		tester.Route(mousePointer.Move(sOff(0, 10)))
		sNil(t, "scale", p.scale)
		sBool(t, "didEndScale", p.didEnd, false)
		scale.Dispose()
	})
}

func TestScaleShouldRecognizeRotationGestures(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		tap := newTapStub(t)
		p := sProbe(scale)

		pointer1 := sTP(1)
		down := pointer1.Down(sOff(0, 0))
		scale.AddPointer(down)
		tap.AddPointer(down)
		tester.CloseArena(1)
		sBool(t, "didStartScale", p.didStart, false)
		sNil(t, "rotation", p.rotation)

		tester.Route(down)
		tester.Route(pointer1.Move(sOff(20, 30)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "delta", p.delta, sOff(20, 30))
		sEq(t, "rotation", p.rotation, 0.0)
		sEq(t, "pointerCount", p.pointerCount, 1)
		p.clearUpdate()

		pointer2 := sTP(2)
		down2 := pointer2.Down(sOff(30, 40))
		scale.AddPointer(down2)
		tap.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		sNil(t, "rotation", p.rotation)
		sBool(t, "didStartScale", p.didStart, false)

		tester.Route(pointer2.Move(sOff(40, 50)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(30, 40))
		sEq(t, "delta", p.delta, sOff(5, 5))
		sEq(t, "rotation", p.rotation, 0.0)
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()

		// Rotation
		tester.Route(pointer2.Move(sOff(0, 10)))
		sEq(t, "focal", p.focal, sOff(10, 20))
		sEq(t, "delta", p.delta, sOff(-20, -20))
		sEq(t, "rotation", p.rotation, math.Pi)
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()

		pointer3 := sTP(3)
		down3 := pointer3.Down(sOff(25, 35))
		scale.AddPointer(down3)
		tap.AddPointer(down3)
		tester.CloseArena(3)
		tester.Route(down3)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		sNil(t, "rotation", p.rotation)

		tester.Route(pointer3.Move(sOff(55, 65)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(25, 35))
		sEq(t, "delta", p.delta, sOff(10, 10))
		sEq(t, "rotation", p.rotation, 0.0)
		sEq(t, "pointerCount", p.pointerCount, 3)
		p.clearUpdate()

		tester.Route(pointer1.Move(sOff(25, 35)))
		tester.Route(pointer2.Move(sOff(20, 30)))
		tester.Route(pointer3.Move(sOff(15, 25)))
		sBool(t, "didStartScale", p.didStart, false)
		sEq(t, "focal", p.focal, sOff(20, 30))
		dx, dy := p.delta.Dx, p.delta.Dy
		sClose(t, "delta.dx", &dx, -13.3, 0.1)
		sClose(t, "delta.dy", &dy, -13.3, 0.1)
		sEq(t, "rotation", p.rotation, 0.0)
		sEq(t, "pointerCount", p.pointerCount, 3)
		p.clearUpdate()

		tester.Route(pointer1.Up())
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		sNil(t, "rotation", p.rotation)

		tester.Route(pointer3.Move(sOff(10, 20)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(15, 25))
		sEq(t, "delta", p.delta, sOff(-2.5, -2.5))
		sEq(t, "rotation", p.rotation, 0.0)
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()

		tester.Route(pointer3.Move(sOff(30, 40)))
		sEq(t, "focal", p.focal, sOff(25, 35))
		sEq(t, "delta", p.delta, sOff(10, 10))
		sEq(t, "rotation", p.rotation, -math.Pi)
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()
		tester.Route(pointer3.Move(sOff(10, 20)))
		sEq(t, "focal", p.focal, sOff(15, 25))
		sEq(t, "delta", p.delta, sOff(-10, -10))
		sEq(t, "rotation", p.rotation, 0.0)
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()

		tester.Route(pointer2.Up())
		sBool(t, "didStartScale", p.didStart, false)
		sNil(t, "rotation", p.rotation)
		sNil(t, "pointerCount", p.pointerCount)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false

		// We are done
		tester.Route(pointer3.Up())
		sBool(t, "didStartScale", p.didStart, false)
		sNil(t, "rotation", p.rotation)
		sBool(t, "didEndScale", p.didEnd, false)

		scale.Dispose()
		tap.Dispose()
	})
}

func TestScaleFirstRotation(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		defer scale.Dispose()
		var updatedRotation *float64
		scale.OnUpdate = func(d *ScaleUpdateDetails) { r := d.Rotation; updatedRotation = &r }

		pointer1 := sTP(1)
		down := pointer1.Down(sOff(0, 0))
		scale.AddPointer(down)
		tester.CloseArena(1)
		tester.Route(down)

		pointer2 := sTP(2)
		down2 := pointer2.Down(sOff(10, 10))
		scale.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		sNil(t, "rotation", updatedRotation)

		// Rotation 45°.
		tester.Route(pointer2.Move(sOff(0, 10)))
		sEq(t, "rotation", updatedRotation, math.Pi/4.0)
	})
}

func TestScalePointerCount(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		var start, update, end int
		scale.OnStart = func(d *ScaleStartDetails) { start = d.PointerCount }
		scale.OnUpdate = func(d *ScaleUpdateDetails) { update = d.PointerCount }
		scale.OnEnd = func(d *ScaleEndDetails) { end = d.PointerCount }
		check := func(what string, got, want int) {
			t.Helper()
			if got != want {
				t.Errorf("%s = %d, want %d", what, got, want)
			}
		}

		pointer1 := sTP(1)
		down := pointer1.Down(sOff(0, 0))
		scale.AddPointer(down)
		tester.CloseArena(1)
		tester.Route(down)
		check("start", start, 1)
		tester.Route(pointer1.Move(sOff(20, 30)))
		check("update", update, 1)

		pointer2 := sTP(2)
		down2 := pointer2.Down(sOff(10, 20))
		scale.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		check("end", end, 2)

		tester.Route(pointer2.Move(sOff(0, 10)))
		check("start", start, 2)
		check("update", update, 2)

		tester.Route(pointer1.Up())
		check("end", end, 1)

		tester.Route(pointer2.Move(sOff(0, 10)))
		check("start", start, 1)
		check("update", update, 1)

		tester.Route(pointer2.Up())
		check("end", end, 0)
		scale.Dispose()
	})
}

func sPanZoomUpdate(p *flutter_test.TestPointer, location ui.Offset, pan ui.Offset, opts ...func(*flutter_test.TestPointerPanZoomUpdateOptions)) *PointerPanZoomUpdateEvent {
	o := flutter_test.TestPointerPanZoomUpdateOptions{Pan: pan}
	for _, f := range opts {
		f(&o)
	}
	return p.PanZoomUpdate(location, o)
}

func TestScaleTrackpadScrollCausesScale(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		start := DragStartBehaviorStart
		scale := NewScaleGestureRecognizer(ScaleGestureRecognizerOptions{DragStartBehavior: &start, TrackpadScrollCausesScale: true})
		p := sProbe(scale)

		pointer1 := sTP(2, ui.PointerDeviceKindTrackpad)
		run := func(dir string, firstPan, secondPan ui.Offset) {
			startEvent := pointer1.PanZoomStart(sOff(0, 0))
			scale.AddPointerPanZoom(startEvent)
			tester.CloseArena(2)
			sBool(t, "didStartScale", p.didStart, false)
			p.expectNoUpdate(t)
			sBool(t, "didEndScale", p.didEnd, false)

			tester.Route(startEvent)
			sBool(t, "didStartScale", p.didStart, true)
			p.didStart = false
			sNil(t, "scale", p.scale)
			sEq(t, "focal", p.focal, sOff(0, 0))
			sNil(t, "delta", p.delta)
			sEq(t, "pointerCount", p.pointerCount, 2)
			p.clearUpdate()
			sBool(t, "didEndScale", p.didEnd, false)

			// Zoom in.
			tester.Route(sPanZoomUpdate(pointer1, sOff(0, 0), firstPan))
			sBool(t, "didStartScale", p.didStart, false)
			sEq(t, "focal", p.focal, sOff(0, 0))
			sEq(t, "scale "+dir, p.scale, math.E)
			sEq(t, "delta", p.delta, sOff(0, 0))
			sEq(t, "pointerCount", p.pointerCount, 2)
			p.clearUpdate()

			// A scroll along the other axis should do nothing.
			tester.Route(sPanZoomUpdate(pointer1, sOff(0, 0), secondPan))
			sEq(t, "focal", p.focal, sOff(0, 0))
			sEq(t, "scale "+dir, p.scale, math.E)
			sEq(t, "delta", p.delta, sOff(0, 0))
			sEq(t, "pointerCount", p.pointerCount, 2)
			p.clearUpdate()

			tester.Route(pointer1.PanZoomEnd())
			sBool(t, "didStartScale", p.didStart, false)
			p.expectNoUpdate(t)
			sBool(t, "didEndScale", p.didEnd, true)
			p.didEnd = false
		}
		run("vertical", sOff(0, -200), sOff(200, -200))
		// Try with a different trackpadScrollToScaleFactor
		scale.TrackpadScrollToScaleFactor = sOff(1.0/125, 0)
		run("horizontal", sOff(125, 0), sOff(125, 125))
		scale.Dispose()
	})
}

func TestScaleEndingVelocity(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		start := DragStartBehaviorStart
		scale := NewScaleGestureRecognizer(ScaleGestureRecognizerOptions{DragStartBehavior: &start, TrackpadScrollCausesScale: true})
		p := sProbe(scale)
		pointer1 := sTP(2, ui.PointerDeviceKindTrackpad)
		startEvent := pointer1.PanZoomStart(sOff(0, 0))
		scale.AddPointerPanZoom(startEvent)
		tester.CloseArena(2)
		tester.Route(startEvent)
		sBool(t, "didStartScale", p.didStart, true)
		sEq(t, "focal", p.focal, sOff(0, 0))

		for i := 0; i < 100; i++ {
			tester.Route(sPanZoomUpdate(pointer1, sOff(0, 0), sOff(0, float64(i)*-10), func(o *flutter_test.TestPointerPanZoomUpdateOptions) {
				o.TimeStamp = time.Duration(i*25) * time.Millisecond
			}))
		}
		tester.Route(pointer1.PanZoomEnd(sTS(2500)))
		sBool(t, "didEndScale", p.didEnd, true)
		// moreOrLessEquals: 1e-10.
		sClose(t, "scaleEndVelocity", p.endVelocity, 281.41454098027765, 1e-10)
		scale.Dispose()
	})
}

func TestScaleDetailsContainEventTimestamps(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		tap := newTapStub(t)
		p := sProbe(scale)
		ms := func(m int) time.Duration { return time.Duration(m) * time.Millisecond }

		pointer1 := sTP(1)
		down := pointer1.Down(sOff(0, 0), sTS(10))
		scale.AddPointer(down)
		tap.AddPointer(down)
		tester.CloseArena(1)
		sNil(t, "updateTimestamp", p.updateTimestamp)
		sNil(t, "startTimestamp", p.startTimestamp)

		tester.Route(down)
		sBool(t, "didStartScale", p.didStart, false)
		sNil(t, "startTimestamp", p.startTimestamp)

		tester.Route(pointer1.Move(sOff(20, 30), sTS(20)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(20, 30))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(20))
		sEq(t, "startTimestamp", p.startTimestamp, ms(10))
		p.clearUpdate()
		p.startTimestamp = nil
		if scale.PointerCount() != 1 {
			t.Errorf("pointerCount %d", scale.PointerCount())
		}

		pointer2 := sTP(2)
		down2 := pointer2.Down(sOff(10, 20), sTS(30))
		scale.AddPointer(down2)
		tap.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		if scale.PointerCount() != 2 {
			t.Errorf("pointerCount %d", scale.PointerCount())
		}
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		sNil(t, "updateTimestamp", p.updateTimestamp)
		sNil(t, "startTimestamp", p.startTimestamp)

		tester.Route(pointer2.Move(sOff(0, 10), sTS(40)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(10, 20))
		sEq(t, "scale", p.scale, 2.0)
		sEq(t, "delta", p.delta, sOff(-5, -5))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(40))
		sEq(t, "startTimestamp", p.startTimestamp, ms(40))
		p.clearUpdate()
		p.startTimestamp = nil

		tester.Route(pointer2.Move(sOff(15, 25), sTS(50)))
		sEq(t, "focal", p.focal, sOff(17.5, 27.5))
		sEq(t, "scale", p.scale, 0.5)
		sEq(t, "delta", p.delta, sOff(7.5, 7.5))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(50))
		sNil(t, "startTimestamp", p.startTimestamp)

		tester.Route(pointer2.Move(sOff(0, 20), sTS(60)))
		sEq(t, "hScale", p.hScale, 2.0)
		sEq(t, "vScale", p.vScale, 1.0)
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(60))

		tester.Route(pointer2.Move(sOff(10, 10), sTS(70)))
		sEq(t, "hScale", p.hScale, 1.0)
		sEq(t, "vScale", p.vScale, 2.0)
		sEq(t, "delta", p.delta, sOff(5, -5))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(70))
		tester.Route(pointer2.Move(sOff(15, 25)))
		p.clearUpdate()

		pointer3 := sTP(3)
		down3 := pointer3.Down(sOff(25, 35), sTS(80))
		scale.AddPointer(down3)
		tap.AddPointer(down3)
		tester.CloseArena(3)
		tester.Route(down3)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		sNil(t, "startTimestamp", p.startTimestamp)

		tester.Route(pointer3.Move(sOff(55, 65), sTS(90)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(30, 40))
		sEq(t, "scale", p.scale, 5.0)
		sEq(t, "delta", p.delta, sOff(10, 10))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(90))
		sEq(t, "startTimestamp", p.startTimestamp, ms(90))
		p.clearUpdate()
		p.startTimestamp = nil

		tester.Route(pointer1.Move(sOff(25, 35), sTS(100)))
		tester.Route(pointer2.Move(sOff(20, 30), sTS(110)))
		tester.Route(pointer3.Move(sOff(15, 25), sTS(120)))
		sBool(t, "didStartScale", p.didStart, false)
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(120))
		sNil(t, "startTimestamp", p.startTimestamp)
		p.clearUpdate()
		tester.Route(pointer1.Up())
		sBool(t, "didEndScale", p.didEnd, true)
		sNil(t, "updateTimestamp", p.updateTimestamp)
		p.didEnd = false

		tester.Route(pointer3.Move(sOff(10, 20), sTS(130)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(15, 25))
		sEq(t, "scale", p.scale, 2.0)
		sEq(t, "delta", p.delta, sOff(-2.5, -2.5))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(130))
		sEq(t, "startTimestamp", p.startTimestamp, ms(130))
		p.clearUpdate()
		p.startTimestamp = nil

		tester.Route(pointer3.Move(sOff(30, 40), sTS(140)))
		sEq(t, "delta", p.delta, sOff(10, 10))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(140))
		p.clearUpdate()
		tester.Route(pointer3.Move(sOff(10, 20), sTS(140)))
		sEq(t, "delta", p.delta, sOff(-10, -10))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(140))
		p.clearUpdate()

		tester.Route(pointer2.Up())
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false

		tester.Route(pointer3.Move(sOff(0, 0), sTS(150)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(0, 0))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(-10, -20))
		sEq(t, "updateTimestamp", p.updateTimestamp, ms(150))
		sEq(t, "startTimestamp", p.startTimestamp, ms(150))
		p.clearUpdate()
		p.startTimestamp = nil

		tester.Route(pointer3.Up())
		sBool(t, "didStartScale", p.didStart, false)
		sBool(t, "didEndScale", p.didEnd, true)
		sNil(t, "updateTimestamp", p.updateTimestamp)
		sNil(t, "startTimestamp", p.startTimestamp)

		scale.Dispose()
		tap.Dispose()
	})
}

func TestScaleStartDetailsKind(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		didStart := false
		var kind *ui.PointerDeviceKind
		scale.OnStart = func(d *ScaleStartDetails) { didStart = true; kind = d.Kind }
		scale.OnEnd = func(*ScaleEndDetails) { didStart = false }

		for i, k := range []ui.PointerDeviceKind{ui.PointerDeviceKindTouch, ui.PointerDeviceKindMouse,
			ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus, ui.PointerDeviceKindUnknown} {
			pointer := sTP(i+1, k)
			loc := sOff(10, 20)
			if i == 0 {
				loc = sOff(0, 0)
			}
			down := pointer.Down(loc)
			scale.AddPointer(down)
			tester.CloseArena(i + 1)
			tester.Route(down)
			tester.Route(pointer.Move(sOff(20, 30)))
			sBool(t, "didStartScale "+k.String(), didStart, true)
			sEq(t, "kind", kind, k)
			tester.Route(pointer.Up())
			sBool(t, "didStartScale after up", didStart, false)
		}

		pointer6 := sTP(6, ui.PointerDeviceKindTrackpad)
		down6 := pointer6.PanZoomStart(sOff(10, 20))
		scale.AddPointerPanZoom(down6)
		tester.CloseArena(6)
		tester.Route(down6)
		tester.Route(sPanZoomUpdate(pointer6, sOff(20, 30), sOff(0, 0)))
		sBool(t, "didStartScale trackpad", didStart, true)
		sEq(t, "kind", kind, ui.PointerDeviceKindTrackpad)
		tester.Route(pointer6.PanZoomEnd())
		sBool(t, "didStartScale after end", didStart, false)
		scale.Dispose()
	})
}

// transformed_scale_test.dart: "gets local coordinates" (adapted: the
// events carry the hit test's transform instead of a pumped widget tree).
func TestTransformedScaleGetsLocalCoordinates(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		var starts []*ScaleStartDetails
		var updates []*ScaleUpdateDetails
		scale := NewScaleGestureRecognizer()
		scale.OnStart = func(d *ScaleStartDetails) { starts = append(starts, d) }
		scale.OnUpdate = func(d *ScaleUpdateDetails) { updates = append(updates, d) }
		transform := vm.NewMatrix4TranslationValues(-350, -250, 0)
		center := sOff(400, 300)

		gesture := sTP(1)
		down1 := gesture.Down(center.Sub(sOff(20, 20)))
		scale.AddPointer(down1.Transformed(transform).(*PointerDownEvent))
		tester.CloseArena(1)
		tester.Route(down1)

		pointer2 := sTP(2)
		down2 := pointer2.Down(center.Add(sOff(30, 30)))
		scale.AddPointer(down2.Transformed(transform).(*PointerDownEvent))
		tester.CloseArena(2)
		tester.Route(down2)
		tester.Route(pointer2.Move(center.Add(sOff(20, 20))))

		if len(updates) != 1 {
			t.Fatalf("%d updates", len(updates))
		}
		if updates[0].LocalFocalPoint != sOff(50, 50) || updates[0].FocalPoint != sOff(400, 300) {
			t.Errorf("update %v %v", updates[0].LocalFocalPoint, updates[0].FocalPoint)
		}
		if len(starts) != 2 {
			t.Fatalf("%d starts", len(starts))
		}
		if starts[0].LocalFocalPoint != sOff(30, 30) || starts[0].FocalPoint != sOff(380, 280) {
			t.Errorf("first start %v %v", starts[0].LocalFocalPoint, starts[0].FocalPoint)
		}
		if starts[1].LocalFocalPoint != sOff(50, 50) || starts[1].FocalPoint != sOff(400, 300) {
			t.Errorf("last start %v %v", starts[1].LocalFocalPoint, starts[1].FocalPoint)
		}
		tester.Route(gesture.Up())
		tester.Route(pointer2.Up())
		scale.Dispose()
	})
}

// Diagnostics of the details (Dart's debugFillProperties order and
// formatting; checked exactly against Flutter by the difftest).
func TestScaleDetailsToString(t *testing.T) {
	d := NewScaleEndDetails()
	if got := d.String(); got != "ScaleEndDetails#"+got[len("ScaleEndDetails#"):] {
		t.Errorf("%q", got)
	}
	u := NewScaleUpdateDetails()
	if u.Scale != 1 || u.HorizontalScale != 1 || u.VerticalScale != 1 || u.LocalFocalPoint != u.FocalPoint {
		t.Errorf("defaults %+v", u)
	}
	s := NewScaleStartDetails(ScaleStartDetailsOptions{FocalPoint: sOff(1, 2)})
	if s.LocalFocalPoint != sOff(1, 2) || s.Kind != nil || s.SourceTimeStamp != nil {
		t.Errorf("start defaults %+v", s)
	}
	defer func() {
		if recover() == nil {
			t.Error("negative scale accepted")
		}
	}()
	neg := -1.0
	NewScaleUpdateDetails(ScaleUpdateDetailsOptions{Scale: &neg})
}

func sPZ(p *flutter_test.TestPointer, location, pan ui.Offset, scale *float64, rotation float64) *PointerPanZoomUpdateEvent {
	return p.PanZoomUpdate(location, flutter_test.TestPointerPanZoomUpdateOptions{Pan: pan, Scale: scale, Rotation: rotation})
}

// scaleDragLog is the competition tests' log of a scale and a horizontal
// drag.
func scaleDragLog(t *testing.T, opts ...ScaleGestureRecognizerOptions) (*ScaleGestureRecognizer, *HorizontalDragGestureRecognizer, *[]string) {
	log := &[]string{}
	add := func(s string) { *log = append(*log, s) }
	scale := NewScaleGestureRecognizer(opts...)
	drag := NewHorizontalDragGestureRecognizer()
	scale.OnStart = func(*ScaleStartDetails) { add("scale-start") }
	scale.OnUpdate = func(*ScaleUpdateDetails) { add("scale-update") }
	scale.OnEnd = func(*ScaleEndDetails) { add("scale-end") }
	drag.OnStart = func(*DragStartDetails) { add("drag-start") }
	drag.OnEnd = func(*DragEndDetails) { add("drag-end") }
	t.Cleanup(scale.Dispose)
	t.Cleanup(drag.Dispose)
	return scale, drag, log
}

func sLog(t *testing.T, log *[]string, want ...string) {
	t.Helper()
	got := append([]string{}, *log...)
	if want == nil {
		want = []string{}
	}
	if len(got) != len(want) {
		t.Fatalf("log %q, want %q", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("log %q, want %q", got, want)
		}
	}
	*log = (*log)[:0]
}

func TestScaleCompetesWithDrag(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale, drag, log := scaleDragLog(t)
		pointer1 := sTP(1)
		down := pointer1.Down(sOff(10, 10))
		scale.AddPointer(down)
		drag.AddPointer(down)
		tester.CloseArena(1)
		sLog(t, log)
		tester.Route(down)
		sLog(t, log)
		tester.Route(pointer1.Move(sOff(10, 50)))
		sLog(t, log, "scale-start", "scale-update")
		pointer2 := sTP(2)
		down2 := pointer2.Down(sOff(10, 20))
		scale.AddPointer(down2)
		drag.AddPointer(down2)
		tester.CloseArena(2)
		sLog(t, log)
		tester.Route(down2)
		sLog(t, log, "scale-end")
		tester.Route(pointer2.Move(sOff(30, 20)))
		sLog(t, log, "scale-start", "scale-update")
		tester.Route(pointer1.Up())
		sLog(t, log, "scale-end")
		tester.Route(pointer2.Up())
		sLog(t, log)
		pointer3 := sTP(3)
		down3 := pointer3.Down(sOff(30, 30))
		scale.AddPointer(down3)
		drag.AddPointer(down3)
		tester.CloseArena(3)
		tester.Route(down3)
		sLog(t, log)
		tester.Route(pointer3.Move(sOff(100, 30)))
		sLog(t, log, "scale-start", "scale-update")
		tester.Route(pointer3.Up())
		sLog(t, log, "scale-end")
	})
}

func TestScaleRecognizesPointerPanZoomEvents(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		t.Cleanup(scale.Dispose)
		drag := NewHorizontalDragGestureRecognizer()
		t.Cleanup(drag.Dispose)
		p := sProbe(scale)
		pointer1 := sTP(2, ui.PointerDeviceKindTrackpad)
		start := pointer1.PanZoomStart(sOff(0, 0))
		scale.AddPointerPanZoom(start)
		drag.AddPointerPanZoom(start)
		tester.CloseArena(2)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		sBool(t, "didEndScale", p.didEnd, false)
		tester.Route(start)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)

		tester.Route(sPZ(pointer1, sOff(0, 0), sOff(20, 30), nil, 0))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(20, 30))
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		two := 2.0
		tester.Route(sPZ(pointer1, sOff(0, 0), sOff(20, 30), &two, 0))
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 2.0)
		sEq(t, "hScale", p.hScale, 2.0)
		sEq(t, "vScale", p.vScale, 2.0)
		sEq(t, "delta", p.delta, sOff(0, 0))
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		tester.Route(sPZ(pointer1, sOff(0, 0), sOff(20, 30), nil, 0))
		sEq(t, "focal", p.focal, sOff(20, 30))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "hScale", p.hScale, 1.0)
		sEq(t, "vScale", p.vScale, 1.0)
		sEq(t, "delta", p.delta, sOff(0, 0))
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		tester.Route(pointer1.PanZoomEnd())
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		sBool(t, "didEndScale", p.didEnd, true)
	})
}

func TestScalePointerPanZoomsWorkAlongsideTouches(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale := NewScaleGestureRecognizer()
		t.Cleanup(scale.Dispose)
		drag := NewHorizontalDragGestureRecognizer()
		t.Cleanup(drag.Dispose)
		p := sProbe(scale)
		touchPointer1, touchPointer2 := sTP(2), sTP(3)
		panZoomPointer := sTP(4, ui.PointerDeviceKindTrackpad)
		panZoomStart := panZoomPointer.PanZoomStart(sOff(0, 0))
		scale.AddPointerPanZoom(panZoomStart)
		drag.AddPointerPanZoom(panZoomStart)
		tester.CloseArena(4)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		tester.Route(panZoomStart)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)

		tester.Route(sPZ(panZoomPointer, sOff(0, 0), sOff(40, 40), nil, 0))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(40, 40))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(40, 40))
		sEq(t, "pointerCount", p.pointerCount, 2)
		p.clearUpdate()

		touchStart1 := touchPointer1.Down(sOff(40, 40))
		scale.AddPointer(touchStart1)
		drag.AddPointer(touchStart1)
		tester.CloseArena(2)
		tester.Route(touchStart1)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false

		tester.Route(touchPointer1.Move(sOff(10, 10)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(25, 25))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(-15, -15))
		sEq(t, "pointerCount", p.pointerCount, 3)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		touchStart2 := touchPointer2.Down(sOff(10, 40))
		scale.AddPointer(touchStart2)
		drag.AddPointer(touchStart2)
		tester.CloseArena(3)
		tester.Route(touchStart2)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false

		tester.Route(touchPointer2.Move(sOff(40, 40)))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(30, 30))
		sEq(t, "scale", p.scale, math.Sqrt(2))
		sEq(t, "hScale", p.hScale, 1.0)
		sEq(t, "vScale", p.vScale, 1.0)
		sEq(t, "delta", p.delta, sOff(10, 0))
		sEq(t, "rotation", p.rotation, -math.Pi/4)
		sEq(t, "pointerCount", p.pointerCount, 4)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		sqrt2 := math.Sqrt(2)
		tester.Route(sPZ(panZoomPointer, sOff(0, 0), sOff(40, 40), &sqrt2, math.Pi/3))
		sBool(t, "didStartScale", p.didStart, false)
		sEq(t, "focal", p.focal, sOff(30, 30))
		sClose(t, "scale", p.scale, 2, 0.0001)
		sEq(t, "hScale", p.hScale, math.Sqrt(2))
		sEq(t, "vScale", p.vScale, math.Sqrt(2))
		sEq(t, "delta", p.delta, sOff(0, 0))
		sClose(t, "rotation", p.rotation, math.Pi/12, 0.0001)
		sEq(t, "pointerCount", p.pointerCount, 4)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		tester.Route(sPZ(panZoomPointer, sOff(15, 15), sOff(55, 55), &sqrt2, math.Pi/3))
		sBool(t, "didStartScale", p.didStart, false)
		sEq(t, "focal", p.focal, sOff(40, 40))
		sClose(t, "scale", p.scale, 2, 0.0001)
		sEq(t, "delta", p.delta, sOff(10, 10))
		sClose(t, "rotation", p.rotation, math.Pi/12, 0.0001)
		sEq(t, "pointerCount", p.pointerCount, 4)
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		tester.Route(panZoomPointer.PanZoomEnd())
		sNil(t, "focal", p.focal)
		sBool(t, "didEndScale", p.didEnd, true)
		p.didEnd = false
		p.expectNoUpdate(t)
		sBool(t, "didStartScale", p.didStart, false)
		tester.Route(touchPointer1.Up())
		sNil(t, "focal", p.focal)
		sBool(t, "didEndScale", p.didEnd, false)
		p.expectNoUpdate(t)
		sBool(t, "didStartScale", p.didStart, false)
		tester.Route(touchPointer2.Up())
		sBool(t, "didEndScale", p.didEnd, false)
		p.expectNoUpdate(t)
		sBool(t, "didStartScale", p.didStart, false)
	})
}

func TestScaleCompetesWithDragForTrackpadGesture(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		scale, drag, log := scaleDragLog(t)
		pointer1 := sTP(2, ui.PointerDeviceKindTrackpad)
		down := pointer1.PanZoomStart(sOff(10, 10))
		scale.AddPointerPanZoom(down)
		drag.AddPointerPanZoom(down)
		tester.CloseArena(2)
		sLog(t, log)
		tester.Route(down)
		sLog(t, log)
		tester.Route(sPZ(pointer1, sOff(10, 10), sOff(10, 40), nil, 0))
		sLog(t, log, "scale-start", "scale-update")
		pointer2 := sTP(3, ui.PointerDeviceKindTrackpad)
		down2 := pointer2.PanZoomStart(sOff(10, 20))
		scale.AddPointerPanZoom(down2)
		drag.AddPointerPanZoom(down2)
		tester.CloseArena(3)
		sLog(t, log)
		tester.Route(down2)
		sLog(t, log, "scale-end")
		tester.Route(sPZ(pointer2, sOff(10, 20), sOff(20, 0), nil, 0))
		sLog(t, log, "scale-start", "scale-update")
		tester.Route(pointer1.PanZoomEnd())
		sLog(t, log, "scale-end")
		tester.Route(pointer2.PanZoomEnd())
		sLog(t, log)
		pointer3 := sTP(4, ui.PointerDeviceKindTrackpad)
		down3 := pointer3.PanZoomStart(sOff(30, 30))
		scale.AddPointerPanZoom(down3)
		drag.AddPointerPanZoom(down3)
		tester.CloseArena(4)
		tester.Route(down3)
		sLog(t, log)
		tester.Route(sPZ(pointer3, sOff(30, 30), sOff(70, 0), nil, 0))
		sLog(t, log, "scale-start", "scale-update")
		tester.Route(pointer3.PanZoomEnd())
		sLog(t, log, "scale-end")
	})
}

func TestScalePanZoomHandlesDragStartBehaviorStart(t *testing.T) {
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		start := DragStartBehaviorStart
		scale := NewScaleGestureRecognizer(ScaleGestureRecognizerOptions{DragStartBehavior: &start})
		t.Cleanup(scale.Dispose)
		drag := NewHorizontalDragGestureRecognizer()
		t.Cleanup(drag.Dispose)
		p := sProbe(scale)
		pointer1 := sTP(2, ui.PointerDeviceKindTrackpad)
		pzStart := pointer1.PanZoomStart(sOff(0, 0))
		scale.AddPointerPanZoom(pzStart)
		drag.AddPointerPanZoom(pzStart)
		tester.CloseArena(2)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		sBool(t, "didEndScale", p.didEnd, false)
		tester.Route(pzStart)
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)

		s11 := 1.1
		tester.Route(sPZ(pointer1, sOff(0, 0), sOff(0, 0), &s11, 1))
		sBool(t, "didStartScale", p.didStart, true)
		p.didStart = false
		sEq(t, "focal", p.focal, sOff(0, 0))
		sEq(t, "scale", p.scale, 1.0)
		sEq(t, "delta", p.delta, sOff(0, 0))
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		s121 := 1.21
		tester.Route(sPZ(pointer1, sOff(0, 0), sOff(0, 0), &s121, 1.5))
		sEq(t, "focal", p.focal, sOff(0, 0))
		sClose(t, "scale", p.scale, 1.1, 0.0001)
		sClose(t, "hScale", p.hScale, 1.1, 0.0001)
		sClose(t, "vScale", p.vScale, 1.1, 0.0001)
		sEq(t, "rotation", p.rotation, 0.5)
		sEq(t, "delta", p.delta, sOff(0, 0))
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		s099 := 0.99
		tester.Route(sPZ(pointer1, sOff(0, 0), sOff(0, 0), &s099, 1.0))
		sEq(t, "focal", p.focal, sOff(0, 0))
		sClose(t, "scale", p.scale, 0.9, 0.0001)
		sClose(t, "hScale", p.hScale, 0.9, 0.0001)
		sClose(t, "vScale", p.vScale, 0.9, 0.0001)
		sEq(t, "rotation", p.rotation, 0.0)
		sEq(t, "delta", p.delta, sOff(0, 0))
		p.clearUpdate()
		sBool(t, "didEndScale", p.didEnd, false)

		tester.Route(pointer1.PanZoomEnd())
		sBool(t, "didStartScale", p.didStart, false)
		p.expectNoUpdate(t)
		sBool(t, "didEndScale", p.didEnd, true)
	})
}
