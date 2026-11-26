package semantics

// Port of semantics.dart (Flutter 3.50.1), part 4 of 4:
// SemanticsConfiguration, DebugSemanticsDumpOrder, SemanticsSortKey,
// OrdinalSortKey and the helpers (_concatAttributedString,
// _mergeHeadingLevels, _tristateFromBoolOrNull, _toBitMask).

import (
	"math"
	"net/url"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// SemanticsConfiguration describes the semantic information associated
// with the owning RenderObject. Use it by pointer (NewSemanticsConfiguration).
type SemanticsConfiguration struct {
	isSemanticBoundary bool
	localeForSubtree   *ui.Locale

	// Locale is the locale of the resulting SemanticsNode, if any.
	Locale *ui.Locale
	// IsBlockingUserActions is whether to block pointer related user
	// actions for the rendering subtree.
	IsBlockingUserActions bool
	// ExplicitChildNodes is whether the configuration forces all children
	// of the owning RenderObject that want to contribute semantic
	// information to the semantics tree to do so in the form of explicit
	// SemanticsNodes.
	ExplicitChildNodes bool
	// IsBlockingSemanticsOfPreviouslyPaintedNodes is whether the owning
	// RenderObject makes other RenderObjects previously painted within the
	// same semantic boundary unreachable for accessibility purposes.
	IsBlockingSemanticsOfPreviouslyPaintedNodes bool

	hasBeenAnnotated bool

	actions       map[ui.SemanticsAction]SemanticsActionHandler
	actionsAsBits int

	onTap                           func()
	onLongPress                     func()
	onScrollLeft                    func()
	onDismiss                       func()
	onScrollRight                   func()
	onScrollUp                      func()
	onScrollDown                    func()
	onScrollToOffset                ScrollToOffsetHandler
	onIncrease                      func()
	onDecrease                      func()
	onCopy                          func()
	onCut                           func()
	onPaste                         func()
	onShowOnScreen                  func()
	onMoveCursorForwardByCharacter  MoveCursorHandler
	onMoveCursorBackwardByCharacter MoveCursorHandler
	onMoveCursorForwardByWord       MoveCursorHandler
	onMoveCursorBackwardByWord      MoveCursorHandler
	onSetSelection                  SetSelectionHandler
	onSetText                       SetTextHandler
	onDidGainAccessibilityFocus     func()
	onDidLoseAccessibilityFocus     func()
	onFocus                         func()
	onExpand                        func()
	onCollapse                      func()
	childConfigurationsDelegate     ChildSemanticsConfigurationsDelegate

	sortKey            SemanticsSortKey
	indexInParent      *int
	scrollChildCount   *int
	scrollIndex        *int
	platformViewID     *int
	maxValueLength     *int
	currentValueLength *int

	isMergingSemanticsOfDescendants bool

	customSemanticsActions customActionEntries

	identifier                string
	traversalParentIdentifier any
	traversalChildIdentifier  any
	role                      ui.SemanticsRole

	attributedLabel          AttributedString
	attributedValue          AttributedString
	attributedIncreasedValue AttributedString
	attributedDecreasedValue AttributedString
	attributedHint           AttributedString
	tooltip                  string
	hintOverrides            *SemanticsHintOverrides

	textDirection *ui.TextDirection

	accessibilityFocusBlockType AccessibilityFocusBlockType
	linkURL                     *url.URL
	headingLevel                int

	textSelection    *services.TextSelection
	scrollPosition   *float64
	scrollExtentMax  *float64
	scrollExtentMin  *float64
	controlsNodes    []string
	validationResult ui.SemanticsValidationResult
	hitTestBehavior  ui.SemanticsHitTestBehavior
	inputType        ui.SemanticsInputType
	maxValue         *string
	minValue         *string

	tagsForChildren []*SemanticsTag

	flags ui.SemanticsFlags
}

// NewSemanticsConfiguration is SemanticsConfiguration().
func NewSemanticsConfiguration() *SemanticsConfiguration {
	return &SemanticsConfiguration{
		actions:                  map[ui.SemanticsAction]SemanticsActionHandler{},
		customSemanticsActions:   customActionEntries{},
		attributedLabel:          NewAttributedString(""),
		attributedValue:          NewAttributedString(""),
		attributedIncreasedValue: NewAttributedString(""),
		attributedDecreasedValue: NewAttributedString(""),
		attributedHint:           NewAttributedString(""),
		flags:                    ui.SemanticsFlagsNone,
	}
}

// ---- SEMANTIC BOUNDARY BEHAVIOR

// IsSemanticBoundary is whether the owner creates a new SemanticsNode.
func (c *SemanticsConfiguration) IsSemanticBoundary() bool { return c.isSemanticBoundary }

// SetIsSemanticBoundary sets IsSemanticBoundary.
func (c *SemanticsConfiguration) SetIsSemanticBoundary(value bool) {
	dartAssert(!c.IsMergingSemanticsOfDescendants() || value, nil)
	c.isSemanticBoundary = value
}

// LocaleForSubtree is the locale for widgets in the subtree (nil: none).
func (c *SemanticsConfiguration) LocaleForSubtree() *ui.Locale { return c.localeForSubtree }

// SetLocaleForSubtree sets LocaleForSubtree (not nil).
func (c *SemanticsConfiguration) SetLocaleForSubtree(value *ui.Locale) {
	dartAssert(value != nil, nil)
	c.localeForSubtree = value
	c.hasBeenAnnotated = true
}

// ---- SEMANTIC ANNOTATIONS

// HasBeenAnnotated is whether this configuration is empty.
func (c *SemanticsConfiguration) HasBeenAnnotated() bool { return c.hasBeenAnnotated }

func (c *SemanticsConfiguration) effectiveActionsAsBits() int {
	if c.IsBlockingUserActions {
		return c.actionsAsBits & kUnblockedUserActions
	}
	return c.actionsAsBits
}

func (c *SemanticsConfiguration) addAction(action ui.SemanticsAction, handler SemanticsActionHandler) {
	c.actions[action] = handler
	c.actionsAsBits |= action.Index()
	c.hasBeenAnnotated = true
}

func (c *SemanticsConfiguration) addArgumentlessAction(action ui.SemanticsAction, handler func()) {
	if handler == nil {
		panic("Null check operator used on a null value")
	}
	c.addAction(action, func(args any) {
		dartAssert(args == nil, nil)
		handler()
	})
}

// OnTap is the handler for SemanticsAction.tap.
func (c *SemanticsConfiguration) OnTap() func() { return c.onTap }

// SetOnTap sets OnTap (not nil).
func (c *SemanticsConfiguration) SetOnTap(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionTap, value)
	c.onTap = value
}

// OnLongPress is the handler for SemanticsAction.longPress.
func (c *SemanticsConfiguration) OnLongPress() func() { return c.onLongPress }

// SetOnLongPress sets OnLongPress (not nil).
func (c *SemanticsConfiguration) SetOnLongPress(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionLongPress, value)
	c.onLongPress = value
}

// OnScrollLeft is the handler for SemanticsAction.scrollLeft.
func (c *SemanticsConfiguration) OnScrollLeft() func() { return c.onScrollLeft }

// SetOnScrollLeft sets OnScrollLeft (not nil).
func (c *SemanticsConfiguration) SetOnScrollLeft(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionScrollLeft, value)
	c.onScrollLeft = value
}

// OnDismiss is the handler for SemanticsAction.dismiss.
func (c *SemanticsConfiguration) OnDismiss() func() { return c.onDismiss }

// SetOnDismiss sets OnDismiss (not nil).
func (c *SemanticsConfiguration) SetOnDismiss(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionDismiss, value)
	c.onDismiss = value
}

// OnScrollRight is the handler for SemanticsAction.scrollRight.
func (c *SemanticsConfiguration) OnScrollRight() func() { return c.onScrollRight }

// SetOnScrollRight sets OnScrollRight (not nil).
func (c *SemanticsConfiguration) SetOnScrollRight(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionScrollRight, value)
	c.onScrollRight = value
}

// OnScrollUp is the handler for SemanticsAction.scrollUp.
func (c *SemanticsConfiguration) OnScrollUp() func() { return c.onScrollUp }

