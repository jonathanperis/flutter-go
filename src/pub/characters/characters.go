// Package characters is package:characters 1.4.1 (lib/src/characters.dart,
// characters_impl.dart, extensions.dart and grapheme_clusters/), the pure
// Dart package Flutter 3.50.1 uses for grapheme clusters ("characters"),
// and which its widgets library re-exports. Grapheme breaking runs the
// package's own generated tables and automatons (table.go is generated
// from table.dart by gen_table.go), so boundaries are Dart's bit for bit.
//
// Strings are Go strings read as WTF-8 (see src/dart's string.go); every
// index is a UTF-16 code unit index, as in Dart. Strings returned are
// canonical WTF-8 (a surrogate pair is the UTF-8 of its code point; an
// unpaired surrogate is a lone surrogate).
//
// Dart's Iterable<String> is All() (an iter.Seq) plus the members the
// package overrides (First, Last, Single, Length, ElementAt, Contains,
// Join, LastWhere, Where, Skip, Take, SkipWhile, TakeWhile...). The
// extension `String.characters` (extension StringCharacters) is the
// function StringCharacters(s).
package characters

import "iter"

// Characters is the characters (Unicode grapheme clusters) of a string.
// Compare with Equal (Dart's ==: same string).
type Characters interface {
	// String is the string to iterate (Dart's `string` getter and
	// toString).
	String() string

	// All iterates the characters (Dart's Iterable<String>).
	All() iter.Seq[string]
	// Iterator is an iterator over the characters of this sequence; it
	// starts at the beginning, with an empty current range.
	Iterator() CharacterRange
	// IteratorAtEnd is an iterator starting at the end.
	IteratorAtEnd() CharacterRange

	First() string
	Last() string
	Single() string
	IsEmpty() bool
	IsNotEmpty() bool
	Length() int
	Join(separator ...string) string
	LastWhere(test func(string) bool, orElse ...func() string) string
	ElementAt(index int) string

	// Contains is whether singleCharacterString occurs as one character.
	Contains(singleCharacterString string) bool
	// ContainsAll is whether other occurs as a sequence of characters.
	ContainsAll(other Characters) bool
	StartsWith(other Characters) bool
	EndsWith(other Characters) bool
	FindFirst(characters Characters) CharacterRange
	FindLast(characters Characters) CharacterRange

	Where(test func(string) bool) Characters
	Skip(count int) Characters
	Take(count int) Characters
	// GetRange is getRange(start, [end]) (end omitted: to the end).
	GetRange(start int, end ...int) Characters
	CharacterAt(position int) Characters
	SkipLast(count int) Characters
	TakeLast(count int) Characters
	SkipWhile(test func(string) bool) Characters
	TakeWhile(test func(string) bool) Characters
	SkipLastWhile(test func(string) bool) Characters
	TakeLastWhile(test func(string) bool) Characters
	// Add is operator +.
	Add(other Characters) Characters
	ReplaceAll(pattern, replacement Characters) Characters
	// Split is split(pattern, [maxParts = 0]).
	Split(pattern Characters, maxParts ...int) iter.Seq[Characters]
	ReplaceFirst(pattern, replacement Characters) Characters
	ToLowerCase() Characters
	ToUpperCase() Characters
	// Equal is ==.
	Equal(other Characters) bool
}

// CharactersEmpty is Characters.empty.
var CharactersEmpty Characters = newStringCharacters("")

// NewCharacters is Characters(string).
func NewCharacters(s string) Characters {
	if s == "" {
		return CharactersEmpty
	}
	return newStringCharacters(s)
}

// CharacterRange is a range of characters of a Characters, also an
// iterator (Dart's Iterator<String>: MoveNext and Current). A nil
// CharacterRange is Dart's null.
type CharacterRange interface {
	Source() Characters
	// UTF16CodeUnits are the code units of the current range.
	UTF16CodeUnits() iter.Seq[int]
	// Runes are the code points of the current range.
	Runes() iter.Seq[rune]
	Current() string
	CurrentCharacters() Characters
	CharactersBefore() Characters
	CharactersAfter() Characters
	StringBefore() string
	StringBeforeLength() int
	StringAfter() string
	StringAfterLength() int
	Copy() CharacterRange
	IsEmpty() bool
	IsNotEmpty() bool

	// MoveNext is moveNext([count = 1]).
	MoveNext(count ...int) bool
	MoveNextAll()
	MoveTo(target Characters) bool
	MoveUntil(target Characters) bool
	MoveBack(count ...int) bool
	MoveBackAll()
	MoveBackTo(target Characters) bool
	MoveBackUntil(target Characters) bool
	ExpandNext(count ...int) bool
	ExpandTo(target Characters) bool
	ExpandUntil(target Characters) bool
	ExpandWhile(test func(string) bool)
	ExpandAll()
	ExpandBack(count ...int) bool
	ExpandBackTo(target Characters) bool
	ExpandBackUntil(target Characters) bool
	ExpandBackWhile(test func(string) bool)
	ExpandBackAll()
	CollapseToStart()
	CollapseToFirst(target Characters) bool
	CollapseToLast(target Characters) bool
	CollapseToEnd()
	DropFirst(count ...int) bool
	DropTo(target Characters) bool
	DropUntil(target Characters) bool
	DropWhile(test func(string) bool)
	DropLast(count ...int) bool
	DropBackTo(target Characters) bool
	DropBackUntil(target Characters) bool
	DropBackWhile(test func(string) bool)
	ReplaceRange(replacement Characters) CharacterRange
	ReplaceAll(pattern, replacement Characters) CharacterRange
	Split(pattern Characters, maxParts ...int) iter.Seq[CharacterRange]
	ReplaceFirst(pattern, replacement Characters) CharacterRange
	StartsWith(characters Characters) bool
	EndsWith(characters Characters) bool
	IsPrecededBy(characters Characters) bool
	IsFollowedBy(characters Characters) bool
}

// NewCharacterRange is CharacterRange(string): an empty range at the start
// of string.
func NewCharacterRange(s string) CharacterRange { return newStringCharacterRange(s) }

// NewCharacterRangeAt is CharacterRange.at(string, startIndex, [endIndex]):
// the range expanded to grapheme cluster boundaries.
func NewCharacterRangeAt(s string, startIndex int, endIndex ...int) CharacterRange {
	return newStringCharacterRangeAt(s, startIndex, endIndex...)
}
