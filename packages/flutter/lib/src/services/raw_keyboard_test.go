package services_test

// Port of packages/flutter/test/services/raw_keyboard_test.dart (Flutter
// 3.50.1): 107/107 tests.
//
// Adapted (widgets' focus system is not ported):
//   - "Dispatch events to all handlers" (RawKeyboard group): the focused
//     RawKeyboardListener is the RawKeyboard listener it registers (added
//     first).
//   - "Key events are responded to correctly." (Android): the focused Focus
//     handling every event is a keyMessageHandler returning true.
//   - "Win32 VK_PROCESSKEY events are skipped": the focused node's onKey is
//     a keyMessageHandler adding the raw event.
// The three browser-only tests (skip: !isBrowser in Dart: "CapsLock should
// not be release when remapped on Web", "Allows inconsistent modifier for
// Web - Alt graph/Alt right") run here too: their logic doesn't depend on
// kIsWeb.

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type L = services.LogicalKeyboardKey

var platformsNoWeb = []string{"linux", "android", "macos", "fuchsia", "windows", "ios"}

func expectKeysPressed(want ...L) {
	expectSet("keysPressed", services.RawKeyboardInstance.KeysPressed(), want...)
}

func ifL(cond bool, keys ...L) []L {
	if cond {
		return keys
	}
	return nil
}

func cat(groups ...[]L) []L {
	var out []L
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func TestRawKeyboardTheCorrectCharacterIsProduced(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		for _, platform := range platformsNoWeb {
			character := ""
			handleKey := func(event services.RawKeyEvent) {
				c := event.Character()
				expect(c != nil && *c == character, "on %s: character %v, want %q", platform, c, character)
			}
			services.RawKeyboardInstance.AddListener(handleKey)
			character = "a"
			keyDown(services.LogicalKeyboardKeyKeyA, on(platform))
			character = "`"
			keyDown(services.LogicalKeyboardKeyBackquote, on(platform))
			services.RawKeyboardInstance.RemoveListener(handleKey)
		}
	})
}

func TestRawKeyboardNoCharacterIsProducedForNonPrintables(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		for _, platform := range []string{"linux", "android", "macos", "fuchsia", "windows", "web", "ios"} {
			handleKey := func(event services.RawKeyEvent) {
				expect(event.Character() == nil, "on %s: character %v", platform, event.Character())
			}
			services.RawKeyboardInstance.AddListener(handleKey)
			keyDown(services.LogicalKeyboardKeyShiftLeft, on(platform))
			services.RawKeyboardInstance.RemoveListener(handleKey)
		}
	})
}

func TestRawKeyboardKeysPressedIsMaintained(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		for _, platform := range platformsNoWeb {
			linux := platform == "linux"
			services.RawKeyboardInstance.ClearKeysPressed()
			expectKeysPressed()
			keyDown(services.LogicalKeyboardKeyShiftLeft, on(platform))
			// Linux doesn't have a concept of left/right keys, so they're all
			// shown as down when either is pressed.
			shift := cat([]L{services.LogicalKeyboardKeyShiftLeft}, ifL(linux, services.LogicalKeyboardKeyShiftRight))
			control := cat([]L{services.LogicalKeyboardKeyControlLeft}, ifL(linux, services.LogicalKeyboardKeyControlRight))
			expectKeysPressed(shift...)
			keyDown(services.LogicalKeyboardKeyControlLeft, on(platform))
			expectKeysPressed(cat(shift, control)...)
			keyDown(services.LogicalKeyboardKeyKeyA, on(platform))
			expectKeysPressed(cat(shift, control, []L{services.LogicalKeyboardKeyKeyA})...)
			keyUp(services.LogicalKeyboardKeyKeyA, on(platform))
			expectKeysPressed(cat(shift, control)...)
			keyUp(services.LogicalKeyboardKeyControlLeft, on(platform))
			expectKeysPressed(shift...)
			keyUp(services.LogicalKeyboardKeyShiftLeft, on(platform))
			expectKeysPressed()
			// The Fn key isn't mapped on linux or Windows.
			if platform != "linux" && platform != "windows" && platform != "ios" {
				fn := ifL(platform != "macos", services.LogicalKeyboardKeyFn)
				keyDown(services.LogicalKeyboardKeyFn, on(platform))
				expectKeysPressed(fn...)
				keyDown(services.LogicalKeyboardKeyF12, on(platform))
				expectKeysPressed(cat(fn, []L{services.LogicalKeyboardKeyF12})...)
				keyUp(services.LogicalKeyboardKeyFn, on(platform))
				expectKeysPressed(services.LogicalKeyboardKeyF12)
				keyUp(services.LogicalKeyboardKeyF12, on(platform))
				expectKeysPressed()
			}
		}
	})
}

func TestRawKeyboardKeysPressedIsCorrectWhenModifierIsReleasedBeforeKey(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		for _, platform := range platformsNoWeb {
			linux := platform == "linux"
			services.RawKeyboardInstance.ClearKeysPressed()
			expectKeysPressed()
			shiftP, aP := services.PhysicalKeyboardKeyShiftLeft, services.PhysicalKeyboardKeyKeyA
			shiftOpts := flutter_test.SimulateKeyEventOptions{Platform: platform, PhysicalKey: &shiftP}
			aOpts := flutter_test.SimulateKeyEventOptions{Platform: platform, PhysicalKey: &aP}
			shift := cat([]L{services.LogicalKeyboardKeyShiftLeft}, ifL(linux, services.LogicalKeyboardKeyShiftRight))
			keyDown(services.LogicalKeyboardKeyShiftLeft, shiftOpts)
			expectKeysPressed(shift...)
			keyDown(services.LogicalKeyboardKeyKeyA, aOpts)
			expectKeysPressed(cat(shift, []L{services.LogicalKeyboardKeyKeyA})...)
			keyUp(services.LogicalKeyboardKeyShiftLeft, shiftOpts)
			expectKeysPressed(services.LogicalKeyboardKeyKeyA)
			keyUp(services.LogicalKeyboardKeyKeyA, aOpts)
			expectKeysPressed()
		}
	})
}

// sendModified sends getKeyData(keyA, platform) with field |= bits.
func sendModified(key L, platform, field string, bits int) {
	data := keyData(key, platform)
	v, _ := getEntry(data, field).(int)
	setEntry(data, field, v|bits)
	sendRawMessage(data)
}

func TestRawKeyboardKeysPressedModifiersAreSynchronized(t *testing.T) {
	for _, c := range []struct {
		platform, field string
		bits            int
		want            []L
	}{
		{"macos", "modifiers", services.RawKeyEventDataMacOsModifierLeftShift | services.RawKeyEventDataMacOsModifierShift, nil},
		// The Dart test uses the macOS masks for iOS too (they are the same).
		{"ios", "modifiers", services.RawKeyEventDataMacOsModifierLeftShift | services.RawKeyEventDataMacOsModifierShift, nil},
		{"windows", "modifiers", services.RawKeyEventDataWindowsModifierLeftShift | services.RawKeyEventDataWindowsModifierShift, nil},
		{"android", "metaState", services.RawKeyEventDataAndroidModifierLeftShift | services.RawKeyEventDataAndroidModifierShift, nil},
		{"fuchsia", "modifiers", services.RawKeyEventDataFuchsiaModifierLeftShift, nil},
		{"linux", "modifiers", services.GLFWKeyHelperModifierShift, []L{services.LogicalKeyboardKeyShiftRight}},
	} {
		t.Run(c.platform, func(t *testing.T) {
			keyTest(t, modeRawKeyData, func() {
				expectKeysPressed()
				sendModified(services.LogicalKeyboardKeyKeyA, c.platform, c.field, c.bits)
				expectKeysPressed(cat([]L{services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyKeyA}, c.want)...)
			})
		})
	}
}

func simulateGTKKeyEvent(keyDown bool, scancode, keycode, modifiers int) {
	typ := "keyup"
	if keyDown {
		typ = "keydown"
	}
	data := &[]dart.MapEntry[string, any]{{Key: "type", Value: typ}, {Key: "keymap", Value: "linux"},
		{Key: "toolkit", Value: "gtk"}, {Key: "scanCode", Value: scancode}, {Key: "keyCode", Value: keycode},
		{Key: "modifiers", Value: modifiers}}
	// An empty key data disables HardwareKeyboard's sanity check (only the
	// raw keyboard is tested).
	ensureInitialized().KeyEventManager().HandleKeyData(ui.KeyData{Type: ui.KeyEventTypeDown})
	sendRawMessage(data)
}

