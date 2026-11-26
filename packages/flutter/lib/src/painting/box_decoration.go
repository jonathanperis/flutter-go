package painting

// Port of box_decoration.dart (Flutter 3.50.1): BoxDecoration and its
// painter.
//
// BoxDecoration is used by pointer (it is Diagnosticable) and built with
// NewBoxDecoration(BoxDecorationOptions), which runs Dart's constructor
// assertion. Its fields are Dart's final fields; nil is null (Color and
// BackgroundBlendMode are pointers). Dart's identical(a, b) in lerp is
// pointer identity: two equal decorations built separately are not
// identical (Dart canonicalizes equal const instances), which only changes
// whether lerp returns a itself or an equal value.

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// BoxDecoration is an immutable description of how to paint a box: a
// background color or gradient, an image, a border, rounded corners and
// shadows.
type BoxDecoration struct {
	DecorationBase
	// Color to fill in the background of the box (nil: none).
	Color *ui.Color
	// Image to paint above the background color or gradient (nil: none).
	Image DecorationImage
	// Border to draw above the background color, gradient or image (nil:
	// none).
	Border BoxBorder
	// BorderRadius rounds the corners of a rectangle (nil: square
	// corners). Only for BoxShapeRectangle.
	BorderRadius BorderRadiusGeometry
	// BoxShadow is the list of shadows cast by the box (nil: none).
	BoxShadow []BoxShadow
	// Gradient to use when filling the box (nil: none).
	Gradient Gradient
	// BackgroundBlendMode is the blend mode applied to the color or
	// gradient background (nil: the default, srcOver).
	BackgroundBlendMode *ui.BlendMode
	// Shape to fill the background color, gradient and image into and to
	// cast as the shadow.
	Shape BoxShape
}

// BoxDecorationOptions are BoxDecoration's named parameters (zero Shape:
// BoxShape.rectangle, Dart's default).
type BoxDecorationOptions struct {
	Color               *ui.Color
	Image               DecorationImage
	Border              BoxBorder
	BorderRadius        BorderRadiusGeometry
	BoxShadow           []BoxShadow
	Gradient            Gradient
	BackgroundBlendMode *ui.BlendMode
	Shape               BoxShape
}

// NewBoxDecoration is BoxDecoration({...}); it panics when
// backgroundBlendMode is given without a color or gradient, as Dart
// asserts.
func NewBoxDecoration(opts ...BoxDecorationOptions) *BoxDecoration {
	var o BoxDecorationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if !(o.BackgroundBlendMode == nil || o.Color != nil || !isNilGradient(o.Gradient)) {
		panic(dart.NewAssertionError("backgroundBlendMode applies to BoxDecoration's background color or " +
			"gradient, but no color or gradient was provided."))
	}
	d := &BoxDecoration{Color: o.Color, Image: o.Image, Border: o.Border, BorderRadius: o.BorderRadius,
		BoxShadow: o.BoxShadow, Gradient: o.Gradient, BackgroundBlendMode: o.BackgroundBlendMode, Shape: o.Shape}
	d.normalize()
	d.SetSelf(d)
	return d
}

// normalize turns nil pointers held in the interface fields into nil
// interfaces (Dart's null).
func (d *BoxDecoration) normalize() {
	if isNilValue(d.Image) {
		d.Image = nil
	}
	if isNilValue(d.Border) {
		d.Border = nil
	}
	if isNilValue(d.BorderRadius) {
		d.BorderRadius = nil
	}
	if isNilGradient(d.Gradient) {
		d.Gradient = nil
	}
}

// BoxDecorationCopyWithOptions are copyWith's named parameters (nil keeps
// the current value).
type BoxDecorationCopyWithOptions struct {
	Color               *ui.Color
	Image               DecorationImage
	Border              BoxBorder
	BorderRadius        BorderRadiusGeometry
	BoxShadow           []BoxShadow
	Gradient            Gradient
	BackgroundBlendMode *ui.BlendMode
	Shape               *BoxShape
}

