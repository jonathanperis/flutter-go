package difftest

// Go side of the dart:ui Path differential test: the same cases as
// flutter/test/path_test.dart (both read path_cases.json), printed the
// same way and compared exactly with Flutter's build/path.json.

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type pathSpec struct {
	Paths map[string]struct {
		Ops   [][]any `json:"ops"`
		Grids [][]any `json:"grids"`
		Skip  []string
	} `json:"paths"`
	Matrices     map[string][]any `json:"matrices"`
	Transforms   []string         `json:"transforms"`
	DefaultGrids [][]any          `json:"defaultGrids"`
}

func specNum(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		switch x {
		case "inf":
			return math.Inf(1)
		case "-inf":
			return math.Inf(-1)
		case "neg0":
			return math.Copysign(0, -1)
		}
		return math.NaN()
	}
	panic("bad number")
}

func (s *pathSpec) matrix(name any) []float64 {
	if name == nil {
		return nil
	}
	var m []float64
	for _, v := range s.Matrices[name.(string)] {
		m = append(m, specNum(v))
	}
	return m
}

func (s *pathSpec) build(name string) *Path {
	p := NewPath()
	for _, op := range s.Paths[name].Ops {
		a := func(i int) float64 { return specNum(op[i]) }
		flag := func(i int) bool { return op[i].(float64) == 1 }
		ell := func(i int) Radius { return NewRadiusElliptical(a(i), a(i+1)) }
		switch op[0].(string) {
		case "M":
			p.MoveTo(a(1), a(2))
		case "m":
			p.RelativeMoveTo(a(1), a(2))
		case "L":
			p.LineTo(a(1), a(2))
		case "l":
			p.RelativeLineTo(a(1), a(2))
		case "Q":
			p.QuadraticBezierTo(a(1), a(2), a(3), a(4))
		case "q":
			p.RelativeQuadraticBezierTo(a(1), a(2), a(3), a(4))
		case "C":
			p.CubicTo(a(1), a(2), a(3), a(4), a(5), a(6))
		case "c":
			p.RelativeCubicTo(a(1), a(2), a(3), a(4), a(5), a(6))
		case "K":
			p.ConicTo(a(1), a(2), a(3), a(4), a(5))
		case "k":
			p.RelativeConicTo(a(1), a(2), a(3), a(4), a(5))
		case "arcTo":
			p.ArcTo(NewRectFromLTRB(a(1), a(2), a(3), a(4)), a(5), a(6), flag(7))
		case "arcToPoint":
			cw := flag(7)
			p.ArcToPoint(NewOffset(a(1), a(2)), ArcToPointOptions{Radius: ell(3), Rotation: a(5), LargeArc: flag(6), Clockwise: &cw})
		case "relArcToPoint":
			cw := flag(7)
			p.RelativeArcToPoint(NewOffset(a(1), a(2)), ArcToPointOptions{Radius: ell(3), Rotation: a(5), LargeArc: flag(6), Clockwise: &cw})
		case "rect":
			p.AddRect(NewRectFromLTRB(a(1), a(2), a(3), a(4)))
		case "oval":
			p.AddOval(NewRectFromLTRB(a(1), a(2), a(3), a(4)))
		case "arc":
			p.AddArc(NewRectFromLTRB(a(1), a(2), a(3), a(4)), a(5), a(6))
		case "poly":
			var pts []Offset
			for i := 2; i+1 < len(op); i += 2 {
				pts = append(pts, NewOffset(a(i), a(i+1)))
			}
			p.AddPolygon(pts, flag(1))
		case "rrect":
			p.AddRRect(NewRRectFromLTRBAndCorners(a(1), a(2), a(3), a(4),
				RRectCorners{TopLeft: ell(5), TopRight: ell(7), BottomRight: ell(9), BottomLeft: ell(11)}))
		case "rse":
			p.AddRSuperellipse(NewRSuperellipseFromLTRBAndCorners(a(1), a(2), a(3), a(4),
				RRectCorners{TopLeft: ell(5), TopRight: ell(7), BottomRight: ell(9), BottomLeft: ell(11)}))
		case "Z":
			p.Close()
		case "fill":
			p.SetFillType(PathFillType(int(a(1))))
		case "reset":
			p.Reset()
		case "addPath":
			p.AddPath(s.build(op[1].(string)), NewOffset(a(2), a(3)), AddPathOptions{Matrix4: s.matrix(op[4])})
		case "extend":
			p.ExtendWithPath(s.build(op[1].(string)), NewOffset(a(2), a(3)), AddPathOptions{Matrix4: s.matrix(op[4])})
		default:
			panic("unknown op " + op[0].(string))
		}
	}
	return p
}

