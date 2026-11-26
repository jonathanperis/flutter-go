package services

// Port of packages/flutter/lib/src/services/text_input.dart (Flutter
// 3.50.1). Text is a Go string read as WTF-8 (src/dart's string.go);
// offsets are UTF-16 code units, as in Dart.

import (
	"fmt"
	"iter"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/vector_math"
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

// SmartDashesType indicates how to handle the intelligent replacement of
// dashes in text input.
type SmartDashesType int

const (
	// SmartDashesTypeDisabled: smart dashes is disabled.
	SmartDashesTypeDisabled SmartDashesType = iota
	// SmartDashesTypeEnabled: smart dashes is enabled.
	SmartDashesTypeEnabled
)

// SmartDashesTypeValues is SmartDashesType.values.
var SmartDashesTypeValues = []SmartDashesType{SmartDashesTypeDisabled, SmartDashesTypeEnabled}

// Name is the enum value's name.
func (t SmartDashesType) Name() string { return [...]string{"disabled", "enabled"}[t] }

func (t SmartDashesType) String() string { return "SmartDashesType." + t.Name() }

// SmartQuotesType indicates how to handle the intelligent replacement of
// quotes in text input.
type SmartQuotesType int

const (
	// SmartQuotesTypeDisabled: smart quotes is disabled.
	SmartQuotesTypeDisabled SmartQuotesType = iota
	// SmartQuotesTypeEnabled: smart quotes is enabled.
	SmartQuotesTypeEnabled
)

// SmartQuotesTypeValues is SmartQuotesType.values.
var SmartQuotesTypeValues = []SmartQuotesType{SmartQuotesTypeDisabled, SmartQuotesTypeEnabled}

// Name is the enum value's name.
func (t SmartQuotesType) Name() string { return [...]string{"disabled", "enabled"}[t] }

func (t SmartQuotesType) String() string { return "SmartQuotesType." + t.Name() }

// TextInputType is the type of information for which to optimize the text
// input control. Signed, Decimal and Password are nil (Dart's null) except
// for the number types. Compare with Equal (Dart's ==).
type TextInputType struct {
	// Index is the index of this type in TextInputTypeValues.
	Index int
	// Signed is whether a number keyboard sends signed numbers (nil: null).
	Signed *bool
	// Decimal is whether a number keyboard sends decimal numbers (nil:
	// null).
	Decimal *bool
	// Password is whether a number keyboard is for a numeric password
	// (nil: null).
	Password *bool
}

// TextInputTypeNumberWithOptionsOptions are numberWithOptions' named
// parameters (all false by default).
type TextInputTypeNumberWithOptionsOptions struct {
	Signed, Decimal, Password bool
}

// NewTextInputTypeNumberWithOptions is TextInputType.numberWithOptions: a
// numeric keyboard with optional sign, decimal point and password flags.
func NewTextInputTypeNumberWithOptions(opts ...TextInputTypeNumberWithOptionsOptions) TextInputType {
	var o TextInputTypeNumberWithOptionsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return TextInputType{Index: 2, Signed: &o.Signed, Decimal: &o.Decimal, Password: &o.Password}
}

// The TextInputType statics.
var (
	TextInputTypeText            = TextInputType{Index: 0}
	TextInputTypeMultiline       = TextInputType{Index: 1}
	TextInputTypeNumber          = NewTextInputTypeNumberWithOptions()
	TextInputTypePhone           = TextInputType{Index: 3}
	TextInputTypeDatetime        = TextInputType{Index: 4}
	TextInputTypeEmailAddress    = TextInputType{Index: 5}
	TextInputTypeURL             = TextInputType{Index: 6}
	TextInputTypeVisiblePassword = TextInputType{Index: 7}
	TextInputTypeName            = TextInputType{Index: 8}
	TextInputTypeStreetAddress   = TextInputType{Index: 9}
	TextInputTypeNone            = TextInputType{Index: 10}
	TextInputTypeWebSearch       = TextInputType{Index: 11}
	TextInputTypeTwitter         = TextInputType{Index: 12}
)

// TextInputTypeValues is TextInputType.values.
var TextInputTypeValues = []TextInputType{
	TextInputTypeText, TextInputTypeMultiline, TextInputTypeNumber, TextInputTypePhone, TextInputTypeDatetime,
	TextInputTypeEmailAddress, TextInputTypeURL, TextInputTypeVisiblePassword, TextInputTypeName,
	TextInputTypeStreetAddress, TextInputTypeNone, TextInputTypeWebSearch, TextInputTypeTwitter,
}

var textInputTypeNames = []string{
	"text", "multiline", "number", "phone", "datetime", "emailAddress", "url", "visiblePassword",
	"name", "address", "none", "webSearch", "twitter",
}

func (t TextInputType) name() string { return "TextInputType." + textInputTypeNames[t.Index] }

// ToJSON returns a representation of this object as a JSON object.
func (t TextInputType) ToJSON() []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{
		{Key: "name", Value: t.name()},
		{Key: "signed", Value: t.Signed},
		{Key: "decimal", Value: t.Decimal},
		{Key: "password", Value: t.Password},
	}
}

func nullableBool(b *bool) string {
	if b == nil {
		return "null"
	}
	return strconv.FormatBool(*b)
}

