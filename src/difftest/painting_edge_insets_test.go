package difftest

// Differential test of painting's edge_insets.go and border_radius.go
// against Flutter 3.50.1 (flutter/test/painting_edge_insets_test.dart):
// every EdgeInsetsGeometry/BorderRadiusGeometry type, mixed sums included,
// through operators, lerps (null ends, t outside 0..1), resolve, toString,
// inflate/deflate, clamp, copyWith and factories. Compared exactly.

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	p "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func eiTypeName(g p.EdgeInsetsGeometry) string {
	switch g.(type) {
	case p.EdgeInsets:
		return "EdgeInsets"
	case p.EdgeInsetsDirectional:
		return "EdgeInsetsDirectional"
	}
	return "_MixedEdgeInsets"
}

func brTypeName(g p.BorderRadiusGeometry) string {
	switch g.(type) {
	case p.BorderRadius:
		return "BorderRadius"
	case p.BorderRadiusDirectional:
		return "BorderRadiusDirectional"
	}
	return "_MixedBorderRadius"
}

func eiLTRB(e p.EdgeInsets) []any { return jnums(e.Left, e.Top, e.Right, e.Bottom) }
func eiSize(s ui.Size) []any      { return jnums(s.Width, s.Height) }
func eiRadius(r ui.Radius) []any  { return jnums(r.X, r.Y) }
func eiRRect(r ui.RRect) []any {
	return jnums(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY,
		r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
}
func eiRSE(r ui.RSuperellipse) []any {
	return jnums(r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY,
		r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
}
func brCorners(b p.BorderRadius) []any {
	return jnums(b.TopLeft.X, b.TopLeft.Y, b.TopRight.X, b.TopRight.Y,
		b.BottomLeft.X, b.BottomLeft.Y, b.BottomRight.X, b.BottomRight.Y)
}

// try returns f's result, or the panic's text (Dart's '$e').
func try(f func() any) (v any) {
	defer func() {
		if r := recover(); r != nil {
			v = apiErr(func() { panic(r) })
		}
	}()
	return f()
}

func throwsAny(f func()) (b bool) {
	defer func() { b = recover() != nil }()
	f()
	return false
}

func eiDesc(g p.EdgeInsetsGeometry) any {
	if g == nil {
		return "null"
	}
	res := func(d ui.TextDirection) any { return try(func() any { return eiLTRB(g.Resolve(&d)) }) }
	out := []any{eiTypeName(g), g.String(), res(ui.TextDirectionLtr), res(ui.TextDirectionRtl),
		jnum(g.Horizontal()), jnum(g.Vertical()), g.IsNonNegative(), eiSize(g.CollapsedSize())}
	if d, ok := g.(p.EdgeInsetsDirectional); ok {
		out = append(out, jnums(d.Start, d.Top, d.End, d.Bottom))
	}
	return out
}

func brDesc(g p.BorderRadiusGeometry) any {
	if g == nil {
		return "null"
	}
	res := func(d ui.TextDirection) any { return try(func() any { return brCorners(g.Resolve(&d)) }) }
	out := []any{brTypeName(g), g.String(), res(ui.TextDirectionLtr), res(ui.TextDirectionRtl)}
	if d, ok := g.(p.BorderRadiusDirectional); ok {
		out = append(out, jnums(d.TopStart.X, d.TopStart.Y, d.TopEnd.X, d.TopEnd.Y,
			d.BottomStart.X, d.BottomStart.Y, d.BottomEnd.X, d.BottomEnd.Y))
	}
	return out
}

func eiPtr(e *p.EdgeInsets) p.EdgeInsetsGeometry {
	if e == nil {
		return nil
	}
	return *e
}

func eidPtr(e *p.EdgeInsetsDirectional) p.EdgeInsetsGeometry {
	if e == nil {
		return nil
	}
	return *e
}

func brPtr(b *p.BorderRadius) p.BorderRadiusGeometry {
	if b == nil {
		return nil
	}
	return *b
}

func brdPtr(b *p.BorderRadiusDirectional) p.BorderRadiusGeometry {
	if b == nil {
		return nil
	}
	return *b
}

func paintingEdgeInsetsCases() map[string]any {
	out := map[string]any{}
	ds := dart.DoubleToString
	inf := jnumInf()

	// ---- EdgeInsetsGeometry ----
	type named[T any] struct {
		n string
		g T
	}
	insets := []named[p.EdgeInsetsGeometry]{
		{"e1", p.NewEdgeInsetsFromLTRB(5, 7.5, 11.25, 13)},
		{"e2", p.NewEdgeInsetsAll(3.3)},
		{"e3", p.NewEdgeInsetsSymmetric(p.EdgeInsetsSymmetricOptions{Vertical: 2, Horizontal: -4.4})},
		{"e4", p.EdgeInsetsZero},
		{"e5", p.EdgeInsets{Left: 0.1, Bottom: 1e22}},
		{"e6", p.NewEdgeInsetsFromLTRB(negZero(), 0.05, -0.05, 0.25)},
		{"e7", p.NewEdgeInsetsAll(inf)},
		{"e8", p.EdgeInsets{Top: 1.25, Right: 2.35}},
		{"d1", p.NewEdgeInsetsDirectionalFromSTEB(1, 2, 3, 4)},
		{"d2", p.NewEdgeInsetsDirectionalAll(2.5)},
		{"d3", p.NewEdgeInsetsDirectionalSymmetric(p.EdgeInsetsSymmetricOptions{Horizontal: 7, Vertical: -1})},
		{"d4", p.EdgeInsetsDirectionalZero},
		{"d5", p.EdgeInsetsDirectional{Start: negZero(), End: 0.04}},
		{"d6", p.EdgeInsetsDirectional{Top: 0.15, Bottom: -2.65}},
		{"m1", p.NewEdgeInsetsDirectionalFromSTEB(1, 2, 3, 4).AddGeometry(p.NewEdgeInsetsFromLTRB(5, 7.5, 11.25, 13))},
		{"m2", p.NewEdgeInsetsAll(3.3).Subtract(p.NewEdgeInsetsDirectionalSymmetric(p.EdgeInsetsSymmetricOptions{Horizontal: 7, Vertical: -1}))},
		{"m3", p.NewEdgeInsetsFromLTRB(5, 7.5, 11.25, 13).AddGeometry(p.EdgeInsetsDirectionalZero)},
		{"m4", p.NewEdgeInsetsDirectionalFromSTEB(1, 2, 3, 4).AddGeometry(p.EdgeInsetsZero)},
		{"m5", p.EdgeInsetsGeometryInfinity},
		{"m6", p.EdgeInsetsDirectional{Start: 1}.AddGeometry(p.EdgeInsets{Top: 3})},
		{"m7", p.EdgeInsetsDirectional{Start: 4}.AddGeometry(p.EdgeInsets{Left: 3})},
		{"m8", p.EdgeInsetsZero.AddGeometry(p.EdgeInsetsDirectionalZero)},
	}
	byName := map[string]p.EdgeInsetsGeometry{}
	for _, x := range insets {
		byName[x.n] = x.g
	}
	divisors := []float64{1.5, 4, -3, 2.5, 5}
	for _, x := range insets {
		n, g := x.n, x.g
		out["ei."+n] = eiDesc(g)
		out["ei."+n+".resolve.null"] = apiErr(func() { g.Resolve(nil) })
		out["ei."+n+".along"] = jnums(g.Along(p.AxisHorizontal), g.Along(p.AxisVertical))
		out["ei."+n+".inflateSize"] = eiSize(g.InflateSize(ui.NewSize(100, 50)))
		out["ei."+n+".deflateSize"] = eiSize(g.DeflateSize(ui.NewSize(100, 50)))
		out["ei."+n+".flipped"] = eiDesc(g.FlippedGeometry())
		out["ei."+n+".neg"] = eiDesc(g.NegGeometry())
		for _, d := range divisors {
			k := "ei." + n
			out[k+".mul."+ds(d)] = eiDesc(g.MulGeometry(d))
			out[k+".div."+ds(d)] = eiDesc(g.DivGeometry(d))
			out[k+".truncDiv."+ds(d)] = try(func() any { return eiDesc(g.TruncDivGeometry(d)) })
			out[k+".mod."+ds(d)] = eiDesc(g.ModGeometry(d))
		}
		out["ei."+n+".lerp.null.a"] = eiDesc(p.EdgeInsetsGeometryLerp(g, nil, 0.25))
		out["ei."+n+".lerp.null.b"] = eiDesc(p.EdgeInsetsGeometryLerp(nil, g, 0.25))
		out["ei."+n+".lerp.self"] = eiDesc(p.EdgeInsetsGeometryLerp(g, g, 0.5))
	}
	pairNames := []string{"e1", "e3", "e6", "d1", "d3", "m1", "m2", "e4", "d4"}
	ts := []float64{0, 0.25, 0.5, 1, 1.5, -0.5, 0.1}
	for _, a := range pairNames {
		for _, b := range pairNames {
			ga, gb := byName[a], byName[b]
			out["ei."+a+".add."+b] = eiDesc(ga.AddGeometry(gb))
			out["ei."+a+".subtract."+b] = eiDesc(ga.Subtract(gb))
			out["ei."+a+".eq."+b] = p.EdgeInsetsGeometryEqual(ga, gb)
			for _, t := range ts {
				out["ei."+a+".lerp."+b+"."+ds(t)] = eiDesc(p.EdgeInsetsGeometryLerp(ga, gb, t))
			}
		}
	}
	out["ei.lerp.null.null"] = eiDesc(p.EdgeInsetsGeometryLerp(nil, nil, 0.5))
	out["ei.lerp.inf.finite"] = throwsAny(func() { p.EdgeInsetsGeometryLerp(byName["m5"], byName["e1"], 0.5) })
	out["ei.lerp.inf.inf"] = eiDesc(p.EdgeInsetsGeometryLerp(byName["e7"], byName["e7"], 0.5))
	out["ei.lerp.inf.inf.0"] = eiDesc(p.EdgeInsetsGeometryLerp(byName["m5"], p.EdgeInsetsGeometryInfinity, 0))

	ea, eb := p.NewEdgeInsetsFromLTRB(5, 7.5, 11.25, 13), p.NewEdgeInsetsFromLTRB(-1, 0.3, 2, 100)
	da, db := p.NewEdgeInsetsDirectionalFromSTEB(1, 2, 3, 4), p.NewEdgeInsetsDirectionalFromSTEB(0.7, -2, 30, 0.1)
	for _, t := range ts {
		out["edgeInsets.lerp."+ds(t)] = eiDesc(eiPtr(p.EdgeInsetsLerp(&ea, &eb, t)))
		out["edgeInsets.lerp.nullA."+ds(t)] = eiDesc(eiPtr(p.EdgeInsetsLerp(nil, &eb, t)))
		out["edgeInsets.lerp.nullB."+ds(t)] = eiDesc(eiPtr(p.EdgeInsetsLerp(&ea, nil, t)))
		out["directional.lerp."+ds(t)] = eiDesc(eidPtr(p.EdgeInsetsDirectionalLerp(&da, &db, t)))
		out["directional.lerp.nullA."+ds(t)] = eiDesc(eidPtr(p.EdgeInsetsDirectionalLerp(nil, &db, t)))
		out["directional.lerp.nullB."+ds(t)] = eiDesc(eidPtr(p.EdgeInsetsDirectionalLerp(&da, nil, t)))
	}
	out["edgeInsets.lerp.null"] = eiDesc(eiPtr(p.EdgeInsetsLerp(nil, nil, 0.5)))
	out["directional.lerp.null"] = eiDesc(eidPtr(p.EdgeInsetsDirectionalLerp(nil, nil, 0.5)))
	out["edgeInsets.plus"] = eiDesc(ea.Add(eb))
	out["edgeInsets.minus"] = eiDesc(ea.Sub(eb))
	out["directional.plus"] = eiDesc(da.Add(db))
	out["directional.minus"] = eiDesc(da.Sub(db))
	out["edgeInsets.corners"] = []any{
		jnums(ea.TopLeft().Dx, ea.TopLeft().Dy, ea.TopRight().Dx, ea.TopRight().Dy),
		jnums(ea.BottomLeft().Dx, ea.BottomLeft().Dy, ea.BottomRight().Dx, ea.BottomRight().Dy),
	}
	rect := ui.NewRectFromLTRB(10, 20, 110, 220)
	for _, n := range []string{"e1", "e3", "e6", "e8"} {
		e := byName[n].(p.EdgeInsets)
		out["edgeInsets."+n+".inflateRect"] = apiRect(e.InflateRect(rect))
		out["edgeInsets."+n+".deflateRect"] = apiRect(e.DeflateRect(rect))
		r := ui.NewRRectFromLTRBAndCorners(10, 20, 110, 220, ui.RRectCorners{
			TopLeft: ui.NewRadiusElliptical(5, 6), TopRight: ui.NewRadiusCircular(3.3),
			BottomRight: ui.NewRadiusElliptical(12, 0.5), BottomLeft: ui.RadiusZero})
		out["edgeInsets."+n+".inflateRRect"] = eiRRect(e.InflateRRect(r))
		out["edgeInsets."+n+".deflateRRect"] = eiRRect(e.DeflateRRect(r))
		out["edgeInsets."+n+".flippedConcrete"] = eiLTRB(e.Flipped())
	}
	f := func(v float64) *float64 { return &v }
	out["edgeInsets.copyWith"] = []any{
		eiLTRB(ea.CopyWith()),
		eiLTRB(ea.CopyWith(p.EdgeInsetsCopyWithOptions{Left: f(1), Bottom: f(-2)})),
		eiLTRB(ea.CopyWith(p.EdgeInsetsCopyWithOptions{Top: f(9), Right: f(0.5)})),
	}
	out["directional.copyWith"] = []any{
		eiDesc(da.CopyWith()),
		eiDesc(da.CopyWith(p.EdgeInsetsDirectionalCopyWithOptions{Start: f(1.5), Bottom: f(-2)})),
		eiDesc(da.CopyWith(p.EdgeInsetsDirectionalCopyWithOptions{Top: f(9), End: f(0.5)})),
	}
	bounds := []named[[2]p.EdgeInsetsGeometry]{
		{"abs", [2]p.EdgeInsetsGeometry{p.NewEdgeInsetsAll(2), p.NewEdgeInsetsAll(6)}},
		{"dir", [2]p.EdgeInsetsGeometry{p.NewEdgeInsetsDirectionalAll(1), p.NewEdgeInsetsDirectionalAll(3)}},
		{"inf", [2]p.EdgeInsetsGeometry{p.EdgeInsetsGeometryInfinity.NegGeometry(), p.EdgeInsetsGeometryInfinity}},
		{"zeroInf", [2]p.EdgeInsetsGeometry{p.EdgeInsetsZero, p.EdgeInsetsGeometryInfinity}},
	}
	for _, n := range []string{"e1", "e3", "e6", "d1", "d3", "d6", "m1", "m2", "m5"} {
		for _, b := range bounds {
			g, mm := byName[n], b.g
			out["ei."+n+".clamp."+b.n] = func() (v any) {
				defer func() {
					if recover() != nil {
						v = "error"
					}
				}()
				return eiDesc(g.Clamp(mm[0], mm[1]))
			}()
		}
	}
	out["ei.factories"] = []any{
		eiDesc(p.NewEdgeInsetsGeometryAll(10)),
		eiDesc(p.NewEdgeInsetsGeometryOnly(p.EdgeInsets{Left: 10, Top: 20, Right: 30, Bottom: 40})),
		eiDesc(p.NewEdgeInsetsGeometryDirectional(p.EdgeInsetsDirectional{Start: 10, Top: 20, End: 30, Bottom: 40})),
		eiDesc(p.NewEdgeInsetsGeometrySymmetric(p.EdgeInsetsSymmetricOptions{Horizontal: 10, Vertical: 20})),
		eiDesc(p.NewEdgeInsetsGeometryFromLTRB(10, 20, 30, 40)),
		eiDesc(p.NewEdgeInsetsGeometryFromSTEB(10, 20, 20, 40)),
		eiDesc(p.EdgeInsetsGeometryZero),
		eiDesc(p.NewEdgeInsetsSymmetric()),
		eiDesc(p.NewEdgeInsetsSymmetric(p.EdgeInsetsSymmetricOptions{Vertical: 3})),
		eiDesc(p.NewEdgeInsetsDirectionalSymmetric(p.EdgeInsetsSymmetricOptions{Horizontal: 3})),
	}

	// ---- BorderRadiusGeometry ----
	ell, circ := ui.NewRadiusElliptical, ui.NewRadiusCircular
	radii := []named[p.BorderRadiusGeometry]{
		{"b1", p.NewBorderRadiusAll(ell(5, 7))},
		{"b2", p.NewBorderRadiusCircular(3.25)},
		{"b3", p.NewBorderRadiusVertical(p.BorderRadiusVerticalOptions{Top: circ(2), Bottom: ell(4, 1)})},
		{"b4", p.NewBorderRadiusHorizontal(p.BorderRadiusHorizontalOptions{Left: circ(-1), Right: ell(0.05, 0.25)})},
		{"b5", p.BorderRadiusZero},
		{"b6", p.BorderRadius{TopRight: circ(1), BottomLeft: ell(2, 3)}},
		{"b7", p.BorderRadius{TopLeft: circ(1)}},
		{"b8", p.BorderRadius{BottomRight: ell(negZero(), 2.35)}},
		{"b9", p.NewBorderRadiusVertical(p.BorderRadiusVerticalOptions{Bottom: circ(1e22)})},
		{"bd1", p.NewBorderRadiusDirectionalAll(ell(5, 7))},
		{"bd2", p.NewBorderRadiusDirectionalCircular(2)},
		{"bd3", p.NewBorderRadiusDirectionalVertical(p.BorderRadiusVerticalOptions{Top: circ(2), Bottom: ell(4, 1)})},
		{"bd4", p.NewBorderRadiusDirectionalHorizontal(p.BorderRadiusDirectionalHorizontalOptions{Start: circ(-1), End: ell(9, 0.5)})},
		{"bd5", p.BorderRadiusDirectionalZero},
		{"bd6", p.BorderRadiusDirectional{TopEnd: circ(1), BottomStart: ell(2, 3)}},
		{"bd7", p.BorderRadiusDirectional{TopStart: circ(4), BottomEnd: circ(6.5)}},
		{"bm1", p.NewBorderRadiusAll(ell(5, 7)).AddGeometry(p.BorderRadiusDirectional{TopEnd: circ(1), BottomStart: ell(2, 3)})},
		{"bm2", p.NewBorderRadiusDirectionalAll(ell(5, 7)).Subtract(p.BorderRadius{TopRight: circ(1), BottomLeft: ell(2, 3)})},
		{"bm3", p.BorderRadiusZero.AddGeometry(p.BorderRadiusDirectionalZero)},
		{"bm4", p.NewBorderRadiusCircular(3.25).AddGeometry(p.NewBorderRadiusDirectionalCircular(2))},
		{"bm5", p.NewBorderRadiusAll(ell(1, 2)).AddGeometry(p.NewBorderRadiusDirectionalAll(ell(3, 4)))},
		{"bm6", p.BorderRadius{TopLeft: circ(1)}.AddGeometry(p.BorderRadiusDirectionalZero)},
	}
	brByName := map[string]p.BorderRadiusGeometry{}
	for _, x := range radii {
		brByName[x.n] = x.g
	}
	rects := []named[ui.Rect]{
		{"[19.0, 23.0, 29.0, 31.0]", ui.NewRectFromLTRB(19, 23, 29, 31)},
		{"[-5.0, 0.0, 95.5, 3.0]", ui.NewRectFromLTWH(-5, 0, 100.5, 3)},
	}
	for _, x := range radii {
		n, g := x.n, x.g
		out["br."+n] = brDesc(g)
		out["br."+n+".resolve.null"] = apiErr(func() { g.Resolve(nil) })
		out["br."+n+".neg"] = brDesc(g.NegGeometry())
		for _, d := range divisors {
			k := "br." + n
			out[k+".mul."+ds(d)] = brDesc(g.MulGeometry(d))
			out[k+".div."+ds(d)] = brDesc(g.DivGeometry(d))
			out[k+".truncDiv."+ds(d)] = try(func() any { return brDesc(g.TruncDivGeometry(d)) })
			out[k+".mod."+ds(d)] = brDesc(g.ModGeometry(d))
		}
		out["br."+n+".lerp.null.a"] = brDesc(p.BorderRadiusGeometryLerp(g, nil, 0.25))
		out["br."+n+".lerp.null.b"] = brDesc(p.BorderRadiusGeometryLerp(nil, g, 0.25))
		out["br."+n+".lerp.self"] = brDesc(p.BorderRadiusGeometryLerp(g, g, 0.5))
		l := ui.TextDirectionLtr
		ltr := g.Resolve(&l)
		for _, r := range rects {
			out["br."+n+".toRRect."+r.n] = eiRRect(ltr.ToRRect(r.g))
			out["br."+n+".toRSuperellipse."+r.n] = eiRSE(ltr.ToRSuperellipse(r.g))
		}
	}
	brPairs := []string{"b1", "b4", "b6", "bd1", "bd4", "bd6", "bm1", "bm2", "b5", "bd5"}
	for _, a := range brPairs {
		for _, b := range brPairs {
			ga, gb := brByName[a], brByName[b]
			out["br."+a+".add."+b] = brDesc(ga.AddGeometry(gb))
			out["br."+a+".subtract."+b] = brDesc(ga.Subtract(gb))
			out["br."+a+".eq."+b] = ga == gb
			for _, t := range ts {
				out["br."+a+".lerp."+b+"."+ds(t)] = brDesc(p.BorderRadiusGeometryLerp(ga, gb, t))
			}
		}
	}
	out["br.lerp.null.null"] = brDesc(p.BorderRadiusGeometryLerp(nil, nil, 0.5))
	ba := p.BorderRadius{TopLeft: ell(10, 20), TopRight: ell(30, 40), BottomLeft: ell(50, 60)}
	bb := p.BorderRadius{TopRight: ell(100, 110), BottomLeft: ell(120.3, 130), BottomRight: ell(-140, 150)}
	bda := p.BorderRadiusDirectional{TopStart: ell(10, 20), TopEnd: ell(30, 40), BottomStart: ell(50, 60)}
	bdb := p.BorderRadiusDirectional{TopEnd: ell(100, 110), BottomStart: ell(120.3, 130), BottomEnd: ell(-140, 150)}
	for _, t := range ts {
		out["borderRadius.lerp."+ds(t)] = brDesc(brPtr(p.BorderRadiusLerp(&ba, &bb, t)))
		out["borderRadius.lerp.nullA."+ds(t)] = brDesc(brPtr(p.BorderRadiusLerp(nil, &bb, t)))
		out["borderRadius.lerp.nullB."+ds(t)] = brDesc(brPtr(p.BorderRadiusLerp(&ba, nil, t)))
		out["brDirectional.lerp."+ds(t)] = brDesc(brdPtr(p.BorderRadiusDirectionalLerp(&bda, &bdb, t)))
		out["brDirectional.lerp.nullA."+ds(t)] = brDesc(brdPtr(p.BorderRadiusDirectionalLerp(nil, &bdb, t)))
		out["brDirectional.lerp.nullB."+ds(t)] = brDesc(brdPtr(p.BorderRadiusDirectionalLerp(&bda, nil, t)))
	}
	out["borderRadius.lerp.null"] = brDesc(brPtr(p.BorderRadiusLerp(nil, nil, 0.5)))
	out["brDirectional.lerp.null"] = brDesc(brdPtr(p.BorderRadiusDirectionalLerp(nil, nil, 0.5)))
	out["borderRadius.plus"] = brDesc(ba.Add(bb))
	out["borderRadius.minus"] = brDesc(ba.Sub(bb))
	out["brDirectional.plus"] = brDesc(bda.Add(bdb))
	out["brDirectional.minus"] = brDesc(bda.Sub(bdb))
	rp := func(r ui.Radius) *ui.Radius { return &r }
	out["borderRadius.copyWith"] = []any{
		brDesc(ba.CopyWith()),
		brDesc(ba.CopyWith(p.BorderRadiusCopyWithOptions{TopLeft: rp(ui.RadiusZero), BottomRight: rp(circ(3))})),
		brDesc(ba.CopyWith(p.BorderRadiusCopyWithOptions{TopRight: rp(ell(1, 2)), BottomLeft: rp(circ(0.5))})),
	}
	r5, r10 := circ(5), ell(10, 11)
	type hz = p.BorderRadiusGeometryHorizontalOptions
	out["br.factories"] = []any{
		brDesc(p.NewBorderRadiusGeometryAll(r10)),
		brDesc(p.NewBorderRadiusGeometryCircular(10)),
		brDesc(p.NewBorderRadiusGeometryHorizontal(hz{Left: &r5, Right: &r10})),
		brDesc(p.NewBorderRadiusGeometryHorizontal(hz{Left: &r5})),
		brDesc(p.NewBorderRadiusGeometryHorizontal(hz{Start: &r5, End: &r10})),
		brDesc(p.NewBorderRadiusGeometryHorizontal(hz{End: &r10})),
		brDesc(p.NewBorderRadiusGeometryHorizontal()),
		throwsAny(func() { p.NewBorderRadiusGeometryHorizontal(hz{Start: &r5, Left: &r10}) }),
		throwsAny(func() { p.NewBorderRadiusGeometryHorizontal(hz{End: &r5, Right: &r10}) }),
		brDesc(p.NewBorderRadiusGeometryOnly(p.BorderRadius{TopLeft: r5, BottomRight: r10})),
		brDesc(p.NewBorderRadiusGeometryDirectional(p.BorderRadiusDirectional{TopStart: r5, BottomEnd: r10})),
		brDesc(p.NewBorderRadiusGeometryVertical(p.BorderRadiusVerticalOptions{Top: r5, Bottom: r10})),
		brDesc(p.BorderRadiusGeometryZero),
		brDesc(p.NewBorderRadiusVertical()),
		brDesc(p.NewBorderRadiusHorizontal(p.BorderRadiusHorizontalOptions{Right: r5})),
		brDesc(p.NewBorderRadiusDirectionalVertical(p.BorderRadiusVerticalOptions{Bottom: r5})),
		brDesc(p.NewBorderRadiusDirectionalHorizontal()),
	}
	return out
}

func negZero() float64 { z := 0.0; return -z }

func jnumInf() float64 { z := 0.0; return 1 / z }

func TestPaintingEdgeInsetsMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_edge_insets.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	cases := paintingEdgeInsetsCases()
	if len(ref) != len(cases) {
		t.Errorf("reference has %d cases, Go has %d", len(ref), len(cases))
	}
	bad := 0
	for name, got := range cases {
		want, ok := ref[name]
		if !ok {
			t.Errorf("%s: missing from the reference", name)
			continue
		}
		// Exact, and sign-sensitive: reflect.DeepEqual would take -0.0 for
		// 0.0, so the decoded values are compared re-encoded.
		g := toJSONShape(t, got)
		gb, _ := json.Marshal(g)
		wb, _ := json.Marshal(want)
		if !reflect.DeepEqual(g, want) || string(gb) != string(wb) {
			if bad++; bad <= 40 {
				t.Errorf("%s:\n Go:      %v\n Flutter: %v", name, g, want)
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d cases differ", bad, len(cases))
	}
}
