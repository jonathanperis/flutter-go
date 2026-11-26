package foundation

// Port of packages/flutter/test/foundation/assertions_test.dart (Flutter
// 3.50.1): all 11 tests. The tests after them are flutter-go's own (stack
// filters, reporting, Go panics), for rules Flutter's tests don't pin down.
//
// Differences:
//   - Dart's `throw x` / `on AssertionError catch (e)` is panic(x) /
//     catchPanic; a failed FlutterError.fromParts check panics with a
//     dart.AssertionError holding the FlutterError, as Dart's assert does.
//   - ArgumentError.notNull('myArgument') is a Go error with the same
//     message (Go has no ArgumentError); the test checks the summary text.
//   - Dart runs each test file in a fresh isolate, so FlutterError's error
//     count starts at 0; Go tests share the package, so tests that dump
//     errors call FlutterErrorResetErrorCount first.

import (
	"iter"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func nodes(n ...DiagnosticsNode) iter.Seq[DiagnosticsNode] { return slices.Values(n) }

func contains(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("%q does not contain %q", s, sub)
	}
}

func TestAssertionsDebugPrintStack(t *testing.T) {
	label := "Example label"
	maxFrames := 7
	log := captureOutput(func() {
		DebugPrintStack(DebugPrintStackOptions{Label: &label, MaxFrames: &maxFrames})
	})
	if len(log) < 2 {
		t.Fatalf("log too short: %q", log)
	}
	contains(t, log[0], "Example label")
	contains(t, log[1], "DebugPrintStack")
}

func TestAssertionsShouldShowMessageOfErrorDescription(t *testing.T) {
	const descriptionMessage = "This is the message"
	errorDescription := NewErrorDescription(descriptionMessage)
	equal(t, errorDescription.String(), descriptionMessage)
}

func TestAssertionsDebugPrintStackDumpErrorToConsole(t *testing.T) {
	FlutterErrorResetErrorCount()
	defer FlutterErrorResetErrorCount()
	log := captureOutput(func() {
		details := NewFlutterErrorDetails("Example exception", FlutterErrorDetailsOptions{
			Stack:   dart.StackTraceCurrent(),
			Library: strp("Example library"),
			Context: NewErrorDescription("Example context"),
			InformationCollector: func() iter.Seq[DiagnosticsNode] {
				return nodes(NewErrorDescription("Example information"))
			},
		})
		FlutterErrorDumpErrorToConsole(details)
	})
	if len(log) <= 3 {
		t.Fatalf("log too short: %q", log)
	}
	contains(t, log[0], "EXAMPLE LIBRARY")
	contains(t, log[1], "Example context")
	contains(t, log[2], "Example exception")
	joined := strings.Join(log, "\n")
	contains(t, joined, "captureOutput")
	contains(t, joined, "\nExample information\n")
}

func infoCollector(n ...DiagnosticsNode) InformationCollector {
	return func() iter.Seq[DiagnosticsNode] { return nodes(n...) }
}

func TestFlutterErrorDetailsToString(t *testing.T) {
	equal(t, NewFlutterErrorDetails("MESSAGE", FlutterErrorDetailsOptions{
		Library:              strp("LIBRARY"),
		Context:              NewErrorDescription("CONTEXTING"),
		InformationCollector: infoCollector(NewErrorDescription("INFO")),
	}).String(),
		"══╡ EXCEPTION CAUGHT BY LIBRARY ╞════════════════════════════════\n"+
			"The following message was thrown CONTEXTING:\n"+
			"MESSAGE\n"+
			"\n"+
			"INFO\n"+
			"═════════════════════════════════════════════════════════════════\n")
	equal(t, NewFlutterErrorDetails("MESSAGE", FlutterErrorDetailsOptions{
		Context:              NewErrorDescription("CONTEXTING"),
		InformationCollector: infoCollector(NewErrorDescription("INFO")),
	}).String(),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The following message was thrown CONTEXTING:\n"+
			"MESSAGE\n"+
			"\n"+
			"INFO\n"+
			"═════════════════════════════════════════════════════════════════\n")
	equal(t, NewFlutterErrorDetails("MESSAGE", FlutterErrorDetailsOptions{
		Context:              NewErrorDescription("CONTEXTING " + "SomeContext(BlaBla)"),
		InformationCollector: infoCollector(NewErrorDescription("INFO")),
	}).String(),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The following message was thrown CONTEXTING SomeContext(BlaBla):\n"+
			"MESSAGE\n"+
			"\n"+
			"INFO\n"+
			"═════════════════════════════════════════════════════════════════\n")
	equal(t, NewFlutterErrorDetails("MESSAGE").String(),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The following message was thrown:\n"+
			"MESSAGE\n"+
			"═════════════════════════════════════════════════════════════════\n")
}

func TestFlutterErrorDetailsToStringShort(t *testing.T) {
	equal(t, NewFlutterErrorDetails("MESSAGE", FlutterErrorDetailsOptions{
		Library:              strp("library"),
		Context:              NewErrorDescription("CONTEXTING"),
		InformationCollector: infoCollector(NewErrorDescription("INFO")),
	}).ToStringShort(), "Exception caught by library")
}

