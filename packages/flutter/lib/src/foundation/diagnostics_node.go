package foundation

// Port of diagnostics.dart (Flutter 3.50.1), part 3 of 5: DiagnosticsNode
// (interface + embeddable DiagnosticsNodeBase), its JSON serialization,
// DiagnosticsSerializationDelegate and DiagnosticsBlock. See diagnostics.go
// for how overridable methods dispatch (the self field).

import (
	"reflect"
	"slices"
	"strings"
)

// DiagnosticsNode is diagnostics data for a value: a name, a description,
// properties and children, plus hints for how to display them.
//
// Implementations embed DiagnosticsNodeBase (directly or through
// DiagnosticsProperty, DiagnosticsBlock...) and implement ToDescription,
// Value, GetProperties and GetChildren.
type DiagnosticsNode interface {
	// Name labels the node. Dart's null name is "" here; the base keeps
	// whether it was null (it matters for JSON and DiagnosticsBlock).
	Name() string
	ToDescription(opts ...ToDescriptionOptions) string
	ShowSeparator() bool
	IsFiltered(minLevel DiagnosticLevel) bool
	Level() DiagnosticLevel
	ShowName() bool
	// LinePrefix is added at the start of each line ("" for Dart's null).
	LinePrefix() string
	EmptyBodyDescription() *string
	Value() any
	Style() DiagnosticsTreeStyle
	AllowWrap() bool
	AllowNameWrap() bool
	AllowTruncate() bool
	GetProperties() []DiagnosticsNode
	GetChildren() []DiagnosticsNode
	ToTimelineArguments() map[string]string
	ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any
	ToJsonMapIterative(delegate DiagnosticsSerializationDelegate) map[string]any
	String() string
	ToString(opts ...DiagnosticsNodeToStringOptions) string
	TextTreeConfiguration() *TextTreeConfiguration
	ToStringDeep(opts ...DiagnosticsNodeToStringDeepOptions) string

	diagnosticsNodeBase() *DiagnosticsNodeBase
}

// ToDescriptionOptions are toDescription's named parameters.
type ToDescriptionOptions struct{ ParentConfiguration *TextTreeConfiguration }

// DiagnosticsNodeToStringOptions are DiagnosticsNode.toString's named
// parameters (MinLevel nil: info).
type DiagnosticsNodeToStringOptions struct {
	ParentConfiguration *TextTreeConfiguration
	MinLevel            *DiagnosticLevel
}

// DiagnosticsNodeToStringDeepOptions are DiagnosticsNode.toStringDeep's
// named parameters (MinLevel nil: debug; WrapWidth nil: 65).
type DiagnosticsNodeToStringDeepOptions struct {
	PrefixLineOne       string
	PrefixOtherLines    *string
	ParentConfiguration *TextTreeConfiguration
	MinLevel            *DiagnosticLevel
	WrapWidth           *int
}

func parentConfig(opts []ToDescriptionOptions) *TextTreeConfiguration {
	if len(opts) > 0 {
		return opts[0].ParentConfiguration
	}
	return nil
}

// DiagnosticsNodeBase is the state and the default methods of Dart's
// abstract DiagnosticsNode. Embed it and call SetSelf with the outer node.
type DiagnosticsNodeBase struct {
	self          DiagnosticsNode
	name          *string
	style         DiagnosticsTreeStyle
	hasStyle      bool
	showName      bool
	showSeparator bool
	linePrefix    string
}

// DiagnosticsNodeOptions are DiagnosticsNode's constructor parameters
// (ShowName and ShowSeparator nil: true).
type DiagnosticsNodeOptions struct {
	Name          *string
	Style         *DiagnosticsTreeStyle
	ShowName      *bool
	ShowSeparator *bool
	LinePrefix    string
}

// NewDiagnosticsNodeBase is the DiagnosticsNode constructor, for types in
// other packages that embed DiagnosticsNodeBase (call SetSelf afterwards).
// A name must not end with a colon.
func NewDiagnosticsNodeBase(opts ...DiagnosticsNodeOptions) DiagnosticsNodeBase {
	var o DiagnosticsNodeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	var b DiagnosticsNodeBase
	b.init(o.Name, o.Style, boolOr(o.ShowName, true), boolOr(o.ShowSeparator, true), o.LinePrefix)
	return b
}