func (t TextInputType) String() string {
	return foundation.ObjectRuntimeType(t, "TextInputType") + "(name: " + t.name() + ", signed: " +
		nullableBool(t.Signed) + ", decimal: " + nullableBool(t.Decimal) + ", password: " + nullableBool(t.Password) + ")"
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Equal is TextInputType's ==.
func (t TextInputType) Equal(other TextInputType) bool {
	return other.Index == t.Index && boolPtrEqual(other.Signed, t.Signed) &&
		boolPtrEqual(other.Decimal, t.Decimal) && boolPtrEqual(other.Password, t.Password)
}

// TextInputAction is an action the user has requested the text input
// control to perform.
type TextInputAction int

// The TextInputAction values.
const (
	TextInputActionNone TextInputAction = iota
	TextInputActionUnspecified
	TextInputActionDone
	TextInputActionGo
	TextInputActionSearch
	TextInputActionSend
	TextInputActionNext
	TextInputActionPrevious
	TextInputActionContinueAction
	TextInputActionJoin
	TextInputActionRoute
	TextInputActionEmergencyCall
	TextInputActionNewline
)

// TextInputActionValues is TextInputAction.values.
var TextInputActionValues = []TextInputAction{
	TextInputActionNone, TextInputActionUnspecified, TextInputActionDone, TextInputActionGo,
	TextInputActionSearch, TextInputActionSend, TextInputActionNext, TextInputActionPrevious,
	TextInputActionContinueAction, TextInputActionJoin, TextInputActionRoute, TextInputActionEmergencyCall,
	TextInputActionNewline,
}

// Name is the enum value's name.
func (a TextInputAction) Name() string {
	return [...]string{"none", "unspecified", "done", "go", "search", "send", "next", "previous",
		"continueAction", "join", "route", "emergencyCall", "newline"}[a]
}

func (a TextInputAction) String() string { return "TextInputAction." + a.Name() }

// TextCapitalization configures how the platform keyboard will select an
// uppercase or lowercase keyboard.
type TextCapitalization int

// The TextCapitalization values.
const (
	TextCapitalizationWords TextCapitalization = iota
	TextCapitalizationSentences
	TextCapitalizationCharacters
	TextCapitalizationNone
)

// TextCapitalizationValues is TextCapitalization.values.
var TextCapitalizationValues = []TextCapitalization{
	TextCapitalizationWords, TextCapitalizationSentences, TextCapitalizationCharacters, TextCapitalizationNone,
}

// Name is the enum value's name.
func (c TextCapitalization) Name() string {
	return [...]string{"words", "sentences", "characters", "none"}[c]
}

func (c TextCapitalization) String() string { return "TextCapitalization." + c.Name() }

// emptyLocales is the default `const <Locale>[]` of hintLocales (one list
// shared by every configuration, as Dart's canonical constant).
var emptyLocales = make([]ui.Locale, 0, 1)

// TextInputConfiguration controls the visual appearance of the text input
// control. Build it with NewTextInputConfiguration; compare with Equal.
type TextInputConfiguration struct {
	// ViewID is the view the text input belongs to (nil: null).
	ViewID *int
	// InputType is the type of information for which to optimize the text
	// input control.
	InputType TextInputType
	// ReadOnly is whether the text field can be edited or not.
	ReadOnly bool
	// ObscureText is whether to hide the text being edited.
	ObscureText bool
	// Autocorrect is whether to enable autocorrection.
	Autocorrect bool
	// AutofillConfiguration is the configuration to use for autofill.
	AutofillConfiguration AutofillConfiguration
	// SmartDashesType is whether to allow the platform to automatically
	// format dashes.
	SmartDashesType SmartDashesType
	// SmartQuotesType is whether to allow the platform to automatically
	// format quotes.
	SmartQuotesType SmartQuotesType
	// EnableSuggestions is whether to show input suggestions as the user
	// types.
	EnableSuggestions bool
	// EnableInteractiveSelection is whether the user can change the text
	// selection.
	EnableInteractiveSelection bool
	// ActionLabel is what text to display in the text input control's
	// action button (nil: null).
	ActionLabel *string
	// InputAction is what kind of action to request for the action button
	// on the IME.
	InputAction TextInputAction
	// TextCapitalization specifies how platforms may automatically
	// capitalize text entered by the user.
	TextCapitalization TextCapitalization
	// KeyboardAppearance is the appearance of the keyboard (iOS only).
	KeyboardAppearance ui.Brightness
	// EnableIMEPersonalizedLearning is whether to enable that the IME
	// update personalized data such as typing history and user dictionary
	// data.
	EnableIMEPersonalizedLearning bool
	// AllowedMimeTypes are the MIME types that may be inserted as content.
	AllowedMimeTypes []string
	// HintLocales are the preferred locales of the text input (nil: null).
	// Compared by identity, as Dart's List ==.
	HintLocales []ui.Locale
	// EnableInlinePrediction is whether to enable inline predictions (nil:
	// null).
	EnableInlinePrediction *bool
	// EnableDeltaModel is whether to enable the delta model.
	EnableDeltaModel bool

	// allConfigurations are an autofill scope's configurations
	// (_AutofillScopeTextInputConfiguration, evaluated when encoded); nil
	// for a plain configuration.
	allConfigurations func() []TextInputConfiguration
}

// TextInputConfigurationOptions are TextInputConfiguration's named
// parameters. A nil pointer takes the Dart default (InputType text,
// Autocorrect/EnableSuggestions/EnableInteractiveSelection/
// EnableIMEPersonalizedLearning true, InputAction done, KeyboardAppearance
// light, TextCapitalization none, AutofillConfiguration disabled, the smart
// dashes/quotes types disabled when ObscureText and enabled otherwise, an
// empty HintLocales; a pointer to a nil slice is Dart's explicit null).
type TextInputConfigurationOptions struct {
	ViewID                        *int
	InputType                     *TextInputType
	ReadOnly                      bool
	ObscureText                   bool
	Autocorrect                   *bool
	SmartDashesType               *SmartDashesType
	SmartQuotesType               *SmartQuotesType
	EnableSuggestions             *bool
	EnableInteractiveSelection    *bool
	ActionLabel                   *string
	InputAction                   *TextInputAction
	KeyboardAppearance            *ui.Brightness
	TextCapitalization            *TextCapitalization
	AutofillConfiguration         *AutofillConfiguration
	EnableIMEPersonalizedLearning *bool
	AllowedMimeTypes              []string
	EnableDeltaModel              bool
	HintLocales                   *[]ui.Locale
	EnableInlinePrediction        *bool
}

func orDefault[T any](p *T, def T) T {
	if p != nil {
		return *p
	}
	return def
}

// NewTextInputConfiguration is TextInputConfiguration({...}).
func NewTextInputConfiguration(opts ...TextInputConfigurationOptions) TextInputConfiguration {
	var o TextInputConfigurationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dashes, quotes := SmartDashesTypeEnabled, SmartQuotesTypeEnabled
	if o.ObscureText {
		dashes, quotes = SmartDashesTypeDisabled, SmartQuotesTypeDisabled
	}
	mimeTypes := o.AllowedMimeTypes
	if mimeTypes == nil {
		mimeTypes = []string{}
	}
	return TextInputConfiguration{
		ViewID:                        o.ViewID,
		InputType:                     orDefault(o.InputType, TextInputTypeText),
		ReadOnly:                      o.ReadOnly,
		ObscureText:                   o.ObscureText,
		Autocorrect:                   orDefault(o.Autocorrect, true),
		SmartDashesType:               orDefault(o.SmartDashesType, dashes),
		SmartQuotesType:               orDefault(o.SmartQuotesType, quotes),
		EnableSuggestions:             orDefault(o.EnableSuggestions, true),
		EnableInteractiveSelection:    orDefault(o.EnableInteractiveSelection, true),
		ActionLabel:                   o.ActionLabel,
		InputAction:                   orDefault(o.InputAction, TextInputActionDone),
		KeyboardAppearance:            orDefault(o.KeyboardAppearance, ui.BrightnessLight),
		TextCapitalization:            orDefault(o.TextCapitalization, TextCapitalizationNone),
		AutofillConfiguration:         orDefault(o.AutofillConfiguration, AutofillConfigurationDisabled),
		EnableIMEPersonalizedLearning: orDefault(o.EnableIMEPersonalizedLearning, true),
		AllowedMimeTypes:              mimeTypes,
		EnableDeltaModel:              o.EnableDeltaModel,
		HintLocales:                   orDefault(o.HintLocales, emptyLocales),
		EnableInlinePrediction:        o.EnableInlinePrediction,
	}
}

// TextInputConfigurationCopyWithOptions are copyWith's parameters (nil:
// keep).
type TextInputConfigurationCopyWithOptions struct {
	ViewID                        *int
	InputType                     *TextInputType
	ReadOnly                      *bool
	ObscureText                   *bool
	Autocorrect                   *bool
	SmartDashesType               *SmartDashesType
	SmartQuotesType               *SmartQuotesType
	EnableSuggestions             *bool
	EnableInteractiveSelection    *bool
	ActionLabel                   *string
	InputAction                   *TextInputAction
	KeyboardAppearance            *ui.Brightness
	TextCapitalization            *TextCapitalization
	EnableIMEPersonalizedLearning *bool
	AllowedMimeTypes              []string
	AutofillConfiguration         *AutofillConfiguration
	EnableDeltaModel              *bool
	HintLocales                   []ui.Locale
	EnableInlinePrediction        *bool
}

// CopyWith creates a copy of this configuration with the given fields
// replaced (a plain TextInputConfiguration, as Dart's copyWith returns).
func (c TextInputConfiguration) CopyWith(opts ...TextInputConfigurationCopyWithOptions) TextInputConfiguration {
	var o TextInputConfigurationCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	out := c
	out.allConfigurations = nil
	if o.ViewID != nil {
		out.ViewID = o.ViewID
	}
	out.InputType = orDefault(o.InputType, c.InputType)
	out.ReadOnly = orDefault(o.ReadOnly, c.ReadOnly)
	out.ObscureText = orDefault(o.ObscureText, c.ObscureText)
	out.Autocorrect = orDefault(o.Autocorrect, c.Autocorrect)
	out.SmartDashesType = orDefault(o.SmartDashesType, c.SmartDashesType)
	out.SmartQuotesType = orDefault(o.SmartQuotesType, c.SmartQuotesType)
	out.EnableSuggestions = orDefault(o.EnableSuggestions, c.EnableSuggestions)
	out.EnableInteractiveSelection = orDefault(o.EnableInteractiveSelection, c.EnableInteractiveSelection)
	if o.ActionLabel != nil {
		out.ActionLabel = o.ActionLabel
	}
	out.InputAction = orDefault(o.InputAction, c.InputAction)
	out.TextCapitalization = orDefault(o.TextCapitalization, c.TextCapitalization)
	out.KeyboardAppearance = orDefault(o.KeyboardAppearance, c.KeyboardAppearance)
	out.EnableIMEPersonalizedLearning = orDefault(o.EnableIMEPersonalizedLearning, c.EnableIMEPersonalizedLearning)
	if o.AllowedMimeTypes != nil {
		out.AllowedMimeTypes = o.AllowedMimeTypes
	}
	out.AutofillConfiguration = orDefault(o.AutofillConfiguration, c.AutofillConfiguration)
	out.EnableDeltaModel = orDefault(o.EnableDeltaModel, c.EnableDeltaModel)
	if o.HintLocales != nil {
		out.HintLocales = o.HintLocales
	}
	if o.EnableInlinePrediction != nil {
		out.EnableInlinePrediction = o.EnableInlinePrediction
	}
	return out
}

func setEntry(m []dart.MapEntry[string, any], key string, value any) []dart.MapEntry[string, any] {
	for i := range m {
		if m[i].Key == key {
			m[i].Value = value
			return m
		}
	}
	return append(m, dart.MapEntry[string, any]{Key: key, Value: value})
}

// ToJSON returns a representation of this object as a JSON object (an
// ordered map, in Dart's key order).
func (c TextInputConfiguration) ToJSON() []dart.MapEntry[string, any] {
	autofill := c.AutofillConfiguration.ToJSON()
	m := []dart.MapEntry[string, any]{
		{Key: "viewId", Value: c.ViewID},
		{Key: "inputType", Value: c.InputType.ToJSON()},
		{Key: "readOnly", Value: c.ReadOnly},
		{Key: "obscureText", Value: c.ObscureText},
		{Key: "autocorrect", Value: c.Autocorrect},
		{Key: "smartDashesType", Value: strconv.Itoa(int(c.SmartDashesType))},
		{Key: "smartQuotesType", Value: strconv.Itoa(int(c.SmartQuotesType))},
		{Key: "enableSuggestions", Value: c.EnableSuggestions},
		{Key: "enableInteractiveSelection", Value: c.EnableInteractiveSelection},
		{Key: "actionLabel", Value: c.ActionLabel},
		{Key: "inputAction", Value: c.InputAction.String()},
		{Key: "textCapitalization", Value: c.TextCapitalization.String()},
		{Key: "keyboardAppearance", Value: c.KeyboardAppearance.String()},
		{Key: "enableIMEPersonalizedLearning", Value: c.EnableIMEPersonalizedLearning},
		{Key: "contentCommitMimeTypes", Value: c.AllowedMimeTypes},
	}
	if autofill != nil {
		m = append(m, dart.MapEntry[string, any]{Key: "autofill", Value: autofill})
	}
	var locales any
	if c.HintLocales != nil {
		tags := make([]string, len(c.HintLocales))
		for i, l := range c.HintLocales {
			tags[i] = l.ToLanguageTag()
		}
		locales = tags
	}
	m = append(m,
		dart.MapEntry[string, any]{Key: "enableDeltaModel", Value: c.EnableDeltaModel},
		dart.MapEntry[string, any]{Key: "hintLocales", Value: locales},
		dart.MapEntry[string, any]{Key: "enableInlinePrediction", Value: c.EnableInlinePrediction},
	)
	if c.allConfigurations != nil {
		// _AutofillScopeTextInputConfiguration.toJson.
		configurations := c.allConfigurations()
		fields := make([]any, len(configurations))
		for i, configuration := range configurations {
			fields[i] = configuration.ToJSON()
		}
		m = setEntry(m, "fields", fields)
	}
	return m
}

func intPtrEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func stringPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// sameList is Dart's == on two lists: identity (the same backing array
// and length; nil is null).
func sameList[T any](a, b []T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return len(a) == len(b) && unsafe.SliceData(a) == unsafe.SliceData(b)
}

// Equal is TextInputConfiguration's == (HintLocales compare by identity, as
// Dart's List ==; an autofill scope's configuration is a different type).
func (c TextInputConfiguration) Equal(other TextInputConfiguration) bool {
	if (c.allConfigurations == nil) != (other.allConfigurations == nil) {
		return false
	}
	return intPtrEqual(other.ViewID, c.ViewID) &&
		other.InputType.Equal(c.InputType) &&
		other.ReadOnly == c.ReadOnly &&
		other.ObscureText == c.ObscureText &&
		other.Autocorrect == c.Autocorrect &&
		other.SmartDashesType == c.SmartDashesType &&
		other.SmartQuotesType == c.SmartQuotesType &&
		other.EnableSuggestions == c.EnableSuggestions &&
		other.EnableInteractiveSelection == c.EnableInteractiveSelection &&
		stringPtrEqual(other.ActionLabel, c.ActionLabel) &&
		other.InputAction == c.InputAction &&
		other.KeyboardAppearance == c.KeyboardAppearance &&
		other.TextCapitalization == c.TextCapitalization &&
		other.AutofillConfiguration.Equal(c.AutofillConfiguration) &&
		other.EnableIMEPersonalizedLearning == c.EnableIMEPersonalizedLearning &&
		foundation.ListEquals(other.AllowedMimeTypes, c.AllowedMimeTypes) &&
		other.EnableDeltaModel == c.EnableDeltaModel &&
		sameList(other.HintLocales, c.HintLocales) &&
		boolPtrEqual(other.EnableInlinePrediction, c.EnableInlinePrediction)
}

func dartListString[T any](list []T, str func(T) string) string {
	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = str(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (c TextInputConfiguration) String() string {
	var d []string
	if c.ViewID != nil {
		d = append(d, "viewId: "+strconv.Itoa(*c.ViewID))
	}
	d = append(d,
		"inputType: "+c.InputType.String(),
		"readOnly: "+strconv.FormatBool(c.ReadOnly),
		"obscureText: "+strconv.FormatBool(c.ObscureText),
		"autocorrect: "+strconv.FormatBool(c.Autocorrect),
		"smartDashesType: "+c.SmartDashesType.String(),
		"smartQuotesType: "+c.SmartQuotesType.String(),
		"enableSuggestions: "+strconv.FormatBool(c.EnableSuggestions),
		"enableInteractiveSelection: "+strconv.FormatBool(c.EnableInteractiveSelection),
	)
	if c.ActionLabel != nil {
		d = append(d, "actionLabel: "+*c.ActionLabel)
	}
	d = append(d,
		"inputAction: "+c.InputAction.String(),
		"keyboardAppearance: "+c.KeyboardAppearance.String(),
		"textCapitalization: "+c.TextCapitalization.String(),
		"autofillConfiguration: "+c.AutofillConfiguration.String(),
		"enableIMEPersonalizedLearning: "+strconv.FormatBool(c.EnableIMEPersonalizedLearning),
		"allowedMimeTypes: "+dartListString(c.AllowedMimeTypes, func(s string) string { return s }),
		"enableDeltaModel: "+strconv.FormatBool(c.EnableDeltaModel),
	)
	if c.HintLocales != nil {
		d = append(d, "hintLocales: "+dartListString(c.HintLocales, ui.Locale.String))
	}
	if c.EnableInlinePrediction != nil {
		d = append(d, "enableInlinePrediction: "+strconv.FormatBool(*c.EnableInlinePrediction))
	}
	return "TextInputConfiguration(" + strings.Join(d, ", ") + ")"
}

// FloatingCursorDragState is the state of a "floating cursor" drag on an
// iOS soft keyboard.
type FloatingCursorDragState int

const (
	// FloatingCursorDragStateStart: the user has just started the drag.
	FloatingCursorDragStateStart FloatingCursorDragState = iota
	// FloatingCursorDragStateUpdate: the user is dragging.
	FloatingCursorDragStateUpdate
	// FloatingCursorDragStateEnd: the user has lifted the finger.
	FloatingCursorDragStateEnd
)

// FloatingCursorDragStateValues is FloatingCursorDragState.values.
var FloatingCursorDragStateValues = []FloatingCursorDragState{
	FloatingCursorDragStateStart, FloatingCursorDragStateUpdate, FloatingCursorDragStateEnd,
}

// Name is the enum value's name.
func (s FloatingCursorDragState) Name() string { return [...]string{"Start", "Update", "End"}[s] }

func (s FloatingCursorDragState) String() string { return "FloatingCursorDragState." + s.Name() }

// FloatingCursorStartLocation is RawFloatingCursorPoint.startLocation's
// record type `(Offset, TextPosition)`.
type FloatingCursorStartLocation struct {
	Offset   ui.Offset
	Position ui.TextPosition
}

// RawFloatingCursorPoint is the current state and position of the floating
// cursor.
type RawFloatingCursorPoint struct {
	// Offset is the raw position of the floating cursor (nil: null).
	Offset *ui.Offset
	// StartLocation is the position of the floating cursor's start (nil:
	// null).
	StartLocation *FloatingCursorStartLocation
	// State is the state of the floating cursor.
	State FloatingCursorDragState
}

// RawFloatingCursorPointOptions are RawFloatingCursorPoint's optional named
// parameters.
type RawFloatingCursorPointOptions struct {
	Offset        *ui.Offset
	StartLocation *FloatingCursorStartLocation
}

// NewRawFloatingCursorPoint is RawFloatingCursorPoint({offset,
// startLocation, required state}); an update needs an offset.
func NewRawFloatingCursorPoint(state FloatingCursorDragState, opts ...RawFloatingCursorPointOptions) RawFloatingCursorPoint {
	var o RawFloatingCursorPointOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if !(state != FloatingCursorDragStateUpdate || o.Offset != nil) {
		panic(dart.NewAssertionError(nil))
	}
	return RawFloatingCursorPoint{Offset: o.Offset, StartLocation: o.StartLocation, State: state}
}

// SelectionChangedCause indicates what triggered the change in selected
// text (including changes to the cursor location).
type SelectionChangedCause int

// The SelectionChangedCause values.
const (
	SelectionChangedCauseTap SelectionChangedCause = iota
	SelectionChangedCauseDoubleTap
	SelectionChangedCauseLongPress
	SelectionChangedCauseForcePress
	SelectionChangedCauseKeyboard
	SelectionChangedCauseToolbar
	SelectionChangedCauseDrag
	SelectionChangedCauseStylusHandwriting
)

// SelectionChangedCauseScribble is the deprecated
// SelectionChangedCause.scribble (stylusHandwriting).
const SelectionChangedCauseScribble = SelectionChangedCauseStylusHandwriting

// SelectionChangedCauseValues is SelectionChangedCause.values.
var SelectionChangedCauseValues = []SelectionChangedCause{
	SelectionChangedCauseTap, SelectionChangedCauseDoubleTap, SelectionChangedCauseLongPress,
	SelectionChangedCauseForcePress, SelectionChangedCauseKeyboard, SelectionChangedCauseToolbar,
	SelectionChangedCauseDrag, SelectionChangedCauseStylusHandwriting,
}

// Name is the enum value's name.
func (c SelectionChangedCause) Name() string {
	return [...]string{"tap", "doubleTap", "longPress", "forcePress", "keyboard", "toolbar", "drag",
		"stylusHandwriting"}[c]
}

func (c SelectionChangedCause) String() string { return "SelectionChangedCause." + c.Name() }

// TextSelectionDelegate is an interface for manipulating the selection, to
// be used by the implementer of the toolbar widget (mixin
// TextSelectionDelegate; embed TextSelectionDelegateMixin for the default
// getters).
type TextSelectionDelegate interface {
	TextEditingValue() TextEditingValue
	UserUpdateTextEditingValue(value TextEditingValue, cause SelectionChangedCause)
	// HideToolbar's Dart default hideHandles is true.
	HideToolbar(hideHandles ...bool)
	BringIntoView(position ui.TextPosition)
	CutEnabled() bool
	CopyEnabled() bool
	PasteEnabled() bool
	SelectAllEnabled() bool
	LookUpEnabled() bool
	SearchWebEnabled() bool
	ShareEnabled() bool
	LiveTextInputEnabled() bool
	CutSelection(cause SelectionChangedCause)
	PasteText(cause SelectionChangedCause) async.Future[async.Void]
	SelectAll(cause SelectionChangedCause)
	CopySelection(cause SelectionChangedCause)
}

// TextSelectionDelegateMixin holds TextSelectionDelegate's default getters.
type TextSelectionDelegateMixin struct{}

func (TextSelectionDelegateMixin) CutEnabled() bool           { return true }
func (TextSelectionDelegateMixin) CopyEnabled() bool          { return true }
func (TextSelectionDelegateMixin) PasteEnabled() bool         { return true }
func (TextSelectionDelegateMixin) SelectAllEnabled() bool     { return true }
func (TextSelectionDelegateMixin) LookUpEnabled() bool        { return true }
func (TextSelectionDelegateMixin) SearchWebEnabled() bool     { return true }
func (TextSelectionDelegateMixin) ShareEnabled() bool         { return true }
func (TextSelectionDelegateMixin) LiveTextInputEnabled() bool { return false }

// TextInputClient is an interface to receive information from TextInput
// (mixin TextInputClient; embed TextInputClientMixin for the default
// methods).
type TextInputClient interface {
	// CurrentTextEditingValue is the current state of the editing value
	// (nil: null).
	CurrentTextEditingValue() *TextEditingValue
	// CurrentAutofillScope is the AutofillScope this client belongs to (nil:
	// null).
	CurrentAutofillScope() AutofillScope
	UpdateEditingValue(value TextEditingValue)
	PerformAction(action TextInputAction)
	InsertContent(content KeyboardInsertedContent)
	PerformPrivateCommand(action string, data map[string]any)
	UpdateFloatingCursor(point RawFloatingCursorPoint)
	ShowAutocorrectionPromptRect(start, end int)
	OnFocusReceived() bool
	ConnectionClosed()
	DidChangeInputControl(oldControl, newControl TextInputControl)
	ShowToolbar()
	InsertTextPlaceholder(size ui.Size)
	RemoveTextPlaceholder()
	PerformSelector(selectorName string)
}

// TextInputClientMixin holds TextInputClient's default methods (all do
// nothing; OnFocusReceived returns false).
type TextInputClientMixin struct{}

func (TextInputClientMixin) InsertContent(KeyboardInsertedContent)                         {}
func (TextInputClientMixin) OnFocusReceived() bool                                         { return false }
func (TextInputClientMixin) DidChangeInputControl(oldControl, newControl TextInputControl) {}
func (TextInputClientMixin) ShowToolbar()                                                  {}
func (TextInputClientMixin) InsertTextPlaceholder(ui.Size)                                 {}
func (TextInputClientMixin) RemoveTextPlaceholder()                                        {}
func (TextInputClientMixin) PerformSelector(string)                                        {}

// ScribbleClient is an interface to receive focus from the engine (Scribble
// on iPadOS).
type ScribbleClient interface {
	ElementIdentifier() string
	OnScribbleFocus(offset ui.Offset)
	IsInScribbleRect(rect ui.Rect) bool
	Bounds() ui.Rect
}

// SelectionRect represents the geometry of a single glyph range or
// selectable range in a text field (a comparable value).
type SelectionRect struct {
	// Position is the position of this rect within the text.
	Position int
	// Bounds is the rectangle representing the bounds of this
	// SelectionRect.
	Bounds ui.Rect
	// Direction is the direction text flows within this rect.
	Direction ui.TextDirection
}

// SelectionRectOptions are SelectionRect's optional named parameters (nil
// Direction: ltr).
type SelectionRectOptions struct{ Direction *ui.TextDirection }

// NewSelectionRect is SelectionRect({required position, required bounds,
// direction = TextDirection.ltr}).
func NewSelectionRect(position int, bounds ui.Rect, opts ...SelectionRectOptions) SelectionRect {
	direction := ui.TextDirectionLtr
	if len(opts) > 0 && opts[0].Direction != nil {
		direction = *opts[0].Direction
	}
	return SelectionRect{Position: position, Bounds: bounds, Direction: direction}
}

func (r SelectionRect) String() string {
	return "SelectionRect(" + strconv.Itoa(r.Position) + ", " + r.Bounds.String() + ")"
}

// DeltaTextInputClient is a TextInputClient that receives the edits as
// TextEditingDeltas (mixin DeltaTextInputClient).
type DeltaTextInputClient interface {
	TextInputClient
	UpdateEditingValueWithDeltas(textEditingDeltas []TextEditingDelta)
}

// TextInputStyle is the text style the text input control should use
// (Diagnosticable; used by pointer, compared with Equal).
type TextInputStyle struct {
	foundation.DiagnosticableMixin
	// FontFamily (nil: null).
	FontFamily *string
	// FontSize (nil: null).
	FontSize *float64
	// FontWeight (nil: null).
	FontWeight *ui.FontWeight
	// TextDirection is the direction of the text.
	TextDirection ui.TextDirection
	// TextAlign is the alignment of the text.
	TextAlign ui.TextAlign
	// LetterSpacing (nil: null).
	LetterSpacing *float64
	// WordSpacing (nil: null).
	WordSpacing *float64
	// LineHeight (nil: null).
	LineHeight *float64
}

// TextInputStyleOptions are TextInputStyle's optional named parameters.
type TextInputStyleOptions struct {
	FontFamily    *string
	FontSize      *float64
	FontWeight    *ui.FontWeight
	LetterSpacing *float64
	WordSpacing   *float64
	LineHeight    *float64
}

// NewTextInputStyle is TextInputStyle({..., required textDirection,
// required textAlign, ...}).
func NewTextInputStyle(textDirection ui.TextDirection, textAlign ui.TextAlign, opts ...TextInputStyleOptions) *TextInputStyle {
	var o TextInputStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := &TextInputStyle{FontFamily: o.FontFamily, FontSize: o.FontSize, FontWeight: o.FontWeight,
		TextDirection: textDirection, TextAlign: textAlign, LetterSpacing: o.LetterSpacing,
		WordSpacing: o.WordSpacing, LineHeight: o.LineHeight}
	s.SetSelf(s)
	return s
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Equal is TextInputStyle's ==.
func (s *TextInputStyle) Equal(other *TextInputStyle) bool {
	if s == other {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return stringPtrEqual(other.FontFamily, s.FontFamily) &&
		floatPtrEqual(other.FontSize, s.FontSize) &&
		((other.FontWeight == nil) == (s.FontWeight == nil) && (s.FontWeight == nil || *other.FontWeight == *s.FontWeight)) &&
		other.TextDirection == s.TextDirection &&
		other.TextAlign == s.TextAlign &&
		floatPtrEqual(other.LetterSpacing, s.LetterSpacing) &&
		floatPtrEqual(other.WordSpacing, s.WordSpacing) &&
		floatPtrEqual(other.LineHeight, s.LineHeight)
}

// ToJSON returns a representation of this object as a JSON object.
func (s *TextInputStyle) ToJSON() []dart.MapEntry[string, any] {
	var fontWeightIndex *int
	if s.FontWeight != nil {
		i := s.FontWeight.Index()
		fontWeightIndex = &i
	}
	return []dart.MapEntry[string, any]{
		{Key: "fontFamily", Value: s.FontFamily},
		{Key: "fontSize", Value: s.FontSize},
		{Key: "fontWeightIndex", Value: fontWeightIndex},
		{Key: "textAlignIndex", Value: int(s.TextAlign)},
		{Key: "textDirectionIndex", Value: int(s.TextDirection)},
		{Key: "letterSpacing", Value: s.LetterSpacing},
		{Key: "wordSpacing", Value: s.WordSpacing},
		{Key: "lineHeight", Value: s.LineHeight},
	}
}

// DebugFillProperties adds the style's properties.
func (s *TextInputStyle) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewStringProperty("fontFamily", s.FontFamily, foundation.StringPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDoubleProperty("fontSize", s.FontSize, foundation.DoublePropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDiagnosticsProperty("fontWeight", s.FontWeight, foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewEnumProperty("textDirection", s.TextDirection))
	properties.Add(foundation.NewEnumProperty("textAlign", s.TextAlign))
	properties.Add(foundation.NewDoubleProperty("letterSpacing", s.LetterSpacing, foundation.DoublePropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDoubleProperty("wordSpacing", s.WordSpacing, foundation.DoublePropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDoubleProperty("lineHeight", s.LineHeight, foundation.DoublePropertyOptions{DefaultValue: foundation.Null}))
}

// TextInputConnection is an interface for interacting with a text input
// control (created by TextInputAttach; used by pointer).
type TextInputConnection struct {
	cachedSize           *ui.Size
	cachedTransform      *vector_math.Matrix4
	cachedRect           *ui.Rect
	cachedCaretRect      *ui.Rect
	cachedSelectionRects []SelectionRect

	id     int
	client TextInputClient
}

var textInputConnectionNextID = 1

func newTextInputConnection(client TextInputClient) *TextInputConnection {
	c := &TextInputConnection{id: textInputConnectionNextID, client: client, cachedSelectionRects: []SelectionRect{}}
	textInputConnectionNextID++
	return c
}

// TextInputConnectionDebugResetID is TextInputConnection.debugResetId({to
// = 1}): resets the internal ID counter for testing purposes (debug builds
// only).
func TextInputConnectionDebugResetID(to ...int) {
	if foundation.KDebugMode {
		textInputConnectionNextID = 1
		if len(to) > 0 {
			textInputConnectionNextID = to[0]
		}
	}
}

// Attached is whether this connection is currently interacting with a text
// input control.
func (c *TextInputConnection) Attached() bool { return textInputInstance().currentConnection == c }

// ScribbleInProgress is whether there is currently a Scribble interaction
// in progress.
func (c *TextInputConnection) ScribbleInProgress() bool {
	return textInputInstance().scribbleInProgress
}

func assertAttached(c *TextInputConnection) {
	if !c.Attached() {
		panic(dart.NewAssertionError(nil))
	}
}

// Show requests that the text input control become visible.
func (c *TextInputConnection) Show() {
	assertAttached(c)
	textInputInstance().show()
}

// RequestAutofill requests the platform autofill UI to appear.
func (c *TextInputConnection) RequestAutofill() {
	assertAttached(c)
	textInputInstance().requestAutofill()
}

// UpdateConfig requests that the text input control update itself
// according to the new TextInputConfiguration.
func (c *TextInputConnection) UpdateConfig(configuration TextInputConfiguration) {
	assertAttached(c)
	textInputInstance().updateConfig(configuration)
}

// SetEditingState requests that the text input control change its internal
// state to match the given state.
func (c *TextInputConnection) SetEditingState(value TextEditingValue) {
	assertAttached(c)
	textInputInstance().setEditingState(value)
}

// SetEditableSizeAndTransform sends the size and transform of the editable
// text to the platform, when either changed.
func (c *TextInputConnection) SetEditableSizeAndTransform(editableBoxSize ui.Size, transform *vector_math.Matrix4) {
	if c.cachedSize == nil || editableBoxSize != *c.cachedSize || !transform.Equal(c.cachedTransform) {
		c.cachedSize = &editableBoxSize
		c.cachedTransform = transform
		textInputInstance().setEditableSizeAndTransform(editableBoxSize, transform)
	}
}

// SetComposingRect sends the coordinates of the composing text's rect
// (a non-finite rect is sent as Offset.zero & Size(-1, -1)).
func (c *TextInputConnection) SetComposingRect(rect ui.Rect) {
	if c.cachedRect != nil && rect == *c.cachedRect {
		return
	}
	c.cachedRect = &rect
	validRect := rect
	if !rect.IsFinite() {
		validRect = ui.NewRectFromLTWH(0, 0, -1, -1)
	}
	textInputInstance().setComposingTextRect(validRect)
}

// SetCaretRect sends the coordinates of the caret's rect (a non-finite rect
// is sent as Offset.zero & Size(-1, -1)).
func (c *TextInputConnection) SetCaretRect(rect ui.Rect) {
	if c.cachedCaretRect != nil && rect == *c.cachedCaretRect {
		return
	}
	c.cachedCaretRect = &rect
	validRect := rect
	if !rect.IsFinite() {
		validRect = ui.NewRectFromLTWH(0, 0, -1, -1)
	}
	textInputInstance().setCaretRect(validRect)
}

// SetSelectionRects sends the selection rects, when they changed.
func (c *TextInputConnection) SetSelectionRects(selectionRects []SelectionRect) {
	if !foundation.ListEquals(c.cachedSelectionRects, selectionRects) {
		c.cachedSelectionRects = selectionRects
		textInputInstance().setSelectionRects(selectionRects)
	}
}

// SetStyle is the deprecated setStyle (use UpdateStyle); its required
// named parameters are positional.
func (c *TextInputConnection) SetStyle(fontFamily *string, fontSize *float64, fontWeight *ui.FontWeight, textDirection ui.TextDirection, textAlign ui.TextAlign) {
	c.UpdateStyle(NewTextInputStyle(textDirection, textAlign,
		TextInputStyleOptions{FontFamily: fontFamily, FontSize: fontSize, FontWeight: fontWeight}))
}

// UpdateStyle sends the text style of the editable text to the platform.
func (c *TextInputConnection) UpdateStyle(style *TextInputStyle) {
	assertAttached(c)
	textInputInstance().updateStyle(style)
}

// Close stops interacting with the text input control.
func (c *TextInputConnection) Close() {
	if c.Attached() {
		textInputInstance().clearClient()
	}
	if c.Attached() {
		panic(dart.NewAssertionError(nil))
	}
}

// ConnectionClosedReceived is called when the platform closed the
// connection.
func (c *TextInputConnection) ConnectionClosedReceived() {
	textInputInstance().currentConnection = nil
	if c.Attached() {
		panic(dart.NewAssertionError(nil))
	}
}

func toTextInputAction(action string) TextInputAction {
	switch action {
	case "TextInputAction.none":
		return TextInputActionNone
	case "TextInputAction.unspecified":
		return TextInputActionUnspecified
	case "TextInputAction.go":
		return TextInputActionGo
	case "TextInputAction.search":
		return TextInputActionSearch
	case "TextInputAction.send":
		return TextInputActionSend
	case "TextInputAction.next":
		return TextInputActionNext
	case "TextInputAction.previous":
		return TextInputActionPrevious
	case "TextInputAction.continueAction":
		return TextInputActionContinueAction
	case "TextInputAction.join":
		return TextInputActionJoin
	case "TextInputAction.route":
		return TextInputActionRoute
	case "TextInputAction.emergencyCall":
		return TextInputActionEmergencyCall
	case "TextInputAction.done":
		return TextInputActionDone
	case "TextInputAction.newline":
		return TextInputActionNewline
	}
	panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
		foundation.NewErrorSummary("Unknown text input action: " + action),
	}))
}

func toTextCursorAction(state string) FloatingCursorDragState {
	switch state {
	case "FloatingCursorDragState.start":
		return FloatingCursorDragStateStart
	case "FloatingCursorDragState.update":
		return FloatingCursorDragStateUpdate
	case "FloatingCursorDragState.end":
		return FloatingCursorDragStateEnd
	}
	panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
		foundation.NewErrorSummary("Unknown text cursor action: " + state),
	}))
}

// numToDouble is Dart's `(x as num).toDouble()` on a decoded number.
func numToDouble(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case float64:
		return x
	}
	panic(dart.TypeError{Message: fmt.Sprintf("type '%T' is not a subtype of type 'num' in type cast", v)})
}

func toTextPoint(state FloatingCursorDragState, encoded map[string]any) RawFloatingCursorPoint {
	if encoded["X"] == nil {
		panic(dart.NewAssertionError("You must provide a value for the horizontal location of the floating cursor."))
	}
	if encoded["Y"] == nil {
		panic(dart.NewAssertionError("You must provide a value for the vertical location of the floating cursor."))
	}
	offset := ui.OffsetZero
	if state == FloatingCursorDragStateUpdate {
		offset = ui.NewOffset(numToDouble(encoded["X"]), numToDouble(encoded["Y"]))
	}
	return NewRawFloatingCursorPoint(state, RawFloatingCursorPointOptions{Offset: &offset})
}

func reportTextInputError(exception any, stack dart.StackTrace, context string, informationCollector ...foundation.InformationCollector) {
	library := "services library"
	o := foundation.FlutterErrorDetailsOptions{
		Stack:   stack,
		Library: &library,
		Context: foundation.NewErrorDescription(context),
	}
	if len(informationCollector) > 0 {
		o.InformationCollector = informationCollector[0]
	}
	foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, o))
}

