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

// Tolerance is Flutter's Tolerance (distance and velocity epsilons).
type Tolerance struct{ Distance, Velocity float64 }

func nearZero(a, eps float64) bool { return a > -eps && a < eps }

// exp mirrors Dart's math.pow(math.e, x), which Flutter's physics uses.
func exp(x float64) float64 { return math.Pow(math.E, x) }

func newtons(initial, target float64, f, df func(float64) float64, iterations int) float64 {
	guess := initial
	for i := 0; i < iterations; i++ {
		guess = guess - (f(guess)-target)/df(guess)
	}
	return guess
}

// FrictionSimulation is Flutter's FrictionSimulation, including its
// optional constant deceleration (used by macOS's "fast" scrolling).
type FrictionSimulation struct {
	drag, dragLog, x, v, constDecel, finalTime float64
	Tol                                        Tolerance
}

func NewFrictionSimulation(drag, position, velocity, constantDeceleration float64, tol Tolerance) *FrictionSimulation {
	s := &FrictionSimulation{drag: drag, dragLog: math.Log(drag), x: position, v: velocity, Tol: tol,
		constDecel: constantDeceleration * sign(velocity), finalTime: math.Inf(1)}
	// As in Flutter: 10 Newton iterations on dx(t) = 0. With no constant
	// deceleration dx never reaches 0 and this lands near t ≈ 5 s, an artifact
	// copied here so x() matches Flutter's exactly.
	s.finalTime = newtons(0, 0, s.Dx, func(t float64) float64 {
		return s.v*math.Pow(s.drag, t)*s.dragLog - s.constDecel
	}, 10)
	return s
}

func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func (s *FrictionSimulation) X(t float64) float64 {
	if t > s.finalTime {
		return s.FinalX()
	}
	return s.x + s.v*math.Pow(s.drag, t)/s.dragLog - s.v/s.dragLog - (s.constDecel/2)*t*t
}

func (s *FrictionSimulation) Dx(t float64) float64 {
	if t > s.finalTime {
		return 0
	}
	return s.v*math.Pow(s.drag, t) - s.constDecel*t
}

func (s *FrictionSimulation) FinalX() float64 {
	if s.constDecel == 0 {
		return s.x - s.v/s.dragLog
	}
	return s.X(s.finalTime)
}

func (s *FrictionSimulation) TimeAtX(x float64) float64 {
	if x == s.x {
		return 0
	}
	fx := s.FinalX()
	if s.v == 0 || (s.v > 0 && (x < s.x || x > fx)) || (s.v < 0 && (x > s.x || x < fx)) {
		return math.Inf(1)
	}
	return newtons(0, x, s.X, s.Dx, 10)
}

func (s *FrictionSimulation) IsDone(t float64) bool { return math.Abs(s.Dx(t)) < s.Tol.Velocity }

// SpringDescription is Flutter's SpringDescription.
type SpringDescription struct{ Mass, Stiffness, Damping float64 }

func SpringWithDampingRatio(mass, stiffness, ratio float64) SpringDescription {
	return SpringDescription{mass, stiffness, ratio * 2.0 * math.Sqrt(mass*stiffness)}
}

type springSolution interface {
	x(t float64) float64
	dx(t float64) float64
}

type criticalSolution struct{ r, c1, c2 float64 }

func (s criticalSolution) x(t float64) float64 { return (s.c1 + s.c2*t) * exp(s.r*t) }
func (s criticalSolution) dx(t float64) float64 {
	p := exp(s.r * t)
	return s.r*(s.c1+s.c2*t)*p + s.c2*p
}

type overdampedSolution struct{ r1, r2, c1, c2 float64 }

func (s overdampedSolution) x(t float64) float64 { return s.c1*exp(s.r1*t) + s.c2*exp(s.r2*t) }
func (s overdampedSolution) dx(t float64) float64 {
	return s.c1*s.r1*exp(s.r1*t) + s.c2*s.r2*exp(s.r2*t)
}

type underdampedSolution struct{ w, r, c1, c2 float64 }

func (s underdampedSolution) x(t float64) float64 {
	return exp(s.r*t) * (s.c1*math.Cos(s.w*t) + s.c2*math.Sin(s.w*t))
}
func (s underdampedSolution) dx(t float64) float64 {
	p, c, sn := exp(s.r*t), math.Cos(s.w*t), math.Sin(s.w*t)
	return p*(s.c2*s.w*c-s.c1*s.w*sn) + s.r*p*(s.c2*sn+s.c1*c)
}

