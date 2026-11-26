package foundation

// Port of packages/flutter/test/foundation/print_test.dart (Flutter 3.50.1)
// and its capture_output.dart helper. All three tests are ported; the tests
// after them are flutter-go's own (debugWordWrap and the throttle's limits,
// which Flutter only covers through debugPrint).
//
// Differences: captureOutput installs a dart.SetPrintHandler handler instead
// of a Zone print override. Zones bind a timer's callback to the zone that
// created it; the global handler doesn't, so the throttling test keeps one
// handler for its whole run and switches the log it appends to, which gives
// the same lines in the same logs. FakeAsync is an async.Loop made current.

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// captureOutput is capture_output.dart's captureOutput: the lines fn prints.
func captureOutput(fn func()) []string {
	log := []string{}
	restore := dart.SetPrintHandler(func(line string) { log = append(log, line) })
	defer restore()
	fn()
	return log
}

func wrap(w int) DebugPrintOptions { return DebugPrintOptions{WrapWidth: &w} }

// withLoop runs fn with a fresh async loop current (FakeAsync().run), then
// flushes the throttle's pending output so later tests start clean.
func withLoop(t *testing.T, fn func(loop *async.Loop)) {
	t.Helper()
	loop := async.NewLoop()
	restore := async.SetCurrent(loop)
	defer restore()
	fn(loop)
	discard := dart.SetPrintHandler(func(string) {})
	loop.FlushTimers(time.Hour, true)
	discard()
}

func TestDebugPrint(t *testing.T) {
	withLoop(t, func(*async.Loop) {
		deepEqual(t, captureOutput(func() { DebugPrintSynchronously("Hello, world") }), []string{"Hello, world"})
		deepEqual(t, captureOutput(func() { DebugPrintSynchronously("Hello, world", wrap(10)) }), []string{"Hello,\nworld"})
		for i := 0; i < 14; i++ {
			deepEqual(t, captureOutput(func() { DebugPrintSynchronously("Hello,   world", wrap(i)) }), []string{"Hello,\nworld"})
		}
		deepEqual(t, captureOutput(func() { DebugPrintThrottled("Hello, world") }), []string{"Hello, world"})
		deepEqual(t, captureOutput(func() { DebugPrintThrottled("Hello, world", wrap(10)) }), []string{"Hello,", "world"})
	})
}

func TestDebugPrintThrottling(t *testing.T) {
	withLoop(t, func(loop *async.Loop) {
		var log *[]string
		restore := dart.SetPrintHandler(func(line string) { *log = append(*log, line) })
		defer restore()
		capture := func(fn func()) *[]string {
			l := []string{}
			log = &l
			fn()
			return &l
		}

		l := capture(func() { DebugPrintThrottled(strings.Repeat("A", 22*1024) + "\nB") })
		equal(t, len(*l), 1)
		loop.Elapse(2 * time.Second)
		equal(t, len(*l), 2)

		l = capture(func() {
			DebugPrintThrottled(strings.Repeat("C", 22*1024))
			DebugPrintThrottled("D")
		})
		equal(t, len(*l), 1)
		loop.Elapse(2 * time.Second)
		equal(t, len(*l), 2)
	})
}

func TestDebugPrintCanPrintNull(t *testing.T) {
	// Dart's null message is "null" in Go (see print.go).
	withLoop(t, func(*async.Loop) {
		deepEqual(t, captureOutput(func() { DebugPrintThrottled("null") }), []string{"null"})
		deepEqual(t, captureOutput(func() { DebugPrintThrottled("null", wrap(80)) }), []string{"null"})
	})
}

// --- flutter-go's own tests

func wordWrap(message string, width int, opts ...DebugWordWrapOptions) []string {
	return slices.Collect(DebugWordWrap(message, width, opts...))
}

func TestDebugWordWrap(t *testing.T) {
	deepEqual(t, wordWrap("short", 10), []string{"short"})
	deepEqual(t, wordWrap("exactly ten", 11), []string{"exactly ten"}) // length 11 == width: wrapping runs, one line
	deepEqual(t, wordWrap("aaa bbb ccc ddd", 7), []string{"aaa bbb", "ccc ddd"})
	deepEqual(t, wordWrap("aaa bbb ccc ddd", 8), []string{"aaa bbb", "ccc ddd"})
	deepEqual(t, wordWrap("aaa bbb ccc ddd", 6), []string{"aaa", "bbb", "ccc", "ddd"})
	// Indentation and list markers are repeated as spaces.
	deepEqual(t, wordWrap("  - item one two three", 12), []string{"  - item one", "    two", "    three"})
	deepEqual(t, wordWrap("12) alpha beta gamma", 10), []string{"12) alpha", "    beta", "    gamma"})
	deepEqual(t, wordWrap("* x y z w", 4), []string{"* x", "  y", "  z", "  w"})
	deepEqual(t, wordWrap("aaa bbb ccc", 5, DebugWordWrapOptions{WrapIndent: ">>"}), []string{"aaa", ">>bbb", ">>ccc"})
	// Stack frames are never wrapped, even indented.
	frame := "   #0      main (file:///a.dart:1:2) with more words"
	deepEqual(t, wordWrap(frame, 10), []string{frame})
	// An unbreakable word stays whole.
	deepEqual(t, wordWrap("abcdefghijkl mn", 5), []string{"abcdefghijkl", "mn"})
	// Widths count UTF-16 code units.
	deepEqual(t, wordWrap("😀😀 😀😀 😀", 5), []string{"😀😀", "😀😀", "😀"})
	deepEqual(t, wordWrap("ééé ééé", 7), []string{"ééé ééé"})
	// Several spaces between words are eaten at a break.
	deepEqual(t, wordWrap("one     two", 5), []string{"one", "two"})
	// A blank message as long as the width has no first character (Dart's
	// RangeError).
	if r := catchPanic(func() { wordWrap("    ", 3) }); r == nil {
		t.Errorf("debugWordWrap of a blank line should panic")
	}
	// ...also when exactly as long as the width (the fast path is length < width).
	if r := catchPanic(func() { wordWrap("   ", 3) }); r == nil {
		t.Errorf("debugWordWrap of a blank line of the width should panic")
	}
	// The fast path compares UTF-16 lengths: two NBSPs (4 UTF-8 bytes) are
	// shorter than 3 and returned as is.
	deepEqual(t, wordWrap("\u00a0\u00a0", 3), []string{"\u00a0\u00a0"})
}

