package difftest

// Go side of the semantics differential test: the same scenes as
// flutter/test/semantics_test.dart, logged in the same format, compared
// line by line exactly (hashes masked).

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var (
	semLog  []string
	semHash = regexp.MustCompile(`#[0-9a-f]{5}`)
)

func slogS(s string) { semLog = append(semLog, semHash.ReplaceAllString(s, "#HASH")) }

func dS(v float64) string { return dart.DoubleToString(v) }

func intsS(l []int32) string {
	parts := make([]string, len(l))
	for i, v := range l {
		parts[i] = strconv.Itoa(int(v))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func doublesS(l []float64) string {
	parts := make([]string, len(l))
	for i, v := range l {
		parts[i] = dS(v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func attrsS(l []ui.StringAttribute) string {
	parts := make([]string, len(l))
	for i, a := range l {
		parts[i] = a.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func listS(l []string) string { return "[" + strings.Join(l, ", ") + "]" }

func nullable[T any](p *T, f func(T) string) string {
	if p == nil {
		return "null"
	}
	return f(*p)
}

func strP(p *string) string { return nullable(p, func(s string) string { return s }) }

func boolP(p *bool) string { return nullable(p, strconv.FormatBool) }

func objS(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprint(v)
}

func dirS(d *ui.TextDirection) string { return nullable(d, ui.TextDirection.String) }

func localeS(l *ui.Locale) string { return nullable(l, ui.Locale.String) }

// spyBuilder is the Dart test's SpyBuilder.
type spyBuilder struct{ builder ui.SemanticsUpdateBuilder }

func (s *spyBuilder) UpdateNode(o ui.SemanticsUpdateBuilderUpdateNodeOptions) {
	controls := "null"
	if o.ControlsNodes != nil {
		controls = listS(o.ControlsNodes)
	}
	slogS("updateNode id=" + strconv.Itoa(o.ID) + " flags=" + listS(o.Flags.ToStrings()) + " actions=" + strconv.Itoa(o.Actions) +
		" rect=" + o.Rect.String() +
		` identifier="` + o.Identifier + `" label="` + o.Label + `" ` + attrsS(o.LabelAttributes) + ` value="` + o.Value + `" ` + attrsS(o.ValueAttributes) +
		` increased="` + o.IncreasedValue + `" ` + attrsS(o.IncreasedValueAttributes) + ` decreased="` + o.DecreasedValue + `" ` + attrsS(o.DecreasedValueAttributes) +
		` hint="` + o.Hint + `" ` + attrsS(o.HintAttributes) + ` tooltip="` + o.Tooltip + `" textDirection=` + dirS(o.TextDirection) +
		" selection=" + strconv.Itoa(o.TextSelectionBase) + "," + strconv.Itoa(o.TextSelectionExtent) + " platformViewId=" + strconv.Itoa(o.PlatformViewID) +
		" maxValueLength=" + strconv.Itoa(o.MaxValueLength) + " currentValueLength=" + strconv.Itoa(o.CurrentValueLength) +
		" scrollChildren=" + strconv.Itoa(o.ScrollChildren) + " scrollIndex=" + strconv.Itoa(o.ScrollIndex) +
		" scroll=" + dS(o.ScrollPosition) + "," + dS(o.ScrollExtentMax) + "," + dS(o.ScrollExtentMin) +
		" transform=" + doublesS(o.Transform) + " hitTestTransform=" + doublesS(o.HitTestTransform) +
		" traversal=" + intsS(o.ChildrenInTraversalOrder) + " hitTest=" + intsS(o.ChildrenInHitTestOrder) +
		" additionalActions=" + intsS(o.AdditionalActions) + " traversalParent=" + strconv.Itoa(o.TraversalParent) +
		" headingLevel=" + strconv.Itoa(o.HeadingLevel) + ` linkUrl="` + o.LinkURL + `" role=` + o.Role.String() + " controlsNodes=" + controls +
		" validationResult=" + o.ValidationResult.String() + " hitTestBehavior=" + o.HitTestBehavior.String() + " inputType=" + o.InputType.String() +
		" locale=" + localeS(o.Locale) + ` minValue="` + o.MinValue + `" maxValue="` + o.MaxValue + `"`)
}

func (s *spyBuilder) UpdateCustomAction(id int, opts ...ui.SemanticsUpdateBuilderUpdateCustomActionOptions) {
	o := opts[0]
	overrideID := -1
	if o.OverrideID != nil {
		overrideID = *o.OverrideID
	}
	slogS("updateCustomAction id=" + strconv.Itoa(id) + " label=" + strP(o.Label) + " hint=" + strP(o.Hint) +
		" overrideId=" + strconv.Itoa(overrideID))
	s.builder.UpdateCustomAction(id, opts...)
}

func (s *spyBuilder) Build() ui.SemanticsUpdate { return s.builder.Build() }

// semBinding is the Dart test's SemanticsDiffBinding (BindingBase with
// SchedulerBinding and SemanticsBinding, its builder the spy).
type semBinding struct {
	scheduler.SchedulerBindingMixin
	sem semantics.SemanticsBindingMixin
}

func (b *semBinding) InitInstances()         { b.sem.InitInstances() }
func (b *semBinding) SemanticsEnabled() bool { return b.sem.SemanticsEnabled() }
func (b *semBinding) AddSemanticsEnabledListener(l func()) {
	b.sem.AddSemanticsEnabledListener(l)
}
func (b *semBinding) RemoveSemanticsEnabledListener(l func()) {
	b.sem.RemoveSemanticsEnabledListener(l)
}
func (b *semBinding) AddSemanticsActionListener(l func(ui.SemanticsActionEvent)) {
	b.sem.AddSemanticsActionListener(l)
}
func (b *semBinding) RemoveSemanticsActionListener(l func(ui.SemanticsActionEvent)) {
	b.sem.RemoveSemanticsActionListener(l)
}
func (b *semBinding) GetRectOfSemanticsNodeInViewCoordinates(v, n int) *ui.Rect {
	return b.sem.GetRectOfSemanticsNodeInViewCoordinates(v, n)
}
func (b *semBinding) DebugOutstandingSemanticsHandles() int {
	return b.sem.DebugOutstandingSemanticsHandles()
}
func (b *semBinding) EnsureSemantics() *semantics.SemanticsHandle    { return b.sem.EnsureSemantics() }
func (b *semBinding) PerformSemanticsAction(ui.SemanticsActionEvent) {}
func (b *semBinding) AccessibilityFeatures() ui.AccessibilityFeatures {
	return b.sem.AccessibilityFeatures()
}
func (b *semBinding) HandleAccessibilityFeaturesChanged() { b.sem.HandleAccessibilityFeaturesChanged() }
func (b *semBinding) CreateSemanticsUpdateBuilder() ui.SemanticsUpdateBuilder {
	return &spyBuilder{builder: ui.NewSemanticsUpdateBuilder()}
}
func (b *semBinding) DisableAnimations() bool { return b.sem.DisableAnimations() }

func nodeAt(r ui.Rect) *semantics.SemanticsNode {
	n := semantics.NewSemanticsNode()
	n.SetRect(r)
	return n
}

func ltrb(l, t, r, b float64) ui.Rect { return ui.NewRectFromLTRB(l, t, r, b) }

func boundaryConfig(merging bool) *semantics.SemanticsConfiguration {
	c := semantics.NewSemanticsConfiguration()
	c.SetIsSemanticBoundary(true)
	c.SetIsMergingSemanticsOfDescendants(merging)
	return c
}

func upd(n *semantics.SemanticsNode, c *semantics.SemanticsConfiguration, children ...*semantics.SemanticsNode) *semantics.SemanticsNode {
	var ch []*semantics.SemanticsNode
	if children != nil {
		ch = children
	}
	n.UpdateWith(semantics.SemanticsNodeUpdateWithOptions{Config: c, ChildrenInInversePaintOrder: ch})
	return n
}

var hiddenLevel = foundation.DiagnosticLevelHidden

func dumpS(name string, node *semantics.SemanticsNode) {
	slogS(name + " traversal:\n" + node.ToStringDeep())
	slogS(name + " inverseHitTest:\n" + node.ToStringDeepInOrder(semantics.DebugSemanticsDumpOrderInverseHitTest))
}

func dumpHiddenS(name string, node *semantics.SemanticsNode) {
	slogS(name + " hidden:\n" + node.ToStringDeep(foundation.DiagnosticableTreeToStringDeepOptions{MinLevel: &hiddenLevel}))
}

func dataS(name string, node *semantics.SemanticsNode) {
	data := node.GetSemanticsData()
	slogS(name + " data: " + data.String())
	slogS(name + " data deep:\n" + data.ToDiagnosticsNode().ToStringDeep(foundation.DiagnosticsNodeToStringDeepOptions{MinLevel: &hiddenLevel}))
	slogS(name + " json: " + dart.JSONEncode(node.ToJson()))
}

func guardS(name string, f func()) {
	defer func() {
		r := recover()
		switch e := r.(type) {
		case nil:
			slogS(name + ": ok")
		case *foundation.FlutterError:
			slogS(name + ": FlutterError: " + e.Message())
		case dart.AssertionFailure:
			slogS(name + ": AssertionError: " + objS(e.AssertionMessage()))
		default:
			panic(r)
		}
	}()
	f()
}

func hexS(b []byte) string {
	if b == nil {
		return "null"
	}
	return hex.EncodeToString(b)
}

func newOwnerS(name string) *semantics.SemanticsOwner {
	return semantics.NewSemanticsOwner(func(update ui.SemanticsUpdate) {
		slogS(name + ": update built")
		update.Dispose()
	})
}

func ltr() *ui.TextDirection { d := ui.TextDirectionLtr; return &d }
func rtl() *ui.TextDirection { d := ui.TextDirectionRtl; return &d }
func sp(s string) *string    { return &s }
func bp(b bool) *bool        { return &b }
func ip(i int) *int          { return &i }

func cfg() *semantics.SemanticsConfiguration { return semantics.NewSemanticsConfiguration() }

func labelled(label string, dir *ui.TextDirection) *semantics.SemanticsConfiguration {
	c := cfg()
	c.SetLabel(label)
	c.SetTextDirection(dir)
	return c
}

func actionsMap(entries ...dart.MapEntry[semantics.CustomSemanticsAction, func()]) []dart.MapEntry[semantics.CustomSemanticsAction, func()] {
	return entries
}

func customEntry(label string, f func()) dart.MapEntry[semantics.CustomSemanticsAction, func()] {
	return dart.MapEntry[semantics.CustomSemanticsAction, func()]{Key: semantics.NewCustomSemanticsAction(label), Value: f}
}

// dartMapString is Dart's Map/List toString of a decoded message.
func dartMapString(v any) string {
	switch x := v.(type) {
	case []dart.MapEntry[any, any]:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = dartMapString(e.Key) + ": " + dartMapString(e.Value)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = dartMapString(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case float64:
		return dS(x)
	}
	return objS(v)
}

func sceneValuesS() {
	slogS("--- values")
	string1 := semantics.NewAttributedString("string1", semantics.AttributedStringOptions{
		Attributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 4))}})
	string2 := semantics.NewAttributedString("abc\ndef", semantics.AttributedStringOptions{
		Attributes: []ui.StringAttribute{ui.NewLocaleStringAttribute(ui.NewTextRange(1, 3), ui.NewLocale("es", "MX"))}})
	slogS(string1.Add(string2).String())
	slogS(semantics.NewAttributedString("").Add(string2).String())
	slogS(string1.Add(semantics.NewAttributedString("")).String())
	slogS("equal " + strconv.FormatBool(string1.Equal(semantics.NewAttributedString("string1",
		semantics.AttributedStringOptions{Attributes: string1.Attributes}))) + " " +
		strconv.FormatBool(string1.Equal(semantics.NewAttributedString("string1", semantics.AttributedStringOptions{
			Attributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 4))}}))) + " " +
		strconv.FormatBool(semantics.NewAttributedString("x").Equal(semantics.NewAttributedString("x"))))
	slogS(semantics.NewAttributedString("😀a", semantics.AttributedStringOptions{
		Attributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(2, 3))}}).Add(string1).String())
	slogS(semantics.NewAttributedStringProperty("label", &string2).String())
	single := foundation.SingleLineTextConfiguration
	slogS(semantics.NewAttributedStringProperty("label", &string2).ToString(foundation.DiagnosticsNodeToStringOptions{ParentConfiguration: single}))
	empty := semantics.NewAttributedString("")
	slogS(semantics.NewAttributedStringProperty("label", &empty).ToString(foundation.DiagnosticsNodeToStringOptions{MinLevel: &hiddenLevel}))
	slogS("interesting " + strconv.FormatBool(semantics.NewAttributedStringProperty("label", &empty).IsInteresting()) + " " +
		strconv.FormatBool(semantics.NewAttributedStringProperty("label", &empty, semantics.AttributedStringPropertyOptions{ShowWhenEmpty: true}).IsInteresting()) + " " +
		strconv.FormatBool(semantics.NewAttributedStringProperty("label", nil, semantics.AttributedStringPropertyOptions{DefaultValue: foundation.Null}).IsInteresting()))

	for _, locale := range []ui.Locale{
		ui.NewLocale("en"), ui.NewLocale("en", "US"), ui.NewLocale("iw", "DD"), ui.NewLocale("en", ""),
		ui.NewLocaleFromSubtags(ui.LocaleFromSubtagsOptions{ScriptCode: sp("Hant"), CountryCode: sp("TW")}),
		ui.NewLocaleFromSubtags(ui.LocaleFromSubtagsOptions{LanguageCode: "zh", ScriptCode: sp("Hans")}),
	} {
		var same ui.Locale
		if c := locale.CountryCode(); c != nil {
			same = ui.NewLocale(locale.LanguageCode(), *c)
		} else {
			same = ui.NewLocale(locale.LanguageCode())
		}
		slogS("locale " + locale.String() + " " + locale.ToLanguageTag() + " " + locale.LanguageCode() + " " +
			strP(locale.CountryCode()) + " " + strconv.FormatBool(locale.Equal(ui.NewLocale("en"))) + " " +
			strconv.FormatBool(locale.Equal(same)))
	}

	up := ui.TextAffinityUpstream
	for _, s := range []services.TextSelection{
		services.NewTextSelection(1, 4),
		services.NewTextSelection(4, 1, services.TextSelectionOptions{IsDirectional: true}),
		services.NewTextSelectionCollapsed(3, up),
		services.NewTextSelection(-1, -1),
	} {
		slogS("selection " + s.String() + " start=" + strconv.Itoa(s.Start) + " end=" + strconv.Itoa(s.End) +
			" base=" + s.Base().String() + " extent=" + s.Extent().String() + " valid=" + strconv.FormatBool(s.IsValid()))
	}

	semantics.CustomSemanticsActionResetForTests()
	a1 := semantics.NewCustomSemanticsAction("one")
	a2 := semantics.NewCustomSemanticsActionOverridingAction("two", ui.SemanticsActionTap)
	slogS(a1.String() + " " + a2.String())
	slogS("ids " + strconv.Itoa(semantics.CustomSemanticsActionGetIdentifier(a2)) + " " +
		strconv.Itoa(semantics.CustomSemanticsActionGetIdentifier(a1)) + " " +
		strconv.Itoa(semantics.CustomSemanticsActionGetIdentifier(semantics.NewCustomSemanticsAction("one"))))
	actionAt := func(id int) string {
		if a := semantics.CustomSemanticsActionGetAction(id); a != nil {
			return a.String()
		}
		return "null"
	}
	slogS(a1.String() + " " + a2.String() + " " + actionAt(0) + " " + actionAt(7))

	var flags, actions, roles []string
	for _, f := range ui.SemanticsFlagValues() {
		flags = append(flags, f.Name())
	}
	for _, a := range ui.SemanticsActionValues() {
		actions = append(actions, a.Name()+"="+strconv.Itoa(a.Index()))
	}
	for _, r := range ui.SemanticsRoleValues {
		roles = append(roles, r.String())
	}
	slogS("flags " + strings.Join(flags, ","))
	slogS("actions " + strings.Join(actions, ","))
	slogS("roles " + strings.Join(roles, ","))

	for _, builder := range []*semantics.SemanticsLabelBuilder{
		semantics.NewSemanticsLabelBuilder(),
		semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{TextDirection: ltr(), Separator: sp(", ")}),
		semantics.NewSemanticsLabelBuilder(semantics.SemanticsLabelBuilderOptions{TextDirection: rtl(), Separator: sp("")}),
	} {
		builder.AddPart("one")
		builder.AddPart("")
		builder.AddPart("two", semantics.SemanticsLabelBuilderAddPartOptions{TextDirection: rtl()})
		builder.AddPart("three", semantics.SemanticsLabelBuilderAddPartOptions{TextDirection: ltr()})
		builder.AddPart("four")
		slogS(`label "` + builder.Build() + `" ` + strconv.Itoa(builder.Length()))
	}

	codec := services.StandardMessageCodec{}
	nodeID, five := 70000, 5
	for _, event := range []semantics.SemanticsEvent{
		semantics.NewAnnounceSemanticsEvent("hello", ui.TextDirectionRtl, 2),
		semantics.NewAnnounceSemanticsEvent("né 😀", ui.TextDirectionLtr, 0, semantics.AnnounceSemanticsEventOptions{Assertiveness: semantics.AssertivenessAssertive}),
		semantics.NewTooltipSemanticsEvent("tip"),
		semantics.NewLongPressSemanticsEvent(),
		semantics.NewTapSemanticEvent(),
		semantics.NewFocusSemanticEvent(),
	} {
		slogS("event " + event.String() + " " + dart.JSONEncode(event.ToMap()) + " " +
			dart.JSONEncode(event.ToMap(semantics.SemanticsEventToMapOptions{NodeID: &nodeID})))
		slogS("bytes " + hexS(codec.EncodeMessage(event.ToMap(semantics.SemanticsEventToMapOptions{NodeID: &five}))))
	}
	slogS("codec " + hexS(codec.EncodeMessage([]any{nil, true, false, 1, -1, 2147483648, -2147483649, 1.5,
		strings.Repeat("a", 300), []byte{1, 2}, []int32{3}, []int64{4}, []float64{0.5}, []float32{0.25},
		[]dart.MapEntry[string, any]{{Key: "k", Value: []any{}}}})))
	slogS("decoded " + dartMapString(codec.DecodeMessage(codec.EncodeMessage([]dart.MapEntry[string, any]{
		{Key: "base", Value: 1}, {Key: "extent", Value: 3}, {Key: "x", Value: []any{1.5, "y"}}}))))
}