func newSpringSolution(sp SpringDescription, distance, velocity float64) springSolution {
	cmk := sp.Damping*sp.Damping - 4*sp.Mass*sp.Stiffness
	switch {
	case cmk > 0:
		r1 := (-sp.Damping - math.Sqrt(cmk)) / (2.0 * sp.Mass)
		r2 := (-sp.Damping + math.Sqrt(cmk)) / (2.0 * sp.Mass)
		c2 := (velocity - r1*distance) / (r2 - r1)
		return overdampedSolution{r1, r2, distance - c2, c2}
	case cmk < 0:
		w := math.Sqrt(4.0*sp.Mass*sp.Stiffness-sp.Damping*sp.Damping) / (2.0 * sp.Mass)
		r := -(sp.Damping / 2.0 / sp.Mass)
		return underdampedSolution{w, r, distance, (velocity - r*distance) / w}
	}
	r := -sp.Damping / (2.0 * sp.Mass)
	return criticalSolution{r, distance, velocity - r*distance}
}

// ScrollSpringSimulation is Flutter's: a spring towards end that snaps to
// end once within tolerance.
type ScrollSpringSimulation struct {
	end float64
	sol springSolution
	Tol Tolerance
}

func NewScrollSpringSimulation(sp SpringDescription, start, end, velocity float64, tol Tolerance) *ScrollSpringSimulation {
	return &ScrollSpringSimulation{end, newSpringSolution(sp, start-end, velocity), tol}
}

func (s *ScrollSpringSimulation) IsDone(t float64) bool {
	return nearZero(s.sol.x(t), s.Tol.Distance) && nearZero(s.sol.dx(t), s.Tol.Velocity)
}
func (s *ScrollSpringSimulation) X(t float64) float64 {
	if s.IsDone(t) {
		return s.end
	}
	return s.end + s.sol.x(t)
}
func (s *ScrollSpringSimulation) Dx(t float64) float64 { return s.sol.dx(t) }

// BouncingScrollSimulation is Flutter's: friction inside the range, handing
// over to a spring when it would cross an edge (or a spring straight away
// when starting out of range).
type BouncingScrollSimulation struct {
	leading, trailing float64
	spring            SpringDescription
	friction          *FrictionSimulation
	springSim         interface {
		X(float64) float64
		IsDone(float64) bool
	}
	springTime float64
	tol        Tolerance
}

const maxSpringTransferVelocity = 5000.0

func NewBouncingScrollSimulation(sp SpringDescription, position, velocity, leading, trailing, constDecel float64, tol Tolerance) *BouncingScrollSimulation {
	s := &BouncingScrollSimulation{leading: leading, trailing: trailing, spring: sp, tol: tol}
	switch {
	case position < leading:
		s.springSim = NewScrollSpringSimulation(sp, position, leading, velocity, tol)
		s.springTime = math.Inf(-1)
	case position > trailing:
		s.springSim = NewScrollSpringSimulation(sp, position, trailing, velocity, tol)
		s.springTime = math.Inf(-1)
	default:
		s.friction = NewFrictionSimulation(0.135, position, velocity, constDecel, tol)
		finalX := s.friction.FinalX()
		switch {
		case velocity > 0 && finalX > trailing:
			s.springTime = s.friction.TimeAtX(trailing)
			s.springSim = NewScrollSpringSimulation(sp, trailing, trailing,
				math.Min(s.friction.Dx(s.springTime), maxSpringTransferVelocity), tol)
		case velocity < 0 && finalX < leading:
			s.springTime = s.friction.TimeAtX(leading)
			s.springSim = NewScrollSpringSimulation(sp, leading, leading,
				math.Min(s.friction.Dx(s.springTime), maxSpringTransferVelocity), tol)
		default:
			s.springTime = math.Inf(1)
		}
	}
	return s
}

func (s *BouncingScrollSimulation) at(t float64) (func(float64) float64, func(float64) bool, float64) {
	if t > s.springTime {
		off := 0.0
		if !math.IsInf(s.springTime, 0) {
			off = s.springTime
		}
		return s.springSim.X, s.springSim.IsDone, off
	}
	return s.friction.X, s.friction.IsDone, 0
}

func (s *BouncingScrollSimulation) X(t float64) float64 {
	x, _, off := s.at(t)
	return x(t - off)
}

func (s *BouncingScrollSimulation) IsDone(t float64) bool {
	_, done, off := s.at(t)
	return done(t - off)
}
