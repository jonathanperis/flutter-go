package ui

// Port of engine/src/flutter/lib/ui/text.dart (Flutter 3.50.1), part 2:
// TextStyle, ParagraphStyle and StrutStyle with their encodings
// (_encodeTextStyle, _encodeParagraphStyle, _encodeStrut), which must match
// the engine's ParagraphBuilder (typography_bridge.cc).

import (
	"math"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// textListEquals is text.dart's _listEquals (nil is null).
func textListEquals[T comparable](a, b []T) bool {
	if a == nil {
		return b == nil
	}
	if b == nil || len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// listToString is Dart's List.toString.
func listToString[T any](l []T, str func(T) string) string {
	parts := make([]string, len(l))
	for i, x := range l {
		parts[i] = str(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func localeEqual(a, b *Locale) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func doubleEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// TextStyleOptions are TextStyle's named arguments (nil: null).
type TextStyleOptions struct {
	Color               *Color
	Decoration          *TextDecoration
	DecorationColor     *Color
	DecorationStyle     *TextDecorationStyle
	DecorationThickness *float64
	FontWeight          *FontWeight
	FontStyle           *FontStyle
	TextBaseline        *TextBaseline
	FontFamily          *string
	FontFamilyFallback  []string
	FontSize            *float64
	LetterSpacing       *float64
	WordSpacing         *float64
	Height              *float64
	LeadingDistribution *TextLeadingDistribution
	Locale              *Locale
	Background          *Paint
	Foreground          *Paint
	Shadows             []Shadow
	FontFeatures        []FontFeature
	FontVariations      []FontVariation
}

// TextStyle is an opaque object that determines the size, position and
// rendering of text (for ParagraphBuilder.PushStyle). Used by pointer:
// pushing a style records the paragraph's default leading distribution in
// it, as Dart's does.
type TextStyle struct {
	encoded             []int32
	fontFamily          string
	fontFamilyFallback  []string
	fontSize            *float64
	letterSpacing       *float64
	wordSpacing         *float64
	height              *float64
	decorationThickness *float64
	locale              *Locale
	background          *Paint
	foreground          *Paint
	shadows             []Shadow
	fontFeatures        []FontFeature
	fontVariations      []FontVariation
	leadingDistribution *TextLeadingDistribution
}

// encodeTextStyle is text.dart's _encodeTextStyle.
func encodeTextStyle(o TextStyleOptions) []int32 {
	result := make([]int32, 9)
	// The 0th bit of result[0] is reserved for leadingDistribution.
	if o.Color != nil {
		result[0] |= 1 << 1
		result[1] = int32(o.Color.Value())
	}
	if o.Decoration != nil {
		result[0] |= 1 << 2
		result[2] = int32(o.Decoration.mask)
	}
	if o.DecorationColor != nil {
		result[0] |= 1 << 3
		result[3] = int32(o.DecorationColor.Value())
	}
	if o.DecorationStyle != nil {
		result[0] |= 1 << 4
		result[4] = int32(*o.DecorationStyle)
	}
	if o.FontWeight != nil {
		result[0] |= 1 << 5
		result[5] = int32(o.FontWeight.Value)
	}
	if o.FontStyle != nil {
		result[0] |= 1 << 6
		result[6] = int32(*o.FontStyle)
	}
	if o.TextBaseline != nil {
		result[0] |= 1 << 7
		result[7] = int32(*o.TextBaseline)
	}
	if o.DecorationThickness != nil {
		result[0] |= 1 << 8
	}
	if o.FontFamily != nil || len(o.FontFamilyFallback) > 0 {
		result[0] |= 1 << 9
		// Passed separately to native.
	}
	if o.FontSize != nil {
		result[0] |= 1 << 10
	}
	if o.LetterSpacing != nil {
		result[0] |= 1 << 11
	}
	if o.WordSpacing != nil {
		result[0] |= 1 << 12
	}
	if o.Height != nil {
		result[0] |= 1 << 13
	}
	if o.Locale != nil {
		result[0] |= 1 << 14
	}
	if o.Background != nil {
		result[0] |= 1 << 15
	}
	if o.Foreground != nil {
		result[0] |= 1 << 16
	}
	if o.Shadows != nil {
		result[0] |= 1 << 17
	}
	if o.FontFeatures != nil {
		result[0] |= 1 << 18
	}
	if o.FontVariations != nil {
		result[0] |= 1 << 19
	}
	return result
}

// NewTextStyle is TextStyle({color, decoration, ...}). Color and
// Foreground can't both be given.
func NewTextStyle(opts ...TextStyleOptions) *TextStyle {
	var o TextStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.Color == nil || o.Foreground == nil,
		"Cannot provide both a color and a foreground\n"+
			`The color argument is just a shorthand for "foreground: Paint()..color = color".`)
	s := &TextStyle{
		encoded:             encodeTextStyle(o),
		fontFamilyFallback:  o.FontFamilyFallback,
		fontSize:            o.FontSize,
		letterSpacing:       o.LetterSpacing,
		wordSpacing:         o.WordSpacing,
		height:              o.Height,
		decorationThickness: o.DecorationThickness,
		locale:              o.Locale,
		background:          o.Background,
		foreground:          o.Foreground,
		shadows:             o.Shadows,
		fontFeatures:        o.FontFeatures,
		fontVariations:      o.FontVariations,
		leadingDistribution: o.LeadingDistribution,
	}
	if o.FontFamily != nil {
		s.fontFamily = *o.FontFamily
	}
	return s
}

// Equal is TextStyle's ==. Paints compare by identity (Paint has no ==).
func (s *TextStyle) Equal(other *TextStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return ((other.leadingDistribution == nil && s.leadingDistribution == nil) ||
		(other.leadingDistribution != nil && s.leadingDistribution != nil &&
			*other.leadingDistribution == *s.leadingDistribution)) &&
		other.fontFamily == s.fontFamily &&
		doubleEqual(other.fontSize, s.fontSize) &&
		doubleEqual(other.letterSpacing, s.letterSpacing) &&
		doubleEqual(other.wordSpacing, s.wordSpacing) &&
		doubleEqual(other.height, s.height) &&
		doubleEqual(other.decorationThickness, s.decorationThickness) &&
		localeEqual(other.locale, s.locale) &&
		other.background == s.background &&
		other.foreground == s.foreground &&
		textListEquals(other.encoded, s.encoded) &&
		textListEquals(other.shadows, s.shadows) &&
		textListEquals(other.fontFamilyFallback, s.fontFamilyFallback) &&
		textListEquals(other.fontFeatures, s.fontFeatures) &&
		textListEquals(other.fontVariations, s.fontVariations)
}

func (s *TextStyle) String() string {
	e := s.encoded
	unspecified := "unspecified"
	pick := func(bit int32, f func() string) string {
		if e[0]&bit == bit {
			return f()
		}
		return unspecified
	}
	heightText := unspecified
	if e[0]&0x02000 == 0x02000 {
		if *s.height == KTextHeightNone {
			heightText = "kTextHeightNone"
		} else {
			heightText = dart.DoubleToString(*s.height) + "x"
		}
	}
	fontFamily := unspecified
	if e[0]&0x00200 == 0x00200 && s.fontFamily != "" {
		fontFamily = s.fontFamily
	}
	fallback := unspecified
	if e[0]&0x00200 == 0x00200 && len(s.fontFamilyFallback) > 0 {
		fallback = listToString(s.fontFamilyFallback, func(x string) string { return x })
	}
	leading := unspecified
	if s.leadingDistribution != nil {
		leading = s.leadingDistribution.String()
	}
	return "TextStyle(" +
		"color: " + pick(0x00002, func() string { return NewColor(int(e[1])).String() }) + ", " +
		"decoration: " + pick(0x00004, func() string { return TextDecoration{int(e[2])}.String() }) + ", " +
		"decorationColor: " + pick(0x00008, func() string { return NewColor(int(e[3])).String() }) + ", " +
		"decorationStyle: " + pick(0x00010, func() string { return TextDecorationStyleValues[e[4]].String() }) + ", " +
		// The decorationThickness is not in encoded order in order to keep it
		// near the other decoration properties.
		"decorationThickness: " + pick(0x00100, func() string { return dart.DoubleToString(*s.decorationThickness) }) + ", " +
		"fontWeight: " + pick(0x00020, func() string { return NewFontWeight(int(e[5])).String() }) + ", " +
		"fontStyle: " + pick(0x00040, func() string { return FontStyleValues[e[6]].String() }) + ", " +
		"textBaseline: " + pick(0x00080, func() string { return TextBaselineValues[e[7]].String() }) + ", " +
		"fontFamily: " + fontFamily + ", " +
		"fontFamilyFallback: " + fallback + ", " +
		"fontSize: " + pick(0x00400, func() string { return dart.DoubleToString(*s.fontSize) }) + ", " +
		"letterSpacing: " + pick(0x00800, func() string { return dart.DoubleToString(*s.letterSpacing) + "x" }) + ", " +
		"wordSpacing: " + pick(0x01000, func() string { return dart.DoubleToString(*s.wordSpacing) + "x" }) + ", " +
		"height: " + heightText + ", " +
		"leadingDistribution: " + leading + ", " +
		"locale: " + pick(0x04000, func() string { return s.locale.String() }) + ", " +
		"background: " + pick(0x08000, func() string { return s.background.String() }) + ", " +
		"foreground: " + pick(0x10000, func() string { return s.foreground.String() }) + ", " +
		"shadows: " + pick(0x20000, func() string { return listToString(s.shadows, Shadow.String) }) + ", " +
		"fontFeatures: " + pick(0x40000, func() string { return listToString(s.fontFeatures, FontFeature.String) }) + ", " +
		"fontVariations: " + pick(0x80000, func() string { return listToString(s.fontVariations, FontVariation.String) }) +
		")"
}

// ParagraphStyleOptions are ParagraphStyle's named arguments (nil: null).
type ParagraphStyleOptions struct {
	TextAlign          *TextAlign
	TextDirection      *TextDirection
	MaxLines           *int
	FontFamily         *string
	FontSize           *float64
	Height             *float64
	TextHeightBehavior *TextHeightBehavior
	FontWeight         *FontWeight
	FontStyle          *FontStyle
	StrutStyle         *StrutStyle
	Ellipsis           *string
	Locale             *Locale
	Hyphens            *Hyphens
}

// ParagraphStyle is an opaque object that determines the configuration
// used by ParagraphBuilder to position lines within a paragraph.
type ParagraphStyle struct {
	encoded             []int32
	fontFamily          *string
	fontSize            *float64
	height              *float64
	strutStyle          *StrutStyle
	ellipsis            *string
	locale              *Locale
	leadingDistribution TextLeadingDistribution
}

// encodeParagraphStyle is text.dart's _encodeParagraphStyle.
func encodeParagraphStyle(o ParagraphStyleOptions) []int32 {
	result := make([]int32, 8) // also update paragraph_builder.cc
	if o.TextAlign != nil {
		result[0] |= 1 << 1
		result[1] = int32(*o.TextAlign)
	}
	if o.TextDirection != nil {
		result[0] |= 1 << 2
		result[2] = int32(*o.TextDirection)
	}
	if o.FontWeight != nil {
		result[0] |= 1 << 3
		result[3] = int32(o.FontWeight.Value)
	}
	if o.FontStyle != nil {
		result[0] |= 1 << 4
		result[4] = int32(*o.FontStyle)
	}
	if o.MaxLines != nil {
		result[0] |= 1 << 5
		result[5] = int32(*o.MaxLines)
	}
	if o.TextHeightBehavior != nil {
		result[0] |= 1 << 6
		result[6] = o.TextHeightBehavior.encode()
	}
	if o.FontFamily != nil {
		result[0] |= 1 << 7
		// Passed separately to native.
	}
	if o.FontSize != nil {
		result[0] |= 1 << 8
	}
	// Paragraph styles are unique in a paragraph, there is no inheriting so
	// height == null and height == kTextHeightNone are semantically
	// equivalent.
	if o.Height != nil && *o.Height != KTextHeightNone {
		result[0] |= 1 << 9
	}
	if o.StrutStyle != nil {
		result[0] |= 1 << 10
	}
	if o.Ellipsis != nil {
		result[0] |= 1 << 11
	}
	if o.Locale != nil {
		result[0] |= 1 << 12
	}
	if o.Hyphens != nil {
		result[0] |= 1 << 13
		result[7] = int32(*o.Hyphens)
	}
	return result
}

// NewParagraphStyle is ParagraphStyle({textAlign, textDirection, ...}).
func NewParagraphStyle(opts ...ParagraphStyleOptions) *ParagraphStyle {
	var o ParagraphStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := &ParagraphStyle{
		encoded:    encodeParagraphStyle(o),
		fontFamily: o.FontFamily,
		fontSize:   o.FontSize,
		height:     o.Height,
		strutStyle: o.StrutStyle,
		ellipsis:   o.Ellipsis,
		locale:     o.Locale,
	}
	if o.TextHeightBehavior != nil {
		s.leadingDistribution = o.TextHeightBehavior.LeadingDistribution
	}
	return s
}

func stringEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Equal is ParagraphStyle's ==.
func (s *ParagraphStyle) Equal(other *ParagraphStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return stringEqual(other.fontFamily, s.fontFamily) &&
		doubleEqual(other.fontSize, s.fontSize) &&
		doubleEqual(other.height, s.height) &&
		other.strutStyle.Equal(s.strutStyle) &&
		stringEqual(other.ellipsis, s.ellipsis) &&
		localeEqual(other.locale, s.locale) &&
		other.leadingDistribution == s.leadingDistribution &&
		textListEquals(other.encoded, s.encoded)
}

func nullableString(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func nullableDouble(d *float64) string {
	if d == nil {
		return "null"
	}
	return dart.DoubleToString(*d)
}

func (s *ParagraphStyle) String() string {
	e := s.encoded
	pick := func(bit int32, f func() string) string {
		if e[0]&bit == bit {
			return f()
		}
		return "unspecified"
	}
	return "ParagraphStyle(" +
		"textAlign: " + pick(0x002, func() string { return TextAlignValues[e[1]].String() }) + ", " +
		"textDirection: " + pick(0x004, func() string { return TextDirectionValues[e[2]].String() }) + ", " +
		"fontWeight: " + pick(0x008, func() string { return NewFontWeight(int(e[3])).String() }) + ", " +
		"fontStyle: " + pick(0x010, func() string { return FontStyleValues[e[4]].String() }) + ", " +
		"maxLines: " + pick(0x020, func() string { return strconv.Itoa(int(e[5])) }) + ", " +
		"textHeightBehavior: " + pick(0x040, func() string {
		return textHeightBehaviorFromEncoded(e[6], s.leadingDistribution).String()
	}) + ", " +
		"fontFamily: " + pick(0x080, func() string { return nullableString(s.fontFamily) }) + ", " +
		"fontSize: " + pick(0x100, func() string { return nullableDouble(s.fontSize) }) + ", " +
		"height: " + pick(0x200, func() string { return nullableDouble(s.height) + "x" }) + ", " +
		"strutStyle: " + pick(0x400, func() string { return s.strutStyle.String() }) + ", " +
		"ellipsis: " + pick(0x800, func() string { return `"` + nullableString(s.ellipsis) + `"` }) + ", " +
		"locale: " + pick(0x1000, func() string { return s.locale.String() }) + ", " +
		"hyphens: " + pick(0x2000, func() string { return HyphensValues[e[7]].String() }) +
		")"
}

// StrutStyleOptions are StrutStyle's named arguments (nil: null).
type StrutStyleOptions struct {
	FontFamily          *string
	FontFamilyFallback  []string
	FontSize            *float64
	Height              *float64
	LeadingDistribution *TextLeadingDistribution
	Leading             *float64
	FontWeight          *FontWeight
	FontStyle           *FontStyle
	ForceStrutHeight    *bool
}

// StrutStyle is the minimum line height of a paragraph's lines (the
// "strut"), as an opaque object for ParagraphStyle.
type StrutStyle struct {
	// encoded is _encoded: a view of the first encodedLength bytes of
	// buffer (Dart compares whole buffers).
	buffer              []byte
	encodedLength       int
	fontFamily          *string
	fontFamilyFallback  []string
	leadingDistribution *TextLeadingDistribution
}

// encodeStrut is text.dart's _encodeStrut: the buffer and the view's
// length.
func encodeStrut(o StrutStyleOptions) ([]byte, int) {
	// Strut styles are unique in a paragraph, there is no inheriting so
	// height == null and height == kTextHeightNone are semantically
	// equivalent.
	hasHeightOverride := o.Height != nil && *o.Height != KTextHeightNone
	if o.FontFamily == nil && o.FontSize == nil && !hasHeightOverride && o.LeadingDistribution == nil &&
		o.Leading == nil && o.FontWeight == nil && o.FontStyle == nil && o.ForceStrutHeight == nil {
		return []byte{}, 0
	}
	data := make([]byte, 24) // Max size is 24 bytes
	bitmask := int32(0)
	byteCount := 4
	if o.FontWeight != nil {
		bitmask |= 1 << 0
		putInt32(data[byteCount:], int32(o.FontWeight.Value))
		byteCount += 4
	}
	if o.FontStyle != nil {
		bitmask |= 1 << 1
		putInt32(data[byteCount:], int32(*o.FontStyle))
		byteCount += 4
	}
	if o.FontFamily != nil || len(o.FontFamilyFallback) > 0 {
		bitmask |= 1 << 2
		// passed separately to native
	}
	// The 3rd bit (0-indexed) is reserved for leadingDistribution.
	if o.FontSize != nil {
		bitmask |= 1 << 4
		putInt32(data[byteCount:], int32(math.Float32bits(float32(*o.FontSize))))
		byteCount += 4
	}
	if hasHeightOverride {
		bitmask |= 1 << 5
		putInt32(data[byteCount:], int32(math.Float32bits(float32(*o.Height))))
		byteCount += 4
	}
	if o.Leading != nil {
		bitmask |= 1 << 6
		putInt32(data[byteCount:], int32(math.Float32bits(float32(*o.Leading))))
		byteCount += 4
	}
	if o.ForceStrutHeight != nil && *o.ForceStrutHeight {
		bitmask |= 1 << 7
	}
	putInt32(data[0:], bitmask)
	return data, byteCount
}

// NewStrutStyle is StrutStyle({fontFamily, fontFamilyFallback, ...}).
func NewStrutStyle(opts ...StrutStyleOptions) *StrutStyle {
	var o StrutStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	buffer, n := encodeStrut(o)
	return &StrutStyle{
		buffer:              buffer,
		encodedLength:       n,
		fontFamily:          o.FontFamily,
		fontFamilyFallback:  o.FontFamilyFallback,
		leadingDistribution: o.LeadingDistribution,
	}
}

// enabled is StrutStyle._enabled.
func (s *StrutStyle) enabled() bool { return s.encodedLength > 0 }

// Equal is StrutStyle's == (nil is null).
func (s *StrutStyle) Equal(other *StrutStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return stringEqual(other.fontFamily, s.fontFamily) &&
		((other.leadingDistribution == nil && s.leadingDistribution == nil) ||
			(other.leadingDistribution != nil && s.leadingDistribution != nil &&
				*other.leadingDistribution == *s.leadingDistribution)) &&
		textListEquals(other.fontFamilyFallback, s.fontFamilyFallback) &&
		textListEquals(other.buffer, s.buffer)
}

// String is Object.toString (StrutStyle has no toString of its own).
func (s *StrutStyle) String() string {
	if s == nil {
		return "null"
	}
	return "Instance of 'StrutStyle'"
}
