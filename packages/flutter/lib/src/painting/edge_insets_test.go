package painting

// Port of packages/flutter/test/painting/edge_insets_test.dart (Flutter
// 3.50.1). Dart's `==` across EdgeInsetsGeometry types is
// EdgeInsetsGeometryEqual; `identical` on const values is ==.
//
// Skipped:
//   - "EdgeInsetsGeometry factories": the fromViewPadding expectation
//     (dart:ui's ViewPadding is not ported); the other factories are
//     checked.
//   - "EdgeInsets control test": hasOneLineDescription and hashCode (Go
//     values have no hashCode; String() is checked for one line).

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

var (
	eiLTR = ui.TextDirectionLtr
	eiRTL = ui.TextDirectionRtl
)

func eqInsets(t *testing.T, got, want EdgeInsetsGeometry) {
	t.Helper()
	if !EdgeInsetsGeometryEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func neInsets(t *testing.T, got, want EdgeInsetsGeometry) {
	t.Helper()
	if EdgeInsetsGeometryEqual(got, want) {
		t.Errorf("got %v, want something else", got)
	}
}

func TestEdgeInsetsConstructors(t *testing.T) {
	l := NewEdgeInsetsFromLTRB(10, 20, 30, 40).Resolve(&eiLTR)
	if l != (EdgeInsets{10, 20, 30, 40}) {
		t.Error(l)
	}
	r := NewEdgeInsetsFromLTRB(10, 20, 30, 40).Resolve(&eiRTL)
	if r != (EdgeInsets{10, 20, 30, 40}) {
		t.Error(r)
	}
	all := NewEdgeInsetsAll(10)
	eqInsets(t, all.Resolve(&eiLTR), NewEdgeInsetsFromLTRB(10, 10, 10, 10))
	eqInsets(t, all.Resolve(&eiRTL), NewEdgeInsetsFromLTRB(10, 10, 10, 10))
	only := EdgeInsets{Left: 10, Top: 20, Right: 30, Bottom: 40}
	eqInsets(t, only.Resolve(&eiLTR), NewEdgeInsetsFromLTRB(10, 20, 30, 40))
	eqInsets(t, only.Resolve(&eiRTL), NewEdgeInsetsFromLTRB(10, 20, 30, 40))
	sym := NewEdgeInsetsSymmetric(EdgeInsetsSymmetricOptions{Horizontal: 10, Vertical: 20})
	eqInsets(t, sym.Resolve(&eiLTR), NewEdgeInsetsFromLTRB(10, 20, 10, 20))
	eqInsets(t, sym.Resolve(&eiRTL), NewEdgeInsetsFromLTRB(10, 20, 10, 20))
}

func TestEdgeInsetsDirectionalConstructors(t *testing.T) {
	l := NewEdgeInsetsDirectionalFromSTEB(10, 20, 30, 40).Resolve(&eiLTR)
	if l != (EdgeInsets{10, 20, 30, 40}) {
		t.Error(l)
	}
	r := NewEdgeInsetsDirectionalFromSTEB(10, 20, 30, 40).Resolve(&eiRTL)
	if r != (EdgeInsets{30, 20, 10, 40}) {
		t.Error(r)
	}
	all := NewEdgeInsetsDirectionalAll(10)
	eqInsets(t, all.Resolve(&eiLTR), NewEdgeInsetsFromLTRB(10, 10, 10, 10))
	eqInsets(t, all.Resolve(&eiRTL), NewEdgeInsetsFromLTRB(10, 10, 10, 10))
	d := EdgeInsetsDirectional{Start: 10, Top: 20, End: 30, Bottom: 40}
	eqInsets(t, d.Resolve(&eiLTR), NewEdgeInsetsFromLTRB(10, 20, 30, 40))
	eqInsets(t, d.Resolve(&eiRTL), NewEdgeInsetsFromLTRB(30, 20, 10, 40))
	sym := NewEdgeInsetsDirectionalSymmetric(EdgeInsetsSymmetricOptions{Horizontal: 10, Vertical: 20})
	eqInsets(t, sym.Resolve(&eiLTR), NewEdgeInsetsFromLTRB(10, 20, 10, 20))
	eqInsets(t, sym.Resolve(&eiRTL), NewEdgeInsetsFromLTRB(10, 20, 10, 20))
}

func TestEdgeInsetsControl(t *testing.T) {
	insets := NewEdgeInsetsFromLTRB(5.0, 7.0, 11.0, 13.0)
	if s := insets.String(); s == "" || strings.Contains(s, "\n") {
		t.Error("one line description", s)
	}
	if insets.TopLeft() != ui.NewOffset(5, 7) || insets.TopRight() != ui.NewOffset(-11, 7) ||
		insets.BottomLeft() != ui.NewOffset(5, -13) || insets.BottomRight() != ui.NewOffset(-11, -13) {
		t.Error("corners")
	}
	if insets.CollapsedSize() != ui.NewSize(16, 20) {
		t.Error("collapsedSize")
	}
	eqInsets(t, insets.Flipped(), NewEdgeInsetsFromLTRB(11, 13, 5, 7))
	if insets.Along(AxisHorizontal) != 16 || insets.Along(AxisVertical) != 20 {
		t.Error("along")
	}
	if insets.InflateRect(ui.NewRectFromLTRB(23, 32, 124, 143)) != ui.NewRectFromLTRB(18, 25, 135, 156) {
		t.Error("inflateRect")
	}
	if insets.DeflateRect(ui.NewRectFromLTRB(23, 32, 124, 143)) != ui.NewRectFromLTRB(28, 39, 113, 130) {
		t.Error("deflateRect")
	}
	if insets.InflateSize(ui.NewSize(100, 125)) != ui.NewSize(116, 145) ||
		insets.DeflateSize(ui.NewSize(100, 125)) != ui.NewSize(84, 105) {
		t.Error("inflate/deflateSize")
	}
	eqInsets(t, insets.Div(2), NewEdgeInsetsFromLTRB(2.5, 3.5, 5.5, 6.5))
	eqInsets(t, insets.TruncDiv(2), NewEdgeInsetsFromLTRB(2, 3, 5, 6))
	eqInsets(t, insets.Mod(5), NewEdgeInsetsFromLTRB(0, 2, 1, 3))
}

func TestEdgeInsetsLerp(t *testing.T) {
	a, b := NewEdgeInsetsAll(10), NewEdgeInsetsAll(20)
	got := *EdgeInsetsLerp(&a, &b, 0.25)
	eqInsets(t, got, a.Mul(1.25))
	eqInsets(t, got, b.Mul(0.625))
	eqInsets(t, got, a.Add(NewEdgeInsetsAll(2.5)))
	eqInsets(t, got, b.Sub(NewEdgeInsetsAll(7.5)))
	if EdgeInsetsLerp(nil, nil, 0.25) != nil {
		t.Error("null lerp")
	}
	eqInsets(t, *EdgeInsetsLerp(nil, &b, 0.25), b.Mul(0.25))
	eqInsets(t, *EdgeInsetsLerp(&a, nil, 0.25), a.Mul(0.75))
	// identical a, b
	if EdgeInsetsLerp(nil, nil, 0) != nil {
		t.Error("null lerp")
	}
	z := EdgeInsetsZero
	if EdgeInsetsLerp(&z, &z, 0.5) != &z {
		t.Error("identical")
	}
}

func TestEdgeInsetsResolve(t *testing.T) {
	for _, c := range []struct {
		in   EdgeInsetsDirectional
		dir  ui.TextDirection
		want EdgeInsets
	}{
		{EdgeInsetsDirectional{10, 20, 30, 40}, eiLTR, EdgeInsets{10, 20, 30, 40}},
		{EdgeInsetsDirectional{99, 98, 97, 96}, eiRTL, EdgeInsets{97, 98, 99, 96}},
		{NewEdgeInsetsDirectionalAll(50), eiLTR, NewEdgeInsetsAll(50)},
		{NewEdgeInsetsDirectionalAll(50), eiRTL, NewEdgeInsetsAll(50)},
		{EdgeInsetsDirectional{Start: 963.25}, eiLTR, EdgeInsets{963.25, 0, 0, 0}},
		{EdgeInsetsDirectional{Top: 963.25}, eiLTR, EdgeInsets{0, 963.25, 0, 0}},
		{EdgeInsetsDirectional{End: 963.25}, eiLTR, EdgeInsets{0, 0, 963.25, 0}},
		{EdgeInsetsDirectional{Bottom: 963.25}, eiLTR, EdgeInsets{0, 0, 0, 963.25}},
		{EdgeInsetsDirectional{Start: 963.25}, eiRTL, EdgeInsets{0, 0, 963.25, 0}},
		{EdgeInsetsDirectional{Top: 963.25}, eiRTL, EdgeInsets{0, 963.25, 0, 0}},
		{EdgeInsetsDirectional{End: 963.25}, eiRTL, EdgeInsets{963.25, 0, 0, 0}},
		{EdgeInsetsDirectional{Bottom: 963.25}, eiRTL, EdgeInsets{0, 0, 0, 963.25}},
	} {
		if got := c.in.Resolve(&c.dir); got != c.want {
			t.Errorf("%v.resolve(%v) = %v, want %v", c.in, c.dir, got, c.want)
		}
	}
	eqInsets(t, EdgeInsetsDirectional{}, EdgeInsetsDirectional{})
	neInsets(t, EdgeInsetsDirectional{Top: 1}, EdgeInsetsDirectional{Bottom: 1})
}

func TestEdgeInsetsEquality(t *testing.T) {
	five := 5.0
	eqInsets(t, EdgeInsetsDirectional{Top: five, Bottom: 7}, EdgeInsetsDirectional{Top: five, Bottom: 7})
	eqInsets(t, EdgeInsets{Top: five, Bottom: 7}, EdgeInsetsDirectional{Top: five, Bottom: 7})
	eqInsets(t, EdgeInsetsDirectional{Top: five, Bottom: 7}, EdgeInsets{Top: five, Bottom: 7})
	eqInsets(t, EdgeInsets{Top: five, Bottom: 7}, EdgeInsets{Top: five, Bottom: 7})
	eqInsets(t, EdgeInsetsDirectional{Start: five}, EdgeInsetsDirectional{Start: five})
	neInsets(t, EdgeInsets{Left: 5}, EdgeInsetsDirectional{Start: 5})
	neInsets(t, EdgeInsetsDirectional{Start: 5}, EdgeInsets{Left: 5})
	eqInsets(t, EdgeInsets{Left: five}, EdgeInsets{Left: five})
	eqInsets(t, EdgeInsetsDirectional{End: five}, EdgeInsetsDirectional{End: five})
	neInsets(t, EdgeInsets{Right: 5}, EdgeInsetsDirectional{End: 5})
	neInsets(t, EdgeInsetsDirectional{End: 5}, EdgeInsets{Right: 5})
	eqInsets(t, EdgeInsets{Right: five}, EdgeInsets{Right: five})
	eqInsets(t, EdgeInsetsDirectional{End: 5}.AddGeometry(EdgeInsets{Right: 5}),
		EdgeInsetsDirectional{End: 5}.AddGeometry(EdgeInsets{Right: 5}))
	neInsets(t, EdgeInsetsDirectional{End: 5}.AddGeometry(EdgeInsets{Right: 5}),
		EdgeInsetsDirectional{End: 5}.AddGeometry(EdgeInsets{Left: 5}))
	eqInsets(t, EdgeInsetsDirectional{Top: 1}.AddGeometry(EdgeInsets{Top: 2}), EdgeInsetsDirectional{Top: 3})
	eqInsets(t, EdgeInsetsDirectional{Top: 1}.AddGeometry(EdgeInsets{Top: 2}), EdgeInsets{Top: 3})

	// Not in Flutter's test: each of the six components alone, both ways.
	base := EdgeInsetsDirectional{Start: 1, Top: 1, End: 1, Bottom: 1}.AddGeometry(NewEdgeInsetsAll(1))
	for _, d := range []EdgeInsetsGeometry{
		EdgeInsets{Left: 1}, EdgeInsets{Right: 1}, EdgeInsets{Top: 1}, EdgeInsets{Bottom: 1},
		EdgeInsetsDirectional{Start: 1}, EdgeInsetsDirectional{End: 1},
	} {
		neInsets(t, base, base.AddGeometry(d))
		neInsets(t, base.AddGeometry(d), base)
		eqInsets(t, base.AddGeometry(d), base.AddGeometry(d))
	}
	if EdgeInsetsGeometryEqual(nil, base) || EdgeInsetsGeometryEqual(base, nil) || !EdgeInsetsGeometryEqual(nil, nil) {
		t.Error("nil equality")
	}
}

func TestEdgeInsetsCopyWith(t *testing.T) {
	src := EdgeInsets{Left: 1, Top: 2, Bottom: 3, Right: 4}
	l, tp := 5.0, 6.0
	if got := src.CopyWith(EdgeInsetsCopyWithOptions{Left: &l, Top: &tp}); got != (EdgeInsets{Left: 5, Top: 6, Bottom: 3, Right: 4}) {
		t.Error(got)
	}
	d := EdgeInsetsDirectional{Start: 1, Top: 2, Bottom: 3, End: 4}
	if got := d.CopyWith(EdgeInsetsDirectionalCopyWithOptions{Start: &l, Top: &tp}); got != (EdgeInsetsDirectional{Start: 5, Top: 6, Bottom: 3, End: 4}) {
		t.Error(got)
	}
}

func TestEdgeInsetsGeometryLerp(t *testing.T) {
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{End: 10}, nil, 0.5), EdgeInsetsDirectional{End: 5})
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{Start: 10}, nil, 0.5), EdgeInsetsDirectional{Start: 5})
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{Top: 10}, nil, 0.5), EdgeInsetsDirectional{Top: 5})
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{Bottom: 10}, nil, 0.5), EdgeInsetsDirectional{Bottom: 5})
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{Bottom: 10}, EdgeInsetsDirectionalZero, 0.5), EdgeInsetsDirectional{Bottom: 5})
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{Bottom: 10}, EdgeInsetsZero, 0.5), EdgeInsetsDirectional{Bottom: 5})
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{Start: 10}, EdgeInsets{Left: 20}, 0.5),
		EdgeInsetsDirectional{Start: 5}.AddGeometry(EdgeInsets{Left: 10}))
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsetsDirectional{Bottom: 1},
		EdgeInsetsDirectional{Start: 1, Bottom: 1}.AddGeometry(EdgeInsets{Right: 2}), 0.5),
		EdgeInsetsDirectional{Start: 0.5}.AddGeometry(EdgeInsets{Right: 1, Bottom: 1}))
	eqInsets(t, EdgeInsetsGeometryLerp(EdgeInsets{Bottom: 1},
		EdgeInsetsDirectional{End: 1, Bottom: 1}.AddGeometry(EdgeInsets{Right: 2}), 0.5),
		EdgeInsetsDirectional{End: 0.5}.AddGeometry(EdgeInsets{Right: 1, Bottom: 1}))

	// identical a, b
	if EdgeInsetsGeometryLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	var z EdgeInsetsGeometry = EdgeInsetsZero
	if EdgeInsetsGeometryLerp(z, z, 0.5) != z {
		t.Error("identical")
	}
	dz := EdgeInsetsDirectionalZero
	if EdgeInsetsDirectionalLerp(nil, nil, 0) != nil || EdgeInsetsDirectionalLerp(&dz, &dz, 0.5) != &dz {
		t.Error("directional identical")
	}
}