func TestFlutterErrorDefaultConstructor(t *testing.T) {
	err := NewFlutterError("My Error Summary.\n" +
		"My first description.\n" +
		"My second description.")
	equal(t, len(err.Diagnostics), 3)
	equal(t, err.Diagnostics[0].Level(), DiagnosticLevelSummary)
	equal(t, err.Diagnostics[1].Level(), DiagnosticLevelInfo)
	equal(t, err.Diagnostics[2].Level(), DiagnosticLevelInfo)
	equal(t, err.Diagnostics[0].String(), "My Error Summary.")
	equal(t, err.Diagnostics[1].String(), "My first description.")
	equal(t, err.Diagnostics[2].String(), "My second description.")
	equal(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   My Error Summary.\n"+
			"   My first description.\n"+
			"   My second description.\n")

	err = NewFlutterError("My Error Summary.\n" +
		"My first description.\n" +
		"My second description.\n" +
		"\n")
	equal(t, len(err.Diagnostics), 5)
	equal(t, err.Diagnostics[0].Level(), DiagnosticLevelSummary)
	equal(t, err.Diagnostics[1].Level(), DiagnosticLevelInfo)
	equal(t, err.Diagnostics[2].Level(), DiagnosticLevelInfo)
	equal(t, err.Diagnostics[0].String(), "My Error Summary.")
	equal(t, err.Diagnostics[1].String(), "My first description.")
	equal(t, err.Diagnostics[2].String(), "My second description.")
	equal(t, err.Diagnostics[3].String(), "")
	equal(t, err.Diagnostics[4].String(), "")
	equal(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   My Error Summary.\n"+
			"   My first description.\n"+
			"   My second description.\n"+
			"\n"+
			"\n")

	err = NewFlutterError("My Error Summary.\n" +
		"My first description.\n" +
		"\n" +
		"My second description.")
	equal(t, len(err.Diagnostics), 4)
	equal(t, err.Diagnostics[0].Level(), DiagnosticLevelSummary)
	equal(t, err.Diagnostics[1].Level(), DiagnosticLevelInfo)
	equal(t, err.Diagnostics[2].Level(), DiagnosticLevelInfo)
	equal(t, err.Diagnostics[3].Level(), DiagnosticLevelInfo)
	equal(t, err.Diagnostics[0].String(), "My Error Summary.")
	equal(t, err.Diagnostics[1].String(), "My first description.")
	equal(t, err.Diagnostics[2].String(), "")
	equal(t, err.Diagnostics[3].String(), "My second description.")
	equal(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   My Error Summary.\n"+
			"   My first description.\n"+
			"\n"+
			"   My second description.\n")

	err = NewFlutterError("My Error Summary.")
	equal(t, len(err.Diagnostics), 1)
	equal(t, err.Diagnostics[0].Level(), DiagnosticLevelSummary)
	equal(t, err.Diagnostics[0].String(), "My Error Summary.")
	equal(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   My Error Summary.\n")
}

// catchAssertion is `try { fn() } on AssertionError catch (e) { return e; }`.
func catchAssertion(t *testing.T, fn func()) dart.AssertionFailure {
	t.Helper()
	r := catchPanic(fn)
	a, ok := r.(dart.AssertionFailure)
	if !ok {
		t.Fatalf("expected an AssertionError, got %#v", r)
	}
	return a
}

func TestMalformedFlutterErrorObjects(t *testing.T) {
	{
		err := catchAssertion(t, func() { panic(NewFlutterErrorFromParts([]DiagnosticsNode{})) })
		equal(t, NewFlutterErrorDetails(err).String(),
			"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
				"The following assertion was thrown:\n"+
				"Empty FlutterError\n"+
				"═════════════════════════════════════════════════════════════════\n")
	}
	{
		err := catchAssertion(t, func() {
			panic(NewFlutterErrorFromParts([]DiagnosticsNode{NewErrorDescription("Error description without a summary")}))
		})
		equal(t, NewFlutterErrorDetails(err).String(),
			"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
				"The following assertion was thrown:\n"+
				"FlutterError is missing a summary.\n"+
				"All FlutterError objects should start with a short (one line)\n"+
				"summary description of the problem that was detected.\n"+
				"Malformed FlutterError:\n"+
				"  Error description without a summary\n"+
				"\n"+
				"This error should still help you solve your problem, however\n"+
				"please also report this malformed error in the framework by\n"+
				"filing a bug on GitHub:\n"+
				"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml\n"+
				"═════════════════════════════════════════════════════════════════\n")
	}
	{
		err := catchAssertion(t, func() {
			panic(NewFlutterErrorFromParts([]DiagnosticsNode{
				NewErrorSummary("Error Summary A"),
				NewErrorDescription("Some descriptionA"),
				NewErrorSummary("Error Summary B"),
				NewErrorDescription("Some descriptionB"),
			}))
		})
		equal(t, NewFlutterErrorDetails(err).String(),
			"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
				"The following assertion was thrown:\n"+
				"FlutterError contained multiple error summaries.\n"+
				"All FlutterError objects should have only a single short (one\n"+
				"line) summary description of the problem that was detected.\n"+
				"Malformed FlutterError:\n"+
				"  Error Summary A\n"+
				"  Some descriptionA\n"+
				"  Error Summary B\n"+
				"  Some descriptionB\n"+
				"\n"+
				"The malformed error has 2 summaries.\n"+
				"Summary 1: Error Summary A\n"+
				"Summary 2: Error Summary B\n"+
				"\n"+
				"This error should still help you solve your problem, however\n"+
				"please also report this malformed error in the framework by\n"+
				"filing a bug on GitHub:\n"+
				"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml\n"+
				"═════════════════════════════════════════════════════════════════\n")
	}
	{
		err := catchAssertion(t, func() {
			panic(NewFlutterErrorFromParts([]DiagnosticsNode{
				NewErrorDescription("Some description"),
				NewErrorSummary("Error summary"),
			}))
		})
		equal(t, NewFlutterErrorDetails(err).String(),
			"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
				"The following assertion was thrown:\n"+
				"FlutterError is missing a summary.\n"+
				"All FlutterError objects should start with a short (one line)\n"+
				"summary description of the problem that was detected.\n"+
				"Malformed FlutterError:\n"+
				"  Some description\n"+
				"  Error summary\n"+
				"\n"+
				"This error should still help you solve your problem, however\n"+
				"please also report this malformed error in the framework by\n"+
				"filing a bug on GitHub:\n"+
				"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml\n"+
				"═════════════════════════════════════════════════════════════════\n")
	}
}

// ArgumentError stands in for Dart's ArgumentError.notNull('myArgument').
type ArgumentError struct{}

func (ArgumentError) Error() string { return "Invalid argument(s) (myArgument): Must not be null" }

func singleErrorSummary(t *testing.T, node DiagnosticsNode) *ErrorSummary {
	t.Helper()
	var found []*ErrorSummary
	for _, p := range node.GetProperties() {
		if s, ok := p.(*ErrorSummary); ok {
			found = append(found, s)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want a single ErrorSummary, got %d", len(found))
	}
	return found[0]
}

func TestUserThrownExceptionsHaveErrorSummaryProperties(t *testing.T) {
	{
		e := catchPanic(func() { panic("User thrown string") })
		node := NewFlutterErrorDetails(e).ToDiagnosticsNode()
		deepEqual(t, singleErrorSummary(t, node).TypedValue(), []any{"User thrown string"})
	}
	{
		e := catchPanic(func() { panic(ArgumentError{}) })
		node := NewFlutterErrorDetails(e).ToDiagnosticsNode()
		deepEqual(t, singleErrorSummary(t, node).TypedValue(), []any{"Invalid argument(s) (myArgument): Must not be null"})
	}
}

func TestIdentifiesUserFault(t *testing.T) {
	// User fault because they called `new Text(null)` from their own code.
	stack := dart.StackTraceFromString(`#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#4      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)
#5      ComponentElement.performRebuild (package:flutter/src/widgets/framework.dart:4303:15)
#6      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)
#7      ComponentElement._firstBuild (package:flutter/src/widgets/framework.dart:4286:5)
#8      StatefulElement._firstBuild (package:flutter/src/widgets/framework.dart:4461:11)
#9      ComponentElement.mount (package:flutter/src/widgets/framework.dart:4281:5)
#10      Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#11     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#12     SingleChildRenderObjectElement.mount (package:flutter/blah.dart:999:9)`)

	details := NewFlutterErrorDetails(dart.NewAssertionError("Test assertion"), FlutterErrorDetailsOptions{Stack: stack})
	builder := NewDiagnosticPropertiesBuilder()
	details.DebugFillProperties(builder)

	equal(t, len(builder.Properties), 4)
	equal(t, builder.Properties[0].String(), "The following assertion was thrown:")
	contains(t, builder.Properties[1].String(), "Assertion failed")
	_, isSpacer := builder.Properties[2].(*ErrorSpacer)
	equal(t, isSpacer, true)
	trace := builder.Properties[3].(*DiagnosticsStackTrace)
	equal(t, trace.Value(), any(stack))
}

func TestIdentifiesOurFault(t *testing.T) {
	// Our fault because we should either have an assertion in `text_helper.dart`
	// or we should make sure not to pass bad values into new Text.
	stack := dart.StackTraceFromString(`#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      new SomeWidgetUsingText (package:flutter/src/widgets/text_helper.dart:287:10)
#4      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#5      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)
#6      ComponentElement.performRebuild (package:flutter/src/widgets/framework.dart:4303:15)
#7      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)
#8      ComponentElement._firstBuild (package:flutter/src/widgets/framework.dart:4286:5)
#9      StatefulElement._firstBuild (package:flutter/src/widgets/framework.dart:4461:11)
#10     ComponentElement.mount (package:flutter/src/widgets/framework.dart:4281:5)
#11     Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#12     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#13     SingleChildRenderObjectElement.mount (package:flutter/blah.dart:999:9)`)

	details := NewFlutterErrorDetails(dart.NewAssertionError("Test assertion"), FlutterErrorDetailsOptions{Stack: stack})
	builder := NewDiagnosticPropertiesBuilder()
	details.DebugFillProperties(builder)
	equal(t, len(builder.Properties), 6)
	equal(t, builder.Properties[0].String(), "The following assertion was thrown:")
	contains(t, builder.Properties[1].String(), "Assertion failed")
	_, isSpacer := builder.Properties[2].(*ErrorSpacer)
	equal(t, isSpacer, true)
	equal(t, builder.Properties[3].String(),
		"Either the assertion indicates an error in the framework itself, or we should "+
			"provide substantially more information in this error message to help you determine "+
			"and fix the underlying cause.\n"+
			"In either case, please report this assertion by filing a bug on GitHub:\n"+
			"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml")
	_, isSpacer = builder.Properties[4].(*ErrorSpacer)
	equal(t, isSpacer, true)
	trace := builder.Properties[5].(*DiagnosticsStackTrace)
	equal(t, trace.Value(), any(stack))
}

func TestRepetitiveStackFrameFilterDoesNotGoOutOfRange(t *testing.T) {
	filter := RepetitiveStackFrameFilter{
		Frames: []PartialStackFrame{
			{ClassName: "TestClass", Method: "test1", Package: dart.StringPattern("package:test/blah.dart")},
			{ClassName: "TestClass", Method: "test2", Package: dart.StringPattern("package:test/blah.dart")},
			{ClassName: "TestClass", Method: "test3", Package: dart.StringPattern("package:test/blah.dart")},
		},
		Replacement: "test",
	}
	reasons := make([]*string, 2)
	filter.Filter([]StackFrame{
		{ClassName: "TestClass", Method: "test1", PackageScheme: "package", Package: "test", PackagePath: "blah.dart", Line: 1, Column: 1, Number: 0, Source: ""},
		{ClassName: "TestClass", Method: "test2", PackageScheme: "package", Package: "test", PackagePath: "blah.dart", Line: 1, Column: 1, Number: 0, Source: ""},
	}, reasons)
	deepEqual(t, reasons, make([]*string, 2))
}

// --- flutter-go's own tests

// withStackFilters runs fn with only the given default stack filters.
func withStackFilters(filters []StackFilter, fn func()) {
	saved := stackFilters
	stackFilters = filters
	defer func() { stackFilters = saved }()
	fn()
}

func filterLines(s string) []string {
	return slices.Collect(FlutterErrorDefaultStackFilter(slices.Values(strings.Split(s, "\n"))))
}

func TestDefaultStackFilterElision(t *testing.T) {
	stack := `#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      main (package:app/main.dart:1:1)
#2      _rootRun (dart:async/zone.dart:1126:13)
#3      _Timer._runTimers (dart:isolate-patch/timer_impl.dart:398:19)
#4      _FakeAsync.run (package:fake_async/fake_async.dart:1:1)
#5      Chain.capture (package:stack_trace/src/chain.dart:1:1)
#6      _asyncThenWrapperHelper (dart:async-patch/async_patch.dart:83:23)
#7      _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174:12)
#8      _FrameCallbackEntry.run (package:flutter/x.dart:1:1)
#9      helper (package:app/main.dart:2:2)
#10     _CustomZone.run (dart:async/zone.dart:1023:19)`
	withStackFilters(nil, func() {
		deepEqual(t, filterLines(stack), []string{
			"#1      main (package:app/main.dart:1:1)",
			"#9      helper (package:app/main.dart:2:2)",
			"(elided 9 frames from class _AssertionError, class _FakeAsync, class _FrameCallbackEntry, " +
				"class _RawReceivePortImpl, class _Timer, dart:async, dart:async-patch, and package:stack_trace)",
		})
		deepEqual(t, filterLines("#0      a (package:app/a.dart:1:1)\n#1      _rootRun (dart:async/zone.dart:1:1)"), []string{
			"#0      a (package:app/a.dart:1:1)",
			"(elided one frame from dart:async)",
		})
		deepEqual(t, filterLines("#0      _rootRun (dart:async/zone.dart:1:1)\n#1      a (package:app/a.dart:1:1)\n#2      b (dart:async-patch/x.dart:1:1)"), []string{
			"#1      a (package:app/a.dart:1:1)",
			"(elided 2 frames from dart:async and dart:async-patch)",
		})
		deepEqual(t, filterLines("#0      _rootRun (dart:async/zone.dart:1:1)\n#1      _rootRun (dart:async/zone.dart:1:1)"), []string{
			"(elided 2 frames from dart:async)",
		})
		// Other dart: libraries and packages are kept; lines that aren't
		// frames are dropped.
		deepEqual(t, filterLines("#0      a (dart:core/x.dart:1:1)\nnot a frame\n#1      b (package:stack_trace_x/y.dart:1:1)"), []string{
			"#0      a (dart:core/x.dart:1:1)",
			"#1      b (package:stack_trace_x/y.dart:1:1)",
		})
	})
}

func TestDefaultStackFilterAppliesStackFilters(t *testing.T) {
	framework := dart.StringPattern("package:flutter/src/widgets/framework.dart")
	filters := []StackFilter{
		RepetitiveStackFrameFilter{
			Frames: []PartialStackFrame{
				{ClassName: "Element", Method: "inflateWidget", Package: framework},
				{ClassName: "Element", Method: "updateChild", Package: framework},
			},
			Replacement: "...     Normal element mounting",
		},
		RepetitiveStackFrameFilter{
			Frames:      []PartialStackFrame{{ClassName: "A", Method: "x", Package: regexp.MustCompile(`^package:app/`)}},
			Replacement: "...     A.x",
		},
	}
	stack := `#0      main (package:app/main.dart:1:1)
#1      Element.inflateWidget (package:flutter/src/widgets/framework.dart:1:1)
#2      Element.updateChild (package:flutter/src/widgets/framework.dart:2:1)
#3      Element.inflateWidget (package:flutter/src/widgets/framework.dart:1:1)
#4      Element.updateChild (package:flutter/src/widgets/framework.dart:2:1)
#5      A.x (package:app/a.dart:1:1)
#6      B.y (package:app/a.dart:1:1)
#7      A.x (package:app/a.dart:1:1)
#8      A.x (package:other/a.dart:1:1)
#9      Element.inflateWidget (package:flutter/src/widgets/framework.dart:1:1)
#10     Element.updateChild (package:flutter/src/widgets/framework.dart:2:1)`
	withStackFilters(filters, func() {
		deepEqual(t, filterLines(stack), []string{
			"#0      main (package:app/main.dart:1:1)",
			// Flutter counts a collapsed run as index - start + 2: one more than
			// the 4 frames it replaces (a quirk kept on purpose).
			"...     Normal element mounting (5 frames)",
			"...     A.x (1 frame)",
			"#6      B.y (package:app/a.dart:1:1)",
			"...     A.x (1 frame)",
			"#8      A.x (package:other/a.dart:1:1)",
			// A run ending at the last frame isn't matched (Flutter's loop
			// stops at length - numFrames).
			"#9      Element.inflateWidget (package:flutter/src/widgets/framework.dart:1:1)",
			"#10     Element.updateChild (package:flutter/src/widgets/framework.dart:2:1)",
		})
	})
	// FlutterErrorAddDefaultStackFilter appends to the registered filters.
	withStackFilters(nil, func() {
		FlutterErrorAddDefaultStackFilter(filters[1])
		equal(t, len(stackFilters), 1)
	})
}

func TestPartialStackFrameMatches(t *testing.T) {
	frame := parseLine(t, "#0      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)")
	equal(t, PartialStackFrame{Package: dart.StringPattern("package:flutter/src/widgets/framework.dart"), ClassName: "Element", Method: "rebuild"}.Matches(frame), true)
	equal(t, PartialStackFrame{Package: dart.StringPattern("widgets/"), ClassName: "Element", Method: "rebuild"}.Matches(frame), true)
	equal(t, PartialStackFrame{Package: dart.StringPattern("package:flutter/src/rendering"), ClassName: "Element", Method: "rebuild"}.Matches(frame), false)
	equal(t, PartialStackFrame{Package: dart.StringPattern(""), ClassName: "Element", Method: "build"}.Matches(frame), false)
	equal(t, PartialStackFrame{Package: dart.StringPattern(""), ClassName: "Elements", Method: "rebuild"}.Matches(frame), false)
	equal(t, PartialStackFrameAsynchronousSuspension.Matches(StackFrameAsynchronousSuspension), true)
	equal(t, PartialStackFrameAsynchronousSuspension.Matches(frame), false)
}

func TestFlutterErrorToStringAndParts(t *testing.T) {
	err := NewFlutterErrorFromParts([]DiagnosticsNode{
		NewErrorSummary("Summary line that is long enough that it would wrap if FlutterError.toString wrapped its lines at all"),
		NewErrorDescription("A description."),
		NewErrorSpacer(),
		NewErrorHint("A hint."),
		NewDiagnosticsProperty("prop", 3.5),
	})
	equal(t, err.String(), "Summary line that is long enough that it would wrap if FlutterError.toString wrapped its lines at all\n"+
		"A description.\n"+
		"\n"+
		"A hint.\n"+
		"prop: 3.5")
	equal(t, err.Message(), err.String())
	equal(t, err.Error(), err.String())
	equal(t, err.AssertionMessage(), any(err.String()))
	var asError error = err
	_, isAssertion := asError.(dart.AssertionFailure)
	equal(t, isAssertion, true)
	equal(t, err.ToStringShort(), "FlutterError")
	equal(t, NewErrorHint("h").Level(), DiagnosticLevelHint)
	equal(t, NewErrorSummary("s").Level(), DiagnosticLevelSummary)
	equal(t, NewErrorDescription("d").Level(), DiagnosticLevelInfo)
	equal(t, NewErrorSpacer().Level(), DiagnosticLevelInfo)
	equal(t, NewErrorSpacer().Style(), DiagnosticsTreeStyleSingleLine)
	equal(t, NewErrorHint("h").Style(), DiagnosticsTreeStyleFlat)
	equal(t, NewErrorHint("h").ShowName(), false)
	equal(t, NewErrorHint("h").ShowSeparator(), false)
}

func TestFlutterErrorDetailsSummaryAndAnotherException(t *testing.T) {
	FlutterErrorResetErrorCount()
	defer FlutterErrorResetErrorCount()
	var console []string
	withDebugPrint(func(message string, _ ...DebugPrintOptions) { console = append(console, message) }, func() {
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails("first"))
		equal(t, len(console), 1)
		console = nil
		// A FlutterError's summary is its ErrorSummary, wherever it is.
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(NewFlutterErrorFromParts([]DiagnosticsNode{
			NewErrorSummary("The summary."), NewErrorDescription("More."),
		})))
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(dart.NewAssertionError(NewFlutterError("Wrapped summary.\nDetails."))))
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails("  multi\nline"))
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(42))
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails(struct{ A int }{7}))
		deepEqual(t, console, []string{
			"Another exception was thrown: The summary.",
			"Another exception was thrown: Wrapped summary.",
			"Another exception was thrown: multi",
			"Another exception was thrown: 42",
			"Another exception was thrown: Instance of 'struct { A int }'",
		})
		// forceReport dumps in full again.
		console = nil
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails("again"), DumpErrorToConsoleOptions{ForceReport: true})
		contains(t, console[0], "EXCEPTION CAUGHT BY FLUTTER FRAMEWORK")
		// After a reset, the next error is in full.
		console = nil
		FlutterErrorResetErrorCount()
		FlutterErrorDumpErrorToConsole(NewFlutterErrorDetails("x"))
		contains(t, console[0], "EXCEPTION CAUGHT BY FLUTTER FRAMEWORK")
	})
}

