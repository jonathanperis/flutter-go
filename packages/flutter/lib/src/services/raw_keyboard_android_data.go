package services

// Port of raw_keyboard_android.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Android sets the 0x80000000 bit on a character to indicate that it is a
// combining character, so this mask removes that bit.
const kCombiningCharacterMask = 0x7fffffff

// RawKeyEventDataAndroid is the platform data of an Android key event
// (deprecated). Compare with Equal (Dart's ==).
type RawKeyEventDataAndroid struct {
	RawKeyEventDataBase
	Flags          int
	CodePoint      int
	PlainCodePoint int
	KeyCode        int
	ScanCode       int
	MetaState      int
	EventSource    int
	VendorId       int
	ProductId      int
	DeviceId       int
	RepeatCount    int
}

// RawKeyEventDataAndroidOptions are RawKeyEventDataAndroid's named
// parameters (all default to 0).
type RawKeyEventDataAndroidOptions struct {
	Flags, CodePoint, PlainCodePoint, KeyCode, ScanCode, MetaState, EventSource, VendorId, ProductId, DeviceId, RepeatCount int
}

// NewRawKeyEventDataAndroid is RawKeyEventDataAndroid({...}).
func NewRawKeyEventDataAndroid(opts ...RawKeyEventDataAndroidOptions) *RawKeyEventDataAndroid {
	var o RawKeyEventDataAndroidOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &RawKeyEventDataAndroid{Flags: o.Flags, CodePoint: o.CodePoint, PlainCodePoint: o.PlainCodePoint,
		KeyCode: o.KeyCode, ScanCode: o.ScanCode, MetaState: o.MetaState, EventSource: o.EventSource,
		VendorId: o.VendorId, ProductId: o.ProductId, DeviceId: o.DeviceId, RepeatCount: o.RepeatCount}
	d.SetSelf(d)
	return d
}

// The source code that indicates that an event came from a joystick.
const androidSourceJoystick = 0x01000010

func (d *RawKeyEventDataAndroid) KeyLabel() string {
	if d.PlainCodePoint == 0 {
		return ""
	}
	return dart.StringFromCharCode(d.PlainCodePoint & kCombiningCharacterMask)
}

func (d *RawKeyEventDataAndroid) PhysicalKey() PhysicalKeyboardKey {
	if k, ok := KAndroidToPhysicalKey[d.ScanCode]; ok {
		return k
	}
	// Android sends DPAD_UP, etc. as the keyCode for joystick DPAD events,
	// but it doesn't set the scanCode for those.
	if d.EventSource&androidSourceJoystick == androidSourceJoystick {
		foundKey, ok := KAndroidToLogicalKey[d.KeyCode]
		if ok {
			switch foundKey {
			case LogicalKeyboardKeyArrowUp:
				return PhysicalKeyboardKeyArrowUp
			case LogicalKeyboardKeyArrowDown:
				return PhysicalKeyboardKeyArrowDown
			case LogicalKeyboardKeyArrowLeft:
				return PhysicalKeyboardKeyArrowLeft
			case LogicalKeyboardKeyArrowRight:
				return PhysicalKeyboardKeyArrowRight
			}
		}
	}
	return PhysicalKeyboardKey{LogicalKeyboardKeyAndroidPlane + d.ScanCode}
}

func (d *RawKeyEventDataAndroid) LogicalKey() LogicalKeyboardKey {
	// A printable number pad key, so that "=" and the number pad's "=" can
	// be told apart.
	if numPadKey, ok := KAndroidNumPadMap[d.KeyCode]; ok {
		return numPadKey
	}
	// A non-control-character label: the existing constant, or a new
	// Unicode-based key.
	keyLabel := d.KeyLabel()
	if keyLabel != "" && !LogicalKeyboardKeyIsControlCharacter(keyLabel) {
		combinedCodePoint := d.PlainCodePoint & kCombiningCharacterMask
		keyID := LogicalKeyboardKeyUnicodePlane | (combinedCodePoint & LogicalKeyboardKeyValueMask)
		return findLogicalKeyOrNew(keyID)
	}
	// A keyCode we have a mapping for.
	if newKey, ok := KAndroidToLogicalKey[d.KeyCode]; ok {
		return newKey
	}
	return LogicalKeyboardKey{d.KeyCode | LogicalKeyboardKeyAndroidPlane}
}

// findLogicalKeyOrNew is `findKeyByKeyId(keyId) ?? LogicalKeyboardKey(keyId)`.
func findLogicalKeyOrNew(keyID int) LogicalKeyboardKey {
	if k := LogicalKeyboardKeyFindKeyByKeyId(keyID); k != nil {
		return *k
	}
	return LogicalKeyboardKey{keyID}
}

func (d *RawKeyEventDataAndroid) isLeftRightModifierPressed(side KeyboardSide, anyMask, leftMask, rightMask int) bool {
	if d.MetaState&anyMask == 0 {
		return false
	}
	switch side {
	case KeyboardSideAny:
		return true
	case KeyboardSideAll:
		return d.MetaState&leftMask != 0 && d.MetaState&rightMask != 0
	case KeyboardSideLeft:
		return d.MetaState&leftMask != 0
	default:
		return d.MetaState&rightMask != 0
	}
}

