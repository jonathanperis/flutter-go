package difftest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// The dart:convert difftest (flutter/test/dart_convert_test.dart): the
// bytes JSONUTF8Encode writes for the same values (by label), and what
// UTF8Decode, JSONDecode and JSONUTF8Decode return or panic with for the
// inputs Flutter recorded, compared exactly.

type withToJSON struct{ value any }

func (w withToJSON) ToJson() any { return w.value }

type noToJSON struct{}

// describeDart is the Dart side's describe(): maps sorted by entry text,
// strings as UTF-16 code units, doubles by Dart's toString.
func describeDart(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool:" + strconv.FormatBool(x)
	case int:
		return "int:" + strconv.Itoa(x)
	case float64:
		return "double:" + dart.DoubleToString(x)
	case string:
		units := utf16.Encode([]rune(x))
		parts := make([]string, len(units))
		for i, u := range units {
			parts[i] = strconv.FormatInt(int64(u), 16)
		}
		return "str:" + strings.Join(parts, ",")
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = describeDart(e)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *dart.LinkedMap:
		var parts []string
		for k, e := range x.Entries() {
			parts = append(parts, describeDart(k)+"="+describeDart(e))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return fmt.Sprintf("other:%T", v)
}

func dartErrorText(r any) string {
	switch e := r.(type) {
	case dart.FormatException:
		return "FormatException|" + e.Error()
	case dart.JSONUnsupportedObjectError:
		return "JsonUnsupportedObjectError|" + e.Error()
	case dart.JSONCyclicError:
		return "JsonCyclicError|" + e.Error()
	}
	return fmt.Sprintf("unexpected %T|%v", r, r)
}

func TestDartConvertMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/dart_convert.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct {
		Encode map[string]string
		Decode []struct {
			Kind string
			In   []int
			Ok   *string
			Err  *string
		}
		UTF8Encode []struct {
			In  []uint16
			Out []byte
		} `json:"utf8Encode"`
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}

	// ---- encode
	controls := []rune{}
	for i := range 32 {
		controls = append(controls, rune(i))
	}
	controls = append(controls, 0x7f)
	cyclic := []any{1, nil}
	cyclic[1] = cyclic
	encodeCases := []struct {
		label string
		value any
	}{
		{"null", nil},
		{"true", true},
		{"false", false},
		{"0", 0},
		{"-1", -1},
		{"maxInt", math.MaxInt64},
		{"minInt", math.MinInt64},
		{"2^32", 4294967296},
		{"0.0", 0.0},
		{"-0.0", math.Copysign(0, -1)},
		{"1.0", 1.0},
		{"1.5", 1.5},
		{"0.1+0.2", 0.1 + float64(0.2)},
		{"1e21", 1e21},
		{"1e20", 1e20},
		{"1e-7", 1e-7},
		{"1e-6", 1e-6},
		{"123456789.125", 123456789.125},
		{"maxFinite", math.MaxFloat64},
		{"-maxFinite", -math.MaxFloat64},
		{"minPositive", math.SmallestNonzeroFloat64},
		{"1/3", 1 / float64(3)},
		{"empty string", ""},
		{"hello", "hello"},
		{"controls", string(controls)},
		{"quotes", `"\/'`},
		{"latin1", "é ÿ"},
		{"bmp", "€ \u2028\u2029 \uFEFF <>&"},
		{"astral", "😂𝄞"},
		{"254a+emoji", strings.Repeat("a", 254) + "😂"},
		{"255a+emoji", strings.Repeat("a", 255) + "😂"},
		// Lone surrogates as WTF-8 (Dart strings can hold them).
		{"lone lead", "a\xed\xa0\x80b"},
		{"lone trail", "\xed\xb0\x80"},
		{"lone lead at end", "x\xed\xaf\xbf"},
		{"reversed pair", "\xed\xb8\x82\xed\xa0\xbd"},
		{"empty list", []any{}},
		{"nested lists", []any{[]any{}, []any{1, []any{2, []any{3}}}}},
		{"mixed list", []any{nil, true, 1, -2.5, "x", map[string]any{}}},
		{"empty map", map[string]any{}},
		{"empty int map", map[int]any{}},
		{"map", map[any]any{"a": 1, "b": []any{true, nil}}},
		{"ordered map", []dart.MapEntry[string, any]{{Key: "z", Value: 1}, {Key: "a", Value: 2}, {Key: "m", Value: 3}}},
		{"map key escapes", map[string]string{"a\"b\n": "é"}},
		{"nested map", map[string]any{"k": map[string]any{"x": []any{1.5, map[string]any{"y": nil}}}}},
		{"Uint8List", []byte{0, 1, 255}},
		{"Int32List", []int32{math.MinInt32, math.MaxInt32}},
		{"Int64List", []int64{math.MinInt64, math.MaxInt64}},
		{"Float32List", []float32{0.1, float32(math.Copysign(0, -1)), math.MaxFloat32}},
		{"Float64List", []float64{1.5, math.Copysign(0, -1), 5e-324}},
		{"toJson", withToJSON{map[string]any{"x": 1}}},
		{"toJson in list", []any{withToJSON{"v"}, 2}},
		{"NaN", math.NaN()},
		{"Infinity", math.Inf(1)},
		{"-Infinity", math.Inf(-1)},
		{"NaN in list", []any{1, math.NaN()}},
		{"NaN in map", map[string]any{"a": math.Inf(1)}},
		{"int key", map[any]any{1: "a"}},
		{"null key", map[any]any{nil: "a"}},
		{"mixed keys", []dart.MapEntry[any, any]{{Key: "a", Value: 1}, {Key: 2, Value: "b"}}},
		{"no toJson", noToJSON{}},
		{"toJson NaN", withToJSON{math.NaN()}},
		{"toJson list NaN", withToJSON{[]any{math.NaN()}}},
		{"cyclic list", cyclic},
	}
	if len(encodeCases) != len(ref.Encode) {
		t.Errorf("%d encode cases, Flutter has %d", len(encodeCases), len(ref.Encode))
	}
	for _, c := range encodeCases {
		want, ok := ref.Encode[c.label]
		if !ok {
			t.Errorf("%s: no Flutter case", c.label)
			continue
		}
		got := func() (s string) {
			defer func() {
				if r := recover(); r != nil {
					s = "ERR " + dartErrorText(r)
				}
			}()
			return hex.EncodeToString(dart.JSONUTF8Encode(c.value))
		}()
		switch c.label {
		case "no toJson", "toJson NaN", "toJson list NaN":
			// Dart prints its class name; Go prints the Go type's.
			want = strings.Replace(want, "'NoToJson'", "'noToJSON'", 1)
			want = strings.Replace(want, "'WithToJson'", "'withToJSON'", 1)
		}
		if got != want {
			t.Errorf("encode %s:\n Go:      %s\n Flutter: %s", c.label, got, want)
		}
	}

	// ---- decode
	for _, c := range ref.Decode {
		var input any
		got := func() (s string) {
			defer func() {
				if r := recover(); r != nil {
					s = dartErrorText(r)
				}
			}()
			switch c.Kind {
			case "json8", "utf8":
				b := make([]byte, len(c.In))
				for i, v := range c.In {
					b[i] = byte(v)
				}
				input = b
				if c.Kind == "utf8" {
					return describeDart(dart.UTF8Decode(b))
				}
				return describeDart(dart.JSONUTF8Decode(b))
			default:
				units := make([]uint16, len(c.In))
				for i, v := range c.In {
					units[i] = uint16(v)
				}
				s := string(utf16.Decode(units))
				input = s
				return describeDart(dart.JSONDecode(s))
			}
		}()
		want := ""
		if c.Ok != nil {
			want = *c.Ok
		} else {
			want = *c.Err
		}
		if got != want {
			t.Errorf("%s %q:\n Go:      %q\n Flutter: %q", c.Kind, input, got, want)
		}
	}

	// ---- utf8.encode (code units with lone surrogates written as WTF-8)
	for _, c := range ref.UTF8Encode {
		s := wtf8(c.In)
		if got := dart.UTF8Encode(s); !slices.Equal(got, c.Out) && !(len(got) == 0 && len(c.Out) == 0) {
			t.Errorf("utf8.encode %x: Go %x, Flutter %x", c.In, got, c.Out)
		}
	}
}

// wtf8 encodes UTF-16 code units as WTF-8: pairs as their code point, lone
// surrogates as three-byte sequences.
func wtf8(units []uint16) string {
	var b []byte
	for i := 0; i < len(units); i++ {
		u := rune(units[i])
		if u >= 0xD800 && u < 0xDC00 && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] < 0xE000 {
			b = append(b, string(utf16.DecodeRune(u, rune(units[i+1])))...)
			i++
			continue
		}
		if u >= 0xD800 && u < 0xE000 {
			b = append(b, byte(0xE0|u>>12), byte(0x80|(u>>6)&0x3F), byte(0x80|u&0x3F))
			continue
		}
		b = append(b, string(u)...)
	}
	return string(b)
}
