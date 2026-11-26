package services

// Port of packages/flutter/lib/src/services/text_editing_delta.dart
// (Flutter 3.50.1).

import (
	"strconv"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// replaceText replaces a range of text in the original string with the
// text given in the replacement string.
func replaceText(originalText, replacementText string, replacementRange ui.TextRange) string {
	if !replacementRange.IsValid() {
		panic(dart.NewAssertionError(nil))
	}
	return dart.ReplaceRange(originalText, replacementRange.Start, replacementRange.End, replacementText)
}

// debugTextRangeIsValid verifies that the given range is within the text.
func debugTextRangeIsValid(r ui.TextRange, text string) bool {
	if !r.IsValid() {
		return true
	}
	n := dart.StringLength(text)
	return (r.Start >= 0 && r.Start <= n) && (r.End >= 0 && r.End <= n)
}

func assertDelta(ok bool, message func() string) {
	if !ok {
		panic(dart.NewAssertionError(message()))
	}
}

func textLength(text string) string { return strconv.Itoa(dart.StringLength(text)) }

// TextEditingDelta is a structure representing a granular change that has
// occurred to the editing state as a result of text editing (an abstract
// class: the four TextEditingDelta... types implement it, and a subclass
// embeds TextEditingDeltaBase).
type TextEditingDelta interface {
	foundation.Diagnosticable
	// OldText is the old text state before the delta has occurred.
	OldText() string
	// Selection is the range of text that is currently selected after the
	// delta has been applied.
	Selection() TextSelection
	// Composing is the range of text that is still being composed after
	// the delta has been applied.
	Composing() ui.TextRange
	// Apply is the TextEditingValue after applying the delta (to oldText,
	// "last write wins").
	Apply(value TextEditingValue) TextEditingValue
}

// TextEditingDeltaBase holds a delta's oldText, selection and composing
// (embed it, then call SetSelf, as Diagnosticable requires).
type TextEditingDeltaBase struct {
	foundation.DiagnosticableMixin
	oldText   string
	selection TextSelection
	composing ui.TextRange
}

// InitTextEditingDelta sets the base fields (Dart's super constructor).
func (d *TextEditingDeltaBase) InitTextEditingDelta(oldText string, selection TextSelection, composing ui.TextRange) {
	d.oldText, d.selection, d.composing = oldText, selection, composing
}

func (d *TextEditingDeltaBase) OldText() string          { return d.oldText }
func (d *TextEditingDeltaBase) Selection() TextSelection { return d.selection }
func (d *TextEditingDeltaBase) Composing() ui.TextRange  { return d.composing }

// NewTextEditingDeltaFromJSON is TextEditingDelta.fromJSON: the insertion,
// deletion, replacement or non-text update the platform's encoded delta
// describes.
func NewTextEditingDeltaFromJSON(encoded *dart.LinkedMap) TextEditingDelta {
	intOr := func(key string) int {
		if v, ok := encoded.Lookup(key).(int); ok {
			return v
		}
		return -1
	}
	oldText := encoded.Lookup("oldText").(string)
	replacementDestinationStart := encoded.Lookup("deltaStart").(int)
	replacementDestinationEnd := encoded.Lookup("deltaEnd").(int)
	replacementSource := encoded.Lookup("deltaText").(string)
	const replacementSourceStart = 0
	replacementSourceEnd := dart.StringLength(replacementSource)

	// This delta is explicitly a non text update.
	isNonTextUpdate := replacementDestinationStart == -1 && replacementDestinationStart == replacementDestinationEnd
	newComposing := ui.NewTextRange(intOr("composingBase"), intOr("composingExtent"))
	affinity := ui.TextAffinityDownstream
	if s, ok := encoded.Lookup("selectionAffinity").(string); ok {
		if a := toTextAffinity(&s); a != nil {
			affinity = *a
		}
	}
	directional, _ := encoded.Lookup("selectionIsDirectional").(bool)
	newSelection := NewTextSelection(intOr("selectionBase"), intOr("selectionExtent"),
		TextSelectionOptions{Affinity: &affinity, IsDirectional: directional})

	if isNonTextUpdate {
		assertDelta(debugTextRangeIsValid(newSelection.TextRange, oldText), func() string {
			return "The selection range: " + newSelection.String() + " is not within the bounds of text: " + oldText + " of length: " + textLength(oldText)
		})
		assertDelta(debugTextRangeIsValid(newComposing, oldText), func() string {
			return "The composing range: " + newComposing.String() + " is not within the bounds of text: " + oldText + " of length: " + textLength(oldText)
		})
		return NewTextEditingDeltaNonTextUpdate(oldText, newSelection, newComposing)
	}

	assertDelta(debugTextRangeIsValid(ui.NewTextRange(replacementDestinationStart, replacementDestinationEnd), oldText), func() string {
		return "The delta range: " + ui.NewTextRange(replacementSourceStart, replacementSourceEnd).String() +
			" is not within the bounds of text: " + oldText + " of length: " + textLength(oldText)
	})

	newText := replaceText(oldText, replacementSource, ui.NewTextRange(replacementDestinationStart, replacementDestinationEnd))

	assertDelta(debugTextRangeIsValid(newSelection.TextRange, newText), func() string {
		return "The selection range: " + newSelection.String() + " is not within the bounds of text: " + newText + " of length: " + textLength(newText)
	})
	assertDelta(debugTextRangeIsValid(newComposing, newText), func() string {
		return "The composing range: " + newComposing.String() + " is not within the bounds of text: " + newText + " of length: " + textLength(newText)
	})

	isEqual := oldText == newText

	isDeletionGreaterThanOne := (replacementDestinationEnd-replacementDestinationStart)-
		(replacementSourceEnd-replacementSourceStart) > 1
	isDeletingByReplacingWithEmpty := replacementSource == "" &&
		replacementSourceStart == 0 && replacementSourceStart == replacementSourceEnd

	isReplacedByShorter := isDeletionGreaterThanOne &&
		(replacementSourceEnd-replacementSourceStart < replacementDestinationEnd-replacementDestinationStart)
	isReplacedByLonger := replacementSourceEnd-replacementSourceStart >
		replacementDestinationEnd-replacementDestinationStart
	isReplacedBySame := replacementSourceEnd-replacementSourceStart ==
		replacementDestinationEnd-replacementDestinationStart

	isInsertingInsideComposingRegion := replacementDestinationStart+replacementSourceEnd > replacementDestinationEnd
	isDeletingInsideComposingRegion := !isReplacedByShorter && !isDeletingByReplacingWithEmpty &&
		replacementDestinationStart+replacementSourceEnd < replacementDestinationEnd

	var newComposingText, originalComposingText string
	if isDeletingByReplacingWithEmpty || isDeletingInsideComposingRegion || isReplacedByShorter {
		newComposingText = dart.Substring(replacementSource, replacementSourceStart, replacementSourceEnd)
		originalComposingText = dart.Substring(oldText, replacementDestinationStart,
			replacementDestinationStart+replacementSourceEnd)
	} else {
		newComposingText = dart.Substring(replacementSource, replacementSourceStart,
			replacementSourceStart+(replacementDestinationEnd-replacementDestinationStart))
		originalComposingText = dart.Substring(oldText, replacementDestinationStart, replacementDestinationEnd)
	}

	isOriginalComposingRegionTextChanged := !(originalComposingText == newComposingText)
	isReplaced := isOriginalComposingRegionTextChanged ||
		(isReplacedByLonger || isReplacedByShorter || isReplacedBySame)

	switch {
	case isEqual:
		return NewTextEditingDeltaNonTextUpdate(oldText, newSelection, newComposing)
	case (isDeletingByReplacingWithEmpty || isDeletingInsideComposingRegion) && !isOriginalComposingRegionTextChanged:
		// Deletion.
		actualStart := replacementDestinationStart
		if !isDeletionGreaterThanOne {
			actualStart = replacementDestinationEnd - 1
		}
		return NewTextEditingDeltaDeletion(oldText, ui.NewTextRange(actualStart, replacementDestinationEnd),
			newSelection, newComposing)
	case (replacementDestinationStart == replacementDestinationEnd || isInsertingInsideComposingRegion) &&
		!isOriginalComposingRegionTextChanged:
		// Insertion.
		return NewTextEditingDeltaInsertion(oldText,
			dart.Substring(replacementSource, replacementDestinationEnd-replacementDestinationStart,
				(replacementDestinationEnd-replacementDestinationStart)+
					(dart.StringLength(replacementSource)-(replacementDestinationEnd-replacementDestinationStart))),
			replacementDestinationEnd, newSelection, newComposing)
	case isReplaced:
		// Replacement.
		return NewTextEditingDeltaReplacement(oldText, replacementSource,
			ui.NewTextRange(replacementDestinationStart, replacementDestinationEnd), newSelection, newComposing)
	}
	panic(dart.NewAssertionError(nil))
}

func applyDelta(kind, rangeName string, r ui.TextRange, oldText, replacement string, selection TextSelection, composing ui.TextRange, value TextEditingValue) TextEditingValue {
	// To stay inline with the plain text model we should follow a last
	// write wins policy and apply the delta to the oldText. This is due to
	// the asynchronous nature of the connection between the framework and
	// platform text input plugins.
	newText := oldText
	assertDelta(debugTextRangeIsValid(r, newText), func() string {
		return "Applying " + kind + " failed, the " + rangeName + " is not within the bounds of " + newText + " of length: " + textLength(newText)
	})
	newText = replaceText(newText, replacement, r)
	assertDelta(debugTextRangeIsValid(selection.TextRange, newText), func() string {
		return "Applying " + kind + " failed, the selection range: " + selection.String() + " is not within the bounds of " + newText + " of length: " + textLength(newText)
	})
	assertDelta(debugTextRangeIsValid(composing, newText), func() string {
		return "Applying " + kind + " failed, the composing range: " + composing.String() + " is not within the bounds of " + newText + " of length: " + textLength(newText)
	})
	return value.CopyWith(TextEditingValueCopyWithOptions{Text: &newText, Selection: &selection, Composing: &composing})
}

func fillDeltaProperties(properties *foundation.DiagnosticPropertiesBuilder, d TextEditingDelta, middle ...foundation.DiagnosticsNode) {
	properties.Add(foundation.NewDiagnosticsProperty("oldText", d.OldText()))
	for _, p := range middle {
		properties.Add(p)
	}
	properties.Add(foundation.NewDiagnosticsProperty("selection", d.Selection()))
	properties.Add(foundation.NewDiagnosticsProperty("composing", d.Composing()))
}

// TextEditingDeltaInsertion is a TextEditingDelta representing an insertion
// at a single point into the text state.
type TextEditingDeltaInsertion struct {
	TextEditingDeltaBase
	// TextInserted is the string that is being inserted into OldText.
	TextInserted string
	// InsertionOffset is the offset in OldText where the insertion begins.
	InsertionOffset int
}

// NewTextEditingDeltaInsertion is TextEditingDeltaInsertion({required
// oldText, required textInserted, required insertionOffset, required
// selection, required composing}).
func NewTextEditingDeltaInsertion(oldText, textInserted string, insertionOffset int, selection TextSelection, composing ui.TextRange) *TextEditingDeltaInsertion {
	d := &TextEditingDeltaInsertion{TextInserted: textInserted, InsertionOffset: insertionOffset}
	d.InitTextEditingDelta(oldText, selection, composing)
	d.SetSelf(d)
	return d
}

// Apply applies the insertion to OldText.
func (d *TextEditingDeltaInsertion) Apply(value TextEditingValue) TextEditingValue {
	return applyDelta("TextEditingDeltaInsertion", "insertionOffset: "+strconv.Itoa(d.InsertionOffset),
		ui.NewTextRangeCollapsed(d.InsertionOffset), d.oldText, d.TextInserted, d.selection, d.composing, value)
}

// DebugFillProperties adds oldText, textInserted, insertionOffset,
// selection and composing.
func (d *TextEditingDeltaInsertion) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	fillDeltaProperties(properties, d,
		foundation.NewDiagnosticsProperty("textInserted", d.TextInserted),
		foundation.NewDiagnosticsProperty("insertionOffset", d.InsertionOffset))
}