type plainError struct{ msg string }

func (e *plainError) Error() string { return e.msg }

func TestFlutterErrorDetailsExceptionKinds(t *testing.T) {
	render := func(exception any) string { return NewFlutterErrorDetails(exception).String() }
	equal(t, render(42),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The number 42 was thrown.\n"+
			"═════════════════════════════════════════════════════════════════\n")
	equal(t, render(2.0),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The number 2.0 was thrown.\n"+
			"═════════════════════════════════════════════════════════════════\n")
	// An error: its type, and its message without a "Type: " prefix.
	equal(t, render(&plainError{"plainError: boom"}),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The following plainError was thrown:\n"+
			"boom\n"+
			"═════════════════════════════════════════════════════════════════\n")
	// Any other object: "<Type> object", its toString indented (the summary
	// trims the indentation).
	equal(t, render(ExampleEnumWorld),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The following ExampleEnum object was thrown:\n"+
			"  ExampleEnum.world\n"+
			"═════════════════════════════════════════════════════════════════\n")
	equal(t, NewFlutterErrorDetails(&plainError{""}).ExceptionAsString(), "  <no message available>")
	equal(t, NewFlutterErrorDetails("trailing  \n ").ExceptionAsString(), "trailing")
	equal(t, NewFlutterErrorDetails(true).ExceptionAsString(), "  true")
	// AssertionError(message): the message is shown via toString.
	equal(t, NewFlutterErrorDetails(dart.NewAssertionError("a \"quoted\"\nmessage")).ExceptionAsString(),
		`Assertion failed: "a \"quoted\"\nmessage"`)
	equal(t, NewFlutterErrorDetails(dart.NewAssertionError(nil)).ExceptionAsString(), "Assertion failed")
	equal(t, NewFlutterErrorDetails(dart.NewAssertionError(7)).ExceptionAsString(), "Assertion failed: 7")
}

