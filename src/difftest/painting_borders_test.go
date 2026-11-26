package difftest

// Differential test of painting's borders.dart, box_border.dart, the
// shape borders and notched_shapes.dart against Flutter: the same cases
// as flutter/test/painting_borders_test.dart, every value printed with
// Dart's toString (doubles bit for bit), compared exactly. Paths are
// compared by bounds, metrics (lengths and tangents at fixed fractions)
// and contains() over a grid; paint() through a recording canvas that
// prints each call when it is made (the borders reuse and mutate their
// Path and Paint objects between calls, as in Dart).

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
	"github.com/jonathanperis/flutter-go/src/dart"
)

func bd(v float64) string { return dart.DoubleToString(v) }

func bdCol(c ui.Color) string { return bd(c.A) + "," + bd(c.R) + "," + bd(c.G) + "," + bd(c.B) }

func bdSideF(s p.BorderSide) string {
	return bdCol(s.Color) + "|" + bd(s.Width) + "|" + s.Style.String() + "|" + bd(s.StrokeAlign)
}

func bdRect(r ui.Rect) string {
	return bd(r.Left) + "," + bd(r.Top) + "," + bd(r.Right) + "," + bd(r.Bottom)
}

func bdCorners(l, t, r, b, a, bb, c, dd, e, f, g, h float64) string {
	return strings.Join([]string{bd(l), bd(t), bd(r), bd(b), bd(a), bd(bb), bd(c), bd(dd), bd(e), bd(f), bd(g), bd(h)}, ",")
}

func bdRRect(r ui.RRect) string {
	return bdCorners(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY,
		r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
}

func bdRSE(r ui.RSuperellipse) string {
	return bdCorners(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY,
		r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
}

func bdOff(o ui.Offset) string { return bd(o.Dx) + "," + bd(o.Dy) }

func bdInsets(g p.EdgeInsetsGeometry, td ui.TextDirection) string {
	e := g.Resolve(&td)
	return bd(e.Left) + "," + bd(e.Top) + "," + bd(e.Right) + "," + bd(e.Bottom)
}

func bdPathSig(path *ui.Path, area ui.Rect, n int) string {
	var b strings.Builder
	b.WriteString("b=" + bdRect(path.GetBounds()) + ";m=")
	for m := range path.ComputeMetrics().All() {
		b.WriteString("[" + bd(m.Length) + "," + fmt.Sprint(m.IsClosed))
		for _, f := range []float64{0, 0.25, 0.5, 0.75, 1} {
			if t := m.GetTangentForOffset(m.Length * f); t == nil {
				b.WriteString(",null")
			} else {
				b.WriteString("," + bdOff(t.Position) + "," + bdOff(t.Vector))
			}
		}
		b.WriteString("]")
	}
	b.WriteString(";c=")
	a := area.Inflate(3)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x := a.Left + float64(a.Width()*float64(i))/float64(n-1)
			y := a.Top + float64(a.Height()*float64(j))/float64(n-1)
			if path.Contains(ui.NewOffset(x, y)) {
				b.WriteString("1")
			} else {
				b.WriteString("0")
			}
		}
	}
	return b.String()
}

func bdPaintF(q *ui.Paint) string {
	return bdCol(q.Color()) + "/" + bd(q.StrokeWidth()) + "/" + q.Style().String()
}

// bdRec is the Dart side's Rec: a canvas whose drawing calls are printed
// when they are made.
type bdRec struct {
	*ft.TestRecordingCanvas
	area ui.Rect
	log  []string
}

func newBdRec(area ui.Rect) *bdRec { return &bdRec{ft.NewTestRecordingCanvas(), area, nil} }

func (r *bdRec) arg(a any) string {
	switch x := a.(type) {
	case ui.Rect:
		return bdRect(x)
	case ui.RRect:
		return bdRRect(x)
	case ui.RSuperellipse:
		return bdRSE(x)
	case *ui.Path:
		return bdPathSig(x, r.area, 7)
	case *ui.Paint:
		return bdPaintF(x)
	case ui.Offset:
		return bdOff(x)
	case float64:
		return bd(x)
	}
	return fmt.Sprint(a)
}

func (r *bdRec) add(name string, args ...any) {
	s := make([]string, len(args))
	for i, a := range args {
		s[i] = r.arg(a)
	}
	r.log = append(r.log, name+"("+strings.Join(s, " ; ")+")")
}