// CopyWith creates a copy of this object but with the given fields
// replaced with the new values.
func (d *BoxDecoration) CopyWith(o BoxDecorationCopyWithOptions) *BoxDecoration {
	r := BoxDecorationOptions{Color: d.Color, Image: d.Image, Border: d.Border, BorderRadius: d.BorderRadius,
		BoxShadow: d.BoxShadow, Gradient: d.Gradient, BackgroundBlendMode: d.BackgroundBlendMode, Shape: d.Shape}
	if o.Color != nil {
		r.Color = o.Color
	}
	if !isNilValue(o.Image) {
		r.Image = o.Image
	}
	if !isNilValue(o.Border) {
		r.Border = o.Border
	}
	if !isNilValue(o.BorderRadius) {
		r.BorderRadius = o.BorderRadius
	}
	if o.BoxShadow != nil {
		r.BoxShadow = o.BoxShadow
	}
	if !isNilGradient(o.Gradient) {
		r.Gradient = o.Gradient
	}
	if o.BackgroundBlendMode != nil {
		r.BackgroundBlendMode = o.BackgroundBlendMode
	}
	if o.Shape != nil {
		r.Shape = *o.Shape
	}
	return NewBoxDecoration(r)
}

const circleWithBorderRadius = "A circle cannot have a border radius. Remove either the shape or the borderRadius argument."

// DebugAssertIsValid asserts that a circle has no border radius.
func (d *BoxDecoration) DebugAssertIsValid() bool {
	if !(d.Shape != BoxShapeCircle || d.BorderRadius == nil) {
		panic(dart.NewAssertionError(circleWithBorderRadius))
	}
	return d.DecorationBase.DebugAssertIsValid()
}

// Padding is the border's dimensions (EdgeInsets.zero without a border).
func (d *BoxDecoration) Padding() EdgeInsetsGeometry {
	if d.Border != nil {
		if p := d.Border.Dimensions(); !isNilValue(p) {
			return p
		}
	}
	return EdgeInsetsZero
}

// GetClipPath is the outline of the box's shape (the circle, or the
// rectangle with its resolved border radius).
func (d *BoxDecoration) GetClipPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	switch d.Shape {
	case BoxShapeCircle:
		center := rect.Center()
		radius := rect.ShortestSide() / 2.0
		square := ui.NewRectFromCircle(center, radius)
		p := ui.NewPath()
		p.AddOval(square)
		return p
	default:
		if d.BorderRadius != nil {
			p := ui.NewPath()
			p.AddRRect(d.BorderRadius.Resolve(&textDirection).ToRRect(rect))
			return p
		}
		p := ui.NewPath()
		p.AddRect(rect)
		return p
	}
}

// Scale returns a new box decoration that is scaled by the given factor
// (backgroundBlendMode is dropped, as in Dart).
func (d *BoxDecoration) Scale(factor float64) *BoxDecoration {
	var gradient Gradient
	if d.Gradient != nil {
		gradient = d.Gradient.Scale(factor)
	}
	return NewBoxDecoration(BoxDecorationOptions{
		Color:        ui.ColorLerp(nil, d.Color, factor),
		Image:        DecorationImageLerp(nil, d.Image, factor),
		Border:       BoxBorderLerp(nil, d.Border, factor),
		BorderRadius: BorderRadiusGeometryLerp(nil, d.BorderRadius, factor),
		BoxShadow:    BoxShadowLerpList(nil, d.BoxShadow, factor),
		Gradient:     gradient,
		Shape:        d.Shape,
	})
}

// IsComplex is true when the decoration has shadows.
func (d *BoxDecoration) IsComplex() bool { return d.BoxShadow != nil }

func nilIfNilBox(d *BoxDecoration) Decoration {
	if d == nil {
		return nil
	}
	return d
}

// LerpFrom lerps from a when it is a BoxDecoration or null.
func (d *BoxDecoration) LerpFrom(a Decoration, t float64) Decoration {
	if isNilValue(a) {
		return d.Scale(t)
	}
	if ba, ok := a.(*BoxDecoration); ok {
		return nilIfNilBox(BoxDecorationLerp(ba, d, t))
	}
	return d.DecorationBase.LerpFrom(a, t)
}

