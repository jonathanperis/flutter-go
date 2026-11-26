package flutter_test

// Port of flutter_test's event_simulation.dart (Flutter 3.50.1):
// KeyEventSimulator, simulateKeyDownEvent/UpEvent/RepeatEvent and
// KeySimulatorTransitModeVariant.
//
// Differences: there is no TestAsyncUtils guard; the transit-mode variant
// is a plain struct (flutter_test's TestVariant is not ported).

import (
	"slices"
	"strings"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// webKeyLocationPair is a tuple of `key` and `location` from Web's
// KeyboardEvent class.
type webKeyLocationPair struct {
	key      string
	location int
}

// keyLabel is the lower-cased label of a key with a one-character label
// (nil otherwise). This code can only simulate keys which appear in the key
// maps.
func keyLabel(key services.LogicalKeyboardKey) *string {
	label := key.KeyLabel()
	if dart.StringLength(label) == 1 {
		s := dart.StringToLowerCase(label)
		return &s
	}
	return nil
}

type keyEventSimulator struct{}

// KeyEventSimulator is flutter_test's class of static methods simulating
// key events (`KeyEventSimulator.simulateKeyDownEvent(...)` is
// `KeyEventSimulator.SimulateKeyDownEvent(...)`).
var KeyEventSimulator keyEventSimulator

// getKeySynonym returns the left version of a synonym key.
func getKeySynonym(origKey services.LogicalKeyboardKey) services.LogicalKeyboardKey {
	switch origKey {
	case services.LogicalKeyboardKeyShift:
		return services.LogicalKeyboardKeyShiftLeft
	case services.LogicalKeyboardKeyAlt:
		return services.LogicalKeyboardKeyAltLeft
	case services.LogicalKeyboardKeyMeta:
		return services.LogicalKeyboardKeyMetaLeft
	case services.LogicalKeyboardKeyControl:
		return services.LogicalKeyboardKeyControlLeft
	}
	return origKey
}

func osIsSupported(platform string) bool {
	switch platform {
	case "android", "fuchsia", "macos", "linux", "web", "ios", "windows":
		return true
	}
	return false
}

func assertSupported(platform string) {
	if !osIsSupported(platform) {
		panic(dart.NewAssertionError("Platform " + platform + " not supported for key simulation"))
	}
}

func getScanCode(key services.PhysicalKeyboardKey, platform string) int {
	assertSupported(platform)
	var m map[int]services.PhysicalKeyboardKey
	var keys []int
	switch platform {
	case "android":
		m, keys = services.KAndroidToPhysicalKey, services.KAndroidToPhysicalKeyKeys
	case "fuchsia":
		m, keys = services.KFuchsiaToPhysicalKey, services.KFuchsiaToPhysicalKeyKeys
	case "macos":
		m, keys = services.KMacOsToPhysicalKey, services.KMacOsToPhysicalKeyKeys
	case "ios":
		m, keys = services.KIosToPhysicalKey, services.KIosToPhysicalKeyKeys
	case "linux":
		m, keys = services.KLinuxToPhysicalKey, services.KLinuxToPhysicalKeyKeys
	case "windows":
		m, keys = services.KWindowsToPhysicalKey, services.KWindowsToPhysicalKeyKeys
	case "web":
		// web doesn't have int type code
		return -1
	}
	for _, code := range keys {
		if key.UsbHidUsage == m[code].UsbHidUsage {
			return code
		}
	}
	panic(dart.NewAssertionError("Physical key for " + key.String() + " not found in " + platform + " scanCode map"))
}

func getKeyCode(key services.LogicalKeyboardKey, platform string) int {
	assertSupported(platform)
	var m map[int]services.LogicalKeyboardKey
	var keys []int
	switch platform {
	case "android":
		m, keys = services.KAndroidToLogicalKey, services.KAndroidToLogicalKeyKeys
	case "fuchsia":
		m, keys = services.KFuchsiaToLogicalKey, services.KFuchsiaToLogicalKeyKeys
	case "macos", "ios", "web":
		// macOS and iOS don't do key codes, just scan codes; web doesn't
		// have int type codes.
		return -1
	case "linux":
		m, keys = services.KGlfwToLogicalKey, services.KGlfwToLogicalKeyKeys
	case "windows":
		m, keys = services.KWindowsToLogicalKey, services.KWindowsToLogicalKeyKeys
	}
	for _, code := range keys {
		if key.KeyID == m[code].KeyID {
			return code
		}
	}
	panic(dart.NewAssertionError("Key " + key.String() + " not found in " + platform + " keyCode map"))
}

func debugNameEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func inferPhysicalKey(key services.LogicalKeyboardKey) services.PhysicalKeyboardKey {
	for physicalKey := range services.PhysicalKeyboardKeyKnownPhysicalKeys() {
		if debugNameEqual(physicalKey.DebugName(), key.DebugName()) {
			return physicalKey
		}
	}
	panic(dart.NewAssertionError("Unable to infer physical key for " + key.String()))
}

func getWebKeyLocation(key services.LogicalKeyboardKey, keyLabel string) webKeyLocationPair {
	for _, k := range services.KWebLocationMapKeys {
		value := services.KWebLocationMap[k]
		foundIndex := -1
		for i := len(value) - 1; i >= 0; i-- {
			if value[i] != nil && *value[i] == key {
				foundIndex = i
				break
			}
		}
		// -1: the key is not in kWebLocationMap; 0: the key is in the
		// standard part of the keyboard, but keyLabel decides.
		if foundIndex != -1 && foundIndex != 0 {
			return webKeyLocationPair{k, foundIndex}
		}
	}
	if keyLabel != "" {
		return webKeyLocationPair{keyLabel, 0}
	}
	for _, code := range services.KWebToLogicalKeyKeys {
		if key.KeyID == services.KWebToLogicalKey[code].KeyID {
			return webKeyLocationPair{code, 0}
		}
	}
	panic(dart.NewAssertionError("Key " + key.String() + " not found in web keyCode map"))
}

func getWebCode(key services.PhysicalKeyboardKey) string {
	for _, code := range services.KWebToPhysicalKeyKeys {
		if services.KWebToPhysicalKey[code].UsbHidUsage == key.UsbHidUsage {
			return code
		}
	}
	panic(dart.NewAssertionError("Key " + key.String() + " not found in web code map"))
}

func findPhysicalKeyByPlatform(key services.LogicalKeyboardKey, platform string) services.PhysicalKeyboardKey {
	assertSupported(platform)
	var values []services.PhysicalKeyboardKey
	intMap := func(m map[int]services.PhysicalKeyboardKey, keys []int) {
		for _, k := range keys {
			values = append(values, m[k])
		}
	}
	switch platform {
	case "android":
		intMap(services.KAndroidToPhysicalKey, services.KAndroidToPhysicalKeyKeys)
	case "fuchsia":
		intMap(services.KFuchsiaToPhysicalKey, services.KFuchsiaToPhysicalKeyKeys)
	case "macos":
		intMap(services.KMacOsToPhysicalKey, services.KMacOsToPhysicalKeyKeys)
	case "ios":
		intMap(services.KIosToPhysicalKey, services.KIosToPhysicalKeyKeys)
	case "linux":
		intMap(services.KLinuxToPhysicalKey, services.KLinuxToPhysicalKeyKeys)
	case "web":
		for _, k := range services.KWebToPhysicalKeyKeys {
			values = append(values, services.KWebToPhysicalKey[k])
		}
	case "windows":
		intMap(services.KWindowsToPhysicalKey, services.KWindowsToPhysicalKeyKeys)
	}
	for _, physicalKey := range values {
		if debugNameEqual(key.DebugName(), physicalKey.DebugName()) {
			return physicalKey
		}
	}
	panic(dart.NewAssertionError("Physical key for " + key.String() + " not found in " + platform + " physical key map"))
}

// GetKeyDataOptions are getKeyData's optional named parameters (IsDown nil:
// true; Platform is required in Dart and positional here).
type GetKeyDataOptions struct {
	IsDown      *bool
	PhysicalKey *services.PhysicalKeyboardKey
	Character   *string
}

// GetKeyData is KeyEventSimulator.getKeyData: the raw key message (as the
// platform's embedder would send it on flutter/keyevent) for key on
// platform, in Dart's key order.
func (keyEventSimulator) GetKeyData(key services.LogicalKeyboardKey, platform string, opts ...GetKeyDataOptions) []dart.MapEntry[string, any] {
	var o GetKeyDataOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	isDown := o.IsDown == nil || *o.IsDown
	assertSupported(platform)
	key = getKeySynonym(key)
	// Find a suitable physical key if none was supplied.
	var physicalKey services.PhysicalKeyboardKey
	if o.PhysicalKey != nil {
		physicalKey = *o.PhysicalKey
	} else {
		physicalKey = findPhysicalKeyByPlatform(key, platform)
	}
	if key.DebugName() == nil {
		panic(dart.NewAssertionError(nil))
	}
	typ := "keyup"
	if isDown {
		typ = "keydown"
	}
	result := []dart.MapEntry[string, any]{{Key: "type", Value: typ}, {Key: "keymap", Value: platform}}
	set := func(k string, v any) {
		for i := range result {
			if result[i].Key == k {
				result[i].Value = v
				return
			}
		}
		result = append(result, dart.MapEntry[string, any]{Key: k, Value: v})
	}
	resultCharacter := ""
	if o.Character != nil {
		resultCharacter = *o.Character
	} else if l := keyLabel(key); l != nil {
		resultCharacter = *l
	}
	assignWeb := func() {
		keyLocation := getWebKeyLocation(key, resultCharacter)
		actualPhysicalKey := physicalKey // never null here: physicalKey ??= above
		set("code", getWebCode(actualPhysicalKey))
		set("key", keyLocation.key)
		set("location", keyLocation.location)
		set("metaState", getWebModifierFlags(key, isDown))
	}
	keyCode := getKeyCode(key, platform)
	scanCode := getScanCode(physicalKey, platform)
	switch platform {
	case "android":
		set("keyCode", keyCode)
		if resultCharacter != "" {
			set("codePoint", dart.CodeUnitAt(resultCharacter, 0))
			set("character", resultCharacter)
		}
		set("scanCode", scanCode)
		set("metaState", getAndroidModifierFlags(key, isDown))
	case "fuchsia":
		set("hidUsage", physicalKey.UsbHidUsage)
		if resultCharacter != "" {
			set("codePoint", dart.CodeUnitAt(resultCharacter, 0))
		}
		set("modifiers", getFuchsiaModifierFlags(key, isDown))
	case "linux":
		set("toolkit", "glfw")
		set("keyCode", keyCode)
		set("scanCode", scanCode)
		set("modifiers", getGlfwModifierFlags(key, isDown))
		usv := 0
		if resultCharacter != "" {
			usv = dart.CodeUnitAt(resultCharacter, 0)
		}
		set("unicodeScalarValues", usv)
	case "macos":
		set("keyCode", scanCode)
		if resultCharacter != "" {
			set("characters", resultCharacter)
			set("charactersIgnoringModifiers", resultCharacter)
		}
		set("modifiers", getMacOsModifierFlags(key, isDown))
	case "ios":
		set("keyCode", scanCode)
		set("characters", resultCharacter)
		set("charactersIgnoringModifiers", resultCharacter)
		set("modifiers", getIOSModifierFlags(key, isDown))
	case "windows":
		set("keyCode", keyCode)
		set("scanCode", scanCode)
		if resultCharacter != "" {
			set("characterCodePoint", dart.CodeUnitAt(resultCharacter, 0))
		}
		set("modifiers", getWindowsModifierFlags(key, isDown))
	case "web":
		assignWeb()
	}
	return result
}

// pressedAfter is RawKeyboard.instance.keysPressed with newKey added (down)
// or removed (up).
func pressedAfter(newKey services.LogicalKeyboardKey, isDown bool) []services.LogicalKeyboardKey {
	pressed := services.RawKeyboardInstance.KeysPressed()
	if isDown {
		if !slices.Contains(pressed, newKey) {
			pressed = append(pressed, newKey)
		}
	} else if i := slices.Index(pressed, newKey); i >= 0 {
		pressed = slices.Delete(pressed, i, i+1)
	}
	return pressed
}

type modifierBits struct {
	key  services.LogicalKeyboardKey
	bits int
}

func flagsFor(pressed []services.LogicalKeyboardKey, table []modifierBits) int {
	result := 0
	for _, e := range table {
		if slices.Contains(pressed, e.key) {
			result |= e.bits
		}
	}
	return result
}

func getAndroidModifierFlags(newKey services.LogicalKeyboardKey, isDown bool) int {
	return flagsFor(pressedAfter(newKey, isDown), []modifierBits{
		{services.LogicalKeyboardKeyShiftLeft, services.RawKeyEventDataAndroidModifierLeftShift | services.RawKeyEventDataAndroidModifierShift},
		{services.LogicalKeyboardKeyShiftRight, services.RawKeyEventDataAndroidModifierRightShift | services.RawKeyEventDataAndroidModifierShift},
		{services.LogicalKeyboardKeyMetaLeft, services.RawKeyEventDataAndroidModifierLeftMeta | services.RawKeyEventDataAndroidModifierMeta},
		{services.LogicalKeyboardKeyMetaRight, services.RawKeyEventDataAndroidModifierRightMeta | services.RawKeyEventDataAndroidModifierMeta},
		{services.LogicalKeyboardKeyControlLeft, services.RawKeyEventDataAndroidModifierLeftControl | services.RawKeyEventDataAndroidModifierControl},
		{services.LogicalKeyboardKeyControlRight, services.RawKeyEventDataAndroidModifierRightControl | services.RawKeyEventDataAndroidModifierControl},
		{services.LogicalKeyboardKeyAltLeft, services.RawKeyEventDataAndroidModifierLeftAlt | services.RawKeyEventDataAndroidModifierAlt},
		{services.LogicalKeyboardKeyAltRight, services.RawKeyEventDataAndroidModifierRightAlt | services.RawKeyEventDataAndroidModifierAlt},
		{services.LogicalKeyboardKeyFn, services.RawKeyEventDataAndroidModifierFunction},
		{services.LogicalKeyboardKeyScrollLock, services.RawKeyEventDataAndroidModifierScrollLock},
		{services.LogicalKeyboardKeyNumLock, services.RawKeyEventDataAndroidModifierNumLock},
		{services.LogicalKeyboardKeyCapsLock, services.RawKeyEventDataAndroidModifierCapsLock},
	})
}

func getGlfwModifierFlags(newKey services.LogicalKeyboardKey, isDown bool) int {
	return flagsFor(pressedAfter(newKey, isDown), []modifierBits{
		{services.LogicalKeyboardKeyShiftLeft, services.GLFWKeyHelperModifierShift},
		{services.LogicalKeyboardKeyShiftRight, services.GLFWKeyHelperModifierShift},
		{services.LogicalKeyboardKeyMetaLeft, services.GLFWKeyHelperModifierMeta},
		{services.LogicalKeyboardKeyMetaRight, services.GLFWKeyHelperModifierMeta},
		{services.LogicalKeyboardKeyControlLeft, services.GLFWKeyHelperModifierControl},
		{services.LogicalKeyboardKeyControlRight, services.GLFWKeyHelperModifierControl},
		{services.LogicalKeyboardKeyAltLeft, services.GLFWKeyHelperModifierAlt},
		{services.LogicalKeyboardKeyAltRight, services.GLFWKeyHelperModifierAlt},
		{services.LogicalKeyboardKeyCapsLock, services.GLFWKeyHelperModifierCapsLock},
	})
}

func getWindowsModifierFlags(newKey services.LogicalKeyboardKey, isDown bool) int {
	return flagsFor(pressedAfter(newKey, isDown), []modifierBits{
		{services.LogicalKeyboardKeyShift, services.RawKeyEventDataWindowsModifierShift},
		{services.LogicalKeyboardKeyShiftLeft, services.RawKeyEventDataWindowsModifierLeftShift},
		{services.LogicalKeyboardKeyShiftRight, services.RawKeyEventDataWindowsModifierRightShift},
		{services.LogicalKeyboardKeyMetaLeft, services.RawKeyEventDataWindowsModifierLeftMeta},
		{services.LogicalKeyboardKeyMetaRight, services.RawKeyEventDataWindowsModifierRightMeta},
		{services.LogicalKeyboardKeyControl, services.RawKeyEventDataWindowsModifierControl},
		{services.LogicalKeyboardKeyControlLeft, services.RawKeyEventDataWindowsModifierLeftControl},
		{services.LogicalKeyboardKeyControlRight, services.RawKeyEventDataWindowsModifierRightControl},
		{services.LogicalKeyboardKeyAlt, services.RawKeyEventDataWindowsModifierAlt},
		{services.LogicalKeyboardKeyAltLeft, services.RawKeyEventDataWindowsModifierLeftAlt},
		{services.LogicalKeyboardKeyAltRight, services.RawKeyEventDataWindowsModifierRightAlt},
		{services.LogicalKeyboardKeyCapsLock, services.RawKeyEventDataWindowsModifierCaps},
		{services.LogicalKeyboardKeyNumLock, services.RawKeyEventDataWindowsModifierNumLock},
		{services.LogicalKeyboardKeyScrollLock, services.RawKeyEventDataWindowsModifierScrollLock},
	})
}

func getFuchsiaModifierFlags(newKey services.LogicalKeyboardKey, isDown bool) int {
	return flagsFor(pressedAfter(newKey, isDown), []modifierBits{
		{services.LogicalKeyboardKeyShiftLeft, services.RawKeyEventDataFuchsiaModifierLeftShift},
		{services.LogicalKeyboardKeyShiftRight, services.RawKeyEventDataFuchsiaModifierRightShift},
		{services.LogicalKeyboardKeyMetaLeft, services.RawKeyEventDataFuchsiaModifierLeftMeta},
		{services.LogicalKeyboardKeyMetaRight, services.RawKeyEventDataFuchsiaModifierRightMeta},
		{services.LogicalKeyboardKeyControlLeft, services.RawKeyEventDataFuchsiaModifierLeftControl},
		{services.LogicalKeyboardKeyControlRight, services.RawKeyEventDataFuchsiaModifierRightControl},
		{services.LogicalKeyboardKeyAltLeft, services.RawKeyEventDataFuchsiaModifierLeftAlt},
		{services.LogicalKeyboardKeyAltRight, services.RawKeyEventDataFuchsiaModifierRightAlt},
		{services.LogicalKeyboardKeyCapsLock, services.RawKeyEventDataFuchsiaModifierCapsLock},
	})
}

func getWebModifierFlags(newKey services.LogicalKeyboardKey, isDown bool) int {
	return flagsFor(pressedAfter(newKey, isDown), []modifierBits{
		{services.LogicalKeyboardKeyShiftLeft, services.RawKeyEventDataWebModifierShift},
		{services.LogicalKeyboardKeyShiftRight, services.RawKeyEventDataWebModifierShift},
		{services.LogicalKeyboardKeyMetaLeft, services.RawKeyEventDataWebModifierMeta},
		{services.LogicalKeyboardKeyMetaRight, services.RawKeyEventDataWebModifierMeta},
		{services.LogicalKeyboardKeyControlLeft, services.RawKeyEventDataWebModifierControl},
		{services.LogicalKeyboardKeyControlRight, services.RawKeyEventDataWebModifierControl},
		{services.LogicalKeyboardKeyAltLeft, services.RawKeyEventDataWebModifierAlt},
		{services.LogicalKeyboardKeyAltRight, services.RawKeyEventDataWebModifierAlt},
		{services.LogicalKeyboardKeyCapsLock, services.RawKeyEventDataWebModifierCapsLock},
		{services.LogicalKeyboardKeyNumLock, services.RawKeyEventDataWebModifierNumLock},
		{services.LogicalKeyboardKeyScrollLock, services.RawKeyEventDataWebModifierScrollLock},
	})
}

var appleFunctionKeys = []services.LogicalKeyboardKey{
	services.LogicalKeyboardKeyF1, services.LogicalKeyboardKeyF2, services.LogicalKeyboardKeyF3,
	services.LogicalKeyboardKeyF4, services.LogicalKeyboardKeyF5, services.LogicalKeyboardKeyF6,
	services.LogicalKeyboardKeyF7, services.LogicalKeyboardKeyF8, services.LogicalKeyboardKeyF9,
	services.LogicalKeyboardKeyF10, services.LogicalKeyboardKeyF11, services.LogicalKeyboardKeyF12,
	services.LogicalKeyboardKeyF13, services.LogicalKeyboardKeyF14, services.LogicalKeyboardKeyF15,
	services.LogicalKeyboardKeyF16, services.LogicalKeyboardKeyF17, services.LogicalKeyboardKeyF18,
	services.LogicalKeyboardKeyF19, services.LogicalKeyboardKeyF20, services.LogicalKeyboardKeyF21,
}

func intersects(pressed, set []services.LogicalKeyboardKey) bool {
	for _, k := range pressed {
		if slices.Contains(set, k) {
			return true
		}
	}
	return false
}

func macOsNumPadKeys() []services.LogicalKeyboardKey {
	var out []services.LogicalKeyboardKey
	for _, k := range services.KMacOsNumPadMapKeys {
		out = append(out, services.KMacOsNumPadMap[k])
	}
	return out
}

func getMacOsModifierFlags(newKey services.LogicalKeyboardKey, isDown bool) int {
	pressed := pressedAfter(newKey, isDown)
	result := flagsFor(pressed, []modifierBits{
		{services.LogicalKeyboardKeyShiftLeft, services.RawKeyEventDataMacOsModifierLeftShift | services.RawKeyEventDataMacOsModifierShift},
		{services.LogicalKeyboardKeyShiftRight, services.RawKeyEventDataMacOsModifierRightShift | services.RawKeyEventDataMacOsModifierShift},
		{services.LogicalKeyboardKeyMetaLeft, services.RawKeyEventDataMacOsModifierLeftCommand | services.RawKeyEventDataMacOsModifierCommand},
		{services.LogicalKeyboardKeyMetaRight, services.RawKeyEventDataMacOsModifierRightCommand | services.RawKeyEventDataMacOsModifierCommand},
		{services.LogicalKeyboardKeyControlLeft, services.RawKeyEventDataMacOsModifierLeftControl | services.RawKeyEventDataMacOsModifierControl},
		{services.LogicalKeyboardKeyControlRight, services.RawKeyEventDataMacOsModifierRightControl | services.RawKeyEventDataMacOsModifierControl},
		{services.LogicalKeyboardKeyAltLeft, services.RawKeyEventDataMacOsModifierLeftOption | services.RawKeyEventDataMacOsModifierOption},
		{services.LogicalKeyboardKeyAltRight, services.RawKeyEventDataMacOsModifierRightOption | services.RawKeyEventDataMacOsModifierOption},
	})
	if intersects(pressed, appleFunctionKeys) {
		result |= services.RawKeyEventDataMacOsModifierFunction
	}
	if intersects(pressed, macOsNumPadKeys()) {
		result |= services.RawKeyEventDataMacOsModifierNumericPad
	}
	if slices.Contains(pressed, services.LogicalKeyboardKeyCapsLock) {
		result |= services.RawKeyEventDataMacOsModifierCapsLock
	}
	return result
}

func getIOSModifierFlags(newKey services.LogicalKeyboardKey, isDown bool) int {
	pressed := pressedAfter(newKey, isDown)
	result := flagsFor(pressed, []modifierBits{
		{services.LogicalKeyboardKeyShiftLeft, services.RawKeyEventDataIosModifierLeftShift | services.RawKeyEventDataIosModifierShift},
		{services.LogicalKeyboardKeyShiftRight, services.RawKeyEventDataIosModifierRightShift | services.RawKeyEventDataIosModifierShift},
		{services.LogicalKeyboardKeyMetaLeft, services.RawKeyEventDataIosModifierLeftCommand | services.RawKeyEventDataIosModifierCommand},
		{services.LogicalKeyboardKeyMetaRight, services.RawKeyEventDataIosModifierRightCommand | services.RawKeyEventDataIosModifierCommand},
		{services.LogicalKeyboardKeyControlLeft, services.RawKeyEventDataIosModifierLeftControl | services.RawKeyEventDataIosModifierControl},
		{services.LogicalKeyboardKeyControlRight, services.RawKeyEventDataIosModifierRightControl | services.RawKeyEventDataIosModifierControl},
		{services.LogicalKeyboardKeyAltLeft, services.RawKeyEventDataIosModifierLeftOption | services.RawKeyEventDataIosModifierOption},
		{services.LogicalKeyboardKeyAltRight, services.RawKeyEventDataIosModifierRightOption | services.RawKeyEventDataIosModifierOption},
	})
	if intersects(pressed, appleFunctionKeys) {
		result |= services.RawKeyEventDataIosModifierFunction
	}
	// Flutter's simulator uses the macOS numpad map for iOS too.
	if intersects(pressed, macOsNumPadKeys()) {
		result |= services.RawKeyEventDataIosModifierNumericPad
	}
	if slices.Contains(pressed, services.LogicalKeyboardKeyCapsLock) {
		result |= services.RawKeyEventDataIosModifierCapsLock
	}
	return result
}

// simulateKeyEventByRawEvent sends the raw key message to the framework (as
// the platform would, through the test messenger) and completes with its
// "handled" answer (false without one).
func simulateKeyEventByRawEvent(buildKeyData func() []dart.MapEntry[string, any]) async.Future[bool] {
	return async.Async(func() bool {
		result := async.NewCompleter[bool]()
		messenger := TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger()
		channel := services.SystemChannels.KeyEvent
		messenger.HandlePlatformMessage(channel.Name, channel.Codec.EncodeMessage(buildKeyData()), func(data []byte) {
			if data == nil {
				result.Complete(false)
				return
			}
			decoded := channel.Codec.DecodeMessage(data).(*dart.LinkedMap)
			result.Complete(decoded.Lookup("handled").(bool))
		}).Await()
		return result.Future().Await()
	})
}

var debugNameToPhysicalKey = func() map[string]services.PhysicalKeyboardKey {
	result := map[string]services.PhysicalKeyboardKey{}
	for key := range services.PhysicalKeyboardKeyKnownPhysicalKeys() {
		if debugName := key.DebugName(); debugName != nil {
			result[*debugName] = key
		}
	}
	return result
}()

func findPhysicalKey(key services.LogicalKeyboardKey) services.PhysicalKeyboardKey {
	if name := key.DebugName(); name != nil {
		if result, ok := debugNameToPhysicalKey[*name]; ok {
			return result
		}
	}
	panic(dart.NewAssertionError("Physical key for " + key.String() + " not found in known physical keys"))
}

const defaultTransitMode = services.KeyDataTransitModeKeyDataThenRawKeyData

// TransitMode is KeyEventSimulator.transitMode: the mode simulated key
// events are delivered in (DebugKeyEventSimulatorTransitModeOverride, or
// keyDataThenRawKeyData).
func (keyEventSimulator) TransitMode() services.KeyDataTransitMode {
	if foundation.KDebugMode && services.DebugKeyEventSimulatorTransitModeOverride != nil {
		return *services.DebugKeyEventSimulatorTransitModeOverride
	}
	return defaultTransitMode
}

func defaultPlatform() string {
	return strings.ToLower(foundation.DefaultTargetPlatform().Name())
}

// SimulateKeyEventOptions are the simulate functions' optional named
// parameters (Platform "" is Dart's null: the default target platform;
// simulateKeyUpEvent ignores Character).
type SimulateKeyEventOptions struct {
	Platform    string
	PhysicalKey *services.PhysicalKeyboardKey
	Character   *string
}

func simulateOpts(opts []SimulateKeyEventOptions) SimulateKeyEventOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return SimulateKeyEventOptions{}
}

