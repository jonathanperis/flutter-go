package painting

// Port of packages/flutter/test/painting/gradient_test.dart (Flutter
// 3.50.1). Skipped: the "Transforms" group (golden-file tests of
// CustomPaint widgets; the painting_decorations_pixels difftest draws the
// same rotated gradients against Flutter's Impeller output instead).
// hashCode checks are dropped. Dart's `identical` is pointer identity.

import (
	"errors"
	"math"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// gtPi is math.pi as a variable: Go folds constant expressions exactly
// (math.Pi*3/4), Dart rounds each step.
var gtPi = math.Pi

func gtColors(values ...int) []ui.Color {
	out := make([]ui.Color, len(values))
	for i, v := range values {
		out[i] = ui.NewColor(v)
	}
	return out
}

func gtF(v float64) *float64 { return &v }

func gtListDoubleMatches(x, y []float64) bool {
	if x == nil && y == nil {
		return true
	}
	if x == nil || y == nil || len(x) != len(y) {
		return false
	}
	for i := range x {
		if math.Abs(x[i]-y[i]) >= 0.0001 {
			return false
		}
	}
	return true
}

func gtListColorMatches(x, y []ui.Color) bool {
	if len(x) != len(y) {
		return false
	}
	const limit = 1.0 / 255
	for i := range x {
		if math.Abs(x[i].A-y[i].A) >= limit || math.Abs(x[i].R-y[i].R) >= limit ||
			math.Abs(x[i].G-y[i].G) >= limit || math.Abs(x[i].B-y[i].B) >= limit {
			return false
		}
	}
	return true
}

func gtMatchesLinear(t *testing.T, item Gradient, target *LinearGradient) {
	t.Helper()
	g, ok := item.(*LinearGradient)
	if !ok || g == nil || !alignmentGeometryEqual(g.Begin, target.Begin) || !alignmentGeometryEqual(g.End, target.End) ||
		g.TileMode != target.TileMode || !gradientTransformsEqual(g.Transform(), target.Transform()) ||
		!gtListColorMatches(g.Colors(), target.Colors()) || !gtListDoubleMatches(g.Stops(), target.Stops()) {
		t.Errorf("got %v, expected %v", item, target)
	}
}

func gtMatchesRadial(t *testing.T, item Gradient, target *RadialGradient) {
	t.Helper()
	g, ok := item.(*RadialGradient)
	if !ok || g == nil || !alignmentGeometryEqual(g.Center, target.Center) || g.Radius != target.Radius ||
		g.TileMode != target.TileMode || !gradientTransformsEqual(g.Transform(), target.Transform()) ||
		!alignmentGeometryEqual(g.Focal, target.Focal) || g.FocalRadius != target.FocalRadius ||
		!gtListColorMatches(g.Colors(), target.Colors()) || !gtListDoubleMatches(g.Stops(), target.Stops()) {
		t.Errorf("got %v, expected %v", item, target)
	}
}

func gtMatchesSweep(t *testing.T, item Gradient, target *SweepGradient) {
	t.Helper()
	g, ok := item.(*SweepGradient)
	if !ok || g == nil || !alignmentGeometryEqual(g.Center, target.Center) || g.StartAngle != target.StartAngle ||
		g.EndAngle != target.EndAngle || g.TileMode != target.TileMode ||
		!gradientTransformsEqual(g.Transform(), target.Transform()) ||
		!gtListColorMatches(g.Colors(), target.Colors()) || !gtListDoubleMatches(g.Stops(), target.Stops()) {
		t.Errorf("got %v, expected %v", item, target)
	}
}

func gtEqual(t *testing.T, a, b Gradient) {
	t.Helper()
	if !GradientsEqual(a, b) {
		t.Errorf("%v != %v", a, b)
	}
}

func TestLinearGradientScale(t *testing.T) {
	g := NewLinearGradient(LinearGradientOptions{Begin: AlignmentBottomRight, End: NewAlignment(0.7, 1.0),
		Colors: gtColors(0x00FFFFFF, 0x11777777, 0x44444444)})
	gtMatchesLinear(t, LinearGradientLerp(nil, g, 0.25), NewLinearGradient(LinearGradientOptions{
		Begin: AlignmentBottomRight, End: NewAlignment(0.7, 1.0), Colors: gtColors(0x00FFFFFF, 0x04777777, 0x11444444)}))
}

func TestLinearGradientLerp(t *testing.T) {
	g1 := NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopLeft, End: AlignmentBottomLeft,
		Colors: gtColors(0x33333333, 0x66666666)})
	g2 := NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopRight, End: AlignmentTopLeft,
		Colors: gtColors(0x44444444, 0x88888888)})
	gtMatchesLinear(t, LinearGradientLerp(g1, g2, 0.5), NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopCenter,
		End: AlignmentCenterLeft, Colors: gtColors(0x3B3B3B3B, 0x77777777), Stops: []float64{0, 1}}))
}

