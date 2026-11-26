// Port of packages/flutter/test/gestures/velocity_tracker_test.dart (Flutter
// 3.50.1): all 11 tests. testWidgets' fake clock (the binding's sampling
// clock) is a fake SamplingClock installed with DebugSetSamplingClock;
// tester.pumpAndSettle() with nothing scheduled advances it by one 100 ms
// pump. velocity_tracker_data.dart is velocity_tracker_data_test.go.
package gestures_test

import (
	"math"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type velocitySample struct {
	kind    string
	ms      int
	pointer int
	x, y    float64
}

func (s velocitySample) event() PointerEvent {
	ts := time.Duration(s.ms) * time.Millisecond
	pos := ui.NewOffset(s.x, s.y)
	switch s.kind {
	case "down":
		return NewPointerDownEvent(PointerDownEventOptions{TimeStamp: ts, Pointer: s.pointer, Position: pos})
	case "move":
		return NewPointerMoveEvent(PointerMoveEventOptions{TimeStamp: ts, Pointer: s.pointer, Position: pos})
	}
	return NewPointerUpEvent(PointerUpEventOptions{TimeStamp: ts, Pointer: s.pointer, Position: pos})
}

// fakeSamplingClock is flutter_test's sampling clock on fake time.
type fakeSamplingClock struct{ now time.Duration }

func (c *fakeSamplingClock) Now() time.Time { return time.Unix(0, 0).Add(c.now) }
func (c *fakeSamplingClock) Stopwatch() *dart.Stopwatch {
	return dart.NewStopwatchWithClock(func() time.Duration { return c.now })
}

func withFakeClock(t *testing.T) *fakeSamplingClock {
	clock := &fakeSamplingClock{}
	t.Cleanup(DebugSetSamplingClock(func() SamplingClock { return clock }))
	return clock
}

func withinTolerance(actual, expected float64) bool {
	const kTolerance = 0.001 // Within .1% of expected value
	diff := (actual - expected) / expected
	return math.Abs(diff) < kTolerance
}

func checkVelocity(actual Velocity, expected ui.Offset) bool {
	return withinTolerance(actual.PixelsPerSecond.Dx, expected.Dx) && withinTolerance(actual.PixelsPerSecond.Dy, expected.Dy)
}

func feed(tracker *VelocityTracker, samples []velocitySample, onUp func(Velocity)) {
	for _, s := range samples {
		event := s.event()
		switch event.(type) {
		case *PointerDownEvent, *PointerMoveEvent:
			tracker.AddPosition(event.TimeStamp(), event.Position())
		case *PointerUpEvent:
			onUp(tracker.GetVelocity())
		}
	}
}

func TestVelocityTrackerExpectedResults(t *testing.T) {
	withFakeClock(t)
	expected := []ui.Offset{
		ui.NewOffset(219.59280094228163, 1304.701682306001),
		ui.NewOffset(355.71046950050845, 967.2112857054104),
		ui.NewOffset(12.657970884022308, -36.90447839251946),
		ui.NewOffset(714.1399654786744, -2561.534447931869),
		ui.NewOffset(-19.668121066218564, -2910.105747052462),
		ui.NewOffset(646.8690114934209, 2976.977762577527),
		ui.NewOffset(396.6988447819592, 2106.225572911095),
		ui.NewOffset(298.31594440044495, -3660.8315955215294),
		ui.NewOffset(-1.7334232785165882, -3288.13174127454),
		ui.NewOffset(384.6361280392334, -2645.6612524779835),
		ui.NewOffset(176.37900397918557, 2711.2542876273264),
		ui.NewOffset(396.9328560260098, 4280.651578291764),
		ui.NewOffset(-71.51939428321249, 3716.7385187526947),
	}
	tracker := NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch)
	i := 0
	feed(tracker, velocityEventDataSamples, func(v Velocity) {
		if !checkVelocity(v, expected[i]) {
			t.Errorf("up %d: %v, want %v", i, v.PixelsPerSecond, expected[i])
		}
		i++
	})
	if i != len(expected) {
		t.Fatalf("%d up events", i)
	}
}

func TestVelocityControl(t *testing.T) {
	velocity1 := Velocity{PixelsPerSecond: ui.NewOffset(7.0, 0.0)}
	velocity2 := Velocity{PixelsPerSecond: ui.NewOffset(12.0, 0.0)}
	if velocity1 != (Velocity{PixelsPerSecond: ui.NewOffset(7.0, 0.0)}) || velocity1 == velocity2 {
		t.Fatal("==")
	}
	if velocity2.Sub(velocity1) != (Velocity{PixelsPerSecond: ui.NewOffset(5.0, 0.0)}) ||
		velocity1.Neg().PixelsPerSecond != ui.NewOffset(-7.0, 0.0) ||
		velocity1.Add(velocity2) != (Velocity{PixelsPerSecond: ui.NewOffset(19.0, 0.0)}) {
		t.Fatal("operators")
	}
	expectOneLine(t, velocity1.String())
	if velocity1.String() != "Velocity(7.0, 0.0)" {
		t.Errorf("toString %q", velocity1.String())
	}
}

