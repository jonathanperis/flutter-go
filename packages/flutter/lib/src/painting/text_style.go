package painting

// Port of text_style.dart (Flutter 3.50.1): TextStyle (copyWith, apply,
// merge, lerp, getTextStyle, getParagraphStyle, compareTo, ==,
// debugFillProperties) and lerpFontVariations.

import (
	"strconv"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

const kDefaultDebugLabel = "unknown"

const kColorForegroundWarning = "Cannot provide both a color and a foreground\n" +
	`The color argument is just a shorthand for "foreground: Paint()..color = color".`

const kColorBackgroundWarning = "Cannot provide both a backgroundColor and a background\n" +
	`The backgroundColor argument is just a shorthand for "background: Paint()..color = color".`

const kTextStyleHeightNaNWarning = "TextStyle.height must not be NaN."

// TextStyle is an immutable style describing how to format and paint text.
// Used by pointer and built with NewTextStyle; its exported fields are
// Dart's (nil is null; a nil slice is a null list, an empty non-nil slice
// an empty one). Dart's `Paint` fields compare by identity.
type TextStyle struct {
	foundation.DiagnosticableMixin

	// Inherit is whether null values in this style are replaced with the
	// values of the ancestor text style (merge).
	Inherit bool
	// Color is the color to use when painting the text (nil: none).
	Color *ui.Color
	// BackgroundColor is the color to use as the background for the text.
	BackgroundColor *ui.Color
	// FontFamily is the name of the font to use, with the package prefix
	// ("packages/<package>/<family>") when a package was given.
	FontFamily *string
	// FontSize is the size of glyphs (in logical pixels).
	FontSize *float64
	// FontWeight is the typeface thickness.
	FontWeight *ui.FontWeight
	// FontStyle is the typeface variant (italic or not).
	FontStyle *ui.FontStyle
	// LetterSpacing is the amount of space to add between each letter.
	LetterSpacing *float64
	// WordSpacing is the amount of space to add at each sequence of
	// white-space.
	WordSpacing *float64
	// TextBaseline is the common baseline used for alignment.
	TextBaseline *ui.TextBaseline
	// Height is the height of this text span, as a multiple of the font
	// size (ui.KTextHeightNone: the font's own metrics).
	Height *float64
	// LeadingDistribution is how the vertical space added by Height is
	// distributed.
	LeadingDistribution *ui.TextLeadingDistribution
	// Locale is the locale used to select region-specific glyphs.
	Locale *ui.Locale
	// Foreground is the paint drawn as a foreground for the text.
	Foreground *ui.Paint
	// Background is the paint drawn as a background for the text.
	Background *ui.Paint
	// Decoration is the decorations to paint near the text.
	Decoration *ui.TextDecoration
	// DecorationColor is the color in which to paint the decorations.
	DecorationColor *ui.Color
	// DecorationStyle is the style in which to paint the decorations.
	DecorationStyle *ui.TextDecorationStyle
	// DecorationThickness is the thickness of the decoration stroke as a
	// multiplier of the font's thickness.
	DecorationThickness *float64
	// DebugLabel is a human-readable description of this style.
	DebugLabel *string
	// Shadows is the list of shadows painted underneath the text.
	Shadows []ui.Shadow
	// FontFeatures is the list of font features to use.
	FontFeatures []ui.FontFeature
	// FontVariations is the list of font variations to use.
	FontVariations []ui.FontVariation
	// Overflow is how visual text overflow should be handled.
	Overflow *TextOverflow

	fontFamilyFallback []string
	pkg                *string
}

// TextStyleOptions are TextStyle's named parameters (nil: Dart's default;
// Inherit nil is true).
type TextStyleOptions struct {
	Inherit             *bool
	Color               *ui.Color
	BackgroundColor     *ui.Color
	FontSize            *float64
	FontWeight          *ui.FontWeight
	FontStyle           *ui.FontStyle
	LetterSpacing       *float64
	WordSpacing         *float64
	TextBaseline        *ui.TextBaseline
	Height              *float64
	LeadingDistribution *ui.TextLeadingDistribution
	Locale              *ui.Locale
	Foreground          *ui.Paint
	Background          *ui.Paint
	Shadows             []ui.Shadow
	FontFeatures        []ui.FontFeature
	FontVariations      []ui.FontVariation
	Decoration          *ui.TextDecoration
	DecorationColor     *ui.Color
	DecorationStyle     *ui.TextDecorationStyle
	DecorationThickness *float64
	DebugLabel          *string
	FontFamily          *string
	FontFamilyFallback  []string
	Package             *string
	Overflow            *TextOverflow
}

func packagePrefixed(pkg, family *string) *string {
	if pkg == nil || family == nil {
		if pkg != nil {
			// 'packages/$package/$fontFamily' with a null family.
			s := "packages/" + *pkg + "/null"
			return &s
		}
		return family
	}
	s := "packages/" + *pkg + "/" + *family
	return &s
}

// NewTextStyle is TextStyle({...}). It panics as Dart's constructor
// asserts: a color with a foreground, a backgroundColor with a background,
// or a NaN height.
func NewTextStyle(opts ...TextStyleOptions) *TextStyle {
	var o TextStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Color != nil && o.Foreground != nil {
		panic(dart.NewAssertionError(kColorForegroundWarning))
	}
	if o.BackgroundColor != nil && o.Background != nil {
		panic(dart.NewAssertionError(kColorBackgroundWarning))
	}
	if o.Height != nil && *o.Height != *o.Height {
		panic(dart.NewAssertionError(kTextStyleHeightNaNWarning))
	}
	s := &TextStyle{
		Inherit:             o.Inherit == nil || *o.Inherit,
		Color:               o.Color,
		BackgroundColor:     o.BackgroundColor,
		FontSize:            o.FontSize,
		FontWeight:          o.FontWeight,
		FontStyle:           o.FontStyle,
		LetterSpacing:       o.LetterSpacing,
		WordSpacing:         o.WordSpacing,
		TextBaseline:        o.TextBaseline,
		Height:              o.Height,
		LeadingDistribution: o.LeadingDistribution,
		Locale:              o.Locale,
		Foreground:          o.Foreground,
		Background:          o.Background,
		Shadows:             o.Shadows,
		FontFeatures:        o.FontFeatures,
		FontVariations:      o.FontVariations,
		Decoration:          o.Decoration,
		DecorationColor:     o.DecorationColor,
		DecorationStyle:     o.DecorationStyle,
		DecorationThickness: o.DecorationThickness,
		DebugLabel:          o.DebugLabel,
		FontFamily:          packagePrefixed(o.Package, o.FontFamily),
		fontFamilyFallback:  o.FontFamilyFallback,
		pkg:                 o.Package,
		Overflow:            o.Overflow,
	}
	s.SetSelf(s)
	return s
}

// FontFamilyFallback is the ordered list of font families to fall back on
// when a glyph cannot be found in FontFamily (prefixed with the package's
// path when a package was given).
func (s *TextStyle) FontFamilyFallback() []string {
	if s.pkg == nil || s.fontFamilyFallback == nil {
		return s.fontFamilyFallback
	}
	out := make([]string, len(s.fontFamilyFallback))
	for i, f := range s.fontFamilyFallback {
		out[i] = "packages/" + *s.pkg + "/" + f
	}
	return out
}

// rawFontFamily is Dart's _fontFamily: the family without the package
// prefix.
func (s *TextStyle) rawFontFamily() *string {
	if s.pkg != nil {
		prefix := "packages/" + *s.pkg + "/"
		if s.FontFamily == nil {
			return nil
		}
		if !strings.HasPrefix(*s.FontFamily, prefix) {
			panic(dart.NewAssertionError(nil))
		}
		f := dart.Substring(*s.FontFamily, dart.StringLength(prefix))
		return &f
	}
	return s.FontFamily
}

// TextStyleCopyWithOptions are TextStyle.copyWith's named parameters (nil:
// keep this style's value).
type TextStyleCopyWithOptions = TextStyleOptions

func or[T any](a, b *T) *T {
	if a != nil {
		return a
	}
	return b
}

func orList[T any](a, b []T) []T {
	if a != nil {
		return a
	}
	return b
}

func strPtr(s string) *string { return &s }

// CopyWith is TextStyle.copyWith: a copy of this style with the given
// fields replaced.
func (s *TextStyle) CopyWith(opts ...TextStyleCopyWithOptions) *TextStyle {
	var o TextStyleCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Color != nil && o.Foreground != nil {
		panic(dart.NewAssertionError(kColorForegroundWarning))
	}
	if o.BackgroundColor != nil && o.Background != nil {
		panic(dart.NewAssertionError(kColorBackgroundWarning))
	}
	var newDebugLabel *string
	if foundation.KDebugMode {
		if o.DebugLabel != nil {
			newDebugLabel = o.DebugLabel
		} else if s.DebugLabel != nil {
			newDebugLabel = strPtr("(" + *s.DebugLabel + ").copyWith")
		}
	}
	inherit := s.Inherit
	if o.Inherit != nil {
		inherit = *o.Inherit
	}
	var color, backgroundColor *ui.Color
	if s.Foreground == nil && o.Foreground == nil {
		color = or(o.Color, s.Color)
	}
	if s.Background == nil && o.Background == nil {
		backgroundColor = or(o.BackgroundColor, s.BackgroundColor)
	}
	return NewTextStyle(TextStyleOptions{
		Inherit:             &inherit,
		Color:               color,
		BackgroundColor:     backgroundColor,
		FontSize:            or(o.FontSize, s.FontSize),
		FontWeight:          or(o.FontWeight, s.FontWeight),
		FontStyle:           or(o.FontStyle, s.FontStyle),
		LetterSpacing:       or(o.LetterSpacing, s.LetterSpacing),
		WordSpacing:         or(o.WordSpacing, s.WordSpacing),
		TextBaseline:        or(o.TextBaseline, s.TextBaseline),
		Height:              or(o.Height, s.Height),
		LeadingDistribution: or(o.LeadingDistribution, s.LeadingDistribution),
		Locale:              or(o.Locale, s.Locale),
		Foreground:          or(o.Foreground, s.Foreground),
		Background:          or(o.Background, s.Background),
		Shadows:             orList(o.Shadows, s.Shadows),
		FontFeatures:        orList(o.FontFeatures, s.FontFeatures),
		FontVariations:      orList(o.FontVariations, s.FontVariations),
		Decoration:          or(o.Decoration, s.Decoration),
		DecorationColor:     or(o.DecorationColor, s.DecorationColor),
		DecorationStyle:     or(o.DecorationStyle, s.DecorationStyle),
		DecorationThickness: or(o.DecorationThickness, s.DecorationThickness),
		DebugLabel:          newDebugLabel,
		FontFamily:          or(o.FontFamily, s.rawFontFamily()),
		FontFamilyFallback:  orList(o.FontFamilyFallback, s.fontFamilyFallback),
		Package:             or(o.Package, s.pkg),
		Overflow:            or(o.Overflow, s.Overflow),
	})
}

// TextStyleApplyOptions are TextStyle.apply's named parameters. The
// factors (nil: 1.0) and deltas (zero: 0) modify the corresponding
// numeric fields.
type TextStyleApplyOptions struct {
	Color                     *ui.Color
	BackgroundColor           *ui.Color
	Decoration                *ui.TextDecoration
	DecorationColor           *ui.Color
	DecorationStyle           *ui.TextDecorationStyle
	DecorationThicknessFactor *float64
	DecorationThicknessDelta  float64
	FontFamily                *string
	FontFamilyFallback        []string
	FontSizeFactor            *float64
	FontSizeDelta             float64
	FontWeightDelta           int
	FontStyle                 *ui.FontStyle
	LetterSpacingFactor       *float64
	LetterSpacingDelta        float64
	WordSpacingFactor         *float64
	WordSpacingDelta          float64
	HeightFactor              *float64
	HeightDelta               float64
	TextBaseline              *ui.TextBaseline
	LeadingDistribution       *ui.TextLeadingDistribution
	Locale                    *ui.Locale
	Shadows                   []ui.Shadow
	FontFeatures              []ui.FontFeature
	FontVariations            []ui.FontVariation
	Package                   *string
	Overflow                  *TextOverflow
}

func factorOr(f *float64) float64 {
	if f == nil {
		return 1.0
	}
	return *f
}

// applyNum is `x == null ? null : x * factor + delta`, with Dart's two
// roundings.
func applyNum(x *float64, factor, delta float64) *float64 {
	if x == nil {
		return nil
	}
	v := float64(*x*factor) + delta
	return &v
}

// Apply is TextStyle.apply: a copy of this style with the given values
// replaced or modified by the factors and deltas. It panics (Dart's
// asserts) when a factor or delta is given for a null field.
func (s *TextStyle) Apply(opts ...TextStyleApplyOptions) *TextStyle {
	var o TextStyleApplyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	fontSizeFactor, letterSpacingFactor := factorOr(o.FontSizeFactor), factorOr(o.LetterSpacingFactor)
	wordSpacingFactor, heightFactor := factorOr(o.WordSpacingFactor), factorOr(o.HeightFactor)
	decorationThicknessFactor := factorOr(o.DecorationThicknessFactor)
	if !(s.FontSize != nil || (fontSizeFactor == 1.0 && o.FontSizeDelta == 0.0)) {
		panic(dart.NewAssertionError(nil))
	}
	if !(s.FontWeight != nil || o.FontWeightDelta == 0.0) {
		panic(dart.NewAssertionError(nil))
	}
	if !(s.LetterSpacing != nil || (letterSpacingFactor == 1.0 && o.LetterSpacingDelta == 0.0)) {
		panic(dart.NewAssertionError(nil))
	}
	if !(s.WordSpacing != nil || (wordSpacingFactor == 1.0 && o.WordSpacingDelta == 0.0)) {
		panic(dart.NewAssertionError(nil))
	}
	if !(s.DecorationThickness != nil || (decorationThicknessFactor == 1.0 && o.DecorationThicknessDelta == 0.0)) {
		panic(dart.NewAssertionError(nil))
	}
	var modifiedDebugLabel *string
	if foundation.KDebugMode && s.DebugLabel != nil {
		modifiedDebugLabel = strPtr("(" + *s.DebugLabel + ").apply")
	}
	var color, backgroundColor *ui.Color
	if s.Foreground == nil {
		color = or(o.Color, s.Color)
	}
	if s.Background == nil {
		backgroundColor = or(o.BackgroundColor, s.BackgroundColor)
	}
	var fontWeight *ui.FontWeight
	if s.FontWeight != nil {
		w := ui.FontWeightValues[min(max(s.FontWeight.Index()+o.FontWeightDelta, 0), len(ui.FontWeightValues)-1)]
		fontWeight = &w
	}
	height := s.Height
	if s.Height != nil && *s.Height != ui.KTextHeightNone {
		height = applyNum(s.Height, heightFactor, o.HeightDelta)
	}
	inherit := s.Inherit
	return NewTextStyle(TextStyleOptions{
		Inherit:             &inherit,
		Color:               color,
		BackgroundColor:     backgroundColor,
		FontFamily:          or(o.FontFamily, s.rawFontFamily()),
		FontFamilyFallback:  orList(o.FontFamilyFallback, s.fontFamilyFallback),
		FontSize:            applyNum(s.FontSize, fontSizeFactor, o.FontSizeDelta),
		FontWeight:          fontWeight,
		FontStyle:           or(o.FontStyle, s.FontStyle),
		LetterSpacing:       applyNum(s.LetterSpacing, letterSpacingFactor, o.LetterSpacingDelta),
		WordSpacing:         applyNum(s.WordSpacing, wordSpacingFactor, o.WordSpacingDelta),
		TextBaseline:        or(o.TextBaseline, s.TextBaseline),
		Height:              height,
		LeadingDistribution: or(o.LeadingDistribution, s.LeadingDistribution),
		Locale:              or(o.Locale, s.Locale),
		Foreground:          s.Foreground,
		Background:          s.Background,
		Shadows:             orList(o.Shadows, s.Shadows),
		FontFeatures:        orList(o.FontFeatures, s.FontFeatures),
		FontVariations:      orList(o.FontVariations, s.FontVariations),
		Decoration:          or(o.Decoration, s.Decoration),
		DecorationColor:     or(o.DecorationColor, s.DecorationColor),
		DecorationStyle:     or(o.DecorationStyle, s.DecorationStyle),
		DecorationThickness: applyNum(s.DecorationThickness, decorationThicknessFactor, o.DecorationThicknessDelta),
		Overflow:            or(o.Overflow, s.Overflow),
		Package:             or(o.Package, s.pkg),
		DebugLabel:          modifiedDebugLabel,
	})
}

// Merge is TextStyle.merge: this style with the non-null values of other
// (other itself when it doesn't inherit; s when other is nil).
func (s *TextStyle) Merge(other *TextStyle) *TextStyle {
	if other == nil {
		return s
	}
	if !other.Inherit {
		return other
	}
	var mergedDebugLabel *string
	if foundation.KDebugMode && (other.DebugLabel != nil || s.DebugLabel != nil) {
		mergedDebugLabel = strPtr("(" + strOr(s.DebugLabel, kDefaultDebugLabel) + ").merge(" +
			strOr(other.DebugLabel, kDefaultDebugLabel) + ")")
	}
	return s.CopyWith(TextStyleCopyWithOptions{
		Color:               other.Color,
		BackgroundColor:     other.BackgroundColor,
		FontSize:            other.FontSize,
		FontWeight:          other.FontWeight,
		FontStyle:           other.FontStyle,
		LetterSpacing:       other.LetterSpacing,
		WordSpacing:         other.WordSpacing,
		TextBaseline:        other.TextBaseline,
		Height:              other.Height,
		LeadingDistribution: other.LeadingDistribution,
		Locale:              other.Locale,
		Foreground:          other.Foreground,
		Background:          other.Background,
		Shadows:             other.Shadows,
		FontFeatures:        other.FontFeatures,
		FontVariations:      other.FontVariations,
		Decoration:          other.Decoration,
		DecorationColor:     other.DecorationColor,
		DecorationStyle:     other.DecorationStyle,
		DecorationThickness: other.DecorationThickness,
		DebugLabel:          mergedDebugLabel,
		FontFamily:          other.rawFontFamily(),
		FontFamilyFallback:  other.fontFamilyFallback,
		Package:             other.pkg,
		Overflow:            other.Overflow,
	})
}

func strOr(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// step is `t < 0.5 ? a : b` on nullable values.
func step[T any](t float64, a, b *T) *T {
	if t < 0.5 {
		return a
	}
	return b
}

func stepList[T any](t float64, a, b []T) []T {
	if t < 0.5 {
		return a
	}
	return b
}

func colorPaintOf(c *ui.Color) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(*c) // Dart's `color!` throws on null.
	return p
}

// TextStyleLerp is TextStyle.lerp: interpolates between two text styles
// (nil is null). It panics with Flutter's FlutterError when the styles'
// inherit values differ and both leave a lerpable field unspecified.
func TextStyleLerp(a, b *TextStyle, t float64) *TextStyle {
	if a == b {
		return a
	}
	var lerpDebugLabel *string
	if foundation.KDebugMode {
		var la, lb *string
		if a != nil {
			la = a.DebugLabel
		}
		if b != nil {
			lb = b.DebugLabel
		}
		lerpDebugLabel = strPtr("lerp(" + strOr(la, kDefaultDebugLabel) + " ⎯" + dart.ToStringAsFixed(t, 1) +
			"→ " + strOr(lb, kDefaultDebugLabel) + ")")
	}

	if a == nil {
		inherit := b.Inherit
		return NewTextStyle(TextStyleOptions{
			Inherit:             &inherit,
			Color:               ui.ColorLerp(nil, b.Color, t),
			BackgroundColor:     ui.ColorLerp(nil, b.BackgroundColor, t),
			FontSize:            step(t, nil, b.FontSize),
			FontWeight:          ui.FontWeightLerp(nil, b.FontWeight, t),
			FontStyle:           step(t, nil, b.FontStyle),
			LetterSpacing:       step(t, nil, b.LetterSpacing),
			WordSpacing:         step(t, nil, b.WordSpacing),
			TextBaseline:        step(t, nil, b.TextBaseline),
			Height:              step(t, nil, b.Height),
			LeadingDistribution: step(t, nil, b.LeadingDistribution),
			Locale:              step(t, nil, b.Locale),
			Foreground:          step(t, nil, b.Foreground),
			Background:          step(t, nil, b.Background),
			Shadows:             stepList(t, nil, b.Shadows),
			FontFeatures:        stepList(t, nil, b.FontFeatures),
			FontVariations:      LerpFontVariations(nil, b.FontVariations, t),
			Decoration:          step(t, nil, b.Decoration),
			DecorationColor:     ui.ColorLerp(nil, b.DecorationColor, t),
			DecorationStyle:     step(t, nil, b.DecorationStyle),
			DecorationThickness: step(t, nil, b.DecorationThickness),
			DebugLabel:          lerpDebugLabel,
			FontFamily:          step(t, nil, b.rawFontFamily()),
			FontFamilyFallback:  stepList(t, nil, b.fontFamilyFallback),
			Package:             step(t, nil, b.pkg),
			Overflow:            step(t, nil, b.Overflow),
		})
	}

	if b == nil {
		inherit := a.Inherit
		return NewTextStyle(TextStyleOptions{
			Inherit: &inherit,
			Color:   ui.ColorLerp(a.Color, nil, t),
			// Flutter's quirk: the background color lerps from null.
			BackgroundColor:     ui.ColorLerp(nil, a.BackgroundColor, t),
			FontSize:            step(t, a.FontSize, nil),
			FontWeight:          ui.FontWeightLerp(a.FontWeight, nil, t),
			FontStyle:           step(t, a.FontStyle, nil),
			LetterSpacing:       step(t, a.LetterSpacing, nil),
			WordSpacing:         step(t, a.WordSpacing, nil),
			TextBaseline:        step(t, a.TextBaseline, nil),
			Height:              step(t, a.Height, nil),
			LeadingDistribution: step(t, a.LeadingDistribution, nil),
			Locale:              step(t, a.Locale, nil),
			Foreground:          step(t, a.Foreground, nil),
			Background:          step(t, a.Background, nil),
			Shadows:             stepList(t, a.Shadows, nil),
			FontFeatures:        stepList(t, a.FontFeatures, nil),
			FontVariations:      LerpFontVariations(a.FontVariations, nil, t),
			Decoration:          step(t, a.Decoration, nil),
			DecorationColor:     ui.ColorLerp(a.DecorationColor, nil, t),
			DecorationStyle:     step(t, a.DecorationStyle, nil),
			DecorationThickness: step(t, a.DecorationThickness, nil),
			DebugLabel:          lerpDebugLabel,
			FontFamily:          step(t, a.rawFontFamily(), nil),
			FontFamilyFallback:  stepList(t, a.fontFamilyFallback, nil),
			Package:             step(t, a.pkg, nil),
			Overflow:            step(t, a.Overflow, nil),
		})
	}

	if a.Inherit != b.Inherit {
		var nullFields []string
		if a.Foreground == nil && b.Foreground == nil && a.Color == nil && b.Color == nil {
			nullFields = append(nullFields, "color")
		}
		if a.Background == nil && b.Background == nil && a.BackgroundColor == nil && b.BackgroundColor == nil {
			nullFields = append(nullFields, "backgroundColor")
		}
		if a.FontSize == nil && b.FontSize == nil {
			nullFields = append(nullFields, "fontSize")
		}
		if a.LetterSpacing == nil && b.LetterSpacing == nil {
			nullFields = append(nullFields, "letterSpacing")
		}
		if a.WordSpacing == nil && b.WordSpacing == nil {
			nullFields = append(nullFields, "wordSpacing")
		}
		if a.Height == nil && b.Height == nil {
			nullFields = append(nullFields, "height")
		}
		if a.DecorationColor == nil && b.DecorationColor == nil {
			nullFields = append(nullFields, "decorationColor")
		}
		if a.DecorationThickness == nil && b.DecorationThickness == nil {
			nullFields = append(nullFields, "decorationThickness")
		}
		if len(nullFields) > 0 {
			quoted := make([]string, len(nullFields))
			for i, n := range nullFields {
				quoted[i] = `"` + n + `"`
			}
			singleLine := foundation.DiagnosticsTreeStyleSingleLine
			from, to := "from", "to"
			panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
				foundation.NewErrorSummary("Failed to interpolate TextStyles with different inherit values."),
				foundation.NewErrorSpacer(),
				foundation.NewErrorDescription("The TextStyles being interpolated were:"),
				a.ToDiagnosticsNode(foundation.ToDiagnosticsNodeOptions{Name: &from, Style: &singleLine}),
				b.ToDiagnosticsNode(foundation.ToDiagnosticsNodeOptions{Name: &to, Style: &singleLine}),
				foundation.NewErrorDescription("The following fields are unspecified in both TextStyles:\n" +
					strings.Join(quoted, ", ") + ".\n" +
					"When \"inherit\" changes during the transition, these fields may " +
					"observe abrupt value changes as a result, causing \"jump\"s in the " +
					"transition."),
				foundation.NewErrorSpacer(),
				foundation.NewErrorHint("In general, TextStyle.lerp only works well when both TextStyles have " +
					"the same \"inherit\" value, and specify the same fields."),
				foundation.NewErrorHint("If the TextStyles were directly created by you, consider bringing " +
					"them to parity to ensure a smooth transition."),
				foundation.NewErrorSpacer(),
				foundation.NewErrorHint("If one of the TextStyles being lerped is significantly more elaborate " +
					"than the other, and has \"inherited\" set to false, it is often because " +
					"it is merged with another TextStyle before being lerped. Comparing " +
					"the \"debugLabel\"s of the two TextStyles may help identify if that was " +
					"the case."),
				foundation.NewErrorHint("For example, you may see this error message when trying to lerp " +
					"between \"ThemeData()\" and \"Theme.of(context)\". This is because " +
					"TextStyles from \"Theme.of(context)\" are merged with TextStyles from " +
					"another theme and thus are more elaborate than the TextStyles from " +
					"\"ThemeData()\" (which is reflected in their \"debugLabel\"s -- " +
					"TextStyles from \"Theme.of(context)\" should have labels in the form of " +
					"\"(<A TextStyle>).merge(<Another TextStyle>)\"). It is recommended to " +
					"only lerp ThemeData with matching TextStyles."),
			}))
		}
	}

	inherit := b.Inherit
	if t < 0.5 {
		inherit = a.Inherit
	}
	var color, backgroundColor *ui.Color
	if a.Foreground == nil && b.Foreground == nil {
		color = ui.ColorLerp(a.Color, b.Color, t)
	}
	if a.Background == nil && b.Background == nil {
		backgroundColor = ui.ColorLerp(a.BackgroundColor, b.BackgroundColor, t)
	}
	var foreground, background *ui.Paint
	if a.Foreground != nil || b.Foreground != nil {
		if t < 0.5 {
			foreground = a.Foreground
			if foreground == nil {
				foreground = colorPaintOf(a.Color)
			}
		} else {
			foreground = b.Foreground
			if foreground == nil {
				foreground = colorPaintOf(b.Color)
			}
		}
	}
	if a.Background != nil || b.Background != nil {
		if t < 0.5 {
			background = a.Background
			if background == nil {
				background = colorPaintOf(a.BackgroundColor)
			}
		} else {
			background = b.Background
			if background == nil {
				background = colorPaintOf(b.BackgroundColor)
			}
		}
	}
	return NewTextStyle(TextStyleOptions{
		Inherit:             &inherit,
		Color:               color,
		BackgroundColor:     backgroundColor,
		FontSize:            ui.LerpDouble(or(a.FontSize, b.FontSize), or(b.FontSize, a.FontSize), t),
		FontWeight:          ui.FontWeightLerp(a.FontWeight, b.FontWeight, t),
		FontStyle:           step(t, a.FontStyle, b.FontStyle),
		LetterSpacing:       ui.LerpDouble(or(a.LetterSpacing, b.LetterSpacing), or(b.LetterSpacing, a.LetterSpacing), t),
		WordSpacing:         ui.LerpDouble(or(a.WordSpacing, b.WordSpacing), or(b.WordSpacing, a.WordSpacing), t),
		TextBaseline:        step(t, a.TextBaseline, b.TextBaseline),
		Height:              ui.LerpDouble(or(a.Height, b.Height), or(b.Height, a.Height), t),
		LeadingDistribution: step(t, a.LeadingDistribution, b.LeadingDistribution),
		Locale:              step(t, a.Locale, b.Locale),
		Foreground:          foreground,
		Background:          background,
		Shadows:             ui.ShadowLerpList(a.Shadows, b.Shadows, t),
		FontFeatures:        stepList(t, a.FontFeatures, b.FontFeatures),
		FontVariations:      LerpFontVariations(a.FontVariations, b.FontVariations, t),
		Decoration:          step(t, a.Decoration, b.Decoration),
		DecorationColor:     ui.ColorLerp(a.DecorationColor, b.DecorationColor, t),
		DecorationStyle:     step(t, a.DecorationStyle, b.DecorationStyle),
		DecorationThickness: ui.LerpDouble(or(a.DecorationThickness, b.DecorationThickness),
			or(b.DecorationThickness, a.DecorationThickness), t),
		DebugLabel:         lerpDebugLabel,
		FontFamily:         step(t, a.rawFontFamily(), b.rawFontFamily()),
		FontFamilyFallback: stepList(t, a.fontFamilyFallback, b.fontFamilyFallback),
		Package:            step(t, a.pkg, b.pkg),
		Overflow:           step(t, a.Overflow, b.Overflow),
	})
}

