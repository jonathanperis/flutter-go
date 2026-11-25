package physics

import (
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	"math"
	"time"
)

// SpringDescription is a spring's mass, stiffness and damping.
type SpringDescription struct {
	Mass, Stiffness, Damping float64
}

// NewSpringDescriptionWithDampingRatio is SpringDescription.withDampingRatio;
// Flutter's default ratio is 1.0 (critically damped).
func NewSpringDescriptionWithDampingRatio(mass, stiffness, ratio float64) SpringDescription {
	return SpringDescription{mass, stiffness, ratio * 2.0 * math.Sqrt(mass*stiffness)}
}

// NewSpringDescriptionWithDurationAndBounce is
// SpringDescription.withDurationAndBounce; Flutter's defaults are 500 ms and
// bounce 0.
func NewSpringDescriptionWithDurationAndBounce(duration time.Duration, bounce float64) SpringDescription {
	if duration.Milliseconds() <= 0 {
		panic("physics: Duration must be positive")
	}
	durationInSeconds := float64(duration.Milliseconds()) / 1000
	const mass = 1.0
	stiffness := (4 * math.Pi * math.Pi * mass) / dart.Pow(durationInSeconds, 2)
	dampingRatio := 1 / (bounce + 1)
	if bounce > 0 {
		dampingRatio = 1.0 - bounce
	}
	return SpringDescription{mass, stiffness, dampingRatio * 2.0 * math.Sqrt(mass*stiffness)}
}

// Duration is the spring's perceived duration, rounded to milliseconds.
func (s SpringDescription) Duration() time.Duration {
	secs := math.Sqrt((4 * math.Pi * math.Pi * s.Mass) / s.Stiffness)
	return time.Duration(math.Round(secs*1000)) * time.Millisecond
}

// Bounce is the spring's bounciness, the inverse of withDurationAndBounce.
func (s SpringDescription) Bounce() float64 {
	ratio := s.Damping / (2.0 * math.Sqrt(s.Mass*s.Stiffness))
	if ratio < 1.0 {
		return 1.0 - ratio
	}
	return (1 / ratio) - 1
}

func (s SpringDescription) String() string {
	return "SpringDescription(mass: " + fixed1(s.Mass) + ", stiffness: " + fixed1(s.Stiffness) +
		", damping: " + fixed1(s.Damping) + ")"
}

// SpringType is the kind of motion a spring makes.
type SpringType int

const (
	SpringTypeCriticallyDamped SpringType = iota
	SpringTypeUnderDamped
	SpringTypeOverDamped
)

func (t SpringType) String() string {
	return [...]string{"SpringType.criticallyDamped", "SpringType.underDamped", "SpringType.overDamped"}[t]
}

// SpringSimulation moves a particle attached to a spring from start toward
// end.
type SpringSimulation struct {
	SimulationBase
	endPosition float64
	solution    springSolution
	snapToEnd   bool
}

// SpringSimulationOptions are SpringSimulation's optional arguments.
type SpringSimulationOptions struct {
	SnapToEnd bool
	Tolerance *Tolerance // default DefaultTolerance
}

func NewSpringSimulation(spring SpringDescription, start, end, velocity float64, opts ...SpringSimulationOptions) *SpringSimulation {
	s := &SpringSimulation{}
	s.init(spring, start, end, velocity, opts...)
	return s
}

func (s *SpringSimulation) init(spring SpringDescription, start, end, velocity float64, opts ...SpringSimulationOptions) {
	s.SimulationBase = NewSimulationBase()
	for _, o := range opts {
		s.snapToEnd = o.SnapToEnd
		if o.Tolerance != nil {
			s.SimulationBase = NewSimulationBase(*o.Tolerance)
		}
	}
	s.endPosition = end
	s.solution = newSpringSolution(spring, start-end, velocity)
}

// Type is the kind of spring motion.
func (s *SpringSimulation) Type() SpringType { return s.solution.springType() }

func (s *SpringSimulation) X(t float64) float64 {
	if s.snapToEnd && s.IsDone(t) {
		return s.endPosition
	}
	return s.endPosition + s.solution.x(t)
}

func (s *SpringSimulation) Dx(t float64) float64 {
	if s.snapToEnd && s.IsDone(t) {
		return 0
	}
	return s.solution.dx(t)
}

func (s *SpringSimulation) IsDone(t float64) bool {
	tol := s.Tolerance()
	return NearZero(s.solution.x(t), tol.Distance) && NearZero(s.solution.dx(t), tol.Velocity)
}

