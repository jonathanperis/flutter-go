package services

// Port of message_codecs.dart (Flutter 3.50.1).
//
// How Dart's values map to Go's in the codecs (see API_MAPPING):
//
//   - null is nil (a nil pointer is null too; a non-nil pointer is the value
//     it points to);
//   - bool is bool; String is string (UTF-8; see dart.UTF8Encode for
//     strings that aren't valid UTF-8);
//   - int is int: decoded integers are always Go ints (64-bit), whether
//     they were sent as 32- or 64-bit; every Go integer type is accepted
//     when encoding, a uint64 above math.MaxInt64 being unencodable (a Dart
//     int is a signed 64-bit integer);
//   - double is float64 (a float32 is encoded as the double it widens to);
//   - Uint8List, Int32List, Int64List, Float32List and Float64List are
//     []byte, []int32, []int64, []float32 and []float64;
//   - List is any other slice or array; decoded lists are []any;
//   - Map is a *dart.LinkedMap (Dart's LinkedHashMap): decoded maps are
//     *dart.LinkedMap for both the standard and the JSON codecs, in the
//     order the message lists them (a repeated key keeps its first place
//     and its last value, keys compare as Dart's map compares them: see
//     dart.LinkedMap), and a *dart.LinkedMap is written in its order. A
//     []dart.MapEntry[K, V] is also written as a map in its order, and, for
//     convenience, a Go map: it has no order, so its entries are written
//     sorted by key (nil, then false and true, then numbers in numeric
//     order, then strings in byte order, then other keys by their Dart
//     toString).

