package animation

// Port of packages/flutter/lib/src/animation/curves.dart (Flutter 3.50.1).
//
// # Overridable methods
//
// Dart's ParametricCurve.transform calls transformInternal, which every
// subclass overrides, and Curve.transform adds the 0/1 shortcut on top. As
// everywhere in flutter-go (API_MAPPING "Mixins and overridable methods"),
// the embeddable bases (ParametricCurveBase, CurveBase, Curve2DBase) keep a
// pointer to the outer object, self, and call overridable methods through
// it. The constructors here set it; a curve defined in another package
// embeds CurveBase (or Curve2DBase) and calls SetSelf once:
//
//	type MyCurve struct{ animation.CurveBase }
//
//	func NewMyCurve() *MyCurve { c := &MyCurve{}; c.SetSelf(c); return c }
//	func (c *MyCurve) TransformInternal(t float64) float64 { return t * t }
//
// Curves are used by pointer. Dart's const curves are values of Curves.
//
// # Floating point
//
// The arithmetic matches Flutter's bit for bit (see the curves difftest):
// math.pow and math.sin are dart.Pow and dart.Sin (libm plus the VM's
// special cases), products that Go could fuse into a multiply-add are
// rounded explicitly, and math.Random is dart.Random (the VM's generator).

import (
	"iter"
	"math"
	"slices"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// ParametricCurve is an abstract parametric curve: Transform maps t in
// [0, 1] to a T. Implementations embed ParametricCurveBase (or CurveBase /
// Curve2DBase) and override TransformInternal.
type ParametricCurve[T any] interface {
	// Transform returns the value of the curve at t, asserting that t is in
	// [0, 1]. Subclasses should override TransformInternal instead.
	Transform(t float64) T
	// TransformInternal returns the value of the curve at t (Dart's
	// @protected transformInternal).
	TransformInternal(t float64) T
	String() string
}

const unboundCurve = "animation: %s used before SetSelf: build curves with their constructors, " +
	"or call SetSelf(outer) once on a type that embeds it"

// ParametricCurveBase is the embeddable implementation of ParametricCurve.
type ParametricCurveBase[T any] struct{ self ParametricCurve[T] }

// SetSelf points the base at the curve that embeds it.
func (c *ParametricCurveBase[T]) SetSelf(self ParametricCurve[T]) { c.self = self }

func (c *ParametricCurveBase[T]) outer() ParametricCurve[T] {
	if c.self == nil {
		panic(strings.Replace(unboundCurve, "%s", "ParametricCurveBase", 1))
	}
	return c.self
}

// Transform asserts that t is in [0, 1] and returns TransformInternal(t).
func (c *ParametricCurveBase[T]) Transform(t float64) T {
	if !(t >= 0.0 && t <= 1.0) {
		panic(dart.NewAssertionError("parametric value " + dart.DoubleToString(t) + " is outside of [0, 1] range."))
	}
	return c.outer().TransformInternal(t)
}

// TransformInternal panics with UnimplementedError, as Dart's does:
// subclasses override it.
func (c *ParametricCurveBase[T]) TransformInternal(float64) T {
	panic(dart.UnimplementedError{})
}

func (c *ParametricCurveBase[T]) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "ParametricCurve")
}

// Curve is an easing curve: a mapping of the unit interval to the unit
// interval (Dart's abstract class Curve). Implementations embed CurveBase.
type Curve interface {
	ParametricCurve[float64]
	// Flipped is this curve flipped horizontally and vertically.
	Flipped() Curve
}

// CurveBase is the embeddable implementation of Curve.
type CurveBase struct{ ParametricCurveBase[float64] }

// SetSelf points the base at the curve that embeds it.
func (c *CurveBase) SetSelf(self Curve) { c.ParametricCurveBase.SetSelf(self) }

// Transform returns t itself for 0 and 1 (every Curve must map them to
// themselves), and otherwise asserts t is in range and returns
// TransformInternal(t).
func (c *CurveBase) Transform(t float64) float64 {
	if t == 0.0 || t == 1.0 {
		return t
	}
	return c.ParametricCurveBase.Transform(t)
}

// Flipped is FlippedCurve(this).
func (c *CurveBase) Flipped() Curve { return NewFlippedCurve(c.outer().(Curve)) }

// _Linear is Curves.linear.
type _Linear struct{ CurveBase }

func newLinear() *_Linear { c := &_Linear{}; c.SetSelf(c); return c }

func (c *_Linear) TransformInternal(t float64) float64 { return t }

// SawTooth is a sawtooth curve that repeats a given number of times over
// the unit interval.
type SawTooth struct {
	CurveBase
	// Count is the number of repetitions of the sawtooth pattern.
	Count int
}

// NewSawTooth is SawTooth(count).
func NewSawTooth(count int) *SawTooth { c := &SawTooth{Count: count}; c.SetSelf(c); return c }

func (c *SawTooth) TransformInternal(t float64) float64 {
	// The product is rounded before it is used twice: Go would otherwise fuse
	// it into `t - trunc(t)` (a multiply-subtract), which Dart never does.
	t = float64(t * float64(c.Count))
	return t - math.Trunc(t)
}

func (c *SawTooth) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "SawTooth") + "(" + itoa(c.Count) + ")"
}

// Interval is a curve that is 0.0 until Begin, then curved (according to
// Curve) from 0.0 at Begin to 1.0 at End, then remains 1.0 past End.
type Interval struct {
	CurveBase
	Begin float64
	End   float64
	Curve Curve
}

// IntervalOptions are Interval's optional arguments.
type IntervalOptions struct {
	// Curve defaults to Curves.Linear.
	Curve Curve
}

// NewInterval is Interval(begin, end, {curve = Curves.linear}).
func NewInterval(begin, end float64, opts ...IntervalOptions) *Interval {
	c := &Interval{Begin: begin, End: end, Curve: Curves.Linear}
	if len(opts) > 0 && opts[0].Curve != nil {
		c.Curve = opts[0].Curve
	}
	c.SetSelf(c)
	return c
}

