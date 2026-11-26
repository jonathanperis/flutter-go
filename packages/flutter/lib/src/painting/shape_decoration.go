package painting

// Port of shape_decoration.dart (Flutter 3.50.1): ShapeDecoration and its
// painter.
//
// ShapeDecoration is used by pointer and built with
// NewShapeDecoration(ShapeDecorationOptions) (Shape is required; the
// constructor asserts that color and gradient are not both given).
// Identity in lerp is pointer identity, as for BoxDecoration.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ShapeDecoration is an immutable description of how to paint an
// arbitrary shape: a ShapeBorder filled with a color, gradient or image,
// casting shadows.
type ShapeDecoration struct {
	DecorationBase
	// Color to fill in the background of the shape (nil: none).
	Color *ui.Color
	// Gradient to use when filling the shape (nil: none).
	Gradient Gradient
	// Image to paint inside the shape (nil: none).
	Image DecorationImage
	// Shadows cast by the shape (nil: none).
	Shadows []BoxShadow
	// Shape to fill the color, gradient and image into and to cast as the
	// shadow.
	Shape ShapeBorder
}

// ShapeDecorationOptions are ShapeDecoration's named parameters (Shape is
// required).
type ShapeDecorationOptions struct {
	Color    *ui.Color
	Image    DecorationImage
	Gradient Gradient
	Shadows  []BoxShadow
	Shape    ShapeBorder
}

// NewShapeDecoration is ShapeDecoration({...}); it panics when both a
// color and a gradient are given, as Dart asserts.
func NewShapeDecoration(o ShapeDecorationOptions) *ShapeDecoration {
	if isNilGradient(o.Gradient) {
		o.Gradient = nil
	}
	if isNilValue(o.Image) {
		o.Image = nil
	}
	if isNilValue(o.Shape) {
		panic(dart.TypeError{Message: "ShapeDecoration: shape is required"})
	}
	if o.Color != nil && o.Gradient != nil {
		panic(dart.NewAssertionError(nil))
	}
	d := &ShapeDecoration{Color: o.Color, Gradient: o.Gradient, Image: o.Image, Shadows: o.Shadows, Shape: o.Shape}
	d.SetSelf(d)
	return d
}

// NewShapeDecorationFromBoxDecoration is ShapeDecoration.fromBoxDecoration:
// the ShapeDecoration equivalent of a BoxDecoration (a circle becomes a
// CircleBorder, a rounded rectangle a RoundedRectangleBorder, otherwise
// the box's border or Border()).
func NewShapeDecorationFromBoxDecoration(source *BoxDecoration) *ShapeDecoration {
	var shape ShapeBorder
	switch source.Shape {
	case BoxShapeCircle:
		if source.Border != nil {
			if !source.Border.IsUniform() {
				panic(dart.NewAssertionError(nil))
			}
			side := source.Border.Top()
			shape = NewCircleBorder(CircleBorderOptions{Side: &side})
		} else {
			shape = NewCircleBorder()
		}
	default:
		if source.BorderRadius != nil {
			if !(source.Border == nil || source.Border.IsUniform()) {
				panic(dart.NewAssertionError(nil))
			}
			side := BorderSideNone
			if source.Border != nil {
				side = source.Border.Top()
			}
			shape = NewRoundedRectangleBorder(RRectBorderOptions{Side: &side, BorderRadius: source.BorderRadius})
		} else if source.Border != nil {
			shape = source.Border
		} else {
			shape = NewBorder()
		}
	}
	return NewShapeDecoration(ShapeDecorationOptions{Color: source.Color, Image: source.Image,
		Gradient: source.Gradient, Shadows: source.BoxShadow, Shape: shape})
}

// GetClipPath is the shape's outer path.
func (d *ShapeDecoration) GetClipPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	return d.Shape.GetOuterPath(rect, &textDirection)
}

// Padding is the shape's dimensions.
func (d *ShapeDecoration) Padding() EdgeInsetsGeometry { return d.Shape.Dimensions() }

// IsComplex is true when the decoration has shadows.
func (d *ShapeDecoration) IsComplex() bool { return d.Shadows != nil }

func nilIfNilShape(d *ShapeDecoration) Decoration {
	if d == nil {
		return nil
	}
	return d
}

