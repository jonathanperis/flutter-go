package ui

// Ports of engine/src/flutter/testing/dart/paragraph_test.dart and
// paragraph_builder_test.dart (Flutter 3.50.1). The engine runs them in
// flutter_tester with its test fonts (useTestFonts). No test is skipped.
//
// Divergence: "PushStyle should not segfault after build()" expects
// StateError; Go panics with dart.StateError (the same "native peer has
// been collected" text, checked by the text difftest).

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

const kAhemBaselineRatio = 1.25

func useTestFonts(t *testing.T) {
	t.Helper()
	EngineUseTestFonts()
}

func textCloseTo(t *testing.T, what string, got, want, eps float64) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Errorf("%s = %v, want %v ± %v", what, got, want, eps)
	}
}

func eq[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func ahemStyle(fontSize float64, dir ...TextDirection) *ParagraphStyle {
	o := ParagraphStyleOptions{FontFamily: ptrTo("Ahem"), FontStyle: ptrTo(FontStyleNormal),
		FontWeight: ptrTo(FontWeightNormal), FontSize: &fontSize}
	if len(dir) > 0 {
		o.TextDirection = &dir[0]
	}
	return NewParagraphStyle(o)
}

func ptrTo[T any](v T) *T { return &v }

func paragraphOf(style *ParagraphStyle, text string, width float64) Paragraph {
	b := NewParagraphBuilder(style)
	b.AddText(text)
	p := b.Build()
	p.Layout(NewParagraphConstraints(width))
	return p
}

// paragraph_test.dart

func TestParagraphPredictablySingleLineAhem(t *testing.T) {
	useTestFonts(t)
	for _, fontSize := range []float64{10.0, 20.0, 30.0, 40.0} {
		p := paragraphOf(ahemStyle(fontSize), "Test", 400.0)
		textCloseTo(t, "height", p.Height(), fontSize, 0.001)
		textCloseTo(t, "width", p.Width(), 400.0, 0.001)
		textCloseTo(t, "minIntrinsicWidth", p.MinIntrinsicWidth(), fontSize*4.0, 0.001)
		textCloseTo(t, "maxIntrinsicWidth", p.MaxIntrinsicWidth(), fontSize*4.0, 0.001)
		textCloseTo(t, "alphabeticBaseline", p.AlphabeticBaseline(), fontSize*.8, 0.001)
		textCloseTo(t, "ideographicBaseline", p.IdeographicBaseline(), p.AlphabeticBaseline()*kAhemBaselineRatio, 0.001)
	}
}

func TestParagraphPredictablySingleLineFlutterTest(t *testing.T) {
	useTestFonts(t)
	for _, fontSize := range []float64{10.0, 20.0, 30.0, 40.0} {
		p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontFamily: ptrTo("FlutterTest"),
			FontStyle: ptrTo(FontStyleNormal), FontWeight: ptrTo(FontWeightNormal), FontSize: &fontSize}), "Test", 400.0)
		eq(t, "height", p.Height(), fontSize)
		eq(t, "width", p.Width(), 400.0)
		eq(t, "minIntrinsicWidth", p.MinIntrinsicWidth(), fontSize*4.0)
		eq(t, "maxIntrinsicWidth", p.MaxIntrinsicWidth(), fontSize*4.0)
		eq(t, "alphabeticBaseline", p.AlphabeticBaseline(), fontSize*0.75)
		eq(t, "ideographicBaseline", p.IdeographicBaseline(), fontSize)
	}
}

func TestParagraphStyleHyphensIsPassedToTheTextLayout(t *testing.T) {
	useTestFonts(t)
	longestLine := func(hyphens *Hyphens) float64 {
		p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontFamily: ptrTo("FlutterTest"),
			FontSize: ptrTo(10.0), Hyphens: hyphens}), "abc­de", 45.0)
		return p.LongestLine()
	}
	if !(longestLine(ptrTo(HyphensManual)) > longestLine(ptrTo(HyphensHidden))) {
		t.Errorf("manual %v should be longer than hidden %v", longestLine(ptrTo(HyphensManual)), longestLine(ptrTo(HyphensHidden)))
	}
	eq(t, "null vs manual", longestLine(nil), longestLine(ptrTo(HyphensManual)))
}