func (s keyEventSimulator) simulate(key services.LogicalKeyboardKey, typ ui.KeyEventType, o SimulateKeyEventOptions) async.Future[bool] {
	simulateByRawEvent := func() async.Future[bool] {
		return simulateKeyEventByRawEvent(func() []dart.MapEntry[string, any] {
			platform := o.Platform
			if platform == "" {
				platform = defaultPlatform()
			}
			isDown := typ != ui.KeyEventTypeUp
			character := o.Character
			if !isDown {
				character = nil
			}
			return s.GetKeyData(key, platform, GetKeyDataOptions{IsDown: &isDown, PhysicalKey: o.PhysicalKey, Character: character})
		})
	}
	if s.TransitMode() == services.KeyDataTransitModeRawKeyData {
		return simulateByRawEvent()
	}
	return async.Async(func() bool {
		logicalKey := getKeySynonym(key)
		physical := o.PhysicalKey
		if physical == nil {
			p := findPhysicalKey(logicalKey)
			physical = &p
		}
		var character *string
		if typ != ui.KeyEventTypeUp {
			character = o.Character
			if character == nil {
				character = keyLabel(key)
			}
		}
		resultByKeyEvent := services.ServicesBindingInstance().KeyEventManager().HandleKeyData(ui.KeyData{
			Type:        typ,
			Physical:    physical.UsbHidUsage,
			Logical:     logicalKey.KeyID,
			TimeStamp:   time.Duration(0),
			Character:   character,
			Synthesized: false,
		})
		return simulateByRawEvent().Await() || resultByKeyEvent
	})
}

