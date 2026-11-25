package difftest

// Go twin of flutter/test/assertions_test.dart: the same FlutterError and
// FlutterErrorDetails reports, StackFrame parses, stack filters,
// debugWordWrap results and debugPrintThrottled schedule, compared with
// Flutter's EXACTLY (object hashes normalized to #00000, as in
// diagnostics_test.go; JSON compared as decoded values).
//
// Dart exception classes have Go twins with the same runtime type name and
// toString (MyException, MyError, Foo). Dart's assert(false, message)
// produces the VM's _AssertionError, whose text includes the Dart test
// file's path and position; the Go twin is built from that text as Flutter
// recorded it (exception.<kind>.raw), so both sides reformat the same input.

import (
	"encoding/json"
	"iter"
	"os"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	f "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

type MyException struct{ text string }

func (e *MyException) Error() string { return e.text }

type MyError struct{ text string }

func (e *MyError) Error() string { return e.text }

type Foo struct{ n int }

func (x *Foo) String() string { return "Foo(" + strconv.Itoa(x.n) + ")" }

// Widgetish is a Diagnosticable used as an expandable property value.
type Widgetish struct {
	f.DiagnosticableMixin
	name string
}

func newWidgetish(name string) *Widgetish {
	w := &Widgetish{name: name}
	w.SetSelf(w)
	return w
}

func (w *Widgetish) ToStringShort() string { return "Widgetish(" + w.name + ")" }

func (w *Widgetish) DebugFillProperties(properties *f.DiagnosticPropertiesBuilder) {
	w.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(f.NewStringProperty("name", w.name))
	properties.Add(f.NewDoubleProperty("width", 12.5))
	properties.Add(f.NewFlagProperty("visible", true, f.FlagPropertyOptions{IfTrue: ptr("visible")}))
}

// _AssertionError is the Dart VM's _AssertionError (see the file comment).
type _AssertionError struct {
	text    string
	message any
}

func (e *_AssertionError) Error() string         { return e.text }
func (e *_AssertionError) AssertionMessage() any { return e.message }

var vmAssertRE = regexp.MustCompile(`(?s)^'.*': Failed assertion: line [0-9]+ pos [0-9]+: 'false': (.*)$`)

// realAssert rebuilds the _AssertionError Flutter recorded (message nil when
// the text ends with "is not true.").
func realAssert(t *testing.T, raw string, hasMessage bool) any {
	m := vmAssertRE.FindStringSubmatch(raw)
	if m == nil {
		t.Fatalf("unexpected assert text %q", raw)
	}
	var message any
	if hasMessage {
		message = m[1]
	}
	return &_AssertionError{text: raw, message: message}
}

const flutterStack = `#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      new SomeWidgetUsingText (package:flutter/src/widgets/text_helper.dart:287:10)
#4      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#5      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)
#6      ComponentElement.performRebuild (package:flutter/src/widgets/framework.dart:4303:15)
#7      Element.rebuild (package:flutter/src/widgets/framework.dart:4027:5)
#8      ComponentElement._firstBuild (package:flutter/src/widgets/framework.dart:4286:5)
#9      Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#10     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#11     Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#12     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#13     _rootRun (dart:async/zone.dart:1126:13)
#14     _CustomZone.run (dart:async/zone.dart:1023:19)
#15     _FakeAsync.run (package:fake_async/fake_async.dart:177:54)
#16     Chain.capture (package:stack_trace/src/chain.dart:94:22)
#17     _asyncThenWrapperHelper (dart:async-patch/async_patch.dart:83:23)
#18     _Timer._runTimers (dart:isolate-patch/timer_impl.dart:398:19)
#19     _FrameCallbackEntry.run (package:flutter/src/scheduler/binding.dart:1:1)
<asynchronous suspension>
#20     main (file:///app/main.dart:10:4)
#21     _RawReceivePortImpl._handleMessage (dart:isolate-patch/isolate_patch.dart:174:12)
...
#22     Element.inflateWidget (package:flutter/src/widgets/framework.dart:3276:14)
#23     Element.updateChild (package:flutter/src/widgets/framework.dart:3070:12)
#24     tail (package:app/tail.dart:1:1)`

const userStack = `#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)
#1      _AssertionError._throwNew (dart:core-patch/errors_patch.dart:38:5)
#2      new Text (package:flutter/src/widgets/text.dart:287:10)
#3      _MyHomePageState.build (package:hello_flutter/main.dart:72:16)
#4      StatefulElement.build (package:flutter/src/widgets/framework.dart:4414:27)`

var frameLines = []string{
	"#0      _AssertionError._doThrowNew (dart:core-patch/errors_patch.dart:42:39)",
	"#1      new Text (package:flutter/src/widgets/text.dart:287:10)",
	"#2      new Text.rich (package:flutter/src/widgets/text.dart:300:1)",
	"#3      new (http://localhost:42191/dart-sdk/lib/async/stream_controller.dart:880:9)",
	"#4      main.<anonymous closure>.<anonymous closure> (file:///temp/path.whatever/listener.dart:47:18)",
	"#5      A.b.c.d (package:x/y/z.dart:1:2)",
	"#6      blah (package:assertions/main.dart:4)",
	"#7      noLine (package:assertions/main.dart)",
	"#8      dots (dart:ui/../ui/./hooks.dart:9:9)",
	"#9      spaces (file:///my%20dir/a%2fb.dart:3:4)",
	"#10     weird (file:///a/b c.dart:3:4)",
	"#11     relative (a/../../b.dart:1:1)",
	"#12     upper (PACKAGE:Foo/Bar.dart:1:1)",
	"#123456 big (package:p/q.dart:12345:67890)",
	"#0      someMethod  (file:///diagnostics_test.dart:42:19)",
	"<asynchronous suspension>",
	"===== asynchronous gap ===========================",
	"...",
	"package:dart-sdk/lib/_internal/js_dev_runtime/private/ddc_runtime/errors.dart 196:49  throw_",
	"package:assertions/main.dart 8:5                                                      main$",
	"object_test.dart 210:5                                                                performLayout",
	`test\bindings_async_gap_test.dart 42:17                        main.<fn>.<fn>`,
	"not a frame at all",
	"    at Foo.bar (http://x/main.dart.js:1:2)",
	"#13     paren (file:///dir (copy)/x.dart:1:2)",
	"#14     new A.b.c (package:x/y.dart:1:1)",
	"a\u2028b.dart 1:2 lineSeparator",
	"nbsp.dart 1:2\u00a0method",
}

const loremIpsum = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore " +
	"et dolore magna aliqua.  Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut " +
	"aliquip ex ea commodo consequat. Supercalifragilisticexpialidocious-is-an-unbreakable-word-here. " +
	"Ünïcödé àccents, emoji 😀😀 and CJK 漢字漢字 too."

var wrapTexts = []string{
	loremIpsum,
	"  - a list item whose text goes on for quite a while so that it has to wrap onto further lines",
	"12. a numbered item whose text goes on for quite a while so that it has to wrap onto lines",
	"* bullet     with     many     spaces     between     the     words     of     this     line",
	"   #0      main (file:///a.dart:1:2) frames are never wrapped even when they are long",
	"short",
	"exactly twenty chars",
	"    leading spaces then words that need wrapping because the width is small",
	"trailing spaces after words that need wrapping because the width is small      ",
	"😀😀😀 😀😀😀 😀😀😀 😀😀😀 😀😀😀",
	"a b c d e f g h i j k l m n o p q r s t u v w x y z",
	"anunbreakablewordlongerthananywidthwetryhere and then short words",
}

var wrapWidths = []int{0, 1, 3, 7, 10, 15, 20, 21, 40, 80}

func richError() *f.FlutterError {
	return f.NewFlutterErrorFromParts([]f.DiagnosticsNode{
		f.NewErrorSummary("Something is wrong with the widget configuration."),
		f.NewErrorDescription("This description is long enough that the error renderer has to wrap it onto several lines " +
			"when it is shown at the default wrap width of a hundred characters."),
		f.NewErrorSpacer(),
		f.NewDiagnosticsProperty("The offending widget was", newWidgetish("Inner"), f.DiagnosticsPropertyOptions{Style: st(f.DiagnosticsTreeStyleErrorProperty)}),
		f.NewDiagnosticsProperty("Expanded", newWidgetish("Exp"), f.DiagnosticsPropertyOptions{ExpandableValue: true, Style: st(f.DiagnosticsTreeStyleWhitespace)}),
		f.NewErrorHint("Try wrapping the widget in something else.\nOr pass a different argument."),
		f.NewErrorSpacer(),
		f.NewErrorDescription("A final note."),
		f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
			Name:       ptr("Block of details"),
			Style:      st(f.DiagnosticsTreeStyleErrorProperty),
			Properties: []f.DiagnosticsNode{f.NewIntProperty("count", 3), f.NewStringProperty("label", "abc")},
		}),
		f.NewDiagnosticsStackTrace("Stack of the creation", dart.StackTraceFromString(userStack)),
	})
}

