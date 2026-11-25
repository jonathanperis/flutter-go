// Ports of packages/flutter/test/physics/*_test.dart (Flutter 3.50.1).
// BouncingScrollSimulation's cases (newton_test.dart) live in widgets, where
// Flutter declares that simulation.
package physics

import (
	"math"
	"testing"
	"time"
)

func moreOrLess(t *testing.T, name string, got, want float64, eps ...float64) {
	t.Helper()
	e := 1e-10 // flutter_test's precisionErrorTolerance
	if len(eps) > 0 {
		e = eps[0]
	}
	if !(math.Abs(got-want) <= e || got == want) {
		t.Errorf("%s = %v, want %v ± %v", name, got, want, e)
	}
}

func eq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func f(v float64) *float64 { return &v }

// clamped_simulation_test.dart
func TestClampedSimulation1(t *testing.T) {
	gravity := NewGravitySimulation(9.81, 10.0, 0.0, 0.0)
	c := NewClampedSimulation(gravity, ClampedSimulationOptions{XMin: f(20), XMax: f(100), DxMin: f(7), DxMax: f(11)})
	eq(t, "x(0)", c.X(0), 20.0)
	eq(t, "dx(0)", c.Dx(0), 7.0)
	eq(t, "x(100)", c.X(100), 100.0)
	eq(t, "dx(100)", c.Dx(100), 11.0)
}

func TestClampedSimulation2(t *testing.T) {
	gravity := NewGravitySimulation(-10, 0.0, 6.0, 10.0)
	c := NewClampedSimulation(gravity, ClampedSimulationOptions{XMin: f(0), XMax: f(2.5), DxMin: f(-1), DxMax: f(1)})
	eq(t, "x(0)", c.X(0), 0.0)
	eq(t, "dx(0)", c.Dx(0), 1.0)
	eq(t, "isDone(0)", c.IsDone(0), false)
	eq(t, "x(1)", c.X(1), 2.5)
	eq(t, "dx(1)", c.Dx(1), 0.0)
	eq(t, "isDone(0.2)", c.IsDone(0.2), false)
	eq(t, "x(2)", c.X(2), 0.0)
	eq(t, "dx(2)", c.Dx(2), -1.0)
	eq(t, "isDone(2)", c.IsDone(2), false)
	eq(t, "x(3)", c.X(3), 0.0)
	eq(t, "dx(3)", c.Dx(3), -1.0)
	eq(t, "isDone(3)", c.IsDone(3), true)
}

// friction_simulation_test.dart
func TestFrictionSimulationPositiveVelocity(t *testing.T) {
	fr := NewFrictionSimulation(0.135, 100.0, 100.0)
	moreOrLess(t, "x(0)", fr.X(0), 100)
	moreOrLess(t, "dx(0)", fr.Dx(0), 100)
	moreOrLess(t, "x(0.1)", fr.X(0.1), 110, 1)
	moreOrLess(t, "x(0.5)", fr.X(0.5), 131, 1)
	moreOrLess(t, "x(2)", fr.X(2), 149, 1)
	moreOrLess(t, "finalX", fr.FinalX(), 149, 1)
	eq(t, "timeAtX(100)", fr.TimeAtX(100), 0.0)
	moreOrLess(t, "timeAtX(x(0.1))", fr.TimeAtX(fr.X(0.1)), 0.1)
	moreOrLess(t, "timeAtX(x(0.5))", fr.TimeAtX(fr.X(0.5)), 0.5)
	moreOrLess(t, "timeAtX(x(2))", fr.TimeAtX(fr.X(2)), 2)
	eq(t, "timeAtX(-1)", fr.TimeAtX(-1), math.Inf(1))
	eq(t, "timeAtX(200)", fr.TimeAtX(200), math.Inf(1))
}

