package gestures_test

// Ports of packages/flutter/test/gestures/transformed_tap_test.dart,
// transformed_double_tap_test.dart and transformed_long_press_test.dart
// (Flutter 3.50.1).
//
// Every test is skipped: they pump widgets (a GestureDetector inside a
// Transform), which needs widgets (GestureDetector/Listener not ported).
// The recognizers' transformed-event handling is covered by the unit tests
// of events and by the tap difftest's local positions.

import "testing"

const needsWidgets = "needs widgets (GestureDetector/Listener not ported)"

func TestTransformedTap(t *testing.T) {
	for _, name := range []string{
		"gets local coordinates",
		"kTouchSlop is evaluated in the global coordinate space when scaled up",
		"kTouchSlop is evaluated in the global coordinate space when scaled down",
	} {
		t.Run(name, func(t *testing.T) { t.Skip(needsWidgets) })
	}
}

func TestTransformedDoubleTap(t *testing.T) {
	for _, name := range []string{
		"kTouchSlop is evaluated in the global coordinate space when scaled up",
		"kTouchSlop is evaluated in the global coordinate space when scaled down",
	} {
		t.Run(name, func(t *testing.T) { t.Skip(needsWidgets) })
	}
}

func TestTransformedLongPress(t *testing.T) {
	for _, name := range []string{"gets local coordinates", "scaled up", "scaled down"} {
		t.Run(name, func(t *testing.T) { t.Skip(needsWidgets) })
	}
}