// TextStyleGetTextStyleOptions are TextStyle.getTextStyle's named
// parameters: the deprecated TextScaleFactor (nil: 1.0) and TextScaler
// (nil: TextScalerNoScaling).
type TextStyleGetTextStyleOptions struct {
	TextScaleFactor *float64
	TextScaler      TextScaler
}

// GetTextStyle is TextStyle.getTextStyle: the style as a dart:ui
// TextStyle, its font size scaled.
func (s *TextStyle) GetTextStyle(opts ...TextStyleGetTextStyleOptions) *ui.TextStyle {
	var o TextStyleGetTextStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	textScaleFactor := factorOr(o.TextScaleFactor)
	textScaler := o.TextScaler
	if textScaler == nil {
		textScaler = TextScalerNoScaling
	}
	if !(textScaler == TextScalerNoScaling || textScaleFactor == 1.0) {
		panic(dart.NewAssertionError("textScaleFactor is deprecated and cannot be specified when textScaler is specified."))
	}
	var fontSize *float64
	if s.FontSize != nil {
		var v float64
		if textScaler == TextScalerNoScaling {
			v = float64(*s.FontSize * textScaleFactor)
		} else {
			v = textScaler.Scale(*s.FontSize)
		}
		fontSize = &v
	}
	background := s.Background
	if background == nil && s.BackgroundColor != nil {
		background = colorPaintOf(s.BackgroundColor)
	}
	return ui.NewTextStyle(ui.TextStyleOptions{
		Color:               s.Color,
		Decoration:          s.Decoration,
		DecorationColor:     s.DecorationColor,
		DecorationStyle:     s.DecorationStyle,
		DecorationThickness: s.DecorationThickness,
		FontWeight:          s.FontWeight,
		FontStyle:           s.FontStyle,
		TextBaseline:        s.TextBaseline,
		LeadingDistribution: s.LeadingDistribution,
		FontFamily:          s.FontFamily,
		FontFamilyFallback:  s.FontFamilyFallback(),
		FontSize:            fontSize,
		LetterSpacing:       s.LetterSpacing,
		WordSpacing:         s.WordSpacing,
		Height:              s.Height,
		Locale:              s.Locale,
		Foreground:          s.Foreground,
		Background:          background,
		Shadows:             s.Shadows,
		FontFeatures:        s.FontFeatures,
		FontVariations:      s.FontVariations,
	})
}

