package foundation

// Port of packages/flutter/lib/src/foundation/assertions.dart (Flutter
// 3.50.1): FlutterError and its static error-reporting machinery,
// FlutterErrorDetails, the Error* diagnostics, DiagnosticsStackTrace, stack
// filters and debugPrintStack.
//
// Exceptions and stack traces (see API_MAPPING "Errors and stack traces"):
//   - Dart's `Object exception` is `any`: a Go error, a panic value, a
//     string... Dart's type tests map as: `is AssertionError` is
//     dart.AssertionFailure (dart.AssertionError and FlutterError), `is
//     String` is string, `is num` is a builtin Go number, `is Error ||
//     is Exception` is error; anything else is "an object".
//   - `StackTrace?` is dart.StackTrace (nil for null). Flutter reads traces
//     as text in the Dart VM's format, and dart.StackTraceCurrent /
//     dart.StackTraceFromPanic render Go stacks in that format.
//   - Static fields are package-level variables named FlutterError<Name>
//     (FlutterErrorOnError, FlutterErrorPresentError...), per API_MAPPING.
//
// Dart's assert(() {...}()) blocks run when asserts are enabled
// (debugAssertsEnabled, as in diagnostics.go); FlutterError.fromParts'
// argument checks always panic, as all argument assertions do in flutter-go.

