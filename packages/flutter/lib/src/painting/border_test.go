package painting_test

// Port of packages/flutter/test/painting/border_test.dart (Flutter
// 3.50.1).
//
// Skipped: "Non-Uniform Border variations" pumps DecoratedBox widgets
// (widgets/rendering are not ported); its paint-level assertions are
// covered by TestBorderNonUniformVariationsPaint below, which paints the
// same borders directly (as BoxDecoration's painter does). Dart's same()
// on the sides is Go's == on the values.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestBorderFromBorderSide(t *testing.T) {
	side := NewBorderSide()
	border := NewBorderFromBorderSide(side)
	if border.Left() != side || border.Top() != side || border.Right() != side || border.Bottom() != side {
		t.Error("sides")
	}
}

func TestBorderSymmetric(t *testing.T) {
	side1 := bdBS(BorderSideOptions{Color: bdCol(0xFFFFFFFF)})
	side2 := NewBorderSide()
	border := NewBorderSymmetric(BorderSymmetricOptions{Vertical: &side1, Horizontal: &side2})
	if border.Left() != side1 || border.Top() != side2 || border.Right() != side1 || border.Bottom() != side2 {
		t.Error("sides")
	}
}

var (
	magenta3    = bdBS(BorderSideOptions{Color: bdCol(0xFFFF00FF), Width: bdF(3.0)})
	magenta6    = bdBS(BorderSideOptions{Color: bdCol(0xFFFF00FF), Width: bdF(6.0)})
	yellow2     = bdBS(BorderSideOptions{Color: bdCol(0xFFFFFF00), Width: bdF(2.0)})
	yellowNone0 = bdBS(BorderSideOptions{Color: bdCol(0xFFFFFF00), Width: bdF(0.0), Style: bdStyle(BorderStyleNone)})
)

func TestBorderMerge(t *testing.T) {
	if BorderMerge(NewBorder(BorderOptions{Top: &yellow2}), NewBorder(BorderOptions{Right: &magenta3})) !=
		NewBorder(BorderOptions{Top: &yellow2, Right: &magenta3}) {
		t.Error("1")
	}
	if BorderMerge(NewBorder(BorderOptions{Bottom: &magenta3}), NewBorder(BorderOptions{Bottom: &magenta3})) !=
		NewBorder(BorderOptions{Bottom: &magenta6}) {
		t.Error("2")
	}
	if BorderMerge(NewBorder(BorderOptions{Left: &magenta3, Right: &yellowNone0}), NewBorder(BorderOptions{Right: &yellow2})) !=
		NewBorder(BorderOptions{Left: &magenta3, Right: &yellow2}) {
		t.Error("3")
	}
	if BorderMerge(NewBorder(), NewBorder()) != NewBorder() {
		t.Error("4")
	}
	bdExpectAssertion(t, func() {
		BorderMerge(NewBorder(BorderOptions{Left: &magenta3}), NewBorder(BorderOptions{Left: &yellow2}))
	})
}

