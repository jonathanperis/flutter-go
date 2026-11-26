package difftest

// Go twin of flutter/test/key_events_test.dart: the same key event
// sequences through flutter_test's simulators and KeyEventManager, per
// platform and transit mode, compared line by line with Flutter's log.

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type keyEventsLog struct {
	lines []string
	raw   func(services.RawKeyEvent)
}

func (l *keyEventsLog) add(s string) { l.lines = append(l.lines, s) }

func hex0x(v int) string { return "0x" + strconv.FormatInt(int64(v), 16) }

func (l *keyEventsLog) logState() {
	hk := services.HardwareKeyboardInstance()
	var hp, hl, locks, rp, rl []string
	for _, k := range hk.PhysicalKeysPressed() {
		hp = append(hp, hex0x(k.UsbHidUsage))
	}
	for _, k := range hk.LogicalKeysPressed() {
		hl = append(hl, hex0x(k.KeyID))
	}
	for _, m := range hk.LockModesEnabled() {
		locks = append(locks, m.Name())
	}
	for _, k := range services.RawKeyboardInstance.PhysicalKeysPressed() {
		rp = append(rp, hex0x(k.UsbHidUsage))
	}
	for _, k := range services.RawKeyboardInstance.KeysPressed() {
		rl = append(rl, hex0x(k.KeyID))
	}
	l.add(fmt.Sprintf("  state hw=[%s] [%s] locks=[%s] raw=[%s] [%s] ctrl=%v shift=%v alt=%v meta=%v",
		strings.Join(hp, ","), strings.Join(hl, ","), strings.Join(locks, ","), strings.Join(rp, ","), strings.Join(rl, ","),
		hk.IsControlPressed(), hk.IsShiftPressed(), hk.IsAltPressed(), hk.IsMetaPressed()))
}

func jsonNullable(s *string) string {
	if s == nil {
		return "null"
	}
	return dart.JSONEncode(*s)
}

func (l *keyEventsLog) install() {
	services.HardwareKeyboardInstance().AddHandler(func(event services.KeyEvent) bool {
		l.add("  handler " + stripHashes(event.String()))
		return event.LogicalKey() == services.LogicalKeyboardKeyKeyB
	})
	l.raw = func(event services.RawKeyEvent) {
		l.add(fmt.Sprintf("  raw %s %s char=%s repeat=%v", foundation.ObjectRuntimeType(event, ""),
			stripHashes(event.Data().String()), jsonNullable(event.Character()), event.Repeat()))
	}
	services.RawKeyboardInstance.AddListener(l.raw)
	services.ServicesBindingInstance().KeyEventManager().KeyMessageHandler = func(message *services.KeyMessage) bool {
		raw := "null"
		if message.RawEvent != nil {
			raw = foundation.ObjectRuntimeType(message.RawEvent, "")
		}
		l.add("  message " + stripHashes(message.String()) + " raw=" + raw)
		for _, e := range message.Events {
			if e.LogicalKey() == services.LogicalKeyboardKeyKeyC {
				return true
			}
		}
		return false
	}
}

func (l *keyEventsLog) reset() {
	services.RawKeyboardInstance.RemoveListener(l.raw)
	services.RawKeyboardInstance.ClearKeysPressed()
	services.HardwareKeyboardInstance().ClearState()
	services.ServicesBindingInstance().KeyEventManager().ClearState()
}

// awaitOn runs the tester's loop until f completes (an await in the test
// body).
func awaitOn[T any](tester *flutter_test.Tester, f async.Future[T]) T {
	var result T
	done := false
	var err error
	async.OnComplete(f, func(v T) { result, done = v, true }, func(e error) { err, done = e, true })
	for i := 0; !done && i < 1000; i++ {
		tester.Loop().FlushMicrotasks()
		tester.Loop().RunPosted()
	}
	if err != nil {
		panic(err)
	}
	if !done {
		panic("future never completed")
	}
	return result
}

func (l *keyEventsLog) step(tester *flutter_test.Tester, desc string, f func() bool) {
	l.add("step " + desc)
	result := awaitOn(tester, async.Async(f))
	l.add(fmt.Sprintf("  -> %v", result))
	l.logState()
}

var keyPlatforms = []string{"android", "fuchsia", "macos", "ios", "linux", "windows", "web"}

