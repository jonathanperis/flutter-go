package characters

// Port of package:characters 1.4.1's test/breaks_test.dart (the generated
// automatons tested directly on Unicode's GraphemeBreakTest data, with the
// BMP/non-BMP/wrapped variants) and test/src/equiv.dart.
//
// Skipped: nothing. The descriptions use the code units only (the
// category debug names of tool/src/debug_names.dart are not ported).

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

func categoryOf(codePoint int) int {
	if codePoint < 0x10000 {
		return low(codePoint)
	}
	nonBmpOffset := codePoint - 0x10000
	return high(nonBmpOffset>>10, nonBmpOffset&0x3ff)
}

type variant struct {
	parts []ustr
	kind  string
}

func runesOf(u ustr) []int {
	var out []int
	for r := range newRange(u, 0, len(u)).Runes() {
		out = append(out, int(r))
	}
	return out
}

func fromCharCodes(runes []int) ustr {
	var out ustr
	for _, r := range runes {
		out = append(out, dart.StringCodeUnits(dart.StringFromCharCode(r))...)
	}
	return out
}

// testVariants is breaks_test.dart's testVariants.
func testVariants(parts []string) []variant {
	var flipped, upper, lower [][]int
	const hasNonBmp, hasBmp = 1, 2
	changes := 0
	var base []ustr
	for _, part := range parts {
		u := dart.StringCodeUnits(part)
		base = append(base, u)
		var f, up, lo []int
		for _, r := range runesOf(u) {
			runeLC, runeFC, runeUC := r, r, r
			category := categoryOf(r)
			if r < 0x10000 {
				if other := upperChars[category]; other >= 0 {
					changes |= hasNonBmp
					runeUC, runeFC = other, other
				}
			} else {
				if other := lowerChars[category]; other >= 0 {
					changes |= hasBmp
					runeLC, runeFC = other, other
				}
			}
			f = append(f, runeFC)
			up = append(up, runeUC)
			lo = append(lo, runeLC)
		}
		flipped = append(flipped, f)
		upper = append(upper, up)
		lower = append(lower, lo)
	}
	conv := func(l [][]int) []ustr {
		var out []ustr
		for _, x := range l {
			out = append(out, fromCharCodes(x))
		}
		return out
	}
	variants := []variant{{base, ""}}
	if changes == hasNonBmp|hasBmp {
		variants = append(variants, variant{conv(flipped), "(Flip)"})
	}
	if changes&hasNonBmp != 0 {
		variants = append(variants, variant{conv(upper), "(non-BMP)"})
	}
	if changes&hasBmp != 0 {
		variants = append(variants, variant{conv(lower), "(BMP)"})
	}
	wrapped := append(append([]ustr{{0}}, base...), ustr{0})
	return append(variants, variant{wrapped, "(Wrapped)"})
}