// TextStyleGetParagraphStyleOptions are TextStyle.getParagraphStyle's
// named parameters (nil: null; TextScaler nil: TextScalerNoScaling).
type TextStyleGetParagraphStyleOptions struct {
	TextAlign          *ui.TextAlign
	TextDirection      *ui.TextDirection
	TextScaler         TextScaler
	Ellipsis           *string
	MaxLines           *int
	TextHeightBehavior *ui.TextHeightBehavior
	Hyphens            *ui.Hyphens
	Locale             *ui.Locale
	FontFamily         *string
	FontSize           *float64
	FontWeight         *ui.FontWeight
	FontStyle          *ui.FontStyle
	Height             *float64
	StrutStyle         *StrutStyle
}

// GetParagraphStyle is TextStyle.getParagraphStyle: the dart:ui
// ParagraphStyle that uses this style as the paragraph's default font
// unless overridden.
func (s *TextStyle) GetParagraphStyle(opts ...TextStyleGetParagraphStyleOptions) *ui.ParagraphStyle {
	var o TextStyleGetParagraphStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.MaxLines != nil && !(*o.MaxLines > 0) {
		panic(dart.NewAssertionError(nil))
	}
	if o.Height != nil && *o.Height != *o.Height {
		panic(dart.NewAssertionError(kTextStyleHeightNaNWarning))
	}
	textScaler := o.TextScaler
	if textScaler == nil {
		textScaler = TextScalerNoScaling
	}
	effectiveTextHeightBehavior := o.TextHeightBehavior
	if effectiveTextHeightBehavior == nil && s.LeadingDistribution != nil {
		thb := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{LeadingDistribution: *s.LeadingDistribution})
		effectiveTextHeightBehavior = &thb
	}
	fontSize := KDefaultFontSize
	if o.FontSize != nil {
		fontSize = *o.FontSize
	} else if s.FontSize != nil {
		fontSize = *s.FontSize
	}
	scaledFontSize := textScaler.Scale(fontSize)
	var strut *ui.StrutStyle
	if o.StrutStyle != nil {
		var strutFontSize *float64
		if o.StrutStyle.FontSize != nil {
			v := textScaler.Scale(*o.StrutStyle.FontSize)
			strutFontSize = &v
		}
		strut = ui.NewStrutStyle(ui.StrutStyleOptions{
			FontFamily:          o.StrutStyle.FontFamily,
			FontFamilyFallback:  o.StrutStyle.FontFamilyFallback(),
			FontSize:            strutFontSize,
			Height:              o.StrutStyle.Height,
			Leading:             o.StrutStyle.Leading,
			LeadingDistribution: o.StrutStyle.LeadingDistribution,
			FontWeight:          o.StrutStyle.FontWeight,
			FontStyle:           o.StrutStyle.FontStyle,
			ForceStrutHeight:    o.StrutStyle.ForceStrutHeight,
		})
	}
	return ui.NewParagraphStyle(ui.ParagraphStyleOptions{
		TextAlign:          o.TextAlign,
		TextDirection:      o.TextDirection,
		FontWeight:         or(o.FontWeight, s.FontWeight),
		FontStyle:          or(o.FontStyle, s.FontStyle),
		FontFamily:         or(o.FontFamily, s.FontFamily),
		FontSize:           &scaledFontSize,
		Height:             or(o.Height, s.Height),
		TextHeightBehavior: effectiveTextHeightBehavior,
		Hyphens:            o.Hyphens,
		StrutStyle:         strut,
		MaxLines:           o.MaxLines,
		Ellipsis:           o.Ellipsis,
		Locale:             o.Locale,
	})
}

