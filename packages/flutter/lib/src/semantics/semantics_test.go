package semantics_test

// Port of packages/flutter/test/semantics/semantics_test.dart (Flutter
// 3.50.1).
//
// Skipped:
//   - "after markNeedsSemanticsUpdate() all render objects between two
//     semantic boundaries are asked for annotations": needs rendering's
//     RenderProxyBox, layout and pumpFrame (rendering is not ported).
//   - "SemanticsOwner dispatches memory events": leak_tracker's
//     memoryEvents helper is not ported; the creation/disposal dispatch is
//     checked directly through FlutterMemoryAllocations instead.
//
// Adapted:
//   - "Tags show up in debug properties" uses a SemanticsTag named
//     'RenderViewport.twoPane' (RenderViewport.useTwoPaneSemantics is in
//     rendering).
//   - "getSemanticsData includes tags": tags are slices in Go, so the set
//     the test mutates after assigning it is assigned again.
//   - "SemanticsConfiguration getter/setter": Dart mutates the map returned
//     by customSemanticsActions; Go sets a new one (which also adds the
//     customAction bit, as the setter does).

import (
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

const kMaxFrameworkAccessibilityIdentifier = (1 << 16) - 1

func setUp(t *testing.T) *testBinding {
	t.Helper()
	b := ensureBinding(t)
	semantics.DebugResetSemanticsIdCounter()
	return b
}

func rectLTRB(l, t, r, b float64) ui.Rect { return ui.NewRectFromLTRB(l, t, r, b) }

func nodeWithRect(r ui.Rect) *semantics.SemanticsNode {
	n := semantics.NewSemanticsNode()
	n.SetRect(r)
	return n
}

func boundary(merging bool) *semantics.SemanticsConfiguration {
	c := semantics.NewSemanticsConfiguration()
	c.SetIsSemanticBoundary(true)
	c.SetIsMergingSemanticsOfDescendants(merging)
	return c
}

func update(n *semantics.SemanticsNode, config *semantics.SemanticsConfiguration, children ...*semantics.SemanticsNode) {
	var c []*semantics.SemanticsNode
	if children != nil {
		c = children
	}
	n.UpdateWith(semantics.SemanticsNodeUpdateWithOptions{Config: config, ChildrenInInversePaintOrder: c})
}

var (
	tag1 = semantics.NewSemanticsTag("Tag One")
	tag2 = semantics.NewSemanticsTag("Tag Two")
	tag3 = semantics.NewSemanticsTag("Tag Three")
)

func TestSemanticsNodeTagging(t *testing.T) {
	setUp(t)
	node := semantics.NewSemanticsNode()
	expectEqual(t, node.IsTagged(tag1), false)
	expectEqual(t, node.IsTagged(tag2), false)
	node.Tags = []*semantics.SemanticsTag{tag1}
	expectEqual(t, node.IsTagged(tag1), true)
	expectEqual(t, node.IsTagged(tag2), false)
	node.Tags = append(node.Tags, tag2)
	expectEqual(t, node.IsTagged(tag1), true)
	expectEqual(t, node.IsTagged(tag2), true)
}

func tagsEqual(a, b []*semantics.SemanticsTag) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGetSemanticsDataIncludesTags(t *testing.T) {
	setUp(t)
	tags := []*semantics.SemanticsTag{tag1, tag2}
	node := nodeWithRect(rectLTRB(0, 0, 10, 10))
	node.Tags = tags
	if !tagsEqual(node.GetSemanticsData().Tags, tags) {
		t.Fatalf("tags %v", node.GetSemanticsData().Tags)
	}
	tags = append(tags, tag3)
	node.Tags = tags
	child := nodeWithRect(rectLTRB(5, 5, 10, 10))
	child.Tags = tags
	update(node, boundary(true), child)
	if !tagsEqual(node.GetSemanticsData().Tags, tags) {
		t.Fatalf("tags %v", node.GetSemanticsData().Tags)
	}
}

func spellOut(start, end int) *ui.SpellOutStringAttribute {
	return ui.NewSpellOutStringAttribute(ui.NewTextRange(start, end))
}

func TestConfigurationStringAndAttributedVersions(t *testing.T) {
	setUp(t)
	config := semantics.NewSemanticsConfiguration()
	nodeString := func() string {
		n := semantics.NewSemanticsNode()
		update(n, config)
		return n.String()
	}
	config.SetLabel("label1")
	expectEqual(t, config.Label(), "label1")
	expectEqual(t, config.AttributedLabel().Text, "label1")
	expectEqual(t, len(config.AttributedLabel().Attributes), 0)
	expectString(t, nodeString(), `SemanticsNode#1(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label1")`)

	config.SetAttributedLabel(semantics.NewAttributedString("label2",
		semantics.AttributedStringOptions{Attributes: []ui.StringAttribute{spellOut(0, 1)}}))
	expectEqual(t, config.Label(), "label2")
	expectEqual(t, len(config.AttributedLabel().Attributes), 1)
	_, isSpellOut := config.AttributedLabel().Attributes[0].(*ui.SpellOutStringAttribute)
	expectEqual(t, isSpellOut, true)
	expectEqual(t, config.AttributedLabel().Attributes[0].Range(), ui.NewTextRange(0, 1))
	expectString(t, nodeString(), `SemanticsNode#2(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label2" [SpellOutStringAttribute(TextRange(start: 0, end: 1))])`)

	config.SetLabel("label3")
	expectEqual(t, len(config.AttributedLabel().Attributes), 0)
	expectString(t, nodeString(), `SemanticsNode#3(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label3")`)

	config.SetValue("value1")
	expectEqual(t, config.Value(), "value1")
	expectEqual(t, config.AttributedValue().Text, "value1")
	expectString(t, nodeString(), `SemanticsNode#4(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label3", value: "value1")`)

	config.SetAttributedValue(semantics.NewAttributedString("value2",
		semantics.AttributedStringOptions{Attributes: []ui.StringAttribute{spellOut(0, 1)}}))
	expectEqual(t, config.Value(), "value2")
	expectEqual(t, len(config.AttributedValue().Attributes), 1)
	expectString(t, nodeString(), `SemanticsNode#5(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label3", value: "value2" [SpellOutStringAttribute(TextRange(start: 0, end: 1))])`)

	config.SetValue("value3")
	expectString(t, nodeString(), `SemanticsNode#6(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label3", value: "value3")`)

	config.SetHint("hint1")
	expectEqual(t, config.Hint(), "hint1")
	expectEqual(t, config.AttributedHint().Text, "hint1")
	expectString(t, nodeString(), `SemanticsNode#7(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label3", value: "value3", hint: "hint1")`)

	config.SetAttributedHint(semantics.NewAttributedString("hint2",
		semantics.AttributedStringOptions{Attributes: []ui.StringAttribute{spellOut(0, 1)}}))
	expectEqual(t, config.Hint(), "hint2")
	expectString(t, nodeString(), `SemanticsNode#8(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label3", value: "value3", hint: "hint2" [SpellOutStringAttribute(TextRange(start: 0, end: 1))])`)

	config.SetHint("hint3")
	expectString(t, nodeString(), `SemanticsNode#9(STALE, owner: null, Rect.fromLTRB(0.0, 0.0, 0.0, 0.0), invisible, label: "label3", value: "value3", hint: "hint3")`)
}

func TestIsMergedIntoParent(t *testing.T) {
	setUp(t)
	root := nodeWithRect(rectLTRB(0, 0, 10, 10))
	node1 := nodeWithRect(rectLTRB(1, 0, 10, 10))
	node11 := nodeWithRect(rectLTRB(2, 0, 10, 10))
	node12 := nodeWithRect(rectLTRB(3, 0, 10, 10))
	noMergeConfig := boundary(false)
	mergeConfig := boundary(true)

	update(node1, noMergeConfig, node11, node12)
	expectEqual(t, node1.IsMergedIntoParent(), false)
	expectEqual(t, node1.MergeAllDescendantsIntoThisNode(), false)
	expectEqual(t, node11.IsMergedIntoParent(), false)
	expectEqual(t, node12.IsMergedIntoParent(), false)
	expectEqual(t, root.IsMergedIntoParent(), false)

	update(root, mergeConfig, node1)
	expectEqual(t, node1.IsMergedIntoParent(), true)
	expectEqual(t, node1.MergeAllDescendantsIntoThisNode(), false)
	expectEqual(t, node11.IsMergedIntoParent(), true)
	expectEqual(t, node12.IsMergedIntoParent(), true)
	expectEqual(t, root.IsMergedIntoParent(), false)
	expectEqual(t, root.MergeAllDescendantsIntoThisNode(), true)

	// Change config
	update(node1, mergeConfig, node11, node12)
	expectEqual(t, node1.IsMergedIntoParent(), true)
	expectEqual(t, node1.MergeAllDescendantsIntoThisNode(), true)
	expectEqual(t, node11.IsMergedIntoParent(), true)
	expectEqual(t, node12.IsMergedIntoParent(), true)
	expectEqual(t, root.IsMergedIntoParent(), false)
	expectEqual(t, root.MergeAllDescendantsIntoThisNode(), true)

	update(root, noMergeConfig, node1)
	expectEqual(t, node1.IsMergedIntoParent(), false)
	expectEqual(t, node1.MergeAllDescendantsIntoThisNode(), true)
	expectEqual(t, node11.IsMergedIntoParent(), true)
	expectEqual(t, node12.IsMergedIntoParent(), true)
	expectEqual(t, root.IsMergedIntoParent(), false)
	expectEqual(t, root.MergeAllDescendantsIntoThisNode(), false)
}

func flutterErrorMessage(t *testing.T, f func()) *foundation.FlutterError {
	t.Helper()
	r := expectPanic(t, f)
	err, ok := r.(*foundation.FlutterError)
	if !ok {
		t.Fatalf("expected a FlutterError, got %T %v", r, r)
	}
	return err
}

func TestSendSemanticsUpdateVerifiesNoInvisibleNodes(t *testing.T) {
	setUp(t)
	invisibleRect := rectLTRB(0, 0, 0, 10)
	visibleRect := rectLTRB(0, 0, 10, 10)
	owner := semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {})
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(invisibleRect)
	child := semantics.NewSemanticsNode()

	// It's ok to have an invisible root.
	owner.SendSemanticsUpdate()

	// It's ok to have an invisible child if it's merged to an ancestor.
	root.SetRect(visibleRect)
	child.SetRect(invisibleRect)
	update(root, boundary(true), child)
	owner.SendSemanticsUpdate()

	// It's ok if all nodes are visible.
	root.SetRect(visibleRect)
	child.SetRect(visibleRect)
	update(root, boundary(false), child)
	owner.SendSemanticsUpdate()

	// Invisible root with children bad.
	root.SetRect(invisibleRect)
	child.SetRect(invisibleRect)
	update(root, boundary(true), child)
	err := flutterErrorMessage(t, owner.SendSemanticsUpdate)
	expectString(t, err.Message(), "Invisible SemanticsNodes should not be added to the tree.\n"+
		"The following invisible SemanticsNodes were added to the tree:\n"+
		"SemanticsNode#0(dirty, merge boundary ⛔️, Rect.fromLTRB(0.0, 0.0, 0.0, 10.0), invisible)\n"+
		"which was added as the root SemanticsNode\n"+
		"An invisible SemanticsNode is one whose rect is not on screen hence not reachable for users, and its semantic information is not merged into a visible parent.\n"+
		"An invisible SemanticsNode makes the accessibility experience confusing, as it does not provide any visual indication when the user selects it via accessibility technologies.\n"+
		"Consider removing the above invisible SemanticsNodes if they were added by your RenderObject.assembleSemanticsNode implementation, or filing a bug on GitHub:\n"+
		"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml")

	// Invisible children bad.
	root.SetRect(visibleRect)
	child.SetRect(invisibleRect)
	update(root, boundary(false), child)
	err = flutterErrorMessage(t, owner.SendSemanticsUpdate)
	expectString(t, err.Message(), "Invisible SemanticsNodes should not be added to the tree.\n"+
		"The following invisible SemanticsNodes were added to the tree:\n"+
		"SemanticsNode#1(dirty, Rect.fromLTRB(0.0, 0.0, 0.0, 10.0), invisible)\n"+
		"which was added as a child of:\n"+
		"  SemanticsNode#0(dirty, Rect.fromLTRB(0.0, 0.0, 10.0, 10.0))\n"+
		"An invisible SemanticsNode is one whose rect is not on screen hence not reachable for users, and its semantic information is not merged into a visible parent.\n"+
		"An invisible SemanticsNode makes the accessibility experience confusing, as it does not provide any visual indication when the user selects it via accessibility technologies.\n"+
		"Consider removing the above invisible SemanticsNodes if they were added by your RenderObject.assembleSemanticsNode implementation, or filing a bug on GitHub:\n"+
		"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml")
}

