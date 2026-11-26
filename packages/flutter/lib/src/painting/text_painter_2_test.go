package painting_test

// Port of packages/flutter/test/painting/text_painter_test.dart (Flutter
// 3.50.1), continued (see text_painter_test.go for the skips).
// MockCanvasWithDrawParagraph is flutter_test's TestRecordingCanvas (its
// recorded drawParagraph call); Dart's MockCanvas (a Fake whose
// drawParagraph throws) is the same canvas, which never throws.

import (
	"math"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestTextPainterWidgetSpan(t *testing.T) {
	painter := newPainter(ltr)
	const text = "test"
	children := []InlineSpan{newPlaceholder(), span(text), newPlaceholder(), newPlaceholder(), span(text)}
	for i := 0; i < 11; i++ {
		children = append(children, newPlaceholder())
	}
	painter.SetText(NewTextSpan(TextSpanOptions{Text: sp(text), Children: children}))
	var dims []PlaceholderDimensions
	for i := 0; i < 14; i++ {
		w := 50.0
		if i == 12 {
			w = 51
		}
		dims = append(dims, placeholderDims(w, 30, 25))
	}
	painter.SetPlaceholderDimensions(dims)
	painter.Layout(maxW(500))

	for _, c := range [][2]float64{{1, 14}, {4, 56}, {5, 106}, {6, 120}, {10, 212}, {11, 262}, {12, 276}, {13, 290},
		{14, 304}, {15, 318}, {16, 368}, {17, 418}, {18, 0}, {19, 50}, {23, 250}} {
		expectEq(t, caret(painter, pos(int(c[0]))).Dx, c[1], "offset", c[0])
	}
	boxes := painter.InlinePlaceholderBoxes()
	expectEq(t, len(boxes), 14)
	for _, c := range []struct {
		i int
		b ui.TextBox
	}{
		{0, box(56, 0, 106, 30, ltr)}, {2, box(212, 0, 262, 30, ltr)}, {3, box(318, 0, 368, 30, ltr)},
		{4, box(368, 0, 418, 30, ltr)}, {5, box(418, 0, 468, 30, ltr)},
		// line should break here
		{6, box(0, 30, 50, 60, ltr)}, {7, box(50, 30, 100, 60, ltr)}, {10, box(200, 30, 250, 60, ltr)},
		{11, box(250, 30, 300, 60, ltr)}, {12, box(300, 30, 351, 60, ltr)}, {13, box(351, 30, 401, 60, ltr)},
	} {
		expectEq(t, boxes[c.i], c.b, "box", c.i)
	}
	painter.Dispose()
}

func TestTextPainterSetTextHeightBehaviorNull(t *testing.T) {
	painter := newPainter(ltr)
	thb := ui.NewTextHeightBehavior()
	painter.SetTextHeightBehavior(&thb)
	painter.SetTextHeightBehavior(nil)
	painter.Dispose()
}

func TestTextPainterSetHyphens(t *testing.T) {
	painter := newPainter(ltr)
	expectEq(t, painter.Hyphens(), ui.HyphensManual)
	painter.SetHyphens(ui.HyphensHidden)
	expectEq(t, painter.Hyphens(), ui.HyphensHidden)
	painter.Dispose()
}

// softHyphenPainter: "abc" + soft hyphen + "de" is 50 wide unbroken, so a
// max width of 49 breaks it at the soft hyphen; the first line is then 30
// wide, or 40 with a hyphen.
func softHyphenPainter(hyphens ui.Hyphens) *TextPainter {
	p := newPainter(ltr)
	p.SetTextWidthBasis(TextWidthBasisLongestLine)
	p.SetHyphens(hyphens)
	p.SetText(span("abc­de", NewTextStyle(TextStyleOptions{FontSize: fp(10.0)})))
	return p
}

func TestTextPainterRendersHyphenAtSoftHyphenBreak(t *testing.T) {
	manual := softHyphenPainter(ui.HyphensManual)
	manual.Layout(maxW(49))
	hidden := softHyphenPainter(ui.HyphensHidden)
	hidden.Layout(maxW(49))
	expectEq(t, len(manual.ComputeLineMetrics()), 2)
	expectEq(t, len(hidden.ComputeLineMetrics()), 2)
	expectEq(t, manual.Width(), 40.0)
	expectEq(t, hidden.Width(), 30.0)
	manual.Dispose()
	hidden.Dispose()
}

func TestChangingHyphensAfterLayoutChangesWidth(t *testing.T) {
	painter := softHyphenPainter(ui.HyphensHidden)
	painter.Layout(maxW(49))
	expectEq(t, painter.Width(), 30.0)
	painter.SetHyphens(ui.HyphensManual)
	painter.Layout(maxW(49))
	expectEq(t, painter.Width(), 40.0)
	painter.Dispose()
}

func TestTextPainterLineMetrics(t *testing.T) {
	painter := newPainter(ltr)
	const text = "test1\nhello line two really long for soft break\nfinal line 4"
	painter.SetText(span(text))
	painter.Layout(maxW(300))
	if !painter.Text().Equal(span(text)) {
		t.Error("text differs")
	}
	expectEq(t, painter.PreferredLineHeight(), 14.0)
	lines := painter.ComputeLineMetrics()
	if len(lines) != 4 {
		t.Fatalf("%d lines", len(lines))
	}
	hard := []bool{true, false, true, true}
	widths := []float64{70, 294, 266, 168}
	for i, l := range lines {
		want := ui.LineMetrics{HardBreak: hard[i], Ascent: 10.5, Descent: 3.5, UnscaledAscent: 10.5,
			Baseline: 10.5 + 14*float64(i), Height: 14, Width: widths[i], Left: 0, LineNumber: i}
		expectEq(t, l, want, "line", i)
	}
	painter.Dispose()
}

// insetsTopBottom is RelativeRect.fromSize(glyphBox, size)'s top and
// bottom.
func insetsTopBottom(glyph ui.Rect, size ui.Size) (float64, float64) {
	return glyph.Top, size.Height - glyph.Bottom
}

func firstGlyphRect(p *TextPainter) ui.Rect {
	return p.GetBoxesForSelection(sel(0, 1))[0].ToRect()
}

func TestTextPainterLineHeightHalfLeading(t *testing.T) {
	even := ui.TextLeadingDistributionEven
	painter := newPainter(ltr)
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{Height: fp(20), FontSize: fp(1), LeadingDistribution: &even})))
	painter.Layout()
	top, bottom := insetsTopBottom(firstGlyphRect(painter), painter.Size())
	expectEq(t, top, bottom)
	expectEq(t, top, (20-1)/2.0)
	painter.Dispose()
}

