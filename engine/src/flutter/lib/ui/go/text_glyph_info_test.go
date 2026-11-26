package ui

import (
	"math"
	"testing"
)

// A zero-size placeholder's glyph info is found but left unset by
// SkParagraph: the engine reports it as Rect.zero, (0, 0), rtl (its
// default-constructed GlyphInfo), which TextPainter's caret code relies on
// (the "placeholders with a size of (0, 0)" workaround). Flutter's output,
// probed with flutter_tester.
func TestGlyphInfoOfZeroSizePlaceholder(t *testing.T) {
	useTestFonts(t)
	size := 10.0
	b := NewParagraphBuilder(NewParagraphStyle())
	b.PushStyle(NewTextStyle(TextStyleOptions{FontSize: &size}))
	b.AddPlaceholder(0, 0, PlaceholderAlignmentBottom)
	b.AddText("ab")
	b.AddPlaceholder(0, 0, PlaceholderAlignmentMiddle)
	b.Pop()
	p := b.Build()
	p.Layout(NewParagraphConstraints(math.Inf(1)))
	want := []string{
		"Glyph(Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), textRange: TextRange(start: 0, end: 0), direction: TextDirection.rtl)",
		"Glyph(Rect.fromLTRB(0.0, 0.0, 10.0, 10.0), textRange: TextRange(start: 1, end: 2), direction: TextDirection.ltr)",
		"Glyph(Rect.fromLTRB(10.0, 0.0, 20.0, 10.0), textRange: TextRange(start: 2, end: 3), direction: TextDirection.ltr)",
		"Glyph(Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), textRange: TextRange(start: 0, end: 0), direction: TextDirection.rtl)",
		"<nil>",
	}
	for i, w := range want {
		got := "<nil>"
		if g := p.GetGlyphInfoAt(i); g != nil {
			got = g.String()
		}
		if got != w {
			t.Errorf("getGlyphInfoAt(%d) = %s, want %s", i, got, w)
		}
	}
}
