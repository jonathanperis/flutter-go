package painting

// Port of circle_border.dart (Flutter 3.50.1): CircleBorder.
//
// OvalBorder extends CircleBorder in Dart, so `x is CircleBorder` is true
// for both: Go checks it with asCircleBorder.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// CircleBorder is a border that fits a circle within the available space
// (an ellipse when Eccentricity > 0). It is a comparable value; build it
// with NewCircleBorder (the zero Side is not BorderSide.none).
type CircleBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
	// Eccentricity defines the ratio (0.0-1.0) from which the border will
	// deform to fit a rectangle (0: a circle, 1: an ellipse filling it).
	Eccentricity float64
}

// CircleBorderOptions are CircleBorder's named parameters (nil Side:
// BorderSide.none).
type CircleBorderOptions struct {
	Side         *BorderSide
	Eccentricity float64
}

func checkEccentricity(e float64) {
	if !(e >= 0.0) {
		panic(dart.NewAssertionError("The eccentricity argument " + dart.DoubleToString(e) + " is not greater than or equal to zero."))
	}
	if !(e <= 1.0) {
		panic(dart.NewAssertionError("The eccentricity argument " + dart.DoubleToString(e) + " is not less than or equal to one."))
	}
}

// NewCircleBorder is CircleBorder({side, eccentricity}).
func NewCircleBorder(opts ...CircleBorderOptions) CircleBorder {
	var o CircleBorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	checkEccentricity(o.Eccentricity)
	return CircleBorder{Side: sideOrNone(o.Side), Eccentricity: o.Eccentricity}
}

func newCircleBorder(side BorderSide, eccentricity float64) CircleBorder {
	checkEccentricity(eccentricity)
	return CircleBorder{side, eccentricity}
}

// asCircleBorder is Dart's `b is CircleBorder` (true for an OvalBorder).
func asCircleBorder(b ShapeBorder) (CircleBorder, bool) {
	switch x := b.(type) {
	case CircleBorder:
		return x, true
	case OvalBorder:
		return CircleBorder(x), true
	}
	return CircleBorder{}, false
}

// GetSide is the border's side.
func (c CircleBorder) GetSide() BorderSide { return c.Side }

// Dimensions is OutlinedBorder's: the side's stroke inset on every edge.
func (c CircleBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(c.Side) }

// AddBorder returns nil.
func (c CircleBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (c CircleBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(c, other) }

// Scale scales the side by t.
func (c CircleBorder) Scale(t float64) ShapeBorder {
	return newCircleBorder(c.Side.Scale(t), c.Eccentricity)
}

func circleLerpEccentricity(a, b, t float64) float64 {
	return ui.ClampDouble(borderLerpDouble(a, b, t), 0.0, 1.0)
}

// LerpFrom interpolates from a CircleBorder (or an OvalBorder, or nil).
func (c CircleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := asCircleBorder(a); ok {
		return newCircleBorder(BorderSideLerp(x.Side, c.Side, t), circleLerpEccentricity(x.Eccentricity, c.Eccentricity, t))
	}
	return shapeBorderLerpFrom(c, a, t)
}

// LerpTo interpolates to a CircleBorder (or an OvalBorder, or nil).
func (c CircleBorder) LerpTo(b ShapeBorder, t float64) ShapeBorder {
	if x, ok := asCircleBorder(b); ok {
		return newCircleBorder(BorderSideLerp(c.Side, x.Side, t), circleLerpEccentricity(c.Eccentricity, x.Eccentricity, t))
	}
	return shapeBorderLerpTo(c, b, t)
}

// GetInnerPath is the oval deflated by the side's stroke inset.
func (c CircleBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddOval(c.adjustRect(rect).Deflate(c.Side.StrokeInset()))
	return p
}

// GetOuterPath is the oval.
func (c CircleBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddOval(c.adjustRect(rect))
	return p
}

// HitTest reports whether position is inside the oval.
func (c CircleBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	adjusted := c.adjustRect(rect)
	return ui.NewRRectFromRectAndRadius(adjusted,
		ui.NewRadiusElliptical(adjusted.Width()/2.0, adjusted.Height()/2.0)).Contains(position)
}

// PaintInterior draws the circle (or oval).
func (c CircleBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	if c.Eccentricity == 0.0 {
		canvas.DrawCircle(rect.Center(), rect.ShortestSide()/2.0, paint)
	} else {
		canvas.DrawOval(c.adjustRect(rect), paint)
	}
}

// PreferPaintInterior is true.
func (c CircleBorder) PreferPaintInterior() bool { return true }

// CircleBorderCopyWithOptions are copyWith's named parameters (nil: keep).
type CircleBorderCopyWithOptions struct {
	Side         *BorderSide
	Eccentricity *float64
}

// CopyWith returns a copy with the given fields replaced.
func (c CircleBorder) CopyWith(opts ...CircleBorderCopyWithOptions) CircleBorder {
	var o CircleBorderCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	side, e := c.Side, c.Eccentricity
	if o.Side != nil {
		side = *o.Side
	}
	if o.Eccentricity != nil {
		e = *o.Eccentricity
	}
	return newCircleBorder(side, e)
}

// CopyWithSide is copyWith({side}).
func (c CircleBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return c.CopyWith(CircleBorderCopyWithOptions{Side: side})
}

// Paint strokes the circle (or oval) with the side.
func (c CircleBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	paintCircle(canvas, rect, c.Side, c.Eccentricity, c.adjustRect)
}

func paintCircle(canvas ui.Canvas, rect ui.Rect, side BorderSide, eccentricity float64, adjust func(ui.Rect) ui.Rect) {
	if side.Style != BorderStyleSolid {
		return
	}
	if eccentricity == 0.0 {
		canvas.DrawCircle(rect.Center(), (rect.ShortestSide()+side.StrokeOffset())/2, side.ToPaint())
	} else {
		borderRect := adjust(rect)
		canvas.DrawOval(borderRect.Inflate(side.StrokeOffset()/2), side.ToPaint())
	}
}

func (c CircleBorder) adjustRect(rect ui.Rect) ui.Rect { return circleAdjustRect(c.Eccentricity, rect) }

func circleAdjustRect(eccentricity float64, rect ui.Rect) ui.Rect {
	if eccentricity == 0.0 || rect.Width() == rect.Height() {
		return ui.NewRectFromCircle(rect.Center(), rect.ShortestSide()/2.0)
	}
	if rect.Width() < rect.Height() {
		delta := (1.0 - eccentricity) * (rect.Height() - rect.Width()) / 2.0
		return ui.NewRectFromLTRB(rect.Left, rect.Top+delta, rect.Right, rect.Bottom-delta)
	}
	delta := (1.0 - eccentricity) * (rect.Width() - rect.Height()) / 2.0
	return ui.NewRectFromLTRB(rect.Left+delta, rect.Top, rect.Right-delta, rect.Bottom)
}

func (c CircleBorder) String() string {
	if c.Eccentricity != 0.0 {
		return "CircleBorder(" + c.Side.String() + ", eccentricity: " + dart.DoubleToString(c.Eccentricity) + ")"
	}
	return "CircleBorder(" + c.Side.String() + ")"
}
