package foundation

// Port of packages/flutter/test/foundation/error_reporting_test.dart
// (Flutter 3.50.1): all 7 tests (two share the name "Error reporting -
// NoSuchMethodError" in Dart; the second is ...String here).
//
// Differences:
//   - assert(false, message) has no Go equivalent; assertFalse builds the
//     Dart VM's _AssertionError for the calling Go line (see _AssertionError
//     in assertions_test.go), so the same reformatting is exercised. Go has
//     no column numbers, so its location reads "line N", not "line N pos M"
//     (the VM leaves "pos" out when the column is unknown too).
//   - getSampleStack is Future.sync(() => StackTrace.current) in Dart, which
//     puts a dart:async frame between the closure and getSampleStack; here a
//     microtask run synchronously by the async loop does the same with
//     src/async frames (printed as dart:async and elided by the
//     filter). There are two of them, so the frames kept are #0, #3 and #4
//     where Dart keeps #0, #2 and #3.
//   - NoSuchMethodError.withInvocation is a Go error type of that name with
//     Dart's message.
//   - The regexps match Go file names (.go, under src/foundation).

import (
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// assertFalse is the _AssertionError that assert(false, message) throws at
// the caller's line.
func assertFalse(message any) any {
	_, file, line, _ := runtime.Caller(1)
	return &_AssertionError{failedAssertion: "false", url: "file://" + file, line: line, message: message}
}

func getAssertionErrorWithMessage() any     { return assertFalse("Message goes here.") }
func getAssertionErrorWithoutMessage() any  { return assertFalse(nil) }
func getAssertionErrorWithLongMessage() any { return assertFalse(strings.Repeat("word ", 100)) }

func getSampleStack() dart.StackTrace {
	var stack dart.StackTrace
	loop := async.NewLoop()
	loop.ScheduleMicrotask(func() { stack = dart.StackTraceCurrent() })
	loop.FlushMicrotasks()
	return stack
}

// NoSuchMethodError stands in for Dart's NoSuchMethodError.withInvocation(5,
// Invocation.method(#foo, [2, 4])).
type NoSuchMethodError struct{}

func (NoSuchMethodError) Error() string {
	return "NoSuchMethodError: int has no foo method accepting arguments (_, _)"
}

// errorReportingTest is setUp/tearDown: debugPrint collects into console,
// which must be empty at the end.
func errorReportingTest(t *testing.T, body func(console *[]string)) {
	t.Helper()
	FlutterErrorResetErrorCount()
	console := []string{}
	withDebugPrint(func(message string, _ ...DebugPrintOptions) { console = append(console, message) }, func() {
		body(&console)
	})
	if len(console) != 0 {
		t.Errorf("console not empty at tearDown: %q", console)
	}
}

func matches(t *testing.T, s, pattern string) {
	t.Helper()
	if !regexp.MustCompile(pattern).MatchString(s) {
		t.Errorf("%s\ndoes not match\n%s", s, pattern)
	}
}

var sampleStack = getSampleStack()

func TestErrorReportingAssertWithMessage(t *testing.T) {
	errorReportingTest(t, func(console *[]string) {
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(getAssertionErrorWithMessage(), FlutterErrorDetailsOptions{
			Stack:   sampleStack,
			Library: strp("error handling test"),
			Context: NewErrorDescription("testing the error handling logic"),
			InformationCollector: infoCollector(
				NewErrorDescription("line 1 of extra information"),
				NewErrorHint("line 2 of extra information\n"),
			),
		}))
		matches(t, strings.Join(*console, "\n"),
			`^══╡ EXCEPTION CAUGHT BY ERROR HANDLING TEST ╞═══════════════════════════════════════════════════════\n`+
				`The following assertion was thrown testing the error handling logic:\n`+
				`Message goes here\.\n`+
				`'[^']+src/foundation/error_reporting_test\.go':\n`+
				`Failed assertion: line [0-9]+: 'false'\n`+
				`\n`+
				`When the exception was thrown, this was the stack:\n`+
				`#0      getSampleStack\.<anonymous closure> \([^)]+src/foundation/error_reporting_test\.go:[0-9]+\)\n`+
				`#3      getSampleStack \([^)]+src/foundation/error_reporting_test\.go:[0-9]+\)\n`+
				`#4      init \([^)]+src/foundation/error_reporting_test\.go:[0-9]+\)\n`+
				`(.+\n)+`)
		*console = (*console)[:0]
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(getAssertionErrorWithMessage()))
		equal(t, strings.Join(*console, "\n"), "Another exception was thrown: Message goes here.")
		*console = (*console)[:0]
		FlutterErrorResetErrorCount()
	})
}

