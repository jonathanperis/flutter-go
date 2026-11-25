package difftest

// Go twin of flutter/test/diagnostics_test.dart: the same diagnostic trees,
// whose text art, toString and JSON must equal Flutter's exactly (object
// hashes, which differ between runs, are normalized to #00000 on both
// sides; JSON is compared as decoded values, so key order and 42 vs 42.0
// don't matter).

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	f "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

func ptr[T any](v T) *T { return &v }

func st(s f.DiagnosticsTreeStyle) *f.DiagnosticsTreeStyle { return &s }
func lv(l f.DiagnosticLevel) *f.DiagnosticLevel           { return &l }

type Fruit int

const (
	FruitApple Fruit = iota
	FruitBananaSplit
)

func (x Fruit) String() string { return [...]string{"apple", "bananaSplit"}[x] }

// Point2 is a Diagnosticable without children.
type Point2 struct {
	f.DiagnosticableMixin
	x, y float64
}

func newPoint2(x, y float64) *Point2 {
	p := &Point2{x: x, y: y}
	p.SetSelf(p)
	return p
}

func (p *Point2) DebugFillProperties(properties *f.DiagnosticPropertiesBuilder) {
	p.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(f.NewDoubleProperty("x", p.x))
	properties.Add(f.NewDoubleProperty("y", p.y, f.DoublePropertyOptions{DefaultValue: 0.0}))
}

// SceneTree is a DiagnosticableTreeMixin tree.
type SceneTree struct {
	f.DiagnosticableTreeMixin
	label      string
	style      *f.DiagnosticsTreeStyle
	children   []*SceneTree
	properties []f.DiagnosticsNode
	emptyBody  *string
}

type sceneArgs struct {
	Style      *f.DiagnosticsTreeStyle
	Children   []*SceneTree
	Properties []f.DiagnosticsNode
	EmptyBody  *string
}

func newSceneTree(label string, a ...sceneArgs) *SceneTree {
	t := &SceneTree{label: label}
	if len(a) > 0 {
		t.style, t.children, t.properties, t.emptyBody = a[0].Style, a[0].Children, a[0].Properties, a[0].EmptyBody
	}
	t.SetSelf(t)
	return t
}

func (t *SceneTree) DebugDescribeChildren() []f.DiagnosticsNode {
	out := []f.DiagnosticsNode{}
	for _, c := range t.children {
		out = append(out, c.ToDiagnosticsNode(f.ToDiagnosticsNodeOptions{Name: ptr(c.label), Style: c.style}))
	}
	return out
}

func (t *SceneTree) DebugFillProperties(properties *f.DiagnosticPropertiesBuilder) {
	t.DiagnosticableTreeMixin.DebugFillProperties(properties)
	if t.style != nil {
		properties.DefaultDiagnosticsTreeStyle = *t.style
	}
	if t.emptyBody != nil {
		properties.EmptyBodyDescription = t.emptyBody
	}
	for _, p := range t.properties {
		properties.Add(p)
	}
}

// PlainTree extends the abstract DiagnosticableTree (not the mixin).
type PlainTree struct {
	f.DiagnosticableTreeBase
	properties []f.DiagnosticsNode
}

func newPlainTree(properties []f.DiagnosticsNode) *PlainTree {
	t := &PlainTree{properties: properties}
	t.SetSelf(t)
	return t
}

func (t *PlainTree) DebugFillProperties(properties *f.DiagnosticPropertiesBuilder) {
	t.DiagnosticableTreeBase.DebugFillProperties(properties)
	for _, p := range t.properties {
		properties.Add(p)
	}
}

var pointOne, pointTwo = 0.1, 0.2

func onTap()  {}
func onDrag() {}