func (d *RawKeyEventDataAndroid) IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool {
	side := modifierSide(opts)
	switch key {
	case ModifierKeyControlModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataAndroidModifierControl, RawKeyEventDataAndroidModifierLeftControl, RawKeyEventDataAndroidModifierRightControl)
	case ModifierKeyShiftModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataAndroidModifierShift, RawKeyEventDataAndroidModifierLeftShift, RawKeyEventDataAndroidModifierRightShift)
	case ModifierKeyAltModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataAndroidModifierAlt, RawKeyEventDataAndroidModifierLeftAlt, RawKeyEventDataAndroidModifierRightAlt)
	case ModifierKeyMetaModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataAndroidModifierMeta, RawKeyEventDataAndroidModifierLeftMeta, RawKeyEventDataAndroidModifierRightMeta)
	case ModifierKeyCapsLockModifier:
		return d.MetaState&RawKeyEventDataAndroidModifierCapsLock != 0
	case ModifierKeyNumLockModifier:
		return d.MetaState&RawKeyEventDataAndroidModifierNumLock != 0
	case ModifierKeyScrollLockModifier:
		return d.MetaState&RawKeyEventDataAndroidModifierScrollLock != 0
	case ModifierKeyFunctionModifier:
		return d.MetaState&RawKeyEventDataAndroidModifierFunction != 0
	default: // symbolModifier
		return d.MetaState&RawKeyEventDataAndroidModifierSym != 0
	}
}

func sidePtr(s KeyboardSide) *KeyboardSide { return &s }

func (d *RawKeyEventDataAndroid) GetModifierSide(key ModifierKey) *KeyboardSide {
	findSide := func(anyMask, leftMask, rightMask int) *KeyboardSide {
		combinedMask := leftMask | rightMask
		combined := d.MetaState & combinedMask
		switch {
		case combined == leftMask:
			return sidePtr(KeyboardSideLeft)
		case combined == rightMask:
			return sidePtr(KeyboardSideRight)
		case combined == combinedMask:
			return sidePtr(KeyboardSideAll)
		}
		// If the platform code sets the "any" modifier, but not a specific
		// side, then we return "all", assuming that there is only one of
		// that modifier key on the keyboard.
		if d.MetaState&anyMask != 0 {
			return sidePtr(KeyboardSideAll)
		}
		return nil
	}
	switch key {
	case ModifierKeyControlModifier:
		return findSide(RawKeyEventDataAndroidModifierControl, RawKeyEventDataAndroidModifierLeftControl, RawKeyEventDataAndroidModifierRightControl)
	case ModifierKeyShiftModifier:
		return findSide(RawKeyEventDataAndroidModifierShift, RawKeyEventDataAndroidModifierLeftShift, RawKeyEventDataAndroidModifierRightShift)
	case ModifierKeyAltModifier:
		return findSide(RawKeyEventDataAndroidModifierAlt, RawKeyEventDataAndroidModifierLeftAlt, RawKeyEventDataAndroidModifierRightAlt)
	case ModifierKeyMetaModifier:
		return findSide(RawKeyEventDataAndroidModifierMeta, RawKeyEventDataAndroidModifierLeftMeta, RawKeyEventDataAndroidModifierRightMeta)
	}
	return sidePtr(KeyboardSideAll)
}

func (d *RawKeyEventDataAndroid) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.RawKeyEventDataBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("flags", d.Flags))
	properties.Add(foundation.NewDiagnosticsProperty("codePoint", d.CodePoint))
	properties.Add(foundation.NewDiagnosticsProperty("plainCodePoint", d.PlainCodePoint))
	properties.Add(foundation.NewDiagnosticsProperty("keyCode", d.KeyCode))
	properties.Add(foundation.NewDiagnosticsProperty("scanCode", d.ScanCode))
	properties.Add(foundation.NewDiagnosticsProperty("metaState", d.MetaState))
}

// Equal is Dart's == (flags, code points, key code, scan code and meta
// state).
func (d *RawKeyEventDataAndroid) Equal(other RawKeyEventData) bool {
	o, ok := other.(*RawKeyEventDataAndroid)
	return ok && o.Flags == d.Flags && o.CodePoint == d.CodePoint && o.PlainCodePoint == d.PlainCodePoint &&
		o.KeyCode == d.KeyCode && o.ScanCode == d.ScanCode && o.MetaState == d.MetaState
}

// Modifier key masks (RawKeyEventDataAndroid's static constants).
const (
	RawKeyEventDataAndroidModifierNone         = 0
	RawKeyEventDataAndroidModifierAlt          = 0x02
	RawKeyEventDataAndroidModifierLeftAlt      = 0x10
	RawKeyEventDataAndroidModifierRightAlt     = 0x20
	RawKeyEventDataAndroidModifierShift        = 0x01
	RawKeyEventDataAndroidModifierLeftShift    = 0x40
	RawKeyEventDataAndroidModifierRightShift   = 0x80
	RawKeyEventDataAndroidModifierSym          = 0x04
	RawKeyEventDataAndroidModifierFunction     = 0x08
	RawKeyEventDataAndroidModifierControl      = 0x1000
	RawKeyEventDataAndroidModifierLeftControl  = 0x2000
	RawKeyEventDataAndroidModifierRightControl = 0x4000
	RawKeyEventDataAndroidModifierMeta         = 0x10000
	RawKeyEventDataAndroidModifierLeftMeta     = 0x20000
	RawKeyEventDataAndroidModifierRightMeta    = 0x40000
	RawKeyEventDataAndroidModifierCapsLock     = 0x100000
	RawKeyEventDataAndroidModifierNumLock      = 0x200000
	RawKeyEventDataAndroidModifierScrollLock   = 0x400000
)