func sceneTreesAndOrderS() {
	slogS("--- trees and traversal order")
	semantics.DebugResetSemanticsIdCounter()
	grid := func(direction *ui.TextDirection) *semantics.SemanticsNode {
		children := []*semantics.SemanticsNode{
			nodeAt(ltrb(100, 0, 150, 40)),
			nodeAt(ltrb(0, 50, 40, 90)),
			nodeAt(ltrb(0, 0, 40, 40)),
			nodeAt(ltrb(50, 0, 90, 40)),
			nodeAt(ltrb(50, 50, 90, 90)),
			nodeAt(ltrb(20, 20, 70, 70)), // a knot
			nodeAt(ltrb(100, 45, 150, 95)),
		}
		root := nodeAt(ltrb(0, 0, 200, 100))
		config := cfg()
		if direction != nil {
			config.SetTextDirection(direction)
		}
		return upd(root, config, children...)
	}
	dumpS("grid null", grid(nil))
	dumpS("grid ltr", grid(ltr()))
	dumpS("grid rtl", grid(rtl()))

	keys := []semantics.SemanticsSortKey{
		semantics.NewOrdinalSortKey(3),
		semantics.NewOrdinalSortKey(1),
		nil,
		semantics.NewOrdinalSortKey(2, semantics.OrdinalSortKeyOptions{Name: sp("b")}),
		semantics.NewOrdinalSortKey(0, semantics.OrdinalSortKeyOptions{Name: sp("b")}),
		semantics.NewOrdinalSortKey(5, semantics.OrdinalSortKeyOptions{Name: sp("a")}),
		semantics.NewOrdinalSortKey(-1),
		semantics.NewOrdinalSortKey(-1),
	}
	var keyed []*semantics.SemanticsNode
	for i, key := range keys {
		config := labelled("k"+strconv.Itoa(i), ltr())
		if key != nil {
			config.SetSortKey(key)
		}
		keyed = append(keyed, upd(nodeAt(ui.NewRectFromLTWH(float64(i)*10.0, 0, 10, 10)), config))
	}
	rootConfig := cfg()
	rootConfig.SetTextDirection(ltr())
	dumpS("keys", upd(nodeAt(ltrb(0, 0, 100, 10)), rootConfig, keyed...))

	var many []*semantics.SemanticsNode
	for i := 0; i < 40; i++ {
		many = append(many, nodeAt(ui.NewRectFromLTWH(float64(i*7%5)*10.0, float64(i*3%4)*10.0, 10, 10)))
	}
	manyConfig := cfg()
	manyConfig.SetTextDirection(ltr())
	manyRoot := upd(nodeAt(ltrb(0, 0, 100, 100)), manyConfig, many...)
	slogS("many traversal " + idsS(manyRoot.DebugListChildrenInOrder(semantics.DebugSemanticsDumpOrderTraversalOrder)))
	var manyKeys []*semantics.SemanticsNode
	for i := 0; i < 40; i++ {
		c := cfg()
		c.SetSortKey(semantics.NewOrdinalSortKey(float64(i * 5 % 7)))
		manyKeys = append(manyKeys, upd(nodeAt(ui.NewRectFromLTWH(float64(i)*1.0, 0, 10, 10)), c))
	}
	manyKeysRoot := upd(nodeAt(ltrb(0, 0, 100, 100)), cfg(), manyKeys...)
	slogS("many keys " + idsS(manyKeysRoot.DebugListChildrenInOrder(semantics.DebugSemanticsDumpOrderTraversalOrder)))

	rotated := nodeAt(ltrb(0, 0, 30, 10))
	rotated.SetTransform(vm.NewMatrix4RotationZ(0.3))
	rotated2 := nodeAt(ltrb(0, 0, 30, 10))
	t2 := vm.NewMatrix4TranslationValues(40, 5, 0)
	t2.Multiply(vm.NewMatrix4RotationZ(-2.5))
	rotated2.SetTransform(t2)
	scaledChild := nodeAt(ltrb(0, 0, 10, 10))
	scaledChild.SetTransform(vm.NewMatrix4Diagonal3Values(3, 3, 1))
	translated := nodeAt(ltrb(0, 0, 10, 10))
	translated.SetTransform(vm.NewMatrix4TranslationValues(60, 1, 0))
	identityChild := nodeAt(ltrb(5, 5, 15, 15))
	identityChild.SetTransform(vm.NewMatrix4Identity())
	trConfig := cfg()
	trConfig.SetTextDirection(rtl())
	transformedRoot := upd(nodeAt(ltrb(0, 0, 100, 100)), trConfig, rotated, rotated2, scaledChild, translated, identityChild)
	dumpS("transformed", transformedRoot)
	dumpHiddenS("transformed", transformedRoot)
	tr := "null"
	if identityChild.Transform() != nil {
		tr = identityChild.Transform().String()
	}
	slogS("identity child transform " + tr)
	dataS("rotated", rotated)
}

