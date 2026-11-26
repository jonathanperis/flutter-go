package services

// Port of raw_keyboard.dart (Flutter 3.50.1): KeyboardSide, ModifierKey,
// RawKeyEventData, RawKeyEvent (RawKeyDownEvent, RawKeyUpEvent) and
// RawKeyboard, all deprecated in Flutter but public. The platform data
// classes are in raw_keyboard_<platform>.go.

import (
	"runtime"
	"slices"
	"strconv"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// KeyboardSide is the side of the keyboard a modifier key is on.
type KeyboardSide int

const (
	// KeyboardSideAny: either side (or the platform can't tell).
	KeyboardSideAny KeyboardSide = iota
	// KeyboardSideLeft: the left side.
	KeyboardSideLeft
	// KeyboardSideRight: the right side.
	KeyboardSideRight
	// KeyboardSideAll: both sides.
	KeyboardSideAll
)

// KeyboardSideValues is KeyboardSide.values.
var KeyboardSideValues = []KeyboardSide{KeyboardSideAny, KeyboardSideLeft, KeyboardSideRight, KeyboardSideAll}

func (s KeyboardSide) Name() string   { return [...]string{"any", "left", "right", "all"}[s] }
func (s KeyboardSide) String() string { return "KeyboardSide." + s.Name() }

// ModifierKey is a modifier key that may be pressed during a key event.
type ModifierKey int

const (
	ModifierKeyControlModifier ModifierKey = iota
	ModifierKeyShiftModifier
	ModifierKeyAltModifier
	ModifierKeyMetaModifier
	ModifierKeyCapsLockModifier
	ModifierKeyNumLockModifier
	ModifierKeyScrollLockModifier
	ModifierKeyFunctionModifier
	ModifierKeySymbolModifier
)

// ModifierKeyValues is ModifierKey.values.
var ModifierKeyValues = []ModifierKey{ModifierKeyControlModifier, ModifierKeyShiftModifier, ModifierKeyAltModifier,
	ModifierKeyMetaModifier, ModifierKeyCapsLockModifier, ModifierKeyNumLockModifier, ModifierKeyScrollLockModifier,
	ModifierKeyFunctionModifier, ModifierKeySymbolModifier}

func (k ModifierKey) Name() string {
	return [...]string{"controlModifier", "shiftModifier", "altModifier", "metaModifier", "capsLockModifier",
		"numLockModifier", "scrollLockModifier", "functionModifier", "symbolModifier"}[k]
}
func (k ModifierKey) String() string { return "ModifierKey." + k.Name() }

// IsModifierPressedOptions are isModifierPressed's named parameters (the
// zero Side is Dart's default, KeyboardSide.any).
type IsModifierPressedOptions struct{ Side KeyboardSide }

func modifierSide(opts []IsModifierPressedOptions) KeyboardSide {
	if len(opts) > 0 {
		return opts[0].Side
	}
	return KeyboardSideAny
}

// RawKeyEventData is the platform-specific data of a raw key event
// (abstract; deprecated): *RawKeyEventDataAndroid, ...Fuchsia, ...Ios,
// ...Linux, ...MacOs, ...Web, ...Windows. Implementations embed
// RawKeyEventDataBase and call SetSelf.
type RawKeyEventData interface {
	foundation.Diagnosticable
	// IsModifierPressed is whether the modifier is pressed on the side.
	IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool
	// GetModifierSide is the side the modifier is pressed on (nil: none).
	GetModifierSide(key ModifierKey) *KeyboardSide
	IsControlPressed() bool
	IsShiftPressed() bool
	IsAltPressed() bool
	IsMetaPressed() bool
	// ModifiersPressed maps each pressed modifier to its side.
	ModifiersPressed() map[ModifierKey]KeyboardSide
	PhysicalKey() PhysicalKeyboardKey
	LogicalKey() LogicalKeyboardKey
	KeyLabel() string
	// ShouldDispatchEvent is whether the event should be dispatched.
	ShouldDispatchEvent() bool
}

// RawKeyEventDataBase is RawKeyEventData's default methods, which call the
// abstract ones through self.
type RawKeyEventDataBase struct {
	foundation.DiagnosticableMixin
	self RawKeyEventData
}

// SetSelf points the base at the data object embedding it.
func (d *RawKeyEventDataBase) SetSelf(self RawKeyEventData) {
	d.self = self
	d.DiagnosticableMixin.SetSelf(self)
}

func (d *RawKeyEventDataBase) IsControlPressed() bool {
	return d.self.IsModifierPressed(ModifierKeyControlModifier)
}
func (d *RawKeyEventDataBase) IsShiftPressed() bool {
	return d.self.IsModifierPressed(ModifierKeyShiftModifier)
}
func (d *RawKeyEventDataBase) IsAltPressed() bool {
	return d.self.IsModifierPressed(ModifierKeyAltModifier)
}
func (d *RawKeyEventDataBase) IsMetaPressed() bool {
	return d.self.IsModifierPressed(ModifierKeyMetaModifier)
}

func (d *RawKeyEventDataBase) ModifiersPressed() map[ModifierKey]KeyboardSide {
	result := map[ModifierKey]KeyboardSide{}
	for _, key := range ModifierKeyValues {
		if d.self.IsModifierPressed(key) {
			side := d.self.GetModifierSide(key)
			if side != nil {
				result[key] = *side
			}
			if foundation.KDebugMode && side == nil {
				foundation.DebugPrint("Raw key data is returning inconsistent information for " +
					"pressed modifiers. isModifierPressed returns true for " + key.String() + " " +
					"being pressed, but when getModifierSide is called, it says " +
					"that no modifiers are pressed.")
				if android, ok := d.self.(*RawKeyEventDataAndroid); ok {
					foundation.DebugPrint("Android raw key metaState: " + strconv.Itoa(android.MetaState))
				}
			}
		}
	}
	return result
}

func (d *RawKeyEventDataBase) ShouldDispatchEvent() bool { return true }

// RawKeyEvent is a raw key event (abstract; deprecated): *RawKeyDownEvent or
// *RawKeyUpEvent.
type RawKeyEvent interface {
	foundation.Diagnosticable
	// Data is the platform-specific data.
	Data() RawKeyEventData
	// Character is the character the key produced (nil: none).
	Character() *string
	// Repeat is whether this is a repeated down event.
	Repeat() bool
	PhysicalKey() PhysicalKeyboardKey
	LogicalKey() LogicalKeyboardKey
	// IsKeyPressed is whether the key is pressed (RawKeyboard's state).
	IsKeyPressed(key LogicalKeyboardKey) bool
	IsControlPressed() bool
	IsShiftPressed() bool
	IsAltPressed() bool
	IsMetaPressed() bool
}

// RawKeyEventBase is RawKeyEvent's state and methods.
type RawKeyEventBase struct {
	foundation.DiagnosticableMixin
	data      RawKeyEventData
	character *string
	repeat    bool
}

func (e *RawKeyEventBase) Data() RawKeyEventData            { return e.data }
func (e *RawKeyEventBase) Character() *string               { return e.character }
func (e *RawKeyEventBase) Repeat() bool                     { return e.repeat }
func (e *RawKeyEventBase) PhysicalKey() PhysicalKeyboardKey { return e.data.PhysicalKey() }
func (e *RawKeyEventBase) LogicalKey() LogicalKeyboardKey   { return e.data.LogicalKey() }

func (e *RawKeyEventBase) IsKeyPressed(key LogicalKeyboardKey) bool {
	return slices.Contains(RawKeyboardInstance.KeysPressed(), key)
}
func (e *RawKeyEventBase) IsControlPressed() bool {
	return e.IsKeyPressed(LogicalKeyboardKeyControlLeft) || e.IsKeyPressed(LogicalKeyboardKeyControlRight)
}
func (e *RawKeyEventBase) IsShiftPressed() bool {
	return e.IsKeyPressed(LogicalKeyboardKeyShiftLeft) || e.IsKeyPressed(LogicalKeyboardKeyShiftRight)
}
func (e *RawKeyEventBase) IsAltPressed() bool {
	return e.IsKeyPressed(LogicalKeyboardKeyAltLeft) || e.IsKeyPressed(LogicalKeyboardKeyAltRight)
}
func (e *RawKeyEventBase) IsMetaPressed() bool {
	return e.IsKeyPressed(LogicalKeyboardKeyMetaLeft) || e.IsKeyPressed(LogicalKeyboardKeyMetaRight)
}

// RawKeyEventOptions are the RawKeyEvent constructors' optional named
// parameters (RawKeyUpEvent ignores Repeat).
type RawKeyEventOptions struct {
	Character *string
	Repeat    bool
}

// RawKeyDownEvent is a raw key press.
type RawKeyDownEvent struct{ RawKeyEventBase }

// NewRawKeyDownEvent is RawKeyDownEvent({required data, character, repeat}).
func NewRawKeyDownEvent(data RawKeyEventData, opts ...RawKeyEventOptions) *RawKeyDownEvent {
	var o RawKeyEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &RawKeyDownEvent{RawKeyEventBase{data: data, character: o.Character, repeat: o.Repeat}}
	e.SetSelf(e)
	return e
}

func (e *RawKeyDownEvent) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.RawKeyEventBase.debugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("repeat", e.repeat))
}