// _AssertionError reproduces the Dart VM's _AssertionError (what
// assert(cond, message) throws), whose toString Flutter reformats.
type _AssertionError struct {
	failedAssertion string
	url             string
	line, column    int
	message         any
}

func (e *_AssertionError) AssertionMessage() any { return e.message }
func (e *_AssertionError) Error() string {
	msg := "is not true."
	if s, ok := e.message.(string); ok {
		msg = s
	} else if e.message != nil {
		msg = dart.SafeToString(e.message)
	}
	columnInfo := ""
	if e.column > 0 {
		columnInfo = " pos " + dartToString(e.column)
	}
	return "'" + e.url + "': Failed assertion: line " + dartToString(e.line) + columnInfo + ": '" + e.failedAssertion + "': " + msg
}

func TestErrorDiagnosticTypesAndDumper(t *testing.T) {
	var description DiagnosticsNode = NewErrorDescription("d")
	_, isDescription := description.(*ErrorDescription)
	var hint DiagnosticsNode = NewErrorHint("h")
	_, isHint := hint.(*ErrorHint)
	_, hintIsDescription := hint.(*ErrorDescription)
	equal(t, isDescription, true)
	equal(t, isHint, true)
	equal(t, hintIsDescription, false)
	var demangler StackTraceDemangler = FlutterErrorDemangleStackTrace
	s := dart.StackTraceFromString("x")
	equal(t, demangler(s), s)
	// ErrorToConsoleDumper.Dump goes through DebugPrint; the web listeners
	// do nothing.
	var got []string
	ErrorToConsoleDumper.AddWebDumpListener(func(m string) { got = append(got, "web "+m) })
	withDebugPrint(func(m string, _ ...DebugPrintOptions) { got = append(got, m) }, func() {
		ErrorToConsoleDumper.Dump("dumped")
	})
	ErrorToConsoleDumper.ClearWebDumpListeners()
	deepEqual(t, got, []string{"dumped"})
}

