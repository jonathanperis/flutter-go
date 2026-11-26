package animation

// Port of packages/flutter/test/animation/curves_test.dart (Flutter
// 3.50.1): all 22 tests, none skipped. Dart's `throwsAssertionError` is a
// panic whose value is a dart.AssertionFailure (a FlutterError counts, as in
// Dart); `throwsArgumentError` is a dart.ArgumentError. The tests after
// "Flutter's own tests end here" are flutter-go's.

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

func off(dx, dy float64) ui.Offset { return ui.NewOffset(dx, dy) }

// hasOneLineDescription is flutter_test's matcher.
func hasOneLineDescription(t *testing.T, v fmt.Stringer) {
	t.Helper()
	d := v.String()
	if d == "" || strings.Contains(d, "\n") || strings.Contains(d, "Instance of ") || strings.TrimSpace(d) != d {
		t.Errorf("%q is not a one-line description", d)
	}
}

func panicValue(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return nil
}

func throwsAssertionError(t *testing.T, f func()) {
	t.Helper()
	r := panicValue(f)
	if _, ok := r.(dart.AssertionFailure); !ok {
		t.Errorf("want an AssertionError, got %T %v", r, r)
	}
}

func throwsArgumentError(t *testing.T, f func()) {
	t.Helper()
	r := panicValue(f)
	if _, ok := r.(dart.ArgumentError); !ok {
		t.Errorf("want an ArgumentError, got %T %v", r, r)
	}
}

func expectEq(t *testing.T, got, want float64) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func moreOrLessEquals(t *testing.T, got, want, epsilon float64) {
	t.Helper()
	if math.Abs(got-want) > epsilon {
		t.Errorf("got %v, want %v ± %v", got, want, epsilon)
	}
}

func TestToStringControlTest(t *testing.T) {
	hasOneLineDescription(t, Curves.Linear)
	hasOneLineDescription(t, NewSawTooth(3))
	hasOneLineDescription(t, NewInterval(0.25, 0.75))
	hasOneLineDescription(t, NewInterval(0.25, 0.75, IntervalOptions{Curve: Curves.Ease}))
	hasOneLineDescription(t, NewSplit(0.25, SplitOptions{BeginCurve: Curves.Ease}))
}

func TestCurveFlippedControlTest(t *testing.T) {
	var ease Curve = Curves.Ease
	flippedEase := ease.Flipped()
	if !(flippedEase.Transform(0.0) < 0.001) {
		t.Error("flipped(0)")
	}
	if !(flippedEase.Transform(0.5) < ease.Transform(0.5)) {
		t.Error("flipped(0.5)")
	}
	if !(flippedEase.Transform(1.0) > 0.999) {
		t.Error("flipped(1)")
	}
	hasOneLineDescription(t, flippedEase)
}

func TestThresholdHasAThreshold(t *testing.T) {
	var step Curve = NewThreshold(0.25)
	expectEq(t, step.Transform(0.0), 0.0)
	expectEq(t, step.Transform(0.24), 0.0)
	expectEq(t, step.Transform(0.25), 1.0)
	expectEq(t, step.Transform(0.26), 1.0)
	expectEq(t, step.Transform(1.0), 1.0)
}

func assertMaximumSlope(t *testing.T, curve Curve, maximumSlope float64) {
	t.Helper()
	const delta = 0.005
	for x := 0.0; x < 1.0-delta; x += delta {
		deltaY := curve.Transform(x) - curve.Transform(x+delta)
		if !(math.Abs(deltaY) < delta*maximumSlope) {
			t.Errorf("%v discontinuous at %v", curve, x)
		}
	}
}

