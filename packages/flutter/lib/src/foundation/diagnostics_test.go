package foundation

// Port of packages/flutter/test/foundation/diagnostics_test.dart (Flutter
// 3.50.1), which tests diagnostics.dart. Goldens are copied from the Dart
// test verbatim.
//
// "Stack trace test" (DiagnosticsStackTrace, from assertions.dart) is
// TestStackTrace.
//
// Not ported:
//   - `present.value equals onClick` (closure identity): Go funcs aren't
//     comparable; the test checks the value is the non-nil func instead.

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func ptr[T any](v T) *T { return &v }

func st(s DiagnosticsTreeStyle) *DiagnosticsTreeStyle { return &s }
func lv(l DiagnosticLevel) *DiagnosticLevel           { return &l }

// TestTree is the Dart test's TestTree (a DiagnosticableTreeMixin).
type TestTree struct {
	DiagnosticableTreeMixin
	name       string
	style      *DiagnosticsTreeStyle
	children   []*TestTree
	properties []DiagnosticsNode
}

type tt struct {
	Name       string
	Style      *DiagnosticsTreeStyle
	Children   []*TestTree
	Properties []DiagnosticsNode
}

func newTestTree(a tt) *TestTree {
	t := &TestTree{name: a.Name, style: a.Style, children: a.Children, properties: a.Properties}
	t.SetSelf(t)
	return t
}

func (t *TestTree) DebugDescribeChildren() []DiagnosticsNode {
	out := []DiagnosticsNode{}
	for _, child := range t.children {
		out = append(out, child.ToDiagnosticsNode(ToDiagnosticsNodeOptions{Name: ptr("child " + child.name), Style: child.style}))
	}
	return out
}

func (t *TestTree) DebugFillProperties(properties *DiagnosticPropertiesBuilder) {
	t.DiagnosticableTreeMixin.DebugFillProperties(properties)
	if t.style != nil {
		properties.DefaultDiagnosticsTreeStyle = *t.style
	}
	for _, p := range t.properties {
		properties.Add(p)
	}
}

// ExampleEnum is the Dart test's enum ExampleEnum { hello, world, deferToChild }.
type ExampleEnum int

const (
	ExampleEnumHello ExampleEnum = iota
	ExampleEnumWorld
	ExampleEnumDeferToChild
)

func (e ExampleEnum) String() string { return [...]string{"hello", "world", "deferToChild"}[e] }

var hashCodeRE = regexp.MustCompile(`#[\da-fA-F]{5}`)

func equalsIgnoringHashCodes(t *testing.T, got, want, reason string) {
	t.Helper()
	got = hashCodeRE.ReplaceAllString(got, "#00000")
	if got != want {
		t.Errorf("%s:\n got:\n%s\nwant:\n%s", reason, got, want)
	}
}

func equal[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func isAllTreeConnectorCharacters(line string) bool {
	for _, c := range line {
		if !(c == ' ' || c == '\t' || strings.ContainsRune("│╎║╏┃┊┋┆┇", c)) {
			return false
		}
	}
	return true
}

// hasAGoodToStringDeep is flutter_test's matcher of the same name.
func hasAGoodToStringDeep(t *testing.T, object DiagnosticableTree) {
	t.Helper()
	var issues []string
	description := object.ToStringDeep()
	if strings.HasSuffix(description, "\n") {
		description = description[:len(description)-1]
	} else {
		issues = append(issues, "Not terminated with a line break.")
	}
	if strings.TrimSpace(description) != description {
		issues = append(issues, "Has trailing whitespace.")
	}
	lines := strings.Split(description, "\n")
	if len(lines) < 2 {
		issues = append(issues, "Does not have multiple lines.")
	}
	if strings.Contains(description, "Instance of ") {
		issues = append(issues, `Contains text "Instance of ".`)
	}
	for i, line := range lines {
		if line == "" {
			issues = append(issues, "Line is empty: "+string(rune('1'+i)))
		}
		if dartTrimRight(line) != line {
			issues = append(issues, "Line has trailing whitespace.")
		}
	}
	if isAllTreeConnectorCharacters(lines[len(lines)-1]) {
		issues = append(issues, "Last line is all tree connector characters.")
	}
	const prefixLineOne = "PREFIX_LINE_ONE____"
	const prefixOtherLines = "PREFIX_OTHER_LINES_"
	withPrefixes := object.ToStringDeep(DiagnosticableTreeToStringDeepOptions{PrefixLineOne: prefixLineOne, PrefixOtherLines: ptr(prefixOtherLines)})
	withPrefixes = strings.TrimSuffix(withPrefixes, "\n")
	prefixedLines := strings.Split(withPrefixes, "\n")
	if !strings.HasPrefix(prefixedLines[0], prefixLineOne) {
		issues = append(issues, "First line does not contain expected prefix.")
	}
	for _, line := range prefixedLines[1:] {
		if !strings.HasPrefix(line, prefixOtherLines) {
			issues = append(issues, "Line does not contain the expected prefix: "+line)
		}
	}
	if len(issues) > 0 {
		t.Errorf("Bad toStringDeep():\n%s\n%s", description, strings.Join(issues, "\n"))
	}
}

// simulateJsonSerialization encodes and decodes toJsonMap, to check it is
// valid JSON.
func simulateJsonSerialization(t *testing.T, node DiagnosticsNode) map[string]any {
	t.Helper()
	data, err := json.Marshal(node.ToJsonMap(NewDiagnosticsSerializationDelegate()))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func orDefault(v any, def any) any {
	if v == nil {
		return def
	}
	return v
}

func validateNodeJsonSerialization(t *testing.T, node DiagnosticsNode) {
	t.Helper()
	validateNodeJsonSerializationHelper(t, simulateJsonSerialization(t, node), node)
}

func validateNodeJsonSerializationHelper(t *testing.T, j map[string]any, node DiagnosticsNode) {
	t.Helper()
	if name, ok := nodeName(node); ok {
		equal[any](t, j["name"], name)
	} else {
		equal(t, j["name"], nil)
	}
	equal(t, orDefault(j["showSeparator"], true), any(node.ShowSeparator()))
	equal(t, j["description"], any(node.ToDescription()))
	equal(t, orDefault(j["level"], DiagnosticLevelInfo.String()), any(node.Level().String()))
	equal(t, orDefault(j["showName"], true), any(node.ShowName()))
	if ebd := node.EmptyBodyDescription(); ebd != nil {
		equal[any](t, j["emptyBodyDescription"], *ebd)
	} else {
		equal(t, j["emptyBodyDescription"], nil)
	}
	equal(t, orDefault(j["style"], DiagnosticsTreeStyleSparse.String()), any(node.Style().String()))
	equal(t, j["type"], any(runtimeTypeName(node)))
	equal(t, orDefault(j["hasChildren"], false), any(len(node.GetChildren()) != 0))
}

func validatePropertyJsonSerialization[T any](t *testing.T, p *DiagnosticsProperty[T]) {
	t.Helper()
	validatePropertyJsonSerializationHelper(t, simulateJsonSerialization(t, p.outer()), p)
}

func validateStringPropertyJsonSerialization(t *testing.T, p *StringProperty) {
	t.Helper()
	j := simulateJsonSerialization(t, p)
	equal(t, j["quoted"], any(p.Quoted))
	validatePropertyJsonSerializationHelper(t, j, &p.DiagnosticsProperty)
}

func validateFlagPropertyJsonSerialization(t *testing.T, p *FlagProperty) {
	t.Helper()
	j := simulateJsonSerialization(t, p)
	if p.IfTrue != nil {
		equal[any](t, j["ifTrue"], *p.IfTrue)
	} else {
		_, has := j["ifTrue"]
		equal(t, has, false)
	}
	if p.IfFalse != nil {
		equal[any](t, j["ifFalse"], *p.IfFalse)
	} else {
		_, has := j["isFalse"]
		equal(t, has, false)
	}
	validatePropertyJsonSerializationHelper(t, j, &p.DiagnosticsProperty)
}

func validateDoublePropertyJsonSerialization(t *testing.T, p *DoubleProperty) {
	t.Helper()
	j := simulateJsonSerialization(t, p.outer())
	if p.Unit != nil {
		equal[any](t, j["unit"], *p.Unit)
	} else {
		_, has := j["unit"]
		equal(t, has, false)
	}
	equal[any](t, j["numberToString"], p.outer().(numberToStringer).NumberToString())
	validatePropertyJsonSerializationHelper(t, j, &p.DiagnosticsProperty)
}

func validateObjectFlagPropertyJsonSerialization[T any](t *testing.T, p *ObjectFlagProperty[T]) {
	t.Helper()
	j := simulateJsonSerialization(t, p)
	if p.IfPresent != nil {
		equal[any](t, j["ifPresent"], *p.IfPresent)
	} else {
		_, has := j["ifPresent"]
		equal(t, has, false)
	}
	validatePropertyJsonSerializationHelper(t, j, &p.DiagnosticsProperty)
}

func validateIterableFlagsPropertyJsonSerialization[T any](t *testing.T, p *FlagsSummary[T]) {
	t.Helper()
	j := simulateJsonSerialization(t, p)
	if len(p.TypedValue()) != 0 {
		var want []any
		for _, e := range p.TypedValue() {
			if !isNull(e.Value) {
				want = append(want, e.Key)
			}
		}
		if want == nil {
			want = []any{}
		}
		if !reflect.DeepEqual(j["values"], want) {
			t.Errorf("values: got %v, want %v", j["values"], want)
		}
	} else {
		_, has := j["values"]
		equal(t, has, false)
	}
	validatePropertyJsonSerializationHelper(t, j, &p.DiagnosticsProperty)
}

func validateIterablePropertyJsonSerialization[T any](t *testing.T, p *IterableProperty[T]) {
	t.Helper()
	j := simulateJsonSerialization(t, p)
	if p.TypedValue() != nil {
		want := []any{}
		for _, v := range p.TypedValue() {
			want = append(want, dartToString(v))
		}
		if !reflect.DeepEqual(j["values"], want) {
			t.Errorf("values: got %v, want %v", j["values"], want)
		}
	} else {
		_, has := j["values"]
		equal(t, has, false)
	}
	validatePropertyJsonSerializationHelper(t, j, &p.DiagnosticsProperty)
}

func validatePropertyJsonSerializationHelper[T any](t *testing.T, j map[string]any, p *DiagnosticsProperty[T]) {
	t.Helper()
	if _, none := p.DefaultValue.(noDefaultValue); !none {
		equal[any](t, j["defaultValue"], dartToString(p.DefaultValue))
	} else {
		_, has := j["defaultValue"]
		equal(t, has, false)
	}
	checkOptional := func(key string, v *string) {
		if v != nil {
			equal[any](t, j[key], *v)
		} else {
			_, has := j[key]
			equal(t, has, false)
		}
	}
	checkOptional("ifEmpty", p.IfEmpty)
	checkOptional("ifNull", p.IfNull)
	checkOptional("tooltip", p.Tooltip)
	equal[any](t, j["missingIfNull"], p.MissingIfNull)
	if ex := p.Exception(); ex != nil {
		equal[any](t, j["exception"], dartToString(ex))
	} else {
		_, has := j["exception"]
		equal(t, has, false)
	}
	equal[any](t, j["propertyType"], p.propertyTypeString())
	_, has := j["defaultLevel"]
	equal(t, has, true)
	if _, ok := p.Value().(Diagnosticable); ok {
		equal[any](t, j["isDiagnosticableValue"], true)
	} else {
		_, has := j["isDiagnosticableValue"]
		equal(t, has, false)
	}
	validateNodeJsonSerializationHelper(t, j, p.outer())
}

func TestTreeDiagnosticsMixinControl(t *testing.T) {
	goldenStyleTest := func(description string, style, lastChildStyle *DiagnosticsTreeStyle, golden string) {
		t.Helper()
		last := style
		if lastChildStyle != nil {
			last = lastChildStyle
		}
		tree := newTestTree(tt{
			Children: []*TestTree{
				newTestTree(tt{Name: "node A", Style: style}),
				newTestTree(tt{
					Name: "node B",
					Children: []*TestTree{
						newTestTree(tt{Name: "node B1", Style: style}),
						newTestTree(tt{Name: "node B2", Style: style}),
						newTestTree(tt{Name: "node B3", Style: last}),
					},
					Style: style,
				}),
				newTestTree(tt{Name: "node C", Style: last}),
			},
			Style: lastChildStyle,
		})
		hasAGoodToStringDeep(t, tree)
		equalsIgnoringHashCodes(t, tree.ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: style}).ToStringDeep(), golden, description)
		validateNodeJsonSerialization(t, tree.ToDiagnosticsNode())
	}

	goldenStyleTest("dense", st(DiagnosticsTreeStyleDense), nil,
		"TestTree#00000\n"+
			"├child node A: TestTree#00000\n"+
			"├child node B: TestTree#00000\n"+
			"│├child node B1: TestTree#00000\n"+
			"│├child node B2: TestTree#00000\n"+
			"│└child node B3: TestTree#00000\n"+
			"└child node C: TestTree#00000\n")
	goldenStyleTest("sparse", st(DiagnosticsTreeStyleSparse), nil,
		"TestTree#00000\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ └─child node B3: TestTree#00000\n"+
			" └─child node C: TestTree#00000\n")
	goldenStyleTest("dashed", st(DiagnosticsTreeStyleOffstage), nil,
		"TestTree#00000\n"+
			" ╎╌child node A: TestTree#00000\n"+
			" ╎╌child node B: TestTree#00000\n"+
			" ╎ ╎╌child node B1: TestTree#00000\n"+
			" ╎ ╎╌child node B2: TestTree#00000\n"+
			" ╎ └╌child node B3: TestTree#00000\n"+
			" └╌child node C: TestTree#00000\n")
	goldenStyleTest("leaf children", st(DiagnosticsTreeStyleSparse), st(DiagnosticsTreeStyleTransition),
		"TestTree#00000\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ ╘═╦══ child node B3 ═══\n"+
			" │   ║ TestTree#00000\n"+
			" │   ╚═══════════\n"+
			" ╘═╦══ child node C ═══\n"+
			"   ║ TestTree#00000\n"+
			"   ╚═══════════\n")
	goldenStyleTest("error children", st(DiagnosticsTreeStyleSparse), st(DiagnosticsTreeStyleError),
		"TestTree#00000\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ ╘═╦══╡ CHILD NODE B3: TESTTREE#00000 ╞═══════════════════════════════\n"+
			" │   ╚══════════════════════════════════════════════════════════════════\n"+
			" ╘═╦══╡ CHILD NODE C: TESTTREE#00000 ╞════════════════════════════════\n"+
			"   ╚══════════════════════════════════════════════════════════════════\n")
	// You would never really want to make everything a transition child like
	// this but you can and still get a readable tree.
	goldenStyleTest("transition", st(DiagnosticsTreeStyleTransition), nil,
		"TestTree#00000:\n"+
			"  ╞═╦══ child node A ═══\n"+
			"  │ ║ TestTree#00000\n"+
			"  │ ╚═══════════\n"+
			"  ╞═╦══ child node B ═══\n"+
			"  │ ║ TestTree#00000:\n"+
			"  │ ║   ╞═╦══ child node B1 ═══\n"+
			"  │ ║   │ ║ TestTree#00000\n"+
			"  │ ║   │ ╚═══════════\n"+
			"  │ ║   ╞═╦══ child node B2 ═══\n"+
			"  │ ║   │ ║ TestTree#00000\n"+
			"  │ ║   │ ╚═══════════\n"+
			"  │ ║   ╘═╦══ child node B3 ═══\n"+
			"  │ ║     ║ TestTree#00000\n"+
			"  │ ║     ╚═══════════\n"+
			"  │ ╚═══════════\n"+
			"  ╘═╦══ child node C ═══\n"+
			"    ║ TestTree#00000\n"+
			"    ╚═══════════\n")
	goldenStyleTest("error", st(DiagnosticsTreeStyleError), nil,
		"══╡ TESTTREE#00000 ╞═════════════════════════════════════════════\n"+
			"╞═╦══╡ CHILD NODE A: TESTTREE#00000 ╞════════════════════════════════\n"+
			"│ ╚══════════════════════════════════════════════════════════════════\n"+
			"╞═╦══╡ CHILD NODE B: TESTTREE#00000 ╞════════════════════════════════\n"+
			"│ ║ ╞═╦══╡ CHILD NODE B1: TESTTREE#00000 ╞═══════════════════════════════\n"+
			"│ ║ │ ╚══════════════════════════════════════════════════════════════════\n"+
			"│ ║ ╞═╦══╡ CHILD NODE B2: TESTTREE#00000 ╞═══════════════════════════════\n"+
			"│ ║ │ ╚══════════════════════════════════════════════════════════════════\n"+
			"│ ║ ╘═╦══╡ CHILD NODE B3: TESTTREE#00000 ╞═══════════════════════════════\n"+
			"│ ║   ╚══════════════════════════════════════════════════════════════════\n"+
			"│ ╚══════════════════════════════════════════════════════════════════\n"+
			"╘═╦══╡ CHILD NODE C: TESTTREE#00000 ╞════════════════════════════════\n"+
			"  ╚══════════════════════════════════════════════════════════════════\n"+
			"═════════════════════════════════════════════════════════════════\n")
	goldenStyleTest("whitespace", st(DiagnosticsTreeStyleWhitespace), nil,
		"TestTree#00000:\n"+
			"  child node A: TestTree#00000\n"+
			"  child node B: TestTree#00000:\n"+
			"    child node B1: TestTree#00000\n"+
			"    child node B2: TestTree#00000\n"+
			"    child node B3: TestTree#00000\n"+
			"  child node C: TestTree#00000\n")
	// Single line mode does not display children. (Dart's test also runs
	// hasAGoodToStringDeep here; the tree's own style is null, so its
	// toStringDeep is the sparse one.)
	goldenStyleTest("single line", st(DiagnosticsTreeStyleSingleLine), nil,
		"TestTree#00000")
}

