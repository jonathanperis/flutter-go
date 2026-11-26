package dart

import (
	"math"
	"reflect"
	"slices"
	"strconv"
	"unicode/utf16"
	"unsafe"
)

// JSONEncode is dart:convert's json.encode (jsonEncode): see JSONUTF8Encode
// for the values it accepts.
func JSONEncode(v any) string { return string(JSONUTF8Encode(v)) }

// JSONUTF8Encode is JsonUtf8Encoder().convert(v) (json.encode's text as
// UTF-8), ported from the SDK's _JsonStringifier. Values: nil, bool,
// strings, Go integers, floats (Dart's double.toString; NaN and infinities
// are unsupported), slices and arrays (typed lists such as []byte are
// lists of numbers, as Dart's Uint8List is a List<int>), maps whose keys
// are all strings at run time (an empty map is "{}" whatever its key
// type), *LinkedMap (Dart's Map, written in its order) and []MapEntry[K, V]
// as ordered maps, a pointer (nil is null,
// otherwise its target), and objects with a `ToJson() any` method (Dart's
// default toEncodable calls toJson). A Go map has no order, so its keys are
// written sorted (Dart writes a map in insertion order; use a *LinkedMap
// or []MapEntry when the order matters). Anything else panics with
// JSONUnsupportedObjectError; a container that contains itself panics with
// JSONCyclicError. Strings are read as WTF-8 (see utf8.go): a lone
// surrogate is written as a \udXXX escape, as Dart does.
func JSONUTF8Encode(v any) []byte {
	e := &jsonEncoder{}
	e.writeObject(v)
	return e.buf
}

// JSONUnsupportedObjectError is dart:convert's JsonUnsupportedObjectError:
// the panic value for a value JSONUTF8Encode can't encode. Cause is the
// error that made it unencodable (Dart: the toJson call's failure).
type JSONUnsupportedObjectError struct {
	Object any
	Cause  any
}

func (e JSONUnsupportedObjectError) Error() string {
	if e.Cause != nil {
		return "Converting object to an encodable object failed: " + jsonSafeToString(e.Object)
	}
	return "Converting object did not return an encodable object: " + jsonSafeToString(e.Object)
}

// jsonSafeToString is Error.safeToString as the VM prints the objects the
// encoder rejects: a map (a Go map or a []MapEntry) is "_Map len:N".
func jsonSafeToString(object any) string {
	if m, ok := object.(*LinkedMap); ok && m != nil {
		return "_Map len:" + strconv.Itoa(m.Len())
	}
	if object != nil {
		rv := reflect.ValueOf(object)
		if rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice && isMapEntryType(rv.Type().Elem()) {
			return "_Map len:" + strconv.Itoa(rv.Len())
		}
	}
	return SafeToString(object)
}

// JSONCyclicError is dart:convert's JsonCyclicError.
type JSONCyclicError struct{ Object any }

func (e JSONCyclicError) Error() string { return "Cyclic error in JSON stringify" }

// noSuchMethodToJSON stands for the NoSuchMethodError Dart's default
// toEncodable throws when an object has no toJson method.
type noSuchMethodToJSON struct{ object any }

func (e noSuchMethodToJSON) Error() string {
	return "NoSuchMethodError: Class '" + reflect.TypeOf(e.object).String() + "' has no instance method 'toJson'."
}

type jsonEncoder struct {
	buf  []byte
	seen []uintptr // containers being written (Dart's _seen)
}

func (e *jsonEncoder) writeObject(object any) {
	if e.writeJSONValue(object) {
		return
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				panic(JSONUnsupportedObjectError{Object: object, Cause: r})
			}
		}()
		var customJSON any
		if t, ok := object.(interface{ ToJson() any }); ok {
			customJSON = t.ToJson()
		} else {
			panic(noSuchMethodToJSON{object})
		}
		if !e.writeJSONValue(customJSON) {
			panic(JSONUnsupportedObjectError{Object: object})
		}
	}()
}

