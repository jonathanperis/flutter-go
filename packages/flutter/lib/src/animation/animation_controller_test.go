package animation

// Port of packages/flutter/test/animation/animation_controller_test.dart
// (Flutter 3.50.1): all 48 tests.
//
// Adaptations: debugSemanticsDisableAnimations is the
// SemanticsDisableAnimations hook (the semantics library is not ported);
// `expect(controller.forward, throwsFlutterError)` checks for a
// *foundation.FlutterError panic; memoryEvents/areCreateAndDispose listen
// to FlutterMemoryAllocations directly.

import (
	"regexp"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// memoryEvents runs f and returns the memory-allocation events of objects
// for which match is true.
func memoryEvents(f func(), match func(any) bool) []foundation.ObjectEvent {
	var events []foundation.ObjectEvent
	listener := func(e foundation.ObjectEvent) {
		if match(e.Object()) {
			events = append(events, e)
		}
	}
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	f()
	return events
}

// areCreateAndDispose is leak_tracker's matcher: one creation, then one
// disposal.
func areCreateAndDispose(t *testing.T, events []foundation.ObjectEvent) {
	t.Helper()
	if len(events) != 2 {
		t.Fatalf("events %v", events)
	}
	if _, ok := events[0].(*foundation.ObjectCreated); !ok {
		t.Fatalf("first event %T", events[0])
	}
	if _, ok := events[1].(*foundation.ObjectDisposed); !ok {
		t.Fatalf("second event %T", events[1])
	}
}

func throwsFlutterError(t *testing.T, f func()) *foundation.FlutterError {
	t.Helper()
	r := panics(f)
	e, ok := r.(*foundation.FlutterError)
	if !ok {
		t.Fatalf("want a FlutterError, got %T %v", r, r)
	}
	return e
}

var hashCodes = regexp.MustCompile(`#[0-9a-f]{5}`)

func equalsIgnoringHashCodes(t *testing.T, got, want string) {
	t.Helper()
	if g := hashCodes.ReplaceAllString(got, "#00000"); g != want {
		t.Fatalf("got\n%s\nwant\n%s", g, want)
	}
}

func newController(opts ...AnimationControllerOptions) *AnimationController {
	return NewAnimationController(vsync, opts...)
}

func ms100() AnimationControllerOptions { return AnimationControllerOptions{Duration: dur(100 * ms)} }

func TestAnimationControllerDispatchesMemoryEvents(t *testing.T) {
	setUp(t)
	events := memoryEvents(func() { newController(ms100()).Dispose() }, func(o any) bool {
		_, ok := o.(*AnimationController)
		return ok
	})
	areCreateAndDispose(t, events)
}

func TestCanStartTheAnimationWhenResyncIsCalled(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	err := panics(func() {
		controller.Forward()
		controller.Resync(testVSync{})
		tick(time.Second)
	})
	if err != nil {
		t.Fatal(err)
	}
	controller.Dispose()
}

func TestCanSetValueDuringStatusCallback(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	didComplete, didDismiss := false, false
	controller.AddStatusListener(func(status AnimationStatus) {
		if status == AnimationStatusCompleted {
			didComplete = true
			controller.SetValue(0.0)
			controller.Forward()
		} else if status == AnimationStatusDismissed {
			didDismiss = true
			controller.SetValue(0.0)
			controller.Forward()
		}
	})
	controller.Forward()
	expect(t, didComplete, false)
	expect(t, didDismiss, false)
	tick(time.Second)
	expect(t, didComplete, false)
	expect(t, didDismiss, false)
	tick(2 * time.Second)
	expect(t, didComplete, true)
	expect(t, didDismiss, true)
	controller.Stop()
	controller.Dispose()
}

func TestReceivesStatusCallbacksForForwardAndReverse(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	var valueLog []float64
	var log []AnimationStatus
	controller.AddStatusListener(func(s AnimationStatus) { log = append(log, s) })
	controller.AddListener(func() { valueLog = append(valueLog, controller.Value()) })

	expectStatuses(t, log)
	expect(t, len(valueLog), 0)
	controller.Forward()
	expectStatuses(t, log, AnimationStatusForward)
	expect(t, len(valueLog), 0)
	controller.Reverse()
	expectStatuses(t, log, AnimationStatusForward, AnimationStatusDismissed)
	expect(t, len(valueLog), 0)
	controller.Reverse()
	expectStatuses(t, log, AnimationStatusForward, AnimationStatusDismissed)
	expect(t, len(valueLog), 0)
	log = nil
	controller.Forward()
	expectStatuses(t, log, AnimationStatusForward)
	expect(t, len(valueLog), 0)
	controller.Forward()
	expectStatuses(t, log, AnimationStatusForward)
	expect(t, len(valueLog), 0)
	controller.Reverse()
	log = nil
	for _, s := range []time.Duration{10, 20, 30, 40} {
		tick(s * time.Second)
		expectStatuses(t, log)
		expect(t, len(valueLog), 0)
	}
	controller.Stop()
	controller.Dispose()
}

func TestForwardAndReverseFromValues(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	var valueLog []float64
	var statusLog []AnimationStatus
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	controller.AddListener(func() { valueLog = append(valueLog, controller.Value()) })

	controller.Reverse(ReverseOptions{From: f64(0.2)})
	expectStatuses(t, statusLog, AnimationStatusReverse)
	if len(valueLog) != 1 || valueLog[0] != 0.2 {
		t.Fatalf("valueLog %v", valueLog)
	}
	expect(t, controller.Value(), 0.2)
	statusLog, valueLog = nil, nil

	controller.Forward(ForwardOptions{From: f64(0.0)})
	expectStatuses(t, statusLog, AnimationStatusDismissed, AnimationStatusForward)
	if len(valueLog) != 1 || valueLog[0] != 0.0 {
		t.Fatalf("valueLog %v", valueLog)
	}
	expect(t, controller.Value(), 0.0)
	controller.Dispose()
}

// checkTicks ticks at each time (ms) and checks the value after it (NaN:
// no check).
func checkTicks(t *testing.T, value func() float64, steps ...[2]float64) {
	t.Helper()
	for _, s := range steps {
		tick(time.Duration(s[0] * float64(ms)))
		if s[1] == s[1] && !near(value(), s[1]) {
			t.Fatalf("at %vms: value %v, want %v", s[0], value(), s[1])
		}
	}
}

func TestForwardAndReverseWithDifferentDurations(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), ReverseDuration: dur(50 * ms)})
	controller.Forward()
	checkTicks(t, controller.Value, [2]float64{10, nan()}, [2]float64{30, 0.2}, [2]float64{60, 0.5},
		[2]float64{90, 0.8}, [2]float64{120, 1.0})
	controller.Stop()
	controller.Reverse()
	checkTicks(t, controller.Value, [2]float64{210, nan()}, [2]float64{220, 0.8}, [2]float64{230, 0.6},
		[2]float64{240, 0.4}, [2]float64{260, 0.0})
	controller.Stop()
	controller.Dispose()

	// Swap which duration is longer.
	controller = newController(AnimationControllerOptions{Duration: dur(50 * ms), ReverseDuration: dur(100 * ms)})
	controller.Forward()
	checkTicks(t, controller.Value, [2]float64{10, nan()}, [2]float64{30, 0.4}, [2]float64{60, 1.0},
		[2]float64{90, 1.0})
	controller.Stop()
	controller.Reverse()
	checkTicks(t, controller.Value, [2]float64{210, nan()}, [2]float64{220, 0.9}, [2]float64{230, 0.8},
		[2]float64{240, 0.7}, [2]float64{260, 0.5}, [2]float64{310, 0.0})
	controller.Stop()
	controller.Dispose()
}

