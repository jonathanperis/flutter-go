package scheduler

// Port of debug.dart (Flutter 3.50.1).

import "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"

// Any changes to this file should be reflected in the
// DebugAssertAllSchedulerVarsUnset function below.

// DebugPrintBeginFrameBanner prints a banner at the beginning of each frame.
//
// Frames triggered by the engine and handled by the scheduler binding have a
// banner giving the frame number and the time stamp of the frame. Frames
// triggered eagerly by the framework (warm-up frames) say "warm-up frame"
// instead of the time stamp.
var DebugPrintBeginFrameBanner = false

// DebugPrintEndFrameBanner prints a banner at the end of each frame.
var DebugPrintEndFrameBanner = false

// DebugPrintScheduleFrameStacks logs the call stacks that cause a frame to
// be scheduled (SchedulerBinding.ScheduleFrame).
var DebugPrintScheduleFrameStacks = false

// DebugTracePostFrameCallbacks records timeline events for post-frame
// callbacks (labelled with AddPostFrameCallback's debugLabel).
var DebugTracePostFrameCallbacks = false

// DebugAssertAllSchedulerVarsUnset returns true if none of the scheduler
// library debug variables have been changed (only the frame banners are
// checked, as in Flutter); it panics with a FlutterError carrying reason
// otherwise. Used in tests, in an assert.
func DebugAssertAllSchedulerVarsUnset(reason string) bool {
	if foundation.KDebugMode && (DebugPrintBeginFrameBanner || DebugPrintEndFrameBanner) {
		panic(foundation.NewFlutterError(reason))
	}
	return true
}