// RawKeyUpEvent is a raw key release.
type RawKeyUpEvent struct{ RawKeyEventBase }

// NewRawKeyUpEvent is RawKeyUpEvent({required data, character}).
func NewRawKeyUpEvent(data RawKeyEventData, opts ...RawKeyEventOptions) *RawKeyUpEvent {
	var o RawKeyEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &RawKeyUpEvent{RawKeyEventBase{data: data, character: o.Character}}
	e.SetSelf(e)
	return e
}

func (e *RawKeyUpEvent) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.RawKeyEventBase.debugFillProperties(properties)
}

func (e *RawKeyEventBase) debugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("logicalKey", e.LogicalKey()))
	properties.Add(foundation.NewDiagnosticsProperty("physicalKey", e.PhysicalKey()))
}

func intOr(message map[string]any, key string, or int) int {
	if v, ok := message[key].(int); ok {
		return v
	}
	return or
}

func stringOr(message map[string]any, key string, or string) string {
	if v, ok := message[key].(string); ok {
		return v
	}
	return or
}

func intPtr(message map[string]any, key string) *int {
	if v, ok := message[key].(int); ok {
		return &v
	}
	return nil
}

// RawKeyEventFromMessage is RawKeyEvent.fromMessage: the event a raw key
// message of the flutter/keyevent channel describes. It panics with a
// FlutterError for an unknown keymap or event type.
func RawKeyEventFromMessage(message map[string]any) RawKeyEvent {
	var character *string
	dataFromWeb := func() RawKeyEventData {
		key, hasKey := message["key"].(string)
		if hasKey && key != "" && dart.StringLength(key) == 1 {
			c := key
			character = &c
		}
		return NewRawKeyEventDataWeb(RawKeyEventDataWebOptions{
			Code:      stringOr(message, "code", ""),
			Key:       key,
			Location:  intOr(message, "location", 0),
			MetaState: intOr(message, "metaState", 0),
			KeyCode:   intOr(message, "keyCode", 0),
		})
	}
	var data RawKeyEventData
	keymap := message["keymap"].(string)
	switch keymap {
	case "android":
		data = NewRawKeyEventDataAndroid(RawKeyEventDataAndroidOptions{
			Flags:          intOr(message, "flags", 0),
			CodePoint:      intOr(message, "codePoint", 0),
			KeyCode:        intOr(message, "keyCode", 0),
			PlainCodePoint: intOr(message, "plainCodePoint", 0),
			ScanCode:       intOr(message, "scanCode", 0),
			MetaState:      intOr(message, "metaState", 0),
			EventSource:    intOr(message, "source", 0),
			VendorId:       intOr(message, "vendorId", 0),
			ProductId:      intOr(message, "productId", 0),
			DeviceId:       intOr(message, "deviceId", 0),
			RepeatCount:    intOr(message, "repeatCount", 0),
		})
		if c, ok := message["character"]; ok {
			if s, ok := c.(string); ok {
				character = &s
			} else {
				character = nil
			}
		}
	case "fuchsia":
		codePoint := intOr(message, "codePoint", 0)
		data = NewRawKeyEventDataFuchsia(RawKeyEventDataFuchsiaOptions{
			HidUsage:  intOr(message, "hidUsage", 0),
			CodePoint: codePoint,
			Modifiers: intOr(message, "modifiers", 0),
		})
		if codePoint != 0 {
			s := dart.StringFromCharCode(codePoint)
			character = &s
		}
	case "macos":
		data = NewRawKeyEventDataMacOs(RawKeyEventDataMacOsOptions{
			Characters:                  stringOr(message, "characters", ""),
			CharactersIgnoringModifiers: stringOr(message, "charactersIgnoringModifiers", ""),
			KeyCode:                     intOr(message, "keyCode", 0),
			Modifiers:                   intOr(message, "modifiers", 0),
			SpecifiedLogicalKey:         intPtr(message, "specifiedLogicalKey"),
		})
		if s, ok := message["characters"].(string); ok {
			character = &s
		}
	case "ios":
		data = NewRawKeyEventDataIos(RawKeyEventDataIosOptions{
			Characters:                  stringOr(message, "characters", ""),
			CharactersIgnoringModifiers: stringOr(message, "charactersIgnoringModifiers", ""),
			KeyCode:                     intOr(message, "keyCode", 0),
			Modifiers:                   intOr(message, "modifiers", 0),
		})
		if s, ok := message["characters"].(string); ok && s != "" {
			character = &s
		}
	case "linux":
		unicodeScalarValues := intOr(message, "unicodeScalarValues", 0)
		data = NewRawKeyEventDataLinux(NewKeyHelper(stringOr(message, "toolkit", "")), message["type"] == "keydown",
			RawKeyEventDataLinuxOptions{
				UnicodeScalarValues: unicodeScalarValues,
				KeyCode:             intOr(message, "keyCode", 0),
				ScanCode:            intOr(message, "scanCode", 0),
				Modifiers:           intOr(message, "modifiers", 0),
				SpecifiedLogicalKey: intPtr(message, "specifiedLogicalKey"),
			})
		if unicodeScalarValues != 0 {
			s := dart.StringFromCharCode(unicodeScalarValues)
			character = &s
		}
	case "windows":
		characterCodePoint := intOr(message, "characterCodePoint", 0)
		data = NewRawKeyEventDataWindows(RawKeyEventDataWindowsOptions{
			KeyCode:            intOr(message, "keyCode", 0),
			ScanCode:           intOr(message, "scanCode", 0),
			CharacterCodePoint: characterCodePoint,
			Modifiers:          intOr(message, "modifiers", 0),
		})
		if characterCodePoint != 0 {
			s := dart.StringFromCharCode(characterCodePoint)
			character = &s
		}
	case "web":
		data = dataFromWeb()
	default:
		panic(foundation.NewFlutterError("Unknown keymap for key events: " + keymap))
	}
	repeat := slices.Contains(RawKeyboardInstance.PhysicalKeysPressed(), data.PhysicalKey())
	typ := message["type"].(string)
	switch typ {
	case "keydown":
		return NewRawKeyDownEvent(data, RawKeyEventOptions{Character: character, Repeat: repeat})
	case "keyup":
		return NewRawKeyUpEvent(data)
	}
	panic(foundation.NewFlutterError("Unknown key event type: " + typ))
}

