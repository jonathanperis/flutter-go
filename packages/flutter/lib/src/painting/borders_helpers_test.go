package painting_test

// Helpers shared by the border tests (ports of Flutter's border tests,
// in the external test package so they can use flutter_test).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Pointer helpers for options (Dart's named arguments).
func bdF(v float64) *float64             { return &v }
func bdCol(v int) *ui.Color              { c := ui.NewColor(v); return &c }
func bdStyle(s BorderStyle) *BorderStyle { return &s }
func bdSide(s BorderSide) *BorderSide    { return &s }

// bs is BorderSide(...) with Dart's named arguments.
func bdBS(o BorderSideOptions) BorderSide { return NewBorderSide(o) }

func bdExpectAssertion(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatal("expected an assertion error")
		}
		if _, ok := r.(dart.AssertionFailure); !ok {
			t.Fatalf("got %T %v, want an AssertionError", r, r)
		}
	}()
	f()
}

func bdTD(d ui.TextDirection) *ui.TextDirection { return &d }

var (
	bdLtr = bdTD(ui.TextDirectionLtr)
	bdRtl = bdTD(ui.TextDirectionRtl)
)

func bdRadius(r BorderRadius) *BorderRadius { return &r }

func bdOffset(x, y float64) ui.Offset { return ui.NewOffset(x, y) }

func bdOffsets(xy ...float64) []ui.Offset {
	var out []ui.Offset
	for i := 0; i+1 < len(xy); i += 2 {
		out = append(out, ui.NewOffset(xy[i], xy[i+1]))
	}
	return out
}

// bdPanics runs f and returns the recovered value (nil: no panic).
func bdPanics(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return nil
}
