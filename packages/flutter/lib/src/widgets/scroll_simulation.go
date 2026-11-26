package widgets

import (
	"fmt"
	"math"

	"github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
)

// Ported from packages/flutter/lib/src/widgets/scroll_simulation.dart
// (Flutter 3.50.1).

// MaxSpringTransferVelocity is BouncingScrollSimulation.maxSpringTransferVelocity.
const MaxSpringTransferVelocity = 5000.0

// BouncingScrollSimulation is friction inside [LeadingExtent,
// TrailingExtent] that hands over to a spring when it crosses an edge, or a
// spring straight away when it starts out of range (iOS-style scrolling).
type BouncingScrollSimulation struct {
	physics.SimulationBase
	LeadingExtent, TrailingExtent float64
	Spring                        physics.SpringDescription

	friction   *physics.FrictionSimulation
	spring     physics.Simulation
	springTime float64
	timeOffset float64
}

// BouncingScrollSimulationOptions are the constructor's arguments (Flutter's
// are all named).
type BouncingScrollSimulationOptions struct {
	Position, Velocity            float64
	LeadingExtent, TrailingExtent float64
	Spring                        physics.SpringDescription
	ConstantDeceleration          float64
	Tolerance                     *physics.Tolerance // default physics.DefaultTolerance
}

func NewBouncingScrollSimulation(o BouncingScrollSimulationOptions) *BouncingScrollSimulation {
	if o.LeadingExtent > o.TrailingExtent {
		panic("widgets: BouncingScrollSimulation leadingExtent > trailingExtent")
	}
	s := &BouncingScrollSimulation{SimulationBase: physics.NewSimulationBase(),
		LeadingExtent: o.LeadingExtent, TrailingExtent: o.TrailingExtent, Spring: o.Spring}
	if o.Tolerance != nil {
		s.SimulationBase = physics.NewSimulationBase(*o.Tolerance)
	}
	switch {
	case o.Position < o.LeadingExtent:
		s.spring = s.underscrollSimulation(o.Position, o.Velocity)
		s.springTime = math.Inf(-1)
	case o.Position > o.TrailingExtent:
		s.spring = s.overscrollSimulation(o.Position, o.Velocity)
		s.springTime = math.Inf(-1)
	default:
		s.friction = physics.NewFrictionSimulation(0.135, o.Position, o.Velocity,
			physics.FrictionSimulationOptions{ConstantDeceleration: o.ConstantDeceleration})
		finalX := s.friction.FinalX()
		switch {
		case o.Velocity > 0 && finalX > o.TrailingExtent:
			s.springTime = s.friction.TimeAtX(o.TrailingExtent)
			s.spring = s.overscrollSimulation(o.TrailingExtent,
				math.Min(s.friction.Dx(s.springTime), MaxSpringTransferVelocity))
		case o.Velocity < 0 && finalX < o.LeadingExtent:
			s.springTime = s.friction.TimeAtX(o.LeadingExtent)
			s.spring = s.underscrollSimulation(o.LeadingExtent,
				math.Min(s.friction.Dx(s.springTime), MaxSpringTransferVelocity))
		default:
			s.springTime = math.Inf(1)
		}
	}
	return s
}

func (s *BouncingScrollSimulation) underscrollSimulation(x, dx float64) physics.Simulation {
	return physics.NewScrollSpringSimulation(s.Spring, x, s.LeadingExtent, dx)
}

func (s *BouncingScrollSimulation) overscrollSimulation(x, dx float64) physics.Simulation {
	return physics.NewScrollSpringSimulation(s.Spring, x, s.TrailingExtent, dx)
}

// simulation picks the active phase and, as Flutter does, gives it this
// simulation's current tolerance.
func (s *BouncingScrollSimulation) simulation(t float64) physics.Simulation {
	var sim physics.Simulation
	if t > s.springTime {
		s.timeOffset = 0
		if !math.IsInf(s.springTime, 0) {
			s.timeOffset = s.springTime
		}
		sim = s.spring
	} else {
		s.timeOffset = 0
		sim = s.friction
	}
	sim.SetTolerance(s.Tolerance())
	return sim
}

func (s *BouncingScrollSimulation) X(t float64) float64 {
	sim := s.simulation(t)
	return sim.X(t - s.timeOffset)
}

func (s *BouncingScrollSimulation) Dx(t float64) float64 {
	sim := s.simulation(t)
	return sim.Dx(t - s.timeOffset)
}

func (s *BouncingScrollSimulation) IsDone(t float64) bool {
	sim := s.simulation(t)
	return sim.IsDone(t - s.timeOffset)
}

func (s *BouncingScrollSimulation) String() string {
	return fmt.Sprintf("BouncingScrollSimulation(leadingExtent: %s, trailingExtent: %s)",
		dart.DoubleToString(s.LeadingExtent), dart.DoubleToString(s.TrailingExtent))
}

// ClampingScrollSimulation is Android's OverScroller fling, as Flutter ports
// it: same constants and formulas.
type ClampingScrollSimulation struct {
	physics.SimulationBase
	Position, Velocity, Friction float64
	duration, distance           float64
}

// ClampingScrollSimulationOptions are the constructor's arguments.
type ClampingScrollSimulationOptions struct {
	Position, Velocity float64
	Friction           *float64 // default 0.015
	Tolerance          *physics.Tolerance
}

var kDecelerationRate = dart.Log(0.78) / dart.Log(0.9)

const kInflexion = 0.35

// physicalCoeff is computed step by step in float64, as Dart evaluates its
// const: a Go constant expression would be exact and round only once.
var physicalCoeff = func() float64 {
	g, inch, px, tune := 9.80665, 39.37, 160.0, 0.84
	return g * inch * px * tune
}()

func NewClampingScrollSimulation(o ClampingScrollSimulationOptions) *ClampingScrollSimulation {
	s := &ClampingScrollSimulation{SimulationBase: physics.NewSimulationBase(),
		Position: o.Position, Velocity: o.Velocity, Friction: 0.015}
	if o.Friction != nil {
		s.Friction = *o.Friction
	}
	if o.Tolerance != nil {
		s.SimulationBase = physics.NewSimulationBase(*o.Tolerance)
	}
	referenceVelocity := s.Friction * physicalCoeff / kInflexion
	androidDuration := dart.Pow(math.Abs(s.Velocity)/referenceVelocity, 1/(kDecelerationRate-1.0))
	s.duration = float64(kDecelerationRate * kInflexion * androidDuration)
	s.distance = s.Velocity * s.duration / kDecelerationRate
	return s
}

func (s *ClampingScrollSimulation) X(time float64) float64 {
	t := ui.ClampDouble(time/s.duration, 0, 1)
	return s.Position + float64(s.distance*(1.0-dart.Pow(1.0-t, kDecelerationRate)))
}

func (s *ClampingScrollSimulation) Dx(time float64) float64 {
	t := ui.ClampDouble(time/s.duration, 0, 1)
	return s.Velocity * dart.Pow(1.0-t, kDecelerationRate-1.0)
}

func (s *ClampingScrollSimulation) IsDone(time float64) bool { return time >= s.duration }

// String is Dart's default Object.toString for this class.
func (s *ClampingScrollSimulation) String() string { return "ClampingScrollSimulation" }
