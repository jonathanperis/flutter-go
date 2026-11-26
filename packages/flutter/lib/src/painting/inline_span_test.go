package painting

// Tests of inline_span.dart (Flutter 3.50.1), which has no test file of
// its own in Flutter: Accumulator, InlineSpanSemanticsInformation (its
// asserts, ==, toString, placeholder), combineSemanticsInfo and the
// template methods of InlineSpan (codeUnitAt, getSpanForPosition,
// toPlainText, getSemanticsInformation, ==).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
)

func TestAccumulator(t *testing.T) {
	a := NewAccumulator()
	if a.Value() != 0 {
		t.Error(a.Value())
	}
	a.Increment(3)
	a.Increment(0)
	if a.Value() != 3 {
		t.Error(a.Value())
	}
	if NewAccumulator(7).Value() != 7 {
		t.Error("initial value")
	}
	if got := tsAssertMessage(func() { a.Increment(-1) }); got == "no error" {
		t.Error("negative addend must assert")
	}
}

func TestInlineSpanSemanticsInformation(t *testing.T) {
	label, id := "label", "id"
	plain := NewInlineSpanSemanticsInformation("text")
	if plain.RequiresOwnNode || plain.IsPlaceholder || plain.StringAttributes == nil || len(plain.StringAttributes) != 0 {
		t.Errorf("plain %+v", plain)
	}
	if !NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{SemanticsIdentifier: &id}).RequiresOwnNode {
		t.Error("identifier requires own node")
	}
	recognizer := gestures.NewTapGestureRecognizer()
	withRecognizer := NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{Recognizer: recognizer})
	if !withRecognizer.RequiresOwnNode {
		t.Error("recognizer requires own node")
	}
	p := InlineSpanSemanticsInformationPlaceholder
	if !p.IsPlaceholder || !p.RequiresOwnNode || p.Text != "￼" {
		t.Errorf("placeholder %+v", p)
	}
	for _, bad := range []func(){
		func() {
			NewInlineSpanSemanticsInformation("x", InlineSpanSemanticsInformationOptions{IsPlaceholder: true})
		},
		func() {
			NewInlineSpanSemanticsInformation("￼", InlineSpanSemanticsInformationOptions{IsPlaceholder: true, SemanticsLabel: &label})
		},
		func() {
			NewInlineSpanSemanticsInformation("￼", InlineSpanSemanticsInformationOptions{IsPlaceholder: true, Recognizer: recognizer})
		},
	} {
		if got := tsAssertMessage(bad); got == "no error" {
			t.Error("invalid placeholder must assert")
		}
	}

	// ==
	a := NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{SemanticsLabel: &label})
	if !a.Equal(NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{SemanticsLabel: strPtr("label")})) {
		t.Error("equal")
	}
	for _, other := range []InlineSpanSemanticsInformation{
		NewInlineSpanSemanticsInformation("u", InlineSpanSemanticsInformationOptions{SemanticsLabel: &label}),
		NewInlineSpanSemanticsInformation("t"),
		NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{SemanticsLabel: &label, SemanticsIdentifier: &id}),
		NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{SemanticsLabel: &label, Recognizer: recognizer}),
	} {
		if a.Equal(other) {
			t.Errorf("%v == %v", a, other)
		}
	}
	// String attributes compare by identity (listEquals on attributes
	// without ==).
	attr := ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 1))
	x1 := NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{StringAttributes: []ui.StringAttribute{attr}})
	x2 := NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{StringAttributes: []ui.StringAttribute{attr}})
	x3 := NewInlineSpanSemanticsInformation("t", InlineSpanSemanticsInformationOptions{
		StringAttributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 1))}})
	if !x1.Equal(x2) || x1.Equal(x3) || x1.Equal(NewInlineSpanSemanticsInformation("t")) {
		t.Error("attributes by identity")
	}

	want := "InlineSpanSemanticsInformation{text: t, semanticsLabel: label, semanticsIdentifier: null, recognizer: null}"
	if got := a.String(); got != want {
		t.Errorf("got %q", got)
	}
}