func allKinds() []f.DiagnosticsNode {
	type fn = func()
	return []f.DiagnosticsNode{
		f.NewStringProperty("string", "hello"),
		f.NewStringProperty("unquoted", "plain", f.StringPropertyOptions{Quoted: ptr(false)}),
		f.NewStringProperty("multiline", "one\ntwo"),
		f.NewStringProperty("empty", "", f.StringPropertyOptions{IfEmpty: ptr("<empty>")}),
		f.NewStringProperty[*string]("nullString", nil),
		f.NewStringProperty[*string]("nullHidden", nil, f.StringPropertyOptions{DefaultValue: f.Null}),
		f.NewStringProperty("described", "x", f.StringPropertyOptions{Description: ptr("custom"), Tooltip: ptr("tip")}),
		f.NewDoubleProperty("double", 1.25),
		f.NewDoubleProperty("tie", 0.25),
		f.NewDoubleProperty("belowTie", 0.35),
		f.NewDoubleProperty("negZero", -0.04),
		f.NewDoubleProperty("big", 1e21),
		f.NewDoubleProperty("nan", math.NaN()),
		f.NewDoubleProperty("inf", math.Inf(-1), f.DoublePropertyOptions{Unit: ptr("px")}),
		f.NewDoubleProperty[*float64]("nullDouble", nil, f.DoublePropertyOptions{IfNull: ptr("unset")}),
		f.NewDoubleProperty("defaultDouble", 1.0, f.DoublePropertyOptions{DefaultValue: 1.0}),
		f.NewDoubleProperty("tooltipDouble", 2.5, f.DoublePropertyOptions{Tooltip: ptr("logical pixels")}),
		f.NewDoublePropertyLazy("lazy", func() float64 { return 3.75 }, f.DoublePropertyOptions{Unit: ptr("ms")}),
		f.NewDoublePropertyLazy("throws", func() float64 { panic(f.NewFlutterError("nope")) }),
		f.NewIntProperty("int", 42),
		f.NewIntProperty("intDefault", 42, f.IntPropertyOptions{DefaultValue: 42.0}),
		f.NewIntProperty("intUnit", -7, f.IntPropertyOptions{Unit: ptr("px")}),
		f.NewIntProperty[*int]("nullInt", nil),
		f.NewPercentProperty("percent", 0.1234),
		f.NewPercentProperty("percentClamped", 1.5, f.PercentPropertyOptions{Unit: ptr("opacity")}),
		f.NewPercentProperty("percentNegZero", math.Copysign(0, -1)),
		f.NewPercentProperty("percentTie", 0.0025, f.PercentPropertyOptions{ShowName: ptr(false)}),
		f.NewFlagProperty("visible", true, f.FlagPropertyOptions{IfTrue: ptr("visible"), IfFalse: ptr("hidden")}),
		f.NewFlagProperty("enabled", false, f.FlagPropertyOptions{IfTrue: ptr("enabled")}),
		f.NewFlagProperty("checked", false, f.FlagPropertyOptions{IfFalse: ptr("unchecked"), ShowName: ptr(true)}),
		f.NewFlagProperty[*bool]("nullFlag", nil, f.FlagPropertyOptions{IfTrue: ptr("on")}),
		f.NewIterableProperty("ints", []int{1, 2, 3}),
		f.NewIterableProperty("doubles", []float64{0.25, 1.0 / 3.0, 2}),
		f.NewIterableProperty("emptyList", []string{}),
		f.NewIterableProperty("emptyNoIfEmpty", []string{}, f.IterablePropertyOptions{IfEmpty: f.NullString}),
		f.NewIterableProperty[string]("nullList", nil),
		f.NewIterableProperty("lines", []string{"a", "b"}, f.IterablePropertyOptions{Style: st(f.DiagnosticsTreeStyleWhitespace)}),
		f.NewEnumProperty("fruit", FruitBananaSplit),
		f.NewEnumProperty("defaultFruit", FruitApple, f.EnumPropertyOptions{DefaultValue: FruitApple}),
		f.NewEnumProperty[*Fruit]("nullFruit", nil),
		f.NewObjectFlagProperty[fn]("onTap", onTap, f.ObjectFlagPropertyOptions{IfPresent: ptr("tappable")}),
		f.NewObjectFlagProperty[fn]("onDrag", nil, f.ObjectFlagPropertyOptions{IfNull: ptr("not draggable")}),
		f.NewObjectFlagProperty[fn]("onHidden", onDrag, f.ObjectFlagPropertyOptions{IfNull: ptr("no drag")}),
		f.NewObjectFlagPropertyHas[fn]("callback", onTap),
		f.NewFlagsSummary("listeners", []dart.MapEntry[string, fn]{{Key: "tap", Value: onTap}, {Key: "drag", Value: nil}, {Key: "hover", Value: onDrag}}),
		f.NewFlagsSummary("noListeners", []dart.MapEntry[string, fn]{{Key: "tap", Value: nil}}, f.FlagsSummaryOptions{IfEmpty: ptr("<none>")}),
		f.NewMessageProperty("note", "a message"),
		f.NewDiagnosticsNodeMessage("a bare message"),
		f.NewDiagnosticsProperty("bool", true),
		f.NewDiagnosticsProperty[*bool]("nullBool", nil, f.DiagnosticsPropertyOptions{MissingIfNull: true}),
		f.NewDiagnosticsProperty[any]("object", nil, f.DiagnosticsPropertyOptions{IfNull: ptr("nothing")}),
		f.NewDiagnosticsProperty("rawDouble", pointOne+pointTwo), // not a constant: Go folds 0.1+0.2 exactly
		f.NewDiagnosticsProperty("tinyDouble", 1.5e-7),
		f.NewDiagnosticsProperty("rawInt", 10),
		f.NewDiagnosticsProperty("list", []int{4, 5}),
		f.NewDiagnosticsProperty("rect", ui.NewRectFromLTRB(1, 2, 3.5, 4)),
		f.NewDiagnosticsProperty("point", newPoint2(1, 2)),
		f.NewDiagnosticsProperty("expanded", newPoint2(3, 0), f.DiagnosticsPropertyOptions{ExpandableValue: true}),
		f.NewDiagnosticsProperty("treeValue", newSceneTree("inner")),
		f.NewDiagnosticsProperty("noSeparator", "value", f.DiagnosticsPropertyOptions{ShowSeparator: ptr(false)}),
		f.NewDiagnosticsProperty("warn", "careful", f.DiagnosticsPropertyOptions{Level: lv(f.DiagnosticLevelWarning)}),
		f.NewDiagnosticsProperty("fine", "detail", f.DiagnosticsPropertyOptions{Level: lv(f.DiagnosticLevelFine)}),
		f.NewDiagnosticsProperty("debugOnly", "debug", f.DiagnosticsPropertyOptions{Level: lv(f.DiagnosticLevelDebug)}),
		f.NewDiagnosticsProperty("hidden", "secret", f.DiagnosticsPropertyOptions{Level: lv(f.DiagnosticLevelHidden)}),
		f.NewDiagnosticsProperty("err", "bad", f.DiagnosticsPropertyOptions{Level: lv(f.DiagnosticLevelError)}),
		f.NewDiagnosticsPropertyLazy("lazyThrows", func() string { panic(f.NewFlutterError("x")) }, f.DiagnosticsPropertyOptions{Description: ptr("had description")}),
		f.NewStringProperty("emptyUnquoted", "", f.StringPropertyOptions{Quoted: ptr(false), IfEmpty: ptr("<none>")}),
		f.NewDiagnosticsProperty("emptyRaw", "", f.DiagnosticsPropertyOptions{IfEmpty: ptr("<blank>")}),
	}
}