// customAssertion is an AssertionError with a given toString and message.
type customAssertion struct {
	s   string
	msg any
}

func (e *customAssertion) Error() string         { return e.s }
func (e *customAssertion) AssertionMessage() any { return e.msg }

func TestExceptionAsStringMovesAssertMessageFirst(t *testing.T) {
	e := &_AssertionError{"x != null", "file:///a/b.dart", 12, 7, "The message.  "}
	equal(t, NewFlutterErrorDetails(e).ExceptionAsString(),
		"The message.\n'file:///a/b.dart':\nFailed assertion: line 12 pos 7: 'x != null'")
	// Without " Failed assertion:" the body stays on one line.
	equal(t, NewFlutterErrorDetails(&customAssertion{"head body: Msg", "Msg"}).ExceptionAsString(), "Msg\nhead body")
	// A message that isn't at the end, isn't after ": ", or leaves fewer
	// than 3 characters before it is left alone.
	for _, s := range []string{"a: Msg tail", "head-Msg", ": Msg", "ab Msg", "Msg"} {
		equal(t, NewFlutterErrorDetails(&customAssertion{s, "Msg"}).ExceptionAsString(), s)
	}
	equal(t, NewFlutterErrorDetails(&customAssertion{"ab: Msg", "Msg"}).ExceptionAsString(), "Msg\nab")
	equal(t, NewFlutterErrorDetails(&customAssertion{"a: Msg", "Msg"}).ExceptionAsString(), "Msg\na") // position 3 > 2
	// The last occurrence is the one moved.
	equal(t, NewFlutterErrorDetails(&customAssertion{"x: Msg y: Msg", "Msg"}).ExceptionAsString(), "Msg\nx: Msg y")
	// A non-string message: toString as is.
	equal(t, NewFlutterErrorDetails(&customAssertion{"a: 5", 5}).ExceptionAsString(), "a: 5")
	noMessage := &_AssertionError{"x", "u", 1, 2, nil}
	equal(t, NewFlutterErrorDetails(noMessage).ExceptionAsString(), "'u': Failed assertion: line 1 pos 2: 'x': is not true.")
}

