package semantics

// Port of semantics.dart (Flutter 3.50.1), part 3 of 4: SemanticsOwner.

import (
	"reflect"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// orderedSet is Dart's default (insertion-ordered) Set, for the sets whose
// iteration order is observable.
type orderedSet[T comparable] struct {
	items []T
	index map[T]struct{}
}

func newOrderedSet[T comparable]() *orderedSet[T] {
	return &orderedSet[T]{index: map[T]struct{}{}}
}

func (s *orderedSet[T]) add(v T) bool {
	if _, ok := s.index[v]; ok {
		return false
	}
	s.index[v] = struct{}{}
	s.items = append(s.items, v)
	return true
}

func (s *orderedSet[T]) contains(v T) bool {
	_, ok := s.index[v]
	return ok
}

func (s *orderedSet[T]) remove(v T) {
	if _, ok := s.index[v]; !ok {
		return
	}
	delete(s.index, v)
	for i, x := range s.items {
		if x == v {
			s.items = append(s.items[:i:i], s.items[i+1:]...)
			return
		}
	}
}

func (s *orderedSet[T]) clear() {
	s.items = nil
	clear(s.index)
}

func (s *orderedSet[T]) isEmpty() bool { return len(s.items) == 0 }

func (s *orderedSet[T]) length() int { return len(s.items) }

func (s *orderedSet[T]) toList() []T { return append([]T{}, s.items...) }

// runtimeType is Dart's runtimeType for comparing sort key classes (nil
// for null).
func runtimeType(v any) reflect.Type {
	if v == nil {
		return nil
	}
	return reflect.TypeOf(v)
}

// SemanticsOwner owns SemanticsNode objects and notifies listeners of
// changes to the render tree semantics. Use it by pointer; create it with
// NewSemanticsOwner.
type SemanticsOwner struct {
	foundation.ChangeNotifierMixin

	// OnSemanticsUpdate is called when SendSemanticsUpdate builds an
	// update.
	OnSemanticsUpdate SemanticsUpdateCallback

	dirtyNodes           *orderedSet[*SemanticsNode]
	nodes                map[int]*SemanticsNode
	detachedNodes        map[*SemanticsNode]struct{}
	traversalParentNodes map[any]*SemanticsNode
	traversalChildNodes  map[any]*orderedSet[*SemanticsNode]
}

// NewSemanticsOwner is SemanticsOwner(onSemanticsUpdate:) (the required
// named parameter is positional).
func NewSemanticsOwner(onSemanticsUpdate SemanticsUpdateCallback) *SemanticsOwner {
	o := &SemanticsOwner{
		OnSemanticsUpdate:    onSemanticsUpdate,
		dirtyNodes:           newOrderedSet[*SemanticsNode](),
		nodes:                map[int]*SemanticsNode{},
		detachedNodes:        map[*SemanticsNode]struct{}{},
		traversalParentNodes: map[any]*SemanticsNode{},
		traversalChildNodes:  map[any]*orderedSet[*SemanticsNode]{},
	}
	o.SetSelf(o)
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchCreated("semantics", "SemanticsOwner", o)
	}
	return o
}

// RootSemanticsNode is the root node of the semantics tree, if any.
func (o *SemanticsOwner) RootSemanticsNode() *SemanticsNode { return o.nodes[0] }

// GetSemanticsNode is the node with the given id, or nil.
func (o *SemanticsOwner) GetSemanticsNode(id int) *SemanticsNode { return o.nodes[id] }

// Dispose discards the owner's state.
func (o *SemanticsOwner) Dispose() {
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchDisposed(o)
	}
	o.dirtyNodes.clear()
	clear(o.nodes)
	clear(o.detachedNodes)
	clear(o.traversalChildNodes)
	clear(o.traversalParentNodes)
	o.ChangeNotifierMixin.Dispose()
}

func sortByDepth(nodes []*SemanticsNode) {
	dart.ListSort(nodes, func(a, b *SemanticsNode) int { return a.Depth() - b.Depth() })
}