func join(parts []ustr) ustr {
	var out ustr
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func describe(parts []ustr) string { return fmt.Sprintf("%x", parts) }

func forEachVariant(t *testing.T, f func(t *testing.T, input ustr, parts []ustr)) {
	for _, expected := range splitTests {
		for _, v := range testVariants(expected) {
			f(t, join(v.parts), v.parts)
			if t.Failed() {
				t.Fatalf("in %s%s", describe(v.parts), v.kind)
			}
		}
	}
}

func TestForwardAutomaton(t *testing.T) {
	forEachVariant(t, func(t *testing.T, input ustr, want []ustr) {
		brks := newBreaks(input, 0, len(input), stateSoTNoBreak)
		var parts []ustr
		start := 0
		for start < len(input) {
			next := brks.nextBreak()
			if next <= start {
				t.Fatalf("break %d not after %d", next, start)
			}
			parts = append(parts, input[start:next])
			start = next
		}
		if describe(parts) != describe(want) {
			t.Errorf("got %s", describe(parts))
		}
	})
}

func TestBackwardAutomaton(t *testing.T) {
	forEachVariant(t, func(t *testing.T, input ustr, want []ustr) {
		brks := newBackBreaks(input, len(input), 0, stateEoTNoBreak)
		var parts []ustr
		start := len(input)
		for start > 0 {
			next := brks.nextBreak()
			if next >= start {
				t.Fatalf("break %d not before %d", next, start)
			}
			parts = append(parts, input[next:start])
			start = next
		}
		slices.Reverse(parts)
		if describe(parts) != describe(want) {
			t.Errorf("got %s", describe(parts))
		}
	})
}

func TestNextBreak(t *testing.T) {
	forEachVariant(t, func(t *testing.T, input ustr, parts []ustr) {
		partCursor, nextExpectedBreak := 0, 0
		for i := 0; i <= len(input); i++ {
			if got := nextBreak(input, 0, len(input), i); got != nextExpectedBreak {
				t.Errorf("at %d: got %d, want %d", i, got, nextExpectedBreak)
			}
			if i == nextExpectedBreak && i < len(input) {
				nextExpectedBreak += len(parts[partCursor])
				partCursor++
			}
		}
	})
}

func TestPreviousBreak(t *testing.T) {
	forEachVariant(t, func(t *testing.T, input ustr, parts []ustr) {
		partCursor, next, expectedBreak := 0, 0, 0
		for i := 0; i <= len(input); i++ {
			if i == next {
				expectedBreak = next
				if i < len(input) {
					next += len(parts[partCursor])
					partCursor++
				}
			}
			if got := previousBreak(input, 0, len(input), i); got != expectedBreak {
				t.Errorf("at %d: got %d, want %d", i, got, expectedBreak)
			}
		}
	})
}

func TestIsGraphemeClusterBreak(t *testing.T) {
	forEachVariant(t, func(t *testing.T, input ustr, parts []ustr) {
		partCursor, next := 0, 0
		for i := 0; i <= len(input); i++ {
			if got := isGraphemeClusterBoundary(input, 0, len(input), i); got != (i == next) {
				t.Errorf("at %d: got %v", i, got)
			}
			if i == next && i < len(input) {
				next += len(parts[partCursor])
				partCursor++
			}
		}
	})
}

// --- Minimal automaton (equiv.dart's Equivalence, as the tests use it).

type equivalence struct {
	equivalences []int
	class        map[int]int
	order        []int // insertion order of class's keys (Dart's map order)
}

func newEquivalenceDistinct(elements []int) *equivalence {
	e := &equivalence{class: map[int]int{}}
	for _, el := range elements {
		e.add(el)
	}
	return e
}

func newEquivalence(classes [][]int) *equivalence {
	e := &equivalence{class: map[int]int{}}
	for _, c := range classes {
		id := len(e.equivalences)
		e.equivalences = append(e.equivalences, id)
		for _, el := range c {
			e.setClass(el, id)
		}
	}
	return e
}

func (e *equivalence) setClass(el, id int) {
	if _, ok := e.class[el]; !ok {
		e.order = append(e.order, el)
	}
	e.class[el] = id
}

func (e *equivalence) add(el int) int {
	id := len(e.equivalences)
	e.equivalences = append(e.equivalences, id)
	e.setClass(el, id)
	return id
}

func (e *equivalence) canonicalizeID(id int) int {
	for {
		next := e.equivalences[id]
		if next == id {
			return id
		}
		e.equivalences[id] = next
		id = next
	}
}

func (e *equivalence) classOf(el int) (int, bool) {
	c, ok := e.class[el]
	if !ok {
		return 0, false
	}
	x := e.equivalences[c]
	if x == c {
		return x, true
	}
	e.equivalences[c] = e.canonicalizeID(x)
	return e.equivalences[c], true
}

func (e *equivalence) eq(v1, v2 int) bool {
	c1, ok1 := e.classOf(v1)
	c2, ok2 := e.classOf(v2)
	return ok1 && ok2 && c1 == c2
}

func (e *equivalence) equate(v1, v2 int) {
	c1, ok1 := e.classOf(v1)
	c2, ok2 := e.classOf(v2)
	switch {
	case !ok1:
		if ok2 {
			e.setClass(v1, c2)
		} else {
			e.setClass(v1, e.add(v2))
		}
	case !ok2:
		e.setClass(v2, c1)
	default:
		newID := c1
		if c1 != c2 {
			if c1 < c2 {
				e.equivalences[c2] = c1
			} else {
				e.equivalences[c1] = c2
				newID = c2
			}
		}
		if c1 != newID {
			e.setClass(v1, newID)
		}
		if c2 != newID {
			e.setClass(v2, newID)
		}
	}
}

// classes groups the elements by canonical class (the number of classes
// is what the tests check).
func (e *equivalence) classes() [][]int {
	byID := map[int][]int{}
	var ids []int
	for _, el := range e.order {
		c, _ := e.classOf(el)
		if _, ok := byID[c]; !ok {
			ids = append(ids, c)
		}
		byID[c] = append(byID[c], el)
	}
	var out [][]int
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out
}

func TestStatesReachable(t *testing.T) {
	states := map[int]bool{}
	for _, s := range []int{stateBreak, stateCR, stateOther, statePrepend, stateL, stateV, stateT,
		statePictographic, statePictographicZWJ, stateRegionalSingle, stateInC, stateInCL,
		stateSoT, stateSoTNoBreak, stateCAny, stateCZWJ, stateCExZ, stateCIE, stateCIEZ, stateCIL,
		stateCILZ, stateCZIE, stateCZIL, stateCReg, stateCExt} {
		states[s] = true
	}
	entry := []int{stateSoTNoBreak, stateSoT, stateCAny}
	unreachable := map[int]bool{}
	for s := range states {
		unreachable[s] = true
	}
	for _, s := range entry {
		delete(unreachable, s)
	}
	work := append([]int{}, entry...)
	var next []int
	for (len(work) > 0 || len(next) > 0) && len(unreachable) > 0 {
		if len(work) == 0 {
			work, next = next, nil
		}
		state := work[len(work)-1]
		work = work[:len(work)-1]
		for c := 0; c < categoryCount; c++ {
			newState := move(state, c) & maskState
			if newState&maskFlags == flagLookahead {
				continue
			}
			if !states[newState] {
				t.Fatalf("(%d,%d): Unexpected output state %d", state, c, newState)
			}
			if unreachable[newState] {
				delete(unreachable, newState)
				next = append(next, newState)
			}
		}
	}
	if len(unreachable) > 0 {
		t.Errorf("unreachable: %v", unreachable)
	}
}

func TestStatesBackwardReachable(t *testing.T) {
	states := map[int]bool{}
	for _, s := range []int{stateBreak, stateLF, stateOther, stateExtend, stateL, stateV, stateT,
		statePictographic, stateRegionalOdd, stateRegionalSingle, stateInC, stateRegionalEven,
		stateEoT, stateEoTNoBreak, stateLookaheadZWJPictographic, stateLookaheadInC,
		stateLookaheadInCL, stateLookaheadRegionalEven, stateLookaheadRegionalOdd} {
		states[s] = true
	}
	entry := []int{stateEoTNoBreak, stateEoT}
	unreachable := map[int]bool{}
	for s := range states {
		unreachable[s] = true
	}
	for _, s := range entry {
		delete(unreachable, s)
	}
	work := append([]int{}, entry...)
	var next []int
	for (len(work) > 0 || len(next) > 0) && len(unreachable) > 0 {
		if len(work) == 0 {
			work, next = next, nil
		}
		state := work[len(work)-1]
		work = work[:len(work)-1]
		for c := 0; c < categoryCount; c++ {
			newState := moveBack(state, c) & maskState
			if !states[newState] {
				t.Fatalf("Unexpected output state %d", newState)
			}
			if unreachable[newState] {
				delete(unreachable, newState)
				next = append(next, newState)
			}
		}
	}
	if len(unreachable) > 0 {
		t.Errorf("unreachable: %v", unreachable)
	}
}

func TestStatesDistinguishable(t *testing.T) {
	testDistinguishable(t, stateLimit, move, false)
}

func TestBackwardStatesDistinguishable(t *testing.T) {
	testDistinguishable(t, backStateLimit, moveBack, true)
}

func testDistinguishable(t *testing.T, limit int, mv func(int, int) int, back bool) {
	var states []int
	for i := 0; i < limit; i += automatonRowLength {
		states = append(states, i)
	}
	eqClasses := [][]int{states}
	eq := newEquivalence(eqClasses)
	stateCount := limit / automatonRowLength
	for r := 0; r <= stateCount; r++ {
		nextEq := newEquivalenceDistinct(states)
		for _, eqClass := range eqClasses {
			for i := 0; i < len(eqClass)-1; i++ {
				state1 := eqClass[i]
			nextPair:
				for j := i + 1; j < len(eqClass); j++ {
					state2 := eqClass[j]
					for c := 0; c < categoryCount; c++ {
						n1, n2 := mv(state1, c), mv(state2, c)
						if (n1^n2)&maskFlags != 0 ||
							(back && (n1 >= stateLookaheadMin || n2 >= stateLookaheadMin)) ||
							!eq.eq(n1&maskState, n2&maskState) {
							continue nextPair // Keep distinguishable.
						}
					}
					nextEq.equate(state1, state2)
				}
			}
		}
		prev := eqClasses
		eqClasses = nextEq.classes()
		eq = nextEq
		if len(prev) == len(eqClasses) || len(prev) == len(states) {
			break
		}
	}
	for _, c := range eqClasses {
		if len(c) != 1 {
			t.Errorf("indistinguishable states %v", c)
		}
	}
}