func localeEq(a, b *ui.Locale) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

// CompareTo is TextStyle.compareTo: how different this style is from
// other, in terms of rendering.
func (s *TextStyle) CompareTo(other *TextStyle) RenderComparison {
	if s == other {
		return RenderComparisonIdentical
	}
	if s.Inherit != other.Inherit ||
		!ptrEq(s.FontFamily, other.FontFamily) ||
		!ptrEq(s.FontSize, other.FontSize) ||
		!ptrEq(s.FontWeight, other.FontWeight) ||
		!ptrEq(s.FontStyle, other.FontStyle) ||
		!ptrEq(s.LetterSpacing, other.LetterSpacing) ||
		!ptrEq(s.WordSpacing, other.WordSpacing) ||
		!ptrEq(s.TextBaseline, other.TextBaseline) ||
		!ptrEq(s.Height, other.Height) ||
		!ptrEq(s.LeadingDistribution, other.LeadingDistribution) ||
		!localeEq(s.Locale, other.Locale) ||
		s.Foreground != other.Foreground ||
		s.Background != other.Background ||
		!foundation.ListEquals(s.Shadows, other.Shadows) ||
		!foundation.ListEquals(s.FontFeatures, other.FontFeatures) ||
		!foundation.ListEquals(s.FontVariations, other.FontVariations) ||
		!foundation.ListEquals(s.FontFamilyFallback(), other.FontFamilyFallback()) ||
		!ptrEq(s.Overflow, other.Overflow) {
		return RenderComparisonLayout
	}
	if !ptrEq(s.Color, other.Color) ||
		!ptrEq(s.BackgroundColor, other.BackgroundColor) ||
		!ptrEq(s.Decoration, other.Decoration) ||
		!ptrEq(s.DecorationColor, other.DecorationColor) ||
		!ptrEq(s.DecorationStyle, other.DecorationStyle) ||
		!ptrEq(s.DecorationThickness, other.DecorationThickness) {
		return RenderComparisonPaint
	}
	return RenderComparisonIdentical
}

