// Package semantics is package:flutter/semantics.dart (Flutter 3.50.1):
// the semantics tree (SemanticsNode, SemanticsOwner), its annotations
// (SemanticsConfiguration, SemanticsProperties, SemanticsData,
// AttributedString, sort keys, custom actions), the SemanticsBinding mixin,
// semantics events and SemanticsService.
//
// semantics.dart is ported in four files: semantics.go (typedefs, the
// role checks, tags, custom actions, attributed strings, SemanticsData,
// SemanticsHintOverrides, SemanticsProperties), semantics_node.go
// (SemanticsNode and the traversal sort), semantics_owner.go
// (SemanticsOwner) and semantics_configuration.go (SemanticsConfiguration,
// the sort keys and the merge helpers).
//
// Go mapping notes (see src/docs/API_MAPPING.md):
//   - Dart's Sets and Maps whose iteration order is observable (tags,
//     controlsNodes, custom actions, the owner's dirty nodes) keep Dart's
//     insertion order: tags and controlsNodes are slices without duplicates,
//     a custom action map is []dart.MapEntry[CustomSemanticsAction, func()].
//   - Uri is *url.URL; Matrix4 is *vector_math.Matrix4; nullable values are
//     pointers.
//   - List.sort is dart.ListSort (Dart's unstable sort, whose tie order is
//     observable in the traversal order).
package semantics

