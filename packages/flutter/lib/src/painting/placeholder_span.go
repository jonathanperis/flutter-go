package painting

// Port of placeholder_span.dart (Flutter 3.50.1): PlaceholderSpan, the
// abstract InlineSpan that reserves space for a placeholder (widgets'
// WidgetSpan implements it).

import (
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PlaceholderSpanPlaceholderCodeUnit is PlaceholderSpan.placeholderCodeUnit,
// the code unit that stands for a placeholder in the plain text (U+FFFC).
const PlaceholderSpanPlaceholderCodeUnit = 0xFFFC

// PlaceholderSpan is an immutable placeholder embedded inline within text.
// Implementations embed PlaceholderSpanBase, call InitPlaceholderSpan and
// implement Build, VisitChildren, VisitDirectChildren,
// GetSpanForPositionVisitor, CodeUnitAtVisitor, CompareTo and
// DebugAssertIsValid (PlaceholderSpan's asserts false, as Dart's does).
type PlaceholderSpan interface {
	InlineSpan
	// Alignment is how the placeholder aligns vertically with the text.
	Alignment() ui.PlaceholderAlignment
	// Baseline is the TextBaseline to align against (nil: none).
	Baseline() *ui.TextBaseline
}

// PlaceholderSpanOptions are PlaceholderSpan's named parameters (Alignment
// nil: PlaceholderAlignment.bottom).
type PlaceholderSpanOptions struct {
	Alignment *ui.PlaceholderAlignment
	Baseline  *ui.TextBaseline
	Style     *TextStyle
}

// PlaceholderSpanBase is PlaceholderSpan's concrete part.
type PlaceholderSpanBase struct {
	InlineSpanBase
	alignment ui.PlaceholderAlignment
	baseline  *ui.TextBaseline
}

// InitPlaceholderSpan sets up the base (Dart's `PlaceholderSpan({...})`)
// for the span self that embeds it.
func (s *PlaceholderSpanBase) InitPlaceholderSpan(self PlaceholderSpan, opts ...PlaceholderSpanOptions) {
	var o PlaceholderSpanOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s.InitInlineSpan(self, o.Style)
	s.alignment = ui.PlaceholderAlignmentBottom
	if o.Alignment != nil {
		s.alignment = *o.Alignment
	}
	s.baseline = o.Baseline
}

// Alignment is how the placeholder aligns vertically with the text.
func (s *PlaceholderSpanBase) Alignment() ui.PlaceholderAlignment { return s.alignment }

// Baseline is the TextBaseline to align against (nil: none).
func (s *PlaceholderSpanBase) Baseline() *ui.TextBaseline { return s.baseline }

// ComputeToPlainText writes the placeholder code unit (U+FFFC) when
// placeholders are included.
func (s *PlaceholderSpanBase) ComputeToPlainText(buffer *strings.Builder, opts ...ToPlainTextOptions) {
	if _, includePlaceholders := plainTextOptions(opts).resolve(); includePlaceholders {
		buffer.WriteString(dart.StringFromCharCode(PlaceholderSpanPlaceholderCodeUnit))
	}
}

// ComputeSemanticsInformation adds InlineSpanSemanticsInformation's
// placeholder.
func (s *PlaceholderSpanBase) ComputeSemanticsInformation(collector *[]InlineSpanSemanticsInformation) {
	*collector = append(*collector, InlineSpanSemanticsInformationPlaceholder)
}

// DebugFillProperties adds the alignment and the baseline.
func (s *PlaceholderSpanBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.InlineSpanBase.DebugFillProperties(properties)
	properties.Add(foundation.NewEnumProperty("alignment", s.alignment, foundation.EnumPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewEnumProperty("baseline", s.baseline, foundation.EnumPropertyOptions{DefaultValue: foundation.Null}))
}

// DebugAssertIsValid asserts false, as Dart's PlaceholderSpan does: a
// concrete placeholder span overrides it.
func (s *PlaceholderSpanBase) DebugAssertIsValid() bool {
	panic(dart.NewAssertionError("Consider implementing the WidgetSpan interface instead."))
}
