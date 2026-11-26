package widgets_test

// Own tests of the ported parts of shortcuts.dart, actions.dart,
// focus_manager.dart (KeyEventResult) and default_text_editing_shortcuts.dart.
// Flutter's shortcuts_test.dart, actions_test.dart and
// default_text_editing_shortcuts_test.dart pump Shortcuts/Actions/Focus
// widgets, which are not ported yet; the editing behavior is checked
// against Flutter by the editing and editing_actions difftests.

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func TestCombineKeyEventResults(t *testing.T) {
	h, i, s := KeyEventResultHandled, KeyEventResultIgnored, KeyEventResultSkipRemainingHandlers
	for _, c := range []struct {
		in   []KeyEventResult
		want KeyEventResult
	}{
		{nil, i}, {[]KeyEventResult{i, i}, i}, {[]KeyEventResult{i, s}, s}, {[]KeyEventResult{s, h}, h},
		{[]KeyEventResult{h, s}, h}, {[]KeyEventResult{s, s}, s},
	} {
		if got := CombineKeyEventResults(c.in); got != c.want {
			t.Errorf("%v: %v, want %v", c.in, got, c.want)
		}
	}
	if KeyEventResultSkipRemainingHandlers.String() != "KeyEventResult.skipRemainingHandlers" ||
		LockStateUnlocked.String() != "LockState.unlocked" {
		t.Fatal("enum names")
	}
}

func TestSingleActivatorAsserts(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a modifier trigger should assert")
		}
	}()
	NewSingleActivator(services.LogicalKeyboardKeyShiftRight)
}

func TestSingleActivatorDescribe(t *testing.T) {
	a := NewSingleActivator(services.LogicalKeyboardKeyKeyA, SingleActivatorOptions{Control: true, Shift: true, Alt: true, Meta: true})
	if got := a.DebugDescribeKeys(); got != "Control + Alt + Meta + Shift + Key A" {
		t.Fatal(got)
	}
	f := false
	if NewSingleActivator(services.LogicalKeyboardKeyKeyA, SingleActivatorOptions{IncludeRepeats: &f}).IncludeRepeats() ||
		!NewSingleActivator(services.LogicalKeyboardKeyKeyA).IncludeRepeats() {
		t.Fatal("includeRepeats")
	}
	c := CharacterActivator{Character: "?", Alt: true, Control: true, Meta: true}
	if got := c.DebugDescribeKeys(); got != "Alt + Control + Meta + '?'" || c.Triggers() != nil {
		t.Fatal(got)
	}
}

func TestShortcutMapWith(t *testing.T) {
	a := NewSingleActivator(services.LogicalKeyboardKeyKeyA)
	b := NewSingleActivator(services.LogicalKeyboardKeyKeyB)
	m := ShortcutMap{}.With(ShortcutEntry{Activator: a, Intent: DoNothingIntent{}}, ShortcutEntry{Activator: b, Intent: DismissIntent{}})
	// An equal const key replaces the intent in place.
	m = m.With(ShortcutEntry{Activator: a, Intent: DismissIntent{}})
	if len(m) != 2 || m[0].Intent != (DismissIntent{}) || m[0].Activator != a {
		t.Fatalf("const replace: %+v", m)
	}
	// A non-const key (a new instance) is always added, and is never
	// replaced.
	m = m.With(ShortcutEntry{Activator: a, Intent: DoNothingIntent{}, NonConst: true})
	m = m.With(ShortcutEntry{Activator: a, Intent: TransposeCharactersIntent{}})
	if len(m) != 3 || m[2].Intent != (DoNothingIntent{}) || m[0].Intent != (TransposeCharactersIntent{}) {
		t.Fatalf("non-const: %+v", m)
	}
}

func keyDown(key services.LogicalKeyboardKey) services.KeyEvent {
	return services.NewKeyDownEvent(services.PhysicalKeyboardKeyKeyA, key, 0)
}

