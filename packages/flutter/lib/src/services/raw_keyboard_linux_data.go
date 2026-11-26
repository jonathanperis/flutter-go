package services

// Port of raw_keyboard_linux.dart (Flutter 3.50.1).

import (
	"reflect"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// RawKeyEventDataLinux is the platform data of a Linux key event (GLFW or
// GTK, see KeyHelper; deprecated). Compare with Equal.
type RawKeyEventDataLinux struct {
	RawKeyEventDataBase
	// KeyHelper interprets the toolkit's codes.
	KeyHelper           KeyHelper
	UnicodeScalarValues int
	ScanCode            int
	KeyCode             int
	Modifiers           int
	// IsDown is whether the event is a key down.
	IsDown bool
	// SpecifiedLogicalKey is the logical key the embedder specified (nil:
	// none).
	SpecifiedLogicalKey *int
}

// RawKeyEventDataLinuxOptions are its optional named parameters.
type RawKeyEventDataLinuxOptions struct {
	UnicodeScalarValues, ScanCode, KeyCode, Modifiers int
	SpecifiedLogicalKey                               *int
}

// NewRawKeyEventDataLinux is RawKeyEventDataLinux({required keyHelper,
// ..., required isDown, ...}) (the required parameters are positional).
func NewRawKeyEventDataLinux(keyHelper KeyHelper, isDown bool, opts ...RawKeyEventDataLinuxOptions) *RawKeyEventDataLinux {
	var o RawKeyEventDataLinuxOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.UnicodeScalarValues&^LogicalKeyboardKeyValueMask != 0 {
		panic(dart.NewAssertionError(nil))
	}
	d := &RawKeyEventDataLinux{KeyHelper: keyHelper, UnicodeScalarValues: o.UnicodeScalarValues, ScanCode: o.ScanCode,
		KeyCode: o.KeyCode, Modifiers: o.Modifiers, IsDown: isDown, SpecifiedLogicalKey: o.SpecifiedLogicalKey}
	d.SetSelf(d)
	return d
}

func (d *RawKeyEventDataLinux) KeyLabel() string {
	if d.UnicodeScalarValues == 0 {
		return ""
	}
	return dart.StringFromCharCode(d.UnicodeScalarValues)
}

func (d *RawKeyEventDataLinux) PhysicalKey() PhysicalKeyboardKey {
	if k, ok := KLinuxToPhysicalKey[d.ScanCode]; ok {
		return k
	}
	return PhysicalKeyboardKey{LogicalKeyboardKeyWebPlane + d.ScanCode}
}

func (d *RawKeyEventDataLinux) LogicalKey() LogicalKeyboardKey {
	if d.SpecifiedLogicalKey != nil {
		return findLogicalKeyOrNew(*d.SpecifiedLogicalKey)
	}
	if numPadKey := d.KeyHelper.NumpadKey(d.KeyCode); numPadKey != nil {
		return *numPadKey
	}
	keyLabel := d.KeyLabel()
	if keyLabel != "" && !LogicalKeyboardKeyIsControlCharacter(keyLabel) {
		keyID := LogicalKeyboardKeyUnicodePlane | (d.UnicodeScalarValues & LogicalKeyboardKeyValueMask)
		return findLogicalKeyOrNew(keyID)
	}
	if newKey := d.KeyHelper.LogicalKey(d.KeyCode); newKey != nil {
		return *newKey
	}
	// A non-printable key we don't know about: mint a new code.
	return LogicalKeyboardKey{d.KeyCode | d.KeyHelper.PlatformPlane()}
}

func (d *RawKeyEventDataLinux) IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool {
	return d.KeyHelper.IsModifierPressed(key, d.Modifiers, d.KeyCode, d.IsDown, opts...)
}

func (d *RawKeyEventDataLinux) GetModifierSide(key ModifierKey) *KeyboardSide {
	return sidePtr(d.KeyHelper.GetModifierSide(key))
}

func (d *RawKeyEventDataLinux) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.RawKeyEventDataBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("toolkit", d.KeyHelper.DebugToolkit()))
	properties.Add(foundation.NewDiagnosticsProperty("unicodeScalarValues", d.UnicodeScalarValues))
	properties.Add(foundation.NewDiagnosticsProperty("scanCode", d.ScanCode))
	properties.Add(foundation.NewDiagnosticsProperty("keyCode", d.KeyCode))
	properties.Add(foundation.NewDiagnosticsProperty("modifiers", d.Modifiers))
	properties.Add(foundation.NewDiagnosticsProperty("isDown", d.IsDown))
	properties.Add(foundation.NewDiagnosticsProperty("specifiedLogicalKey", d.SpecifiedLogicalKey,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
}

// Equal is Dart's == (the key helpers compare by type).
func (d *RawKeyEventDataLinux) Equal(other RawKeyEventData) bool {
	o, ok := other.(*RawKeyEventDataLinux)
	return ok && reflect.TypeOf(o.KeyHelper) == reflect.TypeOf(d.KeyHelper) &&
		o.UnicodeScalarValues == d.UnicodeScalarValues && o.ScanCode == d.ScanCode && o.KeyCode == d.KeyCode &&
		o.Modifiers == d.Modifiers && o.IsDown == d.IsDown
}

// KeyHelper abstracts the key codes and modifiers of a Linux window
// toolkit (deprecated): GLFWKeyHelper or GtkKeyHelper.
type KeyHelper interface {
	// DebugToolkit is the toolkit's name, for debugging.
	DebugToolkit() string
	// GetModifierSide is the side of the modifier.
	GetModifierSide(key ModifierKey) KeyboardSide
	// IsModifierPressed is whether the modifier is pressed (Dart's required
	// named keyCode and isDown are positional).
	IsModifierPressed(key ModifierKey, modifiers, keyCode int, isDown bool, opts ...IsModifierPressedOptions) bool
	// NumpadKey is the numpad key with the key code, or nil.
	NumpadKey(keyCode int) *LogicalKeyboardKey
	// LogicalKey is the logical key with the key code, or nil.
	LogicalKey(keyCode int) *LogicalKeyboardKey
	// PlatformPlane is the plane of the toolkit's private keys.
	PlatformPlane() int
}

// NewKeyHelper is the KeyHelper(toolkit) factory: "glfw" or "gtk" (any
// other toolkit panics with a FlutterError).
func NewKeyHelper(toolkit string) KeyHelper {
	switch toolkit {
	case "glfw":
		return GLFWKeyHelper{}
	case "gtk":
		return GtkKeyHelper{}
	}
	panic(foundation.NewFlutterError("Window toolkit not recognized: " + toolkit))
}

func lookUpLogical(m map[int]LogicalKeyboardKey, keyCode int) *LogicalKeyboardKey {
	if k, ok := m[keyCode]; ok {
		return &k
	}
	return nil
}

// GLFWKeyHelper is the GLFW toolkit's KeyHelper.
type GLFWKeyHelper struct{}

// GLFW modifier masks (GLFWKeyHelper's static constants).
const (
	GLFWKeyHelperModifierCapsLock   = 0x0010
	GLFWKeyHelperModifierShift      = 0x0001
	GLFWKeyHelperModifierControl    = 0x0002
	GLFWKeyHelperModifierAlt        = 0x0004
	GLFWKeyHelperModifierMeta       = 0x0008
	GLFWKeyHelperModifierNumericPad = 0x0020
)

func (GLFWKeyHelper) DebugToolkit() string { return "GLFW" }

func (GLFWKeyHelper) mergeModifiers(modifiers, keyCode int, isDown bool) int {
	// GLFW key codes for modifier keys.
	const (
		shiftLeftKeyCode    = 340
		shiftRightKeyCode   = 344
		controlLeftKeyCode  = 341
		controlRightKeyCode = 345
		altLeftKeyCode      = 342
		altRightKeyCode     = 346
		metaLeftKeyCode     = 343
		metaRightKeyCode    = 347
		capsLockKeyCode     = 280
		numLockKeyCode      = 282
	)
	// On GLFW, the modifiers are the state BEFORE this event, so a key down
	// adds its modifier bits and a key up removes them.
	modifierChange := 0
	switch keyCode {
	case shiftLeftKeyCode, shiftRightKeyCode:
		modifierChange = GLFWKeyHelperModifierShift
	case controlLeftKeyCode, controlRightKeyCode:
		modifierChange = GLFWKeyHelperModifierControl
	case altLeftKeyCode, altRightKeyCode:
		modifierChange = GLFWKeyHelperModifierAlt
	case metaLeftKeyCode, metaRightKeyCode:
		modifierChange = GLFWKeyHelperModifierMeta
	case capsLockKeyCode:
		modifierChange = GLFWKeyHelperModifierCapsLock
	case numLockKeyCode:
		modifierChange = GLFWKeyHelperModifierNumericPad
	}
	if isDown {
		return modifiers | modifierChange
	}
	return modifiers &^ modifierChange
}

func (h GLFWKeyHelper) IsModifierPressed(key ModifierKey, modifiers, keyCode int, isDown bool, opts ...IsModifierPressedOptions) bool {
	modifiers = h.mergeModifiers(modifiers, keyCode, isDown)
	switch key {
	case ModifierKeyControlModifier:
		return modifiers&GLFWKeyHelperModifierControl != 0
	case ModifierKeyShiftModifier:
		return modifiers&GLFWKeyHelperModifierShift != 0
	case ModifierKeyAltModifier:
		return modifiers&GLFWKeyHelperModifierAlt != 0
	case ModifierKeyMetaModifier:
		return modifiers&GLFWKeyHelperModifierMeta != 0
	case ModifierKeyCapsLockModifier:
		return modifiers&GLFWKeyHelperModifierCapsLock != 0
	case ModifierKeyNumLockModifier:
		return modifiers&GLFWKeyHelperModifierNumericPad != 0
	}
	// Function, symbol and scroll lock are not used in GLFW keyboards.
	return false
}

// GetModifierSide: neither GLFW nor X11 distinguish the sides.
func (GLFWKeyHelper) GetModifierSide(ModifierKey) KeyboardSide { return KeyboardSideAll }

func (GLFWKeyHelper) NumpadKey(keyCode int) *LogicalKeyboardKey {
	return lookUpLogical(KGlfwNumpadMap, keyCode)
}
func (GLFWKeyHelper) LogicalKey(keyCode int) *LogicalKeyboardKey {
	return lookUpLogical(KGlfwToLogicalKey, keyCode)
}
func (GLFWKeyHelper) PlatformPlane() int { return LogicalKeyboardKeyGlfwPlane }

// GtkKeyHelper is the GTK toolkit's KeyHelper.
type GtkKeyHelper struct{}

// GTK modifier masks (GtkKeyHelper's static constants).
const (
	GtkKeyHelperModifierShift    = 1 << 0
	GtkKeyHelperModifierCapsLock = 1 << 1
	GtkKeyHelperModifierControl  = 1 << 2
	GtkKeyHelperModifierMod1     = 1 << 3
	GtkKeyHelperModifierMod2     = 1 << 4
	GtkKeyHelperModifierMeta     = 1 << 26
)

func (GtkKeyHelper) DebugToolkit() string { return "GTK" }

func (GtkKeyHelper) mergeModifiers(modifiers, keyCode int, isDown bool) int {
	// GTK key codes for modifier keys.
	const (
		shiftLeftKeyCode    = 0xffe1
		shiftRightKeyCode   = 0xffe2
		controlLeftKeyCode  = 0xffe3
		controlRightKeyCode = 0xffe4
		capsLockKeyCode     = 0xffe5
		shiftLockKeyCode    = 0xffe6
		altLeftKeyCode      = 0xffe9
		altRightKeyCode     = 0xffea
		metaLeftKeyCode     = 0xffeb
		metaRightKeyCode    = 0xffec
		numLockKeyCode      = 0xff7f
	)
	modifierChange := 0
	switch keyCode {
	case shiftLeftKeyCode, shiftRightKeyCode:
		modifierChange = GtkKeyHelperModifierShift
	case controlLeftKeyCode, controlRightKeyCode:
		modifierChange = GtkKeyHelperModifierControl
	case altLeftKeyCode, altRightKeyCode:
		modifierChange = GtkKeyHelperModifierMod1
	case metaLeftKeyCode, metaRightKeyCode:
		modifierChange = GtkKeyHelperModifierMeta
	case capsLockKeyCode, shiftLockKeyCode:
		modifierChange = GtkKeyHelperModifierCapsLock
	case numLockKeyCode:
		modifierChange = GtkKeyHelperModifierMod2
	}
	if isDown {
		return modifiers | modifierChange
	}
	return modifiers &^ modifierChange
}

func (h GtkKeyHelper) IsModifierPressed(key ModifierKey, modifiers, keyCode int, isDown bool, opts ...IsModifierPressedOptions) bool {
	modifiers = h.mergeModifiers(modifiers, keyCode, isDown)
	switch key {
	case ModifierKeyControlModifier:
		return modifiers&GtkKeyHelperModifierControl != 0
	case ModifierKeyShiftModifier:
		return modifiers&GtkKeyHelperModifierShift != 0
	case ModifierKeyAltModifier:
		return modifiers&GtkKeyHelperModifierMod1 != 0
	case ModifierKeyMetaModifier:
		return modifiers&GtkKeyHelperModifierMeta != 0
	case ModifierKeyCapsLockModifier:
		return modifiers&GtkKeyHelperModifierCapsLock != 0
	case ModifierKeyNumLockModifier:
		return modifiers&GtkKeyHelperModifierMod2 != 0
	}
	// Function, symbol and scroll lock are not used in GTK keyboards.
	return false
}

// GetModifierSide: neither GTK nor X11 distinguish the sides.
func (GtkKeyHelper) GetModifierSide(ModifierKey) KeyboardSide { return KeyboardSideAll }

func (GtkKeyHelper) NumpadKey(keyCode int) *LogicalKeyboardKey {
	return lookUpLogical(KGtkNumpadMap, keyCode)
}
func (GtkKeyHelper) LogicalKey(keyCode int) *LogicalKeyboardKey {
	return lookUpLogical(KGtkToLogicalKey, keyCode)
}
func (GtkKeyHelper) PlatformPlane() int { return LogicalKeyboardKeyGtkPlane }
