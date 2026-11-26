package painting_test

// Port of packages/flutter/test/painting/stadium_border_test.dart (Flutter
// 3.50.1). hashCode expectations are skipped (no hashCode in Go).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func stadiumOf(width float64) StadiumBorder {
	return NewStadiumBorder(StadiumBorderOptions{Side: w(width)})
}

func TestStadiumBorderDefaults(t *testing.T) {
	if NewStadiumBorder().Side != BorderSideNone {
		t.Error("defaults")
	}
}

func TestStadiumBorderCopyWithEquals(t *testing.T) {
	if NewStadiumBorder() != NewStadiumBorder().CopyWith(nil) {
		t.Error("copyWith()")
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	if NewStadiumBorder().CopyWith(&s) != NewStadiumBorder(StadiumBorderOptions{Side: &s}) {
		t.Error("side")
	}
}

func TestStadiumBorder(t *testing.T) {
	c10, c15, c20 := stadiumOf(10), stadiumOf(15), stadiumOf(20)
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
	c1 := NewStadiumBorder(StadiumBorderOptions{Side: side(BorderSideOptions{})})
	ft.Expect(t, c1.GetOuterPath(ui.NewRectFromCircle(ui.Offset{}, 1.0), nil), isUnitCircle)
	ft.Expect(t, c1.GetInnerPath(ui.NewRectFromCircle(ui.Offset{}, 2.0), nil), isUnitCircle)
	rect := ui.NewRectFromLTRB(10.0, 20.0, 100.0, 200.0)
	want := ui.NewRRectFromRectAndRadius(rect.Deflate(5.0), ui.NewRadiusCircular(rect.ShortestSide()/2.0-5.0))
	ft.Expect(t, func(c ui.Canvas) { c10.Paint(c, rect, nil) },
		ft.Paints().RRect(ft.PaintPatternRRectOptions{RRect: &want, PaintOptions: ft.PaintOptions{StrokeWidth: bdF(10)}}))
}

func TestStadiumBorderWithStrokeAlign(t *testing.T) {
	center := NewStadiumBorder(StadiumBorderOptions{Side: side(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(BorderSideStrokeAlignCenter)})})
	outside := NewStadiumBorder(StadiumBorderOptions{Side: side(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(BorderSideStrokeAlignOutside)})})
	if center.Dimensions() != EdgeInsetsGeometry(NewEdgeInsetsAll(5)) || outside.Dimensions() != EdgeInsetsGeometry(EdgeInsetsZero) {
		t.Error("dimensions")
	}
	rect := ui.NewRectFromLTRB(10.0, 20.0, 100.0, 200.0)
	base := ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(rect.ShortestSide()/2.0))
	inflated := base.Inflate(5.0)
	ft.Expect(t, func(c ui.Canvas) { center.Paint(c, rect, nil) },
		ft.Paints().RRect(ft.PaintPatternRRectOptions{RRect: &base, PaintOptions: ft.PaintOptions{StrokeWidth: bdF(10)}}))
	ft.Expect(t, func(c ui.Canvas) { outside.Paint(c, rect, nil) },
		ft.Paints().RRect(ft.PaintPatternRRectOptions{RRect: &inflated, PaintOptions: ft.PaintOptions{StrokeWidth: bdF(10)}}))
}