type exceptionKind struct {
	name string
	make func() any
}

func exceptionKinds(t *testing.T, ref map[string]any) []exceptionKind {
	raw := func(name string) string { s, _ := ref["exception."+name+".raw"].(string); return s }
	return []exceptionKind{
		{"string", func() any { return "a thrown string" }},
		{"multilineString", func() any { return "  indented first line\nsecond line\n\n" }},
		{"int", func() any { return 42 }},
		{"double", func() any { return 2.5 }},
		{"doubleWhole", func() any { return 2.0 }},
		{"bool", func() any { return true }},
		{"assertion", func() any { return dart.NewAssertionError("Test assertion") }},
		{"assertionNoMessage", func() any { return dart.NewAssertionError(nil) }},
		{"assertionNumber", func() any { return dart.NewAssertionError(7) }},
		{"assertionQuoted", func() any { return dart.NewAssertionError("quote \" and \\ and\nnewline\ttab") }},
		{"assertionFlutterError", func() any {
			return dart.NewAssertionError(f.NewFlutterError("Wrapped summary.\nWrapped description."))
		}},
		{"flutterError", func() any { return f.NewFlutterError("Plain summary.\nFirst description.\nSecond description.") }},
		{"richFlutterError", func() any { return richError() }},
		{"myException", func() any { return &MyException{"MyException: went wrong"} }},
		{"myExceptionNoPrefix", func() any { return &MyException{"went wrong without prefix"} }},
		{"myError", func() any { return &MyError{"Custom error text\nwith a second line"} }},
		{"emptyError", func() any { return &MyError{""} }},
		{"foo", func() any { return &Foo{1} }},
		{"realAssert", func() any { return realAssert(t, raw("realAssert"), true) }},
		{"realAssertNoMessage", func() any { return realAssert(t, raw("realAssertNoMessage"), false) }},
		{"realAssertLong", func() any { return realAssert(t, raw("realAssertLong"), true) }},
	}
}

