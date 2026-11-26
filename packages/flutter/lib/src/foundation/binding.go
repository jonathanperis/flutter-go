package foundation

// Port of binding.dart (Flutter 3.50.1).
//
// Not ported: _initListenable (it installs Listenable.onError from
// package:listen, not ported yet).
//
// Go has no zones: the "zone" a binding is initialized in is the current
// async.Loop (flutter_test gives each test its own FakeAsync zone, as
// flutter-go gives it its own loop), so DebugCheckZone compares loops.
//
// Dart's assert(() {...}()) blocks with side effects (recording the
// binding type, registering the debug service extensions) run in debug
// builds only; plain assertions panic in every build (API_MAPPING).

import (
	"errors"
	"math"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/developer"
)

// ServiceExtensionCallback is the signature of a service extension: it
// receives the parameters and returns a JSON-encodable map.
type ServiceExtensionCallback func(parameters map[string]string) async.Future[map[string]any]

// BindingBase is the base of every binding: the singletons that glue the
// framework to the engine (SchedulerBinding, WidgetsBinding...).
//
// A binding is a struct embedding BindingBaseBase (Dart: extends
// BindingBase) and the mixins of the bindings it combines; its constructor
// calls Construct(self) on the embedded base, which runs Dart's BindingBase()
// constructor: InitInstances and InitServiceExtensions, called through self
// so overrides run. Overrides call the embedded base's method as "super".
type BindingBase interface {
	// Window is the deprecated main window (Deprecated: use View.of or
	// PlatformDispatcher; deprecated after v3.7.0-32.0.pre).
	Window() *ui.SingletonFlutterWindow
	// PlatformDispatcher is ui.PlatformDispatcherInstance().
	PlatformDispatcher() *ui.PlatformDispatcher
	// InitInstances is called by the constructor: bindings set their
	// singletons here and must call the embedded base's InitInstances.
	InitInstances()
	// InitServiceExtensions is called by the constructor after
	// InitInstances: bindings register their service extensions here and
	// must call the embedded base's.
	InitServiceExtensions()
	// DebugCheckZone checks that the binding is used in the zone (loop) it
	// was initialized in; it returns true, to be used in an assert.
	DebugCheckZone(entryPoint string) bool
	// Locked reports whether events are locked (LockEvents is running).
	Locked() bool
	// LockEvents locks events (pointer events, frames...) until the future
	// callback returns completes, then calls Unlocked.
	LockEvents(callback func() async.Future[async.Void]) async.Future[async.Void]
	// Unlocked is called when events are unlocked; bindings flush the
	// events they queued meanwhile.
	Unlocked()
	// ReassembleApplication causes the entire application to redraw, as
	// after a hot reload (LockEvents(PerformReassemble)).
	ReassembleApplication() async.Future[async.Void]
	// PerformReassemble is where bindings reassemble; it must call the
	// embedded base's.
	PerformReassemble() async.Future[async.Void]
	// RegisterSignalServiceExtension registers a service extension that
	// takes no arguments and returns nothing.
	RegisterSignalServiceExtension(name string, callback AsyncCallback)
	// RegisterBoolServiceExtension registers a service extension that
	// reads (and, with an "enabled" parameter, sets) a boolean.
	RegisterBoolServiceExtension(name string, getter AsyncValueGetter[bool], setter AsyncValueSetter[bool])
	// RegisterNumericServiceExtension registers a service extension that
	// reads (and, with a parameter named name, sets) a double.
	RegisterNumericServiceExtension(name string, getter AsyncValueGetter[float64], setter AsyncValueSetter[float64])
	// RegisterStringServiceExtension registers a service extension that
	// reads (and, with a "value" parameter, sets) a string.
	RegisterStringServiceExtension(name string, getter AsyncValueGetter[string], setter AsyncValueSetter[string])
	// RegisterServiceExtension registers "ext.flutter.<name>" with
	// dart:developer (package developer).
	RegisterServiceExtension(name string, callback ServiceExtensionCallback)
	// PostEvent posts an event to the VM service's Extension stream (tests
	// override it to observe events).
	PostEvent(eventKind string, eventData map[string]any)
	String() string

	bindingBase() *BindingBaseBase
}

