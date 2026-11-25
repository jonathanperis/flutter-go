package foundation

// Port of packages/flutter/lib/src/foundation/stack_frame.dart (Flutter
// 3.50.1): StackFrame, a frame parsed from the text of a Dart stack trace.
// The parser is ported as is (its regular expressions are Dart's; Go's
// regexp picks the same submatches as a backtracking engine), so Dart-format
// traces parse exactly as in Flutter. Go stacks captured with
// dart.StackTraceCurrent / dart.StackTraceFromPanic are rendered in that
// format and parse with it too (without column numbers).

import (
	"fmt"
	"hash/maphash"
	"regexp"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

// StackFrame is a single frame of a stack trace. Fields not given are Dart's
// defaults: ClassName "" and IsConstructor false.
type StackFrame struct {
	// Source is the original line the frame was parsed from.
	Source string
	// Number is the frame's index in the trace (-1 when unknown).
	Number int
	// PackageScheme is the URI scheme of the frame's location (dart,
	// package, file...).
	PackageScheme string
	// Package is the package or dart: library name.
	Package string
	// PackagePath is the path of the file within its package.
	PackagePath string
	// Line is the line number (-1 when unknown).
	Line int
	// Column is the column number (-1 when unknown).
	Column int
	// ClassName is the class of the method, "" for a top-level function.
	ClassName string
	// Method is the method or function name.
	Method string
	// IsConstructor says whether the frame is a constructor ("new Foo").
	IsConstructor bool
}

// StackFrameAsynchronousSuspension is StackFrame.asynchronousSuspension: the
// "<asynchronous suspension>" line of async traces.
var StackFrameAsynchronousSuspension = StackFrame{
	Number:        -1,
	Column:        -1,
	Line:          -1,
	Method:        "asynchronous suspension",
	PackageScheme: "",
	Package:       "",
	PackagePath:   "",
	Source:        "<asynchronous suspension>",
}

// StackFrameStackOverFlowElision is StackFrame.stackOverFlowElision: the
// "..." line the VM prints where it elides frames of a stack overflow.
var StackFrameStackOverFlowElision = StackFrame{
	Number:        -1,
	Column:        -1,
	Line:          -1,
	Method:        "...",
	PackageScheme: "",
	Package:       "",
	PackagePath:   "",
	Source:        "...",
}

// StackFrameFromStackTrace is StackFrame.fromStackTrace: the frames of
// stack's text.
func StackFrameFromStackTrace(stack dart.StackTrace) []StackFrame {
	return StackFrameFromStackString(stack.String())
}

// StackFrameFromStackString is StackFrame.fromStackString: parses each
// non-empty line of stack, skipping lines that aren't frames.
func StackFrameFromStackString(stack string) []StackFrame {
	result := []StackFrame{}
	for _, line := range strings.Split(dartTrim(stack), "\n") {
		if line == "" {
			continue
		}
		// On the Web in non-debug builds the stack trace includes the exception
		// message that precedes the stack trace itself. fromStackTraceLine will
		// return null in that case. We will skip it here.
		if frame, ok := StackFrameFromStackTraceLine(line); ok {
			result = append(result, frame)
		}
	}
	return result
}

func tryParseWebFrame(line string) (StackFrame, bool) {
	// dart2wasm doesn't emit stack frames in the same way DDC does, so we need
	// to do the less clever non-debug path here when compiled to wasm.
	if KDebugMode && !KIsWasm {
		return tryParseWebDebugFrame(line)
	}
	return tryParseWebNonDebugFrame(line)
}

var (
	webDebugPackageFramePattern = regexp.MustCompile(jsRegExp(`^(package.+) (\d+):(\d+)\s+(.+)$`))
	webDebugFramePattern        = regexp.MustCompile(jsRegExp(`^(.+) (\d+):(\d+)\s+(.+)$`))
)

func tryParseWebDebugFrame(line string) (StackFrame, bool) {
	// This RegExp is only partially correct for flutter run/test differences.
	// https://github.com/flutter/flutter/issues/52685
	hasPackage := strings.HasPrefix(line, "package")
	parser := webDebugFramePattern
	if hasPackage {
		parser = webDebugPackageFramePattern
	}
	match := parser.FindStringSubmatch(line)
	if match == nil {
		return StackFrame{}, false
	}
	pkg := "<unknown>"
	packageScheme := "<unknown>"
	packagePath := "<unknown>"
	if hasPackage {
		packageScheme = "package"
		packageURI := dart.ParseURI(match[1])
		pkg = packageURI.PathSegments()[0]
		packagePath = strings.Replace(packageURI.Path, packageURI.PathSegments()[0]+"/", "", 1)
	}
	return StackFrame{
		Number:        -1,
		PackageScheme: packageScheme,
		Package:       pkg,
		PackagePath:   packagePath,
		Line:          dartParseInt(match[2]),
		Column:        dartParseInt(match[3]),
		ClassName:     "<unknown>",
		Method:        match[4],
		Source:        line,
	}, true
}

// Non-debug builds do not point to dart code but compiled JavaScript, so
// line numbers are meaningless. We only attempt to parse the class and
// method name, which is more or less readable in profile builds, and
// minified in release builds.
var webNonDebugFramePattern = regexp.MustCompile(jsRegExp(`^\s*at ([^\s]+).*$`))

// tryParseWebNonDebugFrame parses line as a stack frame in profile and
// release Web builds; false if it isn't one.
func tryParseWebNonDebugFrame(line string) (StackFrame, bool) {
	match := webNonDebugFramePattern.FindStringSubmatch(line)
	if match == nil {
		// On the Web in non-debug builds the stack trace includes the exception
		// message that precedes the stack trace itself. Instead of crashing
		// when a line is not recognized as a stack frame, we return null.
		return StackFrame{}, false
	}
	classAndMethod := strings.Split(match[1], ".")
	className := "<unknown>"
	method := classAndMethod[0]
	if len(classAndMethod) > 1 {
		className = classAndMethod[0]
		method = strings.Join(classAndMethod[1:], ".")
	}
	return StackFrame{
		Number:        -1,
		PackageScheme: "<unknown>",
		Package:       "<unknown>",
		PackagePath:   "<unknown>",
		Line:          -1,
		Column:        -1,
		ClassName:     className,
		Method:        method,
		Source:        line,
	}, true
}

const packageStackTraceAsyncGap = "===== asynchronous gap ==========================="

const stackTraceLineSource = `^#(\d+) +(.+) \((.+?):?(\d+){0,1}:?(\d+){0,1}\)$`

var stackTraceLinePattern = regexp.MustCompile(jsRegExp(stackTraceLineSource))

// StackFrameFromStackTraceLine is StackFrame.fromStackTraceLine: parses one
// line of a trace; false (Dart's null) when a line not starting with '#'
// isn't a web frame either. A '#' line that doesn't match the VM's frame
// format panics (Dart asserts).
func StackFrameFromStackTraceLine(line string) (StackFrame, bool) {
	if line == "<asynchronous suspension>" || line == packageStackTraceAsyncGap {
		return StackFrameAsynchronousSuspension, true
	} else if line == "..." {
		return StackFrameStackOverFlowElision, true
	}

	// Web frames.
	if !strings.HasPrefix(line, "#") {
		return tryParseWebFrame(line)
	}

	parser := stackTraceLinePattern
	match := parser.FindStringSubmatchIndex(line)
	if match == nil {
		panic(dart.NewAssertionError("Expected " + line + " to match RegExp/" + stackTraceLineSource + "/."))
	}
	group := func(i int) (string, bool) {
		if match[2*i] < 0 {
			return "", false
		}
		return line[match[2*i]:match[2*i+1]], true
	}

	isConstructor := false
	className := ""
	g2, _ := group(2)
	method := strings.ReplaceAll(g2, ".<anonymous closure>", "")
	if strings.HasPrefix(method, "new") {
		methodParts := strings.Split(method, " ")
		// Sometimes a web frame will only read "new" and have no class name.
		if len(methodParts) > 1 {
			className = strings.Split(method, " ")[1]
		} else {
			className = "<unknown>"
		}
		method = ""
		if strings.Contains(className, ".") {
			parts := strings.Split(className, ".")
			className = parts[0]
			method = parts[1]
		}
		isConstructor = true
	} else if strings.Contains(method, ".") {
		parts := strings.Split(method, ".")
		className = parts[0]
		method = parts[1]
	}

	g3, _ := group(3)
	packageURI := dart.ParseURI(g3)
	pkg := "<unknown>"
	packagePath := packageURI.Path
	if packageURI.Scheme == "dart" || packageURI.Scheme == "package" {
		pkg = packageURI.PathSegments()[0]
		packagePath = strings.Replace(packageURI.Path, packageURI.PathSegments()[0]+"/", "", 1)
	}

	g1, _ := group(1)
	lineNumber, columnNumber := -1, -1
	if g4, ok := group(4); ok {
		lineNumber = dartParseInt(g4)
	}
	if g5, ok := group(5); ok {
		columnNumber = dartParseInt(g5)
	}
	return StackFrame{
		Number:        dartParseInt(g1),
		ClassName:     className,
		Method:        method,
		PackageScheme: packageURI.Scheme,
		Package:       pkg,
		PackagePath:   packagePath,
		Line:          lineNumber,
		Column:        columnNumber,
		IsConstructor: isConstructor,
		Source:        line,
	}, true
}

// Equal is StackFrame's ==: number, package, line, column, className,
// method and source (not packageScheme, packagePath or isConstructor).
func (f StackFrame) Equal(other StackFrame) bool {
	return other.Number == f.Number &&
		other.Package == f.Package &&
		other.Line == f.Line &&
		other.Column == f.Column &&
		other.ClassName == f.ClassName &&
		other.Method == f.Method &&
		other.Source == f.Source
}

var stackFrameHashSeed = maphash.MakeSeed()

// HashCode hashes the fields Equal compares.
func (f StackFrame) HashCode() int {
	type key struct {
		number            int
		pkg               string
		line, column      int
		className, method string
		source            string
	}
	return int(maphash.Comparable(stackFrameHashSeed, key{f.Number, f.Package, f.Line, f.Column, f.ClassName, f.Method, f.Source}) & 0x3FFFFFFF)
}

// String is StackFrame.toString.
func (f StackFrame) String() string {
	return fmt.Sprintf("%s(#%d, %s:%s/%s:%d:%d, className: %s, method: %s)",
		ObjectRuntimeType(f, "StackFrame"), f.Number, f.PackageScheme, f.Package, f.PackagePath,
		f.Line, f.Column, f.ClassName, f.Method)
}

// dartParseInt is int.parse for the decimal digits the parser captures
// (panics on overflow, as Dart throws).
func dartParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic("FormatException: Invalid radix-10 number (at character 1)\n" + s)
	}
	return n
}

// dartTrim is String.trim (Dart's whitespace set).
func dartTrim(s string) string { return strings.TrimFunc(s, isDartWhitespace) }

// jsWhitespace is the set JavaScript's (and so Dart's) \s matches.
const jsWhitespace = `\t\n\v\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`

// jsRegExp rewrites the parts of a Dart (JavaScript) regular expression whose
// meaning differs in Go: \s also matches Unicode spaces, and . doesn't match
// line terminators. It handles the simple patterns of this file only (no
// escaped brackets).
func jsRegExp(pattern string) string {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '\\' && i+1 < len(pattern) && pattern[i+1] == 's':
			if inClass {
				b.WriteString(jsWhitespace)
			} else {
				b.WriteString("[" + jsWhitespace + "]")
			}
			i++
		case c == '\\' && i+1 < len(pattern):
			b.WriteByte(c)
			b.WriteByte(pattern[i+1])
			i++
		case c == '[':
			inClass = true
			b.WriteByte(c)
		case c == ']':
			inClass = false
			b.WriteByte(c)
		case c == '.' && !inClass:
			b.WriteString(`[^\n\r\x{2028}\x{2029}]`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