func TestEdgeInsetsGeometryLerpKinds(t *testing.T) {
	check := func(a, b EdgeInsetsGeometry, plus, minus EdgeInsetsGeometry) {
		t.Helper()
		got := EdgeInsetsGeometryLerp(a, b, 0.25)
		eqInsets(t, got, a.MulGeometry(1.25))
		eqInsets(t, got, b.MulGeometry(0.625))
		if plus != nil {
			eqInsets(t, got, a.AddGeometry(plus))
			eqInsets(t, got, b.Subtract(minus))
		}
		if EdgeInsetsGeometryLerp(nil, nil, 0.25) != nil {
			t.Error("null")
		}
		eqInsets(t, EdgeInsetsGeometryLerp(nil, b, 0.25), b.MulGeometry(0.25))
		eqInsets(t, EdgeInsetsGeometryLerp(a, nil, 0.25), a.MulGeometry(0.75))
	}
	// normal
	check(NewEdgeInsetsAll(10), NewEdgeInsetsAll(20), NewEdgeInsetsAll(2.5), NewEdgeInsetsAll(7.5))
	// directional
	check(EdgeInsetsDirectional{Start: 10, End: 10}, EdgeInsetsDirectional{Start: 20, End: 20},
		EdgeInsetsDirectional{Start: 2.5, End: 2.5}, EdgeInsetsDirectional{Start: 7.5, End: 7.5})
	// mixed
	check(EdgeInsetsDirectional{Start: 10, End: 10}.AddGeometry(NewEdgeInsetsAll(1)),
		EdgeInsetsDirectional{Start: 20, End: 20}.AddGeometry(NewEdgeInsetsAll(2)), nil, nil)
}

