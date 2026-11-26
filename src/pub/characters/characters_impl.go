package characters

// Port of lib/src/characters_impl.dart (package:characters 1.4.1):
// StringCharacters and StringCharacterRange (unexported: the library
// exposes them only as Characters and CharacterRange).

import (
	"iter"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// --- Dart String operations on code units

type ustr = []uint16

func toStr(u ustr) string { return dart.StringFromCharCodes(u) }

func substring(u ustr, start int, end ...int) string {
	e := len(u)
	if len(end) > 0 {
		e = end[0]
	}
	if start < 0 || start > len(u) {
		panic(dart.NewRangeErrorRange(start, 0, len(u), "start", ""))
	}
	if e < start || e > len(u) {
		panic(dart.NewRangeErrorRange(e, start, len(u), "end", ""))
	}
	return toStr(u[start:e])
}

// strStartsWith is String.startsWith(pattern, index).
func strStartsWith(u, pattern ustr, index int) bool {
	if index < 0 || index > len(u) {
		panic(dart.NewRangeErrorRange(index, 0, len(u), "index", ""))
	}
	if index+len(pattern) > len(u) {
		return false
	}
	for i, c := range pattern {
		if u[index+i] != c {
			return false
		}
	}
	return true
}

// strIndexOf is String.indexOf(pattern, start).
func strIndexOf(u, pattern ustr, start int) int {
	if start < 0 || start > len(u) {
		panic(dart.NewRangeErrorRange(start, 0, len(u), "start", ""))
	}
	for i := start; i+len(pattern) <= len(u); i++ {
		if strStartsWith(u, pattern, i) {
			return i
		}
	}
	return -1
}

// strLastIndexOf is String.lastIndexOf(pattern, start).
func strLastIndexOf(u, pattern ustr, start int) int {
	if start < 0 || start > len(u) {
		panic(dart.NewRangeErrorRange(start, 0, len(u), "start", ""))
	}
	if start+len(pattern) > len(u) {
		start = len(u) - len(pattern)
	}
	for i := start; i >= 0; i-- {
		if strStartsWith(u, pattern, i) {
			return i
		}
	}
	return -1
}

func replaceRange(u ustr, start, end int, replacement ustr) ustr {
	out := make(ustr, 0, len(u)-(end-start)+len(replacement))
	out = append(out, u[:start]...)
	out = append(out, replacement...)
	return append(out, u[end:]...)
}

// notNegativeError is the RangeError RangeError.range(value, min, null, name)
// (and RangeError.checkNotNegative) throws.
type notNegativeError struct {
	value, min int
	name       string
}

func (e notNegativeError) Error() string {
	return "RangeError (" + e.name + "): Invalid value: Not greater than or equal to " + strconv.Itoa(e.min) + ": " + strconv.Itoa(e.value)
}

func checkNotNegative(value int, name string) {
	if value < 0 {
		panic(notNegativeError{value, 0, name})
	}
}

func countArg(count []int) int {
	if len(count) > 0 {
		return count[0]
	}
	return 1
}

// --- StringCharacters

// stringCharacters is StringCharacters: the grapheme clusters of a string.
type stringCharacters struct {
	string string // canonical WTF-8
	units  ustr
}

func newStringCharacters(s string) *stringCharacters {
	u := dart.StringCodeUnits(s)
	return &stringCharacters{string: toStr(u), units: u}
}

func newStringCharactersUnits(u ustr) *stringCharacters {
	return &stringCharacters{string: toStr(u), units: u}
}

func (c *stringCharacters) String() string { return c.string }

func unitsOf(c Characters) ustr {
	if sc, ok := c.(*stringCharacters); ok {
		return sc.units
	}
	return dart.StringCodeUnits(c.String())
}

func (c *stringCharacters) All() iter.Seq[string] {
	return func(yield func(string) bool) {
		it := c.Iterator()
		for it.MoveNext() {
			if !yield(it.Current()) {
				return
			}
		}
	}
}

func (c *stringCharacters) Iterator() CharacterRange { return newRange(c.units, 0, 0) }

func (c *stringCharacters) IteratorAtEnd() CharacterRange {
	return newRange(c.units, len(c.units), len(c.units))
}

func (c *stringCharacters) rangeAll() *stringCharacterRange { return newRange(c.units, 0, len(c.units)) }

func (c *stringCharacters) First() string {
	if len(c.units) == 0 {
		panic(dart.StateError{Message: "No element"})
	}
	return substring(c.units, 0, newBreaks(c.units, 0, len(c.units), stateSoTNoBreak).nextBreak())
}

func (c *stringCharacters) Last() string {
	if len(c.units) == 0 {
		panic(dart.StateError{Message: "No element"})
	}
	return substring(c.units, newBackBreaks(c.units, len(c.units), 0, stateEoTNoBreak).nextBreak())
}

func (c *stringCharacters) Single() string {
	if len(c.units) == 0 {
		panic(dart.StateError{Message: "No element"})
	}
	firstEnd := newBreaks(c.units, 0, len(c.units), stateSoTNoBreak).nextBreak()
	if firstEnd == len(c.units) {
		return c.string
	}
	panic(dart.StateError{Message: "Too many elements"})
}

func (c *stringCharacters) IsEmpty() bool    { return len(c.units) == 0 }
func (c *stringCharacters) IsNotEmpty() bool { return len(c.units) != 0 }

func (c *stringCharacters) Length() int {
	if len(c.units) == 0 {
		return 0
	}
	brk := newBreaks(c.units, 0, len(c.units), stateSoTNoBreak)
	length := 0
	for brk.nextBreak() >= 0 {
		length++
	}
	return length
}

func (c *stringCharacters) Join(separator ...string) string {
	sep := ""
	if len(separator) > 0 {
		sep = separator[0]
	}
	if sep == "" {
		return c.string
	}
	return toStr(explodeReplace(c.units, 0, len(c.units), dart.StringCodeUnits(sep), nil))
}

func (c *stringCharacters) LastWhere(test func(string) bool, orElse ...func() string) string {
	cursor := len(c.units)
	brk := newBackBreaks(c.units, cursor, 0, stateEoTNoBreak)
	for next := brk.nextBreak(); next >= 0; next = brk.nextBreak() {
		current := substring(c.units, next, cursor)
		if test(current) {
			return current
		}
		cursor = next
	}
	if len(orElse) > 0 && orElse[0] != nil {
		return orElse[0]()
	}
	panic(dart.StateError{Message: "No element"})
}

func (c *stringCharacters) ElementAt(index int) string {
	checkNotNegative(index, "index")
	count := 0
	if len(c.units) != 0 {
		brks := newBreaks(c.units, 0, len(c.units), stateSoTNoBreak)
		start := 0
		for end := brks.nextBreak(); end >= 0; end = brks.nextBreak() {
			if count == index {
				return substring(c.units, start, end)
			}
			count++
			start = end
		}
	}
	panic(dart.IndexError{Index: index, Length: count, Name: "index"})
}

func (c *stringCharacters) Contains(singleCharacterString string) bool {
	u := dart.StringCodeUnits(singleCharacterString)
	if len(u) == 0 {
		return false
	}
	next := newBreaks(u, 0, len(u), stateSoTNoBreak).nextBreak()
	if next != len(u) {
		return false
	}
	// singleCharacterString is a single grapheme cluster.
	return indexOf(c.units, u, 0, len(c.units)) >= 0
}

func (c *stringCharacters) StartsWith(characters Characters) bool {
	length := len(c.units)
	other := unitsOf(characters)
	if len(other) == 0 {
		return true
	}
	return strStartsWith(c.units, other, 0) && isGraphemeClusterBoundary(c.units, 0, length, len(other))
}

func (c *stringCharacters) EndsWith(characters Characters) bool {
	length := len(c.units)
	other := unitsOf(characters)
	if len(other) == 0 {
		return true
	}
	start := len(c.units) - len(other)
	return start >= 0 && strStartsWith(c.units, other, start) && isGraphemeClusterBoundary(c.units, 0, length, start)
}

func (c *stringCharacters) ReplaceAll(pattern, replacement Characters) Characters {
	if r := c.rangeAll().ReplaceAll(pattern, replacement); r != nil {
		return r.Source()
	}
	return c
}

func (c *stringCharacters) ReplaceFirst(pattern, replacement Characters) Characters {
	if r := c.rangeAll().ReplaceFirst(pattern, replacement); r != nil {
		return r.Source()
	}
	return c
}

func maxPartsArg(maxParts []int) int {
	if len(maxParts) > 0 {
		return maxParts[0]
	}
	return 0
}

func (c *stringCharacters) Split(pattern Characters, maxParts ...int) iter.Seq[Characters] {
	mp := maxPartsArg(maxParts)
	return func(yield func(Characters) bool) {
		maxParts := mp
		if maxParts == 1 || len(c.units) == 0 {
			yield(c)
			return
		}
		patternString := unitsOf(pattern)
		start := 0
		if len(patternString) != 0 {
			for {
				match := indexOf(c.units, patternString, start, len(c.units))
				if match < 0 {
					break
				}
				if !yield(newStringCharactersUnits(c.units[start:match])) {
					return
				}
				start = match + len(patternString)
				maxParts--
				if maxParts == 1 {
					break
				}
			}
		} else {
			// Empty pattern. Split on internal boundaries only.
			brks := newBreaks(c.units, 0, len(c.units), stateSoTNoBreak)
			for {
				match := brks.nextBreak()
				if match < 0 {
					return
				}
				if !yield(newStringCharactersUnits(c.units[start:match])) {
					return
				}
				start = match
				maxParts--
				if maxParts == 1 {
					break
				}
			}
			if start == len(c.units) {
				return
			}
		}
		yield(newStringCharactersUnits(c.units[start:]))
	}
}

func (c *stringCharacters) ContainsAll(characters Characters) bool {
	return indexOf(c.units, unitsOf(characters), 0, len(c.units)) >= 0
}

// skipIndices returns the break position of the count'th break, starting
// from cursor (using brks, assumed to be at cursor, if given).
func (c *stringCharacters) skipIndices(count, cursor int, brks *breaks) int {
	if count == 0 || cursor == len(c.units) {
		return cursor
	}
	if brks == nil {
		brks = newBreaks(c.units, cursor, len(c.units), stateSoTNoBreak)
	}
	for {
		nb := brks.nextBreak()
		if nb < 0 {
			break
		}
		cursor = nb
		count--
		if count <= 0 {
			break
		}
	}
	return cursor
}

func (c *stringCharacters) Skip(count int) Characters {
	checkNotNegative(count, "count")
	return c.skip(count)
}

func (c *stringCharacters) skip(count int) Characters {
	start := c.skipIndices(count, 0, nil)
	if start == len(c.units) {
		return CharactersEmpty
	}
	return newStringCharactersUnits(c.units[start:])
}

func (c *stringCharacters) Take(count int) Characters {
	checkNotNegative(count, "count")
	return c.take(count)
}

func (c *stringCharacters) take(count int) Characters {
	end := c.skipIndices(count, 0, nil)
	if end == len(c.units) {
		return c
	}
	return newStringCharactersUnits(c.units[:end])
}

func (c *stringCharacters) GetRange(start int, end ...int) Characters {
	checkNotNegative(start, "start")
	if len(end) == 0 {
		return c.skip(start)
	}
	e := end[0]
	if e < start {
		panic(notNegativeError{e, start, "end"})
	}
	if e == start {
		return CharactersEmpty
	}
	if start == 0 {
		return c.take(e)
	}
	if len(c.units) == 0 {
		return c
	}
	brks := newBreaks(c.units, 0, len(c.units), stateSoTNoBreak)
	startIndex := c.skipIndices(start, 0, brks)
	if startIndex == len(c.units) {
		return CharactersEmpty
	}
	endIndex := c.skipIndices(e-start, start, brks)
	return newStringCharactersUnits(c.units[startIndex:endIndex])
}

func (c *stringCharacters) CharacterAt(position int) Characters {
	brks := newBreaks(c.units, 0, len(c.units), stateSoTNoBreak)
	start := 0
	for position > 0 {
		position--
		start = brks.nextBreak()
		if start < 0 {
			panic(dart.StateError{Message: "No element"})
		}
	}
	end := brks.nextBreak()
	if end < 0 {
		panic(dart.StateError{Message: "No element"})
	}
	if start == 0 && end == len(c.units) {
		return c
	}
	return newStringCharactersUnits(c.units[start:end])
}

func (c *stringCharacters) SkipWhile(test func(string) bool) Characters {
	if len(c.units) != 0 {
		stringLength := len(c.units)
		brks := newBreaks(c.units, 0, stringLength, stateSoTNoBreak)
		startIndex := 0
		for index := brks.nextBreak(); index >= 0; index = brks.nextBreak() {
			if !test(substring(c.units, startIndex, index)) {
				if startIndex == 0 {
					return c
				}
				if startIndex == stringLength {
					return CharactersEmpty
				}
				return newStringCharactersUnits(c.units[startIndex:])
			}
			startIndex = index
		}
	}
	return CharactersEmpty
}

func (c *stringCharacters) TakeWhile(test func(string) bool) Characters {
	if len(c.units) != 0 {
		brks := newBreaks(c.units, 0, len(c.units), stateSoTNoBreak)
		endIndex := 0
		for index := brks.nextBreak(); index >= 0; index = brks.nextBreak() {
			if !test(substring(c.units, endIndex, index)) {
				if endIndex == 0 {
					return CharactersEmpty
				}
				return newStringCharactersUnits(c.units[:endIndex])
			}
			endIndex = index
		}
	}
	return c
}

func (c *stringCharacters) Where(test func(string) bool) Characters {
	var b strings.Builder
	for s := range c.All() {
		if test(s) {
			b.WriteString(s)
		}
	}
	s := dart.Concat(b.String())
	if s == "" {
		return CharactersEmpty
	}
	return newStringCharacters(s)
}

func (c *stringCharacters) Add(characters Characters) Characters {
	return newStringCharactersUnits(append(append(ustr{}, c.units...), unitsOf(characters)...))
}

func (c *stringCharacters) SkipLast(count int) Characters {
	checkNotNegative(count, "count")
	if count == 0 {
		return c
	}
	if len(c.units) != 0 {
		brks := newBackBreaks(c.units, len(c.units), 0, stateEoTNoBreak)
		endIndex := len(c.units)
		for count > 0 {
			index := brks.nextBreak()
			if index >= 0 {
				endIndex = index
				count--
			} else {
				return CharactersEmpty
			}
		}
		if endIndex > 0 {
			return newStringCharactersUnits(c.units[:endIndex])
		}
	}
	return CharactersEmpty
}

func (c *stringCharacters) SkipLastWhile(test func(string) bool) Characters {
	if len(c.units) != 0 {
		brks := newBackBreaks(c.units, len(c.units), 0, stateEoTNoBreak)
		end := len(c.units)
		for index := brks.nextBreak(); index >= 0; index = brks.nextBreak() {
			if !test(substring(c.units, index, end)) {
				if end == len(c.units) {
					return c
				}
				if end == 0 {
					return CharactersEmpty
				}
				return newStringCharactersUnits(c.units[:end])
			}
			end = index
		}
	}
	return CharactersEmpty
}

func (c *stringCharacters) TakeLast(count int) Characters {
	checkNotNegative(count, "count")
	if count == 0 {
		return CharactersEmpty
	}
	if len(c.units) != 0 {
		brks := newBackBreaks(c.units, len(c.units), 0, stateEoTNoBreak)
		startIndex := len(c.units)
		for count > 0 {
			index := brks.nextBreak()
			if index >= 0 {
				startIndex = index
				count--
			} else {
				return c
			}
		}
		if startIndex > 0 {
			return newStringCharactersUnits(c.units[startIndex:])
		}
	}
	return c
}

func (c *stringCharacters) TakeLastWhile(test func(string) bool) Characters {
	if len(c.units) != 0 {
		brks := newBackBreaks(c.units, len(c.units), 0, stateEoTNoBreak)
		start := len(c.units)
		for index := brks.nextBreak(); index >= 0; index = brks.nextBreak() {
			if !test(substring(c.units, index, start)) {
				if start == len(c.units) {
					return CharactersEmpty
				}
				return newStringCharactersUnits(c.units[start:])
			}
			start = index
		}
	}
	return c
}

// ToLowerCase is toLowerCase (Go's strings.ToLower: Unicode simple case
// mapping, see the package's API notes).
func (c *stringCharacters) ToLowerCase() Characters {
	return newStringCharacters(strings.ToLower(c.string))
}

// ToUpperCase is toUpperCase (Go's strings.ToUpper).
func (c *stringCharacters) ToUpperCase() Characters {
	return newStringCharacters(strings.ToUpper(c.string))
}

func (c *stringCharacters) Equal(other Characters) bool {
	return other != nil && c.string == dart.Concat(other.String())
}

func (c *stringCharacters) FindFirst(characters Characters) CharacterRange {
	r := c.rangeAll()
	if r.CollapseToFirst(characters) {
		return r
	}
	return nil
}

func (c *stringCharacters) FindLast(characters Characters) CharacterRange {
	r := c.rangeAll()
	if r.CollapseToLast(characters) {
		return r
	}
	return nil
}

// --- StringCharacterRange

// stringCharacterRange is StringCharacterRange: a range of a single string
// whose start and end are grapheme cluster boundaries.
type stringCharacterRange struct {
	str          ustr
	start, end   int
	currentCache *string
}

func newRange(s ustr, start, end int) *stringCharacterRange {
	return &stringCharacterRange{str: s, start: start, end: end}
}

func newStringCharacterRange(s string) *stringCharacterRange {
	return newRange(dart.StringCodeUnits(s), 0, 0)
}

func newStringCharacterRangeAt(s string, startIndex int, endIndex ...int) *stringCharacterRange {
	u := dart.StringCodeUnits(s)
	end := startIndex
	// RangeError.checkValidRange(startIndex, endIndex, string.length, 'startIndex', 'endIndex').
	if startIndex < 0 || startIndex > len(u) {
		panic(dart.NewRangeErrorRange(startIndex, 0, len(u), "startIndex", ""))
	}
	if len(endIndex) > 0 {
		end = endIndex[0]
		if end < startIndex || end > len(u) {
			panic(dart.NewRangeErrorRange(end, startIndex, len(u), "endIndex", ""))
		}
	}
	return expandRange(u, startIndex, end)
}

// move changes the current range, resetting the cached state.
func (r *stringCharacterRange) move(start, end int) {
	r.start = start
	r.end = end
	r.currentCache = nil
}

func (r *stringCharacterRange) breaksFromEnd() *breaks {
	return newBreaks(r.str, r.end, len(r.str), stateSoTNoBreak)
}

func (r *stringCharacterRange) backBreaksFromStart() *backBreaks {
	return newBackBreaks(r.str, r.start, 0, stateEoTNoBreak)
}

func (r *stringCharacterRange) Current() string {
	if r.currentCache == nil {
		s := substring(r.str, r.start, r.end)
		r.currentCache = &s
	}
	return *r.currentCache
}

func (r *stringCharacterRange) MoveNext(count ...int) bool { return r.advanceEnd(countArg(count), r.end) }

func (r *stringCharacterRange) advanceEnd(count, newStart int) bool {
	if count > 0 {
		state := stateSoTNoBreak
		index := r.end
		for index < len(r.str) {
			char := int(r.str[index])
			surrogate := char ^ 0xD800
			category := categoryControl
			nextIndex := index + 1
			if surrogate > 0x3FF {
				category = low(char)
			} else if nextIndex < len(r.str) {
				nextSurrogate := int(r.str[nextIndex]) ^ 0xDC00
				if nextSurrogate <= 0x3FF {
					nextIndex++
					category = high(surrogate, nextSurrogate)
				}
			}
			state = move(state, category)
			if state&maskBreak != flagNoBreak {
				count--
				if count == 0 {
					r.move(newStart, index)
					return true
				}
			}
			index = nextIndex
		}
		r.move(newStart, len(r.str))
		return count == 1 && state != stateSoTNoBreak
	} else if count == 0 {
		r.move(newStart, r.end)
		return true
	}
	panic(notNegativeError{count, 0, "count"})
}

func (r *stringCharacterRange) moveNextPattern(pattern ustr, start, end int) bool {
	offset := indexOf(r.str, pattern, start, end)
	if offset >= 0 {
		r.move(offset, offset+len(pattern))
		return true
	}
	return false
}

func (r *stringCharacterRange) MoveBack(count ...int) bool {
	return r.retractStart(countArg(count), r.start)
}

func (r *stringCharacterRange) retractStart(count, newEnd int) bool {
	checkNotNegative(count, "count")
	brks := r.backBreaksFromStart()
	start := r.start
	for count > 0 {
		nb := brks.nextBreak()
		if nb >= 0 {
			start = nb
		} else {
			break
		}
		count--
	}
	r.move(start, newEnd)
	return count == 0
}

func (r *stringCharacterRange) movePreviousPattern(pattern ustr, start, end int) bool {
	offset := lastIndexOf(r.str, pattern, start, end)
	if offset >= 0 {
		r.move(offset, offset+len(pattern))
		return true
	}
	return false
}

func (r *stringCharacterRange) UTF16CodeUnits() iter.Seq[int] {
	units := r.str[r.start:r.end]
	return func(yield func(int) bool) {
		for _, u := range units {
			if !yield(int(u)) {
				return
			}
		}
	}
}

// Runes are Runes(current): the code points, an unpaired surrogate as
// itself.
func (r *stringCharacterRange) Runes() iter.Seq[rune] {
	units := r.str[r.start:r.end]
	return func(yield func(rune) bool) {
		for i := 0; i < len(units); i++ {
			c := rune(units[i])
			if c >= 0xD800 && c < 0xDC00 && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] < 0xE000 {
				c = 0x10000 + (c-0xD800)<<10 + rune(units[i+1]) - 0xDC00
				i++
			}
			if !yield(c) {
				return
			}
		}
	}
}

