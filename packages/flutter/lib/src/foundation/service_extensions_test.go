// Port of the foundation part of
// packages/flutter/test/foundation/service_extensions_test.dart.
//
// Ported (6 of 31 tests): exit, platformOverride, reassemble,
// brightnessOverride, activeDevToolsServerAddress, connectedVmServiceUri:
// the extensions BindingBase registers. Their TestServiceExtensionsBinding
// test hooks (registerServiceExtension into a map, postEvent recorded,
// performReassemble counted) are reproduced. Skipped: the frame checks of
// hasReassemble and the reassemble test (frameScheduled, doFrame: they need
// SchedulerBinding and WidgetsBinding), and the 25 other tests, which test
// the extensions of the scheduler, services, gestures, painting, semantics,
// rendering and widgets bindings (not ported yet), as does the tearDownAll
// count of every framework extension.
package foundation

import (
	"slices"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type testServiceExtensionsBinding struct {
	BindingBaseBase
	extensions        map[string]ServiceExtensionCallback
	eventsDispatched  map[string][]map[string]any
	reassembled       int
	pendingReassemble bool
}

func (b *testServiceExtensionsBinding) RegisterServiceExtension(name string, callback ServiceExtensionCallback) {
	if _, ok := b.extensions[name]; ok {
		panic("extension registered twice: " + name)
	}
	b.extensions[name] = callback
}

func (b *testServiceExtensionsBinding) PostEvent(eventKind string, eventData map[string]any) {
	b.eventsDispatched[eventKind] = append(b.eventsDispatched[eventKind], eventData)
}

func (b *testServiceExtensionsBinding) getServiceExtensionStateChangedEvents(extensionName string) []map[string]any {
	var out []map[string]any
	for _, event := range b.eventsDispatched["Flutter.ServiceExtensionStateChanged"] {
		if event["extension"] == extensionName {
			out = append(out, event)
		}
	}
	return out
}

func (b *testServiceExtensionsBinding) testExtension(t *testing.T, name string, arguments map[string]string) async.Future[map[string]any] {
	t.Helper()
	callback, ok := b.extensions[name]
	if !ok {
		t.Fatalf("no extension %q", name)
	}
	return callback(arguments)
}

func (b *testServiceExtensionsBinding) PerformReassemble() async.Future[async.Void] {
	b.reassembled++
	b.pendingReassemble = true
	return b.BindingBaseBase.PerformReassemble()
}

func newTestServiceExtensionsBinding(t *testing.T) *testServiceExtensionsBinding {
	resetBindingStatics(t)
	b := &testServiceExtensionsBinding{
		extensions:       map[string]ServiceExtensionCallback{},
		eventsDispatched: map[string][]map[string]any{},
	}
	b.Construct(b)
	return b
}

func TestServiceExtensionsFoundationCount(t *testing.T) {
	b := newTestServiceExtensionsBinding(t)
	var names []string
	for name := range b.extensions {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"activeDevToolsServerAddress", "brightnessOverride",
		"connectedVmServiceUri", "exit", "platformOverride", "reassemble"}) {
		t.Fatal(names)
	}
}

func TestServiceExtensionsExit(t *testing.T) {
	b := newTestServiceExtensionsBinding(t)
	// No test for calling exit: it would terminate the process.
	if _, ok := b.extensions[FoundationServiceExtensionsExit.Name()]; !ok {
		t.Fatal("exit not registered")
	}
}