func TestShortcutManager(t *testing.T) {
	tester := flutter_test.New(t)
	_ = tester
	a := NewSingleActivator(services.LogicalKeyboardKeyKeyA)
	ctrlA := NewSingleActivator(services.LogicalKeyboardKeyKeyA, SingleActivatorOptions{Control: true})
	char := CharacterActivator{Character: "a"}
	m := &ShortcutManager{Shortcuts: ShortcutMap{
		{Activator: char, Intent: DismissIntent{}},
		{Activator: ctrlA, Intent: SelectAllTextIntent{}},
		{Activator: a, Intent: DoNothingIntent{}},
	}}
	state := services.HardwareKeyboardInstance()
	// Candidates with the trigger come before those without triggers.
	if got := m.Find(keyDown(services.LogicalKeyboardKeyKeyA), state); got != (DoNothingIntent{}) {
		t.Fatalf("find: %#v", got)
	}
	ch := "a"
	ev := services.NewKeyDownEvent(services.PhysicalKeyboardKeyKeyB, services.LogicalKeyboardKeyKeyB, 0, services.KeyEventOptions{Character: &ch})
	if got := m.Find(ev, state); got != (DismissIntent{}) {
		t.Fatalf("character: %#v", got)
	}
	// An event both activators accept: the trigger's candidate wins.
	evA := services.NewKeyDownEvent(services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, 0, services.KeyEventOptions{Character: &ch})
	if got := m.Find(evA, state); got != (DoNothingIntent{}) {
		t.Fatalf("trigger first: %#v", got)
	}
	if got := m.Find(services.NewKeyUpEvent(services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, 0), state); got != nil {
		t.Fatalf("key up: %#v", got)
	}
	rep := services.NewKeyRepeatEvent(services.PhysicalKeyboardKeyKeyA, services.LogicalKeyboardKeyKeyA, 0)
	if got := m.Find(rep, state); got != (DoNothingIntent{}) {
		t.Fatalf("repeat: %#v", got)
	}
	f := false
	m2 := &ShortcutManager{Shortcuts: ShortcutMap{{Activator: NewSingleActivator(services.LogicalKeyboardKeyKeyA, SingleActivatorOptions{IncludeRepeats: &f}), Intent: DoNothingIntent{}}}}
	if got := m2.Find(rep, state); got != nil {
		t.Fatalf("excluded repeat: %#v", got)
	}

	// handleKeypress: an enabled action's result; a disabled or missing one
	// is ignored (skipRemainingHandlers when modal).
	invoked := 0
	action := NewCallbackAction(func(DoNothingIntent) any { invoked++; return nil })
	find := func(i Intent) Action {
		if _, ok := i.(DoNothingIntent); ok {
			return action
		}
		return nil
	}
	if r := m.HandleKeypress(keyDown(services.LogicalKeyboardKeyKeyA), find); r != KeyEventResultHandled || invoked != 1 {
		t.Fatalf("handled: %v %d", r, invoked)
	}
	if r := m.HandleKeypress(keyDown(services.LogicalKeyboardKeyKeyC), find); r != KeyEventResultIgnored {
		t.Fatalf("no intent: %v", r)
	}
	if r := m.HandleKeypress(ev, find); r != KeyEventResultIgnored {
		t.Fatalf("no action: %v", r)
	}
	disabled := &disabledAction{}
	disabled.SetSelf(disabled)
	if r := m.HandleKeypress(keyDown(services.LogicalKeyboardKeyKeyA), func(Intent) Action { return disabled }); r != KeyEventResultIgnored || disabled.invoked {
		t.Fatalf("disabled action: %v invoked=%v", r, disabled.invoked)
	}
	m.Modal = true
	if r := m.HandleKeypress(ev, find); r != KeyEventResultSkipRemainingHandlers {
		t.Fatalf("modal: %v", r)
	}
	noConsume := false
	stop := NewDoNothingAction(DoNothingActionOptions{ConsumesKey: &noConsume})
	if r := m.HandleKeypress(keyDown(services.LogicalKeyboardKeyKeyA), func(Intent) Action { return stop }); r != KeyEventResultSkipRemainingHandlers {
		t.Fatalf("do nothing, not consumed: %v", r)
	}
	if NewDoNothingAction().ToKeyEventResult(DoNothingIntent{}, nil) != KeyEventResultHandled {
		t.Fatal("do nothing consumes by default")
	}
}

func TestSingleActivatorModifiersAndNumLock(t *testing.T) {
	tester := flutter_test.New(t)
	ctrlA := NewSingleActivator(services.LogicalKeyboardKeyKeyA, SingleActivatorOptions{Control: true})
	plainA := NewSingleActivator(services.LogicalKeyboardKeyKeyA)
	state := services.HardwareKeyboardInstance()
	ev := keyDown(services.LogicalKeyboardKeyKeyA)
	if ctrlA.Accepts(ev, state) || !plainA.Accepts(ev, state) {
		t.Fatal("no modifiers")
	}
	tester.SendKeyDownEvent(services.LogicalKeyboardKeyControlRight)
	if !ctrlA.Accepts(ev, state) || plainA.Accepts(ev, state) {
		t.Fatal("control right is control")
	}
	tester.SendKeyDownEvent(services.LogicalKeyboardKeyShiftLeft)
	if ctrlA.Accepts(ev, state) {
		t.Fatal("an extra modifier must not be held")
	}
	tester.SendKeyUpEvent(services.LogicalKeyboardKeyShiftLeft)
	tester.SendKeyUpEvent(services.LogicalKeyboardKeyControlRight)
	for _, mod := range []struct {
		key services.LogicalKeyboardKey
		opt SingleActivatorOptions
	}{
		{services.LogicalKeyboardKeyAltLeft, SingleActivatorOptions{Alt: true}},
		{services.LogicalKeyboardKeyMetaRight, SingleActivatorOptions{Meta: true}},
		{services.LogicalKeyboardKeyShiftRight, SingleActivatorOptions{Shift: true}},
	} {
		a := NewSingleActivator(services.LogicalKeyboardKeyKeyA, mod.opt)
		tester.SendKeyDownEvent(mod.key)
		if !a.Accepts(ev, state) || plainA.Accepts(ev, state) {
			t.Fatalf("%v", mod.key)
		}
		tester.SendKeyUpEvent(mod.key)
		if a.Accepts(ev, state) {
			t.Fatalf("%v released", mod.key)
		}
	}
	locked := NewSingleActivator(services.LogicalKeyboardKeyNumpad1, SingleActivatorOptions{NumLock: LockStateLocked})
	unlocked := NewSingleActivator(services.LogicalKeyboardKeyNumpad1, SingleActivatorOptions{NumLock: LockStateUnlocked})
	num := keyDown(services.LogicalKeyboardKeyNumpad1)
	if locked.Accepts(num, state) || !unlocked.Accepts(num, state) {
		t.Fatal("num lock off")
	}
	tester.SendKeyEvent(services.LogicalKeyboardKeyNumLock)
	if !locked.Accepts(num, state) || unlocked.Accepts(num, state) {
		t.Fatal("num lock on")
	}
	tester.SendKeyEvent(services.LogicalKeyboardKeyNumLock)
}

