package gestures

// Port of drag.dart (Flutter 3.50.1).

// Drag is the interface for objects that receive updates about drags
// (an abstract class with empty default methods in Dart; embed DragBase
// for the defaults).
type Drag interface {
	// Update is called when the pointer has moved.
	Update(details *DragUpdateDetails)
	// End is called when the pointer is no longer in contact with the
	// screen.
	End(details *DragEndDetails)
	// Cancel is called when the input from the pointer that was
	// generating this drag update is no longer available.
	Cancel()
}

// DragBase is Drag's default (empty) methods, to embed.
type DragBase struct{}

func (DragBase) Update(details *DragUpdateDetails) {}
func (DragBase) End(details *DragEndDetails)       {}
func (DragBase) Cancel()                           {}