func (n *DiagnosticsNodeBase) init(name *string, style *DiagnosticsTreeStyle, showName, showSeparator bool, linePrefix string) {
	if name != nil && strings.HasSuffix(*name, ":") {
		// A name ending with ':' indicates that the user forgot that the ':' will
		// be automatically added for them when generating descriptions of the
		// property.
		panic("Names of diagnostic nodes must not end with colons.\nname:\n  \"" + *name + "\"")
	}
	n.name = name
	if style != nil {
		n.style, n.hasStyle = *style, true
	}
	n.showName = showName
	n.showSeparator = showSeparator
	n.linePrefix = linePrefix
}

// SetSelf points the base at the outer node, so the base's algorithms call
// the outer type's overrides.
func (n *DiagnosticsNodeBase) SetSelf(self DiagnosticsNode) { n.self = self }

func (n *DiagnosticsNodeBase) diagnosticsNodeBase() *DiagnosticsNodeBase { return n }

func (n *DiagnosticsNodeBase) outer() DiagnosticsNode {
	if n.self == nil {
		panic("foundation: DiagnosticsNodeBase used before SetSelf")
	}
	return n.self
}

// nodeName returns the node's name and whether it is non-null.
func nodeName(node DiagnosticsNode) (string, bool) {
	b := node.diagnosticsNodeBase()
	if b.name == nil {
		return "", false
	}
	return *b.name, true
}

// nameOrNull is '$name' in Dart: "null" for a null name.
func nameOrNull(node DiagnosticsNode) string {
	if name, ok := nodeName(node); ok {
		return name
	}
	return "null"
}

func (n *DiagnosticsNodeBase) Name() string {
	if n.name == nil {
		return ""
	}
	return *n.name
}

func (n *DiagnosticsNodeBase) ShowSeparator() bool { return n.showSeparator }
func (n *DiagnosticsNodeBase) ShowName() bool      { return n.showName }
func (n *DiagnosticsNodeBase) LinePrefix() string  { return n.linePrefix }

// Style is the node's display style (DiagnosticsTreeStyleNone when it was
// never given, where Dart's would be null).
func (n *DiagnosticsNodeBase) Style() DiagnosticsTreeStyle { return n.style }

// IsFiltered reports whether the node's level is below minLevel.
func (n *DiagnosticsNodeBase) IsFiltered(minLevel DiagnosticLevel) bool {
	return KReleaseMode || n.outer().Level() < minLevel
}

func (n *DiagnosticsNodeBase) Level() DiagnosticLevel {
	if KReleaseMode {
		return DiagnosticLevelHidden
	}
	return DiagnosticLevelInfo
}

func (n *DiagnosticsNodeBase) EmptyBodyDescription() *string { return nil }
func (n *DiagnosticsNodeBase) AllowWrap() bool               { return false }
func (n *DiagnosticsNodeBase) AllowNameWrap() bool           { return false }
func (n *DiagnosticsNodeBase) AllowTruncate() bool           { return false }

func (n *DiagnosticsNodeBase) separator() string {
	if n.outer().ShowSeparator() {
		return ":"
	}
	return ""
}

// ToTimelineArguments converts the properties to timeline event arguments
// (debug builds only; panics in profile builds, nil in release builds).
func (n *DiagnosticsNodeBase) ToTimelineArguments() map[string]string {
	if !KReleaseMode {
		// We don't throw in release builds, to avoid hurting users. We also don't do anything useful.
		if KProfileMode {
			panic(NewFlutterError(
				// Parts of this string are searched for verbatim by a test in dev/bots/test.dart.
				"DiagnosticsNode.toTimelineArguments used in non-debug build.\n" +
					"The DiagnosticsNode.toTimelineArguments API is expensive and causes timeline traces " +
					"to be non-representative. As such, it should not be used in profile builds. However, " +
					"this application is compiled in profile mode and yet still invoked the method."))
		}
		result := map[string]string{}
		for _, property := range n.outer().GetProperties() {
			if name, ok := nodeName(property); ok {
				result[name] = property.ToDescription(ToDescriptionOptions{ParentConfiguration: SingleLineTextConfiguration})
			}
		}
		return result
	}
	return nil
}