// SetOnScrollUp sets OnScrollUp (not nil).
func (c *SemanticsConfiguration) SetOnScrollUp(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionScrollUp, value)
	c.onScrollUp = value
}

// OnScrollDown is the handler for SemanticsAction.scrollDown.
func (c *SemanticsConfiguration) OnScrollDown() func() { return c.onScrollDown }

// SetOnScrollDown sets OnScrollDown (not nil).
func (c *SemanticsConfiguration) SetOnScrollDown(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionScrollDown, value)
	c.onScrollDown = value
}

// OnScrollToOffset is the handler for SemanticsAction.scrollToOffset.
func (c *SemanticsConfiguration) OnScrollToOffset() ScrollToOffsetHandler { return c.onScrollToOffset }

// SetOnScrollToOffset sets OnScrollToOffset (not nil); the action's
// argument is a Float64List [x, y].
func (c *SemanticsConfiguration) SetOnScrollToOffset(value ScrollToOffsetHandler) {
	dartAssert(value != nil, nil)
	c.addAction(ui.SemanticsActionScrollToOffset, func(args any) {
		list := args.([]float64)
		value(ui.NewOffset(list[0], list[1]))
	})
	c.onScrollToOffset = value
}

// OnIncrease is the handler for SemanticsAction.increase.
func (c *SemanticsConfiguration) OnIncrease() func() { return c.onIncrease }

// SetOnIncrease sets OnIncrease (not nil).
func (c *SemanticsConfiguration) SetOnIncrease(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionIncrease, value)
	c.onIncrease = value
}

// OnDecrease is the handler for SemanticsAction.decrease.
func (c *SemanticsConfiguration) OnDecrease() func() { return c.onDecrease }

// SetOnDecrease sets OnDecrease (not nil).
func (c *SemanticsConfiguration) SetOnDecrease(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionDecrease, value)
	c.onDecrease = value
}

// OnCopy is the handler for SemanticsAction.copy.
func (c *SemanticsConfiguration) OnCopy() func() { return c.onCopy }

// SetOnCopy sets OnCopy (not nil).
func (c *SemanticsConfiguration) SetOnCopy(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionCopy, value)
	c.onCopy = value
}

// OnCut is the handler for SemanticsAction.cut.
func (c *SemanticsConfiguration) OnCut() func() { return c.onCut }

// SetOnCut sets OnCut (not nil).
func (c *SemanticsConfiguration) SetOnCut(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionCut, value)
	c.onCut = value
}

// OnPaste is the handler for SemanticsAction.paste.
func (c *SemanticsConfiguration) OnPaste() func() { return c.onPaste }

// SetOnPaste sets OnPaste (not nil).
func (c *SemanticsConfiguration) SetOnPaste(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionPaste, value)
	c.onPaste = value
}

// OnShowOnScreen is the handler for SemanticsAction.showOnScreen.
func (c *SemanticsConfiguration) OnShowOnScreen() func() { return c.onShowOnScreen }

// SetOnShowOnScreen sets OnShowOnScreen (not nil).
func (c *SemanticsConfiguration) SetOnShowOnScreen(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionShowOnScreen, value)
	c.onShowOnScreen = value
}

func moveCursorAction(value MoveCursorHandler) SemanticsActionHandler {
	return func(args any) {
		extendSelection := args.(bool)
		value(extendSelection)
	}
}

// OnMoveCursorForwardByCharacter is the handler for
// SemanticsAction.moveCursorForwardByCharacter.
func (c *SemanticsConfiguration) OnMoveCursorForwardByCharacter() MoveCursorHandler {
	return c.onMoveCursorForwardByCharacter
}

// SetOnMoveCursorForwardByCharacter sets OnMoveCursorForwardByCharacter
// (not nil).
func (c *SemanticsConfiguration) SetOnMoveCursorForwardByCharacter(value MoveCursorHandler) {
	dartAssert(value != nil, nil)
	c.addAction(ui.SemanticsActionMoveCursorForwardByCharacter, moveCursorAction(value))
	c.onMoveCursorForwardByCharacter = value
}

// OnMoveCursorBackwardByCharacter is the handler for
// SemanticsAction.moveCursorBackwardByCharacter.
func (c *SemanticsConfiguration) OnMoveCursorBackwardByCharacter() MoveCursorHandler {
	return c.onMoveCursorBackwardByCharacter
}

// SetOnMoveCursorBackwardByCharacter sets OnMoveCursorBackwardByCharacter
// (not nil).
func (c *SemanticsConfiguration) SetOnMoveCursorBackwardByCharacter(value MoveCursorHandler) {
	dartAssert(value != nil, nil)
	c.addAction(ui.SemanticsActionMoveCursorBackwardByCharacter, moveCursorAction(value))
	c.onMoveCursorBackwardByCharacter = value
}

// OnMoveCursorForwardByWord is the handler for
// SemanticsAction.moveCursorForwardByWord.
func (c *SemanticsConfiguration) OnMoveCursorForwardByWord() MoveCursorHandler {
	return c.onMoveCursorForwardByWord
}

// SetOnMoveCursorForwardByWord sets the handler for
// SemanticsAction.moveCursorForwardByWord (not nil). As in Flutter, it
// stores the handler as OnMoveCursorForwardByCharacter (Flutter's setter
// assigns _onMoveCursorForwardByCharacter), so OnMoveCursorForwardByWord
// stays nil.
func (c *SemanticsConfiguration) SetOnMoveCursorForwardByWord(value MoveCursorHandler) {
	dartAssert(value != nil, nil)
	c.addAction(ui.SemanticsActionMoveCursorForwardByWord, moveCursorAction(value))
	c.onMoveCursorForwardByCharacter = value
}

// OnMoveCursorBackwardByWord is the handler for
// SemanticsAction.moveCursorBackwardByWord.
func (c *SemanticsConfiguration) OnMoveCursorBackwardByWord() MoveCursorHandler {
	return c.onMoveCursorBackwardByWord
}

// SetOnMoveCursorBackwardByWord sets the handler for
// SemanticsAction.moveCursorBackwardByWord (not nil); like Flutter, it
// stores it as OnMoveCursorBackwardByCharacter.
func (c *SemanticsConfiguration) SetOnMoveCursorBackwardByWord(value MoveCursorHandler) {
	dartAssert(value != nil, nil)
	c.addAction(ui.SemanticsActionMoveCursorBackwardByWord, moveCursorAction(value))
	c.onMoveCursorBackwardByCharacter = value
}

// OnSetSelection is the handler for SemanticsAction.setSelection.
func (c *SemanticsConfiguration) OnSetSelection() SetSelectionHandler { return c.onSetSelection }

// SetOnSetSelection sets OnSetSelection (not nil). The action's argument
// is a map with int 'base' and 'extent'.
func (c *SemanticsConfiguration) SetOnSetSelection(value SetSelectionHandler) {
	dartAssert(value != nil, nil)
	c.addAction(ui.SemanticsActionSetSelection, func(args any) {
		dartAssert(args != nil, nil)
		base, okBase := services.MapLookup(args, "base")
		extent, okExtent := services.MapLookup(args, "extent")
		dartAssert(okBase && base != nil && okExtent && extent != nil, nil)
		value(services.NewTextSelection(base.(int), extent.(int)))
	})
	c.onSetSelection = value
}

// OnSetText is the handler for SemanticsAction.setText.
func (c *SemanticsConfiguration) OnSetText() SetTextHandler { return c.onSetText }

// SetOnSetText sets OnSetText (not nil).
func (c *SemanticsConfiguration) SetOnSetText(value SetTextHandler) {
	dartAssert(value != nil, nil)
	c.addAction(ui.SemanticsActionSetText, func(args any) {
		_, isString := args.(string)
		dartAssert(args != nil && isString, nil)
		value(args.(string))
	})
	c.onSetText = value
}

// OnDidGainAccessibilityFocus is the handler for
// SemanticsAction.didGainAccessibilityFocus.
func (c *SemanticsConfiguration) OnDidGainAccessibilityFocus() func() {
	return c.onDidGainAccessibilityFocus
}

