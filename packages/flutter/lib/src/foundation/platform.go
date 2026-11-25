package foundation

import (
	"os"
	"runtime"
)

// TargetPlatform is the platform Flutter adapts its behavior to.
type TargetPlatform int

const (
	TargetPlatformAndroid TargetPlatform = iota
	TargetPlatformFuchsia
	TargetPlatformIOS
	TargetPlatformLinux
	TargetPlatformMacOS
	TargetPlatformWindows
)

func (p TargetPlatform) String() string {
	return [...]string{"TargetPlatform.android", "TargetPlatform.fuchsia", "TargetPlatform.iOS",
		"TargetPlatform.linux", "TargetPlatform.macOS", "TargetPlatform.windows"}[p]
}

var debugDefaultTargetPlatformOverride *TargetPlatform

// DebugDefaultTargetPlatformOverride is the override getter; nil means none.
func DebugDefaultTargetPlatformOverride() *TargetPlatform { return debugDefaultTargetPlatformOverride }

// SetDebugDefaultTargetPlatformOverride sets the override (nil clears it).
// Like Flutter, it is an error outside debug builds.
func SetDebugDefaultTargetPlatformOverride(value *TargetPlatform) {
	if !KDebugMode {
		panic(NewFlutterError("Cannot modify debugDefaultTargetPlatformOverride in non-debug builds."))
	}
	debugDefaultTargetPlatformOverride = value
}

// DefaultTargetPlatform is the platform the app runs on, as in
// _platform_io.dart: from the OS, Android under `flutter test`
// (FLUTTER_TEST set) in debug mode, and the debug override if set.
func DefaultTargetPlatform() TargetPlatform {
	var result *TargetPlatform
	set := func(p TargetPlatform) { result = &p }
	switch runtime.GOOS {
	case "android":
		set(TargetPlatformAndroid)
	case "ios":
		set(TargetPlatformIOS)
	case "linux":
		set(TargetPlatformLinux)
	case "darwin":
		set(TargetPlatformMacOS)
	case "windows":
		set(TargetPlatformWindows)
	}
	if KDebugMode {
		if _, ok := os.LookupEnv("FLUTTER_TEST"); ok {
			set(TargetPlatformAndroid)
		}
		if debugDefaultTargetPlatformOverride != nil {
			result = debugDefaultTargetPlatformOverride
		}
	}
	if result == nil {
		panic(NewFlutterError("Unknown platform.\n" + runtime.GOOS + " was not recognized as a target platform. " +
			"Consider updating the list of TargetPlatforms to include this platform."))
	}
	return *result
}