import (
	"iter"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// FlutterExceptionHandler is the signature of FlutterErrorOnError handlers.
type FlutterExceptionHandler func(details *FlutterErrorDetails)

// DiagnosticPropertiesTransformer transforms the properties of a
// FlutterErrorDetails' DiagnosticsNode (see
// FlutterErrorDetailsPropertiesTransformers).
type DiagnosticPropertiesTransformer func(properties iter.Seq[DiagnosticsNode]) iter.Seq[DiagnosticsNode]

// InformationCollector collects information describing an error
// (FlutterErrorDetails.InformationCollector).
type InformationCollector func() iter.Seq[DiagnosticsNode]

// StackTraceDemangler turns a StackTrace into one StackFrame can parse (see
// FlutterErrorDemangleStackTrace).
type StackTraceDemangler func(details dart.StackTrace) dart.StackTrace

// PartialStackFrame is partial information from a stack frame, for stack
// filtering (RepetitiveStackFrameFilter).
type PartialStackFrame struct {
	// Package matches the frame's "scheme:package/path" anywhere: a
	// dart.StringPattern or a *regexp.Regexp (Dart's Pattern).
	Package dart.Pattern
	// ClassName must equal the frame's class name.
	ClassName string
	// Method must equal the frame's method name.
	Method string
}

// PartialStackFrameAsynchronousSuspension is
// PartialStackFrame.asynchronousSuspension: matches the
// "<asynchronous suspension>" frame.
var PartialStackFrameAsynchronousSuspension = PartialStackFrame{
	Package:   dart.StringPattern(""),
	ClassName: "",
	Method:    "asynchronous suspension",
}

// Matches tests whether stackFrame matches this partial frame.
func (p PartialStackFrame) Matches(stackFrame StackFrame) bool {
	stackFramePackage := stackFrame.PackageScheme + ":" + stackFrame.Package + "/" + stackFrame.PackagePath
	// Ideally this wouldn't be necessary.
	// TODO(dnfield): https://github.com/dart-lang/sdk/issues/40117
	if KIsWeb {
		method := p.Method
		if strings.HasPrefix(method, "_") {
			method = "[" + method + "]"
		}
		return p.Package.MatchString(stackFramePackage) && stackFrame.Method == method
	}
	return p.Package.MatchString(stackFramePackage) &&
		stackFrame.Method == p.Method &&
		stackFrame.ClassName == p.ClassName
}

// StackFilter is a filter that FlutterErrorDefaultStackFilter applies to a
// trace's frames (register one with FlutterErrorAddDefaultStackFilter).
// Filter sets reasons[i] (nil: keep) for each frame it elides; consecutive
// frames with equal reasons are collapsed into one line.
type StackFilter interface {
	Filter(stackFrames []StackFrame, reasons []*string)
}

// RepetitiveStackFrameFilter replaces each run of frames matching Frames
// with Replacement.
type RepetitiveStackFrameFilter struct {
	// Frames is the sequence of frames to match.
	Frames []PartialStackFrame
	// Replacement is the reason reported for the elided frames.
	Replacement string
}

// NumFrames is the number of frames this filter matches.
func (f RepetitiveStackFrameFilter) NumFrames() int { return len(f.Frames) }

// Filter marks each run of Frames. Like Flutter's, it never matches a run
// that ends at the last frame (the loop stops at len-numFrames).
func (f RepetitiveStackFrameFilter) Filter(stackFrames []StackFrame, reasons []*string) {
	numFrames := f.NumFrames()
	for index := 0; index < len(stackFrames)-numFrames; index++ {
		if f.matchesFrames(stackFrames[index:min(index+numFrames, len(stackFrames))]) {
			for i := index; i < index+numFrames; i++ {
				replacement := f.Replacement
				reasons[i] = &replacement
			}
			index += numFrames - 1
		}
	}
}

func (f RepetitiveStackFrameFilter) matchesFrames(stackFrames []StackFrame) bool {
	if len(stackFrames) < f.NumFrames() {
		return false
	}
	for index := range stackFrames {
		if !f.Frames[index].Matches(stackFrames[index]) {
			return false
		}
	}
	return true
}

// errorDiagnostic is _ErrorDiagnostic: a DiagnosticsProperty<List<Object>>
// with no name, whose description is its message parts joined.
type errorDiagnostic struct {
	DiagnosticsProperty[[]any]
}

func (d *errorDiagnostic) initErrorDiagnostic(self DiagnosticsNode, message string, level DiagnosticLevel) {
	style := DiagnosticsTreeStyleFlat
	d.initProperty(self, nil, []any{message}, propertyArgs{
		showName:      pFalse,
		showSeparator: pFalse,
		defaultValue:  nil,
		style:         &style,
		level:         &level,
		valueComputed: true,
	})
}

// ToString is the message (valueToString), whatever the parent's style.
func (d *errorDiagnostic) ToString(opts ...DiagnosticsNodeToStringOptions) string {
	var parent *TextTreeConfiguration
	if len(opts) > 0 {
		parent = opts[0].ParentConfiguration
	}
	return d.outer().(valueToStringer).ValueToString(ValueToStringOptions{ParentConfiguration: parent})
}

// ValueToString joins the message parts (Dart's value.join()).
func (d *errorDiagnostic) ValueToString(...ValueToStringOptions) string {
	var b strings.Builder
	for _, part := range d.TypedValue() {
		b.WriteString(dartToString(part))
	}
	return b.String()
}

// ErrorDescription is an explanation of the problem and its cause (level
// info).
type ErrorDescription struct{ errorDiagnostic }

// NewErrorDescription is ErrorDescription(message).
func NewErrorDescription(message string) *ErrorDescription {
	d := &ErrorDescription{}
	d.initErrorDiagnostic(d, message, DiagnosticLevelInfo)
	return d
}

// ErrorSummary is a short (one line) description of the problem (level
// summary); every FlutterError starts with one.
type ErrorSummary struct{ errorDiagnostic }

// NewErrorSummary is ErrorSummary(message).
func NewErrorSummary(message string) *ErrorSummary {
	d := &ErrorSummary{}
	d.initErrorDiagnostic(d, message, DiagnosticLevelSummary)
	return d
}

// ErrorHint is a hint about how to fix the problem (level hint).
type ErrorHint struct{ errorDiagnostic }

// NewErrorHint is ErrorHint(message).
func NewErrorHint(message string) *ErrorHint {
	d := &ErrorHint{}
	d.initErrorDiagnostic(d, message, DiagnosticLevelHint)
	return d
}

// ErrorSpacer is an empty line between the parts of an error message.
type ErrorSpacer struct {
	DiagnosticsProperty[*struct{}]
}

// NewErrorSpacer is ErrorSpacer().
func NewErrorSpacer() *ErrorSpacer {
	s := &ErrorSpacer{}
	empty := ""
	s.initProperty(s, &empty, nil, propertyArgs{ // name '' (not null), as in Dart
		description:   &empty,
		showName:      pFalse,
		defaultValue:  KNoDefaultValue,
		valueComputed: true,
	})
	return s
}

// FlutterErrorDetails describes an error for FlutterErrorReportError and
// FlutterErrorOnError. Build it with NewFlutterErrorDetails; a type
// extending it embeds it and calls SetSelf.
type FlutterErrorDetails struct {
	DiagnosticableMixin

	// Exception is what was thrown (an error, a panic value...).
	Exception any
	// Stack is where it was thrown, or nil.
	Stack dart.StackTrace
	// Library is the library the error was caught in, or nil.
	Library *string
	// Context describes what was happening when the error was caught (it
	// follows "thrown" in the report), or nil.
	Context DiagnosticsNode
	// StackFilter filters the stack's lines (nil:
	// FlutterErrorDefaultStackFilter).
	StackFilter IterableFilter[string]
	// InformationCollector adds information at the end of the report, or
	// nil.
	InformationCollector InformationCollector
	// Silent says the error is expected and needn't be printed outside
	// debug builds.
	Silent bool
}

// FlutterErrorDetailsOptions are FlutterErrorDetails' named parameters
// (Library nil: "Flutter framework"; NullString for Dart's null).
type FlutterErrorDetailsOptions struct {
	Stack                dart.StackTrace
	Library              *string
	Context              DiagnosticsNode
	StackFilter          IterableFilter[string]
	InformationCollector InformationCollector
	Silent               bool
}

// NewFlutterErrorDetails is FlutterErrorDetails(exception: exception, ...).
func NewFlutterErrorDetails(exception any, opts ...FlutterErrorDetailsOptions) *FlutterErrorDetails {
	var o FlutterErrorDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &FlutterErrorDetails{
		Exception:            exception,
		Stack:                o.Stack,
		Library:              resolveString(o.Library, strp("Flutter framework")),
		Context:              o.Context,
		StackFilter:          o.StackFilter,
		InformationCollector: o.InformationCollector,
		Silent:               o.Silent,
	}
	d.SetSelf(d)
	return d
}

// FlutterErrorDetailsCopyWithOptions are copyWith's parameters (nil: keep
// the current value).
type FlutterErrorDetailsCopyWithOptions struct {
	Context              DiagnosticsNode
	Exception            any
	InformationCollector InformationCollector
	Library              *string
	Silent               *bool
	Stack                dart.StackTrace
	StackFilter          IterableFilter[string]
}

// CopyWith is a copy with the given fields replaced.
func (d *FlutterErrorDetails) CopyWith(opts ...FlutterErrorDetailsCopyWithOptions) *FlutterErrorDetails {
	var o FlutterErrorDetailsCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	c := &FlutterErrorDetails{
		Context:              d.Context,
		Exception:            d.Exception,
		InformationCollector: d.InformationCollector,
		Library:              d.Library,
		Silent:               d.Silent,
		Stack:                d.Stack,
		StackFilter:          d.StackFilter,
	}
	if o.Context != nil {
		c.Context = o.Context
	}
	if o.Exception != nil {
		c.Exception = o.Exception
	}
	if o.InformationCollector != nil {
		c.InformationCollector = o.InformationCollector
	}
	if o.Library != nil {
		c.Library = o.Library
	}
	if o.Silent != nil {
		c.Silent = *o.Silent
	}
	if o.Stack != nil {
		c.Stack = o.Stack
	}
	if o.StackFilter != nil {
		c.StackFilter = o.StackFilter
	}
	c.SetSelf(c)
	return c
}

// FlutterErrorDetailsPropertiesTransformers is
// FlutterErrorDetails.propertiesTransformers: transformers applied, in
// order, to the properties of every FlutterErrorDetails' DiagnosticsNode.
var FlutterErrorDetailsPropertiesTransformers []DiagnosticPropertiesTransformer

// isDartNum reports whether v is a Dart num: a builtin Go number.
func isDartNum(v any) bool {
	if v == nil {
		return false
	}
	t := reflect.TypeOf(v)
	if t.PkgPath() != "" {
		return false
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// ExceptionAsString is the exception as a string: an assertion's message
// moved before the location of the failed assert, a String as is, an error
// as its toString, any other object indented by two spaces.
func (d *FlutterErrorDetails) ExceptionAsString() string {
	var longMessage string
	hasLongMessage := false
	if assertion, ok := d.Exception.(dart.AssertionFailure); ok {
		// Regular _AssertionErrors thrown by assert() put the message last, after
		// some code snippets. This leads to ugly messages. To avoid this, we move
		// the assertion message up to before the code snippets, separated by a
		// newline, if we recognize that format is being used.
		message := assertion.AssertionMessage()
		fullMessage := dartToString(d.Exception)
		if message, ok := message.(string); ok && message != fullMessage {
			if len(fullMessage) > len(message) {
				position := strings.LastIndex(fullMessage, message)
				if position == len(fullMessage)-len(message) &&
					position > 2 &&
					fullMessage[position-2:position] == ": " {
					// Add a linebreak so that the filename at the start of the
					// assertion message is always on its own line.
					body := fullMessage[:position-2]
					if splitPoint := strings.Index(body, " Failed assertion:"); splitPoint >= 0 {
						body = body[:splitPoint] + "\n" + body[splitPoint+1:]
					}
					longMessage, hasLongMessage = dartTrimRight(message)+"\n"+body, true
				}
			}
		}
		if !hasLongMessage {
			longMessage = fullMessage
		}
	} else if s, ok := d.Exception.(string); ok {
		longMessage = s
	} else if _, ok := d.Exception.(error); ok {
		longMessage = dartToString(d.Exception)
	} else {
		longMessage = "  " + dartToString(d.Exception)
	}
	longMessage = dartTrimRight(longMessage)
	if longMessage == "" {
		longMessage = "  <no message available>"
	}
	return longMessage
}

func (d *FlutterErrorDetails) exceptionToDiagnosticable() Diagnosticable {
	exception := d.Exception
	if e, ok := exception.(*FlutterError); ok {
		return e
	}
	if a, ok := exception.(dart.AssertionFailure); ok {
		if e, ok := a.AssertionMessage().(*FlutterError); ok {
			return e
		}
	}
	return nil
}

// Summary is a short (one line) description of the problem: the
// exception's ErrorSummary when it is a FlutterError, else the first line of
// ExceptionAsString.
func (d *FlutterErrorDetails) Summary() DiagnosticsNode {
	formatException := func() string {
		first, _, _ := strings.Cut(d.ExceptionAsString(), "\n")
		return strings.TrimLeftFunc(first, isDartWhitespace)
	}
	if KReleaseMode {
		return NewDiagnosticsNodeMessage(formatException())
	}
	diagnosticable := d.exceptionToDiagnosticable()
	var summary DiagnosticsNode
	if diagnosticable != nil {
		builder := NewDiagnosticPropertiesBuilder()
		d.outer().DebugFillProperties(builder)
		for _, node := range builder.Properties {
			if node.Level() == DiagnosticLevelSummary {
				summary = node
				break
			}
		}
	}
	if summary != nil {
		return summary
	}
	return NewErrorSummary(formatException())
}

// DebugFillProperties adds the report's parts: what was thrown, the
// exception's diagnostics or message, a hint when the assertion looks like
// the framework's fault, the stack, and the collected information.
func (d *FlutterErrorDetails) DebugFillProperties(properties *DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	contextText := ""
	if d.Context != nil {
		contextText = NewErrorDescription(" " + d.Context.String()).String()
	}
	verb := NewErrorDescription("thrown" + contextText)
	diagnosticable := d.exceptionToDiagnosticable()
	if isDartNum(d.Exception) {
		properties.Add(NewErrorDescription("The number " + dartToString(d.Exception) + " was " + verb.String() + "."))
	} else {
		var name string
		_, isAssertion := d.Exception.(dart.AssertionFailure)
		_, isString := d.Exception.(string)
		_, isError := d.Exception.(error)
		switch {
		case isAssertion:
			name = "assertion"
		case isString:
			name = "message"
		case isError:
			name = dartRuntimeType(d.Exception)
		default:
			name = dartRuntimeType(d.Exception) + " object"
		}
		errorName := NewErrorDescription(name)
		properties.Add(NewErrorDescription("The following " + errorName.String() + " was " + verb.String() + ":"))
		if diagnosticable != nil {
			diagnosticable.DebugFillProperties(properties)
		} else {
			// Many exception classes put their type at the head of their message.
			// This is redundant with the way we display exceptions, so attempt to
			// strip out that header when we see it.
			prefix := dartRuntimeType(d.Exception) + ": "
			message := d.ExceptionAsString()
			message = strings.TrimPrefix(message, prefix)
			properties.Add(NewErrorSummary(message))
		}
	}

	if d.Stack != nil {
		if _, isAssertion := d.Exception.(dart.AssertionFailure); isAssertion && diagnosticable == nil {
			// After popping off any dart: stack frames, are there at least two more
			// stack frames coming from package flutter?
			//
			// If not: Error is in user code (user violated assertion in framework).
			// If so:  Error is in Framework. We either need an assertion higher up
			//         in the stack, or we've violated our own assertions.
			stackFrames := StackFrameFromStackTrace(FlutterErrorDemangleStackTrace(d.Stack))
			skip := 0
			for skip < len(stackFrames) && stackFrames[skip].PackageScheme == "dart" {
				skip++
			}
			stackFrames = stackFrames[skip:]
			ourFault := len(stackFrames) >= 2 &&
				stackFrames[0].Package == "flutter" &&
				stackFrames[1].Package == "flutter"
			if ourFault {
				properties.Add(NewErrorSpacer())
				properties.Add(NewErrorHint(
					"Either the assertion indicates an error in the framework itself, or we should " +
						"provide substantially more information in this error message to help you determine " +
						"and fix the underlying cause.\n" +
						"In either case, please report this assertion by filing a bug on GitHub:\n" +
						"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml",
				))
			}
		}
		properties.Add(NewErrorSpacer())
		properties.Add(NewDiagnosticsStackTrace(
			"When the exception was thrown, this was the stack",
			d.Stack,
			DiagnosticsStackTraceOptions{StackFilter: d.StackFilter},
		))
	}
	if d.InformationCollector != nil {
		properties.Add(NewErrorSpacer())
		for node := range d.InformationCollector() {
			properties.Add(node)
		}
	}
}

// ToStringShort is "Exception caught by <library>".
func (d *FlutterErrorDetails) ToStringShort() string {
	if d.Library != nil {
		return "Exception caught by " + *d.Library
	}
	return "Exception caught"
}

// ToString renders the report in the error style (MinLevel nil: info).
func (d *FlutterErrorDetails) ToString(opts ...DiagnosticableToStringOptions) string {
	var o DiagnosticableToStringOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	minLevel := levelOr(o.MinLevel, DiagnosticLevelInfo)
	style := DiagnosticsTreeStyleError
	return d.outer().ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: &style}).
		ToStringDeep(DiagnosticsNodeToStringDeepOptions{MinLevel: &minLevel})
}

func (d *FlutterErrorDetails) String() string { return d.outer().ToString() }

// ToDiagnosticsNode is a node whose properties pass through
// FlutterErrorDetailsPropertiesTransformers.
func (d *FlutterErrorDetails) ToDiagnosticsNode(opts ...ToDiagnosticsNodeOptions) DiagnosticsNode {
	n := &_FlutterErrorDetailsNode{}
	n.initDiagnosticableNode(n, d.outer(), []DiagnosticableNodeOptions{toDiagnosticsNodeOpts(opts)})
	return n
}

// FlutterError is an error in the framework's use: a list of
// DiagnosticsNode that starts with an ErrorSummary. It is a Go error and,
// as in Dart, an AssertionError (dart.AssertionFailure).
type FlutterError struct {
	DiagnosticableTreeMixin
	// Diagnostics are the parts of the message.
	Diagnostics []DiagnosticsNode
}

// NewFlutterError is FlutterError(message): the first line becomes the
// ErrorSummary, each further line an ErrorDescription.
func NewFlutterError(message string) *FlutterError {
	lines := strings.Split(message, "\n")
	parts := []DiagnosticsNode{NewErrorSummary(lines[0])}
	for _, line := range lines[1:] {
		parts = append(parts, NewErrorDescription(line))
	}
	return NewFlutterErrorFromParts(parts)
}

const malformedErrorFooter = "\nThis error should still help you solve your problem, " +
	"however please also report this malformed error in the " +
	"framework by filing a bug on GitHub:\n" +
	"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml"

// NewFlutterErrorFromParts is FlutterError.fromParts. It panics with a
// dart.AssertionError holding a FlutterError when diagnostics is empty or
// doesn't start with an ErrorSummary, and (when asserts are enabled) with a
// FlutterError when it has more than one summary.
func NewFlutterErrorFromParts(diagnostics []DiagnosticsNode) *FlutterError {
	if len(diagnostics) == 0 {
		panic(dart.NewAssertionError(NewFlutterErrorFromParts([]DiagnosticsNode{NewErrorSummary("Empty FlutterError")})))
	}
	e := &FlutterError{Diagnostics: diagnostics}
	e.SetSelf(e)
	malformed := func() DiagnosticsNode {
		style := DiagnosticsTreeStyleWhitespace
		return NewDiagnosticsProperty[*FlutterError]("Malformed", e, DiagnosticsPropertyOptions{
			ExpandableValue: true,
			ShowSeparator:   pFalse,
			Style:           &style,
		})
	}
	if diagnostics[0].Level() != DiagnosticLevelSummary {
		panic(dart.NewAssertionError(NewFlutterErrorFromParts([]DiagnosticsNode{
			NewErrorSummary("FlutterError is missing a summary."),
			NewErrorDescription(
				"All FlutterError objects should start with a short (one line) " +
					"summary description of the problem that was detected.",
			),
			malformed(),
			NewErrorDescription(malformedErrorFooter),
		})))
	}
	if debugAssertsEnabled {
		var summaries []DiagnosticsNode
		for _, node := range diagnostics {
			if node.Level() == DiagnosticLevelSummary {
				summaries = append(summaries, node)
			}
		}
		if len(summaries) > 1 {
			message := []DiagnosticsNode{
				NewErrorSummary("FlutterError contained multiple error summaries."),
				NewErrorDescription(
					"All FlutterError objects should have only a single short " +
						"(one line) summary description of the problem that was " +
						"detected.",
				),
				malformed(),
				NewErrorDescription("\nThe malformed error has " + dartToString(len(summaries)) + " summaries."),
			}
			for i, summary := range summaries {
				message = append(message, NewDiagnosticsProperty("Summary "+dartToString(i+1), summary,
					DiagnosticsPropertyOptions{ExpandableValue: true}))
			}
			message = append(message, NewErrorDescription(malformedErrorFooter))
			panic(NewFlutterErrorFromParts(message))
		}
	}
	return e
}

// Message is the message getter: the same as ToString.
func (e *FlutterError) Message() string { return e.ToString() }

// AssertionMessage is Dart's AssertionError.message (FlutterError
// implements AssertionError): the same as ToString.
func (e *FlutterError) AssertionMessage() any { return e.Message() }

// Error makes FlutterError a Go error: the same as ToString.
func (e *FlutterError) Error() string { return e.ToString() }

// FlutterErrorOnError is FlutterError.onError: called by
// FlutterErrorReportError for every error the framework catches (nil:
// ignore them). It starts as FlutterErrorPresentError's initial value;
// tests and apps replace it.
var FlutterErrorOnError FlutterExceptionHandler = dumpErrorToConsoleHandler

// FlutterErrorDemangleStackTrace is FlutterError.demangleStackTrace: turns a
// StackTrace into one StackFrame can parse (default: the identity).
var FlutterErrorDemangleStackTrace StackTraceDemangler = defaultStackTraceDemangler

func defaultStackTraceDemangler(stackTrace dart.StackTrace) dart.StackTrace { return stackTrace }

// FlutterErrorPresentError is FlutterError.presentError: how errors are
// shown to the user (default: FlutterErrorDumpErrorToConsole).
var FlutterErrorPresentError FlutterExceptionHandler = dumpErrorToConsoleHandler

func dumpErrorToConsoleHandler(details *FlutterErrorDetails) { FlutterErrorDumpErrorToConsole(details) }

var errorCount = 0

// FlutterErrorResetErrorCount is FlutterError.resetErrorCount: the next
// error is dumped in full again.
func FlutterErrorResetErrorCount() { errorCount = 0 }

// FlutterErrorWrapWidth is FlutterError.wrapWidth: the width error
// descriptions are wrapped to.
const FlutterErrorWrapWidth = 100

// DumpErrorToConsoleOptions are dumpErrorToConsole's named parameters.
type DumpErrorToConsoleOptions struct {
	// ForceReport dumps the full report even if it isn't the first error
	// and even if the error is silent.
	ForceReport bool
}

// FlutterErrorDumpErrorToConsole is FlutterError.dumpErrorToConsole: prints
// the first error (since FlutterErrorResetErrorCount) in full, through
// ErrorToConsoleDumper, and only its summary afterwards.
func FlutterErrorDumpErrorToConsole(details *FlutterErrorDetails, opts ...DumpErrorToConsoleOptions) {
	forceReport := len(opts) > 0 && opts[0].ForceReport
	isInDebugMode := false
	if debugAssertsEnabled {
		// In debug mode, we ignore the "silent" flag.
		isInDebugMode = true
	}
	reportError := isInDebugMode || !details.Silent
	if !reportError && !forceReport {
		return
	}
	if errorCount == 0 || forceReport {
		// Diagnostics is only available in debug mode. In profile and release modes fallback to plain print.
		if isInDebugMode {
			wrapWidth, maxDescendents := FlutterErrorWrapWidth, 5
			style := DiagnosticsTreeStyleError
			ErrorToConsoleDumper.Dump(dartTrimRight(NewTextTreeRenderer(TextTreeRendererOptions{
				WrapWidthProperties:           &wrapWidth,
				MaxDescendentsTruncatableNode: &maxDescendents,
			}).Render(details.outer().ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: &style}))))
		} else {
			label := dartToString(details.Exception)
			maxFrames := 100
			DebugPrintStack(DebugPrintStackOptions{StackTrace: details.Stack, Label: &label, MaxFrames: &maxFrames})
		}
	} else {
		ErrorToConsoleDumper.Dump("Another exception was thrown: " + details.Summary().String())
	}
	errorCount++
}