func TestBorderAdd(t *testing.T) {
	eq := func(name string, got ShapeBorder, want ShapeBorder) {
		t.Helper()
		if !ShapeBorderEqual(got, want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
	eq("1", NewBorder(BorderOptions{Top: &yellow2}).Add(NewBorder(BorderOptions{Right: &magenta3})),
		NewBorder(BorderOptions{Top: &yellow2, Right: &magenta3}))
	eq("2", NewBorder(BorderOptions{Bottom: &magenta3}).Add(NewBorder(BorderOptions{Bottom: &magenta3})),
		NewBorder(BorderOptions{Bottom: &magenta6}))
	eq("3", NewBorder(BorderOptions{Left: &magenta3, Right: &yellowNone0}).Add(NewBorder(BorderOptions{Right: &yellow2})),
		NewBorder(BorderOptions{Left: &magenta3, Right: &yellow2}))
	eq("4", NewBorder().Add(NewBorder()), NewBorder())
	if _, ok := NewBorder(BorderOptions{Left: &magenta3}).Add(NewBorder(BorderOptions{Left: &yellow2})).(Border); ok {
		t.Error("isNot(isA<Border>())")
	}
	b3 := NewBorder(BorderOptions{Top: &magenta3})
	b6 := NewBorder(BorderOptions{Top: &magenta6})
	eq("b3+b3", b3.Add(b3), b6)
	b0 := NewBorder(BorderOptions{Top: &yellowNone0})
	bZ := NewBorder()
	eq("b0+b0", b0.Add(b0), bZ)
	eq("bZ+bZ", bZ.Add(bZ), bZ)
	eq("b0+bZ", b0.Add(bZ), bZ)
	eq("bZ+b0", bZ.Add(b0), bZ)
}

func TestBorderScale(t *testing.T) {
	b3 := NewBorder(BorderOptions{Left: &magenta3})
	b6 := NewBorder(BorderOptions{Left: &magenta6})
	if b3.Scale(2.0) != ShapeBorder(b6) {
		t.Error("b3")
	}
	bY0 := NewBorder(BorderOptions{Top: &yellowNone0})
	if bY0.Scale(3.0) != ShapeBorder(bY0) {
		t.Error("bY0")
	}
	bY2 := NewBorder(BorderOptions{Top: &yellow2})
	if bY2.Scale(0.0) != ShapeBorder(bY0) {
		t.Error("bY2")
	}
}

func TestBorderDimensions(t *testing.T) {
	w := func(v float64) *BorderSide { s := bdBS(BorderSideOptions{Width: bdF(v)}); return &s }
	got := NewBorder(BorderOptions{Left: w(2), Top: w(3), Bottom: w(5), Right: w(7)}).Dimensions()
	if got != EdgeInsetsGeometry(NewEdgeInsetsFromLTRB(2.0, 3.0, 7.0, 5.0)) {
		t.Errorf("got %v", got)
	}
}

func TestBorderIsUniform(t *testing.T) {
	s := func(o BorderSideOptions) *BorderSide { x := bdBS(o); return &x }
	none := func() *BorderSide { return s(BorderSideOptions{Style: bdStyle(BorderStyleNone)}) }
	for i, c := range []struct {
		b    Border
		want bool
	}{
		{NewBorder(BorderOptions{Left: s(BorderSideOptions{Width: bdF(3)}), Top: s(BorderSideOptions{Width: bdF(3)}),
			Right: s(BorderSideOptions{Width: bdF(3)}), Bottom: s(BorderSideOptions{Width: bdF(3.1)})}), false},
		{NewBorder(BorderOptions{Left: s(BorderSideOptions{Width: bdF(3)}), Top: s(BorderSideOptions{Width: bdF(3)}),
			Right: s(BorderSideOptions{Width: bdF(3)}), Bottom: s(BorderSideOptions{Width: bdF(3)})}), true},
		{NewBorder(BorderOptions{Left: s(BorderSideOptions{Color: bdCol(0xFFFFFFFE)}), Top: s(BorderSideOptions{Color: bdCol(0xFFFFFFFF)}),
			Right: s(BorderSideOptions{Color: bdCol(0xFFFFFFFF)}), Bottom: s(BorderSideOptions{Color: bdCol(0xFFFFFFFF)})}), false},
		{NewBorder(BorderOptions{Left: s(BorderSideOptions{Color: bdCol(0xFFFFFFFF)}), Top: s(BorderSideOptions{Color: bdCol(0xFFFFFFFF)}),
			Right: s(BorderSideOptions{Color: bdCol(0xFFFFFFFF)}), Bottom: s(BorderSideOptions{Color: bdCol(0xFFFFFFFF)})}), true},
		{NewBorder(BorderOptions{Left: none(), Top: none(), Right: none(), Bottom: s(BorderSideOptions{Width: bdF(0)})}), false},
		{NewBorder(BorderOptions{Left: none(), Top: none(), Right: none()}), false},
		{NewBorder(BorderOptions{Left: s(BorderSideOptions{}), Top: s(BorderSideOptions{StrokeAlign: bdF(BorderSideStrokeAlignCenter)}),
			Right: s(BorderSideOptions{StrokeAlign: bdF(BorderSideStrokeAlignOutside)})}), false},
		{NewBorder(), true},
	} {
		if c.b.IsUniform() != c.want {
			t.Errorf("#%d", i)
		}
	}
}

func TestBorderLerp(t *testing.T) {
	w := func(v float64) *BorderSide { s := bdBS(BorderSideOptions{Width: bdF(v)}); return &s }
	visualWithTop10 := NewBorder(BorderOptions{Top: w(10)})
	atMinus100 := NewBorder(BorderOptions{Left: w(0), Right: w(300)})
	at0 := NewBorder(BorderOptions{Left: w(100), Right: w(200)})
	at25 := NewBorder(BorderOptions{Left: w(125), Right: w(175)})
	at75 := NewBorder(BorderOptions{Left: w(175), Right: w(125)})
	at100 := NewBorder(BorderOptions{Left: w(200), Right: w(100)})
	at200 := NewBorder(BorderOptions{Left: w(300), Right: w(0)})
	top := func(v float64) Border { return NewBorder(BorderOptions{Top: w(v)}) }
	zero := NewBorder()
	for _, c := range []struct {
		t                    float64
		aNil, nilB, interior Border
	}{
		{-1.0, top(20), zero, atMinus100},
		{0.0, top(10), zero, at0},
		{0.25, top(7.5), top(2.5), at25},
		{0.75, top(2.5), top(7.5), at75},
		{1.0, zero, top(10), at100},
		{2.0, zero, top(20), at200},
	} {
		if BorderLerp(nil, nil, c.t) != nil {
			t.Errorf("%v: null", c.t)
		}
		if got := *BorderLerp(&visualWithTop10, nil, c.t); got != c.aNil {
			t.Errorf("%v: a,null: %v", c.t, got)
		}
		if got := *BorderLerp(nil, &visualWithTop10, c.t); got != c.nilB {
			t.Errorf("%v: null,b: %v", c.t, got)
		}
		if got := *BorderLerp(&at0, &at100, c.t); got != c.interior {
			t.Errorf("%v: %v", c.t, got)
		}
	}
}

func TestBorderThrowsCorrectExceptionWithStrokeAlign(t *testing.T) {
	canvas := ft.NewTestRecordingCanvas()
	left := bdBS(BorderSideOptions{StrokeAlign: bdF(BorderSideStrokeAlignCenter), Color: bdCol(0xff000001)})
	right := bdBS(BorderSideOptions{StrokeAlign: bdF(BorderSideStrokeAlignOutside), Color: bdCol(0xff000002)})
	r := bdPanics(func() {
		NewBorder(BorderOptions{Left: &left, Right: &right}).Paint(canvas, ui.NewRectFromLTWH(10.0, 20.0, 30.0, 40.0), nil)
	})
	err, ok := r.(*foundation.FlutterError)
	if !ok {
		t.Fatalf("got %v, want a FlutterError", r)
	}
	if len(err.Diagnostics) != 1 {
		t.Fatalf("diagnostics: %d", len(err.Diagnostics))
	}
	want := "A Border can only draw strokeAlign different than\nBorderSide.strokeAlignInside on borders with uniform colors.\n"
	if got := err.Diagnostics[0].ToStringDeep(); got != want {
		t.Errorf("got %q", got)
	}
}

func TestBorderDimension(t *testing.T) {
	eq := func(name string, got, want EdgeInsetsGeometry) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
	eq("inside", NewBorderAll(BorderSideOptions{Width: bdF(10)}).Dimensions(), NewEdgeInsetsAll(10))
	eq("center", NewBorderAll(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(BorderSideStrokeAlignCenter)}).Dimensions(),
		NewEdgeInsetsAll(5))
	eq("outside", NewBorderAll(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(BorderSideStrokeAlignOutside)}).Dimensions(),
		EdgeInsetsZero)
	all := func(s BorderSide) BorderDirectional {
		return NewBorderDirectional(BorderDirectionalOptions{Top: &s, Bottom: &s, Start: &s, End: &s})
	}
	eq("dir inside", all(bdBS(BorderSideOptions{Width: bdF(10)})).Dimensions(), NewEdgeInsetsDirectionalAll(10))
	eq("dir center", all(bdBS(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(BorderSideStrokeAlignCenter)})).Dimensions(),
		NewEdgeInsetsDirectionalAll(5))
	eq("dir outside", all(bdBS(BorderSideOptions{Width: bdF(10), StrokeAlign: bdF(BorderSideStrokeAlignOutside)})).Dimensions(),
		EdgeInsetsDirectionalZero)
	s := func(w, a float64) *BorderSide {
		x := bdBS(BorderSideOptions{Width: bdF(w), StrokeAlign: bdF(a)})
		return &x
	}
	in, ce, out := BorderSideStrokeAlignInside, BorderSideStrokeAlignCenter, BorderSideStrokeAlignOutside
	eq("non-uniform", NewBorder(BorderOptions{Left: s(5, in), Top: s(10, ce), Right: s(15, out), Bottom: s(20, in)}).Dimensions(),
		NewEdgeInsetsFromLTRB(5, 5, 0, 20))
	eq("non-uniform dir", NewBorderDirectional(BorderDirectionalOptions{Start: s(5, in), Top: s(10, ce), End: s(15, out),
		Bottom: s(20, in)}).Dimensions(), NewEdgeInsetsDirectionalFromSTEB(5, 5, 0, 20))
	eq("uniform width", NewBorder(BorderOptions{Left: s(10, in), Top: s(10, ce), Right: s(10, out), Bottom: s(10, in)}).Dimensions(),
		NewEdgeInsetsFromLTRB(10, 5, 0, 10))
	eq("uniform width dir", NewBorderDirectional(BorderDirectionalOptions{Start: s(10, in), Top: s(10, ce), End: s(10, out),
		Bottom: s(10, in)}).Dimensions(), NewEdgeInsetsDirectionalFromSTEB(10, 5, 0, 10))
}

