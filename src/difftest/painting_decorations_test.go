package difftest

// Differential test of painting's box_decoration.dart and
// shape_decoration.dart against Flutter: the same cases as
// flutter/test/painting_decorations_test.dart, every value printed with
// Dart's toString (doubles bit for bit), compared exactly: lerps (Box,
// Shape, and across through Decoration.lerp), scale, padding, isComplex,
// toString/toStringDeep, fromBoxDecoration, getClipPath (path signature as
// in the borders difftest), hitTest grids, and the canvas calls of
// createBoxPainter().paint, printed when made (painted three times with
// one painter, to exercise its caches).

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	p "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// deShaderID numbers shaders in the order a painter first uses them (the
// Dart side's shaderId).
func deShaderID(sh ui.Shader, seen *[]ui.Shader) string {
	if sh == nil {
		return "noshader"
	}
	for i, x := range *seen {
		if x == sh {
			return fmt.Sprintf("shader#%d", i)
		}
	}
	*seen = append(*seen, sh)
	return fmt.Sprintf("shader#%d", len(*seen)-1)
}

func dePaintF(q *ui.Paint, seen *[]ui.Shader) string {
	mf := "null"
	if m := q.MaskFilter(); m != nil {
		mf = m.String()
	}
	shader := deShaderID(q.Shader(), seen)
	return bdCol(q.Color()) + "/" + bd(q.StrokeWidth()) + "/" + q.Style().String() + "/" + q.BlendMode().String() + "/" +
		mf + "/" + shader + "/" + q.FilterQuality().String() + "/" + fmt.Sprint(q.InvertColors()) + "/" +
		fmt.Sprint(q.IsAntiAlias())
}

// deRec is the Dart side's Rec: every canvas call is printed when it is
// made (positional arguments only, as Dart's noSuchMethod Invocation).
type deRec struct {
	*ft.TestRecordingCanvas
	area    ui.Rect
	log     []string
	shaders *[]ui.Shader
}

func newDeRec(area ui.Rect, shaders *[]ui.Shader) *deRec {
	return &deRec{ft.NewTestRecordingCanvas(), area, nil, shaders}
}

func (r *deRec) arg(a any) string {
	switch x := a.(type) {
	case ui.Rect:
		return bdRect(x)
	case *ui.Rect:
		if x == nil {
			return "null"
		}
		return bdRect(*x)
	case ui.RRect:
		return bdRRect(x)
	case ui.RSuperellipse:
		return bdRSE(x)
	case *ui.Path:
		return bdPathSig(x, r.area, 7)
	case *ui.Paint:
		return dePaintF(x, r.shaders)
	case ui.Offset:
		return bdOff(x)
	case float64:
		return bd(x)
	case *ui.Image:
		return fmt.Sprintf("%dx%d", x.Width, x.Height)
	}
	return fmt.Sprint(a)
}

func (r *deRec) add(name string, args ...any) {
	s := make([]string, len(args))
	for i, a := range args {
		s[i] = r.arg(a)
	}
	r.log = append(r.log, name+"("+strings.Join(s, " ; ")+")")
}

