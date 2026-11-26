// Port of packages/flutter/test/foundation/binding_test.dart (its one
// test), plus tests of BindingBase's construction, checkInstance errors,
// lockEvents and the real service-extension registration (through
// package developer), which Flutter exercises only through bigger
// bindings.
//
// Dart runs each test file in its own isolate; Go runs the package's tests
// in one process, so the binding statics (the initialized type, "service
// extensions registered") are reset between tests, and only this file's
// binding registers its extensions with package developer (the others
// override RegisterServiceExtension, as Flutter's service_extensions_test
// does).
package foundation

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/developer"
)

// resetBindingStatics forgets the initialized binding (a new isolate, in
// Dart).
func resetBindingStatics(t *testing.T) {
	bindingDebugInitializedType = nil
	bindingDebugServiceExtensionsRegistered = false
	t.Cleanup(func() {
		bindingDebugInitializedType = nil
		bindingDebugServiceExtensionsRegistered = false
	})
}

// FooBinding is binding_test.dart's `mixin FooBinding on BindingBase`.
type FooBinding interface {
	BindingBase
	isFooBinding()
}

var fooBindingInstance FooBinding

func FooBindingInstance() FooBinding { return BindingBaseCheckInstance(fooBindingInstance) }

type FooLibraryBinding struct {
	BindingBaseBase
}

func (b *FooLibraryBinding) isFooBinding() {}

func (b *FooLibraryBinding) InitInstances() {
	b.BindingBaseBase.InitInstances()
	fooBindingInstance = b
}

func fooLibraryBindingEnsureInitialized() FooBinding {
	if fooBindingInstance == nil {
		b := &FooLibraryBinding{}
		b.Construct(b)
	}
	return FooBindingInstance()
}

var fooBindingRegistered = false

func TestBindingBaseDebugBindingType(t *testing.T) {
	resetBindingStatics(t)
	if fooBindingRegistered {
		t.Skip("the developer registry already has this binding's extensions")
	}
	fooBindingRegistered = true
	fooBindingInstance = nil
	if BindingBaseDebugBindingType() != nil {
		t.Fatal("not null before initialization")
	}
	binding := fooLibraryBindingEnsureInitialized()
	if BindingBaseDebugBindingType() != reflect.TypeFor[FooLibraryBinding]() {
		t.Fatal(BindingBaseDebugBindingType())
	}
	equal(t, binding.String(), "<FooLibraryBinding>")
	if FooBindingInstance() != binding || fooLibraryBindingEnsureInitialized() != binding {
		t.Fatal("instance")
	}

	// The extensions were registered with dart:developer, and invoking one
	// waits for the outer event loop, runs it and encodes its result.
	for _, name := range []string{"reassemble", "exit", "connectedVmServiceUri",
		"activeDevToolsServerAddress", "platformOverride", "brightnessOverride"} {
		if developer.LookupExtension("ext.flutter."+name) == nil {
			t.Fatalf("ext.flutter.%s not registered", name)
		}
	}
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	handler := developer.LookupExtension("ext.flutter.activeDevToolsServerAddress")
	var response *developer.ServiceExtensionResponse
	async.OnComplete(handler("ext.flutter.activeDevToolsServerAddress", map[string]string{"value": `a"b`}),
		func(r *developer.ServiceExtensionResponse) { response = r }, nil)
	l.FlushMicrotasks()
	if response != nil {
		t.Fatal("answered before the outer event loop turned")
	}
	l.FlushTimers(time.Hour, true)
	if response == nil || response.IsError() || *response.Result != `{"value":"a\"b"}` {
		t.Fatalf("response = %#v", response)
	}
	equal(t, *ActiveDevToolsServerAddress, `a"b`)
	ActiveDevToolsServerAddress = nil

	// A second registration of the same name is an ArgumentError.
	expectPanic(t, func(r any) bool {
		e, ok := r.(dart.ArgumentError)
		return ok && e.Error() == "Invalid argument(s): Extension already registered: ext.flutter.exit"
	}, func() { binding.RegisterServiceExtension("exit", nil) })

	// A failing extension reports the error and answers with an extension
	// error whose detail is the exception, the stack and the method.
	var reported []*FlutterErrorDetails
	originalOnError := FlutterErrorOnError
	FlutterErrorOnError = func(d *FlutterErrorDetails) { reported = append(reported, d) }
	defer func() { FlutterErrorOnError = originalOnError }()
	binding.RegisterServiceExtension("failing", func(map[string]string) async.Future[map[string]any] {
		return async.FutureError[map[string]any](errors.New("boom"))
	})
	response = nil
	async.OnComplete(developer.LookupExtension("ext.flutter.failing")("ext.flutter.failing", nil),
		func(r *developer.ServiceExtensionResponse) { response = r }, nil)
	l.FlushTimers(time.Hour, true)
	if response == nil || !response.IsError() || *response.ErrorCode != developer.ServiceExtensionResponseExtensionError {
		t.Fatalf("response = %#v", response)
	}
	if !strings.HasPrefix(*response.ErrorDetail, `{"exception":"boom","stack":"`) ||
		!strings.HasSuffix(*response.ErrorDetail, `","method":"ext.flutter.failing"}`) {
		t.Fatal(*response.ErrorDetail)
	}
	equal(t, len(reported), 1)
	equal(t, reported[0].Context.String(), `during a service extension callback for "ext.flutter.failing"`)
}