import (
	"cmp"
	"math"
	"reflect"
	"slices"
	"strings"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

const writeBufferStartCapacity = 64

// BinaryCodec is a MessageCodec with unencoded binary messages ([]byte).
type BinaryCodec struct{}

// DecodeMessage returns message unchanged.
func (BinaryCodec) DecodeMessage(message []byte) []byte { return message }

// EncodeMessage returns message unchanged.
func (BinaryCodec) EncodeMessage(message []byte) []byte { return message }

// StringCodec is a MessageCodec with UTF-8 encoded String messages
// (*string: nil is null).
type StringCodec struct{}

// DecodeMessage decodes UTF-8 (panicking with a dart.FormatException on
// malformed input; a leading byte-order mark is dropped, as Dart's
// utf8.decode does).
func (StringCodec) DecodeMessage(message []byte) *string {
	if message == nil {
		return nil
	}
	s := dart.UTF8Decode(message)
	return &s
}

// EncodeMessage encodes message in UTF-8.
func (StringCodec) EncodeMessage(message *string) []byte {
	if message == nil {
		return nil
	}
	return dart.UTF8Encode(*message)
}

// JSONMessageCodec is a MessageCodec with UTF-8 encoded JSON messages:
// acyclic values made of nil, bools, numbers, strings, lists and maps with
// string keys. Decoded objects are *dart.LinkedMap (string keys, in the
// message's order), arrays []any, integers int and other numbers float64.
type JSONMessageCodec struct{}

// EncodeMessage encodes message as UTF-8 JSON (dart:convert's
// JsonUtf8Encoder; it panics with a dart.JSONUnsupportedObjectError for a
// value JSON can't represent).
func (JSONMessageCodec) EncodeMessage(message any) []byte {
	if message == nil {
		return nil
	}
	return dart.JSONUTF8Encode(message)
}

// DecodeMessage decodes UTF-8 JSON (utf8.decoder fused with json.decoder;
// it panics with a dart.FormatException on malformed input).
func (JSONMessageCodec) DecodeMessage(message []byte) any {
	if message == nil {
		return nil
	}
	return dart.JSONUTF8Decode(message)
}

// JSONMethodCodec is a MethodCodec with UTF-8 encoded JSON method calls
// and result envelopes: a method call is the map {"method": name, "args":
// arguments}; a success envelope is [result], an error envelope [code,
// message, details] (or [code, message, details, stacktrace] from the
// platform).
type JSONMethodCodec struct{}

func (JSONMethodCodec) EncodeMethodCall(methodCall MethodCall) []byte {
	return JSONMessageCodec{}.EncodeMessage([]dart.MapEntry[string, any]{
		{Key: "method", Value: methodCall.Method},
		{Key: "args", Value: methodCall.Arguments},
	})
}

func (JSONMethodCodec) DecodeMethodCall(methodCall []byte) MethodCall {
	decoded := JSONMessageCodec{}.DecodeMessage(methodCall)
	m, ok := decoded.(*dart.LinkedMap)
	if !ok {
		panic(dart.NewFormatException("Expected method call Map, got " + dart.ToString(decoded)))
	}
	if method, ok := m.Lookup("method").(string); ok {
		return MethodCall{Method: method, Arguments: m.Lookup("args")}
	}
	panic(dart.NewFormatException("Invalid method call: " + dart.ToString(decoded)))
}

func isNullOrString(v any) bool {
	_, ok := v.(string)
	return v == nil || ok
}

func stringPointer(v any) *string {
	if v == nil {
		return nil
	}
	s := v.(string)
	return &s
}

func (JSONMethodCodec) DecodeEnvelope(envelope []byte) any {
	decoded := JSONMessageCodec{}.DecodeMessage(envelope)
	list, ok := decoded.([]any)
	if !ok {
		panic(dart.NewFormatException("Expected envelope List, got " + dart.ToString(decoded)))
	}
	if len(list) == 1 {
		return list[0]
	}
	if len(list) == 3 {
		if code, ok := list[0].(string); ok && isNullOrString(list[1]) {
			panic(&PlatformException{Code: code, Message: stringPointer(list[1]), Details: list[2]})
		}
	}
	if len(list) == 4 {
		if code, ok := list[0].(string); ok && isNullOrString(list[1]) && isNullOrString(list[3]) {
			panic(&PlatformException{Code: code, Message: stringPointer(list[1]), Details: list[2],
				Stacktrace: stringPointer(list[3])})
		}
	}
	panic(dart.NewFormatException("Invalid envelope: " + dart.ToString(decoded)))
}

func (JSONMethodCodec) EncodeSuccessEnvelope(result any) []byte {
	return JSONMessageCodec{}.EncodeMessage([]any{result})
}

func (JSONMethodCodec) EncodeErrorEnvelope(code string, opts ...EncodeErrorEnvelopeOptions) []byte {
	o := errorEnvelopeOptions(opts)
	var message any
	if o.Message != nil {
		message = *o.Message
	}
	return JSONMessageCodec{}.EncodeMessage([]any{code, message, o.Details})
}

// StandardMessageCodecOverridable are StandardMessageCodec's methods that a
// type embedding it may override; the codec calls them through its self
// (see API_MAPPING "Mixins and overridable methods").
type StandardMessageCodecOverridable interface {
	WriteValue(buffer *foundation.WriteBuffer, value any)
	ReadValue(buffer *foundation.ReadBuffer) any
	ReadValueOfType(typ int, buffer *foundation.ReadBuffer) any
	WriteSize(buffer *foundation.WriteBuffer, value int)
	ReadSize(buffer *foundation.ReadBuffer) int
}

// StandardMessageCodec is a MessageCodec using the Flutter standard binary
// encoding (see the file comment for the Go values it encodes and
// decodes). It is used by pointer; its zero value is ready to use. The
// codec is extensible: a type embedding it overrides WriteValue and
// ReadValueOfType (using type discriminators of 128 and up), calls the
// embedded methods for the values it doesn't handle, and calls SetSelf
// once when created.
type StandardMessageCodec struct {
	self StandardMessageCodecOverridable
}

// NewStandardMessageCodec is StandardMessageCodec().
func NewStandardMessageCodec() *StandardMessageCodec { return &StandardMessageCodec{} }

// SetSelf points the codec at the value embedding it, whose methods it
// calls so overrides run.
func (c *StandardMessageCodec) SetSelf(self StandardMessageCodecOverridable) { c.self = self }

func (c StandardMessageCodec) outer() StandardMessageCodecOverridable {
	if c.self == nil {
		return c
	}
	return c.self
}

// The type labels of the format (they must not change: the encoding may be
// used for persistent storage).
const (
	valueNull        = 0
	valueTrue        = 1
	valueFalse       = 2
	valueInt32       = 3
	valueInt64       = 4
	valueLargeInt    = 5
	valueFloat64     = 6
	valueString      = 7
	valueUint8List   = 8
	valueInt32List   = 9
	valueInt64List   = 10
	valueFloat64List = 11
	valueList        = 12
	valueMap         = 13
	valueFloat32List = 14
)

func (c StandardMessageCodec) EncodeMessage(message any) []byte {
	if isNull(message) {
		return nil
	}
	buffer := foundation.NewWriteBuffer(foundation.WriteBufferOptions{StartCapacity: writeBufferStartCapacity})
	c.outer().WriteValue(buffer, message)
	return buffer.Done()
}

func (c StandardMessageCodec) DecodeMessage(message []byte) any {
	if message == nil {
		return nil
	}
	buffer := foundation.NewReadBuffer(message)
	result := c.outer().ReadValue(buffer)
	if buffer.HasRemaining() {
		panic(dart.NewFormatException("Message corrupted"))
	}
	return result
}

// isNull tells whether v is nil or a nil pointer.
func isNull(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// WriteValue writes value to buffer: a type discriminator byte, then the
// value itself (recursively for lists and maps). Discriminators 0 to 127
// are reserved for this class: null 0, true 1, false 2, 32-bit int 3,
// 64-bit int 4, larger integers 5 (never written; see Flutter's docs),
// double 6, String 7, Uint8List 8, Int32List 9, Int64List 10, Float64List
// 11, List 12, Map 13, Float32List 14. It panics with a dart.ArgumentError
// for a value it can't encode.
func (c StandardMessageCodec) WriteValue(buffer *foundation.WriteBuffer, value any) {
	self := c.outer()
	switch v := value.(type) {
	case nil:
		buffer.PutUint8(valueNull)
		return
	case bool:
		if v {
			buffer.PutUint8(valueTrue)
		} else {
			buffer.PutUint8(valueFalse)
		}
		return
	case float64:
		buffer.PutUint8(valueFloat64)
		buffer.PutFloat64(v)
		return
	case float32:
		buffer.PutUint8(valueFloat64)
		buffer.PutFloat64(float64(v))
		return
	case string:
		buffer.PutUint8(valueString)
		utf8Bytes := dart.UTF8Encode(v)
		self.WriteSize(buffer, len(utf8Bytes))
		buffer.PutUint8List(utf8Bytes)
		return
	case []byte:
		buffer.PutUint8(valueUint8List)
		self.WriteSize(buffer, len(v))
		buffer.PutUint8List(v)
		return
	case []int32:
		buffer.PutUint8(valueInt32List)
		self.WriteSize(buffer, len(v))
		buffer.PutInt32List(v)
		return
	case []int64:
		buffer.PutUint8(valueInt64List)
		self.WriteSize(buffer, len(v))
		buffer.PutInt64List(v)
		return
	case []float32:
		buffer.PutUint8(valueFloat32List)
		self.WriteSize(buffer, len(v))
		buffer.PutFloat32List(v)
		return
	case []float64:
		buffer.PutUint8(valueFloat64List)
		self.WriteSize(buffer, len(v))
		buffer.PutFloat64List(v)
		return
	case *dart.LinkedMap:
		if v == nil {
			self.WriteValue(buffer, nil)
			return
		}
		buffer.PutUint8(valueMap)
		self.WriteSize(buffer, v.Len())
		for key, value := range v.Entries() {
			self.WriteValue(buffer, key)
			self.WriteValue(buffer, value)
		}
		return
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		writeInt(buffer, rv.Int())
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if u := rv.Uint(); u <= math.MaxInt64 {
			writeInt(buffer, int64(u))
			return
		}
	case reflect.Pointer:
		if rv.IsNil() {
			self.WriteValue(buffer, nil)
		} else {
			self.WriteValue(buffer, rv.Elem().Interface())
		}
		return
	case reflect.Slice, reflect.Array:
		if isMapEntrySlice(rv) {
			buffer.PutUint8(valueMap)
			self.WriteSize(buffer, rv.Len())
			for i := range rv.Len() {
				entry := rv.Index(i)
				self.WriteValue(buffer, entry.Field(0).Interface())
				self.WriteValue(buffer, entry.Field(1).Interface())
			}
			return
		}
		buffer.PutUint8(valueList)
		self.WriteSize(buffer, rv.Len())
		for i := range rv.Len() {
			self.WriteValue(buffer, rv.Index(i).Interface())
		}
		return
	case reflect.Map:
		buffer.PutUint8(valueMap)
		self.WriteSize(buffer, rv.Len())
		keys := rv.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int { return compareKeys(a.Interface(), b.Interface()) })
		for _, k := range keys {
			self.WriteValue(buffer, k.Interface())
			self.WriteValue(buffer, rv.MapIndex(k).Interface())
		}
		return
	}
	panic(dart.NewArgumentErrorValue(value, "", ""))
}

func writeInt(buffer *foundation.WriteBuffer, value int64) {
	if -0x7fffffff-1 <= value && value <= 0x7fffffff {
		buffer.PutUint8(valueInt32)
		buffer.PutInt32(int(value))
	} else {
		buffer.PutUint8(valueInt64)
		buffer.PutInt64(int(value))
	}
}

var mapEntryPkgPath = reflect.TypeFor[dart.MapEntry[string, any]]().PkgPath()

func isMapEntrySlice(rv reflect.Value) bool {
	t := rv.Type().Elem()
	return t.Kind() == reflect.Struct && t.PkgPath() == mapEntryPkgPath && strings.HasPrefix(t.Name(), "MapEntry[")
}

// keyRank orders the kinds of map keys: null, bools, numbers, strings,
// anything else.
func keyRank(v any) (rank int, num float64, isInt bool, i int64) {
	if isNull(v) {
		return 0, 0, false, 0
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return 1, 1, false, 0
		}
		return 1, 0, false, 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 2, float64(rv.Int()), true, rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return 2, float64(rv.Uint()), true, int64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return 2, rv.Float(), false, 0
	case reflect.String:
		return 3, 0, false, 0
	}
	return 4, 0, false, 0
}

