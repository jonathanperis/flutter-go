package gestures

// Port of debug.dart (Flutter 3.50.1).

import "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"

// Any changes to this file should be reflected in the
// DebugAssertAllGesturesVarsUnset function below.

// DebugPrintHitTestResults prints the results of each hit test to the
// console.
var DebugPrintHitTestResults = false

// DebugPrintMouseHoverEvents prints the mouse hover events to the console.
var DebugPrintMouseHoverEvents = false

// DebugPrintGestureArenaDiagnostics prints the gesture arena's diagnostics
// (members added, accepted, rejected, resolved) to the console.
var DebugPrintGestureArenaDiagnostics = false

// DebugPrintRecognizerCallbacksTrace logs a message each time a recognizer
// calls a callback.
var DebugPrintRecognizerCallbacksTrace = false

// DebugPrintResamplingMargin prints the resampling margin to the console.
var DebugPrintResamplingMargin = false

// DebugAssertAllGesturesVarsUnset returns true if none of the gestures
// library debug variables have been changed (DebugPrintMouseHoverEvents is
// not checked, as in Flutter); it panics with a FlutterError carrying
// reason otherwise. Used in tests, in an assert.
func DebugAssertAllGesturesVarsUnset(reason string) bool {
	if foundation.KDebugMode && (DebugPrintHitTestResults ||
		DebugPrintGestureArenaDiagnostics ||
		DebugPrintRecognizerCallbacksTrace ||
		DebugPrintResamplingMargin) {
		panic(foundation.NewFlutterError(reason))
	}
	return true
}
