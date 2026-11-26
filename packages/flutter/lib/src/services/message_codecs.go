package services

// Partial port of packages/flutter/lib/src/services/message_codec.dart and
// message_codecs.dart (Flutter 3.50.1): MessageCodec and
// StandardMessageCodec, which SystemChannels.accessibility and the
// semantics binding use. The other codecs (binary, string, JSON, method
// codecs) are not ported yet.
//
// Values map to Go as: null nil, bool, int (any Go integer when encoding),
// double float64, String string, Uint8List []byte, Int32List []int32,
// Int64List []int64, Float32List []float32, Float64List []float64, List
// []any (any Go slice of other types when encoding), Map
// []dart.MapEntry[any, any] (decoded maps keep Dart's insertion order;
// encoding also accepts []dart.MapEntry[string, any] and Go maps, whose
// entries are written sorted by key, Go maps having no order).

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// MessageCodec is a message encoding/decoding mechanism: messages are
// encoded to binary (nil for Dart's null ByteData) and decoded back.
type MessageCodec[T any] interface {
	// EncodeMessage encodes the specified message in binary.
	EncodeMessage(message T) []byte
	// DecodeMessage decodes the specified message from binary.
	DecodeMessage(message []byte) T
}

// FormatException is dart:core's FormatException (the message only).
type FormatException struct{ Message string }

func (e FormatException) Error() string { return "FormatException: " + e.Message }

const writeBufferStartCapacity = 64

// StandardMessageCodec is MessageCodec using the Flutter standard binary
// encoding (see the file comment for the Go types). Subclasses that extend
// the encoding (overriding writeValue/readValueOfType) are not supported
// yet.
type StandardMessageCodec struct{}

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

// EncodeMessage encodes message (nil for a nil message).
func (c StandardMessageCodec) EncodeMessage(message any) []byte {
	if message == nil {
		return nil
	}
	buffer := foundation.NewWriteBuffer(foundation.WriteBufferOptions{StartCapacity: writeBufferStartCapacity})
	c.WriteValue(buffer, message)
	return buffer.Done()
}

// DecodeMessage decodes message (nil for a nil message); it panics with a
// FormatException on corrupted input.
func (c StandardMessageCodec) DecodeMessage(message []byte) any {
	if message == nil {
		return nil
	}
	buffer := foundation.NewReadBuffer(message)
	result := c.ReadValue(buffer)
	if buffer.HasRemaining() {
		panic(FormatException{"Message corrupted"})
	}
	return result
}

// WriteValue writes value to buffer by prefixing it with a type byte. It
// panics with Dart's ArgumentError for a value it can't encode.
func (c StandardMessageCodec) WriteValue(buffer *foundation.WriteBuffer, value any) {
	switch v := value.(type) {
	case nil:
		buffer.PutUint8(valueNull)
	case bool:
		if v {
			buffer.PutUint8(valueTrue)
		} else {
			buffer.PutUint8(valueFalse)
		}
	case float64:
		buffer.PutUint8(valueFloat64)
		buffer.PutFloat64(v)
	case float32:
		buffer.PutUint8(valueFloat64)
		buffer.PutFloat64(float64(v))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		rv := reflect.ValueOf(v)
		var n int64
		if rv.CanInt() {
			n = rv.Int()
		} else {
			n = int64(rv.Uint())
		}
		if -0x7fffffff-1 <= n && n <= 0x7fffffff {
			buffer.PutUint8(valueInt32)
			buffer.PutInt32(int(n))
		} else {
			buffer.PutUint8(valueInt64)
			buffer.PutInt64(int(n))
		}
	case string:
		buffer.PutUint8(valueString)
		// A Go string is already UTF-8: the ASCII prefix and the UTF-8 rest
		// Dart writes are its bytes.
		c.WriteSize(buffer, len(v))
		buffer.PutUint8List([]byte(v))
	case []byte:
		buffer.PutUint8(valueUint8List)
		c.WriteSize(buffer, len(v))
		buffer.PutUint8List(v)
	case []int32:
		buffer.PutUint8(valueInt32List)
		c.WriteSize(buffer, len(v))
		buffer.PutInt32List(v)
	case []int64:
		buffer.PutUint8(valueInt64List)
		c.WriteSize(buffer, len(v))
		buffer.PutInt64List(v)
	case []float32:
		buffer.PutUint8(valueFloat32List)
		c.WriteSize(buffer, len(v))
		buffer.PutFloat32List(v)
	case []float64:
		buffer.PutUint8(valueFloat64List)
		c.WriteSize(buffer, len(v))
		buffer.PutFloat64List(v)
	case []dart.MapEntry[any, any]:
		buffer.PutUint8(valueMap)
		c.WriteSize(buffer, len(v))
		for _, e := range v {
			c.WriteValue(buffer, e.Key)
			c.WriteValue(buffer, e.Value)
		}
	case []dart.MapEntry[string, any]:
		buffer.PutUint8(valueMap)
		c.WriteSize(buffer, len(v))
		for _, e := range v {
			c.WriteValue(buffer, e.Key)
			c.WriteValue(buffer, e.Value)
		}
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			buffer.PutUint8(valueList)
			c.WriteSize(buffer, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				c.WriteValue(buffer, rv.Index(i).Interface())
			}
			return
		case reflect.Map:
			buffer.PutUint8(valueMap)
			c.WriteSize(buffer, rv.Len())
			keys := rv.MapKeys()
			sort.Slice(keys, func(i, j int) bool {
				return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
			})
			for _, k := range keys {
				c.WriteValue(buffer, k.Interface())
				c.WriteValue(buffer, rv.MapIndex(k).Interface())
			}
			return
		case reflect.Pointer:
			if rv.IsNil() {
				buffer.PutUint8(valueNull)
				return
			}
			c.WriteValue(buffer, rv.Elem().Interface())
			return
		}
		panic(dart.NewArgumentErrorValue(value, "", ""))
	}
}

