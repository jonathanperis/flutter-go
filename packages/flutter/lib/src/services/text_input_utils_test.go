package services_test

// Port of packages/flutter/test/services/text_input_utils.dart (Flutter
// 3.50.1): FakeTextChannel, FakeScribbleElement, FakeTextInputClient.

import (
	"fmt"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// fakeTextChannel is FakeTextChannel: a services.TextInputChannel that
// records the calls the framework sends and answers with outgoing.
type fakeTextChannel struct {
	outgoing      func(call services.MethodCall) async.Future[any]
	incoming      func(call services.MethodCall) async.Future[any]
	outgoingCalls []services.MethodCall
}

func newFakeTextChannel(outgoing func(call services.MethodCall) async.Future[any]) *fakeTextChannel {
	return &fakeTextChannel{outgoing: outgoing}
}

func newFakeTextChannelNull() *fakeTextChannel {
	return newFakeTextChannel(func(services.MethodCall) async.Future[any] {
		return async.Async(func() any { return nil })
	})
}

func (c *fakeTextChannel) InvokeMethod(method string, arguments ...any) async.Future[any] {
	return async.Async(func() any {
		var args any
		if len(arguments) > 0 {
			args = arguments[0]
		}
		call := services.MethodCall{Method: method, Arguments: args}
		c.outgoingCalls = append(c.outgoingCalls, call)
		return c.outgoing(call).Await()
	})
}

func (c *fakeTextChannel) SetMethodCallHandler(handler func(call services.MethodCall) async.Future[any]) {
	c.incoming = handler
}

// validateOutgoingMethodCalls compares the calls' JSON encodings.
func (c *fakeTextChannel) validateOutgoingMethodCalls(calls []services.MethodCall) error {
	if len(c.outgoingCalls) != len(calls) {
		return fmt.Errorf("%d outgoing calls, want %d: %v", len(c.outgoingCalls), len(calls), c.outgoingCalls)
	}
	var output strings.Builder
	for i := range calls {
		outgoing := string(jsonMethod.EncodeMethodCall(c.outgoingCalls[i]))
		expected := string(jsonMethod.EncodeMethodCall(calls[i]))
		if outgoing != expected {
			fmt.Fprintf(&output, "Index %d did not match:\n  actual:   %s\n  expected: %s\n", i, outgoing, expected)
		}
	}
	if output.Len() > 0 {
		return fmt.Errorf("Calls did not match:\n%s", output.String())
	}
	return nil
}

// fakeScribbleElement is FakeScribbleElement.
type fakeScribbleElement struct {
	elementIdentifier string
	bounds            ui.Rect
	latestMethodCall  string
}

func (e *fakeScribbleElement) Bounds() ui.Rect                    { return e.bounds }
func (e *fakeScribbleElement) ElementIdentifier() string          { return e.elementIdentifier }
func (e *fakeScribbleElement) IsInScribbleRect(rect ui.Rect) bool { return e.bounds.Overlaps(rect) }
func (e *fakeScribbleElement) OnScribbleFocus(offset ui.Offset) {
	e.latestMethodCall = "onScribbleFocus"
}

// fakeTextInputClient is text_input_test.dart's FakeTextInputClient (the
// utils' one plus onFocusReceived).
type fakeTextInputClient struct {
	services.TextInputClientMixin
	latestMethodCall          string
	performedSelectors        []string
	latestPrivateCommandData  *dart.LinkedMap
	currentTextEditingValue   services.TextEditingValue
	latestFloatingCursorPoint *services.RawFloatingCursorPoint
}

func newFakeTextInputClient(value services.TextEditingValue) *fakeTextInputClient {
	return &fakeTextInputClient{currentTextEditingValue: value}
}

func (c *fakeTextInputClient) CurrentTextEditingValue() *services.TextEditingValue {
	return &c.currentTextEditingValue
}
func (c *fakeTextInputClient) CurrentAutofillScope() services.AutofillScope { return nil }
func (c *fakeTextInputClient) PerformAction(services.TextInputAction) {
	c.latestMethodCall = "performAction"
}
func (c *fakeTextInputClient) PerformPrivateCommand(action string, data *dart.LinkedMap) {
	c.latestMethodCall = "performPrivateCommand"
	c.latestPrivateCommandData = data
}
func (c *fakeTextInputClient) InsertContent(services.KeyboardInsertedContent) {
	c.latestMethodCall = "commitContent"
}
func (c *fakeTextInputClient) UpdateEditingValue(services.TextEditingValue) {
	c.latestMethodCall = "updateEditingValue"
}
func (c *fakeTextInputClient) UpdateFloatingCursor(point services.RawFloatingCursorPoint) {
	c.latestMethodCall = "updateFloatingCursor"
	c.latestFloatingCursorPoint = &point
}
func (c *fakeTextInputClient) ConnectionClosed() { c.latestMethodCall = "connectionClosed" }
func (c *fakeTextInputClient) ShowAutocorrectionPromptRect(start, end int) {
	c.latestMethodCall = "showAutocorrectionPromptRect"
}
func (c *fakeTextInputClient) ShowToolbar() { c.latestMethodCall = "showToolbar" }
func (c *fakeTextInputClient) configuration() services.TextInputConfiguration {
	return services.NewTextInputConfiguration()
}
func (c *fakeTextInputClient) DidChangeInputControl(oldControl, newControl services.TextInputControl) {
	c.latestMethodCall = "didChangeInputControl"
}
func (c *fakeTextInputClient) InsertTextPlaceholder(ui.Size) {
	c.latestMethodCall = "insertTextPlaceholder"
}
func (c *fakeTextInputClient) RemoveTextPlaceholder() { c.latestMethodCall = "removeTextPlaceholder" }
func (c *fakeTextInputClient) PerformSelector(selectorName string) {
	c.latestMethodCall = "performSelector"
	c.performedSelectors = append(c.performedSelectors, selectorName)
}
func (c *fakeTextInputClient) OnFocusReceived() bool {
	c.latestMethodCall = "onFocusReceived"
	return true
}

// fakeTextInputControl is text_input_test.dart's FakeTextInputControl.
type fakeTextInputControl struct {
	services.TextInputControlMixin
	methodCalls []string
	inputType   services.TextInputType
}

func (c *fakeTextInputControl) Attach(client services.TextInputClient, configuration services.TextInputConfiguration) {
	c.methodCalls = append(c.methodCalls, "attach")
	c.inputType = configuration.InputType
}
func (c *fakeTextInputControl) Detach(services.TextInputClient) {
	c.methodCalls = append(c.methodCalls, "detach")
}
func (c *fakeTextInputControl) SetEditingState(services.TextEditingValue) {
	c.methodCalls = append(c.methodCalls, "setEditingState")
}
func (c *fakeTextInputControl) UpdateConfig(configuration services.TextInputConfiguration) {
	c.methodCalls = append(c.methodCalls, "updateConfig")
	c.inputType = configuration.InputType
}
func (c *fakeTextInputControl) Show() { c.methodCalls = append(c.methodCalls, "show") }
func (c *fakeTextInputControl) Hide() { c.methodCalls = append(c.methodCalls, "hide") }
func (c *fakeTextInputControl) SetComposingRect(ui.Rect) {
	c.methodCalls = append(c.methodCalls, "setComposingRect")
}
func (c *fakeTextInputControl) SetCaretRect(ui.Rect) {
	c.methodCalls = append(c.methodCalls, "setCaretRect")
}
func (c *fakeTextInputControl) SetEditableSizeAndTransform(ui.Size, *vector_math.Matrix4) {
	c.methodCalls = append(c.methodCalls, "setEditableSizeAndTransform")
}
func (c *fakeTextInputControl) SetSelectionRects([]services.SelectionRect) {
	c.methodCalls = append(c.methodCalls, "setSelectionRects")
}
func (c *fakeTextInputControl) SetStyle(*string, *float64, *ui.FontWeight, ui.TextDirection, ui.TextAlign) {
	c.methodCalls = append(c.methodCalls, "setStyle")
}
func (c *fakeTextInputControl) UpdateStyle(*services.TextInputStyle) {
	c.methodCalls = append(c.methodCalls, "updateStyle")
}
func (c *fakeTextInputControl) FinishAutofillContext(bool) {
	c.methodCalls = append(c.methodCalls, "finishAutofillContext")
}
func (c *fakeTextInputControl) RequestAutofill() {
	c.methodCalls = append(c.methodCalls, "requestAutofill")
}