func captureDebugPrint(fn func()) []string {
	log := []string{}
	saved := f.DebugPrint
	f.DebugPrint = func(message string, _ ...f.DebugPrintOptions) { log = append(log, message) }
	defer func() { f.DebugPrint = saved }()
	fn()
	return log
}

func capturePrint(fn func()) []string {
	log := []string{}
	defer dart.SetPrintHandler(func(line string) { log = append(log, line) })()
	fn()
	return log
}

func frameJSON(fr f.StackFrame) map[string]any {
	return map[string]any{
		"number":        fr.Number,
		"packageScheme": fr.PackageScheme,
		"package":       fr.Package,
		"packagePath":   fr.PackagePath,
		"line":          fr.Line,
		"column":        fr.Column,
		"className":     fr.ClassName,
		"method":        fr.Method,
		"isConstructor": fr.IsConstructor,
		"source":        fr.Source,
		"toString":      fr.String(),
	}
}

// runtimeType is a node's Dart runtimeType, as its JSON "type" spells it.
func runtimeType(n f.DiagnosticsNode) string {
	return n.ToJsonMap(f.NewDiagnosticsSerializationDelegate())["type"].(string)
}

func propertyStrings(details *f.FlutterErrorDetails) []string {
	builder := f.NewDiagnosticPropertiesBuilder()
	details.DebugFillProperties(builder)
	out := []string{}
	for _, p := range builder.Properties {
		out = append(out, runtimeType(p)+" "+p.Level().String()+" "+p.Style().String()+": "+p.String())
	}
	return out
}

