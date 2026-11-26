package gestures_test

// Port of packages/flutter/test/gestures/force_press_test.dart (Flutter
// 3.50.1): all 9 tests.

import (
	"math"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/animation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// Device specific constants that represent those from the iPhone X.
const (
	fpPressureMin = 0.0
	fpPressureMax = 6.66
)

type forceCounts struct{ started, peaked, updated, ended int }

func countingForce(t *testing.T, c *forceCounts, opts ...ForcePressGestureRecognizerOptions) *ForcePressGestureRecognizer {
	force := NewForcePressGestureRecognizer(opts...)
	t.Cleanup(force.Dispose)
	force.OnStart = func(*ForcePressDetails) { c.started++ }
	force.OnPeak = func(*ForcePressDetails) { c.peaked++ }
	force.OnUpdate = func(*ForcePressDetails) { c.updated++ }
	force.OnEnd = func(*ForcePressDetails) { c.ended++ }
	return force
}

// fpMove is PointerMoveEvent(pointer: 1, position: (10, 10), pressure,
// pressureMin, pressureMax); a nil pressure is the default (1.0).
func fpMove(pressure *float64, pressureMax ...float64) *PointerMoveEvent {
	max := fpPressureMax
	if len(pressureMax) > 0 {
		max = pressureMax[0]
	}
	return peMove(1, 10, 10, peOpt{Pressure: pressure, PressureMin: tapF(fpPressureMin), PressureMax: tapF(max)})
}

// fpDown sets up TestPointer and the down event as the tests do.
func fpDown(pressure *float64, pressureMax float64) (*flutter_test.TestPointer, *PointerDownEvent) {
	pointer := flutter_test.NewTestPointer()
	down := peDown(1, 10, 10, peOpt{Pressure: pressure, PressureMin: tapF(fpPressureMin), PressureMax: tapF(pressureMax)})
	pointer.SetDownInfo(down, ui.NewOffset(10, 10), nil)
	return pointer, down
}

func expectCounts(t *testing.T, c forceCounts, started, peaked, updated, ended int) {
	t.Helper()
	expectDeep(t, c, forceCounts{started, peaked, updated, ended})
}

func TestForcePressCanBeRecognized(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		var startGlobalPosition *ui.Offset
		force := countingForce(t, &c)
		force.OnStart = func(d *ForcePressDetails) {
			p := d.GlobalPosition()
			startGlobalPosition = &p
			c.started++
		}
		pointer, down := fpDown(tapF(0), fpPressureMax)
		force.AddPointer(down)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)

		tester.Route(fpMove(tapF(2.5)))
		// We have not hit the start pressure, so no events should be true.
		expectCounts(t, c, 0, 0, 0, 0)

		tester.Route(fpMove(tapF(2.8)))
		// We have just hit the start pressure: start and one update.
		expectCounts(t, c, 1, 0, 1, 0)

		tester.Route(fpMove(tapF(3.3)))
		tester.Route(fpMove(tapF(4.0)))
		tester.Route(fpMove(tapF(5.0)))
		tester.Route(fpMove(nil))
		expectCounts(t, c, 1, 0, 5, 0)
		if startGlobalPosition == nil || *startGlobalPosition != ui.NewOffset(10, 10) {
			t.Fatalf("startGlobalPosition %v", startGlobalPosition)
		}

		tester.Route(fpMove(tapF(6.0)))
		// We have exceeded the peak pressure.
		expectCounts(t, c, 1, 1, 6, 0)

		tester.Route(fpMove(tapF(3.3)))
		tester.Route(fpMove(tapF(4.0)))
		tester.Route(fpMove(tapF(5.0)))
		tester.Route(fpMove(nil))
		// Update is still called.
		expectCounts(t, c, 1, 1, 10, 0)

		tester.Route(pointer.Up())
		expectCounts(t, c, 1, 1, 10, 1)
	})
}

func TestForcePressInvalidPressureRangesAreNotRecognized(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		for _, pressureMax := range []float64{0, 1, -1, 0.5} {
			var c forceCounts
			force := countingForce(t, &c)
			pointer, down := fpDown(tapF(0), pressureMax)
			force.AddPointer(down)
			tester.CloseArena(1)
			expectCounts(t, c, 0, 0, 0, 0)

			tester.Route(fpMove(tapF(10), pressureMax))
			// Regardless of current pressure, this recognizer shouldn't
			// participate or trigger any callbacks.
			expectCounts(t, c, 0, 0, 0, 0)

			tester.Route(pointer.Up())
			expectCounts(t, c, 0, 0, 0, 0)
		}
	})
}

func TestForcePressMinimumPressureNotReached(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		force := countingForce(t, &c)
		pointer, down := fpDown(tapF(0), fpPressureMax)
		force.AddPointer(down)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)
		tester.Route(fpMove(tapF(2.5)))
		expectCounts(t, c, 0, 0, 0, 0)
		tester.Route(pointer.Up())
		expectCounts(t, c, 0, 0, 0, 0)
	})
}