func dd(v float64) string { return dart.DoubleToString(v) }
func rectStr(r Rect) string {
	return dd(r.Left) + "," + dd(r.Top) + "," + dd(r.Right) + "," + dd(r.Bottom)
}
func tangentStr(t *Tangent, angle bool) string {
	if t == nil {
		return "null"
	}
	s := dd(t.Position.Dx) + "," + dd(t.Position.Dy) + "," + dd(t.Vector.Dx) + "," + dd(t.Vector.Dy)
	if angle {
		s += "," + dd(t.Angle())
	}
	return s
}

func gridStr(p *Path, g []any) string {
	x0, y0, step := specNum(g[0]), specNum(g[1]), specNum(g[2])
	nx, ny := int(specNum(g[3])), int(specNum(g[4]))
	rows := make([]string, ny)
	for j := 0; j < ny; j++ {
		var b strings.Builder
		for i := 0; i < nx; i++ {
			// x0 + i * step, rounded per step as Dart does (no fused multiply-add).
			if p.Contains(NewOffset(x0+float64(float64(i)*step), y0+float64(float64(j)*step))) {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		rows[j] = b.String()
	}
	return strings.Join(rows, "|")
}

// probesStr mirrors path_test.dart's probes: contains just off the outline.
func probesStr(p, q *Path) string {
	eps := []float64{0, 1e-4, -1e-4, 1.0 / 4096, -1.0 / 4096, 2.5e-4, -2.5e-4, 1e-3, -1e-3}
	var out []string
	for m := range p.ComputeMetrics().All() {
		var b strings.Builder
		for k := 0; k < 64; k++ {
			t := m.GetTangentForOffset(float64(m.Length*float64(k)) / 64)
			if t == nil {
				b.WriteByte('n')
				continue
			}
			for _, e := range eps {
				x := t.Position.Dx - float64(t.Vector.Dy*e)
				y := t.Position.Dy + float64(t.Vector.Dx*e)
				if q.Contains(NewOffset(x, y)) {
					b.WriteByte('1')
				} else {
					b.WriteByte('0')
				}
			}
		}
		out = append(out, b.String())
	}
	return strings.Join(out, "|")
}

func metricsStr(p *Path, forceClosed bool) string {
	var out []string
	for m := range p.ComputeMetrics(ComputeMetricsOptions{ForceClosed: forceClosed}).All() {
		l := m.Length
		out = append(out, dd(l)+","+strconv.FormatBool(m.IsClosed)+","+strconv.Itoa(m.ContourIndex))
		ds := []float64{-1, 0, 1e-3}
		for k := 0; k <= 8; k++ {
			ds = append(ds, float64(l*float64(k))/8)
		}
		ds = append(ds, l+1, math.NaN())
		var ts []string
		for _, x := range ds {
			ts = append(ts, tangentStr(m.GetTangentForOffset(x), true))
		}
		out = append(out, strings.Join(ts, ";"))
		se := [][2]float64{{0, l / 3}, {l / 4, float64(l*3) / 4}, {l / 2, l}, {-5, l + 5}, {l / 2, l / 2}, {3, 2},
			{float64(l * 0.37), float64(l * 0.38)}}
		for _, s := range se {
			for _, mv := range []bool{true, false} {
				q := m.ExtractPath(s[0], s[1], ExtractPathOptions{StartWithMoveTo: &mv})
				parts := []string{rectStr(q.GetBounds())}
				for qm := range q.ComputeMetrics().All() {
					parts = append(parts, dd(qm.Length)+","+strconv.FormatBool(qm.IsClosed)+","+
						tangentStr(qm.GetTangentForOffset(qm.Length/2), false))
				}
				out = append(out, strings.Join(parts, ";"))
			}
		}
	}
	return strings.Join(out, "|")
}

func summaryStr(p *Path) string {
	out := []string{rectStr(p.GetBounds())}
	for m := range p.ComputeMetrics().All() {
		out = append(out, dd(m.Length)+","+strconv.FormatBool(m.IsClosed))
		var ts []string
		for k := 0; k <= 4; k++ {
			ts = append(ts, tangentStr(m.GetTangentForOffset(float64(m.Length*float64(k))/4), true))
		}
		out = append(out, strings.Join(ts, ";"))
	}
	return strings.Join(out, "|")
}

func loadPathSpec(t testing.TB) *pathSpec {
	data, err := os.ReadFile("path_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var s pathSpec
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	return &s
}

// pathCases mirrors path_test.dart's main.
func pathCases(s *pathSpec) map[string]string {
	out := map[string]string{}
	for name, ps := range s.Paths {
		p := s.build(name)
		out[name+".bounds"] = rectStr(p.GetBounds())
		out[name+".fillType"] = p.FillType().String()
		grids := ps.Grids
		if grids == nil {
			grids = s.DefaultGrids
		}
		for i, g := range grids {
			nz := NewPathFrom(p)
			nz.SetFillType(PathFillTypeNonZero)
			eo := NewPathFrom(p)
			eo.SetFillType(PathFillTypeEvenOdd)
			out[name+".grid"+strconv.Itoa(i)+".nz"] = gridStr(nz, g)
			out[name+".grid"+strconv.Itoa(i)+".eo"] = gridStr(eo, g)
		}
		nzp, eop := NewPathFrom(p), NewPathFrom(p)
		nzp.SetFillType(PathFillTypeNonZero)
		eop.SetFillType(PathFillTypeEvenOdd)
		out[name+".probes.nz"] = probesStr(p, nzp)
		out[name+".probes.eo"] = probesStr(p, eop)
		out[name+".grid.own"] = gridStr(p, grids[0])
		out[name+".metrics.open"] = metricsStr(p, false)
		out[name+".metrics.closed"] = metricsStr(p, true)
		out[name+".metrics.toString"] = p.ComputeMetrics().String()
	transforms:
		for _, t := range s.Transforms {
			for _, sk := range ps.Skip {
				if sk == t {
					continue transforms
				}
			}
			out[name+".transform."+t] = summaryStr(p.Transform(s.matrix(t)))
		}
		out[name+".shift"] = summaryStr(p.Shift(NewOffset(3.25, -1.5)))
	}
	return out
}

func TestPathMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/path.json")
	if err != nil {
		t.Skipf("no reference (%v); run flutter test test/path_test.dart in flutter/", err)
	}
	var ref map[string]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	got := pathCases(loadPathSpec(t))
	if len(ref) != len(got) {
		t.Errorf("reference has %d cases, Go has %d", len(ref), len(got))
	}
	names := make([]string, 0, len(got))
	for name := range got {
		names = append(names, name)
	}
	sort.Strings(names)
	failures := 0
	for _, name := range names {
		want, ok := ref[name]
		if !ok {
			t.Errorf("%s: missing from the reference", name)
			continue
		}
		if g := got[name]; g != want {
			failures++
			if failures <= 30 {
				t.Errorf("%s:\n Go      %s\n Flutter %s", name, firstDiff(g, want), firstDiff(want, g))
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d of %d cases differ", failures, len(got))
	}
}

// firstDiff shows a's text around its first difference from b.
func firstDiff(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	lo, hi := max(0, i-80), min(len(a), i+80)
	return "…" + a[lo:hi] + "…"
}
