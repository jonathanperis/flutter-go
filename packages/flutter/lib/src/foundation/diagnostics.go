package foundation

// Port of packages/flutter/lib/src/foundation/diagnostics.dart (Flutter
// 3.50.1), split over several files because the Dart file is large:
//
//   - diagnostics.go: enums, kNoDefaultValue, shortHash/describeIdentity,
//     describeEnum, and the helpers that reproduce Dart's toString,
//     runtimeType and string semantics;
//   - diagnostics_text_tree.go: TextTreeConfiguration, the named
//     configurations, the prefixed string builder and TextTreeRenderer;
//   - diagnostics_node.go: DiagnosticsNode, JSON serialization,
//     DiagnosticsSerializationDelegate and DiagnosticsBlock;
//   - diagnostics_properties.go: DiagnosticsProperty and its subclasses;
//   - diagnostics_diagnosticable.go: DiagnosticPropertiesBuilder,
//     DiagnosticableNode, DiagnosticableTreeNode, Diagnosticable,
//     DiagnosticableTree and DiagnosticableTreeMixin.
//
// # Overridable methods (Dart virtual dispatch in Go)
//
// Dart classes here override each other's methods (a subclass's
// valueToString, level or showName; a user's debugFillProperties or
// debugDescribeChildren), and the base class's algorithms must call the
// override. Go's embedding doesn't dispatch that way, so every embeddable
// base in this file keeps a pointer to the outer object, `self`, and calls
// every overridable method through it:
//
//   - DiagnosticsNodeBase (and so DiagnosticsProperty, DiagnosticsBlock,
//     DiagnosticableNode...) holds self DiagnosticsNode. The constructors in
//     this package set it. A type in another package that embeds one of them
//     calls SetSelf(outer) once, after building the embedded value:
//
//     p := &ColorProperty{DiagnosticsProperty: *NewDiagnosticsProperty(name, c)}
//     p.SetSelf(p)
//
//   - DiagnosticableMixin, DiagnosticableTreeBase and DiagnosticableTreeMixin
//     hold self Diagnosticable / DiagnosticableTree. A type embedding one calls
//     SetSelf(outer) when it is created; using it before panics with a
//     message saying so. Such types are used by pointer (a copy would still
//     point at the original).
//
// An override calls the "super" implementation through the embedded field:
// `t.DiagnosticableTreeMixin.DebugFillProperties(b)`.
//
// # Nullable values
//
// DiagnosticsProperty[T] holds a T; Dart's null is the nil of a nilable T
// (pointer, interface, slice, map, func). A nullable number, bool or struct
// is a pointer type (`DiagnosticsProperty[*float64]`). Constructors of the
// fixed-type properties accept either form (`NewDoubleProperty("x", 1.5)` or
// `NewDoubleProperty("x", ptr)`). Options follow API_MAPPING: an unset
// option takes the Dart default; Dart's explicit null for an option whose
// default is not null is Null (any-typed options) or NullString (*string
// options).
//
// Dart's assert(() {...}()) blocks, which compute output only when asserts
// are enabled, run when KDebugMode is true (Flutter enables asserts exactly
// in debug builds); assertions that reject bad arguments always panic, as
// everywhere in flutter-go.

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

// debugAssertsEnabled says whether Dart's assert(() {...}()) blocks run.
const debugAssertsEnabled = KDebugMode

// DiagnosticLevel is the priority used to filter which diagnostics are
// shown.
type DiagnosticLevel int

