package painting

// Port of rounded_rectangle_border.dart (Flutter 3.50.1):
// RoundedRectangleBorder, RoundedSuperellipseBorder and the private
// _ShapeToCircleBorder<T> lerp intermediates (_RoundedRectangleToCircleBorder,
// _RoundedSuperellipseToCircleBorder), which are one unexported type with a
// flag for T.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// radiusOrZero is the field's value, BorderRadius.zero when left nil (a
// struct literal without it).
func radiusOrZero(r BorderRadiusGeometry) BorderRadiusGeometry {
	if r == nil {
		return BorderRadiusZero
	}
	return r
}

func isBorderRadiusZero(r BorderRadiusGeometry) bool {
	return r == BorderRadiusGeometry(BorderRadiusZero)
}

// RoundedRectangleBorder is a rectangular border with rounded corners. It
// is a comparable value; build it with NewRoundedRectangleBorder.
type RoundedRectangleBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
	// BorderRadius is the radii for each corner (nil: BorderRadius.zero).
	BorderRadius BorderRadiusGeometry
}

// RRectBorderOptions are the named parameters of RoundedRectangleBorder,
// RoundedSuperellipseBorder, BeveledRectangleBorder and
// ContinuousRectangleBorder (nil Side: BorderSide.none; nil BorderRadius:
// BorderRadius.zero).
type RRectBorderOptions struct {
	Side         *BorderSide
	BorderRadius BorderRadiusGeometry
}

func (o RRectBorderOptions) values() (BorderSide, BorderRadiusGeometry) {
	return sideOrNone(o.Side), radiusOrZero(o.BorderRadius)
}

func rrectBorderOptions(opts []RRectBorderOptions) RRectBorderOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return RRectBorderOptions{}
}

// NewRoundedRectangleBorder is RoundedRectangleBorder({side, borderRadius}).
func NewRoundedRectangleBorder(opts ...RRectBorderOptions) RoundedRectangleBorder {
	s, r := rrectBorderOptions(opts).values()
	return RoundedRectangleBorder{s, r}
}

func (b RoundedRectangleBorder) radius() BorderRadiusGeometry { return radiusOrZero(b.BorderRadius) }

// GetSide is the border's side.
func (b RoundedRectangleBorder) GetSide() BorderSide { return b.Side }

// Dimensions is OutlinedBorder's.
func (b RoundedRectangleBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(b.Side) }

