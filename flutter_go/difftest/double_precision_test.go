package difftest

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// Same values as flutter/test/double_precision_test.dart (read from its
// output as bit patterns).
func TestDoublePrecisionMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/double_precision.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct {
		Values  []string
		Results [][]string
		Debug   [][]string
		Errors  []string
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	values := make([]float64, len(ref.Values))
	for i, hex := range ref.Values {
		bits, err := strconv.ParseUint(hex, 16, 64)
		if err != nil {
			t.Fatal(err)
		}
		values[i] = math.Float64frombits(bits)
	}
	mismatches := 0
	for i, v := range values {
		for p := 1; p <= 21; p++ {
			if got, want := dart.ToStringAsPrecision(v, p), ref.Results[i][p-1]; got != want {
				if mismatches++; mismatches <= 20 {
					t.Errorf("%v (%s).toStringAsPrecision(%d): Go %q, Dart %q", v, ref.Values[i], p, got, want)
				}
			}
		}
	}
	for i, row := range ref.Debug {
		v := values[i]
		got := []string{foundation.DebugFormatDouble(&v)}
		for p := 1; p <= 21; p++ {
			foundation.DebugDoublePrecision = &p
			got = append(got, foundation.DebugFormatDouble(&v))
		}
		foundation.DebugDoublePrecision = nil
		for j := range row {
			if got[j] != row[j] {
				if mismatches++; mismatches <= 20 {
					t.Errorf("debugFormatDouble(%v) with precision %d: Go %q, Dart %q", v, j, got[j], row[j])
				}
			}
		}
	}
	for i, p := range []int{0, 22} {
		func() {
			defer func() {
				r := recover()
				if e, ok := r.(error); !ok || e.Error() != ref.Errors[i] {
					t.Errorf("precision %d: Go %v, Dart %q", p, r, ref.Errors[i])
				}
			}()
			dart.ToStringAsPrecision(1, p)
		}()
	}
	if mismatches > 0 {
		t.Errorf("%d mismatches", mismatches)
	}
}