func TestFrictionSimulationNegativeVelocity(t *testing.T) {
	fr := NewFrictionSimulation(0.135, 100.0, -100.0)
	moreOrLess(t, "x(0)", fr.X(0), 100)
	moreOrLess(t, "dx(0)", fr.Dx(0), -100)
	moreOrLess(t, "x(0.1)", fr.X(0.1), 91, 1)
	moreOrLess(t, "x(0.5)", fr.X(0.5), 68, 1)
	moreOrLess(t, "x(2)", fr.X(2), 51, 1)
	moreOrLess(t, "finalX", fr.FinalX(), 50, 1)
	eq(t, "timeAtX(100)", fr.TimeAtX(100), 0.0)
	moreOrLess(t, "timeAtX(x(0.1))", fr.TimeAtX(fr.X(0.1)), 0.1)
	moreOrLess(t, "timeAtX(x(0.5))", fr.TimeAtX(fr.X(0.5)), 0.5)
	moreOrLess(t, "timeAtX(x(2))", fr.TimeAtX(fr.X(2)), 2)
	eq(t, "timeAtX(101)", fr.TimeAtX(101), math.Inf(1))
	eq(t, "timeAtX(40)", fr.TimeAtX(40), math.Inf(1))
}

func TestFrictionSimulationConstantDeceleration(t *testing.T) {
	fr := NewFrictionSimulation(0.135, 100.0, -100.0, FrictionSimulationOptions{ConstantDeceleration: 100})
	moreOrLess(t, "x(0)", fr.X(0), 100)
	moreOrLess(t, "dx(0)", fr.Dx(0), -100)
	moreOrLess(t, "x(0.1)", fr.X(0.1), 91, 1)
	moreOrLess(t, "x(0.5)", fr.X(0.5), 80, 1)
	moreOrLess(t, "x(2)", fr.X(2), 80, 1)
	moreOrLess(t, "finalX", fr.FinalX(), 80, 1)
	eq(t, "timeAtX(100)", fr.TimeAtX(100), 0.0)
	moreOrLess(t, "timeAtX(x(0.1))", fr.TimeAtX(fr.X(0.1)), 0.1)
	moreOrLess(t, "timeAtX(x(0.2))", fr.TimeAtX(fr.X(0.2)), 0.2)
	moreOrLess(t, "timeAtX(x(0.3))", fr.TimeAtX(fr.X(0.3)), 0.3)
	eq(t, "timeAtX(101)", fr.TimeAtX(101), math.Inf(1))
	eq(t, "timeAtX(40)", fr.TimeAtX(40), math.Inf(1))
}

// gravity_simulation_test.dart
func TestGravitySimulation1(t *testing.T) {
	g := NewGravitySimulation(9.81, 10.0, 0.0, 0.0)
	if s := g.String(); s == "" || containsNewline(s) {
		t.Errorf("String() = %q, want one line", s)
	}
	moreOrLess(t, "x(10)", g.X(10), 50.0*9.81+10.0)
}

func containsNewline(s string) bool {
	for _, r := range s {
		if r == '\n' {
			return true
		}
	}
	return false
}

func TestGravitySimulation2(t *testing.T) {
	g := NewGravitySimulation(-10, 0.0, 6.0, 10.0)
	eq(t, "x(0)", g.X(0), 0.0)
	eq(t, "dx(0)", g.Dx(0), 10.0)
	eq(t, "isDone(0)", g.IsDone(0), false)
	eq(t, "x(1)", g.X(1), 5.0)
	eq(t, "dx(1)", g.Dx(1), 0.0)
	eq(t, "isDone(0.2)", g.IsDone(0.2), false)
	eq(t, "x(2)", g.X(2), 0.0)
	eq(t, "dx(2)", g.Dx(2), -10.0)
	eq(t, "isDone(2)", g.IsDone(2), false)
	eq(t, "x(3)", g.X(3), -15.0)
	eq(t, "dx(3)", g.Dx(3), -20.0)
	eq(t, "isDone(3)", g.IsDone(3), true)
}

