package services

// Port of the "Standard message codec" group of
// packages/flutter/test/services/message_codecs_test.dart (Flutter 3.50.1),
// 4/4 tests; the binary, string, JSON and method codec groups and the
// toString test cover codecs not ported yet. Plus own tests of
// TextSelection (text_editing.dart has no Flutter test file of its own) and
// MapLookup.

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func checkEncoding(t *testing.T, codec MessageCodec[any], message any, expected []byte) {
	t.Helper()
	if got := codec.EncodeMessage(message); !bytes.Equal(got, expected) {
		t.Fatalf("encoding %v:\n got  %v\n want %v", message, got[:min(len(got), 16)], expected[:min(len(expected), 16)])
	}
}

// deepEqual is flutter_test's equals for decoded messages: NaN equals NaN,
// -0.0 differs from 0.0 only by sign (Dart's == says equal; the test's
// matcher compares the lists element by element with ==).
func deepEqual(a, b any) bool {
	switch x := a.(type) {
	case float64:
		y, ok := b.(float64)
		return ok && (x == y || math.IsNaN(x) && math.IsNaN(y))
	case []float64:
		y, ok := b.([]float64)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !deepEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case []float32:
		y, ok := b.([]float32)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !(x[i] == y[i] || math.IsNaN(float64(x[i])) && math.IsNaN(float64(y[i]))) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !deepEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case []dart.MapEntry[any, any]:
		y, ok := b.([]dart.MapEntry[any, any])
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !deepEqual(x[i].Key, y[i].Key) || !deepEqual(x[i].Value, y[i].Value) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

func checkEncodeDecode(t *testing.T, codec MessageCodec[any], message any) {
	t.Helper()
	encoded := codec.EncodeMessage(message)
	decoded := codec.DecodeMessage(encoded)
	if message == nil {
		if encoded != nil || decoded != nil {
			t.Fatal("null message")
		}
		return
	}
	if !deepEqual(message, decoded) {
		t.Fatalf("decoded %#v, want %#v", decoded, message)
	}
	if !bytes.Equal(codec.EncodeMessage(decoded), encoded) {
		t.Fatalf("re-encoding %v differs", message)
	}
}

func filled(n int, v byte, prefix ...byte) []byte {
	return append(prefix, bytes.Repeat([]byte{v}, n)...)
}

func TestStandardCodecSizesAtBoundaryCases(t *testing.T) {
	var standard MessageCodec[any] = StandardMessageCodec{}
	checkEncoding(t, standard, make([]byte, 253), filled(253, 0, 8, 253))
	checkEncoding(t, standard, make([]byte, 254), filled(254, 0, 8, 254, 254, 0))
	checkEncoding(t, standard, make([]byte, 0xffff), filled(0xffff, 0, 8, 254, 0xff, 0xff))
	checkEncoding(t, standard, make([]byte, 0xffff+1), filled(0xffff+1, 0, 8, 255, 0, 0, 1, 0))
}

func TestStandardCodecSimpleMessages(t *testing.T) {
	var standard MessageCodec[any] = StandardMessageCodec{}
	for _, m := range []any{nil, true, false, 7, -7, 98742923489, -98742923489, 3.14, math.Inf(1), math.NaN(),
		"", "hello", "special chars >\u263A\U0001F602<"} {
		checkEncodeDecode(t, standard, m)
	}
}

func TestStandardCodecCompositeMessage(t *testing.T) {
	var standard MessageCodec[any] = StandardMessageCodec{}
	specials := []float64{math.Inf(-1), -math.MaxFloat64, -math.SmallestNonzeroFloat64, math.Copysign(0, -1), 0,
		math.SmallestNonzeroFloat64, math.MaxFloat64, math.Inf(1), math.NaN()}
	specials32 := make([]float32, len(specials))
	for i, v := range specials {
		specials32[i] = float32(v)
	}
	message := []any{
		nil, true, false, -707, -7000000007, -3.14, "", "hello",
		[]byte{0xBA, 0x5E, 0xBA, 0x11},
		[]int32{-0x7fffffff - 1, 0, 0x7fffffff},
		nil, // ensures the offset of the following list is unaligned.
		nil, // ensures the offset of the following list is unaligned.
		specials,
		specials32,
		[]any{"nested", []any{}},
		[]dart.MapEntry[any, any]{{Key: "a", Value: "nested"}, {Key: nil, Value: []dart.MapEntry[any, any]{}}},
		"world",
	}
	checkEncodeDecode(t, standard, message)
}

func TestStandardCodecInt32Boundaries(t *testing.T) {
	checkEncoding(t, StandardMessageCodec{}, -0x80000000, []byte{3, 0, 0, 0, 0x80})
	checkEncoding(t, StandardMessageCodec{}, 0x7fffffff, []byte{3, 0xff, 0xff, 0xff, 0x7f})
	checkEncoding(t, StandardMessageCodec{}, 0x80000000, []byte{4, 0, 0, 0, 0x80, 0, 0, 0, 0})
}

func TestStandardCodecAlignsDoubles(t *testing.T) {
	checkEncoding(t, StandardMessageCodec{}, 1.0, []byte{6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xf0, 0x3f})
}

func TestStandardCodecCorruptedAndUnsupported(t *testing.T) {
	codec := StandardMessageCodec{}
	for _, bad := range [][]byte{{}, {99}, {3, 0, 0, 0, 0, 1}} {
		func() {
			defer func() {
				if _, ok := recover().(FormatException); !ok {
					t.Fatalf("%v: no FormatException", bad)
				}
			}()
			codec.DecodeMessage(bad)
		}()
	}
	func() {
		defer func() {
			if _, ok := recover().(dart.ArgumentError); !ok {
				t.Fatal("no ArgumentError")
			}
		}()
		codec.EncodeMessage(struct{}{})
	}()
	// Go maps are written sorted by key; pointers are their values.
	one := 1
	if got := codec.EncodeMessage(map[string]any{"b": &one, "a": (*int)(nil)}); !bytes.Equal(got,
		codec.EncodeMessage([]dart.MapEntry[string, any]{{Key: "a", Value: nil}, {Key: "b", Value: 1}})) {
		t.Fatal("map encoding")
	}
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
	// A repeated key keeps its first position and the last value.
	dup := codec.DecodeMessage([]byte{13, 2, 7, 1, 'k', 3, 1, 0, 0, 0, 7, 1, 'k', 3, 2, 0, 0, 0})
	if got := dup.([]dart.MapEntry[any, any]); len(got) != 1 || got[0].Value != 2 {
		t.Fatalf("%v", dup)
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
