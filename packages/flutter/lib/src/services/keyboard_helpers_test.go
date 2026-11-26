package services_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

var (
	modesAll                  = []services.KeyDataTransitMode{services.KeyDataTransitModeRawKeyData, services.KeyDataTransitModeKeyDataThenRawKeyData}
	modeRawKeyData            = []services.KeyDataTransitMode{services.KeyDataTransitModeRawKeyData}
	modeKeyDataThenRawKeyData = []services.KeyDataTransitMode{services.KeyDataTransitModeKeyDataThenRawKeyData}
	noVariant                 []services.KeyDataTransitMode
)

// keyTest is testWidgets for the keyboard tests: body runs as an async test
// body once per transit mode of the variant (once without one), with
// flutter test's default platform (android), and the keyboards' state is
// reset afterwards as TestWidgetsFlutterBinding and the variant's tearDown
// do.
func keyTest(t *testing.T, modes []services.KeyDataTransitMode, body func()) {
	t.Helper()
	b := ensureInitialized()
	t.Setenv("FLUTTER_TEST", "true")
	run := func(t *testing.T) {
		defer func() {
			b.KeyEventManager().KeyMessageHandler = nil
			services.RawKeyboardInstance.ClearKeysPressed()
			services.HardwareKeyboardInstance().ClearState()
			b.KeyEventManager().ClearState()
		}()
		runAsync(t, body)
	}
	if modes == nil {
		run(t)
		return
	}
	for _, mode := range modes {
		variant := flutter_test.KeySimulatorTransitModeVariant{Values: modes}
		t.Run(variant.DescribeValue(mode), func(t *testing.T) {
			memento := variant.SetUp(mode)
			defer variant.TearDown(mode, memento)
			run(t)
		})
	}
}

func keyDown(key services.LogicalKeyboardKey, opts ...flutter_test.SimulateKeyEventOptions) bool {
	return flutter_test.SimulateKeyDownEvent(key, opts...).Await()
}

func keyUp(key services.LogicalKeyboardKey, opts ...flutter_test.SimulateKeyEventOptions) bool {
	return flutter_test.SimulateKeyUpEvent(key, opts...).Await()
}

func keyRepeat(key services.LogicalKeyboardKey, opts ...flutter_test.SimulateKeyEventOptions) bool {
	return flutter_test.SimulateKeyRepeatEvent(key, opts...).Await()
}

func on(platform string) flutter_test.SimulateKeyEventOptions {
	return flutter_test.SimulateKeyEventOptions{Platform: platform}
}

// sameSet is Dart's equals on two Sets (order-insensitive).
func sameSet[T comparable](got []T, want ...T) bool {
	if len(got) != len(want) {
		return false
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			return false
		}
	}
	return true
}

func expectSet[T comparable](what string, got []T, want ...T) {
	expect(sameSet(got, want...), "%s: got %v, want %v", what, got, want)
}

// keyData returns getKeyData's message as a mutable map.
func keyData(key services.LogicalKeyboardKey, platform string, opts ...flutter_test.GetKeyDataOptions) *[]dart.MapEntry[string, any] {
	d := flutter_test.KeyEventSimulator.GetKeyData(key, platform, opts...)
	return &d
}

func setEntry(d *[]dart.MapEntry[string, any], k string, v any) {
	for i := range *d {
		if (*d)[i].Key == k {
			(*d)[i].Value = v
			return
		}
	}
	*d = append(*d, dart.MapEntry[string, any]{Key: k, Value: v})
}

func getEntry(d *[]dart.MapEntry[string, any], k string) any {
	for _, e := range *d {
		if e.Key == k {
			return e.Value
		}
	}
	return nil
}

// asMap is the message as the JSON codec decodes it on the other side.
func asMap(d *[]dart.MapEntry[string, any]) map[string]any {
	m := map[string]any{}
	for _, e := range *d {
		m[e.Key] = e.Value
	}
	return m
}

// sendRawMessage delivers a raw key message on flutter/keyevent, as the
// platform would.
func sendRawMessage(d *[]dart.MapEntry[string, any]) {
	channel := services.SystemChannels.KeyEvent
	messenger().HandlePlatformMessage(channel.Name, channel.Codec.EncodeMessage(*d), func([]byte) {}).Await()
}

func describe(v any) string { return fmt.Sprint(v) }
