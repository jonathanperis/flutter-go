package services

// Port of raw_keyboard_windows.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// vkProcessKey is the Win32 virtual key VK_PROCESSKEY (IME key downs).
const vkProcessKey = 0xe5

// RawKeyEventDataWindows is the platform data of a Windows key event
// (deprecated). Compare with Equal.
type RawKeyEventDataWindows struct {
	RawKeyEventDataBase
	KeyCode            int
	ScanCode           int
	CharacterCodePoint int
	Modifiers          int
}

// RawKeyEventDataWindowsOptions are its named parameters (default 0).
type RawKeyEventDataWindowsOptions struct{ KeyCode, ScanCode, CharacterCodePoint, Modifiers int }

// NewRawKeyEventDataWindows is RawKeyEventDataWindows({...}).
func NewRawKeyEventDataWindows(opts ...RawKeyEventDataWindowsOptions) *RawKeyEventDataWindows {
	var o RawKeyEventDataWindowsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &RawKeyEventDataWindows{KeyCode: o.KeyCode, ScanCode: o.ScanCode, CharacterCodePoint: o.CharacterCodePoint, Modifiers: o.Modifiers}
	d.SetSelf(d)
	return d
}

func (d *RawKeyEventDataWindows) KeyLabel() string {
	if d.CharacterCodePoint == 0 {
		return ""
	}
	return dart.StringFromCharCode(d.CharacterCodePoint)
}

func (d *RawKeyEventDataWindows) PhysicalKey() PhysicalKeyboardKey {
	if k, ok := KWindowsToPhysicalKey[d.ScanCode]; ok {
		return k
	}
	return PhysicalKeyboardKey{LogicalKeyboardKeyWindowsPlane + d.ScanCode}
}

func (d *RawKeyEventDataWindows) LogicalKey() LogicalKeyboardKey {
	if numPadKey, ok := KWindowsNumPadMap[d.KeyCode]; ok {
		return numPadKey
	}
	keyLabel := d.KeyLabel()
	if keyLabel != "" && !LogicalKeyboardKeyIsControlCharacter(keyLabel) {
		keyID := LogicalKeyboardKeyUnicodePlane | (d.CharacterCodePoint & LogicalKeyboardKeyValueMask)
		return findLogicalKeyOrNew(keyID)
	}
	if newKey, ok := KWindowsToLogicalKey[d.KeyCode]; ok {
		return newKey
	}
	return LogicalKeyboardKey{d.KeyCode | LogicalKeyboardKeyWindowsPlane}
}

func (d *RawKeyEventDataWindows) isLeftRightModifierPressed(side KeyboardSide, anyMask, leftMask, rightMask int) bool {
	if d.Modifiers&(leftMask|rightMask|anyMask) == 0 {
		return false
	}
	if d.Modifiers&(leftMask|rightMask|anyMask) == anyMask {
		// Only the "any" bit: true for either side.
		return true
	}
	switch side {
	case KeyboardSideAny:
		return true
	case KeyboardSideAll:
		return d.Modifiers&leftMask != 0 && d.Modifiers&rightMask != 0
	case KeyboardSideLeft:
		return d.Modifiers&leftMask != 0
	default:
		return d.Modifiers&rightMask != 0
	}
}

func (d *RawKeyEventDataWindows) IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool {
	side := modifierSide(opts)
	var result bool
	switch key {
	case ModifierKeyControlModifier:
		result = d.isLeftRightModifierPressed(side, RawKeyEventDataWindowsModifierControl, RawKeyEventDataWindowsModifierLeftControl, RawKeyEventDataWindowsModifierRightControl)
	case ModifierKeyShiftModifier:
		result = d.isLeftRightModifierPressed(side, RawKeyEventDataWindowsModifierShift, RawKeyEventDataWindowsModifierLeftShift, RawKeyEventDataWindowsModifierRightShift)
	case ModifierKeyAltModifier:
		result = d.isLeftRightModifierPressed(side, RawKeyEventDataWindowsModifierAlt, RawKeyEventDataWindowsModifierLeftAlt, RawKeyEventDataWindowsModifierRightAlt)
	case ModifierKeyMetaModifier:
		// Windows does not provide an "any" key for win key press.
		result = d.isLeftRightModifierPressed(side, RawKeyEventDataWindowsModifierLeftMeta|RawKeyEventDataWindowsModifierRightMeta,
			RawKeyEventDataWindowsModifierLeftMeta, RawKeyEventDataWindowsModifierRightMeta)
	case ModifierKeyCapsLockModifier:
		result = d.Modifiers&RawKeyEventDataWindowsModifierCaps != 0
	case ModifierKeyScrollLockModifier:
		result = d.Modifiers&RawKeyEventDataWindowsModifierScrollLock != 0
	case ModifierKeyNumLockModifier:
		result = d.Modifiers&RawKeyEventDataWindowsModifierNumLock != 0
	default:
		// The OS does not expose the Fn key; symbol is not used.
		result = false
	}
	if result && d.GetModifierSide(key) == nil {
		panic(dart.NewAssertionError(foundation.ObjectRuntimeType(d, "") +
			" thinks that a modifier is pressed, but can't figure out what side it's on."))
	}
	return result
}

