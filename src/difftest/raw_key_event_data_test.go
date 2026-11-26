package difftest

// Go twin of flutter/test/raw_key_event_data_test.dart: the same
// pseudo-random RawKeyEventData of every platform, every getter compared
// line by line with Flutter's; plus the known keys, the VM's string case
// mapping and String.hashCode.

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type keyRand struct{ seed int }

func (r *keyRand) next() int {
	r.seed = (r.seed*1103515245 + 12345) % 2147483648
	return r.seed
}

func pickOf[T any](r *keyRand, list []T) T { return list[r.next()%len(list)] }

var keyHashRE = regexp.MustCompile(`#[0-9a-f]{5}`)

// stripHashes drops object identity hashes (they differ between runs).
func stripHashes(s string) string { return keyHashRE.ReplaceAllString(s, "#") }

// keyGuard is the Dart side's guard: a string is JSON-encoded, anything
// else printed as Dart's '$v'; a panic is "ERR".
func keyGuard(f func() any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = "ERR"
		}
	}()
	switch v := f().(type) {
	case nil:
		return "null"
	case string:
		return dart.JSONEncode(v)
	case *string:
		if v == nil {
			return "null"
		}
		return dart.JSONEncode(*v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprint(v)
	}
}

var rawCodePoints = []int{
	0, 0x61, 0x41, 0x31, 0x7a, 0x60, 0x20, 0x7f, 0x1b, 0x0d, 0x09, 0xe9, 0xc9, 0xdf, 0xff, 0xb5,
	0x3a3, 0x3c3, 0x80000301, 0x301, 0xf700, 0xf728, 0x1f600, 0x2028, 0xd800, 0x10ffff, 0x110000,
}

var rawStrings = []string{
	"", "a", "A", "z", "1", " ", "\t", "\u001b", "é", "É", "ß", "ÿ", "ab", "😀", "e\u0301", "\uF700", "\uF728",
	"Ω", "ω", "\u2028", "=", "/",
}

var androidSources = []int{0, 0x101, 0x201, 0x401, 0x01000010, 0x02000001}

func describeRawData(lines *[]string, tag string, data, previous services.RawKeyEventData, message map[string]any) {
	parts := []string{tag}
	parts = append(parts, "label="+keyGuard(func() any { return data.KeyLabel() }))
	parts = append(parts, "physical="+keyGuard(func() any { return "0x" + strconv.FormatInt(int64(data.PhysicalKey().UsbHidUsage), 16) }))
	parts = append(parts, "logical="+keyGuard(func() any { return "0x" + strconv.FormatInt(int64(data.LogicalKey().KeyID), 16) }))
	for _, key := range services.ModifierKeyValues {
		var pressed []string
		for _, side := range services.KeyboardSideValues {
			pressed = append(pressed, keyGuard(func() any {
				if data.IsModifierPressed(key, services.IsModifierPressedOptions{Side: side}) {
					return 1
				}
				return 0
			}))
		}
		parts = append(parts, key.Name()+"=["+strings.Join(pressed, ",")+"] side="+keyGuard(func() any {
			if s := data.GetModifierSide(key); s != nil {
				return s.Name()
			}
			return nil
		}))
	}
	parts = append(parts, "modifiersPressed="+keyGuard(func() any {
		m := data.ModifiersPressed()
		var out []string
		for _, k := range services.ModifierKeyValues {
			if side, ok := m[k]; ok {
				out = append(out, k.Name()+":"+side.Name())
			}
		}
		return strings.Join(out, " ")
	}))
	parts = append(parts, "cs am="+keyGuard(func() any { return data.IsControlPressed() })+keyGuard(func() any { return data.IsShiftPressed() })+
		keyGuard(func() any { return data.IsAltPressed() })+keyGuard(func() any { return data.IsMetaPressed() }))
	parts = append(parts, "dispatch="+keyGuard(func() any { return data.ShouldDispatchEvent() }))
	parts = append(parts, "str="+keyGuard(func() any { return stripHashes(data.String()) }))
	if previous != nil {
		eq := data.(interface {
			Equal(services.RawKeyEventData) bool
		})
		parts = append(parts, fmt.Sprintf("eqPrev=%v eqSelf=%v", eq.Equal(previous), eq.Equal(data)))
	}
	parts = append(parts, "msg="+keyGuard(func() any {
		event := services.RawKeyEventFromMessage(message)
		character := "null"
		if c := event.Character(); c != nil {
			character = dart.JSONEncode(*c)
		}
		return foundation.ObjectRuntimeType(event, "") + " " + character + " " + strconv.FormatBool(event.Repeat()) + " " + stripHashes(event.String())
	}))
	*lines = append(*lines, strings.Join(parts, " | "))
}

