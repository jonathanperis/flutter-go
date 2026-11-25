package foundation

// Port of packages/flutter/test/foundation/stack_frame_test.dart (Flutter
// 3.50.1). The stack strings and expected frames at the end of this file
// were converted mechanically from the Dart file. Frames compare with
// StackFrame.Equal (Dart's ==, which ignores packageScheme, packagePath and
// isConstructor), as in Flutter.
//
// Not ported:
//   - "Live handling of Stack Overflows": a Go stack overflow is a fatal
//     error, not a recoverable panic, so there is no trace to parse.
//   - "debugPrintStack does not throw on a ParallelWaitError stack": needs
//     Dart's (f1, f2).wait and ParallelWaitError, which flutter_go/async
//     doesn't have, and a package:stack_trace-style trace; the parsing it
//     exercises ("===== asynchronous gap" lines) is covered by "debugPrintStack
//     does not throw on a stack carrying an async-gap marker".

import (
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

func framesEqual(t *testing.T, got, want []StackFrame) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d frames, want %d:\n%v", len(got), len(want), got)
	}
	for i := range got {
		if !got[i].Equal(want[i]) {
			t.Errorf("frame %d:\n got  %v %q\n want %v %q", i, got[i], got[i].Source, want[i], want[i].Source)
		}
	}
}

func parseLine(t *testing.T, line string) StackFrame {
	t.Helper()
	frame, ok := StackFrameFromStackTraceLine(line)
	if !ok {
		t.Fatalf("%q parsed to null", line)
	}
	return frame
}

func TestStackFrameParsesLine(t *testing.T) {
	framesEqual(t, []StackFrame{parseLine(t, strings.Split(stackString, "\n")[0])}, stackFrames[:1])
}

func TestStackFrameParsesString(t *testing.T) {
	framesEqual(t, StackFrameFromStackString(stackString), stackFrames)
}

func TestStackFrameParsesStackTrace(t *testing.T) {
	framesEqual(t, StackFrameFromStackTrace(dart.StackTraceFromString(stackString)), stackFrames)
}

func TestStackFrameParsesComplexStack(t *testing.T) {
	framesEqual(t, StackFrameFromStackString(asyncStackString), asyncStackFrames)
}

func TestStackFrameParsesStackWithoutCols(t *testing.T) {
	framesEqual(t, StackFrameFromStackString(stackFrameNoCols), stackFrameNoColsFrames)
}

func TestStackFrameParsesWebStack(t *testing.T) {
	// skip: kIsWasm (false in flutter-go).
	framesEqual(t, StackFrameFromStackString(webStackTrace), webStackTraceFrames)
}

func TestStackFrameParsesEllipsis(t *testing.T) {
	framesEqual(t, []StackFrame{parseLine(t, "...")}, []StackFrame{StackFrameStackOverFlowElision})
}

func TestStackFrameTracesFromPackageStackTraceParseWithoutThrowing(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/179018.
	frames := StackFrameFromStackString(mangledStackString)
	if !slices.ContainsFunc(frames, StackFrameAsynchronousSuspension.Equal) {
		t.Errorf("frames should contain asynchronousSuspension: %v", frames)
	}
	if len(frames) == 0 {
		t.Errorf("frames should not be empty")
	}
}

func TestStackFrameAsyncGapMarkerParsesToAsynchronousSuspension(t *testing.T) {
	framesEqual(t, []StackFrame{parseLine(t, "===== asynchronous gap ===========================")},
		[]StackFrame{StackFrameAsynchronousSuspension})
}

func TestStackFrameNearMissAsyncGapMarkersAreNotParsedAsAsynchronousSuspension(t *testing.T) {
	for _, line := range []string{
		"===== asynchronous gap ==========================",
		"===== asynchronous gap =========================== ",
		"===== asynchronous frame ===========================",
	} {
		if frame, ok := StackFrameFromStackTraceLine(line); ok {
			t.Errorf("%q parsed to %v, want null", line, frame)
		}
	}
}

func TestStackFrameDebugPrintStackDoesNotThrowOnAStackCarryingAnAsyncGapMarker(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/179018.
	label := "ParallelWaitError-style stack"
	printed := captureOutput(func() {
		withDebugPrint(DebugPrintSynchronously, func() {
			DebugPrintStack(DebugPrintStackOptions{Label: &label, StackTrace: dart.StackTraceFromString(mangledStackString)})
		})
	})
	if len(printed) == 0 {
		t.Errorf("nothing printed")
	}
}