func TestCurveIsContinuous(t *testing.T) {
	for _, c := range []Curve{
		Curves.Linear, Curves.Decelerate, Curves.FastOutSlowIn, Curves.SlowMiddle, Curves.BounceIn,
		Curves.BounceOut, Curves.BounceInOut, Curves.ElasticOut, Curves.ElasticInOut, Curves.Ease,

		Curves.EaseIn, Curves.EaseInSine, Curves.EaseInQuad, Curves.EaseInCubic, Curves.EaseInQuart,
		Curves.EaseInQuint, Curves.EaseInExpo, Curves.EaseInCirc,

		Curves.EaseOut, Curves.EaseOutSine, Curves.EaseOutQuad, Curves.EaseOutCubic,
		Curves.EaseInOutCubicEmphasized, Curves.EaseOutQuart, Curves.EaseOutQuint, Curves.EaseOutExpo,
		Curves.EaseOutCirc,

		// Curves.easeInOutExpo is discontinuous at its midpoint, so not included
		// here

		Curves.EaseInOut, Curves.EaseInOutSine, Curves.EaseInOutQuad, Curves.EaseInOutCubic,
		Curves.EaseInOutQuart, Curves.EaseInOutQuint, Curves.EaseInOutCirc,
	} {
		assertMaximumSlope(t, c, 20.0)
	}
}

func expectStaysInBounds(t *testing.T, curve Curve) {
	t.Helper()
	for _, x := range []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0} {
		if v := curve.Transform(x); !(v >= 0.0 && v <= 1.0) {
			t.Errorf("%v.transform(%v) = %v, out of [0, 1]", curve, x, v)
		}
	}
}

func TestBounceStaysInBounds(t *testing.T) {
	expectStaysInBounds(t, Curves.BounceIn)
	expectStaysInBounds(t, Curves.BounceOut)
	expectStaysInBounds(t, Curves.BounceInOut)
}

func estimateBounds(curve Curve) [2]float64 {
	values := []float64{}
	for _, x := range []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0} {
		values = append(values, curve.Transform(x))
	}
	return [2]float64{slices.Min(values), slices.Max(values)}
}

func TestElasticOvershootsItsBounds(t *testing.T) {
	hasOneLineDescription(t, Curves.ElasticIn)
	hasOneLineDescription(t, Curves.ElasticOut)
	hasOneLineDescription(t, Curves.ElasticInOut)

	bounds := estimateBounds(Curves.ElasticIn)
	if !(bounds[0] < 0.0 && bounds[1] <= 1.0) {
		t.Errorf("elasticIn bounds %v", bounds)
	}
	bounds = estimateBounds(Curves.ElasticOut)
	if !(bounds[0] >= 0.0 && bounds[1] > 1.0) {
		t.Errorf("elasticOut bounds %v", bounds)
	}
	bounds = estimateBounds(Curves.ElasticInOut)
	if !(bounds[0] < 0.0 && bounds[1] > 1.0) {
		t.Errorf("elasticInOut bounds %v", bounds)
	}
}

func TestBackOvershootsItsBounds(t *testing.T) {
	hasOneLineDescription(t, Curves.EaseInBack)
	hasOneLineDescription(t, Curves.EaseOutBack)
	hasOneLineDescription(t, Curves.EaseInOutBack)

	bounds := estimateBounds(Curves.EaseInBack)
	if !(bounds[0] < 0.0 && bounds[1] <= 1.0) {
		t.Errorf("easeInBack bounds %v", bounds)
	}
	bounds = estimateBounds(Curves.EaseOutBack)
	if !(bounds[0] >= 0.0 && bounds[1] > 1.0) {
		t.Errorf("easeOutBack bounds %v", bounds)
	}
	bounds = estimateBounds(Curves.EaseInOutBack)
	if !(bounds[0] < 0.0 && bounds[1] > 1.0) {
		t.Errorf("easeInOutBack bounds %v", bounds)
	}
}

func TestDecelerateDoesSo(t *testing.T) {
	hasOneLineDescription(t, Curves.Decelerate)

	bounds := estimateBounds(Curves.Decelerate)
	if !(bounds[0] >= 0.0 && bounds[1] <= 1.0) {
		t.Errorf("decelerate bounds %v", bounds)
	}

	d1 := Curves.Decelerate.Transform(0.2) - Curves.Decelerate.Transform(0.0)
	d2 := Curves.Decelerate.Transform(1.0) - Curves.Decelerate.Transform(0.8)
	if !(d2 < d1) {
		t.Errorf("d2 %v >= d1 %v", d2, d1)
	}
}