func (r *deRec) Save()                                  { r.add("save") }
func (r *deRec) Restore()                               { r.add("restore") }
func (r *deRec) SaveLayer(bounds *ui.Rect, q *ui.Paint) { r.add("saveLayer", bounds, q) }
func (r *deRec) Translate(dx, dy float64)               { r.add("translate", dx, dy) }
func (r *deRec) Scale(sx float64, sy ...float64) {
	args := []any{sx}
	for _, y := range sy {
		args = append(args, y)
	}
	r.add("scale", args...)
}
func (r *deRec) ClipRect(rect ui.Rect, _ ...ui.CanvasClipRectOptions) { r.add("clipRect", rect) }
func (r *deRec) ClipRRect(rr ui.RRect, _ ...ui.CanvasClipOptions)     { r.add("clipRRect", rr) }
func (r *deRec) ClipPath(path *ui.Path, _ ...ui.CanvasClipOptions)    { r.add("clipPath", path) }
func (r *deRec) DrawPaint(q *ui.Paint)                                { r.add("drawPaint", q) }
func (r *deRec) DrawLine(a, b ui.Offset, q *ui.Paint)                 { r.add("drawLine", a, b, q) }
func (r *deRec) DrawRect(rect ui.Rect, q *ui.Paint)                   { r.add("drawRect", rect, q) }
func (r *deRec) DrawRRect(rr ui.RRect, q *ui.Paint)                   { r.add("drawRRect", rr, q) }
func (r *deRec) DrawDRRect(o, i ui.RRect, q *ui.Paint)                { r.add("drawDRRect", o, i, q) }
func (r *deRec) DrawOval(rect ui.Rect, q *ui.Paint)                   { r.add("drawOval", rect, q) }
func (r *deRec) DrawPath(path *ui.Path, q *ui.Paint)                  { r.add("drawPath", path, q) }
func (r *deRec) DrawRSuperellipse(x ui.RSuperellipse, q *ui.Paint)    { r.add("drawRSuperellipse", x, q) }
func (r *deRec) DrawCircle(c ui.Offset, radius float64, q *ui.Paint) {
	r.add("drawCircle", c, radius, q)
}
func (r *deRec) DrawImage(img *ui.Image, o ui.Offset, q *ui.Paint) { r.add("drawImage", img, o, q) }
func (r *deRec) DrawImageRect(img *ui.Image, src, dst ui.Rect, q *ui.Paint) {
	r.add("drawImageRect", img, src, dst, q)
}
func (r *deRec) DrawImageNine(img *ui.Image, center, dst ui.Rect, q *ui.Paint) {
	r.add("drawImageNine", img, center, dst, q)
}

// result is the log, with any call this recorder doesn't print flagged.
func (r *deRec) result() string {
	for _, inv := range r.Invocations {
		r.log = append(r.log, "UNLOGGED "+inv.Invocation.MemberName)
	}
	return strings.Join(r.log, "\n")
}

func deErr(e any) string {
	if _, ok := e.(dart.AssertionFailure); ok {
		return "assert"
	}
	if _, ok := e.(*foundation.FlutterError); ok {
		return "assert"
	}
	if _, ok := e.(dart.TypeError); ok {
		return "throws _TypeError"
	}
	return fmt.Sprintf("throws %T: %v", e, e)
}

func deTry(f func() string) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = deErr(r)
		}
	}()
	return f()
}

// deImg is the Dart side's Img: a provider of an already decoded image.
type deImg struct {
	p.ImageProviderBase
	image *ui.Image
	name  string
}

func newDeImg(image *ui.Image, name string) *deImg {
	x := &deImg{image: image, name: name}
	x.SetSelf(x)
	return x
}

func (x *deImg) ObtainKey(p.ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](x)
}

func (x *deImg) LoadImage(any, p.ImageDecoderCallback) p.ImageStreamCompleter {
	return p.NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(p.NewImageInfo(x.image.Clone())))
}

func (x *deImg) String() string { return "Img(" + x.name + ")" }

func deBS(color int, width float64, strokeAlign ...float64) p.BorderSide {
	a := p.BorderSideStrokeAlignInside
	if len(strokeAlign) > 0 {
		a = strokeAlign[0]
	}
	return bdBS(color, width, p.BorderStyleSolid, a)
}

func deCol(c int) *ui.Color { x := ui.NewColor(c); return &x }

func deColors(cs ...int) []ui.Color {
	out := make([]ui.Color, len(cs))
	for i, c := range cs {
		out[i] = ui.NewColor(c)
	}
	return out
}

func deShadow(color int, dx, dy, blur, spread float64, style ui.BlurStyle) p.BoxShadow {
	return p.NewBoxShadow(p.BoxShadowOptions{Color: deCol(color), Offset: ui.NewOffset(dx, dy), BlurRadius: blur,
		SpreadRadius: spread, BlurStyle: style})
}

type deFixture struct {
	lin, rad, swp      p.Gradient
	shadows1, shadows2 []p.BoxShadow
}

