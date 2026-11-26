package painting

// Port of strut_style.dart (Flutter 3.50.1): StrutStyle, the minimum
// vertical layout metrics of a paragraph.

import (
	"strconv"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// StrutStyle defines the strut, which sets the minimum height a line can
// be relative to the baseline. Used by pointer and built with
// NewStrutStyle; nil fields are Dart's nulls.
type StrutStyle struct {
	foundation.DiagnosticableMixin

	// FontFamily is the name of the font to use when calculating the strut
	// (with the package prefix when a package was given).
	FontFamily *string
	// FontSize is the size of text (in logical pixels) to use when
	// obtaining metrics from the font.
	FontSize *float64
	// Height is the minimum height of the strut, as a multiple of FontSize.
	Height *float64
	// LeadingDistribution is how the vertical space added by Height is
	// distributed.
	LeadingDistribution *ui.TextLeadingDistribution
	// FontWeight is the typeface thickness to use when calculating the
	// strut.
	FontWeight *ui.FontWeight
	// FontStyle is the typeface variant to use when calculating the strut.
	FontStyle *ui.FontStyle
	// Leading is the additional leading to apply to the strut as a
	// multiple of FontSize.
	Leading *float64
	// ForceStrutHeight is whether the strut height should be forced.
	ForceStrutHeight *bool
	// DebugLabel is a human-readable description of this strut style.
	DebugLabel *string

	fontFamilyFallback []string
	pkg                *string
}

// StrutStyleOptions are StrutStyle's named parameters (nil: null).
type StrutStyleOptions struct {
	FontFamily          *string
	FontFamilyFallback  []string
	FontSize            *float64
	Height              *float64
	LeadingDistribution *ui.TextLeadingDistribution
	Leading             *float64
	FontWeight          *ui.FontWeight
	FontStyle           *ui.FontStyle
	ForceStrutHeight    *bool
	DebugLabel          *string
	Package             *string
}

// NewStrutStyle is StrutStyle({...}). It panics as Dart asserts: a
// non-positive fontSize, a negative leading, or a package with neither a
// family nor fallbacks.
func NewStrutStyle(opts ...StrutStyleOptions) *StrutStyle {
	var o StrutStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.FontSize != nil && !(*o.FontSize > 0) {
		panic(dart.NewAssertionError(nil))
	}
	if o.Leading != nil && !(*o.Leading >= 0) {
		panic(dart.NewAssertionError(nil))
	}
	if o.Package != nil && o.FontFamily == nil && o.FontFamilyFallback == nil {
		panic(dart.NewAssertionError(nil))
	}
	s := &StrutStyle{
		FontFamily:          packagePrefixed(o.Package, o.FontFamily),
		FontSize:            o.FontSize,
		Height:              o.Height,
		LeadingDistribution: o.LeadingDistribution,
		FontWeight:          o.FontWeight,
		FontStyle:           o.FontStyle,
		Leading:             o.Leading,
		ForceStrutHeight:    o.ForceStrutHeight,
		DebugLabel:          o.DebugLabel,
		fontFamilyFallback:  o.FontFamilyFallback,
		pkg:                 o.Package,
	}
	s.SetSelf(s)
	return s
}

// NewStrutStyleFromTextStyle is StrutStyle.fromTextStyle: a strut style
// whose unset values are textStyle's.
func NewStrutStyleFromTextStyle(textStyle *TextStyle, opts ...StrutStyleOptions) *StrutStyle {
	var o StrutStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	fontFamily := textStyle.FontFamily
	if o.FontFamily != nil {
		fontFamily = packagePrefixed(o.Package, o.FontFamily)
	}
	return NewStrutStyle(StrutStyleOptions{
		FontFamily:          fontFamily,
		FontFamilyFallback:  orList(o.FontFamilyFallback, textStyle.FontFamilyFallback()),
		Height:              or(o.Height, textStyle.Height),
		LeadingDistribution: or(o.LeadingDistribution, textStyle.LeadingDistribution),
		FontSize:            or(o.FontSize, textStyle.FontSize),
		Leading:             o.Leading,
		FontWeight:          or(o.FontWeight, textStyle.FontWeight),
		FontStyle:           or(o.FontStyle, textStyle.FontStyle),
		ForceStrutHeight:    o.ForceStrutHeight,
		DebugLabel:          or(o.DebugLabel, textStyle.DebugLabel),
		Package:             o.Package,
	})
}

// StrutStyleDisabled is StrutStyle.disabled: a strut with height and
// leading 0, which disables the strut.
var StrutStyleDisabled = NewStrutStyle(StrutStyleOptions{Height: f64(0.0), Leading: f64(0.0)})

func f64(v float64) *float64 { return &v }

// FontFamilyFallback is the ordered list of font families to fall back on
// (prefixed with the package's path when a package was given).
func (s *StrutStyle) FontFamilyFallback() []string {
	if s.pkg != nil && s.fontFamilyFallback != nil {
		out := make([]string, len(s.fontFamilyFallback))
		for i, f := range s.fontFamilyFallback {
			out[i] = "packages/" + *s.pkg + "/" + f
		}
		return out
	}
	return s.fontFamilyFallback
}

// CompareTo is StrutStyle.compareTo: identical, or layout when any value
// differs.
func (s *StrutStyle) CompareTo(other *StrutStyle) RenderComparison {
	if s == other {
		return RenderComparisonIdentical
	}
	if !ptrEq(s.FontFamily, other.FontFamily) ||
		!ptrEq(s.FontSize, other.FontSize) ||
		!ptrEq(s.FontWeight, other.FontWeight) ||
		!ptrEq(s.FontStyle, other.FontStyle) ||
		!ptrEq(s.Height, other.Height) ||
		!ptrEq(s.Leading, other.Leading) ||
		!ptrEq(s.ForceStrutHeight, other.ForceStrutHeight) ||
		!foundation.ListEquals(s.FontFamilyFallback(), other.FontFamilyFallback()) ||
		(s.Height != nil && !ptrEq(s.LeadingDistribution, other.LeadingDistribution)) {
		return RenderComparisonLayout
	}
	return RenderComparisonIdentical
}

// InheritFromTextStyle is StrutStyle.inheritFromTextStyle: this strut with
// its null values taken from other (s itself when other is nil).
func (s *StrutStyle) InheritFromTextStyle(other *TextStyle) *StrutStyle {
	if other == nil {
		return s
	}
	effectiveHeight := or(s.Height, other.Height)
	var leadingDistribution *ui.TextLeadingDistribution
	if effectiveHeight != nil {
		leadingDistribution = or(s.LeadingDistribution, other.LeadingDistribution)
	}
	return NewStrutStyle(StrutStyleOptions{
		FontFamily:          or(s.FontFamily, other.FontFamily),
		FontFamilyFallback:  orList(s.FontFamilyFallback(), other.FontFamilyFallback()),
		FontSize:            or(s.FontSize, other.FontSize),
		Height:              effectiveHeight,
		Leading:             s.Leading, // No equivalent property in TextStyle yet.
		FontWeight:          or(s.FontWeight, other.FontWeight),
		FontStyle:           or(s.FontStyle, other.FontStyle),
		ForceStrutHeight:    s.ForceStrutHeight, // StrutStyle-unique property.
		DebugLabel:          or(s.DebugLabel, other.DebugLabel),
		LeadingDistribution: leadingDistribution,
		// Package is embedded within the getters for fontFamilyFallback.
	})
}

// Merge is StrutStyle.merge: this strut with the non-null values of other
// (s itself when other is nil).
func (s *StrutStyle) Merge(other *StrutStyle) *StrutStyle {
	if other == nil {
		return s
	}
	return NewStrutStyle(StrutStyleOptions{
		FontFamily:          or(other.FontFamily, s.FontFamily),
		FontFamilyFallback:  orList(other.FontFamilyFallback(), s.FontFamilyFallback()),
		FontSize:            or(other.FontSize, s.FontSize),
		Height:              or(other.Height, s.Height),
		LeadingDistribution: or(other.LeadingDistribution, s.LeadingDistribution),
		Leading:             or(other.Leading, s.Leading),
		FontWeight:          or(other.FontWeight, s.FontWeight),
		FontStyle:           or(other.FontStyle, s.FontStyle),
		ForceStrutHeight:    or(other.ForceStrutHeight, s.ForceStrutHeight),
		DebugLabel:          or(other.DebugLabel, s.DebugLabel),
		Package:             or(other.pkg, s.pkg),
	})
}

// Equal is StrutStyle's ==; nil is null.
func (s *StrutStyle) Equal(other *StrutStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return ptrEq(other.FontFamily, s.FontFamily) &&
		ptrEq(other.FontSize, s.FontSize) &&
		ptrEq(other.FontWeight, s.FontWeight) &&
		ptrEq(other.FontStyle, s.FontStyle) &&
		ptrEq(other.Height, s.Height) &&
		ptrEq(other.Leading, s.Leading) &&
		ptrEq(other.ForceStrutHeight, s.ForceStrutHeight) &&
		(s.Height == nil || ptrEq(s.LeadingDistribution, other.LeadingDistribution)) &&
		foundation.ListEquals(other.FontFamilyFallback(), s.FontFamilyFallback())
}

// ToStringShort is objectRuntimeType(this, 'StrutStyle').
func (s *StrutStyle) ToStringShort() string { return "StrutStyle" }

// DebugFillProperties adds the strut's properties (no prefix).
func (s *StrutStyle) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.DebugFillPropertiesWithPrefix(properties, "")
}

