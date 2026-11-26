package painting

// Port of stadium_border.dart (Flutter 3.50.1): StadiumBorder and its
// private lerp intermediates _StadiumToCircleBorder and
// _StadiumToRoundedRectangleBorder.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// StadiumBorder is a border that fits a stadium-shaped border (a box with
// semicircles on the ends). Build it with NewStadiumBorder.
type StadiumBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
}

// StadiumBorderOptions are StadiumBorder's named parameters (nil Side:
// BorderSide.none).
type StadiumBorderOptions struct{ Side *BorderSide }

// NewStadiumBorder is StadiumBorder({side}).
func NewStadiumBorder(opts ...StadiumBorderOptions) StadiumBorder {
	var o StadiumBorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return StadiumBorder{sideOrNone(o.Side)}
}

// GetSide is the border's side.
func (b StadiumBorder) GetSide() BorderSide { return b.Side }

// Dimensions is OutlinedBorder's.
func (b StadiumBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(b.Side) }

// AddBorder returns nil.
func (b StadiumBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (b StadiumBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

// Scale scales the side by t.
func (b StadiumBorder) Scale(t float64) ShapeBorder { return StadiumBorder{b.Side.Scale(t)} }

// LerpFrom interpolates from a StadiumBorder, CircleBorder,
// RoundedRectangleBorder or nil.
func (b StadiumBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(StadiumBorder); ok {
		return StadiumBorder{BorderSideLerp(x.Side, b.Side, t)}
	}
	if x, ok := asCircleBorder(a); ok {
		return stadiumToCircleBorder{BorderSideLerp(x.Side, b.Side, t), 1.0 - t, x.Eccentricity}
	}
	if x, ok := a.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(x.Side, b.Side, t), x.radius(), 1.0 - t}
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a StadiumBorder, CircleBorder,
// RoundedRectangleBorder or nil.
func (b StadiumBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(StadiumBorder); ok {
		return StadiumBorder{BorderSideLerp(b.Side, x.Side, t)}
	}
	if x, ok := asCircleBorder(other); ok {
		return stadiumToCircleBorder{BorderSideLerp(b.Side, x.Side, t), t, x.Eccentricity}
	}
	if x, ok := other.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(b.Side, x.Side, t), x.radius(), t}
	}
	return shapeBorderLerpTo(b, other, t)
}

// CopyWith is copyWith({side}) (nil: keep).
func (b StadiumBorder) CopyWith(side *BorderSide) StadiumBorder {
	if side != nil {
		b.Side = *side
	}
	return b
}

// CopyWithSide is copyWith({side}).
func (b StadiumBorder) CopyWithSide(side *BorderSide) OutlinedBorder { return b.CopyWith(side) }

func stadiumRRect(rect ui.Rect) ui.RRect {
	return ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(rect.ShortestSide()/2.0))
}

// GetInnerPath is the stadium deflated by the stroke inset.
func (b StadiumBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(stadiumRRect(rect).Deflate(b.Side.StrokeInset()))
	return p
}

// GetOuterPath is the stadium.
func (b StadiumBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(stadiumRRect(rect))
	return p
}

// HitTest reports whether position is inside the stadium.
func (b StadiumBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return stadiumRRect(rect).Contains(position)
}

// PaintInterior draws the stadium.
func (b StadiumBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	canvas.DrawRRect(stadiumRRect(rect), paint)
}

// PreferPaintInterior is true.
func (b StadiumBorder) PreferPaintInterior() bool { return true }

// Paint strokes the stadium with the side.
func (b StadiumBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if b.Side.Style != BorderStyleSolid {
		return
	}
	borderRect := ui.NewRRectFromRectAndRadius(rect, ui.NewRadiusCircular(rect.ShortestSide()/2))
	canvas.DrawRRect(borderRect.Inflate(b.Side.StrokeOffset()/2), b.Side.ToPaint())
}

func (b StadiumBorder) String() string { return "StadiumBorder(" + b.Side.String() + ")" }

// stadiumToCircleBorder is _StadiumToCircleBorder. Dart's == ignores the
// eccentricity: compare with Equal.
type stadiumToCircleBorder struct {
	side         BorderSide
	circularity  float64
	eccentricity float64
}

func (b stadiumToCircleBorder) GetSide() BorderSide { return b.side }