const (
	// DiagnosticLevelHidden: diagnostics that should not be shown.
	DiagnosticLevelHidden DiagnosticLevel = iota
	// DiagnosticLevelFine: low-value diagnostics (e.g. default values).
	DiagnosticLevelFine
	// DiagnosticLevelDebug: shown only when debugging the object closely.
	DiagnosticLevelDebug
	// DiagnosticLevelInfo: interesting diagnostics, typically shown.
	DiagnosticLevelInfo
	// DiagnosticLevelWarning: problematic property values.
	DiagnosticLevelWarning
	// DiagnosticLevelHint: hints about best practices.
	DiagnosticLevelHint
	// DiagnosticLevelSummary: summaries of other diagnostics.
	DiagnosticLevelSummary
	// DiagnosticLevelError: errors or unexpected conditions.
	DiagnosticLevelError
	// DiagnosticLevelOff: filters out every diagnostic.
	DiagnosticLevelOff
)

var diagnosticLevelNames = [...]string{"hidden", "fine", "debug", "info", "warning", "hint", "summary", "error", "off"}

// String is the Dart enum name.
func (l DiagnosticLevel) String() string { return diagnosticLevelNames[l] }

// DiagnosticsTreeStyle is how a node of a DiagnosticsNode tree is displayed.
type DiagnosticsTreeStyle int

const (
	DiagnosticsTreeStyleNone DiagnosticsTreeStyle = iota
	DiagnosticsTreeStyleSparse
	DiagnosticsTreeStyleOffstage
	DiagnosticsTreeStyleDense
	DiagnosticsTreeStyleTransition
	DiagnosticsTreeStyleError
	DiagnosticsTreeStyleWhitespace
	DiagnosticsTreeStyleFlat
	DiagnosticsTreeStyleSingleLine
	DiagnosticsTreeStyleErrorProperty
	DiagnosticsTreeStyleShallow
	DiagnosticsTreeStyleTruncateChildren
)

var diagnosticsTreeStyleNames = [...]string{"none", "sparse", "offstage", "dense", "transition", "error",
	"whitespace", "flat", "singleLine", "errorProperty", "shallow", "truncateChildren"}

// String is the Dart enum name.
func (s DiagnosticsTreeStyle) String() string { return diagnosticsTreeStyleNames[s] }

type noDefaultValue struct{}

func (noDefaultValue) String() string { return "Instance of '_NoDefaultValue'" }

// KNoDefaultValue is kNoDefaultValue: marks a DiagnosticsNode as having no
// default value.
var KNoDefaultValue any = noDefaultValue{}

type nullValue struct{}

func (nullValue) String() string { return "null" }

// Null stands for Dart's null in an any-typed option whose Dart default is
// not null (DefaultValue: Null is Dart's defaultValue: null). Leaving the
// option unset gives the Dart default (kNoDefaultValue for DefaultValue).
var Null any = nullValue{}

// NullString stands for Dart's null in a *string option whose Dart default
// is not null (IterableProperty's IfEmpty: NullString is Dart's
// ifEmpty: null). Compared by identity; never write through it.
var NullString = new(string)

// resolveDefault turns an any-typed option into the stored value: unset
// (nil) is def, Null is Dart's null (nil).
func resolveDefault(v any, def any) any {
	if v == nil {
		return def
	}
	if _, ok := v.(nullValue); ok {
		return nil
	}
	return v
}

// resolveString turns a *string option into the stored value: unset (nil)
// is def, NullString is Dart's null.
func resolveString(v *string, def *string) *string {
	if v == nil {
		return def
	}
	if v == NullString {
		return nil
	}
	return v
}

func strp(s string) *string { return &s }

func boolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func levelOr(p *DiagnosticLevel, def DiagnosticLevel) DiagnosticLevel {
	if p == nil {
		return def
	}
	return *p
}

func styleOr(p *DiagnosticsTreeStyle, def DiagnosticsTreeStyle) DiagnosticsTreeStyle {
	if p == nil {
		return def
	}
	return *p
}

func intOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

var (
	identityMu     sync.Mutex
	identityHashes = map[uintptr]int{}
	identityNext   = 0
)

