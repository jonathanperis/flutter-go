package widgets

// Port of packages/flutter/lib/src/widgets/default_text_editing_shortcuts.dart
// (Flutter 3.50.1): the shortcut maps of every platform, the web/macOS/iOS
// disabling maps and intentForMacOSSelector. ScrollIntent (scrollable
// helpers, which need painting's AxisDirection), DirectionalFocusIntent,
// NextFocusIntent and PreviousFocusIntent are not ported yet: their entries
// map to scrollPageIntent / focusIntent placeholders that no action handles
// (so they are ignored, as EditableText ignores them for a single-line
// field without a focus traversal).
//
// The DefaultTextEditingShortcuts widget builds two nested Shortcuts; the
// interim editing bridge applies them through
// DefaultTextEditingShortcutsHandlers (the disabling map's handler first,
// as the inner Shortcuts' focus node sees the event first).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// scrollPageIntent stands for ScrollIntent(direction: up/down, type: page)
// until ScrollIntent is ported.
type scrollPageIntent struct {
	IntentBase
	down bool
}

// focusIntent stands for NextFocusIntent/PreviousFocusIntent until focus
// traversal is ported.
type focusIntent struct {
	IntentBase
	next bool
}

type mods struct {
	control, shift, alt, meta bool
	numLock                   LockState
}

func sa(key services.LogicalKeyboardKey, m mods) SingleActivator {
	return NewSingleActivator(key, SingleActivatorOptions{Control: m.control, Shift: m.shift, Alt: m.alt, Meta: m.meta, NumLock: m.numLock})
}

func entry(key services.LogicalKeyboardKey, m mods, intent Intent) ShortcutEntry {
	return ShortcutEntry{Activator: sa(key, m), Intent: intent}
}

// nonConst is an entry built in a collection-for (a new SingleActivator).
func nonConst(key services.LogicalKeyboardKey, m mods, intent Intent) ShortcutEntry {
	return ShortcutEntry{Activator: sa(key, m), Intent: intent, NonConst: true}
}

type lk = services.LogicalKeyboardKey

var (
	kBackspace     = services.LogicalKeyboardKeyBackspace
	kDelete        = services.LogicalKeyboardKeyDelete
	kArrowLeft     = services.LogicalKeyboardKeyArrowLeft
	kArrowRight    = services.LogicalKeyboardKeyArrowRight
	kArrowUp       = services.LogicalKeyboardKeyArrowUp
	kArrowDown     = services.LogicalKeyboardKeyArrowDown
	kPageUp        = services.LogicalKeyboardKeyPageUp
	kPageDown      = services.LogicalKeyboardKeyPageDown
	kHome          = services.LogicalKeyboardKeyHome
	kEnd           = services.LogicalKeyboardKeyEnd
	kSpace         = services.LogicalKeyboardKeySpace
	kEnter         = services.LogicalKeyboardKeyEnter
	kInsert        = services.LogicalKeyboardKeyInsert
	kEscape        = services.LogicalKeyboardKeyEscape
	kTab           = services.LogicalKeyboardKeyTab
	kNumpadDecimal = services.LogicalKeyboardKeyNumpadDecimal
)

var (
	none         = mods{}
	shift        = mods{shift: true}
	alt          = mods{alt: true}
	control      = mods{control: true}
	meta         = mods{meta: true}
	shiftAlt     = mods{shift: true, alt: true}
	shiftControl = mods{shift: true, control: true}
	shiftMeta    = mods{shift: true, meta: true}
)

func withShift(m mods, pressShift bool) mods {
	m.shift = pressShift
	return m
}

var stop = DoNothingAndStopPropagationTextIntent{}

func cut() CopySelectionTextIntent {
	return NewCopySelectionTextIntentCut(services.SelectionChangedCauseKeyboard)
}

func paste() PasteTextIntent { return PasteTextIntent{Cause: services.SelectionChangedCauseKeyboard} }

func selectAll() SelectAllTextIntent {
	return SelectAllTextIntent{Cause: services.SelectionChangedCauseKeyboard}
}