func TestMutateExistingSemanticNodeListErrors(t *testing.T) {
	setUp(t)
	node := nodeWithRect(rectLTRB(0, 0, 10, 10))
	config := boundary(true)
	// A Dart list mutated in place is a Go slice whose backing array is
	// reused: leave room for the append.
	children := make([]*semantics.SemanticsNode, 1, 2)
	children[0] = nodeWithRect(rectLTRB(5, 5, 10, 10))
	update(node, config, children...)
	children = append(children, nodeWithRect(rectLTRB(42, 42, 52, 52)))

	err := flutterErrorMessage(t, func() { update(node, config, children...) })
	expectIgnoringHashCodes(t, err.String(),
		"Failed to replace child semantics nodes because the list of `SemanticsNode`s was mutated.\n"+
			"Instead of mutating the existing list, create a new list containing the desired `SemanticsNode`s.\n"+
			"Error details:\n"+
			"The list's length has changed from 1 to 2.")
	hints := 0
	for _, d := range err.Diagnostics {
		if d.Level() == foundation.DiagnosticLevelHint {
			hints++
			expectString(t, d.String(), "Instead of mutating the existing list, create a new list containing the desired `SemanticsNode`s.")
		}
	}
	expectEqual(t, hints, 1)

	modifiedChildren := []*semantics.SemanticsNode{
		nodeWithRect(rectLTRB(5, 5, 10, 10)),
		nodeWithRect(rectLTRB(10, 10, 20, 20)),
	}
	update(node, config, modifiedChildren...)
	err = flutterErrorMessage(t, func() {
		modifiedChildren[0] = nodeWithRect(rectLTRB(0, 0, 20, 20))
		modifiedChildren[1] = nodeWithRect(rectLTRB(40, 14, 60, 60))
		update(node, config, modifiedChildren...)
	})
	expectIgnoringHashCodes(t, err.ToStringDeep(),
		"FlutterError\n"+
			"   Failed to replace child semantics nodes because the list of\n"+
			"   `SemanticsNode`s was mutated.\n"+
			"   Instead of mutating the existing list, create a new list\n"+
			"   containing the desired `SemanticsNode`s.\n"+
			"   Error details:\n"+
			"   Child node at position 0 was replaced:\n"+
			"   Previous child: SemanticsNode#4(STALE, owner: null, merged up ⬆️, Rect.fromLTRB(5.0, 5.0, 10.0, 10.0))\n"+
			"   New child: SemanticsNode#6(STALE, owner: null, merged up ⬆️, Rect.fromLTRB(0.0, 0.0, 20.0, 20.0))\n"+
			"\n"+
			"   Child node at position 1 was replaced:\n"+
			"   Previous child: SemanticsNode#5(STALE, owner: null, merged up ⬆️, Rect.fromLTRB(10.0, 10.0, 20.0, 20.0))\n"+
			"   New child: SemanticsNode#7(STALE, owner: null, merged up ⬆️, Rect.fromLTRB(40.0, 14.0, 60.0, 60.0))\n")
	nodes := 0
	for _, d := range err.Diagnostics {
		if _, ok := d.Value().(*semantics.SemanticsNode); ok {
			nodes++
		}
	}
	// Two previous children and two new children.
	expectEqual(t, nodes, 4)
}

