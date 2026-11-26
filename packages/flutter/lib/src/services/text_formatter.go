package services

// Port of packages/flutter/lib/src/services/text_formatter.dart (Flutter
// 3.50.1). Patterns are dart.Pattern values: dart.StringPattern (a Dart
// String) or *dart.RegExp (a Dart RegExp, matched with Dart's rules on
// UTF-16 code units); see src/dart's regexp.go.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/characters"
)

// MaxLengthEnforcement is the mechanism used to enforce the maximum
// length of a text field.
type MaxLengthEnforcement int

const (
	// MaxLengthEnforcementNone: no enforcement applied to the editing value.
	MaxLengthEnforcementNone MaxLengthEnforcement = iota
	// MaxLengthEnforcementEnforced: keep the length of the text input from
	// exceeding the max length, even while composing.
	MaxLengthEnforcementEnforced
	// MaxLengthEnforcementTruncateAfterCompositionEnds: users can input text
	// beyond the max length while composing; it is truncated after the
	// composition ends.
	MaxLengthEnforcementTruncateAfterCompositionEnds
)

// MaxLengthEnforcementValues is MaxLengthEnforcement.values.
var MaxLengthEnforcementValues = []MaxLengthEnforcement{MaxLengthEnforcementNone, MaxLengthEnforcementEnforced,
	MaxLengthEnforcementTruncateAfterCompositionEnds}

// Name is the enum value's Dart name.
func (e MaxLengthEnforcement) Name() string {
	return [...]string{"none", "enforced", "truncateAfterCompositionEnds"}[e]
}

func (e MaxLengthEnforcement) String() string { return "MaxLengthEnforcement." + e.Name() }

// TextInputFormatter provides a way to modify the input to a text field as
// it is being typed.
type TextInputFormatter interface {
	// FormatEditUpdate is called when text is being typed or cut/copy/
	// pasted in the EditableText.
	FormatEditUpdate(oldValue, newValue TextEditingValue) TextEditingValue
}

// TextInputFormatFunction is the function TextInputFormatter.withFunction
// wraps.
type TextInputFormatFunction func(oldValue, newValue TextEditingValue) TextEditingValue

// NewTextInputFormatterWithFunction is TextInputFormatter.withFunction: a
// formatter that calls formatFunction.
func NewTextInputFormatterWithFunction(formatFunction TextInputFormatFunction) TextInputFormatter {
	return simpleTextInputFormatter{formatFunction}
}

type simpleTextInputFormatter struct{ formatFunction TextInputFormatFunction }

func (f simpleTextInputFormatter) FormatEditUpdate(oldValue, newValue TextEditingValue) TextEditingValue {
	return f.formatFunction(oldValue, newValue)
}

// mutableTextRange is a mutable, half-open range [base, extent) within a
// string.
type mutableTextRange struct{ base, extent int }

func mutableTextRangeFromComposingRange(r ui.TextRange) *mutableTextRange {
	if r.IsValid() && !r.IsCollapsed() {
		return &mutableTextRange{r.Start, r.End}
	}
	return nil
}

func mutableTextRangeFromTextSelection(s TextSelection) *mutableTextRange {
	if s.IsValid() {
		return &mutableTextRange{s.BaseOffset, s.ExtentOffset}
	}
	return nil
}

// textEditingValueAccumulator is the intermediate state of a
// FilteringTextInputFormatter when it's formatting a new user input.
type textEditingValueAccumulator struct {
	inputValue      TextEditingValue
	inputUnits      []uint16
	buffer          []uint16
	selection       *mutableTextRange
	composingRegion *mutableTextRange
	debugFinalized  bool
}

func newTextEditingValueAccumulator(inputValue TextEditingValue) *textEditingValueAccumulator {
	return &textEditingValueAccumulator{
		inputValue:      inputValue,
		inputUnits:      dart.StringCodeUnits(inputValue.Text),
		selection:       mutableTextRangeFromTextSelection(inputValue.Selection),
		composingRegion: mutableTextRangeFromComposingRange(inputValue.Composing),
	}
}

func (a *textEditingValueAccumulator) finalize() TextEditingValue {
	a.debugFinalized = true
	composing := ui.TextRangeEmpty
	if a.composingRegion != nil && a.composingRegion.base != a.composingRegion.extent {
		composing = ui.NewTextRange(a.composingRegion.base, a.composingRegion.extent)
	}
	selection := NewTextSelectionCollapsed(-1)
	if a.selection != nil {
		// Try to preserve the selection affinity and isDirectional. This may
		// not make sense if the selection has changed.
		affinity := a.inputValue.Selection.Affinity
		selection = NewTextSelection(a.selection.base, a.selection.extent,
			TextSelectionOptions{Affinity: &affinity, IsDirectional: a.inputValue.Selection.IsDirectional})
	}
	return NewTextEditingValue(TextEditingValueOptions{Text: dart.StringFromCharCodes(a.buffer),
		Composing: &composing, Selection: &selection})
}