// Equal is TextStyle's ==; nil is null.
func (s *TextStyle) Equal(other *TextStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return other.Inherit == s.Inherit &&
		ptrEq(other.Color, s.Color) &&
		ptrEq(other.BackgroundColor, s.BackgroundColor) &&
		ptrEq(other.FontSize, s.FontSize) &&
		ptrEq(other.FontWeight, s.FontWeight) &&
		ptrEq(other.FontStyle, s.FontStyle) &&
		ptrEq(other.LetterSpacing, s.LetterSpacing) &&
		ptrEq(other.WordSpacing, s.WordSpacing) &&
		ptrEq(other.TextBaseline, s.TextBaseline) &&
		ptrEq(other.Height, s.Height) &&
		ptrEq(other.LeadingDistribution, s.LeadingDistribution) &&
		localeEq(other.Locale, s.Locale) &&
		other.Foreground == s.Foreground &&
		other.Background == s.Background &&
		foundation.ListEquals(other.Shadows, s.Shadows) &&
		foundation.ListEquals(other.FontFeatures, s.FontFeatures) &&
		foundation.ListEquals(other.FontVariations, s.FontVariations) &&
		ptrEq(other.Decoration, s.Decoration) &&
		ptrEq(other.DecorationColor, s.DecorationColor) &&
		ptrEq(other.DecorationStyle, s.DecorationStyle) &&
		ptrEq(other.DecorationThickness, s.DecorationThickness) &&
		ptrEq(other.FontFamily, s.FontFamily) &&
		foundation.ListEquals(other.FontFamilyFallback(), s.FontFamilyFallback()) &&
		ptrEq(other.pkg, s.pkg) &&
		ptrEq(other.Overflow, s.Overflow)
}