// runtimeTypeName is Dart's runtimeType.toString() for a node.
func runtimeTypeName(node any) string { return dartTypeName(reflect.TypeOf(node)) }

// ToJsonMap serializes the node as the delegate says (debug builds only;
// otherwise an empty map). Overrides add their keys to the base's map.
func (n *DiagnosticsNodeBase) ToJsonMap(delegate DiagnosticsSerializationDelegate) map[string]any {
	result := map[string]any{}
	if debugAssertsEnabled {
		self := n.outer()
		hasChildren := len(self.GetChildren()) != 0
		result["description"] = self.ToDescription()
		result["type"] = runtimeTypeName(self)
		if n.name != nil {
			result["name"] = *n.name
		}
		if !self.ShowSeparator() {
			result["showSeparator"] = false
		}
		if level := self.Level(); level != DiagnosticLevelInfo {
			result["level"] = level.String()
		}
		if !self.ShowName() {
			result["showName"] = false
		}
		if ebd := self.EmptyBodyDescription(); ebd != nil {
			result["emptyBodyDescription"] = *ebd
		}
		if style := self.Style(); style != DiagnosticsTreeStyleSparse {
			result["style"] = style.String()
		}
		if self.AllowTruncate() {
			result["allowTruncate"] = true
		}
		if hasChildren {
			result["hasChildren"] = true
		}
		if lp := self.LinePrefix(); lp != "" {
			result["linePrefix"] = lp
		}
		if !self.AllowWrap() {
			result["allowWrap"] = false
		}
		if self.AllowNameWrap() {
			result["allowNameWrap"] = true
		}
		for k, v := range delegate.AdditionalNodeProperties(self) {
			result[k] = v
		}
		if delegate.IncludeProperties() {
			result["properties"] = DiagnosticsNodeToJsonList(delegate.FilterProperties(self.GetProperties(), self), self, delegate)
		}
		if delegate.SubtreeDepth() > 0 {
			result["children"] = DiagnosticsNodeToJsonList(delegate.FilterChildren(self.GetChildren(), self), self, delegate)
		}
	}
	return result
}

type jsonifyItem struct {
	node     DiagnosticsNode
	callback func(map[string]any)
}

// ToJsonMapIterative serializes the node and its children without
// recursion and with fewer details (the widget inspector's summary tree).
func (n *DiagnosticsNodeBase) ToJsonMapIterative(delegate DiagnosticsSerializationDelegate) map[string]any {
	var childrenToJsonify []jsonifyItem
	result := map[string]any{}
	if debugAssertsEnabled {
		result = toJsonShallow(n.outer(), delegate, &childrenToJsonify)
		jsonifyNextNodesInStack(&childrenToJsonify, delegate)
	}
	return result
}

func jsonifyNextNodesInStack(toJsonify *[]jsonifyItem, delegate DiagnosticsSerializationDelegate) {
	for len(*toJsonify) != 0 {
		next := (*toJsonify)[0]
		*toJsonify = (*toJsonify)[1:]
		nodeAsJson := toJsonShallow(next.node, delegate, toJsonify)
		next.callback(nodeAsJson)
	}
}

// toJsonShallow is DiagnosticsNode._toJson.
func toJsonShallow(node DiagnosticsNode, delegate DiagnosticsSerializationDelegate, childrenToJsonify *[]jsonifyItem) map[string]any {
	result := map[string]any{}
	childrenJsonList := []map[string]any{}
	includeChildren := len(node.GetChildren()) != 0 && delegate.SubtreeDepth() > 0

	// Collect the children nodes to convert to JSON later.
	truncated := false
	if includeChildren {
		childrenNodes := delegate.FilterChildren(node.GetChildren(), node)
		originalNodeCount := len(childrenNodes)
		childrenNodes = delegate.TruncateNodesList(childrenNodes, node)
		if len(childrenNodes) != originalNodeCount {
			childrenNodes = append(slices.Clip(childrenNodes), NewDiagnosticsNodeMessage("..."))
			truncated = true
		}
		for _, child := range childrenNodes {
			*childrenToJsonify = append(*childrenToJsonify, jsonifyItem{child, func(jsonChild map[string]any) {
				childrenJsonList = append(childrenJsonList, jsonChild)
				result["children"] = childrenJsonList
			}})
		}
	}

	description := node.ToDescription()
	widgetRuntimeType, _, _ := strings.Cut(description, "-")
	if description == "[root]" {
		widgetRuntimeType = "RootWidget"
	}
	shouldIndent := node.Style() != DiagnosticsTreeStyleFlat && node.Style() != DiagnosticsTreeStyleError

	result["description"] = description
	result["shouldIndent"] = shouldIndent
	result["widgetRuntimeType"] = widgetRuntimeType
	if truncated {
		result["truncated"] = true
	}
	for k, v := range delegate.AdditionalNodeProperties(node, AdditionalNodePropertiesOptions{FullDetails: pFalse}) {
		result[k] = v
	}
	if includeChildren {
		result["children"] = childrenJsonList
	}
	return result
}

