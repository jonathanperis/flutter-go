package ui

// Port of engine/src/flutter/testing/dart/geometry_test.dart (Flutter 3.50.1).

import (
	"math"
	"testing"
)

func check[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func close12(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Errorf("%s = %v, want %v ± 1e-12", name, got, want)
	}
}

func isNaN(t *testing.T, name string, v float64) {
	t.Helper()
	if !math.IsNaN(v) {
		t.Errorf("%s = %v, want NaN", name, v)
	}
}

func TestOffsetBaseComparisons(t *testing.T) {
	o, nan := Offset{0, 0}, math.NaN()
	ge := map[Offset]bool{{0, -1}: true, {-1, 0}: true, {-1, -1}: true, {0, 0}: true, {10, -10}: false}
	for b, want := range ge {
		check(t, "o >= "+b.String(), o.GreaterEq(b), want)
	}
	check(t, ">= NaN y", o.GreaterEq(Offset{0, nan}), false)
	check(t, ">= NaN x", o.GreaterEq(Offset{nan, 0}), false)
	le := map[Offset]bool{{0, 1}: true, {1, 0}: true, {0, 0}: true, {10, -10}: false}
	for b, want := range le {
		check(t, "o <= "+b.String(), o.LessEq(b), want)
	}
	check(t, "<= NaN y", o.LessEq(Offset{0, nan}), false)
	check(t, "<= NaN x", o.LessEq(Offset{nan, 0}), false)
	check(t, "> (-1,-1)", o.Greater(Offset{-1, -1}), true)
	check(t, "> (0,-1)", o.Greater(Offset{0, -1}), false)
	check(t, "> (-1,0)", o.Greater(Offset{-1, 0}), false)
	check(t, "> (NaN,-1)", o.Greater(Offset{nan, -1}), false)
	check(t, "< (1,1)", o.Less(Offset{1, 1}), true)
	check(t, "< (0,1)", o.Less(Offset{0, 1}), false)
	check(t, "< (1,0)", o.Less(Offset{1, 0}), false)
	check(t, "< (NaN,1)", o.Less(Offset{nan, 1}), false)
	check(t, "==", o == Offset{0, 0}, true)
	check(t, "!= (1,0)", o == Offset{1, 0}, false)
	check(t, "!= (0,1)", o == Offset{0, 1}, false)
}

func TestOffsetDirection(t *testing.T) {
	pi := math.Pi
	for o, want := range map[Offset]float64{{0, 0}: 0, {0, 1}: pi / 2, {0, -1}: -pi / 2, {1, 0}: 0, {1, 1}: pi / 4,
		{1, -1}: -pi / 4, {-1, 0}: pi, {-1, 1}: pi * 3 / 4, {-1, -1}: -pi * 3 / 4} {
		check(t, o.String()+".direction", o.Direction(), want)
	}
}

func TestOffsetFromDirection(t *testing.T) {
	pi, r := math.Pi, 1/math.Sqrt(2)
	check(t, "(0,0)", NewOffsetFromDirection(0, 0), Offset{0, 0})
	close12(t, "pi/2 dx", NewOffsetFromDirection(pi/2).Dx, 0)
	check(t, "pi/2 dy", NewOffsetFromDirection(pi/2).Dy, 1.0)
	close12(t, "-pi/2 dx", NewOffsetFromDirection(-pi/2).Dx, 0)
	check(t, "-pi/2 dy", NewOffsetFromDirection(-pi/2).Dy, -1.0)
	check(t, "0", NewOffsetFromDirection(0), Offset{1, 0})
	for _, c := range [][3]float64{{pi / 4, r, r}, {-pi / 4, r, -r}, {pi * 3 / 4, -r, r}, {-pi * 3 / 4, -r, -r}} {
		o := NewOffsetFromDirection(c[0])
		close12(t, "dx", o.Dx, c[1])
		close12(t, "dy", o.Dy, c[2])
	}
	check(t, "pi dx", NewOffsetFromDirection(pi).Dx, -1.0)
	close12(t, "pi dy", NewOffsetFromDirection(pi).Dy, 0)
	check(t, "0, 2", NewOffsetFromDirection(0, 2), Offset{2, 0})
	close12(t, "pi/6 dx", NewOffsetFromDirection(pi/6, 2).Dx, math.Sqrt(3))
	close12(t, "pi/6 dy", NewOffsetFromDirection(pi/6, 2).Dy, 1)
}

