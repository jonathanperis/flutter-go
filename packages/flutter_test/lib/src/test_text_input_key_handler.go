package flutter_test

// Port of packages/flutter_test/lib/src/test_text_input_key_handler.dart
// (Flutter 3.50.1): the macOS text input's response to key events the
// framework did not handle (NSStandardKeyBindingResponding selectors sent
// with TextInputClient.performSelectors).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/src/async"
)

// TestTextInputKeyHandler is a platform-specific key handler of
// TestTextInput, for unhandled key events.
type TestTextInputKeyHandler interface {
	HandleKeyDownEvent(key services.LogicalKeyboardKey) async.Future[async.Void]
	HandleKeyUpEvent(key services.LogicalKeyboardKey) async.Future[async.Void]
}

// MacOSTestTextInputKeyHandler is the macOS TestTextInputKeyHandler: a key
// down with the modifiers held sends its selectors.
type MacOSTestTextInputKeyHandler struct {
	// Client is the text input client id the selectors are sent to.
	Client                     int
	shift, alt, meta, control bool
}

// NewMacOSTestTextInputKeyHandler is MacOSTestTextInputKeyHandler(client).
func NewMacOSTestTextInputKeyHandler(client int) *MacOSTestTextInputKeyHandler {
	return &MacOSTestTextInputKeyHandler{Client: client}
}

func (h *MacOSTestTextInputKeyHandler) sendSelectors(selectors []string) async.Future[async.Void] {
	channel := services.SystemChannels.TextInput
	selectorList := make([]any, len(selectors))
	for i, s := range selectors {
		selectorList[i] = s
	}
	return async.Then(testMessenger().HandlePlatformMessage(channel.Name,
		channel.Codec.EncodeMethodCall(services.MethodCall{Method: "TextInputClient.performSelectors",
			Arguments: []any{h.Client, selectorList}}),
		func([]byte) { /* response from framework is discarded */ }),
		func([]byte) async.Void { return async.Void{} })
}

type activatorSelectors struct {
	activator widgets.SingleActivator
	selectors []string
}