func TestTextPainterLineHeightHalfLeadingSmallHeight(t *testing.T) {
	even := ui.TextLeadingDistributionEven
	painter := newPainter(ltr)
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{Height: fp(0.1), FontSize: fp(10), LeadingDistribution: &even})))
	painter.Layout()
	top, bottom := insetsTopBottom(firstGlyphRect(painter), painter.Size())
	expectEq(t, top, bottom)
	expectEq(t, top, (1-10)/2.0)
	painter.Dispose()
}

func TestTextPainterLineHeightHalfLeadingWithTrim(t *testing.T) {
	even := ui.TextLeadingDistributionEven
	painter := newPainter(ltr)
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{Height: fp(0.1), FontSize: fp(10), LeadingDistribution: &even})))
	thb := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{ApplyHeightToFirstAscent: bp(false),
		ApplyHeightToLastDescent: bp(false)})
	painter.SetTextHeightBehavior(&thb)
	painter.Layout()
	glyph := firstGlyphRect(painter)
	expectEq(t, painter.Size(), glyph.Size())
	expectEq(t, glyph.TopLeft(), ui.OffsetZero)
	painter.Dispose()
}

func TestTextLeadingDistributionFallsBackToParagraphStyle(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{Height: fp(20), FontSize: fp(1)})))
	thb := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{LeadingDistribution: ui.TextLeadingDistributionEven})
	painter.SetTextHeightBehavior(&thb)
	painter.Layout()
	top, bottom := insetsTopBottom(firstGlyphRect(painter), painter.Size())
	expectEq(t, top, bottom)
	expectEq(t, top, (20-1)/2.0)
	painter.Dispose()
}