// TestBorderNonUniformVariationsPaint is "Non-Uniform Border variations"
// without widgets: each border is painted as BoxDecoration paints it
// (BoxBorder.paint with the decoration's shape and radius); Flutter's
// takeException() is the panic.
func TestBorderNonUniformVariationsPaint(t *testing.T) {
	s := func(w, a float64) *BorderSide {
		x := bdBS(BorderSideOptions{Width: bdF(w), StrokeAlign: bdF(a)})
		return &x
	}
	in, ce, out := BorderSideStrokeAlignInside, BorderSideStrokeAlignCenter, BorderSideStrokeAlignOutside
	rect := ui.NewRectFromLTWH(0, 0, 800, 600)
	r25, r20, r40 := NewBorderRadiusAll(ui.NewRadiusCircular(25)), NewBorderRadiusAll(ui.NewRadiusCircular(20)),
		NewBorderRadiusAll(ui.NewRadiusCircular(40))
	allowed := NewBorder(BorderOptions{Left: s(5, in), Top: s(10, ce), Right: s(15, out), Bottom: s(20, in)})
	allowedDir := NewBorderDirectional(BorderDirectionalOptions{Start: s(5, in), Top: s(10, ce), End: s(15, out), Bottom: s(20, in)})
	styleNone := bdBS(BorderSideOptions{Width: bdF(5), Style: bdStyle(BorderStyleNone)})
	colored := bdBS(BorderSideOptions{Width: bdF(5), Color: bdCol(0xff123456)})
	hairline := bdBS(BorderSideOptions{Width: bdF(0)})
	for i, c := range []struct {
		border BoxBorder
		opts   BoxBorderPaintOptions
		fails  bool
	}{
		{allowed, BoxBorderPaintOptions{}, true},
		{allowed, BoxBorderPaintOptions{BorderRadius: &r25}, false},
		{allowed, BoxBorderPaintOptions{Shape: BoxShapeCircle}, false},
		{NewBorder(BorderOptions{Left: &styleNone, Top: s(10, in), Right: s(15, in), Bottom: s(20, in)}), BoxBorderPaintOptions{BorderRadius: &r25}, false},
		{NewBorder(BorderOptions{Left: &colored, Top: s(10, in), Right: s(15, in), Bottom: s(20, in)}), BoxBorderPaintOptions{BorderRadius: &r20}, true},
		{NewBorder(BorderOptions{Bottom: &hairline}), BoxBorderPaintOptions{BorderRadius: bdRadius(BorderRadiusZero)}, false},
		{NewBorder(BorderOptions{Bottom: &hairline}), BoxBorderPaintOptions{BorderRadius: &r40}, true},
		{allowedDir, BoxBorderPaintOptions{}, true},
		{allowedDir, BoxBorderPaintOptions{BorderRadius: &r25}, false},
		{allowedDir, BoxBorderPaintOptions{Shape: BoxShapeCircle}, false},
	} {
		c.opts.TextDirection = bdLtr
		r := bdPanics(func() { c.border.PaintBox(ft.NewTestRecordingCanvas(), rect, c.opts) })
		if (r != nil) != c.fails {
			t.Errorf("#%d: panic %v, want failure %v", i, r, c.fails)
		}
		if r != nil {
			if _, ok := r.(dart.AssertionFailure); !ok {
				t.Errorf("#%d: %T is not an assertion error", i, r)
			}
		}
	}
}