func TestUpdateWithMarksDirtyWhenRoleChanges(t *testing.T) {
	setUp(t)
	node := semantics.NewSemanticsNode()
	expectEqual(t, node.Role(), ui.SemanticsRoleNone)
	expectEqual(t, *node.DebugIsDirty(), false)
	config := semantics.NewSemanticsConfiguration()
	config.SetRole(ui.SemanticsRoleTab)
	update(node, config)
	expectEqual(t, node.Role(), config.Role())
	expectEqual(t, *node.DebugIsDirty(), true)
}

func customActions(actions ...semantics.CustomSemanticsAction) []dart.MapEntry[semantics.CustomSemanticsAction, func()] {
	m := []dart.MapEntry[semantics.CustomSemanticsAction, func()]{}
	for _, a := range actions {
		m = append(m, dart.MapEntry[semantics.CustomSemanticsAction, func()]{Key: a, Value: func() {}})
	}
	return m
}

func TestUpdateWithMarksDirtyWhenCustomActionsChange(t *testing.T) {
	setUp(t)
	owner := semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {})
	node := semantics.NewSemanticsNodeRoot(owner)
	node.SetRect(rectLTRB(0, 0, 10, 10))
	action1 := semantics.NewCustomSemanticsAction("Action 1")
	config1 := semantics.NewSemanticsConfiguration()
	config1.SetIsSemanticBoundary(true)
	config1.SetCustomSemanticsActions(customActions(action1))
	update(node, config1)
	expectEqual(t, *node.DebugIsDirty(), true)
	owner.SendSemanticsUpdate()
	expectEqual(t, *node.DebugIsDirty(), false)
	action2 := semantics.NewCustomSemanticsAction("Action 2")
	config2 := semantics.NewSemanticsConfiguration()
	config2.SetIsSemanticBoundary(true)
	config2.SetCustomSemanticsActions(customActions(action2))
	update(node, config2)
	expectEqual(t, *node.DebugIsDirty(), true)
}

func TestToStringDeepWithNullTransform(t *testing.T) {
	setUp(t)
	child1 := nodeWithRect(rectLTRB(0, 0, 5, 5))
	child2 := nodeWithRect(rectLTRB(5, 0, 10, 5))
	root := nodeWithRect(rectLTRB(0, 0, 10, 5))
	update(root, nil, child1, child2)
	if root.Transform() != nil || child1.Transform() != nil || child2.Transform() != nil {
		t.Fatal("transform not null")
	}
	expectString(t, root.ToStringDeep(), "SemanticsNode#3\n"+
		" │ STALE\n"+
		" │ owner: null\n"+
		" │ Rect.fromLTRB(0.0, 0.0, 10.0, 5.0)\n"+
		" │\n"+
		" ├─SemanticsNode#1\n"+
		" │   STALE\n"+
		" │   owner: null\n"+
		" │   Rect.fromLTRB(0.0, 0.0, 5.0, 5.0)\n"+
		" │\n"+
		" └─SemanticsNode#2\n"+
		"     STALE\n"+
		"     owner: null\n"+
		"     Rect.fromLTRB(5.0, 0.0, 10.0, 5.0)\n")
}

// customSortKey is the test's CustomSortKey (extends OrdinalSortKey).
type customSortKey struct{ semantics.OrdinalSortKey }

func newCustomSortKey(order float64) *customSortKey {
	k := &customSortKey{OrdinalSortKey: *semantics.NewOrdinalSortKey(order)}
	k.SetSelf(k)
	return k
}

func ordinal(order float64, name ...string) *semantics.OrdinalSortKey {
	if len(name) > 0 {
		return semantics.NewOrdinalSortKey(order, semantics.OrdinalSortKeyOptions{Name: &name[0]})
	}
	return semantics.NewOrdinalSortKey(order)
}

func TestIncompatibleOrdinalSortKeyThrows(t *testing.T) {
	setUp(t)
	r := expectPanic(t, func() { ordinal(0).CompareTo(newCustomSortKey(0)) })
	if !isAssertionError(r) {
		t.Fatalf("not an assertion: %v", r)
	}
}