func (b stadiumToCircleBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(b.side) }

func (b stadiumToCircleBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

func (b stadiumToCircleBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

func (b stadiumToCircleBorder) Scale(t float64) ShapeBorder {
	return stadiumToCircleBorder{b.side.Scale(t), t, b.eccentricity}
}

func (b stadiumToCircleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(StadiumBorder); ok {
		return stadiumToCircleBorder{BorderSideLerp(x.Side, b.side, t), b.circularity * t, b.eccentricity}
	}
	if x, ok := asCircleBorder(a); ok {
		return stadiumToCircleBorder{BorderSideLerp(x.Side, b.side, t),
			b.circularity + float64((1.0-b.circularity)*(1.0-t)), x.Eccentricity}
	}
	if x, ok := a.(stadiumToCircleBorder); ok {
		return stadiumToCircleBorder{BorderSideLerp(x.side, b.side, t),
			borderLerpDouble(x.circularity, b.circularity, t), borderLerpDouble(x.eccentricity, b.eccentricity, t)}
	}
	return shapeBorderLerpFrom(b, a, t)
}

func (b stadiumToCircleBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(StadiumBorder); ok {
		return stadiumToCircleBorder{BorderSideLerp(b.side, x.Side, t), b.circularity * (1.0 - t), b.eccentricity}
	}
	if x, ok := asCircleBorder(other); ok {
		return stadiumToCircleBorder{BorderSideLerp(b.side, x.Side, t),
			b.circularity + float64((1.0-b.circularity)*t), x.Eccentricity}
	}
	if x, ok := other.(stadiumToCircleBorder); ok {
		return stadiumToCircleBorder{BorderSideLerp(b.side, x.side, t),
			borderLerpDouble(b.circularity, x.circularity, t), borderLerpDouble(b.eccentricity, x.eccentricity, t)}
	}
	return shapeBorderLerpTo(b, other, t)
}

func (b stadiumToCircleBorder) adjustBorderRadius(rect ui.Rect) BorderRadius {
	circleRadius := NewBorderRadiusCircular(rect.ShortestSide() / 2)
	if b.eccentricity != 0.0 {
		target := ovalRadius(b.eccentricity, rect)
		return *BorderRadiusLerp(&circleRadius, &target, b.circularity)
	}
	return circleRadius
}

func (b stadiumToCircleBorder) rrect(rect ui.Rect) ui.RRect {
	return b.adjustBorderRadius(rect).ToRRect(toCircleAdjustRect(b.circularity, b.eccentricity, rect))
}

func (b stadiumToCircleBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.rrect(rect).Deflate(b.side.StrokeInset()))
	return p
}

func (b stadiumToCircleBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.rrect(rect))
	return p
}

func (b stadiumToCircleBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return b.rrect(rect).Contains(position)
}

func (b stadiumToCircleBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	canvas.DrawRRect(b.rrect(rect), paint)
}

func (b stadiumToCircleBorder) PreferPaintInterior() bool { return true }

func (b stadiumToCircleBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	if side != nil {
		b.side = *side
	}
	return b
}

func (b stadiumToCircleBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if b.side.Style != BorderStyleSolid {
		return
	}
	canvas.DrawRRect(b.rrect(rect).Inflate(b.side.StrokeOffset()/2), b.side.ToPaint())
}

// Equal is Dart's ==: the same side and circularity.
func (b stadiumToCircleBorder) Equal(other ShapeBorder) bool {
	o, ok := other.(stadiumToCircleBorder)
	return ok && o.side == b.side && o.circularity == b.circularity
}

func (b stadiumToCircleBorder) String() string {
	if b.eccentricity != 0.0 {
		return "StadiumBorder(" + b.side.String() + ", " + percent1(b.circularity) +
			"% of the way to being a CircleBorder that is " + percent1(b.eccentricity) + "% oval)"
	}
	return "StadiumBorder(" + b.side.String() + ", " + percent1(b.circularity) + "% of the way to being a CircleBorder)"
}

// stadiumToRoundedRectangleBorder is _StadiumToRoundedRectangleBorder.
type stadiumToRoundedRectangleBorder struct {
	side           BorderSide
	borderRadius   BorderRadiusGeometry
	rectilinearity float64
}