func (r *stringCharacterRange) Copy() CharacterRange { return newRange(r.str, r.start, r.end) }

func (r *stringCharacterRange) CollapseToEnd()   { r.move(r.end, r.end) }
func (r *stringCharacterRange) CollapseToStart() { r.move(r.start, r.start) }

func (r *stringCharacterRange) DropFirst(count ...int) bool {
	n := countArg(count)
	checkNotNegative(n, "count")
	if r.start == r.end {
		return n == 0
	}
	brks := newBreaks(r.str, r.start, r.end, stateSoTNoBreak)
	for n > 0 {
		nb := brks.nextBreak()
		if nb >= 0 {
			r.start = nb
			r.currentCache = nil
			n--
		} else {
			return false
		}
	}
	return true
}

func (r *stringCharacterRange) DropTo(target Characters) bool {
	if r.start == r.end {
		return target.IsEmpty()
	}
	t := unitsOf(target)
	index := indexOf(r.str, t, r.start, r.end)
	if index >= 0 {
		r.move(index+len(t), r.end)
		return true
	}
	return false
}

func (r *stringCharacterRange) DropUntil(target Characters) bool {
	if r.start == r.end {
		return target.IsEmpty()
	}
	t := unitsOf(target)
	index := indexOf(r.str, t, r.start, r.end)
	if index >= 0 {
		r.move(index, r.end)
		return true
	}
	r.move(r.end, r.end)
	return false
}

