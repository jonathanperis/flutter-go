package painting_test

// Ports of packages/flutter/test/painting/{circle_border_test,
// oval_border_test,rounded_rectangle_border_test}.dart (Flutter 3.50.1).
// hashCode expectations are skipped (Go values have no hashCode; equal
// values are ==).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func circleOf(width float64) CircleBorder {
	return NewCircleBorder(CircleBorderOptions{Side: w(width)})
}

func TestCircleBorderDefaults(t *testing.T) {
	border := NewCircleBorder()
	if border.Side != BorderSideNone || border.Eccentricity != 0.0 {
		t.Error("defaults")
	}
}

func TestCircleBorderGetInnerPathAndGetOuterPath(t *testing.T) {
	circleRect := ui.NewRectFromLTWH(50, 0, 100, 100)
	rect := ui.NewRectFromLTWH(0, 0, 200, 100)
	bdBounds(t, "inner", NewCircleBorder().GetInnerPath(rect, nil), circleRect)
	bdBounds(t, "outer", NewCircleBorder().GetOuterPath(rect, nil), circleRect)
	oval := NewCircleBorder(CircleBorderOptions{Eccentricity: 1.0})
	bdBounds(t, "oval outer", oval.GetOuterPath(rect, nil), rect)
	bdBounds(t, "oval inner", oval.GetInnerPath(rect, nil), rect)
	o10 := NewCircleBorder(CircleBorderOptions{Side: w(10), Eccentricity: 1.0})
	bdBounds(t, "o10 outer", o10.GetOuterPath(rect, nil), ui.NewRectFromLTWH(0, 0, 200, 100))
	bdBounds(t, "o10 inner", o10.GetInnerPath(rect, nil), ui.NewRectFromLTWH(10, 10, 180, 80))
}

