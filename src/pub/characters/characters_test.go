package characters

// Port of package:characters 1.4.1's test/characters_test.dart (with
// test/src/various_tests.dart's zalgo data).
//
// Skipped: nothing. Adapted: `Iterable` results are compared through
// All(); the random walk uses a seeded Go generator (the Dart test seeds
// dart:math's Random from Random()); toUpperCase/toLowerCase compare with
// dart.StringToUpperCase/ToLowerCase (the VM.s mapping), which the port uses (see characters.go).

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

var zalgo = [][]string{{"Z͑ͫ̓ͪ̂ͫ̽͏̴̙̤̞͉͚̯̞̠͍",
	"A̴̵̜̰͔ͫ͗͢", "L̠ͨͧͩ͘",
	"G̴̻͈͍͔̹̑͗̎̅͛́",
	"Ǫ̵̹̻̝̳͂̌̌͘",
	"!͖̬̰̙̗̿̋ͥͥ̂ͣ̐́́͜͞"}}

func gc(s string) Characters { return NewCharacters(s) }

func list(c Characters) []string { return slices.Collect(c.All()) }

func check(t *testing.T, ok bool, format string, args ...any) {
	t.Helper()
	if !ok {
		t.Errorf(format, args...)
	}
}

func eqS(t *testing.T, got, want any) {
	t.Helper()
	if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func panicsWith[E any](t *testing.T, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if _, ok := r.(E); !ok {
			t.Errorf("expected a %T panic, got %v", *new(E), r)
		}
	}()
	f()
}

func TestCharactersOperations(t *testing.T) {
	flag := "\U0001F1E9\U0001F1F0" // Regional Indicators "DK".
	s := "Hi " + flag + "!"
	check(t, dart.StringLength(s) == 8, "length")
	cs := gc(s)
	check(t, cs.Length() == 5, "cs.length")
	eqS(t, list(cs), []string{"H", "i", " ", flag, "!"})
	eqS(t, cs.Skip(2).String(), " "+flag+"!")
	eqS(t, cs.SkipLast(2).String(), "Hi ")
	eqS(t, cs.Take(2).String(), "Hi")
	eqS(t, cs.TakeLast(2).String(), flag+"!")
	eqS(t, cs.GetRange(1, 4).String(), "i "+flag)
	eqS(t, cs.CharacterAt(1).String(), "i")
	eqS(t, cs.CharacterAt(3).String(), flag)

	check(t, !cs.Contains("\U0001F1E9"), "contains RI")
	check(t, cs.Contains(flag), "contains flag")
	check(t, !cs.Contains(flag+"!"), "contains flag!")
	check(t, cs.ContainsAll(gc(flag+"!")), "containsAll")

	notSpace := func(x string) bool { return x != " " }
	eqS(t, cs.TakeWhile(notSpace).String(), "Hi")
	eqS(t, cs.TakeLastWhile(notSpace).String(), flag+"!")
	eqS(t, cs.SkipWhile(notSpace).String(), " "+flag+"!")
	eqS(t, cs.SkipLastWhile(notSpace).String(), "Hi ")

	check(t, !cs.FindFirst(gc("")).MoveBack(), "findFirst('').moveBack")
	eqS(t, cs.FindFirst(gc(flag)).Current(), flag)
	eqS(t, cs.FindLast(gc(flag)).Current(), flag)
	check(t, cs.Iterator().MoveNext(), "iterator.moveNext")
	check(t, !cs.Iterator().MoveBack(), "iterator.moveBack")
	it := cs.Iterator()
	it.MoveNext()
	eqS(t, it.Current(), "H")
	check(t, !cs.IteratorAtEnd().MoveNext(), "atEnd.moveNext")
	check(t, cs.IteratorAtEnd().MoveBack(), "atEnd.moveBack")
	it = cs.IteratorAtEnd()
	it.MoveBack()
	eqS(t, it.Current(), "!")
}

