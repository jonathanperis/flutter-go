package difftest

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// jnum is a double as the Dart side writes it to JSON: non-finite values
// become their toString.
func jnum(v float64) any {
	switch {
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case math.IsNaN(v):
		return "NaN"
	}
	return v
}

func jnums(vs ...float64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = jnum(v)
	}
	return out
}

func apiColor(x Color) []any { return []any{x.A, x.R, x.G, x.B, float64(x.ColorSpace)} }
func apiRect(x Rect) []any   { return jnums(x.Left, x.Top, x.Right, x.Bottom) }

// apiErr is the Dart test's err(): the thrown error's toString, or
// "no error".
func apiErr(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case error:
				s = e.Error()
			default:
				s = fmt.Sprint(e)
			}
		}
	}()
	f()
	return "no error"
}

func strOrNull[T fmt.Stringer](v T, isNil bool) string {
	if isNil {
		return "null"
	}
	return v.String()
}

func paintState(p *Paint) []any {
	mf := p.MaskFilter()
	cf := p.ColorFilter()
	imf := p.ImageFilter()
	ms, cs, is := "null", "null", "null"
	if mf != nil {
		ms = mf.String()
	}
	if cf != nil {
		cs = cf.String()
	}
	if imf != nil {
		is = imf.String()
	}
	return []any{p.IsAntiAlias(), apiColor(p.Color()), float64(p.BlendMode()), float64(p.Style()), jnum(p.StrokeWidth()),
		float64(p.StrokeCap()), float64(p.StrokeJoin()), jnum(p.StrokeMiterLimit()), float64(p.FilterQuality()),
		p.InvertColors(), ms, cs, is}
}

func canvasState(script func(c Canvas), cull ...Rect) map[string]any {
	rec := NewPictureRecorder()
	c := NewCanvas(rec, cull...)
	script(c)
	out := map[string]any{
		"transform": jnums(c.GetTransform()...),
		"saveCount": float64(c.GetSaveCount()),
		"local":     apiRect(c.GetLocalClipBounds()),
		"device":    apiRect(c.GetDestinationClipBounds()),
	}
	rec.EndRecording()
	return out
}

