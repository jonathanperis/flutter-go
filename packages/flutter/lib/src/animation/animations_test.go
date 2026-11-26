package animation

// Port of packages/flutter/test/animation/animations_test.dart (Flutter
// 3.50.1): all 18 tests.

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// bogusCurve is the test file's BogusCurve: it overrides transform itself
// and maps everything to 100.
type bogusCurve struct{ CurveBase }

func newBogusCurve() *bogusCurve { c := &bogusCurve{}; c.SetSelf(c); return c }

func (*bogusCurve) Transform(float64) float64 { return 100.0 }

func TestAnimationsToStringControlTest(t *testing.T) {
	setUp(t)
	hasOneLineDescription(t, KAlwaysCompleteAnimation)
	hasOneLineDescription(t, KAlwaysDismissedAnimation)
	hasOneLineDescription(t, NewAlwaysStoppedAnimation(0.5))
	curvedAnimation := NewCurvedAnimation(KAlwaysDismissedAnimation, Curves.Ease)
	hasOneLineDescription(t, curvedAnimation)
	curvedAnimation.ReverseCurve = Curves.ElasticOut
	hasOneLineDescription(t, curvedAnimation)
	controller := newController(AnimationControllerOptions{Duration: dur(500 * ms)})
	controller.SetValue(0.5)
	controller.Reverse()
	curvedAnimation = NewCurvedAnimation(controller, Curves.Ease, CurvedAnimationOptions{ReverseCurve: Curves.ElasticOut})
	hasOneLineDescription(t, curvedAnimation)
	controller.Stop()
}

func TestProxyAnimationToStringControlTest(t *testing.T) {
	setUp(t)
	animation := NewProxyAnimation()
	expect(t, animation.Value(), 0.0)
	expect(t, animation.Status(), AnimationStatusDismissed)
	hasOneLineDescription(t, animation)
	animation.SetParent(KAlwaysDismissedAnimation)
	hasOneLineDescription(t, animation)
}

func TestProxyAnimationSetParentGeneratesValueChanged(t *testing.T) {
	setUp(t)
	controller := newController()
	controller.SetValue(0.5)
	didReceiveCallback := false
	animation := NewProxyAnimation()
	animation.AddListener(func() { didReceiveCallback = true })
	expect(t, didReceiveCallback, false)
	animation.SetParent(controller)
	expect(t, didReceiveCallback, true)
	didReceiveCallback = false
	expect(t, didReceiveCallback, false)
	controller.SetValue(0.6)
	expect(t, didReceiveCallback, true)
}

func TestReverseAnimationCallsListeners(t *testing.T) {
	setUp(t)
	controller := newController()
	controller.SetValue(0.5)
	didReceiveCallback := false
	listener := func() { didReceiveCallback = true }
	animation := NewReverseAnimation(controller)
	animation.AddListener(listener)
	expect(t, didReceiveCallback, false)
	controller.SetValue(0.6)
	expect(t, didReceiveCallback, true)
	didReceiveCallback = false
	animation.RemoveListener(listener)
	expect(t, didReceiveCallback, false)
	controller.SetValue(0.7)
	expect(t, didReceiveCallback, false)
	hasOneLineDescription(t, animation)
}

func TestTrainHoppingAnimation(t *testing.T) {
	setUp(t)
	currentTrain := newController()
	t.Cleanup(currentTrain.Dispose)
	nextTrain := newController()
	t.Cleanup(nextTrain.Dispose)
	currentTrain.SetValue(0.5)
	nextTrain.SetValue(0.75)
	didSwitchTrains := false
	animation := NewTrainHoppingAnimation(currentTrain, nextTrain,
		TrainHoppingAnimationOptions{OnSwitchedTrain: func() { didSwitchTrains = true }})
	expect(t, didSwitchTrains, false)
	expect(t, animation.Value(), 0.5)
	hasOneLineDescription(t, animation)
	nextTrain.SetValue(0.25)
	expect(t, didSwitchTrains, true)
	expect(t, animation.Value(), 0.25)
	hasOneLineDescription(t, animation)
	if !strings.Contains(animation.String(), "no next") {
		t.Fatal(animation.String())
	}
}

func TestTrainHoppingAnimationDispatchesMemoryEvents(t *testing.T) {
	setUp(t)
	events := memoryEvents(func() {
		NewTrainHoppingAnimation(NewAlwaysStoppedAnimation(1.0), NewAlwaysStoppedAnimation(1.0)).Dispose()
	}, func(o any) bool { _, ok := o.(*TrainHoppingAnimation); return ok })
	areCreateAndDispose(t, events)
}

