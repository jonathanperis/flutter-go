package services_test

// Own tests of TextEditingValue's range assertions (Dart asserts that the
// selection and composing ranges are inside the text).

import (
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

func expectAssert(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("no assertion, want %q", want)
		}
		if msg := r.(error).Error(); !strings.Contains(msg, want) {
			t.Fatalf("assertion %q, want %q", msg, want)
		}
	}()
	f()
}

func TestTextEditingValueRangeAssertions(t *testing.T) {
	expectAssert(t, "Range end 5 is out of text of length 3", func() {
		services.NewTextEditingValueFromJSON(linkedOf(map[string]any{"text": "abc", "composingBase": 1, "composingExtent": 5}))
	})
	expectAssert(t, "Range start 4 is out of text of length 3", func() {
		services.NewTextEditingValueFromJSON(linkedOf(map[string]any{"text": "abc", "selectionBase": 5, "selectionExtent": 4}))
	})
	// (-1, -1) is "no range" and passes.
	v := services.NewTextEditingValueFromJSON(linkedOf(map[string]any{"text": "abc"}))
	if v.Selection.IsValid() || v.Composing.IsValid() {
		t.Fatal(v)
	}
}