// BindingBaseBase is BindingBase's state and default methods, to embed.
type BindingBaseBase struct {
	self             BindingBase
	debugConstructed bool
	debugBindingZone *async.Loop
	lockCount        int
}

var (
	bindingDebugInitializedType             reflect.Type
	bindingDebugServiceExtensionsRegistered = false
)

// BindingBaseDebugZoneErrorsAreFatal makes DebugCheckZone panic instead of
// reporting an error (BindingBase.debugZoneErrorsAreFatal). Set it before
// the bindings are initialized.
var BindingBaseDebugZoneErrorsAreFatal = false

// Construct is Dart's BindingBase() constructor: it points the base at
// self (the binding embedding it) and initializes the binding through
// self.InitInstances and self.InitServiceExtensions.
func (b *BindingBaseBase) Construct(self BindingBase) {
	b.self = self
	if !KReleaseMode {
		FlutterTimeline.StartSync("Framework initialization")
	}
	if KDebugMode {
		b.debugConstructed = true
	}
	if bindingDebugInitializedType != nil {
		panic(dart.NewAssertionError("Binding is already initialized to " + bindingTypeName(bindingDebugInitializedType)))
	}
	self.InitInstances()
	dartAssert(bindingDebugInitializedType != nil, nil)
	dartAssert(!bindingDebugServiceExtensionsRegistered, nil)
	self.InitServiceExtensions()
	dartAssert(bindingDebugServiceExtensionsRegistered, nil)
	if !KReleaseMode {
		developer.PostEvent("Flutter.FrameworkInitialization", map[string]any{})
		FlutterTimeline.FinishSync()
	}
}

func (b *BindingBaseBase) bindingBase() *BindingBaseBase { return b }

func (b *BindingBaseBase) outer() BindingBase {
	if b.self == nil {
		panic("foundation: BindingBaseBase used before Construct; call Construct with the binding when creating it")
	}
	return b.self
}

