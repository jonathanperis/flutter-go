package painting

// Port of flutter_logo.dart (Flutter 3.50.1): FlutterLogoStyle and
// FlutterLogoDecoration.
//
// PARTIAL: the "Flutter" label of the horizontal and stacked styles needs
// TextPainter (text_painter.dart, not ported yet, nor is dart:ui's
// Paragraph). The painter draws everything else exactly as Dart does (the
// mark, its position and size for every style and transition, the opacity
// layer) and leaves out the label's drawing (its clip, transform, text and
// fade).

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// FlutterLogoStyle is the possible ways to draw Flutter's logo.
type FlutterLogoStyle int

const (
	// FlutterLogoStyleMarkOnly shows only Flutter's logo, not the "Flutter"
	// label.
	FlutterLogoStyleMarkOnly FlutterLogoStyle = iota
	// FlutterLogoStyleHorizontal shows Flutter's logo on the left, and the
	// "Flutter" label to its right.
	FlutterLogoStyleHorizontal
	// FlutterLogoStyleStacked shows Flutter's logo above the "Flutter"
	// label.
	FlutterLogoStyleStacked
)

// FlutterLogoStyleValues is FlutterLogoStyle.values.
var FlutterLogoStyleValues = []FlutterLogoStyle{FlutterLogoStyleMarkOnly, FlutterLogoStyleHorizontal,
	FlutterLogoStyleStacked}

// Name is the Dart enum value's name.
func (s FlutterLogoStyle) Name() string { return [...]string{"markOnly", "horizontal", "stacked"}[s] }

func (s FlutterLogoStyle) String() string { return "FlutterLogoStyle." + s.Name() }

// FlutterLogoDecoration is an immutable description of how to paint
// Flutter's logo. Used by pointer.
type FlutterLogoDecoration struct {
	DecorationBase
	// TextColor is the color used to paint the "Flutter" text on the logo.
	TextColor ui.Color
	// Style is whether and where to draw the "Flutter" text.
	Style FlutterLogoStyle
	// Margin is how far to inset the logo from the edge of the container.
	Margin EdgeInsets

	position float64 // -1.0 for stacked, 1.0 for horizontal, 0.0 for no logo
	opacity  float64 // 0.0 .. 1.0
}

// FlutterLogoDecorationOptions are FlutterLogoDecoration's named
// parameters (nil TextColor: Color(0xFF757575); zero Style: markOnly; zero
// Margin: EdgeInsets.zero).
type FlutterLogoDecorationOptions struct {
	TextColor *ui.Color
	Style     FlutterLogoStyle
	Margin    EdgeInsets
}

// NewFlutterLogoDecoration is FlutterLogoDecoration({...}).
func NewFlutterLogoDecoration(opts ...FlutterLogoDecorationOptions) *FlutterLogoDecoration {
	var o FlutterLogoDecorationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	textColor := ui.NewColor(0xFF757575)
	if o.TextColor != nil {
		textColor = *o.TextColor
	}
	position := -1.0
	switch o.Style {
	case FlutterLogoStyleMarkOnly:
		position = 0.0
	case FlutterLogoStyleHorizontal:
		position = 1.0
	}
	return newFlutterLogoDecoration(textColor, o.Style, o.Margin, position, 1.0)
}

func newFlutterLogoDecoration(textColor ui.Color, style FlutterLogoStyle, margin EdgeInsets, position, opacity float64) *FlutterLogoDecoration {
	d := &FlutterLogoDecoration{TextColor: textColor, Style: style, Margin: margin, position: position, opacity: opacity}
	d.SetSelf(d)
	return d
}

func (d *FlutterLogoDecoration) inTransition() bool {
	return d.opacity != 1.0 || (d.position != -1.0 && d.position != 0.0 && d.position != 1.0)
}

// DebugAssertIsValid asserts that the position is finite and the opacity
// in 0..1.
func (d *FlutterLogoDecoration) DebugAssertIsValid() bool {
	if !(!math.IsInf(d.position, 0) && !math.IsNaN(d.position) && d.opacity >= 0.0 && d.opacity <= 1.0) {
		panic(dart.NewAssertionError(nil))
	}
	return true
}

// IsComplex is true unless the decoration is in a transition.
func (d *FlutterLogoDecoration) IsComplex() bool { return !d.inTransition() }