func (r *stringCharacterRange) DropWhile(test func(string) bool) {
	if r.start == r.end {
		return
	}
	brks := newBreaks(r.str, r.start, r.end, stateSoTNoBreak)
	cursor := r.start
	for next := brks.nextBreak(); next >= 0; next = brks.nextBreak() {
		if !test(substring(r.str, cursor, next)) {
			break
		}
		cursor = next
	}
	r.move(cursor, r.end)
}

func (r *stringCharacterRange) DropLast(count ...int) bool {
	n := countArg(count)
	checkNotNegative(n, "count")
	brks := newBackBreaks(r.str, r.end, r.start, stateEoTNoBreak)
	for n > 0 {
		nb := brks.nextBreak()
		if nb >= 0 {
			r.end = nb
			r.currentCache = nil
			n--
		} else {
			return false
		}
	}
	return true
}

func (r *stringCharacterRange) DropBackTo(target Characters) bool {
	if r.start == r.end {
		return target.IsEmpty()
	}
	t := unitsOf(target)
	index := lastIndexOf(r.str, t, r.start, r.end)
	if index >= 0 {
		r.move(r.start, index)
		return true
	}
	return false
}

func (r *stringCharacterRange) DropBackUntil(target Characters) bool {
	if r.start == r.end {
		return target.IsEmpty()
	}
	t := unitsOf(target)
	index := lastIndexOf(r.str, t, r.start, r.end)
	if index >= 0 {
		r.move(r.start, index+len(t))
		return true
	}
	r.move(r.start, r.start)
	return false
}