// FilteringTextInputFormatter is a TextInputFormatter that prevents the
// insertion of characters matching (or not matching) a particular pattern,
// by replacing them with ReplacementString.
type FilteringTextInputFormatter struct {
	// FilterPattern is the pattern to match (dart.StringPattern or
	// *dart.RegExp).
	FilterPattern dart.Pattern
	// Allow is whether the pattern is an allow list (true) or a deny list.
	Allow bool
	// ReplacementString replaces banned patterns.
	ReplacementString string
}

// FilteringTextInputFormatterOptions are the constructors' optional named
// parameters.
type FilteringTextInputFormatterOptions struct {
	ReplacementString string
}

// NewFilteringTextInputFormatter is FilteringTextInputFormatter(pattern,
// {required allow, replacementString = ”}).
func NewFilteringTextInputFormatter(filterPattern dart.Pattern, allow bool, opts ...FilteringTextInputFormatterOptions) *FilteringTextInputFormatter {
	var o FilteringTextInputFormatterOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &FilteringTextInputFormatter{FilterPattern: filterPattern, Allow: allow, ReplacementString: o.ReplacementString}
}

// NewFilteringTextInputFormatterAllow is FilteringTextInputFormatter.allow.
func NewFilteringTextInputFormatterAllow(filterPattern dart.Pattern, opts ...FilteringTextInputFormatterOptions) *FilteringTextInputFormatter {
	return NewFilteringTextInputFormatter(filterPattern, true, opts...)
}

// NewFilteringTextInputFormatterDeny is FilteringTextInputFormatter.deny.
func NewFilteringTextInputFormatterDeny(filterPattern dart.Pattern, opts ...FilteringTextInputFormatterOptions) *FilteringTextInputFormatter {
	return NewFilteringTextInputFormatter(filterPattern, false, opts...)
}

// FormatEditUpdate filters newValue's text (oldValue is unused).
func (f *FilteringTextInputFormatter) FormatEditUpdate(oldValue, newValue TextEditingValue) TextEditingValue {
	formatState := newTextEditingValueAccumulator(newValue)
	if formatState.debugFinalized {
		panic(dart.NewAssertionError(nil))
	}
	matches := dart.AllMatches(f.FilterPattern, newValue.Text)
	var previousMatch *dart.Match
	previousEnd := func() int {
		if previousMatch == nil {
			return 0
		}
		return previousMatch.End
	}
	for i := range matches {
		match := &matches[i]
		if match.End < match.Start {
			panic(dart.NewAssertionError(nil))
		}
		// The non-matching region.
		f.processRegion(f.Allow, previousEnd(), match.Start, formatState)
		// The matched region.
		f.processRegion(!f.Allow, match.Start, match.End, formatState)
		previousMatch = match
	}
	// Handle the last non-matching region between the last match region and
	// the end of the text.
	f.processRegion(f.Allow, previousEnd(), len(formatState.inputUnits), formatState)
	return formatState.finalize()
}

func (f *FilteringTextInputFormatter) processRegion(isBannedRegion bool, regionStart, regionEnd int, state *textEditingValueAccumulator) {
	var replacement []uint16
	if isBannedRegion {
		if regionStart != regionEnd {
			replacement = dart.StringCodeUnits(f.ReplacementString)
		}
	} else {
		replacement = state.inputUnits[regionStart:regionEnd]
	}
	state.buffer = append(state.buffer, replacement...)
	if len(replacement) == regionEnd-regionStart {
		// We don't have to adjust the indices if the replaced string and the
		// replacement string have the same length.
		return
	}
	adjustIndex := func(originalIndex int) int {
		// The length added by adding the replacementString.
		replacedLength := len(replacement)
		if originalIndex <= regionStart && originalIndex < regionEnd {
			replacedLength = 0
		}
		// The length removed by removing the replacementRange.
		removedLength := min(max(originalIndex, regionStart), regionEnd) - regionStart
		return replacedLength - removedLength
	}
	if s := state.selection; s != nil {
		s.base += adjustIndex(state.inputValue.Selection.BaseOffset)
		s.extent += adjustIndex(state.inputValue.Selection.ExtentOffset)
	}
	if c := state.composingRegion; c != nil {
		c.base += adjustIndex(state.inputValue.Composing.Start)
		c.extent += adjustIndex(state.inputValue.Composing.End)
	}
}

// FilteringTextInputFormatterSingleLineFormatter is
// FilteringTextInputFormatter.singleLineFormatter: it forces input to be a
// single line by denying '\n'.
var FilteringTextInputFormatterSingleLineFormatter TextInputFormatter = NewFilteringTextInputFormatterDeny(dart.StringPattern("\n"))

// FilteringTextInputFormatterDigitsOnly is
// FilteringTextInputFormatter.digitsOnly: it takes in digits [0-9] only.
var FilteringTextInputFormatterDigitsOnly TextInputFormatter = NewFilteringTextInputFormatterAllow(dart.NewRegExp(`[0-9]`))

