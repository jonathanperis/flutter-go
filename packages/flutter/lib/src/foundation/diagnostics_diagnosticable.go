package foundation

// Port of diagnostics.dart (Flutter 3.50.1), part 5 of 5:
// DiagnosticPropertiesBuilder, DiagnosticableNode, DiagnosticableTreeNode,
// the Diagnosticable mixin, DiagnosticableTree and DiagnosticableTreeMixin.
// See diagnostics.go for the self-dispatch pattern (SetSelf).

import "strings"

// DiagnosticPropertiesBuilder accumulates the properties and configuration
// of a Diagnosticable's DiagnosticsNode.
type DiagnosticPropertiesBuilder struct {
	// Properties accumulated so far.
	Properties []DiagnosticsNode
	// DefaultDiagnosticsTreeStyle is used when no style is specified.
	DefaultDiagnosticsTreeStyle DiagnosticsTreeStyle
	// EmptyBodyDescription is shown when the node has no displayed
	// properties or children.
	EmptyBodyDescription *string
}

// NewDiagnosticPropertiesBuilder is DiagnosticPropertiesBuilder().
func NewDiagnosticPropertiesBuilder() *DiagnosticPropertiesBuilder {
	return &DiagnosticPropertiesBuilder{Properties: []DiagnosticsNode{}, DefaultDiagnosticsTreeStyle: DiagnosticsTreeStyleSparse}
}

// NewDiagnosticPropertiesBuilderFromProperties is
// DiagnosticPropertiesBuilder.fromProperties.
func NewDiagnosticPropertiesBuilderFromProperties(properties []DiagnosticsNode) *DiagnosticPropertiesBuilder {
	return &DiagnosticPropertiesBuilder{Properties: properties, DefaultDiagnosticsTreeStyle: DiagnosticsTreeStyleSparse}
}

// Add adds a property (debug builds only).
func (b *DiagnosticPropertiesBuilder) Add(property DiagnosticsNode) {
	if debugAssertsEnabled {
		b.Properties = append(b.Properties, property)
	}
}

// DiagnosticableNode is a DiagnosticsNode that lazily asks its
// Diagnosticable value for its properties.
type DiagnosticableNode[T Diagnosticable] struct {
	DiagnosticsNodeBase
	value         T
	cachedBuilder *DiagnosticPropertiesBuilder
}

// DiagnosticableNodeOptions are DiagnosticableNode's named parameters (a
// nil Style is resolved from the builder's default style).
type DiagnosticableNodeOptions struct {
	Name  *string
	Style *DiagnosticsTreeStyle
}

// NewDiagnosticableNode is DiagnosticableNode<T>(name:, value:, style:).
func NewDiagnosticableNode[T Diagnosticable](value T, opts ...DiagnosticableNodeOptions) *DiagnosticableNode[T] {
	n := &DiagnosticableNode[T]{}
	n.initDiagnosticableNode(n, value, opts)
	return n
}

func (n *DiagnosticableNode[T]) initDiagnosticableNode(self DiagnosticsNode, value T, opts []DiagnosticableNodeOptions) {
	var o DiagnosticableNodeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	n.init(o.Name, o.Style, true, true, "")
	n.value = value
	n.SetSelf(self)
}

// Value is the Diagnosticable described.
func (n *DiagnosticableNode[T]) Value() any { return n.value }

// TypedValue is Value as a T.
func (n *DiagnosticableNode[T]) TypedValue() T { return n.value }

// Builder is the value's filled DiagnosticPropertiesBuilder, computed once
// (nil outside debug builds).
func (n *DiagnosticableNode[T]) Builder() *DiagnosticPropertiesBuilder {
	if KReleaseMode {
		return nil
	}
	if debugAssertsEnabled && n.cachedBuilder == nil {
		n.cachedBuilder = NewDiagnosticPropertiesBuilder()
		n.value.DebugFillProperties(n.cachedBuilder)
	}
	return n.cachedBuilder
}

