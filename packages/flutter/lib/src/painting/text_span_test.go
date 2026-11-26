package painting

// Port of packages/flutter/test/painting/text_span_test.dart (Flutter
// 3.50.1). WidgetSpan (a widgets class) is the test stand-in
// ptextbWidgetSpan (ptextb_widget_span_test.go), whose children are
// comparable values standing for the tests' const widgets.
//
// Skipped:
//   - "handles mouse cursor" and "handles onEnter and onExit": they pump a
//     Text.rich widget and hover it with a mouse TestGesture
//     (RenderParagraph, the mouse tracker and TestGesture are not ported).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

func tspText(text string, children ...InlineSpan) *TextSpan {
	return NewTextSpan(TextSpanOptions{Text: &text, Children: children})
}

func tspChildren(children ...InlineSpan) *TextSpan {
	return NewTextSpan(TextSpanOptions{Children: children})
}

func tspCallback1(*gestures.PointerEnterEvent) {}
func tspCallback2(*gestures.PointerEnterEvent) {}

func TestTextSpanEquals(t *testing.T) {
	a1 := tspText("a")
	a2 := tspText("a")
	b1 := tspChildren(a1)
	b2 := tspChildren(a2)
	c1 := NewTextSpan()
	c2 := NewTextSpan()

	check := func(got, want bool, what string) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %v", what, got)
		}
	}
	check(a1.Equal(a2), true, "a1 == a2")
	check(b1.Equal(b2), true, "b1 == b2")
	check(c1.Equal(c2), true, "c1 == c2")
	check(a1.Equal(b2), false, "a1 == b2")
	check(b1.Equal(c2), false, "b1 == c2")
	check(c1.Equal(a2), false, "c1 == a2")
	check(a1.Equal(c2), false, "a1 == c2")
	check(b1.Equal(a2), false, "b1 == a2")
	check(c1.Equal(b2), false, "c1 == b2")

	a := "a"
	d1 := NewTextSpan(TextSpanOptions{Text: &a, OnEnter: tspCallback1})
	d2 := NewTextSpan(TextSpanOptions{Text: &a, OnEnter: tspCallback1})
	d3 := NewTextSpan(TextSpanOptions{Text: &a, OnEnter: tspCallback2})
	e1 := NewTextSpan(TextSpanOptions{Text: &a, OnEnter: tspCallback2, MouseCursor: services.SystemMouseCursors.Forbidden})
	e2 := NewTextSpan(TextSpanOptions{Text: &a, OnEnter: tspCallback2, MouseCursor: services.SystemMouseCursors.Forbidden})
	check(a1.Equal(d1), false, "a1 == d1")
	check(d1.Equal(d2), true, "d1 == d2")
	check(d2.Equal(d3), false, "d2 == d3")
	check(d3.Equal(e1), false, "d3 == e1")
	check(e1.Equal(e2), true, "e1 == e2")
}

