package services

// Port of raw_keyboard_fuchsia.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// RawKeyEventDataFuchsia is the platform data of a Fuchsia key event
// (deprecated). Compare with Equal.
type RawKeyEventDataFuchsia struct {
	RawKeyEventDataBase
	HidUsage  int
	CodePoint int
	Modifiers int
}

// RawKeyEventDataFuchsiaOptions are its named parameters (default 0).
type RawKeyEventDataFuchsiaOptions struct{ HidUsage, CodePoint, Modifiers int }

// NewRawKeyEventDataFuchsia is RawKeyEventDataFuchsia({...}).
func NewRawKeyEventDataFuchsia(opts ...RawKeyEventDataFuchsiaOptions) *RawKeyEventDataFuchsia {
	var o RawKeyEventDataFuchsiaOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &RawKeyEventDataFuchsia{HidUsage: o.HidUsage, CodePoint: o.CodePoint, Modifiers: o.Modifiers}
	d.SetSelf(d)
	return d
}

func (d *RawKeyEventDataFuchsia) KeyLabel() string {
	if d.CodePoint == 0 {
		return ""
	}
	return dart.StringFromCharCode(d.CodePoint)
}

func (d *RawKeyEventDataFuchsia) LogicalKey() LogicalKeyboardKey {
	// A printable representation makes a logical key based on it.
	if d.CodePoint != 0 {
		flutterID := LogicalKeyboardKeyUnicodePlane | d.CodePoint&LogicalKeyboardKeyValueMask
		if k, ok := KFuchsiaToLogicalKey[flutterID]; ok {
			return k
		}
		return LogicalKeyboardKey{LogicalKeyboardKeyUnicodePlane | d.CodePoint&LogicalKeyboardKeyValueMask}
	}
	if newKey, ok := KFuchsiaToLogicalKey[d.HidUsage|LogicalKeyboardKeyFuchsiaPlane]; ok {
		return newKey
	}
	return LogicalKeyboardKey{d.HidUsage | LogicalKeyboardKeyFuchsiaPlane}
}

func (d *RawKeyEventDataFuchsia) PhysicalKey() PhysicalKeyboardKey {
	if k, ok := KFuchsiaToPhysicalKey[d.HidUsage]; ok {
		return k
	}
	return PhysicalKeyboardKey{LogicalKeyboardKeyFuchsiaPlane + d.HidUsage}
}

func (d *RawKeyEventDataFuchsia) isLeftRightModifierPressed(side KeyboardSide, anyMask, leftMask, rightMask int) bool {
	if d.Modifiers&anyMask == 0 {
		return false
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

func (d *RawKeyEventDataFuchsia) IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool {
	side := modifierSide(opts)
	switch key {
	case ModifierKeyControlModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataFuchsiaModifierControl, RawKeyEventDataFuchsiaModifierLeftControl, RawKeyEventDataFuchsiaModifierRightControl)
	case ModifierKeyShiftModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataFuchsiaModifierShift, RawKeyEventDataFuchsiaModifierLeftShift, RawKeyEventDataFuchsiaModifierRightShift)
	case ModifierKeyAltModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataFuchsiaModifierAlt, RawKeyEventDataFuchsiaModifierLeftAlt, RawKeyEventDataFuchsiaModifierRightAlt)
	case ModifierKeyMetaModifier:
		return d.isLeftRightModifierPressed(side, RawKeyEventDataFuchsiaModifierMeta, RawKeyEventDataFuchsiaModifierLeftMeta, RawKeyEventDataFuchsiaModifierRightMeta)
	case ModifierKeyCapsLockModifier:
		return d.Modifiers&RawKeyEventDataFuchsiaModifierCapsLock != 0
	}
	// Fuchsia doesn't have masks for the other keys (yet).
	return false
}

