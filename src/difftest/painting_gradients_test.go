package difftest

// Differential test of painting's gradient.dart, box_shadow.dart,
// decoration.dart, decoration_image.dart (DecorationImage, its painters,
// paintImage) and flutter_logo.dart against Flutter: the same cases as
// flutter/test/painting_gradients_test.dart, compared exactly (doubles
// bit for bit, -0.0 and non-finite values as strings). Canvas calls are
// recorded on flutter_test's TestRecordingCanvas and turned into the shape
// the Dart side's RecCanvas writes.

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func pgj(v float64) any {
	if v == 0 && math.Signbit(v) {
		return "-0.0"
	}
	return jnum(v)
}

func pgjs(vs ...float64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = pgj(v)
	}
	return out
}

func pgRect(r ui.Rect) []any { return pgjs(r.Left, r.Top, r.Right, r.Bottom) }
func pgCol(c ui.Color) []any {
	return []any{pgj(c.A), pgj(c.R), pgj(c.G), pgj(c.B), int(c.ColorSpace)}
}
func pgCols(cs []ui.Color) []any {
	out := make([]any, len(cs))
	for i, c := range cs {
		out[i] = pgCol(c)
	}
	return out
}

func pgErr(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = pgErrString(r)
		}
	}()
	f()
	return "no error"
}

func pgErrString(r any) string {
	if e, ok := r.(error); ok {
		var a dart.AssertionFailure
		if errors.As(e, &a) {
			return "assertion"
		}
		return e.Error()
	}
	if _, ok := r.(string); ok {
		return "assertion" // ui's dartAssert and painting's plain panics stand for Dart's asserts
	}
	return "assertion"
}

func pgErrOr(f func() any) (v any) {
	defer func() {
		if r := recover(); r != nil {
			v = pgErrString(r)
		}
	}()
	return f()
}

func pgAlign(a AlignmentGeometry) any {
	if a == nil {
		return nil
	}
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	l, r := a.Resolve(&ltr), a.Resolve(&rtl)
	return []any{a.String(), pgjs(l.X, l.Y), pgjs(r.X, r.Y)}
}

func pgGrad(g Gradient) any {
	if g == nil {
		return nil
	}
	var stops, transform any
	if g.Stops() != nil {
		stops = pgjs(g.Stops()...)
	}
	if g.Transform() != nil {
		transform = g.Transform().(interface{ String() string }).String()
	}
	common := []any{foundation.ObjectRuntimeType(g, "Gradient"), pgCols(g.Colors()), stops, transform}
	switch v := g.(type) {
	case *LinearGradient:
		return append(common, pgAlign(v.Begin), pgAlign(v.End), int(v.TileMode))
	case *RadialGradient:
		return append(common, pgAlign(v.Center), pgj(v.Radius), int(v.TileMode), pgAlign(v.Focal), pgj(v.FocalRadius))
	case *SweepGradient:
		return append(common, pgAlign(v.Center), pgj(v.StartAngle), pgj(v.EndAngle), int(v.TileMode))
	}
	return common
}

func pgPaint(p *ui.Paint) any {
	var cf, mf any
	if p.ColorFilter() != nil {
		cf = p.ColorFilter().String()
	}
	if p.MaskFilter() != nil {
		mf = p.MaskFilter().String()
	}
	return []any{pgCol(p.Color()), int(p.BlendMode()), int(p.FilterQuality()), p.IsAntiAlias(), p.InvertColors(), cf,
		p.Shader() != nil, mf}
}

