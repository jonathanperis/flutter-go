package services

// Port of raw_keyboard_macos.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// RuneToLowerCase is runeToLowerCase: the lower case of a Basic
// Multilingual Plane rune (others are returned as they are).
func RuneToLowerCase(r int) int {
	const utf16BmpUpperBound = 0xD7FF
	if r > utf16BmpUpperBound {
		return r
	}
	return dart.CodeUnitAt(dart.StringToLowerCase(dart.StringFromCharCode(r)), 0)
}

// RawKeyEventDataMacOs is the platform data of a macOS key event
// (deprecated). Compare with Equal.
type RawKeyEventDataMacOs struct {
	RawKeyEventDataBase
	Characters                  string
	CharactersIgnoringModifiers string
	KeyCode                     int
	Modifiers                   int
	// SpecifiedLogicalKey is the logical key the embedder specified (nil:
	// none).
	SpecifiedLogicalKey *int
}

// RawKeyEventDataMacOsOptions are its named parameters.
type RawKeyEventDataMacOsOptions struct {
	Characters, CharactersIgnoringModifiers string
	KeyCode, Modifiers                      int
	SpecifiedLogicalKey                     *int
}

// NewRawKeyEventDataMacOs is RawKeyEventDataMacOs({...}).
func NewRawKeyEventDataMacOs(opts ...RawKeyEventDataMacOsOptions) *RawKeyEventDataMacOs {
	var o RawKeyEventDataMacOsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &RawKeyEventDataMacOs{Characters: o.Characters, CharactersIgnoringModifiers: o.CharactersIgnoringModifiers,
		KeyCode: o.KeyCode, Modifiers: o.Modifiers, SpecifiedLogicalKey: o.SpecifiedLogicalKey}
	d.SetSelf(d)
	return d
}

func (d *RawKeyEventDataMacOs) KeyLabel() string { return d.CharactersIgnoringModifiers }

func (d *RawKeyEventDataMacOs) PhysicalKey() PhysicalKeyboardKey {
	if k, ok := KMacOsToPhysicalKey[d.KeyCode]; ok {
		return k
	}
	return PhysicalKeyboardKey{LogicalKeyboardKeyWindowsPlane + d.KeyCode}
}

// stringRunes is String.runes.toList(): the code points (a lone surrogate
// is one).
func stringRunes(s string) []int {
	units := dart.StringCodeUnits(s)
	var out []int
	for i := 0; i < len(units); i++ {
		u := int(units[i])
		if u >= 0xD800 && u < 0xDC00 && i+1 < len(units) && units[i+1] >= 0xDC00 && units[i+1] < 0xE000 {
			out = append(out, 0x10000+((u-0xD800)<<10)+(int(units[i+1])-0xDC00))
			i++
			continue
		}
		out = append(out, u)
	}
	return out
}

func (d *RawKeyEventDataMacOs) LogicalKey() LogicalKeyboardKey {
	if d.SpecifiedLogicalKey != nil {
		return findLogicalKeyOrNew(*d.SpecifiedLogicalKey)
	}
	// A printable number pad key, so that "=" and the number pad's "=" can
	// be told apart.
	if numPadKey, ok := KMacOsNumPadMap[d.KeyCode]; ok {
		return numPadKey
	}
	// Keys that can't be derived with charactersIgnoringModifiers.
	if knownKey, ok := KMacOsToLogicalKey[d.KeyCode]; ok {
		return knownKey
	}
	// A single printable character generates the key from its Unicode
	// value; control keys, function keys (unprintable) and multi-char
	// characters are discarded.
	var character *int
	keyLabel := d.KeyLabel()
	if keyLabel != "" {
		codePoints := stringRunes(keyLabel)
		if len(codePoints) == 1 && !LogicalKeyboardKeyIsControlCharacter(keyLabel) && !appleIsUnprintableKey(keyLabel) {
			c := RuneToLowerCase(codePoints[0])
			character = &c
		}
	}
	if character != nil {
		keyID := LogicalKeyboardKeyUnicodePlane | (*character & LogicalKeyboardKeyValueMask)
		return findLogicalKeyOrNew(keyID)
	}
	// A non-printable key we don't know about: mint a new code.
	return LogicalKeyboardKey{d.KeyCode | LogicalKeyboardKeyMacosPlane}
}