// Not in Flutter's tests.

type plainBinding struct {
	BindingBaseBase
	extensions map[string]ServiceExtensionCallback
	events     []map[string]any
	unlocked   int
}

func (b *plainBinding) RegisterServiceExtension(name string, callback ServiceExtensionCallback) {
	if b.extensions == nil {
		b.extensions = map[string]ServiceExtensionCallback{}
	}
	if _, ok := b.extensions[name]; ok {
		panic("registered twice: " + name)
	}
	b.extensions[name] = callback
}

func (b *plainBinding) PostEvent(eventKind string, eventData map[string]any) {
	b.events = append(b.events, map[string]any{"kind": eventKind, "data": eventData})
}

func (b *plainBinding) Unlocked() {
	b.BindingBaseBase.Unlocked()
	b.unlocked++
}

func newPlainBinding() *plainBinding {
	b := &plainBinding{}
	b.Construct(b)
	return b
}

func TestBindingBaseConstructionAndCheckInstance(t *testing.T) {
	resetBindingStatics(t)
	isFlutterError := func(summary string) func(any) bool {
		return func(r any) bool {
			e, ok := r.(*FlutterError)
			return ok && strings.HasPrefix(e.Error(), summary)
		}
	}
	var none *plainBinding
	expectPanic(t, isFlutterError("Binding has not yet been initialized."), func() { BindingBaseCheckInstance(none) })

	b := newPlainBinding()
	equal(t, len(b.extensions), 6)
	// Flutter quirk: checkInstance asserts `_debugInitializedType == null`
	// before throwing "Binding mixin instance is null but bindings are
	// already initialized.", so with assertions on that FlutterError is
	// unreachable and a bare AssertionError is thrown instead.
	expectPanic(t, func(r any) bool {
		e, ok := r.(*dart.AssertionError)
		return ok && e.AssertionMessage() == nil
	}, func() { BindingBaseCheckInstance(none) })
	if BindingBaseCheckInstance(b) != b {
		t.Fatal("checkInstance")
	}
	// Initializing a second binding is an assertion.
	expectPanic(t, func(r any) bool {
		e, ok := r.(*dart.AssertionError)
		return ok && e.AssertionMessage() == "Binding is already initialized to plainBinding"
	}, func() { newPlainBinding() })
	// The failed constructor left its "Framework initialization" timeline
	// block open (as in Dart); close it for the other tests.
	FlutterTimeline.FinishSync()
	// A binding that was never constructed.
	expectPanic(t, isFlutterError("Binding did not complete initialization."),
		func() { BindingBaseCheckInstance(&plainBinding{}) })
	// Constructed, but the statics were reset (initInstances not called).
	bindingDebugInitializedType = nil
	expectPanic(t, isFlutterError("Binding initialized without calling initInstances."),
		func() { BindingBaseCheckInstance(b) })
	equal(t, b.Window().PlatformBrightness().String(), "Brightness.light")
	if b.PlatformDispatcher() != b.Window().PlatformDispatcher() {
		t.Fatal("platformDispatcher")
	}
}