func TestTextLeadingDistributionNothingWithoutHeight(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{FontSize: fp(1)})))
	thb := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{LeadingDistribution: ui.TextLeadingDistributionEven})
	painter.SetTextHeightBehavior(&thb)
	painter.Layout()
	glyph := firstGlyphRect(painter)
	plain := ui.NewTextHeightBehavior()
	painter.SetTextHeightBehavior(&plain)
	painter.Layout()
	expectEq(t, firstGlyphRect(painter), glyph)
	painter.Dispose()
}

func TestTextPainterHandlesInvalidUTF16(t *testing.T) {
	var errorDetails *foundation.FlutterErrorDetails
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) { errorDetails = details }
	defer func() { foundation.FlutterErrorOnError = old }()

	painter := newPainter(ltr)
	const fontSize = 20.0
	painter.SetText(span(dart.StringFromCharCodes([]uint16{'H', 'e', 'l', 'l', 'o', 0xD83D, 'W', 'o', 'r', 'l', 'd'}),
		NewTextStyle(TextStyleOptions{FontSize: fp(fontSize)})))
	painter.Layout()
	// The layout should include one replacement character.
	expectEq(t, painter.Width(), fontSize)
	if errorDetails == nil || errorDetails.Exception == nil || !errorDetails.Silent {
		t.Errorf("error details: %v", errorDetails)
	}
	painter.Dispose()
}

func TestDiacritic(t *testing.T) {
	painter := newPainter(ltr)
	const text = "ฟห้"
	painter.SetText(span(text))
	painter.Layout()
	expectEq(t, caret(painter, upstream(dart.StringLength(text))).Dx, painter.Width())
	painter.Dispose()
}

func TestTextPainterLineMetricsUpdateAfterLayout(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("word1 word2 word3"))
	painter.Layout(maxW(80))
	expectEq(t, len(painter.ComputeLineMetrics()), 3)
	painter.Layout(maxW(1000))
	expectEq(t, len(painter.ComputeLineMetrics()), 1)
	painter.Dispose()
}

func TestTextPainterLineMetricsCachedWithPaintOffset(t *testing.T) {
	for _, textAlign := range ui.TextAlignValues {
		painter := newPainter(ltr)
		painter.SetTextAlign(textAlign)
		painter.SetTextWidthBasis(TextWidthBasisLongestLine)
		painter.SetText(span("word1 word2 word3", NewTextStyle(TextStyleOptions{FontSize: fp(10.0)})))
		painter.Layout(maxW(130))
		lines := painter.ComputeLineMetrics()
		expectEq(t, len(lines), 2, textAlign)
		// Identity of the list: the same backing array.
		again := painter.ComputeLineMetrics()
		expectEq(t, &again[0], &lines[0], textAlign, "identical")
		painter.Layout(maxW(1000))
		expectEq(t, len(painter.ComputeLineMetrics()), 1, textAlign)
		painter.Dispose()
	}
}

func TestTextPainterLineMetricsShiftedByPaintOffset(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetTextAlign(ui.TextAlignCenter)
	painter.SetTextWidthBasis(TextWidthBasisLongestLine)
	painter.SetText(span("aaaaaa\naaaaaa", NewTextStyle(TextStyleOptions{FontSize: fp(10.0)})))
	painter.Layout(maxW(100))
	expectEq(t, painter.Width(), 60.0)
	for _, m := range painter.ComputeLineMetrics() {
		expectEq(t, m.Left, 0.0)
	}
	painter.Dispose()
}

func flutterErrorMessage(r any) string {
	if e, ok := r.(*foundation.FlutterError); ok {
		return e.Message()
	}
	return ""
}