func TestSizeFromDoubles(t *testing.T) {
	s := Size{5, 7}
	check(t, "width", s.Width, 5.0)
	check(t, "height", s.Height, 7.0)
	check(t, "shortestSide", s.ShortestSide(), 5.0)
	check(t, "longestSide", s.LongestSide(), 7.0)
}

func TestSizeAspectRatio(t *testing.T) {
	nz, inf := math.Copysign(0, -1), math.Inf(1)
	for _, c := range [][3]float64{{0, 0, 0}, {nz, 0, 0}, {0, nz, 0}, {nz, nz, 0}, {0, 1, 0}, {0, -1, nz}, {1, 0, inf},
		{1, 1, 1}, {1, -1, -1}, {-1, 0, -inf}, {-1, 1, -1}, {-1, -1, 1}, {3, 4, 3.0 / 4.0}} {
		check(t, Size{c[0], c[1]}.String()+".aspectRatio", Size{c[0], c[1]}.AspectRatio(), c[2])
	}
}

func TestRectBasics(t *testing.T) {
	r := NewRectFromLTRB(1, 3, 5, 7)
	check(t, "toString", r.String(), "Rect.fromLTRB(1.0, 3.0, 5.0, 7.0)")
	check(t, "accessors", r, Rect{Left: 1, Top: 3, Right: 5, Bottom: 7})

	c := NewRectFromCenter(Offset{1, 3}, 5, 7)
	check(t, "fromCenter", c, Rect{-1.5, -0.5, 3.5, 6.5})
	check(t, "fromCenter zero", NewRectFromCenter(Offset{0, 0}, 0, 0), Rect{0, 0, 0, 0})
	n := NewRectFromCenter(Offset{math.NaN(), 0}, 0, 0)
	isNaN(t, "left", n.Left)
	isNaN(t, "right", n.Right)
	check(t, "top", n.Top, 0.0)
	check(t, "bottom", n.Bottom, 0.0)
	n = NewRectFromCenter(Offset{0, math.NaN()}, 0, 0)
	isNaN(t, "top", n.Top)
	isNaN(t, "bottom", n.Bottom)
	check(t, "left", n.Left, 0.0)
	check(t, "right", n.Right, 0.0)

	w := NewRectFromLTWH(1, 3, 5, 7)
	check(t, "fromLTWH", w, Rect{1, 3, 6, 10})
	check(t, "shortestSide", w.ShortestSide(), 5.0)
	check(t, "longestSide", w.LongestSide(), 7.0)
}

func TestRectIntersectAndExpand(t *testing.T) {
	r1, r2 := Rect{0, 0, 100, 100}, Rect{50, 50, 200, 200}
	check(t, "intersect", r1.Intersect(r2), Rect{50, 50, 100, 100})
	check(t, "intersect symmetric", r2.Intersect(r1), r1.Intersect(r2))
	check(t, "expand", r1.ExpandToInclude(r2), Rect{0, 0, 200, 200})
	check(t, "expand symmetric", r2.ExpandToInclude(r1), r1.ExpandToInclude(r2))
	c1, c2 := Rect{50, 0, 50, 200}, Rect{0, 50, 200, 50}
	check(t, "expand crossing", c1.ExpandToInclude(c2), Rect{0, 0, 200, 200})
	check(t, "expand crossing symmetric", c2.ExpandToInclude(c1), c1.ExpandToInclude(c2))
}