func filterList(lines []string) []string {
	return slices.Collect(f.FlutterErrorDefaultStackFilter(slices.Values(lines)))
}

func throttleEvents() []string {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	events := []string{}
	ms := func() string { return strconv.FormatInt(loop.Elapsed().Milliseconds(), 10) + "ms " }
	log := func(what string) { events = append(events, ms()+what) }
	defer dart.SetPrintHandler(func(line string) {
		u := []rune(line)
		head := string(u[:min(3, len(u))])
		events = append(events, ms()+"print "+strconv.Itoa(len(line))+" "+head)
	})()
	done := func(what string) { f.DebugPrintDone().OnComplete(func(async.Void) { log(what) }, nil) }
	done("done (idle)")
	loop.FlushMicrotasks()
	f.DebugPrintThrottled(strings.Repeat("a", 5000) + "\n" + strings.Repeat("b", 5000) + "\n" + strings.Repeat("c", 5000) + "\n" + strings.Repeat("d", 3000))
	done("done 1")
	f.DebugPrintThrottled(strings.Repeat("e", 20000) + "\nf\ng")
	w := 20
	f.DebugPrintThrottled(strings.Repeat("h ", 30), f.DebugPrintOptions{WrapWidth: &w})
	loop.Elapse(500 * time.Millisecond)
	log("half a second")
	loop.Elapse(500 * time.Millisecond)
	loop.Elapse(3 * time.Second)
	done("done (idle again)")
	loop.FlushMicrotasks()
	f.DebugPrintThrottled(strings.Repeat("i", 13000) + "\nj")
	loop.Elapse(2 * time.Second)
	return events
}

func jsonOf(m map[string]any) any {
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		panic(err)
	}
	return v
}