func undo() UndoTextIntent { return UndoTextIntent{Cause: services.SelectionChangedCauseKeyboard} }
func redo() RedoTextIntent { return RedoTextIntent{Cause: services.SelectionChangedCauseKeyboard} }

var commonShortcuts = func() ShortcutMap {
	var m ShortcutMap
	for _, pressShift := range []bool{true, false} {
		m = m.With(
			nonConst(kBackspace, withShift(none, pressShift), NewDeleteCharacterIntent(false)),
			nonConst(kBackspace, withShift(control, pressShift), NewDeleteToNextWordBoundaryIntent(false)),
			nonConst(kBackspace, withShift(alt, pressShift), NewDeleteToLineBreakIntent(false)),
			nonConst(kDelete, withShift(control, pressShift), NewDeleteToNextWordBoundaryIntent(true)),
			nonConst(kDelete, withShift(alt, pressShift), NewDeleteToLineBreakIntent(true)),
		)
	}
	return m.With(
		entry(kDelete, none, NewDeleteCharacterIntent(true)),
		entry(kArrowLeft, none, NewExtendSelectionByCharacterIntent(false, true)),
		entry(kArrowRight, none, NewExtendSelectionByCharacterIntent(true, true)),
		entry(kArrowUp, none, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		entry(kArrowDown, none, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),
		entry(kArrowLeft, shift, NewExtendSelectionByCharacterIntent(false, false)),
		entry(kArrowRight, shift, NewExtendSelectionByCharacterIntent(true, false)),
		entry(kArrowUp, shift, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		entry(kArrowDown, shift, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),
		entry(kArrowLeft, alt, NewExtendSelectionToLineBreakIntent(false, true)),
		entry(kArrowRight, alt, NewExtendSelectionToLineBreakIntent(true, true)),
		entry(kArrowUp, alt, NewExtendSelectionToDocumentBoundaryIntent(false, true)),
		entry(kArrowDown, alt, NewExtendSelectionToDocumentBoundaryIntent(true, true)),
		entry(kArrowLeft, shiftAlt, NewExtendSelectionToLineBreakIntent(false, false)),
		entry(kArrowRight, shiftAlt, NewExtendSelectionToLineBreakIntent(true, false)),
		entry(kArrowUp, shiftAlt, NewExtendSelectionToDocumentBoundaryIntent(false, false)),
		entry(kArrowDown, shiftAlt, NewExtendSelectionToDocumentBoundaryIntent(true, false)),
		entry(kArrowLeft, control, NewExtendSelectionToNextWordBoundaryIntent(false, true)),
		entry(kArrowRight, control, NewExtendSelectionToNextWordBoundaryIntent(true, true)),
		entry(kArrowLeft, shiftControl, NewExtendSelectionToNextWordBoundaryIntent(false, false)),
		entry(kArrowRight, shiftControl, NewExtendSelectionToNextWordBoundaryIntent(true, false)),
		entry(kArrowUp, shiftControl, NewExtendSelectionToNextParagraphBoundaryIntent(false, false)),
		entry(kArrowDown, shiftControl, NewExtendSelectionToNextParagraphBoundaryIntent(true, false)),
		entry(kPageUp, none, NewExtendSelectionVerticallyToAdjacentPageIntent(false, true)),
		entry(kPageDown, none, NewExtendSelectionVerticallyToAdjacentPageIntent(true, true)),
		entry(kPageUp, shift, NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)),
		entry(kPageDown, shift, NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)),
	)
}()

var clipboardShortcuts = ShortcutMap{}.With(
	entry(services.LogicalKeyboardKeyKeyX, control, cut()),
	entry(services.LogicalKeyboardKeyKeyC, control, CopySelectionTextIntentCopy),
	entry(services.LogicalKeyboardKeyKeyV, control, paste()),
	entry(kDelete, shift, cut()),
	entry(kInsert, control, CopySelectionTextIntentCopy),
	entry(kInsert, shift, paste()),
	entry(services.LogicalKeyboardKeyKeyA, control, selectAll()),
	entry(services.LogicalKeyboardKeyKeyZ, control, undo()),
	entry(services.LogicalKeyboardKeyKeyZ, shiftControl, redo()),
	entry(kSpace, none, stop),
	entry(kEnter, none, stop),
)

