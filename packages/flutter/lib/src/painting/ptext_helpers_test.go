package painting_test

// Helpers shared by the text_painter tests (Go side of the top-level
// functions of text_painter_test.dart, plus small builders).

import (
	"strings"
	"sync"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/characters"
)

var ptextFontsOnce sync.Once

// ptextFonts turns on flutter_tester's test fonts (FlutterTest for every
// family but Ahem and Cough), as `flutter test` runs Dart's tests.
func ptextFonts() { ptextFontsOnce.Do(ui.EngineUseTestFonts) }

func sp(s string) *string                      { return &s }
func bp(v bool) *bool                          { return &v }
func tdp(d ui.TextDirection) *ui.TextDirection { return &d }
func tap(a ui.TextAlign) *ui.TextAlign         { return &a }

var (
	ltr = ui.TextDirectionLtr
	rtl = ui.TextDirectionRtl
)

// span is TextSpan(text: text, style: style).
func span(text string, style ...*TextStyle) *TextSpan {
	o := TextSpanOptions{Text: &text}
	if len(style) > 0 {
		o.Style = style[0]
	}
	return NewTextSpan(o)
}

// spans is TextSpan(children: children, style: style).
func spans(style *TextStyle, children ...InlineSpan) *TextSpan {
	if children == nil {
		children = []InlineSpan{}
	}
	return NewTextSpan(TextSpanOptions{Children: children, Style: style})
}

func newPainter(dir ui.TextDirection) *TextPainter {
	ptextFonts()
	return NewTextPainter(TextPainterOptions{TextDirection: &dir})
}

func maxW(w float64) TextPainterLayoutOptions { return TextPainterLayoutOptions{MaxWidth: &w} }

func minMax(lo, hi float64) TextPainterLayoutOptions {
	return TextPainterLayoutOptions{MinWidth: lo, MaxWidth: &hi}
}

func pos(offset int) ui.TextPosition { return ui.NewTextPosition(offset) }

func upstream(offset int) ui.TextPosition {
	return ui.NewTextPosition(offset, ui.TextAffinityUpstream)
}

func sel(base, extent int) services.TextSelection { return services.NewTextSelection(base, extent) }

func caret(p *TextPainter, position ui.TextPosition) ui.Offset {
	return p.GetOffsetForCaret(position, ui.RectZero)
}

func expectEq[T comparable](t *testing.T, got, want T, what ...any) {
	t.Helper()
	if got != want {
		t.Errorf("%v: got %v, want %v", what, got, want)
	}
}

func expectBoxes(t *testing.T, got, want []ui.TextBox, what ...any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%v: got %v, want %v", what, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%v: got %v, want %v", what, got, want)
			return
		}
	}
}

func box(l, t, r, b float64, d ui.TextDirection) ui.TextBox {
	return ui.NewTextBoxFromLTRBD(l, t, r, b, d)
}

// panicOf runs f and returns the recovered panic (nil if none).
func panicOf(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return nil
}

func panicText(r any) string {
	switch e := r.(type) {
	case nil:
		return ""
	case interface{ ToStringDeep() string }:
		return e.ToStringDeep()
	case error:
		return e.Error()
	}
	return dart.ToString(r)
}

// checkCaretOffsetsLtrAt is _checkCaretOffsetsLtrAt.
func checkCaretOffsetsLtrAt(t *testing.T, text string, boundaries []int) {
	t.Helper()
	expectEq(t, boundaries[0], 0, "first boundary")
	expectEq(t, boundaries[len(boundaries)-1], dart.StringLength(text), "last boundary")
	painter := newPainter(ltr)
	// Lay out the string up to each boundary, and record the width.
	var prefixWidths []float64
	for _, boundary := range boundaries {
		painter.SetText(span(dart.Substring(text, 0, boundary)))
		painter.Layout()
		prefixWidths = append(prefixWidths, painter.Width())
	}
	caretOffset := func(offset int) float64 { return caret(painter, pos(offset)).Dx }
	for i, b := range boundaries {
		if got := caretOffset(b); got != prefixWidths[i] {
			t.Errorf("%q: caret at %d = %v, want %v", text, b, got, prefixWidths[i])
		}
	}
	lastOffset := caretOffset(0)
	for i := 1; i <= dart.StringLength(text); i++ {
		offset := caretOffset(i)
		if offset < lastOffset {
			t.Errorf("%q: caret at %d = %v < %v", text, i, offset, lastOffset)
		}
		lastOffset = offset
	}
	painter.Dispose()
}

