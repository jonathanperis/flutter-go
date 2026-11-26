package painting_test

// Port of packages/flutter/test/painting/rounded_superellipse_border_test.dart
// (Flutter 3.50.1).
//
// Skipped: the "RoundedSuperellipseBorder looks correct" group (7 golden
// file tests pumping Container/ShapeDecoration widgets: widgets are not
// ported and flutter-go has no golden files; the painting_borders_pixels
// difftest compares RoundedSuperellipseBorder pixels with Flutter
// instead). hashCode expectations are skipped (no hashCode in Go).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func rsb(width, radius float64) RoundedSuperellipseBorder {
	return NewRoundedSuperellipseBorder(RRectBorderOptions{Side: w(width),
		BorderRadius: NewBorderRadiusAll(ui.NewRadiusCircular(radius))})
}

func TestRoundedSuperellipseBorderDefaults(t *testing.T) {
	b := NewRoundedSuperellipseBorder()
	if b.Side != BorderSideNone || b.BorderRadius != BorderRadiusGeometry(BorderRadiusZero) {
		t.Error("defaults")
	}
}

func TestRoundedSuperellipseBorderCopyWithEquals(t *testing.T) {
	if NewRoundedSuperellipseBorder() != NewRoundedSuperellipseBorder().CopyWith() {
		t.Error("copyWith()")
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	radius := NewBorderRadiusAll(ui.NewRadiusCircular(16.0))
	directionalRadius := NewBorderRadiusDirectionalAll(ui.NewRadiusCircular(16.0))
	if NewRoundedSuperellipseBorder().CopyWith(RRectBorderOptions{Side: &s, BorderRadius: radius}) !=
		NewRoundedSuperellipseBorder(RRectBorderOptions{Side: &s, BorderRadius: radius}) {
		t.Error("radius")
	}
	if NewRoundedSuperellipseBorder().CopyWith(RRectBorderOptions{Side: &s, BorderRadius: directionalRadius}) !=
		NewRoundedSuperellipseBorder(RRectBorderOptions{Side: &s, BorderRadius: directionalRadius}) {
		t.Error("directional")
	}
}

func TestRoundedSuperellipseBorder(t *testing.T) {
	c10, c15, c20 := rsb(10, 100), rsb(15, 150), rsb(20, 200)
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
	c1, c2 := rsb(1, 1), rsb(1, 2)
	ft.Expect(t, c2.GetInnerPath(ui.NewRectFromCircle(ui.Offset{}, 2.0), nil), isUnitCircle)
	ft.Expect(t, c1.GetOuterPath(ui.NewRectFromCircle(ui.Offset{}, 1.0), nil), isUnitCircle)
	rect := ui.NewRectFromLTRB(10.0, 20.0, 80.0, 190.0)
	want := ui.NewRSuperellipseFromRectAndRadius(rect.Deflate(5.0), ui.NewRadiusCircular(95.0))
	ft.Expect(t, func(c ui.Canvas) { c10.Paint(c, rect, nil) }, ft.Paints().RSuperellipse(
		ft.PaintPatternRSuperellipseOptions{RSuperellipse: &want, StrokeWidth: bdF(10)}))
	directional := NewRoundedSuperellipseBorder(RRectBorderOptions{
		BorderRadius: BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(20)}})
	if !ShapeBorderEqual(ShapeBorderLerp(directional, c10, 1.0), ShapeBorderLerp(c10, directional, 0.0)) {
		t.Error("directional lerp")
	}
}

func TestRoundedSuperellipseBorderAndCircleBorder(t *testing.T) {
	r := NewRoundedSuperellipseBorder(RRectBorderOptions{BorderRadius: NewBorderRadiusAll(ui.NewRadiusCircular(10.0))})
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
		return "RoundedSuperellipseBorder(BorderSide(width: 0.0, style: none), BorderRadius.circular(10.0), " + p +
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

func TestRoundedSuperellipseBorderDimensions(t *testing.T) {
	s := func(a float64) *BorderSide { return side(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(a)}) }
	for i, c := range []struct {
		b    ShapeBorder
		want EdgeInsetsGeometry
	}{
		{NewRoundedSuperellipseBorder(RRectBorderOptions{Side: w(10)}), NewEdgeInsetsAll(10)},
		{NewRoundedSuperellipseBorder(RRectBorderOptions{Side: s(BorderSideStrokeAlignCenter)}), NewEdgeInsetsAll(5)},
		{NewRoundedSuperellipseBorder(RRectBorderOptions{Side: s(BorderSideStrokeAlignOutside)}), EdgeInsetsZero},
	} {
		if c.b.Dimensions() != c.want {
			t.Errorf("#%d: %v", i, c.b.Dimensions())
		}
	}
}

func TestRoundedSuperellipseBorderLerpWithDifferentStrokeAlign(t *testing.T) {
	rInside := NewRoundedSuperellipseBorder(RRectBorderOptions{Side: w(10)})
	rOutside := NewRoundedSuperellipseBorder(RRectBorderOptions{Side: side(BorderSideOptions{Width: bdF(20), StrokeAlign: bdF(BorderSideStrokeAlignOutside)})})
	rCenter := NewRoundedSuperellipseBorder(RRectBorderOptions{Side: side(BorderSideOptions{Width: bdF(15), StrokeAlign: bdF(BorderSideStrokeAlignCenter)})})
	if ShapeBorderLerp(rInside, rOutside, 0.5) != ShapeBorder(rCenter) {
		t.Errorf("got %v", ShapeBorderLerp(rInside, rOutside, 0.5))
	}
}