// pgCalls turns TestRecordingCanvas's invocations into the Dart RecCanvas's
// records.
func pgCalls(c *flutter_test.TestRecordingCanvas) []any {
	out := []any{}
	for _, r := range c.Invocations {
		inv := r.Invocation
		a := inv.PositionalArguments
		named := func(name string, def any) any {
			for _, e := range inv.NamedArguments {
				if e.Key == name {
					return e.Value
				}
			}
			return def
		}
		switch inv.MemberName {
		case "save", "restore":
			out = append(out, inv.MemberName)
		case "saveLayer":
			var b any
			if a[0] != nil {
				b = pgRect(a[0].(ui.Rect))
			}
			out = append(out, []any{"saveLayer", b, pgPaint(a[1].(*ui.Paint))})
		case "translate":
			out = append(out, []any{"translate", pgj(a[0].(float64)), pgj(a[1].(float64))})
		case "scale":
			var sy any
			if len(a) > 1 {
				sy = pgj(a[1].(float64))
			}
			out = append(out, []any{"scale", pgj(a[0].(float64)), sy})
		case "transform":
			out = append(out, []any{"transform", pgjs(a[0].([]float64)...)})
		case "clipRect":
			op := named("clipOp", ui.ClipOpIntersect)
			if p, ok := op.(*ui.ClipOp); ok {
				op = *p
			}
			aa := named("doAntiAlias", true)
			if p, ok := aa.(*bool); ok {
				aa = *p
			}
			out = append(out, []any{"clipRect", pgRect(a[0].(ui.Rect)), int(op.(ui.ClipOp)), aa})
		case "clipPath":
			aa := named("doAntiAlias", true)
			if p, ok := aa.(*bool); ok {
				aa = *p
			}
			out = append(out, []any{"clipPath", pgRect(a[0].(*ui.Path).GetBounds()), aa})
		case "drawImageRect", "drawImageNine":
			img := a[0].(*ui.Image)
			out = append(out, []any{inv.MemberName, []int{img.Width, img.Height}, pgRect(a[1].(ui.Rect)),
				pgRect(a[2].(ui.Rect)), pgPaint(a[3].(*ui.Paint))})
		case "drawPath":
			out = append(out, []any{"drawPath", pgRect(a[0].(*ui.Path).GetBounds()), pgPaint(a[1].(*ui.Paint))})
		case "drawRect":
			out = append(out, []any{"drawRect", pgRect(a[0].(ui.Rect)), pgPaint(a[1].(*ui.Paint))})
		default:
			out = append(out, "unexpected "+inv.MemberName)
		}
	}
	return out
}

// pgProvider is the Dart side's TestProvider: a stable toString/== and a
// synchronous image.
type pgProvider struct {
	ImageProviderBase
	name  string
	image *ui.Image
	scale float64
	label *string
}

func newPgProvider(name string, image *ui.Image, scale float64, label *string) *pgProvider {
	p := &pgProvider{name: name, image: image, scale: scale, label: label}
	p.SetSelf(p)
	return p
}

func (p *pgProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *pgProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	s := p.scale
	return NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(NewImageInfo(p.image.Clone(),
		ImageInfoOptions{Scale: &s, DebugLabel: p.label})))
}

func (p *pgProvider) Equal(other any) bool {
	o, ok := other.(*pgProvider)
	return ok && o.name == p.name
}

func (p *pgProvider) String() string { return "TestProvider(" + p.name + ")" }

func pgF(v float64) *float64 { return &v }

func pgGradients() []Gradient {
	c := func(vs ...int) []ui.Color {
		out := make([]ui.Color, len(vs))
		for i, v := range vs {
			out[i] = ui.NewColor(v)
		}
		return out
	}
	return []Gradient{
		NewLinearGradient(LinearGradientOptions{Colors: c(0xFF112233, 0x80FFEEDD)}),
		NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopLeft, End: NewAlignment(0.7, 1.0),
			Colors: c(0x00FFFFFF, 0x11777777, 0x44444444), Stops: []float64{0.1, 0.35, 0.9}, TileMode: ui.TileModeMirror,
			Transform: NewGradientRotation(0.7)}),
		NewLinearGradient(LinearGradientOptions{Begin: AlignmentDirectionalTopStart, End: NewAlignmentDirectional(0.25, 0.5),
			Colors: c(0xFF00FF00, 0xFF0000FF, 0xFFFF0000, 0x33333333), TileMode: ui.TileModeRepeated}),
		NewRadialGradient(RadialGradientOptions{Colors: c(0xFFFFFF00, 0xFF8000FF)}),
		NewRadialGradient(RadialGradientOptions{Center: NewAlignment(0.3, -0.2), Radius: pgF(0.8),
			Colors: c(0xFF123456, 0xFF654321, 0x80ABCDEF), Stops: []float64{0.0, 0.6, 1.0}, TileMode: ui.TileModeDecal,
			Focal: NewAlignment(-0.1, 0.4), FocalRadius: 0.15, Transform: NewGradientRotation(-1.2)}),
		NewRadialGradient(RadialGradientOptions{Center: AlignmentDirectionalCenterEnd, Radius: pgF(2.5),
			Colors: c(0xFFFFFFFF, 0xFF000000), Stops: []float64{0.25, 0.75}, Focal: NewAlignmentDirectional(-0.5, 0.0)}),
		NewSweepGradient(SweepGradientOptions{Colors: c(0xFFFF0000, 0xFF00FF00, 0xFFFF0000)}),
		NewSweepGradient(SweepGradientOptions{Center: NewAlignment(-0.4, 0.6), StartAngle: 0.5, EndAngle: pgF(4.0),
			Colors: c(0xFF0F0F0F, 0xF0F0F0F0), Stops: []float64{0.3, 0.7}, TileMode: ui.TileModeRepeated,
			Transform: NewGradientRotation(dartPi / 3)}),
	}
}