// appleIsUnprintableKey is _isUnprintableKey (macOS and iOS): one code unit
// in the private function-key range.
func appleIsUnprintableKey(label string) bool {
	if dart.StringLength(label) != 1 {
		return false
	}
	codeUnit := dart.CodeUnitAt(label, 0)
	return codeUnit >= 0xF700 && codeUnit <= 0xF8FF
}

// appleIsLeftRightModifierPressed is macOS's and iOS's
// _isLeftRightModifierPressed.
func appleIsLeftRightModifierPressed(modifiers int, side KeyboardSide, anyMask, leftMask, rightMask int) bool {
	if modifiers&anyMask == 0 {
		return false
	}
	if modifiers&(leftMask|rightMask|anyMask) == anyMask {
		// Only the "any" bit: true for either side.
		return true
	}
	switch side {
	case KeyboardSideAny:
		return true
	case KeyboardSideAll:
		return modifiers&leftMask != 0 && modifiers&rightMask != 0
	case KeyboardSideLeft:
		return modifiers&leftMask != 0
	default:
		return modifiers&rightMask != 0
	}
}

// appleIsModifierPressed is macOS's and iOS's isModifierPressed (their
// masks are the same).
func appleIsModifierPressed(self RawKeyEventData, modifiers int, key ModifierKey, side KeyboardSide) bool {
	independentModifier := modifiers & RawKeyEventDataMacOsDeviceIndependentMask
	var result bool
	switch key {
	case ModifierKeyControlModifier:
		result = appleIsLeftRightModifierPressed(modifiers, side, independentModifier&RawKeyEventDataMacOsModifierControl,
			RawKeyEventDataMacOsModifierLeftControl, RawKeyEventDataMacOsModifierRightControl)
	case ModifierKeyShiftModifier:
		result = appleIsLeftRightModifierPressed(modifiers, side, independentModifier&RawKeyEventDataMacOsModifierShift,
			RawKeyEventDataMacOsModifierLeftShift, RawKeyEventDataMacOsModifierRightShift)
	case ModifierKeyAltModifier:
		result = appleIsLeftRightModifierPressed(modifiers, side, independentModifier&RawKeyEventDataMacOsModifierOption,
			RawKeyEventDataMacOsModifierLeftOption, RawKeyEventDataMacOsModifierRightOption)
	case ModifierKeyMetaModifier:
		result = appleIsLeftRightModifierPressed(modifiers, side, independentModifier&RawKeyEventDataMacOsModifierCommand,
			RawKeyEventDataMacOsModifierLeftCommand, RawKeyEventDataMacOsModifierRightCommand)
	case ModifierKeyCapsLockModifier:
		result = independentModifier&RawKeyEventDataMacOsModifierCapsLock != 0
	default:
		// The function modifier bit is set for any function key, and the
		// other masks are not used in Apple keyboards.
		result = false
	}
	if result && self.GetModifierSide(key) == nil {
		panic(dart.NewAssertionError(foundation.ObjectRuntimeType(self, "") +
			" thinks that a modifier is pressed, but can't figure out what side it's on."))
	}
	return result
}

