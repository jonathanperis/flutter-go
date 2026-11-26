package ui

// Port of engine/src/flutter/lib/ui/key.dart (Flutter 3.50.1): KeyData,
// the raw key event the engine hands the framework, and its enums. The
// engine side (PlatformDispatcher.onKeyData on the flutter/keydata channel)
// is in pointer_dispatch.go.

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// KeyEventType is the type of a key event.
type KeyEventType int

const (
	// KeyEventTypeDown: the key is pressed.
	KeyEventTypeDown KeyEventType = iota
	// KeyEventTypeUp: the key is released.
	KeyEventTypeUp
	// KeyEventTypeRepeat: the key is held, causing a repeated key input.
	KeyEventTypeRepeat
)

// KeyEventTypeValues is KeyEventType.values.
var KeyEventTypeValues = []KeyEventType{KeyEventTypeDown, KeyEventTypeUp, KeyEventTypeRepeat}

// Name is the enum value's name (Dart's .name).
func (t KeyEventType) Name() string { return [...]string{"down", "up", "repeat"}[t] }

func (t KeyEventType) String() string { return "KeyEventType." + t.Name() }

// Label is a human-readable label ("Key Down").
func (t KeyEventType) Label() string { return [...]string{"Key Down", "Key Up", "Key Repeat"}[t] }

// KeyEventDeviceType is the source device of a key event.
type KeyEventDeviceType int

const (
	// KeyEventDeviceTypeKeyboard: a keyboard.
	KeyEventDeviceTypeKeyboard KeyEventDeviceType = iota
	// KeyEventDeviceTypeDirectionalPad: a directional pad.
	KeyEventDeviceTypeDirectionalPad
	// KeyEventDeviceTypeGamepad: a gamepad.
	KeyEventDeviceTypeGamepad
	// KeyEventDeviceTypeJoystick: a joystick.
	KeyEventDeviceTypeJoystick
	// KeyEventDeviceTypeHdmi: a device connected over HDMI.
	KeyEventDeviceTypeHdmi
)

// KeyEventDeviceTypeValues is KeyEventDeviceType.values.
var KeyEventDeviceTypeValues = []KeyEventDeviceType{KeyEventDeviceTypeKeyboard, KeyEventDeviceTypeDirectionalPad,
	KeyEventDeviceTypeGamepad, KeyEventDeviceTypeJoystick, KeyEventDeviceTypeHdmi}

// Name is the enum value's name (Dart's .name).
func (t KeyEventDeviceType) Name() string {
	return [...]string{"keyboard", "directionalPad", "gamepad", "joystick", "hdmi"}[t]
}

func (t KeyEventDeviceType) String() string { return "KeyEventDeviceType." + t.Name() }

// Label is a human-readable label ("Directional Pad").
func (t KeyEventDeviceType) Label() string {
	return [...]string{"Keyboard", "Directional Pad", "Gamepad", "Joystick", "HDMI"}[t]
}

// KeyData is information about a key event. The zero DeviceType is Dart's
// default (keyboard); a nil Character is null.
type KeyData struct {
	TimeStamp   time.Duration
	Type        KeyEventType
	DeviceType  KeyEventDeviceType
	Physical    int
	Logical     int
	Character   *string
	Synthesized bool
}

// keyDataNonValueBits is KeyData._nonValueBits (the VM branch): the bits
// above the 32-bit value mask, limited to 20 bits.
func keyDataNonValueBits(n int) int {
	const valueMaskWidth = 32
	const maxSafeIntegerWidth = 52
	const nonValueMask = (1 << (maxSafeIntegerWidth - valueMaskWidth)) - 1
	return (n >> valueMaskWidth) & nonValueMask
}

func (k KeyData) logicalToString() string {
	result := "0x" + strconv.FormatInt(int64(k.Logical), 16)
	planeNum := keyDataNonValueBits(k.Logical) & 0x0FF
	planeDescription := ""
	switch planeNum {
	case 0x000:
		planeDescription = " (Unicode)"
	case 0x001:
		planeDescription = " (Unprintable)"
	case 0x002:
		planeDescription = " (Flutter)"
	case 0x011:
		planeDescription = " (Android)"
	case 0x012:
		planeDescription = " (Fuchsia)"
	case 0x013:
		planeDescription = " (iOS)"
	case 0x014:
		planeDescription = " (macOS)"
	case 0x015:
		planeDescription = " (GTK)"
	case 0x016:
		planeDescription = " (Windows)"
	case 0x017:
		planeDescription = " (Web)"
	case 0x018:
		planeDescription = " (GLFW)"
	}
	return result + planeDescription
}

func (k KeyData) escapeCharacter() string {
	if k.Character == nil {
		return "<none>"
	}
	switch *k.Character {
	case "\n":
		return `"\n"`
	case "\t":
		return `"\t"`
	case "\r":
		return `"\r"`
	case "\b":
		return `"\b"`
	case "\f":
		return `"\f"`
	}
	return `"` + *k.Character + `"`
}

func (k KeyData) quotedCharCode() string {
	if k.Character == nil {
		return ""
	}
	units := utf16.Encode([]rune(*k.Character))
	hexChars := make([]string, len(units))
	for i, code := range units {
		h := strconv.FormatInt(int64(code), 16)
		if len(h) < 2 {
			h = "0" + h
		}
		hexChars[i] = h
	}
	return " (0x" + strings.Join(hexChars, " ") + ")"
}

func (k KeyData) String() string {
	synthesized := ""
	if k.Synthesized {
		synthesized = ", synthesized"
	}
	// Dart's toString has no closing parenthesis.
	return "KeyData(" + k.Type.Label() + ", " +
		"physical: 0x" + strconv.FormatInt(int64(k.Physical), 16) + ", " +
		"logical: " + k.logicalToString() + ", " +
		"character: " + k.escapeCharacter() + k.quotedCharCode() +
		synthesized
}

// ToStringFull returns a detailed description of the key data.
func (k KeyData) ToStringFull() string {
	return "KeyData(type: " + k.Type.Label() + ", " +
		"deviceType: " + k.DeviceType.Label() + ", " +
		"timeStamp: " + dart.DurationToString(k.TimeStamp) + ", " +
		"physical: 0x" + strconv.FormatInt(int64(k.Physical), 16) + ", " +
		"logical: 0x" + strconv.FormatInt(int64(k.Logical), 16) + ", " +
		"character: " + k.escapeCharacter() + ", " +
		"synthesized: " + strconv.FormatBool(k.Synthesized) +
		")"
}