func (c *Interval) TransformInternal(t float64) float64 {
	assert(c.Begin >= 0.0, nil)   // assert(begin >= 0.0)
	assert(c.Begin <= 1.0, nil)   // assert(begin <= 1.0)
	assert(c.End >= 0.0, nil)     // assert(end >= 0.0)
	assert(c.End <= 1.0, nil)     // assert(end <= 1.0)
	assert(c.End >= c.Begin, nil) // assert(end >= begin)
	t = ui.ClampDouble((t-c.Begin)/(c.End-c.Begin), 0.0, 1.0)
	if t == 0.0 || t == 1.0 {
		return t
	}
	return c.Curve.Transform(t)
}

func (c *Interval) String() string {
	s := foundation.ObjectRuntimeType(c.outer(), "Interval") + "(" + dart.DoubleToString(c.Begin) + "⋯" +
		dart.DoubleToString(c.End) + ")"
	if _, linear := c.Curve.(*_Linear); !linear {
		return s + "➩" + c.Curve.String()
	}
	return s
}

// Split is a curve that progresses according to BeginCurve until Split,
// then according to EndCurve.
type Split struct {
	CurveBase
	Split      float64
	BeginCurve Curve
	EndCurve   Curve
}

// SplitOptions are Split's optional arguments.
type SplitOptions struct {
	// BeginCurve defaults to Curves.Linear.
	BeginCurve Curve
	// EndCurve defaults to Curves.EaseOutCubic.
	EndCurve Curve
}

// NewSplit is Split(split, {beginCurve = Curves.linear, endCurve = Curves.easeOutCubic}).
func NewSplit(split float64, opts ...SplitOptions) *Split {
	c := &Split{Split: split, BeginCurve: Curves.Linear, EndCurve: Curves.EaseOutCubic}
	if len(opts) > 0 {
		if opts[0].BeginCurve != nil {
			c.BeginCurve = opts[0].BeginCurve
		}
		if opts[0].EndCurve != nil {
			c.EndCurve = opts[0].EndCurve
		}
	}
	c.SetSelf(c)
	return c
}

// Transform is Split's own transform (Flutter overrides transform, not
// transformInternal).
func (c *Split) Transform(t float64) float64 {
	assert(t >= 0.0 && t <= 1.0, nil)             // assert(t >= 0.0 && t <= 1.0)
	assert(c.Split >= 0.0 && c.Split <= 1.0, nil) // assert(split >= 0.0 && split <= 1.0)

	if t == 0.0 || t == 1.0 {
		return t
	}

	if t == c.Split {
		return c.Split
	}

	if t < c.Split {
		curveProgress := t / c.Split
		transformed := c.BeginCurve.Transform(curveProgress)
		return lerpDouble(0, c.Split, transformed)
	}
	curveProgress := (t - c.Split) / (1 - c.Split)
	transformed := c.EndCurve.Transform(curveProgress)
	return lerpDouble(c.Split, 1, transformed)
}

func (c *Split) String() string {
	return foundation.DescribeIdentity(c.outer()) + "(" + dart.DoubleToString(c.Split) + ", " +
		c.BeginCurve.String() + ", " + c.EndCurve.String() + ")"
}

// Threshold is a curve that is 0.0 until it hits the threshold, then jumps
// to 1.0.
type Threshold struct {
	CurveBase
	Threshold float64
}

// NewThreshold is Threshold(threshold).
func NewThreshold(threshold float64) *Threshold {
	c := &Threshold{Threshold: threshold}
	c.SetSelf(c)
	return c
}

func (c *Threshold) TransformInternal(t float64) float64 {
	assert(c.Threshold >= 0.0, nil) // assert(threshold >= 0.0)
	assert(c.Threshold <= 1.0, nil) // assert(threshold <= 1.0)
	if t < c.Threshold {
		return 0.0
	}
	return 1.0
}

// Cubic is a cubic polynomial mapping of the unit interval: the cubic
// Bézier from (0, 0) to (1, 1) with control points (A, B) and (C, D).
type Cubic struct {
	CurveBase
	A, B, C, D float64
}

// NewCubic is Cubic(a, b, c, d).
func NewCubic(a, b, c, d float64) *Cubic {
	x := &Cubic{A: a, B: b, C: c, D: d}
	x.SetSelf(x)
	return x
}

const cubicErrorBound = 0.001

func evaluateCubic(a, b, m float64) float64 {
	return float64(3*a*(1-m)*(1-m)*m) + float64(3*b*(1-m)*m*m) + float64(m*m*m)
}

func (c *Cubic) TransformInternal(t float64) float64 {
	if math.IsNaN(t) {
		panic(dart.NewArgumentErrorValue(t, "t", "must not be NaN"))
	}
	if t <= 0.0 {
		return 0.0
	}
	if t >= 1.0 {
		return 1.0
	}
	start := 0.0
	end := 1.0
	for {
		midpoint := (start + end) / 2
		estimate := evaluateCubic(c.A, c.C, midpoint)
		if math.Abs(t-estimate) < cubicErrorBound {
			return evaluateCubic(c.B, c.D, midpoint)
		}
		if estimate < t {
			start = midpoint
		} else {
			end = midpoint
		}
	}
}

func (c *Cubic) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "Cubic") + "(" + dart.ToStringAsFixed(c.A, 2) + ", " +
		dart.ToStringAsFixed(c.B, 2) + ", " + dart.ToStringAsFixed(c.C, 2) + ", " + dart.ToStringAsFixed(c.D, 2) + ")"
}

// ThreePointCubic is a cubic polynomial composed of two curves that share
// an edge point (Midpoint).
type ThreePointCubic struct {
	CurveBase
	A1, B1, Midpoint, A2, B2 ui.Offset
}

// NewThreePointCubic is ThreePointCubic(a1, b1, midpoint, a2, b2).
func NewThreePointCubic(a1, b1, midpoint, a2, b2 ui.Offset) *ThreePointCubic {
	c := &ThreePointCubic{A1: a1, B1: b1, Midpoint: midpoint, A2: a2, B2: b2}
	c.SetSelf(c)
	return c
}