func TestStackFrameCanParseWebConstructorInvocationWithUnknownClassName(t *testing.T) {
	const stackTraceLine = "#32     new (http://localhost:42191/dart-sdk/lib/async/stream_controller.dart:880:9)"
	framesEqual(t, []StackFrame{parseLine(t, stackTraceLine)}, []StackFrame{{
		Number:        32,
		ClassName:     "<unknown>",
		Method:        "",
		PackageScheme: "http",
		Package:       "<unknown>",
		PackagePath:   "dart-sdk/lib/async/stream_controller.dart",
		Line:          880,
		Column:        9,
		Source:        stackTraceLine,
	}})
}

func TestStackFrameParsesToNullForWrongFormat(t *testing.T) {
	if frame, ok := StackFrameFromStackTraceLine("wrong stack trace format"); ok {
		t.Errorf("parsed to %v, want null", frame)
	}
}

// --- flutter-go's own tests

// withDebugPrint runs fn with DebugPrint replaced.
func withDebugPrint(p DebugPrintCallback, fn func()) {
	prev := DebugPrint
	DebugPrint = p
	defer func() { DebugPrint = prev }()
	fn()
}

func TestStackFrameAllFields(t *testing.T) {
	// Equal ignores packageScheme, packagePath and isConstructor; check them.
	frame := parseLine(t, "#2      new Text.rich (package:flutter/src/widgets/text.dart:287:10)")
	equal(t, frame, StackFrame{Number: 2, ClassName: "Text", Method: "rich", PackageScheme: "package",
		Package: "flutter", PackagePath: "src/widgets/text.dart", Line: 287, Column: 10, IsConstructor: true,
		Source: "#2      new Text.rich (package:flutter/src/widgets/text.dart:287:10)"})
	frame = parseLine(t, "#7      A.b.<anonymous closure> (file:///x/y.dart:3)")
	equal(t, frame, StackFrame{Number: 7, ClassName: "A", Method: "b", PackageScheme: "file",
		Package: "<unknown>", PackagePath: "/x/y.dart", Line: 3, Column: -1,
		Source: "#7      A.b.<anonymous closure> (file:///x/y.dart:3)"})
	frame = parseLine(t, "#0      main (dart:ui/../ui/hooks.dart)")
	equal(t, frame, StackFrame{Number: 0, Method: "main", PackageScheme: "dart", Package: "ui",
		PackagePath: "/hooks.dart", Line: -1, Column: -1, Source: "#0      main (dart:ui/../ui/hooks.dart)"})
	equal(t, frame.String(), "StackFrame(#0, dart:ui//hooks.dart:-1:-1, className: , method: main)")
	if !frame.Equal(frame) || frame.HashCode() != parseLine(t, frame.Source).HashCode() {
		t.Errorf("equal frames must be Equal with equal hashes")
	}
	// A '#' line that isn't a VM frame asserts.
	if catchPanic(func() { StackFrameFromStackTraceLine("#0 nothing") }) == nil {
		t.Errorf("a malformed '#' line should panic")
	}
}

func TestStackFrameEqual(t *testing.T) {
	base := StackFrame{Number: 1, PackageScheme: "package", Package: "p", PackagePath: "a.dart", Line: 2,
		Column: 3, ClassName: "C", Method: "m", IsConstructor: false, Source: "src"}
	equal(t, base.Equal(base), true)
	// Fields Dart's == compares.
	for _, change := range []func(*StackFrame){
		func(f *StackFrame) { f.Number++ },
		func(f *StackFrame) { f.Package += "x" },
		func(f *StackFrame) { f.Line++ },
		func(f *StackFrame) { f.Column++ },
		func(f *StackFrame) { f.ClassName += "x" },
		func(f *StackFrame) { f.Method += "x" },
		func(f *StackFrame) { f.Source += "x" },
	} {
		other := base
		change(&other)
		equal(t, base.Equal(other), false)
		equal(t, other.Equal(base), false)
	}
	// Fields it doesn't.
	other := base
	other.PackageScheme, other.PackagePath, other.IsConstructor = "dart", "b.dart", true
	equal(t, base.Equal(other), true)
	equal(t, base.HashCode(), other.HashCode())
}

