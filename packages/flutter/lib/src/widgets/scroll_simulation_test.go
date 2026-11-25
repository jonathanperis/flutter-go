// Ports of packages/flutter/test/widgets/scroll_simulation_test.dart and the
// BouncingScrollSimulation cases of test/physics/newton_test.dart
// (Flutter 3.50.1).
package widgets

import (
	"math"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
)

func near(t *testing.T, name string, got, want float64, eps ...float64) {
	t.Helper()
	e := 1e-10
	if len(eps) > 0 {
		e = eps[0]
	}
	if !(math.Abs(got-want) <= e || got == want) {
		t.Errorf("%s = %v, want %v ± %v", name, got, want, e)
	}
}

func clamping(position, velocity float64) *ClampingScrollSimulation {
	return NewClampingScrollSimulation(ClampingScrollSimulationOptions{Position: position, Velocity: velocity})
}

var clampingCases = [][2]float64{{51.0, 2866.91537}, {584.0, 2617.294734}, {345.0, 1982.785934}, {0.0, 1831.366634},
	{-156.2, 1541.57665}, {4.0, 1139.250439}, {4534.0, 1073.553798}, {75.0, 614.2093}, {5469.0, 182.114534}}

func TestClampingScrollSimulationStableInitialConditions(t *testing.T) {
	for _, c := range clampingCases {
		s := clamping(c[0], c[1])
		near(t, "x(0)", s.X(0), c[0])
		near(t, "dx(0)", s.Dx(0), c[1])
	}
}

func TestClampingScrollSimulationOnlyDecelerates(t *testing.T) {
	s := clamping(0, 8000)
	tm, v := 0.0, s.Dx(0)
	for !s.IsDone(tm) {
		if tm >= 3 {
			t.Fatalf("not done by 3 s")
		}
		tm += 1.0 / 60
		next := s.Dx(tm)
		if next > v {
			t.Fatalf("sped up at %v: %v > %v", tm, next, v)
		}
		v = next
	}
}

func TestClampingScrollSimulationSmoothStop(t *testing.T) {
	const maxDeceleration, delta = 5130.0, 1.0 / 60
	s := clamping(0, 8000)
	tm, v := 0.0, s.Dx(0)
	for {
		if tm >= 3 {
			t.Fatalf("not done by 3 s")
		}
		tm += delta
		next := s.Dx(tm)
		if math.Abs(next-v) >= delta*maxDeceleration {
			t.Fatalf("velocity jumps at %v", tm)
		}
		v = next
		if s.IsDone(tm) {
			break
		}
	}
	near(t, "final velocity", v, 0)
}

func TestClampingScrollSimulationIsBallistic(t *testing.T) {
	const delta = 1.0 / 90
	undisturbed := clamping(0, 8000)
	restarted := undisturbed
	var xr, xu, dr, du []float64
	tm := 0.0
	for {
		if tm >= 4 {
			t.Fatalf("not done by 4 s")
		}
		tm += delta
		restarted = clamping(restarted.X(delta), restarted.Dx(delta))
		xr, xu = append(xr, restarted.X(0)), append(xu, undisturbed.X(tm))
		dr, du = append(dr, restarted.Dx(0)), append(du, undisturbed.Dx(tm))
		if restarted.IsDone(0) && undisturbed.IsDone(tm) {
			break
		}
	}
	for i := range xr {
		near(t, "x", xr[i], xu[i])
		near(t, "dx", dr[i], du[i])
	}
}

func TestClampingScrollSimulationPhysicalAcceleration(t *testing.T) {
	kDecel := math.Log(0.78) / math.Log(0.9)
	const refVelocity = .015 * 9.80665 * 39.37 * 160.0 * 0.84 / 0.35
	refDuration := kDecel * 0.35
	refDecel := (kDecel - 1) * refVelocity / refDuration
	sign := func(v float64) float64 { return math.Copysign(1, v) }
	acceleration := func(v float64) float64 {
		return -sign(v) * refDecel * math.Pow(math.Abs(v)/refVelocity, (kDecel-2)/(kDecel-1))
	}
	jerk := func(v float64) float64 {
		return refVelocity / refDuration / refDuration * (kDecel - 1) * (kDecel - 2) *
			math.Pow(math.Abs(v)/refVelocity, (kDecel-3)/(kDecel-1))
	}
	const delta = 1.0 / 60
	for _, c := range clampingCases {
		s := clamping(c[0], c[1])
		for tm := 0.0; tm < 2; tm += delta {
			diff := s.Dx(tm+delta) - s.Dx(tm)
			predicted := delta * acceleration(s.Dx(tm+delta/2))
			if math.Abs(diff-predicted) >= jerk(s.Dx(tm+delta))*delta*delta/2 {
				t.Fatalf("(%v, %v) t=%v: acceleration off: %v vs %v", c[0], c[1], tm, diff, predicted)
			}
		}
	}
}

