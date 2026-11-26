package difftest

// Differential test of painting's colors.dart (HSVColor, HSLColor,
// ColorSwatch, ColorProperty), matrix_utils.dart (MatrixUtils,
// debugDescribeTransform, TransformProperty) and paint_utilities.dart
// (paintZigZag, recorded on flutter_test's TestRecordingCanvas) against
// Flutter: the same cases as flutter/test/painting_colors_test.dart,
// compared exactly.

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// dartPi (semantics_test.go) is math.Pi as a variable: Dart's
// `math.pi / 3` divides doubles, while Go folds the constant expression
// exactly and rounds once (a different last bit).

func pcHSV(c painting.HSVColor) []any { return jnums(c.Alpha, c.Hue, c.Saturation, c.Value) }
func pcHSL(c painting.HSLColor) []any { return jnums(c.Alpha, c.Hue, c.Saturation, c.Lightness) }

func pcErr(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(dart.AssertionFailure); ok {
				s = "assertion"
				return
			}
			s = apiErr(func() { panic(r) })
		}
	}()
	f()
	return "no error"
}

// zigZagCalls records what _ZigZagCanvas records in the Dart test.
func zigZagCalls(c *flutter_test.TestRecordingCanvas) []any {
	var out []any
	for _, inv := range c.Invocations {
		a := inv.Invocation.PositionalArguments
		switch inv.Invocation.MemberName {
		case "save", "restore":
			out = append(out, inv.Invocation.MemberName)
		case "translate":
			out = append(out, []any{"translate", jnum(a[0].(float64)), jnum(a[1].(float64))})
		case "rotate":
			out = append(out, []any{"rotate", jnum(a[0].(float64))})
		case "drawPath":
			path := a[0].(*ui.Path)
			metrics := []any{}
			for m := range path.ComputeMetrics().All() {
				metrics = append(metrics, []any{jnum(m.Length), m.IsClosed})
				for d := 0.0; d <= m.Length; d += m.Length / 8 {
					t := m.GetTangentForOffset(d)
					metrics = append(metrics, jnums(t.Position.Dx, t.Position.Dy))
				}
			}
			out = append(out, []any{"drawPath", apiRect(path.GetBounds()), metrics})
		}
	}
	return out
}

func pcMatrices() []*vm.Matrix4 {
	persp := vm.NewMatrix4Identity()
	persp.SetEntry(3, 0, 0.002)
	persp.SetEntry(3, 1, -0.001)
	skew := vm.NewMatrix4Identity()
	skew.SetEntry(0, 1, 0.3)
	skew.SetEntry(1, 0, -0.7)
	general := vm.NewMatrix4(2, 1, 0.5, 0.001, -1, 3, 0.25, 0.002, 0.3, -0.7, 4, 0.05, 5, -2, 1, 1.5)
	rotated := vm.NewMatrix4Identity()
	rotated.RotateZ(0.3)
	rotated.RotateX(0.2)
	rotated.RotateY(-0.4)
	rotated.Translate(10.0, 20.0)
	rotated.Scale(1.5, 0.5)
	p := func(v float64) *float64 { return &v }
	h := painting.AxisHorizontal
	return []*vm.Matrix4{
		vm.NewMatrix4Identity(),
		vm.NewMatrix4TranslationValues(10.5, -3.25, 0),
		vm.NewMatrix4TranslationValues(1, 2, 3),
		vm.NewMatrix4Diagonal3Values(2, 2, 1),
		vm.NewMatrix4Diagonal3Values(-1.5, 0.75, 1),
		vm.NewMatrix4RotationZ(0.3),
		vm.NewMatrix4RotationZ(dartPi / 2),
		vm.NewMatrix4RotationX(1.1),
		vm.NewMatrix4RotationY(-0.6),
		persp,
		skew,
		general,
		rotated,
		painting.MatrixUtils.CreateCylindricalProjectionTransform(100, dartPi/3),
		painting.MatrixUtils.CreateCylindricalProjectionTransform(10, dartPi/8,
			painting.CreateCylindricalProjectionTransformOptions{Perspective: p(0.3)}),
		painting.MatrixUtils.CreateCylindricalProjectionTransform(50, -0.7,
			painting.CreateCylindricalProjectionTransformOptions{Perspective: p(0.05), Orientation: &h}),
		painting.MatrixUtils.ForceToPoint(ui.NewOffset(20, -30)),
		vm.NewMatrix4Zero(),
	}
}

