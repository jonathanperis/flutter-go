package ui

// Port of engine/src/flutter/lib/ui/natives.dart and platform_isolate.dart
// (Flutter 3.50.1): DartPluginRegistrant, runOnPlatformThread and
// isRunningOnPlatformThread. (natives.dart's print/microtask hooks and
// service extensions belong to the Dart VM's embedding; Go's are
// dart.Print and async.Loop.)

import (
	"sync"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// dartPluginRegistrant is DartPluginRegistrant, a class of static members.
type dartPluginRegistrant struct {
	once sync.Once
}

// DartPluginRegistrant initializes the app's Dart plugins
// (DartPluginRegistrant.EnsureInitialized()).
var DartPluginRegistrant = &dartPluginRegistrant{}

// EnginePluginRegistrant is the engine's plugin registration
// (DartPluginRegistrant_EnsureInitialized: the generated
// dart_plugin_registrant's register()); nil registers nothing. The
// embedder sets it.
var EnginePluginRegistrant func()

// EnsureInitialized registers the app's Dart plugins, once
// (EnginePluginRegistrant).
func (r *dartPluginRegistrant) EnsureInitialized() {
	r.once.Do(func() {
		if EnginePluginRegistrant != nil {
			EnginePluginRegistrant()
		}
	})
}

// DartPluginRegistrantEnsureInitialized is
// DartPluginRegistrant.EnsureInitialized.
func DartPluginRegistrantEnsureInitialized() { DartPluginRegistrant.EnsureInitialized() }

// IsRunningOnPlatformThread is whether the current isolate runs on the
// platform thread. flutter-go's UI runs on the platform thread (the main
// thread, as flutter_tester's merged platform and UI threads), so it is
// true.
var IsRunningOnPlatformThread = true

// EnginePlatformIsolatesEnabled is _platformIsolatesEnabled: enabled, as in
// flutter_tester.
const EnginePlatformIsolatesEnabled = true

// RunOnPlatformThread runs computation on the platform thread and returns
// its result. The UI already runs there, so (as Dart does when
// isRunningOnPlatformThread) it is Future(computation): computation runs
// in a timer of the current loop, and a panic becomes the future's error.
func RunOnPlatformThread[R any](computation func() R) async.Future[R] {
	if !EnginePlatformIsolatesEnabled {
		panic(dart.UnsupportedError{Message: "Platform thread isolates are not supported by this platform."})
	}
	c := async.NewSyncCompleter[R]()
	async.TimerRun(func() {
		failed := true
		defer func() {
			if failed {
				r := recover()
				err, ok := r.(error)
				if !ok {
					err = async.PanicError{Value: r}
				}
				c.CompleteError(err)
			}
		}()
		result := computation()
		failed = false
		c.Complete(result)
	})
	return c.Future()
}