// identityHashCode stands in for Dart's identityHashCode: a stable,
// arbitrary-looking number per object (pointer-like values; other values
// hash by content).
func identityHashCode(object any) int {
	v := reflect.ValueOf(object)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		identityMu.Lock()
		defer identityMu.Unlock()
		p := v.Pointer()
		h, ok := identityHashes[p]
		if !ok {
			identityNext++
			h = (identityNext * 0x9E3779B1) & 0x3FFFFFFF // spread consecutive ids
			identityHashes[p] = h
		}
		return h
	}
	return int(hashString(fmt.Sprintf("%#v", object)))
}

func hashString(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h & 0x3FFFFFFF
}

// ShortHash is shortHash: five hex digits identifying the object.
func ShortHash(object any) string {
	return fmt.Sprintf("%05x", identityHashCode(object)&0xFFFFF)
}

// DescribeIdentity is describeIdentity: the runtime type and short hash.
func DescribeIdentity(object any) string {
	return ObjectRuntimeType(object, "<optimized out>") + "#" + ShortHash(object)
}

// DescribeEnum is describeEnum (deprecated in Flutter): the name of an enum
// value. A Go enum (an integer type with a String method, per API_MAPPING)
// returns its String(); anything else has its toString stripped up to the
// first dot, and panics if there is none.
func DescribeEnum(enumEntry any) string {
	if isEnum(enumEntry) {
		return enumEntry.(fmt.Stringer).String()
	}
	description := dartToString(enumEntry)
	indexOfDot := strings.IndexByte(description, '.')
	if indexOfDot == -1 || indexOfDot >= len(description)-1 {
		panic(`The provided object "` + description + `" is not an enum.`)
	}
	return description[indexOfDot+1:]
}

// isNull reports whether v is Dart's null: nil, or the nil of a nilable
// type.
func isNull(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return rv.IsNil()
	}
	return false
}

// deref follows non-nil pointers to non-struct values (the nullable-value
// convention: *float64 is double?) and returns Dart's null as nil.
func deref(v any) any {
	if isNull(v) {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer && rv.Elem().Kind() != reflect.Struct {
		rv = rv.Elem()
		if (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) && rv.IsNil() {
			return nil
		}
	}
	return rv.Interface()
}

// dartToString is Dart's `object.toString()` for a Go value: null for nil,
// String() for Stringers, Dart number formatting, and Dart's defaults for
// collections, closures and plain objects.
func dartToString(v any) string {
	if isNull(v) {
		return "null"
	}
	switch x := v.(type) {
	case time.Duration:
		return dartDurationString(x)
	case fmt.Stringer:
		if isEnum(v) {
			// Dart's Enum.toString is "Type.name"; a Go enum's String() is the name.
			return dartTypeName(reflect.TypeOf(v)) + "." + x.String()
		}
		return x.String()
	case error:
		return x.Error()
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return dart.DoubleToString(x)
	case float32:
		return dart.DoubleToString(float64(x))
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return dart.DoubleToString(rv.Float())
	case reflect.String:
		return rv.String()
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.Pointer:
		return dartToString(rv.Elem().Interface())
	case reflect.Slice, reflect.Array:
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = dartToString(rv.Index(i).Interface())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case reflect.Map:
		keys := rv.MapKeys()
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = dartToString(k.Interface()) + ": " + dartToString(rv.MapIndex(k).Interface())
		}
		sort.Strings(parts) // Go maps have no order; sort for stable output
		return "{" + strings.Join(parts, ", ") + "}"
	case reflect.Func:
		closure := "Closure: " + dartFunctionType(rv.Type())
		if name, ok := staticFunctionName(rv); ok {
			// Dart names top-level functions: "Closure: () => void from Function 'f': static."
			closure += " from Function '" + name + "': static."
		}
		return closure
	}
	return "Instance of '" + dartRuntimeType(v) + "'"
}

// isEnum reports whether v is a Go enum per API_MAPPING: a named integer
// type with a String method (time.Duration excepted).
func isEnum(v any) bool {
	if _, ok := v.(fmt.Stringer); !ok {
		return false
	}
	if _, ok := v.(time.Duration); ok {
		return false
	}
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return t.PkgPath() != ""
	}
	return false
}