// near_equal_test.dart (test_null has no Go equivalent: floats can't be nil)
func TestNearEqualFriction(t *testing.T) {
	eq(t, "1", NearEqual(5, 6, 2), true)
	eq(t, "2", NearEqual(6, 5, 2), true)
	eq(t, "3", NearEqual(5, 6, 0.5), false)
	eq(t, "4", NearEqual(6, 5, 0.5), false)
}

// utils_test.dart
func TestNearEquals(t *testing.T) {
	inf := math.Inf(1)
	eq(t, "inf", NearEqual(inf, inf, 0.1), true)
	eq(t, "-inf", NearEqual(-inf, -inf, 0.1), true)
	eq(t, "inf/-inf", NearEqual(inf, -inf, 0.1), false)
	eq(t, "0.1/0.11 tight", NearEqual(0.1, 0.11, 0.001), false)
	eq(t, "0.1/0.11 loose", NearEqual(0.1, 0.11, 0.1), true)
	eq(t, "0.1/0.1", NearEqual(0.1, 0.1, 0.0000001), true)
}

// newton_test.dart
func TestFriction(t *testing.T) {
	fr := NewFrictionSimulation(0.3, 100.0, 400.0)
	fr.SetTolerance(NewTolerance(ToleranceOptions{Velocity: f(1)}))
	eq(t, "isDone(0)", fr.IsDone(0), false)
	eq(t, "x(0)", fr.X(0), 100.0)
	eq(t, "dx(0)", fr.Dx(0), 400.0)
	if x := fr.X(1); !(x > 330 && x < 335) {
		t.Errorf("x(1) = %v", x)
	}
	eq(t, "dx(1)", fr.Dx(1), 120.0)
	eq(t, "dx(2)", fr.Dx(2), 36.0)
	moreOrLess(t, "dx(3)", fr.Dx(3), 10.8)
	if fr.Dx(4) >= 3.5 {
		t.Errorf("dx(4) = %v", fr.Dx(4))
	}
	eq(t, "isDone(5)", fr.IsDone(5), true)
	if x := fr.X(5); !(x > 431 && x < 432) {
		t.Errorf("x(5) = %v", x)
	}
}

func TestFrictionThrough(t *testing.T) {
	startPosition, startVelocity := 10.0, 600.0
	fs := NewFrictionSimulation(0.025, startPosition, startVelocity)
	endPosition, endVelocity := fs.X(1), fs.Dx(1)
	if !(endPosition > startPosition && endVelocity < startVelocity) {
		t.Fatalf("bad setup")
	}
	fr := NewFrictionSimulationThrough(startPosition, endPosition, startVelocity, endVelocity)
	eq(t, "isDone(0)", fr.IsDone(0), false)
	eq(t, "x(0)", fr.X(0), 10.0)
	eq(t, "dx(0)", fr.Dx(0), 600.0)
	eq(t, "isDone(1+e)", fr.IsDone(1.0+1e-10), true)
	moreOrLess(t, "x(1)", fr.X(1), endPosition)
	moreOrLess(t, "dx(1)", fr.Dx(1), endVelocity)

	startPosition, startVelocity = 1000.0, -500.0
	fs = NewFrictionSimulation(0.025, 1000.0, -500.0)
	endPosition, endVelocity = fs.X(1), fs.Dx(1)
	if !(endPosition < startPosition && endVelocity > startVelocity) {
		t.Fatalf("bad setup")
	}
	fr = NewFrictionSimulationThrough(startPosition, endPosition, startVelocity, endVelocity)
	eq(t, "isDone(1+e)", fr.IsDone(1.0+1e-10), true)
	moreOrLess(t, "x(1)", fr.X(1), endPosition)
	moreOrLess(t, "dx(1)", fr.Dx(1), endVelocity)
}

