package characters

// Port of lib/src/grapheme_clusters/breaks.dart (package:characters 1.4.1),
// with table.dart's lookup functions (low, high, move, moveBack); the
// tables themselves are in the generated table.go. Strings are handled as
// their UTF-16 code units, as Dart's String is.

import "github.com/jonathanperis/flutter-go/src/dart"

func assert(cond bool) {
	if !cond {
		panic(dart.NewAssertionError(nil))
	}
}

// low is the category of a BMP code unit (table.dart's low).
func low(codeUnit int) int {
	assert(codeUnit <= 0xFFFF)
	chunkStart := int(tableStart[codeUnit>>5])
	index := chunkStart + (codeUnit & 31)
	return int(tableData[index])
}

// high is the category of the code point of a surrogate pair, given the
// low ten bits of its lead and tail surrogates (table.dart's high).
func high(lead, tail int) int {
	assert(lead <= 0x3FF && tail <= 0x3FF)
	chunkIndex := (tail >> 8) + (lead << 2)
	byteIndex := tail & 255
	chunkStart := int(tableStart[2048+chunkIndex])
	return int(tableData[chunkStart+byteIndex])
}

func move(state, inputCategory int) int {
	return int(stateMachine[(state&-4)+inputCategory])
}

func moveBack(state, inputCategory int) int {
	return int(backStateMachine[(state&-4)+inputCategory])
}

// breaks iterates grapheme cluster breaks of base[cursor:end] (Dart's
// Breaks).
type breaks struct {
	base   []uint16
	end    int
	cursor int
	state  int
}

func newBreaks(base []uint16, cursor, end, state int) *breaks {
	return &breaks{base: base, cursor: cursor, end: end, state: state}
}

func (b *breaks) copy() *breaks { return newBreaks(b.base, b.cursor, b.end, b.state) }

// nextBreak is the index of the next grapheme cluster break, or a negative
// number if there are no further breaks.
func (b *breaks) nextBreak() int {
	for b.cursor < b.end {
		breakAt := b.cursor
		b.step()
		if b.state&maskFlags != flagNoBreak {
			return breakAt
		}
	}
	b.state = move(b.state, categoryEoT)
	if b.state&maskFlags != flagNoBreak {
		return b.cursor
	}
	return -1
}

// step takes one step forward in the state machine.
func (b *breaks) step() {
	assert(b.cursor < b.end)
	char := int(b.base[b.cursor])
	b.cursor++
	surrogate := char ^ 0xD800
	if surrogate > 0x3FF {
		b.state = move(b.state, low(char))
		return
	}
	// The category of an unpaired lead surrogate is Control.
	var category int
	if b.cursor < b.end {
		if nextSurrogate := int(b.base[b.cursor]) ^ 0xDC00; nextSurrogate <= 0x3FF {
			category = high(surrogate, nextSurrogate)
			b.cursor++
			b.state = move(b.state, category)
			return
		}
	}
	category = categoryControl
	b.state = move(b.state, category)
}

// unknownPositionFirstStep starts with no knowledge about the position at
// cursor: from state CAny, it takes one step based on the latest character
// starting earlier than cursor, and returns that character's start.
func (b *breaks) unknownPositionFirstStep(start int) int {
	if b.cursor == start {
		b.state = stateSoTNoBreak
		return b.cursor
	}
	cursorBefore := b.cursor - 1
	prevChar := int(b.base[cursorBefore])
	prevSurrogate := prevChar ^ 0xD800
	if prevSurrogate > 0x7FF {
		// Not surrogate.
		prevCategory := low(prevChar)
		b.state = move(stateCAny, prevCategory)
		return cursorBefore
	}
	var prevCategory int
	if prevSurrogate > 0x3FF {
		// Tail surrogate, check for prior lead surrogate.
		leadIndex := cursorBefore - 1
		prevSurrogate &= 0x3FF
		if leadIndex >= start {
			if leadSurrogate := int(b.base[leadIndex]) ^ 0xD800; leadSurrogate <= 0x3FF {
				prevCategory = high(leadSurrogate, prevSurrogate)
				cursorBefore = leadIndex
			} else {
				prevCategory = categoryControl
			}
		} else {
			prevCategory = categoryControl
		}
	} else {
		// Lead surrogate. Check for a following tail surrogate.
		prevCategory = categoryControl
		if b.cursor < b.end {
			if tailSurrogate := int(b.base[b.cursor]) ^ 0xDC00; tailSurrogate <= 0x3FF {
				b.cursor++
				prevCategory = high(prevSurrogate, tailSurrogate)
			}
		}
	}
	b.state = move(stateCAny, prevCategory)
	return cursorBefore
}