// SetOnDidGainAccessibilityFocus sets OnDidGainAccessibilityFocus (not
// nil).
func (c *SemanticsConfiguration) SetOnDidGainAccessibilityFocus(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionDidGainAccessibilityFocus, value)
	c.onDidGainAccessibilityFocus = value
}

// OnDidLoseAccessibilityFocus is the handler for
// SemanticsAction.didLoseAccessibilityFocus.
func (c *SemanticsConfiguration) OnDidLoseAccessibilityFocus() func() {
	return c.onDidLoseAccessibilityFocus
}

// SetOnDidLoseAccessibilityFocus sets OnDidLoseAccessibilityFocus (not
// nil).
func (c *SemanticsConfiguration) SetOnDidLoseAccessibilityFocus(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionDidLoseAccessibilityFocus, value)
	c.onDidLoseAccessibilityFocus = value
}

// OnFocus is the handler for SemanticsAction.focus.
func (c *SemanticsConfiguration) OnFocus() func() { return c.onFocus }

// SetOnFocus sets OnFocus (not nil).
func (c *SemanticsConfiguration) SetOnFocus(value func()) {
	c.addArgumentlessAction(ui.SemanticsActionFocus, value)
	c.onFocus = value
}

// OnExpand is the handler for SemanticsAction.expand.
func (c *SemanticsConfiguration) OnExpand() func() { return c.onExpand }

// SetOnExpand sets OnExpand (not nil).
func (c *SemanticsConfiguration) SetOnExpand(value func()) {
	dartAssert(value != nil, nil)
	c.addArgumentlessAction(ui.SemanticsActionExpand, value)
	c.onExpand = value
}

// OnCollapse is the handler for SemanticsAction.collapse.
func (c *SemanticsConfiguration) OnCollapse() func() { return c.onCollapse }

// SetOnCollapse sets OnCollapse (not nil).
func (c *SemanticsConfiguration) SetOnCollapse(value func()) {
	dartAssert(value != nil, nil)
	c.addArgumentlessAction(ui.SemanticsActionCollapse, value)
	c.onCollapse = value
}

// ChildConfigurationsDelegate decides how the configurations of the
// children merge.
func (c *SemanticsConfiguration) ChildConfigurationsDelegate() ChildSemanticsConfigurationsDelegate {
	return c.childConfigurationsDelegate
}

// SetChildConfigurationsDelegate sets ChildConfigurationsDelegate (not
// nil; it does not annotate the configuration).
func (c *SemanticsConfiguration) SetChildConfigurationsDelegate(value ChildSemanticsConfigurationsDelegate) {
	dartAssert(value != nil, nil)
	c.childConfigurationsDelegate = value
}

// GetActionHandler returns the action handler registered for action, or
// nil.
func (c *SemanticsConfiguration) GetActionHandler(action ui.SemanticsAction) SemanticsActionHandler {
	return c.actions[action]
}

// SortKey determines the position of this node among its siblings in the
// traversal sort order.
func (c *SemanticsConfiguration) SortKey() SemanticsSortKey { return c.sortKey }

// SetSortKey sets SortKey (not nil).
func (c *SemanticsConfiguration) SetSortKey(value SemanticsSortKey) {
	dartAssert(value != nil, nil)
	c.sortKey = value
	c.hasBeenAnnotated = true
}

// IndexInParent is the index of this node within the parent's list of
// semantic children that count for scrolling.
func (c *SemanticsConfiguration) IndexInParent() *int { return c.indexInParent }

// SetIndexInParent sets IndexInParent.
func (c *SemanticsConfiguration) SetIndexInParent(value *int) {
	c.indexInParent = value
	c.hasBeenAnnotated = true
}

// ScrollChildCount is the total number of scrollable children that
// contribute to semantics.
func (c *SemanticsConfiguration) ScrollChildCount() *int { return c.scrollChildCount }

// SetScrollChildCount sets ScrollChildCount.
func (c *SemanticsConfiguration) SetScrollChildCount(value *int) {
	if ptrEqual(value, c.ScrollChildCount()) {
		return
	}
	c.scrollChildCount = value
	c.hasBeenAnnotated = true
}

// ScrollIndex is the index of the first visible scrollable child.
func (c *SemanticsConfiguration) ScrollIndex() *int { return c.scrollIndex }

// SetScrollIndex sets ScrollIndex.
func (c *SemanticsConfiguration) SetScrollIndex(value *int) {
	if ptrEqual(value, c.ScrollIndex()) {
		return
	}
	c.scrollIndex = value
	c.hasBeenAnnotated = true
}

// PlatformViewID is the id of the platform view whose semantics nodes will
// be added as children to this node.
func (c *SemanticsConfiguration) PlatformViewID() *int { return c.platformViewID }

// SetPlatformViewID sets PlatformViewID.
func (c *SemanticsConfiguration) SetPlatformViewID(value *int) {
	if ptrEqual(value, c.PlatformViewID()) {
		return
	}
	c.platformViewID = value
	c.hasBeenAnnotated = true
}

// MaxValueLength is the maximum number of characters that can be entered
// into an editable text field.
func (c *SemanticsConfiguration) MaxValueLength() *int { return c.maxValueLength }

// SetMaxValueLength sets MaxValueLength.
func (c *SemanticsConfiguration) SetMaxValueLength(value *int) {
	if ptrEqual(value, c.MaxValueLength()) {
		return
	}
	c.maxValueLength = value
	c.hasBeenAnnotated = true
}

// CurrentValueLength is the current number of characters that have been
// entered into an editable text field.
func (c *SemanticsConfiguration) CurrentValueLength() *int { return c.currentValueLength }

// SetCurrentValueLength sets CurrentValueLength.
func (c *SemanticsConfiguration) SetCurrentValueLength(value *int) {
	if ptrEqual(value, c.CurrentValueLength()) {
		return
	}
	c.currentValueLength = value
	c.hasBeenAnnotated = true
}

// IsMergingSemanticsOfDescendants is whether the semantic information
// provided by the owning RenderObject and all of its descendants should be
// treated as one logical entity.
func (c *SemanticsConfiguration) IsMergingSemanticsOfDescendants() bool {
	return c.isMergingSemanticsOfDescendants
}

// SetIsMergingSemanticsOfDescendants sets IsMergingSemanticsOfDescendants
// (only on a semantic boundary).
func (c *SemanticsConfiguration) SetIsMergingSemanticsOfDescendants(value bool) {
	dartAssert(c.IsSemanticBoundary(), nil)
	c.isMergingSemanticsOfDescendants = value
	c.hasBeenAnnotated = true
}

// CustomSemanticsActions is the handlers for each supported
// CustomSemanticsAction, in insertion order.
func (c *SemanticsConfiguration) CustomSemanticsActions() []dart.MapEntry[CustomSemanticsAction, func()] {
	return c.customSemanticsActions
}

// SetCustomSemanticsActions sets CustomSemanticsActions (and the
// customAction action).
func (c *SemanticsConfiguration) SetCustomSemanticsActions(value []dart.MapEntry[CustomSemanticsAction, func()]) {
	c.hasBeenAnnotated = true
	c.actionsAsBits |= ui.SemanticsActionCustomAction.Index()
	c.customSemanticsActions = value
	c.actions[ui.SemanticsActionCustomAction] = c.onCustomSemanticsAction
}

func (c *SemanticsConfiguration) onCustomSemanticsAction(args any) {
	action := CustomSemanticsActionGetAction(args.(int))
	if action == nil {
		return
	}
	if i := customActionIndex(c.customSemanticsActions, *action); i >= 0 && c.customSemanticsActions[i].Value != nil {
		c.customSemanticsActions[i].Value()
	}
}

// Identifier is the identifier of the node in the platform's accessibility
// tree.
func (c *SemanticsConfiguration) Identifier() string { return c.identifier }

// SetIdentifier sets Identifier.
func (c *SemanticsConfiguration) SetIdentifier(identifier string) {
	c.identifier = identifier
	c.hasBeenAnnotated = true
}