func idsS(nodes []*semantics.SemanticsNode) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = strconv.Itoa(n.ID())
	}
	return strings.Join(parts, ",")
}

func sceneMergingS() {
	slogS("--- merging")
	semantics.DebugResetSemanticsIdCounter()
	semantics.CustomSemanticsActionResetForTests()
	tagA := semantics.NewSemanticsTag("tagA")
	tagB := semantics.NewSemanticsTag("tagB")

	leaf1 := nodeAt(ltrb(0, 0, 10, 10))
	leaf1.Tags = []*semantics.SemanticsTag{tagB}
	c1 := labelled("first", rtl())
	c1.SetHint("h1")
	c1.SetValue("v1")
	c1.SetIncreasedValue("v2")
	c1.SetOnIncrease(func() {})
	c1.SetIsChecked(bp(true))
	c1.SetIsToggled(bp(false))
	c1.SetControlsNodes([]string{"x", "y"})
	c1.SetValidationResult(ui.SemanticsValidationResultValid)
	c1.SetTooltip("tip")
	c1.SetHeadingLevel(3)
	c1.SetCustomSemanticsActions(actionsMap(customEntry("custom1", func() {})))
	c1.SetHintOverrides(semantics.NewSemanticsHintOverrides(semantics.SemanticsHintOverridesOptions{OnTapHint: sp("tap me")}))
	c1.SetMinValue(sp("1"))
	upd(leaf1, c1)

	leaf2 := nodeAt(ltrb(10, 0, 20, 10))
	leaf2.Tags = []*semantics.SemanticsTag{tagA, tagB}
	c2 := cfg()
	c2.SetAttributedLabel(semantics.NewAttributedString("second", semantics.AttributedStringOptions{
		Attributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 3))}}))
	c2.SetTextDirection(ltr())
	c2.SetHint("h2")
	c2.IsBlockingUserActions = true
	c2.SetOnTap(func() {})
	c2.SetOnDidGainAccessibilityFocus(func() {})
	c2.SetIsSelected(true)
	c2.SetIsEnabled(bp(true))
	c2.SetControlsNodes([]string{"y", "z"})
	c2.SetValidationResult(ui.SemanticsValidationResultInvalid)
	c2.SetRole(ui.SemanticsRoleTab)
	c2.SetInputType(ui.SemanticsInputTypeSearch)
	c2.SetMaxValue(sp("9"))
	c2.SetIdentifier("id2")
	c2.SetScrollPosition(fp(3.5))
	c2.SetScrollExtentMax(fp(10))
	c2.SetScrollExtentMin(fp(0))
	c2.SetScrollChildCount(ip(4))
	c2.SetScrollIndex(ip(1))
	c2.SetCustomSemanticsActions(actionsMap(customEntry("custom2", func() {}), customEntry("custom1", func() {})))
	upd(leaf2, c2)

	middle := upd(nodeAt(ltrb(0, 0, 20, 10)), cfg(), leaf1, leaf2)
	mergeRoot := nodeAt(ltrb(0, 0, 30, 30))
	mergeRoot.Tags = []*semantics.SemanticsTag{tagA}
	rc := boundaryConfig(true)
	rc.SetLabel("root")
	rc.SetTextDirection(ltr())
	fr := ui.NewLocale("fr", "CA")
	rc.Locale = &fr
	rc.SetIsButton(true)
	upd(mergeRoot, rc, middle)
	dumpS("merge root", mergeRoot)
	dumpHiddenS("merge root", mergeRoot)
	dataS("merge root", mergeRoot)
	dataS("leaf2", leaf2)
	slogS("merged flags " + strconv.FormatBool(leaf1.IsMergedIntoParent()) + " " + strconv.FormatBool(leaf2.IsMergedIntoParent()) + " " +
		strconv.FormatBool(middle.IsMergedIntoParent()) + " " + strconv.FormatBool(mergeRoot.IsPartOfNodeMerging()))
	slogS("equal data " + strconv.FormatBool(mergeRoot.GetSemanticsData().Equal(mergeRoot.GetSemanticsData())) + " " +
		strconv.FormatBool(leaf1.GetSemanticsData().Equal(leaf2.GetSemanticsData())))
	h1 := semantics.NewSemanticsHintOverrides(semantics.SemanticsHintOverridesOptions{OnTapHint: sp("a")})
	h2 := semantics.NewSemanticsHintOverrides(semantics.SemanticsHintOverridesOptions{OnTapHint: sp("a")})
	slogS("hint overrides " + strconv.FormatBool(h1.Equal(h2)) + " " +
		semantics.NewSemanticsHintOverrides(semantics.SemanticsHintOverridesOptions{OnTapHint: sp("a"), OnLongPressHint: sp("b")}).ToStringDeep())
}