// RawKeyEventHandler is the signature of RawKeyboard.keyEventHandler.
type RawKeyEventHandler func(event RawKeyEvent) bool

// RawKeyboard is the legacy keyboard state and raw key event dispatcher
// (deprecated). Its one instance is RawKeyboardInstance.
type RawKeyboard struct {
	listeners               []func(RawKeyEvent)
	cachedKeyEventHandler   RawKeyEventHandler
	cachedKeyMessageHandler KeyMessageHandler
	keysPressed             linkedMap[PhysicalKeyboardKey, LogicalKeyboardKey]
}

// RawKeyboardInstance is RawKeyboard.instance.
var RawKeyboardInstance = &RawKeyboard{}

// AddListener calls listener with every raw key event.
func (r *RawKeyboard) AddListener(listener func(RawKeyEvent)) {
	r.listeners = append(r.listeners, listener)
}

// RemoveListener stops calling listener (compared with
// dart.FunctionsEqual).
func (r *RawKeyboard) RemoveListener(listener func(RawKeyEvent)) {
	for i, l := range r.listeners {
		if dart.FunctionsEqual(l, listener) {
			r.listeners = slices.Delete(r.listeners, i, i+1)
			return
		}
	}
}

// KeyEventHandler is a dummy handler kept for compatibility while
// KeyEventManager's keyMessageHandler is set (nil otherwise).
func (r *RawKeyboard) KeyEventHandler() RawKeyEventHandler {
	current := ServicesBindingInstance().KeyEventManager().KeyMessageHandler
	if !keyMessageHandlersEqual(current, r.cachedKeyMessageHandler) {
		r.cachedKeyMessageHandler = current
		if current == nil {
			r.cachedKeyEventHandler = nil
		} else {
			r.cachedKeyEventHandler = func(RawKeyEvent) bool {
				panic(dart.NewAssertionError("The RawKeyboard.instance.keyEventHandler assigned by Flutter is a dummy " +
					"callback kept for compatibility and should not be directly called. Use " +
					"ServicesBinding.instance!.keyMessageHandler instead."))
			}
		}
	}
	return r.cachedKeyEventHandler
}

