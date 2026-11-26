package app

// Port of the engine's FlutterEmbedderKeyResponderTest.mm (Flutter 3.50.1):
// BasicKeyEvent, NonAsciiCharacters, MultipleCharacters,
// SynthesizeForDuplicateDownEvent, IgnoreDuplicateUpEvent,
// ConvertAbruptRepeatEventsToDown, ToggleModifiersDuringKeyTap,
// ConvertCapsLockEvents, SynchronizeCapsLockStateOnCapsLock and
// SynchronizeCapsLockStateOnNormalKey (10 of 14). Not ported:
// SpecialModiferFlags, IdentifyLeftAndRightModifiers,
// SynthesizeMissedModifierEvents and
// SynthesizeMissedModifierEventsInNormalEvents (long tables of the same
// synchronizeModifiers path, covered here by ToggleModifiersDuringKeyTap
// and the CapsLock tests); plus a test of the channel responder's messages.

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

type sentKeyEvent struct {
	data     ui.KeyData
	callback func(bool)
}

const (
	kKeyCodeKeyA       = 0x00
	kKeyCodeKeyW       = 0x0d
	kKeyCodeShiftRight = 0x3c
	kKeyCodeCapsLock   = 0x39
	kKeyCodeAltRight   = 0x3d
)

var (
	kPhysicalKeyA       = services.PhysicalKeyboardKeyKeyA.UsbHidUsage
	kLogicalKeyA        = services.LogicalKeyboardKeyKeyA.KeyID
	kPhysicalKeyW       = services.PhysicalKeyboardKeyKeyW.UsbHidUsage
	kLogicalKeyW        = services.LogicalKeyboardKeyKeyW.KeyID
	kPhysicalShiftRight = services.PhysicalKeyboardKeyShiftRight.UsbHidUsage
	kLogicalShiftRight  = services.LogicalKeyboardKeyShiftRight.KeyID
	kPhysicalAltRight   = services.PhysicalKeyboardKeyAltRight.UsbHidUsage
	kLogicalAltRight    = services.LogicalKeyboardKeyAltRight.KeyID
	kPhysicalCapsLock   = services.PhysicalKeyboardKeyCapsLock.UsbHidUsage
	kLogicalCapsLock    = services.LogicalKeyboardKeyCapsLock.KeyID
)

func newTestResponder() (*embedderKeyResponder, *[]sentKeyEvent) {
	var events []sentKeyEvent
	r := newEmbedderKeyResponder(func(data ui.KeyData, callback func(bool)) {
		events = append(events, sentKeyEvent{data, callback})
	})
	return r, &events
}

func ev(typ nsEventType, flags int, chars, ignoring string, repeat bool, keyCode int) keyEvent {
	return keyEvent{Type: typ, ModifierFlags: flags, Characters: chars, CharactersIgnoringModifiers: ignoring,
		IsARepeat: repeat, KeyCode: keyCode}
}

func checkEvent(t *testing.T, e sentKeyEvent, typ ui.KeyEventType, physical, logical int, character *string, synthesized bool) {
	t.Helper()
	d := e.data
	if d.Type != typ || d.Physical != physical || d.Logical != logical || d.Synthesized != synthesized {
		t.Errorf("event %v: want type %v physical %#x logical %#x synthesized %v", d, typ, physical, logical, synthesized)
	}
	if (d.Character == nil) != (character == nil) || (character != nil && *d.Character != *character) {
		t.Errorf("character %v, want %v", d.Character, character)
	}
}

func s(v string) *string { return &v }

func TestEmbedderBasicKeyEvent(t *testing.T) {
	r, events := newTestResponder()
	lastHandled := false
	down := ev(nsEventTypeKeyDown, 0x100, "a", "a", false, 0)
	down.Timestamp = 123
	r.handleEvent(down, func(h bool) { lastHandled = h })
	if len(*events) != 1 {
		t.Fatal(len(*events))
	}
	e := (*events)[0]
	checkEvent(t, e, ui.KeyEventTypeDown, kPhysicalKeyA, kLogicalKeyA, s("a"), false)
	if e.data.TimeStamp != 123*time.Second {
		t.Errorf("timestamp %v", e.data.TimeStamp)
	}
	if lastHandled || e.callback == nil {
		t.Fatal("callback")
	}
	e.callback(true)
	if !lastHandled {
		t.Error("not handled")
	}
	*events = nil
	lastHandled = false
	r.handleEvent(ev(nsEventTypeKeyDown, 0x100, "a", "a", true, kKeyCodeKeyA), func(h bool) { lastHandled = h })
	checkEvent(t, (*events)[0], ui.KeyEventTypeRepeat, kPhysicalKeyA, kLogicalKeyA, s("a"), false)
	(*events)[0].callback(true)
	if !lastHandled {
		t.Error("repeat not handled")
	}
	*events = nil
	lastHandled = true
	up := ev(nsEventTypeKeyUp, 0x100, "a", "a", false, kKeyCodeKeyA)
	up.Timestamp = 124
	r.handleEvent(up, func(h bool) { lastHandled = h })
	checkEvent(t, (*events)[0], ui.KeyEventTypeUp, kPhysicalKeyA, kLogicalKeyA, nil, false)
	if (*events)[0].data.TimeStamp != 124*time.Second {
		t.Error("up timestamp")
	}
	(*events)[0].callback(false)
	if lastHandled {
		t.Error("responding false")
	}
}

