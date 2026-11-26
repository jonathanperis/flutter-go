package services_test

// Port of packages/flutter/test/services/hardware_keyboard_test.dart
// (Flutter 3.50.1), 12/12. Adapted: the tests that pump a KeyboardListener
// (or RawKeyboardListener) to observe the events the focus system receives
// install a KeyEventManager.keyMessageHandler doing what FocusManager's
// does for a focused listener (each KeyMessage's events, then its raw
// event), since widgets' focus system is not ported yet.

import (
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func expectState(expected map[services.PhysicalKeyboardKey]services.LogicalKeyboardKey) {
	var physical []services.PhysicalKeyboardKey
	var logical []services.LogicalKeyboardKey
	for p, l := range expected {
		physical = append(physical, p)
		if !slicesContains(logical, l) {
			logical = append(logical, l)
		}
	}
	expectSet("physicalKeysPressed", services.HardwareKeyboardInstance().PhysicalKeysPressed(), physical...)
	expectSet("logicalKeysPressed", services.HardwareKeyboardInstance().LogicalKeysPressed(), logical...)
}

func slicesContains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// listenerHandler is a focused KeyboardListener's view of the key messages:
// onKeyEvent for each event (and onKey for the raw event).
func listenerHandler(onKeyEvent func(services.KeyEvent), onKey func(services.RawKeyEvent)) services.KeyMessageHandler {
	return func(message *services.KeyMessage) bool {
		if onKey != nil && message.RawEvent != nil {
			onKey(message.RawEvent)
		}
		for _, e := range message.Events {
			if onKeyEvent != nil {
				onKeyEvent(e)
			}
		}
		return false
	}
}

func TestHardwareKeyboardRecordsPressedKeysAndEnabledLocks(t *testing.T) {
	keyTest(t, modeKeyDataThenRawKeyData, func() {
		hk := services.HardwareKeyboardInstance()
		check := func(physical []services.PhysicalKeyboardKey, logical []services.LogicalKeyboardKey, locks ...services.KeyboardLockMode) {
			expectSet("physical", hk.PhysicalKeysPressed(), physical...)
			expectSet("logical", hk.LogicalKeysPressed(), logical...)
			expectSet("locks", hk.LockModesEnabled(), locks...)
		}
		type P = []services.PhysicalKeyboardKey
		type L = []services.LogicalKeyboardKey
		keyDown(services.LogicalKeyboardKeyNumLock, on("windows"))
		check(P{services.PhysicalKeyboardKeyNumLock}, L{services.LogicalKeyboardKeyNumLock}, services.KeyboardLockModeNumLock)
		keyDown(services.LogicalKeyboardKeyNumpad1, on("windows"))
		check(P{services.PhysicalKeyboardKeyNumLock, services.PhysicalKeyboardKeyNumpad1},
			L{services.LogicalKeyboardKeyNumLock, services.LogicalKeyboardKeyNumpad1}, services.KeyboardLockModeNumLock)
		keyRepeat(services.LogicalKeyboardKeyNumpad1, on("windows"))
		check(P{services.PhysicalKeyboardKeyNumLock, services.PhysicalKeyboardKeyNumpad1},
			L{services.LogicalKeyboardKeyNumLock, services.LogicalKeyboardKeyNumpad1}, services.KeyboardLockModeNumLock)
		keyUp(services.LogicalKeyboardKeyNumLock)
		check(P{services.PhysicalKeyboardKeyNumpad1}, L{services.LogicalKeyboardKeyNumpad1}, services.KeyboardLockModeNumLock)
		keyDown(services.LogicalKeyboardKeyNumLock, on("windows"))
		check(P{services.PhysicalKeyboardKeyNumLock, services.PhysicalKeyboardKeyNumpad1},
			L{services.LogicalKeyboardKeyNumLock, services.LogicalKeyboardKeyNumpad1})
		keyUp(services.LogicalKeyboardKeyNumpad1, on("windows"))
		check(P{services.PhysicalKeyboardKeyNumLock}, L{services.LogicalKeyboardKeyNumLock})
		keyUp(services.LogicalKeyboardKeyNumLock, on("windows"))
		check(nil, nil)
	})
}

func TestKeyEventCanTellWhichKeysArePressed(t *testing.T) {
	keyTest(t, modeKeyDataThenRawKeyData, func() {
		hk := services.HardwareKeyboardInstance()
		pressed := func(p services.PhysicalKeyboardKey, l services.LogicalKeyboardKey, want bool) {
			expect(hk.IsPhysicalKeyPressed(p) == want, "physical %v pressed != %v", p, want)
			expect(hk.IsLogicalKeyPressed(l) == want, "logical %v pressed != %v", l, want)
		}
		keyDown(services.LogicalKeyboardKeyNumLock, on("windows"))
		pressed(services.PhysicalKeyboardKeyNumLock, services.LogicalKeyboardKeyNumLock, true)
		keyDown(services.LogicalKeyboardKeyNumpad1, on("windows"))
		pressed(services.PhysicalKeyboardKeyNumpad1, services.LogicalKeyboardKeyNumpad1, true)
		keyRepeat(services.LogicalKeyboardKeyNumpad1, on("windows"))
		pressed(services.PhysicalKeyboardKeyNumpad1, services.LogicalKeyboardKeyNumpad1, true)
		keyUp(services.LogicalKeyboardKeyNumLock)
		pressed(services.PhysicalKeyboardKeyNumpad1, services.LogicalKeyboardKeyNumpad1, true)
		keyDown(services.LogicalKeyboardKeyNumLock, on("windows"))
		pressed(services.PhysicalKeyboardKeyNumLock, services.LogicalKeyboardKeyNumLock, true)
		keyUp(services.LogicalKeyboardKeyNumpad1, on("windows"))
		pressed(services.PhysicalKeyboardKeyNumpad1, services.LogicalKeyboardKeyNumpad1, false)
		keyUp(services.LogicalKeyboardKeyNumLock, on("windows"))
		pressed(services.PhysicalKeyboardKeyNumLock, services.LogicalKeyboardKeyNumLock, false)
	})
}

func TestKeyboardManagerSynthesizesModifierKeysInRawKeyDataMode(t *testing.T) {
	keyTest(t, noVariant, func() {
		var events []services.KeyEvent
		services.HardwareKeyboardInstance().AddHandler(func(event services.KeyEvent) bool {
			events = append(events, event)
			return false
		})
		// While ShiftLeft is held (the event of which was skipped), press keyA.
		raw := keyData(services.LogicalKeyboardKeyKeyA, "android")
		setEntry(raw, "metaState", services.RawKeyEventDataAndroidModifierLeftShift|services.RawKeyEventDataAndroidModifierShift)
		ensureInitialized().KeyEventManager().HandleRawKeyMessage(asMap(raw)).Await()
		expect(len(events) == 2, "events: %v", events)
		expect(events[0].PhysicalKey() == services.PhysicalKeyboardKeyShiftLeft, "0 physical")
		expect(events[0].LogicalKey() == services.LogicalKeyboardKeyShiftLeft, "0 logical")
		expect(events[0].Synthesized(), "0 synthesized")
		expect(events[1].PhysicalKey() == services.PhysicalKeyboardKeyKeyA, "1 physical")
		expect(events[1].LogicalKey() == services.LogicalKeyboardKeyKeyA, "1 logical")
		expect(!events[1].Synthesized(), "1 synthesized")
	})
}

func TestDispatchEventsToAllHandlers(t *testing.T) {
	keyTest(t, modesAll, func() {
		var logs []int
		ensureInitialized().KeyEventManager().KeyMessageHandler = listenerHandler(func(services.KeyEvent) { logs = append(logs, 1) }, nil)
		expectLogs := func(want ...int) {
			expect(slicesEqual(logs, want), "logs %v, want %v", logs, want)
			logs = nil
		}
		// Only the Service binding handler.
		expect(!keyDown(services.LogicalKeyboardKeyKeyA), "handled")
		expectLogs(1)
		// Add a handler.
		handler2Result := false
		handler2 := func(services.KeyEvent) bool {
			logs = append(logs, 2)
			return handler2Result
		}
		hk := services.HardwareKeyboardInstance()
		hk.AddHandler(handler2)
		expect(!keyUp(services.LogicalKeyboardKeyKeyA), "handled")
		expectLogs(2, 1)
		handler2Result = true
		expect(keyDown(services.LogicalKeyboardKeyKeyA), "not handled")
		expectLogs(2, 1)
		// Add another handler.
		handler2Result = false
		handler3Result := false
		handler3 := func(services.KeyEvent) bool {
			logs = append(logs, 3)
			return handler3Result
		}
		hk.AddHandler(handler3)
		expect(!keyUp(services.LogicalKeyboardKeyKeyA), "handled")
		expectLogs(2, 3, 1)
		handler2Result = true
		expect(keyDown(services.LogicalKeyboardKeyKeyA), "not handled")
		expectLogs(2, 3, 1)
		handler3Result = true
		expect(keyUp(services.LogicalKeyboardKeyKeyA), "not handled")
		expectLogs(2, 3, 1)
		// Add handler2 again.
		hk.AddHandler(handler2)
		handler3Result = false
		handler2Result = false
		expect(!keyDown(services.LogicalKeyboardKeyKeyA), "handled")
		expectLogs(2, 3, 2, 1)
		handler2Result = true
		expect(keyUp(services.LogicalKeyboardKeyKeyA), "not handled")
		expectLogs(2, 3, 2, 1)
		// Remove handler2 once.
		hk.RemoveHandler(handler2)
		expect(keyDown(services.LogicalKeyboardKeyKeyA), "not handled")
		expectLogs(3, 2, 1)
	})
}

func slicesEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Regression test for flutter#99196.
func TestCorrectlyConvertDownEventsThatAreSynthesizedReleased(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		var events []services.KeyEvent
		ensureInitialized().KeyEventManager().KeyMessageHandler = listenerHandler(func(e services.KeyEvent) { events = append(events, e) }, nil)
		// Dispatch an arbitrary event to bypass the pressedKeys check.
		keyDown(services.LogicalKeyboardKeyKeyA, on("web"))
		data2 := keyData(services.LogicalKeyboardKeyShiftLeft, "web")
		setEntry(data2, "metaState", 0)
		sendRawMessage(data2)
		expect(len(events) == 3, "events %v", events)
		_, down := events[1].(*services.KeyDownEvent)
		expect(down && events[1].LogicalKey() == services.LogicalKeyboardKeyShiftLeft && !events[1].Synthesized(), "event 1 %v", events[1])
		_, up := events[2].(*services.KeyUpEvent)
		expect(up && events[2].LogicalKey() == services.LogicalKeyboardKeyShiftLeft && events[2].Synthesized(), "event 2 %v", events[2])
		expectSet("pressed", ensureInitialized().Keyboard().PhysicalKeysPressed(), services.PhysicalKeyboardKeyKeyA)
	})
}

