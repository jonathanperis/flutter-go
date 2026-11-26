package widgets_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func caretDrawn(t *flutter_test.Tester) bool {
	for _, op := range t.Canvas.Ops {
		if strings.HasPrefix(op, "rect(") && strings.Contains(op, ", 2, ") { // 2px-wide caret
			return true
		}
	}
	return false
}

func TestCaretBlinksWithTimersNotFrames(t *testing.T) {
	tester := flutter_test.New(t)
	tester.PumpWidget(TextField{Autofocus: true})
	if !caretDrawn(tester) {
		t.Fatal("caret should be visible right after focus")
	}
	if tester.Binding().HasScheduledFrame() {
		t.Fatal("blinking must not keep frames scheduled (Flutter uses a Timer)")
	}
	tester.Pump(499 * time.Millisecond)
	if !caretDrawn(tester) {
		t.Fatal("caret hidden too early")
	}
	tester.Pump(1 * time.Millisecond)
	if caretDrawn(tester) {
		t.Fatal("caret should hide at 500ms")
	}
	tester.Pump(500 * time.Millisecond)
	if !caretDrawn(tester) {
		t.Fatal("caret should reappear at 1000ms")
	}
	tester.Pump(400 * time.Millisecond)
	tester.EnterText("a") // typing shows the caret and restarts the timer
	tester.Pump(499 * time.Millisecond)
	if !caretDrawn(tester) {
		t.Fatal("typing should restart the blink period")
	}
}

func TestTapPlacesCaret(t *testing.T) {
	c := NewTextEditingController("hello world")
	tester := flutter_test.New(t)
	tester.PumpWidget(Align{Alignment: painting.AlignmentTopLeft, Child: SizedBox{Width: F(400), Child: TextField{Controller: c}}})
	// Test font: 14px per glyph. x=45 is nearest the boundary after "hel" (42).
	tester.TapAt(45, 5)
	tester.Pump()
	if got := c.Value().Selection; !got.Equal(services.NewTextSelectionCollapsed(3)) {
		t.Fatalf("selection = %v, want caret at 3", got)
	}
}

func TestClipboardAndSubmit(t *testing.T) {
	var submitted string
	c := NewTextEditingController("")
	tester := flutter_test.New(t)
	var clip *string
	messenger := tester.Binding().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
	messenger.SetMockMethodCallHandler(services.SystemChannels.Platform, func(call services.MethodCall) async.Future[any] {
		switch call.Method {
		case "Clipboard.setData":
			s := call.Arguments.(*dart.LinkedMap).Lookup("text").(string)
			clip = &s
		case "Clipboard.getData":
			if clip != nil {
				return async.FutureValue[any](map[string]any{"text": *clip})
			}
		}
		return async.FutureValue[any](nil)
	})
	defer messenger.SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
	tester.PumpWidget(TextField{Controller: c, Autofocus: true, OnSubmitted: func(s string) { submitted = s }})
	tester.EnterText("copy me")
	chord := func(key services.LogicalKeyboardKey) {
		tester.SendKeyDownEvent(services.LogicalKeyboardKeyControlLeft)
		tester.SendKeyEvent(key)
		tester.SendKeyUpEvent(services.LogicalKeyboardKeyControlLeft)
		tester.Pump()
	}
	chord(services.LogicalKeyboardKeyKeyA)
	chord(services.LogicalKeyboardKeyKeyX)
	if c.Text() != "" || clip == nil || *clip != "copy me" {
		t.Fatalf("cut: text %q clipboard %v", c.Text(), clip)
	}
	chord(services.LogicalKeyboardKeyKeyV)
	chord(services.LogicalKeyboardKeyKeyV)
	// Enter is the platform's: its text input sends the action.
	if tester.SendKeyEvent(services.LogicalKeyboardKeyEnter) {
		t.Fatal("Enter should not be handled by the framework (DoNothingAndStopPropagationTextIntent)")
	}
	tester.Binding().TestTextInput().ReceiveAction(services.TextInputActionDone)
	tester.Pump()
	if submitted != "copy mecopy me" {
		t.Fatalf("submitted %q", submitted)
	}
}

func TestFocusMovesBetweenFields(t *testing.T) {
	a, b := NewTextEditingController(""), NewTextEditingController("")
	tester := flutter_test.New(t)
	tester.PumpWidget(Column{Children: []Widget{
		SizedBox{Height: F(40), Child: TextField{Controller: a}},
		SizedBox{Height: F(40), Child: TextField{Controller: b}},
	}})
	tester.TapAt(10, 10)
	tester.EnterText("first")
	tester.TapAt(10, 50)
	tester.EnterText("second")
	if a.Text() != "first" || b.Text() != "second" {
		t.Fatalf("a=%q b=%q", a.Text(), b.Text())
	}
}

func TestMacOSShortcuts(t *testing.T) {
	mac := foundation.TargetPlatformMacOS
	foundation.SetDebugDefaultTargetPlatformOverride(&mac)
	defer foundation.SetDebugDefaultTargetPlatformOverride(nil)
	c := NewTextEditingController("")
	tester := flutter_test.New(t)
	tester.Owner.Platform = foundation.TargetPlatformMacOS
	tester.PumpWidget(TextField{Controller: c, Autofocus: true})
	tester.EnterText("hello brave world")
	with := func(mod, key services.LogicalKeyboardKey) {
		tester.SendKeyDownEvent(mod)
		tester.SendKeyEvent(key)
		tester.SendKeyUpEvent(mod)
		tester.Pump()
	}
	with(services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyArrowLeft) // word left on macOS
	if !c.Value().Selection.Equal(services.NewTextSelectionCollapsed(12)) {
		t.Fatalf("alt+left: %v", c.Value().Selection)
	}
	with(services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyArrowLeft) // line start on macOS
	if !c.Value().Selection.Equal(services.NewTextSelectionCollapsed(0)) {
		t.Fatalf("cmd+left: %v", c.Value().Selection)
	}
	tester.SendKeyEvent(services.LogicalKeyboardKeyEnd) // scrolls on macOS; caret stays
	tester.Pump()
	if !c.Value().Selection.Equal(services.NewTextSelectionCollapsed(0)) {
		t.Fatalf("end moved the caret on macOS: %v", c.Value().Selection)
	}
}