// LerpTo lerps to b when it is a BoxDecoration or null.
func (d *BoxDecoration) LerpTo(b Decoration, t float64) Decoration {
	if isNilValue(b) {
		return d.Scale(1.0 - t)
	}
	if bb, ok := b.(*BoxDecoration); ok {
		return nilIfNilBox(BoxDecorationLerp(d, bb, t))
	}
	return d.DecorationBase.LerpTo(b, t)
}

// BoxDecorationLerp is BoxDecoration.lerp: linearly interpolates between
// two box decorations (nil is null). The shape switches at t = 0.5;
// backgroundBlendMode is dropped (as in Dart).
func BoxDecorationLerp(a, b *BoxDecoration, t float64) *BoxDecoration {
	if a == b {
		return a
	}
	if a == nil {
		return b.Scale(t)
	}
	if b == nil {
		return a.Scale(1.0 - t)
	}
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	shape := b.Shape
	if t < 0.5 {
		shape = a.Shape
	}
	return NewBoxDecoration(BoxDecorationOptions{
		Color:        ui.ColorLerp(a.Color, b.Color, t),
		Image:        DecorationImageLerp(a.Image, b.Image, t),
		Border:       BoxBorderLerp(a.Border, b.Border, t),
		BorderRadius: BorderRadiusGeometryLerp(a.BorderRadius, b.BorderRadius, t),
		BoxShadow:    BoxShadowLerpList(a.BoxShadow, b.BoxShadow, t),
		Gradient:     GradientLerp(a.Gradient, b.Gradient, t),
		Shape:        shape,
	})
}

func boxBordersEqual(a, b BoxBorder) bool {
	var sa, sb ShapeBorder
	if a != nil {
		sa = a
	}
	if b != nil {
		sb = b
	}
	return ShapeBorderEqual(sa, sb)
}

// Equal is ==: the same runtime type and equal fields.
func (d *BoxDecoration) Equal(other Decoration) bool {
	o, ok := other.(*BoxDecoration)
	if !ok || o == nil {
		return false
	}
	if d == o {
		return true
	}
	return ptrEq(o.Color, d.Color) &&
		decorationImagesEqual(o.Image, d.Image) &&
		boxBordersEqual(o.Border, d.Border) &&
		o.BorderRadius == d.BorderRadius &&
		foundation.ListEquals(o.BoxShadow, d.BoxShadow) &&
		GradientsEqual(o.Gradient, d.Gradient) &&
		ptrEq(o.BackgroundBlendMode, d.BackgroundBlendMode) &&
		o.Shape == d.Shape
}