func (c *ThreePointCubic) TransformInternal(t float64) float64 {
	firstCurve := t < c.Midpoint.Dx
	scaleX, scaleY, offsetX := 1.0-c.Midpoint.Dx, 1.0-c.Midpoint.Dy, c.Midpoint.Dx
	if firstCurve {
		scaleX, scaleY, offsetX = c.Midpoint.Dx, c.Midpoint.Dy, 0.0
	}
	scaledT := (t - offsetX) / scaleX
	if firstCurve {
		return NewCubic(
			c.A1.Dx/scaleX,
			c.A1.Dy/scaleY,
			c.B1.Dx/scaleX,
			c.B1.Dy/scaleY,
		).Transform(scaledT) * scaleY
	}
	return float64(NewCubic(
		(c.A2.Dx-c.Midpoint.Dx)/scaleX,
		(c.A2.Dy-c.Midpoint.Dy)/scaleY,
		(c.B2.Dx-c.Midpoint.Dx)/scaleX,
		(c.B2.Dy-c.Midpoint.Dy)/scaleY,
	).Transform(scaledT)*scaleY) + c.Midpoint.Dy
}

// String is Flutter's: in debug builds just the runtime type and a space
// (objectRuntimeType's optimized value, the full description, is only used
// in release builds).
func (c *ThreePointCubic) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "ThreePointCubic("+c.A1.String()+", "+c.B1.String()+", "+
		c.Midpoint.String()+", "+c.A2.String()+", "+c.B2.String()+")") + " "
}

// Curve2D is a parametric curve in two dimensions (Dart's abstract class
// Curve2D). Implementations embed Curve2DBase and override
// TransformInternal (and SamplingSeed).
type Curve2D interface {
	ParametricCurve[ui.Offset]
	// GenerateSamples samples the curve, concentrating samples where the
	// curvature is high.
	GenerateSamples(opts ...Curve2DGenerateSamplesOptions) iter.Seq[Curve2DSample]
	// SamplingSeed is the seed of the random generator GenerateSamples uses
	// (Dart's @protected samplingSeed getter).
	SamplingSeed() int
	// FindInverse returns the parametric value t whose X is x.
	FindInverse(x float64) float64
}

// Curve2DGenerateSamplesOptions are GenerateSamples' optional arguments;
// unset fields take Flutter's defaults (start 0.0, end 1.0, tolerance 1e-10).
type Curve2DGenerateSamplesOptions struct {
	Start     float64
	End       *float64
	Tolerance *float64
}

// Curve2DBase is the embeddable implementation of Curve2D.
type Curve2DBase struct{ ParametricCurveBase[ui.Offset] }

// SetSelf points the base at the curve that embeds it.
func (c *Curve2DBase) SetSelf(self Curve2D) { c.ParametricCurveBase.SetSelf(self) }

func (c *Curve2DBase) outer2D() Curve2D { return c.outer().(Curve2D) }

// GenerateSamples returns the samples (Dart returns the list it builds as
// an Iterable).
func (c *Curve2DBase) GenerateSamples(opts ...Curve2DGenerateSamplesOptions) iter.Seq[Curve2DSample] {
	return slices.Values(c.generateSamples(opts...))
}

func (c *Curve2DBase) generateSamples(opts ...Curve2DGenerateSamplesOptions) []Curve2DSample {
	start, end, tolerance := 0.0, 1.0, 1e-10
	if len(opts) > 0 {
		start = opts[0].Start
		if opts[0].End != nil {
			end = *opts[0].End
		}
		if opts[0].Tolerance != nil {
			tolerance = *opts[0].Tolerance
		}
	}
	self := c.outer2D()
	// The sampling algorithm is:
	// 1. Evaluate the area of the triangle (a proxy for the "flatness" of the
	//    curve) formed by two points and a test point.
	// 2. If the area of the triangle is small enough (below tolerance), then
	//    the two points form the final segment.
	// 3. If the area is still too large, divide the interval into two parts
	//    using a random subdivision point to avoid aliasing.
	// 4. Recursively sample the two parts.
	//
	// This algorithm concentrates samples in areas of high curvature.
	assert(end > start, nil) // assert(end > start)
	// We want to pick a random seed that will keep the result stable if
	// evaluated again, so we use the first non-generated control point.
	rand := dart.NewRandom(self.SamplingSeed())
	isFlat := func(p, q, r ui.Offset) bool {
		// Calculates the area of the triangle given by the three points.
		pr := p.Sub(r)
		qr := q.Sub(r)
		z := float64(pr.Dx*qr.Dy) - float64(qr.Dx*pr.Dy)
		return (z * z) < tolerance
	}
	first := NewCurve2DSample(start, self.Transform(start))
	last := NewCurve2DSample(end, self.Transform(end))
	samples := []Curve2DSample{first}
	var sample func(p, q Curve2DSample, forceSubdivide bool)
	sample = func(p, q Curve2DSample, forceSubdivide bool) {
		// Pick a random point somewhat near the center, which avoids aliasing
		// problems with periodic curves.
		t := p.T + float64((0.45+float64(0.1*rand.NextDouble()))*(q.T-p.T))
		r := NewCurve2DSample(t, self.Transform(t))
		if !forceSubdivide && isFlat(p.Value, q.Value, r.Value) {
			samples = append(samples, q)
		} else {
			sample(p, r, false)
			sample(r, q, false)
		}
	}
	// If the curve starts and ends on the same point, then we force it to
	// subdivide at least once, because otherwise it will terminate immediately.
	sample(first, last,
		math.Abs(first.Value.Dx-last.Value.Dx) < tolerance &&
			math.Abs(first.Value.Dy-last.Value.Dy) < tolerance)
	return samples
}

// SamplingSeed is 0 unless overridden.
func (c *Curve2DBase) SamplingSeed() int { return 0 }