func TestOrdinalSortKeySameNames(t *testing.T) {
	setUp(t)
	tests := [][2]semantics.SemanticsSortKey{
		{ordinal(0), ordinal(0)}, {ordinal(0), ordinal(1)}, {ordinal(1), ordinal(0)}, {ordinal(1), ordinal(1)},
		{ordinal(0, "a"), ordinal(0, "a")}, {ordinal(0, "a"), ordinal(1, "a")},
		{ordinal(1, "a"), ordinal(0, "a")}, {ordinal(1, "a"), ordinal(1, "a")},
	}
	expected := []int{0, -1, 1, 0, 0, -1, 1, 0}
	for i, tuple := range tests {
		expectEqual(t, tuple[0].CompareTo(tuple[1]), expected[i])
	}
	r := expectPanic(t, func() { ordinal(0).CompareTo(newCustomSortKey(0)) })
	if !isAssertionError(r) {
		t.Fatalf("not an assertion: %v", r)
	}
}

func TestOrdinalSortKeyDifferentNames(t *testing.T) {
	setUp(t)
	tests := [][2]semantics.SemanticsSortKey{
		{ordinal(0), ordinal(0, "bar")}, {ordinal(0), ordinal(1, "bar")},
		{ordinal(1), ordinal(0, "bar")}, {ordinal(1), ordinal(1, "bar")},
		{ordinal(0, "foo"), ordinal(0)}, {ordinal(0, "foo"), ordinal(1)},
		{ordinal(1, "foo"), ordinal(0)}, {ordinal(1, "foo"), ordinal(1)},
		{ordinal(0, "foo"), ordinal(0, "bar")}, {ordinal(0, "foo"), ordinal(1, "bar")},
		{ordinal(1, "foo"), ordinal(0, "bar")}, {ordinal(1, "foo"), ordinal(1, "bar")},
		{ordinal(0, "bar"), ordinal(0, "foo")}, {ordinal(0, "bar"), ordinal(1, "foo")},
		{ordinal(1, "bar"), ordinal(0, "foo")}, {ordinal(1, "bar"), ordinal(1, "foo")},
	}
	expected := []int{-1, -1, -1, -1, 1, 1, 1, 1, 1, 1, 1, 1, -1, -1, -1, -1}
	for i, tuple := range tests {
		expectEqual(t, tuple[0].CompareTo(tuple[1]), expected[i])
	}
}

func TestToStringDeepRespectsChildOrder(t *testing.T) {
	setUp(t)
	child1 := nodeWithRect(rectLTRB(15, 0, 20, 5))
	child2 := nodeWithRect(rectLTRB(10, 0, 15, 5))
	root := nodeWithRect(rectLTRB(0, 0, 20, 5))
	update(root, nil, child1, child2)
	want := "SemanticsNode#3\n" +
		" │ STALE\n" +
		" │ owner: null\n" +
		" │ Rect.fromLTRB(0.0, 0.0, 20.0, 5.0)\n" +
		" │\n" +
		" ├─SemanticsNode#1\n" +
		" │   STALE\n" +
		" │   owner: null\n" +
		" │   Rect.fromLTRB(15.0, 0.0, 20.0, 5.0)\n" +
		" │\n" +
		" └─SemanticsNode#2\n" +
		"     STALE\n" +
		"     owner: null\n" +
		"     Rect.fromLTRB(10.0, 0.0, 15.0, 5.0)\n"
	expectString(t, root.ToStringDeep(), want)
	expectString(t, root.ToStringDeepInOrder(semantics.DebugSemanticsDumpOrderInverseHitTest), want)

	child3 := nodeWithRect(rectLTRB(0, 0, 10, 5))
	update(child3, nil, nodeWithRect(rectLTRB(5, 0, 10, 5)), nodeWithRect(rectLTRB(0, 0, 5, 5)))
	rootComplex := nodeWithRect(rectLTRB(0, 0, 25, 5))
	update(rootComplex, nil, child1, child2, child3)
	wantComplex := "SemanticsNode#7\n" +
		" │ STALE\n" +
		" │ owner: null\n" +
		" │ Rect.fromLTRB(0.0, 0.0, 25.0, 5.0)\n" +
		" │\n" +
		" ├─SemanticsNode#1\n" +
		" │   STALE\n" +
		" │   owner: null\n" +
		" │   Rect.fromLTRB(15.0, 0.0, 20.0, 5.0)\n" +
		" │\n" +
		" ├─SemanticsNode#2\n" +
		" │   STALE\n" +
		" │   owner: null\n" +
		" │   Rect.fromLTRB(10.0, 0.0, 15.0, 5.0)\n" +
		" │\n" +
		" └─SemanticsNode#4\n" +
		"   │ STALE\n" +
		"   │ owner: null\n" +
		"   │ Rect.fromLTRB(0.0, 0.0, 10.0, 5.0)\n" +
		"   │\n" +
		"   ├─SemanticsNode#5\n" +
		"   │   STALE\n" +
		"   │   owner: null\n" +
		"   │   Rect.fromLTRB(5.0, 0.0, 10.0, 5.0)\n" +
		"   │\n" +
		"   └─SemanticsNode#6\n" +
		"       STALE\n" +
		"       owner: null\n" +
		"       Rect.fromLTRB(0.0, 0.0, 5.0, 5.0)\n"
	expectString(t, rootComplex.ToStringDeep(), wantComplex)
	expectString(t, rootComplex.ToStringDeepInOrder(semantics.DebugSemanticsDumpOrderInverseHitTest), wantComplex)
}

const hiddenPropertiesTail = "   flags: []\n" +
	"   invisible\n" +
	"   isHidden: false\n" +
	"   identifier: \"\"\n" +
	"   traversalParentIdentifier: null\n" +
	"   traversalChildIdentifier: null\n" +
	"   label: \"\"\n" +
	"   value: \"\"\n" +
	"   increasedValue: \"\"\n" +
	"   decreasedValue: \"\"\n" +
	"   hint: \"\"\n" +
	"   tooltip: \"\"\n" +
	"   textDirection: null\n" +
	"   sortKey: null\n" +
	"   platformViewId: null\n" +
	"   maxValueLength: null\n" +
	"   currentValueLength: null\n" +
	"   scrollChildren: null\n" +
	"   scrollIndex: null\n" +
	"   scrollExtentMin: null\n" +
	"   scrollPosition: null\n" +
	"   scrollExtentMax: null\n" +
	"   indexInParent: null\n" +
	"   headingLevel: 0\n" +
	"   minValue: null\n" +
	"   maxValue: null\n"