func bigTree() *SceneTree {
	return newSceneTree("root", sceneArgs{
		Properties: allKinds(),
		Children: []*SceneTree{
			newSceneTree("a", sceneArgs{Properties: []f.DiagnosticsNode{f.NewIntProperty("depth", 1)}}),
			newSceneTree("b", sceneArgs{
				Style: st(f.DiagnosticsTreeStyleOffstage),
				Children: []*SceneTree{
					newSceneTree("b1", sceneArgs{Properties: []f.DiagnosticsNode{f.NewStringProperty("s", "v")}}),
					newSceneTree("b2", sceneArgs{Style: st(f.DiagnosticsTreeStyleDense), Properties: []f.DiagnosticsNode{
						f.NewIntProperty("n", 2), f.NewFlagProperty("f", true, f.FlagPropertyOptions{IfTrue: ptr("yes")})}}),
				},
			}),
			newSceneTree("c", sceneArgs{EmptyBody: ptr("<no props>")}),
		},
	})
}

var treeStyles = []f.DiagnosticsTreeStyle{
	f.DiagnosticsTreeStyleSparse,
	f.DiagnosticsTreeStyleOffstage,
	f.DiagnosticsTreeStyleDense,
	f.DiagnosticsTreeStyleTransition,
	f.DiagnosticsTreeStyleError,
	f.DiagnosticsTreeStyleWhitespace,
	f.DiagnosticsTreeStyleFlat,
	f.DiagnosticsTreeStyleSingleLine,
	f.DiagnosticsTreeStyleErrorProperty,
	f.DiagnosticsTreeStyleShallow,
	f.DiagnosticsTreeStyleTruncateChildren,
}

