package ui

// Partial port of engine/src/flutter/lib/ui/window.dart (Flutter 3.50.1):
// Brightness, and the deprecated SingletonFlutterWindow / window with the
// members that delegate to what PlatformDispatcher has so far.

// Brightness describes the contrast of a theme or color palette.
type Brightness int

const (
	// BrightnessDark: the color is dark and needs a light text color.
	BrightnessDark Brightness = iota
	// BrightnessLight: the color is light and needs a dark text color.
	BrightnessLight
)

func (b Brightness) String() string {
	return [...]string{"Brightness.dark", "Brightness.light"}[b]
}

// SingletonFlutterWindow is the deprecated FlutterView of a single-view
// application (Deprecated: use FlutterView or PlatformDispatcher).
type SingletonFlutterWindow struct {
	platformDispatcher *PlatformDispatcher
}

// PlatformDispatcher is the dispatcher this window belongs to.
func (w *SingletonFlutterWindow) PlatformDispatcher() *PlatformDispatcher {
	return w.platformDispatcher
}

// PlatformBrightness is platformDispatcher.platformBrightness.
func (w *SingletonFlutterWindow) PlatformBrightness() Brightness {
	return w.platformDispatcher.PlatformBrightness()
}

// Window is dart:ui's top-level window (Deprecated: look up the FlutterView
// or use PlatformDispatcher.instance).
var Window = &SingletonFlutterWindow{platformDispatcher: PlatformDispatcherInstance()}