// TextEditingDeltaDeletion is a TextEditingDelta representing a deletion of
// a single contiguous range in the text state.
type TextEditingDeltaDeletion struct {
	TextEditingDeltaBase
	// DeletedRange is the range in OldText that is being deleted.
	DeletedRange ui.TextRange
}

// NewTextEditingDeltaDeletion is TextEditingDeltaDeletion({required oldText,
// required deletedRange, required selection, required composing}).
func NewTextEditingDeltaDeletion(oldText string, deletedRange ui.TextRange, selection TextSelection, composing ui.TextRange) *TextEditingDeltaDeletion {
	d := &TextEditingDeltaDeletion{DeletedRange: deletedRange}
	d.InitTextEditingDelta(oldText, selection, composing)
	d.SetSelf(d)
	return d
}

// TextDeleted is the string being deleted in OldText.
func (d *TextEditingDeltaDeletion) TextDeleted() string {
	return dart.Substring(d.oldText, d.DeletedRange.Start, d.DeletedRange.End)
}

// Apply applies the deletion to OldText.
func (d *TextEditingDeltaDeletion) Apply(value TextEditingValue) TextEditingValue {
	return applyDelta("TextEditingDeltaDeletion", "deletedRange: "+d.DeletedRange.String(),
		d.DeletedRange, d.oldText, "", d.selection, d.composing, value)
}