func TestEdgeInsetsOperators(t *testing.T) {
	a := NewEdgeInsetsFromLTRB(1, 2, 3, 5)
	eqInsets(t, a.Mul(2), NewEdgeInsetsFromLTRB(2, 4, 6, 10))
	eqInsets(t, a.Div(2), NewEdgeInsetsFromLTRB(0.5, 1, 1.5, 2.5))
	eqInsets(t, a.Mod(2), NewEdgeInsetsFromLTRB(1, 0, 1, 1))
	eqInsets(t, a.TruncDiv(2), NewEdgeInsetsFromLTRB(0, 1, 1, 2))
	eqInsets(t, a.Add(a), a.Mul(2))
	eqInsets(t, a.Sub(a), EdgeInsetsZero)
	eqInsets(t, a.AddGeometry(a), a.Mul(2))
	eqInsets(t, a.Subtract(a), EdgeInsetsZero)

	d := NewEdgeInsetsDirectionalFromSTEB(1, 2, 3, 5)
	eqInsets(t, d.Mul(2), NewEdgeInsetsDirectionalFromSTEB(2, 4, 6, 10))
	eqInsets(t, d.Div(2), NewEdgeInsetsDirectionalFromSTEB(0.5, 1, 1.5, 2.5))
	eqInsets(t, d.Mod(2), NewEdgeInsetsDirectionalFromSTEB(1, 0, 1, 1))
	eqInsets(t, d.TruncDiv(2), NewEdgeInsetsDirectionalFromSTEB(0, 1, 1, 2))
	eqInsets(t, d.Add(d), d.Mul(2))
	eqInsets(t, d.Sub(d), EdgeInsetsDirectionalZero)
	eqInsets(t, d.AddGeometry(d), d.Mul(2))
	eqInsets(t, d.Subtract(d), EdgeInsetsDirectionalZero)

	g := NewEdgeInsetsDirectionalFromSTEB(1, 2, 3, 5).AddGeometry(EdgeInsetsZero)
	if _, ok := g.(EdgeInsetsDirectional); ok {
		t.Error("mixed should not be directional")
	}
	eqInsets(t, g.MulGeometry(2), NewEdgeInsetsDirectionalFromSTEB(2, 4, 6, 10))
	eqInsets(t, g.DivGeometry(2), NewEdgeInsetsDirectionalFromSTEB(0.5, 1, 1.5, 2.5))
	eqInsets(t, g.ModGeometry(2), NewEdgeInsetsDirectionalFromSTEB(1, 0, 1, 1))
	eqInsets(t, g.TruncDivGeometry(2), NewEdgeInsetsDirectionalFromSTEB(0, 1, 1, 2))
	eqInsets(t, g.AddGeometry(g), g.MulGeometry(2))
	eqInsets(t, g.Subtract(g), EdgeInsetsDirectionalZero)
	eqInsets(t, g.Subtract(g), EdgeInsetsZero)
}

