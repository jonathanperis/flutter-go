package dart

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Exception is dart:core's Exception(message) (the VM's _Exception): its
// toString is "Exception: message", or "Exception" when the message is
// null (a nil Message).
type Exception struct{ Message any }

// NewException is Exception([message]).
func NewException(message ...any) Exception {
	if len(message) > 0 {
		return Exception{message[0]}
	}
	return Exception{}
}

func (e Exception) Error() string {
	if e.Message == nil {
		return "Exception"
	}
	return "Exception: " + ToString(e.Message)
}

// TypeError is dart:core's TypeError, which the VM throws for a failed
// cast or null check; Message is its toString ("type 'int' is not a
// subtype of type 'String?' in type cast", "Null check operator used on a
// null value").
type TypeError struct{ Message string }

func (e TypeError) Error() string { return e.Message }

// ToString is Dart's toString of a value as flutter-go represents it (string
// interpolation, "$value"): nil is "null", strings are themselves, numbers
// print as Dart's int and double do (a Go float is a double), slices and
// arrays as Dart lists ("[1, 2]"), maps as Dart maps ("{a: 1}"; a Go map's
// entries sorted by their text, a []MapEntry in its order), errors through
// Error() and Stringers through String(); any other value is
// "Instance of 'Type'".
func ToString(v any) string {
	var b strings.Builder
	writeToString(&b, v)
	return b.String()
}

func writeToString(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
		return
	case string:
		b.WriteString(x)
		return
	case error:
		b.WriteString(x.Error())
		return
	case fmt.Stringer:
		b.WriteString(x.String())
		return
	case float32:
		b.WriteString(DoubleToString(float64(x)))
		return
	case float64:
		b.WriteString(DoubleToString(x))
		return
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		b.WriteString(strconv.FormatBool(rv.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		b.WriteString(strconv.FormatInt(rv.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		b.WriteString(strconv.FormatUint(rv.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		b.WriteString(DoubleToString(rv.Float()))
	case reflect.String:
		b.WriteString(rv.String())
	case reflect.Slice, reflect.Array:
		if isMapEntryType(rv.Type().Elem()) {
			b.WriteByte('{')
			for i := range rv.Len() {
				if i > 0 {
					b.WriteString(", ")
				}
				e := rv.Index(i)
				writeToString(b, e.Field(0).Interface())
				b.WriteString(": ")
				writeToString(b, e.Field(1).Interface())
			}
			b.WriteByte('}')
			return
		}
		b.WriteByte('[')
		for i := range rv.Len() {
			if i > 0 {
				b.WriteString(", ")
			}
			writeToString(b, rv.Index(i).Interface())
		}
		b.WriteByte(']')
	case reflect.Map:
		type entry struct{ k, v string }
		entries := make([]entry, 0, rv.Len())
		for it := rv.MapRange(); it.Next(); {
			entries = append(entries, entry{ToString(it.Key().Interface()), ToString(it.Value().Interface())})
		}
		slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(a.k, b.k) })
		b.WriteByte('{')
		for i, e := range entries {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(e.k + ": " + e.v)
		}
		b.WriteByte('}')
	case reflect.Pointer:
		if rv.IsNil() {
			b.WriteString("null")
			return
		}
		writeToString(b, rv.Elem().Interface())
	default:
		b.WriteString(SafeToString(v))
	}
}

var mapEntryPkg = reflect.TypeFor[MapEntry[string, any]]().PkgPath()

// isMapEntryType tells whether t is an instantiation of MapEntry.
func isMapEntryType(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && t.PkgPath() == mapEntryPkg && strings.HasPrefix(t.Name(), "MapEntry[")
}

// FormatException is dart:core's FormatException([message, source,
// offset]). Source is the input being processed (a string, a byte slice...
// or nil) and HasOffset tells whether Offset is set (Dart's int? offset).
type FormatException struct {
	Message   string
	Source    any
	Offset    int
	HasOffset bool
}

// NewFormatException is FormatException([message = "", source, offset]):
// pass an int offset as the third argument to set it.
func NewFormatException(message string, sourceAndOffset ...any) FormatException {
	e := FormatException{Message: message}
	if len(sourceAndOffset) > 0 {
		e.Source = sourceAndOffset[0]
	}
	if len(sourceAndOffset) > 1 && sourceAndOffset[1] != nil {
		e.Offset = sourceAndOffset[1].(int)
		e.HasOffset = true
	}
	return e
}

// Error is FormatException.toString. Offsets into a string source count
// UTF-16 code units, as in Dart.
func (e FormatException) Error() string {
	report := "FormatException"
	if e.Message != "" {
		report += ": " + e.Message
	}
	src, isString := e.Source.(string)
	if !isString {
		if e.HasOffset {
			report += " (at offset " + strconv.Itoa(e.Offset) + ")"
		}
		return report
	}
	source := utf16.Encode([]rune(src))
	offset, hasOffset := e.Offset, e.HasOffset
	if hasOffset && (offset < 0 || offset > len(source)) {
		hasOffset = false
	}
	if !hasOffset {
		if len(source) > 78 {
			return report + "\n" + string(utf16.Decode(source[:75])) + "..."
		}
		return report + "\n" + src
	}
	lineNum := 1
	lineStart := 0
	previousCharWasCR := false
	for i := 0; i < offset; i++ {
		switch source[i] {
		case 0x0a:
			if lineStart != i || !previousCharWasCR {
				lineNum++
			}
			lineStart = i + 1
			previousCharWasCR = false
		case 0x0d:
			lineNum++
			lineStart = i + 1
			previousCharWasCR = true
		}
	}
	if lineNum > 1 {
		report += " (at line " + strconv.Itoa(lineNum) + ", character " + strconv.Itoa(offset-lineStart+1) + ")\n"
	} else {
		report += " (at character " + strconv.Itoa(offset+1) + ")\n"
	}
	lineEnd := len(source)
	for i := offset; i < len(source); i++ {
		if source[i] == 0x0a || source[i] == 0x0d {
			lineEnd = i
			break
		}
	}
	length := lineEnd - lineStart
	start, end := lineStart, lineEnd
	prefix, postfix := "", ""
	if length > 78 {
		// Can't show entire line. Try to anchor at the nearest end, if one is
		// within reach.
		index := offset - lineStart
		if index < 75 {
			end = start + 75
			postfix = "..."
		} else if end-offset < 75 {
			start = end - 75
			prefix = "..."
		} else {
			// Neither end is near, just pick an area around the offset.
			start = offset - 36
			end = offset + 36
			prefix, postfix = "...", "..."
		}
	}
	slice := string(utf16.Decode(source[start:end]))
	markOffset := offset - start + len(prefix)
	return report + prefix + slice + postfix + "\n" + strings.Repeat(" ", markOffset) + "^\n"
}
