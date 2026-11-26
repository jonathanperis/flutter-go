package services_test

// Own tests: Dart's Maps keep their insertion order through the codecs
// (decoded maps are *dart.LinkedMap, written back in their order), as
// message_codecs.dart's `<Object?, Object?>{}` result and `value.forEach`
// writer do. Flutter's own map tests are in message_codecs_test.go.

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestStandardMessageCodecWritesMapsInInsertionOrder(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	m := dart.NewLinkedMapOf("c", 1, "a", 2, "b", 3)
	want := []byte{13, 3, 7, 1, 'c', 3, 1, 0, 0, 0, 7, 1, 'a', 3, 2, 0, 0, 0, 7, 1, 'b', 3, 3, 0, 0, 0}
	if got := standard.EncodeMessage(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("got % x", got)
	}
	// Removing and setting again moves a key last.
	m.Remove("c")
	m.Set("c", 1)
	want = []byte{13, 3, 7, 1, 'a', 3, 2, 0, 0, 0, 7, 1, 'b', 3, 3, 0, 0, 0, 7, 1, 'c', 3, 1, 0, 0, 0}
	if got := standard.EncodeMessage(m); !reflect.DeepEqual(got, want) {
		t.Fatalf("got % x", got)
	}
	// A nil *dart.LinkedMap is null; a Go map is written sorted.
	if got := standard.EncodeMessage([]any{(*dart.LinkedMap)(nil)}); !reflect.DeepEqual(got, []byte{12, 1, 0}) {
		t.Fatalf("got % x", got)
	}
	if got, want := standard.EncodeMessage(map[string]int{"c": 1, "a": 2, "b": 3}),
		standard.EncodeMessage(dart.NewLinkedMapOf("a", 2, "b", 3, "c", 1)); !reflect.DeepEqual(got, want) {
		t.Fatalf("got % x, want % x", got, want)
	}
}

func TestStandardMessageCodecDecodesMapsInMessageOrder(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	for _, keys := range [][]any{
		{"c", "a", "b"},
		{3, 1, 2, -1},
		{true, nil, false, 1.5, "s", 0},
		{"z", []any{1}, 2.5, []byte{1}, dart.NewLinkedMapOf("k", 1)},
	} {
		m := &dart.LinkedMap{}
		for i, k := range keys {
			m.Set(k, i)
		}
		encoded := standard.EncodeMessage(m)
		decoded, ok := standard.DecodeMessage(encoded).(*dart.LinkedMap)
		if !ok {
			t.Fatalf("%v: decoded %T", keys, decoded)
		}
		if got, want := dart.ToString(decoded), m.String(); got != want {
			t.Errorf("decoded %s, want %s", got, want)
		}
		for i, k := range slices.Collect(decoded.Keys()) {
			if v, _ := decoded.Get(k); v != i {
				t.Errorf("%v: key %d (%v) holds %v", keys, i, k, v)
			}
		}
		if again := standard.EncodeMessage(decoded); !reflect.DeepEqual(again, encoded) {
			t.Errorf("re-encoded % x, want % x", again, encoded)
		}
	}
	// Nested maps keep each level's order, through the method codec too.
	nested := dart.NewLinkedMapOf("outer2", dart.NewLinkedMapOf("y", 1, "x", 2), "outer1", []any{dart.NewLinkedMapOf(2, "b", 1, "a")})
	method := services.NewStandardMethodCodec()
	call := method.DecodeMethodCall(method.EncodeMethodCall(services.NewMethodCall("m", nested)))
	if got := call.String(); got != "MethodCall(m, {outer2: {y: 1, x: 2}, outer1: [{2: b, 1: a}]})" {
		t.Errorf("got %s", got)
	}
	if got := dart.ToString(method.DecodeEnvelope(method.EncodeSuccessEnvelope(nested))); got != "{outer2: {y: 1, x: 2}, outer1: [{2: b, 1: a}]}" {
		t.Errorf("got %s", got)
	}
}

