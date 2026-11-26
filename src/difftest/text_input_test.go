package difftest

// Go twin of flutter/test/text_input_test.dart: services' text input
// (configurations, editing values, the bytes sent on flutter/textinput,
// the client callbacks and replies for the platform's calls), autofill,
// keyboard inserted content, the system context menu and Clipboard,
// compared line by line.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var tiLog []string

func tiAdd(s string) { tiLog = append(tiLog, s) }

func tiMessenger() *flutter_test.TestDefaultBinaryMessenger {
	return flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
}

func tiBytes(data []byte) string {
	if data == nil {
		return "null"
	}
	return string(data)
}

func dartTypeName(v any) string {
	s := fmt.Sprintf("%T", v)
	return s[strings.LastIndex(s, ".")+1:]
}

// tiClient is LogClient.
type tiClient struct {
	services.TextInputClientMixin
	name  string
	value services.TextEditingValue
	scope services.AutofillScope
}

func (c *tiClient) add(s string) { tiAdd(c.name + "." + s) }

func (c *tiClient) CurrentTextEditingValue() *services.TextEditingValue { return &c.value }
func (c *tiClient) CurrentAutofillScope() services.AutofillScope        { return c.scope }
func (c *tiClient) UpdateEditingValue(v services.TextEditingValue) {
	c.add("updateEditingValue " + v.String())
}
func (c *tiClient) UpdateEditingValueWithDeltas(deltas []services.TextEditingDelta) {
	for _, d := range deltas {
		c.add("delta " + dartTypeName(d) + " " + d.OldText() + " " + d.Selection().String() + " " + d.Composing().String() +
			" -> " + d.Apply(c.value).String())
	}
}
func (c *tiClient) PerformAction(action services.TextInputAction) {
	c.add("performAction " + action.String())
}
func (c *tiClient) InsertContent(content services.KeyboardInsertedContent) {
	c.add("insertContent " + content.String() + " " + strconv.FormatBool(content.HasData()))
}
func (c *tiClient) PerformPrivateCommand(action string, data *dart.LinkedMap) {
	c.add("performPrivateCommand " + action + " " + dart.JSONEncode(data))
}
func (c *tiClient) UpdateFloatingCursor(point services.RawFloatingCursorPoint) {
	offset, start := "null", "null"
	if point.Offset != nil {
		offset = point.Offset.String()
	}
	if point.StartLocation != nil {
		start = "(" + point.StartLocation.Offset.String() + ", " + point.StartLocation.Position.String() + ")"
	}
	c.add("updateFloatingCursor " + point.State.String() + " " + offset + " " + start)
}
func (c *tiClient) ShowAutocorrectionPromptRect(start, end int) {
	c.add(fmt.Sprintf("showAutocorrectionPromptRect %d %d", start, end))
}
func (c *tiClient) OnFocusReceived() bool {
	c.add("onFocusReceived")
	return true
}
func (c *tiClient) ConnectionClosed() { c.add("connectionClosed") }
func (c *tiClient) DidChangeInputControl(oldControl, newControl services.TextInputControl) {
	name := func(control services.TextInputControl) string {
		switch control.(type) {
		case *tiControl:
			return "log"
		case nil:
			return "null"
		}
		return "platform"
	}
	c.add("didChangeInputControl " + name(oldControl) + " " + name(newControl))
}
func (c *tiClient) ShowToolbar()                    { c.add("showToolbar") }
func (c *tiClient) InsertTextPlaceholder(s ui.Size) { c.add("insertTextPlaceholder " + s.String()) }
func (c *tiClient) RemoveTextPlaceholder()          { c.add("removeTextPlaceholder") }
func (c *tiClient) PerformSelector(selectorName string) {
	c.add("performSelector " + selectorName)
}

// tiControl is LogControl.
type tiControl struct{ services.TextInputControlMixin }

func (c *tiControl) add(s string) { tiAdd("control." + s) }
func (c *tiControl) Attach(client services.TextInputClient, configuration services.TextInputConfiguration) {
	c.add("attach " + configuration.InputType.String())
}
func (c *tiControl) Detach(services.TextInputClient) { c.add("detach") }
func (c *tiControl) Show()                           { c.add("show") }
func (c *tiControl) Hide()                           { c.add("hide") }
func (c *tiControl) UpdateConfig(configuration services.TextInputConfiguration) {
	c.add("updateConfig " + configuration.InputType.String())
}
func (c *tiControl) SetEditingState(v services.TextEditingValue) {
	c.add("setEditingState " + v.String())
}
func (c *tiControl) SetEditableSizeAndTransform(size ui.Size, _ *vector_math.Matrix4) {
	c.add("setEditableSizeAndTransform " + size.String())
}
func (c *tiControl) SetComposingRect(r ui.Rect) { c.add("setComposingRect " + r.String()) }
func (c *tiControl) SetCaretRect(r ui.Rect)     { c.add("setCaretRect " + r.String()) }
func (c *tiControl) SetSelectionRects(rects []services.SelectionRect) {
	parts := make([]string, len(rects))
	for i, r := range rects {
		parts[i] = r.String()
	}
	c.add("setSelectionRects [" + strings.Join(parts, ", ") + "]")
}
func (c *tiControl) UpdateStyle(style *services.TextInputStyle) {
	c.add("updateStyle " + dart.JSONEncode(style.ToJSON()))
}
func (c *tiControl) RequestAutofill() { c.add("requestAutofill") }
func (c *tiControl) FinishAutofillContext(shouldSave bool) {
	c.add("finishAutofillContext " + strconv.FormatBool(shouldSave))
}