func TestLinearGradientLerpIdentical(t *testing.T) {
	if LinearGradientLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	g := NewLinearGradient(LinearGradientOptions{Colors: gtColors(0x33333333, 0x66666666)})
	if LinearGradientLerp(g, g, 0.5) != g {
		t.Error("identical")
	}
	r := NewRadialGradient(RadialGradientOptions{Colors: gtColors(0x33333333, 0x66666666)})
	if RadialGradientLerp(nil, nil, 0) != nil || RadialGradientLerp(r, r, 0.5) != r {
		t.Error("radial identical")
	}
	s := NewSweepGradient(SweepGradientOptions{Colors: gtColors(0x33333333, 0x66666666)})
	if SweepGradientLerp(nil, nil, 0) != nil || SweepGradientLerp(s, s, 0.5) != s {
		t.Error("sweep identical")
	}
	if GradientLerp(nil, nil, 0.5) != nil || GradientLerp(g, g, 0.5) != g {
		t.Error("Gradient.lerp identical")
	}
}

func TestLinearGradientLerpStops(t *testing.T) {
	g1 := NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopLeft, End: AlignmentBottomLeft,
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0.0, 0.5}})
	g2 := NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopRight, End: AlignmentTopLeft,
		Colors: gtColors(0x44444444, 0x88888888), Stops: []float64{0.5, 1.0}})
	gtMatchesLinear(t, LinearGradientLerp(g1, g2, 0.5), NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopCenter,
		End: AlignmentCenterLeft, Colors: gtColors(0x3B3B3B3B, 0x55555555, 0x77777777), Stops: []float64{0.0, 0.5, 1.0}}))
}

func TestLinearGradientLerpUnequalColors(t *testing.T) {
	g1 := NewLinearGradient(LinearGradientOptions{Colors: gtColors(0x22222222, 0x66666666)})
	g2 := NewLinearGradient(LinearGradientOptions{Colors: gtColors(0x44444444, 0x66666666, 0x88888888)})
	gtMatchesLinear(t, LinearGradientLerp(g1, g2, 0.5), NewLinearGradient(LinearGradientOptions{
		Colors: gtColors(0x33333333, 0x55555555, 0x77777777), Stops: []float64{0.0, 0.5, 1.0}}))
}

func TestLinearGradientLerpStopsUnequalColors(t *testing.T) {
	g1 := NewLinearGradient(LinearGradientOptions{Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0.0, 0.5}})
	g2 := NewLinearGradient(LinearGradientOptions{Colors: gtColors(0x44444444, 0x48484848, 0x88888888),
		Stops: []float64{0.5, 0.7, 1.0}})
	gtMatchesLinear(t, LinearGradientLerp(g1, g2, 0.5), NewLinearGradient(LinearGradientOptions{
		Colors: gtColors(0x3B3B3B3B, 0x55555555, 0x57575757, 0x77777777), Stops: []float64{0.0, 0.5, 0.7, 1.0}}))
}