// Regression test for flutter#93278.
func TestRawKeyboardKeysPressedModifiersAreSynchronizedOnLinuxGTKDownEvents(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		expectKeysPressed()
		simulateGTKKeyEvent(true, 0x6c /*AltRight*/, 0xffea /*AltRight*/, 0x2000000)
		simulateGTKKeyEvent(true, 0x32 /*ShiftLeft*/, 0xfe08 /*NextGroup*/, 0x2000008 /*MOD3*/)
		simulateGTKKeyEvent(false, 0x6c /*AltRight*/, 0xfe03 /*AltRight*/, 0x2002008 /*MOD3|Reserve14*/)
		simulateGTKKeyEvent(true, 0x6c /*AltRight*/, 0xfe03 /*AltRight*/, 0x2002000 /*Reserve14*/)
		expectKeysPressed(services.LogicalKeyboardKeyAltRight)
	})
}

// Regression test for flutter#114591.
func TestRawKeyboardCapsLockShouldNotBeReleasedWhenRemappedOnLinux(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		expectKeysPressed()
		simulateGTKKeyEvent(true, 0x42 /*CapsLock*/, 0xff08 /*Backspace*/, 0x2000000)
		expectKeysPressed(services.LogicalKeyboardKeyBackspace)
	})
}

func webMessage(entries ...dart.MapEntry[string, any]) *[]dart.MapEntry[string, any] {
	d := append([]dart.MapEntry[string, any]{{Key: "type", Value: "keydown"}, {Key: "keymap", Value: "web"}}, entries...)
	return &d
}

func e(k string, v any) dart.MapEntry[string, any] {
	return dart.MapEntry[string, any]{Key: k, Value: v}
}

// Regression test for flutter#114591 (browser-only in Dart).
func TestRawKeyboardCapsLockShouldNotBeReleasedWhenRemappedOnWeb(t *testing.T) {
	keyTest(t, modeRawKeyData, func() {
		var events []services.RawKeyEvent
		listener := func(e services.RawKeyEvent) { events = append(events, e) }
		services.RawKeyboardInstance.AddListener(listener)
		defer services.RawKeyboardInstance.RemoveListener(listener)
		sendRawMessage(webMessage(e("code", "CapsLock"), e("key", "Backspace"), e("location", 0), e("metaState", 0), e("keyCode", 8)))
		expectKeysPressed(services.LogicalKeyboardKeyBackspace)
	})
}

func TestRawKeyboardKeysPressedModifiersAreSynchronizedOnWeb(t *testing.T) {
	keyTest(t, noVariant, func() {
		expectKeysPressed()
		sendModified(services.LogicalKeyboardKeyKeyA, "web", "metaState", services.RawKeyEventDataWebModifierShift)
		expectKeysPressed(services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyKeyA)
		isUp := false
		data2 := keyData(services.LogicalKeyboardKeyKeyA, "web", flutter_test.GetKeyDataOptions{IsDown: &isUp})
		setEntry(data2, "metaState", 0)
		sendRawMessage(data2)
		expectKeysPressed()
		// Press right modifier key (the Dart test sets the modifier on the
		// previous message's data, so this one has none).
		sendRawMessage(keyData(services.LogicalKeyboardKeyShiftRight, "web"))
		expectKeysPressed(services.LogicalKeyboardKeyShiftRight)
		data4 := keyData(services.LogicalKeyboardKeyShiftRight, "web", flutter_test.GetKeyDataOptions{IsDown: &isUp})
		setEntry(data4, "metaState", 0)
		sendRawMessage(data4)
		expectKeysPressed()
	})
}

var allSidedModifiers = []L{services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight,
	services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight, services.LogicalKeyboardKeyControlLeft,
	services.LogicalKeyboardKeyControlRight, services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight,
	services.LogicalKeyboardKeyKeyA}

func TestRawKeyboardSidedModifiersWithoutASide(t *testing.T) {
	for _, c := range []struct {
		name, platform, field string
		bits                  int
		modes                 []services.KeyDataTransitMode
		want                  []L
	}{
		{"all sides on Android", "android", "metaState", services.RawKeyEventDataAndroidModifierShift | services.RawKeyEventDataAndroidModifierAlt |
			services.RawKeyEventDataAndroidModifierControl | services.RawKeyEventDataAndroidModifierMeta, modeRawKeyData, allSidedModifiers},
		{"all sides on macOS", "macos", "modifiers", services.RawKeyEventDataMacOsModifierShift | services.RawKeyEventDataMacOsModifierOption |
			services.RawKeyEventDataMacOsModifierCommand | services.RawKeyEventDataMacOsModifierControl, noVariant, allSidedModifiers},
		{"all sides on iOS", "ios", "modifiers", services.RawKeyEventDataIosModifierShift | services.RawKeyEventDataIosModifierOption |
			services.RawKeyEventDataIosModifierCommand | services.RawKeyEventDataIosModifierControl, noVariant, allSidedModifiers},
		{"all sides on Windows", "windows", "modifiers", services.RawKeyEventDataWindowsModifierShift | services.RawKeyEventDataWindowsModifierAlt |
			services.RawKeyEventDataWindowsModifierControl, noVariant, allSidedModifiers[:6:6]},
		{"all sides on Linux GLFW", "linux", "modifiers", services.GLFWKeyHelperModifierShift | services.GLFWKeyHelperModifierAlt |
			services.GLFWKeyHelperModifierControl | services.GLFWKeyHelperModifierMeta, noVariant, allSidedModifiers},
		{"left sides on web", "web", "metaState", services.RawKeyEventDataWebModifierShift | services.RawKeyEventDataWebModifierAlt |
			services.RawKeyEventDataWebModifierControl | services.RawKeyEventDataWebModifierMeta, noVariant,
			[]L{services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyControlLeft,
				services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyKeyA}},
	} {
		t.Run(c.name, func(t *testing.T) {
			keyTest(t, c.modes, func() {
				expectKeysPressed()
				sendModified(services.LogicalKeyboardKeyKeyA, c.platform, c.field, c.bits)
				want := c.want
				if c.platform == "windows" {
					want = append(append([]L{}, want...), services.LogicalKeyboardKeyKeyA)
				}
				expectKeysPressed(want...)
			})
		})
	}
}

func TestRawKeyboardRepeatEvents(t *testing.T) {
	keyTest(t, noVariant, func() {
		expectKeysPressed()
		var receivedEvent services.RawKeyEvent
		services.RawKeyboardInstance.SetKeyEventHandler(func(event services.RawKeyEvent) bool {
			receivedEvent = event
			return true
		})
		downData := keyData(services.LogicalKeyboardKeyKeyA, "windows")
		sendRawMessage(downData)
		expect(!receivedEvent.Repeat(), "first down repeats")
		sendRawMessage(downData)
		expect(receivedEvent.Repeat(), "second down doesn't repeat")
		isUp := false
		sendRawMessage(keyData(services.LogicalKeyboardKeyKeyA, "windows", flutter_test.GetKeyDataOptions{IsDown: &isUp}))
		expect(!receivedEvent.Repeat(), "up repeats")
		services.RawKeyboardInstance.SetKeyEventHandler(nil)
	})
}

func TestRawKeyboardAssertsIfNoKeysAreInKeysPressedAfterAKeyDownEvent(t *testing.T) {
	keyTest(t, noVariant, func() {
		message := &[]dart.MapEntry[string, any]{e("type", "keydown"), e("keymap", "windows"), e("keyCode", 16),
			e("characterCodePoint", 0), e("scanCode", 42), e("modifiers", 0x0)}
		var caught any
		func() {
			defer func() { caught = recover() }()
			sendRawMessage(message)
		}()
		err, ok := caught.(error)
		expect(ok && strings.Contains(err.Error(), "Attempted to send a key down event when no keys are in keysPressed"),
			"caught %v", caught)
	})
}