func TestEmbedderNonAsciiCharacters(t *testing.T) {
	r, events := newTestResponder()
	nop := func(bool) {}
	r.handleEvent(ev(nsEventTypeFlagsChanged, 0x80140, "", "", false, kKeyCodeAltRight), nop)
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, kPhysicalAltRight, kLogicalAltRight, nil, false)
	*events = nil
	r.handleEvent(ev(nsEventTypeKeyDown, 0x80140, "∑", "w", false, kKeyCodeKeyW), nop)
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, kPhysicalKeyW, kLogicalKeyW, s("∑"), false)
	*events = nil
	r.handleEvent(ev(nsEventTypeFlagsChanged, 0x100, "", "", false, kKeyCodeAltRight), nop)
	checkEvent(t, (*events)[0], ui.KeyEventTypeUp, kPhysicalAltRight, kLogicalAltRight, nil, false)
	*events = nil
	r.handleEvent(ev(nsEventTypeKeyUp, 0x100, "w", "w", false, kKeyCodeKeyW), nop)
	checkEvent(t, (*events)[0], ui.KeyEventTypeUp, kPhysicalKeyW, kLogicalKeyW, nil, false)
}

func TestEmbedderMultipleCharacters(t *testing.T) {
	r, events := newTestResponder()
	nop := func(bool) {}
	r.handleEvent(ev(nsEventTypeKeyDown, 0, "àn", "àn", false, kKeyCodeKeyA), nop)
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, kPhysicalKeyA, 0x1400000000, s("àn"), false)
	*events = nil
	r.handleEvent(ev(nsEventTypeKeyUp, 0, "a", "a", false, kKeyCodeKeyA), nop)
	checkEvent(t, (*events)[0], ui.KeyEventTypeUp, kPhysicalKeyA, 0x1400000000, nil, false)
}

func TestEmbedderSynthesizeForDuplicateDownEvent(t *testing.T) {
	r, events := newTestResponder()
	lastHandled := true
	r.handleEvent(ev(nsEventTypeKeyDown, 0x100, "a", "a", false, kKeyCodeKeyA), func(h bool) { lastHandled = h })
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, kPhysicalKeyA, kLogicalKeyA, s("a"), false)
	(*events)[0].callback(false)
	if lastHandled {
		t.Error("handled")
	}
	*events = nil
	lastHandled = true
	r.handleEvent(ev(nsEventTypeKeyDown, 0x100, "à", "à", false, kKeyCodeKeyA), func(h bool) { lastHandled = h })
	if len(*events) != 2 {
		t.Fatal(len(*events))
	}
	checkEvent(t, (*events)[0], ui.KeyEventTypeUp, kPhysicalKeyA, kLogicalKeyA, nil, true)
	checkEvent(t, (*events)[1], ui.KeyEventTypeDown, kPhysicalKeyA, 0xE0, s("à"), false)
	(*events)[1].callback(false)
	if lastHandled {
		t.Error("handled")
	}
}

func TestEmbedderIgnoreDuplicateUpEvent(t *testing.T) {
	r, events := newTestResponder()
	lastHandled := false
	r.handleEvent(ev(nsEventTypeKeyUp, 0x100, "a", "a", false, kKeyCodeKeyA), func(h bool) { lastHandled = h })
	if len(*events) != 1 || !lastHandled || (*events)[0].data.Physical != 0 || (*events)[0].data.Logical != 0 {
		t.Errorf("%v %v", *events, lastHandled)
	}
}

func TestEmbedderConvertAbruptRepeatEventsToDown(t *testing.T) {
	r, events := newTestResponder()
	lastHandled := true
	r.handleEvent(ev(nsEventTypeKeyDown, 0x100, "a", "a", true, kKeyCodeKeyA), func(h bool) { lastHandled = h })
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, kPhysicalKeyA, kLogicalKeyA, s("a"), false)
	(*events)[0].callback(false)
	if lastHandled {
		t.Error("handled")
	}
}