func TestLinearGradientLerpTransforms(t *testing.T) {
	g1 := NewLinearGradient(LinearGradientOptions{Transform: NewGradientRotation(gtPi / 4),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0, 1}})
	g2 := NewLinearGradient(LinearGradientOptions{Transform: NewGradientRotation(gtPi / 2),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0, 1}})
	gtEqual(t, g1, LinearGradientLerp(g1, g2, 0.0))
	gtEqual(t, g2, LinearGradientLerp(g1, g2, 1.0))
	gtEqual(t, g2, LinearGradientLerp(g1, g2, 0.5))
}

func TestLinearGradientToString(t *testing.T) {
	got := NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopLeft, End: AlignmentBottomLeft,
		Transform: NewGradientRotation(1.6), Colors: gtColors(0x33333333, 0x66666666)}).String()
	want := "LinearGradient(begin: Alignment.topLeft, end: Alignment.bottomLeft, colors: [" +
		ui.NewColor(0x33333333).String() + ", " + ui.NewColor(0x66666666).String() +
		"], tileMode: TileMode.clamp, transform: GradientRotation(radians: 1.6))"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestLinearGradientDifferentTransforms(t *testing.T) {
	mk := func(r float64) *LinearGradient {
		return NewLinearGradient(LinearGradientOptions{Transform: NewGradientRotation(r), Colors: gtColors(0x33333333, 0x66666666)})
	}
	g1, g1Copy, g2 := mk(gtPi/4), mk(gtPi/4), mk(gtPi/2)
	gtEqual(t, g1, g1Copy)
	if GradientsEqual(g1, g2) {
		t.Error("different transforms compare equal")
	}
}

func gtThrowsAssertion(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		e, ok := r.(error)
		if !ok {
			t.Fatalf("got %v, want an assertion error", r)
		}
		var a dart.AssertionFailure
		if !errors.As(e, &a) {
			t.Fatalf("got %v, want an assertion error", r)
		}
	}()
	f()
}

func TestGradientsWithAlignmentDirectional(t *testing.T) {
	rect := ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)
	rtl, ltr := ui.TextDirectionRtl, ui.TextDirectionLtr
	white := gtColors(0xFFFFFFFF, 0xFFFFFFFF)
	lin := NewLinearGradient(LinearGradientOptions{Begin: AlignmentDirectionalTopStart, Colors: white})
	gtThrowsAssertion(t, func() { lin.CreateShader(rect, nil) })
	lin.CreateShader(rect, &rtl)
	lin.CreateShader(rect, &ltr)
	NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopLeft, Colors: white}).CreateShader(rect, nil)
	rad := NewRadialGradient(RadialGradientOptions{Center: AlignmentDirectionalTopStart, Colors: white})
	gtThrowsAssertion(t, func() { rad.CreateShader(rect, nil) })
	rad.CreateShader(rect, &rtl)
	rad.CreateShader(rect, &ltr)
	NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Colors: white}).CreateShader(rect, nil)
}

func TestLinearGradientWithOpacity(t *testing.T) {
	half := 0.5
	wv := func(c int) ui.Color { return ui.NewColor(c).WithValues(ui.ColorWithValuesOptions{Alpha: &half}) }
	for _, tr := range []GradientTransform{nil, NewGradientRotation(1)} {
		g := NewLinearGradient(LinearGradientOptions{Begin: AlignmentBottomRight, End: AlignmentTopCenter,
			Colors: gtColors(0xFFFFFFFF, 0xAF777777, 0x44444444), Transform: tr})
		gtEqual(t, g.WithOpacity(0.5), NewLinearGradient(LinearGradientOptions{Begin: AlignmentBottomRight,
			End: AlignmentTopCenter, Colors: []ui.Color{wv(0xFFFFFFFF), wv(0xAF777777), wv(0x44444444)}, Transform: tr}))
	}
}