func TestFlutterErrorReportErrorAndHandlers(t *testing.T) {
	saved := FlutterErrorOnError
	defer func() { FlutterErrorOnError = saved }()
	var got *FlutterErrorDetails
	FlutterErrorOnError = func(d *FlutterErrorDetails) { got = d }
	details := NewFlutterErrorDetails("x")
	FlutterErrorReportError(details)
	equal(t, got, details)
	FlutterErrorOnError = nil
	FlutterErrorReportError(details) // no handler: ignored
	// The defaults both dump to the console.
	FlutterErrorResetErrorCount()
	defer FlutterErrorResetErrorCount()
	var console []string
	withDebugPrint(func(message string, _ ...DebugPrintOptions) { console = append(console, message) }, func() {
		saved(details)
		FlutterErrorPresentError(details)
	})
	equal(t, len(console), 2)
	contains(t, console[0], "EXCEPTION CAUGHT BY FLUTTER FRAMEWORK")
	equal(t, console[1], "Another exception was thrown: x")
}

func TestFlutterErrorDetailsCopyWithAndTransformers(t *testing.T) {
	base := NewFlutterErrorDetails("e", FlutterErrorDetailsOptions{Library: NullString, Silent: true})
	equal(t, base.Library == nil, true)
	equal(t, base.ToStringShort(), "Exception caught")
	c := base.CopyWith(FlutterErrorDetailsCopyWithOptions{Library: strp("lib"), Exception: "f"})
	equal(t, *c.Library, "lib")
	equal(t, c.Exception, any("f"))
	equal(t, c.Silent, true)
	equal(t, base.Exception, any("e"))
	equal(t, c.String(), "══╡ EXCEPTION CAUGHT BY LIB ╞════════════════════════════════════\n"+
		"The following message was thrown:\n"+
		"f\n"+
		"═════════════════════════════════════════════════════════════════\n")

	saved := FlutterErrorDetailsPropertiesTransformers
	defer func() { FlutterErrorDetailsPropertiesTransformers = saved }()
	FlutterErrorDetailsPropertiesTransformers = []DiagnosticPropertiesTransformer{
		func(properties iter.Seq[DiagnosticsNode]) iter.Seq[DiagnosticsNode] {
			return func(yield func(DiagnosticsNode) bool) {
				for p := range properties {
					if !yield(p) {
						return
					}
				}
				yield(NewErrorHint("added by a transformer"))
			}
		},
	}
	equal(t, NewFlutterErrorDetails("m").String(),
		"══╡ EXCEPTION CAUGHT BY FLUTTER FRAMEWORK ╞══════════════════════\n"+
			"The following message was thrown:\n"+
			"m\n"+
			"added by a transformer\n"+
			"═════════════════════════════════════════════════════════════════\n")
}

