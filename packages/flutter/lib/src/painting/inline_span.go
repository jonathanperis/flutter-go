package painting

// Port of inline_span.dart (Flutter 3.50.1): InlineSpan (an interface plus
// the embeddable InlineSpanBase), Accumulator, InlineSpanVisitor,
// InlineSpanSemanticsInformation and combineSemanticsInfo.

import (
	"reflect"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Accumulator is a mutable wrapper of an int that can be passed by
// reference to track a value across a recursive stack.
type Accumulator struct{ value int }

// NewAccumulator is Accumulator([value = 0]).
func NewAccumulator(value ...int) *Accumulator {
	a := &Accumulator{}
	if len(value) > 0 {
		a.value = value[0]
	}
	return a
}

// Value is the integer stored in this accumulator.
func (a *Accumulator) Value() int { return a.value }

// Increment increases the value by addend (which must not be negative).
func (a *Accumulator) Increment(addend int) {
	if !(addend >= 0) {
		panic(dart.NewAssertionError(nil))
	}
	a.value += addend
}

// InlineSpanVisitor is called on each span as InlineSpan.VisitChildren walks
// the tree; returning false stops the walk.
type InlineSpanVisitor func(span InlineSpan) bool

// InlineSpanSemanticsInformation is the textual and semantic label
// information for an InlineSpan. Compare with Equal.
type InlineSpanSemanticsInformation struct {
	// Text is the text info for a TextSpan.
	Text string
	// SemanticsLabel is the semanticsLabel, if any (nil: null).
	SemanticsLabel *string
	// SemanticsIdentifier is the semanticsIdentifier, if any.
	SemanticsIdentifier *string
	// Recognizer is the gesture recognizer, if any.
	Recognizer gestures.GestureRecognizer
	// IsPlaceholder is whether this is for a placeholder span.
	IsPlaceholder bool
	// RequiresOwnNode is true if this object needs its own semantics node
	// (a placeholder, a recognizer or a semantics identifier).
	RequiresOwnNode bool
	// StringAttributes are the string attributes attached to this
	// semantics information.
	StringAttributes []ui.StringAttribute
}

// InlineSpanSemanticsInformationOptions are
// InlineSpanSemanticsInformation's named parameters.
type InlineSpanSemanticsInformationOptions struct {
	IsPlaceholder       bool
	SemanticsLabel      *string
	SemanticsIdentifier *string
	// StringAttributes nil is Dart's default, an empty list.
	StringAttributes []ui.StringAttribute
	Recognizer       gestures.GestureRecognizer
}

// NewInlineSpanSemanticsInformation is InlineSpanSemanticsInformation(text,
// {...}). A placeholder must have the text "￼" and no label or
// recognizer (Dart asserts).
func NewInlineSpanSemanticsInformation(text string, opts ...InlineSpanSemanticsInformationOptions) InlineSpanSemanticsInformation {
	var o InlineSpanSemanticsInformationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.IsPlaceholder && !(text == "￼" && o.SemanticsLabel == nil && o.Recognizer == nil) {
		panic(dart.NewAssertionError(nil))
	}
	attrs := o.StringAttributes
	if attrs == nil {
		attrs = []ui.StringAttribute{}
	}
	return InlineSpanSemanticsInformation{
		Text:                text,
		SemanticsLabel:      o.SemanticsLabel,
		SemanticsIdentifier: o.SemanticsIdentifier,
		Recognizer:          o.Recognizer,
		IsPlaceholder:       o.IsPlaceholder,
		RequiresOwnNode:     o.IsPlaceholder || o.Recognizer != nil || o.SemanticsIdentifier != nil,
		StringAttributes:    attrs,
	}
}

// InlineSpanSemanticsInformationPlaceholder is
// InlineSpanSemanticsInformation.placeholder.
var InlineSpanSemanticsInformationPlaceholder = NewInlineSpanSemanticsInformation("￼",
	InlineSpanSemanticsInformationOptions{IsPlaceholder: true})

// Equal is InlineSpanSemanticsInformation's == (the string attributes by
// identity, as Dart's listEquals on them).
func (i InlineSpanSemanticsInformation) Equal(other InlineSpanSemanticsInformation) bool {
	if !(other.Text == i.Text && ptrEq(other.SemanticsLabel, i.SemanticsLabel) &&
		ptrEq(other.SemanticsIdentifier, i.SemanticsIdentifier) && other.Recognizer == i.Recognizer &&
		other.IsPlaceholder == i.IsPlaceholder) {
		return false
	}
	if (other.StringAttributes == nil) != (i.StringAttributes == nil) || len(other.StringAttributes) != len(i.StringAttributes) {
		return false
	}
	for k := range i.StringAttributes {
		if other.StringAttributes[k] != i.StringAttributes[k] {
			return false
		}
	}
	return true
}

func nullableString(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func (i InlineSpanSemanticsInformation) String() string {
	recognizer := "null"
	if i.Recognizer != nil {
		recognizer = dart.ToString(i.Recognizer)
	}
	return "InlineSpanSemanticsInformation{text: " + i.Text + ", semanticsLabel: " + nullableString(i.SemanticsLabel) +
		", semanticsIdentifier: " + nullableString(i.SemanticsIdentifier) + ", recognizer: " + recognizer + "}"
}

// CombineSemanticsInfo is combineSemanticsInfo: combines adjacent entries
// that don't need their own node into one, keeping their labels and string
// attributes (shifted to the combined label).
func CombineSemanticsInfo(infoList []InlineSpanSemanticsInformation) []InlineSpanSemanticsInformation {
	combined := []InlineSpanSemanticsInformation{}
	workingText := ""
	workingLabel := ""
	workingAttributes := []ui.StringAttribute{}
	for _, info := range infoList {
		if info.RequiresOwnNode {
			label := workingLabel
			combined = append(combined, NewInlineSpanSemanticsInformation(workingText,
				InlineSpanSemanticsInformationOptions{SemanticsLabel: &label, StringAttributes: workingAttributes}))
			workingText = ""
			workingLabel = ""
			workingAttributes = []ui.StringAttribute{}
			combined = append(combined, info)
		} else {
			workingText = dart.Concat(workingText, info.Text)
			effectiveLabel := info.Text
			if info.SemanticsLabel != nil {
				effectiveLabel = *info.SemanticsLabel
			}
			labelLength := dart.StringLength(workingLabel)
			for _, infoAttribute := range info.StringAttributes {
				workingAttributes = append(workingAttributes, infoAttribute.Copy(ui.NewTextRange(
					infoAttribute.Range().Start+labelLength, infoAttribute.Range().End+labelLength)))
			}
			workingLabel = dart.Concat(workingLabel, effectiveLabel)
		}
	}
	label := workingLabel
	combined = append(combined, NewInlineSpanSemanticsInformation(workingText,
		InlineSpanSemanticsInformationOptions{SemanticsLabel: &label, StringAttributes: workingAttributes}))
	return combined
}

// InlineSpanBuildOptions are InlineSpan.build's named parameters
// (TextScaler nil: TextScalerNoScaling; Dimensions nil: null).
type InlineSpanBuildOptions struct {
	TextScaler TextScaler
	Dimensions []PlaceholderDimensions
}

func (o InlineSpanBuildOptions) textScaler() TextScaler {
	if o.TextScaler == nil {
		return TextScalerNoScaling
	}
	return o.TextScaler
}

// ToPlainTextOptions are InlineSpan.toPlainText's (and
// computeToPlainText's) named parameters; nil is Dart's default, true.
type ToPlainTextOptions struct {
	IncludeSemanticsLabels *bool
	IncludePlaceholders    *bool
}

func (o ToPlainTextOptions) resolve() (labels, placeholders bool) {
	return o.IncludeSemanticsLabels == nil || *o.IncludeSemanticsLabels,
		o.IncludePlaceholders == nil || *o.IncludePlaceholders
}

func plainTextOptions(opts []ToPlainTextOptions) ToPlainTextOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return ToPlainTextOptions{}
}

// InlineSpan is an immutable span of inline content which forms part of a
// paragraph (Dart's abstract class). Implementations embed InlineSpanBase
// (or a span that does) and call SetSelf.
type InlineSpan interface {
	foundation.DiagnosticableTree

	// Style is the TextStyle to apply to this span and its children (nil:
	// none).
	Style() *TextStyle
	// Build applies the span's style, text and children to the builder.
	Build(builder ui.ParagraphBuilder, opts ...InlineSpanBuildOptions)
	// VisitChildren walks this span and its descendants in pre-order,
	// calling visitor on each span that has content; it returns false when
	// the visitor stopped the walk.
	VisitChildren(visitor InlineSpanVisitor) bool
	// VisitDirectChildren calls visitor on each direct child span.
	VisitDirectChildren(visitor InlineSpanVisitor) bool
	// GetSpanForPosition returns the span that contains the given position
	// (nil: none).
	GetSpanForPosition(position ui.TextPosition) InlineSpan
	// GetSpanForPositionVisitor performs the check at each span for
	// GetSpanForPosition (protected).
	GetSpanForPositionVisitor(position ui.TextPosition, offset *Accumulator) InlineSpan
	// ToPlainText flattens the span tree into a single string.
	ToPlainText(opts ...ToPlainTextOptions) string
	// GetSemanticsInformation flattens the span tree to its semantics
	// information.
	GetSemanticsInformation() []InlineSpanSemanticsInformation
	// ComputeSemanticsInformation walks the span tree and accumulates its
	// semantics information into collector (protected).
	ComputeSemanticsInformation(collector *[]InlineSpanSemanticsInformation)
	// ComputeToPlainText walks the span tree and writes its text to buffer
	// (protected).
	ComputeToPlainText(buffer *strings.Builder, opts ...ToPlainTextOptions)
	// CodeUnitAt returns the UTF-16 code unit at index in the flattened
	// string (nil: out of range).
	CodeUnitAt(index int) *int
	// CodeUnitAtVisitor performs the check at each span for CodeUnitAt
	// (protected).
	CodeUnitAtVisitor(index int, offset *Accumulator) *int
	// DebugAssertIsValid is the debug check that the span tree is valid.
	DebugAssertIsValid() bool
	// CompareTo describes the difference between this span and other in
	// terms of their effect on rendering.
	CompareTo(other InlineSpan) RenderComparison
	// Equal is InlineSpan's ==.
	Equal(other InlineSpan) bool
}

// InlineSpanBase is InlineSpan's concrete part: the style, the template
// methods built on the overridable ones (called through self) and
// DiagnosticableTree.
type InlineSpanBase struct {
	foundation.DiagnosticableTreeBase
	self  InlineSpan
	style *TextStyle
}

// InitInlineSpan sets up the base (Dart's `InlineSpan({style})`) for the
// span self that embeds it.
func (s *InlineSpanBase) InitInlineSpan(self InlineSpan, style *TextStyle) {
	s.self = self
	s.style = style
	s.DiagnosticableTreeBase.SetSelf(self)
}

func (s *InlineSpanBase) outer() InlineSpan {
	if s.self == nil {
		panic("painting: InlineSpanBase used before InitInlineSpan")
	}
	return s.self
}

// Style is the span's style (nil: none).
func (s *InlineSpanBase) Style() *TextStyle { return s.style }

// GetSpanForPosition returns the span that contains position (nil: none).
func (s *InlineSpanBase) GetSpanForPosition(position ui.TextPosition) InlineSpan {
	self := s.outer()
	if !self.DebugAssertIsValid() {
		panic(dart.NewAssertionError(nil))
	}
	offset := NewAccumulator()
	var result InlineSpan
	self.VisitChildren(func(span InlineSpan) bool {
		result = span.GetSpanForPositionVisitor(position, offset)
		return result == nil
	})
	return result
}

// ToPlainText flattens the span tree into a single string.
func (s *InlineSpanBase) ToPlainText(opts ...ToPlainTextOptions) string {
	var buffer strings.Builder
	s.outer().ComputeToPlainText(&buffer, plainTextOptions(opts))
	return buffer.String()
}

// GetSemanticsInformation flattens the span tree to its semantics
// information.
func (s *InlineSpanBase) GetSemanticsInformation() []InlineSpanSemanticsInformation {
	collector := []InlineSpanSemanticsInformation{}
	s.outer().ComputeSemanticsInformation(&collector)
	return collector
}

// CodeUnitAt returns the UTF-16 code unit at index (nil: out of range).
func (s *InlineSpanBase) CodeUnitAt(index int) *int {
	if index < 0 {
		return nil
	}
	offset := NewAccumulator()
	var result *int
	s.outer().VisitChildren(func(span InlineSpan) bool {
		result = span.CodeUnitAtVisitor(index, offset)
		return result == nil
	})
	return result
}

// DebugAssertIsValid is true.
func (s *InlineSpanBase) DebugAssertIsValid() bool { return true }

// Equal is InlineSpan's ==: the same dynamic type and an equal style.
func (s *InlineSpanBase) Equal(other InlineSpan) bool {
	self := s.outer()
	if other == nil {
		return false
	}
	if self == other {
		return true
	}
	if !sameRuntimeType(self, other) {
		return false
	}
	return other.Style().Equal(s.style)
}

// DebugFillProperties uses the whitespace tree style and adds the style's
// properties.
func (s *InlineSpanBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.DiagnosticableTreeBase.DebugFillProperties(properties)
	properties.DefaultDiagnosticsTreeStyle = foundation.DiagnosticsTreeStyleWhitespace
	if s.style != nil {
		s.style.DebugFillProperties(properties)
	}
}

// sameRuntimeType is Dart's `a.runtimeType == b.runtimeType`.
func sameRuntimeType(a, b any) bool { return reflect.TypeOf(a) == reflect.TypeOf(b) }