func TestCharactersParts(t *testing.T) {
	testParts(t, gc("a"), gc("b"), gc("c"), gc("d"), gc("e"))
	flag := "\U0001f3f3"
	white := "️"
	zwj := "‍"
	rainbow := "\U0001f308"
	testParts(t, gc(flag+white+zwj+rainbow), gc(flag+white), gc(rainbow), gc(flag+zwj+rainbow), gc("!"))
}

func TestCharacterRangeNew(t *testing.T) {
	r := NewCharacterRange("abc")
	check(t, r.IsEmpty(), "isEmpty")
	check(t, r.MoveNext(), "moveNext")
	eqS(t, r.Current(), "a")
}

func TestCharacterRangeAtSimple(t *testing.T) {
	r := NewCharacterRangeAt("abc", 0)
	check(t, r.IsEmpty(), "1")
	check(t, r.MoveNext(), "2")
	eqS(t, r.Current(), "a")

	r = NewCharacterRangeAt("abc", 1)
	check(t, r.IsEmpty(), "3")
	check(t, r.MoveNext(), "4")
	eqS(t, r.Current(), "b")

	r = NewCharacterRangeAt("abc", 1, 2)
	check(t, !r.IsEmpty(), "5")
	eqS(t, r.Current(), "b")
	check(t, r.MoveNext(), "6")

	r = NewCharacterRangeAt("abc", 0, 3)
	check(t, !r.IsEmpty(), "7")
	eqS(t, r.Current(), "abc")
	check(t, !r.MoveNext(), "8")
}

func TestCharacterRangeAtComplicated(t *testing.T) {
	flag, white, zwj, rainbow := "\U0001f3f3", "️", "‍", "\U0001f308"
	rainbowFlag := flag + white + zwj + rainbow
	s := "-" + rainbowFlag + "-"
	r := NewCharacterRangeAt(s, 1)
	check(t, r.IsEmpty(), "1")
	check(t, r.MoveNext(), "2")
	eqS(t, r.Current(), rainbowFlag)

	r = NewCharacterRangeAt(s, 2)
	check(t, !r.IsEmpty(), "3")
	eqS(t, r.Current(), rainbowFlag)

	r = NewCharacterRangeAt(s, 0, 2)
	check(t, !r.IsEmpty(), "4")
	eqS(t, r.Current(), "-"+rainbowFlag)

	r = NewCharacterRangeAt(s, 2, dart.StringLength("-"+rainbowFlag)-1)
	check(t, !r.IsEmpty(), "5")
	eqS(t, r.Current(), rainbowFlag)
	check(t, r.StringBeforeLength() == 1, "6")

	r = NewCharacterRangeAt(s, 0, dart.StringLength(s))
	check(t, !r.IsEmpty(), "7")
	eqS(t, r.Current(), s)
}

func charsOf(text string) []string {
	var out []string
	for _, r := range text {
		out = append(out, string(r))
	}
	return out
}

func TestGC(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	expectGC(t, random, gc(""), nil)
	for _, text := range []string{"", "A", "123456abcdefab", "à la mode", "rødgrød-æble-ål"} {
		expectGC(t, random, gc(text), charsOf(text))
	}
	expectGC(t, random, gc("a\r\nb"), []string{"a", "\r\n", "b"})
	expectGC(t, random, gc("a\n\rb"), []string{"a", "\n", "\r", "b"})
	expectGC(t, random, gc("à la mode"), []string{"à", " ", "l", "a", " ", "m", "o", "d", "e"})
	expectGC(t, random, gc("æble-ål"), []string{"æ", "b", "l", "e", "-", "å", "l"})
	expectGC(t, random, gc("🇦🇩🇰🇾🇪🇸"), []string{"🇦🇩", "🇰🇾", "🇪🇸"})
	expectGC(t, random, gc("X🇦🇩🇰🇾🇪🇸"), []string{"X", "🇦🇩", "🇰🇾", "🇪🇸"})
	expectGC(t, random, gc("🇩🇰🇾🇪🇸"), []string{"🇩🇰", "🇾🇪", "🇸"})
	expectGC(t, random, gc("X🇩🇰🇾🇪🇸"), []string{"X", "🇩🇰", "🇾🇪", "🇸"})
	expectGC(t, random, gc("읍쌍된밟"), []string{"읍", "쌍", "된", "밟"})
	for _, set := range [][][]string{splitTests, emojis, zalgo} {
		for _, gcs := range set {
			expectGC(t, random, gc(strings.Join(gcs, "")), gcs)
			if t.Failed() {
				t.Fatalf("in %q", gcs)
			}
		}
	}
}

