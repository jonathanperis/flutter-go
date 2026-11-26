package difftest

// Differential test of painting's alignment.dart, fractional_offset.dart,
// box_fit.dart, geometry.dart and basic_types.dart against Flutter: the
// same cases as flutter/test/painting_alignment_test.dart, compared
// exactly (every double bit for bit, -0.0 included; every string).

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	p "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// pn is the Dart side's n(): non-finite doubles and -0.0 as strings.
func pn(v float64) any {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return dart.DoubleToString(v)
	}
	if v == 0 && math.Signbit(v) {
		return "-0.0"
	}
	return v
}

func pns(vs ...float64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = pn(v)
	}
	return out
}

func pErr(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				s = e.Error()
			} else {
				s = fmt.Sprint(r)
			}
		}
	}()
	f()
	return "no error"
}

var (
	pLtr = ui.TextDirectionLtr
	pRtl = ui.TextDirectionRtl
)

func alignDesc(g p.AlignmentGeometry) any {
	if g == nil {
		return "null"
	}
	switch v := g.(type) {
	case p.FractionalOffset:
		return append([]any{"FractionalOffset", v.String()}, pns(v.X, v.Y, v.Dx(), v.Dy())...)
	case p.Alignment:
		return append([]any{"Alignment", v.String()}, pns(v.X, v.Y)...)
	case p.AlignmentDirectional:
		return append([]any{"AlignmentDirectional", v.String()}, pns(v.Start, v.Y)...)
	}
	l, r := g.Resolve(&pLtr), g.Resolve(&pRtl)
	return append([]any{"_MixedAlignment", g.String()}, pns(l.X, r.X, l.Y)...)
}

func alignLikeDesc(g p.AlignmentLike) any {
	if g == nil {
		return "null"
	}
	return alignDesc(g)
}

func pOff(o ui.Offset) []any { return pns(o.Dx, o.Dy) }
func pSz(s ui.Size) []any    { return pns(s.Width, s.Height) }
func pRc(r ui.Rect) []any    { return pns(r.Left, r.Top, r.Right, r.Bottom) }

func alignmentGeoms() []p.AlignmentGeometry {
	return []p.AlignmentGeometry{
		p.AlignmentTopLeft,
		p.AlignmentCenter,
		p.AlignmentBottomRight,
		p.NewAlignment(0.3, -0.7),
		p.NewAlignment(math.Copysign(0, -1), 0.5),
		p.NewAlignment(1e-3, 2.5),
		p.NewAlignment(-1.25, 0.1),
		p.NewFractionalOffset(0.25, 0.8),
		p.FractionalOffsetCenter,
		p.NewFractionalOffset(0.1, 0.7),
		p.AlignmentDirectionalTopStart,
		p.NewAlignmentDirectional(0.35, -0.2),
		p.NewAlignmentDirectional(-2.5, 1.5),
		p.NewAlignment(0.1, 0.2).Add(p.NewAlignmentDirectional(0.3, 0.4)),
		p.AlignmentBottomRight.Add(p.AlignmentDirectionalBottomEnd),
		p.NewAlignment(0.5, 0.5).Add(p.NewAlignmentDirectional(0.0, 0.25)),
		p.NewAlignmentDirectional(0.7, 0.5).Add(p.NewAlignment(0.0, 0.25)),
		p.NewAlignment(0.3, -0.7).Add(p.NewAlignmentDirectional(0.2, 0.0)),
		p.NewAlignment(-0.6, 0.3).Add(p.NewAlignmentDirectional(0.6, 0.0)),
		nil,
	}
}

