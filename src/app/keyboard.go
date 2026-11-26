package app

// The macOS embedder's key handling, ported from the engine of Flutter
// 3.50.1 (shell/platform/darwin/macos/framework/Source):
// FlutterEmbedderKeyResponder (NSEvents → key data on flutter/keydata, with
// the synthesized events that keep modifiers and CapsLock in sync) and
// FlutterChannelKeyResponder (NSEvents → raw "keymap: macos" messages on
// flutter/keyevent), the two primary responders FlutterKeyboardManager
// sends every event to. The window toolkit (GLFW, see app.go) is turned into
// the NSEvent fields these use (keyEvent).
//
// Not ported: FlutterKeyboardLayout's layout map (logical keys of printable
// keys from the current keyboard layout's "layout goals", which needs
// UCKeyTranslate over the input source); the logical key of a printable key
// comes from charactersIgnoringModifiers, as the embedder does when the map
// has no entry. Nor the text input plugin the manager redispatches
// unhandled events to (ENGINE GAP: committed text reaches widgets through
// GLFW's char callback).

import (
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// NSEvent types the responders handle.
type nsEventType int

const (
	nsEventTypeKeyDown nsEventType = iota
	nsEventTypeKeyUp
	nsEventTypeFlagsChanged
)

// NSEventModifierFlags.
const (
	nsModifierFlagCapsLock = 1 << 16
	nsModifierFlagShift    = 1 << 17
	nsModifierFlagControl  = 1 << 18
	nsModifierFlagOption   = 1 << 19
	nsModifierFlagCommand  = 1 << 20
)

// The device-dependent (sided) modifier flags (KeyCodeMap_Internal.h).
const (
	kModifierFlagControlLeft  = 0x1
	kModifierFlagShiftLeft    = 0x2
	kModifierFlagShiftRight   = 0x4
	kModifierFlagMetaLeft     = 0x8
	kModifierFlagMetaRight    = 0x10
	kModifierFlagAltLeft      = 0x20
	kModifierFlagAltRight     = 0x40
	kModifierFlagControlRight = 0x200
)

// keyCodeToModifierFlag and modifierFlagToKeyCode (KeyCodeMap.g.mm).
var keyCodeToModifierFlag = map[int]int{
	0x38: kModifierFlagShiftLeft, 0x3c: kModifierFlagShiftRight,
	0x3b: kModifierFlagControlLeft, 0x3e: kModifierFlagControlRight,
	0x3a: kModifierFlagAltLeft, 0x3d: kModifierFlagAltRight,
	0x37: kModifierFlagMetaLeft, 0x36: kModifierFlagMetaRight,
}

var modifierFlagToKeyCode = map[int]int{
	kModifierFlagShiftLeft: 0x38, kModifierFlagShiftRight: 0x3c,
	kModifierFlagControlLeft: 0x3b, kModifierFlagControlRight: 0x3e,
	kModifierFlagAltLeft: 0x3a, kModifierFlagAltRight: 0x3d,
	kModifierFlagMetaLeft: 0x37, kModifierFlagMetaRight: 0x36,
}

// modifierFlagToKeyCodeOrder is the order NSDictionary enumerates
// modifierFlagToKeyCode in (by key, as its small-integer keys hash).
var modifierFlagToKeyCodeOrder = []int{kModifierFlagControlLeft, kModifierFlagShiftLeft, kModifierFlagShiftRight,
	kModifierFlagMetaLeft, kModifierFlagMetaRight, kModifierFlagAltLeft, kModifierFlagAltRight, kModifierFlagControlRight}

const (
	kCapsLockPhysicalKey = 0x00070039
	kCapsLockLogicalKey  = 0x100000104
	kValueMask           = 0x000ffffffff
	kUnicodePlane        = 0x00000000000
	kMacosPlane          = 0x01400000000
)

// keyEvent is the part of an NSEvent the responders read.
type keyEvent struct {
	Type                        nsEventType
	KeyCode                     int
	ModifierFlags               int
	Characters                  string
	CharactersIgnoringModifiers string
	IsARepeat                   bool
	// Timestamp is NSEvent.timestamp (seconds).
	Timestamp float64
}

// keyCodeToLogicalKey is the embedder's map: the framework's macOS map of
// keys that can't be derived from their characters, plus the numpad's.
func keyCodeToLogicalKey(keyCode int) (int, bool) {
	if k, ok := services.KMacOsToLogicalKey[keyCode]; ok {
		return k.KeyID, true
	}
	if k, ok := services.KMacOsNumPadMap[keyCode]; ok {
		return k.KeyID, true
	}
	return 0, false
}

func lowestSetBit(bitmask int) int { return bitmask & -bitmask }

func keyOfPlane(baseKey, plane int) int { return plane | (baseKey & kValueMask) }

func physicalKeyForKeyCode(keyCode int) int {
	if k, ok := services.KMacOsToPhysicalKey[keyCode]; ok {
		return k.UsbHidUsage
	}
	return keyOfPlane(keyCode, kMacosPlane)
}

func logicalKeyForModifier(keyCode, hidCode int) int {
	if k, ok := keyCodeToLogicalKey(keyCode); ok {
		return k
	}
	return keyOfPlane(hidCode, kMacosPlane)
}

// toLower lower-cases ASCII and Latin-1 letters, independent of locale.
func toLower(n int) int {
	const (
		lowerA, upperA, upperZ                      = 0x61, 0x41, 0x5a
		lowerAGrave, upperAGrave, upperThorn, divis = 0xe0, 0xc0, 0xde, 0xf7
	)
	if n >= upperA && n <= upperZ {
		return n - upperA + lowerA
	}
	if n >= upperAGrave && n <= upperThorn && n != divis {
		return n - upperAGrave + lowerAGrave
	}
	return n
}

// decodeUTF16 returns the code points of s (as the embedder's DecodeUtf16).
func decodeUTF16(s string) []int {
	var out []int
	high := 0
	for _, u := range dart.StringCodeUnits(s) {
		c := int(u)
		switch {
		case c <= 0xD7FF || c >= 0xE000:
			out = append(out, c)
		case c <= 0xDBFF:
			high = c - 0xD800
		default:
			out = append(out, (high<<10)+(c-0xDC00)+0x10000)
		}
	}
	return out
}

// logicalKeyForEvent is GetLogicalKeyForEvent (key down/up events).
func logicalKeyForEvent(event keyEvent) int {
	if k, ok := keyCodeToLogicalKey(event.KeyCode); ok {
		return k
	}
	character := 0
	if event.CharactersIgnoringModifiers != "" {
		if keyLabel := decodeUTF16(event.CharactersIgnoringModifiers); len(keyLabel) == 1 {
			character = keyLabel[0]
		}
	}
	if character != 0 {
		return keyOfPlane(toLower(character), kUnicodePlane)
	}
	return keyOfPlane(event.KeyCode, kMacosPlane)
}

// flutterTimestamp converts NSEvent.timestamp to microseconds.
func flutterTimestamp(timestamp float64) time.Duration {
	return time.Duration(timestamp*1000000.0) * time.Microsecond
}

// eventString is getEventString: nil for no characters or a function key.
func eventString(characters string) *string {
	if characters == "" {
		return nil
	}
	if u := dart.CodeUnitAt(characters, 0); u >= 0xf700 && u <= 0xf7ff {
		return nil
	}
	return &characters
}

// keyCallbackGuard is FlutterKeyCallbackGuard: the event's callback,
// handled exactly once (pended or resolved).
type keyCallbackGuard struct {
	callback      func(handled bool)
	handled       bool
	sentAnyEvents bool
}

func (g *keyCallbackGuard) pend() func(bool) {
	if g.handled {
		panic("keyboard: the callback has been handled")
	}
	g.handled = true
	return g.callback
}

func (g *keyCallbackGuard) resolve(handled bool) {
	if g.handled {
		panic("keyboard: the callback has been handled")
	}
	g.callback(handled)
	g.handled = true
}

// embedderKeyResponder is FlutterEmbedderKeyResponder: it converts events
// to key data, sent with sendEvent (whose callback, nil for synthesized
// events, receives the framework's answer).
type embedderKeyResponder struct {
	sendEvent                   func(data ui.KeyData, callback func(handled bool))
	pressingRecords             map[int]int
	modifierFlagOfInterestMask  int
	lastModifierFlagsOfInterest int
}

func newEmbedderKeyResponder(sendEvent func(ui.KeyData, func(bool))) *embedderKeyResponder {
	mask := nsModifierFlagCapsLock
	for _, flag := range keyCodeToModifierFlag {
		mask |= flag
	}
	return &embedderKeyResponder{sendEvent: sendEvent, pressingRecords: map[int]int{}, modifierFlagOfInterestMask: mask}
}

func (r *embedderKeyResponder) handleEvent(event keyEvent, callback func(handled bool)) {
	guard := &keyCallbackGuard{callback: callback}
	switch event.Type {
	case nsEventTypeKeyDown:
		r.handleDownEvent(event, guard)
	case nsEventTypeKeyUp:
		r.handleUpEvent(event, guard)
	case nsEventTypeFlagsChanged:
		r.handleFlagEvent(event, guard)
	}
	if !guard.handled {
		panic("keyboard: the callback is returned without being handled")
	}
	if !guard.sentAnyEvents {
		// An empty event, so the framework infers its transit mode.
		r.sendEvent(ui.KeyData{Type: ui.KeyEventTypeDown}, nil)
	}
}

func (r *embedderKeyResponder) synchronizeModifiers(currentFlags, ignoringFlags int, timestamp float64, guard *keyCallbackGuard) {
	updatingMask := r.modifierFlagOfInterestMask &^ ignoringFlags
	currentFlagsOfInterest := currentFlags & updatingMask
	lastFlagsOfInterest := r.lastModifierFlagsOfInterest & updatingMask
	flagDifference := currentFlagsOfInterest ^ lastFlagsOfInterest
	if flagDifference&nsModifierFlagCapsLock != 0 {
		r.sendCapsLockTap(timestamp, true, guard)
		flagDifference &^= nsModifierFlagCapsLock
	}
	for {
		currentFlag := lowestSetBit(flagDifference)
		if currentFlag == 0 {
			break
		}
		flagDifference &^= currentFlag
		keyCode, ok := modifierFlagToKeyCode[currentFlag]
		if !ok {
			continue
		}
		isDownEvent := currentFlagsOfInterest&currentFlag != 0
		r.sendModifierEvent(isDownEvent, timestamp, keyCode, true, guard)
	}
	r.lastModifierFlagsOfInterest = (r.lastModifierFlagsOfInterest &^ updatingMask) | currentFlagsOfInterest
}

func (r *embedderKeyResponder) updateKey(physicalKey, logicalKey int) {
	if logicalKey == 0 {
		delete(r.pressingRecords, physicalKey)
	} else {
		r.pressingRecords[physicalKey] = logicalKey
	}
}

func (r *embedderKeyResponder) sendPrimary(data ui.KeyData, guard *keyCallbackGuard) {
	callback := guard.pend()
	r.sendEvent(data, callback)
	guard.sentAnyEvents = true
}

func (r *embedderKeyResponder) sendSynthesized(data ui.KeyData, guard *keyCallbackGuard) {
	r.sendEvent(data, nil)
	guard.sentAnyEvents = true
}

func (r *embedderKeyResponder) sendCapsLockTap(timestamp float64, synthesizeDown bool, guard *keyCallbackGuard) {
	data := ui.KeyData{TimeStamp: flutterTimestamp(timestamp), Type: ui.KeyEventTypeDown,
		Physical: kCapsLockPhysicalKey, Logical: kCapsLockLogicalKey, Synthesized: synthesizeDown}
	if !synthesizeDown {
		r.sendPrimary(data, guard)
	} else {
		r.sendSynthesized(data, guard)
	}
	data.Type = ui.KeyEventTypeUp
	data.Synthesized = true
	r.sendSynthesized(data, guard)
}

func (r *embedderKeyResponder) sendModifierEvent(isDownEvent bool, timestamp float64, keyCode int, synthesized bool, guard *keyCallbackGuard) {
	physicalKey := physicalKeyForKeyCode(keyCode)
	logicalKey := logicalKeyForModifier(keyCode, physicalKey)
	if physicalKey == 0 || logicalKey == 0 {
		guard.resolve(true)
		return
	}
	typ := ui.KeyEventTypeUp
	pressed := 0
	if isDownEvent {
		typ, pressed = ui.KeyEventTypeDown, logicalKey
	}
	data := ui.KeyData{TimeStamp: flutterTimestamp(timestamp), Type: typ, Physical: physicalKey, Logical: logicalKey,
		Synthesized: synthesized}
	r.updateKey(physicalKey, pressed)
	if !synthesized {
		r.sendPrimary(data, guard)
	} else {
		r.sendSynthesized(data, guard)
	}
}

func (r *embedderKeyResponder) handleDownEvent(event keyEvent, guard *keyCallbackGuard) {
	physicalKey := physicalKeyForKeyCode(event.KeyCode)
	logicalKey := logicalKeyForEvent(event)
	r.synchronizeModifiers(event.ModifierFlags, 0, event.Timestamp, guard)
	pressedLogicalKey, pressed := r.pressingRecords[physicalKey]
	if pressed && !event.IsARepeat {
		// The key is pressed again without an up event in between: release
		// it first (a missed up event, e.g. while the window was inactive).
		r.sendSynthesized(ui.KeyData{TimeStamp: flutterTimestamp(event.Timestamp), Type: ui.KeyEventTypeUp,
			Physical: physicalKey, Logical: pressedLogicalKey, Synthesized: true}, guard)
		pressed = false
	}
	if !pressed {
		r.updateKey(physicalKey, logicalKey)
	}
	data := ui.KeyData{TimeStamp: flutterTimestamp(event.Timestamp), Physical: physicalKey,
		Character: eventString(event.Characters)}
	if !pressed {
		data.Type, data.Logical = ui.KeyEventTypeDown, logicalKey
	} else {
		data.Type, data.Logical = ui.KeyEventTypeRepeat, pressedLogicalKey
	}
	r.sendPrimary(data, guard)
}

func (r *embedderKeyResponder) handleUpEvent(event keyEvent, guard *keyCallbackGuard) {
	r.synchronizeModifiers(event.ModifierFlags, 0, event.Timestamp, guard)
	physicalKey := physicalKeyForKeyCode(event.KeyCode)
	pressedLogicalKey, pressed := r.pressingRecords[physicalKey]
	if !pressed {
		// The key was released while the window was inactive (its down
		// event was missed): nothing to release.
		guard.resolve(true)
		return
	}
	r.updateKey(physicalKey, 0)
	r.sendPrimary(ui.KeyData{TimeStamp: flutterTimestamp(event.Timestamp), Type: ui.KeyEventTypeUp,
		Physical: physicalKey, Logical: pressedLogicalKey}, guard)
}

func (r *embedderKeyResponder) handleCapsLockEvent(event keyEvent, guard *keyCallbackGuard) {
	r.synchronizeModifiers(event.ModifierFlags, nsModifierFlagCapsLock, event.Timestamp, guard)
	if r.lastModifierFlagsOfInterest&nsModifierFlagCapsLock != event.ModifierFlags&nsModifierFlagCapsLock {
		r.sendCapsLockTap(event.Timestamp, false, guard)
		r.lastModifierFlagsOfInterest ^= nsModifierFlagCapsLock
	} else {
		guard.resolve(true)
	}
}

func (r *embedderKeyResponder) handleFlagEvent(event keyEvent, guard *keyCallbackGuard) {
	targetModifierFlag := keyCodeToModifierFlag[event.KeyCode]
	targetKey := physicalKeyForKeyCode(event.KeyCode)
	if targetKey == kCapsLockPhysicalKey {
		r.handleCapsLockEvent(event, guard)
		return
	}
	r.synchronizeModifiers(event.ModifierFlags, targetModifierFlag, event.Timestamp, guard)
	_, lastTargetPressed := r.pressingRecords[targetKey]
	shouldBePressed := event.ModifierFlags&targetModifierFlag != 0
	if lastTargetPressed == shouldBePressed {
		guard.resolve(true)
		return
	}
	r.lastModifierFlagsOfInterest ^= targetModifierFlag
	r.sendModifierEvent(shouldBePressed, event.Timestamp, event.KeyCode, false, guard)
}

// channelKeyResponder is FlutterChannelKeyResponder: it sends each event as
// a raw key message (the "keymap: macos" map) with send, whose callback
// gets the framework's reply.
type channelKeyResponder struct {
	send                   func(message map[string]any, callback func(reply *dart.LinkedMap))
	previouslyPressedFlags int
}

func (r *channelKeyResponder) handleEvent(event keyEvent, callback func(handled bool)) {
	modifierFlags := event.ModifierFlags &^ 0x100
	var typ string
	switch event.Type {
	case nsEventTypeKeyDown:
		typ = "keydown"
	case nsEventTypeKeyUp:
		typ = "keyup"
	default:
		switch {
		case modifierFlags < r.previouslyPressedFlags:
			typ = "keyup"
		case modifierFlags > r.previouslyPressedFlags:
			typ = "keydown"
		default:
			callback(true)
			return
		}
	}
	r.previouslyPressedFlags = modifierFlags
	message := map[string]any{"keymap": "macos", "type": typ, "keyCode": event.KeyCode, "modifiers": modifierFlags}
	if event.Type == nsEventTypeKeyDown || event.Type == nsEventTypeKeyUp {
		message["characters"] = event.Characters
		message["charactersIgnoringModifiers"] = event.CharactersIgnoringModifiers
	}
	r.send(message, func(reply *dart.LinkedMap) {
		if reply == nil {
			callback(true)
			return
		}
		handled, _ := reply.Lookup("handled").(bool)
		callback(handled)
	})
}

// keyboardManager is FlutterKeyboardManager's primary-responder part: an
// event goes to both responders; onUnhandled runs if neither handled it
// (the embedder then hands it to the text input plugin).
type keyboardManager struct {
	embedder *embedderKeyResponder
	channel  *channelKeyResponder
}

func (m *keyboardManager) handleEvent(event keyEvent, onUnhandled func()) {
	unreplied := 2
	anyHandled := false
	reply := func(handled bool) {
		unreplied--
		anyHandled = anyHandled || handled
		if unreplied == 0 && !anyHandled && onUnhandled != nil {
			onUnhandled()
		}
	}
	m.embedder.handleEvent(event, reply)
	m.channel.handleEvent(event, reply)
}
