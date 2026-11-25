package animation

import "math"

// Simulation is a time → value function, like Flutter's Simulation.
type Simulation interface {
	X(t float64) float64 // value at t seconds
	IsDone(t float64) bool
}

// ClampingScrollSimulation is Flutter's port of Android's OverScroller fling
// (scroll_simulation.dart), with identical constants and formulas.
type ClampingScrollSimulation struct {
	Position, Velocity, Friction float64
	duration, distance           float64
}

var kDecelerationRate = math.Log(0.78) / math.Log(0.9)

const (
	kInflexion    = 0.35
	physicalCoeff = 9.80665 * 39.37 * 160.0 * 0.84
)

func NewClampingScrollSimulation(position, velocity float64) *ClampingScrollSimulation {
	s := &ClampingScrollSimulation{Position: position, Velocity: velocity, Friction: 0.015}
	referenceVelocity := s.Friction * physicalCoeff / kInflexion
	androidDuration := math.Pow(math.Abs(velocity)/referenceVelocity, 1/(kDecelerationRate-1.0))
	s.duration = kDecelerationRate * kInflexion * androidDuration
	s.distance = velocity * s.duration / kDecelerationRate
	return s
}

func (s *ClampingScrollSimulation) X(time float64) float64 {
	t := math.Max(0, math.Min(1, time/s.duration))
	return s.Position + s.distance*(1.0-math.Pow(1.0-t, kDecelerationRate))
}

func (s *ClampingScrollSimulation) IsDone(time float64) bool { return time >= s.duration }

// AnimateWith drives the controller with a simulation (Flutter's
// animateWith): each tick sets value = sim.X(elapsed seconds).
func (c *Controller) AnimateWith(sim Simulation) {
	c.ticker.Stop()
	c.sim = sim
	c.setStatus(Forward)
	c.value = math.Max(c.lower, math.Min(c.upper, sim.X(0)))
	c.ticker.Start()
	c.notify()
}

// Stop halts the controller where it is.
func (c *Controller) Stop() { c.ticker.Stop(); c.sim = nil }
