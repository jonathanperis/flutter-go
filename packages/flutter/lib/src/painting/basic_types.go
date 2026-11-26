package painting

// Port of basic_types.dart (Flutter 3.50.1): RenderComparison, Axis,
// VerticalDirection, AxisDirection and their helpers. basic_types.dart's
// re-exports of dart:ui (Color, Offset, TextDirection...) and foundation's
// VoidCallback are not repeated: import ui and foundation (API_MAPPING,
// "Libraries").

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// RenderComparison is the description of the difference between two
// objects, in the context of how it will affect the rendering. Values are
// ordered so that a greater value means a bigger difference.
type RenderComparison int

const (
	// RenderComparisonIdentical: no difference.
	RenderComparisonIdentical RenderComparison = iota
	// RenderComparisonMetadata: the two objects differ only in metadata
	// that does not affect rendering.
	RenderComparisonMetadata
	// RenderComparisonPaint: the difference requires repainting only.
	RenderComparisonPaint
	// RenderComparisonLayout: the difference may change the layout.
	RenderComparisonLayout
)

// RenderComparisonValues is RenderComparison.values.
var RenderComparisonValues = []RenderComparison{RenderComparisonIdentical,
	RenderComparisonMetadata, RenderComparisonPaint, RenderComparisonLayout}

// Name is the Dart enum value's name.
func (c RenderComparison) Name() string {
	return [...]string{"identical", "metadata", "paint", "layout"}[c]
}

func (c RenderComparison) String() string { return "RenderComparison." + c.Name() }

// Axis is one of the two cardinal directions in two dimensions.
type Axis int

const (
	// AxisHorizontal: left and right.
	AxisHorizontal Axis = iota
	// AxisVertical: up and down.
	AxisVertical
)

// AxisValues is Axis.values.
var AxisValues = []Axis{AxisHorizontal, AxisVertical}

// Name is the Dart enum value's name.
func (a Axis) Name() string { return [...]string{"horizontal", "vertical"}[a] }

func (a Axis) String() string { return "Axis." + a.Name() }

// FlipAxis returns the axis that is perpendicular to the given axis.
func FlipAxis(direction Axis) Axis {
	switch direction {
	case AxisHorizontal:
		return AxisVertical
	default:
		return AxisHorizontal
	}
}

// VerticalDirection is a direction in which boxes flow vertically.
type VerticalDirection int

const (
	// VerticalDirectionUp: boxes start at the bottom and are stacked up.
	VerticalDirectionUp VerticalDirection = iota
	// VerticalDirectionDown: boxes start at the top and are stacked down.
	VerticalDirectionDown
)

// VerticalDirectionValues is VerticalDirection.values.
var VerticalDirectionValues = []VerticalDirection{VerticalDirectionUp, VerticalDirectionDown}

// Name is the Dart enum value's name.
func (d VerticalDirection) Name() string { return [...]string{"up", "down"}[d] }

func (d VerticalDirection) String() string { return "VerticalDirection." + d.Name() }

// AxisDirection is a direction along either the horizontal or vertical
// Axis in which the origin, or zero position, is determined.
type AxisDirection int

const (
	// AxisDirectionUp: zero is at the bottom and positive values are above it.
	AxisDirectionUp AxisDirection = iota
	// AxisDirectionRight: zero is on the left and positive values are to its right.
	AxisDirectionRight
	// AxisDirectionDown: zero is at the top and positive values are below it.
	AxisDirectionDown
	// AxisDirectionLeft: zero is to the right and positive values are to its left.
	AxisDirectionLeft
)

// AxisDirectionValues is AxisDirection.values.
var AxisDirectionValues = []AxisDirection{AxisDirectionUp, AxisDirectionRight,
	AxisDirectionDown, AxisDirectionLeft}

// Name is the Dart enum value's name.
func (d AxisDirection) Name() string { return [...]string{"up", "right", "down", "left"}[d] }

func (d AxisDirection) String() string { return "AxisDirection." + d.Name() }

// AxisDirectionToAxis returns the Axis that contains the given
// AxisDirection.
func AxisDirectionToAxis(axisDirection AxisDirection) Axis {
	switch axisDirection {
	case AxisDirectionUp, AxisDirectionDown:
		return AxisVertical
	default:
		return AxisHorizontal
	}
}

// TextDirectionToAxisDirection returns the AxisDirection in which reading
// occurs in the given TextDirection.
func TextDirectionToAxisDirection(textDirection ui.TextDirection) AxisDirection {
	switch textDirection {
	case ui.TextDirectionRtl:
		return AxisDirectionLeft
	default:
		return AxisDirectionRight
	}
}

// FlipAxisDirection returns the opposite of the given AxisDirection.
func FlipAxisDirection(axisDirection AxisDirection) AxisDirection {
	switch axisDirection {
	case AxisDirectionUp:
		return AxisDirectionDown
	case AxisDirectionRight:
		return AxisDirectionLeft
	case AxisDirectionDown:
		return AxisDirectionUp
	default:
		return AxisDirectionRight
	}
}

// AxisDirectionIsReversed reports whether traveling along the given axis
// direction visits coordinates along that axis in numerically decreasing
// order.
func AxisDirectionIsReversed(axisDirection AxisDirection) bool {
	switch axisDirection {
	case AxisDirectionUp, AxisDirectionLeft:
		return true
	default:
		return false
	}
}