import (
	"net/url"
	"strconv"
	"strings"
	"unicode/utf16"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// SemanticsNodeVisitor is the signature of SemanticsNode.VisitChildren:
// return false to stop visiting.
type SemanticsNodeVisitor func(node *SemanticsNode) bool

// MoveCursorHandler is the signature of handlers for cursor movement
// actions.
type MoveCursorHandler func(extendSelection bool)

// SetSelectionHandler is the signature of handlers for the setSelection
// action.
type SetSelectionHandler func(selection services.TextSelection)

// SetTextHandler is the signature of handlers for the setText action.
type SetTextHandler func(text string)

// ScrollToOffsetHandler is the signature of handlers for the
// scrollToOffset action.
type ScrollToOffsetHandler func(targetOffset ui.Offset)

// SemanticsActionHandler is the signature of handlers for semantics
// actions (args is nil for Dart's null).
type SemanticsActionHandler func(args any)

// SemanticsUpdateCallback is the signature of a callback that receives a
// built semantics update.
type SemanticsUpdateCallback func(update ui.SemanticsUpdate)

// ChildSemanticsConfigurationsDelegate is the signature of the function
// that decides how the configurations of a node's children merge.
type ChildSemanticsConfigurationsDelegate func(configs []*SemanticsConfiguration) *ChildSemanticsConfigurationsResult

// AccessibilityFocusBlockType describes how accessibility focus is blocked
// for a node and its subtree.
type AccessibilityFocusBlockType int

const (
	// AccessibilityFocusBlockTypeNone: accessibility focus is not blocked.
	AccessibilityFocusBlockTypeNone AccessibilityFocusBlockType = iota
	// AccessibilityFocusBlockTypeBlockSubtree: blocks accessibility focus
	// for the entire subtree.
	AccessibilityFocusBlockTypeBlockSubtree
	// AccessibilityFocusBlockTypeBlockNode: blocks accessibility focus for
	// the node only.
	AccessibilityFocusBlockTypeBlockNode
)

// AccessibilityFocusBlockTypeValues is AccessibilityFocusBlockType.values.
var AccessibilityFocusBlockTypeValues = []AccessibilityFocusBlockType{AccessibilityFocusBlockTypeNone,
	AccessibilityFocusBlockTypeBlockSubtree, AccessibilityFocusBlockTypeBlockNode}

// Name is the enum value's name.
func (t AccessibilityFocusBlockType) Name() string {
	return [...]string{"none", "blockSubtree", "blockNode"}[t]
}

func (t AccessibilityFocusBlockType) String() string {
	return "AccessibilityFocusBlockType." + t.Name()
}

func (t AccessibilityFocusBlockType) merge(other AccessibilityFocusBlockType) AccessibilityFocusBlockType {
	// 1. If either is blockSubtree, the result is blockSubtree.
	if t == AccessibilityFocusBlockTypeBlockSubtree || other == AccessibilityFocusBlockTypeBlockSubtree {
		return AccessibilityFocusBlockTypeBlockSubtree
	}
	// 2. If either is blockNode, the result is blockNode
	if t == AccessibilityFocusBlockTypeBlockNode || other == AccessibilityFocusBlockTypeBlockNode {
		return AccessibilityFocusBlockTypeBlockNode
	}
	// 3. If neither is blockSubtree nor blockNode, both must be none.
	return AccessibilityFocusBlockTypeNone
}

var kUnblockedUserActions = ui.SemanticsActionDidGainAccessibilityFocus.Index() |
	ui.SemanticsActionDidLoseAccessibilityFocus.Index()

func dartAssert(cond bool, message any) {
	if !cond {
		panic(dart.NewAssertionError(message))
	}
}

// ---- _DebugSemanticsRoleChecks

func checkSemanticsData(node *SemanticsNode) *foundation.FlutterError {
	var check func(*SemanticsNode) *foundation.FlutterError
	switch node.Role() {
	case ui.SemanticsRoleAlertDialog, ui.SemanticsRoleDialog, ui.SemanticsRoleNone, ui.SemanticsRoleTabPanel,
		ui.SemanticsRoleList, ui.SemanticsRoleForm, ui.SemanticsRoleLoadingSpinner:
		check = noCheckRequired
	case ui.SemanticsRoleTab:
		check = semanticsTab
	case ui.SemanticsRoleTabBar:
		check = semanticsTabBar
	case ui.SemanticsRoleTable:
		check = semanticsTable
	case ui.SemanticsRoleCell:
		check = semanticsCell
	case ui.SemanticsRoleRow:
		check = semanticsRow
	case ui.SemanticsRoleColumnHeader:
		check = semanticsColumnHeader
	case ui.SemanticsRoleRadioGroup:
		check = semanticsRadioGroup
	case ui.SemanticsRoleMenu:
		check = semanticsMenu
	case ui.SemanticsRoleMenuBar:
		check = semanticsMenuBar
	case ui.SemanticsRoleMenuItem:
		check = semanticsMenuItem
	case ui.SemanticsRoleMenuItemCheckbox:
		check = semanticsMenuItemCheckbox
	case ui.SemanticsRoleMenuItemRadio:
		check = semanticsMenuItemRadio
	case ui.SemanticsRoleAlert, ui.SemanticsRoleStatus:
		check = noLiveRegion
	case ui.SemanticsRoleListItem:
		check = semanticsListItem
	case ui.SemanticsRoleComplementary:
		check = semanticsComplementary
	case ui.SemanticsRoleContentInfo:
		check = semanticsContentInfo
	case ui.SemanticsRoleMain:
		check = semanticsMain
	case ui.SemanticsRoleNavigation:
		check = semanticsNavigation
	case ui.SemanticsRoleRegion:
		check = semanticsRegion
	case ui.SemanticsRoleProgressBar:
		check = semanticsProgressBar
	case ui.SemanticsRoleSlider:
		check = semanticsSlider
	default: // dragHandle, spinButton, comboBox, tooltip, hotKey
		check = unimplementedRoleCheck
	}
	if err := check(node); err != nil {
		return err
	}
	return semanticsGeneral(node)
}

func unimplementedRoleCheck(node *SemanticsNode) *foundation.FlutterError {
	return foundation.NewFlutterError("Missing checks for role " + node.GetSemanticsData().Role.String())
}

func noCheckRequired(*SemanticsNode) *foundation.FlutterError { return nil }

func strOrNull(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func semanticsProgressBar(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	// Check if value is present
	if data.Value() == "" || data.MinValue == nil || *data.MinValue == "" || data.MaxValue == nil || *data.MaxValue == "" {
		return foundation.NewFlutterError("A progress bar must have a value, a minValue, a maxValue.")
	}
	minVal, minOK := dart.DoubleTryParse(*data.MinValue)
	maxVal, maxOK := dart.DoubleTryParse(*data.MaxValue)
	// The value can either be a direct number fall within the min and max
	// range or a percentage number between 0% and 100%
	currentValue, currentOK := dart.DoubleTryParse(data.Value())
	var percentValue float64
	percentOK := false
	if strings.HasSuffix(data.Value(), "%") {
		u := utf16.Encode([]rune(data.Value()))
		percentValue, percentOK = dart.DoubleTryParse(string(utf16.Decode(u[:len(u)-1])))
	}
	if !minOK || !maxOK || (!currentOK && !percentOK) {
		return foundation.NewFlutterError("Progress bar value, minValue, and maxValue must be valid numbers. " +
			`value: "` + data.Value() + `", minValue: "` + strOrNull(data.MinValue) + `", maxValue: "` +
			strOrNull(data.MaxValue) + `"`)
	}
	// Validate min/max relation since both are guaranteed to be valid numbers.
	if minVal >= maxVal {
		return foundation.NewFlutterError("Progress bar minValue (" + strOrNull(data.MinValue) +
			") must be less than maxValue (" + strOrNull(data.MaxValue) + ")")
	}
	// If the value is a number, it must be between min and max.
	if currentOK {
		if currentValue < minVal || currentValue > maxVal {
			return foundation.NewFlutterError("Progress bar value (" + data.Value() + ") must be between minValue (" +
				strOrNull(data.MinValue) + ") and maxValue (" + strOrNull(data.MaxValue) + ")")
		}
		return nil
	}
	// If the value is a percentage string (e.g., "50%"), it must be between
	// 0% and 100%.
	if percentOK && (percentValue < 0 || percentValue > 100) {
		return foundation.NewFlutterError("Progress bar percentage value (" + data.Value() +
			") must be between 0% and 100%")
	}
	return nil
}

func semanticsSlider(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	// Check if value is present
	if data.Value() == "" {
		return foundation.NewFlutterError("A slider must have a value")
	}
	// Check if minValue and maxValue are present
	if data.MinValue == nil || *data.MinValue == "" {
		return foundation.NewFlutterError("A slider must have a minValue")
	}
	if data.MaxValue == nil || *data.MaxValue == "" {
		return foundation.NewFlutterError("A slider must have a maxValue")
	}
	// Validate that value is within min and max range (double.parse throws
	// a FormatException on the first invalid number).
	currentValue, ok1 := dart.DoubleTryParse(strings.ReplaceAll(data.Value(), "%", ""))
	minVal, ok2 := dart.DoubleTryParse(*data.MinValue)
	maxVal, ok3 := dart.DoubleTryParse(*data.MaxValue)
	if !ok1 || !ok2 || !ok3 {
		return foundation.NewFlutterError("Slider value, minValue, and maxValue must be valid numbers. " +
			`value: "` + data.Value() + `", minValue: "` + strOrNull(data.MinValue) + `", maxValue: "` +
			strOrNull(data.MaxValue) + `"`)
	}
	if minVal >= maxVal {
		return foundation.NewFlutterError("Slider minValue (" + strOrNull(data.MinValue) +
			") must be less than maxValue (" + strOrNull(data.MaxValue) + ")")
	}
	if strings.Contains(data.Value(), "%") {
		if currentValue < 0.0 || currentValue > 100.0 {
			return foundation.NewFlutterError("Slider percentage value (" + data.Value() +
				") must be between 0% and 100%")
		}
	} else if currentValue < minVal || currentValue > maxVal {
		return foundation.NewFlutterError("Slider value (" + data.Value() + ") must be between minValue (" +
			strOrNull(data.MinValue) + ") and maxValue (" + strOrNull(data.MaxValue) + ")")
	}
	return nil
}

func semanticsTab(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	if data.FlagsCollection.IsSelected == ui.TristateNone {
		return foundation.NewFlutterError("A tab needs selected states")
	}
	if node.AreUserActionsBlocked() {
		return nil
	}
	if data.FlagsCollection.IsEnabled != ui.TristateIsFalse && !data.HasAction(ui.SemanticsActionTap) {
		return foundation.NewFlutterError("A tab must have a tap action")
	}
	return nil
}

func semanticsTabBar(node *SemanticsNode) *foundation.FlutterError {
	if node.ChildrenCount() < 1 {
		return foundation.NewFlutterError("a TabBar cannot be empty")
	}
	var err *foundation.FlutterError
	node.VisitChildren(func(child *SemanticsNode) bool {
		if child.GetSemanticsData().Role != ui.SemanticsRoleTab {
			err = foundation.NewFlutterError("Children of TabBar must have the tab role")
		}
		return err == nil
	})
	return err
}

func semanticsTable(node *SemanticsNode) *foundation.FlutterError {
	var err *foundation.FlutterError
	node.VisitChildren(func(child *SemanticsNode) bool {
		if child.GetSemanticsData().Role != ui.SemanticsRoleRow {
			err = foundation.NewFlutterError("Children of Table must have the row role")
		}
		return err == nil
	})
	return err
}

func parentRole(node *SemanticsNode) (ui.SemanticsRole, bool) {
	if node.Parent() == nil {
		return 0, false
	}
	return node.Parent().Role(), true
}

func semanticsRow(node *SemanticsNode) *foundation.FlutterError {
	if r, ok := parentRole(node); !ok || r != ui.SemanticsRoleTable {
		return foundation.NewFlutterError("A row must be a child of a table")
	}
	var err *foundation.FlutterError
	node.VisitChildren(func(child *SemanticsNode) bool {
		if child.GetSemanticsData().Role != ui.SemanticsRoleCell &&
			child.GetSemanticsData().Role != ui.SemanticsRoleColumnHeader {
			err = foundation.NewFlutterError("Children of Row must have the cell or columnHeader role")
		}
		return err == nil
	})
	return err
}

func semanticsCell(node *SemanticsNode) *foundation.FlutterError {
	if r, ok := parentRole(node); !ok || (r != ui.SemanticsRoleRow && r != ui.SemanticsRoleCell) {
		return foundation.NewFlutterError("A cell must be a child of a row or another cell")
	}
	return nil
}

func semanticsColumnHeader(node *SemanticsNode) *foundation.FlutterError {
	if r, ok := parentRole(node); !ok || (r != ui.SemanticsRoleRow && r != ui.SemanticsRoleCell) {
		return foundation.NewFlutterError("A columnHeader must be a child or another cell")
	}
	return nil
}

func semanticsRadioGroup(node *SemanticsNode) *foundation.FlutterError {
	var err *foundation.FlutterError
	hasCheckedChild := false
	var validateRadioGroupChildren func(node *SemanticsNode) bool
	validateRadioGroupChildren = func(node *SemanticsNode) bool {
		data := node.GetSemanticsData()
		if data.Role == ui.SemanticsRoleRadioGroup {
			// Children under sub radio groups don't belong to this radio group.
			return err == nil
		}
		if !data.FlagsCollection.IsInMutuallyExclusiveGroup {
			node.VisitChildren(validateRadioGroupChildren)
			return err == nil
		}
		if data.FlagsCollection.IsChecked == ui.CheckedStateIsTrue {
			if hasCheckedChild {
				err = foundation.NewFlutterError("Radio groups must not have multiple checked children")
				return false
			}
			hasCheckedChild = true
		}
		dartAssert(err == nil, nil)
		return true
	}
	node.VisitChildren(validateRadioGroupChildren)
	return err
}

func semanticsMenu(node *SemanticsNode) *foundation.FlutterError {
	if node.ChildrenCount() < 1 {
		return foundation.NewFlutterError("a menu cannot be empty")
	}
	return nil
}

func semanticsMenuBar(node *SemanticsNode) *foundation.FlutterError {
	if node.ChildrenCount() < 1 {
		return foundation.NewFlutterError("a menu bar cannot be empty")
	}
	return nil
}

// hasMenuAncestor walks up from node looking for a menu or menu bar.
func hasMenuAncestor(node *SemanticsNode) bool {
	currentNode := node
	for currentNode != nil && currentNode.Parent() != nil {
		if r := currentNode.Parent().Role(); r == ui.SemanticsRoleMenu || r == ui.SemanticsRoleMenuBar {
			return true
		}
		currentNode = currentNode.Parent()
	}
	return false
}

func semanticsMenuItem(node *SemanticsNode) *foundation.FlutterError {
	if hasMenuAncestor(node) {
		return nil
	}
	return foundation.NewFlutterError("A menu item must be a child of a menu or a menu bar")
}

func semanticsMenuItemCheckbox(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	if data.FlagsCollection.IsChecked == ui.CheckedStateNone {
		return foundation.NewFlutterError("a menu item checkbox must be checkable")
	}
	if hasMenuAncestor(node) {
		return nil
	}
	return foundation.NewFlutterError("A menu item checkbox must be a child of a menu or a menu bar")
}

func semanticsMenuItemRadio(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	if data.FlagsCollection.IsChecked == ui.CheckedStateNone {
		return foundation.NewFlutterError("a menu item radio must be checkable")
	}
	if hasMenuAncestor(node) {
		return nil
	}
	return foundation.NewFlutterError("A menu item radio must be a child of a menu or a menu bar")
}

func noLiveRegion(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	if data.FlagsCollection.IsLiveRegion {
		return foundation.NewFlutterError("Node " + strconv.Itoa(node.ID()) + " has role " + data.Role.String() +
			" but is also a live region. A node can not have " + data.Role.String() +
			" and be live region at the same time. Either remove the role or the live region")
	}
	return nil
}

func semanticsListItem(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	parent := node.Parent()
	if parent == nil {
		return foundation.NewFlutterError("Semantics node " + strconv.Itoa(node.ID()) + " has role " +
			data.Role.String() + " but doesn't have a parent")
	}
	parentSemanticsData := parent.GetSemanticsData()
	if parentSemanticsData.Role != ui.SemanticsRoleList {
		return foundation.NewFlutterError("Semantics node " + strconv.Itoa(node.ID()) + " has role " +
			data.Role.String() + ", but its parent node " + strconv.Itoa(parent.ID()) +
			" doesn't have the role " + ui.SemanticsRoleList.String() + ". Please assign the " +
			ui.SemanticsRoleList.String() + " to node " + strconv.Itoa(parent.ID()))
	}
	return nil
}

func isLandmarkRole(nodeData *SemanticsData) bool {
	switch nodeData.Role {
	case ui.SemanticsRoleComplementary, ui.SemanticsRoleContentInfo, ui.SemanticsRoleMain,
		ui.SemanticsRoleNavigation, ui.SemanticsRoleRegion:
		return true
	}
	return false
}

func isSameRoleExisted(semanticsNode *SemanticsNode) bool {
	sameRoleCount := 0
	for _, node := range semanticsNode.Owner().nodes {
		if node.GetSemanticsData().Role == semanticsNode.Role() {
			sameRoleCount++
			if sameRoleCount > 1 {
				return true
			}
		}
	}
	return false
}

func landmarkCheck(node *SemanticsNode, name string, checkAncestors bool) *foundation.FlutterError {
	if checkAncestors {
		for currentNode := node.Parent(); currentNode != nil; currentNode = currentNode.Parent() {
			if isLandmarkRole(currentNode.GetSemanticsData()) {
				return foundation.NewFlutterError("The " + name +
					" landmark role should not contained within any other landmark roles.")
			}
		}
	}
	data := node.GetSemanticsData()
	if isSameRoleExisted(node) && data.Label() == "" {
		return foundation.NewFlutterError("The " + name +
			" landmark role should have a unique label as it is used more than once.")
	}
	return nil
}

func semanticsComplementary(node *SemanticsNode) *foundation.FlutterError {
	return landmarkCheck(node, "complementary", true)
}

func semanticsContentInfo(node *SemanticsNode) *foundation.FlutterError {
	return landmarkCheck(node, "contentInfo", true)
}

func semanticsMain(node *SemanticsNode) *foundation.FlutterError {
	return landmarkCheck(node, "main", true)
}

func semanticsNavigation(node *SemanticsNode) *foundation.FlutterError {
	return landmarkCheck(node, "navigation", false)
}

func semanticsRegion(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	if data.Label() == "" {
		return foundation.NewFlutterError("A region role should include a label that describes the purpose of the content.")
	}
	return nil
}

func semanticsGeneral(node *SemanticsNode) *foundation.FlutterError {
	data := node.GetSemanticsData()
	if isExpanded := data.FlagsCollection.IsExpanded.ToBoolOrNull(); isExpanded != nil {
		hasExpandAction := data.HasAction(ui.SemanticsActionExpand)
		hasCollapseAction := data.HasAction(ui.SemanticsActionCollapse)
		if hasExpandAction && hasCollapseAction {
			return foundation.NewFlutterError("An expandable node cannot have both expand and collapse actions set at the same time.")
		}
		if *isExpanded && hasExpandAction {
			return foundation.NewFlutterError("An expanded node cannot have an expand action.")
		}
		if !*isExpanded && hasCollapseAction {
			return foundation.NewFlutterError("A collapsed node cannot have a collapse action.")
		}
	}
	if data.FlagsCollection.IsAccessibilityFocusBlocked && data.FlagsCollection.IsFocused != ui.TristateNone {
		return foundation.NewFlutterError("A node that is keyboard focusable cannot be set to accessibility unfocusable")
	}
	return nil
}

// ---- SemanticsTag

// SemanticsTag is a tag for a SemanticsNode. Tags are compared by identity
// (Dart's SemanticsTag has no ==; equal const tags are one object), so use
// the same *SemanticsTag.
type SemanticsTag struct {
	// Name is a human-readable name for this tag, for debugging.
	Name string
}

// NewSemanticsTag is SemanticsTag(name).
func NewSemanticsTag(name string) *SemanticsTag { return &SemanticsTag{Name: name} }

func (t *SemanticsTag) String() string {
	return foundation.ObjectRuntimeType(t, "SemanticsTag") + "(" + t.Name + ")"
}

// ---- ChildSemanticsConfigurationsResult

// ChildSemanticsConfigurationsResult is the result of a
// ChildSemanticsConfigurationsDelegate: the configurations to merge into
// the parent, and the groups of sibling configurations to merge together.
type ChildSemanticsConfigurationsResult struct {
	// MergeUp is the configurations to merge into the parent node.
	MergeUp []*SemanticsConfiguration
	// SiblingMergeGroups is the groups of configurations that merge into
	// one node each.
	SiblingMergeGroups [][]*SemanticsConfiguration
}

// ChildSemanticsConfigurationsResultBuilder builds a
// ChildSemanticsConfigurationsResult.
type ChildSemanticsConfigurationsResultBuilder struct {
	mergeUp            []*SemanticsConfiguration
	siblingMergeGroups [][]*SemanticsConfiguration
}

// NewChildSemanticsConfigurationsResultBuilder is
// ChildSemanticsConfigurationsResultBuilder().
func NewChildSemanticsConfigurationsResultBuilder() *ChildSemanticsConfigurationsResultBuilder {
	return &ChildSemanticsConfigurationsResultBuilder{
		mergeUp: []*SemanticsConfiguration{}, siblingMergeGroups: [][]*SemanticsConfiguration{},
	}
}

// MarkAsMergeUp marks config to be merged into the parent node.
func (b *ChildSemanticsConfigurationsResultBuilder) MarkAsMergeUp(config *SemanticsConfiguration) {
	b.mergeUp = append(b.mergeUp, config)
}

// MarkAsSiblingMergeGroup marks configs to be merged into one node.
func (b *ChildSemanticsConfigurationsResultBuilder) MarkAsSiblingMergeGroup(configs []*SemanticsConfiguration) {
	b.siblingMergeGroups = append(b.siblingMergeGroups, configs)
}

// Build builds the result (asserting, in debug builds, that no
// configuration was marked twice).
func (b *ChildSemanticsConfigurationsResultBuilder) Build() *ChildSemanticsConfigurationsResult {
	if foundation.KDebugMode {
		seenConfigs := map[*SemanticsConfiguration]bool{}
		all := append([]*SemanticsConfiguration(nil), b.mergeUp...)
		for _, group := range b.siblingMergeGroups {
			all = append(all, group...)
		}
		for _, config := range all {
			dartAssert(!seenConfigs[config], "Duplicated SemanticsConfigurations. This can happen if the same "+
				"SemanticsConfiguration was marked twice in markAsMergeUp and/or markAsSiblingMergeGroup")
			seenConfigs[config] = true
		}
	}
	return &ChildSemanticsConfigurationsResult{MergeUp: b.mergeUp, SiblingMergeGroups: b.siblingMergeGroups}
}

// ---- CustomSemanticsAction

// CustomSemanticsAction is an identifier of a custom semantics action: a
// label, or a hint overriding a standard action. It is a comparable value
// (Dart's ==): an empty Label or Hint and the zero Action stand for Dart's
// null (Dart asserts the strings are not empty).
type CustomSemanticsAction struct {
	// Label is the user-readable name of this custom semantics action.
	Label string
	// Hint is the hint description of this custom semantics action.
	Hint string
	// Action is the standard semantics action this action replaces.
	Action ui.SemanticsAction
}

// NewCustomSemanticsAction is CustomSemanticsAction(label:).
func NewCustomSemanticsAction(label string) CustomSemanticsAction {
	dartAssert(label != "", nil)
	return CustomSemanticsAction{Label: label}
}

// NewCustomSemanticsActionOverridingAction is
// CustomSemanticsAction.overridingAction(hint:, action:).
func NewCustomSemanticsActionOverridingAction(hint string, action ui.SemanticsAction) CustomSemanticsAction {
	dartAssert(hint != "", nil)
	return CustomSemanticsAction{Hint: hint, Action: action}
}

func nullableString(s string) string {
	if s == "" {
		return "null"
	}
	return s
}

func (a CustomSemanticsAction) String() string {
	id := "null"
	if i, ok := customSemanticsActionIDs[a]; ok {
		id = strconv.Itoa(i)
	}
	action := "null"
	if a.Action != (ui.SemanticsAction{}) {
		action = a.Action.String()
	}
	return "CustomSemanticsAction(" + id + ", label:" + nullableString(a.Label) + ", hint:" +
		nullableString(a.Hint) + ", action:" + action + ")"
}

var (
	customSemanticsActionNextID  = 0
	customSemanticsActionActions = map[int]CustomSemanticsAction{}
	customSemanticsActionIDs     = map[CustomSemanticsAction]int{}
)

// CustomSemanticsActionGetIdentifier is CustomSemanticsAction.getIdentifier:
// the unique identifier of action, assigned on first use.
func CustomSemanticsActionGetIdentifier(action CustomSemanticsAction) int {
	result, ok := customSemanticsActionIDs[action]
	if !ok {
		result = customSemanticsActionNextID
		customSemanticsActionNextID++
		customSemanticsActionIDs[action] = result
		customSemanticsActionActions[result] = action
	}
	return result
}

// CustomSemanticsActionGetAction is CustomSemanticsAction.getAction: the
// action with identifier id, or nil.
func CustomSemanticsActionGetAction(id int) *CustomSemanticsAction {
	if a, ok := customSemanticsActionActions[id]; ok {
		return &a
	}
	return nil
}

// CustomSemanticsActionResetForTests is CustomSemanticsAction.resetForTests
// (debug builds only).
func CustomSemanticsActionResetForTests() {
	if foundation.KDebugMode {
		clear(customSemanticsActionActions)
		clear(customSemanticsActionIDs)
		customSemanticsActionNextID = 0
	}
}

// customActionEntries is Dart's Map<CustomSemanticsAction, VoidCallback>
// with its insertion order.
type customActionEntries = []dart.MapEntry[CustomSemanticsAction, func()]

func customActionIndex(m customActionEntries, key CustomSemanticsAction) int {
	for i, e := range m {
		if e.Key == key {
			return i
		}
	}
	return -1
}

// putCustomAction is map[key] = value (an existing key keeps its place).
func putCustomAction(m customActionEntries, key CustomSemanticsAction, value func()) customActionEntries {
	if i := customActionIndex(m, key); i >= 0 {
		m[i].Value = value
		return m
	}
	return append(m, dart.MapEntry[CustomSemanticsAction, func()]{Key: key, Value: value})
}

// customActionsEqual is mapEquals on two custom action maps (callbacks
// compared as Dart compares functions).
func customActionsEqual(a, b customActionEntries) bool {
	if len(a) != len(b) {
		return false
	}
	for _, e := range a {
		i := customActionIndex(b, e.Key)
		if i < 0 || !dart.FunctionsEqual(b[i].Value, e.Value) {
			return false
		}
	}
	return true
}

// ---- AttributedString

// AttributedString is a string that carries a list of StringAttributes.
// Dart's string field is Text (String is toString). Compare with Equal:
// the attributes are compared by identity, as Dart's listEquals does.
type AttributedString struct {
	// Text is the plain string stored in the attributed string.
	Text string
	// Attributes is the attributes this string carries (their ranges are
	// UTF-16 offsets into Text).
	Attributes []ui.StringAttribute
}

// AttributedStringOptions are AttributedString's named parameters.
type AttributedStringOptions struct{ Attributes []ui.StringAttribute }

func len16(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// NewAttributedString is AttributedString(string, {attributes}).
func NewAttributedString(s string, opts ...AttributedStringOptions) AttributedString {
	var o AttributedStringOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(s != "" || len(o.Attributes) == 0, nil)
	if foundation.KDebugMode {
		for _, attribute := range o.Attributes {
			dartAssert(len16(s) >= attribute.Range().Start && len16(s) >= attribute.Range().End,
				"The range in "+attribute.String()+" is outside of the string "+s)
		}
	}
	attributes := o.Attributes
	if attributes == nil {
		attributes = []ui.StringAttribute{}
	}
	return AttributedString{Text: s, Attributes: attributes}
}

// Add is operator +: the concatenation, with other's attribute ranges
// shifted by this string's length.
func (a AttributedString) Add(other AttributedString) AttributedString {
	if a.Text == "" {
		return other
	}
	if other.Text == "" {
		return a
	}
	// None of the strings is empty.
	newString := a.Text + other.Text
	newAttributes := append([]ui.StringAttribute{}, a.Attributes...)
	if len(other.Attributes) > 0 {
		offset := len16(a.Text)
		for _, attribute := range other.Attributes {
			newRange := ui.NewTextRange(attribute.Range().Start+offset, attribute.Range().End+offset)
			newAttributes = append(newAttributes, attribute.Copy(newRange))
		}
	}
	return NewAttributedString(newString, AttributedStringOptions{Attributes: newAttributes})
}

// Equal is AttributedString's ==.
func (a AttributedString) Equal(other AttributedString) bool {
	if a.Text != other.Text || len(a.Attributes) != len(other.Attributes) {
		return false
	}
	for i := range a.Attributes {
		if a.Attributes[i] != other.Attributes[i] {
			return false
		}
	}
	return true
}

func attributesString(attributes []ui.StringAttribute) string {
	parts := make([]string, len(attributes))
	for i, a := range attributes {
		parts[i] = a.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (a AttributedString) String() string {
	return foundation.ObjectRuntimeType(a, "AttributedString") + "('" + a.Text + "', attributes: " +
		attributesString(a.Attributes) + ")"
}

// AttributedStringProperty is a DiagnosticsProperty for AttributedStrings,
// which shows a string when there are no attributes, and more details
// otherwise.
type AttributedStringProperty struct {
	foundation.DiagnosticsProperty[*AttributedString]
	// ShowWhenEmpty: whether to show the property when the value is an
	// empty string.
	ShowWhenEmpty bool
}

// AttributedStringPropertyOptions are AttributedStringProperty's named
// parameters.
type AttributedStringPropertyOptions struct {
	ShowName      *bool
	ShowWhenEmpty bool
	DefaultValue  any
	Level         *foundation.DiagnosticLevel
	Description   *string
}

// NewAttributedStringProperty is AttributedStringProperty(name, value,
// {...}) (a nil value is null).
func NewAttributedStringProperty(name string, value *AttributedString, opts ...AttributedStringPropertyOptions) *AttributedStringProperty {
	var o AttributedStringPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &AttributedStringProperty{
		DiagnosticsProperty: *foundation.NewDiagnosticsProperty(name, value, foundation.DiagnosticsPropertyOptions{
			ShowName: o.ShowName, DefaultValue: o.DefaultValue, Level: o.Level, Description: o.Description,
		}),
		ShowWhenEmpty: o.ShowWhenEmpty,
	}
	p.SetSelf(p)
	return p
}

// IsInteresting also hides an empty string unless ShowWhenEmpty.
func (p *AttributedStringProperty) IsInteresting() bool {
	value := p.TypedValue()
	return p.DiagnosticsProperty.IsInteresting() && (p.ShowWhenEmpty || (value != nil && value.Text != ""))
}

// ValueToString is the quoted string, then the attributes.
func (p *AttributedStringProperty) ValueToString(opts ...foundation.ValueToStringOptions) string {
	value := p.TypedValue()
	if value == nil {
		return "null"
	}
	text := value.Text
	if len(opts) > 0 && opts[0].ParentConfiguration != nil && !opts[0].ParentConfiguration.LineBreakProperties {
		// This follows a similar pattern to StringProperty.
		text = strings.ReplaceAll(text, "\n", `\n`)
	}
	if len(value.Attributes) == 0 {
		return `"` + text + `"`
	}
	return `"` + text + `" ` + attributesString(value.Attributes)
}

// ---- SemanticsLabelBuilder

type labelPart struct {
	text          string
	textDirection *ui.TextDirection
}

// SemanticsLabelBuilder builds a semantics label from parts, separated by
// Separator, wrapping parts whose text direction differs from the
// builder's in Unicode directional embeddings.
type SemanticsLabelBuilder struct {
	// Separator is the string between parts (default ' ').
	Separator string
	// TextDirection is the overall text direction, or nil.
	TextDirection *ui.TextDirection
	parts         []labelPart
}

// SemanticsLabelBuilderOptions are SemanticsLabelBuilder's named
// parameters (Separator nil: ' ').
type SemanticsLabelBuilderOptions struct {
	Separator     *string
	TextDirection *ui.TextDirection
}

// NewSemanticsLabelBuilder is SemanticsLabelBuilder({separator,
// textDirection}).
func NewSemanticsLabelBuilder(opts ...SemanticsLabelBuilderOptions) *SemanticsLabelBuilder {
	var o SemanticsLabelBuilderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	separator := " "
	if o.Separator != nil {
		separator = *o.Separator
	}
	return &SemanticsLabelBuilder{Separator: separator, TextDirection: o.TextDirection}
}

// SemanticsLabelBuilderAddPartOptions are addPart's named parameters.
type SemanticsLabelBuilderAddPartOptions struct{ TextDirection *ui.TextDirection }

// AddPart adds a non-empty label part.
func (b *SemanticsLabelBuilder) AddPart(label string, opts ...SemanticsLabelBuilderAddPartOptions) {
	var o SemanticsLabelBuilderAddPartOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if label != "" {
		b.parts = append(b.parts, labelPart{label, o.TextDirection})
	}
}

// IsEmpty reports whether no parts were added.
func (b *SemanticsLabelBuilder) IsEmpty() bool { return len(b.parts) == 0 }

// Length is the number of parts.
func (b *SemanticsLabelBuilder) Length() int { return len(b.parts) }

// Build concatenates the parts.
func (b *SemanticsLabelBuilder) Build() string {
	if len(b.parts) == 0 {
		return ""
	}
	if len(b.parts) == 1 {
		return b.parts[0].text
	}
	// Concatenate multiple parts with proper text direction handling
	var buffer strings.Builder
	buffer.WriteString(b.parts[0].text)
	for _, part := range b.parts[1:] {
		partDirection := part.textDirection
		if partDirection == nil {
			partDirection = b.TextDirection
		}
		if b.Separator != "" {
			buffer.WriteString(b.Separator)
		}
		processedText := part.text
		if b.TextDirection != nil && partDirection != nil && *b.TextDirection != *partDirection {
			directionalEmbedding := foundation.Unicode.LRE
			if *partDirection == ui.TextDirectionRtl {
				directionalEmbedding = foundation.Unicode.RLE
			}
			processedText = directionalEmbedding + part.text + foundation.Unicode.PDF
		}
		buffer.WriteString(processedText)
	}
	return buffer.String()
}

// Clear removes every part.
func (b *SemanticsLabelBuilder) Clear() { b.parts = nil }

// ---- SemanticsData

// SemanticsDataOptions are SemanticsData's fields (all required in Dart
// except Tags, Transform and CustomSemanticsActionIDs; nil pointers and
// slices are Dart's null).
type SemanticsDataOptions struct {
	FlagsCollection           ui.SemanticsFlags
	Actions                   int
	Identifier                string
	TraversalParentIdentifier any
	TraversalChildIdentifier  any
	AttributedLabel           AttributedString
	AttributedValue           AttributedString
	AttributedIncreasedValue  AttributedString
	AttributedDecreasedValue  AttributedString
	AttributedHint            AttributedString
	Tooltip                   string
	TextDirection             *ui.TextDirection
	Rect                      ui.Rect
	TextSelection             *services.TextSelection
	ScrollIndex               *int
	ScrollChildCount          *int
	ScrollPosition            *float64
	ScrollExtentMax           *float64
	ScrollExtentMin           *float64
	PlatformViewID            *int
	MaxValueLength            *int
	CurrentValueLength        *int
	HeadingLevel              int
	LinkURL                   *url.URL
	Role                      ui.SemanticsRole
	ControlsNodes             []string
	ValidationResult          ui.SemanticsValidationResult
	HitTestBehavior           ui.SemanticsHitTestBehavior
	InputType                 ui.SemanticsInputType
	Locale                    *ui.Locale
	MinValue                  *string
	MaxValue                  *string
	Tags                      []*SemanticsTag
	Transform                 *vm.Matrix4
	CustomSemanticsActionIDs  []int
}

// SemanticsData is a summary of the semantic information of a
// SemanticsNode (its fields are SemanticsDataOptions'). Compare with Equal.
type SemanticsData struct {
	foundation.DiagnosticableMixin
	SemanticsDataOptions
}

// NewSemanticsData is SemanticsData({...}), with its assertions.
func NewSemanticsData(o SemanticsDataOptions) *SemanticsData {
	dartAssert(o.Tooltip == "" || o.TextDirection != nil,
		`A SemanticsData object with tooltip "`+o.Tooltip+`" had a null textDirection.`)
	dartAssert(o.AttributedLabel.Text == "" || o.TextDirection != nil,
		`A SemanticsData object with label "`+o.AttributedLabel.Text+`" had a null textDirection.`)
	dartAssert(o.AttributedValue.Text == "" || o.TextDirection != nil,
		`A SemanticsData object with value "`+o.AttributedValue.Text+`" had a null textDirection.`)
	dartAssert(o.AttributedDecreasedValue.Text == "" || o.TextDirection != nil,
		`A SemanticsData object with decreasedValue "`+o.AttributedDecreasedValue.Text+`" had a null textDirection.`)
	dartAssert(o.AttributedIncreasedValue.Text == "" || o.TextDirection != nil,
		`A SemanticsData object with increasedValue "`+o.AttributedIncreasedValue.Text+`" had a null textDirection.`)
	dartAssert(o.AttributedHint.Text == "" || o.TextDirection != nil,
		`A SemanticsData object with hint "`+o.AttributedHint.Text+`" had a null textDirection.`)
	dartAssert(o.HeadingLevel >= 0 && o.HeadingLevel <= 6, "Heading level must be between 0 and 6")
	dartAssert(o.LinkURL == nil || o.FlagsCollection.IsLink,
		"A SemanticsData object with a linkUrl must have the isLink flag set to true")
	d := &SemanticsData{SemanticsDataOptions: o}
	d.SetSelf(d)
	return d
}

// Flags is the bit field of the flags (Deprecated: use FlagsCollection).
func (d *SemanticsData) Flags() int { return toBitMask(d.FlagsCollection) }

// Label is the label's text.
func (d *SemanticsData) Label() string { return d.AttributedLabel.Text }

// Value is the value's text.
func (d *SemanticsData) Value() string { return d.AttributedValue.Text }

// IncreasedValue is the increased value's text.
func (d *SemanticsData) IncreasedValue() string { return d.AttributedIncreasedValue.Text }

// DecreasedValue is the decreased value's text.
func (d *SemanticsData) DecreasedValue() string { return d.AttributedDecreasedValue.Text }

// Hint is the hint's text.
func (d *SemanticsData) Hint() string { return d.AttributedHint.Text }

// HasFlag reports whether flag is set (Deprecated: use FlagsCollection).
func (d *SemanticsData) HasFlag(flag ui.SemanticsFlag) bool { return d.Flags()&flag.Index() != 0 }

// HasAction reports whether action is set.
func (d *SemanticsData) HasAction(action ui.SemanticsAction) bool {
	return d.Actions&action.Index() != 0
}

// ToJson is a JSON representation of the data (an ordered map).
func (d *SemanticsData) ToJson() []dart.MapEntry[string, any] {
	flagsList := []string{}
	for _, flag := range ui.SemanticsFlagValues() {
		if d.HasFlag(flag) {
			flagsList = append(flagsList, flag.Name())
		}
	}
	actionsList := []string{}
	for _, action := range ui.SemanticsActionValues() {
		if d.HasAction(action) {
			actionsList = append(actionsList, action.Name())
		}
	}
	m := []dart.MapEntry[string, any]{
		{Key: "label", Value: d.Label()},
		{Key: "value", Value: d.Value()},
		{Key: "hint", Value: d.Hint()},
		{Key: "tooltip", Value: d.Tooltip},
		{Key: "increasedValue", Value: d.IncreasedValue()},
		{Key: "decreasedValue", Value: d.DecreasedValue()},
		{Key: "flags", Value: flagsList},
		{Key: "actions", Value: actionsList},
		{Key: "rect", Value: []dart.MapEntry[string, any]{
			{Key: "left", Value: d.Rect.Left},
			{Key: "top", Value: d.Rect.Top},
			{Key: "width", Value: d.Rect.Width()},
			{Key: "height", Value: d.Rect.Height()},
		}},
	}
	if d.Transform != nil {
		m = append(m, dart.MapEntry[string, any]{Key: "transform", Value: append([]float64(nil), d.Transform.Storage()...)})
	}
	return m
}

func (d *SemanticsData) ToStringShort() string {
	return foundation.ObjectRuntimeType(d, "SemanticsData")
}

func (d *SemanticsData) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	f := false
	properties.Add(foundation.NewDiagnosticsProperty("rect", d.Rect, foundation.DiagnosticsPropertyOptions{ShowName: &f}))
	properties.Add(painting.NewTransformProperty("transform", d.Transform,
		painting.TransformPropertyOptions{ShowName: &f, DefaultValue: foundation.Null}))
	actionSummary := []string{}
	for _, action := range ui.SemanticsActionValues() {
		if d.Actions&action.Index() != 0 {
			actionSummary = append(actionSummary, action.Name())
		}
	}
	dartAssert(d.CustomSemanticsActionIDs != nil, "Null check operator used on a null value")
	customSemanticsActionSummary := make([]*string, len(d.CustomSemanticsActionIDs))
	for i, actionID := range d.CustomSemanticsActionIDs {
		customSemanticsActionSummary[i] = labelOf(*CustomSemanticsActionGetAction(actionID))
	}
	properties.Add(foundation.NewIterableProperty("actions", actionSummary,
		foundation.IterablePropertyOptions{IfEmpty: foundation.NullString}))
	properties.Add(foundation.NewIterableProperty("customActions", customSemanticsActionSummary,
		foundation.IterablePropertyOptions{IfEmpty: foundation.NullString}))
	flagSummary := d.FlagsCollection.ToStrings()
	properties.Add(foundation.NewIterableProperty("flags", flagSummary, foundation.IterablePropertyOptions{IfEmpty: foundation.NullString}))
	properties.Add(foundation.NewStringProperty("identifier", d.Identifier, foundation.StringPropertyOptions{DefaultValue: ""}))
	properties.Add(foundation.NewDiagnosticsProperty("traversalParentIdentifier", d.TraversalParentIdentifier,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewDiagnosticsProperty("traversalChildIdentifier", d.TraversalChildIdentifier,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(NewAttributedStringProperty("label", &d.AttributedLabel))
	properties.Add(NewAttributedStringProperty("value", &d.AttributedValue))
	properties.Add(NewAttributedStringProperty("increasedValue", &d.AttributedIncreasedValue))
	properties.Add(NewAttributedStringProperty("decreasedValue", &d.AttributedDecreasedValue))
	properties.Add(NewAttributedStringProperty("hint", &d.AttributedHint))
	properties.Add(foundation.NewStringProperty("tooltip", d.Tooltip, foundation.StringPropertyOptions{DefaultValue: ""}))
	properties.Add(foundation.NewEnumProperty("textDirection", d.TextDirection,
		foundation.EnumPropertyOptions{DefaultValue: foundation.Null}))
	if d.TextSelection != nil && d.TextSelection.IsValid() {
		properties.Add(foundation.NewMessageProperty("textSelection", "["+strconv.Itoa(d.TextSelection.Start)+", "+
			strconv.Itoa(d.TextSelection.End)+"]"))
	}
	addIntProperty(properties, "platformViewId", d.PlatformViewID)
	addIntProperty(properties, "maxValueLength", d.MaxValueLength)
	addIntProperty(properties, "currentValueLength", d.CurrentValueLength)
	addIntProperty(properties, "scrollChildren", d.ScrollChildCount)
	addIntProperty(properties, "scrollIndex", d.ScrollIndex)
	addDoubleProperty(properties, "scrollExtentMin", d.ScrollExtentMin)
	addDoubleProperty(properties, "scrollPosition", d.ScrollPosition)
	addDoubleProperty(properties, "scrollExtentMax", d.ScrollExtentMax)
	properties.Add(foundation.NewIntProperty("headingLevel", d.HeadingLevel, foundation.IntPropertyOptions{DefaultValue: 0}))
	properties.Add(foundation.NewDiagnosticsProperty("linkUrl", d.LinkURL,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	if d.ControlsNodes != nil {
		properties.Add(foundation.NewIterableProperty("controls", d.ControlsNodes,
			foundation.IterablePropertyOptions{IfEmpty: foundation.NullString}))
	}
	if d.Role != ui.SemanticsRoleNone {
		properties.Add(foundation.NewEnumProperty("role", d.Role,
			foundation.EnumPropertyOptions{DefaultValue: ui.SemanticsRoleNone}))
	}
	if d.ValidationResult != ui.SemanticsValidationResultNone {
		properties.Add(foundation.NewEnumProperty("validationResult", d.ValidationResult,
			foundation.EnumPropertyOptions{DefaultValue: ui.SemanticsValidationResultNone}))
	}
	properties.Add(foundation.NewStringProperty("minValue", d.MinValue, foundation.StringPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewStringProperty("maxValue", d.MaxValue, foundation.StringPropertyOptions{DefaultValue: foundation.Null}))
}

func labelOf(a CustomSemanticsAction) *string {
	if a.Label == "" {
		return nil
	}
	l := a.Label
	return &l
}

func addIntProperty(properties *foundation.DiagnosticPropertiesBuilder, name string, v *int) {
	properties.Add(foundation.NewIntProperty(name, v, foundation.IntPropertyOptions{DefaultValue: foundation.Null}))
}

func addDoubleProperty(properties *foundation.DiagnosticPropertiesBuilder, name string, v *float64) {
	properties.Add(foundation.NewDoubleProperty(name, v, foundation.DoublePropertyOptions{DefaultValue: foundation.Null}))
}

// Equal is SemanticsData's ==.
func (d *SemanticsData) Equal(other *SemanticsData) bool {
	if other == nil {
		return false
	}
	return other.Flags() == d.Flags() &&
		other.Actions == d.Actions &&
		other.Identifier == d.Identifier &&
		objectsEqual(other.TraversalParentIdentifier, d.TraversalParentIdentifier) &&
		objectsEqual(other.TraversalChildIdentifier, d.TraversalChildIdentifier) &&
		other.AttributedLabel.Equal(d.AttributedLabel) &&
		other.AttributedValue.Equal(d.AttributedValue) &&
		other.AttributedIncreasedValue.Equal(d.AttributedIncreasedValue) &&
		other.AttributedDecreasedValue.Equal(d.AttributedDecreasedValue) &&
		other.AttributedHint.Equal(d.AttributedHint) &&
		other.Tooltip == d.Tooltip &&
		ptrEqual(other.TextDirection, d.TextDirection) &&
		other.Rect == d.Rect &&
		tagSetsEqual(other.Tags, d.Tags) &&
		ptrEqual(other.ScrollChildCount, d.ScrollChildCount) &&
		ptrEqual(other.ScrollIndex, d.ScrollIndex) &&
		textSelectionsEqual(other.TextSelection, d.TextSelection) &&
		ptrEqual(other.ScrollPosition, d.ScrollPosition) &&
		ptrEqual(other.ScrollExtentMax, d.ScrollExtentMax) &&
		ptrEqual(other.ScrollExtentMin, d.ScrollExtentMin) &&
		ptrEqual(other.PlatformViewID, d.PlatformViewID) &&
		ptrEqual(other.MaxValueLength, d.MaxValueLength) &&
		ptrEqual(other.CurrentValueLength, d.CurrentValueLength) &&
		other.Transform.Equal(d.Transform) &&
		other.HeadingLevel == d.HeadingLevel &&
		urlsEqual(other.LinkURL, d.LinkURL) &&
		other.Role == d.Role &&
		other.ValidationResult == d.ValidationResult &&
		other.InputType == d.InputType &&
		other.HitTestBehavior == d.HitTestBehavior &&
		sortedListsEqual(other.CustomSemanticsActionIDs, d.CustomSemanticsActionIDs) &&
		stringSetsEqual(d.ControlsNodes, other.ControlsNodes) &&
		ptrEqual(other.MinValue, d.MinValue) &&
		ptrEqual(other.MaxValue, d.MaxValue)
}

func sortedListsEqual(left, right []int) bool {
	if left == nil && right == nil {
		return true
	}
	if left != nil && right != nil {
		if len(left) != len(right) {
			return false
		}
		for i := range left {
			if left[i] != right[i] {
				return false
			}
		}
		return true
	}
	return false
}

// ptrEqual is Dart's == on nullable values (NaN is unequal to itself).
func ptrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// objectsEqual is Dart's == on Object? values (Go's == on comparable
// dynamic values; values Go can't compare are equal only when identical,
// which Go can't tell, so never).
func objectsEqual(a, b any) (eq bool) {
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return a == b
}

func textSelectionsEqual(a, b *services.TextSelection) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func urlsEqual(a, b *url.URL) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.String() == b.String()
}

// tagSetsEqual is setEquals on two tag sets.
func tagSetsEqual(a, b []*SemanticsTag) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if len(a) != len(b) {
		return false
	}
	for _, t := range a {
		if !containsTag(b, t) {
			return false
		}
	}
	return true
}

func containsTag(tags []*SemanticsTag, tag *SemanticsTag) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// stringSetsEqual is setEquals<String>.
func stringSetsEqual(a, b []string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if len(a) != len(b) {
		return false
	}
	set := map[string]bool{}
	for _, s := range b {
		set[s] = true
	}
	for _, s := range a {
		if !set[s] {
			return false
		}
	}
	return true
}

// unionStrings is {...a, ...b} (insertion order, no duplicates).
func unionStrings(a, b []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range append(append([]string(nil), a...), b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ---- _SemanticsDiagnosticableNode

type semanticsDiagnosticableNode struct {
	foundation.DiagnosticableNode[*SemanticsNode]
	childOrder DebugSemanticsDumpOrder
}

func newSemanticsDiagnosticableNode(name *string, value *SemanticsNode, style *foundation.DiagnosticsTreeStyle, childOrder DebugSemanticsDumpOrder) *semanticsDiagnosticableNode {
	n := &semanticsDiagnosticableNode{
		DiagnosticableNode: *foundation.NewDiagnosticableNode(value, foundation.DiagnosticableNodeOptions{Name: name, Style: style}),
		childOrder:         childOrder,
	}
	n.SetSelf(n)
	return n
}

func (n *semanticsDiagnosticableNode) GetChildren() []foundation.DiagnosticsNode {
	return n.TypedValue().DebugDescribeChildrenInOrder(n.childOrder)
}

// ---- SemanticsHintOverrides

// SemanticsHintOverridesOptions are SemanticsHintOverrides' fields (nil:
// null; Dart asserts they are not empty).
type SemanticsHintOverridesOptions struct {
	OnTapHint       *string
	OnLongPressHint *string
}

// SemanticsHintOverrides provides hint values which override the default
// hints on supported platforms. Compare with Equal.
type SemanticsHintOverrides struct {
	foundation.DiagnosticableTreeBase
	SemanticsHintOverridesOptions
}

// NewSemanticsHintOverrides is SemanticsHintOverrides({onTapHint,
// onLongPressHint}).
func NewSemanticsHintOverrides(o SemanticsHintOverridesOptions) *SemanticsHintOverrides {
	dartAssert(o.OnTapHint == nil || *o.OnTapHint != "", nil)
	dartAssert(o.OnLongPressHint == nil || *o.OnLongPressHint != "", nil)
	h := &SemanticsHintOverrides{SemanticsHintOverridesOptions: o}
	h.SetSelf(h)
	return h
}

// IsNotEmpty reports whether there are any non-null hint values.
func (h *SemanticsHintOverrides) IsNotEmpty() bool {
	return h.OnTapHint != nil || h.OnLongPressHint != nil
}

// Equal is SemanticsHintOverrides' ==.
func (h *SemanticsHintOverrides) Equal(other *SemanticsHintOverrides) bool {
	if h == nil || other == nil {
		return h == other
	}
	return ptrEqual(other.OnTapHint, h.OnTapHint) && ptrEqual(other.OnLongPressHint, h.OnLongPressHint)
}

func (h *SemanticsHintOverrides) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	h.DiagnosticableTreeBase.DebugFillProperties(properties)
	properties.Add(foundation.NewStringProperty("onTapHint", h.OnTapHint, foundation.StringPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewStringProperty("onLongPressHint", h.OnLongPressHint, foundation.StringPropertyOptions{DefaultValue: foundation.Null}))
}

// ---- SemanticsProperties

// SemanticsPropertiesOptions are SemanticsProperties' fields (nil: null;
// ValidationResult defaults to none).
type SemanticsPropertiesOptions struct {
	Enabled                         *bool
	Checked                         *bool
	Mixed                           *bool
	Expanded                        *bool
	Selected                        *bool
	Toggled                         *bool
	Button                          *bool
	Link                            *bool
	LinkURL                         *url.URL
	Header                          *bool
	HeadingLevel                    *int
	TextField                       *bool
	Slider                          *bool
	KeyboardKey                     *bool
	ReadOnly                        *bool
	Focusable                       *bool // Deprecated: use Focused.
	Focused                         *bool
	AccessibilityFocusBlockType     *AccessibilityFocusBlockType
	InMutuallyExclusiveGroup        *bool
	Hidden                          *bool
	Obscured                        *bool
	Multiline                       *bool
	ScopesRoute                     *bool
	NamesRoute                      *bool
	Image                           *bool
	LiveRegion                      *bool
	IsRequired                      *bool
	MaxValueLength                  *int
	CurrentValueLength              *int
	Identifier                      *string
	TraversalParentIdentifier       any
	TraversalChildIdentifier        any
	Label                           *string
	AttributedLabel                 *AttributedString
	Value                           *string
	AttributedValue                 *AttributedString
	IncreasedValue                  *string
	AttributedIncreasedValue        *AttributedString
	DecreasedValue                  *string
	AttributedDecreasedValue        *AttributedString
	Hint                            *string
	Tooltip                         *string
	AttributedHint                  *AttributedString
	HintOverrides                   *SemanticsHintOverrides
	TextDirection                   *ui.TextDirection
	SortKey                         SemanticsSortKey
	TagForChildren                  *SemanticsTag
	Role                            *ui.SemanticsRole
	ControlsNodes                   []string
	InputType                       *ui.SemanticsInputType
	ValidationResult                ui.SemanticsValidationResult
	HitTestBehavior                 *ui.SemanticsHitTestBehavior
	OnTap                           func()
	OnLongPress                     func()
	OnScrollLeft                    func()
	OnScrollRight                   func()
	OnScrollUp                      func()
	OnScrollDown                    func()
	OnIncrease                      func()
	OnDecrease                      func()
	OnCopy                          func()
	OnCut                           func()
	OnPaste                         func()
	OnMoveCursorForwardByCharacter  MoveCursorHandler
	OnMoveCursorBackwardByCharacter MoveCursorHandler
	OnMoveCursorForwardByWord       MoveCursorHandler
	OnMoveCursorBackwardByWord      MoveCursorHandler
	OnSetSelection                  SetSelectionHandler
	OnSetText                       SetTextHandler
	OnDidGainAccessibilityFocus     func()
	OnDidLoseAccessibilityFocus     func()
	OnFocus                         func()
	OnDismiss                       func()
	OnExpand                        func()
	OnCollapse                      func()
	CustomSemanticsActions          []dart.MapEntry[CustomSemanticsAction, func()]
	MinValue                        *string
	MaxValue                        *string
}

// SemanticsProperties contains properties used by assistive technologies
// to make the application more accessible (its fields are
// SemanticsPropertiesOptions').
type SemanticsProperties struct {
	foundation.DiagnosticableTreeBase
	SemanticsPropertiesOptions
}

// NewSemanticsProperties is SemanticsProperties({...}), with its
// assertions.
func NewSemanticsProperties(o SemanticsPropertiesOptions) *SemanticsProperties {
	dartAssert(o.Label == nil || o.AttributedLabel == nil, "Only one of label or attributedLabel should be provided")
	dartAssert(o.Value == nil || o.AttributedValue == nil, "Only one of value or attributedValue should be provided")
	dartAssert(o.IncreasedValue == nil || o.AttributedIncreasedValue == nil,
		"Only one of increasedValue or attributedIncreasedValue should be provided")
	dartAssert(o.DecreasedValue == nil || o.AttributedDecreasedValue == nil,
		"Only one of decreasedValue or attributedDecreasedValue should be provided")
	dartAssert(o.Hint == nil || o.AttributedHint == nil, "Only one of hint or attributedHint should be provided")
	dartAssert(o.HeadingLevel == nil || (*o.HeadingLevel > 0 && *o.HeadingLevel <= 6),
		"Heading level must be between 1 and 6")
	dartAssert(o.LinkURL == nil || (o.Link != nil && *o.Link), "If linkUrl is set then link must be true")
	p := &SemanticsProperties{SemanticsPropertiesOptions: o}
	p.SetSelf(p)
	return p
}

func (p *SemanticsProperties) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	p.DiagnosticableTreeBase.DebugFillProperties(properties)
	null := foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}
	nullString := foundation.StringPropertyOptions{DefaultValue: foundation.Null}
	nullAttributed := AttributedStringPropertyOptions{DefaultValue: foundation.Null}
	properties.Add(foundation.NewDiagnosticsProperty("checked", p.Checked, null))
	properties.Add(foundation.NewDiagnosticsProperty("mixed", p.Mixed, null))
	properties.Add(foundation.NewDiagnosticsProperty("expanded", p.Expanded, null))
	properties.Add(foundation.NewDiagnosticsProperty("selected", p.Selected, null))
	properties.Add(foundation.NewDiagnosticsProperty("isRequired", p.IsRequired, null))
	properties.Add(foundation.NewStringProperty("identifier", p.Identifier, nullString))
	properties.Add(foundation.NewDiagnosticsProperty("traversalParentIdentifier", p.TraversalParentIdentifier, null))
	properties.Add(foundation.NewDiagnosticsProperty("traversalChildIdentifier", p.TraversalChildIdentifier, null))
	properties.Add(foundation.NewStringProperty("label", p.Label, nullString))
	properties.Add(NewAttributedStringProperty("attributedLabel", p.AttributedLabel, nullAttributed))
	properties.Add(foundation.NewStringProperty("value", p.Value, nullString))
	properties.Add(NewAttributedStringProperty("attributedValue", p.AttributedValue, nullAttributed))
	// Flutter shows value under the increasedValue and decreasedValue names.
	properties.Add(foundation.NewStringProperty("increasedValue", p.Value, nullString))
	properties.Add(NewAttributedStringProperty("attributedIncreasedValue", p.AttributedIncreasedValue, nullAttributed))
	properties.Add(foundation.NewStringProperty("decreasedValue", p.Value, nullString))
	properties.Add(NewAttributedStringProperty("attributedDecreasedValue", p.AttributedDecreasedValue, nullAttributed))
	properties.Add(foundation.NewStringProperty("hint", p.Hint, nullString))
	properties.Add(NewAttributedStringProperty("attributedHint", p.AttributedHint, nullAttributed))
	properties.Add(foundation.NewStringProperty("tooltip", p.Tooltip, nullString))
	properties.Add(foundation.NewEnumProperty("textDirection", p.TextDirection,
		foundation.EnumPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewEnumProperty("role", p.Role, foundation.EnumPropertyOptions{DefaultValue: foundation.Null}))
	properties.Add(foundation.NewEnumProperty("validationResult", p.ValidationResult,
		foundation.EnumPropertyOptions{DefaultValue: ui.SemanticsValidationResultNone}))
	properties.Add(foundation.NewDiagnosticsProperty("sortKey", p.SortKey, null))
	properties.Add(foundation.NewDiagnosticsProperty("hintOverrides", p.HintOverrides, null))
}

func (p *SemanticsProperties) ToStringShort() string {
	return foundation.ObjectRuntimeType(p, "SemanticsProperties")
}

// DebugResetSemanticsIdCounter is debugResetSemanticsIdCounter: in tests,
// resets the identifiers generated for SemanticsNodes.
func DebugResetSemanticsIdCounter() { lastIdentifier = 0 }