func nan() float64 { z := 0.0; return z / z }

func TestToggleWithDifferentDurations(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), ReverseDuration: dur(50 * ms)})
	controller.Toggle()
	checkTicks(t, controller.Value, [2]float64{10, nan()}, [2]float64{30, 0.2}, [2]float64{60, 0.5},
		[2]float64{90, 0.8}, [2]float64{120, 1.0})
	controller.Stop()
	controller.Toggle()
	checkTicks(t, controller.Value, [2]float64{210, nan()}, [2]float64{220, 0.8}, [2]float64{230, 0.6},
		[2]float64{240, 0.4}, [2]float64{260, 0.0})
	controller.Stop()
	controller.Dispose()

	controller = newController(AnimationControllerOptions{Duration: dur(50 * ms), ReverseDuration: dur(100 * ms)})
	controller.Toggle()
	checkTicks(t, controller.Value, [2]float64{10, nan()}, [2]float64{30, 0.4}, [2]float64{60, 1.0},
		[2]float64{90, 1.0})
	controller.Stop()
	controller.Toggle()
	checkTicks(t, controller.Value, [2]float64{210, nan()}, [2]float64{220, 0.9}, [2]float64{230, 0.8},
		[2]float64{240, 0.7}, [2]float64{260, 0.5}, [2]float64{310, 0.0})
	controller.Stop()
	controller.Dispose()
}