func describeConfig(name string, c *semantics.SemanticsConfiguration) {
	controls := "null"
	if c.ControlsNodes() != nil {
		controls = "{" + strings.Join(c.ControlsNodes(), ", ") + "}"
	}
	sortKey := "null"
	if c.SortKey() != nil {
		sortKey = c.SortKey().String()
	}
	var actions []string
	for _, a := range ui.SemanticsActionValues() {
		if c.GetActionHandler(a) != nil {
			actions = append(actions, a.Name())
		}
	}
	b := strconv.FormatBool
	slogS(name + ": annotated=" + b(c.HasBeenAnnotated()) + ` label="` + c.Label() + `" ` + c.AttributedLabel().String() +
		` hint="` + c.Hint() + `" value="` + c.Value() + `" flags=` + b(c.IsButton()) + "," + boolP(c.IsChecked()) + "," +
		boolP(c.IsCheckStateMixed()) + "," + b(c.IsSelected()) + "," + boolP(c.IsEnabled()) + "," + boolP(c.IsToggled()) + "," +
		boolP(c.IsExpanded()) + "," + boolP(c.IsFocused()) + "," + boolP(c.IsRequired()) + "," + b(c.IsLink()) + "," +
		b(c.IsHeader()) + "," + b(c.IsTextField()) + " textDirection=" + dirS(c.TextDirection()) +
		" heading=" + strconv.Itoa(c.HeadingLevel()) + " role=" + c.Role().String() + " validation=" + c.ValidationResult().String() +
		" hitTest=" + c.HitTestBehavior().String() + " controls=" + controls + " focusBlock=" + c.AccessibilityFocusBlockType().String() +
		" traversal=" + objS(c.TraversalParentIdentifier()) + "," + objS(c.TraversalChildIdentifier()) +
		" min=" + strP(c.MinValue()) + " max=" + strP(c.MaxValue()) + ` tooltip="` + c.Tooltip() + `" identifier="` + c.Identifier() +
		`" sortKey=` + sortKey + " actions=" + listS(actions))
}

