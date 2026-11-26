package services_test

// Ports of packages/flutter/test/services/message_codecs_test.dart,
// message_codecs_vm_test.dart and message_codecs_testing.dart (Flutter
// 3.50.1), plus own tests of the Go value mapping and the error paths.
//
// Skipped / adapted:
//   - "ByteData with offset": a Go []byte view (a sub-slice) has no
//     separate offset; ported as decoding a sub-slice.
//   - JSON "should preserve UTF-8 encoding and ByteData bounds across
//     encoder chunks": ported with a sub-slice for the padded view.
//   - "should encode and decode composite message" decodes Dart's map
//     {'a': 'nested', null: {}} into a Go map[any]any (nil is a valid key).

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func strp(s string) *string { return &s }

// entries is an ordered map literal: key, value, key, value...
func entries(kv ...any) []dart.MapEntry[any, any] {
	var out []dart.MapEntry[any, any]
	for i := 0; i < len(kv); i += 2 {
		out = append(out, dart.MapEntry[any, any]{Key: kv[i], Value: kv[i+1]})
	}
	return out
}

func checkEncoding[T any](t *testing.T, codec services.MessageCodec[T], message T, expectedBytes []byte) {
	t.Helper()
	encoded := codec.EncodeMessage(message)
	if !reflect.DeepEqual(encoded, expectedBytes) {
		t.Fatalf("encoding %v: got % x, want % x", message, encoded, expectedBytes)
	}
}

func checkEncodeDecode[T any](t *testing.T, codec services.MessageCodec[T], message T) {
	t.Helper()
	encoded := codec.EncodeMessage(message)
	decoded := codec.DecodeMessage(encoded)
	if isNil(message) {
		if encoded != nil {
			t.Fatalf("null encoded as % x", encoded)
		}
		if !isNil(decoded) {
			t.Fatalf("null decoded as %v", decoded)
		}
		return
	}
	if !deepEquals(message, decoded) {
		t.Fatalf("decoded %#v, want %#v", decoded, message)
	}
	encodedAgain := codec.EncodeMessage(decoded)
	if !reflect.DeepEqual(encodedAgain, encoded) {
		t.Fatalf("re-encoded % x, want % x", encodedAgain, encoded)
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map:
		return rv.IsNil()
	}
	return false
}