func TestForcePressCustomInterpolation(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		var startGlobalPosition *ui.Offset
		interpolateWithEasing := func(min, max, t float64) float64 {
			lerp := (t - min) / (max - min)
			return animation.Curves.EaseIn.Transform(lerp)
		}
		force := countingForce(t, &c, ForcePressGestureRecognizerOptions{Interpolation: interpolateWithEasing})
		force.OnStart = func(d *ForcePressDetails) {
			p := d.GlobalPosition()
			startGlobalPosition = &p
			c.started++
		}
		pointer, down := fpDown(tapF(0), fpPressureMax)
		force.AddPointer(down)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)

		tester.Route(fpMove(tapF(2.5)))
		expectCounts(t, c, 0, 0, 0, 0)
		tester.Route(fpMove(tapF(2.8)))
		expectCounts(t, c, 0, 0, 0, 0)
		tester.Route(fpMove(tapF(3.3)))
		expectDeep(t, c.started, 0)
		tester.Route(fpMove(tapF(4.0)))
		expectDeep(t, c.started, 1)
		tester.Route(fpMove(tapF(5.0)))
		tester.Route(fpMove(nil))
		expectCounts(t, c, 1, 0, 3, 0)
		if startGlobalPosition == nil || *startGlobalPosition != ui.NewOffset(10, 10) {
			t.Fatalf("startGlobalPosition %v", startGlobalPosition)
		}

		tester.Route(fpMove(tapF(6.0)))
		expectCounts(t, c, 1, 0, 4, 0)

		tester.Route(fpMove(tapF(3.3)))
		tester.Route(fpMove(tapF(4.0)))
		tester.Route(fpMove(tapF(6.5)))
		tester.Route(fpMove(nil))
		expectCounts(t, c, 1, 1, 8, 0)

		tester.Route(pointer.Up())
		expectCounts(t, c, 1, 1, 8, 1)
	})
}

func TestForcePressPressureOutsideDeviceRangeGivesNoError(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		force := countingForce(t, &c)
		pointer, down := fpDown(tapF(0), fpPressureMax)
		force.AddPointer(down)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)
		tester.Route(fpMove(tapF(2.5)))
		expectCounts(t, c, 0, 0, 0, 0)
		// If the pressure greater than the max pressure were not handled
		// correctly, this move event would throw an error.
		tester.Route(fpMove(tapF(8.0)))
		tester.Route(pointer.Up())
		expectCounts(t, c, 1, 1, 1, 1)
	})
}

func TestForcePressPressureOfNaNGivesNoError(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		force := countingForce(t, &c)
		pointer, down := fpDown(tapF(0), fpPressureMax)
		force.AddPointer(down)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)
		tester.Route(fpMove(tapF(2.5)))
		expectCounts(t, c, 0, 0, 0, 0)
		tester.Route(fpMove(tapF(6.0)))
		expectDeep(t, c.peaked, 1)
		tester.Route(fpMove(tapF(math.NaN())))
		tester.Route(pointer.Up())
		expectCounts(t, c, 1, 1, 1, 1)
	})
}

// forceAndPan is the tests' ForcePressGestureRecognizer and competing
// PanGestureRecognizer, and the pointer set down at (10, 10).
func forceAndPan(t *testing.T, c *forceCounts, didStartPan *int) *flutter_test.TestPointer {
	drag := NewPanGestureRecognizer()
	t.Cleanup(drag.Dispose)
	force := countingForce(t, c)
	drag.OnStart = func(*DragStartDetails) { *didStartPan++ }
	pointer, down := fpDown(nil, fpPressureMax)
	force.AddPointer(down)
	drag.AddPointer(down)
	return pointer
}

func TestForcePressShouldRecognizeDragAndNotForceTouchWithDragRecognizer(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		didStartPan := 0
		pointer := forceAndPan(t, &c, &didStartPan)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 0)
		tester.Route(pointer.Move(ui.NewOffset(30, 30)))
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 1)
		tester.Route(fpMove(tapF(2.5)))
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 1)
		tester.Route(fpMove(tapF(4.0)))
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 1)
		tester.Route(pointer.Up())
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 1)
	})
}

func TestForcePressShouldNotCallEndedIfNeverAccepted(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		didStartPan := 0
		pointer := forceAndPan(t, &c, &didStartPan)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 0)
		tester.Route(pointer.Up())
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 0)
	})
}

func TestForcePressShouldCallStartOnceWithCompetingRecognizer(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var c forceCounts
		didStartPan := 0
		pointer := forceAndPan(t, &c, &didStartPan)
		tester.CloseArena(1)
		expectCounts(t, c, 0, 0, 0, 0)
		expectDeep(t, didStartPan, 0)
		tester.Route(fpMove(tapF(3.0)))
		expectCounts(t, c, 1, 0, 1, 0)
		expectDeep(t, didStartPan, 0)
		tester.Route(pointer.Up())
		expectCounts(t, c, 1, 0, 1, 1)
		expectDeep(t, didStartPan, 0)
	})
}