// Regression test for https://github.com/flutter/flutter/issues/178336.
func TestTrainHoppingAnimationNotifiesStatusListeners(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	t.Cleanup(controller.Dispose)
	animation := NewTrainHoppingAnimation(
		NewTween(TweenOptions[float64]{Begin: 1.0, End: -1.0}).Animate(controller),
		NewTween(TweenOptions[float64]{Begin: -1.0, End: 1.0}).Animate(controller))
	t.Cleanup(animation.Dispose)
	var statusLog []AnimationStatus
	animation.AddStatusListener(func(s AnimationStatus) { statusLog = append(statusLog, s) })
	expectStatuses(t, statusLog)
	controller.Forward()
	expectStatuses(t, statusLog, AnimationStatusForward)
	statusLog = nil
	controller.Reverse()
	expectStatuses(t, statusLog, AnimationStatusDismissed)
}

func TestAnimationMeanControlTest(t *testing.T) {
	setUp(t)
	left := newController(AnimationControllerOptions{Value: f64(0.5)})
	right := newController()
	mean := NewAnimationMean(left, right)
	hasOneLineDescription(t, mean)
	expect(t, mean.Value(), 0.25)
	var log []float64
	logValue := func() { log = append(log, mean.Value()) }
	mean.AddListener(logValue)
	right.SetValue(1.0)
	expect(t, mean.Value(), 0.75)
	if len(log) != 1 || log[0] != 0.75 {
		t.Fatal(log)
	}
	log = nil
	mean.RemoveListener(logValue)
	left.SetValue(0.0)
	expect(t, mean.Value(), 0.50)
	expect(t, len(log), 0)
}

func TestAnimationMaxControlTest(t *testing.T) {
	setUp(t)
	first := newController(AnimationControllerOptions{Value: f64(0.5)})
	second := newController()
	max := NewAnimationMax[float64](first, second)
	hasOneLineDescription(t, max)
	expect(t, max.Value(), 0.5)
	var log []float64
	logValue := func() { log = append(log, max.Value()) }
	max.AddListener(logValue)
	second.SetValue(1.0)
	expect(t, max.Value(), 1.0)
	if len(log) != 1 || log[0] != 1.0 {
		t.Fatal(log)
	}
	log = nil
	max.RemoveListener(logValue)
	first.SetValue(0.0)
	expect(t, max.Value(), 1.0)
	expect(t, len(log), 0)
}

func TestAnimationMinControlTest(t *testing.T) {
	setUp(t)
	first := newController(AnimationControllerOptions{Value: f64(0.5)})
	second := newController()
	min := NewAnimationMin[float64](first, second)
	hasOneLineDescription(t, min)
	expect(t, min.Value(), 0.0)
	var log []float64
	logValue := func() { log = append(log, min.Value()) }
	min.AddListener(logValue)
	second.SetValue(1.0)
	expect(t, min.Value(), 0.5)
	if len(log) != 1 || log[0] != 0.5 {
		t.Fatal(log)
	}
	log = nil
	min.RemoveListener(logValue)
	first.SetValue(0.25)
	expect(t, min.Value(), 0.25)
	expect(t, len(log), 0)
}

func TestCurvedAnimationWithBogusCurve(t *testing.T) {
	setUp(t)
	controller := newController()
	curved := NewCurvedAnimation(controller, newBogusCurve())
	err := throwsFlutterError(t, func() { _ = curved.Value() })
	re := regexp.MustCompile(`^FlutterError
   Invalid curve endpoint at \d+(\.\d*)?\.
   Curves must map 0\.0 to near zero and 1\.0 to near one but
   bogusCurve mapped \d+(\.\d*)? to \d+(\.\d*)?, which is near \d+(\.\d*)?\.
$`)
	if s := err.ToStringDeep(); !re.MatchString(s) {
		t.Fatalf("%q", s)
	}
}

func TestCurvedAnimationRunningWithDifferentForwardAndReverseDurations(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), ReverseDuration: dur(50 * ms)})
	curved := NewCurvedAnimation(controller, Curves.Linear, CurvedAnimationOptions{ReverseCurve: Curves.Linear})
	controller.Forward()
	tick(0)
	for i := 1; i <= 10; i++ {
		tick(time.Duration(i*10) * ms)
		if !near(curved.Value(), float64(i)/10) {
			t.Fatalf("at %d: %v", i, curved.Value())
		}
	}
	controller.Reverse()
	for i, want := range []float64{1.0, 0.8, 0.6, 0.4, 0.2, 0.0} {
		tick(time.Duration(110+i*10) * ms)
		if !near(curved.Value(), want) {
			t.Fatalf("at %d: %v, want %v", i, curved.Value(), want)
		}
	}
}