// LengthLimitingTextInputFormatter is a TextInputFormatter that prevents
// the insertion of more characters (grapheme clusters) than allowed.
type LengthLimitingTextInputFormatter struct {
	// MaxLength is the limit (nil or -1: no limit).
	MaxLength *int
	// MaxLengthEnforcement determines how the limit is enforced (nil: the
	// platform's default, getDefaultMaxLengthEnforcement).
	MaxLengthEnforcement *MaxLengthEnforcement
}

// LengthLimitingTextInputFormatterOptions are the constructor's named
// parameters.
type LengthLimitingTextInputFormatterOptions struct {
	MaxLengthEnforcement *MaxLengthEnforcement
}

// NewLengthLimitingTextInputFormatter is
// LengthLimitingTextInputFormatter(maxLength, {maxLengthEnforcement}).
func NewLengthLimitingTextInputFormatter(maxLength *int, opts ...LengthLimitingTextInputFormatterOptions) *LengthLimitingTextInputFormatter {
	if !(maxLength == nil || *maxLength == -1 || *maxLength > 0) {
		panic(dart.NewAssertionError(nil))
	}
	var o LengthLimitingTextInputFormatterOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &LengthLimitingTextInputFormatter{MaxLength: maxLength, MaxLengthEnforcement: o.MaxLengthEnforcement}
}

// LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement is
// LengthLimitingTextInputFormatter.getDefaultMaxLengthEnforcement([platform])
// (omitted: defaultTargetPlatform).
func LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(platform ...foundation.TargetPlatform) MaxLengthEnforcement {
	if foundation.KIsWeb {
		return MaxLengthEnforcementTruncateAfterCompositionEnds
	}
	p := foundation.DefaultTargetPlatform()
	if len(platform) > 0 {
		p = platform[0]
	}
	switch p {
	case foundation.TargetPlatformAndroid, foundation.TargetPlatformWindows:
		return MaxLengthEnforcementEnforced
	default: // iOS, macOS, linux, fuchsia
		return MaxLengthEnforcementTruncateAfterCompositionEnds
	}
}

// LengthLimitingTextInputFormatterTruncate is
// LengthLimitingTextInputFormatter.truncate: value truncated to maxLength
// characters (grapheme clusters).
func LengthLimitingTextInputFormatterTruncate(value TextEditingValue, maxLength int) TextEditingValue {
	iterator := characters.NewCharacterRange(value.Text)
	if characters.StringCharacters(value.Text).Length() > maxLength {
		iterator.ExpandNext(maxLength)
	}
	truncated := iterator.Current()
	truncatedLength := dart.StringLength(truncated)
	baseOffset := min(value.Selection.Start, truncatedLength)
	extentOffset := min(value.Selection.End, truncatedLength)
	selection := value.Selection.CopyWith(TextSelectionCopyWithOptions{BaseOffset: &baseOffset, ExtentOffset: &extentOffset})
	composing := ui.TextRangeEmpty
	if !value.Composing.IsCollapsed() && truncatedLength > value.Composing.Start {
		composing = ui.NewTextRange(value.Composing.Start, min(value.Composing.End, truncatedLength))
	}
	return NewTextEditingValue(TextEditingValueOptions{Text: truncated, Selection: &selection, Composing: &composing})
}

// FormatEditUpdate limits newValue to MaxLength characters.
func (f *LengthLimitingTextInputFormatter) FormatEditUpdate(oldValue, newValue TextEditingValue) TextEditingValue {
	if f.MaxLength == nil || *f.MaxLength == -1 || characters.StringCharacters(newValue.Text).Length() <= *f.MaxLength {
		return newValue
	}
	maxLength := *f.MaxLength
	if !(maxLength > 0) {
		panic(dart.NewAssertionError(nil))
	}
	enforcement := LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement()
	if f.MaxLengthEnforcement != nil {
		enforcement = *f.MaxLengthEnforcement
	}
	switch enforcement {
	case MaxLengthEnforcementNone:
		return newValue
	case MaxLengthEnforcementEnforced:
		// If already at the maximum and tried to enter even more, and has no
		// selection, keep the old value.
		if characters.StringCharacters(oldValue.Text).Length() == maxLength && oldValue.Selection.IsCollapsed() {
			return oldValue
		}
		// Enforced to return a truncated value.
		return LengthLimitingTextInputFormatterTruncate(newValue, maxLength)
	default: // truncateAfterCompositionEnds
		// If already at the maximum and tried to enter even more, and the old
		// value is not composing, keep the old value.
		if characters.StringCharacters(oldValue.Text).Length() == maxLength && !oldValue.Composing.IsValid() {
			return oldValue
		}
		// Temporarily exempt newValue from the maxLength limit if it has a
		// composing text going and no enforcement to the composing value,
		// until the composing is finished.
		if newValue.Composing.IsValid() {
			return newValue
		}
		return LengthLimitingTextInputFormatterTruncate(newValue, maxLength)
	}
}