// runtimeType is the binding's Dart runtimeType: its struct type.
func runtimeTypeOf(v any) reflect.Type {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func bindingTypeName(t reflect.Type) string { return dartTypeName(t) }

func (b *BindingBaseBase) Window() *ui.SingletonFlutterWindow { return ui.Window }

func (b *BindingBaseBase) PlatformDispatcher() *ui.PlatformDispatcher {
	return ui.PlatformDispatcherInstance()
}

func (b *BindingBaseBase) InitInstances() {
	dartAssert(bindingDebugInitializedType == nil, nil)
	if KDebugMode {
		bindingDebugInitializedType = runtimeTypeOf(b.outer())
		b.debugBindingZone = async.Current()
	}
}

// BindingBaseCheckInstance is BindingBase.checkInstance, for bindings'
// instance getters: in debug builds it panics with a FlutterError
// explaining the mistake when instance is nil or not fully initialized;
// it returns instance.
func BindingBaseCheckInstance[T BindingBase](instance T) T {
	typeName := dartTypeName(reflect.TypeFor[T]())
	isNil := isNull(instance)
	if KDebugMode {
		if bindingDebugInitializedType == nil && isNil {
			panic(NewFlutterErrorFromParts([]DiagnosticsNode{
				NewErrorSummary("Binding has not yet been initialized."),
				NewErrorDescription(`The "instance" getter on the ` + typeName + ` binding mixin is only available once that binding has been initialized.`),
				NewErrorHint(`Typically, this is done by calling "WidgetsFlutterBinding.ensureInitialized()" or "runApp()" (the ` +
					`latter calls the former). Typically this call is done in the "void main()" method. The "ensureInitialized" method ` +
					`is idempotent; calling it multiple times is not harmful. After calling that method, the "instance" getter will ` +
					`return the binding.`),
				NewErrorHint(`In a test, one can call "TestWidgetsFlutterBinding.ensureInitialized()" as the first line in the test's "main()" method ` +
					`to initialize the binding.`),
				NewErrorHint(`If ` + typeName + ` is a custom binding mixin, there must also be a custom binding class, like WidgetsFlutterBinding, ` +
					`but that mixes in the selected binding, and that is the class that must be constructed before using the "instance" getter.`),
			}))
		}
		if isNil {
			dartAssert(bindingDebugInitializedType == nil, nil)
			panic(NewFlutterErrorFromParts([]DiagnosticsNode{
				NewErrorSummary("Binding mixin instance is null but bindings are already initialized."),
				NewErrorDescription(`The "instance" property of the ` + typeName + ` binding mixin was accessed, but that binding was not initialized when ` +
					`the "initInstances()" method was called.`),
				NewErrorHint(`This probably indicates that the ` + typeName + ` mixin was not mixed into the class that was used to initialize the binding. ` +
					`If this is a custom binding mixin, there must also be a custom binding class, like WidgetsFlutterBinding, ` +
					`but that mixes in the selected binding. If this is a test binding, check that the binding being initialized ` +
					`is the same as the one into which the test binding is mixed.`),
				NewErrorHint(`It is also possible that ` + typeName + ` does not implement "initInstances()" to assign a value to "instance". See the ` +
					`documentation of the BindingBase class for more details.`),
				NewErrorHint(`The binding that was initialized was of the type "` + typeNameOrNull(bindingDebugInitializedType) + `". `),
			}))
		}
		// Dart's `on NoSuchMethodError` branch ("Binding does not extend
		// BindingBase") can't happen: a BindingBase embeds BindingBaseBase.
		base := instance.bindingBase()
		if base.debugConstructed && bindingDebugInitializedType == nil {
			panic(NewFlutterErrorFromParts([]DiagnosticsNode{
				NewErrorSummary("Binding initialized without calling initInstances."),
				NewErrorDescription(`An instance of ` + typeName + ` is non-null, but BindingBase.initInstances() has not yet been called.`),
				NewErrorHint(`This could happen because a binding mixin was somehow used outside of the normal binding mechanisms, or because ` +
					`the binding's initInstances() method did not call "super.initInstances()".`),
				NewErrorHint(`This could also happen if some code was invoked that used the binding while the binding was initializing, ` +
					`for example if the "initInstances" method invokes a callback. Bindings should not invoke callbacks before ` +
					`"initInstances" has completed.`),
			}))
		}
		if !base.debugConstructed {
			panic(NewFlutterErrorFromParts([]DiagnosticsNode{
				NewErrorSummary("Binding did not complete initialization."),
				NewErrorDescription(`An instance of ` + typeName + ` is non-null, but the BindingBase() constructor has not yet been called.`),
				NewErrorHint(`This could also happen if some code was invoked that used the binding while the binding was initializing, ` +
					`for example if the binding's constructor itself invokes a callback. Bindings should not invoke callbacks ` +
					`before "initInstances" has completed.`),
			}))
		}
	}
	if isNil {
		panic(dart.StateError{Message: "Null check operator used on a null value"})
	}
	return instance
}

func typeNameOrNull(t reflect.Type) string {
	if t == nil {
		return "null"
	}
	return dartTypeName(t)
}

// BindingBaseDebugBindingType is BindingBase.debugBindingType: the type of
// the initialized binding, or nil (always nil outside debug builds).
func BindingBaseDebugBindingType() reflect.Type { return bindingDebugInitializedType }

func (b *BindingBaseBase) DebugCheckZone(entryPoint string) bool {
	if KDebugMode {
		dartAssert(b.debugBindingZone != nil, "debugCheckZone can only be used after the binding is fully initialized.")
		if async.Current() != b.debugBindingZone {
			toggle, value := "warning fatal", "true"
			if BindingBaseDebugZoneErrorsAreFatal {
				toggle, value = "error non-fatal", "false"
			}
			message := NewFlutterError("Zone mismatch.\n" +
				"The Flutter bindings were initialized in a different zone than is now being used. " +
				"This will likely cause confusion and bugs as any zone-specific configuration will " +
				"inconsistently use the configuration of the original binding initialization zone " +
				"or this zone based on hard-to-predict factors such as which zone was active when " +
				"a particular callback was set.\n" +
				"It is important to use the same zone when calling `ensureInitialized` on the binding " +
				"as when calling `" + entryPoint + "` later.\n" +
				"To make this " + toggle + ", " +
				"set BindingBase.debugZoneErrorsAreFatal to " + value + " before the " +
				"bindings are initialized (i.e. as the first statement in `void main() { }`).")
			if BindingBaseDebugZoneErrorsAreFatal {
				panic(message)
			}
			FlutterErrorReportError(NewFlutterErrorDetails(message, FlutterErrorDetailsOptions{
				Stack:   dart.StackTraceCurrent(),
				Context: NewErrorDescription("during " + entryPoint),
			}))
		}
	}
	return true
}

func (b *BindingBaseBase) InitServiceExtensions() {
	self := b.outer()
	dartAssert(!bindingDebugServiceExtensionsRegistered, nil)
	if KDebugMode {
		self.RegisterSignalServiceExtension(FoundationServiceExtensionsReassemble.Name(), self.ReassembleApplication)
	}
	if !KReleaseMode {
		if !KIsWeb {
			self.RegisterSignalServiceExtension(FoundationServiceExtensionsExit.Name(), exitApplication)
		}
		self.RegisterStringServiceExtension(FoundationServiceExtensionsConnectedVmServiceUri.Name(),
			func() async.Future[string] { return asyncValue(stringOrEmpty(ConnectedVmServiceUri)) },
			func(uri string) async.Future[async.Void] {
				return async.AsyncVoid(func() { ConnectedVmServiceUri = &uri })
			})
		self.RegisterStringServiceExtension(FoundationServiceExtensionsActiveDevToolsServerAddress.Name(),
			func() async.Future[string] { return asyncValue(stringOrEmpty(ActiveDevToolsServerAddress)) },
			func(serverAddress string) async.Future[async.Void] {
				return async.AsyncVoid(func() { ActiveDevToolsServerAddress = &serverAddress })
			})
	}
	if KDebugMode {
		self.RegisterServiceExtension(FoundationServiceExtensionsPlatformOverride.Name(),
			func(parameters map[string]string) async.Future[map[string]any] {
				return async.Async(func() map[string]any {
					if value, ok := parameters["value"]; ok {
						SetDebugDefaultTargetPlatformOverride(nil)
						for _, candidate := range TargetPlatformValues {
							if candidate.Name() == value {
								SetDebugDefaultTargetPlatformOverride(&candidate)
								break
							}
						}
						b.postExtensionStateChangedEvent(FoundationServiceExtensionsPlatformOverride.Name(), DefaultTargetPlatform().Name())
						self.ReassembleApplication().Await()
					}
					return map[string]any{"value": DefaultTargetPlatform().Name()}
				})
			})
		self.RegisterServiceExtension(FoundationServiceExtensionsBrightnessOverride.Name(),
			func(parameters map[string]string) async.Future[map[string]any] {
				return async.Async(func() map[string]any {
					if value, ok := parameters["value"]; ok {
						var override *ui.Brightness
						switch value {
						case "Brightness.light":
							override = ptrTo(ui.BrightnessLight)
						case "Brightness.dark":
							override = ptrTo(ui.BrightnessDark)
						}
						DebugBrightnessOverride = override
						b.postExtensionStateChangedEvent(FoundationServiceExtensionsBrightnessOverride.Name(),
							b.effectiveBrightness().String())
						self.ReassembleApplication().Await()
					}
					return map[string]any{"value": b.effectiveBrightness().String()}
				})
			})
		bindingDebugServiceExtensionsRegistered = true
	}
}

func (b *BindingBaseBase) effectiveBrightness() ui.Brightness {
	if DebugBrightnessOverride != nil {
		return *DebugBrightnessOverride
	}
	return b.outer().PlatformDispatcher().PlatformBrightness()
}

func ptrTo[T any](v T) *T { return &v }

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// asyncValue is the future of an async function returning v without
// awaiting (`() async => v`).
func asyncValue[T any](v T) async.Future[T] { return async.Async(func() T { return v }) }

func (b *BindingBaseBase) Locked() bool { return b.lockCount > 0 }

func (b *BindingBaseBase) LockEvents(callback func() async.Future[async.Void]) async.Future[async.Void] {
	self := b.outer()
	var debugTimelineTask *developer.TimelineTask
	if !KReleaseMode {
		debugTimelineTask = developer.NewTimelineTask()
		debugTimelineTask.Start("Lock events", nil)
	}
	b.lockCount++
	future := callback()
	future.WhenComplete(func() {
		b.lockCount--
		if !self.Locked() {
			if !KReleaseMode {
				debugTimelineTask.Finish(nil)
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						FlutterErrorReportError(NewFlutterErrorDetails(r, FlutterErrorDetailsOptions{
							Stack:   dart.StackTraceFromPanic(),
							Library: strp("foundation"),
							Context: NewErrorDescription("while handling pending events"),
						}))
					}
				}()
				self.Unlocked()
			}()
		}
	})
	return future
}