func TestCurvedAnimationStopsListeningToParentWhenDisposed(t *testing.T) {
	setUp(t)
	forwardCurve := NewInterval(0.0, 0.5)
	reverseCurve := NewInterval(0.5, 1.0)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), ReverseDuration: dur(100 * ms)})
	curved := NewCurvedAnimation(controller, forwardCurve, CurvedAnimationOptions{ReverseCurve: reverseCurve})
	expect(t, forwardCurve.Transform(0.5), 1.0)
	expect(t, reverseCurve.Transform(0.5), 0.0)

	controller.Forward(ForwardOptions{From: f64(0.5)})
	expect(t, controller.Status(), AnimationStatusForward)
	expect(t, curved.Value(), 1.0)
	controller.SetValue(1.0)
	expect(t, controller.Status(), AnimationStatusCompleted)
	controller.Reverse(ReverseOptions{From: f64(0.5)})
	expect(t, controller.Status(), AnimationStatusReverse)
	expect(t, curved.Value(), 0.0)

	expect(t, curved.IsDisposed, false)
	curved.Dispose()
	expect(t, curved.IsDisposed, true)

	controller.SetValue(0.0)
	expect(t, controller.Status(), AnimationStatusDismissed)
	controller.Forward(ForwardOptions{From: f64(0.5)})
	expect(t, controller.Status(), AnimationStatusForward)
	expect(t, curved.Value(), 0.0)
}

func TestReverseAnimationRunningWithDifferentForwardAndReverseDurations(t *testing.T) {
	setUp(t)
	controller := newController(AnimationControllerOptions{Duration: dur(100 * ms), ReverseDuration: dur(50 * ms)})
	reversed := NewReverseAnimation(NewCurvedAnimation(controller, Curves.Linear,
		CurvedAnimationOptions{ReverseCurve: Curves.Linear}))
	controller.Forward()
	tick(0)
	for i := 1; i <= 10; i++ {
		tick(time.Duration(i*10) * ms)
		if !near(reversed.Value(), 1-float64(i)/10) {
			t.Fatalf("at %d: %v", i, reversed.Value())
		}
	}
	controller.Reverse()
	for i, want := range []float64{0.0, 0.2, 0.4, 0.6, 0.8, 1.0} {
		tick(time.Duration(110+i*10) * ms)
		if !near(reversed.Value(), want) {
			t.Fatalf("at %d: %v, want %v", i, reversed.Value(), want)
		}
	}
}

func TestTweenSequence(t *testing.T) {
	setUp(t)
	controller := newController()
	animation := NewTweenSequence([]TweenSequenceItem[float64]{
		NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 5.0, End: 10.0}), 4.0),
		NewTweenSequenceItem[float64](NewConstantTween(10.0), 2.0),
		NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 10.0, End: 5.0}), 4.0),
	}).Animate(controller)
	for _, c := range [][2]float64{{0, 5.0}, {0.2, 7.5}, {0.4, 10.0}, {0.6, 10.0}, {0.8, 7.5}, {1.0, 5.0}} {
		controller.SetValue(c[0])
		expect(t, animation.Value(), c[1])
	}
}

func TestTweenSequenceWithCurves(t *testing.T) {
	setUp(t)
	controller := newController()
	animation := NewTweenSequence([]TweenSequenceItem[float64]{
		NewTweenSequenceItem(NewTween(TweenOptions[float64]{Begin: 5.0, End: 10.0}).Chain(NewCurveTween(NewInterval(0.5, 1.0))), 4.0),
		NewTweenSequenceItem(NewConstantTween(10.0).Chain(NewCurveTween(Curves.Linear)), 2.0), // linear is a no-op
		NewTweenSequenceItem(NewTween(TweenOptions[float64]{Begin: 10.0, End: 5.0}).Chain(NewCurveTween(NewInterval(0.0, 0.5))), 4.0),
	}).Animate(controller)
	for _, c := range [][2]float64{{0, 5.0}, {0.2, 5.0}, {0.4, 10.0}, {0.6, 10.0}, {0.8, 5.0}, {1.0, 5.0}} {
		controller.SetValue(c[0])
		expect(t, animation.Value(), c[1])
	}
}

func TestTweenSequenceOneTween(t *testing.T) {
	setUp(t)
	controller := newController()
	animation := NewTweenSequence([]TweenSequenceItem[float64]{
		NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 5.0, End: 10.0}), 1.0),
	}).Animate(controller)
	expect(t, animation.Value(), 5.0)
	controller.SetValue(0.5)
	expect(t, animation.Value(), 7.5)
	controller.SetValue(1.0)
	expect(t, animation.Value(), 10.0)
}

func TestCurvedAnimationDispatchesMemoryEvents(t *testing.T) {
	setUp(t)
	events := memoryEvents(func() {
		NewCurvedAnimation(newController(ms100()), Curves.Linear).Dispose()
	}, func(o any) bool { _, ok := o.(*CurvedAnimation); return ok })
	areCreateAndDispose(t, events)
}
