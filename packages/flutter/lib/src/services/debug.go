package services

// Partial port of debug.dart (Flutter 3.50.1): the deprecated
// debugKeyEventSimulatorTransitModeOverride (a KeyDataTransitMode, from
// hardware_keyboard.dart) comes with the keyboard port.

import (
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// DebugPrintKeyboardEvents prints keyboard events in debug builds (read by
// HardwareKeyboard and KeyEventManager, not ported yet).
var DebugPrintKeyboardEvents = false

// DebugAssertAllServicesVarsUnset panics with a FlutterError carrying
// reason if a services debug variable has been changed (debug builds); it
// returns true. (DebugProfilePlatformChannels is not checked, as in
// Flutter.)
func DebugAssertAllServicesVarsUnset(reason string) bool {
	if foundation.KDebugMode {
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
