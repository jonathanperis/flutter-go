package painting_test

// Ports of packages/flutter/test/painting/{beveled_rectangle_border_test,
// continuous_rectangle_border_test}.dart (Flutter 3.50.1).
//
// Skipped: the 7 ContinuousRectangleBorder golden tests (they pump
// DecoratedBox/ShapeDecoration widgets and compare golden files; widgets
// are not ported and flutter-go has no golden files: the
// painting_borders_pixels difftest compares ContinuousRectangleBorder
// pixels with Flutter's instead), and hashCode expectations (no hashCode
// in Go).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// rrectLikeBorder builds one of the two border types from options.
type rrectLikeBorder func(o RRectBorderOptions) OutlinedBorder

var (
	beveled    rrectLikeBorder = func(o RRectBorderOptions) OutlinedBorder { return NewBeveledRectangleBorder(o) }
	continuous rrectLikeBorder = func(o RRectBorderOptions) OutlinedBorder { return NewContinuousRectangleBorder(o) }
)

func testRRectLikeDefaultsAndCopyWith(t *testing.T, mk rrectLikeBorder, copyWith func(b OutlinedBorder, o RRectBorderOptions) OutlinedBorder) {
	t.Helper()
	d := mk(RRectBorderOptions{})
	if d.GetSide() != BorderSideNone {
		t.Error("default side")
	}
	if copyWith(d, RRectBorderOptions{}) != d {
		t.Error("copyWith()")
	}
	s := bdBS(BorderSideOptions{Width: bdF(10.0), Color: bdCol(0xff123456)})
	for _, r := range []BorderRadiusGeometry{NewBorderRadiusAll(ui.NewRadiusCircular(16)), NewBorderRadiusDirectionalAll(ui.NewRadiusCircular(16))} {
		if copyWith(d, RRectBorderOptions{Side: &s, BorderRadius: r}) != mk(RRectBorderOptions{Side: &s, BorderRadius: r}) {
			t.Errorf("copyWith %v", r)
		}
	}
}

func testRRectLikeScaleAndLerp(t *testing.T, mk rrectLikeBorder) {
	t.Helper()
	b := func(width, radius float64) OutlinedBorder {
		return mk(RRectBorderOptions{Side: w(width), BorderRadius: NewBorderRadiusAll(ui.NewRadiusCircular(radius))})
	}
	c10, c15, c20 := b(10, 100), b(15, 150), b(20, 200)
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
}

func testRRectLikePaths(t *testing.T, mk rrectLikeBorder) {
	t.Helper()
	rect1 := ui.NewRectFromLTRB(10.0, 20.0, 30.0, 40.0)
	looksLikeRect1 := bdPath(bdOffsets(10, 20, 20, 30), bdOffsets(9, 19, 31, 41))
	// Default border radius and border side are zero, i.e. just a rectangle.
	ft.Expect(t, mk(RRectBorderOptions{}).GetOuterPath(rect1, nil), looksLikeRect1)
	ft.Expect(t, mk(RRectBorderOptions{}).GetInnerPath(rect1, nil), looksLikeRect1)
	// The inner path when borderSide.width = 4: rect1 inset by 4.
	looksLikeInnerPath := bdPath(bdOffsets(14, 24, 16, 26), bdOffsets(9, 23, 27, 37))
	ft.Expect(t, mk(RRectBorderOptions{Side: w(4)}).GetOuterPath(rect1, nil), looksLikeRect1)
	ft.Expect(t, mk(RRectBorderOptions{Side: w(4)}).GetInnerPath(rect1, nil), looksLikeInnerPath)

	// non-zero BorderRadius
	looksLikeRect := bdPath(bdOffsets(15, 25, 20, 30), bdOffsets(10, 20, 30, 40))
	border := mk(RRectBorderOptions{BorderRadius: NewBorderRadiusAll(ui.NewRadiusCircular(5.0))})
	ft.Expect(t, border.GetOuterPath(rect1, nil), looksLikeRect)
	ft.Expect(t, border.GetInnerPath(rect1, nil), looksLikeRect)

	// non-zero BorderRadiusDirectional
	looksLikeRectLtr := bdPath(bdOffsets(15, 25, 20, 30), bdOffsets(10, 20, 10, 40))
	dir := mk(RRectBorderOptions{BorderRadius: BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(5.0),
		BottomStart: ui.NewRadiusCircular(5.0)}})
	ft.Expect(t, dir.GetOuterPath(rect1, bdLtr), looksLikeRectLtr)
	ft.Expect(t, dir.GetInnerPath(rect1, bdLtr), looksLikeRectLtr)
	looksLikeRectRtl := bdPath(bdOffsets(25, 35, 25, 25), bdOffsets(30, 20, 30, 40))
	ft.Expect(t, dir.GetOuterPath(rect1, bdRtl), looksLikeRectRtl)
	ft.Expect(t, dir.GetInnerPath(rect1, bdRtl), looksLikeRectRtl)
}

