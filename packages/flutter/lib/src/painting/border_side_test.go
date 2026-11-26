package painting_test

// Port of packages/flutter/test/painting/border_side_test.dart (Flutter
// 3.50.1). Nothing skipped. Dart's identical() on the lerp result is Go's
// == on the value.

import (
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestBorderSideAssertsWhenConstructedIncorrectly(t *testing.T) {
	if NewBorderSide() != NewBorderSide() {
		t.Error("==")
	}
	bdExpectAssertion(t, func() { bdBS(BorderSideOptions{Width: bdF(-1.0)}) })
	negZero := dart.Mod(0, 1) * -1
	if bdBS(BorderSideOptions{Width: &negZero}) != bdBS(BorderSideOptions{Width: bdF(0.0)}) {
		t.Error("-0.0 width")
	}
}

func TestBorderSideMerging(t *testing.T) {
	blue := bdBS(BorderSideOptions{Color: bdCol(0xFF0000FF)})
	blue2 := bdBS(BorderSideOptions{Color: bdCol(0xFF0000FF), Width: bdF(2.0)})
	green := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00)})
	green2 := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00), Width: bdF(2.0)})
	green3 := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00), Width: bdF(3.0)})
	green5 := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00), Width: bdF(5.0)})
	none := bdBS(BorderSideOptions{Style: bdStyle(BorderStyleNone)})
	none2 := bdBS(BorderSideOptions{Color: bdCol(0xFF0000FF), Width: bdF(2.0), Style: bdStyle(BorderStyleNone)})
	none3 := bdBS(BorderSideOptions{Style: bdStyle(BorderStyleNone), Width: bdF(3.0)})
	side2 := bdBS(BorderSideOptions{Width: bdF(2.0)})
	side3 := bdBS(BorderSideOptions{Width: bdF(3.0)})
	side5 := bdBS(BorderSideOptions{Width: bdF(5.0)})
	solid := NewBorderSide()
	yellowNone := bdBS(BorderSideOptions{Style: bdStyle(BorderStyleNone), Color: bdCol(0xFFFFFF00), Width: bdF(0.0)})
	n := BorderSideNone
	for i, c := range []struct {
		a, b BorderSide
		want bool
	}{
		{n, n, true}, {n, side2, true}, {n, yellowNone, true}, {green, blue, false}, {green2, blue2, false},
		{green2, green3, true}, {green2, none2, false}, {none3, n, true}, {none3, side2, false},
		{none3, yellowNone, true}, {side2, n, true}, {side2, none3, false}, {side2, side3, true},
		{side2, yellowNone, true}, {side3, side2, true}, {solid, none, false}, {yellowNone, side2, true},
		{yellowNone, yellowNone, true},
	} {
		if BorderSideCanMerge(c.a, c.b) != c.want {
			t.Errorf("canMerge #%d", i)
		}
	}
	for i, c := range [][3]BorderSide{
		{n, n, n}, {n, side2, side2}, {n, yellowNone, n}, {green2, green3, green5}, {none3, n, none3},
		{none3, yellowNone, none3}, {side2, n, side2}, {side2, side3, side5}, {side2, yellowNone, side2},
		{side3, side2, side5}, {yellowNone, side2, side2}, {yellowNone, yellowNone, n},
	} {
		if got := BorderSideMerge(c[0], c[1]); got != c[2] {
			t.Errorf("merge #%d: %v, want %v", i, got, c[2])
		}
	}
	for _, c := range [][2]BorderSide{{green, blue}, {green2, blue2}, {green2, none2}, {none3, side2}, {side2, none3}, {solid, none}} {
		bdExpectAssertion(t, func() { BorderSideMerge(c[0], c[1]) })
	}
}

func TestBorderSideAssertsWhenCopiedIncorrectly(t *testing.T) {
	green2 := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00), Width: bdF(2.0)})
	blue3 := bdBS(BorderSideOptions{Color: bdCol(0xFF0000FF), Width: bdF(3.0)})
	blue2 := bdBS(BorderSideOptions{Color: bdCol(0xFF0000FF), Width: bdF(2.0)})
	green3 := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00), Width: bdF(3.0)})
	none2 := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00), Width: bdF(2.0), Style: bdStyle(BorderStyleNone)})
	if green2.CopyWith(BorderSideCopyWithOptions{Color: bdCol(0xFF0000FF), Width: bdF(3.0)}) != blue3 ||
		green2.CopyWith(BorderSideCopyWithOptions{Width: bdF(3.0)}) != green3 ||
		green2.CopyWith(BorderSideCopyWithOptions{Color: bdCol(0xFF0000FF)}) != blue2 ||
		green2.CopyWith(BorderSideCopyWithOptions{Style: bdStyle(BorderStyleNone)}) != none2 {
		t.Error("copyWith")
	}
	bdExpectAssertion(t, func() { green2.CopyWith(BorderSideCopyWithOptions{Width: bdF(-1)}) })
}