func TestToggleActsCorrectlyBasedOnTheAnimationState(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), ReverseDuration: dur(100 * ms)})
	controller.Forward()
	expect(t, controller.Status(), AnimationStatusForward)
	expect(t, controller.IsForwardOrCompleted(), true)
	tick(10 * ms)
	tick(60 * ms)
	if !near(controller.Value(), 0.5) {
		t.Fatal(controller.Value())
	}
	expect(t, controller.IsForwardOrCompleted(), true)
	controller.Toggle()
	tick(10 * ms)
	expect(t, controller.Status(), AnimationStatusReverse)
	expect(t, controller.IsForwardOrCompleted(), false)
	tick(110 * ms)
	if !near(controller.Value(), 0) {
		t.Fatal(controller.Value())
	}
	expect(t, controller.Status(), AnimationStatusDismissed)
	expect(t, controller.IsForwardOrCompleted(), false)
	controller.Dispose()
}

func TestForwardOnlyFromValue(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	var valueLog []float64
	var statusLog []AnimationStatus
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	controller.AddListener(func() { valueLog = append(valueLog, controller.Value()) })
	controller.Forward(ForwardOptions{From: f64(0.2)})
	expectStatuses(t, statusLog, AnimationStatusForward)
	if len(valueLog) != 1 || valueLog[0] != 0.2 {
		t.Fatalf("valueLog %v", valueLog)
	}
	expect(t, controller.Value(), 0.2)
	controller.Dispose()
}

func TestCanFlingToUpperAndLowerBounds(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	controller.Fling()
	tick(time.Second)
	tick(2 * time.Second)
	expect(t, controller.Value(), 1.0)
	controller.Stop()

	largeRangeController := newController(AnimationControllerOptions{Duration: dur(100 * ms),
		LowerBound: f64(-30.0), UpperBound: f64(45.0)})
	largeRangeController.Fling()
	tick(3 * time.Second)
	tick(4 * time.Second)
	expect(t, largeRangeController.Value(), 45.0)
	largeRangeController.Fling(FlingOptions{Velocity: f64(-1.0)})
	tick(5 * time.Second)
	tick(6 * time.Second)
	expect(t, largeRangeController.Value(), -30.0)
	largeRangeController.Stop()
	controller.Dispose()
	largeRangeController.Dispose()
}

func TestCustomSpringDescriptionCanBeApplied(t *testing.T) {
	setUp(t)
	controller := newController()
	customSpringController := newController()
	controller.Fling()
	// Will produce longer and smoother animation than the default.
	spring := physics.NewSpringDescriptionWithDampingRatio(0.01, 10.0, 2.0)
	customSpringController.Fling(FlingOptions{SpringDescription: &spring})
	tick(0)
	tick(50 * ms)
	expect(t, customSpringController.Value() < controller.Value(), true)
	controller.Dispose()
	customSpringController.Dispose()
}

func TestLastElapsedDurationControlTest(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	controller.Forward()
	tick(20 * ms)
	tick(30 * ms)
	tick(40 * ms)
	expect(t, *controller.LastElapsedDuration(), 20*ms)
	controller.Stop()
	controller.Dispose()
}

func TestAnimationControllerToStringControlTest(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	hasOneLineDescription(t, controller)
	controller.Forward()
	tick(10 * ms)
	tick(20 * ms)
	hasOneLineDescription(t, controller)
	tick(30 * ms)
	hasOneLineDescription(t, controller)
	controller.Reverse()
	tick(40 * ms)
	tick(50 * ms)
	hasOneLineDescription(t, controller)
	controller.Stop()
	controller.Dispose()
}

func inRange(t *testing.T, v, lo, hi float64) {
	t.Helper()
	if v < lo || v > hi {
		t.Fatalf("%v not in [%v, %v]", v, lo, hi)
	}
}

