package services

// Port of system_sound.dart (Flutter 3.50.1).

import "github.com/jonathanperis/flutter-go/src/async"

// SystemSoundType is a sound provided by the system.
type SystemSoundType int

const (
	// SystemSoundTypeClick is a short indication that a button was pressed.
	SystemSoundTypeClick SystemSoundType = iota
	// SystemSoundTypeTick is a short system click, such as when a picker
	// scrolls (iOS only).
	SystemSoundTypeTick
	// SystemSoundTypeAlert is the system alert sound (desktop platforms).
	SystemSoundTypeAlert
)

// SystemSoundTypeValues is SystemSoundType.values.
var SystemSoundTypeValues = []SystemSoundType{SystemSoundTypeClick, SystemSoundTypeTick, SystemSoundTypeAlert}

// Name is the enum value's Dart name.
func (t SystemSoundType) Name() string { return [...]string{"click", "tick", "alert"}[t] }

func (t SystemSoundType) String() string { return "SystemSoundType." + t.Name() }

type systemSound struct{}

// SystemSound provides access to the library of short system specific
// sounds for common tasks (a class of static methods).
var SystemSound systemSound

// Play plays the specified system sound. If that sound is not present on
// the system, the call is ignored.
func (systemSound) Play(typ SystemSoundType) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("SystemSound.play", typ.String()).Await()
	})
}
