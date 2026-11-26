package painting_test

// Ports of packages/flutter/test/painting/{star_border_test,
// linear_border_test,notched_shapes_test}.dart (Flutter 3.50.1).
//
// Skipped from star_border_test: the golden tests (basic geometry,
// parameters, polygon parameters, and the lerps with StarBorder,
// CircleBorder, RoundedRectangleBorder and StadiumBorder: they pump
// Container/ShapeDecoration widgets and compare golden files; the
// painting_borders difftest compares those paths with Flutter's exactly
// and painting_borders_pixels their pixels), "StarBorder doesn't try to
// scale an infinite scale matrix" (pumps Stack/Positioned widgets; the
// path generation it exercises is called directly below), the
// StarBorderSubclass inequality (Go types can't be subclassed) and
// hashCode expectations.
//
// Skipped from notched_shapes_test: "AutomaticNotchedShape - with guest"
// (Path.combine is an ENGINE GAP: it panics with UnimplementedError).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestStarBorderDefaults(t *testing.T) {
	star := NewStarBorder()
	if star.Side != BorderSideNone || star.Points != 5 || star.InnerRadiusRatio() != 0.4 || star.Rotation() != 0 ||
		star.PointRounding != 0 || star.ValleyRounding != 0 || star.Squash != 0 {
		t.Error("star")
	}
	polygon := NewStarBorderPolygon()
	if polygon.Points != 5 || polygon.PointRounding != 0 || polygon.Rotation() != 0 || polygon.Squash != 0 {
		t.Error("polygon")
	}
}

func TestStarBorderCopyWithEquals(t *testing.T) {
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	cp := NewStarBorder().CopyWith(StarBorderCopyWithOptions{Side: &s, Points: bdF(3), InnerRadiusRatio: bdF(0.1),
		PointRounding: bdF(0.2), ValleyRounding: bdF(0.3), Rotation: bdF(180), Squash: bdF(0.4)})
	expected := NewStarBorder(StarBorderOptions{Side: &s, Points: bdF(3), InnerRadiusRatio: bdF(0.1),
		PointRounding: 0.2, ValleyRounding: 0.3, Rotation: 180, Squash: 0.4})
	if NewStarBorder() != NewStarBorder().CopyWith() {
		t.Error("copyWith()")
	}
	if cp != expected {
		t.Errorf("copy %v", cp)
	}
	// A star and a polygon differ (the polygon's innerRadiusRatio is null).
	if NewStarBorder(StarBorderOptions{Points: bdF(3), InnerRadiusRatio: bdF(1), PointRounding: 0.2, Rotation: 180, Squash: 0.4}) ==
		NewStarBorderPolygon(StarBorderPolygonOptions{Sides: bdF(3), PointRounding: 0.2, Rotation: 180, Squash: 0.4}) {
		t.Error("star == polygon")
	}
	if cp != cp {
		t.Error("copy != copy")
	}
	for i, other := range []StarBorder{
		cp.CopyWith(StarBorderCopyWithOptions{Side: side(BorderSideOptions{})}),
		cp.CopyWith(StarBorderCopyWithOptions{Points: bdF(10)}),
		cp.CopyWith(StarBorderCopyWithOptions{InnerRadiusRatio: bdF(0.5)}),
		cp.CopyWith(StarBorderCopyWithOptions{PointRounding: bdF(0.5)}),
		cp.CopyWith(StarBorderCopyWithOptions{ValleyRounding: bdF(0.5)}),
		cp.CopyWith(StarBorderCopyWithOptions{Rotation: bdF(10)}),
		cp.CopyWith(StarBorderCopyWithOptions{Squash: bdF(0.0)}),
	} {
		if cp == other {
			t.Errorf("#%d equal", i)
		}
	}
}

// TestStarBorderZeroSizePath is what "StarBorder doesn't try to scale an
// infinite scale matrix" exercises: a star in an empty rect.
func TestStarBorderZeroSizePath(t *testing.T) {
	if r := bdPanics(func() { NewStarBorder().GetOuterPath(ui.NewRectFromLTWH(0, 0, 0, 0), nil) }); r != nil {
		t.Errorf("panic %v", r)
	}
}

var (
	canvasRect     = ui.NewRectFromLTWH(0, 0, 100, 100)
	linearSide     = bdBS(BorderSideOptions{Width: bdF(4), Color: bdCol(0x0f00ff00)})
	leftRectIncl   = rectIncludes(ui.NewRectFromLTWH(0, 0, 4, 100))
	rightRectIncl  = rectIncludes(ui.NewRectFromLTWH(96, 0, 4, 100))
	topRectIncl    = rectIncludes(ui.NewRectFromLTWH(0, 0, 100, 4))
	bottomRectIncl = rectIncludes(ui.NewRectFromLTWH(0, 96, 100, 4))
)

