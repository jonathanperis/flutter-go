package services_test

// Port of packages/flutter/test/services/keyboard_key_test.dart (11/11) and
// keyboard_maps_test.dart (1/1) (Flutter 3.50.1), plus own tests of the
// generated data and the key Diagnosticable output.

import (
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

func TestPhysicalKeyboardKeyLookUpByCode(t *testing.T) {
	for code, want := range map[int]services.PhysicalKeyboardKey{
		0x00070004: services.PhysicalKeyboardKeyKeyA,      // a letter key
		0x00070029: services.PhysicalKeyboardKeyEscape,    // a control key
		0x000700e1: services.PhysicalKeyboardKeyShiftLeft, // a modifier key
	} {
		if got := services.PhysicalKeyboardKeyFindKeyByCode(code); got == nil || *got != want {
			t.Errorf("findKeyByCode(%#x) = %v", code, got)
		}
	}
}

func TestPhysicalKeyboardKeyValuesAreEqual(t *testing.T) {
	if services.PhysicalKeyboardKeyKeyA != services.NewPhysicalKeyboardKey(services.PhysicalKeyboardKeyKeyA.UsbHidUsage) {
		t.Error("keyA")
	}
	if services.NewPhysicalKeyboardKey(0x12345) != (services.PhysicalKeyboardKey{UsbHidUsage: 0x12345}) {
		t.Error("0x12345")
	}
}

func TestPhysicalKeyboardKeyDebugNames(t *testing.T) {
	for key, want := range map[services.PhysicalKeyboardKey]string{
		services.PhysicalKeyboardKeyKeyA:         "Key A",
		services.PhysicalKeyboardKeyBackslash:    "Backslash",
		services.NewPhysicalKeyboardKey(0x12345): "Key with ID 0x00012345",
	} {
		if got := key.DebugName(); got == nil || *got != want {
			t.Errorf("%#x debugName %v, want %q", key.UsbHidUsage, got, want)
		}
	}
}

func TestLogicalKeyboardKeyLookUpByCode(t *testing.T) {
	for _, key := range []services.LogicalKeyboardKey{services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyEscape,
		services.LogicalKeyboardKeyShiftLeft} {
		if got := services.LogicalKeyboardKeyFindKeyByKeyId(key.KeyID); got == nil || *got != key {
			t.Errorf("findKeyByKeyId(%#x) = %v", key.KeyID, got)
		}
	}
}

func TestControlCharactersAreRecognizedAsSuch(t *testing.T) {
	for _, c := range []string{"\x08", "\x09", "\x0a", "\x0d", "\x1b", "\x7f"} {
		if !services.LogicalKeyboardKeyIsControlCharacter(c) {
			t.Errorf("%q is a control character", c)
		}
	}
	for _, c := range []string{"A", " ", "~", " "} {
		if services.LogicalKeyboardKeyIsControlCharacter(c) {
			t.Errorf("%q is not a control character", c)
		}
	}
}

func TestControlCharactersAreNotUsingIncorrectValues(t *testing.T) {
	for key, unit := range map[services.LogicalKeyboardKey]int{
		services.LogicalKeyboardKeyBackspace: 0x08, services.LogicalKeyboardKeyTab: 0x09,
		services.LogicalKeyboardKeyEnter: 0x0d, services.LogicalKeyboardKeyEscape: 0x1b, services.LogicalKeyboardKeyDelete: 0x7f,
	} {
		if key.KeyID != services.LogicalKeyboardKeyUnprintablePlane+unit {
			t.Errorf("%#x", key.KeyID)
		}
	}
}

func TestBasicSynonymsCanBeLookedUp(t *testing.T) {
	for key, want := range map[services.LogicalKeyboardKey]services.LogicalKeyboardKey{
		services.LogicalKeyboardKeyShiftLeft: services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyControlLeft: services.LogicalKeyboardKeyControl,
		services.LogicalKeyboardKeyAltLeft: services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyMetaLeft: services.LogicalKeyboardKeyMeta,
		services.LogicalKeyboardKeyShiftRight: services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyControlRight: services.LogicalKeyboardKeyControl,
		services.LogicalKeyboardKeyAltRight: services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyMetaRight: services.LogicalKeyboardKeyMeta,
	} {
		if s := key.Synonyms(); len(s) == 0 || s[0] != want {
			t.Errorf("%v synonyms %v", key.KeyID, s)
		}
	}
	if s := services.LogicalKeyboardKeyKeyA.Synonyms(); len(s) != 0 {
		t.Errorf("keyA synonyms %v", s)
	}
}

func TestSynonymsGetCollapsedProperly(t *testing.T) {
	if got := services.LogicalKeyboardKeyCollapseSynonyms(nil); len(got) != 0 {
		t.Error(got)
	}
	general := []services.LogicalKeyboardKey{services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyControl,
		services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyMeta}
	left := []services.LogicalKeyboardKey{services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyControlLeft,
		services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyMetaLeft}
	right := []services.LogicalKeyboardKey{services.LogicalKeyboardKeyShiftRight, services.LogicalKeyboardKeyControlRight,
		services.LogicalKeyboardKeyAltRight, services.LogicalKeyboardKeyMetaRight}
	for _, in := range [][]services.LogicalKeyboardKey{left, right, append(append([]services.LogicalKeyboardKey{}, left...), right...)} {
		if got := services.LogicalKeyboardKeyCollapseSynonyms(in); !sameSet(got, general...) {
			t.Errorf("collapse(%v) = %v", in, got)
		}
	}
	// expandSynonyms (own test): pseudo-keys expand in the reverse table's
	// order, other keys stay.
	got := services.LogicalKeyboardKeyExpandSynonyms([]services.LogicalKeyboardKey{services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyShift})
	want := []services.LogicalKeyboardKey{services.LogicalKeyboardKeyKeyA, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight}
	if !slicesEqual(got, want) {
		t.Errorf("expand = %v", got)
	}
}

func TestLogicalKeyboardKeyValuesAreEqual(t *testing.T) {
	if services.LogicalKeyboardKeyKeyA != services.NewLogicalKeyboardKey(services.LogicalKeyboardKeyKeyA.KeyID) {
		t.Error("keyA")
	}
}

func TestLogicalKeyboardKeyKeyLabel(t *testing.T) {
	for key, want := range map[services.LogicalKeyboardKey]string{
		services.LogicalKeyboardKeyKeyA:             "A",
		services.LogicalKeyboardKeyBackslash:        `\`,
		services.NewLogicalKeyboardKey(0xD9):        "Ù",
		services.NewLogicalKeyboardKey(0xF9):        "Ù",
		services.LogicalKeyboardKeyShiftLeft:        "Shift Left",
		services.LogicalKeyboardKeyNumpadDecimal:    "Numpad Decimal",
		services.LogicalKeyboardKeyNumpad1:          "Numpad 1",
		services.LogicalKeyboardKeyDelete:           "Delete",
		services.LogicalKeyboardKeyF12:              "F12",
		services.LogicalKeyboardKeyMediaPlay:        "Media Play",
		services.NewLogicalKeyboardKey(0x100012345): "",
	} {
		if got := key.KeyLabel(); got != want {
			t.Errorf("%#x keyLabel %q, want %q", key.KeyID, got, want)
		}
	}
}

func TestLogicalKeyboardKeyDebugName(t *testing.T) {
	for key, want := range map[services.LogicalKeyboardKey]string{
		services.LogicalKeyboardKeyKeyA:             "Key A",
		services.LogicalKeyboardKeyBackslash:        "Backslash",
		services.NewLogicalKeyboardKey(0xD9):        "Key Ù",
		services.LogicalKeyboardKeyMediaPlay:        "Media Play",
		services.NewLogicalKeyboardKey(0x100012345): "Key with ID 0x00100012345",
	} {
		if got := key.DebugName(); got == nil || *got != want {
			t.Errorf("%#x debugName %v, want %q", key.KeyID, got, want)
		}
	}
}

func TestWebPhysicalKeyMappingHasNoEntriesWithoutAChromeCode(t *testing.T) {
	// Regression test for flutter#106074.
	if _, ok := services.KWebToPhysicalKey["KbdIllumDown"]; ok {
		t.Error("KbdIllumDown is mapped")
	}
}

// Own tests.

func TestGeneratedKeyTables(t *testing.T) {
	n := 0
	for range services.LogicalKeyboardKeyKnownLogicalKeys() {
		n++
	}
	if n != 444 {
		t.Errorf("%d known logical keys, want 444 (keyboard_key.g.dart)", n)
	}
	n = 0
	for range services.PhysicalKeyboardKeyKnownPhysicalKeys() {
		n++
	}
	if n != 269 {
		t.Errorf("%d known physical keys, want 269", n)
	}
	if len(services.KAndroidToLogicalKey) != len(services.KAndroidToLogicalKeyKeys) {
		t.Error("map and key order disagree")
	}
	if !services.NewLogicalKeyboardKey(0x01100000001).IsAutogenerated() || services.LogicalKeyboardKeyKeyA.IsAutogenerated() {
		t.Error("isAutogenerated")
	}
}

func TestKeyToString(t *testing.T) {
	s := services.LogicalKeyboardKeyKeyA.String()
	i := strings.Index(s, "#")
	if !strings.HasPrefix(s, "LogicalKeyboardKey#") || s[i+6:] != `(keyId: "0x00000061", keyLabel: "A", debugName: "Key A")` {
		t.Errorf("%s", s)
	}
	s = services.PhysicalKeyboardKeyKeyA.String()
	i = strings.Index(s, "#")
	if !strings.HasPrefix(s, "PhysicalKeyboardKey#") || s[i+6:] != `(usbHidUsage: "0x00070004", debugName: "Key A")` {
		t.Errorf("%s", s)
	}
}
