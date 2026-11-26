package painting

// Port of text_span.dart (Flutter 3.50.1): TextSpan, an InlineSpan of text
// that is also a HitTestTarget and a MouseTrackerAnnotation.

import (
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// TextSpan is an immutable span of text. Used by pointer and built with
// NewTextSpan. Text nil is Dart's null text (an empty string is not); a nil
// Children is null.
type TextSpan struct {
	InlineSpanBase

	// Text is the text contained in this span (nil: none).
	Text *string
	// Children are additional spans to include as children (nil: none).
	Children []InlineSpan
	// Recognizer is a gesture recognizer that will receive events that hit
	// this span (nil: none).
	Recognizer gestures.GestureRecognizer
	// MouseCursor is the mouse cursor for pointers hovering over this
	// span.
	MouseCursor services.MouseCursor
	// SemanticsLabel is an alternative semantics label for this span.
	SemanticsLabel *string
	// SemanticsIdentifier is a unique identifier for the semantics node of
	// this span.
	SemanticsIdentifier *string
	// Locale is the language of the text for speech synthesis.
	Locale *ui.Locale
	// SpellOut is whether the assistive technologies should spell out this
	// text character by character (nil: inherited).
	SpellOut *bool

	onEnter services.PointerEnterEventListener
	onExit  services.PointerExitEventListener
}

// TextSpanOptions are TextSpan's named parameters (nil: null; MouseCursor
// nil: MouseCursor.defer, or SystemMouseCursors.click with a recognizer).
type TextSpanOptions struct {
	Text                *string
	Children            []InlineSpan
	Style               *TextStyle
	Recognizer          gestures.GestureRecognizer
	MouseCursor         services.MouseCursor
	OnEnter             services.PointerEnterEventListener
	OnExit              services.PointerExitEventListener
	SemanticsLabel      *string
	SemanticsIdentifier *string
	Locale              *ui.Locale
	SpellOut            *bool
}

// NewTextSpan is TextSpan({...}). It panics (Dart's assert) when a
// semantics label is given without text.
func NewTextSpan(opts ...TextSpanOptions) *TextSpan {
	var o TextSpanOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := &TextSpan{}
	s.Construct(s, o)
	return s
}

// Construct sets up a TextSpan embedded in self (Dart's `super(...)` of a
// TextSpan subclass).
func (s *TextSpan) Construct(self InlineSpan, o TextSpanOptions) {
	if o.Text == nil && o.SemanticsLabel != nil {
		panic(dart.NewAssertionError(nil))
	}
	s.InitInlineSpan(self, o.Style)
	s.Text, s.Children, s.Recognizer = o.Text, o.Children, o.Recognizer
	s.SemanticsLabel, s.SemanticsIdentifier, s.Locale, s.SpellOut = o.SemanticsLabel, o.SemanticsIdentifier, o.Locale, o.SpellOut
	s.onEnter, s.onExit = o.OnEnter, o.OnExit
	s.MouseCursor = o.MouseCursor
	if s.MouseCursor == nil {
		if o.Recognizer == nil {
			s.MouseCursor = services.MouseCursorDefer
		} else {
			s.MouseCursor = services.SystemMouseCursors.Click
		}
	}
}

// OnEnter is triggered when a mouse pointer enters this span (nil: none).
func (s *TextSpan) OnEnter() services.PointerEnterEventListener { return s.onEnter }

// OnExit is triggered when a mouse pointer exits this span (nil: none).
func (s *TextSpan) OnExit() services.PointerExitEventListener { return s.onExit }

// Cursor is MouseCursor (MouseTrackerAnnotation's cursor).
func (s *TextSpan) Cursor() services.MouseCursor { return s.MouseCursor }

// ValidForMouseTracker is true.
func (s *TextSpan) ValidForMouseTracker() bool { return true }

// HandleEvent gives a pointer down event to the recognizer.
func (s *TextSpan) HandleEvent(event gestures.PointerEvent, entry gestures.HitTestEntry) {
	if down, ok := event.(*gestures.PointerDownEvent); ok && s.Recognizer != nil {
		s.Recognizer.AddPointer(down)
	}
}

func isArgumentError(r any) bool {
	switch r.(type) {
	case dart.ArgumentError, *dart.ArgumentError:
		return true
	}
	return false
}

// Build applies the style, the text and the children to builder. Text the
// builder rejects (malformed UTF-16) is reported silently and replaced
// with U+FFFD.
func (s *TextSpan) Build(builder ui.ParagraphBuilder, opts ...InlineSpanBuildOptions) {
	var o InlineSpanBuildOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	self := s.outer()
	if !self.DebugAssertIsValid() {
		panic(dart.NewAssertionError(nil))
	}
	style := s.Style()
	hasStyle := style != nil
	if hasStyle {
		builder.PushStyle(style.GetTextStyle(TextStyleGetTextStyleOptions{TextScaler: o.textScaler()}))
	}
	if s.Text != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if !isArgumentError(r) {
						panic(r)
					}
					library := "painting library"
					foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(r, foundation.FlutterErrorDetailsOptions{
						Stack:   dart.StackTraceFromPanic(),
						Library: &library,
						Context: foundation.NewErrorDescription("while building a TextSpan"),
						Silent:  true,
					}))
					// Use a Unicode replacement character as a substitute for
					// invalid text.
					builder.AddText("�")
				}
			}()
			builder.AddText(*s.Text)
		}()
	}
	for _, child := range s.Children {
		child.Build(builder, InlineSpanBuildOptions{TextScaler: o.textScaler(), Dimensions: o.Dimensions})
	}
	if hasStyle {
		builder.Pop()
	}
}