func (r *stringCharacterRange) DropBackWhile(test func(string) bool) {
	if r.start == r.end {
		return
	}
	brks := newBackBreaks(r.str, r.end, r.start, stateEoTNoBreak)
	cursor := r.end
	for next := brks.nextBreak(); next >= 0; next = brks.nextBreak() {
		if !test(substring(r.str, next, cursor)) {
			break
		}
		cursor = next
	}
	r.move(r.start, cursor)
}

func (r *stringCharacterRange) ExpandNext(count ...int) bool {
	return r.advanceEnd(countArg(count), r.start)
}

func (r *stringCharacterRange) ExpandTo(target Characters) bool {
	t := unitsOf(target)
	index := indexOf(r.str, t, r.end, len(r.str))
	if index >= 0 {
		r.move(r.start, index+len(t))
		return true
	}
	return false
}

func (r *stringCharacterRange) ExpandWhile(test func(string) bool) {
	brks := r.breaksFromEnd()
	cursor := r.end
	for next := brks.nextBreak(); next >= 0; next = brks.nextBreak() {
		if !test(substring(r.str, cursor, next)) {
			break
		}
		cursor = next
	}
	r.move(r.start, cursor)
}

func (r *stringCharacterRange) ExpandAll() { r.move(r.start, len(r.str)) }