// TextInputChannel is the part of MethodChannel that TextInput uses (Dart's
// TextInput.setChannel takes any MethodChannel implementation, as tests'
// fakes are); *MethodChannel implements it.
type TextInputChannel interface {
	InvokeMethod(method string, arguments ...any) async.Future[any]
	SetMethodCallHandler(handler func(call MethodCall) async.Future[any])
}

// TextInput is the low-level interface to the system's text input control.
// Dart's class is statics around a private singleton: its statics are the
// TextInput... functions.
type TextInput struct {
	channel TextInputChannel

	currentControl TextInputControl
	inputControls  []TextInputControl // a LinkedHashSet: insertion order

	currentConnection    *TextInputConnection
	currentConfiguration TextInputConfiguration
	lastConnection       *TextInputConnection

	scribbleClientIDs  []string // insertion order of scribbleClients' keys
	scribbleClients    map[string]ScribbleClient
	scribbleInProgress bool

	hidePending bool
}

var textInputSingleton *TextInput

// textInputInstance is TextInput._instance (created on first use, as
// Dart's static final: it registers the method call handler).
func textInputInstance() *TextInput {
	if textInputSingleton == nil {
		t := &TextInput{
			currentControl:  platformTextInputControlInstance,
			inputControls:   []TextInputControl{platformTextInputControlInstance},
			scribbleClients: map[string]ScribbleClient{},
		}
		textInputSingleton = t
		t.channel = SystemChannels.TextInput
		t.channel.SetMethodCallHandler(t.loudlyHandleTextInputInvocation)
	}
	return textInputSingleton
}