// TraversalParentIdentifier is the identifier of this node as a traversal
// parent (nil: none).
func (c *SemanticsConfiguration) TraversalParentIdentifier() any { return c.traversalParentIdentifier }

// SetTraversalParentIdentifier sets TraversalParentIdentifier.
func (c *SemanticsConfiguration) SetTraversalParentIdentifier(value any) {
	if objectsEqual(value, c.TraversalParentIdentifier()) {
		return
	}
	c.traversalParentIdentifier = value
	c.hasBeenAnnotated = true
}

// TraversalChildIdentifier is the identifier of this node's traversal
// parent (nil: none).
func (c *SemanticsConfiguration) TraversalChildIdentifier() any { return c.traversalChildIdentifier }

// SetTraversalChildIdentifier sets TraversalChildIdentifier.
func (c *SemanticsConfiguration) SetTraversalChildIdentifier(value any) {
	if objectsEqual(value, c.TraversalChildIdentifier()) {
		return
	}
	c.traversalChildIdentifier = value
	c.hasBeenAnnotated = true
}

// Role is the semantics role of this node.
func (c *SemanticsConfiguration) Role() ui.SemanticsRole { return c.role }

// SetRole sets Role.
func (c *SemanticsConfiguration) SetRole(value ui.SemanticsRole) {
	c.role = value
	c.hasBeenAnnotated = true
}

// Label is a textual description of the owning RenderObject.
func (c *SemanticsConfiguration) Label() string { return c.attributedLabel.Text }

// SetLabel sets Label (without attributes).
func (c *SemanticsConfiguration) SetLabel(label string) {
	c.attributedLabel = NewAttributedString(label)
	c.hasBeenAnnotated = true
}

// AttributedLabel is the label with its attributes.
func (c *SemanticsConfiguration) AttributedLabel() AttributedString { return c.attributedLabel }

// SetAttributedLabel sets AttributedLabel.
func (c *SemanticsConfiguration) SetAttributedLabel(attributedLabel AttributedString) {
	c.attributedLabel = attributedLabel
	c.hasBeenAnnotated = true
}

// Value is a textual description for the current value of the owning
// RenderObject.
func (c *SemanticsConfiguration) Value() string { return c.attributedValue.Text }

// SetValue sets Value (without attributes).
func (c *SemanticsConfiguration) SetValue(value string) {
	c.attributedValue = NewAttributedString(value)
	c.hasBeenAnnotated = true
}

// AttributedValue is the value with its attributes.
func (c *SemanticsConfiguration) AttributedValue() AttributedString { return c.attributedValue }

// SetAttributedValue sets AttributedValue.
func (c *SemanticsConfiguration) SetAttributedValue(attributedValue AttributedString) {
	c.attributedValue = attributedValue
	c.hasBeenAnnotated = true
}

// IncreasedValue is the value that Value will have after an increase.
func (c *SemanticsConfiguration) IncreasedValue() string { return c.attributedIncreasedValue.Text }

// SetIncreasedValue sets IncreasedValue (without attributes).
func (c *SemanticsConfiguration) SetIncreasedValue(increasedValue string) {
	c.attributedIncreasedValue = NewAttributedString(increasedValue)
	c.hasBeenAnnotated = true
}

// AttributedIncreasedValue is IncreasedValue with its attributes.
func (c *SemanticsConfiguration) AttributedIncreasedValue() AttributedString {
	return c.attributedIncreasedValue
}

// SetAttributedIncreasedValue sets AttributedIncreasedValue.
func (c *SemanticsConfiguration) SetAttributedIncreasedValue(attributedIncreasedValue AttributedString) {
	c.attributedIncreasedValue = attributedIncreasedValue
	c.hasBeenAnnotated = true
}

// DecreasedValue is the value that Value will have after a decrease.
func (c *SemanticsConfiguration) DecreasedValue() string { return c.attributedDecreasedValue.Text }

// SetDecreasedValue sets DecreasedValue (without attributes).
func (c *SemanticsConfiguration) SetDecreasedValue(decreasedValue string) {
	c.attributedDecreasedValue = NewAttributedString(decreasedValue)
	c.hasBeenAnnotated = true
}

// AttributedDecreasedValue is DecreasedValue with its attributes.
func (c *SemanticsConfiguration) AttributedDecreasedValue() AttributedString {
	return c.attributedDecreasedValue
}

// SetAttributedDecreasedValue sets AttributedDecreasedValue.
func (c *SemanticsConfiguration) SetAttributedDecreasedValue(attributedDecreasedValue AttributedString) {
	c.attributedDecreasedValue = attributedDecreasedValue
	c.hasBeenAnnotated = true
}

// Hint is a brief description of the result of performing an action on
// this node.
func (c *SemanticsConfiguration) Hint() string { return c.attributedHint.Text }

// SetHint sets Hint (without attributes).
func (c *SemanticsConfiguration) SetHint(hint string) {
	c.attributedHint = NewAttributedString(hint)
	c.hasBeenAnnotated = true
}

// AttributedHint is the hint with its attributes.
func (c *SemanticsConfiguration) AttributedHint() AttributedString { return c.attributedHint }

// SetAttributedHint sets AttributedHint.
func (c *SemanticsConfiguration) SetAttributedHint(attributedHint AttributedString) {
	c.attributedHint = attributedHint
	c.hasBeenAnnotated = true
}

// Tooltip is a textual description of the widget's tooltip.
func (c *SemanticsConfiguration) Tooltip() string { return c.tooltip }

// SetTooltip sets Tooltip.
func (c *SemanticsConfiguration) SetTooltip(tooltip string) {
	c.tooltip = tooltip
	c.hasBeenAnnotated = true
}

// HintOverrides provides hint values which override the default hints.
func (c *SemanticsConfiguration) HintOverrides() *SemanticsHintOverrides { return c.hintOverrides }

// SetHintOverrides sets HintOverrides (nil is ignored).
func (c *SemanticsConfiguration) SetHintOverrides(value *SemanticsHintOverrides) {
	if value == nil {
		return
	}
	c.hintOverrides = value
	c.hasBeenAnnotated = true
}

func (c *SemanticsConfiguration) setFlags(flags ui.SemanticsFlags) {
	c.flags = flags
	c.hasBeenAnnotated = true
}

// ScopesRoute is whether the semantics node is the root of a subtree for
// which values should be announced.
func (c *SemanticsConfiguration) ScopesRoute() bool { return c.flags.ScopesRoute }

// SetScopesRoute sets ScopesRoute.
func (c *SemanticsConfiguration) SetScopesRoute(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{ScopesRoute: &value}))
}

// NamesRoute is whether the semantics node contains the label of a route.
func (c *SemanticsConfiguration) NamesRoute() bool { return c.flags.NamesRoute }

// SetNamesRoute sets NamesRoute.
func (c *SemanticsConfiguration) SetNamesRoute(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{NamesRoute: &value}))
}

// IsImage is whether the semantics node represents an image.
func (c *SemanticsConfiguration) IsImage() bool { return c.flags.IsImage }

// SetIsImage sets IsImage.
func (c *SemanticsConfiguration) SetIsImage(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsImage: &value}))
}

// LiveRegion is whether the semantics node is a live region.
func (c *SemanticsConfiguration) LiveRegion() bool { return c.flags.IsLiveRegion }

// SetLiveRegion sets LiveRegion.
func (c *SemanticsConfiguration) SetLiveRegion(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsLiveRegion: &value}))
}

// TextDirection is the reading direction for the text in Label, Value,
// Hint, IncreasedValue and DecreasedValue (nil: unknown).
func (c *SemanticsConfiguration) TextDirection() *ui.TextDirection { return c.textDirection }

// SetTextDirection sets TextDirection.
func (c *SemanticsConfiguration) SetTextDirection(textDirection *ui.TextDirection) {
	c.textDirection = textDirection
	c.hasBeenAnnotated = true
}

// IsSelected is whether the owning RenderObject is selected.
func (c *SemanticsConfiguration) IsSelected() bool { return c.flags.IsSelected == ui.TristateIsTrue }

// SetIsSelected sets IsSelected.
func (c *SemanticsConfiguration) SetIsSelected(value bool) {
	t := tristateFromBoolOrNull(&value)
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsSelected: &t}))
}

