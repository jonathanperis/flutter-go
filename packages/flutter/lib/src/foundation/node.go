package foundation

// Port of node.dart (Flutter 3.50.1).

// AbstractNode is a base class for nodes in a tree (Deprecated: if needed,
// inline any required functionality of AbstractNode in your class
// directly; deprecated after v3.12.0-4.0.pre).
//
// Nodes have a depth, an owner, a parent and attach/detach state. Subclasses
// embed AbstractNodeBase, call SetSelf on creation and override
// RedepthChildren, Attach, Detach, AdoptChild and DropChild (calling the
// embedded base's method as "super"). Dart's assertions panic.
type AbstractNode interface {
	// Depth is the depth of this node in the tree: greater than its
	// parent's.
	Depth() int
	// RedepthChild adjusts child's depth to be greater than this node's.
	// Only for subclasses, from RedepthChildren.
	RedepthChild(child AbstractNode)
	// RedepthChildren calls RedepthChild for each child.
	RedepthChildren()
	// Owner is the owner of the tree this node is attached to, or nil.
	Owner() any
	// Attached reports whether the node is in a tree with an owner.
	Attached() bool
	// Attach marks the node attached to owner.
	Attach(owner any)
	// Detach marks the node detached.
	Detach()
	// Parent is this node's parent, or nil.
	Parent() AbstractNode
	// AdoptChild marks child as this node's child (for subclasses).
	AdoptChild(child AbstractNode)
	// DropChild disconnects child from this node (for subclasses).
	DropChild(child AbstractNode)

	abstractNodeBase() *AbstractNodeBase
}

// AbstractNodeBase is AbstractNode's state and default methods, to embed.
type AbstractNodeBase struct {
	self   AbstractNode
	depth  int
	owner  any
	parent AbstractNode
}

// NewAbstractNode creates a plain AbstractNode (Dart's AbstractNode()).
func NewAbstractNode() *AbstractNodeBase {
	n := &AbstractNodeBase{}
	n.self = n
	return n
}

// SetSelf points the base at the node that embeds it.
func (n *AbstractNodeBase) SetSelf(self AbstractNode) { n.self = self }

func (n *AbstractNodeBase) abstractNodeBase() *AbstractNodeBase { return n }

func (n *AbstractNodeBase) outer() AbstractNode {
	if n.self == nil {
		panic("foundation: AbstractNodeBase used before SetSelf; call SetSelf with the outer object when creating it")
	}
	return n.self
}

func (n *AbstractNodeBase) Depth() int { return n.depth }

func (n *AbstractNodeBase) RedepthChild(child AbstractNode) {
	dartAssert(child.Owner() == n.outer().Owner(), nil)
	c := child.abstractNodeBase()
	if c.depth <= n.depth {
		c.depth = n.depth + 1
		child.RedepthChildren()
	}
}

func (n *AbstractNodeBase) RedepthChildren() {}

func (n *AbstractNodeBase) Owner() any { return n.owner }

func (n *AbstractNodeBase) Attached() bool { return n.owner != nil }

func (n *AbstractNodeBase) Attach(owner any) {
	dartAssert(n.owner == nil, nil)
	n.owner = owner
}

func (n *AbstractNodeBase) Detach() {
	dartAssert(n.owner != nil, nil)
	n.owner = nil
	self := n.outer()
	dartAssert(self.Parent() == nil || self.Attached() == self.Parent().Attached(), nil)
}

func (n *AbstractNodeBase) Parent() AbstractNode { return n.parent }

func (n *AbstractNodeBase) AdoptChild(child AbstractNode) {
	c := child.abstractNodeBase()
	dartAssert(c.parent == nil, nil)
	self := n.outer()
	node := self
	for node.Parent() != nil {
		node = node.Parent()
	}
	dartAssert(node != child, nil) // indicates we are about to create a cycle
	c.parent = self
	if self.Attached() {
		child.Attach(n.owner)
	}
	self.RedepthChild(child)
}

func (n *AbstractNodeBase) DropChild(child AbstractNode) {
	c := child.abstractNodeBase()
	self := n.outer()
	dartAssert(c.parent == self, nil)
	dartAssert(child.Attached() == self.Attached(), nil)
	c.parent = nil
	if self.Attached() {
		child.Detach()
	}
}