// macOSActivatorToSelectors is _macOSActivatorToSelectors, in its map
// order. These combinations must match NSStandardKeyBindingResponding.
var macOSActivatorToSelectors = func() []activatorSelectors {
	type m = widgets.SingleActivatorOptions
	a := func(key services.LogicalKeyboardKey, o m, selectors ...string) activatorSelectors {
		return activatorSelectors{widgets.NewSingleActivator(key, o), selectors}
	}
	var out []activatorSelectors
	for _, shift := range []bool{true, false} {
		out = append(out,
			a(services.LogicalKeyboardKeyBackspace, m{Shift: shift}, "deleteBackward:"),
			a(services.LogicalKeyboardKeyBackspace, m{Alt: true, Shift: shift}, "deleteWordBackward:"),
			a(services.LogicalKeyboardKeyBackspace, m{Meta: true, Shift: shift}, "deleteToBeginningOfLine:"),
			a(services.LogicalKeyboardKeyBackspace, m{Control: true, Shift: shift}, "deleteBackwardByDecomposingPreviousCharacter:"),
			a(services.LogicalKeyboardKeyDelete, m{Shift: shift}, "deleteForward:"),
			a(services.LogicalKeyboardKeyDelete, m{Alt: true, Shift: shift}, "deleteWordForward:"),
			a(services.LogicalKeyboardKeyDelete, m{Meta: true, Shift: shift}, "deleteToEndOfLine:"),
		)
	}
	L, R, U, D := services.LogicalKeyboardKeyArrowLeft, services.LogicalKeyboardKeyArrowRight,
		services.LogicalKeyboardKeyArrowUp, services.LogicalKeyboardKeyArrowDown
	return append(out,
		a(L, m{}, "moveLeft:"),
		a(R, m{}, "moveRight:"),
		a(U, m{}, "moveUp:"),
		a(D, m{}, "moveDown:"),
		a(L, m{Shift: true}, "moveLeftAndModifySelection:"),
		a(R, m{Shift: true}, "moveRightAndModifySelection:"),
		a(U, m{Shift: true}, "moveUpAndModifySelection:"),
		a(D, m{Shift: true}, "moveDownAndModifySelection:"),
		a(L, m{Alt: true}, "moveWordLeft:"),
		a(R, m{Alt: true}, "moveWordRight:"),
		a(U, m{Alt: true}, "moveBackward:", "moveToBeginningOfParagraph:"),
		a(D, m{Alt: true}, "moveForward:", "moveToEndOfParagraph:"),
		a(L, m{Alt: true, Shift: true}, "moveWordLeftAndModifySelection:"),
		a(R, m{Alt: true, Shift: true}, "moveWordRightAndModifySelection:"),
		a(U, m{Alt: true, Shift: true}, "moveParagraphBackwardAndModifySelection:"),
		a(D, m{Alt: true, Shift: true}, "moveParagraphForwardAndModifySelection:"),
		a(L, m{Meta: true}, "moveToLeftEndOfLine:"),
		a(R, m{Meta: true}, "moveToRightEndOfLine:"),
		a(U, m{Meta: true}, "moveToBeginningOfDocument:"),
		a(D, m{Meta: true}, "moveToEndOfDocument:"),
		a(L, m{Meta: true, Shift: true}, "moveToLeftEndOfLineAndModifySelection:"),
		a(R, m{Meta: true, Shift: true}, "moveToRightEndOfLineAndModifySelection:"),
		a(U, m{Meta: true, Shift: true}, "moveToBeginningOfDocumentAndModifySelection:"),
		a(D, m{Meta: true, Shift: true}, "moveToEndOfDocumentAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyA, m{Control: true, Shift: true}, "moveToBeginningOfParagraphAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyA, m{Control: true}, "moveToBeginningOfParagraph:"),
		a(services.LogicalKeyboardKeyKeyB, m{Control: true, Shift: true}, "moveBackwardAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyB, m{Control: true}, "moveBackward:"),
		a(services.LogicalKeyboardKeyKeyE, m{Control: true, Shift: true}, "moveToEndOfParagraphAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyE, m{Control: true}, "moveToEndOfParagraph:"),
		a(services.LogicalKeyboardKeyKeyF, m{Control: true, Shift: true}, "moveForwardAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyF, m{Control: true}, "moveForward:"),
		a(services.LogicalKeyboardKeyKeyK, m{Control: true}, "deleteToEndOfParagraph"),
		a(services.LogicalKeyboardKeyKeyL, m{Control: true}, "centerSelectionInVisibleArea"),
		a(services.LogicalKeyboardKeyKeyN, m{Control: true}, "moveDown:"),
		a(services.LogicalKeyboardKeyKeyN, m{Control: true, Shift: true}, "moveDownAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyO, m{Control: true}, "insertNewlineIgnoringFieldEditor:"),
		a(services.LogicalKeyboardKeyKeyP, m{Control: true}, "moveUp:"),
		a(services.LogicalKeyboardKeyKeyP, m{Control: true, Shift: true}, "moveUpAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyT, m{Control: true}, "transpose:"),
		a(services.LogicalKeyboardKeyKeyV, m{Control: true}, "pageDown:"),
		a(services.LogicalKeyboardKeyKeyV, m{Control: true, Shift: true}, "pageDownAndModifySelection:"),
		a(services.LogicalKeyboardKeyKeyY, m{Control: true}, "yank:"),
		a(services.LogicalKeyboardKeyQuoteSingle, m{Control: true}, "insertSingleQuoteIgnoringSubstitution:"),
		a(services.LogicalKeyboardKeyQuote, m{Control: true}, "insertDoubleQuoteIgnoringSubstitution:"),
		a(services.LogicalKeyboardKeyHome, m{}, "scrollToBeginningOfDocument:"),
		a(services.LogicalKeyboardKeyEnd, m{}, "scrollToEndOfDocument:"),
		a(services.LogicalKeyboardKeyHome, m{Shift: true}, "moveToBeginningOfDocumentAndModifySelection:"),
		a(services.LogicalKeyboardKeyEnd, m{Shift: true}, "moveToEndOfDocumentAndModifySelection:"),
		a(services.LogicalKeyboardKeyPageUp, m{}, "scrollPageUp:"),
		a(services.LogicalKeyboardKeyPageDown, m{}, "scrollPageDown:"),
		a(services.LogicalKeyboardKeyPageUp, m{Shift: true}, "pageUpAndModifySelection:"),
		a(services.LogicalKeyboardKeyPageDown, m{Shift: true}, "pageDownAndModifySelection:"),
		a(services.LogicalKeyboardKeyEscape, m{}, "cancelOperation:"),
		a(services.LogicalKeyboardKeyEnter, m{}, "insertNewline:"),
		a(services.LogicalKeyboardKeyEnter, m{Alt: true}, "insertNewlineIgnoringFieldEditor:"),
		a(services.LogicalKeyboardKeyEnter, m{Control: true}, "insertLineBreak:"),
		a(services.LogicalKeyboardKeyTab, m{}, "insertTab:"),
		a(services.LogicalKeyboardKeyTab, m{Alt: true}, "insertTabIgnoringFieldEditor:"),
		a(services.LogicalKeyboardKeyTab, m{Shift: true}, "insertBacktab:"),
	)
}()

// HandleKeyDownEvent records a modifier, or sends the selectors of the
// first activator whose trigger and modifiers match.
func (h *MacOSTestTextInputKeyHandler) HandleKeyDownEvent(key services.LogicalKeyboardKey) async.Future[async.Void] {
	switch key {
	case services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight:
		h.shift = true
	case services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight:
		h.alt = true
	case services.LogicalKeyboardKeyMeta, services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight:
		h.meta = true
	case services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyControlLeft, services.LogicalKeyboardKeyControlRight:
		h.control = true
	default:
		for _, entry := range macOSActivatorToSelectors {
			a := entry.activator
			if a.Trigger == key && a.Shift == h.shift && a.Alt == h.alt && a.Meta == h.meta && a.Control == h.control {
				return h.sendSelectors(entry.selectors)
			}
		}
	}
	return async.FutureValue(async.Void{})
}

// HandleKeyUpEvent forgets a released modifier.
func (h *MacOSTestTextInputKeyHandler) HandleKeyUpEvent(key services.LogicalKeyboardKey) async.Future[async.Void] {
	switch key {
	case services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight:
		h.shift = false
	case services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight:
		h.alt = false
	case services.LogicalKeyboardKeyMeta, services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight:
		h.meta = false
	case services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyControlLeft, services.LogicalKeyboardKeyControlRight:
		h.control = false
	}
	return async.FutureValue(async.Void{})
}