func TestInterruptedVelocityEstimation(t *testing.T) {
	withFakeClock(t)
	tracker := NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch)
	feed(tracker, interruptedVelocityEventDataSamples, func(v Velocity) {
		if !checkVelocity(v, ui.NewOffset(649.5, 3890.3)) {
			t.Errorf("velocity %v", v)
		}
	})
}

func TestNoDataVelocityEstimation(t *testing.T) {
	withFakeClock(t)
	if NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch).GetVelocity() != VelocityZero {
		t.Fatal("not zero")
	}
}

func recoverMessage(f func()) (message string) {
	defer func() {
		if r := recover(); r != nil {
			message = dart.ToString(r)
		}
	}()
	f()
	return ""
}

func TestIOSTrackerThrowsWhenNoPoints(t *testing.T) {
	withFakeClock(t)
	tracker := NewIOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch)
	if m := recoverMessage(func() { tracker.GetVelocity() }); !strings.Contains(m, "at least 1 point") {
		t.Fatalf("message %q", m)
	}
}

func TestIOSTrackerThrowsWhenNewPointPrecedesPrevious(t *testing.T) {
	withFakeClock(t)
	tracker := NewIOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch)
	tracker.AddPosition(time.Hour, ui.OffsetZero)
	m := recoverMessage(func() {
		tracker.GetVelocity()
		tracker.AddPosition(time.Second, ui.OffsetZero)
	})
	if !strings.Contains(m, "has a smaller timestamp") {
		t.Fatalf("message %q", m)
	}
}

func TestIOSEstimateDoesNotThrowWithMoreThanOnePoint(t *testing.T) {
	withFakeClock(t)
	tracker := NewIOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch)
	position, tm := ui.OffsetZero, time.Duration(0)
	for i := 0; i < 5; i++ {
		position = position.Add(ui.NewOffset(0, -1))
		tm += time.Second
		tracker.AddPosition(tm, position)
		if m := recoverMessage(func() { tracker.GetVelocity() }); m != "" {
			t.Fatalf("panicked: %s", m)
		}
	}
}

func TestIOSConsistentVelocityEstimates(t *testing.T) {
	withFakeClock(t)
	tracker := NewIOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch)
	position, tm := ui.OffsetZero, time.Duration(0)
	positionDelta := ui.NewOffset(0, -1)
	for i := 0; i < 10; i++ {
		position = position.Add(positionDelta)
		tm += time.Second
		tracker.AddPosition(tm, position)
		if i >= 3 {
			if got := tracker.GetVelocity().PixelsPerSecond; got != positionDelta {
				t.Fatalf("i=%d: %v", i, got)
			}
		}
	}
}

func TestAssumeZeroVelocityWithoutRecentSamples(t *testing.T) {
	for name, tracker := range map[string]func() *VelocityTracker{
		"base": func() *VelocityTracker { return NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch) },
		"IOS": func() *VelocityTracker {
			return &NewIOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch).VelocityTracker
		},
		"MacOS": func() *VelocityTracker {
			return &NewMacOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch).VelocityTracker
		},
	} {
		clock := withFakeClock(t)
		tr := tracker()
		position, tm := ui.OffsetZero, time.Duration(0)
		for i := 0; i < 10; i++ {
			position = position.Add(ui.NewOffset(0, -1))
			tm += time.Second
			tr.AddPosition(tm, position)
		}
		if name != "base" && tr.GetVelocity().PixelsPerSecond == ui.OffsetZero {
			t.Fatalf("%s: zero before the pump (the test would be vacuous)", name)
		}
		clock.now += 100 * time.Millisecond // tester.pumpAndSettle()
		if got := tr.GetVelocity().PixelsPerSecond; got != ui.OffsetZero {
			t.Errorf("%s: %v", name, got)
		}
	}
}

// Not in Flutter's tests: the boundary of "pointer stopped" (more than 40
// whole milliseconds since the last sample).
func TestVelocityTrackerStoppedBoundary(t *testing.T) {
	clock := withFakeClock(t)
	tr := NewIOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch)
	tr.AddPosition(0, ui.OffsetZero)
	tr.AddPosition(10*time.Millisecond, ui.NewOffset(0, 10))
	clock.now += 40*time.Millisecond + 999*time.Microsecond
	if tr.GetVelocity().PixelsPerSecond == ui.OffsetZero {
		t.Fatal("stopped at 40.999 ms")
	}
	clock.now += time.Microsecond
	if tr.GetVelocity().PixelsPerSecond != ui.OffsetZero {
		t.Fatal("not stopped at 41 ms")
	}
}
