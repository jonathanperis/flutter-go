package services

// Port of hardware_keyboard.dart (Flutter 3.50.1): KeyboardLockMode,
// KeyEvent and its subclasses, HardwareKeyboard, and the deprecated
// KeyDataTransitMode, KeyMessage and KeyEventManager.
//
// Dart's Sets and LinkedHashMaps keep insertion order, which these
// classes expose (pressed keys, synthesized events); Go keeps it with
// linkedMap and slices without duplicates.

import (
	"iter"
	"slices"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// linkedMap is Dart's default (insertion-ordered) Map: setting an
// existing key keeps its position, removing and adding moves it to the end.
type linkedMap[K comparable, V any] struct {
	keys []K
	m    map[K]V
}

func (lm *linkedMap[K, V]) get(k K) (V, bool) {
	v, ok := lm.m[k]
	return v, ok
}

func (lm *linkedMap[K, V]) set(k K, v V) {
	if lm.m == nil {
		lm.m = map[K]V{}
	}
	if _, ok := lm.m[k]; !ok {
		lm.keys = append(lm.keys, k)
	}
	lm.m[k] = v
}

func (lm *linkedMap[K, V]) remove(k K) {
	if _, ok := lm.m[k]; !ok {
		return
	}
	delete(lm.m, k)
	lm.keys = slices.DeleteFunc(lm.keys, func(x K) bool { return x == k })
}

func (lm *linkedMap[K, V]) containsKey(k K) bool {
	_, ok := lm.m[k]
	return ok
}

func (lm *linkedMap[K, V]) len() int { return len(lm.keys) }

func (lm *linkedMap[K, V]) clear() {
	lm.keys = nil
	lm.m = nil
}

// keySet returns the keys (Dart's keys.toSet()).
func (lm *linkedMap[K, V]) keySet() []K { return slices.Clone(lm.keys) }

// valueSet returns the values without duplicates, in order (Dart's
// values.toSet()).
func valueSet[K, V comparable](lm *linkedMap[K, V]) []V {
	out := []V{}
	for _, k := range lm.keys {
		if v := lm.m[k]; !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func containsValue[K, V comparable](lm *linkedMap[K, V], v V) bool {
	for _, k := range lm.keys {
		if lm.m[k] == v {
			return true
		}
	}
	return false
}

// keyboardDebug is _keyboardDebug: it prints messageFunc's message (and the
// details) when DebugPrintKeyboardEvents is set. Call it only in debug
// builds (Dart calls it inside asserts).
func keyboardDebug(messageFunc func() string, detailsFunc func() []string) bool {
	if foundation.KReleaseMode {
		panic(dart.StateError{Message: "_keyboardDebug was called in Release mode, which means they are called " +
			"without being wrapped in an assert. Always call _keyboardDebug like so:\n" +
			"  assert(_keyboardDebug(() => 'Blah $foo'));"})
	}
	if !DebugPrintKeyboardEvents {
		return true
	}
	foundation.DebugPrint("KEYBOARD: " + messageFunc())
	if detailsFunc != nil {
		for _, detail := range detailsFunc() {
			foundation.DebugPrint("    " + detail)
		}
	}
	return true
}

// KeyboardLockMode is a lock mode of the keyboard, toggled by its key.
type KeyboardLockMode int

const (
	// KeyboardLockModeNumLock is the Num Lock mode.
	KeyboardLockModeNumLock KeyboardLockMode = iota
	// KeyboardLockModeScrollLock is the Scroll Lock mode.
	KeyboardLockModeScrollLock
	// KeyboardLockModeCapsLock is the Caps Lock mode.
	KeyboardLockModeCapsLock
)

// KeyboardLockModeValues is KeyboardLockMode.values.
var KeyboardLockModeValues = []KeyboardLockMode{KeyboardLockModeNumLock, KeyboardLockModeScrollLock, KeyboardLockModeCapsLock}

func (m KeyboardLockMode) Name() string {
	return [...]string{"numLock", "scrollLock", "capsLock"}[m]
}

func (m KeyboardLockMode) String() string { return "KeyboardLockMode." + m.Name() }

// LogicalKey is the logical key that triggers this lock mode.
func (m KeyboardLockMode) LogicalKey() LogicalKeyboardKey {
	return [...]LogicalKeyboardKey{LogicalKeyboardKeyNumLock, LogicalKeyboardKeyScrollLock, LogicalKeyboardKeyCapsLock}[m]
}

// KeyboardLockModeFindLockByLogicalKey is
// KeyboardLockMode.findLockByLogicalKey: the lock mode the key triggers, or
// nil.
func KeyboardLockModeFindLockByLogicalKey(logicalKey LogicalKeyboardKey) *KeyboardLockMode {
	for _, m := range KeyboardLockModeValues {
		if m.LogicalKey().KeyID == logicalKey.KeyID {
			return &m
		}
	}
	return nil
}

// KeyEvent is a key event of the keyboard (abstract): *KeyDownEvent,
// *KeyUpEvent or *KeyRepeatEvent (Dart's `event is KeyDownEvent` is a type
// switch). Its fields are getters.
type KeyEvent interface {
	foundation.Diagnosticable
	// PhysicalKey is the key's position on the keyboard.
	PhysicalKey() PhysicalKeyboardKey
	// LogicalKey is the key's meaning in the current layout.
	LogicalKey() LogicalKeyboardKey
	// Character is the character produced by the key (nil: none).
	Character() *string
	// TimeStamp is the time the event happened.
	TimeStamp() time.Duration
	// DeviceType is the type of the device that sent the event.
	DeviceType() ui.KeyEventDeviceType
	// Synthesized is whether the event was synthesized by Flutter.
	Synthesized() bool

	keyEvent() *KeyEventBase
}

// KeyEventBase is the state and getters of KeyEvent, embedded by the event
// classes.
type KeyEventBase struct {
	foundation.DiagnosticableMixin
	physicalKey PhysicalKeyboardKey
	logicalKey  LogicalKeyboardKey
	character   *string
	timeStamp   time.Duration
	deviceType  ui.KeyEventDeviceType
	synthesized bool
}

func (e *KeyEventBase) PhysicalKey() PhysicalKeyboardKey  { return e.physicalKey }
func (e *KeyEventBase) LogicalKey() LogicalKeyboardKey    { return e.logicalKey }
func (e *KeyEventBase) Character() *string                { return e.character }
func (e *KeyEventBase) TimeStamp() time.Duration          { return e.timeStamp }
func (e *KeyEventBase) DeviceType() ui.KeyEventDeviceType { return e.deviceType }
func (e *KeyEventBase) Synthesized() bool                 { return e.synthesized }
func (e *KeyEventBase) keyEvent() *KeyEventBase           { return e }

func (e *KeyEventBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("physicalKey", e.physicalKey))
	properties.Add(foundation.NewDiagnosticsProperty("logicalKey", e.logicalKey))
	properties.Add(foundation.NewStringProperty("character", e.character))
	properties.Add(foundation.NewDiagnosticsProperty("timeStamp", e.timeStamp))
	ifTrue := "synthesized"
	properties.Add(foundation.NewFlagProperty("synthesized", e.synthesized, foundation.FlagPropertyOptions{IfTrue: &ifTrue}))
}

// KeyEventOptions are the optional named parameters of the KeyEvent
// constructors (the zero DeviceType is Dart's default, keyboard). KeyUpEvent
// ignores Character, and KeyRepeatEvent ignores Synthesized (their Dart
// constructors don't take them).
type KeyEventOptions struct {
	Character   *string
	DeviceType  ui.KeyEventDeviceType
	Synthesized bool
}

func keyEventOpts(opts []KeyEventOptions) KeyEventOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return KeyEventOptions{}
}

// KeyDownEvent is a key press: the key was pressed with no other key event
// before for it, or after its key up.
type KeyDownEvent struct{ KeyEventBase }

// NewKeyDownEvent is KeyDownEvent({required physicalKey, required
// logicalKey, character, required timeStamp, synthesized, deviceType}) (the
// required parameters are positional).
func NewKeyDownEvent(physicalKey PhysicalKeyboardKey, logicalKey LogicalKeyboardKey, timeStamp time.Duration, opts ...KeyEventOptions) *KeyDownEvent {
	o := keyEventOpts(opts)
	e := &KeyDownEvent{KeyEventBase{physicalKey: physicalKey, logicalKey: logicalKey, character: o.Character,
		timeStamp: timeStamp, deviceType: o.DeviceType, synthesized: o.Synthesized}}
	e.SetSelf(e)
	return e
}

// KeyUpEvent is a key release.
type KeyUpEvent struct{ KeyEventBase }

// NewKeyUpEvent is KeyUpEvent({required physicalKey, required logicalKey,
// required timeStamp, synthesized, deviceType}).
func NewKeyUpEvent(physicalKey PhysicalKeyboardKey, logicalKey LogicalKeyboardKey, timeStamp time.Duration, opts ...KeyEventOptions) *KeyUpEvent {
	o := keyEventOpts(opts)
	e := &KeyUpEvent{KeyEventBase{physicalKey: physicalKey, logicalKey: logicalKey,
		timeStamp: timeStamp, deviceType: o.DeviceType, synthesized: o.Synthesized}}
	e.SetSelf(e)
	return e
}

// KeyRepeatEvent is a key repeat (the key is held down).
type KeyRepeatEvent struct{ KeyEventBase }

// NewKeyRepeatEvent is KeyRepeatEvent({required physicalKey, required
// logicalKey, character, required timeStamp, deviceType}).
func NewKeyRepeatEvent(physicalKey PhysicalKeyboardKey, logicalKey LogicalKeyboardKey, timeStamp time.Duration, opts ...KeyEventOptions) *KeyRepeatEvent {
	o := keyEventOpts(opts)
	e := &KeyRepeatEvent{KeyEventBase{physicalKey: physicalKey, logicalKey: logicalKey, character: o.Character,
		timeStamp: timeStamp, deviceType: o.DeviceType}}
	e.SetSelf(e)
	return e
}

// KeyEventCallback is the signature of a HardwareKeyboard handler: it
// returns whether it handled the event.
type KeyEventCallback func(event KeyEvent) bool

// HardwareKeyboard manages the state of the hardware keyboard (pressed keys
// and lock modes) and dispatches key events to its handlers. Use
// HardwareKeyboardInstance().
type HardwareKeyboard struct {
	pressedKeys      linkedMap[PhysicalKeyboardKey, LogicalKeyboardKey]
	lockModes        []KeyboardLockMode
	handlers         []KeyEventCallback
	duringDispatch   bool
	modifiedHandlers []KeyEventCallback // nil: not modified during the dispatch
}

// NewHardwareKeyboard is HardwareKeyboard() (ServicesBinding creates the
// instance).
func NewHardwareKeyboard() *HardwareKeyboard { return &HardwareKeyboard{} }

// HardwareKeyboardInstance is HardwareKeyboard.instance
// (ServicesBinding.instance.keyboard).
func HardwareKeyboardInstance() *HardwareKeyboard { return ServicesBindingInstance().Keyboard() }

// PhysicalKeysPressed is the set of physical keys pressed (a Dart Set: in
// order, without duplicates).
func (h *HardwareKeyboard) PhysicalKeysPressed() []PhysicalKeyboardKey { return h.pressedKeys.keySet() }

// LogicalKeysPressed is the set of logical keys pressed.
func (h *HardwareKeyboard) LogicalKeysPressed() []LogicalKeyboardKey { return valueSet(&h.pressedKeys) }

// LookUpLayout returns the logical key the physical key was pressed with,
// or nil if it is not pressed.
func (h *HardwareKeyboard) LookUpLayout(physicalKey PhysicalKeyboardKey) *LogicalKeyboardKey {
	if l, ok := h.pressedKeys.get(physicalKey); ok {
		return &l
	}
	return nil
}

// LockModesEnabled is the set of lock modes enabled.
func (h *HardwareKeyboard) LockModesEnabled() []KeyboardLockMode { return slices.Clone(h.lockModes) }

// IsLogicalKeyPressed is whether a physical key is pressed with this logical
// key.
func (h *HardwareKeyboard) IsLogicalKeyPressed(key LogicalKeyboardKey) bool {
	return containsValue(&h.pressedKeys, key)
}

// IsPhysicalKeyPressed is whether the physical key is pressed.
func (h *HardwareKeyboard) IsPhysicalKeyPressed(key PhysicalKeyboardKey) bool {
	return h.pressedKeys.containsKey(key)
}

// IsControlPressed is whether either control key is pressed.
func (h *HardwareKeyboard) IsControlPressed() bool {
	return h.IsLogicalKeyPressed(LogicalKeyboardKeyControlLeft) || h.IsLogicalKeyPressed(LogicalKeyboardKeyControlRight)
}

// IsShiftPressed is whether either shift key is pressed.
func (h *HardwareKeyboard) IsShiftPressed() bool {
	return h.IsLogicalKeyPressed(LogicalKeyboardKeyShiftLeft) || h.IsLogicalKeyPressed(LogicalKeyboardKeyShiftRight)
}

// IsAltPressed is whether either alt key is pressed.
func (h *HardwareKeyboard) IsAltPressed() bool {
	return h.IsLogicalKeyPressed(LogicalKeyboardKeyAltLeft) || h.IsLogicalKeyPressed(LogicalKeyboardKeyAltRight)
}

// IsMetaPressed is whether either meta key is pressed.
func (h *HardwareKeyboard) IsMetaPressed() bool {
	return h.IsLogicalKeyPressed(LogicalKeyboardKeyMetaLeft) || h.IsLogicalKeyPressed(LogicalKeyboardKeyMetaRight)
}

// logEventIfIrregular prints debug messages if the event is inconsistent
// with the current state (DebugPrintKeyboardEvents).
func (h *HardwareKeyboard) logEventIfIrregular(event KeyEvent) {
	if !foundation.KDebugMode {
		return
	}
	const common = "This is typically either due to https://github.com/flutter/flutter/issues/125975, " +
		"or a bug in the embedding's key event conciliation logic."
	typeName := foundation.ObjectRuntimeType(event, "")
	switch event.(type) {
	case *KeyDownEvent:
		if pressed, ok := h.pressedKeys.get(event.PhysicalKey()); ok {
			keyboardDebug(func() string {
				return "ERROR: Received unexpected " + typeName + " for key that is already pressed.\n" +
					common + "\n    Event: " + event.String() + "\n    Pressed logical key: " + pressed.String()
			}, nil)
		}
	case *KeyRepeatEvent, *KeyUpEvent:
		pressed, ok := h.pressedKeys.get(event.PhysicalKey())
		if !ok {
			keyboardDebug(func() string {
				return "ERROR: Received unexpected " + typeName + " for key that is not pressed:\n" +
					common + "\n    Event: " + event.String()
			}, nil)
		} else if pressed != event.LogicalKey() {
			keyboardDebug(func() string {
				return "ERROR: Received unexpected " + typeName + " for key with mismatched logical key:\n" +
					common + "\n    Event: " + event.String() + "\n    Pressed logical key: " + pressed.String()
			}, nil)
		}
	default:
		panic(dart.NewAssertionError("Received unexpected key event class " + typeName))
	}
}

// AddHandler registers a handler (handlers added during a dispatch take
// effect after it).
func (h *HardwareKeyboard) AddHandler(handler KeyEventCallback) {
	if h.duringDispatch {
		if h.modifiedHandlers == nil {
			h.modifiedHandlers = slices.Clone(h.handlers)
		}
		h.modifiedHandlers = append(h.modifiedHandlers, handler)
	} else {
		h.handlers = append(h.handlers, handler)
	}
}

func removeFirstHandler(list []KeyEventCallback, handler KeyEventCallback) []KeyEventCallback {
	for i, h := range list {
		if dart.FunctionsEqual(h, handler) {
			return slices.Delete(list, i, i+1)
		}
	}
	return list
}

// RemoveHandler removes a handler added with AddHandler (compared with
// dart.FunctionsEqual).
func (h *HardwareKeyboard) RemoveHandler(handler KeyEventCallback) {
	if h.duringDispatch {
		if h.modifiedHandlers == nil {
			h.modifiedHandlers = slices.Clone(h.handlers)
		}
		h.modifiedHandlers = removeFirstHandler(h.modifiedHandlers, handler)
	} else {
		h.handlers = removeFirstHandler(h.handlers, handler)
	}
}

// SyncKeyboardState queries the platform for the keys pressed
// (flutter/keyboard's getKeyboardState) and records them.
func (h *HardwareKeyboard) SyncKeyboardState() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		keyboardState := MethodChannelInvokeMapMethod[int, int](SystemChannels.Keyboard, "getKeyboardState").Await()
		if keyboardState != nil {
			// Dart iterates the decoded map in the message's order; Go's
			// decoded map has none, so the keys are recorded in increasing
			// order.
			keys := make([]int, 0, len(keyboardState))
			for k := range keyboardState {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, key := range keys {
				h.pressedKeys.set(PhysicalKeyboardKey{key}, LogicalKeyboardKey{keyboardState[key]})
			}
		}
	})
}

