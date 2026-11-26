package flutter_test_test

// Tests of TestTextInput. Flutter's own
// (packages/flutter_test/test/test_text_input_test.dart) are skipped: all
// pump a TextField/EditableText (widgets not ported), and the macOS key
// handler tests need the keyboard library. These are flutter-go's, through
// a TextInputClient attached with services.TextInputAttach.

import (
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type client struct {
	services.TextInputClientMixin
	value   services.TextEditingValue
	actions []services.TextInputAction
	closed  bool
}

func (c *client) CurrentTextEditingValue() *services.TextEditingValue  { return &c.value }
func (c *client) CurrentAutofillScope() services.AutofillScope         { return nil }
func (c *client) UpdateEditingValue(v services.TextEditingValue)       { c.value = v }
func (c *client) PerformAction(a services.TextInputAction)             { c.actions = append(c.actions, a) }
func (c *client) PerformPrivateCommand(string, *dart.LinkedMap)        {}
func (c *client) UpdateFloatingCursor(services.RawFloatingCursorPoint) {}
func (c *client) ShowAutocorrectionPromptRect(int, int)                {}
func (c *client) ConnectionClosed()                                    { c.closed = true }

func TestTestTextInputRecordsTheFrameworksCalls(t *testing.T) {
	tester := flutter_test.New(t)
	input := tester.Binding().TestTextInput()
	if !input.IsRegistered() || input.HasAnyClients() || input.IsVisible() {
		t.Fatal("initial state")
	}
	services.TextInputConnectionDebugResetID()
	c := &client{}
	connection := services.TextInputAttach(c, services.NewTextInputConfiguration())
	tester.Pump()
	if !input.HasAnyClients() || input.SetClientArgs.Lookup("inputAction") != "TextInputAction.done" {
		t.Fatalf("setClient: %v", input.SetClientArgs)
	}
	connection.Show()
	sel := services.NewTextSelectionCollapsed(2)
	connection.SetEditingState(services.NewTextEditingValue(services.TextEditingValueOptions{Text: "ab", Selection: &sel}))
	tester.Pump()
	if !input.IsVisible() || input.EditingState.Lookup("text") != "ab" || input.EditingState.Lookup("selectionBase") != 2 {
		t.Fatalf("show/setEditingState: %v %v", input.IsVisible(), input.EditingState)
	}

	// The platform's side: typing, actions, closing.
	input.EnterText("héllo😀")
	tester.Pump()
	want := services.NewTextSelectionCollapsed(7)
	if c.value.Text != "héllo😀" || !c.value.Selection.Equal(want) {
		t.Fatalf("enterText: %v", c.value)
	}
	input.ReceiveAction(services.TextInputActionGo)
	tester.Pump()
	if !reflect.DeepEqual(c.actions, []services.TextInputAction{services.TextInputActionGo}) {
		t.Fatalf("actions %v", c.actions)
	}
	input.CloseConnection()
	tester.Pump()
	if !c.closed {
		t.Fatal("not closed")
	}

	connection.Close()
	if input.IsVisible() {
		t.Fatal("visible after clearClient")
	}
	tester.Pump()
	var methods []string
	for _, call := range input.Log {
		methods = append(methods, call.Method)
	}
	wantMethods := []string{"TextInput.setClient", "TextInput.show", "TextInput.setEditingState", "TextInput.clearClient", "TextInput.hide"}
	if !reflect.DeepEqual(methods, wantMethods) {
		t.Fatalf("log %v", methods)
	}
	if input.HasAnyClients() || input.IsVisible() {
		t.Fatal("cleared state")
	}
	input.Reset()
	if input.Log != nil || input.SetClientArgs != nil || input.EditingState != nil {
		t.Fatal("reset")
	}
}

type scribble struct {
	id      string
	bounds  ui.Rect
	focused bool
}

func (s *scribble) ElementIdentifier() string       { return s.id }
func (s *scribble) OnScribbleFocus(ui.Offset)       { s.focused = true }
func (s *scribble) IsInScribbleRect(r ui.Rect) bool { return s.bounds.Overlaps(r) }
func (s *scribble) Bounds() ui.Rect                 { return s.bounds }

func TestTestTextInputScribble(t *testing.T) {
	tester := flutter_test.New(t)
	input := tester.Binding().TestTextInput()
	services.TextInputConnectionDebugResetID()
	connection := services.TextInputAttach(&client{}, services.NewTextInputConfiguration())
	defer connection.Close()
	s := &scribble{id: "a", bounds: ui.NewRectFromLTWH(0, 0, 10, 10)}
	services.TextInputRegisterScribbleElement("a", s)
	defer services.TextInputUnregisterScribbleElement("a")

	input.StartScribbleInteraction()
	tester.Pump()
	if !connection.ScribbleInProgress() {
		t.Fatal("not in progress")
	}
	input.FinishScribbleInteraction()
	tester.Pump()
	if connection.ScribbleInProgress() {
		t.Fatal("in progress")
	}
	input.ScribbleFocusElement("a", ui.NewOffset(1, 1))
	tester.Pump()
	if !s.focused {
		t.Fatal("not focused")
	}
	f := input.ScribbleRequestElementsInRect(ui.NewRectFromLTWH(5, 5, 10, 10))
	tester.Pump()
	var got [][]any
	async.OnComplete(f, func(v [][]any) { got = v }, func(err error) { t.Fatal(err) })
	tester.Pump()
	if !reflect.DeepEqual(got, [][]any{{"a", 0.0, 0.0, 10.0, 10.0}}) {
		t.Fatalf("elements %v", got)
	}
}