func (d *RawKeyEventDataWindows) GetModifierSide(key ModifierKey) *KeyboardSide {
	findSide := func(leftMask, rightMask, anyMask int) *KeyboardSide {
		combinedMask := leftMask | rightMask
		combined := d.Modifiers & combinedMask
		switch {
		case combined == leftMask:
			return sidePtr(KeyboardSideLeft)
		case combined == rightMask:
			return sidePtr(KeyboardSideRight)
		case combined == combinedMask || d.Modifiers&(combinedMask|anyMask) == anyMask:
			return sidePtr(KeyboardSideAll)
		}
		return nil
	}
	switch key {
	case ModifierKeyControlModifier:
		return findSide(RawKeyEventDataWindowsModifierLeftControl, RawKeyEventDataWindowsModifierRightControl, RawKeyEventDataWindowsModifierControl)
	case ModifierKeyShiftModifier:
		return findSide(RawKeyEventDataWindowsModifierLeftShift, RawKeyEventDataWindowsModifierRightShift, RawKeyEventDataWindowsModifierShift)
	case ModifierKeyAltModifier:
		return findSide(RawKeyEventDataWindowsModifierLeftAlt, RawKeyEventDataWindowsModifierRightAlt, RawKeyEventDataWindowsModifierAlt)
	case ModifierKeyMetaModifier:
		return findSide(RawKeyEventDataWindowsModifierLeftMeta, RawKeyEventDataWindowsModifierRightMeta, 0)
	}
	return sidePtr(KeyboardSideAll)
}

// ShouldDispatchEvent: IME key downs (VK_PROCESSKEY) and their key ups are
// skipped (the text input API handles IME operations).
func (d *RawKeyEventDataWindows) ShouldDispatchEvent() bool { return d.KeyCode != vkProcessKey }

func (d *RawKeyEventDataWindows) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.RawKeyEventDataBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("keyCode", d.KeyCode))
	properties.Add(foundation.NewDiagnosticsProperty("scanCode", d.ScanCode))
	properties.Add(foundation.NewDiagnosticsProperty("characterCodePoint", d.CharacterCodePoint))
	properties.Add(foundation.NewDiagnosticsProperty("modifiers", d.Modifiers))
}

// Equal is Dart's ==.
func (d *RawKeyEventDataWindows) Equal(other RawKeyEventData) bool {
	o, ok := other.(*RawKeyEventDataWindows)
	return ok && o.KeyCode == d.KeyCode && o.ScanCode == d.ScanCode && o.CharacterCodePoint == d.CharacterCodePoint &&
		o.Modifiers == d.Modifiers
}

// Modifier masks (Flutter's own, not Win32's; see the Windows embedder's
// keyboard_key_channel_handler.cc).
const (
	RawKeyEventDataWindowsModifierShift        = 1 << 0
	RawKeyEventDataWindowsModifierLeftShift    = 1 << 1
	RawKeyEventDataWindowsModifierRightShift   = 1 << 2
	RawKeyEventDataWindowsModifierControl      = 1 << 3
	RawKeyEventDataWindowsModifierLeftControl  = 1 << 4
	RawKeyEventDataWindowsModifierRightControl = 1 << 5
	RawKeyEventDataWindowsModifierAlt          = 1 << 6
	RawKeyEventDataWindowsModifierLeftAlt      = 1 << 7
	RawKeyEventDataWindowsModifierRightAlt     = 1 << 8
	RawKeyEventDataWindowsModifierLeftMeta     = 1 << 9
	RawKeyEventDataWindowsModifierRightMeta    = 1 << 10
	RawKeyEventDataWindowsModifierCaps         = 1 << 11
	RawKeyEventDataWindowsModifierNumLock      = 1 << 12
	RawKeyEventDataWindowsModifierScrollLock   = 1 << 13
)