var stackFilters []StackFilter

// FlutterErrorAddDefaultStackFilter is FlutterError.addDefaultStackFilter:
// adds filter to those FlutterErrorDefaultStackFilter applies.
func FlutterErrorAddDefaultStackFilter(filter StackFilter) {
	stackFilters = append(stackFilters, filter)
}

// FlutterErrorDefaultStackFilter is FlutterError.defaultStackFilter: removes
// frames of dart:async, package:stack_trace and a few internal classes
// (counting them in a final "(elided ...)" line), then applies the
// registered StackFilters.
func FlutterErrorDefaultStackFilter(frames iter.Seq[string]) iter.Seq[string] {
	removedOrder := []string{
		"dart:async-patch",
		"dart:async",
		"package:stack_trace",
		"class _AssertionError",
		"class _FakeAsync",
		"class _FrameCallbackEntry",
		"class _Timer",
		"class _RawReceivePortImpl",
	}
	removedPackagesAndClasses := map[string]int{}
	for _, key := range removedOrder {
		removedPackagesAndClasses[key] = 0
	}
	skipped := 0
	parsedFrames := StackFrameFromStackString(strings.Join(slices.Collect(frames), "\n"))

	for index := 0; index < len(parsedFrames); index++ {
		frame := parsedFrames[index]
		className := "class " + frame.ClassName
		pkg := frame.PackageScheme + ":" + frame.Package
		if _, ok := removedPackagesAndClasses[className]; ok {
			skipped++
			removedPackagesAndClasses[className]++
			parsedFrames = slices.Delete(parsedFrames, index, index+1)
			index--
		} else if _, ok := removedPackagesAndClasses[pkg]; ok {
			skipped++
			removedPackagesAndClasses[pkg]++
			parsedFrames = slices.Delete(parsedFrames, index, index+1)
			index--
		}
	}
	reasons := make([]*string, len(parsedFrames))
	for _, filter := range stackFilters {
		filter.Filter(parsedFrames, reasons)
	}

	result := []string{}

	// Collapse duplicated reasons.
	for index := 0; index < len(parsedFrames); index++ {
		start := index
		for index < len(reasons)-1 &&
			reasons[index] != nil &&
			reasons[index+1] != nil && *reasons[index+1] == *reasons[index] {
			index++
		}
		suffix := ""
		if reasons[index] != nil {
			if index != start {
				suffix = " (" + dartToString(index-start+2) + " frames)"
			} else {
				suffix = " (1 frame)"
			}
		}
		line := parsedFrames[index].Source
		if reasons[index] != nil {
			line = *reasons[index]
		}
		result = append(result, line+suffix)
	}

	// Only include packages we actually elided from.
	var where []string
	for _, key := range removedOrder {
		if removedPackagesAndClasses[key] > 0 {
			where = append(where, key)
		}
	}
	sort.Strings(where)
	if skipped == 1 {
		result = append(result, "(elided one frame from "+where[0]+")")
	} else if skipped > 1 {
		if len(where) > 1 {
			where[len(where)-1] = "and " + where[len(where)-1]
		}
		if len(where) > 2 {
			result = append(result, "(elided "+dartToString(skipped)+" frames from "+strings.Join(where, ", ")+")")
		} else {
			result = append(result, "(elided "+dartToString(skipped)+" frames from "+strings.Join(where, " ")+")")
		}
	}
	return slices.Values(result)
}

