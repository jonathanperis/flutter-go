package difftest

// Differential test of painting's TextScaler (text_scaler.dart) and the
// text_painter.dart enums ported so far, against
// flutter/test/painting_text_scaler_test.dart: every line compared exactly
// (doubles through Dart's toString). Dart's `identical(clamp(), base)` is
// printed only for the user-defined scaler (a pointer in Go): the linear
// scaler is a Go value, whose identity Go can't observe.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type tsCustom struct {
	painting.TextScalerBase
	k float64
}

func (s *tsCustom) Scale(fontSize float64) float64 { return float64(fontSize*s.k) + 1 }
func (s *tsCustom) TextScaleFactor() float64       { return s.k }
func (s *tsCustom) String() string                 { return "custom(" + dart.DoubleToString(s.k) + ")" }

func tsVal(f func() string) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*dart.AssertionError); ok {
				s = "AssertionError"
				return
			}
			panic(r)
		}
	}()
	return f()
}

func tsErr(f func()) string { return tsVal(func() string { f(); return "no error" }) }

func tsList(vs []float64) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = dart.DoubleToString(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func tsDescribe(s painting.TextScaler) string {
	var vals []string
	for _, f := range []float64{0, 1, 9.5, 14, 33.3} {
		vals = append(vals, tsVal(func() string { return dart.DoubleToString(s.Scale(f)) }))
	}
	return dart.ToString(s) + " tsf=" + dart.DoubleToString(s.TextScaleFactor()) + " " + strings.Join(vals, ",")
}

func tsClampOpts(r []float64) painting.TextScalerClampOptions {
	lo, hi := r[0], r[1]
	return painting.TextScalerClampOptions{MinScaleFactor: &lo, MaxScaleFactor: &hi}
}

func paintingTextScalerLines() []string {
	inf, nan := math.Inf(1), math.NaN()
	ranges := [][]float64{{0, inf}, {0, 1}, {0.5, 2}, {1, 1}, {2, 3}, {1.1, 1.3}, {4, 8}, {0.3, inf}, {2.5, 2.5}}
	var out []string
	var built []painting.TextScaler
	custom := &tsCustom{k: 1.5}
	custom.SetSelf(custom)
	var bases []painting.TextScaler
	for _, f := range []float64{0, 0.5, 1, 1.25, 3, 7.5} {
		bases = append(bases, painting.NewTextScalerLinear(f))
	}
	bases = append(bases, custom)
	for _, base := range bases {
		out = append(out, "base: "+tsDescribe(base))
		for _, r := range ranges {
			c := base.Clamp(tsClampOpts(r))
			built = append(built, c)
			same := ""
			if base == painting.TextScaler(custom) {
				same = fmt.Sprintf(" same=%t", c == base)
			}
			out = append(out, dart.ToString(base)+" clamp "+tsList(r)+": "+tsDescribe(c)+same)
			for _, r2 := range ranges {
				c2 := c.Clamp(tsClampOpts(r2))
				out = append(out, dart.ToString(base)+" clamp "+tsList(r)+" "+tsList(r2)+": "+tsDescribe(c2))
			}
		}
	}
	for i := range built {
		var b strings.Builder
		for _, x := range built {
			if x == built[i] {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		out = append(out, fmt.Sprintf("eq %d: %s", i, b.String()))
	}
	for _, bad := range [][]float64{{5, 4}, {-1, 2}, {inf, inf}, {1, nan}, {nan, 2}, {math.Copysign(0, -1), 0}, {0, math.Copysign(0, -1)}} {
		one, five := 1.0, 5.0
		out = append(out, "assert "+tsList(bad)+": "+
			tsErr(func() { painting.TextScalerNoScaling.Clamp(tsClampOpts(bad)) })+" "+
			tsErr(func() { custom.Clamp(tsClampOpts(bad)) })+" "+
			tsErr(func() {
				custom.Clamp(painting.TextScalerClampOptions{MinScaleFactor: &one, MaxScaleFactor: &five}).Clamp(tsClampOpts(bad))
			}))
	}
	one, two := 1.0, 2.0
	cl := func() painting.TextScaler {
		return custom.Clamp(painting.TextScalerClampOptions{MinScaleFactor: &one, MaxScaleFactor: &two})
	}
	out = append(out,
		"linear(-1): "+tsErr(func() { painting.NewTextScalerLinear(-1) }),
		"scale(-1): "+tsErr(func() { painting.TextScalerNoScaling.Scale(-1) }),
		"scale(inf): "+tsErr(func() { painting.TextScalerNoScaling.Scale(inf) }),
		"scale(nan): "+tsErr(func() { painting.TextScalerNoScaling.Scale(nan) }),
		"clamped scale(-1): "+tsErr(func() { cl().Scale(-1) }),
		"clamped scale(nan): "+tsErr(func() { cl().Scale(nan) }))
	var ov, wb []string
	for _, o := range painting.TextOverflowValues {
		ov = append(ov, o.String())
	}
	for _, b := range painting.TextWidthBasisValues {
		wb = append(wb, b.String())
	}
	out = append(out, "enums ["+strings.Join(ov, ", ")+"] ["+strings.Join(wb, ", ")+"] "+
		dart.DoubleToString(painting.KDefaultFontSize))
	return out
}

func TestPaintingTextScalerMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_text_scaler.json")
	if err != nil {
		t.Fatalf("no reference (%v); run the Flutter side: flutter test test/painting_text_scaler_test.dart", err)
	}
	var want []string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	got := paintingTextScalerLines()
	if len(got) != len(want) {
		t.Errorf("%d lines, Flutter has %d", len(got), len(want))
	}
	bad := 0
	for i := 0; i < len(want) && i < len(got); i++ {
		if got[i] != want[i] {
			if bad++; bad <= 20 {
				t.Errorf("line %d:\n flutter %s\n go      %s", i, want[i], got[i])
			}
		}
	}
	t.Logf("%d lines", len(want))
}
