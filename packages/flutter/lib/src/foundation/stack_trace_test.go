package foundation

// Port of packages/flutter/test/foundation/stack_trace_test.dart (Flutter
// 3.50.1): both tests, on Go stacks rendered in the Dart VM's format.
//
// Differences: frame #0 is the Go test function (Dart's is the test's
// closure, main.<anonymous closure>), and frame #1 is the Go test runner,
// testing.tRunner, where Dart's is package:test_api's Declarer; Go has no
// "<asynchronous suspension>" line to follow it. The async variant runs the
// body as a src/async coroutine (Dart's async test body).

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestFlutterErrorDefaultStackFilter(t *testing.T) {
	filtered := slices.Collect(FlutterErrorDefaultStackFilter(
		slices.Values(strings.Split(dartTrimRight(dart.StackTraceCurrent().String()), "\n"))))
	if len(filtered) < 2 {
		t.Fatalf("too few frames: %q", filtered)
	}
	matches(t, filtered[0], `^#0 +TestFlutterErrorDefaultStackFilter \(.*stack_trace_test\.go:[0-9]+\)$`)
	matches(t, filtered[1], `^#1 +tRunner \(file:///.+/testing/testing\.go:[0-9]+\)$`)
}

func TestFlutterErrorDefaultStackFilterAsyncTestBody(t *testing.T) {
	var filtered []string
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	async.AsyncVoid(func() {
		async.FutureValue(0).Await()
		filtered = slices.Collect(FlutterErrorDefaultStackFilter(
			slices.Values(strings.Split(dartTrimRight(dart.StackTraceCurrent().String()), "\n"))))
	})
	loop.FlushMicrotasks()
	if len(filtered) < 1 {
		t.Fatalf("too few frames: %q", filtered)
	}
	matches(t, filtered[0], `^#0 +TestFlutterErrorDefaultStackFilterAsyncTestBody\.<anonymous closure> \(.*stack_trace_test\.go:[0-9]+\)$`)
	// The coroutine's own frames are src/async's, elided as dart:async.
	if !regexp.MustCompile(`^\(elided (one frame|[0-9]+ frames) from dart:async\)$`).MatchString(filtered[len(filtered)-1]) {
		t.Errorf("last line: %q (all: %q)", filtered[len(filtered)-1], filtered)
	}
}