func TestServiceExtensionsPlatformOverride(t *testing.T) {
	t.Setenv("FLUTTER_TEST", "true") // flutter test's environment: android by default
	b := newTestServiceExtensionsBinding(t)
	defer SetDebugDefaultTargetPlatformOverride(nil)
	name := FoundationServiceExtensionsPlatformOverride.Name()
	equal(t, b.reassembled, 0)
	equal(t, DefaultTargetPlatform(), TargetPlatformAndroid)
	check := func(arguments map[string]string, wantValue string, wantReassembled int, want TargetPlatform, wantEvents int) {
		t.Helper()
		var result map[string]any
		if err := runLoop(t, func() { result = b.testExtension(t, name, arguments).Await() }); err != nil {
			t.Fatal(err)
		}
		equal(t, dart.JSONEncode(result), `{"value":"`+wantValue+`"}`)
		equal(t, b.reassembled, wantReassembled)
		equal(t, DefaultTargetPlatform(), want)
		events := b.getServiceExtensionStateChangedEvents("ext.flutter.platformOverride")
		equal(t, len(events), wantEvents)
		if wantEvents > 0 {
			last := events[len(events)-1]
			equal(t, last["extension"], any("ext.flutter.platformOverride"))
			equal(t, last["value"], any(wantValue))
		}
	}
	check(map[string]string{}, "android", 0, TargetPlatformAndroid, 0)
	check(map[string]string{"value": "iOS"}, "iOS", 1, TargetPlatformIOS, 1)
	check(map[string]string{"value": "macOS"}, "macOS", 2, TargetPlatformMacOS, 2)
	check(map[string]string{"value": "android"}, "android", 3, TargetPlatformAndroid, 3)
	check(map[string]string{"value": "fuchsia"}, "fuchsia", 4, TargetPlatformFuchsia, 4)
	check(map[string]string{"value": "default"}, "android", 5, TargetPlatformAndroid, 5)
	check(map[string]string{"value": "iOS"}, "iOS", 6, TargetPlatformIOS, 6)
	check(map[string]string{"value": "linux"}, "linux", 7, TargetPlatformLinux, 7)
	check(map[string]string{"value": "windows"}, "windows", 8, TargetPlatformWindows, 8)
	check(map[string]string{"value": "bogus"}, "android", 9, TargetPlatformAndroid, 9)
	b.reassembled = 0
}

func TestServiceExtensionsReassemble(t *testing.T) {
	b := newTestServiceExtensionsBinding(t)
	equal(t, b.reassembled, 0)
	var result map[string]any
	if err := runLoop(t, func() {
		result = b.testExtension(t, FoundationServiceExtensionsReassemble.Name(), map[string]string{}).Await()
	}); err != nil {
		t.Fatal(err)
	}
	equal(t, dart.JSONEncode(result), `{}`)
	equal(t, b.reassembled, 1)
}

func TestServiceExtensionsBrightnessOverride(t *testing.T) {
	b := newTestServiceExtensionsBinding(t)
	defer func() { DebugBrightnessOverride = nil }()
	var results []map[string]any
	if err := runLoop(t, func() {
		ext := FoundationServiceExtensionsBrightnessOverride.Name()
		results = append(results,
			b.testExtension(t, ext, map[string]string{}).Await(),
			b.testExtension(t, ext, map[string]string{"value": "Brightness.dark"}).Await(),
			b.testExtension(t, ext, map[string]string{"value": "Brightness.light"}).Await(),
			b.testExtension(t, ext, map[string]string{"value": "bogus"}).Await(),
		)
	}); err != nil {
		t.Fatal(err)
	}
	// Flutter's test checks the first; the others are this port's.
	for i, want := range []string{"Brightness.light", "Brightness.dark", "Brightness.light", "Brightness.light"} {
		equal(t, results[i]["value"], any(want))
	}
	equal(t, b.reassembled, 3)
	if DebugBrightnessOverride != nil {
		t.Fatal("bogus must reset the override")
	}
	events := b.getServiceExtensionStateChangedEvents("ext.flutter.brightnessOverride")
	equal(t, len(events), 3)
	equal(t, events[0]["value"], any(ui.BrightnessDark.String()))
}

func testStringExtension(t *testing.T, name string, values ...string) {
	b := newTestServiceExtensionsBinding(t)
	var got []any
	if err := runLoop(t, func() {
		got = append(got, b.testExtension(t, name, map[string]string{}).Await()["value"])
		for _, v := range values {
			got = append(got, b.testExtension(t, name, map[string]string{"value": v}).Await()["value"])
		}
	}); err != nil {
		t.Fatal(err)
	}
	equal(t, got[0], any(""))
	for i, v := range values {
		equal(t, got[i+1], any(v))
	}
}

func TestServiceExtensionsActiveDevToolsServerAddress(t *testing.T) {
	defer func() { ActiveDevToolsServerAddress = nil }()
	testStringExtension(t, FoundationServiceExtensionsActiveDevToolsServerAddress.Name(),
		"http://127.0.0.1:9101", "http://127.0.0.1:9102")
}

func TestServiceExtensionsConnectedVmServiceUri(t *testing.T) {
	defer func() { ConnectedVmServiceUri = nil }()
	testStringExtension(t, FoundationServiceExtensionsConnectedVmServiceUri.Name(),
		"http://127.0.0.1:54669/kMUMseKAnog=/", "http://127.0.0.1:54000/kMUMseKAnog=/")
}