func TestTreeDiagnosticsMixinTreeWithProperties(t *testing.T) {
	goldenStyleTest := func(description string, name *string, style, lastChildStyle *DiagnosticsTreeStyle, propertyStyleP *DiagnosticsTreeStyle, golden string) {
		t.Helper()
		propertyStyle := DiagnosticsTreeStyleSingleLine
		if propertyStyleP != nil {
			propertyStyle = *propertyStyleP
		}
		ps := &propertyStyle
		last := style
		if lastChildStyle != nil {
			last = lastChildStyle
		}
		tree := newTestTree(tt{
			Properties: []DiagnosticsNode{
				NewStringProperty("stringProperty1", "value1", StringPropertyOptions{Quoted: ptr(false), Style: ps}),
				NewDoubleProperty("doubleProperty1", 42.5, DoublePropertyOptions{Style: ps}),
				NewDoubleProperty("roundedProperty", 1.0/3.0, DoublePropertyOptions{Style: ps}),
				NewStringProperty("DO_NOT_SHOW", "DO_NOT_SHOW", StringPropertyOptions{Level: lv(DiagnosticLevelHidden), Quoted: ptr(false), Style: ps}),
				NewDiagnosticsProperty[any]("DO_NOT_SHOW_NULL", nil, DiagnosticsPropertyOptions{DefaultValue: Null, Style: ps}),
				NewDiagnosticsProperty[any]("nullProperty", nil, DiagnosticsPropertyOptions{Style: ps}),
				NewStringProperty("node_type", "<root node>", StringPropertyOptions{ShowName: ptr(false), Quoted: ptr(false), Style: ps}),
			},
			Children: []*TestTree{
				newTestTree(tt{Name: "node A", Style: style}),
				newTestTree(tt{
					Name: "node B",
					Properties: []DiagnosticsNode{
						NewStringProperty("p1", "v1", StringPropertyOptions{Quoted: ptr(false), Style: ps}),
						NewStringProperty("p2", "v2", StringPropertyOptions{Quoted: ptr(false), Style: ps}),
					},
					Children: []*TestTree{
						newTestTree(tt{Name: "node B1", Style: style}),
						newTestTree(tt{
							Name:       "node B2",
							Properties: []DiagnosticsNode{NewStringProperty("property1", "value1", StringPropertyOptions{Quoted: ptr(false), Style: ps})},
							Style:      style,
						}),
						newTestTree(tt{
							Name: "node B3",
							Properties: []DiagnosticsNode{
								NewStringProperty("node_type", "<leaf node>", StringPropertyOptions{ShowName: ptr(false), Quoted: ptr(false), Style: ps}),
								NewIntProperty("foo", 42, IntPropertyOptions{Style: ps}),
							},
							Style: last,
						}),
					},
					Style: style,
				}),
				newTestTree(tt{
					Name:       "node C",
					Properties: []DiagnosticsNode{NewStringProperty("foo", "multi\nline\nvalue!", StringPropertyOptions{Quoted: ptr(false), Style: ps})},
					Style:      last,
				}),
			},
			Style: lastChildStyle,
		})

		if tree.style == nil || (*tree.style != DiagnosticsTreeStyleSingleLine && *tree.style != DiagnosticsTreeStyleErrorProperty) {
			hasAGoodToStringDeep(t, tree)
		}
		equalsIgnoringHashCodes(t, tree.ToDiagnosticsNode(ToDiagnosticsNodeOptions{Name: name, Style: style}).ToStringDeep(), golden, description)
		validateNodeJsonSerialization(t, tree.ToDiagnosticsNode())
	}

	sparse, errorProperty := st(DiagnosticsTreeStyleSparse), st(DiagnosticsTreeStyleErrorProperty)
	goldenStyleTest("sparse", nil, sparse, nil, nil,
		"TestTree#00000\n"+
			" │ stringProperty1: value1\n"+
			" │ doubleProperty1: 42.5\n"+
			" │ roundedProperty: 0.3\n"+
			" │ nullProperty: null\n"+
			" │ <root node>\n"+
			" │\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ │ p1: v1\n"+
			" │ │ p2: v2\n"+
			" │ │\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ │   property1: value1\n"+
			" │ │\n"+
			" │ └─child node B3: TestTree#00000\n"+
			" │     <leaf node>\n"+
			" │     foo: 42\n"+
			" │\n"+
			" └─child node C: TestTree#00000\n"+
			"     foo:\n"+
			"       multi\n"+
			"       line\n"+
			"       value!\n")
	goldenStyleTest("sparse with indented single line properties", nil, sparse, nil, errorProperty,
		"TestTree#00000\n"+
			" │ stringProperty1:\n"+
			" │   value1\n"+
			" │ doubleProperty1:\n"+
			" │   42.5\n"+
			" │ roundedProperty:\n"+
			" │   0.3\n"+
			" │ nullProperty:\n"+
			" │   null\n"+
			" │ <root node>\n"+
			" │\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ │ p1:\n"+
			" │ │   v1\n"+
			" │ │ p2:\n"+
			" │ │   v2\n"+
			" │ │\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ │   property1:\n"+
			" │ │     value1\n"+
			" │ │\n"+
			" │ └─child node B3: TestTree#00000\n"+
			" │     <leaf node>\n"+
			" │     foo:\n"+
			" │       42\n"+
			" │\n"+
			" └─child node C: TestTree#00000\n"+
			"     foo:\n"+
			"       multi\n"+
			"       line\n"+
			"       value!\n")
	goldenStyleTest("dense", nil, st(DiagnosticsTreeStyleDense), nil, nil,
		"TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)\n"+
			"├child node A: TestTree#00000\n"+
			"├child node B: TestTree#00000(p1: v1, p2: v2)\n"+
			"│├child node B1: TestTree#00000\n"+
			"│├child node B2: TestTree#00000(property1: value1)\n"+
			"│└child node B3: TestTree#00000(<leaf node>, foo: 42)\n"+
			"└child node C: TestTree#00000(foo: multi\\nline\\nvalue!)\n")
	goldenStyleTest("dashed", nil, st(DiagnosticsTreeStyleOffstage), nil, nil,
		"TestTree#00000\n"+
			" │ stringProperty1: value1\n"+
			" │ doubleProperty1: 42.5\n"+
			" │ roundedProperty: 0.3\n"+
			" │ nullProperty: null\n"+
			" │ <root node>\n"+
			" │\n"+
			" ╎╌child node A: TestTree#00000\n"+
			" ╎╌child node B: TestTree#00000\n"+
			" ╎ │ p1: v1\n"+
			" ╎ │ p2: v2\n"+
			" ╎ │\n"+
			" ╎ ╎╌child node B1: TestTree#00000\n"+
			" ╎ ╎╌child node B2: TestTree#00000\n"+
			" ╎ ╎   property1: value1\n"+
			" ╎ ╎\n"+
			" ╎ └╌child node B3: TestTree#00000\n"+
			" ╎     <leaf node>\n"+
			" ╎     foo: 42\n"+
			" ╎\n"+
			" └╌child node C: TestTree#00000\n"+
			"     foo:\n"+
			"       multi\n"+
			"       line\n"+
			"       value!\n")
	goldenStyleTest("transition children", nil, sparse, st(DiagnosticsTreeStyleTransition), nil,
		"TestTree#00000\n"+
			" │ stringProperty1: value1\n"+
			" │ doubleProperty1: 42.5\n"+
			" │ roundedProperty: 0.3\n"+
			" │ nullProperty: null\n"+
			" │ <root node>\n"+
			" │\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ │ p1: v1\n"+
			" │ │ p2: v2\n"+
			" │ │\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ │   property1: value1\n"+
			" │ │\n"+
			" │ ╘═╦══ child node B3 ═══\n"+
			" │   ║ TestTree#00000:\n"+
			" │   ║   <leaf node>\n"+
			" │   ║   foo: 42\n"+
			" │   ╚═══════════\n"+
			" ╘═╦══ child node C ═══\n"+
			"   ║ TestTree#00000:\n"+
			"   ║   foo:\n"+
			"   ║     multi\n"+
			"   ║     line\n"+
			"   ║     value!\n"+
			"   ╚═══════════\n")
	goldenStyleTest("error children", nil, sparse, st(DiagnosticsTreeStyleError), nil,
		"TestTree#00000\n"+
			" │ stringProperty1: value1\n"+
			" │ doubleProperty1: 42.5\n"+
			" │ roundedProperty: 0.3\n"+
			" │ nullProperty: null\n"+
			" │ <root node>\n"+
			" │\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ │ p1: v1\n"+
			" │ │ p2: v2\n"+
			" │ │\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ │   property1: value1\n"+
			" │ │\n"+
			" │ ╘═╦══╡ CHILD NODE B3: TESTTREE#00000 ╞═══════════════════════════════\n"+
			" │   ║ <leaf node>\n"+
			" │   ║ foo: 42\n"+
			" │   ╚══════════════════════════════════════════════════════════════════\n"+
			" ╘═╦══╡ CHILD NODE C: TESTTREE#00000 ╞════════════════════════════════\n"+
			"   ║ foo:\n"+
			"   ║   multi\n"+
			"   ║   line\n"+
			"   ║   value!\n"+
			"   ╚══════════════════════════════════════════════════════════════════\n")
	// You would never really want to make everything a transition child like
	// this but you can and still get a readable tree.
	goldenStyleTest("transition", nil, st(DiagnosticsTreeStyleTransition), nil, nil,
		"TestTree#00000:\n"+
			"  stringProperty1: value1\n"+
			"  doubleProperty1: 42.5\n"+
			"  roundedProperty: 0.3\n"+
			"  nullProperty: null\n"+
			"  <root node>\n"+
			"  ╞═╦══ child node A ═══\n"+
			"  │ ║ TestTree#00000\n"+
			"  │ ╚═══════════\n"+
			"  ╞═╦══ child node B ═══\n"+
			"  │ ║ TestTree#00000:\n"+
			"  │ ║   p1: v1\n"+
			"  │ ║   p2: v2\n"+
			"  │ ║   ╞═╦══ child node B1 ═══\n"+
			"  │ ║   │ ║ TestTree#00000\n"+
			"  │ ║   │ ╚═══════════\n"+
			"  │ ║   ╞═╦══ child node B2 ═══\n"+
			"  │ ║   │ ║ TestTree#00000:\n"+
			"  │ ║   │ ║   property1: value1\n"+
			"  │ ║   │ ╚═══════════\n"+
			"  │ ║   ╘═╦══ child node B3 ═══\n"+
			"  │ ║     ║ TestTree#00000:\n"+
			"  │ ║     ║   <leaf node>\n"+
			"  │ ║     ║   foo: 42\n"+
			"  │ ║     ╚═══════════\n"+
			"  │ ╚═══════════\n"+
			"  ╘═╦══ child node C ═══\n"+
			"    ║ TestTree#00000:\n"+
			"    ║   foo:\n"+
			"    ║     multi\n"+
			"    ║     line\n"+
			"    ║     value!\n"+
			"    ╚═══════════\n")
	goldenStyleTest("whitespace", nil, st(DiagnosticsTreeStyleWhitespace), nil, nil,
		"TestTree#00000:\n"+
			"  stringProperty1: value1\n"+
			"  doubleProperty1: 42.5\n"+
			"  roundedProperty: 0.3\n"+
			"  nullProperty: null\n"+
			"  <root node>\n"+
			"  child node A: TestTree#00000\n"+
			"  child node B: TestTree#00000:\n"+
			"    p1: v1\n"+
			"    p2: v2\n"+
			"    child node B1: TestTree#00000\n"+
			"    child node B2: TestTree#00000:\n"+
			"      property1: value1\n"+
			"    child node B3: TestTree#00000:\n"+
			"      <leaf node>\n"+
			"      foo: 42\n"+
			"  child node C: TestTree#00000:\n"+
			"    foo:\n"+
			"      multi\n"+
			"      line\n"+
			"      value!\n")
	goldenStyleTest("flat", nil, st(DiagnosticsTreeStyleFlat), nil, nil,
		"TestTree#00000:\n"+
			"stringProperty1: value1\n"+
			"doubleProperty1: 42.5\n"+
			"roundedProperty: 0.3\n"+
			"nullProperty: null\n"+
			"<root node>\n"+
			"child node A: TestTree#00000\n"+
			"child node B: TestTree#00000:\n"+
			"p1: v1\n"+
			"p2: v2\n"+
			"child node B1: TestTree#00000\n"+
			"child node B2: TestTree#00000:\n"+
			"property1: value1\n"+
			"child node B3: TestTree#00000:\n"+
			"<leaf node>\n"+
			"foo: 42\n"+
			"child node C: TestTree#00000:\n"+
			"foo:\n"+
			"  multi\n"+
			"  line\n"+
			"  value!\n")
	// Single line mode does not display children.
	goldenStyleTest("single line", nil, st(DiagnosticsTreeStyleSingleLine), nil, nil,
		"TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)")
	goldenStyleTest("single line", ptr("some name"), st(DiagnosticsTreeStyleSingleLine), nil, nil,
		"some name: TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)")
	// No name so we don't indent.
	goldenStyleTest("indented single line", nil, errorProperty, nil, nil,
		"TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)\n")
	goldenStyleTest("indented single line", ptr("some name"), errorProperty, nil, nil,
		"some name:\n"+
			"  TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)\n")
	goldenStyleTest("single line last child", nil, sparse, st(DiagnosticsTreeStyleSingleLine), nil,
		"TestTree#00000\n"+
			" │ stringProperty1: value1\n"+
			" │ doubleProperty1: 42.5\n"+
			" │ roundedProperty: 0.3\n"+
			" │ nullProperty: null\n"+
			" │ <root node>\n"+
			" │\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ │ p1: v1\n"+
			" │ │ p2: v2\n"+
			" │ │\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ │   property1: value1\n"+
			" │ │\n"+
			" │ └─child node B3: TestTree#00000(<leaf node>, foo: 42)\n"+
			" └─child node C: TestTree#00000(foo: multi\\nline\\nvalue!)\n")
	goldenStyleTest("indented single line last child", nil, sparse, errorProperty, nil,
		"TestTree#00000\n"+
			" │ stringProperty1: value1\n"+
			" │ doubleProperty1: 42.5\n"+
			" │ roundedProperty: 0.3\n"+
			" │ nullProperty: null\n"+
			" │ <root node>\n"+
			" │\n"+
			" ├─child node A: TestTree#00000\n"+
			" ├─child node B: TestTree#00000\n"+
			" │ │ p1: v1\n"+
			" │ │ p2: v2\n"+
			" │ │\n"+
			" │ ├─child node B1: TestTree#00000\n"+
			" │ ├─child node B2: TestTree#00000\n"+
			" │ │   property1: value1\n"+
			" │ │\n"+
			" │ └─child node B3:\n"+
			" │     TestTree#00000(<leaf node>, foo: 42)\n"+
			" └─child node C:\n"+
			"     TestTree#00000(foo: multi\\nline\\nvalue!)\n")
}