func TestBoundedFrictionSimulation(t *testing.T) {
	fr := NewBoundedFrictionSimulation(0.3, 100.0, 400.0, 50.0, 150.0)
	fr.SetTolerance(NewTolerance(ToleranceOptions{Velocity: f(1)}))
	eq(t, "isDone(0)", fr.IsDone(0), false)
	eq(t, "x(0)", fr.X(0), 100.0)
	eq(t, "dx(0)", fr.Dx(0), 400.0)
	eq(t, "x(1)", fr.X(1), 150.0)
	eq(t, "isDone(1)", fr.IsDone(1), true)
}

func TestGravity(t *testing.T) {
	g := NewGravitySimulation(200.0, 100.0, 600.0, 0.0)
	eq(t, "isDone(0)", g.IsDone(0), false)
	eq(t, "x(0)", g.X(0), 100.0)
	eq(t, "dx(0)", g.Dx(0), 0.0)
	xs := map[float64]float64{0.25: 106.25, 0.5: 125, 0.75: 156.25, 1: 200, 1.25: 256.25, 1.5: 325, 1.75: 406.25}
	for tm, want := range xs {
		eq(t, "x", g.X(tm), want)
		eq(t, "dx", g.Dx(tm), 200*tm)
	}
	eq(t, "isDone(2.5)", g.IsDone(2.5), true)
	eq(t, "x(2.5)", g.X(2.5), 725.0)
	eq(t, "dx(2.5)", g.Dx(2.5), 500.0)
}

func TestSpringTypes(t *testing.T) {
	sim := func(ratio float64) *SpringSimulation {
		return NewSpringSimulation(NewSpringDescriptionWithDampingRatio(1, 100, ratio), 0, 300, 0)
	}
	eq(t, "crit", sim(1).Type(), SpringTypeCriticallyDamped)
	eq(t, "under", sim(0.75).Type(), SpringTypeUnderDamped)
	eq(t, "over", sim(1.25).Type(), SpringTypeOverDamped)
	other := NewSpringSimulation(SpringDescription{Mass: 1, Stiffness: 100, Damping: 20}, 0, 20, 20)
	eq(t, "other", other.Type(), SpringTypeCriticallyDamped)
}

func TestCritSpring(t *testing.T) {
	c := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(1, 100, 1), 0, 500, 0)
	c.SetTolerance(NewTolerance(ToleranceOptions{Distance: f(0.01), Velocity: f(0.01)}))
	eq(t, "type", c.Type(), SpringTypeCriticallyDamped)
	eq(t, "isDone(0)", c.IsDone(0), false)
	eq(t, "x(0)", c.X(0), 0.0)
	eq(t, "dx(0)", c.Dx(0), 0.0)
	eq(t, "x(.25)", math.Floor(c.X(0.25)), 356.0)
	eq(t, "x(.5)", math.Floor(c.X(0.5)), 479.0)
	eq(t, "x(.75)", math.Floor(c.X(0.75)), 497.0)
	eq(t, "dx(.25)", math.Floor(c.Dx(0.25)), 1026.0)
	eq(t, "dx(.5)", math.Floor(c.Dx(0.5)), 168.0)
	eq(t, "dx(.75)", math.Floor(c.Dx(0.75)), 20.0)
	if x := c.X(1.5); !(x > 499 && x < 501) {
		t.Errorf("x(1.5) = %v", x)
	}
	if c.Dx(1.5) >= 0.1 {
		t.Errorf("dx(1.5) = %v", c.Dx(1.5))
	}
	eq(t, "isDone(1.6)", c.IsDone(1.6), true)
}

func TestOverdampedSpring(t *testing.T) {
	o := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(1, 100, 1.25), 0, 500, 0)
	o.SetTolerance(NewTolerance(ToleranceOptions{Distance: f(0.01), Velocity: f(0.01)}))
	eq(t, "type", o.Type(), SpringTypeOverDamped)
	eq(t, "isDone(0)", o.IsDone(0), false)
	eq(t, "x(0)", o.X(0), 0.0)
	moreOrLess(t, "dx(0)", o.Dx(0), 0)
	eq(t, "x(.5)", math.Floor(o.X(0.5)), 445.0)
	eq(t, "x(1)", math.Floor(o.X(1)), 495.0)
	eq(t, "x(1.5)", math.Floor(o.X(1.5)), 499.0)
	eq(t, "dx(.5)", math.Floor(o.Dx(0.5)), 273.0)
	eq(t, "dx(1)", math.Floor(o.Dx(1)), 22.0)
	eq(t, "dx(1.5)", math.Floor(o.Dx(1.5)), 1.0)
	eq(t, "isDone(3)", o.IsDone(3), true)
}

