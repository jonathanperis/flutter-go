package flutter_test

// Port of packages/flutter_test/lib/src/test_text_input.dart (Flutter
// 3.50.1): TestTextInput, the binding's mock of the platform's text input
// plugin, with its macOS key handler (test_text_input_key_handler.go). Not
// ported: TestAsyncUtils.guard around receiveAction.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// TestTextInput is a testing utility for simulating the text input control
// of the platform: it records the method calls the framework sends on the
// text input channel and sends the platform's calls back.
type TestTextInput struct {
	// OnCleared is called when the framework calls TextInput.clearClient
	// (nil: null).
	OnCleared func()
	// Log is the messages received on the text input channel.
	Log []services.MethodCall
	// SetClientArgs are the arguments of the last TextInput.setClient or
	// TextInput.updateConfig call (nil: null).
	SetClientArgs *dart.LinkedMap
	// EditingState is the last value sent with TextInput.setEditingState
	// (nil: null).
	EditingState *dart.LinkedMap

	client    *int
	isVisible bool
	// keyHandler is the platform-specific key handler that can process
	// unhandled keyboard events (nil: none).
	keyHandler TestTextInputKeyHandler
	handler    func(services.MethodCall) async.Future[any]
}

// TestTextInputOptions are TestTextInput's named parameters.
type TestTextInputOptions struct{ OnCleared func() }

// NewTestTextInput is TestTextInput({onCleared}).
func NewTestTextInput(opts ...TestTextInputOptions) *TestTextInput {
	t := &TestTextInput{}
	if len(opts) > 0 {
		t.OnCleared = opts[0].OnCleared
	}
	t.handler = t.handleTextInputCall
	return t
}

func testMessenger() *TestDefaultBinaryMessenger {
	return TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*TestDefaultBinaryMessenger)
}

// Register installs this object as a mock handler for
// SystemChannels.TextInput.
func (t *TestTextInput) Register() {
	testMessenger().SetMockMethodCallHandler(services.SystemChannels.TextInput, t.handler)
}

// Unregister removes this object as a mock handler for
// SystemChannels.TextInput.
func (t *TestTextInput) Unregister() {
	testMessenger().SetMockMethodCallHandler(services.SystemChannels.TextInput, nil)
}

// IsRegistered is whether this TestTextInput is registered with
// SystemChannels.TextInput.
func (t *TestTextInput) IsRegistered() bool {
	return testMessenger().CheckMockMessageHandler(services.SystemChannels.TextInput.Name, t.handler)
}

func (t *TestTextInput) assertRegistered() {
	if !t.IsRegistered() {
		panic(dart.NewAssertionError(nil))
	}
}

// HasAnyClients is whether there are any active clients listening to text
// input.
func (t *TestTextInput) HasAnyClients() bool {
	t.assertRegistered()
	return t.client != nil && *t.client > 0
}

// IsVisible is whether the onscreen keyboard is visible to the user.
func (t *TestTextInput) IsVisible() bool {
	t.assertRegistered()
	return t.isVisible
}

// Reset resets any internal state of this object and calls Log's clear.
func (t *TestTextInput) Reset() {
	t.Log = nil
	t.client = nil
	t.SetClientArgs = nil
	t.EditingState = nil
	t.isVisible = false
	t.keyHandler = nil
}

func (t *TestTextInput) handleTextInputCall(methodCall services.MethodCall) async.Future[any] {
	return async.Async(func() any {
		t.Log = append(t.Log, methodCall)
		switch methodCall.Method {
		case "TextInput.setClient":
			arguments := methodCall.Arguments.([]any)
			client := arguments[0].(int)
			t.client = &client
			t.SetClientArgs = arguments[1].(*dart.LinkedMap)
		case "TextInput.updateConfig":
			t.SetClientArgs = methodCall.Arguments.(*dart.LinkedMap)
		case "TextInput.clearClient":
			t.client = nil
			t.isVisible = false
			t.keyHandler = nil
			if t.OnCleared != nil {
				t.OnCleared()
			}
		case "TextInput.setEditingState":
			t.EditingState = methodCall.Arguments.(*dart.LinkedMap)
		case "TextInput.show":
			t.isVisible = true
			if !foundation.KIsWeb && foundation.DefaultTargetPlatform() == foundation.TargetPlatformMacOS && t.keyHandler == nil {
				t.keyHandler = NewMacOSTestTextInputKeyHandler(t.clientID())
			}
		case "TextInput.hide":
			t.isVisible = false
			t.keyHandler = nil
		}
		return nil
	})
}

// Hide simulates the user hiding the onscreen keyboard.
func (t *TestTextInput) Hide() {
	t.assertRegistered()
	t.isVisible = false
}

func (t *TestTextInput) clientID() int {
	if t.client != nil {
		return *t.client
	}
	return -1
}

func (t *TestTextInput) send(method string, arguments any, callback ui.PlatformMessageResponseCallback) async.Future[[]byte] {
	channel := services.SystemChannels.TextInput
	return testMessenger().HandlePlatformMessage(channel.Name,
		channel.Codec.EncodeMethodCall(services.MethodCall{Method: method, Arguments: arguments}), callback)
}