func TestParagraphPredictablyMultiLine(t *testing.T) {
	useTestFonts(t)
	for _, fontSize := range []float64{10.0, 20.0, 30.0, 40.0} {
		p := paragraphOf(ahemStyle(fontSize), "Test Ahem", fontSize*5.0)
		textCloseTo(t, "height", p.Height(), fontSize*2.0, 0.001) // because it wraps
		textCloseTo(t, "width", p.Width(), fontSize*5.0, 0.001)
		textCloseTo(t, "minIntrinsicWidth", p.MinIntrinsicWidth(), fontSize*4.0, 0.001)
		textCloseTo(t, "maxIntrinsicWidth", p.MaxIntrinsicWidth(), fontSize*9.0, 0.001)
		textCloseTo(t, "alphabeticBaseline", p.AlphabeticBaseline(), fontSize*.8, 0.001)
		textCloseTo(t, "ideographicBaseline", p.IdeographicBaseline(), p.AlphabeticBaseline()*kAhemBaselineRatio, 0.001)
	}
}

func lineIs(t *testing.T, p Paragraph, pos TextPosition, start, end int) {
	t.Helper()
	line := p.GetLineBoundary(pos)
	if line.Start != start || line.End != end {
		t.Errorf("getLineBoundary(%v) = %v, want [%d, %d)", pos, line, start, end)
	}
}

func TestParagraphGetLineBoundary(t *testing.T) {
	useTestFonts(t)
	const fontSize = 10.0
	p := paragraphOf(ahemStyle(fontSize), "Test Ahem", fontSize*5.0)
	textCloseTo(t, "height", p.Height(), fontSize*2.0, 0.001)
	lineIs(t, p, NewTextPosition(5), 5, 9)
	lineIs(t, p, NewTextPosition(5, TextAffinityUpstream), 0, 5)
	lineIs(t, p, NewTextPosition(0), 0, 5)
	lineIs(t, p, NewTextPosition(9), 5, 9)
}

func TestParagraphGetLineBoundaryRTL(t *testing.T) {
	useTestFonts(t)
	const fontSize = 10.0
	p := paragraphOf(ahemStyle(fontSize, TextDirectionRtl), "القاهرةالقاهرة", fontSize*5.0)
	textCloseTo(t, "height", p.Height(), fontSize*3.0, 0.001)
	lineIs(t, p, NewTextPosition(5), 5, 10)
	lineIs(t, p, NewTextPosition(5, TextAffinityUpstream), 0, 5)
	lineIs(t, p, NewTextPosition(0), 0, 5)
	lineIs(t, p, NewTextPosition(9), 5, 10)
}

func TestParagraphGetLineBoundaryEmptyLine(t *testing.T) {
	useTestFonts(t)
	const fontSize = 10.0
	p := paragraphOf(ahemStyle(fontSize, TextDirectionRtl), "Test\n\nAhem", fontSize*5.0)
	textCloseTo(t, "height", p.Height(), fontSize*3.0, 0.001)
	lineIs(t, p, NewTextPosition(5), 5, 5)
	lineIs(t, p, NewTextPosition(5, TextAffinityUpstream), 5, 5)
	lineIs(t, p, NewTextPosition(4), 0, 4)
	lineIs(t, p, NewTextPosition(6), 6, 10)
}

func TestParagraphGetLineMetricsAt(t *testing.T) {
	useTestFonts(t)
	const fontSize = 10.0
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(fontSize),
		TextDirection: ptrTo(TextDirectionRtl), Height: ptrTo(2.0)}), "Test\npppp", 100.0)
	line := p.GetLineMetricsAt(1)
	if line == nil {
		t.Fatal("no line 1")
	}
	eq(t, "hardBreak", line.HardBreak, true)
	eq(t, "ascent", line.Ascent, 15.0)
	eq(t, "descent", line.Descent, 5.0)
	eq(t, "height", line.Height, 20.0)
	eq(t, "width", line.Width, 4*10.0)
	eq(t, "left", line.Left, 100.0-40.0)
	eq(t, "baseline", line.Baseline, 20.0+15.0)
	eq(t, "lineNumber", line.LineNumber, 1)
}

func lineNumberIs(t *testing.T, p Paragraph, offset int, want *int) {
	t.Helper()
	got := p.GetLineNumberAt(offset)
	if (got == nil) != (want == nil) || (got != nil && *got != *want) {
		t.Errorf("getLineNumberAt(%d) = %v, want %v", offset, got, want)
	}
}