// TextInputSetChannel is TextInput.setChannel: sets the channel TextInput
// uses (tests' fakes; debug builds only).
func TextInputSetChannel(newChannel TextInputChannel) {
	if foundation.KDebugMode {
		t := textInputInstance()
		t.channel = newChannel
		newChannel.SetMethodCallHandler(t.loudlyHandleTextInputInvocation)
	}
}

func textInputAddInputControl(control TextInputControl) {
	t := textInputInstance()
	if control != TextInputControl(platformTextInputControlInstance) && !slices.Contains(t.inputControls, control) {
		t.inputControls = append(t.inputControls, control)
	}
}

func textInputRemoveInputControl(control TextInputControl) {
	t := textInputInstance()
	if control != TextInputControl(platformTextInputControlInstance) {
		if i := slices.Index(t.inputControls, control); i >= 0 {
			t.inputControls = slices.Delete(t.inputControls, i, i+1)
		}
	}
}

// TextInputSetInputControl is TextInput.setInputControl: sets the current
// text input control (nil: none).
func TextInputSetInputControl(newControl TextInputControl) {
	t := textInputInstance()
	oldControl := t.currentControl
	if newControl == oldControl {
		return
	}
	if newControl != nil {
		textInputAddInputControl(newControl)
	}
	if oldControl != nil {
		textInputRemoveInputControl(oldControl)
	}
	t.currentControl = newControl
	if t.currentConnection != nil {
		t.currentConnection.client.DidChangeInputControl(oldControl, newControl)
	}
}

