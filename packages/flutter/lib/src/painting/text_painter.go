package painting

// Port of text_painter.dart (Flutter 3.50.1): kDefaultFontSize,
// TextOverflow, PlaceholderDimensions, TextWidthBasis, WordBoundary (with
// the private _UntilTextBoundary) and TextPainter, on dart:ui's Paragraph.
// text_painter.dart's re-exports (dart:ui's Hyphens and LineMetrics,
// services' TextRange and TextSelection) are not repeated.

import (
	"math"
	"unicode"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

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

// PlaceholderDimensions holds the dimensions, alignment and baseline of a
// placeholder for TextPainter.SetPlaceholderDimensions. Compare with
// Equal.
type PlaceholderDimensions struct {
	// Size is the width and height of the placeholder.
	Size ui.Size
	// Alignment is how the placeholder aligns vertically with the text.
	Alignment ui.PlaceholderAlignment
	// BaselineOffset is the distance from the top edge of the rectangle to
	// the baseline position (nil: null).
	BaselineOffset *float64
	// Baseline is the TextBaseline to align to (nil: null).
	Baseline *ui.TextBaseline
}

// PlaceholderDimensionsEmpty is PlaceholderDimensions.empty: a zero size
// aligned to the bottom.
var PlaceholderDimensionsEmpty = PlaceholderDimensions{Size: ui.SizeZero, Alignment: ui.PlaceholderAlignmentBottom}

// Equal is PlaceholderDimensions' ==.
func (d PlaceholderDimensions) Equal(other PlaceholderDimensions) bool {
	return other.Size == d.Size && other.Alignment == d.Alignment && ptrEq(other.Baseline, d.Baseline) &&
		ptrEq(other.BaselineOffset, d.BaselineOffset)
}

func (d PlaceholderDimensions) String() string {
	if d.Alignment == ui.PlaceholderAlignmentBaseline {
		offset := "null"
		if d.BaselineOffset != nil {
			offset = dart.DoubleToString(*d.BaselineOffset)
		}
		return "PlaceholderDimensions(" + d.Size.String() + ", " + d.Alignment.String() + "(" + offset + " from top))"
	}
	return "PlaceholderDimensions(" + d.Size.String() + ", " + d.Alignment.String() + ")"
}

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

// WordBoundary is a TextBoundary that uses the word boundaries of a laid
// out paragraph (UAX #29), from TextPainter.WordBoundaries.
type WordBoundary struct {
	services.TextBoundaryBase
	text               InlineSpan
	paragraph          ui.Paragraph
	moveByWordBoundary services.TextBoundary
}

func newWordBoundary(text InlineSpan, paragraph ui.Paragraph) *WordBoundary {
	b := &WordBoundary{text: text, paragraph: paragraph}
	b.SetSelf(b)
	return b
}

// GetTextBoundaryAt is the paragraph's word boundary at position (negative
// positions are clamped to 0).
func (b *WordBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return b.paragraph.GetWordBoundary(ui.NewTextPosition(max(position, 0)))
}

func hexCodeUnit(v int) string {
	s := []byte{}
	for _, c := range []byte(dart.StringToUpperCase(formatHex(v))) {
		s = append(s, c)
	}
	for len(s) < 4 {
		s = append([]byte{'0'}, s...)
	}
	return string(s)
}

func formatHex(v int) string {
	const digits = "0123456789abcdef"
	if v == 0 {
		return "0"
	}
	var out []byte
	for v > 0 {
		out = append([]byte{digits[v%16]}, out...)
		v /= 16
	}
	return string(out)
}

// codePointFromSurrogates combines a high and a low surrogate into the
// supplementary character's code point.
func codePointFromSurrogates(highSurrogate, lowSurrogate int) int {
	if !TextPainterIsHighSurrogate(highSurrogate) {
		panic(dart.NewAssertionError("U+" + hexCodeUnit(highSurrogate) + ") is not a high surrogate."))
	}
	if !TextPainterIsLowSurrogate(lowSurrogate) {
		panic(dart.NewAssertionError("U+" + hexCodeUnit(lowSurrogate) + ") is not a low surrogate."))
	}
	const base = 0x010000 - (0xD800 << 10) - 0xDC00
	return (highSurrogate << 10) + lowSurrogate + base
}

// codePointAt is _codePointAt: the code point whose code unit is at index
// (nil past the text). Like Dart's `!`, it panics on a lone surrogate at
// the text's edge.
func (b *WordBoundary) codePointAt(index int) *int {
	codeUnitAtIndex := b.text.CodeUnitAt(index)
	if codeUnitAtIndex == nil {
		return nil
	}
	var v int
	switch *codeUnitAtIndex & 0xFC00 {
	case 0xD800:
		v = codePointFromSurrogates(*codeUnitAtIndex, *nonNull(b.text.CodeUnitAt(index+1)))
	case 0xDC00:
		v = codePointFromSurrogates(*nonNull(b.text.CodeUnitAt(index-1)), *codeUnitAtIndex)
	default:
		v = *codeUnitAtIndex
	}
	return &v
}

// nonNull is Dart's `!` (a null check that throws).
func nonNull[T any](v *T) *T {
	if v == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	return v
}

// isNewline is WordBoundary._isNewline (carriage return is not a hard
// line break).
func isNewline(codePoint int) bool {
	switch codePoint {
	case 0x000A, // Line Feed
		0x0085, // New Line
		0x000B, // Form Feed
		0x000C, // Vertical Feed
		0x2028, // Line Separator
		0x2029: // Paragraph Separator
		return true
	}
	return false
}

// isSpaceSeparatorOrPunctuation is RegExp(r'[\p{Space_Separator}\p{Punctuation}]',
// unicode: true).hasMatch(String.fromCharCode(c)).
func isSpaceSeparatorOrPunctuation(c int) bool {
	if c > unicode.MaxRune || c < 0 {
		return false
	}
	return unicode.Is(unicode.Zs, rune(c)) || unicode.Is(unicode.P, rune(c))
}

func (b *WordBoundary) skipSpacesAndPunctuations(offset int, forward bool) bool {
	// Use code point since some punctuations are supplementary characters.
	// "inner" here refers to the code unit that's before the break in the
	// search direction (`forward`).
	innerAt, outerAt := offset, offset-1
	if forward {
		innerAt, outerAt = offset-1, offset
	}
	innerCodePoint := b.codePointAt(innerAt)
	outerCodeUnit := b.text.CodeUnitAt(outerAt)

	// Make sure the hard break rules in UAX#29 take precedence over the ones
	// we add below. WB1 & WB2: always break at the start or the end of the
	// text. WB3a & WB3b: always break before and after newlines.
	hardBreakRulesApply := innerCodePoint == nil || outerCodeUnit == nil ||
		isNewline(*innerCodePoint) || isNewline(*outerCodeUnit)
	return hardBreakRulesApply || !isSpaceSeparatorOrPunctuation(*innerCodePoint)
}

// MoveByWordBoundary is a TextBoundary suited for moving the caret by
// words: it skips over word boundaries next to spaces and punctuation.
func (b *WordBoundary) MoveByWordBoundary() services.TextBoundary {
	if b.moveByWordBoundary == nil {
		b.moveByWordBoundary = newUntilTextBoundary(b, b.skipSpacesAndPunctuations)
	}
	return b.moveByWordBoundary
}

// untilTextBoundary is text_painter.dart's _UntilTextBoundary.
type untilTextBoundary struct {
	services.TextBoundaryBase
	textBoundary services.TextBoundary
	predicate    services.UntilPredicate
}

func newUntilTextBoundary(textBoundary services.TextBoundary, predicate services.UntilPredicate) *untilTextBoundary {
	b := &untilTextBoundary{textBoundary: textBoundary, predicate: predicate}
	b.SetSelf(b)
	return b
}

func (b *untilTextBoundary) GetLeadingTextBoundaryAt(position int) *int {
	if position < 0 {
		return nil
	}
	offset := b.textBoundary.GetLeadingTextBoundaryAt(position)
	if offset == nil || b.predicate(*offset, false) {
		return offset
	}
	return b.GetLeadingTextBoundaryAt(*offset - 1)
}

func (b *untilTextBoundary) GetTrailingTextBoundaryAt(position int) *int {
	offset := b.textBoundary.GetTrailingTextBoundaryAt(max(position, 0))
	if offset == nil || b.predicate(*offset, true) {
		return offset
	}
	return b.GetTrailingTextBoundaryAt(*offset)
}

// textLayout is _TextLayout: a laid out paragraph and its writing
// direction.
type textLayout struct {
	writingDirection ui.TextDirection
	painter          *TextPainter
	paragraph        ui.Paragraph

	endOfTextCaretMetrics *lineCaretMetrics
}

func (l *textLayout) width() float64                  { return l.paragraph.Width() }
func (l *textLayout) height() float64                 { return l.paragraph.Height() }
func (l *textLayout) minIntrinsicLineExtent() float64 { return l.paragraph.MinIntrinsicWidth() }
func (l *textLayout) maxIntrinsicLineExtent() float64 { return l.paragraph.MaxIntrinsicWidth() }
func (l *textLayout) longestLine() float64            { return l.paragraph.LongestLine() }

func (l *textLayout) getDistanceToBaseline(baseline ui.TextBaseline) float64 {
	if baseline == ui.TextBaselineIdeographic {
		return l.paragraph.IdeographicBaseline()
	}
	return l.paragraph.AlphabeticBaseline()
}

func (l *textLayout) getEndOfTextCaretMetrics() lineCaretMetrics {
	if l.endOfTextCaretMetrics == nil {
		m := l.computeEndOfTextCaretAnchorOffset()
		l.endOfTextCaretMetrics = &m
	}
	return *l.endOfTextCaretMetrics
}

func (l *textLayout) computeEndOfTextCaretAnchorOffset() lineCaretMetrics {
	rawString := l.painter.PlainText()
	length := dart.StringLength(rawString)
	lastLineIndex := l.paragraph.NumberOfLines() - 1
	if !(lastLineIndex >= 0) {
		panic(dart.NewAssertionError(nil))
	}
	lineMetrics := *nonNull(l.paragraph.GetLineMetricsAt(lastLineIndex))
	// Trailing white spaces don't contribute to the line width and thus
	// require special handling when they're present. They have the same
	// bidi embedding level as the paragraph (UAX #9 L1), so the caret is
	// anchored to the last logical trailing space. Whitespace character
	// definitions refer to Java/ICU, not Unicode-Zs.
	lastCodeUnit := dart.CodeUnitAt(rawString, length-1)
	var hasTrailingSpaces bool
	switch lastCodeUnit {
	case 0x0009: // horizontal tab
		hasTrailingSpaces = true
	case 0x00A0, // no-break space
		0x2007, // figure space
		0x202F: // narrow no-break space
		hasTrailingSpaces = false
	default:
		hasTrailingSpaces = (lastCodeUnit < 0xD800 || lastCodeUnit > 0xDFFF) && unicode.Is(unicode.Zs, rune(lastCodeUnit))
	}

	baseline := lineMetrics.Baseline
	var dx, height float64
	var lastGlyph *ui.GlyphInfo
	if hasTrailingSpaces {
		lastGlyph = l.paragraph.GetGlyphInfoAt(length - 1)
	}
	if hasTrailingSpaces && lastGlyph != nil {
		glyphBounds := lastGlyph.GraphemeClusterLayoutBounds
		if glyphBounds.IsEmpty() {
			panic(dart.NewAssertionError(nil))
		}
		if l.writingDirection == ui.TextDirectionLtr {
			dx = glyphBounds.Right
		} else {
			dx = glyphBounds.Left
		}
		height = glyphBounds.Height()
	} else {
		if l.writingDirection == ui.TextDirectionLtr {
			dx = lineMetrics.Left + lineMetrics.Width
		} else {
			dx = lineMetrics.Left
		}
		height = lineMetrics.Height
	}
	return lineCaretMetrics{offset: ui.NewOffset(dx, baseline), writingDirection: l.writingDirection, height: height}
}

func (l *textLayout) contentWidthFor(minWidth, maxWidth float64, widthBasis TextWidthBasis) float64 {
	if widthBasis == TextWidthBasisLongestLine {
		return ui.ClampDouble(l.longestLine(), minWidth, maxWidth)
	}
	return ui.ClampDouble(l.maxIntrinsicLineExtent(), minWidth, maxWidth)
}

// textPainterLayoutCacheWithOffset is _TextPainterLayoutCacheWithOffset:
// the current text layout and its paint offset/content width, plus cached
// metrics invalidated with the layout.
type textPainterLayoutCacheWithOffset struct {
	layout *textLayout
	// The input width used to lay out the paragraph.
	layoutMaxWidth float64
	// The content width the text painter should report in Width.
	contentWidth float64
	// The effective text alignment in [0, 1]: 0 for left aligned, 1 for
	// right aligned.
	textAlignment float64

	cachedInlinePlaceholderBoxes []ui.TextBox
	cachedRawLineMetrics         []ui.LineMetrics
	cachedLineMetrics            []ui.LineMetrics
	cachedLineMetricsPaintOffset *ui.Offset
	previousCaretPositionKey     *int
}

func newTextPainterLayoutCacheWithOffset(layout *textLayout, textAlignment, layoutMaxWidth, contentWidth float64) *textPainterLayoutCacheWithOffset {
	if !(textAlignment >= 0.0 && textAlignment <= 1.0) || math.IsNaN(layoutMaxWidth) || math.IsNaN(contentWidth) {
		panic(dart.NewAssertionError(nil))
	}
	return &textPainterLayoutCacheWithOffset{layout: layout, textAlignment: textAlignment,
		layoutMaxWidth: layoutMaxWidth, contentWidth: contentWidth}
}

func (c *textPainterLayoutCacheWithOffset) paragraph() ui.Paragraph { return c.layout.paragraph }

// paintOffset is the paragraph's paint offset in the painter's canvas
// (never NaN).
func (c *textPainterLayoutCacheWithOffset) paintOffset() ui.Offset {
	if c.textAlignment == 0 {
		return ui.OffsetZero
	}
	if w := c.paragraph().Width(); math.IsInf(w, 0) || math.IsNaN(w) {
		return ui.NewOffset(math.Inf(1), 0.0)
	}
	dx := float64(c.textAlignment * (c.contentWidth - c.paragraph().Width()))
	if math.IsNaN(dx) {
		panic(dart.NewAssertionError(nil))
	}
	return ui.NewOffset(dx, 0)
}

func isFinite(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }

// resizeToFit tries to fit the new constraints by only adjusting the
// content width (no line breaking); false when the text must be laid out
// again.
func (c *textPainterLayoutCacheWithOffset) resizeToFit(minWidth, maxWidth float64, widthBasis TextWidthBasis) bool {
	if !isFinite(c.layout.maxIntrinsicLineExtent()) || !(minWidth <= maxWidth) {
		panic(dart.NewAssertionError(nil))
	}
	// If a Paragraph's width is already >= its maxIntrinsicWidth, further
	// increasing the input width does not change its layout (but may change
	// the paint offset if it's not left-aligned), even for justify. The
	// exception is an infinite input width with non-left alignment.
	if maxWidth == c.contentWidth && minWidth == c.contentWidth {
		c.contentWidth = c.layout.contentWidthFor(minWidth, maxWidth, widthBasis)
		return true
	}

	// When the paint offset and the paragraph width are both +∞, the text
	// layout engine likely skipped layout because there was nothing to
	// paint. Always try to re-compute the text layout.
	if !isFinite(c.paintOffset().Dx) && !isFinite(c.paragraph().Width()) && isFinite(minWidth) {
		if c.paintOffset().Dx != math.Inf(1) || c.paragraph().Width() != math.Inf(1) {
			panic(dart.NewAssertionError(nil))
		}
		return false
	}

	maxIntrinsicWidth := c.paragraph().MaxIntrinsicWidth()
	// Skip line breaking if the input width remains the same, or there
	// will be no soft breaks.
	skipLineBreaking := maxWidth == c.layoutMaxWidth ||
		((c.paragraph().Width()-maxIntrinsicWidth) > -foundation.PrecisionErrorTolerance &&
			(maxWidth-maxIntrinsicWidth) > -foundation.PrecisionErrorTolerance)
	if skipLineBreaking {
		// Adjust the content width in case the TextWidthBasis changed.
		c.contentWidth = c.layout.contentWidthFor(minWidth, maxWidth, widthBasis)
		return true
	}
	return false
}

func (c *textPainterLayoutCacheWithOffset) inlinePlaceholderBoxes() []ui.TextBox {
	if c.cachedInlinePlaceholderBoxes == nil {
		c.cachedInlinePlaceholderBoxes = c.paragraph().GetBoxesForPlaceholders()
	}
	return c.cachedInlinePlaceholderBoxes
}

func (c *textPainterLayoutCacheWithOffset) rawLineMetrics() []ui.LineMetrics {
	if c.cachedRawLineMetrics == nil {
		c.cachedRawLineMetrics = c.paragraph().ComputeLineMetrics()
	}
	return c.cachedRawLineMetrics
}

func (c *textPainterLayoutCacheWithOffset) lineMetrics() []ui.LineMetrics {
	offset := c.paintOffset()
	if c.cachedLineMetrics != nil && c.cachedLineMetricsPaintOffset != nil && offset == *c.cachedLineMetricsPaintOffset {
		return c.cachedLineMetrics
	}
	var shiftedMetrics []ui.LineMetrics
	switch {
	case !isFinite(offset.Dx) || !isFinite(offset.Dy):
		shiftedMetrics = []ui.LineMetrics{}
	case offset == ui.OffsetZero:
		shiftedMetrics = c.rawLineMetrics()
	default:
		raw := c.rawLineMetrics()
		shiftedMetrics = make([]ui.LineMetrics, len(raw))
		for i, m := range raw {
			shiftedMetrics[i] = shiftLineMetrics(m, offset)
		}
	}
	c.cachedLineMetricsPaintOffset = &offset
	c.cachedLineMetrics = shiftedMetrics
	return shiftedMetrics
}

// lineCaretMetrics is _LineCaretMetrics: the caret's anchor point (top
// for a glyph, baseline for the end of text), writing direction and
// height.
type lineCaretMetrics struct {
	offset           ui.Offset
	writingDirection ui.TextDirection
	height           float64
}

func (m lineCaretMetrics) shift(offset ui.Offset) lineCaretMetrics {
	if offset == ui.OffsetZero {
		return m
	}
	return lineCaretMetrics{offset: offset.Add(m.offset), writingDirection: m.writingDirection, height: m.height}
}

// TextPainter paints an InlineSpan tree into a Canvas. Used by pointer and
// built with NewTextPainter; set Text and TextDirection, call Layout, then
// Paint. Call Dispose when it is no longer needed.
type TextPainter struct {
	text               InlineSpan
	textAlign          ui.TextAlign
	textDirection      *ui.TextDirection
	textScaler         TextScaler
	maxLines           *int
	ellipsis           *string
	locale             *ui.Locale
	strutStyle         *StrutStyle
	textWidthBasis     TextWidthBasis
	textHeightBehavior *ui.TextHeightBehavior
	hyphens            ui.Hyphens

	// Whether textWidthBasis has changed after the most recent Layout.
	debugNeedsRelayout bool
	// The result of the most recent Layout.
	layoutCache *textPainterLayoutCacheWithOffset
	// Whether layoutCache holds outdated paint information.
	rebuildParagraphForPaint bool

	debugMarkNeedsLayoutCallStack dart.StackTrace
	cachedPlainText               *string
	placeholderDimensions         []PlaceholderDimensions
	layoutTemplate                ui.Paragraph
	caretMetrics                  lineCaretMetrics
	disposed                      bool

	// DebugPaintTextLayoutBoxes is whether Paint draws the bounding box of
	// each glyph (debug).
	DebugPaintTextLayoutBoxes bool
}

// TextPainterOptions are TextPainter's named parameters (nil: Dart's
// default: TextAlign start, Hyphens manual (the zero value), TextScaler
// TextScaler.linear(TextScaleFactor)).
type TextPainterOptions struct {
	Text          InlineSpan
	TextAlign     *ui.TextAlign
	TextDirection *ui.TextDirection
	// TextScaleFactor is deprecated (nil: 1.0); give TextScaler instead.
	TextScaleFactor    *float64
	TextScaler         TextScaler
	MaxLines           *int
	Ellipsis           *string
	Locale             *ui.Locale
	StrutStyle         *StrutStyle
	TextWidthBasis     TextWidthBasis
	TextHeightBehavior *ui.TextHeightBehavior
	Hyphens            ui.Hyphens
}

// NewTextPainter is TextPainter({...}). It panics as Dart asserts: an
// invalid text, maxLines <= 0, or both a textScaleFactor and a
// textScaler.
func NewTextPainter(opts ...TextPainterOptions) *TextPainter {
	var o TextPainterOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Text != nil && !o.Text.DebugAssertIsValid() {
		panic(dart.NewAssertionError(nil))
	}
	if o.MaxLines != nil && !(*o.MaxLines > 0) {
		panic(dart.NewAssertionError(nil))
	}
	textScaleFactor := factorOr(o.TextScaleFactor)
	if !(textScaleFactor == 1.0 || o.TextScaler == nil) {
		panic(dart.NewAssertionError("Use textScaler instead."))
	}
	textScaler := o.TextScaler
	if textScaler == nil {
		textScaler = NewTextScalerLinear(textScaleFactor)
	}
	textAlign := ui.TextAlignStart
	if o.TextAlign != nil {
		textAlign = *o.TextAlign
	}
	p := &TextPainter{
		text:                     o.Text,
		textAlign:                textAlign,
		textDirection:            o.TextDirection,
		textScaler:               textScaler,
		maxLines:                 o.MaxLines,
		ellipsis:                 o.Ellipsis,
		locale:                   o.Locale,
		strutStyle:               o.StrutStyle,
		textWidthBasis:           o.TextWidthBasis,
		textHeightBehavior:       o.TextHeightBehavior,
		hyphens:                  o.Hyphens,
		debugNeedsRelayout:       true,
		rebuildParagraphForPaint: true,
	}
	foundation.DebugMaybeDispatchCreated("painting", "TextPainter", p)
	return p
}

// TextPainterComputeOptions are the named parameters of
// TextPainter.computeWidth and computeMaxIntrinsicWidth besides the
// required text and textDirection (MaxWidth nil: infinity; TextScaler
// nil: TextScaler.noScaling).
type TextPainterComputeOptions struct {
	TextAlign          *ui.TextAlign
	TextScaleFactor    *float64
	TextScaler         TextScaler
	MaxLines           *int
	Ellipsis           *string
	Locale             *ui.Locale
	StrutStyle         *StrutStyle
	TextWidthBasis     TextWidthBasis
	TextHeightBehavior *ui.TextHeightBehavior
	Hyphens            ui.Hyphens
	MinWidth           float64
	MaxWidth           *float64
}

func computePainter(text InlineSpan, textDirection ui.TextDirection, opts []TextPainterComputeOptions) *TextPainter {
	var o TextPainterComputeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	textScaler := o.TextScaler
	if textScaler == nil {
		textScaler = TextScalerNoScaling
	}
	textScaleFactor := factorOr(o.TextScaleFactor)
	if !(textScaleFactor == 1.0 || textScaler == TextScalerNoScaling) {
		panic(dart.NewAssertionError("Use textScaler instead."))
	}
	if textScaler == TextScalerNoScaling {
		textScaler = NewTextScalerLinear(textScaleFactor)
	}
	painter := NewTextPainter(TextPainterOptions{
		Text:               text,
		TextAlign:          o.TextAlign,
		TextDirection:      &textDirection,
		TextScaler:         textScaler,
		MaxLines:           o.MaxLines,
		Ellipsis:           o.Ellipsis,
		Locale:             o.Locale,
		StrutStyle:         o.StrutStyle,
		TextWidthBasis:     o.TextWidthBasis,
		TextHeightBehavior: o.TextHeightBehavior,
		Hyphens:            o.Hyphens,
	})
	painter.Layout(TextPainterLayoutOptions{MinWidth: o.MinWidth, MaxWidth: o.MaxWidth})
	return painter
}

// TextPainterComputeWidth is TextPainter.computeWidth: the width of the
// text laid out with the given parameters.
func TextPainterComputeWidth(text InlineSpan, textDirection ui.TextDirection, opts ...TextPainterComputeOptions) float64 {
	painter := computePainter(text, textDirection, opts)
	defer painter.Dispose()
	return painter.Width()
}

// TextPainterComputeMaxIntrinsicWidth is
// TextPainter.computeMaxIntrinsicWidth: the max intrinsic width of the text
// laid out with the given parameters.
func TextPainterComputeMaxIntrinsicWidth(text InlineSpan, textDirection ui.TextDirection, opts ...TextPainterComputeOptions) float64 {
	painter := computePainter(text, textDirection, opts)
	defer painter.Dispose()
	return painter.MaxIntrinsicWidth()
}

// debugAssertTextLayoutIsValid panics with Flutter's "Text layout not
// available" FlutterError when the painter has no layout.
func (p *TextPainter) debugAssertTextLayoutIsValid() {
	if p.DebugDisposed() {
		panic(dart.NewAssertionError(nil))
	}
	if p.layoutCache == nil {
		parts := []foundation.DiagnosticsNode{foundation.NewErrorSummary("Text layout not available")}
		if p.debugMarkNeedsLayoutCallStack != nil {
			parts = append(parts, foundation.NewDiagnosticsStackTrace(
				"The calls that first invalidated the text layout were", p.debugMarkNeedsLayoutCallStack))
		} else {
			parts = append(parts, foundation.NewErrorDescription("The TextPainter has never been laid out."))
		}
		panic(foundation.NewFlutterErrorFromParts(parts))
	}
}

func (p *TextPainter) debugAssertNoRelayout() {
	if p.debugNeedsRelayout {
		panic(dart.NewAssertionError(nil))
	}
}

// MarkNeedsLayout marks the painter's layout information as dirty and
// removes the cached layout.
func (p *TextPainter) MarkNeedsLayout() {
	if foundation.KDebugMode && p.layoutCache != nil && p.debugMarkNeedsLayoutCallStack == nil {
		p.debugMarkNeedsLayoutCallStack = dart.StackTraceCurrent()
	}
	if p.layoutCache != nil {
		p.layoutCache.paragraph().Dispose()
	}
	p.layoutCache = nil
}

// Text is the (potentially styled) text to paint (nil: none).
func (p *TextPainter) Text() InlineSpan { return p.text }

func inlineSpansEqual(a, b InlineSpan) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func spanStyle(s InlineSpan) *TextStyle {
	if s == nil {
		return nil
	}
	return s.Style()
}

// SetText sets the text, marking the layout dirty or the paragraph stale
// as the change requires.
func (p *TextPainter) SetText(value InlineSpan) {
	if value != nil && !value.DebugAssertIsValid() {
		panic(dart.NewAssertionError(nil))
	}
	if inlineSpansEqual(p.text, value) {
		return
	}
	if !spanStyle(p.text).Equal(spanStyle(value)) {
		if p.layoutTemplate != nil {
			p.layoutTemplate.Dispose()
		}
		p.layoutTemplate = nil
	}
	comparison := RenderComparisonLayout
	if value != nil && p.text != nil {
		comparison = p.text.CompareTo(value)
	}
	p.text = value
	p.cachedPlainText = nil
	if comparison >= RenderComparisonLayout {
		p.MarkNeedsLayout()
	} else if comparison >= RenderComparisonPaint {
		// Don't invalidate the layout cache just yet: it still contains valid
		// layout information.
		p.rebuildParagraphForPaint = true
	}
	// Neither relayout nor repaint is needed.
}

// PlainText is the text as a plain string, without semantics labels ("" for
// no text).
func (p *TextPainter) PlainText() string {
	if p.cachedPlainText == nil && p.text != nil {
		f := false
		s := p.text.ToPlainText(ToPlainTextOptions{IncludeSemanticsLabels: &f})
		p.cachedPlainText = &s
	}
	if p.cachedPlainText == nil {
		return ""
	}
	return *p.cachedPlainText
}

// TextAlign is how the text should be aligned horizontally.
func (p *TextPainter) TextAlign() ui.TextAlign { return p.textAlign }

// SetTextAlign sets the alignment (marking the layout dirty).
func (p *TextPainter) SetTextAlign(value ui.TextAlign) {
	if p.textAlign == value {
		return
	}
	p.textAlign = value
	p.MarkNeedsLayout()
}

// TextDirection is the default directionality of the text (nil: unset).
func (p *TextPainter) TextDirection() *ui.TextDirection { return p.textDirection }

func (p *TextPainter) disposeLayoutTemplate() {
	if p.layoutTemplate != nil {
		p.layoutTemplate.Dispose()
	}
	p.layoutTemplate = nil
}

// SetTextDirection sets the text direction (marking the layout dirty).
func (p *TextPainter) SetTextDirection(value *ui.TextDirection) {
	if ptrEq(p.textDirection, value) {
		return
	}
	p.textDirection = value
	p.MarkNeedsLayout()
	p.disposeLayoutTemplate() // Shouldn't really matter, but for strict correctness...
}

// TextScaleFactor is the deprecated number of font pixels per logical
// pixel (TextScaler().TextScaleFactor()).
func (p *TextPainter) TextScaleFactor() float64 { return p.textScaler.TextScaleFactor() }

// SetTextScaleFactor is the deprecated setter: SetTextScaler(linear(value)).
func (p *TextPainter) SetTextScaleFactor(value float64) { p.SetTextScaler(NewTextScalerLinear(value)) }

// TextScaler is the font scaling strategy to use when laying out and
// rendering the text.
func (p *TextPainter) TextScaler() TextScaler { return p.textScaler }

// SetTextScaler sets the text scaler (marking the layout dirty).
func (p *TextPainter) SetTextScaler(value TextScaler) {
	if value == p.textScaler {
		return
	}
	p.textScaler = value
	p.MarkNeedsLayout()
	p.disposeLayoutTemplate()
}

// Ellipsis is the string used to ellipsize overflowing text (nil: none).
func (p *TextPainter) Ellipsis() *string { return p.ellipsis }

// SetEllipsis sets the ellipsis (not empty; marking the layout dirty).
func (p *TextPainter) SetEllipsis(value *string) {
	if value != nil && *value == "" {
		panic(dart.NewAssertionError(nil))
	}
	if ptrEq(p.ellipsis, value) {
		return
	}
	p.ellipsis = value
	p.MarkNeedsLayout()
}

// Locale is the locale used to select region-specific glyphs (nil: none).
func (p *TextPainter) Locale() *ui.Locale { return p.locale }

// SetLocale sets the locale (marking the layout dirty).
func (p *TextPainter) SetLocale(value *ui.Locale) {
	if localeEq(p.locale, value) {
		return
	}
	p.locale = value
	p.MarkNeedsLayout()
}

// MaxLines is the maximum number of lines (nil: unlimited).
func (p *TextPainter) MaxLines() *int { return p.maxLines }

// SetMaxLines sets the maximum number of lines (> 0; marking the layout
// dirty).
func (p *TextPainter) SetMaxLines(value *int) {
	if value != nil && !(*value > 0) {
		panic(dart.NewAssertionError(nil))
	}
	if ptrEq(p.maxLines, value) {
		return
	}
	p.maxLines = value
	p.MarkNeedsLayout()
}

// StrutStyle is the strut style to use (nil: none).
func (p *TextPainter) StrutStyle() *StrutStyle { return p.strutStyle }

// SetStrutStyle sets the strut style (marking the layout dirty).
func (p *TextPainter) SetStrutStyle(value *StrutStyle) {
	if p.strutStyle.Equal(value) {
		return
	}
	p.strutStyle = value
	p.MarkNeedsLayout()
}

// TextWidthBasis defines how to measure the width of the rendered text.
func (p *TextPainter) TextWidthBasis() TextWidthBasis { return p.textWidthBasis }

// SetTextWidthBasis sets the width basis (Layout must be called again).
func (p *TextPainter) SetTextWidthBasis(value TextWidthBasis) {
	if p.textWidthBasis == value {
		return
	}
	if foundation.KDebugMode {
		p.debugNeedsRelayout = true
	}
	p.textWidthBasis = value
}

// TextHeightBehavior defines how the paragraph applies the text style's
// height to its first and last lines (nil: none).
func (p *TextPainter) TextHeightBehavior() *ui.TextHeightBehavior { return p.textHeightBehavior }

// SetTextHeightBehavior sets the height behavior (marking the layout
// dirty).
func (p *TextPainter) SetTextHeightBehavior(value *ui.TextHeightBehavior) {
	if ptrEq(p.textHeightBehavior, value) {
		return
	}
	p.textHeightBehavior = value
	p.MarkNeedsLayout()
}

// Hyphens is how words are hyphenated.
func (p *TextPainter) Hyphens() ui.Hyphens { return p.hyphens }

// SetHyphens sets the hyphenation (marking the layout dirty).
func (p *TextPainter) SetHyphens(value ui.Hyphens) {
	if p.hyphens == value {
		return
	}
	p.hyphens = value
	p.MarkNeedsLayout()
}

// InlinePlaceholderBoxes is the boxes of the placeholders, shifted by the
// paint offset (nil before Layout; empty when the paint offset is
// infinite).
func (p *TextPainter) InlinePlaceholderBoxes() []ui.TextBox {
	layout := p.layoutCache
	if layout == nil {
		return nil
	}
	offset := layout.paintOffset()
	if !isFinite(offset.Dx) || !isFinite(offset.Dy) {
		return []ui.TextBox{}
	}
	rawBoxes := layout.inlinePlaceholderBoxes()
	if offset == ui.OffsetZero {
		return rawBoxes
	}
	out := make([]ui.TextBox, len(rawBoxes))
	for i, b := range rawBoxes {
		out[i] = shiftTextBox(b, offset)
	}
	return out
}

func placeholderDimensionsEqual(a, b []PlaceholderDimensions) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// SetPlaceholderDimensions sets the dimensions of the text's placeholders,
// in order (nil or empty: no change). It panics (Dart's assert) when their
// number differs from the text's PlaceholderSpans.
func (p *TextPainter) SetPlaceholderDimensions(value []PlaceholderDimensions) {
	if len(value) == 0 || placeholderDimensionsEqual(value, p.placeholderDimensions) {
		return
	}
	placeholderCount := 0
	if p.text == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	p.text.VisitChildren(func(span InlineSpan) bool {
		if _, ok := span.(PlaceholderSpan); ok {
			placeholderCount++
		}
		return len(value) >= placeholderCount
	})
	if placeholderCount != len(value) {
		panic(dart.NewAssertionError(nil))
	}
	p.placeholderDimensions = value
	p.MarkNeedsLayout()
}

func (p *TextPainter) createParagraphStyle(textAlignOverride *ui.TextAlign) *ui.ParagraphStyle {
	if p.textDirection == nil {
		panic(dart.NewAssertionError("TextPainter.textDirection must be set to a non-null value before using the TextPainter."))
	}
	baseStyle := spanStyle(p.text)
	if baseStyle == nil {
		baseStyle = NewTextStyle()
	}
	textAlign := p.textAlign
	if textAlignOverride != nil {
		textAlign = *textAlignOverride
	}
	hyphens := p.hyphens
	return baseStyle.GetParagraphStyle(TextStyleGetParagraphStyleOptions{
		TextAlign:          &textAlign,
		TextDirection:      p.textDirection,
		TextScaler:         p.textScaler,
		MaxLines:           p.maxLines,
		TextHeightBehavior: p.textHeightBehavior,
		Hyphens:            &hyphens,
		Ellipsis:           p.ellipsis,
		Locale:             p.locale,
		StrutStyle:         p.strutStyle,
	})
}

func (p *TextPainter) createLayoutTemplate() ui.Paragraph {
	left := ui.TextAlignLeft
	// Direction doesn't matter, text is just a space.
	builder := ui.NewParagraphBuilder(p.createParagraphStyle(&left))
	if style := spanStyle(p.text); style != nil {
		builder.PushStyle(style.GetTextStyle(TextStyleGetTextStyleOptions{TextScaler: p.textScaler}))
	}
	builder.AddText(" ")
	paragraph := builder.Build()
	paragraph.Layout(ui.NewParagraphConstraints(math.Inf(1)))
	return paragraph
}

func (p *TextPainter) getOrCreateLayoutTemplate() ui.Paragraph {
	if p.layoutTemplate == nil {
		p.layoutTemplate = p.createLayoutTemplate()
	}
	return p.layoutTemplate
}

// PreferredLineHeight is the height of a space in the text's style: the
// height of an empty line.
func (p *TextPainter) PreferredLineHeight() float64 { return p.getOrCreateLayoutTemplate().Height() }

// MinIntrinsicWidth is the width at which no more horizontal space would
// keep the text from wrapping more (Layout first).
func (p *TextPainter) MinIntrinsicWidth() float64 {
	p.debugAssertTextLayoutIsValid()
	return p.layoutCache.layout.minIntrinsicLineExtent()
}

// MaxIntrinsicWidth is the width at which increasing the width no longer
// decreases the height (Layout first).
func (p *TextPainter) MaxIntrinsicWidth() float64 {
	p.debugAssertTextLayoutIsValid()
	return p.layoutCache.layout.maxIntrinsicLineExtent()
}

// Width is the horizontal space required to paint the text (Layout
// first).
func (p *TextPainter) Width() float64 {
	p.debugAssertTextLayoutIsValid()
	p.debugAssertNoRelayout()
	return p.layoutCache.contentWidth
}

// Height is the vertical space required to paint the text (Layout first).
func (p *TextPainter) Height() float64 {
	p.debugAssertTextLayoutIsValid()
	return p.layoutCache.layout.height()
}

// Size is the amount of space required to paint the text (Layout first).
func (p *TextPainter) Size() ui.Size {
	p.debugAssertTextLayoutIsValid()
	p.debugAssertNoRelayout()
	return ui.NewSize(p.Width(), p.Height())
}

// ComputeDistanceToActualBaseline is the distance from the top of the text
// to the first baseline of the given type (Layout first).
func (p *TextPainter) ComputeDistanceToActualBaseline(baseline ui.TextBaseline) float64 {
	p.debugAssertTextLayoutIsValid()
	return p.layoutCache.layout.getDistanceToBaseline(baseline)
}

// DidExceedMaxLines is whether the text was truncated or ellipsized by
// MaxLines (Layout first).
func (p *TextPainter) DidExceedMaxLines() bool {
	p.debugAssertTextLayoutIsValid()
	return p.layoutCache.paragraph().DidExceedMaxLines()
}

func (p *TextPainter) createParagraph(text InlineSpan) ui.Paragraph {
	builder := ui.NewParagraphBuilder(p.createParagraphStyle(nil))
	text.Build(builder, InlineSpanBuildOptions{TextScaler: p.textScaler, Dimensions: p.placeholderDimensions})
	if foundation.KDebugMode {
		p.debugMarkNeedsLayoutCallStack = nil
	}
	p.rebuildParagraphForPaint = false
	return builder.Build()
}

// TextPainterLayoutOptions are TextPainter.layout's named parameters
// (MinWidth zero is Dart's 0.0; MaxWidth nil is infinity).
type TextPainterLayoutOptions struct {
	MinWidth float64
	MaxWidth *float64
}

// Layout computes the visual position of the glyphs for painting the
// text, within [MinWidth, MaxWidth]. It panics with a StateError when the
// text or the text direction is not set.
func (p *TextPainter) Layout(opts ...TextPainterLayoutOptions) {
	var o TextPainterLayoutOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	minWidth := o.MinWidth
	maxWidth := math.Inf(1)
	if o.MaxWidth != nil {
		maxWidth = *o.MaxWidth
	}
	if math.IsNaN(maxWidth) || math.IsNaN(minWidth) {
		panic(dart.NewAssertionError(nil))
	}
	if foundation.KDebugMode {
		p.debugNeedsRelayout = false
	}

	cachedLayout := p.layoutCache
	if cachedLayout != nil && cachedLayout.resizeToFit(minWidth, maxWidth, p.textWidthBasis) {
		return
	}

	text := p.text
	if text == nil {
		panic(dart.StateError{Message: "TextPainter.text must be set to a non-null value before using the TextPainter."})
	}
	if p.textDirection == nil {
		panic(dart.StateError{Message: "TextPainter.textDirection must be set to a non-null value before using the TextPainter."})
	}
	textDirection := *p.textDirection

	paintOffsetAlignment := computePaintOffsetFraction(p.textAlign, textDirection)
	// Try to avoid laying out the paragraph with maxWidth=infinity when the
	// text is not left-aligned, so we don't have to deal with an infinite
	// paint offset.
	adjustMaxWidth := !isFinite(maxWidth) && paintOffsetAlignment != 0
	var adjustedMaxWidth *float64
	if !adjustMaxWidth {
		adjustedMaxWidth = &maxWidth
	} else if cachedLayout != nil {
		v := cachedLayout.layout.maxIntrinsicLineExtent()
		adjustedMaxWidth = &v
	}
	layoutMaxWidth := maxWidth
	if adjustedMaxWidth != nil {
		layoutMaxWidth = *adjustedMaxWidth
	}

	// Only rebuild the paragraph when there are layout changes, even when
	// rebuildParagraphForPaint is true.
	var paragraph ui.Paragraph
	if cachedLayout != nil {
		paragraph = cachedLayout.paragraph()
	} else {
		paragraph = p.createParagraph(text)
	}
	paragraph.Layout(ui.NewParagraphConstraints(layoutMaxWidth))
	layout := &textLayout{paragraph: paragraph, writingDirection: textDirection, painter: p}
	contentWidth := layout.contentWidthFor(minWidth, maxWidth, p.textWidthBasis)

	var newLayoutCache *textPainterLayoutCacheWithOffset
	// Call layout again if newLayoutCache had an infinite paint offset.
	// This is not as expensive as it seems, line breaking is relatively
	// cheap as compared to shaping.
	if adjustedMaxWidth == nil && isFinite(minWidth) {
		if !math.IsInf(maxWidth, 0) {
			panic(dart.NewAssertionError(nil))
		}
		newInputWidth := layout.maxIntrinsicLineExtent()
		paragraph.Layout(ui.NewParagraphConstraints(newInputWidth))
		newLayoutCache = newTextPainterLayoutCacheWithOffset(layout, paintOffsetAlignment, newInputWidth, contentWidth)
	} else {
		newLayoutCache = newTextPainterLayoutCacheWithOffset(layout, paintOffsetAlignment, layoutMaxWidth, contentWidth)
	}
	p.layoutCache = newLayoutCache
}

// Paint paints the text onto canvas at offset (Layout first; a StateError
// otherwise). Nothing is painted when the paint offset is infinite.
func (p *TextPainter) Paint(canvas ui.Canvas, offset ui.Offset) {
	layoutCache := p.layoutCache
	if layoutCache == nil {
		panic(dart.StateError{Message: "TextPainter.paint called when text geometry was not yet calculated.\n" +
			"Please call layout() before paint() to position the text before painting it."})
	}
	if po := layoutCache.paintOffset(); !isFinite(po.Dx) || !isFinite(po.Dy) {
		return
	}
	if p.rebuildParagraphForPaint {
		var debugSize ui.Size
		if foundation.KDebugMode {
			debugSize = p.Size()
		}
		paragraph := layoutCache.paragraph()
		// Even if there are only paint changes, there's no API to only make
		// those updates so the paragraph has to be recreated and re-laid out.
		if math.IsNaN(layoutCache.layoutMaxWidth) {
			panic(dart.NewAssertionError(nil))
		}
		newParagraph := p.createParagraph(p.text)
		newParagraph.Layout(ui.NewParagraphConstraints(layoutCache.layoutMaxWidth))
		layoutCache.layout.paragraph = newParagraph
		if paragraph.Width() != newParagraph.Width() {
			panic(dart.NewAssertionError(nil))
		}
		paragraph.Dispose()
		if foundation.KDebugMode && debugSize != p.Size() {
			panic(dart.NewAssertionError(nil))
		}
	}
	if p.rebuildParagraphForPaint {
		panic(dart.NewAssertionError(nil))
	}
	if p.DebugPaintTextLayoutBoxes {
		p.debugPaintCharacterLayoutBoxes(canvas, layoutCache, offset)
	}
	canvas.DrawParagraph(layoutCache.paragraph(), offset.Add(layoutCache.paintOffset()))
}

func (p *TextPainter) debugPaintCharacterLayoutBoxes(canvas ui.Canvas, layout *textPainterLayoutCacheWithOffset, offset ui.Offset) {
	paint := ui.NewPaint()
	paint.SetStyle(ui.PaintingStyleStroke)
	paint.SetStrokeWidth(1.0)
	paint.SetColor(ui.NewColor(0xFF00FFFF))
	textBoxes := p.GetBoxesForSelection(services.NewTextSelection(0, dart.StringLength(p.PlainText())))
	for _, textBox := range textBoxes {
		canvas.DrawRect(textBox.ToRect().Shift(offset), paint)
	}
}

// isUTF16 reports whether value falls in the valid range of the UTF-16
// encoding (Dart's _isUTF16, whose bound is 0xFFFFF).
func isUTF16(value int) bool { return value >= 0x0 && value <= 0xFFFFF }

// TextPainterIsHighSurrogate is TextPainter.isHighSurrogate: whether the
// UTF-16 code unit is a high (leading) surrogate.
func TextPainterIsHighSurrogate(value int) bool {
	if !isUTF16(value) {
		panic(dart.NewAssertionError(nil))
	}
	return value&0xFC00 == 0xD800
}

// TextPainterIsLowSurrogate is TextPainter.isLowSurrogate: whether the
// UTF-16 code unit is a low (trailing) surrogate.
func TextPainterIsLowSurrogate(value int) bool {
	if !isUTF16(value) {
		panic(dart.NewAssertionError(nil))
	}
	return value&0xFC00 == 0xDC00
}

func (p *TextPainter) textOrPanic() InlineSpan {
	if p.text == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	return p.text
}

// GetOffsetAfter is the offset after the code point at offset (nil past
// the text). It doesn't handle grapheme clusters of several code points.
func (p *TextPainter) GetOffsetAfter(offset int) *int {
	nextCodeUnit := p.textOrPanic().CodeUnitAt(offset)
	if nextCodeUnit == nil {
		return nil
	}
	v := offset + 1
	if TextPainterIsHighSurrogate(*nextCodeUnit) {
		v = offset + 2
	}
	return &v
}

// GetOffsetBefore is the offset before the code point that ends at offset
// (nil at the start).
func (p *TextPainter) GetOffsetBefore(offset int) *int {
	prevCodeUnit := p.textOrPanic().CodeUnitAt(offset - 1)
	if prevCodeUnit == nil {
		return nil
	}
	v := offset - 1
	if TextPainterIsLowSurrogate(*prevCodeUnit) {
		v = offset - 2
	}
	return &v
}

func computePaintOffsetFraction(textAlign ui.TextAlign, textDirection ui.TextDirection) float64 {
	switch textAlign {
	case ui.TextAlignLeft:
		return 0.0
	case ui.TextAlignRight:
		return 1.0
	case ui.TextAlignCenter:
		return 0.5
	case ui.TextAlignStart, ui.TextAlignJustify:
		if textDirection == ui.TextDirectionLtr {
			return 0.0
		}
		return 1.0
	default: // end
		if textDirection == ui.TextDirectionLtr {
			return 1.0
		}
		return 0.0
	}
}

// GetOffsetForCaret is the offset at which to paint the caret for
// position (caretPrototype's width is reserved on the trailing side of RTL
// text).
func (p *TextPainter) GetOffsetForCaret(position ui.TextPosition, caretPrototype ui.Rect) ui.Offset {
	layoutCache := p.layoutCache
	if layoutCache == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	caretMetrics := p.computeCaretMetrics(position)
	if caretMetrics == nil {
		paintOffsetAlignment := computePaintOffsetFraction(p.textAlign, *nonNull(p.textDirection))
		// The full width is not (width - caretPrototype.width), because
		// RenderEditable reserves cursor width on the right.
		dx := 0.0
		if paintOffsetAlignment != 0 {
			dx = float64(paintOffsetAlignment * layoutCache.contentWidth)
		}
		return ui.NewOffset(dx, 0.0)
	}
	rawOffset := caretMetrics.offset
	if caretMetrics.writingDirection == ui.TextDirectionRtl {
		rawOffset = ui.NewOffset(caretMetrics.offset.Dx-caretPrototype.Width(), caretMetrics.offset.Dy)
	}
	// If offset.dx is outside of the advertised content area, the
	// associated glyph belongs to a trailing whitespace character.
	paintOffset := layoutCache.paintOffset()
	adjustedDx := ui.ClampDouble(rawOffset.Dx+paintOffset.Dx, 0, layoutCache.contentWidth)
	return ui.NewOffset(adjustedDx, rawOffset.Dy+paintOffset.Dy)
}

// strutDisabled mirrors SkParagraph's condition for a disabled strut.
func (p *TextPainter) strutDisabled() bool {
	if p.strutStyle == nil || p.strutStyle.Equal(StrutStyleDisabled) {
		return true
	}
	return p.strutStyle.FontSize != nil && *p.strutStyle.FontSize == 0.0
}

// GetFullHeightForCaret is the strut-bounded height of the glyph at
// position (the template line's strut height when there is a strut).
func (p *TextPainter) GetFullHeightForCaret(position ui.TextPosition, caretPrototype ui.Rect) float64 {
	if p.strutDisabled() {
		if m := p.computeCaretMetrics(position); m != nil {
			return m.height
		}
	}
	boxes := p.getOrCreateLayoutTemplate().GetBoxesForRange(0, 1,
		ui.GetBoxesForRangeOptions{BoxHeightStyle: ui.BoxHeightStyleStrut})
	// The list may be empty under degenerate layout configurations (e.g.,
	// when the TextStyle.height is non-zero and TextStyle.fontSize or
	// textScaler is 0.0). In such cases, fall back to preferredLineHeight.
	if len(boxes) == 0 {
		return p.PreferredLineHeight()
	}
	if len(boxes) > 1 {
		panic(dart.StateError{Message: "Too many elements"})
	}
	return boxes[0].ToRect().Height()
}

func (p *TextPainter) isNewlineAtOffset(offset int) bool {
	plain := p.PlainText()
	return 0 <= offset && offset < dart.StringLength(plain) && isNewline(dart.CodeUnitAt(plain, offset))
}

// computeCaretMetrics is _computeCaretMetrics: the caret's offset and
// height for position, or nil when the paragraph is empty. See
// text_painter.dart for the edge cases (end of text, (0, upstream),
// upstream after a newline, multi-code-unit graphemes).
func (p *TextPainter) computeCaretMetrics(position ui.TextPosition) *lineCaretMetrics {
	p.debugAssertTextLayoutIsValid()
	p.debugAssertNoRelayout()

	cachedLayout := p.layoutCache
	// If nothing is laid out, top start is the only reasonable place to
	// place the cursor.
	if cachedLayout.paragraph().NumberOfLines() < 1 {
		return nil
	}

	var offset int
	var anchorToLeadingEdge bool
	switch {
	case position.Offset == 0:
		// As a special case, always anchor to the leading edge of the first
		// grapheme regardless of the affinity.
		offset, anchorToLeadingEdge = 0, true
	case position.Affinity == ui.TextAffinityDownstream:
		offset, anchorToLeadingEdge = position.Offset, true
	case p.isNewlineAtOffset(position.Offset - 1):
		offset, anchorToLeadingEdge = position.Offset, true
	default:
		offset, anchorToLeadingEdge = position.Offset-1, false
	}

	caretPositionCacheKey := -offset - 1
	if anchorToLeadingEdge {
		caretPositionCacheKey = offset
	}
	if cachedLayout.previousCaretPositionKey != nil && caretPositionCacheKey == *cachedLayout.previousCaretPositionKey {
		m := p.caretMetrics
		return &m
	}

	glyphInfo := cachedLayout.paragraph().GetGlyphInfoAt(offset)
	if glyphInfo == nil {
		// The position points to a character that is not laid out (the part
		// of text is invisible due to maxLines or infinite paragraph x
		// offset). Use the EOT caret.
		template := p.getOrCreateLayoutTemplate()
		if template.NumberOfLines() != 1 {
			panic(dart.NewAssertionError(nil))
		}
		baselineOffset := nonNull(template.GetLineMetricsAt(0)).Baseline
		m := cachedLayout.layout.getEndOfTextCaretMetrics().shift(ui.NewOffset(0.0, -baselineOffset))
		return &m
	}

	graphemeRange := glyphInfo.GraphemeClusterCodeUnitRange

	// Works around a SkParagraph bug: placeholders with a size of (0, 0)
	// always have a rect of Rect.zero and a range of (0, 0).
	if graphemeRange.IsCollapsed() {
		if graphemeRange.Start != 0 {
			panic(dart.NewAssertionError(nil))
		}
		return p.computeCaretMetrics(ui.NewTextPosition(offset + 1))
	}
	if anchorToLeadingEdge && graphemeRange.Start != offset {
		if !(graphemeRange.End > graphemeRange.Start+1) {
			panic(dart.NewAssertionError(nil))
		}
		// offset points to a multi-code-unit grapheme that doesn't start at
		// offset.
		return p.computeCaretMetrics(ui.NewTextPosition(graphemeRange.End))
	}

	boxes := cachedLayout.paragraph().GetBoxesForRange(graphemeRange.Start, graphemeRange.End,
		ui.GetBoxesForRangeOptions{BoxHeightStyle: ui.BoxHeightStyleStrut})

	anchorToLeft := anchorToLeadingEdge
	if glyphInfo.WritingDirection == ui.TextDirectionRtl {
		anchorToLeft = !anchorToLeadingEdge
	}
	if len(boxes) == 0 {
		panic(dart.StateError{Message: "No element"})
	}
	box := boxes[len(boxes)-1]
	if anchorToLeft {
		box = boxes[0]
	}
	x := box.Right
	if anchorToLeft {
		x = box.Left
	}
	metrics := lineCaretMetrics{offset: ui.NewOffset(x, box.Top), writingDirection: box.Direction,
		height: box.Bottom - box.Top}

	cachedLayout.previousCaretPositionKey = &caretPositionCacheKey
	p.caretMetrics = metrics
	return &metrics
}

// GetBoxesForSelectionOptions are getBoxesForSelection's named parameters
// (zero values: tight).
type GetBoxesForSelectionOptions struct {
	BoxHeightStyle ui.BoxHeightStyle
	BoxWidthStyle  ui.BoxWidthStyle
}

// GetBoxesForSelection is the boxes that bound the selection, shifted by
// the paint offset (empty when it is infinite).
func (p *TextPainter) GetBoxesForSelection(selection services.TextSelection, opts ...GetBoxesForSelectionOptions) []ui.TextBox {
	var o GetBoxesForSelectionOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p.debugAssertTextLayoutIsValid()
	if !selection.IsValid() {
		panic(dart.NewAssertionError(nil))
	}
	p.debugAssertNoRelayout()
	cachedLayout := p.layoutCache
	offset := cachedLayout.paintOffset()
	if !isFinite(offset.Dx) || !isFinite(offset.Dy) {
		return []ui.TextBox{}
	}
	boxes := cachedLayout.paragraph().GetBoxesForRange(selection.Start, selection.End,
		ui.GetBoxesForRangeOptions{BoxHeightStyle: o.BoxHeightStyle, BoxWidthStyle: o.BoxWidthStyle})
	if offset == ui.OffsetZero {
		return boxes
	}
	out := make([]ui.TextBox, len(boxes))
	for i, b := range boxes {
		out[i] = shiftTextBox(b, offset)
	}
	return out
}

// GetClosestGlyphForOffset is the glyph closest to offset (in the
// painter's coordinates; nil when the text is empty).
func (p *TextPainter) GetClosestGlyphForOffset(offset ui.Offset) *ui.GlyphInfo {
	p.debugAssertTextLayoutIsValid()
	p.debugAssertNoRelayout()
	cachedLayout := p.layoutCache
	paintOffset := cachedLayout.paintOffset()
	rawGlyphInfo := cachedLayout.paragraph().GetClosestGlyphInfoForOffset(offset.Sub(paintOffset))
	if rawGlyphInfo == nil || paintOffset == ui.OffsetZero {
		return rawGlyphInfo
	}
	return &ui.GlyphInfo{
		GraphemeClusterLayoutBounds:  rawGlyphInfo.GraphemeClusterLayoutBounds.Shift(paintOffset),
		GraphemeClusterCodeUnitRange: rawGlyphInfo.GraphemeClusterCodeUnitRange,
		WritingDirection:             rawGlyphInfo.WritingDirection,
	}
}

// GetPositionForOffset is the position within the text for offset (in the
// painter's coordinates).
func (p *TextPainter) GetPositionForOffset(offset ui.Offset) ui.TextPosition {
	p.debugAssertTextLayoutIsValid()
	p.debugAssertNoRelayout()
	cachedLayout := p.layoutCache
	return cachedLayout.paragraph().GetPositionForOffset(offset.Sub(cachedLayout.paintOffset()))
}

// GetWordBoundary is the word around position (UAX #29).
func (p *TextPainter) GetWordBoundary(position ui.TextPosition) ui.TextRange {
	p.debugAssertTextLayoutIsValid()
	return p.layoutCache.paragraph().GetWordBoundary(position)
}

// WordBoundaries is a TextBoundary over the laid out text's words.
func (p *TextPainter) WordBoundaries() *WordBoundary {
	if p.layoutCache == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	return newWordBoundary(p.textOrPanic(), p.layoutCache.paragraph())
}

// GetLineBoundary is the line around position.
func (p *TextPainter) GetLineBoundary(position ui.TextPosition) ui.TextRange {
	p.debugAssertTextLayoutIsValid()
	return p.layoutCache.paragraph().GetLineBoundary(position)
}

func shiftLineMetrics(metrics ui.LineMetrics, offset ui.Offset) ui.LineMetrics {
	if !isFinite(offset.Dx) || !isFinite(offset.Dy) {
		panic(dart.NewAssertionError(nil))
	}
	metrics.Left += offset.Dx
	metrics.Baseline += offset.Dy
	return metrics
}

func shiftTextBox(box ui.TextBox, offset ui.Offset) ui.TextBox {
	if !isFinite(offset.Dx) || !isFinite(offset.Dy) {
		panic(dart.NewAssertionError(nil))
	}
	return ui.NewTextBoxFromLTRBD(box.Left+offset.Dx, box.Top+offset.Dy, box.Right+offset.Dx,
		box.Bottom+offset.Dy, box.Direction)
}

// ComputeLineMetrics is the metrics of each line, shifted by the paint
// offset (empty when it is infinite).
func (p *TextPainter) ComputeLineMetrics() []ui.LineMetrics {
	p.debugAssertTextLayoutIsValid()
	p.debugAssertNoRelayout()
	return p.layoutCache.lineMetrics()
}

// DebugDisposed is whether Dispose was called (debug builds only; a
// StateError otherwise).
func (p *TextPainter) DebugDisposed() bool {
	if !foundation.KDebugMode {
		panic(dart.StateError{Message: "debugDisposed only available when asserts are on."})
	}
	return p.disposed
}

// Dispose releases the painter's paragraphs. The painter must not be used
// afterwards.
func (p *TextPainter) Dispose() {
	if p.DebugDisposed() {
		panic(dart.NewAssertionError(nil))
	}
	if foundation.KDebugMode {
		p.disposed = true
	}
	foundation.DebugMaybeDispatchDisposed(p)
	p.disposeLayoutTemplate()
	if p.layoutCache != nil {
		p.layoutCache.paragraph().Dispose()
	}
	p.layoutCache = nil
	p.text = nil
}
