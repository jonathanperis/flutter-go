package foundation

// Port of packages/flutter/lib/src/foundation/print.dart (Flutter 3.50.1):
// debugPrint and its two implementations, and debugWordWrap.
//
// Dart's print is dart.Print (capturable with dart.SetPrintHandler, Go's
// stand-in for a Zone's print override); the throttle's Timer and Completer
// are src/async's, so the pauses run on the current async.Loop
// exactly as Dart's run in fake_async. The throttle's Stopwatch measures
// wall-clock time, as Dart's does (fake_async doesn't fake Stopwatch).
//
// Dart's `String? message` is a Go string: a null message prints "null" in
// every implementation, so Dart's null is passed as "null".

import (
	"iter"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// DebugPrintOptions are a DebugPrintCallback's named parameters (WrapWidth
// nil: no wrapping).
type DebugPrintOptions struct{ WrapWidth *int }

// DebugPrintCallback is the signature of DebugPrint implementations. With a
// WrapWidth, each line of the message is word-wrapped to that width.
type DebugPrintCallback func(message string, opts ...DebugPrintOptions)

// DebugPrint prints a message to the console. Replace it to capture or
// redirect output (flutter_test and Flutter's own tests do). The default is
// DebugPrintThrottled; DebugPrintSynchronously doesn't throttle.
var DebugPrint DebugPrintCallback = DebugPrintThrottled

func wrapWidthOf(opts []DebugPrintOptions) *int {
	if len(opts) > 0 {
		return opts[0].WrapWidth
	}
	return nil
}

// DebugPrintSynchronously is debugPrintSynchronously: debugPrint without
// throttling (used by tests).
func DebugPrintSynchronously(message string, opts ...DebugPrintOptions) {
	if wrapWidth := wrapWidthOf(opts); wrapWidth != nil {
		var lines []string
		for _, line := range strings.Split(message, "\n") {
			lines = slices.AppendSeq(lines, DebugWordWrap(line, *wrapWidth))
		}
		dart.Print(strings.Join(lines, "\n"))
	} else {
		dart.Print(message)
	}
}

// DebugPrintThrottled is debugPrintThrottled: prints at most 12 KiB of
// characters, then pauses for a second before printing the rest, so
// platforms that rate-limit their logs (Android) don't drop messages. With a
// WrapWidth, lines are wrapped with DebugWordWrap.
func DebugPrintThrottled(message string, opts ...DebugPrintOptions) {
	messageLines := strings.Split(message, "\n")
	if wrapWidth := wrapWidthOf(opts); wrapWidth != nil {
		for _, line := range messageLines {
			debugPrintBuffer = slices.AppendSeq(debugPrintBuffer, DebugWordWrap(line, *wrapWidth))
		}
	} else {
		debugPrintBuffer = append(debugPrintBuffer, messageLines...)
	}
	if !debugPrintScheduled {
		debugPrintTask()
	}
}

var (
	debugPrintedCharacters = 0
	debugPrintBuffer       []string
	debugPrintStopwatch    stopwatch
	debugPrintCompleter    *async.Completer[async.Void]
	debugPrintScheduled    = false
)

const (
	kDebugPrintCapacity  = 12 * 1024
	kDebugPrintPauseTime = time.Second
)

func debugPrintTask() {
	debugPrintScheduled = false
	if debugPrintStopwatch.elapsed() > kDebugPrintPauseTime {
		debugPrintStopwatch.stop()
		debugPrintStopwatch.reset()
		debugPrintedCharacters = 0
	}
	for debugPrintedCharacters < kDebugPrintCapacity && len(debugPrintBuffer) != 0 {
		line := debugPrintBuffer[0]
		debugPrintBuffer = debugPrintBuffer[1:]
		debugPrintedCharacters += len16(line) // TODO(ianh): Use the UTF-8 byte length instead
		dart.Print(line)
	}
	if len(debugPrintBuffer) != 0 {
		debugPrintScheduled = true
		debugPrintedCharacters = 0
		async.NewTimer(kDebugPrintPauseTime, debugPrintTask)
		if debugPrintCompleter == nil {
			debugPrintCompleter = async.NewCompleter[async.Void]()
		}
	} else {
		debugPrintStopwatch.start()
		if debugPrintCompleter != nil {
			debugPrintCompleter.Complete(async.Void{})
		}
		debugPrintCompleter = nil
	}
}

// DebugPrintDone is the debugPrintDone getter: a Future that completes when
// DebugPrintThrottled has no buffered output left.
func DebugPrintDone() async.Future[async.Void] {
	if debugPrintCompleter != nil {
		return debugPrintCompleter.Future()
	}
	return async.FutureValue(async.Void{})
}

// stopwatch is dart:core's Stopwatch (wall-clock time).
type stopwatch struct {
	running bool
	started time.Time
	total   time.Duration
}

func (s *stopwatch) start() {
	if !s.running {
		s.running = true
		s.started = time.Now()
	}
}

func (s *stopwatch) stop() {
	if s.running {
		s.total += time.Since(s.started)
		s.running = false
	}
}

func (s *stopwatch) reset() {
	s.total = 0
	if s.running {
		s.started = time.Now()
	}
}

func (s *stopwatch) elapsed() time.Duration {
	if s.running {
		return s.total + time.Since(s.started)
	}
	return s.total
}

var indentPattern = regexp.MustCompile(`^ *(?:[-+*] |[0-9]+[.):] )?`)

type debugWordWrapParseMode int

const (
	debugWordWrapInSpace debugWordWrapParseMode = iota
	debugWordWrapInWord
	debugWordWrapAtBreak
)

// DebugWordWrapOptions are debugWordWrap's named parameters.
type DebugWordWrapOptions struct {
	// WrapIndent prefixes every line after the first.
	WrapIndent string
}

// DebugWordWrap is debugWordWrap: wraps message (which should not contain
// newlines) at spaces to width UTF-16 code units. Lines starting with '#'
// (stack frames) aren't wrapped. Later lines repeat the first line's
// indentation (and a list marker's width), after WrapIndent. Like Dart, it
// panics on a blank message of at least width characters.
func DebugWordWrap(message string, width int, opts ...DebugWordWrapOptions) iter.Seq[string] {
	wrapIndent := ""
	if len(opts) > 0 {
		wrapIndent = opts[0].WrapIndent
	}
	return slices.Values(debugWordWrap(message, width, wrapIndent))
}

func debugWordWrap(messageStr string, width int, wrapIndent string) []string {
	message := toUTF16(messageStr)
	if len(message) < width || dartTrimLeftFirst(messageStr) == '#' {
		return []string{messageStr}
	}
	wrapped := []string{}
	prefixMatch := indentPattern.FindString(messageStr) // ASCII: bytes are code units
	prefix := toUTF16(wrapIndent + strings.Repeat(" ", len(prefixMatch)))
	start := 0
	startForLengthCalculations := 0
	addPrefix := false
	index := len(prefix)
	mode := debugWordWrapInSpace
	var lastWordStart int
	lastWordEnd := -1 // -1 is Dart's null
	for {
		switch mode {
		case debugWordWrapInSpace: // at start of break point (or start of line); can't break until next break
			for index < len(message) && message[index] == ' ' {
				index++
			}
			lastWordStart = index
			mode = debugWordWrapInWord
		case debugWordWrapInWord: // looking for a good break point
			for index < len(message) && message[index] != ' ' {
				index++
			}
			mode = debugWordWrapAtBreak
		case debugWordWrapAtBreak: // at start of break point
			if index-startForLengthCalculations > width || index == len(message) {
				// we are over the width line, so break
				if index-startForLengthCalculations <= width || lastWordEnd == -1 {
					// we should use this point, because either it doesn't actually go over the
					// end (last line), or it does, but there was no earlier break point
					lastWordEnd = index
				}
				if addPrefix {
					wrapped = append(wrapped, fromUTF16(prefix)+fromUTF16(message[start:lastWordEnd]))
				} else {
					wrapped = append(wrapped, fromUTF16(message[start:lastWordEnd]))
					addPrefix = true
				}
				if lastWordEnd >= len(message) {
					return wrapped
				}
				// just yielded a line
				if lastWordEnd == index {
					// we broke at current position
					// eat all the spaces, then set our start point
					for index < len(message) && message[index] == ' ' {
						index++
					}
					start = index
					mode = debugWordWrapInWord
				} else {
					// we broke at the previous break point, and we're at the start of a new one
					if lastWordStart <= lastWordEnd {
						panic(dart.NewAssertionError(nil))
					}
					start = lastWordStart
					mode = debugWordWrapAtBreak
				}
				startForLengthCalculations = start - len(prefix)
				lastWordEnd = -1
			} else {
				// save this break point, we're not yet over the line width
				lastWordEnd = index
				// skip to the end of this break point
				mode = debugWordWrapInSpace
			}
		}
	}
}

// dartTrimLeftFirst is message.trimLeft()[0] as a code unit; like Dart it
// panics (RangeError) when nothing is left.
func dartTrimLeftFirst(s string) uint16 {
	t := toUTF16(strings.TrimLeftFunc(s, isDartWhitespace))
	if len(t) == 0 {
		panic("RangeError (index): Index out of range: no indices are valid: 0")
	}
	return t[0]
}