func TestParagraphLineNumber(t *testing.T) {
	useTestFonts(t)
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(10.0)}), "Test\n\nTest", 100.0)
	eq(t, "numberOfLines", p.NumberOfLines(), 3)
	lineNumberIs(t, p, 4, ptrTo(0)) // first LF
	lineNumberIs(t, p, 5, ptrTo(1)) // second LF
	lineNumberIs(t, p, 6, ptrTo(2)) // "T" in the second "Test"
}

func TestParagraphEmpty(t *testing.T) {
	useTestFonts(t)
	p := NewParagraphBuilder(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(10.0)})).Build()
	p.Layout(NewParagraphConstraints(math.Inf(1)))
	if p.GetClosestGlyphInfoForOffset(OffsetZero) != nil || p.GetGlyphInfoAt(0) != nil {
		t.Error("glyph info of an empty paragraph")
	}
	if p.GetLineMetricsAt(0) != nil {
		t.Error("line metrics of an empty paragraph")
	}
	eq(t, "numberOfLines", p.NumberOfLines(), 0)
	lineNumberIs(t, p, 0, nil)
	if p.GetGlyphInfoAt(0) != nil || p.GetClosestGlyphInfoForOffset(OffsetZero) != nil {
		t.Error("glyph info of an empty paragraph")
	}
}

func TestParagraphOOBIndicesAsInput(t *testing.T) {
	useTestFonts(t)
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(10.0), MaxLines: ptrTo(1),
		Ellipsis: ptrTo("BBB")}), strings.Repeat("A", 100), 100)
	eq(t, "numberOfLines", p.NumberOfLines(), 1)
	if p.GetLineMetricsAt(-1) != nil {
		t.Error("line -1")
	}
	if m := p.GetLineMetricsAt(0); m == nil || m.LineNumber != 0 {
		t.Errorf("line 0: %v", m)
	}
	if p.GetLineMetricsAt(1) != nil {
		t.Error("line 1")
	}
	lineNumberIs(t, p, -1, nil)
	lineNumberIs(t, p, 0, ptrTo(0))
	lineNumberIs(t, p, 6, ptrTo(0))
	if p.GetLineMetricsAt(7) != nil {
		t.Error("line 7")
	}
	if p.GetGlyphInfoAt(-1) != nil {
		t.Error("glyph -1")
	}
	if g := p.GetGlyphInfoAt(0); g == nil || g.GraphemeClusterCodeUnitRange != NewTextRange(0, 1) {
		t.Errorf("glyph 0: %v", g)
	}
	if g := p.GetGlyphInfoAt(6); g == nil || g.GraphemeClusterCodeUnitRange != NewTextRange(6, 7) {
		t.Errorf("glyph 6: %v", g)
	}
	if p.GetGlyphInfoAt(7) != nil || p.GetGlyphInfoAt(200) != nil {
		t.Error("glyphs past the ellipsis")
	}
}

func TestParagraphQueryingGlyphInfo(t *testing.T) {
	useTestFonts(t)
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(10.0)}), "Test\nTest", math.Inf(1))
	bottomRight := p.GetClosestGlyphInfoForOffset(NewOffset(99.0, 99.0))
	last := p.GetGlyphInfoAt(8)
	if bottomRight == nil || last == nil || *bottomRight != *last {
		t.Fatalf("closest %v, last %v", bottomRight, last)
	}
	if first := p.GetGlyphInfoAt(0); *bottomRight == *first {
		t.Error("bottom right is the first glyph")
	}
	eq(t, "bounds", bottomRight.GraphemeClusterLayoutBounds, NewRectFromLTWH(30, 10, 10, 10))
	eq(t, "range", bottomRight.GraphemeClusterCodeUnitRange, NewTextRange(8, 9))
	eq(t, "direction", bottomRight.WritingDirection, TextDirectionLtr)
}

func TestParagraphPaintingADisposedParagraphDoesNotCrash(t *testing.T) {
	useTestFonts(t)
	p := NewParagraphBuilder(NewParagraphStyle()).Build()
	p.Dispose()
	recorder := NewPictureRecorder()
	canvas := NewCanvas(recorder)
	defer recorder.EndRecording()
	defer func() {
		if _, ok := recover().(dart.AssertionFailure); !ok {
			t.Error("drawParagraph of a disposed paragraph should throw an AssertionError")
		}
	}()
	canvas.DrawParagraph(p, OffsetZero)
}