// Style is the given style, or the builder's default.
func (n *DiagnosticableNode[T]) Style() DiagnosticsTreeStyle {
	if KReleaseMode {
		return DiagnosticsTreeStyleNone
	}
	if n.hasStyle {
		return n.style
	}
	return n.Builder().DefaultDiagnosticsTreeStyle
}

func (n *DiagnosticableNode[T]) EmptyBodyDescription() *string {
	if KReleaseMode || KProfileMode {
		return strp("")
	}
	return n.Builder().EmptyBodyDescription
}

func (n *DiagnosticableNode[T]) GetProperties() []DiagnosticsNode {
	if KReleaseMode || KProfileMode {
		return []DiagnosticsNode{}
	}
	return n.Builder().Properties
}

func (n *DiagnosticableNode[T]) GetChildren() []DiagnosticsNode { return []DiagnosticsNode{} }

// ToDescription is the value's toStringShort (debug builds only).
func (n *DiagnosticableNode[T]) ToDescription(...ToDescriptionOptions) string {
	result := ""
	if debugAssertsEnabled {
		result = n.value.ToStringShort()
	}
	return result
}

// DiagnosticableTreeNode is the DiagnosticsNode of a DiagnosticableTree.
type DiagnosticableTreeNode struct {
	DiagnosticableNode[DiagnosticableTree]
}

// NewDiagnosticableTreeNode is DiagnosticableTreeNode(name:, value:, style:).
func NewDiagnosticableTreeNode(value DiagnosticableTree, opts ...DiagnosticableNodeOptions) *DiagnosticableTreeNode {
	n := &DiagnosticableTreeNode{}
	n.initDiagnosticableNode(n, value, opts)
	return n
}

func (n *DiagnosticableTreeNode) GetChildren() []DiagnosticsNode {
	return n.value.DebugDescribeChildren()
}

// Diagnosticable is an object with string and DiagnosticsNode debug
// representations of its properties. Implement it by embedding
// DiagnosticableMixin (and calling SetSelf), overriding DebugFillProperties.
type Diagnosticable interface {
	ToStringShort() string
	String() string
	ToString(opts ...DiagnosticableToStringOptions) string
	ToDiagnosticsNode(opts ...ToDiagnosticsNodeOptions) DiagnosticsNode
	DebugFillProperties(properties *DiagnosticPropertiesBuilder)
}

// DiagnosticableToStringOptions are Diagnosticable.toString's named
// parameters (MinLevel nil: info).
type DiagnosticableToStringOptions struct{ MinLevel *DiagnosticLevel }

// ToDiagnosticsNodeOptions are toDiagnosticsNode's named parameters.
type ToDiagnosticsNodeOptions struct {
	Name  *string
	Style *DiagnosticsTreeStyle
}

func (o ToDiagnosticsNodeOptions) nodeOptions() DiagnosticableNodeOptions {
	return DiagnosticableNodeOptions(o)
}

func toDiagnosticsNodeOpts(opts []ToDiagnosticsNodeOptions) DiagnosticableNodeOptions {
	if len(opts) > 0 {
		return opts[0].nodeOptions()
	}
	return DiagnosticableNodeOptions{}
}

const unboundMixin = "foundation: %s used before SetSelf; call SetSelf with the outer object when creating it"

// DiagnosticableMixin is the Diagnosticable mixin: embed it and call
// SetSelf(outer) when creating the outer object.
type DiagnosticableMixin struct{ self Diagnosticable }

// SetSelf points the mixin at the object it is mixed into.
func (m *DiagnosticableMixin) SetSelf(self Diagnosticable) { m.self = self }

func (m *DiagnosticableMixin) outer() Diagnosticable {
	if m.self == nil {
		panic(strings.Replace(unboundMixin, "%s", "DiagnosticableMixin", 1))
	}
	return m.self
}

// ToStringShort is describeIdentity(this).
func (m *DiagnosticableMixin) ToStringShort() string { return DescribeIdentity(m.outer()) }

// String is toString().
func (m *DiagnosticableMixin) String() string { return m.outer().ToString() }