func homeEndShortcuts(continuesAtWrap bool) []ShortcutEntry {
	o := ExtendSelectionToLineBreakIntentOptions{ContinuesAtWrap: continuesAtWrap}
	return []ShortcutEntry{
		entry(kHome, none, NewExtendSelectionToLineBreakIntent(false, true, o)),
		entry(kEnd, none, NewExtendSelectionToLineBreakIntent(true, true, o)),
		entry(kHome, shift, NewExtendSelectionToLineBreakIntent(false, false, o)),
		entry(kEnd, shift, NewExtendSelectionToLineBreakIntent(true, false, o)),
		entry(kHome, control, NewExtendSelectionToDocumentBoundaryIntent(false, true)),
		entry(kEnd, control, NewExtendSelectionToDocumentBoundaryIntent(true, true)),
		entry(kHome, shiftControl, NewExtendSelectionToDocumentBoundaryIntent(false, false)),
		entry(kEnd, shiftControl, NewExtendSelectionToDocumentBoundaryIntent(true, false)),
	}
}

var androidShortcuts = commonShortcuts.With(clipboardShortcuts...).With(homeEndShortcuts(true)...)

var fuchsiaShortcuts = androidShortcuts

var linuxNumpadShortcuts = func() ShortcutMap {
	locked := func(m mods) mods { m.numLock = LockStateLocked; return m }
	unlocked := func(m mods) mods { m.numLock = LockStateUnlocked; return m }
	n := func(i int) lk {
		return []lk{services.LogicalKeyboardKeyNumpad0, services.LogicalKeyboardKeyNumpad1, services.LogicalKeyboardKeyNumpad2,
			services.LogicalKeyboardKeyNumpad3, services.LogicalKeyboardKeyNumpad4, services.LogicalKeyboardKeyNumpad5,
			services.LogicalKeyboardKeyNumpad6, services.LogicalKeyboardKeyNumpad7, services.LogicalKeyboardKeyNumpad8,
			services.LogicalKeyboardKeyNumpad9}[i]
	}
	return ShortcutMap{}.With(
		entry(n(6), locked(shift), NewExtendSelectionByCharacterIntent(true, false)),
		entry(n(4), locked(shift), NewExtendSelectionByCharacterIntent(false, false)),
		entry(n(8), locked(shift), NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		entry(n(2), locked(shift), NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),
		entry(n(6), locked(shiftControl), NewExtendSelectionToNextWordBoundaryIntent(true, false)),
		entry(n(4), locked(shiftControl), NewExtendSelectionToNextWordBoundaryIntent(false, false)),
		entry(n(8), locked(shiftControl), NewExtendSelectionToNextParagraphBoundaryIntent(false, false)),
		entry(n(2), locked(shiftControl), NewExtendSelectionToNextParagraphBoundaryIntent(true, false)),
		entry(n(9), locked(shift), NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)),
		entry(n(3), locked(shift), NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)),
		entry(n(7), locked(shift), NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		entry(n(1), locked(shift), NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),
		entry(kNumpadDecimal, locked(shift), NewDeleteCharacterIntent(true)),
		entry(kNumpadDecimal, locked(shiftControl), NewDeleteToNextWordBoundaryIntent(true)),
		entry(n(6), unlocked(none), NewExtendSelectionByCharacterIntent(true, true)),
		entry(n(4), unlocked(none), NewExtendSelectionByCharacterIntent(false, true)),
		entry(n(8), unlocked(none), NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		entry(n(2), unlocked(none), NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),
		entry(n(6), unlocked(control), NewExtendSelectionToNextWordBoundaryIntent(true, true)),
		entry(n(4), unlocked(control), NewExtendSelectionToNextWordBoundaryIntent(false, true)),
		entry(n(8), unlocked(control), NewExtendSelectionToNextParagraphBoundaryIntent(false, true)),
		entry(n(2), unlocked(control), NewExtendSelectionToNextParagraphBoundaryIntent(true, true)),
		entry(n(9), unlocked(none), NewExtendSelectionVerticallyToAdjacentPageIntent(false, true)),
		entry(n(3), unlocked(none), NewExtendSelectionVerticallyToAdjacentPageIntent(true, true)),
		entry(n(7), unlocked(none), NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		entry(n(1), unlocked(none), NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),
		entry(kNumpadDecimal, unlocked(none), NewDeleteCharacterIntent(true)),
		entry(kNumpadDecimal, unlocked(control), NewDeleteToNextWordBoundaryIntent(true)),
	)
}()

var linuxShortcuts = commonShortcuts.With(clipboardShortcuts...).With(linuxNumpadShortcuts...).With(homeEndShortcuts(false)...)

var macShortcuts = func() ShortcutMap {
	var m ShortcutMap
	for _, pressShift := range []bool{true, false} {
		m = m.With(
			nonConst(kBackspace, withShift(none, pressShift), NewDeleteCharacterIntent(false)),
			nonConst(kBackspace, withShift(alt, pressShift), NewDeleteToNextWordBoundaryIntent(false)),
			nonConst(kBackspace, withShift(meta, pressShift), NewDeleteToLineBreakIntent(false)),
			nonConst(kDelete, withShift(none, pressShift), NewDeleteCharacterIntent(true)),
			nonConst(kDelete, withShift(alt, pressShift), NewDeleteToNextWordBoundaryIntent(true)),
			nonConst(kDelete, withShift(meta, pressShift), NewDeleteToLineBreakIntent(true)),
		)
	}
	k := func(c string) lk {
		return map[string]lk{"x": services.LogicalKeyboardKeyKeyX, "c": services.LogicalKeyboardKeyKeyC,
			"v": services.LogicalKeyboardKeyKeyV, "a": services.LogicalKeyboardKeyKeyA, "z": services.LogicalKeyboardKeyKeyZ,
			"e": services.LogicalKeyboardKeyKeyE, "f": services.LogicalKeyboardKeyKeyF, "b": services.LogicalKeyboardKeyKeyB,
			"n": services.LogicalKeyboardKeyKeyN, "p": services.LogicalKeyboardKeyKeyP, "t": services.LogicalKeyboardKeyKeyT}[c]
	}
	return m.With(
		entry(kArrowLeft, none, NewExtendSelectionByCharacterIntent(false, true)),
		entry(kArrowRight, none, NewExtendSelectionByCharacterIntent(true, true)),
		entry(kArrowUp, none, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		entry(kArrowDown, none, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),
		entry(kArrowLeft, shift, NewExtendSelectionByCharacterIntent(false, false)),
		entry(kArrowRight, shift, NewExtendSelectionByCharacterIntent(true, false)),
		entry(kArrowUp, shift, NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)),
		entry(kArrowDown, shift, NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)),
		entry(kArrowLeft, alt, NewExtendSelectionToNextWordBoundaryIntent(false, true)),
		entry(kArrowRight, alt, NewExtendSelectionToNextWordBoundaryIntent(true, true)),
		entry(kArrowUp, alt, NewExtendSelectionToLineBreakIntent(false, true)),
		entry(kArrowDown, alt, NewExtendSelectionToLineBreakIntent(true, true)),
		entry(kArrowLeft, shiftAlt, NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(false)),
		entry(kArrowRight, shiftAlt, NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(true)),
		entry(kArrowUp, shiftAlt, NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(false)),
		entry(kArrowDown, shiftAlt, NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(true)),
		entry(kArrowLeft, meta, NewExtendSelectionToLineBreakIntent(false, true)),
		entry(kArrowRight, meta, NewExtendSelectionToLineBreakIntent(true, true)),
		entry(kArrowUp, meta, NewExtendSelectionToDocumentBoundaryIntent(false, true)),
		entry(kArrowDown, meta, NewExtendSelectionToDocumentBoundaryIntent(true, true)),
		entry(kArrowLeft, shiftMeta, NewExpandSelectionToLineBreakIntent(false)),
		entry(kArrowRight, shiftMeta, NewExpandSelectionToLineBreakIntent(true)),
		entry(kArrowUp, shiftMeta, NewExpandSelectionToDocumentBoundaryIntent(false)),
		entry(kArrowDown, shiftMeta, NewExpandSelectionToDocumentBoundaryIntent(true)),
		entry(k("t"), control, TransposeCharactersIntent{}),
		entry(kHome, none, NewScrollToDocumentBoundaryIntent(false)),
		entry(kEnd, none, NewScrollToDocumentBoundaryIntent(true)),
		entry(kHome, shift, NewExpandSelectionToDocumentBoundaryIntent(false)),
		entry(kEnd, shift, NewExpandSelectionToDocumentBoundaryIntent(true)),
		entry(kPageUp, none, scrollPageIntent{down: false}),
		entry(kPageDown, none, scrollPageIntent{down: true}),
		entry(kPageUp, shift, NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)),
		entry(kPageDown, shift, NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)),
		entry(k("x"), meta, cut()),
		entry(k("c"), meta, CopySelectionTextIntentCopy),
		entry(k("v"), meta, paste()),
		entry(k("a"), meta, selectAll()),
		entry(k("z"), meta, undo()),
		entry(k("z"), shiftMeta, redo()),
		entry(k("e"), control, NewExtendSelectionToLineBreakIntent(true, true)),
		entry(k("a"), control, NewExtendSelectionToLineBreakIntent(false, true)),
		entry(k("f"), control, NewExtendSelectionByCharacterIntent(true, true)),
		entry(k("b"), control, NewExtendSelectionByCharacterIntent(false, true)),
		entry(k("n"), control, NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)),
		entry(k("p"), control, NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)),
		entry(kSpace, none, stop),
		entry(kEnter, none, stop),
	)
}()

