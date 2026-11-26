package services

// Own tests of text_editing.go's TextSelection (text_editing.dart has no
// Flutter test file of its own) and of MapLookup.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestMapLookup(t *testing.T) {
	m := []dart.MapEntry[any, any]{{Key: "base", Value: 1}}
	if v, ok := MapLookup(m, "base"); !ok || v != 1 {
		t.Fatal("lookup")
	}
	if _, ok := MapLookup(map[string]int{"x": 1}, "y"); ok {
		t.Fatal("lookup missing")
	}
	if v, ok := MapLookup(map[any]any{"x": 2}, "x"); !ok || v != 2 {
		t.Fatal("lookup any map")
	}
	if v, ok := MapLookup([]dart.MapEntry[string, any]{{Key: "s", Value: 3}}, "s"); !ok || v != 3 {
		t.Fatal("lookup string entries")
	}
	if _, ok := MapLookup(5, "x"); ok {
		t.Fatal("not a map")
	}
	if _, ok := MapLookup(map[string]int{"x": 1}, 1); ok {
		t.Fatal("key of another type")
	}
	if _, ok := MapLookup([]dart.MapEntry[any, any]{{Key: []int{1}, Value: 1}}, []int{1}); ok {
		t.Fatal("uncomparable keys are unequal")
	}
}

func TestTextSelection(t *testing.T) {
	up := ui.TextAffinityUpstream
	s := NewTextSelection(4, 1, TextSelectionOptions{IsDirectional: true})
	if s.Start != 1 || s.End != 4 || s.Base() != ui.NewTextPosition(4, ui.TextAffinityUpstream) ||
		s.Extent() != ui.NewTextPosition(1) || s.String() != "TextSelection(baseOffset: 4, extentOffset: 1, isDirectional: true)" {
		t.Fatalf("%v %v %v", s, s.Base(), s.Extent())
	}
	c := NewTextSelectionCollapsed(3, up)
	if c.String() != "TextSelection.collapsed(offset: 3, affinity: TextAffinity.upstream, isDirectional: false)" ||
		c.Equal(NewTextSelectionCollapsed(3)) || !c.Equal(NewTextSelectionFromPosition(ui.NewTextPosition(3, up))) {
		t.Fatal(c)
	}
	if !NewTextSelection(-1, -1).Equal(NewTextSelection(-1, -5+4)) || NewTextSelection(-1, -1).String() != "TextSelection.invalid" {
		t.Fatal("invalid")
	}
	// Affinity only matters when collapsed.
	if !NewTextSelection(1, 2).Equal(NewTextSelection(1, 2, TextSelectionOptions{Affinity: &up})) {
		t.Fatal("affinity")
	}
	if got := NewTextSelection(2, 5).ExpandTo(ui.NewTextPosition(0)); got.BaseOffset != 0 || got.ExtentOffset != 5 {
		t.Fatal(got)
	}
	if got := NewTextSelection(5, 2).ExpandTo(ui.NewTextPosition(8)); got.BaseOffset != 8 || got.ExtentOffset != 2 {
		t.Fatal(got)
	}
	if got := NewTextSelection(2, 5).ExpandTo(ui.NewTextPosition(0, up), true); got.BaseOffset != 5 || got.ExtentOffset != 0 || got.Affinity != up {
		t.Fatal(got)
	}
	if got := NewTextSelection(2, 5).ExpandTo(ui.NewTextPosition(9), true); got.BaseOffset != 2 || got.ExtentOffset != 9 {
		t.Fatal(got)
	}
	if got := NewTextSelection(2, 5).ExpandTo(ui.NewTextPosition(3)); !got.Equal(NewTextSelection(2, 5)) {
		t.Fatal(got)
	}
	if got := NewTextSelection(2, 5).ExtendTo(ui.NewTextPosition(7)); got.BaseOffset != 2 || got.ExtentOffset != 7 {
		t.Fatal(got)
	}
	if got := NewTextSelection(2, 5).ExtendTo(ui.NewTextPosition(5, up)); got.ExtentOffset != 5 {
		t.Fatal(got)
	}
	if got := ui.NewTextRange(1, 3).TextInside("a😀b"); got != "😀" {
		t.Fatal(got)
	}
}
