package painting

// Port of linear_border.dart (Flutter 3.50.1): LinearBorderEdge and
// LinearBorder.

import (
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// LinearBorderEdge defines the relative size and alignment of one
// LinearBorder edge. It is a comparable value; build it with
// NewLinearBorderEdge (the zero Size is not Dart's default 1.0).
type LinearBorderEdge struct {
	// Size is the edge's length as a fraction of the shape's width or
	// height (0..1).
	Size float64
	// Alignment is the edge's position within the shape (-1..1).
	Alignment float64
}

// LinearBorderEdgeOptions are LinearBorderEdge's named parameters (nil
// Size: 1.0).
type LinearBorderEdgeOptions struct {
	Size      *float64
	Alignment float64
}

// NewLinearBorderEdge is LinearBorderEdge({size, alignment}); it panics
// unless 0 <= size <= 1, as Dart asserts.
func NewLinearBorderEdge(opts ...LinearBorderEdgeOptions) LinearBorderEdge {
	var o LinearBorderEdgeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return newLinearBorderEdge(floatOr(o.Size, 1.0), o.Alignment)
}

func newLinearBorderEdge(size, alignment float64) LinearBorderEdge {
	if !(size >= 0.0 && size <= 1.0) {
		panic(dart.NewAssertionError("size >= 0.0 && size <= 1.0"))
	}
	return LinearBorderEdge{size, alignment}
}

// LinearBorderEdgeLerp is LinearBorderEdge.lerp (nil is null; a missing
// end is a zero-size edge with the other's alignment).
func LinearBorderEdgeLerp(a, b *LinearBorderEdge, t float64) *LinearBorderEdge {
	if a == b {
		return a
	}
	if a == nil {
		a = &LinearBorderEdge{Alignment: b.Alignment, Size: 0}
	}
	if b == nil {
		b = &LinearBorderEdge{Alignment: a.Alignment, Size: 0}
	}
	r := newLinearBorderEdge(borderLerpDouble(a.Size, b.Size, t), borderLerpDouble(a.Alignment, b.Alignment, t))
	return &r
}

func (e LinearBorderEdge) String() string {
	var s strings.Builder
	s.WriteString("LinearBorderEdge(")
	if e.Size != 1.0 {
		s.WriteString("size: " + dart.DoubleToString(e.Size))
	}
	if e.Alignment != 0 {
		comma := ""
		if e.Size != 1.0 {
			comma = ", "
		}
		s.WriteString(comma + "alignment: " + dart.DoubleToString(e.Alignment))
	}
	s.WriteString(")")
	return s.String()
}

// LinearBorder is an OutlinedBorder like BoxBorder that allows one to
// define a rectangular border in terms of zero to four edges (nil edges
// are not drawn). It compares with Equal (its edges are pointers).
type LinearBorder struct {
	// Side is the edges' color and weight.
	Side BorderSide
	// Start, End, Top, Bottom are the edges (nil: no edge).
	Start, End, Top, Bottom *LinearBorderEdge
}

// LinearBorderOptions are LinearBorder's named parameters (nil Side:
// BorderSide.none).
type LinearBorderOptions struct {
	Side                    *BorderSide
	Start, End, Top, Bottom *LinearBorderEdge
}

// NewLinearBorder is LinearBorder({side, start, end, top, bottom}).
func NewLinearBorder(opts ...LinearBorderOptions) LinearBorder {
	var o LinearBorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return LinearBorder{sideOrNone(o.Side), o.Start, o.End, o.Top, o.Bottom}
}

// LinearBorderEdgeConstructorOptions are the named parameters of
// LinearBorder.start/end/top/bottom (nil Side: BorderSide.none; nil Size:
// 1.0).
type LinearBorderEdgeConstructorOptions struct {
	Side      *BorderSide
	Alignment float64
	Size      *float64
}

func (o LinearBorderEdgeConstructorOptions) edge() *LinearBorderEdge {
	e := newLinearBorderEdge(floatOr(o.Size, 1.0), o.Alignment)
	return &e
}

func linearOpts(opts []LinearBorderEdgeConstructorOptions) LinearBorderEdgeConstructorOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return LinearBorderEdgeConstructorOptions{}
}

// NewLinearBorderStart is LinearBorder.start: only a start edge.
func NewLinearBorderStart(opts ...LinearBorderEdgeConstructorOptions) LinearBorder {
	o := linearOpts(opts)
	return LinearBorder{Side: sideOrNone(o.Side), Start: o.edge()}
}