// TextInputRestorePlatformInputControl is
// TextInput.restorePlatformInputControl.
func TextInputRestorePlatformInputControl() {
	TextInputSetInputControl(platformTextInputControlInstance)
}

var textInputAndroidSupportedInputActions = []TextInputAction{
	TextInputActionNone, TextInputActionUnspecified, TextInputActionDone, TextInputActionSend,
	TextInputActionGo, TextInputActionSearch, TextInputActionNext, TextInputActionPrevious,
	TextInputActionNewline,
}

var textInputIOSSupportedInputActions = []TextInputAction{
	TextInputActionUnspecified, TextInputActionDone, TextInputActionSend, TextInputActionGo,
	TextInputActionSearch, TextInputActionNext, TextInputActionNewline, TextInputActionContinueAction,
	TextInputActionJoin, TextInputActionRoute, TextInputActionEmergencyCall,
}

// TextInputEnsureInitialized is TextInput.ensureInitialized: creates the
// singleton (registering its method call handler).
func TextInputEnsureInitialized() { textInputInstance() }

// TextInputAttach is TextInput.attach: begins interacting with the text
// input control.
func TextInputAttach(client TextInputClient, configuration TextInputConfiguration) *TextInputConnection {
	connection := newTextInputConnection(client)
	textInputInstance().attach(connection, configuration)
	return connection
}

func (t *TextInput) attach(connection *TextInputConnection, configuration TextInputConfiguration) {
	debugEnsureInputActionWorksOnPlatform(configuration.InputAction)
	t.currentConnection = connection
	t.currentConfiguration = configuration
	t.lastConnection = connection
	t.setClient(connection.client, configuration)
}

// debugEnsureInputActionWorksOnPlatform checks the action against the host
// platform (dart:io's Platform: flutter-go runs on macOS, where nothing is
// checked).
func debugEnsureInputActionWorksOnPlatform(inputAction TextInputAction) {
	if !foundation.KDebugMode {
		return
	}
	switch runtime.GOOS {
	case "ios":
		if !slices.Contains(textInputIOSSupportedInputActions, inputAction) {
			panic(dart.NewAssertionError("The requested TextInputAction \"" + inputAction.String() + "\" is not supported on iOS."))
		}
	case "android":
		if !slices.Contains(textInputAndroidSupportedInputActions, inputAction) {
			panic(dart.NewAssertionError("The requested TextInputAction \"" + inputAction.String() + "\" is not supported on Android."))
		}
	}
}

// TextInputScribbleClients is TextInput.scribbleClients (for testing): the
// registered Scribble clients by element identifier.
func TextInputScribbleClients() map[string]ScribbleClient { return textInputInstance().scribbleClients }

var errorPropertyStyle = foundation.DiagnosticsTreeStyleErrorProperty

func (t *TextInput) loudlyHandleTextInputInvocation(call MethodCall) async.Future[any] {
	return async.Async(func() any {
		defer func() {
			if exception := recover(); exception != nil {
				stack := dart.StackTraceFromPanic()
				reportTextInputError(exception, stack, "during method call "+call.Method,
					func() iter.Seq[foundation.DiagnosticsNode] {
						return slices.Values([]foundation.DiagnosticsNode{
							foundation.NewDiagnosticsProperty("call", call,
								foundation.DiagnosticsPropertyOptions{Style: &errorPropertyStyle}),
						})
					})
				panic(exception)
			}
		}()
		return async.Async(func() any { return t.handleTextInputInvocation(call) }).Await()
	})
}