func styledTree(style f.DiagnosticsTreeStyle) *SceneTree {
	return newSceneTree("styled", sceneArgs{
		Style: &style,
		Properties: []f.DiagnosticsNode{
			f.NewStringProperty("name", "Größe straße"),
			f.NewDoubleProperty("width", 10.5, f.DoublePropertyOptions{Unit: ptr("px")}),
			f.NewStringProperty("multi", "first\nsecond", f.StringPropertyOptions{Quoted: ptr(false)}),
		},
		Children: []*SceneTree{
			newSceneTree("first", sceneArgs{Style: &style, Properties: []f.DiagnosticsNode{f.NewIntProperty("i", 1)}}),
			newSceneTree("second", sceneArgs{
				Style: &style,
				Children: []*SceneTree{
					newSceneTree("leaf1", sceneArgs{Style: &style}),
					newSceneTree("leaf2", sceneArgs{Style: st(f.DiagnosticsTreeStyleSparse), Properties: []f.DiagnosticsNode{
						f.NewFlagProperty("on", true, f.FlagPropertyOptions{IfTrue: ptr("on")})}}),
				},
			}),
			newSceneTree("third", sceneArgs{Style: st(f.DiagnosticsTreeStyleTransition), Properties: []f.DiagnosticsNode{
				f.NewMessageProperty("m", "transition child")}}),
		},
	})
}

const longText = "The overflowing RenderFlex has an orientation of Axis.horizontal. " +
	"The edge of the RenderFlex that is overflowing has been marked in the rendering " +
	"with a yellow and black striped pattern. Café — naïve 😀 emoji 😀😀 and 中文字符 text " +
	"to check UTF-16 widths. Averyveryveryveryveryveryveryveryverylongwordthatcannotwrap here."

func wrapTree(rootStyle, propertyStyle f.DiagnosticsTreeStyle) f.DiagnosticsNode {
	ps := &propertyStyle
	msg := func(m string) f.DiagnosticsNode {
		return f.NewDiagnosticsNodeMessage(m, f.DiagnosticsNodeMessageOptions{Style: ps})
	}
	return f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
		Name:        ptr("Wrapping straße"),
		Style:       &rootStyle,
		Description: "A block with long text",
		Properties: []f.DiagnosticsNode{
			msg(longText),
			f.NewDiagnosticsProperty("long property name that also wraps when it is long enough", longText, f.DiagnosticsPropertyOptions{Style: ps}),
			f.NewDiagnosticsProperty("", "no wrap: "+longText, f.DiagnosticsPropertyOptions{AllowWrap: ptr(false), Style: ps}),
			f.NewStringProperty("quoted", "short", f.StringPropertyOptions{Style: ps}),
			// A wrappable non-ASCII name before a value that must not wrap: the
			// wrappable range ends where the name ends, in UTF-16 units.
			f.NewDiagnosticsProperty("Größe 😀😀😀 straße", "a b c d e f g h no wrap: "+longText, f.DiagnosticsPropertyOptions{AllowWrap: ptr(false), Style: ps}),
			// A name that must not wrap before a value starting with a space: the
			// space opening the wrappable range is a break point.
			f.NewDiagnosticsProperty("a long property name that is not allowed to wrap at all",
				" value with one leading space and more words to wrap",
				f.DiagnosticsPropertyOptions{AllowNameWrap: ptr(false), Style: ps}),
			msg("  indented line\n    more indented\nback"),
		},
		Children: []f.DiagnosticsNode{
			f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
				Name:  ptr("child block"),
				Style: st(f.DiagnosticsTreeStyleError),
				Properties: []f.DiagnosticsNode{
					f.NewDiagnosticsNodeMessage(longText, f.DiagnosticsNodeMessageOptions{Style: st(f.DiagnosticsTreeStyleFlat)})},
			}),
			msg("final child message that is fairly long and will need to be wrapped somewhere"),
		},
	})
}