// shapeWithInterior and shapeWithoutInterior are the test's ShapeBorder
// subclasses (extending ShapeBorder through ShapeBorderBase).
type shapeWithInterior struct {
	ShapeBorderBase
	prefer bool
}

func newShape(prefer bool) *shapeWithInterior {
	s := &shapeWithInterior{prefer: prefer}
	s.SetSelf(s)
	return s
}

func (s *shapeWithInterior) PreferPaintInterior() bool                                      { return s.prefer }
func (s *shapeWithInterior) Scale(t float64) ShapeBorder                                    { return s }
func (s *shapeWithInterior) Dimensions() EdgeInsetsGeometry                                 { return EdgeInsetsZero }
func (s *shapeWithInterior) GetInnerPath(ui.Rect, *ui.TextDirection) *ui.Path               { return ui.NewPath() }
func (s *shapeWithInterior) GetOuterPath(ui.Rect, *ui.TextDirection) *ui.Path               { return ui.NewPath() }
func (s *shapeWithInterior) PaintInterior(ui.Canvas, ui.Rect, *ui.Paint, *ui.TextDirection) {}
func (s *shapeWithInterior) Paint(ui.Canvas, ui.Rect, *ui.TextDirection)                    {}

func TestCompoundBordersWithDifferingPreferPaintInteriors(t *testing.T) {
	with, without := func() ShapeBorder { return newShape(true) }, func() ShapeBorder { return newShape(false) }
	if !with().PreferPaintInterior() || without().PreferPaintInterior() {
		t.Error("single")
	}
	if !with().Add(with()).PreferPaintInterior() || with().Add(without()).PreferPaintInterior() ||
		without().Add(with()).PreferPaintInterior() || without().Add(without()).PreferPaintInterior() {
		t.Error("compound")
	}
}

