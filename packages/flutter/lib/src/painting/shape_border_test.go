package painting_test

// Port of packages/flutter/test/painting/shape_border_test.dart (Flutter
// 3.50.1). Nothing skipped. Dart's identical() is Go's == on the values;
// runtimeType is the Go dynamic type.

import (
	"fmt"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func TestBorderLerpIdenticalAB(t *testing.T) {
	if BorderLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	border := NewBorder()
	if BorderLerp(&border, &border, 0.5) != &border {
		t.Error("identical")
	}
}

func TestBoxBorderLerpIdenticalAB(t *testing.T) {
	if BoxBorderLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	var border BoxBorder = NewBorder()
	if BoxBorderLerp(border, border, 0.5) != border {
		t.Error("identical")
	}
}

func TestBorderDirectionalLerpIdenticalAB(t *testing.T) {
	if BorderDirectionalLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	border := NewBorderDirectional()
	if ShapeBorderLerp(border, border, 0.5) != ShapeBorder(border) {
		t.Error("identical")
	}
}

func TestOutlinedBorderLerpIdenticalAB(t *testing.T) {
	if OutlinedBorderLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	var border OutlinedBorder = NewRoundedRectangleBorder()
	if OutlinedBorderLerp(border, border, 0.5) != border {
		t.Error("identical")
	}
}

func TestShapeBorderLerpIdenticalAB(t *testing.T) {
	if ShapeBorderLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	var border ShapeBorder = NewCircleBorder()
	if ShapeBorderLerp(border, border, 0.5) != border {
		t.Error("identical")
	}
}

// lerpBorder is the test's _LerpBorder: an OutlinedBorder that records
// the t it was created with (nil: null). reverse selects the
// _ReverseLerpToBorder (1) and _ReverseLerpFromBorder (2) subclasses.
type lerpBorder struct {
	ShapeBorderBase
	t       *float64
	side    BorderSide
	reverse int
}

func newLerpBorder(t *float64, side BorderSide, reverse int) *lerpBorder {
	b := &lerpBorder{t: t, side: side, reverse: reverse}
	b.SetSelf(b)
	return b
}

func lerpB(t ...float64) *lerpBorder {
	var p *float64
	if len(t) > 0 {
		p = &t[0]
	}
	return newLerpBorder(p, BorderSideNone, 0)
}

func (b *lerpBorder) GetSide() BorderSide { return b.side }
func (b *lerpBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	s := b.side
	if side != nil {
		s = *side
	}
	return newLerpBorder(b.t, s, 0)
}
func (b *lerpBorder) Dimensions() EdgeInsetsGeometry {
	return NewEdgeInsetsAll(max(b.side.StrokeInset(), 0))
}
func (b *lerpBorder) GetInnerPath(rect ui.Rect, _ *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}
func (b *lerpBorder) GetOuterPath(rect ui.Rect, td *ui.TextDirection) *ui.Path {
	return b.GetInnerPath(rect, td)
}
func (b *lerpBorder) Paint(ui.Canvas, ui.Rect, *ui.TextDirection) {}
func (b *lerpBorder) Scale(t float64) ShapeBorder                 { return newLerpBorder(&t, b.side.Scale(t), 0) }
func (b *lerpBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if _, ok := other.(*lerpBorder); ok && b.reverse == 1 {
		return lerpB(t)
	}
	return b.ShapeBorderBase.LerpTo(other, t)
}
func (b *lerpBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if _, ok := a.(*lerpBorder); ok && b.reverse == 2 {
		return lerpB(t)
	}
	return b.ShapeBorderBase.LerpFrom(a, t)
}
func (b *lerpBorder) Equal(other ShapeBorder) bool {
	o, ok := other.(*lerpBorder)
	return ok && o.reverse == b.reverse && (o.t == b.t || (o.t != nil && b.t != nil && *o.t == *b.t)) && o.side == b.side
}
func (b *lerpBorder) String() string {
	if b.t == nil {
		return "_LerpBorder(t: null)"
	}
	return fmt.Sprintf("_LerpBorder(t: %v)", *b.t)
}

func TestShapeBorderLerpTriesEquivalentReverseInterpolation(t *testing.T) {
	if got := ShapeBorderLerp(lerpB(), newLerpBorder(nil, BorderSideNone, 1), 0.25); !ShapeBorderEqual(got, lerpB(0.75)) {
		t.Errorf("lerpTo: %v", got)
	}
	if got := ShapeBorderLerp(newLerpBorder(nil, BorderSideNone, 2), lerpB(), 0.25); !ShapeBorderEqual(got, lerpB(0.75)) {
		t.Errorf("lerpFrom: %v", got)
	}
}

func TestOutlinedBorderLerpTriesEquivalentReverseInterpolation(t *testing.T) {
	if got := OutlinedBorderLerp(lerpB(), newLerpBorder(nil, BorderSideNone, 1), 0.25); !ShapeBorderEqual(got, lerpB(0.75)) {
		t.Errorf("lerpTo: %v", got)
	}
	if got := OutlinedBorderLerp(newLerpBorder(nil, BorderSideNone, 2), lerpB(), 0.25); !ShapeBorderEqual(got, lerpB(0.75)) {
		t.Errorf("lerpFrom: %v", got)
	}
}

func TestOutlinedShapeBordersInterpolateSymmetrically(t *testing.T) {
	borders := []OutlinedBorder{NewRoundedRectangleBorder(), NewStadiumBorder(), NewCircleBorder(), NewOvalBorder(), NewStarBorder()}
	for _, a := range borders {
		for _, b := range borders {
			if x, y := fmt.Sprintf("%T", ShapeBorderLerp(a, b, 0.25)), fmt.Sprintf("%T", ShapeBorderLerp(b, a, 0.75)); x != y {
				t.Errorf("ShapeBorder.lerp(%v, %v, 0.25): %s vs %s", a, b, x, y)
			}
			if x, y := fmt.Sprintf("%T", OutlinedBorderLerp(a, b, 0.25)), fmt.Sprintf("%T", OutlinedBorderLerp(b, a, 0.75)); x != y {
				t.Errorf("OutlinedBorder.lerp(%v, %v, 0.25): %s vs %s", a, b, x, y)
			}
		}
	}
}

func TestCompoundBorders(t *testing.T) {
	b1 := NewBorderAll(BorderSideOptions{Color: bdCol(0xFF00FF00)})
	b2 := NewBorderAll(BorderSideOptions{Color: bdCol(0xFF0000FF)})
	c := func(v int) string { return ui.NewColor(v).String() }
	check := func(name string, got ShapeBorder, want string) {
		t.Helper()
		if got.String() != want {
			t.Errorf("%s:\n got %s\nwant %s", name, got, want)
		}
	}
	check("b1+b2", b1.Add(b2), "Border.all(BorderSide(color: "+c(0xff00ff00)+")) + Border.all(BorderSide(color: "+c(0xff0000ff)+"))")
	check("b1+(b2+b2)", b1.Add(b2.Add(b2)), "Border.all(BorderSide(color: "+c(0xff00ff00)+")) + Border.all(BorderSide(color: "+c(0xff0000ff)+", width: 2.0))")
	check("(b1+b2)+b2", b1.Add(b2).Add(b2), "Border.all(BorderSide(color: "+c(0xff00ff00)+")) + Border.all(BorderSide(color: "+c(0xff0000ff)+", width: 2.0))")
	if !ShapeBorderEqual(b1.Add(b2).Add(b2), b1.Add(b2.Add(b2))) {
		t.Error("associativity")
	}
	check("scale 3", b1.Add(b2).Scale(3.0), "Border.all(BorderSide(color: "+c(0xff00ff00)+", width: 3.0)) + Border.all(BorderSide(color: "+c(0xff0000ff)+", width: 3.0))")
	check("scale 0", b1.Add(b2).Scale(0.0), "Border.all(BorderSide(color: "+c(0xff00ff00)+", width: 0.0, style: none)) + Border.all(BorderSide(color: "+c(0xff0000ff)+", width: 0.0, style: none))")
	for _, x := range []struct {
		t    float64
		a, b int
	}{{0.0, 0xff0000ff, 0xff00ff00}, {0.20, 0xff0033cc, 0xff00cc33}, {1.0 / 3, 0xff0055aa, 0xff00aa55}, {1.0, 0xff00ff00, 0xff0000ff}} {
		check(fmt.Sprint("lerp ", x.t), ShapeBorderLerp(b2.Add(b1), b1.Add(b2), x.t),
			"Border.all(BorderSide(color: "+c(x.a)+")) + Border.all(BorderSide(color: "+c(x.b)+"))")
	}
	if !EdgeInsetsGeometryEqual(b1.Add(b2).Dimensions(), NewEdgeInsetsAll(2.0)) {
		t.Error("dimensions")
	}
	rect := ui.NewRectFromLTRB(11.0, 15.0, 299.0, 175.0)
	r := func(d float64) *ui.Rect { x := rect.Deflate(d); return &x }
	col := func(b Border) *ui.Color { x := b.Top().Color; return &x }
	ft.Expect(t, func(cv ui.Canvas) { b1.Add(b2).Paint(cv, rect, nil) }, ft.Paints().
		Rect(ft.PaintPatternRectOptions{Rect: r(0.5), PaintOptions: ft.PaintOptions{Color: col(b2)}}).
		Rect(ft.PaintPatternRectOptions{Rect: r(1.5), PaintOptions: ft.PaintOptions{Color: col(b1)}}))
	if !EdgeInsetsGeometryEqual(b1.Add(b2).Add(b1).Dimensions(), NewEdgeInsetsAll(3.0)) {
		t.Error("dimensions 3")
	}
	ft.Expect(t, func(cv ui.Canvas) { b1.Add(b2).Add(b1).Paint(cv, rect, nil) }, ft.Paints().
		Rect(ft.PaintPatternRectOptions{Rect: r(0.5), PaintOptions: ft.PaintOptions{Color: col(b1)}}).
		Rect(ft.PaintPatternRectOptions{Rect: r(1.5), PaintOptions: ft.PaintOptions{Color: col(b2)}}).
		Rect(ft.PaintPatternRectOptions{Rect: r(2.5), PaintOptions: ft.PaintOptions{Color: col(b1)}}))
}

func TestCompoundBordersDirectional(t *testing.T) {
	side1 := bdBS(BorderSideOptions{Color: bdCol(0xFF00FF00)})
	side2 := bdBS(BorderSideOptions{Color: bdCol(0xFF0000FF)})
	b1 := NewBorderDirectional(BorderDirectionalOptions{Top: &side1, Start: &side1, End: &side1, Bottom: &side1})
	b2 := NewBorderDirectional(BorderDirectionalOptions{Top: &side2, Start: &side2, End: &side2, Bottom: &side2})
	bd := func(color int, extra string) string {
		s := "BorderSide(color: " + ui.NewColor(color).String() + extra + ")"
		return "BorderDirectional(top: " + s + ", start: " + s + ", end: " + s + ", bottom: " + s + ")"
	}
	check := func(name string, got ShapeBorder, want string) {
		t.Helper()
		if got.String() != want {
			t.Errorf("%s:\n got %s\nwant %s", name, got, want)
		}
	}
	check("b1+b2", b1.Add(b2), bd(0xff00ff00, "")+" + "+bd(0xff0000ff, ""))
	check("b1+(b2+b2)", b1.Add(b2.Add(b2)), bd(0xff00ff00, "")+" + "+bd(0xff0000ff, ", width: 2.0"))
	check("(b1+b2)+b2", b1.Add(b2).Add(b2), bd(0xff00ff00, "")+" + "+bd(0xff0000ff, ", width: 2.0"))
	if !ShapeBorderEqual(b1.Add(b2).Add(b2), b1.Add(b2.Add(b2))) {
		t.Error("associativity")
	}
	check("scale 3", b1.Add(b2).Scale(3.0), bd(0xff00ff00, ", width: 3.0")+" + "+bd(0xff0000ff, ", width: 3.0"))
	check("scale 0", b1.Add(b2).Scale(0.0), bd(0xff00ff00, ", width: 0.0, style: none")+" + "+bd(0xff0000ff, ", width: 0.0, style: none"))
	for _, x := range []struct {
		t    float64
		a, b int
	}{{0.0, 0xff0000ff, 0xff00ff00}, {0.20, 0xff0033cc, 0xff00cc33}, {1.0 / 3, 0xff0055aa, 0xff00aa55}, {1.0, 0xff00ff00, 0xff0000ff}} {
		check(fmt.Sprint("lerp ", x.t), ShapeBorderLerp(b2.Add(b1), b1.Add(b2), x.t), bd(x.a, "")+" + "+bd(x.b, ""))
	}
	if !EdgeInsetsGeometryEqual(b1.Add(b2).Dimensions(), NewEdgeInsetsDirectionalFromSTEB(2, 2, 2, 2)) {
		t.Error("dimensions")
	}
	rect := ui.NewRectFromLTRB(11.0, 15.0, 299.0, 175.0)
	r := func(d float64) *ui.Rect { x := rect.Deflate(d); return &x }
	c1, c2 := side1.Color, side2.Color
	ft.Expect(t, func(cv ui.Canvas) { b1.Add(b2).Paint(cv, rect, bdRtl) }, ft.Paints().
		Rect(ft.PaintPatternRectOptions{Rect: r(0.5), PaintOptions: ft.PaintOptions{Color: &c2}}).
		Rect(ft.PaintPatternRectOptions{Rect: r(1.5), PaintOptions: ft.PaintOptions{Color: &c1}}))
	if !EdgeInsetsGeometryEqual(b1.Add(b2).Add(b1).Dimensions(), NewEdgeInsetsDirectionalFromSTEB(3, 3, 3, 3)) {
		t.Error("dimensions 3")
	}
	ft.Expect(t, func(cv ui.Canvas) { b1.Add(b2).Add(b1).Paint(cv, rect, bdRtl) }, ft.Paints().
		Rect(ft.PaintPatternRectOptions{Rect: r(0.5), PaintOptions: ft.PaintOptions{Color: &c1}}).
		Rect(ft.PaintPatternRectOptions{Rect: r(1.5), PaintOptions: ft.PaintOptions{Color: &c2}}).
		Rect(ft.PaintPatternRectOptions{Rect: r(2.5), PaintOptions: ft.PaintOptions{Color: &c1}}))
}
