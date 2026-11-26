package services

// Port of debug.dart (Flutter 3.50.1).

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// DebugKeyEventSimulatorTransitModeOverride overrides the transit mode
// flutter_test's key event simulator uses (nil: its default; deprecated).
var DebugKeyEventSimulatorTransitModeOverride *KeyDataTransitMode

// DebugPrintKeyboardEvents prints keyboard events in debug builds (read by
// HardwareKeyboard).
var DebugPrintKeyboardEvents = false

// DebugAssertAllServicesVarsUnset panics with a FlutterError carrying
// reason if a services debug variable has been changed (debug builds); it
// returns true. (DebugProfilePlatformChannels is not checked, as in
// Flutter.)
func DebugAssertAllServicesVarsUnset(reason string) bool {
	if foundation.KDebugMode {
		if DebugKeyEventSimulatorTransitModeOverride != nil {
			panic(foundation.NewFlutterError(reason))
		}
		if DebugPrintKeyboardEvents {
			panic(foundation.NewFlutterError(reason))
		}
	}
	return true
}

// DebugProfilePlatformChannels profiles where platform channel messages
// go (see ShouldProfilePlatformChannels); it is the
// ext.flutter.profilePlatformChannels service extension's value.
var DebugProfilePlatformChannels = false