func TestBoxBorderFactories(t *testing.T) {
	side1 := NewBorderSide()
	side2 := bdBS(BorderSideOptions{Width: bdF(2)})
	side3 := bdBS(BorderSideOptions{Width: bdF(3)})
	side4 := bdBS(BorderSideOptions{Width: bdF(4)})
	if NewBoxBorderFromLTRB(BorderOptions{Left: &side1, Top: &side2, Right: &side3, Bottom: &side4}) !=
		BoxBorder(NewBorder(BorderOptions{Left: &side1, Top: &side2, Right: &side3, Bottom: &side4})) {
		t.Error("fromLTRB")
	}
	if NewBoxBorderAll(BorderSideOptions{Width: bdF(4)}) != BoxBorder(NewBorderAll(BorderSideOptions{Width: bdF(4)})) {
		t.Error("all")
	}
	if NewBoxBorderFromBorderSide(side3) != BoxBorder(NewBorderFromBorderSide(side3)) {
		t.Error("fromBorderSide")
	}
	if NewBoxBorderSymmetric(BorderSymmetricOptions{Horizontal: &side2, Vertical: &side3}) !=
		BoxBorder(NewBorderSymmetric(BorderSymmetricOptions{Horizontal: &side2, Vertical: &side3})) {
		t.Error("symmetric")
	}
	if NewBoxBorderFromSTEB(BorderDirectionalOptions{Start: &side1, Top: &side2, End: &side3, Bottom: &side4}) !=
		BoxBorder(NewBorderDirectional(BorderDirectionalOptions{Start: &side1, Top: &side2, End: &side3, Bottom: &side4})) {
		t.Error("fromSTEB")
	}
}
