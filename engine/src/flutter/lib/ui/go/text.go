package ui

// Partial port of engine/src/flutter/lib/ui/text.dart (Flutter 3.50.1):
// TextDirection, TextAffinity, TextPosition and TextRange (what semantics
// and services' TextSelection need). Text styles, paragraphs and the rest of
// text.dart come with the painting port.

import (
	"strconv"
	"unicode/utf16"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// TextDirection is a direction in which text flows.
type TextDirection int

const (
	// TextDirectionRtl: the text flows from right to left.
	TextDirectionRtl TextDirection = iota
	// TextDirectionLtr: the text flows from left to right.
	TextDirectionLtr
)

// TextDirectionValues is TextDirection.values.
var TextDirectionValues = []TextDirection{TextDirectionRtl, TextDirectionLtr}

// Name is the enum value's name.
func (d TextDirection) Name() string { return [...]string{"rtl", "ltr"}[d] }

func (d TextDirection) String() string { return "TextDirection." + d.Name() }

// TextAffinity is a way to disambiguate a TextPosition when its offset
// could match two different locations in the rendered string.
type TextAffinity int

const (
	// TextAffinityUpstream: the position has affinity for the upstream side
	// of the text position (the end of the line before a soft wrap).
	TextAffinityUpstream TextAffinity = iota
	// TextAffinityDownstream: the position has affinity for the downstream
	// side of the text position.
	TextAffinityDownstream
)

// TextAffinityValues is TextAffinity.values.
var TextAffinityValues = []TextAffinity{TextAffinityUpstream, TextAffinityDownstream}

// Name is the enum value's name.
func (a TextAffinity) Name() string { return [...]string{"upstream", "downstream"}[a] }

func (a TextAffinity) String() string { return "TextAffinity." + a.Name() }

// TextPosition is a position in a string of text (an offset in UTF-16 code
// units, and an affinity). Compare with ==.
type TextPosition struct {
	Offset   int
	Affinity TextAffinity
}

// NewTextPosition is TextPosition(offset:, affinity:) (affinity defaults
// to downstream; Dart's required named offset is positional).
func NewTextPosition(offset int, affinity ...TextAffinity) TextPosition {
	a := TextAffinityDownstream
	if len(affinity) > 0 {
		a = affinity[0]
	}
	return TextPosition{Offset: offset, Affinity: a}
}

func (p TextPosition) String() string {
	return "TextPosition(offset: " + strconv.Itoa(p.Offset) + ", affinity: " + p.Affinity.String() + ")"
}

// TextRange is a range of characters in a string of text (UTF-16 code
// unit offsets; -1 for an empty range). Compare with ==.
type TextRange struct {
	Start int
	End   int
}

// NewTextRange is TextRange(start:, end:) (both >= -1).
func NewTextRange(start, end int) TextRange {
	if start < -1 || end < -1 {
		panic(dart.NewAssertionError(nil))
	}
	return TextRange{Start: start, End: end}
}

// NewTextRangeCollapsed is TextRange.collapsed: an empty range at offset.
func NewTextRangeCollapsed(offset int) TextRange {
	if offset < -1 {
		panic(dart.NewAssertionError(nil))
	}
	return TextRange{Start: offset, End: offset}
}

// TextRangeEmpty is TextRange.empty: a range that contains nothing and is
// not in the text.
var TextRangeEmpty = TextRange{Start: -1, End: -1}

// IsValid reports whether this range represents a valid position in the
// text.
func (r TextRange) IsValid() bool { return r.Start >= 0 && r.End >= 0 }

// IsCollapsed reports whether this range is empty (but still potentially
// placed inside the text).
func (r TextRange) IsCollapsed() bool { return r.Start == r.End }

// IsNormalized reports whether the start of this range precedes the end.
func (r TextRange) IsNormalized() bool { return r.End >= r.Start }

// TextBefore is the text before this range.
func (r TextRange) TextBefore(text string) string {
	if !r.IsNormalized() {
		panic(dart.NewAssertionError(nil))
	}
	return utf16Substring(text, 0, r.Start)
}

// TextAfter is the text after this range.
func (r TextRange) TextAfter(text string) string {
	if !r.IsNormalized() {
		panic(dart.NewAssertionError(nil))
	}
	u := utf16.Encode([]rune(text))
	return utf16Substring(text, r.End, len(u))
}

// TextInside is the text inside this range.
func (r TextRange) TextInside(text string) string {
	if !r.IsNormalized() {
		panic(dart.NewAssertionError(nil))
	}
	return utf16Substring(text, r.Start, r.End)
}

func (r TextRange) String() string {
	return "TextRange(start: " + strconv.Itoa(r.Start) + ", end: " + strconv.Itoa(r.End) + ")"
}

// utf16Substring is Dart's String.substring(start, end) (UTF-16 offsets;
// it panics with a RangeError as Dart throws).
func utf16Substring(s string, start, end int) string {
	u := utf16.Encode([]rune(s))
	if start < 0 || start > len(u) {
		panic(dart.NewRangeErrorRange(start, 0, len(u), "start", ""))
	}
	if end < start || end > len(u) {
		panic(dart.NewRangeErrorRange(end, start, len(u), "end", ""))
	}
	return string(utf16.Decode(u[start:end]))
}