func TestThreePointCubicInterpolatesMidpoint(t *testing.T) {
	test := NewThreePointCubic(
		off(0.05, 0),
		off(0.133333, 0.06),
		off(0.166666, 0.4),
		off(0.208333, 0.82),
		off(0.25, 1),
	)
	expectEq(t, test.Transform(0.166666), 0.4)
}

// Regression test for https://github.com/flutter/flutter/issues/171364.
func TestCubicTransformInternalClampsValuesOutsideTheUnitInterval(t *testing.T) {
	expectEq(t, Curves.EaseOut.TransformInternal(2.0), 1.0)
	expectEq(t, Curves.EaseOut.TransformInternal(-1.0), 0.0)
}

func TestCubicTransformInternalThrowsForNaN(t *testing.T) {
	throwsArgumentError(t, func() { Curves.EaseOut.TransformInternal(math.NaN()) })
}

func TestInvalidTransformParameterShouldAssert(t *testing.T) {
	for _, c := range []Curve{
		NewSawTooth(2),
		NewInterval(0.0, 1.0),
		NewSplit(0.0),
		NewThreshold(0.5),
		NewElasticInCurve(),
		NewElasticOutCurve(),
		NewCubic(0.42, 0.0, 0.58, 1.0),
		Curves.EaseInOutCubicEmphasized,
		Curves.Decelerate,
		Curves.BounceIn,
		Curves.BounceOut,
		Curves.BounceInOut,
	} {
		throwsAssertionError(t, func() { c.Transform(-0.0001) })
		throwsAssertionError(t, func() { c.Transform(1.0001) })
	}
}

func TestCurveTransformReturnsZeroAndOneAtTheEnds(t *testing.T) {
	for _, c := range []Curve{
		NewSawTooth(2),
		NewInterval(0, 1),
		NewSplit(0.5),
		NewThreshold(0.5),
		NewElasticInCurve(),
		NewElasticOutCurve(),
		NewElasticInOutCurve(),
		Curves.Linear,
		Curves.EaseInOutExpo,
		Curves.EaseInOutCubicEmphasized,
		NewFlippedCurve(Curves.EaseInOutExpo),
		Curves.Decelerate,
		Curves.BounceIn,
		Curves.BounceOut,
		Curves.BounceInOut,
	} {
		expectEq(t, c.Transform(0), 0)
		expectEq(t, c.Transform(1), 1)
	}
}

func TestSplitInterpolatesValuesProperly(t *testing.T) {
	curve := NewSplit(0.3)

	const tolerance = 1e-6
	expectEq(t, curve.Transform(0.0), 0.0)
	expectEq(t, curve.Transform(0.1), 0.1)
	expectEq(t, curve.Transform(0.25), 0.25)
	expectEq(t, curve.Transform(0.3), 0.3)
	moreOrLessEquals(t, curve.Transform(0.5), 0.760461, tolerance)
	moreOrLessEquals(t, curve.Transform(0.75), 0.962055, tolerance)
	expectEq(t, curve.Transform(1.0), 1.0)
}

var splinePoints = []ui.Offset{
	ui.OffsetZero,
	off(0.01, 0.25),
	off(0.2, 0.25),
	off(0.33, 0.25),
	off(0.5, 1.0),
	off(0.66, 0.75),
	off(1.0, 1.0),
}

func checkSpline(t *testing.T, curve Curve2D) {
	t.Helper()
	const tolerance = 1e-6
	for _, c := range []struct{ t, dx, dy float64 }{
		{0.0, 0.0, 0.0},
		{0.25, 0.0966945, 0.2626806},
		{0.5, 0.33, 0.25},
		{0.75, 0.570260, 0.883085},
		{1.0, 1.0, 1.0},
	} {
		moreOrLessEquals(t, curve.Transform(c.t).Dx, c.dx, tolerance)
		moreOrLessEquals(t, curve.Transform(c.t).Dy, c.dy, tolerance)
	}
}

