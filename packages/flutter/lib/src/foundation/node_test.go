// Tests of node.go and capabilities.go. Flutter has no foundation test for
// AbstractNode (deprecated; RenderObject and SemanticsNode used to cover
// it) nor for capabilities.dart; these check the ported rules.
package foundation

import (
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

type testNode struct {
	AbstractNodeBase
	name     string
	children []*testNode
	log      *[]string
}

func newTestNode(name string, log *[]string) *testNode {
	n := &testNode{name: name, log: log}
	n.SetSelf(n)
	return n
}

func (n *testNode) RedepthChildren() {
	*n.log = append(*n.log, "redepth "+n.name)
	for _, c := range n.children {
		n.RedepthChild(c)
	}
}

func (n *testNode) Attach(owner any) {
	n.AbstractNodeBase.Attach(owner)
	*n.log = append(*n.log, "attach "+n.name)
	for _, c := range n.children {
		c.Attach(owner)
	}
}

func (n *testNode) Detach() {
	n.AbstractNodeBase.Detach()
	*n.log = append(*n.log, "detach "+n.name)
	for _, c := range n.children {
		c.Detach()
	}
}

func (n *testNode) add(c *testNode) {
	n.children = append(n.children, c)
	n.AdoptChild(c)
}

func (n *testNode) remove(c *testNode) {
	for i, x := range n.children {
		if x == c {
			n.children = append(n.children[:i], n.children[i+1:]...)
		}
	}
	n.DropChild(c)
}

func isAssertionError(r any) bool { _, ok := r.(*dart.AssertionError); return ok }

func TestAbstractNode(t *testing.T) {
	var log []string
	root, a, b, c := newTestNode("root", &log), newTestNode("a", &log), newTestNode("b", &log), newTestNode("c", &log)
	equal(t, root.Depth(), 0)
	equal(t, root.Attached(), false)
	a.add(b) // b gets depth 1 (a is 0)
	equal(t, b.Depth(), 1)
	if b.Parent() != AbstractNode(a) {
		t.Fatal("parent")
	}
	root.Attach("owner")
	equal(t, root.Attached(), true)
	equal(t, root.Owner(), any("owner"))
	log = nil
	root.add(a) // a: depth 1, so b is redepthed to 2; a attaches with the owner
	equal(t, a.Depth(), 1)
	equal(t, b.Depth(), 2)
	equal(t, b.Owner(), any("owner"))
	if got := []string{"attach a", "attach b", "redepth a", "redepth b"}; !equalStrings(log, got) {
		t.Fatalf("log %q, want %q", log, got)
	}
	// A child already deeper than the parent keeps its depth.
	c.add(newTestNode("d", &log))
	deep := newTestNode("deep", &log)
	deep.depth = 7
	log = nil
	b.add(deep)
	equal(t, deep.Depth(), 7)
	if !equalStrings(log, []string{"attach deep"}) {
		t.Fatalf("log %q", log)
	}
	// Dropping detaches.
	log = nil
	root.remove(a)
	equal(t, a.Attached(), false)
	equal(t, b.Attached(), false)
	if a.Parent() != nil {
		t.Fatal("parent kept")
	}
	if !equalStrings(log, []string{"detach a", "detach b", "detach deep"}) {
		t.Fatalf("log %q", log)
	}
	// Assertions.
	expectPanic(t, isAssertionError, func() { a.AdoptChild(b) })         // b already has a parent
	expectPanic(t, isAssertionError, func() { b.AdoptChild(a) })         // cycle
	expectPanic(t, isAssertionError, func() { c.DropChild(b) })          // not c's child
	expectPanic(t, isAssertionError, func() { a.Detach() })              // not attached
	expectPanic(t, isAssertionError, func() { root.Attach("other") })    // already attached
	expectPanic(t, isAssertionError, func() { root.RedepthChild(deep) }) // different owner
	plain := NewAbstractNode()
	equal(t, plain.Depth(), 0)
	plain.RedepthChildren()
}

func equalStrings(a, b []string) bool {
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

func TestCapabilitiesThrowOutsideTheWeb(t *testing.T) {
	isUnimplemented := func(message string) func(any) bool {
		return func(r any) bool {
			e, ok := r.(dart.UnimplementedError)
			return ok && e.Error() == "UnimplementedError: "+message
		}
	}
	expectPanic(t, isUnimplemented("isCanvasKit is not implemented for dart:io."), func() { IsCanvasKit() })
	expectPanic(t, isUnimplemented("isSkwasm is not implemented for dart:io."), func() { IsSkwasm() })
	expectPanic(t, isUnimplemented("isCanvasKit is not implemented for dart:io."), func() { IsSkiaWeb() })
}