func TestStackFrameAllFieldsOfFlutterData(t *testing.T) {
	// Flutter's expected frames also hold packageScheme, packagePath and
	// isConstructor, which its == doesn't compare. They match the parser
	// except for three typos in Flutter's test data (packagePath only).
	typos := map[string]string{
		"#3      main (file:///path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart:46:40)": "/path/to/flutter/packages/flutter/foundation/error_reporting_test.dart",
		"#8      _startIsolate.<anonymous closure> (dart:isolate-patch/isolate_patch.dart:307)":                   "isolate-patch.dart",
		"#9      _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174)":                  "isolate-patch.dart",
	}
	found := 0
	for _, c := range []struct {
		stack string
		want  []StackFrame
	}{{stackString, stackFrames}, {asyncStackString, asyncStackFrames}, {stackFrameNoCols, stackFrameNoColsFrames}, {webStackTrace, webStackTraceFrames}} {
		got := StackFrameFromStackString(c.stack)
		equal(t, len(got), len(c.want))
		for i := range min(len(got), len(c.want)) {
			want := c.want[i]
			if typo, ok := typos[want.Source]; ok {
				equal(t, want.PackagePath, typo)
				want.PackagePath = got[i].PackagePath
				found++
			}
			equal(t, got[i], want)
		}
	}
	equal(t, found, len(typos))
}

func TestStackTraceOfAGoPanic(t *testing.T) {
	var stack dart.StackTrace
	func() {
		defer func() {
			recover()
			stack = dart.StackTraceFromPanic()
		}()
		var xs []int
		_ = xs[3] // a runtime panic: index out of range
	}()
	frames := StackFrameFromStackTrace(stack)
	if len(frames) < 2 {
		t.Fatalf("too few frames:\n%s", stack)
	}
	// Frame #0 is the function that panicked (the closure, printed as
	// Dart prints one), located like a Dart frame of package:flutter.
	equal(t, frames[0].Number, 0)
	if !strings.HasPrefix(frames[0].Source, "#0      TestStackTraceOfAGoPanic.<anonymous closure> (package:flutter/src/foundation/stack_frame_test.go:") {
		t.Errorf("frame 0: %q", frames[0].Source)
	}
	equal(t, frames[0].ClassName, "")
	equal(t, frames[0].Method, "TestStackTraceOfAGoPanic")
	equal(t, frames[0].PackageScheme, "package")
	equal(t, frames[0].Package, "flutter")
	equal(t, frames[0].PackagePath, "src/foundation/stack_frame_test.go")
	if frames[0].Line <= 0 || frames[0].Column != -1 {
		t.Errorf("frame 0 line/column: %v", frames[0])
	}
	equal(t, frames[1].Method, "TestStackTraceOfAGoPanic")
	equal(t, frames[1].ClassName, "")
	// StackTraceCurrent starts at its caller.
	current := StackFrameFromStackTrace(dart.StackTraceCurrent())
	equal(t, current[0].Method, "TestStackTraceOfAGoPanic")
	equal(t, current[0].PackagePath, "src/foundation/stack_frame_test.go")
}

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
#10     Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#11     Element.updateChild (package:flutter/src/widgets/framework.dart)
#12     SingleChildRenderObjectElement.mount (package:flutter/blah.dart:999:9)
#13     main (package:hello_flutter/main.dart:10:4)`

