package services

// Port of packages/flutter/lib/src/services/text_boundary.dart (Flutter
// 3.50.1). Positions are UTF-16 code unit offsets; a nil *int is Dart's
// null.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/characters"
)

// UntilPredicate is the signature of a predicate that, given a text offset
// and whether the search moves forward, decides whether to stop.
type UntilPredicate func(offset int, forward bool) bool

// TextBoundary is an interface for retrieving the logical text boundary
// (as opposed to the visual boundary) at a given code unit offset in a
// document. A subclass embeds TextBoundaryBase, calls SetSelf, and
// overrides either GetTextBoundaryAt or both of the other two methods.
type TextBoundary interface {
	// GetLeadingTextBoundaryAt returns the offset of the closest text
	// boundary before or at the given position, or nil if none.
	GetLeadingTextBoundaryAt(position int) *int
	// GetTrailingTextBoundaryAt returns the offset of the closest text
	// boundary after the given position, or nil if none.
	GetTrailingTextBoundaryAt(position int) *int
	// GetTextBoundaryAt returns the text boundary range that encloses the
	// input position (-1 for a missing end).
	GetTextBoundaryAt(position int) ui.TextRange
}

// TextBoundaryBase is the embeddable base of a TextBoundary: each default
// method is defined in terms of the others, called through self.
type TextBoundaryBase struct{ self TextBoundary }

// SetSelf points the base at the boundary that embeds it.
func (b *TextBoundaryBase) SetSelf(self TextBoundary) { b.self = self }

func (b *TextBoundaryBase) outer() TextBoundary {
	if b.self == nil {
		panic("services: TextBoundaryBase used before SetSelf (call SetSelf on the embedding boundary)")
	}
	return b.self
}

func intPtr(v int) *int { return &v }

func orMinusOne(v *int) int {
	if v == nil {
		return -1
	}
	return *v
}

// GetLeadingTextBoundaryAt is the default: the start of
// GetTextBoundaryAt(position).
func (b *TextBoundaryBase) GetLeadingTextBoundaryAt(position int) *int {
	if position < 0 {
		return nil
	}
	start := b.outer().GetTextBoundaryAt(position).Start
	if start >= 0 {
		return &start
	}
	return nil
}

// GetTrailingTextBoundaryAt is the default: the end of
// GetTextBoundaryAt(max(0, position)).
func (b *TextBoundaryBase) GetTrailingTextBoundaryAt(position int) *int {
	end := b.outer().GetTextBoundaryAt(max(0, position)).End
	if end >= 0 {
		return &end
	}
	return nil
}

// GetTextBoundaryAt is the default: the leading and trailing boundaries.
func (b *TextBoundaryBase) GetTextBoundaryAt(position int) ui.TextRange {
	start := orMinusOne(b.outer().GetLeadingTextBoundaryAt(position))
	end := orMinusOne(b.outer().GetTrailingTextBoundaryAt(position))
	return ui.NewTextRange(start, end)
}

// CharacterBoundary is a TextBoundary whose boundaries are the grapheme
// cluster boundaries of the text (user-perceived characters).
type CharacterBoundary struct {
	TextBoundaryBase
	text string
}

// NewCharacterBoundary is CharacterBoundary(text).
func NewCharacterBoundary(text string) *CharacterBoundary {
	b := &CharacterBoundary{text: text}
	b.SetSelf(b)
	return b
}

func (b *CharacterBoundary) GetLeadingTextBoundaryAt(position int) *int {
	if position < 0 {
		return nil
	}
	graphemeStart := characters.NewCharacterRangeAt(b.text, min(position, dart.StringLength(b.text))).StringBeforeLength()
	if !characters.NewCharacterRangeAt(b.text, graphemeStart).IsEmpty() {
		panic(dart.NewAssertionError(nil))
	}
	return &graphemeStart
}

func (b *CharacterBoundary) GetTrailingTextBoundaryAt(position int) *int {
	length := dart.StringLength(b.text)
	if position >= length {
		return nil
	}
	rangeAtPosition := characters.NewCharacterRangeAt(b.text, max(0, position+1))
	nextBoundary := rangeAtPosition.StringBeforeLength() + dart.StringLength(rangeAtPosition.Current())
	if !(nextBoundary == length || characters.NewCharacterRangeAt(b.text, nextBoundary).IsEmpty()) {
		panic(dart.NewAssertionError(nil))
	}
	return &nextBoundary
}