func (r *stringCharacterRange) ExpandBack(count ...int) bool {
	return r.retractStart(countArg(count), r.end)
}

func (r *stringCharacterRange) ExpandBackTo(target Characters) bool {
	t := unitsOf(target)
	index := lastIndexOf(r.str, t, 0, r.start)
	if index >= 0 {
		r.move(index, r.end)
		return true
	}
	return false
}

func (r *stringCharacterRange) ExpandBackWhile(test func(string) bool) {
	brks := r.backBreaksFromStart()
	cursor := r.start
	for next := brks.nextBreak(); next >= 0; next = brks.nextBreak() {
		if !test(substring(r.str, next, cursor)) {
			r.move(cursor, r.end)
			return
		}
		cursor = next
	}
	r.move(0, r.end)
}

func (r *stringCharacterRange) ExpandBackUntil(target Characters) bool {
	return r.retractStartUntil(unitsOf(target), r.end)
}

func (r *stringCharacterRange) ExpandBackAll() { r.move(0, r.end) }

func (r *stringCharacterRange) ExpandUntil(target Characters) bool {
	return r.advanceEndUntil(unitsOf(target), r.start)
}

func (r *stringCharacterRange) IsEmpty() bool    { return r.start == r.end }
func (r *stringCharacterRange) IsNotEmpty() bool { return r.start != r.end }