func TestVelocityTestLinear(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(1000 * ms)})

	// mid-flight
	controller.Forward()
	tick(0)
	tick(500 * ms)
	inRange(t, controller.Velocity(), 0.9, 1.1)

	// edges
	controller.Forward()
	inRange(t, controller.Velocity(), 0.4, 0.6)
	tick(0)
	inRange(t, controller.Velocity(), 0.4, 0.6)
	tick(5 * ms)
	inRange(t, controller.Velocity(), 0.9, 1.1)

	controller.Forward(ForwardOptions{From: f64(0.5)})
	inRange(t, controller.Velocity(), 0.4, 0.6)
	tick(0)
	inRange(t, controller.Velocity(), 0.4, 0.6)
	tick(5 * ms)
	inRange(t, controller.Velocity(), 0.9, 1.1)

	// stopped
	controller.Forward(ForwardOptions{From: f64(1.0)})
	expect(t, controller.Velocity(), 0.0)
	tick(0)
	expect(t, controller.Velocity(), 0.0)
	tick(500 * ms)
	expect(t, controller.Velocity(), 0.0)

	controller.Forward()
	tick(0)
	tick(1000 * ms)
	expect(t, controller.Velocity(), 0.0)

	controller.Stop()
	controller.Dispose()
}

func TestDisposedAnimationControllerToStringWorks(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	controller.Dispose()
	hasOneLineDescription(t, controller)
}

func TestAnimationControllerErrorHandling(t *testing.T) {
	setUp(t)
	controller := newController()
	throwsFlutterError(t, func() { controller.Forward() })
	throwsFlutterError(t, func() { controller.Reverse() })
	throwsFlutterError(t, func() { controller.AnimateTo(0.5) })
	throwsFlutterError(t, func() { controller.Repeat() })

	controller.Dispose()
	result := throwsFlutterError(t, func() { controller.Dispose() })
	equalsIgnoringHashCodes(t, result.ToStringDeep(),
		"FlutterError\n"+
			"   AnimationController.dispose() called more than once.\n"+
			"   A given AnimationController cannot be disposed more than once.\n"+
			"   The following AnimationController object was disposed multiple\n"+
			"   times:\n"+
			"     AnimationController#00000(⏮ 0.000; paused; DISPOSED)\n")
	builder := &foundation.DiagnosticPropertiesBuilder{}
	result.DebugFillProperties(builder)
	controllerProperty := builder.Properties[len(builder.Properties)-1]
	expect(t, controllerProperty.Name(), "The following AnimationController object was disposed multiple times")
	expect(t, controllerProperty.Value(), any(controller))
}

func TestAnimationControllerRepeatThrowsIfPeriodIsNotSpecified(t *testing.T) {
	setUp(t)
	controller := newController()
	throwsFlutterError(t, func() { controller.Repeat() })
	throwsFlutterError(t, func() { controller.Repeat() })
	controller.Dispose()
}

func doNotAnimate(t *testing.T, value float64) {
	setUp(t)
	var statusLog []AnimationStatus
	controller := newController(AnimationControllerOptions{Value: f64(value)})
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expect(t, controller.Value(), value)
	controller.AnimateTo(value, AnimateToOptions{Duration: dur(100 * ms)})
	expectStatuses(t, statusLog, AnimationStatusCompleted)
	expect(t, controller.Value(), value)
	controller.Dispose()
}

func TestDoNotAnimateIfAlreadyAtTarget(t *testing.T) { doNotAnimate(t, 0.5) }

func TestDoNotAnimateToUpperBoundIfAlreadyAtUpperBound(t *testing.T) { doNotAnimate(t, 1.0) }

func TestDoNotAnimateToLowerBoundIfAlreadyAtLowerBound(t *testing.T) { doNotAnimate(t, 0.0) }

func TestDoNotAnimateIfAlreadyAtTargetMidFlightForward(t *testing.T) {
	setUp(t)
	var statusLog []AnimationStatus
	controller := newController(AnimationControllerOptions{Value: f64(0.0), Duration: dur(1000 * ms)})
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expect(t, controller.Value(), 0.0)
	controller.Forward()
	tick(0)
	tick(500 * ms)
	inRange(t, controller.Value(), 0.4, 0.6)
	expectStatuses(t, statusLog, AnimationStatusForward)
	currentValue := controller.Value()
	controller.AnimateTo(currentValue, AnimateToOptions{Duration: dur(100 * ms)})
	expectStatuses(t, statusLog, AnimationStatusForward, AnimationStatusCompleted)
	expect(t, controller.Value(), currentValue)
	controller.Dispose()
}