// DebugFillPropertiesWithPrefix is debugFillProperties(properties,
// {prefix}).
func (s *StrutStyle) DebugFillPropertiesWithPrefix(properties *foundation.DiagnosticPropertiesBuilder, prefix string) {
	s.DiagnosticableMixin.DebugFillProperties(properties)
	if s.DebugLabel != nil {
		properties.Add(foundation.NewMessageProperty(prefix+"debugLabel", *s.DebugLabel))
	}
	null := foundation.Null
	quoted := false
	styles := []foundation.DiagnosticsNode{
		foundation.NewStringProperty(prefix+"family", s.FontFamily, foundation.StringPropertyOptions{DefaultValue: null, Quoted: &quoted}),
		foundation.NewIterableProperty(prefix+"familyFallback", s.FontFamilyFallback(), foundation.IterablePropertyOptions{DefaultValue: null}),
		foundation.NewDoubleProperty(prefix+"size", s.FontSize, foundation.DoublePropertyOptions{DefaultValue: null}),
	}
	var weightDescription *string
	if s.FontWeight != nil {
		weightDescription = strPtr("w" + strconv.Itoa(s.FontWeight.Index()+1) + "00")
	}
	styles = append(styles,
		foundation.NewDiagnosticsProperty(prefix+"weight", s.FontWeight,
			foundation.DiagnosticsPropertyOptions{Description: weightDescription, DefaultValue: null}),
		foundation.NewEnumProperty(prefix+"style", s.FontStyle, foundation.EnumPropertyOptions{DefaultValue: null}),
		foundation.NewDoubleProperty(prefix+"height", s.Height, foundation.DoublePropertyOptions{Unit: strPtr("x"), DefaultValue: null}),
		foundation.NewFlagProperty(prefix+"forceStrutHeight", s.ForceStrutHeight, foundation.FlagPropertyOptions{
			IfTrue: strPtr(prefix + "<strut height forced>"), IfFalse: strPtr(prefix + "<strut height normal>"),
		}),
	)
	if s.Height != nil {
		styles = append(styles, foundation.NewEnumProperty(prefix+"leadingDistribution", s.LeadingDistribution,
			foundation.EnumPropertyOptions{DefaultValue: null}))
	}

	styleSpecified := false
	for _, n := range styles {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
			styleSpecified = true
			break
		}
	}
	for _, n := range styles {
		properties.Add(n)
	}
	if !styleSpecified {
		properties.Add(foundation.NewFlagProperty("forceStrutHeight", s.ForceStrutHeight, foundation.FlagPropertyOptions{
			IfTrue: strPtr(prefix + "<strut height forced>"), IfFalse: strPtr(prefix + "<strut height normal>"),
		}))
	}
}