const asyncStackString = `#0      getSampleStack.<anonymous closure> (file:///path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart:40:57)
#1      new Future.sync (dart:async/future.dart:224:31)
#2      getSampleStack (file:///path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart:40:10)
#3      main (file:///path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart:46:40)
#4      main (package:flutter_goldens/flutter_goldens.dart:43:17)
<asynchronous suspension>
#5      main.<anonymous closure>.<anonymous closure> (file:///temp/path.whatever/listener.dart:47:18)
#6      _rootRun (dart:async/zone.dart:1126:13)
#7      _CustomZone.run (dart:async/zone.dart:1023:19)
#8      _runZoned (dart:async/zone.dart:1518:10)
#9      runZoned (dart:async/zone.dart:1465:12)
#10     Declarer.declare (package:test_api/src/backend/declarer.dart:130:22)
#11     RemoteListener.start.<anonymous closure>.<anonymous closure>.<anonymous closure> (package:test_api/src/remote_listener.dart:124:26)
<asynchronous suspension>
#12     _rootRun (dart:async/zone.dart:1126:13)
#13     _CustomZone.run (dart:async/zone.dart:1023:19)
#14     _runZoned (dart:async/zone.dart:1518:10)
#15     runZoned (dart:async/zone.dart:1502:12)
#16     RemoteListener.start.<anonymous closure>.<anonymous closure> (package:test_api/src/remote_listener.dart:70:9)
#17     _rootRun (dart:async/zone.dart:1126:13)
#18     _CustomZone.run (dart:async/zone.dart:1023:19)
#19     _runZoned (dart:async/zone.dart:1518:10)
#20     runZoned (dart:async/zone.dart:1465:12)
#21     StackTraceFormatter.asCurrent (package:test_api/src/backend/stack_trace_formatter.dart:41:31)
#22     RemoteListener.start.<anonymous closure> (package:test_api/src/remote_listener.dart:69:29)
#23     _rootRun (dart:async/zone.dart:1126:13)
#24     _CustomZone.run (dart:async/zone.dart:1023:19)
#25     _runZoned (dart:async/zone.dart:1518:10)
#26     runZoned (dart:async/zone.dart:1465:12)
#27     SuiteChannelManager.asCurrent (package:test_api/src/suite_channel_manager.dart:34:31)
#28     RemoteListener.start (package:test_api/src/remote_listener.dart:68:27)
#29     serializeSuite (file:///temp/path.whatever/listener.dart:17:25)
#30     main (file:///temp/path.whatever/listener.dart:43:36)
#31     _runMainZoned.<anonymous closure>.<anonymous closure> (dart:ui/hooks.dart:239:25)
#32     _rootRun (dart:async/zone.dart:1126:13)
#33     _CustomZone.run (dart:async/zone.dart:1023:19)
#34     _runZoned (dart:async/zone.dart:1518:10)
#35     runZoned (dart:async/zone.dart:1502:12)
#36     _runMainZoned.<anonymous closure> (dart:ui/hooks.dart:231:5)
#37     _startIsolate.<anonymous closure> (dart:isolate-patch/isolate_patch.dart:307:19)
#38     _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174:12)`

const mangledStackString = `dart:async/future_impl.dart 23:44                              _Completer.completeError
test\bindings_async_gap_test.dart 42:17                        main.<fn>.<fn>
package:flutter_test/src/binding.dart 744:19                   TestWidgetsFlutterBinding._runTestBody
===== asynchronous gap ===========================
dart:async/zone.dart 1121:19                                   _CustomZone.registerUnaryCallback
dart:async-patch/async_patch.dart 83:23                        _asyncThenWrapperHelper
dart:async/zone.dart 1222:13                                   _rootRunBinary
dart:async/zone.dart 1107:19                                   _CustomZone.runBinary
package:flutter_test/src/binding.dart 724:14                   TestWidgetsFlutterBinding._runTest
package:flutter_test/src/binding.dart 1124:24                  AutomatedTestWidgetsFlutterBinding.runTest.<fn>
package:fake_async/fake_async.dart 177:54                      FakeAsync.run.<fn>.<fn>
dart:async/zone.dart 1190:13                                   _rootRun
`

const stackFrameNoCols = `#0      blah (package:assertions/main.dart:4)
#1      main (package:assertions/main.dart:8)
#2      _runMainZoned.<anonymous closure>.<anonymous closure> (dart:ui/hooks.dart:239)
#3      _rootRun (dart:async/zone.dart:1126)
#4      _CustomZone.run (dart:async/zone.dart:1023)
#5      _runZoned (dart:async/zone.dart:1518)
#6      runZoned (dart:async/zone.dart:1502)
#7      _runMainZoned.<anonymous closure> (dart:ui/hooks.dart:231)
#8      _startIsolate.<anonymous closure> (dart:isolate-patch/isolate_patch.dart:307)
#9      _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174)`

const webStackTrace = `package:dart-sdk/lib/_internal/js_dev_runtime/private/ddc_runtime/errors.dart 196:49  throw_
package:assertions/main.dart 4:3                                                      blah
package:assertions/main.dart 8:5                                                      main$
package:assertions/main_web_entrypoint.dart 9:3                                       main$
package:dart-sdk/lib/_internal/js_dev_runtime/patch/async_patch.dart 47:50            onValue
package:dart-sdk/lib/async/zone.dart 1381:54                                          runUnary
object_test.dart 210:5                                                                performLayout
package:dart-sdk/lib/async/future_impl.dart 140:18                                    handleValue
package:dart-sdk/lib/async/future_impl.dart 682:44                                    handleValueCallback
package:dart-sdk/lib/async/future_impl.dart 711:32                                    _propagateToListeners
package:dart-sdk/lib/async/future_impl.dart 391:9                                     callback
package:dart-sdk/lib/async/schedule_microtask.dart 43:11                              _microtaskLoop
package:dart-sdk/lib/async/schedule_microtask.dart 52:5                               _startMicrotaskLoop
package:dart-sdk/lib/_internal/js_dev_runtime/patch/async_patch.dart 168:15           <fn>`