func isEvenLen(s string) bool { return dart.StringLength(s)%2 == 0 }

func filterStrings(xs []string, f func(string) bool) []string {
	var out []string
	for _, x := range xs {
		if f(x) {
			out = append(out, x)
		}
	}
	return out
}

func skipWhileS(xs []string, f func(string) bool) []string {
	i := 0
	for i < len(xs) && f(xs[i]) {
		i++
	}
	return xs[i:]
}

func takeWhileS(xs []string, f func(string) bool) []string {
	i := 0
	for i < len(xs) && f(xs[i]) {
		i++
	}
	return xs[:i]
}

func reversed(xs []string) []string {
	out := slices.Clone(xs)
	slices.Reverse(out)
	return out
}

func joinS(xs []string) string { return dart.Concat(xs...) }

func expectGC(t *testing.T, random *rand.Rand, actual Characters, expected []string) {
	t.Helper()
	text := joinS(expected)
	eqS(t, actual.String(), text)
	eqS(t, list(actual), expected)
	check(t, actual.Length() == len(expected), "length %d", actual.Length())
	if len(expected) > 0 {
		eqS(t, actual.First(), expected[0])
		eqS(t, actual.Last(), expected[len(expected)-1])
	} else {
		panicsWith[dart.StateError](t, func() { actual.First() })
		panicsWith[dart.StateError](t, func() { actual.Last() })
	}
	if len(expected) == 1 {
		eqS(t, actual.Single(), expected[0])
	} else {
		panicsWith[dart.StateError](t, func() { actual.Single() })
	}
	check(t, actual.IsEmpty() == (len(expected) == 0), "isEmpty")
	check(t, actual.IsNotEmpty() == (len(expected) != 0), "isNotEmpty")
	check(t, !actual.Contains(""), "contains ''")
	for _, char := range expected {
		check(t, actual.Contains(char), "contains %q", char)
	}
	for i := 1; i < len(expected); i++ {
		check(t, !actual.Contains(expected[i-1]+expected[i]), "contains pair %d", i)
	}
	skip1 := expected[min(1, len(expected)):]
	take1 := expected[:min(1, len(expected))]
	eqS(t, list(actual.Skip(1)), skip1)
	eqS(t, list(actual.Take(1)), take1)
	eqS(t, actual.Skip(1).String(), joinS(skip1))
	eqS(t, actual.Take(1).String(), joinS(take1))
	take2 := expected[:min(2, len(expected))]
	eqS(t, actual.GetRange(1, 2).String(), joinS(take2[min(1, len(take2)):]))

	if len(expected) > 0 {
		eqS(t, list(actual.SkipLast(1)), expected[:len(expected)-1])
		eqS(t, list(actual.TakeLast(1)), expected[len(expected)-1:])
		eqS(t, actual.SkipLast(1).String(), joinS(expected[:len(expected)-1]))
		eqS(t, actual.TakeLast(1).String(), joinS(expected[len(expected)-1:]))
	}

	eqS(t, list(actual.SkipWhile(isEvenLen)), skipWhileS(expected, isEvenLen))
	eqS(t, list(actual.TakeWhile(isEvenLen)), takeWhileS(expected, isEvenLen))
	eqS(t, actual.SkipWhile(isEvenLen).String(), joinS(skipWhileS(expected, isEvenLen)))
	eqS(t, actual.TakeWhile(isEvenLen).String(), joinS(takeWhileS(expected, isEvenLen)))
	eqS(t, actual.SkipLastWhile(isEvenLen).String(), joinS(reversed(skipWhileS(reversed(expected), isEvenLen))))
	eqS(t, actual.TakeLastWhile(isEvenLen).String(), joinS(reversed(takeWhileS(reversed(expected), isEvenLen))))
	eqS(t, actual.Where(isEvenLen).String(), joinS(filterStrings(expected, isEvenLen)))
	eqS(t, actual.Add(actual).String(), joinS([]string{actual.String(), actual.String()}))

	// Iteration.
	it := actual.Iterator()
	check(t, it.IsEmpty(), "it.isEmpty")
	for i := range expected {
		check(t, it.MoveNext(), "moveNext %d", i)
		eqS(t, it.Current(), expected[i])
		eqS(t, actual.ElementAt(i), expected[i])
		eqS(t, actual.Skip(i).First(), expected[i])
		eqS(t, actual.CharacterAt(i).String(), expected[i])
		eqS(t, actual.GetRange(i, i+1).String(), expected[i])
	}
	check(t, !it.MoveNext(), "moveNext at end")
	for i := len(expected) - 1; i >= 0; i-- {
		check(t, it.MoveBack(), "moveBack %d", i)
		eqS(t, it.Current(), expected[i])
	}
	check(t, !it.MoveBack(), "moveBack at start")
	check(t, it.IsEmpty(), "it.isEmpty at start")

	eqS(t, actual.ToUpperCase().String(), dart.StringToUpperCase(text))
	eqS(t, actual.ToLowerCase().String(), dart.StringToLowerCase(text))
	eqS(t, actual.String(), text)

	check(t, actual.ContainsAll(gc("")), "containsAll ''")
	check(t, actual.ContainsAll(actual), "containsAll self")
	if len(expected) > 0 {
		steps := min(5, len(expected))
		for s := 0; s <= steps; s++ {
			i := len(expected) * s / steps
			check(t, actual.StartsWith(gc(joinS(expected[:i]))), "startsWith %d", i)
			check(t, actual.EndsWith(gc(joinS(expected[i:]))), "endsWith %d", i)
			for tt := s + 1; tt <= steps; tt++ {
				j := len(expected) * tt / steps
				check(t, actual.ContainsAll(gc(joinS(expected[i:j]))), "containsAll %d..%d", i, j)
			}
		}
	}

	// Random walk back and forth.
	it = actual.Iterator()
	pos := -1
	if random.Intn(2) == 0 {
		pos = len(expected)
		it = actual.IteratorAtEnd()
	}
	steps := 5 + random.Intn(len(expected)*2+1)
	lastMove := false
	for {
		back := false
		if pos < 0 {
			check(t, !lastMove && it.IsEmpty(), "walk before start")
		} else if pos >= len(expected) {
			check(t, !lastMove && it.IsEmpty(), "walk after end")
			back = true
		} else {
			check(t, lastMove, "walk lastMove")
			eqS(t, it.Current(), expected[pos])
			back = random.Intn(2) == 0
		}
		steps--
		if steps < 0 {
			break
		}
		if back {
			lastMove = it.MoveBack()
			pos--
		} else {
			lastMove = it.MoveNext()
			pos++
		}
	}
}