func TestCombineSemanticsInfo(t *testing.T) {
	label := "LABEL"
	recognizer := gestures.NewTapGestureRecognizer()
	attr := ui.NewSpellOutStringAttribute(ui.NewTextRange(1, 3))
	infos := []InlineSpanSemanticsInformation{
		NewInlineSpanSemanticsInformation("ab"),
		NewInlineSpanSemanticsInformation("cd", InlineSpanSemanticsInformationOptions{SemanticsLabel: &label,
			StringAttributes: []ui.StringAttribute{attr}}),
		NewInlineSpanSemanticsInformation("tap", InlineSpanSemanticsInformationOptions{Recognizer: recognizer}),
		NewInlineSpanSemanticsInformation("😀x", InlineSpanSemanticsInformationOptions{StringAttributes: []ui.StringAttribute{attr}}),
		InlineSpanSemanticsInformationPlaceholder,
	}
	combined := CombineSemanticsInfo(infos)
	if len(combined) != 5 {
		t.Fatalf("%d entries: %v", len(combined), combined)
	}
	c0 := combined[0]
	if c0.Text != "abcd" || *c0.SemanticsLabel != "abLABEL" || len(c0.StringAttributes) != 1 ||
		c0.StringAttributes[0].Range() != ui.NewTextRange(3, 5) {
		t.Errorf("combined[0] %v %v", c0, c0.StringAttributes)
	}
	if !combined[1].Equal(infos[2]) {
		t.Errorf("combined[1] %v", combined[1])
	}
	// The emoji is two UTF-16 code units; the attribute isn't shifted (the
	// label was reset by the recognizer's node).
	c2 := combined[2]
	if c2.Text != "😀x" || *c2.SemanticsLabel != "😀x" || c2.StringAttributes[0].Range() != ui.NewTextRange(1, 3) {
		t.Errorf("combined[2] %v", c2)
	}
	if !combined[3].Equal(InlineSpanSemanticsInformationPlaceholder) {
		t.Errorf("combined[3] %v", combined[3])
	}
	last := combined[4]
	if last.Text != "" || *last.SemanticsLabel != "" || len(last.StringAttributes) != 0 {
		t.Errorf("combined[4] %v", last)
	}

	// Labels after a supplementary character shift by UTF-16 length.
	shifted := CombineSemanticsInfo([]InlineSpanSemanticsInformation{
		NewInlineSpanSemanticsInformation("😀"),
		NewInlineSpanSemanticsInformation("y", InlineSpanSemanticsInformationOptions{StringAttributes: []ui.StringAttribute{attr}}),
	})
	if len(shifted) != 1 || shifted[0].StringAttributes[0].Range() != ui.NewTextRange(3, 5) {
		t.Errorf("shifted %v", shifted)
	}
	if got := CombineSemanticsInfo(nil); len(got) != 1 || got[0].Text != "" {
		t.Errorf("empty %v", got)
	}
}

func TestInlineSpanTemplateMethods(t *testing.T) {
	span := tspText("ab", tspText("😀"), newPtextbWidgetSpan(ptextbSizedBox{1, 1}), NewTextSpan(), tspText("c"))
	wantUnits := []int{'a', 'b', 0xD83D, 0xDE00, 0xFFFC, 'c'}
	for i, w := range wantUnits {
		if got := span.CodeUnitAt(i); got == nil || *got != w {
			t.Errorf("codeUnitAt(%d) = %v, want %#x", i, got, w)
		}
	}
	for _, i := range []int{-1, len(wantUnits), 100} {
		if got := span.CodeUnitAt(i); got != nil {
			t.Errorf("codeUnitAt(%d) = %d", i, *got)
		}
	}
	if got := span.ToPlainText(); got != "ab😀￼c" {
		t.Errorf("%q", got)
	}
	f := false
	if got := span.ToPlainText(ToPlainTextOptions{IncludePlaceholders: &f}); got != "ab😀c" {
		t.Errorf("%q", got)
	}
	info := span.GetSemanticsInformation()
	if len(info) != 4 || info[0].Text != "ab" || info[1].Text != "😀" || !info[2].Equal(InlineSpanSemanticsInformationPlaceholder) ||
		info[3].Text != "c" {
		t.Errorf("%v", info)
	}
	// Upstream affinity at the end of a span returns that span.
	if got := tspTextOf(span.GetSpanForPosition(ui.TextPosition{Offset: 2, Affinity: ui.TextAffinityUpstream})); got != "ab" {
		t.Errorf("upstream: %v", got)
	}
	if got := tspTextOf(span.GetSpanForPosition(ui.NewTextPosition(2))); got != "😀" {
		t.Errorf("downstream: %v", got)
	}

	// InlineSpan's ==: the style (and runtime type) only, for a type that
	// doesn't override it beyond the base.
	s1 := NewTextStyle(TextStyleOptions{FontSize: f64(3)})
	w1 := newPtextbWidgetSpan(ptextbSizedBox{1, 1}, PlaceholderSpanOptions{Style: s1})
	if !w1.InlineSpanBase.Equal(newPtextbWidgetSpan(ptextbSizedBox{2, 2}, PlaceholderSpanOptions{Style: NewTextStyle(TextStyleOptions{FontSize: f64(3)})})) {
		t.Error("base == compares the style")
	}
	if w1.InlineSpanBase.Equal(tspText("x")) || w1.InlineSpanBase.Equal(nil) {
		t.Error("base == compares the runtime type")
	}
}