func TestDebugPrintThrottledCapacity(t *testing.T) {
	withLoop(t, func(loop *async.Loop) {
		var lines []string
		restore := dart.SetPrintHandler(func(line string) { lines = append(lines, line) })
		defer restore()
		// Up to the capacity (12 KiB) prints, and the next line waits.
		idleThrottle()
		DebugPrintThrottled(strings.Repeat("x", 12*1024-1) + "\ny\nz")
		equal(t, len(lines), 2)
		done := false
		async.OnComplete(DebugPrintDone(), func(async.Void) { done = true }, nil)
		loop.Elapse(999 * time.Millisecond)
		equal(t, len(lines), 2)
		equal(t, done, false)
		loop.Elapse(time.Millisecond)
		equal(t, len(lines), 3)
		loop.FlushMicrotasks()
		equal(t, done, true)
		// Characters are UTF-16 code units: an emoji counts 2, so 6144 fill
		// the capacity, and "é" counts 1.
		lines = nil
		idleThrottle()
		DebugPrintThrottled(strings.Repeat("😀", 6144) + "\nb")
		equal(t, len(lines), 1)
		loop.Elapse(time.Second)
		equal(t, len(lines), 2)
		lines = nil
		idleThrottle()
		DebugPrintThrottled(strings.Repeat("é", 7000) + "\nb")
		equal(t, len(lines), 2)
		// After more than a second without printing, the count restarts.
		lines = nil
		idleThrottle()
		DebugPrintThrottled(strings.Repeat("x", 12*1024))
		debugPrintStopwatch.total += 1001 * time.Millisecond
		DebugPrintThrottled("y")
		equal(t, len(lines), 2)
		// ...but not after exactly a second.
		lines = nil
		idleThrottle()
		DebugPrintThrottled(strings.Repeat("x", 12*1024))
		debugPrintStopwatch.stop()
		debugPrintStopwatch.total = time.Second
		DebugPrintThrottled("y")
		equal(t, len(lines), 1)
		loop.Elapse(time.Second)
		equal(t, len(lines), 2)
	})
}

// idleThrottle puts the throttle in the state it has after more than a
// second without printing.
func idleThrottle() {
	debugPrintStopwatch.stop()
	debugPrintStopwatch.total = 2 * time.Second
}

func TestDebugPrintThrottledWrapsAndQueues(t *testing.T) {
	withLoop(t, func(loop *async.Loop) {
		var lines []string
		restore := dart.SetPrintHandler(func(line string) { lines = append(lines, line) })
		defer restore()
		idleThrottle()
		DebugPrintThrottled("aaa bbb\nccc", wrap(4))
		deepEqual(t, lines, []string{"aaa", "bbb", "ccc"})
		// Emptying the buffer restarts the stopwatch from zero (it was reset
		// because more than a second had passed).
		equal(t, debugPrintStopwatch.running, true)
		if debugPrintStopwatch.elapsed() > 500*time.Millisecond {
			t.Errorf("stopwatch not restarted: %v", debugPrintStopwatch.elapsed())
		}
		// While a pause is pending, new messages queue behind it, in order.
		lines = nil
		DebugPrintThrottled(strings.Repeat("q", 13*1024) + "\nr")
		DebugPrintThrottled("s")
		equal(t, len(lines), 1)
		loop.Elapse(time.Second)
		deepEqual(t, lines[1:], []string{"r", "s"})
	})
}

func TestDebugPrintDoneWhenIdle(t *testing.T) {
	withLoop(t, func(loop *async.Loop) {
		done := false
		async.OnComplete(DebugPrintDone(), func(async.Void) { done = true }, nil)
		loop.FlushMicrotasks()
		equal(t, done, true)
	})
}

func deepEqual[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v", got, want)
	}
}

func catchPanic(fn func()) (r any) {
	defer func() { r = recover() }()
	fn()
	return nil
}
