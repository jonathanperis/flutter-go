package painting_test

// Port of packages/flutter/test/painting/text_painter_test.dart (Flutter
// 3.50.1), under flutter_tester's test fonts (ui.EngineUseTestFonts).
// WidgetSpan (widgets) is stood in for by testPlaceholderSpan
// (ptext_helpers_test.go), which builds placeholders as WidgetSpan does.
//
// Skipped:
//   - "TextPainter caret Devanagari test, full strength" and "TextPainter
//     intrinsic dimensions": skipped upstream (skip: true).
//   - Tests skipped upstream on non-web/Skia (`skip: isBrowser ...` with
//     the opposite condition) are ported only where they run on the VM.
//   - "TextPainter dispatches memory events": needs leak_tracker's
//     memory-event matcher (FlutterMemoryAllocations dispatch is covered by
//     foundation's tests; here the creation/disposal dispatch is checked
//     with a listener instead, see TestTextPainterDispatchesMemoryEvents).
//   - hashCode assertions (Go has no hashCode).
//   - Tests that pump widgets (RenderParagraph, Text, Directionality) need
//     rendering/widgets: listed where they occur.

import (
	"math"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestTextPainterCaret(t *testing.T) {
	painter := newPainter(ltr)
	text := "A"
	checkCaretOffsetsLtr(t, text)
	painter.SetText(span(text))
	painter.Layout()
	expectEq(t, caret(painter, pos(0)).Dx, 0.0)
	expectEq(t, caret(painter, pos(dart.StringLength(text))).Dx, painter.Width())

	// A character encoded as a surrogate pair.
	text = "A\U0001F600"
	checkCaretOffsetsLtr(t, text)
	painter.SetText(span(text))
	painter.Layout()
	expectEq(t, caret(painter, pos(dart.StringLength(text))).Dx, painter.Width())

	// Characters in Unicode-Zs that SkParagraph treats as whitespace.
	verify := func(character string) {
		t.Helper()
		reason := "character " + character
		text = "A" + character
		checkCaretOffsetsLtr(t, text)
		painter.SetText(span(text))
		painter.Layout()
		expectEq(t, caret(painter, pos(0)).Dx, 0.0, reason)
		expectEq(t, caret(painter, pos(1)).Dx, 14.0, reason)
		expectEq(t, caret(painter, pos(dart.StringLength(text))).Dx, painter.Width(), reason)
		painter.Layout(maxW(14.0))
		lines := painter.ComputeLineMetrics()
		expectEq(t, len(lines), 1, reason)
		expectEq(t, lines[0].Width, 14.0, reason)
	}
	for _, c := range []string{" ", "　", " ", " ", " ", " ", " ", " ",
		" ", " ", " ", " ", " ", " "} {
		verify(c)
	}
	painter.Dispose()
}

func placeholderDims(w, h float64, baselineOffset float64) PlaceholderDimensions {
	return PlaceholderDimensions{Size: ui.NewSize(w, h), BaselineOffset: &baselineOffset,
		Alignment: ui.PlaceholderAlignmentBottom}
}

func TestTextPainterCaretWithWidgetSpan(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(spans(nil, span("before"), newPlaceholder(), span("after")))
	painter.SetPlaceholderDimensions([]PlaceholderDimensions{placeholderDims(50, 30, 25)})
	painter.Layout()
	o := caret(painter, pos(dart.StringLength(painter.Text().ToPlainText())))
	expectEq(t, o.Dx, painter.Width())
	painter.Dispose()
}

func TestTextPainterNullText(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(spans(nil, span("B"), span("C")))
	painter.Layout()
	expectEq(t, caret(painter, pos(0)).Dx, 0.0)
	expectEq(t, caret(painter, pos(1)).Dx, painter.Width()/2)
	expectEq(t, caret(painter, pos(2)).Dx, painter.Width())

	painter.SetText(spans(nil))
	painter.Layout()
	expectEq(t, caret(painter, pos(0)).Dx, 0.0)
	expectEq(t, caret(painter, pos(1)).Dx, 0.0)
	painter.Dispose()
}

func TestTextPainterCaretEmoji(t *testing.T) {
	painter := newPainter(ltr)
	const text = "👩‍👩‍👦👩‍👩‍👧‍👧👏🏽"
	checkCaretOffsetsLtr(t, text)
	painter.SetText(span(text))
	painter.Layout(maxW(10000))
	expectEq(t, dart.StringLength(text), 23)
	expectEq(t, caret(painter, pos(0)).Dx, 0.0)
	expectEq(t, caret(painter, pos(23)).Dx, painter.Width())
	want := []float64{0, 42, 42, 42, 42, 42, 42, 42, 42, 98, 98, 98, 98, 98, 98, 98, 98, 98, 98, 98, 126, 126, 126, 126}
	for i := 1; i <= 23; i++ {
		expectEq(t, caret(painter, pos(i)).Dx, want[i], "offset", i)
	}
	painter.Dispose()
}

func TestTextPainterCaretEmojiSingleLong(t *testing.T) {
	for _, s := range []string{"👩‍🚀", "👩‍❤️‍💋‍👩", "👨‍👩‍👦‍👦", "👨🏾‍🤝‍👨🏻", "👨‍👦", "👩‍👦", "🏌🏿‍♀️", "🏊‍♀️", "🏄🏻‍♂️",
		"🇺🇳", "👩‍❤️‍👨"} {
		checkCaretOffsetsLtr(t, s)
	}
}

func TestTextPainterCaretEmojiLettersThenEmoji(t *testing.T) {
	for _, s := range []string{"a👩‍🚀", "ab👩‍🚀", "abc👩‍🚀", "abcd👩‍🚀"} {
		checkCaretOffsetsLtr(t, s)
	}
}

func TestTextPainterCaretZalgo(t *testing.T) {
	checkCaretOffsetsLtr(t, "Z͉̳̺ͥͬ̾a̴͕̲̒̒͌̋ͪl̨͎̰̘͉̟ͤ̀̈̚͜g͕͔̤͖̟̒͝ͅo̵̡̡̼͚̐ͯ̅ͪ̆ͣ̚")
}

func TestTextPainterCaretDevanagari(t *testing.T) {
	checkCaretOffsetsLtrFromPieces(t, []string{"प्रा", "प्त", " ", "व", "र्ण", "न", " ", "प्र", "व्रु", "ति"})
}

func boldStyle() *TextStyle { return NewTextStyle(TextStyleOptions{FontWeight: &ui.FontWeightBold}) }

func TestTextPainterCaretEmojiLtrSeparateBoxes(t *testing.T) {
	expectFloats(t, caretOffsetsForTextSpan(t, ltr, spans(nil, span("👩‍🚀", NewTextStyle()), span(" words", boldStyle()))),
		[]float64{0, 28, 28, 28, 28, 28, 42, 56, 70, 84, 98, 112})
	expectFloats(t, caretOffsetsForTextSpan(t, ltr, spans(nil, span("words ", boldStyle()), span("👩‍🚀", NewTextStyle()))),
		[]float64{0, 14, 28, 42, 56, 70, 84, 112, 112, 112, 112, 112})
}

func TestTextPainterCaretEmojiRtlSeparateBoxes(t *testing.T) {
	expectFloats(t, caretOffsetsForTextSpan(t, rtl, spans(nil, span("👩‍🚀", NewTextStyle()), span(" מילים", boldStyle()))),
		[]float64{112, 84, 84, 84, 84, 84, 70, 56, 42, 28, 14, 0})
	expectFloats(t, caretOffsetsForTextSpan(t, rtl, spans(nil, span("מילים ", boldStyle()), span("👩‍🚀", NewTextStyle()))),
		[]float64{112, 98, 84, 70, 56, 42, 28, 0, 0, 0, 0, 0})
}

func TestTextPainterCaretCenterSpace(t *testing.T) {
	painter := newPainter(ltr)
	const text = "test text with space at end   "
	painter.SetText(span(text))
	painter.SetTextAlign(ui.TextAlignCenter)
	painter.Layout()
	expectEq(t, caret(painter, pos(0)).Dx, 21.0)
	// The end of the line is 441, but the width is only 420, so the cursor
	// is stopped there without overflowing.
	expectEq(t, caret(painter, pos(len(text))).Dx, painter.Width())
	expectEq(t, caret(painter, pos(1)).Dx, 35.0)
	expectEq(t, caret(painter, pos(2)).Dx, 49.0)
	painter.Dispose()
}

func TestTextPainterCaretHeightAndLineHeight(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetStrutStyle(NewStrutStyle(StrutStyleOptions{FontSize: fp(50.0)}))
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{Height: fp(1.0)})))
	painter.Layout()
	expectEq(t, painter.GetFullHeightForCaret(pos(0), ui.RectZero), 50.0)
	painter.Dispose()
}