// NewLinearBorderEnd is LinearBorder.end: only an end edge.
func NewLinearBorderEnd(opts ...LinearBorderEdgeConstructorOptions) LinearBorder {
	o := linearOpts(opts)
	return LinearBorder{Side: sideOrNone(o.Side), End: o.edge()}
}

// NewLinearBorderTop is LinearBorder.top: only a top edge.
func NewLinearBorderTop(opts ...LinearBorderEdgeConstructorOptions) LinearBorder {
	o := linearOpts(opts)
	return LinearBorder{Side: sideOrNone(o.Side), Top: o.edge()}
}

// NewLinearBorderBottom is LinearBorder.bottom: only a bottom edge.
func NewLinearBorderBottom(opts ...LinearBorderEdgeConstructorOptions) LinearBorder {
	o := linearOpts(opts)
	return LinearBorder{Side: sideOrNone(o.Side), Bottom: o.edge()}
}

// LinearBorderNone is LinearBorder.none: no edges.
var LinearBorderNone = LinearBorder{Side: BorderSideNone}

// GetSide is the border's side.
func (b LinearBorder) GetSide() BorderSide { return b.Side }

// AddBorder returns nil.
func (b LinearBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (b LinearBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

// Scale is LinearBorder(side: side.scale(t)): as in Flutter, the edges are
// dropped.
func (b LinearBorder) Scale(t float64) ShapeBorder { return LinearBorder{Side: b.Side.Scale(t)} }

// Dimensions is the side's width on each edge that is present.
func (b LinearBorder) Dimensions() EdgeInsetsGeometry {
	width := b.Side.Width
	w := func(e *LinearBorderEdge) float64 {
		if e == nil {
			return 0.0
		}
		return width
	}
	return NewEdgeInsetsDirectionalFromSTEB(w(b.Start), w(b.Top), w(b.End), w(b.Bottom))
}

func (b LinearBorder) lerp(a, c LinearBorder, t float64) LinearBorder {
	return LinearBorder{
		Side:   BorderSideLerp(a.Side, c.Side, t),
		Start:  LinearBorderEdgeLerp(a.Start, c.Start, t),
		End:    LinearBorderEdgeLerp(a.End, c.End, t),
		Top:    LinearBorderEdgeLerp(a.Top, c.Top, t),
		Bottom: LinearBorderEdgeLerp(a.Bottom, c.Bottom, t),
	}
}

// LerpFrom interpolates from a LinearBorder or nil.
func (b LinearBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(LinearBorder); ok {
		return b.lerp(x, b, t)
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a LinearBorder or nil.
func (b LinearBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(LinearBorder); ok {
		return b.lerp(b, x, t)
	}
	return shapeBorderLerpTo(b, other, t)
}

// LinearBorderCopyWithOptions are copyWith's named parameters (nil:
// keep).
type LinearBorderCopyWithOptions = LinearBorderOptions

// CopyWith returns a copy with the given fields replaced.
func (b LinearBorder) CopyWith(opts ...LinearBorderCopyWithOptions) LinearBorder {
	var o LinearBorderCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Side != nil {
		b.Side = *o.Side
	}
	if o.Start != nil {
		b.Start = o.Start
	}
	if o.End != nil {
		b.End = o.End
	}
	if o.Top != nil {
		b.Top = o.Top
	}
	if o.Bottom != nil {
		b.Bottom = o.Bottom
	}
	return b
}

// CopyWithSide is copyWith({side}).
func (b LinearBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return b.CopyWith(LinearBorderCopyWithOptions{Side: side})
}

// GetInnerPath is the rect deflated by the resolved dimensions.
func (b LinearBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(b.Dimensions().Resolve(textDirection).DeflateRect(rect))
	return p
}

// GetOuterPath is the rect.
func (b LinearBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}

// HitTest is ShapeBorder's: inside the outer path.
func (b LinearBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return shapeBorderHitTest(b, rect, position, textDirection)
}

// PaintInterior panics (preferPaintInterior is false), as Dart asserts.
func (b LinearBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	shapeBorderPaintInterior(b)
}

// PreferPaintInterior is false.
func (b LinearBorder) PreferPaintInterior() bool { return false }

// Paint draws each edge that is present as a filled rectangle (or a
// hairline when its width or height is zero).
func (b LinearBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	insets := b.Dimensions().Resolve(textDirection)
	rtl := textDirection != nil && *textDirection == ui.TextDirectionRtl
	path := ui.NewPath()
	paint := ui.NewPaint()
	paint.SetStrokeWidth(0.0)

	drawEdge := func(rect ui.Rect, color ui.Color) {
		paint.SetColor(color)
		path.Reset()
		path.MoveTo(rect.Left, rect.Top)
		if rect.Width() == 0.0 {
			paint.SetStyle(ui.PaintingStyleStroke)
			path.LineTo(rect.Left, rect.Bottom)
		} else if rect.Height() == 0.0 {
			paint.SetStyle(ui.PaintingStyleStroke)
			path.LineTo(rect.Right, rect.Top)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Right, rect.Top)
			path.LineTo(rect.Right, rect.Bottom)
			path.LineTo(rect.Left, rect.Bottom)
		}
		canvas.DrawPath(path, paint)
	}

	visible := b.Side.Style != BorderStyleNone
	if b.Start != nil && b.Start.Size != 0.0 && visible {
		insetRect := ui.NewRectFromLTWH(rect.Left, rect.Top+insets.Top, rect.Width(), rect.Height()-insets.Vertical())
		x, width := rect.Left, insets.Left
		if rtl {
			x, width = rect.Right-insets.Right, insets.Right
		}
		height := float64(insetRect.Height() * b.Start.Size)
		y := float64((insetRect.Height() - height) * ((b.Start.Alignment + 1.0) / 2.0))
		drawEdge(ui.NewRectFromLTWH(x, y, width, height), b.Side.Color)
	}
	if b.End != nil && b.End.Size != 0.0 && visible {
		insetRect := ui.NewRectFromLTWH(rect.Left, rect.Top+insets.Top, rect.Width(), rect.Height()-insets.Vertical())
		x, width := rect.Right-insets.Right, insets.Right
		if rtl {
			x, width = rect.Left, insets.Left
		}
		height := float64(insetRect.Height() * b.End.Size)
		y := float64((insetRect.Height() - height) * ((b.End.Alignment + 1.0) / 2.0))
		drawEdge(ui.NewRectFromLTWH(x, y, width, height), b.Side.Color)
	}
	if b.Top != nil && b.Top.Size != 0.0 && visible {
		width := float64(rect.Width() * b.Top.Size)
		startX := float64((rect.Width() - width) * ((b.Top.Alignment + 1.0) / 2.0))
		x := startX
		if rtl {
			x = rect.Width() - startX - width
		}
		drawEdge(ui.NewRectFromLTWH(x, rect.Top, width, insets.Top), b.Side.Color)
	}
	if b.Bottom != nil && b.Bottom.Size != 0.0 && visible {
		width := float64(rect.Width() * b.Bottom.Size)
		startX := float64((rect.Width() - width) * ((b.Bottom.Alignment + 1.0) / 2.0))
		x := startX
		if rtl {
			x = rect.Width() - startX - width
		}
		drawEdge(ui.NewRectFromLTWH(x, rect.Bottom-insets.Bottom, width, b.Side.Width), b.Side.Color)
	}
}

func edgeEqual(a, b *LinearBorderEdge) bool {
	return a == b || (a != nil && b != nil && *a == *b)
}

// Equal is Dart's ==: the same side and edges.
func (b LinearBorder) Equal(other ShapeBorder) bool {
	o, ok := other.(LinearBorder)
	return ok && o.Side == b.Side && edgeEqual(o.Start, b.Start) && edgeEqual(o.End, b.End) &&
		edgeEqual(o.Top, b.Top) && edgeEqual(o.Bottom, b.Bottom)
}

func (b LinearBorder) String() string {
	if b.Equal(LinearBorderNone) {
		return "LinearBorder.none"
	}
	var s strings.Builder
	s.WriteString("LinearBorder(side: " + b.Side.String())
	for _, e := range []struct {
		name string
		edge *LinearBorderEdge
	}{{"start", b.Start}, {"end", b.End}, {"top", b.Top}, {"bottom", b.Bottom}} {
		if e.edge != nil {
			s.WriteString(", " + e.name + ": " + e.edge.String())
		}
	}
	s.WriteString(")")
	return s.String()
}
