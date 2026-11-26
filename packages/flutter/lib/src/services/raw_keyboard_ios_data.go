package services

// Port of raw_keyboard_ios.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// RawKeyEventDataIos is the platform data of an iOS key event
// (deprecated). Compare with Equal.
type RawKeyEventDataIos struct {
	RawKeyEventDataBase
	Characters                  string
	CharactersIgnoringModifiers string
	KeyCode                     int
	Modifiers                   int
}

// RawKeyEventDataIosOptions are its named parameters.
type RawKeyEventDataIosOptions struct {
	Characters, CharactersIgnoringModifiers string
	KeyCode, Modifiers                      int
}

// NewRawKeyEventDataIos is RawKeyEventDataIos({...}).
func NewRawKeyEventDataIos(opts ...RawKeyEventDataIosOptions) *RawKeyEventDataIos {
	var o RawKeyEventDataIosOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &RawKeyEventDataIos{Characters: o.Characters, CharactersIgnoringModifiers: o.CharactersIgnoringModifiers,
		KeyCode: o.KeyCode, Modifiers: o.Modifiers}
	d.SetSelf(d)
	return d
}

func (d *RawKeyEventDataIos) KeyLabel() string { return d.CharactersIgnoringModifiers }

func (d *RawKeyEventDataIos) PhysicalKey() PhysicalKeyboardKey {
	if k, ok := KIosToPhysicalKey[d.KeyCode]; ok {
		return k
	}
	return PhysicalKeyboardKey{LogicalKeyboardKeyIosPlane + d.KeyCode}
}

func (d *RawKeyEventDataIos) LogicalKey() LogicalKeyboardKey {
	if numPadKey, ok := KIosNumPadMap[d.KeyCode]; ok {
		return numPadKey
	}
	// A keyLabel we know about.
	if specialKey, ok := KIosSpecialLogicalMap[d.KeyLabel()]; ok {
		return specialKey
	}
	// Keys that can't be derived with charactersIgnoringModifiers.
	if knownKey, ok := KIosToLogicalKey[d.KeyCode]; ok {
		return knownKey
	}
	// A printable key generates the key from its Unicode value (at most
	// two code units).
	keyLabel := d.KeyLabel()
	if keyLabel != "" && !LogicalKeyboardKeyIsControlCharacter(keyLabel) && !appleIsUnprintableKey(keyLabel) {
		n := dart.StringLength(d.CharactersIgnoringModifiers)
		if n > 2 {
			panic(dart.NewAssertionError(nil))
		}
		codeUnit := dart.CodeUnitAt(d.CharactersIgnoringModifiers, 0)
		if n == 2 {
			secondCode := dart.CodeUnitAt(d.CharactersIgnoringModifiers, 1)
			codeUnit = (codeUnit << 16) | secondCode
		}
		keyID := LogicalKeyboardKeyUnicodePlane | (codeUnit & LogicalKeyboardKeyValueMask)
		return findLogicalKeyOrNew(keyID)
	}
	// A non-printable key we don't know about: mint a new code.
	return LogicalKeyboardKey{d.KeyCode | LogicalKeyboardKeyIosPlane}
}

func (d *RawKeyEventDataIos) IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool {
	return appleIsModifierPressed(d, d.Modifiers, key, modifierSide(opts))
}

func (d *RawKeyEventDataIos) GetModifierSide(key ModifierKey) *KeyboardSide {
	return appleGetModifierSide(d.Modifiers, key)
}

func (d *RawKeyEventDataIos) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.RawKeyEventDataBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("characters", d.Characters))
	properties.Add(foundation.NewDiagnosticsProperty("charactersIgnoringModifiers", d.CharactersIgnoringModifiers))
	properties.Add(foundation.NewDiagnosticsProperty("keyCode", d.KeyCode))
	properties.Add(foundation.NewDiagnosticsProperty("modifiers", d.Modifiers))
}

// Equal is Dart's ==.
func (d *RawKeyEventDataIos) Equal(other RawKeyEventData) bool {
	o, ok := other.(*RawKeyEventDataIos)
	return ok && o.Characters == d.Characters && o.CharactersIgnoringModifiers == d.CharactersIgnoringModifiers &&
		o.KeyCode == d.KeyCode && o.Modifiers == d.Modifiers
}

// Modifier key masks (UIKeyModifierFlags and IOLLEvent.h).
const (
	RawKeyEventDataIosModifierCapsLock      = 0x10000
	RawKeyEventDataIosModifierShift         = 0x20000
	RawKeyEventDataIosModifierLeftShift     = 0x02
	RawKeyEventDataIosModifierRightShift    = 0x04
	RawKeyEventDataIosModifierControl       = 0x40000
	RawKeyEventDataIosModifierLeftControl   = 0x01
	RawKeyEventDataIosModifierRightControl  = 0x2000
	RawKeyEventDataIosModifierOption        = 0x80000
	RawKeyEventDataIosModifierLeftOption    = 0x20
	RawKeyEventDataIosModifierRightOption   = 0x40
	RawKeyEventDataIosModifierCommand       = 0x100000
	RawKeyEventDataIosModifierLeftCommand   = 0x08
	RawKeyEventDataIosModifierRightCommand  = 0x10
	RawKeyEventDataIosModifierNumericPad    = 0x200000
	RawKeyEventDataIosModifierHelp          = 0x400000
	RawKeyEventDataIosModifierFunction      = 0x800000
	RawKeyEventDataIosDeviceIndependentMask = 0xffff0000
)