// DebugFillProperties adds oldText, textDeleted, deletedRange, selection
// and composing.
func (d *TextEditingDeltaDeletion) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	fillDeltaProperties(properties, d,
		foundation.NewDiagnosticsProperty("textDeleted", d.TextDeleted()),
		foundation.NewDiagnosticsProperty("deletedRange", d.DeletedRange))
}

// TextEditingDeltaReplacement is a TextEditingDelta representing a
// replacement of a single contiguous range in the text state.
type TextEditingDeltaReplacement struct {
	TextEditingDeltaBase
	// ReplacementText is the new text that is replacing the ReplacedRange
	// in OldText.
	ReplacementText string
	// ReplacedRange is the range in OldText that is being replaced.
	ReplacedRange ui.TextRange
}

// NewTextEditingDeltaReplacement is TextEditingDeltaReplacement({required
// oldText, required replacementText, required replacedRange, required
// selection, required composing}).
func NewTextEditingDeltaReplacement(oldText, replacementText string, replacedRange ui.TextRange, selection TextSelection, composing ui.TextRange) *TextEditingDeltaReplacement {
	d := &TextEditingDeltaReplacement{ReplacementText: replacementText, ReplacedRange: replacedRange}
	d.InitTextEditingDelta(oldText, selection, composing)
	d.SetSelf(d)
	return d
}