func keysOf[K comparable, V any](m map[K]V, keys []K) []K { return keys }

func typeOf(i int) string {
	if i%2 == 0 {
		return "keydown"
	}
	return "keyup"
}

func rawKeyEventDataLines() []string {
	var lines []string
	r := &keyRand{seed: 12345}
	const n = 300
	knownLogical := func() []services.LogicalKeyboardKey {
		var out []services.LogicalKeyboardKey
		for k := range services.LogicalKeyboardKeyKnownLogicalKeys() {
			out = append(out, k)
		}
		return out
	}()
	// Android
	var previous services.RawKeyEventData
	for i := 0; i < n; i++ {
		var keyCode, scanCode int
		if r.next()%3 == 0 {
			keyCode = r.next() % 400
		} else {
			keyCode = pickOf(r, services.KAndroidToLogicalKeyKeys)
		}
		if r.next()%4 == 0 {
			scanCode = r.next() % 800
		} else {
			scanCode = pickOf(r, services.KAndroidToPhysicalKeyKeys)
		}
		plain := pickOf(r, rawCodePoints)
		code := pickOf(r, rawCodePoints)
		meta := r.next() % 0x800000
		source := pickOf(r, androidSources)
		flags := r.next() % 8
		data := services.NewRawKeyEventDataAndroid(services.RawKeyEventDataAndroidOptions{Flags: flags, CodePoint: code,
			PlainCodePoint: plain, KeyCode: keyCode, ScanCode: scanCode, MetaState: meta, EventSource: source})
		message := map[string]any{"type": typeOf(i), "keymap": "android", "flags": flags, "codePoint": code,
			"plainCodePoint": plain, "keyCode": keyCode, "scanCode": scanCode, "metaState": meta, "source": source}
		if i%3 == 0 {
			if plain == 0 {
				message["character"] = nil
			} else {
				message["character"] = "c"
			}
		}
		describeRawData(&lines, fmt.Sprintf("android %d", i), data, previous, message)
		previous = data
	}
	// Fuchsia
	previous = nil
	for i := 0; i < n; i++ {
		var hid int
		if r.next()%4 == 0 {
			hid = r.next() % 0x80000
		} else {
			hid = pickOf(r, services.KFuchsiaToPhysicalKeyKeys)
		}
		code := pickOf(r, rawCodePoints)
		modifiers := r.next() % 0x200
		data := services.NewRawKeyEventDataFuchsia(services.RawKeyEventDataFuchsiaOptions{HidUsage: hid, CodePoint: code, Modifiers: modifiers})
		describeRawData(&lines, fmt.Sprintf("fuchsia %d", i), data, previous, map[string]any{"type": typeOf(i), "keymap": "fuchsia",
			"hidUsage": hid, "codePoint": code, "modifiers": modifiers})
		previous = data
	}
	// macOS and iOS
	for _, platform := range []string{"macos", "ios"} {
		previous = nil
		var keyCodes []int
		labels := append([]string{}, rawStrings...)
		if platform == "macos" {
			keyCodes = append(keyCodes, services.KMacOsToPhysicalKeyKeys...)
			keyCodes = append(keyCodes, services.KMacOsToLogicalKeyKeys...)
			keyCodes = append(keyCodes, services.KMacOsNumPadMapKeys...)
			keyCodes = append(keyCodes, services.KMacOsFunctionKeyMapKeys...)
		} else {
			keyCodes = append(keyCodes, services.KIosToPhysicalKeyKeys...)
			keyCodes = append(keyCodes, services.KIosToLogicalKeyKeys...)
			keyCodes = append(keyCodes, services.KIosNumPadMapKeys...)
			labels = append(labels, services.KIosSpecialLogicalMapKeys...)
		}
		for i := 0; i < n; i++ {
			var keyCode int
			if r.next()%5 == 0 {
				keyCode = r.next() % 300
			} else {
				keyCode = pickOf(r, keyCodes)
			}
			chars := pickOf(r, labels)
			ignoring := pickOf(r, labels)
			modifiers := r.next() % 0x1000000
			masked := modifiers
			if r.next()%2 == 0 {
				masked = modifiers & 0xffff0000
			}
			var specified *int
			if platform == "macos" && r.next()%6 == 0 {
				id := pickOf(r, knownLogical).KeyID
				specified = &id
			}
			var data services.RawKeyEventData
			message := map[string]any{"type": typeOf(i), "keymap": platform, "characters": chars,
				"charactersIgnoringModifiers": ignoring, "keyCode": keyCode, "modifiers": masked}
			if platform == "macos" {
				data = services.NewRawKeyEventDataMacOs(services.RawKeyEventDataMacOsOptions{Characters: chars,
					CharactersIgnoringModifiers: ignoring, KeyCode: keyCode, Modifiers: masked, SpecifiedLogicalKey: specified})
				if specified != nil {
					message["specifiedLogicalKey"] = *specified
				}
			} else {
				data = services.NewRawKeyEventDataIos(services.RawKeyEventDataIosOptions{Characters: chars,
					CharactersIgnoringModifiers: ignoring, KeyCode: keyCode, Modifiers: masked})
			}
			describeRawData(&lines, fmt.Sprintf("%s %d", platform, i), data, previous, message)
			previous = data
		}
	}
	// Linux
	for _, toolkit := range []string{"glfw", "gtk"} {
		previous = nil
		var keyCodes []int
		if toolkit == "glfw" {
			keyCodes = append(keyCodes, services.KGlfwToLogicalKeyKeys...)
			keyCodes = append(keyCodes, services.KGlfwNumpadMapKeys...)
			keyCodes = append(keyCodes, 340, 344, 341, 342, 343, 280, 282)
		} else {
			keyCodes = append(keyCodes, services.KGtkToLogicalKeyKeys...)
			keyCodes = append(keyCodes, services.KGtkNumpadMapKeys...)
			keyCodes = append(keyCodes, 0xffe1, 0xffe5, 0xffe6, 0xffe9, 0xff7f)
		}
		gtkBits := []int{1, 2, 4, 8, 16, 1 << 26}
		for i := 0; i < n; i++ {
			var keyCode, scanCode int
			if r.next()%5 == 0 {
				keyCode = r.next() % 0x10000
			} else {
				keyCode = pickOf(r, keyCodes)
			}
			if r.next()%4 == 0 {
				scanCode = r.next() % 300
			} else {
				scanCode = pickOf(r, services.KLinuxToPhysicalKeyKeys)
			}
			usv := pickOf(r, rawCodePoints)
			modifiers := 0
			if toolkit == "glfw" {
				modifiers = r.next() % 0x40
			} else {
				for _, bit := range gtkBits {
					if r.next()%2 == 0 {
						modifiers |= bit
					}
				}
			}
			isDown := r.next()%2 == 0
			var specified *int
			if r.next()%6 == 0 {
				id := pickOf(r, knownLogical).KeyID
				specified = &id
			}
			data := services.NewRawKeyEventDataLinux(services.NewKeyHelper(toolkit), isDown, services.RawKeyEventDataLinuxOptions{
				UnicodeScalarValues: usv, ScanCode: scanCode, KeyCode: keyCode, Modifiers: modifiers, SpecifiedLogicalKey: specified})
			typ := "keyup"
			if isDown {
				typ = "keydown"
			}
			message := map[string]any{"type": typ, "keymap": "linux", "toolkit": toolkit, "unicodeScalarValues": usv,
				"keyCode": keyCode, "scanCode": scanCode, "modifiers": modifiers}
			if specified != nil {
				message["specifiedLogicalKey"] = *specified
			}
			describeRawData(&lines, fmt.Sprintf("linux-%s %d", toolkit, i), data, previous, message)
			previous = data
		}
	}
	// Windows
	previous = nil
	windowsKeyCodes := append(append(append([]int{}, services.KWindowsToLogicalKeyKeys...), services.KWindowsNumPadMapKeys...), 0xe5)
	for i := 0; i < n; i++ {
		var keyCode, scanCode int
		if r.next()%5 == 0 {
			keyCode = r.next() % 300
		} else {
			keyCode = pickOf(r, windowsKeyCodes)
		}
		if r.next()%4 == 0 {
			scanCode = r.next() % 0x200
		} else {
			scanCode = pickOf(r, services.KWindowsToPhysicalKeyKeys)
		}
		cp := pickOf(r, rawCodePoints)
		modifiers := r.next() % 0x4000
		data := services.NewRawKeyEventDataWindows(services.RawKeyEventDataWindowsOptions{KeyCode: keyCode, ScanCode: scanCode,
			CharacterCodePoint: cp, Modifiers: modifiers})
		describeRawData(&lines, fmt.Sprintf("windows %d", i), data, previous, map[string]any{"type": typeOf(i), "keymap": "windows",
			"keyCode": keyCode, "scanCode": scanCode, "characterCodePoint": cp, "modifiers": modifiers})
		previous = data
	}
	// Web
	previous = nil
	webCodes := append(append([]string{}, services.KWebToPhysicalKeyKeys...), "UnknownCode", "Foo", "")
	webKeys := append(append(append(append([]string{}, services.KWebToLogicalKeyKeys...), services.KWebLocationMapKeys...), rawStrings...), "Unidentified")
	for i := 0; i < n; i++ {
		code := pickOf(r, webCodes)
		key := pickOf(r, webKeys)
		location := r.next() % 4
		metaState := r.next() % 0x80
		keyCode := r.next() % 200
		data := services.NewRawKeyEventDataWeb(services.RawKeyEventDataWebOptions{Code: code, Key: key, Location: location,
			MetaState: metaState, KeyCode: keyCode})
		describeRawData(&lines, fmt.Sprintf("web %d", i), data, previous, map[string]any{"type": typeOf(i), "keymap": "web",
			"code": code, "key": key, "location": location, "metaState": metaState, "keyCode": keyCode})
		previous = data
	}
	lines = append(lines, "unknown keymap "+keyGuard(func() any {
		return services.RawKeyEventFromMessage(map[string]any{"type": "keydown", "keymap": "beos"})
	}))
	lines = append(lines, "unknown type "+keyGuard(func() any {
		return services.RawKeyEventFromMessage(map[string]any{"type": "keypress", "keymap": "android"})
	}))
	// The known keys.
	for _, key := range knownLogical {
		var syn []string
		for _, s := range key.Synonyms() {
			syn = append(syn, strconv.Itoa(s.KeyID))
		}
		lines = append(lines, fmt.Sprintf("logical %s %s %s %v %s", stripHashes(key.String()), dart.JSONEncode(key.KeyLabel()),
			*key.DebugName(), key.IsAutogenerated(), strings.Join(syn, ",")))
	}
	for key := range services.PhysicalKeyboardKeyKnownPhysicalKeys() {
		lines = append(lines, fmt.Sprintf("physical %s %s", stripHashes(key.String()), *key.DebugName()))
	}
	for _, id := range []int{0x41, 0xd9, 0xf9, 0x1f600, 0x100012345, 0x1100000042, 0x10ffff, 0xd800} {
		key := services.NewLogicalKeyboardKey(id)
		lines = append(lines, fmt.Sprintf("logical %d %s %s %s", id, keyGuard(func() any { return key.KeyLabel() }),
			keyGuard(func() any { return key.DebugName() }), keyGuard(func() any { return stripHashes(key.String()) })))
	}
	join := func(s string) string {
		var parts []string
		for _, u := range dart.StringCodeUnits(s) {
			parts = append(parts, strconv.Itoa(int(u)))
		}
		return strings.Join(parts, ",")
	}
	for c := 0; c < 0x20000; c++ {
		s := dart.StringFromCharCode(c)
		up := dart.StringToUpperCase(s)
		low := dart.StringToLowerCase(s)
		if join(up) != join(s) || join(low) != join(s) {
			lines = append(lines, fmt.Sprintf("case %d %s %s", c, join(up), join(low)))
		}
	}
	for _, s := range append(append(append([]string{}, webCodes...), rawStrings...), "hello world", strings.Repeat("a", 100)) {
		lines = append(lines, fmt.Sprintf("hash %s %d", dart.JSONEncode(s), dart.StringHashCode(s)))
	}
	for _, c := range append(append([]int{}, rawCodePoints...), 0xff21, 0xd7ff, 0x10400) {
		if c <= 0x10ffff {
			lines = append(lines, fmt.Sprintf("runeToLowerCase %d %s", c, keyGuard(func() any { return services.RuneToLowerCase(c) })))
		}
	}
	return lines
}

func compareKeyLines(t *testing.T, ref string, got []string) {
	t.Helper()
	data, err := os.ReadFile(ref)
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want struct{ Lines []string }
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	errors := 0
	for i := 0; i < max(len(got), len(want.Lines)); i++ {
		var g, w string
		if i < len(got) {
			g = got[i]
		}
		if i < len(want.Lines) {
			w = want.Lines[i]
		}
		if g != w {
			t.Errorf("line %d:\nflutter %s\ngo      %s", i, w, g)
			if errors++; errors > 10 {
				t.Fatal("too many differences")
			}
		}
	}
}

func TestRawKeyEventDataMatchesFlutter(t *testing.T) {
	compareKeyLines(t, "flutter/build/raw_key_event_data.json", rawKeyEventDataLines())
}

var _ = keysOf[int, int]