func TestDebugProperties(t *testing.T) {
	setUp(t)
	minimalProperties := semantics.NewSemanticsNode()
	expectString(t, minimalProperties.ToStringDeep(), "SemanticsNode#1\n"+
		"   Rect.fromLTRB(0.0, 0.0, 0.0, 0.0)\n"+
		"   invisible\n")
	hidden := foundation.DiagnosticLevelHidden
	expectString(t, minimalProperties.ToStringDeep(foundation.DiagnosticableTreeToStringDeepOptions{MinLevel: &hidden}),
		"SemanticsNode#1\n"+
			"   owner: null\n"+
			"   isMergedIntoParent: false\n"+
			"   mergeAllDescendantsIntoThisNode: false\n"+
			"   Rect.fromLTRB(0.0, 0.0, 0.0, 0.0)\n"+
			"   tags: null\n"+
			"   actions: []\n"+
			"   customActions: []\n"+
			hiddenPropertiesTail)

	config := boundary(true)
	config.SetOnScrollUp(func() {})
	config.SetOnLongPress(func() {})
	config.SetOnShowOnScreen(func() {})
	config.SetIsChecked(ptr(false))
	config.SetIsSelected(true)
	config.SetIsButton(true)
	config.SetLabel("Use all the properties")
	config.SetTextDirection(ptr(ui.TextDirectionRtl))
	config.SetSortKey(ordinal(1))
	allProperties := semantics.NewSemanticsNode()
	allProperties.SetRect(ui.NewRectFromLTWH(50, 10, 20, 30))
	allProperties.SetTransform(vm.NewMatrix4Translation(vm.NewVector3(10, 10, 0)))
	update(allProperties, config)
	expectIgnoringHashCodes(t, allProperties.ToStringDeep(), "SemanticsNode#2\n"+
		"   STALE\n"+
		"   owner: null\n"+
		"   merge boundary ⛔️\n"+
		"   Rect.fromLTRB(60.0, 20.0, 80.0, 50.0)\n"+
		"   actions: longPress, scrollUp, showOnScreen\n"+
		"   flags: hasCheckedState, isSelected, isButton, hasSelectedState\n"+
		"   label: \"Use all the properties\"\n"+
		"   textDirection: rtl\n"+
		"   sortKey: OrdinalSortKey#19df5(order: 1.0)\n")
	expectString(t, allProperties.GetSemanticsData().String(), "SemanticsData("+
		"Rect.fromLTRB(50.0, 10.0, 70.0, 40.0), "+
		"[1.0,0.0,0.0,10.0; 0.0,1.0,0.0,10.0; 0.0,0.0,1.0,0.0; 0.0,0.0,0.0,1.0], "+
		"actions: [longPress, scrollUp, showOnScreen], "+
		"flags: [hasCheckedState, isSelected, isButton, hasSelectedState], "+
		"label: \"Use all the properties\", textDirection: rtl)")

	scaled := semantics.NewSemanticsNode()
	scaled.SetRect(ui.NewRectFromLTWH(50, 10, 20, 30))
	scaled.SetTransform(vm.NewMatrix4Diagonal3(vm.NewVector3(10, 10, 1)))
	expectString(t, scaled.ToStringDeep(), "SemanticsNode#3\n"+
		"   STALE\n"+
		"   owner: null\n"+
		"   Rect.fromLTRB(50.0, 10.0, 70.0, 40.0) scaled by 10.0x\n")
	expectString(t, scaled.GetSemanticsData().String(),
		"SemanticsData(Rect.fromLTRB(50.0, 10.0, 70.0, 40.0), [10.0,0.0,0.0,0.0; 0.0,10.0,0.0,0.0; 0.0,0.0,1.0,0.0; 0.0,0.0,0.0,1.0])")
}

func TestBlockedActionsDebugProperties(t *testing.T) {
	setUp(t)
	config := semantics.NewSemanticsConfiguration()
	config.IsBlockingUserActions = true
	config.SetOnScrollUp(func() {})
	config.SetOnLongPress(func() {})
	config.SetOnShowOnScreen(func() {})
	config.SetOnDidGainAccessibilityFocus(func() {})
	blocked := semantics.NewSemanticsNode()
	blocked.SetRect(ui.NewRectFromLTWH(50, 10, 20, 30))
	blocked.SetTransform(vm.NewMatrix4Translation(vm.NewVector3(10, 10, 0)))
	update(blocked, config)
	expectIgnoringHashCodes(t, blocked.ToStringDeep(), "SemanticsNode#1\n"+
		"   STALE\n"+
		"   owner: null\n"+
		"   Rect.fromLTRB(60.0, 20.0, 80.0, 50.0)\n"+
		"   actions: didGainAccessibilityFocus, longPress🚫️, scrollUp🚫️,\n"+
		"     showOnScreen🚫️\n")
}

func TestValidationResultDebugProperties(t *testing.T) {
	setUp(t)
	config := semantics.NewSemanticsConfiguration()
	config.SetValidationResult(ui.SemanticsValidationResultValid)
	node := semantics.NewSemanticsNode()
	update(node, config)
	expectString(t, node.ToStringDeep(), "SemanticsNode#1\n"+
		"   STALE\n"+
		"   owner: null\n"+
		"   Rect.fromLTRB(0.0, 0.0, 0.0, 0.0)\n"+
		"   invisible\n"+
		"   validationResult: valid\n")
}

func TestCustomActionsDebugProperties(t *testing.T) {
	setUp(t)
	configuration := semantics.NewSemanticsConfiguration()
	configuration.SetCustomSemanticsActions(customActions(
		semantics.NewCustomSemanticsAction("action1"),
		semantics.NewCustomSemanticsAction("action2"),
		semantics.NewCustomSemanticsAction("action3")))
	actionNode := semantics.NewSemanticsNode()
	update(actionNode, configuration)
	hidden := foundation.DiagnosticLevelHidden
	expectString(t, actionNode.ToStringDeep(foundation.DiagnosticableTreeToStringDeepOptions{MinLevel: &hidden}),
		"SemanticsNode#1\n"+
			"   STALE\n"+
			"   owner: null\n"+
			"   isMergedIntoParent: false\n"+
			"   mergeAllDescendantsIntoThisNode: false\n"+
			"   Rect.fromLTRB(0.0, 0.0, 0.0, 0.0)\n"+
			"   tags: null\n"+
			"   actions: customAction\n"+
			"   customActions: action1, action2, action3\n"+
			hiddenPropertiesTail)
}

func TestAttributedStringCanConcat(t *testing.T) {
	setUp(t)
	string1 := semantics.NewAttributedString("string1",
		semantics.AttributedStringOptions{Attributes: []ui.StringAttribute{spellOut(0, 4)}})
	string2 := semantics.NewAttributedString("string2", semantics.AttributedStringOptions{Attributes: []ui.StringAttribute{
		ui.NewLocaleStringAttribute(ui.NewTextRange(0, 4), ui.NewLocale("es", "MX")),
	}})
	result := string1.Add(string2)
	expectEqual(t, result.Text, "string1string2")
	expectEqual(t, len(result.Attributes), 2)
	expectEqual(t, result.Attributes[0].Range(), ui.NewTextRange(0, 4))
	_, isSpellOut := result.Attributes[0].(*ui.SpellOutStringAttribute)
	expectEqual(t, isSpellOut, true)
	expectString(t, result.String(), "AttributedString('string1string2', attributes: [SpellOutStringAttribute(TextRange(start: 0, end: 4)), LocaleStringAttribute(TextRange(start: 7, end: 11), es-MX)])")
}