// SendSemanticsUpdate updates the semantics using onSemanticsUpdate.
func (o *SemanticsOwner) SendSemanticsUpdate() {
	// Once the tree is up-to-date, verify that every node is visible.
	if foundation.KDebugMode {
		o.debugCheckInvisibleNodes()
	}

	if o.dirtyNodes.isEmpty() {
		return
	}
	customSemanticsActionIDs := newOrderedSet[int]()
	visitedNodes := []*SemanticsNode{}
	for !o.dirtyNodes.isEmpty() {
		localDirtyNodes := []*SemanticsNode{}
		for _, node := range o.dirtyNodes.items {
			if _, detached := o.detachedNodes[node]; !detached {
				localDirtyNodes = append(localDirtyNodes, node)
			}
		}
		o.dirtyNodes.clear()
		clear(o.detachedNodes)
		sortByDepth(localDirtyNodes)
		visitedNodes = append(visitedNodes, localDirtyNodes...)
		for _, node := range localDirtyNodes {
			dartAssert(node.dirty, nil)
			dartAssert(node.Parent() == nil || !node.Parent().IsPartOfNodeMerging() || node.IsMergedIntoParent(), nil)
			if node.IsPartOfNodeMerging() {
				dartAssert(node.MergeAllDescendantsIntoThisNode() || node.Parent() != nil, nil)
				// If child node is merged into its parent, make sure the parent is
				// marked as dirty
				if node.Parent() != nil && node.Parent().IsPartOfNodeMerging() {
					node.Parent().markDirty() // this can add the node to the dirty list
					node.dirty = false        // Do not send update for this node, as it's now part of its parent
				}
			}

			// Clean up the dirty entry in owner._traversalParentNodes map because
			// it will be updated later.
			for key, oldNode := range o.traversalParentNodes {
				if node == oldNode {
					delete(o.traversalParentNodes, key)
				}
			}
			// Clean up the node from the value set in owner._traversalChildNodes.
			for key, childSet := range o.traversalChildNodes {
				childSet.remove(node)
				if childSet.isEmpty() {
					delete(o.traversalChildNodes, key)
				}
			}

			isTraversalParent := node.isTraversalParent()
			isTraversalChild := node.isTraversalChild()

			// If the node is a traversal parent, then add it to the
			// _traversalParentNodes map for later grafting. Similarly, add the
			// node to the _traversalChildNodes map if it is a traversal child.
			if isTraversalParent {
				existing, has := o.traversalParentNodes[node.traversalParentIdentifier]
				dartAssert(!has || existing == node,
					"The traversalParentIdentifier must be unique. No two semantics nodes can share the same traversalParentIdentifier.")
				o.traversalParentNodes[node.TraversalParentIdentifier()] = node
			} else if isTraversalChild {
				set, ok := o.traversalChildNodes[node.TraversalChildIdentifier()]
				if !ok {
					set = newOrderedSet[*SemanticsNode]()
					o.traversalChildNodes[node.TraversalChildIdentifier()] = set
				}
				set.add(node)
			}

			if !foundation.KIsWeb {
				if node.isTraversalChild() {
					// The node's hit-test parent and traversal parent are different,
					// so its traversal parent should update its children to include
					// this node (OverlayPortal).
					parentNode := o.traversalParentNodes[node.TraversalChildIdentifier()]
					if parentNode != nil && !containsNode(visitedNodes, parentNode) {
						parentNode.markDirty()
					}
				}
			}
		}
	}
	sortByDepth(visitedNodes)
	builder := SemanticsBindingInstance().CreateSemanticsUpdateBuilder()
	for _, node := range visitedNodes {
		dartAssert(node.Parent() == nil || !node.Parent().dirty, nil) // could be null (no parent) or false (not dirty)
		// The _serialize() method marks the node as not dirty, and recurses
		// through the tree to do a deep serialization of all contiguous dirty
		// nodes. Subsequent nodes may no longer be dirty; skip them, and nodes
		// that were reset and dropped entirely.
		if node.dirty && node.Attached() {
			node.addToUpdate(builder, customSemanticsActionIDs)
		}
	}
	o.dirtyNodes.clear()
	for _, actionID := range customSemanticsActionIDs.items {
		action := CustomSemanticsActionGetAction(actionID)
		dartAssert(action != nil, "Null check operator used on a null value")
		var label, hint *string
		if action.Label != "" {
			l := action.Label
			label = &l
		}
		if action.Hint != "" {
			h := action.Hint
			hint = &h
		}
		overrideID := -1
		if action.Action != (ui.SemanticsAction{}) {
			overrideID = action.Action.Index()
		}
		builder.UpdateCustomAction(actionID, ui.SemanticsUpdateBuilderUpdateCustomActionOptions{
			Label: label, Hint: hint, OverrideID: &overrideID,
		})
	}
	o.OnSemanticsUpdate(builder.Build())
	o.NotifyListeners()
}

func containsNode(nodes []*SemanticsNode, node *SemanticsNode) bool {
	for _, n := range nodes {
		if n == node {
			return true
		}
	}
	return false
}

