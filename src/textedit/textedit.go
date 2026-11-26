// Package textedit is the text-editing model: values, selections, text
// boundaries, and the editing actions behind keyboard shortcuts. It is a
// port of Flutter's EditableText actions (_DeleteTextAction,
// _UpdateTextSelectionAction) and services/text_boundary.dart.
//
// Offsets are UTF-16 code units, exactly as in Flutter, so behavior can be
// compared offset for offset. Text itself is a normal Go (UTF-8) string.
package textedit

import (
	"unicode"
	"unicode/utf16"

	"github.com/rivo/uniseg"
)

// Selection is Flutter's TextSelection (UTF-16 offsets).
type Selection struct{ Base, Extent int }

func Collapsed(at int) Selection             { return Selection{at, at} }
func (s Selection) IsCollapsed() bool        { return s.Base == s.Extent }
func (s Selection) Start() int               { return min(s.Base, s.Extent) }
func (s Selection) End() int                 { return max(s.Base, s.Extent) }
func (s Selection) ExtendTo(e int) Selection { return Selection{s.Base, e} }

// Value is Flutter's TextEditingValue. Composing is the IME composing range
// (Start == End == -1 when not composing).
type Value struct {
	Text      string
	Selection Selection
	Composing Selection
}

// Len returns the text length in UTF-16 code units.
func (v Value) Len() int { return Len16(v.Text) }

