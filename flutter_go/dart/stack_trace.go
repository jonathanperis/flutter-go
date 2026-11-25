package dart

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// StackTrace is dart:core's StackTrace: an object whose String() is the
// trace's text. nil is Dart's null.
//
// Flutter parses stack traces as text in the Dart VM's format
// ("#0      Class.method (package:pkg/file.dart:12:3)"), so the Go stacks
// captured by StackTraceCurrent and StackTraceFromPanic are rendered in that
// format too (see goFrameLocation and goFrameMethod for the mapping).
type StackTrace interface {
	String() string
}

type stringStackTrace struct{ s string }

func (t *stringStackTrace) String() string { return t.s }

// StackTraceFromString is StackTrace.fromString: a trace whose String() is s.
// Each call returns a distinct object (Dart compares traces by identity).
func StackTraceFromString(s string) StackTrace { return &stringStackTrace{s} }

// goStackTrace is a captured Go call stack, rendered on first use. The
// first skip logical frames (inlined calls count individually) are omitted.
type goStackTrace struct {
	pcs  []uintptr
	skip int
	once sync.Once
	text string
}

func (t *goStackTrace) String() string {
	t.once.Do(func() { t.text = renderGoStack(t.pcs, t.skip) })
	return t.text
}

func callers(skip int) []uintptr {
	pcs := make([]uintptr, 256)
	for {
		n := runtime.Callers(skip+1, pcs)
		if n < len(pcs) {
			return pcs[:n]
		}
		pcs = make([]uintptr, 2*len(pcs))
	}
}

// StackTraceCurrent is StackTrace.current: the stack of the calling
// goroutine, starting at the function that calls StackTraceCurrent (frame
// #0), as Dart's starts at the function that reads StackTrace.current.
func StackTraceCurrent() StackTrace {
	return &goStackTrace{pcs: callers(2)}
}

// StackTraceFromPanic is the stack a Dart `catch (e, stack)` receives: call
// it in a deferred function after recover() and it returns the stack of the
// panic, starting at the function that panicked (the deferred call and the
// runtime frames that raised the panic are dropped). When the goroutine is
// not panicking it is the stack of StackTraceFromPanic's caller's caller.
func StackTraceFromPanic() StackTrace {
	pcs := callers(2) // from the caller of StackTraceFromPanic
	var funcs []string
	frames := runtime.CallersFrames(pcs)
	for {
		f, more := frames.Next()
		funcs = append(funcs, f.Function)
		if !more {
			break
		}
	}
	start := -1
	for i, fn := range funcs {
		if fn == "runtime.gopanic" {
			start = i + 1
		}
	}
	if start < 0 {
		return &goStackTrace{pcs: pcs, skip: 1}
	}
	// Skip the runtime frames that panicked on the program's behalf
	// (runtime.panicIndex, runtime.sigpanic...).
	for start < len(funcs) && strings.HasPrefix(funcs[start], "runtime.") {
		start++
	}
	return &goStackTrace{pcs: pcs, skip: start}
}

// renderGoStack writes Go frames in the Dart VM's format, one per line:
//
//	#0      FlutterError.ToString (package:flutter/src/foundation/assertions.go:123)
//
// Go has no column numbers, so the location is uri:line. runtime.goexit and
// runtime.main (the goroutine's and the program's roots) are omitted.
func renderGoStack(pcs []uintptr, skip int) string {
	var b strings.Builder
	n := 0
	frames := runtime.CallersFrames(pcs)
	for {
		f, more := frames.Next()
		if skip > 0 {
			skip--
		} else if f.Function != "runtime.goexit" && f.Function != "runtime.main" && (f.Function != "" || f.File != "") {
			fmt.Fprintf(&b, "#%-6d %s (%s:%d)\n", n, goFrameMethod(f.Function), goFrameLocation(f.File), f.Line)
			n++
		}
		if !more {
			break
		}
	}
	return b.String()
}

var (
	moduleRootOnce sync.Once
	moduleRoot     string
)

