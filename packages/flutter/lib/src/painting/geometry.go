package painting

// Port of geometry.dart (Flutter 3.50.1).

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// PositionDependentBoxOptions are positionDependentBox's optional named
// parameters (nil: Dart's defaults, verticalOffset 0.0 and margin 10.0).
type PositionDependentBoxOptions struct {
	VerticalOffset *float64
	Margin         *float64
}

// PositionDependentBox positions a child box within a container box,
// either above or below a target point (tooltips). Dart's required named
// parameters size, childSize, target and preferBelow are positional.
func PositionDependentBox(size, childSize ui.Size, target ui.Offset, preferBelow bool, opts ...PositionDependentBoxOptions) ui.Offset {
	verticalOffset, margin := 0.0, 10.0
	if len(opts) > 0 {
		if opts[0].VerticalOffset != nil {
			verticalOffset = *opts[0].VerticalOffset
		}
		if opts[0].Margin != nil {
			margin = *opts[0].Margin
		}
	}
	// VERTICAL DIRECTION
	fitsBelow := target.Dy+verticalOffset+childSize.Height <= size.Height-margin
	fitsAbove := target.Dy-verticalOffset-childSize.Height >= margin
	tooltipBelow := fitsBelow
	if fitsAbove == fitsBelow {
		tooltipBelow = preferBelow
	}
	var y float64
	if tooltipBelow {
		y = math.Min(target.Dy+verticalOffset, size.Height-margin)
	} else {
		y = math.Max(target.Dy-verticalOffset-childSize.Height, margin)
	}
	// HORIZONTAL DIRECTION
	flexibleSpace := size.Width - childSize.Width
	var x float64
	if flexibleSpace <= 2*margin {
		x = flexibleSpace / 2.0
	} else {
		x = ui.ClampDouble(target.Dx-childSize.Width/2, margin, flexibleSpace-margin)
	}
	return ui.NewOffset(x, y)
}