func expectAllowsInconsistent(message *[]dart.MapEntry[string, any], physical services.PhysicalKeyboardKey, logical L) {
	var events []services.RawKeyEvent
	listener := func(e services.RawKeyEvent) { events = append(events, e) }
	services.RawKeyboardInstance.AddListener(listener)
	defer services.RawKeyboardInstance.RemoveListener(listener)
	sendRawMessage(message)
	expect(len(events) == 1, "events %v", events)
	expect(events[0].Data().PhysicalKey() == physical, "physical %v", events[0].Data().PhysicalKey())
	expect(events[0].Data().LogicalKey() == logical, "logical %v", events[0].Data().LogicalKey())
	expect(slicesContains(services.RawKeyboardInstance.KeysPressed(), logical), "pressed %v", services.RawKeyboardInstance.KeysPressed())
}

func TestRawKeyboardAllowsInconsistentModifier(t *testing.T) {
	t.Run("iOS", func(t *testing.T) {
		keyTest(t, noVariant, func() {
			expectAllowsInconsistent(&[]dart.MapEntry[string, any]{e("type", "keydown"), e("keymap", "ios"), e("keyCode", 0x00000039),
				e("characters", ""), e("charactersIgnoringModifiers", ""), e("modifiers", 0)},
				services.PhysicalKeyboardKeyCapsLock, services.LogicalKeyboardKeyCapsLock)
		})
	})
	t.Run("Android", func(t *testing.T) {
		keyTest(t, noVariant, func() {
			expectAllowsInconsistent(&[]dart.MapEntry[string, any]{e("type", "keydown"), e("keymap", "android"), e("keyCode", 115),
				e("plainCodePoint", 0), e("codePoint", 0), e("scanCode", 58), e("metaState", 0), e("source", 0x101), e("deviceId", 1)},
				services.PhysicalKeyboardKeyCapsLock, services.LogicalKeyboardKeyCapsLock)
		})
	})
	t.Run("Web - Alt graph", func(t *testing.T) {
		keyTest(t, noVariant, func() {
			expectAllowsInconsistent(webMessage(e("code", "AltRight"), e("key", "AltGraph"), e("location", 2), e("metaState", 0), e("keyCode", 225)),
				services.PhysicalKeyboardKeyAltRight, services.LogicalKeyboardKeyAltGraph)
		})
	})
	t.Run("Web - Alt right", func(t *testing.T) {
		keyTest(t, noVariant, func() {
			expectAllowsInconsistent(webMessage(e("code", "AltRight"), e("key", "Alt"), e("location", 2), e("metaState", 0), e("keyCode", 225)),
				services.PhysicalKeyboardKeyAltRight, services.LogicalKeyboardKeyAltRight)
		})
	})
}

func TestRawKeyboardDispatchEventsToAllHandlers(t *testing.T) {
	keyTest(t, modesAll, func() {
		var logs []int
		// The focused RawKeyboardListener's own RawKeyboard listener.
		focused := func(services.RawKeyEvent) { logs = append(logs, 1) }
		services.RawKeyboardInstance.AddListener(focused)
		defer services.RawKeyboardInstance.RemoveListener(focused)
		check := func(handled bool, want ...int) {
			expect(!handled, "handled")
			expect(slicesEqual(logs, want), "logs %v, want %v", logs, want)
			logs = nil
		}
		check(keyDown(services.LogicalKeyboardKeyKeyA), 1)
		handler2 := func(services.RawKeyEvent) { logs = append(logs, 2) }
		services.RawKeyboardInstance.AddListener(handler2)
		check(keyUp(services.LogicalKeyboardKeyKeyA), 1, 2)
		handler3 := func(services.RawKeyEvent) { logs = append(logs, 3) }
		services.RawKeyboardInstance.AddListener(handler3)
		check(keyDown(services.LogicalKeyboardKeyKeyA), 1, 2, 3)
		services.RawKeyboardInstance.AddListener(handler2)
		check(keyUp(services.LogicalKeyboardKeyKeyA), 1, 2, 3, 2)
		services.RawKeyboardInstance.RemoveListener(handler2)
		check(keyDown(services.LogicalKeyboardKeyKeyA), 1, 3, 2)
		services.RawKeyboardInstance.RemoveListener(handler2)
		services.RawKeyboardInstance.RemoveListener(handler3)
	})
}

func TestRawKeyboardExceptionsFromListenersAreCaughtAndReported(t *testing.T) {
	keyTest(t, noVariant, func() {
		throwingListener := func(services.RawKeyEvent) { panic(1) }
		services.RawKeyboardInstance.AddListener(throwingListener)
		var record *foundation.FlutterErrorDetails
		runWhileOverridingOnError(func() { keyDown(services.LogicalKeyboardKeyKeyA) },
			func(d *foundation.FlutterErrorDetails) { record = d })
		expect(record != nil && record.Exception == 1, "record %v", record)
		_, ok := groupDiagnosticsByName(record)["Event"].(*foundation.DiagnosticsProperty[services.RawKeyEvent])
		expect(ok, "Event info: %T", groupDiagnosticsByName(record)["Event"])
		services.RawKeyboardInstance.RemoveListener(throwingListener)
		record = nil
		runWhileOverridingOnError(func() { keyUp(services.LogicalKeyboardKeyKeyA) },
			func(d *foundation.FlutterErrorDetails) { record = d })
		expect(record == nil, "record %v", record)
	})
}

// ---------------------------------------------------------------------------
// Platform data groups.

type modifierCheck struct {
	modifier int
	key      services.ModifierKey
	side     services.KeyboardSide
}

func fromMessage(m map[string]any) services.RawKeyEventData {
	return services.RawKeyEventFromMessage(m).Data()
}

// checkModifiersIndividually is the groups' "modifier keys are recognized
// individually" loop.
func checkModifiersIndividually(t *testing.T, tests []modifierCheck, checkSide bool, data func(modifier int) services.RawKeyEventData) {
	t.Helper()
	for _, mt := range tests {
		d := data(mt.modifier)
		for _, key := range services.ModifierKeyValues {
			pressed := d.IsModifierPressed(key, services.IsModifierPressedOptions{Side: mt.side})
			if mt.key == key {
				if !pressed {
					t.Errorf("%v should be pressed with metaState %#x, but isn't.", key, mt.modifier)
				}
				if checkSide {
					if s := d.GetModifierSide(key); s == nil || *s != mt.side {
						t.Errorf("%v side with %#x: %v, want %v", key, mt.modifier, s, mt.side)
					}
				}
			} else if pressed {
				t.Errorf("%v should not be pressed with metaState %#x.", key, mt.modifier)
			}
		}
	}
}

// checkModifiersCombined is the groups' "modifier keys are recognized when
// combined" loop: each modifier with an additional one.
func checkModifiersCombined(t *testing.T, tests []modifierCheck, extra int, extraKey services.ModifierKey, checkSide bool,
	data func(modifier int) services.RawKeyEventData) {
	t.Helper()
	for _, mt := range tests {
		if mt.modifier == extra {
			continue
		}
		d := data(mt.modifier | extra)
		for _, key := range services.ModifierKeyValues {
			pressed := d.IsModifierPressed(key, services.IsModifierPressedOptions{Side: mt.side})
			if mt.key == key || key == extraKey {
				if !pressed {
					t.Errorf("%v should be pressed with metaState %#x and additional key %#x, but isn't.", key, mt.modifier, extra)
				}
				if checkSide {
					want := mt.side
					if key == extraKey {
						want = services.KeyboardSideAll
					}
					if s := d.GetModifierSide(key); s == nil || *s != want {
						t.Errorf("%v side with %#x|%#x: %v, want %v", key, mt.modifier, extra, s, want)
					}
				}
			} else if pressed {
				t.Errorf("%v should not be pressed with metaState %#x and additional key %#x.", key, mt.modifier, extra)
			}
		}
	}
}

func expectKeys(t *testing.T, d services.RawKeyEventData, physical services.PhysicalKeyboardKey, logical L, label *string) {
	t.Helper()
	if d.PhysicalKey() != physical {
		t.Errorf("physicalKey %v, want %v", d.PhysicalKey(), physical)
	}
	if d.LogicalKey() != logical {
		t.Errorf("logicalKey %v, want %v", d.LogicalKey(), logical)
	}
	if label != nil && d.KeyLabel() != *label {
		t.Errorf("keyLabel %q, want %q", d.KeyLabel(), *label)
	}
}