// IsExpanded is whether the owning RenderObject is expanded (nil: not
// expandable).
func (c *SemanticsConfiguration) IsExpanded() *bool { return c.flags.IsExpanded.ToBoolOrNull() }

// SetIsExpanded sets IsExpanded.
func (c *SemanticsConfiguration) SetIsExpanded(value *bool) {
	t := tristateFromBoolOrNull(value)
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsExpanded: &t}))
}

// IsEnabled is whether the owning RenderObject is enabled (nil: no
// enabled state).
func (c *SemanticsConfiguration) IsEnabled() *bool { return c.flags.IsEnabled.ToBoolOrNull() }

// SetIsEnabled sets IsEnabled.
func (c *SemanticsConfiguration) SetIsEnabled(value *bool) {
	t := tristateFromBoolOrNull(value)
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsEnabled: &t}))
}

// IsChecked is whether the owning RenderObject is checked (nil: no checked
// state).
func (c *SemanticsConfiguration) IsChecked() *bool {
	if c.flags.IsChecked == ui.CheckedStateNone {
		return nil
	}
	v := c.flags.IsChecked == ui.CheckedStateIsTrue
	return &v
}

// SetIsChecked sets IsChecked (nil leaves the state unchanged but marks
// the configuration annotated, as Flutter does).
func (c *SemanticsConfiguration) SetIsChecked(value *bool) {
	if value != nil {
		state := ui.CheckedStateIsFalse
		if *value {
			state = ui.CheckedStateIsTrue
		}
		c.flags = c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsChecked: &state})
	}
	c.hasBeenAnnotated = true
}

// IsCheckStateMixed is whether the check state is mixed (nil: no checked
// state).
func (c *SemanticsConfiguration) IsCheckStateMixed() *bool {
	if c.flags.IsChecked == ui.CheckedStateNone {
		return nil
	}
	v := c.flags.IsChecked == ui.CheckedStateMixed
	return &v
}

// SetIsCheckStateMixed sets IsCheckStateMixed (only true changes the
// state).
func (c *SemanticsConfiguration) SetIsCheckStateMixed(value *bool) {
	if value != nil && *value {
		state := ui.CheckedStateMixed
		c.flags = c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsChecked: &state})
	}
	c.hasBeenAnnotated = true
}

// IsToggled is whether the owning RenderObject is toggled (nil: no toggled
// state).
func (c *SemanticsConfiguration) IsToggled() *bool { return c.flags.IsToggled.ToBoolOrNull() }

// SetIsToggled sets IsToggled.
func (c *SemanticsConfiguration) SetIsToggled(value *bool) {
	t := tristateFromBoolOrNull(value)
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsToggled: &t}))
}

// IsInMutuallyExclusiveGroup is whether the owning RenderObject
// corresponds to UI that allows the user to pick one of several mutually
// exclusive options.
func (c *SemanticsConfiguration) IsInMutuallyExclusiveGroup() bool {
	return c.flags.IsInMutuallyExclusiveGroup
}

// SetIsInMutuallyExclusiveGroup sets IsInMutuallyExclusiveGroup.
func (c *SemanticsConfiguration) SetIsInMutuallyExclusiveGroup(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsInMutuallyExclusiveGroup: &value}))
}

// IsFocusable is whether the owning RenderObject can hold the input focus
// (Deprecated: check whether IsFocused is nil).
func (c *SemanticsConfiguration) IsFocusable() bool { return c.flags.IsFocused != ui.TristateNone }

// SetIsFocusable sets IsFocusable (Deprecated: setting IsFocused sets it).
func (c *SemanticsConfiguration) SetIsFocusable(value bool) {
	// If value is false, set `isFocused` to none. If value is true,
	// `isFocused` should be true or false: if it is none, change it to
	// false.
	if !value {
		none := ui.TristateNone
		c.flags = c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsFocused: &none})
	} else if c.flags.IsFocused == ui.TristateNone {
		f := ui.TristateIsFalse
		c.flags = c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsFocused: &f})
	}
	c.hasBeenAnnotated = true
}

// IsFocused is whether the owning RenderObject currently holds the input
// focus (nil: not focusable).
func (c *SemanticsConfiguration) IsFocused() *bool { return c.flags.IsFocused.ToBoolOrNull() }

// SetIsFocused sets IsFocused.
func (c *SemanticsConfiguration) SetIsFocused(value *bool) {
	t := tristateFromBoolOrNull(value)
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsFocused: &t}))
}

// AccessibilityFocusBlockType is how accessibility focus is blocked for
// this node.
func (c *SemanticsConfiguration) AccessibilityFocusBlockType() AccessibilityFocusBlockType {
	return c.accessibilityFocusBlockType
}

// SetAccessibilityFocusBlockType sets AccessibilityFocusBlockType.
func (c *SemanticsConfiguration) SetAccessibilityFocusBlockType(value AccessibilityFocusBlockType) {
	if c.accessibilityFocusBlockType == value {
		return
	}
	c.accessibilityFocusBlockType = value
	blocked := value != AccessibilityFocusBlockTypeNone
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsAccessibilityFocusBlocked: &blocked}))
}

// IsButton is whether the owning RenderObject is a button.
func (c *SemanticsConfiguration) IsButton() bool { return c.flags.IsButton }

// SetIsButton sets IsButton.
func (c *SemanticsConfiguration) SetIsButton(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsButton: &value}))
}

// IsLink is whether the owning RenderObject is a link.
func (c *SemanticsConfiguration) IsLink() bool { return c.flags.IsLink }

// SetIsLink sets IsLink.
func (c *SemanticsConfiguration) SetIsLink(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsLink: &value}))
}

// LinkURL is the URL that the owning RenderObject links to.
func (c *SemanticsConfiguration) LinkURL() *url.URL { return c.linkURL }

// SetLinkURL sets LinkURL.
func (c *SemanticsConfiguration) SetLinkURL(value *url.URL) {
	if urlsEqual(value, c.linkURL) {
		return
	}
	c.linkURL = value
	c.hasBeenAnnotated = true
}

// IsHeader is whether the owning RenderObject is a header.
func (c *SemanticsConfiguration) IsHeader() bool { return c.flags.IsHeader }

// SetIsHeader sets IsHeader.
func (c *SemanticsConfiguration) SetIsHeader(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsHeader: &value}))
}

// HeadingLevel is the heading level (0 when not a heading; 1 to 6).
func (c *SemanticsConfiguration) HeadingLevel() int { return c.headingLevel }

// SetHeadingLevel sets HeadingLevel (0 to 6).
func (c *SemanticsConfiguration) SetHeadingLevel(value int) {
	dartAssert(value >= 0 && value <= 6, nil)
	if value == c.HeadingLevel() {
		return
	}
	c.headingLevel = value
	c.hasBeenAnnotated = true
}

// IsSlider is whether the owning RenderObject is a slider.
func (c *SemanticsConfiguration) IsSlider() bool { return c.flags.IsSlider }

// SetIsSlider sets IsSlider.
func (c *SemanticsConfiguration) SetIsSlider(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsSlider: &value}))
}

// IsKeyboardKey is whether the owning RenderObject is a keyboard key.
func (c *SemanticsConfiguration) IsKeyboardKey() bool { return c.flags.IsKeyboardKey }

// SetIsKeyboardKey sets IsKeyboardKey.
func (c *SemanticsConfiguration) SetIsKeyboardKey(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsKeyboardKey: &value}))
}

// IsHidden is whether the owning RenderObject is considered hidden.
func (c *SemanticsConfiguration) IsHidden() bool { return c.flags.IsHidden }

// SetIsHidden sets IsHidden.
func (c *SemanticsConfiguration) SetIsHidden(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsHidden: &value}))
}

// IsTextField is whether the owning RenderObject is a text field.
func (c *SemanticsConfiguration) IsTextField() bool { return c.flags.IsTextField }

// SetIsTextField sets IsTextField.
func (c *SemanticsConfiguration) SetIsTextField(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsTextField: &value}))
}