// ToStringShort is objectRuntimeType(this, 'TextStyle').
func (s *TextStyle) ToStringShort() string { return "TextStyle" }

// DebugFillProperties adds the style's properties (no prefix).
func (s *TextStyle) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.DebugFillPropertiesWithPrefix(properties, "")
}

// DebugFillPropertiesWithPrefix is debugFillProperties(properties,
// {prefix}): every property name starts with prefix.
func (s *TextStyle) DebugFillPropertiesWithPrefix(properties *foundation.DiagnosticPropertiesBuilder, prefix string) {
	s.DiagnosticableMixin.DebugFillProperties(properties)
	if s.DebugLabel != nil {
		properties.Add(foundation.NewMessageProperty(prefix+"debugLabel", *s.DebugLabel))
	}
	null := foundation.Null
	quoted := false
	styles := []foundation.DiagnosticsNode{
		NewColorProperty(prefix+"color", s.Color, ColorPropertyOptions{DefaultValue: null}),
		NewColorProperty(prefix+"backgroundColor", s.BackgroundColor, ColorPropertyOptions{DefaultValue: null}),
		foundation.NewStringProperty(prefix+"family", s.FontFamily, foundation.StringPropertyOptions{DefaultValue: null, Quoted: &quoted}),
		foundation.NewIterableProperty(prefix+"familyFallback", s.FontFamilyFallback(), foundation.IterablePropertyOptions{DefaultValue: null}),
		foundation.NewDoubleProperty(prefix+"size", s.FontSize, foundation.DoublePropertyOptions{DefaultValue: null}),
	}
	var weightDescription *string
	if s.FontWeight != nil {
		weightDescription = strPtr(strconv.Itoa(s.FontWeight.Index()+1) + "00")
	}
	styles = append(styles,
		foundation.NewDiagnosticsProperty(prefix+"weight", s.FontWeight,
			foundation.DiagnosticsPropertyOptions{Description: weightDescription, DefaultValue: null}),
		foundation.NewEnumProperty(prefix+"style", s.FontStyle, foundation.EnumPropertyOptions{DefaultValue: null}),
		foundation.NewDoubleProperty(prefix+"letterSpacing", s.LetterSpacing, foundation.DoublePropertyOptions{DefaultValue: null}),
		foundation.NewDoubleProperty(prefix+"wordSpacing", s.WordSpacing, foundation.DoublePropertyOptions{DefaultValue: null}),
		foundation.NewEnumProperty(prefix+"baseline", s.TextBaseline, foundation.EnumPropertyOptions{DefaultValue: null}),
		foundation.NewDoubleProperty(prefix+"height", s.Height, foundation.DoublePropertyOptions{Unit: strPtr("x"), DefaultValue: null}),
		foundation.NewEnumProperty(prefix+"leadingDistribution", s.LeadingDistribution, foundation.EnumPropertyOptions{DefaultValue: null}),
		foundation.NewDiagnosticsProperty(prefix+"locale", s.Locale, foundation.DiagnosticsPropertyOptions{DefaultValue: null}),
		foundation.NewDiagnosticsProperty(prefix+"foreground", s.Foreground, foundation.DiagnosticsPropertyOptions{DefaultValue: null}),
		foundation.NewDiagnosticsProperty(prefix+"background", s.Background, foundation.DiagnosticsPropertyOptions{DefaultValue: null}),
	)
	if s.Decoration != nil || s.DecorationColor != nil || s.DecorationStyle != nil || s.DecorationThickness != nil {
		var decorationDescription []string
		if s.DecorationStyle != nil {
			decorationDescription = append(decorationDescription, s.DecorationStyle.Name())
		}
		// Hide decorationColor from the default text view as it is shown in
		// the terse decoration summary as well.
		fine := foundation.DiagnosticLevelFine
		styles = append(styles, NewColorProperty(prefix+"decorationColor", s.DecorationColor,
			ColorPropertyOptions{DefaultValue: null, Level: &fine}))
		if s.DecorationColor != nil {
			decorationDescription = append(decorationDescription, s.DecorationColor.String())
		}
		// Intentionally collide with the property 'decoration' added below.
		hidden := foundation.DiagnosticLevelHidden
		styles = append(styles, foundation.NewDiagnosticsProperty(prefix+"decoration", s.Decoration,
			foundation.DiagnosticsPropertyOptions{DefaultValue: null, Level: &hidden}))
		if s.Decoration != nil {
			decorationDescription = append(decorationDescription, s.Decoration.String())
		}
		if len(decorationDescription) == 0 {
			panic(dart.NewAssertionError(nil))
		}
		styles = append(styles,
			foundation.NewMessageProperty(prefix+"decoration", strings.Join(decorationDescription, " ")),
			foundation.NewDoubleProperty(prefix+"decorationThickness", s.DecorationThickness,
				foundation.DoublePropertyOptions{Unit: strPtr("x"), DefaultValue: null}))
	}

	styleSpecified := false
	for _, n := range styles {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
			styleSpecified = true
			break
		}
	}
	level := foundation.DiagnosticLevelInfo
	if !styleSpecified && s.Inherit {
		level = foundation.DiagnosticLevelFine
	}
	properties.Add(foundation.NewDiagnosticsProperty(prefix+"inherit", s.Inherit,
		foundation.DiagnosticsPropertyOptions{Level: &level}))
	for _, n := range styles {
		properties.Add(n)
	}
	if !styleSpecified {
		properties.Add(foundation.NewFlagProperty("inherit", s.Inherit, foundation.FlagPropertyOptions{
			IfTrue: strPtr(prefix + "<all styles inherited>"), IfFalse: strPtr(prefix + "<no style specified>"),
		}))
	}
	// Dart adds the overflow property to `styles` after they were added to
	// properties, so it is never shown.
}