func (r *stringCharacterRange) MoveBackUntil(target Characters) bool {
	return r.retractStartUntil(unitsOf(target), r.start)
}

func (r *stringCharacterRange) retractStartUntil(target ustr, newEnd int) bool {
	index := lastIndexOf(r.str, target, 0, r.start)
	if index >= 0 {
		r.move(index+len(target), newEnd)
		return true
	}
	r.move(0, newEnd)
	return false
}

func (r *stringCharacterRange) CollapseToFirst(target Characters) bool {
	return r.moveNextPattern(unitsOf(target), r.start, r.end)
}

func (r *stringCharacterRange) CollapseToLast(target Characters) bool {
	return r.movePreviousPattern(unitsOf(target), r.start, r.end)
}

func (r *stringCharacterRange) MoveUntil(target Characters) bool {
	return r.advanceEndUntil(unitsOf(target), r.end)
}

func (r *stringCharacterRange) advanceEndUntil(target ustr, newStart int) bool {
	index := indexOf(r.str, target, r.end, len(r.str))
	if index >= 0 {
		r.move(newStart, index)
		return true
	}
	r.move(newStart, len(r.str))
	return false
}

func (r *stringCharacterRange) ReplaceFirst(pattern, replacement Characters) CharacterRange {
	p := unitsOf(pattern)
	rep := unitsOf(replacement)
	var replaced ustr
	if len(p) == 0 {
		replaced = replaceRange(r.str, r.start, r.start, rep)
	} else {
		index := indexOf(r.str, p, r.start, r.end)
		if index < 0 {
			return nil
		}
		replaced = replaceRange(r.str, index, index+len(p), rep)
	}
	newEnd := len(replaced) - len(r.str) + r.end
	return expandRange(replaced, r.start, newEnd)
}