func buildAssertionScenes(t *testing.T, ref map[string]any) map[string]any {
	out := map[string]any{}

	// The throttle first, while its state is fresh.
	out["throttle.events"] = throttleEvents()

	// FlutterError.
	plain := f.NewFlutterError("Plain summary.\nFirst description.\n\nThird description.")
	out["flutterError.plain.toString"] = plain.String()
	out["flutterError.plain.toStringDeep"] = plain.ToStringDeep()
	out["flutterError.plain.toStringShort"] = plain.ToStringShort()
	out["flutterError.plain.message"] = plain.Message()
	rich := richError()
	out["flutterError.rich.toString"] = rich.String()
	out["flutterError.rich.toStringDeep"] = rich.ToStringDeep()
	out["flutterError.rich.toStringDeep.wrap40"] = rich.ToStringDeep(f.DiagnosticableTreeToStringDeepOptions{WrapWidth: ptr(40)})
	out["flutterError.rich.toStringShallow"] = rich.ToStringShallow(f.ToStringShallowOptions{Joiner: ptr(" | ")})
	diags := []string{}
	for _, d := range rich.Diagnostics {
		diags = append(diags, runtimeType(d)+" "+d.Level().String()+": "+d.String())
	}
	out["flutterError.rich.diagnostics"] = diags
	out["flutterError.rich.json"] = jsonOf(rich.ToDiagnosticsNode().ToJsonMap(
		f.NewDiagnosticsSerializationDelegate(f.DiagnosticsSerializationDelegateOptions{IncludeProperties: true, SubtreeDepth: 1})))
	out["errorSummary.json"] = jsonOf(f.NewErrorSummary("A summary").ToJsonMap(f.NewDiagnosticsSerializationDelegate()))
	out["errorHint.json"] = jsonOf(f.NewErrorHint("A hint").ToJsonMap(f.NewDiagnosticsSerializationDelegate()))
	out["errorSpacer.json"] = jsonOf(f.NewErrorSpacer().ToJsonMap(f.NewDiagnosticsSerializationDelegate()))
	out["errorSpacer.toString"] = f.NewErrorSpacer().String()
	out["errorDescription.dense"] = f.NewErrorDescription("dense desc").ToString(f.DiagnosticsNodeToStringOptions{ParentConfiguration: f.DenseTextConfiguration})

	// Malformed errors.
	malformed := []struct {
		name  string
		parts []f.DiagnosticsNode
	}{
		{"empty", []f.DiagnosticsNode{}},
		{"noSummary", []f.DiagnosticsNode{f.NewErrorDescription("Error description without a summary")}},
		{"twoSummaries", []f.DiagnosticsNode{
			f.NewErrorSummary("Error Summary A"),
			f.NewErrorDescription("Some descriptionA"),
			f.NewErrorSummary("Error Summary B"),
			f.NewErrorHint("A hint B"),
		}},
		{"summaryLast", []f.DiagnosticsNode{f.NewErrorDescription("Some description"), f.NewErrorSummary("Error summary")}},
	}
	for _, m := range malformed {
		var err any
		func() {
			defer func() { err = recover() }()
			panic(f.NewFlutterErrorFromParts(m.parts))
		}()
		if _, ok := err.(dart.AssertionFailure); !ok {
			t.Errorf("malformed %s: not an AssertionError: %#v", m.name, err)
		}
		out["malformed."+m.name+".details"] = f.NewFlutterErrorDetails(err).String()
		out["malformed."+m.name+".summary"] = f.NewFlutterErrorDetails(err).Summary().String()
	}

	// Exception kinds.
	for _, k := range exceptionKinds(t, ref) {
		e := k.make()
		if strings.HasPrefix(k.name, "realAssert") {
			out["exception."+k.name+".raw"] = dartString(e)
		}
		details := f.NewFlutterErrorDetails(e)
		out["exception."+k.name+".toString"] = details.String()
		out["exception."+k.name+".summary"] = details.Summary().String()
		out["exception."+k.name+".exceptionAsString"] = details.ExceptionAsString()
		out["exception."+k.name+".properties"] = propertyStrings(details)
		out["exception."+k.name+".dump"] = captureDebugPrint(func() {
			f.FlutterErrorResetErrorCount()
			f.FlutterErrorDumpErrorToConsole(details)
			f.FlutterErrorDumpErrorToConsole(details)
		})
		withStack := f.NewFlutterErrorDetails(e, f.FlutterErrorDetailsOptions{
			Stack:   dart.StackTraceFromString(flutterStack),
			Library: ptr("kinds library"),
			Context: f.NewErrorDescription("while testing " + k.name),
		})
		out["exception."+k.name+".withStack"] = withStack.String()
		out["exception."+k.name+".withStack.properties"] = propertyStrings(withStack)
	}
	f.FlutterErrorResetErrorCount()

	// Full reports.
	full := f.NewFlutterErrorDetails(
		dart.NewAssertionError("A failed assertion message that is fairly long, long enough to wrap when it is dumped."),
		f.FlutterErrorDetailsOptions{
			Stack:   dart.StackTraceFromString(flutterStack),
			Library: ptr("widgets library"),
			Context: f.NewErrorDescription("while building MyWidget(dirty, state: _MyWidgetState#12345)"),
			InformationCollector: func() iter.Seq[f.DiagnosticsNode] {
				return slices.Values([]f.DiagnosticsNode{
					f.NewDiagnosticsProperty("The relevant error-causing widget was", newWidgetish("Relevant")),
					f.NewErrorHint("Line one of a hint.\nLine two of a hint."),
					f.NewErrorDescription(strings.Repeat("word ", 40)),
					f.NewDiagnosticsStackTraceSingleFrame("Created at", "#0      creator (package:app/creator.dart:1:2)"),
					f.NewDiagnosticsProperty("Long", newWidgetish(strings.Repeat("a ", 80)),
						f.DiagnosticsPropertyOptions{ExpandableValue: true, Style: st(f.DiagnosticsTreeStyleWhitespace)}),
					f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
						Name:          ptr("Many children"),
						AllowTruncate: true,
						Children:      manyChildren(9),
					}),
				})
			},
		})
	out["full.toString"] = full.String()
	out["full.toString.debug"] = full.ToString(f.DiagnosticableToStringOptions{MinLevel: lv(f.DiagnosticLevelDebug)})
	out["full.toStringShort"] = full.ToStringShort()
	out["full.summary"] = full.Summary().String()
	out["full.dump"] = captureDebugPrint(func() {
		f.FlutterErrorResetErrorCount()
		f.FlutterErrorDumpErrorToConsole(full)
		f.FlutterErrorDumpErrorToConsole(full)
		f.FlutterErrorDumpErrorToConsole(full, f.DumpErrorToConsoleOptions{ForceReport: true})
		f.FlutterErrorDumpErrorToConsole(f.NewFlutterErrorDetails(richError()))
		f.FlutterErrorResetErrorCount()
		f.FlutterErrorDumpErrorToConsole(f.NewFlutterErrorDetails(richError(), f.FlutterErrorDetailsOptions{Library: f.NullString, Silent: true}))
	})
	out["full.json"] = jsonOf(full.ToDiagnosticsNode().ToJsonMap(
		f.NewDiagnosticsSerializationDelegate(f.DiagnosticsSerializationDelegateOptions{IncludeProperties: true})))
	out["full.properties"] = propertyStrings(full)
	userFault := f.NewFlutterErrorDetails(dart.NewAssertionError("user"), f.FlutterErrorDetailsOptions{Stack: dart.StackTraceFromString(userStack)})
	out["userFault.toString"] = userFault.String()
	out["userFault.properties"] = propertyStrings(userFault)
	out["customFilter.toString"] = f.NewFlutterErrorDetails("filtered", f.FlutterErrorDetailsOptions{
		Stack: dart.StackTraceFromString(flutterStack),
		StackFilter: func(lines iter.Seq[string]) iter.Seq[string] {
			return func(yield func(string) bool) {
				for l := range lines {
					if strings.Contains(l, "Element") && !yield("> "+l) {
						return
					}
				}
			}
		},
	}).String()
	out["emptyStack.toString"] = f.NewFlutterErrorDetails("x", f.FlutterErrorDetailsOptions{Stack: dart.StackTraceFromString("")}).String()
	summaryFirst := f.NewFlutterErrorDetails(f.NewFlutterError("The real summary.\nIts description."), f.FlutterErrorDetailsOptions{
		InformationCollector: func() iter.Seq[f.DiagnosticsNode] {
			return slices.Values([]f.DiagnosticsNode{f.NewErrorSummary("A later summary.")})
		},
	})
	out["summaryFirst.summary"] = summaryFirst.Summary().String()
	copied := full.CopyWith(f.FlutterErrorDetailsCopyWithOptions{Exception: "copied", Library: ptr("copy lib")})
	out["copyWith.toString"] = copied.String()

	f.FlutterErrorDetailsPropertiesTransformers = append(f.FlutterErrorDetailsPropertiesTransformers,
		func(properties iter.Seq[f.DiagnosticsNode]) iter.Seq[f.DiagnosticsNode] {
			result := []f.DiagnosticsNode{f.NewErrorDescription("(transformed)")}
			for p := range properties {
				if _, spacer := p.(*f.ErrorSpacer); !spacer {
					result = append(result, p)
				}
			}
			return slices.Values(result)
		})
	out["transformed.toString"] = full.String()
	f.FlutterErrorDetailsPropertiesTransformers = nil

	// StackFrame parsing.
	lines := []any{}
	for _, line := range frameLines {
		if fr, ok := f.StackFrameFromStackTraceLine(line); ok {
			lines = append(lines, frameJSON(fr))
		} else {
			lines = append(lines, nil)
		}
	}
	out["frames.lines"] = lines
	frames := []any{}
	for _, fr := range f.StackFrameFromStackString(flutterStack) {
		frames = append(frames, frameJSON(fr))
	}
	out["frames.flutterStack"] = frames

	// Stack filters and DiagnosticsStackTrace.
	out["filter.flutterStack"] = filterList(strings.Split(flutterStack, "\n"))
	out["filter.oneElided"] = filterList([]string{
		"#0      a (package:app/a.dart:1:1)",
		"#1      _rootRun (dart:async/zone.dart:1:1)",
	})
	out["filter.twoKinds"] = filterList([]string{
		"#0      _rootRun (dart:async/zone.dart:1:1)",
		"#1      a (package:app/a.dart:1:1)",
		"#2      b (dart:async-patch/x.dart:1:1)",
		"#3      c (dart:async-patch/x.dart:1:1)",
	})
	out["filter.nothing"] = filterList([]string{"#0      a (package:app/a.dart:1:1)", "not a frame", ""})
	out["stackTrace.deep"] = f.NewDiagnosticsStackTrace("Stack trace", dart.StackTraceFromString(flutterStack)).ToStringDeep()
	out["stackTrace.noSeparator"] = f.NewDiagnosticsStackTrace("-- callback --", dart.StackTraceFromString(userStack),
		f.DiagnosticsStackTraceOptions{ShowSeparator: ptr(false)}).ToStringDeep()
	out["stackTrace.null"] = f.NewDiagnosticsStackTrace("Nothing", nil).ToStringDeep()
	out["stackTrace.singleFrame"] = f.NewDiagnosticsStackTraceSingleFrame("One", "#0      f (package:a/b.dart:1:2)").ToStringDeep()
	out["stackTrace.json"] = jsonOf(f.NewDiagnosticsStackTrace("S", dart.StackTraceFromString(userStack)).ToJsonMap(
		f.NewDiagnosticsSerializationDelegate(f.DiagnosticsSerializationDelegateOptions{IncludeProperties: true})))
	out["debugPrintStack"] = captureDebugPrint(func() {
		f.DebugPrintStack(f.DebugPrintStackOptions{StackTrace: dart.StackTraceFromString(flutterStack), Label: ptr("The label"), MaxFrames: ptr(15)})
		f.DebugPrintStack(f.DebugPrintStackOptions{StackTrace: dart.StackTraceFromString(userStack)})
	})

	// Registered filters last: FlutterErrorAddDefaultStackFilter can't be
	// undone.
	framework := dart.StringPattern("package:flutter/src/widgets/framework.dart")
	f.FlutterErrorAddDefaultStackFilter(f.RepetitiveStackFrameFilter{
		Frames: []f.PartialStackFrame{
			{Package: framework, ClassName: "Element", Method: "inflateWidget"},
			{Package: framework, ClassName: "Element", Method: "updateChild"},
		},
		Replacement: "...     Normal element mounting",
	})
	f.FlutterErrorAddDefaultStackFilter(f.RepetitiveStackFrameFilter{
		Frames:      []f.PartialStackFrame{{Package: regexp.MustCompile(`^package:hello_`), ClassName: "_MyHomePageState", Method: "build"}},
		Replacement: "...     my build",
	})
	f.FlutterErrorAddDefaultStackFilter(f.RepetitiveStackFrameFilter{
		Frames:      []f.PartialStackFrame{f.PartialStackFrameAsynchronousSuspension},
		Replacement: "...     async gap",
	})
	ov := dart.StringPattern("package:ov/")
	f.FlutterErrorAddDefaultStackFilter(f.RepetitiveStackFrameFilter{
		Frames: []f.PartialStackFrame{
			{Package: ov, ClassName: "Ov", Method: "a"},
			{Package: ov, ClassName: "Ov", Method: "b"},
			{Package: ov, ClassName: "Ov", Method: "a"},
		},
		Replacement: "...     overlapping",
	})
	out["repetitive.overlapping"] = filterList([]string{
		"#0      Ov.a (package:ov/o.dart:1:1)",
		"#1      Ov.b (package:ov/o.dart:2:1)",
		"#2      Ov.a (package:ov/o.dart:1:1)",
		"#3      Ov.b (package:ov/o.dart:2:1)",
		"#4      Ov.a (package:ov/o.dart:1:1)",
		"#5      Ov.b (package:ov/o.dart:2:1)",
		"#6      Ov.a (package:ov/o.dart:1:1)",
		"#7      tail (package:app/tail.dart:1:1)",
		"#8      tail (package:app/tail.dart:1:1)",
	})
	out["repetitive.flutterStack"] = filterList(strings.Split(flutterStack, "\n"))
	out["repetitive.details"] = f.NewFlutterErrorDetails("with filters", f.FlutterErrorDetailsOptions{Stack: dart.StackTraceFromString(flutterStack)}).String()

	// debugWordWrap and debugPrint wrapping.
	for i, text := range wrapTexts {
		for _, w := range wrapWidths {
			out["wrap."+strconv.Itoa(i)+"."+strconv.Itoa(w)] = slices.Collect(f.DebugWordWrap(text, w))
		}
		out["wrap."+strconv.Itoa(i)+".indent"] = slices.Collect(f.DebugWordWrap(text, 25, f.DebugWordWrapOptions{WrapIndent: ">> "}))
	}
	for _, c := range []struct {
		text   string
		w      int
		indent string
	}{{"1. a", 0, "  "}, {"   ", 3, ""}, {"   ", 4, ""}} {
		key := "wrap.throws." + strconv.Quote(c.text) + "." + strconv.Itoa(c.w)
		func() {
			defer func() {
				if recover() != nil {
					out[key] = "throws"
				}
			}()
			out[key] = slices.Collect(f.DebugWordWrap(c.text, c.w, f.DebugWordWrapOptions{WrapIndent: c.indent}))
		}()
	}
	out["debugPrintSynchronously"] = capturePrint(func() {
		f.DebugPrintSynchronously(strings.Join(wrapTexts, "\n"), f.DebugPrintOptions{WrapWidth: ptr(30)})
		f.DebugPrintSynchronously(loremIpsum)
		f.DebugPrintSynchronously("Hello,   world", f.DebugPrintOptions{WrapWidth: ptr(0)})
	})
	return out
}