// SimulateKeyDownEvent simulates a key down of key (its physical key and
// character inferred unless given), delivered as the transit mode says; it
// completes with whether the framework handled it.
func (s keyEventSimulator) SimulateKeyDownEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) async.Future[bool] {
	return s.simulate(key, ui.KeyEventTypeDown, simulateOpts(opts))
}

// SimulateKeyUpEvent simulates a key up of key.
func (s keyEventSimulator) SimulateKeyUpEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) async.Future[bool] {
	return s.simulate(key, ui.KeyEventTypeUp, simulateOpts(opts))
}

// SimulateKeyRepeatEvent simulates a key repeat of key.
func (s keyEventSimulator) SimulateKeyRepeatEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) async.Future[bool] {
	return s.simulate(key, ui.KeyEventTypeRepeat, simulateOpts(opts))
}

// SimulateKeyDownEvent is simulateKeyDownEvent: KeyEventSimulator's, then,
// when the framework didn't handle the key and the binding is the test
// binding, TestTextInput's handleKeyDownEvent (the platform's text input
// may respond to it, e.g. macOS's key bindings).
func SimulateKeyDownEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) async.Future[bool] {
	return async.Async(func() bool {
		handled := KeyEventSimulator.SimulateKeyDownEvent(key, opts...).Await()
		if b, ok := services.ServicesBindingInstance().(*AutomatedTestWidgetsFlutterBinding); !handled && ok {
			b.TestTextInput().HandleKeyDownEvent(key).Await()
		}
		return handled
	})
}

