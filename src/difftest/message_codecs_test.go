package difftest

// Go twin of flutter/test/message_codecs_test.dart: the same values through
// services' codecs, compared with Flutter's bytes, decoded values and error
// texts, case by case.

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// dartNaN is Dart's double.nan (0.0 / 0.0), whose bits differ from
// math.NaN()'s.
var dartNaN = math.Float64frombits(0xfff8000000000000)

// Object stands for an instance of Dart's Object (unencodable).
type Object struct{}

func codecHex(b []byte) string {
	if b == nil {
		return "null"
	}
	return hex.EncodeToString(b)
}

func doubleBits(v float64) string { return fmt.Sprintf("%016x", math.Float64bits(v)) }

// codecDesc is the Dart side's desc.
func codecDesc(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool:" + strconv.FormatBool(x)
	case int:
		return "int:" + strconv.Itoa(x)
	case float64:
		return "double:" + doubleBits(x)
	case string:
		return "String:" + dart.JSONEncode(x)
	case []byte:
		return "Uint8List:" + joinInts(x)
	case []int32:
		return "Int32List:" + joinInts(x)
	case []int64:
		return "Int64List:" + joinInts(x)
	case []float32:
		var s []string
		for _, f := range x {
			s = append(s, doubleBits(float64(f)))
		}
		return "Float32List:" + strings.Join(s, ",")
	case []float64:
		var s []string
		for _, f := range x {
			s = append(s, doubleBits(f))
		}
		return "Float64List:" + strings.Join(s, ",")
	case []any:
		var s []string
		for _, e := range x {
			s = append(s, codecDesc(e))
		}
		return "List:[" + strings.Join(s, ",") + "]"
	case *dart.LinkedMap:
		var s []string
		for k, e := range x.Entries() {
			s = append(s, codecDesc(k)+"="+codecDesc(e))
		}
		return "Map:{" + strings.Join(s, ",") + "}"
	}
	return fmt.Sprintf("other:%v", v)
}

func joinInts[T int32 | int64 | byte](x []T) string {
	var s []string
	for _, v := range x {
		s = append(s, strconv.FormatInt(int64(v), 10))
	}
	return strings.Join(s, ",")
}