func newDeFixture() deFixture {
	return deFixture{
		lin: p.NewLinearGradient(p.LinearGradientOptions{Begin: p.AlignmentDirectionalTopStart, End: p.AlignmentBottomRight,
			Colors: deColors(0xFFFF0000, 0xFF00FF00, 0x800000FF), Stops: []float64{0, 0.4, 1}}),
		rad: p.NewRadialGradient(p.RadialGradientOptions{Center: p.NewAlignment(0.2, -0.3), Radius: ptr(0.7),
			Colors: deColors(0xFFFFFF00, 0xFF8000FF)}),
		swp: p.NewSweepGradient(p.SweepGradientOptions{Colors: deColors(0xFF00FFFF, 0xFFFF00FF, 0xFF00FFFF),
			Transform: p.NewGradientRotation(0.5)}),
		shadows1: []p.BoxShadow{
			deShadow(0x80000000, 2, 3, 4, 1, ui.BlurStyleNormal),
			deShadow(0x40FF0000, -1, 5, 10, -2, ui.BlurStyleOuter),
		},
		shadows2: []p.BoxShadow{
			deShadow(0xFF123456, 0, 1, 2.5, 0, ui.BlurStyleSolid),
			deShadow(0x33000000, 0, 0, 0, 3, ui.BlurStyleInner),
			deShadow(0x99009900, 4, -4, 6, 0, ui.BlurStyleOuter),
		},
	}
}

func deBoxes(f deFixture, img *ui.Image) []*p.BoxDecoration {
	nb := p.NewBoxDecoration
	type o = p.BoxDecorationOptions
	circ := func(r float64) ui.Radius { return ui.NewRadiusCircular(r) }
	bdir := func(s, e, t p.BorderSide) p.BoxBorder {
		return p.NewBorderDirectional(p.BorderDirectionalOptions{Start: &s, End: &e, Top: &t})
	}
	top, left := deBS(0xFF0000FF, 5), deBS(0xFF00FF00, 3)
	s4 := deBS(0xFF000000, 4)
	top2 := deBS(0xFF000000, 2)
	cover, repeatX := p.BoxFitCover, p.ImageRepeatRepeatX
	return []*p.BoxDecoration{
		nb(),
		nb(o{Color: deCol(0xFF3366CC)}),
		nb(o{Color: deCol(0xFF22AA44), Border: p.NewBorderAll(p.BorderSideOptions{Width: ptr(3.0), Color: deCol(0xFF2196F3)}),
			BorderRadius: p.NewBorderRadiusAll(circ(12))}),
		nb(o{Color: deCol(0xFFAA2244), Shape: p.BoxShapeCircle,
			Border: p.NewBorderAll(p.BorderSideOptions{Width: ptr(4.0), Color: deCol(0x802196F3)})}),
		nb(o{Gradient: f.lin, BorderRadius: p.BorderRadiusDirectional{TopStart: circ(20), BottomEnd: ui.NewRadiusElliptical(8, 30)}}),
		nb(o{Color: deCol(0xFF444444), BoxShadow: f.shadows1}),
		nb(o{Color: deCol(0xFF119988), Border: bdir(deBS(0xFF000000, 4), deBS(0x80000000, 2),
			deBS(0xFFFF0000, 6, p.BorderSideStrokeAlignCenter))}),
		nb(o{Color: deCol(0xFFFFCC00), BackgroundBlendMode: ptr(ui.BlendModeMultiply),
			Border: p.NewBorder(p.BorderOptions{Top: &top, Left: &left})}),
		nb(o{Gradient: f.rad, Shape: p.BoxShapeCircle, BoxShadow: f.shadows2}),
		nb(o{Color: deCol(0xFF8800FF), Border: p.NewBorderAll(p.BorderSideOptions{Width: ptr(6.0), Color: deCol(0x80FFFFFF)}),
			BorderRadius: p.NewBorderRadiusAll(circ(9))}),
		nb(o{Color: deCol(0xFF00AAFF), Border: p.NewBorderAll(p.BorderSideOptions{Width: ptr(5.0),
			StrokeAlign: ptr(p.BorderSideStrokeAlignOutside)})}),
		nb(o{Color: deCol(0xFF654321), BorderRadius: p.BorderRadiusZero}),
		nb(o{Color: deCol(0xFF123456), BorderRadius: p.BorderRadiusDirectionalZero}),
		nb(o{Gradient: f.swp, BorderRadius: p.BorderRadius{TopLeft: ui.NewRadiusElliptical(30, 10), BottomRight: circ(15)},
			BoxShadow: f.shadows1}),
		nb(o{Color: deCol(0xFF00FF00), Shape: p.BoxShapeCircle,
			Image: p.NewDecorationImage(p.DecorationImageOptions{Image: newDeImg(img, "a"), Fit: &cover})}),
		nb(o{Image: p.NewDecorationImage(p.DecorationImageOptions{Image: newDeImg(img, "b"), Alignment: p.AlignmentDirectionalBottomEnd}),
			BorderRadius: p.NewBorderRadiusAll(circ(10)), Border: p.NewBorderAll(p.BorderSideOptions{Width: ptr(2.0)})}),
		nb(o{Image: p.NewDecorationImage(p.DecorationImageOptions{Image: newDeImg(img, "c"), Repeat: &repeatX, Scale: ptr(2.0)})}),
		nb(o{Color: deCol(0xFF444444), BoxShadow: []p.BoxShadow{f.shadows2[0], f.shadows2[1]}}),
		nb(o{Color: deCol(0xFF00FF88), Border: p.NewBorderDirectional(p.BorderDirectionalOptions{Top: &s4, Start: &s4, End: &s4, Bottom: &s4})}),
		nb(o{Color: deCol(0xFF884488), Shape: p.BoxShapeCircle, Border: p.NewBorder(p.BorderOptions{Top: &top2})}),
	}
}