func TestGradientFromColor(t *testing.T) {
	green := ui.NewColor(0xFF00FF00)
	lin := NewLinearGradient(LinearGradientOptions{Begin: AlignmentBottomRight, End: AlignmentTopCenter,
		Colors: gtColors(0xFFFFFFFF, 0xAF777777, 0x44444444), Stops: []float64{0.0, 0.3, 1.0},
		TileMode: ui.TileModeMirror, Transform: NewGradientRotation(1)})
	gtEqual(t, lin.FromColor(green), NewLinearGradient(LinearGradientOptions{Begin: AlignmentBottomRight,
		End: AlignmentTopCenter, Colors: gtColors(0xFF00FF00, 0xFF00FF00, 0xFF00FF00), Stops: []float64{0.0, 0.3, 1.0},
		TileMode: ui.TileModeMirror, Transform: NewGradientRotation(1)}))
	rad := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Radius: gtF(0.25),
		Colors: gtColors(0xFFFFFFFF, 0x44444444), Focal: AlignmentCenterRight, FocalRadius: 0.1,
		TileMode: ui.TileModeMirror, Transform: NewGradientRotation(1)})
	gtEqual(t, rad.FromColor(green), NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Radius: gtF(0.25),
		Colors: gtColors(0xFF00FF00, 0xFF00FF00), Focal: AlignmentCenterRight, FocalRadius: 0.1,
		TileMode: ui.TileModeMirror, Transform: NewGradientRotation(1)}))
	sw := NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft, StartAngle: 0.1, EndAngle: gtF(2.0),
		Colors: gtColors(0xFFFFFFFF, 0x44444444), TileMode: ui.TileModeRepeated, Transform: NewGradientRotation(1)})
	gtEqual(t, sw.FromColor(green), NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft, StartAngle: 0.1,
		EndAngle: gtF(2.0), Colors: gtColors(0xFF00FF00, 0xFF00FF00), TileMode: ui.TileModeRepeated,
		Transform: NewGradientRotation(1)}))
	// Gradient's default fromColor (a Go-only check of GradientBase).
	base := &testGradient{}
	base.Init(base, gtColors(1, 2, 3), []float64{0, 0.5, 1}, NewGradientRotation(2))
	gtEqual(t, base.FromColor(green), NewLinearGradient(LinearGradientOptions{
		Colors: gtColors(0xFF00FF00, 0xFF00FF00, 0xFF00FF00), Stops: []float64{0, 0.5, 1}, Transform: NewGradientRotation(2)}))
}

// testGradient is a Gradient written outside the library, on GradientBase.
type testGradient struct{ GradientBase }

func (g *testGradient) CreateShader(rect ui.Rect, textDirection *ui.TextDirection) ui.Shader {
	return nil
}
func (g *testGradient) Scale(factor float64) Gradient {
	s := &testGradient{}
	s.Init(s, scaledColors(g.colors, factor), g.stops, g.transform)
	return s
}
func (g *testGradient) WithOpacity(opacity float64) Gradient { return g }
func (g *testGradient) String() string                       { return "testGradient(" + colorsString(g.colors) + ")" }

func TestGradientBaseLerpDefaults(t *testing.T) {
	g := &testGradient{}
	g.Init(g, gtColors(0xFF000000, 0xFFFFFFFF), nil, nil)
	// lerp from/to null scales; with another gradient there is no lerp, so
	// Gradient.lerp scales each half.
	if got := GradientLerp(nil, g, 0.25); !gtListColorMatches(got.Colors(), gtColors(0x40000000, 0x40FFFFFF)) {
		t.Errorf("from null: %v", got)
	}
	if got := GradientLerp(g, nil, 0.25); !gtListColorMatches(got.Colors(), gtColors(0xBF000000, 0xBFFFFFFF)) {
		t.Errorf("to null: %v", got)
	}
	lin := NewLinearGradient(LinearGradientOptions{Colors: gtColors(0xFF00FF00, 0xFF0000FF)})
	if got := GradientLerp(g, lin, 0.25); !gtListColorMatches(got.Colors(), gtColors(0x80000000, 0x80FFFFFF)) {
		t.Errorf("first half: %v", got)
	}
	if got := GradientLerp(g, lin, 0.75); !gtListColorMatches(got.Colors(), gtColors(0x8000FF00, 0x800000FF)) {
		t.Errorf("second half: %v", got)
	}
}