func rectIncludes(r ui.Rect) []ui.Offset {
	return []ui.Offset{r.TopLeft(), r.TopRight(), r.BottomLeft(), r.BottomRight(), r.Center()}
}

func TestLinearBorderEdgeDefaults(t *testing.T) {
	if e := NewLinearBorderEdge(); e.Size != 1 || e.Alignment != 0 {
		t.Error("defaults")
	}
}

func TestLinearBorderDefaults(t *testing.T) {
	none := LinearBorderNone
	if none.Side != BorderSideNone || none.Dimensions() != EdgeInsetsGeometry(NewEdgeInsetsDirectionalFromSTEB(0, 0, 0, 0)) ||
		!EdgeInsetsGeometryEqual(none.Dimensions(), EdgeInsetsZero) || none.PreferPaintInterior() ||
		none.Start != nil || none.End != nil || none.Top != nil || none.Bottom != nil {
		t.Error("none")
	}
	def := NewLinearBorderEdge()
	check := func(name string, b LinearBorder, start, end, top, bottom bool) {
		t.Helper()
		if b.Side != BorderSideNone {
			t.Errorf("%s side", name)
		}
		for i, e := range []struct {
			edge *LinearBorderEdge
			want bool
		}{{b.Start, start}, {b.End, end}, {b.Top, top}, {b.Bottom, bottom}} {
			if (e.edge != nil) != e.want || (e.edge != nil && *e.edge != def) {
				t.Errorf("%s edge %d", name, i)
			}
		}
	}
	check("start", NewLinearBorderStart(), true, false, false, false)
	check("end", NewLinearBorderEnd(), false, true, false, false)
	check("top", NewLinearBorderTop(), false, false, true, false)
	check("bottom", NewLinearBorderBottom(), false, false, false, true)
}

func TestLinearBorderCopyWithEquals(t *testing.T) {
	if !ShapeBorderEqual(LinearBorderNone, LinearBorderNone.CopyWith()) {
		t.Error("copyWith()")
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	if !ShapeBorderEqual(LinearBorderNone.CopyWith(LinearBorderCopyWithOptions{Side: &s}), NewLinearBorder(LinearBorderOptions{Side: &s})) {
		t.Error("side")
	}
}

func TestLinearBorderLerpIdenticalAB(t *testing.T) {
	if OutlinedBorderLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	var border OutlinedBorder = LinearBorderNone
	if !ShapeBorderEqual(OutlinedBorderLerp(border, border, 0.5), border) {
		t.Error("identical")
	}
}

func TestLinearBorderEdgeLerpIdenticalAB(t *testing.T) {
	if LinearBorderEdgeLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	edge := NewLinearBorderEdge()
	if LinearBorderEdgeLerp(&edge, &edge, 0.5) != &edge {
		t.Error("identical")
	}
}

func TestLinearBorderEdgeLinearBorderToString(t *testing.T) {
	if got := NewLinearBorderEdge(LinearBorderEdgeOptions{Size: bdF(0.5), Alignment: -0.5}).String(); got != "LinearBorderEdge(size: 0.5, alignment: -0.5)" {
		t.Error(got)
	}
	if LinearBorderNone.String() != "LinearBorder.none" {
		t.Error(LinearBorderNone.String())
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	c := ui.NewColor(0xff123456).String()
	if got := NewLinearBorder(LinearBorderOptions{Side: &s}).String(); got != "LinearBorder(side: BorderSide(color: "+c+", width: 10.0))" {
		t.Error(got)
	}
	e := func(size, alignment float64) *LinearBorderEdge {
		x := NewLinearBorderEdge(LinearBorderEdgeOptions{Size: &size, Alignment: alignment})
		return &x
	}
	got := NewLinearBorder(LinearBorderOptions{Side: &s, Start: e(0, -0.75), End: e(0.25, -0.5), Top: e(0.5, 0.5), Bottom: e(0.75, 0.75)}).String()
	want := "LinearBorder(side: BorderSide(color: " + c + ", width: 10.0), " +
		"start: LinearBorderEdge(size: 0.0, alignment: -0.75), " +
		"end: LinearBorderEdge(size: 0.25, alignment: -0.5), " +
		"top: LinearBorderEdge(size: 0.5, alignment: 0.5), " +
		"bottom: LinearBorderEdge(size: 0.75, alignment: 0.75))"
	if got != want {
		t.Errorf("got %s", got)
	}
}

func linearPaints(includes, excludes []ui.Offset) ft.Matcher {
	c := linearSide.Color
	return ft.Paints().Path(ft.PaintPatternPathOptions{Includes: includes, Excludes: excludes, PaintOptions: ft.PaintOptions{Color: &c}})
}

func TestLinearBorderStart(t *testing.T) {
	border := NewLinearBorderStart(LinearBorderEdgeConstructorOptions{Side: &linearSide})
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, canvasRect, bdLtr) }, linearPaints(leftRectIncl, rightRectIncl))
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, canvasRect, bdRtl) }, linearPaints(rightRectIncl, leftRectIncl))
}