func TestTextPainterThrowsWithStackTraceWhenAccessingLayout(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("TEXT"))
	r := panicOf(func() { painter.GetPositionForOffset(ui.OffsetZero) })
	if !strings.Contains(flutterErrorMessage(r), "The TextPainter has never been laid out.") {
		t.Errorf("got %v", r)
	}
	if r := panicOf(func() {
		painter.Layout()
		painter.GetPositionForOffset(ui.OffsetZero)
	}); r != nil {
		t.Errorf("unexpected %v", r)
	}
	r = panicOf(func() {
		painter.MarkNeedsLayout()
		painter.GetPositionForOffset(ui.OffsetZero)
	})
	if !strings.Contains(flutterErrorMessage(r), "The calls that first invalidated the text layout were:") {
		t.Errorf("got %v", r)
	}
	painter.Dispose()
}

func threePlaceholderPainter() *TextPainter {
	painter := newPainter(ltr)
	painter.SetText(spans(nil, span("before"), newPlaceholder(), newPlaceholder(), newPlaceholder(), span("after")))
	return painter
}

func bottomDims(sizes ...ui.Size) []PlaceholderDimensions {
	var out []PlaceholderDimensions
	for _, s := range sizes {
		out = append(out, PlaceholderDimensions{Size: s, Alignment: ui.PlaceholderAlignmentBottom})
	}
	return out
}

func isNotLaidOutStateError(r any) bool {
	e, ok := r.(dart.StateError)
	return ok && strings.Contains(e.Message, "TextPainter.paint called when text geometry was not yet calculated")
}

func TestTextPainterRequiresLayoutAfterDifferentPlaceholderDimensions(t *testing.T) {
	painter := threePlaceholderPainter()
	painter.SetPlaceholderDimensions(bottomDims(ui.NewSize(30, 30), ui.NewSize(40, 30), ui.NewSize(50, 30)))
	painter.Layout()
	painter.SetPlaceholderDimensions(bottomDims(ui.NewSize(30, 30), ui.NewSize(40, 20), ui.NewSize(50, 30)))
	if r := panicOf(func() { painter.Paint(ft.NewTestRecordingCanvas(), ui.OffsetZero) }); !isNotLaidOutStateError(r) {
		t.Errorf("got %v", r)
	}
	painter.Dispose()
}

func TestTextPainterNoLayoutAfterIdenticalPlaceholderDimensions(t *testing.T) {
	painter := threePlaceholderPainter()
	painter.SetPlaceholderDimensions(bottomDims(ui.NewSize(30, 30), ui.NewSize(40, 30), ui.NewSize(50, 30)))
	painter.Layout()
	painter.SetPlaceholderDimensions(bottomDims(ui.NewSize(30, 30), ui.NewSize(40, 30), ui.NewSize(50, 30)))
	if r := panicOf(func() { painter.Paint(ft.NewTestRecordingCanvas(), ui.OffsetZero) }); isNotLaidOutStateError(r) {
		t.Errorf("got %v", r)
	}
	painter.Dispose()
}

func TestTextPainterDebugDisposed(t *testing.T) {
	painter := NewTextPainter()
	expectEq(t, painter.DebugDisposed(), false)
	painter.Dispose()
	expectEq(t, painter.DebugDisposed(), true)
}

func TestTextPainterAssertsIfDisposedTwice(t *testing.T) {
	painter := NewTextPainter()
	painter.Dispose()
	expectEq(t, painter.DebugDisposed(), true)
	if _, ok := panicOf(painter.Dispose).(dart.AssertionFailure); !ok {
		t.Error("expected an AssertionError")
	}
}

func TestTextPainterComputeWidth(t *testing.T) {
	ptextFonts()
	text := span("foobar")
	painter := NewTextPainter(TextPainterOptions{Text: text, TextDirection: tdp(ltr)})
	painter.Layout()
	expectEq(t, painter.Width(), TextPainterComputeWidth(text, ltr))
	painter.Layout(TextPainterLayoutOptions{MinWidth: 500})
	expectEq(t, painter.Width(), TextPainterComputeWidth(text, ltr, TextPainterComputeOptions{MinWidth: 500}))
	painter.Dispose()
}

