package services_test

// Port of packages/flutter/test/services/text_input_test.dart (Flutter
// 3.50.1), and binding_test.dart's "initInstances sets a default method
// call handler for SystemChannels.textInput".
//
// Skipped / adapted:
//   - the hashCode tests (TextSelection, TextInputConfiguration,
//     TextInputType, TextInputStyle): Go values have no hashCode; equality
//     (Equal) is tested.
//   - "the platform input control receives isMultiline true on attach":
//     skipped in Dart too off the web (skip: !kIsWeb).
//   - "Invalid TextRange fails loudly": Dart's AssertionError text includes
//     the failed condition ('range.start >= 0 && range.start <= text.length');
//     a Go assertion has only the message, which is checked.
//   - The "TextInput error reporting" testWidgets don't pump widgets: they
//     run as async tests (pumpAndSettle is flushing the event loop).
//   - setSelectionRects' arguments are []any of []any (Dart's
//     List<List<num>>).

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

func sel(base, extent int) services.TextSelection { return services.NewTextSelection(base, extent) }
func collapsed(offset int) services.TextSelection {
	return services.NewTextSelectionCollapsed(offset)
}
func tev(text string, selection ...services.TextSelection) services.TextEditingValue {
	o := services.TextEditingValueOptions{Text: text}
	if len(selection) > 0 {
		o.Selection = &selection[0]
	}
	return services.NewTextEditingValue(o)
}

func TestTextSelectionInvalidSelectionIsASingleton(t *testing.T) {
	up := ui.TextAffinityUpstream
	a := services.NewTextSelection(-1, 0, services.TextSelectionOptions{IsDirectional: true})
	b := services.NewTextSelection(123, -1, services.TextSelectionOptions{Affinity: &up})
	if !a.Equal(b) {
		t.Fatal("invalid selections differ")
	}
}

func TestTextSelectionAffinityDoesNotAffectNonCollapsed(t *testing.T) {
	up := ui.TextAffinityUpstream
	if !sel(1, 2).Equal(services.NewTextSelection(1, 2, services.TextSelectionOptions{Affinity: &up})) {
		t.Fatal("differ")
	}
}

