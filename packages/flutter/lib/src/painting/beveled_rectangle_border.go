package painting

// Port of beveled_rectangle_border.dart (Flutter 3.50.1).

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// BeveledRectangleBorder is a rectangular border with flattened or
// "beveled" corners. Build it with NewBeveledRectangleBorder.
type BeveledRectangleBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
	// BorderRadius is the radii for each corner (nil: BorderRadius.zero).
	BorderRadius BorderRadiusGeometry
}

// NewBeveledRectangleBorder is BeveledRectangleBorder({side, borderRadius}).
func NewBeveledRectangleBorder(opts ...RRectBorderOptions) BeveledRectangleBorder {
	s, r := rrectBorderOptions(opts).values()
	return BeveledRectangleBorder{s, r}
}

func (b BeveledRectangleBorder) radius() BorderRadiusGeometry { return radiusOrZero(b.BorderRadius) }

// GetSide is the border's side.
func (b BeveledRectangleBorder) GetSide() BorderSide { return b.Side }

// Dimensions is OutlinedBorder's.
func (b BeveledRectangleBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(b.Side) }

// AddBorder returns nil.
func (b BeveledRectangleBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (b BeveledRectangleBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

// Scale scales the side and the radii by t.
func (b BeveledRectangleBorder) Scale(t float64) ShapeBorder {
	return BeveledRectangleBorder{b.Side.Scale(t), b.radius().MulGeometry(t)}
}

// LerpFrom interpolates from a BeveledRectangleBorder or nil.
func (b BeveledRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(BeveledRectangleBorder); ok {
		return BeveledRectangleBorder{BorderSideLerp(x.Side, b.Side, t), BorderRadiusGeometryLerp(x.radius(), b.radius(), t)}
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a BeveledRectangleBorder or nil.
func (b BeveledRectangleBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(BeveledRectangleBorder); ok {
		return BeveledRectangleBorder{BorderSideLerp(b.Side, x.Side, t), BorderRadiusGeometryLerp(b.radius(), x.radius(), t)}
	}
	return shapeBorderLerpTo(b, other, t)
}

// CopyWith returns a copy with the given fields replaced.
func (b BeveledRectangleBorder) CopyWith(opts ...RRectBorderCopyWithOptions) BeveledRectangleBorder {
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
func (b BeveledRectangleBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return b.CopyWith(RRectBorderCopyWithOptions{Side: side})
}

func beveledPath(rrect ui.RRect) *ui.Path {
	center := rrect.Center()
	centerLeft := ui.NewOffset(rrect.Left, center.Dy)
	centerRight := ui.NewOffset(rrect.Right, center.Dy)
	centerTop := ui.NewOffset(center.Dx, rrect.Top)
	centerBottom := ui.NewOffset(center.Dx, rrect.Bottom)

	tlRadiusX := math.Max(0.0, rrect.TlRadiusX)
	tlRadiusY := math.Max(0.0, rrect.TlRadiusY)
	trRadiusX := math.Max(0.0, rrect.TrRadiusX)
	trRadiusY := math.Max(0.0, rrect.TrRadiusY)
	blRadiusX := math.Max(0.0, rrect.BlRadiusX)
	blRadiusY := math.Max(0.0, rrect.BlRadiusY)
	brRadiusX := math.Max(0.0, rrect.BrRadiusX)
	brRadiusY := math.Max(0.0, rrect.BrRadiusY)

	vertices := []ui.Offset{
		ui.NewOffset(rrect.Left, math.Min(centerLeft.Dy, rrect.Top+tlRadiusY)),
		ui.NewOffset(math.Min(centerTop.Dx, rrect.Left+tlRadiusX), rrect.Top),
		ui.NewOffset(math.Max(centerTop.Dx, rrect.Right-trRadiusX), rrect.Top),
		ui.NewOffset(rrect.Right, math.Min(centerRight.Dy, rrect.Top+trRadiusY)),
		ui.NewOffset(rrect.Right, math.Max(centerRight.Dy, rrect.Bottom-brRadiusY)),
		ui.NewOffset(math.Max(centerBottom.Dx, rrect.Right-brRadiusX), rrect.Bottom),
		ui.NewOffset(math.Min(centerBottom.Dx, rrect.Left+blRadiusX), rrect.Bottom),
		ui.NewOffset(rrect.Left, math.Max(centerLeft.Dy, rrect.Bottom-blRadiusY)),
	}
	p := ui.NewPath()
	p.AddPolygon(vertices, true)
	return p
}

// GetInnerPath is the beveled rectangle deflated by the stroke inset.
func (b BeveledRectangleBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return beveledPath(b.radius().Resolve(textDirection).ToRRect(rect).Deflate(b.Side.StrokeInset()))
}

// GetOuterPath is the beveled rectangle.
func (b BeveledRectangleBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return beveledPath(b.radius().Resolve(textDirection).ToRRect(rect))
}

// HitTest is ShapeBorder's: inside the outer path.
func (b BeveledRectangleBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return shapeBorderHitTest(b, rect, position, textDirection)
}

// PaintInterior panics (preferPaintInterior is false), as Dart asserts.
func (b BeveledRectangleBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	shapeBorderPaintInterior(b)
}

// PreferPaintInterior is false.
func (b BeveledRectangleBorder) PreferPaintInterior() bool { return false }

// Paint strokes the outline: the outer polygon plus the inner path.
func (b BeveledRectangleBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if rect.IsEmpty() {
		return
	}
	if b.Side.Style != BorderStyleSolid {
		return
	}
	borderRect := b.radius().Resolve(textDirection).ToRRect(rect)
	adjustedRect := borderRect.Inflate(b.Side.StrokeOutset())
	path := beveledPath(adjustedRect)
	path.AddPath(b.GetInnerPath(rect, textDirection), ui.Offset{})
	canvas.DrawPath(path, b.Side.ToPaint())
}

func (b BeveledRectangleBorder) String() string {
	return "BeveledRectangleBorder(" + b.Side.String() + ", " + b.radius().String() + ")"
}
