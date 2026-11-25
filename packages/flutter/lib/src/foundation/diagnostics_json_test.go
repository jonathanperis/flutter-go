package foundation

// Port of packages/flutter/test/foundation/diagnostics_json_test.dart
// (Flutter 3.50.1), which tests diagnostics.dart's JSON serialization.
//
// Not ported: "Element diagnostics json includes widgetRuntimeType" and
// "StatefulElement diagnostics are stateful" — they need widgets' Element,
// StatefulElement and Placeholder, which aren't ported to the Go widgets
// package; they belong with framework.dart.

import "testing"

type nodeDelegator func(node DiagnosticsNode, delegate testDiagnosticsSerializationDelegate) DiagnosticsSerializationDelegate
type nodeTruncator func(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode
type nodeFilter func(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode

// testDiagnosticsSerializationDelegate is the Dart test's
// TestDiagnosticsSerializationDelegate.
type testDiagnosticsSerializationDelegate struct {
	includeProperties           bool
	subtreeDepth                int
	additionalNodePropertiesMap map[string]any
	childFilter                 nodeFilter
	propertyFilter              nodeFilter
	nodeTruncator               nodeTruncator
	nodeDelegator               nodeDelegator
}

func (d testDiagnosticsSerializationDelegate) AdditionalNodeProperties(DiagnosticsNode, ...AdditionalNodePropertiesOptions) map[string]any {
	return d.additionalNodePropertiesMap
}

func (d testDiagnosticsSerializationDelegate) DelegateForNode(node DiagnosticsNode) DiagnosticsSerializationDelegate {
	if d.nodeDelegator != nil {
		return d.nodeDelegator(node, d)
	}
	if d.subtreeDepth > 0 {
		depth := d.subtreeDepth - 1
		return d.CopyWith(DiagnosticsSerializationDelegateCopyWithOptions{SubtreeDepth: &depth})
	}
	return d
}

func (d testDiagnosticsSerializationDelegate) ExpandPropertyValues() bool { return false }

func (d testDiagnosticsSerializationDelegate) FilterChildren(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
	if d.childFilter != nil {
		return d.childFilter(nodes, owner)
	}
	return nodes
}

func (d testDiagnosticsSerializationDelegate) FilterProperties(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
	if d.propertyFilter != nil {
		return d.propertyFilter(nodes, owner)
	}
	return nodes
}

func (d testDiagnosticsSerializationDelegate) IncludeProperties() bool { return d.includeProperties }
func (d testDiagnosticsSerializationDelegate) SubtreeDepth() int       { return d.subtreeDepth }

func (d testDiagnosticsSerializationDelegate) TruncateNodesList(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
	if d.nodeTruncator != nil {
		return d.nodeTruncator(nodes, owner)
	}
	return nodes
}

func (d testDiagnosticsSerializationDelegate) CopyWith(opts ...DiagnosticsSerializationDelegateCopyWithOptions) DiagnosticsSerializationDelegate {
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

func serializationTestTree() *TestTree {
	return newTestTree(tt{
		Properties: []DiagnosticsNode{
			NewStringProperty("stringProperty1", "value1", StringPropertyOptions{Quoted: ptr(false)}),
			NewDoubleProperty("doubleProperty1", 42.5),
			NewDoubleProperty("roundedProperty", 1.0/3.0),
			NewStringProperty("DO_NOT_SHOW", "DO_NOT_SHOW", StringPropertyOptions{Level: lv(DiagnosticLevelHidden), Quoted: ptr(false)}),
			NewDiagnosticsProperty[any]("DO_NOT_SHOW_NULL", nil, DiagnosticsPropertyOptions{DefaultValue: Null}),
			NewDiagnosticsProperty[any]("nullProperty", nil),
			NewStringProperty("node_type", "<root node>", StringPropertyOptions{ShowName: ptr(false), Quoted: ptr(false)}),
		},
		Children: []*TestTree{
			newTestTree(tt{Name: "node A"}),
			newTestTree(tt{
				Name: "node B",
				Properties: []DiagnosticsNode{
					NewStringProperty("p1", "v1", StringPropertyOptions{Quoted: ptr(false)}),
					NewStringProperty("p2", "v2", StringPropertyOptions{Quoted: ptr(false)}),
				},
				Children: []*TestTree{
					newTestTree(tt{Name: "node B1"}),
					newTestTree(tt{
						Name:       "node B2",
						Properties: []DiagnosticsNode{NewStringProperty("property1", "value1", StringPropertyOptions{Quoted: ptr(false)})},
					}),
					newTestTree(tt{
						Name: "node B3",
						Properties: []DiagnosticsNode{
							NewStringProperty("node_type", "<leaf node>", StringPropertyOptions{ShowName: ptr(false), Quoted: ptr(false)}),
							NewIntProperty("foo", 42),
						},
					}),
				},
			}),
			newTestTree(tt{
				Name:       "node C",
				Properties: []DiagnosticsNode{NewStringProperty("foo", "multi\nline\nvalue!", StringPropertyOptions{Quoted: ptr(false)})},
			}),
		},
	})
}

// These are always included.
var defaultDiagnosticKeys = []string{"description"}

// These are only included when fullDetails = false.
var essentialDiagnosticKeys = []string{"shouldIndent"}

// These are only included with fullDetails = true.
var detailedDiagnosticKeys = []string{"type", "hasChildren", "allowWrap"}

func hasKey(m map[string]any, k string) bool { _, ok := m[k]; return ok }

func jsonList(t *testing.T, v any) []map[string]any {
	t.Helper()
	l, ok := v.([]map[string]any)
	if !ok {
		t.Fatalf("not a JSON list: %#v", v)
	}
	return l
}

func TestSerializationDefault(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(NewDiagnosticsSerializationDelegate())
	equal(t, hasKey(result, "properties"), false)
	equal(t, hasKey(result, "children"), false)
	for _, k := range defaultDiagnosticKeys {
		equal(t, hasKey(result, k), true)
	}
	for _, k := range detailedDiagnosticKeys {
		equal(t, hasKey(result, k), true)
	}
}

func TestSerializationIterativeWithoutFullDetails(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMapIterative(NewDiagnosticsSerializationDelegate())
	equal(t, hasKey(result, "properties"), false)
	equal(t, hasKey(result, "children"), false)
	for _, k := range defaultDiagnosticKeys {
		equal(t, hasKey(result, k), true)
	}
	for _, k := range essentialDiagnosticKeys {
		equal(t, hasKey(result, k), true)
	}
	for _, k := range detailedDiagnosticKeys {
		equal(t, hasKey(result, k), false)
	}
	// The truncated value should not be included if it is false.
	equal(t, result["truncated"] == nil || result["truncated"] == true, true)
}

func TestSerializationSubtreeDepth1(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(NewDiagnosticsSerializationDelegate(DiagnosticsSerializationDelegateOptions{SubtreeDepth: 1}))
	equal(t, hasKey(result, "properties"), false)
	children := jsonList(t, result["children"])
	equal(t, hasKey(children[0], "children"), false)
	equal(t, hasKey(children[1], "children"), false)
	equal(t, hasKey(children[2], "children"), false)
}

func TestSerializationSubtreeDepth5(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(NewDiagnosticsSerializationDelegate(DiagnosticsSerializationDelegateOptions{SubtreeDepth: 5}))
	equal(t, hasKey(result, "properties"), false)
	children := jsonList(t, result["children"])
	equal(t, len(jsonList(t, children[0]["children"])), 0)
	equal(t, len(jsonList(t, children[1]["children"])), 3)
	equal(t, len(jsonList(t, children[2]["children"])), 0)
}

func TestSerializationIncludeProperties(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(NewDiagnosticsSerializationDelegate(DiagnosticsSerializationDelegateOptions{IncludeProperties: true}))
	equal(t, hasKey(result, "children"), false)
	equal(t, len(jsonList(t, result["properties"])), 7)
}

func TestSerializationIncludePropertiesWithSubtreeDepth1(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(NewDiagnosticsSerializationDelegate(DiagnosticsSerializationDelegateOptions{IncludeProperties: true, SubtreeDepth: 1}))
	equal(t, len(jsonList(t, result["properties"])), 7)
	children := jsonList(t, result["children"])
	equal(t, len(children), 3)
	equal(t, len(jsonList(t, children[0]["properties"])), 0)
	equal(t, len(jsonList(t, children[1]["properties"])), 2)
	equal(t, len(jsonList(t, children[2]["properties"])), 1)
}

func TestSerializationAdditionalNodeProperties(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(testDiagnosticsSerializationDelegate{
		includeProperties:           true,
		subtreeDepth:                1,
		additionalNodePropertiesMap: map[string]any{"foo": true},
	})
	equal(t, result["foo"], any(true))
	properties := jsonList(t, result["properties"])
	equal(t, len(properties), 7)
	for _, p := range properties {
		equal(t, p["foo"], any(true))
	}
	children := jsonList(t, result["children"])
	equal(t, len(children), 3)
	for _, c := range children {
		equal(t, c["foo"], any(true))
	}
}

func TestSerializationFilterPropertiesSublist(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(testDiagnosticsSerializationDelegate{
		includeProperties: true,
		propertyFilter: func(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
			var out []DiagnosticsNode
			for _, n := range nodes {
				if _, ok := n.(*StringProperty); ok {
					out = append(out, n)
				}
			}
			return out
		},
	})
	properties := jsonList(t, result["properties"])
	equal(t, len(properties), 3)
	for _, p := range properties {
		equal(t, p["type"], any("StringProperty"))
	}
}

func TestSerializationFilterPropertiesReplace(t *testing.T) {
	replaced := false
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(testDiagnosticsSerializationDelegate{
		includeProperties: true,
		propertyFilter: func(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
			if replaced {
				return nodes
			}
			replaced = true
			return []DiagnosticsNode{NewStringProperty("foo", "bar")}
		},
	})
	properties := jsonList(t, result["properties"])
	equal(t, len(properties), 1)
	equal(t, properties[0]["name"], any("foo"))
}

func TestSerializationFilterChildrenSublist(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(testDiagnosticsSerializationDelegate{
		subtreeDepth: 1,
		childFilter: func(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
			var out []DiagnosticsNode
			for _, n := range nodes {
				if len(n.GetProperties()) == 0 {
					out = append(out, n)
				}
			}
			return out
		},
	})
	equal(t, len(jsonList(t, result["children"])), 1)
}

func TestSerializationFilterChildrenReplace(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(testDiagnosticsSerializationDelegate{
		subtreeDepth: 1,
		childFilter: func(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
			var out []DiagnosticsNode
			for _, n := range nodes {
				out = append(out, n.GetChildren()...)
			}
			return out
		},
	})
	children := jsonList(t, result["children"])
	equal(t, len(children), 3)
	equal(t, children[0]["name"], any("child node B1"))
}

func TestSerializationNodeTruncator(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(testDiagnosticsSerializationDelegate{
		subtreeDepth:      5,
		includeProperties: true,
		nodeTruncator: func(nodes []DiagnosticsNode, owner DiagnosticsNode) []DiagnosticsNode {
			return nodes[:min(2, len(nodes))]
		},
	})
	children := jsonList(t, result["children"])
	equal(t, len(children), 3)
	equal(t, children[len(children)-1]["truncated"], any(true))

	properties := jsonList(t, result["properties"])
	equal(t, len(properties), 3)
	equal(t, properties[len(properties)-1]["truncated"], any(true))
}

func TestSerializationDelegateForAddingNodes(t *testing.T) {
	result := serializationTestTree().ToDiagnosticsNode().ToJsonMap(testDiagnosticsSerializationDelegate{
		subtreeDepth:      5,
		includeProperties: true,
		nodeDelegator: func(node DiagnosticsNode, delegate testDiagnosticsSerializationDelegate) DiagnosticsSerializationDelegate {
			return delegate.CopyWith(DiagnosticsSerializationDelegateCopyWithOptions{IncludeProperties: ptr(false)})
		},
	})
	properties := jsonList(t, result["properties"])
	equal(t, len(properties), 7)
	for _, p := range properties {
		equal(t, hasKey(p, "properties"), false)
	}
	children := jsonList(t, result["children"])
	equal(t, len(children), 3)
	for _, c := range children {
		equal(t, hasKey(c, "properties"), false)
	}
}