// LerpFontVariations interpolates font variation lists (nil is null).
// Variations with the same axis at the same index are lerped pairwise;
// past the first mismatch, the remaining axes are matched by name in the
// iteration order of Dart's HashSet.
func LerpFontVariations(a, b []ui.FontVariation, t float64) []ui.FontVariation {
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	if len(a) == 0 || len(b) == 0 {
		// If one side is empty, that means anything on the other side will
		// use the null-to-something lerp, which is to say, a step function
		// at t=0.5.
		if t < 0.5 {
			return a
		}
		return b
	}
	result := []ui.FontVariation{}
	// First, try the efficient O(N) solution in the event that the lists are
	// compatible.
	index := 0
	minLength := min(len(a), len(b))
	for ; index < minLength; index++ {
		if a[index].Axis != b[index].Axis {
			// The lists aren't compatible.
			break
		}
		result = append(result, *ui.FontVariationLerp(&a[index], &b[index], t))
	}
	maxLength := max(len(a), len(b))
	if index < maxLength {
		// If we get here, we have found some case where we cannot use the
		// efficient approach.
		axes := newDartStringHashSet()
		aVariations := map[string]ui.FontVariation{}
		for indexA := index; indexA < len(a); indexA++ {
			aVariations[a[indexA].Axis] = a[indexA]
			axes.add(a[indexA].Axis)
		}
		bVariations := map[string]ui.FontVariation{}
		for indexB := index; indexB < len(b); indexB++ {
			bVariations[b[indexB].Axis] = b[indexB]
			axes.add(b[indexB].Axis)
		}
		for _, axis := range axes.all() {
			var va, vb *ui.FontVariation
			if v, ok := aVariations[axis]; ok {
				va = &v
			}
			if v, ok := bVariations[axis]; ok {
				vb = &v
			}
			if variation := ui.FontVariationLerp(va, vb, t); variation != nil {
				result = append(result, *variation)
			}
		}
	}
	return result
}

