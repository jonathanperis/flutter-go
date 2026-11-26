package difftest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/animation"
)

// Same curves as flutter/test/curves_test.dart.

func o(dx, dy float64) ui.Offset { return ui.NewOffset(dx, dy) }

// Circle is the Dart test's Circle: a closed Curve2D with its own seed.
type Circle struct {
	animation.Curve2DBase
	seed int
}

func newCircle(seed int) *Circle { c := &Circle{seed: seed}; c.SetSelf(c); return c }

func (c *Circle) TransformInternal(t float64) ui.Offset {
	return o(dart.Cos(t*2*math.Pi), dart.Sin(t*2*math.Pi))
}
func (c *Circle) SamplingSeed() int { return c.seed }

// Wave is the Dart test's Wave (default seed).
type Wave struct{ animation.Curve2DBase }

func newWave() *Wave { c := &Wave{}; c.SetSelf(c); return c }

func (c *Wave) TransformInternal(t float64) ui.Offset { return o(t, dart.Sin(t*10)*0.5) }

var curveSplinePoints = []ui.Offset{
	ui.OffsetZero, o(0.01, 0.25), o(0.2, 0.25), o(0.33, 0.25), o(0.5, 1.0), o(0.66, 0.75), o(1.0, 1.0),
}

var curveCurvePoints = []ui.Offset{o(0.2, 0.25), o(0.33, 0.25), o(0.5, 1.0), o(0.8, 0.75)}