// tiScribble is LogScribble.
type tiScribble struct {
	id     string
	bounds ui.Rect
	always bool
}

func (s *tiScribble) ElementIdentifier() string       { return s.id }
func (s *tiScribble) Bounds() ui.Rect                 { return s.bounds }
func (s *tiScribble) OnScribbleFocus(o ui.Offset)     { tiAdd("scribble." + s.id + ".focus " + o.String()) }
func (s *tiScribble) IsInScribbleRect(r ui.Rect) bool { return s.always || s.bounds.Overlaps(r) }

// tiAutofillClient is LogAutofillClient.
type tiAutofillClient struct {
	id     string
	config services.TextInputConfiguration
}

func (c *tiAutofillClient) AutofillID() string { return c.id }
func (c *tiAutofillClient) TextInputConfiguration() services.TextInputConfiguration {
	return c.config
}
func (c *tiAutofillClient) Autofill(v services.TextEditingValue) {
	tiAdd("autofill." + c.id + " " + v.String())
}

// tiScope is LogScope.
type tiScope struct {
	services.AutofillScopeMixin
	clients []*tiAutofillClient
}

func (s *tiScope) AutofillClients() []services.AutofillClient {
	out := make([]services.AutofillClient, len(s.clients))
	for i, c := range s.clients {
		out[i] = c
	}
	return out
}

func (s *tiScope) GetAutofillClient(id string) services.AutofillClient {
	for _, c := range s.clients {
		if c.id == id {
			return c
		}
	}
	return nil
}

var (
	tiJSONMethod  = services.JSONMethodCodec{}
	tiJSONMessage = services.JSONMessageCodec{}
)

func tiPlatform(method string, args any) {
	message := tiJSONMessage.EncodeMessage([]dart.MapEntry[string, any]{{Key: "method", Value: method}, {Key: "args", Value: args}})
	tiMessenger().HandlePlatformMessage("flutter/textinput", message, func(reply []byte) {
		tiAdd("reply " + method + " " + tiBytes(reply))
	}).Await()
}

func tiDelay() { async.FutureDelayed(0, func() any { return nil }).Await() }

func tiP[T any](v T) *T { return &v }

func tiValue(text string, selection *services.TextSelection, composing *ui.TextRange) services.TextEditingValue {
	return services.NewTextEditingValue(services.TextEditingValueOptions{Text: text, Selection: selection, Composing: composing})
}