// ToString is the single-line description with properties at minLevel or
// above (toStringShort outside debug builds).
func (m *DiagnosticableMixin) ToString(opts ...DiagnosticableToStringOptions) string {
	return diagnosticableToString(m.outer(), opts)
}

func diagnosticableToString(self Diagnosticable, opts []DiagnosticableToStringOptions) string {
	var o DiagnosticableToStringOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if debugAssertsEnabled {
		style := DiagnosticsTreeStyleSingleLine
		return self.ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: &style}).
			ToString(DiagnosticsNodeToStringOptions{MinLevel: o.MinLevel})
	}
	return self.ToStringShort()
}

// ToDiagnosticsNode is a DiagnosticableNode describing this object.
func (m *DiagnosticableMixin) ToDiagnosticsNode(opts ...ToDiagnosticsNodeOptions) DiagnosticsNode {
	return NewDiagnosticableNode[Diagnosticable](m.outer(), toDiagnosticsNodeOpts(opts))
}

// DebugFillProperties adds nothing; overrides add their properties after
// calling it.
func (m *DiagnosticableMixin) DebugFillProperties(*DiagnosticPropertiesBuilder) {}

// DiagnosticableTree is a Diagnosticable that also describes its children.
type DiagnosticableTree interface {
	Diagnosticable
	ToStringShallow(opts ...ToStringShallowOptions) string
	ToStringDeep(opts ...DiagnosticableTreeToStringDeepOptions) string
	DebugDescribeChildren() []DiagnosticsNode
}

// ToStringShallowOptions are toStringShallow's named parameters (Joiner
// nil: ", "; MinLevel nil: debug).
type ToStringShallowOptions struct {
	Joiner   *string
	MinLevel *DiagnosticLevel
}

// DiagnosticableTreeToStringDeepOptions are DiagnosticableTree.toStringDeep's
// named parameters (MinLevel nil: debug; WrapWidth nil: 65).
type DiagnosticableTreeToStringDeepOptions struct {
	PrefixLineOne    string
	PrefixOtherLines *string
	MinLevel         *DiagnosticLevel
	WrapWidth        *int
}

func toStringShallow(self DiagnosticableTree, first string, opts []ToStringShallowOptions) (string, bool) {
	var o ToStringShallowOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if !debugAssertsEnabled {
		return "", false
	}
	joiner := ", "
	if o.Joiner != nil {
		joiner = *o.Joiner
	}
	minLevel := levelOr(o.MinLevel, DiagnosticLevelDebug)
	var result strings.Builder
	result.WriteString(first)
	result.WriteString(joiner)
	builder := NewDiagnosticPropertiesBuilder()
	self.DebugFillProperties(builder)
	var parts []string
	for _, n := range builder.Properties {
		if !n.IsFiltered(minLevel) {
			parts = append(parts, n.String())
		}
	}
	result.WriteString(strings.Join(parts, joiner))
	return result.String(), true
}

func treeToStringDeep(self DiagnosticableTree, opts []DiagnosticableTreeToStringDeepOptions) string {
	var o DiagnosticableTreeToStringDeepOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return self.ToDiagnosticsNode().ToStringDeep(DiagnosticsNodeToStringDeepOptions{
		PrefixLineOne:    o.PrefixLineOne,
		PrefixOtherLines: o.PrefixOtherLines,
		MinLevel:         o.MinLevel,
		WrapWidth:        o.WrapWidth,
	})
}

// DiagnosticableTreeBase is the abstract class DiagnosticableTree: embed it
// and call SetSelf(outer) when creating the outer object.
type DiagnosticableTreeBase struct{ self DiagnosticableTree }

// SetSelf points the base at the object that embeds it.
func (t *DiagnosticableTreeBase) SetSelf(self DiagnosticableTree) { t.self = self }

func (t *DiagnosticableTreeBase) outer() DiagnosticableTree {
	if t.self == nil {
		panic(strings.Replace(unboundMixin, "%s", "DiagnosticableTreeBase", 1))
	}
	return t.self
}

func (t *DiagnosticableTreeBase) String() string { return t.outer().ToString() }

