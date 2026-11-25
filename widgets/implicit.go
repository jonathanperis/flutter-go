package widgets

import (
	"time"

	"github.com/jonathanperis/flutter-go/animation"
	r "github.com/jonathanperis/flutter-go/rendering"
)

// AnimatedBox animates its size and color when they change, like Flutter's
// AnimatedContainer with width, height and color. When a new target arrives
// mid-animation it starts from the current value (Flutter's
// ImplicitlyAnimatedWidget behavior).
type AnimatedBox struct {
	Width, Height float64
	Color         r.Color
	Duration      time.Duration
	Curve         animation.Curve // nil means Linear
	Child         Widget
}

func (AnimatedBox) CreateState() State { return &animatedBoxState{} }

type boxValues struct {
	w, h float64
	c    r.Color
}

type animatedBoxState struct {
	StateOf[AnimatedBox]
	ctrl       *animation.Controller
	begin, end boxValues
}

func targetOf(w AnimatedBox) boxValues { return boxValues{w.Width, w.Height, w.Color} }

func (s *animatedBoxState) InitState() {
	s.ctrl = animation.NewController(s.Context().Scheduler(), s.Widget().Duration)
	s.ctrl.AddListener(func() { s.SetState(nil) })
	s.begin, s.end = targetOf(s.Widget()), targetOf(s.Widget())
}

func (s *animatedBoxState) DidUpdateWidget(old Widget) {
	w := s.Widget()
	s.ctrl.Duration = w.Duration
	if targetOf(w) != s.end {
		s.begin, s.end = s.current(), targetOf(w)
		s.ctrl.SetValue(0)
		s.ctrl.Forward()
	}
}

func (s *animatedBoxState) Dispose() { s.ctrl.Dispose() }

// current evaluates the tweens at the curved animation value.
func (s *animatedBoxState) current() boxValues {
	t := s.ctrl.Value()
	if c := s.Widget().Curve; c != nil {
		t = c.Transform(t)
	}
	l := animation.LerpDouble
	return boxValues{
		l(s.begin.w, s.end.w, t), l(s.begin.h, s.end.h, t),
		r.Color{R: l(s.begin.c.R, s.end.c.R, t), G: l(s.begin.c.G, s.end.c.G, t),
			B: l(s.begin.c.B, s.end.c.B, t), A: l(s.begin.c.A, s.end.c.A, t)},
	}
}

func (s *animatedBoxState) Build(*Element) Widget {
	v := s.current()
	return SizedBox{Width: F(v.w), Height: F(v.h), Child: ColoredBox{Color: v.c, Child: s.Widget().Child}}
}
