package physics

import "math"

// GravitySimulation is a particle under constant acceleration; it is done
// once |x| reaches endDistance.
type GravitySimulation struct {
	SimulationBase
	x, v, a, end float64
}

func NewGravitySimulation(acceleration, distance, endDistance, velocity float64) *GravitySimulation {
	if endDistance < 0 {
		panic("physics: endDistance must be >= 0")
	}
	return &GravitySimulation{NewSimulationBase(), distance, velocity, acceleration, endDistance}
}

func (s *GravitySimulation) X(t float64) float64   { return s.x + s.v*t + 0.5*s.a*t*t }
func (s *GravitySimulation) Dx(t float64) float64  { return s.v + t*s.a }
func (s *GravitySimulation) IsDone(t float64) bool { return math.Abs(s.X(t)) >= s.end }
func (s *GravitySimulation) String() string {
	return "GravitySimulation(g: " + fixed1(s.a) + ", x₀: " + fixed1(s.x) + ", dx₀: " + fixed1(s.v) +
		", xₘₐₓ: ±" + fixed1(s.end) + ")"
}