func sceneConfigurationS() {
	slogS("--- configuration")
	parent := labelled("parent", ltr())
	parent.SetIsButton(true)
	parent.SetOnTap(func() {})
	parent.SetControlsNodes([]string{"a"})
	parent.SetValidationResult(ui.SemanticsValidationResultValid)
	parent.SetTraversalParentIdentifier("p")
	child := labelled("child", rtl())
	child.SetHint("child hint")
	child.SetIsChecked(bp(false))
	child.SetIsCheckStateMixed(bp(true))
	child.SetOnLongPress(func() {})
	child.SetOnScrollUp(func() {})
	child.IsBlockingUserActions = true
	child.SetOnDidLoseAccessibilityFocus(func() {})
	child.SetControlsNodes([]string{"b", "a"})
	child.SetValidationResult(ui.SemanticsValidationResultInvalid)
	child.SetAccessibilityFocusBlockType(semantics.AccessibilityFocusBlockTypeBlockNode)
	child.SetHeadingLevel(2)
	child.SetSortKey(semantics.NewOrdinalSortKey(4))
	child.SetHitTestBehavior(ui.SemanticsHitTestBehaviorOpaque)
	child.SetMinValue(sp("0"))
	child.SetTooltip("tt")
	child.SetIdentifier("cid")
	describeConfig("parent", parent)
	describeConfig("child", child)
	b := strconv.FormatBool
	slogS("compatible " + b(parent.IsCompatibleWith(child)) + " " + b(child.IsCompatibleWith(parent)) + " " +
		b(parent.IsCompatibleWith(cfg())) + " " + b(parent.IsCompatibleWith(nil)))
	with := func(f func(c *semantics.SemanticsConfiguration)) *semantics.SemanticsConfiguration {
		c := cfg()
		f(c)
		return c
	}
	en := ui.NewLocale("en")
	pairs := [][2]*semantics.SemanticsConfiguration{
		{with(func(c *semantics.SemanticsConfiguration) { c.SetOnTap(func() {}) }), with(func(c *semantics.SemanticsConfiguration) { c.SetOnTap(func() {}) })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetIsButton(true) }), with(func(c *semantics.SemanticsConfiguration) { c.SetIsButton(true) })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetIsButton(true) }), with(func(c *semantics.SemanticsConfiguration) { c.SetIsHeader(true) })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetValue("a") }), with(func(c *semantics.SemanticsConfiguration) { c.SetValue("b") })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetIsTextField(true) }), with(func(c *semantics.SemanticsConfiguration) { c.SetIsImage(true) })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetIsTextField(true) }), with(func(c *semantics.SemanticsConfiguration) { c.SetLabel("x") })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetRole(ui.SemanticsRoleTab) }), with(func(c *semantics.SemanticsConfiguration) { c.SetIsLink(true) })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetLabel("a") }), with(func(c *semantics.SemanticsConfiguration) {
			c.SetHitTestBehavior(ui.SemanticsHitTestBehaviorTransparent)
		})},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetPlatformViewID(ip(1)) }), with(func(c *semantics.SemanticsConfiguration) { c.SetPlatformViewID(ip(2)) })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetLocaleForSubtree(&en) }), with(func(c *semantics.SemanticsConfiguration) { c.SetLabel("x") })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetTraversalChildIdentifier(1) }), with(func(c *semantics.SemanticsConfiguration) { c.SetLabel("x") })},
		{cfg(), with(func(c *semantics.SemanticsConfiguration) { c.SetTraversalChildIdentifier(1) })},
		{with(func(c *semantics.SemanticsConfiguration) {
			c.SetAccessibilityFocusBlockType(semantics.AccessibilityFocusBlockTypeBlockSubtree)
		}), with(func(c *semantics.SemanticsConfiguration) { c.SetLabel("y") })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetMinValue(sp("1")) }), with(func(c *semantics.SemanticsConfiguration) { c.SetMinValue(sp("2")) })},
		{with(func(c *semantics.SemanticsConfiguration) { c.SetIsFocused(bp(true)) }), with(func(c *semantics.SemanticsConfiguration) { c.SetIsFocused(bp(false)) })},
	}
	for i, p := range pairs {
		slogS("pair " + strconv.Itoa(i) + " " + b(p[0].IsCompatibleWith(p[1])))
	}
	parent.Absorb(child)
	describeConfig("absorbed", parent)
	describeConfig("copy", parent.Copy())
	unannotated := cfg()
	unannotated.SetLabel("keep")
	unannotated.Absorb(cfg())
	describeConfig("absorb unannotated", unannotated)
	focus := cfg()
	focus.SetIsFocusable(true)
	describeConfig("focusable", focus)
	focus.SetIsFocused(bp(true))
	focus.SetIsFocusable(false)
	describeConfig("unfocusable", focus)
	values := cfg()
	values.SetIsChecked(nil)
	values.SetIsCheckStateMixed(bp(false))
	values.SetIsExpanded(bp(false))
	values.SetIsRequired(bp(true))
	link, _ := url.Parse("https://flutter.dev/a?b=1")
	values.SetLinkURL(link)
	values.SetIsLink(true)
	values.SetIndexInParent(ip(3))
	values.SetScrollIndex(ip(2))
	values.SetScrollChildCount(ip(2))
	describeConfig("values", values)
	cursor := cfg()
	cursor.SetOnMoveCursorForwardByWord(func(b bool) { slogS("word forward " + strconv.FormatBool(b)) })
	word := "null"
	if cursor.OnMoveCursorForwardByWord() != nil {
		word = "closure"
	}
	slogS("cursor " + word + " " + b(cursor.OnMoveCursorForwardByCharacter() != nil))
}