func deShapes(f deFixture, img *ui.Image) []*p.ShapeDecoration {
	ns := p.NewShapeDecoration
	type o = p.ShapeDecorationOptions
	circ := func(r float64) ui.Radius { return ui.NewRadiusCircular(r) }
	side := func(s p.BorderSide) *p.BorderSide { return &s }
	fill := p.BoxFitFill
	return []*p.ShapeDecoration{
		ns(o{Shape: p.NewBorder()}),
		ns(o{Color: deCol(0xFFAA2244), Shape: p.NewCircleBorder(p.CircleBorderOptions{Side: side(deBS(0xFF2196F3, 3))})}),
		ns(o{Gradient: f.lin, Shape: p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: side(deBS(0xFF000000, 2)),
			BorderRadius: p.NewBorderRadiusAll(circ(12))})}),
		ns(o{Color: deCol(0xFF22AA44), Shadows: f.shadows1, Shape: p.NewStadiumBorder(p.StadiumBorderOptions{Side: side(deBS(0x80FF0000, 4))})}),
		ns(o{Color: deCol(0xFF4422AA), Shadows: f.shadows2, Shape: p.NewStarBorder(p.StarBorderOptions{
			Side: side(deBS(0xFF000000, 1.5)), Points: ptr(6.0), InnerRadiusRatio: ptr(0.5), PointRounding: 0.2, Rotation: 15})}),
		ns(o{Color: deCol(0xFF884400), Shadows: f.shadows1, Shape: p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{
			Side: side(deBS(0xFFFFFFFF, 3)), BorderRadius: p.NewBorderRadiusAll(circ(18))})}),
		ns(o{Gradient: f.rad, Shadows: f.shadows2, Shape: p.NewBeveledRectangleBorder(p.RRectBorderOptions{
			BorderRadius: p.BorderRadiusDirectional{TopStart: circ(14)}})}),
		ns(o{Color: deCol(0xFF00FFAA), Shadows: f.shadows1, Shape: p.NewBorderAll(p.BorderSideOptions{Width: ptr(4.0), Color: deCol(0xFF333333)})}),
		ns(o{Color: deCol(0xFF0088FF), Shape: p.NewOvalBorder(p.OvalBorderOptions{Side: side(deBS(0x802196F3, 5, p.BorderSideStrokeAlignCenter))})}),
		ns(o{Color: deCol(0xFFFF8800), Shape: p.NewCircleBorder(p.CircleBorderOptions{Side: side(deBS(0xFF000000, 2))}).
			Add(p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: side(deBS(0xFF0000FF, 3))}))}),
		ns(o{Color: deCol(0xFF777777), Shape: p.NewLinearBorderBottom(p.LinearBorderEdgeConstructorOptions{Side: side(deBS(0xFF000000, 3))})}),
		ns(o{Gradient: f.swp, Shape: p.NewContinuousRectangleBorder(p.RRectBorderOptions{Side: side(deBS(0xFF00FF00, 2)),
			BorderRadius: p.NewBorderRadiusAll(circ(30))})}),
		ns(o{Image: p.NewDecorationImage(p.DecorationImageOptions{Image: newDeImg(img, "s"), Fit: &fill}),
			Shape: p.NewCircleBorder(p.CircleBorderOptions{Side: side(deBS(0xFF000000, 2))})}),
		ns(o{Color: deCol(0xFFCCCCCC), Image: p.NewDecorationImage(p.DecorationImageOptions{Image: newDeImg(img, "t"),
			Alignment: p.AlignmentDirectionalCenterStart}),
			Shape: p.NewRoundedRectangleBorder(p.RRectBorderOptions{BorderRadius: p.BorderRadiusDirectional{TopEnd: circ(25)}})}),
		ns(o{Image: p.NewDecorationImage(p.DecorationImageOptions{Image: newDeImg(img, "s2"), Fit: &fill}),
			Shape: p.NewCircleBorder(p.CircleBorderOptions{Side: side(deBS(0xFF000000, 2))})}),
		ns(o{Color: deCol(0xFF336699), Shadows: f.shadows1, Shape: p.NewRoundedRectangleBorder(p.RRectBorderOptions{
			BorderRadius: p.BorderRadiusDirectional{TopStart: circ(20)}})}),
	}
}