func (h *HardwareKeyboard) dispatchKeyEvent(event KeyEvent) bool {
	if h.duringDispatch {
		panic(dart.NewAssertionError("Nested keyboard dispatching is not supported"))
	}
	h.duringDispatch = true
	handled := false
	for _, handler := range h.handlers {
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					reportKeyError(exception, "while processing a key handler", "Event", event)
				}
			}()
			thisResult := handler(event)
			handled = handled || thisResult
		}()
	}
	h.duringDispatch = false
	if h.modifiedHandlers != nil {
		h.handlers = h.modifiedHandlers
		h.modifiedHandlers = nil
	}
	return handled
}

// reportKeyError reports an exception caught in a key handler, as the
// keyboard classes do (library "services library").
func reportKeyError[T any](exception any, context, name string, value T) {
	stack := dart.StackTraceFromPanic()
	var collector foundation.InformationCollector
	if foundation.KDebugMode {
		collector = func() iter.Seq[foundation.DiagnosticsNode] {
			return slices.Values([]foundation.DiagnosticsNode{foundation.NewDiagnosticsProperty(name, value)})
		}
	}
	library := "services library"
	foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
		Stack:                stack,
		Library:              &library,
		Context:              foundation.NewErrorDescription(context),
		InformationCollector: collector,
	}))
}