func TestToString(t *testing.T) {
	tree := newTestTree(tt{
		Properties: []DiagnosticsNode{
			NewStringProperty("stringProperty1", "value1", StringPropertyOptions{Quoted: ptr(false)}),
			NewDoubleProperty("doubleProperty1", 42.5),
			NewDoubleProperty("roundedProperty", 1.0/3.0),
			NewStringProperty("DO_NOT_SHOW", "DO_NOT_SHOW", StringPropertyOptions{Level: lv(DiagnosticLevelHidden), Quoted: ptr(false)}),
			NewStringProperty("DEBUG_ONLY", "DEBUG_ONLY", StringPropertyOptions{Level: lv(DiagnosticLevelDebug), Quoted: ptr(false)}),
		},
		// child to verify that children are not included in the toString.
		Children: []*TestTree{newTestTree(tt{Name: "node A"})},
	})
	equalsIgnoringHashCodes(t, tree.String(),
		"TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3)", "toString")
	equalsIgnoringHashCodes(t, tree.ToString(DiagnosticableToStringOptions{MinLevel: lv(DiagnosticLevelDebug)}),
		"TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, DEBUG_ONLY: DEBUG_ONLY)", "toString(minLevel: debug)")
}

func TestTransition(t *testing.T) {
	// Test multiple styles integrating together in the same tree due to using
	// transition to go between styles that would otherwise be incompatible.
	dense := st(DiagnosticsTreeStyleDense)
	tree := newTestTree(tt{
		Style:      st(DiagnosticsTreeStyleSparse),
		Properties: []DiagnosticsNode{NewStringProperty("stringProperty1", "value1")},
		Children: []*TestTree{
			newTestTree(tt{
				Style: st(DiagnosticsTreeStyleTransition),
				Name:  "node transition",
				Properties: []DiagnosticsNode{
					NewStringProperty("p1", "v1"),
					newTestTree(tt{Properties: []DiagnosticsNode{NewDiagnosticsProperty("survived", true)}}).
						ToDiagnosticsNode(ToDiagnosticsNodeOptions{Name: ptr("tree property"), Style: st(DiagnosticsTreeStyleWhitespace)}),
				},
				Children: []*TestTree{
					newTestTree(tt{Name: "dense child", Style: dense}),
					newTestTree(tt{Name: "dense", Properties: []DiagnosticsNode{NewStringProperty("property1", "value1")}, Style: dense}),
					newTestTree(tt{
						Name: "node B3",
						Properties: []DiagnosticsNode{
							NewStringProperty("node_type", "<leaf node>", StringPropertyOptions{ShowName: ptr(false), Quoted: ptr(false)}),
							NewIntProperty("foo", 42),
						},
						Style: dense,
					}),
				},
			}),
			newTestTree(tt{
				Name:       "node C",
				Properties: []DiagnosticsNode{NewStringProperty("foo", "multi\nline\nvalue!", StringPropertyOptions{Quoted: ptr(false)})},
				Style:      st(DiagnosticsTreeStyleSparse),
			}),
		},
	})
	hasAGoodToStringDeep(t, tree)
	equalsIgnoringHashCodes(t, tree.ToDiagnosticsNode().ToStringDeep(),
		"TestTree#00000\n"+
			" │ stringProperty1: \"value1\"\n"+
			" ╞═╦══ child node transition ═══\n"+
			" │ ║ TestTree#00000:\n"+
			" │ ║   p1: \"v1\"\n"+
			" │ ║   tree property: TestTree#00000:\n"+
			" │ ║     survived: true\n"+
			" │ ║   ├child dense child: TestTree#00000\n"+
			" │ ║   ├child dense: TestTree#00000(property1: \"value1\")\n"+
			" │ ║   └child node B3: TestTree#00000(<leaf node>, foo: 42)\n"+
			" │ ╚═══════════\n"+
			" └─child node C: TestTree#00000\n"+
			"     foo:\n"+
			"       multi\n"+
			"       line\n"+
			"       value!\n", "transition")
}