func (d *RawKeyEventDataFuchsia) GetModifierSide(key ModifierKey) *KeyboardSide {
	findSide := func(anyMask, leftMask, rightMask int) *KeyboardSide {
		combined := d.Modifiers & anyMask
		switch combined {
		case leftMask:
			return sidePtr(KeyboardSideLeft)
		case rightMask:
			return sidePtr(KeyboardSideRight)
		case anyMask:
			return sidePtr(KeyboardSideAll)
		}
		return nil
	}
	switch key {
	case ModifierKeyControlModifier:
		return findSide(RawKeyEventDataFuchsiaModifierControl, RawKeyEventDataFuchsiaModifierLeftControl, RawKeyEventDataFuchsiaModifierRightControl)
	case ModifierKeyShiftModifier:
		return findSide(RawKeyEventDataFuchsiaModifierShift, RawKeyEventDataFuchsiaModifierLeftShift, RawKeyEventDataFuchsiaModifierRightShift)
	case ModifierKeyAltModifier:
		return findSide(RawKeyEventDataFuchsiaModifierAlt, RawKeyEventDataFuchsiaModifierLeftAlt, RawKeyEventDataFuchsiaModifierRightAlt)
	case ModifierKeyMetaModifier:
		return findSide(RawKeyEventDataFuchsiaModifierMeta, RawKeyEventDataFuchsiaModifierLeftMeta, RawKeyEventDataFuchsiaModifierRightMeta)
	case ModifierKeyCapsLockModifier:
		if d.Modifiers&RawKeyEventDataFuchsiaModifierCapsLock == 0 {
			return nil
		}
		return sidePtr(KeyboardSideAll)
	}
	// Fuchsia doesn't support the other modifiers, so they can't be pressed.
	return nil
}

func (d *RawKeyEventDataFuchsia) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.RawKeyEventDataBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("hidUsage", d.HidUsage))
	properties.Add(foundation.NewDiagnosticsProperty("codePoint", d.CodePoint))
	properties.Add(foundation.NewDiagnosticsProperty("modifiers", d.Modifiers))
}

// Equal is Dart's ==.
func (d *RawKeyEventDataFuchsia) Equal(other RawKeyEventData) bool {
	o, ok := other.(*RawKeyEventDataFuchsia)
	return ok && o.HidUsage == d.HidUsage && o.CodePoint == d.CodePoint && o.Modifiers == d.Modifiers
}

// Keyboard modifier masks for Fuchsia modifiers.
const (
	RawKeyEventDataFuchsiaModifierNone         = 0x0
	RawKeyEventDataFuchsiaModifierCapsLock     = 0x1
	RawKeyEventDataFuchsiaModifierLeftShift    = 0x2
	RawKeyEventDataFuchsiaModifierRightShift   = 0x4
	RawKeyEventDataFuchsiaModifierShift        = RawKeyEventDataFuchsiaModifierLeftShift | RawKeyEventDataFuchsiaModifierRightShift
	RawKeyEventDataFuchsiaModifierLeftControl  = 0x8
	RawKeyEventDataFuchsiaModifierRightControl = 0x10
	RawKeyEventDataFuchsiaModifierControl      = RawKeyEventDataFuchsiaModifierLeftControl | RawKeyEventDataFuchsiaModifierRightControl
	RawKeyEventDataFuchsiaModifierLeftAlt      = 0x20
	RawKeyEventDataFuchsiaModifierRightAlt     = 0x40
	RawKeyEventDataFuchsiaModifierAlt          = RawKeyEventDataFuchsiaModifierLeftAlt | RawKeyEventDataFuchsiaModifierRightAlt
	RawKeyEventDataFuchsiaModifierLeftMeta     = 0x80
	RawKeyEventDataFuchsiaModifierRightMeta    = 0x100
	RawKeyEventDataFuchsiaModifierMeta         = RawKeyEventDataFuchsiaModifierLeftMeta | RawKeyEventDataFuchsiaModifierRightMeta
)