func TestLinearGradientScalePreservesTransform(t *testing.T) {
	g := NewLinearGradient(LinearGradientOptions{Begin: AlignmentBottomRight, End: AlignmentTopLeft,
		Stops: []float64{0, 0.5, 1}, Colors: gtColors(0xFFFF0000, 0xFF00FF00, 0xFF0000FF), TileMode: ui.TileModeDecal,
		Transform: NewGradientRotation(gtPi / 4)})
	gtMatchesLinear(t, g.Scale(0.5), NewLinearGradient(LinearGradientOptions{Begin: AlignmentBottomRight,
		End: AlignmentTopLeft, Stops: []float64{0, 0.5, 1}, Colors: gtColors(0x80FF0000, 0x8000FF00, 0x800000FF),
		TileMode: ui.TileModeDecal, Transform: NewGradientRotation(gtPi / 4)}))
}

func TestRadialGradientLerp(t *testing.T) {
	g1 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Radius: gtF(20.0), Colors: gtColors(0x33333333, 0x66666666)})
	g2 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopRight, Radius: gtF(10.0), Colors: gtColors(0x44444444, 0x88888888)})
	gtMatchesRadial(t, RadialGradientLerp(g1, g2, 0.5), NewRadialGradient(RadialGradientOptions{Center: AlignmentTopCenter,
		Radius: gtF(15.0), Colors: gtColors(0x3B3B3B3B, 0x77777777), Stops: []float64{0.0, 1.0}}))
}

func TestRadialGradientLerpStops(t *testing.T) {
	g1 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Radius: gtF(20.0),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0.0, 0.5}})
	g2 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopRight, Radius: gtF(10.0),
		Colors: gtColors(0x44444444, 0x88888888), Stops: []float64{0.5, 1.0}})
	actual := RadialGradientLerp(g1, g2, 0.5)
	gtMatchesRadial(t, actual, NewRadialGradient(RadialGradientOptions{Center: AlignmentTopCenter, Radius: gtF(15.0),
		Colors: gtColors(0x3B3B3B3B, 0x55555555, 0x77777777), Stops: []float64{0.0, 0.5, 1.0}}))
	if actual.Focal != nil {
		t.Error("focal")
	}
}

func TestRadialGradientLerpUnequalColors(t *testing.T) {
	g1 := NewRadialGradient(RadialGradientOptions{Colors: gtColors(0x22222222, 0x66666666)})
	g2 := NewRadialGradient(RadialGradientOptions{Colors: gtColors(0x44444444, 0x66666666, 0x88888888)})
	gtMatchesRadial(t, RadialGradientLerp(g1, g2, 0.5), NewRadialGradient(RadialGradientOptions{
		Colors: gtColors(0x33333333, 0x55555555, 0x77777777), Stops: []float64{0.0, 0.5, 1.0}}))
	g3 := NewRadialGradient(RadialGradientOptions{Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0.0, 0.5}})
	g4 := NewRadialGradient(RadialGradientOptions{Colors: gtColors(0x44444444, 0x48484848, 0x88888888), Stops: []float64{0.5, 0.7, 1.0}})
	gtMatchesRadial(t, RadialGradientLerp(g3, g4, 0.5), NewRadialGradient(RadialGradientOptions{
		Colors: gtColors(0x3B3B3B3B, 0x55555555, 0x57575757, 0x77777777), Stops: []float64{0.0, 0.5, 0.7, 1.0}}))
}

func TestRadialGradientLerpTransforms(t *testing.T) {
	g1 := NewRadialGradient(RadialGradientOptions{Transform: NewGradientRotation(gtPi / 4),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0, 1}})
	g2 := NewRadialGradient(RadialGradientOptions{Transform: NewGradientRotation(gtPi / 2),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0, 1}})
	gtEqual(t, g1, RadialGradientLerp(g1, g2, 0.0))
	gtEqual(t, g2, RadialGradientLerp(g1, g2, 1.0))
	gtEqual(t, g2, RadialGradientLerp(g1, g2, 0.5))
}