func TestDoNotAnimateIfAlreadyAtTargetMidFlightReverse(t *testing.T) {
	setUp(t)
	var statusLog []AnimationStatus
	controller := newController(AnimationControllerOptions{Value: f64(1.0), Duration: dur(1000 * ms)})
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expect(t, controller.Value(), 1.0)
	controller.Reverse()
	tick(0)
	tick(500 * ms)
	inRange(t, controller.Value(), 0.4, 0.6)
	expectStatuses(t, statusLog, AnimationStatusReverse)
	currentValue := controller.Value()
	controller.AnimateTo(currentValue, AnimateToOptions{Duration: dur(100 * ms)})
	expectStatuses(t, statusLog, AnimationStatusReverse, AnimationStatusCompleted)
	expect(t, controller.Value(), currentValue)
	controller.Dispose()
}

func TestAnimateToCanDealWithDurationZero(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	controller.Forward(ForwardOptions{From: f64(0.2)})
	expect(t, controller.Value(), 0.2)
	controller.AnimateTo(1.0, AnimateToOptions{Duration: dur(0)})
	if n := ensureBinding().TransientCallbackCount(); n != 0 {
		t.Fatalf("Expected no animation: %d transient callbacks", n)
	}
	expect(t, controller.Value(), 1.0)
	controller.Dispose()
}

func TestResettingAnimationWorksAtAllPhases(t *testing.T) {
	setUp(t)
	var statusLog []AnimationStatus
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	controller.Reset()
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)

	statusLog = nil
	controller.Forward()
	tick(0)
	tick(50 * ms)
	expect(t, controller.Status(), AnimationStatusForward)
	controller.Reset()
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	expectStatuses(t, statusLog, AnimationStatusForward, AnimationStatusDismissed)

	controller.SetValue(1.0)
	statusLog = nil
	controller.Reverse()
	tick(0)
	tick(50 * ms)
	expect(t, controller.Status(), AnimationStatusReverse)
	controller.Reset()
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	expectStatuses(t, statusLog, AnimationStatusReverse, AnimationStatusDismissed)

	statusLog = nil
	controller.Forward()
	tick(0)
	tick(150 * ms)
	expect(t, controller.Status(), AnimationStatusCompleted)
	controller.Reset()
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	expectStatuses(t, statusLog, AnimationStatusForward, AnimationStatusCompleted, AnimationStatusDismissed)
	controller.Dispose()
}

func TestSettingValueDirectlySetsCorrectStatus(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Value: f64(0.0)})
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	for _, c := range []struct {
		v float64
		s AnimationStatus
	}{{0.5, AnimationStatusForward}, {1.0, AnimationStatusCompleted}, {0.5, AnimationStatusForward},
		{0.0, AnimationStatusDismissed}} {
		controller.SetValue(c.v)
		expect(t, controller.Value(), c.v)
		expect(t, controller.Status(), c.s)
	}
	controller.Dispose()
}

func TestAnimateToSetsCorrectStatus(t *testing.T) {
	setUp(t)
	var statusLog []AnimationStatus
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	for _, target := range []float64{0.5, 1.0, 0.5, 0.0} {
		controller.AnimateTo(target)
		tick(0)
		tick(150 * ms)
		expect(t, controller.Value(), target)
		expectStatuses(t, statusLog, AnimationStatusForward, AnimationStatusCompleted)
		statusLog = nil
	}
	controller.Dispose()
}

func TestAfterAReverseCallAnimateToSetsCorrectStatus(t *testing.T) {
	setUp(t)
	var statusLog []AnimationStatus
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(1.0)})
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expect(t, controller.Value(), 1.0)
	expect(t, controller.Status(), AnimationStatusCompleted)
	controller.Reverse()
	tick(0)
	tick(150 * ms)
	expect(t, controller.Value(), 0.0)
	expectStatuses(t, statusLog, AnimationStatusReverse, AnimationStatusDismissed)
	statusLog = nil
	controller.AnimateTo(0.5)
	tick(0)
	tick(150 * ms)
	expect(t, controller.Value(), 0.5)
	expectStatuses(t, statusLog, AnimationStatusForward, AnimationStatusCompleted)
	controller.Dispose()
}

func TestAfterAForwardCallAnimateToSetsCorrectStatus(t *testing.T) {
	setUp(t)
	var statusLog []AnimationStatus
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	controller.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	controller.Forward()
	tick(0)
	tick(150 * ms)
	expect(t, controller.Value(), 1.0)
	expectStatuses(t, statusLog, AnimationStatusForward, AnimationStatusCompleted)
	statusLog = nil
	controller.AnimateTo(0.5)
	tick(0)
	tick(150 * ms)
	expect(t, controller.Value(), 0.5)
	expectStatuses(t, statusLog, AnimationStatusForward, AnimationStatusCompleted)
	controller.Dispose()
}

