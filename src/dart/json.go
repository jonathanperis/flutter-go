package dart

import (
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// JSONEncode is dart:convert's json.encode for the values Flutter encodes:
// nil, bool, strings, Go integers, floats (Dart's double.toString), slices,
// maps with string keys and []MapEntry[string, any] (an ordered map). A Go
// map has no order, so its keys are written sorted (Dart writes a map in
// insertion order; use []MapEntry when the order matters). It panics, as
// Dart's JsonUnsupportedObjectError, on other values and on NaN or infinite
// doubles.
func JSONEncode(v any) string {
	var b strings.Builder
	writeJSON(&b, v)
	return b.String()
}

// JSONUnsupportedObjectError is the panic value of JSONEncode for a value
// it can't encode.
type JSONUnsupportedObjectError struct{ Object any }

func (e JSONUnsupportedObjectError) Error() string {
	return "Converting object to an encodable object failed: " + SafeToString(e.Object)
}

func writeJSON(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
		return
	case bool:
		b.WriteString(strconv.FormatBool(x))
		return
	case string:
		b.WriteString(JSONEncodeString(x))
		return
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			panic(JSONUnsupportedObjectError{x})
		}
		b.WriteString(DoubleToString(x))
		return
	case float32:
		writeJSON(b, float64(x))
		return
	case []MapEntry[string, any]:
		b.WriteByte('{')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(JSONEncodeString(e.Key))
			b.WriteByte(':')
			writeJSON(b, e.Value)
		}
		b.WriteByte('}')
		return
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b.WriteString(strconv.FormatInt(rv.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		b.WriteString(strconv.FormatUint(rv.Uint(), 10))
	case reflect.Slice, reflect.Array:
		b.WriteByte('[')
		for i := range rv.Len() {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJSON(b, rv.Index(i).Interface())
		}
		b.WriteByte(']')
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			panic(JSONUnsupportedObjectError{v})
		}
		keys := make([]string, 0, rv.Len())
		for _, k := range rv.MapKeys() {
			keys = append(keys, k.String())
		}
		slices.Sort(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(JSONEncodeString(k))
			b.WriteByte(':')
			writeJSON(b, rv.MapIndex(reflect.ValueOf(k).Convert(rv.Type().Key())).Interface())
		}
		b.WriteByte('}')
	default:
		panic(JSONUnsupportedObjectError{v})
	}
}