var iOSShortcuts = macShortcuts

var windowsShortcuts = commonShortcuts.With(clipboardShortcuts...).With(
	entry(kPageUp, none, NewExtendSelectionVerticallyToAdjacentPageIntent(false, true)),
	entry(kPageDown, none, NewExtendSelectionVerticallyToAdjacentPageIntent(true, true)),
).With(homeEndShortcuts(true)...)

var commonDisablingTextShortcuts = ShortcutMap{}.With(
	entry(kArrowDown, alt, stop),
	entry(kArrowLeft, alt, stop),
	entry(kArrowRight, alt, stop),
	entry(kArrowUp, alt, stop),
	entry(kArrowDown, meta, stop),
	entry(kArrowLeft, meta, stop),
	entry(kArrowRight, meta, stop),
	entry(kArrowUp, meta, stop),
	entry(kArrowDown, none, stop),
	entry(kArrowLeft, none, stop),
	entry(kArrowRight, none, stop),
	entry(kArrowUp, none, stop),
	entry(kArrowLeft, control, stop),
	entry(kArrowRight, control, stop),
	entry(kArrowLeft, shiftControl, stop),
	entry(kArrowRight, shiftControl, stop),
	entry(kSpace, none, stop),
	entry(kEnter, none, stop),
)

var webDisablingTextShortcuts = func() ShortcutMap {
	var m ShortcutMap
	for _, pressShift := range []bool{true, false} {
		m = m.With(
			nonConst(kBackspace, withShift(none, pressShift), stop),
			nonConst(kDelete, withShift(none, pressShift), stop),
			nonConst(kBackspace, withShift(alt, pressShift), stop),
			nonConst(kDelete, withShift(alt, pressShift), stop),
			nonConst(kBackspace, withShift(control, pressShift), stop),
			nonConst(kDelete, withShift(control, pressShift), stop),
			nonConst(kBackspace, withShift(meta, pressShift), stop),
			nonConst(kDelete, withShift(meta, pressShift), stop),
		)
	}
	m = m.With(commonDisablingTextShortcuts...)
	for _, e := range clipboardShortcuts {
		m = m.With(ShortcutEntry{Activator: e.Activator, Intent: stop})
	}
	return m.With(
		entry(services.LogicalKeyboardKeyKeyX, meta, stop),
		entry(services.LogicalKeyboardKeyKeyC, meta, stop),
		entry(services.LogicalKeyboardKeyKeyV, meta, stop),
		entry(services.LogicalKeyboardKeyKeyA, control, stop),
		entry(services.LogicalKeyboardKeyKeyA, meta, stop),
	)
}()