func curveDifftestCurves() map[string]animation.Curve {
	C := animation.Curves
	return map[string]animation.Curve{
		"linear": C.Linear, "decelerate": C.Decelerate, "fastLinearToSlowEaseIn": C.FastLinearToSlowEaseIn,
		"fastEaseInToSlowEaseOut": C.FastEaseInToSlowEaseOut, "ease": C.Ease, "easeIn": C.EaseIn,
		"easeInToLinear": C.EaseInToLinear, "easeInSine": C.EaseInSine, "easeInQuad": C.EaseInQuad,
		"easeInCubic": C.EaseInCubic, "easeInQuart": C.EaseInQuart, "easeInQuint": C.EaseInQuint,
		"easeInExpo": C.EaseInExpo, "easeInCirc": C.EaseInCirc, "easeInBack": C.EaseInBack, "easeOut": C.EaseOut,
		"linearToEaseOut": C.LinearToEaseOut, "easeOutSine": C.EaseOutSine, "easeOutQuad": C.EaseOutQuad,
		"easeOutCubic": C.EaseOutCubic, "easeOutQuart": C.EaseOutQuart, "easeOutQuint": C.EaseOutQuint,
		"easeOutExpo": C.EaseOutExpo, "easeOutCirc": C.EaseOutCirc, "easeOutBack": C.EaseOutBack,
		"easeInOut": C.EaseInOut, "easeInOutSine": C.EaseInOutSine, "easeInOutQuad": C.EaseInOutQuad,
		"easeInOutCubic": C.EaseInOutCubic, "easeInOutCubicEmphasized": C.EaseInOutCubicEmphasized,
		"easeInOutQuart": C.EaseInOutQuart, "easeInOutQuint": C.EaseInOutQuint, "easeInOutExpo": C.EaseInOutExpo,
		"easeInOutCirc": C.EaseInOutCirc, "easeInOutBack": C.EaseInOutBack, "fastOutSlowIn": C.FastOutSlowIn,
		"slowMiddle": C.SlowMiddle, "bounceIn": C.BounceIn, "bounceOut": C.BounceOut, "bounceInOut": C.BounceInOut,
		"elasticIn": C.ElasticIn, "elasticOut": C.ElasticOut, "elasticInOut": C.ElasticInOut,

		"sawTooth1":       animation.NewSawTooth(1),
		"sawTooth3":       animation.NewSawTooth(3),
		"sawTooth7":       animation.NewSawTooth(7),
		"interval":        animation.NewInterval(0.25, 0.75),
		"intervalFull":    animation.NewInterval(0.0, 1.0),
		"intervalEaseOut": animation.NewInterval(0.1, 0.9, animation.IntervalOptions{Curve: C.EaseOut}),
		"intervalNested": animation.NewInterval(0.2, 0.6, animation.IntervalOptions{
			Curve: animation.NewInterval(0.25, 0.75, animation.IntervalOptions{Curve: C.BounceOut})}),
		"intervalElastic":      animation.NewInterval(0.0, 0.5, animation.IntervalOptions{Curve: C.ElasticOut}),
		"intervalDegenerate":   animation.NewInterval(0.5, 0.5),
		"intervalThird":        animation.NewInterval(1.0/3, 2.0/3, animation.IntervalOptions{Curve: C.EaseInOutBack}),
		"split":                animation.NewSplit(0.3),
		"splitCurves":          animation.NewSplit(0.5, animation.SplitOptions{BeginCurve: C.EaseIn, EndCurve: C.BounceOut}),
		"splitZero":            animation.NewSplit(0.0),
		"splitOne":             animation.NewSplit(1.0, animation.SplitOptions{BeginCurve: C.ElasticIn}),
		"splitThird":           animation.NewSplit(1.0/3, animation.SplitOptions{BeginCurve: C.Decelerate, EndCurve: C.ElasticInOut}),
		"threshold":            animation.NewThreshold(0.25),
		"thresholdZero":        animation.NewThreshold(0.0),
		"thresholdOne":         animation.NewThreshold(1.0),
		"thresholdHalf":        animation.NewThreshold(0.5),
		"cubic":                animation.NewCubic(0.1, 0.2, 0.3, 0.4),
		"cubicOvershoot":       animation.NewCubic(0.0, 1.5, 1.0, -0.5),
		"cubicSteep":           animation.NewCubic(0.9, 0.0, 0.1, 1.0),
		"threePointCubic":      animation.NewThreePointCubic(o(0.1, 0.05), o(0.2, 0.3), o(0.4, 0.6), o(0.6, 0.8), o(0.8, 0.95)),
		"elasticIn03":          animation.NewElasticInCurve(0.3),
		"elasticIn075":         animation.NewElasticInCurve(0.75),
		"elasticOut02":         animation.NewElasticOutCurve(0.2),
		"elasticOut1":          animation.NewElasticOutCurve(1.0),
		"elasticInOut05":       animation.NewElasticInOutCurve(0.5),
		"elasticInOut015":      animation.NewElasticInOutCurve(0.15),
		"flippedEaseIn":        animation.NewFlippedCurve(C.EaseIn),
		"flippedBounceOut":     C.BounceOut.Flipped(),
		"flippedTwice":         animation.NewFlippedCurve(animation.NewFlippedCurve(C.ElasticIn)),
		"flippedInterval":      animation.NewInterval(0.2, 0.8, animation.IntervalOptions{Curve: C.FastOutSlowIn}).Flipped(),
		"flippedEmphasized":    C.EaseInOutCubicEmphasized.Flipped(),
		"catmullRom":           animation.NewCatmullRomCurve(curveCurvePoints),
		"catmullRomPrecompute": animation.NewCatmullRomCurvePrecompute(curveCurvePoints),
		"catmullRomTension":    animation.NewCatmullRomCurve(curveCurvePoints, animation.CatmullRomCurveOptions{Tension: 0.5}),
		"catmullRomS":          animation.NewCatmullRomCurve([]ui.Offset{o(0.25, 0.1), o(0.5, 0.5), o(0.75, 0.9)}),
		"catmullRomTwo": animation.NewCatmullRomCurvePrecompute([]ui.Offset{o(0.3, 0.6), o(0.6, 0.3)},
			animation.CatmullRomCurveOptions{Tension: 0.25}),
	}
}

func curveDifftestCurves2D() map[string]animation.Curve2D {
	sh, eh, eh2 := o(0.0, -0.3), o(1.3, 1.3), o(2, 0)
	return map[string]animation.Curve2D{
		"splineHandles": animation.NewCatmullRomSpline(curveSplinePoints,
			animation.CatmullRomSplineOptions{StartHandle: &sh, EndHandle: &eh}),
		"splineDefault": animation.NewCatmullRomSpline(curveSplinePoints),
		"splineTension": animation.NewCatmullRomSpline(curveSplinePoints, animation.CatmullRomSplineOptions{Tension: 0.3}),
		"splinePrecompute": animation.NewCatmullRomSplinePrecompute(curveSplinePoints,
			animation.CatmullRomSplineOptions{Tension: 0.7, EndHandle: &eh2}),
		"splineLoop": animation.NewCatmullRomSpline([]ui.Offset{
			o(0, 0), o(1, 0), o(1, 1), o(0, 1), o(0, 0.0001), o(0.5, -0.5),
		}, animation.CatmullRomSplineOptions{Tension: 0.1}),
		"splineTensionOne": animation.NewCatmullRomSpline(curveSplinePoints, animation.CatmullRomSplineOptions{Tension: 1.0}),
		"circle":           newCircle(0),
		"circleSeed":       newCircle(12345),
		"wave":             newWave(),
	}
}

