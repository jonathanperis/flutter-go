package dart

import (
	"math"
	"strings"
	"testing"
)

// Expected values are the Dart VM's (checked by difftest/dart_convert_test).

func catchError(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				s = e.Error()
			} else {
				s = "panic"
			}
		}
	}()
	f()
	return ""
}

func TestUTF8Decode(t *testing.T) {
	for _, c := range []struct {
		in   string
		want string
	}{
		{"", ""},
		{"abc", "abc"},
		{"\xEF\xBB\xBFa", "a"},
		{"\xEF\xBB\xBF\xEF\xBB\xBF", "\uFEFF"},
		{"a\xEF\xBB\xBF", "a\uFEFF"},
		{"\xC2\x80\xDF\xBF", "\u0080߿"},
		{"\xF4\x8F\xBF\xBF", "\U0010FFFF"},
		{"\xED\x9F\xBF", "퟿"},
		{"\xEF\xBF\xBF", "￿"},
	} {
		if got := UTF8Decode([]byte(c.in)); got != c.want {
			t.Errorf("UTF8Decode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	for _, c := range []struct{ in, want string }{
		{"\xEF\xBB", "FormatException: Unfinished UTF-8 octet sequence (at offset 2)"},
		{"\xC3", "FormatException: Unfinished UTF-8 octet sequence (at offset 1)"},
		{"abc\xC3", "FormatException: Unfinished UTF-8 octet sequence (at offset 4)"},
		{"\x80", "FormatException: Unexpected extension byte (at offset 0)"},
		{"a\x80b", "FormatException: Unexpected extension byte (at offset 1)"},
		{"\xC3\x41", "FormatException: Missing extension byte (at offset 1)"},
		{"\xC3\xC3\xA9", "FormatException: Missing extension byte (at offset 1)"},
		{"\xED\xA0\x80", "FormatException: Encoded surrogate (at offset 1)"},
		{"\xC0\x80", "FormatException: Invalid UTF-8 byte (at offset 0)"},
		{"\xE0\x80\x80", "FormatException: Overlong encoding (at offset 1)"},
		{"\xF0\x80\x80\x80", "FormatException: Overlong encoding (at offset 1)"},
		{"\xF4\x90\x80\x80", "FormatException: Out of unicode range (at offset 1)"},
		{"\xF5\x80", "FormatException: Invalid UTF-8 byte (at offset 0)"},
		{"\xFF", "FormatException: Invalid UTF-8 byte (at offset 0)"},
		{"\xE2\x82", "FormatException: Unfinished UTF-8 octet sequence (at offset 2)"},
		{"ab\xC3\xA9\x80", "FormatException: Unexpected extension byte (at offset 4)"},
	} {
		if got := catchError(func() { UTF8Decode([]byte(c.in)) }); got != c.want {
			t.Errorf("UTF8Decode(%q): %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUTF8Encode(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"é€😂", "é€😂"},
		{"a\xed\xa0\x80b", "a�b"},         // a lone surrogate (WTF-8)
		{"\xed\xa0\xbd\xed\xb8\x82", "😂"}, // WTF-8 halves of a pair: one code point, as in Dart
		{"\xff", "�"},
		{"\xed\xa0", "��"},
	} {
		if got := string(UTF8Encode(c.in)); got != c.want {
			t.Errorf("UTF8Encode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestJSONUTF8Encode(t *testing.T) {
	for _, c := range []struct {
		v    any
		want string
	}{
		{"\x00\x1f\x7f\b\t\n\f\r\"\\/", `"\u0000\u001f` + "\x7f" + `\b\t\n\f\r\"\\/"`},
		{"\x0b\x1a", `"\u000b\u001a"`},
		{"a\xed\xa0\x80b", `"a\ud800b"`},
		{"\xed\xb0\x80", `"\udc00"`},
		{"x\xed\xaf\xbf", `"x\udbff"`},
		{"\xed\xb8\x82\xed\xa0\xbd", `"\ude02\ud83d"`},
		{map[any]any{"b": 1, "a": 2}, `{"a":2,"b":1}`},
		{map[int]any{}, `{}`},
		{[]MapEntry[any, any]{}, `{}`},
		{[]MapEntry[any, any]{{"z", 1}, {"a", nil}}, `{"z":1,"a":null}`},
		{[]byte{1, 255}, `[1,255]`},
		{[]float32{0.1}, `[0.10000000149011612]`},
		{9223372036854775807.0, `9223372036854776000.0`}, {(*int)(nil), `null`},
		{new(int), `0`},
		{uint8(7), `7`},
	} {
		if got := JSONEncode(c.v); got != c.want {
			t.Errorf("JSONEncode(%#v) = %s, want %s", c.v, got, c.want)
		}
	}
	nan := math.NaN()
	cyclic := []any{1, nil}
	cyclic[1] = cyclic
	for _, c := range []struct {
		v    any
		want string
	}{
		{nan, "Converting object to an encodable object failed: NaN"},
		{[]any{1, math.Inf(-1)}, "Converting object to an encodable object failed: -Infinity"},
		{map[any]any{1: "a"}, "Converting object to an encodable object failed: _Map len:1"},
		{[]MapEntry[any, any]{{"a", 1}, {nil, 2}}, "Converting object to an encodable object failed: _Map len:2"},
		{map[string]any{"a": []any{map[any]any{true: 1, false: 2}}}, "Converting object to an encodable object failed: _Map len:2"},
		{struct{}{}, "Converting object to an encodable object failed: Instance of 'struct {}'"},
		{cyclic, "Cyclic error in JSON stringify"},
	} {
		if got := catchError(func() { JSONEncode(c.v) }); got != c.want {
			t.Errorf("JSONEncode(%v): %q, want %q", c.v, got, c.want)
		}
	}
	if (JSONUnsupportedObjectError{Object: 1}).Error() != "Converting object did not return an encodable object: 1" {
		t.Error("no cause")
	}
	// An object's toJson whose result fails: the object is reported.
	if got := catchError(func() { JSONEncode([]any{testToJSON{[]any{nan}}}) }); !strings.HasSuffix(got, "Instance of 'testToJSON'") {
		t.Errorf("toJson failure: %q", got)
	}
	if got := JSONEncode(testToJSON{[]any{"x"}}); got != `["x"]` {
		t.Errorf("toJson: %s", got)
	}
}

type testToJSON struct{ v any }

func (t testToJSON) ToJson() any { return t.v }

func TestJSONDecode(t *testing.T) {
	negZero := math.Copysign(0, -1)
	for _, c := range []struct {
		in   string
		want any
	}{
		{"-0", 0},
		{"[-0]", []any{0}},
		{"-0.0", negZero},
		{"[-0.0]", []any{negZero}},
		{"[-0e5]", []any{negZero}},
		{"[-1e999]", []any{math.Inf(-1)}},
		{"[1e999]", []any{math.Inf(1)}},
		{"[-1e-999]", []any{negZero}},
		{"[0e999]", []any{0.0}},
		{"9223372036854775807", 9223372036854775807},
		{"[9223372036854775807]", []any{9223372036854775807}},
		{"[-9223372036854775808]", []any{-9223372036854775808}},
		{"[9223372036854775808]", []any{9223372036854775808.0}},
		{"[-9223372036854775809]", []any{-9223372036854775809.0}},
		{"[922337203685477580]", []any{922337203685477580}},
		{"[922337203685477589]", []any{922337203685477589}},
		{"[-922337203685477589]", []any{-922337203685477589}},
		{"[1234567890123456789]", []any{1234567890123456789}},
		{"[12345678901234567890]", []any{12345678901234567890.0}},
		{"[0.1, 1e22, 1e23, 1.5e-3, 25e-1, 1E+2]", []any{0.1, 1e22, 1e23, 1.5e-3, 2.5, 100.0}},
		{"[9007199254740993.0, 123.456e-20, 1e-22]", []any{9007199254740993.0, 123.456e-20, 1e-22}},
		{"[1e-23]", []any{1e-23}},
		{"[9007199254740993e-1, 900719925474099.3]", []any{900719925474099.3, 900719925474099.3}},
		{" \t\r\n[ 1 , 2 ] \n", []any{1, 2}},
		{`"é😂\/\b\f\n\r\t\"\\"`, "é😂/\b\f\n\r\t\"\\"},
		{`"ꯍxꯍ"`, "ꯍxꯍ"},
		{`"\ud800"`, "�"},
		{`{"a":1,"b":2,"a":3}`, map[string]any{"a": 3, "b": 2}},
		{`[true,false,null,{}]`, []any{true, false, nil, map[string]any{}}},
		{`{"a":{"b":[1,{"c":null}]}}`, map[string]any{"a": map[string]any{"b": []any{1, map[string]any{"c": nil}}}}},
	} {
		got := JSONDecode(c.in)
		if ToString(got) != ToString(c.want) || describeKind(got) != describeKind(c.want) {
			t.Errorf("JSONDecode(%q) = %#v, want %#v", c.in, got, c.want)
		}
		got8 := JSONUTF8Decode([]byte(c.in))
		if ToString(got8) != ToString(c.want) || describeKind(got8) != describeKind(c.want) {
			t.Errorf("JSONUTF8Decode(%q) = %#v, want %#v", c.in, got8, c.want)
		}
	}
	if !math.Signbit(JSONDecode("[-0.0]").([]any)[0].(float64)) || !math.Signbit(JSONDecode("-0e3").(float64)) ||
		!math.Signbit(JSONDecode("[-1e-999]").([]any)[0].(float64)) {
		t.Error("-0.0 lost its sign")
	}
	if got := JSONUTF8Decode([]byte("\xEF\xBB\xBF[1]")); ToString(got) != "[1]" {
		t.Errorf("BOM: %v", got)
	}
	if got := JSONUTF8Decode([]byte("\"\xEF\xBB\xBFa\"")); got != "\uFEFFa" {
		t.Errorf("BOM in string: %q", got)
	}
	if got := JSONUTF8Decode([]byte("\"\xC3\xA9\\u00e9\xE2\x82\xAC\"")); got != "éé€" {
		t.Errorf("escapes between UTF-8: %q", got)
	}
}

// describeKind tells ints from doubles in nested values.
func describeKind(v any) string {
	switch x := v.(type) {
	case int:
		return "i"
	case float64:
		if math.Signbit(x) {
			return "-d"
		}
		return "d"
	case []any:
		s := "["
		for _, e := range x {
			s += describeKind(e)
		}
		return s + "]"
	case map[string]any:
		s := "{"
		for _, k := range []string{"a", "b", "c"} {
			s += describeKind(x[k])
		}
		return s + "}"
	}
	return "o"
}

func TestJSONDecodeErrors(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", "FormatException: Unexpected end of input (at offset 0)"},
		{" ", "FormatException: Unexpected end of input (at offset 1)"},
		{"01", "FormatException: Unexpected character (at offset 1)"},
		{"-01", "FormatException: Unexpected character (at offset 2)"},
		{"[00]", "FormatException: Unexpected character (at offset 2)"},
		{"1.", "FormatException: Unterminated number literal (at offset 2)"},
		{"1e+", "FormatException: Unterminated number literal (at offset 3)"},
		{"-", "FormatException: Unterminated number literal (at offset 1)"},
		{"[1.", "FormatException: Unterminated number literal (at offset 3)"},
		{"[1", "FormatException: Unexpected end of input (at offset 2)"},
		{"[1.a]", "FormatException: Unexpected character (at offset 3)"},
		{"[1e]", "FormatException: Missing expected digit (at offset 3)"},
		{"[1e-]", "FormatException: Missing expected digit (at offset 4)"},
		{"[-]", "FormatException: Missing expected digit (at offset 2)"},
		{"--1", "FormatException: Missing expected digit (at offset 1)"},
		{"+1", "FormatException: Unexpected character (at offset 0)"},
		{".5", "FormatException: Unexpected character (at offset 0)"},
		{`"\u12"`, "FormatException: Invalid unicode escape (at offset 2)"},
		{`"\u12G4"`, "FormatException: Invalid unicode escape (at offset 2)"},
		{`"\u12`, "FormatException: Unterminated string (at offset 5)"},
		{`"\x"`, "FormatException: Unrecognized string escape (at offset 3)"},
		{"\"\\\x01\"", "FormatException: Control character in string (at offset 3)"},
		{`"\`, "FormatException: Unterminated string (at offset 2)"},
		{`"abc`, "FormatException: Unterminated string (at offset 4)"},
		{"\"a\nb\"", "FormatException: Control character in string (at offset 2)"},
		{"\"a\\nb\x01\"", "FormatException: Unexpected character (at offset 5)"},
		{"tru", "FormatException: Unexpected end of input (at offset 3)"},
		{"trux", "FormatException: Unexpected character (at offset 0)"},
		{"truee", "FormatException: Unexpected character (at offset 4)"},
		{"nulx", "FormatException: Unexpected character (at offset 0)"},
		{"falsy", "FormatException: Unexpected character (at offset 0)"},
		{"[1,]", "FormatException: Unexpected character (at offset 3)"},
		{"[,1]", "FormatException: Unexpected character (at offset 1)"},
		{"[1 2]", "FormatException: Unexpected character (at offset 3)"},
		{"]", "FormatException: Unexpected character (at offset 0)"},
		{"[]]", "FormatException: Unexpected character (at offset 2)"},
		{"{1:2}", "FormatException: Unexpected character (at offset 1)"},
		{`{"a" 1}`, "FormatException: Unexpected character (at offset 5)"},
		{`{"a":1,}`, "FormatException: Unexpected character (at offset 7)"},
		{`{"a",1}`, "FormatException: Unexpected character (at offset 4)"},
		{`{"a":`, "FormatException: Unexpected end of input (at offset 5)"},
		{"[nul]", "FormatException: Unexpected character (at offset 1)"},
		{"fx", "FormatException: Unexpected character (at offset 0)"},
		{"[nx", "FormatException: Unexpected character (at offset 1)"},
		{"[tr", "FormatException: Unexpected end of input (at offset 3)"},
		{"\f1", "FormatException: Unexpected character (at offset 0)"},
		{"\xEF\xBB", "FormatException: Unexpected end of input (at offset 2)"},
		{"\xEF\xBB\xBF", "FormatException: Unexpected end of input (at offset 3)"},
		{"\xEF\x31", "FormatException: Unexpected character (at offset 1)"},
		{"\xEF\xBB\xBF\xEF\xBB\xBF1", "FormatException: Unexpected character (at offset 3)"},
		{"\"\xC3\"", "FormatException: Unfinished UTF-8 octet sequence"},
		{"\"\xC3\\n\"", "FormatException: Unfinished UTF-8 octet sequence"},
		{"\"\xC3\\n\xA9\"", "FormatException: Unfinished UTF-8 octet sequence"},
		{"\"\x80\"", "FormatException: Unexpected extension byte (at offset 1)"},
		{"\"a\\nb\xC3\xA9\x80\"", "FormatException: Unexpected extension byte (at offset 7)"},
		{"\"\xED\xA0\x80\"", "FormatException: Encoded surrogate (at offset 2)"},
		{"\"\xC3\x01\"", "FormatException: Control character in string (at offset 2)"},
		{"[\xC3\xA9]", "FormatException: Unexpected character (at offset 1)"},
	} {
		if got := catchError(func() { JSONUTF8Decode([]byte(c.in)) }); got != c.want {
			t.Errorf("JSONUTF8Decode(%q): %q, want %q", c.in, got, c.want)
		}
	}
	for _, c := range []struct{ in, want string }{
		{"01", "FormatException: Unexpected character (at character 2)\n01\n ^\n"},
		{"\"é😂\"x", "FormatException: Unexpected character (at character 6)\n\"é😂\"x\n     ^\n"},
		{"Ā", "FormatException: Unexpected character (at character 1)\nĀ\n^\n"},
		{" 1", "FormatException: Unexpected character (at character 1)\n 1\n^\n"},
		{"[\n1,\n x]", "FormatException: Unexpected character (at line 3, character 2)\n x]\n ^\n"},
	} {
		if got := catchError(func() { JSONDecode(c.in) }); got != c.want {
			t.Errorf("JSONDecode(%q): %q, want %q", c.in, got, c.want)
		}
	}
}