func TestDescribeEnum(t *testing.T) {
	equal(t, DescribeEnum(ExampleEnumHello), "hello")
	equal(t, DescribeEnum(ExampleEnumWorld), "world")
	equal(t, DescribeEnum(ExampleEnumDeferToChild), "deferToChild")
	func() {
		defer func() {
			equal[any](t, recover(), `The provided object "Hello World" is not an enum.`)
		}()
		DescribeEnum("Hello World")
	}()
}

func TestStringProperty(t *testing.T) {
	equal(t, NewStringProperty("name", "value", StringPropertyOptions{Quoted: ptr(false)}).String(), "name: value")

	stringProperty := NewStringProperty("name", "value", StringPropertyOptions{Description: ptr("VALUE"), IfEmpty: ptr("<hidden>"), Quoted: ptr(false)})
	equal(t, stringProperty.String(), "name: VALUE")
	validateStringPropertyJsonSerialization(t, stringProperty)

	equal(t, NewStringProperty("name", "value", StringPropertyOptions{ShowName: ptr(false), IfEmpty: ptr("<hidden>"), Quoted: ptr(false)}).String(), "value")
	equal(t, NewStringProperty("name", "", StringPropertyOptions{IfEmpty: ptr("<hidden>")}).String(), "name: <hidden>")
	equal(t, NewStringProperty("name", "", StringPropertyOptions{IfEmpty: ptr("<hidden>"), ShowName: ptr(false)}).String(), "<hidden>")

	equal(t, NewStringProperty[*string]("name", nil).IsFiltered(DiagnosticLevelInfo), false)
	equal(t, NewStringProperty("name", "value", StringPropertyOptions{Level: lv(DiagnosticLevelHidden)}).IsFiltered(DiagnosticLevelInfo), true)
	equal(t, NewStringProperty[*string]("name", nil, StringPropertyOptions{DefaultValue: Null}).IsFiltered(DiagnosticLevelInfo), true)
	quoted := NewStringProperty("name", "value")
	equal(t, quoted.String(), `name: "value"`)
	validateStringPropertyJsonSerialization(t, quoted)

	equal(t, NewStringProperty("name", "value", StringPropertyOptions{ShowName: ptr(false)}).String(), `"value"`)
	equal(t, NewStringProperty[*string]("name", nil, StringPropertyOptions{ShowName: ptr(false)}).String(), "null")
}

func TestBoolProperty(t *testing.T) {
	trueProperty := NewDiagnosticsProperty("name", true)
	falseProperty := NewDiagnosticsProperty("name", false)
	equal(t, trueProperty.String(), "name: true")
	equal(t, trueProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, trueProperty.TypedValue(), true)
	equal(t, falseProperty.String(), "name: false")
	equal(t, falseProperty.TypedValue(), false)
	equal(t, falseProperty.IsFiltered(DiagnosticLevelInfo), false)
	validatePropertyJsonSerialization(t, trueProperty)
	validatePropertyJsonSerialization(t, falseProperty)
	truthyProperty := NewDiagnosticsProperty("name", true, DiagnosticsPropertyOptions{Description: ptr("truthy")})
	equal(t, truthyProperty.String(), "name: truthy")
	validatePropertyJsonSerialization(t, truthyProperty)
	equal(t, NewDiagnosticsProperty("name", true, DiagnosticsPropertyOptions{ShowName: ptr(false)}).String(), "true")

	equal(t, NewDiagnosticsProperty[*bool]("name", nil).IsFiltered(DiagnosticLevelInfo), false)
	equal(t, NewDiagnosticsProperty("name", true, DiagnosticsPropertyOptions{Level: lv(DiagnosticLevelHidden)}).IsFiltered(DiagnosticLevelInfo), true)
	equal(t, NewDiagnosticsProperty[*bool]("name", nil, DiagnosticsPropertyOptions{DefaultValue: Null}).IsFiltered(DiagnosticLevelInfo), true)
	missingBool := NewDiagnosticsProperty[*bool]("name", nil, DiagnosticsPropertyOptions{IfNull: ptr("missing")})
	equal(t, missingBool.String(), "name: missing")
	validatePropertyJsonSerialization(t, missingBool)
}

func TestFlagProperty(t *testing.T) {
	trueFlag := NewFlagProperty("myFlag", true, FlagPropertyOptions{IfTrue: ptr("myFlag")})
	falseFlag := NewFlagProperty("myFlag", false, FlagPropertyOptions{IfTrue: ptr("myFlag")})
	equal(t, trueFlag.String(), "myFlag")
	validateFlagPropertyJsonSerialization(t, trueFlag)
	validateFlagPropertyJsonSerialization(t, falseFlag)

	equal(t, *trueFlag.TypedValue(), true)
	equal(t, *falseFlag.TypedValue(), false)

	equal(t, trueFlag.IsFiltered(DiagnosticLevelFine), false)
	equal(t, falseFlag.IsFiltered(DiagnosticLevelFine), true)
}

func TestPropertyWithTooltip(t *testing.T) {
	withTooltip := NewDiagnosticsProperty("name", "value", DiagnosticsPropertyOptions{Tooltip: ptr("tooltip")})
	equal(t, withTooltip.String(), "name: value (tooltip)")
	equal(t, withTooltip.TypedValue(), "value")
	equal(t, withTooltip.IsFiltered(DiagnosticLevelFine), false)
	validatePropertyJsonSerialization(t, withTooltip)
}

func TestDoubleProperty(t *testing.T) {
	doubleProperty := NewDoubleProperty("name", 42.0)
	equal(t, doubleProperty.String(), "name: 42.0")
	equal(t, doubleProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, *doubleProperty.TypedValue(), 42.0)
	validateDoublePropertyJsonSerialization(t, doubleProperty)

	equal(t, NewDoubleProperty("name", 1.3333).String(), "name: 1.3")

	equal(t, NewDoubleProperty[*float64]("name", nil).String(), "name: null")
	equal(t, NewDoubleProperty[*float64]("name", nil).IsFiltered(DiagnosticLevelInfo), false)

	equal(t, NewDoubleProperty[*float64]("name", nil, DoublePropertyOptions{IfNull: ptr("missing")}).String(), "name: missing")

	doubleWithUnit := NewDoubleProperty("name", 42.0, DoublePropertyOptions{Unit: ptr("px")})
	equal(t, doubleWithUnit.String(), "name: 42.0px")
	validateDoublePropertyJsonSerialization(t, doubleWithUnit)
}

func TestDoubleInfinitySerialization(t *testing.T) {
	inf := 1.0
	inf /= 0
	infProperty1 := NewDoubleProperty("double1", inf)
	validateDoublePropertyJsonSerialization(t, infProperty1)
	equal(t, infProperty1.String(), "double1: Infinity")

	infProperty2 := NewDoubleProperty("double2", -inf)
	validateDoublePropertyJsonSerialization(t, infProperty2)
	equal(t, infProperty2.String(), "double2: -Infinity")
}

func TestUnsafeDoubleProperty(t *testing.T) {
	safe := NewDoublePropertyLazy("name", func() float64 { return 42.0 })
	equal(t, safe.String(), "name: 42.0")
	equal(t, safe.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, *safe.TypedValue(), 42.0)
	validateDoublePropertyJsonSerialization(t, safe)
	equal(t, NewDoublePropertyLazy("name", func() float64 { return 1.3333 }).String(), "name: 1.3")

	equal(t, NewDoublePropertyLazy("name", func() *float64 { return nil }).String(), "name: null")
	equal(t, NewDoublePropertyLazy("name", func() *float64 { return nil }).IsFiltered(DiagnosticLevelInfo), false)

	throwingProperty := NewDoublePropertyLazy("name", func() float64 {
		panic(NewFlutterErrorFromParts([]DiagnosticsNode{NewErrorSummary("Invalid constraints")}))
	})
	// TODO(jacobr): it would be better if throwingProperty.object threw an
	// exception.
	equal(t, throwingProperty.Value(), nil)
	equal(t, throwingProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, throwingProperty.String(), "name: EXCEPTION (FlutterError)")
	equal(t, throwingProperty.Level(), DiagnosticLevelError)
	validateDoublePropertyJsonSerialization(t, throwingProperty)
}

