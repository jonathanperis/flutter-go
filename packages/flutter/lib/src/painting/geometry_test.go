package painting

// Port of packages/flutter/test/painting/geometry_test.dart (Flutter
// 3.50.1). Every test is ported (tooltip_test.dart's further cases need
// material).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func TestPositionDependentBox(t *testing.T) {
	m := func(v float64) PositionDependentBoxOptions { return PositionDependentBoxOptions{Margin: &v} }
	cases := []struct {
		childW, targetX, margin float64
		want                    ui.Offset
	}{
		{20, 50, 0, ui.NewOffset(40, 40)},
		{200, 50, 0, ui.NewOffset(-50, 40)},
		{200, 0, 0, ui.NewOffset(-50, 40)},
		{200, 100, 0, ui.NewOffset(-50, 40)},
		{50, 50, 20, ui.NewOffset(25, 40)}, // 60.0 left
		{50, 50, 30, ui.NewOffset(25, 40)}, // 40.0 left
		{50, 0, 20, ui.NewOffset(20, 40)},
		{50, 0, 30, ui.NewOffset(25, 40)},
		{50, 100, 20, ui.NewOffset(30, 40)},
		{50, 100, 30, ui.NewOffset(25, 40)},
	}
	for _, c := range cases {
		got := PositionDependentBox(ui.NewSize(100, 100), ui.NewSize(c.childW, 10), ui.NewOffset(c.targetX, 50), false, m(c.margin))
		if got != c.want {
			t.Errorf("%+v: got %v", c, got)
		}
	}
}
