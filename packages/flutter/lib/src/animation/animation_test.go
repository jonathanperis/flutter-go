package animation

// flutter-go's own tests of AnimationController timing (Flutter has no
// animation_test.dart; these were the tests of the old ad-hoc controller,
// kept on the real one), and of the Go-specific parts of the port.
//
// Flutter test files of test/animation not ported (they test flutter_test,
// not the animation library): animation_sheet_test.dart (3 tests,
// AnimationSheetBuilder golden files) and live_binding_test.dart (2 tests,
// LiveTestWidgetsFlutterBinding's pointer indicator goldens); neither
// binding nor golden-file comparison is ported.

import (
	"math"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
)

func TestForwardFromIdleStartsAtFirstTick(t *testing.T) {
	setUp(t)
	s := ensureBinding()
	tick(100 * ms) // an idle frame; time passes
	c := NewAnimationController(vsync, AnimationControllerOptions{Duration: dur(200 * ms)})
	t.Cleanup(c.Dispose)
	var statuses []AnimationStatus
	c.AddStatusListener(func(st AnimationStatus) { statuses = append(statuses, st) })
	c.Forward() // outside a frame
	tick(150 * ms)
	if c.Value() != 0 {
		t.Fatalf("first tick defines the start time, value = %v", c.Value())
	}
	tick(250 * ms)
	if math.Abs(c.Value()-0.5) > 1e-12 {
		t.Fatalf("value after 100ms of 200ms = %v", c.Value())
	}
	tick(400 * ms)
	if c.Value() != 1 || c.IsAnimating() {
		t.Fatalf("value %v animating %v", c.Value(), c.IsAnimating())
	}
	expectStatuses(t, statuses, AnimationStatusForward, AnimationStatusCompleted)
	if s.HasScheduledFrame() {
		t.Fatal("a finished animation must stop requesting frames")
	}
}

func TestStartedInsideFrameUsesFrameTimestamp(t *testing.T) {
	setUp(t)
	s := ensureBinding()
	c := NewAnimationController(vsync, AnimationControllerOptions{Duration: dur(100 * ms)})
	t.Cleanup(c.Dispose)
	ts := 1000 * ms
	s.HandleBeginFrame(&ts)
	c.Forward() // e.g. from didUpdateWidget during build
	s.HandleDrawFrame()
	tick(1050 * ms)
	if math.Abs(c.Value()-0.5) > 1e-12 {
		t.Fatalf("value = %v, want 0.5", c.Value())
	}
}

func TestReverseDurationScalesWithDistance(t *testing.T) {
	setUp(t)
	c := NewAnimationController(vsync, AnimationControllerOptions{Duration: dur(400 * ms)})
	t.Cleanup(c.Dispose)
	c.SetValue(0.25)
	c.Reverse() // 0.25 of the range → 100ms
	tick(0)
	tick(50 * ms)
	if math.Abs(c.Value()-0.125) > 1e-12 {
		t.Fatalf("value = %v, want 0.125", c.Value())
	}
	// At exactly the duration the value is at the end, but the simulation
	// is done only once the time is past it (isDone is `t > duration`).
	tick(100 * ms)
	if c.Value() != 0 || c.Status() != AnimationStatusReverse {
		t.Fatalf("value %v status %v", c.Value(), c.Status())
	}
	tick(100*ms + 1000)
	if c.Value() != 0 || c.Status() != AnimationStatusDismissed {
		t.Fatalf("value %v status %v", c.Value(), c.Status())
	}
}

func TestCubicEndpointsAndMonotonic(t *testing.T) {
	for _, c := range []Curve{Curves.Ease, Curves.EaseIn, Curves.EaseOut, Curves.EaseInOut, Curves.FastOutSlowIn} {
		if c.Transform(0) != 0 || c.Transform(1) != 1 {
			t.Fatalf("%v endpoints", c)
		}
		prev := 0.0
		for i := 1; i < 100; i++ {
			v := c.Transform(float64(i) / 100)
			if v < prev-1e-3 {
				t.Fatalf("%v not monotonic at %d", c, i)
			}
			prev = v
		}
	}
	if v := Curves.EaseInOut.Transform(0.5); math.Abs(v-0.5) > 1e-3 {
		t.Fatalf("easeInOut(0.5) = %v", v)
	}
}

func TestAnimationStatusNamesAndPredicates(t *testing.T) {
	want := []string{"dismissed", "forward", "reverse", "completed"}
	for i, s := range []AnimationStatus{AnimationStatusDismissed, AnimationStatusForward, AnimationStatusReverse, AnimationStatusCompleted} {
		expect(t, s.Name(), want[i])
		expect(t, s.String(), "AnimationStatus."+want[i])
		expect(t, s.IsDismissed(), i == 0)
		expect(t, s.IsCompleted(), i == 3)
		expect(t, s.IsAnimating(), i == 1 || i == 2)
		expect(t, s.IsForwardOrCompleted(), i == 1 || i == 3)
	}
	expect(t, AnimationBehaviorNormal.String(), "AnimationBehavior.normal")
	expect(t, AnimationBehaviorPreserve.String(), "AnimationBehavior.preserve")
}

// An animation embedding AnimationBase from outside, without SetSelf,
// panics with a message saying so.
func TestUnboundSelfPanics(t *testing.T) {
	var a AnimationBase[float64]
	r := panics(func() { _ = a.IsAnimating() })
	if s, _ := r.(string); !strings.Contains(s, "SetSelf") {
		t.Fatalf("panic %v", r)
	}
	var tw Tween[float64]
	if r := panics(func() { tw.Transform(0.5) }); r == nil {
		t.Fatal("an unbound Tween must panic")
	}
}

// A type extending AnimationController (as _TestAnimationController does)
// gets its own type in toString and in error messages.
type myController struct{ AnimationController }

func TestExtendedControllerUsesOuterType(t *testing.T) {
	st := setUp(t)
	c := &myController{}
	c.Construct(c, vsync)
	if !strings.HasPrefix(c.String(), "myController#") {
		t.Fatalf("String() = %q", c.String())
	}
	c.AddListener(func() { panic("boom") })
	c.SetValue(0.5)
	if e := st.takeException(); e != "boom" {
		t.Fatalf("exception %v", e)
	}
	_ = physics.DefaultTolerance
	_ = foundation.KDebugMode
}