func TestPercentProperty(t *testing.T) {
	equal(t, NewPercentProperty("name", 0.4).String(), "name: 40.0%")

	complexPercentProperty := NewPercentProperty("name", 0.99, PercentPropertyOptions{Unit: ptr("invisible"), Tooltip: ptr("almost transparent")})
	equal(t, complexPercentProperty.String(), "name: 99.0% invisible (almost transparent)")
	validateDoublePropertyJsonSerialization(t, &complexPercentProperty.DoubleProperty)

	equal(t, NewPercentProperty[*float64]("name", nil, PercentPropertyOptions{Unit: ptr("invisible"), Tooltip: ptr("!")}).String(), "name: null (!)")

	equal(t, *NewPercentProperty("name", 0.4).TypedValue(), 0.4)
	equal(t, NewPercentProperty("name", 0.0).String(), "name: 0.0%")
	equal(t, NewPercentProperty("name", -10.0).String(), "name: 0.0%")
	equal(t, NewPercentProperty("name", 1.0).String(), "name: 100.0%")
	equal(t, NewPercentProperty("name", 3.0).String(), "name: 100.0%")
	equal(t, NewPercentProperty[*float64]("name", nil).String(), "name: null")
	equal(t, NewPercentProperty[*float64]("name", nil, PercentPropertyOptions{IfNull: ptr("missing")}).String(), "name: missing")
	equal(t, NewPercentProperty[*float64]("name", nil, PercentPropertyOptions{IfNull: ptr("missing"), ShowName: ptr(false)}).String(), "missing")
	equal(t, NewPercentProperty("name", 0.5, PercentPropertyOptions{ShowName: ptr(false)}).String(), "50.0%")
}

func TestCallbackProperty(t *testing.T) {
	onClick := func() {}
	present := NewObjectFlagProperty("onClick", onClick, ObjectFlagPropertyOptions{IfPresent: ptr("clickable")})
	missing := NewObjectFlagProperty[func()]("onClick", nil, ObjectFlagPropertyOptions{IfPresent: ptr("clickable")})

	equal(t, present.String(), "clickable")
	equal(t, present.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, present.TypedValue() != nil, true)
	validateObjectFlagPropertyJsonSerialization(t, present)
	equal(t, missing.String(), "onClick: null")
	equal(t, missing.IsFiltered(DiagnosticLevelFine), true)
	validateObjectFlagPropertyJsonSerialization(t, missing)
}

func TestMissingCallbackProperty(t *testing.T) {
	onClick := func() {}
	present := NewObjectFlagProperty("onClick", onClick, ObjectFlagPropertyOptions{IfNull: ptr("disabled")})
	missing := NewObjectFlagProperty[func()]("onClick", nil, ObjectFlagPropertyOptions{IfNull: ptr("disabled")})

	equal(t, present.String(), "onClick: Closure: () => void")
	equal(t, present.IsFiltered(DiagnosticLevelFine), true)
	equal(t, present.TypedValue() != nil, true)
	equal(t, missing.String(), "disabled")
	equal(t, missing.IsFiltered(DiagnosticLevelInfo), false)
	validateObjectFlagPropertyJsonSerialization(t, present)
	validateObjectFlagPropertyJsonSerialization(t, missing)
}

func TestDescribeBoolProperty(t *testing.T) {
	yes := NewFlagProperty("name", true, FlagPropertyOptions{IfTrue: ptr("YES"), IfFalse: ptr("NO"), ShowName: ptr(true)})
	no := NewFlagProperty("name", false, FlagPropertyOptions{IfTrue: ptr("YES"), IfFalse: ptr("NO"), ShowName: ptr(true)})
	equal(t, yes.String(), "name: YES")
	equal(t, yes.Level(), DiagnosticLevelInfo)
	equal(t, *yes.TypedValue(), true)
	validateFlagPropertyJsonSerialization(t, yes)
	equal(t, no.String(), "name: NO")
	equal(t, no.Level(), DiagnosticLevelInfo)
	equal(t, *no.TypedValue(), false)
	validateFlagPropertyJsonSerialization(t, no)

	equal(t, NewFlagProperty("name", true, FlagPropertyOptions{IfTrue: ptr("YES"), IfFalse: ptr("NO")}).String(), "YES")
	equal(t, NewFlagProperty("name", false, FlagPropertyOptions{IfTrue: ptr("YES"), IfFalse: ptr("NO")}).String(), "NO")
	equal(t, NewFlagProperty("name", true, FlagPropertyOptions{IfTrue: ptr("YES"), IfFalse: ptr("NO"),
		Level: lv(DiagnosticLevelHidden), ShowName: ptr(true)}).Level(), DiagnosticLevelHidden)
}

func TestEnumProperty(t *testing.T) {
	hello := NewEnumProperty("name", ExampleEnumHello)
	world := NewEnumProperty("name", ExampleEnumWorld)
	deferToChild := NewEnumProperty("name", ExampleEnumDeferToChild)
	nullEnum := NewEnumProperty[*ExampleEnum]("name", nil)
	equal(t, hello.Level(), DiagnosticLevelInfo)
	equal(t, hello.TypedValue(), ExampleEnumHello)
	equal(t, hello.String(), "name: hello")
	validatePropertyJsonSerialization(t, &hello.DiagnosticsProperty)

	equal(t, world.Level(), DiagnosticLevelInfo)
	equal(t, world.TypedValue(), ExampleEnumWorld)
	equal(t, world.String(), "name: world")
	validatePropertyJsonSerialization(t, &world.DiagnosticsProperty)

	equal(t, deferToChild.Level(), DiagnosticLevelInfo)
	equal(t, deferToChild.TypedValue(), ExampleEnumDeferToChild)
	equal(t, deferToChild.String(), "name: deferToChild")
	validatePropertyJsonSerialization(t, &deferToChild.DiagnosticsProperty)

	equal(t, nullEnum.Level(), DiagnosticLevelInfo)
	equal(t, nullEnum.Value(), nil)
	equal(t, nullEnum.String(), "name: null")
	validatePropertyJsonSerialization(t, &nullEnum.DiagnosticsProperty)

	matchesDefault := NewEnumProperty("name", ExampleEnumHello, EnumPropertyOptions{DefaultValue: ExampleEnumHello})
	equal(t, matchesDefault.String(), "name: hello")
	equal(t, matchesDefault.TypedValue(), ExampleEnumHello)
	equal(t, matchesDefault.IsFiltered(DiagnosticLevelInfo), true)
	validatePropertyJsonSerialization(t, &matchesDefault.DiagnosticsProperty)

	equal(t, NewEnumProperty("name", ExampleEnumHello, EnumPropertyOptions{Level: lv(DiagnosticLevelHidden)}).Level(), DiagnosticLevelHidden)
}

func TestIntProperty(t *testing.T) {
	regular := NewIntProperty("name", 42)
	equal(t, regular.String(), "name: 42")
	equal(t, *regular.TypedValue(), 42)
	equal(t, regular.Level(), DiagnosticLevelInfo)

	nullValue := NewIntProperty[*int]("name", nil)
	equal(t, nullValue.String(), "name: null")
	equal(t, nullValue.Value(), nil)
	equal(t, nullValue.Level(), DiagnosticLevelInfo)

	hideNull := NewIntProperty[*int]("name", nil, IntPropertyOptions{DefaultValue: Null})
	equal(t, hideNull.String(), "name: null")
	equal(t, hideNull.Value(), nil)
	equal(t, hideNull.IsFiltered(DiagnosticLevelInfo), true)

	nullDescription := NewIntProperty[*int]("name", nil, IntPropertyOptions{IfNull: ptr("missing")})
	equal(t, nullDescription.String(), "name: missing")
	equal(t, nullDescription.Value(), nil)
	equal(t, nullDescription.Level(), DiagnosticLevelInfo)

	hideName := NewIntProperty("name", 42, IntPropertyOptions{ShowName: ptr(false)})
	equal(t, hideName.String(), "42")
	equal(t, *hideName.TypedValue(), 42)
	equal(t, hideName.Level(), DiagnosticLevelInfo)

	withUnit := NewIntProperty("name", 42, IntPropertyOptions{Unit: ptr("pt")})
	equal(t, withUnit.String(), "name: 42pt")
	equal(t, *withUnit.TypedValue(), 42)
	equal(t, withUnit.Level(), DiagnosticLevelInfo)

	defaultValue := NewIntProperty("name", 42, IntPropertyOptions{DefaultValue: 42})
	equal(t, defaultValue.String(), "name: 42")
	equal(t, *defaultValue.TypedValue(), 42)
	equal(t, defaultValue.IsFiltered(DiagnosticLevelInfo), true)

	notDefaultValue := NewIntProperty("name", 43, IntPropertyOptions{DefaultValue: 42})
	equal(t, notDefaultValue.String(), "name: 43")
	equal(t, *notDefaultValue.TypedValue(), 43)
	equal(t, notDefaultValue.Level(), DiagnosticLevelInfo)

	hidden := NewIntProperty("name", 42, IntPropertyOptions{Level: lv(DiagnosticLevelHidden)})
	equal(t, hidden.String(), "name: 42")
	equal(t, *hidden.TypedValue(), 42)
	equal(t, hidden.Level(), DiagnosticLevelHidden)
}

func TestObjectProperty(t *testing.T) {
	rect := ui.NewRectFromLTRB(0.0, 0.0, 20.0, 20.0)
	simple := NewDiagnosticsProperty("name", rect)
	equal(t, simple.TypedValue(), rect)
	equal(t, simple.Level(), DiagnosticLevelInfo)
	equal(t, simple.String(), "name: Rect.fromLTRB(0.0, 0.0, 20.0, 20.0)")
	validatePropertyJsonSerialization(t, simple)

	withDescription := NewDiagnosticsProperty("name", rect, DiagnosticsPropertyOptions{Description: ptr("small rect")})
	equal(t, withDescription.TypedValue(), rect)
	equal(t, withDescription.Level(), DiagnosticLevelInfo)
	equal(t, withDescription.String(), "name: small rect")
	validatePropertyJsonSerialization(t, withDescription)

	nullProperty := NewDiagnosticsProperty[any]("name", nil)
	equal(t, nullProperty.Value(), nil)
	equal(t, nullProperty.Level(), DiagnosticLevelInfo)
	equal(t, nullProperty.String(), "name: null")
	validatePropertyJsonSerialization(t, nullProperty)

	hideNullProperty := NewDiagnosticsProperty[any]("name", nil, DiagnosticsPropertyOptions{DefaultValue: Null})
	equal(t, hideNullProperty.Value(), nil)
	equal(t, hideNullProperty.IsFiltered(DiagnosticLevelInfo), true)
	equal(t, hideNullProperty.String(), "name: null")
	validatePropertyJsonSerialization(t, hideNullProperty)

	nullDescription := NewDiagnosticsProperty[any]("name", nil, DiagnosticsPropertyOptions{IfNull: ptr("missing")})
	equal(t, nullDescription.Value(), nil)
	equal(t, nullDescription.Level(), DiagnosticLevelInfo)
	equal(t, nullDescription.String(), "name: missing")
	validatePropertyJsonSerialization(t, nullDescription)

	hideName := NewDiagnosticsProperty("name", rect, DiagnosticsPropertyOptions{ShowName: ptr(false), Level: lv(DiagnosticLevelWarning)})
	equal(t, hideName.TypedValue(), rect)
	equal(t, hideName.Level(), DiagnosticLevelWarning)
	equal(t, hideName.String(), "Rect.fromLTRB(0.0, 0.0, 20.0, 20.0)")
	validatePropertyJsonSerialization(t, hideName)

	hideSeparator := NewDiagnosticsProperty("Creator", rect, DiagnosticsPropertyOptions{ShowSeparator: ptr(false)})
	equal(t, hideSeparator.TypedValue(), rect)
	equal(t, hideSeparator.Level(), DiagnosticLevelInfo)
	equal(t, hideSeparator.String(), "Creator Rect.fromLTRB(0.0, 0.0, 20.0, 20.0)")
	validatePropertyJsonSerialization(t, hideSeparator)
}