func TestTextEditingValueReplaced(t *testing.T) {
	const testText = "From a false proposition, anything follows."
	cases := []struct {
		name      string
		selection services.TextSelection
		r         ui.TextRange
		with      string
		want      services.TextEditingValue
	}{
		{"selection deletion", sel(5, 13), sel(5, 13).TextRange, "", tev("From proposition, anything follows.", collapsed(5))},
		{"reversed selection deletion", sel(13, 5), sel(13, 5).TextRange, "", tev("From proposition, anything follows.", collapsed(5))},
		{"insert", collapsed(5), collapsed(5).TextRange, "AA", tev("From AAa false proposition, anything follows.", collapsed(7))},
		{"replace before selection", sel(13, 5), ui.NewTextRange(4, 5), "AA", tev("FromAAa false proposition, anything follows.", sel(14, 6))},
		{"replace after selection", sel(13, 5), ui.NewTextRange(13, 14), "AA", tev("From a false AAroposition, anything follows.", sel(13, 5))},
		{"replace inside selection - start boundary", sel(13, 5), ui.NewTextRange(5, 6), "AA", tev("From AA false proposition, anything follows.", sel(14, 5))},
		{"replace inside selection - end boundary", sel(13, 5), ui.NewTextRange(12, 13), "AA", tev("From a falseAAproposition, anything follows.", sel(14, 5))},
		{"delete after selection", sel(13, 5), ui.NewTextRange(13, 14), "", tev("From a false roposition, anything follows.", sel(13, 5))},
		{"delete inside selection - start boundary", sel(13, 5), ui.NewTextRange(5, 6), "", tev("From  false proposition, anything follows.", sel(12, 5))},
		{"delete inside selection - end boundary", sel(13, 5), ui.NewTextRange(12, 13), "", tev("From a falseproposition, anything follows.", sel(12, 5))},
	}
	for _, c := range cases {
		got := tev(testText, c.selection).Replaced(c.r, c.with)
		if !got.Equal(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// textInputChannelTest is the "TextInput message channels" group's setUp
// and tearDown.
func textInputChannelTest(t *testing.T, body func(channel *fakeTextChannel)) {
	ensureInitialized()
	services.TextInputConnectionDebugResetID() // each Dart test file starts at 1 (its own isolate)
	channel := newFakeTextChannelNull()
	services.TextInputSetChannel(channel)
	defer func() {
		services.TextInputConnectionDebugResetID()
		services.TextInputSetChannel(services.SystemChannels.TextInput)
	}()
	runAsync(t, func() { body(channel) })
}

func mustValidate(c *fakeTextChannel, calls ...services.MethodCall) {
	if err := c.validateOutgoingMethodCalls(calls); err != nil {
		panic(err)
	}
}

func call(method string, arguments any) services.MethodCall {
	return services.MethodCall{Method: method, Arguments: arguments}
}

func TestTextInputClientHandlerRespondsToReattachWithSetClient(t *testing.T) {
	textInputChannelTest(t, func(channel *fakeTextChannel) {
		client := newFakeTextInputClient(tev("test1"))
		services.TextInputAttach(client, client.configuration())
		setClient := call("TextInput.setClient", []any{1, client.configuration().ToJSON()})
		mustValidate(channel, setClient)
		channel.incoming(call("TextInputClient.requestExistingInputState", nil)).Await()
		expect(len(channel.outgoingCalls) == 3, "%d calls", len(channel.outgoingCalls))
		mustValidate(channel, setClient, setClient, call("TextInput.setEditingState", client.currentTextEditingValue.ToJSON()))
	})
}

func TestTextInputClientHandlerRespondsToReattachWithSetClientEmptyValue(t *testing.T) {
	textInputChannelTest(t, func(channel *fakeTextChannel) {
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		services.TextInputAttach(client, client.configuration())
		setClient := call("TextInput.setClient", []any{1, client.configuration().ToJSON()})
		mustValidate(channel, setClient)
		channel.incoming(call("TextInputClient.requestExistingInputState", nil)).Await()
		expect(len(channel.outgoingCalls) == 3, "%d calls", len(channel.outgoingCalls))
		mustValidate(channel, setClient, setClient, call("TextInput.setEditingState", []dart.MapEntry[string, any]{
			{Key: "text", Value: ""}, {Key: "selectionBase", Value: -1}, {Key: "selectionExtent", Value: -1},
			{Key: "selectionAffinity", Value: "TextAffinity.downstream"}, {Key: "selectionIsDirectional", Value: false},
			{Key: "composingBase", Value: -1}, {Key: "composingExtent", Value: -1},
		}))
	})
}

// recordErrors replaces FlutterError.onError for the rest of the test.
func recordErrors(t *testing.T) *[]*foundation.FlutterErrorDetails {
	record := &[]*foundation.FlutterErrorDetails{}
	previous := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) { *record = append(*record, details) }
	t.Cleanup(func() { foundation.FlutterErrorOnError = previous })
	return record
}

// sendTextInput sends a JSON message on flutter/textinput as the platform
// does and returns the reply.
func sendTextInput(message []dart.MapEntry[string, any]) []byte {
	var reply []byte
	messenger().HandlePlatformMessage("flutter/textinput", jsonMessage.EncodeMessage(message), func(data []byte) {
		reply = data
	}).Await()
	return reply
}

func methodMessage(method string, args any) []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{{Key: "method", Value: method}, {Key: "args", Value: args}}
}

func TestInvalidTextRangeFailsLoudlyWhenConvertedToJSON(t *testing.T) {
	record := recordErrors(t)
	textInputChannelTest(t, func(channel *fakeTextChannel) {
		client := newFakeTextInputClient(tev("test3"))
		services.TextInputAttach(client, services.NewTextInputConfiguration())
		sendTextInput(methodMessage("TextInputClient.updateEditingState", []any{-1,
			[]dart.MapEntry[string, any]{{Key: "text", Value: "1"}, {Key: "selectionBase", Value: 2}, {Key: "selectionExtent", Value: 3}}}))
		expect(len(*record) == 1, "%d errors", len(*record))
		text := dart.ToString((*record)[0].Exception)
		expect(strings.Contains(text, "Range start 2 is out of text of length 1"), "%s", text)
	})
}

func TestFloatingCursorCoordinatesTypeCasting(t *testing.T) {
	record := recordErrors(t)
	textInputChannelTest(t, func(channel *fakeTextChannel) {
		client := newFakeTextInputClient(tev("test3"))
		services.TextInputAttach(client, services.NewTextInputConfiguration())
		sendTextInput(methodMessage("TextInputClient.updateFloatingCursor", []any{-1, "FloatingCursorDragState.update",
			[]dart.MapEntry[string, any]{{Key: "X", Value: 2}, {Key: "Y", Value: 3}}}))
		expect(len(*record) == 0, "errors %v", *record)
		// Own: the point reached the client.
		p := client.latestFloatingCursorPoint
		expect(p != nil && p.State == services.FloatingCursorDragStateUpdate && *p.Offset == ui.NewOffset(2, 3), "point %v", p)
	})
}

func fakeConfiguration() services.TextInputConfiguration {
	viewID, label := 1, "label1"
	dashes, quotes := services.SmartDashesTypeEnabled, services.SmartQuotesTypeEnabled
	return services.NewTextInputConfiguration(services.TextInputConfigurationOptions{
		ViewID: &viewID, ActionLabel: &label, SmartDashesType: &dashes, SmartQuotesType: &quotes,
		AllowedMimeTypes: []string{"text/plain", "application/pdf"},
	})
}

func compareConfigurations(t *testing.T, a, b services.TextInputConfiguration) {
	t.Helper()
	if !a.Equal(b) {
		t.Fatalf("%v != %v", a, b)
	}
	if *a.ViewID != *b.ViewID || !a.InputType.Equal(b.InputType) || a.InputAction != b.InputAction ||
		a.Autocorrect != b.Autocorrect || a.EnableSuggestions != b.EnableSuggestions || a.ObscureText != b.ObscureText ||
		a.ReadOnly != b.ReadOnly || a.SmartDashesType != b.SmartDashesType || a.SmartQuotesType != b.SmartQuotesType ||
		a.EnableInteractiveSelection != b.EnableInteractiveSelection || *a.ActionLabel != *b.ActionLabel ||
		a.KeyboardAppearance != b.KeyboardAppearance || a.TextCapitalization != b.TextCapitalization ||
		!a.AutofillConfiguration.Equal(b.AutofillConfiguration) ||
		a.EnableIMEPersonalizedLearning != b.EnableIMEPersonalizedLearning ||
		!reflect.DeepEqual(a.AllowedMimeTypes, b.AllowedMimeTypes) || a.EnableDeltaModel != b.EnableDeltaModel ||
		a.EnableInlinePrediction != b.EnableInlinePrediction {
		t.Fatal("fields differ")
	}
}

func TestTextInputConfigurationEqualityOperator(t *testing.T) {
	c := fakeConfiguration()
	compareConfigurations(t, c, c.CopyWith())
	// Own: a field change breaks equality; lists compare element-wise
	// (allowedMimeTypes) or by identity (hintLocales).
	other := c.CopyWith(services.TextInputConfigurationCopyWithOptions{ReadOnly: ptr(true)})
	if c.Equal(other) {
		t.Fatal("readOnly ignored")
	}
	if !c.Equal(c.CopyWith(services.TextInputConfigurationCopyWithOptions{AllowedMimeTypes: []string{"text/plain", "application/pdf"}})) {
		t.Fatal("allowedMimeTypes compared by identity")
	}
	if c.Equal(c.CopyWith(services.TextInputConfigurationCopyWithOptions{HintLocales: []ui.Locale{}})) {
		t.Fatal("hintLocales compared by value")
	}
	if !services.NewTextInputConfiguration().Equal(services.NewTextInputConfiguration()) {
		t.Fatal("default configurations differ")
	}
}

func TestTextInputConfigurationCopyWith(t *testing.T) {
	c := fakeConfiguration()
	compareConfigurations(t, c, c.CopyWith())
	// Own: every option replaces its field.
	inputType, action, brightness, capitalization := services.TextInputTypeMultiline, services.TextInputActionGo, ui.BrightnessDark, services.TextCapitalizationWords
	viewID, label := 7, "x"
	got := c.CopyWith(services.TextInputConfigurationCopyWithOptions{
		ViewID: &viewID, InputType: &inputType, ReadOnly: ptr(true), ObscureText: ptr(true), Autocorrect: ptr(false),
		SmartDashesType: ptr(services.SmartDashesTypeDisabled), SmartQuotesType: ptr(services.SmartQuotesTypeDisabled),
		EnableSuggestions: ptr(false), EnableInteractiveSelection: ptr(false), ActionLabel: &label,
		InputAction: &action, KeyboardAppearance: &brightness, TextCapitalization: &capitalization,
		EnableIMEPersonalizedLearning: ptr(false), AllowedMimeTypes: []string{"a"}, EnableDeltaModel: ptr(true),
		EnableInlinePrediction: ptr(true),
	})
	want := "TextInputConfiguration(viewId: 7, inputType: TextInputType(name: TextInputType.multiline, signed: null, decimal: null, password: null), readOnly: true, obscureText: true, autocorrect: false, smartDashesType: SmartDashesType.disabled, smartQuotesType: SmartQuotesType.disabled, enableSuggestions: false, enableInteractiveSelection: false, actionLabel: x, inputAction: TextInputAction.go, keyboardAppearance: Brightness.dark, textCapitalization: TextCapitalization.words, autofillConfiguration: AutofillConfiguration(enabled: false, uniqueIdentifier: , autofillHints: [], currentEditingValue: TextEditingValue(text: ┤├, selection: TextSelection.invalid, composing: TextRange(start: -1, end: -1))), enableIMEPersonalizedLearning: false, allowedMimeTypes: [a], enableDeltaModel: true, hintLocales: [], enableInlinePrediction: true)"
	if got.String() != want {
		t.Fatalf("got  %s\nwant %s", got.String(), want)
	}
}

func ptr[T any](v T) *T { return &v }

func TestTextInputConfigurationSetsExpectedDefaults(t *testing.T) {
	c := services.NewTextInputConfiguration()
	if !c.InputType.Equal(services.TextInputTypeText) || c.ReadOnly || c.ObscureText || c.EnableDeltaModel ||
		!c.Autocorrect || c.ActionLabel != nil || c.TextCapitalization != services.TextCapitalizationNone ||
		c.KeyboardAppearance != ui.BrightnessLight || c.EnableInlinePrediction != nil {
		t.Fatalf("%v", c)
	}
	// Own: the rest of the defaults.
	if !c.EnableSuggestions || !c.EnableInteractiveSelection || c.InputAction != services.TextInputActionDone ||
		!c.EnableIMEPersonalizedLearning || c.SmartDashesType != services.SmartDashesTypeEnabled ||
		c.SmartQuotesType != services.SmartQuotesTypeEnabled || c.AutofillConfiguration.Enabled ||
		c.HintLocales == nil || len(c.HintLocales) != 0 || c.AllowedMimeTypes == nil {
		t.Fatalf("%v", c)
	}
	obscure := services.NewTextInputConfiguration(services.TextInputConfigurationOptions{ObscureText: true})
	if obscure.SmartDashesType != services.SmartDashesTypeDisabled || obscure.SmartQuotesType != services.SmartQuotesTypeDisabled {
		t.Fatal("obscured text keeps smart dashes/quotes")
	}
}

func jsonValue(t *testing.T, m []dart.MapEntry[string, any], key string) any {
	t.Helper()
	decoded := jsonMessage.DecodeMessage(jsonMessage.EncodeMessage(m)).(*dart.LinkedMap)
	return decoded.Lookup(key)
}

func TestTextInputConfigurationSerializesToJSON(t *testing.T) {
	label := "xyzzy"
	c := services.NewTextInputConfiguration(services.TextInputConfigurationOptions{
		ReadOnly: true, ObscureText: true, Autocorrect: ptr(false), ActionLabel: &label,
	})
	json := c.ToJSON()
	if got := jsonValue(t, json, "inputType"); !decodedEqual(got, map[string]any{"name": "TextInputType.text", "signed": nil, "decimal": nil, "password": nil}) {
		t.Fatalf("inputType %v", got)
	}
	if jsonValue(t, json, "readOnly") != true || jsonValue(t, json, "obscureText") != true ||
		jsonValue(t, json, "autocorrect") != false || jsonValue(t, json, "actionLabel") != "xyzzy" ||
		jsonValue(t, json, "enableInlinePrediction") != nil {
		t.Fatal(string(jsonMessage.EncodeMessage(json)))
	}
	// enableInlinePrediction serializes to JSON.
	for _, v := range []*bool{ptr(true), ptr(false), nil} {
		got := jsonValue(t, services.NewTextInputConfiguration(services.TextInputConfigurationOptions{EnableInlinePrediction: v}).ToJSON(), "enableInlinePrediction")
		if v == nil && got != nil || v != nil && got != *v {
			t.Fatalf("enableInlinePrediction %v: %v", v, got)
		}
	}
	// number and numeric password serialize to JSON.
	number := services.NewTextInputTypeNumberWithOptions(services.TextInputTypeNumberWithOptionsOptions{Decimal: true})
	c = services.NewTextInputConfiguration(services.TextInputConfigurationOptions{InputType: &number, ObscureText: true, Autocorrect: ptr(false), ActionLabel: &label})
	json = c.ToJSON()
	if got := jsonValue(t, json, "inputType"); !decodedEqual(got, map[string]any{"name": "TextInputType.number", "signed": false, "decimal": true, "password": false}) {
		t.Fatalf("inputType %v", got)
	}
	if jsonValue(t, json, "readOnly") != false || jsonValue(t, json, "obscureText") != true ||
		jsonValue(t, json, "autocorrect") != false || jsonValue(t, json, "actionLabel") != "xyzzy" {
		t.Fatal(string(jsonMessage.EncodeMessage(json)))
	}
	password := services.NewTextInputTypeNumberWithOptions(services.TextInputTypeNumberWithOptionsOptions{Password: true})
	json = services.NewTextInputConfiguration(services.TextInputConfigurationOptions{InputType: &password}).ToJSON()
	if got := jsonValue(t, json, "inputType"); !decodedEqual(got, map[string]any{"name": "TextInputType.number", "signed": false, "decimal": false, "password": true}) {
		t.Fatalf("inputType %v", got)
	}
	// Own: the whole default encoding, byte for byte.
	want := `{"viewId":null,"inputType":{"name":"TextInputType.text","signed":null,"decimal":null,"password":null},"readOnly":false,"obscureText":false,"autocorrect":true,"smartDashesType":"1","smartQuotesType":"1","enableSuggestions":true,"enableInteractiveSelection":true,"actionLabel":null,"inputAction":"TextInputAction.done","textCapitalization":"TextCapitalization.none","keyboardAppearance":"Brightness.light","enableIMEPersonalizedLearning":true,"contentCommitMimeTypes":[],"enableDeltaModel":false,"hintLocales":[],"enableInlinePrediction":null}`
	if got := string(jsonMessage.EncodeMessage(services.NewTextInputConfiguration().ToJSON())); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestTextInputTypeBasicStructure(t *testing.T) {
	text, number, number2 := services.TextInputTypeText, services.TextInputTypeNumber, services.TextInputTypeNumber
	opts := func(s, d, p bool) services.TextInputType {
		return services.NewTextInputTypeNumberWithOptions(services.TextInputTypeNumberWithOptionsOptions{Signed: s, Decimal: d, Password: p})
	}
	signed, signed2, decimal, signedDecimal := opts(true, false, false), opts(true, false, false), opts(false, true, false), opts(true, true, false)
	password, password2, signedDecimalPassword := opts(false, false, true), opts(false, false, true), opts(true, true, true)
	strs := map[string]services.TextInputType{
		"TextInputType(name: TextInputType.text, signed: null, decimal: null, password: null)":            text,
		"TextInputType(name: TextInputType.number, signed: false, decimal: false, password: false)":       number,
		"TextInputType(name: TextInputType.number, signed: true, decimal: false, password: false)":        signed,
		"TextInputType(name: TextInputType.number, signed: false, decimal: true, password: false)":        decimal,
		"TextInputType(name: TextInputType.number, signed: true, decimal: true, password: false)":         signedDecimal,
		"TextInputType(name: TextInputType.number, signed: false, decimal: false, password: true)":        password,
		"TextInputType(name: TextInputType.number, signed: true, decimal: true, password: true)":          signedDecimalPassword,
		"TextInputType(name: TextInputType.multiline, signed: null, decimal: null, password: null)":       services.TextInputTypeMultiline,
		"TextInputType(name: TextInputType.phone, signed: null, decimal: null, password: null)":           services.TextInputTypePhone,
		"TextInputType(name: TextInputType.datetime, signed: null, decimal: null, password: null)":        services.TextInputTypeDatetime,
		"TextInputType(name: TextInputType.emailAddress, signed: null, decimal: null, password: null)":    services.TextInputTypeEmailAddress,
		"TextInputType(name: TextInputType.url, signed: null, decimal: null, password: null)":             services.TextInputTypeURL,
		"TextInputType(name: TextInputType.visiblePassword, signed: null, decimal: null, password: null)": services.TextInputTypeVisiblePassword,
		"TextInputType(name: TextInputType.name, signed: null, decimal: null, password: null)":            services.TextInputTypeName,
		"TextInputType(name: TextInputType.address, signed: null, decimal: null, password: null)":         services.TextInputTypeStreetAddress,
		"TextInputType(name: TextInputType.none, signed: null, decimal: null, password: null)":            services.TextInputTypeNone,
		"TextInputType(name: TextInputType.webSearch, signed: null, decimal: null, password: null)":       services.TextInputTypeWebSearch,
		"TextInputType(name: TextInputType.twitter, signed: null, decimal: null, password: null)":         services.TextInputTypeTwitter,
	}
	for want, ty := range strs {
		if ty.String() != want {
			t.Errorf("got %s, want %s", ty.String(), want)
		}
	}
	eq := []struct {
		a, b services.TextInputType
		want bool
	}{
		{text, number, false}, {number, number2, true}, {number, signed, false}, {number, password, false},
		{signed, signed2, true}, {signed, decimal, false}, {signed, signedDecimal, false}, {decimal, signedDecimal, false},
		{password, password2, true}, {password, signedDecimalPassword, false}, {signedDecimal, signedDecimalPassword, false},
	}
	for i, c := range eq {
		if c.a.Equal(c.b) != c.want {
			t.Errorf("case %d: %v == %v is %v", i, c.a, c.b, !c.want)
		}
	}
	for i, ty := range services.TextInputTypeValues {
		if ty.Index != i {
			t.Errorf("%v.index = %d", ty, ty.Index)
		}
	}
	if got := services.TextEditingValueEmpty.String(); got != "TextEditingValue(text: ┤├, selection: "+collapsed(-1).String()+", composing: "+ui.TextRangeEmpty.String()+")" {
		t.Error(got)
	}
	if got := tev("Sample Text").String(); got != "TextEditingValue(text: ┤Sample Text├, selection: "+collapsed(-1).String()+", composing: "+ui.TextRangeEmpty.String()+")" {
		t.Error(got)
	}
}

func TestTextInputClientOnConnectionClosedIsCalled(t *testing.T) {
	textInputChannelTest(t, func(*fakeTextChannel) {
		client := newFakeTextInputClient(tev("test3"))
		services.TextInputAttach(client, services.NewTextInputConfiguration())
		expect(client.latestMethodCall == "", "%s", client.latestMethodCall)
		sendTextInput([]dart.MapEntry[string, any]{{Key: "args", Value: []any{1}}, {Key: "method", Value: "TextInputClient.onConnectionClosed"}})
		expect(client.latestMethodCall == "connectionClosed", "%s", client.latestMethodCall)
	})
}

func TestTextInputClientOnFocusReceivedIsCalled(t *testing.T) {
	textInputChannelTest(t, func(*fakeTextChannel) {
		client := newFakeTextInputClient(tev("test3"))
		connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
		expect(connection.Attached(), "not attached")
		expect(client.latestMethodCall == "", "%s", client.latestMethodCall)
		connection.ConnectionClosedReceived()
		expect(!connection.Attached(), "attached")
		expect(client.latestMethodCall == "", "%s", client.latestMethodCall)
		reply := sendTextInput(methodMessage("TextInputClient.onFocusReceived", []any{1}))
		expect(client.latestMethodCall == "onFocusReceived", "%s", client.latestMethodCall)
		// Own: the client's answer is the reply.
		expect(string(reply) == "[true]", "reply %s", reply)
	})
}

func TestTextInputClientOnFocusReceivedIsCalledEvenIfAlreadyConnected(t *testing.T) {
	textInputChannelTest(t, func(*fakeTextChannel) {
		client := newFakeTextInputClient(tev("test3"))
		connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
		expect(connection.Attached(), "not attached")
		sendTextInput(methodMessage("TextInputClient.onFocusReceived", []any{1}))
		expect(connection.Attached(), "not attached")
		expect(client.latestMethodCall == "onFocusReceived", "%s", client.latestMethodCall)
		// Own: another client id answers false.
		client.latestMethodCall = ""
		reply := sendTextInput(methodMessage("TextInputClient.onFocusReceived", []any{2}))
		expect(client.latestMethodCall == "" && string(reply) == "[false]", "%s %s", client.latestMethodCall, reply)
	})
}

func clientCallTest(t *testing.T, args []any, method, want string, check ...func(client *fakeTextInputClient)) {
	textInputChannelTest(t, func(*fakeTextChannel) {
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		services.TextInputAttach(client, services.NewTextInputConfiguration())
		expect(client.latestMethodCall == "", "%s", client.latestMethodCall)
		sendTextInput([]dart.MapEntry[string, any]{{Key: "args", Value: args}, {Key: "method", Value: method}})
		expect(client.latestMethodCall == want, "%s", client.latestMethodCall)
		for _, c := range check {
			c(client)
		}
	})
}

func jsonDecode(s string) any { return jsonMessage.DecodeMessage([]byte(s)) }

func TestTextInputClientInsertContentIsCalled(t *testing.T) {
	clientCallTest(t, []any{1, "TextInputAction.commitContent",
		jsonDecode(`{"mimeType": "image/gif", "data": [0,1,0,1,0,1,0,0,0], "uri": "content://com.google.android.inputmethod.latin.fileprovider/test.gif"}`)},
		"TextInputClient.performAction", "commitContent")
}

func TestTextInputClientPerformSelectorsIsCalled(t *testing.T) {
	clientCallTest(t, []any{1, []any{"selector1", "selector2"}}, "TextInputClient.performSelectors", "performSelector",
		func(c *fakeTextInputClient) {
			expect(reflect.DeepEqual(c.performedSelectors, []string{"selector1", "selector2"}), "%v", c.performedSelectors)
		})
}

func TestTextInputClientPerformPrivateCommandIsCalled(t *testing.T) {
	for _, data := range []string{
		`{"action": "actionCommand", "data": {"input_context" : "abcdefg"}}`,
		`{"action": "actionCommand", "data": {"input_context" : 0.5}}`,
		`{"action": "actionCommand", "data": {"input_context" : ["abc", "efg"]}}`,
		`{"action": "actionCommand", "data": {"input_context" : "abc"}}`,
		`{"action": "actionCommand", "data": {"input_context" : [0.5, 0.8]}}`,
	} {
		clientCallTest(t, []any{1, jsonDecode(data)}, "TextInputClient.performPrivateCommand", "performPrivateCommand")
	}
}

func TestTextInputClientPerformPrivateCommandWithNoData(t *testing.T) {
	clientCallTest(t, []any{1, jsonDecode(`{"action": "actionCommand"}`)}, "TextInputClient.performPrivateCommand",
		"performPrivateCommand", func(c *fakeTextInputClient) {
			expect(c.latestPrivateCommandData != nil && c.latestPrivateCommandData.IsEmpty(), "%v", c.latestPrivateCommandData)
		})
}

func TestTextInputClientShowAutocorrectionPromptRectIsCalled(t *testing.T) {
	clientCallTest(t, []any{1, 0, 1}, "TextInputClient.showAutocorrectionPromptRect", "showAutocorrectionPromptRect")
}

func TestTextInputClientShowToolbarIsCalled(t *testing.T) {
	clientCallTest(t, []any{1, 0, 1}, "TextInputClient.showToolbar", "showToolbar")
}

func TestScribbleInteractionBeganAndFinished(t *testing.T) {
	ensureInitialized()
	defer services.TextInputConnectionDebugResetID()
	runAsync(t, func() {
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
		expect(!connection.ScribbleInProgress(), "in progress")
		sendTextInput([]dart.MapEntry[string, any]{{Key: "args", Value: []any{1, 0, 1}}, {Key: "method", Value: "TextInputClient.scribbleInteractionBegan"}})
		expect(connection.ScribbleInProgress(), "not in progress")
		sendTextInput([]dart.MapEntry[string, any]{{Key: "args", Value: []any{1, 0, 1}}, {Key: "method", Value: "TextInputClient.scribbleInteractionFinished"}})
		expect(!connection.ScribbleInProgress(), "in progress")
	})
}

func TestScribbleFocusElement(t *testing.T) {
	ensureInitialized()
	defer services.TextInputConnectionDebugResetID()
	runAsync(t, func() {
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		services.TextInputAttach(client, services.NewTextInputConfiguration())
		target := &fakeScribbleElement{elementIdentifier: "target"}
		services.TextInputRegisterScribbleElement(target.elementIdentifier, target)
		other := &fakeScribbleElement{elementIdentifier: "other"}
		services.TextInputRegisterScribbleElement(other.elementIdentifier, other)
		expect(target.latestMethodCall == "" && other.latestMethodCall == "", "calls")
		sendTextInput([]dart.MapEntry[string, any]{{Key: "args", Value: []any{target.elementIdentifier, 0.0, 0.0}}, {Key: "method", Value: "TextInputClient.focusElement"}})
		services.TextInputUnregisterScribbleElement(target.elementIdentifier)
		services.TextInputUnregisterScribbleElement(other.elementIdentifier)
		expect(target.latestMethodCall == "onScribbleFocus" && other.latestMethodCall == "", "%s %s", target.latestMethodCall, other.latestMethodCall)
		expect(len(services.TextInputScribbleClients()) == 0, "clients left")
	})
}

func TestScribbleRequestElementsInRect(t *testing.T) {
	ensureInitialized()
	defer services.TextInputConnectionDebugResetID()
	runAsync(t, func() {
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		services.TextInputAttach(client, services.NewTextInputConfiguration())
		targets := []*fakeScribbleElement{
			{elementIdentifier: "target1", bounds: ui.NewRectFromLTWH(0, 0, 100, 100)},
			{elementIdentifier: "target2", bounds: ui.NewRectFromLTWH(0, 100, 100, 100)},
		}
		others := []*fakeScribbleElement{
			{elementIdentifier: "other1", bounds: ui.NewRectFromLTWH(100, 0, 100, 100)},
			{elementIdentifier: "other2", bounds: ui.NewRectFromLTWH(100, 100, 100, 100)},
		}
		all := append(append([]*fakeScribbleElement{}, targets...), others...)
		for _, e := range all {
			services.TextInputRegisterScribbleElement(e.elementIdentifier, e)
		}
		reply := sendTextInput([]dart.MapEntry[string, any]{{Key: "args", Value: []any{0.0, 50.0, 50.0, 100.0}}, {Key: "method", Value: "TextInputClient.requestElementsInRect"}})
		for _, e := range all {
			services.TextInputUnregisterScribbleElement(e.elementIdentifier)
		}
		responses := jsonMessage.DecodeMessage(reply).([]any)
		first := responses[0].([]any)
		expect(len(first) == 2, "%v", first)
		expect(reflect.DeepEqual(first[0], []any{"target1", 0.0, 0.0, 100.0, 100.0}), "%v", first[0])
		expect(reflect.DeepEqual(first[1], []any{"target2", 0.0, 100.0, 100.0, 100.0}), "%v", first[1])
	})
}

func TestTextEditingValueIsComposingRangeValid(t *testing.T) {
	if services.TextEditingValueEmpty.IsComposingRangeValid() {
		t.Error("empty")
	}
	for _, c := range []struct {
		start, end int
		want       bool
	}{{1, 0, false}, {1, 5, false}, {-1, 4, false}, {1, 4, true}} {
		r := ui.TextRange{Start: c.start, End: c.end}
		v := services.NewTextEditingValue(services.TextEditingValueOptions{Text: "test", Composing: &r})
		if v.IsComposingRangeValid() != c.want {
			t.Errorf("%v: %v", r, !c.want)
		}
	}
}

// textInputControlTest is the "TextInputControl" group's setUp/tearDown.
func textInputControlTest(t *testing.T, body func(channel *fakeTextChannel)) {
	ensureInitialized()
	services.TextInputConnectionDebugResetID()
	channel := newFakeTextChannelNull()
	services.TextInputSetChannel(channel)
	defer func() {
		services.TextInputRestorePlatformInputControl()
		services.TextInputConnectionDebugResetID()
		services.TextInputSetChannel(services.SystemChannels.TextInput)
	}()
	runAsync(t, func() { body(channel) })
}

func TestTextInputControlGetsAttachedAndDetached(t *testing.T) {
	textInputControlTest(t, func(*fakeTextChannel) {
		control := &fakeTextInputControl{}
		services.TextInputSetInputControl(control)
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
		expect(reflect.DeepEqual(control.methodCalls, []string{"attach"}), "%v", control.methodCalls)
		connection.Close()
		expect(reflect.DeepEqual(control.methodCalls, []string{"attach", "detach"}), "%v", control.methodCalls)
	})
}

func TestTextInputControlReceivesTextInputStateChanges(t *testing.T) {
	textInputControlTest(t, func(*fakeTextChannel) {
		control := &fakeTextInputControl{}
		services.TextInputSetInputControl(control)
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
		control.methodCalls = nil
		connection.UpdateConfig(services.NewTextInputConfiguration())
		expect(reflect.DeepEqual(control.methodCalls, []string{"updateConfig"}), "%v", control.methodCalls)
		connection.SetEditingState(services.TextEditingValueEmpty)
		expect(reflect.DeepEqual(control.methodCalls, []string{"updateConfig", "setEditingState"}), "%v", control.methodCalls)
		connection.Close()
		expect(reflect.DeepEqual(control.methodCalls, []string{"updateConfig", "setEditingState", "detach"}), "%v", control.methodCalls)
	})
}

func TestTextInputControlDoesNotInterfereWithPlatformTextInput(t *testing.T) {
	textInputControlTest(t, func(channel *fakeTextChannel) {
		control := &fakeTextInputControl{}
		services.TextInputSetInputControl(control)
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		services.TextInputAttach(client, services.NewTextInputConfiguration())
		channel.outgoingCalls = nil
		channel.incoming(call("TextInputClient.updateEditingState", []any{1, jsonMessage.DecodeMessage(jsonMessage.EncodeMessage(services.TextEditingValueEmpty.ToJSON()))}))
		expect(client.latestMethodCall == "updateEditingValue", "%s", client.latestMethodCall)
		expect(reflect.DeepEqual(control.methodCalls, []string{"attach", "setEditingState"}), "%v", control.methodCalls)
		expect(len(channel.outgoingCalls) == 0, "%v", channel.outgoingCalls)
	})
}

func TestTextInputControlBothInputControlsReceiveRequests(t *testing.T) {
	textInputControlTest(t, func(channel *fakeTextChannel) {
		control := &fakeTextInputControl{}
		services.TextInputSetInputControl(control)
		textConfig := services.NewTextInputConfiguration()
		numberConfig := services.NewTextInputConfiguration(services.TextInputConfigurationOptions{InputType: &services.TextInputTypeNumber})
		multilineConfig := services.NewTextInputConfiguration(services.TextInputConfigurationOptions{InputType: &services.TextInputTypeMultiline})
		noneConfig := services.NewTextInputConfiguration(services.TextInputConfigurationOptions{InputType: &services.TextInputTypeNone})
		noneJSON := noneConfig.ToJSON()

		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		connection := services.TextInputAttach(client, textConfig)
		expected := []string{"attach"}
		check := func(n int, lastMethod string) {
			expect(reflect.DeepEqual(control.methodCalls, expected), "%v", control.methodCalls)
			expect(len(channel.outgoingCalls) == n, "%d calls", len(channel.outgoingCalls))
			if lastMethod != "" {
				expect(channel.outgoingCalls[n-1].Method == lastMethod, "%s", channel.outgoingCalls[n-1].Method)
			}
		}
		check(1, "")
		expect(control.inputType.Equal(services.TextInputTypeText), "%v", control.inputType)
		mustValidate(channel, call("TextInput.setClient", []any{1, noneJSON}))

		connection.Show()
		expected = append(expected, "show")
		check(2, "TextInput.show")

		connection.UpdateConfig(numberConfig)
		expected = append(expected, "updateConfig")
		check(3, "")
		expect(control.inputType.Equal(services.TextInputTypeNumber), "%v", control.inputType)
		mustValidate(channel, call("TextInput.setClient", []any{1, noneJSON}), call("TextInput.show", nil),
			call("TextInput.updateConfig", noneJSON))

		connection.UpdateConfig(multilineConfig)
		expected = append(expected, "updateConfig")
		check(4, "")
		expect(control.inputType.Equal(services.TextInputTypeMultiline), "%v", control.inputType)
		mustValidate(channel, call("TextInput.setClient", []any{1, noneJSON}), call("TextInput.show", nil),
			call("TextInput.updateConfig", noneJSON), call("TextInput.updateConfig", noneJSON))

		connection.SetComposingRect(ui.RectZero)
		expected = append(expected, "setComposingRect")
		check(5, "TextInput.setMarkedTextRect")

		connection.SetCaretRect(ui.RectZero)
		expected = append(expected, "setCaretRect")
		check(6, "TextInput.setCaretRect")

		connection.SetEditableSizeAndTransform(ui.SizeZero, vector_math.NewMatrix4Identity())
		expected = append(expected, "setEditableSizeAndTransform")
		check(7, "TextInput.setEditableSizeAndTransform")

		rtl := ui.TextDirectionRtl
		connection.SetSelectionRects([]services.SelectionRect{
			services.NewSelectionRect(1, ui.NewRectFromLTWH(2, 3, 4, 5), services.SelectionRectOptions{Direction: &rtl}),
		})
		expected = append(expected, "setSelectionRects")
		check(8, "TextInput.setSelectionRects")
		sentList := channel.outgoingCalls[7].Arguments.([]any)
		expect(len(sentList) == 1, "%v", sentList)
		expect(reflect.DeepEqual(sentList[0], []any{2.0, 3.0, 4.0, 5.0, 1, int(ui.TextDirectionRtl)}), "%v", sentList[0])

		connection.UpdateStyle(services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignLeft))
		expected = append(expected, "updateStyle")
		check(9, "TextInput.setStyle")

		connection.Close()
		expected = append(expected, "detach")
		check(10, "TextInput.clearClient")

		expected = append(expected, "hide")
		async.FutureMicrotask(func() any { return nil }).Await() // binding.runAsync(() async {})
		check(11, "TextInput.hide")
	})
}

func TestTextInputControlNotifiesChangesToTheAttachedClient(t *testing.T) {
	textInputControlTest(t, func(*fakeTextChannel) {
		control := &fakeTextInputControl{}
		services.TextInputSetInputControl(control)
		client := newFakeTextInputClient(services.TextEditingValueEmpty)
		connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
		services.TextInputSetInputControl(nil)
		expect(client.latestMethodCall == "didChangeInputControl", "%s", client.latestMethodCall)
		connection.Show()
		expect(client.latestMethodCall == "didChangeInputControl", "%s", client.latestMethodCall)
	})
}

// nodeValue is a property node's value, nullable values dereferenced.
func nodeValue(n foundation.DiagnosticsNode) any {
	switch v := n.Value().(type) {
	case *bool:
		return *v
	case *string:
		return *v
	}
	return n.Value()
}

func propertyNodes(d foundation.Diagnosticable) []foundation.DiagnosticsNode {
	return d.ToDiagnosticsNode().GetProperties()
}

func TestSystemContextMenuControllerDebugFillProperties(t *testing.T) {
	ensureInitialized()
	controller := services.NewSystemContextMenuController(services.SystemContextMenuControllerOptions{OnSystemHide: func() {}})
	defer services.ServicesBindingSetSystemContextMenuClient(nil)
	nodes := propertyNodes(controller)
	if len(nodes) != 4 {
		t.Fatalf("%d nodes", len(nodes))
	}
	want := []struct {
		name  string
		value any
	}{{"isVisible", false}, {"onSystemHide", true}, {"_hiddenBySystem", false}, {"_isDisposed", false}}
	for i, w := range want {
		if nodes[i].Name() != w.name || nodeValue(nodes[i]) != w.value {
			t.Errorf("%d: %s %v", i, nodes[i].Name(), nodeValue(nodes[i]))
		}
	}
}

func TestIOSSystemContextMenuItemDataDebugFillProperties(t *testing.T) {
	const title = "my title"
	for _, item := range []foundation.Diagnosticable{
		services.NewIOSSystemContextMenuItemDataLookUp(title),
		services.NewIOSSystemContextMenuItemDataSearchWeb(title),
		services.NewIOSSystemContextMenuItemDataShare(title),
	} {
		nodes := propertyNodes(item)
		if len(nodes) != 1 || nodes[0].Name() != "title" || nodeValue(nodes[0]) != title {
			t.Errorf("%T: %v", item, nodes)
		}
	}
}

func textInputStyles() (*services.TextInputStyle, *services.TextInputStyle, *services.TextInputStyle) {
	full := func(family string) *services.TextInputStyle {
		return services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignCenter, services.TextInputStyleOptions{
			FontFamily: &family, FontSize: ptr(16.0), FontWeight: &ui.FontWeightBold,
			LetterSpacing: ptr(1.2), WordSpacing: ptr(2.0), LineHeight: ptr(24.0),
		})
	}
	other := "Other"
	style3 := services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignCenter, services.TextInputStyleOptions{
		FontFamily: &other, FontSize: ptr(16.0), FontWeight: &ui.FontWeightBold,
	})
	return full("Roboto"), full("Roboto"), style3
}