// DebugFillProperties adds the decoration's fields (whitespace style,
// "<no decorations specified>" when empty).
func (d *BoxDecoration) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DecorationBase.DebugFillProperties(properties)
	properties.DefaultDiagnosticsTreeStyle = foundation.DiagnosticsTreeStyleWhitespace
	empty := "<no decorations specified>"
	properties.EmptyBodyDescription = &empty
	properties.Add(NewColorProperty("color", d.Color, ColorPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDiagnosticsProperty("image", d.Image,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDiagnosticsProperty("border", d.Border,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDiagnosticsProperty("borderRadius", d.BorderRadius,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	whitespace := foundation.DiagnosticsTreeStyleWhitespace
	properties.Add(foundation.NewIterableProperty("boxShadow", d.BoxShadow,
		foundation.IterablePropertyOptions{DefaultValue: foundation.Null, Style: &whitespace}))
	properties.Add(foundation.NewDiagnosticsProperty("gradient", d.Gradient,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewEnumProperty("shape", d.Shape, foundation.EnumPropertyOptions{DefaultValue: BoxShapeRectangle}))
}

// HitTest reports whether position (inside size, as Dart asserts) hits the
// box's shape.
func (d *BoxDecoration) HitTest(size ui.Size, position ui.Offset, textDirection *ui.TextDirection) bool {
	if !ui.OffsetZero.And(size).Contains(position) {
		panic(dart.NewAssertionError(nil))
	}
	switch d.Shape {
	case BoxShapeCircle:
		// Circles are inscribed into our smallest dimension.
		center := size.Center(ui.OffsetZero)
		radius := math.Min(size.Width, size.Height) / 2.0
		// Comparing squared distances avoids computing sqrt(dx * dx + dy * dy).
		return position.Sub(center).DistanceSquared() <= float64(radius*radius)
	default:
		if d.BorderRadius != nil {
			bounds := d.BorderRadius.Resolve(textDirection).ToRRect(ui.OffsetZero.And(size))
			return bounds.Contains(position)
		}
		return true
	}
}

// CreateBoxPainter returns the decoration's painter; onChanged (nil:
// null) is required when the decoration has an image, as Dart asserts.
func (d *BoxDecoration) CreateBoxPainter(onChanged func()) BoxPainter {
	if !(onChanged != nil || d.Image == nil) {
		panic(dart.NewAssertionError(nil))
	}
	return &boxDecorationPainter{BoxPainterBase: NewBoxPainterBase(onChanged), decoration: d}
}

// boxDecorationPainter is Dart's _BoxDecorationPainter.
type boxDecorationPainter struct {
	BoxPainterBase
	decoration *BoxDecoration

	cachedBackgroundPaint        *ui.Paint
	rectForCachedBackgroundPaint *ui.Rect
	imagePainter                 DecorationImagePainter
}

func (p *boxDecorationPainter) getBackgroundPaint(rect ui.Rect, textDirection *ui.TextDirection) *ui.Paint {
	if !(p.decoration.Gradient != nil || p.rectForCachedBackgroundPaint == nil) {
		panic(dart.NewAssertionError(nil))
	}
	if p.cachedBackgroundPaint == nil ||
		(p.decoration.Gradient != nil && *p.rectForCachedBackgroundPaint != rect) {
		paint := ui.NewPaint()
		if p.decoration.BackgroundBlendMode != nil {
			paint.SetBlendMode(*p.decoration.BackgroundBlendMode)
		}
		if p.decoration.Color != nil {
			paint.SetColor(*p.decoration.Color)
		}
		if p.decoration.Gradient != nil {
			paint.SetShader(p.decoration.Gradient.CreateShader(rect, textDirection))
			r := rect
			p.rectForCachedBackgroundPaint = &r
		}
		p.cachedBackgroundPaint = paint
	}
	return p.cachedBackgroundPaint
}

func (p *boxDecorationPainter) paintBox(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	switch p.decoration.Shape {
	case BoxShapeCircle:
		if p.decoration.BorderRadius != nil {
			panic(dart.NewAssertionError(circleWithBorderRadius))
		}
		center := rect.Center()
		radius := rect.ShortestSide() / 2.0
		canvas.DrawCircle(center, radius, paint)
	default:
		if p.decoration.BorderRadius == nil || p.decoration.BorderRadius == BorderRadiusGeometry(BorderRadiusZero) {
			canvas.DrawRect(rect, paint)
		} else {
			canvas.DrawRRect(p.decoration.BorderRadius.Resolve(textDirection).ToRRect(rect), paint)
		}
	}
}

func (p *boxDecorationPainter) paintShadows(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if p.decoration.BoxShadow == nil {
		return
	}
	for _, boxShadow := range p.decoration.BoxShadow {
		paint := boxShadow.ToPaint()
		bounds := rect.Shift(boxShadow.Offset).Inflate(boxShadow.SpreadRadius)
		debugOuter := foundation.KDebugMode && DebugDisableShadows && boxShadow.BlurStyle == ui.BlurStyleOuter
		if debugOuter {
			canvas.Save()
			canvas.ClipRect(bounds)
		}
		p.paintBox(canvas, bounds, paint, textDirection)
		if debugOuter {
			canvas.Restore()
		}
	}
}

func (p *boxDecorationPainter) paintBackgroundColor(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if p.decoration.Color != nil || p.decoration.Gradient != nil {
		// When border is filled, the rect is reduced to avoid anti-aliasing
		// rounding error leaking the background color around the clipped shape.
		adjustedRect := p.adjustedRectOnOutlinedBorder(rect, textDirection)
		p.paintBox(canvas, adjustedRect, p.getBackgroundPaint(rect, textDirection), textDirection)
	}
}

func calculateAdjustedSide(side BorderSide) float64 {
	if side.Color.Alpha() == 255 && side.Style == BorderStyleSolid {
		return side.StrokeInset()
	}
	return 0
}

func (p *boxDecorationPainter) adjustedRectOnOutlinedBorder(rect ui.Rect, textDirection *ui.TextDirection) ui.Rect {
	if p.decoration.Border == nil {
		return rect
	}
	if border, ok := p.decoration.Border.(Border); ok {
		insets := NewEdgeInsetsFromLTRB(
			calculateAdjustedSide(border.Left()),
			calculateAdjustedSide(border.Top()),
			calculateAdjustedSide(border.Right()),
			calculateAdjustedSide(border.Bottom()),
		).Div(2)
		return ui.NewRectFromLTRB(rect.Left+insets.Left, rect.Top+insets.Top, rect.Right-insets.Right,
			rect.Bottom-insets.Bottom)
	} else if border, ok := p.decoration.Border.(BorderDirectional); ok && textDirection != nil {
		leftSide, rightSide := border.Start(), border.End()
		if *textDirection == ui.TextDirectionRtl {
			leftSide, rightSide = border.End(), border.Start()
		}
		insets := NewEdgeInsetsFromLTRB(
			calculateAdjustedSide(leftSide),
			calculateAdjustedSide(border.Top()),
			calculateAdjustedSide(rightSide),
			calculateAdjustedSide(border.Bottom()),
		).Div(2)
		return ui.NewRectFromLTRB(rect.Left+insets.Left, rect.Top+insets.Top, rect.Right-insets.Right,
			rect.Bottom-insets.Bottom)
	}
	return rect
}

func (p *boxDecorationPainter) paintBackgroundImage(canvas ui.Canvas, rect ui.Rect, configuration ImageConfiguration) {
	if p.decoration.Image == nil {
		return
	}
	if p.imagePainter == nil {
		onChanged := p.OnChanged()
		if onChanged == nil {
			panic(dart.TypeError{Message: "Null check operator used on a null value"})
		}
		p.imagePainter = p.decoration.Image.CreatePainter(onChanged)
	}
	var clipPath *ui.Path
	switch p.decoration.Shape {
	case BoxShapeCircle:
		if p.decoration.BorderRadius != nil {
			panic(dart.NewAssertionError(circleWithBorderRadius))
		}
		center := rect.Center()
		radius := rect.ShortestSide() / 2.0
		square := ui.NewRectFromCircle(center, radius)
		clipPath = ui.NewPath()
		clipPath.AddOval(square)
	default:
		if p.decoration.BorderRadius != nil {
			clipPath = ui.NewPath()
			clipPath.AddRRect(p.decoration.BorderRadius.Resolve(configuration.TextDirection).ToRRect(rect))
		}
	}
	p.imagePainter.Paint(canvas, rect, clipPath, configuration)
}

// Dispose disposes the image painter.
func (p *boxDecorationPainter) Dispose() {
	if p.imagePainter != nil {
		p.imagePainter.Dispose()
	}
	p.BoxPainterBase.Dispose()
}

// Paint paints the decoration's shadows, background, image and border at
// offset & configuration.Size.
func (p *boxDecorationPainter) Paint(canvas ui.Canvas, offset ui.Offset, configuration ImageConfiguration) {
	if configuration.Size == nil {
		panic(dart.NewAssertionError(nil))
	}
	rect := offset.And(*configuration.Size)
	textDirection := configuration.TextDirection
	p.paintShadows(canvas, rect, textDirection)
	p.paintBackgroundColor(canvas, rect, textDirection)
	p.paintBackgroundImage(canvas, rect, configuration)
	if p.decoration.Border != nil {
		var borderRadius *BorderRadius
		if p.decoration.BorderRadius != nil {
			r := p.decoration.BorderRadius.Resolve(textDirection)
			borderRadius = &r
		}
		p.decoration.Border.PaintBox(canvas, rect, BoxBorderPaintOptions{
			Shape: p.decoration.Shape, BorderRadius: borderRadius, TextDirection: configuration.TextDirection})
	}
}

func (p *boxDecorationPainter) String() string { return "BoxPainter for " + p.decoration.String() }
