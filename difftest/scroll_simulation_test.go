package difftest

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/jonathanperis/flutter-go/physics"
	"github.com/jonathanperis/flutter-go/widgets"
)

// Same scenes as flutter/test/scroll_simulation_test.dart.
func scrollSimulationScenes() map[string]physics.Simulation {
	ios := physics.NewSpringDescriptionWithDampingRatio(0.5, 100, 1.1)
	mac := physics.NewSpringDescriptionWithDampingRatio(0.3, 75, 1.3)
	dpr3 := physics.NewTolerance(physics.ToleranceOptions{Velocity: fp(1 / (0.050 * 3)), Distance: fp(1.0 / 3)})
	b := func(pos, vel, lead, trail float64, sp physics.SpringDescription, decel float64, tol *physics.Tolerance) physics.Simulation {
		return widgets.NewBouncingScrollSimulation(widgets.BouncingScrollSimulationOptions{Position: pos, Velocity: vel,
			LeadingExtent: lead, TrailingExtent: trail, Spring: sp, ConstantDeceleration: decel, Tolerance: tol})
	}
	c := func(pos, vel float64, friction *float64, tol *physics.Tolerance) physics.Simulation {
		return widgets.NewClampingScrollSimulation(widgets.ClampingScrollSimulationOptions{Position: pos, Velocity: vel,
			Friction: friction, Tolerance: tol})
	}
	return map[string]physics.Simulation{
		"bounce_within":      b(300, 800, 0, 1000, ios, 0, &dpr3),
		"bounce_cap_over":    b(900, 12000, 0, 1000, ios, 0, nil),
		"bounce_cap_under":   b(100, -12000, 0, 1000, ios, 0, nil),
		"bounce_under_start": b(-0.5, 0, 0, 1000, ios, 0, &dpr3),
		"bounce_over_start":  b(1000.5, 300, 0, 1000, ios, 0, nil),
		"bounce_mac_fast":    b(500, 4000, 0, 1000, mac, 1400, &dpr3),
		"bounce_empty_range": b(0, 2000, 0, 0, ios, 0, nil),
		"clamping_fast":      c(0, 8000, nil, nil),
		"clamping_up":        c(100, -3000, nil, &dpr3),
		"clamping_slow":      c(50, 0.5, nil, nil),
		"clamping_friction":  c(0, 2500, fp(0.03), nil),
	}
}

func TestScrollSimulationsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/scroll_simulation.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string][][4]float64
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	scenes := scrollSimulationScenes()
	if len(ref) != len(scenes) {
		t.Fatalf("reference has %d scenes, Go has %d", len(ref), len(scenes))
	}
	maxDiff := 0.0
	for name, s := range scenes {
		for _, smp := range ref[name] {
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
	}
	t.Logf("max difference from Flutter: %g", maxDiff)
}
