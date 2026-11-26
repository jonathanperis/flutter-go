package dart

import (
	"regexp"
	"strings"
	"testing"
)

func TestSafeToStringAndAssertionError(t *testing.T) {
	type thing struct{}
	type named int
	for _, c := range []struct {
		v    any
		want string
	}{
		{nil, "null"},
		{3, "3"},
		{2.0, "2.0"},
		{true, "true"},
		{"a\"b\\c\n\t\x01</>& é", `"a\"b\\c\n\t\u0001</>&` + " " + `é"`},
		{&thing{}, "Instance of 'thing'"},
		{named(1), "Instance of 'named'"},
	} {
		if got := SafeToString(c.v); got != c.want {
			t.Errorf("SafeToString(%#v) = %q, want %q", c.v, got, c.want)
		}
	}
	if got := NewAssertionError("Test assertion").Error(); got != `Assertion failed: "Test assertion"` {
		t.Errorf("got %q", got)
	}
	if got := NewAssertionError(nil).String(); got != "Assertion failed" {
		t.Errorf("got %q", got)
	}
}

func TestPrintHandler(t *testing.T) {
	var lines []string
	restore := SetPrintHandler(func(l string) { lines = append(lines, l) })
	Print("a")
	inner := SetPrintHandler(func(l string) { lines = append(lines, "inner "+l) })
	Print("b")
	inner()
	Print("c")
	restore()
	if strings.Join(lines, "|") != "a|inner b|c" {
		t.Errorf("got %q", lines)
	}
}

func TestStringPattern(t *testing.T) {
	if !StringPattern("").MatchString("x") || !StringPattern("b").MatchString("abc") || StringPattern("d").MatchString("abc") {
		t.Error("StringPattern")
	}
}

func TestGoFrameMethod(t *testing.T) {
	for in, want := range map[string]string{
		"github.com/x/y/foundation.(*FlutterError).ToString":          "FlutterError.ToString",
		"github.com/x/y/foundation.FlutterError.String":               "FlutterError.String",
		"github.com/x/y/foundation.TestX.func1.2":                     "TestX.<anonymous closure>.<anonymous closure>",
		"github.com/x/y/foundation.(*DiagnosticsProperty[...]).Value": "DiagnosticsProperty.Value",
		"github.com/x/y/foundation.Map[...]":                          "Map",
		"main.main":                                                   "main",
		"main.(*T).M-fm":                                              "T.M",
		"main.run.gowrap1":                                            "run.<anonymous closure>",
		"github.com/a.b/c%2ed.init.0":                                 "init.<anonymous closure>",
		"":                                                            "<unknown>",
	} {
		if got := goFrameMethod(in); got != want {
			t.Errorf("goFrameMethod(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGoFrameLocation(t *testing.T) {
	root := flutterGoRoot()
	for in, want := range map[string]string{
		root + "/packages/flutter/lib/src/foundation/assertions.go": "package:flutter/src/foundation/assertions.go",
		root + "/packages/flutter_test/lib/src/binding.go":          "package:flutter_test/src/binding.go",
		root + "/packages/flutter/lib/foundation.go":                "package:flutter/foundation.go",
		root + "/engine/src/flutter/lib/ui/go/geometry.go":          "dart:ui/geometry.go",
		root + "/src/async/loop.go":                          "dart:async/loop.go",
		root + "/src/dart/core.go":                           "dart:core/core.go",
		root + "/src/app/app.go":                             "file://" + root + "/src/app/app.go",
		"/usr/local/go/src/testing/testing.go":                      "file:///usr/local/go/src/testing/testing.go",
		"/tmp/my app/main.go":                                       "file:///tmp/my%20app/main.go",
	} {
		if got := goFrameLocation(in); got != want {
			t.Errorf("goFrameLocation(%q) = %q, want %q", in, got, want)
		}
	}
}

func helperThatPanics() { panic("boom") }

func TestStackTraceCurrentAndFromPanic(t *testing.T) {
	cur := StackTraceCurrent().String()
	if !regexp.MustCompile(`^#0      TestStackTraceCurrentAndFromPanic \(dart:core/core_test\.go:[0-9]+\)\n#1      tRunner \(file:///`).MatchString(cur) {
		t.Errorf("current:\n%s", cur)
	}
	if strings.Contains(cur, "goexit") {
		t.Errorf("goexit should be dropped:\n%s", cur)
	}
	var stack StackTrace
	func() {
		defer func() {
			recover()
			stack = StackTraceFromPanic()
		}()
		helperThatPanics()
	}()
	s := stack.String()
	if !regexp.MustCompile(`^#0      helperThatPanics \(dart:core/core_test\.go:[0-9]+\)\n#1      TestStackTraceCurrentAndFromPanic\.<anonymous closure> \(dart:core/core_test\.go:[0-9]+\)\n#2      TestStackTraceCurrentAndFromPanic \(`).MatchString(s) {
		t.Errorf("from panic:\n%s", s)
	}
	// Not panicking: the caller's caller.
	var notPanicking StackTrace
	func() { notPanicking = StackTraceFromPanic() }()
	if !strings.HasPrefix(notPanicking.String(), "#0      TestStackTraceCurrentAndFromPanic (") {
		t.Errorf("not panicking:\n%s", notPanicking)
	}
	a, b := StackTraceFromString("x"), StackTraceFromString("x")
	if a == b || a.String() != "x" {
		t.Error("StackTraceFromString objects are distinct and keep their text")
	}
}

func TestParseURI(t *testing.T) {
	for in, want := range map[string]URI{
		"package:flutter/src/x.dart":        {Scheme: "package", Path: "flutter/src/x.dart"},
		"DART:core-patch/errors_patch.dart": {Scheme: "dart", Path: "core-patch/errors_patch.dart"},
		"file:///a/b.dart?q#f":              {Scheme: "file", Path: "/a/b.dart", HasAuthority: true},
		"http://h:1/a/./b/../c.dart":        {Scheme: "http", Path: "/a/c.dart", HasAuthority: true},
		"dart:ui/../ui/hooks.dart":          {Scheme: "dart", Path: "/ui/hooks.dart"},
		"a/../../b":                         {Path: "../b"},
		"file:x y%41%7e%zz":                 {Scheme: "file", Path: "/x%20yA~%25zz"},
		`c\d`:                               {Path: "c/d"},
		"<anonymous>":                       {Path: "%3Canonymous%3E"},
	} {
		if got := ParseURI(in); got != want {
			t.Errorf("ParseURI(%q) = %+v, want %+v", in, got, want)
		}
	}
	if s := ParseURI("package:a%20b/c").PathSegments(); len(s) != 2 || s[0] != "a b" {
		t.Errorf("PathSegments: %q", s)
	}
	if s := ParseURI("dart:").PathSegments(); len(s) != 0 {
		t.Errorf("PathSegments: %q", s)
	}
}