func TestTextSpanToStringDeep(t *testing.T) {
	a := "a"
	span := NewTextSpan(TextSpanOptions{
		Text:  &a,
		Style: NewTextStyle(TextStyleOptions{FontSize: f64(10.0)}),
		Children: []InlineSpan{
			tspText("b", NewTextSpan()),
			tspText("c"),
		},
	})
	want := "TextSpan:\n" +
		"  inherit: true\n" +
		"  size: 10.0\n" +
		"  \"a\"\n" +
		"  TextSpan:\n" +
		"    \"b\"\n" +
		"    TextSpan:\n" +
		"      (empty)\n" +
		"  TextSpan:\n" +
		"    \"c\"\n"
	if got := span.ToStringDeep(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestTextSpanToStringDeepForMouse(t *testing.T) {
	if got := tspText("a").ToStringDeep(); got != "TextSpan:\n  \"a\"\n" {
		t.Errorf("got %q", got)
	}
	a := "a"
	test2 := NewTextSpan(TextSpanOptions{
		Text:        &a,
		OnEnter:     func(*gestures.PointerEnterEvent) {},
		OnExit:      func(*gestures.PointerExitEvent) {},
		MouseCursor: services.SystemMouseCursors.Forbidden,
	})
	want := "TextSpan:\n" +
		"  \"a\"\n" +
		"  callbacks: enter, exit\n" +
		"  mouseCursor: SystemMouseCursor(forbidden)\n"
	if got := test2.ToStringDeep(); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestTextSpanToPlainText(t *testing.T) {
	if got := tspText("a", tspText("b"), tspText("c")).ToPlainText(); got != "abc" {
		t.Error(got)
	}
}

func TestWidgetSpanToPlainText(t *testing.T) {
	span := tspText("a", tspText("b"), newPtextbWidgetSpan(ptextbSizedBox{10, 10}), tspText("c"))
	if got := span.ToPlainText(); got != "ab￼c" {
		t.Errorf("%q", got)
	}
}

func TestTextSpanToPlainTextWithSemanticsLabel(t *testing.T) {
	b, foo := "b", "foo"
	span := tspText("a", NewTextSpan(TextSpanOptions{Text: &b, SemanticsLabel: &foo}), tspText("c"))
	if got := span.ToPlainText(); got != "afooc" {
		t.Error(got)
	}
	f := false
	if got := span.ToPlainText(ToPlainTextOptions{IncludeSemanticsLabels: &f}); got != "abc" {
		t.Error(got)
	}
}

func TestTextSpanWidgetChange(t *testing.T) {
	top := ui.PlaceholderAlignmentTop
	mk := func(child any, opts ...PlaceholderSpanOptions) *TextSpan {
		return tspText("a", tspText("b"), newPtextbWidgetSpan(child, opts...), tspText("c"))
	}
	textSpan1 := mk(ptextbSizedBox{10, 10})
	textSpan2 := mk(ptextbSizedBox{10, 10})
	textSpan3 := mk(ptextbSizedBox{11, 10})
	textSpan4 := mk(ptextbTextRich{"test"})
	textSpan5 := mk(ptextbTextRich{"different!"})
	textSpan6 := mk(ptextbSizedBox{10, 10}, PlaceholderSpanOptions{Alignment: &top})

	check := func(got, want RenderComparison) {
		t.Helper()
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	check(textSpan1.CompareTo(textSpan3), RenderComparisonLayout)
	check(textSpan1.CompareTo(textSpan4), RenderComparisonLayout)
	check(textSpan1.CompareTo(textSpan1), RenderComparisonIdentical)
	check(textSpan2.CompareTo(textSpan2), RenderComparisonIdentical)
	check(textSpan3.CompareTo(textSpan3), RenderComparisonIdentical)
	check(textSpan2.CompareTo(textSpan3), RenderComparisonLayout)
	check(textSpan4.CompareTo(textSpan5), RenderComparisonLayout)
	check(textSpan3.CompareTo(textSpan5), RenderComparisonLayout)
	check(textSpan2.CompareTo(textSpan5), RenderComparisonLayout)
	check(textSpan1.CompareTo(textSpan5), RenderComparisonLayout)
	check(textSpan1.CompareTo(textSpan6), RenderComparisonLayout)
	// Not in Dart's test: equal const children compare identical.
	check(textSpan1.CompareTo(textSpan2), RenderComparisonIdentical)
}

func TestTextSpanNestedWidgetChange(t *testing.T) {
	textSpan1 := tspText("a", tspText("b"),
		newPtextbWidgetSpan(ptextbTextRich{"[WidgetSpan(SizedBox(10, 10)), 'The sky is falling :)']"}), tspText("c"))
	textSpan2 := tspText("a", tspText("b"),
		newPtextbWidgetSpan(ptextbTextRich{"[WidgetSpan(SizedBox(10, 11)), 'The sky is falling :)']"}), tspText("c"))
	if got := textSpan1.CompareTo(textSpan2); got != RenderComparisonLayout {
		t.Error(got)
	}
	if textSpan1.CompareTo(textSpan1) != RenderComparisonIdentical || textSpan2.CompareTo(textSpan2) != RenderComparisonIdentical {
		t.Error("identical")
	}
}

func tspTextOf(s InlineSpan) any {
	ts, ok := s.(*TextSpan)
	if !ok || ts == nil || ts.Text == nil {
		return nil
	}
	return *ts.Text
}

func TestGetSpanForPosition(t *testing.T) {
	span := tspText("", tspText("", tspText("a")), tspText("b"), tspText("c"))
	for i, want := range []any{"a", "b", "c", nil} {
		if got := tspTextOf(span.GetSpanForPosition(ui.NewTextPosition(i))); got != want {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestGetSpanForPositionWithWidgetSpan(t *testing.T) {
	span := tspText("a", tspText("b"),
		newPtextbWidgetSpan(ptextbTextRich{"[WidgetSpan(SizedBox(10, 10)), 'The sky is falling :)']"}), tspText("c"))
	for i, wantWidget := range []bool{false, false, true, false} {
		got := span.GetSpanForPosition(ui.NewTextPosition(i))
		_, isWidget := got.(*ptextbWidgetSpan)
		_, isText := got.(*TextSpan)
		if isWidget != wantWidget || isText == wantWidget {
			t.Errorf("%d: got %T", i, got)
		}
	}
}

func TestTextSpanComputeSemanticsInformation(t *testing.T) {
	collector := []InlineSpanSemanticsInformation{}
	aaa, bbb, ccc := "aaa", "bbb", "ccc"
	NewTextSpan(TextSpanOptions{Text: &aaa, SemanticsLabel: &bbb, SemanticsIdentifier: &ccc}).ComputeSemanticsInformation(&collector)
	if collector[0].Text != "aaa" || *collector[0].SemanticsLabel != "bbb" || *collector[0].SemanticsIdentifier != "ccc" {
		t.Errorf("%v", collector[0])
	}
}

func TestTextSpanVisitDirectChildren(t *testing.T) {
	directChildrenOf := func(root InlineSpan) []InlineSpan {
		visitOrder := []InlineSpan{}
		root.VisitDirectChildren(func(span InlineSpan) bool {
			visitOrder = append(visitOrder, span)
			return true
		})
		return visitOrder
	}
	leaf1 := tspText("leaf1")
	leaf2 := tspText("leaf2")
	branch1 := tspChildren(leaf1, leaf2)
	branch2 := tspText("branch2")
	root := tspChildren(branch1, branch2)

	same := func(got []InlineSpan, want ...InlineSpan) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("got %d children, want %d", len(got), len(want))
			return
		}
		for i := range got {
			if !got[i].Equal(want[i]) {
				t.Errorf("child %d differs", i)
			}
		}
	}
	same(directChildrenOf(root), branch1, branch2)
	same(directChildrenOf(branch1), leaf1, leaf2)
	same(directChildrenOf(branch2))
	same(directChildrenOf(leaf1))
	same(directChildrenOf(leaf2))

	indexInTree := func(target InlineSpan) *int {
		index := 0
		var findInSubtree func(subtreeRoot InlineSpan) bool
		findInSubtree = func(subtreeRoot InlineSpan) bool {
			if target == subtreeRoot {
				// return false to stop traversal.
				return false
			}
			index++
			return subtreeRoot.VisitDirectChildren(findInSubtree)
		}
		if findInSubtree(root) {
			return nil
		}
		return &index
	}
	for want, span := range []InlineSpan{root, branch1, leaf1, leaf2, branch2} {
		if got := indexInTree(span); got == nil || *got != want {
			t.Errorf("indexInTree %d: %v", want, got)
		}
	}
	if indexInTree(tspText("foobar")) != nil {
		t.Error("foobar")
	}
}

func TestTextSpanComputesStringAttributes(t *testing.T) {
	tr, fa := true, false
	esMX := ui.NewLocale("es", "MX")
	y, x, a := "yyyyy", "xxxxx", "aaaaa"
	span := NewTextSpan(TextSpanOptions{
		Text:     &a,
		SpellOut: &tr,
		Children: []InlineSpan{
			NewTextSpan(TextSpanOptions{Text: &y, Locale: &esMX}),
			NewTextSpan(TextSpanOptions{
				Text:     &x,
				SpellOut: &fa,
				Children: []InlineSpan{
					tspText("zzzzz"),
					NewTextSpan(TextSpanOptions{Text: strPtr("bbbbb"), SpellOut: &tr}),
				},
			}),
		},
	})
	collector := []InlineSpanSemanticsInformation{}
	span.ComputeSemanticsInformation(&collector)

	type attr struct {
		spellOut   bool
		start, end int
	}
	checkAttrs := func(what string, got []ui.StringAttribute, want ...attr) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("%s: %d attributes, want %d", what, len(got), len(want))
			return
		}
		for i, w := range want {
			var ok bool
			if w.spellOut {
				_, ok = got[i].(*ui.SpellOutStringAttribute)
			} else {
				var l *ui.LocaleStringAttribute
				l, ok = got[i].(*ui.LocaleStringAttribute)
				if ok && !l.Locale.Equal(ui.NewLocale("es", "MX")) {
					t.Errorf("%s[%d]: locale %v", what, i, l)
				}
			}
			if !ok || got[i].Range() != ui.NewTextRange(w.start, w.end) {
				t.Errorf("%s[%d]: got %v", what, i, got[i])
			}
		}
	}
	if len(collector) != 5 {
		t.Fatalf("%d entries", len(collector))
	}
	checkAttrs("0", collector[0].StringAttributes, attr{true, 0, 5})
	checkAttrs("1", collector[1].StringAttributes, attr{true, 0, 5}, attr{false, 0, 5})
	checkAttrs("2", collector[2].StringAttributes)
	checkAttrs("3", collector[3].StringAttributes)
	checkAttrs("4", collector[4].StringAttributes, attr{true, 0, 5})

	combined := CombineSemanticsInfo(collector)
	if len(combined) != 1 {
		t.Fatalf("%d combined", len(combined))
	}
	checkAttrs("combined", combined[0].StringAttributes, attr{true, 0, 5}, attr{true, 5, 10}, attr{false, 5, 10}, attr{true, 20, 25})
}