func tiConfigurations() {
	type opts = services.TextInputConfigurationOptions
	cfg := services.NewTextInputConfiguration
	configs := []services.TextInputConfiguration{
		cfg(),
		cfg(opts{ViewID: tiP(3), ReadOnly: true, ObscureText: true, Autocorrect: tiP(false)}),
		cfg(opts{ObscureText: true, SmartDashesType: tiP(services.SmartDashesTypeEnabled)}),
		cfg(opts{SmartQuotesType: tiP(services.SmartQuotesTypeDisabled), EnableSuggestions: tiP(false)}),
		cfg(opts{EnableInteractiveSelection: tiP(false), ActionLabel: tiP("Go!")}),
		cfg(opts{InputAction: tiP(services.TextInputActionNewline), KeyboardAppearance: tiP(ui.BrightnessDark)}),
		cfg(opts{TextCapitalization: tiP(services.TextCapitalizationSentences)}),
		cfg(opts{EnableIMEPersonalizedLearning: tiP(false), AllowedMimeTypes: []string{"image/png", "image/gif"}}),
		cfg(opts{EnableDeltaModel: true, EnableInlinePrediction: tiP(true)}),
		cfg(opts{EnableInlinePrediction: tiP(false), HintLocales: tiP[[]ui.Locale](nil)}),
		cfg(opts{HintLocales: &[]ui.Locale{ui.NewLocale("en", "US"), ui.NewLocale("pt"),
			ui.NewLocaleFromSubtags(ui.LocaleFromSubtagsOptions{LanguageCode: "zh", ScriptCode: tiP("Hant"), CountryCode: tiP("TW")})}}),
		cfg(opts{AutofillConfiguration: tiP(services.NewAutofillConfiguration("u1",
			[]string{services.AutofillHints.Email, services.AutofillHints.Username},
			tiValue("me@x", tiP(services.NewTextSelectionCollapsed(4)), nil),
			services.AutofillConfigurationOptions{HintText: tiP("Your mail")}))}),
		cfg(opts{AutofillConfiguration: tiP(services.NewAutofillConfiguration("u2", []string{}, services.TextEditingValueEmpty))}),
	}
	for _, ty := range services.TextInputTypeValues {
		configs = append(configs, cfg(opts{InputType: &ty}))
	}
	for _, s := range []bool{false, true} {
		for _, d := range []bool{false, true} {
			for _, p := range []bool{false, true} {
				ty := services.NewTextInputTypeNumberWithOptions(services.TextInputTypeNumberWithOptionsOptions{Signed: s, Decimal: d, Password: p})
				configs = append(configs, cfg(opts{InputType: &ty}))
			}
		}
	}
	for _, a := range services.TextInputActionValues {
		configs = append(configs, cfg(opts{InputAction: &a}))
	}
	for _, c := range services.TextCapitalizationValues {
		configs = append(configs, cfg(opts{TextCapitalization: &c}))
	}
	for _, c := range configs {
		tiAdd("json " + dart.JSONEncode(c.ToJSON()))
		tiAdd("str " + c.String())
	}
	base := configs[1]
	tiAdd("copy " + base.CopyWith().String())
	type copyOpts = services.TextInputConfigurationCopyWithOptions
	tiAdd("copy " + base.CopyWith(copyOpts{
		ViewID: tiP(9), InputType: &services.TextInputTypePhone, ReadOnly: tiP(false), ObscureText: tiP(false),
		Autocorrect: tiP(true), SmartDashesType: tiP(services.SmartDashesTypeEnabled),
		SmartQuotesType: tiP(services.SmartQuotesTypeEnabled), EnableSuggestions: tiP(false),
		EnableInteractiveSelection: tiP(false), ActionLabel: tiP("a"), InputAction: tiP(services.TextInputActionSearch),
		KeyboardAppearance: tiP(ui.BrightnessDark), TextCapitalization: tiP(services.TextCapitalizationWords),
		EnableIMEPersonalizedLearning: tiP(false), AllowedMimeTypes: []string{"x"}, EnableDeltaModel: tiP(true),
		HintLocales: []ui.Locale{ui.NewLocale("fr")}, EnableInlinePrediction: tiP(false),
	}).String())
	tiAdd(fmt.Sprintf("eq %v %v %v", base.Equal(base.CopyWith()), base.Equal(base.CopyWith(copyOpts{ReadOnly: tiP(false)})),
		cfg().Equal(cfg())))
	for _, ty := range services.TextInputTypeValues {
		tiAdd("type " + ty.String() + " " + dart.JSONEncode(ty.ToJSON()))
	}
	style := services.NewTextInputStyle(ui.TextDirectionRtl, ui.TextAlignEnd, services.TextInputStyleOptions{
		FontFamily: tiP("Roboto"), FontSize: tiP(16.5), FontWeight: &ui.FontWeightW300,
		LetterSpacing: tiP(0.1), WordSpacing: tiP(2.0), LineHeight: tiP(1.25),
	})
	tiAdd("style " + dart.JSONEncode(style.ToJSON()))
	tiAdd("style " + dart.JSONEncode(services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignJustify).ToJSON()))
	for _, w := range ui.FontWeightValues {
		tiAdd("weight " + dart.JSONEncode(services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignStart,
			services.TextInputStyleOptions{FontWeight: &w}).ToJSON()))
	}
	tiAdd("autofill " + configs[11].AutofillConfiguration.String())
	disabled := "null"
	if j := services.AutofillConfigurationDisabled.ToJSON(); j != nil {
		disabled = dart.JSONEncode(j)
	}
	tiAdd("autofill " + services.AutofillConfigurationDisabled.String() + " " + disabled)
}