// FindInverse returns the parametric value t whose X is x, found by
// bisection to within 1e-6 or 100 subdivisions, whichever comes first.
// The curve must be single-valued in X.
func (c *Curve2DBase) FindInverse(x float64) float64 {
	self := c.outer2D()
	start := 0.0
	end := 1.0
	var mid float64
	offsetToOrigin := func(pos float64) float64 { return x - self.Transform(pos).Dx }
	// Use a binary search to find the inverse point within 1e-6, or 100
	// subdivisions, whichever comes first.
	const errorLimit = 1e-6
	count := 100
	startValue := offsetToOrigin(start)
	for (end-start)/2.0 > errorLimit && count > 0 {
		mid = (end + start) / 2.0
		value := offsetToOrigin(mid)
		if dartSign(value) == dartSign(startValue) {
			start = mid
		} else {
			end = mid
		}
		count--
	}
	return mid
}

// dartSign is double.sign: -1.0, 1.0, or the value itself for ±0.0 and NaN.
func dartSign(v float64) float64 {
	switch {
	case v > 0:
		return 1.0
	case v < 0:
		return -1.0
	}
	return v
}

// Curve2DSample is a sample of a Curve2D: the parametric value T and the
// point Value.
type Curve2DSample struct {
	T     float64
	Value ui.Offset
}

// NewCurve2DSample is Curve2DSample(t, value).
func NewCurve2DSample(t float64, value ui.Offset) Curve2DSample { return Curve2DSample{t, value} }

func (s Curve2DSample) String() string {
	return "[(" + dart.ToStringAsFixed(s.Value.Dx, 2) + ", " + dart.ToStringAsFixed(s.Value.Dy, 2) + "), " +
		dart.ToStringAsFixed(s.T, 2) + "]"
}

// CatmullRomSpline is a 2D spline that passes smoothly through the given
// control points using a centripetal Catmull-Rom spline.
type CatmullRomSpline struct {
	Curve2DBase
	// The list of control point lists for each cubic segment of the spline.
	cubicSegments [][4]ui.Offset
	// This is non-nil only if the cubicSegments are being computed lazily.
	controlPoints []ui.Offset
	startHandle   *ui.Offset
	endHandle     *ui.Offset
	tension       float64
}

// CatmullRomSplineOptions are CatmullRomSpline's optional arguments.
type CatmullRomSplineOptions struct {
	Tension     float64
	StartHandle *ui.Offset
	EndHandle   *ui.Offset
}

func catmullRomSplineAsserts(controlPoints []ui.Offset, tension float64) {
	assert(tension <= 1.0, "tension "+dart.DoubleToString(tension)+" must not be greater than 1.0.")
	assert(tension >= 0.0, "tension "+dart.DoubleToString(tension)+" must not be negative.")
	assert(len(controlPoints) > 3, "There must be at least four control points to create a CatmullRomSpline.")
}

// NewCatmullRomSpline is CatmullRomSpline(controlPoints, {tension = 0.0,
// startHandle, endHandle}). The segments are computed on first use.
func NewCatmullRomSpline(controlPoints []ui.Offset, opts ...CatmullRomSplineOptions) *CatmullRomSpline {
	var o CatmullRomSplineOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	catmullRomSplineAsserts(controlPoints, o.Tension)
	if controlPoints == nil {
		controlPoints = []ui.Offset{}
	}
	c := &CatmullRomSpline{controlPoints: controlPoints, tension: o.Tension, startHandle: o.StartHandle, endHandle: o.EndHandle}
	c.SetSelf(c)
	return c
}

// NewCatmullRomSplinePrecompute is CatmullRomSpline.precompute: the
// segments are computed immediately.
func NewCatmullRomSplinePrecompute(controlPoints []ui.Offset, opts ...CatmullRomSplineOptions) *CatmullRomSpline {
	var o CatmullRomSplineOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	catmullRomSplineAsserts(controlPoints, o.Tension)
	c := &CatmullRomSpline{cubicSegments: catmullRomComputeSegments(controlPoints, o.Tension, o.StartHandle, o.EndHandle)}
	c.SetSelf(c)
	return c
}

func catmullRomComputeSegments(controlPoints []ui.Offset, tension float64, startHandle, endHandle *ui.Offset) [][4]ui.Offset {
	assert(startHandle == nil || startHandle.IsFinite(),
		"The provided startHandle of CatmullRomSpline must be finite. The startHandle given was "+offsetOrNull(startHandle)+".")
	assert(endHandle == nil || endHandle.IsFinite(),
		"The provided endHandle of CatmullRomSpline must be finite. The endHandle given was "+offsetOrNull(endHandle)+".")
	for index := range controlPoints {
		if !controlPoints[index].IsFinite() {
			panic(foundation.NewFlutterError("The provided CatmullRomSpline control point at index " + itoa(index) +
				" is not finite. The control point given was " + controlPoints[index].String() + "."))
		}
	}
	// If not specified, select the first and last control points (which are
	// handles: they are not intersected by the resulting curve) so that they
	// extend the first and last segments, respectively.
	var start, end ui.Offset
	if startHandle != nil {
		start = *startHandle
	} else {
		start = controlPoints[0].Mul(2.0).Sub(controlPoints[1])
	}
	if endHandle != nil {
		end = *endHandle
	} else {
		end = controlPoints[len(controlPoints)-1].Mul(2.0).Sub(controlPoints[len(controlPoints)-2])
	}
	allPoints := append(append([]ui.Offset{start}, controlPoints...), end)

	// An alpha of 0.5 is what makes it a centripetal Catmull-Rom spline. A
	// value of 0.0 would make it a uniform Catmull-Rom spline, and a value of
	// 1.0 would make it a chordal Catmull-Rom spline. Non-centripetal values
	// for alpha can give self-intersecting behavior or looping within a
	// segment.
	const alpha = 0.5
	reverseTension := 1.0 - tension
	result := [][4]ui.Offset{}
	for i := 0; i < len(allPoints)-3; i++ {
		curve := [4]ui.Offset{allPoints[i], allPoints[i+1], allPoints[i+2], allPoints[i+3]}
		diffCurve10 := curve[1].Sub(curve[0])
		diffCurve21 := curve[2].Sub(curve[1])
		diffCurve32 := curve[3].Sub(curve[2])
		t01 := dart.Pow(diffCurve10.Distance(), alpha)
		t12 := dart.Pow(diffCurve21.Distance(), alpha)
		t23 := dart.Pow(diffCurve32.Distance(), alpha)

		m1 := diffCurve21.Add(diffCurve10.Div(t01).Sub(curve[2].Sub(curve[0]).Div(t01 + t12)).Mul(t12)).Mul(reverseTension)
		m2 := diffCurve21.Add(diffCurve32.Div(t23).Sub(curve[3].Sub(curve[1]).Div(t12 + t23)).Mul(t12)).Mul(reverseTension)
		sumM12 := m1.Add(m2)

		result = append(result, [4]ui.Offset{
			diffCurve21.Mul(-2.0).Add(sumM12),
			diffCurve21.Mul(3.0).Sub(m1).Sub(sumM12),
			m1,
			curve[1],
		})
	}
	return result
}

