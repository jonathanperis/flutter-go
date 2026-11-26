package services

// Port of packages/flutter/lib/src/services/text_layout_metrics.dart
// (Flutter 3.50.1).

import ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"

// TextLayoutMetrics is a read-only interface for accessing visual
// information about the implementing text.
type TextLayoutMetrics interface {
	// GetLineAtOffset returns a TextSelection containing the line of the
	// given TextPosition.
	GetLineAtOffset(position ui.TextPosition) TextSelection
	// GetWordBoundary returns the TextRange of the word at the given
	// TextPosition.
	GetWordBoundary(position ui.TextPosition) ui.TextRange
	// GetTextPositionAbove returns the TextPosition above the given offset
	// into the text.
	GetTextPositionAbove(position ui.TextPosition) ui.TextPosition
	// GetTextPositionBelow returns the TextPosition below the given offset
	// into the text.
	GetTextPositionBelow(position ui.TextPosition) ui.TextPosition
}

// TextLayoutMetricsIsWhitespace is TextLayoutMetrics.isWhitespace: whether
// the given UTF-16 code unit is a white space.
func TextLayoutMetricsIsWhitespace(codeUnit int) bool {
	switch codeUnit {
	case 0x9, // horizontal tab
		0xA,    // line feed
		0xB,    // vertical tab
		0xC,    // form feed
		0xD,    // carriage return
		0x1C,   // file separator
		0x1D,   // group separator
		0x1E,   // record separator
		0x1F,   // unit separator
		0x20,   // space
		0xA0,   // no-break space
		0x1680, // ogham space mark
		0x2000, // en quad
		0x2001, // em quad
		0x2002, // en space
		0x2003, // em space
		0x2004, // three-per-em space
		0x2005, // four-er-em space
		0x2006, // six-per-em space
		0x2007, // figure space
		0x2008, // punctuation space
		0x2009, // thin space
		0x200A, // hair space
		0x202F, // narrow no-break space
		0x205F, // medium mathematical space
		0x3000: // ideographic space
		return true
	}
	return false
}

// TextLayoutMetricsIsLineTerminator is TextLayoutMetrics.isLineTerminator:
// whether the given UTF-16 code unit is a line terminator.
func TextLayoutMetricsIsLineTerminator(codeUnit int) bool {
	switch codeUnit {
	case 0x0A, // line feed
		0x0B,   // vertical feed
		0x0C,   // form feed
		0x0D,   // carriage return
		0x85,   // new line
		0x2028, // line separator
		0x2029: // paragraph separator
		return true
	}
	return false
}