func TestSemanticsIDDoesNotRepeat(t *testing.T) {
	setUp(t)
	owner := semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {})
	const expectID = 1400
	var nodeToRemove *semantics.SemanticsNode
	for i := 0; i < kMaxFrameworkAccessibilityIdentifier; i++ {
		node := semantics.NewSemanticsNode()
		node.Attach(owner)
		if node.ID() == expectID {
			nodeToRemove = node
		}
	}
	nodeToRemove.Detach()
	newNode := semantics.NewSemanticsNode()
	newNode.Attach(owner)
	// Id is reused.
	expectEqual(t, newNode.ID(), expectID)
}

func TestPerformActionAtMergedNode(t *testing.T) {
	setUp(t)
	tapped := false
	owner := semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {})
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rectLTRB(0, 0, 10, 10))
	merged := nodeWithRect(rectLTRB(0, 0, 10, 10))
	mergeConfig := boundary(true)
	mergeConfig.SetOnTap(func() { tapped = true })
	rootConfig := semantics.NewSemanticsConfiguration()
	rootConfig.SetIsSemanticBoundary(true)
	update(merged, mergeConfig, []*semantics.SemanticsNode{}...)
	update(root, rootConfig, merged)
	owner.PerformActionAt(ui.NewOffset(5, 5), ui.SemanticsActionTap)
	expectEqual(t, tapped, true)
}

func TestTagsShowUpInDebugProperties(t *testing.T) {
	setUp(t)
	actionNode := semantics.NewSemanticsNode()
	actionNode.Tags = []*semantics.SemanticsTag{semantics.NewSemanticsTag("RenderViewport.twoPane")}
	if !strings.Contains(actionNode.ToStringDeep(), "\n   tags: RenderViewport.twoPane\n") {
		t.Fatal(actionNode.ToStringDeep())
	}
}

func TestSemanticsConfigurationGetterSetter(t *testing.T) {
	setUp(t)
	config := semantics.NewSemanticsConfiguration()
	customAction := semantics.NewCustomSemanticsAction("test")

	expectEqual(t, config.IsSemanticBoundary(), false)
	expectEqual(t, config.IsButton(), false)
	expectEqual(t, config.IsLink(), false)
	expectEqual(t, config.IsMergingSemanticsOfDescendants(), false)
	expectEqual(t, config.IsEnabled() == nil, true)
	expectEqual(t, config.IsChecked() == nil, true)
	expectEqual(t, config.IsSelected(), false)
	expectEqual(t, config.IsBlockingSemanticsOfPreviouslyPaintedNodes, false)
	expectEqual(t, config.IsFocused() == nil, true)
	expectEqual(t, config.IsTextField(), false)

	expectEqual(t, config.OnShowOnScreen() == nil, true)
	expectEqual(t, config.OnScrollDown() == nil, true)
	expectEqual(t, config.OnScrollUp() == nil, true)
	expectEqual(t, config.OnScrollLeft() == nil, true)
	expectEqual(t, config.OnScrollRight() == nil, true)
	expectEqual(t, config.OnScrollToOffset() == nil, true)
	expectEqual(t, config.OnLongPress() == nil, true)
	expectEqual(t, config.OnDecrease() == nil, true)
	expectEqual(t, config.OnIncrease() == nil, true)
	expectEqual(t, config.OnMoveCursorForwardByCharacter() == nil, true)
	expectEqual(t, config.OnMoveCursorBackwardByCharacter() == nil, true)
	expectEqual(t, config.OnTap() == nil, true)
	expectEqual(t, len(config.CustomSemanticsActions()), 0)

	config.SetIsSemanticBoundary(true)
	config.SetIsButton(true)
	config.SetIsLink(true)
	config.SetIsMergingSemanticsOfDescendants(true)
	config.SetIsEnabled(ptr(true))
	config.SetIsChecked(ptr(true))
	config.SetIsSelected(true)
	config.IsBlockingSemanticsOfPreviouslyPaintedNodes = true
	config.SetIsFocused(ptr(true))
	config.SetIsTextField(true)

	onShowOnScreen := func() {}
	onScrollDown := func() {}
	onScrollUp := func() {}
	onScrollLeft := func() {}
	onScrollRight := func() {}
	onScrollToOffset := func(ui.Offset) {}
	onLongPress := func() {}
	onDecrease := func() {}
	onIncrease := func() {}
	onMoveCursorForwardByCharacter := func(bool) {}
	onMoveCursorBackwardByCharacter := func(bool) {}
	onTap := func() {}
	onCustomAction := func() {}

	config.SetOnShowOnScreen(onShowOnScreen)
	config.SetOnScrollDown(onScrollDown)
	config.SetOnScrollUp(onScrollUp)
	config.SetOnScrollLeft(onScrollLeft)
	config.SetOnScrollRight(onScrollRight)
	config.SetOnScrollToOffset(onScrollToOffset)
	config.SetOnLongPress(onLongPress)
	config.SetOnDecrease(onDecrease)
	config.SetOnIncrease(onIncrease)
	config.SetOnMoveCursorForwardByCharacter(onMoveCursorForwardByCharacter)
	config.SetOnMoveCursorBackwardByCharacter(onMoveCursorBackwardByCharacter)
	config.SetOnTap(onTap)
	config.SetCustomSemanticsActions([]dart.MapEntry[semantics.CustomSemanticsAction, func()]{{Key: customAction, Value: onCustomAction}})

	expectEqual(t, config.IsSemanticBoundary(), true)
	expectEqual(t, config.IsButton(), true)
	expectEqual(t, config.IsLink(), true)
	expectEqual(t, config.IsMergingSemanticsOfDescendants(), true)
	expectEqual(t, *config.IsEnabled(), true)
	expectEqual(t, *config.IsChecked(), true)
	expectEqual(t, config.IsSelected(), true)
	expectEqual(t, config.IsBlockingSemanticsOfPreviouslyPaintedNodes, true)
	expectEqual(t, *config.IsFocused(), true)
	expectEqual(t, config.IsTextField(), true)

	expectEqual(t, dart.FunctionsEqual(config.OnShowOnScreen(), onShowOnScreen), true)
	expectEqual(t, dart.FunctionsEqual(config.OnScrollDown(), onScrollDown), true)
	expectEqual(t, dart.FunctionsEqual(config.OnScrollUp(), onScrollUp), true)
	expectEqual(t, dart.FunctionsEqual(config.OnScrollLeft(), onScrollLeft), true)
	expectEqual(t, dart.FunctionsEqual(config.OnScrollRight(), onScrollRight), true)
	expectEqual(t, dart.FunctionsEqual(config.OnScrollToOffset(), semantics.ScrollToOffsetHandler(onScrollToOffset)), true)
	expectEqual(t, dart.FunctionsEqual(config.OnLongPress(), onLongPress), true)
	expectEqual(t, dart.FunctionsEqual(config.OnDecrease(), onDecrease), true)
	expectEqual(t, dart.FunctionsEqual(config.OnIncrease(), onIncrease), true)
	expectEqual(t, dart.FunctionsEqual(config.OnMoveCursorForwardByCharacter(),
		semantics.MoveCursorHandler(onMoveCursorForwardByCharacter)), true)
	expectEqual(t, dart.FunctionsEqual(config.OnMoveCursorBackwardByCharacter(),
		semantics.MoveCursorHandler(onMoveCursorBackwardByCharacter)), true)
	expectEqual(t, dart.FunctionsEqual(config.OnTap(), onTap), true)
	actions := config.CustomSemanticsActions()
	expectEqual(t, len(actions), 1)
	expectEqual(t, actions[0].Key, customAction)
	expectEqual(t, dart.FunctionsEqual(actions[0].Value, onCustomAction), true)
}