func sceneOwnerS() {
	slogS("--- owner")
	semantics.DebugResetSemanticsIdCounter()
	semantics.CustomSemanticsActionResetForTests()
	owner := newOwnerS("owner")
	root := semantics.NewSemanticsNodeRoot(owner, semantics.SemanticsNodeOptions{ShowOnScreen: func() { slogS("root showOnScreen") }})
	root.SetRect(ltrb(0, 0, 400, 300))
	a := nodeAt(ltrb(0, 0, 100, 100))
	b := nodeAt(ltrb(100, 0, 200, 100))
	b.SetTransform(vm.NewMatrix4TranslationValues(50, 20, 0))
	b1 := nodeAt(ltrb(0, 0, 20, 20))
	b2 := nodeAt(ltrb(20, 0, 40, 20))
	c := nodeAt(ltrb(0, 100, 300, 200))
	bc := boundaryConfig(true)
	bc.SetLabel("b")
	bc.SetTextDirection(ltr())
	bc.SetOnTap(func() { slogS("b tapped") })
	upd(b, bc, b1, b2)
	b1c := labelled("b1", ltr())
	b1c.SetOnLongPress(func() { slogS("b1 long pressed") })
	b1c.SetCustomSemanticsActions(actionsMap(customEntry("b1 custom", func() { slogS("b1 custom") })))
	upd(b1, b1c)
	b2c := labelled("b2", ltr())
	b2c.SetOnScrollToOffset(func(o ui.Offset) { slogS("b2 scroll to " + o.String()) })
	b2c.SetOnSetSelection(func(s services.TextSelection) { slogS("b2 selection " + s.String()) })
	b2c.SetOnSetText(func(s string) { slogS("b2 text " + s) })
	b2c.SetCustomSemanticsActions(actionsMap(customEntry("b2 custom", func() { slogS("b2 custom") })))
	upd(b2, b2c)
	ac := labelled("a", ltr())
	ac.SetOnTap(func() { slogS("a tapped") })
	sel := services.NewTextSelection(2, 1)
	ac.SetTextSelection(&sel)
	ac.SetPlatformViewID(ip(7))
	ac.SetMaxValueLength(ip(10))
	ac.SetCurrentValueLength(ip(3))
	link, _ := url.Parse("https://example.com/x")
	ac.SetLinkURL(link)
	ac.SetIsLink(true)
	ac.SetHeadingLevel(1)
	gb := ui.NewLocale("en", "GB")
	ac.Locale = &gb
	ac.SetIndexInParent(ip(0))
	upd(a, ac)
	cc := labelled("c", rtl())
	cc.SetValue("50")
	cc.SetIncreasedValue("60")
	cc.SetDecreasedValue("40")
	cc.SetOnIncrease(func() { slogS("c increase") })
	cc.SetOnDecrease(func() { slogS("c decrease") })
	cc.SetHintOverrides(semantics.NewSemanticsHintOverrides(semantics.SemanticsHintOverridesOptions{OnLongPressHint: sp("hold")}))
	upd(c, cc)
	rootLtr := cfg()
	rootLtr.SetTextDirection(ltr())
	upd(root, rootLtr, a, b, c)
	owner.SendSemanticsUpdate()
	slogS("dirty after " + boolP(root.DebugIsDirty()) + " " + boolP(b1.DebugIsDirty()))
	owner.SendSemanticsUpdate() // nothing dirty: no update

	// Change a leaf of the merged node: the merge root is sent again.
	upd(b2, labelled("b2 changed", rtl()))
	slogS("b2 dirty " + boolP(b2.DebugIsDirty()))
	owner.SendSemanticsUpdate()

	// Reorder and drop children; mark several dirty.
	c.SetRect(ltrb(0, 100, 290, 200))
	a.SetRect(ltrb(0, 0, 90, 100))
	rootRtl := cfg()
	rootRtl.SetTextDirection(rtl())
	upd(root, rootRtl, c, a)
	slogS("b attached " + strconv.FormatBool(b.Attached()) + " " + strconv.FormatBool(b1.Attached()))
	dumpS("owner root", root)
	owner.SendSemanticsUpdate()

	// Re-add b: it was detached, so it is sent again.
	rootRtl2 := cfg()
	rootRtl2.SetTextDirection(rtl())
	upd(root, rootRtl2, c, a, b)
	owner.SendSemanticsUpdate()

	// Actions.
	owner.PerformAction(a.ID(), ui.SemanticsActionTap)
	owner.PerformAction(b.ID(), ui.SemanticsActionTap)
	owner.PerformAction(b.ID(), ui.SemanticsActionLongPress)
	owner.PerformAction(b.ID(), ui.SemanticsActionCustomAction,
		semantics.CustomSemanticsActionGetIdentifier(semantics.NewCustomSemanticsAction("b1 custom")))
	owner.PerformAction(b2.ID(), ui.SemanticsActionSetText, "typed")
	owner.PerformAction(b.ID(), ui.SemanticsActionScrollToOffset, []float64{1.5, 2.5})
	owner.PerformAction(b.ID(), ui.SemanticsActionSetSelection, []dart.MapEntry[any, any]{{Key: "base", Value: 4}, {Key: "extent", Value: 2}})
	owner.PerformAction(c.ID(), ui.SemanticsActionIncrease)
	owner.PerformAction(c.ID(), ui.SemanticsActionDecrease)
	owner.PerformAction(c.ID(), ui.SemanticsActionTap)
	owner.PerformAction(root.ID(), ui.SemanticsActionShowOnScreen)
	owner.PerformAction(999, ui.SemanticsActionTap)
	owner.PerformActionAt(ui.NewOffset(10, 10), ui.SemanticsActionTap)
	owner.PerformActionAt(ui.NewOffset(160, 30), ui.SemanticsActionTap) // b is translated by (50, 20)
	owner.PerformActionAt(ui.NewOffset(160, 30), ui.SemanticsActionLongPress)
	owner.PerformActionAt(ui.NewOffset(149, 30), ui.SemanticsActionTap)
	owner.PerformActionAt(ui.NewOffset(150, 150), ui.SemanticsActionIncrease)
	owner.PerformActionAt(ui.NewOffset(500, 500), ui.SemanticsActionTap)
	dumpHiddenS("owner b", b)
	dataS("owner a", a)
	dataS("owner c", c)
	slogS("root data " + root.GetSemanticsData().String())
	slogS("owner node " + strconv.FormatBool(owner.GetSemanticsNode(b.ID()) == b) + " " +
		strconv.FormatBool(owner.RootSemanticsNode() == root))
	owner.Dispose()
}

func sceneTraversalGraftingS() {
	slogS("--- traversal grafting")
	semantics.DebugResetSemanticsIdCounter()
	owner := newOwnerS("graft")
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(ltrb(0, 0, 400, 400))
	anchor := nodeAt(ltrb(10, 10, 60, 60))
	anchor.SetTransform(vm.NewMatrix4TranslationValues(5, 5, 0))
	anchorChild := nodeAt(ltrb(0, 0, 20, 20))
	overlay := nodeAt(ltrb(100, 100, 300, 300))
	ot := vm.NewMatrix4TranslationValues(10, 0, 0)
	ot.Multiply(vm.NewMatrix4Diagonal3Values(2, 2, 1))
	overlay.SetTransform(ot)
	menu := nodeAt(ltrb(0, 0, 50, 50))
	orphan := nodeAt(ltrb(60, 0, 90, 50))
	anchorConfig := labelled("anchor", ltr())
	anchorConfig.SetTraversalParentIdentifier("portal")
	upd(anchor, anchorConfig, anchorChild)
	upd(anchorChild, labelled("anchor child", ltr()))
	menuConfig := labelled("menu item", ltr())
	menuConfig.SetTraversalChildIdentifier("portal")
	upd(menu, menuConfig)
	orphanConfig := labelled("orphan", ltr())
	orphanConfig.SetTraversalChildIdentifier("nobody")
	upd(orphan, orphanConfig)
	upd(overlay, cfg(), menu, orphan)
	rootConfig := cfg()
	rootConfig.SetTextDirection(ltr())
	upd(root, rootConfig, anchor, overlay)
	owner.SendSemanticsUpdate()
	tp := "null"
	if menu.TraversalParent() != nil {
		tp = strconv.Itoa(menu.TraversalParent().ID())
	}
	slogS("menu traversal parent " + tp)
	dumpS("graft root", root)
	lone := nodeAt(ltrb(300, 0, 350, 50))
	loneChild := nodeAt(ltrb(0, 0, 10, 10))
	loneChild.SetTransform(vm.NewMatrix4RotationZ(dartPi / 6))
	loneConfig := cfg()
	loneConfig.SetTraversalParentIdentifier(7)
	upd(lone, loneConfig)
	loneChildConfig := cfg()
	loneChildConfig.SetTraversalChildIdentifier(7)
	upd(loneChild, loneChildConfig)
	holder := upd(nodeAt(ltrb(300, 300, 400, 400)), cfg(), loneChild)
	rootConfig2 := cfg()
	rootConfig2.SetTextDirection(ltr())
	upd(root, rootConfig2, anchor, overlay, lone, holder)
	owner.SendSemanticsUpdate()
	rootConfig3 := cfg()
	rootConfig3.SetTextDirection(ltr())
	upd(root, rootConfig3, anchor, lone, holder)
	owner.SendSemanticsUpdate()
	dumpS("graft root after", root)
	owner.Dispose()
}

var dartPi = 3.141592653589793

type roleCase struct {
	name    string
	configs []*semantics.SemanticsConfiguration
}