func (s *SpringSimulation) String() string {
	return "SpringSimulation(end: " + fixed1(s.endPosition) + ", " + s.Type().String() + ")"
}

// ScrollSpringSimulation is a SpringSimulation whose position snaps to the
// end once it is done (its velocity does not).
type ScrollSpringSimulation struct{ SpringSimulation }

// ScrollSpringSimulationOptions are ScrollSpringSimulation's optional
// arguments.
type ScrollSpringSimulationOptions struct {
	Tolerance *Tolerance
}

func NewScrollSpringSimulation(spring SpringDescription, start, end, velocity float64, opts ...ScrollSpringSimulationOptions) *ScrollSpringSimulation {
	s := &ScrollSpringSimulation{}
	o := SpringSimulationOptions{}
	for _, x := range opts {
		o.Tolerance = x.Tolerance
	}
	s.init(spring, start, end, velocity, o)
	return s
}

func (s *ScrollSpringSimulation) X(t float64) float64 {
	if s.IsDone(t) {
		return s.endPosition
	}
	return s.SpringSimulation.X(t)
}

type springSolution interface {
	x(t float64) float64
	dx(t float64) float64
	springType() SpringType
}

func newSpringSolution(spring SpringDescription, initialPosition, initialVelocity float64) springSolution {
	cmk := float64(spring.Damping*spring.Damping) - float64(4*spring.Mass*spring.Stiffness)
	switch {
	case cmk > 0.0:
		r1 := (-spring.Damping - math.Sqrt(cmk)) / (2.0 * spring.Mass)
		r2 := (-spring.Damping + math.Sqrt(cmk)) / (2.0 * spring.Mass)
		c2 := (initialVelocity - float64(r1*initialPosition)) / (r2 - r1)
		return overdampedSolution{r1, r2, initialPosition - c2, c2}
	case cmk < 0.0:
		w := math.Sqrt(float64(4.0*spring.Mass*spring.Stiffness)-float64(spring.Damping*spring.Damping)) / (2.0 * spring.Mass)
		r := -(spring.Damping / 2.0 / spring.Mass)
		return underdampedSolution{w, r, initialPosition, (initialVelocity - float64(r*initialPosition)) / w}
	}
	r := -spring.Damping / (2.0 * spring.Mass)
	return criticalSolution{r, initialPosition, initialVelocity - float64((r * initialPosition))}
}

type criticalSolution struct{ r, c1, c2 float64 }

func (s criticalSolution) x(t float64) float64 { return (s.c1 + float64(s.c2*t)) * powE(s.r*t) }
func (s criticalSolution) dx(t float64) float64 {
	p := powE(s.r * t)
	return float64(s.r*(s.c1+float64(s.c2*t))*p) + float64(s.c2*p)
}
func (criticalSolution) springType() SpringType { return SpringTypeCriticallyDamped }

type overdampedSolution struct{ r1, r2, c1, c2 float64 }

func (s overdampedSolution) x(t float64) float64 {
	return float64(s.c1*powE(s.r1*t)) + float64(s.c2*powE(s.r2*t))
}
func (s overdampedSolution) dx(t float64) float64 {
	return float64(s.c1*s.r1*powE(s.r1*t)) + float64(s.c2*s.r2*powE(s.r2*t))
}
func (overdampedSolution) springType() SpringType { return SpringTypeOverDamped }

type underdampedSolution struct{ w, r, c1, c2 float64 }

func (s underdampedSolution) x(t float64) float64 {
	return powE(s.r*t) * (float64(s.c1*dart.Cos(s.w*t)) + float64(s.c2*dart.Sin(s.w*t)))
}
func (s underdampedSolution) dx(t float64) float64 {
	p, c, sn := powE(s.r*t), dart.Cos(s.w*t), dart.Sin(s.w*t)
	return float64(p*(float64(s.c2*s.w*c)-float64(s.c1*s.w*sn))) + float64(s.r*p*(float64(s.c2*sn)+float64(s.c1*c)))
}
func (underdampedSolution) springType() SpringType { return SpringTypeUnderDamped }

// String matches Flutter's debug-mode objectRuntimeType, which prints the
// runtime type.
func (s *ScrollSpringSimulation) String() string {
	return "ScrollSpringSimulation(end: " + fixed1(s.endPosition) + ", " + s.Type().String() + ")"
}