func TestStadiumBorderAndCircleBorder(t *testing.T) {
	stadium, circle := NewStadiumBorder(), NewCircleBorder()
	rect := ui.NewRectFromLTWH(0.0, 0.0, 100.0, 20.0)
	looksLikeS := bdPath(bdOffsets(30, 10, 50, 10), bdOffsets(1, 1, 99, 19))
	looksLikeC := bdPath(bdOffsets(50, 10), bdOffsets(1, 1, 30, 10, 99, 19))
	lerp := ShapeBorderLerp
	outer := func(b ShapeBorder) *ui.Path { return b.GetOuterPath(rect, nil) }
	s, c := ShapeBorder(stadium), ShapeBorder(circle)
	ft.Expect(t, outer(s), looksLikeS)
	ft.Expect(t, outer(c), looksLikeC)
	ft.Expect(t, outer(lerp(s, c, 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(s, c, 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(lerp(s, c, 0.9), s, 0.1)), looksLikeC)
	ft.Expect(t, outer(lerp(lerp(s, c, 0.9), s, 0.9)), looksLikeS)
	ft.Expect(t, outer(lerp(lerp(s, c, 0.1), c, 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(lerp(s, c, 0.1), c, 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(lerp(s, c, 0.1), lerp(s, c, 0.9), 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(lerp(s, c, 0.1), lerp(s, c, 0.9), 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(s, lerp(s, c, 0.9), 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(s, lerp(s, c, 0.9), 0.9)), looksLikeC)
	ft.Expect(t, outer(lerp(c, lerp(s, c, 0.1), 0.1)), looksLikeC)
	ft.Expect(t, outer(lerp(c, lerp(s, c, 0.1), 0.9)), looksLikeS)
	pct := func(p string) string {
		return "StadiumBorder(BorderSide(width: 0.0, style: none), " + p + "% of the way to being a CircleBorder)"
	}
	for i, x := range []struct {
		b    ShapeBorder
		want string
	}{
		{lerp(s, c, 0.1), pct("10.0")},
		{lerp(s, c, 0.2), pct("20.0")},
		{lerp(lerp(s, c, 0.1), lerp(s, c, 0.9), 0.9), pct("82.0")},
		{lerp(c, s, 0.9), pct("10.0")},
		{lerp(c, s, 0.8), pct("20.0")},
		{lerp(lerp(s, c, 0.9), lerp(s, c, 0.1), 0.1), pct("82.0")},
	} {
		if x.b.String() != x.want {
			t.Errorf("#%d: %s", i, x.b)
		}
	}
	if !ShapeBorderEqual(lerp(s, c, 0.1), lerp(s, c, 0.1)) {
		t.Error("==")
	}
	direct50 := lerp(s, c, 0.5)
	indirect50 := lerp(lerp(c, s, 0.1), lerp(c, s, 0.9), 0.5)
	if !ShapeBorderEqual(direct50, indirect50) || direct50.String() != indirect50.String() {
		t.Errorf("direct50 %v, indirect50 %v", direct50, indirect50)
	}
}

func stadiumAndRRect(t *testing.T, radius BorderRadiusGeometry, radiusString string, td *ui.TextDirection) {
	t.Helper()
	s := ShapeBorder(NewStadiumBorder())
	r := ShapeBorder(NewRoundedRectangleBorder(RRectBorderOptions{BorderRadius: radius}))
	rect := ui.NewRectFromLTWH(0.0, 0.0, 100.0, 50.0)
	looksLikeS := bdPath(bdOffsets(25, 25, 50, 25, 7.33, 7.33), bdOffsets(0.001, 0.001, 99.999, 0.001, 99.999, 49.999, 0.001, 49.999))
	looksLikeR := bdPath(bdOffsets(25, 25, 50, 25, 7.33, 7.33, 4, 4, 96, 4, 96, 46, 4, 46), nil)
	lerp := ShapeBorderLerp
	outer := func(b ShapeBorder) *ui.Path { return b.GetOuterPath(rect, td) }
	ft.Expect(t, outer(s), looksLikeS)
	ft.Expect(t, outer(r), looksLikeR)
	ft.Expect(t, outer(lerp(s, r, 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(s, r, 0.9)), looksLikeR)
	ft.Expect(t, outer(lerp(lerp(s, r, 0.9), s, 0.1)), looksLikeR)
	ft.Expect(t, outer(lerp(lerp(s, r, 0.9), s, 0.9)), looksLikeS)
	ft.Expect(t, outer(lerp(lerp(s, r, 0.1), r, 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(lerp(s, r, 0.1), r, 0.9)), looksLikeS)
	ft.Expect(t, outer(lerp(lerp(s, r, 0.1), lerp(s, r, 0.9), 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(lerp(s, r, 0.1), lerp(s, r, 0.9), 0.9)), looksLikeR)
	ft.Expect(t, outer(lerp(s, lerp(s, r, 0.9), 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(s, lerp(s, r, 0.9), 0.9)), looksLikeR)
	ft.Expect(t, outer(lerp(r, lerp(s, r, 0.1), 0.1)), looksLikeS)
	ft.Expect(t, outer(lerp(r, lerp(s, r, 0.1), 0.9)), looksLikeS)
	pct := func(p string) string {
		return "StadiumBorder(BorderSide(width: 0.0, style: none), " + radiusString + ", " + p +
			"% of the way to being a RoundedRectangleBorder)"
	}
	for i, x := range []struct {
		b    ShapeBorder
		want string
	}{
		{lerp(s, r, 0.1), pct("10.0")},
		{lerp(s, r, 0.2), pct("20.0")},
		{lerp(lerp(s, r, 0.1), lerp(s, r, 0.9), 0.9), pct("82.0")},
		{lerp(r, s, 0.9), pct("10.0")},
		{lerp(r, s, 0.8), pct("20.0")},
		{lerp(lerp(s, r, 0.9), lerp(s, r, 0.1), 0.1), pct("82.0")},
	} {
		if x.b.String() != x.want {
			t.Errorf("#%d: %s", i, x.b)
		}
	}
	if !ShapeBorderEqual(lerp(s, r, 0.1), lerp(s, r, 0.1)) {
		t.Error("==")
	}
	direct50 := lerp(s, r, 0.5)
	indirect50 := lerp(lerp(r, s, 0.1), lerp(r, s, 0.9), 0.5)
	if !ShapeBorderEqual(direct50, indirect50) || direct50.String() != indirect50.String() {
		t.Errorf("direct50 %v, indirect50 %v", direct50, indirect50)
	}
}

func TestStadiumBorderAndRoundedRectBorderWithBorderRadiusZero(t *testing.T) {
	stadiumAndRRect(t, BorderRadiusZero, "BorderRadius.zero", nil)
}

func TestStadiumBorderAndRoundedRectBorderWithCircularBorderRadius(t *testing.T) {
	stadiumAndRRect(t, NewBorderRadiusAll(ui.NewRadiusCircular(10)), "BorderRadius.circular(10.0)", nil)
}

func TestStadiumBorderAndRoundedRectBorderWithBorderRadiusDirectional(t *testing.T) {
	stadiumAndRRect(t, BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(10), BottomEnd: ui.NewRadiusCircular(10)},
		"BorderRadiusDirectional.only(topStart: Radius.circular(10.0), bottomEnd: Radius.circular(10.0))", bdRtl)
}