func pgShadows() []BoxShadow {
	c := func(v int) *ui.Color { x := ui.NewColor(v); return &x }
	return []BoxShadow{
		NewBoxShadow(),
		NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0}),
		NewBoxShadow(BoxShadowOptions{Color: c(0x80FF0000), Offset: ui.NewOffset(3, -2.5), BlurRadius: 7.3, SpreadRadius: 1.25}),
		NewBoxShadow(BoxShadowOptions{Color: c(0xFF00FF00), Offset: ui.NewOffset(-1, 4), BlurRadius: 0.5, BlurStyle: ui.BlurStyleOuter}),
		NewBoxShadow(BoxShadowOptions{Color: c(0x33123456), SpreadRadius: -2.0, BlurStyle: ui.BlurStyleSolid}),
		NewBoxShadow(BoxShadowOptions{BlurRadius: 12.0, BlurStyle: ui.BlurStyleInner}),
	}
}

func pgShadow(s *BoxShadow) any {
	if s == nil {
		return nil
	}
	return []any{pgCol(s.Color), pgj(s.Offset.Dx), pgj(s.Offset.Dy), pgj(s.BlurRadius), pgj(s.SpreadRadius),
		int(s.BlurStyle), s.String()}
}

var pgTs = []float64{-0.5, 0.0, 0.25, 0.5, 0.75, 1.0, 1.5}

