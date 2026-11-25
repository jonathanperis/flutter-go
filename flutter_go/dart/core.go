package dart

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// --- print

var printHandler func(line string)

// Print is dart:core's print: writes line to standard output, or hands it
// to the handler installed with SetPrintHandler.
//
// Dart's tests capture print with a Zone (runZoned(zoneSpecification:
// ZoneSpecification(print: ...))); flutter-go has no zones, so the handler is
// global: a timer that fires after the capture ended prints wherever the
// handler current at that time sends it.
func Print(line string) {
	if printHandler != nil {
		printHandler(line)
		return
	}
	fmt.Fprintln(os.Stdout, line)
}

// SetPrintHandler routes Print to handler (nil restores standard output)
// and returns a function that restores the previous handler.
func SetPrintHandler(handler func(line string)) (restore func()) {
	prev := printHandler
	printHandler = handler
	return func() { printHandler = prev }
}

// --- AssertionError

// AssertionFailure is what Dart's `x is AssertionError` tests: satisfied by
// *AssertionError and by types that implement Dart's AssertionError
// interface (Flutter's FlutterError). AssertionMessage is Dart's `message`
// getter.
type AssertionFailure interface {
	error
	AssertionMessage() any
}

// AssertionError is dart:core's AssertionError(message): what a failed
// assert throws.
type AssertionError struct{ message any }

// NewAssertionError is AssertionError([message]); nil is no message.
func NewAssertionError(message any) *AssertionError { return &AssertionError{message} }

// AssertionMessage is the message given to the constructor (nil for none).
func (e *AssertionError) AssertionMessage() any { return e.message }

// String is AssertionError.toString: "Assertion failed" and, when there is
// a message, ": " and Error.safeToString(message).
func (e *AssertionError) String() string {
	if e.message != nil {
		return "Assertion failed: " + SafeToString(e.message)
	}
	return "Assertion failed"
}

func (e *AssertionError) Error() string { return e.String() }

// SafeToString is Error.safeToString: numbers, booleans and null use their
// toString, strings are JSON-quoted, and any other object is
// "Instance of 'Type'" (its own toString is never called).
func SafeToString(object any) string {
	if object == nil {
		return "null"
	}
	v := reflect.ValueOf(object)
	if v.Type().PkgPath() == "" { // builtin types only: a named type is an object
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(v.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return strconv.FormatUint(v.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			return DoubleToString(v.Float())
		case reflect.Bool:
			return strconv.FormatBool(v.Bool())
		case reflect.String:
			return JSONEncodeString(v.String())
		}
	}
	t := v.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := t.Name()
	if name == "" {
		name = t.String()
	}
	if i := strings.IndexByte(name, '['); i >= 0 {
		name = name[:i]
	}
	return "Instance of '" + name + "'"
}

// JSONEncodeString is jsonEncode(string): the string in double quotes with
// Dart's JSON escapes (\b \f \n \r \t, other control characters as \u00xx,
// and \" \\). Unlike Go's encoding/json, <, >, & and U+2028/U+2029 are kept.
func JSONEncodeString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// --- Pattern

// Pattern is dart:core's Pattern as Flutter uses it (does it match anywhere
// in a string?). *regexp.Regexp is a Pattern; StringPattern is a String
// used as a pattern.
type Pattern interface {
	MatchString(s string) bool
}

// StringPattern is a Dart String used as a Pattern: it matches where it
// occurs (the empty string matches everything).
type StringPattern string

// MatchString reports whether the pattern occurs in s
// (pattern.allMatches(s).isNotEmpty).
func (p StringPattern) MatchString(s string) bool { return strings.Contains(s, string(p)) }