func sceneRoleChecksS() {
	slogS("--- role checks")
	role := func(r ui.SemanticsRole, f ...func(c *semantics.SemanticsConfiguration)) *semantics.SemanticsConfiguration {
		c := cfg()
		c.SetRole(r)
		c.SetTextDirection(ltr())
		for _, fn := range f {
			fn(c)
		}
		return c
	}
	plain := func(f func(c *semantics.SemanticsConfiguration)) *semantics.SemanticsConfiguration {
		c := cfg()
		f(c)
		return c
	}
	type C = semantics.SemanticsConfiguration
	vals := func(v, minV, maxV string) func(c *C) {
		return func(c *C) {
			if v != "" {
				c.SetValue(v)
			}
			if minV != "" {
				c.SetMinValue(sp(minV))
			}
			if maxV != "" {
				c.SetMaxValue(sp(maxV))
			}
		}
	}
	radio := func(checked bool) *C {
		return plain(func(c *C) { c.SetIsInMutuallyExclusiveGroup(true); c.SetIsChecked(bp(checked)) })
	}
	cases := []roleCase{
		{"tab no selected", []*C{role(ui.SemanticsRoleTab)}},
		{"tab no tap", []*C{role(ui.SemanticsRoleTab, func(c *C) { c.SetIsSelected(false) })}},
		{"tab ok", []*C{role(ui.SemanticsRoleTab, func(c *C) { c.SetIsSelected(true); c.SetOnTap(func() {}) })}},
		{"tab disabled", []*C{role(ui.SemanticsRoleTab, func(c *C) { c.SetIsSelected(true); c.SetIsEnabled(bp(false)) })}},
		{"empty tab bar", []*C{role(ui.SemanticsRoleTabBar)}},
		{"tab bar child", []*C{role(ui.SemanticsRoleTabBar), role(ui.SemanticsRoleCell)}},
		{"table child", []*C{role(ui.SemanticsRoleTable), role(ui.SemanticsRoleCell)}},
		{"row outside table", []*C{role(ui.SemanticsRoleRow)}},
		{"cell outside row", []*C{role(ui.SemanticsRoleCell)}},
		{"column header", []*C{role(ui.SemanticsRoleColumnHeader)}},
		{"radio group", []*C{role(ui.SemanticsRoleRadioGroup), radio(true), radio(true)}},
		{"radio group ok", []*C{role(ui.SemanticsRoleRadioGroup), radio(true), radio(false)}},
		{"empty menu", []*C{role(ui.SemanticsRoleMenu)}},
		{"empty menu bar", []*C{role(ui.SemanticsRoleMenuBar)}},
		{"menu item", []*C{role(ui.SemanticsRoleMenuItem)}},
		{"menu item checkbox", []*C{role(ui.SemanticsRoleMenuItemCheckbox)}},
		{"menu item checkbox in menu", []*C{role(ui.SemanticsRoleMenu), role(ui.SemanticsRoleMenuItemCheckbox, func(c *C) { c.SetIsChecked(bp(true)) })}},
		{"menu item radio", []*C{role(ui.SemanticsRoleMenuItemRadio, func(c *C) { c.SetIsChecked(bp(false)) })}},
		{"alert live region", []*C{role(ui.SemanticsRoleAlert, func(c *C) { c.SetLiveRegion(true) })}},
		{"list item", []*C{role(ui.SemanticsRoleListItem)}},
		{"list item in list", []*C{role(ui.SemanticsRoleList), role(ui.SemanticsRoleListItem)}},
		{"region", []*C{role(ui.SemanticsRoleRegion)}},
		{"region labelled", []*C{role(ui.SemanticsRoleRegion, func(c *C) { c.SetLabel("r") })}},
		{"main in navigation", []*C{role(ui.SemanticsRoleNavigation, func(c *C) { c.SetLabel("n") }), role(ui.SemanticsRoleMain)}},
		{"complementary", []*C{role(ui.SemanticsRoleComplementary)}},
		{"progress bar missing", []*C{role(ui.SemanticsRoleProgressBar, vals("5", "", ""))}},
		{"progress bar bad number", []*C{role(ui.SemanticsRoleProgressBar, vals("x", "0", "10"))}},
		{"progress bar min max", []*C{role(ui.SemanticsRoleProgressBar, vals("5", "10", "1e1"))}},
		{"progress bar out of range", []*C{role(ui.SemanticsRoleProgressBar, vals(" 11 ", "0", "10"))}},
		{"progress bar percent", []*C{role(ui.SemanticsRoleProgressBar, vals("150%", "0", "10"))}},
		{"progress bar ok", []*C{role(ui.SemanticsRoleProgressBar, vals("50%", "-1.5", "Infinity"))}},
		{"slider missing", []*C{role(ui.SemanticsRoleSlider, vals("1", "0", ""))}},
		{"slider bad", []*C{role(ui.SemanticsRoleSlider, vals("1", "0x1", "2"))}},
		{"slider percent", []*C{role(ui.SemanticsRoleSlider, vals("1%0%", "0", "200"))}},
		{"slider range", []*C{role(ui.SemanticsRoleSlider, vals("-.5", "0", "2"))}},
		{"slider ok", []*C{role(ui.SemanticsRoleSlider, vals("1.", "0", "2"))}},
		{"drag handle", []*C{role(ui.SemanticsRoleDragHandle)}},
		{"expanded with expand", []*C{plain(func(c *C) { c.SetIsExpanded(bp(true)); c.SetOnExpand(func() {}) })}},
		{"collapsed with collapse", []*C{plain(func(c *C) { c.SetIsExpanded(bp(false)); c.SetOnCollapse(func() {}) })}},
		{"expand and collapse", []*C{plain(func(c *C) {
			c.SetIsExpanded(bp(true))
			c.SetOnExpand(func() {})
			c.SetOnCollapse(func() {})
		})}},
		{"focus blocked", []*C{plain(func(c *C) {
			c.SetIsFocused(bp(false))
			c.SetAccessibilityFocusBlockType(semantics.AccessibilityFocusBlockTypeBlockNode)
		})}},
	}
	for _, entry := range cases {
		semantics.DebugResetSemanticsIdCounter()
		owner := newOwnerS(entry.name)
		root := semantics.NewSemanticsNodeRoot(owner)
		root.SetRect(ltrb(0, 0, 100, 100))
		parent := nodeAt(ltrb(0, 0, 50, 50))
		var children []*semantics.SemanticsNode
		for i := 1; i < len(entry.configs); i++ {
			children = append(children, upd(nodeAt(ui.NewRectFromLTWH(float64(i)*5.0, 0, 5, 5)), entry.configs[i]))
		}
		if children == nil {
			children = []*semantics.SemanticsNode{}
		}
		upd(parent, entry.configs[0], children...)
		upd(root, cfg(), parent)
		guardS(entry.name, owner.SendSemanticsUpdate)
		owner.Dispose()
	}
}

func sceneErrorsS() {
	slogS("--- errors")
	semantics.DebugResetSemanticsIdCounter()
	guardS("non-finite rect", func() { semantics.NewSemanticsNode().SetRect(ltrb(0, 0, infS(), 1)) })
	guardS("increase without increasedValue", func() {
		c := labelled("", ltr())
		c.SetValue("v")
		c.SetOnIncrease(func() {})
		semantics.NewSemanticsNode().UpdateWith(semantics.SemanticsNodeUpdateWithOptions{Config: c})
	})
	guardS("data with label and no direction", func() {
		c := cfg()
		c.SetLabel("x")
		upd(semantics.NewSemanticsNode(), c).GetSemanticsData()
	})
	guardS("merging without boundary", func() { cfg().SetIsMergingSemanticsOfDescendants(true) })
	guardS("attributed range", func() {
		semantics.NewAttributedString("ab", semantics.AttributedStringOptions{
			Attributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 3))}})
	})
	guardS("duplicate builder", func() {
		config := cfg()
		builder := semantics.NewChildSemanticsConfigurationsResultBuilder()
		builder.MarkAsMergeUp(config)
		builder.MarkAsSiblingMergeGroup([]*semantics.SemanticsConfiguration{config})
		builder.Build()
	})
	builder := semantics.NewChildSemanticsConfigurationsResultBuilder()
	builder.MarkAsMergeUp(cfg())
	builder.MarkAsSiblingMergeGroup([]*semantics.SemanticsConfiguration{cfg(), cfg()})
	result := builder.Build()
	slogS("result " + strconv.Itoa(len(result.MergeUp)) + " " + strconv.Itoa(len(result.SiblingMergeGroups)) + " " +
		strconv.Itoa(len(result.SiblingMergeGroups[0])))
	owner := newOwnerS("graft errors")
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(ltrb(0, 0, 100, 100))
	child := nodeAt(ltrb(0, 0, 50, 50))
	grandChild := nodeAt(ltrb(0, 0, 10, 10))
	childConfig := cfg()
	childConfig.SetTraversalChildIdentifier("loop")
	upd(child, childConfig, grandChild)
	grandConfig := cfg()
	grandConfig.SetTraversalParentIdentifier("loop")
	upd(grandChild, grandConfig)
	upd(root, cfg(), child)
	guardS("graft cycle", owner.SendSemanticsUpdate)
}