func TestErrorReportingAssertWithLongMessage(t *testing.T) {
	errorReportingTest(t, func(console *[]string) {
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(getAssertionErrorWithLongMessage()))
		matches(t, strings.Join(*console, "\n"),
			`^══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞═════════════════════════════════════════════════════════\n`+
				`The following assertion was thrown:\n`+
				`word word word word word word word word word word word word word word word word word word word word\n`+
				`word word word word word word word word word word word word word word word word word word word word\n`+
				`word word word word word word word word word word word word word word word word word word word word\n`+
				`word word word word word word word word word word word word word word word word word word word word\n`+
				`word word word word word word word word word word word word word word word word word word word word\n`+
				`'[^']+src/foundation/error_reporting_test\.go':\n`+
				`Failed assertion: line [0-9]+: 'false'\n`+
				`════════════════════════════════════════════════════════════════════════════════════════════════════$`)
		*console = (*console)[:0]
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(getAssertionErrorWithLongMessage()))
		equal(t, strings.Join(*console, "\n"),
			"Another exception was thrown: "+
				"word word word word word word word word word word word word word word word word word word word word "+
				"word word word word word word word word word word word word word word word word word word word word "+
				"word word word word word word word word word word word word word word word word word word word word "+
				"word word word word word word word word word word word word word word word word word word word word "+
				"word word word word word word word word word word word word word word word word word word word word")
		*console = (*console)[:0]
		FlutterErrorResetErrorCount()
	})
}

func TestErrorReportingAssertWithNoMessage(t *testing.T) {
	errorReportingTest(t, func(console *[]string) {
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(getAssertionErrorWithoutMessage(), FlutterErrorDetailsOptions{
			Stack:   sampleStack,
			Library: strp("error handling test"),
			Context: NewErrorDescription("testing the error handling logic"),
			InformationCollector: infoCollector(
				NewErrorDescription("line 1 of extra information"),
				NewErrorDescription("line 2 of extra information\n"), // the trailing newlines here are intentional
			),
		}))
		matches(t, strings.Join(*console, "\n"),
			`^══╡ EXCEPTION CAUGHT BY ERROR HANDLING TEST ╞═══════════════════════════════════════════════════════\n`+
				`The following assertion was thrown testing the error handling logic:\n`+
				`'[^']+src/foundation/error_reporting_test\.go':[\n ]`+
				`Failed[\n ]assertion:[\n ]line[\n ][0-9]+:[\n ]'false':[\n ]is[\n ]not[\n ]true\.\n`+
				`\n`+
				`When the exception was thrown, this was the stack:\n`+
				`#0      getSampleStack\.<anonymous closure> \([^)]+src/foundation/error_reporting_test\.go:[0-9]+\)\n`+
				`#3      getSampleStack \([^)]+src/foundation/error_reporting_test\.go:[0-9]+\)\n`+
				`#4      init \([^)]+src/foundation/error_reporting_test\.go:[0-9]+\)\n`+
				`(.+\n)+`)
		*console = (*console)[:0]
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(getAssertionErrorWithoutMessage()))
		matches(t, strings.Join(*console, "\n"),
			`Another exception was thrown: '[^']+src/foundation/error_reporting_test\.go': Failed assertion: line [0-9]+: 'false': is not true\.`)
		*console = (*console)[:0]
		FlutterErrorResetErrorCount()
	})
}