// LerpFrom lerps from a BoxDecoration (converted with
// fromBoxDecoration), a ShapeDecoration or null.
func (d *ShapeDecoration) LerpFrom(a Decoration, t float64) Decoration {
	if isNilValue(a) {
		return nilIfNilShape(ShapeDecorationLerp(nil, d, t))
	}
	switch a := a.(type) {
	case *BoxDecoration:
		return nilIfNilShape(ShapeDecorationLerp(NewShapeDecorationFromBoxDecoration(a), d, t))
	case *ShapeDecoration:
		return nilIfNilShape(ShapeDecorationLerp(a, d, t))
	}
	return d.DecorationBase.LerpFrom(a, t)
}

// LerpTo lerps to a BoxDecoration (converted with fromBoxDecoration), a
// ShapeDecoration or null.
func (d *ShapeDecoration) LerpTo(b Decoration, t float64) Decoration {
	if isNilValue(b) {
		return nilIfNilShape(ShapeDecorationLerp(d, nil, t))
	}
	switch b := b.(type) {
	case *BoxDecoration:
		return nilIfNilShape(ShapeDecorationLerp(d, NewShapeDecorationFromBoxDecoration(b), t))
	case *ShapeDecoration:
		return nilIfNilShape(ShapeDecorationLerp(d, b, t))
	}
	return d.DecorationBase.LerpTo(b, t)
}

// ShapeDecorationLerp is ShapeDecoration.lerp: linearly interpolates
// between two shape decorations (nil is null). A color lerped with a
// gradient becomes a uniform-color gradient of the other side's type.
func ShapeDecorationLerp(a, b *ShapeDecoration, t float64) *ShapeDecoration {
	if a == b {
		return a
	}
	if a != nil && b != nil {
		if t == 0.0 {
			return a
		}
		if t == 1.0 {
			return b
		}
	}
	var aGradient, bGradient Gradient
	var aColor, bColor *ui.Color
	var aImage, bImage DecorationImage
	var aShadows, bShadows []BoxShadow
	var aShape, bShape ShapeBorder
	if a != nil {
		aGradient, aColor, aImage, aShadows, aShape = a.Gradient, a.Color, a.Image, a.Shadows, a.Shape
	}
	if b != nil {
		bGradient, bColor, bImage, bShadows, bShape = b.Gradient, b.Color, b.Image, b.Shadows, b.Shape
	}
	if aGradient == nil && bGradient != nil && aColor != nil {
		aGradient = bGradient.FromColor(*aColor)
	} else if bGradient == nil && aGradient != nil && bColor != nil {
		bGradient = aGradient.FromColor(*bColor)
	}
	gradient := GradientLerp(aGradient, bGradient, t)
	var color *ui.Color
	if isNilGradient(gradient) {
		gradient = nil
		color = ui.ColorLerp(aColor, bColor, t)
	}
	shape := ShapeBorderLerp(aShape, bShape, t)
	if shape == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	return NewShapeDecoration(ShapeDecorationOptions{
		Color:    color,
		Gradient: gradient,
		Image:    DecorationImageLerp(aImage, bImage, t),
		Shadows:  BoxShadowLerpList(aShadows, bShadows, t),
		Shape:    shape,
	})
}

// Equal is ==: the same runtime type and equal fields.
func (d *ShapeDecoration) Equal(other Decoration) bool {
	o, ok := other.(*ShapeDecoration)
	if !ok || o == nil {
		return false
	}
	if d == o {
		return true
	}
	return ptrEq(o.Color, d.Color) &&
		GradientsEqual(o.Gradient, d.Gradient) &&
		decorationImagesEqual(o.Image, d.Image) &&
		foundation.ListEquals(o.Shadows, d.Shadows) &&
		ShapeBorderEqual(o.Shape, d.Shape)
}