// AddBorder returns nil.
func (b RoundedRectangleBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (b RoundedRectangleBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

// Scale scales the side and the radii by t.
func (b RoundedRectangleBorder) Scale(t float64) ShapeBorder {
	return RoundedRectangleBorder{b.Side.Scale(t), b.radius().MulGeometry(t)}
}

// LerpFrom interpolates from a RoundedRectangleBorder, a CircleBorder
// (through the private to-circle border) or nil.
func (b RoundedRectangleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(RoundedRectangleBorder); ok {
		return RoundedRectangleBorder{BorderSideLerp(x.Side, b.Side, t), BorderRadiusGeometryLerp(x.radius(), b.radius(), t)}
	}
	if x, ok := asCircleBorder(a); ok {
		return newShapeToCircleBorder(false, BorderSideLerp(x.Side, b.Side, t), b.radius(), 1.0-t, x.Eccentricity)
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a RoundedRectangleBorder, a CircleBorder or nil.
func (b RoundedRectangleBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(RoundedRectangleBorder); ok {
		return RoundedRectangleBorder{BorderSideLerp(b.Side, x.Side, t), BorderRadiusGeometryLerp(b.radius(), x.radius(), t)}
	}
	if x, ok := asCircleBorder(other); ok {
		return newShapeToCircleBorder(false, BorderSideLerp(b.Side, x.Side, t), b.radius(), t, x.Eccentricity)
	}
	return shapeBorderLerpTo(b, other, t)
}

// RRectBorderCopyWithOptions are copyWith's named parameters (nil: keep).
type RRectBorderCopyWithOptions = RRectBorderOptions

// CopyWith returns a copy with the given fields replaced.
func (b RoundedRectangleBorder) CopyWith(opts ...RRectBorderCopyWithOptions) RoundedRectangleBorder {
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
func (b RoundedRectangleBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return b.CopyWith(RRectBorderCopyWithOptions{Side: side})
}

// GetInnerPath is the rounded rectangle deflated by the stroke inset.
func (b RoundedRectangleBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	borderRect := b.radius().Resolve(textDirection).ToRRect(rect)
	p := ui.NewPath()
	p.AddRRect(borderRect.Deflate(b.Side.StrokeInset()))
	return p
}

// GetOuterPath is the rounded rectangle.
func (b RoundedRectangleBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRRect(b.radius().Resolve(textDirection).ToRRect(rect))
	return p
}

// HitTest reports whether position is inside the rounded rectangle.
func (b RoundedRectangleBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	resolved := b.radius().Resolve(textDirection)
	if resolved == BorderRadiusZero {
		return rect.Contains(position)
	}
	return resolved.ToRRect(rect).Contains(position)
}

// PaintInterior draws the rectangle or rounded rectangle.
func (b RoundedRectangleBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	if isBorderRadiusZero(b.radius()) {
		canvas.DrawRect(rect, paint)
	} else {
		canvas.DrawRRect(b.radius().Resolve(textDirection).ToRRect(rect), paint)
	}
}

// PreferPaintInterior is true.
func (b RoundedRectangleBorder) PreferPaintInterior() bool { return true }

// Paint draws the outline (a hairline stroke, or the ring between the
// inner and outer rounded rectangles).
func (b RoundedRectangleBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if b.Side.Style != BorderStyleSolid {
		return
	}
	if b.Side.Width == 0.0 {
		canvas.DrawRRect(b.radius().Resolve(textDirection).ToRRect(rect), b.Side.ToPaint())
		return
	}
	paint := ui.NewPaint()
	paint.SetColor(b.Side.Color)
	borderRect := b.radius().Resolve(textDirection).ToRRect(rect)
	inner := borderRect.Deflate(b.Side.StrokeInset())
	outer := borderRect.Inflate(b.Side.StrokeOutset())
	canvas.DrawDRRect(outer, inner, paint)
}

func (b RoundedRectangleBorder) String() string {
	return "RoundedRectangleBorder(" + b.Side.String() + ", " + b.radius().String() + ")"
}

// RoundedSuperellipseBorder is a rectangular border with rounded corners
// following the shape of an RSuperellipse. Build it with
// NewRoundedSuperellipseBorder.
type RoundedSuperellipseBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
	// BorderRadius is the radii for each corner (nil: BorderRadius.zero).
	BorderRadius BorderRadiusGeometry
}

// NewRoundedSuperellipseBorder is RoundedSuperellipseBorder({side,
// borderRadius}).
func NewRoundedSuperellipseBorder(opts ...RRectBorderOptions) RoundedSuperellipseBorder {
	s, r := rrectBorderOptions(opts).values()
	return RoundedSuperellipseBorder{s, r}
}

func (b RoundedSuperellipseBorder) radius() BorderRadiusGeometry { return radiusOrZero(b.BorderRadius) }

// GetSide is the border's side.
func (b RoundedSuperellipseBorder) GetSide() BorderSide { return b.Side }

// Dimensions is OutlinedBorder's.
func (b RoundedSuperellipseBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(b.Side) }

// AddBorder returns nil.
func (b RoundedSuperellipseBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder {
	return nil
}

// Add is operator +.
func (b RoundedSuperellipseBorder) Add(other ShapeBorder) ShapeBorder {
	return shapeBorderPlus(b, other)
}

// Scale scales the side and the radii by t.
func (b RoundedSuperellipseBorder) Scale(t float64) ShapeBorder {
	return RoundedSuperellipseBorder{b.Side.Scale(t), b.radius().MulGeometry(t)}
}

// LerpFrom interpolates from a RoundedSuperellipseBorder, a CircleBorder
// or nil.
func (b RoundedSuperellipseBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(RoundedSuperellipseBorder); ok {
		return RoundedSuperellipseBorder{BorderSideLerp(x.Side, b.Side, t),
			radiusOrZero(BorderRadiusGeometryLerp(x.radius(), b.radius(), t))}
	}
	if x, ok := asCircleBorder(a); ok {
		return newShapeToCircleBorder(true, BorderSideLerp(x.Side, b.Side, t), b.radius(), 1.0-t, x.Eccentricity)
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a RoundedSuperellipseBorder, a CircleBorder or
// nil.
func (b RoundedSuperellipseBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(RoundedSuperellipseBorder); ok {
		return RoundedSuperellipseBorder{BorderSideLerp(b.Side, x.Side, t),
			radiusOrZero(BorderRadiusGeometryLerp(b.radius(), x.radius(), t))}
	}
	if x, ok := asCircleBorder(other); ok {
		return newShapeToCircleBorder(true, BorderSideLerp(b.Side, x.Side, t), b.radius(), t, x.Eccentricity)
	}
	return shapeBorderLerpTo(b, other, t)
}

// CopyWith returns a copy with the given fields replaced.
func (b RoundedSuperellipseBorder) CopyWith(opts ...RRectBorderCopyWithOptions) RoundedSuperellipseBorder {
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
func (b RoundedSuperellipseBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return b.CopyWith(RRectBorderCopyWithOptions{Side: side})
}

// GetInnerPath is the superellipse (or rectangle) deflated by the stroke
// inset.
func (b RoundedSuperellipseBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	if isBorderRadiusZero(b.radius()) {
		p.AddRect(rect.Deflate(b.Side.StrokeInset()))
	} else {
		borderRect := b.radius().Resolve(textDirection).ToRSuperellipse(rect)
		p.AddRSuperellipse(borderRect.Deflate(b.Side.StrokeInset()))
	}
	return p
}

// GetOuterPath is the superellipse (or rectangle).
func (b RoundedSuperellipseBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	if isBorderRadiusZero(b.radius()) {
		p.AddRect(rect)
	} else {
		p.AddRSuperellipse(b.radius().Resolve(textDirection).ToRSuperellipse(rect))
	}
	return p
}

// HitTest reports whether position is inside the superellipse.
func (b RoundedSuperellipseBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	resolved := b.radius().Resolve(textDirection)
	if resolved == BorderRadiusZero {
		return rect.Contains(position)
	}
	return resolved.ToRSuperellipse(rect).Contains(position)
}

// PaintInterior draws the rectangle or superellipse.
func (b RoundedSuperellipseBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	if isBorderRadiusZero(b.radius()) {
		canvas.DrawRect(rect, paint)
	} else {
		canvas.DrawRSuperellipse(b.radius().Resolve(textDirection).ToRSuperellipse(rect), paint)
	}
}

// PreferPaintInterior is true.
func (b RoundedSuperellipseBorder) PreferPaintInterior() bool { return true }

// Paint strokes the superellipse (or rectangle) with the side.
func (b RoundedSuperellipseBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if b.Side.Style != BorderStyleSolid {
		return
	}
	strokeOffset := (b.Side.StrokeOutset() - b.Side.StrokeInset()) / 2
	if isBorderRadiusZero(b.radius()) {
		canvas.DrawRect(rect.Inflate(strokeOffset), b.Side.ToPaint())
	} else {
		base := b.radius().Resolve(textDirection).ToRSuperellipse(rect).Inflate(strokeOffset)
		canvas.DrawRSuperellipse(base, b.Side.ToPaint())
	}
}

func (b RoundedSuperellipseBorder) String() string {
	return "RoundedSuperellipseBorder(" + b.Side.String() + ", " + b.radius().String() + ")"
}

// shapeToCircleBorder is _ShapeToCircleBorder<T>: T is
// RoundedSuperellipseBorder when superellipse, RoundedRectangleBorder
// otherwise. Dart's == ignores the eccentricity: compare with Equal.
type shapeToCircleBorder struct {
	superellipse bool
	side         BorderSide
	borderRadius BorderRadiusGeometry
	circularity  float64
	eccentricity float64
}

func newShapeToCircleBorder(superellipse bool, side BorderSide, borderRadius BorderRadiusGeometry,
	circularity, eccentricity float64) shapeToCircleBorder {
	return shapeToCircleBorder{superellipse, side, radiusOrZero(borderRadius), circularity, eccentricity}
}

func (b shapeToCircleBorder) typeName() string {
	if b.superellipse {
		return "RoundedSuperellipseBorder"
	}
	return "RoundedRectangleBorder"
}

// isT is Dart's `x is T`; it returns x's side and radius.
func (b shapeToCircleBorder) isT(x ShapeBorder) (BorderSide, BorderRadiusGeometry, bool) {
	if b.superellipse {
		if r, ok := x.(RoundedSuperellipseBorder); ok {
			return r.Side, r.radius(), true
		}
	} else if r, ok := x.(RoundedRectangleBorder); ok {
		return r.Side, r.radius(), true
	}
	return BorderSide{}, nil, false
}

func (b shapeToCircleBorder) with(side BorderSide, borderRadius BorderRadiusGeometry, circularity, eccentricity float64) shapeToCircleBorder {
	if borderRadius == nil {
		borderRadius = b.borderRadius
	}
	return shapeToCircleBorder{b.superellipse, side, borderRadius, circularity, eccentricity}
}

func (b shapeToCircleBorder) GetSide() BorderSide { return b.side }

func (b shapeToCircleBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(b.side) }

func (b shapeToCircleBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

func (b shapeToCircleBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

func (b shapeToCircleBorder) Scale(t float64) ShapeBorder {
	return b.with(b.side.Scale(t), b.borderRadius.MulGeometry(t), t, b.eccentricity)
}

func (b shapeToCircleBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if side, radius, ok := b.isT(a); ok {
		return b.with(BorderSideLerp(side, b.side, t), BorderRadiusGeometryLerp(radius, b.borderRadius, t),
			b.circularity*t, b.eccentricity)
	}
	if x, ok := asCircleBorder(a); ok {
		return b.with(BorderSideLerp(x.Side, b.side, t), b.borderRadius,
			b.circularity+float64((1.0-b.circularity)*(1.0-t)), x.Eccentricity)
	}
	if x, ok := a.(shapeToCircleBorder); ok && x.superellipse == b.superellipse {
		return b.with(BorderSideLerp(x.side, b.side, t), BorderRadiusGeometryLerp(x.borderRadius, b.borderRadius, t),
			borderLerpDouble(x.circularity, b.circularity, t), b.eccentricity)
	}
	return shapeBorderLerpFrom(b, a, t)
}

func (b shapeToCircleBorder) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if side, radius, ok := b.isT(other); ok {
		return b.with(BorderSideLerp(b.side, side, t), BorderRadiusGeometryLerp(b.borderRadius, radius, t),
			b.circularity*(1.0-t), b.eccentricity)
	}
	if x, ok := asCircleBorder(other); ok {
		return b.with(BorderSideLerp(b.side, x.Side, t), b.borderRadius,
			b.circularity+float64((1.0-b.circularity)*t), x.Eccentricity)
	}
	if x, ok := other.(shapeToCircleBorder); ok && x.superellipse == b.superellipse {
		return b.with(BorderSideLerp(b.side, x.side, t), BorderRadiusGeometryLerp(b.borderRadius, x.borderRadius, t),
			borderLerpDouble(b.circularity, x.circularity, t), b.eccentricity)
	}
	return shapeBorderLerpTo(b, other, t)
}

func (b shapeToCircleBorder) adjustRect(rect ui.Rect) ui.Rect {
	return toCircleAdjustRect(b.circularity, b.eccentricity, rect)
}

// toCircleAdjustRect is _ShapeToCircleBorder's and _StadiumToCircleBorder's
// _adjustRect.
func toCircleAdjustRect(circularity, eccentricity float64, rect ui.Rect) ui.Rect {
	if circularity == 0.0 || rect.Width() == rect.Height() {
		return rect
	}
	if rect.Width() < rect.Height() {
		partialDelta := (rect.Height() - rect.Width()) / 2
		delta := float64(circularity * partialDelta * (1.0 - eccentricity))
		return ui.NewRectFromLTRB(rect.Left, rect.Top+delta, rect.Right, rect.Bottom-delta)
	}
	partialDelta := (rect.Width() - rect.Height()) / 2
	delta := float64(circularity * partialDelta * (1.0 - eccentricity))
	return ui.NewRectFromLTRB(rect.Left+delta, rect.Top, rect.Right-delta, rect.Bottom)
}

// ovalRadius is BorderRadius.all(Radius.elliptical(...)) toward which the
// to-circle borders lerp an eccentric shape.
func ovalRadius(eccentricity float64, rect ui.Rect) BorderRadius {
	if rect.Width() < rect.Height() {
		return NewBorderRadiusAll(ui.NewRadiusElliptical(rect.Width()/2, (0.5+eccentricity/2)*rect.Height()/2))
	}
	return NewBorderRadiusAll(ui.NewRadiusElliptical((0.5+eccentricity/2)*rect.Width()/2, rect.Height()/2))
}

func (b shapeToCircleBorder) adjustBorderRadius(rect ui.Rect, textDirection *ui.TextDirection) BorderRadius {
	resolved := b.borderRadius.Resolve(textDirection)
	if b.circularity == 0.0 {
		return resolved
	}
	if b.eccentricity != 0.0 {
		target := ovalRadius(b.eccentricity, rect)
		return *BorderRadiusLerp(&resolved, &target, b.circularity)
	}
	target := NewBorderRadiusCircular(rect.ShortestSide() / 2)
	return *BorderRadiusLerp(&resolved, &target, b.circularity)
}

func (b shapeToCircleBorder) buildPath(rect ui.Rect, radius BorderRadius, inflation *float64) *ui.Path {
	p := ui.NewPath()
	if b.superellipse {
		r := radius.ToRSuperellipse(rect)
		if inflation != nil {
			r = r.Inflate(*inflation)
		}
		p.AddRSuperellipse(r)
	} else {
		r := radius.ToRRect(rect)
		if inflation != nil {
			r = r.Inflate(*inflation)
		}
		p.AddRRect(r)
	}
	return p
}

func (b shapeToCircleBorder) drawShape(canvas ui.Canvas, rect ui.Rect, radius BorderRadius, paint *ui.Paint, inflation *float64) {
	if b.superellipse {
		r := radius.ToRSuperellipse(rect)
		if inflation != nil {
			r = r.Inflate(*inflation)
		}
		canvas.DrawRSuperellipse(r, paint)
	} else {
		r := radius.ToRRect(rect)
		if inflation != nil {
			r = r.Inflate(*inflation)
		}
		canvas.DrawRRect(r, paint)
	}
}

func (b shapeToCircleBorder) containsOuterShape(rect ui.Rect, radius BorderRadius, position ui.Offset) bool {
	if b.superellipse {
		return radius.ToRSuperellipse(rect).Contains(position)
	}
	return radius.ToRRect(rect).Contains(position)
}

func (b shapeToCircleBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	inflation := -borderLerpDouble(b.side.Width, 0, b.side.StrokeAlign)
	return b.buildPath(b.adjustRect(rect), b.adjustBorderRadius(rect, textDirection), &inflation)
}

func (b shapeToCircleBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return b.buildPath(b.adjustRect(rect), b.adjustBorderRadius(rect, textDirection), nil)
}

func (b shapeToCircleBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	adjustedRect := b.adjustRect(rect)
	adjustedRadius := b.adjustBorderRadius(rect, textDirection)
	if adjustedRadius == BorderRadiusZero {
		return adjustedRect.Contains(position)
	}
	return b.containsOuterShape(adjustedRect, adjustedRadius, position)
}

func (b shapeToCircleBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	adjustedRadius := b.adjustBorderRadius(rect, textDirection)
	if adjustedRadius == BorderRadiusZero {
		canvas.DrawRect(b.adjustRect(rect), paint)
	} else {
		b.drawShape(canvas, b.adjustRect(rect), adjustedRadius, paint, nil)
	}
}

func (b shapeToCircleBorder) PreferPaintInterior() bool { return true }

func (b shapeToCircleBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	if side == nil {
		return b
	}
	return b.with(*side, nil, b.circularity, b.eccentricity)
}

func (b shapeToCircleBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if b.side.Style != BorderStyleSolid {
		return
	}
	inflation := b.side.StrokeOffset() / 2
	b.drawShape(canvas, b.adjustRect(rect), b.adjustBorderRadius(rect, textDirection), b.side.ToPaint(), &inflation)
}

// Equal is Dart's ==: the same T, side, radius and circularity (not the
// eccentricity).
func (b shapeToCircleBorder) Equal(other ShapeBorder) bool {
	o, ok := other.(shapeToCircleBorder)
	return ok && o.superellipse == b.superellipse && o.side == b.side && o.borderRadius == b.borderRadius &&
		o.circularity == b.circularity
}

func percent1(v float64) string { return dart.ToStringAsFixed(v*100, 1) }

func (b shapeToCircleBorder) String() string {
	if b.eccentricity != 0.0 {
		return b.typeName() + "(" + b.side.String() + ", " + b.borderRadius.String() + ", " + percent1(b.circularity) +
			"% of the way to being a CircleBorder that is " + percent1(b.eccentricity) + "% oval)"
	}
	return b.typeName() + "(" + b.side.String() + ", " + b.borderRadius.String() + ", " + percent1(b.circularity) +
		"% of the way to being a CircleBorder)"
}