// EnterText simulates the user changing the text of the focused text
// field, and resetting the selection to the end of the new text.
func (t *TestTextInput) EnterText(text string) {
	selection := services.NewTextSelectionCollapsed(dart.StringLength(text))
	t.UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOptions{Text: text, Selection: &selection}))
}

// UpdateEditingValue simulates the user changing the TextEditingValue to
// the given value.
func (t *TestTextInput) UpdateEditingValue(value services.TextEditingValue) {
	t.send("TextInputClient.updateEditingState", []any{t.clientID(), value.ToJSON()}, func([]byte) {})
}

// ReceiveAction simulates the user pressing one of the TextInputAction
// buttons; the future fails with the error the framework replied with.
func (t *TestTextInput) ReceiveAction(action services.TextInputAction) async.Future[async.Void] {
	completer := async.NewCompleter[async.Void]()
	t.send("TextInputClient.performAction", []any{t.clientID(), action.String()}, func(data []byte) {
		if data == nil {
			panic(dart.NewAssertionError(nil))
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = dart.Exception{Message: dart.ToString(r)}
					}
					completer.CompleteError(err)
				}
			}()
			// Decoding throws a PlatformException if the data represents an
			// error, and that's all we care about here.
			services.SystemChannels.TextInput.Codec.DecodeEnvelope(data)
			completer.Complete(async.Void{})
		}()
	})
	return completer.Future()
}

// CloseConnection simulates the platform closing the connection.
func (t *TestTextInput) CloseConnection() {
	t.send("TextInputClient.onConnectionClosed", []any{t.clientID()}, func([]byte) {})
}

func (t *TestTextInput) sendAndWait(method string, arguments any, callback ui.PlatformMessageResponseCallback) async.Future[async.Void] {
	t.assertRegistered()
	f := t.send(method, arguments, callback)
	return async.AsyncVoid(func() { f.Await() })
}

// StartScribbleInteraction simulates a Scribble interaction starting.
func (t *TestTextInput) StartScribbleInteraction() async.Future[async.Void] {
	return t.sendAndWait("TextInputClient.scribbleInteractionBegan", []any{t.clientID()}, func([]byte) {})
}

// FinishScribbleInteraction simulates a Scribble interaction finishing.
func (t *TestTextInput) FinishScribbleInteraction() async.Future[async.Void] {
	return t.sendAndWait("TextInputClient.scribbleInteractionFinished", []any{t.clientID()}, func([]byte) {})
}

// ScribbleFocusElement simulates a Scribble focus.
func (t *TestTextInput) ScribbleFocusElement(elementIdentifier string, offset ui.Offset) async.Future[async.Void] {
	return t.sendAndWait("TextInputClient.focusElement", []any{elementIdentifier, offset.Dx, offset.Dy}, func([]byte) {})
}

// ScribbleRequestElementsInRect simulates iOS asking for the list of
// Scribble elements during UIIndirectScribbleInteraction.
func (t *TestTextInput) ScribbleRequestElementsInRect(rect ui.Rect) async.Future[[][]any] {
	t.assertRegistered()
	response := [][]any{}
	f := t.send("TextInputClient.requestElementsInRect", []any{rect.Left, rect.Top, rect.Width(), rect.Height()},
		func(data []byte) {
			response = nil
			for _, element := range services.SystemChannels.TextInput.Codec.DecodeEnvelope(data).([]any) {
				response = append(response, element.([]any))
			}
		})
	return async.Async(func() [][]any {
		f.Await()
		return response
	})
}

// ScribbleInsertPlaceholder simulates iOS inserting a UITextPlaceholder
// during a long press with the pencil.
func (t *TestTextInput) ScribbleInsertPlaceholder() async.Future[async.Void] {
	return t.sendAndWait("TextInputClient.insertTextPlaceholder", []any{t.clientID(), 0.0, 0.0}, func([]byte) {})
}

// ScribbleRemovePlaceholder simulates iOS removing a UITextPlaceholder after
// a long press with the pencil is released.
func (t *TestTextInput) ScribbleRemovePlaceholder() async.Future[async.Void] {
	return t.sendAndWait("TextInputClient.removeTextPlaceholder", []any{t.clientID()}, func([]byte) {})
}

// HandleKeyboardUndo simulates iOS responding to an undo or redo
// keyboard shortcut.
func (t *TestTextInput) HandleKeyboardUndo(direction string) async.Future[async.Void] {
	return t.sendAndWait("TextInputClient.handleUndo", []any{direction}, func([]byte) {})
}

// HandleKeyDownEvent gives text input a chance to respond to an unhandled
// key down event.
func (t *TestTextInput) HandleKeyDownEvent(key services.LogicalKeyboardKey) async.Future[async.Void] {
	if t.keyHandler == nil {
		return async.FutureValue(async.Void{})
	}
	return t.keyHandler.HandleKeyDownEvent(key)
}

// HandleKeyUpEvent gives text input a chance to respond to an unhandled
// key up event.
func (t *TestTextInput) HandleKeyUpEvent(key services.LogicalKeyboardKey) async.Future[async.Void] {
	if t.keyHandler == nil {
		return async.FutureValue(async.Void{})
	}
	return t.keyHandler.HandleKeyUpEvent(key)
}