func (b *BindingBaseBase) Unlocked() { dartAssert(!b.outer().Locked(), nil) }

func (b *BindingBaseBase) ReassembleApplication() async.Future[async.Void] {
	return b.outer().LockEvents(b.outer().PerformReassemble)
}

func (b *BindingBaseBase) PerformReassemble() async.Future[async.Void] {
	FlutterErrorResetErrorCount()
	return async.FutureValue(async.Void{})
}

func (b *BindingBaseBase) RegisterSignalServiceExtension(name string, callback AsyncCallback) {
	b.outer().RegisterServiceExtension(name, func(map[string]string) async.Future[map[string]any] {
		return async.Async(func() map[string]any {
			callback().Await()
			return map[string]any{}
		})
	})
}

func (b *BindingBaseBase) RegisterBoolServiceExtension(name string, getter AsyncValueGetter[bool], setter AsyncValueSetter[bool]) {
	boolString := func(v bool) string {
		if v {
			return "true"
		}
		return "false"
	}
	b.outer().RegisterServiceExtension(name, func(parameters map[string]string) async.Future[map[string]any] {
		return async.Async(func() map[string]any {
			if enabled, ok := parameters["enabled"]; ok {
				setter(enabled == "true").Await()
				b.postExtensionStateChangedEvent(name, boolString(getter().Await()))
			}
			return map[string]any{"enabled": boolString(getter().Await())}
		})
	})
}