// IsReadOnly is whether the owning RenderObject is read only.
func (c *SemanticsConfiguration) IsReadOnly() bool { return c.flags.IsReadOnly }

// SetIsReadOnly sets IsReadOnly.
func (c *SemanticsConfiguration) SetIsReadOnly(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsReadOnly: &value}))
}

// IsObscured is whether the Value should be obscured.
func (c *SemanticsConfiguration) IsObscured() bool { return c.flags.IsObscured }

// SetIsObscured sets IsObscured.
func (c *SemanticsConfiguration) SetIsObscured(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsObscured: &value}))
}

// IsMultiline is whether the text field is multiline.
func (c *SemanticsConfiguration) IsMultiline() bool { return c.flags.IsMultiline }

// SetIsMultiline sets IsMultiline.
func (c *SemanticsConfiguration) SetIsMultiline(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsMultiline: &value}))
}

// IsRequired is whether the owning RenderObject is required (nil: no
// required state).
func (c *SemanticsConfiguration) IsRequired() *bool { return c.flags.IsRequired.ToBoolOrNull() }

// SetIsRequired sets IsRequired.
func (c *SemanticsConfiguration) SetIsRequired(value *bool) {
	t := tristateFromBoolOrNull(value)
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{IsRequired: &t}))
}

// HasImplicitScrolling is whether the platform can scroll the semantics
// node when the user attempts to move focus to an offscreen child.
func (c *SemanticsConfiguration) HasImplicitScrolling() bool { return c.flags.HasImplicitScrolling }

// SetHasImplicitScrolling sets HasImplicitScrolling.
func (c *SemanticsConfiguration) SetHasImplicitScrolling(value bool) {
	c.setFlags(c.flags.CopyWith(ui.SemanticsFlagsCopyWithOptions{HasImplicitScrolling: &value}))
}

// TextSelection is the currently selected text.
func (c *SemanticsConfiguration) TextSelection() *services.TextSelection { return c.textSelection }

// SetTextSelection sets TextSelection (not nil).
func (c *SemanticsConfiguration) SetTextSelection(value *services.TextSelection) {
	dartAssert(value != nil, nil)
	c.textSelection = value
	c.hasBeenAnnotated = true
}

// ScrollPosition indicates the current scrolling position in logical
// pixels.
func (c *SemanticsConfiguration) ScrollPosition() *float64 { return c.scrollPosition }

// SetScrollPosition sets ScrollPosition (not nil).
func (c *SemanticsConfiguration) SetScrollPosition(value *float64) {
	dartAssert(value != nil, nil)
	c.scrollPosition = value
	c.hasBeenAnnotated = true
}

// ScrollExtentMax indicates the maximum in-range value for ScrollPosition.
func (c *SemanticsConfiguration) ScrollExtentMax() *float64 { return c.scrollExtentMax }

// SetScrollExtentMax sets ScrollExtentMax (not nil).
func (c *SemanticsConfiguration) SetScrollExtentMax(value *float64) {
	dartAssert(value != nil, nil)
	c.scrollExtentMax = value
	c.hasBeenAnnotated = true
}

// ScrollExtentMin indicates the minimum in-range value for ScrollPosition.
func (c *SemanticsConfiguration) ScrollExtentMin() *float64 { return c.scrollExtentMin }

// SetScrollExtentMin sets ScrollExtentMin (not nil).
func (c *SemanticsConfiguration) SetScrollExtentMin(value *float64) {
	dartAssert(value != nil, nil)
	c.scrollExtentMin = value
	c.hasBeenAnnotated = true
}

// ControlsNodes is the identifiers of the nodes this node controls.
func (c *SemanticsConfiguration) ControlsNodes() []string { return c.controlsNodes }

// SetControlsNodes sets ControlsNodes (not nil; a set: no duplicates).
func (c *SemanticsConfiguration) SetControlsNodes(value []string) {
	dartAssert(value != nil, nil)
	c.controlsNodes = value
	c.hasBeenAnnotated = true
}

// ValidationResult is the validation result for this node.
func (c *SemanticsConfiguration) ValidationResult() ui.SemanticsValidationResult {
	return c.validationResult
}

// SetValidationResult sets ValidationResult.
func (c *SemanticsConfiguration) SetValidationResult(value ui.SemanticsValidationResult) {
	c.validationResult = value
	c.hasBeenAnnotated = true
}

// HitTestBehavior is how this node participates in platform hit testing.
func (c *SemanticsConfiguration) HitTestBehavior() ui.SemanticsHitTestBehavior {
	return c.hitTestBehavior
}

// SetHitTestBehavior sets HitTestBehavior.
func (c *SemanticsConfiguration) SetHitTestBehavior(value ui.SemanticsHitTestBehavior) {
	c.hitTestBehavior = value
	c.hasBeenAnnotated = true
}

// InputType is the input type for this node.
func (c *SemanticsConfiguration) InputType() ui.SemanticsInputType { return c.inputType }

// SetInputType sets InputType.
func (c *SemanticsConfiguration) SetInputType(value ui.SemanticsInputType) {
	c.inputType = value
	c.hasBeenAnnotated = true
}

// MaxValue is the maximum value of the node.
func (c *SemanticsConfiguration) MaxValue() *string { return c.maxValue }

// SetMaxValue sets MaxValue.
func (c *SemanticsConfiguration) SetMaxValue(value *string) {
	c.maxValue = value
	c.hasBeenAnnotated = true
}

// MinValue is the minimum value of the node.
func (c *SemanticsConfiguration) MinValue() *string { return c.minValue }

// SetMinValue sets MinValue.
func (c *SemanticsConfiguration) SetMinValue(value *string) {
	c.minValue = value
	c.hasBeenAnnotated = true
}

// ---- TAGS

// TagsForChildren is the SemanticsTags that the children of this node
// should be tagged with (nil: none).
func (c *SemanticsConfiguration) TagsForChildren() []*SemanticsTag { return c.tagsForChildren }

// TagsChildrenWith reports whether the children are tagged with tag.
func (c *SemanticsConfiguration) TagsChildrenWith(tag *SemanticsTag) bool {
	return containsTag(c.tagsForChildren, tag)
}

// AddTagForChildren specifies a SemanticsTag that this configuration
// wants to apply to all child SemanticsNodes.
func (c *SemanticsConfiguration) AddTagForChildren(tag *SemanticsTag) {
	if c.tagsForChildren == nil {
		c.tagsForChildren = []*SemanticsTag{}
	}
	c.tagsForChildren = addTags(c.tagsForChildren, []*SemanticsTag{tag})
}

// ---- INTERNAL FLAG MANAGEMENT

func (c *SemanticsConfiguration) hasExplicitRole() bool {
	if c.role != ui.SemanticsRoleNone {
		return true
	}
	if c.flags.IsTextField ||
		// In non web platforms, the header is a trait.
		(c.flags.IsHeader && foundation.KIsWeb) ||
		c.flags.IsSlider ||
		c.flags.IsLink ||
		c.flags.ScopesRoute ||
		c.flags.IsImage ||
		c.flags.IsKeyboardKey {
		return true
	}
	return false
}

// ---- CONFIGURATION COMBINATION LOGIC

// IsCompatibleWith reports whether this configuration is compatible with
// other (nil is compatible): whether they can be merged into one node.
func (c *SemanticsConfiguration) IsCompatibleWith(other *SemanticsConfiguration) bool {
	if other == nil || !other.HasBeenAnnotated() {
		return true
	}
	// The parent node should reject child node as long as their
	// traversalChildIdentifiers are different, even if the parent node has
	// not been annotated.
	if !objectsEqual(c.traversalChildIdentifier, other.traversalChildIdentifier) {
		return false
	}
	if !c.HasBeenAnnotated() {
		return true
	}
	if c.actionsAsBits&other.actionsAsBits != 0 {
		return false
	}
	if c.flags.HasConflictingFlags(other.flags) {
		return false
	}
	if c.platformViewID != nil && other.platformViewID != nil {
		return false
	}
	if c.maxValueLength != nil && other.maxValueLength != nil {
		return false
	}
	if c.currentValueLength != nil && other.currentValueLength != nil {
		return false
	}
	if c.attributedValue.Text != "" && other.attributedValue.Text != "" {
		return false
	}
	if !localesEqual(c.localeForSubtree, other.localeForSubtree) {
		return false
	}
	if c.hasExplicitRole() && other.hasExplicitRole() {
		return false
	}
	if c.hitTestBehavior != ui.SemanticsHitTestBehaviorDefer || other.hitTestBehavior != ui.SemanticsHitTestBehaviorDefer {
		return false
	}
	if c.minValue != nil && other.minValue != nil {
		return false
	}
	if c.maxValue != nil && other.maxValue != nil {
		return false
	}
	return true
}