// FlutterLogoDecorationLerp is FlutterLogoDecoration.lerp (nil is null).
func FlutterLogoDecorationLerp(a, b *FlutterLogoDecoration, t float64) *FlutterLogoDecoration {
	if a != nil {
		a.DebugAssertIsValid()
	}
	if b != nil {
		b.DebugAssertIsValid()
	}
	if a == b {
		return a
	}
	if a == nil {
		return newFlutterLogoDecoration(b.TextColor, b.Style, b.Margin.Mul(t), b.position,
			b.opacity*ui.ClampDouble(t, 0.0, 1.0))
	}
	if b == nil {
		return newFlutterLogoDecoration(a.TextColor, a.Style, a.Margin.Mul(t), a.position,
			a.opacity*ui.ClampDouble(1.0-t, 0.0, 1.0))
	}
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	style := b.Style
	if t < 0.5 {
		style = a.Style
	}
	return newFlutterLogoDecoration(*ui.ColorLerp(&a.TextColor, &b.TextColor, t), style,
		*EdgeInsetsLerp(&a.Margin, &b.Margin, t), a.position+float64((b.position-a.position)*t),
		ui.ClampDouble(a.opacity+float64((b.opacity-a.opacity)*t), 0.0, 1.0))
}

func nilIfNilLogo(d *FlutterLogoDecoration) Decoration {
	if d == nil {
		return nil
	}
	return d
}

// LerpFrom lerps from a when it is a FlutterLogoDecoration or null.
func (d *FlutterLogoDecoration) LerpFrom(a Decoration, t float64) Decoration {
	d.DebugAssertIsValid()
	if isNilValue(a) {
		return nilIfNilLogo(FlutterLogoDecorationLerp(nil, d, t))
	}
	if la, ok := a.(*FlutterLogoDecoration); ok {
		la.DebugAssertIsValid()
		return nilIfNilLogo(FlutterLogoDecorationLerp(la, d, t))
	}
	return d.DecorationBase.LerpFrom(a, t)
}

// LerpTo lerps to b when it is a FlutterLogoDecoration or null.
func (d *FlutterLogoDecoration) LerpTo(b Decoration, t float64) Decoration {
	d.DebugAssertIsValid()
	if isNilValue(b) {
		return nilIfNilLogo(FlutterLogoDecorationLerp(d, nil, t))
	}
	if lb, ok := b.(*FlutterLogoDecoration); ok {
		lb.DebugAssertIsValid()
		return nilIfNilLogo(FlutterLogoDecorationLerp(d, lb, t))
	}
	return d.DecorationBase.LerpTo(b, t)
}

// HitTest returns true.
func (d *FlutterLogoDecoration) HitTest(size ui.Size, position ui.Offset, textDirection *ui.TextDirection) bool {
	return true
}

// CreateBoxPainter returns the logo's painter.
func (d *FlutterLogoDecoration) CreateBoxPainter(onChanged func()) BoxPainter {
	d.DebugAssertIsValid()
	return newFlutterLogoPainter(d)
}

// GetClipPath is the rect.
func (d *FlutterLogoDecoration) GetClipPath(rect ui.Rect, textDirection ui.TextDirection) *ui.Path {
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}

// Equal is ==: same text color, position and opacity (style and margin are
// not compared, as in Dart).
func (d *FlutterLogoDecoration) Equal(other Decoration) bool {
	d.DebugAssertIsValid()
	o, ok := other.(*FlutterLogoDecoration)
	if !ok || o == nil {
		return false
	}
	return d == o || (o.TextColor == d.TextColor && o.position == d.position && o.opacity == d.opacity)
}

// DebugFillProperties adds textColor, style and the transition state.
func (d *FlutterLogoDecoration) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DecorationBase.DebugFillProperties(properties)
	properties.Add(NewColorProperty("textColor", d.TextColor))
	properties.Add(foundation.NewEnumProperty("style", d.Style))
	if d.inTransition() {
		properties.Add(foundation.NewDiagnosticsNodeMessage("transition " + formatDouble(d.position) + ":" +
			formatDouble(d.opacity)))
	}
}

type flutterLogoPainter struct {
	BoxPainterBase
	config *FlutterLogoDecoration
}

func newFlutterLogoPainter(config *FlutterLogoDecoration) *flutterLogoPainter {
	config.DebugAssertIsValid()
	return &flutterLogoPainter{config: config}
}

func pathOf(points ...float64) *ui.Path {
	p := ui.NewPath()
	p.MoveTo(points[0], points[1])
	for i := 2; i < len(points); i += 2 {
		p.LineTo(points[i], points[i+1])
	}
	return p
}

func colorPaint(color int) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(color))
	return p
}

