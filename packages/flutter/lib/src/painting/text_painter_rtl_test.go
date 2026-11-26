package painting_test

// Port of packages/flutter/test/painting/text_painter_rtl_test.dart
// (Flutter 3.50.1), under flutter_tester's test fonts.
//
// Skipped upstream (skip: skipTestsWithKnownBugs, flutter/flutter#87536),
// so not ported: "bidi overrides in LTR", "bidi overrides in RTL", "line
// wrap mid-word", "line wrap mid-word, bidi - LTR base", "line wrap
// mid-word, bidi - RTL base", "multiple levels", "getPositionForOffset -
// RTL in LTR", "getPositionForOffset - LTR in RTL", "Spaces".

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestTextPainterRtlBasicWords(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("ABC DEF\nGHI", NewTextStyle(TextStyleOptions{FontSize: fp(10.0)})))
	painter.Layout()
	expectEq(t, painter.GetWordBoundary(pos(1)), ui.NewTextRange(0, 3))
	expectEq(t, painter.GetWordBoundary(pos(5)), ui.NewTextRange(4, 7))
	expectEq(t, painter.GetWordBoundary(pos(9)), ui.NewTextRange(8, 11))
	painter.Dispose()
}

func TestTextPainterForcedLineWrappingWithBidi(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("Aא", NewTextStyle(TextStyleOptions{FontSize: fp(10.0)}))) // A, Alef
	expectEq(t, dart.StringLength(*painter.Text().(*TextSpan).Text), 2)
	painter.Layout(maxW(10.0))
	for index := 0; index <= 2; index++ {
		expectEq(t, painter.GetWordBoundary(pos(0)), ui.NewTextRange(0, 2))
	}
	expectEq(t, caret(painter, upstream(0)), ui.OffsetZero, "before the A")
	expectEq(t, caret(painter, pos(0)), ui.OffsetZero, "before the A")
	expectEq(t, caret(painter, upstream(1)), ui.NewOffset(10, 0), "after the A")
	expectEq(t, caret(painter, pos(1)), ui.NewOffset(10, 10), "before the Alef")
	expectEq(t, caret(painter, upstream(2)), ui.NewOffset(0, 10), "after the Alef")
	expectEq(t, caret(painter, pos(2)), ui.NewOffset(10, 10), "right of the Alef")

	expectBoxes(t, painter.GetBoxesForSelection(sel(0, 2)), []ui.TextBox{
		box(0, 0, 10, 10, ltr),  // A
		box(0, 10, 10, 20, rtl), // Alef
	})
	expectBoxes(t, painter.GetBoxesForSelection(sel(0, 1)), []ui.TextBox{box(0, 0, 10, 10, ltr)})
	expectBoxes(t, painter.GetBoxesForSelection(sel(1, 2)), []ui.TextBox{box(0, 10, 10, 20, rtl)})
	painter.Dispose()
}

func TestTextPainterEmptyTextBaseline(t *testing.T) {
	painter := newPainter(ltr)
	painter.SetText(span("", NewTextStyle(TextStyleOptions{FontFamily: sp("FlutterTest"), FontSize: fp(100.0),
		Height: fp(1.0)})))
	painter.Layout()
	expectEq(t, painter.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic), 75.0)
	painter.Dispose()
}
