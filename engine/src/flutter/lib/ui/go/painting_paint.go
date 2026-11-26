package ui

// Port of painting.dart's Paint (Flutter 3.50.1), and the engine's
// lib/ui/painting/paint.cc, which turns it into a DlPaint (here an
// ImpellerPaint).

import (
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// Paint describes the style to use when drawing on a Canvas.
//
// Dart's Paint keeps its scalar state in a 68-byte ByteData of 32-bit
// slots shared with the engine (encoded so that all-zero bytes are the
// defaults); the fields below are those slots, so every value read back
// went through the same float32 storage and encodings (alpha stored as
// 1 - a, blend mode XOR srcOver, miter limit minus 4).
type Paint struct {
	isAntiAlias        int32 // 0: true
	red, green, blue   float32
	alpha              float32 // 1 - a
	colorSpace         int32
	blendMode          int32 // index ^ srcOver
	style              int32
	strokeWidth        float32
	strokeCap          int32
	strokeJoin         int32
	strokeMiterLimit   float32 // value - 4
	filterQuality      int32
	maskFilter         int32 // 0: none, 1: blur
	maskFilterStyle    int32
	maskFilterSigma    float32
	invertColor        int32
	objectsInitialized bool // Dart's _objects != null
	shader             Shader
	colorFilter        *ColorFilter
	imageFilter        ImageFilter
}

const (
	kColorDefault            = 0xFF000000
	kStrokeMiterLimitDefault = 4.0
	maskFilterTypeNone       = 0
	maskFilterTypeBlur       = 1
	kBlendModeDefault        = int32(BlendModeSrcOver)
)

// NewPaint is Paint(): a paint with every value at its default.
func NewPaint() *Paint { return &Paint{} }

// NewPaintFrom is Paint.from: a copy of other.
func NewPaintFrom(other *Paint) *Paint {
	p := *other
	return &p
}

// IsAntiAlias is whether shapes are anti-aliased (default true).
func (p *Paint) IsAntiAlias() bool { return p.isAntiAlias == 0 }

// SetIsAntiAlias sets IsAntiAlias.
func (p *Paint) SetIsAntiAlias(v bool) {
	if v {
		p.isAntiAlias = 0
	} else {
		p.isAntiAlias = 1
	}
}

// Color is the color used to stroke or fill (default opaque black).
func (p *Paint) Color() Color {
	return NewColorFrom(ColorFromOptions{Alpha: 1.0 - float64(p.alpha), Red: float64(p.red),
		Green: float64(p.green), Blue: float64(p.blue), ColorSpace: indexToColorSpace(int(p.colorSpace))})
}

// SetColor sets Color.
func (p *Paint) SetColor(c Color) {
	p.red, p.green, p.blue = float32(c.R), float32(c.G), float32(c.B)
	p.alpha = float32(1.0 - c.A)
	p.colorSpace = int32(c.ColorSpace)
}

func indexToColorSpace(i int) ColorSpace {
	switch i {
	case 0:
		return ColorSpaceSRGB
	case 1:
		return ColorSpaceExtendedSRGB
	case 2:
		return ColorSpaceDisplayP3
	}
	panic(argumentError("Unknown color space: " + strconv.Itoa(i)))
}

// BlendMode is the blend mode used to composite drawings (default srcOver).
func (p *Paint) BlendMode() BlendMode { return BlendMode(p.blendMode ^ kBlendModeDefault) }

// SetBlendMode sets BlendMode.
func (p *Paint) SetBlendMode(m BlendMode) { p.blendMode = int32(m) ^ kBlendModeDefault }

// Style is whether to fill or stroke shapes (default fill).
func (p *Paint) Style() PaintingStyle { return PaintingStyle(p.style) }

// SetStyle sets Style.
func (p *Paint) SetStyle(s PaintingStyle) { p.style = int32(s) }

// StrokeWidth is the stroke width (default 0.0, a hairline).
func (p *Paint) StrokeWidth() float64 { return float64(p.strokeWidth) }

// SetStrokeWidth sets StrokeWidth (stored as a float32).
func (p *Paint) SetStrokeWidth(v float64) { p.strokeWidth = float32(v) }

// StrokeCap is the end of stroked lines (default butt).
func (p *Paint) StrokeCap() StrokeCap { return StrokeCap(p.strokeCap) }

// SetStrokeCap sets StrokeCap.
func (p *Paint) SetStrokeCap(c StrokeCap) { p.strokeCap = int32(c) }

// StrokeJoin is the join of stroked segments (default miter).
func (p *Paint) StrokeJoin() StrokeJoin { return StrokeJoin(p.strokeJoin) }

// SetStrokeJoin sets StrokeJoin.
func (p *Paint) SetStrokeJoin(j StrokeJoin) { p.strokeJoin = int32(j) }

// StrokeMiterLimit is the miter limit. As in Flutter 3.50.1, the getter
// returns the stored value, which is the limit minus 4 (so a new Paint
// reports 0.0); the engine adds 4 back when drawing.
func (p *Paint) StrokeMiterLimit() float64 { return float64(p.strokeMiterLimit) }

// SetStrokeMiterLimit sets the miter limit (stored as value - 4.0).
func (p *Paint) SetStrokeMiterLimit(v float64) {
	p.strokeMiterLimit = float32(v - kStrokeMiterLimitDefault)
}

// MaskFilter is the mask filter (e.g. a blur) applied to shapes; nil for
// none.
func (p *Paint) MaskFilter() *MaskFilter {
	switch p.maskFilter {
	case maskFilterTypeBlur:
		m := NewMaskFilterBlur(BlurStyle(p.maskFilterStyle), float64(p.maskFilterSigma))
		return &m
	}
	return nil
}

// SetMaskFilter sets MaskFilter (nil for none).
func (p *Paint) SetMaskFilter(m *MaskFilter) {
	if m == nil {
		p.maskFilter, p.maskFilterStyle, p.maskFilterSigma = maskFilterTypeNone, 0, 0
	} else {
		p.maskFilter, p.maskFilterStyle, p.maskFilterSigma = maskFilterTypeBlur, int32(m.style), float32(m.sigma)
	}
}

// FilterQuality is the quality of image sampling (default none).
func (p *Paint) FilterQuality() FilterQuality { return FilterQuality(p.filterQuality) }

// SetFilterQuality sets FilterQuality.
func (p *Paint) SetFilterQuality(q FilterQuality) { p.filterQuality = int32(q) }

// Shader is the shader used to fill or stroke (nil: Color is used).
func (p *Paint) Shader() Shader { return p.shader }

// SetShader sets Shader (nil for none).
func (p *Paint) SetShader(s Shader) {
	if debugMode {
		if s != nil && s.DebugDisposed() {
			panic(dart.NewAssertionError("Attempted to set a disposed shader to " + p.String()))
		}
		if fs, ok := s.(interface {
			validateSamplers() bool
			debugName() string
		}); ok && !fs.validateSamplers() {
			panic(dart.NewException("Invalid FragmentShader " + fs.debugName() + ": missing sampler"))
		}
	}
	p.objectsInitialized = true
	p.shader = s
}

// ColorFilter is the color filter applied when drawing (nil for none).
func (p *Paint) ColorFilter() *ColorFilter { return p.colorFilter }

// SetColorFilter sets ColorFilter (nil for none).
func (p *Paint) SetColorFilter(f *ColorFilter) {
	if f == nil {
		p.colorFilter = nil
		return
	}
	p.objectsInitialized = true
	p.colorFilter = f
}

// ImageFilter is the image filter applied when drawing (nil for none).
func (p *Paint) ImageFilter() ImageFilter { return p.imageFilter }

// SetImageFilter sets ImageFilter. As in Dart, setting a filter equal to
// the current one keeps the current one.
func (p *Paint) SetImageFilter(f ImageFilter) {
	if f == nil {
		p.imageFilter = nil
		return
	}
	p.objectsInitialized = true
	if p.imageFilter == nil || !p.imageFilter.equalImageFilter(f) {
		p.imageFilter = f
	}
}

// InvertColors is whether colors are inverted when drawing (default false).
func (p *Paint) InvertColors() bool { return p.invertColor == 1 }

// SetInvertColors sets InvertColors.
func (p *Paint) SetInvertColors(v bool) {
	if v {
		p.invertColor = 1
	} else {
		p.invertColor = 0
	}
}

func (p *Paint) String() string {
	var b strings.Builder
	semicolon := ""
	b.WriteString("Paint(")
	if p.Style() == PaintingStyleStroke {
		b.WriteString(p.Style().String())
		if p.StrokeWidth() != 0.0 {
			b.WriteString(" " + dart.ToStringAsFixed(p.StrokeWidth(), 1))
		} else {
			b.WriteString(" hairline")
		}
		if p.StrokeCap() != StrokeCapButt {
			b.WriteString(" " + p.StrokeCap().String())
		}
		if p.StrokeJoin() == StrokeJoinMiter {
			if p.StrokeMiterLimit() != kStrokeMiterLimitDefault {
				b.WriteString(" " + p.StrokeJoin().String() + " up to " + dart.ToStringAsFixed(p.StrokeMiterLimit(), 1))
			}
		} else {
			b.WriteString(" " + p.StrokeJoin().String())
		}
		semicolon = "; "
	}
	if !p.IsAntiAlias() {
		b.WriteString(semicolon + "antialias off")
		semicolon = "; "
	}
	if p.Color() != NewColor(kColorDefault) {
		b.WriteString(semicolon + p.Color().String())
		semicolon = "; "
	}
	if int32(p.BlendMode()) != kBlendModeDefault {
		b.WriteString(semicolon + p.BlendMode().String())
		semicolon = "; "
	}
	if p.ColorFilter() != nil {
		b.WriteString(semicolon + "colorFilter: " + p.ColorFilter().String())
		semicolon = "; "
	}
	if m := p.MaskFilter(); m != nil {
		b.WriteString(semicolon + "maskFilter: " + m.String())
		semicolon = "; "
	}
	if p.FilterQuality() != FilterQualityNone {
		b.WriteString(semicolon + "filterQuality: " + p.FilterQuality().String())
		semicolon = "; "
	}
	if p.Shader() != nil {
		b.WriteString(semicolon + "shader: " + shaderString(p.Shader()))
		semicolon = "; "
	}
	if p.ImageFilter() != nil {
		b.WriteString(semicolon + "imageFilter: " + p.ImageFilter().String())
		semicolon = "; "
	}
	if p.InvertColors() {
		b.WriteString(semicolon + "invert: true")
	}
	b.WriteString(")")
	return b.String()
}

// shaderString is Dart's '$shader': a shader's own toString when it has
// one, Object.toString's "Instance of 'T'" otherwise.
func shaderString(s Shader) string {
	if st, ok := s.(interface{ String() string }); ok {
		return st.String()
	}
	switch s.(type) {
	case *Gradient:
		return "Instance of 'Gradient'"
	case *ImageShader:
		return "Instance of 'ImageShader'"
	}
	return "Instance of 'Shader'"
}

// ---- the engine side (paint.cc): Paint → ImpellerPaint ----

// paintFlags are the DisplayListAttributeFlags that matter here: which of
// the paint's attributes an operation uses.
type paintFlags struct {
	shader, colorFilter, imageFilter, maskFilter, stroke bool
	forceStroke                                          bool // drawLine/drawPoints: always stroked
	clampBlur                                            bool // drawPaint: an unspecified blur tile mode is clamp
}

var (
	flagsDrawGeometry  = paintFlags{shader: true, colorFilter: true, imageFilter: true, maskFilter: true, stroke: true}
	flagsDrawPaint     = paintFlags{shader: true, colorFilter: true, imageFilter: true, clampBlur: true}
	flagsDrawLine      = paintFlags{shader: true, colorFilter: true, imageFilter: true, maskFilter: true, stroke: true, forceStroke: true}
	flagsDrawImage     = paintFlags{colorFilter: true, imageFilter: true, maskFilter: true}
	flagsSaveLayer     = paintFlags{colorFilter: true, imageFilter: true}
	flagsDrawWithColor = paintFlags{colorFilter: true, imageFilter: true, maskFilter: true}
)

// impellerPaint builds the ImpellerPaint for p as paint.cc builds a DlPaint
// (the color goes to extended sRGB; the miter limit gets its 4 back;
// invertColors becomes a color matrix composed with a matrix color filter).
// The caller releases the result.
func (p *Paint) impellerPaint(f paintFlags) *impeller.Paint {
	ip := impeller.NewPaint()
	c := p.Color()
	if c.ColorSpace != ColorSpaceSRGB {
		cs := ColorSpaceExtendedSRGB
		c = c.WithValues(ColorWithValuesOptions{ColorSpace: &cs})
	}
	ip.SetColorWithSpace(impeller.Color{R: float32(c.R), G: float32(c.G), B: float32(c.B), A: float32(c.A),
		Space: int(ColorSpaceExtendedSRGB)})
	ip.SetBlendMode(int(p.BlendMode()))
	style := p.Style()
	if f.forceStroke {
		style = PaintingStyleStroke
	}
	if f.stroke {
		ip.SetDrawStyle(int(style))
		if style == PaintingStyleStroke {
			ip.SetStrokeWidth(p.strokeWidth)
			ip.SetStrokeMiter(p.strokeMiterLimit + kStrokeMiterLimitDefault)
			ip.SetStrokeCap(int(p.strokeCap))
			ip.SetStrokeJoin(int(p.strokeJoin))
		}
	}
	if f.shader && p.shader != nil {
		var src *impeller.ColorSource
		if is, ok := p.shader.(*ImageShader); ok {
			src = is.colorSourceWithQuality(p.FilterQuality())
		} else {
			src = p.shader.impellerColorSource()
		}
		if src != nil {
			ip.SetColorSource(src)
		}
	}
	if f.colorFilter {
		if cf := p.effectiveColorFilter(); cf != nil {
			ip.SetColorFilter(cf)
		}
	}
	if f.imageFilter && p.imageFilter != nil {
		tile := TileModeDecal // canvas.cc passes kDecal to Paint::paint for every op but drawPaint
		if f.clampBlur {
			tile = TileModeClamp
		}
		if imf := filterWithTileMode(p.imageFilter, tile); imf != nil {
			ip.SetImageFilter(imf)
		}
	}
	if f.maskFilter && p.maskFilter == maskFilterTypeBlur {
		// DlBlurMaskFilter::Make returns null for a non-positive or
		// non-finite sigma.
		if s := p.maskFilterSigma; s > 0 && !isInfOrNaN32(s) {
			if mf := impeller.NewBlurMaskFilter(int(p.maskFilterStyle), s); mf != nil {
				ip.SetMaskFilter(mf)
				mf.Release()
			}
		}
	}
	return ip
}

func isInfOrNaN32(f float32) bool { return f != f || f > 3.4028235e38 || f < -3.4028235e38 }

// invertColorMatrix is Impeller's kColorInversion
// (impeller/display_list/color_filter.h), applied for Paint.invertColors.
var invertColorMatrix = [20]float32{
	-1.0, 0, 0, 1.0, 0,
	0, -1.0, 0, 1.0, 0,
	0, 0, -1.0, 1.0, 0,
	1.0, 1.0, 1.0, 1.0, 0,
}

// effectiveColorFilter is the paint's color filter with invertColors
// applied after it, as Impeller does (invert(filter(color))). ENGINE GAP:
// the SDK can't compose two color filters, so invertColors combined with
// a non-matrix color filter (mode, gamma) drops the inversion.
func (p *Paint) effectiveColorFilter() *impeller.ColorFilter {
	cf := p.colorFilter
	if !p.InvertColors() {
		if cf == nil {
			return nil
		}
		return cf.impellerColorFilter()
	}
	if cf == nil {
		return p.cachedInvert()
	}
	if cf.kind == colorFilterMatrix && len(cf.matrix) == 20 {
		return cf.invertedImpellerColorFilter()
	}
	return cf.impellerColorFilter()
}

var invertFilter *impeller.ColorFilter

func (p *Paint) cachedInvert() *impeller.ColorFilter {
	if invertFilter == nil {
		invertFilter = impeller.NewMatrixColorFilter(invertColorMatrix)
	}
	return invertFilter
}