func cur(t *testing.T, it CharacterRange, want string) {
	t.Helper()
	eqS(t, it.Current(), want)
}

func eqC(t *testing.T, got, want Characters) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("got %q, want %q", got.String(), want.String())
	}
}

func listC(seq func(func(Characters) bool)) []string {
	var out []string
	for c := range seq {
		out = append(out, c.String())
	}
	return out
}

func listR(seq func(func(CharacterRange) bool)) []string {
	var out []string
	for r := range seq {
		out = append(out, r.Current())
	}
	return out
}

func testParts(t *testing.T, a, b, c, d, e Characters) {
	A, B, C, D, E := a.String(), b.String(), c.String(), d.String(), e.String()
	cs := gc(A + B + C + D + E)
	it := cs.Iterator()
	check(t, it.IsEmpty() && !it.IsNotEmpty(), "start empty")
	cur(t, it, "")

	for _, want := range []string{A, B, C, D, E} {
		check(t, it.MoveNext(), "moveNext")
		check(t, !it.IsEmpty(), "not empty")
		cur(t, it, want)
	}
	check(t, !it.MoveNext(), "moveNext end")
	check(t, it.IsEmpty(), "empty end")
	cur(t, it, "")

	for _, want := range []string{E, D, C, B, A} {
		check(t, it.MoveBack(), "moveBack")
		check(t, !it.IsEmpty(), "not empty")
		cur(t, it, want)
	}
	check(t, !it.MoveBack(), "moveBack start")
	check(t, it.IsEmpty(), "empty start")
	cur(t, it, "")

	check(t, it.MoveTo(c), "moveTo c")
	cur(t, it, C)
	check(t, !it.MoveTo(b), "moveTo b")
	check(t, !it.MoveTo(c), "moveTo c again")
	cur(t, it, C)
	check(t, it.MoveTo(d), "moveTo d")
	cur(t, it, D)

	check(t, it.MoveBackTo(c), "moveBackTo c")
	cur(t, it, C)
	check(t, !it.MoveBackTo(d), "moveBackTo d")
	check(t, !it.MoveBackTo(c), "moveBackTo c again")
	check(t, it.MoveBackTo(a), "moveBackTo a")
	cur(t, it, A)

	check(t, !it.MoveBack(), "moveBack at a")

	check(t, it.MoveNext(2), "moveNext(2)")
	cur(t, it, A+B)
	check(t, !it.MoveNext(4), "moveNext(4)")
	cur(t, it, C+D+E)
	check(t, it.MoveNext(0), "moveNext(0)")
	cur(t, it, "")
	check(t, !it.MoveNext(1), "moveNext(1)")
	cur(t, it, "")

	check(t, it.MoveBack(2), "moveBack(2)")
	cur(t, it, D+E)
	check(t, it.MoveBack(1), "moveBack(1)")
	cur(t, it, C)
	check(t, !it.MoveBack(3), "moveBack(3)")
	cur(t, it, A+B)
	check(t, !it.MoveBack(), "moveBack()")

	it.ExpandAll()
	cur(t, it, A+B+C+D+E)
	check(t, it.CollapseToFirst(b), "collapseToFirst b")
	cur(t, it, B)
	it.ExpandAll()
	cur(t, it, B+C+D+E)
	check(t, !it.CollapseToFirst(a), "collapseToFirst a")
	cur(t, it, B+C+D+E)

	it.ExpandBackAll()
	cur(t, it, A+B+C+D+E)
	check(t, it.CollapseToLast(c), "collapseToLast c")
	cur(t, it, C)

	check(t, it.ExpandTo(e), "expandTo e")
	cur(t, it, C+D+E)
	check(t, !it.ExpandTo(e), "expandTo e again")
	check(t, it.ExpandBackTo(b), "expandBackTo b")
	cur(t, it, B+C+D+E)
	check(t, !it.ExpandBackTo(b), "expandBackTo b again")
	cur(t, it, B+C+D+E)
	check(t, it.CollapseToFirst(c), "collapseToFirst c")
	cur(t, it, C)

	check(t, it.ExpandBackUntil(a), "expandBackUntil a")
	cur(t, it, B+C)
	check(t, it.ExpandBackUntil(a), "expandBackUntil a again")
	cur(t, it, B+C)
	check(t, it.ExpandUntil(e), "expandUntil e")
	cur(t, it, B+C+D)
	check(t, it.ExpandUntil(e), "expandUntil e again")
	cur(t, it, B+C+D)

	check(t, it.DropFirst(), "dropFirst")
	cur(t, it, C+D)
	check(t, it.DropLast(), "dropLast")
	cur(t, it, C)
	it.ExpandBackAll()
	it.ExpandAll()
	cur(t, it, A+B+C+D+E)
	check(t, it.DropTo(b), "dropTo b")
	cur(t, it, C+D+E)
	check(t, it.DropBackTo(d), "dropBackTo d")
	cur(t, it, C)

	it.ExpandBackAll()
	it.ExpandAll()
	cur(t, it, A+B+C+D+E)

	check(t, it.DropUntil(b), "dropUntil b")
	cur(t, it, B+C+D+E)
	check(t, it.DropBackUntil(d), "dropBackUntil d")
	cur(t, it, B+C+D)

	it.DropWhile(func(x string) bool { return x == B })
	cur(t, it, C+D)
	it.ExpandBackAll()
	cur(t, it, A+B+C+D)
	it.DropBackWhile(func(x string) bool { return x != B })
	cur(t, it, A+B)
	it.DropBackWhile(func(string) bool { return false })
	cur(t, it, A+B)

	it.ExpandWhile(func(string) bool { return false })
	cur(t, it, A+B)
	it.ExpandWhile(func(x string) bool { return x != E })
	cur(t, it, A+B+C+D)
	check(t, it.CollapseToFirst(c), "collapseToFirst c 2")
	cur(t, it, C)
	it.ExpandBackWhile(func(string) bool { return false })
	cur(t, it, C)
	it.ExpandBackWhile(func(x string) bool { return x != A })
	cur(t, it, B+C)

	cs2 := cs.ReplaceAll(c, gc(""))
	cs3 := cs.ReplaceFirst(c, gc(""))
	cs4 := cs.FindFirst(c).ReplaceRange(gc("")).Source()
	cse := gc(A + B + D + E)
	eqC(t, cs2, cse)
	eqC(t, cs3, cse)
	eqC(t, cs4, cse)
	cs5 := cs4.ReplaceAll(a, c)
	eqC(t, cs5, gc(C+B+D+E))
	cs6 := cs5.ReplaceAll(gc(""), a)
	eqC(t, cs6, gc(A+C+A+B+A+D+A+E+A))
	cs7 := cs6.ReplaceFirst(b, a)
	eqC(t, cs7, gc(A+C+A+A+A+D+A+E+A))
	cs8 := cs7.ReplaceFirst(e, a)
	eqC(t, cs8, gc(A+C+A+A+A+D+A+A+A))
	cs9 := cs8.ReplaceAll(a.Add(a), b)
	eqC(t, cs9, gc(A+C+B+A+D+B+A))
	it = cs9.Iterator()
	it.MoveTo(b.Add(a))
	cur(t, it, B+A)
	it.ExpandTo(b.Add(a))
	cur(t, it, B+A+D+B+A)
	cs10 := it.ReplaceAll(b.Add(a), e.Add(e))
	eqC(t, cs10.CurrentCharacters(), e.Add(e).Add(d).Add(e).Add(e))
	eqC(t, cs10.Source(), gc(A+C+E+E+D+E+E))
	cs11 := it.ReplaceRange(e)
	eqC(t, cs11.CurrentCharacters(), e)
	eqC(t, cs11.Source(), gc(A+C+E))

	cs12 := gc(A + B + A)
	eqS(t, listC(cs12.Split(b)), []string{A, A})
	eqS(t, listC(cs12.Split(a)), []string{"", B, ""})
	eqS(t, listC(cs12.Split(a, 2)), []string{"", B + A})
	eqS(t, listC(cs12.Split(gc(""))), []string{A, B, A})
	eqS(t, listC(cs12.Split(gc(""), 2)), []string{A, B + A})
	eqS(t, listC(gc("").Split(gc(""))), []string{""})

	cs13 := gc(B + A + B + A + B + A)
	eqS(t, listC(cs13.Split(b)), []string{"", A, A, A})
	eqS(t, listC(cs13.Split(b, 1)), []string{cs13.String()})
	eqS(t, listC(cs13.Split(b, 2)), []string{"", A + B + A + B + A})
	eqS(t, listC(cs13.Split(b, 3)), []string{"", A, A + B + A})
	for _, n := range []int{4, 5, 9999, 0, -1, -9999} {
		eqS(t, listC(cs13.Split(b, n)), []string{"", A, A, A})
	}

	it = cs13.Iterator()
	it.ExpandAll()
	cur(t, it, B+A+B+A+B+A)
	it.DropFirst()
	it.DropLast()
	cur(t, it, A+B+A+B)
	eqS(t, listR(it.Split(a)), []string{"", B, B})
	eqS(t, listR(it.Split(a, 2)), []string{"", B + A + B})
	first := true
	for r := range it.Split(a) {
		if r.IsEmpty() {
			check(t, first, "first range empty")
			first = false
			continue
		}
		cur(t, r, B)
		r.MoveBack()
		cur(t, r, A)
	}
	eqS(t, listR(it.Split(gc(""))), []string{A, B, A, B})
	eqS(t, listR(gc("").Iterator().Split(gc(""))), []string{""})

	check(t, cs.StartsWith(gc("")), "sw ''")
	check(t, cs.StartsWith(a), "sw a")
	check(t, cs.StartsWith(a.Add(b)), "sw ab")
	check(t, cs.StartsWith(gc(A+B+C)), "sw abc")
	check(t, cs.StartsWith(gc(A+B+C+D)), "sw abcd")
	check(t, cs.StartsWith(gc(A+B+C+D+E)), "sw abcde")
	for _, x := range []Characters{b, c, d, e} {
		check(t, !cs.StartsWith(x), "sw %q", x.String())
	}
	check(t, cs.EndsWith(gc("")), "ew ''")
	check(t, cs.EndsWith(e), "ew e")
	check(t, cs.EndsWith(d.Add(e)), "ew de")
	check(t, cs.EndsWith(gc(C+D+E)), "ew cde")
	check(t, cs.EndsWith(gc(B+C+D+E)), "ew bcde")
	check(t, cs.EndsWith(gc(A+B+C+D+E)), "ew abcde")
	for _, x := range []Characters{d, c, b, a} {
		check(t, !cs.EndsWith(x), "ew %q", x.String())
	}

	it = cs.FindFirst(b.Add(c))
	check(t, it.StartsWith(gc("")), "it sw ''")
	check(t, it.StartsWith(b), "it sw b")
	check(t, it.StartsWith(b.Add(c)), "it sw bc")
	check(t, !it.StartsWith(a.Add(b).Add(c)), "it sw abc")
	check(t, !it.StartsWith(b.Add(c).Add(d)), "it sw bcd")
	check(t, !it.StartsWith(a), "it sw a")
	check(t, it.EndsWith(gc("")), "it ew ''")
	check(t, it.EndsWith(c), "it ew c")
	check(t, it.EndsWith(b.Add(c)), "it ew bc")
	check(t, !it.EndsWith(a.Add(b).Add(c)), "it ew abc")
	check(t, !it.EndsWith(b.Add(c).Add(d)), "it ew bcd")
	check(t, !it.EndsWith(d), "it ew d")

	it.CollapseToFirst(c)
	check(t, it.IsPrecededBy(gc("")), "pb ''")
	check(t, it.IsPrecededBy(b), "pb b")
	check(t, it.IsPrecededBy(a.Add(b)), "pb ab")
	check(t, !it.IsPrecededBy(a.Add(b).Add(c)), "pb abc")
	check(t, !it.IsPrecededBy(a), "pb a")
	check(t, it.IsFollowedBy(gc("")), "fb ''")
	check(t, it.IsFollowedBy(d), "fb d")
	check(t, it.IsFollowedBy(d.Add(e)), "fb de")
	check(t, !it.IsFollowedBy(c.Add(d).Add(e)), "fb cde")
	check(t, !it.IsFollowedBy(e), "fb e")
}