func tiEditingValues() {
	up := ui.TextAffinityUpstream
	values := []services.TextEditingValue{
		services.TextEditingValueEmpty,
		tiValue("hello", tiP(services.NewTextSelection(1, 4)), nil),
		tiValue("a😀b👍🏽c", tiP(services.NewTextSelection(5, 1, services.TextSelectionOptions{Affinity: &up, IsDirectional: true})),
			tiP(ui.NewTextRange(1, 3))),
		tiValue("x", tiP(services.NewTextSelectionCollapsed(1, ui.TextAffinityUpstream)), nil),
	}
	for _, v := range values {
		j := dart.JSONEncode(v.ToJSON())
		tiAdd("value " + v.String() + " " + j + " " + strconv.FormatBool(v.IsComposingRangeValid()))
		tiAdd("fromJSON " + services.NewTextEditingValueFromJSON(tiJSONMessage.DecodeMessage([]byte(j)).(*dart.LinkedMap)).String())
	}
	for _, j := range []string{
		`{"text": "abc"}`,
		`{"text": "abc", "selectionBase": 1}`,
		`{"text": "abc", "selectionBase": 1, "selectionExtent": 2, "selectionAffinity": "TextAffinity.upstream", "selectionIsDirectional": true, "composingBase": 0, "composingExtent": 3}`,
		`{"text": "abc", "selectionBase": 2, "selectionExtent": 2, "selectionAffinity": "bogus"}`,
		`{"text": "\ud83d\ude00z", "selectionBase": 2, "selectionExtent": 3, "composingBase": 0, "composingExtent": 2}`,
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					tiAdd("fromJSON " + j + " throws " + dart.ToString(r.(dart.AssertionFailure).AssertionMessage()))
				}
			}()
			v := services.NewTextEditingValueFromJSON(tiJSONMessage.DecodeMessage([]byte(j)).(*dart.LinkedMap))
			tiAdd("fromJSON " + j + " -> " + v.String() + " " + dart.JSONEncode(v.ToJSON()))
		}()
	}
	const text = "From a 😀 false 👍🏽 proposition"
	selections := []services.TextSelection{
		services.NewTextSelectionCollapsed(0), services.NewTextSelectionCollapsed(7),
		services.NewTextSelection(5, 13), services.NewTextSelection(13, 5), services.NewTextSelection(20, 30),
		services.NewTextSelectionCollapsed(8, ui.TextAffinityUpstream).CopyWith(services.TextSelectionCopyWithOptions{IsDirectional: tiP(true)}),
		services.NewTextSelection(6, 10, services.TextSelectionOptions{IsDirectional: true}),
	}
	ranges := []ui.TextRange{
		ui.NewTextRange(0, 0), ui.NewTextRange(4, 5), ui.NewTextRange(5, 6), ui.NewTextRange(7, 9),
		ui.NewTextRange(8, 9), ui.NewTextRange(12, 13), ui.NewTextRange(13, 14), ui.NewTextRange(16, 20),
		ui.NewTextRange(0, 31), ui.TextRangeEmpty,
	}
	for _, s := range selections {
		for _, r := range ranges {
			for _, with := range []string{"", "AA", "😀", "xyz"} {
				v := tiValue(text, &s, tiP(ui.NewTextRange(2, 9))).Replaced(r, with)
				tiAdd(fmt.Sprintf("replaced %v %v \"%s\" -> %d %v %v %s", s, r, with, dart.StringLength(v.Text), v.Selection,
					v.Composing, dart.JSONEncode(v.Text)))
			}
		}
	}
}

