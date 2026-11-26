package painting_test

// Helpers shared by the border tests (ports of Flutter's border tests,
// in the external test package so they can use flutter_test).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// isUnitCircle is common_matchers.dart's isUnitCircle (test/painting).
var isUnitCircle = ft.IsPathThat(ft.IsPathThatOptions{
	Includes: bdOffsets(
		-0.6035617555492896, 0.2230970398703236,
		-0.7738478165627277, 0.5640447581420576,
		-0.46090034164788385, -0.692017006684612,
		-0.2138540316101296, -0.09997005339529785,
		-0.46919827227410416, 0.29581721423767027,
		-0.43628713652733153, 0.5065324817995975,
		0, 0,
		0.49296904381712725, -0.5922438805080081,
		0.2901141594861445, -0.3181478162967859,
		0.45229946324502146, 0.4324593232323706,
		0.11827752132593572, 0.806442226027837,
		0.8854165569581154, -0.08604230149167624,
	),
	Excludes: bdOffsets(
		-100.0, -100.0,
		-100.0, 100.0,
		-1.1104403014186688, -1.1234939207590569,
		-1.1852827482514838, -0.5029551986333607,
		-1.0253256532179804, -0.02034402043932526,
		-1.4488532714237397, 0.4948740308904742,
		-1.03142206223176, 0.81070400258819,
		-1.006747917852356, 1.3712062218039343,
		-0.5241429900291878, -1.2852518410112541,
		-0.8879593765104428, -0.9999680025850874,
		-0.9120835110799488, -0.4361605900585557,
		-0.8184877240407303, 1.1202520775469589,
		-0.15746058420492282, -1.1905035795387513,
		-0.11519948876183506, 1.3848147258237393,
		0.0035741796943844495, -1.3383908620447724,
		0.34408827443814394, 1.4514436242950461,
		0.709487222145941, -1.3468012918181573,
		0.6287522653614315, -0.8315879623940617,
		0.9716071801865485, 0.24311969613525442,
		0.7632982576031955, 0.8329765574976169,
		0.9923766847309081, 1.0592617071813715,
		1.2696730082820435, -1.0353385446957046,
		1.4266154921521208, -0.8382633931857755,
		1.298035226938996, -0.11544603567954526,
		1.4143230992455558, 0.10842501221141165,
		1.465352952354424, 0.6999947490821032,
		1.0462985816010146, 1.3874230508561505,
		100.0, -100.0,
		100.0, 100.0,
	),
})

// bdPath is isPathThat(includes:, excludes:).
func bdPath(includes, excludes []ui.Offset) ft.Matcher {
	return ft.IsPathThat(ft.IsPathThatOptions{Includes: includes, Excludes: excludes})
}

// bdBounds compares a path's bounds.
func bdBounds(t *testing.T, name string, p *ui.Path, want ui.Rect) {
	t.Helper()
	if got := p.GetBounds(); got != want {
		t.Errorf("%s: bounds %v, want %v", name, got, want)
	}
}

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