// backBreaks iterates grapheme cluster breaks backwards, from cursor to
// start (Dart's BackBreaks).
type backBreaks struct {
	base   []uint16
	start  int
	cursor int
	state  int
}

func newBackBreaks(base []uint16, cursor, start, state int) *backBreaks {
	return &backBreaks{base: base, cursor: cursor, start: start, state: state}
}

func (b *backBreaks) copy() *backBreaks { return newBackBreaks(b.base, b.cursor, b.start, b.state) }

// nextBreak is the index of the next grapheme cluster break going
// backwards, or a negative number if cursor was already at start.
func (b *backBreaks) nextBreak() int {
	for b.cursor > b.start {
		breakAt := b.cursor
		b.step()
		if b.state&maskFlags == flagNoBreak {
			continue
		}
		if b.state&maskLookahead != 0 {
			b.lookaheadInNextBreak()
		}
		if b.state&maskBreak != flagNoBreak {
			return breakAt
		}
	}
	b.state = moveBack(b.state, categoryEoT)
	assert(b.state < stateLookaheadMin)
	if b.state&maskBreak != flagNoBreak {
		return b.cursor
	}
	return -1
}

// step reads a single code point before cursor and transitions on it.
func (b *backBreaks) step() {
	assert(b.cursor > b.start)
	b.cursor--
	char := int(b.base[b.cursor])
	surrogate := char ^ 0xDC00
	if surrogate > 0x3FF {
		category := low(char)
		b.state = moveBack(b.state, category)
		return
	}
	// Found tail surrogate, check for prior lead surrogate.
	// The category of an unpaired tail surrogate is Control.
	var category int
	// Dart: cursor >= start && (prevSurrogate = base.codeUnitAt(--cursor) ^ 0xD800) <= 0x3FF.
	matched := false
	if b.cursor >= b.start {
		b.cursor--
		if prevSurrogate := codeUnitAt(b.base, b.cursor) ^ 0xD800; prevSurrogate <= 0x3FF {
			category = high(prevSurrogate, surrogate)
			matched = true
		}
	}
	if !matched {
		category = categoryControl
		b.cursor++
	}
	b.state = moveBack(b.state, category)
}

// codeUnitAt is String.codeUnitAt (a RangeError below 0).
func codeUnitAt(base []uint16, i int) int {
	if i < 0 || i >= len(base) {
		panic(dart.IndexError{Index: i, Length: len(base), Name: "index"})
	}
	return int(base[i])
}

// lookahead steps back using lookahead states; it returns the position
// before the last scanned character.
func (b *backBreaks) lookahead() int {
	assert(b.state >= stateLookaheadMin)
	for b.cursor > b.start {
		cursorBeforeLast := b.cursor
		b.step()
		if b.state < stateLookaheadMin {
			return cursorBeforeLast
		}
	}
	b.state = moveBack(b.state, categorySoT)
	assert(b.state < stateLookaheadMin)
	return b.start
}

// lookaheadInNextBreak performs a lookahead from nextBreak and sets the
// resulting state.
func (b *backBreaks) lookaheadInNextBreak() {
	assert(b.state >= stateLookaheadMin)
	preState := b.state
	preCursor := b.cursor
	breakAt := b.lookahead()
	if preState >= stateLookaheadRegionalEven {
		assert(preState == (stateLookaheadRegionalOdd|flagLookahead) ||
			preState == (stateLookaheadRegionalEven|flagLookahead))
		assert(b.state == (stateRegionalEven|flagNoBreak) || b.state == (stateRegionalOdd|flagBreak))
		// Always reset cursor for regional lookahead.
		b.cursor = preCursor
	} else if b.state&maskFlags == flagLookaheadBreakBoth {
		// Keep cursor at or just before last read.
		b.cursor = breakAt
	}
}