func keyMessageHandlersEqual(a, b KeyMessageHandler) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return dart.FunctionsEqual(a, b)
}

// SetKeyEventHandler sets KeyEventManager's keyMessageHandler to one
// calling handler with the messages' raw events.
func (r *RawKeyboard) SetKeyEventHandler(handler RawKeyEventHandler) {
	r.cachedKeyEventHandler = handler
	if handler == nil {
		r.cachedKeyMessageHandler = nil
	} else {
		r.cachedKeyMessageHandler = func(message *KeyMessage) bool {
			if message.RawEvent != nil {
				return handler(message.RawEvent)
			}
			return false
		}
	}
	ServicesBindingInstance().KeyEventManager().KeyMessageHandler = r.cachedKeyMessageHandler
}

// HandleRawKeyEvent updates the pressed keys with the event and calls the
// listeners; it returns false.
func (r *RawKeyboard) HandleRawKeyEvent(event RawKeyEvent) bool {
	switch event.(type) {
	case *RawKeyDownEvent:
		r.keysPressed.set(event.PhysicalKey(), event.LogicalKey())
	case *RawKeyUpEvent:
		// Use the physical key in the key up event to find the physical key
		// from the corresponding key down event and remove it, even if the
		// logical keys don't match.
		r.keysPressed.remove(event.PhysicalKey())
	}
	// Make sure that the modifiers reflect reality, in case a modifier key
	// was pressed/released while the app didn't have focus.
	r.synchronizeModifiers(event)
	if _, down := event.(*RawKeyDownEvent); down && r.keysPressed.len() == 0 {
		panic(dart.NewAssertionError("Attempted to send a key down event when no keys are in keysPressed. " +
			"This state can occur if the key event being sent doesn't properly " +
			"set its modifier flags. This was the event: " + event.String() + " and its data: " +
			event.Data().String()))
	}
	// Send the event to passive listeners.
	for _, listener := range slices.Clone(r.listeners) {
		if !slices.ContainsFunc(r.listeners, func(l func(RawKeyEvent)) bool { return dart.FunctionsEqual(l, listener) }) {
			continue
		}
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					reportKeyError(exception, "while processing a raw key listener", "Event", event)
				}
			}()
			listener(event)
		}()
	}
	return false
}