func str(s string) *string { return &s }

func expectToString(t *testing.T, d services.RawKeyEventData, want string) {
	t.Helper()
	if got := stripHash(d.String()); got != stripHash(want) {
		t.Errorf("toString %s, want %s", got, want)
	}
}

// stripHash is equalsIgnoringHashCodes.
func stripHash(s string) string {
	i := strings.Index(s, "#")
	if i < 0 || i+6 > len(s) {
		return s
	}
	return s[:i] + "#00000" + s[i+6:]
}

type equaler interface {
	Equal(services.RawKeyEventData) bool
}

func expectEqual(t *testing.T, a, b services.RawKeyEventData, want bool) {
	t.Helper()
	if a.(equaler).Equal(b) != want {
		t.Errorf("%v == %v: want %v", a, b, want)
	}
}

func android(m map[string]any) map[string]any {
	out := map[string]any{"type": "keydown", "keymap": "android"}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func TestRawKeyEventDataAndroid(t *testing.T) {
	A := services.RawKeyEventDataAndroidModifierAlt
	tests := []modifierCheck{
		{A | services.RawKeyEventDataAndroidModifierLeftAlt, services.ModifierKeyAltModifier, services.KeyboardSideLeft},
		{A | services.RawKeyEventDataAndroidModifierRightAlt, services.ModifierKeyAltModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataAndroidModifierShift | services.RawKeyEventDataAndroidModifierLeftShift, services.ModifierKeyShiftModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataAndroidModifierShift | services.RawKeyEventDataAndroidModifierRightShift, services.ModifierKeyShiftModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataAndroidModifierSym, services.ModifierKeySymbolModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataAndroidModifierFunction, services.ModifierKeyFunctionModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataAndroidModifierControl | services.RawKeyEventDataAndroidModifierLeftControl, services.ModifierKeyControlModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataAndroidModifierControl | services.RawKeyEventDataAndroidModifierRightControl, services.ModifierKeyControlModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataAndroidModifierMeta | services.RawKeyEventDataAndroidModifierLeftMeta, services.ModifierKeyMetaModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataAndroidModifierMeta | services.RawKeyEventDataAndroidModifierRightMeta, services.ModifierKeyMetaModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataAndroidModifierCapsLock, services.ModifierKeyCapsLockModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataAndroidModifierNumLock, services.ModifierKeyNumLockModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataAndroidModifierScrollLock, services.ModifierKeyScrollLockModifier, services.KeyboardSideAll},
	}
	data := func(modifier int) services.RawKeyEventData {
		return fromMessage(android(map[string]any{"keyCode": 0x04, "plainCodePoint": 0x64, "codePoint": 0x44, "scanCode": 0x20,
			"metaState": modifier, "source": 0x101, "deviceId": 1}))
	}
	t.Run("modifier keys are recognized individually", func(t *testing.T) { checkModifiersIndividually(t, tests, true, data) })
	t.Run("modifier keys are recognized when combined", func(t *testing.T) {
		checkModifiersCombined(t, tests, services.RawKeyEventDataAndroidModifierFunction, services.ModifierKeyFunctionModifier, true, data)
	})
	t.Run("Printable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, fromMessage(android(map[string]any{"keyCode": 29, "plainCodePoint": int('a'), "codePoint": int('A'), "character": "A",
			"scanCode": 30, "metaState": 0, "source": 0x101, "deviceId": 1})), services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, str("a"))
	})
	t.Run("Control keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, fromMessage(android(map[string]any{"keyCode": 111, "codePoint": 0, "scanCode": 1, "metaState": 0, "source": 0x101, "deviceId": 1})),
			services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, str(""))
	})
	t.Run("Modifier keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, fromMessage(android(map[string]any{"keyCode": 59, "plainCodePoint": 0, "codePoint": 0, "scanCode": 42,
			"metaState": services.RawKeyEventDataAndroidModifierLeftShift, "source": 0x101, "deviceId": 1})),
			services.PhysicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftLeft, str(""))
	})
	t.Run("DPAD keys from a joystick give physical key mappings", func(t *testing.T) {
		expectKeys(t, fromMessage(android(map[string]any{"keyCode": 20, "plainCodePoint": 0, "codePoint": 0, "scanCode": 0, "metaState": 0,
			"source": 0x1000010, "deviceId": 1})), services.PhysicalKeyboardKeyArrowDown, services.LogicalKeyboardKeyArrowDown, str(""))
	})
	t.Run("Arrow keys from a keyboard give correct physical key mappings", func(t *testing.T) {
		expectKeys(t, fromMessage(android(map[string]any{"keyCode": 20, "plainCodePoint": 0, "codePoint": 0, "scanCode": 108, "metaState": 0,
			"source": 0x101})), services.PhysicalKeyboardKeyArrowDown, services.LogicalKeyboardKeyArrowDown, str(""))
	})
	t.Run("DPAD center from a game pad gives physical key mappings", func(t *testing.T) {
		expectKeys(t, fromMessage(android(map[string]any{"keyCode": 23, "plainCodePoint": 0, "codePoint": 0, "scanCode": 317, "metaState": 0,
			"source": 0x501, "deviceId": 1})), services.PhysicalKeyboardKeyGameButtonThumbLeft, services.LogicalKeyboardKeySelect, str(""))
	})
	t.Run("Device id is read from message", func(t *testing.T) {
		d := fromMessage(android(map[string]any{"keyCode": 23, "plainCodePoint": 0, "codePoint": 0, "scanCode": 317, "metaState": 0,
			"source": 0x501, "deviceId": 10})).(*services.RawKeyEventDataAndroid)
		if d.DeviceId != 10 {
			t.Error(d.DeviceId)
		}
	})
	t.Run("Repeat count is passed correctly", func(t *testing.T) {
		d := fromMessage(android(map[string]any{"keyCode": 29, "plainCodePoint": int('a'), "codePoint": int('A'), "character": "A",
			"scanCode": 30, "metaState": 0, "source": 0x101, "repeatCount": 42})).(*services.RawKeyEventDataAndroid)
		if d.RepeatCount != 42 {
			t.Error(d.RepeatCount)
		}
	})
	t.Run("Key events are responded to correctly.", func(t *testing.T) {
		keyTest(t, noVariant, func() {
			expectKeysPressed()
			data := keyData(services.LogicalKeyboardKeyKeyA, "android")
			var message map[string]any
			send := func() {
				channel := services.SystemChannels.KeyEvent
				messenger().HandlePlatformMessage(channel.Name, channel.Codec.EncodeMessage(*data), func(reply []byte) {
					message, _ = channel.Codec.DecodeMessage(reply).(map[string]any)
				}).Await()
			}
			send()
			expect(message["handled"] == false && len(message) == 1, "message %v", message)
			message = nil
			// A focused Focus whose onKey handles all events.
			ensureInitialized().KeyEventManager().KeyMessageHandler = func(*services.KeyMessage) bool { return true }
			send()
			expect(message["handled"] == true && len(message) == 1, "message %v", message)
		})
	})
	m := android(map[string]any{"keyCode": 29, "plainCodePoint": 97, "codePoint": 65, "character": "A", "scanCode": 30,
		"metaState": 0, "source": 0x101, "repeatCount": 42})
	t.Run("data.toString", func(t *testing.T) {
		expectToString(t, fromMessage(m), "RawKeyEventDataAndroid#00000(flags: 0, codePoint: 65, plainCodePoint: 97, keyCode: 29, scanCode: 30, metaState: 0)")
	})
	t.Run("data.equality", func(t *testing.T) {
		expectEqual(t, fromMessage(m), services.NewRawKeyEventDataAndroid(services.RawKeyEventDataAndroidOptions{CodePoint: 65,
			PlainCodePoint: 97, KeyCode: 29, ScanCode: 30}), true)
		expectEqual(t, fromMessage(m), services.NewRawKeyEventDataAndroid(), false)
	})
}

