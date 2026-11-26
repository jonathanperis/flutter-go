package services

// Port of packages/flutter/test/services/text_formatter_test.dart (Flutter
// 3.50.1).
//
// Skipped: "test const constructor" (Dart's const canonicalization:
// identical(const X(), const X()); Go has no const objects). The
// "(deprecated names)" and "Whitelisting..." duplicates are ported once
// each, as in Dart they run the same code.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func tev(text string, sel ...int) TextEditingValue {
	o := TextEditingValueOptions{Text: text}
	if len(sel) == 2 {
		s := NewTextSelection(sel[0], sel[1])
		o.Selection = &s
	}
	return NewTextEditingValue(o)
}

func expectValue(t *testing.T, got, want TextEditingValue) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func expectSel(t *testing.T, got, want TextSelection) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func withSel(v TextEditingValue, base, extent int) TextEditingValue {
	s := NewTextSelection(base, extent)
	return v.CopyWith(TextEditingValueCopyWithOptions{Selection: &s})
}

func withComposing(v TextEditingValue, start, end int) TextEditingValue {
	r := ui.NewTextRange(start, end)
	return v.CopyWith(TextEditingValueCopyWithOptions{Composing: &r})
}

func ip(v int) *int { return &v }

func TestTextFormatterWithFunctionWrapsFormattingFunction(t *testing.T) {
	testOldValue, testNewValue := TextEditingValueEmpty, TextEditingValueEmpty
	var calledOld, calledNew TextEditingValue
	f := NewTextInputFormatterWithFunction(func(oldValue, newValue TextEditingValue) TextEditingValue {
		calledOld, calledNew = oldValue, newValue
		return TextEditingValueEmpty
	})
	f.FormatEditUpdate(testOldValue, testNewValue)
	expectValue(t, calledOld, testOldValue)
	expectValue(t, calledNew, testNewValue)
}

func providedNewValue() TextEditingValue { return tev("a1b2c3\nd4e5f6", 3, 9) }

func TestFilteringFormatterExample(t *testing.T) {
	old := TextEditingValueEmpty
	woods := tev("Into the Woods")
	star := FilteringTextInputFormatterOptions{ReplacementString: "*"}
	expectValue(t, NewFilteringTextInputFormatter(dart.StringPattern("o"), true, star).FormatEditUpdate(old, woods), tev("*o*oo*"))
	expectValue(t, NewFilteringTextInputFormatter(dart.StringPattern("o"), false, star).FormatEditUpdate(old, woods), tev("Int* the W**ds"))
	expectValue(t, NewFilteringTextInputFormatter(dart.NewRegExp("o+"), true, star).FormatEditUpdate(old, woods), tev("*o*oo*"))
	expectValue(t, NewFilteringTextInputFormatter(dart.NewRegExp("o+"), false, star).FormatEditUpdate(old, woods), tev("Int* the W*ds"))

	selected := tev("Into the Woods", 11, 14)
	expectValue(t, NewFilteringTextInputFormatter(dart.StringPattern("o"), true, star).FormatEditUpdate(old, selected), tev("*o*oo*", 4, 6))
	expectValue(t, NewFilteringTextInputFormatter(dart.StringPattern("o"), false, star).FormatEditUpdate(old, selected), tev("Int* the W**ds", 11, 14))
	expectValue(t, NewFilteringTextInputFormatter(dart.NewRegExp("o+"), true, star).FormatEditUpdate(old, selected), tev("*o*oo*", 4, 6))
	expectValue(t, NewFilteringTextInputFormatter(dart.NewRegExp("o+"), false, star).FormatEditUpdate(old, selected), tev("Int* the W*ds", 11, 13))
}

func TestFilteringFormatterDenyMode(t *testing.T) {
	got := NewFilteringTextInputFormatterDeny(dart.NewRegExp(`[a-z]`)).FormatEditUpdate(TextEditingValueEmpty, providedNewValue())
	expectValue(t, got, tev("123\n456", 1, 5))
}

func TestSingleLineFormatter(t *testing.T) {
	got := FilteringTextInputFormatterSingleLineFormatter.FormatEditUpdate(TextEditingValueEmpty, providedNewValue())
	expectValue(t, got, tev("a1b2c3d4e5f6", 3, 8))
}

func TestFilteringFormatterAllowMode(t *testing.T) {
	got := NewFilteringTextInputFormatterAllow(dart.NewRegExp(`[a-c]`)).FormatEditUpdate(TextEditingValueEmpty, providedNewValue())
	expectValue(t, got, tev("abc", 2, 3))
}

func TestDigitsOnlyFormatter(t *testing.T) {
	got := FilteringTextInputFormatterDigitsOnly.FormatEditUpdate(TextEditingValueEmpty, providedNewValue())
	expectValue(t, got, tev("123456", 1, 4))
}