func bouncing(position, velocity, leading, trailing float64, spring physics.SpringDescription) *BouncingScrollSimulation {
	return NewBouncingScrollSimulation(BouncingScrollSimulationOptions{Position: position, Velocity: velocity,
		LeadingExtent: leading, TrailingExtent: trailing, Spring: spring})
}

func tolerance(distance, velocity float64) physics.Tolerance {
	o := physics.ToleranceOptions{Velocity: F(velocity)}
	if distance > 0 {
		o.Distance = F(distance)
	}
	return physics.NewTolerance(o)
}

// newton_test.dart: test_kinetic_scroll
func TestKineticScroll(t *testing.T) {
	spring := physics.NewSpringDescriptionWithDampingRatio(1, 50, 0.5)
	for _, v := range []float64{800, -800} {
		s := bouncing(100, v, 0, 300, spring)
		s.SetTolerance(tolerance(0.1, 0.5))
		if s.IsDone(0) || s.IsDone(0.5) || !s.IsDone(3.5) {
			t.Errorf("v=%v: isDone(0, 0.5, 3.5) = %v %v %v", v, s.IsDone(0), s.IsDone(0.5), s.IsDone(3.5))
		}
	}
}

// newton_test.dart: scroll_with_inf_edge_ends
func TestScrollWithInfEdgeEnds(t *testing.T) {
	s := bouncing(100, 400, 0, math.Inf(1), physics.NewSpringDescriptionWithDampingRatio(1, 50, 0.5))
	s.SetTolerance(tolerance(0, 1))
	if s.IsDone(0) {
		t.Errorf("done at 0")
	}
	near(t, "x(0)", s.X(0), 100)
	near(t, "dx(0)", s.Dx(0), 400)
	near(t, "x(1)", s.X(1), 272, 1)
	near(t, "dx(1)", s.Dx(1), 54, 1)
	near(t, "dx(2)", s.Dx(2), 7, 1)
	if s.Dx(3) >= 1 {
		t.Errorf("dx(3) = %v", s.Dx(3))
	}
	if !s.IsDone(5) {
		t.Errorf("not done at 5")
	}
	near(t, "x(5)", s.X(5), 300, 1)
}

// newton_test.dart: over/under scroll spring
func TestOverUnderScrollSpring(t *testing.T) {
	s := bouncing(500, -7500, 0, 1000, physics.NewSpringDescriptionWithDampingRatio(1, 170, 1.1))
	s.SetTolerance(tolerance(1.5, 45))
	check := func(tm float64, done bool, x, dx, eps float64) {
		t.Helper()
		if s.IsDone(tm) != done {
			t.Errorf("isDone(%v) = %v", tm, !done)
		}
		near(t, "x", s.X(tm), x, eps)
		near(t, "dx", s.Dx(tm), dx, eps)
	}
	check(0, false, 500, -7500, 1e-10)
	check(0.065, false, 42, -6584, 1)
	check(0.1, false, -123, -2613, 1)
	check(0.5, false, -15, 124, 1)
	check(0, false, 500, -7500, 1e-10) // jump back to the beginning
	if !s.IsDone(2) {
		t.Errorf("not done at 2")
	}
	if s.X(2) != 0 {
		t.Errorf("x(2) = %v, want exactly 0", s.X(2))
	}
	near(t, "dx(2)", s.Dx(2), 0, 1)
}

func TestScrollSimulationStrings(t *testing.T) {
	s := bouncing(0, 0, 0, 100, physics.NewSpringDescriptionWithDampingRatio(1, 50, 0.5))
	if got := s.String(); got != "BouncingScrollSimulation(leadingExtent: 0.0, trailingExtent: 100.0)" {
		t.Errorf("String() = %q", got)
	}
}