var stackFrames = []StackFrame{
	{Number: 0, ClassName: "_AssertionError", Method: "_doThrowNew", PackageScheme: "dart", Package: "core-patch", PackagePath: "errors_patch.dart", Line: 42, Column: 39, Source: "#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)"},
	{Number: 1, ClassName: "_AssertionError", Method: "_throwNew", PackageScheme: "dart", Package: "core-patch", PackagePath: "errors_patch.dart", Line: 38, Column: 5, Source: "#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)"},
	{Number: 2, ClassName: "Text", Method: "", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/text.dart", Line: 287, Column: 10, IsConstructor: true, Source: "#2      new Text (package:flutter/src/widgets/text.dart:287:10)"},
	{Number: 3, ClassName: "_MyHomePageState", Method: "build", PackageScheme: "package", Package: "hello_flutter", PackagePath: "main.dart", Line: 72, Column: 16, Source: "#3      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)"},
	{Number: 4, ClassName: "StatefulElement", Method: "build", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: 4414, Column: 27, Source: "#4      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)"},
	{Number: 5, ClassName: "ComponentElement", Method: "performRebuild", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: 4303, Column: 15, Source: "#5      ComponentElement.performRebuild (package:flutter/src/widgets/framework.dart:4303:15)"},
	{Number: 6, ClassName: "Element", Method: "rebuild", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: 4027, Column: 5, Source: "#6      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)"},
	{Number: 7, ClassName: "ComponentElement", Method: "_firstBuild", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: 4286, Column: 5, Source: "#7      ComponentElement._firstBuild (package:flutter/src/widgets/framework.dart:4286:5)"},
	{Number: 8, ClassName: "StatefulElement", Method: "_firstBuild", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: 4461, Column: 11, Source: "#8      StatefulElement._firstBuild (package:flutter/src/widgets/framework.dart:4461:11)"},
	{Number: 9, ClassName: "ComponentElement", Method: "mount", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: 4281, Column: 5, Source: "#9      ComponentElement.mount (package:flutter/src/widgets/framework.dart:4281:5)"},
	{Number: 10, ClassName: "Element", Method: "inflateWidget", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: 3276, Column: 14, Source: "#10     Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)"},
	{Number: 11, ClassName: "Element", Method: "updateChild", PackageScheme: "package", Package: "flutter", PackagePath: "src/widgets/framework.dart", Line: -1, Column: -1, Source: "#11     Element.updateChild (package:flutter/src/widgets/framework.dart)"},
	{Number: 12, ClassName: "SingleChildRenderObjectElement", Method: "mount", PackageScheme: "package", Package: "flutter", PackagePath: "blah.dart", Line: 999, Column: 9, Source: "#12     SingleChildRenderObjectElement.mount (package:flutter/blah.dart:999:9)"},
	{Number: 13, Method: "main", PackageScheme: "package", Package: "hello_flutter", PackagePath: "main.dart", Line: 10, Column: 4, Source: "#13     main (package:hello_flutter/main.dart:10:4)"},
}

var asyncStackFrames = []StackFrame{
	{Number: 0, Method: "getSampleStack", PackageScheme: "file", Package: "<unknown>", PackagePath: "/path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart", Line: 40, Column: 57, Source: "#0      getSampleStack.<anonymous closure> (file:///path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart:40:57)"},
	{Number: 1, ClassName: "Future", Method: "sync", PackageScheme: "dart", Package: "async", PackagePath: "future.dart", Line: 224, Column: 31, IsConstructor: true, Source: "#1      new Future.sync (dart:async/future.dart:224:31)"},
	{Number: 2, Method: "getSampleStack", PackageScheme: "file", Package: "<unknown>", PackagePath: "/path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart", Line: 40, Column: 10, Source: "#2      getSampleStack (file:///path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart:40:10)"},
	{Number: 3, Method: "main", PackageScheme: "file", Package: "<unknown>", PackagePath: "/path/to/flutter/packages/flutter/foundation/error_reporting_test.dart", Line: 46, Column: 40, Source: "#3      main (file:///path/to/flutter/packages/flutter/test/foundation/error_reporting_test.dart:46:40)"},
	{Number: 4, Method: "main", PackageScheme: "package", Package: "flutter_goldens", PackagePath: "flutter_goldens.dart", Line: 43, Column: 17, Source: "#4      main (package:flutter_goldens/flutter_goldens.dart:43:17)"},
	StackFrameAsynchronousSuspension,
	{Number: 5, Method: "main", PackageScheme: "file", Package: "<unknown>", PackagePath: "/temp/path.whatever/listener.dart", Line: 47, Column: 18, Source: "#5      main.<anonymous closure>.<anonymous closure> (file:///temp/path.whatever/listener.dart:47:18)"},
	{Number: 6, Method: "_rootRun", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1126, Column: 13, Source: "#6      _rootRun (dart:async/zone.dart:1126:13)"},
	{Number: 7, ClassName: "_CustomZone", Method: "run", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1023, Column: 19, Source: "#7      _CustomZone.run (dart:async/zone.dart:1023:19)"},
	{Number: 8, Method: "_runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1518, Column: 10, Source: "#8      _runZoned (dart:async/zone.dart:1518:10)"},
	{Number: 9, Method: "runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1465, Column: 12, Source: "#9      runZoned (dart:async/zone.dart:1465:12)"},
	{Number: 10, ClassName: "Declarer", Method: "declare", PackageScheme: "package", Package: "test_api", PackagePath: "src/backend/declarer.dart", Line: 130, Column: 22, Source: "#10     Declarer.declare (package:test_api/src/backend/declarer.dart:130:22)"},
	{Number: 11, ClassName: "RemoteListener", Method: "start", PackageScheme: "package", Package: "test_api", PackagePath: "src/remote_listener.dart", Line: 124, Column: 26, Source: "#11     RemoteListener.start.<anonymous closure>.<anonymous closure>.<anonymous closure> (package:test_api/src/remote_listener.dart:124:26)"},
	StackFrameAsynchronousSuspension,
	{Number: 12, Method: "_rootRun", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1126, Column: 13, Source: "#12     _rootRun (dart:async/zone.dart:1126:13)"},
	{Number: 13, ClassName: "_CustomZone", Method: "run", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1023, Column: 19, Source: "#13     _CustomZone.run (dart:async/zone.dart:1023:19)"},
	{Number: 14, Method: "_runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1518, Column: 10, Source: "#14     _runZoned (dart:async/zone.dart:1518:10)"},
	{Number: 15, Method: "runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1502, Column: 12, Source: "#15     runZoned (dart:async/zone.dart:1502:12)"},
	{Number: 16, ClassName: "RemoteListener", Method: "start", PackageScheme: "package", Package: "test_api", PackagePath: "src/remote_listener.dart", Line: 70, Column: 9, Source: "#16     RemoteListener.start.<anonymous closure>.<anonymous closure> (package:test_api/src/remote_listener.dart:70:9)"},
	{Number: 17, Method: "_rootRun", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1126, Column: 13, Source: "#17     _rootRun (dart:async/zone.dart:1126:13)"},
	{Number: 18, ClassName: "_CustomZone", Method: "run", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1023, Column: 19, Source: "#18     _CustomZone.run (dart:async/zone.dart:1023:19)"},
	{Number: 19, Method: "_runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1518, Column: 10, Source: "#19     _runZoned (dart:async/zone.dart:1518:10)"},
	{Number: 20, Method: "runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1465, Column: 12, Source: "#20     runZoned (dart:async/zone.dart:1465:12)"},
	{Number: 21, ClassName: "StackTraceFormatter", Method: "asCurrent", PackageScheme: "package", Package: "test_api", PackagePath: "src/backend/stack_trace_formatter.dart", Line: 41, Column: 31, Source: "#21     StackTraceFormatter.asCurrent (package:test_api/src/backend/stack_trace_formatter.dart:41:31)"},
	{Number: 22, ClassName: "RemoteListener", Method: "start", PackageScheme: "package", Package: "test_api", PackagePath: "src/remote_listener.dart", Line: 69, Column: 29, Source: "#22     RemoteListener.start.<anonymous closure> (package:test_api/src/remote_listener.dart:69:29)"},
	{Number: 23, Method: "_rootRun", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1126, Column: 13, Source: "#23     _rootRun (dart:async/zone.dart:1126:13)"},
	{Number: 24, ClassName: "_CustomZone", Method: "run", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1023, Column: 19, Source: "#24     _CustomZone.run (dart:async/zone.dart:1023:19)"},
	{Number: 25, Method: "_runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1518, Column: 10, Source: "#25     _runZoned (dart:async/zone.dart:1518:10)"},
	{Number: 26, Method: "runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1465, Column: 12, Source: "#26     runZoned (dart:async/zone.dart:1465:12)"},
	{Number: 27, ClassName: "SuiteChannelManager", Method: "asCurrent", PackageScheme: "package", Package: "test_api", PackagePath: "src/suite_channel_manager.dart", Line: 34, Column: 31, Source: "#27     SuiteChannelManager.asCurrent (package:test_api/src/suite_channel_manager.dart:34:31)"},
	{Number: 28, ClassName: "RemoteListener", Method: "start", PackageScheme: "package", Package: "test_api", PackagePath: "src/remote_listener.dart", Line: 68, Column: 27, Source: "#28     RemoteListener.start (package:test_api/src/remote_listener.dart:68:27)"},
	{Number: 29, Method: "serializeSuite", PackageScheme: "file", Package: "<unknown>", PackagePath: "/temp/path.whatever/listener.dart", Line: 17, Column: 25, Source: "#29     serializeSuite (file:///temp/path.whatever/listener.dart:17:25)"},
	{Number: 30, Method: "main", PackageScheme: "file", Package: "<unknown>", PackagePath: "/temp/path.whatever/listener.dart", Line: 43, Column: 36, Source: "#30     main (file:///temp/path.whatever/listener.dart:43:36)"},
	{Number: 31, Method: "_runMainZoned", PackageScheme: "dart", Package: "ui", PackagePath: "hooks.dart", Line: 239, Column: 25, Source: "#31     _runMainZoned.<anonymous closure>.<anonymous closure> (dart:ui/hooks.dart:239:25)"},
	{Number: 32, Method: "_rootRun", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1126, Column: 13, Source: "#32     _rootRun (dart:async/zone.dart:1126:13)"},
	{Number: 33, ClassName: "_CustomZone", Method: "run", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1023, Column: 19, Source: "#33     _CustomZone.run (dart:async/zone.dart:1023:19)"},
	{Number: 34, Method: "_runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1518, Column: 10, Source: "#34     _runZoned (dart:async/zone.dart:1518:10)"},
	{Number: 35, Method: "runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1502, Column: 12, Source: "#35     runZoned (dart:async/zone.dart:1502:12)"},
	{Number: 36, Method: "_runMainZoned", PackageScheme: "dart", Package: "ui", PackagePath: "hooks.dart", Line: 231, Column: 5, Source: "#36     _runMainZoned.<anonymous closure> (dart:ui/hooks.dart:231:5)"},
	{Number: 37, Method: "_startIsolate", PackageScheme: "dart", Package: "isolate-patch", PackagePath: "isolate_patch.dart", Line: 307, Column: 19, Source: "#37     _startIsolate.<anonymous closure> (dart:isolate-patch/isolate_patch.dart:307:19)"},
	{Number: 38, ClassName: "_RawReceivePortImpl", Method: "_handleMessage", PackageScheme: "dart", Package: "isolate-patch", PackagePath: "isolate_patch.dart", Line: 174, Column: 12, Source: "#38     _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174:12)"},
}

var stackFrameNoColsFrames = []StackFrame{
	{Number: 0, Method: "blah", PackageScheme: "package", Package: "assertions", PackagePath: "main.dart", Line: 4, Column: -1, Source: "#0      blah (package:assertions/main.dart:4)"},
	{Number: 1, Method: "main", PackageScheme: "package", Package: "assertions", PackagePath: "main.dart", Line: 8, Column: -1, Source: "#1      main (package:assertions/main.dart:8)"},
	{Number: 2, Method: "_runMainZoned", PackageScheme: "dart", Package: "ui", PackagePath: "hooks.dart", Line: 239, Column: -1, Source: "#2      _runMainZoned.<anonymous closure>.<anonymous closure> (dart:ui/hooks.dart:239)"},
	{Number: 3, Method: "_rootRun", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1126, Column: -1, Source: "#3      _rootRun (dart:async/zone.dart:1126)"},
	{Number: 4, ClassName: "_CustomZone", Method: "run", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1023, Column: -1, Source: "#4      _CustomZone.run (dart:async/zone.dart:1023)"},
	{Number: 5, Method: "_runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1518, Column: -1, Source: "#5      _runZoned (dart:async/zone.dart:1518)"},
	{Number: 6, Method: "runZoned", PackageScheme: "dart", Package: "async", PackagePath: "zone.dart", Line: 1502, Column: -1, Source: "#6      runZoned (dart:async/zone.dart:1502)"},
	{Number: 7, Method: "_runMainZoned", PackageScheme: "dart", Package: "ui", PackagePath: "hooks.dart", Line: 231, Column: -1, Source: "#7      _runMainZoned.<anonymous closure> (dart:ui/hooks.dart:231)"},
	{Number: 8, Method: "_startIsolate", PackageScheme: "dart", Package: "isolate-patch", PackagePath: "isolate-patch.dart", Line: 307, Column: -1, Source: "#8      _startIsolate.<anonymous closure> (dart:isolate-patch/isolate_patch.dart:307)"},
	{Number: 9, ClassName: "_RawReceivePortImpl", Method: "_handleMessage", PackageScheme: "dart", Package: "isolate-patch", PackagePath: "isolate-patch.dart", Line: 174, Column: -1, Source: "#9      _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174)"},
}

var webStackTraceFrames = []StackFrame{
	{Number: -1, ClassName: "<unknown>", Method: "throw_", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/_internal/js_dev_runtime/private/ddc_runtime/errors.dart", Line: 196, Column: 49, Source: "package:dart-sdk/lib/_internal/js_dev_runtime/private/ddc_runtime/errors.dart 196:49  throw_"},
	{Number: -1, ClassName: "<unknown>", Method: "blah", PackageScheme: "package", Package: "assertions", PackagePath: "main.dart", Line: 4, Column: 3, Source: "package:assertions/main.dart 4:3                                                      blah"},
	{Number: -1, ClassName: "<unknown>", Method: "main$", PackageScheme: "package", Package: "assertions", PackagePath: "main.dart", Line: 8, Column: 5, Source: "package:assertions/main.dart 8:5                                                      main$"},
	{Number: -1, ClassName: "<unknown>", Method: "main$", PackageScheme: "package", Package: "assertions", PackagePath: "main_web_entrypoint.dart", Line: 9, Column: 3, Source: "package:assertions/main_web_entrypoint.dart 9:3                                       main$"},
	{Number: -1, ClassName: "<unknown>", Method: "onValue", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/_internal/js_dev_runtime/patch/async_patch.dart", Line: 47, Column: 50, Source: "package:dart-sdk/lib/_internal/js_dev_runtime/patch/async_patch.dart 47:50            onValue"},
	{Number: -1, ClassName: "<unknown>", Method: "runUnary", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/async/zone.dart", Line: 1381, Column: 54, Source: "package:dart-sdk/lib/async/zone.dart 1381:54                                          runUnary"},
	{Number: -1, ClassName: "<unknown>", Method: "performLayout", PackageScheme: "<unknown>", Package: "<unknown>", PackagePath: "<unknown>", Line: 210, Column: 5, Source: "object_test.dart 210:5                                                                performLayout"},
	{Number: -1, ClassName: "<unknown>", Method: "handleValue", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/async/future_impl.dart", Line: 140, Column: 18, Source: "package:dart-sdk/lib/async/future_impl.dart 140:18                                    handleValue"},
	{Number: -1, ClassName: "<unknown>", Method: "handleValueCallback", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/async/future_impl.dart", Line: 682, Column: 44, Source: "package:dart-sdk/lib/async/future_impl.dart 682:44                                    handleValueCallback"},
	{Number: -1, ClassName: "<unknown>", Method: "_propagateToListeners", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/async/future_impl.dart", Line: 711, Column: 32, Source: "package:dart-sdk/lib/async/future_impl.dart 711:32                                    _propagateToListeners"},
	{Number: -1, ClassName: "<unknown>", Method: "callback", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/async/future_impl.dart", Line: 391, Column: 9, Source: "package:dart-sdk/lib/async/future_impl.dart 391:9                                     callback"},
	{Number: -1, ClassName: "<unknown>", Method: "_microtaskLoop", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/async/schedule_microtask.dart", Line: 43, Column: 11, Source: "package:dart-sdk/lib/async/schedule_microtask.dart 43:11                              _microtaskLoop"},
	{Number: -1, ClassName: "<unknown>", Method: "_startMicrotaskLoop", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/async/schedule_microtask.dart", Line: 52, Column: 5, Source: "package:dart-sdk/lib/async/schedule_microtask.dart 52:5                               _startMicrotaskLoop"},
	{Number: -1, ClassName: "<unknown>", Method: "<fn>", PackageScheme: "package", Package: "dart-sdk", PackagePath: "lib/_internal/js_dev_runtime/patch/async_patch.dart", Line: 168, Column: 15, Source: "package:dart-sdk/lib/_internal/js_dev_runtime/patch/async_patch.dart 168:15           <fn>"},
}