func localesEqual(a, b *ui.Locale) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

// Absorb absorbs the semantic information provided by child into this
// configuration.
func (c *SemanticsConfiguration) Absorb(child *SemanticsConfiguration) {
	dartAssert(!c.ExplicitChildNodes, nil)

	if !child.HasBeenAnnotated() {
		return
	}
	if child.IsBlockingUserActions {
		for key, value := range child.actions {
			if kUnblockedUserActions&key.Index() > 0 {
				c.actions[key] = value
			}
		}
	} else {
		for key, value := range child.actions {
			c.actions[key] = value
		}
	}
	c.actionsAsBits |= child.effectiveActionsAsBits()
	for _, e := range child.customSemanticsActions {
		c.customSemanticsActions = putCustomAction(c.customSemanticsActions, e.Key, e.Value)
	}
	c.flags = c.flags.Merge(child.flags)
	c.linkURL = orFirst(c.linkURL, child.linkURL)
	c.textSelection = orFirst(c.textSelection, child.textSelection)
	c.scrollPosition = orFirst(c.scrollPosition, child.scrollPosition)
	c.scrollExtentMax = orFirst(c.scrollExtentMax, child.scrollExtentMax)
	c.scrollExtentMin = orFirst(c.scrollExtentMin, child.scrollExtentMin)
	c.hintOverrides = orFirst(c.hintOverrides, child.hintOverrides)
	c.indexInParent = orFirst(c.indexInParent, child.IndexInParent())
	c.scrollIndex = orFirst(c.scrollIndex, child.scrollIndex)
	c.scrollChildCount = orFirst(c.scrollChildCount, child.scrollChildCount)
	c.platformViewID = orFirst(c.platformViewID, child.platformViewID)
	c.maxValueLength = orFirst(c.maxValueLength, child.maxValueLength)
	c.currentValueLength = orFirst(c.currentValueLength, child.currentValueLength)

	// A node cannot have both `_traversalChildIdentifier` and
	// `_traversalParentIdentifier` not null.
	if c.traversalChildIdentifier == nil && c.traversalParentIdentifier == nil {
		c.traversalParentIdentifier = child.traversalParentIdentifier
	}
	if c.traversalChildIdentifier == nil {
		c.traversalChildIdentifier = child.traversalChildIdentifier
	}

	c.headingLevel = mergeHeadingLevels(child.headingLevel, c.headingLevel)

	if c.TextDirection() == nil {
		c.SetTextDirection(child.TextDirection())
	}
	if c.sortKey == nil {
		c.sortKey = child.sortKey
	}
	if c.identifier == "" {
		c.identifier = child.identifier
	}
	c.attributedLabel = concatAttributedString(c.attributedLabel, c.TextDirection(), child.attributedLabel, child.TextDirection())
	if c.attributedValue.Text == "" {
		c.attributedValue = child.attributedValue
	}
	if c.attributedIncreasedValue.Text == "" {
		c.attributedIncreasedValue = child.attributedIncreasedValue
	}
	if c.attributedDecreasedValue.Text == "" {
		c.attributedDecreasedValue = child.attributedDecreasedValue
	}
	if c.role == ui.SemanticsRoleNone {
		c.role = child.role
	}
	if c.inputType == ui.SemanticsInputTypeNone {
		c.inputType = child.inputType
	}
	c.attributedHint = concatAttributedString(c.attributedHint, c.TextDirection(), child.attributedHint, child.TextDirection())
	if c.tooltip == "" {
		c.tooltip = child.tooltip
	}

	if c.controlsNodes == nil {
		c.controlsNodes = child.controlsNodes
	} else if child.controlsNodes != nil {
		c.controlsNodes = unionStrings(c.controlsNodes, child.controlsNodes)
	}

	if child.validationResult != c.validationResult {
		if child.validationResult == ui.SemanticsValidationResultInvalid {
			// Invalid result always takes precedence.
			c.validationResult = ui.SemanticsValidationResultInvalid
		} else if c.validationResult == ui.SemanticsValidationResultNone {
			c.validationResult = child.validationResult
		}
	}
	c.accessibilityFocusBlockType = c.accessibilityFocusBlockType.merge(child.accessibilityFocusBlockType)

	c.minValue = orFirst(c.minValue, child.minValue)
	c.maxValue = orFirst(c.maxValue, child.maxValue)
	if c.hitTestBehavior == ui.SemanticsHitTestBehaviorDefer &&
		child.hitTestBehavior != ui.SemanticsHitTestBehaviorDefer {
		c.hitTestBehavior = child.hitTestBehavior
	}

	c.hasBeenAnnotated = c.HasBeenAnnotated() || child.HasBeenAnnotated()
}

// Copy returns an exact copy of this configuration.
func (c *SemanticsConfiguration) Copy() *SemanticsConfiguration {
	r := NewSemanticsConfiguration()
	r.isSemanticBoundary = c.isSemanticBoundary
	r.ExplicitChildNodes = c.ExplicitChildNodes
	r.IsBlockingSemanticsOfPreviouslyPaintedNodes = c.IsBlockingSemanticsOfPreviouslyPaintedNodes
	r.hasBeenAnnotated = c.HasBeenAnnotated()
	r.isMergingSemanticsOfDescendants = c.isMergingSemanticsOfDescendants
	r.textDirection = c.textDirection
	r.sortKey = c.sortKey
	r.identifier = c.identifier
	r.traversalParentIdentifier = c.traversalParentIdentifier
	r.traversalChildIdentifier = c.traversalChildIdentifier
	r.attributedLabel = c.attributedLabel
	r.attributedIncreasedValue = c.attributedIncreasedValue
	r.attributedValue = c.attributedValue
	r.attributedDecreasedValue = c.attributedDecreasedValue
	r.attributedHint = c.attributedHint
	r.accessibilityFocusBlockType = c.accessibilityFocusBlockType
	r.hintOverrides = c.hintOverrides
	r.tooltip = c.tooltip
	r.flags = c.flags
	r.tagsForChildren = c.tagsForChildren
	r.textSelection = c.textSelection
	r.scrollPosition = c.scrollPosition
	r.scrollExtentMax = c.scrollExtentMax
	r.scrollExtentMin = c.scrollExtentMin
	r.actionsAsBits = c.actionsAsBits
	r.indexInParent = c.IndexInParent()
	r.scrollIndex = c.scrollIndex
	r.scrollChildCount = c.scrollChildCount
	r.platformViewID = c.platformViewID
	r.maxValueLength = c.maxValueLength
	r.currentValueLength = c.currentValueLength
	for k, v := range c.actions {
		r.actions[k] = v
	}
	for _, e := range c.customSemanticsActions {
		r.customSemanticsActions = putCustomAction(r.customSemanticsActions, e.Key, e.Value)
	}
	r.IsBlockingUserActions = c.IsBlockingUserActions
	r.headingLevel = c.headingLevel
	r.linkURL = c.linkURL
	r.role = c.role
	r.controlsNodes = c.controlsNodes
	r.validationResult = c.validationResult
	r.inputType = c.inputType
	r.hitTestBehavior = c.hitTestBehavior
	r.minValue = c.minValue
	r.maxValue = c.maxValue
	return r
}

// DebugSemanticsDumpOrder is used by debugDumpSemanticsTree to specify the
// order in which child nodes are printed.
type DebugSemanticsDumpOrder int

const (
	// DebugSemanticsDumpOrderInverseHitTest prints nodes in inverse hit
	// test order.
	DebugSemanticsDumpOrderInverseHitTest DebugSemanticsDumpOrder = iota
	// DebugSemanticsDumpOrderTraversalOrder prints nodes in semantic
	// traversal order.
	DebugSemanticsDumpOrderTraversalOrder
)

