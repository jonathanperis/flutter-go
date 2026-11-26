package painting

// Port of colors.dart (Flutter 3.50.1): HSVColor, HSLColor, ColorSwatch and
// ColorProperty.

import (
	"iter"
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func getHue(red, green, blue, max, delta float64) float64 {
	var hue float64
	switch {
	case max == 0.0:
		hue = 0.0
	case max == red:
		hue = float64(60.0 * dart.Mod((green-blue)/delta, 6))
	case max == green:
		hue = float64(60.0 * (((blue - red) / delta) + 2))
	case max == blue:
		hue = float64(60.0 * (((red - green) / delta) + 4))
	}
	if math.IsNaN(hue) {
		hue = 0.0
	}
	return hue
}

func lerpHue(a, b, t float64) float64 {
	delta := b - a
	if delta > 180.0 {
		delta -= 360.0
	} else if delta < -180.0 {
		delta += 360.0
	}
	return dart.Mod(a+float64(delta*t), 360.0)
}

// dartRound is Dart's double.round() (halves away from zero; throws on
// NaN and infinities, here a panic with dart.UnsupportedError).
func dartRound(x float64) int {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		panic(dart.UnsupportedError{Message: dart.DoubleToString(x)})
	}
	return int(math.Round(x))
}

func colorFromHue(alpha, hue, chroma, secondary, match float64) ui.Color {
	var red, green, blue float64
	switch {
	case hue < 60.0:
		red, green, blue = chroma, secondary, 0.0
	case hue < 120.0:
		red, green, blue = secondary, chroma, 0.0
	case hue < 180.0:
		red, green, blue = 0.0, chroma, secondary
	case hue < 240.0:
		red, green, blue = 0.0, secondary, chroma
	case hue < 300.0:
		red, green, blue = secondary, 0.0, chroma
	default:
		red, green, blue = chroma, 0.0, secondary
	}
	return ui.NewColorFromARGB(dartRound(alpha*0xFF), dartRound((red+match)*0xFF),
		dartRound((green+match)*0xFF), dartRound((blue+match)*0xFF))
}

func lerpUnit(a, b, t float64) float64 { return ui.ClampDouble(*ui.LerpDouble(&a, &b, t), 0.0, 1.0) }

func assertUnit(name string, v float64) {
	if !(v >= 0.0) || !(v <= 1.0) {
		panic(dart.NewAssertionError(name + " must be between 0.0 and 1.0: " + dart.DoubleToString(v)))
	}
}

func assertHue(v float64) {
	if !(v >= 0.0) || !(v <= 360.0) {
		panic(dart.NewAssertionError("hue must be between 0.0 and 360.0: " + dart.DoubleToString(v)))
	}
}

// HSVColor is a color represented using alpha, hue, saturation and value
// (HSV, also known as HSB). Compare with ==.
type HSVColor struct {
	// Alpha is the opacity, from 0.0 to 1.0.
	Alpha float64
	// Hue is the color's position on the color wheel, from 0.0 to 360.0.
	Hue float64
	// Saturation is the colorfulness, from 0.0 (grey) to 1.0.
	Saturation float64
	// Value is the brightness, from 0.0 (black) to 1.0.
	Value float64
}

// NewHSVColorFromAHSV is HSVColor.fromAHSV (asserting the ranges, as Dart
// does in debug mode).
func NewHSVColorFromAHSV(alpha, hue, saturation, value float64) HSVColor {
	assertUnit("alpha", alpha)
	assertHue(hue)
	assertUnit("saturation", saturation)
	assertUnit("value", value)
	return HSVColor{Alpha: alpha, Hue: hue, Saturation: saturation, Value: value}
}

// NewHSVColorFromColor is HSVColor.fromColor.
func NewHSVColorFromColor(color ui.Color) HSVColor {
	red := float64(color.Red()) / 0xFF
	green := float64(color.Green()) / 0xFF
	blue := float64(color.Blue()) / 0xFF
	max := math.Max(red, math.Max(green, blue))
	min := math.Min(red, math.Min(green, blue))
	delta := max - min
	alpha := float64(color.Alpha()) / 0xFF
	hue := getHue(red, green, blue, max, delta)
	saturation := 0.0
	if max != 0.0 {
		saturation = delta / max
	}
	return NewHSVColorFromAHSV(alpha, hue, saturation, max)
}

