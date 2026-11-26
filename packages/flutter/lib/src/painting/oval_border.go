package painting

// Port of oval_border.dart (Flutter 3.50.1): OvalBorder, a CircleBorder
// whose eccentricity defaults to 1.0. It has CircleBorder's fields (the
// two types convert into each other), and the other borders treat it as a
// CircleBorder, as Dart's `is CircleBorder` does.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// OvalBorder is a border that fits an elliptical shape. Build it with
// NewOvalBorder.
type OvalBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
	// Eccentricity: see CircleBorder.Eccentricity (default 1.0).
	Eccentricity float64
}

// OvalBorderOptions are OvalBorder's named parameters (nil Side:
// BorderSide.none; nil Eccentricity: 1.0).
type OvalBorderOptions struct {
	Side         *BorderSide
	Eccentricity *float64
}

// NewOvalBorder is OvalBorder({side, eccentricity}).
func NewOvalBorder(opts ...OvalBorderOptions) OvalBorder {
	var o OvalBorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := 1.0
	if o.Eccentricity != nil {
		e = *o.Eccentricity
	}
	return newOvalBorder(sideOrNone(o.Side), e)
}

func newOvalBorder(side BorderSide, eccentricity float64) OvalBorder {
	checkEccentricity(eccentricity)
	return OvalBorder{side, eccentricity}
}

func (o OvalBorder) circle() CircleBorder { return CircleBorder(o) }

// GetSide is the border's side.
func (o OvalBorder) GetSide() BorderSide { return o.Side }

// Dimensions is OutlinedBorder's.
func (o OvalBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(o.Side) }

// AddBorder returns nil.
func (o OvalBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (o OvalBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(o, other) }

// Scale scales the side by t.
func (o OvalBorder) Scale(t float64) ShapeBorder {
	return newOvalBorder(o.Side.Scale(t), o.Eccentricity)
}

// LerpFrom interpolates from an OvalBorder; otherwise as CircleBorder.
func (o OvalBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(OvalBorder); ok {
		return newOvalBorder(BorderSideLerp(x.Side, o.Side, t), circleLerpEccentricity(x.Eccentricity, o.Eccentricity, t))
	}
	if x, ok := asCircleBorder(a); ok {
		return newCircleBorder(BorderSideLerp(x.Side, o.Side, t), circleLerpEccentricity(x.Eccentricity, o.Eccentricity, t))
	}
	return shapeBorderLerpFrom(o, a, t)
}

// LerpTo interpolates to an OvalBorder; otherwise as CircleBorder.
func (o OvalBorder) LerpTo(b ShapeBorder, t float64) ShapeBorder {
	if x, ok := b.(OvalBorder); ok {
		return newOvalBorder(BorderSideLerp(o.Side, x.Side, t), circleLerpEccentricity(o.Eccentricity, x.Eccentricity, t))
	}
	if x, ok := asCircleBorder(b); ok {
		return newCircleBorder(BorderSideLerp(o.Side, x.Side, t), circleLerpEccentricity(o.Eccentricity, x.Eccentricity, t))
	}
	return shapeBorderLerpTo(o, b, t)
}

// GetInnerPath is CircleBorder's.
func (o OvalBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return o.circle().GetInnerPath(rect, textDirection)
}

// GetOuterPath is CircleBorder's.
func (o OvalBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return o.circle().GetOuterPath(rect, textDirection)
}

// HitTest is CircleBorder's.
func (o OvalBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return o.circle().HitTest(rect, position, textDirection)
}

// PaintInterior is CircleBorder's.
func (o OvalBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	o.circle().PaintInterior(canvas, rect, paint, textDirection)
}

// PreferPaintInterior is true.
func (o OvalBorder) PreferPaintInterior() bool { return true }

// OvalBorderCopyWithOptions are copyWith's named parameters (nil: keep).
type OvalBorderCopyWithOptions = CircleBorderCopyWithOptions

// CopyWith returns a copy with the given fields replaced.
func (o OvalBorder) CopyWith(opts ...OvalBorderCopyWithOptions) OvalBorder {
	return OvalBorder(o.circle().CopyWith(opts...))
}

// CopyWithSide is copyWith({side}).
func (o OvalBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return o.CopyWith(OvalBorderCopyWithOptions{Side: side})
}

// Paint is CircleBorder's.
func (o OvalBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	o.circle().Paint(canvas, rect, textDirection)
}

func (o OvalBorder) String() string {
	if o.Eccentricity != 1.0 {
		return "OvalBorder(" + o.Side.String() + ", eccentricity: " + dart.DoubleToString(o.Eccentricity) + ")"
	}
	return "OvalBorder(" + o.Side.String() + ")"
}