func TestCircleBorderCopyWithEquals(t *testing.T) {
	if NewCircleBorder() != NewCircleBorder().CopyWith() {
		t.Error("copyWith()")
	}
	if NewCircleBorder(CircleBorderOptions{Eccentricity: 0.5}) != NewCircleBorder().CopyWith(CircleBorderCopyWithOptions{Eccentricity: bdF(0.5)}) {
		t.Error("eccentricity")
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	if NewCircleBorder().CopyWith(CircleBorderCopyWithOptions{Side: &s}) != NewCircleBorder(CircleBorderOptions{Side: &s}) {
		t.Error("side")
	}
}

func TestCircleBorder(t *testing.T) {
	c10, c15, c20 := circleOf(10), circleOf(15), circleOf(20)
	if c10.Dimensions() != EdgeInsetsGeometry(NewEdgeInsetsAll(10)) {
		t.Error("dimensions")
	}
	if c10.Scale(2.0) != ShapeBorder(c20) || c20.Scale(0.5) != ShapeBorder(c10) {
		t.Error("scale")
	}
	if ShapeBorderLerp(c10, c20, 0.0) != ShapeBorder(c10) || ShapeBorderLerp(c10, c20, 0.5) != ShapeBorder(c15) ||
		ShapeBorderLerp(c10, c20, 1.0) != ShapeBorder(c20) {
		t.Error("lerp")
	}
	ft.Expect(t, c10.GetInnerPath(ui.NewRectFromCircle(ui.Offset{}, 1.0).Inflate(10.0), nil), isUnitCircle)
	ft.Expect(t, c10.GetOuterPath(ui.NewRectFromCircle(ui.Offset{}, 1.0), nil), isUnitCircle)
	ft.Expect(t, func(c ui.Canvas) { c10.Paint(c, ui.NewRectFromLTWH(10.0, 20.0, 30.0, 40.0), nil) },
		ft.Paints().Circle(ft.PaintPatternCircleOptions{X: bdF(25), Y: bdF(40), Radius: bdF(10),
			PaintOptions: ft.PaintOptions{StrokeWidth: bdF(10)}}))
}

func ovalOf(width float64) OvalBorder { return NewOvalBorder(OvalBorderOptions{Side: w(width)}) }

func TestOvalBorderDefaults(t *testing.T) {
	if b := NewOvalBorder(); b.Side != BorderSideNone || b.Eccentricity != 1.0 {
		t.Error("defaults")
	}
}

func TestOvalBorderCopyWithEquals(t *testing.T) {
	if NewOvalBorder() != NewOvalBorder().CopyWith() {
		t.Error("copyWith()")
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	if NewOvalBorder().CopyWith(OvalBorderCopyWithOptions{Side: &s}) != NewOvalBorder(OvalBorderOptions{Side: &s}) {
		t.Error("side")
	}
}

func TestOvalBorder(t *testing.T) {
	c10, c15, c20 := ovalOf(10), ovalOf(15), ovalOf(20)
	if c10.Dimensions() != EdgeInsetsGeometry(NewEdgeInsetsAll(10)) {
		t.Error("dimensions")
	}
	if c10.Scale(2.0) != ShapeBorder(c20) || c20.Scale(0.5) != ShapeBorder(c10) {
		t.Error("scale")
	}
	if ShapeBorderLerp(c10, c20, 0.0) != ShapeBorder(c10) || ShapeBorderLerp(c10, c20, 0.5) != ShapeBorder(c15) ||
		ShapeBorderLerp(c10, c20, 1.0) != ShapeBorder(c20) {
		t.Error("lerp")
	}
	ft.Expect(t, c10.GetInnerPath(ui.NewRectFromLTWH(0, 0, 100, 40), nil),
		bdPath(bdOffsets(12, 19, 50, 10, 88, 19, 50, 29), bdOffsets(17, 26, 15, 15, 74, 10, 76, 28)))
	ft.Expect(t, c10.GetOuterPath(ui.NewRectFromLTWH(0, 0, 100, 20), nil),
		bdPath(bdOffsets(2, 9, 50, 0, 98, 9, 50, 19), bdOffsets(7, 16, 10, 2, 84, 1, 86, 18)))
}

func rrb(width, radius float64) RoundedRectangleBorder {
	return NewRoundedRectangleBorder(RRectBorderOptions{Side: w(width),
		BorderRadius: NewBorderRadiusAll(ui.NewRadiusCircular(radius))})
}

func TestRoundedRectangleBorderDefaults(t *testing.T) {
	b := NewRoundedRectangleBorder()
	if b.Side != BorderSideNone || b.BorderRadius != BorderRadiusGeometry(BorderRadiusZero) {
		t.Error("defaults")
	}
}

func TestRoundedRectangleBorderCopyWithEquals(t *testing.T) {
	if NewRoundedRectangleBorder() != NewRoundedRectangleBorder().CopyWith() {
		t.Error("copyWith()")
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	radius := NewBorderRadiusAll(ui.NewRadiusCircular(16.0))
	directionalRadius := NewBorderRadiusDirectionalAll(ui.NewRadiusCircular(16.0))
	if NewRoundedRectangleBorder().CopyWith(RRectBorderOptions{Side: &s, BorderRadius: radius}) !=
		NewRoundedRectangleBorder(RRectBorderOptions{Side: &s, BorderRadius: radius}) {
		t.Error("radius")
	}
	if NewRoundedRectangleBorder().CopyWith(RRectBorderOptions{Side: &s, BorderRadius: directionalRadius}) !=
		NewRoundedRectangleBorder(RRectBorderOptions{Side: &s, BorderRadius: directionalRadius}) {
		t.Error("directional")
	}
}

func TestRoundedRectangleBorder(t *testing.T) {
	c10, c15, c20 := rrb(10, 100), rrb(15, 150), rrb(20, 200)
	if c10.Dimensions() != EdgeInsetsGeometry(NewEdgeInsetsAll(10)) {
		t.Error("dimensions")
	}
	if c10.Scale(2.0) != ShapeBorder(c20) || c20.Scale(0.5) != ShapeBorder(c10) {
		t.Error("scale")
	}
	if ShapeBorderLerp(c10, c20, 0.0) != ShapeBorder(c10) || ShapeBorderLerp(c10, c20, 0.5) != ShapeBorder(c15) ||
		ShapeBorderLerp(c10, c20, 1.0) != ShapeBorder(c20) {
		t.Error("lerp")
	}
	c1, c2 := rrb(1, 1), rrb(1, 2)
	ft.Expect(t, c2.GetInnerPath(ui.NewRectFromCircle(ui.Offset{}, 2.0), nil), isUnitCircle)
	ft.Expect(t, c1.GetOuterPath(ui.NewRectFromCircle(ui.Offset{}, 1.0), nil), isUnitCircle)
	rect := ui.NewRectFromLTRB(10.0, 20.0, 80.0, 190.0)
	outer := ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(100.0))
	inner := ui.NewRRectFromRectAndRadius(rect.Deflate(10.0), ui.NewRadiusCircular(90.0))
	ft.Expect(t, func(c ui.Canvas) { c10.Paint(c, rect, nil) }, ft.Paints().DRRect(ft.PaintPatternDRRectOptions{
		Outer: &outer, Inner: &inner, PaintOptions: ft.PaintOptions{StrokeWidth: bdF(0)}}))
	directional := NewRoundedRectangleBorder(RRectBorderOptions{
		BorderRadius: BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(20)}})
	if !ShapeBorderEqual(ShapeBorderLerp(directional, c10, 1.0), ShapeBorderLerp(c10, directional, 0.0)) {
		t.Error("directional lerp")
	}
}

func TestRoundedRectangleBorderAndCircleBorder(t *testing.T) {
	r := NewRoundedRectangleBorder(RRectBorderOptions{BorderRadius: NewBorderRadiusAll(ui.NewRadiusCircular(10.0))})
	c := NewCircleBorder()
	rect := ui.NewRectFromLTWH(0.0, 0.0, 100.0, 20.0) // center is x=40..60 y=10
	looksLikeR := bdPath(bdOffsets(30, 10, 50, 10), bdOffsets(1, 1, 99, 19))
	looksLikeC := bdPath(bdOffsets(50, 10), bdOffsets(1, 1, 30, 10, 99, 19))
	lerp := ShapeBorderLerp
	outer := func(b ShapeBorder) *ui.Path { return b.GetOuterPath(rect, nil) }
	ft.Expect(t, outer(r), looksLikeR)
	ft.Expect(t, outer(c), looksLikeC)
	ft.Expect(t, outer(lerp(r, c, 0.1)), looksLikeR)
	ft.Expect(t, outer(lerp(r, c, 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(lerp(r, c, 0.9), r, 0.1)), looksLikeC)
	ft.Expect(t, outer(lerp(lerp(r, c, 0.9), r, 0.9)), looksLikeR)
	ft.Expect(t, outer(lerp(lerp(r, c, 0.1), c, 0.1)), looksLikeR)
	ft.Expect(t, outer(lerp(lerp(r, c, 0.1), c, 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(lerp(r, c, 0.1), lerp(r, c, 0.9), 0.1)), looksLikeR)
	ft.Expect(t, outer(lerp(lerp(r, c, 0.1), lerp(r, c, 0.9), 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(r, lerp(r, c, 0.9), 0.1)), looksLikeR)
	ft.Expect(t, outer(lerp(r, lerp(r, c, 0.9), 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(c, lerp(r, c, 0.1), 0.1)), looksLikeC)
	ft.Expect(t, outer(lerp(c, lerp(r, c, 0.1), 0.9)), looksLikeR)

	pct := func(p string) string {
		return "RoundedRectangleBorder(BorderSide(width: 0.0, style: none), BorderRadius.circular(10.0), " + p +
			"% of the way to being a CircleBorder)"
	}
	for i, x := range []struct {
		b    ShapeBorder
		want string
	}{
		{lerp(r, c, 0.1), pct("10.0")},
		{lerp(r, c, 0.2), pct("20.0")},
		{lerp(lerp(r, c, 0.1), lerp(r, c, 0.9), 0.9), pct("82.0")},
		{lerp(c, r, 0.9), pct("10.0")},
		{lerp(c, r, 0.8), pct("20.0")},
		{lerp(lerp(r, c, 0.9), lerp(r, c, 0.1), 0.1), pct("82.0")},
	} {
		if x.b.String() != x.want {
			t.Errorf("#%d: %s", i, x.b)
		}
	}
	if !ShapeBorderEqual(lerp(r, c, 0.1), lerp(r, c, 0.1)) {
		t.Error("==")
	}
	direct50 := lerp(r, c, 0.5)
	indirect50 := lerp(lerp(c, r, 0.1), lerp(c, r, 0.9), 0.5)
	if !ShapeBorderEqual(direct50, indirect50) || direct50.String() != indirect50.String() {
		t.Errorf("direct50 %v, indirect50 %v", direct50, indirect50)
	}
}

func TestRoundedRectangleBorderAndCircleBorderDimensions(t *testing.T) {
	s := func(a float64) *BorderSide { return side(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(a)}) }
	for i, c := range []struct {
		b    ShapeBorder
		want EdgeInsetsGeometry
	}{
		{NewRoundedRectangleBorder(RRectBorderOptions{Side: w(10)}), NewEdgeInsetsAll(10)},
		{NewRoundedRectangleBorder(RRectBorderOptions{Side: s(BorderSideStrokeAlignCenter)}), NewEdgeInsetsAll(5)},
		{NewRoundedRectangleBorder(RRectBorderOptions{Side: s(BorderSideStrokeAlignOutside)}), EdgeInsetsZero},
		{NewCircleBorder(CircleBorderOptions{Side: w(10)}), NewEdgeInsetsAll(10)},
		{NewCircleBorder(CircleBorderOptions{Side: s(BorderSideStrokeAlignCenter)}), NewEdgeInsetsAll(5)},
		{NewCircleBorder(CircleBorderOptions{Side: s(BorderSideStrokeAlignOutside)}), EdgeInsetsZero},
	} {
		if c.b.Dimensions() != c.want {
			t.Errorf("#%d: %v", i, c.b.Dimensions())
		}
	}
}

func TestRoundedRectangleBorderLerpWithDifferentStrokeAlign(t *testing.T) {
	rInside := NewRoundedRectangleBorder(RRectBorderOptions{Side: w(10)})
	rOutside := NewRoundedRectangleBorder(RRectBorderOptions{Side: side(BorderSideOptions{Width: bdF(20), StrokeAlign: bdF(BorderSideStrokeAlignOutside)})})
	rCenter := NewRoundedRectangleBorder(RRectBorderOptions{Side: side(BorderSideOptions{Width: bdF(15), StrokeAlign: bdF(BorderSideStrokeAlignCenter)})})
	if ShapeBorderLerp(rInside, rOutside, 0.5) != ShapeBorder(rCenter) {
		t.Errorf("got %v", ShapeBorderLerp(rInside, rOutside, 0.5))
	}
}