// compareKeys is the order in which a Go map's entries are written.
func compareKeys(a, b any) int {
	ra, na, ia, xa := keyRank(a)
	rb, nb, ib, xb := keyRank(b)
	if ra != rb {
		return cmp.Compare(ra, rb)
	}
	switch ra {
	case 1:
		return cmp.Compare(na, nb)
	case 2:
		if ia && ib {
			return cmp.Compare(xa, xb)
		}
		if c := cmp.Compare(na, nb); c != 0 {
			return c
		}
		return cmp.Compare(dart.ToString(a), dart.ToString(b))
	case 3:
		return strings.Compare(reflect.ValueOf(a).String(), reflect.ValueOf(b).String())
	}
	return strings.Compare(dart.ToString(a), dart.ToString(b))
}

// ReadValue reads a value from buffer as written by WriteValue (for types
// overriding ReadValueOfType).
func (c StandardMessageCodec) ReadValue(buffer *foundation.ReadBuffer) any {
	if !buffer.HasRemaining() {
		panic(dart.NewFormatException("Message corrupted"))
	}
	typ := buffer.GetUint8()
	return c.outer().ReadValueOfType(typ, buffer)
}

// ReadValueOfType reads a value of the indicated type from buffer. A type
// extending the codec overrides it and calls this method for the types it
// doesn't handle. It panics with a dart.FormatException for an unknown
// type.
func (c StandardMessageCodec) ReadValueOfType(typ int, buffer *foundation.ReadBuffer) any {
	self := c.outer()
	switch typ {
	case valueNull:
		return nil
	case valueTrue:
		return true
	case valueFalse:
		return false
	case valueInt32:
		return buffer.GetInt32()
	case valueInt64:
		return buffer.GetInt64()
	case valueFloat64:
		return buffer.GetFloat64()
	case valueLargeInt, valueString:
		length := self.ReadSize(buffer)
		return dart.UTF8Decode(buffer.GetUint8List(length))
	case valueUint8List:
		length := self.ReadSize(buffer)
		return distinctEmpty(buffer.GetUint8List(length))
	case valueInt32List:
		length := self.ReadSize(buffer)
		return distinctEmpty(buffer.GetInt32List(length))
	case valueInt64List:
		length := self.ReadSize(buffer)
		return distinctEmpty(buffer.GetInt64List(length))
	case valueFloat32List:
		length := self.ReadSize(buffer)
		return distinctEmpty(buffer.GetFloat32List(length))
	case valueFloat64List:
		length := self.ReadSize(buffer)
		return distinctEmpty(buffer.GetFloat64List(length))
	case valueList:
		length := self.ReadSize(buffer)
		result := distinctEmpty(make([]any, length))
		for i := range length {
			result[i] = self.ReadValue(buffer)
		}
		return result
	case valueMap:
		length := self.ReadSize(buffer)
		result := &dart.LinkedMap{}
		for range length {
			key := self.ReadValue(buffer)
			result.Set(key, self.ReadValue(buffer))
		}
		return result
	}
	panic(dart.NewFormatException("Message corrupted"))
}