func TestInstantlyDispatchSynthesizedKeyEventsWhenTheQueueIsEmpty(t *testing.T) {
	keyTest(t, modeKeyDataThenRawKeyData, func() {
		var logs []int
		b := ensureInitialized()
		b.KeyEventManager().KeyMessageHandler = listenerHandler(func(services.KeyEvent) { logs = append(logs, 1) }, nil)
		b.Keyboard().AddHandler(func(services.KeyEvent) bool {
			logs = append(logs, 2)
			return false
		})
		// Dispatch a solitary synthesized event.
		expect(!b.KeyEventManager().HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown,
			Logical: services.LogicalKeyboardKeyKeyA.KeyID, Physical: services.PhysicalKeyboardKeyKeyA.UsbHidUsage,
			Synthesized: true}), "handled")
		expect(slicesEqual(logs, []int{2, 1}), "logs %v", logs)
	})
}

func TestPostponeSynthesizedKeyEventsWhenTheQueueIsNotEmpty(t *testing.T) {
	keyTest(t, modeKeyDataThenRawKeyData, func() {
		var logs []string
		b := ensureInitialized()
		typeName := func(v any) string { return foundation.ObjectRuntimeType(v, "") }
		b.KeyEventManager().KeyMessageHandler = listenerHandler(
			func(e services.KeyEvent) { logs = append(logs, typeName(e)) },
			func(e services.RawKeyEvent) { logs = append(logs, typeName(e)) })
		// On macOS, a CapsLock tap yields a down event and a synthesized up event.
		expect(!b.KeyEventManager().HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown,
			Logical: services.LogicalKeyboardKeyCapsLock.KeyID, Physical: services.PhysicalKeyboardKeyCapsLock.UsbHidUsage}), "handled")
		expect(!b.KeyEventManager().HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeUp,
			Logical: services.LogicalKeyboardKeyCapsLock.KeyID, Physical: services.PhysicalKeyboardKeyCapsLock.UsbHidUsage,
			Synthesized: true}), "handled")
		result := b.KeyEventManager().HandleRawKeyMessage(map[string]any{
			"type": "keydown", "keymap": "macos", "keyCode": 0x00000039, "characters": "",
			"charactersIgnoringModifiers": "", "modifiers": 0x10000,
		}).Await()
		expect(result["handled"] == false && len(result) == 1, "result %v", result)
		expect(slicesEqual(logs, []string{"RawKeyDownEvent", "KeyDownEvent", "KeyUpEvent"}), "logs %v", logs)
	})
}