// DebugFillProperties adds the error's diagnostics.
func (e *FlutterError) DebugFillProperties(properties *DiagnosticPropertiesBuilder) {
	for _, node := range e.Diagnostics {
		properties.Add(node)
	}
}

// ToStringShort is "FlutterError".
func (e *FlutterError) ToStringShort() string { return "FlutterError" }

// ToString renders each part on its own lines, without wrapping (the
// first message part in release builds).
func (e *FlutterError) ToString(...DiagnosticableToStringOptions) string {
	if KReleaseMode {
		for _, node := range e.Diagnostics {
			if d, ok := node.(interface{ errorDiagnosticMessage() string }); ok {
				return d.errorDiagnosticMessage()
			}
		}
		return e.ToStringShort()
	}
	// Avoid wrapping lines.
	wrapWidth := 4000000000
	renderer := NewTextTreeRenderer(TextTreeRendererOptions{WrapWidth: &wrapWidth})
	parts := make([]string, len(e.Diagnostics))
	for i, node := range e.Diagnostics {
		parts[i] = dartTrimRight(renderer.Render(node))
	}
	return strings.Join(parts, "\n")
}

func (e *FlutterError) String() string { return e.ToString() }

func (d *errorDiagnostic) errorDiagnosticMessage() string {
	return d.outer().(valueToStringer).ValueToString()
}