func (e *jsonEncoder) checkCycle(object any, id uintptr) {
	if id != 0 && slices.Contains(e.seen, id) {
		panic(JSONCyclicError{object})
	}
	e.seen = append(e.seen, id)
}

func (e *jsonEncoder) removeSeen() { e.seen = e.seen[:len(e.seen)-1] }

func (e *jsonEncoder) writeJSONValue(object any) bool {
	switch x := object.(type) {
	case nil:
		e.buf = append(e.buf, "null"...)
		return true
	case bool:
		e.buf = strconv.AppendBool(e.buf, x)
		return true
	case string:
		e.buf = append(e.buf, '"')
		e.writeStringContent(x)
		e.buf = append(e.buf, '"')
		return true
	case float64:
		return e.writeDouble(x)
	case float32:
		return e.writeDouble(float64(x))
	case *LinkedMap:
		if x == nil {
			e.buf = append(e.buf, "null"...)
			return true
		}
		e.checkCycle(object, uintptr(unsafe.Pointer(x)))
		ok := e.writeLinkedMap(x)
		e.removeSeen()
		return ok
	}
	rv := reflect.ValueOf(object)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.buf = strconv.AppendInt(e.buf, rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.buf = strconv.AppendUint(e.buf, rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return e.writeDouble(rv.Float())
	case reflect.Bool:
		e.buf = strconv.AppendBool(e.buf, rv.Bool())
	case reflect.String:
		return e.writeJSONValue(rv.String())
	case reflect.Pointer:
		if rv.IsNil() {
			e.buf = append(e.buf, "null"...)
			return true
		}
		return e.writeJSONValue(rv.Elem().Interface())
	case reflect.Slice, reflect.Array:
		var id uintptr
		if rv.Kind() == reflect.Slice && rv.Len() > 0 {
			id = rv.Pointer()
		}
		if isMapEntryType(rv.Type().Elem()) {
			e.checkCycle(object, id)
			ok := e.writeEntries(rv)
			e.removeSeen()
			return ok
		}
		e.checkCycle(object, id)
		e.buf = append(e.buf, '[')
		for i := range rv.Len() {
			if i > 0 {
				e.buf = append(e.buf, ',')
			}
			e.writeObject(rv.Index(i).Interface())
		}
		e.buf = append(e.buf, ']')
		e.removeSeen()
	case reflect.Map:
		var id uintptr
		if rv.Len() > 0 {
			id = rv.Pointer()
		}
		e.checkCycle(object, id)
		ok := e.writeMap(rv)
		e.removeSeen()
		return ok
	default:
		return false
	}
	return true
}

func (e *jsonEncoder) writeDouble(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	e.buf = append(e.buf, DoubleToString(v)...)
	return true
}

// jsonKey returns a map key as a string, if it is one.
func jsonKey(k reflect.Value) (string, bool) {
	if k.Kind() == reflect.Interface {
		if k.IsNil() {
			return "", false
		}
		k = k.Elem()
	}
	if k.Kind() != reflect.String {
		return "", false
	}
	return k.String(), true
}

func (e *jsonEncoder) writeMap(rv reflect.Value) bool {
	if rv.Len() == 0 {
		e.buf = append(e.buf, "{}"...)
		return true
	}
	type entry struct {
		key   string
		value any
	}
	entries := make([]entry, 0, rv.Len())
	for it := rv.MapRange(); it.Next(); {
		k, ok := jsonKey(it.Key())
		if !ok {
			return false
		}
		entries = append(entries, entry{k, it.Value().Interface()})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if a.key < b.key {
			return -1
		} else if a.key > b.key {
			return 1
		}
		return 0
	})
	e.buf = append(e.buf, '{')
	for i, en := range entries {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		e.buf = append(e.buf, '"')
		e.writeStringContent(en.key)
		e.buf = append(e.buf, '"', ':')
		e.writeObject(en.value)
	}
	e.buf = append(e.buf, '}')
	return true
}