func truncTree() f.DiagnosticsNode {
	var deep func(level, width int) f.DiagnosticsNode
	deep = func(level, width int) f.DiagnosticsNode {
		var children []f.DiagnosticsNode
		for i := 0; level > 0 && i < width; i++ {
			children = append(children, deep(level-1, width))
		}
		return f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
			Name:          ptr(fmt.Sprintf("level %d", level)),
			Style:         st(f.DiagnosticsTreeStyleSparse),
			AllowTruncate: true,
			Properties:    []f.DiagnosticsNode{f.NewIntProperty("level", level)},
			Children:      children,
		})
	}
	return deep(3, 4)
}

type rootStyles struct{ root, prop f.DiagnosticsTreeStyle }

func buildDiagnosticsScenes() map[string]any {
	out := map[string]any{}

	tree := bigTree()
	out["big.toStringDeep"] = tree.ToStringDeep()
	out["big.toStringDeep.wrap40"] = tree.ToStringDeep(f.DiagnosticableTreeToStringDeepOptions{WrapWidth: ptr(40)})
	out["big.toStringDeep.prefixed"] = tree.ToStringDeep(f.DiagnosticableTreeToStringDeepOptions{PrefixLineOne: ">> ", PrefixOtherLines: ptr(".. ")})
	out["big.toString"] = tree.String()
	out["big.toString.debug"] = tree.ToString(f.DiagnosticableToStringOptions{MinLevel: lv(f.DiagnosticLevelDebug)})
	out["big.toStringShallow"] = tree.ToStringShallow()
	out["big.toStringShallow.joiner"] = tree.ToStringShallow(f.ToStringShallowOptions{Joiner: ptr("\n"), MinLevel: lv(f.DiagnosticLevelHidden)})
	out["big.toStringShort"] = tree.ToStringShort()
	for level := f.DiagnosticLevelHidden; level <= f.DiagnosticLevelOff; level++ {
		out["big.minLevel."+level.String()] = tree.ToStringDeep(f.DiagnosticableTreeToStringDeepOptions{MinLevel: lv(level)})
	}
	out["big.timeline"] = tree.ToDiagnosticsNode().ToTimelineArguments()
	out["big.json.default"] = tree.ToDiagnosticsNode().ToJsonMap(f.NewDiagnosticsSerializationDelegate())
	out["big.json.full"] = tree.ToDiagnosticsNode().ToJsonMap(f.NewDiagnosticsSerializationDelegate(
		f.DiagnosticsSerializationDelegateOptions{SubtreeDepth: 3, IncludeProperties: true}))
	out["big.json.iterative"] = tree.ToDiagnosticsNode().ToJsonMapIterative(f.NewDiagnosticsSerializationDelegate(
		f.DiagnosticsSerializationDelegateOptions{SubtreeDepth: 5}))

	for i, p := range allKinds() {
		key := fmt.Sprintf("prop.%d.", i)
		out[key+"toString"] = p.String()
		out[key+"dense"] = p.ToString(f.DiagnosticsNodeToStringOptions{ParentConfiguration: f.DenseTextConfiguration})
		out[key+"toStringDeep"] = p.ToStringDeep(f.DiagnosticsNodeToStringDeepOptions{MinLevel: lv(f.DiagnosticLevelHidden)})
		out[key+"level"] = p.Level().String()
		out[key+"showName"] = p.ShowName()
		out[key+"json"] = p.ToJsonMap(f.NewDiagnosticsSerializationDelegate(f.DiagnosticsSerializationDelegateOptions{IncludeProperties: true}))
	}

	for _, style := range treeStyles {
		t := styledTree(style)
		key := "style." + style.String() + "."
		out[key+"deep"] = t.ToStringDeep()
		out[key+"asRoot"] = t.ToDiagnosticsNode(f.ToDiagnosticsNodeOptions{Name: ptr("named root"), Style: &style}).ToStringDeep()
		out[key+"toString"] = t.ToDiagnosticsNode(f.ToDiagnosticsNodeOptions{Style: &style}).String()
		out[key+"json"] = t.ToDiagnosticsNode().ToJsonMap(f.NewDiagnosticsSerializationDelegate(f.DiagnosticsSerializationDelegateOptions{SubtreeDepth: 2}))
	}

	widths := []int{20, 40, 65, 100}
	for _, rs := range []rootStyles{
		{f.DiagnosticsTreeStyleError, f.DiagnosticsTreeStyleSingleLine},
		{f.DiagnosticsTreeStyleError, f.DiagnosticsTreeStyleErrorProperty},
		{f.DiagnosticsTreeStyleSparse, f.DiagnosticsTreeStyleFlat},
		{f.DiagnosticsTreeStyleWhitespace, f.DiagnosticsTreeStyleSingleLine},
		{f.DiagnosticsTreeStyleTransition, f.DiagnosticsTreeStyleShallow},
	} {
		for _, w := range widths {
			out[fmt.Sprintf("wrap.%s.%s.%d", rs.root, rs.prop, w)] =
				f.NewTextTreeRenderer(f.TextTreeRendererOptions{WrapWidth: ptr(w), WrapWidthProperties: ptr(w / 2)}).Render(wrapTree(rs.root, rs.prop))
		}
		out[fmt.Sprintf("wrap.%s.%s.deep", rs.root, rs.prop)] = wrapTree(rs.root, rs.prop).ToStringDeep()
	}

	for _, n := range []int{0, 1, 2, 5} {
		out[fmt.Sprintf("trunc.%d", n)] = f.NewTextTreeRenderer(f.TextTreeRendererOptions{MaxDescendentsTruncatableNode: ptr(n)}).Render(truncTree())
	}
	out["truncateChildren"] = f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
		Name:     ptr("Block"),
		Style:    st(f.DiagnosticsTreeStyleTruncateChildren),
		Children: []f.DiagnosticsNode{truncTree(), f.NewDiagnosticsNodeMessage("tail")},
	}).ToStringDeep()
	out["truncateChildren.oneChild"] = f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
		Style:    st(f.DiagnosticsTreeStyleTruncateChildren),
		Children: []f.DiagnosticsNode{f.NewDiagnosticsNodeMessage("only")},
	}).ToStringDeep()
	out["truncateChildren.none"] = f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{Name: ptr("Empty"), Style: st(f.DiagnosticsTreeStyleTruncateChildren)}).ToStringDeep()

	block := f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{
		Name:        ptr("Errors"),
		Style:       st(f.DiagnosticsTreeStyleError),
		LinePrefix:  "| ",
		Value:       7,
		Description: "Something went wrong",
		Level:       lv(f.DiagnosticLevelError),
		Properties: []f.DiagnosticsNode{
			f.NewDiagnosticsNodeMessage("Summary line.", f.DiagnosticsNodeMessageOptions{Level: lv(f.DiagnosticLevelSummary)}),
			f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{Name: ptr("Hint"), Style: st(f.DiagnosticsTreeStyleWhitespace), Properties: []f.DiagnosticsNode{
				f.NewDiagnosticsNodeMessage("Try this instead.", f.DiagnosticsNodeMessageOptions{Level: lv(f.DiagnosticLevelHint)})}}),
		},
		Children: []f.DiagnosticsNode{newSceneTree("culprit", sceneArgs{Properties: []f.DiagnosticsNode{f.NewIntProperty("id", 3)}}).
			ToDiagnosticsNode(f.ToDiagnosticsNodeOptions{Name: ptr("The culprit")})},
	})
	out["block.deep"] = block.ToStringDeep()
	out["block.toString"] = block.String()
	out["block.json"] = block.ToJsonMap(f.NewDiagnosticsSerializationDelegate(f.DiagnosticsSerializationDelegateOptions{SubtreeDepth: 2, IncludeProperties: true}))
	out["block.noName"] = f.NewDiagnosticsBlock(f.DiagnosticsBlockOptions{Description: "no name", Properties: []f.DiagnosticsNode{f.NewIntProperty("x", 1)}}).ToStringDeep()

	plain := newPlainTree([]f.DiagnosticsNode{f.NewIntProperty("a", 1), f.NewStringProperty("b", "two")})
	out["plain.toString"] = plain.String()
	out["plain.toStringShallow"] = plain.ToStringShallow()
	out["plain.toStringDeep"] = plain.ToStringDeep()
	point := newPoint2(1.5, 0)
	out["point.toString"] = point.String()
	out["point.toString.fine"] = point.ToString(f.DiagnosticableToStringOptions{MinLevel: lv(f.DiagnosticLevelFine)})
	out["point.deep"] = point.ToDiagnosticsNode(f.ToDiagnosticsNodeOptions{Name: ptr("pt")}).ToStringDeep()
	out["point.json"] = point.ToDiagnosticsNode().ToJsonMap(f.NewDiagnosticsSerializationDelegate(f.DiagnosticsSerializationDelegateOptions{IncludeProperties: true}))
	return out
}

