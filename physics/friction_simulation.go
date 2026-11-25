package physics

import "math"

func newtonsMethod(initialGuess, target float64, f, df func(float64) float64, iterations int) float64 {
	guess := initialGuess
	for i := 0; i < iterations; i++ {
		guess = guess - (f(guess)-target)/df(guess)
	}
	return guess
}

// FrictionSimulation is a particle slowed by drag (velocity multiplied by
// drag every second) and an optional constant deceleration.
type FrictionSimulation struct {
	SimulationBase
	drag, dragLog, x, v, constantDeceleration, finalTime float64
}

// FrictionSimulationOptions are the optional arguments of Flutter's
// FrictionSimulation constructor.
type FrictionSimulationOptions struct {
	Tolerance            *Tolerance // default DefaultTolerance
	ConstantDeceleration float64    // default 0
}

func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return v // keeps -0.0 and NaN, like Dart's double.sign
}

func NewFrictionSimulation(drag, position, velocity float64, opts ...FrictionSimulationOptions) *FrictionSimulation {
	s := &FrictionSimulation{}
	s.init(drag, position, velocity, opts...)
	return s
}

func (s *FrictionSimulation) init(drag, position, velocity float64, opts ...FrictionSimulationOptions) {
	o := FrictionSimulationOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	s.SimulationBase = NewSimulationBase()
	if o.Tolerance != nil {
		s.SimulationBase = NewSimulationBase(*o.Tolerance)
	}
	s.drag, s.dragLog, s.x, s.v = drag, math.Log(drag), position, velocity
	s.constantDeceleration = o.ConstantDeceleration * sign(velocity)
	s.finalTime = math.Inf(1) // must be infinite while Newton's method calls Dx
	s.finalTime = newtonsMethod(0, 0, s.Dx, func(t float64) float64 {
		return (s.v * math.Pow(s.drag, t) * s.dragLog) - s.constantDeceleration
	}, 10)
}

// NewFrictionSimulationThrough is FrictionSimulation.through: a simulation
// that passes startPosition at startVelocity and endPosition at endVelocity.
func NewFrictionSimulationThrough(startPosition, endPosition, startVelocity, endVelocity float64) *FrictionSimulation {
	if !(startVelocity == 0 || endVelocity == 0 || sign(startVelocity) == sign(endVelocity)) ||
		math.Abs(startVelocity) < math.Abs(endVelocity) || sign(endPosition-startPosition) != sign(startVelocity) {
		panic("physics: FrictionSimulation.through arguments are inconsistent")
	}
	tol := NewTolerance(ToleranceOptions{Velocity: ptr(math.Abs(endVelocity))})
	return NewFrictionSimulation(dragFor(startPosition, endPosition, startVelocity, endVelocity),
		startPosition, startVelocity, FrictionSimulationOptions{Tolerance: &tol})
}

func ptr[T any](v T) *T { return &v }

func dragFor(startPosition, endPosition, startVelocity, endVelocity float64) float64 {
	return math.Pow(math.E, (startVelocity-endVelocity)/(startPosition-endPosition))
}

func (s *FrictionSimulation) X(t float64) float64 {
	if t > s.finalTime {
		return s.FinalX()
	}
	return s.x + s.v*math.Pow(s.drag, t)/s.dragLog - s.v/s.dragLog - ((s.constantDeceleration / 2) * t * t)
}

func (s *FrictionSimulation) Dx(t float64) float64 {
	if t > s.finalTime {
		return 0
	}
	return s.v*math.Pow(s.drag, t) - s.constantDeceleration*t
}

// FinalX is where the particle comes to rest.
func (s *FrictionSimulation) FinalX() float64 {
	if s.constantDeceleration == 0 {
		return s.x - s.v/s.dragLog
	}
	return s.X(s.finalTime)
}

// TimeAtX is when the particle reaches x, or +Inf if it never does.
func (s *FrictionSimulation) TimeAtX(x float64) float64 {
	if x == s.x {
		return 0
	}
	if s.v == 0 || func() bool {
		if s.v > 0 {
			return x < s.x || x > s.FinalX()
		}
		return x > s.x || x < s.FinalX()
	}() {
		return math.Inf(1)
	}
	return newtonsMethod(0, x, s.X, s.Dx, 10)
}

func (s *FrictionSimulation) IsDone(t float64) bool {
	return math.Abs(s.Dx(t)) < s.Tolerance().Velocity
}

func (s *FrictionSimulation) String() string {
	return "FrictionSimulation(cₓ: " + fixed1(s.drag) + ", x₀: " + fixed1(s.x) + ", dx₀: " + fixed1(s.v) + ")"
}

// BoundedFrictionSimulation is a FrictionSimulation clamped to [minX, maxX]
// that is also done when it reaches either bound.
type BoundedFrictionSimulation struct {
	FrictionSimulation
	minX, maxX float64
}

func NewBoundedFrictionSimulation(drag, position, velocity, minX, maxX float64) *BoundedFrictionSimulation {
	if clampDouble(position, minX, maxX) != position {
		panic("physics: BoundedFrictionSimulation position is out of bounds")
	}
	s := &BoundedFrictionSimulation{minX: minX, maxX: maxX}
	s.init(drag, position, velocity)
	return s
}

func (s *BoundedFrictionSimulation) X(t float64) float64 {
	return clampDouble(s.FrictionSimulation.X(t), s.minX, s.maxX)
}

func (s *BoundedFrictionSimulation) IsDone(t float64) bool {
	d := s.Tolerance().Distance
	return s.FrictionSimulation.IsDone(t) || math.Abs(s.X(t)-s.minX) < d || math.Abs(s.X(t)-s.maxX) < d
}

func (s *BoundedFrictionSimulation) String() string {
	return "BoundedFrictionSimulation(cₓ: " + fixed1(s.drag) + ", x₀: " + fixed1(s.x) + ", dx₀: " + fixed1(s.v) +
		", x: " + fixed1(s.minX) + ".." + fixed1(s.maxX) + ")"
}