// deepEquals is message_codecs_testing.dart's deepEquals over Go values: a
// decoded int equals any Go integer of the same value, NaN equals NaN, a
// typed list equals a slice of the same type, lists and maps compare
// element by element.
func deepEquals(a, b any) bool {
	if p, ok := a.(*string); ok {
		if q, ok := b.(*string); ok {
			return (p == nil) == (q == nil) && (p == nil || *p == *q)
		}
		return false
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch a.(type) {
	case []byte, []int32, []int64, []float32, []float64:
		if ra.Type() != rb.Type() || ra.Len() != rb.Len() {
			return false
		}
		for i := range ra.Len() {
			if !deepEquals(ra.Index(i).Interface(), rb.Index(i).Interface()) {
				return false
			}
		}
		return true
	}
	switch ra.Kind() {
	case reflect.Float32, reflect.Float64:
		x := ra.Float()
		if rb.Kind() != reflect.Float64 && rb.Kind() != reflect.Float32 {
			return false
		}
		if math.IsNaN(x) {
			return math.IsNaN(rb.Float())
		}
		return x == rb.Float() && math.Signbit(x) == math.Signbit(rb.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rb.CanInt() && ra.Int() == rb.Int()
	case reflect.Slice, reflect.Array:
		if ra.Kind() == reflect.Slice && ra.Type().Elem().Kind() == reflect.Struct { // []MapEntry
			m := map[any]any{}
			for i := range ra.Len() {
				m[ra.Index(i).Field(0).Interface()] = ra.Index(i).Field(1).Interface()
			}
			return deepEquals(m, b)
		}
		if rb.Kind() != reflect.Slice || ra.Len() != rb.Len() {
			return false
		}
		for i := range ra.Len() {
			if !deepEquals(ra.Index(i).Interface(), rb.Index(i).Interface()) {
				return false
			}
		}
		return true
	case reflect.Map:
		if rb.Kind() != reflect.Map || ra.Len() != rb.Len() {
			return false
		}
		for _, k := range ra.MapKeys() {
			var vb reflect.Value
			for _, kb := range rb.MapKeys() {
				if deepEquals(k.Interface(), kb.Interface()) {
					vb = rb.MapIndex(kb)
				}
			}
			if !vb.IsValid() || !deepEquals(ra.MapIndex(k).Interface(), vb.Interface()) {
				return false
			}
		}
		return true
	}
	return a == b
}

// panicOf runs f and returns what it panicked with (nil if it returned).
func panicOf(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return nil
}

func TestBinaryCodecShouldEncodeAndDecodeSimpleMessages(t *testing.T) {
	binary := services.BinaryCodec{}
	checkEncodeDecode[[]byte](t, binary, nil)
	checkEncodeDecode(t, binary, []byte{})
	checkEncodeDecode(t, binary, []byte{0xff, 0xff, 0xff, 0xf9}) // ByteData(4)..setInt32(0, -7)
}

func TestStringCodecShouldEncodeAndDecodeSimpleMessages(t *testing.T) {
	stringCodec := services.StringCodec{}
	checkEncodeDecode(t, stringCodec, nil)
	checkEncodeDecode(t, stringCodec, strp(""))
	checkEncodeDecode(t, stringCodec, strp("hello"))
	checkEncodeDecode(t, stringCodec, strp("special chars >☺\U0001F602<"))
}

func TestStringCodecByteDataWithOffset(t *testing.T) {
	stringCodec := services.StringCodec{}
	helloWorldByteData := stringCodec.EncodeMessage(strp("hello world"))
	helloByteData := stringCodec.EncodeMessage(strp("hello"))
	offsetByteData := helloWorldByteData[len(helloByteData):]
	if got := *stringCodec.DecodeMessage(offsetByteData); got != " world" {
		t.Fatalf("got %q", got)
	}
}

func TestStandardMethodCodecShouldEncodeAndDecodeObjectsProducedFromCodec(t *testing.T) {
	messageCodec := services.NewStandardMessageCodec()
	data := messageCodec.EncodeMessage(map[any]any{"foo": true, 3: "fizz"})
	if got := messageCodec.DecodeMessage(data); !deepEquals(got, map[any]any{"foo": true, 3: "fizz"}) {
		t.Fatalf("got %v", got)
	}
}

func platformExceptionOf(t *testing.T, f func()) *services.PlatformException {
	t.Helper()
	r := panicOf(f)
	e, ok := r.(*services.PlatformException)
	if !ok {
		t.Fatalf("panicked with %#v, want a *PlatformException", r)
	}
	return e
}

func TestStandardMethodCodecShouldDecodeErrorEnvelopeWithoutNativeStacktrace(t *testing.T) {
	method := services.StandardMethodCodec{}
	errorData := method.EncodeErrorEnvelope("errorCode",
		services.EncodeErrorEnvelopeOptions{Message: strp("errorMessage"), Details: "errorDetails"})
	e := platformExceptionOf(t, func() { method.DecodeEnvelope(errorData) })
	if e.Code != "errorCode" || *e.Message != "errorMessage" || e.Details != "errorDetails" {
		t.Fatalf("got %v", e)
	}
}

func TestStandardMethodCodecShouldDecodeErrorEnvelopeWithNativeStacktrace(t *testing.T) {
	method := services.StandardMethodCodec{}
	messageCodec := services.NewStandardMessageCodec()
	buffer := foundation.NewWriteBuffer()
	buffer.PutUint8(1)
	messageCodec.WriteValue(buffer, "errorCode")
	messageCodec.WriteValue(buffer, "errorMessage")
	messageCodec.WriteValue(buffer, "errorDetails")
	messageCodec.WriteValue(buffer, "errorStacktrace")
	errorData := buffer.Done()
	e := platformExceptionOf(t, func() { method.DecodeEnvelope(errorData) })
	if e.Stacktrace == nil || *e.Stacktrace != "errorStacktrace" {
		t.Fatalf("got %v", e)
	}
}

func TestStandardMethodCodecShouldAllowNullErrorMessage(t *testing.T) {
	method := services.StandardMethodCodec{}
	errorData := method.EncodeErrorEnvelope("errorCode", services.EncodeErrorEnvelopeOptions{Details: "errorDetails"})
	e := platformExceptionOf(t, func() { method.DecodeEnvelope(errorData) })
	if e.Code != "errorCode" || e.Message != nil || e.Details != "errorDetails" {
		t.Fatalf("got %v", e)
	}
}

func TestJSONMethodCodecShouldDecodeErrorEnvelopeWithoutNativeStacktrace(t *testing.T) {
	jsonMethodCodec := services.JSONMethodCodec{}
	errorData := jsonMethodCodec.EncodeErrorEnvelope("errorCode",
		services.EncodeErrorEnvelopeOptions{Message: strp("errorMessage"), Details: "errorDetails"})
	e := platformExceptionOf(t, func() { jsonMethodCodec.DecodeEnvelope(errorData) })
	if e.Code != "errorCode" || *e.Message != "errorMessage" || e.Details != "errorDetails" {
		t.Fatalf("got %v", e)
	}
}

func TestJSONMethodCodecShouldDecodeErrorEnvelopeWithNativeStacktrace(t *testing.T) {
	jsonMethodCodec := services.JSONMethodCodec{}
	errorData := services.StringCodec{}.EncodeMessage(strp(dart.JSONEncode(
		[]any{"errorCode", "errorMessage", "errorDetails", "errorStacktrace"})))
	e := platformExceptionOf(t, func() { jsonMethodCodec.DecodeEnvelope(errorData) })
	if e.Stacktrace == nil || *e.Stacktrace != "errorStacktrace" {
		t.Fatalf("got %v", e)
	}
}

func TestJSONMessageCodecShouldEncodeAndDecodeSimpleMessages(t *testing.T) {
	json := services.JSONMessageCodec{}
	for _, m := range []any{nil, true, false, 7, -7, 98742923489, -98742923489, 3.14, "", "hello",
		"special chars >☺\U0001F602<"} {
		checkEncodeDecode(t, json, m)
	}
}

func TestJSONMessageCodecShouldEncodeAndDecodeCompositeMessage(t *testing.T) {
	message := []any{nil, true, false, -707, -7000000007, -3.14, "", "hello",
		[]any{"nested", []any{}},
		map[string]any{"a": "nested", "b": map[string]any{}},
		"world",
	}
	checkEncodeDecode[any](t, services.JSONMessageCodec{}, message)
}

func TestJSONMessageCodecShouldPreserveUTF8AcrossEncoderChunks(t *testing.T) {
	json := services.JSONMessageCodec{}
	// Place a four-byte character across JsonUtf8Encoder's default 256-byte
	// chunk boundary.
	message := strings.Repeat("a", 254) + "\U0001F602"
	expected := dart.UTF8Encode(dart.JSONEncode(message))
	encoded := json.EncodeMessage(message)
	if !reflect.DeepEqual(encoded, expected) {
		t.Fatalf("got % x", encoded)
	}
	padded := append(append([]byte{0x5b}, expected...), 0x5d)
	view := padded[1 : len(expected)+1]
	if got := json.DecodeMessage(view); got != message {
		t.Fatalf("got %v", got)
	}
}

func TestJSONMessageCodecShouldEncodeAndDecodeBigNumbers(t *testing.T) {
	json := services.JSONMessageCodec{}
	checkEncodeDecode[any](t, json, 9223372036854775807)
	checkEncodeDecode[any](t, json, -9223372036854775807)
	checkEncodeDecode[any](t, json, []any{-7000000000000000007})
}

func TestStandardMessageCodecShouldEncodeSizesCorrectlyAtBoundaryCases(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	checkEncoding[any](t, standard, make([]byte, 253), append([]byte{8, 253}, make([]byte, 253)...))
	checkEncoding[any](t, standard, make([]byte, 254), append([]byte{8, 254, 254, 0}, make([]byte, 254)...))
	checkEncoding[any](t, standard, make([]byte, 0xffff), append([]byte{8, 254, 0xff, 0xff}, make([]byte, 0xffff)...))
	checkEncoding[any](t, standard, make([]byte, 0xffff+1), append([]byte{8, 255, 0, 0, 1, 0}, make([]byte, 0xffff+1)...))
}

func TestStandardMessageCodecShouldEncodeAndDecodeSimpleMessages(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	for _, m := range []any{nil, true, false, 7, -7, 98742923489, -98742923489, 3.14, math.Inf(1), math.NaN(),
		"", "hello", "special chars >☺\U0001F602<"} {
		checkEncodeDecode(t, standard, m)
	}
}

func TestStandardMessageCodecShouldEncodeAndDecodeCompositeMessage(t *testing.T) {
	special := []float64{math.Inf(-1), -math.MaxFloat64, -math.SmallestNonzeroFloat64, math.Copysign(0, -1), 0,
		math.SmallestNonzeroFloat64, math.MaxFloat64, math.Inf(1), math.NaN()}
	special32 := make([]float32, len(special))
	for i, v := range special {
		special32[i] = float32(v)
	}
	message := []any{nil, true, false, -707, -7000000007, -3.14, "", "hello",
		[]byte{0xBA, 0x5E, 0xBA, 0x11},
		[]int32{-0x7fffffff - 1, 0, 0x7fffffff},
		nil, // ensures the offset of the following list is unaligned.
		nil, // ensures the offset of the following list is unaligned.
		special,
		special32,
		[]any{"nested", []any{}},
		map[any]any{"a": "nested", nil: map[any]any{}},
		"world",
	}
	checkEncodeDecode[any](t, services.NewStandardMessageCodec(), message)
}

func TestStandardMessageCodecShouldAlignDoublesTo8Bytes(t *testing.T) {
	checkEncoding[any](t, services.NewStandardMessageCodec(), 1.0,
		[]byte{6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xf0, 0x3f})
}

func TestStandardMessageCodecShouldEncodeIntegersCorrectlyAtBoundaryCases(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	checkEncoding[any](t, standard, -0x7fffffff-1, []byte{3, 0x00, 0x00, 0x00, 0x80})
	checkEncoding[any](t, standard, -0x7fffffff-2, []byte{4, 0xff, 0xff, 0xff, 0x7f, 0xff, 0xff, 0xff, 0xff})
	checkEncoding[any](t, standard, 0x7fffffff, []byte{3, 0xff, 0xff, 0xff, 0x7f})
	checkEncoding[any](t, standard, 0x7fffffff+1, []byte{4, 0x00, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00})
	checkEncoding[any](t, standard, -0x7fffffffffffffff-1, []byte{4, 0, 0, 0, 0, 0, 0, 0, 0x80})
	// -0x7fffffffffffffff - 2 and 0x7fffffffffffffff + 1 wrap around in
	// Dart's 64-bit ints, as in Go's.
	minus2 := int64(-0x7fffffffffffffff - 1)
	minus2--
	checkEncoding[any](t, standard, minus2, []byte{4, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f})
	checkEncoding[any](t, standard, 0x7fffffffffffffff, []byte{4, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f})
	plus1 := int64(0x7fffffffffffffff)
	plus1++
	checkEncoding[any](t, standard, plus1, []byte{4, 0, 0, 0, 0, 0, 0, 0, 0x80})
}

func TestStandardMessageCodecShouldEncodeAndDecodeBigNumbers(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	checkEncodeDecode[any](t, standard, 9223372036854775807)
	checkEncodeDecode[any](t, standard, -9223372036854775807)
	checkEncodeDecode[any](t, standard, []any{-7000000000000000007,
		[]int64{-0x7fffffffffffffff - 1, 0, 0x7fffffffffffffff}})
}

func TestToStringWorksAsIntended(t *testing.T) {
	methodCall := services.NewMethodCall("sample method")
	platformException := services.NewPlatformException("100")
	missingPluginException := services.NewMissingPluginException()
	if got := methodCall.String(); got != "MethodCall(sample method, null)" {
		t.Errorf("got %q", got)
	}
	if got := platformException.Error(); got != "PlatformException(100, null, null, null)" {
		t.Errorf("got %q", got)
	}
	if got := missingPluginException.Error(); got != "MissingPluginException(null)" {
		t.Errorf("got %q", got)
	}
	// Own: with values.
	if got := services.NewMethodCall("m", []any{1, 2.5, "x", nil}).String(); got != "MethodCall(m, [1, 2.5, x, null])" {
		t.Errorf("got %q", got)
	}
	if got := services.NewPlatformException("c", services.PlatformExceptionOptions{Message: strp("msg"),
		Details: map[string]any{"a": 1}, Stacktrace: strp("st")}).Error(); got != "PlatformException(c, msg, {a: 1}, st)" {
		t.Errorf("got %q", got)
	}
	if got := services.NewMissingPluginException("m").Error(); got != "MissingPluginException(m)" {
		t.Errorf("got %q", got)
	}
}

// ---- own tests

func TestStandardMessageCodecGoValues(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	same := func(a, b any) {
		t.Helper()
		if x, y := standard.EncodeMessage(a), standard.EncodeMessage(b); !reflect.DeepEqual(x, y) {
			t.Errorf("%#v encodes as % x, %#v as % x", a, x, b, y)
		}
	}
	// Every Go integer type is a Dart int.
	same(int8(-3), -3)
	same(uint8(200), 200)
	same(int16(-300), -300)
	same(uint16(60000), 60000)
	same(int32(-7), -7)
	same(uint32(0xffffffff), 0xffffffff)
	same(int64(1)<<40, 1<<40)
	same(uint64(1)<<62, 1<<62)
	same(uint(5), 5)
	same(uint64(math.MaxInt64), math.MaxInt64)
	// float32 is the double it widens to.
	same(float32(0.1), float64(float32(0.1)))
	// Pointers: their value, or null.
	same(strp("x"), "x")
	same((*string)(nil), nil)
	// Typed slices other than the typed lists are lists.
	same([]string{"a", "b"}, []any{"a", "b"})
	same([2]int{1, 2}, []any{1, 2})
	same([]int{1}, []any{1})
	// A Go map's entries are sorted; a []MapEntry keeps its order.
	same(map[string]int{"b": 1, "a": 2}, []dart.MapEntry[any, any]{{Key: "a", Value: 2}, {Key: "b", Value: 1}})
	same(map[any]any{"s": 1, 2: 1, 1.5: 1, true: 1, false: 1, nil: 1, -1: 1},
		entries(nil, 1, false, 1, true, 1, -1, 1, 1.5, 1, 2, 1, "s", 1))
	same(map[any]any{2.0: 0, 2: 0, 1: 0}, entries(1, 0, 2, 0, 2.0, 0))
	same(map[any]any{[2]int{1, 2}: 0, [1]int{3}: 0}, entries([2]int{1, 2}, 0, [1]int{3}, 0))
	if got := standard.EncodeMessage([]dart.MapEntry[string, int]{{Key: "b", Value: 1}, {Key: "a", Value: 2}}); !reflect.DeepEqual(got,
		[]byte{13, 2, 7, 1, 'b', 3, 1, 0, 0, 0, 7, 1, 'a', 3, 2, 0, 0, 0}) {
		t.Errorf("got % x", got)
	}
	// A nil slice or map is an empty list or map (only nil interfaces and
	// pointers are null).
	same([]any(nil), []any{})
	same(map[string]any(nil), map[string]any{})
	// uint64 beyond int64 and unsupported values throw ArgumentError.value.
	for _, v := range []any{uint64(1) << 63, struct{}{}, func() {}, make(chan int)} {
		r := panicOf(func() { standard.EncodeMessage(v) })
		if e, ok := r.(dart.ArgumentError); !ok || !e.HasValue {
			t.Errorf("%#v: panicked with %#v", v, r)
		}
	}
	if got := panicOf(func() { standard.EncodeMessage(uint64(1) << 63) }).(error).Error(); got != "Invalid argument: 9223372036854775808" {
		t.Errorf("got %q", got)
	}
}

func TestStandardMessageCodecDecodedTypes(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	decoded := standard.DecodeMessage(standard.EncodeMessage([]any{int8(1), int64(1) << 40, float32(1.5), []byte{1},
		[]int32{1}, []int64{1}, []float32{1}, []float64{1}, map[string]int{"a": 1}, []string{"x"}}))
	want := []string{"[]interface {}", "int", "int", "float64", "[]uint8", "[]int32", "[]int64", "[]float32",
		"[]float64", "map[interface {}]interface {}", "[]interface {}"}
	if got := reflect.TypeOf(decoded).String(); got != want[0] {
		t.Fatalf("got %s", got)
	}
	for i, v := range decoded.([]any) {
		if got := reflect.TypeOf(v).String(); got != want[i+1] {
			t.Errorf("%d: got %s, want %s", i, got, want[i+1])
		}
	}
}

func TestStandardMessageCodecErrors(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	formatError := func(data []byte, want string) {
		t.Helper()
		r := panicOf(func() { standard.DecodeMessage(data) })
		e, ok := r.(dart.FormatException)
		if !ok || e.Error() != want {
			t.Errorf("% x: panicked with %#v, want %q", data, r, want)
		}
	}
	formatError([]byte{}, "FormatException: Message corrupted")
	formatError([]byte{0, 0}, "FormatException: Message corrupted")  // trailing data
	formatError([]byte{15}, "FormatException: Message corrupted")    // unknown type
	formatError([]byte{12, 1}, "FormatException: Message corrupted") // missing element
	formatError([]byte{7, 1, 0xff}, "FormatException: Invalid UTF-8 byte (at offset 0)")
	// The large-int type is read as a string.
	if got := standard.DecodeMessage([]byte{5, 2, '1', 'f'}); got != "1f" {
		t.Errorf("got %v", got)
	}
	// readSize's 254 and 255 forms.
	if got := standard.DecodeMessage(append([]byte{8, 254, 1, 0}, make([]byte, 1)...)); len(got.([]byte)) != 1 {
		t.Errorf("got %v", got)
	}
	if got := standard.DecodeMessage(append([]byte{8, 255, 2, 0, 0, 0}, make([]byte, 2)...)); len(got.([]byte)) != 2 {
		t.Errorf("got %v", got)
	}
	// A key Go can't hash.
	r := panicOf(func() { standard.DecodeMessage([]byte{13, 1, 12, 0, 0}) })
	if _, ok := r.(dart.UnsupportedError); !ok {
		t.Errorf("got %#v", r)
	}
	// writeSize's assert.
	if r := panicOf(func() { standard.WriteSize(foundation.NewWriteBuffer(), -1) }); r == nil {
		t.Error("no panic")
	}
	if r := panicOf(func() { standard.WriteSize(foundation.NewWriteBuffer(), 0x100000000) }); r == nil {
		t.Error("no panic")
	}
	b := foundation.NewWriteBuffer()
	standard.WriteSize(b, 0xffffffff)
	if got := b.Done(); !reflect.DeepEqual(got, []byte{255, 0xff, 0xff, 0xff, 0xff}) {
		t.Errorf("got % x", got)
	}
	// Strings: UTF-8, size in bytes.
	if got := standard.EncodeMessage("é"); !reflect.DeepEqual(got, []byte{7, 2, 0xc3, 0xa9}) {
		t.Errorf("got % x", got)
	}
}

// extendedCodec extends StandardMessageCodec with a type of its own, as
// Flutter's docs describe.
type extendedCodec struct{ services.StandardMessageCodec }

type point struct{ x, y int }

func (c *extendedCodec) WriteValue(buffer *foundation.WriteBuffer, value any) {
	if p, ok := value.(point); ok {
		buffer.PutUint8(128)
		c.StandardMessageCodec.WriteValue(buffer, p.x)
		c.StandardMessageCodec.WriteValue(buffer, p.y)
		return
	}
	c.StandardMessageCodec.WriteValue(buffer, value)
}

func (c *extendedCodec) ReadValueOfType(typ int, buffer *foundation.ReadBuffer) any {
	if typ == 128 {
		return point{c.ReadValue(buffer).(int), c.ReadValue(buffer).(int)}
	}
	return c.StandardMessageCodec.ReadValueOfType(typ, buffer)
}

func TestStandardMessageCodecSubclass(t *testing.T) {
	c := &extendedCodec{}
	c.SetSelf(c)
	message := []any{point{1, 2}, map[any]any{"p": point{3, 4}}}
	encoded := c.EncodeMessage(message)
	if !reflect.DeepEqual(encoded, []byte{12, 2, 128, 3, 1, 0, 0, 0, 3, 2, 0, 0, 0, 13, 1, 7, 1, 'p', 128, 3, 3, 0, 0, 0, 3, 4, 0, 0, 0}) {
		t.Fatalf("got % x", encoded)
	}
	if got := c.DecodeMessage(encoded); !reflect.DeepEqual(got, []any{point{1, 2}, map[any]any{"p": point{3, 4}}}) {
		t.Fatalf("got %v", got)
	}
	// Through a method codec.
	m := services.NewStandardMethodCodec(&c.StandardMessageCodec)
	if got := m.DecodeEnvelope(m.EncodeSuccessEnvelope(point{5, 6})); got != (point{5, 6}) {
		t.Fatalf("got %v", got)
	}
	call := m.DecodeMethodCall(m.EncodeMethodCall(services.NewMethodCall("m", point{7, 8})))
	if call.Arguments != (point{7, 8}) {
		t.Fatalf("got %v", call)
	}
	// The plain codec rejects the type.
	if _, ok := panicOf(func() { services.NewStandardMessageCodec().EncodeMessage(point{}) }).(dart.ArgumentError); !ok {
		t.Fatal("no ArgumentError")
	}
}

func TestStandardMethodCodecErrors(t *testing.T) {
	m := services.StandardMethodCodec{}
	formatError := func(f func(), want string) {
		t.Helper()
		r := panicOf(f)
		e, ok := r.(dart.FormatException)
		if !ok || e.Error() != want {
			t.Errorf("panicked with %#v, want %q", r, want)
		}
	}
	formatError(func() { m.DecodeEnvelope([]byte{}) }, "FormatException: Expected envelope, got nothing")
	formatError(func() { m.DecodeEnvelope([]byte{1, 3, 1, 0, 0, 0, 0, 0}) }, "FormatException: Invalid envelope")
	formatError(func() { m.DecodeEnvelope([]byte{1, 7, 1, 'c', 3, 1, 0, 0, 0, 0}) }, "FormatException: Invalid envelope")
	formatError(func() { m.DecodeEnvelope([]byte{1, 7, 1, 'c', 0, 0, 0, 0}) }, "FormatException: Invalid envelope")
	formatError(func() { m.DecodeMethodCall([]byte{3, 1, 0, 0, 0, 0}) }, "FormatException: Invalid method call")
	formatError(func() { m.DecodeMethodCall([]byte{7, 1, 'm', 0, 0}) }, "FormatException: Invalid method call")
	if r := panicOf(func() { m.DecodeMethodCall(nil) }); r == nil {
		t.Error("no panic for a null method call")
	}
	r := panicOf(func() { m.DecodeEnvelope([]byte{1, 7, 1, 'c', 0, 0, 3, 1, 0, 0, 0}) })
	if e, ok := r.(dart.TypeError); !ok || e.Message != "type 'int' is not a subtype of type 'String?' in type cast" {
		t.Errorf("got %#v", r)
	}
	// A null stack trace after the details.
	e := platformExceptionOf(t, func() { m.DecodeEnvelope([]byte{1, 7, 1, 'c', 0, 0, 0}) })
	if e.Stacktrace != nil || e.Message != nil {
		t.Errorf("got %v", e)
	}
	// Success.
	if got := m.DecodeEnvelope([]byte{0, 7, 1, 'x'}); got != "x" {
		t.Errorf("got %v", got)
	}
	if got := m.EncodeMethodCall(services.MethodCall{Method: "m"}); !reflect.DeepEqual(got, []byte{7, 1, 'm', 0}) {
		t.Errorf("got % x", got)
	}
	if got := m.EncodeErrorEnvelope("c"); !reflect.DeepEqual(got, []byte{1, 7, 1, 'c', 0, 0}) {
		t.Errorf("got % x", got)
	}
}

func TestJSONMethodCodecErrors(t *testing.T) {
	m := services.JSONMethodCodec{}
	formatError := func(json string, envelope bool, want string) {
		t.Helper()
		r := panicOf(func() {
			if envelope {
				m.DecodeEnvelope([]byte(json))
			} else {
				m.DecodeMethodCall([]byte(json))
			}
		})
		e, ok := r.(dart.FormatException)
		if !ok || e.Error() != want {
			t.Errorf("%s: panicked with %#v, want %q", json, r, want)
		}
	}
	formatError(`[1]`, false, "FormatException: Expected method call Map, got [1]")
	formatError(`{"method": 1}`, false, "FormatException: Invalid method call: {method: 1}")
	formatError(`{"args": 1}`, false, "FormatException: Invalid method call: {args: 1}")
	formatError(`{"a": 1}`, true, "FormatException: Expected envelope List, got {a: 1}")
	formatError(`[1, 2]`, true, "FormatException: Invalid envelope: [1, 2]")
	formatError(`[1, 2, 3]`, true, "FormatException: Invalid envelope: [1, 2, 3]")
	formatError(`["c", 2, 3]`, true, "FormatException: Invalid envelope: [c, 2, 3]")
	formatError(`["c", "m", 3, 4]`, true, "FormatException: Invalid envelope: [c, m, 3, 4]")
	formatError(`["c", 1, 3, "s"]`, true, "FormatException: Invalid envelope: [c, 1, 3, s]")
	formatError(`[2, "m", 3, "s"]`, true, "FormatException: Invalid envelope: [2, m, 3, s]")
	if r := panicOf(func() { m.DecodeMethodCall(nil) }); r.(dart.FormatException).Error() != "FormatException: Expected method call Map, got null" {
		t.Errorf("got %v", r)
	}
	call := m.DecodeMethodCall([]byte(`{"method": "m"}`))
	if call.Method != "m" || call.Arguments != nil {
		t.Errorf("got %v", call)
	}
	e := platformExceptionOf(t, func() { m.DecodeEnvelope([]byte(`["c", null, 3, null]`)) })
	if e.Message != nil || e.Stacktrace != nil || e.Details != 3 {
		t.Errorf("got %v", e)
	}
	if got := string(m.EncodeMethodCall(services.NewMethodCall("m", map[string]any{"b": 1, "a": 2}))); got != `{"method":"m","args":{"a":2,"b":1}}` {
		t.Errorf("got %s", got)
	}
	if got := string(m.EncodeErrorEnvelope("c")); got != `["c",null,null]` {
		t.Errorf("got %s", got)
	}
	if got := string(m.EncodeSuccessEnvelope(nil)); got != `[null]` {
		t.Errorf("got %s", got)
	}
}

func TestCodecsNull(t *testing.T) {
	if (services.JSONMessageCodec{}).EncodeMessage(nil) != nil || (services.JSONMessageCodec{}).DecodeMessage(nil) != nil {
		t.Error("JSON null")
	}
	if services.NewStandardMessageCodec().DecodeMessage(nil) != nil {
		t.Error("standard null")
	}
	var pe *services.PlatformException
	if !errors.As(error(services.NewPlatformException("x")), &pe) {
		t.Error("errors.As")
	}
}
