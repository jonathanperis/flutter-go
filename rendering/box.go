package rendering

import "fmt"

// RenderBox is a node in the render tree.
//
// Go has no inheritance, so Flutter's RenderObject → RenderBox → RenderProxyBox
// hierarchy becomes: every render object embeds a Box (shared state) and
// implements this interface (the "virtual" methods).
type RenderBox interface {
	// Base returns the embedded shared state. Embedding Box provides it.
	Base() *Box
	// PerformLayout computes this object's size and lays out and positions
	// children (via Layout and child.Base().Offset). Constraints are in Box().
	PerformLayout(c Constraints) Size
	// Paint paints this object and its children at offset.
	Paint(cv Canvas, offset Offset)
	// Children returns the children in paint order.
	Children() []RenderBox
}

// SelfHitTester is implemented by render objects that are hit themselves
// (Flutter's hitTestSelf). Others are hit only through their children.
type SelfHitTester interface{ HitTestSelf(pos Offset) bool }

// Disposer is implemented by render objects that hold native resources.
type Disposer interface{ Dispose() }

// Box is the state every render object shares.
type Box struct {
	Size Size
	// Offset is this box's position in its parent's coordinate space
	// (Flutter's BoxParentData.offset). Set by the parent during layout.
	Offset Offset
	// Flex parent data, read by RenderFlex (set by Expanded/Flexible).
	Flex int
	Fit  FlexFit

	parent      RenderBox
	constraints Constraints
	laidOut     bool
	needsLayout bool
	// onNeedsFrame is set on the root only.
	onNeedsFrame func()
}

// Base implements RenderBox for every type that embeds Box.
func (b *Box) Base() *Box { return b }

func (b *Box) Parent() RenderBox        { return b.parent }
func (b *Box) Constraints() Constraints { return b.constraints }

// Layout lays out r with constraints c, skipping work if neither the
// constraints nor r's subtree changed since the last layout.
func Layout(r RenderBox, c Constraints) Size {
	b := r.Base()
	if b.laidOut && !b.needsLayout && b.constraints == c {
		return b.Size
	}
	b.constraints = c
	size := r.PerformLayout(c)
	if !c.IsSatisfiedBy(size) {
		panic(fmt.Sprintf("%T returned size %v that violates %v", r, size, c))
	}
	b.Size, b.laidOut, b.needsLayout = size, true, false
	return size
}

// MarkNeedsLayout marks r and its ancestors dirty and requests a frame.
// (Flutter stops at relayout boundaries; we always walk to the root.)
func MarkNeedsLayout(r RenderBox) {
	for r != nil {
		b := r.Base()
		b.needsLayout = true
		if b.parent == nil && b.onNeedsFrame != nil {
			b.onNeedsFrame()
		}
		r = b.parent
	}
}

// MarkNeedsPaint requests a frame. Every frame repaints the whole tree; there
// are no repaint boundaries or layers yet.
func MarkNeedsPaint(r RenderBox) {
	for r.Base().parent != nil {
		r = r.Base().parent
	}
	if f := r.Base().onNeedsFrame; f != nil {
		f()
	}
}

// SetRoot makes r a root whose dirtiness calls onNeedsFrame.
func SetRoot(r RenderBox, onNeedsFrame func()) {
	r.Base().parent = nil
	r.Base().onNeedsFrame = onNeedsFrame
}

// adopt sets parent pointers for a new child list and reports whether it differs.
func adopt(parent RenderBox, old, new []RenderBox) bool {
	changed := len(old) != len(new)
	for i, c := range new {
		if !changed && old[i] != c {
			changed = true
		}
		c.Base().parent = parent
	}
	return changed
}

// GlobalOffset returns r's position relative to the root.
func GlobalOffset(r RenderBox) Offset {
	var o Offset
	for ; r != nil; r = r.Base().parent {
		o = o.Add(r.Base().Offset)
	}
	return o
}

// HitTest returns the render objects under pos, deepest first (Flutter's
// HitTestResult path). pos is in r's parent coordinate space.
func HitTest(r RenderBox, pos Offset) []RenderBox {
	var path []RenderBox
	hitTest(r, pos.Sub(r.Base().Offset), &path)
	return path
}

func hitTest(r RenderBox, local Offset, path *[]RenderBox) bool {
	if !r.Base().Size.Contains(local) {
		return false
	}
	hit := false
	children := r.Children()
	for i := len(children) - 1; i >= 0 && !hit; i-- { // topmost child first
		c := children[i]
		hit = hitTest(c, local.Sub(c.Base().Offset), path)
	}
	if !hit {
		if s, ok := r.(SelfHitTester); ok {
			hit = s.HitTestSelf(local)
		}
	}
	if hit {
		*path = append(*path, r)
	}
	return hit
}

// Walk visits r and its descendants depth-first, pre-order.
func Walk(r RenderBox, visit func(r RenderBox, depth int)) {
	var walk func(RenderBox, int)
	walk = func(r RenderBox, d int) {
		visit(r, d)
		for _, c := range r.Children() {
			walk(c, d+1)
		}
	}
	walk(r, 0)
}

// Dispose releases native resources in r's subtree.
func Dispose(r RenderBox) {
	Walk(r, func(r RenderBox, _ int) {
		if d, ok := r.(Disposer); ok {
			d.Dispose()
		}
	})
}

// ---------------------------------------------------------------------------
// Child-holding helpers, embedded by concrete render objects.

// SingleChild holds zero or one child.
type SingleChild struct {
	child RenderBox
	one   [1]RenderBox // backing store so Children() doesn't allocate
}

func (s *SingleChild) Child() RenderBox { return s.child }
func (s *SingleChild) Children() []RenderBox {
	if s.child == nil {
		return nil
	}
	s.one[0] = s.child
	return s.one[:]
}

// SetChild replaces the child of self, which must embed this SingleChild.
func (s *SingleChild) SetChild(self RenderBox, child RenderBox) {
	var old, new []RenderBox
	if s.child != nil {
		old = []RenderBox{s.child}
	}
	if child != nil {
		new = []RenderBox{child}
	}
	if adopt(self, old, new) {
		s.child = child
		MarkNeedsLayout(self)
	}
}

// MultiChild holds an ordered list of children.
type MultiChild struct{ children []RenderBox }

func (m *MultiChild) Children() []RenderBox { return m.children }

func (m *MultiChild) SetChildren(self RenderBox, children []RenderBox) {
	if adopt(self, m.children, children) {
		m.children = append([]RenderBox(nil), children...)
		MarkNeedsLayout(self)
	}
}