// SimulateKeyUpEvent is simulateKeyUpEvent (with TestTextInput's
// handleKeyUpEvent for an unhandled key, as SimulateKeyDownEvent).
func SimulateKeyUpEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) async.Future[bool] {
	return async.Async(func() bool {
		handled := KeyEventSimulator.SimulateKeyUpEvent(key, opts...).Await()
		if b, ok := services.ServicesBindingInstance().(*AutomatedTestWidgetsFlutterBinding); !handled && ok {
			b.TestTextInput().HandleKeyUpEvent(key).Await()
		}
		return handled
	})
}

// SimulateKeyRepeatEvent is simulateKeyRepeatEvent.
func SimulateKeyRepeatEvent(key services.LogicalKeyboardKey, opts ...SimulateKeyEventOptions) async.Future[bool] {
	return KeyEventSimulator.SimulateKeyRepeatEvent(key, opts...)
}

// KeySimulatorTransitModeVariant runs a test once per transit mode (a
// TestVariant in Dart; deprecated).
type KeySimulatorTransitModeVariant struct {
	// Values are the modes to run with.
	Values []services.KeyDataTransitMode
}

// NewKeySimulatorTransitModeVariantAll is KeySimulatorTransitModeVariant.all().
func NewKeySimulatorTransitModeVariantAll() KeySimulatorTransitModeVariant {
	return KeySimulatorTransitModeVariant{Values: slices.Clone(services.KeyDataTransitModeValues)}
}