func TestTextPainterComputeMaxIntrinsicWidth(t *testing.T) {
	ptextFonts()
	text := span("foobar")
	painter := NewTextPainter(TextPainterOptions{Text: text, TextDirection: tdp(ltr)})
	painter.Layout()
	expectEq(t, painter.MaxIntrinsicWidth(), TextPainterComputeMaxIntrinsicWidth(text, ltr))
	painter.Layout(TextPainterLayoutOptions{MinWidth: 500})
	expectEq(t, painter.MaxIntrinsicWidth(),
		TextPainterComputeMaxIntrinsicWidth(text, ltr, TextPainterComputeOptions{MinWidth: 500}))
	painter.Dispose()
}

func TestTextPainterGetWordBoundaryWorks(t *testing.T) {
	ptextFonts()
	const testCluster = "👨‍👩‍👦👨‍👩‍👦👨‍👩‍👦" // 8 * 3
	painter := NewTextPainter(TextPainterOptions{Text: span(testCluster), TextDirection: tdp(ltr)})
	painter.Layout()
	expectEq(t, painter.GetWordBoundary(pos(8)), ui.NewTextRange(8, 16))
}

func TestTextHeightBehaviorWithStrutOnEmptyParagraph(t *testing.T) {
	ptextFonts()
	style := NewTextStyle(TextStyleOptions{Height: fp(11), FontSize: fp(7)})
	simple := span("x", style)
	emptyString := span("", style)
	emptyParagraph := NewTextSpan(TextSpanOptions{Style: style})
	thb := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{ApplyHeightToFirstAscent: bp(false),
		ApplyHeightToLastDescent: bp(false)})
	painter := NewTextPainter(TextPainterOptions{
		TextDirection:      tdp(ltr),
		StrutStyle:         NewStrutStyleFromTextStyle(style, StrutStyleOptions{ForceStrutHeight: bp(true)}),
		TextHeightBehavior: &thb,
	})
	painter.SetText(simple)
	painter.Layout()
	height := painter.Height()
	for _, s := range []*TextSpan{simple, emptyString, emptyParagraph} {
		painter.SetText(s)
		painter.Layout()
		expectEq(t, painter.Height(), height, s)
		expectEq(t, painter.PreferredLineHeight(), height, s)
	}
}

func TestTextPainterPlainTextGetter(t *testing.T) {
	painter := newPainter(ltr)
	expectEq(t, painter.PlainText(), "")
	painter.SetText(spans(nil, span("before\n"), newPlaceholder(), span("after")))
	expectEq(t, painter.PlainText(), "before\n￼after")
	painter.SetPlaceholderDimensions(bottomDims(ui.NewSize(50, 30)))
	painter.Layout()
	expectEq(t, painter.PlainText(), "before\n￼after")
	painter.SetText(spans(nil, span("be\nfo\nre\n"), newPlaceholder(), span("af\nter")))
	expectEq(t, painter.PlainText(), "be\nfo\nre\n￼af\nter")
	painter.Layout()
	expectEq(t, painter.PlainText(), "be\nfo\nre\n￼af\nter")
	painter.Dispose()
}

// paintedParagraphOffset paints into a recording canvas and returns the
// drawParagraph offset's dx and the paragraph center's x (nil when
// nothing is drawn).
func paintedParagraphOffset(t *testing.T, painter *TextPainter) (offsetX, centerX *float64) {
	t.Helper()
	canvas := ft.NewTestRecordingCanvas()
	if r := panicOf(func() { painter.Paint(canvas, ui.OffsetZero) }); r != nil {
		t.Fatalf("paint: %v", r)
	}
	for _, inv := range canvas.Invocations {
		if inv.Invocation.MemberName == "drawParagraph" {
			paragraph := inv.Invocation.PositionalArguments[0].(ui.Paragraph)
			offset := inv.Invocation.PositionalArguments[1].(ui.Offset)
			ox, cx := offset.Dx, offset.Dx+paragraph.Width()/2
			return &ox, &cx
		}
	}
	return nil, nil
}