func sceneActionArgsS() {
	slogS("--- action args")
	semantics.DebugResetSemanticsIdCounter()
	semantics.CustomSemanticsActionResetForTests()
	owner := newOwnerS("args")
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(ltrb(0, 0, 100, 100))
	field := nodeAt(ltrb(0, 0, 50, 50))
	config := labelled("field", ltr())
	config.SetOnScrollToOffset(func(o ui.Offset) { slogS("scroll to " + o.String()) })
	config.SetOnSetSelection(func(s services.TextSelection) { slogS("selection " + s.String()) })
	config.SetOnSetText(func(s string) { slogS("text " + s) })
	config.SetOnMoveCursorForwardByCharacter(func(b bool) { slogS("forward char " + strconv.FormatBool(b)) })
	config.SetOnMoveCursorBackwardByWord(func(b bool) { slogS("backward word " + strconv.FormatBool(b)) })
	config.SetOnCopy(func() { slogS("copy") })
	config.SetOnDismiss(func() { slogS("dismiss") })
	config.SetCustomSemanticsActions(actionsMap(customEntry("x", func() { slogS("custom x") })))
	upd(field, config)
	parentConfig := boundaryConfig(true)
	parentConfig.SetCustomSemanticsActions(actionsMap(customEntry("y", func() { slogS("custom y") })))
	mergeParent := upd(nodeAt(ltrb(0, 0, 100, 100)), parentConfig, field)
	upd(root, cfg(), mergeParent)
	owner.SendSemanticsUpdate()
	x := semantics.CustomSemanticsActionGetIdentifier(semantics.NewCustomSemanticsAction("x"))
	y := semantics.CustomSemanticsActionGetIdentifier(semantics.NewCustomSemanticsAction("y"))
	slogS("ids " + strconv.Itoa(x) + " " + strconv.Itoa(y))
	id := mergeParent.ID()
	owner.PerformAction(id, ui.SemanticsActionScrollToOffset, []float64{3.0, 4.5})
	owner.PerformAction(id, ui.SemanticsActionSetSelection, []dart.MapEntry[any, any]{{Key: "base", Value: 5}, {Key: "extent", Value: 1}})
	owner.PerformAction(id, ui.SemanticsActionSetText, "hello")
	owner.PerformAction(id, ui.SemanticsActionMoveCursorForwardByCharacter, true)
	owner.PerformAction(id, ui.SemanticsActionMoveCursorBackwardByWord, false)
	owner.PerformAction(id, ui.SemanticsActionCopy)
	owner.PerformAction(id, ui.SemanticsActionDismiss)
	owner.PerformAction(id, ui.SemanticsActionCustomAction, x)
	owner.PerformAction(id, ui.SemanticsActionCustomAction, y)
	owner.PerformAction(id, ui.SemanticsActionCustomAction, 99)
	owner.PerformActionAt(ui.NewOffset(10, 10), ui.SemanticsActionCustomAction, x)
	owner.PerformActionAt(ui.NewOffset(60, 60), ui.SemanticsActionCustomAction, y)
	owner.PerformActionAt(ui.NewOffset(60, 60), ui.SemanticsActionSetText, "at")
	word := "null"
	if config.OnMoveCursorBackwardByWord() != nil {
		word = "closure"
	}
	slogS("handlers " + word + " " + strconv.FormatBool(config.OnMoveCursorBackwardByCharacter() != nil) + " " +
		strconv.FormatBool(config.GetActionHandler(ui.SemanticsActionMoveCursorBackwardByWord) != nil))
	dataS("merge parent", mergeParent)
	owner.Dispose()
}

func sceneExtraS() {
	slogS("--- extra")
	semantics.DebugResetSemanticsIdCounter()
	knotA := nodeAt(ltrb(30, 30, 70, 70))
	knotB := nodeAt(ltrb(20, 42, 60, 82))
	knotC := nodeAt(ltrb(45, 35, 85, 75))
	for _, dir := range []*ui.TextDirection{ltr(), rtl()} {
		c := cfg()
		c.SetTextDirection(dir)
		knotRoot := upd(nodeAt(ltrb(0, 0, 100, 100)), c, knotA, knotB, knotC)
		slogS("knot " + dir.String() + " " + idsS(knotRoot.DebugListChildrenInOrder(semantics.DebugSemanticsDumpOrderTraversalOrder)))
	}

	knotX := nodeAt(ltrb(40, 40, 60, 50))
	knotY := nodeAt(ltrb(30, 30, 50, 88))
	k2c := cfg()
	k2c.SetTextDirection(ltr())
	knot2 := upd(nodeAt(ltrb(0, 0, 100, 100)), k2c, knotX, knotY)
	slogS("knot2 " + idsS(knot2.DebugListChildrenInOrder(semantics.DebugSemanticsDumpOrderTraversalOrder)))

	owner := newOwnerS("extra")
	root := semantics.NewSemanticsNodeRoot(owner)
	root.SetRect(ltrb(0, 0, 100, 100))
	underC := cfg()
	underC.SetOnTap(func() { slogS("under tapped") })
	under := upd(nodeAt(ltrb(0, 0, 60, 60)), underC)
	overC := cfg()
	overC.SetOnTap(func() { slogS("over tapped") })
	over := upd(nodeAt(ltrb(40, 40, 100, 100)), overC)
	upd(root, cfg(), under, over)
	owner.PerformActionAt(ui.NewOffset(50, 50), ui.SemanticsActionTap)
	owner.PerformActionAt(ui.NewOffset(10, 10), ui.SemanticsActionTap)
	owner.Dispose()

	parent := labelled("p", ltr())
	parent.SetHeadingLevel(2)
	parent.SetTraversalChildIdentifier("c")
	child := cfg()
	child.SetLabel("q")
	child.SetHeadingLevel(3)
	child.SetTraversalParentIdentifier("p")
	parent.Absorb(child)
	slogS("absorbed " + parent.AttributedLabel().String() + " heading=" + strconv.Itoa(parent.HeadingLevel()) +
		" traversal=" + objS(parent.TraversalParentIdentifier()) + "," + objS(parent.TraversalChildIdentifier()))

	fc := cfg()
	fc.SetIsFocused(bp(false))
	fc.SetIsRequired(bp(false))
	fc.SetIsExpanded(bp(true))
	focusable := upd(nodeAt(ltrb(0, 0, 10, 10)), fc)
	slogS("focusable json " + dart.JSONEncode(focusable.ToJson()) + " flags=" + strconv.Itoa(focusable.GetSemanticsData().Flags()))
}

func infS() float64   { return 1 / zeroS() }
func zeroS() float64 { return 0 }

// Same scenes as flutter/test/semantics_test.dart.
func TestSemanticsMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/semantics.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	if !isolated(t) { // it constructs its own binding
		return
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	defer async.SetCurrent(async.NewLoop())()
	b := &semBinding{}
	base := &foundation.BindingBaseBase{}
	b.SchedulerBindingMixin.BindingBase = base
	b.sem.BindingBase = &b.SchedulerBindingMixin
	b.SchedulerBindingMixin.SetSelf(b)
	b.sem.SetSelf(b)
	base.Construct(b)

	semLog = nil
	sceneValuesS()
	sceneTreesAndOrderS()
	sceneMergingS()
	sceneConfigurationS()
	sceneOwnerS()
	sceneTraversalGraftingS()
	sceneRoleChecksS()
	sceneErrorsS()
	sceneActionArgsS()
	sceneExtraS()

	n := min(len(semLog), len(ref.Log))
	for i := 0; i < n; i++ {
		if semLog[i] != ref.Log[i] {
			t.Fatalf("line %d differs:\n go:      %q\n flutter: %q", i, semLog[i], ref.Log[i])
		}
	}
	if len(semLog) != len(ref.Log) {
		t.Fatalf("go logged %d lines, flutter %d", len(semLog), len(ref.Log))
	}
}