func TestRepeatWithReverseAlternatesBetweenBounds(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	expect(t, controller.Value(), 0.0)
	reverse := RepeatOptions{Reverse: true}

	controller.Repeat(reverse)
	tick(0)
	tick(25 * ms)
	expect(t, controller.Value(), 0.25)
	tick(0)
	tick(125 * ms)
	expect(t, controller.Value(), 0.75)

	controller.Reset()
	controller.SetValue(1.0)
	expect(t, controller.Value(), 1.0)
	controller.Repeat(reverse)
	tick(0)
	tick(25 * ms)
	expect(t, controller.Value(), 0.75)
	tick(0)
	tick(125 * ms)
	expect(t, controller.Value(), 0.25)

	controller.Reset()
	controller.SetValue(0.5)
	expect(t, controller.Value(), 0.5)
	controller.Repeat(reverse)
	tick(0)
	tick(50 * ms)
	expect(t, controller.Value(), 1.0)
	tick(0)
	tick(150 * ms)
	expect(t, controller.Value(), 0.0)
	controller.Dispose()
}

func TestRepeatWithMinAndMaxAlternates(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	expect(t, controller.Value(), 0.0)

	controller.Repeat(RepeatOptions{Reverse: true, Min: f64(0.5), Max: f64(1.0)})
	tick(0)
	tick(50 * ms)
	expect(t, controller.Value(), 0.75)
	tick(0)
	tick(100 * ms)
	expect(t, controller.Value(), 1.00)
	tick(0)
	tick(200 * ms)
	expect(t, controller.Value(), 0.5)

	controller.Reset()
	controller.SetValue(0.0)
	expect(t, controller.Value(), 0.0)
	controller.Repeat(RepeatOptions{Reverse: true, Min: f64(1.0), Max: f64(1.0)})
	tick(0)
	tick(25 * ms)
	expect(t, controller.Value(), 1.0)
	tick(0)
	tick(125 * ms)
	expect(t, controller.Value(), 1.0)

	controller.Reset()
	controller.SetValue(0.2)
	expect(t, controller.Value(), 0.2)
	controller.Repeat(RepeatOptions{Reverse: true, Min: f64(0.2), Max: f64(0.6)})
	tick(0)
	tick(50 * ms)
	expect(t, controller.Value(), 0.4)
	controller.Dispose()
}

func TestRepeatWithNegativeMinAndPositiveMax(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(1.0),
		LowerBound: f64(-1), UpperBound: f64(3)})
	expect(t, controller.Value(), 1.0)
	controller.Repeat(RepeatOptions{Min: f64(1), Max: f64(3)})
	tick(0)
	expect(t, controller.Value(), 1.0)
	tick(50 * ms)
	expect(t, controller.Value(), 2.0)

	controller.Reset()
	controller.SetValue(0.0)
	controller.Repeat(RepeatOptions{Min: f64(-1), Max: f64(3)})
	tick(0)
	expect(t, controller.Value(), 0.0)
	tick(25 * ms)
	expect(t, controller.Value(), 1.0)
	controller.Dispose()
}

func TestAnimationBehaviorDefaultValuesForConstructor(t *testing.T) {
	setUp(t)
	controller := newController()
	expect(t, controller.AnimationBehavior, AnimationBehaviorNormal)
	repeating := NewAnimationControllerUnbounded(vsync)
	expect(t, repeating.AnimationBehavior, AnimationBehaviorPreserve)
	controller.Dispose()
	repeating.Dispose()
}

func disableAnimations() { SemanticsDisableAnimations = func() bool { return true } }

func TestAnimationBehaviorPreserveRunsAtNormalSpeedWhenAnimatingTo(t *testing.T) {
	setUp(t)
	disableAnimations()
	controller := newController(AnimationControllerOptions{AnimationBehavior: AnimationBehaviorPreserve})
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	controller.AnimateTo(1.0, AnimateToOptions{Duration: dur(100 * ms)})
	tick(0)
	tick(50 * ms)
	expect(t, controller.Value(), 0.5)
	expect(t, controller.Status(), AnimationStatusForward)
	tick(0)
	tick(150 * ms)
	expect(t, controller.Value(), 1.0)
	expect(t, controller.Status(), AnimationStatusCompleted)
	controller.Dispose()
}

