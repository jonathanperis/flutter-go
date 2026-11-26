package app

// GLFW's key events as the NSEvents the macOS embedder's key responders
// read (keyboard.go), delivered to the framework as Flutter's macOS
// embedder does: key data on flutter/keydata, then the raw key message on
// flutter/keyevent.
//
// GLFW gives, for a key: the macOS virtual key code (its scancode on
// macOS), the action (press, repeat, release) and modifiers without sides;
// modifier keys and CapsLock arrive as key events (macOS's flagsChanged).
// What NSEvent has and GLFW doesn't is rebuilt:
//   - modifierFlags: the sided device-dependent bits are tracked from the
//     modifier keys' own events, the device-independent bits derived from
//     them, CapsLock from GLFW's lock-key modifier;
//   - charactersIgnoringModifiers: glfwGetKeyName, the layout's character of
//     the key without modifiers (macOS's keeps Shift, but the embedder
//     lower-cases it anyway);
//   - characters: what GLFW's char callback reports right after the key
//     press (a key event is held until then, or until the next event); with
//     Command held, GLFW reports no character, and the key's name is used
//     (shifted if Shift is held), as macOS would give; with Control, the
//     ASCII control character of a letter.

import (
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/textedit"
)

// glfwModifierKeys are GLFW's modifier keys with the macOS key code and
// sided flag they stand for.
var glfwModifierKeys = map[glfw.Key]int{
	glfw.KeyLeftShift: kModifierFlagShiftLeft, glfw.KeyRightShift: kModifierFlagShiftRight,
	glfw.KeyLeftControl: kModifierFlagControlLeft, glfw.KeyRightControl: kModifierFlagControlRight,
	glfw.KeyLeftAlt: kModifierFlagAltLeft, glfw.KeyRightAlt: kModifierFlagAltRight,
	glfw.KeyLeftSuper: kModifierFlagMetaLeft, glfw.KeyRightSuper: kModifierFlagMetaRight,
}

// glfwKeyboard turns GLFW key and char callbacks into keyEvents for the
// keyboard manager.
type glfwKeyboard struct {
	manager    *keyboardManager
	sidedFlags int
	capsLock   bool
	pending    *keyEvent
}

// independentFlags derives NSEvent's device-independent modifier bits from
// the sided ones.
func independentFlags(sided int) int {
	flags := sided
	if sided&(kModifierFlagShiftLeft|kModifierFlagShiftRight) != 0 {
		flags |= nsModifierFlagShift
	}
	if sided&(kModifierFlagControlLeft|kModifierFlagControlRight) != 0 {
		flags |= nsModifierFlagControl
	}
	if sided&(kModifierFlagAltLeft|kModifierFlagAltRight) != 0 {
		flags |= nsModifierFlagOption
	}
	if sided&(kModifierFlagMetaLeft|kModifierFlagMetaRight) != 0 {
		flags |= nsModifierFlagCommand
	}
	return flags
}

func (k *glfwKeyboard) modifierFlags() int {
	flags := independentFlags(k.sidedFlags) | 0x100 // NSEvent's non-coalesced bit
	if k.capsLock {
		flags |= nsModifierFlagCapsLock
	}
	return flags
}

func (k *glfwKeyboard) flush() {
	if k.pending != nil {
		e := *k.pending
		k.pending = nil
		k.manager.handleEvent(e, nil)
	}
}

func (k *glfwKeyboard) onKey(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	k.flush()
	now := glfw.GetTime()
	k.capsLock = mods&glfw.ModCapsLock != 0
	if flag, ok := glfwModifierKeys[key]; ok {
		if action == glfw.Release {
			k.sidedFlags &^= flag
		} else {
			k.sidedFlags |= flag
		}
		k.manager.handleEvent(keyEvent{Type: nsEventTypeFlagsChanged, KeyCode: scancode, ModifierFlags: k.modifierFlags(), Timestamp: now}, nil)
		return
	}
	if key == glfw.KeyCapsLock {
		// macOS sends one flagsChanged per toggle: GLFW reports it as a press
		// when CapsLock turns on and a release when it turns off.
		k.capsLock = action != glfw.Release
		k.manager.handleEvent(keyEvent{Type: nsEventTypeFlagsChanged, KeyCode: scancode, ModifierFlags: k.modifierFlags(), Timestamp: now}, nil)
		return
	}
	name := glfw.GetKeyName(key, scancode)
	e := keyEvent{KeyCode: scancode, ModifierFlags: k.modifierFlags(), CharactersIgnoringModifiers: name, Characters: name, Timestamp: now}
	if action == glfw.Release {
		e.Type = nsEventTypeKeyUp
		k.manager.handleEvent(e, nil)
		return
	}
	e.Type = nsEventTypeKeyDown
	e.IsARepeat = action == glfw.Repeat
	e.Characters = ""
	switch {
	case mods&glfw.ModSuper != 0 && name != "":
		e.Characters = name
		if mods&glfw.ModShift != 0 {
			e.Characters = dart.StringToUpperCase(name)
		}
	case mods&glfw.ModControl != 0 && len(name) == 1 && name[0] >= 'a' && name[0] <= 'z':
		e.Characters = string(rune(name[0] - 'a' + 1))
	}
	k.pending = &e
}