// The first key data received from the engine might be an empty key data:
// no event, but it makes KeyEventManager infer keyDataThenRawKeyData.
func TestEmptyKeyDataYieldsNoEventButTriggersInference(t *testing.T) {
	keyTest(t, noVariant, func() {
		b := ensureInitialized()
		var events []services.KeyEvent
		var rawEvents []services.RawKeyEvent
		b.Keyboard().AddHandler(func(e services.KeyEvent) bool {
			events = append(events, e)
			return true
		})
		listener := func(e services.RawKeyEvent) { rawEvents = append(rawEvents, e) }
		services.RawKeyboardInstance.AddListener(listener)
		defer services.RawKeyboardInstance.RemoveListener(listener)
		a := "a"
		b.KeyEventManager().HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown, Character: &a})
		message := map[string]any{"type": "keydown", "keymap": "windows", "keyCode": 0x04, "scanCode": 0x04,
			"characterCodePoint": 0, "modifiers": 0}
		b.KeyEventManager().HandleRawKeyMessage(message).Await()
		expect(len(events) == 0, "events %d", len(events))
		expect(len(rawEvents) == 1, "raw events %d", len(rawEvents))
		// Another key data proves the mode (otherwise it asserts).
		quote := `"`
		b.KeyEventManager().HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown, Logical: 0x22, Physical: 0x70034, Character: &quote})
		b.KeyEventManager().HandleRawKeyMessage(message).Await()
		expect(len(events) == 1, "events %d", len(events))
		expect(len(rawEvents) == 2, "raw events %d", len(rawEvents))
	})
}

