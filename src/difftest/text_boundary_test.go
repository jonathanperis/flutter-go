package difftest

// The text boundary and characters difftest
// (flutter/test/text_boundary_test.dart): services' CharacterBoundary,
// ParagraphBoundary, DocumentBoundary and LineBoundary (over the same fixed
// layout) at every offset of the strings Flutter recorded, and
// package:characters' graphemes, CharacterRange.at, skip/take, contains,
// find, moves and replacements, compared exactly as JSON (a Dart throw is
// "error"; Go's panic is recovered as "error").

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/characters"
)

type textBoundaryRef struct {
	Boundaries []struct {
		S    []uint16          `json:"s"`
		Kind string            `json:"kind"`
		At   []json.RawMessage `json:"at"`
	} `json:"boundaries"`
	Characters []map[string]json.RawMessage `json:"characters"`
}

// fakeLayout is the Dart side's FakeLayout.
type fakeLayout struct{ text []uint16 }

func (l fakeLayout) GetLineAtOffset(position ui.TextPosition) services.TextSelection {
	start := 0
	for i, c := range l.text {
		if c == 0x0A {
			if position.Offset <= i {
				return services.NewTextSelection(start, i+1)
			}
			start = i + 1
		}
	}
	return services.NewTextSelection(start, len(l.text))
}

func (fakeLayout) GetWordBoundary(ui.TextPosition) ui.TextRange { panic(dart.UnimplementedError{}) }
func (fakeLayout) GetTextPositionAbove(ui.TextPosition) ui.TextPosition {
	panic(dart.UnimplementedError{})
}
func (fakeLayout) GetTextPositionBelow(ui.TextPosition) ui.TextPosition {
	panic(dart.UnimplementedError{})
}

// guard is the Dart side's guard: f's value, or "error" if it panics.
func guard(f func() any) (v any) {
	defer func() {
		if r := recover(); r != nil {
			v = "error"
		}
	}()
	return f()
}

func ptrOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func units16(s string) []uint16 {
	u := dart.StringCodeUnits(s)
	if u == nil {
		return []uint16{}
	}
	return u
}