// WriteSize writes a non-negative 32-bit integer to buffer in an expanding
// 1-5 byte encoding that optimizes for small values (for types overriding
// WriteValue).
func (c StandardMessageCodec) WriteSize(buffer *foundation.WriteBuffer, value int) {
	if !(0 <= value && value <= 0xffffffff) {
		panic(dart.NewAssertionError(nil))
	}
	if value < 254 {
		buffer.PutUint8(value)
	} else if value <= 0xffff {
		buffer.PutUint8(254)
		buffer.PutUint16(value)
	} else {
		buffer.PutUint8(255)
		buffer.PutUint32(value)
	}
}

// ReadSize reads a non-negative int from buffer as written by WriteSize
// (for types overriding ReadValueOfType).
func (c StandardMessageCodec) ReadSize(buffer *foundation.ReadBuffer) int {
	value := buffer.GetUint8()
	switch value {
	case 254:
		return buffer.GetUint16()
	case 255:
		return buffer.GetUint32()
	}
	return value
}

// StandardMethodCodec is a MethodCodec using the Flutter standard binary
// encoding: a method call is the method name followed by the arguments; an
// envelope is a byte (0 success, 1 error) followed by the result, or by the
// error code, message and details (and, from the platform, a stack trace).
// Its zero value uses a plain StandardMessageCodec.
type StandardMethodCodec struct {
	// MessageCodec is the message codec this method codec uses for
	// encoding values (nil: a StandardMessageCodec).
	MessageCodec *StandardMessageCodec
}