var hashRE = regexp.MustCompile(`#[0-9a-fA-F]{5}`)

// normalizeHashes replaces object hashes in every string of a decoded JSON
// value.
func normalizeHashes(v any) any {
	switch x := v.(type) {
	case string:
		return hashRE.ReplaceAllString(x, "#00000")
	case []any:
		for i := range x {
			x[i] = normalizeHashes(x[i])
		}
	case map[string]any:
		for k := range x {
			x[k] = normalizeHashes(x[k])
		}
	}
	return v
}

func TestDiagnosticsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/diagnostics.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	goScenes := buildDiagnosticsScenes()
	goData, err := json.Marshal(goScenes)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(goData, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(ref) {
		t.Errorf("%d scenes in Go, %d in Flutter", len(got), len(ref))
	}
	keys := make([]string, 0, len(ref))
	for k := range ref {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	failures := 0
	for _, k := range keys {
		want := normalizeHashes(ref[k])
		g, ok := got[k]
		if !ok {
			t.Errorf("%s: missing in Go", k)
			continue
		}
		g = normalizeHashes(g)
		if reflect.DeepEqual(g, want) {
			continue
		}
		failures++
		if ws, ok := want.(string); ok {
			gs, _ := g.(string)
			t.Errorf("%s differs:\n--- Go\n%s\n--- Flutter\n%s\n--- first difference at line %d", k, gs, ws, firstDiffLine(gs, ws))
		} else {
			var diffs []string
			jsonDiff(k, g, want, &diffs)
			t.Errorf("%s differs (Go vs Flutter):\n%s", k, strings.Join(diffs[:min(len(diffs), 20)], "\n"))
		}
		if failures > 15 {
			t.Fatal("too many differences")
		}
	}
}

func firstDiffLine(a, b string) int {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := range min(len(al), len(bl)) {
		if al[i] != bl[i] {
			return i + 1
		}
	}
	return min(len(al), len(bl)) + 1
}

// jsonDiff lists the paths where two decoded JSON values differ.
func jsonDiff(path string, g, w any, out *[]string) {
	gm, gok := g.(map[string]any)
	wm, wok := w.(map[string]any)
	if gok && wok {
		keys := map[string]bool{}
		for k := range gm {
			keys[k] = true
		}
		for k := range wm {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			jsonDiff(path+"."+k, gm[k], wm[k], out)
		}
		return
	}
	gl, gok := g.([]any)
	wl, wok := w.([]any)
	if gok && wok && len(gl) == len(wl) {
		for i := range gl {
			jsonDiff(fmt.Sprintf("%s[%d]", path, i), gl[i], wl[i], out)
		}
		return
	}
	if !reflect.DeepEqual(g, w) {
		*out = append(*out, fmt.Sprintf("  %s: %#v vs %#v", path, g, w))
	}
}
