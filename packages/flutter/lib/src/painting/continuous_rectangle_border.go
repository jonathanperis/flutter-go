package painting

// Port of continuous_rectangle_border.dart (Flutter 3.50.1).

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// ContinuousRectangleBorder is a rectangular border with smooth
// continuous transitions between the straight sides and the rounded
// corners. Build it with NewContinuousRectangleBorder.
type ContinuousRectangleBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
	// BorderRadius is the radii for each corner (nil: BorderRadius.zero).
	BorderRadius BorderRadiusGeometry
}

// NewContinuousRectangleBorder is ContinuousRectangleBorder({side,
// borderRadius}).
func NewContinuousRectangleBorder(opts ...RRectBorderOptions) ContinuousRectangleBorder {
	s, r := rrectBorderOptions(opts).values()
	return ContinuousRectangleBorder{s, r}
}

func (b ContinuousRectangleBorder) radius() BorderRadiusGeometry { return radiusOrZero(b.BorderRadius) }

// GetSide is the border's side.
func (b ContinuousRectangleBorder) GetSide() BorderSide { return b.Side }

// Dimensions is EdgeInsets.all(side.width).
func (b ContinuousRectangleBorder) Dimensions() EdgeInsetsGeometry {
	return NewEdgeInsetsAll(b.Side.Width)
}

// AddBorder returns nil.
func (b ContinuousRectangleBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder {
	return nil
}

// Add is operator +.
func (b ContinuousRectangleBorder) Add(other ShapeBorder) ShapeBorder {
	return shapeBorderPlus(b, other)
}

// Scale scales the side and the radii by t.
func (b ContinuousRectangleBorder) Scale(t float64) ShapeBorder {
	return ContinuousRectangleBorder{b.Side.Scale(t), b.radius().MulGeometry(t)}
}

// LerpFrom interpolates from a ContinuousRectangleBorder or nil.
func (b ContinuousRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(ContinuousRectangleBorder); ok {
		return ContinuousRectangleBorder{BorderSideLerp(x.Side, b.Side, t), BorderRadiusGeometryLerp(x.radius(), b.radius(), t)}
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a ContinuousRectangleBorder or nil.
func (b ContinuousRectangleBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(ContinuousRectangleBorder); ok {
		return ContinuousRectangleBorder{BorderSideLerp(b.Side, x.Side, t), BorderRadiusGeometryLerp(b.radius(), x.radius(), t)}
	}
	return shapeBorderLerpTo(b, other, t)
}

func clampToShortest(rrect ui.RRect, value float64) float64 {
	if value > rrect.ShortestSide() {
		return rrect.ShortestSide()
	}
	return value
}

func continuousPath(rrect ui.RRect) *ui.Path {
	left, right, top, bottom := rrect.Left, rrect.Right, rrect.Top, rrect.Bottom
	// Radii are only clamped to the shortest side here (Flutter's quirk:
	// the X radius is used along Y and vice versa).
	tlRadiusX := math.Max(0.0, clampToShortest(rrect, rrect.TlRadiusX))
	tlRadiusY := math.Max(0.0, clampToShortest(rrect, rrect.TlRadiusY))
	trRadiusX := math.Max(0.0, clampToShortest(rrect, rrect.TrRadiusX))
	trRadiusY := math.Max(0.0, clampToShortest(rrect, rrect.TrRadiusY))
	blRadiusX := math.Max(0.0, clampToShortest(rrect, rrect.BlRadiusX))
	blRadiusY := math.Max(0.0, clampToShortest(rrect, rrect.BlRadiusY))
	brRadiusX := math.Max(0.0, clampToShortest(rrect, rrect.BrRadiusX))
	brRadiusY := math.Max(0.0, clampToShortest(rrect, rrect.BrRadiusY))

	p := ui.NewPath()
	p.MoveTo(left, top+tlRadiusX)
	p.CubicTo(left, top, left, top, left+tlRadiusY, top)
	p.LineTo(right-trRadiusX, top)
	p.CubicTo(right, top, right, top, right, top+trRadiusY)
	p.LineTo(right, bottom-brRadiusX)
	p.CubicTo(right, bottom, right, bottom, right-brRadiusY, bottom)
	p.LineTo(left+blRadiusX, bottom)
	p.CubicTo(left, bottom, left, bottom, left, bottom-blRadiusY)
	p.Close()
	return p
}

// GetInnerPath is the outline deflated by the side's width.
func (b ContinuousRectangleBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return continuousPath(b.radius().Resolve(textDirection).ToRRect(rect).Deflate(b.Side.Width))
}

// GetOuterPath is the outline.
func (b ContinuousRectangleBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return continuousPath(b.radius().Resolve(textDirection).ToRRect(rect))
}

// HitTest is ShapeBorder's: inside the outer path.
func (b ContinuousRectangleBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return shapeBorderHitTest(b, rect, position, textDirection)
}

// PaintInterior panics (preferPaintInterior is false), as Dart asserts.
func (b ContinuousRectangleBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	shapeBorderPaintInterior(b)
}

// PreferPaintInterior is false.
func (b ContinuousRectangleBorder) PreferPaintInterior() bool { return false }

// CopyWith returns a copy with the given fields replaced.
func (b ContinuousRectangleBorder) CopyWith(opts ...RRectBorderCopyWithOptions) ContinuousRectangleBorder {
	o := rrectBorderOptions(opts)
	if o.Side != nil {
		b.Side = *o.Side
	}
	if o.BorderRadius != nil {
		b.BorderRadius = o.BorderRadius
	}
	return b
}

// CopyWithSide is copyWith({side}).
func (b ContinuousRectangleBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return b.CopyWith(RRectBorderCopyWithOptions{Side: side})
}

// Paint strokes the outer path with the side.
func (b ContinuousRectangleBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if rect.IsEmpty() {
		return
	}
	if b.Side.Style != BorderStyleSolid {
		return
	}
	canvas.DrawPath(b.GetOuterPath(rect, textDirection), b.Side.ToPaint())
}

func (b ContinuousRectangleBorder) String() string {
	return "ContinuousRectangleBorder(" + b.Side.String() + ", " + b.radius().String() + ")"
}