// NewStandardMethodCodec is StandardMethodCodec([messageCodec]).
func NewStandardMethodCodec(messageCodec ...*StandardMessageCodec) StandardMethodCodec {
	if len(messageCodec) > 0 {
		return StandardMethodCodec{MessageCodec: messageCodec[0]}
	}
	return StandardMethodCodec{}
}

var defaultStandardMessageCodec = &StandardMessageCodec{}

func (m StandardMethodCodec) codec() StandardMessageCodecOverridable {
	if m.MessageCodec == nil {
		return defaultStandardMessageCodec
	}
	return m.MessageCodec.outer()
}

func (m StandardMethodCodec) EncodeMethodCall(methodCall MethodCall) []byte {
	buffer := foundation.NewWriteBuffer(foundation.WriteBufferOptions{StartCapacity: writeBufferStartCapacity})
	m.codec().WriteValue(buffer, methodCall.Method)
	m.codec().WriteValue(buffer, methodCall.Arguments)
	return buffer.Done()
}

// DecodeMethodCall panics if methodCall is nil (Dart's null check).
func (m StandardMethodCodec) DecodeMethodCall(methodCall []byte) MethodCall {
	if methodCall == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	buffer := foundation.NewReadBuffer(methodCall)
	method := m.codec().ReadValue(buffer)
	arguments := m.codec().ReadValue(buffer)
	if name, ok := method.(string); ok && !buffer.HasRemaining() {
		return MethodCall{Method: name, Arguments: arguments}
	}
	panic(dart.NewFormatException("Invalid method call"))
}