// DiagnosticsNodeToJsonList is DiagnosticsNode.toJsonList: serializes nodes
// (properties or children of parent) as the delegate says.
func DiagnosticsNodeToJsonList(nodes []DiagnosticsNode, parent DiagnosticsNode, delegate DiagnosticsSerializationDelegate) []map[string]any {
	truncated := false
	if nodes == nil {
		return []map[string]any{}
	}
	originalNodeCount := len(nodes)
	nodes = delegate.TruncateNodesList(nodes, parent)
	if len(nodes) != originalNodeCount {
		nodes = append(slices.Clip(nodes), NewDiagnosticsNodeMessage("..."))
		truncated = true
	}
	json := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		json[i] = node.ToJsonMap(delegate.DelegateForNode(node))
	}
	if truncated {
		json[len(json)-1]["truncated"] = true
	}
	return json
}

// String is toString().
func (n *DiagnosticsNodeBase) String() string { return n.outer().ToString() }

// ToString describes the node (with its properties when single line),
// compatible with the parent's style. In non-debug builds it is Dart's
// default "Instance of '...'".
func (n *DiagnosticsNodeBase) ToString(opts ...DiagnosticsNodeToStringOptions) string {
	var o DiagnosticsNodeToStringOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	self := n.outer()
	result := "Instance of '" + runtimeTypeName(self) + "'"
	if debugAssertsEnabled {
		if isSingleLine(self.Style()) {
			minLevel := levelOr(o.MinLevel, DiagnosticLevelInfo)
			result = self.ToStringDeep(DiagnosticsNodeToStringDeepOptions{ParentConfiguration: o.ParentConfiguration, MinLevel: &minLevel})
		} else {
			description := self.ToDescription(ToDescriptionOptions{ParentConfiguration: o.ParentConfiguration})
			name, hasName := nodeName(self)
			if !hasName || name == "" || !self.ShowName() {
				result = description
			} else if strings.Contains(description, "\n") {
				result = name + n.separator() + "\n" + description
			} else {
				result = name + n.separator() + " " + description
			}
		}
	}
	return result
}

// TextTreeConfiguration is the text art configuration for the node's style
// (nil for DiagnosticsTreeStyleNone).
func (n *DiagnosticsNodeBase) TextTreeConfiguration() *TextTreeConfiguration {
	switch n.outer().Style() {
	case DiagnosticsTreeStyleNone:
		return nil
	case DiagnosticsTreeStyleDense:
		return DenseTextConfiguration
	case DiagnosticsTreeStyleSparse:
		return SparseTextConfiguration
	case DiagnosticsTreeStyleOffstage:
		return DashedTextConfiguration
	case DiagnosticsTreeStyleWhitespace:
		return WhitespaceTextConfiguration
	case DiagnosticsTreeStyleTransition:
		return TransitionTextConfiguration
	case DiagnosticsTreeStyleSingleLine:
		return SingleLineTextConfiguration
	case DiagnosticsTreeStyleErrorProperty:
		return ErrorPropertyTextConfiguration
	case DiagnosticsTreeStyleShallow:
		return ShallowTextConfiguration
	case DiagnosticsTreeStyleError:
		return ErrorTextConfiguration
	case DiagnosticsTreeStyleFlat:
		return FlatTextConfiguration
	case DiagnosticsTreeStyleTruncateChildren:
		// Truncate children doesn't really need its own text style as the
		// rendering is quite custom.
		return WhitespaceTextConfiguration
	}
	return nil
}

