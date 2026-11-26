package gestures

// Compile-time checks that the recognizers implement their Dart
// superclasses' interfaces.
var (
	_ BaseTapGestureRecognizer        = (*TapGestureRecognizer)(nil)
	_ GestureRecognizer               = (*DoubleTapGestureRecognizer)(nil)
	_ GestureRecognizer               = (*MultiTapGestureRecognizer)(nil)
	_ GestureRecognizer               = (*SerialTapGestureRecognizer)(nil)
	_ PrimaryPointerGestureRecognizer = (*LongPressGestureRecognizer)(nil)
	_ OneSequenceGestureRecognizer    = (*ForcePressGestureRecognizer)(nil)
	_ OneSequenceGestureRecognizer    = (*EagerGestureRecognizer)(nil)
)