func TestUpstreamDownstreamSameLineSameBidiRun(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("aa"))
	painter.Layout()
	largeRect := ui.OffsetZero.And(ui.NewSizeSquare(5))
	expectEq(t, painter.GetOffsetForCaret(pos(1), largeRect), painter.GetOffsetForCaret(upstream(1), largeRect))
}

func TestTrailingNewlines(t *testing.T) {
	const fontSize = 14.0
	ptextFonts()
	painter := NewTextPainter()
	largeRect := ui.OffsetZero.And(ui.NewSizeSquare(5))
	text := "a    "
	painter.SetText(span(text))
	painter.SetTextDirection(tdp(ltr))
	painter.Layout(minMax(1000, 1000))
	expectEq(t, painter.GetOffsetForCaret(pos(dart.StringLength(text)), largeRect).Dx,
		float64(dart.StringLength(text))*fontSize)

	text = "ل    "
	painter.SetText(span(text))
	painter.SetTextDirection(tdp(rtl))
	painter.Layout(minMax(1000, 1000))
	expectEq(t, painter.GetOffsetForCaret(pos(dart.StringLength(text)), largeRect).Dx,
		1000-float64(dart.StringLength(text))*fontSize-largeRect.Width())
}

func TestEndOfTextCaretPlusOneBidiLevel(t *testing.T) {
	const fontSize = 14.0
	ptextFonts()
	painter := NewTextPainter()
	largeRect := ui.OffsetZero.And(ui.NewSizeSquare(5))
	painter.SetText(span("aل"))
	painter.SetTextDirection(tdp(ltr))
	painter.Layout(minMax(1000, 1000))
	expectEq(t, painter.GetOffsetForCaret(pos(0), largeRect).Dx, 0.0)
	expectEq(t, painter.GetOffsetForCaret(pos(1), largeRect).Dx, fontSize*2-largeRect.Width())
	expectEq(t, painter.GetOffsetForCaret(pos(2), largeRect).Dx, fontSize*2)
}