// jsonDouble reads one of the reference's doubles: a JSON number, or
// "NaN"/"Infinity"/"-Infinity".
func jsonDouble(t *testing.T, v any) float64 {
	t.Helper()
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			t.Fatalf("bad double %q", x)
		}
		return f
	}
	t.Fatalf("bad double %#v", v)
	return 0
}

// same is exact equality, bits included (so -0.0 differs from 0.0), with
// NaN equal to NaN.
func same(a, b float64) bool {
	return math.Float64bits(a) == math.Float64bits(b) || (math.IsNaN(a) && math.IsNaN(b))
}

// panicMessage is the Dart test's errorMessage: what f throws, as
// "<runtimeType>: <message or toString>".
func panicMessage(f func()) (msg any) {
	defer func() {
		switch r := recover().(type) {
		case nil:
		case dart.AssertionFailure:
			msg = "AssertionError: " + fmt.Sprint(nilToNull(r.AssertionMessage()))
		case dart.ArgumentError:
			msg = "ArgumentError: " + r.Error()
		case dart.UnsupportedError:
			msg = "UnsupportedError: " + r.Error()
		default:
			msg = fmt.Sprintf("%T: %v", r, r)
		}
	}()
	f()
	return nil
}

func nilToNull(v any) any {
	if v == nil {
		return "null"
	}
	return v
}

type curvesRef struct {
	Ts     []any `json:"ts"`
	Curves map[string]struct {
		Values  []any  `json:"values"`
		String  string `json:"string"`
		Flipped string `json:"flipped"`
	} `json:"curves"`
	Curves2D map[string]struct {
		Values       [][2]any `json:"values"`
		Samples      [][4]any `json:"samples"`
		SamplesTol   [][4]any `json:"samplesTol"`
		SamplesRange [][4]any `json:"samplesRange"`
		Inverse      []any    `json:"inverse"`
		Seed         int      `json:"seed"`
		String       string   `json:"string"`
	} `json:"curves2D"`
	Errors   map[string]any `json:"errors"`
	Validate [][3]any       `json:"validate"`
}

func TestCurvesMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/curves.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref curvesRef
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	ts := make([]float64, len(ref.Ts))
	for i, v := range ref.Ts {
		ts[i] = jsonDouble(t, v)
	}
	// The t values are computed the same way on both sides.
	for i := 0; i <= 97; i++ {
		if !same(ts[i], float64(i)/97) {
			t.Fatalf("t #%d: Go %v, Flutter %v", i, float64(i)/97, ts[i])
		}
	}
	compared := 0

	curves := curveDifftestCurves()
	if len(curves) != len(ref.Curves) {
		t.Errorf("Go has %d curves, Flutter %d", len(curves), len(ref.Curves))
	}
	for name, c := range curves {
		r, ok := ref.Curves[name]
		if !ok {
			t.Errorf("%s: missing from reference", name)
			continue
		}
		for i, tm := range ts {
			got, want := c.Transform(tm), jsonDouble(t, r.Values[i])
			compared++
			if !same(got, want) {
				t.Errorf("%s.transform(%v) = %v, Flutter %v", name, tm, got, want)
			}
		}
		if got := hashRE.ReplaceAllString(c.String(), "#HASH"); got != r.String {
			t.Errorf("%s String() = %q, Flutter %q", name, got, r.String)
		}
		if got := hashRE.ReplaceAllString(c.Flipped().String(), "#HASH"); got != r.Flipped {
			t.Errorf("%s.Flipped() String() = %q, Flutter %q", name, got, r.Flipped)
		}
	}

	curves2D := curveDifftestCurves2D()
	if len(curves2D) != len(ref.Curves2D) {
		t.Errorf("Go has %d 2D curves, Flutter %d", len(curves2D), len(ref.Curves2D))
	}
	for name, c := range curves2D {
		r, ok := ref.Curves2D[name]
		if !ok {
			t.Errorf("%s: missing from reference", name)
			continue
		}
		for i, tm := range ts {
			got := c.Transform(tm)
			compared += 2
			if !same(got.Dx, jsonDouble(t, r.Values[i][0])) || !same(got.Dy, jsonDouble(t, r.Values[i][1])) {
				t.Errorf("%s.transform(%v) = %v, Flutter %v", name, tm, got, r.Values[i])
			}
		}
		end, tol := 0.7, 1e-4
		tol2 := 1e-6
		for _, s := range []struct {
			what string
			got  []animation.Curve2DSample
			want [][4]any
		}{
			{"generateSamples()", slices.Collect(c.GenerateSamples()), r.Samples},
			{"generateSamples(tolerance: 1e-4)", slices.Collect(c.GenerateSamples(animation.Curve2DGenerateSamplesOptions{Tolerance: &tol})), r.SamplesTol},
			{"generateSamples(0.2, 0.7, 1e-6)", slices.Collect(c.GenerateSamples(animation.Curve2DGenerateSamplesOptions{
				Start: 0.2, End: &end, Tolerance: &tol2})), r.SamplesRange},
		} {
			if len(s.got) != len(s.want) {
				t.Errorf("%s.%s: %d samples, Flutter %d", name, s.what, len(s.got), len(s.want))
				continue
			}
			for i, g := range s.got {
				w := s.want[i]
				compared += 3
				if !same(g.T, jsonDouble(t, w[0])) || !same(g.Value.Dx, jsonDouble(t, w[1])) ||
					!same(g.Value.Dy, jsonDouble(t, w[2])) || g.String() != w[3] {
					t.Errorf("%s.%s[%d] = %v %v, Flutter %v", name, s.what, i, g.T, g, w)
					break
				}
			}
		}
		for i := 0; i <= 20; i++ {
			got := c.FindInverse(float64(i) / 20)
			compared++
			if want := jsonDouble(t, r.Inverse[i]); !same(got, want) {
				t.Errorf("%s.findInverse(%v) = %v, Flutter %v", name, float64(i)/20, got, want)
			}
		}
		if got := c.SamplingSeed(); got != r.Seed {
			t.Errorf("%s.samplingSeed = %d, Flutter %d", name, got, r.Seed)
		}
		if got := c.String(); got != r.String {
			t.Errorf("%s String() = %q, Flutter %q", name, got, r.String)
		}
	}

	C := animation.Curves
	nan, inf := math.NaN(), math.Inf(1)
	sh, ehInf := o(nan, 0), o(0, inf)
	zero4 := []ui.Offset{ui.OffsetZero, ui.OffsetZero, ui.OffsetZero, ui.OffsetZero}
	errors := map[string]any{
		"cubicNaN":          panicMessage(func() { C.Ease.TransformInternal(nan) }),
		"cubicInternalLow":  C.Ease.TransformInternal(-1.0),
		"cubicInternalHigh": C.Ease.TransformInternal(2.0),
		"transformHigh":     panicMessage(func() { C.Ease.Transform(1.5) }),
		"transformNaN":      panicMessage(func() { C.Ease.Transform(nan) }),
		"transform2D":       panicMessage(func() { animation.NewCatmullRomSpline(curveSplinePoints).Transform(-0.5) }),
		"splitHigh":         panicMessage(func() { animation.NewSplit(0.5).Transform(1.5) }),
		"splitBadRange":     panicMessage(func() { animation.NewSplit(1.5).Transform(0.5) }),
		"intervalBad":       panicMessage(func() { animation.NewInterval(0.6, 0.4).Transform(0.5) }),
		"thresholdBad":      panicMessage(func() { animation.NewThreshold(1.5).Transform(0.5) }),
		"splineTension": panicMessage(func() {
			animation.NewCatmullRomSpline(curveSplinePoints, animation.CatmullRomSplineOptions{Tension: 1.5})
		}),
		"splineNegTension": panicMessage(func() {
			animation.NewCatmullRomSpline(curveSplinePoints, animation.CatmullRomSplineOptions{Tension: -0.5})
		}),
		"splineFew": panicMessage(func() { animation.NewCatmullRomSpline([]ui.Offset{ui.OffsetZero}) }),
		"splineInfinite": panicMessage(func() {
			animation.NewCatmullRomSplinePrecompute([]ui.Offset{ui.OffsetZero, o(1, 1), o(inf, 2), o(3, 3)})
		}),
		"splineStartHandle": panicMessage(func() {
			animation.NewCatmullRomSplinePrecompute(curveSplinePoints, animation.CatmullRomSplineOptions{StartHandle: &sh})
		}),
		"splineEndHandle": panicMessage(func() {
			animation.NewCatmullRomSplinePrecompute(curveSplinePoints, animation.CatmullRomSplineOptions{EndHandle: &ehInf})
		}),
		"generateBadRange": panicMessage(func() {
			e := 0.5
			newWave().GenerateSamples(animation.Curve2DGenerateSamplesOptions{Start: 0.5, End: &e})
		}),
		"splineNaNSeed": panicMessage(func() { animation.NewCatmullRomSpline(zero4).GenerateSamples() }),
		"curveFew":      panicMessage(func() { animation.NewCatmullRomCurve([]ui.Offset{o(0.5, 0.5)}) }),
		"curveMulti":    panicMessage(func() { animation.NewCatmullRomCurve([]ui.Offset{o(0.5, 0.05), o(0.5, 0.95)}) }),
		"curveRange": panicMessage(func() {
			animation.NewCatmullRomCurvePrecompute([]ui.Offset{o(0.2, 0.25), o(1.01, 0.25)})
		}),
		"curveY0": panicMessage(func() {
			animation.NewCatmullRomCurve([]ui.Offset{o(0.05, 0.50), o(0.50, 0.50), o(0.75, 0.75)})
		}),
		"curveY1": panicMessage(func() {
			animation.NewCatmullRomCurve([]ui.Offset{o(0.25, 0.25), o(0.50, 0.50), o(0.95, 0.51)})
		}),
	}
	if len(errors) != len(ref.Errors) {
		t.Errorf("Go has %d error cases, Flutter %d", len(errors), len(ref.Errors))
	}
	for name, got := range errors {
		want := ref.Errors[name]
		if f, ok := got.(float64); ok {
			if !same(f, jsonDouble(t, want)) {
				t.Errorf("%s = %v, Flutter %v", name, f, want)
			}
		} else if got != want {
			t.Errorf("%s:\n  Go      %v\n  Flutter %v", name, got, want)
		}
	}

	validate := [][]ui.Offset{
		nil,
		{},
		{o(0.5, 0.5)},
		curveCurvePoints,
		{o(0.2, 0.25), o(0.01, 0.25)},
		{o(0.2, 0.25), o(1.01, 0.25)},
		{o(0.0, 0.25), o(0.5, 0.25)},
		{o(0.05, 0.50), o(0.50, 0.50), o(0.75, 0.75)},
		{o(0.25, 0.25), o(0.50, 0.50), o(0.95, 0.51)},
		{o(0.5, 0.05), o(0.5, 0.95)},
		{o(0.1, 0.9), o(0.15, 0.1), o(0.2, 0.9), o(0.9, 0.1)},
		{o(0.3, 0.6), o(0.6, 0.3)},
		// First sample's y between 1e-3 and 1e-2; X slightly below 0; X slightly above 1.
		{o(0.001, 0.01), o(0.5, 0.5)},
		{o(0.031, -1.0), o(0.5, 0.5)},
		{o(0.5, 0.5), o(0.951, -1.0)},
		{o(0.5, 0.5), o(1.0, 0.5)},
	}
	i := 0
	for _, points := range validate {
		for _, tension := range []float64{0.0, 0.5, 1.0} {
			reasons := []string{}
			withReasons := animation.CatmullRomCurveValidateControlPoints(points,
				animation.CatmullRomCurveValidateControlPointsOptions{Tension: tension, Reasons: &reasons})
			without := animation.CatmullRomCurveValidateControlPoints(points,
				animation.CatmullRomCurveValidateControlPointsOptions{Tension: tension})
			got := fmt.Sprint([]any{withReasons, without, reasons})
			want := ref.Validate[i]
			wantReasons := []string{}
			for _, r := range want[2].([]any) {
				wantReasons = append(wantReasons, r.(string))
			}
			if w := fmt.Sprint([]any{want[0], want[1], wantReasons}); got != w {
				t.Errorf("validateControlPoints(%v, tension: %v):\n  Go      %s\n  Flutter %s", points, tension, got, w)
			}
			i++
		}
	}
	if i != len(ref.Validate) {
		t.Errorf("Go has %d validations, Flutter %d", i, len(ref.Validate))
	}
	t.Logf("%d doubles compared exactly", compared)
}