func TestRawKeyEventDataFuchsia(t *testing.T) {
	tests := []modifierCheck{
		{services.RawKeyEventDataFuchsiaModifierAlt, services.ModifierKeyAltModifier, services.KeyboardSideAny},
		{services.RawKeyEventDataFuchsiaModifierLeftAlt, services.ModifierKeyAltModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataFuchsiaModifierRightAlt, services.ModifierKeyAltModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataFuchsiaModifierShift, services.ModifierKeyShiftModifier, services.KeyboardSideAny},
		{services.RawKeyEventDataFuchsiaModifierLeftShift, services.ModifierKeyShiftModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataFuchsiaModifierRightShift, services.ModifierKeyShiftModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataFuchsiaModifierControl, services.ModifierKeyControlModifier, services.KeyboardSideAny},
		{services.RawKeyEventDataFuchsiaModifierLeftControl, services.ModifierKeyControlModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataFuchsiaModifierRightControl, services.ModifierKeyControlModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataFuchsiaModifierMeta, services.ModifierKeyMetaModifier, services.KeyboardSideAny},
		{services.RawKeyEventDataFuchsiaModifierLeftMeta, services.ModifierKeyMetaModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataFuchsiaModifierRightMeta, services.ModifierKeyMetaModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataFuchsiaModifierCapsLock, services.ModifierKeyCapsLockModifier, services.KeyboardSideAny},
	}
	fuchsia := func(m map[string]any) services.RawKeyEventData {
		m["type"], m["keymap"] = "keydown", "fuchsia"
		return fromMessage(m)
	}
	data := func(modifier int) services.RawKeyEventData {
		return fuchsia(map[string]any{"hidUsage": 0x04, "codePoint": 0x64, "modifiers": modifier})
	}
	t.Run("modifier keys are recognized individually", func(t *testing.T) { checkModifiersIndividually(t, tests, false, data) })
	t.Run("modifier keys are recognized when combined", func(t *testing.T) {
		checkModifiersCombined(t, tests, services.RawKeyEventDataFuchsiaModifierCapsLock, services.ModifierKeyCapsLockModifier, false, data)
	})
	t.Run("Printable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, fuchsia(map[string]any{"hidUsage": 0x00070004, "codePoint": int('a'), "character": "a"}),
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, str("a"))
	})
	t.Run("Control keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, fuchsia(map[string]any{"hidUsage": 0x00070029}), services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, str(""))
	})
	t.Run("Modifier keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, fuchsia(map[string]any{"hidUsage": 0x000700e1}), services.PhysicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftLeft, str(""))
	})
	m := func() services.RawKeyEventData {
		return fuchsia(map[string]any{"hidUsage": 0x00070004, "codePoint": 97, "character": "a", "modifiers": 0x10})
	}
	t.Run("data.toString", func(t *testing.T) {
		expectToString(t, m(), "RawKeyEventDataFuchsia#00000(hidUsage: 458756, codePoint: 97, modifiers: 16)")
	})
	t.Run("data.equality", func(t *testing.T) {
		expectEqual(t, m(), services.NewRawKeyEventDataFuchsia(services.RawKeyEventDataFuchsiaOptions{HidUsage: 0x00070004, CodePoint: 97, Modifiers: 0x10}), true)
		expectEqual(t, m(), services.NewRawKeyEventDataFuchsia(), false)
	})
}

func appleModifierTests(option, left, right, shift, lshift, rshift, control, lcontrol, rcontrol, command, lcommand, rcommand, caps int) []modifierCheck {
	return []modifierCheck{
		{option | left, services.ModifierKeyAltModifier, services.KeyboardSideLeft},
		{option | right, services.ModifierKeyAltModifier, services.KeyboardSideRight},
		{shift | lshift, services.ModifierKeyShiftModifier, services.KeyboardSideLeft},
		{shift | rshift, services.ModifierKeyShiftModifier, services.KeyboardSideRight},
		{control | lcontrol, services.ModifierKeyControlModifier, services.KeyboardSideLeft},
		{control | rcontrol, services.ModifierKeyControlModifier, services.KeyboardSideRight},
		{command | lcommand, services.ModifierKeyMetaModifier, services.KeyboardSideLeft},
		{command | rcommand, services.ModifierKeyMetaModifier, services.KeyboardSideRight},
		{option, services.ModifierKeyAltModifier, services.KeyboardSideAll},
		{shift, services.ModifierKeyShiftModifier, services.KeyboardSideAll},
		{control, services.ModifierKeyControlModifier, services.KeyboardSideAll},
		{command, services.ModifierKeyMetaModifier, services.KeyboardSideAll},
		{caps, services.ModifierKeyCapsLockModifier, services.KeyboardSideAll},
	}
}

func appleMessage(keymap string, m map[string]any) services.RawKeyEventData {
	m["type"], m["keymap"] = "keydown", keymap
	return fromMessage(m)
}

func TestRawKeyEventDataMacOs(t *testing.T) {
	tests := appleModifierTests(services.RawKeyEventDataMacOsModifierOption, services.RawKeyEventDataMacOsModifierLeftOption,
		services.RawKeyEventDataMacOsModifierRightOption, services.RawKeyEventDataMacOsModifierShift, services.RawKeyEventDataMacOsModifierLeftShift,
		services.RawKeyEventDataMacOsModifierRightShift, services.RawKeyEventDataMacOsModifierControl, services.RawKeyEventDataMacOsModifierLeftControl,
		services.RawKeyEventDataMacOsModifierRightControl, services.RawKeyEventDataMacOsModifierCommand, services.RawKeyEventDataMacOsModifierLeftCommand,
		services.RawKeyEventDataMacOsModifierRightCommand, services.RawKeyEventDataMacOsModifierCapsLock)
	mac := func(m map[string]any) services.RawKeyEventData { return appleMessage("macos", m) }
	data := func(modifier int) services.RawKeyEventData {
		return mac(map[string]any{"keyCode": 0x04, "characters": "a", "charactersIgnoringModifiers": "a", "modifiers": modifier})
	}
	t.Run("modifier keys are recognized individually", func(t *testing.T) { checkModifiersIndividually(t, tests, true, data) })
	t.Run("modifier keys are recognized when combined", func(t *testing.T) {
		checkModifiersCombined(t, tests, services.RawKeyEventDataMacOsModifierCapsLock, services.ModifierKeyCapsLockModifier, true, data)
	})
	t.Run("Lower letter keys are correctly translated", func(t *testing.T) {
		expectKeys(t, mac(map[string]any{"keyCode": 0, "characters": "a", "charactersIgnoringModifiers": "a", "modifiers": 0}),
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, str("a"))
	})
	t.Run("Upper letter keys are correctly translated", func(t *testing.T) {
		expectKeys(t, mac(map[string]any{"keyCode": 0, "characters": "A", "charactersIgnoringModifiers": "A", "modifiers": 0x20002}),
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, str("A"))
	})
	t.Run("Control keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, mac(map[string]any{"keyCode": 0x35, "characters": "", "charactersIgnoringModifiers": "", "modifiers": 0}),
			services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, str(""))
	})
	t.Run("Modifier keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, mac(map[string]any{"keyCode": 0x38, "characters": "", "charactersIgnoringModifiers": "",
			"modifiers": services.RawKeyEventDataMacOsModifierLeftShift}),
			services.PhysicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftLeft, str(""))
	})
	t.Run("Unprintable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, mac(map[string]any{"keyCode": 0x7B, "characters": "", "charactersIgnoringModifiers": "",
			"modifiers": services.RawKeyEventDataMacOsModifierFunction}),
			services.PhysicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowLeft, nil)
	})
	t.Run("Multi-char keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, mac(map[string]any{"keyCode": 0, "characters": "án", "charactersIgnoringModifiers": "án", "modifiers": 0}),
			services.PhysicalKeyboardKeyKeyA, services.NewLogicalKeyboardKey(0x1400000000), nil)
	})
	t.Run("Prioritize logical key from specifiedLogicalKey", func(t *testing.T) {
		expectKeys(t, mac(map[string]any{"keyCode": 0x12, "characters": "&", "charactersIgnoringModifiers": "&",
			"specifiedLogicalKey": 0x31, "modifiers": 0}), services.PhysicalKeyboardKeyDigit1, services.LogicalKeyboardKeyDigit1, nil)
	})
	m := func() services.RawKeyEventData {
		return mac(map[string]any{"keyCode": 0x60, "characters": "A", "charactersIgnoringModifiers": "a", "modifiers": 0x10})
	}
	t.Run("data.toString", func(t *testing.T) {
		expectToString(t, m(), "RawKeyEventDataMacOs#00000(characters: A, charactersIgnoringModifiers: a, keyCode: 96, modifiers: 16)")
	})
	t.Run("data.equality", func(t *testing.T) {
		expectEqual(t, m(), services.NewRawKeyEventDataMacOs(services.RawKeyEventDataMacOsOptions{KeyCode: 0x60, Characters: "A",
			CharactersIgnoringModifiers: "a", Modifiers: 0x10}), true)
		expectEqual(t, m(), services.NewRawKeyEventDataMacOs(), false)
	})
}