func TestTextPainterInfiniteWidthCentered(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetTextAlign(ui.TextAlignCenter)
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{FontSize: fp(10)})))

	painter.Layout(TextPainterLayoutOptions{MinWidth: math.Inf(1)})
	expectEq(t, painter.Width(), math.Inf(1))
	if _, cx := paintedParagraphOffset(t, painter); cx != nil {
		t.Errorf("centerX %v, want null", *cx)
	}
	painter.Layout()
	expectEq(t, painter.Width(), 10.0)
	if _, cx := paintedParagraphOffset(t, painter); cx == nil || *cx != 5 {
		t.Errorf("centerX %v", cx)
	}
	painter.Layout(TextPainterLayoutOptions{MinWidth: 100})
	expectEq(t, painter.Width(), 100.0)
	if _, cx := paintedParagraphOffset(t, painter); cx == nil || *cx != 50 {
		t.Errorf("centerX %v", cx)
	}
	painter.Dispose()
}

func TestTextPainterInfiniteWidthLtrJustified(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetTextAlign(ui.TextAlignJustify)
	painter.SetText(span("A", NewTextStyle(TextStyleOptions{FontSize: fp(10)})))
	for _, c := range []struct {
		opts  TextPainterLayoutOptions
		width float64
	}{{TextPainterLayoutOptions{MinWidth: math.Inf(1)}, math.Inf(1)}, {TextPainterLayoutOptions{}, 10},
		{TextPainterLayoutOptions{MinWidth: 100}, 100}} {
		painter.Layout(c.opts)
		expectEq(t, painter.Width(), c.width)
		if ox, _ := paintedParagraphOffset(t, painter); ox == nil || *ox != 0 {
			t.Errorf("offsetX %v", ox)
		}
	}
	painter.Dispose()
}

func TestLongestLineTextPainterRelayoutWhenMaxWidthChanges(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetTextAlign(ui.TextAlignJustify)
	painter.SetTextWidthBasis(TextWidthBasisLongestLine)
	painter.SetText(span(strings.Repeat("A", 100), NewTextStyle(TextStyleOptions{FontSize: fp(10)})))
	painter.Layout(maxW(1000))
	expectEq(t, painter.Width(), 1000.0)
	painter.Layout(maxW(100))
	expectEq(t, painter.Width(), 100.0)
	painter.Layout(maxW(1000))
	expectEq(t, painter.Width(), 1000.0)
}

func TestTextPainterLineBreakingDoesNotRound(t *testing.T) {
	ptextFonts()
	const fontSize = 1.25
	const text = "12345"
	total := float64(len(text)) * fontSize
	if math.Trunc(total) == total {
		t.Fatal("bad setup")
	}
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr),
		Text: span(text, NewTextStyle(TextStyleOptions{FontSize: fp(fontSize)}))})
	painter.Layout(maxW(total))
	expectEq(t, painter.MaxIntrinsicWidth(), total)
	metrics := painter.ComputeLineMetrics()
	if len(metrics) != 1 {
		t.Fatalf("%d lines", len(metrics))
	}
	expectEq(t, metrics[0].Width, total)
}

func TestStrutStyleAppliesWhenSpanHasNoStyle(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr), Text: NewTextSpan(),
		StrutStyle: NewStrutStyle(StrutStyleOptions{Height: fp(10), FontSize: fp(10)})})
	painter.Layout()
	expectEq(t, painter.Height(), 100.0)
}

func TestStrutStyleLeadingIsFontSizeMultiplier(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr), Text: NewTextSpan(),
		StrutStyle: NewStrutStyle(StrutStyleOptions{Height: fp(10), FontSize: fp(10), Leading: fp(2)})})
	painter.Layout()
	expectEq(t, painter.Height(), 100.0+20)
	// Top leading + scaled ascent.
	expectEq(t, painter.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic), 10+10*7.5)
}

func TestStrutNoHalfLeadingForceStrutHeight(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr),
		Text:       span("A", NewTextStyle(TextStyleOptions{FontSize: fp(20)})),
		StrutStyle: NewStrutStyle(StrutStyleOptions{Height: fp(10), FontSize: fp(10), ForceStrutHeight: bp(true)})})
	painter.Layout()
	expectEq(t, painter.Height(), 100.0)
	const baseline = 75.0
	expectBoxes(t, painter.GetBoxesForSelection(sel(0, 1)), []ui.TextBox{box(0, baseline-15, 20, baseline+5, ltr)})
}