func TestEmbedderToggleModifiersDuringKeyTap(t *testing.T) {
	r, events := newTestResponder()
	nop := func(bool) {}
	step := func(e keyEvent, typ ui.KeyEventType, physical, logical int, character *string) {
		t.Helper()
		*events = nil
		r.handleEvent(e, nop)
		if len(*events) != 1 {
			t.Fatalf("%d events", len(*events))
		}
		checkEvent(t, (*events)[0], typ, physical, logical, character, false)
		if (*events)[0].callback != nil {
			(*events)[0].callback(true)
		}
	}
	shiftDown := ev(nsEventTypeFlagsChanged, 0x20104, "", "", false, kKeyCodeShiftRight)
	shiftDown.Timestamp = 123
	step(shiftDown, ui.KeyEventTypeDown, kPhysicalShiftRight, kLogicalShiftRight, nil)
	if (*events)[0].data.TimeStamp != 123*time.Second {
		t.Error("timestamp")
	}
	step(ev(nsEventTypeKeyDown, 0x20104, "A", "A", false, kKeyCodeKeyA), ui.KeyEventTypeDown, kPhysicalKeyA, kLogicalKeyA, s("A"))
	step(ev(nsEventTypeKeyDown, 0x20104, "A", "A", true, kKeyCodeKeyA), ui.KeyEventTypeRepeat, kPhysicalKeyA, kLogicalKeyA, s("A"))
	step(ev(nsEventTypeFlagsChanged, 0x100, "", "", false, kKeyCodeShiftRight), ui.KeyEventTypeUp, kPhysicalShiftRight, kLogicalShiftRight, nil)
	step(ev(nsEventTypeKeyDown, 0x100, "a", "a", true, kKeyCodeKeyA), ui.KeyEventTypeRepeat, kPhysicalKeyA, kLogicalKeyA, s("a"))
	step(ev(nsEventTypeKeyUp, 0x100, "a", "a", false, kKeyCodeKeyA), ui.KeyEventTypeUp, kPhysicalKeyA, kLogicalKeyA, nil)
}

func TestEmbedderConvertCapsLockEvents(t *testing.T) {
	r, events := newTestResponder()
	for _, flags := range []int{0x10100, 0x100} {
		*events = nil
		lastHandled := false
		r.handleEvent(ev(nsEventTypeFlagsChanged, flags, "", "", false, kKeyCodeCapsLock), func(h bool) { lastHandled = h })
		if len(*events) != 2 {
			t.Fatalf("flags %#x: %d events", flags, len(*events))
		}
		checkEvent(t, (*events)[0], ui.KeyEventTypeDown, kPhysicalCapsLock, kLogicalCapsLock, nil, false)
		checkEvent(t, (*events)[1], ui.KeyEventTypeUp, kPhysicalCapsLock, kLogicalCapsLock, nil, true)
		if lastHandled {
			t.Error("handled early")
		}
		(*events)[0].callback(true)
		if !lastHandled {
			t.Error("not handled")
		}
	}
}

func TestEmbedderSynchronizeCapsLockStateOnCapsLock(t *testing.T) {
	r, events := newTestResponder()
	lastHandled := false
	r.handleEvent(ev(nsEventTypeFlagsChanged, 0x100, "", "", false, kKeyCodeCapsLock), func(h bool) { lastHandled = h })
	if len(*events) != 1 || !lastHandled || (*events)[0].data.Physical != 0 || (*events)[0].data.Logical != 0 {
		t.Errorf("%v %v", *events, lastHandled)
	}
}

func TestEmbedderSynchronizeCapsLockStateOnNormalKey(t *testing.T) {
	r, events := newTestResponder()
	lastHandled := false
	r.handleEvent(ev(nsEventTypeKeyDown, 0x10100, "A", "a", false, kKeyCodeKeyA), func(h bool) { lastHandled = h })
	if len(*events) != 3 {
		t.Fatal(len(*events))
	}
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, kPhysicalCapsLock, kLogicalCapsLock, nil, true)
	checkEvent(t, (*events)[1], ui.KeyEventTypeUp, kPhysicalCapsLock, kLogicalCapsLock, nil, true)
	checkEvent(t, (*events)[2], ui.KeyEventTypeDown, kPhysicalKeyA, kLogicalKeyA, s("A"), false)
	if lastHandled {
		t.Error("handled early")
	}
	(*events)[2].callback(true)
	if !lastHandled {
		t.Error("not handled")
	}
}

