package services_test

// Port of the "wordBoundary.moveByWordBoundary" test of
// packages/flutter/test/services/text_boundary_test.dart (Flutter 3.50.1).
// It lives in an external test package because it needs painting's
// TextPainter, and painting imports services.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
)

func TestWordBoundaryMoveByWordBoundary(t *testing.T) {
	ui.EngineUseTestFonts()
	text := "ABC   ABC\n" + // [0, 10)
		"AA\u0301    A\u0301\n" + // [10, 20)
		"         \n" + // [20, 30)
		"ABC!!!ABC\n" + // [30, 40)
		"  !ABC !!\n" + // [40, 50)
		"A  𑗋𑗋 A\n" // [50, 60)

	textPainter := painting.NewTextPainter()
	ltr := ui.TextDirectionLtr
	textPainter.SetTextDirection(&ltr)
	textPainter.SetText(painting.NewTextSpan(painting.TextSpanOptions{Text: &text}))
	textPainter.Layout()

	boundary := textPainter.WordBoundaries().MoveByWordBoundary()
	check := func(got *int, want int) {
		t.Helper()
		if got == nil || *got != want {
			if got == nil {
				t.Errorf("got null, want %d", want)
			} else {
				t.Errorf("got %d, want %d", *got, want)
			}
		}
	}

	// 4 points to the 2nd whitespace in the first line.
	// Don't break between horizontal spaces and letters/numbers.
	check(boundary.GetLeadingTextBoundaryAt(4), 0)
	check(boundary.GetTrailingTextBoundaryAt(4), 9)

	// Works when words are starting/ending with a combining diacritical mark.
	check(boundary.GetLeadingTextBoundaryAt(14), 10)
	check(boundary.GetTrailingTextBoundaryAt(14), 19)

	// Do break before and after newlines.
	check(boundary.GetLeadingTextBoundaryAt(24), 20)
	check(boundary.GetTrailingTextBoundaryAt(24), 29)

	// Do not break on punctuations.
	check(boundary.GetLeadingTextBoundaryAt(34), 30)
	check(boundary.GetTrailingTextBoundaryAt(34), 39)

	// Ok to break if next to punctuations or separating spaces.
	check(boundary.GetLeadingTextBoundaryAt(44), 43)
	check(boundary.GetTrailingTextBoundaryAt(44), 46)

	// 44 points to a low surrogate of a punctuation.
	check(boundary.GetLeadingTextBoundaryAt(54), 50)
	check(boundary.GetTrailingTextBoundaryAt(54), 59)
}