func TestAccessibilityFocusBlockTypeUnchangedNotAnnotated(t *testing.T) {
	setUp(t)
	config := semantics.NewSemanticsConfiguration()
	expectEqual(t, config.AccessibilityFocusBlockType(), semantics.AccessibilityFocusBlockTypeNone)
	expectEqual(t, config.HasBeenAnnotated(), false)
	config.SetAccessibilityFocusBlockType(semantics.AccessibilityFocusBlockTypeNone)
	expectEqual(t, config.AccessibilityFocusBlockType(), semantics.AccessibilityFocusBlockTypeNone)
	expectEqual(t, config.HasBeenAnnotated(), false)
	config.SetAccessibilityFocusBlockType(semantics.AccessibilityFocusBlockTypeBlockSubtree)
	expectEqual(t, config.AccessibilityFocusBlockType(), semantics.AccessibilityFocusBlockTypeBlockSubtree)
	expectEqual(t, config.HasBeenAnnotated(), true)
}

func TestCopyPreservesHitTestBehavior(t *testing.T) {
	setUp(t)
	config := semantics.NewSemanticsConfiguration()
	config.SetIsSemanticBoundary(true)
	config.SetLabel("test")
	config.SetHitTestBehavior(ui.SemanticsHitTestBehaviorOpaque)
	expectEqual(t, config.HitTestBehavior(), ui.SemanticsHitTestBehaviorOpaque)
	c := config.Copy()
	expectEqual(t, c.HitTestBehavior(), ui.SemanticsHitTestBehaviorOpaque)
	expectEqual(t, c.IsSemanticBoundary(), true)
	expectEqual(t, c.Label(), "test")
}

func TestCopyPreservesAllHitTestBehaviors(t *testing.T) {
	setUp(t)
	for _, b := range []ui.SemanticsHitTestBehavior{ui.SemanticsHitTestBehaviorDefer,
		ui.SemanticsHitTestBehaviorOpaque, ui.SemanticsHitTestBehaviorTransparent} {
		config := semantics.NewSemanticsConfiguration()
		if b != ui.SemanticsHitTestBehaviorDefer {
			config.SetHitTestBehavior(b)
		}
		expectEqual(t, config.Copy().HitTestBehavior(), b)
	}
}

// Replaces "SemanticsOwner dispatches memory events" (leak_tracker's
// memoryEvents is not ported): the creation and disposal reach
// FlutterMemoryAllocations.
func TestSemanticsOwnerDispatchesMemoryEvents(t *testing.T) {
	setUp(t)
	var events []foundation.ObjectEvent
	listener := func(e foundation.ObjectEvent) { events = append(events, e) }
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	owner := semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {})
	owner.Dispose()
	created, disposed := 0, 0
	for _, e := range events {
		if e.Object() != any(owner) {
			continue
		}
		switch e.(type) {
		case *foundation.ObjectCreated:
			created++
		case *foundation.ObjectDisposed:
			disposed++
		}
	}
	expectEqual(t, created, 1)
	expectEqual(t, disposed, 1)
}

func TestIndexInParentInStringOutput(t *testing.T) {
	setUp(t)
	node := semantics.NewSemanticsNode()
	node.IndexInParent = ptr(10)
	if !strings.Contains(node.String(), "indexInParent: 10") {
		t.Fatal(node.String())
	}
}

func TestSemanticsLabelBuilder(t *testing.T) {
	setUp(t)
	t.Run("basic functionality with default separator", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		expectEqual(t, builder.IsEmpty(), true)
		expectEqual(t, builder.Length(), 0)
		builder.AddPart("Hello")
		expectEqual(t, builder.IsEmpty(), false)
		expectEqual(t, builder.Length(), 1)
		builder.AddPart("world")
		expectEqual(t, builder.Length(), 2)
		expectEqual(t, builder.Build(), "Hello world")
	})
	sep := func(s string) semantics.SemanticsLabelBuilderOptions {
		return semantics.SemanticsLabelBuilderOptions{Separator: &s}
	}
	t.Run("custom separator", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder(sep(", "))
		builder.AddPart("One")
		builder.AddPart("Two")
		builder.AddPart("Three")
		expectEqual(t, builder.Build(), "One, Two, Three")
	})
	t.Run("empty separator", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder(sep(""))
		builder.AddPart("Hello")
		builder.AddPart("World")
		expectEqual(t, builder.Build(), "HelloWorld")
	})
	t.Run("ignores empty parts", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart("Hello")
		builder.AddPart("")
		builder.AddPart("world")
		expectEqual(t, builder.Build(), "Hello world")
		expectEqual(t, builder.Length(), 2)
	})
	t.Run("single part", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart("Single")
		expectEqual(t, builder.Build(), "Single")
	})
	t.Run("empty builder", func(t *testing.T) {
		expectEqual(t, semantics.NewSemanticsLabelBuilder().Build(), "")
	})
	t.Run("clear functionality", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart("Hello")
		builder.AddPart("world")
		expectEqual(t, builder.Length(), 2)
		builder.Clear()
		expectEqual(t, builder.IsEmpty(), true)
		expectEqual(t, builder.Length(), 0)
		expectEqual(t, builder.Build(), "")
	})
	t.Run("reusable builder", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart("First")
		builder.AddPart("use")
		expectEqual(t, builder.Build(), "First use")
		builder.Clear()
		builder.AddPart("Second")
		builder.AddPart("use")
		expectEqual(t, builder.Build(), "Second use")
	})
	t.Run("reuse without clearing", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart("First")
		builder.AddPart("batch")
		expectEqual(t, builder.Build(), "First batch")
		expectEqual(t, builder.Length(), 2)
		builder.AddPart("Second")
		builder.AddPart("batch")
		expectEqual(t, builder.Build(), "First batch Second batch")
		expectEqual(t, builder.Length(), 4)
		builder.AddPart("Final")
		expectEqual(t, builder.Build(), "First batch Second batch Final")
		expectEqual(t, builder.Length(), 5)
	})
}