type modifierSidePair struct {
	modifier ModifierKey
	side     KeyboardSide
}

var modifierKeyMap = map[modifierSidePair][]PhysicalKeyboardKey{
	{ModifierKeyAltModifier, KeyboardSideLeft}:       {PhysicalKeyboardKeyAltLeft},
	{ModifierKeyAltModifier, KeyboardSideRight}:      {PhysicalKeyboardKeyAltRight},
	{ModifierKeyAltModifier, KeyboardSideAll}:        {PhysicalKeyboardKeyAltLeft, PhysicalKeyboardKeyAltRight},
	{ModifierKeyAltModifier, KeyboardSideAny}:        {PhysicalKeyboardKeyAltLeft},
	{ModifierKeyShiftModifier, KeyboardSideLeft}:     {PhysicalKeyboardKeyShiftLeft},
	{ModifierKeyShiftModifier, KeyboardSideRight}:    {PhysicalKeyboardKeyShiftRight},
	{ModifierKeyShiftModifier, KeyboardSideAll}:      {PhysicalKeyboardKeyShiftLeft, PhysicalKeyboardKeyShiftRight},
	{ModifierKeyShiftModifier, KeyboardSideAny}:      {PhysicalKeyboardKeyShiftLeft},
	{ModifierKeyControlModifier, KeyboardSideLeft}:   {PhysicalKeyboardKeyControlLeft},
	{ModifierKeyControlModifier, KeyboardSideRight}:  {PhysicalKeyboardKeyControlRight},
	{ModifierKeyControlModifier, KeyboardSideAll}:    {PhysicalKeyboardKeyControlLeft, PhysicalKeyboardKeyControlRight},
	{ModifierKeyControlModifier, KeyboardSideAny}:    {PhysicalKeyboardKeyControlLeft},
	{ModifierKeyMetaModifier, KeyboardSideLeft}:      {PhysicalKeyboardKeyMetaLeft},
	{ModifierKeyMetaModifier, KeyboardSideRight}:     {PhysicalKeyboardKeyMetaRight},
	{ModifierKeyMetaModifier, KeyboardSideAll}:       {PhysicalKeyboardKeyMetaLeft, PhysicalKeyboardKeyMetaRight},
	{ModifierKeyMetaModifier, KeyboardSideAny}:       {PhysicalKeyboardKeyMetaLeft},
	{ModifierKeyCapsLockModifier, KeyboardSideAll}:   {PhysicalKeyboardKeyCapsLock},
	{ModifierKeyNumLockModifier, KeyboardSideAll}:    {PhysicalKeyboardKeyNumLock},
	{ModifierKeyScrollLockModifier, KeyboardSideAll}: {PhysicalKeyboardKeyScrollLock},
	{ModifierKeyFunctionModifier, KeyboardSideAll}:   {PhysicalKeyboardKeyFn},
	// The symbolModifier doesn't have a key representation on any of the
	// platforms, so don't map it here.
}