// runWhileOverridingOnError runs body with FlutterError.onError replaced.
func runWhileOverridingOnError(body func(), onError func(*foundation.FlutterErrorDetails)) {
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = onError
	defer func() { foundation.FlutterErrorOnError = old }()
	body()
}

func groupDiagnosticsByName(details *foundation.FlutterErrorDetails) map[string]foundation.DiagnosticsNode {
	out := map[string]foundation.DiagnosticsNode{}
	for node := range details.InformationCollector() {
		out[node.Name()] = node
	}
	return out
}

func TestExceptionsFromKeyMessageHandlerAreCaughtAndReported(t *testing.T) {
	keyTest(t, noVariant, func() {
		manager := ensureInitialized().KeyEventManager()
		old := manager.KeyMessageHandler
		defer func() { manager.KeyMessageHandler = old }()
		// When keyMessageHandler throws an error...
		manager.KeyMessageHandler = func(*services.KeyMessage) bool { panic(1) }
		var record *foundation.FlutterErrorDetails
		runWhileOverridingOnError(func() { keyDown(services.LogicalKeyboardKeyKeyA) },
			func(d *foundation.FlutterErrorDetails) { record = d })
		// ... the error should be caught.
		expect(record != nil && record.Exception == 1, "record %v", record)
		_, ok := groupDiagnosticsByName(record)["KeyMessage"].(*foundation.DiagnosticsProperty[*services.KeyMessage])
		expect(ok, "KeyMessage info")
		// The exception should not interrupt recording the state.
		manager.KeyMessageHandler = nil
		record = nil
		runWhileOverridingOnError(func() { keyUp(services.LogicalKeyboardKeyKeyA) },
			func(d *foundation.FlutterErrorDetails) { record = d })
		expect(record == nil, "record %v", record)
	})
}