func TestUnderdampedSpring(t *testing.T) {
	u := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(1, 100, 0.25), 0, 300, 0)
	eq(t, "type", u.Type(), SpringTypeUnderDamped)
	eq(t, "isDone(0)", u.IsDone(0), false)
	moreOrLess(t, "x(0)", u.X(0), 0)
	moreOrLess(t, "dx(0)", u.Dx(0), 0)
	eq(t, "x(1)", math.Floor(u.X(1)), 325.0)
	eq(t, "dx(1)", math.Floor(u.Dx(1)), -65.0)
	eq(t, "dx(6)", math.Floor(u.Dx(6)), 0.0)
	eq(t, "x(6)", math.Floor(u.X(6)), 299.0)
	eq(t, "isDone(6)", u.IsDone(6), true)
}

// spring_simulation_test.dart
func TestSpringSnapToEnd(t *testing.T) {
	d := NewSpringDescriptionWithDampingRatio(1, 400, 1)
	tol := NewTolerance(ToleranceOptions{Distance: f(0.1), Velocity: f(0.1)})
	regular := NewSpringSimulation(d, 0, 1, 0, SpringSimulationOptions{Tolerance: &tol})
	if !(regular.X(0.4) < 1 && regular.Dx(0.4) > 0) {
		t.Errorf("regular: x=%v dx=%v", regular.X(0.4), regular.Dx(0.4))
	}
	snapping := NewSpringSimulation(d, 0, 1, 0, SpringSimulationOptions{SnapToEnd: true, Tolerance: &tol})
	eq(t, "snap x", snapping.X(0.4), 1.0)
	eq(t, "snap dx", snapping.Dx(0.4), 0.0)
}

func TestSpringContinuousNearCriticalDamping(t *testing.T) {
	for _, ratio := range []float64{1, 1 + 1e-3, 1 - 1e-3} {
		s := NewSpringSimulation(NewSpringDescriptionWithDampingRatio(0.4, 0.4, ratio), 0, 1, 0)
		moreOrLess(t, "x", s.X(0.4), 0.06155, 0.01)
		moreOrLess(t, "dx", s.Dx(0.4), 0.2681, 0.01)
	}
}

func TestSpringWithDurationAndBounce(t *testing.T) {
	s := NewSpringDescriptionWithDurationAndBounce(500*time.Millisecond, 0.3)
	eq(t, "mass", s.Mass, 1.0)
	moreOrLess(t, "stiffness", s.Stiffness, 157.91, 0.01)
	moreOrLess(t, "damping", s.Damping, 17.59, 0.01)
	moreOrLess(t, "bounce", s.Bounce(), 0.3, 0.0001)
	eq(t, "duration", s.Duration().Milliseconds(), int64(500))

	s = NewSpringDescriptionWithDurationAndBounce(500*time.Millisecond, -0.3)
	moreOrLess(t, "stiffness", s.Stiffness, 157.91, 0.01)
	moreOrLess(t, "damping", s.Damping, 35.90, 0.01)
	moreOrLess(t, "bounce", s.Bounce(), -0.3, 0.0001)
	eq(t, "duration", s.Duration().Milliseconds(), int64(500))

	s = SpringDescription{Mass: 1, Stiffness: 157.91, Damping: 17.59}
	moreOrLess(t, "bounce", s.Bounce(), 0.3, 0.001)
	eq(t, "duration", s.Duration().Milliseconds(), int64(500))

	s = NewSpringDescriptionWithDurationAndBounce(100*time.Millisecond, 0)
	moreOrLess(t, "stiffness", s.Stiffness, 3947.84, 0.01)
	moreOrLess(t, "damping", s.Damping, 125.66, 0.01)
	moreOrLess(t, "bounce", s.Bounce(), 0, 0.001)
	eq(t, "duration", s.Duration().Milliseconds(), int64(100))

	for _, d := range []time.Duration{-time.Second, 0} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("duration %v: want panic", d)
				}
			}()
			NewSpringDescriptionWithDurationAndBounce(d, 0.3)
		}()
	}
}