func TestCatmullRomSplineInterpolatesValuesProperly(t *testing.T) {
	sh, eh := off(0.0, -0.3), off(1.3, 1.3)
	checkSpline(t, NewCatmullRomSpline(splinePoints, CatmullRomSplineOptions{StartHandle: &sh, EndHandle: &eh}))
}

var zeros4 = []ui.Offset{ui.OffsetZero, ui.OffsetZero, ui.OffsetZero, ui.OffsetZero}

func catmullRomSplineContract(t *testing.T, mk func([]ui.Offset, ...CatmullRomSplineOptions) *CatmullRomSpline, generate bool) {
	t.Helper()
	run := func(points []ui.Offset, opts ...CatmullRomSplineOptions) {
		t.Helper()
		throwsAssertionError(t, func() {
			s := mk(points, opts...)
			if generate {
				s.GenerateSamples()
			}
		})
	}
	inf := math.Inf(1)
	run([]ui.Offset{})
	run([]ui.Offset{ui.OffsetZero})
	run([]ui.Offset{ui.OffsetZero, ui.OffsetZero})
	run([]ui.Offset{ui.OffsetZero, ui.OffsetZero, ui.OffsetZero})
	run(zeros4, CatmullRomSplineOptions{Tension: -1.0})
	run(zeros4, CatmullRomSplineOptions{Tension: 2.0})
	run([]ui.Offset{off(inf, 0.0), ui.OffsetZero, ui.OffsetZero, ui.OffsetZero})
	run([]ui.Offset{off(0.0, inf), ui.OffsetZero, ui.OffsetZero, ui.OffsetZero})
	h := off(0.0, inf)
	run(zeros4, CatmullRomSplineOptions{StartHandle: &h})
	run(zeros4, CatmullRomSplineOptions{EndHandle: &h})
}

func TestCatmullRomSplineEnforcesContract(t *testing.T) {
	catmullRomSplineContract(t, NewCatmullRomSpline, true)
}

func TestCatmullRomSplineInterpolatesValuesProperlyWhenPrecomputed(t *testing.T) {
	sh, eh := off(0.0, -0.3), off(1.3, 1.3)
	checkSpline(t, NewCatmullRomSplinePrecompute(splinePoints, CatmullRomSplineOptions{StartHandle: &sh, EndHandle: &eh}))
}

func TestCatmullRomSplineEnforcesContractWhenPrecomputed(t *testing.T) {
	catmullRomSplineContract(t, NewCatmullRomSplinePrecompute, false)
}

var curvePoints = []ui.Offset{off(0.2, 0.25), off(0.33, 0.25), off(0.5, 1.0), off(0.8, 0.75)}

func checkCatmullRomCurve(t *testing.T, curve Curve) {
	t.Helper()
	// These values are approximations.
	const tolerance = 1e-6
	for _, c := range []struct{ t, want float64 }{
		{0.0, 0.0},
		{0.01, 0.012874734350170863},
		{0.2, 0.24989646045277542},
		{0.33, 0.250037698527661},
		{0.5, 0.9999057323235939},
		{0.6, 0.9357294964536621},
		{0.8, 0.7500423402378034},
		{1.0, 1.0},
	} {
		moreOrLessEquals(t, curve.Transform(c.t), c.want, tolerance)
	}
}

func TestCatmullRomCurveInterpolatesGivenPointsCorrectly(t *testing.T) {
	checkCatmullRomCurve(t, NewCatmullRomCurve(curvePoints))
}

func TestCatmullRomCurveInterpolatesGivenPointsCorrectlyWhenPrecomputed(t *testing.T) {
	checkCatmullRomCurve(t, NewCatmullRomCurvePrecompute(curvePoints))
}

