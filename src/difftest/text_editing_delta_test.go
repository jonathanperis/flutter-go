package difftest

// Go twin of flutter/test/text_editing_delta_test.dart: every case's
// TextEditingDelta.fromJSON, its fields, properties and apply result (or
// the assertion it fails), compared exactly. The cases' texts are rebuilt
// from their UTF-16 code units (Go's JSON decoders replace lone
// surrogates, which the cases contain).

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func describeDelta(d services.TextEditingDelta) []string {
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

func runDelta(encoded *dart.LinkedMap) (lines []string) {
	var raw []string
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case dart.AssertionFailure:
				raw = append(raw, "assert "+dart.ToString(e.AssertionMessage()))
			case dart.RangeError:
				raw = append(raw, "range "+e.Error())
			default:
				panic(r)
			}
		}
		for _, l := range raw {
			lines = append(lines, dart.JSONEncode(l))
		}
	}()
	d := services.NewTextEditingDeltaFromJSON(encoded)
	raw = append(raw, dartTypeName(d)+" "+dart.JSONEncode(d.OldText())+" "+d.Selection().String()+" "+d.Composing().String())
	switch x := d.(type) {
	case *services.TextEditingDeltaInsertion:
		raw = append(raw, "inserted "+dart.JSONEncode(x.TextInserted)+" at "+dart.JSONEncode(x.InsertionOffset))
	case *services.TextEditingDeltaDeletion:
		raw = append(raw, "deleted "+x.DeletedRange.String()+" "+dart.JSONEncode(x.TextDeleted()))
	case *services.TextEditingDeltaReplacement:
		raw = append(raw, "replaced "+x.ReplacedRange.String()+" "+dart.JSONEncode(x.TextReplaced())+" with "+dart.JSONEncode(x.ReplacementText))
	case *services.TextEditingDeltaNonTextUpdate:
		raw = append(raw, "nonText")
	}
	raw = append(raw, describeDelta(d)...)
	applied := d.Apply(services.NewTextEditingValue(services.TextEditingValueOptions{Text: "ignored " + d.OldText()}))
	raw = append(raw, "apply "+dart.JSONEncode(applied.Text)+" "+applied.Selection.String()+" "+applied.Composing.String())
	return
}

func unitsString(units []uint16) string { return dart.StringFromCharCodes(units) }

func TestTextEditingDeltaMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/text_editing_delta.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct {
		Cases []struct {
			JSON       string   `json:"json"`
			OldUnits   []uint16 `json:"oldUnits"`
			DeltaUnits []uint16 `json:"deltaUnits"`
			Out        []string `json:"out"`
		}
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	failures := 0
	for i, c := range ref.Cases {
		encoded := services.JSONMessageCodec{}.DecodeMessage([]byte(c.JSON)).(*dart.LinkedMap)
		if c.OldUnits != nil {
			encoded.Set("oldText", unitsString(c.OldUnits))
			encoded.Set("deltaText", unitsString(c.DeltaUnits))
		}
		got := runDelta(encoded)
		if strings.Join(got, "\n") != strings.Join(c.Out, "\n") {
			t.Errorf("case %d %s:\n Go:      %q\n Flutter: %q", i, c.JSON, got, c.Out)
			if failures++; failures > 5 {
				t.FailNow()
			}
		}
	}
	t.Logf("%d cases identical", len(ref.Cases))
}