func TestHandlesNewlinesProperly(t *testing.T) {
	painter := newPainter(ltr)
	const a = 14.0 // square size of "a" character
	both := func(text string, offset int, dx, dy float64) {
		t.Helper()
		for _, p := range []ui.TextPosition{pos(offset), upstream(offset)} {
			o := caret(painter, p)
			if o.Dx != dx || o.Dy != dy {
				t.Errorf("%q %v: got %v, want (%v, %v)", text, p, o, dx, dy)
			}
		}
	}
	set := func(text string, opts ...TextPainterLayoutOptions) {
		painter.SetText(span(text))
		painter.Layout(opts...)
	}

	// A plain one-line string is the same for either affinity.
	set("aaa")
	for offset := 0; offset <= 3; offset++ {
		both("aaa", offset, a*float64(offset), 0)
	}

	// Explicit newlines: the caret is where offset says, whatever the
	// affinity.
	set("\n\n")
	both("\n\n", 0, 0, 0)
	both("\n\n", 1, 0, a)
	both("\n\n", 2, 0, a*2)

	set("\naaa")
	both("\naaa", 0, 0, 0)
	both("\naaa", 1, 0, a)

	// When text wraps on its own, affinity disambiguates.
	text := "aaaaaaaa"
	set(text, maxW(100))
	o := caret(painter, pos(len(text)-1))
	expectEq(t, o, ui.NewOffset(0, a), "downstream at wrap")
	o = caret(painter, upstream(len(text)-1))
	expectEq(t, o, ui.NewOffset(98, 0), "upstream at wrap")

	// A trailing newline: the start of the next line for either affinity.
	set("aaa\n")
	both("aaa\n", 4, 0, a)

	// One-line right-aligned string, offset 0.
	painter.SetTextAlign(ui.TextAlignRight)
	set("aaa")
	expectEq(t, caret(painter, pos(0)), ui.NewOffset(0, 0), "right aligned")
	painter.SetTextAlign(ui.TextAlignLeft)

	// After a newline in the middle of a string.
	set("aaa\naaa")
	both("aaa\naaa", 4, 0, a)

	// Multiple trailing newlines (the first check uses the previous
	// layout, as Dart's test does).
	text = "aaa\n\n\n"
	both(text, 3, a*3, 0)
	set(text)
	both(text, 4, 0, a)
	both(text, 5, 0, a*2)
	both(text, 6, 0, a*3)

	// Multiple leading newlines.
	text = "\n\n\naaa"
	set(text)
	both(text, 3, 0, a*3)
	both(text, 2, 0, a*2)
	both(text, 1, 0, a)
	both(text, 0, 0, 0)
	painter.Dispose()
}