// Sample characters from each breaking algorithm category.
var (
	cCR  = gc("\r")
	cLF  = gc("\n")
	cOtr = gc(" ")
	cExt = gc("‌")
	cZWJ = gc("‍")
	cPic = gc("©")
	cReg = gc("\U0001f1e6")
	cInc = gc("क")
	cIne = gc("̀")
	cInl = gc("्")
)

func cat(cs ...Characters) Characters {
	out := CharactersEmpty
	for _, c := range cs {
		out = out.Add(c)
	}
	return out
}

func TestReplaceMethods(t *testing.T) {
	pattern := gc("\t")
	non := gc("")

	c := cat(cOtr, cCR, pattern, cLF, cPic, pattern, cZWJ, cPic, cOtr)
	r := c.ReplaceAll(pattern, non)
	eqC(t, r, cat(cOtr, cCR, cLF, cPic, cZWJ, cPic, cOtr))
	ci := c.Iterator()
	ci.MoveNextAll()
	ri := ci.ReplaceAll(pattern, non)
	eqC(t, ri.CurrentCharacters(), cat(cOtr, cCR, cLF, cPic, cZWJ, cPic, cOtr))
	ci.DropFirst()
	ci.DropLast()
	eqC(t, ci.CurrentCharacters(), cat(cCR, pattern, cLF, cPic, pattern, cZWJ, cPic))
	check(t, ci.CurrentCharacters().Length() == 7, "length 7")
	ri = ci.ReplaceAll(pattern, non)
	eqC(t, ri.CurrentCharacters(), cat(cCR, cLF, cPic, cZWJ, cPic))
	check(t, ri.CurrentCharacters().Length() == 2, "length 2")
	ci.DropFirst()
	ci.DropLast(5)
	eqC(t, ci.CurrentCharacters(), pattern)
	ri = ci.ReplaceAll(pattern, non)
	eqC(t, ri.CurrentCharacters(), cat(cCR, cLF))
	ci.MoveNext(2)
	ci.MoveNext(1)
	eqC(t, ci.CurrentCharacters(), pattern)
	ri = ci.ReplaceAll(pattern, non)
	eqC(t, ri.CurrentCharacters(), cat(cPic, cZWJ, cPic))

	c = cat(cOtr, cPic, cExt, pattern, cPic, cExt, cOtr)
	check(t, c.Length() == 5, "length 5")
	ci = c.Iterator()
	ci.MoveTo(pattern)
	eqC(t, ci.CurrentCharacters(), pattern)
	ri = ci.ReplaceAll(pattern, cZWJ)
	eqC(t, ri.CurrentCharacters(), cat(cPic, cExt, cZWJ, cPic, cExt))

	c = cat(cReg, pattern, cReg, cReg)
	ci = c.Iterator()
	ci.MoveTo(pattern)
	ri = ci.ReplaceRange(non)
	eqC(t, ri.CurrentCharacters(), cat(cReg, cReg))
	check(t, ri.MoveNext(), "moveNext")
	eqC(t, ri.CurrentCharacters(), cReg)

	c = cat(cInc, cIne, pattern, cIne, cZWJ, cInc, cExt)
	check(t, c.Length() == 4, "length 4")
	ci = c.Iterator()
	ci.MoveTo(pattern)
	ri = ci.ReplaceRange(non)
	eqC(t, ri.CurrentCharacters(), cat(cInc, cIne, cIne, cZWJ))
	check(t, ri.Source().Length() == 2, "source length 2")

	ci = c.Iterator()
	ci.MoveTo(pattern)
	ri = ci.ReplaceRange(cInl)
	eqC(t, ri.CurrentCharacters(), cat(cInc, cIne, cInl, cIne, cZWJ, cInc, cExt))
	check(t, ri.Source().Length() == 1, "source length 1")
}