func TestDiagnosticsStackTraceSingleFrameAndFilter(t *testing.T) {
	equal(t, NewDiagnosticsStackTraceSingleFrame("Frame", "#0      a (package:x/y.dart:1:2)").ToStringDeep(),
		"Frame:\n  #0      a (package:x/y.dart:1:2)\n")
	upper := func(lines iter.Seq[string]) iter.Seq[string] {
		return func(yield func(string) bool) {
			for l := range lines {
				if !yield(strings.ToUpper(l)) {
					return
				}
			}
		}
	}
	s := NewDiagnosticsStackTrace("S", dart.StackTraceFromString("#0      a (dart:async/y.dart:1:2)\n"), DiagnosticsStackTraceOptions{StackFilter: upper})
	equal(t, s.ToStringDeep(), "S:\n#0      A (DART:ASYNC/Y.DART:1:2)\n")
	equal(t, s.AllowTruncate(), false)
	equal(t, NewDiagnosticsStackTrace("Empty", nil).ToStringDeep(), "Empty\n")
	// The demangler runs before the filter.
	saved := FlutterErrorDemangleStackTrace
	defer func() { FlutterErrorDemangleStackTrace = saved }()
	FlutterErrorDemangleStackTrace = func(dart.StackTrace) dart.StackTrace {
		return dart.StackTraceFromString("#0      demangled (package:x/y.dart:1:2)")
	}
	equal(t, NewDiagnosticsStackTrace("D", dart.StackTraceFromString("garbage")).ToStringDeep(),
		"D:\n#0      demangled (package:x/y.dart:1:2)\n")
}