var deTs = []float64{-0.5, 0, 0.25, 0.5, 0.75, 1, 1.5}

type deCall struct {
	o ui.Offset
	s ui.Size
}

func tdKey(td *ui.TextDirection) string {
	if td == nil {
		return "null"
	}
	return td.String()
}

func dePaintLog(dec p.Decoration, calls []deCall, td *ui.TextDirection, tdPerCall ...*ui.TextDirection) string {
	painter := dec.CreateBoxPainter(func() {})
	var out []string
	var shaders []ui.Shader
	for i, c := range calls {
		rec := newDeRec(c.o.And(c.s), &shaders)
		s := c.s
		callTd := td
		if tdPerCall != nil {
			callTd = tdPerCall[i]
		}
		out = append(out, deTry(func() string {
			painter.Paint(rec, c.o, p.ImageConfiguration{Size: &s, TextDirection: callTd})
			return rec.result()
		}))
	}
	painter.Dispose()
	return strings.Join(out, "\n--\n")
}

func deHitGrid(dec p.Decoration, size ui.Size, td *ui.TextDirection) string {
	return deTry(func() string {
		var b strings.Builder
		const n = 11
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				x := float64(size.Width*float64(i)) / float64(n-1)
				y := float64(size.Height*float64(j)) / float64(n-1)
				if dec.HitTest(size, ui.NewOffset(x, y), td) {
					b.WriteString("1")
				} else {
					b.WriteString("0")
				}
			}
		}
		return b.String()
	})
}

func deStr(x fmt.Stringer) string {
	if x == nil {
		return "null"
	}
	return x.String()
}

func deBoxStr(x *p.BoxDecoration) string {
	if x == nil {
		return "null"
	}
	return x.String()
}

func deShapeStr(x *p.ShapeDecoration) string {
	if x == nil {
		return "null"
	}
	return x.String()
}