var iOSDisablingTextShortcuts = ShortcutMap{}.With(
	entry(kBackspace, none, stop),
	entry(kBackspace, shift, stop),
	entry(kDelete, none, stop),
	entry(kDelete, shift, stop),
	entry(kBackspace, shiftAlt, stop),
	entry(kBackspace, alt, stop),
	entry(kDelete, shiftAlt, stop),
	entry(kDelete, alt, stop),
)

var macDisablingTextShortcuts = commonDisablingTextShortcuts.With(iOSDisablingTextShortcuts...).With(
	entry(kEscape, none, stop),
	entry(kTab, none, stop),
	entry(kTab, shift, stop),
	entry(kArrowDown, shiftAlt, stop),
	entry(kArrowUp, shiftAlt, stop),
	entry(kArrowLeft, shift, stop),
	entry(kArrowRight, shift, stop),
	entry(kArrowLeft, shiftAlt, stop),
	entry(kArrowRight, shiftAlt, stop),
	entry(kArrowLeft, shiftMeta, stop),
	entry(kArrowRight, shiftMeta, stop),
	entry(kPageUp, none, stop),
	entry(kPageDown, none, stop),
	entry(kEnd, none, stop),
	entry(kHome, none, stop),
	entry(kPageUp, shift, stop),
	entry(kPageDown, shift, stop),
	entry(kEnd, shift, stop),
	entry(kHome, shift, stop),
	entry(kEnd, control, stop),
	entry(kHome, control, stop),
)