func (t *TextInput) handleTextInputInvocation(methodCall MethodCall) any {
	method := methodCall.Method
	switch method {
	case "TextInputClient.focusElement":
		args := methodCall.Arguments.([]any)
		if c := t.scribbleClients[args[0].(string)]; c != nil {
			c.OnScribbleFocus(ui.NewOffset(numToDouble(args[1]), numToDouble(args[2])))
		}
		return nil
	case "TextInputClient.requestElementsInRect":
		raw := methodCall.Arguments.([]any)
		args := make([]float64, len(raw))
		for i, v := range raw {
			args[i] = numToDouble(v)
		}
		result := []any{}
		for _, elementIdentifier := range t.scribbleClientIDs {
			rect := ui.NewRectFromLTWH(args[0], args[1], args[2], args[3])
			client := t.scribbleClients[elementIdentifier]
			if client == nil || !client.IsInScribbleRect(rect) {
				continue
			}
			bounds := client.Bounds()
			if bounds == ui.RectZero || bounds.HasNaN() || bounds.IsInfinite() {
				continue
			}
			result = append(result, []any{elementIdentifier, bounds.Left, bounds.Top, bounds.Width(), bounds.Height()})
		}
		return result
	case "TextInputClient.scribbleInteractionBegan":
		t.scribbleInProgress = true
		return nil
	case "TextInputClient.scribbleInteractionFinished":
		t.scribbleInProgress = false
		return nil
	case "TextInputClient.onFocusReceived":
		args := methodCall.Arguments.([]any)
		clientID := args[0].(int)
		if t.lastConnection != nil && t.lastConnection.id == clientID {
			return t.lastConnection.client.OnFocusReceived()
		}
		return false
	}
	if t.currentConnection == nil {
		return nil
	}

	// The requestExistingInputState request needs to be handled regardless
	// of the client ID, as long as we have a _currentConnection.
	if method == "TextInputClient.requestExistingInputState" {
		t.attach(t.currentConnection, t.currentConfiguration)
		if editingValue := t.currentConnection.client.CurrentTextEditingValue(); editingValue != nil {
			t.setEditingState(*editingValue)
		}
		return nil
	}

	args := methodCall.Arguments.([]any)

	// The updateEditingStateWithTag request (autofill) can come up even to a
	// text field that doesn't have a connection.
	if method == "TextInputClient.updateEditingStateWithTag" {
		client := t.currentConnection.client
		scope := client.CurrentAutofillScope()
		editingValue := args[1].(map[string]any)
		// Decoded JSON maps lose their key order: the tags are visited in
		// sorted order.
		for _, tag := range slices.Sorted(maps.Keys(editingValue)) {
			textEditingValue := NewTextEditingValueFromJSON(editingValue[tag].(map[string]any))
			if scope == nil {
				continue
			}
			if client := scope.GetAutofillClient(tag); client != nil &&
				client.TextInputConfiguration().AutofillConfiguration.Enabled {
				client.Autofill(textEditingValue)
			}
		}
		return nil
	}

	client := args[0].(int)
	if client != t.currentConnection.id {
		// If the client IDs don't match, the incoming message was for a
		// different client. In debug builds "-1" is a magical client ID
		// that ignores this verification step.
		if !(foundation.KDebugMode && client == -1) {
			return nil
		}
	}

	switch method {
	case "TextInputClient.updateEditingState":
		value := NewTextEditingValueFromJSON(args[1].(map[string]any))
		t.updateEditingValue(value, platformTextInputControlInstance)
	case "TextInputClient.updateEditingStateWithDeltas":
		deltaClient, ok := t.currentConnection.client.(DeltaTextInputClient)
		if !ok {
			panic(dart.NewAssertionError("You must be using a DeltaTextInputClient if TextInputConfiguration.enableDeltaModel is set to true"))
		}
		encoded := args[1].(map[string]any)
		deltas := []TextEditingDelta{}
		for _, encodedDelta := range encoded["deltas"].([]any) {
			deltas = append(deltas, NewTextEditingDeltaFromJSON(encodedDelta.(map[string]any)))
		}
		deltaClient.UpdateEditingValueWithDeltas(deltas)
	case "TextInputClient.performAction":
		if args[1].(string) == "TextInputAction.commitContent" {
			content := NewKeyboardInsertedContentFromJSON(args[2].(map[string]any))
			t.currentConnection.client.InsertContent(content)
		} else {
			t.currentConnection.client.PerformAction(toTextInputAction(args[1].(string)))
		}
	case "TextInputClient.performSelectors":
		raw := args[1].([]any)
		selectors := make([]string, len(raw))
		for i, s := range raw {
			selectors[i] = s.(string)
		}
		for _, selector := range selectors {
			t.currentConnection.client.PerformSelector(selector)
		}
	case "TextInputClient.performPrivateCommand":
		firstArg := args[1].(map[string]any)
		data := map[string]any{}
		if firstArg["data"] != nil {
			data = firstArg["data"].(map[string]any)
		}
		t.currentConnection.client.PerformPrivateCommand(firstArg["action"].(string), data)
	case "TextInputClient.updateFloatingCursor":
		t.currentConnection.client.UpdateFloatingCursor(
			toTextPoint(toTextCursorAction(args[1].(string)), args[2].(map[string]any)))
	case "TextInputClient.onConnectionClosed":
		t.currentConnection.client.ConnectionClosed()
	case "TextInputClient.showAutocorrectionPromptRect":
		t.currentConnection.client.ShowAutocorrectionPromptRect(args[1].(int), args[2].(int))
	case "TextInputClient.showToolbar":
		t.currentConnection.client.ShowToolbar()
	case "TextInputClient.insertTextPlaceholder":
		t.currentConnection.client.InsertTextPlaceholder(ui.NewSize(numToDouble(args[1]), numToDouble(args[2])))
	case "TextInputClient.removeTextPlaceholder":
		t.currentConnection.client.RemoveTextPlaceholder()
	default:
		panic(NewMissingPluginException())
	}
	return nil
}

func (t *TextInput) scheduleHide() {
	if t.hidePending {
		return
	}
	t.hidePending = true

	// Schedule a deferred task that hides the text input. If someone else
	// shows the keyboard during this update cycle, then the task will do
	// nothing.
	async.ScheduleMicrotask(func() {
		t.hidePending = false
		if t.currentConnection == nil {
			t.hide()
		}
	})
}

// controls is a snapshot of the input controls to iterate.
func (t *TextInput) controls() []TextInputControl { return slices.Clone(t.inputControls) }

func (t *TextInput) setClient(client TextInputClient, configuration TextInputConfiguration) {
	for _, control := range t.controls() {
		control.Attach(client, configuration)
	}
}

func (t *TextInput) clearClient() {
	client := t.currentConnection.client
	for _, control := range t.controls() {
		control.Detach(client)
	}
	t.currentConnection = nil
	t.scheduleHide()
}

func (t *TextInput) updateConfig(configuration TextInputConfiguration) {
	for _, control := range t.controls() {
		control.UpdateConfig(configuration)
	}
}

func (t *TextInput) setEditingState(value TextEditingValue) {
	for _, control := range t.controls() {
		control.SetEditingState(value)
	}
}

func (t *TextInput) show() {
	for _, control := range t.controls() {
		control.Show()
	}
}

func (t *TextInput) hide() {
	for _, control := range t.controls() {
		control.Hide()
	}
}

func (t *TextInput) setEditableSizeAndTransform(editableBoxSize ui.Size, transform *vector_math.Matrix4) {
	for _, control := range t.controls() {
		control.SetEditableSizeAndTransform(editableBoxSize, transform)
	}
}

func (t *TextInput) setComposingTextRect(rect ui.Rect) {
	for _, control := range t.controls() {
		control.SetComposingRect(rect)
	}
}

func (t *TextInput) setCaretRect(rect ui.Rect) {
	for _, control := range t.controls() {
		control.SetCaretRect(rect)
	}
}

func (t *TextInput) setSelectionRects(selectionRects []SelectionRect) {
	for _, control := range t.controls() {
		control.SetSelectionRects(selectionRects)
	}
}

func (t *TextInput) updateStyle(style *TextInputStyle) {
	for _, control := range t.controls() {
		control.UpdateStyle(style)
	}
}

func (t *TextInput) requestAutofill() {
	for _, control := range t.controls() {
		control.RequestAutofill()
	}
}

func (t *TextInput) updateEditingValue(value TextEditingValue, exclude TextInputControl) {
	if t.currentConnection == nil {
		return
	}
	for _, control := range t.controls() {
		if control != exclude {
			control.SetEditingState(value)
		}
	}
	t.currentConnection.client.UpdateEditingValue(value)
}

// TextInputUpdateEditingValue is TextInput.updateEditingValue: informs the
// current client of the new value, and every input control but the current
// one.
func TextInputUpdateEditingValue(value TextEditingValue) {
	t := textInputInstance()
	t.updateEditingValue(value, t.currentControl)
}

// TextInputFinishAutofillContextOptions are finishAutofillContext's named
// parameters (nil ShouldSave: true).
type TextInputFinishAutofillContextOptions struct{ ShouldSave *bool }

// TextInputFinishAutofillContext is TextInput.finishAutofillContext({bool
// shouldSave = true}).
func TextInputFinishAutofillContext(opts ...TextInputFinishAutofillContextOptions) {
	shouldSave := true
	if len(opts) > 0 && opts[0].ShouldSave != nil {
		shouldSave = *opts[0].ShouldSave
	}
	for _, control := range textInputInstance().controls() {
		control.FinishAutofillContext(shouldSave)
	}
}

// TextInputRegisterScribbleElement is TextInput.registerScribbleElement.
func TextInputRegisterScribbleElement(elementIdentifier string, scribbleClient ScribbleClient) {
	t := textInputInstance()
	if _, ok := t.scribbleClients[elementIdentifier]; !ok {
		t.scribbleClientIDs = append(t.scribbleClientIDs, elementIdentifier)
	}
	t.scribbleClients[elementIdentifier] = scribbleClient
}

// TextInputUnregisterScribbleElement is TextInput.unregisterScribbleElement.
func TextInputUnregisterScribbleElement(elementIdentifier string) {
	t := textInputInstance()
	if _, ok := t.scribbleClients[elementIdentifier]; ok {
		delete(t.scribbleClients, elementIdentifier)
		t.scribbleClientIDs = slices.DeleteFunc(t.scribbleClientIDs, func(id string) bool { return id == elementIdentifier })
	}
}

// TextInputControl is an interface for implementing text input controls
// that receive text editing state changes and visual input control
// requests (mixin TextInputControl; embed TextInputControlMixin for the
// default no-op methods). Controls compare by ==: use pointer types.
type TextInputControl interface {
	Attach(client TextInputClient, configuration TextInputConfiguration)
	Detach(client TextInputClient)
	Show()
	Hide()
	UpdateConfig(configuration TextInputConfiguration)
	SetEditingState(value TextEditingValue)
	SetEditableSizeAndTransform(editableBoxSize ui.Size, transform *vector_math.Matrix4)
	SetComposingRect(rect ui.Rect)
	SetCaretRect(rect ui.Rect)
	SetSelectionRects(selectionRects []SelectionRect)
	// SetStyle is deprecated (use UpdateStyle); its required named
	// parameters are positional.
	SetStyle(fontFamily *string, fontSize *float64, fontWeight *ui.FontWeight, textDirection ui.TextDirection, textAlign ui.TextAlign)
	UpdateStyle(style *TextInputStyle)
	RequestAutofill()
	// FinishAutofillContext's Dart default shouldSave is true.
	FinishAutofillContext(shouldSave bool)
}

// TextInputControlMixin holds TextInputControl's default methods (all do
// nothing).
type TextInputControlMixin struct{}

func (TextInputControlMixin) Attach(TextInputClient, TextInputConfiguration)            {}
func (TextInputControlMixin) Detach(TextInputClient)                                    {}
func (TextInputControlMixin) Show()                                                     {}
func (TextInputControlMixin) Hide()                                                     {}
func (TextInputControlMixin) UpdateConfig(TextInputConfiguration)                       {}
func (TextInputControlMixin) SetEditingState(TextEditingValue)                          {}
func (TextInputControlMixin) SetEditableSizeAndTransform(ui.Size, *vector_math.Matrix4) {}
func (TextInputControlMixin) SetComposingRect(ui.Rect)                                  {}
func (TextInputControlMixin) SetCaretRect(ui.Rect)                                      {}
func (TextInputControlMixin) SetSelectionRects([]SelectionRect)                         {}
func (TextInputControlMixin) UpdateStyle(*TextInputStyle)                               {}
func (TextInputControlMixin) RequestAutofill()                                          {}
func (TextInputControlMixin) FinishAutofillContext(bool)                                {}
func (TextInputControlMixin) SetStyle(*string, *float64, *ui.FontWeight, ui.TextDirection, ui.TextAlign) {
}

// platformTextInputControl is _PlatformTextInputControl: the control that
// talks to the platform over the text input channel.
type platformTextInputControl struct{ TextInputControlMixin }

