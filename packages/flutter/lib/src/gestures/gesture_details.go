package gestures

// Port of gesture_details.dart (Flutter 3.50.1).

import ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"

// PositionedGestureDetails is the interface of the details of gestures
// with a position.
type PositionedGestureDetails interface {
	// GlobalPosition is the global position at which the pointer
	// interacted with the screen.
	GlobalPosition() ui.Offset
	// LocalPosition is that position in the local coordinate system of the
	// event receiver.
	LocalPosition() ui.Offset
}