func TestRawKeyEventDataIos(t *testing.T) {
	tests := appleModifierTests(services.RawKeyEventDataIosModifierOption, services.RawKeyEventDataIosModifierLeftOption,
		services.RawKeyEventDataIosModifierRightOption, services.RawKeyEventDataIosModifierShift, services.RawKeyEventDataIosModifierLeftShift,
		services.RawKeyEventDataIosModifierRightShift, services.RawKeyEventDataIosModifierControl, services.RawKeyEventDataIosModifierLeftControl,
		services.RawKeyEventDataIosModifierRightControl, services.RawKeyEventDataIosModifierCommand, services.RawKeyEventDataIosModifierLeftCommand,
		services.RawKeyEventDataIosModifierRightCommand, services.RawKeyEventDataIosModifierCapsLock)
	ios := func(m map[string]any) services.RawKeyEventData { return appleMessage("ios", m) }
	data := func(modifier int) services.RawKeyEventData {
		return ios(map[string]any{"keyCode": 0x04, "characters": "a", "charactersIgnoringModifiers": "a", "modifiers": modifier})
	}
	t.Run("modifier keys are recognized individually", func(t *testing.T) { checkModifiersIndividually(t, tests, true, data) })
	t.Run("modifier keys are recognized when combined", func(t *testing.T) {
		checkModifiersCombined(t, tests, services.RawKeyEventDataIosModifierCapsLock, services.ModifierKeyCapsLockModifier, true, data)
	})
	t.Run("Printable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, ios(map[string]any{"keyCode": 0x04, "characters": "a", "charactersIgnoringModifiers": "a", "modifiers": 0}),
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, str("a"))
	})
	t.Run("Control keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, ios(map[string]any{"keyCode": 0x29, "characters": "", "charactersIgnoringModifiers": "", "modifiers": 0}),
			services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, str(""))
	})
	t.Run("Modifier keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, ios(map[string]any{"keyCode": 0xe1, "characters": "", "charactersIgnoringModifiers": "",
			"modifiers": services.RawKeyEventDataIosModifierLeftShift}),
			services.PhysicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftLeft, str(""))
	})
	t.Run("Unprintable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, ios(map[string]any{"keyCode": 0x50, "characters": "", "charactersIgnoringModifiers": "UIKeyInputLeftArrow",
			"modifiers": services.RawKeyEventDataIosModifierFunction}),
			services.PhysicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowLeft, nil)
	})
	m := func() services.RawKeyEventData {
		return ios(map[string]any{"keyCode": 0x04, "characters": "A", "charactersIgnoringModifiers": "a", "modifiers": 0x10})
	}
	t.Run("data.toString", func(t *testing.T) {
		expectToString(t, m(), "RawKeyEventDataIos#00000(characters: A, charactersIgnoringModifiers: a, keyCode: 4, modifiers: 16)")
	})
	t.Run("data.equality", func(t *testing.T) {
		expectEqual(t, m(), services.NewRawKeyEventDataIos(services.RawKeyEventDataIosOptions{KeyCode: 4, Characters: "A",
			CharactersIgnoringModifiers: "a", Modifiers: 0x10}), true)
		expectEqual(t, m(), services.NewRawKeyEventDataIos(), false)
	})
}

func TestRawKeyEventDataWindows(t *testing.T) {
	tests := []modifierCheck{
		{services.RawKeyEventDataWindowsModifierLeftAlt, services.ModifierKeyAltModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataWindowsModifierRightAlt, services.ModifierKeyAltModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataWindowsModifierLeftShift, services.ModifierKeyShiftModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataWindowsModifierRightShift, services.ModifierKeyShiftModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataWindowsModifierLeftControl, services.ModifierKeyControlModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataWindowsModifierRightControl, services.ModifierKeyControlModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataWindowsModifierLeftMeta, services.ModifierKeyMetaModifier, services.KeyboardSideLeft},
		{services.RawKeyEventDataWindowsModifierRightMeta, services.ModifierKeyMetaModifier, services.KeyboardSideRight},
		{services.RawKeyEventDataWindowsModifierShift, services.ModifierKeyShiftModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataWindowsModifierControl, services.ModifierKeyControlModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataWindowsModifierAlt, services.ModifierKeyAltModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataWindowsModifierCaps, services.ModifierKeyCapsLockModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataWindowsModifierNumLock, services.ModifierKeyNumLockModifier, services.KeyboardSideAll},
		{services.RawKeyEventDataWindowsModifierScrollLock, services.ModifierKeyScrollLockModifier, services.KeyboardSideAll},
	}
	windows := func(m map[string]any) services.RawKeyEventData {
		m["type"], m["keymap"] = "keydown", "windows"
		return fromMessage(m)
	}
	data := func(modifier int) services.RawKeyEventData {
		return windows(map[string]any{"keyCode": 0x04, "characterCodePoint": 0, "scanCode": 0x04, "modifiers": modifier})
	}
	t.Run("modifier keys are recognized individually", func(t *testing.T) { checkModifiersIndividually(t, tests, true, data) })
	t.Run("modifier keys are recognized when combined", func(t *testing.T) {
		checkModifiersCombined(t, tests, services.RawKeyEventDataWindowsModifierCaps, services.ModifierKeyCapsLockModifier, true, data)
	})
	t.Run("Printable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, windows(map[string]any{"keyCode": 0, "characterCodePoint": 97, "scanCode": 0x1e, "modifiers": 0}),
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, str("a"))
	})
	t.Run("Control keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, windows(map[string]any{"keyCode": 27, "scanCode": 0x01, "characterCodePoint": 0, "modifiers": 0}),
			services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, str(""))
	})
	t.Run("Modifier keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, windows(map[string]any{"keyCode": 160, "scanCode": 0x2a, "characterCodePoint": 0,
			"modifiers": services.RawKeyEventDataWindowsModifierLeftShift}),
			services.PhysicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftLeft, str(""))
	})
	t.Run("Unprintable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, windows(map[string]any{"keyCode": 37, "scanCode": 0xe04b, "characterCodePoint": 0, "modifiers": 0}),
			services.PhysicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowLeft, nil)
	})
	t.Run("Win32 VK_PROCESSKEY events are skipped", func(t *testing.T) {
		keyTest(t, modeKeyDataThenRawKeyData, func() {
			lastHandled := true
			var events []services.RawKeyEvent
			listener := func(e services.RawKeyEvent) { events = append(events, e) }
			services.RawKeyboardInstance.AddListener(listener)
			defer services.RawKeyboardInstance.RemoveListener(listener)
			// The focused node's onKey.
			ensureInitialized().KeyEventManager().KeyMessageHandler = func(m *services.KeyMessage) bool {
				if m.RawEvent != nil {
					events = append(events, m.RawEvent)
				}
				return false
			}
			keyDown(services.LogicalKeyboardKeyKeyA)
			keyUp(services.LogicalKeyboardKeyKeyA)
			expect(len(events) == 4, "events %d", len(events))
			events = nil
			simulate := func(typ string, keyCode, scanCode int) {
				data := []dart.MapEntry[string, any]{e("type", typ), e("keymap", "windows"), e("keyCode", keyCode),
					e("scanCode", scanCode), e("modifiers", 0)}
				channel := services.SystemChannels.KeyEvent
				messenger().HandlePlatformMessage(channel.Name, channel.Codec.EncodeMessage(data), func(reply []byte) {
					lastHandled = channel.Codec.DecodeMessage(reply).(map[string]any)["handled"].(bool)
				}).Await()
			}
			simulate("keydown", 229, 30)
			expect(len(events) == 0 && lastHandled, "events %d handled %v", len(events), lastHandled)
			expectKeysPressed()
			simulate("keyup", 65, 30)
			expect(len(events) == 0 && lastHandled, "events %d handled %v", len(events), lastHandled)
			expectKeysPressed()
		})
	})
	m := func() services.RawKeyEventData {
		return windows(map[string]any{"keyCode": 0x10, "characterCodePoint": 10, "scanCode": 0x1e, "modifiers": 0x20})
	}
	t.Run("data.toString", func(t *testing.T) {
		expectToString(t, m(), "RawKeyEventDataWindows#00000(keyCode: 16, scanCode: 30, characterCodePoint: 10, modifiers: 32)")
	})
	t.Run("data.equality", func(t *testing.T) {
		expectEqual(t, m(), services.NewRawKeyEventDataWindows(services.RawKeyEventDataWindowsOptions{KeyCode: 0x10, ScanCode: 0x1e,
			Modifiers: 0x20, CharacterCodePoint: 10}), true)
		expectEqual(t, m(), services.NewRawKeyEventDataWindows(), false)
	})
}