// FlutterErrorReportError is FlutterError.reportError: passes details to
// FlutterErrorOnError (if set).
func FlutterErrorReportError(details *FlutterErrorDetails) {
	if FlutterErrorOnError != nil {
		FlutterErrorOnError(details)
	}
}

// DebugPrintStackOptions are debugPrintStack's named parameters.
type DebugPrintStackOptions struct {
	// StackTrace is the trace to print (nil: the caller's,
	// dart.StackTraceCurrent).
	StackTrace dart.StackTrace
	// Label is printed first when not nil.
	Label *string
	// MaxFrames limits the frames printed when not nil.
	MaxFrames *int
}

// DebugPrintStack is debugPrintStack: dumps a stack trace to the console
// (filtered by FlutterErrorDefaultStackFilter), after an optional label.
func DebugPrintStack(opts ...DebugPrintStackOptions) {
	var o DebugPrintStackOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Label != nil {
		ErrorToConsoleDumper.Dump(*o.Label)
	}
	stackTrace := o.StackTrace
	if stackTrace == nil {
		stackTrace = dart.StackTraceCurrent()
	} else {
		stackTrace = FlutterErrorDemangleStackTrace(stackTrace)
	}
	lines := strings.Split(dartTrimRight(stackTrace.String()), "\n")
	if KIsWeb && len(lines) != 0 {
		// Remove extra call to StackTrace.current for web platform.
		for len(lines) > 0 && (strings.Contains(lines[0], "StackTrace.current") ||
			strings.Contains(lines[0], "dart-sdk/lib/_internal") ||
			strings.Contains(lines[0], "dart:sdk_internal")) {
			lines = lines[1:]
		}
	}
	if o.MaxFrames != nil && *o.MaxFrames < len(lines) {
		lines = lines[:max(*o.MaxFrames, 0)]
	}
	ErrorToConsoleDumper.Dump(strings.Join(slices.Collect(FlutterErrorDefaultStackFilter(slices.Values(lines))), "\n"))
}