func TestEdgeInsetsGeometryToString(t *testing.T) {
	for _, c := range []struct {
		g    EdgeInsetsGeometry
		want string
	}{
		{EdgeInsetsZero, "EdgeInsets.zero"},
		{EdgeInsets{Top: 1.01, Left: 1.01, Right: 1.01, Bottom: 1.01}, "EdgeInsets.all(1.0)"},
		{EdgeInsetsDirectional{Start: 1.01, End: 1.01, Top: 1.01, Bottom: 1.01}, "EdgeInsetsDirectional(1.0, 1.0, 1.0, 1.0)"},
		{EdgeInsetsDirectional{Start: 4}.AddGeometry(EdgeInsets{Top: 3}), "EdgeInsetsDirectional(4.0, 3.0, 0.0, 0.0)"},
		{EdgeInsetsDirectional{Top: 4}.AddGeometry(EdgeInsets{Right: 3}), "EdgeInsets(0.0, 4.0, 3.0, 0.0)"},
		{EdgeInsetsDirectional{Start: 4}.AddGeometry(EdgeInsets{Left: 3}), "EdgeInsets(3.0, 0.0, 0.0, 0.0) + EdgeInsetsDirectional(4.0, 0.0, 0.0, 0.0)"},
	} {
		if got := c.g.String(); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestEdgeInsetsGeometryFactories(t *testing.T) {
	eqInsets(t, NewEdgeInsetsGeometryAll(10), NewEdgeInsetsAll(10))
	eqInsets(t, NewEdgeInsetsGeometryOnly(EdgeInsets{Left: 10, Top: 20, Right: 30, Bottom: 40}),
		EdgeInsets{Left: 10, Top: 20, Right: 30, Bottom: 40})
	eqInsets(t, NewEdgeInsetsGeometryDirectional(EdgeInsetsDirectional{Start: 10, Top: 20, End: 30, Bottom: 40}),
		EdgeInsetsDirectional{Start: 10, Top: 20, End: 30, Bottom: 40})
	eqInsets(t, NewEdgeInsetsGeometrySymmetric(EdgeInsetsSymmetricOptions{Horizontal: 10, Vertical: 20}),
		NewEdgeInsetsSymmetric(EdgeInsetsSymmetricOptions{Horizontal: 10, Vertical: 20}))
	eqInsets(t, NewEdgeInsetsGeometryFromLTRB(10, 20, 30, 40), NewEdgeInsetsFromLTRB(10, 20, 30, 40))
	eqInsets(t, NewEdgeInsetsGeometryFromSTEB(10, 20, 20, 40), NewEdgeInsetsDirectionalFromSTEB(10, 20, 20, 40))
	eqInsets(t, EdgeInsetsGeometryZero, EdgeInsetsZero)
}

func expectNoTextDirection(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		e, ok := recover().(*foundation.FlutterError)
		if !ok {
			t.Fatal("want a FlutterError")
		}
		m := e.Message()
		if !strings.Contains(m, "No TextDirection found.") || !strings.Contains(m, "without a Directionality ancestor") {
			t.Errorf("message %q", m)
		}
	}()
	f()
}

func TestEdgeInsetsResolveNullDirection(t *testing.T) {
	expectNoTextDirection(t, func() { NewEdgeInsetsDirectionalAll(10).Resolve(nil) })
	a := EdgeInsets{Top: 5, Left: 5}
	b := EdgeInsetsDirectional{Top: 15, Start: 15}
	expectNoTextDirection(t, func() { a.AddGeometry(b).Resolve(nil) })
	expectNoTextDirection(t, func() { b.Subtract(a).Resolve(nil) })
}

func TestEdgeInsetsRRect(t *testing.T) {
	original := ui.NewRRectFromLTRBAndCorners(10, 20, 30, 60, ui.RRectCorners{
		TopLeft: ui.NewRadiusCircular(5), TopRight: ui.NewRadiusCircular(6),
		BottomRight: ui.NewRadiusCircular(7), BottomLeft: ui.NewRadiusCircular(8)})
	insets := NewEdgeInsetsFromLTRB(2, 3, 4, 5)
	inf := insets.InflateRRect(original)
	if inf.Left != original.Left-insets.Left || inf.Top != original.Top-insets.Top ||
		inf.Right != original.Right+insets.Right || inf.Bottom != original.Bottom+insets.Bottom ||
		inf.TlRadiusX != original.TlRadiusX+insets.Left || inf.TlRadiusY != original.TlRadiusY+insets.Top ||
		inf.TrRadiusX != original.TrRadiusX+insets.Right || inf.TrRadiusY != original.TrRadiusY+insets.Top ||
		inf.BrRadiusX != original.BrRadiusX+insets.Right || inf.BrRadiusY != original.BrRadiusY+insets.Bottom ||
		inf.BlRadiusX != original.BlRadiusX+insets.Left || inf.BlRadiusY != original.BlRadiusY+insets.Bottom {
		t.Error("inflateRRect", inf)
	}
	original2 := ui.NewRRectFromLTRBAndCorners(10, 20, 30, 60, ui.RRectCorners{
		TopLeft: ui.NewRadiusCircular(8), TopRight: ui.NewRadiusCircular(9),
		BottomRight: ui.NewRadiusCircular(10), BottomLeft: ui.NewRadiusCircular(11)})
	def := insets.DeflateRRect(original2)
	if def.Left != original2.Left+insets.Left || def.Top != original2.Top+insets.Top ||
		def.Right != original2.Right-insets.Right || def.Bottom != original2.Bottom-insets.Bottom ||
		def.TlRadiusX != original2.TlRadiusX-insets.Left || def.TlRadiusY != original2.TlRadiusY-insets.Top ||
		def.TrRadiusX != original2.TrRadiusX-insets.Right || def.TrRadiusY != original2.TrRadiusY-insets.Top ||
		def.BrRadiusX != original2.BrRadiusX-insets.Right || def.BrRadiusY != original2.BrRadiusY-insets.Bottom ||
		def.BlRadiusX != original2.BlRadiusX-insets.Left || def.BlRadiusY != original2.BlRadiusY-insets.Bottom {
		t.Error("deflateRRect", def)
	}
	one := ui.NewRRectFromLTRBAndCorners(10, 20, 30, 60, ui.RRectCorners{TopLeft: ui.NewRadiusCircular(5)})
	if EdgeInsetsZero.InflateRRect(one) != one || EdgeInsetsZero.DeflateRRect(one) != one {
		t.Error("zero insets")
	}
	three := ui.NewRRectFromLTRBAndCorners(10, 20, 30, 60, ui.RRectCorners{
		TopLeft: ui.NewRadiusCircular(3), TopRight: ui.NewRadiusCircular(3),
		BottomRight: ui.NewRadiusCircular(3), BottomLeft: ui.NewRadiusCircular(3)})
	d := NewEdgeInsetsAll(5).DeflateRRect(three)
	if d.TlRadiusX != 0 || d.TlRadiusY != 0 || d.TrRadiusX != 0 || d.TrRadiusY != 0 ||
		d.BrRadiusX != 0 || d.BrRadiusY != 0 || d.BlRadiusX != 0 || d.BlRadiusY != 0 {
		t.Error("clamp to zero", d)
	}
	near := func(a, b float64) bool { return a-b < 0.001 && b-a < 0.001 }
	for _, r := range []ui.RRect{insets.DeflateRRect(insets.InflateRRect(original)), insets.InflateRRect(insets.DeflateRRect(original))} {
		if !near(r.Left, original.Left) || !near(r.Top, original.Top) || !near(r.Right, original.Right) || !near(r.Bottom, original.Bottom) {
			t.Error("inverse", r)
		}
	}
}
