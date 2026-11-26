package services

// Port of raw_keyboard_web.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func webUnicodeChar(key string) *string {
	if dart.StringLength(key) == 1 {
		s := dart.Substring(key, 0, 1)
		return &s
	}
	return nil
}

// RawKeyEventDataWeb is the platform data of a web key event (deprecated).
// Compare with Equal.
type RawKeyEventDataWeb struct {
	RawKeyEventDataBase
	// Code is the KeyboardEvent's code (the physical key).
	Code string
	// Key is the KeyboardEvent's key (the logical key).
	Key       string
	Location  int
	MetaState int
	KeyCode   int
}

// RawKeyEventDataWebOptions are its named parameters (Code and Key are
// required in Dart).
type RawKeyEventDataWebOptions struct {
	Code, Key                    string
	Location, MetaState, KeyCode int
}

// NewRawKeyEventDataWeb is RawKeyEventDataWeb({required code, required
// key, location, metaState, keyCode}).
func NewRawKeyEventDataWeb(o RawKeyEventDataWebOptions) *RawKeyEventDataWeb {
	d := &RawKeyEventDataWeb{Code: o.Code, Key: o.Key, Location: o.Location, MetaState: o.MetaState, KeyCode: o.KeyCode}
	d.SetSelf(d)
	return d
}

func (d *RawKeyEventDataWeb) KeyLabel() string {
	if d.Key == "Unidentified" {
		return ""
	}
	if c := webUnicodeChar(d.Key); c != nil {
		return *c
	}
	return ""
}

func (d *RawKeyEventDataWeb) PhysicalKey() PhysicalKeyboardKey {
	if k, ok := KWebToPhysicalKey[d.Code]; ok {
		return k
	}
	return PhysicalKeyboardKey{LogicalKeyboardKeyWebPlane + dart.StringHashCode(d.Code)}
}

func (d *RawKeyEventDataWeb) LogicalKey() LogicalKeyboardKey {
	// A key based on location: numpad keys and left/right modifiers.
	if keys, ok := KWebLocationMap[d.Key]; ok {
		if d.Location < 0 || d.Location >= len(keys) {
			panic(dart.IndexError{Index: d.Location, Length: len(keys), Name: "index"})
		}
		if k := keys[d.Location]; k != nil {
			return *k
		}
	}
	if newKey, ok := KWebToLogicalKey[d.Key]; ok {
		return newKey
	}
	if dart.StringLength(d.Key) == 1 {
		return LogicalKeyboardKey{dart.CodeUnitAt(dart.StringToLowerCase(d.Key), 0)}
	}
	// A non-printable key we don't know about: mint a key from the code
	// (the key would always be "Unidentified").
	return LogicalKeyboardKey{dart.StringHashCode(d.Code) + LogicalKeyboardKeyWebPlane}
}

func (d *RawKeyEventDataWeb) IsModifierPressed(key ModifierKey, opts ...IsModifierPressedOptions) bool {
	switch key {
	case ModifierKeyControlModifier:
		return d.MetaState&RawKeyEventDataWebModifierControl != 0
	case ModifierKeyShiftModifier:
		return d.MetaState&RawKeyEventDataWebModifierShift != 0
	case ModifierKeyAltModifier:
		return d.MetaState&RawKeyEventDataWebModifierAlt != 0
	case ModifierKeyMetaModifier:
		return d.MetaState&RawKeyEventDataWebModifierMeta != 0
	case ModifierKeyNumLockModifier:
		return d.MetaState&RawKeyEventDataWebModifierNumLock != 0
	case ModifierKeyCapsLockModifier:
		return d.MetaState&RawKeyEventDataWebModifierCapsLock != 0
	case ModifierKeyScrollLockModifier:
		return d.MetaState&RawKeyEventDataWebModifierScrollLock != 0
	}
	// The browser doesn't report the state of the FN and SYM modifiers.
	return false
}

// GetModifierSide: the web doesn't distinguish the sides.
func (d *RawKeyEventDataWeb) GetModifierSide(ModifierKey) *KeyboardSide {
	return sidePtr(KeyboardSideAny)
}

func (d *RawKeyEventDataWeb) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.RawKeyEventDataBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("code", d.Code))
	properties.Add(foundation.NewDiagnosticsProperty("key", d.Key))
	properties.Add(foundation.NewDiagnosticsProperty("location", d.Location))
	properties.Add(foundation.NewDiagnosticsProperty("metaState", d.MetaState))
	properties.Add(foundation.NewDiagnosticsProperty("keyCode", d.KeyCode))
}

// Equal is Dart's ==.
func (d *RawKeyEventDataWeb) Equal(other RawKeyEventData) bool {
	o, ok := other.(*RawKeyEventDataWeb)
	return ok && o.Code == d.Code && o.Key == d.Key && o.Location == d.Location && o.MetaState == d.MetaState &&
		o.KeyCode == d.KeyCode
}

// Modifier key masks.
const (
	RawKeyEventDataWebModifierNone       = 0
	RawKeyEventDataWebModifierShift      = 0x01
	RawKeyEventDataWebModifierAlt        = 0x02
	RawKeyEventDataWebModifierControl    = 0x04
	RawKeyEventDataWebModifierMeta       = 0x08
	RawKeyEventDataWebModifierNumLock    = 0x10
	RawKeyEventDataWebModifierCapsLock   = 0x20
	RawKeyEventDataWebModifierScrollLock = 0x40
)