// NewKeySimulatorTransitModeVariantKeyDataThenRawKeyData is
// KeySimulatorTransitModeVariant.keyDataThenRawKeyData().
func NewKeySimulatorTransitModeVariantKeyDataThenRawKeyData() KeySimulatorTransitModeVariant {
	return KeySimulatorTransitModeVariant{Values: []services.KeyDataTransitMode{services.KeyDataTransitModeKeyDataThenRawKeyData}}
}

// NewKeySimulatorTransitModeVariantRawKeyData is
// KeySimulatorTransitModeVariant.rawKeyData().
func NewKeySimulatorTransitModeVariantRawKeyData() KeySimulatorTransitModeVariant {
	return KeySimulatorTransitModeVariant{Values: []services.KeyDataTransitMode{services.KeyDataTransitModeRawKeyData}}
}

// DescribeValue names a mode in test descriptions.
func (KeySimulatorTransitModeVariant) DescribeValue(value services.KeyDataTransitMode) string {
	if value == services.KeyDataTransitModeRawKeyData {
		return "RawKeyEvent"
	}
	return "ui.KeyData then RawKeyEvent"
}

// SetUp sets the simulator's mode; it returns the previous override.
func (KeySimulatorTransitModeVariant) SetUp(value services.KeyDataTransitMode) *services.KeyDataTransitMode {
	previous := services.DebugKeyEventSimulatorTransitModeOverride
	v := value
	services.DebugKeyEventSimulatorTransitModeOverride = &v
	return previous
}

// TearDown clears the keyboards' state and restores the override.
func (KeySimulatorTransitModeVariant) TearDown(value services.KeyDataTransitMode, memento *services.KeyDataTransitMode) {
	services.RawKeyboardInstance.ClearKeysPressed()
	services.HardwareKeyboardInstance().ClearState()
	services.ServicesBindingInstance().KeyEventManager().ClearState()
	services.DebugKeyEventSimulatorTransitModeOverride = memento
}