func TestTextInputStyleEqualityOperator(t *testing.T) {
	style1, style2, style3 := textInputStyles()
	if !style1.Equal(style2) || style1.Equal(style3) {
		t.Fatal("equality")
	}
	// Own: each field matters.
	style4, _, _ := textInputStyles()
	style4.LineHeight = ptr(25.0)
	if style1.Equal(style4) {
		t.Fatal("lineHeight ignored")
	}
	style4.LineHeight, style4.TextAlign = style1.LineHeight, ui.TextAlignLeft
	if style1.Equal(style4) {
		t.Fatal("textAlign ignored")
	}
}

func TestTextInputStyleToJSON(t *testing.T) {
	style1, _, _ := textInputStyles()
	got := string(jsonMessage.EncodeMessage(style1.ToJSON()))
	want := `{"fontFamily":"Roboto","fontSize":16.0,"fontWeightIndex":6,"textAlignIndex":2,"textDirectionIndex":1,"letterSpacing":1.2,"wordSpacing":2.0,"lineHeight":24.0}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	got = string(jsonMessage.EncodeMessage(services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignLeft).ToJSON()))
	want = `{"fontFamily":null,"fontSize":null,"fontWeightIndex":null,"textAlignIndex":0,"textDirectionIndex":1,"letterSpacing":null,"wordSpacing":null,"lineHeight":null}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

// errorReportingTest is a "TextInput error reporting" test: the platform
// fails failingMethod, act triggers it, and the error is reported with
// context.
func errorReportingTest(t *testing.T, failingMethod, message, context string, act func(connection *services.TextInputConnection)) {
	ensureInitialized()
	services.TextInputRestorePlatformInputControl()
	defer services.TextInputConnectionDebugResetID()
	record := recordErrors(t)
	messenger().SetMockMethodCallHandler(services.SystemChannels.TextInput, func(methodCall services.MethodCall) async.Future[any] {
		return async.Async(func() any {
			if methodCall.Method == failingMethod {
				panic(message)
			}
			return nil
		})
	})
	defer messenger().SetMockMethodCallHandler(services.SystemChannels.TextInput, nil)
	runAsync(t, func() {
		client := newFakeTextInputClient(tev("test"))
		connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
		act(connection)
	})
	if len(*record) != 1 {
		t.Fatalf("%d errors: %v", len(*record), *record)
	}
	var platform *services.PlatformException
	err, _ := (*record)[0].Exception.(error)
	if !errors.As(err, &platform) || platform.Message == nil || *platform.Message != message {
		t.Fatalf("exception %v", (*record)[0].Exception)
	}
	if got := (*record)[0].Context.String(); !strings.Contains(got, context) {
		t.Fatalf("context %q", got)
	}
}

func TestTextInputErrorReporting(t *testing.T) {
	idle := func() { async.FutureMicrotask(func() any { return nil }).Await() }
	t.Run("attach reports error when setClient fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.setClient", "Failed to set client", "while attaching the text input client", func(*services.TextInputConnection) {})
	})
	t.Run("detach reports error when clearClient fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.clearClient", "Failed to clear client", "while detaching the text input client", func(c *services.TextInputConnection) {
			idle()
			c.Close()
		})
	})
	t.Run("updateConfig reports error when updateConfig fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.updateConfig", "Failed to update config", "while updating text input configuration", func(c *services.TextInputConnection) {
			c.UpdateConfig(services.NewTextInputConfiguration())
		})
	})
	t.Run("setEditingState reports error when setEditingState fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.setEditingState", "Failed to set editing state", "while setting text input editing state", func(c *services.TextInputConnection) {
			c.SetEditingState(tev("new test"))
		})
	})
	t.Run("show reports error when show fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.show", "Failed to show", "while showing the text input client", func(c *services.TextInputConnection) { c.Show() })
	})
	t.Run("hide reports error when hide fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.hide", "Failed to hide", "while hiding the text input client", func(c *services.TextInputConnection) {
			idle()
			c.Close()
		})
	})
	t.Run("setEditableSizeAndTransform reports error when fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.setEditableSizeAndTransform", "Failed to set size and transform", "while setting text input size and transform", func(c *services.TextInputConnection) {
			c.SetEditableSizeAndTransform(ui.SizeZero, vector_math.NewMatrix4Identity())
		})
	})
	t.Run("setComposingRect reports error when setMarkedTextRect fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.setMarkedTextRect", "Failed to set composing rect", "while setting text input composing rect", func(c *services.TextInputConnection) {
			c.SetComposingRect(ui.RectZero)
		})
	})
	t.Run("setCaretRect reports error when setCaretRect fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.setCaretRect", "Failed to set caret rect", "while setting text input caret rect", func(c *services.TextInputConnection) {
			c.SetCaretRect(ui.RectZero)
		})
	})
	t.Run("setSelectionRects reports error when fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.setSelectionRects", "Failed to set selection rects", "while setting text input selection rects", func(c *services.TextInputConnection) {
			c.SetSelectionRects([]services.SelectionRect{services.NewSelectionRect(0, ui.RectZero)})
		})
	})
	t.Run("updateStyle reports error when setStyle fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.setStyle", "Failed to set style", "while updating text input style", func(c *services.TextInputConnection) {
			family := "Roboto"
			c.SetStyle(&family, ptr(12.0), &ui.FontWeightNormal, ui.TextDirectionLtr, ui.TextAlignLeft)
		})
	})
	t.Run("requestAutofill reports error when fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.requestAutofill", "Failed to request autofill", "while requesting autofill", func(c *services.TextInputConnection) {
			c.RequestAutofill()
		})
	})
	t.Run("finishAutofillContext reports error when fails", func(t *testing.T) {
		errorReportingTest(t, "TextInput.finishAutofillContext", "Failed to finish autofill context", "while finishing autofill context", func(*services.TextInputConnection) {
			services.TextInputFinishAutofillContext()
		})
	})
}