func linuxGroup(t *testing.T, toolkit string, tests []modifierCheck, keyCodeFor func(modifier int, isLeft bool) int, control int,
	escapeKeyCode, shiftKeyCode int, debugToolkit string) {
	linux := func(m map[string]any) services.RawKeyEventData {
		if _, ok := m["type"]; !ok {
			m["type"] = "keydown"
		}
		m["keymap"], m["toolkit"] = "linux", toolkit
		return fromMessage(m)
	}
	t.Run("modifier keys are recognized individually", func(t *testing.T) {
		for _, mt := range tests {
			for _, isDown := range []bool{true, false} {
				for _, isLeft := range []bool{true, false} {
					typ, modifiers := "keyup", mt.modifier
					if isDown {
						typ, modifiers = "keydown", 0
					}
					d := linux(map[string]any{"type": typ, "keyCode": keyCodeFor(mt.modifier, isLeft), "scanCode": 0x26,
						"unicodeScalarValues": 97, "modifiers": modifiers})
					for _, key := range services.ModifierKeyValues {
						pressed := d.IsModifierPressed(key, services.IsModifierPressedOptions{Side: mt.side})
						if mt.key == key {
							if pressed != isDown {
								t.Errorf("%v pressed %v with %#x down=%v left=%v", key, pressed, mt.modifier, isDown, isLeft)
							}
							if s := d.GetModifierSide(key); s == nil || *s != mt.side {
								t.Errorf("%v side %v", key, s)
							}
						} else if pressed {
							t.Errorf("%v should not be pressed with %#x", key, mt.modifier)
						}
					}
				}
			}
		}
	})
	t.Run("modifier keys are recognized when combined", func(t *testing.T) {
		checkModifiersCombined(t, tests, control, services.ModifierKeyControlModifier, true, func(modifier int) services.RawKeyEventData {
			return linux(map[string]any{"keyCode": 65, "scanCode": 0x26, "unicodeScalarValues": 97, "modifiers": modifier})
		})
	})
	t.Run("Printable keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, linux(map[string]any{"keyCode": 65, "scanCode": 0x26, "unicodeScalarValues": 113, "modifiers": 0}),
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyQ, str("q"))
	})
	t.Run("Code points with two Unicode scalar values are allowed", func(t *testing.T) {
		d := linux(map[string]any{"keyCode": 65, "scanCode": 0x26, "unicodeScalarValues": 0x10FFFF, "modifiers": 0})
		expectKeys(t, d, services.PhysicalKeyboardKeyKeyA, services.NewLogicalKeyboardKey(0x10FFFF), str("\U0010FFFF"))
	})
	t.Run("Code points with more than three Unicode scalar values are not allowed", func(t *testing.T) {
		defer func() {
			if _, ok := recover().(*dart.AssertionError); !ok {
				t.Error("no AssertionError")
			}
		}()
		linux(map[string]any{"keyCode": 65, "scanCode": 0x26, "unicodeScalarValues": 0x1F00000000, "modifiers": 0})
	})
	t.Run("Control keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, linux(map[string]any{"keyCode": escapeKeyCode, "scanCode": 0x09, "unicodeScalarValues": 0, "modifiers": 0}),
			services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, str(""))
	})
	t.Run("Modifier keyboard keys are correctly translated", func(t *testing.T) {
		expectKeys(t, linux(map[string]any{"keyCode": shiftKeyCode, "scanCode": 0x32, "unicodeScalarValues": 0}),
			services.PhysicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftLeft, str(""))
	})
	usv := 0x10FFFF
	if toolkit == "gtk" {
		usv = 113
	}
	m := func() services.RawKeyEventData {
		return linux(map[string]any{"keyCode": 65, "scanCode": 0x26, "unicodeScalarValues": usv, "modifiers": 0x10})
	}
	t.Run("data.toString", func(t *testing.T) {
		expectToString(t, m(), "RawKeyEventDataLinux#00000(toolkit: "+debugToolkit+", unicodeScalarValues: "+itoa(usv)+
			", scanCode: 38, keyCode: 65, modifiers: 16, isDown: true)")
	})
	t.Run("data.equality", func(t *testing.T) {
		expectEqual(t, m(), services.NewRawKeyEventDataLinux(services.NewKeyHelper(toolkit), true, services.RawKeyEventDataLinuxOptions{
			UnicodeScalarValues: usv, KeyCode: 65, ScanCode: 0x26, Modifiers: 0x10}), true)
		if toolkit == "gtk" {
			expectEqual(t, m(), services.NewRawKeyEventDataLinux(services.NewKeyHelper("glfw"), true, services.RawKeyEventDataLinuxOptions{
				UnicodeScalarValues: usv, KeyCode: 65, ScanCode: 0x26, Modifiers: 0x10}), false)
		}
		expectEqual(t, m(), services.NewRawKeyEventDataLinux(services.NewKeyHelper(toolkit), true), false)
	})
	if toolkit == "gtk" {
		t.Run("Prioritize logical key from specifiedLogicalKey", func(t *testing.T) {
			expectKeys(t, linux(map[string]any{"keyCode": 0x6c6, "scanCode": 0x26, "unicodeScalarValues": 0x424, "specifiedLogicalKey": 0x61}),
				services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, nil)
		})
	}
}

func itoa(i int) string { return describe(i) }

func TestRawKeyEventDataLinuxGLFW(t *testing.T) {
	tests := []modifierCheck{
		{services.GLFWKeyHelperModifierAlt, services.ModifierKeyAltModifier, services.KeyboardSideAll},
		{services.GLFWKeyHelperModifierShift, services.ModifierKeyShiftModifier, services.KeyboardSideAll},
		{services.GLFWKeyHelperModifierControl, services.ModifierKeyControlModifier, services.KeyboardSideAll},
		{services.GLFWKeyHelperModifierMeta, services.ModifierKeyMetaModifier, services.KeyboardSideAll},
		{services.GLFWKeyHelperModifierNumericPad, services.ModifierKeyNumLockModifier, services.KeyboardSideAll},
		{services.GLFWKeyHelperModifierCapsLock, services.ModifierKeyCapsLockModifier, services.KeyboardSideAll},
	}
	keyCodeFor := func(modifier int, isLeft bool) int {
		pick := func(l, r int) int {
			if isLeft {
				return l
			}
			return r
		}
		switch modifier {
		case services.GLFWKeyHelperModifierAlt:
			return pick(342, 346)
		case services.GLFWKeyHelperModifierShift:
			return pick(340, 344)
		case services.GLFWKeyHelperModifierControl:
			return pick(341, 345)
		case services.GLFWKeyHelperModifierMeta:
			return pick(343, 347)
		case services.GLFWKeyHelperModifierNumericPad:
			return 282
		case services.GLFWKeyHelperModifierCapsLock:
			return 280
		}
		return 65 // keyA
	}
	linuxGroup(t, "glfw", tests, keyCodeFor, services.GLFWKeyHelperModifierControl, 256, 340, "GLFW")
}