func shadowList(l []Shadow) string {
	if l == nil {
		return "null"
	}
	parts := make([]string, len(l))
	for i, s := range l {
		parts[i] = s.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func tm(m TileMode) *TileMode           { return &m }
func fq(q FilterQuality) *FilterQuality { return &q }
func cop(o ClipOp) *ClipOp              { return &o }

// Same cases as flutter/test/painting_api_test.dart.
func paintingAPICases() map[string]any {
	out := map[string]any{}

	// ---- Paint ----
	out["paint.default"] = paintState(NewPaint())
	out["paint.default.toString"] = NewPaint().String()
	p1 := NewPaint()
	p1.SetColor(NewColorFrom(ColorFromOptions{Alpha: 0.3, Red: 0.1, Green: 0.7, Blue: 0.123456789}))
	p1.SetStrokeWidth(0.1)
	p1.SetStrokeMiterLimit(5.3)
	p1.SetStyle(PaintingStyleStroke)
	p1.SetStrokeCap(StrokeCapRound)
	p1.SetStrokeJoin(StrokeJoinBevel)
	p1.SetBlendMode(BlendModeMultiply)
	p1.SetIsAntiAlias(false)
	p1.SetFilterQuality(FilterQualityHigh)
	p1.SetInvertColors(true)
	mf := NewMaskFilterBlur(BlurStyleOuter, 2.25)
	p1.SetMaskFilter(&mf)
	p1.SetColorFilter(NewColorFilterMode(NewColor(0x80123456), BlendModePlus))
	p1.SetImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1.5, SigmaY: 2}))
	out["paint.set"] = paintState(p1)
	out["paint.set.toString"] = p1.String()
	out["paint.from"] = paintState(NewPaintFrom(p1))
	pc := func(c Color) []any { p := NewPaint(); p.SetColor(c); return apiColor(p.Color()) }
	out["paint.p3"] = pc(NewColorFrom(ColorFromOptions{Alpha: 1, Red: 1.2, Green: -0.1, Blue: 0.3, ColorSpace: ColorSpaceDisplayP3}))
	out["paint.ext"] = pc(NewColorFrom(ColorFromOptions{Alpha: 0.5, Red: 1.5, Green: -0.5, Blue: 0.3, ColorSpace: ColorSpaceExtendedSRGB}))
	out["paint.alphaRounding"] = pc(NewColorFrom(ColorFromOptions{Alpha: 0.07823470411233702}))
	stroke := func(f func(p *Paint)) string {
		p := NewPaint()
		p.SetStyle(PaintingStyleStroke)
		f(p)
		return p.String()
	}
	out["paint.stroke.hairline"] = stroke(func(*Paint) {})
	out["paint.stroke.miter4"] = stroke(func(p *Paint) { p.SetStrokeMiterLimit(4.0); p.SetStrokeWidth(3.25) })
	out["paint.stroke.miter10"] = stroke(func(p *Paint) { p.SetStrokeMiterLimit(10.0); p.SetStrokeWidth(1.0) })
	out["paint.stroke.square.round"] = stroke(func(p *Paint) { p.SetStrokeCap(StrokeCapSquare); p.SetStrokeJoin(StrokeJoinRound) })
	ps := func(f func(p *Paint)) string { p := NewPaint(); f(p); return p.String() }
	out["paint.color.toString"] = ps(func(p *Paint) { p.SetColor(NewColor(0x12345678)) })
	out["paint.shader.toString"] = ps(func(p *Paint) {
		p.SetShader(NewGradientLinear(Offset{}, NewOffset(1, 1), []Color{NewColor(0xFF000000), NewColor(0xFFFFFFFF)}))
	})
	out["paint.filters.toString"] = ps(func(p *Paint) {
		p.SetColorFilter(NewColorFilterLinearToSrgbGamma())
		p.SetImageFilter(NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: 1, RadiusY: 2}))
		p.SetFilterQuality(FilterQualityLow)
	})
	out["paint.blend.toString"] = ps(func(p *Paint) { p.SetBlendMode(BlendModeClear) })
	out["paint.invert.toString"] = ps(func(p *Paint) { p.SetInvertColors(true) })
	blurA := NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 3, SigmaY: 3})
	blurB := NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 3, SigmaY: 3})
	keep := NewPaint()
	keep.SetImageFilter(blurA)
	keep.SetImageFilter(blurB)
	out["paint.imageFilter.keepsEqual"] = keep.ImageFilter() == blurA
	keep.SetImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 4, SigmaY: 3}))
	out["paint.imageFilter.replacesUnequal"] = keep.ImageFilter() == blurA
	var widths, miters []any
	for _, w := range []float64{0.1, 1.0 / 3, 1e-50, 1e50, -2.5} {
		p := NewPaint()
		p.SetStrokeWidth(w)
		widths = append(widths, jnum(p.StrokeWidth()))
	}
	for _, w := range []float64{0.1, 1.0 / 3, 4.0, 100, -2.5} {
		p := NewPaint()
		p.SetStrokeMiterLimit(w)
		miters = append(miters, jnum(p.StrokeMiterLimit()))
	}
	out["paint.widths"], out["paint.miters"] = widths, miters
	third := NewMaskFilterBlur(BlurStyleNormal, 1.0/3)
	pm := NewPaint()
	pm.SetMaskFilter(&third)
	out["paint.maskSigma"] = pm.MaskFilter().String()
	pm.SetMaskFilter(nil)
	out["paint.maskNull"] = strOrNull(pm.MaskFilter(), pm.MaskFilter() == nil)

	// ---- MaskFilter ----
	out["mask.toString"] = []any{NewMaskFilterBlur(BlurStyleNormal, 0.05).String(), NewMaskFilterBlur(BlurStyleInner, 12.345).String()}
	out["mask.equal"] = []any{
		NewMaskFilterBlur(BlurStyleNormal, 1) == NewMaskFilterBlur(BlurStyleNormal, 1),
		NewMaskFilterBlur(BlurStyleNormal, 1) == NewMaskFilterBlur(BlurStyleSolid, 1),
		NewMaskFilterBlur(BlurStyleNormal, 1) == NewMaskFilterBlur(BlurStyleNormal, 2),
	}

	// ---- ColorFilter ----
	ident := []float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0}
	cfs := []*ColorFilter{
		NewColorFilterMode(NewColor(0xFF00FF00), BlendModeSrcIn),
		NewColorFilterMode(NewColor(0xFF00FF00), BlendModeSrcOut),
		NewColorFilterMode(NewColor(0x8000FF00), BlendModeSrcIn),
		NewColorFilterMatrix(ident),
		NewColorFilterMatrix([]float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0.5}),
		NewColorFilterLinearToSrgbGamma(),
		NewColorFilterSrgbToLinearGamma(),
		NewColorFilterSaturation(0.3),
	}
	var cfStr, cfShort, cfEq []any
	for _, f := range cfs {
		cfStr = append(cfStr, f.String())
		cfShort = append(cfShort, f.DebugShortDescription())
	}
	for _, a := range cfs {
		for _, b := range cfs {
			cfEq = append(cfEq, a.Equal(b))
		}
	}
	out["colorFilter.toString"], out["colorFilter.short"], out["colorFilter.equal"] = cfStr, cfShort, cfEq
	out["colorFilter.equal.copies"] = []any{
		NewColorFilterMatrix(ident).Equal(NewColorFilterMatrix(append([]float64(nil), ident...))),
		NewColorFilterSaturation(0.3).Equal(NewColorFilterSaturation(0.3)),
		NewColorFilterSaturation(0.3).Equal(NewColorFilterSaturation(0.31)),
	}
	out["colorFilter.saturation"] = []any{NewColorFilterSaturation(0).String(), NewColorFilterSaturation(1.7).String(),
		NewColorFilterSaturation(-0.25).String()}

	// ---- ImageFilter ----
	blur := func(sx, sy float64, t *TileMode) ImageFilter {
		return NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: sx, SigmaY: sy, TileMode: t})
	}
	bounds := ltrb(1, 2, 3.5, 4)
	mat := []float64{2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, -1.5, 0.25, 0, 1}
	common := func() []ImageFilter {
		return []ImageFilter{
			blur(1.9, 2.1, nil),
			blur(1.9, 2.1, tm(TileModeDecal)),
			blur(1.9, 2.1, tm(TileModeClamp)),
			blur(1.9, 2.1, tm(TileModeMirror)),
			blur(1.9, 2.1, tm(TileModeRepeated)),
			NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1.9, SigmaY: 2.1, Bounds: &bounds}),
			NewImageFilterBlur(),
			NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: 1, RadiusY: 2.5}),
			NewImageFilterErode(ImageFilterRadiusOptions{RadiusX: 0.5}),
			NewImageFilterMatrix(mat),
			NewImageFilterMatrix(mat, ImageFilterMatrixOptions{FilterQuality: fq(FilterQualityNone)}),
			NewImageFilterCompose(blur(20, 20, tm(TileModeDecal)), blur(10, 10, nil)),
		}
	}
	ifs := append(common(),
		NewImageFilterCompose(
			NewImageFilterCompose(blur(30, 30, tm(TileModeMirror)), blur(20, 20, tm(TileModeRepeated))),
			NewImageFilterCompose(NewColorFilterMode(NewColor(0xFFABCDEF), BlendModeColor),
				NewImageFilterMatrix([]float64{10, 0, 0, 0, 0, 10, 0, 0, 0, 0, 1, 0, math.Copysign(0, -1), math.Copysign(0, -1), 0, 1},
					ImageFilterMatrixOptions{FilterQuality: fq(FilterQualityLow)}))),
		NewColorFilterMode(NewColor(0xFF00FF00), BlendModeSrcIn),
	)
	var ifStr, ifShort, ifEq []any
	for _, f := range ifs {
		ifStr = append(ifStr, f.String())
		ifShort = append(ifShort, f.DebugShortDescription())
	}
	out["imageFilter.toString"], out["imageFilter.short"] = ifStr, ifShort
	ifs2 := append(common(),
		NewImageFilterCompose(blur(10, 10, nil), blur(20, 20, tm(TileModeDecal))),
		NewColorFilterMode(NewColor(0xFF00FF00), BlendModeSrcIn),
	)
	for _, a := range ifs2 {
		for _, b := range ifs2 {
			ifEq = append(ifEq, ImageFilterEqual(a, b))
		}
	}
	out["imageFilter.equal"] = ifEq
	out["imageFilter.matrix.badLength"] = apiErr(func() { NewImageFilterMatrix(make([]float64, 15)) })

	// ---- Gradient / Vertices argument checks ----
	two := []Color{NewColor(0xFF000000), NewColor(0xFFFFFFFF)}
	three := []Color{NewColor(0xFF000000), NewColor(0xFF888888), NewColor(0xFFFFFFFF)}
	x1 := NewOffset(1, 0)
	out["gradient.errors"] = []any{
		apiErr(func() { NewGradientLinear(Offset{}, x1, three) }),
		apiErr(func() { NewGradientLinear(Offset{}, x1, three, GradientLinearOptions{ColorStops: []float64{0, 1}}) }),
		apiErr(func() { NewGradientRadial(Offset{}, 1, three) }),
		apiErr(func() { NewGradientSweep(Offset{}, three, GradientSweepOptions{ColorStops: []float64{0, 0.5}}) }),
		apiErr(func() { NewGradientLinear(Offset{}, x1, two) }),
		apiErr(func() {
			NewGradientLinear(Offset{}, x1, three, GradientLinearOptions{ColorStops: []float64{0, 0.5, 1}})
		}),
	}
	tri := []Offset{{}, NewOffset(10, 0), NewOffset(0, 10)}
	out["vertices.errors"] = []any{
		apiErr(func() { NewVertices(VertexModeTriangles, tri, VerticesOptions{Colors: []Color{NewColor(0xFF000000)}}) }),
		apiErr(func() { NewVertices(VertexModeTriangles, tri, VerticesOptions{TextureCoordinates: []Offset{{}}}) }),
		apiErr(func() { NewVertices(VertexModeTriangles, tri, VerticesOptions{Indices: []int{0, 1, 3}}) }),
		apiErr(func() { NewVertices(VertexModeTriangles, tri, VerticesOptions{Indices: []int{0, 1, 2}}) }),
		apiErr(func() { NewVerticesRaw(VertexModeTriangles, make([]float32, 5)) }),
		apiErr(func() {
			NewVerticesRaw(VertexModeTriangles, make([]float32, 6), VerticesRawOptions{Colors: make([]int32, 2)})
		}),
		apiErr(func() {
			NewVerticesRaw(VertexModeTriangles, make([]float32, 6), VerticesRawOptions{TextureCoordinates: make([]float32, 4)})
		}),
		apiErr(func() {
			NewVerticesRaw(VertexModeTriangles, make([]float32, 6), VerticesRawOptions{Indices: []uint16{0, 3}})
		}),
	}

	// ---- Shadow ----
	c1 := NewColor(0x80112233)
	s1 := NewShadow(ShadowOptions{Color: &c1, Offset: NewOffset(1.5, -2), BlurRadius: 3.3})
	s2 := NewShadow(ShadowOptions{Offset: NewOffset(10, 20), BlurRadius: 0.7})
	shadowStr := func(s *Shadow) string {
		if s == nil {
			return "null"
		}
		return s.String()
	}
	out["shadow.toString"] = []any{s1.String(), NewShadow().String(), s1.Scale(2.5).String()}
	out["shadow.sigma"] = []any{ShadowConvertRadiusToSigma(0), ShadowConvertRadiusToSigma(3.3), ShadowConvertRadiusToSigma(-1), s2.BlurSigma()}
	out["shadow.lerp"] = []any{shadowStr(ShadowLerp(&s1, &s2, 0.3)), shadowStr(ShadowLerp(nil, &s2, 0.3)),
		shadowStr(ShadowLerp(&s1, nil, 0.3)), shadowStr(ShadowLerp(nil, nil, 0.3))}
	out["shadow.lerpList"] = []any{shadowList(ShadowLerpList([]Shadow{s1, s2}, []Shadow{s2}, 0.25)),
		shadowList(ShadowLerpList(nil, []Shadow{s1}, 0.5)), shadowList(ShadowLerpList(nil, nil, 0.5))}
	out["shadow.equal"] = []any{s1 == NewShadow(ShadowOptions{Color: &c1, Offset: NewOffset(1.5, -2), BlurRadius: 3.3}), s1 == s2}
	out["shadow.toPaint"] = s1.ToPaint().String()

	// ---- enums ----
	var enums []any
	for _, m := range BlendModeValues() {
		enums = append(enums, m.String())
	}
	for i := 0; i < 4; i++ {
		enums = append(enums, FilterQuality(i).String())
	}
	for i := 0; i < 3; i++ {
		enums = append(enums, StrokeCap(i).String())
	}
	for i := 0; i < 3; i++ {
		enums = append(enums, StrokeJoin(i).String())
	}
	enums = append(enums, PaintingStyleFill.String(), PaintingStyleStroke.String())
	for i := 0; i < 4; i++ {
		enums = append(enums, Clip(i).String())
	}
	for i := 0; i < 4; i++ {
		enums = append(enums, BlurStyle(i).String())
	}
	for i := 0; i < 4; i++ {
		enums = append(enums, TileMode(i).String())
	}
	for i := 0; i < 3; i++ {
		enums = append(enums, VertexMode(i).String())
	}
	for i := 0; i < 3; i++ {
		enums = append(enums, PointMode(i).String())
	}
	enums = append(enums, ClipOpDifference.String(), ClipOpIntersect.String())
	out["enums"] = enums

	// ---- Canvas state ----
	out["canvas.fresh"] = canvasState(func(Canvas) {})
	out["canvas.fresh.cull"] = canvasState(func(Canvas) {}, ltrb(10.5, 20.25, 300.75, 400.125))
	out["canvas.translate.scale"] = canvasState(func(c Canvas) {
		c.Translate(10.3, -4.7)
		c.Scale(2.5, 0.3)
		c.ClipRect(ltrb(1.1, 2.2, 50.7, 60.3))
	})
	out["canvas.scale.one"] = canvasState(func(c Canvas) {
		c.Scale(3)
		c.ClipRect(ltrb(1.1, 2.2, 50.7, 60.3), CanvasClipRectOptions{DoAntiAlias: bp(false)})
	})
	for _, angle := range []float64{0.3, math.Pi / 2, math.Pi, 1.0e-3, -2.2, 0.5, 1.3} {
		key := map[float64]string{0.3: "0.3", math.Pi / 2: "1.5707963267948966", math.Pi: "3.141592653589793",
			1.0e-3: "0.001", -2.2: "-2.2", 0.5: "0.5", 1.3: "1.3"}[angle]
		out["canvas.rotate."+key] = canvasState(func(c Canvas) {
			c.Translate(100, 50)
			c.Rotate(angle)
			c.ClipRect(ltrb(-20.5, -10.25, 30, 40))
		})
	}
	out["canvas.skew"] = canvasState(func(c Canvas) {
		c.Skew(0.25, -0.5)
		c.ClipRect(ltrb(0, 0, 100, 50))
	})
	out["canvas.transform"] = canvasState(func(c Canvas) {
		c.Transform([]float64{1.5, 0.2, 0, 0, -0.3, 0.8, 0, 0, 0, 0, 1, 0, 12.5, -7.25, 0, 1})
		c.ClipRect(ltrb(3, 4, 50, 60))
	})
	out["canvas.perspective"] = canvasState(func(c Canvas) {
		c.Transform([]float64{1, 0, 0, 0.001, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1})
		c.ClipRect(ltrb(3, 4, 50, 60))
	})
	out["canvas.transform.badLength"] = apiErr(func() { NewCanvas(NewPictureRecorder()).Transform(make([]float64, 4)) })
	out["canvas.clip.difference"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(0, 0, 100, 100))
		c.ClipRect(ltrb(0, 0, 100, 40.4), CanvasClipRectOptions{ClipOp: cop(ClipOpDifference)})
	})
	out["canvas.clip.difference.noaa"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(0, 0, 100, 100))
		c.ClipRect(ltrb(-5, 60.6, 105, 120), CanvasClipRectOptions{ClipOp: cop(ClipOpDifference), DoAntiAlias: bp(false)})
	})
	out["canvas.clip.difference.middle"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(0, 0, 100, 100))
		c.ClipRect(ltrb(20, 20, 80, 80), CanvasClipRectOptions{ClipOp: cop(ClipOpDifference)})
	})
	out["canvas.clip.unsorted"] = canvasState(func(c Canvas) { c.ClipRect(ltrb(80, 90, 10, 20)) })
	out["canvas.clip.empty"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(0, 0, 10, 10))
		c.ClipRect(ltrb(20, 20, 30, 30))
	})
	out["canvas.clip.rrect"] = canvasState(func(c Canvas) {
		c.Scale(1.5)
		c.ClipRRect(NewRRectFromLTRBR(10.2, 20.7, 90.1, 80.3, NewRadiusCircular(5)))
	})
	out["canvas.clip.rrect.noaa"] = canvasState(func(c Canvas) {
		c.Rotate(0.5)
		c.ClipRRect(NewRRectFromLTRBR(10.2, 20.7, 90.1, 80.3, NewRadiusCircular(5)), CanvasClipOptions{DoAntiAlias: bp(false)})
	})
	out["canvas.clip.rse"] = canvasState(func(c Canvas) {
		c.Translate(0.5, 0.25)
		c.ClipRSuperellipse(NewRSuperellipseFromLTRBR(10.2, 20.7, 90.1, 80.3, NewRadiusCircular(5)))
	})
	out["canvas.clip.path"] = canvasState(func(c Canvas) {
		c.Scale(2)
		p := NewPath()
		p.MoveTo(10.3, 5)
		p.LineTo(50.5, 70.2)
		p.QuadraticBezierTo(0, 100, -5.5, 30)
		p.Close()
		c.ClipPath(p)
	})
	out["canvas.save.restore"] = canvasState(func(c Canvas) {
		c.Save()
		c.Translate(10, 10)
		c.ClipRect(ltrb(0, 0, 50, 50))
		c.Save()
		c.Scale(2)
		c.Restore()
	})
	out["canvas.restoreToCount"] = canvasState(func(c Canvas) {
		c.Save()
		c.ClipRect(ltrb(0, 0, 50, 50))
		c.Save()
		c.Translate(5, 5)
		c.Save()
		c.RestoreToCount(2)
	})
	out["canvas.restoreToCount.zero"] = canvasState(func(c Canvas) {
		c.Save()
		c.Save()
		c.RestoreToCount(0)
		c.Restore()
	})
	layerBounds := ltrb(10.5, 10.5, 60.25, 70.75)
	out["canvas.saveLayer.bounds"] = canvasState(func(c Canvas) {
		c.Translate(3.3, 4.4)
		c.SaveLayer(&layerBounds, NewPaint())
	})
	out["canvas.saveLayer.null"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(0, 0, 100, 100))
		p := NewPaint()
		p.SetColor(NewColor(0x80000000))
		c.SaveLayer(nil, p)
	})
	blurPaint := func() *Paint {
		p := NewPaint()
		p.SetImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 2.5, SigmaY: 1}))
		return p
	}
	out["canvas.saveLayer.blur"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(10, 10, 100, 100))
		c.Scale(2)
		c.SaveLayer(nil, blurPaint())
	})
	small := ltrb(20, 20, 40, 40)
	out["canvas.saveLayer.blur.bounds"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(10, 10, 100, 100))
		c.SaveLayer(&small, blurPaint())
	})
	out["canvas.saveLayer.dilate"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(10, 10, 100, 100))
		p := NewPaint()
		p.SetImageFilter(NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: 3.5, RadiusY: 1}))
		c.SaveLayer(nil, p)
	})
	out["canvas.saveLayer.erode"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(10, 10, 100, 100))
		c.Rotate(0.2)
		p := NewPaint()
		p.SetImageFilter(NewImageFilterErode(ImageFilterRadiusOptions{RadiusX: 3.5, RadiusY: 1}))
		c.SaveLayer(nil, p)
	})
	out["canvas.saveLayer.restore"] = canvasState(func(c Canvas) {
		c.ClipRect(ltrb(10, 10, 100, 100))
		c.SaveLayer(&small, NewPaint())
		c.Restore()
	})
	out["canvas.many"] = canvasState(func(c Canvas) {
		c.Translate(17.25, -3.5)
		c.Rotate(0.7)
		c.Scale(1.3, 0.9)
		c.Skew(0.1, 0.05)
		c.ClipRect(ltrb(-30, -40, 120, 80))
		c.Rotate(-0.3)
		c.ClipRect(ltrb(-10, -20, 70, 60), CanvasClipRectOptions{DoAntiAlias: bp(false)})
	})
	out["canvas.nonfinite"] = canvasState(func(c Canvas) {
		c.Translate(math.Inf(1), 2)
		c.Scale(math.NaN())
		c.ClipRect(ltrb(0, 0, math.Inf(1), 10))
	})

	// ---- Canvas/recorder plumbing ----
	rec := NewPictureRecorder()
	out["recorder.before"] = fmt.Sprintf("%v %v", rec, rec.IsRecording())
	canvas := NewCanvas(rec)
	out["recorder.during"] = fmt.Sprintf("%v %v %v", rec, rec.IsRecording(), canvas)
	out["recorder.twice"] = apiErr(func() { NewCanvas(rec) })
	pic := rec.EndRecording()
	out["recorder.after"] = fmt.Sprintf("%v %v %v %v", rec, rec.IsRecording(), canvas, pic)
	out["recorder.endTwice"] = apiErr(func() { rec.EndRecording() })
	out["canvas.afterEnd"] = []any{
		apiErr(func() { canvas.GetSaveCount() }),
		apiErr(func() { canvas.GetTransform() }),
		apiErr(func() { canvas.Save() }),
		apiErr(func() { canvas.DrawRect(Rect{}, NewPaint()) }),
		apiErr(func() { canvas.ClipRect(Rect{}) }),
	}
	out["picture.toImage.bad"] = apiErr(func() { pic.ToImageSync(0, 10) })
	out["canvas.drawRawPoints.odd"] = apiErr(func() {
		NewCanvas(NewPictureRecorder()).DrawRawPoints(PointModePoints, make([]float32, 3), NewPaint())
	})
	return out
}

// toJSONShape round-trips through JSON so Go values compare with decoded
// reference values (numbers as float64, slices as []any).
func toJSONShape(t *testing.T, v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPaintingAPIMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_api.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	cases := paintingAPICases()
	if len(ref) != len(cases) {
		t.Errorf("reference has %d cases, Go has %d", len(ref), len(cases))
	}
	for name, got := range cases {
		want, ok := ref[name]
		if !ok {
			t.Errorf("%s: missing from the reference", name)
			continue
		}
		if g := toJSONShape(t, got); !reflect.DeepEqual(g, want) { // exact
			t.Errorf("%s:\n Go:      %v\n Flutter: %v", name, g, want)
		}
	}
}
