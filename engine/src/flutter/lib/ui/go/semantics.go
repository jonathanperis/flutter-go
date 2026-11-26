package ui

// Port of engine/src/flutter/lib/ui/semantics.dart (Flutter 3.50.1):
// SemanticsAction, SemanticsRole, SemanticsInputType, SemanticsFlag,
// CheckedState, Tristate, SemanticsHitTestBehavior, SemanticsFlags,
// SemanticsValidationResult, StringAttribute (SpellOutStringAttribute,
// LocaleStringAttribute), SemanticsUpdateBuilder and SemanticsUpdate.
//
// The natives (NativeSemanticsFlags::initSemanticsFlags,
// NativeStringAttribute::init*, SemanticsUpdateBuilder::updateNode/
// updateCustomAction/build) build the engine's C++ copies of these values,
// which the engine hands to the platform's accessibility bridge when a
// view's updateSemantics is called. ENGINE GAP: flutter-go has no
// accessibility bridge, so nothing consumes an update; the builder records
// every call instead, and EngineSemanticsUpdateOf returns what was built
// (as flutter_test's semantics tooling inspects it), so the framework's
// output can be tested.

import (
	"math"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// SemanticsAction is an action that can be performed on a semantics node:
// a bit (Index) and a name. Compare with ==; the zero SemanticsAction is
// no action (it stands for Dart's null where an action is optional).
type SemanticsAction struct {
	index int
	name  string
}

// Index is the numerical value for this action: a single bit.
func (a SemanticsAction) Index() int { return a.index }

// Name is a human-readable name for this action.
func (a SemanticsAction) Name() string { return a.name }

func (a SemanticsAction) String() string { return "SemanticsAction." + a.name }

const (
	kTapIndex                           = 1 << 0
	kLongPressIndex                     = 1 << 1
	kScrollLeftIndex                    = 1 << 2
	kScrollRightIndex                   = 1 << 3
	kScrollUpIndex                      = 1 << 4
	kScrollDownIndex                    = 1 << 5
	kIncreaseIndex                      = 1 << 6
	kDecreaseIndex                      = 1 << 7
	kShowOnScreenIndex                  = 1 << 8
	kMoveCursorForwardByCharacterIndex  = 1 << 9
	kMoveCursorBackwardByCharacterIndex = 1 << 10
	kSetSelectionIndex                  = 1 << 11
	kCopyIndex                          = 1 << 12
	kCutIndex                           = 1 << 13
	kPasteIndex                         = 1 << 14
	kDidGainAccessibilityFocusIndex     = 1 << 15
	kDidLoseAccessibilityFocusIndex     = 1 << 16
	kCustomActionIndex                  = 1 << 17
	kDismissIndex                       = 1 << 18
	kMoveCursorForwardByWordIndex       = 1 << 19
	kMoveCursorBackwardByWordIndex      = 1 << 20
	kSetTextIndex                       = 1 << 21
	kFocusIndex                         = 1 << 22
	kScrollToOffsetIndex                = 1 << 23
	kExpandIndex                        = 1 << 24
	kCollapseIndex                      = 1 << 25
)

// The semantics actions (SemanticsAction.tap, ...).
var (
	SemanticsActionTap                           = SemanticsAction{kTapIndex, "tap"}
	SemanticsActionLongPress                     = SemanticsAction{kLongPressIndex, "longPress"}
	SemanticsActionScrollLeft                    = SemanticsAction{kScrollLeftIndex, "scrollLeft"}
	SemanticsActionScrollRight                   = SemanticsAction{kScrollRightIndex, "scrollRight"}
	SemanticsActionScrollUp                      = SemanticsAction{kScrollUpIndex, "scrollUp"}
	SemanticsActionScrollDown                    = SemanticsAction{kScrollDownIndex, "scrollDown"}
	SemanticsActionScrollToOffset                = SemanticsAction{kScrollToOffsetIndex, "scrollToOffset"}
	SemanticsActionIncrease                      = SemanticsAction{kIncreaseIndex, "increase"}
	SemanticsActionDecrease                      = SemanticsAction{kDecreaseIndex, "decrease"}
	SemanticsActionShowOnScreen                  = SemanticsAction{kShowOnScreenIndex, "showOnScreen"}
	SemanticsActionMoveCursorForwardByCharacter  = SemanticsAction{kMoveCursorForwardByCharacterIndex, "moveCursorForwardByCharacter"}
	SemanticsActionMoveCursorBackwardByCharacter = SemanticsAction{kMoveCursorBackwardByCharacterIndex, "moveCursorBackwardByCharacter"}
	SemanticsActionSetText                       = SemanticsAction{kSetTextIndex, "setText"}
	SemanticsActionSetSelection                  = SemanticsAction{kSetSelectionIndex, "setSelection"}
	SemanticsActionCopy                          = SemanticsAction{kCopyIndex, "copy"}
	SemanticsActionCut                           = SemanticsAction{kCutIndex, "cut"}
	SemanticsActionPaste                         = SemanticsAction{kPasteIndex, "paste"}
	SemanticsActionDidGainAccessibilityFocus     = SemanticsAction{kDidGainAccessibilityFocusIndex, "didGainAccessibilityFocus"}
	SemanticsActionDidLoseAccessibilityFocus     = SemanticsAction{kDidLoseAccessibilityFocusIndex, "didLoseAccessibilityFocus"}
	SemanticsActionCustomAction                  = SemanticsAction{kCustomActionIndex, "customAction"}
	SemanticsActionDismiss                       = SemanticsAction{kDismissIndex, "dismiss"}
	SemanticsActionMoveCursorForwardByWord       = SemanticsAction{kMoveCursorForwardByWordIndex, "moveCursorForwardByWord"}
	SemanticsActionMoveCursorBackwardByWord      = SemanticsAction{kMoveCursorBackwardByWordIndex, "moveCursorBackwardByWord"}
	SemanticsActionFocus                         = SemanticsAction{kFocusIndex, "focus"}
	SemanticsActionExpand                        = SemanticsAction{kExpandIndex, "expand"}
	SemanticsActionCollapse                      = SemanticsAction{kCollapseIndex, "collapse"}
)

// semanticsActionsByID is _kActionById, in its (insertion) order.
var semanticsActionsByID = []SemanticsAction{
	SemanticsActionTap, SemanticsActionLongPress, SemanticsActionScrollLeft, SemanticsActionScrollRight,
	SemanticsActionScrollUp, SemanticsActionScrollDown, SemanticsActionScrollToOffset, SemanticsActionIncrease,
	SemanticsActionDecrease, SemanticsActionShowOnScreen, SemanticsActionMoveCursorForwardByCharacter,
	SemanticsActionMoveCursorBackwardByCharacter, SemanticsActionSetSelection, SemanticsActionCopy,
	SemanticsActionCut, SemanticsActionPaste, SemanticsActionDidGainAccessibilityFocus,
	SemanticsActionDidLoseAccessibilityFocus, SemanticsActionCustomAction, SemanticsActionDismiss,
	SemanticsActionMoveCursorForwardByWord, SemanticsActionMoveCursorBackwardByWord, SemanticsActionSetText,
	SemanticsActionFocus, SemanticsActionExpand, SemanticsActionCollapse,
}

// SemanticsActionValues is SemanticsAction.values: every action, in the
// engine's map order.
func SemanticsActionValues() []SemanticsAction {
	return append([]SemanticsAction(nil), semanticsActionsByID...)
}

// SemanticsActionFromIndex is SemanticsAction.fromIndex: the action whose
// bit is index, or nil.
func SemanticsActionFromIndex(index int) *SemanticsAction {
	for _, a := range semanticsActionsByID {
		if a.index == index {
			a := a
			return &a
		}
	}
	return nil
}

// SemanticsRole is an enum to describe the role of a semantics node.
type SemanticsRole int

// The semantics roles.
const (
	SemanticsRoleNone SemanticsRole = iota
	SemanticsRoleTab
	SemanticsRoleTabBar
	SemanticsRoleTabPanel
	SemanticsRoleDialog
	SemanticsRoleAlertDialog
	SemanticsRoleTable
	SemanticsRoleCell
	SemanticsRoleRow
	SemanticsRoleColumnHeader
	SemanticsRoleDragHandle
	SemanticsRoleSpinButton
	SemanticsRoleComboBox
	SemanticsRoleMenuBar
	SemanticsRoleMenu
	SemanticsRoleMenuItem
	SemanticsRoleMenuItemCheckbox
	SemanticsRoleMenuItemRadio
	SemanticsRoleList
	SemanticsRoleListItem
	SemanticsRoleForm
	SemanticsRoleTooltip
	SemanticsRoleLoadingSpinner
	SemanticsRoleProgressBar
	SemanticsRoleHotKey
	SemanticsRoleRadioGroup
	SemanticsRoleStatus
	SemanticsRoleAlert
	SemanticsRoleComplementary
	SemanticsRoleContentInfo
	SemanticsRoleMain
	SemanticsRoleNavigation
	SemanticsRoleRegion
	SemanticsRoleSlider
)

var semanticsRoleNames = [...]string{"none", "tab", "tabBar", "tabPanel", "dialog", "alertDialog", "table",
	"cell", "row", "columnHeader", "dragHandle", "spinButton", "comboBox", "menuBar", "menu", "menuItem",
	"menuItemCheckbox", "menuItemRadio", "list", "listItem", "form", "tooltip", "loadingSpinner",
	"progressBar", "hotKey", "radioGroup", "status", "alert", "complementary", "contentInfo", "main",
	"navigation", "region", "slider"}

// SemanticsRoleValues is SemanticsRole.values.
var SemanticsRoleValues = func() []SemanticsRole {
	v := make([]SemanticsRole, len(semanticsRoleNames))
	for i := range v {
		v[i] = SemanticsRole(i)
	}
	return v
}()

// Name is the enum value's name.
func (r SemanticsRole) Name() string { return semanticsRoleNames[r] }

func (r SemanticsRole) String() string { return "SemanticsRole." + semanticsRoleNames[r] }

// SemanticsInputType is the type of text input a text field accepts.
type SemanticsInputType int

// The semantics input types.
const (
	SemanticsInputTypeNone SemanticsInputType = iota
	SemanticsInputTypeText
	SemanticsInputTypeURL
	SemanticsInputTypePhone
	SemanticsInputTypeSearch
	SemanticsInputTypeEmail
)

var semanticsInputTypeNames = [...]string{"none", "text", "url", "phone", "search", "email"}

// SemanticsInputTypeValues is SemanticsInputType.values.
var SemanticsInputTypeValues = []SemanticsInputType{SemanticsInputTypeNone, SemanticsInputTypeText,
	SemanticsInputTypeURL, SemanticsInputTypePhone, SemanticsInputTypeSearch, SemanticsInputTypeEmail}

// Name is the enum value's name.
func (t SemanticsInputType) Name() string { return semanticsInputTypeNames[t] }

func (t SemanticsInputType) String() string {
	return "SemanticsInputType." + semanticsInputTypeNames[t]
}

// SemanticsFlag is a Boolean value that can be associated with a semantics
// node (deprecated in favor of SemanticsFlags): a bit and a name. Compare
// with ==.
type SemanticsFlag struct {
	index int
	name  string
}

// Index is the numerical value for this flag: a single bit.
func (f SemanticsFlag) Index() int { return f.index }

// Name is a human-readable name for this flag.
func (f SemanticsFlag) Name() string { return f.name }

func (f SemanticsFlag) String() string { return "SemanticsFlag." + f.name }

const (
	kHasCheckedStateIndex            = 1 << 0
	kIsCheckedIndex                  = 1 << 1
	kIsSelectedIndex                 = 1 << 2
	kIsButtonIndex                   = 1 << 3
	kIsTextFieldIndex                = 1 << 4
	kIsFocusedIndex                  = 1 << 5
	kHasEnabledStateIndex            = 1 << 6
	kIsEnabledIndex                  = 1 << 7
	kIsInMutuallyExclusiveGroupIndex = 1 << 8
	kIsHeaderIndex                   = 1 << 9
	kIsObscuredIndex                 = 1 << 10
	kScopesRouteIndex                = 1 << 11
	kNamesRouteIndex                 = 1 << 12
	kIsHiddenIndex                   = 1 << 13
	kIsImageIndex                    = 1 << 14
	kIsLiveRegionIndex               = 1 << 15
	kHasToggledStateIndex            = 1 << 16
	kIsToggledIndex                  = 1 << 17
	kHasImplicitScrollingIndex       = 1 << 18
	kIsMultilineIndex                = 1 << 19
	kIsReadOnlyIndex                 = 1 << 20
	kIsFocusableIndex                = 1 << 21
	kIsLinkIndex                     = 1 << 22
	kIsSliderIndex                   = 1 << 23
	kIsKeyboardKeyIndex              = 1 << 24
	kIsCheckStateMixedIndex          = 1 << 25
	kHasExpandedStateIndex           = 1 << 26
	kIsExpandedIndex                 = 1 << 27
	kHasSelectedStateIndex           = 1 << 28
	kHasRequiredStateIndex           = 1 << 29
	kIsRequiredIndex                 = 1 << 30
)

// The semantics flags (SemanticsFlag.hasCheckedState, ...).
var (
	SemanticsFlagHasCheckedState            = SemanticsFlag{kHasCheckedStateIndex, "hasCheckedState"}
	SemanticsFlagIsChecked                  = SemanticsFlag{kIsCheckedIndex, "isChecked"}
	SemanticsFlagIsCheckStateMixed          = SemanticsFlag{kIsCheckStateMixedIndex, "isCheckStateMixed"}
	SemanticsFlagHasSelectedState           = SemanticsFlag{kHasSelectedStateIndex, "hasSelectedState"}
	SemanticsFlagIsSelected                 = SemanticsFlag{kIsSelectedIndex, "isSelected"}
	SemanticsFlagIsButton                   = SemanticsFlag{kIsButtonIndex, "isButton"}
	SemanticsFlagIsTextField                = SemanticsFlag{kIsTextFieldIndex, "isTextField"}
	SemanticsFlagIsSlider                   = SemanticsFlag{kIsSliderIndex, "isSlider"}
	SemanticsFlagIsKeyboardKey              = SemanticsFlag{kIsKeyboardKeyIndex, "isKeyboardKey"}
	SemanticsFlagIsReadOnly                 = SemanticsFlag{kIsReadOnlyIndex, "isReadOnly"}
	SemanticsFlagIsLink                     = SemanticsFlag{kIsLinkIndex, "isLink"}
	SemanticsFlagIsFocusable                = SemanticsFlag{kIsFocusableIndex, "isFocusable"}
	SemanticsFlagIsFocused                  = SemanticsFlag{kIsFocusedIndex, "isFocused"}
	SemanticsFlagHasEnabledState            = SemanticsFlag{kHasEnabledStateIndex, "hasEnabledState"}
	SemanticsFlagIsEnabled                  = SemanticsFlag{kIsEnabledIndex, "isEnabled"}
	SemanticsFlagIsInMutuallyExclusiveGroup = SemanticsFlag{kIsInMutuallyExclusiveGroupIndex, "isInMutuallyExclusiveGroup"}
	SemanticsFlagIsHeader                   = SemanticsFlag{kIsHeaderIndex, "isHeader"}
	SemanticsFlagIsObscured                 = SemanticsFlag{kIsObscuredIndex, "isObscured"}
	SemanticsFlagIsMultiline                = SemanticsFlag{kIsMultilineIndex, "isMultiline"}
	SemanticsFlagScopesRoute                = SemanticsFlag{kScopesRouteIndex, "scopesRoute"}
	SemanticsFlagNamesRoute                 = SemanticsFlag{kNamesRouteIndex, "namesRoute"}
	SemanticsFlagIsHidden                   = SemanticsFlag{kIsHiddenIndex, "isHidden"}
	SemanticsFlagIsImage                    = SemanticsFlag{kIsImageIndex, "isImage"}
	SemanticsFlagIsLiveRegion               = SemanticsFlag{kIsLiveRegionIndex, "isLiveRegion"}
	SemanticsFlagHasToggledState            = SemanticsFlag{kHasToggledStateIndex, "hasToggledState"}
	SemanticsFlagIsToggled                  = SemanticsFlag{kIsToggledIndex, "isToggled"}
	SemanticsFlagHasImplicitScrolling       = SemanticsFlag{kHasImplicitScrollingIndex, "hasImplicitScrolling"}
	SemanticsFlagHasExpandedState           = SemanticsFlag{kHasExpandedStateIndex, "hasExpandedState"}
	SemanticsFlagIsExpanded                 = SemanticsFlag{kIsExpandedIndex, "isExpanded"}
	SemanticsFlagHasRequiredState           = SemanticsFlag{kHasRequiredStateIndex, "hasRequiredState"}
	SemanticsFlagIsRequired                 = SemanticsFlag{kIsRequiredIndex, "isRequired"}
)

// semanticsFlagsByID is _kFlagById, in its (insertion) order.
var semanticsFlagsByID = []SemanticsFlag{
	SemanticsFlagHasCheckedState, SemanticsFlagIsChecked, SemanticsFlagHasSelectedState, SemanticsFlagIsSelected,
	SemanticsFlagIsButton, SemanticsFlagIsTextField, SemanticsFlagIsFocused, SemanticsFlagHasEnabledState,
	SemanticsFlagIsEnabled, SemanticsFlagIsInMutuallyExclusiveGroup, SemanticsFlagIsHeader,
	SemanticsFlagIsObscured, SemanticsFlagScopesRoute, SemanticsFlagNamesRoute, SemanticsFlagIsHidden,
	SemanticsFlagIsImage, SemanticsFlagIsLiveRegion, SemanticsFlagHasToggledState, SemanticsFlagIsToggled,
	SemanticsFlagHasImplicitScrolling, SemanticsFlagIsMultiline, SemanticsFlagIsReadOnly,
	SemanticsFlagIsFocusable, SemanticsFlagIsLink, SemanticsFlagIsSlider, SemanticsFlagIsKeyboardKey,
	SemanticsFlagIsCheckStateMixed, SemanticsFlagHasExpandedState, SemanticsFlagIsExpanded,
	SemanticsFlagHasRequiredState, SemanticsFlagIsRequired,
}

// SemanticsFlagValues is SemanticsFlag.values: every flag, in the engine's
// map order.
func SemanticsFlagValues() []SemanticsFlag {
	return append([]SemanticsFlag(nil), semanticsFlagsByID...)
}

// SemanticsFlagFromIndex is SemanticsFlag.fromIndex: the flag whose bit is
// index, or nil.
func SemanticsFlagFromIndex(index int) *SemanticsFlag {
	for _, f := range semanticsFlagsByID {
		if f.index == index {
			f := f
			return &f
		}
	}
	return nil
}

// CheckedState is the possible checked states of a semantics node.
type CheckedState int

// The checked states (Value is the enum's value field).
const (
	CheckedStateNone CheckedState = iota
	CheckedStateIsTrue
	CheckedStateIsFalse
	CheckedStateMixed
)

// Value is the integer the engine receives for this state.
func (s CheckedState) Value() int { return int(s) }

// Name is the enum value's name.
func (s CheckedState) Name() string { return [...]string{"none", "isTrue", "isFalse", "mixed"}[s] }

func (s CheckedState) String() string { return "CheckedState." + s.Name() }

// HasConflict reports whether both states are set (not none).
func (s CheckedState) HasConflict(other CheckedState) bool {
	return s != CheckedStateNone && other != CheckedStateNone
}

// Merge combines two states: mixed wins, then true, then false.
func (s CheckedState) Merge(other CheckedState) CheckedState {
	if s == CheckedStateMixed || other == CheckedStateMixed {
		return CheckedStateMixed
	}
	if s == CheckedStateIsTrue || other == CheckedStateIsTrue {
		return CheckedStateIsTrue
	}
	if s == CheckedStateIsFalse || other == CheckedStateIsFalse {
		return CheckedStateIsFalse
	}
	return CheckedStateNone
}

// Tristate is a tristate (none, true, false) flag of a semantics node.
type Tristate int

// The tristate values (Value is the enum's value field).
const (
	TristateNone Tristate = iota
	TristateIsTrue
	TristateIsFalse
)

// Value is the integer the engine receives for this state.
func (t Tristate) Value() int { return int(t) }

// Name is the enum value's name.
func (t Tristate) Name() string { return [...]string{"none", "isTrue", "isFalse"}[t] }

func (t Tristate) String() string { return "Tristate." + t.Name() }

// HasConflict reports whether both values are set (not none).
func (t Tristate) HasConflict(other Tristate) bool {
	return t != TristateNone && other != TristateNone
}

// Merge combines two values: true wins, then false.
func (t Tristate) Merge(other Tristate) Tristate {
	if t == TristateIsTrue || other == TristateIsTrue {
		return TristateIsTrue
	}
	if t == TristateIsFalse || other == TristateIsFalse {
		return TristateIsFalse
	}
	return TristateNone
}

// ToBoolOrNull is nil for none, otherwise the bool.
func (t Tristate) ToBoolOrNull() *bool {
	switch t {
	case TristateIsTrue:
		v := true
		return &v
	case TristateIsFalse:
		v := false
		return &v
	}
	return nil
}

// SemanticsHitTestBehavior describes how a semantic node participates in
// platform hit testing.
type SemanticsHitTestBehavior int

// The hit test behaviors.
const (
	SemanticsHitTestBehaviorDefer SemanticsHitTestBehavior = iota
	SemanticsHitTestBehaviorOpaque
	SemanticsHitTestBehaviorTransparent
)

// Name is the enum value's name.
func (b SemanticsHitTestBehavior) Name() string {
	return [...]string{"defer", "opaque", "transparent"}[b]
}

func (b SemanticsHitTestBehavior) String() string { return "SemanticsHitTestBehavior." + b.Name() }

// SemanticsFlags are the boolean and tristate properties of a semantics
// node. Every field's zero value is its Dart default, so SemanticsFlags{...}
// is Dart's SemanticsFlags(...); compare with == (Dart's ==). The native
// initializer (NativeSemanticsFlags::initSemanticsFlags) keeps an engine
// copy; ENGINE GAP: nothing reads it (see the file comment).
type SemanticsFlags struct {
	IsChecked                   CheckedState
	IsSelected                  Tristate
	IsEnabled                   Tristate
	IsToggled                   Tristate
	IsExpanded                  Tristate
	IsRequired                  Tristate
	IsFocused                   Tristate
	IsAccessibilityFocusBlocked bool
	IsButton                    bool
	IsTextField                 bool
	IsInMutuallyExclusiveGroup  bool
	IsHeader                    bool
	IsObscured                  bool
	ScopesRoute                 bool
	NamesRoute                  bool
	IsHidden                    bool
	IsImage                     bool
	IsLiveRegion                bool
	HasImplicitScrolling        bool
	IsMultiline                 bool
	IsReadOnly                  bool
	IsLink                      bool
	IsSlider                    bool
	IsKeyboardKey               bool
}

// SemanticsFlagsNone is SemanticsFlags.none, the default flags (an
// assignable static in Dart, as here).
var SemanticsFlagsNone = SemanticsFlags{}

// Merge combines the flags: tristates and checked states merge, booleans
// are or-ed.
func (f SemanticsFlags) Merge(other SemanticsFlags) SemanticsFlags {
	return SemanticsFlags{
		IsChecked:                   f.IsChecked.Merge(other.IsChecked),
		IsSelected:                  f.IsSelected.Merge(other.IsSelected),
		IsEnabled:                   f.IsEnabled.Merge(other.IsEnabled),
		IsToggled:                   f.IsToggled.Merge(other.IsToggled),
		IsExpanded:                  f.IsExpanded.Merge(other.IsExpanded),
		IsRequired:                  f.IsRequired.Merge(other.IsRequired),
		IsFocused:                   f.IsFocused.Merge(other.IsFocused),
		IsButton:                    f.IsButton || other.IsButton,
		IsTextField:                 f.IsTextField || other.IsTextField,
		IsInMutuallyExclusiveGroup:  f.IsInMutuallyExclusiveGroup || other.IsInMutuallyExclusiveGroup,
		IsHeader:                    f.IsHeader || other.IsHeader,
		IsObscured:                  f.IsObscured || other.IsObscured,
		ScopesRoute:                 f.ScopesRoute || other.ScopesRoute,
		NamesRoute:                  f.NamesRoute || other.NamesRoute,
		IsHidden:                    f.IsHidden || other.IsHidden,
		IsImage:                     f.IsImage || other.IsImage,
		IsLiveRegion:                f.IsLiveRegion || other.IsLiveRegion,
		HasImplicitScrolling:        f.HasImplicitScrolling || other.HasImplicitScrolling,
		IsMultiline:                 f.IsMultiline || other.IsMultiline,
		IsReadOnly:                  f.IsReadOnly || other.IsReadOnly,
		IsLink:                      f.IsLink || other.IsLink,
		IsSlider:                    f.IsSlider || other.IsSlider,
		IsKeyboardKey:               f.IsKeyboardKey || other.IsKeyboardKey,
		IsAccessibilityFocusBlocked: f.IsAccessibilityFocusBlocked || other.IsAccessibilityFocusBlocked,
	}
}

// SemanticsFlagsCopyWithOptions are SemanticsFlags.copyWith's parameters
// (nil: keep the current value).
type SemanticsFlagsCopyWithOptions struct {
	IsChecked                   *CheckedState
	IsSelected                  *Tristate
	IsEnabled                   *Tristate
	IsToggled                   *Tristate
	IsExpanded                  *Tristate
	IsRequired                  *Tristate
	IsFocused                   *Tristate
	IsButton                    *bool
	IsTextField                 *bool
	IsInMutuallyExclusiveGroup  *bool
	IsHeader                    *bool
	IsObscured                  *bool
	ScopesRoute                 *bool
	NamesRoute                  *bool
	IsHidden                    *bool
	IsImage                     *bool
	IsLiveRegion                *bool
	HasImplicitScrolling        *bool
	IsMultiline                 *bool
	IsReadOnly                  *bool
	IsLink                      *bool
	IsSlider                    *bool
	IsKeyboardKey               *bool
	IsAccessibilityFocusBlocked *bool
}

func orValue[T any](p *T, v T) T {
	if p != nil {
		return *p
	}
	return v
}

// CopyWith is a copy of the flags with the given values replaced.
func (f SemanticsFlags) CopyWith(opts ...SemanticsFlagsCopyWithOptions) SemanticsFlags {
	var o SemanticsFlagsCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return SemanticsFlags{
		IsChecked:                   orValue(o.IsChecked, f.IsChecked),
		IsSelected:                  orValue(o.IsSelected, f.IsSelected),
		IsButton:                    orValue(o.IsButton, f.IsButton),
		IsTextField:                 orValue(o.IsTextField, f.IsTextField),
		IsFocused:                   orValue(o.IsFocused, f.IsFocused),
		IsEnabled:                   orValue(o.IsEnabled, f.IsEnabled),
		IsInMutuallyExclusiveGroup:  orValue(o.IsInMutuallyExclusiveGroup, f.IsInMutuallyExclusiveGroup),
		IsHeader:                    orValue(o.IsHeader, f.IsHeader),
		IsObscured:                  orValue(o.IsObscured, f.IsObscured),
		ScopesRoute:                 orValue(o.ScopesRoute, f.ScopesRoute),
		NamesRoute:                  orValue(o.NamesRoute, f.NamesRoute),
		IsHidden:                    orValue(o.IsHidden, f.IsHidden),
		IsImage:                     orValue(o.IsImage, f.IsImage),
		IsLiveRegion:                orValue(o.IsLiveRegion, f.IsLiveRegion),
		IsToggled:                   orValue(o.IsToggled, f.IsToggled),
		HasImplicitScrolling:        orValue(o.HasImplicitScrolling, f.HasImplicitScrolling),
		IsMultiline:                 orValue(o.IsMultiline, f.IsMultiline),
		IsReadOnly:                  orValue(o.IsReadOnly, f.IsReadOnly),
		IsLink:                      orValue(o.IsLink, f.IsLink),
		IsSlider:                    orValue(o.IsSlider, f.IsSlider),
		IsKeyboardKey:               orValue(o.IsKeyboardKey, f.IsKeyboardKey),
		IsExpanded:                  orValue(o.IsExpanded, f.IsExpanded),
		IsRequired:                  orValue(o.IsRequired, f.IsRequired),
		IsAccessibilityFocusBlocked: orValue(o.IsAccessibilityFocusBlocked, f.IsAccessibilityFocusBlocked),
	}
}

// ToStrings is the names of the set flags, in the engine's order.
func (f SemanticsFlags) ToStrings() []string {
	out := []string{}
	add := func(cond bool, name string) {
		if cond {
			out = append(out, name)
		}
	}
	add(f.IsChecked != CheckedStateNone, "hasCheckedState")
	add(f.IsChecked == CheckedStateIsTrue, "isChecked")
	add(f.IsSelected == TristateIsTrue, "isSelected")
	add(f.IsButton, "isButton")
	add(f.IsTextField, "isTextField")
	add(f.IsFocused == TristateIsTrue, "isFocused")
	add(f.IsEnabled != TristateNone, "hasEnabledState")
	add(f.IsEnabled == TristateIsTrue, "isEnabled")
	add(f.IsInMutuallyExclusiveGroup, "isInMutuallyExclusiveGroup")
	add(f.IsHeader, "isHeader")
	add(f.IsObscured, "isObscured")
	add(f.ScopesRoute, "scopesRoute")
	add(f.NamesRoute, "namesRoute")
	add(f.IsHidden, "isHidden")
	add(f.IsImage, "isImage")
	add(f.IsLiveRegion, "isLiveRegion")
	add(f.IsToggled != TristateNone, "hasToggledState")
	add(f.IsToggled == TristateIsTrue, "isToggled")
	add(f.HasImplicitScrolling, "hasImplicitScrolling")
	add(f.IsMultiline, "isMultiline")
	add(f.IsReadOnly, "isReadOnly")
	add(f.IsFocused != TristateNone, "isFocusable")
	add(f.IsAccessibilityFocusBlocked, "isAccessibilityFocusBlocked")
	add(f.IsLink, "isLink")
	add(f.IsSlider, "isSlider")
	add(f.IsKeyboardKey, "isKeyboardKey")
	add(f.IsChecked == CheckedStateMixed, "isCheckStateMixed")
	add(f.IsExpanded != TristateNone, "hasExpandedState")
	add(f.IsExpanded == TristateIsTrue, "isExpanded")
	add(f.IsSelected != TristateNone, "hasSelectedState")
	add(f.IsRequired != TristateNone, "hasRequiredState")
	add(f.IsRequired == TristateIsTrue, "isRequired")
	return out
}

// HasRepeatedFlags reports whether any flag is set in both (Deprecated: use
// HasConflictingFlags; it ignores isAccessibilityFocusBlocked).
func (f SemanticsFlags) HasRepeatedFlags(other SemanticsFlags) bool {
	return f.IsChecked.HasConflict(other.IsChecked) ||
		f.IsSelected.HasConflict(other.IsSelected) ||
		f.IsEnabled.HasConflict(other.IsEnabled) ||
		f.IsToggled.HasConflict(other.IsToggled) ||
		f.IsEnabled.HasConflict(other.IsEnabled) ||
		f.IsExpanded.HasConflict(other.IsExpanded) ||
		f.IsRequired.HasConflict(other.IsRequired) ||
		f.IsFocused.HasConflict(other.IsFocused) ||
		(f.IsButton && other.IsButton) ||
		(f.IsTextField && other.IsTextField) ||
		(f.IsInMutuallyExclusiveGroup && other.IsInMutuallyExclusiveGroup) ||
		(f.IsHeader && other.IsHeader) ||
		(f.IsObscured && other.IsObscured) ||
		(f.ScopesRoute && other.ScopesRoute) ||
		(f.NamesRoute && other.NamesRoute) ||
		(f.IsHidden && other.IsHidden) ||
		(f.IsImage && other.IsImage) ||
		(f.IsLiveRegion && other.IsLiveRegion) ||
		(f.HasImplicitScrolling && other.HasImplicitScrolling) ||
		(f.IsMultiline && other.IsMultiline) ||
		(f.IsReadOnly && other.IsReadOnly) ||
		(f.IsLink && other.IsLink) ||
		(f.IsSlider && other.IsSlider) ||
		(f.IsKeyboardKey && other.IsKeyboardKey)
}

// HasConflictingFlags reports whether the two sets of flags can't be
// merged: a flag set in both, or different isAccessibilityFocusBlocked.
func (f SemanticsFlags) HasConflictingFlags(other SemanticsFlags) bool {
	return f.HasRepeatedFlags(other) ||
		(f.IsAccessibilityFocusBlocked != other.IsAccessibilityFocusBlocked)
}

// SemanticsValidationResult is the validation result of a form field.
type SemanticsValidationResult int

// The validation results.
const (
	SemanticsValidationResultNone SemanticsValidationResult = iota
	SemanticsValidationResultValid
	SemanticsValidationResultInvalid
)

// Name is the enum value's name.
func (r SemanticsValidationResult) Name() string {
	return [...]string{"none", "valid", "invalid"}[r]
}

func (r SemanticsValidationResult) String() string { return "SemanticsValidationResult." + r.Name() }

// StringAttribute is an abstract interface for string attributes that
// affect how assistive technologies, e.g. VoiceOver or TalkBack, treat the
// text. Attributes are compared by identity (Dart's StringAttribute has no
// ==), so implementations are pointers.
type StringAttribute interface {
	// Range is the range of the text to which this attribute applies.
	Range() TextRange
	// Copy returns a copy of this attribute with the given range.
	Copy(rng TextRange) StringAttribute
	String() string
	isStringAttribute()
}

// SpellOutStringAttribute indicates the string should be spelled out
// character by character.
type SpellOutStringAttribute struct {
	rng TextRange
}

// NewSpellOutStringAttribute is SpellOutStringAttribute(range:).
func NewSpellOutStringAttribute(rng TextRange) *SpellOutStringAttribute {
	return &SpellOutStringAttribute{rng: rng}
}

func (a *SpellOutStringAttribute) Range() TextRange { return a.rng }

func (a *SpellOutStringAttribute) Copy(rng TextRange) StringAttribute {
	return NewSpellOutStringAttribute(rng)
}

func (a *SpellOutStringAttribute) String() string {
	return "SpellOutStringAttribute(" + a.rng.String() + ")"
}

func (*SpellOutStringAttribute) isStringAttribute() {}

// LocaleStringAttribute indicates the string should be pronounced with a
// different locale.
type LocaleStringAttribute struct {
	rng TextRange
	// Locale is the locale of this attribute.
	Locale Locale
}

// NewLocaleStringAttribute is LocaleStringAttribute(range:, locale:).
func NewLocaleStringAttribute(rng TextRange, locale Locale) *LocaleStringAttribute {
	return &LocaleStringAttribute{rng: rng, Locale: locale}
}

func (a *LocaleStringAttribute) Range() TextRange { return a.rng }

func (a *LocaleStringAttribute) Copy(rng TextRange) StringAttribute {
	return NewLocaleStringAttribute(rng, a.Locale)
}

func (a *LocaleStringAttribute) String() string {
	return "LocaleStringAttribute(" + a.rng.String() + ", " + a.Locale.ToLanguageTag() + ")"
}

func (*LocaleStringAttribute) isStringAttribute() {}

// SemanticsUpdateBuilderUpdateNodeOptions are SemanticsUpdateBuilder.
// updateNode's named parameters. Most are required in Dart; every optional
// one defaults to its Go zero value (headingLevel 0, linkUrl ”, role none,
// validationResult none, hitTestBehavior defer). Nullable ones are nil for
// Dart's null (TextDirection, ControlsNodes, Locale).
type SemanticsUpdateBuilderUpdateNodeOptions struct {
	ID                       int
	Flags                    SemanticsFlags
	Actions                  int
	MaxValueLength           int
	CurrentValueLength       int
	TextSelectionBase        int
	TextSelectionExtent      int
	PlatformViewID           int
	ScrollChildren           int
	ScrollIndex              int
	TraversalParent          int
	ScrollPosition           float64
	ScrollExtentMax          float64
	ScrollExtentMin          float64
	Rect                     Rect
	Identifier               string
	Label                    string
	LabelAttributes          []StringAttribute
	Value                    string
	ValueAttributes          []StringAttribute
	IncreasedValue           string
	IncreasedValueAttributes []StringAttribute
	DecreasedValue           string
	DecreasedValueAttributes []StringAttribute
	Hint                     string
	HintAttributes           []StringAttribute
	Tooltip                  string
	TextDirection            *TextDirection
	Transform                []float64
	HitTestTransform         []float64
	ChildrenInTraversalOrder []int32
	ChildrenInHitTestOrder   []int32
	AdditionalActions        []int32
	HeadingLevel             int
	LinkURL                  string
	Role                     SemanticsRole
	ControlsNodes            []string
	ValidationResult         SemanticsValidationResult
	HitTestBehavior          SemanticsHitTestBehavior
	InputType                SemanticsInputType
	Locale                   *Locale
	MinValue                 string
	MaxValue                 string
}

// SemanticsUpdateBuilderUpdateCustomActionOptions are updateCustomAction's
// optional parameters (Label and Hint nil: null; OverrideID nil: -1).
type SemanticsUpdateBuilderUpdateCustomActionOptions struct {
	Label      *string
	Hint       *string
	OverrideID *int
}

// SemanticsUpdateBuilder is an object that creates SemanticsUpdate objects.
// Once created, the SemanticsUpdate objects can be passed to
// PlatformDispatcher.updateSemantics to update the semantics conveyed to
// the user.
type SemanticsUpdateBuilder interface {
	// UpdateNode updates the properties of a particular semantics node
	// (Dart's named parameters are the fields of opts).
	UpdateNode(opts SemanticsUpdateBuilderUpdateNodeOptions)
	// UpdateCustomAction updates the custom semantics action associated
	// with the given id.
	UpdateCustomAction(id int, opts ...SemanticsUpdateBuilderUpdateCustomActionOptions)
	// Build creates a SemanticsUpdate object that encapsulates the updates
	// recorded by this object.
	Build() SemanticsUpdate
}

// NewSemanticsUpdateBuilder is SemanticsUpdateBuilder() (the native
// builder).
func NewSemanticsUpdateBuilder() SemanticsUpdateBuilder { return &nativeSemanticsUpdateBuilder{} }

// EngineSemanticsNodeUpdate is the engine's record of one updateNode call
// (lib/ui/semantics/semantics_node.h's SemanticsNode, as the arguments the
// native receives: the rect as left/top/right/bottom, the text direction as
// index + 1 or 0, the role, validation result, hit test behavior and input
// type as indices, and the locale as a language tag or ”).
type EngineSemanticsNodeUpdate struct {
	SemanticsUpdateBuilderUpdateNodeOptions
	TextDirectionIndex    int
	RoleIndex             int
	ValidationResultIndex int
	HitTestBehaviorIndex  int
	InputTypeIndex        int
	LocaleTag             string
}

// EngineCustomAccessibilityActionUpdate is the engine's record of one
// updateCustomAction call (null label and hint are ”).
type EngineCustomAccessibilityActionUpdate struct {
	ID         int
	Label      string
	Hint       string
	OverrideID int
}

// EngineSemanticsUpdate is what a built SemanticsUpdate holds for the
// engine: the node and custom action updates, in the order they were made.
type EngineSemanticsUpdate struct {
	Nodes         []EngineSemanticsNodeUpdate
	CustomActions []EngineCustomAccessibilityActionUpdate
}

// nativeSemanticsUpdateBuilder is _NativeSemanticsUpdateBuilder.
type nativeSemanticsUpdateBuilder struct {
	update EngineSemanticsUpdate
}

func (b *nativeSemanticsUpdateBuilder) UpdateNode(o SemanticsUpdateBuilderUpdateNodeOptions) {
	matrix4IsValid(o.Transform)
	if !(o.HeadingLevel >= 0 && o.HeadingLevel <= 6) {
		panic(dart.NewAssertionError("Heading level must be between 1 and 6, or 0 to indicate that this node is not a heading."))
	}
	textDirection := 0
	if o.TextDirection != nil {
		textDirection = int(*o.TextDirection) + 1
	}
	locale := ""
	if o.Locale != nil {
		locale = o.Locale.ToLanguageTag()
	}
	b.update.Nodes = append(b.update.Nodes, EngineSemanticsNodeUpdate{
		SemanticsUpdateBuilderUpdateNodeOptions: o,
		TextDirectionIndex:                      textDirection,
		RoleIndex:                               int(o.Role),
		ValidationResultIndex:                   int(o.ValidationResult),
		HitTestBehaviorIndex:                    int(o.HitTestBehavior),
		InputTypeIndex:                          int(o.InputType),
		LocaleTag:                               locale,
	})
}

func (b *nativeSemanticsUpdateBuilder) UpdateCustomAction(id int, opts ...SemanticsUpdateBuilderUpdateCustomActionOptions) {
	var o SemanticsUpdateBuilderUpdateCustomActionOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	overrideID := -1
	if o.OverrideID != nil {
		overrideID = *o.OverrideID
	}
	b.update.CustomActions = append(b.update.CustomActions, EngineCustomAccessibilityActionUpdate{
		ID: id, Label: orValue(o.Label, ""), Hint: orValue(o.Hint, ""), OverrideID: overrideID,
	})
}

func (b *nativeSemanticsUpdateBuilder) Build() SemanticsUpdate {
	u := &nativeSemanticsUpdate{update: b.update}
	b.update = EngineSemanticsUpdate{}
	return u
}

func (b *nativeSemanticsUpdateBuilder) String() string { return "SemanticsUpdateBuilder" }

// SemanticsUpdate is an opaque object representing a batch of semantics
// updates. To create a SemanticsUpdate object, use a
// SemanticsUpdateBuilder.
type SemanticsUpdate interface {
	// Dispose releases the resources used by this semantics update.
	Dispose()
}

// nativeSemanticsUpdate is _NativeSemanticsUpdate.
type nativeSemanticsUpdate struct {
	update   EngineSemanticsUpdate
	disposed bool
}

func (u *nativeSemanticsUpdate) Dispose() { u.disposed = true }

func (u *nativeSemanticsUpdate) String() string { return "SemanticsUpdate" }

// EngineSemanticsUpdateOf is the engine side of an update built by
// NewSemanticsUpdateBuilder (nil for another implementation or a disposed
// update): flutter-go's inspectable stand-in for the update the
// accessibility bridge would receive (ENGINE GAP, see the file comment).
func EngineSemanticsUpdateOf(update SemanticsUpdate) *EngineSemanticsUpdate {
	u, ok := update.(*nativeSemanticsUpdate)
	if !ok || u.disposed {
		return nil
	}
	return &u.update
}

// matrix4IsValid is painting.dart's _matrix4IsValid assertion.
func matrix4IsValid(matrix4 []float64) {
	if len(matrix4) != 16 {
		panic(dart.NewAssertionError("Matrix4 must have 16 entries."))
	}
	for _, v := range matrix4 {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			panic(dart.NewAssertionError("Matrix4 entries must be finite."))
		}
	}
}