func TestRadialGradientLerpFocal(t *testing.T) {
	g1 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Focal: AlignmentCenterLeft, Radius: gtF(20.0),
		FocalRadius: 10.0, Colors: gtColors(0x33333333, 0x66666666)})
	g2 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopRight, Focal: AlignmentCenterRight, Radius: gtF(10.0),
		FocalRadius: 5.0, Colors: gtColors(0x44444444, 0x88888888)})
	g3 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopRight, Radius: gtF(10.0), Colors: gtColors(0x44444444, 0x88888888)})
	gtMatchesRadial(t, RadialGradientLerp(g1, g2, 0.5), NewRadialGradient(RadialGradientOptions{Center: AlignmentTopCenter,
		Focal: AlignmentCenter, Radius: gtF(15.0), FocalRadius: 7.5, Colors: gtColors(0x3B3B3B3B, 0x77777777),
		Stops: []float64{0.0, 1.0}}))
	gtMatchesRadial(t, RadialGradientLerp(g1, g3, 0.5), NewRadialGradient(RadialGradientOptions{Center: AlignmentTopCenter,
		Focal: NewAlignment(-0.5, 0.0), Radius: gtF(15.0), FocalRadius: 5.0, Colors: gtColors(0x3B3B3B3B, 0x77777777),
		Stops: []float64{0.0, 1.0}}))
}

func TestRadialGradientWithOpacity(t *testing.T) {
	for _, tr := range []GradientTransform{nil, NewGradientRotation(1)} {
		g := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Focal: AlignmentCenterLeft, Radius: gtF(20.0),
			FocalRadius: 10.0, Colors: gtColors(0xFFFFFFFF, 0xAF777777, 0x44444444), Transform: tr})
		gtEqual(t, g.WithOpacity(0.5), NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft,
			Focal: AlignmentCenterLeft, Radius: gtF(20.0), FocalRadius: 10.0,
			Colors: gtColors(0x80FFFFFF, 0x80777777, 0x80444444), Transform: tr}))
	}
}

func TestRadialGradientScalePreservesTransform(t *testing.T) {
	g := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Focal: AlignmentCenterLeft, Radius: gtF(20.0),
		FocalRadius: 10.0, Stops: []float64{0, 0.5, 1}, Colors: gtColors(0xFFFF0000, 0xFF00FF00, 0xFF0000FF),
		TileMode: ui.TileModeDecal, Transform: NewGradientRotation(gtPi / 4)})
	gtMatchesRadial(t, g.Scale(0.5), NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft,
		Focal: AlignmentCenterLeft, Radius: gtF(20.0), FocalRadius: 10.0, Stops: []float64{0, 0.5, 1},
		Colors: gtColors(0x80FF0000, 0x8000FF00, 0x800000FF), TileMode: ui.TileModeDecal,
		Transform: NewGradientRotation(gtPi / 4)}))
}

func TestSweepGradientLerp(t *testing.T) {
	for _, stops := range [][2][]float64{{nil, nil}, {{0.0, 0.5}, {0.5, 1.0}}} {
		g1 := NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft, EndAngle: gtF(gtPi / 2),
			Colors: gtColors(0x33333333, 0x66666666), Stops: stops[0]})
		g2 := NewSweepGradient(SweepGradientOptions{Center: AlignmentTopRight, StartAngle: gtPi / 2, EndAngle: gtF(gtPi),
			Colors: gtColors(0x44444444, 0x88888888), Stops: stops[1]})
		want := NewSweepGradient(SweepGradientOptions{Center: AlignmentTopCenter, StartAngle: gtPi / 4,
			EndAngle: gtF(gtPi * 3 / 4), Colors: gtColors(0x3B3B3B3B, 0x77777777), Stops: []float64{0.0, 1.0}})
		if stops[0] != nil {
			want = NewSweepGradient(SweepGradientOptions{Center: AlignmentTopCenter, StartAngle: gtPi / 4,
				EndAngle: gtF(gtPi * 3 / 4), Colors: gtColors(0x3B3B3B3B, 0x55555555, 0x77777777),
				Stops: []float64{0.0, 0.5, 1.0}})
		}
		gtMatchesSweep(t, SweepGradientLerp(g1, g2, 0.5), want)
	}
}

