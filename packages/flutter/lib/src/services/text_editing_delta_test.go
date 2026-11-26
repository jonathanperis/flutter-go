package services_test

// Port of packages/flutter/test/services/text_editing_delta_test.dart
// (Flutter 3.50.1); nothing skipped.

import (
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func deltaFromJSON(t *testing.T, json string) services.TextEditingDelta {
	t.Helper()
	return services.NewTextEditingDeltaFromJSON(jsonDecode(json).(map[string]any))
}

func deltaJSON(oldText, deltaText string, start, end, selection int, composingBase, composingExtent int) string {
	return `{"oldText": ` + dart.JSONEncode(oldText) + `, "deltaText": ` + dart.JSONEncode(deltaText) +
		`, "deltaStart": ` + deltaItoa(start) + `, "deltaEnd": ` + deltaItoa(end) +
		`, "selectionBase": ` + deltaItoa(selection) + `, "selectionExtent": ` + deltaItoa(selection) +
		`, "selectionAffinity" : "TextAffinity.downstream", "selectionIsDirectional": false` +
		`, "composingBase": ` + deltaItoa(composingBase) + `, "composingExtent": ` + deltaItoa(composingExtent) + `}`
}

func deltaItoa(i int) string { return dart.JSONEncode(i) }

func checkDelta(t *testing.T, d services.TextEditingDelta, oldText string, selection services.TextSelection, composing ui.TextRange) {
	t.Helper()
	if d.OldText() != oldText || !d.Selection().Equal(selection) || d.Composing() != composing {
		t.Fatalf("delta %v: %q %v %v", d, d.OldText(), d.Selection(), d.Composing())
	}
}

func TestTextEditingDeltaInsertionCreation(t *testing.T) {
	d := deltaFromJSON(t, deltaJSON("", "let there be text", 0, 0, 17, -1, -1)).(*services.TextEditingDeltaInsertion)
	checkDelta(t, d, "", collapsed(17), ui.TextRangeEmpty)
	if d.TextInserted != "let there be text" || d.InsertionOffset != 0 {
		t.Fatalf("%q %d", d.TextInserted, d.InsertionOffset)
	}
	d = deltaFromJSON(t, deltaJSON("hello worl", "world", 6, 10, 11, 6, 11)).(*services.TextEditingDeltaInsertion)
	checkDelta(t, d, "hello worl", collapsed(11), ui.NewTextRange(6, 11))
	if d.TextInserted != "d" || d.InsertionOffset != 10 {
		t.Fatalf("%q %d", d.TextInserted, d.InsertionOffset)
	}
}

func expectAssertion(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if _, ok := r.(dart.AssertionFailure); !ok {
			t.Fatalf("panic %v, want an AssertionError", r)
		}
	}()
	f()
}

func TestTextEditingDeltaInvalidFailsToApply(t *testing.T) {
	expectAssertion(t, func() {
		services.NewTextEditingDeltaInsertion("hello worl", "d", 11, collapsed(11), ui.TextRangeEmpty).Apply(services.TextEditingValueEmpty)
	})
	expectAssertion(t, func() {
		services.NewTextEditingDeltaDeletion("hello world", ui.NewTextRange(5, 12), collapsed(5), ui.TextRangeEmpty).Apply(services.TextEditingValueEmpty)
	})
	expectAssertion(t, func() {
		services.NewTextEditingDeltaReplacement("hello worl", "world", ui.NewTextRange(5, 11), collapsed(11), ui.TextRangeEmpty).Apply(services.TextEditingValueEmpty)
	})
	expectAssertion(t, func() {
		services.NewTextEditingDeltaNonTextUpdate("hello world", collapsed(12), ui.TextRangeEmpty).Apply(services.TextEditingValueEmpty)
	})
}

func describeDelta(d foundation.Diagnosticable) []string {
	builder := foundation.NewDiagnosticPropertiesBuilder()
	d.DebugFillProperties(builder)
	var out []string
	for _, node := range builder.Properties {
		if !node.IsFiltered(foundation.DiagnosticLevelInfo) {
			out = append(out, node.String())
		}
	}
	return out
}

