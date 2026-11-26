package painting

// basic_types.dart has no test file of its own in Flutter
// (packages/flutter/test/painting); these cover its functions and enum
// names, which the painting difftest also compares with Flutter.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func TestBasicTypes(t *testing.T) {
	if FlipAxis(AxisHorizontal) != AxisVertical || FlipAxis(AxisVertical) != AxisHorizontal {
		t.Error("flipAxis")
	}
	toAxis := map[AxisDirection]Axis{AxisDirectionUp: AxisVertical, AxisDirectionDown: AxisVertical,
		AxisDirectionLeft: AxisHorizontal, AxisDirectionRight: AxisHorizontal}
	flip := map[AxisDirection]AxisDirection{AxisDirectionUp: AxisDirectionDown, AxisDirectionDown: AxisDirectionUp,
		AxisDirectionLeft: AxisDirectionRight, AxisDirectionRight: AxisDirectionLeft}
	reversed := map[AxisDirection]bool{AxisDirectionUp: true, AxisDirectionLeft: true}
	for _, d := range AxisDirectionValues {
		if AxisDirectionToAxis(d) != toAxis[d] || FlipAxisDirection(d) != flip[d] || AxisDirectionIsReversed(d) != reversed[d] {
			t.Errorf("%v", d)
		}
	}
	if TextDirectionToAxisDirection(ui.TextDirectionRtl) != AxisDirectionLeft ||
		TextDirectionToAxisDirection(ui.TextDirectionLtr) != AxisDirectionRight {
		t.Error("textDirectionToAxisDirection")
	}
	for _, c := range []struct{ got, want string }{
		{AxisHorizontal.String(), "Axis.horizontal"}, {AxisVertical.String(), "Axis.vertical"},
		{VerticalDirectionUp.String(), "VerticalDirection.up"}, {VerticalDirectionDown.String(), "VerticalDirection.down"},
		{AxisDirectionLeft.String(), "AxisDirection.left"}, {AxisDirectionRight.String(), "AxisDirection.right"},
		{AxisDirectionUp.String(), "AxisDirection.up"}, {AxisDirectionDown.String(), "AxisDirection.down"},
		{RenderComparisonIdentical.String(), "RenderComparison.identical"},
		{RenderComparisonMetadata.String(), "RenderComparison.metadata"},
		{RenderComparisonPaint.String(), "RenderComparison.paint"},
		{RenderComparisonLayout.String(), "RenderComparison.layout"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	if !(RenderComparisonIdentical < RenderComparisonMetadata && RenderComparisonMetadata < RenderComparisonPaint &&
		RenderComparisonPaint < RenderComparisonLayout) {
		t.Error("RenderComparison order")
	}
}