// writeLinkedMap writes a *LinkedMap in its order (_JsonStringifier's
// writeMap: every key must be a string).
func (e *jsonEncoder) writeLinkedMap(m *LinkedMap) bool {
	if m.IsEmpty() {
		e.buf = append(e.buf, "{}"...)
		return true
	}
	for k := range m.Keys() {
		if _, ok := k.(string); !ok {
			return false
		}
	}
	e.buf = append(e.buf, '{')
	i := 0
	for k, v := range m.Entries() {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		i++
		e.buf = append(e.buf, '"')
		e.writeStringContent(k.(string))
		e.buf = append(e.buf, '"', ':')
		e.writeObject(v)
	}
	e.buf = append(e.buf, '}')
	return true
}

// writeEntries writes a []MapEntry as a map.
func (e *jsonEncoder) writeEntries(rv reflect.Value) bool {
	if rv.Len() == 0 {
		e.buf = append(e.buf, "{}"...)
		return true
	}
	keys := make([]string, rv.Len())
	for i := range rv.Len() {
		k, ok := jsonKey(rv.Index(i).Field(0))
		if !ok {
			return false
		}
		keys[i] = k
	}
	e.buf = append(e.buf, '{')
	for i, k := range keys {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		e.buf = append(e.buf, '"')
		e.writeStringContent(k)
		e.buf = append(e.buf, '"', ':')
		e.writeObject(rv.Index(i).Field(1).Interface())
	}
	e.buf = append(e.buf, '}')
	return true
}

func jsonHexDigit(x int) byte {
	if x < 10 {
		return byte(48 + x)
	}
	return byte(87 + x)
}

// writeStringContent is _JsonStringifier.writeStringContent over the
// string's UTF-16 code units, written as UTF-8.
func (e *jsonEncoder) writeStringContent(s string) {
	units := utf16Units(s)
	offset := 0
	slice := func(start, end int) {
		e.buf = append(e.buf, string(utf16.Decode(units[start:end]))...)
	}
	for i, u := range units {
		charCode := int(u)
		if charCode > '\\' {
			if charCode >= 0xd800 {
				// Possible surrogate. Check if it is unpaired.
				if (charCode&0xfc00 == 0xd800 && !(i+1 < len(units) && units[i+1]&0xfc00 == 0xdc00)) ||
					(charCode&0xfc00 == 0xdc00 && !(i-1 >= 0 && units[i-1]&0xfc00 == 0xd800)) {
					// Lone surrogate.
					if i > offset {
						slice(offset, i)
					}
					offset = i + 1
					e.buf = append(e.buf, '\\', 'u', 'd', jsonHexDigit((charCode>>8)&0xf),
						jsonHexDigit((charCode>>4)&0xf), jsonHexDigit(charCode&0xf))
				}
			}
			continue
		}
		if charCode < 32 {
			if i > offset {
				slice(offset, i)
			}
			offset = i + 1
			e.buf = append(e.buf, '\\')
			switch charCode {
			case 0x08:
				e.buf = append(e.buf, 'b')
			case 0x09:
				e.buf = append(e.buf, 't')
			case 0x0a:
				e.buf = append(e.buf, 'n')
			case 0x0c:
				e.buf = append(e.buf, 'f')
			case 0x0d:
				e.buf = append(e.buf, 'r')
			default:
				e.buf = append(e.buf, 'u', '0', '0', jsonHexDigit((charCode>>4)&0xf), jsonHexDigit(charCode&0xf))
			}
		} else if charCode == '"' || charCode == '\\' {
			if i > offset {
				slice(offset, i)
			}
			offset = i + 1
			e.buf = append(e.buf, '\\', byte(charCode))
		}
	}
	if offset < len(units) {
		slice(offset, len(units))
	}
}