func TestTextEditingDeltaDebugFillProperties(t *testing.T) {
	cases := []struct {
		delta foundation.Diagnosticable
		want  []string
	}{
		{services.NewTextEditingDeltaInsertion("hello worl", "d", 10, collapsed(11), ui.TextRangeEmpty), []string{
			"oldText: hello worl", "textInserted: d", "insertionOffset: 10",
			"selection: TextSelection.collapsed(offset: 11, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: -1, end: -1)",
		}},
		{services.NewTextEditingDeltaDeletion("hello world", ui.NewTextRange(6, 10), collapsed(6), ui.TextRangeEmpty), []string{
			"oldText: hello world", "textDeleted: worl", "deletedRange: TextRange(start: 6, end: 10)",
			"selection: TextSelection.collapsed(offset: 6, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: -1, end: -1)",
		}},
		{services.NewTextEditingDeltaReplacement("hello world", "h", ui.NewTextRange(6, 11), collapsed(7), ui.NewTextRange(6, 7)), []string{
			"oldText: hello world", "textReplaced: world", "replacementText: h", "replacedRange: TextRange(start: 6, end: 11)",
			"selection: TextSelection.collapsed(offset: 7, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: 6, end: 7)",
		}},
		{services.NewTextEditingDeltaNonTextUpdate("hello world", collapsed(7), ui.NewTextRange(6, 7)), []string{
			"oldText: hello world",
			"selection: TextSelection.collapsed(offset: 7, affinity: TextAffinity.downstream, isDirectional: false)",
			"composing: TextRange(start: 6, end: 7)",
		}},
	}
	for _, c := range cases {
		if got := describeDelta(c.delta); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%T:\n got  %q\n want %q", c.delta, got, c.want)
		}
	}
}

func TestTextEditingDeltaDeletionCreation(t *testing.T) {
	d := deltaFromJSON(t, deltaJSON("let there be text.", "", 1, 2, 1, -1, -1)).(*services.TextEditingDeltaDeletion)
	checkDelta(t, d, "let there be text.", collapsed(1), ui.TextRangeEmpty)
	if d.TextDeleted() != "e" || d.DeletedRange != ui.NewTextRange(1, 2) {
		t.Fatalf("%q %v", d.TextDeleted(), d.DeletedRange)
	}
	d = deltaFromJSON(t, deltaJSON("hello world", "worl", 6, 11, 10, 6, 10)).(*services.TextEditingDeltaDeletion)
	checkDelta(t, d, "hello world", collapsed(10), ui.NewTextRange(6, 10))
	if d.TextDeleted() != "d" || d.DeletedRange != ui.NewTextRange(10, 11) {
		t.Fatalf("%q %v", d.TextDeleted(), d.DeletedRange)
	}
}

func TestTextEditingDeltaReplacementCreation(t *testing.T) {
	cases := []struct {
		oldText, deltaText       string
		selection                int
		composingBase, composing int
		replaced                 string
	}{
		{"hello worfi", "working", 13, 6, 13, "worfi"},
		{"hello world", "h", 7, 6, 7, "world"},
		{"hello world", "words", 11, 6, 11, "world"},
	}
	for _, c := range cases {
		d := deltaFromJSON(t, deltaJSON(c.oldText, c.deltaText, 6, 11, c.selection, c.composingBase, c.composing)).(*services.TextEditingDeltaReplacement)
		checkDelta(t, d, c.oldText, collapsed(c.selection), ui.NewTextRange(c.composingBase, c.composing))
		if d.TextReplaced() != c.replaced || d.ReplacementText != c.deltaText || d.ReplacedRange != ui.NewTextRange(6, 11) {
			t.Fatalf("%q %q %v", d.TextReplaced(), d.ReplacementText, d.ReplacedRange)
		}
	}
}

func TestTextEditingDeltaNonTextUpdateCreation(t *testing.T) {
	d := deltaFromJSON(t, deltaJSON("hello world", "", -1, -1, 10, 6, 11)).(*services.TextEditingDeltaNonTextUpdate)
	checkDelta(t, d, "hello world", collapsed(10), ui.NewTextRange(6, 11))
}