func TestJSONCodecsKeepObjectOrder(t *testing.T) {
	json := services.JSONMessageCodec{}
	decoded, ok := json.DecodeMessage([]byte(`{"z":1,"a":{"y":[{"q":1,"p":2}],"b":null},"m":3,"z":4}`)).(*dart.LinkedMap)
	if !ok {
		t.Fatalf("decoded %T", decoded)
	}
	if got := decoded.String(); got != "{z: 4, a: {y: [{q: 1, p: 2}], b: null}, m: 3}" {
		t.Errorf("got %s", got)
	}
	if got := string(json.EncodeMessage(decoded)); got != `{"z":4,"a":{"y":[{"q":1,"p":2}],"b":null},"m":3}` {
		t.Errorf("got %s", got)
	}
	method := services.JSONMethodCodec{}
	call := method.DecodeMethodCall([]byte(`{"args":{"b":1,"a":2},"method":"m"}`))
	if got := call.String(); call.Method != "m" || got != "MethodCall(m, {b: 1, a: 2})" {
		t.Errorf("got %s", got)
	}
	if got := string(method.EncodeMethodCall(call)); got != `{"method":"m","args":{"b":1,"a":2}}` {
		t.Errorf("got %s", got)
	}
	if got := dart.ToString(method.DecodeEnvelope([]byte(`[{"d":1,"c":2}]`))); got != "{d: 1, c: 2}" {
		t.Errorf("got %s", got)
	}
}

// The services that read decoded maps follow the platform's order.
func TestDecodedMapOrderReachesServices(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		channel := services.NewMethodChannel("flutter/processtext")
		messenger().SetMockMethodCallHandler(channel, func(services.MethodCall) async.Future[any] {
			return async.FutureValue[any](dart.NewLinkedMapOf("com.z", "Z", "com.a", "A", "com.m", "M"))
		})
		defer messenger().SetMockMethodCallHandler(channel, nil)
		actions := services.NewDefaultProcessTextService().QueryTextActions().Await()
		var ids []string
		for _, a := range actions {
			ids = append(ids, a.ID+":"+a.Label)
		}
		expect(slices.Equal(ids, []string{"com.z:Z", "com.a:A", "com.m:M"}), "actions %v", ids)

		messenger().SetMockMethodCallHandler(services.SystemChannels.Keyboard, func(services.MethodCall) async.Future[any] {
			return async.FutureValue[any](dart.NewLinkedMapOf(0x7001d, 0x7a, 0x70004, 0x61, 0x700e1, 0x200000102))
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.Keyboard, func(services.MethodCall) async.Future[any] {
			return async.FutureValue[any](&dart.LinkedMap{})
		})
		keyboard := services.HardwareKeyboardInstance()
		keyboard.ClearState()
		defer keyboard.ClearState()
		keyboard.SyncKeyboardState().Await()
		var pressed []int
		for _, k := range keyboard.PhysicalKeysPressed() {
			pressed = append(pressed, k.UsbHidUsage)
		}
		expect(slices.Equal(pressed, []int{0x7001d, 0x70004, 0x700e1}), "pressed %x", pressed)
	})
}

// orderLoggingScope is a fakeAutofillScope recording the tags it is asked
// for.
type orderLoggingScope struct {
	*fakeAutofillScope
	asked []string
}

func (s *orderLoggingScope) GetAutofillClient(id string) services.AutofillClient {
	s.asked = append(s.asked, id)
	return s.fakeAutofillScope.GetAutofillClient(id)
}

// updateEditingStateWithTag visits the tags in the platform's order.
func TestAutofillTagsFollowTheMessageOrder(t *testing.T) {
	textInputChannelTest(t, func(channel *fakeTextChannel) {
		scope := &orderLoggingScope{fakeAutofillScope: newFakeAutofillScope()}
		scope.SetSelf(scope)
		client := &fakeAutofillClient{fakeTextInputClient: *newFakeTextInputClient(tev("test"))}
		autofill := services.NewAutofillConfiguration(client.AutofillID(), []string{"hint"}, client.currentTextEditingValue)
		client.textInputConfiguration = services.NewTextInputConfiguration(services.TextInputConfigurationOptions{AutofillConfiguration: &autofill})
		scope.register(client)
		client.currentAutofillScope = scope
		scope.Attach(client, client.textInputConfiguration)
		value := jsonMessage.DecodeMessage(jsonMessage.EncodeMessage(tev("x").ToJSON()))
		channel.incoming(call("TextInputClient.updateEditingStateWithTag", []any{0,
			dart.NewLinkedMapOf("zeta", value, client.AutofillID(), value, "alpha", value, "mid", value)})).Await()
		expect(slices.Equal(scope.asked, []string{"zeta", client.AutofillID(), "alpha", "mid"}), "asked %v", scope.asked)
		expect(client.currentTextEditingValue.Text == "x", "%v", client.currentTextEditingValue)
	})
}