// allModifiersExceptFn maps every modifier key but Fn (in Dart's order).
var allModifiersExceptFn = []struct {
	physical PhysicalKeyboardKey
	logical  LogicalKeyboardKey
}{
	{PhysicalKeyboardKeyAltLeft, LogicalKeyboardKeyAltLeft},
	{PhysicalKeyboardKeyAltRight, LogicalKeyboardKeyAltRight},
	{PhysicalKeyboardKeyShiftLeft, LogicalKeyboardKeyShiftLeft},
	{PhysicalKeyboardKeyShiftRight, LogicalKeyboardKeyShiftRight},
	{PhysicalKeyboardKeyControlLeft, LogicalKeyboardKeyControlLeft},
	{PhysicalKeyboardKeyControlRight, LogicalKeyboardKeyControlRight},
	{PhysicalKeyboardKeyMetaLeft, LogicalKeyboardKeyMetaLeft},
	{PhysicalKeyboardKeyMetaRight, LogicalKeyboardKeyMetaRight},
	{PhysicalKeyboardKeyCapsLock, LogicalKeyboardKeyCapsLock},
	{PhysicalKeyboardKeyNumLock, LogicalKeyboardKeyNumLock},
	{PhysicalKeyboardKeyScrollLock, LogicalKeyboardKeyScrollLock},
}

func lookUpModifierExceptFn(p PhysicalKeyboardKey) *LogicalKeyboardKey {
	for _, e := range allModifiersExceptFn {
		if e.physical == p {
			l := e.logical
			return &l
		}
	}
	return nil
}

// allModifiers is allModifiersExceptFn plus Fn.
func lookUpModifier(p PhysicalKeyboardKey) LogicalKeyboardKey {
	if p == PhysicalKeyboardKeyFn {
		return LogicalKeyboardKeyFn
	}
	return *lookUpModifierExceptFn(p)
}