// panicsWith runs fn and reports its panic as Flutter would: through
// FlutterErrorReportError, with the panic's stack.
func reportPanic(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			FlutterErrorReportError(NewFlutterErrorDetails(r, FlutterErrorDetailsOptions{
				Stack:   dart.StackTraceFromPanic(),
				Library: strp("go panic test"),
			}))
		}
	}()
	fn()
}

func explode(m map[string]int) int {
	var p *struct{ x int }
	if m == nil {
		return p.x // nil dereference: a runtime.Error
	}
	return 0
}

func TestReportErrorOnARealGoPanic(t *testing.T) {
	FlutterErrorResetErrorCount()
	defer FlutterErrorResetErrorCount()
	var console []string
	withDebugPrint(func(message string, _ ...DebugPrintOptions) { console = append(console, message) }, func() {
		reportPanic(func() { explode(nil) })
	})
	equal(t, len(console), 1)
	out := console[0]
	lines := strings.Split(out, "\n")
	t.Log("\n" + out)
	equal(t, lines[0], "══╡ EXCEPTION CAUGHT BY GO PANIC TEST ╞"+strings.Repeat("═", 61))
	equal(t, len16(lines[0]), 100) // FlutterErrorWrapWidth
	equal(t, lines[1], "The following errorString was thrown:")
	equal(t, lines[2], "runtime error: invalid memory address or nil pointer dereference")
	equal(t, lines[3], "")
	equal(t, lines[4], "When the exception was thrown, this was the stack:")
	// The first frame is the function that panicked, then its callers.
	if !regexp.MustCompile(`^#0      explode \(package:flutter/src/foundation/assertions_test\.go:[0-9]+\)$`).MatchString(lines[5]) {
		t.Errorf("frame 0: %q", lines[5])
	}
	if !regexp.MustCompile(`^#1      TestReportErrorOnARealGoPanic\.<anonymous closure>\.<anonymous closure> \(package:flutter/src/foundation/assertions_test\.go:[0-9]+\)$`).MatchString(lines[6]) {
		t.Errorf("frame 1: %q", lines[6])
	}
	contains(t, out, "\n#3      TestReportErrorOnARealGoPanic.<anonymous closure> (package:flutter/src/foundation/assertions_test.go:")
	if !regexp.MustCompile(`\n#[0-9]+ +tRunner \(file:///.*/testing/testing\.go:[0-9]+\)\n`).MatchString(out) {
		t.Errorf("no testing.tRunner frame in:\n%s", out)
	}
	if strings.Contains(out, "runtime.") || strings.Contains(out, "gopanic") {
		t.Errorf("runtime frames should be dropped:\n%s", out)
	}
}

func TestReportErrorOnAGoPanicInsideAsync(t *testing.T) {
	// A panic in an async microtask: src/async frames are dart:async
	// and the default filter elides them, as Flutter elides dart:async.
	FlutterErrorResetErrorCount()
	defer FlutterErrorResetErrorCount()
	var console []string
	withDebugPrint(func(message string, _ ...DebugPrintOptions) { console = append(console, message) }, func() {
		loop := async.NewLoop()
		defer async.SetCurrent(loop)()
		loop.ScheduleMicrotask(func() { reportPanic(func() { panic(NewFlutterError("Async failure.\nWith details.")) }) })
		loop.FlushMicrotasks()
	})
	equal(t, len(console), 1)
	out := console[0]
	contains(t, out, "The following assertion was thrown:\nAsync failure.\nWith details.\n\nWhen the exception was thrown, this was the stack:\n#0      TestReportErrorOnAGoPanicInsideAsync.")
	if !regexp.MustCompile(`\(elided [0-9]+ frames from dart:async\)`).MatchString(out) {
		t.Errorf("dart:async frames not elided:\n%s", out)
	}
	if strings.Contains(out, "(dart:async/") {
		t.Errorf("dart:async frame kept:\n%s", out)
	}
}