func TestBorderSideScale(t *testing.T) {
	side3 := bdBS(BorderSideOptions{Width: bdF(3.0), Color: bdCol(0xFF0000FF)})
	side6 := bdBS(BorderSideOptions{Width: bdF(6.0), Color: bdCol(0xFF0000FF)})
	none := bdBS(BorderSideOptions{Style: bdStyle(BorderStyleNone), Width: bdF(0.0), Color: bdCol(0xFF0000FF)})
	if side3.Scale(2.0) != side6 || side6.Scale(0.5) != side3 || side6.Scale(0.0) != none ||
		side6.Scale(-1.0) != none || none.Scale(2.0) != none {
		t.Error("scale")
	}
}

func TestBorderSideToPaint(t *testing.T) {
	paint1 := bdBS(BorderSideOptions{Width: bdF(2.5), Color: bdCol(0xFFFFFF00)}).ToPaint()
	if paint1.StrokeWidth() != 2.5 || paint1.Style() != ui.PaintingStyleStroke ||
		paint1.Color() != ui.NewColor(0xFFFFFF00) || paint1.BlendMode() != ui.BlendModeSrcOver {
		t.Error("paint1")
	}
	paint2 := bdBS(BorderSideOptions{Width: bdF(2.5), Color: bdCol(0xFFFFFF00), Style: bdStyle(BorderStyleNone)}).ToPaint()
	if paint2.StrokeWidth() != 0.0 || paint2.Style() != ui.PaintingStyleStroke ||
		paint2.Color() != ui.NewColor(0x00000000) || paint2.BlendMode() != ui.BlendModeSrcOver {
		t.Error("paint2")
	}
}

func TestBorderSideLerpIdentical(t *testing.T) {
	border := NewBorderSide()
	if BorderSideLerp(border, border, 0.5) != border {
		t.Error("identical")
	}
}

func TestBorderSideWontLerpIntoNegativeWidths(t *testing.T) {
	side0 := bdBS(BorderSideOptions{Width: bdF(0.0)})
	side1 := NewBorderSide()
	side2 := bdBS(BorderSideOptions{Width: bdF(2.0)})
	if BorderSideLerp(side2, side1, 10.0) != BorderSideNone || BorderSideLerp(side1, side2, -10.0) != BorderSideNone ||
		BorderSideLerp(side0, side1, 2.0) != side2 || BorderSideLerp(side1, side0, 2.0) != BorderSideNone ||
		BorderSideLerp(side2, side1, 2.0) != side0 {
		t.Error("lerp")
	}
}

func TestBorderSideToString(t *testing.T) {
	got := bdBS(BorderSideOptions{Color: bdCol(0xFFAABBCC), Width: bdF(1.2345)}).String()
	want := "BorderSide(color: " + ui.NewColor(0xffaabbcc).String() + ", width: 1.2)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBorderSideLerpWithStrokeAlign(t *testing.T) {
	side0 := bdBS(BorderSideOptions{Width: bdF(2.0)})
	side1 := bdBS(BorderSideOptions{Width: bdF(2.0), StrokeAlign: bdF(BorderSideStrokeAlignOutside)})
	if BorderSideLerp(side0, side1, 0) != side0 ||
		BorderSideLerp(side0, side1, 0.5) != bdBS(BorderSideOptions{Width: bdF(2.0), StrokeAlign: bdF(BorderSideStrokeAlignCenter)}) ||
		BorderSideLerp(side0, side1, 1) != side1 {
		t.Error("outside")
	}
	side3 := bdBS(BorderSideOptions{Width: bdF(2.0), StrokeAlign: bdF(BorderSideStrokeAlignCenter)})
	if BorderSideLerp(side0, side3, 0) != side0 ||
		BorderSideLerp(side0, side3, 0.5) != bdBS(BorderSideOptions{Width: bdF(2.0), StrokeAlign: bdF(-0.5)}) ||
		BorderSideLerp(side0, side3, 1) != side3 {
		t.Error("center")
	}
}