func (b *CharacterBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	if position < 0 {
		return ui.NewTextRange(-1, orMinusOne(b.GetTrailingTextBoundaryAt(position)))
	} else if position >= dart.StringLength(b.text) {
		return ui.NewTextRange(orMinusOne(b.GetLeadingTextBoundaryAt(position)), -1)
	}
	rangeAtPosition := characters.NewCharacterRangeAt(b.text, position)
	if rangeAtPosition.IsNotEmpty() {
		return ui.NewTextRange(rangeAtPosition.StringBeforeLength(),
			rangeAtPosition.StringBeforeLength()+dart.StringLength(rangeAtPosition.Current()))
	}
	// rangeAtPosition is empty means `position` is a grapheme boundary.
	return ui.NewTextRange(rangeAtPosition.StringBeforeLength(), orMinusOne(b.GetTrailingTextBoundaryAt(position)))
}

// LineBoundary is a TextBoundary whose boundaries are the line breaks of a
// text layout.
type LineBoundary struct {
	TextBoundaryBase
	textLayout TextLayoutMetrics
}

// NewLineBoundary is LineBoundary(textLayout).
func NewLineBoundary(textLayout TextLayoutMetrics) *LineBoundary {
	b := &LineBoundary{textLayout: textLayout}
	b.SetSelf(b)
	return b
}

func (b *LineBoundary) GetTextBoundaryAt(position int) ui.TextRange {
	return b.textLayout.GetLineAtOffset(ui.NewTextPosition(max(position, 0))).TextRange
}

// ParagraphBoundary is a TextBoundary whose boundaries are the paragraphs
// of the text: a paragraph ends with a line terminator (CR LF counts as
// one), which belongs to it.
type ParagraphBoundary struct {
	TextBoundaryBase
	text string
}

// NewParagraphBoundary is ParagraphBoundary(text).
func NewParagraphBoundary(text string) *ParagraphBoundary {
	b := &ParagraphBoundary{text: text}
	b.SetSelf(b)
	return b
}

func (b *ParagraphBoundary) GetLeadingTextBoundaryAt(position int) *int {
	units := dart.StringCodeUnits(b.text)
	if position < 0 || len(units) == 0 {
		return nil
	}
	if position >= len(units) {
		return intPtr(len(units))
	}
	if position == 0 {
		return intPtr(0)
	}
	codeUnitAt := func(i int) int { return int(units[i]) }
	index := position
	if index > 1 && codeUnitAt(index) == 0x0A && codeUnitAt(index-1) == 0x0D {
		index -= 2
	} else if TextLayoutMetricsIsLineTerminator(codeUnitAt(index)) {
		index--
	}
	for index > 0 {
		if TextLayoutMetricsIsLineTerminator(codeUnitAt(index)) {
			return intPtr(index + 1)
		}
		index--
	}
	return intPtr(max(index, 0))
}

func (b *ParagraphBoundary) GetTrailingTextBoundaryAt(position int) *int {
	units := dart.StringCodeUnits(b.text)
	if position >= len(units) || len(units) == 0 {
		return nil
	}
	if position < 0 {
		return intPtr(0)
	}
	index := position
	for !TextLayoutMetricsIsLineTerminator(int(units[index])) {
		index++
		if index == len(units) {
			return intPtr(index)
		}
	}
	if index < len(units)-1 && units[index] == 0x0D && units[index+1] == 0x0A {
		return intPtr(index + 2)
	}
	return intPtr(index + 1)
}

// DocumentBoundary is a text boundary that uses the entire document as
// the logical boundary.
type DocumentBoundary struct {
	TextBoundaryBase
	text string
}

// NewDocumentBoundary is DocumentBoundary(text).
func NewDocumentBoundary(text string) *DocumentBoundary {
	b := &DocumentBoundary{text: text}
	b.SetSelf(b)
	return b
}

func (b *DocumentBoundary) GetLeadingTextBoundaryAt(position int) *int {
	if position < 0 {
		return nil
	}
	return intPtr(0)
}

func (b *DocumentBoundary) GetTrailingTextBoundaryAt(position int) *int {
	length := dart.StringLength(b.text)
	if position >= length {
		return nil
	}
	return intPtr(length)
}