func (h *HardwareKeyboard) debugPressedKeysDetails() []string {
	if h.pressedKeys.len() == 0 {
		return []string{"Empty"}
	}
	var out []string
	for _, physicalKey := range h.pressedKeys.keys {
		logical, _ := h.pressedKeys.get(physicalKey)
		out = append(out, physicalKey.String()+": "+logical.String())
	}
	return out
}

// HandleKeyEvent processes a key event: it updates the pressed keys and
// lock modes, then dispatches it to the handlers. It returns whether a
// handler handled it.
func (h *HardwareKeyboard) HandleKeyEvent(event KeyEvent) bool {
	if foundation.KDebugMode {
		keyboardDebug(func() string { return "Key event received: " + event.String() }, nil)
		keyboardDebug(func() string { return "Pressed state before processing the event:" }, h.debugPressedKeysDetails)
	}
	h.logEventIfIrregular(event)
	physicalKey := event.PhysicalKey()
	logicalKey := event.LogicalKey()
	switch event.(type) {
	case *KeyDownEvent:
		h.pressedKeys.set(physicalKey, logicalKey)
		if lockMode := KeyboardLockModeFindLockByLogicalKey(event.LogicalKey()); lockMode != nil {
			if i := slices.Index(h.lockModes, *lockMode); i >= 0 {
				h.lockModes = slices.Delete(h.lockModes, i, i+1)
			} else {
				h.lockModes = append(h.lockModes, *lockMode)
			}
		}
	case *KeyUpEvent:
		h.pressedKeys.remove(physicalKey)
	case *KeyRepeatEvent:
		// Update the logical key in case it has changed.
		h.pressedKeys.set(physicalKey, logicalKey)
	}
	if foundation.KDebugMode {
		keyboardDebug(func() string { return "Pressed state after processing the event:" }, h.debugPressedKeysDetails)
	}
	return h.dispatchKeyEvent(event)
}