func TestDefaultTextEditingShortcutsMaps(t *testing.T) {
	find := func(m ShortcutMap, a SingleActivator) Intent {
		for _, e := range m {
			if e.Activator == a {
				return e.Intent
			}
		}
		return nil
	}
	left := services.LogicalKeyboardKeyArrowLeft
	for _, c := range []struct {
		platform foundation.TargetPlatform
		act      SingleActivator
		want     Intent
		disabled bool
	}{
		{foundation.TargetPlatformAndroid, NewSingleActivator(left, SingleActivatorOptions{Control: true}), NewExtendSelectionToNextWordBoundaryIntent(false, true), false},
		{foundation.TargetPlatformMacOS, NewSingleActivator(left, SingleActivatorOptions{Alt: true}), NewExtendSelectionToNextWordBoundaryIntent(false, true), true},
		{foundation.TargetPlatformMacOS, NewSingleActivator(services.LogicalKeyboardKeyKeyT, SingleActivatorOptions{Control: true}), TransposeCharactersIntent{}, false},
		{foundation.TargetPlatformLinux, NewSingleActivator(services.LogicalKeyboardKeyHome), NewExtendSelectionToLineBreakIntent(false, true), false},
		{foundation.TargetPlatformWindows, NewSingleActivator(services.LogicalKeyboardKeyHome),
			NewExtendSelectionToLineBreakIntent(false, true, ExtendSelectionToLineBreakIntentOptions{ContinuesAtWrap: true}), false},
		{foundation.TargetPlatformFuchsia, NewSingleActivator(services.LogicalKeyboardKeyEnd, SingleActivatorOptions{Control: true}),
			NewExtendSelectionToDocumentBoundaryIntent(true, true), false},
		{foundation.TargetPlatformIOS, NewSingleActivator(services.LogicalKeyboardKeyBackspace), NewDeleteCharacterIntent(false), true},
		{foundation.TargetPlatformAndroid, NewSingleActivator(services.LogicalKeyboardKeySpace), DoNothingAndStopPropagationTextIntent{}, false},
		{foundation.TargetPlatformWindows, NewSingleActivator(services.LogicalKeyboardKeyEnter), DoNothingAndStopPropagationTextIntent{}, false},
	} {
		p := c.platform
		foundation.SetDebugDefaultTargetPlatformOverride(&p)
		if got := find(DefaultTextEditingShortcutsShortcuts(), c.act); got != c.want {
			t.Errorf("%v %s: %#v, want %#v", p, c.act.DebugDescribeKeys(), got, c.want)
		}
		disabling := DefaultTextEditingShortcutsDisablingShortcuts()
		if got := find(disabling, c.act); (got == DoNothingAndStopPropagationTextIntent{}) != c.disabled {
			t.Errorf("%v %s disabled: %#v", p, c.act.DebugDescribeKeys(), got)
		}
		if n := len(DefaultTextEditingShortcutsHandlers(func(Intent) Action { return nil })); (n == 2) != (disabling != nil) {
			t.Errorf("%v: %d handlers", p, n)
		}
	}
	foundation.SetDebugDefaultTargetPlatformOverride(nil)
	if IntentForMacOSSelector("moveWordLeftAndModifySelection:") != NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(false) ||
		IntentForMacOSSelector("cancelOperation:") != (DismissIntent{}) || IntentForMacOSSelector("noSuchSelector:") != nil ||
		IntentForMacOSSelector("transpose:") != (TransposeCharactersIntent{}) {
		t.Fatal("macOS selectors")
	}
}

func TestEnterTextUTF16Caret(t *testing.T) {
	c := NewTextEditingController("")
	tester := flutter_test.New(t)
	tester.PumpWidget(TextField{Controller: c, Autofocus: true})
	tester.EnterText("é😀")
	if got := c.Value().Selection; !got.Equal(services.NewTextSelectionCollapsed(3)) {
		t.Fatalf("caret after é😀: %v", got)
	}
}

type disabledAction struct {
	ActionBase
	invoked bool
}

func (a *disabledAction) IsEnabled(Intent) bool { return false }
func (a *disabledAction) Invoke(Intent) any     { a.invoked = true; return nil }