// isGraphemeClusterBoundary is whether there is a grapheme cluster
// boundary before index in text[start:end].
func isGraphemeClusterBoundary(text []uint16, start, end, index int) bool {
	assert(0 <= start)
	assert(start <= index)
	assert(index <= end)
	assert(end <= len(text))
	if start < index && index < end {
		brks := newBreaks(text, index, end, stateCAny)
		cursorBefore := brks.unknownPositionFirstStep(start)
		// If cursor moved, index is in the middle of a surrogate pair.
		if brks.cursor != index {
			return false
		}
		brks.step()
		if brks.state&maskBreak != flagNoBreak {
			return true
		}
		if brks.state&maskLookahead == 0 {
			return false
		}
		assert(brks.state >= stateLookaheadMin)
		back := newBackBreaks(text, cursorBefore, start, brks.state)
		back.lookahead()
		return back.state&maskBreak != flagNoBreak
	}
	return true
}

// previousBreak is the most recent break no later than index in
// text[start:end].
func previousBreak(text []uint16, start, end, index int) int {
	assert(0 <= start)
	assert(start <= index)
	assert(index <= end)
	assert(end <= len(text))
	if start < index && index < end {
		cursorBefore := index
		nextChar := int(text[index])
		nextSurrogate := nextChar ^ 0xD800
		category := categoryControl
		if nextSurrogate > 0x7FF {
			category = low(nextChar)
		} else if nextSurrogate <= 0x3FF {
			indexAfter := index + 1
			if indexAfter < end {
				secondSurrogate := int(text[indexAfter]) ^ 0xDC00
				if secondSurrogate <= 0x3FF {
					category = high(nextSurrogate, secondSurrogate)
				}
			}
		} else {
			prevSurrogate := int(text[index-1]) ^ 0xD800
			nextSurrogate &= 0x3FF
			if prevSurrogate <= 0x3FF {
				category = high(prevSurrogate, nextSurrogate)
				cursorBefore--
			}
		}
		return newBackBreaks(text, cursorBefore, start, moveBack(stateEoTNoBreak, category)).nextBreak()
	}
	return index
}

// nextBreak is the next break no earlier than index in text[start:end].
func nextBreak(text []uint16, start, end, index int) int {
	assert(0 <= start)
	assert(start <= index)
	assert(index <= end)
	assert(end <= len(text))
	// Always break at start or end (GB1).
	if index == start || index == end {
		return index
	}
	brks := newBreaks(text, index, end, stateCAny)
	cursorBefore := brks.unknownPositionFirstStep(start)
	possibleBreak := brks.nextBreak()
	assert(brks.state&maskFlags != 0)
	if brks.state&maskFlags == flagBreak {
		return possibleBreak
	}
	lookbehindState := brks.state
	assert(lookbehindState&maskFlags == flagLookahead)
	assert(lookbehindState&maskState >= stateLookaheadMin)

	back := newBackBreaks(text, cursorBefore, start, lookbehindState)
	back.lookahead()
	if back.state&maskBreak != flagNoBreak {
		return possibleBreak
	}
	// Find the correct forward category to continue with.
	if lookbehindState == stateLookaheadRegionalEven|flagLookahead {
		assert(back.state == stateRegionalEven)
		// Started by RI+RI.
		brks.state = stateRegionalEven
	} else {
		// Was triggered by ZWJ+Pic or InCB={Extend|Linked}+InCB=Consonant.
		assert(lookbehindState == (stateLookaheadZWJPictographic|flagLookahead) ||
			lookbehindState == (stateLookaheadInC|flagLookahead) ||
			lookbehindState == (stateLookaheadInCL|flagLookahead))
		assert(lookbehindState != (stateLookaheadZWJPictographic|flagLookahead) ||
			back.state == statePictographic)
		assert(lookbehindState != (stateLookaheadInC|flagLookahead) &&
			lookbehindState != (stateLookaheadInCL|flagLookahead) ||
			back.state == stateInC)
		brks.state = back.state
	}
	result := brks.nextBreak()
	assert(brks.state&maskFlags == flagBreak)
	return result
}