// ClearState clears the state of the keyboard (for tests).
func (h *HardwareKeyboard) ClearState() {
	h.pressedKeys.clear()
	h.lockModes = nil
	h.handlers = nil
	if h.modifiedHandlers != nil {
		panic(dart.NewAssertionError(nil))
	}
}

// KeyDataTransitMode is the mode a platform delivers key events in
// (deprecated: always key data now).
type KeyDataTransitMode int

const (
	// KeyDataTransitModeRawKeyData: key events are converted from raw key
	// messages.
	KeyDataTransitModeRawKeyData KeyDataTransitMode = iota
	// KeyDataTransitModeKeyDataThenRawKeyData: key data arrive first, then
	// a raw key message.
	KeyDataTransitModeKeyDataThenRawKeyData
)

// KeyDataTransitModeValues is KeyDataTransitMode.values.
var KeyDataTransitModeValues = []KeyDataTransitMode{KeyDataTransitModeRawKeyData, KeyDataTransitModeKeyDataThenRawKeyData}

func (m KeyDataTransitMode) Name() string {
	return [...]string{"rawKeyData", "keyDataThenRawKeyData"}[m]
}

func (m KeyDataTransitMode) String() string { return "KeyDataTransitMode." + m.Name() }

// KeyMessage is the key events of a platform message together with the
// raw key event (deprecated).
type KeyMessage struct {
	// Events are the key events, possibly empty.
	Events []KeyEvent
	// RawEvent is the legacy raw key event, or nil.
	RawEvent RawKeyEvent
}