// DiagnosticsStackTrace is a DiagnosticsBlock listing the frames of a stack
// trace, one per line, unwrapped.
type DiagnosticsStackTrace struct {
	DiagnosticsBlock
}

// DiagnosticsStackTraceOptions are DiagnosticsStackTrace's named parameters
// (StackFilter nil: FlutterErrorDefaultStackFilter; ShowSeparator nil:
// true).
type DiagnosticsStackTraceOptions struct {
	StackFilter   IterableFilter[string]
	ShowSeparator *bool
}

// NewDiagnosticsStackTrace is DiagnosticsStackTrace(name, stack,
// {stackFilter, showSeparator}).
func NewDiagnosticsStackTrace(name string, stack dart.StackTrace, opts ...DiagnosticsStackTraceOptions) *DiagnosticsStackTrace {
	var o DiagnosticsStackTraceOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	style := DiagnosticsTreeStyleFlat
	var value any
	if stack != nil {
		value = stack
	}
	s := &DiagnosticsStackTrace{DiagnosticsBlock: *NewDiagnosticsBlock(DiagnosticsBlockOptions{
		Name:          &name,
		Value:         value,
		Properties:    applyStackFilter(stack, o.StackFilter),
		Style:         &style,
		AllowTruncate: true,
		ShowSeparator: o.ShowSeparator,
	})}
	s.SetSelf(s)
	return s
}

