package difftest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
)

func tol(o physics.ToleranceOptions) *physics.Tolerance { t := physics.NewTolerance(o); return &t }
func fp(v float64) *float64                             { return &v }

// Same scenes as flutter/test/physics_test.dart.
var physicsScenes = map[string]func() physics.Simulation{
	"friction_pos": func() physics.Simulation { return physics.NewFrictionSimulation(0.135, 0, 1000) },
	"friction_neg": func() physics.Simulation { return physics.NewFrictionSimulation(0.135, 100, -1000) },
	"friction_decel": func() physics.Simulation {
		return physics.NewFrictionSimulation(0.135, 0, 3000, physics.FrictionSimulationOptions{ConstantDeceleration: 1400})
	},
	"friction_tol": func() physics.Simulation {
		return physics.NewFrictionSimulation(0.3, 100, 400, physics.FrictionSimulationOptions{Tolerance: tol(physics.ToleranceOptions{Velocity: fp(1)})})
	},
	"friction_through": func() physics.Simulation { return physics.NewFrictionSimulationThrough(10, 300, 600, 20) },
	"bounded_max":      func() physics.Simulation { return physics.NewBoundedFrictionSimulation(0.3, 100, 400, 50, 150) },
	"bounded_min":      func() physics.Simulation { return physics.NewBoundedFrictionSimulation(0.3, 100, -400, 50, 150) },
	"gravity":          func() physics.Simulation { return physics.NewGravitySimulation(-10, 0, 6, 10) },
	"clamped": func() physics.Simulation {
		return physics.NewClampedSimulation(physics.NewGravitySimulation(9.81, 10, 0, 0),
			physics.ClampedSimulationOptions{XMin: fp(20), XMax: fp(100), DxMin: fp(7), DxMax: fp(11)})
	},
	"spring_crit": func() physics.Simulation {
		return physics.NewSpringSimulation(physics.NewSpringDescriptionWithDampingRatio(1, 100, 1), 0, 500, 0)
	},
	"spring_under": func() physics.Simulation {
		return physics.NewSpringSimulation(physics.NewSpringDescriptionWithDampingRatio(1, 100, 0.25), 0, 300, 50)
	},
	"spring_over": func() physics.Simulation {
		return physics.NewSpringSimulation(physics.NewSpringDescriptionWithDampingRatio(1, 100, 1.25), 0, 500, -200)
	},
	"spring_snap": func() physics.Simulation {
		return physics.NewSpringSimulation(physics.NewSpringDescriptionWithDampingRatio(1, 400, 1), 0, 1, 0,
			physics.SpringSimulationOptions{SnapToEnd: true, Tolerance: tol(physics.ToleranceOptions{Distance: fp(0.1), Velocity: fp(0.1)})})
	},
	"spring_bounce": func() physics.Simulation {
		return physics.NewSpringSimulation(physics.NewSpringDescriptionWithDurationAndBounce(500*time.Millisecond, 0.3), 0, 1, 0)
	},
	"scroll_spring": func() physics.Simulation {
		return physics.NewScrollSpringSimulation(physics.NewSpringDescriptionWithDampingRatio(0.5, 100, 1.1), 0, 100, 0,
			physics.ScrollSpringSimulationOptions{Tolerance: tol(physics.ToleranceOptions{Distance: fp(0.5), Velocity: fp(5)})})
	},
}

type physicsRef struct {
	Samples [][4]float64       `json:"samples"`
	Scalars map[string]float64 `json:"scalars"`
	String  string             `json:"string"`
}

// closeTo is exact equality: with dart.Pow/Log/Exp/... (the same libm
// calls and special cases as the Dart VM), no fused multiply-adds
// (fmacheck) and step-by-step constants, these values match Flutter bit for
// bit. It is kept as a function so a tolerance could be reintroduced in one
// place if a platform's libm ever differed.
func closeTo(a, b float64) bool { return a == b || (a != a && b != b) }

func TestPhysicsMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/physics.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]physicsRef
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != len(physicsScenes)+1 {
		t.Fatalf("reference has %d scenes, Go has %d (+1 spring_description)", len(ref), len(physicsScenes))
	}
	maxDiff := 0.0
	for name, mk := range physicsScenes {
		r, ok := ref[name]
		if !ok {
			t.Fatalf("%s: missing from reference", name)
		}
		s := mk()
		for _, smp := range r.Samples {
			tm := smp[0]
			done := 0.0
			if s.IsDone(tm) {
				done = 1
			}
			got := [4]float64{tm, s.X(tm), s.Dx(tm), done}
			for i := 1; i < 4; i++ {
				maxDiff = math.Max(maxDiff, math.Abs(got[i]-smp[i]))
				if !closeTo(got[i], smp[i]) {
					t.Errorf("%s t=%v: got %v, Flutter %v", name, tm, got, smp)
					break
				}
			}
		}
		if f, ok := s.(interface{ FinalX() float64 }); ok {
			if !closeTo(f.FinalX(), r.Scalars["finalX"]) {
				t.Errorf("%s finalX = %v, Flutter %v", name, f.FinalX(), r.Scalars["finalX"])
			}
			ta := s.(interface{ TimeAtX(float64) float64 })
			for _, at := range []float64{0.1, 0.5, 2.0} {
				key := fmt.Sprintf("timeAtX(%v)", at)
				if at == 2 {
					key = "timeAtX(2.0)"
				}
				want, ok := r.Scalars[key]
				if got := ta.TimeAtX(s.X(at)); ok && !closeTo(got, want) {
					t.Errorf("%s %s = %v, Flutter %v", name, key, got, want)
				}
			}
		}
		if ty, ok := s.(interface{ Type() physics.SpringType }); ok && float64(ty.Type()) != r.Scalars["type"] {
			t.Errorf("%s type = %v, Flutter %v", name, ty.Type(), r.Scalars["type"])
		}
		if got := fmt.Sprint(s); got != r.String {
			t.Errorf("%s String() = %q, Flutter %q", name, got, r.String)
		}
	}
	d := physics.NewSpringDescriptionWithDurationAndBounce(350*time.Millisecond, -0.2)
	sc := ref["spring_description"].Scalars
	for k, got := range map[string]float64{"mass": d.Mass, "stiffness": d.Stiffness, "damping": d.Damping,
		"bounce": d.Bounce(), "durationMs": float64(d.Duration().Milliseconds())} {
		if !closeTo(got, sc[k]) {
			t.Errorf("spring_description %s = %v, Flutter %v", k, got, sc[k])
		}
	}
	t.Logf("max difference from Flutter: %g", maxDiff)
}