// checkCaretOffsetsLtr is checkCaretOffsetsLtr: caret offsets at every
// grapheme boundary equal the prefix widths.
func checkCaretOffsetsLtr(t *testing.T, text string) {
	t.Helper()
	var characterBoundaries []int
	r := characters.NewCharacterRangeAt(text, 0)
	for {
		characterBoundaries = append(characterBoundaries, dart.StringLength(r.Current()))
		if r.StringAfterLength() <= 0 {
			break
		}
		r.ExpandNext()
	}
	checkCaretOffsetsLtrAt(t, text, characterBoundaries)
}

// checkCaretOffsetsLtrFromPieces is checkCaretOffsetsLtrFromPieces.
func checkCaretOffsetsLtrFromPieces(t *testing.T, clusters []string) {
	t.Helper()
	var buffer strings.Builder
	boundaries := []int{0}
	for _, c := range clusters {
		buffer.WriteString(c)
		boundaries = append(boundaries, dart.StringLength(buffer.String()))
	}
	checkCaretOffsetsLtrAt(t, buffer.String(), boundaries)
}

// caretOffsetsForTextSpan is caretOffsetsForTextSpan.
func caretOffsetsForTextSpan(t *testing.T, dir ui.TextDirection, text *TextSpan) []float64 {
	t.Helper()
	painter := newPainter(dir)
	painter.SetText(text)
	painter.Layout()
	length := dart.StringLength(text.ToPlainText())
	result := make([]float64, length+1)
	for offset := range result {
		result[offset] = caret(painter, pos(offset)).Dx
	}
	if dir == ui.TextDirectionLtr {
		expectEq(t, result[length], painter.Width(), "end caret")
	} else {
		expectEq(t, result[0], painter.Width(), "start caret")
	}
	painter.Dispose()
	return result
}

func expectFloats(t *testing.T, got, want []float64, what ...any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%v: got %v, want %v", what, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%v: got %v, want %v", what, got, want)
			return
		}
	}
}

// testPlaceholderSpan stands for widgets' WidgetSpan: a PlaceholderSpan
// that adds a placeholder with the dimensions TextPainter was given
// (WidgetSpan.build), counting placeholders through the build options.
type testPlaceholderSpan struct {
	PlaceholderSpanBase
}

func newPlaceholder(alignment ...ui.PlaceholderAlignment) *testPlaceholderSpan {
	s := &testPlaceholderSpan{}
	o := PlaceholderSpanOptions{}
	if len(alignment) > 0 {
		o.Alignment = &alignment[0]
	}
	s.InitPlaceholderSpan(s, o)
	return s
}

func newPlaceholderStyled(style *TextStyle) *testPlaceholderSpan {
	s := &testPlaceholderSpan{}
	s.InitPlaceholderSpan(s, PlaceholderSpanOptions{Style: style})
	return s
}

// placeholderIndex is the number of placeholders a builder has seen
// (WidgetSpan uses builder.placeholderCount).
func (s *testPlaceholderSpan) Build(builder ui.ParagraphBuilder, opts ...InlineSpanBuildOptions) {
	var o InlineSpanBuildOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	textScaler := o.TextScaler
	if textScaler == nil {
		textScaler = TextScalerNoScaling
	}
	if s.Style() != nil {
		builder.PushStyle(s.Style().GetTextStyle(TextStyleGetTextStyleOptions{TextScaler: textScaler}))
	}
	index := builder.PlaceholderCount()
	if !(index < len(o.Dimensions)) {
		panic(dart.NewAssertionError(nil))
	}
	d := o.Dimensions[index]
	builder.AddPlaceholder(d.Size.Width, d.Size.Height, s.Alignment(), ui.AddPlaceholderOptions{
		Baseline: d.Baseline, BaselineOffset: d.BaselineOffset,
	})
	if s.Style() != nil {
		builder.Pop()
	}
}

func (s *testPlaceholderSpan) VisitChildren(visitor InlineSpanVisitor) bool { return visitor(s) }
func (s *testPlaceholderSpan) VisitDirectChildren(InlineSpanVisitor) bool   { return true }
func (s *testPlaceholderSpan) GetSpanForPositionVisitor(position ui.TextPosition, offset *Accumulator) InlineSpan {
	if position.Offset == offset.Value() {
		return s
	}
	offset.Increment(1)
	return nil
}
func (s *testPlaceholderSpan) CodeUnitAtVisitor(index int, offset *Accumulator) *int {
	localOffset := index - offset.Value()
	offset.Increment(1)
	if localOffset == 0 {
		v := PlaceholderSpanPlaceholderCodeUnit
		return &v
	}
	return nil
}
func (s *testPlaceholderSpan) CompareTo(other InlineSpan) RenderComparison {
	if InlineSpan(s) == other {
		return RenderComparisonIdentical
	}
	return RenderComparisonLayout
}
func (s *testPlaceholderSpan) DebugAssertIsValid() bool { return true }