// DiagnosticsStackTraceSingleFrameOptions are
// DiagnosticsStackTrace.singleFrame's named parameters (ShowSeparator nil:
// true).
type DiagnosticsStackTraceSingleFrameOptions struct {
	ShowSeparator *bool
}

// NewDiagnosticsStackTraceSingleFrame is DiagnosticsStackTrace.singleFrame
// (name, frame: frame, {showSeparator}): a single frame line.
func NewDiagnosticsStackTraceSingleFrame(name string, frame string, opts ...DiagnosticsStackTraceSingleFrameOptions) *DiagnosticsStackTrace {
	var o DiagnosticsStackTraceSingleFrameOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	style := DiagnosticsTreeStyleWhitespace
	s := &DiagnosticsStackTrace{DiagnosticsBlock: *NewDiagnosticsBlock(DiagnosticsBlockOptions{
		Name:          &name,
		Properties:    []DiagnosticsNode{createStackFrame(frame)},
		Style:         &style,
		ShowSeparator: o.ShowSeparator,
	})}
	s.SetSelf(s)
	return s
}

func applyStackFilter(stack dart.StackTrace, stackFilter IterableFilter[string]) []DiagnosticsNode {
	if stack == nil {
		return []DiagnosticsNode{}
	}
	filter := stackFilter
	if filter == nil {
		filter = FlutterErrorDefaultStackFilter
	}
	frames := filter(slices.Values(strings.Split(dartTrimRight(FlutterErrorDemangleStackTrace(stack).String()), "\n")))
	result := []DiagnosticsNode{}
	for frame := range frames {
		result = append(result, createStackFrame(frame))
	}
	return result
}

func createStackFrame(frame string) DiagnosticsNode {
	return NewDiagnosticsNodeMessage(frame, DiagnosticsNodeMessageOptions{AllowWrap: pFalse})
}

// AllowTruncate is false: stack traces are never truncated.
func (s *DiagnosticsStackTrace) AllowTruncate() bool { return false }

// _FlutterErrorDetailsNode is Dart's _FlutterErrorDetailsNode (named alike
// so its runtimeType prints the same): a DiagnosticableNode whose builder
// applies FlutterErrorDetailsPropertiesTransformers.
type _FlutterErrorDetailsNode struct {
	DiagnosticableNode[Diagnosticable]
}

// Builder is the value's properties passed through every transformer (a
// new builder on each call, as in Flutter).
func (n *_FlutterErrorDetailsNode) Builder() *DiagnosticPropertiesBuilder {
	builder := n.DiagnosticableNode.Builder()
	if builder == nil {
		return nil
	}
	properties := slices.Values(builder.Properties)
	for _, transformer := range FlutterErrorDetailsPropertiesTransformers {
		properties = transformer(properties)
	}
	return NewDiagnosticPropertiesBuilderFromProperties(slices.AppendSeq([]DiagnosticsNode{}, properties))
}