// invalidCurvePoints are the invalid inputs of "CatmullRomCurve enforces
// contract" after the first three (too few points).
var invalidCurvePoints = [][]ui.Offset{
	// Monotonically increasing in X.
	{off(0.2, 0.25), off(0.01, 0.25)},
	// X within range (0.0, 1.0).
	{off(0.2, 0.25), off(1.01, 0.25)},
	// Not multi-valued in Y at x=0.0.
	{off(0.05, 0.50), off(0.50, 0.50), off(0.75, 0.75)},
	// Not multi-valued in Y at x=1.0.
	{off(0.25, 0.25), off(0.50, 0.50), off(0.95, 0.51)},
	// Not multi-valued in Y in between x = 0.0 and x = 1.0.
	{off(0.5, 0.05), off(0.5, 0.95)},
}

func TestCatmullRomCurveEnforcesContract(t *testing.T) {
	throwsAssertionError(t, func() { NewCatmullRomCurve([]ui.Offset{}) })
	throwsAssertionError(t, func() { NewCatmullRomCurve([]ui.Offset{ui.OffsetZero}) })
	throwsAssertionError(t, func() { NewCatmullRomCurve([]ui.Offset{ui.OffsetZero, ui.OffsetZero}) })
	for _, points := range invalidCurvePoints {
		if CatmullRomCurveValidateControlPoints(points) {
			t.Errorf("validateControlPoints(%v) = true", points)
		}
		throwsAssertionError(t, func() { NewCatmullRomCurve(points) })
	}
}

func TestCatmullRomCurveEnforcesContractWhenPrecomputed(t *testing.T) {
	throwsAssertionError(t, func() { NewCatmullRomCurvePrecompute([]ui.Offset{}) })
	throwsAssertionError(t, func() { NewCatmullRomCurvePrecompute([]ui.Offset{ui.OffsetZero}) })
	throwsAssertionError(t, func() { NewCatmullRomCurvePrecompute([]ui.Offset{ui.OffsetZero, ui.OffsetZero}) })
	for _, points := range invalidCurvePoints {
		throwsAssertionError(t, func() { NewCatmullRomCurvePrecompute(points) })
	}
}

// Flutter's own tests end here.

// userCurve is a curve defined outside the package's constructors: it
// embeds CurveBase and overrides TransformInternal, as a Dart subclass
// would.
type userCurve struct {
	CurveBase
	calls int
}

func (c *userCurve) TransformInternal(t float64) float64 { c.calls++; return t * t }

type userCurve2D struct{ Curve2DBase }

func (c *userCurve2D) TransformInternal(t float64) ui.Offset { return off(t, t*t) }

func TestUserCurveOverridesThroughSelf(t *testing.T) {
	c := &userCurve{}
	c.SetSelf(c)
	expectEq(t, c.Transform(0.5), 0.25)
	expectEq(t, c.Transform(0), 0) // Curve's shortcut: TransformInternal not called
	expectEq(t, c.Transform(1), 1)
	if c.calls != 1 {
		t.Errorf("TransformInternal called %d times, want 1", c.calls)
	}
	expectEq(t, c.Flipped().Transform(0.5), 0.75)
	if got := c.String(); got != "userCurve" {
		t.Errorf("String() = %q", got)
	}
	if got := c.Flipped().String(); got != "FlippedCurve(userCurve)" {
		t.Errorf("Flipped().String() = %q", got)
	}
	throwsAssertionError(t, func() { c.Transform(1.5) })

	u := &userCurve2D{}
	u.SetSelf(u)
	if got := u.Transform(0.5); got != off(0.5, 0.25) {
		t.Errorf("Transform = %v", got)
	}
	samples := slices.Collect(u.GenerateSamples())
	if len(samples) < 2 || samples[0].T != 0 || samples[len(samples)-1].T != 1 {
		t.Errorf("samples %v", samples)
	}
	if got := u.FindInverse(0.25); math.Abs(got-0.25) > 1e-5 {
		t.Errorf("FindInverse = %v", got)
	}
	if got := u.String(); got != "userCurve2D" {
		t.Errorf("String() = %q", got)
	}
}