func (r *stringCharacterRange) ReplaceAll(pattern, replacement Characters) CharacterRange {
	p := unitsOf(pattern)
	rep := unitsOf(replacement)
	if len(p) == 0 {
		replaced := explodeReplace(r.str, r.start, r.end, rep, rep)
		newEnd := len(replaced) - (len(r.str) - r.end)
		return expandRange(replaced, r.start, newEnd)
	}
	if r.start == r.end {
		return nil
	}
	start := 0
	cursor := r.start
	var buffer ustr
	found := false
	for {
		cursor = indexOf(r.str, p, cursor, r.end)
		if cursor < 0 {
			break
		}
		found = true
		buffer = append(buffer, r.str[start:cursor]...)
		buffer = append(buffer, rep...)
		cursor += len(p)
		start = cursor
	}
	if !found {
		return nil
	}
	buffer = append(buffer, r.str[start:]...)
	newEnd := len(buffer) - (len(r.str) - r.end)
	return expandRange(buffer, r.start, newEnd)
}

func (r *stringCharacterRange) ReplaceRange(replacement Characters) CharacterRange {
	rep := unitsOf(replacement)
	result := replaceRange(r.str, r.start, r.end, rep)
	return expandRange(result, r.start, r.start+len(rep))
}

// expandRange expands a range whose start or end is not a grapheme cluster
// boundary (no validation: 0 <= start <= end <= len).
func expandRange(s ustr, start, end int) *stringCharacterRange {
	start = previousBreak(s, 0, len(s), start)
	if end != start {
		end = nextBreak(s, 0, len(s), end)
	}
	return newRange(s, start, end)
}

func (r *stringCharacterRange) Source() Characters { return NewCharacters(toStr(r.str)) }

func (r *stringCharacterRange) StartsWith(characters Characters) bool {
	return r.startsWith(r.start, r.end, unitsOf(characters))
}

func (r *stringCharacterRange) EndsWith(characters Characters) bool {
	return r.endsWith(r.start, r.end, unitsOf(characters))
}

func (r *stringCharacterRange) IsFollowedBy(characters Characters) bool {
	return r.startsWith(r.end, len(r.str), unitsOf(characters))
}

func (r *stringCharacterRange) IsPrecededBy(characters Characters) bool {
	return r.endsWith(0, r.start, unitsOf(characters))
}

func (r *stringCharacterRange) endsWith(start, end int, s ustr) bool {
	stringStart := end - len(s)
	return stringStart >= start && strStartsWith(r.str, s, stringStart) &&
		isGraphemeClusterBoundary(r.str, start, end, stringStart)
}

func (r *stringCharacterRange) startsWith(start, end int, s ustr) bool {
	stringEnd := start + len(s)
	return stringEnd <= end && strStartsWith(r.str, s, start) &&
		isGraphemeClusterBoundary(r.str, start, end, stringEnd)
}

func (r *stringCharacterRange) MoveBackTo(target Characters) bool {
	t := unitsOf(target)
	index := lastIndexOf(r.str, t, 0, r.start)
	if index >= 0 {
		r.move(index, index+len(t))
		return true
	}
	return false
}

func (r *stringCharacterRange) MoveTo(target Characters) bool {
	t := unitsOf(target)
	index := indexOf(r.str, t, r.end, len(r.str))
	if index >= 0 {
		r.move(index, index+len(t))
		return true
	}
	return false
}

func (r *stringCharacterRange) CharactersAfter() Characters {
	return newStringCharactersUnits(r.str[r.end:])
}

func (r *stringCharacterRange) CharactersBefore() Characters {
	return newStringCharactersUnits(r.str[:r.start])
}

func (r *stringCharacterRange) CurrentCharacters() Characters { return newStringCharacters(r.Current()) }

func (r *stringCharacterRange) MoveBackAll() { r.move(0, r.start) }
func (r *stringCharacterRange) MoveNextAll() { r.move(r.end, len(r.str)) }

func (r *stringCharacterRange) StringAfter() string     { return substring(r.str, r.end) }
func (r *stringCharacterRange) StringAfterLength() int  { return len(r.str) - r.end }
func (r *stringCharacterRange) StringBefore() string    { return substring(r.str, 0, r.start) }
func (r *stringCharacterRange) StringBeforeLength() int { return r.start }