// ReadValue reads a value from buffer as written by WriteValue.
func (c StandardMessageCodec) ReadValue(buffer *foundation.ReadBuffer) any {
	if !buffer.HasRemaining() {
		panic(FormatException{"Message corrupted"})
	}
	typ := buffer.GetUint8()
	return c.ReadValueOfType(typ, buffer)
}

// ReadValueOfType reads a value of the indicated type from buffer.
func (c StandardMessageCodec) ReadValueOfType(typ int, buffer *foundation.ReadBuffer) any {
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
		length := c.ReadSize(buffer)
		return string(buffer.GetUint8List(length))
	case valueUint8List:
		length := c.ReadSize(buffer)
		return buffer.GetUint8List(length)
	case valueInt32List:
		length := c.ReadSize(buffer)
		return buffer.GetInt32List(length)
	case valueInt64List:
		length := c.ReadSize(buffer)
		return buffer.GetInt64List(length)
	case valueFloat32List:
		length := c.ReadSize(buffer)
		return buffer.GetFloat32List(length)
	case valueFloat64List:
		length := c.ReadSize(buffer)
		return buffer.GetFloat64List(length)
	case valueList:
		length := c.ReadSize(buffer)
		result := make([]any, length)
		for i := 0; i < length; i++ {
			result[i] = c.ReadValue(buffer)
		}
		return result
	case valueMap:
		length := c.ReadSize(buffer)
		result := []dart.MapEntry[any, any]{}
		for i := 0; i < length; i++ {
			key := c.ReadValue(buffer)
			value := c.ReadValue(buffer)
			result = putMapEntry(result, key, value)
		}
		return result
	}
	panic(FormatException{"Message corrupted"})
}

// putMapEntry is Dart's map[key] = value on an ordered map: an existing
// equal key keeps its position. Keys Go can't compare (slices) are never
// equal, as distinct Dart lists aren't.
func putMapEntry(m []dart.MapEntry[any, any], key, value any) []dart.MapEntry[any, any] {
	for i := range m {
		if comparableEqual(m[i].Key, key) {
			m[i].Value = value
			return m
		}
	}
	return append(m, dart.MapEntry[any, any]{Key: key, Value: value})
}

func comparableEqual(a, b any) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return a == b
}

// WriteSize writes a non-negative 32-bit integer value to buffer using an
// expanding 1-5 byte encoding that optimizes for small values.
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

// ReadSize reads a value written using WriteSize.
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

// MapLookup is a lookup in a decoded (or encodable) message map: an
// ordered []dart.MapEntry map or a Go map. ok is false when the key is
// absent or m is not a map.
func MapLookup(m any, key any) (value any, ok bool) {
	switch x := m.(type) {
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