// DebugFillProperties adds the decoration's fields (whitespace style).
func (d *ShapeDecoration) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DecorationBase.DebugFillProperties(properties)
	properties.DefaultDiagnosticsTreeStyle = foundation.DiagnosticsTreeStyleWhitespace
	properties.Add(NewColorProperty("color", d.Color, ColorPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDiagnosticsProperty("gradient", d.Gradient,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDiagnosticsProperty("image", d.Image,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	whitespace := foundation.DiagnosticsTreeStyleWhitespace
	properties.Add(foundation.NewIterableProperty("shadows", d.Shadows,
		foundation.IterablePropertyOptions{DefaultValue: foundation.Null, Style: &whitespace}))
	properties.Add(foundation.NewDiagnosticsProperty("shape", d.Shape))
}

// HitTest delegates to the shape's HitTest over Offset.zero & size.
func (d *ShapeDecoration) HitTest(size ui.Size, position ui.Offset, textDirection *ui.TextDirection) bool {
	return d.Shape.HitTest(ui.OffsetZero.And(size), position, textDirection)
}

// CreateBoxPainter returns the decoration's painter. onChanged is required
// (Dart asserts it is given when there is an image, then applies `!`,
// which throws for a null callback in every case).
func (d *ShapeDecoration) CreateBoxPainter(onChanged func()) BoxPainter {
	if !(onChanged != nil || d.Image == nil) {
		panic(dart.NewAssertionError(nil))
	}
	if onChanged == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	return &shapeDecorationPainter{BoxPainterBase: NewBoxPainterBase(onChanged), decoration: d}
}

// shapeDecorationPainter is Dart's _ShapeDecorationPainter.
type shapeDecorationPainter struct {
	BoxPainterBase
	decoration *ShapeDecoration

	lastRect          *ui.Rect
	lastTextDirection *ui.TextDirection
	outerPath         *ui.Path
	innerPath         *ui.Path
	interiorPaint     *ui.Paint
	shadowCount       *int
	shadowBounds      []ui.Rect
	shadowPaths       []*ui.Path
	shadowPaints      []*ui.Paint
	imagePainter      DecorationImagePainter
}

func (p *shapeDecorationPainter) precache(rect ui.Rect, textDirection *ui.TextDirection) {
	if p.lastRect != nil && rect == *p.lastRect && ptrEq(textDirection, p.lastTextDirection) {
		return
	}
	d := p.decoration
	// We reach here in two cases:
	//  - the very first time we paint, in which case everything except
	//    the decoration is null
	//  - subsequent times, if the rect has changed, in which case we only
	//    need to update the features that depend on the actual rect.
	if p.interiorPaint == nil && (d.Color != nil || d.Gradient != nil) {
		p.interiorPaint = ui.NewPaint()
		if d.Color != nil {
			p.interiorPaint.SetColor(*d.Color)
		}
	}
	if d.Gradient != nil {
		p.interiorPaint.SetShader(d.Gradient.CreateShader(rect, textDirection))
	}
	if d.Shadows != nil {
		if p.shadowCount == nil {
			n := len(d.Shadows)
			p.shadowCount = &n
			p.shadowPaints = make([]*ui.Paint, 0, n)
			for _, shadow := range d.Shadows {
				p.shadowPaints = append(p.shadowPaints, shadow.ToPaint())
			}
		}
		if d.Shape.PreferPaintInterior() {
			p.shadowBounds = make([]ui.Rect, 0, len(d.Shadows))
			for _, shadow := range d.Shadows {
				p.shadowBounds = append(p.shadowBounds, rect.Shift(shadow.Offset).Inflate(shadow.SpreadRadius))
			}
		} else {
			p.shadowPaths = make([]*ui.Path, 0, len(d.Shadows))
			for _, shadow := range d.Shadows {
				p.shadowPaths = append(p.shadowPaths,
					d.Shape.GetOuterPath(rect.Shift(shadow.Offset).Inflate(shadow.SpreadRadius), textDirection))
			}
		}
	}
	if !d.Shape.PreferPaintInterior() && (p.interiorPaint != nil || p.shadowCount != nil) {
		p.outerPath = d.Shape.GetOuterPath(rect, textDirection)
	}
	if d.Image != nil {
		p.innerPath = d.Shape.GetInnerPath(rect, textDirection)
	}
	r := rect
	p.lastRect = &r
	p.lastTextDirection = textDirection
}

// debugHandleDisabledShadowStart clips out the shape so that an outer
// shadow painted without its blur (debugDisableShadows) stays outside.
func debugHandleDisabledShadowStart(canvas ui.Canvas, boxShadow BoxShadow, path *ui.Path) {
	if DebugDisableShadows && boxShadow.BlurStyle == ui.BlurStyleOuter {
		canvas.Save()
		clipPath := ui.NewPath()
		clipPath.SetFillType(ui.PathFillTypeEvenOdd)
		clipPath.AddRect(ui.RectLargest)
		clipPath.AddPath(path, ui.OffsetZero)
		canvas.ClipPath(clipPath)
	}
}

func debugHandleDisabledShadowEnd(canvas ui.Canvas, boxShadow BoxShadow) {
	if DebugDisableShadows && boxShadow.BlurStyle == ui.BlurStyleOuter {
		canvas.Restore()
	}
}

func (p *shapeDecorationPainter) paintShadows(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	// The debugHandleDisabledShadowStart/End calls run in debug mode only
	// (Dart's asserts) to support BlurStyle.outer when debugDisableShadows
	// is set.
	if p.shadowCount == nil {
		return
	}
	d := p.decoration
	if d.Shape.PreferPaintInterior() {
		for index := 0; index < *p.shadowCount; index += 1 {
			if foundation.KDebugMode {
				debugHandleDisabledShadowStart(canvas, d.Shadows[index],
					d.Shape.GetOuterPath(p.shadowBounds[index], textDirection))
			}
			d.Shape.PaintInterior(canvas, p.shadowBounds[index], p.shadowPaints[index], textDirection)
			if foundation.KDebugMode {
				debugHandleDisabledShadowEnd(canvas, d.Shadows[index])
			}
		}
	} else {
		for index := 0; index < *p.shadowCount; index += 1 {
			if foundation.KDebugMode {
				debugHandleDisabledShadowStart(canvas, d.Shadows[index], p.shadowPaths[index])
			}
			canvas.DrawPath(p.shadowPaths[index], p.shadowPaints[index])
			if foundation.KDebugMode {
				debugHandleDisabledShadowEnd(canvas, d.Shadows[index])
			}
		}
	}
}

func (p *shapeDecorationPainter) paintInterior(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if p.interiorPaint != nil {
		if p.decoration.Shape.PreferPaintInterior() {
			// When border is filled, the rect is reduced to avoid anti-aliasing
			// rounding error leaking the background color around the clipped shape.
			adjustedRect := p.adjustedRectOnOutlinedBorder(rect)
			p.decoration.Shape.PaintInterior(canvas, adjustedRect, p.interiorPaint, textDirection)
		} else {
			canvas.DrawPath(p.outerPath, p.interiorPaint)
		}
	}
}

func (p *shapeDecorationPainter) adjustedRectOnOutlinedBorder(rect ui.Rect) ui.Rect {
	if outlined, ok := p.decoration.Shape.(OutlinedBorder); ok && p.decoration.Color != nil {
		side := outlined.GetSide()
		if side.Color.Alpha() == 255 && side.Style == BorderStyleSolid {
			return rect.Deflate(side.StrokeInset() / 2)
		}
	}
	return rect
}

func (p *shapeDecorationPainter) paintImage(canvas ui.Canvas, configuration ImageConfiguration) {
	if p.decoration.Image == nil {
		return
	}
	if p.imagePainter == nil {
		p.imagePainter = p.decoration.Image.CreatePainter(p.OnChanged())
	}
	p.imagePainter.Paint(canvas, *p.lastRect, p.innerPath, configuration)
}

// Dispose disposes the image painter.
func (p *shapeDecorationPainter) Dispose() {
	if p.imagePainter != nil {
		p.imagePainter.Dispose()
	}
	p.BoxPainterBase.Dispose()
}

// Paint paints the decoration's shadows, interior, image and shape at
// offset & configuration.Size.
func (p *shapeDecorationPainter) Paint(canvas ui.Canvas, offset ui.Offset, configuration ImageConfiguration) {
	if configuration.Size == nil {
		panic(dart.NewAssertionError(nil))
	}
	rect := offset.And(*configuration.Size)
	textDirection := configuration.TextDirection
	p.precache(rect, textDirection)
	p.paintShadows(canvas, rect, textDirection)
	p.paintInterior(canvas, rect, textDirection)
	p.paintImage(canvas, configuration)
	p.decoration.Shape.Paint(canvas, rect, textDirection)
}