func paintingAlignmentCases() map[string]any {
	out := map[string]any{}
	ds := dart.DoubleToString
	geoms := alignmentGeoms()
	ts := []float64{0.0, 0.25, 0.5, 1.0, -0.5, 1.5, 0.1, 1.0 / 3.0}
	scalars := []float64{2.0, 0.5, -3.0, 0.3, 1.0 / 3.0, 7.0, 0.0, -0.25}

	for i, g := range geoms {
		out[fmt.Sprintf("desc.%d", i)] = alignDesc(g)
		if g == nil {
			continue
		}
		out[fmt.Sprintf("resolve.%d.ltr", i)] = alignDesc(g.Resolve(&pLtr))
		out[fmt.Sprintf("resolve.%d.rtl", i)] = alignDesc(g.Resolve(&pRtl))
		if f, ok := g.(p.FractionalOffset); ok {
			// Dart's resolve returns the FractionalOffset itself (typed
			// Alignment); Go's Resolve returns an Alignment, the embedded
			// one (a documented divergence): check that, then describe f.
			if f.Resolve(&pLtr) == f.Alignment && f.Resolve(&pRtl) == f.Alignment && f.Resolve(nil) == f.Alignment {
				out[fmt.Sprintf("resolve.%d.ltr", i)] = alignDesc(f)
				out[fmt.Sprintf("resolve.%d.rtl", i)] = alignDesc(f)
			}
		}
		out[fmt.Sprintf("resolve.%d.null", i)] = pErr(func() { g.Resolve(nil) })
		out[fmt.Sprintf("neg.%d", i)] = alignDesc(p.AlignmentGeometryNeg(g))
		for k, s := range scalars {
			out[fmt.Sprintf("mul.%d.%d", i, k)] = alignDesc(p.AlignmentGeometryMul(g, s))
			out[fmt.Sprintf("div.%d.%d", i, k)] = alignDesc(p.AlignmentGeometryDiv(g, s))
			out[fmt.Sprintf("mod.%d.%d", i, k)] = alignDesc(p.AlignmentGeometryMod(g, s))
			key := fmt.Sprintf("truncDiv.%d.%d", i, k)
			out[key] = pErr(func() { out[key+".value"] = alignDesc(p.AlignmentGeometryTruncDiv(g, s)) })
		}
		if a, ok := g.(p.AlignmentLike); ok {
			al := a.AsAlignment()
			for _, o := range []ui.Offset{ui.NewOffset(10, 20), ui.NewOffset(-3.3, 7.7), ui.NewOffset(0.1, 1e6)} {
				out[fmt.Sprintf("alongOffset.%d.%v", i, o)] = pOff(al.AlongOffset(o))
			}
			rects := []ui.Rect{ui.NewRectFromLTWH(5, 6, 100, 50), ui.NewRectFromLTRB(-1.1, 2.2, 3.3, 4.4), ui.NewRectFromLTWH(0, 0, 1, 1)}
			for _, s := range []ui.Size{ui.NewSize(10, 20), ui.NewSize(3.3, 7.7), ui.SizeZero} {
				out[fmt.Sprintf("alongSize.%d.%v", i, s)] = pOff(al.AlongSize(s))
				for _, r := range rects {
					out[fmt.Sprintf("inscribe.%d.%v.%v", i, s, r)] = pRc(al.Inscribe(s, r))
				}
			}
			for _, r := range rects[:2] {
				out[fmt.Sprintf("withinRect.%d.%v", i, r)] = pOff(al.WithinRect(r))
			}
		}
	}

	for i, a := range geoms {
		for j, b := range geoms {
			al, aIsAl := a.(p.AlignmentLike)
			bl, bIsAl := b.(p.AlignmentLike)
			ad, aIsAd := a.(p.AlignmentDirectional)
			bd, bIsAd := b.(p.AlignmentDirectional)
			af, aIsF := a.(p.FractionalOffset)
			bf, bIsF := b.(p.FractionalOffset)
			for k, t := range ts {
				out[fmt.Sprintf("lerp.%d.%d.%d", i, j, k)] = alignDesc(p.AlignmentGeometryLerp(a, b, t))
				if (a == nil || aIsAl) && (b == nil || bIsAl) {
					out[fmt.Sprintf("alignmentLerp.%d.%d.%d", i, j, k)] = alignLikeDesc(p.AlignmentLerp(al, bl, t))
				}
				if (a == nil || aIsAd) && (b == nil || bIsAd) {
					var pa, pb *p.AlignmentDirectional
					if a != nil {
						pa = &ad
					}
					if b != nil {
						pb = &bd
					}
					r := p.AlignmentDirectionalLerp(pa, pb, t)
					if r == nil {
						out[fmt.Sprintf("directionalLerp.%d.%d.%d", i, j, k)] = "null"
					} else {
						out[fmt.Sprintf("directionalLerp.%d.%d.%d", i, j, k)] = alignDesc(*r)
					}
				}
				if (a == nil || aIsF) && (b == nil || bIsF) {
					var pa, pb *p.FractionalOffset
					if a != nil {
						pa = &af
					}
					if b != nil {
						pb = &bf
					}
					r := p.FractionalOffsetLerp(pa, pb, t)
					if r == nil {
						out[fmt.Sprintf("fractionalLerp.%d.%d.%d", i, j, k)] = "null"
					} else {
						out[fmt.Sprintf("fractionalLerp.%d.%d.%d", i, j, k)] = alignDesc(*r)
					}
				}
			}
			if a == nil || b == nil {
				continue
			}
			out[fmt.Sprintf("add.%d.%d", i, j)] = alignDesc(a.Add(b))
			out[fmt.Sprintf("eq.%d.%d", i, j)] = a.Equal(b)
			if aIsAl && bIsAl {
				if f, ok := a.(p.FractionalOffset); ok {
					out[fmt.Sprintf("plus.%d.%d", i, j)] = alignDesc(f.Plus(bl))
					out[fmt.Sprintf("minus.%d.%d", i, j)] = alignDesc(f.Sub(bl))
				} else {
					out[fmt.Sprintf("plus.%d.%d", i, j)] = alignDesc(al.AsAlignment().Plus(bl))
					out[fmt.Sprintf("minus.%d.%d", i, j)] = alignDesc(al.AsAlignment().Sub(bl))
				}
			}
			if aIsAd && bIsAd {
				out[fmt.Sprintf("plus.%d.%d", i, j)] = alignDesc(ad.Plus(bd))
				out[fmt.Sprintf("minus.%d.%d", i, j)] = alignDesc(ad.Sub(bd))
			}
		}
	}

	named := map[string]p.AlignmentGeometry{
		"Alignment.topLeft":                 p.AlignmentTopLeft,
		"Alignment.topCenter":               p.AlignmentTopCenter,
		"Alignment.topRight":                p.AlignmentTopRight,
		"Alignment.centerLeft":              p.AlignmentCenterLeft,
		"Alignment.center":                  p.AlignmentCenter,
		"Alignment.centerRight":             p.AlignmentCenterRight,
		"Alignment.bottomLeft":              p.AlignmentBottomLeft,
		"Alignment.bottomCenter":            p.AlignmentBottomCenter,
		"Alignment.bottomRight":             p.AlignmentBottomRight,
		"AlignmentDirectional.topStart":     p.AlignmentDirectionalTopStart,
		"AlignmentDirectional.topCenter":    p.AlignmentDirectionalTopCenter,
		"AlignmentDirectional.topEnd":       p.AlignmentDirectionalTopEnd,
		"AlignmentDirectional.centerStart":  p.AlignmentDirectionalCenterStart,
		"AlignmentDirectional.center":       p.AlignmentDirectionalCenter,
		"AlignmentDirectional.centerEnd":    p.AlignmentDirectionalCenterEnd,
		"AlignmentDirectional.bottomStart":  p.AlignmentDirectionalBottomStart,
		"AlignmentDirectional.bottomCenter": p.AlignmentDirectionalBottomCenter,
		"AlignmentDirectional.bottomEnd":    p.AlignmentDirectionalBottomEnd,
		"FractionalOffset.topLeft":          p.FractionalOffsetTopLeft,
		"FractionalOffset.topCenter":        p.FractionalOffsetTopCenter,
		"FractionalOffset.topRight":         p.FractionalOffsetTopRight,
		"FractionalOffset.centerLeft":       p.FractionalOffsetCenterLeft,
		"FractionalOffset.center":           p.FractionalOffsetCenter,
		"FractionalOffset.centerRight":      p.FractionalOffsetCenterRight,
		"FractionalOffset.bottomLeft":       p.FractionalOffsetBottomLeft,
		"FractionalOffset.bottomCenter":     p.FractionalOffsetBottomCenter,
		"FractionalOffset.bottomRight":      p.FractionalOffsetBottomRight,
		"AlignmentGeometry.topStart":        p.AlignmentGeometryTopStart,
		"AlignmentGeometry.bottomEnd":       p.AlignmentGeometryBottomEnd,
		"AlignmentGeometry.xy":              p.NewAlignmentGeometryXY(0.25, -4),
		"AlignmentGeometry.directional":     p.NewAlignmentGeometryDirectional(0.25, -4),
	}
	for k, v := range named {
		out["named."+k] = alignDesc(v)
	}

	for _, v := range [][2]float64{
		{math.Copysign(0, -1), -1.0}, {0.05, 0.15}, {0.25, 0.35}, {-0.05, -0.15}, {1e21, 123456.789},
		{math.Inf(1), math.Inf(-1)}, {math.NaN(), 0.0},
	} {
		out["toString."+ds(v[0])+"."+ds(v[1])] = []any{
			p.NewAlignment(v[0], v[1]).String(),
			p.NewAlignmentDirectional(v[0], v[1]).String(),
			p.NewFractionalOffset(v[0], v[1]).String(),
			p.NewAlignment(v[0], v[1]).Add(p.NewAlignmentDirectional(v[1], v[0])).String(),
		}
	}

	out["truncDiv.huge"] = []any{
		alignDesc(p.NewAlignment(1e300, -1e300).TruncDiv(1.0)),
		alignDesc(p.NewFractionalOffset(1e300, -0.3).TruncDiv(0.5)),
		alignDesc(p.NewAlignmentDirectional(-0.25, 0.5).TruncDiv(2.0)),
		alignDesc(p.NewAlignment(math.Copysign(0, -1), -0.75).TruncDiv(1.0)),
		pErr(func() { p.NewAlignment(math.Inf(1), 0.5).TruncDiv(2.0) }),
	}

	for _, o := range []ui.Offset{ui.NewOffset(100, 100), ui.NewOffset(-3.3, 7.1), ui.NewOffset(0.1, 0.2)} {
		for _, s := range []ui.Size{ui.NewSize(200, 400), ui.NewSize(3, 7), ui.NewSize(0.3, 0.7)} {
			out[fmt.Sprintf("fromOffsetAndSize.%v.%v", o, s)] = alignDesc(p.NewFractionalOffsetFromOffsetAndSize(o, s))
		}
		for _, r := range []ui.Rect{ui.NewRectFromLTWH(50, 20, 200, 400), ui.NewRectFromLTRB(-1.1, 2.2, 3.3, 4.4)} {
			out[fmt.Sprintf("fromOffsetAndRect.%v.%v", o, r)] = alignDesc(p.NewFractionalOffsetFromOffsetAndRect(o, r))
		}
	}

	out["textAlignVertical"] = []any{
		p.TextAlignVerticalTop.String(), p.TextAlignVerticalCenter.String(), p.TextAlignVerticalBottom.String(),
		p.NewTextAlignVertical(0.3).String(), p.NewTextAlignVertical(math.Copysign(0, -1)).String(),
	}

	sizes := []ui.Size{
		ui.NewSize(100, 1000), ui.NewSize(200, 2000), ui.NewSize(300, 3000), ui.NewSize(2000, 400),
		ui.NewSize(1000, 100), ui.NewSize(1000, 300), ui.NewSize(400, 2000), ui.NewSize(33.3, 77.7),
		ui.NewSize(0.7, 0.1), ui.NewSize(64, 64), ui.NewSize(0, 10), ui.NewSize(10, 0), ui.NewSize(-5, 10),
		ui.NewSize(1e-5, 1e5), ui.NewSize(100, 100), ui.NewSize(10, 100.5), ui.NewSize(3, 7), ui.NewSize(0.5, 7.7), ui.NewSize(4, 7.7),
	}
	for _, fit := range p.BoxFitValues {
		out[fmt.Sprintf("boxFit.name.%d", int(fit))] = fit.String()
		for i, a := range sizes {
			for j, b := range sizes {
				f := p.ApplyBoxFit(fit, a, b)
				out[fmt.Sprintf("boxFit.%d.%d.%d", int(fit), i, j)] = append(pSz(f.Source), pSz(f.Destination)...)
			}
		}
	}

	outer := []ui.Size{ui.NewSize(100, 100), ui.NewSize(800, 600), ui.NewSize(37.5, 300.3)}
	child := []ui.Size{ui.NewSize(20, 10), ui.NewSize(200, 10), ui.NewSize(50, 33.3), ui.NewSize(10, 590)}
	targets := []ui.Offset{ui.NewOffset(50, 50), ui.NewOffset(0, 50), ui.NewOffset(100, 5), ui.NewOffset(400.5, 590),
		ui.NewOffset(-10, 300), ui.NewOffset(18.7, 150.15)}
	for a, os := range outer {
		for b, cs := range child {
			for c, tg := range targets {
				for _, below := range []bool{false, true} {
					out[fmt.Sprintf("pdb.%d.%d.%d.%t", a, b, c, below)] = pOff(p.PositionDependentBox(os, cs, tg, below))
					for _, vo := range []float64{0.0, 24.0, 7.5} {
						for _, m := range []float64{0.0, 10.0, 20.0, 3.3} {
							vo, m := vo, m
							out[fmt.Sprintf("pdb.%d.%d.%d.%t.%s.%s", a, b, c, below, ds(vo), ds(m))] = pOff(p.PositionDependentBox(os, cs, tg, below,
								p.PositionDependentBoxOptions{VerticalOffset: &vo, Margin: &m}))
						}
					}
				}
			}
		}
	}

	var bt []any
	for _, a := range p.AxisValues {
		bt = append(bt, []any{a.String(), p.FlipAxis(a).String()})
	}
	for _, d := range p.AxisDirectionValues {
		bt = append(bt, []any{d.String(), p.AxisDirectionToAxis(d).String(), p.FlipAxisDirection(d).String(), p.AxisDirectionIsReversed(d)})
	}
	for _, d := range ui.TextDirectionValues {
		bt = append(bt, p.TextDirectionToAxisDirection(d).String())
	}
	for _, d := range p.VerticalDirectionValues {
		bt = append(bt, d.String())
	}
	for _, c := range p.RenderComparisonValues {
		bt = append(bt, c.String())
	}
	out["basicTypes"] = bt
	return out
}

// pNormalize makes Go's values comparable with decoded JSON (via a JSON
// round trip: float64s print as the shortest repr, as Dart's do).
func pNormalize(t *testing.T, v any) any {
	t.Helper()
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

func TestPaintingAlignmentMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_alignment.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string]any
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	got := pNormalize(t, paintingAlignmentCases()).(map[string]any)
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	bad := 0
	for _, k := range keys {
		g, ok := got[k]
		if !ok {
			t.Errorf("%s: missing in Go", k)
			bad++
		} else if !reflect.DeepEqual(g, want[k]) {
			t.Errorf("%s:\n flutter %v\n go      %v", k, want[k], g)
			bad++
		}
		if bad > 40 {
			t.Fatal("too many differences")
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s: not in the Flutter reference", k)
		}
	}
	t.Logf("%d cases", len(want))
}