func (m StandardMethodCodec) EncodeSuccessEnvelope(result any) []byte {
	buffer := foundation.NewWriteBuffer(foundation.WriteBufferOptions{StartCapacity: writeBufferStartCapacity})
	buffer.PutUint8(0)
	m.codec().WriteValue(buffer, result)
	return buffer.Done()
}

func (m StandardMethodCodec) EncodeErrorEnvelope(code string, opts ...EncodeErrorEnvelopeOptions) []byte {
	o := errorEnvelopeOptions(opts)
	buffer := foundation.NewWriteBuffer(foundation.WriteBufferOptions{StartCapacity: writeBufferStartCapacity})
	buffer.PutUint8(1)
	m.codec().WriteValue(buffer, code)
	m.codec().WriteValue(buffer, o.Message)
	m.codec().WriteValue(buffer, o.Details)
	return buffer.Done()
}

func (m StandardMethodCodec) DecodeEnvelope(envelope []byte) any {
	// First byte is zero in success case, and non-zero otherwise.
	if len(envelope) == 0 {
		panic(dart.NewFormatException("Expected envelope, got nothing"))
	}
	buffer := foundation.NewReadBuffer(envelope)
	if buffer.GetUint8() == 0 {
		return m.codec().ReadValue(buffer)
	}
	errorCode := m.codec().ReadValue(buffer)
	errorMessage := m.codec().ReadValue(buffer)
	errorDetails := m.codec().ReadValue(buffer)
	var errorStacktrace *string
	if buffer.HasRemaining() {
		if s := m.codec().ReadValue(buffer); s != nil {
			str, ok := s.(string)
			if !ok {
				panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(s, "Object") +
					"' is not a subtype of type 'String?' in type cast"})
			}
			errorStacktrace = &str
		}
	}
	if code, ok := errorCode.(string); ok && isNullOrString(errorMessage) && !buffer.HasRemaining() {
		panic(&PlatformException{Code: code, Message: stringPointer(errorMessage), Details: errorDetails,
			Stacktrace: errorStacktrace})
	}
	panic(dart.NewFormatException("Invalid envelope"))
}

// distinctEmpty gives an empty decoded list its own backing array: Dart's
// lists are distinct objects, and a decoded list used as a map key is
// compared by identity (see dart.LinkedMap), which Go's zero-size
// allocations, all at one address, would lose.
func distinctEmpty[E any](list []E) []E {
	if len(list) == 0 {
		return make([]E, 0, 1)
	}
	return list
}

// MapLookup is a lookup in a decoded (or encodable) message map: a
// *dart.LinkedMap, a Go map or an ordered []dart.MapEntry map. ok is false
// when the key is absent or m is not a map. (A Go helper for reading Dart
// map messages; not Flutter API.)
func MapLookup(m any, key any) (value any, ok bool) {
	switch x := m.(type) {
	case *dart.LinkedMap:
		if x == nil {
			return nil, false
		}
		return x.Get(key)
	case []dart.MapEntry[any, any]:
		for _, e := range x {
			if comparableEqual(e.Key, key) {
				return e.Value, true
			}
		}
		return nil, false
	case []dart.MapEntry[string, any]:
		for _, e := range x {
			if comparableEqual(e.Key, key) {
				return e.Value, true
			}
		}
		return nil, false
	}
	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Map {
		return nil, false
	}
	kv := reflect.ValueOf(key)
	if !kv.IsValid() || !kv.Type().AssignableTo(rv.Type().Key()) {
		return nil, false
	}
	v := rv.MapIndex(kv)
	if !v.IsValid() {
		return nil, false
	}
	return v.Interface(), true
}

func comparableEqual(a, b any) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return a == b
}