// WithAlpha returns a copy of this color with a different alpha.
func (c HSVColor) WithAlpha(alpha float64) HSVColor {
	return NewHSVColorFromAHSV(alpha, c.Hue, c.Saturation, c.Value)
}

// WithHue returns a copy of this color with a different hue.
func (c HSVColor) WithHue(hue float64) HSVColor {
	return NewHSVColorFromAHSV(c.Alpha, hue, c.Saturation, c.Value)
}

// WithSaturation returns a copy of this color with a different saturation.
func (c HSVColor) WithSaturation(saturation float64) HSVColor {
	return NewHSVColorFromAHSV(c.Alpha, c.Hue, saturation, c.Value)
}

// WithValue returns a copy of this color with a different value.
func (c HSVColor) WithValue(value float64) HSVColor {
	return NewHSVColorFromAHSV(c.Alpha, c.Hue, c.Saturation, value)
}

// ToColor returns this color in RGB.
func (c HSVColor) ToColor() ui.Color {
	chroma := float64(c.Saturation * c.Value)
	secondary := float64(chroma * (1.0 - math.Abs(dart.Mod(c.Hue/60.0, 2.0)-1.0)))
	match := c.Value - chroma
	return colorFromHue(c.Alpha, c.Hue, chroma, secondary, match)
}

func (c HSVColor) scaleAlpha(factor float64) HSVColor { return c.WithAlpha(c.Alpha * factor) }

// HSVColorLerp is HSVColor.lerp: linear interpolation, hue along the
// shortest arc; nil is Dart's null (a null end fades the other's alpha).
func HSVColorLerp(a, b *HSVColor, t float64) *HSVColor {
	if a == b {
		return a
	}
	var r HSVColor
	switch {
	case a == nil:
		r = b.scaleAlpha(t)
	case b == nil:
		r = a.scaleAlpha(1.0 - t)
	default:
		r = NewHSVColorFromAHSV(lerpUnit(a.Alpha, b.Alpha, t), lerpHue(a.Hue, b.Hue, t),
			lerpUnit(a.Saturation, b.Saturation, t), lerpUnit(a.Value, b.Value, t))
	}
	return &r
}

func (c HSVColor) String() string {
	return foundation.ObjectRuntimeType(c, "HSVColor") + "(" + dart.DoubleToString(c.Alpha) + ", " +
		dart.DoubleToString(c.Hue) + ", " + dart.DoubleToString(c.Saturation) + ", " + dart.DoubleToString(c.Value) + ")"
}

// HSLColor is a color represented using alpha, hue, saturation and
// lightness. Compare with ==.
type HSLColor struct {
	// Alpha is the opacity, from 0.0 to 1.0.
	Alpha float64
	// Hue is the color's position on the color wheel, from 0.0 to 360.0.
	Hue float64
	// Saturation is the colorfulness, from 0.0 (grey) to 1.0.
	Saturation float64
	// Lightness is from 0.0 (black) through 0.5 (fully saturated) to 1.0
	// (white).
	Lightness float64
}

// NewHSLColorFromAHSL is HSLColor.fromAHSL (asserting the ranges).
func NewHSLColorFromAHSL(alpha, hue, saturation, lightness float64) HSLColor {
	assertUnit("alpha", alpha)
	assertHue(hue)
	assertUnit("saturation", saturation)
	assertUnit("lightness", lightness)
	return HSLColor{Alpha: alpha, Hue: hue, Saturation: saturation, Lightness: lightness}
}