func shiftFlags(platform string) []dart.MapEntry[string, any] {
	switch platform {
	case "android":
		return []dart.MapEntry[string, any]{{Key: "metaState", Value: services.RawKeyEventDataAndroidModifierLeftShift | services.RawKeyEventDataAndroidModifierShift}}
	case "fuchsia":
		return []dart.MapEntry[string, any]{{Key: "modifiers", Value: services.RawKeyEventDataFuchsiaModifierLeftShift}}
	case "macos":
		return []dart.MapEntry[string, any]{{Key: "modifiers", Value: services.RawKeyEventDataMacOsModifierLeftShift | services.RawKeyEventDataMacOsModifierShift}}
	case "ios":
		return []dart.MapEntry[string, any]{{Key: "modifiers", Value: services.RawKeyEventDataIosModifierLeftShift | services.RawKeyEventDataIosModifierShift}}
	case "linux":
		return []dart.MapEntry[string, any]{{Key: "modifiers", Value: services.GLFWKeyHelperModifierShift}}
	case "windows":
		return []dart.MapEntry[string, any]{{Key: "modifiers", Value: services.RawKeyEventDataWindowsModifierLeftShift | services.RawKeyEventDataWindowsModifierShift}}
	}
	return []dart.MapEntry[string, any]{{Key: "metaState", Value: services.RawKeyEventDataWebModifierShift}}
}

// putEntry is Dart's map[key] = value on an ordered map.
func putEntry(d []dart.MapEntry[string, any], k string, v any) []dart.MapEntry[string, any] {
	for i := range d {
		if d[i].Key == k {
			d[i].Value = v
			return d
		}
	}
	return append(d, dart.MapEntry[string, any]{Key: k, Value: v})
}

func sendRawKeyMessage(data []dart.MapEntry[string, any]) {
	channel := services.SystemChannels.KeyEvent
	flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().HandlePlatformMessage(
		channel.Name, channel.Codec.EncodeMessage(data), func([]byte) {}).Await()
}

func nullableInt(p *services.PhysicalKeyboardKey) string {
	if p == nil {
		return "null"
	}
	return strconv.Itoa(p.UsbHidUsage)
}

