package physics

import (
	"fmt"
	"math"
	"reflect"

	"github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// ClampedSimulation clamps another simulation's position and velocity.
type ClampedSimulation struct {
	SimulationBase
	Simulation               Simulation
	XMin, XMax, DxMin, DxMax float64
}

// ClampedSimulationOptions are the optional bounds; nil means unbounded.
type ClampedSimulationOptions struct {
	XMin, XMax, DxMin, DxMax *float64
}

// NewClampedSimulation is Flutter's ClampedSimulation(simulation, {xMin,
// xMax, dxMin, dxMax}); unset bounds are ±infinity.
func NewClampedSimulation(simulation Simulation, opts ...ClampedSimulationOptions) *ClampedSimulation {
	s := &ClampedSimulation{SimulationBase: NewSimulationBase(), Simulation: simulation,
		XMin: math.Inf(-1), XMax: math.Inf(1), DxMin: math.Inf(-1), DxMax: math.Inf(1)}
	for _, o := range opts {
		set := func(dst *float64, v *float64) {
			if v != nil {
				*dst = *v
			}
		}
		set(&s.XMin, o.XMin)
		set(&s.XMax, o.XMax)
		set(&s.DxMin, o.DxMin)
		set(&s.DxMax, o.DxMax)
	}
	if s.XMax < s.XMin || s.DxMax < s.DxMin {
		panic("physics: ClampedSimulation bounds are inverted")
	}
	return s
}

func (s *ClampedSimulation) X(t float64) float64 {
	return ui.ClampDouble(s.Simulation.X(t), s.XMin, s.XMax)
}
func (s *ClampedSimulation) Dx(t float64) float64 {
	return ui.ClampDouble(s.Simulation.Dx(t), s.DxMin, s.DxMax)
}
func (s *ClampedSimulation) IsDone(t float64) bool { return s.Simulation.IsDone(t) }
func (s *ClampedSimulation) String() string {
	return "ClampedSimulation(simulation: " + describe(s.Simulation) + ", x: " + fixed1(s.XMin) + ".." +
		fixed1(s.XMax) + ", dx: " + fixed1(s.DxMin) + ".." + fixed1(s.DxMax) + ")"
}

// describe is Dart's '$simulation': its toString, or else the runtime type
// name (Object.toString in Dart's objectRuntimeType form).
func describe(s Simulation) string {
	if st, ok := s.(fmt.Stringer); ok {
		return st.String()
	}
	t := reflect.TypeOf(s)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}