// ToStringDeep renders the node and its descendants as text art (debug
// builds only).
func (n *DiagnosticsNodeBase) ToStringDeep(opts ...DiagnosticsNodeToStringDeepOptions) string {
	var o DiagnosticsNodeToStringDeepOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	result := ""
	if debugAssertsEnabled {
		minLevel := levelOr(o.MinLevel, DiagnosticLevelDebug)
		wrapWidth := intOr(o.WrapWidth, 65)
		result = NewTextTreeRenderer(TextTreeRendererOptions{MinLevel: &minLevel, WrapWidth: &wrapWidth}).
			Render(n.outer(), TextTreeRendererRenderOptions{
				PrefixLineOne:       o.PrefixLineOne,
				PrefixOtherLines:    o.PrefixOtherLines,
				ParentConfiguration: o.ParentConfiguration,
			})
	}
	return result
}

// DiagnosticsNodeMessageOptions are DiagnosticsNode.message's named
// parameters (nil: singleLine, info, true).
type DiagnosticsNodeMessageOptions struct {
	Style     *DiagnosticsTreeStyle
	Level     *DiagnosticLevel
	AllowWrap *bool
}

// NewDiagnosticsNodeMessage is DiagnosticsNode.message: a node holding
// just a message (a DiagnosticsProperty<void> with an empty name).
func NewDiagnosticsNodeMessage(message string, opts ...DiagnosticsNodeMessageOptions) DiagnosticsNode {
	var o DiagnosticsNodeMessageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	style := styleOr(o.Style, DiagnosticsTreeStyleSingleLine)
	level := levelOr(o.Level, DiagnosticLevelInfo)
	p := &DiagnosticsProperty[*struct{}]{}
	p.initProperty(p, strp(""), nil, propertyArgs{
		description:   &message,
		style:         &style,
		showName:      pFalse,
		allowWrap:     o.AllowWrap,
		level:         &level,
		defaultValue:  KNoDefaultValue,
		valueComputed: true,
	})
	return p
}