func TestRRectFromRectXY(t *testing.T) {
	r := NewRRectFromRectXY(NewRectFromLTWH(1, 3, 5, 7), 1, 1)
	check(t, "outer", r.OuterRect(), Rect{1, 3, 6, 10})
	check(t, "shortestSide", r.ShortestSide(), 5.0)
	check(t, "longestSide", r.LongestSide(), 7.0)
}

func TestRRectContains(t *testing.T) {
	rr := NewRRectFromRectAndCorners(Rect{1, 1, 2, 2}, RRectCorners{TopLeft: NewRadiusCircular(0.5),
		TopRight: NewRadiusCircular(0.25), BottomRight: NewRadiusElliptical(0.25, 0.75)})
	for o, want := range map[Offset]bool{{1, 1}: false, {1.1, 1.1}: false, {1.15, 1.15}: true, {2, 1}: false,
		{1.93, 1.07}: false, {1.97, 1.7}: false, {1.7, 1.97}: true, {1, 1.99}: true} {
		check(t, "contains "+o.String(), rr.Contains(o), want)
	}
}

func TestRRectContainsLargeRadii(t *testing.T) {
	rr := NewRRectFromRectAndCorners(Rect{1, 1, 2, 2}, RRectCorners{TopLeft: NewRadiusCircular(5000),
		TopRight: NewRadiusCircular(2500), BottomRight: NewRadiusElliptical(2500, 7500)})
	for o, want := range map[Offset]bool{{1, 1}: false, {1.1, 1.1}: false, {1.15, 1.15}: true, {2, 1}: false,
		{1.93, 1.07}: false, {1.97, 1.7}: false, {1.7, 1.97}: true, {1, 1.99}: true} {
		check(t, "contains "+o.String(), rr.Contains(o), want)
	}
}

func radii(r RRect) [8]float64 {
	return [8]float64{r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY, r.BlRadiusX, r.BlRadiusY, r.BrRadiusX, r.BrRadiusY}
}

func TestRRectScaleRadii(t *testing.T) {
	want := [8]float64{0.5, 0.5, 0.25, 0.25, 0, 0, 0.25, 0.75}
	for _, c := range []RRectCorners{
		{TopLeft: NewRadiusCircular(0.5), TopRight: NewRadiusCircular(0.25), BottomRight: NewRadiusElliptical(0.25, 0.75)},
		{TopLeft: NewRadiusCircular(5000), TopRight: NewRadiusCircular(2500), BottomRight: NewRadiusElliptical(2500, 7500)},
	} {
		r := NewRRectFromRectAndCorners(Rect{1, 1, 2, 2}, c).ScaleRadii()
		check(t, "outer", r.OuterRect(), Rect{1, 1, 2, 2})
		check(t, "radii", radii(r), want)
	}
}

func TestRadiusClamp(t *testing.T) {
	zero, ten := RadiusZero, NewRadiusCircular(10)
	for _, c := range []struct {
		in, want Radius
		min, max *Radius
	}{
		{NewRadiusCircular(-100), NewRadiusCircular(0), &zero, nil},
		{NewRadiusCircular(100), NewRadiusCircular(10), nil, &ten},
		{NewRadiusElliptical(-100, 100), NewRadiusElliptical(0, 10), &zero, &ten},
		{NewRadiusElliptical(100, -100), NewRadiusElliptical(10, 0), &zero, &ten},
	} {
		r := NewRRectFromLTRBR(1, 3, 5, 7, c.in.Clamp(c.min, c.max))
		check(t, "outer", r.OuterRect(), Rect{1, 3, 5, 7})
		check(t, "trRadius", r.TrRadius(), c.want)
		check(t, "blRadius", r.BlRadius(), c.want)
	}
}