func keyEventsScene(t *testing.T, l *keyEventsLog, platform string, mode services.KeyDataTransitMode) {
	tester := flutter_test.New(t)
	m := mode
	services.DebugKeyEventSimulatorTransitModeOverride = &m
	defer func() { services.DebugKeyEventSimulatorTransitModeOverride = nil }()
	l.add("scene " + platform + " " + mode.Name())
	l.install()
	down := func(k services.LogicalKeyboardKey, physical *services.PhysicalKeyboardKey, character *string) {
		c := "null"
		if character != nil {
			c = *character
		}
		l.step(tester, "down "+k.KeyLabel()+" "+nullableInt(physical)+" "+c, func() bool {
			return flutter_test.SimulateKeyDownEvent(k, flutter_test.SimulateKeyEventOptions{Platform: platform, PhysicalKey: physical, Character: character}).Await()
		})
	}
	up := func(k services.LogicalKeyboardKey, physical *services.PhysicalKeyboardKey) {
		l.step(tester, "up "+k.KeyLabel()+" "+nullableInt(physical), func() bool {
			return flutter_test.SimulateKeyUpEvent(k, flutter_test.SimulateKeyEventOptions{Platform: platform, PhysicalKey: physical}).Await()
		})
	}
	repeat := func(k services.LogicalKeyboardKey) {
		l.step(tester, "repeat "+k.KeyLabel(), func() bool {
			return flutter_test.SimulateKeyRepeatEvent(k, flutter_test.SimulateKeyEventOptions{Platform: platform}).Await()
		})
	}
	tap := func(k services.LogicalKeyboardKey) {
		down(k, nil, nil)
		up(k, nil)
	}
	down(services.LogicalKeyboardKeyShiftLeft, nil, nil)
	down(services.LogicalKeyboardKeyKeyA, nil, nil)
	if mode == services.KeyDataTransitModeKeyDataThenRawKeyData {
		repeat(services.LogicalKeyboardKeyKeyA)
	}
	up(services.LogicalKeyboardKeyKeyA, nil)
	up(services.LogicalKeyboardKeyShiftLeft, nil)
	tap(services.LogicalKeyboardKeyKeyB)
	x := "x"
	down(services.LogicalKeyboardKeyKeyC, nil, &x)
	up(services.LogicalKeyboardKeyKeyC, nil)
	tap(services.LogicalKeyboardKeyCapsLock)
	tap(services.LogicalKeyboardKeyCapsLock)
	if platform != "macos" && platform != "ios" && platform != "fuchsia" {
		tap(services.LogicalKeyboardKeyNumLock)
	}
	down(services.LogicalKeyboardKeyControlLeft, nil, nil)
	down(services.LogicalKeyboardKeyAltLeft, nil, nil)
	down(services.LogicalKeyboardKeyMetaLeft, nil, nil)
	tap(services.LogicalKeyboardKeyDigit1)
	up(services.LogicalKeyboardKeyMetaLeft, nil)
	up(services.LogicalKeyboardKeyAltLeft, nil)
	up(services.LogicalKeyboardKeyControlLeft, nil)
	down(services.LogicalKeyboardKeyShiftRight, nil, nil)
	tap(services.LogicalKeyboardKeyArrowLeft)
	up(services.LogicalKeyboardKeyShiftRight, nil)
	if platform != "linux" && platform != "windows" && platform != "ios" && platform != "web" {
		down(services.LogicalKeyboardKeyFn, nil, nil)
		down(services.LogicalKeyboardKeyF12, nil, nil)
		up(services.LogicalKeyboardKeyFn, nil)
		up(services.LogicalKeyboardKeyF12, nil)
	}
	tap(services.LogicalKeyboardKeyEnter)
	tap(services.LogicalKeyboardKeyNumpad1)
	if mode == services.KeyDataTransitModeKeyDataThenRawKeyData {
		physA := services.PhysicalKeyboardKeyKeyA
		down(services.LogicalKeyboardKeyKeyB, &physA, nil)
		up(services.LogicalKeyboardKeyKeyB, &physA)
	}
	if mode == services.KeyDataTransitModeRawKeyData {
		l.reset()
		l.install()
		isUp := false
		l.step(tester, "raw keyA with shift flags", func() bool {
			data := flutter_test.KeyEventSimulator.GetKeyData(services.LogicalKeyboardKeyKeyA, platform)
			for _, e := range shiftFlags(platform) {
				data = putEntry(data, e.Key, e.Value)
			}
			sendRawKeyMessage(data)
			return false
		})
		l.step(tester, "raw keyA up without flags", func() bool {
			sendRawKeyMessage(flutter_test.KeyEventSimulator.GetKeyData(services.LogicalKeyboardKeyKeyA, platform, flutter_test.GetKeyDataOptions{IsDown: &isUp}))
			return false
		})
		l.step(tester, "raw shiftLeft down without flags", func() bool {
			sendRawKeyMessage(flutter_test.KeyEventSimulator.GetKeyData(services.LogicalKeyboardKeyKeyA, platform))
			data := flutter_test.KeyEventSimulator.GetKeyData(services.LogicalKeyboardKeyShiftLeft, platform)
			data = putEntry(data, "metaState", 0)
			data = putEntry(data, "modifiers", 0)
			sendRawKeyMessage(data)
			return false
		})
		l.step(tester, "raw keyA up", func() bool {
			sendRawKeyMessage(flutter_test.KeyEventSimulator.GetKeyData(services.LogicalKeyboardKeyKeyA, platform, flutter_test.GetKeyDataOptions{IsDown: &isUp}))
			return false
		})
	}
	l.reset()
}

func replyString(r map[string]any) string { return fmt.Sprintf("{handled: %v}", r["handled"]) }