// NewHSLColorFromColor is HSLColor.fromColor.
func NewHSLColorFromColor(color ui.Color) HSLColor {
	red := float64(color.Red()) / 0xFF
	green := float64(color.Green()) / 0xFF
	blue := float64(color.Blue()) / 0xFF
	max := math.Max(red, math.Max(green, blue))
	min := math.Min(red, math.Min(green, blue))
	delta := max - min
	alpha := float64(color.Alpha()) / 0xFF
	hue := getHue(red, green, blue, max, delta)
	lightness := (max + min) / 2.0
	// Saturation can exceed 1.0 with rounding errors, so clamp it.
	saturation := 0.0
	if min != max {
		saturation = ui.ClampDouble(delta/(1.0-math.Abs(float64(2.0*lightness)-1.0)), 0.0, 1.0)
	}
	return NewHSLColorFromAHSL(alpha, hue, saturation, lightness)
}

// WithAlpha returns a copy of this color with a different alpha.
func (c HSLColor) WithAlpha(alpha float64) HSLColor {
	return NewHSLColorFromAHSL(alpha, c.Hue, c.Saturation, c.Lightness)
}

// WithHue returns a copy of this color with a different hue.
func (c HSLColor) WithHue(hue float64) HSLColor {
	return NewHSLColorFromAHSL(c.Alpha, hue, c.Saturation, c.Lightness)
}

// WithSaturation returns a copy of this color with a different saturation.
func (c HSLColor) WithSaturation(saturation float64) HSLColor {
	return NewHSLColorFromAHSL(c.Alpha, c.Hue, saturation, c.Lightness)
}

// WithLightness returns a copy of this color with a different lightness.
func (c HSLColor) WithLightness(lightness float64) HSLColor {
	return NewHSLColorFromAHSL(c.Alpha, c.Hue, c.Saturation, lightness)
}

// ToColor returns this color in RGB.
func (c HSLColor) ToColor() ui.Color {
	chroma := float64((1.0 - math.Abs(float64(2.0*c.Lightness)-1.0)) * c.Saturation)
	secondary := float64(chroma * (1.0 - math.Abs(dart.Mod(c.Hue/60.0, 2.0)-1.0)))
	match := c.Lightness - chroma/2.0
	return colorFromHue(c.Alpha, c.Hue, chroma, secondary, match)
}

func (c HSLColor) scaleAlpha(factor float64) HSLColor { return c.WithAlpha(c.Alpha * factor) }

// HSLColorLerp is HSLColor.lerp (see HSVColorLerp).
func HSLColorLerp(a, b *HSLColor, t float64) *HSLColor {
	if a == b {
		return a
	}
	var r HSLColor
	switch {
	case a == nil:
		r = b.scaleAlpha(t)
	case b == nil:
		r = a.scaleAlpha(1.0 - t)
	default:
		r = NewHSLColorFromAHSL(lerpUnit(a.Alpha, b.Alpha, t), lerpHue(a.Hue, b.Hue, t),
			lerpUnit(a.Saturation, b.Saturation, t), lerpUnit(a.Lightness, b.Lightness, t))
	}
	return &r
}

func (c HSLColor) String() string {
	return foundation.ObjectRuntimeType(c, "HSLColor") + "(" + dart.DoubleToString(c.Alpha) + ", " +
		dart.DoubleToString(c.Hue) + ", " + dart.DoubleToString(c.Saturation) + ", " + dart.DoubleToString(c.Lightness) + ")"
}

// ColorSwatch is a color that has a small table of related colors called
// a "swatch" (Dart: `class ColorSwatch<T> extends Color`). It embeds its
// primary ui.Color; the swatch keeps Dart's map insertion order. Used by
// pointer; == is Equal.
type ColorSwatch[T comparable] struct {
	ui.Color
	swatch []dart.MapEntry[T, ui.Color]
}

// NewColorSwatch is ColorSwatch(primary, swatch): primary is the
// 0xAARRGGBB value of the primary color; swatch's entries are the Dart
// map's, in order.
func NewColorSwatch[T comparable](primary int, swatch []dart.MapEntry[T, ui.Color]) *ColorSwatch[T] {
	return &ColorSwatch[T]{Color: ui.NewColor(primary), swatch: swatch}
}

// Get is operator []: the color for key, nil if the swatch has none.
func (s *ColorSwatch[T]) Get(key T) *ui.Color {
	for _, e := range s.swatch {
		if e.Key == key {
			c := e.Value
			return &c
		}
	}
	return nil
}