func (c *CatmullRomSpline) initializeIfNeeded() {
	if len(c.cubicSegments) != 0 {
		return
	}
	c.cubicSegments = append(c.cubicSegments,
		catmullRomComputeSegments(c.controlPoints, c.tension, c.startHandle, c.endHandle)...)
}

// SamplingSeed is derived from the first non-generated control point.
func (c *CatmullRomSpline) SamplingSeed() int {
	c.initializeIfNeeded()
	seedPoint := c.cubicSegments[0][1]
	return dartRound(float64((seedPoint.Dx + seedPoint.Dy) * 10000))
}

func (c *CatmullRomSpline) TransformInternal(t float64) ui.Offset {
	c.initializeIfNeeded()
	length := float64(len(c.cubicSegments))
	var position, localT float64
	var index int
	if t < 1.0 {
		position = float64(t * length)
		localT = dart.Mod(position, 1.0)
		index = int(math.Floor(position))
	} else {
		position = length
		localT = 1.0
		index = len(c.cubicSegments) - 1
	}
	cubicControlPoints := c.cubicSegments[index]
	localT2 := float64(localT * localT)
	return cubicControlPoints[0].Mul(localT2).Mul(localT).
		Add(cubicControlPoints[1].Mul(localT2)).
		Add(cubicControlPoints[2].Mul(localT)).
		Add(cubicControlPoints[3])
}

// CatmullRomCurve is an animation easing curve that passes smoothly through
// the given control points using a centripetal Catmull-Rom spline.
type CatmullRomCurve struct {
	CurveBase
	// The precomputed approximation curve, so that evaluation of the curve
	// is efficient. Empty if the curve is constructed lazily; filled the
	// first time Transform is called.
	precomputedSamples []Curve2DSample
	// ControlPoints are the control points used to create this curve.
	ControlPoints []ui.Offset
	// Tension is the "tension" of the curve (0.0 to 1.0).
	Tension float64
}

// CatmullRomCurveOptions are CatmullRomCurve's optional arguments.
type CatmullRomCurveOptions struct{ Tension float64 }

func catmullRomCurveAssert(controlPoints []ui.Offset, tension float64) {
	reasons := []string{}
	if !CatmullRomCurveValidateControlPoints(controlPoints, CatmullRomCurveValidateControlPointsOptions{
		Tension: tension, Reasons: &reasons,
	}) {
		panic(dart.NewAssertionError("control points " + offsetList(controlPoints) + " could not be validated:\n  " +
			strings.Join(reasons, "\n  ")))
	}
}

// NewCatmullRomCurve is CatmullRomCurve(controlPoints, {tension = 0.0}).
// The samples are computed on first use.
func NewCatmullRomCurve(controlPoints []ui.Offset, opts ...CatmullRomCurveOptions) *CatmullRomCurve {
	var o CatmullRomCurveOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	catmullRomCurveAssert(controlPoints, o.Tension)
	c := &CatmullRomCurve{ControlPoints: controlPoints, Tension: o.Tension, precomputedSamples: []Curve2DSample{}}
	c.SetSelf(c)
	return c
}

// NewCatmullRomCurvePrecompute is CatmullRomCurve.precompute: the samples
// are computed immediately.
func NewCatmullRomCurvePrecompute(controlPoints []ui.Offset, opts ...CatmullRomCurveOptions) *CatmullRomCurve {
	var o CatmullRomCurveOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	catmullRomCurveAssert(controlPoints, o.Tension)
	c := &CatmullRomCurve{ControlPoints: controlPoints, Tension: o.Tension,
		precomputedSamples: catmullRomCurveComputeSamples(controlPoints, o.Tension)}
	c.SetSelf(c)
	return c
}

func catmullRomCurveComputeSamples(controlPoints []ui.Offset, tension float64) []Curve2DSample {
	// Force the first and last control points for the spline to be (0, 0)
	// and (1, 1), respectively.
	points := append(append([]ui.Offset{ui.OffsetZero}, controlPoints...), ui.NewOffset(1.0, 1.0))
	tolerance := 1e-12
	return NewCatmullRomSplinePrecompute(points, CatmullRomSplineOptions{Tension: tension}).
		generateSamples(Curve2DGenerateSamplesOptions{Tolerance: &tolerance})
}

// CatmullRomCurveValidateControlPointsOptions are the optional arguments of
// CatmullRomCurveValidateControlPoints. Reasons, when not nil, receives a
// description of each failure (debug builds only).
type CatmullRomCurveValidateControlPointsOptions struct {
	Tension float64
	Reasons *[]string
}

