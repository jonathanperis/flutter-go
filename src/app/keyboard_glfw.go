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
	"github.com/go-gl/glfw/v3.3/glfw"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/dart"
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
	// host receives the key downs the framework did not handle.
	host textInputHost
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
		var onUnhandled func()
		if e.Type == nsEventTypeKeyDown && k.host.focused != nil {
			onUnhandled = func() { k.host.handleUnhandled(e) }
		}
		k.manager.handleEvent(e, onUnhandled)
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
	channel := &channelKeyResponder{send: func(message map[string]any, callback func(*dart.LinkedMap)) {
		pc.DispatchPlatformMessage(&ui.PlatformMessage{Channel: "flutter/keyevent", Data: codec.EncodeMessage(message),
			Response: platformResponse(func(reply []byte) {
				if reply == nil {
					callback(nil)
					return
				}
				m, _ := codec.DecodeMessage(reply).(*dart.LinkedMap)
				callback(m)
			})})
	}}
	return &keyboardManager{embedder: embedder, channel: channel}
}

// textInputHost stands for the macOS embedder's text input plugin
// (FlutterTextInputPlugin) receiving a key down the framework did not
// handle (interpretKeyEvents:): a key bound by NSStandardKeyBindingResponding
// becomes its selectors (TextInputClient.performSelectors; insertNewline:
// on a single-line field is performAction), other characters are inserted
// (insertText:). ENGINE GAP: no NSTextInputContext (no IME composition,
// no user key bindings); the bindings are the standard ones, as
// flutter_test's MacOSTestTextInputKeyHandler lists them.
type textInputHost struct {
	focused func() widgets.KeyTarget
}

type macBinding struct {
	key                       services.LogicalKeyboardKey
	shift, alt, meta, control bool
}