// to_string_test.dart
type TestSimulation struct{ SimulationBase }

func (*TestSimulation) X(float64) float64   { return 0 }
func (*TestSimulation) Dx(float64) float64  { return 0 }
func (*TestSimulation) IsDone(float64) bool { return true }

func TestSimulationToString(t *testing.T) {
	eq(t, "clamped", NewClampedSimulation(&TestSimulation{}, ClampedSimulationOptions{XMin: f(-1), XMax: f(2), DxMin: f(-3), DxMax: f(4)}).String(),
		"ClampedSimulation(simulation: TestSimulation, x: -1.0..2.0, dx: -3.0..4.0)")
	eq(t, "gravity", NewGravitySimulation(1, -2, 3, -4).String(), "GravitySimulation(g: 1.0, x₀: -2.0, dx₀: -4.0, xₘₐₓ: ±3.0)")
	eq(t, "friction", NewFrictionSimulation(1, -2, 3).String(), "FrictionSimulation(cₓ: 1.0, x₀: -2.0, dx₀: 3.0)")
	eq(t, "bounded", NewBoundedFrictionSimulation(1, -2, 3, -4, 5).String(),
		"BoundedFrictionSimulation(cₓ: 1.0, x₀: -2.0, dx₀: 3.0, x: -4.0..5.0)")
	eq(t, "spring desc", SpringDescription{Mass: 1, Stiffness: -2, Damping: 3}.String(),
		"SpringDescription(mass: 1.0, stiffness: -2.0, damping: 3.0)")
	eq(t, "spring ratio", NewSpringDescriptionWithDampingRatio(1, 9, 1).String(),
		"SpringDescription(mass: 1.0, stiffness: 9.0, damping: 6.0)")
	eq(t, "spring sim", NewSpringSimulation(SpringDescription{Mass: 1, Stiffness: 2, Damping: 3}, 0, 1, 2).String(),
		"SpringSimulation(end: 1.0, SpringType.overDamped)")
}

// tolerance_test.dart
func TestToleranceString(t *testing.T) {
	eq(t, "default", DefaultTolerance.String(), "Tolerance(distance: ±0.001, time: ±0.001, velocity: ±0.001)")
}

// Not covered by Flutter's physics tests (it is exercised through scrolling):
// ScrollSpringSimulation's position snaps to the end once done, its velocity
// does not, and a plain SpringSimulation does neither.
func TestScrollSpringSimulationSnapsPosition(t *testing.T) {
	sp := NewSpringDescriptionWithDampingRatio(0.5, 100, 1.1)
	tol := NewTolerance(ToleranceOptions{Distance: f(0.5), Velocity: f(5)})
	s := NewScrollSpringSimulation(sp, 0, 100, 0, ScrollSpringSimulationOptions{Tolerance: &tol})
	plain := NewSpringSimulation(sp, 0, 100, 0, SpringSimulationOptions{Tolerance: &tol})
	at := 0.0
	for !s.IsDone(at) {
		at += 0.001
	}
	eq(t, "x when done", s.X(at), 100.0)
	if plain.X(at) == 100 {
		t.Errorf("plain spring snapped: %v", plain.X(at))
	}
	if s.Dx(at) == 0 {
		t.Errorf("velocity snapped to 0")
	}
	if s.X(at-0.01) == 100 {
		t.Errorf("snapped before done")
	}
}