func (r *stringCharacterRange) Split(pattern Characters, maxParts ...int) iter.Seq[CharacterRange] {
	mp := maxPartsArg(maxParts)
	return func(yield func(CharacterRange) bool) {
		maxParts := mp
		if maxParts == 1 || r.start == r.end {
			yield(r)
			return
		}
		p := unitsOf(pattern)
		start := r.start
		if len(p) != 0 {
			for {
				match := indexOf(r.str, p, start, r.end)
				if match < 0 {
					break
				}
				if !yield(newRange(r.str, start, match)) {
					return
				}
				start = match + len(p)
				maxParts--
				if maxParts == 1 {
					break
				}
			}
			yield(newRange(r.str, start, r.end))
		} else {
			// Empty pattern. Split on internal boundaries only.
			brks := newBreaks(r.str, r.start, r.end, stateSoTNoBreak)
			for {
				match := brks.nextBreak()
				if match < 0 {
					return
				}
				if !yield(newRange(r.str, start, match)) {
					return
				}
				start = match
				maxParts--
				if maxParts == 1 {
					break
				}
			}
			if start < r.end {
				yield(newRange(r.str, start, r.end))
			}
		}
	}
}

// explodeReplace inserts internalReplacement between the characters of
// s[start:end] and outerReplacement around them (nil: empty).
func explodeReplace(s ustr, start, end int, internalReplacement, outerReplacement ustr) ustr {
	if start == end {
		return replaceRange(s, start, start, outerReplacement)
	}
	buffer := append(ustr{}, s[:start]...)
	brks := newBreaks(s, start, end, stateSoTNoBreak)
	replacement := outerReplacement
	for index := brks.nextBreak(); index >= 0; index = brks.nextBreak() {
		buffer = append(buffer, replacement...)
		buffer = append(buffer, s[start:index]...)
		start = index
		replacement = internalReplacement
	}
	buffer = append(buffer, outerReplacement...)
	return append(buffer, s[end:]...)
}

// indexOf finds pattern in source[start:end] (start and end are grapheme
// cluster boundaries) at grapheme cluster boundaries.
func indexOf(source, pattern ustr, start, end int) int {
	patternLength := len(pattern)
	if patternLength == 0 {
		return start
	}
	// Any start position after realEnd won't fit the pattern before end.
	realEnd := end - patternLength
	if realEnd < start {
		return -1
	}
	// Use indexOf if what we can overshoot is less than twice as much as
	// what we have left to search.
	rest := len(source) - realEnd
	if rest <= (realEnd-start)*2 {
		for start < realEnd {
			index := strIndexOf(source, pattern, start)
			if index < 0 {
				break
			}
			if index > realEnd {
				return -1
			}
			if isGraphemeClusterBoundary(source, start, end, index) &&
				isGraphemeClusterBoundary(source, start, end, index+patternLength) {
				return index
			}
			start = index + 1
		}
		return -1
	}
	return gcIndexOf(source, pattern, start, end)
}

func gcIndexOf(source, pattern ustr, start, end int) int {
	brks := newBreaks(source, start, end, stateSoT)
	for index := brks.nextBreak(); index >= 0; index = brks.nextBreak() {
		endIndex := index + len(pattern)
		if endIndex > end {
			break
		}
		if strStartsWith(source, pattern, index) && isGraphemeClusterBoundary(source, start, end, endIndex) {
			return index
		}
	}
	return -1
}

// lastIndexOf finds pattern in source[start:end] (start and end are
// grapheme cluster boundaries), last occurrence first.
func lastIndexOf(source, pattern ustr, start, end int) int {
	patternLength := len(pattern)
	if patternLength == 0 {
		return end
	}
	// Start of pattern must be in range [start .. end - patternLength].
	realEnd := end - patternLength
	if realEnd < start {
		return -1
	}
	// If the range from 0 to start is no more than double the range from
	// start to end, use lastIndexOf.
	if realEnd*2 > start {
		for realEnd >= start {
			index := strLastIndexOf(source, pattern, realEnd)
			if index < 0 {
				break
			}
			if index < start {
				return -1
			}
			if isGraphemeClusterBoundary(source, start, end, index) &&
				isGraphemeClusterBoundary(source, start, end, index+patternLength) {
				return index
			}
			realEnd = index - 1
		}
		return -1
	}
	return gcLastIndexOf(source, pattern, start, end)
}

func gcLastIndexOf(source, pattern ustr, start, end int) int {
	brks := newBackBreaks(source, end, start, stateEoT)
	for index := brks.nextBreak(); index >= 0; index = brks.nextBreak() {
		startIndex := index - len(pattern)
		if startIndex < start {
			break
		}
		if strStartsWith(source, pattern, startIndex) && isGraphemeClusterBoundary(source, start, end, startIndex) {
			return startIndex
		}
	}
	return -1
}