func keyDataScene(t *testing.T, l *keyEventsLog) {
	tester := flutter_test.New(t)
	l.add("scene key data")
	l.install()
	manager := services.ServicesBindingInstance().KeyEventManager()
	data := func(typ ui.KeyEventType, physical, logical int, character *string, synthesized bool, us int) ui.KeyData {
		return ui.KeyData{Type: typ, Physical: physical, Logical: logical, TimeStamp: time.Duration(us) * time.Microsecond,
			Character: character, Synthesized: synthesized}
	}
	macRaw := func(typ string, keyCode, modifiers int, chars string) map[string]any {
		return map[string]any{"type": typ, "keymap": "macos", "keyCode": keyCode, "characters": chars,
			"charactersIgnoringModifiers": chars, "modifiers": modifiers}
	}
	str := func(s string) *string { return &s }
	keydata := func(d ui.KeyData) { l.add(fmt.Sprintf("  keydata %v", manager.HandleKeyData(d))) }
	reply := func(m map[string]any) { l.add("  reply " + replyString(manager.HandleRawKeyMessage(m).Await())) }
	down, up, rep := ui.KeyEventTypeDown, ui.KeyEventTypeUp, ui.KeyEventTypeRepeat
	pA, lA := services.PhysicalKeyboardKeyKeyA.UsbHidUsage, services.LogicalKeyboardKeyKeyA.KeyID
	pCaps, lCaps := services.PhysicalKeyboardKeyCapsLock.UsbHidUsage, services.LogicalKeyboardKeyCapsLock.KeyID
	pShift, lShift := services.PhysicalKeyboardKeyShiftLeft.UsbHidUsage, services.LogicalKeyboardKeyShiftLeft.KeyID
	pQ, lQ := services.PhysicalKeyboardKeyKeyQ.UsbHidUsage, services.LogicalKeyboardKeyKeyQ.KeyID
	pFn, lFn := services.PhysicalKeyboardKeyFn.UsbHidUsage, services.LogicalKeyboardKeyFn.KeyID
	l.step(tester, "empty key data", func() bool {
		keydata(data(down, 0, 0, str("a"), false, 0))
		reply(macRaw("keydown", 0x00, 0, "a"))
		return false
	})
	l.step(tester, "synthesized solitary", func() bool {
		keydata(data(up, pA, lA, nil, true, 5))
		return false
	})
	l.step(tester, "caps lock tap", func() bool {
		keydata(data(down, pCaps, lCaps, nil, false, 10))
		keydata(data(up, pCaps, lCaps, nil, true, 10))
		reply(macRaw("keydown", 0x39, 0x10000, ""))
		return false
	})
	l.step(tester, "batch", func() bool {
		keydata(data(down, pShift, lShift, nil, true, 20))
		keydata(data(down, pQ, lQ, str("Q"), false, 21))
		keydata(data(rep, pQ, 0x51, str("Q"), false, 22))
		keydata(data(down, 0x123456, 0x1400000099, nil, false, 23))
		reply(macRaw("keydown", 0x0c, services.RawKeyEventDataMacOsModifierLeftShift|services.RawKeyEventDataMacOsModifierShift, "Q"))
		return false
	})
	l.step(tester, "batch up", func() bool {
		for _, k := range [][2]int{{pQ, 0x51}, {0x123456, 0x1400000099}, {pShift, lShift}} {
			keydata(data(up, k[0], k[1], nil, false, 30))
		}
		reply(macRaw("keyup", 0x0c, 0, ""))
		return false
	})
	l.step(tester, "fn skipped", func() bool {
		keydata(data(down, pFn, lFn, nil, false, 40))
		reply(macRaw("keydown", 0x3f, 0x800000, ""))
		keydata(data(up, pFn, lFn, nil, false, 41))
		reply(macRaw("keyup", 0x3f, 0, ""))
		return false
	})
	l.reset()
}

// dartJSON encodes an ordered map as Dart's jsonEncode does.
func dartJSON(entries []dart.MapEntry[string, any]) string {
	var parts []string
	for _, e := range entries {
		parts = append(parts, dart.JSONEncode(e.Key)+":"+dart.JSONEncode(e.Value))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func TestKeyEventsMatchFlutter(t *testing.T) {
	t.Setenv("FLUTTER_TEST", "true")
	l := &keyEventsLog{}
	for _, mode := range services.KeyDataTransitModeValues {
		for _, platform := range keyPlatforms {
			t.Run(platform+" "+mode.Name(), func(t *testing.T) { keyEventsScene(t, l, platform, mode) })
		}
	}
	t.Run("key data sequences", func(t *testing.T) { keyDataScene(t, l) })
	isUp := false
	for _, platform := range keyPlatforms {
		for _, key := range []services.LogicalKeyboardKey{services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyShift,
			services.LogicalKeyboardKeyControlRight, services.LogicalKeyboardKeyEnter, services.LogicalKeyboardKeyNumpad5,
			services.LogicalKeyboardKeyBackquote, services.LogicalKeyboardKeyArrowUp, services.LogicalKeyboardKeyEscape} {
			l.add(fmt.Sprintf("getKeyData %s %s %s", platform, key.KeyLabel(), dartJSON(flutter_test.KeyEventSimulator.GetKeyData(key, platform))))
			l.add(fmt.Sprintf("getKeyData %s %s up %s", platform, key.KeyLabel(),
				dartJSON(flutter_test.KeyEventSimulator.GetKeyData(key, platform, flutter_test.GetKeyDataOptions{IsDown: &isUp}))))
		}
	}
	compareKeyLines(t, "flutter/build/key_events.json", l.lines)
}