// ToString is Diagnosticable's toString.
func (t *DiagnosticableTreeBase) ToString(opts ...DiagnosticableToStringOptions) string {
	return diagnosticableToString(t.outer(), opts)
}

// ToStringShallow is a one-line description without children: toString(),
// the joiner, then the properties (DiagnosticableTree writes the full
// toString first, so the properties appear twice; Flutter does the same).
func (t *DiagnosticableTreeBase) ToStringShallow(opts ...ToStringShallowOptions) string {
	self := t.outer()
	if s, ok := toStringShallow(self, self.String(), opts); ok {
		return s
	}
	return self.String()
}

// ToStringDeep renders this object and its descendants.
func (t *DiagnosticableTreeBase) ToStringDeep(opts ...DiagnosticableTreeToStringDeepOptions) string {
	return treeToStringDeep(t.outer(), opts)
}

func (t *DiagnosticableTreeBase) ToStringShort() string { return DescribeIdentity(t.outer()) }

func (t *DiagnosticableTreeBase) ToDiagnosticsNode(opts ...ToDiagnosticsNodeOptions) DiagnosticsNode {
	return NewDiagnosticableTreeNode(t.outer(), toDiagnosticsNodeOpts(opts))
}

// DebugDescribeChildren describes no children.
func (t *DiagnosticableTreeBase) DebugDescribeChildren() []DiagnosticsNode {
	return []DiagnosticsNode{}
}

// DebugFillProperties adds nothing.
func (t *DiagnosticableTreeBase) DebugFillProperties(*DiagnosticPropertiesBuilder) {}

// DiagnosticableTreeMixin is the DiagnosticableTreeMixin mixin: embed it and
// call SetSelf(outer) when creating the outer object.
type DiagnosticableTreeMixin struct{ self DiagnosticableTree }

// SetSelf points the mixin at the object it is mixed into.
func (t *DiagnosticableTreeMixin) SetSelf(self DiagnosticableTree) { t.self = self }

func (t *DiagnosticableTreeMixin) outer() DiagnosticableTree {
	if t.self == nil {
		panic(strings.Replace(unboundMixin, "%s", "DiagnosticableTreeMixin", 1))
	}
	return t.self
}

func (t *DiagnosticableTreeMixin) String() string { return t.outer().ToString() }

// ToString is the single-line description (unlike Diagnosticable's, it
// doesn't fall back to toStringShort outside debug builds).
func (t *DiagnosticableTreeMixin) ToString(opts ...DiagnosticableToStringOptions) string {
	var o DiagnosticableToStringOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	style := DiagnosticsTreeStyleSingleLine
	return t.outer().ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: &style}).
		ToString(DiagnosticsNodeToStringOptions{MinLevel: o.MinLevel})
}

// ToStringShallow is toStringShort, the joiner, then the properties.
func (t *DiagnosticableTreeMixin) ToStringShallow(opts ...ToStringShallowOptions) string {
	self := t.outer()
	if s, ok := toStringShallow(self, self.ToStringShort(), opts); ok {
		return s
	}
	return self.String()
}

// ToStringDeep renders this object and its descendants.
func (t *DiagnosticableTreeMixin) ToStringDeep(opts ...DiagnosticableTreeToStringDeepOptions) string {
	return treeToStringDeep(t.outer(), opts)
}

func (t *DiagnosticableTreeMixin) ToStringShort() string { return DescribeIdentity(t.outer()) }

func (t *DiagnosticableTreeMixin) ToDiagnosticsNode(opts ...ToDiagnosticsNodeOptions) DiagnosticsNode {
	return NewDiagnosticableTreeNode(t.outer(), toDiagnosticsNodeOpts(opts))
}

// DebugDescribeChildren describes no children.
func (t *DiagnosticableTreeMixin) DebugDescribeChildren() []DiagnosticsNode {
	return []DiagnosticsNode{}
}

// DebugFillProperties adds nothing.
func (t *DiagnosticableTreeMixin) DebugFillProperties(*DiagnosticPropertiesBuilder) {}
