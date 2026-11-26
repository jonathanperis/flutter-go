package ui

// Partial port of engine/src/flutter/lib/ui/platform_dispatcher.dart
// (Flutter 3.50.1): the PlatformDispatcher singleton and the part of its
// platform configuration that foundation reads (platformBrightness). The
// engine-facing parts (frame callbacks, views, messages, locales,
// accessibility...) come with the scheduler and services ports.

// PlatformDispatcher is the most basic interface to the host operating
// system's interface.
type PlatformDispatcher struct {
	configuration platformConfiguration
}

// platformConfiguration is _PlatformConfiguration (the fields ported so far)
// with its defaults.
type platformConfiguration struct {
	platformBrightness Brightness
}

var platformDispatcherInstance = &PlatformDispatcher{
	configuration: platformConfiguration{platformBrightness: BrightnessLight},
}

// PlatformDispatcherInstance is PlatformDispatcher.instance, the singleton.
func PlatformDispatcherInstance() *PlatformDispatcher { return platformDispatcherInstance }

// PlatformBrightness is the setting indicating the current brightness mode
// of the host platform (Brightness.light when it has no preference).
func (p *PlatformDispatcher) PlatformBrightness() Brightness {
	return p.configuration.platformBrightness
}