func TestErrorReportingNoSuchMethodError(t *testing.T) {
	errorReportingTest(t, func(console *[]string) {
		var exception any = NoSuchMethodError{}
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(exception))
		matches(t, strings.Join(*console, "\n"),
			`^══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞═════════════════════════════════════════════════════════\n`+
				`The following NoSuchMethodError was thrown:\n`+
				`int has no foo method accepting arguments \(_, _\)\n`+
				`════════════════════════════════════════════════════════════════════════════════════════════════════$`)
		*console = (*console)[:0]
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(exception))
		equal(t, strings.Join(*console, "\n"),
			"Another exception was thrown: NoSuchMethodError: int has no foo method accepting arguments (_, _)")
		*console = (*console)[:0]
		FlutterErrorResetErrorCount()
	})
}

func TestErrorReportingString(t *testing.T) {
	errorReportingTest(t, func(console *[]string) {
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails("hello"))
		matches(t, strings.Join(*console, "\n"),
			`^══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞═════════════════════════════════════════════════════════\n`+
				`The following message was thrown:\n`+
				`hello\n`+
				`════════════════════════════════════════════════════════════════════════════════════════════════════$`)
		*console = (*console)[:0]
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails("hello again"))
		equal(t, strings.Join(*console, "\n"), "Another exception was thrown: hello again")
		*console = (*console)[:0]
		FlutterErrorResetErrorCount()
	})
}

// Regression test for https://github.com/flutter/flutter/issues/62223
func TestErrorReportingEmptyStack(t *testing.T) {
	errorReportingTest(t, func(console *[]string) {
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails("exception - empty stack", FlutterErrorDetailsOptions{
			Stack: dart.StackTraceFromString(""),
		}))
		matches(t, strings.Join(*console, "\n"),
			`^══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞═════════════════════════════════════════════════════════\n`+
				`The following message was thrown:\n`+
				`exception - empty stack\n`+
				`\n`+
				`When the exception was thrown, this was the stack\n`+
				`════════════════════════════════════════════════════════════════════════════════════════════════════$`)
		*console = (*console)[:0]
		FlutterErrorResetErrorCount()
	})
}

func TestErrorReportingStackTracesAreNotTruncated(t *testing.T) {
	const stackString = `#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#4      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)
#5      ComponentElement.performRebuild (package:flutter/src/widgets/framework.dart:4303:15)
#6      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)
#7      ComponentElement._firstBuild (package:flutter/src/widgets/framework.dart:4286:5)
#8      StatefulElement._firstBuild (package:flutter/src/widgets/framework.dart:4461:11)
#9      ComponentElement.mount (package:flutter/src/widgets/framework.dart:4281:5)
#10      Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)`
	errorReportingTest(t, func(console *[]string) {
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(dart.NewAssertionError("Test assertion"), FlutterErrorDetailsOptions{
			Stack: dart.StackTraceFromString(stackString),
		}))
		x := strings.Join(*console, "\n")
		want := `══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞═════════════════════════════════════════════════════════
The following assertion was thrown:
Assertion failed: "Test assertion"

When the exception was thrown, this was the stack:
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#4      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)
#5      ComponentElement.performRebuild (package:flutter/src/widgets/framework.dart:4303:15)
#6      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)
#7      ComponentElement._firstBuild (package:flutter/src/widgets/framework.dart:4286:5)
#8      StatefulElement._firstBuild (package:flutter/src/widgets/framework.dart:4461:11)
#9      ComponentElement.mount (package:flutter/src/widgets/framework.dart:4281:5)`
		if !strings.HasPrefix(x, want) {
			t.Errorf("got\n%s\nwant prefix\n%s", x, want)
		}
		*console = (*console)[:0]
		FlutterErrorResetErrorCount()
	})
}