// appleGetModifierSide is macOS's and iOS's getModifierSide.
func appleGetModifierSide(modifiers int, key ModifierKey) *KeyboardSide {
	findSide := func(anyMask, leftMask, rightMask int) *KeyboardSide {
		combinedMask := leftMask | rightMask
		combined := modifiers & combinedMask
		switch {
		case combined == leftMask:
			return sidePtr(KeyboardSideLeft)
		case combined == rightMask:
			return sidePtr(KeyboardSideRight)
		case combined == combinedMask || modifiers&(combinedMask|anyMask) == anyMask:
			// Only the "either" flag, or both sides without it.
			return sidePtr(KeyboardSideAll)
		}
		return nil
	}
	switch key {
	case ModifierKeyControlModifier:
		return findSide(RawKeyEventDataMacOsModifierControl, RawKeyEventDataMacOsModifierLeftControl, RawKeyEventDataMacOsModifierRightControl)
	case ModifierKeyShiftModifier:
		return findSide(RawKeyEventDataMacOsModifierShift, RawKeyEventDataMacOsModifierLeftShift, RawKeyEventDataMacOsModifierRightShift)
	case ModifierKeyAltModifier:
		return findSide(RawKeyEventDataMacOsModifierOption, RawKeyEventDataMacOsModifierLeftOption, RawKeyEventDataMacOsModifierRightOption)
	case ModifierKeyMetaModifier:
		return findSide(RawKeyEventDataMacOsModifierCommand, RawKeyEventDataMacOsModifierLeftCommand, RawKeyEventDataMacOsModifierRightCommand)
	}
	return sidePtr(KeyboardSideAll)
}

func (d *RawKeyEventDataMacOs) IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool {
	return appleIsModifierPressed(d, d.Modifiers, key, modifierSide(opts))
}

func (d *RawKeyEventDataMacOs) GetModifierSide(key ModifierKey) *KeyboardSide {
	return appleGetModifierSide(d.Modifiers, key)
}

// ShouldDispatchEvent: the Fn key's own events are ignored on macOS (laptop
// keyboards send them along with the keys Fn modifies).
func (d *RawKeyEventDataMacOs) ShouldDispatchEvent() bool {
	return d.LogicalKey() != LogicalKeyboardKeyFn
}

func (d *RawKeyEventDataMacOs) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.RawKeyEventDataBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("characters", d.Characters))
	properties.Add(foundation.NewDiagnosticsProperty("charactersIgnoringModifiers", d.CharactersIgnoringModifiers))
	properties.Add(foundation.NewDiagnosticsProperty("keyCode", d.KeyCode))
	properties.Add(foundation.NewDiagnosticsProperty("modifiers", d.Modifiers))
	properties.Add(foundation.NewDiagnosticsProperty("specifiedLogicalKey", d.SpecifiedLogicalKey,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
}

// Equal is Dart's == (not comparing specifiedLogicalKey, as Dart doesn't).
func (d *RawKeyEventDataMacOs) Equal(other RawKeyEventData) bool {
	o, ok := other.(*RawKeyEventDataMacOs)
	return ok && o.Characters == d.Characters && o.CharactersIgnoringModifiers == d.CharactersIgnoringModifiers &&
		o.KeyCode == d.KeyCode && o.Modifiers == d.Modifiers
}

// Modifier key masks (NSEventModifierFlags and IOLLEvent.h).
const (
	RawKeyEventDataMacOsModifierCapsLock      = 0x10000
	RawKeyEventDataMacOsModifierShift         = 0x20000
	RawKeyEventDataMacOsModifierLeftShift     = 0x02
	RawKeyEventDataMacOsModifierRightShift    = 0x04
	RawKeyEventDataMacOsModifierControl       = 0x40000
	RawKeyEventDataMacOsModifierLeftControl   = 0x01
	RawKeyEventDataMacOsModifierRightControl  = 0x2000
	RawKeyEventDataMacOsModifierOption        = 0x80000
	RawKeyEventDataMacOsModifierLeftOption    = 0x20
	RawKeyEventDataMacOsModifierRightOption   = 0x40
	RawKeyEventDataMacOsModifierCommand       = 0x100000
	RawKeyEventDataMacOsModifierLeftCommand   = 0x08
	RawKeyEventDataMacOsModifierRightCommand  = 0x10
	RawKeyEventDataMacOsModifierNumericPad    = 0x200000
	RawKeyEventDataMacOsModifierHelp          = 0x400000
	RawKeyEventDataMacOsModifierFunction      = 0x800000
	RawKeyEventDataMacOsDeviceIndependentMask = 0xffff0000
)