var macStandardKeyBindings = func() map[macBinding][]string {
	m := map[macBinding][]string{}
	add := func(key services.LogicalKeyboardKey, mods string, selectors ...string) {
		b := macBinding{key: key}
		for _, c := range mods {
			switch c {
			case 's':
				b.shift = true
			case 'a':
				b.alt = true
			case 'm':
				b.meta = true
			case 'c':
				b.control = true
			}
		}
		m[b] = selectors
	}
	for _, s := range []string{"s", ""} {
		add(services.LogicalKeyboardKeyBackspace, s, "deleteBackward:")
		add(services.LogicalKeyboardKeyBackspace, s+"a", "deleteWordBackward:")
		add(services.LogicalKeyboardKeyBackspace, s+"m", "deleteToBeginningOfLine:")
		add(services.LogicalKeyboardKeyBackspace, s+"c", "deleteBackwardByDecomposingPreviousCharacter:")
		add(services.LogicalKeyboardKeyDelete, s, "deleteForward:")
		add(services.LogicalKeyboardKeyDelete, s+"a", "deleteWordForward:")
		add(services.LogicalKeyboardKeyDelete, s+"m", "deleteToEndOfLine:")
	}
	L, R, U, D := services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight,
		services.LogicalKeyboardKeyArrowUp, services.LogicalKeyboardKeyArrowDown
	add(L, "", "moveLeft:")
	add(R, "", "moveRight:")
	add(U, "", "moveUp:")
	add(D, "", "moveDown:")
	add(L, "s", "moveLeftAndModifySelection:")
	add(R, "s", "moveRightAndModifySelection:")
	add(U, "s", "moveUpAndModifySelection:")
	add(D, "s", "moveDownAndModifySelection:")
	add(L, "a", "moveWordLeft:")
	add(R, "a", "moveWordRight:")
	add(U, "a", "moveBackward:", "moveToBeginningOfParagraph:")
	add(D, "a", "moveForward:", "moveToEndOfParagraph:")
	add(L, "as", "moveWordLeftAndModifySelection:")
	add(R, "as", "moveWordRightAndModifySelection:")
	add(U, "as", "moveParagraphBackwardAndModifySelection:")
	add(D, "as", "moveParagraphForwardAndModifySelection:")
	add(L, "m", "moveToLeftEndOfLine:")
	add(R, "m", "moveToRightEndOfLine:")
	add(U, "m", "moveToBeginningOfDocument:")
	add(D, "m", "moveToEndOfDocument:")
	add(L, "ms", "moveToLeftEndOfLineAndModifySelection:")
	add(R, "ms", "moveToRightEndOfLineAndModifySelection:")
	add(U, "ms", "moveToBeginningOfDocumentAndModifySelection:")
	add(D, "ms", "moveToEndOfDocumentAndModifySelection:")
	add(services.LogicalKeyboardKeyKeyA, "cs", "moveToBeginningOfParagraphAndModifySelection:")
	add(services.LogicalKeyboardKeyKeyA, "c", "moveToBeginningOfParagraph:")
	add(services.LogicalKeyboardKeyKeyB, "cs", "moveBackwardAndModifySelection:")
	add(services.LogicalKeyboardKeyKeyB, "c", "moveBackward:")
	add(services.LogicalKeyboardKeyKeyE, "cs", "moveToEndOfParagraphAndModifySelection:")
	add(services.LogicalKeyboardKeyKeyE, "c", "moveToEndOfParagraph:")
	add(services.LogicalKeyboardKeyKeyF, "cs", "moveForwardAndModifySelection:")
	add(services.LogicalKeyboardKeyKeyF, "c", "moveForward:")
	add(services.LogicalKeyboardKeyKeyN, "c", "moveDown:")
	add(services.LogicalKeyboardKeyKeyN, "cs", "moveDownAndModifySelection:")
	add(services.LogicalKeyboardKeyKeyP, "c", "moveUp:")
	add(services.LogicalKeyboardKeyKeyP, "cs", "moveUpAndModifySelection:")
	add(services.LogicalKeyboardKeyKeyT, "c", "transpose:")
	add(services.LogicalKeyboardKeyHome, "", "scrollToBeginningOfDocument:")
	add(services.LogicalKeyboardKeyEnd, "", "scrollToEndOfDocument:")
	add(services.LogicalKeyboardKeyHome, "s", "moveToBeginningOfDocumentAndModifySelection:")
	add(services.LogicalKeyboardKeyEnd, "s", "moveToEndOfDocumentAndModifySelection:")
	add(services.LogicalKeyboardKeyPageUp, "", "scrollPageUp:")
	add(services.LogicalKeyboardKeyPageDown, "", "scrollPageDown:")
	add(services.LogicalKeyboardKeyPageUp, "s", "pageUpAndModifySelection:")
	add(services.LogicalKeyboardKeyPageDown, "s", "pageDownAndModifySelection:")
	add(services.LogicalKeyboardKeyEscape, "", "cancelOperation:")
	add(services.LogicalKeyboardKeyEnter, "", "insertNewline:")
	add(services.LogicalKeyboardKeyNumpadEnter, "", "insertNewline:")
	add(services.LogicalKeyboardKeyTab, "", "insertTab:")
	add(services.LogicalKeyboardKeyTab, "s", "insertBacktab:")
	return m
}()

// handleUnhandled is the text input plugin's handling of an unhandled key
// down.
func (h textInputHost) handleUnhandled(e keyEvent) {
	target := h.focused()
	client, ok := target.(services.TextInputClient)
	if !ok {
		return
	}
	b := macBinding{shift: e.ModifierFlags&nsModifierFlagShift != 0, alt: e.ModifierFlags&nsModifierFlagOption != 0,
		meta: e.ModifierFlags&nsModifierFlagCommand != 0, control: e.ModifierFlags&nsModifierFlagControl != 0}
	if key, ok := services.KMacOsToLogicalKey[e.KeyCode]; ok {
		b.key = key
	} else if len(e.CharactersIgnoringModifiers) > 0 {
		b.key = services.LogicalKeyboardKey{KeyID: int([]rune(e.CharactersIgnoringModifiers)[0])}
	}
	if selectors, ok := macStandardKeyBindings[b]; ok {
		for _, selector := range selectors {
			if selector == "insertNewline:" {
				client.PerformAction(services.TextInputActionDone)
				continue
			}
			client.PerformSelector(selector)
		}
		return
	}
	if b.meta || b.control || e.Characters == "" {
		return
	}
	for _, r := range e.Characters {
		if r < 0x20 || r == 0x7f || (r >= 0xF700 && r <= 0xF8FF) { // control and function-key characters
			return
		}
	}
	target.InsertText(e.Characters)
}