func TestAnimationBehaviorNormalRuns20xFasterWhenAnimatingTo(t *testing.T) {
	setUp(t)
	disableAnimations()
	controller := newController()
	expect(t, controller.Value(), 0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	controller.AnimateTo(1.0, AnimateToOptions{Duration: dur(100 * ms)})
	tick(0)
	tick(2500 * time.Microsecond)
	expect(t, controller.Value(), 0.5)
	expect(t, controller.Status(), AnimationStatusForward)
	tick(0)
	tick(5*ms + 1000*time.Microsecond)
	expect(t, controller.Value(), 1.0)
	expect(t, controller.Status(), AnimationStatusCompleted)
	controller.Dispose()
}

func TestAnimationBehaviorNormalRunsFasterThanPreserve(t *testing.T) {
	setUp(t)
	disableAnimations()
	controller := newController()
	fastController := newController()
	preserve, normal := AnimationBehaviorPreserve, AnimationBehaviorNormal
	controller.Fling(FlingOptions{AnimationBehavior: &preserve})
	fastController.Fling(FlingOptions{AnimationBehavior: &normal})
	tick(0)
	tick(50 * ms)
	// We don't assert a specific fraction that normal animation.
	expect(t, controller.Value() < fastController.Value(), true)
	controller.Dispose()
	fastController.Dispose()
}

func TestAnimationControllerMethodsAssertTickerIsNotNull(t *testing.T) {
	setUp(t)
	controller := newController()
	controller.Dispose()
	throwsAssertionError(t, func() { controller.AnimateBack(0) })
	throwsAssertionError(t, func() { controller.AnimateTo(0) })
	throwsAssertionError(t, func() { controller.AnimateWith(physics.NewGravitySimulation(0, 0, 0, 0)) })
	throwsAssertionError(t, func() { controller.Stop() })
	throwsAssertionError(t, func() { controller.Forward() })
	throwsAssertionError(t, func() { controller.Reverse() })
}

// testSimulation is the test file's TestSimulation: x(t) = t, never done.
type testSimulation struct{ physics.SimulationBase }

func newTestSimulation() *testSimulation {
	return &testSimulation{SimulationBase: physics.NewSimulationBase()}
}

func (*testSimulation) X(time float64) float64  { return time }
func (*testSimulation) Dx(time float64) float64 { return time }
func (*testSimulation) IsDone(float64) bool     { return false }

func TestSimulationsRunForward(t *testing.T) {
	setUp(t)
	var statuses []AnimationStatus
	controller := newController(AnimationControllerOptions{Duration: dur(time.Second)})
	controller.AddStatusListener(func(s AnimationStatus) { statuses = append(statuses, s) })
	controller.AnimateWith(newTestSimulation())
	tick(0)
	tick(2 * time.Second)
	expectStatuses(t, statuses, AnimationStatusForward)
	controller.Dispose()
}

func TestSimulationsRunForwardEvenAfterAReverseRun(t *testing.T) {
	setUp(t)
	var statuses []AnimationStatus
	controller := newController(AnimationControllerOptions{Duration: dur(time.Second)})
	controller.AddStatusListener(func(s AnimationStatus) { statuses = append(statuses, s) })
	controller.Reverse(ReverseOptions{From: f64(1.0)})
	tick(0)
	tick(2 * time.Second)
	expectStatuses(t, statuses, AnimationStatusCompleted, AnimationStatusReverse, AnimationStatusDismissed)
	statuses = nil
	controller.AnimateWith(newTestSimulation())
	tick(0)
	tick(2 * time.Second)
	expectStatuses(t, statuses, AnimationStatusForward)
	controller.Dispose()
}

func TestRepeatingAnimationWithReverseReportsForwardAndReverse(t *testing.T) {
	setUp(t)
	var statuses []AnimationStatus
	controller := newController(AnimationControllerOptions{Duration: dur(time.Second)})
	controller.AddStatusListener(func(s AnimationStatus) { statuses = append(statuses, s) })
	controller.Repeat(RepeatOptions{Reverse: true})
	tick(0)
	tick(999 * ms)
	expectStatuses(t, statuses, AnimationStatusForward)
	statuses = nil
	tick(time.Second)
	expectStatuses(t, statuses, AnimationStatusReverse)
	controller.Dispose()
}

func TestAnimateBackRunsWithJustReverseDurationSet(t *testing.T) {
	setUp(t)
	var statuses []AnimationStatus
	controller := newController(AnimationControllerOptions{ReverseDuration: dur(2 * time.Second)})
	controller.AddStatusListener(func(s AnimationStatus) { statuses = append(statuses, s) })
	controller.AnimateBack(0.8)
	expectStatuses(t, statuses, AnimationStatusReverse)
	statuses = nil
	tick(0)
	tick(2 * time.Second)
	expectStatuses(t, statuses, AnimationStatusDismissed)
	controller.Dispose()
}

func TestDurationErrorForward(t *testing.T) {
	setUp(t)
	controller := newController()
	err := throwsFlutterError(t, func() { controller.Forward() })
	expect(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   AnimationController.forward() called with no default duration.\n"+
			"   The \"duration\" property should be set, either in the constructor\n"+
			"   or later, before calling the forward() function.\n")
	controller.Dispose()
}

func TestDurationErrorAnimateTo(t *testing.T) {
	setUp(t)
	controller := newController()
	err := throwsFlutterError(t, func() { controller.AnimateTo(0.8) })
	expect(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   AnimationController.animateTo() called with no explicit duration\n"+
			"   and no default duration.\n"+
			"   Either the \"duration\" argument to the animateTo() method should\n"+
			"   be provided, or the \"duration\" property should be set, either in\n"+
			"   the constructor or later, before calling the animateTo()\n"+
			"   function.\n")
	controller.Dispose()
}

func TestDurationErrorReverse(t *testing.T) {
	setUp(t)
	controller := newController()
	err := throwsFlutterError(t, func() { controller.Reverse() })
	expect(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   AnimationController.reverse() called with no default duration or\n"+
			"   reverseDuration.\n"+
			"   The \"duration\" or \"reverseDuration\" property should be set,\n"+
			"   either in the constructor or later, before calling the reverse()\n"+
			"   function.\n")
	controller.Dispose()
}

func TestDurationErrorAnimateBack(t *testing.T) {
	setUp(t)
	controller := newController()
	err := throwsFlutterError(t, func() { controller.AnimateBack(0.8) })
	expect(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   AnimationController.animateBack() called with no explicit\n"+
			"   duration and no default duration or reverseDuration.\n"+
			"   Either the \"duration\" argument to the animateBack() method should\n"+
			"   be provided, or the \"duration\" or \"reverseDuration\" property\n"+
			"   should be set, either in the constructor or later, before calling\n"+
			"   the animateBack() function.\n")
	controller.Dispose()
}

func TestRepeatCountZeroThrowsAssertion(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	expect(t, controller.Value(), 0.0)
	throwsAssertionError(t, func() { controller.Repeat(RepeatOptions{Reverse: true, Count: intp(0)}) })
}

func TestRepeatCountNegativeThrowsAssertion(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	expect(t, controller.Value(), 0.0)
	throwsAssertionError(t, func() { controller.Repeat(RepeatOptions{Reverse: true, Count: intp(-1)}) })
}

func TestRepeatCountWithoutReverse(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	expect(t, controller.Value(), 0.0)
	controller.Repeat(RepeatOptions{Count: intp(1)})
	tick(0)
	tick(25 * ms)
	expect(t, controller.Value(), 0.25)
	tick(50 * ms)
	expect(t, controller.Value(), 0.5)
	tick(99 * ms)
	expect(t, controller.Value(), 0.99)
	tick(100 * ms)
	expect(t, controller.Value(), 0.0)

	controller.Reset()
	expect(t, controller.Value(), 0.0)
	controller.Repeat(RepeatOptions{Count: intp(2)})
	tick(0)
	tick(25 * ms)
	expect(t, controller.Value(), 0.25)
	tick(50 * ms)
	expect(t, controller.Value(), 0.5)
	tick(200 * ms)
	expect(t, controller.Value(), 0.0)
	controller.Reset()
	controller.Dispose()
}

func TestRepeatCountWithReverse(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), Value: f64(0.0)})
	expect(t, controller.Value(), 0.0)
	controller.Repeat(RepeatOptions{Reverse: true, Count: intp(4)})
	tick(0)
	tick(25 * ms)
	expect(t, controller.Value(), 0.25)
	tick(50 * ms)
	expect(t, controller.Value(), 0.5)
	tick(99 * ms)
	expect(t, controller.Value(), 0.99)
	tick(100 * ms)
	expect(t, controller.Value(), 1.0)
	tick(60 * ms)
	expect(t, dart.ToStringAsFixed(controller.Value(), 1), "0.6")
	controller.Reset()
	controller.Dispose()
}
