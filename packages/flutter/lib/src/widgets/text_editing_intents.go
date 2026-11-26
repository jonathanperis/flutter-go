package widgets

// Port of packages/flutter/lib/src/widgets/text_editing_intents.dart
// (Flutter 3.50.1), except EditableTextTapOutsideIntent and
// EditableTextTapUpOutsideIntent (they carry a FocusNode, not ported yet).
// Intents are comparable values; their required named parameters are
// positional constructor arguments.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

// DoNothingAndStopPropagationTextIntent is an Intent to send the event
// straight to the engine (the platform may handle it, e.g. macOS's
// selectors).
type DoNothingAndStopPropagationTextIntent struct{ IntentBase }

// DirectionalTextEditingIntent is a text editing related Intent that
// performs an operation towards a given direction of the current caret
// location.
type DirectionalTextEditingIntent interface {
	Intent
	// Forward is whether the input field, if applicable, should perform
	// the text editing operation from the current caret location towards
	// the end of the document.
	Forward() bool
}

type directional struct {
	IntentBase
	forward bool
}

func (d directional) Forward() bool { return d.forward }

// DeleteCharacterIntent deletes the character before or after the caret
// (or the selection).
type DeleteCharacterIntent struct{ directional }

// NewDeleteCharacterIntent is DeleteCharacterIntent(forward:).
func NewDeleteCharacterIntent(forward bool) DeleteCharacterIntent {
	return DeleteCharacterIntent{directional{forward: forward}}
}

// DeleteToNextWordBoundaryIntent deletes from the caret to the next word
// boundary.
type DeleteToNextWordBoundaryIntent struct{ directional }

// NewDeleteToNextWordBoundaryIntent is DeleteToNextWordBoundaryIntent(forward:).
func NewDeleteToNextWordBoundaryIntent(forward bool) DeleteToNextWordBoundaryIntent {
	return DeleteToNextWordBoundaryIntent{directional{forward: forward}}
}

// DeleteToLineBreakIntent deletes from the caret to the closest line break.
type DeleteToLineBreakIntent struct{ directional }

// NewDeleteToLineBreakIntent is DeleteToLineBreakIntent(forward:).
func NewDeleteToLineBreakIntent(forward bool) DeleteToLineBreakIntent {
	return DeleteToLineBreakIntent{directional{forward: forward}}
}

// DirectionalCaretMovementIntent is a DirectionalTextEditingIntent that
// moves the caret or the selection to a new location.
type DirectionalCaretMovementIntent interface {
	DirectionalTextEditingIntent
	// CollapseSelection is whether this intent should make the selection
	// collapsed (so it becomes a caret), after the movement.
	CollapseSelection() bool
	// CollapseAtReversal is whether the selection collapses to its base
	// when the movement would reverse it.
	CollapseAtReversal() bool
	// ContinuesAtWrap is whether a caret at a word wrap moves to the other
	// side of the wrap first.
	ContinuesAtWrap() bool
}

type caretMovement struct {
	directional
	collapseSelection, collapseAtReversal, continuesAtWrap bool
}

func newCaretMovement(forward, collapseSelection, collapseAtReversal, continuesAtWrap bool) caretMovement {
	if collapseSelection && collapseAtReversal {
		panic(assertionFailed("!collapseSelection || !collapseAtReversal"))
	}
	return caretMovement{directional{forward: forward}, collapseSelection, collapseAtReversal, continuesAtWrap}
}

func (c caretMovement) CollapseSelection() bool  { return c.collapseSelection }
func (c caretMovement) CollapseAtReversal() bool { return c.collapseAtReversal }
func (c caretMovement) ContinuesAtWrap() bool    { return c.continuesAtWrap }

// ExtendSelectionByCharacterIntent expands or moves the selection by one
// character.
type ExtendSelectionByCharacterIntent struct{ caretMovement }