func TestSemanticsLabelBuilderTextDirection(t *testing.T) {
	setUp(t)
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	part := func(d *ui.TextDirection) semantics.SemanticsLabelBuilderAddPartOptions {
		return semantics.SemanticsLabelBuilderAddPartOptions{TextDirection: d}
	}
	t.Run("no text direction embedding when overall direction is null", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart("Hello", part(&ltr))
		builder.AddPart("مرحبا", part(&rtl))
		expectEqual(t, builder.Build(), "Hello مرحبا")
	})
	t.Run("text direction embedding with LTR overall direction", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{TextDirection: &ltr})
		builder.AddPart("Hello", part(&ltr))
		builder.AddPart("مرحبا", part(&rtl))
		builder.AddPart("world", part(&ltr))
		expectEqual(t, builder.Build(), "Hello ‫مرحبا‬ world")
	})
	t.Run("text direction embedding with RTL overall direction", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{TextDirection: &rtl})
		builder.AddPart("مرحبا", part(&rtl))
		builder.AddPart("Hello", part(&ltr))
		expectEqual(t, builder.Build(), "مرحبا ‪Hello‬")
	})
	t.Run("no embedding when all parts have same direction", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{TextDirection: &ltr})
		builder.AddPart("Hello", part(&ltr))
		builder.AddPart("world", part(&ltr))
		expectEqual(t, builder.Build(), "Hello world")
	})
	t.Run("part direction falls back to overall direction", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{TextDirection: &ltr})
		builder.AddPart("Hello")
		builder.AddPart("world")
		expectEqual(t, builder.Build(), "Hello world")
	})
	t.Run("complex multilingual example", func(t *testing.T) {
		s := ", "
		builder := semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{TextDirection: &ltr, Separator: &s})
		builder.AddPart("Welcome", part(&ltr))
		builder.AddPart("مرحبا", part(&rtl))
		builder.AddPart("שלום", part(&rtl))
		builder.AddPart("to our app", part(&ltr))
		expectEqual(t, builder.Build(), "Welcome, ‫مرحبا‬, ‫שלום‬, to our app")
	})
}

func TestSemanticsLabelBuilderEdgeCases(t *testing.T) {
	setUp(t)
	t.Run("very long labels", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart(strings.Repeat("A", 1000))
		builder.AddPart("short")
		label := builder.Build()
		expectEqual(t, len(label), 1006) // 1000 + 1 (space) + 5
		expectEqual(t, strings.HasPrefix(label, "AAAA"), true)
		expectEqual(t, strings.HasSuffix(label, " short"), true)
	})
	t.Run("many parts", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		for i := 0; i < 100; i++ {
			builder.AddPart("part" + strconvItoa(i))
		}
		label := builder.Build()
		expectEqual(t, strings.HasPrefix(label, "part0 part1"), true)
		expectEqual(t, strings.HasSuffix(label, "part98 part99"), true)
		expectEqual(t, builder.Length(), 100)
	})
	t.Run("special characters in separators", func(t *testing.T) {
		s := " | "
		builder := semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{Separator: &s})
		builder.AddPart("first")
		builder.AddPart("second")
		builder.AddPart("third")
		expectEqual(t, builder.Build(), "first | second | third")
	})
	t.Run("Unicode characters in content", func(t *testing.T) {
		builder := semantics.NewSemanticsLabelBuilder()
		builder.AddPart("Emoji: 😀🎉")
		builder.AddPart("Math: ∑∆π")
		builder.AddPart("Currency: €£¥")
		expectEqual(t, builder.Build(), "Emoji: 😀🎉 Math: ∑∆π Currency: €£¥")
	})
}

func jsonValue(m []dart.MapEntry[string, any], key string) any {
	for _, e := range m {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

func TestToJson(t *testing.T) {
	setUp(t)
	config := semantics.NewSemanticsConfiguration()
	config.SetLabel("Test Label")
	config.SetTextDirection(ptr(ui.TextDirectionLtr))
	config.SetValue("Test Value")
	config.SetHint("Test Hint")
	config.SetIsButton(true)
	node := nodeWithRect(rectLTRB(0, 0, 100, 50))
	update(node, config)

	check := func(m []dart.MapEntry[string, any]) {
		t.Helper()
		expectEqual(t, jsonValue(m, "label"), any("Test Label"))
		expectEqual(t, jsonValue(m, "value"), any("Test Value"))
		expectEqual(t, jsonValue(m, "hint"), any("Test Hint"))
		flags := jsonValue(m, "flags").([]string)
		found := false
		for _, f := range flags {
			found = found || f == "isButton"
		}
		expectEqual(t, found, true)
		expectEqual(t, dart.JSONEncode(jsonValue(m, "rect")), `{"left":0.0,"top":0.0,"width":100.0,"height":50.0}`)
	}
	check(node.GetSemanticsData().ToJson())
	nodeJSON := node.ToJson()
	expectEqual(t, jsonValue(nodeJSON, "id"), any(node.ID()))
	check(nodeJSON)
	expectEqual(t, len(jsonValue(nodeJSON, "childrenInTraversalOrder").([]int)), 0)
	expectEqual(t, len(jsonValue(nodeJSON, "childrenInHitTestOrder").([]int)), 0)
}
func strconvItoa(i int) string { return strconv.Itoa(i) }

// Own tests: SemanticsData equality and the identity transform rule.
func TestSemanticsDataEqualityAndIdentityTransform(t *testing.T) {
	setUp(t)
	a := nodeWithRect(rectLTRB(0, 0, 10, 10))
	b := nodeWithRect(rectLTRB(0, 0, 10, 11))
	expectEqual(t, a.GetSemanticsData().Equal(a.GetSemanticsData()), true)
	expectEqual(t, a.GetSemanticsData().Equal(b.GetSemanticsData()), false)

	// Reordering the same children marks the parent dirty.
	owner := semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {})
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(rectLTRB(0, 0, 100, 100))
	c1, c2 := nodeWithRect(rectLTRB(0, 0, 10, 10)), nodeWithRect(rectLTRB(20, 0, 30, 10))
	update(root, nil, c1, c2)
	owner.SendSemanticsUpdate()
	expectEqual(t, *root.DebugIsDirty(), false)
	update(root, nil, c1, c2)
	expectEqual(t, *root.DebugIsDirty(), false)
	update(root, nil, c2, c1)
	expectEqual(t, *root.DebugIsDirty(), true)

	n := nodeWithRect(rectLTRB(0, 0, 10, 10))
	n.SetTransform(vm.NewMatrix4TranslationValues(1, 2, 0))
	if n.Transform() == nil {
		t.Fatal("translation not stored")
	}
	n.SetTransform(vm.NewMatrix4Identity())
	if n.Transform() != nil {
		t.Fatal("identity must be stored as nil")
	}
}