func TestLinearBorderEnd(t *testing.T) {
	border := NewLinearBorderEnd(LinearBorderEdgeConstructorOptions{Side: &linearSide})
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, canvasRect, bdLtr) }, linearPaints(rightRectIncl, leftRectIncl))
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, canvasRect, bdRtl) }, linearPaints(leftRectIncl, rightRectIncl))
}

func TestLinearBorderTop(t *testing.T) {
	border := NewLinearBorderTop(LinearBorderEdgeConstructorOptions{Side: &linearSide})
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, canvasRect, bdLtr) }, linearPaints(topRectIncl, bottomRectIncl))
}

func TestLinearBorderBottom(t *testing.T) {
	border := NewLinearBorderBottom(LinearBorderEdgeConstructorOptions{Side: &linearSide})
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, canvasRect, bdLtr) }, linearPaints(bottomRectIncl, topRectIncl))
}

func rectPath(r ui.Rect) *ui.Path {
	p := ui.NewPath()
	p.AddRect(r)
	return p
}

// pathDoesNotContainCircle is the notched shapes test's helper.
func pathDoesNotContainCircle(path *ui.Path, circleBounds ui.Rect) bool {
	radius := circleBounds.Width() / 2.0
	pi := dartPiForTests
	for theta := 0.0; theta <= 2.0*pi; theta += pi / 20.0 {
		for i := 0.0; i < 1; i += 0.01 {
			x := i * radius * dart.Cos(theta)
			y := i * radius * dart.Sin(theta)
			if path.Contains(ui.NewOffset(x, y).Add(circleBounds.Center())) {
				return false
			}
		}
	}
	return true
}

var dartPiForTests = 3.141592653589793

func TestCircularNotchedRectangle(t *testing.T) {
	host := ui.NewRectFromLTRB(0.0, 100.0, 300.0, 300.0)
	guest := ui.NewRectFromLTWH(50.0, 50.0, 10.0, 10.0)
	// guest and host don't overlap
	ft.Expect(t, CircularNotchedRectangle{}.GetOuterPath(host, &guest),
		ft.CoversSameAreaAs(rectPath(host), host.Inflate(5.0), ft.CoversSameAreaAsOptions{SampleSize: 40}))
	for i, c := range []struct {
		inverted bool
		guest    ui.Rect
	}{
		{false, ui.NewRectFromLTRB(190.0, 85.0, 210.0, 105.0)}, // guest center above host
		{false, ui.NewRectFromLTRB(190.0, 95.0, 210.0, 115.0)}, // guest center below host
		{true, ui.NewRectFromLTRB(190.0, 285.0, 210.0, 305.0)}, // inverted guest center above host
		{true, ui.NewRectFromLTRB(190.0, 295.0, 210.0, 315.0)}, // inverted guest center below host
	} {
		g := c.guest
		if !pathDoesNotContainCircle(CircularNotchedRectangle{Inverted: c.inverted}.GetOuterPath(host, &g), g) {
			t.Errorf("#%d contains the circle", i)
		}
	}
	// no guest is ok
	ft.Expect(t, CircularNotchedRectangle{}.GetOuterPath(host, nil),
		ft.CoversSameAreaAs(rectPath(host), host.Inflate(800.0), ft.CoversSameAreaAsOptions{SampleSize: 100}))
}

func TestAutomaticNotchedShapeNoGuest(t *testing.T) {
	expected := ui.NewPath()
	expected.MoveTo(-200.0, -100.0)
	expected.LineTo(-150.0, -100.0)
	expected.LineTo(-150.0, 0.0)
	expected.LineTo(-200.0, 0.0)
	expected.Close()
	shape := NewAutomaticNotchedShape(NewRoundedRectangleBorder(), NewRoundedRectangleBorder())
	ft.Expect(t, shape.GetOuterPath(ui.NewRectFromLTWH(-200.0, -100.0, 50.0, 100.0), nil),
		ft.CoversSameAreaAs(expected, ui.NewRectFromLTWH(-300.0, -300.0, 600.0, 600.0), ft.CoversSameAreaAsOptions{SampleSize: 100}))
}