func TestSweepGradientLerpUnequalColors(t *testing.T) {
	g1 := NewSweepGradient(SweepGradientOptions{Colors: gtColors(0x22222222, 0x66666666)})
	g2 := NewSweepGradient(SweepGradientOptions{Colors: gtColors(0x44444444, 0x66666666, 0x88888888)})
	gtMatchesSweep(t, SweepGradientLerp(g1, g2, 0.5), NewSweepGradient(SweepGradientOptions{
		Colors: gtColors(0x33333333, 0x55555555, 0x77777777), Stops: []float64{0.0, 0.5, 1.0}}))
	g3 := NewSweepGradient(SweepGradientOptions{Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0.0, 0.5}})
	g4 := NewSweepGradient(SweepGradientOptions{Colors: gtColors(0x44444444, 0x48484848, 0x88888888), Stops: []float64{0.5, 0.7, 1.0}})
	gtMatchesSweep(t, SweepGradientLerp(g3, g4, 0.5), NewSweepGradient(SweepGradientOptions{
		Colors: gtColors(0x3B3B3B3B, 0x55555555, 0x57575757, 0x77777777), Stops: []float64{0.0, 0.5, 0.7, 1.0}}))
}

func TestSweepGradientLerpTransforms(t *testing.T) {
	g1 := NewSweepGradient(SweepGradientOptions{Transform: NewGradientRotation(gtPi / 4),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0, 1}})
	g2 := NewSweepGradient(SweepGradientOptions{Transform: NewGradientRotation(gtPi / 2),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0, 1}})
	gtEqual(t, g1, SweepGradientLerp(g1, g2, 0.0))
	gtEqual(t, g2, SweepGradientLerp(g1, g2, 1.0))
	gtEqual(t, g2, SweepGradientLerp(g1, g2, 0.5))
}

func TestSweepGradientScaleAndOpacity(t *testing.T) {
	g := NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft, EndAngle: gtF(gtPi / 2),
		Colors: gtColors(0xff333333, 0xff666666)})
	gtMatchesSweep(t, g.Scale(0.5), NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft,
		EndAngle: gtF(gtPi / 2), Colors: gtColors(0x80333333, 0x80666666)}))
	for _, tr := range []GradientTransform{nil, NewGradientRotation(1)} {
		g := NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft, EndAngle: gtF(gtPi / 2),
			Colors: gtColors(0xFFFFFFFF, 0xAF777777, 0x44444444), Transform: tr})
		gtEqual(t, g.WithOpacity(0.5), NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft,
			EndAngle: gtF(gtPi / 2), Colors: gtColors(0x80FFFFFF, 0x80777777, 0x80444444), Transform: tr}))
	}
	s := NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft, StartAngle: gtPi / 4, EndAngle: gtF(gtPi * 3 / 4),
		Stops: []float64{0, 0.5, 1}, Colors: gtColors(0xFFFF0000, 0xFF00FF00, 0xFF0000FF), TileMode: ui.TileModeDecal,
		Transform: NewGradientRotation(gtPi / 4)})
	gtMatchesSweep(t, s.Scale(0.5), NewSweepGradient(SweepGradientOptions{Center: AlignmentTopLeft, StartAngle: gtPi / 4,
		EndAngle: gtF(gtPi * 3 / 4), Stops: []float64{0, 0.5, 1}, Colors: gtColors(0x80FF0000, 0x8000FF00, 0x800000FF),
		TileMode: ui.TileModeDecal, Transform: NewGradientRotation(gtPi / 4)}))
}