// Keys returns the swatch's keys in order.
func (s *ColorSwatch[T]) Keys() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, e := range s.swatch {
			if !yield(e.Key) {
				return
			}
		}
	}
}

// Swatch is the protected _swatch map (as ordered entries; don't mutate).
func (s *ColorSwatch[T]) Swatch() []dart.MapEntry[T, ui.Color] { return s.swatch }

// Equal is Dart's ==: same primary color and an equal swatch (mapEquals:
// same keys and colors, in any order).
func (s *ColorSwatch[T]) Equal(other *ColorSwatch[T]) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil || s.Color != other.Color || len(s.swatch) != len(other.swatch) {
		return false
	}
	for _, e := range s.swatch {
		c := other.Get(e.Key)
		if c == nil || *c != e.Value {
			return false
		}
	}
	return true
}

func (s *ColorSwatch[T]) String() string {
	return foundation.ObjectRuntimeType(s, "ColorSwatch") + "(primary value: " + s.Color.String() + ")"
}

// ColorSwatchLerp is ColorSwatch.lerp (nil is Dart's null).
func ColorSwatchLerp[T comparable](a, b *ColorSwatch[T], t float64) *ColorSwatch[T] {
	if a == b {
		return a
	}
	var swatch []dart.MapEntry[T, ui.Color]
	var ac, bc *ui.Color
	switch {
	case b == nil:
		ac = &a.Color
		for _, e := range a.swatch {
			swatch = append(swatch, dart.MapEntry[T, ui.Color]{Key: e.Key, Value: *ui.ColorLerp(&e.Value, nil, t)})
		}
	case a == nil:
		bc = &b.Color
		for _, e := range b.swatch {
			swatch = append(swatch, dart.MapEntry[T, ui.Color]{Key: e.Key, Value: *ui.ColorLerp(nil, &e.Value, t)})
		}
	default:
		ac, bc = &a.Color, &b.Color
		for _, e := range a.swatch {
			swatch = append(swatch, dart.MapEntry[T, ui.Color]{Key: e.Key, Value: *ui.ColorLerp(&e.Value, b.Get(e.Key), t)})
		}
	}
	return NewColorSwatch(ui.ColorLerp(ac, bc, t).Value(), swatch)
}

// ColorProperty is a DiagnosticsProperty that has a Color as value (nil:
// null); its JSON adds the color's components as `valueProperties`.
type ColorProperty struct {
	foundation.DiagnosticsProperty[*ui.Color]
}

// ColorPropertyOptions are ColorProperty's named parameters.
type ColorPropertyOptions struct {
	ShowName     *bool
	DefaultValue any
	Style        *foundation.DiagnosticsTreeStyle
	Level        *foundation.DiagnosticLevel
}

// NewColorProperty is ColorProperty(name, value, {showName, defaultValue,
// style, level}); value is a ui.Color or a *ui.Color (nil: null).
func NewColorProperty[V ui.Color | *ui.Color](name string, value V, opts ...ColorPropertyOptions) *ColorProperty {
	var o ColorPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	var v *ui.Color
	switch x := any(value).(type) {
	case ui.Color:
		v = &x
	case *ui.Color:
		v = x
	}
	p := &ColorProperty{DiagnosticsProperty: *foundation.NewDiagnosticsProperty(name, v,
		foundation.DiagnosticsPropertyOptions{ShowName: o.ShowName, DefaultValue: o.DefaultValue, Style: o.Style, Level: o.Level})}
	p.SetSelf(p)
	return p
}

// ToJsonMap adds valueProperties (red, green, blue, alpha) when the value
// is not null.
func (p *ColorProperty) ToJsonMap(delegate foundation.DiagnosticsSerializationDelegate) map[string]any {
	json := p.DiagnosticsProperty.ToJsonMap(delegate)
	if v := p.TypedValue(); v != nil {
		json["valueProperties"] = map[string]any{
			"red":   v.Red(),
			"green": v.Green(),
			"blue":  v.Blue(),
			"alpha": v.Alpha(),
		}
	}
	return json
}
