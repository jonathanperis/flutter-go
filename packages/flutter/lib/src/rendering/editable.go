package rendering

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"math"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// CursorWidth is Flutter's default cursorWidth.
const CursorWidth = 2.0

// RenderEditable displays one line of editable text with a caret and a
// selection highlight, scrolling horizontally to keep the caret visible.
// It is the render half of TextField (Flutter's RenderEditable, single line).
type RenderEditable struct {
	Box
	value          services.Value
	style          TextStyle
	focused        bool
	caretVisible   bool
	CaretColor     Color
	SelectionColor Color
	para           Paragraph
	lineHeight     float64
	scrollX        float64
}

func NewEditable(style TextStyle) *RenderEditable {
	return &RenderEditable{style: style, CaretColor: RGB(0x1d1b20), SelectionColor: ui.Color{R: 0.4, G: 0.31, B: 0.64, A: 0.3}}
}

func (r *RenderEditable) Children() []RenderBox   { return nil }
func (r *RenderEditable) HitTestSelf(Offset) bool { return true }
func (r *RenderEditable) Value() services.Value   { return r.value }

func (r *RenderEditable) Set(v services.Value, style TextStyle, focused, caretVisible bool) {
	if v.Text != r.value.Text || style != r.style {
		MarkNeedsLayout(r)
	} else if v != r.value || focused != r.focused || caretVisible != r.caretVisible {
		MarkNeedsLayout(r) // selection may need scrolling; layout is cheap and cached text is reused
	}
	r.value, r.style, r.focused, r.caretVisible = v, style, focused, caretVisible
}

func (r *RenderEditable) PerformLayout(c Constraints) Size {
	if r.para == nil || r.para.Text() != r.value.Text || r.lineHeight == 0 {
		if r.para != nil {
			r.para.Release()
		}
		r.para = textEngine.Layout(r.value.Text, r.style, Inf)
		r.lineHeight = r.para.Height()
		if r.value.Text == "" { // an empty paragraph may have no line height
			probe := textEngine.Layout(" ", r.style, Inf)
			r.lineHeight = probe.Height()
			probe.Release()
		}
	}
	w := c.MaxW
	if w == Inf {
		w = r.para.MaxIntrinsicWidth() + CursorWidth
	}
	size := c.Constrain(ui.NewSize(w, r.lineHeight))
	// Keep the caret (selection extent) visible.
	caret := r.para.CaretX(r.value.Selection.Extent)
	if caret-r.scrollX > size.Width-CursorWidth {
		r.scrollX = caret - size.Width + CursorWidth
	}
	if caret < r.scrollX {
		r.scrollX = caret
	}
	r.scrollX = math.Max(0, math.Min(r.scrollX, math.Max(0, r.para.MaxIntrinsicWidth()+CursorWidth-size.Width)))
	return size
}

// OffsetAt returns the UTF-16 caret offset for a point in local coordinates.
func (r *RenderEditable) OffsetAt(local Offset) int {
	if r.para == nil {
		return 0
	}
	return r.para.OffsetAtX(local.Dx + r.scrollX)
}

func (r *RenderEditable) Paint(cv Canvas, offset Offset) {
	cv.Save()
	cv.ClipRect(offset.And(r.Size))
	origin := ui.NewOffset(offset.Dx-r.scrollX, offset.Dy)
	sel := r.value.Selection
	if r.focused && !sel.IsCollapsed() {
		x0, x1 := r.para.CaretX(sel.Start()), r.para.CaretX(sel.End())
		cv.DrawRect(ui.NewRectFromLTWH(origin.Dx+x0, origin.Dy, x1-x0, r.lineHeight), r.SelectionColor)
	}
	cv.DrawParagraph(r.para, origin)
	if r.focused && r.caretVisible && sel.IsCollapsed() {
		x := r.para.CaretX(sel.Extent)
		cv.DrawRect(ui.NewRectFromLTWH(origin.Dx+x, origin.Dy, CursorWidth, r.lineHeight), r.CaretColor)
	}
	cv.Restore()
}

func (r *RenderEditable) Dispose() {
	if r.para != nil {
		r.para.Release()
		r.para = nil
	}
}