func TestStrutHalfLeadingForceStrutHeight(t *testing.T) {
	ptextFonts()
	even := ui.TextLeadingDistributionEven
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr),
		Text: span("A", NewTextStyle(TextStyleOptions{FontSize: fp(20)})),
		StrutStyle: NewStrutStyle(StrutStyleOptions{Height: fp(10), FontSize: fp(10), ForceStrutHeight: bp(true),
			LeadingDistribution: &even})})
	painter.Layout()
	expectEq(t, painter.Height(), 100.0)
	const baseline = 45 + 7.5
	expectBoxes(t, painter.GetBoxesForSelection(sel(0, 1)), []ui.TextBox{box(0, baseline-15, 20, baseline+5, ltr)})
}

func TestForceStrutHeightAppliesToWidgetSpans(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr), Text: newPlaceholder(),
		StrutStyle: NewStrutStyle(StrutStyleOptions{Height: fp(10), FontSize: fp(10), ForceStrutHeight: bp(true)})})
	painter.SetPlaceholderDimensions(bottomDims(ui.NewSize(1000, 1000)))
	painter.Layout()
	expectEq(t, painter.Height(), 100.0)
}

func TestGetOffsetForCaretDecomposedCharacters(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr),
		Text: span("각", NewTextStyle(TextStyleOptions{FontSize: fp(10)}))})
	painter.Layout(maxW(1)) // Force the jamo characters to soft wrap.
	if r := panicOf(func() { painter.GetOffsetForCaret(pos(0), ui.RectZero) }); r != nil {
		t.Errorf("unexpected %v", r)
	}
}

func TestKTextHeightNoneUnsetsHeight(t *testing.T) {
	ptextFonts()
	painter := NewTextPainter(TextPainterOptions{TextDirection: tdp(ltr),
		Text: spans(NewTextStyle(TextStyleOptions{FontSize: fp(10), Height: fp(1000)}),
			span("A", NewTextStyle(TextStyleOptions{Height: fp(ui.KTextHeightNone)})))})
	painter.Layout()
	expectEq(t, painter.Height(), 10.0)
}

func TestDebugPaintTextLayoutBoxes(t *testing.T) {
	// ascent = 96, descent = 32
	s := NewTextSpan(TextSpanOptions{Text: sp("M"), Style: NewTextStyle(TextStyleOptions{FontSize: fp(128)}),
		Children: []InlineSpan{span("M", NewTextStyle(TextStyleOptions{FontSize: fp(64)}))}})
	painter := newPainter(ltr)
	painter.SetText(s)
	painter.Layout()
	r1 := ui.OffsetZero.And(ui.NewSizeSquare(128))
	r2 := ui.NewOffset(128, 96-48).And(ui.NewSizeSquare(64))
	ft.Expect(t, func(canvas ui.Canvas) {
		painter.DebugPaintTextLayoutBoxes = true
		painter.Paint(canvas, ui.OffsetZero)
		painter.DebugPaintTextLayoutBoxes = false
	}, ft.Paints().Rect(ft.PaintPatternRectOptions{Rect: &r1}).Rect(ft.PaintPatternRectOptions{Rect: &r2}))
}

// Dart's version uses leak_tracker's memoryEvents/areCreateAndDispose; here
// FlutterMemoryAllocations' listener sees the same two events.
func TestTextPainterDispatchesMemoryEvents(t *testing.T) {
	var events []foundation.ObjectEvent
	listener := func(e foundation.ObjectEvent) { events = append(events, e) }
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	painter := NewTextPainter()
	painter.Dispose()
	var mine []foundation.ObjectEvent
	for _, e := range events {
		if e.Object() == any(painter) {
			mine = append(mine, e)
		}
	}
	if len(mine) != 2 {
		t.Fatalf("events %v", mine)
	}
	created, ok := mine[0].(*foundation.ObjectCreated)
	if !ok || created.Library != "package:flutter/painting.dart" && created.Library != "painting" || created.ClassName != "TextPainter" {
		t.Errorf("created %#v", mine[0])
	}
	if _, ok := mine[1].(*foundation.ObjectDisposed); !ok {
		t.Errorf("disposed %#v", mine[1])
	}
}