// dartDurationString is Dart's Duration.toString ("1:02:03.000004").
func dartDurationString(d time.Duration) string {
	us := d.Microseconds()
	sign := ""
	if us < 0 {
		sign = "-"
		us = -us
	}
	hours := us / 3600000000
	us %= 3600000000
	minutes := us / 60000000
	us %= 60000000
	seconds := us / 1000000
	us %= 1000000
	return fmt.Sprintf("%s%d:%02d:%02d.%06d", sign, hours, minutes, seconds, us)
}

// staticFunctionName returns the name of a package-level function (Dart's
// top-level function), or false for closures and method values.
func staticFunctionName(fn reflect.Value) (string, bool) {
	f := runtime.FuncForPC(fn.Pointer())
	if f == nil {
		return "", false
	}
	full := f.Name()
	if i := strings.LastIndexByte(full, '/'); i >= 0 {
		full = full[i+1:]
	}
	parts := strings.Split(full, ".")
	if len(parts) != 2 || strings.HasPrefix(parts[1], "func") || strings.Contains(parts[1], "-") {
		return "", false
	}
	return parts[1], true
}

// dartFunctionType writes a Go func type as Dart writes a function type:
// func(int) string is "(int) => String".
func dartFunctionType(t reflect.Type) string {
	params := make([]string, t.NumIn())
	for i := range params {
		params[i] = dartTypeName(t.In(i))
	}
	result := "void"
	if t.NumOut() == 1 {
		result = dartTypeName(t.Out(0))
	} else if t.NumOut() > 1 {
		outs := make([]string, t.NumOut())
		for i := range outs {
			outs[i] = dartTypeName(t.Out(i))
		}
		result = "(" + strings.Join(outs, ", ") + ")"
	}
	return "(" + strings.Join(params, ", ") + ") => " + result
}

// dartRuntimeType is Dart's `object.runtimeType.toString()` for a Go value.
func dartRuntimeType(v any) string {
	if v == nil {
		return "Null"
	}
	return dartTypeName(reflect.TypeOf(v))
}

// dartTypeName writes a Go type as Dart would print the corresponding type:
// pointers are dropped (nullability), generic arguments use <>, builtin
// types take their Dart names.
func dartTypeName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() == "" || t.PkgPath() == "" {
		return dartTypeString(t.String())
	}
	return dartTypeString(t.Name())
}

// dartTypeString converts the textual form of a Go type (as reflect prints
// it, with package paths) to Dart's spelling.
func dartTypeString(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasPrefix(s, "*") {
		s = s[1:]
	}
	switch s {
	case "float64", "float32":
		return "double"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return "int"
	case "string":
		return "String"
	case "bool":
		return "bool"
	case "interface {}", "any":
		return "Object"
	case "struct {}":
		return "void"
	}
	if strings.HasPrefix(s, "func(") {
		return "Function"
	}
	if strings.HasPrefix(s, "[]") {
		elem := s[2:]
		if strings.Contains(elem, "MapEntry[") {
			if args := splitTypeArgs(elem); len(args) == 2 {
				return "Map<" + dartTypeString(args[0]) + ", " + dartTypeString(args[1]) + "?>"
			}
		}
		return "List<" + dartTypeString(elem) + ">"
	}
	if strings.HasPrefix(s, "map[") {
		depth, i := 0, 3
		for ; i < len(s); i++ {
			if s[i] == '[' {
				depth++
			} else if s[i] == ']' {
				depth--
				if depth == 0 {
					break
				}
			}
		}
		return "Map<" + dartTypeString(s[4:i]) + ", " + dartTypeString(s[i+1:]) + ">"
	}
	head, args := s, ""
	if i := strings.IndexByte(s, '['); i >= 0 && strings.HasSuffix(s, "]") {
		head, args = s[:i], s[i+1:len(s)-1]
	}
	if i := strings.LastIndexByte(head, '/'); i >= 0 {
		head = head[i+1:]
	}
	if i := strings.IndexByte(head, '.'); i >= 0 {
		head = head[i+1:]
	}
	if args == "" {
		return head
	}
	parts := splitTypeArgs(s)
	for i, p := range parts {
		parts[i] = dartTypeString(p)
	}
	return head + "<" + strings.Join(parts, ", ") + ">"
}