func TestGradientLerpRadial(t *testing.T) {
	g1 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopLeft, Radius: gtF(20.0),
		Colors: gtColors(0x33333333, 0x66666666), Stops: []float64{0.0, 1.0}})
	g2 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopCenter, Radius: gtF(15.0),
		Colors: gtColors(0x3B3B3B3B, 0x77777777), Stops: []float64{0.0, 1.0}})
	g3 := NewRadialGradient(RadialGradientOptions{Center: AlignmentTopRight, Radius: gtF(10.0),
		Colors: gtColors(0x44444444, 0x88888888), Stops: []float64{0.0, 1.0}})
	gtMatchesRadial(t, GradientLerp(g1, g3, 0.0), g1)
	gtMatchesRadial(t, GradientLerp(g1, g3, 0.5), g2)
	gtMatchesRadial(t, GradientLerp(g1, g3, 1.0), g3)
	gtMatchesRadial(t, GradientLerp(g3, g1, 0.0), g3)
	gtMatchesRadial(t, GradientLerp(g3, g1, 0.5), g2)
	gtMatchesRadial(t, GradientLerp(g3, g1, 1.0), g1)
}

func TestGradientLerpLinearToRadial(t *testing.T) {
	g1 := NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopLeft, End: AlignmentBottomRight,
		Colors: gtColors(0x33333333, 0x66666666)})
	g2 := NewRadialGradient(RadialGradientOptions{Radius: gtF(20.0), Colors: gtColors(0x44444444, 0x88888888)})
	gtEqual(t, GradientLerp(g1, g2, 0.0), g1)
	gtEqual(t, GradientLerp(g1, g2, 1.0), g2)
	gtEqual(t, GradientLerp(g1, g2, 0.5), g2.Scale(0.0))
}

func TestGradientsMissingAndMismatchedStops(t *testing.T) {
	colors := gtColors(0x11111111, 0x22222222, 0x33333333)
	rect := ui.NewRectFromLTWH(1.0, 2.0, 3.0, 4.0)
	if NewLinearGradient(LinearGradientOptions{Colors: colors}).CreateShader(rect, nil) == nil ||
		NewRadialGradient(RadialGradientOptions{Colors: colors}).CreateShader(rect, nil) == nil {
		t.Error("nil shader")
	}
	for _, g := range []Gradient{
		NewLinearGradient(LinearGradientOptions{Colors: colors, Stops: []float64{0.0, 1.0}}),
		NewRadialGradient(RadialGradientOptions{Colors: colors, Stops: []float64{0.0, 1.0}}),
	} {
		func() {
			defer func() {
				if _, ok := recover().(dart.ArgumentError); !ok {
					t.Errorf("%v: want an ArgumentError", g)
				}
			}()
			g.CreateShader(rect, nil)
		}()
	}
}

// Go-only: toString of radial/sweep gradients (stops, focal, transform),
// and GradientRotation's transform.
func TestRadialAndSweepToString(t *testing.T) {
	c := ui.NewColor(0x33333333).String()
	r := NewRadialGradient(RadialGradientOptions{Colors: gtColors(0x33333333, 0x33333333), Stops: []float64{0, 1},
		Focal: AlignmentTopLeft, FocalRadius: 0.25, Transform: NewGradientRotation(2)})
	if got, want := r.String(), "RadialGradient(center: Alignment.center, radius: 0.5, colors: ["+c+", "+c+
		"], stops: [0.0, 1.0], tileMode: TileMode.clamp, focal: Alignment.topLeft, focalRadius: 0.3, transform: GradientRotation(radians: 2.0))"; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	s := NewSweepGradient(SweepGradientOptions{Colors: gtColors(0x33333333, 0x33333333), TileMode: ui.TileModeMirror})
	if got, want := s.String(), "SweepGradient(center: Alignment.center, startAngle: 0.0, endAngle: 6.3, colors: ["+c+", "+c+
		"], tileMode: TileMode.mirror)"; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
