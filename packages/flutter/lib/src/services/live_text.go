package services

// Port of packages/flutter/lib/src/services/live_text.dart (Flutter
// 3.50.1).

import "github.com/jonathanperis/flutter-go/src/async"

// liveText is the type of LiveText (Dart's abstract final class of static
// methods).
type liveText struct{}

// LiveText is a static class for live text (OCR) input on iOS
// (LiveText.isLiveTextInputAvailable() is
// LiveText.IsLiveTextInputAvailable()...).
var LiveText liveText

// IsLiveTextInputAvailable returns whether live text input is available on
// the device (false when the platform returns null).
func (liveText) IsLiveTextInputAvailable() async.Future[bool] {
	return async.Async(func() bool {
		result := SystemChannels.Platform.InvokeMethod("LiveText.isLiveTextInputAvailable").Await()
		if result == nil {
			return false
		}
		return result.(bool)
	})
}

// StartLiveTextInput starts live text input.
func (liveText) StartLiveTextInput() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.TextInput.InvokeMethod("TextInput.startLiveTextInput").Await()
	})
}