func TestLazyObjectProperty(t *testing.T) {
	rect := ui.NewRectFromLTRB(0.0, 0.0, 20.0, 20.0)
	simple := NewDiagnosticsPropertyLazy("name", func() ui.Rect { return rect }, DiagnosticsPropertyOptions{Description: ptr("small rect")})
	equal(t, simple.TypedValue(), rect)
	equal(t, simple.Level(), DiagnosticLevelInfo)
	equal(t, simple.String(), "name: small rect")
	validatePropertyJsonSerialization(t, simple)

	nullProperty := NewDiagnosticsPropertyLazy("name", func() any { return nil }, DiagnosticsPropertyOptions{Description: ptr("missing")})
	equal(t, nullProperty.Value(), nil)
	equal(t, nullProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, nullProperty.String(), "name: missing")
	validatePropertyJsonSerialization(t, nullProperty)

	hideNullProperty := NewDiagnosticsPropertyLazy("name", func() any { return nil }, DiagnosticsPropertyOptions{Description: ptr("missing"), DefaultValue: Null})
	equal(t, hideNullProperty.Value(), nil)
	equal(t, hideNullProperty.IsFiltered(DiagnosticLevelInfo), true)
	equal(t, hideNullProperty.String(), "name: missing")
	validatePropertyJsonSerialization(t, hideNullProperty)

	hideName := NewDiagnosticsPropertyLazy("name", func() ui.Rect { return rect }, DiagnosticsPropertyOptions{Description: ptr("small rect"), ShowName: ptr(false)})
	equal(t, hideName.TypedValue(), rect)
	equal(t, hideName.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, hideName.String(), "small rect")
	validatePropertyJsonSerialization(t, hideName)

	throwingWithDescription := NewDiagnosticsPropertyLazy("name", func() any {
		panic(NewFlutterErrorFromParts([]DiagnosticsNode{NewErrorSummary("Property not available")}))
	},
		DiagnosticsPropertyOptions{Description: ptr("missing"), DefaultValue: Null})
	equal(t, throwingWithDescription.Value(), nil)
	_, isFlutterError := throwingWithDescription.Exception().(*FlutterError)
	equal(t, isFlutterError, true)
	equal(t, throwingWithDescription.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, throwingWithDescription.String(), "name: missing")
	validatePropertyJsonSerialization(t, throwingWithDescription)

	throwingProperty := NewDiagnosticsPropertyLazy("name", func() any {
		panic(NewFlutterErrorFromParts([]DiagnosticsNode{NewErrorSummary("Property not available")}))
	},
		DiagnosticsPropertyOptions{DefaultValue: Null})
	equal(t, throwingProperty.Value(), nil)
	_, isFlutterError = throwingProperty.Exception().(*FlutterError)
	equal(t, isFlutterError, true)
	equal(t, throwingProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, throwingProperty.String(), "name: EXCEPTION (FlutterError)")
	validatePropertyJsonSerialization(t, throwingProperty)
}

func TestColorProperty(t *testing.T) {
	// Add more tests if colorProperty becomes more than a wrapper around
	// objectProperty.
	color := ui.NewColorFromARGB(255, 255, 255, 255)
	simple := NewDiagnosticsProperty("name", color)
	validatePropertyJsonSerialization(t, simple)
	equal(t, simple.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, simple.TypedValue(), color)
	equal(t, simple.PropertyType(), reflect.TypeFor[ui.Color]())
	equal(t, simple.String(), "name: "+ui.NewColor(0xffffffff).String())
	validatePropertyJsonSerialization(t, simple)
}

func TestFlagProperty2(t *testing.T) {
	show := NewFlagProperty("wasLayout", true, FlagPropertyOptions{IfTrue: ptr("layout computed")})
	equal(t, show.Name(), "wasLayout")
	equal(t, *show.TypedValue(), true)
	equal(t, show.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, show.String(), "layout computed")
	validateFlagPropertyJsonSerialization(t, show)

	hide := NewFlagProperty("wasLayout", false, FlagPropertyOptions{IfTrue: ptr("layout computed")})
	equal(t, hide.Name(), "wasLayout")
	equal(t, *hide.TypedValue(), false)
	equal(t, hide.Level(), DiagnosticLevelHidden)
	equal(t, hide.String(), "wasLayout: false")
	validateFlagPropertyJsonSerialization(t, hide)

	hideTrue := NewFlagProperty("wasLayout", true, FlagPropertyOptions{IfFalse: ptr("no layout computed")})
	equal(t, hideTrue.Name(), "wasLayout")
	equal(t, *hideTrue.TypedValue(), true)
	equal(t, hideTrue.Level(), DiagnosticLevelHidden)
	equal(t, hideTrue.String(), "wasLayout: true")
	validateFlagPropertyJsonSerialization(t, hideTrue)
}

func TestHasProperty(t *testing.T) {
	onClick := func() {}
	has := NewObjectFlagPropertyHas("onClick", onClick)
	equal(t, has.Name(), "onClick")
	equal(t, has.TypedValue() != nil, true)
	equal(t, has.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, has.String(), "has onClick")
	validateObjectFlagPropertyJsonSerialization(t, has)

	missing := NewObjectFlagPropertyHas[func()]("onClick", nil)
	equal(t, missing.Name(), "onClick")
	equal(t, missing.Value(), nil)
	equal(t, missing.IsFiltered(DiagnosticLevelInfo), true)
	equal(t, missing.String(), "onClick: null")
	validateObjectFlagPropertyJsonSerialization(t, missing)
}

type flagEntry = dart.MapEntry[string, func()]

func TestIterableFlagsProperty(t *testing.T) {
	// Normal property
	{
		onClick, onMove := func() {}, func() {}
		value := []flagEntry{{Key: "click", Value: onClick}, {Key: "move", Value: onMove}}
		flags := NewFlagsSummary("listeners", value)
		equal(t, flags.Name(), "listeners")
		equal(t, len(flags.TypedValue()), 2)
		equal(t, flags.IsFiltered(DiagnosticLevelInfo), false)
		equal(t, flags.String(), "listeners: click, move")
		validateIterableFlagsPropertyJsonSerialization(t, flags)
	}

	// Reversed-order property
	{
		onClick, onMove := func() {}, func() {}
		value := []flagEntry{{Key: "move", Value: onMove}, {Key: "click", Value: onClick}}
		flags := NewFlagsSummary("listeners", value)
		equal(t, flags.String(), "listeners: move, click")
		equal(t, flags.IsFiltered(DiagnosticLevelInfo), false)
		validateIterableFlagsPropertyJsonSerialization(t, flags)
	}

	// Partially empty property
	{
		onClick := func() {}
		value := []flagEntry{{Key: "move", Value: nil}, {Key: "click", Value: onClick}}
		flags := NewFlagsSummary("listeners", value)
		equal(t, flags.String(), "listeners: click")
		equal(t, flags.IsFiltered(DiagnosticLevelInfo), false)
		validateIterableFlagsPropertyJsonSerialization(t, flags)
	}

	// Empty property (without ifEmpty)
	{
		value := []flagEntry{{Key: "enter", Value: nil}}
		flags := NewFlagsSummary("listeners", value)
		equal(t, flags.IsFiltered(DiagnosticLevelInfo), true)
		validateIterableFlagsPropertyJsonSerialization(t, flags)
	}

	// Empty property (without ifEmpty)
	{
		value := []flagEntry{{Key: "enter", Value: nil}}
		flags := NewFlagsSummary("listeners", value, FlagsSummaryOptions{IfEmpty: ptr("<none>")})
		equal(t, flags.String(), "listeners: <none>")
		equal(t, flags.IsFiltered(DiagnosticLevelInfo), false)
		validateIterableFlagsPropertyJsonSerialization(t, flags)
	}
}

func TestIterableProperty(t *testing.T) {
	white := ui.NewColor(0xffffffff).String()

	ints := []int{1, 2, 3}
	intsProperty := NewIterableProperty("ints", ints)
	equal(t, len(intsProperty.TypedValue()), 3)
	equal(t, intsProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, intsProperty.String(), "ints: 1, 2, 3")

	doubles := []float64{1, 2, 3}
	doublesProperty := NewIterableProperty("doubles", doubles)
	equal(t, len(doublesProperty.TypedValue()), 3)
	equal(t, doublesProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, doublesProperty.String(), "doubles: 1.0, 2.0, 3.0")

	emptyProperty := NewIterableProperty("name", []any{})
	equal(t, len(emptyProperty.TypedValue()), 0)
	equal(t, emptyProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, emptyProperty.String(), "name: []")
	validateIterablePropertyJsonSerialization(t, emptyProperty)

	nullProperty := NewIterableProperty[any]("list", nil)
	equal(t, nullProperty.Value(), nil)
	equal(t, nullProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, nullProperty.String(), "list: null")
	validateIterablePropertyJsonSerialization(t, nullProperty)

	hideNullProperty := NewIterableProperty[any]("list", nil, IterablePropertyOptions{DefaultValue: Null})
	equal(t, hideNullProperty.Value(), nil)
	equal(t, hideNullProperty.IsFiltered(DiagnosticLevelInfo), true)
	equal(t, hideNullProperty.Level(), DiagnosticLevelFine)
	equal(t, hideNullProperty.String(), "list: null")
	validateIterablePropertyJsonSerialization(t, hideNullProperty)

	objects := []any{ui.NewRectFromLTRB(0.0, 0.0, 20.0, 20.0), ui.NewColorFromARGB(255, 255, 255, 255)}
	objectsProperty := NewIterableProperty("objects", objects)
	equal(t, len(objectsProperty.TypedValue()), 2)
	equal(t, objectsProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, objectsProperty.String(),
		"objects: Rect.fromLTRB(0.0, 0.0, 20.0, 20.0), "+white)
	validateIterablePropertyJsonSerialization(t, objectsProperty)

	multiLineProperty := NewIterableProperty("objects", objects, IterablePropertyOptions{Style: st(DiagnosticsTreeStyleWhitespace)})
	equal(t, len(multiLineProperty.TypedValue()), 2)
	equal(t, multiLineProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, multiLineProperty.String(),
		"objects:\n"+
			"Rect.fromLTRB(0.0, 0.0, 20.0, 20.0)\n"+
			white)
	equal(t, multiLineProperty.ToStringDeep(DiagnosticsNodeToStringDeepOptions{WrapWidth: ptr(100)}),
		"objects:\n"+
			"  Rect.fromLTRB(0.0, 0.0, 20.0, 20.0)\n"+
			"  "+white+"\n")
	validateIterablePropertyJsonSerialization(t, multiLineProperty)

	equalsIgnoringHashCodes(t, newTestTree(tt{Properties: []DiagnosticsNode{multiLineProperty}}).
		ToStringDeep(DiagnosticableTreeToStringDeepOptions{WrapWidth: ptr(100)}),
		"TestTree#00000\n"+
			"   objects:\n"+
			"     Rect.fromLTRB(0.0, 0.0, 20.0, 20.0)\n"+
			"     "+white+"\n", "multi line property in a tree")

	equalsIgnoringHashCodes(t, newTestTree(tt{
		Properties: []DiagnosticsNode{objectsProperty, NewIntProperty("foo", 42)},
		Style:      st(DiagnosticsTreeStyleSingleLine),
	}).ToStringDeep(),
		"TestTree#00000(objects: [Rect.fromLTRB(0.0, 0.0, 20.0, 20.0), "+white+"], foo: 42)", "single line tree")

	// Iterable with a single entry. Verify that rendering is sensible and that
	// multi line rendering isn't used even though it is not helpful.
	singleElementList := []any{ui.NewColorFromARGB(255, 255, 255, 255)}

	objectProperty := NewIterableProperty("object", singleElementList, IterablePropertyOptions{Style: st(DiagnosticsTreeStyleWhitespace)})
	equal(t, len(objectProperty.TypedValue()), 1)
	equal(t, objectProperty.IsFiltered(DiagnosticLevelInfo), false)
	equal(t, objectProperty.String(),
		"object: "+white)
	equal(t, objectProperty.ToStringDeep(DiagnosticsNodeToStringDeepOptions{WrapWidth: ptr(100)}),
		"object: "+white+"\n")
	validateIterablePropertyJsonSerialization(t, objectProperty)
	equalsIgnoringHashCodes(t, newTestTree(tt{Name: "root", Properties: []DiagnosticsNode{objectProperty}}).
		ToStringDeep(DiagnosticableTreeToStringDeepOptions{WrapWidth: ptr(120)}),
		"TestTree#00000\n"+
			"   object: "+white+"\n", "single element")
}