func TestLengthLimitingFormatter(t *testing.T) {
	old := TextEditingValueEmpty
	expectValue(t, NewLengthLimitingTextInputFormatter(ip(6)).FormatEditUpdate(old, providedNewValue()), tev("a1b2c3", 3, 6))

	// Zero-length string.
	expectValue(t, NewLengthLimitingTextInputFormatter(ip(1)).FormatEditUpdate(old, tev("", 0, 0)), tev("", 0, 0))

	// Non-BMP Unicode scalar values.
	got := NewLengthLimitingTextInputFormatter(ip(2)).FormatEditUpdate(old, tev("\U0001f984\U0001f984\U0001f984\U0001f984", 8, 8))
	expectValue(t, got, tev("\U0001f984\U0001f984", 4, 4))

	// Complex Unicode characters.
	got = NewLengthLimitingTextInputFormatter(ip(1)).FormatEditUpdate(old, tev("\U0001F984 ", 1, 1))
	expectValue(t, got, tev("\U0001F984", 1, 1))
	got = NewLengthLimitingTextInputFormatter(ip(1)).FormatEditUpdate(old, tev("XY", 1, 1))
	expectValue(t, got, tev("X", 1, 1))

	// Selection off the end.
	expectValue(t, NewLengthLimitingTextInputFormatter(ip(2)).FormatEditUpdate(old, providedNewValue()), tev("a1", 2, 2))
}

func TestLengthLimitingTruncate(t *testing.T) {
	if got := LengthLimitingTextInputFormatterTruncate(tev("01234567890"), 10).Text; got != "0123456789" {
		t.Errorf("removes from the end: %q", got)
	}
	for _, c := range []struct{ overflowing, truncated string }{
		{"😆01234567890", "😆012345678"},
		{"👨‍👩‍👦01234567890", "👨‍👩‍👦012345678"},
	} {
		n := dart.StringLength(c.overflowing)
		got := LengthLimitingTextInputFormatterTruncate(tev(c.overflowing, n, n), 10)
		want := dart.StringLength(c.truncated)
		if got.Text != c.truncated || got.Selection.BaseOffset != want || got.Selection.ExtentOffset != want {
			t.Errorf("got %v", got)
		}
	}
}

func TestLengthLimitingFormatEditUpdate(t *testing.T) {
	f := NewLengthLimitingTextInputFormatter(ip(10))
	if got := f.FormatEditUpdate(tev("aaa"), tev("aaab")).Text; got != "aaab" {
		t.Errorf("passes through: %q", got)
	}
	if got := f.FormatEditUpdate(tev("aaaaaaaaaa"), tev("aaaaabbbbbaaaaa")).Text; got != "aaaaaaaaaa" {
		t.Errorf("uses old value at the limit: %q", got)
	}
	if got := f.FormatEditUpdate(tev("aaaaaaaaaaaaaaaaaaaa"), tev("bbbbbbbbbbbbbbbbbbbb")).Text; got != "bbbbbbbbbb" {
		t.Errorf("truncates: %q", got)
	}
}

func TestGetDefaultMaxLengthEnforcement(t *testing.T) {
	if got := LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(foundation.TargetPlatformWindows); got != MaxLengthEnforcementEnforced {
		t.Errorf("windows: %v", got)
	}
	if got := LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(foundation.TargetPlatformMacOS); got != MaxLengthEnforcementTruncateAfterCompositionEnds {
		t.Errorf("macOS: %v", got)
	}
}

func collapsedValue(text string, offset int) TextEditingValue { return tev(text, offset, offset) }

func TestFilteringFormatterReturnsOldValueForNonAllowedCharacter(t *testing.T) {
	formatted := FilteringTextInputFormatterDigitsOnly.FormatEditUpdate(tev("12345"), tev("12345@"))
	if formatted.Text != "12345" {
		t.Errorf("got %q", formatted.Text)
	}
}

func TestFilteringFormatterMovesTheCursor(t *testing.T) {
	f := FilteringTextInputFormatterDigitsOnly
	oldValue := collapsedValue("123", 0)
	newValue := collapsedValue("123456", 6)
	if got := f.FormatEditUpdate(oldValue, newValue).Selection.BaseOffset; got != 6 {
		t.Errorf("end: %d", got)
	}
	oldValue = withSel(newValue, 4, 4)
	text := "1239456"
	newValue = oldValue.CopyWith(TextEditingValueCopyWithOptions{Text: &text})
	if got := f.FormatEditUpdate(oldValue, newValue).Selection.BaseOffset; got != 4 {
		t.Errorf("middle: %d", got)
	}
}