// DiagnosticsSerializationDelegate configures how a tree of DiagnosticsNode
// is serialized to JSON.
type DiagnosticsSerializationDelegate interface {
	AdditionalNodeProperties(node DiagnosticsNode, opts ...AdditionalNodePropertiesOptions) map[string]any
	FilterChildren(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode
	FilterProperties(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode
	TruncateNodesList(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode
	DelegateForNode(node DiagnosticsNode) DiagnosticsSerializationDelegate
	SubtreeDepth() int
	IncludeProperties() bool
	ExpandPropertyValues() bool
	CopyWith(opts ...DiagnosticsSerializationDelegateCopyWithOptions) DiagnosticsSerializationDelegate
}

// AdditionalNodePropertiesOptions are additionalNodeProperties' named
// parameters (FullDetails nil: true).
type AdditionalNodePropertiesOptions struct{ FullDetails *bool }

// DiagnosticsSerializationDelegateCopyWithOptions are copyWith's named
// parameters (nil: keep the current value).
type DiagnosticsSerializationDelegateCopyWithOptions struct {
	SubtreeDepth      *int
	IncludeProperties *bool
}

// DiagnosticsSerializationDelegateOptions are the default delegate's
// parameters.
type DiagnosticsSerializationDelegateOptions struct {
	SubtreeDepth      int
	IncludeProperties bool
}

// NewDiagnosticsSerializationDelegate is DiagnosticsSerializationDelegate(
// {subtreeDepth, includeProperties}): the default delegate.
func NewDiagnosticsSerializationDelegate(opts ...DiagnosticsSerializationDelegateOptions) DiagnosticsSerializationDelegate {
	var o DiagnosticsSerializationDelegateOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return defaultDiagnosticsSerializationDelegate{includeProperties: o.IncludeProperties, subtreeDepth: o.SubtreeDepth}
}

type defaultDiagnosticsSerializationDelegate struct {
	includeProperties bool
	subtreeDepth      int
}

func (d defaultDiagnosticsSerializationDelegate) AdditionalNodeProperties(DiagnosticsNode, ...AdditionalNodePropertiesOptions) map[string]any {
	return map[string]any{}
}

func (d defaultDiagnosticsSerializationDelegate) DelegateForNode(DiagnosticsNode) DiagnosticsSerializationDelegate {
	if d.subtreeDepth > 0 {
		depth := d.subtreeDepth - 1
		return d.CopyWith(DiagnosticsSerializationDelegateCopyWithOptions{SubtreeDepth: &depth})
	}
	return d
}

func (d defaultDiagnosticsSerializationDelegate) ExpandPropertyValues() bool { return false }

func (d defaultDiagnosticsSerializationDelegate) FilterChildren(nodes []DiagnosticsNode, _ DiagnosticsNode) []DiagnosticsNode {
	return nodes
}

func (d defaultDiagnosticsSerializationDelegate) FilterProperties(nodes []DiagnosticsNode, _ DiagnosticsNode) []DiagnosticsNode {
	return nodes
}

func (d defaultDiagnosticsSerializationDelegate) IncludeProperties() bool { return d.includeProperties }
func (d defaultDiagnosticsSerializationDelegate) SubtreeDepth() int       { return d.subtreeDepth }

func (d defaultDiagnosticsSerializationDelegate) TruncateNodesList(nodes []DiagnosticsNode, _ DiagnosticsNode) []DiagnosticsNode {
	return nodes
}

func (d defaultDiagnosticsSerializationDelegate) CopyWith(opts ...DiagnosticsSerializationDelegateCopyWithOptions) DiagnosticsSerializationDelegate {
	c := d
	for _, o := range opts {
		if o.SubtreeDepth != nil {
			c.subtreeDepth = *o.SubtreeDepth
		}
		if o.IncludeProperties != nil {
			c.includeProperties = *o.IncludeProperties
		}
	}
	return c
}

// DiagnosticsBlock is a container for other diagnostics, typically without
// a meaningful value of its own (used for nested error messages).
type DiagnosticsBlock struct {
	DiagnosticsNodeBase
	children      []DiagnosticsNode
	properties    []DiagnosticsNode
	level         DiagnosticLevel
	description   string
	value         any
	allowTruncate bool
}

// DiagnosticsBlockOptions are DiagnosticsBlock's named parameters (nil:
// style whitespace, showName and showSeparator true, level info).
type DiagnosticsBlockOptions struct {
	Name          *string
	Style         *DiagnosticsTreeStyle
	ShowName      *bool
	ShowSeparator *bool
	LinePrefix    string
	Value         any
	Description   string
	Level         *DiagnosticLevel
	AllowTruncate bool
	Children      []DiagnosticsNode
	Properties    []DiagnosticsNode
}

// NewDiagnosticsBlock is DiagnosticsBlock({...}).
func NewDiagnosticsBlock(opts ...DiagnosticsBlockOptions) *DiagnosticsBlock {
	var o DiagnosticsBlockOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	b := &DiagnosticsBlock{
		children:      o.Children,
		properties:    o.Properties,
		level:         levelOr(o.Level, DiagnosticLevelInfo),
		description:   o.Description,
		value:         o.Value,
		allowTruncate: o.AllowTruncate,
	}
	if b.children == nil {
		b.children = []DiagnosticsNode{}
	}
	if b.properties == nil {
		b.properties = []DiagnosticsNode{}
	}
	style := styleOr(o.Style, DiagnosticsTreeStyleWhitespace)
	b.init(o.Name, &style, boolOr(o.ShowName, true) && o.Name != nil, boolOr(o.ShowSeparator, true), o.LinePrefix)
	b.SetSelf(b)
	return b
}

func (b *DiagnosticsBlock) Level() DiagnosticLevel           { return b.level }
func (b *DiagnosticsBlock) Value() any                       { return b.value }
func (b *DiagnosticsBlock) AllowTruncate() bool              { return b.allowTruncate }
func (b *DiagnosticsBlock) GetChildren() []DiagnosticsNode   { return b.children }
func (b *DiagnosticsBlock) GetProperties() []DiagnosticsNode { return b.properties }
func (b *DiagnosticsBlock) ToDescription(...ToDescriptionOptions) string {
	return b.description
}
