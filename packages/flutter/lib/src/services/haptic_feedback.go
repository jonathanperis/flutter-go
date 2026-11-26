package services

// Port of haptic_feedback.dart (Flutter 3.50.1).

import "github.com/jonathanperis/flutter-go/src/async"

type hapticFeedback struct{}

// HapticFeedback allows access to the haptic feedback interface on the
// device (a class of static methods: HapticFeedback.Vibrate()...).
var HapticFeedback hapticFeedback

func hapticVibrate(arguments ...any) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("HapticFeedback.vibrate", arguments...).Await()
	})
}

// Vibrate provides vibration haptic feedback to the user for a short
// duration.
func (hapticFeedback) Vibrate() async.Future[async.Void] { return hapticVibrate() }

// LightImpact provides a haptic feedback corresponding a collision
// impact with a light mass.
func (hapticFeedback) LightImpact() async.Future[async.Void] {
	return hapticVibrate("HapticFeedbackType.lightImpact")
}

// MediumImpact provides a haptic feedback corresponding a collision
// impact with a medium mass.
func (hapticFeedback) MediumImpact() async.Future[async.Void] {
	return hapticVibrate("HapticFeedbackType.mediumImpact")
}

// HeavyImpact provides a haptic feedback corresponding a collision
// impact with a heavy mass.
func (hapticFeedback) HeavyImpact() async.Future[async.Void] {
	return hapticVibrate("HapticFeedbackType.heavyImpact")
}

// SelectionClick provides a haptic feedback indication selection
// changing through discrete values.
func (hapticFeedback) SelectionClick() async.Future[async.Void] {
	return hapticVibrate("HapticFeedbackType.selectionClick")
}

// SuccessNotification provides a haptic feedback indicating that a task
// or action has completed successfully.
func (hapticFeedback) SuccessNotification() async.Future[async.Void] {
	return hapticVibrate("HapticFeedbackType.successNotification")
}

// WarningNotification provides a haptic feedback indicating that a task
// or action has produced a warning.
func (hapticFeedback) WarningNotification() async.Future[async.Void] {
	return hapticVibrate("HapticFeedbackType.warningNotification")
}

// ErrorNotification provides a haptic feedback indicating that a task or
// action has failed.
func (hapticFeedback) ErrorNotification() async.Future[async.Void] {
	return hapticVibrate("HapticFeedbackType.errorNotification")
}
