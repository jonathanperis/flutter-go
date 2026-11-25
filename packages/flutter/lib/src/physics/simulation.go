package physics

import (
	"math"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

// Simulation is a one-dimensional simulation over time (Flutter's abstract
// Simulation). Implementations usually embed SimulationBase for the tolerance.
type Simulation interface {
	// X is the position at time (seconds).
	X(time float64) float64
	// Dx is the velocity at time.
	Dx(time float64) float64
	// IsDone reports whether the simulation has settled at time.
	IsDone(time float64) bool
	Tolerance() Tolerance
	SetTolerance(Tolerance)
}

// SimulationBase holds Simulation.tolerance; embed it in a Simulation.
type SimulationBase struct{ tolerance Tolerance }

// NewSimulationBase returns a base with the given tolerance, or
// DefaultTolerance when none is given.
func NewSimulationBase(tolerance ...Tolerance) SimulationBase {
	if len(tolerance) > 0 {
		return SimulationBase{tolerance[0]}
	}
	return SimulationBase{DefaultTolerance}
}

func (s *SimulationBase) Tolerance() Tolerance     { return s.tolerance }
func (s *SimulationBase) SetTolerance(t Tolerance) { s.tolerance = t }

// powE is Dart's math.pow(math.e, x), which Flutter uses instead of exp; the
// two can differ in the last bit, and the difftests compare exactly.
func powE(x float64) float64 { return math.Pow(math.E, x) }

func dartDouble(v float64) string { return dart.DoubleToString(v) }
func fixed1(v float64) string     { return dart.ToStringAsFixed(v, 1) }