// DefaultTextEditingShortcutsShortcuts is _shortcuts: the map of the
// default target platform.
func DefaultTextEditingShortcutsShortcuts() ShortcutMap {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformFuchsia:
		return fuchsiaShortcuts
	case foundation.TargetPlatformIOS:
		return iOSShortcuts
	case foundation.TargetPlatformLinux:
		return linuxShortcuts
	case foundation.TargetPlatformMacOS:
		return macShortcuts
	case foundation.TargetPlatformWindows:
		return windowsShortcuts
	}
	return androidShortcuts
}

// DefaultTextEditingShortcutsDisablingShortcuts is _getDisablingShortcut
// (never a web build): the iOS or macOS map, nil elsewhere.
func DefaultTextEditingShortcutsDisablingShortcuts() ShortcutMap {
	if foundation.KIsWeb {
		if foundation.DefaultTargetPlatform() == foundation.TargetPlatformLinux {
			m := webDisablingTextShortcuts
			for _, e := range linuxNumpadShortcuts {
				m = m.With(ShortcutEntry{Activator: e.Activator, Intent: stop})
			}
			return m
		}
		return webDisablingTextShortcuts
	}
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformIOS:
		return iOSDisablingTextShortcuts
	case foundation.TargetPlatformMacOS:
		return macDisablingTextShortcuts
	}
	return nil
}

// DefaultTextEditingShortcutsHandlers are the key handlers of the
// Shortcuts widgets DefaultTextEditingShortcuts builds, leaf first: the
// disabling shortcuts (when the platform has them), then the default ones.
// findAction is Actions.maybeFind from the primary focus.
func DefaultTextEditingShortcutsHandlers(findAction func(Intent) Action) []OnKeyEventCallback {
	var handlers []OnKeyEventCallback
	if disabling := DefaultTextEditingShortcutsDisablingShortcuts(); disabling != nil {
		m := &ShortcutManager{Shortcuts: disabling}
		handlers = append(handlers, func(event services.KeyEvent) KeyEventResult { return m.HandleKeypress(event, findAction) })
	}
	m := &ShortcutManager{Shortcuts: DefaultTextEditingShortcutsShortcuts()}
	return append(handlers, func(event services.KeyEvent) KeyEventResult { return m.HandleKeypress(event, findAction) })
}

