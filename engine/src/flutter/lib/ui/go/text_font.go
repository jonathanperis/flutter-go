package ui

// Port of engine/src/flutter/lib/ui/text.dart (Flutter 3.50.1), part 1:
// kTextHeightNone, FontStyle, FontWeight, FontFeature, FontVariation,
// TextAlign, Hyphens, TextBaseline, TextDecoration, TextDecorationStyle,
// TextLeadingDistribution and TextHeightBehavior.

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// KTextHeightNone is kTextHeightNone: the height value that means "use the
// font's own line height" (reset an inherited height).
const KTextHeightNone = 0.0

// FontStyle is whether to slant the glyphs in the font.
type FontStyle int

const (
	// FontStyleNormal uses the upright glyphs.
	FontStyleNormal FontStyle = iota
	// FontStyleItalic uses glyphs designed for slanting.
	FontStyleItalic
)

// FontStyleValues is FontStyle.values.
var FontStyleValues = []FontStyle{FontStyleNormal, FontStyleItalic}

// Name is the enum value's name.
func (s FontStyle) Name() string { return [...]string{"normal", "italic"}[s] }

func (s FontStyle) String() string { return "FontStyle." + s.Name() }

// FontWeight is the thickness of the glyphs used to draw the text (a value
// from 1 to 1000). Compare with ==.
type FontWeight struct {
	// Value is the thickness value of this font weight.
	Value int
}

// NewFontWeight is FontWeight(value) (1 ≤ value ≤ 1000).
func NewFontWeight(value int) FontWeight {
	dartAssert(value >= 1, "Font weight must be between 1 and 1000")
	dartAssert(value <= 1000, "Font weight must be between 1 and 1000")
	return FontWeight{Value: value}
}

// Index is the encoded integer value of this font weight (deprecated:
// value is more precise).
func (w FontWeight) Index() int {
	i := w.Value/100 - 1
	return min(max(i, 0), 8)
}

// The font weights (FontWeight.w100 ... w900, normal, bold).
var (
	FontWeightW100   = FontWeight{100}
	FontWeightW200   = FontWeight{200}
	FontWeightW300   = FontWeight{300}
	FontWeightW400   = FontWeight{400}
	FontWeightW500   = FontWeight{500}
	FontWeightW600   = FontWeight{600}
	FontWeightW700   = FontWeight{700}
	FontWeightW800   = FontWeight{800}
	FontWeightW900   = FontWeight{900}
	FontWeightNormal = FontWeightW400
	FontWeightBold   = FontWeightW700
)

// FontWeightValues is FontWeight.values (w100 to w900).
var FontWeightValues = []FontWeight{
	FontWeightW100, FontWeightW200, FontWeightW300, FontWeightW400, FontWeightW500,
	FontWeightW600, FontWeightW700, FontWeightW800, FontWeightW900,
}

// lerpInt is lerp.dart's _lerpInt.
func lerpInt(a, b int, t float64) float64 { return float64(a) + float64(float64(b-a)*t) }

// FontWeightLerp is FontWeight.lerp (nil is null): the interpolated value,
// rounded and clamped to 100..900.
func FontWeightLerp(a, b *FontWeight, t float64) *FontWeight {
	if a == nil && b == nil {
		return nil
	}
	av, bv := FontWeightNormal.Value, FontWeightNormal.Value
	if a != nil {
		av = a.Value
	}
	if b != nil {
		bv = b.Value
	}
	v := int(math.Round(lerpInt(av, bv, t)))
	w := NewFontWeight(min(max(v, 100), 900))
	return &w
}

func (w FontWeight) String() string {
	if w.Value%100 != 0 {
		return "FontWeight(" + strconv.Itoa(w.Value) + ")"
	}
	return "FontWeight.w" + strconv.Itoa((w.Index()+1)*100)
}