// Len16 is the UTF-16 length of s.
func Len16(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

// byteAt converts a UTF-16 offset into a byte offset in s.
func byteAt(s string, off16 int) int {
	n := 0
	for i, r := range s {
		if n >= off16 {
			return i
		}
		n += utf16.RuneLen(r)
	}
	return len(s)
}

// Slice16 returns s[a:b] with UTF-16 offsets.
func Slice16(s string, a, b int) string { return s[byteAt(s, a):byteAt(s, b)] }

// runeAt16 returns the code point containing UTF-16 offset off, or -1.
func runeAt16(s string, off int) rune {
	if off < 0 {
		return -1
	}
	n := 0
	for _, r := range s {
		w := utf16.RuneLen(r)
		if off < n+w {
			return r
		}
		n += w
	}
	return -1
}

// codeUnitAt16 mirrors Dart's String.codeUnitAt (a lone surrogate counts).
func codeUnitAt16(s string, off int) (rune, bool) {
	if off < 0 {
		return 0, false
	}
	n := 0
	for _, r := range s {
		w := utf16.RuneLen(r)
		if off < n+w {
			if w == 2 {
				hi, lo := utf16.EncodeRune(r)
				if off == n {
					return hi, true
				}
				return lo, true
			}
			return r, true
		}
		n += w
	}
	return 0, false
}

// ---------------------------------------------------------------------------
// Text boundaries (Flutter's TextBoundary: leading/trailing, nil = none).

type Boundary interface {
	Leading(pos int) (int, bool)
	Trailing(pos int) (int, bool)
}

// GraphemeStops returns the UTF-16 offsets of grapheme cluster boundaries,
// including 0 and the text length.
func GraphemeStops(s string) []int {
	stops := []int{0}
	g := uniseg.NewGraphemes(s)
	n := 0
	for g.Next() {
		n += Len16(g.Str())
		stops = append(stops, n)
	}
	return stops
}

// CharacterBoundary is grapheme-cluster based, like Flutter's.
type CharacterBoundary struct {
	stops []int
	n     int
}

func NewCharacterBoundary(s string) CharacterBoundary {
	return CharacterBoundary{GraphemeStops(s), Len16(s)}
}

func (b CharacterBoundary) Leading(pos int) (int, bool) {
	if pos < 0 {
		return 0, false
	}
	pos = min(pos, b.n)
	start := 0
	for _, st := range b.stops {
		if st <= pos {
			start = st
		}
	}
	return start, true
}

func (b CharacterBoundary) Trailing(pos int) (int, bool) {
	if pos >= b.n {
		return 0, false
	}
	pos = max(0, pos)
	for _, st := range b.stops {
		if st > pos {
			return st, true
		}
	}
	return b.n, true
}

// WordBoundary is the paragraph's getWordBoundary (UAX #29 words; Flutter
// uses ICU, which agrees for ordinary text).
type WordBoundary struct {
	stops []int
	n     int
}

func NewWordBoundary(s string) WordBoundary {
	stops := []int{0}
	n := 0
	state := -1
	rest := s
	var word string
	for len(rest) > 0 {
		word, rest, state = uniseg.FirstWordInString(rest, state)
		n += Len16(word)
		stops = append(stops, n)
	}
	return WordBoundary{stops, n}
}

// rangeAt returns the word range containing pos ([n,n] at the end).
func (b WordBoundary) rangeAt(pos int) (int, int) {
	for i := 1; i < len(b.stops); i++ {
		if pos < b.stops[i] {
			return b.stops[i-1], b.stops[i]
		}
	}
	return b.n, b.n
}

func (b WordBoundary) Leading(pos int) (int, bool) {
	if pos < 0 {
		return 0, false
	}
	s, _ := b.rangeAt(pos)
	return s, true
}

func (b WordBoundary) Trailing(pos int) (int, bool) {
	_, e := b.rangeAt(max(0, pos))
	return e, true
}

// WordMoveBoundary is Flutter's WordBoundary.moveByWordBoundary: word
// boundaries, skipping those adjacent to spaces or punctuation.
type WordMoveBoundary struct {
	words WordBoundary
	text  string
}

func NewWordMoveBoundary(s string) WordMoveBoundary {
	return WordMoveBoundary{NewWordBoundary(s), s}
}

func isNewline(r rune) bool {
	switch r {
	case 0x0A, 0x85, 0x0B, 0x0C, 0x2028, 0x2029:
		return true
	}
	return false
}

// skipSpacesAndPunctuations is Flutter's predicate of the same name.
func (b WordMoveBoundary) stopsAt(offset int, forward bool) bool {
	innerAt, outerAt := offset, offset-1
	if forward {
		innerAt, outerAt = offset-1, offset
	}
	inner := runeAt16(b.text, innerAt)
	outer, ok := codeUnitAt16(b.text, outerAt)
	if inner < 0 || !ok || isNewline(inner) || isNewline(outer) {
		return true
	}
	return !(unicode.Is(unicode.Zs, inner) || unicode.IsPunct(inner))
}

func (b WordMoveBoundary) Leading(pos int) (int, bool) {
	for {
		if pos < 0 {
			return 0, false
		}
		off, ok := b.words.Leading(pos)
		if !ok || b.stopsAt(off, false) {
			return off, ok
		}
		pos = off - 1
	}
}

func (b WordMoveBoundary) Trailing(pos int) (int, bool) {
	for {
		off, ok := b.words.Trailing(max(pos, 0))
		if !ok || b.stopsAt(off, true) || off == pos {
			return off, ok
		}
		pos = off
	}
}

// LineBoundary for a single line: the whole text.
type LineBoundary struct{ n int }

func (b LineBoundary) Leading(pos int) (int, bool) {
	if pos < 0 {
		return 0, false
	}
	return 0, true
}
func (b LineBoundary) Trailing(int) (int, bool) { return b.n, true }

// ---------------------------------------------------------------------------
// Boundary application (Flutter's _moveBeyondTextBoundary/_moveToTextBoundary).

func moveBeyond(v Value, extent int, forward bool, b Boundary) int {
	if forward {
		if t, ok := b.Trailing(extent); ok {
			return t
		}
		return v.Len()
	}
	if l, ok := b.Leading(extent - 1); ok {
		return l
	}
	return 0
}

func moveTo(v Value, extent int, forward bool, b Boundary) int {
	if forward {
		if t, ok := b.Trailing(extent); ok {
			return t
		}
		return v.Len()
	}
	if l, ok := b.Leading(extent); ok {
		return l
	}
	return 0
}

// ---------------------------------------------------------------------------
// Actions

// Replace replaces [start,end) with s and puts the caret after it (Flutter's
// TextEditingValue.replaced for a keyboard edit).
func Replace(v Value, start, end int, s string) Value {
	text := Slice16(v.Text, 0, start) + s + Slice16(v.Text, end, v.Len())
	return Value{Text: text, Selection: Collapsed(start + Len16(s)), Composing: Selection{-1, -1}}
}

// InsertText replaces the selection with s (what a keyboard does).
func InsertText(v Value, s string) Value {
	return Replace(v, v.Selection.Start(), v.Selection.End(), s)
}

type BoundaryKind int

const (
	ByCharacter BoundaryKind = iota
	ByWord
	ByLine
)

func boundaryFor(v Value, k BoundaryKind) (Boundary, func(Value, int, bool, Boundary) int) {
	switch k {
	case ByWord:
		return NewWordMoveBoundary(v.Text), moveBeyond
	case ByLine:
		return LineBoundary{v.Len()}, moveTo
	}
	return NewCharacterBoundary(v.Text), moveBeyond
}

// Delete is Flutter's _DeleteTextAction: deletes the selection (widened to
// whole graphemes), or from the caret to the next boundary of kind k.
func Delete(v Value, forward bool, k BoundaryKind) Value {
	atomic := NewCharacterBoundary(v.Text)
	sel := v.Selection
	if !sel.IsCollapsed() {
		start, ok := atomic.Leading(sel.Start())
		if !ok {
			start = v.Len()
		}
		end, ok := atomic.Trailing(sel.End() - 1)
		if !ok {
			end = 0
		}
		return Replace(v, start, end, "")
	}
	b, apply := boundaryFor(v, k)
	target := apply(v, sel.Base, forward, b)
	var from int
	if forward {
		var ok bool
		if from, ok = atomic.Leading(sel.Base); !ok {
			from = v.Len()
		}
	} else {
		var ok bool
		if from, ok = atomic.Trailing(sel.Base - 1); !ok {
			from = 0
		}
	}
	return Replace(v, min(from, target), max(from, target), "")
}

// Move is Flutter's _UpdateTextSelectionAction. collapse moves the caret;
// otherwise the selection's extent moves and its base stays.
func Move(v Value, forward bool, k BoundaryKind, collapse bool) Value {
	sel := v.Selection
	// Character moves collapse a selection first; word and line moves don't
	// (ignoreNonCollapsedSelection in Flutter).
	if !sel.IsCollapsed() && collapse && k == ByCharacter {
		if forward {
			v.Selection = Collapsed(sel.End())
		} else {
			v.Selection = Collapsed(sel.Start())
		}
		return v
	}
	b, apply := boundaryFor(v, k)
	ext := apply(v, sel.Extent, forward, b)
	if collapse || ext == sel.Base {
		v.Selection = Collapsed(ext)
	} else {
		v.Selection = sel.ExtendTo(ext)
	}
	return v
}

// MoveVertically moves to the adjacent line; in a single line that is the
// start (up) or end (down) of the text.
func MoveVertically(v Value, down, collapse bool) Value {
	target := 0
	if down {
		target = v.Len()
	}
	if collapse || target == v.Selection.Base {
		v.Selection = Collapsed(target)
	} else {
		v.Selection = v.Selection.ExtendTo(target)
	}
	return v
}

// SelectAll selects the whole text.
func SelectAll(v Value) Value {
	v.Selection = Selection{0, v.Len()}
	return v
}

// SelectedText returns the selected substring.
func (v Value) SelectedText() string {
	return Slice16(v.Text, v.Selection.Start(), v.Selection.End())
}