func TestRadiusClampValues(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	for _, c := range []struct {
		in, want Radius
		o        RadiusClampValuesOptions
	}{
		{NewRadiusCircular(-100), NewRadiusCircular(0), RadiusClampValuesOptions{MinimumX: f(0), MinimumY: f(0)}},
		{NewRadiusCircular(100), NewRadiusElliptical(10, 20), RadiusClampValuesOptions{MaximumX: f(10), MaximumY: f(20)}},
		{NewRadiusElliptical(-100, 100), NewRadiusElliptical(5, 20), RadiusClampValuesOptions{f(5), f(6), f(10), f(20)}},
		{NewRadiusElliptical(100, -100), NewRadiusElliptical(10, 6), RadiusClampValuesOptions{f(5), f(6), f(10), f(20)}},
	} {
		r := NewRRectFromLTRBR(1, 3, 5, 7, c.in.ClampValues(c.o))
		check(t, "outer", r.OuterRect(), Rect{1, 3, 5, 7})
		check(t, "trRadius", r.TrRadius(), c.want)
		check(t, "blRadius", r.BlRadius(), c.want)
	}
}

func TestRRectNegativeRadiiPanic(t *testing.T) {
	for _, c := range []RRectCorners{{TopLeft: NewRadiusCircular(-1)}, {TopRight: NewRadiusCircular(-2)},
		{BottomLeft: NewRadiusCircular(-3)}, {BottomRight: NewRadiusCircular(-4)}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%+v: want panic", c)
				}
			}()
			NewRRectFromRectAndCorners(Rect{10, 20, 30, 40}, c)
		}()
	}
}

func TestRRectInflateDeflateClamp(t *testing.T) {
	steps := [][8]float64{
		{0, 0, 1, 1, 2, 2, 3, 3},
		{0, 0, 0, 0, 1, 1, 2, 2},
		{0, 0, 0, 0, 0, 0, 1, 1},
		{0, 0, 0, 0, 0, 0, 0, 0},
	}
	start := NewRRectFromRectAndCorners(Rect{10, 20, 30, 40}, RRectCorners{TopLeft: NewRadiusCircular(1),
		TopRight: NewRadiusCircular(2), BottomLeft: NewRadiusCircular(3), BottomRight: NewRadiusCircular(4)})
	for _, deflate := range []bool{false, true} {
		r := start
		for i, want := range steps {
			if deflate {
				r = r.Deflate(1)
			} else {
				r = r.Inflate(-1)
			}
			check(t, "step radii", radii(r), want)
			_ = i
		}
	}
}

func TestInfinityLerp(t *testing.T) {
	a, b := Offset{math.Inf(1), math.Inf(1)}, Offset{4, 4}
	r := OffsetLerp(&a, &b, 0.5)
	if r == nil || !math.IsInf(r.Dx, 1) || !math.IsInf(r.Dy, 1) {
		t.Errorf("lerp = %v, want (inf, inf)", r)
	}
}

func checkPointWithOffset(t *testing.T, rse RSuperellipse, in, outward Offset) {
	t.Helper()
	if !rse.Contains(in) {
		t.Errorf("%v should contain %v", rse, in)
	}
	if rse.Contains(in.Add(outward)) {
		t.Errorf("%v should not contain %v", rse, in.Add(outward))
	}
}

func TestRSuperellipseContainsNoCorners(t *testing.T) {
	rse := NewRSuperellipseFromLTRBXY(-50, -50, 50, 50, 0, 0)
	for o, want := range map[Offset]bool{{-50, -50}: true, {-50, 49.99}: true, {49.99, -50}: true, {49.99, 49.99}: true,
		{-50.01, -50}: false, {-50, -50.01}: false, {-50.01, 50}: false, {-50, 50.01}: false,
		{50.01, -50}: false, {50, -50.01}: false, {50.01, 50}: false, {50, 50.01}: false} {
		check(t, "contains "+o.String(), rse.Contains(o), want)
	}
}