// NewKeyMessage is KeyMessage(events, rawEvent).
func NewKeyMessage(events []KeyEvent, rawEvent RawKeyEvent) *KeyMessage {
	return &KeyMessage{Events: events, RawEvent: rawEvent}
}

func (m *KeyMessage) String() string {
	s := "KeyMessage(["
	for i, e := range m.Events {
		if i > 0 {
			s += ", "
		}
		s += e.String()
	}
	return s + "])"
}

// KeyMessageHandler is the signature of KeyEventManager.keyMessageHandler
// (deprecated).
type KeyMessageHandler func(message *KeyMessage) bool

// KeyEventManager dispatches the platform's key messages (key data and raw
// key messages) to HardwareKeyboard, RawKeyboard and its keyMessageHandler
// (deprecated).
type KeyEventManager struct {
	// KeyMessageHandler is the handler of key messages (the focus system's).
	KeyMessageHandler KeyMessageHandler

	hardwareKeyboard *HardwareKeyboard
	rawKeyboard      *RawKeyboard
	// transitMode is nil before the first event, then inferred.
	transitMode               *KeyDataTransitMode
	keyEventsSinceLastMessage []KeyEvent
	skippedRawKeysPressed     []PhysicalKeyboardKey
	convertListener           func(RawKeyEvent)
}

