package services

// Port of packages/flutter/lib/src/services/text_input.dart (Flutter
// 3.50.1). Text is a Go string read as WTF-8 (src/dart's string.go);
// offsets are UTF-16 code units, as in Dart.

import (
	"strconv"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func toTextAffinity(affinity *string) *ui.TextAffinity {
	if affinity == nil {
		return nil
	}
	var a ui.TextAffinity
	switch *affinity {
	case "TextAffinity.downstream":
		a = ui.TextAffinityDownstream
	case "TextAffinity.upstream":
		a = ui.TextAffinityUpstream
	default:
		return nil
	}
	return &a
}

// TextEditingValue is the current text, selection, and composing state for
// editing a run of text. Compare with Equal (Dart's ==).
type TextEditingValue struct {
	// Text is the current text being edited.
	Text string
	// Selection is the range of text that is currently selected.
	Selection TextSelection
	// Composing is the range of text that is still being composed.
	Composing ui.TextRange
}

// TextEditingValueOptions are TextEditingValue's named parameters (nil:
// the Dart defaults, TextSelection.collapsed(offset: -1) and
// TextRange.empty).
type TextEditingValueOptions struct {
	Text      string
	Selection *TextSelection
	Composing *ui.TextRange
}

// NewTextEditingValue is TextEditingValue({text, selection, composing}).
func NewTextEditingValue(opts ...TextEditingValueOptions) TextEditingValue {
	var o TextEditingValueOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	v := TextEditingValue{Text: o.Text, Selection: NewTextSelectionCollapsed(-1), Composing: ui.TextRangeEmpty}
	if o.Selection != nil {
		v.Selection = *o.Selection
	}
	if o.Composing != nil {
		v.Composing = *o.Composing
	}
	return v
}

// TextEditingValueEmpty is TextEditingValue.empty: the empty string with no
// selection and no composing range.
var TextEditingValueEmpty = NewTextEditingValue()

// NewTextEditingValueFromJSON is TextEditingValue.fromJSON.
func NewTextEditingValueFromJSON(encoded map[string]any) TextEditingValue {
	text := encoded["text"].(string)
	intOr := func(key string) int {
		if v, ok := encoded[key].(int); ok {
			return v
		}
		return -1
	}
	affinity := ui.TextAffinityDownstream
	if s, ok := encoded["selectionAffinity"].(string); ok {
		if a := toTextAffinity(&s); a != nil {
			affinity = *a
		}
	}
	directional, _ := encoded["selectionIsDirectional"].(bool)
	selection := NewTextSelection(intOr("selectionBase"), intOr("selectionExtent"),
		TextSelectionOptions{Affinity: &affinity, IsDirectional: directional})
	composing := ui.NewTextRange(intOr("composingBase"), intOr("composingExtent"))
	textRangeIsValid(selection.TextRange, text)
	textRangeIsValid(composing, text)
	return TextEditingValue{Text: text, Selection: selection, Composing: composing}
}

// TextEditingValueCopyWithOptions are copyWith's parameters (nil: keep).
type TextEditingValueCopyWithOptions struct {
	Text      *string
	Selection *TextSelection
	Composing *ui.TextRange
}

// CopyWith creates a copy of this value but with the given fields replaced
// with the new values.
func (v TextEditingValue) CopyWith(opts ...TextEditingValueCopyWithOptions) TextEditingValue {
	var o TextEditingValueCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Text != nil {
		v.Text = *o.Text
	}
	if o.Selection != nil {
		v.Selection = *o.Selection
	}
	if o.Composing != nil {
		v.Composing = *o.Composing
	}
	return v
}

// IsComposingRangeValid is whether the composing range is a valid range
// within the text.
func (v TextEditingValue) IsComposingRangeValid() bool {
	return v.Composing.IsValid() && v.Composing.IsNormalized() && v.Composing.End <= dart.StringLength(v.Text)
}

// Replaced returns this value with its text partially replaced by
// replacementString, adjusting the selection and the composing range so
// they point to the same substrings. It does nothing when replacementRange
// is not valid.
func (v TextEditingValue) Replaced(replacementRange ui.TextRange, replacementString string) TextEditingValue {
	if !replacementRange.IsValid() {
		return v
	}
	newText := dart.ReplaceRange(v.Text, replacementRange.Start, replacementRange.End, replacementString)
	replacementLength := dart.StringLength(replacementString)
	if replacementRange.End-replacementRange.Start == replacementLength {
		return v.CopyWith(TextEditingValueCopyWithOptions{Text: &newText})
	}
	adjustIndex := func(originalIndex int) int {
		// The length added by adding the replacementString.
		replacedLength := replacementLength
		if originalIndex <= replacementRange.Start && originalIndex < replacementRange.End {
			replacedLength = 0
		}
		// The length removed by removing the replacementRange.
		removedLength := min(max(originalIndex, replacementRange.Start), replacementRange.End) - replacementRange.Start
		return originalIndex + replacedLength - removedLength
	}
	adjustedSelection := NewTextSelection(adjustIndex(v.Selection.BaseOffset), adjustIndex(v.Selection.ExtentOffset))
	adjustedComposing := ui.NewTextRange(adjustIndex(v.Composing.Start), adjustIndex(v.Composing.End))
	textRangeIsValid(adjustedSelection.TextRange, newText)
	textRangeIsValid(adjustedComposing, newText)
	return TextEditingValue{Text: newText, Selection: adjustedSelection, Composing: adjustedComposing}
}

// ToJSON returns a representation of this object as a JSON object (an
// ordered map, as Dart's map literal).
func (v TextEditingValue) ToJSON() []dart.MapEntry[string, any] {
	textRangeIsValid(v.Selection.TextRange, v.Text)
	textRangeIsValid(v.Composing, v.Text)
	return []dart.MapEntry[string, any]{
		{Key: "text", Value: v.Text},
		{Key: "selectionBase", Value: v.Selection.BaseOffset},
		{Key: "selectionExtent", Value: v.Selection.ExtentOffset},
		{Key: "selectionAffinity", Value: v.Selection.Affinity.String()},
		{Key: "selectionIsDirectional", Value: v.Selection.IsDirectional},
		{Key: "composingBase", Value: v.Composing.Start},
		{Key: "composingExtent", Value: v.Composing.End},
	}
}

func (v TextEditingValue) String() string {
	return foundation.ObjectRuntimeType(v, "TextEditingValue") + "(text: ┤" + v.Text + "├, selection: " +
		v.Selection.String() + ", composing: " + v.Composing.String() + ")"
}

// Equal is TextEditingValue's ==.
func (v TextEditingValue) Equal(other TextEditingValue) bool {
	return other.Text == v.Text && other.Selection.Equal(v.Selection) && other.Composing == v.Composing
}

// textRangeIsValid verifies that the given range is within the text (Dart
// asserts it).
func textRangeIsValid(r ui.TextRange, text string) bool {
	if r.Start == -1 && r.End == -1 {
		return true
	}
	n := dart.StringLength(text)
	if !(r.Start >= 0 && r.Start <= n) {
		panic(dart.NewAssertionError("Range start " + strconv.Itoa(r.Start) + " is out of text of length " + strconv.Itoa(n)))
	}
	if !(r.End >= 0 && r.End <= n) {
		panic(dart.NewAssertionError("Range end " + strconv.Itoa(r.End) + " is out of text of length " + strconv.Itoa(n)))
	}
	return true
}