// VisitChildren calls visitor on this span (when it has text) and then on
// its descendants, in order.
func (s *TextSpan) VisitChildren(visitor InlineSpanVisitor) bool {
	if s.Text != nil && !visitor(s.outer()) {
		return false
	}
	for _, child := range s.Children {
		if !child.VisitChildren(visitor) {
			return false
		}
	}
	return true
}

// VisitDirectChildren calls visitor on each child.
func (s *TextSpan) VisitDirectChildren(visitor InlineSpanVisitor) bool {
	for _, child := range s.Children {
		if !visitor(child) {
			return false
		}
	}
	return true
}

// GetSpanForPositionVisitor returns this span if it contains position
// (offset is the start of its text in the flattened string).
func (s *TextSpan) GetSpanForPositionVisitor(position ui.TextPosition, offset *Accumulator) InlineSpan {
	if s.Text == nil || *s.Text == "" {
		return nil
	}
	length := dart.StringLength(*s.Text)
	affinity := position.Affinity
	targetOffset := position.Offset
	endOffset := offset.Value() + length
	if offset.Value() == targetOffset && affinity == ui.TextAffinityDownstream ||
		offset.Value() < targetOffset && targetOffset < endOffset ||
		endOffset == targetOffset && affinity == ui.TextAffinityUpstream {
		return s.outer()
	}
	offset.Increment(length)
	return nil
}

// ComputeToPlainText writes the span's text (or its semantics label) and
// its children's to buffer.
func (s *TextSpan) ComputeToPlainText(buffer *strings.Builder, opts ...ToPlainTextOptions) {
	o := plainTextOptions(opts)
	includeSemanticsLabels, _ := o.resolve()
	if !s.outer().DebugAssertIsValid() {
		panic(dart.NewAssertionError(nil))
	}
	if s.SemanticsLabel != nil && includeSemanticsLabels {
		buffer.WriteString(*s.SemanticsLabel)
	} else if s.Text != nil {
		buffer.WriteString(*s.Text)
	}
	for _, child := range s.Children {
		child.ComputeToPlainText(buffer, o)
	}
}

// TextSpanComputeSemanticsInformationOptions are TextSpan's extra named
// parameters of computeSemanticsInformation.
type TextSpanComputeSemanticsInformationOptions struct {
	InheritedLocale   *ui.Locale
	InheritedSpellOut bool
}

// ComputeSemanticsInformation is computeSemanticsInformation(collector).
func (s *TextSpan) ComputeSemanticsInformation(collector *[]InlineSpanSemanticsInformation) {
	s.ComputeSemanticsInformationInherited(collector, TextSpanComputeSemanticsInformationOptions{})
}