// IntentForMacOSSelector maps a macOS NSStandardKeyBindingResponding
// selector name to an intent (nil: no intent).
func IntentForMacOSSelector(selectorName string) Intent {
	switch selectorName {
	case "deleteBackward:":
		return NewDeleteCharacterIntent(false)
	case "deleteWordBackward:":
		return NewDeleteToNextWordBoundaryIntent(false)
	case "deleteToBeginningOfLine:":
		return NewDeleteToLineBreakIntent(false)
	case "deleteForward:":
		return NewDeleteCharacterIntent(true)
	case "deleteWordForward:":
		return NewDeleteToNextWordBoundaryIntent(true)
	case "deleteToEndOfLine:":
		return NewDeleteToLineBreakIntent(true)
	case "moveLeft:":
		return NewExtendSelectionByCharacterIntent(false, true)
	case "moveRight:":
		return NewExtendSelectionByCharacterIntent(true, true)
	case "moveForward:":
		return NewExtendSelectionByCharacterIntent(true, true)
	case "moveBackward:":
		return NewExtendSelectionByCharacterIntent(false, true)
	case "moveUp:":
		return NewExtendSelectionVerticallyToAdjacentLineIntent(false, true)
	case "moveDown:":
		return NewExtendSelectionVerticallyToAdjacentLineIntent(true, true)
	case "moveLeftAndModifySelection:":
		return NewExtendSelectionByCharacterIntent(false, false)
	case "moveRightAndModifySelection:":
		return NewExtendSelectionByCharacterIntent(true, false)
	case "moveUpAndModifySelection:":
		return NewExtendSelectionVerticallyToAdjacentLineIntent(false, false)
	case "moveDownAndModifySelection:":
		return NewExtendSelectionVerticallyToAdjacentLineIntent(true, false)
	case "moveWordLeft:":
		return NewExtendSelectionToNextWordBoundaryIntent(false, true)
	case "moveWordRight:":
		return NewExtendSelectionToNextWordBoundaryIntent(true, true)
	case "moveToBeginningOfParagraph:":
		return NewExtendSelectionToLineBreakIntent(false, true)
	case "moveToEndOfParagraph:":
		return NewExtendSelectionToLineBreakIntent(true, true)
	case "moveWordLeftAndModifySelection:":
		return NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(false)
	case "moveWordRightAndModifySelection:":
		return NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(true)
	case "moveParagraphBackwardAndModifySelection:":
		return NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(false)
	case "moveParagraphForwardAndModifySelection:":
		return NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(true)
	case "moveToLeftEndOfLine:":
		return NewExtendSelectionToLineBreakIntent(false, true)
	case "moveToRightEndOfLine:":
		return NewExtendSelectionToLineBreakIntent(true, true)
	case "moveToBeginningOfDocument:":
		return NewExtendSelectionToDocumentBoundaryIntent(false, true)
	case "moveToEndOfDocument:":
		return NewExtendSelectionToDocumentBoundaryIntent(true, true)
	case "moveToLeftEndOfLineAndModifySelection:":
		return NewExpandSelectionToLineBreakIntent(false)
	case "moveToRightEndOfLineAndModifySelection:":
		return NewExpandSelectionToLineBreakIntent(true)
	case "moveToBeginningOfDocumentAndModifySelection:":
		return NewExpandSelectionToDocumentBoundaryIntent(false)
	case "moveToEndOfDocumentAndModifySelection:":
		return NewExpandSelectionToDocumentBoundaryIntent(true)
	case "transpose:":
		return TransposeCharactersIntent{}
	case "scrollToBeginningOfDocument:":
		return NewScrollToDocumentBoundaryIntent(false)
	case "scrollToEndOfDocument:":
		return NewScrollToDocumentBoundaryIntent(true)
	case "scrollPageUp:":
		return scrollPageIntent{down: false}
	case "scrollPageDown:":
		return scrollPageIntent{down: true}
	case "pageUpAndModifySelection:":
		return NewExtendSelectionVerticallyToAdjacentPageIntent(false, false)
	case "pageDownAndModifySelection:":
		return NewExtendSelectionVerticallyToAdjacentPageIntent(true, false)
	case "cancelOperation:":
		return DismissIntent{}
	case "insertTab:":
		return focusIntent{next: true}
	case "insertBacktab:":
		return focusIntent{next: false}
	}
	return nil
}