// FontFeature is a feature tag and value that affect the selection of
// glyphs in a font. Compare with ==.
type FontFeature struct {
	// Feature is the tag that identifies the effect of this feature (four
	// ASCII characters).
	Feature string
	// Value is the value assigned to this feature (0 disables it, 1
	// enables it; other values select alternates).
	Value int
}

// NewFontFeature is FontFeature(feature, [value = 1]).
func NewFontFeature(feature string, value ...int) FontFeature {
	v := 1
	if len(value) > 0 {
		v = value[0]
	}
	dartAssert(len(dart.UTF16CodeUnits(feature)) == 4, "Feature tag must be exactly four characters long.")
	dartAssert(v >= 0, "Feature value must be zero or a positive integer.")
	return FontFeature{Feature: feature, Value: v}
}

// NewFontFeatureEnable is FontFeature.enable(feature).
func NewFontFeatureEnable(feature string) FontFeature { return NewFontFeature(feature, 1) }

// NewFontFeatureDisable is FontFeature.disable(feature).
func NewFontFeatureDisable(feature string) FontFeature { return NewFontFeature(feature, 0) }

// NewFontFeatureAlternative is FontFeature.alternative(value) ('aalt').
func NewFontFeatureAlternative(value int) FontFeature {
	return FontFeature{Feature: "aalt", Value: value}
}

// NewFontFeatureAlternativeFractions is FontFeature.alternativeFractions().
func NewFontFeatureAlternativeFractions() FontFeature { return FontFeature{"afrc", 1} }

// NewFontFeatureContextualAlternates is FontFeature.contextualAlternates().
func NewFontFeatureContextualAlternates() FontFeature { return FontFeature{"calt", 1} }

// NewFontFeatureCaseSensitiveForms is FontFeature.caseSensitiveForms().
func NewFontFeatureCaseSensitiveForms() FontFeature { return FontFeature{"case", 1} }

// NewFontFeatureCharacterVariant is FontFeature.characterVariant(value)
// (1 ≤ value ≤ 99: 'cv01'...).
func NewFontFeatureCharacterVariant(value int) FontFeature {
	textAssert(value >= 1)
	textAssert(value <= 99)
	return NewFontFeature(fmt.Sprintf("cv%02d", value))
}

// NewFontFeatureDenominator is FontFeature.denominator().
func NewFontFeatureDenominator() FontFeature { return FontFeature{"dnom", 1} }

// NewFontFeatureFractions is FontFeature.fractions().
func NewFontFeatureFractions() FontFeature { return FontFeature{"frac", 1} }

// NewFontFeatureHistoricalForms is FontFeature.historicalForms().
func NewFontFeatureHistoricalForms() FontFeature { return FontFeature{"hist", 1} }

// NewFontFeatureHistoricalLigatures is FontFeature.historicalLigatures().
func NewFontFeatureHistoricalLigatures() FontFeature { return FontFeature{"hlig", 1} }

// NewFontFeatureLiningFigures is FontFeature.liningFigures().
func NewFontFeatureLiningFigures() FontFeature { return FontFeature{"lnum", 1} }

// FontFeatureLocaleAwareOptions are FontFeature.localeAware's named
// arguments (Enable nil: true).
type FontFeatureLocaleAwareOptions struct {
	Enable *bool
}

// NewFontFeatureLocaleAware is FontFeature.localeAware({enable = true}).
func NewFontFeatureLocaleAware(opts ...FontFeatureLocaleAwareOptions) FontFeature {
	v := 1
	if len(opts) > 0 && opts[0].Enable != nil && !*opts[0].Enable {
		v = 0
	}
	return FontFeature{"locl", v}
}

// NewFontFeatureNotationalForms is FontFeature.notationalForms([value = 1]).
func NewFontFeatureNotationalForms(value ...int) FontFeature {
	v := 1
	if len(value) > 0 {
		v = value[0]
	}
	textAssert(v >= 0)
	return FontFeature{"nalt", v}
}

// NewFontFeatureNumerators is FontFeature.numerators().
func NewFontFeatureNumerators() FontFeature { return FontFeature{"numr", 1} }