func TestExceptionsFromHardwareKeyboardHandlersAreCaughtAndReported(t *testing.T) {
	keyTest(t, modesAll, func() {
		throwingCallback := func(services.KeyEvent) bool { panic(1) }
		// When the handler throws an error...
		services.HardwareKeyboardInstance().AddHandler(throwingCallback)
		var record *foundation.FlutterErrorDetails
		runWhileOverridingOnError(func() { keyDown(services.LogicalKeyboardKeyKeyA) },
			func(d *foundation.FlutterErrorDetails) { record = d })
		expect(record != nil && record.Exception == 1, "record %v", record)
		_, ok := groupDiagnosticsByName(record)["Event"].(*foundation.DiagnosticsProperty[services.KeyEvent])
		expect(ok, "Event info: %T", groupDiagnosticsByName(record)["Event"])
		services.HardwareKeyboardInstance().RemoveHandler(throwingCallback)
		record = nil
		runWhileOverridingOnError(func() { keyUp(services.LogicalKeyboardKeyKeyA) },
			func(d *foundation.FlutterErrorDetails) { record = d })
		expect(record == nil, "record %v", record)
	})
}

func captureDebugPrint(f func()) []string {
	var messages []string
	old := foundation.DebugPrint
	foundation.DebugPrint = func(message string, _ ...foundation.DebugPrintOptions) { messages = append(messages, message) }
	defer func() { foundation.DebugPrint = old }()
	f()
	return messages
}

func TestDebugPrintKeyboardEventsCausesLoggingOfKeyEvents(t *testing.T) {
	keyTest(t, noVariant, func() {
		messages := captureDebugPrint(func() {
			services.DebugPrintKeyboardEvents = true
			defer func() { services.DebugPrintKeyboardEvents = false }()
			keyDown(services.LogicalKeyboardKeyKeyA)
		})
		all := strings.Join(messages, "\n")
		for _, want := range []string{"KEYBOARD: Key event received: ", "KEYBOARD: Pressed state before processing the event:",
			"KEYBOARD: Pressed state after processing the event:"} {
			expect(strings.Contains(all, want), "missing %q in %s", want, all)
		}
	})
}

