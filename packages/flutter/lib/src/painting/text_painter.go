package painting

// PARTIAL port of text_painter.dart (Flutter 3.50.1): only the declarations
// that don't need dart:ui's Paragraph (kDefaultFontSize, TextOverflow,
// TextWidthBasis). TextPainter, PlaceholderDimensions and WordBoundary come
// with the text port.

// KDefaultFontSize is the default font size if none is specified (kept in
// sync with the engine's defaults).
const KDefaultFontSize = 14.0

// TextOverflow is how overflowing text should be handled.
type TextOverflow int

const (
	// TextOverflowClip clips the overflowing text to fix its container.
	TextOverflowClip TextOverflow = iota
	// TextOverflowFade fades the overflowing text to transparent.
	TextOverflowFade
	// TextOverflowEllipsis uses an ellipsis to indicate that the text has
	// overflowed.
	TextOverflowEllipsis
	// TextOverflowVisible renders overflowing text outside of its container.
	TextOverflowVisible
)

// TextOverflowValues is TextOverflow.values.
var TextOverflowValues = []TextOverflow{TextOverflowClip, TextOverflowFade, TextOverflowEllipsis, TextOverflowVisible}

// Name is the Dart enum value's name.
func (o TextOverflow) Name() string { return [...]string{"clip", "fade", "ellipsis", "visible"}[o] }

func (o TextOverflow) String() string { return "TextOverflow." + o.Name() }

// TextWidthBasis is the way of measuring the width of one or more lines of
// text.
type TextWidthBasis int

const (
	// TextWidthBasisParent: multiline text takes the full width given by
	// the parent; single-line text only the width it needs.
	TextWidthBasisParent TextWidthBasis = iota
	// TextWidthBasisLongestLine: the width is exactly enough to contain the
	// longest line.
	TextWidthBasisLongestLine
)

// TextWidthBasisValues is TextWidthBasis.values.
var TextWidthBasisValues = []TextWidthBasis{TextWidthBasisParent, TextWidthBasisLongestLine}

// Name is the Dart enum value's name.
func (b TextWidthBasis) Name() string { return [...]string{"parent", "longestLine"}[b] }

func (b TextWidthBasis) String() string { return "TextWidthBasis." + b.Name() }