// NewFontFeatureOldstyleFigures is FontFeature.oldstyleFigures().
func NewFontFeatureOldstyleFigures() FontFeature { return FontFeature{"onum", 1} }

// NewFontFeatureOrdinalForms is FontFeature.ordinalForms().
func NewFontFeatureOrdinalForms() FontFeature { return FontFeature{"ordn", 1} }

// NewFontFeatureProportionalFigures is FontFeature.proportionalFigures().
func NewFontFeatureProportionalFigures() FontFeature { return FontFeature{"pnum", 1} }

// NewFontFeatureRandomize is FontFeature.randomize().
func NewFontFeatureRandomize() FontFeature { return FontFeature{"rand", 1} }

// NewFontFeatureStylisticAlternates is FontFeature.stylisticAlternates().
func NewFontFeatureStylisticAlternates() FontFeature { return FontFeature{"salt", 1} }

// NewFontFeatureScientificInferiors is FontFeature.scientificInferiors().
func NewFontFeatureScientificInferiors() FontFeature { return FontFeature{"sinf", 1} }

// NewFontFeatureStylisticSet is FontFeature.stylisticSet(value)
// (1 ≤ value ≤ 20: 'ss01'...).
func NewFontFeatureStylisticSet(value int) FontFeature {
	textAssert(value >= 1)
	textAssert(value <= 20)
	return NewFontFeature(fmt.Sprintf("ss%02d", value))
}

// NewFontFeatureSubscripts is FontFeature.subscripts().
func NewFontFeatureSubscripts() FontFeature { return FontFeature{"subs", 1} }

// NewFontFeatureSuperscripts is FontFeature.superscripts().
func NewFontFeatureSuperscripts() FontFeature { return FontFeature{"sups", 1} }

// NewFontFeatureSwash is FontFeature.swash([value = 1]).
func NewFontFeatureSwash(value ...int) FontFeature {
	v := 1
	if len(value) > 0 {
		v = value[0]
	}
	textAssert(v >= 0)
	return FontFeature{"swsh", v}
}

// NewFontFeatureTabularFigures is FontFeature.tabularFigures().
func NewFontFeatureTabularFigures() FontFeature { return FontFeature{"tnum", 1} }

// NewFontFeatureSlashedZero is FontFeature.slashedZero().
func NewFontFeatureSlashedZero() FontFeature { return FontFeature{"zero", 1} }

// fontFeatureEncodedSize is FontFeature._kEncodedSize.
const fontFeatureEncodedSize = 8

// encode is FontFeature._encode: the tag's four code units, then the value
// as an int32 (host endian).
func (f FontFeature) encode(b []byte) {
	units := dart.UTF16CodeUnits(f.Feature)
	for _, c := range units {
		textAssert(c >= 0x20 && c <= 0x7F)
	}
	for i := 0; i < 4; i++ {
		b[i] = byte(units[i])
	}
	putInt32(b[4:], int32(f.Value))
}

func (f FontFeature) String() string {
	return "FontFeature('" + f.Feature + "', " + strconv.Itoa(f.Value) + ")"
}

// FontVariation is an axis tag and value that select a variation of a
// variable font. Compare with ==.
type FontVariation struct {
	// Axis is the tag that identifies the design axis (four ASCII
	// characters).
	Axis string
	// Value is the value assigned to this design axis.
	Value float64
}

// NewFontVariation is FontVariation(axis, value).
func NewFontVariation(axis string, value float64) FontVariation {
	dartAssert(len(dart.UTF16CodeUnits(axis)) == 4, "Axis tag must be exactly four characters long.")
	dartAssert(value >= -32768.0 && value < 32768.0,
		"Value must be representable as a signed 16.16 fixed-point number, i.e. it must be in this range: -32768.0 ≤ value < 32768.0")
	return FontVariation{Axis: axis, Value: value}
}