// DebugSemanticsDumpOrderValues is DebugSemanticsDumpOrder.values.
var DebugSemanticsDumpOrderValues = []DebugSemanticsDumpOrder{DebugSemanticsDumpOrderInverseHitTest,
	DebugSemanticsDumpOrderTraversalOrder}

// Name is the enum value's name.
func (o DebugSemanticsDumpOrder) Name() string {
	return [...]string{"inverseHitTest", "traversalOrder"}[o]
}

func (o DebugSemanticsDumpOrder) String() string { return "DebugSemanticsDumpOrder." + o.Name() }

func concatAttributedString(thisAttributedString AttributedString, thisTextDirection *ui.TextDirection,
	otherAttributedString AttributedString, otherTextDirection *ui.TextDirection) AttributedString {
	if otherAttributedString.Text == "" {
		return thisAttributedString
	}
	if !ptrEqual(thisTextDirection, otherTextDirection) && otherTextDirection != nil {
		directionEmbedding := NewAttributedString(foundation.Unicode.LRE)
		if *otherTextDirection == ui.TextDirectionRtl {
			directionEmbedding = NewAttributedString(foundation.Unicode.RLE)
		}
		otherAttributedString = directionEmbedding.Add(otherAttributedString).Add(NewAttributedString(foundation.Unicode.PDF))
	}
	if thisAttributedString.Text == "" {
		return otherAttributedString
	}
	return thisAttributedString.Add(NewAttributedString("\n")).Add(otherAttributedString)
}

// ---- SemanticsSortKey

// SemanticsSortKey is the base for all sort keys: it orders a node among
// its siblings in the traversal order. Keys of different types (or
// names) are not compared. Implement it by embedding SemanticsSortKeyBase
// (and calling SetSelf) and defining DoCompare. Sort keys are compared by
// identity (Dart's SemanticsSortKey has no ==): use them by pointer.
type SemanticsSortKey interface {
	foundation.Diagnosticable
	// Name is an optional name that groups keys: only keys with the same
	// name are compared (nil: none).
	Name() *string
	// CompareTo compares this key to other (of the same type), by name
	// first, then with DoCompare.
	CompareTo(other SemanticsSortKey) int
	// DoCompare compares this key to other, of the same type and name.
	DoCompare(other SemanticsSortKey) int
}

// SemanticsSortKeyBase is the abstract class SemanticsSortKey's state and
// default methods.
type SemanticsSortKeyBase struct {
	foundation.DiagnosticableMixin
	name *string
	self SemanticsSortKey
}

// NewSemanticsSortKeyBase is SemanticsSortKey({name}), for embedding.
func NewSemanticsSortKeyBase(name *string) SemanticsSortKeyBase {
	return SemanticsSortKeyBase{name: name}
}

// SetSelf points the base at the key that embeds it.
func (k *SemanticsSortKeyBase) SetSelf(self SemanticsSortKey) {
	k.self = self
	k.DiagnosticableMixin.SetSelf(self)
}

func (k *SemanticsSortKeyBase) outer() SemanticsSortKey {
	if k.self == nil {
		panic("semantics: SemanticsSortKeyBase used before SetSelf; call SetSelf with the outer key when creating it")
	}
	return k.self
}

// Name is the key's name (nil: none).
func (k *SemanticsSortKeyBase) Name() *string { return k.name }

// CompareTo sorts by name first (keys without a name first), then by
// DoCompare.
func (k *SemanticsSortKeyBase) CompareTo(other SemanticsSortKey) int {
	self := k.outer()
	// Sort by name first and then subclass ordering.
	dartAssert(runtimeType(self) == runtimeType(other),
		"Semantics sort keys can only be compared to other sort keys of the same type.")

	// Defer to the subclass implementation for ordering only if the names
	// are identical (or both null).
	if ptrEqual(k.name, other.Name()) {
		return self.DoCompare(other)
	}

	// Keys that don't have a name are sorted together and come before those
	// with a name.
	if k.name == nil && other.Name() != nil {
		return -1
	} else if k.name != nil && other.Name() == nil {
		return 1
	}

	return dart.CompareStrings(*k.name, *other.Name())
}

func (k *SemanticsSortKeyBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	k.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewStringProperty("name", k.name, foundation.StringPropertyOptions{DefaultValue: foundation.Null}))
}

// OrdinalSortKey is a SemanticsSortKey that sorts based on the numerical
// order of Order.
type OrdinalSortKey struct {
	SemanticsSortKeyBase
	// Order determines the placement of this key in a sequence of keys
	// that obey the same name; it must be finite.
	Order float64
}

// OrdinalSortKeyOptions are OrdinalSortKey's named parameters.
type OrdinalSortKeyOptions struct{ Name *string }

// NewOrdinalSortKey is OrdinalSortKey(order, {name}).
func NewOrdinalSortKey(order float64, opts ...OrdinalSortKeyOptions) *OrdinalSortKey {
	var o OrdinalSortKeyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(order > math.Inf(-1), nil)
	dartAssert(order < math.Inf(1), nil)
	k := &OrdinalSortKey{SemanticsSortKeyBase: NewSemanticsSortKeyBase(o.Name), Order: order}
	k.SetSelf(k)
	return k
}

// DoCompare compares the orders.
func (k *OrdinalSortKey) DoCompare(other SemanticsSortKey) int {
	o := other.(*OrdinalSortKey)
	if o.Order == k.Order {
		return 0
	}
	return dart.CompareDoubles(k.Order, o.Order)
}

func (k *OrdinalSortKey) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	k.SemanticsSortKeyBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDoubleProperty("order", k.Order, foundation.DoublePropertyOptions{DefaultValue: foundation.Null}))
}

func mergeHeadingLevels(sourceLevel, targetLevel int) int {
	if targetLevel == 0 {
		return sourceLevel
	}
	return targetLevel
}

func tristateFromBoolOrNull(value *bool) ui.Tristate {
	if value == nil {
		return ui.TristateNone
	}
	if *value {
		return ui.TristateIsTrue
	}
	return ui.TristateIsFalse
}

func toBitMask(flags ui.SemanticsFlags) int {
	bitmask := 0
	set := func(cond bool, bit int) {
		if cond {
			bitmask |= 1 << bit
		}
	}
	set(flags.IsChecked != ui.CheckedStateNone, 0)
	set(flags.IsChecked == ui.CheckedStateIsTrue, 1)
	set(flags.IsSelected == ui.TristateIsTrue, 2)
	set(flags.IsButton, 3)
	set(flags.IsTextField, 4)
	set(flags.IsFocused == ui.TristateIsTrue, 5)
	set(flags.IsEnabled != ui.TristateNone, 6)
	set(flags.IsEnabled == ui.TristateIsTrue, 7)
	set(flags.IsInMutuallyExclusiveGroup, 8)
	set(flags.IsHeader, 9)
	set(flags.IsObscured, 10)
	set(flags.ScopesRoute, 11)
	set(flags.NamesRoute, 12)
	set(flags.IsHidden, 13)
	set(flags.IsImage, 14)
	set(flags.IsLiveRegion, 15)
	set(flags.IsToggled != ui.TristateNone, 16)
	set(flags.IsToggled == ui.TristateIsTrue, 17)
	set(flags.HasImplicitScrolling, 18)
	set(flags.IsMultiline, 19)
	set(flags.IsReadOnly, 20)
	set(flags.IsFocused != ui.TristateNone, 21)
	set(flags.IsLink, 22)
	set(flags.IsSlider, 23)
	set(flags.IsKeyboardKey, 24)
	set(flags.IsChecked == ui.CheckedStateMixed, 25)
	set(flags.IsExpanded != ui.TristateNone, 26)
	set(flags.IsExpanded == ui.TristateIsTrue, 27)
	set(flags.IsSelected != ui.TristateNone, 28)
	set(flags.IsRequired != ui.TristateNone, 29)
	set(flags.IsRequired == ui.TristateIsTrue, 30)
	return bitmask
}