// CatmullRomCurveValidateControlPoints is CatmullRomCurve.validateControlPoints:
// whether the control points make a valid curve (one that is single-valued
// in X and starts and ends at the corners of the unit square). A nil slice
// is Dart's null.
func CatmullRomCurveValidateControlPoints(controlPoints []ui.Offset, opts ...CatmullRomCurveValidateControlPointsOptions) bool {
	var o CatmullRomCurveValidateControlPointsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	tension, reasons := o.Tension, o.Reasons
	// addReason is Flutter's assert(() { reasons?.add(...); ... }()): it runs
	// in debug builds and reports whether to bail (always outside debug).
	addReason := func(reason func() string) (bail bool) {
		if foundation.KDebugMode {
			if reasons != nil {
				*reasons = append(*reasons, reason())
			}
			return reasons == nil
		}
		return true
	}
	if controlPoints == nil {
		addReason(func() string { return "Supplied control points cannot be null" })
		return false
	}

	if len(controlPoints) < 2 {
		addReason(func() string { return "There must be at least two points supplied to create a valid curve." })
		return false
	}

	controlPoints = append(append([]ui.Offset{ui.OffsetZero}, controlPoints...), ui.NewOffset(1.0, 1.0))
	startHandle := controlPoints[0].Mul(2.0).Sub(controlPoints[1])
	endHandle := controlPoints[len(controlPoints)-1].Mul(2.0).Sub(controlPoints[len(controlPoints)-2])
	controlPoints = append(append([]ui.Offset{startHandle}, controlPoints...), endHandle)
	lastX := math.Inf(-1)
	for i := 0; i < len(controlPoints); i++ {
		if i > 1 &&
			i < len(controlPoints)-2 &&
			(controlPoints[i].Dx <= 0.0 || controlPoints[i].Dx >= 1.0) {
			addReason(func() string {
				return "Control points must have X values between 0.0 and 1.0, exclusive. " +
					"Point " + itoa(i) + " has an x value (" + dart.DoubleToString(controlPoints[i].Dx) + ") which is outside the range."
			})
			return false
		}
		if controlPoints[i].Dx <= lastX {
			addReason(func() string {
				return "Each X coordinate must be greater than the preceding X coordinate " +
					"(i.e. must be monotonically increasing in X). Point " + itoa(i) + " has an x value of " +
					dart.DoubleToString(controlPoints[i].Dx) + ", which is not greater than " + dart.DoubleToString(lastX)
			})
			return false
		}
		lastX = controlPoints[i].Dx
	}

	success := true

	// An empirical test to make sure things are single-valued in X.
	lastX = math.Inf(-1)
	const tolerance = 1e-3
	testSpline := NewCatmullRomSpline(controlPoints, CatmullRomSplineOptions{Tension: tension})
	start := testSpline.FindInverse(0.0)
	end := testSpline.FindInverse(1.0)
	samplePoints := testSpline.generateSamples(Curve2DGenerateSamplesOptions{Start: start, End: &end})
	if math.Abs(samplePoints[0].Value.Dy) > tolerance ||
		math.Abs(1.0-samplePoints[len(samplePoints)-1].Value.Dy) > tolerance {
		success = false
		if addReason(func() string {
			return "The curve has more than one Y value at X = " + dart.DoubleToString(samplePoints[0].Value.Dx) + ". " +
				"Try moving some control points further away from this value of X, or increasing " +
				"the tension."
		}) {
			// If we're not in debug mode, then we want to bail immediately
			// instead of checking everything else.
			return false
		}
	}
	for _, sample := range samplePoints {
		point := sample.Value
		t := sample.T
		x := point.Dx
		if t >= start && t <= end && (x < -1e-3 || x > 1.0+1e-3) {
			success = false
			if addReason(func() string {
				return "The resulting curve has an X value (" + dart.DoubleToString(x) + ") which is outside " +
					"the range [0.0, 1.0], inclusive."
			}) {
				// If we're not in debug mode, then we want to bail immediately
				// instead of checking all the segments.
				return false
			}
		}
		if x < lastX {
			success = false
			if addReason(func() string {
				return "The curve has more than one Y value at x = " + dart.DoubleToString(x) + ". Try moving " +
					"some control points further apart in X, or increasing the tension."
			}) {
				// If we're not in debug mode, then we want to bail immediately
				// instead of checking all the segments.
				return false
			}
		}
		lastX = x
	}
	return success
}

func (c *CatmullRomCurve) TransformInternal(t float64) float64 {
	// Linearly interpolate between the two closest samples generated when the
	// curve was created.
	if len(c.precomputedSamples) == 0 {
		// Compute the samples now if we were constructed lazily.
		c.precomputedSamples = append(c.precomputedSamples, catmullRomCurveComputeSamples(c.ControlPoints, c.Tension)...)
	}
	start := 0
	end := len(c.precomputedSamples) - 1
	var mid int
	var value ui.Offset
	startValue := c.precomputedSamples[start].Value
	endValue := c.precomputedSamples[end].Value
	// Use a binary search to find the index of the sample point that is just
	// before t.
	for end-start > 1 {
		mid = (end + start) / 2
		value = c.precomputedSamples[mid].Value
		if t >= value.Dx {
			start = mid
			startValue = value
		} else {
			end = mid
			endValue = value
		}
	}
	// Now interpolate between the found sample and the next one.
	t2 := (t - startValue.Dx) / (endValue.Dx - startValue.Dx)
	return lerpDouble(startValue.Dy, endValue.Dy, t2)
}

// FlippedCurve is a curve that is the reverse inversion of its given curve:
// it maps t to 1 - Curve.Transform(1 - t).
type FlippedCurve struct {
	CurveBase
	Curve Curve
}

// NewFlippedCurve is FlippedCurve(curve).
func NewFlippedCurve(curve Curve) *FlippedCurve {
	c := &FlippedCurve{Curve: curve}
	c.SetSelf(c)
	return c
}

func (c *FlippedCurve) TransformInternal(t float64) float64 { return 1.0 - c.Curve.Transform(1.0-t) }

func (c *FlippedCurve) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "FlippedCurve") + "(" + c.Curve.String() + ")"
}

// _DecelerateCurve is Curves.decelerate.
type _DecelerateCurve struct{ CurveBase }

func newDecelerateCurve() *_DecelerateCurve { c := &_DecelerateCurve{}; c.SetSelf(c); return c }