// NewFontVariationItalic is FontVariation.italic(value) ('ital', 0..1).
func NewFontVariationItalic(value float64) FontVariation {
	textAssert(value >= 0.0)
	textAssert(value <= 1.0)
	return FontVariation{"ital", value}
}

// NewFontVariationOpticalSize is FontVariation.opticalSize(value) ('opsz').
func NewFontVariationOpticalSize(value float64) FontVariation {
	textAssert(value > 0.0)
	return FontVariation{"opsz", value}
}

// NewFontVariationSlant is FontVariation.slant(value) ('slnt', -90..90).
func NewFontVariationSlant(value float64) FontVariation {
	textAssert(value > -90.0)
	textAssert(value < 90.0)
	return FontVariation{"slnt", value}
}

// NewFontVariationWidth is FontVariation.width(value) ('wdth').
func NewFontVariationWidth(value float64) FontVariation {
	textAssert(value >= 0.0)
	return FontVariation{"wdth", value}
}

// NewFontVariationWeight is FontVariation.weight(value) ('wght', 1..1000).
func NewFontVariationWeight(value float64) FontVariation {
	textAssert(value >= 1)
	textAssert(value <= 1000)
	return FontVariation{"wght", value}
}

// fontVariationEncodedSize is FontVariation._kEncodedSize.
const fontVariationEncodedSize = 8

// encode is FontVariation._encode: the tag's four code units, then the
// value as a float32 (host endian).
func (v FontVariation) encode(b []byte) {
	units := dart.UTF16CodeUnits(v.Axis)
	for _, c := range units {
		textAssert(c >= 0x20 && c <= 0x7F)
	}
	for i := 0; i < 4; i++ {
		b[i] = byte(units[i])
	}
	putInt32(b[4:], int32(math.Float32bits(float32(v.Value))))
}

// FontVariationLerp is FontVariation.lerp (nil is null): variations of
// different axes don't interpolate (a below t = 0.5, else b).
func FontVariationLerp(a, b *FontVariation, t float64) *FontVariation {
	if (a == nil) != (b == nil) || (a != nil && a.Axis != b.Axis) || (a == nil && b == nil) {
		if t < 0.5 {
			return a
		}
		return b
	}
	r := NewFontVariation(a.Axis, ClampDouble(lerpDouble(a.Value, b.Value, t), -32768.0, 32768.0-1.0/65536.0))
	return &r
}

func (v FontVariation) String() string {
	return "FontVariation('" + v.Axis + "', " + dart.DoubleToString(v.Value) + ")"
}