var platformTextInputControlInstance = &platformTextInputControl{}

func (p *platformTextInputControl) channel() TextInputChannel { return textInputInstance().channel }

func (p *platformTextInputControl) configurationToJSON(configuration TextInputConfiguration) []dart.MapEntry[string, any] {
	json := configuration.ToJSON()
	if textInputInstance().currentControl != TextInputControl(platformTextInputControlInstance) {
		// The web engine also gets isMultiline (kIsWeb is false here).
		json = setEntry(json, "inputType", TextInputTypeNone.ToJSON())
	}
	return json
}

// invoke is `_channel.invokeMethod<void>(...).then((_) {}, onError: ...)`.
func (p *platformTextInputControl) invoke(context, method string, arguments ...any) {
	async.OnComplete(p.channel().InvokeMethod(method, arguments...), func(any) {}, func(err error) {
		reportTextInputError(err, nil, context)
	})
}

func (p *platformTextInputControl) Attach(client TextInputClient, configuration TextInputConfiguration) {
	p.invoke("while attaching the text input client", "TextInput.setClient",
		[]any{textInputInstance().currentConnection.id, p.configurationToJSON(configuration)})
}

func (p *platformTextInputControl) Detach(client TextInputClient) {
	p.invoke("while detaching the text input client", "TextInput.clearClient")
}

func (p *platformTextInputControl) UpdateConfig(configuration TextInputConfiguration) {
	p.invoke("while updating text input configuration", "TextInput.updateConfig", p.configurationToJSON(configuration))
}

func (p *platformTextInputControl) SetEditingState(value TextEditingValue) {
	p.invoke("while setting text input editing state", "TextInput.setEditingState", value.ToJSON())
}

func (p *platformTextInputControl) Show() {
	p.invoke("while showing the text input client", "TextInput.show")
}

func (p *platformTextInputControl) Hide() {
	p.invoke("while hiding the text input client", "TextInput.hide")
}

func (p *platformTextInputControl) SetEditableSizeAndTransform(editableBoxSize ui.Size, transform *vector_math.Matrix4) {
	p.invoke("while setting text input size and transform", "TextInput.setEditableSizeAndTransform",
		[]dart.MapEntry[string, any]{
			{Key: "width", Value: editableBoxSize.Width},
			{Key: "height", Value: editableBoxSize.Height},
			{Key: "transform", Value: slices.Clone(transform.Storage())},
		})
}

func rectMap(rect ui.Rect) []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{
		{Key: "width", Value: rect.Width()},
		{Key: "height", Value: rect.Height()},
		{Key: "x", Value: rect.Left},
		{Key: "y", Value: rect.Top},
	}
}

func (p *platformTextInputControl) SetComposingRect(rect ui.Rect) {
	p.invoke("while setting text input composing rect", "TextInput.setMarkedTextRect", rectMap(rect))
}

func (p *platformTextInputControl) SetCaretRect(rect ui.Rect) {
	p.invoke("while setting text input caret rect", "TextInput.setCaretRect", rectMap(rect))
}

func (p *platformTextInputControl) SetSelectionRects(selectionRects []SelectionRect) {
	rects := make([]any, len(selectionRects))
	for i, rect := range selectionRects {
		rects[i] = []any{rect.Bounds.Left, rect.Bounds.Top, rect.Bounds.Width(), rect.Bounds.Height(),
			rect.Position, int(rect.Direction)}
	}
	p.invoke("while setting text input selection rects", "TextInput.setSelectionRects", rects)
}

func (p *platformTextInputControl) SetStyle(fontFamily *string, fontSize *float64, fontWeight *ui.FontWeight, textDirection ui.TextDirection, textAlign ui.TextAlign) {
	p.UpdateStyle(NewTextInputStyle(textDirection, textAlign,
		TextInputStyleOptions{FontFamily: fontFamily, FontSize: fontSize, FontWeight: fontWeight}))
}

func (p *platformTextInputControl) UpdateStyle(style *TextInputStyle) {
	p.invoke("while updating text input style", "TextInput.setStyle", style.ToJSON())
}

func (p *platformTextInputControl) RequestAutofill() {
	p.invoke("while requesting autofill", "TextInput.requestAutofill")
}

func (p *platformTextInputControl) FinishAutofillContext(shouldSave bool) {
	p.invoke("while finishing autofill context", "TextInput.finishAutofillContext", shouldSave)
}

// SystemContextMenuController allows access to functionality related to
// the system-rendered context menu (used by pointer).
type SystemContextMenuController struct {
	foundation.DiagnosticableMixin

	// OnSystemHide is called when the system hides the menu (nil: null).
	OnSystemHide func()

	customActionCallbacks map[string]func()
	lastTargetRect        *ui.Rect
	lastItems             []IOSSystemContextMenuItemData
	hiddenBySystem        bool
	isDisposed            bool
}

var systemContextMenuControllerLastShown *SystemContextMenuController

// SystemContextMenuControllerOptions are SystemContextMenuController's
// named parameters.
type SystemContextMenuControllerOptions struct{ OnSystemHide func() }

// NewSystemContextMenuController is SystemContextMenuController({onSystemHide});
// it registers itself as ServicesBinding's systemContextMenuClient.
func NewSystemContextMenuController(opts ...SystemContextMenuControllerOptions) *SystemContextMenuController {
	c := &SystemContextMenuController{customActionCallbacks: map[string]func(){}}
	if len(opts) > 0 {
		c.OnSystemHide = opts[0].OnSystemHide
	}
	c.SetSelf(c)
	ServicesBindingSetSystemContextMenuClient(c)
	return c
}

// IsVisible is whether this controller's menu is currently shown (for
// testing).
func (c *SystemContextMenuController) IsVisible() bool {
	return c == systemContextMenuControllerLastShown && !c.hiddenBySystem
}

// HandleSystemHide is called when the system hides the menu
// (SystemContextMenuClient).
func (c *SystemContextMenuController) HandleSystemHide() {
	if c.isDisposed || !c.IsVisible() {
		panic(dart.NewAssertionError(nil))
	}
	if systemContextMenuControllerLastShown == c {
		systemContextMenuControllerLastShown = nil
	}
	c.hiddenBySystem = true
	clear(c.customActionCallbacks)
	if c.OnSystemHide != nil {
		c.OnSystemHide()
	}
}

// HandleCustomContextMenuAction is called when a custom item is tapped
// (SystemContextMenuClient).
func (c *SystemContextMenuController) HandleCustomContextMenuAction(callbackID string) {
	callback := c.customActionCallbacks[callbackID]
	if callback == nil {
		panic(dart.NewAssertionError("Custom action callback not found for id: " + callbackID + ". " +
			"This may indicate that the menu item was not properly registered."))
	}
	callback()
}

func targetRectMap(targetRect ui.Rect) []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{
		{Key: "x", Value: targetRect.Left},
		{Key: "y", Value: targetRect.Top},
		{Key: "width", Value: targetRect.Width()},
		{Key: "height", Value: targetRect.Height()},
	}
}

func toVoid(f async.Future[any]) async.Future[async.Void] {
	return async.Then(f, func(any) async.Void { return async.Void{} })
}

// Show shows the system context menu anchored on targetRect (deprecated:
// use ShowWithItems).
func (c *SystemContextMenuController) Show(targetRect ui.Rect) async.Future[async.Void] {
	if c.isDisposed {
		panic(dart.NewAssertionError(nil))
	}
	if textInputInstance().currentConnection == nil {
		panic(dart.NewAssertionError("Currently, the system context menu can only be shown for an active text input connection"))
	}
	// Don't show the same thing that's already being shown.
	last := systemContextMenuControllerLastShown
	if last != nil && last.IsVisible() && last.lastTargetRect != nil && *last.lastTargetRect == targetRect {
		return async.FutureValue(async.Void{})
	}
	if !(last == nil || last == c || !last.IsVisible()) {
		panic(dart.NewAssertionError("Attempted to show while another instance was still visible."))
	}
	ServicesBindingSetSystemContextMenuClient(c)
	c.lastTargetRect = &targetRect
	systemContextMenuControllerLastShown = c
	c.hiddenBySystem = false
	return toVoid(SystemChannels.Platform.InvokeMethod("ContextMenu.showSystemContextMenu",
		[]dart.MapEntry[string, any]{{Key: "targetRect", Value: targetRectMap(targetRect)}}))
}

func itemsEqual(a, b []IOSSystemContextMenuItemData) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !IOSSystemContextMenuItemDataEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// ShowWithItems shows the system context menu with the given items,
// anchored on targetRect.
func (c *SystemContextMenuController) ShowWithItems(targetRect ui.Rect, items []IOSSystemContextMenuItemData) async.Future[async.Void] {
	if c.isDisposed || len(items) == 0 {
		panic(dart.NewAssertionError(nil))
	}
	if textInputInstance().currentConnection == nil {
		panic(dart.NewAssertionError("Currently, the system context menu can only be shown for an active text input connection"))
	}
	// Don't show the same thing that's already being shown.
	last := systemContextMenuControllerLastShown
	if last != nil && last.IsVisible() && last.lastTargetRect != nil && *last.lastTargetRect == targetRect &&
		itemsEqual(last.lastItems, items) {
		return async.FutureValue(async.Void{})
	}
	if !(last == nil || last == c || !last.IsVisible()) {
		panic(dart.NewAssertionError("Attempted to show while another instance was still visible."))
	}
	ServicesBindingSetSystemContextMenuClient(c)

	clear(c.customActionCallbacks)
	for _, item := range items {
		if custom, ok := item.(*IOSSystemContextMenuItemDataCustom); ok {
			id := custom.CallbackID()
			if existing, ok := c.customActionCallbacks[id]; ok && !dart.FunctionsEqual(existing, custom.OnPressed) {
				panic(dart.NewAssertionError("Duplicate callback ID \"" + id + "\" with different callbacks. " +
					"Each custom menu item must have a unique ID or the same callback."))
			}
			c.customActionCallbacks[id] = custom.OnPressed
		}
	}
	itemsJSON := make([]any, len(items))
	for i, item := range items {
		itemsJSON[i] = item.json()
	}
	c.lastTargetRect = &targetRect
	c.lastItems = items
	systemContextMenuControllerLastShown = c
	c.hiddenBySystem = false
	return toVoid(SystemChannels.Platform.InvokeMethod("ContextMenu.showSystemContextMenu",
		[]dart.MapEntry[string, any]{
			{Key: "targetRect", Value: targetRectMap(targetRect)},
			{Key: "items", Value: itemsJSON},
		}))
}

