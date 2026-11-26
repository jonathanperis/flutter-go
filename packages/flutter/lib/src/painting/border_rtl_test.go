package painting_test

// Port of packages/flutter/test/painting/border_rtl_test.dart (Flutter
// 3.50.1).
//
// Skipped: "BorderDirectional hashCode" (Go values have no hashCode; equal
// values compare equal with ==), and "BoxDecoration.border takes a
// BorderDirectional" (BoxDecoration is not ported yet; it belongs to
// box_decoration.dart).

import (
	"math"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// sillyBorder is the test's SillyBorder: a BoxBorder whose every member
// returns null (Dart's noSuchMethod).
type sillyBorder struct{}

func (sillyBorder) Top() BorderSide                                                { return BorderSide{} }
func (sillyBorder) Bottom() BorderSide                                             { return BorderSide{} }
func (sillyBorder) IsUniform() bool                                                { return false }
func (sillyBorder) PaintBox(ui.Canvas, ui.Rect, BoxBorderPaintOptions)             {}
func (sillyBorder) Dimensions() EdgeInsetsGeometry                                 { return nil }
func (sillyBorder) AddBorder(ShapeBorder, bool) ShapeBorder                        { return nil }
func (sillyBorder) Add(ShapeBorder) ShapeBorder                                    { return nil }
func (sillyBorder) Scale(float64) ShapeBorder                                      { return nil }
func (sillyBorder) LerpFrom(ShapeBorder, float64) ShapeBorder                      { return nil }
func (sillyBorder) LerpTo(ShapeBorder, float64) ShapeBorder                        { return nil }
func (sillyBorder) GetOuterPath(ui.Rect, *ui.TextDirection) *ui.Path               { return nil }
func (sillyBorder) GetInnerPath(ui.Rect, *ui.TextDirection) *ui.Path               { return nil }
func (sillyBorder) HitTest(ui.Rect, ui.Offset, *ui.TextDirection) bool             { return false }
func (sillyBorder) PaintInterior(ui.Canvas, ui.Rect, *ui.Paint, *ui.TextDirection) {}
func (sillyBorder) PreferPaintInterior() bool                                      { return false }
func (sillyBorder) Paint(ui.Canvas, ui.Rect, *ui.TextDirection)                    {}
func (sillyBorder) String() string                                                 { return "SillyBorder" }

func sideMatches(x, y BorderSide) bool {
	const limit = 1.0 / 255
	return math.Abs(x.Color.A-y.Color.A) < limit && math.Abs(x.Color.R-y.Color.R) < limit &&
		math.Abs(x.Color.G-y.Color.G) < limit && math.Abs(x.Color.B-y.Color.B) < limit &&
		x.Width == y.Width && x.Style == y.Style && x.StrokeAlign == y.StrokeAlign
}

func borderMatches(item, target ShapeBorder) bool {
	if a, ok := item.(Border); ok {
		b, ok := target.(Border)
		return ok && sideMatches(a.Top(), b.Top()) && sideMatches(a.Right(), b.Right()) &&
			sideMatches(a.Bottom(), b.Bottom()) && sideMatches(a.Left(), b.Left())
	}
	if a, ok := item.(BorderDirectional); ok {
		b, ok := target.(BorderDirectional)
		return ok && sideMatches(a.Top(), b.Top()) && sideMatches(a.Start(), b.Start()) &&
			sideMatches(a.Bottom(), b.Bottom()) && sideMatches(a.End(), b.End())
	}
	return false
}

func side(o BorderSideOptions) *BorderSide { s := bdBS(o); return &s }

func w(v float64) *BorderSide { return side(BorderSideOptions{Width: bdF(v)}) }

func wc(v float64, c int) *BorderSide { return side(BorderSideOptions{Width: bdF(v), Color: bdCol(c)}) }

func dirB(o BorderDirectionalOptions) BoxBorder { return NewBorderDirectional(o) }

func visB(o BorderOptions) BoxBorder { return NewBorder(o) }

func TestBoxBorderLerp(t *testing.T) {
	directionalWithMagentaTop5 := dirB(BorderDirectionalOptions{Top: wc(5, 0xFFFF00FF)})
	directionalWithMagentaTop5At25 := dirB(BorderDirectionalOptions{Top: wc(1.25, 0x3F3F003F)})
	directionalWithMagentaTop5At75 := dirB(BorderDirectionalOptions{Top: wc(3.75, 0xBFBF00BF)})
	directionalWithSides10 := dirB(BorderDirectionalOptions{Start: w(10), End: w(20)})
	directionalWithSides10At25 := dirB(BorderDirectionalOptions{Start: wc(2.5, 0x3F000000), End: wc(5, 0x3F000000)})
	directionalWithSides10At50 := dirB(BorderDirectionalOptions{Start: wc(5, 0x7F000000), End: wc(10, 0x7F000000)})
	directionalWithSides10At75 := dirB(BorderDirectionalOptions{Start: wc(7.5, 0xBF000000), End: wc(15, 0xBF000000)})
	directionalWithSides20 := dirB(BorderDirectionalOptions{Start: w(20), End: w(40)})
	directionalWithSides30 := dirB(BorderDirectionalOptions{Start: w(30), End: w(60)})
	directionalWithTop10 := dirB(BorderDirectionalOptions{Top: w(10)})
	directionalWithYellowTop10 := dirB(BorderDirectionalOptions{Top: wc(10, 0xFFFFFF00)})
	directionalWithYellowTop5 := dirB(BorderDirectionalOptions{Top: wc(5, 0xFFFFFF00)})
	visualWithMagentaTop10 := visB(BorderOptions{Top: wc(10, 0xFFFF00FF)})
	visualWithMagentaTop5 := visB(BorderOptions{Top: wc(5, 0xFFFF00FF)})
	visualWithSides10 := visB(BorderOptions{Left: w(10), Right: w(20)})
	visualWithSides10At25 := visB(BorderOptions{Left: wc(2.5, 0x3F000000), Right: wc(5, 0x3F000000)})
	visualWithSides10At50 := visB(BorderOptions{Left: wc(5, 0x7F000000), Right: wc(10, 0x7F000000)})
	visualWithSides10At75 := visB(BorderOptions{Left: wc(7.5, 0xBF000000), Right: wc(15, 0xBF000000)})
	visualWithSides20 := visB(BorderOptions{Left: w(20), Right: w(40)})
	visualWithSides30 := visB(BorderOptions{Left: w(30), Right: w(60)})
	visualWithTop10 := visB(BorderOptions{Top: w(10)})
	visualWithTop100 := visB(BorderOptions{Top: w(100)})
	visualWithTop190 := visB(BorderOptions{Top: w(190)})
	visualWithYellowTop5 := visB(BorderOptions{Top: wc(5, 0xFFFFFF00)})
	visualWithYellowTop5At25 := visB(BorderOptions{Top: wc(1.25, 0x3F3F3F00)})
	visualWithYellowTop5At75 := visB(BorderOptions{Top: wc(3.75, 0xBFBFBF00)})
	all := func(v float64) BoxBorder { return NewBorderAll(BorderSideOptions{Width: bdF(v)}) }
	allNone := NewBorderAll(BorderSideOptions{Width: bdF(0), Style: bdStyle(BorderStyleNone)})
	silly := sillyBorder{}
	colorLerp := func(t float64) BoxBorder {
		a, b := ui.NewColor(0xFFFFFF00), ui.NewColor(0xFFFF00FF)
		c := *ui.ColorLerp(&a, &b, t)
		return visB(BorderOptions{Top: side(BorderSideOptions{Width: bdF(5), Color: &c})})
	}

	type lerpCase struct {
		a, b  BoxBorder
		t     float64
		want  BoxBorder
		match bool // _matchesBorder/_matchesBorderDirectional (colors within 1/255)
	}
	cases := []lerpCase{
		{nil, nil, -1, nil, false},
		{all(10), nil, -1, all(20), false},
		{nil, all(10), -1, allNone, false},
		{directionalWithTop10, nil, -1, dirB(BorderDirectionalOptions{Top: w(20)}), false},
		{nil, directionalWithTop10, -1, dirB(BorderDirectionalOptions{}), false},
		{directionalWithTop10, visualWithTop100, -1, visB(BorderOptions{}), false},
		{visualWithSides10, directionalWithMagentaTop5, -1, visualWithSides20, false},
		{visualWithYellowTop5, directionalWithMagentaTop5, -1, visB(BorderOptions{Top: wc(5, 0xFFFFFF00)}), false},
		{visualWithSides10, directionalWithSides10, -1, visualWithSides30, false},
		{visualWithYellowTop5, directionalWithSides10, -1, directionalWithYellowTop10, false},
		{silly, visB(BorderOptions{}), -1, silly, false},

		{nil, nil, 0, nil, false},
		{all(10), nil, 0, all(10), false},
		{nil, all(10), 0, visB(BorderOptions{}), false},
		{directionalWithTop10, nil, 0, dirB(BorderDirectionalOptions{Top: w(10)}), false},
		{nil, directionalWithTop10, 0, dirB(BorderDirectionalOptions{}), false},
		{directionalWithTop10, visualWithTop100, 0, visualWithTop10, false},
		{visualWithSides10, directionalWithMagentaTop5, 0, visualWithSides10, false},
		{visualWithYellowTop5, directionalWithMagentaTop5, 0, visB(BorderOptions{Top: wc(5, 0xFFFFFF00)}), false},
		{visualWithSides10, directionalWithSides10, 0, visualWithSides10, false},
		{visualWithYellowTop5, directionalWithSides10, 0, directionalWithYellowTop5, false},
		{silly, visB(BorderOptions{}), 0, silly, false},

		{nil, nil, 0.25, nil, false},
		{all(10), nil, 0.25, all(7.5), false},
		{nil, all(10), 0.25, all(2.5), false},
		{directionalWithTop10, nil, 0.25, dirB(BorderDirectionalOptions{Top: w(7.5)}), false},
		{nil, directionalWithTop10, 0.25, dirB(BorderDirectionalOptions{Top: w(2.5)}), false},
		{directionalWithTop10, visualWithTop100, 0.25, visB(BorderOptions{Top: w(32.5)}), false},
		{visualWithSides10, directionalWithMagentaTop5, 0.25,
			visualWithSides10At75.Add(directionalWithMagentaTop5At25).(BoxBorder), true},
		{visualWithYellowTop5, directionalWithMagentaTop5, 0.25, colorLerp(0.25), false},
		{visualWithSides10, directionalWithSides10, 0.25, visualWithSides10At50, true},
		{visualWithYellowTop5, directionalWithSides10, 0.25,
			visualWithYellowTop5At75.Add(directionalWithSides10At25).(BoxBorder), true},
		{silly, visB(BorderOptions{}), 0.25, silly, false},

		{nil, nil, 0.75, nil, false},
		{all(10), nil, 0.75, all(2.5), false},
		{nil, all(10), 0.75, all(7.5), false},
		{directionalWithTop10, nil, 0.75, dirB(BorderDirectionalOptions{Top: w(2.5)}), false},
		{nil, directionalWithTop10, 0.75, dirB(BorderDirectionalOptions{Top: w(7.5)}), false},
		{directionalWithTop10, visualWithTop100, 0.75, visB(BorderOptions{Top: w(77.5)}), false},
		{visualWithSides10, directionalWithMagentaTop5, 0.75,
			visualWithSides10At25.Add(directionalWithMagentaTop5At75).(BoxBorder), true},
		{visualWithYellowTop5, directionalWithMagentaTop5, 0.75, colorLerp(0.75), false},
		{visualWithSides10, directionalWithSides10, 0.75, directionalWithSides10At50, true},
		{visualWithYellowTop5, directionalWithSides10, 0.75,
			visualWithYellowTop5At25.Add(directionalWithSides10At75).(BoxBorder), true},
		{silly, visB(BorderOptions{}), 0.75, allNone, false},

		{nil, nil, 1, nil, false},
		{all(10), nil, 1, allNone, false},
		{nil, all(10), 1, all(10), false},
		{directionalWithTop10, nil, 1, dirB(BorderDirectionalOptions{}), false},
		{nil, directionalWithTop10, 1, dirB(BorderDirectionalOptions{Top: w(10)}), false},
		{directionalWithTop10, visualWithTop100, 1, visualWithTop100, false},
		{visualWithSides10, directionalWithMagentaTop5, 1, visualWithMagentaTop5, false},
		{visualWithYellowTop5, directionalWithMagentaTop5, 1, visualWithMagentaTop5, false},
		{visualWithSides10, directionalWithSides10, 1, directionalWithSides10, false},
		{visualWithYellowTop5, directionalWithSides10, 1, directionalWithSides10, false},
		{silly, visB(BorderOptions{}), 1, allNone, false},

		{nil, nil, 2, nil, false},
		{all(10), nil, 2, allNone, false},
		{nil, all(10), 2, all(20), false},
		{directionalWithTop10, nil, 2, dirB(BorderDirectionalOptions{}), false},
		{nil, directionalWithTop10, 2, dirB(BorderDirectionalOptions{Top: w(20)}), false},
		{directionalWithTop10, visualWithTop100, 2, visualWithTop190, false},
		{visualWithSides10, directionalWithMagentaTop5, 2, visualWithMagentaTop10, false},
		{visualWithYellowTop5, directionalWithMagentaTop5, 2, visualWithMagentaTop5, false},
		{visualWithSides10, directionalWithSides10, 2, directionalWithSides30, false},
		{visualWithYellowTop5, directionalWithSides10, 2, directionalWithSides20, false},
		{silly, visB(BorderOptions{}), 2, allNone, false},
	}
	for i, c := range cases {
		got := BoxBorderLerp(c.a, c.b, c.t)
		ok := false
		switch {
		case c.match:
			ok = borderMatches(got, c.want)
		case c.want == nil:
			ok = got == nil
		default:
			ok = got == c.want
		}
		if !ok {
			t.Errorf("#%d lerp(%v, %v, %v) = %v, want %v", i, c.a, c.b, c.t, got, c.want)
		}
	}
}

var rtlPathExcludesOuter = bdOffsets(40, 60, 50, 50, 111, 190, 110, 191, 111, 191, 0, 0, -10, -10, 0, -10, -10, 0, 1000, 1000)

func TestBoxBorderGetInnerPathGetOuterPath(t *testing.T) {
	border := NewBorder(BorderOptions{Top: w(10), Right: w(20)})
	borderDirectional := NewBorderDirectional(BorderDirectionalOptions{Top: w(10), End: w(20)})
	rect := ui.NewRectFromLTRB(50.0, 60.0, 110.0, 190.0)
	outer := ft.IsPathThat(ft.IsPathThatOptions{
		Includes: bdOffsets(50, 60, 60, 60, 60, 70, 80, 190, 109, 189, 110, 80, 110, 190),
		Excludes: rtlPathExcludesOuter,
	})
	// inner path is a rect from 50.0,70.0 to 90.0,190.0
	innerLeft := ft.IsPathThat(ft.IsPathThatOptions{
		Includes: bdOffsets(50, 70, 55, 70, 50, 75, 70, 70, 70, 71, 71, 70, 71, 71, 80, 180, 80, 190, 89, 189, 90, 190),
		Excludes: bdOffsets(40, 60, 50, 50, 50, 60, 60, 60, 0, 0, -10, -10, 0, -10, -10, 0, 110, 80, 89, 191, 90, 191,
			91, 189, 91, 190, 91, 191, 109, 189, 110, 190, 1000, 1000),
	})
	// inner path is a rect from 70.0,70.0 to 110.0,190.0
	innerRight := ft.IsPathThat(ft.IsPathThatOptions{
		Includes: bdOffsets(70, 70, 70, 71, 71, 70, 71, 71, 80, 180, 80, 190, 89, 189, 90, 190, 91, 189, 91, 190,
			109, 189, 110, 80, 110, 190),
		Excludes: bdOffsets(40, 60, 50, 50, 50, 60, 50, 70, 50, 75, 55, 70, 60, 60, 0, 0, -10, -10, 0, -10, -10, 0,
			89, 191, 90, 191, 91, 191, 110, 191, 111, 190, 111, 191, 1000, 1000),
	})
	ft.Expect(t, border.GetOuterPath(rect, bdRtl), outer)
	ft.Expect(t, border.GetInnerPath(rect, bdRtl), innerLeft)
	ft.Expect(t, borderDirectional.GetOuterPath(rect, bdRtl), outer)
	ft.Expect(t, borderDirectional.GetInnerPath(rect, bdRtl), innerRight)
	ft.Expect(t, borderDirectional.GetOuterPath(rect, bdLtr), outer)
	ft.Expect(t, borderDirectional.GetInnerPath(rect, bdLtr), innerLeft)
}

func TestBorderDirectionalMerge(t *testing.T) {
	d := func(o BorderDirectionalOptions) BorderDirectional { return NewBorderDirectional(o) }
	if BorderDirectionalMerge(d(BorderDirectionalOptions{Top: &yellow2}), d(BorderDirectionalOptions{End: &magenta3})) !=
		d(BorderDirectionalOptions{Top: &yellow2, End: &magenta3}) {
		t.Error("1")
	}
	if BorderDirectionalMerge(d(BorderDirectionalOptions{Bottom: &magenta3}), d(BorderDirectionalOptions{Bottom: &magenta3})) !=
		d(BorderDirectionalOptions{Bottom: &magenta6}) {
		t.Error("2")
	}
	if BorderDirectionalMerge(d(BorderDirectionalOptions{Start: &magenta3, End: &yellowNone0}), d(BorderDirectionalOptions{End: &yellow2})) !=
		d(BorderDirectionalOptions{Start: &magenta3, End: &yellow2}) {
		t.Error("3")
	}
	if BorderDirectionalMerge(d(BorderDirectionalOptions{}), d(BorderDirectionalOptions{})) != d(BorderDirectionalOptions{}) {
		t.Error("4")
	}
	bdExpectAssertion(t, func() {
		BorderDirectionalMerge(d(BorderDirectionalOptions{Start: &magenta3}), d(BorderDirectionalOptions{Start: &yellow2}))
	})
}

func TestBorderDirectionalDimensions(t *testing.T) {
	got := NewBorderDirectional(BorderDirectionalOptions{Top: w(3), Start: w(2), End: w(7), Bottom: w(5)}).Dimensions()
	if got != EdgeInsetsGeometry(NewEdgeInsetsDirectionalFromSTEB(2.0, 3.0, 7.0, 5.0)) {
		t.Errorf("got %v", got)
	}
}

func TestBorderDirectionalIsUniform(t *testing.T) {
	col := func(c int) *BorderSide { return side(BorderSideOptions{Color: bdCol(c)}) }
	none := func() *BorderSide { return side(BorderSideOptions{Style: bdStyle(BorderStyleNone)}) }
	for i, c := range []struct {
		b    BorderDirectional
		want bool
	}{
		{NewBorderDirectional(BorderDirectionalOptions{Top: w(3), Start: w(3), End: w(3), Bottom: w(3.1)}), false},
		{NewBorderDirectional(BorderDirectionalOptions{Top: w(3), Start: w(3), End: w(3), Bottom: w(3)}), true},
		{NewBorderDirectional(BorderDirectionalOptions{Top: col(0xFFFFFFFF), Start: col(0xFFFFFFFE), End: col(0xFFFFFFFF), Bottom: col(0xFFFFFFFF)}), false},
		{NewBorderDirectional(BorderDirectionalOptions{Top: col(0xFFFFFFFF), Start: col(0xFFFFFFFF), End: col(0xFFFFFFFF), Bottom: col(0xFFFFFFFF)}), true},
		{NewBorderDirectional(BorderDirectionalOptions{Top: none(), Start: none(), End: none(), Bottom: w(0)}), false},
		{NewBorderDirectional(BorderDirectionalOptions{Top: none(), Start: none(), End: none()}), false},
		{NewBorderDirectional(), true},
	} {
		if c.b.IsUniform() != c.want {
			t.Errorf("#%d", i)
		}
	}
}

func TestBorderDirectionalAddAllDirectional(t *testing.T) {
	d := func(o BorderDirectionalOptions) ShapeBorder { return NewBorderDirectional(o) }
	eq := func(name string, got, want ShapeBorder) {
		t.Helper()
		if !ShapeBorderEqual(got, want) {
			t.Errorf("%s: got %v", name, got)
		}
	}
	eq("1", d(BorderDirectionalOptions{Top: &yellow2}).Add(d(BorderDirectionalOptions{End: &magenta3})),
		d(BorderDirectionalOptions{Top: &yellow2, End: &magenta3}))
	eq("2", d(BorderDirectionalOptions{Bottom: &magenta3}).Add(d(BorderDirectionalOptions{Bottom: &magenta3})),
		d(BorderDirectionalOptions{Bottom: &magenta6}))
	eq("3", d(BorderDirectionalOptions{Start: &magenta3, End: &yellowNone0}).Add(d(BorderDirectionalOptions{End: &yellow2})),
		d(BorderDirectionalOptions{Start: &magenta3, End: &yellow2}))
	eq("4", d(BorderDirectionalOptions{}).Add(d(BorderDirectionalOptions{})), d(BorderDirectionalOptions{}))
	if _, ok := d(BorderDirectionalOptions{Start: &magenta3}).Add(d(BorderDirectionalOptions{Start: &yellow2})).(BorderDirectional); ok {
		t.Error("isNot(isA<BorderDirectional>())")
	}
	b3, b6 := d(BorderDirectionalOptions{Top: &magenta3}), d(BorderDirectionalOptions{Top: &magenta6})
	eq("b3", b3.Add(b3), b6)
	b0, bZ := d(BorderDirectionalOptions{Top: &yellowNone0}), d(BorderDirectionalOptions{})
	eq("b0+b0", b0.Add(b0), bZ)
	eq("bZ+bZ", bZ.Add(bZ), bZ)
	eq("b0+bZ", b0.Add(bZ), bZ)
	eq("bZ+b0", bZ.Add(b0), bZ)
}

func TestBorderDirectionalAdd(t *testing.T) {
	side1 := bdBS(BorderSideOptions{Color: bdCol(0x11111111)})
	doubleSide1 := bdBS(BorderSideOptions{Color: bdCol(0x11111111), Width: bdF(2.0)})
	side2 := bdBS(BorderSideOptions{Color: bdCol(0x22222222)})
	doubleSide2 := bdBS(BorderSideOptions{Color: bdCol(0x22222222), Width: bdF(2.0)})
	v := func(o BorderOptions) ShapeBorder { return NewBorder(o) }
	d := func(o BorderDirectionalOptions) ShapeBorder { return NewBorderDirectional(o) }
	eq := func(name string, got, want ShapeBorder) {
		t.Helper()
		if !ShapeBorderEqual(got, want) {
			t.Errorf("%s: got %v", name, got)
		}
	}
	// adding tops and sides
	eq("1", v(BorderOptions{Left: &side1}).Add(d(BorderDirectionalOptions{Top: &side2})), v(BorderOptions{Left: &side1, Top: &side2}))
	eq("2", d(BorderDirectionalOptions{Start: &side1}).Add(v(BorderOptions{Top: &side2})), d(BorderDirectionalOptions{Start: &side1, Top: &side2}))
	eq("3", v(BorderOptions{Top: &side2}).Add(d(BorderDirectionalOptions{Start: &side1})), d(BorderDirectionalOptions{Start: &side1, Top: &side2}))
	eq("4", d(BorderDirectionalOptions{Top: &side2}).Add(v(BorderOptions{Left: &side1})), v(BorderOptions{Left: &side1, Top: &side2}))
	// adding incompatible tops and bottoms
	for i, s := range []ShapeBorder{
		v(BorderOptions{Top: &side1}).Add(d(BorderDirectionalOptions{Top: &side2})),
		d(BorderDirectionalOptions{Top: &side2}).Add(v(BorderOptions{Top: &side1})),
		v(BorderOptions{Bottom: &side1}).Add(d(BorderDirectionalOptions{Bottom: &side2})),
		d(BorderDirectionalOptions{Bottom: &side2}).Add(v(BorderOptions{Bottom: &side1})),
	} {
		if !strings.Contains(s.String(), " + ") {
			t.Errorf("incompatible #%d: %v", i, s)
		}
	}
	// adding compatible tops and bottoms
	eq("5", d(BorderDirectionalOptions{Top: &side1}).Add(v(BorderOptions{Top: &side1})), v(BorderOptions{Top: &doubleSide1}))
	eq("6", v(BorderOptions{Top: &side1}).Add(d(BorderDirectionalOptions{Top: &side1})), v(BorderOptions{Top: &doubleSide1}))
	eq("7", d(BorderDirectionalOptions{Bottom: &side1}).Add(v(BorderOptions{Bottom: &side1})), v(BorderOptions{Bottom: &doubleSide1}))
	eq("8", v(BorderOptions{Bottom: &side1}).Add(d(BorderDirectionalOptions{Bottom: &side1})), v(BorderOptions{Bottom: &doubleSide1}))

	borderWithLeft := v(BorderOptions{Left: &side1, Top: &side2, Bottom: &side2})
	borderWithRight := v(BorderOptions{Right: &side1, Top: &side2, Bottom: &side2})
	borderWithoutSides := v(BorderOptions{Top: &side2, Bottom: &side2})
	borderDirectionalWithStart := d(BorderDirectionalOptions{Start: &side1, Top: &side2, Bottom: &side2})
	borderDirectionalWithEnd := d(BorderDirectionalOptions{End: &side1, Top: &side2, Bottom: &side2})
	borderDirectionalWithoutSides := d(BorderDirectionalOptions{Top: &side2, Bottom: &side2})
	plus := func(a, b ShapeBorder) string { return a.String() + " + " + b.String() }
	for i, c := range []struct {
		got  ShapeBorder
		want string
	}{
		{borderWithLeft.Add(borderDirectionalWithStart), plus(borderWithLeft, borderDirectionalWithStart)},
		{borderWithLeft.Add(borderDirectionalWithEnd), plus(borderWithLeft, borderDirectionalWithEnd)},
		{borderWithLeft.Add(borderDirectionalWithoutSides), v(BorderOptions{Left: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderWithRight.Add(borderDirectionalWithStart), plus(borderWithRight, borderDirectionalWithStart)},
		{borderWithRight.Add(borderDirectionalWithEnd), plus(borderWithRight, borderDirectionalWithEnd)},
		{borderWithRight.Add(borderDirectionalWithoutSides), v(BorderOptions{Right: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderWithoutSides.Add(borderDirectionalWithStart), d(BorderDirectionalOptions{Start: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderWithoutSides.Add(borderDirectionalWithEnd), d(BorderDirectionalOptions{End: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderWithoutSides.Add(borderDirectionalWithoutSides), v(BorderOptions{Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderDirectionalWithStart.Add(borderWithLeft), plus(borderDirectionalWithStart, borderWithLeft)},
		{borderDirectionalWithEnd.Add(borderWithLeft), plus(borderDirectionalWithEnd, borderWithLeft)},
		{borderDirectionalWithoutSides.Add(borderWithLeft), v(BorderOptions{Left: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderDirectionalWithStart.Add(borderWithRight), plus(borderDirectionalWithStart, borderWithRight)},
		{borderDirectionalWithEnd.Add(borderWithRight), plus(borderDirectionalWithEnd, borderWithRight)},
		{borderDirectionalWithoutSides.Add(borderWithRight), v(BorderOptions{Right: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderDirectionalWithStart.Add(borderWithoutSides), d(BorderDirectionalOptions{Start: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderDirectionalWithEnd.Add(borderWithoutSides), d(BorderDirectionalOptions{End: &side1, Top: &doubleSide2, Bottom: &doubleSide2}).String()},
		{borderDirectionalWithoutSides.Add(borderWithoutSides), v(BorderOptions{Top: &doubleSide2, Bottom: &doubleSide2}).String()},
	} {
		if c.got.String() != c.want {
			t.Errorf("#%d: got %q, want %q", i, c.got.String(), c.want)
		}
	}
}

func TestBorderDirectionalScale(t *testing.T) {
	b3 := NewBorderDirectional(BorderDirectionalOptions{Start: &magenta3})
	b6 := NewBorderDirectional(BorderDirectionalOptions{Start: &magenta6})
	bY0 := NewBorderDirectional(BorderDirectionalOptions{Top: &yellowNone0})
	bY2 := NewBorderDirectional(BorderDirectionalOptions{Top: &yellow2})
	if b3.Scale(2.0) != ShapeBorder(b6) || bY0.Scale(3.0) != ShapeBorder(bY0) || bY2.Scale(0.0) != ShapeBorder(bY0) {
		t.Error("scale")
	}
}

func TestBorderDirectionalLerp(t *testing.T) {
	d := func(o BorderDirectionalOptions) BorderDirectional { return NewBorderDirectional(o) }
	top := func(v float64) BorderDirectional { return d(BorderDirectionalOptions{Top: w(v)}) }
	se := func(s, e float64) BorderDirectional { return d(BorderDirectionalOptions{Start: w(s), End: w(e)}) }
	directionalWithTop10 := top(10)
	at0, at100 := se(100, 200), se(200, 100)
	zero := d(BorderDirectionalOptions{})
	for _, c := range []struct {
		t                    float64
		aNil, nilB, interior BorderDirectional
	}{
		{-1, top(20), zero, se(0, 300)},
		{0, top(10), zero, at0},
		{0.25, top(7.5), top(2.5), se(125, 175)},
		{0.75, top(2.5), top(7.5), se(175, 125)},
		{1, zero, top(10), at100},
		{2, zero, top(20), se(300, 0)},
	} {
		if BorderDirectionalLerp(nil, nil, c.t) != nil {
			t.Errorf("%v: null", c.t)
		}
		if got := *BorderDirectionalLerp(&directionalWithTop10, nil, c.t); got != c.aNil {
			t.Errorf("%v: a,null %v", c.t, got)
		}
		if got := *BorderDirectionalLerp(nil, &directionalWithTop10, c.t); got != c.nilB {
			t.Errorf("%v: null,b %v", c.t, got)
		}
		if got := *BorderDirectionalLerp(&at0, &at100, c.t); got != c.interior {
			t.Errorf("%v: %v", c.t, got)
		}
	}
}

func TestBorderDirectionalPaint(t *testing.T) {
	border := NewBorderDirectional(BorderDirectionalOptions{End: wc(10.0, 0xFF00FF00)})
	rect := ui.NewRectFromLTRB(10.0, 20.0, 30.0, 40.0)
	green := ui.NewColor(0xFF00FF00)
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, rect, bdRtl) },
		ft.Paints().Path(ft.PaintPatternPathOptions{Includes: bdOffsets(15, 30), Excludes: bdOffsets(25, 30),
			PaintOptions: ft.PaintOptions{Color: &green}}))
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, rect, bdLtr) },
		ft.Paints().Path(ft.PaintPatternPathOptions{Includes: bdOffsets(25, 30), Excludes: bdOffsets(15, 30),
			PaintOptions: ft.PaintOptions{Color: &green}}))
	ft.Expect(t, func(c ui.Canvas) { border.Paint(c, rect, nil) }, ft.PaintsAssertion()) // no TextDirection
}