func TestStackTrace(t *testing.T) {
	stack := dart.StackTraceFromString(
		"#0      someMethod  (file:///diagnostics_test.dart:42:19)\n" +
			"#1      someMethod2  (file:///diagnostics_test.dart:12:3)\n" +
			"#2      someMethod3  (file:///foo.dart:4:1)\n",
	)

	equalsIgnoringHashCodes(t, NewDiagnosticsStackTrace("Stack trace", stack).ToStringDeep(),
		"Stack trace:\n"+
			"#0      someMethod  (file:///diagnostics_test.dart:42:19)\n"+
			"#1      someMethod2  (file:///diagnostics_test.dart:12:3)\n"+
			"#2      someMethod3  (file:///foo.dart:4:1)\n", "stack trace")

	equalsIgnoringHashCodes(t, NewDiagnosticsStackTrace("-- callback 2 --", stack,
		DiagnosticsStackTraceOptions{ShowSeparator: ptr(false)}).ToStringDeep(),
		"-- callback 2 --\n"+
			"#0      someMethod  (file:///diagnostics_test.dart:42:19)\n"+
			"#1      someMethod2  (file:///diagnostics_test.dart:12:3)\n"+
			"#2      someMethod3  (file:///foo.dart:4:1)\n", "no separator")
}

func TestMessage(t *testing.T) {
	message := NewDiagnosticsNodeMessage("hello world")
	equal(t, message.String(), "hello world")
	equal(t, message.Name(), "")
	_, nonNull := nodeName(message)
	equal(t, nonNull, true) // Dart: name is '' (isEmpty), not null
	equal(t, message.Value(), nil)
	equal(t, message.ShowName(), false)
	validateNodeJsonSerialization(t, message)

	messageProperty := NewMessageProperty("diagnostics", "hello world")
	equal(t, messageProperty.String(), "diagnostics: hello world")
	equal(t, messageProperty.Name(), "diagnostics")
	equal(t, messageProperty.Value(), nil)
	equal(t, messageProperty.ShowName(), true)
	validatePropertyJsonSerialization(t, &messageProperty.DiagnosticsProperty)
}

func TestErrorMessageStyleWrap(t *testing.T) {
	// This tests wrapping of properties with styles typical for error messages.
	createTreeWithWrappingNodes := func(rootStyle *DiagnosticsTreeStyle, propertyStyle DiagnosticsTreeStyle) DiagnosticsNode {
		ps := &propertyStyle
		msg := func(m string) DiagnosticsNode {
			return NewDiagnosticsNodeMessage(m, DiagnosticsNodeMessageOptions{Style: ps})
		}
		return newTestTree(tt{
			Name: "Test tree",
			Properties: []DiagnosticsNode{
				msg("--- example property at max length --"),
				msg("This is a very long message that must wrap as it cannot fit on one line. " +
					"This is a very long message that must wrap as it cannot fit on one line. " +
					"This is a very long message that must wrap as it cannot fit on one line."),
				msg("--- example property at max length --"),
				NewDiagnosticsProperty("",
					"Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap.",
					DiagnosticsPropertyOptions{AllowWrap: ptr(false)}),
				msg("--- example property at max length --"),
				msg("This property has a very long property name that will be allowed to wrap unlike most property names. This property has a very long property name that will be allowed to wrap unlike most property names:\n" +
					"  http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html"),
				msg("This property has a very long property name that will be allowed to wrap unlike most property names. This property has a very long property name that will be allowed to wrap unlike most property names:\n" +
					"  https://goo.gl/"),
				msg("Click on the following url:\n" +
					"  http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html"),
				msg("Click on the following url\n" +
					"  https://goo.gl/"),
				msg("--- example property at max length --"),
				NewDiagnosticsProperty("multi-line value",
					"[1.0, 0.0, 0.0, 0.0]\n"+
						"[1.0, 1.0, 0.0, 0.0]\n"+
						"[1.0, 0.0, 1.0, 0.0]\n"+
						"[1.0, 0.0, 0.0, 1.0]\n",
					DiagnosticsPropertyOptions{Style: ps}),
				msg("--- example property at max length --"),
				NewDiagnosticsProperty("This property has a very long property name that will be allowed to wrap unlike most property names. This property has a very long property name that will be allowed to wrap unlike most property names",
					"This is a very long message that must wrap as it cannot fit on one line. "+
						"This is a very long message that must wrap as it cannot fit on one line. "+
						"This is a very long message that must wrap as it cannot fit on one line.",
					DiagnosticsPropertyOptions{Style: ps}),
				msg("--- example property at max length --"),
				NewDiagnosticsProperty("This property has a very long property name that will be allowed to wrap unlike most property names. This property has a very long property name that will be allowed to wrap unlike most property names",
					"[1.0, 0.0, 0.0, 0.0]\n"+
						"[1.0, 1.0, 0.0, 0.0]\n"+
						"[1.0, 0.0, 1.0, 0.0]\n"+
						"[1.0, 0.0, 0.0, 1.0]\n",
					DiagnosticsPropertyOptions{Style: ps}),
				msg("--- example property at max length --"),
				NewMessageProperty("diagnosis", "insufficient data to draw conclusion (less than five repaints)", MessagePropertyOptions{Style: ps}),
			},
		}).ToDiagnosticsNode(ToDiagnosticsNodeOptions{Style: rootStyle})
	}

	renderer := NewTextTreeRenderer(TextTreeRendererOptions{WrapWidth: ptr(40), WrapWidthProperties: ptr(40)})
	equalsIgnoringHashCodes(t, renderer.Render(createTreeWithWrappingNodes(st(DiagnosticsTreeStyleError), DiagnosticsTreeStyleSingleLine)),
		"══╡ TESTTREE#00000 ╞════════════════════\n"+
			"--- example property at max length --\n"+
			"This is a very long message that must\n"+
			"wrap as it cannot fit on one line. This\n"+
			"is a very long message that must wrap as\n"+
			"it cannot fit on one line. This is a\n"+
			"very long message that must wrap as it\n"+
			"cannot fit on one line.\n"+
			"--- example property at max length --\n"+
			"Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap.\n"+
			"--- example property at max length --\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  https://goo.gl/\n"+
			"Click on the following url:\n"+
			"  http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html\n"+
			"Click on the following url\n"+
			"  https://goo.gl/\n"+
			"--- example property at max length --\n"+
			"multi-line value:\n"+
			"  [1.0, 0.0, 0.0, 0.0]\n"+
			"  [1.0, 1.0, 0.0, 0.0]\n"+
			"  [1.0, 0.0, 1.0, 0.0]\n"+
			"  [1.0, 0.0, 0.0, 1.0]\n"+
			"--- example property at max length --\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  This is a very long message that must\n"+
			"  wrap as it cannot fit on one line.\n"+
			"  This is a very long message that must\n"+
			"  wrap as it cannot fit on one line.\n"+
			"  This is a very long message that must\n"+
			"  wrap as it cannot fit on one line.\n"+
			"--- example property at max length --\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  [1.0, 0.0, 0.0, 0.0]\n"+
			"  [1.0, 1.0, 0.0, 0.0]\n"+
			"  [1.0, 0.0, 1.0, 0.0]\n"+
			"  [1.0, 0.0, 0.0, 1.0]\n"+
			"--- example property at max length --\n"+
			"diagnosis: insufficient data to draw\n"+
			"  conclusion (less than five repaints)\n"+
			"════════════════════════════════════════\n", "error / singleLine")

	// This output looks ugly but verifies that no indentation on word wrap
	// leaks in if the style is flat.
	equalsIgnoringHashCodes(t, renderer.Render(createTreeWithWrappingNodes(st(DiagnosticsTreeStyleSparse), DiagnosticsTreeStyleFlat)),
		"TestTree#00000\n"+
			"   --- example property at max length --\n"+
			"   This is a very long message that must\n"+
			"   wrap as it cannot fit on one line. This\n"+
			"   is a very long message that must wrap as\n"+
			"   it cannot fit on one line. This is a\n"+
			"   very long message that must wrap as it\n"+
			"   cannot fit on one line.\n"+
			"   --- example property at max length --\n"+
			"   Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap.\n"+
			"   --- example property at max length --\n"+
			"   This property has a very long property\n"+
			"   name that will be allowed to wrap unlike\n"+
			"   most property names. This property has a\n"+
			"   very long property name that will be\n"+
			"   allowed to wrap unlike most property\n"+
			"   names:\n"+
			"     http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html\n"+
			"   This property has a very long property\n"+
			"   name that will be allowed to wrap unlike\n"+
			"   most property names. This property has a\n"+
			"   very long property name that will be\n"+
			"   allowed to wrap unlike most property\n"+
			"   names:\n"+
			"     https://goo.gl/\n"+
			"   Click on the following url:\n"+
			"     http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html\n"+
			"   Click on the following url\n"+
			"     https://goo.gl/\n"+
			"   --- example property at max length --\n"+
			"   multi-line value:\n"+
			"   [1.0, 0.0, 0.0, 0.0]\n"+
			"   [1.0, 1.0, 0.0, 0.0]\n"+
			"   [1.0, 0.0, 1.0, 0.0]\n"+
			"   [1.0, 0.0, 0.0, 1.0]\n"+
			"   --- example property at max length --\n"+
			"   This property has a very long property\n"+
			"   name that will be allowed to wrap unlike\n"+
			"   most property names. This property has a\n"+
			"   very long property name that will be\n"+
			"   allowed to wrap unlike most property\n"+
			"   names:\n"+
			"   This is a very long message that must\n"+
			"   wrap as it cannot fit on one line. This\n"+
			"   is a very long message that must wrap as\n"+
			"   it cannot fit on one line. This is a\n"+
			"   very long message that must wrap as it\n"+
			"   cannot fit on one line.\n"+
			"   --- example property at max length --\n"+
			"   This property has a very long property\n"+
			"   name that will be allowed to wrap unlike\n"+
			"   most property names. This property has a\n"+
			"   very long property name that will be\n"+
			"   allowed to wrap unlike most property\n"+
			"   names:\n"+
			"   [1.0, 0.0, 0.0, 0.0]\n"+
			"   [1.0, 1.0, 0.0, 0.0]\n"+
			"   [1.0, 0.0, 1.0, 0.0]\n"+
			"   [1.0, 0.0, 0.0, 1.0]\n"+
			"   --- example property at max length --\n"+
			"   diagnosis: insufficient data to draw\n"+
			"   conclusion (less than five repaints)\n", "sparse / flat")

	// This case matches the styles that should generally be used for error
	// messages
	equalsIgnoringHashCodes(t, renderer.Render(createTreeWithWrappingNodes(st(DiagnosticsTreeStyleError), DiagnosticsTreeStyleErrorProperty)),
		"══╡ TESTTREE#00000 ╞════════════════════\n"+
			"--- example property at max length --\n"+
			"This is a very long message that must\n"+
			"wrap as it cannot fit on one line. This\n"+
			"is a very long message that must wrap as\n"+
			"it cannot fit on one line. This is a\n"+
			"very long message that must wrap as it\n"+
			"cannot fit on one line.\n"+
			"--- example property at max length --\n"+
			"Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap even though it is very long. Message that is not allowed to wrap.\n"+
			"--- example property at max length --\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  https://goo.gl/\n"+
			"Click on the following url:\n"+
			"  http://someverylongurl.com/that-might-be-tempting-to-wrap-even-though-it-is-a-url-so-should-not-wrap.html\n"+
			"Click on the following url\n"+
			"  https://goo.gl/\n"+
			"--- example property at max length --\n"+
			"multi-line value:\n"+
			"  [1.0, 0.0, 0.0, 0.0]\n"+
			"  [1.0, 1.0, 0.0, 0.0]\n"+
			"  [1.0, 0.0, 1.0, 0.0]\n"+
			"  [1.0, 0.0, 0.0, 1.0]\n"+
			"--- example property at max length --\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  This is a very long message that must\n"+
			"  wrap as it cannot fit on one line.\n"+
			"  This is a very long message that must\n"+
			"  wrap as it cannot fit on one line.\n"+
			"  This is a very long message that must\n"+
			"  wrap as it cannot fit on one line.\n"+
			"--- example property at max length --\n"+
			"This property has a very long property\n"+
			"name that will be allowed to wrap unlike\n"+
			"most property names. This property has a\n"+
			"very long property name that will be\n"+
			"allowed to wrap unlike most property\n"+
			"names:\n"+
			"  [1.0, 0.0, 0.0, 0.0]\n"+
			"  [1.0, 1.0, 0.0, 0.0]\n"+
			"  [1.0, 0.0, 1.0, 0.0]\n"+
			"  [1.0, 0.0, 0.0, 1.0]\n"+
			"--- example property at max length --\n"+
			"diagnosis:\n"+
			"  insufficient data to draw conclusion\n"+
			"  (less than five repaints)\n"+
			"════════════════════════════════════════\n", "error / errorProperty")
}