// NewKeyEventManager is KeyEventManager(hardwareKeyboard, rawKeyboard).
func NewKeyEventManager(hardwareKeyboard *HardwareKeyboard, rawKeyboard *RawKeyboard) *KeyEventManager {
	m := &KeyEventManager{hardwareKeyboard: hardwareKeyboard, rawKeyboard: rawKeyboard}
	m.convertListener = m.convertRawEventAndStore
	return m
}

// HandleKeyData is the platform's key data callback (PlatformDispatcher.
// onKeyData).
func (m *KeyEventManager) HandleKeyData(data ui.KeyData) bool {
	if m.transitMode == nil {
		mode := KeyDataTransitModeKeyDataThenRawKeyData
		m.transitMode = &mode
	}
	switch *m.transitMode {
	case KeyDataTransitModeRawKeyData:
		panic(dart.NewAssertionError("Should never encounter KeyData when transitMode is rawKeyData."))
	default:
		// Having 0 as the physical and logical ID indicates an empty key
		// data, transmitted to ensure that the transit mode is correctly
		// inferred. These events should be ignored.
		if data.Physical == 0 && data.Logical == 0 {
			return false
		}
		if !(data.Physical != 0 && data.Logical != 0) {
			panic(dart.NewAssertionError(nil))
		}
		event := keyEventFromData(data)
		if data.Synthesized && len(m.keyEventsSinceLastMessage) == 0 {
			// Dispatch the event instantly if both conditions are met: it is
			// synthesized (the result does not matter) and the queue is
			// empty (the order does not matter).
			m.hardwareKeyboard.HandleKeyEvent(event)
			m.dispatchKeyMessage([]KeyEvent{event}, nil)
		} else {
			// Otherwise, postpone key event dispatching until the next raw
			// event.
			m.keyEventsSinceLastMessage = append(m.keyEventsSinceLastMessage, event)
		}
		return false
	}
}

func (m *KeyEventManager) dispatchKeyMessage(keyEvents []KeyEvent, rawEvent RawKeyEvent) (result bool) {
	if m.KeyMessageHandler != nil {
		message := NewKeyMessage(keyEvents, rawEvent)
		defer func() {
			if exception := recover(); exception != nil {
				reportKeyError(exception, "while processing the key message handler", "KeyMessage", message)
				result = false
			}
		}()
		return m.KeyMessageHandler(message)
	}
	return false
}