// putInt32 writes v in the host's (little-endian) byte order, as text.dart's
// _kFakeHostEndian.
func putInt32(b []byte, v int32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func getInt32(b []byte) int32 {
	return int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
}

// TextAlign is whether and how to align text horizontally.
type TextAlign int

const (
	// TextAlignLeft aligns the text on the left edge of the container.
	TextAlignLeft TextAlign = iota
	// TextAlignRight aligns the text on the right edge of the container.
	TextAlignRight
	// TextAlignCenter aligns the text in the center of the container.
	TextAlignCenter
	// TextAlignJustify stretches lines that end with a soft line break to
	// fill the width of the container.
	TextAlignJustify
	// TextAlignStart aligns the text on the leading edge of the container.
	TextAlignStart
	// TextAlignEnd aligns the text on the trailing edge of the container.
	TextAlignEnd
)

// TextAlignValues is TextAlign.values.
var TextAlignValues = []TextAlign{TextAlignLeft, TextAlignRight, TextAlignCenter, TextAlignJustify, TextAlignStart, TextAlignEnd}

// Name is the enum value's name.
func (a TextAlign) Name() string {
	return [...]string{"left", "right", "center", "justify", "start", "end"}[a]
}

func (a TextAlign) String() string { return "TextAlign." + a.Name() }

// Hyphens is how soft hyphens (U+00AD) are rendered.
type Hyphens int

const (
	// HyphensManual renders a hyphen where a line breaks at a soft hyphen.
	HyphensManual Hyphens = iota
	// HyphensHidden never renders soft hyphens.
	HyphensHidden
)

// HyphensValues is Hyphens.values.
var HyphensValues = []Hyphens{HyphensManual, HyphensHidden}

// Name is the enum value's name.
func (h Hyphens) Name() string { return [...]string{"manual", "hidden"}[h] }

func (h Hyphens) String() string { return "Hyphens." + h.Name() }

// TextBaseline is a horizontal line used for aligning text.
type TextBaseline int

const (
	// TextBaselineAlphabetic is the line the bottom of alphabetic glyphs
	// align with.
	TextBaselineAlphabetic TextBaseline = iota
	// TextBaselineIdeographic is the line ideographic glyphs align with.
	TextBaselineIdeographic
)

// TextBaselineValues is TextBaseline.values.
var TextBaselineValues = []TextBaseline{TextBaselineAlphabetic, TextBaselineIdeographic}

// Name is the enum value's name.
func (b TextBaseline) Name() string { return [...]string{"alphabetic", "ideographic"}[b] }

func (b TextBaseline) String() string { return "TextBaseline." + b.Name() }

// TextDecoration is a linear decoration to draw near the text (a mask).
// Compare with ==.
type TextDecoration struct {
	mask int
}

// TextDecorationCombine is TextDecoration.combine: a decoration with every
// line of the given ones.
func TextDecorationCombine(decorations []TextDecoration) TextDecoration {
	mask := 0
	for _, d := range decorations {
		mask |= d.mask
	}
	return TextDecoration{mask}
}

// MaskValue is the integer bitmask of the decoration's lines.
func (d TextDecoration) MaskValue() int { return d.mask }

// Contains reports whether this decoration has every line other has.
func (d TextDecoration) Contains(other TextDecoration) bool { return (d.mask | other.mask) == d.mask }

// The decorations (TextDecoration.none, underline, overline, lineThrough).
var (
	TextDecorationNone        = TextDecoration{0x0}
	TextDecorationUnderline   = TextDecoration{0x1}
	TextDecorationOverline    = TextDecoration{0x2}
	TextDecorationLineThrough = TextDecoration{0x4}
)

func (d TextDecoration) String() string {
	if d.mask == 0 {
		return "TextDecoration.none"
	}
	var values []string
	if d.mask&TextDecorationUnderline.mask != 0 {
		values = append(values, "underline")
	}
	if d.mask&TextDecorationOverline.mask != 0 {
		values = append(values, "overline")
	}
	if d.mask&TextDecorationLineThrough.mask != 0 {
		values = append(values, "lineThrough")
	}
	if len(values) == 1 {
		return "TextDecoration." + values[0]
	}
	return "TextDecoration.combine([" + strings.Join(values, ", ") + "])"
}

// TextDecorationStyle is the style in which to draw a text decoration.
type TextDecorationStyle int

const (
	// TextDecorationStyleSolid draws a solid line.
	TextDecorationStyleSolid TextDecorationStyle = iota
	// TextDecorationStyleDouble draws two lines.
	TextDecorationStyleDouble
	// TextDecorationStyleDotted draws a dotted line.
	TextDecorationStyleDotted
	// TextDecorationStyleDashed draws a dashed line.
	TextDecorationStyleDashed
	// TextDecorationStyleWavy draws a sinusoidal line.
	TextDecorationStyleWavy
)

// TextDecorationStyleValues is TextDecorationStyle.values.
var TextDecorationStyleValues = []TextDecorationStyle{TextDecorationStyleSolid, TextDecorationStyleDouble,
	TextDecorationStyleDotted, TextDecorationStyleDashed, TextDecorationStyleWavy}

// Name is the enum value's name.
func (s TextDecorationStyle) Name() string {
	return [...]string{"solid", "double", "dotted", "dashed", "wavy"}[s]
}

func (s TextDecorationStyle) String() string { return "TextDecorationStyle." + s.Name() }

// TextLeadingDistribution is how the "leading" (the extra space a line
// height adds) is distributed over and under the text.
type TextLeadingDistribution int

const (
	// TextLeadingDistributionProportional distributes the leading in
	// proportion to the font's ascent and descent.
	TextLeadingDistributionProportional TextLeadingDistribution = iota
	// TextLeadingDistributionEven distributes the leading evenly.
	TextLeadingDistributionEven
)

// TextLeadingDistributionValues is TextLeadingDistribution.values.
var TextLeadingDistributionValues = []TextLeadingDistribution{TextLeadingDistributionProportional, TextLeadingDistributionEven}

// Name is the enum value's name.
func (d TextLeadingDistribution) Name() string { return [...]string{"proportional", "even"}[d] }

func (d TextLeadingDistribution) String() string { return "TextLeadingDistribution." + d.Name() }

// TextHeightBehavior defines how a paragraph's height applies to the
// ascent of its first line and the descent of its last. Compare with ==.
type TextHeightBehavior struct {
	// ApplyHeightToFirstAscent: whether the height applies to the first
	// line's ascent.
	ApplyHeightToFirstAscent bool
	// ApplyHeightToLastDescent: whether the height applies to the last
	// line's descent.
	ApplyHeightToLastDescent bool
	// LeadingDistribution is how the leading is distributed.
	LeadingDistribution TextLeadingDistribution
}

// TextHeightBehaviorOptions are TextHeightBehavior's named arguments (nil:
// true).
type TextHeightBehaviorOptions struct {
	ApplyHeightToFirstAscent *bool
	ApplyHeightToLastDescent *bool
	LeadingDistribution      TextLeadingDistribution
}

// NewTextHeightBehavior is TextHeightBehavior({applyHeightToFirstAscent =
// true, applyHeightToLastDescent = true, leadingDistribution =
// proportional}).
func NewTextHeightBehavior(opts ...TextHeightBehaviorOptions) TextHeightBehavior {
	b := TextHeightBehavior{ApplyHeightToFirstAscent: true, ApplyHeightToLastDescent: true}
	if len(opts) > 0 {
		o := opts[0]
		if o.ApplyHeightToFirstAscent != nil {
			b.ApplyHeightToFirstAscent = *o.ApplyHeightToFirstAscent
		}
		if o.ApplyHeightToLastDescent != nil {
			b.ApplyHeightToLastDescent = *o.ApplyHeightToLastDescent
		}
		b.LeadingDistribution = o.LeadingDistribution
	}
	return b
}

// textHeightBehaviorFromEncoded is TextHeightBehavior._fromEncoded.
func textHeightBehaviorFromEncoded(encoded int32, leadingDistribution TextLeadingDistribution) TextHeightBehavior {
	return TextHeightBehavior{
		ApplyHeightToFirstAscent: encoded&0x1 == 0,
		ApplyHeightToLastDescent: encoded&0x2 == 0,
		LeadingDistribution:      leadingDistribution,
	}
}

// encode is TextHeightBehavior._encode.
func (b TextHeightBehavior) encode() int32 {
	var r int32
	if !b.ApplyHeightToFirstAscent {
		r |= 1 << 0
	}
	if !b.ApplyHeightToLastDescent {
		r |= 1 << 1
	}
	return r
}

func (b TextHeightBehavior) String() string {
	return "TextHeightBehavior(" +
		"applyHeightToFirstAscent: " + strconv.FormatBool(b.ApplyHeightToFirstAscent) + ", " +
		"applyHeightToLastDescent: " + strconv.FormatBool(b.ApplyHeightToLastDescent) + ", " +
		"leadingDistribution: " + b.LeadingDistribution.String() +
		")"
}

// textAssert is a Dart assert without a message (its AssertionError's
// message is null).
func textAssert(cond bool) {
	if !cond {
		panic(dart.NewAssertionError(nil))
	}
}