func paintingDecorationsCases(img *ui.Image) map[string]string {
	out := map[string]string{}
	f := newDeFixture()
	bx := deBoxes(f, img)
	sx := deShapes(f, img)
	rects := []ui.Rect{ui.NewRectFromLTRB(10, 20, 110, 80), ui.NewRectFromLTWH(-5, 3.5, 40, 90)}
	sizes := []ui.Size{ui.NewSize(100, 60), ui.NewSize(40, 90)}
	calls := []deCall{{ui.NewOffset(5, 7), ui.NewSize(100, 60)}, {ui.NewOffset(5, 7), ui.NewSize(100, 60)},
		{ui.NewOffset(-3, 2.5), ui.NewSize(40, 90)}}
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	tds := []*ui.TextDirection{&ltr, &rtl, nil}

	type entry struct {
		key string
		dec p.Decoration
	}
	var all []entry
	byKey := map[string]p.Decoration{}
	for i, b := range bx {
		all = append(all, entry{fmt.Sprintf("box.%d", i), b})
	}
	for i, s := range sx {
		all = append(all, entry{fmt.Sprintf("shape.%d", i), s})
	}
	for _, e := range all {
		byKey[e.key] = e.dec
		k, dec := e.key, e.dec
		out[k+".str"] = deTry(func() string { return dec.String() })
		out[k+".prop"] = deTry(func() string {
			return foundation.NewDiagnosticsProperty[p.Decoration]("decoration", dec).ToStringDeep()
		})
		out[k+".deep"] = deTry(func() string { return dec.ToDiagnosticsNode().ToStringDeep() })
		out[k+".complex"] = fmt.Sprint(dec.IsComplex())
		for _, td := range []ui.TextDirection{ltr, rtl} {
			out[k+".padding."+td.String()] = deTry(func() string { return bdInsets(dec.Padding(), td) })
		}
		out[k+".valid"] = deTry(func() string { return fmt.Sprint(dec.DebugAssertIsValid()) })
		for r, rect := range rects {
			for _, td := range []ui.TextDirection{ltr, rtl} {
				out[fmt.Sprintf("%s.clip.%d.%s", k, r, td)] = deTry(func() string {
					return bdPathSig(dec.GetClipPath(rect, td), rect, 9)
				})
			}
		}
		for s, size := range sizes {
			for _, td := range tds {
				out[fmt.Sprintf("%s.hit.%d.%s", k, s, tdKey(td))] = deHitGrid(dec, size, td)
			}
		}
		for _, td := range tds {
			out[k+".paint."+tdKey(td)] = dePaintLog(dec, calls, td)
		}
		// One painter, the same rect, the text direction switching.
		out[k+".paint.switch"] = dePaintLog(dec, []deCall{calls[0], calls[0], calls[0]}, nil, &ltr, &rtl, &rtl)
		for _, t := range []float64{0, 0.3, 1} {
			out[k+".lerpNull."+bd(t)] = deTry(func() string { return deStr(p.DecorationLerp(dec, nil, t)) })
			out[k+".nullLerp."+bd(t)] = deTry(func() string { return deStr(p.DecorationLerp(nil, dec, t)) })
		}
	}
	p.DebugDisableShadows = true
	for _, k := range []string{"box.5", "box.8", "box.13", "shape.3", "shape.4", "shape.5", "shape.6", "shape.7"} {
		out[k+".paint.noShadows"] = dePaintLog(byKey[k], calls, &ltr)
	}
	p.DebugDisableShadows = false
	for _, e := range all {
		dec := e.dec
		out[e.key+".painterNoCallback"] = deTry(func() string {
			dec.CreateBoxPainter(nil)
			return "ok"
		})
	}

	for i, b := range bx {
		for _, fct := range []float64{-0.5, 0, 0.5, 2} {
			out[fmt.Sprintf("box.%d.scale.%s", i, bd(fct))] = deTry(func() string { return b.Scale(fct).String() })
		}
		out[fmt.Sprintf("box.%d.scale.paint", i)] = dePaintLog(b.Scale(0.5), calls, &ltr)
		out[fmt.Sprintf("box.%d.fromBox", i)] = deTry(func() string { return p.NewShapeDecorationFromBoxDecoration(b).String() })
		out[fmt.Sprintf("box.%d.copy", i)] = deTry(func() string {
			return b.CopyWith(p.BoxDecorationCopyWithOptions{Shape: ptr(p.BoxShapeCircle), Color: deCol(0xFF010203)}).String()
		})
		for _, t := range deTs {
			out[fmt.Sprintf("box.%d.lerp.null.%s", i, bd(t))] = deTry(func() string { return deBoxStr(p.BoxDecorationLerp(nil, b, t)) })
			out[fmt.Sprintf("box.%d.lerp.%s.null", i, bd(t))] = deTry(func() string { return deBoxStr(p.BoxDecorationLerp(b, nil, t)) })
		}
		for j, c := range bx {
			out[fmt.Sprintf("box.%d.%d.==", i, j)] = fmt.Sprint(b.Equal(c))
			for _, t := range deTs {
				out[fmt.Sprintf("box.%d.%d.lerp.%s", i, j, bd(t))] = deTry(func() string { return deBoxStr(p.BoxDecorationLerp(b, c, t)) })
			}
		}
	}
	for i, s := range sx {
		for _, t := range deTs {
			out[fmt.Sprintf("shape.%d.lerp.null.%s", i, bd(t))] = deTry(func() string { return deShapeStr(p.ShapeDecorationLerp(nil, s, t)) })
			out[fmt.Sprintf("shape.%d.lerp.%s.null", i, bd(t))] = deTry(func() string { return deShapeStr(p.ShapeDecorationLerp(s, nil, t)) })
		}
		for j, c := range sx {
			out[fmt.Sprintf("shape.%d.%d.==", i, j)] = fmt.Sprint(s.Equal(c))
			for _, t := range deTs {
				out[fmt.Sprintf("shape.%d.%d.lerp.%s", i, j, bd(t))] = deTry(func() string { return deShapeStr(p.ShapeDecorationLerp(s, c, t)) })
			}
		}
	}
	for i, b := range bx {
		for j, s := range sx {
			for _, t := range deTs {
				out[fmt.Sprintf("cross.%d.%d.%s", i, j, bd(t))] = deTry(func() string { return deStr(p.DecorationLerp(b, s, t)) })
				out[fmt.Sprintf("cross.%d.%d.%s", j, i, bd(t))] = deTry(func() string { return deStr(p.DecorationLerp(s, b, t)) })
			}
		}
	}
	for _, ij := range [][2]int{{2, 3}, {4, 13}, {5, 8}, {6, 7}, {9, 10}} {
		for _, t := range []float64{0.3, 0.7} {
			k := fmt.Sprintf("boxlerp.%d.%d.%s", ij[0], ij[1], bd(t))
			dec := p.BoxDecorationLerp(bx[ij[0]], bx[ij[1]], t)
			out[k+".paint"] = dePaintLog(dec, calls, &rtl)
			out[k+".hit"] = deHitGrid(dec, sizes[0], &rtl)
		}
	}
	for _, ij := range [][2]int{{1, 2}, {3, 4}, {5, 6}, {2, 9}, {7, 11}} {
		for _, t := range []float64{0.3, 0.7} {
			k := fmt.Sprintf("shapelerp.%d.%d.%s", ij[0], ij[1], bd(t))
			dec := p.ShapeDecorationLerp(sx[ij[0]], sx[ij[1]], t)
			out[k+".paint"] = dePaintLog(dec, calls, &ltr)
			out[k+".hit"] = deHitGrid(dec, sizes[1], &ltr)
			out[k+".clip"] = deTry(func() string { return bdPathSig(dec.GetClipPath(rects[0], ltr), rects[0], 9) })
		}
	}
	return out
}

func TestPaintingDecorationsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_decorations.json")
	if err != nil {
		t.Fatalf("no reference (%v); run the Flutter side: flutter test test/painting_decorations_test.dart", err)
	}
	var want map[string]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	tester := ft.New(t)
	// As the Dart side: the test binding disables shadows; paint them.
	defer func(old bool) { p.DebugDisableShadows = old }(p.DebugDisableShadows)
	p.DebugDisableShadows = false
	img := makeImage(tester, 16, 8)
	got := paintingDecorationsCases(img)
	p.ImageCacheInstance().Clear()
	p.ImageCacheInstance().ClearLiveImages()
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	bad := 0
	for _, k := range keys {
		g, ok := got[k]
		if !ok {
			t.Errorf("%s: missing on the Go side", k)
			bad++
		} else if g != want[k] {
			if bad < 25 {
				t.Errorf("%s:\n flutter %s\n      go %s", k, want[k], g)
			}
			bad++
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: not in the Flutter reference", k)
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d cases differ", bad, len(want))
	}
	t.Logf("%d cases", len(want))
}