func manyChildren(n int) []f.DiagnosticsNode {
	out := []f.DiagnosticsNode{}
	for i := range n {
		out = append(out, f.NewDiagnosticsNodeMessage("child "+strconv.Itoa(i)))
	}
	return out
}

// dartString is the Dart toString of an exception twin.
func dartString(e any) string {
	switch x := e.(type) {
	case interface{ String() string }:
		return x.String()
	case error:
		return x.Error()
	}
	return ""
}

func TestAssertionsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/assertions.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	goData, err := json.Marshal(buildAssertionScenes(t, ref))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(goData, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(ref) {
		t.Errorf("%d scenes in Go, %d in Flutter", len(got), len(ref))
	}
	keys := make([]string, 0, len(ref))
	for k := range ref {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	failures := 0
	for _, k := range keys {
		want := normalizeHashes(ref[k])
		g, ok := got[k]
		if !ok {
			t.Errorf("%s: missing in Go", k)
			continue
		}
		g = normalizeHashes(g)
		if reflect.DeepEqual(g, want) {
			continue
		}
		failures++
		if ws, ok := want.(string); ok {
			gs, _ := g.(string)
			t.Errorf("%s differs:\n--- Go\n%s\n--- Flutter\n%s\n--- first difference at line %d", k, gs, ws, firstDiffLine(gs, ws))
		} else {
			var diffs []string
			jsonDiff(k, g, want, &diffs)
			t.Errorf("%s differs (Go vs Flutter):\n%s", k, strings.Join(diffs[:min(len(diffs), 20)], "\n"))
		}
		if failures > 15 {
			t.Fatal("too many differences")
		}
	}
	t.Logf("%d scenes compared", len(keys))
}