// sameJSON compares a Go value with the reference's JSON.
func sameJSON(t *testing.T, ref json.RawMessage, got any) bool {
	t.Helper()
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal(ref, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

func loadTextBoundaryRef(t *testing.T) *textBoundaryRef {
	data, err := os.ReadFile("flutter/build/text_boundary.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref textBoundaryRef
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	return &ref
}

func TestTextBoundariesMatchFlutter(t *testing.T) {
	ref := loadTextBoundaryRef(t)
	if len(ref.Boundaries) < 90 {
		t.Fatalf("only %d cases", len(ref.Boundaries))
	}
	for _, c := range ref.Boundaries {
		s := dart.StringFromCharCodes(c.S)
		var b services.TextBoundary
		switch c.Kind {
		case "character":
			b = services.NewCharacterBoundary(s)
		case "paragraph":
			b = services.NewParagraphBoundary(s)
		case "document":
			b = services.NewDocumentBoundary(s)
		case "line":
			b = services.NewLineBoundary(fakeLayout{c.S})
		}
		for k, want := range c.At {
			i := k - 2
			got := []any{
				guard(func() any { return ptrOrNil(b.GetLeadingTextBoundaryAt(i)) }),
				guard(func() any { return ptrOrNil(b.GetTrailingTextBoundaryAt(i)) }),
				guard(func() any { r := b.GetTextBoundaryAt(i); return []int{r.Start, r.End} }),
			}
			if !sameJSON(t, want, got) {
				t.Errorf("%s %x at %d: flutter %s, go %v", c.Kind, c.S, i, want, got)
			}
		}
	}
}

func charactersCase(s string) map[string]any {
	cs := characters.StringCharacters(s)
	graphemes := slices.Collect(cs.All())
	n := dart.StringLength(s)
	out := map[string]any{}
	gl := []any{}
	for _, g := range graphemes {
		gl = append(gl, units16(g))
	}
	out["graphemes"] = gl
	out["length"] = cs.Length()
	out["first"] = guard(func() any { return units16(cs.First()) })
	out["last"] = guard(func() any { return units16(cs.Last()) })
	out["single"] = guard(func() any { return units16(cs.Single()) })
	at := []any{}
	for i := 0; i <= n; i++ {
		at = append(at, guard(func() any {
			r := characters.NewCharacterRangeAt(s, i)
			return []any{r.StringBeforeLength(), units16(r.Current())}
		}))
	}
	out["at"] = at
	atRange := []any{}
	for i := 0; i <= n; i++ {
		for j := i; j <= n; j++ {
			atRange = append(atRange, guard(func() any {
				r := characters.NewCharacterRangeAt(s, i, j)
				return []any{r.StringBeforeLength(), r.StringAfterLength()}
			}))
		}
	}
	out["atRange"] = atRange
	skipTake := []any{}
	for k := 0; k <= 3; k++ {
		skipTake = append(skipTake, []any{
			guard(func() any { return units16(cs.Skip(k).String()) }),
			guard(func() any { return units16(cs.Take(k).String()) }),
			guard(func() any { return units16(cs.SkipLast(k).String()) }),
			guard(func() any { return units16(cs.TakeLast(k).String()) }),
			guard(func() any { return units16(cs.GetRange(k, k+2).String()) }),
			guard(func() any { return units16(cs.CharacterAt(k).String()) }),
			guard(func() any { return units16(cs.ElementAt(k)) }),
		})
	}
	out["skipTake"] = skipTake
	out["contains"] = guard(func() any {
		l := []any{}
		for _, g := range graphemes {
			l = append(l, cs.Contains(g))
		}
		for i := 1; i < len(graphemes); i++ {
			l = append(l, cs.Contains(dart.Concat(graphemes[i-1], graphemes[i])))
		}
		for _, g := range graphemes {
			l = append(l, cs.ContainsAll(characters.StringCharacters(g)))
		}
		for _, g := range graphemes {
			l = append(l, cs.StartsWith(characters.StringCharacters(g)))
		}
		for _, g := range graphemes {
			l = append(l, cs.EndsWith(characters.StringCharacters(g)))
		}
		for i := 0; i < n; i++ {
			l = append(l, cs.ContainsAll(characters.StringCharacters(dart.Substring(s, i, i+1))))
		}
		for i := 0; i+2 <= n; i++ {
			l = append(l, cs.ContainsAll(characters.StringCharacters(dart.Substring(s, i, i+2))))
		}
		return l
	})
	find := []any{}
	for _, g := range graphemes {
		find = append(find, []any{
			guard(func() any {
				if r := cs.FindFirst(characters.StringCharacters(g)); r != nil {
					return r.StringBeforeLength()
				}
				return nil
			}),
			guard(func() any {
				if r := cs.FindLast(characters.StringCharacters(g)); r != nil {
					return r.StringBeforeLength()
				}
				return nil
			}),
		})
	}
	out["find"] = find
	it := cs.Iterator()
	state := func(moved bool) any { return []any{moved, it.StringBeforeLength(), units16(it.Current())} }
	steps := []func() any{
		func() any { return state(it.MoveNext(1)) },
		func() any { return state(it.MoveNext(2)) },
		func() any { return state(it.MoveNext(3)) },
		func() any { return state(it.MoveBack(2)) },
		func() any { return state(it.ExpandNext(2)) },
		func() any { return state(it.DropFirst()) },
		func() any { return state(it.ExpandBack(3)) },
		func() any { return state(it.DropLast()) },
		func() any { it.ExpandAll(); return []any{it.StringBeforeLength(), units16(it.Current())} },
		func() any { it.CollapseToEnd(); return state(it.MoveBack()) },
		func() any { it.MoveBackAll(); return []any{it.StringBeforeLength(), units16(it.Current())} },
		func() any { it.ExpandAll(); return state(it.DropUntil(characters.StringCharacters("Q"))) },
	}
	moves := []any{}
	for _, step := range steps {
		r := guard(step)
		moves = append(moves, r)
		if r == "error" {
			break
		}
	}
	out["moves"] = moves
	if len(graphemes) == 0 {
		out["replace"] = nil
	} else {
		out["replace"] = guard(func() any {
			split := func(seq func(func(characters.Characters) bool)) []any {
				l := []any{}
				for p := range seq {
					l = append(l, units16(p.String()))
				}
				return l
			}
			return []any{
				units16(cs.ReplaceAll(characters.StringCharacters(graphemes[0]), characters.StringCharacters("X")).String()),
				units16(cs.ReplaceFirst(characters.StringCharacters(graphemes[len(graphemes)-1]), characters.StringCharacters("\u0301")).String()),
				units16(cs.ReplaceAll(characters.StringCharacters(""), characters.StringCharacters("|")).String()),
				split(cs.Split(characters.StringCharacters(graphemes[0]))),
				split(cs.Split(characters.StringCharacters(""), 3)),
				units16(cs.Where(func(g string) bool { return dart.StringLength(g) == 1 }).String()),
				units16(cs.Join("+")),
			}
		})
	}
	return out
}

func TestCharactersMatchFlutter(t *testing.T) {
	ref := loadTextBoundaryRef(t)
	if len(ref.Characters) < 20 {
		t.Fatalf("only %d cases", len(ref.Characters))
	}
	for _, c := range ref.Characters {
		var units []uint16
		if err := json.Unmarshal(c["s"], &units); err != nil {
			t.Fatal(err)
		}
		got := charactersCase(dart.StringFromCharCodes(units))
		for key, want := range c {
			if key == "s" {
				continue
			}
			if !sameJSON(t, want, got[key]) {
				data, _ := json.Marshal(got[key])
				t.Errorf("%x %s:\nflutter %s\ngo      %s", units, key, want, data)
			}
		}
	}
}