// ComputeSemanticsInformationInherited is TextSpan's
// computeSemanticsInformation(collector, {inheritedLocale,
// inheritedSpellOut}).
func (s *TextSpan) ComputeSemanticsInformationInherited(collector *[]InlineSpanSemanticsInformation, o TextSpanComputeSemanticsInformationOptions) {
	if !s.outer().DebugAssertIsValid() {
		panic(dart.NewAssertionError(nil))
	}
	effectiveLocale := s.Locale
	if effectiveLocale == nil {
		effectiveLocale = o.InheritedLocale
	}
	effectiveSpellOut := o.InheritedSpellOut
	if s.SpellOut != nil {
		effectiveSpellOut = *s.SpellOut
	}
	if s.Text != nil {
		var textLength int
		if s.SemanticsLabel != nil {
			textLength = dart.StringLength(*s.SemanticsLabel)
		} else {
			textLength = dart.StringLength(*s.Text)
		}
		attributes := []ui.StringAttribute{}
		if effectiveSpellOut && textLength > 0 {
			attributes = append(attributes, ui.NewSpellOutStringAttribute(ui.NewTextRange(0, textLength)))
		}
		if effectiveLocale != nil && textLength > 0 {
			attributes = append(attributes, ui.NewLocaleStringAttribute(ui.NewTextRange(0, textLength), *effectiveLocale))
		}
		*collector = append(*collector, NewInlineSpanSemanticsInformation(*s.Text, InlineSpanSemanticsInformationOptions{
			StringAttributes:    attributes,
			SemanticsLabel:      s.SemanticsLabel,
			SemanticsIdentifier: s.SemanticsIdentifier,
			Recognizer:          s.Recognizer,
		}))
	}
	for _, child := range s.Children {
		if ts, ok := child.(textSpanLike); ok {
			ts.ComputeSemanticsInformationInherited(collector, TextSpanComputeSemanticsInformationOptions{
				InheritedLocale: effectiveLocale, InheritedSpellOut: effectiveSpellOut,
			})
		} else {
			child.ComputeSemanticsInformation(collector)
		}
	}
}

// textSpanLike is Dart's `is TextSpan` (a TextSpan or a type embedding
// it).
type textSpanLike interface {
	InlineSpan
	ComputeSemanticsInformationInherited(collector *[]InlineSpanSemanticsInformation, o TextSpanComputeSemanticsInformationOptions)
}

// CodeUnitAtVisitor returns the code unit at index if it is in this span's
// text, and advances offset past it.
func (s *TextSpan) CodeUnitAtVisitor(index int, offset *Accumulator) *int {
	if s.Text == nil {
		return nil
	}
	localOffset := index - offset.Value()
	if !(localOffset >= 0) {
		panic(dart.NewAssertionError(nil))
	}
	length := dart.StringLength(*s.Text)
	offset.Increment(length)
	if localOffset < length {
		v := dart.CodeUnitAt(*s.Text, localOffset)
		return &v
	}
	return nil
}

// DebugAssertIsValid checks the children.
func (s *TextSpan) DebugAssertIsValid() bool {
	for _, child := range s.Children {
		if !child.DebugAssertIsValid() {
			panic(dart.NewAssertionError(nil))
		}
	}
	return s.InlineSpanBase.DebugAssertIsValid()
}

func styleCompare(a, b *TextStyle) RenderComparison { return a.CompareTo(b) }

// CompareTo describes the difference between this span and other in terms
// of their effect on rendering.
func (s *TextSpan) CompareTo(other InlineSpan) RenderComparison {
	self := s.outer()
	if self == other {
		return RenderComparisonIdentical
	}
	if !sameRuntimeType(self, other) {
		return RenderComparisonLayout
	}
	textSpan := asTextSpan(other)
	if !ptrEq(textSpan.Text, s.Text) || !sameLength(s.Children, textSpan.Children) ||
		(s.Style() == nil) != (textSpan.Style() == nil) {
		return RenderComparisonLayout
	}
	result := RenderComparisonMetadata
	if s.Recognizer == textSpan.Recognizer {
		result = RenderComparisonIdentical
	}
	if s.Style() != nil {
		candidate := styleCompare(s.Style(), textSpan.Style())
		if candidate > result {
			result = candidate
		}
		if result == RenderComparisonLayout {
			return result
		}
	}
	for index := range s.Children {
		candidate := s.Children[index].CompareTo(textSpan.Children[index])
		if candidate > result {
			result = candidate
		}
		if result == RenderComparisonLayout {
			return result
		}
	}
	return result
}

