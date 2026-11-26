package ui

// Port of engine/src/flutter/lib/ui/painting.dart (Flutter 3.50.1): Color
// and ColorSpace. The rest of painting.dart is in painting_*.go.

import (
	"math"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// ColorSpace is the color space a Color's components are expressed in.
type ColorSpace int

const (
	ColorSpaceSRGB ColorSpace = iota
	ColorSpaceExtendedSRGB
	ColorSpaceDisplayP3
)

func (c ColorSpace) String() string {
	return [...]string{"ColorSpace.sRGB", "ColorSpace.extendedSRGB", "ColorSpace.displayP3"}[c]
}

// Color is an immutable color: alpha, red, green and blue as doubles in
// [0, 1] (beyond that for extended sRGB), in a color space.
type Color struct {
	A, R, G, B float64
	ColorSpace ColorSpace
}

// NewColor is Color(int value): 0xAARRGGBB in sRGB.
func NewColor(value int) Color {
	return colorFromARGBC(value>>24, value>>16, value>>8, value, ColorSpaceSRGB)
}

// ColorFromOptions are Color.from's named arguments.
type ColorFromOptions struct {
	Alpha, Red, Green, Blue float64
	ColorSpace              ColorSpace // default sRGB (the zero value)
}

// NewColorFrom is Color.from(alpha:, red:, green:, blue:, colorSpace:).
func NewColorFrom(o ColorFromOptions) Color {
	return Color{A: o.Alpha, R: o.Red, G: o.Green, B: o.Blue, ColorSpace: o.ColorSpace}
}

// NewColorFromARGB is Color.fromARGB(a, r, g, b), each an 8-bit int.
func NewColorFromARGB(a, r, g, b int) Color { return colorFromARGBC(a, r, g, b, ColorSpaceSRGB) }

func colorFromARGBC(alpha, red, green, blue int, cs ColorSpace) Color {
	return colorFromRGBOC(red, green, blue, float64(alpha&0xff)/255, cs)
}

// NewColorFromRGBO is Color.fromRGBO(r, g, b, opacity).
func NewColorFromRGBO(r, g, b int, opacity float64) Color {
	return colorFromRGBOC(r, g, b, opacity, ColorSpaceSRGB)
}

func colorFromRGBOC(r, g, b int, opacity float64, cs ColorSpace) Color {
	return Color{A: opacity, R: float64(r&0xff) / 255, G: float64(g&0xff) / 255, B: float64(b&0xff) / 255, ColorSpace: cs}
}

// dartRound is Dart's double.round(): halves away from zero; it throws
// (here: panics with dart.UnsupportedError) on NaN and infinities.
func dartRound(x float64) int {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		panic(dart.UnsupportedError{Message: dart.DoubleToString(x)})
	}
	return int(math.Round(x))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func floatToInt8(x float64) int { return clampInt(dartRound(x*255.0), 0, 255) }

// Value is the deprecated Color.value (use ToARGB32).
func (c Color) Value() int { return c.ToARGB32() }

// ToARGB32 is the color as 0xAARRGGBB.
func (c Color) ToARGB32() int {
	return floatToInt8(c.A)<<24 | floatToInt8(c.R)<<16 | floatToInt8(c.G)<<8 | floatToInt8(c.B)<<0
}

// Alpha, Opacity, Red, Green and Blue are the deprecated 8-bit accessors.
func (c Color) Alpha() int            { return (0xff000000 & c.Value()) >> 24 }
func (c Color) Opacity() float64      { return float64(c.Alpha()) / 0xFF }
func (c Color) Red() int              { return (0x00ff0000 & c.Value()) >> 16 }
func (c Color) Green() int            { return (0x0000ff00 & c.Value()) >> 8 }
func (c Color) Blue() int             { return (0x000000ff & c.Value()) >> 0 }
func (c Color) WithAlpha(a int) Color { return NewColorFromARGB(a, c.Red(), c.Green(), c.Blue()) }
func (c Color) WithRed(r int) Color   { return NewColorFromARGB(c.Alpha(), r, c.Green(), c.Blue()) }
func (c Color) WithGreen(g int) Color { return NewColorFromARGB(c.Alpha(), c.Red(), g, c.Blue()) }
func (c Color) WithBlue(b int) Color  { return NewColorFromARGB(c.Alpha(), c.Red(), c.Green(), b) }

// WithOpacity is the deprecated Color.withOpacity.
func (c Color) WithOpacity(opacity float64) Color {
	if opacity < 0 || opacity > 1 {
		panic("ui: opacity must be in [0, 1]")
	}
	return c.WithAlpha(dartRound(255.0 * opacity))
}

// ColorWithValuesOptions are withValues' named arguments; nil keeps the
// current value.
type ColorWithValuesOptions struct {
	Alpha, Red, Green, Blue *float64
	ColorSpace              *ColorSpace
}

// WithValues returns a copy with the given components replaced, converted
// to ColorSpace when one is given.
func (c Color) WithValues(o ColorWithValuesOptions) Color {
	var updated *Color
	if o.Alpha != nil || o.Red != nil || o.Green != nil || o.Blue != nil {
		or := func(p *float64, d float64) float64 {
			if p != nil {
				return *p
			}
			return d
		}
		u := Color{A: or(o.Alpha, c.A), R: or(o.Red, c.R), G: or(o.Green, c.G), B: or(o.Blue, c.B), ColorSpace: c.ColorSpace}
		updated = &u
	}
	src := c
	if updated != nil {
		src = *updated
	}
	if o.ColorSpace != nil && *o.ColorSpace != c.ColorSpace {
		return colorTransform(c.ColorSpace, *o.ColorSpace)(src, *o.ColorSpace)
	}
	return src
}

func linearizeColorComponent(component float64) float64 {
	if component <= 0.03928 {
		return component / 12.92
	}
	return dart.Pow((component+0.055)/1.055, 2.4)
}

// ComputeLuminance is the relative luminance (WCAG), from 0 to 1.
func (c Color) ComputeLuminance() float64 {
	if c.ColorSpace == ColorSpaceExtendedSRGB {
		panic("ui: computeLuminance doesn't support extended sRGB")
	}
	// float64(...) around each product stops Go from fusing multiply-adds
	// (arm64), which Dart never does; without it the last bit differs.
	return float64(0.2126*linearizeColorComponent(c.R)) + float64(0.7152*linearizeColorComponent(c.G)) + float64(0.0722*linearizeColorComponent(c.B))
}

func scaleAlpha(x Color, factor float64) Color {
	a := ClampDouble(x.A*factor, 0, 1)
	return x.WithValues(ColorWithValuesOptions{Alpha: &a})
}

func widerColorSpace(a, b ColorSpace) ColorSpace {
	if a == ColorSpaceDisplayP3 || b == ColorSpaceDisplayP3 {
		return ColorSpaceDisplayP3
	}
	return a
}

// ColorLerp is Color.lerp; nil stands for Dart's null.
func ColorLerp(x, y *Color, t float64) *Color {
	if (x != nil && x.ColorSpace == ColorSpaceExtendedSRGB) || (y != nil && y.ColorSpace == ColorSpaceExtendedSRGB) {
		panic("ui: Color.lerp doesn't support extended sRGB")
	}
	switch {
	case x == nil && y == nil:
		return nil
	case y == nil:
		r := scaleAlpha(*x, 1.0-t)
		return &r
	case x == nil:
		r := scaleAlpha(*y, t)
		return &r
	}
	a, b, cs := *x, *y, x.ColorSpace
	if x.ColorSpace != y.ColorSpace {
		cs = widerColorSpace(x.ColorSpace, y.ColorSpace)
		a = x.WithValues(ColorWithValuesOptions{ColorSpace: &cs})
		b = y.WithValues(ColorWithValuesOptions{ColorSpace: &cs})
	}
	r := Color{A: ClampDouble(lerpDouble(a.A, b.A, t), 0, 1), R: ClampDouble(lerpDouble(a.R, b.R, t), 0, 1),
		G: ClampDouble(lerpDouble(a.G, b.G, t), 0, 1), B: ClampDouble(lerpDouble(a.B, b.B, t), 0, 1), ColorSpace: cs}
	return &r
}

// ColorAlphaBlend is Color.alphaBlend: foreground composited over background.
func ColorAlphaBlend(foreground, background Color) Color {
	if foreground.ColorSpace != background.ColorSpace || foreground.ColorSpace == ColorSpaceExtendedSRGB {
		panic("ui: alphaBlend needs matching, non-extended color spaces")
	}
	alpha := foreground.A
	if alpha == 0 {
		return background
	}
	invAlpha := 1 - alpha
	backAlpha := background.A
	if backAlpha == 1 {
		return Color{A: 1, R: float64(alpha*foreground.R) + float64(invAlpha*background.R),
			G: float64(alpha*foreground.G) + float64(invAlpha*background.G),
			B: float64(alpha*foreground.B) + float64(invAlpha*background.B), ColorSpace: foreground.ColorSpace}
	}
	backAlpha = float64(backAlpha * invAlpha)
	outAlpha := alpha + backAlpha
	return Color{A: outAlpha, R: (float64(foreground.R*alpha) + float64(background.R*backAlpha)) / outAlpha,
		G: (float64(foreground.G*alpha) + float64(background.G*backAlpha)) / outAlpha,
		B: (float64(foreground.B*alpha) + float64(background.B*backAlpha)) / outAlpha, ColorSpace: foreground.ColorSpace}
}

// ColorGetAlphaFromOpacity is Color.getAlphaFromOpacity.
func ColorGetAlphaFromOpacity(opacity float64) int {
	return dartRound(ClampDouble(opacity, 0.0, 1.0) * 255)
}

func (c Color) String() string {
	f := func(v float64) string { return dart.ToStringAsFixed(v, 4) }
	return "Color(alpha: " + f(c.A) + ", red: " + f(c.R) + ", green: " + f(c.G) + ", blue: " + f(c.B) +
		", colorSpace: " + c.ColorSpace.String() + ")"
}

// ---- color space transforms (painting.dart's _ColorTransform) ----

type colorTransformFunc func(c Color, result ColorSpace) Color

const (
	kSrgbGamma                    = 2.4
	kSrgbLinearThreshold          = 0.04045
	kSrgbLinearSlope              = 12.92
	kSrgbEncodedOffset            = 0.055
	kSrgbEncodedDivisor           = 1.055
	kSrgbLinearToEncodedThreshold = 0.0031308
)

func srgbEOTF(v float64) float64 {
	if v <= kSrgbLinearThreshold {
		return v / kSrgbLinearSlope
	}
	return dart.Pow((v+kSrgbEncodedOffset)/kSrgbEncodedDivisor, kSrgbGamma)
}

func srgbOETF(v float64) float64 {
	if v <= kSrgbLinearToEncodedThreshold {
		return v * kSrgbLinearSlope
	}
	return float64(kSrgbEncodedDivisor*dart.Pow(v, 1.0/kSrgbGamma)) - kSrgbEncodedOffset // no FMA, as in Dart
}

func srgbEOTFExtended(v float64) float64 {
	if v < 0 {
		return -srgbEOTF(-v)
	}
	return srgbEOTF(v)
}

func srgbOETFExtended(v float64) float64 {
	if v < 0 {
		return -srgbOETF(-v)
	}
	return srgbOETF(v)
}

var kP3ToSrgbLinear = [9]float64{1.2249401, -0.2249402, 0.0, -0.0420569, 1.0420571, 0.0, -0.0196376, -0.0786507, 1.0982884}
var kSrgbToP3Linear = [9]float64{0.8224622, 0.1775380, 0.0, 0.0331942, 0.9668058, 0.0, 0.0170806, 0.0723974, 0.9105220}

func matrixTransform(m [9]float64) colorTransformFunc {
	return func(c Color, cs ColorSpace) Color {
		rLin, gLin, bLin := srgbEOTFExtended(c.R), srgbEOTFExtended(c.G), srgbEOTFExtended(c.B)
		rOut := float64(m[0]*rLin) + float64(m[1]*gLin) + float64(m[2]*bLin) // no FMA, as in Dart
		gOut := float64(m[3]*rLin) + float64(m[4]*gLin) + float64(m[5]*bLin)
		bOut := float64(m[6]*rLin) + float64(m[7]*gLin) + float64(m[8]*bLin)
		return Color{A: c.A, R: srgbOETFExtended(rOut), G: srgbOETFExtended(gOut), B: srgbOETFExtended(bOut), ColorSpace: cs}
	}
}

func identityTransform(c Color, _ ColorSpace) Color { return c }

// clampTransform is _ClampTransform. Flutter's never calls its child
// transform: it only clamps the input, so extendedSRGB → displayP3 does no
// P3 conversion. Kept as is.
func clampTransform(colorTransformFunc) colorTransformFunc {
	return func(c Color, cs ColorSpace) Color {
		return Color{A: ClampDouble(c.A, 0, 1), R: ClampDouble(c.R, 0, 1), G: ClampDouble(c.G, 0, 1),
			B: ClampDouble(c.B, 0, 1), ColorSpace: cs}
	}
}

func colorTransform(source, destination ColorSpace) colorTransformFunc {
	p3ToSrgb, srgbToP3 := matrixTransform(kP3ToSrgbLinear), matrixTransform(kSrgbToP3Linear)
	switch source {
	case ColorSpaceSRGB:
		if destination == ColorSpaceDisplayP3 {
			return srgbToP3
		}
		return identityTransform
	case ColorSpaceExtendedSRGB:
		switch destination {
		case ColorSpaceSRGB:
			return clampTransform(identityTransform)
		case ColorSpaceDisplayP3:
			return clampTransform(srgbToP3)
		}
		return identityTransform
	}
	switch destination { // displayP3
	case ColorSpaceSRGB:
		return clampTransform(p3ToSrgb)
	case ColorSpaceExtendedSRGB:
		return p3ToSrgb
	}
	return identityTransform
}