func (r *bdRec) DrawRect(rect ui.Rect, q *ui.Paint)                { r.add("drawRect", rect, q) }
func (r *bdRec) DrawRRect(rr ui.RRect, q *ui.Paint)                { r.add("drawRRect", rr, q) }
func (r *bdRec) DrawDRRect(o, i ui.RRect, q *ui.Paint)             { r.add("drawDRRect", o, i, q) }
func (r *bdRec) DrawOval(rect ui.Rect, q *ui.Paint)                { r.add("drawOval", rect, q) }
func (r *bdRec) DrawPath(path *ui.Path, q *ui.Paint)               { r.add("drawPath", path, q) }
func (r *bdRec) DrawRSuperellipse(x ui.RSuperellipse, q *ui.Paint) { r.add("drawRSuperellipse", x, q) }
func (r *bdRec) DrawCircle(c ui.Offset, radius float64, q *ui.Paint) {
	r.add("drawCircle", c, radius, q)
}

func (r *bdRec) String() string { return strings.Join(r.log, "\n") }

func bdErr(e any) string {
	if _, ok := e.(dart.AssertionFailure); ok {
		return "assert"
	}
	if _, ok := e.(*foundation.FlutterError); ok {
		return "assert"
	}
	return fmt.Sprintf("throws %T: %v", e, e)
}

func bdTry(f func() string) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = bdErr(r)
		}
	}()
	return f()
}

func bdBS(color int, width float64, style p.BorderStyle, strokeAlign float64) p.BorderSide {
	c := ui.NewColor(color)
	return p.NewBorderSide(p.BorderSideOptions{Color: &c, Width: &width, Style: &style, StrokeAlign: &strokeAlign})
}

func bdSides() []p.BorderSide {
	solid, none := p.BorderStyleSolid, p.BorderStyleNone
	in, ce, out := p.BorderSideStrokeAlignInside, p.BorderSideStrokeAlignCenter, p.BorderSideStrokeAlignOutside
	return []p.BorderSide{
		p.NewBorderSide(),
		p.BorderSideNone,
		bdBS(0xFF2196F3, 3, solid, in),
		bdBS(0x802196F3, 0.5, solid, ce),
		bdBS(0xFFE91E63, 7, solid, out),
		bdBS(0xFF4CAF50, 2, none, in),
		bdBS(0xFFFF9800, 0, solid, in),
		bdBS(0x33123456, 12.75, solid, -0.25),
	}
}

var bdTs = []float64{-0.5, 0, 0.1, 0.25, 0.5, 0.75, 1, 1.5}

func bdCirc(r float64) ui.Radius { return ui.NewRadiusCircular(r) }