// onChar adds GLFW's committed character to the key down it follows.
func (k *glfwKeyboard) onChar(ch rune) {
	if k.pending != nil && k.pending.Type == nsEventTypeKeyDown {
		k.pending.Characters += string(ch)
	}
}

// platformResponse adapts a callback to the engine's platform message
// response.
type platformResponse func(data []byte)

func (f platformResponse) Complete(data []byte) { f(data) }
func (f platformResponse) CompleteEmpty()       { f(nil) }

// newKeyboardManager connects the responders to the framework through the
// engine's platform messages.
func newKeyboardManager(pc *ui.PlatformConfiguration) *keyboardManager {
	embedder := newEmbedderKeyResponder(func(data ui.KeyData, callback func(bool)) {
		var response ui.PlatformMessageResponse
		if callback != nil {
			response = platformResponse(func(reply []byte) { callback(len(reply) > 0 && reply[0] == 1) })
		}
		pc.DispatchPlatformMessage(&ui.PlatformMessage{Channel: "flutter/keydata", Data: ui.PackKeyDataPacket(data), Response: response})
	})
	codec := services.JSONMessageCodec{}
	channel := &channelKeyResponder{send: func(message map[string]any, callback func(map[string]any)) {
		pc.DispatchPlatformMessage(&ui.PlatformMessage{Channel: "flutter/keyevent", Data: codec.EncodeMessage(message),
			Response: platformResponse(func(reply []byte) {
				if reply == nil {
					callback(nil)
					return
				}
				m, _ := codec.DecodeMessage(reply).(map[string]any)
				callback(m)
			})})
	}}
	return &keyboardManager{embedder: embedder, channel: channel}
}

// legacyKeys maps logical keys to the interim text-editing model's keys.
var legacyKeys = map[services.LogicalKeyboardKey]textedit.Key{
	services.LogicalKeyboardKeyBackspace: textedit.KeyBackspace, services.LogicalKeyboardKeyDelete: textedit.KeyDelete,
	services.LogicalKeyboardKeyArrowLeft: textedit.KeyLeft, services.LogicalKeyboardKeyArrowRight: textedit.KeyRight,
	services.LogicalKeyboardKeyArrowUp: textedit.KeyUp, services.LogicalKeyboardKeyArrowDown: textedit.KeyDown,
	services.LogicalKeyboardKeyHome: textedit.KeyHome, services.LogicalKeyboardKeyEnd: textedit.KeyEnd,
	services.LogicalKeyboardKeyEnter: textedit.KeyEnter, services.LogicalKeyboardKeyNumpadEnter: textedit.KeyEnter,
	services.LogicalKeyboardKeyTab: textedit.KeyTab, services.LogicalKeyboardKeyKeyA: textedit.KeyA,
	services.LogicalKeyboardKeyKeyC: textedit.KeyC, services.LogicalKeyboardKeyKeyV: textedit.KeyV,
	services.LogicalKeyboardKeyKeyX: textedit.KeyX,
}

// legacyKeyMessageHandler is a temporary KeyEventManager.keyMessageHandler
// (until widgets' focus and shortcuts are ported): it hands key downs and
// repeats to the focused widget as the interim model's key events.
func legacyKeyMessageHandler(dispatch func(textedit.KeyEvent) bool) services.KeyMessageHandler {
	return func(message *services.KeyMessage) bool {
		handled := false
		for _, e := range message.Events {
			switch e.(type) {
			case *services.KeyDownEvent, *services.KeyRepeatEvent:
			default:
				continue
			}
			key, ok := legacyKeys[e.LogicalKey()]
			if !ok {
				continue
			}
			hk := services.HardwareKeyboardInstance()
			if dispatch(textedit.KeyEvent{Key: key, Shift: hk.IsShiftPressed(), Ctrl: hk.IsControlPressed(),
				Alt: hk.IsAltPressed(), Meta: hk.IsMetaPressed()}) {
				handled = true
			}
		}
		return handled
	}
}

var _ = time.Second