func (o *SemanticsOwner) debugCheckInvisibleNodes() {
	invisibleNodes := []*SemanticsNode{}
	// Finds the invisible nodes in the tree rooted at `node` and adds them to
	// the invisibleNodes list. If a node is itself invisible, all its
	// descendants will be skipped.
	var findInvisibleNodes func(node *SemanticsNode) bool
	findInvisibleNodes = func(node *SemanticsNode) bool {
		if node.Rect().IsEmpty() {
			invisibleNodes = append(invisibleNodes, node)
		} else if !node.MergeAllDescendantsIntoThisNode() {
			node.VisitChildren(findInvisibleNodes)
		}
		return true
	}

	if root := o.RootSemanticsNode(); root != nil {
		// The root node is allowed to be invisible when it has no children.
		if root.ChildrenCount() > 0 && root.Rect().IsEmpty() {
			invisibleNodes = append(invisibleNodes, root)
		} else if !root.MergeAllDescendantsIntoThisNode() {
			root.VisitChildren(findInvisibleNodes)
		}
	}

	if len(invisibleNodes) == 0 {
		return
	}

	errorProperty := foundation.DiagnosticsTreeStyleErrorProperty
	parts := []foundation.DiagnosticsNode{
		foundation.NewErrorSummary("Invisible SemanticsNodes should not be added to the tree."),
		foundation.NewErrorDescription("The following invisible SemanticsNodes were added to the tree:"),
	}
	for _, invisibleNode := range invisibleNodes {
		parts = append(parts, invisibleNode.ToDiagnosticsNode(foundation.ToDiagnosticsNodeOptions{Style: &errorProperty}))
		if parent := invisibleNode.Parent(); parent != nil {
			name := "which was added as a child of"
			parts = append(parts, parent.ToDiagnosticsNode(foundation.ToDiagnosticsNodeOptions{Name: &name, Style: &errorProperty}))
		} else {
			parts = append(parts, foundation.NewErrorDescription("which was added as the root SemanticsNode"))
		}
	}
	parts = append(parts,
		foundation.NewErrorHint("An invisible SemanticsNode is one whose rect is not on screen hence not reachable for users, "+
			"and its semantic information is not merged into a visible parent."),
		foundation.NewErrorHint("An invisible SemanticsNode makes the accessibility experience confusing, "+
			"as it does not provide any visual indication when the user selects it "+
			"via accessibility technologies."),
		foundation.NewErrorHint("Consider removing the above invisible SemanticsNodes if they were added by your "+
			"RenderObject.assembleSemanticsNode implementation, or filing a bug on GitHub:\n"+
			"  https://github.com/flutter/flutter/issues/new?template=02_bug.yml"),
	)
	panic(foundation.NewFlutterErrorFromParts(parts))
}

func (o *SemanticsOwner) getSemanticsActionHandlerForID(id int, action ui.SemanticsAction, args any) SemanticsActionHandler {
	result := o.nodes[id]
	if result == nil {
		return nil
	}
	// For merged nodes, walk descendants whenever the merge root itself does
	// not handle the specific (action, args) pair.
	if result.IsPartOfNodeMerging() && !result.canHandleAction(action, args) {
		var found *SemanticsNode
		result.visitDescendants(func(node *SemanticsNode) bool {
			if node.canHandleAction(action, args) {
				found = node
				return false // found node, abort walk
			}
			return true // continue walk
		})
		result = found
	}
	if result == nil || !result.canHandleAction(action, args) {
		return nil
	}
	return result.actions[action]
}

// PerformAction asks the SemanticsNode with the given id to perform the
// given action (args nil: none). A merged node's descendants that can
// handle the action handle it.
func (o *SemanticsOwner) PerformAction(id int, action ui.SemanticsAction, args ...any) {
	var arg any
	if len(args) > 0 {
		arg = args[0]
	}
	if handler := o.getSemanticsActionHandlerForID(id, action, arg); handler != nil {
		handler(arg)
		return
	}

	// Default actions if no handler was provided.
	if action == ui.SemanticsActionShowOnScreen && o.nodes[id] != nil && o.nodes[id].showOnScreen != nil {
		o.nodes[id].showOnScreen()
	}
}

func (o *SemanticsOwner) getSemanticsActionHandlerForPosition(node *SemanticsNode, position ui.Offset, action ui.SemanticsAction, args any) SemanticsActionHandler {
	if node.Transform() != nil {
		inverse := vm.NewMatrix4Identity()
		if inverse.CopyInverse(node.Transform()) == 0.0 {
			return nil
		}
		position = painting.MatrixUtils.TransformPoint(inverse, position)
	}
	if !node.Rect().Contains(position) {
		return nil
	}
	if node.MergeAllDescendantsIntoThisNode() {
		if node.canHandleAction(action, args) {
			return node.actions[action]
		}
		var result *SemanticsNode
		node.visitDescendants(func(child *SemanticsNode) bool {
			if child.canHandleAction(action, args) {
				result = child
				return false
			}
			return true
		})
		if result == nil {
			return nil
		}
		return result.actions[action]
	}
	if node.HasChildren() {
		for i := len(node.children) - 1; i >= 0; i-- {
			if handler := o.getSemanticsActionHandlerForPosition(node.children[i], position, action, args); handler != nil {
				return handler
			}
		}
	}
	return node.actions[action]
}

// PerformActionAt asks the SemanticsNode at the given position to perform
// the given action (args nil: none).
func (o *SemanticsOwner) PerformActionAt(position ui.Offset, action ui.SemanticsAction, args ...any) {
	node := o.RootSemanticsNode()
	if node == nil {
		return
	}
	var arg any
	if len(args) > 0 {
		arg = args[0]
	}
	if handler := o.getSemanticsActionHandlerForPosition(node, position, action, arg); handler != nil {
		handler(arg)
	}
}

func (o *SemanticsOwner) String() string { return foundation.DescribeIdentity(o) }