func tiConnectionScene() {
	services.TextInputConnectionDebugResetID(5)
	client := &tiClient{name: "c1", value: tiValue("abc", tiP(services.NewTextSelectionCollapsed(3)), nil)}
	connection := services.TextInputAttach(client, services.NewTextInputConfiguration(services.TextInputConfigurationOptions{ActionLabel: tiP("ok")}))
	tiAdd("attached " + strconv.FormatBool(connection.Attached()))
	connection.Show()
	connection.SetEditingState(tiValue("abcd", tiP(services.NewTextSelection(0, 4)), nil))
	m := vector_math.NewMatrix4Identity()
	m.TranslateByDouble(10.0, 20.5, 0.0, 1.0)
	m.ScaleByDouble(2.0, 3.0, 1.0, 1.0)
	connection.SetEditableSizeAndTransform(ui.NewSize(100.5, 30), m)
	connection.SetEditableSizeAndTransform(ui.NewSize(100.5, 30), m)
	connection.SetEditableSizeAndTransform(ui.NewSize(100.5, 31), m)
	connection.SetEditableSizeAndTransform(ui.NewSize(100.5, 31), vector_math.NewMatrix4Identity())
	connection.SetComposingRect(ui.NewRectFromLTWH(1, 2, 3, 4))
	connection.SetComposingRect(ui.NewRectFromLTWH(1, 2, 3, 4))
	connection.SetComposingRect(ui.NewRectFromLTRB(0, 0, math.Inf(1), 5))
	connection.SetCaretRect(ui.NewRectFromLTWH(5.5, 6, 1, 14))
	connection.SetCaretRect(ui.NewRectFromLTWH(5.5, 6, 1, 14))
	connection.SetCaretRect(ui.NewRectFromLTWH(math.NaN(), 6, 1, 14))
	connection.SetSelectionRects([]services.SelectionRect{})
	rtl := ui.TextDirectionRtl
	rects := func() []services.SelectionRect {
		return []services.SelectionRect{
			services.NewSelectionRect(0, ui.NewRectFromLTWH(0, 0, 7.5, 14)),
			services.NewSelectionRect(1, ui.NewRectFromLTWH(7.5, 0, 8, 14), services.SelectionRectOptions{Direction: &rtl}),
		}
	}
	connection.SetSelectionRects(rects())
	connection.SetSelectionRects(rects())
	connection.UpdateStyle(services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignCenter, services.TextInputStyleOptions{
		FontFamily: tiP("Ahem"), FontSize: tiP(14.0), FontWeight: &ui.FontWeightBold, LetterSpacing: tiP(1.0),
		WordSpacing: tiP(0.5), LineHeight: tiP(1.5),
	}))
	connection.SetStyle(nil, tiP(12.0), nil, ui.TextDirectionRtl, ui.TextAlignRight)
	connection.RequestAutofill()
	connection.UpdateConfig(services.NewTextInputConfiguration(services.TextInputConfigurationOptions{
		InputType: &services.TextInputTypeMultiline, InputAction: tiP(services.TextInputActionNewline)}))
	services.TextInputFinishAutofillContext(services.TextInputFinishAutofillContextOptions{ShouldSave: tiP(false)})
	services.TextInputFinishAutofillContext()
	services.TextInputUpdateEditingValue(tiValue("zz", nil, nil))

	control := &tiControl{}
	services.TextInputSetInputControl(control)
	connection.Show()
	connection.UpdateConfig(services.NewTextInputConfiguration(services.TextInputConfigurationOptions{InputType: &services.TextInputTypeEmailAddress}))
	connection.SetEditingState(tiValue("q", nil, nil))
	connection.SetComposingRect(ui.NewRectFromLTWH(9, 9, 9, 9))
	connection.SetCaretRect(ui.NewRectFromLTWH(8, 8, 8, 8))
	connection.SetEditableSizeAndTransform(ui.NewSize(1, 1), vector_math.NewMatrix4Identity())
	connection.SetSelectionRects([]services.SelectionRect{services.NewSelectionRect(4, ui.NewRectFromLTWH(1, 1, 1, 1))})
	connection.UpdateStyle(services.NewTextInputStyle(ui.TextDirectionLtr, ui.TextAlignLeft))
	connection.RequestAutofill()
	services.TextInputFinishAutofillContext()
	services.TextInputUpdateEditingValue(tiValue("from control", nil, nil))
	client2 := &tiClient{name: "c2", value: services.TextEditingValueEmpty}
	connection2 := services.TextInputAttach(client2, services.NewTextInputConfiguration(services.TextInputConfigurationOptions{InputType: &services.TextInputTypeURL}))
	tiAdd(fmt.Sprintf("attached %v %v", connection.Attached(), connection2.Attached()))
	tiPlatform("TextInputClient.updateEditingState", []any{6, services.TextEditingValueEmpty.ToJSON()})
	services.TextInputSetInputControl(nil)
	connection2.Show()
	services.TextInputRestorePlatformInputControl()
	connection2.Show()
	connection2.Close()
	tiAdd("closed " + strconv.FormatBool(connection2.Attached()))
	connection2.Close()
	tiDelay()
	tiAdd("after microtask")
	connection3 := services.TextInputAttach(client2, services.NewTextInputConfiguration())
	connection3.Close()
	connection4 := services.TextInputAttach(client2, services.NewTextInputConfiguration())
	connection4.Close()
	tiDelay()
	tiAdd("after microtask 2")
	connection5 := services.TextInputAttach(client2, services.NewTextInputConfiguration())
	connection5.Close()
	connection6 := services.TextInputAttach(client2, services.NewTextInputConfiguration())
	tiDelay()
	tiAdd("after microtask 3 " + strconv.FormatBool(connection6.Attached()))
	connection6.ConnectionClosedReceived()
	tiAdd("received " + strconv.FormatBool(connection6.Attached()))

	scope := &tiScope{}
	scope.SetSelf(scope)
	scope.clients = []*tiAutofillClient{
		{id: "a", config: services.NewTextInputConfiguration(services.TextInputConfigurationOptions{AutofillConfiguration: tiP(
			services.NewAutofillConfiguration("a", []string{"username"}, tiValue("joe", nil, nil)))})},
		{id: "b", config: services.NewTextInputConfiguration(services.TextInputConfigurationOptions{ObscureText: true, AutofillConfiguration: tiP(
			services.NewAutofillConfiguration("b", []string{"password"}, services.TextEditingValueEmpty,
				services.AutofillConfigurationOptions{HintText: tiP("pw")}))})},
	}
	trigger := &tiClient{name: "t", value: services.TextEditingValueEmpty, scope: scope}
	autofillConnection := scope.Attach(trigger, services.NewTextInputConfiguration(services.TextInputConfigurationOptions{
		ViewID: tiP(2), ReadOnly: true, EnableDeltaModel: true, InputType: &services.TextInputTypeName, ActionLabel: tiP("x"),
		AutofillConfiguration: tiP(services.NewAutofillConfiguration("a", []string{"username"}, tiValue("joe", nil, nil))),
	}))
	tiPlatform("TextInputClient.updateEditingStateWithTag", []any{0, []dart.MapEntry[string, any]{
		{Key: "a", Value: tiValue("jim", tiP(services.NewTextSelectionCollapsed(3)), nil).ToJSON()},
		{Key: "zzz", Value: services.TextEditingValueEmpty.ToJSON()},
	}})
	autofillConnection.Close()
	tiDelay()
}

