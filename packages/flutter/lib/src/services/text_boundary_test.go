package services

// Port of packages/flutter/test/services/text_boundary_test.dart (Flutter
// 3.50.1).
//
// Skipped: "wordBoundary.moveByWordBoundary" (needs painting's TextPainter
// and its WordBoundary, not ported yet).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// expectConsistent is _ConsistentTextRangeImplementationMatcher: for every
// position in [-1, length+1], getTextBoundaryAt agrees with the leading and
// trailing boundaries.
func expectConsistent(t *testing.T, b TextBoundary, length int) {
	t.Helper()
	for i := -1; i <= length+1; i++ {
		leading := orMinusOne(b.GetLeadingTextBoundaryAt(i))
		trailing := orMinusOne(b.GetTrailingTextBoundaryAt(i))
		if got := b.GetTextBoundaryAt(i); got.Start != leading || got.End != trailing {
			t.Errorf("at position %d, expected %v but got %v", i, ui.NewTextRange(leading, trailing), got)
		}
	}
}

func expectPos(t *testing.T, got *int, want any) {
	t.Helper()
	switch w := want.(type) {
	case nil:
		if got != nil {
			t.Errorf("got %d, want null", *got)
		}
	case int:
		if got == nil || *got != w {
			if got == nil {
				t.Errorf("got null, want %d", w)
			} else {
				t.Errorf("got %d, want %d", *got, w)
			}
		}
	}
}

func TestCharacterBoundaryWorks(t *testing.T) {
	b := NewCharacterBoundary("abc")
	expectConsistent(t, b, 3)
	expectPos(t, b.GetLeadingTextBoundaryAt(-1), nil)
	expectPos(t, b.GetTrailingTextBoundaryAt(-1), 0)
	for i := 0; i < 3; i++ {
		expectPos(t, b.GetLeadingTextBoundaryAt(i), i)
		expectPos(t, b.GetTrailingTextBoundaryAt(i), i+1)
	}
	expectPos(t, b.GetLeadingTextBoundaryAt(3), 3)
	expectPos(t, b.GetTrailingTextBoundaryAt(3), nil)
	expectPos(t, b.GetLeadingTextBoundaryAt(4), 3)
	expectPos(t, b.GetTrailingTextBoundaryAt(4), nil)
}

func TestCharacterBoundaryWorksWithGrapheme(t *testing.T) {
	text := "a❄︎c"
	b := NewCharacterBoundary(text)
	n := dart.StringLength(text)
	expectConsistent(t, b, n)
	expectPos(t, b.GetLeadingTextBoundaryAt(-1), nil)
	expectPos(t, b.GetTrailingTextBoundaryAt(-1), 0)
	expectPos(t, b.GetLeadingTextBoundaryAt(0), 0)
	expectPos(t, b.GetTrailingTextBoundaryAt(0), 1)
	// The `❄` takes two character length.
	expectPos(t, b.GetLeadingTextBoundaryAt(1), 1)
	expectPos(t, b.GetTrailingTextBoundaryAt(1), 3)
	expectPos(t, b.GetLeadingTextBoundaryAt(2), 1)
	expectPos(t, b.GetTrailingTextBoundaryAt(2), 3)
	expectPos(t, b.GetLeadingTextBoundaryAt(3), 3)
	expectPos(t, b.GetTrailingTextBoundaryAt(3), 4)
	expectPos(t, b.GetLeadingTextBoundaryAt(n), n)
	expectPos(t, b.GetTrailingTextBoundaryAt(n), nil)
}

// testTextLayoutMetrics is the test's TestTextLayoutMetrics.
type testTextLayoutMetrics struct{}

var lineAt3 = NewTextSelection(0, 10)

func (testTextLayoutMetrics) GetLineAtOffset(position ui.TextPosition) TextSelection {
	if position.Offset == 3 {
		return lineAt3
	}
	panic(dart.UnimplementedError{})
}

func (testTextLayoutMetrics) GetTextPositionAbove(ui.TextPosition) ui.TextPosition {
	panic(dart.UnimplementedError{})
}

func (testTextLayoutMetrics) GetTextPositionBelow(ui.TextPosition) ui.TextPosition {
	panic(dart.UnimplementedError{})
}

func (testTextLayoutMetrics) GetWordBoundary(position ui.TextPosition) ui.TextRange {
	if position.Offset == 3 {
		return ui.NewTextRange(4, 7)
	}
	panic(dart.UnimplementedError{})
}