func (c *_DecelerateCurve) TransformInternal(t float64) float64 {
	// Intended to match the behavior of:
	// https://android.googlesource.com/platform/frameworks/base/+/main/core/java/android/view/animation/DecelerateInterpolator.java
	// ...as of December 2016.
	t = 1.0 - t
	return 1.0 - float64(t*t)
}

// BOUNCE CURVES

func bounce(t float64) float64 {
	if t < 1.0/2.75 {
		return 7.5625 * t * t
	} else if t < 2/2.75 {
		t -= 1.5 / 2.75
		return float64(7.5625*t*t) + 0.75
	} else if t < 2.5/2.75 {
		t -= 2.25 / 2.75
		return float64(7.5625*t*t) + 0.9375
	}
	t -= 2.625 / 2.75
	return float64(7.5625*t*t) + 0.984375
}

// _BounceInCurve is Curves.bounceIn.
type _BounceInCurve struct{ CurveBase }

func newBounceInCurve() *_BounceInCurve { c := &_BounceInCurve{}; c.SetSelf(c); return c }

func (c *_BounceInCurve) TransformInternal(t float64) float64 { return 1.0 - bounce(1.0-t) }

// _BounceOutCurve is Curves.bounceOut.
type _BounceOutCurve struct{ CurveBase }

func newBounceOutCurve() *_BounceOutCurve { c := &_BounceOutCurve{}; c.SetSelf(c); return c }

func (c *_BounceOutCurve) TransformInternal(t float64) float64 { return bounce(t) }

// _BounceInOutCurve is Curves.bounceInOut.
type _BounceInOutCurve struct{ CurveBase }

func newBounceInOutCurve() *_BounceInOutCurve { c := &_BounceInOutCurve{}; c.SetSelf(c); return c }

func (c *_BounceInOutCurve) TransformInternal(t float64) float64 {
	if t < 0.5 {
		return (1.0 - bounce(1.0-float64(t*2.0))) * 0.5
	}
	return float64(bounce(float64(t*2.0)-1.0)*0.5) + 0.5
}

// ELASTIC CURVES

// ElasticInCurve is an oscillating curve that grows in magnitude while
// overshooting its bounds.
type ElasticInCurve struct {
	CurveBase
	// Period is the duration of the oscillation (default 0.4).
	Period float64
}

func optionalPeriod(period []float64) float64 {
	if len(period) > 0 {
		return period[0]
	}
	return 0.4
}

// NewElasticInCurve is ElasticInCurve([period = 0.4]).
func NewElasticInCurve(period ...float64) *ElasticInCurve {
	c := &ElasticInCurve{Period: optionalPeriod(period)}
	c.SetSelf(c)
	return c
}

func (c *ElasticInCurve) TransformInternal(t float64) float64 {
	s := c.Period / 4.0
	t = t - 1.0
	return -dart.Pow(2.0, 10.0*t) * dart.Sin((t-s)*(math.Pi*2.0)/c.Period)
}

func (c *ElasticInCurve) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "ElasticInCurve") + "(" + dart.DoubleToString(c.Period) + ")"
}

// ElasticOutCurve is an oscillating curve that shrinks in magnitude while
// overshooting its bounds.
type ElasticOutCurve struct {
	CurveBase
	// Period is the duration of the oscillation (default 0.4).
	Period float64
}

// NewElasticOutCurve is ElasticOutCurve([period = 0.4]).
func NewElasticOutCurve(period ...float64) *ElasticOutCurve {
	c := &ElasticOutCurve{Period: optionalPeriod(period)}
	c.SetSelf(c)
	return c
}

func (c *ElasticOutCurve) TransformInternal(t float64) float64 {
	s := c.Period / 4.0
	return float64(dart.Pow(2.0, -10*t)*dart.Sin((t-s)*(math.Pi*2.0)/c.Period)) + 1.0
}

func (c *ElasticOutCurve) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "ElasticOutCurve") + "(" + dart.DoubleToString(c.Period) + ")"
}

// ElasticInOutCurve is an oscillating curve that grows and then shrinks in
// magnitude while overshooting its bounds.
type ElasticInOutCurve struct {
	CurveBase
	// Period is the duration of the oscillation (default 0.4).
	Period float64
}

// NewElasticInOutCurve is ElasticInOutCurve([period = 0.4]).
func NewElasticInOutCurve(period ...float64) *ElasticInOutCurve {
	c := &ElasticInOutCurve{Period: optionalPeriod(period)}
	c.SetSelf(c)
	return c
}

func (c *ElasticInOutCurve) TransformInternal(t float64) float64 {
	s := c.Period / 4.0
	t = float64(2.0*t) - 1.0
	if t < 0.0 {
		return -0.5 * dart.Pow(2.0, 10.0*t) * dart.Sin((t-s)*(math.Pi*2.0)/c.Period)
	}
	return float64(dart.Pow(2.0, -10.0*t)*dart.Sin((t-s)*(math.Pi*2.0)/c.Period)*0.5) + 1.0
}

func (c *ElasticInOutCurve) String() string {
	return foundation.ObjectRuntimeType(c.outer(), "ElasticInOutCurve") + "(" + dart.DoubleToString(c.Period) + ")"
}

// PREDEFINED CURVES

