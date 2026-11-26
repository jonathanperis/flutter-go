package widgets_test

import (
	"strings"
	"testing"
	"time"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/src/textedit"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
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
	tester.PumpWidget(Align{Alignment: r.TopLeft, Child: SizedBox{Width: F(400), Child: TextField{Controller: c}}})
	// Test font: 14px per glyph. x=45 is nearest the boundary after "hel" (42).
	tester.TapAt(45, 5)
	tester.Pump()
	if got := c.Value().Selection; got != textedit.Collapsed(3) {
		t.Fatalf("selection = %+v, want caret at 3", got)
	}
}

func TestClipboardAndSubmit(t *testing.T) {
	var submitted string
	c := NewTextEditingController("")
	tester := flutter_test.New(t)
	tester.PumpWidget(TextField{Controller: c, Autofocus: true, OnSubmitted: func(s string) { submitted = s }})
	tester.EnterText("copy me")
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyA, Ctrl: true})
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyX, Ctrl: true})
	if c.Text() != "" || tester.Owner.Clipboard.Get() != "copy me" {
		t.Fatalf("cut: text %q clipboard %q", c.Text(), tester.Owner.Clipboard.Get())
	}
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyV, Ctrl: true})
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyV, Ctrl: true})
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyEnter})
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
	c := NewTextEditingController("")
	tester := flutter_test.New(t)
	tester.Owner.Platform = foundation.TargetPlatformMacOS
	tester.PumpWidget(TextField{Controller: c, Autofocus: true})
	tester.EnterText("hello brave world")
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyLeft, Alt: true}) // word left on macOS
	if c.Value().Selection != textedit.Collapsed(12) {
		t.Fatalf("alt+left: %+v", c.Value().Selection)
	}
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyLeft, Meta: true}) // line start on macOS
	if c.Value().Selection != textedit.Collapsed(0) {
		t.Fatalf("cmd+left: %+v", c.Value().Selection)
	}
	tester.SendKey(textedit.KeyEvent{Key: textedit.KeyEnd}) // scrolls on macOS; caret stays
	if c.Value().Selection != textedit.Collapsed(0) {
		t.Fatalf("end moved the caret on macOS: %+v", c.Value().Selection)
	}
}