// paintLogo is _paintLogo. Its magic numbers were derived from the values
// in the SVG files exported from the original artwork source.
func (p *flutterLogoPainter) paintLogo(canvas ui.Canvas, rect ui.Rect) {
	// Our points are in a coordinate space that's 166 pixels wide and 202
	// pixels high. First, transform the rectangle so that our coordinate
	// space is a square 202 pixels to a side, with the top left at the
	// origin.
	canvas.Save()
	canvas.Translate(rect.Left, rect.Top)
	canvas.Scale(rect.Width()/202.0, rect.Height()/202.0)
	// Next, offset it some more so that the 166 horizontal pixels are
	// centered in that square.
	canvas.Translate((202.0-166.0)/2.0, 0.0)

	// Set up the styles.
	lightPaint := colorPaint(0xFF54C5F8)
	mediumPaint := colorPaint(0xFF29B6F6)
	darkPaint := colorPaint(0xFF01579B)
	// Dart's const offsets fold each sum in double arithmetic.
	x0, y0, x1, y1 := 87.2623, 28.8384, 42.9205, 35.0952
	dx, dy := 37.9092, 123.4389
	triangleGradient := ui.NewGradientLinear(ui.NewOffset(x0+dx, y0+dy), ui.NewOffset(x1+dx, y1+dy),
		[]ui.Color{ui.NewColor(0x001A237E), ui.NewColor(0x661A237E)})
	trianglePaint := ui.NewPaint()
	trianglePaint.SetShader(triangleGradient)

	// Draw the basic shape.
	canvas.DrawPath(pathOf(37.7, 128.9, 9.8, 101.0, 100.4, 10.4, 156.2, 10.4), lightPaint)
	canvas.DrawPath(pathOf(156.2, 94.0, 100.4, 94.0, 78.5, 115.9, 106.4, 143.8), lightPaint)
	canvas.DrawPath(pathOf(79.5, 170.7, 100.4, 191.6, 156.2, 191.6, 107.4, 142.8), darkPaint)

	// The overlap between middle and bottom beam.
	canvas.Save()
	canvas.Transform([]float64{
		// careful, this is in _column_-major order
		0.7071, -0.7071, 0.0, 0.0,
		0.7071, 0.7071, 0.0, 0.0,
		0.0, 0.0, 1.0, 0.0,
		-77.697, 98.057, 0.0, 1.0,
	})
	canvas.DrawRect(ui.NewRectFromLTWH(59.8, 123.1, 39.4, 39.4), mediumPaint)
	canvas.Restore()

	// The gradients below the middle beam on top of the bottom beam.
	canvas.DrawPath(pathOf(79.5, 170.7, 120.9, 156.4, 107.4, 142.8), trianglePaint)

	canvas.Restore()
}

func (p *flutterLogoPainter) Paint(canvas ui.Canvas, offset ui.Offset, configuration ImageConfiguration) {
	offset = offset.Add(p.config.Margin.TopLeft())
	canvasSize := p.config.Margin.DeflateSize(*configuration.Size)
	if canvasSize.IsEmpty() {
		return
	}
	var logoSize ui.Size
	switch {
	case p.config.position > 0.0:
		logoSize = ui.NewSize(820.0, 232.0) // horizontal style
	case p.config.position < 0.0:
		logoSize = ui.NewSize(252.0, 306.0) // stacked style
	default:
		logoSize = ui.NewSize(202.0, 202.0) // only the mark
	}
	fittedSize := ApplyBoxFit(BoxFitContain, logoSize, canvasSize)
	if fittedSize.Source != logoSize {
		panic(dart.NewAssertionError(nil))
	}
	rect := AlignmentCenter.Inscribe(fittedSize.Destination, offset.And(canvasSize))
	centerSquareHeight := canvasSize.ShortestSide()
	centerSquare := ui.NewRectFromLTWH(offset.Dx+(canvasSize.Width-centerSquareHeight)/2.0,
		offset.Dy+(canvasSize.Height-centerSquareHeight)/2.0, centerSquareHeight, centerSquareHeight)
	var logoTargetSquare ui.Rect
	switch {
	case p.config.position > 0.0:
		// horizontal style
		logoTargetSquare = ui.NewRectFromLTWH(rect.Left, rect.Top, rect.Height(), rect.Height())
	case p.config.position < 0.0:
		// stacked style
		logoHeight := rect.Height() * 191.0 / 306.0
		logoTargetSquare = ui.NewRectFromLTWH(rect.Left+(rect.Width()-logoHeight)/2.0, rect.Top, logoHeight, logoHeight)
	default:
		// only the mark
		logoTargetSquare = centerSquare
	}
	logoSquare := *ui.RectLerp(&centerSquare, &logoTargetSquare, math.Abs(p.config.position))

	if p.config.opacity < 1.0 {
		layerPaint := ui.NewPaint()
		alpha := p.config.opacity
		layerPaint.SetColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFFFFFF).WithValues(ui.ColorWithValuesOptions{Alpha: &alpha}),
			ui.BlendModeModulate))
		bounds := offset.And(canvasSize)
		canvas.SaveLayer(&bounds, layerPaint)
	}
	// PARTIAL: the "Flutter" label of the horizontal (position > 0) and
	// stacked (position < 0) styles is not drawn (no TextPainter yet).
	p.paintLogo(canvas, logoSquare)
	if p.config.opacity < 1.0 {
		canvas.Restore()
	}
}