func TestUnboundCurvePanics(t *testing.T) {
	r := panicValue(func() { (&userCurve{}).Transform(0.5) })
	if s, _ := r.(string); !strings.Contains(s, "SetSelf") {
		t.Errorf("panic %v", r)
	}
}

func TestInterfaces(t *testing.T) {
	var _ ParametricCurve[float64] = Curves.Ease
	var _ ParametricCurve[ui.Offset] = NewCatmullRomSpline(splinePoints)
	var e *ElasticInOutCurve = NewElasticInOutCurve(0.3)
	if e.Period != 0.3 || Curves.ElasticInOut.Period != 0.4 {
		t.Error("ElasticInOutCurve period")
	}
}

func TestParameterAsserts(t *testing.T) {
	throwsAssertionError(t, func() { NewSplit(1.5).Transform(0.5) })
	throwsAssertionError(t, func() { NewSplit(-0.5).Transform(0.5) })
	throwsAssertionError(t, func() { NewInterval(0.6, 0.4).Transform(0.5) })
	throwsAssertionError(t, func() { NewInterval(-0.1, 0.4).Transform(0.5) })
	throwsAssertionError(t, func() { NewThreshold(1.5).Transform(0.5) })
}

type noInternal struct{ CurveBase }

func TestTransformInternalIsUnimplemented(t *testing.T) {
	c := &noInternal{}
	c.SetSelf(c)
	if _, ok := panicValue(func() { c.Transform(0.5) }).(dart.UnimplementedError); !ok {
		t.Error("want UnimplementedError")
	}
	// Split overrides transform, so its transformInternal is the base's.
	if _, ok := panicValue(func() { NewSplit(0.5).TransformInternal(0.5) }).(dart.UnimplementedError); !ok {
		t.Error("want UnimplementedError from Split.TransformInternal")
	}
}

func TestCurveStrings(t *testing.T) {
	for _, c := range []struct {
		v    fmt.Stringer
		want string
	}{
		{Curves.Linear, "_Linear"},
		{Curves.Decelerate, "_DecelerateCurve"},
		{Curves.BounceInOut, "_BounceInOutCurve"},
		{Curves.Ease, "Cubic(0.25, 0.10, 0.25, 1.00)"},
		{NewSawTooth(3), "SawTooth(3)"},
		{NewThreshold(0.5), "Threshold"},
		{NewInterval(0.25, 0.75), "Interval(0.25⋯0.75)"},
		{NewInterval(0, 1, IntervalOptions{Curve: Curves.Ease}), "Interval(0.0⋯1.0)➩Cubic(0.25, 0.10, 0.25, 1.00)"},
		{Curves.ElasticIn, "ElasticInCurve(0.4)"},
		{NewElasticOutCurve(0.3), "ElasticOutCurve(0.3)"},
		{Curves.FastEaseInToSlowEaseOut, "ThreePointCubic "},
		{NewFlippedCurve(Curves.BounceIn), "FlippedCurve(_BounceInCurve)"},
		{NewCatmullRomCurve(curvePoints), "CatmullRomCurve"},
		{NewCatmullRomSpline(splinePoints), "CatmullRomSpline"},
		{NewCurve2DSample(0.125, off(1.0051, -2)), "[(1.01, -2.00), 0.13]"},
	} {
		if got := c.v.String(); got != c.want {
			t.Errorf("String() = %q, want %q", got, c.want)
		}
	}
	if got := NewSplit(0.25).String(); !strings.HasPrefix(got, "Split#") ||
		!strings.HasSuffix(got, "(0.25, _Linear, Cubic(0.21, 0.61, 0.35, 1.00))") {
		t.Errorf("Split.String() = %q", got)
	}
}
