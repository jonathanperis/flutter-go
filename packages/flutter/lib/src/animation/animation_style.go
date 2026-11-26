package animation

// Port of packages/flutter/lib/src/animation/animation_style.dart.
//
// AnimationStyle is immutable and Diagnosticable: it is used by pointer
// (DiagnosticableMixin needs self), built with NewAnimationStyle, and
// compared with Equal (Dart's ==; Go's == on the pointers is identical()).

import (
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AnimationStyle is used to override the default parameters of an
// animation (curves and durations); nil fields fall back to the widget's
// defaults.
type AnimationStyle struct {
	foundation.DiagnosticableMixin
	// Curve is the curve of the forward animation (nil: none given).
	Curve Curve
	// Duration is the duration of the forward animation (nil: none given).
	Duration *time.Duration
	// ReverseCurve is the curve of the reverse animation.
	ReverseCurve Curve
	// ReverseDuration is the duration of the reverse animation.
	ReverseDuration *time.Duration
}

// AnimationStyleOptions are AnimationStyle's named parameters.
type AnimationStyleOptions struct {
	Curve           Curve
	Duration        *time.Duration
	ReverseCurve    Curve
	ReverseDuration *time.Duration
}

// NewAnimationStyle creates an animation style.
func NewAnimationStyle(opts ...AnimationStyleOptions) *AnimationStyle {
	var o AnimationStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := &AnimationStyle{Curve: o.Curve, Duration: o.Duration, ReverseCurve: o.ReverseCurve,
		ReverseDuration: o.ReverseDuration}
	s.SetSelf(s)
	return s
}

// AnimationStyleNoAnimation is AnimationStyle.noAnimation: a style that
// disables animations (zero durations).
var AnimationStyleNoAnimation = func() *AnimationStyle {
	zero := time.Duration(0)
	return NewAnimationStyle(AnimationStyleOptions{Duration: &zero, ReverseDuration: &zero})
}()

// CopyWithOptions are AnimationStyle.CopyWith's named parameters (nil:
// keep the current value).
type CopyWithOptions struct {
	Curve           Curve
	Duration        *time.Duration
	ReverseCurve    Curve
	ReverseDuration *time.Duration
}

// CopyWith creates a new AnimationStyle based on this one, with the given
// values replaced.
func (s *AnimationStyle) CopyWith(opts ...CopyWithOptions) *AnimationStyle {
	var o CopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	n := AnimationStyleOptions{Curve: s.Curve, Duration: s.Duration, ReverseCurve: s.ReverseCurve,
		ReverseDuration: s.ReverseDuration}
	if o.Curve != nil {
		n.Curve = o.Curve
	}
	if o.Duration != nil {
		n.Duration = o.Duration
	}
	if o.ReverseCurve != nil {
		n.ReverseCurve = o.ReverseCurve
	}
	if o.ReverseDuration != nil {
		n.ReverseDuration = o.ReverseDuration
	}
	return NewAnimationStyle(n)
}

// Merge returns a copy of this style with other's non-null values (this
// style itself when other is nil).
func (s *AnimationStyle) Merge(other *AnimationStyle) *AnimationStyle {
	if other == nil {
		return s
	}
	return s.CopyWith(CopyWithOptions{Curve: other.Curve, Duration: other.Duration,
		ReverseCurve: other.ReverseCurve, ReverseDuration: other.ReverseDuration})
}

// AnimationStyleLerp is AnimationStyle.lerp: a linear interpolation
// between two styles (nil is null). Curves are blended, durations
// interpolated (a null one counts as zero).
func AnimationStyleLerp(a, b *AnimationStyle, t float64) *AnimationStyle {
	if a == b {
		return a
	}
	var ac, bc, arc, brc Curve
	var ad, bd, ard, brd *time.Duration
	if a != nil {
		ac, ad, arc, ard = a.Curve, a.Duration, a.ReverseCurve, a.ReverseDuration
	}
	if b != nil {
		bc, bd, brc, brd = b.Curve, b.Duration, b.ReverseCurve, b.ReverseDuration
	}
	return NewAnimationStyle(AnimationStyleOptions{
		Curve:           lerpCurve(ac, bc, t),
		Duration:        lerpDurationPtr(ad, bd, t),
		ReverseCurve:    lerpCurve(arc, brc, t),
		ReverseDuration: lerpDurationPtr(ard, brd, t),
	})
}

// lerpCurve is _lerp with _LerpedCurve.new.
func lerpCurve(a, b Curve, t float64) Curve {
	if curvesEqual(a, b) || t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	return newLerpedCurve(a, b, t)
}

// lerpDurationPtr is _lerp with _lerpDuration.
func lerpDurationPtr(a, b *time.Duration, t float64) *time.Duration {
	if durationsEqual(a, b) || t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	var au, bu int64
	if a != nil {
		au = a.Microseconds()
	}
	if b != nil {
		bu = b.Microseconds()
	}
	d := time.Duration(dartRound(float64(float64(au)*(1.0-t))+float64(float64(bu)*t))) * time.Microsecond
	return &d
}

func durationsEqual(a, b *time.Duration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// curvesEqual is Dart's == on curves: identity, except _LerpedCurve's
// field-wise ==.
func curvesEqual(a, b Curve) bool {
	if la, ok := a.(*_LerpedCurve); ok {
		lb, ok := b.(*_LerpedCurve)
		return ok && curvesEqual(lb.first, la.first) && curvesEqual(lb.second, la.second) && lb.t == la.t
	}
	return a == b
}

// Equal is Dart's ==: same curves (by identity) and durations.
func (s *AnimationStyle) Equal(other *AnimationStyle) bool {
	if s == other {
		return true
	}
	if other == nil || s == nil {
		return false
	}
	return curvesEqual(other.Curve, s.Curve) && durationsEqual(other.Duration, s.Duration) &&
		curvesEqual(other.ReverseCurve, s.ReverseCurve) && durationsEqual(other.ReverseDuration, s.ReverseDuration)
}

// DebugFillProperties adds the four properties (hidden when null).
func (s *AnimationStyle) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.DiagnosticableMixin.DebugFillProperties(properties)
	nullDefault := foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}
	properties.Add(foundation.NewDiagnosticsProperty("curve", s.Curve, nullDefault))
	properties.Add(foundation.NewDiagnosticsProperty("duration", durationOrNull(s.Duration), nullDefault))
	properties.Add(foundation.NewDiagnosticsProperty("reverseCurve", s.ReverseCurve, nullDefault))
	properties.Add(foundation.NewDiagnosticsProperty("reverseDuration", durationOrNull(s.ReverseDuration), nullDefault))
}

// durationOrNull is a Duration? property value: the duration, or null (a
// nil any) when unset.
func durationOrNull(d *time.Duration) any {
	if d == nil {
		return nil
	}
	return *d
}

// _LerpedCurve is a blend of two curves (null is Curves.linear).
type _LerpedCurve struct {
	CurveBase
	first, second Curve
	t             float64
}

func newLerpedCurve(a, b Curve, t float64) *_LerpedCurve {
	c := &_LerpedCurve{first: a, second: b, t: t}
	if c.first == nil {
		c.first = Curves.Linear
	}
	if c.second == nil {
		c.second = Curves.Linear
	}
	c.SetSelf(c)
	return c
}

// Transform overrides Curve.transform itself (no 0/1 shortcut).
func (c *_LerpedCurve) Transform(t float64) float64 {
	a := c.first.Transform(t)
	b := c.second.Transform(t)
	return float64(a*(1.0-c.t)) + float64(b*c.t)
}

func (c *_LerpedCurve) String() string {
	return "_LerpedCurve(" + c.first.String() + ", " + c.second.String() + ", t: " + dart.DoubleToString(c.t) + ")"
}