func (b stadiumToRoundedRectangleBorder) GetSide() BorderSide { return b.side }

func (b stadiumToRoundedRectangleBorder) Dimensions() EdgeInsetsGeometry {
	return outlinedDimensions(b.side)
}

func (b stadiumToRoundedRectangleBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder {
	return nil
}

func (b stadiumToRoundedRectangleBorder) Add(other ShapeBorder) ShapeBorder {
	return shapeBorderPlus(b, other)
}

func (b stadiumToRoundedRectangleBorder) Scale(t float64) ShapeBorder {
	return stadiumToRoundedRectangleBorder{b.side.Scale(t), b.borderRadius.MulGeometry(t), t}
}

func (b stadiumToRoundedRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(StadiumBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(x.Side, b.side, t), b.borderRadius, b.rectilinearity * t}
	}
	if x, ok := a.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(x.Side, b.side, t), b.borderRadius,
			b.rectilinearity + float64((1.0-b.rectilinearity)*(1.0-t))}
	}
	if x, ok := a.(stadiumToRoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(x.side, b.side, t),
			BorderRadiusGeometryLerp(x.borderRadius, b.borderRadius, t), borderLerpDouble(x.rectilinearity, b.rectilinearity, t)}
	}
	return shapeBorderLerpFrom(b, a, t)
}

func (b stadiumToRoundedRectangleBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(StadiumBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(b.side, x.Side, t), b.borderRadius, b.rectilinearity * (1.0 - t)}
	}
	if x, ok := other.(RoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(b.side, x.Side, t), b.borderRadius,
			b.rectilinearity + float64((1.0-b.rectilinearity)*t)}
	}
	if x, ok := other.(stadiumToRoundedRectangleBorder); ok {
		return stadiumToRoundedRectangleBorder{BorderSideLerp(b.side, x.side, t),
			BorderRadiusGeometryLerp(b.borderRadius, x.borderRadius, t), borderLerpDouble(b.rectilinearity, x.rectilinearity, t)}
	}
	return shapeBorderLerpTo(b, other, t)
}

func (b stadiumToRoundedRectangleBorder) adjustBorderRadius(rect ui.Rect) BorderRadiusGeometry {
	return BorderRadiusGeometryLerp(b.borderRadius, NewBorderRadiusAll(ui.NewRadiusCircular(rect.ShortestSide()/2.0)),
		1.0-b.rectilinearity)
}

func (b stadiumToRoundedRectangleBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	borderRect := b.adjustBorderRadius(rect).Resolve(textDirection).ToRRect(rect)
	p := ui.NewPath()
	p.AddRRect(borderRect.Deflate(borderLerpDouble(b.side.Width, 0, b.side.StrokeAlign)))
	return p
}

func (b stadiumToRoundedRectangleBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.adjustBorderRadius(rect).Resolve(textDirection).ToRRect(rect))
	return p
}

func (b stadiumToRoundedRectangleBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	adjusted := b.adjustBorderRadius(rect).Resolve(textDirection)
	if adjusted == BorderRadiusZero {
		return rect.Contains(position)
	}
	return adjusted.ToRRect(rect).Contains(position)
}

func (b stadiumToRoundedRectangleBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	adjusted := b.adjustBorderRadius(rect)
	if isBorderRadiusZero(adjusted) {
		canvas.DrawRect(rect, paint)
	} else {
		canvas.DrawRRect(adjusted.Resolve(textDirection).ToRRect(rect), paint)
	}
}

func (b stadiumToRoundedRectangleBorder) PreferPaintInterior() bool { return true }

func (b stadiumToRoundedRectangleBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	if side != nil {
		b.side = *side
	}
	return b
}

func (b stadiumToRoundedRectangleBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if b.side.Style != BorderStyleSolid {
		return
	}
	borderRect := b.adjustBorderRadius(rect).Resolve(textDirection).ToRRect(rect)
	canvas.DrawRRect(borderRect.Inflate(b.side.StrokeOffset()/2), b.side.ToPaint())
}

func (b stadiumToRoundedRectangleBorder) String() string {
	return "StadiumBorder(" + b.side.String() + ", " + b.borderRadius.String() + ", " + percent1(b.rectilinearity) +
		"% of the way to being a RoundedRectangleBorder)"
}