// sameLength is `a?.length != b?.length` negated (nil is null).
func sameLength(a, b []InlineSpan) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return len(a) == len(b)
}

// asTextSpan returns the TextSpan inside a span of TextSpan's runtime type
// (a *TextSpan, or a type embedding one).
func asTextSpan(span InlineSpan) *TextSpan {
	switch v := span.(type) {
	case *TextSpan:
		return v
	case interface{ textSpan() *TextSpan }:
		return v.textSpan()
	}
	panic("painting: not a TextSpan")
}

func (s *TextSpan) textSpan() *TextSpan { return s }

func mouseCursorEqual(a, b services.MouseCursor) bool {
	if e, ok := a.(interface {
		Equal(services.MouseCursor) bool
	}); ok {
		return e.Equal(b)
	}
	return a == b
}

func spansEqual(a, b []InlineSpan) bool {
	if (a == nil) != (b == nil) || len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// Equal is TextSpan's ==: the same runtime type, style, text, recognizer
// (by identity), labels, callbacks (dart.FunctionsEqual), cursor and
// children.
func (s *TextSpan) Equal(other InlineSpan) bool {
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
	if !s.InlineSpanBase.Equal(other) {
		return false
	}
	o := asTextSpan(other)
	return ptrEq(o.Text, s.Text) &&
		o.Recognizer == s.Recognizer &&
		ptrEq(o.SemanticsLabel, s.SemanticsLabel) &&
		ptrEq(o.SemanticsIdentifier, s.SemanticsIdentifier) &&
		funcsEqual(s.onEnter, o.onEnter) &&
		funcsEqual(s.onExit, o.onExit) &&
		mouseCursorEqual(s.MouseCursor, o.MouseCursor) &&
		spansEqual(o.Children, s.Children)
}

func funcsEqual[F any](a, b F) bool { return dart.FunctionsEqual(a, b) }

// ToStringShort is objectRuntimeType(this, 'TextSpan').
func (s *TextSpan) ToStringShort() string { return foundation.ObjectRuntimeType(s.outer(), "TextSpan") }

// DebugFillProperties adds the text, the recognizer, the callbacks, the
// cursor and the semantics properties.
func (s *TextSpan) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.InlineSpanBase.DebugFillProperties(properties)
	null := foundation.Null
	showName := false
	properties.Add(foundation.NewStringProperty("text", s.Text, foundation.StringPropertyOptions{ShowName: &showName, DefaultValue: null}))
	if s.Style() == nil && s.Text == nil && s.Children == nil {
		properties.Add(foundation.NewDiagnosticsNodeMessage("(empty)"))
	}
	var recognizerDescription *string
	if s.Recognizer != nil {
		recognizerDescription = strPtr(foundation.ObjectRuntimeType(s.Recognizer, ""))
	}
	properties.Add(foundation.NewDiagnosticsProperty("recognizer", s.Recognizer,
		foundation.DiagnosticsPropertyOptions{Description: recognizerDescription, DefaultValue: null}))
	var enter, exit any
	if s.onEnter != nil {
		enter = s.onEnter
	}
	if s.onExit != nil {
		exit = s.onExit
	}
	properties.Add(foundation.NewFlagsSummary("callbacks", []dart.MapEntry[string, any]{
		{Key: "enter", Value: enter}, {Key: "exit", Value: exit},
	}))
	properties.Add(foundation.NewDiagnosticsProperty("mouseCursor", s.Cursor(),
		foundation.DiagnosticsPropertyOptions{DefaultValue: services.MouseCursorDefer}))
	if s.SemanticsLabel != nil {
		properties.Add(foundation.NewStringProperty("semanticsLabel", s.SemanticsLabel))
	}
	if s.SemanticsIdentifier != nil {
		properties.Add(foundation.NewStringProperty("semanticsIdentifier", s.SemanticsIdentifier))
	}
}

// DebugDescribeChildren is the children's diagnostics nodes.
func (s *TextSpan) DebugDescribeChildren() []foundation.DiagnosticsNode {
	out := []foundation.DiagnosticsNode{}
	for _, child := range s.Children {
		out = append(out, child.ToDiagnosticsNode())
	}
	return out
}