func TestParagraphRoundingHackDisabled(t *testing.T) {
	useTestFonts(t)
	const fontSize = 1.25
	const text = "12345"
	if math.Trunc(fontSize*float64(len(text))) == fontSize*float64(len(text)) {
		t.Fatal("bad premise")
	}
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(fontSize)}), text, float64(len(text))*fontSize)
	eq(t, "maxIntrinsicWidth", p.MaxIntrinsicWidth(), float64(len(text))*fontSize)
	metrics := p.ComputeLineMetrics()
	if len(metrics) != 1 {
		t.Fatalf("%d lines", len(metrics))
	}
	eq(t, "width", metrics[0].Width, float64(len(text))*fontSize)
}

func TestParagraphKTextHeightNoneUnsetsTheHeightMultiplier(t *testing.T) {
	useTestFonts(t)
	const fontSize = 10.0
	b := NewParagraphBuilder(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(fontSize), Height: ptrTo(10.0)}))
	b.PushStyle(NewTextStyle(TextStyleOptions{Height: ptrTo(KTextHeightNone)}))
	b.AddText("A")
	p := b.Build()
	p.Layout(NewParagraphConstraints(1000))
	eq(t, "height", p.Height(), fontSize)
}

func TestParagraphKTextHeightNoneParagraphStyle(t *testing.T) {
	useTestFonts(t)
	const fontSize = 10.0
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(fontSize), Height: ptrTo(KTextHeightNone)}), "A", 1000)
	eq(t, "height", p.Height(), fontSize)
}

func TestParagraphKTextHeightNoneStrutStyle(t *testing.T) {
	useTestFonts(t)
	const fontSize = 10.0
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontSize: ptrTo(100.0),
		StrutStyle: NewStrutStyle(StrutStyleOptions{ForceStrutHeight: ptrTo(true), Height: ptrTo(KTextHeightNone),
			FontSize: ptrTo(fontSize)})}), "A", 1000)
	eq(t, "height", p.Height(), fontSize)
}

// paragraph_builder_test.dart

func TestParagraphBuilderBuildAndLayout(t *testing.T) {
	useTestFonts(t)
	p := paragraphOf(NewParagraphStyle(), "Hello", 800.0)
	if p.Width() == 0 || p.Height() == 0 {
		t.Errorf("width %v height %v", p.Width(), p.Height())
	}
}

func TestParagraphBuilderPushStyleShouldNotSegfaultAfterBuild(t *testing.T) {
	useTestFonts(t)
	b := NewParagraphBuilder(NewParagraphStyle())
	b.Build()
	defer func() {
		var se dart.StateError
		if e, ok := recover().(error); !ok || !errors.As(e, &se) {
			t.Error("pushStyle after build should throw a StateError")
		}
	}()
	b.PushStyle(NewTextStyle())
}

func TestParagraphBuilderGetRectsForRangeSmoke(t *testing.T) {
	useTestFonts(t)
	p := paragraphOf(NewParagraphStyle(), "Hello", 800.0)
	if p.Width() == 0 || p.Height() == 0 {
		t.Errorf("width %v height %v", p.Width(), p.Height())
	}
	boxes := p.GetBoxesForRange(0, 3)
	if len(boxes) != 1 {
		t.Fatalf("%d boxes", len(boxes))
	}
	eq(t, "box", boxes[0], NewTextBoxFromLTRBD(0.0, 0.0, 42.0, 14.0, TextDirectionLtr))
}

func TestParagraphBuilderLineMetricsSmoke(t *testing.T) {
	useTestFonts(t)
	p := paragraphOf(NewParagraphStyle(), "Hello", 800.0)
	metrics := p.ComputeLineMetrics()
	if len(metrics) != 1 {
		t.Fatalf("%d lines", len(metrics))
	}
	eq(t, "metrics", metrics[0], LineMetrics{HardBreak: true, Ascent: 10.5, Descent: 3.5, UnscaledAscent: 10.5,
		Height: 14.0, Width: 70.0, Left: 0.0, Baseline: 10.5, LineNumber: 0})
}