func TestChannelResponderMessages(t *testing.T) {
	var messages []map[string]any
	var replies []func(map[string]any)
	r := &channelKeyResponder{send: func(m map[string]any, cb func(map[string]any)) {
		messages = append(messages, m)
		replies = append(replies, cb)
	}}
	handled := false
	r.handleEvent(ev(nsEventTypeKeyDown, 0x100, "a", "a", false, kKeyCodeKeyA), func(h bool) { handled = h })
	m := messages[0]
	if m["type"] != "keydown" || m["keymap"] != "macos" || m["keyCode"] != 0 || m["modifiers"] != 0 || m["characters"] != "a" {
		t.Errorf("%v", m)
	}
	replies[0](map[string]any{"handled": true})
	if !handled {
		t.Error("reply")
	}
	// A flags-changed event is a key down when flags grow, up when they
	// shrink, nothing when equal.
	r.handleEvent(ev(nsEventTypeFlagsChanged, 0x20102, "", "", false, 0x38), func(bool) {})
	r.handleEvent(ev(nsEventTypeFlagsChanged, 0x100, "", "", false, 0x38), func(bool) {})
	called := false
	r.handleEvent(ev(nsEventTypeFlagsChanged, 0x100, "", "", false, 0x38), func(h bool) { called = h })
	if messages[1]["type"] != "keydown" || messages[1]["modifiers"] != 0x20002 || messages[2]["type"] != "keyup" || !called || len(messages) != 3 {
		t.Errorf("%v %v", messages, called)
	}
	if _, ok := messages[1]["characters"]; ok {
		t.Error("flags changed has characters")
	}
	// A null reply counts as handled.
	nullReplyCheck(t, r, &replies)
}

// The embedder's key code maps (KeyCodeMap.g.mm in the Flutter clone) are
// the framework's macOS maps the port uses.
func TestKeyCodeMapsMatchTheEngine(t *testing.T) {
	src, err := os.ReadFile(os.ExpandEnv("$HOME/Projetos/jonathanperis/flutter/engine/src/flutter/shell/platform/darwin/macos/framework/Source/KeyCodeMap.g.mm"))
	if err != nil {
		t.Skip(err)
	}
	entry := regexp.MustCompile(`@0x([0-9a-f]+) : @0x([0-9a-f]+),`)
	section := func(name string) map[int]int {
		text := string(src)
		text = text[strings.Index(text, name):]
		text = text[:strings.Index(text, "};")]
		m := map[int]int{}
		for _, e := range entry.FindAllStringSubmatch(text, -1) {
			k, _ := strconv.ParseInt(e[1], 16, 64)
			v, _ := strconv.ParseInt(e[2], 16, 64)
			m[int(k)] = int(v)
		}
		return m
	}
	physical := section("keyCodeToPhysicalKey = @{")
	logical := section("keyCodeToLogicalKey = @{")
	if len(physical) == 0 || len(logical) == 0 {
		t.Fatal("maps not found")
	}
	for k, v := range physical {
		if got := physicalKeyForKeyCode(k); got != v {
			t.Errorf("physical %#x: %#x, engine %#x", k, got, v)
		}
	}
	n := 0
	for k := 0; k < 0x100; k++ {
		got, ok := keyCodeToLogicalKey(k)
		want, wok := logical[k]
		if ok != wok || got != want {
			t.Errorf("logical %#x: %#x %v, engine %#x %v", k, got, ok, want, wok)
		}
		if ok {
			n++
		}
	}
	if n != len(logical) {
		t.Errorf("%d logical entries, engine %d", n, len(logical))
	}
}

// Own tests: Latin-1 upper-case labels give lower-case logical keys, and
// function-key characters (U+F700..U+F7FF) are not sent as characters.
func TestEmbedderLatin1AndFunctionKeys(t *testing.T) {
	r, events := newTestResponder()
	r.handleEvent(ev(nsEventTypeKeyDown, 0x100, "É", "É", false, 0x0e), func(bool) {})
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, services.PhysicalKeyboardKeyKeyE.UsbHidUsage, 0xe9, s("É"), false)
	*events = nil
	r.handleEvent(ev(nsEventTypeKeyDown, 0x100, "", "", false, 0x7b), func(bool) {})
	checkEvent(t, (*events)[0], ui.KeyEventTypeDown, services.PhysicalKeyboardKeyArrowLeft.UsbHidUsage,
		services.LogicalKeyboardKeyArrowLeft.KeyID, nil, false)
}

func nullReplyCheck(t *testing.T, r *channelKeyResponder, replies *[]func(map[string]any)) {
	handled := false
	r.handleEvent(ev(nsEventTypeKeyUp, 0, "a", "a", false, kKeyCodeKeyA), func(h bool) { handled = h })
	(*replies)[len(*replies)-1](nil)
	if !handled {
		t.Error("null reply")
	}
}