func bdShapes() []p.ShapeBorder {
	s := bdSides()
	return []p.ShapeBorder{
		p.NewBorderAll(p.BorderSideOptions{Width: ptr(3.0), Color: ptr(ui.NewColor(0xFF2196F3))}),
		p.NewBorder(p.BorderOptions{Top: &s[2], Right: &s[4], Bottom: &s[6], Left: &s[3]}),
		p.NewBorder(p.BorderOptions{Left: &s[2]}),
		p.NewBorderSymmetric(p.BorderSymmetricOptions{Vertical: &s[3], Horizontal: &s[2]}),
		p.NewBorderDirectional(p.BorderDirectionalOptions{Top: &s[2], Start: &s[4], End: &s[3], Bottom: &s[7]}),
		p.NewBorderDirectional(p.BorderDirectionalOptions{Start: &s[2]}),
		p.NewCircleBorder(p.CircleBorderOptions{Side: &s[2]}),
		p.NewCircleBorder(p.CircleBorderOptions{Side: &s[4], Eccentricity: 0.5}),
		p.NewOvalBorder(p.OvalBorderOptions{Side: &s[3]}),
		p.NewOvalBorder(p.OvalBorderOptions{Side: &s[2], Eccentricity: ptr(0.25)}),
		p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &s[2], BorderRadius: p.NewBorderRadiusAll(bdCirc(12))}),
		p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &s[3],
			BorderRadius: p.BorderRadiusDirectional{TopStart: bdCirc(20), BottomEnd: ui.NewRadiusElliptical(8, 30)}}),
		p.NewRoundedRectangleBorder(),
		p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{Side: &s[2], BorderRadius: p.NewBorderRadiusAll(bdCirc(18))}),
		p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{Side: &s[4]}),
		p.NewStadiumBorder(p.StadiumBorderOptions{Side: &s[2]}),
		p.NewStadiumBorder(p.StadiumBorderOptions{Side: &s[7]}),
		p.NewBeveledRectangleBorder(p.RRectBorderOptions{Side: &s[2], BorderRadius: p.NewBorderRadiusAll(bdCirc(10))}),
		p.NewBeveledRectangleBorder(p.RRectBorderOptions{Side: &s[4],
			BorderRadius: p.BorderRadiusDirectional{TopStart: bdCirc(30), BottomStart: bdCirc(5)}}),
		p.NewContinuousRectangleBorder(p.RRectBorderOptions{Side: &s[2], BorderRadius: p.NewBorderRadiusAll(bdCirc(24))}),
		p.NewContinuousRectangleBorder(p.RRectBorderOptions{Side: &s[3], BorderRadius: p.BorderRadius{TopLeft: ui.NewRadiusElliptical(60, 20)}}),
		p.NewStarBorder(p.StarBorderOptions{Side: &s[2]}),
		p.NewStarBorder(p.StarBorderOptions{Side: &s[3], Points: ptr(7.5), InnerRadiusRatio: ptr(0.6), PointRounding: 0.3,
			ValleyRounding: 0.2, Rotation: 27, Squash: 0.5}),
		p.NewStarBorderPolygon(p.StarBorderPolygonOptions{Side: &s[4], Sides: ptr(6.0), PointRounding: 0.4, Rotation: 90, Squash: 1}),
		p.NewStarBorderPolygon(p.StarBorderPolygonOptions{Side: &s[2], Sides: ptr(3.5)}),
		p.NewLinearBorderStart(p.LinearBorderEdgeConstructorOptions{Side: &s[2], Alignment: -0.5, Size: ptr(0.75)}),
		p.NewLinearBorder(p.LinearBorderOptions{Side: &s[4], Top: ptr(p.NewLinearBorderEdge(p.LinearBorderEdgeOptions{Size: ptr(0.5), Alignment: 1})),
			Bottom: ptr(p.NewLinearBorderEdge())}),
		p.LinearBorderNone,
	}
}

var bdRects = []ui.Rect{
	ui.NewRectFromLTRB(10, 20, 110, 70),
	ui.NewRectFromLTRB(5.5, 3.25, 45.75, 120.5),
	ui.NewRectFromLTWH(-20, -20, 40, 40),
}

// bdTypeName is the border's Dart runtimeType.
func bdTypeName(x p.ShapeBorder) string {
	s := fmt.Sprintf("%T", x)
	switch s {
	case "*painting.compoundBorder":
		return "_CompoundBorder"
	case "painting.stadiumToCircleBorder":
		return "_StadiumToCircleBorder"
	case "painting.stadiumToRoundedRectangleBorder":
		return "_StadiumToRoundedRectangleBorder"
	case "painting.shapeToCircleBorder":
		if strings.HasPrefix(x.String(), "RoundedSuperellipseBorder") {
			return "_RoundedSuperellipseToCircleBorder"
		}
		return "_RoundedRectangleToCircleBorder"
	}
	return strings.TrimPrefix(s, "painting.")
}

func bdDesc(x p.ShapeBorder) string {
	if x == nil {
		return "null"
	}
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	return bdTypeName(x) + "|" + x.String() + "|" + bdTry(func() string { return bdInsets(x.Dimensions(), ltr) }) + "|" +
		bdTry(func() string { return bdInsets(x.Dimensions(), rtl) }) + "|" + fmt.Sprint(x.PreferPaintInterior()) + "|" +
		bdTry(func() string { return bdPathSig(x.GetOuterPath(bdRects[0], &ltr), bdRects[0], 7) }) + "|" +
		bdTry(func() string { return bdPathSig(x.GetInnerPath(bdRects[0], &rtl), bdRects[0], 7) })
}

func bdPaint(area ui.Rect, f func(c ui.Canvas)) string {
	rec := newBdRec(area)
	f(rec)
	return rec.String()
}