// TextReplaced is the string in OldText that is being replaced.
func (d *TextEditingDeltaReplacement) TextReplaced() string {
	return dart.Substring(d.oldText, d.ReplacedRange.Start, d.ReplacedRange.End)
}

// Apply applies the replacement to OldText.
func (d *TextEditingDeltaReplacement) Apply(value TextEditingValue) TextEditingValue {
	return applyDelta("TextEditingDeltaReplacement", "replacedRange: "+d.ReplacedRange.String(),
		d.ReplacedRange, d.oldText, d.ReplacementText, d.selection, d.composing, value)
}

// DebugFillProperties adds oldText, textReplaced, replacementText,
// replacedRange, selection and composing.
func (d *TextEditingDeltaReplacement) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	fillDeltaProperties(properties, d,
		foundation.NewDiagnosticsProperty("textReplaced", d.TextReplaced()),
		foundation.NewDiagnosticsProperty("replacementText", d.ReplacementText),
		foundation.NewDiagnosticsProperty("replacedRange", d.ReplacedRange))
}

// TextEditingDeltaNonTextUpdate is a TextEditingDelta representing a change
// in the selection and/or composing region.
type TextEditingDeltaNonTextUpdate struct{ TextEditingDeltaBase }

// NewTextEditingDeltaNonTextUpdate is TextEditingDeltaNonTextUpdate({required
// oldText, required selection, required composing}).
func NewTextEditingDeltaNonTextUpdate(oldText string, selection TextSelection, composing ui.TextRange) *TextEditingDeltaNonTextUpdate {
	d := &TextEditingDeltaNonTextUpdate{}
	d.InitTextEditingDelta(oldText, selection, composing)
	d.SetSelf(d)
	return d
}

// Apply returns OldText with the new selection and composing region.
func (d *TextEditingDeltaNonTextUpdate) Apply(value TextEditingValue) TextEditingValue {
	assertDelta(debugTextRangeIsValid(d.selection.TextRange, d.oldText), func() string {
		return "Applying TextEditingDeltaNonTextUpdate failed, the selection range: " + d.selection.String() +
			" is not within the bounds of " + d.oldText + " of length: " + textLength(d.oldText)
	})
	assertDelta(debugTextRangeIsValid(d.composing, d.oldText), func() string {
		return "Applying TextEditingDeltaNonTextUpdate failed, the composing region: " + d.composing.String() +
			" is not within the bounds of " + d.oldText + " of length: " + textLength(d.oldText)
	})
	selection, composing := d.selection, d.composing
	return NewTextEditingValue(TextEditingValueOptions{Text: d.oldText, Selection: &selection, Composing: &composing})
}

// DebugFillProperties adds oldText, selection and composing.
func (d *TextEditingDeltaNonTextUpdate) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	fillDeltaProperties(properties, d)
}