func (r *RawKeyboard) synchronizeModifiers(event RawKeyEvent) {
	// Compare modifier states to the ground truth as specified by
	// event.data.modifiersPressed and update unsynchronized ones.
	modifiersPressed := event.Data().ModifiersPressed()
	var modifierKeys linkedMap[PhysicalKeyboardKey, LogicalKeyboardKey]
	// Physical keys whose modifiers are pressed at any side.
	var anySideKeys []PhysicalKeyboardKey
	keysPressedAfterEvent := r.keysPressed.keySet()
	if _, down := event.(*RawKeyDownEvent); down {
		keysPressedAfterEvent = addToSet(keysPressedAfterEvent, event.PhysicalKey())
	}
	var thisKeyModifier *ModifierKey
	for _, key := range ModifierKeyValues {
		thisModifierKeys, ok := modifierKeyMap[modifierSidePair{key, KeyboardSideAll}]
		if !ok {
			continue
		}
		if slices.Contains(thisModifierKeys, event.PhysicalKey()) {
			k := key
			thisKeyModifier = &k
		}
		side, pressed := modifiersPressed[key]
		if pressed && side == KeyboardSideAny {
			for _, k := range thisModifierKeys {
				anySideKeys = addToSet(anySideKeys, k)
			}
			if slices.ContainsFunc(thisModifierKeys, func(k PhysicalKeyboardKey) bool {
				return slices.Contains(keysPressedAfterEvent, k)
			}) {
				continue
			}
		}
		var mappedKeys []PhysicalKeyboardKey
		mapped := true
		if pressed {
			mappedKeys, mapped = modifierKeyMap[modifierSidePair{key, side}]
		}
		if foundation.KDebugMode && !mapped {
			foundation.DebugPrint("Platform key support for " + hostOperatingSystem() + " is " +
				"producing unsupported modifier combinations for " +
				"modifier " + key.String() + " on side " + side.String() + ".")
			if android, ok := event.Data().(*RawKeyEventDataAndroid); ok {
				foundation.DebugPrint("Android raw key metaState: " + strconv.Itoa(android.MetaState))
			}
		}
		if !mapped {
			continue
		}
		for _, physicalModifier := range mappedKeys {
			modifierKeys.set(physicalModifier, lookUpModifier(physicalModifier))
		}
	}
	// On Linux, CapsLock key can be mapped to a non-modifier logical key:
	// https://github.com/flutter/flutter/issues/114591.
	_, isLinux := event.Data().(*RawKeyEventDataLinux)
	_, isWeb := event.Data().(*RawKeyEventDataWeb)
	capsLogical, capsPressed := r.keysPressed.get(PhysicalKeyboardKeyCapsLock)
	nonModifierCapsLock := (isLinux || isWeb) && capsPressed && capsLogical != LogicalKeyboardKeyCapsLock
	for _, e := range allModifiersExceptFn {
		skipReleasingKey := nonModifierCapsLock && e.physical == PhysicalKeyboardKeyCapsLock
		if !slices.Contains(anySideKeys, e.physical) && !skipReleasingKey {
			r.keysPressed.remove(e.physical)
		}
	}
	_, isFuchsia := event.Data().(*RawKeyEventDataFuchsia)
	_, isMacOs := event.Data().(*RawKeyEventDataMacOs)
	if !isFuchsia && !isMacOs {
		// On Fuchsia and macOS, the Fn key is not considered a modifier key.
		r.keysPressed.remove(PhysicalKeyboardKeyFn)
	}
	for _, k := range modifierKeys.keys {
		v, _ := modifierKeys.get(k)
		r.keysPressed.set(k, v)
	}
	// In rare cases, the event presses a modifier key but the key does not
	// exist in the modifier list. Enforce the pressing state.
	if _, down := event.(*RawKeyDownEvent); down && thisKeyModifier != nil && !r.keysPressed.containsKey(event.PhysicalKey()) {
		_, isIos := event.Data().(*RawKeyEventDataIos)
		_, isAndroid := event.Data().(*RawKeyEventDataAndroid)
		// Found on Linux GTK for AltRight (flutter#93278), and on Android and
		// iOS (flutter#101090).
		if (isLinux && event.PhysicalKey() == PhysicalKeyboardKeyAltRight) || isIos || isAndroid {
			if logicalKey := lookUpModifierExceptFn(event.PhysicalKey()); logicalKey != nil {
				r.keysPressed.set(event.PhysicalKey(), *logicalKey)
			}
		}
		// On Web, PhysicalKeyboardKey.altRight can be map to
		// LogicalKeyboardKey.altGraph or LogicalKeyboardKey.altRight
		// (flutter#113836).
		if isWeb && event.PhysicalKey() == PhysicalKeyboardKeyAltRight {
			r.keysPressed.set(event.PhysicalKey(), event.LogicalKey())
		}
	}
}

// KeysPressed is the set of logical keys pressed.
func (r *RawKeyboard) KeysPressed() []LogicalKeyboardKey { return valueSet(&r.keysPressed) }

// PhysicalKeysPressed is the set of physical keys pressed.
func (r *RawKeyboard) PhysicalKeysPressed() []PhysicalKeyboardKey { return r.keysPressed.keySet() }

// LookUpLayout returns the logical key the physical key was pressed with,
// or nil.
func (r *RawKeyboard) LookUpLayout(physicalKey PhysicalKeyboardKey) *LogicalKeyboardKey {
	if l, ok := r.keysPressed.get(physicalKey); ok {
		return &l
	}
	return nil
}

// ClearKeysPressed clears the pressed keys (for tests).
func (r *RawKeyboard) ClearKeysPressed() { r.keysPressed.clear() }

// hostOperatingSystem is dart:io's Platform.operatingSystem.
func hostOperatingSystem() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}