func paintingBordersCases() map[string]string {
	out := map[string]string{}
	s := bdSides()
	for i, a := range s {
		out[fmt.Sprintf("side.%d", i)] = a.String() + "|" + bdSideF(a) + "|" + bd(a.StrokeInset()) + "|" +
			bd(a.StrokeOutset()) + "|" + bd(a.StrokeOffset()) + "|" + bdPaintF(a.ToPaint())
		for _, t := range []float64{-1, 0, 0.5, 2} {
			out[fmt.Sprintf("side.%d.scale.%s", i, bd(t))] = bdTry(func() string { return bdSideF(a.Scale(t)) })
		}
		for j, b := range s {
			k := fmt.Sprintf("side.%d.%d", i, j)
			out[k+".canMerge"] = fmt.Sprint(p.BorderSideCanMerge(a, b))
			out[k+".merge"] = bdTry(func() string { return bdSideF(p.BorderSideMerge(a, b)) })
			out[k+".=="] = fmt.Sprint(a == b)
			for _, t := range bdTs {
				out[k+".lerp."+bd(t)] = bdTry(func() string { return bdSideF(p.BorderSideLerp(a, b, t)) })
			}
		}
	}
	out["side.copyWith"] = bdSideF(s[2].CopyWith(p.BorderSideCopyWithOptions{Color: ptr(ui.NewColor(0xFF00FF00)), Width: ptr(4.0),
		Style: ptr(p.BorderStyleNone), StrokeAlign: ptr(0.5)}))
	out["side.copyWith.neg"] = bdTry(func() string { return bdSideF(s[2].CopyWith(p.BorderSideCopyWithOptions{Width: ptr(-1.0)})) })

	b := bdShapes()
	tds := []ui.TextDirection{ui.TextDirectionRtl, ui.TextDirectionLtr} // TextDirection.values
	for i, x := range b {
		out[fmt.Sprintf("shape.%d", i)] = bdDesc(x)
		for _, t := range []float64{0, 0.5, 2} {
			out[fmt.Sprintf("shape.%d.scale.%s", i, bd(t))] = bdTry(func() string { return bdDesc(x.Scale(t)) })
		}
		for _, t := range []float64{0, 0.3, 1} {
			out[fmt.Sprintf("shape.%d.lerpNull.%s", i, bd(t))] = bdTry(func() string { return bdDesc(p.ShapeBorderLerp(x, nil, t)) })
			out[fmt.Sprintf("shape.%d.nullLerp.%s", i, bd(t))] = bdTry(func() string { return bdDesc(p.ShapeBorderLerp(nil, x, t)) })
		}
		for r, rect := range bdRects {
			for _, td := range tds {
				k := fmt.Sprintf("shape.%d.r%d.%s", i, r, td.Name())
				out[k+".outer"] = bdTry(func() string { return bdPathSig(x.GetOuterPath(rect, &td), rect, 9) })
				out[k+".inner"] = bdTry(func() string { return bdPathSig(x.GetInnerPath(rect, &td), rect, 9) })
				out[k+".hit"] = bdTry(func() string {
					var buf strings.Builder
					a := rect.Inflate(2)
					for pp := 0; pp < 9; pp++ {
						for q := 0; q < 9; q++ {
							pos := ui.NewOffset(a.Left+float64(a.Width()*float64(pp))/8, a.Top+float64(a.Height()*float64(q))/8)
							if x.HitTest(rect, pos, &td) {
								buf.WriteString("1")
							} else {
								buf.WriteString("0")
							}
						}
					}
					return buf.String()
				})
				out[k+".paint"] = bdTry(func() string { return bdPaint(rect, func(c ui.Canvas) { x.Paint(c, rect, &td) }) })
				out[k+".paintInterior"] = bdTry(func() string {
					return bdPaint(rect, func(c ui.Canvas) {
						q := ui.NewPaint()
						q.SetColor(ui.NewColor(0xFF123456))
						x.PaintInterior(c, rect, q, &td)
					})
				})
			}
		}
		for j, y := range b {
			k := fmt.Sprintf("shape.%d.%d", i, j)
			out[k+".add"] = bdTry(func() string { return bdDesc(x.Add(y)) })
			for _, t := range []float64{-0.25, 0, 0.2, 0.5, 0.9, 1, 1.25} {
				out[k+".lerp."+bd(t)] = bdTry(func() string { return bdDesc(p.ShapeBorderLerp(x, y, t)) })
			}
			xo, ok1 := x.(p.OutlinedBorder)
			yo, ok2 := y.(p.OutlinedBorder)
			if ok1 && ok2 {
				out[k+".olerp.0.4"] = bdTry(func() string {
					r := p.OutlinedBorderLerp(xo, yo, 0.4)
					if r == nil {
						return bdDesc(nil)
					}
					return bdDesc(r)
				})
			}
			xb, ok1 := x.(p.BoxBorder)
			yb, ok2 := y.(p.BoxBorder)
			if ok1 && ok2 {
				for _, t := range []float64{-0.5, 0.25, 0.5, 0.75, 1.5} {
					out[k+".boxlerp."+bd(t)] = bdTry(func() string {
						r := p.BoxBorderLerp(xb, yb, t)
						if r == nil {
							return bdDesc(nil)
						}
						return bdDesc(r)
					})
				}
			}
		}
	}

	rr := p.NewRoundedRectangleBorder(p.RRectBorderOptions{Side: &s[2], BorderRadius: p.NewBorderRadiusAll(bdCirc(12))})
	rs := p.NewRoundedSuperellipseBorder(p.RRectBorderOptions{Side: &s[3], BorderRadius: p.NewBorderRadiusAll(bdCirc(18))})
	st := p.NewStadiumBorder(p.StadiumBorderOptions{Side: &s[4]})
	ci := p.NewCircleBorder(p.CircleBorderOptions{Side: &s[7], Eccentricity: 0.4})
	ov := p.NewOvalBorder(p.OvalBorderOptions{Side: &s[2]})
	type named struct {
		name string
		b    p.ShapeBorder
	}
	inter := []named{
		{"rr-ci", p.ShapeBorderLerp(rr, ci, 0.3)},
		{"ci-rr", p.ShapeBorderLerp(ci, rr, 0.6)},
		{"rs-ov", p.ShapeBorderLerp(rs, ov, 0.45)},
		{"st-ci", p.ShapeBorderLerp(st, ci, 0.35)},
		{"st-rr", p.ShapeBorderLerp(st, rr, 0.65)},
		{"rr-st", p.ShapeBorderLerp(rr, st, 0.2)},
		{"compound", b[0].Add(b[2]).Add(b[6])},
	}
	others := []p.ShapeBorder{rr, rs, st, ci, ov}
	for _, e := range inter {
		others = append(others, e.b)
	}
	for _, e := range inter {
		out["inter."+e.name] = bdDesc(e.b)
		out["inter."+e.name+".copyWithSide"] = bdTry(func() string {
			o, ok := e.b.(p.OutlinedBorder)
			if !ok {
				return "throws _TypeError"
			}
			return bdDesc(o.CopyWithSide(&s[6]))
		})
		for r, rect := range bdRects {
			for _, td := range tds {
				k := fmt.Sprintf("inter.%s.r%d.%s", e.name, r, td.Name())
				out[k+".outer"] = bdTry(func() string { return bdPathSig(e.b.GetOuterPath(rect, &td), rect, 9) })
				out[k+".inner"] = bdTry(func() string { return bdPathSig(e.b.GetInnerPath(rect, &td), rect, 9) })
				out[k+".paint"] = bdTry(func() string { return bdPaint(rect, func(c ui.Canvas) { e.b.Paint(c, rect, &td) }) })
			}
		}
		for j, o := range others {
			for _, t := range []float64{0.15, 0.5, 0.85} {
				k := fmt.Sprintf("inter.%s.%d.", e.name, j)
				out[k+"lerp."+bd(t)] = bdTry(func() string { return bdDesc(p.ShapeBorderLerp(e.b, o, t)) })
				out[k+"rlerp."+bd(t)] = bdTry(func() string { return bdDesc(p.ShapeBorderLerp(o, e.b, t)) })
			}
		}
	}

	var boxes []p.BoxBorder
	for _, x := range b {
		if bx, ok := x.(p.BoxBorder); ok {
			boxes = append(boxes, bx)
		}
	}
	w := func(side p.BorderSide, width float64) *p.BorderSide {
		r := side.CopyWith(p.BorderSideCopyWithOptions{Width: &width})
		return &r
	}
	boxes = append(boxes,
		p.NewBorder(p.BorderOptions{Top: &s[2], Right: w(s[2], 6), Bottom: &s[2], Left: w(s[2], 1)}),
		p.NewBorder(p.BorderOptions{Top: &s[2], Left: &s[5]}),
		p.NewBorder(p.BorderOptions{Bottom: &s[6]}),
		p.NewBorderDirectional(p.BorderDirectionalOptions{Start: &s[2], Top: w(s[2], 9), End: &s[2]}),
		p.NewBorderAll(p.BorderSideOptions{Color: ptr(ui.NewColor(0xFF00FF00)), Width: ptr(4.0), StrokeAlign: ptr(p.BorderSideStrokeAlignCenter)}),
		p.NewBorderAll(p.BorderSideOptions{Color: ptr(ui.NewColor(0xFF00FF00)), Width: ptr(0.0)}),
		p.NewBorderAll(p.BorderSideOptions{Style: ptr(p.BorderStyleNone)}),
	)
	radii := []*p.BorderRadius{nil, ptr(p.BorderRadiusZero), ptr(p.NewBorderRadiusAll(bdCirc(8))),
		ptr(p.BorderRadius{TopLeft: ui.NewRadiusElliptical(20, 10), BottomRight: bdCirc(4)})}
	for i, bx := range boxes {
		out[fmt.Sprintf("box.%d", i)] = bx.String() + "|" + fmt.Sprint(bx.IsUniform()) + "|" + bdSideF(bx.Top()) + "|" + bdSideF(bx.Bottom())
		for _, shape := range p.BoxShapeValues {
			for _, radius := range radii {
				for _, td := range []*ui.TextDirection{ptr(ui.TextDirectionLtr), ptr(ui.TextDirectionRtl), nil} {
					rk, tk := "null", "null"
					if radius != nil {
						rk = bdRRect(radius.ToRRect(bdRects[0]))
					}
					if td != nil {
						tk = td.String()
					}
					out[fmt.Sprintf("box.%d.%s.%s.%s", i, shape, rk, tk)] = bdTry(func() string {
						return bdPaint(bdRects[0], func(c ui.Canvas) {
							bx.PaintBox(c, bdRects[0], p.BoxBorderPaintOptions{TextDirection: td, Shape: shape, BorderRadius: radius})
						})
					})
				}
			}
		}
	}

	for i := range s {
		for j := range s {
			out[fmt.Sprintf("paintBorder.%d.%d", i, j)] = bdTry(func() string {
				return bdPaint(bdRects[1], func(c ui.Canvas) {
					p.PaintBorder(c, bdRects[1], p.PaintBorderOptions{Top: &s[i], Right: &s[j], Bottom: &s[(i+j)%len(s)], Left: &s[(i+2*j)%len(s)]})
				})
			})
		}
	}

	k := 0
	for _, points := range []float64{2, 2.5, 3, 4.25, 5, 8} {
		for _, inner := range []float64{0, 0.35, 1} {
			for _, pv := range [][2]float64{{0, 0}, {0.5, 0.2}, {1, 0}, {0, 1}} {
				for _, rot := range []float64{0, 27, 270} {
					for _, squash := range []float64{0, 0.5, 1} {
						k++
						if k%3 != 0 {
							continue
						}
						star := p.NewStarBorder(p.StarBorderOptions{Points: ptr(points), InnerRadiusRatio: ptr(inner),
							PointRounding: pv[0], ValleyRounding: pv[1], Rotation: rot, Squash: squash})
						key := fmt.Sprintf("star.%s.%s.%s.%s.%s.%s", bd(points), bd(inner), bd(pv[0]), bd(pv[1]), bd(rot), bd(squash))
						out[key] = star.String() + "|" + bd(star.Rotation()) + "|" + bdPathSig(star.GetOuterPath(bdRects[k%3], nil), bdRects[k%3], 9)
					}
				}
			}
		}
	}
	for _, sides := range []float64{2, 3, 3.5, 6, 11.75} {
		for _, pr := range []float64{0, 0.5, 1} {
			poly := p.NewStarBorderPolygon(p.StarBorderPolygonOptions{Sides: ptr(sides), PointRounding: pr, Rotation: 45, Squash: 0.25})
			key := fmt.Sprintf("polygon.%s.%s", bd(sides), bd(pr))
			out[key] = poly.String() + "|" + bd(poly.InnerRadiusRatio()) + "|" + bdPathSig(poly.GetOuterPath(bdRects[0], nil), bdRects[0], 9)
			out[key+".copy"] = poly.CopyWith(p.StarBorderCopyWithOptions{Points: ptr(sides + 1)}).String()
		}
	}
	for _, other := range []p.ShapeBorder{
		p.NewCircleBorder(),
		p.NewCircleBorder(p.CircleBorderOptions{Eccentricity: 0.6}),
		p.NewStadiumBorder(),
		p.NewRoundedRectangleBorder(),
		p.NewRoundedRectangleBorder(p.RRectBorderOptions{BorderRadius: p.BorderRadius{BottomLeft: bdCirc(10)}}),
		p.NewStarBorder(p.StarBorderOptions{Points: ptr(6.0), PointRounding: 0.5, ValleyRounding: 0.5, InnerRadiusRatio: ptr(0.5), Rotation: 90}),
	} {
		for _, from := range []p.StarBorder{p.NewStarBorder(), p.NewStarBorderPolygon(p.StarBorderPolygonOptions{Sides: ptr(2.2)})} {
			for _, t := range []float64{0, 0.2, 0.3, 0.5, 0.7, 0.9, 1} {
				key := "starlerp." + from.String() + "." + other.String() + "." + bd(t)
				out[key+".to"] = bdTry(func() string { return bdDesc(from.LerpTo(other, t)) })
				out[key+".from"] = bdTry(func() string { return bdDesc(from.LerpFrom(other, t)) })
			}
		}
	}

	edges := []*p.LinearBorderEdge{nil, ptr(p.NewLinearBorderEdge()),
		ptr(p.NewLinearBorderEdge(p.LinearBorderEdgeOptions{Size: ptr(0.25), Alignment: -1})),
		ptr(p.NewLinearBorderEdge(p.LinearBorderEdgeOptions{Size: ptr(0.0), Alignment: 0.5}))}
	for i, a := range edges {
		if a == nil {
			out[fmt.Sprintf("edge.%d", i)] = "null"
		} else {
			out[fmt.Sprintf("edge.%d", i)] = a.String()
		}
		for j, c := range edges {
			for _, t := range []float64{0, 0.4, 1} {
				out[fmt.Sprintf("edge.%d.%d.%s", i, j, bd(t))] = bdTry(func() string {
					e := p.LinearBorderEdgeLerp(a, c, t)
					if e == nil {
						return "null"
					}
					return e.String() + "|" + bd(e.Size) + "|" + bd(e.Alignment)
				})
			}
		}
	}

	host := ui.NewRectFromLTRB(0, 100, 300, 300)
	for _, inverted := range []bool{false, true} {
		for _, guest := range []*ui.Rect{nil, ptr(ui.NewRectFromLTWH(50, 50, 10, 10)),
			ptr(ui.NewRectFromLTRB(190, 85, 210, 105)), ptr(ui.NewRectFromLTRB(190, 95, 210, 115)),
			ptr(ui.NewRectFromLTRB(190, 285, 210, 305)), ptr(ui.NewRectFromLTRB(190, 295, 210, 315)),
			ptr(ui.NewRectFromLTRB(100.5, 70.25, 160.5, 130.25))} {
			gk := "null"
			if guest != nil {
				gk = bdRect(*guest)
			}
			out[fmt.Sprintf("notch.%v.%s", inverted, gk)] = bdTry(func() string {
				return bdPathSig(p.CircularNotchedRectangle{Inverted: inverted}.GetOuterPath(host, guest), host, 15)
			})
		}
	}
	out["notch.auto.noguest"] = bdPathSig(p.NewAutomaticNotchedShape(b[10], b[6]).GetOuterPath(bdRects[1], nil), bdRects[1], 9)

	for i := range b {
		var buf strings.Builder
		for j := range b {
			if p.ShapeBorderEqual(b[i], b[j]) {
				buf.WriteString("1")
			} else {
				buf.WriteString("0")
			}
		}
		out[fmt.Sprintf("eq.%d", i)] = buf.String()
	}
	return out
}

func TestPaintingBordersMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_borders.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	got := paintingBordersCases()
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