func TestRSuperellipseContainsTinyCorners(t *testing.T) {
	rse := NewRSuperellipseFromLTRBXY(-50, -50, 50, 50, 0.01, 0.01)
	for _, o := range []Offset{{-50, -50}, {-50, 50}, {50, -50}, {50, 50}} {
		check(t, "contains "+o.String(), rse.Contains(o), false)
	}
}

func checkMirrors(t *testing.T, rse RSuperellipse, center Offset, points []Offset) {
	t.Helper()
	for _, g := range points {
		p := g.Sub(center)
		checkPointWithOffset(t, rse, Offset{p.Dx, p.Dy}.Add(center), Offset{0.02, 0.02})
		checkPointWithOffset(t, rse, Offset{p.Dx, -p.Dy}.Add(center), Offset{0.02, -0.02})
		checkPointWithOffset(t, rse, Offset{-p.Dx, p.Dy}.Add(center), Offset{-0.02, 0.02})
		checkPointWithOffset(t, rse, Offset{-p.Dx, -p.Dy}.Add(center), Offset{-0.02, -0.02})
	}
}

func TestRSuperellipseContainsUniformCorners(t *testing.T) {
	checkMirrors(t, NewRSuperellipseFromLTRBXY(-50, -50, 50, 50, 5, 5), Offset{}, []Offset{
		{0, 49.995}, {44.245, 49.95}, {45.72, 49.87}, {48.53, 48.53}, {49.87, 45.72}, {49.95, 44.245}, {49.995, 0}})
}

func TestRSuperellipseContainsUniformEllipticalCorners(t *testing.T) {
	checkMirrors(t, NewRSuperellipseFromLTRBXY(-50, -50, 50, 50, 5, 10), Offset{}, []Offset{
		{0, 49.995}, {44.245, 49.911}, {45.72, 49.75}, {48.51, 47.07}, {49.87, 41.44}, {49.95, 38.49}, {49.995, 0}})
}

func TestRSuperellipseContainsUnequalSides(t *testing.T) {
	rse := NewRSuperellipseFromLTRBXY(0, 0, 50, 100, 23, 30)
	checkMirrors(t, rse, rse.OuterRect().Center(), []Offset{{24.99, 99.99}, {29.99, 99.64}, {34.99, 98.06},
		{39.99, 94.73}, {44.13, 89.99}, {48.46, 79.99}, {49.70, 69.99}, {49.97, 59.99}, {49.99, 49.99}})
}

func TestRSuperellipseContainsSlimDiagonal(t *testing.T) {
	rse := NewRSuperellipseFromLTRBAndCorners(-50, -50, 50, 50, RRectCorners{TopLeft: NewRadiusCircular(1),
		TopRight: NewRadiusCircular(99), BottomLeft: NewRadiusCircular(99), BottomRight: NewRadiusCircular(1)})
	check(t, "center", rse.Contains(Offset{0, 0}), true)
	for _, o := range []Offset{{-49.999, -49.999}, {-49.999, 49.999}, {49.999, 49.999}, {49.999, -49.999}} {
		check(t, "contains "+o.String(), rse.Contains(o), false)
	}
	checkPointWithOffset(t, rse, Offset{-49.70, -49.70}, Offset{-0.02, -0.02})
	checkPointWithOffset(t, rse, Offset{49.70, 49.70}, Offset{0.02, 0.02})
	for _, p := range []Offset{{-40, -49.59}, {-20, -45.64}, {0, -37.01}, {20, -21.96}, {21.05, -20.92}, {40, 5.68}} {
		checkPointWithOffset(t, rse, p, Offset{0.02, -0.02})
		checkPointWithOffset(t, rse, p.Neg(), Offset{-0.02, 0.02})
	}
}

func TestRSuperellipseContainsOutsideSharpCorner(t *testing.T) {
	rse := NewRSuperellipseFromLTRBAndCorners(196, 0, 294, 28, RRectCorners{TopRight: NewRadiusCircular(3),
		BottomRight: NewRadiusCircular(3)})
	check(t, "outside", rse.Contains(Offset{147, 14}), false)
}