func paintingGradientsCases(t *testing.T, tester *flutter_test.Tester) map[string]any {
	out := map[string]any{}

	// ---- Gradients ----
	gs := pgGradients()
	withNull := append([]Gradient{nil}, gs...)
	var describe, toString []any
	for _, g := range gs {
		describe = append(describe, pgGrad(g))
		toString = append(toString, g.String())
	}
	out["gradient.describe"], out["gradient.toString"] = describe, toString
	var eq []any
	for _, a := range gs {
		for _, b := range pgGradients() {
			eq = append(eq, GradientsEqual(a, b))
		}
	}
	out["gradient.equal"] = eq
	var lerps []any
	for _, a := range withNull {
		for _, b := range withNull {
			for _, tt := range pgTs {
				lerps = append(lerps, pgErrOr(func() any { return pgGrad(GradientLerp(a, b, tt)) }))
			}
		}
	}
	out["gradient.lerp"] = lerps
	lin := []*LinearGradient{nil, gs[0].(*LinearGradient), gs[1].(*LinearGradient), gs[2].(*LinearGradient)}
	rad := []*RadialGradient{nil, gs[3].(*RadialGradient), gs[4].(*RadialGradient), gs[5].(*RadialGradient)}
	swp := []*SweepGradient{nil, gs[6].(*SweepGradient), gs[7].(*SweepGradient)}
	var typed []any
	asG := func(g Gradient, isNil bool) any {
		if isNil {
			return nil
		}
		return pgGrad(g)
	}
	for _, a := range lin {
		for _, b := range lin {
			for _, tt := range pgTs {
				typed = append(typed, pgErrOr(func() any { r := LinearGradientLerp(a, b, tt); return asG(r, r == nil) }))
			}
		}
	}
	for _, a := range rad {
		for _, b := range rad {
			for _, tt := range pgTs {
				typed = append(typed, pgErrOr(func() any { r := RadialGradientLerp(a, b, tt); return asG(r, r == nil) }))
			}
		}
	}
	for _, a := range swp {
		for _, b := range swp {
			for _, tt := range pgTs {
				typed = append(typed, pgErrOr(func() any { r := SweepGradientLerp(a, b, tt); return asG(r, r == nil) }))
			}
		}
	}
	out["gradient.lerp.typed"] = typed
	var scale, opacity, fromColor []any
	for _, g := range gs {
		for _, f := range []float64{-0.5, 0.0, 0.3, 1.0, 2.0} {
			scale = append(scale, pgErrOr(func() any { return pgGrad(g.Scale(f)) }))
		}
		for _, o := range []float64{0.0, 0.37, 1.0, 1.5} {
			opacity = append(opacity, pgErrOr(func() any { return pgGrad(g.WithOpacity(o)) }))
		}
		fromColor = append(fromColor, pgGrad(g.FromColor(ui.NewColor(0xFF0080FF))))
	}
	out["gradient.scale"], out["gradient.withOpacity"], out["gradient.fromColor"] = scale, opacity, fromColor
	// Duplicate stops (the sample's first/last-stop and lastIndexWhere
	// rules) and gradients that differ only in their stops.
	c4 := []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF00FF00), ui.NewColor(0xFF0000FF), ui.NewColor(0x80FFFFFF)}
	dup := []Gradient{
		NewLinearGradient(LinearGradientOptions{Colors: c4, Stops: []float64{0.25, 0.25, 0.6, 1.0}}),
		NewLinearGradient(LinearGradientOptions{Colors: c4, Stops: []float64{0.0, 0.5, 0.5, 1.0}}),
		NewRadialGradient(RadialGradientOptions{Colors: c4, Stops: []float64{0.0, 0.4, 0.75, 0.75}}),
		NewSweepGradient(SweepGradientOptions{Colors: c4, Stops: []float64{0.3, 0.3, 0.3, 0.9}}),
	}
	var dupLerps []any
	for _, a := range dup {
		for _, b := range append(append([]Gradient{}, dup...), gs[1], gs[4], gs[7]) {
			for _, tt := range []float64{0.0, 0.3, 1.0} {
				dupLerps = append(dupLerps, pgErrOr(func() any { return pgGrad(GradientLerp(a, b, tt)) }))
			}
		}
	}
	out["gradient.duplicateStops"] = dupLerps
	bw := []ui.Color{ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF)}
	lg := func(s []float64) Gradient { return NewLinearGradient(LinearGradientOptions{Colors: bw, Stops: s}) }
	out["gradient.equalStops"] = []bool{
		GradientsEqual(lg(nil), lg([]float64{0, 1})),
		GradientsEqual(lg([]float64{0, 0.5}), lg([]float64{0, 1})),
		GradientsEqual(NewRadialGradient(RadialGradientOptions{Colors: bw}),
			NewRadialGradient(RadialGradientOptions{Colors: bw, Stops: []float64{0, 1}})),
		GradientsEqual(NewSweepGradient(SweepGradientOptions{Colors: bw, Stops: []float64{0, 0.5}}),
			NewSweepGradient(SweepGradientOptions{Colors: bw, Stops: []float64{0, 1}})),
		GradientsEqual(lg([]float64{0, 1}), lg([]float64{0, 1})),
	}
	shaderRect := ui.NewRectFromLTWH(1.0, 2.0, 30.0, 40.0)
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	var shaders []any
	for _, g := range gs {
		for _, d := range []*ui.TextDirection{nil, &ltr, &rtl} {
			shaders = append(shaders, pgErr(func() { g.CreateShader(shaderRect, d) }))
		}
	}
	black := ui.NewColor(0xFF000000)
	shaders = append(shaders,
		pgErr(func() {
			NewLinearGradient(LinearGradientOptions{Colors: []ui.Color{black, black, black}, Stops: []float64{0, 1}}).
				CreateShader(shaderRect, nil)
		}),
		pgErr(func() {
			NewSweepGradient(SweepGradientOptions{Colors: []ui.Color{black, black}, StartAngle: 2, EndAngle: pgF(1)}).
				CreateShader(shaderRect, nil)
		}))
	out["gradient.createShader"] = shaders
	var rotations []any
	for _, r := range []float64{0.0, 0.5, dartPi / 4, dartPi / 2, -1.3, 3.0, 7.5} {
		for _, b := range []ui.Rect{ui.NewRectFromLTWH(0, 0, 300, 400), ui.NewRectFromLTRB(-10.5, 3.25, 17.75, 99.0), ui.RectZero} {
			rotations = append(rotations, pgjs(NewGradientRotation(r).Transform(b, &rtl).Storage()...))
		}
		rotations = append(rotations, NewGradientRotation(r).String())
	}
	rotations = append(rotations, []bool{NewGradientRotation(1.0) == NewGradientRotation(1.0),
		NewGradientRotation(1.0) == NewGradientRotation(1.5)})
	out["gradientRotation"] = rotations

	// ---- BoxShadow ----
	ss := pgShadows()
	ssNull := []*BoxShadow{nil}
	for i := range ss {
		ssNull = append(ssNull, &ss[i])
	}
	var shLerp []any
	for _, a := range ssNull {
		for _, b := range ssNull {
			for _, tt := range pgTs {
				shLerp = append(shLerp, pgErrOr(func() any { return pgShadow(BoxShadowLerp(a, b, tt)) }))
			}
		}
	}
	out["boxShadow.lerp"] = shLerp
	lists := [][]BoxShadow{nil, {}, {ss[1]}, {ss[2], ss[3]}, {ss[5], ss[4], ss[0]}}
	var lerpList []any
	for _, a := range lists {
		for _, b := range lists {
			for _, tt := range []float64{0.0, 0.3, 1.0} {
				lerpList = append(lerpList, pgErrOr(func() any {
					r := BoxShadowLerpList(a, b, tt)
					if r == nil {
						return nil
					}
					l := []any{}
					for i := range r {
						l = append(l, pgShadow(&r[i]))
					}
					return l
				}))
			}
		}
	}
	out["boxShadow.lerpList"] = lerpList
	var shScale, copyWith, shEq, toPaint []any
	for _, s := range ss {
		for _, f := range []float64{0.0, 0.5, 3.0, -1.0} {
			shScale = append(shScale, pgErrOr(func() any { r := s.Scale(f); return pgShadow(&r) }))
		}
		c1, c2 := ui.NewColor(0xFF445566), 2.0
		o, sp, st := ui.NewOffset(1, 1), 9.0, ui.BlurStyleOuter
		for _, r := range []BoxShadow{s.CopyWith(BoxShadowCopyWithOptions{}),
			s.CopyWith(BoxShadowCopyWithOptions{Color: &c1, BlurRadius: &c2}),
			s.CopyWith(BoxShadowCopyWithOptions{Offset: &o, SpreadRadius: &sp, BlurStyle: &st})} {
			copyWith = append(copyWith, pgShadow(&r))
		}
		for _, b := range pgShadows() {
			shEq = append(shEq, s == b)
		}
	}
	defer func(old bool) { DebugDisableShadows = old }(DebugDisableShadows)
	for _, disable := range []bool{false, true} {
		DebugDisableShadows = disable
		for _, s := range ss {
			toPaint = append(toPaint, []any{pgPaint(s.ToPaint()), pgj(s.BlurSigma())})
		}
	}
	out["boxShadow.scale"], out["boxShadow.copyWith"], out["boxShadow.equal"], out["boxShadow.toPaint"] =
		shScale, copyWith, shEq, toPaint

	// ---- DecorationImage ----
	img := func(w, h int) *ui.Image {
		return flutter_test.AwaitFuture(tester, flutter_test.CreateTestImage(flutter_test.CreateTestImageOptions{Width: w, Height: h, NoCache: true}))
	}
	img100, img37x23, img512 := img(100, 100), img(37, 23), img(512, 300)
	label := "b.png"
	pa, pb := newPgProvider("a", img100, 1.0, nil), newPgProvider("b", img37x23, 2.0, &label)
	fitP := func(f BoxFit) *BoxFit { return &f }
	repP := func(r ImageRepeat) *ImageRepeat { return &r }
	fq := ui.FilterQualityNone
	slice, slice2 := ui.NewRectFromLTRB(10, 20, 40, 60), ui.NewRectFromLTWH(1, 2, 3, 4)
	dis := []DecorationImage{
		NewDecorationImage(DecorationImageOptions{Image: pa}),
		NewDecorationImage(DecorationImageOptions{Image: pb, Fit: fitP(BoxFitCover), Alignment: AlignmentTopRight, Opacity: pgF(0.5)}),
		NewDecorationImage(DecorationImageOptions{Image: pa, Placeholder: pb,
			ColorFilter: ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeSrc), Fit: fitP(BoxFitFill),
			Alignment: AlignmentDirectionalBottomStart, CenterSlice: &slice, Repeat: repP(ImageRepeatRepeatX),
			MatchTextDirection: true, Scale: pgF(1.25), Opacity: pgF(0.75), FilterQuality: &fq, InvertColors: true,
			IsAntiAlias: true}),
		NewDecorationImage(DecorationImageOptions{Image: pb, Fit: fitP(BoxFitScaleDown), Repeat: repP(ImageRepeatRepeat), Scale: pgF(0.04)}),
		NewDecorationImage(DecorationImageOptions{Image: pa, Fit: fitP(BoxFitFill)}),
		NewDecorationImage(DecorationImageOptions{Image: pa, Fit: fitP(BoxFitScaleDown), CenterSlice: &slice2}),
	}
	var diStr, diEq, diLerp []any
	for _, d := range dis {
		diStr = append(diStr, d.String())
		for _, e := range dis {
			diEq = append(diEq, d.Equal(e))
		}
	}
	diEq = append(diEq, NewDecorationImage(DecorationImageOptions{Image: pa}).Equal(
		NewDecorationImage(DecorationImageOptions{Image: newPgProvider("a", img37x23, 1.0, nil)})))
	disNull := []DecorationImage{nil, dis[0], dis[1], dis[2]}
	for _, a := range disNull {
		for _, b := range disNull {
			for _, tt := range []float64{0.0, 0.4, 1.0} {
				r := DecorationImageLerp(a, b, tt)
				if r == nil {
					diLerp = append(diLerp, "null")
				} else {
					diLerp = append(diLerp, r.String())
				}
			}
		}
	}
	out["decorationImage.toString"], out["decorationImage.equal"], out["decorationImage.lerp"] = diStr, diEq, diLerp

	// ---- paintImage ----
	var pis []any
	fits := []*BoxFit{nil}
	for _, f := range BoxFitValues {
		fits = append(fits, fitP(f))
	}
	for _, ri := range []struct {
		rect  ui.Rect
		image *ui.Image
	}{{ui.NewRectFromLTWH(10.0, 20.0, 250.0, 120.0), img100}, {ui.NewRectFromLTRB(-3.5, 7.25, 41.0, 99.5), img37x23}} {
		for _, fit := range fits {
			for _, a := range []Alignment{AlignmentCenter, AlignmentBottomRight, NewAlignment(-0.3, 0.7)} {
				for _, repeat := range ImageRepeatValues {
					for _, flip := range []bool{false, true} {
						for _, sc := range []float64{1.0, 2.5} {
							for _, sl := range []*ui.Rect{nil, {Left: 5, Top: 6, Right: 20, Bottom: 18}} {
								c := flutter_test.NewTestRecordingCanvas()
								e := pgErr(func() {
									PaintImage(c, ri.rect, ri.image, PaintImageOptions{Fit: fit, Alignment: a, Repeat: repP(repeat),
										FlipHorizontally: flip, Scale: pgF(sc), CenterSlice: sl})
								})
								pis = append(pis, []any{e, pgCalls(c)})
							}
						}
					}
				}
			}
		}
	}
	out["paintImage"] = pis
	var pis2 []any
	for _, o := range []struct {
		opacity float64
		cf      *ui.ColorFilter
		invert  bool
		fq      ui.FilterQuality
		aa      bool
		bm      ui.BlendMode
	}{
		{0.5, nil, false, ui.FilterQualityMedium, false, ui.BlendModeSrcOver},
		{1.7, ui.NewColorFilterMode(ui.NewColor(0xFF0000FF), ui.BlendModeMultiply), true, ui.FilterQualityNone, true, ui.BlendModePlus},
		{-0.2, ui.NewColorFilterLinearToSrgbGamma(), false, ui.FilterQualityHigh, true, ui.BlendModeDstIn},
		{0.123, nil, true, ui.FilterQualityLow, false, ui.BlendModeModulate},
	} {
		c := flutter_test.NewTestRecordingCanvas()
		PaintImage(c, ui.NewRectFromLTWH(0, 0, 64, 48), img37x23, PaintImageOptions{Opacity: &o.opacity, ColorFilter: o.cf,
			InvertColors: o.invert, FilterQuality: &o.fq, IsAntiAlias: o.aa, BlendMode: &o.bm})
		pis2 = append(pis2, pgCalls(c))
	}
	empty := flutter_test.NewTestRecordingCanvas()
	PaintImage(empty, ui.NewRectFromLTWH(5, 5, 0, 10), img100)
	pis2 = append(pis2, pgCalls(empty))
	out["paintImage.paint"] = pis2

	DebugInvertOversizedImages = true
	oldOnError := foundation.FlutterErrorOnError
	messages := []any{}
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) {
		lib := "null"
		if d.Library != nil {
			lib = *d.Library
		}
		messages = append(messages, d.ExceptionAsString()+"|"+lib+"|"+d.Context.String())
	}
	var inverted []any
	big, fits2 := "Big", "Fits"
	small, allowance := "Small", "Allowance"
	for _, o := range []struct {
		rect  ui.Rect
		label *string
		fit   BoxFit
	}{
		{ui.NewRectFromLTWH(50.0, 50.0, 10.0, 7.0), &big, BoxFitFill},
		{ui.NewRectFromLTWH(-5.0, 3.0, 20.5, 11.25), nil, BoxFitContain},
		{ui.NewRectFromLTWH(0, 0, 400, 290), &fits2, BoxFitFill},
		{ui.NewRectFromLTWH(0, 0, 90, 90), &small, BoxFitCover},
		{ui.NewRectFromLTWH(0, 0, 130, 120), &allowance, BoxFitFill},
	} {
		c := flutter_test.NewTestRecordingCanvas()
		PaintImage(c, o.rect, img512, PaintImageOptions{DebugImageLabel: o.label, Fit: fitP(o.fit), FlipHorizontally: true})
		inverted = append(inverted, pgCalls(c))
	}
	inverted = append(inverted, messages)
	foundation.FlutterErrorOnError = oldOnError
	DebugInvertOversizedImages = false
	out["paintImage.inverted"] = inverted

	var painters []any
	for _, d := range dis {
		for _, dir := range []ui.TextDirection{ui.TextDirectionLtr, ui.TextDirectionRtl} {
			c := flutter_test.NewTestRecordingCanvas()
			p := d.CreatePainter(func() {})
			clip := ui.NewPath()
			clip.AddRect(ui.NewRectFromLTWH(0, 0, 50, 60))
			e := pgErr(func() {
				p.Paint(c, ui.NewRectFromLTWH(2, 3, 120, 80), clip, ImageConfiguration{TextDirection: &dir})
			})
			painters = append(painters, []any{e, pgCalls(c)})
			p.Dispose()
		}
	}
	for _, l := range []struct {
		a, b DecorationImage
		t    float64
	}{{dis[0], dis[1], 0.3}, {nil, dis[1], 0.6}, {dis[1], nil, 0.25}, {dis[3], dis[0], 0.5}} {
		c := flutter_test.NewTestRecordingCanvas()
		p := DecorationImageLerp(l.a, l.b, l.t).CreatePainter(func() {})
		blend, mode := 0.8, ui.BlendModeMultiply
		p.Paint(c, ui.NewRectFromLTWH(0, 0, 64, 64), nil, ImageConfiguration{TextDirection: &ltr},
			DecorationImagePainterPaintOptions{Blend: &blend, BlendMode: &mode})
		painters = append(painters, pgCalls(c))
		p.Dispose()
	}
	out["decorationImage.paint"] = painters

	// ---- Decoration.lerp / FlutterLogoDecoration ----
	c1, c2 := ui.NewColor(0xFFD4F144), ui.NewColor(0xFF81D4FA)
	l0 := NewFlutterLogoDecoration()
	l1 := NewFlutterLogoDecoration(FlutterLogoDecorationOptions{TextColor: &c1, Style: FlutterLogoStyleStacked,
		Margin: EdgeInsets{Left: 10, Top: 10, Right: 10, Bottom: 10}})
	l2 := NewFlutterLogoDecoration(FlutterLogoDecorationOptions{TextColor: &c2, Style: FlutterLogoStyleHorizontal,
		Margin: EdgeInsets{Left: 1, Top: 2, Right: 3, Bottom: 4}})
	logos := []*FlutterLogoDecoration{nil, l0, l1, l2}
	logo := func(d Decoration) any {
		if d == nil {
			return nil
		}
		ld, ok := d.(*FlutterLogoDecoration)
		if !ok {
			return foundation.ObjectRuntimeType(d, "Decoration")
		}
		return []any{pgCol(ld.TextColor), int(ld.Style), pgjs(ld.Margin.Left, ld.Margin.Top, ld.Margin.Right, ld.Margin.Bottom),
			ld.IsComplex(), ld.String()}
	}
	asDec := func(d *FlutterLogoDecoration) Decoration {
		if d == nil {
			return nil
		}
		return d
	}
	var logoLerp, logoEq []any
	logoEq = append(logoEq, FlutterLogoDecorationLerp(nil, l0, 0.5).Equal(l0),
		FlutterLogoDecorationLerp(nil, l0, 0.5).Equal(FlutterLogoDecorationLerp(nil, l0, 0.5)),
		FlutterLogoDecorationLerp(l1, nil, 0.25).Equal(l1))
	var unrelated []any
	for _, tt := range pgTs {
		unrelated = append(unrelated, logo(DecorationLerp(l1, NewPlainDecoration(), tt)),
			logo(DecorationLerp(NewPlainDecoration(), l2, tt)))
	}
	out["decoration.lerp.unrelated"] = unrelated
	for _, a := range logos {
		for _, b := range logos {
			for _, tt := range pgTs {
				logoLerp = append(logoLerp,
					pgErrOr(func() any { return logo(asDec(FlutterLogoDecorationLerp(a, b, tt))) }),
					pgErrOr(func() any { return logo(DecorationLerp(asDec(a), asDec(b), tt)) }))
			}
			logoEq = append(logoEq, (a == nil && b == nil) || (a != nil && b != nil && a.Equal(b)))
		}
	}
	out["flutterLogo.lerp"], out["flutterLogo.equal"] = logoLerp, logoEq
	var logoPaint []any
	for _, d := range []*FlutterLogoDecoration{l0,
		NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Margin: EdgeInsets{Left: 3, Top: 5, Right: 7, Bottom: 11}}),
		FlutterLogoDecorationLerp(nil, l0, 0.4), FlutterLogoDecorationLerp(l0, nil, 0.9)} {
		for _, g := range []struct {
			o ui.Offset
			s ui.Size
		}{{ui.OffsetZero, ui.NewSize(202, 202)}, {ui.NewOffset(10.5, -3), ui.NewSize(64, 40)},
			{ui.NewOffset(1, 2), ui.NewSize(17.3, 99.9)}, {ui.OffsetZero, ui.NewSize(5, 5)}} {
			c := flutter_test.NewTestRecordingCanvas()
			d.CreateBoxPainter(nil).Paint(c, g.o, ImageConfiguration{Size: &g.s})
			logoPaint = append(logoPaint, pgCalls(c))
		}
	}
	out["flutterLogo.paint"] = logoPaint
	var styles, repeats []any
	for _, s := range FlutterLogoStyleValues {
		styles = append(styles, s.String())
	}
	for _, r := range ImageRepeatValues {
		repeats = append(repeats, r.String())
	}
	out["flutterLogo.misc"] = []any{
		pgRect(l0.GetClipPath(ui.NewRectFromLTWH(1, 2, 3, 4), ltr).GetBounds()),
		l0.HitTest(ui.NewSize(10, 10), ui.NewOffset(100, 100), nil),
		l0.Padding().(EdgeInsets).String(),
		l0.ToStringShort(),
		styles, repeats,
	}
	return out
}

func TestPaintingGradientsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_gradients.json")
	if err != nil {
		t.Fatalf("no reference (%v); run the Flutter side: flutter test test/painting_gradients_test.dart", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	tester := flutter_test.New(t)
	// flutter_tester's view has a device pixel ratio of 3.0.
	ui.NewPlatformConfiguration(nil).UpdateViewMetrics(0, viewMetricsWithDPR(3.0))
	defer ui.NewPlatformConfiguration(nil).UpdateViewMetrics(0, ui.NewViewportMetrics())
	cases := paintingGradientsCases(t, tester)
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
		if reflect.DeepEqual(g, want) { // exact; -0.0 is a string on both sides
			continue
		}
		gl, gok := g.([]any)
		wl, wok := want.([]any)
		if gok && wok && len(gl) == len(wl) {
			n := 0
			for i := range gl {
				if !reflect.DeepEqual(gl[i], wl[i]) {
					if n < 4 {
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

// PlainDecoration is the Dart side's decoration that lerps with nothing.
type PlainDecoration struct{ DecorationBase }

func NewPlainDecoration() *PlainDecoration {
	d := &PlainDecoration{}
	d.SetSelf(d)
	return d
}

func (d *PlainDecoration) CreateBoxPainter(onChanged func()) BoxPainter { return nil }