func TestIrregularKeyEventsAreProcessed(t *testing.T) {
	keyTest(t, modeKeyDataThenRawKeyData, func() {
		services.DebugPrintKeyboardEvents = true
		defer func() { services.DebugPrintKeyboardEvents = false }()
		expectIsRegular := func(simulate func() bool, isRegular bool) {
			var handled bool
			logs := captureDebugPrint(func() { handled = simulate() })
			// All simulated events should be dispatched to handlers (not
			// handled).
			expect(!handled, "handled")
			hasError := strings.Contains(strings.Join(logs, "\n"), "ERROR")
			expect(hasError != isRegular, "regular %v, logs %v", isRegular, logs)
		}
		type S = map[services.PhysicalKeyboardKey]services.LogicalKeyboardKey
		physA := &services.PhysicalKeyboardKeyKeyA
		// 1. Press keyA.
		expectIsRegular(func() bool { return keyDown(services.LogicalKeyboardKeyKeyA) }, true)
		expectState(S{services.PhysicalKeyboardKeyKeyA: services.LogicalKeyboardKeyKeyA})
		// Press keyA again with a mismatched logical key.
		expectIsRegular(func() bool {
			return keyDown(services.LogicalKeyboardKeyKeyB, flutter_test.SimulateKeyEventOptions{PhysicalKey: physA})
		}, false)
		expectState(S{services.PhysicalKeyboardKeyKeyA: services.LogicalKeyboardKeyKeyB})
		// 2. Release keyA.
		expectIsRegular(func() bool {
			return keyUp(services.LogicalKeyboardKeyKeyB, flutter_test.SimulateKeyEventOptions{PhysicalKey: physA})
		}, true)
		expectState(S{})
		// Release keyA again.
		expectIsRegular(func() bool { return keyUp(services.LogicalKeyboardKeyKeyA) }, false)
		expectState(S{})
		// 3. A repeat event for keyA affects the state.
		expectIsRegular(func() bool { return keyRepeat(services.LogicalKeyboardKeyKeyA) }, false)
		expectState(S{services.PhysicalKeyboardKeyKeyA: services.LogicalKeyboardKeyKeyA})
		// 4. A down event with a mismatched logical key.
		expectIsRegular(func() bool {
			return keyDown(services.LogicalKeyboardKeyKeyB, flutter_test.SimulateKeyEventOptions{PhysicalKey: physA})
		}, false)
		expectState(S{services.PhysicalKeyboardKeyKeyA: services.LogicalKeyboardKeyKeyB})
		// 5. A key up event with a mismatched logical key.
		expectIsRegular(func() bool { return keyUp(services.LogicalKeyboardKeyKeyA) }, false)
		expectState(S{})
	})
}

// Own test: in raw key data mode, an Android event's source gives the
// converted key events' device type.
func TestRawKeyDataDeviceTypes(t *testing.T) {
	keyTest(t, noVariant, func() {
		var got []ui.KeyEventDeviceType
		services.HardwareKeyboardInstance().AddHandler(func(e services.KeyEvent) bool {
			got = append(got, e.DeviceType())
			return false
		})
		manager := ensureInitialized().KeyEventManager()
		for _, source := range []int{0x101, 0x201, 0x401, 0x01000010, 0x02000001, 0x7} {
			for _, typ := range []string{"keydown", "keyup"} {
				manager.HandleRawKeyMessage(map[string]any{"type": typ, "keymap": "android", "keyCode": 29,
					"scanCode": 30, "plainCodePoint": 97, "codePoint": 97, "metaState": 0, "source": source}).Await()
			}
		}
		want := []ui.KeyEventDeviceType{ui.KeyEventDeviceTypeKeyboard, ui.KeyEventDeviceTypeKeyboard,
			ui.KeyEventDeviceTypeDirectionalPad, ui.KeyEventDeviceTypeDirectionalPad,
			ui.KeyEventDeviceTypeGamepad, ui.KeyEventDeviceTypeGamepad,
			ui.KeyEventDeviceTypeJoystick, ui.KeyEventDeviceTypeJoystick,
			ui.KeyEventDeviceTypeHdmi, ui.KeyEventDeviceTypeHdmi,
			ui.KeyEventDeviceTypeKeyboard, ui.KeyEventDeviceTypeKeyboard}
		expect(slicesEqual(got, want), "device types %v", got)
	})
}

var _ = time.Duration(0)