// flutterGoRoot is the directory of the flutter-go module as source file
// names spell it (an absolute path, or the module path under -trimpath),
// read from this file's own name (…/flutter_go/dart/stack_trace.go).
func flutterGoRoot() string {
	moduleRootOnce.Do(func() {
		file := runtime.FuncForPC(reflect.ValueOf(StackTraceCurrent).Pointer()).Entry()
		f, _ := runtime.CallersFrames([]uintptr{file + 1}).Next()
		moduleRoot = strings.TrimSuffix(f.File, "/flutter_go/dart/stack_trace.go")
	})
	return moduleRoot
}

// splitGoFunction splits a runtime function name
// ("example.com/a/b.(*T).M.func1") into its import path and the rest.
func splitGoFunction(name string) (pkg, rest string) {
	slash := strings.LastIndexByte(name, '/')
	dot := strings.IndexByte(name[slash+1:], '.')
	if dot < 0 {
		return name, ""
	}
	dot += slash + 1
	return name[:dot], name[dot+1:]
}

// goFrameLocation is the URI a Go frame is printed with. Source files of the
// Go packages that stand for Dart libraries (see API_MAPPING.md) get those
// libraries' URIs, so Flutter's stack filters and parsers treat them as
// Dart's:
//
//	<module>/packages/<pkg>/lib/<dir>/<file>.go   → package:<pkg>/<dir>/<file>.go
//	<module>/engine/src/flutter/lib/ui/go/<file>.go → dart:ui/<file>.go
//	<module>/flutter_go/async/<file>.go             → dart:async/<file>.go
//	<module>/flutter_go/dart/<file>.go              → dart:core/<file>.go
//
// Everything else (applications, the Go standard library) is a file: URI of
// the source file, as the Dart VM prints scripts outside a package. The
// file, not the function name, decides: a closure inlined from another
// package is named after its caller but lives in its own package's file.
func goFrameLocation(file string) string {
	slash := strings.LastIndexByte(file, '/')
	base := file[slash+1:]
	if rel, ok := strings.CutPrefix(file[:max(slash, 0)], flutterGoRoot()+"/"); ok && slash > 0 {
		switch {
		case strings.HasPrefix(rel, "packages/"):
			parts := strings.SplitN(strings.TrimPrefix(rel, "packages/"), "/", 3)
			if len(parts) >= 2 && parts[1] == "lib" {
				dir := ""
				if len(parts) == 3 {
					dir = parts[2] + "/"
				}
				return "package:" + parts[0] + "/" + dir + base
			}
		case rel == "engine/src/flutter/lib/ui/go":
			return "dart:ui/" + base
		case rel == "flutter_go/async":
			return "dart:async/" + base
		case rel == "flutter_go/dart":
			return "dart:core/" + base
		}
	}
	return (&url.URL{Scheme: "file", Path: file}).String()
}

var goClosureSegment = regexp.MustCompile(`^(func|gowrap|deferwrap)?[0-9]+$`)

// goFrameMethod is the method a Go frame is printed with, in Dart's
// spelling: the package qualifier is dropped, a method is Type.Method (no
// pointer or type arguments), and closures are "<anonymous closure>":
//
//	foundation.(*FlutterError).ToString → FlutterError.ToString
//	foundation.TestX.func1.2            → TestX.<anonymous closure>.<anonymous closure>
func goFrameMethod(function string) string {
	if function == "" {
		return "<unknown>"
	}
	_, rest := splitGoFunction(function)
	if rest == "" {
		return function
	}
	rest = strings.TrimSuffix(rest, "-fm") // method values
	parts := splitOutsideBrackets(rest)
	for i, p := range parts {
		if i == 0 && strings.HasPrefix(p, "(") {
			p = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(p, "("), "*"), ")")
		}
		if j := strings.IndexByte(p, '['); j > 0 {
			p = p[:j]
		}
		if i > 0 && goClosureSegment.MatchString(p) {
			p = "<anonymous closure>"
		}
		parts[i] = p
	}
	return strings.Join(parts, ".")
}

// splitOutsideBrackets splits s on dots that are not inside [] or ().
func splitOutsideBrackets(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case '.':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}