func TestBeveledRectangleBorderDefaultsAndCopyWith(t *testing.T) {
	if NewBeveledRectangleBorder().BorderRadius != BorderRadiusGeometry(BorderRadiusZero) {
		t.Error("default radius")
	}
	testRRectLikeDefaultsAndCopyWith(t, beveled, func(b OutlinedBorder, o RRectBorderOptions) OutlinedBorder {
		return b.(BeveledRectangleBorder).CopyWith(o)
	})
}

func TestBeveledRectangleBorderScaleAndLerp(t *testing.T) { testRRectLikeScaleAndLerp(t, beveled) }

func TestBeveledRectangleBorderPaths(t *testing.T) { testRRectLikePaths(t, beveled) }

func TestBeveledRectangleBorderWithStrokeAlign(t *testing.T) {
	radius := NewBorderRadiusAll(ui.NewRadiusCircular(10))
	mk := func(a float64) BeveledRectangleBorder {
		return NewBeveledRectangleBorder(RRectBorderOptions{Side: side(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(a)}), BorderRadius: radius})
	}
	inside, center, outside := mk(BorderSideStrokeAlignInside), mk(BorderSideStrokeAlignCenter), mk(BorderSideStrokeAlignOutside)
	if inside.Dimensions() != EdgeInsetsGeometry(NewEdgeInsetsAll(10)) || center.Dimensions() != EdgeInsetsGeometry(NewEdgeInsetsAll(5)) ||
		outside.Dimensions() != EdgeInsetsGeometry(EdgeInsetsZero) {
		t.Error("dimensions")
	}
	rect := ui.NewRectFromLTWH(0.0, 0.0, 120.0, 40.0)
	ft.Expect(t, inside.GetInnerPath(rect, nil), bdPath(bdOffsets(10, 20, 100, 10, 50, 30, 50, 20), bdOffsets(9, 9, 100, 0, 110, 31, 9, 31)))
	ft.Expect(t, center.GetInnerPath(rect, nil), bdPath(bdOffsets(9, 9, 100, 10, 110, 31, 9, 31), bdOffsets(4, 4, 100, 0, 116, 31, 4, 31)))
	ft.Expect(t, outside.GetInnerPath(rect, nil), bdPath(bdOffsets(5, 5, 110, 0, 116, 31, 4, 31), bdOffsets(0, 0, 120, 0, 120, 31, 0, 31)))
}

func TestContinuousRectangleBorderDefaultsAndCopyWith(t *testing.T) {
	if NewContinuousRectangleBorder().BorderRadius != BorderRadiusGeometry(BorderRadiusZero) {
		t.Error("default radius")
	}
	testRRectLikeDefaultsAndCopyWith(t, continuous, func(b OutlinedBorder, o RRectBorderOptions) OutlinedBorder {
		return b.(ContinuousRectangleBorder).CopyWith(o)
	})
}

func TestContinuousRectangleBorderScaleAndLerp(t *testing.T) {
	testRRectLikeScaleAndLerp(t, continuous)
}

func TestContinuousRectangleBorderPaths(t *testing.T) { testRRectLikePaths(t, continuous) }