func TestBindingBaseLockEvents(t *testing.T) {
	resetBindingStatics(t)
	b := newPlainBinding()
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	first, second := async.NewCompleter[async.Void](), async.NewCompleter[async.Void]()
	equal(t, b.Locked(), false)
	b.LockEvents(first.Future)
	b.LockEvents(second.Future)
	equal(t, b.Locked(), true)
	first.Complete(async.Void{})
	l.FlushMicrotasks()
	equal(t, b.Locked(), true)
	equal(t, b.unlocked, 0)
	second.Complete(async.Void{})
	l.FlushMicrotasks()
	equal(t, b.Locked(), false)
	equal(t, b.unlocked, 1)

	// An Unlocked that panics is reported, not propagated.
	var reported []*FlutterErrorDetails
	originalOnError := FlutterErrorOnError
	FlutterErrorOnError = func(d *FlutterErrorDetails) { reported = append(reported, d) }
	defer func() { FlutterErrorOnError = originalOnError }()
	bad := &badUnlockBinding{}
	bindingDebugInitializedType = nil
	bindingDebugServiceExtensionsRegistered = false
	bad.Construct(bad)
	bad.LockEvents(func() async.Future[async.Void] { return async.FutureValue(async.Void{}) })
	l.FlushMicrotasks()
	equal(t, len(reported), 1)
	equal(t, *reported[0].Library, "foundation")
	equal(t, reported[0].Context.String(), "while handling pending events")
}

type badUnlockBinding struct{ plainBinding }

func (b *badUnlockBinding) Unlocked() { panic("unlock failed") }

func TestBindingBaseRegisterBoolAndNumericServiceExtensions(t *testing.T) {
	resetBindingStatics(t)
	b := newPlainBinding()
	enabled := false
	value := 1.0
	b.RegisterBoolServiceExtension("flag",
		func() async.Future[bool] { return async.FutureValue(enabled) },
		func(v bool) async.Future[async.Void] { enabled = v; return async.FutureValue(async.Void{}) })
	b.RegisterNumericServiceExtension("scale",
		func() async.Future[float64] { return async.FutureValue(value) },
		func(v float64) async.Future[async.Void] { value = v; return async.FutureValue(async.Void{}) })
	var results []map[string]any
	err := runLoop(t, func() {
		results = append(results,
			b.extensions["flag"](map[string]string{}).Await(),
			b.extensions["flag"](map[string]string{"enabled": "true"}).Await(),
			b.extensions["flag"](map[string]string{"enabled": "yes"}).Await(),
			b.extensions["scale"](map[string]string{}).Await(),
			b.extensions["scale"](map[string]string{"scale": " 2.5 "}).Await(),
			b.extensions["scale"](map[string]string{"value": "3"}).Await(),
		)
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`{"enabled":"false"}`, `{"enabled":"true"}`, `{"enabled":"false"}`,
		`{"scale":"1.0"}`, `{"scale":"2.5"}`, `{"scale":"2.5"}`}
	for i, r := range results {
		equal(t, dart.JSONEncode(r), want[i])
	}
	var events []string
	for _, e := range b.events {
		events = append(events, e["kind"].(string)+" "+dart.JSONEncode(e["data"]))
	}
	equal(t, strings.Join(events, "\n"), strings.Join([]string{
		`Flutter.ServiceExtensionStateChanged {"extension":"ext.flutter.flag","value":"true"}`,
		`Flutter.ServiceExtensionStateChanged {"extension":"ext.flutter.flag","value":"false"}`,
		`Flutter.ServiceExtensionStateChanged {"extension":"ext.flutter.scale","value":"2.5"}`,
	}, "\n"))
	// double.parse rejects what Dart rejects.
	for _, bad := range []string{"", "abc", "0x10", "1_0", "inf", "--1"} {
		expectPanic(t, func(r any) bool { _, ok := r.(FormatException); return ok }, func() { parseDouble(bad) })
	}
	equal(t, dart.DoubleToString(parseDouble("-Infinity")), "-Infinity")
	equal(t, dart.DoubleToString(parseDouble("1e400")), "Infinity")
	equal(t, dart.DoubleToString(parseDouble(".5")), "0.5")
}