func TestFilteringFormatterIndependentOfSelectionAndComposing(t *testing.T) {
	f := NewFilteringTextInputFormatterDeny(dart.StringPattern("abc"), FilteringTextInputFormatterOptions{ReplacementString: "*"})
	newValue := tev("abcabcabc")
	filtered := f.FormatEditUpdate(TextEditingValueEmpty, newValue).Text
	for i := 0; i < dart.StringLength(newValue.Text); i++ {
		if got := f.FormatEditUpdate(TextEditingValueEmpty, withSel(newValue, i, i)).Text; got != filtered {
			t.Errorf("selection %d: %q", i, got)
		}
		if got := f.FormatEditUpdate(TextEditingValueEmpty, withComposing(newValue, i, i)).Text; got != filtered {
			t.Errorf("composing %d: %q", i, got)
		}
	}
}

func TestFilteringFormatterBasicFiltering(t *testing.T) {
	f := NewFilteringTextInputFormatterAllow(dart.NewRegExp(`[A-Za-z0-9.@-]*`))
	if got := f.FormatEditUpdate(TextEditingValueEmpty, tev("ab&&ca@bcabc")).Text; got != "abca@bcabc" {
		t.Errorf("got %q", got)
	}
}

func TestFilteringFormatterPreservesSelectionRegion(t *testing.T) {
	old := TextEditingValueEmpty
	nv := tev("AAABBBCCC")
	deny := func(rep string) *FilteringTextInputFormatter {
		return NewFilteringTextInputFormatterDeny(dart.StringPattern("BBB"), FilteringTextInputFormatterOptions{ReplacementString: rep})
	}
	expectSel(t, deny("****").FormatEditUpdate(old, withSel(nv, 6, 3)).Selection, NewTextSelection(7, 3))
	expectSel(t, deny("****").FormatEditUpdate(old, withSel(nv, 9, 3)).Selection, NewTextSelection(10, 3))
	expectSel(t, deny("****").FormatEditUpdate(old, withSel(nv, 9, 6)).Selection, NewTextSelection(10, 7))
	expectSel(t, deny("***").FormatEditUpdate(old, withSel(nv, 5, 4)).Selection, NewTextSelection(5, 4))
	expectSel(t, deny("").FormatEditUpdate(old, withSel(nv, 6, 3)).Selection, NewTextSelection(3, 3))
	expectSel(t, deny("****").FormatEditUpdate(old, withSel(nv, 5, 4)).Selection, NewTextSelection(7, 7))

	allow := func(rep string) *FilteringTextInputFormatter {
		return NewFilteringTextInputFormatterAllow(dart.StringPattern("BBB"), FilteringTextInputFormatterOptions{ReplacementString: rep})
	}
	expectSel(t, allow("****").FormatEditUpdate(old, withSel(nv, 6, 3)).Selection, NewTextSelection(7, 4))
	expectSel(t, allow("****").FormatEditUpdate(old, withSel(nv, 9, 0)).Selection, NewTextSelection(11, 0))
	expectSel(t, allow("****").FormatEditUpdate(old, withSel(nv, 9, 6)).Selection, NewTextSelection(11, 7))
	expectSel(t, allow("").FormatEditUpdate(old, tev("AAABBBBBCCC", 8, 3)).Selection, NewTextSelection(3, 0))
}

func TestFilteringFormatterPreservesComposingRegion(t *testing.T) {
	old := TextEditingValueEmpty
	nv := tev("AAABBBCCC")
	deny := func(rep string) *FilteringTextInputFormatter {
		return NewFilteringTextInputFormatterDeny(dart.StringPattern("BBB"), FilteringTextInputFormatterOptions{ReplacementString: rep})
	}
	expect := func(got, want ui.TextRange) {
		t.Helper()
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	expect(deny("****").FormatEditUpdate(old, withComposing(nv, 3, 6)).Composing, ui.NewTextRange(3, 7))
	expect(deny("****").FormatEditUpdate(old, withComposing(nv, 3, 9)).Composing, ui.NewTextRange(3, 10))
	expect(deny("****").FormatEditUpdate(old, withComposing(nv, 6, 9)).Composing, ui.NewTextRange(7, 10))
	expect(deny("***").FormatEditUpdate(old, withComposing(nv, 4, 5)).Composing, ui.NewTextRange(4, 5))
	expect(deny("").FormatEditUpdate(old, withComposing(nv, 3, 6)).Composing, ui.TextRangeEmpty)
}

func TestMaxLengthEnforcementNames(t *testing.T) {
	if MaxLengthEnforcementTruncateAfterCompositionEnds.String() != "MaxLengthEnforcement.truncateAfterCompositionEnds" ||
		len(MaxLengthEnforcementValues) != 3 {
		t.Error(MaxLengthEnforcementTruncateAfterCompositionEnds.String())
	}
}