// Curves holds Flutter's predefined curves (Dart's class Curves, a class
// of static constants). See https://api.flutter.dev/flutter/animation/Curves-class.html
// for pictures of each.
var Curves = struct {
	Linear                   Curve
	Decelerate               Curve
	FastLinearToSlowEaseIn   *Cubic
	FastEaseInToSlowEaseOut  *ThreePointCubic
	Ease                     *Cubic
	EaseIn                   *Cubic
	EaseInToLinear           *Cubic
	EaseInSine               *Cubic
	EaseInQuad               *Cubic
	EaseInCubic              *Cubic
	EaseInQuart              *Cubic
	EaseInQuint              *Cubic
	EaseInExpo               *Cubic
	EaseInCirc               *Cubic
	EaseInBack               *Cubic
	EaseOut                  *Cubic
	LinearToEaseOut          *Cubic
	EaseOutSine              *Cubic
	EaseOutQuad              *Cubic
	EaseOutCubic             *Cubic
	EaseOutQuart             *Cubic
	EaseOutQuint             *Cubic
	EaseOutExpo              *Cubic
	EaseOutCirc              *Cubic
	EaseOutBack              *Cubic
	EaseInOut                *Cubic
	EaseInOutSine            *Cubic
	EaseInOutQuad            *Cubic
	EaseInOutCubic           *Cubic
	EaseInOutCubicEmphasized *ThreePointCubic
	EaseInOutQuart           *Cubic
	EaseInOutQuint           *Cubic
	EaseInOutExpo            *Cubic
	EaseInOutCirc            *Cubic
	EaseInOutBack            *Cubic
	FastOutSlowIn            *Cubic
	SlowMiddle               *Cubic
	BounceIn                 Curve
	BounceOut                Curve
	BounceInOut              Curve
	ElasticIn                *ElasticInCurve
	ElasticOut               *ElasticOutCurve
	ElasticInOut             *ElasticInOutCurve
}{
	Linear:                 newLinear(),
	Decelerate:             newDecelerateCurve(),
	FastLinearToSlowEaseIn: NewCubic(0.18, 1.0, 0.04, 1.0),
	FastEaseInToSlowEaseOut: NewThreePointCubic(
		ui.NewOffset(0.056, 0.024),
		ui.NewOffset(0.108, 0.3085),
		ui.NewOffset(0.198, 0.541),
		ui.NewOffset(0.3655, 1.0),
		ui.NewOffset(0.5465, 0.989),
	),
	Ease:            NewCubic(0.25, 0.1, 0.25, 1.0),
	EaseIn:          NewCubic(0.42, 0.0, 1.0, 1.0),
	EaseInToLinear:  NewCubic(0.67, 0.03, 0.65, 0.09),
	EaseInSine:      NewCubic(0.47, 0.0, 0.745, 0.715),
	EaseInQuad:      NewCubic(0.55, 0.085, 0.68, 0.53),
	EaseInCubic:     NewCubic(0.55, 0.055, 0.675, 0.19),
	EaseInQuart:     NewCubic(0.895, 0.03, 0.685, 0.22),
	EaseInQuint:     NewCubic(0.755, 0.05, 0.855, 0.06),
	EaseInExpo:      NewCubic(0.95, 0.05, 0.795, 0.035),
	EaseInCirc:      NewCubic(0.6, 0.04, 0.98, 0.335),
	EaseInBack:      NewCubic(0.6, -0.28, 0.735, 0.045),
	EaseOut:         NewCubic(0.0, 0.0, 0.58, 1.0),
	LinearToEaseOut: NewCubic(0.35, 0.91, 0.33, 0.97),
	EaseOutSine:     NewCubic(0.39, 0.575, 0.565, 1.0),
	EaseOutQuad:     NewCubic(0.25, 0.46, 0.45, 0.94),
	EaseOutCubic:    NewCubic(0.215, 0.61, 0.355, 1.0),
	EaseOutQuart:    NewCubic(0.165, 0.84, 0.44, 1.0),
	EaseOutQuint:    NewCubic(0.23, 1.0, 0.32, 1.0),
	EaseOutExpo:     NewCubic(0.19, 1.0, 0.22, 1.0),
	EaseOutCirc:     NewCubic(0.075, 0.82, 0.165, 1.0),
	EaseOutBack:     NewCubic(0.175, 0.885, 0.32, 1.275),
	EaseInOut:       NewCubic(0.42, 0.0, 0.58, 1.0),
	EaseInOutSine:   NewCubic(0.445, 0.05, 0.55, 0.95),
	EaseInOutQuad:   NewCubic(0.455, 0.03, 0.515, 0.955),
	EaseInOutCubic:  NewCubic(0.645, 0.045, 0.355, 1.0),
	EaseInOutCubicEmphasized: NewThreePointCubic(
		ui.NewOffset(0.05, 0),
		ui.NewOffset(0.133333, 0.06),
		ui.NewOffset(0.166666, 0.4),
		ui.NewOffset(0.208333, 0.82),
		ui.NewOffset(0.25, 1),
	),
	EaseInOutQuart: NewCubic(0.77, 0.0, 0.175, 1.0),
	EaseInOutQuint: NewCubic(0.86, 0.0, 0.07, 1.0),
	EaseInOutExpo:  NewCubic(1.0, 0.0, 0.0, 1.0),
	EaseInOutCirc:  NewCubic(0.785, 0.135, 0.15, 0.86),
	EaseInOutBack:  NewCubic(0.68, -0.55, 0.265, 1.55),
	FastOutSlowIn:  NewCubic(0.4, 0.0, 0.2, 1.0),
	SlowMiddle:     NewCubic(0.15, 0.85, 0.85, 0.15),
	BounceIn:       newBounceInCurve(),
	BounceOut:      newBounceOutCurve(),
	BounceInOut:    newBounceInOutCurve(),
	ElasticIn:      NewElasticInCurve(),
	ElasticOut:     NewElasticOutCurve(),
	ElasticInOut:   NewElasticInOutCurve(),
}

// Helpers.

// assert is a Dart assert that Flutter relies on (always on in flutter-go).
func assert(cond bool, message any) {
	if !cond {
		panic(dart.NewAssertionError(message))
	}
}

// lerpDouble is dart:ui's lerpDouble on non-null values (with its
// debug-mode asserts).
func lerpDouble(a, b, t float64) float64 { return *ui.LerpDouble(&a, &b, t) }

// dartRound is double.round(): half away from zero; Dart throws for NaN and
// infinities.
func dartRound(v float64) int {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		panic(dart.UnsupportedError{Message: "Infinity or NaN toInt"})
	}
	return int(math.Round(v))
}

func itoa(i int) string { return dart.SafeToString(i) }

func offsetOrNull(o *ui.Offset) string {
	if o == nil {
		return "null"
	}
	return o.String()
}

// offsetList is List<Offset>.toString().
func offsetList(points []ui.Offset) string {
	parts := make([]string, len(points))
	for i, p := range points {
		parts[i] = p.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