// dartStringHashSet is the Dart VM's HashSet<String> (dart:collection's
// _HashSet), kept for its iteration order: buckets indexed by the string's
// hash code, each a chain with the newest entry first, doubled when more
// than 75% full.
type dartStringHashSet struct {
	buckets [][]dartHashSetEntry // each chain, head first
	count   int
}

type dartHashSetEntry struct {
	key  string
	hash int
}

func newDartStringHashSet() *dartStringHashSet {
	return &dartStringHashSet{buckets: make([][]dartHashSetEntry, 8)}
}

func (s *dartStringHashSet) add(key string) {
	hash := dart.StringHashCode(key)
	index := hash & (len(s.buckets) - 1)
	for _, e := range s.buckets[index] {
		if e.key == key {
			return
		}
	}
	s.buckets[index] = append([]dartHashSetEntry{{key, hash}}, s.buckets[index]...)
	s.count++
	length := len(s.buckets)
	if (s.count << 2) > ((length << 1) + length) {
		newLength := length << 1
		newBuckets := make([][]dartHashSetEntry, newLength)
		for _, chain := range s.buckets {
			for _, e := range chain {
				i := e.hash & (newLength - 1)
				newBuckets[i] = append([]dartHashSetEntry{e}, newBuckets[i]...)
			}
		}
		s.buckets = newBuckets
	}
}

func (s *dartStringHashSet) all() []string {
	var out []string
	for _, chain := range s.buckets {
		for _, e := range chain {
			out = append(out, e.key)
		}
	}
	return out
}
