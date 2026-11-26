package painting

// Port of packages/flutter/test/painting/box_fit_test.dart (Flutter
// 3.50.1). Every test is ported.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func expectFitted(t *testing.T, r FittedSizes, source, destination ui.Size) {
	t.Helper()
	if r.Source != source || r.Destination != destination {
		t.Errorf("got %v -> %v, want %v -> %v", r.Source, r.Destination, source, destination)
	}
}

func TestApplyBoxFit(t *testing.T) {
	sz := ui.NewSize
	expectFitted(t, ApplyBoxFit(BoxFitScaleDown, sz(100, 1000), sz(200, 2000)), sz(100, 1000), sz(100, 1000))
	expectFitted(t, ApplyBoxFit(BoxFitScaleDown, sz(300, 3000), sz(200, 2000)), sz(300, 3000), sz(200, 2000))
	expectFitted(t, ApplyBoxFit(BoxFitFitWidth, sz(2000, 400), sz(1000, 100)), sz(2000, 200), sz(1000, 100))
	expectFitted(t, ApplyBoxFit(BoxFitFitWidth, sz(2000, 400), sz(1000, 300)), sz(2000, 400), sz(1000, 200))
	expectFitted(t, ApplyBoxFit(BoxFitFitHeight, sz(400, 2000), sz(100, 1000)), sz(200, 2000), sz(100, 1000))
	expectFitted(t, ApplyBoxFit(BoxFitFitHeight, sz(400, 2000), sz(300, 1000)), sz(400, 2000), sz(200, 1000))
	for _, fit := range BoxFitValues {
		testZeroAndNegativeSizes(t, fit)
	}
	if BoxFitScaleDown.String() != "BoxFit.scaleDown" || BoxFitFitWidth.String() != "BoxFit.fitWidth" {
		t.Error("BoxFit names")
	}
}

func testZeroAndNegativeSizes(t *testing.T, fit BoxFit) {
	t.Helper()
	sz := ui.NewSize
	for _, c := range [][2]ui.Size{
		{sz(-400, 2000), sz(100, 1000)},
		{sz(400, -2000), sz(100, 1000)},
		{sz(400, 2000), sz(-100, 1000)},
		{sz(400, 2000), sz(100, -1000)},
		{sz(0, 2000), sz(100, 1000)},
		{sz(400, 0), sz(100, 1000)},
		{sz(400, 2000), sz(0, 1000)},
		{sz(400, 2000), sz(100, 0)},
	} {
		expectFitted(t, ApplyBoxFit(fit, c[0], c[1]), ui.SizeZero, ui.SizeZero)
	}
}