func (b *BindingBaseBase) RegisterNumericServiceExtension(name string, getter AsyncValueGetter[float64], setter AsyncValueSetter[float64]) {
	b.outer().RegisterServiceExtension(name, func(parameters map[string]string) async.Future[map[string]any] {
		return async.Async(func() map[string]any {
			if value, ok := parameters[name]; ok {
				setter(parseDouble(value)).Await()
				b.postExtensionStateChangedEvent(name, dart.DoubleToString(getter().Await()))
			}
			return map[string]any{name: dart.DoubleToString(getter().Await())}
		})
	})
}

// parseDouble is double.parse: decimal or exponent notation, "NaN" and
// "Infinity" with an optional sign, surrounded by optional whitespace;
// anything else panics with a FormatException.
func parseDouble(source string) float64 {
	s := strings.TrimSpace(source)
	unsigned := strings.TrimLeft(s, "+-")
	switch unsigned {
	case "NaN":
		return math.NaN()
	case "Infinity":
		if strings.HasPrefix(s, "-") {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	if len(s)-len(unsigned) > 1 || strings.ContainsAny(unsigned, "_xXpPiInN") {
		panic(FormatException{Message: "Invalid double", Source: source})
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		panic(FormatException{Message: "Invalid double", Source: source})
	}
	return v
}

// FormatException is dart:core's FormatException(message, source).
type FormatException struct {
	Message string
	Source  string
}

func (e FormatException) Error() string { return "FormatException: " + e.Message + "\n" + e.Source }

func (b *BindingBaseBase) postExtensionStateChangedEvent(name string, value any) {
	b.outer().PostEvent("Flutter.ServiceExtensionStateChanged", map[string]any{
		"extension": "ext.flutter." + name,
		"value":     value,
	})
}

func (b *BindingBaseBase) PostEvent(eventKind string, eventData map[string]any) {
	developer.PostEvent(eventKind, eventData)
}

func (b *BindingBaseBase) RegisterStringServiceExtension(name string, getter AsyncValueGetter[string], setter AsyncValueSetter[string]) {
	b.outer().RegisterServiceExtension(name, func(parameters map[string]string) async.Future[map[string]any] {
		return async.Async(func() map[string]any {
			if value, ok := parameters["value"]; ok {
				setter(value).Await()
				b.postExtensionStateChangedEvent(name, getter().Await())
			}
			return map[string]any{"value": getter().Await()}
		})
	})
}

// RegisterServiceExtension registers "ext.flutter.<name>" with
// developer.RegisterExtension. When called, the extension waits for the
// outer event loop (a zero-duration timer), runs callback and answers with
// its JSON-encoded result, or reports the error with
// FlutterErrorReportError and answers with an extension error.
//
// ENGINE GAP: nothing calls the handler but developer.LookupExtension (no
// VM service).
func (b *BindingBaseBase) RegisterServiceExtension(name string, callback ServiceExtensionCallback) {
	methodName := "ext.flutter." + name
	developer.RegisterExtension(methodName, func(method string, parameters map[string]string) async.Future[*developer.ServiceExtensionResponse] {
		return async.Async(func() *developer.ServiceExtensionResponse {
			dartAssert(method == methodName, nil)
			if KDebugMode && DebugInstrumentationEnabled {
				DebugPrint("service extension method received: " + method + "(" + dartMapString(parameters) + ")")
			}
			DebugInstrumentAction("Wait for outer event loop", func() async.Future[async.Void] {
				return async.FutureDelayed[async.Void](0, nil)
			}).Await()
			var response *developer.ServiceExtensionResponse
			func() {
				defer func() {
					if exception := recover(); exception != nil {
						stack := dart.StackTraceFromPanic()
						FlutterErrorReportError(NewFlutterErrorDetails(exception, FlutterErrorDetailsOptions{
							Stack:   stack,
							Context: NewErrorDescription(`during a service extension callback for "` + method + `"`),
						}))
						response = developer.NewServiceExtensionResponseError(
							developer.ServiceExtensionResponseExtensionError,
							dart.JSONEncode([]dart.MapEntry[string, any]{
								{Key: "exception", Value: dartToString(exception)},
								{Key: "stack", Value: stackTraceString(stack)},
								{Key: "method", Value: method},
							}))
					}
				}()
				result := callback(parameters).Await()
				response = developer.NewServiceExtensionResponseResult(dart.JSONEncode(result))
			}()
			return response
		})
	})
}

func stackTraceString(stack dart.StackTrace) string {
	if stack == nil {
		return "null"
	}
	return stack.String()
}

// dartMapString is Map<String, String>.toString ("{a: 1, b: 2}"); a Go map
// has no insertion order, so keys are sorted.
func dartMapString(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + m[k]
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (b *BindingBaseBase) String() string {
	return "<" + ObjectRuntimeType(b.outer(), "BindingBase") + ">"
}

// exitApplication is _exitApplication: exit(0).
func exitApplication() async.Future[async.Void] {
	return async.AsyncVoid(func() { os.Exit(0) })
}