func codecNullable(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func codecErrorText(r any) string {
	switch e := r.(type) {
	case dart.FormatException:
		return "FormatException:" + e.Error()
	case *services.PlatformException:
		return "PlatformException:" + e.Code + "|" + codecNullable(e.Message) + "|" + codecDesc(e.Details) + "|" +
			codecNullable(e.Stacktrace) + "|" + e.Error()
	case *services.MissingPluginException:
		return "MissingPluginException:" + e.Error()
	case dart.ArgumentError:
		return "ArgumentError:" + e.Error()
	case dart.JSONUnsupportedObjectError:
		return "JsonUnsupportedObjectError:" + e.Error()
	case dart.TypeError:
		return "TypeError:" + e.Error()
	case runtime.Error:
		if strings.Contains(e.Error(), "out of range") {
			return "RangeError"
		}
	}
	return fmt.Sprintf("other:%T %v", r, r)
}

type codecCase struct{ Name, Out string }

type codecRecorder struct{ cases []codecCase }

func (c *codecRecorder) record(name string, body func() string) {
	var out string
	func() {
		defer func() {
			if r := recover(); r != nil {
				out = "throws " + codecErrorText(r)
			}
		}()
		out = body()
	}()
	c.cases = append(c.cases, codecCase{name, out})
}

func (c *codecRecorder) roundTrip(name string, codec services.MessageCodec[any], value any) {
	c.record(name, func() string {
		encoded := codec.EncodeMessage(value)
		var decoded string
		func() {
			defer func() {
				if r := recover(); r != nil {
					decoded = "throws " + codecErrorText(r)
				}
			}()
			decoded = codecDesc(codec.DecodeMessage(encoded))
		}()
		return codecHex(encoded) + " -> " + decoded
	})
}

func (c *codecRecorder) decode(name string, codec services.MessageCodec[any], b []byte) {
	c.record(name, func() string { return codecDesc(codec.DecodeMessage(b)) })
}

func decodeCall(codec services.MethodCodec, b []byte) string {
	call := codec.DecodeMethodCall(b)
	return call.Method + " " + codecDesc(call.Arguments) + " | " + call.String()
}

func entriesOf(kv ...any) []dart.MapEntry[any, any] {
	var out []dart.MapEntry[any, any]
	for i := 0; i < len(kv); i += 2 {
		out = append(out, dart.MapEntry[any, any]{Key: kv[i], Value: kv[i+1]})
	}
	return out
}

func standardValues() []any {
	f32 := func(v ...float64) []float32 {
		out := make([]float32, len(v))
		for i, x := range v {
			out[i] = float32(x)
		}
		return out
	}
	negZero := math.Copysign(0, -1)
	values := []any{
		nil, true, false, 0, 1, -1, 127, 255,
		-0x7fffffff - 1, 0x7fffffff, -0x7fffffff - 2, 0x7fffffff + 1,
		math.MinInt64, math.MaxInt64, 9007199254740993,
		0.0, negZero, 1.0, 0.1, -3.14, dartNaN, math.Inf(1), math.Inf(-1),
		math.MaxFloat64, math.SmallestNonzeroFloat64, 2.2250738585072014e-308, 1e300,
		"", "a", "hello world",
		strings.Repeat("a", 253), strings.Repeat("b", 254), strings.Repeat("c", 300),
		strings.Repeat("d", 65535), strings.Repeat("e", 65536),
		"é", "日本語", "\U0001F602", "a\U0001F602b\x00c\n\t\"\\", "\uFEFFbom", "x\uFEFF",
		"mixed ascii then é and 😂 end", "\u0080\u07FF\u0800\uFFFF\U00010000\U0010FFFF",
		[]byte{}, []byte{1, 2, 3, 255}, make([]byte, 254),
		[]int32{-0x7fffffff - 1, 0, 0x7fffffff},
		[]int64{math.MinInt64, 0, math.MaxInt64},
		f32(0.1, negZero, math.Inf(1), dartNaN, 3.4028234663852886e38, 1e-45),
		[]float64{0.1, negZero, math.Inf(1), dartNaN, math.MaxFloat64, math.SmallestNonzeroFloat64},
		[]int32{}, []float64{},
		[]any{}, []any{nil, 1, "two", 3.0, []any{}}, []any{[]any{[]any{1}}},
	}
	for pad := range 8 {
		l := make([]any, pad)
		l = append(l, []int32{7}, nil, []int64{8}, true, []float32{1.5}, false, []float64{2.5}, 3.5)
		values = append(values, l)
	}
	values = append(values,
		map[any]any{},
		entriesOf("b", 1, "a", 2),
		entriesOf(1, "one", 2, "two", nil, "null", true, "true", 2.5, "double"),
		entriesOf("nested", entriesOf("list", []any{1, 2}, "map", map[any]any{})),
		entriesOf("z", []byte{1}, "y", []float64{1.0}),
	)
	list := make([]any, 300)
	for i := range list {
		list[i] = i
	}
	var kv []any
	for i := range 260 {
		kv = append(kv, "k"+strconv.Itoa(i), i)
	}
	return append(values, list, entriesOf(kv...))
}

func TestMessageCodecsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/message_codecs.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Cases []codecCase }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}

	c := &codecRecorder{}
	standard := services.NewStandardMessageCodec()
	standardMethod := services.StandardMethodCodec{}
	jsonCodec := services.JSONMessageCodec{}
	jsonMethod := services.JSONMethodCodec{}
	stringCodec := services.StringCodec{}

	// ---- StandardMessageCodec round trips.
	for i, v := range standardValues() {
		c.roundTrip(fmt.Sprintf("standard %d", i), standard, v)
	}

	// ---- StandardMessageCodec decoding hand-made bytes.
	raw := []struct {
		name string
		b    []byte
	}{
		{"null message", nil},
		{"empty", []byte{}},
		{"trailing", []byte{0, 0}},
		{"unknown type", []byte{15}},
		{"type 127", []byte{127}},
		{"type 128", []byte{128}},
		{"truncated int32", []byte{3, 1}},
		{"truncated int64", []byte{4, 1, 2, 3, 4}},
		{"truncated float64", []byte{6, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"truncated string", []byte{7, 5, 97}},
		{"missing size", []byte{7}},
		{"truncated size 254", []byte{7, 254, 1}},
		{"truncated size 255", []byte{7, 255, 1, 0}},
		{"missing list element", []byte{12, 2, 0}},
		{"missing map value", []byte{13, 1, 0}},
		{"bad utf8", []byte{7, 2, 0xc3, 0x28}},
		{"encoded surrogate", []byte{7, 3, 0xed, 0xa0, 0x80}},
		{"overlong", []byte{7, 2, 0xc0, 0x80}},
		{"unfinished utf8", []byte{7, 1, 0xe6}},
		{"bom string", []byte{7, 4, 0xef, 0xbb, 0xbf, 0x41}},
		{"double bom string", []byte{7, 6, 0xef, 0xbb, 0xbf, 0xef, 0xbb, 0xbf}},
		{"large int", []byte{5, 4, 0x31, 0x66, 0x66, 0x66}},
		{"size 254 form small", []byte{8, 254, 1, 0, 9}},
		{"size 255 form small", []byte{8, 255, 2, 0, 0, 0, 9, 8}},
		{"unaligned double", []byte{6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xf0, 0x3f}},
		{"int32 list no padding", []byte{9, 1, 0, 0, 1, 0, 0, 0}},
	}
	for _, r := range raw {
		c.decode("standard decode "+r.name, standard, r.b)
	}

	// ---- Maps with keys Dart's == merges (or doesn't).
	negZero := math.Copysign(0, -1)
	mapEntries := []struct {
		name string
		kv   []any
	}{
		{"int then double", []any{1, "a", 1.0, "b"}},
		{"double then int", []any{1.0, "a", 1, "b"}},
		{"zeros", []any{negZero, "a", 0.0, "b", 0, "c"}},
		{"int zero then neg zero", []any{0, "a", negZero, "b"}},
		{"pos zero then neg zero", []any{0.0, "a", negZero, "b"}},
		{"duplicate strings", []any{"a", 1, "a", 2, "b", 3}},
		{"nan keys", []any{dartNaN, 1, dartNaN, 2}},
		{"int64 equal double", []any{9007199254740992, "a", 9007199254740992.0, "b"}},
		{"int64 unequal double", []any{9007199254740993, "a", 9007199254740992.0, "b"}},
		{"double then int64 unequal", []any{9007199254740992.0, "a", 9007199254740993, "b"}},
		{"min int64 and double", []any{math.MinInt64, "a", -9223372036854775808.0, "b"}},
		{"huge double", []any{9223372036854775808.0, "a", math.MaxInt64, "b"}},
		{"int max then huge double", []any{math.MaxInt64, "a", 9223372036854775808.0, "b"}},
		{"bools and null", []any{true, 1, nil, 2, false, 3, true, 4, nil, 5}},
		{"string and int", []any{"1", 1, 1, 2}},
	}
	for _, m := range mapEntries {
		c.record("standard map "+m.name, func() string {
			buffer := foundation.NewWriteBuffer()
			buffer.PutUint8(13)
			standard.WriteSize(buffer, len(m.kv)/2)
			for _, v := range m.kv {
				standard.WriteValue(buffer, v)
			}
			b := buffer.Done()
			return codecHex(b) + " -> " + codecDesc(standard.DecodeMessage(b))
		})
	}

	// ---- Unencodable values.
	c.record("standard unencodable", func() string { return codecHex(standard.EncodeMessage(Object{})) })
	c.record("standard unencodable nested", func() string { return codecHex(standard.EncodeMessage([]any{1, Object{}})) })

	// ---- StandardMethodCodec.
	args := []any{nil, "hello", []any{1, 2.5, "x"}, entriesOf("a", 1, "b", []any{true, false}), []float64{1.0}}
	errorEnvelope := func(codec services.MethodCodec, i int) []byte {
		var o services.EncodeErrorEnvelopeOptions
		if i%2 == 0 {
			m := "message " + strconv.Itoa(i)
			o.Message = &m
		}
		o.Details = args[i]
		return codec.EncodeErrorEnvelope("code"+strconv.Itoa(i), o)
	}
	for i := range args {
		c.record(fmt.Sprintf("method call %d", i), func() string {
			encoded := standardMethod.EncodeMethodCall(services.NewMethodCall("method"+strconv.Itoa(i), args[i]))
			return codecHex(encoded) + " -> " + decodeCall(standardMethod, encoded)
		})
		c.record(fmt.Sprintf("success envelope %d", i), func() string {
			encoded := standardMethod.EncodeSuccessEnvelope(args[i])
			return codecHex(encoded) + " -> " + codecDesc(standardMethod.DecodeEnvelope(encoded))
		})
		c.record(fmt.Sprintf("error envelope %d", i), func() string {
			encoded := errorEnvelope(standardMethod, i)
			return codecHex(encoded) + " -> " + codecDesc(standardMethod.DecodeEnvelope(encoded))
		})
	}
	for _, r := range []struct {
		name string
		b    []byte
	}{
		{"method name not string", []byte{3, 1, 0, 0, 0, 0}},
		{"method call trailing", []byte{7, 1, 109, 0, 0}},
		{"method call missing args", []byte{7, 1, 109}},
		{"method call empty", []byte{}},
	} {
		c.record("standard "+r.name, func() string { return decodeCall(standardMethod, r.b) })
	}
	c.record("standard method call null", func() string { return decodeCall(standardMethod, nil) })
	for _, r := range []struct {
		name string
		b    []byte
	}{
		{"empty", []byte{}},
		{"success", []byte{0, 7, 1, 120}},
		{"success trailing", []byte{0, 0, 0}},
		{"success missing value", []byte{0}},
		{"error with stacktrace", []byte{1, 7, 1, 99, 7, 1, 109, 0, 7, 2, 115, 116}},
		{"error null stacktrace", []byte{1, 7, 1, 99, 0, 0, 0}},
		{"error int stacktrace", []byte{1, 7, 1, 99, 0, 0, 3, 1, 0, 0, 0}},
		{"error code not string", []byte{1, 3, 1, 0, 0, 0, 0, 0}},
		{"error message not string", []byte{1, 7, 1, 99, 3, 1, 0, 0, 0, 0}},
		{"error trailing", []byte{1, 7, 1, 99, 0, 0, 0, 0}},
		{"error missing details", []byte{1, 7, 1, 99, 0}},
		{"error flag 2", []byte{2, 7, 1, 99, 0, 0}},
		{"error flag 255", []byte{255, 7, 1, 99, 0, 0}},
	} {
		c.record("standard envelope "+r.name, func() string { return codecDesc(standardMethod.DecodeEnvelope(r.b)) })
	}

	// ---- JSONMessageCodec.
	jsonValues := []any{
		nil, true, false, 0, -7, 98742923489, math.MaxInt64, math.MinInt64,
		0.0, negZero, 3.14, 1e21, 1e-7, 123456789.0, math.MaxFloat64, math.SmallestNonzeroFloat64,
		"", "hello", "special chars >\u263A\U0001F602<",
		"quote \" backslash \\ slash / controls \x00\x01\x1f\b\f\n\r\t \x7f\u2028",
		strings.Repeat("a", 254) + "\U0001F602",
		[]any{},
		[]any{nil, true, 1, 2.5, "x", []any{}, map[string]any{}},
		[]dart.MapEntry[string, any]{{Key: "b", Value: 1}, {Key: "a", Value: []any{1, 2}},
			{Key: "c", Value: map[string]any{"d": nil}}},
		[]byte{1, 2}, []int32{-1}, []float64{1.5, 2.0},
		dartNaN, math.Inf(1), []any{1, math.Inf(-1)},
		map[any]any{1: "x"},
	}
	for i, v := range jsonValues {
		c.roundTrip(fmt.Sprintf("json %d", i), jsonCodec, v)
	}
	str := func(s string) []byte { return []byte(s) }
	for _, r := range []struct {
		name string
		b    []byte
	}{
		{"null message", nil},
		{"object", str(`{"a": 1, "b": [true, null, 2.5e3, -0, -0.0, 1E2]}`)},
		{"whitespace", str(" \t\r\n[ 1 , 2 ] \n")},
		{"big int", str("9223372036854775808")},
		{"neg big int", str("-9223372036854775809")},
		{"int max", str("9223372036854775807")},
		{"exponent", str("1e400")},
		{"escapes", str(`"\u00e9\ud83d\ude02\/\b\f\n\r\t\"\\"`)},
		{"empty", str("")},
		{"garbage", str("[1,]")},
		{"trailing", str("[1] x")},
		{"unterminated", str(`"abc`)},
		{"control char", str("\"a\x01\"")},
		{"leading zero", str("01")},
		{"duplicate keys", str(`{"a": 1, "a": 2}`)},
	} {
		c.decode("json decode "+r.name, jsonCodec, r.b)
	}
	c.decode("json decode bad utf8", jsonCodec, []byte{0x22, 0xc3, 0x28, 0x22})
	c.decode("json decode bom", jsonCodec, []byte{0xef, 0xbb, 0xbf, 0x31})

	// ---- JSONMethodCodec.
	for i := range len(args) - 1 {
		c.record(fmt.Sprintf("json method call %d", i), func() string {
			encoded := jsonMethod.EncodeMethodCall(services.NewMethodCall("method"+strconv.Itoa(i), args[i]))
			return codecHex(encoded) + " -> " + decodeCall(jsonMethod, encoded)
		})
		c.record(fmt.Sprintf("json success envelope %d", i), func() string {
			encoded := jsonMethod.EncodeSuccessEnvelope(args[i])
			return codecHex(encoded) + " -> " + codecDesc(jsonMethod.DecodeEnvelope(encoded))
		})
		c.record(fmt.Sprintf("json error envelope %d", i), func() string {
			encoded := errorEnvelope(jsonMethod, i)
			return codecHex(encoded) + " -> " + codecDesc(jsonMethod.DecodeEnvelope(encoded))
		})
	}
	for _, r := range []struct {
		name string
		b    []byte
	}{
		{"null", nil},
		{"list", str(`[1, "x"]`)},
		{"string", str(`"m"`)},
		{"no method", str(`{"args": 1}`)},
		{"method not string", str(`{"method": 1}`)},
		{"method only", str(`{"method": "m"}`)},
		{"method null args", str(`{"method": "m", "args": null}`)},
		{"extra keys", str(`{"method": "m", "args": 1, "x": 2}`)},
	} {
		c.record("json method call decode "+r.name, func() string { return decodeCall(jsonMethod, r.b) })
	}
	for _, r := range []struct{ name, s string }{
		{"map", `{"a": 1}`},
		{"empty list", `[]`},
		{"one", `[{"x": [1]}]`},
		{"two", `[1, 2]`},
		{"three", `["c", "m", {"d": 1}]`},
		{"three null message", `["c", null, null]`},
		{"three bad code", `[1, "m", null]`},
		{"three bad message", `["c", 1, null]`},
		{"four", `["c", "m", 3, "trace"]`},
		{"four null", `["c", null, 3, null]`},
		{"four bad trace", `["c", "m", 3, 4]`},
		{"four bad message", `["c", 1, 3, "s"]`},
		{"five", `["c", "m", 3, "s", 5]`},
		{"null", `null`},
		{"string", `"s"`},
	} {
		c.record("json envelope "+r.name, func() string { return codecDesc(jsonMethod.DecodeEnvelope(str(r.s))) })
	}

	// ---- StringCodec and BinaryCodec.
	for _, s := range []*string{nil, ptr(""), ptr("hello"), ptr("日本😂"), ptr("\uFEFFx")} {
		name := "null"
		if s != nil {
			name = dart.JSONEncode(*s)
		}
		c.record("string "+name, func() string {
			encoded := stringCodec.EncodeMessage(s)
			decoded := stringCodec.DecodeMessage(encoded)
			if decoded == nil {
				return codecHex(encoded) + " -> null"
			}
			return codecHex(encoded) + " -> " + codecDesc(*decoded)
		})
	}
	c.record("string decode bad utf8", func() string { return codecDesc(*stringCodec.DecodeMessage([]byte{0xff})) })
	c.record("string decode bom", func() string { return codecDesc(*stringCodec.DecodeMessage([]byte{0xef, 0xbb, 0xbf})) })
	c.record("binary", func() string { return codecHex(services.BinaryCodec{}.EncodeMessage([]byte{1, 2})) })

	// ---- toString.
	c.record("toString", func() string {
		return strings.Join([]string{
			services.NewMethodCall("m").String(),
			services.NewMethodCall("m", []any{1, "a", nil, 2.0, []any{}}).String(),
			services.NewMethodCall("m", map[string]any{"k": 1.5}).String(),
			services.NewPlatformException("c").Error(),
			services.NewPlatformException("c", services.PlatformExceptionOptions{Message: ptr("m"), Details: []any{1, 2},
				Stacktrace: ptr("s")}).Error(),
			services.NewMissingPluginException().Error(),
			services.NewMissingPluginException("x").Error(),
		}, " | ")
	})

	if len(c.cases) != len(ref.Cases) {
		t.Errorf("%d cases, Flutter has %d", len(c.cases), len(ref.Cases))
	}
	mismatches := 0
	for i := range min(len(c.cases), len(ref.Cases)) {
		got, want := c.cases[i], ref.Cases[i]
		if got != want {
			mismatches++
			if mismatches <= 20 {
				t.Errorf("case %d:\n Go:      %s => %s\n Flutter: %s => %s", i, got.Name, got.Out, want.Name, want.Out)
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d of %d cases differ", mismatches, len(ref.Cases))
	}
}