func tiIncomingScene() {
	services.TextInputConnectionDebugResetID()
	client := &tiClient{name: "in", value: tiValue("hello", tiP(services.NewTextSelectionCollapsed(5)), nil)}
	connection := services.TextInputAttach(client, services.NewTextInputConfiguration())
	id := 1
	tiPlatform("TextInputClient.updateEditingState", []any{id, tiValue("hi😀", tiP(services.NewTextSelection(2, 4)), tiP(ui.NewTextRange(0, 2))).ToJSON()})
	tiPlatform("TextInputClient.updateEditingState", []any{id + 1, services.TextEditingValueEmpty.ToJSON()})
	tiPlatform("TextInputClient.updateEditingState", []any{-1, tiValue("any", nil, nil).ToJSON()})
	type m = []dart.MapEntry[string, any]
	tiPlatform("TextInputClient.updateEditingStateWithDeltas", []any{id, m{{Key: "deltas", Value: []any{
		m{{Key: "oldText", Value: "hello"}, {Key: "deltaText", Value: "!"}, {Key: "deltaStart", Value: 5}, {Key: "deltaEnd", Value: 5}, {Key: "selectionBase", Value: 6}, {Key: "selectionExtent", Value: 6}, {Key: "composingBase", Value: -1}, {Key: "composingExtent", Value: -1}},
		m{{Key: "oldText", Value: "hello"}, {Key: "deltaText", Value: ""}, {Key: "deltaStart", Value: 3}, {Key: "deltaEnd", Value: 5}, {Key: "selectionBase", Value: 3}, {Key: "selectionExtent", Value: 3}},
		m{{Key: "oldText", Value: "hello"}, {Key: "deltaText", Value: "p"}, {Key: "deltaStart", Value: 0}, {Key: "deltaEnd", Value: 1}, {Key: "selectionBase", Value: 1}, {Key: "selectionExtent", Value: 1}, {Key: "selectionAffinity", Value: "TextAffinity.upstream"}, {Key: "selectionIsDirectional", Value: true}},
		m{{Key: "oldText", Value: "hello"}, {Key: "deltaText", Value: ""}, {Key: "deltaStart", Value: -1}, {Key: "deltaEnd", Value: -1}, {Key: "selectionBase", Value: 0}, {Key: "selectionExtent", Value: 2}, {Key: "composingBase", Value: 0}, {Key: "composingExtent", Value: 5}},
	}}}})
	for _, action := range services.TextInputActionValues {
		tiPlatform("TextInputClient.performAction", []any{id, action.String()})
	}
	tiPlatform("TextInputClient.performAction", []any{id, "TextInputAction.bogus"})
	tiPlatform("TextInputClient.performAction", []any{id, "TextInputAction.commitContent",
		m{{Key: "mimeType", Value: "image/gif"}, {Key: "data", Value: []int{0, 1, 255, 256, 513}}, {Key: "uri", Value: "content://x/y.gif"}}})
	tiPlatform("TextInputClient.performAction", []any{id, "TextInputAction.commitContent",
		m{{Key: "mimeType", Value: "image/png"}, {Key: "uri", Value: "content://z"}}})
	tiPlatform("TextInputClient.performAction", []any{id, "TextInputAction.commitContent",
		m{{Key: "mimeType", Value: "image/webp"}, {Key: "data", Value: []int{}}, {Key: "uri", Value: "content://e"}}})
	tiPlatform("TextInputClient.performSelectors", []any{id, []string{"deleteBackward:", "insertNewline:"}})
	tiPlatform("TextInputClient.performPrivateCommand", []any{id, m{{Key: "action", Value: "a1"}, {Key: "data", Value: m{{Key: "k", Value: []any{1, 2.5, "x"}}}}}})
	tiPlatform("TextInputClient.performPrivateCommand", []any{id, m{{Key: "action", Value: "a2"}}})
	tiPlatform("TextInputClient.updateFloatingCursor", []any{id, "FloatingCursorDragState.start", m{{Key: "X", Value: 1}, {Key: "Y", Value: 2}}})
	tiPlatform("TextInputClient.updateFloatingCursor", []any{id, "FloatingCursorDragState.update", m{{Key: "X", Value: 3}, {Key: "Y", Value: 4.5}}})
	tiPlatform("TextInputClient.updateFloatingCursor", []any{id, "FloatingCursorDragState.end", m{{Key: "X", Value: 0}, {Key: "Y", Value: 0}}})
	tiPlatform("TextInputClient.updateFloatingCursor", []any{id, "FloatingCursorDragState.bogus", m{{Key: "X", Value: 0}, {Key: "Y", Value: 0}}})
	tiPlatform("TextInputClient.showAutocorrectionPromptRect", []any{id, 2, 4})
	tiPlatform("TextInputClient.showToolbar", []any{id})
	tiPlatform("TextInputClient.insertTextPlaceholder", []any{id, 10, 20.5})
	tiPlatform("TextInputClient.removeTextPlaceholder", []any{id})
	tiPlatform("TextInputClient.requestExistingInputState", nil)
	tiPlatform("TextInputClient.onFocusReceived", []any{id})
	tiPlatform("TextInputClient.onFocusReceived", []any{99})
	tiPlatform("TextInputClient.unknownMethod", []any{id})
	tiPlatform("TextInputClient.unknownMethod", []any{42})

	services.TextInputRegisterScribbleElement("e1", &tiScribble{"e1", ui.NewRectFromLTWH(0, 0, 50, 50), false})
	services.TextInputRegisterScribbleElement("e2", &tiScribble{"e2", ui.NewRectFromLTWH(40, 40, 50, 50), false})
	services.TextInputRegisterScribbleElement("e3", &tiScribble{"e3", ui.RectZero, false})
	services.TextInputRegisterScribbleElement("e0", &tiScribble{"e0", ui.NewRectFromLTWH(45, 0, 10, 10), false})
	services.TextInputRegisterScribbleElement("e4", &tiScribble{"e4", ui.RectZero, true})
	services.TextInputRegisterScribbleElement("e5", &tiScribble{"e5", ui.NewRectFromLTRB(0, 0, math.Inf(1), 10), true})
	services.TextInputRegisterScribbleElement("e6", &tiScribble{"e6", ui.NewRectFromLTRB(0, 0, math.NaN(), 10), true})
	services.TextInputRegisterScribbleElement("e7", &tiScribble{"e7", ui.NewRectFromLTWH(500, 500, 1, 1), true})
	tiPlatform("TextInputClient.focusElement", []any{"e2", 3, 4.5})
	tiPlatform("TextInputClient.focusElement", []any{"nope", 3, 4.5})
	tiPlatform("TextInputClient.requestElementsInRect", []any{30, 0, 20, 60.5})
	tiPlatform("TextInputClient.requestElementsInRect", []any{0, 0, 1000, 1000})
	services.TextInputUnregisterScribbleElement("e1")
	tiPlatform("TextInputClient.requestElementsInRect", []any{0, 0, 1000, 1000})
	tiPlatform("TextInputClient.scribbleInteractionBegan", []any{id})
	tiAdd("scribble " + strconv.FormatBool(connection.ScribbleInProgress()))
	tiPlatform("TextInputClient.scribbleInteractionFinished", []any{id})
	tiAdd("scribble " + strconv.FormatBool(connection.ScribbleInProgress()))
	services.TextInputUnregisterScribbleElement("e2")
	services.TextInputUnregisterScribbleElement("e3")
	services.TextInputUnregisterScribbleElement("e0")
	for _, id := range []string{"e4", "e5", "e6", "e7"} {
		services.TextInputUnregisterScribbleElement(id)
	}

	tiPlatform("TextInputClient.onConnectionClosed", []any{id})
	connection.ConnectionClosedReceived()
	tiPlatform("TextInputClient.updateEditingState", []any{id, services.TextEditingValueEmpty.ToJSON()})
	tiPlatform("TextInputClient.onFocusReceived", []any{id})
	tiPlatform("TextInputClient.requestExistingInputState", nil)
}