var pcRects = []ui.Rect{
	ui.NewRectFromLTRB(10, 20, 30, 40),
	ui.NewRectFromLTRB(-5.5, 3.25, 100.125, 7),
	ui.NewRectFromLTRB(0, 0, 0, 0),
	ui.NewRectFromLTRB(30, 40, 10, 20),
	ui.NewRectFromLTRB(0.1, 0.2, 0.3, 0.7),
	ui.NewRectFromLTRB(-1e6, -2e6, 3e6, 4e6),
	ui.NewRectFromLTRB(0, -1.7976931348623157e+308, 800, 1.7976931348623157e+308),
	ui.NewRectFromLTRB(math.Inf(-1), 0, 10, 10),
	ui.NewRectFromLTRB(0, 0, math.Inf(1), 2),
	ui.RectLargest,
}

func paintingColorsCases() map[string]any {
	out := map[string]any{}
	mu := painting.MatrixUtils

	// ---- HSV/HSL from Color over a grid ----
	var fromColor []any
	for r := 0; r < 256; r += 17 {
		for g := 0; g < 256; g += 17 {
			for b := 0; b < 256; b += 17 {
				c := ui.NewColorFromARGB(255, r, g, b)
				v, l := painting.NewHSVColorFromColor(c), painting.NewHSLColorFromColor(c)
				fromColor = append(fromColor, append(append(pcHSV(v), pcHSL(l)...), v.ToColor().Value(), l.ToColor().Value()))
			}
		}
	}
	out["fromColor.grid"] = fromColor
	var odd []any
	for _, v := range []int{0x00000000, 0x80FF0000, 0x01020304, 0xFF7F7F80, 0xFFFE0000, 0xFF00FE01, 0xC0123456,
		0x7F808080, 0xFFFF3398, 0xFF00FFFF, 0xFFBFBF00, 0xFF0D0E0F, 0xFFFFFFFE, 0xFF010000, 0x33A1B2C3, 0xFFEB3F7B} {
		a, b := painting.NewHSVColorFromColor(ui.NewColor(v)), painting.NewHSLColorFromColor(ui.NewColor(v))
		odd = append(odd, append(append(pcHSV(a), pcHSL(b)...), a.ToColor().Value(), b.ToColor().Value()))
	}
	out["fromColor.odd"] = odd
	var fl []any
	for _, c := range []ui.Color{
		ui.NewColorFrom(ui.ColorFromOptions{Alpha: 0.5, Red: 0.1234, Green: 0.5678, Blue: 0.9}),
		ui.NewColorFrom(ui.ColorFromOptions{Alpha: 1, Red: 0.333, Green: 0.333, Blue: 0.334}),
	} {
		fl = append(fl, append(pcHSV(painting.NewHSVColorFromColor(c)), pcHSL(painting.NewHSLColorFromColor(c))...))
	}
	out["fromColor.float"] = fl

	// ---- toColor over an HSV/HSL grid ----
	units := []float64{0, 0.1, 0.25, 1.0 / 3, 0.5, 0.77, 0.999, 1}
	var toColor []any
	for h := 0.0; h <= 360; h += 7.5 {
		for _, s := range units {
			for _, x := range units {
				toColor = append(toColor, []any{painting.NewHSVColorFromAHSV(0.6, h, s, x).ToColor().Value(),
					painting.NewHSLColorFromAHSL(0.6, h, s, x).ToColor().Value()})
			}
		}
	}
	out["toColor.grid"] = toColor
	var hues []any
	for _, h := range []float64{59.999999, 60, 60.5, 119.9999999999, 120.5, 179.5, 180.5, 239.99, 240.5, 299.0000001, 300.5, 359.9999, 360, 1e-9} {
		hues = append(hues, []any{painting.NewHSVColorFromAHSV(1, h, 0.8, 0.9).ToColor().Value(),
			painting.NewHSLColorFromAHSL(1, h, 0.8, 0.4).ToColor().Value()})
	}
	out["toColor.hues"] = hues
	var alphas []any
	for _, a := range []float64{0, 0.001, 0.0019, 0.002, 0.5, 0.501, 0.998, 1} {
		alphas = append(alphas, []any{painting.NewHSVColorFromAHSV(a, 10, 0.5, 0.5).ToColor().Value(),
			painting.NewHSLColorFromAHSL(a, 10, 0.5, 0.5).ToColor().Value()})
	}
	out["toColor.alpha"] = alphas

	// ---- with*, toString, ranges ----
	v0 := painting.NewHSVColorFromAHSV(0.7, 28.0, 0.3, 0.6)
	l0 := painting.NewHSLColorFromAHSL(0.7, 28.0, 0.3, 0.6)
	out["with"] = []any{
		pcHSV(v0.WithAlpha(0.25)), pcHSV(v0.WithHue(359.5)), pcHSV(v0.WithSaturation(1)), pcHSV(v0.WithValue(0)),
		pcHSL(l0.WithAlpha(0.25)), pcHSL(l0.WithHue(359.5)), pcHSL(l0.WithSaturation(1)), pcHSL(l0.WithLightness(0)),
	}
	pointOne, pointTwo := 0.1, 0.2 // Dart's 0.1 + 0.2 is evaluated in doubles
	out["toString"] = []any{
		v0.String(), l0.String(),
		painting.NewHSVColorFromAHSV(1, 1e-7, pointOne+pointTwo, 1.0/3).String(),
		painting.NewHSLColorFromAHSL(0, 360, 1e-21, 0.5).String(),
		painting.NewHSVColorFromColor(ui.NewColor(0xFF123456)).String(),
		painting.NewHSLColorFromColor(ui.NewColor(0xFF123456)).String(),
	}
	out["asserts"] = []any{
		pcErr(func() { painting.NewHSVColorFromAHSV(1.1, 0, 0, 0) }),
		pcErr(func() { painting.NewHSVColorFromAHSV(1, 360.5, 0, 0) }),
		pcErr(func() { painting.NewHSLColorFromAHSL(1, 0, -0.1, 0) }),
		pcErr(func() { painting.NewHSLColorFromAHSL(1, 0, 0, 1.0000001) }),
		pcErr(func() { painting.NewHSVColorFromAHSV(1, 0, 0, 1) }),
	}

	// ---- lerps ----
	ts := []float64{-0.5, -0.1, 0, 0.1, 0.25, 0.5, 0.75, 0.9, 1, 1.1, 1.5, 2}
	hsv := painting.NewHSVColorFromAHSV
	pairs := [][2]painting.HSVColor{
		{v0, hsv(0.3, 128.0, 0.7, 0.2)},
		{hsv(1, 300, 1, 1), hsv(1, 60, 1, 1)},
		{hsv(1, 10, 0.5, 0.5), hsv(0, 350, 0.1, 0.9)},
		{hsv(1, 0, 1, 1), hsv(1, 180, 1, 1)},
		{hsv(1, 180, 1, 1), hsv(1, 0, 1, 1)},
		{hsv(1, 0, 0, 0), hsv(1, 360, 1, 1)},
		{hsv(0.123, 33.3, 0.456, 0.789), hsv(0.987, 213.7, 0.654, 0.321)},
	}
	var hsvLerp, hslLerp []any
	for _, p := range pairs {
		for _, t := range ts {
			r := painting.HSVColorLerp(&p[0], &p[1], t)
			hsvLerp = append(hsvLerp, append(pcHSV(*r), r.ToColor().Value()))
			a := painting.NewHSLColorFromAHSL(p[0].Alpha, p[0].Hue, p[0].Saturation, p[0].Value)
			b := painting.NewHSLColorFromAHSL(p[1].Alpha, p[1].Hue, p[1].Saturation, p[1].Value)
			l := painting.HSLColorLerp(&a, &b, t)
			hslLerp = append(hslLerp, append(pcHSL(*l), l.ToColor().Value()))
		}
	}
	out["hsv.lerp"] = hsvLerp
	out["hsl.lerp"] = hslLerp
	var hsvNull, hslNull []any
	for _, t := range []float64{0, 0.3, 0.5, 1} {
		hsvNull = append(hsvNull, pcHSV(*painting.HSVColorLerp(nil, &v0, t)), pcHSV(*painting.HSVColorLerp(&v0, nil, t)))
		hslNull = append(hslNull, pcHSL(*painting.HSLColorLerp(nil, &l0, t)), pcHSL(*painting.HSLColorLerp(&l0, nil, t)))
	}
	hsvNull = append(hsvNull, pcErr(func() { painting.HSVColorLerp(nil, &v0, 1.5) }),
		pcErr(func() { painting.HSVColorLerp(&v0, nil, -0.5) }))
	out["hsv.lerp.null"] = hsvNull
	out["hsl.lerp.null"] = hslNull

	// ---- ColorSwatch ----
	e := func(k int, c int) dart.MapEntry[int, ui.Color] {
		return dart.MapEntry[int, ui.Color]{Key: k, Value: ui.NewColor(c)}
	}
	swA := painting.NewColorSwatch(0xFF102030, []dart.MapEntry[int, ui.Color]{e(50, 0xFFFFEEDD), e(100, 0x80112233), e(500, 0xFF102030)})
	swB := painting.NewColorSwatch(0x40FFFFFF, []dart.MapEntry[int, ui.Color]{e(100, 0xFF000000), e(500, 0xFFABCDEF), e(900, 0xFF010101)})
	keys := func(s *painting.ColorSwatch[int]) []any {
		var ks []any
		for k := range s.Keys() {
			ks = append(ks, k)
		}
		return ks
	}
	missing := "null"
	if c := swA.Get(900); c != nil {
		missing = c.String()
	}
	out["swatch"] = []any{
		swA.Value(), apiColor(swA.Color), apiColor(*swA.Get(100)), missing, keys(swA), swA.String(),
		swA.Equal(painting.NewColorSwatch(0xFF102030, []dart.MapEntry[int, ui.Color]{e(500, 0xFF102030), e(50, 0xFFFFEEDD), e(100, 0x80112233)})),
		swA.Equal(swB),
	}
	swatchState := func(s *painting.ColorSwatch[int]) []any {
		r := []any{apiColor(s.Color), keys(s)}
		for k := range s.Keys() {
			r = append(r, apiColor(*s.Get(k)))
		}
		return r
	}
	var swLerp []any
	for _, t := range []float64{-0.2, 0, 0.3, 0.5, 1, 1.4} {
		swLerp = append(swLerp, swatchState(painting.ColorSwatchLerp(swA, swB, t)),
			swatchState(painting.ColorSwatchLerp(nil, swB, t)), swatchState(painting.ColorSwatchLerp(swA, nil, t)))
	}
	out["swatch.lerp"] = swLerp

	// ---- ColorProperty ----
	cp := painting.NewColorProperty("foo", ui.NewColorFromARGB(10, 20, 30, 40))
	cpNull := painting.NewColorProperty[*ui.Color]("bar", nil)
	f := false
	cpDefault := painting.NewColorProperty("baz", ui.NewColor(0xFF00FF00),
		painting.ColorPropertyOptions{DefaultValue: ui.NewColor(0xFF00FF00), ShowName: &f})
	delegate := foundation.NewDiagnosticsSerializationDelegate()
	out["colorProperty"] = []any{cp.String(), cpNull.String(), cpDefault.String(), cpDefault.Level().String(),
		cp.ToJsonMap(delegate), cpNull.ToJsonMap(delegate)}

	// ---- MatrixUtils ----
	ms := pcMatrices()
	var mats, tr, itr, tp, getAs, eq, cyl, mip, ddt []any
	for _, m := range ms {
		mats = append(mats, jnums(m.Storage()...))
		for _, r := range pcRects {
			tr = append(tr, apiRect(mu.TransformRect(m, r)))
		}
	}
	for _, m := range ms[:len(ms)-2] {
		for _, r := range pcRects[:6] {
			itr = append(itr, apiRect(mu.InverseTransformRect(m, r)))
		}
	}
	points := []ui.Offset{ui.OffsetZero, ui.NewOffset(1, 1), ui.NewOffset(-20, 30), ui.NewOffset(-1.2344, 1422434.23), ui.NewOffset(0.1, 0.7)}
	for _, m := range ms {
		for _, p := range points {
			o := mu.TransformPoint(m, p)
			tp = append(tp, jnums(o.Dx, o.Dy))
		}
		var tr, sc any
		if o := mu.GetAsTranslation(m); o != nil {
			tr = jnums(o.Dx, o.Dy)
		}
		if s := mu.GetAsScale(m); s != nil {
			sc = jnum(*s)
		}
		getAs = append(getAs, []any{tr, sc, mu.IsIdentity(m)})
		for _, b := range ms {
			eq = append(eq, mu.MatrixEquals(m, b))
		}
		ddt = append(ddt, painting.DebugDescribeTransform(m))
	}
	eq = append(eq, mu.MatrixEquals(nil, vm.NewMatrix4Identity()), mu.MatrixEquals(ms[1], nil))
	ddt = append(ddt, painting.DebugDescribeTransform(nil))
	out["matrices"] = mats
	out["transformRect"] = tr
	out["inverseTransformRect"] = itr
	out["transformPoint"] = tp
	out["getAs"] = getAs
	out["matrixEquals"] = eq
	out["debugDescribeTransform"] = ddt
	for _, radius := range []float64{0, 1, 37.5, 1000000} {
		for _, angle := range []float64{0, 0.1, dartPi / 3, -2.5, dartPi} {
			for _, perspective := range []float64{0, 0.001, 0.3, 1} {
				for _, axis := range painting.AxisValues {
					pp, ax := perspective, axis
					cyl = append(cyl, jnums(mu.CreateCylindricalProjectionTransform(radius, angle,
						painting.CreateCylindricalProjectionTransformOptions{Perspective: &pp, Orientation: &ax}).Storage()...))
				}
			}
		}
	}
	bad := 1.5
	cyl = append(cyl, pcErr(func() {
		mu.CreateCylindricalProjectionTransform(1, 1, painting.CreateCylindricalProjectionTransformOptions{Perspective: &bad})
	}))
	out["cylindrical"] = cyl
	out["forceToPoint"] = []any{jnums(mu.ForceToPoint(ui.NewOffset(20, -30)).Storage()...),
		jnums(mu.ForceToPoint(ui.NewOffset(-0.5, 1e10)).Storage()...)}
	for _, a := range ms[:14] {
		for _, b := range ms[5:14] {
			bb := b.Clone()
			mu.MultiplyInPlace(a, bb)
			mip = append(mip, jnums(bb.Storage()...))
		}
	}
	out["multiplyInPlace"] = mip
	f2 := false
	out["transformProperty"] = []any{
		painting.NewTransformProperty("transform", ms[12]).String(),
		painting.NewTransformProperty("transform", ms[12]).ToString(foundation.DiagnosticsNodeToStringOptions{
			ParentConfiguration: foundation.SingleLineTextConfiguration}),
		painting.NewTransformProperty("t", nil).String(),
		painting.NewTransformProperty("t", vm.NewMatrix4Identity(), painting.TransformPropertyOptions{ShowName: &f2,
			DefaultValue: vm.NewMatrix4Identity()}).String(),
	}

	// ---- paintZigZag ----
	var zig []any
	for _, z := range []struct {
		start, end ui.Offset
		zigs       int
		width      float64
	}{
		{ui.NewOffset(0, 0), ui.NewOffset(100, 0), 5, 3},
		{ui.NewOffset(10, 20), ui.NewOffset(-30, 70), 7, 2.5},
		{ui.NewOffset(1.5, 2.5), ui.NewOffset(1.5, 50), 1, 10},
		{ui.NewOffset(5, 5), ui.NewOffset(5, 5), 2, 1},
	} {
		c := flutter_test.NewTestRecordingCanvas()
		painting.PaintZigZag(c, ui.NewPaint(), z.start, z.end, z.zigs, z.width)
		zig = append(zig, zigZagCalls(c))
	}
	zig = append(zig, pcErr(func() {
		painting.PaintZigZag(flutter_test.NewTestRecordingCanvas(), ui.NewPaint(), ui.OffsetZero, ui.NewOffset(1, 1), 0, 1)
	}))
	out["paintZigZag"] = zig
	return out
}

func TestPaintingColorsMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_colors.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	cases := paintingColorsCases()
	if len(ref) != len(cases) {
		t.Errorf("reference has %d cases, Go has %d", len(ref), len(cases))
	}
	for name, got := range cases {
		want, ok := ref[name]
		if !ok {
			t.Errorf("%s: missing from the reference", name)
			continue
		}
		g := toJSONShape(t, got)
		if reflect.DeepEqual(g, want) { // exact
			continue
		}
		gl, gok := g.([]any)
		wl, wok := want.([]any)
		if gok && wok && len(gl) == len(wl) {
			n := 0
			for i := range gl {
				if !reflect.DeepEqual(gl[i], wl[i]) {
					if n < 5 {
						t.Errorf("%s[%d]:\n Go:      %v\n Flutter: %v", name, i, gl[i], wl[i])
					}
					n++
				}
			}
			t.Errorf("%s: %d of %d entries differ", name, n, len(gl))
			continue
		}
		t.Errorf("%s:\n Go:      %v\n Flutter: %v", name, g, want)
	}
}
