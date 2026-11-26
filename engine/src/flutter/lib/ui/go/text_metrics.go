package ui

// Port of engine/src/flutter/lib/ui/text.dart (Flutter 3.50.1), part 3:
// GlyphInfo, TextBox, ParagraphConstraints, BoxHeightStyle, BoxWidthStyle,
// PlaceholderAlignment and LineMetrics.

import (
	"strconv"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// GlyphInfo is the layout information of a glyph (its grapheme cluster).
// Compare with ==.
type GlyphInfo struct {
	// GraphemeClusterLayoutBounds is the layout bounding rect of the
	// grapheme cluster, in the paragraph's coordinates.
	GraphemeClusterLayoutBounds Rect
	// GraphemeClusterCodeUnitRange is the UTF-16 range of the grapheme
	// cluster.
	GraphemeClusterCodeUnitRange TextRange
	// WritingDirection is the writing direction within the grapheme cluster.
	WritingDirection TextDirection
}

// NewGlyphInfo is GlyphInfo(graphemeClusterLayoutBounds,
// graphemeClusterCodeUnitRange, writingDirection).
func NewGlyphInfo(bounds Rect, codeUnitRange TextRange, writingDirection TextDirection) GlyphInfo {
	return GlyphInfo{bounds, codeUnitRange, writingDirection}
}

// newGlyphInfo is GlyphInfo._ (what the engine returns).
func newGlyphInfo(left, top, right, bottom float64, graphemeStart, graphemeEnd int, isLTR bool) GlyphInfo {
	d := TextDirectionRtl
	if isLTR {
		d = TextDirectionLtr
	}
	return GlyphInfo{NewRectFromLTRB(left, top, right, bottom), NewTextRange(graphemeStart, graphemeEnd), d}
}

func (g GlyphInfo) String() string {
	return "Glyph(" + g.GraphemeClusterLayoutBounds.String() + ", textRange: " +
		g.GraphemeClusterCodeUnitRange.String() + ", direction: " + g.WritingDirection.String() + ")"
}

// TextBox is a rectangle enclosing a run of text, with the run's direction.
// Compare with ==.
type TextBox struct {
	Left, Top, Right, Bottom float64
	Direction                TextDirection
}

// NewTextBoxFromLTRBD is TextBox.fromLTRBD.
func NewTextBoxFromLTRBD(left, top, right, bottom float64, direction TextDirection) TextBox {
	return TextBox{left, top, right, bottom, direction}
}

// ToRect is the box as a Rect (losing the direction).
func (b TextBox) ToRect() Rect { return NewRectFromLTRB(b.Left, b.Top, b.Right, b.Bottom) }

// Start is the side where the text starts (left for LTR, right for RTL).
func (b TextBox) Start() float64 {
	if b.Direction == TextDirectionLtr {
		return b.Left
	}
	return b.Right
}

// End is the side where the text ends.
func (b TextBox) End() float64 {
	if b.Direction == TextDirectionLtr {
		return b.Right
	}
	return b.Left
}

func (b TextBox) String() string {
	return "TextBox.fromLTRBD(" + dart.ToStringAsFixed(b.Left, 1) + ", " + dart.ToStringAsFixed(b.Top, 1) + ", " +
		dart.ToStringAsFixed(b.Right, 1) + ", " + dart.ToStringAsFixed(b.Bottom, 1) + ", " + b.Direction.String() + ")"
}

// ParagraphConstraints is the layout constraints for a paragraph: its
// width. Compare with ==.
type ParagraphConstraints struct {
	Width float64
}

// NewParagraphConstraints is ParagraphConstraints(width:).
func NewParagraphConstraints(width float64) ParagraphConstraints { return ParagraphConstraints{width} }

func (c ParagraphConstraints) String() string {
	return "ParagraphConstraints(width: " + dart.DoubleToString(c.Width) + ")"
}

// BoxHeightStyle defines the heights of the boxes getBoxesForRange returns.
type BoxHeightStyle int

const (
	// BoxHeightStyleTight: boxes as tall as the glyphs' runs.
	BoxHeightStyleTight BoxHeightStyle = iota
	// BoxHeightStyleMax: every box of a line as tall as the line's tallest
	// run.
	BoxHeightStyleMax
	// BoxHeightStyleIncludeLineSpacingMiddle: line spacing split between
	// the top and bottom of the boxes.
	BoxHeightStyleIncludeLineSpacingMiddle
	// BoxHeightStyleIncludeLineSpacingTop: line spacing added to the top.
	BoxHeightStyleIncludeLineSpacingTop
	// BoxHeightStyleIncludeLineSpacingBottom: line spacing added to the
	// bottom.
	BoxHeightStyleIncludeLineSpacingBottom
	// BoxHeightStyleStrut: boxes as tall as the strut.
	BoxHeightStyleStrut
)

// BoxHeightStyleValues is BoxHeightStyle.values.
var BoxHeightStyleValues = []BoxHeightStyle{BoxHeightStyleTight, BoxHeightStyleMax,
	BoxHeightStyleIncludeLineSpacingMiddle, BoxHeightStyleIncludeLineSpacingTop,
	BoxHeightStyleIncludeLineSpacingBottom, BoxHeightStyleStrut}

// Name is the enum value's name.
func (s BoxHeightStyle) Name() string {
	return [...]string{"tight", "max", "includeLineSpacingMiddle", "includeLineSpacingTop",
		"includeLineSpacingBottom", "strut"}[s]
}

func (s BoxHeightStyle) String() string { return "BoxHeightStyle." + s.Name() }

// BoxWidthStyle defines the widths of the boxes getBoxesForRange returns.
type BoxWidthStyle int

const (
	// BoxWidthStyleTight: boxes as wide as the glyphs.
	BoxWidthStyleTight BoxWidthStyle = iota
	// BoxWidthStyleMax: the last box of each line extended to the widest
	// line's width.
	BoxWidthStyleMax
)

// BoxWidthStyleValues is BoxWidthStyle.values.
var BoxWidthStyleValues = []BoxWidthStyle{BoxWidthStyleTight, BoxWidthStyleMax}

// Name is the enum value's name.
func (s BoxWidthStyle) Name() string { return [...]string{"tight", "max"}[s] }

func (s BoxWidthStyle) String() string { return "BoxWidthStyle." + s.Name() }

// PlaceholderAlignment is where to vertically align a placeholder relative
// to the surrounding text.
type PlaceholderAlignment int

const (
	// PlaceholderAlignmentBaseline matches the placeholder's baseline
	// offset with the text's baseline.
	PlaceholderAlignmentBaseline PlaceholderAlignment = iota
	// PlaceholderAlignmentAboveBaseline: the placeholder's bottom edge on
	// the baseline.
	PlaceholderAlignmentAboveBaseline
	// PlaceholderAlignmentBelowBaseline: the placeholder's top edge on the
	// baseline.
	PlaceholderAlignmentBelowBaseline
	// PlaceholderAlignmentTop: the placeholder's top edge with the font's
	// top.
	PlaceholderAlignmentTop
	// PlaceholderAlignmentBottom: the placeholder's bottom edge with the
	// font's bottom.
	PlaceholderAlignmentBottom
	// PlaceholderAlignmentMiddle: the placeholder's middle with the text's
	// middle.
	PlaceholderAlignmentMiddle
)

// PlaceholderAlignmentValues is PlaceholderAlignment.values.
var PlaceholderAlignmentValues = []PlaceholderAlignment{PlaceholderAlignmentBaseline,
	PlaceholderAlignmentAboveBaseline, PlaceholderAlignmentBelowBaseline, PlaceholderAlignmentTop,
	PlaceholderAlignmentBottom, PlaceholderAlignmentMiddle}

// Name is the enum value's name.
func (a PlaceholderAlignment) Name() string {
	return [...]string{"baseline", "aboveBaseline", "belowBaseline", "top", "bottom", "middle"}[a]
}

func (a PlaceholderAlignment) String() string { return "PlaceholderAlignment." + a.Name() }

// LineMetrics are the metrics of a line of a laid-out paragraph (every
// field is required: build it with a struct literal). Compare with ==.
type LineMetrics struct {
	// HardBreak: whether the line ends with an explicit line break or the
	// end of the paragraph.
	HardBreak bool
	// Ascent is the rise from the baseline (positive: up).
	Ascent float64
	// Descent is the drop from the baseline (positive: down).
	Descent float64
	// UnscaledAscent is the ascent ignoring the style's height.
	UnscaledAscent float64
	// Height is the line's height: round(ascent + descent).
	Height float64
	// Width is the line's width from the left of its leftmost glyph to the
	// right of its rightmost.
	Width float64
	// Left is the x of the line's left edge.
	Left float64
	// Baseline is the y of the line's baseline from the paragraph's top.
	Baseline float64
	// LineNumber is the line's index (from 0).
	LineNumber int
}

func (m LineMetrics) String() string {
	return "LineMetrics(hardBreak: " + strconv.FormatBool(m.HardBreak) + ", " +
		"ascent: " + dart.DoubleToString(m.Ascent) + ", " +
		"descent: " + dart.DoubleToString(m.Descent) + ", " +
		"unscaledAscent: " + dart.DoubleToString(m.UnscaledAscent) + ", " +
		"height: " + dart.DoubleToString(m.Height) + ", " +
		"width: " + dart.DoubleToString(m.Width) + ", " +
		"left: " + dart.DoubleToString(m.Left) + ", " +
		"baseline: " + dart.DoubleToString(m.Baseline) + ", " +
		"lineNumber: " + strconv.Itoa(m.LineNumber) + ")"
}