// HandleRawKeyMessage is the handler of the raw key event channel
// (flutter/keyevent): it dispatches the raw event and the key events
// accumulated since the last message, and answers {"handled": bool}.
func (m *KeyEventManager) HandleRawKeyMessage(message any) async.Future[map[string]any] {
	return async.Async(func() map[string]any {
		if m.transitMode == nil {
			mode := KeyDataTransitModeRawKeyData
			m.transitMode = &mode
			// Convert raw events using a listener so that conversion only
			// occurs if the raw event should be dispatched.
			m.rawKeyboard.AddListener(m.convertListener)
		}
		rawEvent := RawKeyEventFromMessage(message.(map[string]any))
		shouldDispatch := true
		switch rawEvent.(type) {
		case *RawKeyDownEvent:
			if !rawEvent.Data().ShouldDispatchEvent() {
				shouldDispatch = false
				m.skippedRawKeysPressed = addToSet(m.skippedRawKeysPressed, rawEvent.PhysicalKey())
			} else {
				m.skippedRawKeysPressed = removeFromSet(m.skippedRawKeysPressed, rawEvent.PhysicalKey())
			}
		case *RawKeyUpEvent:
			if slices.Contains(m.skippedRawKeysPressed, rawEvent.PhysicalKey()) {
				m.skippedRawKeysPressed = removeFromSet(m.skippedRawKeysPressed, rawEvent.PhysicalKey())
				shouldDispatch = false
			}
		}
		handled := true
		if shouldDispatch {
			// handleRawKeyEvent calls convertRawEventAndStore unless the
			// event is not dispatched.
			handled = m.rawKeyboard.HandleRawKeyEvent(rawEvent)
			for _, event := range m.keyEventsSinceLastMessage {
				handled = m.hardwareKeyboard.HandleKeyEvent(event) || handled
			}
			if *m.transitMode == KeyDataTransitModeRawKeyData {
				if !foundation.SetEquals(toSet(m.rawKeyboard.PhysicalKeysPressed()), toSet(m.hardwareKeyboard.PhysicalKeysPressed())) {
					panic(dart.NewAssertionError("RawKeyboard reported " + setString(m.rawKeyboard.PhysicalKeysPressed()) +
						", while HardwareKeyboard reported " + setString(m.hardwareKeyboard.PhysicalKeysPressed())))
				}
			}
			handled = m.dispatchKeyMessage(m.keyEventsSinceLastMessage, rawEvent) || handled
			m.keyEventsSinceLastMessage = nil
		}
		return map[string]any{"handled": handled}
	})
}

