package rendering

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"strings"
	"testing"
)

func f(v float64) *float64 { return &v }

func sized(w, h float64) *RenderConstrainedBox { return NewConstrainedBox(Tight(ui.NewSize(w, h))) }

func withChild[T interface {
	RenderBox
	SetChild(self, child RenderBox)
}](parent T, child RenderBox) T {
	parent.SetChild(parent, child)
	return parent
}

func flex(d Axis, main MainAxisAlignment, cross CrossAxisAlignment, size MainAxisSize, children ...RenderBox) *RenderFlex {
	r := NewFlex(d, main, cross, size)
	r.SetChildren(r, children)
	return r
}

func expanded(flexFactor int, r RenderBox) RenderBox {
	r.Base().Flex = flexFactor
	return r
}

var screen = Tight(ui.NewSize(800, 600))

func assertRect(t *testing.T, name string, r RenderBox, want Rect) {
	t.Helper()
	got := GlobalOffset(r).And(r.Base().Size)
	if got != want {
		t.Errorf("%s: got %+v, want %+v", name, got, want)
	}
}

func TestCenterPadding(t *testing.T) {
	leaf := sized(100, 50)
	pad := withChild(NewPadding(painting.NewEdgeInsetsAll(10)), leaf)
	center := withChild(NewPositionedBox(painting.AlignmentCenter, nil, nil), pad)
	Layout(center, screen)

	assertRect(t, "center", center, ui.NewRectFromLTWH(0, 0, 800, 600))
	assertRect(t, "padding", pad, ui.NewRectFromLTWH(340, 265, 120, 70))
	assertRect(t, "leaf", leaf, ui.NewRectFromLTWH(350, 275, 100, 50))
}

func TestAlignFactors(t *testing.T) {
	leaf := sized(100, 40)
	align := withChild(NewPositionedBox(painting.AlignmentBottomRight, f(2), f(1.5)), leaf)
	root := withChild(NewPositionedBox(painting.AlignmentTopLeft, nil, nil), align)
	Layout(root, screen)
	assertRect(t, "align", align, ui.NewRectFromLTWH(0, 0, 200, 60))
	assertRect(t, "leaf", leaf, ui.NewRectFromLTWH(100, 20, 100, 40))
}

func TestRowExpandedRatios(t *testing.T) {
	a, b, c := sized(100, 20), NewConstrainedBox(Constraints{0, Inf, 0, Inf}), NewConstrainedBox(Constraints{0, Inf, 0, Inf})
	row := flex(Horizontal, MainStart, CrossStart, MainMax, a, expanded(1, b), expanded(3, c))
	Layout(row, screen)
	assertRect(t, "fixed", a, ui.NewRectFromLTWH(0, 0, 100, 20))
	assertRect(t, "flex1", b, ui.NewRectFromLTWH(100, 0, 175, 0))
	assertRect(t, "flex3", c, ui.NewRectFromLTWH(275, 0, 525, 0))
}

func TestColumnSpaceBetweenStretch(t *testing.T) {
	a, b := sized(50, 100), NewConstrainedBox(Constraints{0, Inf, 100, 100})
	col := flex(Vertical, MainSpaceBetween, CrossStretch, MainMax, a, b)
	Layout(col, screen)
	// Stretch forces width 800; the tight SizedBox(50) loses to the tight cross constraint.
	assertRect(t, "a", a, ui.NewRectFromLTWH(0, 0, 800, 100))
	assertRect(t, "b", b, ui.NewRectFromLTWH(0, 500, 800, 100))
}

func TestRowMinSizeCentered(t *testing.T) {
	a, b := sized(30, 10), sized(50, 40)
	row := flex(Horizontal, MainStart, CrossCenter, MainMin, a, b)
	center := withChild(NewPositionedBox(painting.AlignmentCenter, nil, nil), row)
	Layout(center, screen)
	assertRect(t, "row", row, ui.NewRectFromLTWH(360, 280, 80, 40))
	assertRect(t, "a", a, ui.NewRectFromLTWH(360, 295, 30, 10))
	assertRect(t, "b", b, ui.NewRectFromLTWH(390, 280, 50, 40))
}

func TestParagraphWrapsWithTestFont(t *testing.T) {
	p := NewParagraph("hello world", TextStyle{FontSize: 10})
	box := withChild(NewConstrainedBox(Constraints{0, 60, 0, Inf}), p)
	Layout(withChild(NewPositionedBox(painting.AlignmentTopLeft, nil, nil), box), screen)
	// 11 glyphs × 10px = 110px unwrapped; at 60px wide that's 6 per line → 2 lines.
	if got := p.Base().Size; got != (ui.NewSize(60, 20)) {
		t.Errorf("size = %+v", got)
	}
}

func TestLayoutIsCachedUntilMarkedDirty(t *testing.T) {
	leaf := &countingBox{}
	root := withChild(NewPadding(painting.NewEdgeInsetsAll(1)), leaf)
	Layout(root, screen)
	Layout(root, screen)
	if leaf.layouts != 1 {
		t.Fatalf("layouts = %d, want 1 (cached)", leaf.layouts)
	}
	frames := 0
	SetRoot(root, func() { frames++ })
	MarkNeedsLayout(leaf)
	Layout(root, screen)
	if leaf.layouts != 2 || frames != 1 {
		t.Fatalf("layouts = %d frames = %d, want 2 and 1", leaf.layouts, frames)
	}
}

func TestHitTestDeepestFirst(t *testing.T) {
	leaf := NewDecoratedBox(RGB(0xff0000), 0)
	sizedLeaf := withChild(sized(100, 100), leaf)
	tap := withChild(NewTapRegion(nil), sizedLeaf)
	root := withChild(NewPositionedBox(painting.AlignmentCenter, nil, nil), tap)
	Layout(root, screen)

	path := HitTest(root, ui.NewOffset(400, 300))
	if len(path) != 4 || path[0] != RenderBox(leaf) || path[2] != RenderBox(tap) {
		t.Fatalf("unexpected path %T", path)
	}
	if HitTest(root, ui.NewOffset(10, 10)) != nil {
		t.Fatal("miss outside the box should not hit (Align is not hit itself)")
	}
}

func TestPaintOrderAndOffsets(t *testing.T) {
	inner := NewDecoratedBox(RGB(0x00ff00), 4)
	inner.SetChild(inner, sized(20, 10))
	outer := NewDecoratedBox(RGB(0xff0000), 0)
	outer.SetChild(outer, withChild(NewPadding(painting.NewEdgeInsetsAll(5)), inner))
	Layout(withChild(NewPositionedBox(painting.AlignmentTopLeft, nil, nil), outer), screen)

	var rec Recorder
	outer.Paint(&rec, Offset{})
	want := "rect(0, 0, 30, 20) #ff0000\nrrect(5, 5, 20, 10, r=4) #00ff00"
	if got := rec.String(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestOverflowClips(t *testing.T) {
	row := flex(Horizontal, MainStart, CrossStart, MainMax, sized(500, 10), sized(500, 10))
	Layout(row, screen)
	var rec Recorder
	row.Paint(&rec, Offset{})
	if !strings.Contains(rec.String(), "clipRect(0, 0, 800, 600)") {
		t.Fatalf("expected clip, got:\n%s", rec.String())
	}
}

type countingBox struct {
	Box
	layouts int
}

func (c *countingBox) PerformLayout(cs Constraints) Size { c.layouts++; return cs.Smallest() }
func (c *countingBox) Paint(Canvas, Offset)              {}
func (c *countingBox) Children() []RenderBox             { return nil }