func TestRawKeyEventDataLinuxGTK(t *testing.T) {
	tests := []modifierCheck{
		{services.GtkKeyHelperModifierMod1, services.ModifierKeyAltModifier, services.KeyboardSideAll},
		{services.GtkKeyHelperModifierShift, services.ModifierKeyShiftModifier, services.KeyboardSideAll},
		{services.GtkKeyHelperModifierControl, services.ModifierKeyControlModifier, services.KeyboardSideAll},
		{services.GtkKeyHelperModifierMeta, services.ModifierKeyMetaModifier, services.KeyboardSideAll},
		{services.GtkKeyHelperModifierMod2, services.ModifierKeyNumLockModifier, services.KeyboardSideAll},
		{services.GtkKeyHelperModifierCapsLock, services.ModifierKeyCapsLockModifier, services.KeyboardSideAll},
	}
	keyCodeFor := func(modifier int, isLeft bool) int {
		pick := func(l, r int) int {
			if isLeft {
				return l
			}
			return r
		}
		switch modifier {
		case services.GtkKeyHelperModifierShift:
			return pick(65505, 65506)
		case services.GtkKeyHelperModifierControl:
			return pick(65507, 65508)
		case services.GtkKeyHelperModifierMeta:
			return pick(65515, 65516)
		case services.GtkKeyHelperModifierMod1:
			return 65513
		case services.GtkKeyHelperModifierMod2:
			return 65407
		case services.GtkKeyHelperModifierCapsLock:
			return 65509
		}
		return 65 // keyA
	}
	linuxGroup(t, "gtk", tests, keyCodeFor, services.GtkKeyHelperModifierControl, 65307, 65505, "GTK")
}

func TestRawKeyEventDataWeb(t *testing.T) {
	tests := []struct {
		modifier int
		key      services.ModifierKey
	}{
		{services.RawKeyEventDataWebModifierAlt, services.ModifierKeyAltModifier},
		{services.RawKeyEventDataWebModifierShift, services.ModifierKeyShiftModifier},
		{services.RawKeyEventDataWebModifierControl, services.ModifierKeyControlModifier},
		{services.RawKeyEventDataWebModifierMeta, services.ModifierKeyMetaModifier},
		{services.RawKeyEventDataWebModifierCapsLock, services.ModifierKeyCapsLockModifier},
		{services.RawKeyEventDataWebModifierNumLock, services.ModifierKeyNumLockModifier},
		{services.RawKeyEventDataWebModifierScrollLock, services.ModifierKeyScrollLockModifier},
	}
	web := func(m map[string]any) services.RawKeyEventData {
		m["type"], m["keymap"] = "keydown", "web"
		return fromMessage(m)
	}
	t.Run("modifier keys are recognized individually", func(t *testing.T) {
		for _, mt := range tests {
			d := web(map[string]any{"code": "RandomCode", "metaState": mt.modifier})
			for _, key := range services.ModifierKeyValues {
				if d.IsModifierPressed(key) != (mt.key == key) {
					t.Errorf("%v with %#x", key, mt.modifier)
				}
			}
		}
	})
	t.Run("modifier keys are recognized when combined", func(t *testing.T) {
		for _, mt := range tests {
			if mt.modifier == services.RawKeyEventDataWebModifierMeta {
				continue
			}
			d := web(map[string]any{"code": "RandomCode", "metaState": mt.modifier | services.RawKeyEventDataWebModifierMeta})
			for _, key := range services.ModifierKeyValues {
				if d.IsModifierPressed(key) != (mt.key == key || key == services.ModifierKeyMetaModifier) {
					t.Errorf("%v with %#x|meta", key, mt.modifier)
				}
			}
		}
	})
	t.Run("modifier keys with no location are mapped to left", func(t *testing.T) {
		for key, want := range map[string]L{"Alt": services.LogicalKeyboardKeyAltLeft, "Shift": services.LogicalKeyboardKeyShiftLeft,
			"Control": services.LogicalKeyboardKeyControlLeft, "Meta": services.LogicalKeyboardKeyMetaLeft} {
			if got := web(map[string]any{"key": key, "location": 0}).LogicalKey(); got != want {
				t.Errorf("%s: %v", key, got)
			}
		}
	})
	expectWeb := func(t *testing.T, m map[string]any, physical services.PhysicalKeyboardKey, logical L, label string, keyCode int) {
		t.Helper()
		d := web(m)
		expectKeys(t, d, physical, logical, &label)
		if got := d.(*services.RawKeyEventDataWeb).KeyCode; got != keyCode {
			t.Errorf("keyCode %d", got)
		}
	}
	t.Run("Lower letter keys are correctly translated", func(t *testing.T) {
		expectWeb(t, map[string]any{"code": "KeyA", "key": "a", "location": 0, "metaState": 0, "keyCode": 0x41},
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, "a", 0x41)
	})
	t.Run("Upper letter keys are correctly translated", func(t *testing.T) {
		expectWeb(t, map[string]any{"code": "KeyA", "key": "A", "location": 0, "metaState": 0x1, "keyCode": 0x41},
			services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, "A", 0x41)
	})
	t.Run("Control keyboard keys are correctly translated", func(t *testing.T) {
		expectWeb(t, map[string]any{"code": "Escape", "key": "Escape", "location": 0, "metaState": 0, "keyCode": 0x1B},
			services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, "", 0x1B)
	})
	t.Run("Modifier keyboard keys are correctly translated", func(t *testing.T) {
		expectWeb(t, map[string]any{"code": "ShiftLeft", "key": "Shift", "location": 1, "metaState": services.RawKeyEventDataWebModifierShift,
			"keyCode": 0x10}, services.PhysicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftLeft, "", 0x10)
	})
	t.Run("Esc keys generated by older browsers are correctly translated", func(t *testing.T) {
		expectWeb(t, map[string]any{"code": "Esc", "key": "Esc", "location": 0, "metaState": 0, "keyCode": 0x1B},
			services.PhysicalKeyboardKeyEscape, services.LogicalKeyboardKeyEscape, "", 0x1B)
	})
	t.Run("Arrow keys from a keyboard give correct physical key mappings", func(t *testing.T) {
		expectWeb(t, map[string]any{"code": "ArrowDown", "key": "ArrowDown", "location": 0, "metaState": 0, "keyCode": 0x28},
			services.PhysicalKeyboardKeyArrowDown, services.LogicalKeyboardKeyArrowDown, "", 0x28)
	})
	t.Run("Unrecognized keys are mapped to Web plane", func(t *testing.T) {
		d := web(map[string]any{"code": "Unrecog1", "key": "Unrecog2", "location": 0, "metaState": 0})
		p, l := d.PhysicalKey().UsbHidUsage, d.LogicalKey().KeyID
		if !(p > 0x01700000000 && p < 0x01800000000 && l > 0x01700000000 && l < 0x01800000000) || d.KeyLabel() != "" ||
			d.(*services.RawKeyEventDataWeb).KeyCode != 0 {
			t.Errorf("%#x %#x %q", p, l, d.KeyLabel())
		}
	})
	t.Run("data.toString", func(t *testing.T) {
		expectToString(t, web(map[string]any{"code": "KeyA", "key": "a", "location": 2, "metaState": 0x10, "keyCode": 0x41}),
			"RawKeyEventDataWeb#00000(code: KeyA, key: a, location: 2, metaState: 16, keyCode: 65)")
		expectToString(t, web(map[string]any{"code": "KeyA", "key": "a", "metaState": 0x10, "keyCode": 0x41}),
			"RawKeyEventDataWeb#00000(code: KeyA, key: a, location: 0, metaState: 16, keyCode: 65)")
	})
	t.Run("data.equality", func(t *testing.T) {
		d := web(map[string]any{"code": "KeyA", "key": "a", "location": 2, "metaState": 0x10, "keyCode": 0x41})
		expectEqual(t, d, services.NewRawKeyEventDataWeb(services.RawKeyEventDataWebOptions{Key: "a", Code: "KeyA", Location: 2,
			MetaState: 0x10, KeyCode: 0x41}), true)
		expectEqual(t, d, services.NewRawKeyEventDataWeb(services.RawKeyEventDataWebOptions{Code: "KeyA", Key: "a"}), false)
	})
}