// NewExtendSelectionByCharacterIntent is
// ExtendSelectionByCharacterIntent(forward:, collapseSelection:).
func NewExtendSelectionByCharacterIntent(forward, collapseSelection bool) ExtendSelectionByCharacterIntent {
	return ExtendSelectionByCharacterIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextWordBoundaryIntent extends the selection to the
// next word boundary.
type ExtendSelectionToNextWordBoundaryIntent struct{ caretMovement }

// NewExtendSelectionToNextWordBoundaryIntent is
// ExtendSelectionToNextWordBoundaryIntent(forward:, collapseSelection:).
func NewExtendSelectionToNextWordBoundaryIntent(forward, collapseSelection bool) ExtendSelectionToNextWordBoundaryIntent {
	return ExtendSelectionToNextWordBoundaryIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextWordBoundaryOrCaretLocationIntent extends the
// selection to the next word boundary, or collapses it to the caret if it
// would reverse.
type ExtendSelectionToNextWordBoundaryOrCaretLocationIntent struct{ caretMovement }

// NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent is
// ExtendSelectionToNextWordBoundaryOrCaretLocationIntent(forward:).
func NewExtendSelectionToNextWordBoundaryOrCaretLocationIntent(forward bool) ExtendSelectionToNextWordBoundaryOrCaretLocationIntent {
	return ExtendSelectionToNextWordBoundaryOrCaretLocationIntent{newCaretMovement(forward, false, true, false)}
}

// ExpandSelectionToDocumentBoundaryIntent expands the selection to the
// document boundary.
type ExpandSelectionToDocumentBoundaryIntent struct{ caretMovement }

// NewExpandSelectionToDocumentBoundaryIntent is
// ExpandSelectionToDocumentBoundaryIntent(forward:).
func NewExpandSelectionToDocumentBoundaryIntent(forward bool) ExpandSelectionToDocumentBoundaryIntent {
	return ExpandSelectionToDocumentBoundaryIntent{newCaretMovement(forward, false, false, false)}
}

// ExpandSelectionToLineBreakIntent expands the selection to the closest
// line break.
type ExpandSelectionToLineBreakIntent struct{ caretMovement }

// NewExpandSelectionToLineBreakIntent is
// ExpandSelectionToLineBreakIntent(forward:).
func NewExpandSelectionToLineBreakIntent(forward bool) ExpandSelectionToLineBreakIntent {
	return ExpandSelectionToLineBreakIntent{newCaretMovement(forward, false, false, false)}
}

// ExtendSelectionToLineBreakIntent extends the selection to the closest
// line break.
type ExtendSelectionToLineBreakIntent struct{ caretMovement }

// ExtendSelectionToLineBreakIntentOptions are the optional named
// parameters.
type ExtendSelectionToLineBreakIntentOptions struct {
	CollapseAtReversal bool
	ContinuesAtWrap    bool
}

// NewExtendSelectionToLineBreakIntent is
// ExtendSelectionToLineBreakIntent(forward:, collapseSelection:,
// collapseAtReversal:, continuesAtWrap:).
func NewExtendSelectionToLineBreakIntent(forward, collapseSelection bool, opts ...ExtendSelectionToLineBreakIntentOptions) ExtendSelectionToLineBreakIntent {
	var o ExtendSelectionToLineBreakIntentOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return ExtendSelectionToLineBreakIntent{newCaretMovement(forward, collapseSelection, o.CollapseAtReversal, o.ContinuesAtWrap)}
}

// ExtendSelectionVerticallyToAdjacentLineIntent extends the selection to
// the adjacent line.
type ExtendSelectionVerticallyToAdjacentLineIntent struct{ caretMovement }

// NewExtendSelectionVerticallyToAdjacentLineIntent is
// ExtendSelectionVerticallyToAdjacentLineIntent(forward:, collapseSelection:).
func NewExtendSelectionVerticallyToAdjacentLineIntent(forward, collapseSelection bool) ExtendSelectionVerticallyToAdjacentLineIntent {
	return ExtendSelectionVerticallyToAdjacentLineIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionVerticallyToAdjacentPageIntent extends the selection by a
// page.
type ExtendSelectionVerticallyToAdjacentPageIntent struct{ caretMovement }

// NewExtendSelectionVerticallyToAdjacentPageIntent is
// ExtendSelectionVerticallyToAdjacentPageIntent(forward:, collapseSelection:).
func NewExtendSelectionVerticallyToAdjacentPageIntent(forward, collapseSelection bool) ExtendSelectionVerticallyToAdjacentPageIntent {
	return ExtendSelectionVerticallyToAdjacentPageIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextParagraphBoundaryIntent extends the selection to
// the next paragraph boundary.
type ExtendSelectionToNextParagraphBoundaryIntent struct{ caretMovement }

// NewExtendSelectionToNextParagraphBoundaryIntent is
// ExtendSelectionToNextParagraphBoundaryIntent(forward:, collapseSelection:).
func NewExtendSelectionToNextParagraphBoundaryIntent(forward, collapseSelection bool) ExtendSelectionToNextParagraphBoundaryIntent {
	return ExtendSelectionToNextParagraphBoundaryIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent extends the
// selection to the next paragraph boundary, or collapses it to the caret
// if it would reverse.
type ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent struct{ caretMovement }

// NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent is
// ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(forward:).
func NewExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent(forward bool) ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent {
	return ExtendSelectionToNextParagraphBoundaryOrCaretLocationIntent{newCaretMovement(forward, false, true, false)}
}

// ExtendSelectionToDocumentBoundaryIntent extends the selection to the
// document boundary.
type ExtendSelectionToDocumentBoundaryIntent struct{ caretMovement }

// NewExtendSelectionToDocumentBoundaryIntent is
// ExtendSelectionToDocumentBoundaryIntent(forward:, collapseSelection:).
func NewExtendSelectionToDocumentBoundaryIntent(forward, collapseSelection bool) ExtendSelectionToDocumentBoundaryIntent {
	return ExtendSelectionToDocumentBoundaryIntent{newCaretMovement(forward, collapseSelection, false, false)}
}

// ScrollToDocumentBoundaryIntent scrolls to the beginning or end of the
// document.
type ScrollToDocumentBoundaryIntent struct{ directional }

// NewScrollToDocumentBoundaryIntent is ScrollToDocumentBoundaryIntent(forward:).
func NewScrollToDocumentBoundaryIntent(forward bool) ScrollToDocumentBoundaryIntent {
	return ScrollToDocumentBoundaryIntent{directional{forward: forward}}
}

// SelectAllTextIntent selects everything in the field.
type SelectAllTextIntent struct {
	IntentBase
	// Cause is the SelectionChangedCause that triggered the intent.
	Cause services.SelectionChangedCause
}

// CopySelectionTextIntent copies (or cuts) the selection to the clipboard.
type CopySelectionTextIntent struct {
	IntentBase
	// Cause is the SelectionChangedCause that triggered the intent.
	Cause services.SelectionChangedCause
	// CollapseSelection is whether the original text needs to be removed
	// from the input field if the copy action was successful (cut).
	CollapseSelection bool
}

// NewCopySelectionTextIntentCut is CopySelectionTextIntent.cut(cause).
func NewCopySelectionTextIntentCut(cause services.SelectionChangedCause) CopySelectionTextIntent {
	return CopySelectionTextIntent{Cause: cause, CollapseSelection: true}
}

// CopySelectionTextIntentCopy is CopySelectionTextIntent.copy.
var CopySelectionTextIntentCopy = CopySelectionTextIntent{Cause: services.SelectionChangedCauseKeyboard}

// PasteTextIntent pastes the clipboard's text into the field.
type PasteTextIntent struct {
	IntentBase
	// Cause is the SelectionChangedCause that triggered the intent.
	Cause services.SelectionChangedCause
}

// RedoTextIntent redoes the last undone change.
type RedoTextIntent struct {
	IntentBase
	// Cause is the SelectionChangedCause that triggered the intent.
	Cause services.SelectionChangedCause
}

// ReplaceTextIntent replaces the text in replacementRange of
// currentTextEditingValue with replacementText.
type ReplaceTextIntent struct {
	IntentBase
	CurrentTextEditingValue services.TextEditingValue
	ReplacementText         string
	ReplacementRange        ui.TextRange
	Cause                   services.SelectionChangedCause
}

// UndoTextIntent undoes the last change.
type UndoTextIntent struct {
	IntentBase
	// Cause is the SelectionChangedCause that triggered the intent.
	Cause services.SelectionChangedCause
}

// UpdateSelectionIntent updates the selection of currentTextEditingValue
// to newSelection.
type UpdateSelectionIntent struct {
	IntentBase
	CurrentTextEditingValue services.TextEditingValue
	NewSelection            services.TextSelection
	Cause                   services.SelectionChangedCause
}

// TransposeCharactersIntent swaps the characters before and after the
// caret.
type TransposeCharactersIntent struct{ IntentBase }