func TestCaretHeightReflectsRunHeightIfStrutDisabled(t *testing.T) {
	s := NewTextSpan(TextSpanOptions{Text: sp("M"), Style: NewTextStyle(TextStyleOptions{FontSize: fp(128)}),
		Children: []InlineSpan{
			span("M", NewTextStyle(TextStyleOptions{FontSize: fp(32)})),
			span("M", NewTextStyle(TextStyleOptions{FontSize: fp(64)})),
		}})
	painter := newPainter(ltr)
	painter.SetText(s)
	painter.Layout()
	h := func(p ui.TextPosition) float64 { return painter.GetFullHeightForCaret(p, ui.RectZero) }
	expectEq(t, h(upstream(0)), 128.0)
	expectEq(t, h(pos(0)), 128.0)
	expectEq(t, h(upstream(1)), 128.0)
	expectEq(t, h(pos(1)), 32.0)
	expectEq(t, h(upstream(2)), 32.0)
	expectEq(t, h(pos(2)), 64.0)
	expectEq(t, h(upstream(3)), 64.0)
	expectEq(t, h(pos(3)), 128.0)
	painter.Dispose()
}

func TestGetFullHeightForCaretDegenerateLayout(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr),
		Text: span("", NewTextStyle(TextStyleOptions{Height: fp(1.0), FontSize: fp(0.0)}))})
	painter.Layout()
	expectEq(t, painter.GetFullHeightForCaret(pos(0), ui.RectZero), painter.PreferredLineHeight())
	painter.Dispose()
}

func TestTextPainterErrorTest(t *testing.T) {
	painter := newPainter(ltr)
	r := panicOf(func() { painter.Paint(ui.NewCanvas(ui.NewPictureRecorder()), ui.OffsetZero) })
	e, ok := r.(dart.StateError)
	if !ok || !strings.Contains(e.Message, "TextPainter.paint called when text geometry was not yet calculated") {
		t.Errorf("got %v", r)
	}
	painter.Dispose()
}

func TestTextPainterRequiresTextDirection(t *testing.T) {
	ptextFonts()
	painter1 := NewTextPainter(TextPainterOptions{Text: span("")})
	if _, ok := panicOf(func() { painter1.Layout() }).(dart.StateError); !ok {
		t.Error("expected a StateError")
	}
	painter2 := NewTextPainter(TextPainterOptions{Text: span(""), TextDirection: tdp(rtl)})
	if r := panicOf(func() { painter2.Layout() }); r != nil {
		t.Errorf("unexpected %v", r)
	}
}

func noInherit(size float64) *TextStyle {
	return NewTextStyle(TextStyleOptions{Inherit: bp(false), FontSize: &size})
}

func TestTextPainterSize(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{Text: span("X", noInherit(123.0)), TextDirection: tdp(ltr)})
	painter.Layout()
	expectEq(t, painter.Size(), ui.NewSize(123, 123))
	painter.Dispose()
}

func TestTextPainterTextScaler(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{Text: span("X", noInherit(10.0)), TextDirection: tdp(ltr),
		TextScaler: NewTextScalerLinear(2.0)})
	painter.Layout()
	expectEq(t, painter.Size(), ui.NewSize(20, 20))
	painter.Dispose()
}

func TestTextPainterTextScalerNullStyle(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{Text: span("X"), TextDirection: tdp(ltr),
		TextScaler: NewTextScalerLinear(2.0)})
	painter.Layout()
	expectEq(t, painter.Size(), ui.NewSize(28, 28))
	painter.Dispose()
}

func TestTextPainterDefaultTextHeightIs14(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{Text: span("x"), TextDirection: tdp(ltr)})
	painter.Layout()
	expectEq(t, painter.PreferredLineHeight(), 14.0)
	expectEq(t, painter.Size(), ui.NewSize(14, 14))
	painter.Dispose()
}

func TestTextPainterSetsParagraphSizeFromRoot(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{Text: span("x", NewTextStyle(TextStyleOptions{FontSize: fp(100)})),
		TextDirection: tdp(ltr)})
	painter.Layout()
	expectEq(t, painter.PreferredLineHeight(), 100.0)
	expectEq(t, painter.Size(), ui.NewSize(100, 100))
	painter.Dispose()
}

var _ = math.Inf