func TestLineBoundaryWorks(t *testing.T) {
	b := NewLineBoundary(testTextLayoutMetrics{})
	expectPos(t, b.GetLeadingTextBoundaryAt(3), lineAt3.Start)
	expectPos(t, b.GetTrailingTextBoundaryAt(3), lineAt3.End)
	if got := b.GetTextBoundaryAt(3); got != lineAt3.TextRange {
		t.Errorf("got %v", got)
	}
}

func expectParagraph(t *testing.T, text string, position, leading, trailing int) {
	t.Helper()
	b := NewParagraphBoundary(text)
	expectPos(t, b.GetLeadingTextBoundaryAt(position), leading)
	expectPos(t, b.GetTrailingTextBoundaryAt(position), trailing)
}

func TestParagraphBoundary(t *testing.T) {
	// Simple cases.
	expectParagraph(t, "abcd efg hi\njklmno\npqrstuv", 10, 0, 12)
	expectParagraph(t, "abcd efg hi\r\njklmno\npqrstuv", 10, 0, 13)
	textF := "Now is the time for\nall good people\nto come to the aid\nof their country."
	expectParagraph(t, textF, 11, 0, 20)

	// Consecutive line terminators involving CRLF.
	textI := "Now is the time for\nall good people\n\r\nto come to the aid\nof their country."
	expectParagraph(t, textI, 56, 38, 57)
	expectParagraph(t, textI, 38, 38, 57)
	expectParagraph(t, textI, 37, 36, 38)

	// Consecutive line terminators.
	textJ := "Now is the time for\nall good people\n\nto come to the aid\nof their country."
	expectParagraph(t, textJ, 55, 37, 56)
	expectParagraph(t, textJ, 37, 37, 56)
	expectParagraph(t, textJ, 36, 36, 37)

	// Leading boundary for consecutive CRLF.
	expectParagraph(t, "abcd efg hi\r\n\r\n\r\n\r\n\r\n\r\n\r\n\n\n\n\n\njklmno\npqrstuv", 18, 17, 19)
	// Trailing boundary for consecutive CRLF.
	expectParagraph(t, "abcd efg hi\r\n\n\n\n\n\n\r\n\r\n\r\n\r\n\n\n\n\n\njklmno\npqrstuv", 18, 18, 20)
	// Position between two CRLF.
	expectParagraph(t, "abcd efg hi\r\nhello\r\n\n", 16, 13, 20)
	// Multiple consecutive line terminators.
	expectParagraph(t, "abcd efg hi\r\n\n\n\n\n\n\n\n\n\n\n\njklmno\npqrstuv", 18, 18, 19)
	expectParagraph(t, "abcd efg hi\r\n\n\n\n", 14, 14, 15)
}

func TestDocumentBoundaryWorks(t *testing.T) {
	text := "abcd efg hi\njklmno\npqrstuv"
	n := dart.StringLength(text)
	b := NewDocumentBoundary(text)
	expectConsistent(t, b, n)
	expectPos(t, b.GetLeadingTextBoundaryAt(-1), nil)
	expectPos(t, b.GetTrailingTextBoundaryAt(-1), n)
	expectPos(t, b.GetLeadingTextBoundaryAt(0), 0)
	expectPos(t, b.GetTrailingTextBoundaryAt(0), n)
	expectPos(t, b.GetLeadingTextBoundaryAt(10), 0)
	expectPos(t, b.GetTrailingTextBoundaryAt(10), n)
	expectPos(t, b.GetLeadingTextBoundaryAt(n), 0)
	expectPos(t, b.GetTrailingTextBoundaryAt(n), nil)
	expectPos(t, b.GetLeadingTextBoundaryAt(n+1), 0)
	expectPos(t, b.GetTrailingTextBoundaryAt(n+1), nil)
}

func TestTextLayoutMetricsStatics(t *testing.T) {
	for _, c := range []int{0x9, 0x20, 0xA0, 0x3000, 0x200A} {
		if !TextLayoutMetricsIsWhitespace(c) {
			t.Errorf("%#x is whitespace", c)
		}
	}
	for _, c := range []int{0x41, 0x200B, 0x85} {
		if TextLayoutMetricsIsWhitespace(c) {
			t.Errorf("%#x is not whitespace", c)
		}
	}
	for _, c := range []int{0x0A, 0x0B, 0x0C, 0x0D, 0x85, 0x2028, 0x2029} {
		if !TextLayoutMetricsIsLineTerminator(c) {
			t.Errorf("%#x is a line terminator", c)
		}
	}
	if TextLayoutMetricsIsLineTerminator(0x20) {
		t.Error("space")
	}
}