// splitTypeArgs returns the top-level type arguments of a generic type
// string ("A[x,y[z]]" → x, y[z]).
func splitTypeArgs(s string) []string {
	i := strings.IndexByte(s, '[')
	if i < 0 || !strings.HasSuffix(s, "]") {
		return nil
	}
	inner := s[i+1 : len(s)-1]
	var parts []string
	depth, start := 0, 0
	for j := 0; j < len(inner); j++ {
		switch inner[j] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, inner[start:j])
				start = j + 1
			}
		}
	}
	return append(parts, inner[start:])
}

// dartEquals is Dart's == for the values diagnostics compare (a property
// value against its default): nulls are equal, numbers compare by value
// across int and double (1 == 1.0), a pointer equals the value it points at
// (the nullable-value convention), and other values use Go's ==.
func dartEquals(a, b any) bool {
	an, bn := isNull(a), isNull(b)
	if an || bn {
		return an && bn
	}
	if equalComparable(a, b) {
		return true
	}
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.Kind() == reflect.Pointer && rb.Kind() == reflect.Pointer {
		if ra.Type() == rb.Type() {
			return false // same pointer type: identity, already compared
		}
	}
	if ra.Kind() == reflect.Pointer {
		return dartEquals(ra.Elem().Interface(), b)
	}
	if rb.Kind() == reflect.Pointer {
		return dartEquals(a, rb.Elem().Interface())
	}
	if fa, ok := asNumber(ra); ok {
		if fb, ok := asNumber(rb); ok {
			return fa == fb
		}
	}
	return false
}

func equalComparable(a, b any) bool {
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb || !ta.Comparable() {
		return false
	}
	defer func() { _ = recover() }() // interface fields holding uncomparable values
	return a == b
}

func asNumber(v reflect.Value) (float64, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	}
	return 0, false
}

// jsonNumber returns v as a JSON number if it is a Dart num (following the
// nullable pointer), with Dart's workaround: non-finite doubles become
// their toString.
func jsonNumber(v any) (any, bool) {
	d := deref(v)
	if d == nil {
		return nil, false
	}
	rv := reflect.ValueOf(d)
	if rv.Type().PkgPath() != "" {
		return nil, false // a named type (an enum, a Duration...) is not a Dart num
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint(), true
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return dart.DoubleToString(f), true
		}
		return f, true
	}
	return nil, false
}

// --- Dart String semantics: lengths and indexes are in UTF-16 code units.

func len16(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

func toUTF16(s string) []uint16 { return utf16.Encode([]rune(s)) }

func fromUTF16(u []uint16) string { return string(utf16.Decode(u)) }

// isDartWhitespace is the set String.trim/trimRight remove.
func isDartWhitespace(r rune) bool {
	switch {
	case r >= 9 && r <= 13, r == 32, r == 0x85, r == 0xA0, r == 0x1680,
		r >= 0x2000 && r <= 0x200A, r == 0x2028, r == 0x2029, r == 0x202F,
		r == 0x205F, r == 0x3000, r == 0xFEFF:
		return true
	}
	return false
}

func dartTrimRight(s string) string { return strings.TrimRightFunc(s, isDartWhitespace) }

// dartToUpperCase is String.toUpperCase. The Dart VM maps each character
// on its own (simple case mapping: "ß" stays "ß", checked against Flutter in
// the difftest), as Go's strings.ToUpper does.
func dartToUpperCase(s string) string { return strings.ToUpper(s) }