func tiPlatformScene() {
	var platformCalls []string
	var clipboard any
	tiMessenger().SetMockMessageHandler("flutter/platform", func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			tiAdd("platform " + tiBytes(message))
			call := tiJSONMethod.DecodeMethodCall(message)
			platformCalls = append(platformCalls, call.Method)
			switch call.Method {
			case "Clipboard.setData":
				clipboard = call.Arguments
			case "Clipboard.getData":
				return tiJSONMethod.EncodeSuccessEnvelope(clipboard)
			case "Clipboard.hasStrings":
				data, _ := clipboard.(*dart.LinkedMap)
				return tiJSONMethod.EncodeSuccessEnvelope(map[string]any{"value": data != nil && data.Lookup("text") != nil})
			}
			return tiJSONMethod.EncodeSuccessEnvelope(nil)
		})
	})
	text := func(d *services.ClipboardData) string {
		if d == nil || d.Text == nil {
			return "null"
		}
		return *d.Text
	}
	tiAdd("hasStrings " + strconv.FormatBool(services.Clipboard.HasStrings().Await()))
	tiAdd("getData " + text(services.Clipboard.GetData(services.ClipboardKTextPlain).Await()))
	services.Clipboard.SetData(services.NewClipboardData("copied 😀")).Await()
	tiAdd("hasStrings " + strconv.FormatBool(services.Clipboard.HasStrings().Await()))
	tiAdd("getData " + text(services.Clipboard.GetData("text/plain").Await()))

	services.TextInputConnectionDebugResetID()
	connection := services.TextInputAttach(&tiClient{name: "m", value: services.TextEditingValueEmpty}, services.NewTextInputConfiguration())
	controller := services.NewSystemContextMenuController(services.SystemContextMenuControllerOptions{OnSystemHide: func() { tiAdd("onSystemHide") }})
	items := func() []services.IOSSystemContextMenuItemData {
		return []services.IOSSystemContextMenuItemData{
			services.IOSSystemContextMenuItemDataCut{}, services.IOSSystemContextMenuItemDataCopy{},
			services.IOSSystemContextMenuItemDataPaste{}, services.IOSSystemContextMenuItemDataSelectAll{},
			services.NewIOSSystemContextMenuItemDataLookUp("Look Up"), services.NewIOSSystemContextMenuItemDataSearchWeb("Search Web"),
			services.NewIOSSystemContextMenuItemDataShare("Share… 😀"), services.IOSSystemContextMenuItemDataLiveText{},
		}
	}
	controller.ShowWithItems(ui.NewRectFromLTWH(1, 2, 30.5, 40), items()).Await()
	tiAdd("visible " + strconv.FormatBool(controller.IsVisible()))
	controller.ShowWithItems(ui.NewRectFromLTWH(1, 2, 30.5, 40), items()).Await()
	controller.ShowWithItems(ui.NewRectFromLTWH(1, 2, 30.5, 40), []services.IOSSystemContextMenuItemData{services.IOSSystemContextMenuItemDataCopy{}}).Await()
	controller.Show(ui.NewRectFromLTWH(0, 0, 1, 1)).Await()
	tiMessenger().HandlePlatformMessage("flutter/platform",
		tiJSONMethod.EncodeMethodCall(services.MethodCall{Method: "ContextMenu.onDismissSystemContextMenu"}),
		func(reply []byte) { tiAdd("reply " + tiBytes(reply)) }).Await()
	tiAdd("visible " + strconv.FormatBool(controller.IsVisible()))
	controller.Hide().Await()
	controller.ShowWithItems(ui.NewRectFromLTWH(5, 5, 5, 5), []services.IOSSystemContextMenuItemData{services.NewIOSSystemContextMenuItemDataSearchWeb("")}).Await()
	controller.Hide().Await()
	tiAdd("visible " + strconv.FormatBool(controller.IsVisible()))
	controller.Dispose()
	connection.Close()
	tiAdd("platform calls [" + strings.Join(platformCalls, ", ") + "]")
	tiMessenger().SetMockMessageHandler("flutter/platform", nil)
}

func TestTextInputMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/text_input.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	if !isolated(t) { // it constructs its own binding
		return
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	newChannelsBinding()
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	flutter_test.SetCurrentTest(t)

	body := func() {
		foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
			e := details.Exception
			var shown string
			switch e.(type) {
			case *services.MissingPluginException, *foundation.FlutterError, *services.PlatformException:
				shown = dart.ToString(e)
			default:
				shown = fmt.Sprintf("%T", e)
			}
			library := "null"
			if details.Library != nil {
				library = *details.Library
			}
			tiAdd("error " + shown + " | " + details.Context.String() + " | " + library)
		}
		tiMessenger().SetMockMessageHandler("flutter/textinput", func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				tiAdd("out " + tiBytes(message))
				return tiJSONMethod.EncodeSuccessEnvelope(nil)
			})
		})
		tiConfigurations()
		tiEditingValues()
		tiConnectionScene()
		tiIncomingScene()

		tiMessenger().SetMockMessageHandler("flutter/textinput", func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				call := tiJSONMethod.DecodeMethodCall(message)
				tiAdd("out " + call.Method)
				msg := "failed " + call.Method
				return tiJSONMethod.EncodeErrorEnvelope("E"+strconv.Itoa(len(call.Method)),
					services.EncodeErrorEnvelopeOptions{Message: &msg, Details: []int{1}})
			})
		})
		services.TextInputConnectionDebugResetID()
		c := services.TextInputAttach(&tiClient{name: "e", value: services.TextEditingValueEmpty}, services.NewTextInputConfiguration())
		c.Show()
		c.SetEditingState(services.TextEditingValueEmpty)
		c.UpdateConfig(services.NewTextInputConfiguration())
		c.SetCaretRect(ui.RectZero)
		c.RequestAutofill()
		c.Close()
		tiDelay()
		tiDelay()
		tiMessenger().SetMockMessageHandler("flutter/textinput", nil)

		tiPlatformScene()
	}

	done := false
	var failure error
	async.OnComplete(async.AsyncVoid(body), func(async.Void) { done = true }, func(err error) { failure = err; done = true })
	for i := 0; !done && i < 100000; i++ {
		loop.FlushMicrotasks()
		if done || loop.RunPosted() > 0 {
			continue
		}
		if d, ok := loop.NextDeadline(); ok {
			loop.ElapseTo(d)
		} else {
			break
		}
	}
	if failure != nil {
		t.Fatalf("scenes failed: %v", failure)
	}
	if !done {
		t.Fatal("scenes never completed")
	}
	for i := range max(len(tiLog), len(ref.Log)) {
		var got, want string
		if i < len(tiLog) {
			got = tiLog[i]
		}
		if i < len(ref.Log) {
			want = ref.Log[i]
		}
		if got != want {
			t.Fatalf("line %d differs:\n Go:      %q\n Flutter: %q\ncontext (Go): %q", i, got, want, tiLog[max(0, i-5):min(len(tiLog), i+3)])
		}
	}
	t.Logf("%d lines identical", len(tiLog))
}