// binding_test.dart: initInstances sets a default method call handler for
// SystemChannels.textInput.
func TestInitInstancesSetsDefaultTextInputHandler(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		// As in binding_test.dart's isolate, no text input connection is open.
		services.TextInputAttach(newFakeTextInputClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration()).ConnectionClosedReceived()
		message := jsonMessage.EncodeMessage([]dart.MapEntry[string, any]{{Key: "method", Value: "TextInput.requestElementsInRect"}, {Key: "args", Value: nil}})
		called := false
		messenger().HandlePlatformMessage("flutter/textinput", message, func(data []byte) {
			called = true
			expect(data != nil, "no reply")
		}).Await()
		expect(called, "no callback")
	})
}

// Own: TextEditingValue equality compares text, selection and composing.
func TestTextEditingValueEquality(t *testing.T) {
	a := tev("abc", sel(0, 1))
	if !a.Equal(tev("abc", sel(0, 1))) {
		t.Fatal("equal values differ")
	}
	r := ui.NewTextRange(0, 1)
	for _, b := range []services.TextEditingValue{
		tev("abd", sel(0, 1)), tev("abc", sel(0, 2)),
		services.NewTextEditingValue(services.TextEditingValueOptions{Text: "abc", Selection: ptr(sel(0, 1)), Composing: &r}),
	} {
		if a.Equal(b) {
			t.Errorf("%v == %v", a, b)
		}
	}
}