func toSet[T comparable](s []T) map[T]struct{} {
	m := make(map[T]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

// setString is a Dart Set's toString: {a, b}.
func setString[T interface{ String() string }](s []T) string {
	out := "{"
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v.String()
	}
	return out + "}"
}

func addToSet[T comparable](s []T, v T) []T {
	if slices.Contains(s, v) {
		return s
	}
	return append(s, v)
}

func removeFromSet[T comparable](s []T, v T) []T {
	if i := slices.Index(s, v); i >= 0 {
		return slices.Delete(s, i, i+1)
	}
	return s
}

func convertDeviceType(rawEvent RawKeyEvent) ui.KeyEventDeviceType {
	data, ok := rawEvent.Data().(*RawKeyEventDataAndroid)
	// Device type is only available from Android.
	if !ok {
		return ui.KeyEventDeviceTypeKeyboard
	}
	switch data.EventSource {
	case 0x00000101:
		return ui.KeyEventDeviceTypeKeyboard
	case 0x00000201:
		return ui.KeyEventDeviceTypeDirectionalPad
	case 0x00000401:
		return ui.KeyEventDeviceTypeGamepad
	case 0x01000010:
		return ui.KeyEventDeviceTypeJoystick
	case 0x02000001:
		return ui.KeyEventDeviceTypeHdmi
	}
	return ui.KeyEventDeviceTypeKeyboard
}

// convertRawEventAndStore converts the raw event to key events, including
// synthesized events for modifiers, and stores them in
// keyEventsSinceLastMessage (only in raw key data mode, for events that
// are dispatched).
func (m *KeyEventManager) convertRawEventAndStore(rawEvent RawKeyEvent) {
	physicalKey := rawEvent.PhysicalKey()
	logicalKey := rawEvent.LogicalKey()
	physicalKeysPressed := m.hardwareKeyboard.PhysicalKeysPressed()
	var eventAfterwards []KeyEvent
	var mainEvent KeyEvent
	recordedLogicalMain := m.hardwareKeyboard.LookUpLayout(physicalKey)
	timeStamp := ServicesBindingInstance().CurrentSystemFrameTimeStamp()
	character := rawEvent.Character()
	if character != nil && *character == "" {
		character = nil
	}
	deviceType := convertDeviceType(rawEvent)
	if _, down := rawEvent.(*RawKeyDownEvent); down {
		if recordedLogicalMain == nil {
			mainEvent = NewKeyDownEvent(physicalKey, logicalKey, timeStamp, KeyEventOptions{Character: character, DeviceType: deviceType})
			physicalKeysPressed = addToSet(physicalKeysPressed, physicalKey)
		} else {
			if !slices.Contains(physicalKeysPressed, physicalKey) {
				panic(dart.NewAssertionError(nil))
			}
			mainEvent = NewKeyRepeatEvent(physicalKey, *recordedLogicalMain, timeStamp, KeyEventOptions{Character: character, DeviceType: deviceType})
		}
	} else {
		if _, up := rawEvent.(*RawKeyUpEvent); !up {
			panic(dart.NewAssertionError("Unexpected subclass of RawKeyEvent: " + foundation.ObjectRuntimeType(rawEvent, "")))
		}
		if recordedLogicalMain != nil {
			mainEvent = NewKeyUpEvent(physicalKey, *recordedLogicalMain, timeStamp, KeyEventOptions{DeviceType: deviceType})
			physicalKeysPressed = removeFromSet(physicalKeysPressed, physicalKey)
		}
	}
	rawPressed := m.rawKeyboard.PhysicalKeysPressed()
	for _, key := range physicalKeysPressed {
		if slices.Contains(rawPressed, key) {
			continue
		}
		if key == physicalKey {
			// Somehow, a down event is dispatched but the key is absent from
			// keysPressed. Synthesize a up event for the key, but this event
			// must be added after the main key down event.
			eventAfterwards = append(eventAfterwards, NewKeyUpEvent(key, logicalKey, timeStamp,
				KeyEventOptions{Synthesized: true, DeviceType: deviceType}))
		} else {
			m.keyEventsSinceLastMessage = append(m.keyEventsSinceLastMessage, NewKeyUpEvent(key,
				*m.hardwareKeyboard.LookUpLayout(key), timeStamp, KeyEventOptions{Synthesized: true, DeviceType: deviceType}))
		}
	}
	for _, key := range rawPressed {
		if slices.Contains(physicalKeysPressed, key) {
			continue
		}
		m.keyEventsSinceLastMessage = append(m.keyEventsSinceLastMessage, NewKeyDownEvent(key,
			*m.rawKeyboard.LookUpLayout(key), timeStamp, KeyEventOptions{Synthesized: true, DeviceType: deviceType}))
	}
	if mainEvent != nil {
		m.keyEventsSinceLastMessage = append(m.keyEventsSinceLastMessage, mainEvent)
	}
	m.keyEventsSinceLastMessage = append(m.keyEventsSinceLastMessage, eventAfterwards...)
}

// ClearState resets the manager (for tests; debug builds only, as Dart's
// assert block).
func (m *KeyEventManager) ClearState() {
	if foundation.KDebugMode {
		m.transitMode = nil
		m.rawKeyboard.RemoveListener(m.convertListener)
		m.keyEventsSinceLastMessage = nil
	}
}

func keyEventFromData(keyData ui.KeyData) KeyEvent {
	physicalKey := PhysicalKeyboardKey{keyData.Physical}
	if k := PhysicalKeyboardKeyFindKeyByCode(keyData.Physical); k != nil {
		physicalKey = *k
	}
	logicalKey := LogicalKeyboardKey{keyData.Logical}
	if k := LogicalKeyboardKeyFindKeyByKeyId(keyData.Logical); k != nil {
		logicalKey = *k
	}
	timeStamp := keyData.TimeStamp
	switch keyData.Type {
	case ui.KeyEventTypeDown:
		return NewKeyDownEvent(physicalKey, logicalKey, timeStamp, KeyEventOptions{Character: keyData.Character,
			Synthesized: keyData.Synthesized, DeviceType: keyData.DeviceType})
	case ui.KeyEventTypeUp:
		if keyData.Character != nil {
			panic(dart.NewAssertionError(nil))
		}
		return NewKeyUpEvent(physicalKey, logicalKey, timeStamp, KeyEventOptions{Synthesized: keyData.Synthesized,
			DeviceType: keyData.DeviceType})
	default:
		return NewKeyRepeatEvent(physicalKey, logicalKey, timeStamp, KeyEventOptions{Character: keyData.Character,
			DeviceType: keyData.DeviceType})
	}
}