// Hide hides this controller's system context menu, if it is shown.
func (c *SystemContextMenuController) Hide() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		if c.isDisposed {
			panic(dart.NewAssertionError(nil))
		}
		// This check prevents the instance from accidentally hiding some
		// other instance, since only one can be visible at a time.
		if c != systemContextMenuControllerLastShown {
			return
		}
		systemContextMenuControllerLastShown = nil
		ServicesBindingSetSystemContextMenuClient(nil)
		clear(c.customActionCallbacks)
		// This may be called unnecessarily in the case where the user has
		// already hidden the menu (for example by tapping the screen).
		SystemChannels.Platform.InvokeMethod("ContextMenu.hideSystemContextMenu").Await()
	})
}

// DebugFillProperties adds isVisible, onSystemHide, _hiddenBySystem and
// _isDisposed.
func (c *SystemContextMenuController) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	c.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("isVisible", c.IsVisible()))
	ifTrue, ifFalse, showName := "callback set", "callback null", true
	properties.Add(foundation.NewFlagProperty("onSystemHide", c.OnSystemHide != nil,
		foundation.FlagPropertyOptions{IfTrue: &ifTrue, IfFalse: &ifFalse, ShowName: &showName}))
	properties.Add(foundation.NewDiagnosticsProperty("_hiddenBySystem", c.hiddenBySystem))
	properties.Add(foundation.NewDiagnosticsProperty("_isDisposed", c.isDisposed))
}

// Dispose hides the menu and disposes this controller.
func (c *SystemContextMenuController) Dispose() {
	if c.isDisposed {
		panic(dart.NewAssertionError(nil))
	}
	c.Hide()
	c.isDisposed = true
}

// IOSSystemContextMenuItemData describes a context menu button to be
// rendered in the iOS system context menu (a sealed class: only this
// package's types implement it). Compare with
// IOSSystemContextMenuItemDataEqual.
type IOSSystemContextMenuItemData interface {
	// Title is the text to display to the user (nil: null).
	Title() *string
	// HashCode is Dart's hashCode (the title's), sent as the item's
	// callbackId.
	HashCode() int
	jsonType() string
	json() []dart.MapEntry[string, any]
}

func itemDataHashCode(title *string) int {
	if title == nil {
		return dart.NullHashCode
	}
	return dart.StringHashCode(*title)
}

func itemDataJSON(item IOSSystemContextMenuItemData) []dart.MapEntry[string, any] {
	m := []dart.MapEntry[string, any]{{Key: "callbackId", Value: item.HashCode()}}
	if title := item.Title(); title != nil {
		m = append(m, dart.MapEntry[string, any]{Key: "title", Value: *title})
	}
	return append(m, dart.MapEntry[string, any]{Key: "type", Value: item.jsonType()})
}

// IOSSystemContextMenuItemDataEqual is == on two items (the same type and
// title; custom items compare their title and callback).
func IOSSystemContextMenuItemDataEqual(a, b IOSSystemContextMenuItemData) bool {
	if ca, ok := a.(*IOSSystemContextMenuItemDataCustom); ok {
		cb, ok := b.(*IOSSystemContextMenuItemDataCustom)
		return ok && (ca == cb || ca.TitleText == cb.TitleText && dart.FunctionsEqual(ca.OnPressed, cb.OnPressed))
	}
	if fmt.Sprintf("%T", a) != fmt.Sprintf("%T", b) {
		return false
	}
	return stringPtrEqual(a.Title(), b.Title())
}

// IOSSystemContextMenuItemDataCopy is the copy button.
type IOSSystemContextMenuItemDataCopy struct{}

func (IOSSystemContextMenuItemDataCopy) Title() *string                       { return nil }
func (IOSSystemContextMenuItemDataCopy) HashCode() int                        { return itemDataHashCode(nil) }
func (IOSSystemContextMenuItemDataCopy) jsonType() string                     { return "copy" }
func (d IOSSystemContextMenuItemDataCopy) json() []dart.MapEntry[string, any] { return itemDataJSON(d) }

// IOSSystemContextMenuItemDataCut is the cut button.
type IOSSystemContextMenuItemDataCut struct{}

func (IOSSystemContextMenuItemDataCut) Title() *string                       { return nil }
func (IOSSystemContextMenuItemDataCut) HashCode() int                        { return itemDataHashCode(nil) }
func (IOSSystemContextMenuItemDataCut) jsonType() string                     { return "cut" }
func (d IOSSystemContextMenuItemDataCut) json() []dart.MapEntry[string, any] { return itemDataJSON(d) }

// IOSSystemContextMenuItemDataPaste is the paste button.
type IOSSystemContextMenuItemDataPaste struct{}

func (IOSSystemContextMenuItemDataPaste) Title() *string   { return nil }
func (IOSSystemContextMenuItemDataPaste) HashCode() int    { return itemDataHashCode(nil) }
func (IOSSystemContextMenuItemDataPaste) jsonType() string { return "paste" }
func (d IOSSystemContextMenuItemDataPaste) json() []dart.MapEntry[string, any] {
	return itemDataJSON(d)
}

// IOSSystemContextMenuItemDataSelectAll is the select all button.
type IOSSystemContextMenuItemDataSelectAll struct{}

func (IOSSystemContextMenuItemDataSelectAll) Title() *string   { return nil }
func (IOSSystemContextMenuItemDataSelectAll) HashCode() int    { return itemDataHashCode(nil) }
func (IOSSystemContextMenuItemDataSelectAll) jsonType() string { return "selectAll" }
func (d IOSSystemContextMenuItemDataSelectAll) json() []dart.MapEntry[string, any] {
	return itemDataJSON(d)
}

// IOSSystemContextMenuItemDataLiveText is the live text (capture text from
// camera) button.
type IOSSystemContextMenuItemDataLiveText struct{}

func (IOSSystemContextMenuItemDataLiveText) Title() *string   { return nil }
func (IOSSystemContextMenuItemDataLiveText) HashCode() int    { return itemDataHashCode(nil) }
func (IOSSystemContextMenuItemDataLiveText) jsonType() string { return "captureTextFromCamera" }
func (d IOSSystemContextMenuItemDataLiveText) json() []dart.MapEntry[string, any] {
	return itemDataJSON(d)
}

// titledItemData is the shared shape of the titled, Diagnosticable items.
type titledItemData struct {
	foundation.DiagnosticableMixin
	// TitleText is the item's title (Dart's `title`; Title is the
	// interface's nullable getter).
	TitleText string
}

func (d *titledItemData) Title() *string { return &d.TitleText }
func (d *titledItemData) HashCode() int  { return dart.StringHashCode(d.TitleText) }

func (d *titledItemData) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewStringProperty("title", d.TitleText))
}

// IOSSystemContextMenuItemDataLookUp is the look up button.
type IOSSystemContextMenuItemDataLookUp struct{ titledItemData }

// NewIOSSystemContextMenuItemDataLookUp is
// IOSSystemContextMenuItemDataLookUp({required title}).
func NewIOSSystemContextMenuItemDataLookUp(title string) *IOSSystemContextMenuItemDataLookUp {
	d := &IOSSystemContextMenuItemDataLookUp{titledItemData{TitleText: title}}
	d.SetSelf(d)
	return d
}

func (*IOSSystemContextMenuItemDataLookUp) jsonType() string { return "lookUp" }
func (d *IOSSystemContextMenuItemDataLookUp) json() []dart.MapEntry[string, any] {
	return itemDataJSON(d)
}

// IOSSystemContextMenuItemDataSearchWeb is the search web button.
type IOSSystemContextMenuItemDataSearchWeb struct{ titledItemData }

// NewIOSSystemContextMenuItemDataSearchWeb is
// IOSSystemContextMenuItemDataSearchWeb({required title}).
func NewIOSSystemContextMenuItemDataSearchWeb(title string) *IOSSystemContextMenuItemDataSearchWeb {
	d := &IOSSystemContextMenuItemDataSearchWeb{titledItemData{TitleText: title}}
	d.SetSelf(d)
	return d
}

func (*IOSSystemContextMenuItemDataSearchWeb) jsonType() string { return "searchWeb" }
func (d *IOSSystemContextMenuItemDataSearchWeb) json() []dart.MapEntry[string, any] {
	return itemDataJSON(d)
}

// IOSSystemContextMenuItemDataShare is the share button.
type IOSSystemContextMenuItemDataShare struct{ titledItemData }

// NewIOSSystemContextMenuItemDataShare is
// IOSSystemContextMenuItemDataShare({required title}).
func NewIOSSystemContextMenuItemDataShare(title string) *IOSSystemContextMenuItemDataShare {
	d := &IOSSystemContextMenuItemDataShare{titledItemData{TitleText: title}}
	d.SetSelf(d)
	return d
}

func (*IOSSystemContextMenuItemDataShare) jsonType() string { return "share" }
func (d *IOSSystemContextMenuItemDataShare) json() []dart.MapEntry[string, any] {
	return itemDataJSON(d)
}

// IOSSystemContextMenuItemDataCustom is a custom button whose OnPressed is
// called when it is tapped.
type IOSSystemContextMenuItemDataCustom struct {
	titledItemData
	// OnPressed is called when the item is tapped.
	OnPressed func()
}

// NewIOSSystemContextMenuItemDataCustom is
// IOSSystemContextMenuItemDataCustom({required title, required onPressed}).
func NewIOSSystemContextMenuItemDataCustom(title string, onPressed func()) *IOSSystemContextMenuItemDataCustom {
	d := &IOSSystemContextMenuItemDataCustom{titledItemData: titledItemData{TitleText: title}, OnPressed: onPressed}
	d.SetSelf(d)
	return d
}

// HashCode is Object.hash(title, onPressed); the callback's hash is
// flutter-go's identity hash, so the value differs from Dart's.
func (d *IOSSystemContextMenuItemDataCustom) HashCode() int {
	return (dart.StringHashCode(d.TitleText)*31 + dart.IdentityHashCode(d.OnPressed)) & 0x3FFFFFFF
}

// CallbackID is the item's callback identifier, its hash code as a string.
func (d *IOSSystemContextMenuItemDataCustom) CallbackID() string { return strconv.Itoa(d.HashCode()) }

func (*IOSSystemContextMenuItemDataCustom) jsonType() string { return "custom" }

func (d *IOSSystemContextMenuItemDataCustom) json() []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{
		{Key: "id", Value: d.CallbackID()},
		{Key: "title", Value: d.TitleText},
		{Key: "type", Value: d.jsonType()},
	}
}

// DebugFillProperties adds title, callbackId and onPressed.
func (d *IOSSystemContextMenuItemDataCustom) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.titledItemData.DebugFillProperties(properties)
	properties.Add(foundation.NewStringProperty("callbackId", d.CallbackID()))
	properties.Add(foundation.NewDiagnosticsProperty("onPressed", d.OnPressed))
}