func TestDiagnosticsPropertyBasicTypesHaveValueInJson(t *testing.T) {
	check := func(node DiagnosticsNode, name string, value any) {
		t.Helper()
		j := simulateJsonSerialization(t, node)
		equal[any](t, j["name"], name)
		equal(t, j["value"], value)
	}
	check(NewDiagnosticsProperty("int1", 10), "int1", any(10.0))
	check(NewIntProperty("int2", 20), "int2", any(20.0))
	check(NewDiagnosticsProperty("double", 33.3), "double", any(33.3))
	check(NewDoubleProperty("double2", 33.3), "double2", any(33.3))
	check(NewDiagnosticsProperty("bool", true), "bool", any(true))
	check(NewDiagnosticsProperty("string1", "hello"), "string1", any("hello"))
	check(NewStringProperty("string2", "world"), "string2", any("world"))
}

func TestIntPropertyArgumentsPassedToSuper(t *testing.T) {
	property := NewIntProperty("Example", 0, IntPropertyOptions{
		IfNull:       ptr("is null"),
		ShowName:     ptr(false),
		DefaultValue: 1,
		Style:        st(DiagnosticsTreeStyleNone),
		Level:        lv(DiagnosticLevelOff),
	})
	equal(t, *property.TypedValue(), 0)
	equal(t, *property.IfNull, "is null")
	equal(t, property.ShowName(), false)
	equal(t, property.DefaultValue, any(1))
	equal(t, property.Style(), DiagnosticsTreeStyleNone)
	equal(t, property.Level(), DiagnosticLevelOff)
}

// Tests beyond Flutter's diagnostics_test.dart, for parts it only covers
// indirectly.

func TestTextTreeConfigurationForStyle(t *testing.T) {
	want := map[DiagnosticsTreeStyle]*TextTreeConfiguration{
		DiagnosticsTreeStyleNone:             nil,
		DiagnosticsTreeStyleDense:            DenseTextConfiguration,
		DiagnosticsTreeStyleSparse:           SparseTextConfiguration,
		DiagnosticsTreeStyleOffstage:         DashedTextConfiguration,
		DiagnosticsTreeStyleWhitespace:       WhitespaceTextConfiguration,
		DiagnosticsTreeStyleTransition:       TransitionTextConfiguration,
		DiagnosticsTreeStyleSingleLine:       SingleLineTextConfiguration,
		DiagnosticsTreeStyleErrorProperty:    ErrorPropertyTextConfiguration,
		DiagnosticsTreeStyleShallow:          ShallowTextConfiguration,
		DiagnosticsTreeStyleError:            ErrorTextConfiguration,
		DiagnosticsTreeStyleFlat:             FlatTextConfiguration,
		DiagnosticsTreeStyleTruncateChildren: WhitespaceTextConfiguration,
	}
	for style, config := range want {
		block := NewDiagnosticsBlock(DiagnosticsBlockOptions{Style: st(style)})
		if got := block.TextTreeConfiguration(); got != config {
			t.Errorf("%v: wrong configuration", style)
		}
	}
	// childLinkSpace is as long as linkCharacter in UTF-16 units, not bytes.
	equal(t, SparseTextConfiguration.ChildLinkSpace, " ")
	equal(t, FlatTextConfiguration.ChildLinkSpace, "")
	c := NewTextTreeConfiguration(TextTreeConfigurationOptions{LinkCharacter: "😀"})
	equal(t, c.ChildLinkSpace, "  ")
	equal(t, c.LineBreak, "\n")
	equal(t, c.AfterName, ":")
	equal(t, c.ShowChildren, true)
}

func TestTextTreeRendererDefaults(t *testing.T) {
	// minLevel debug: a fine property is filtered, a debug one shown.
	node := NewDiagnosticsBlock(DiagnosticsBlockOptions{Name: ptr("block"), Properties: []DiagnosticsNode{
		NewIntProperty("fine", 1, IntPropertyOptions{Level: lv(DiagnosticLevelFine)}),
		NewIntProperty("debug", 2, IntPropertyOptions{Level: lv(DiagnosticLevelDebug)}),
	}})
	equal(t, NewTextTreeRenderer().Render(node), "block:\n  debug: 2\n")
	equal(t, NewTextTreeRenderer(TextTreeRendererOptions{MinLevel: lv(DiagnosticLevelHidden)}).Render(node), "block:\n  fine: 1\n  debug: 2\n")
}

func TestShortHashAndDescribeIdentity(t *testing.T) {
	tree := newTestTree(tt{})
	h := ShortHash(tree)
	if !regexp.MustCompile(`^[0-9a-f]{5}$`).MatchString(h) || ShortHash(tree) != h {
		t.Errorf("ShortHash = %q, want five stable hex digits", h)
	}
	equal(t, DescribeIdentity(tree), "TestTree#"+h)
	equal(t, tree.ToStringShort(), "TestTree#"+h)
}

func TestDebugFormatDouble(t *testing.T) {
	equal(t, DebugFormatDouble(nil), "null")
	equal(t, DebugFormatDouble(ptr(0.25)), "0.3") // Dart rounds ties away from zero
	equal(t, DebugFormatDouble(ptr(-0.04)), "-0.0")
	equal(t, DebugFormatDouble(ptr(1e21)), "1e+21")
}

func TestKNoDefaultValueAndNull(t *testing.T) {
	p := NewDiagnosticsProperty("x", 1)
	equal(t, p.DefaultValue, KNoDefaultValue)
	equal(t, NewDiagnosticsProperty("x", 1, DiagnosticsPropertyOptions{DefaultValue: Null}).DefaultValue, nil)
	equal(t, NewFlagProperty("f", true, FlagPropertyOptions{IfTrue: ptr("f")}).DefaultValue, nil) // FlagProperty defaults to null
	equal(t, *NewIterableProperty("l", []int{}).IfEmpty, "[]")
	equal(t, NewIterableProperty("l", []int{}, IterablePropertyOptions{IfEmpty: NullString}).IfEmpty, (*string)(nil))
}

func TestComputePropertyValueCallback(t *testing.T) {
	calls := 0
	var compute ComputePropertyValueCallback[*int] = func() *int { calls++; return ptr(7) }
	p := NewDiagnosticsPropertyLazy("lazy", compute)
	equal(t, calls, 0)
	equal(t, p.String(), "lazy: 7")
	equal(t, p.String(), "lazy: 7")
	equal(t, calls, 1) // computed once, then cached
}

func TestDiagnosticableNodesAndMessageProperty(t *testing.T) {
	tree := newTestTree(tt{Children: []*TestTree{newTestTree(tt{Name: "c"})}})
	node := tree.ToDiagnosticsNode()
	if _, ok := node.(*DiagnosticableTreeNode); !ok {
		t.Fatalf("DiagnosticableTree.toDiagnosticsNode is %T", node)
	}
	equal(t, len(node.GetChildren()), 1)
	plain := NewDiagnosticableNode[Diagnosticable](tree)
	equal(t, len(plain.GetChildren()), 0) // DiagnosticableNode has no children
	equal(t, plain.Style(), DiagnosticsTreeStyleSparse)
	var _ DiagnosticsNode = (*MessageProperty)(nil)
	var _ DiagnosticsNode = (*EnumProperty[ExampleEnum])(nil)
	var _ DiagnosticsNode = (*IntProperty)(nil)
	equal(t, runtimeTypeName(NewMessageProperty("m", "x")), "MessageProperty")
	equal(t, runtimeTypeName(NewDiagnosticsNodeMessage("x")), "DiagnosticsProperty<void>")
	equal(t, runtimeTypeName(plain), "DiagnosticableNode<Diagnosticable>")
	equal(t, runtimeTypeName(NewEnumProperty[*ExampleEnum]("e", nil)), "EnumProperty<ExampleEnum>")
}
