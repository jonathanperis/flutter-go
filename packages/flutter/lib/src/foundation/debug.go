package foundation

// Port of debug.dart (Flutter 3.50.1). The debugPrint family is print.dart
// (print.go).

import (
	"unsafe"

	"github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/async"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

// DebugAssertAllFoundationVarsUnsetOptions are the optional parameters of
// DebugAssertAllFoundationVarsUnset.
type DebugAssertAllFoundationVarsUnsetOptions struct {
	// DebugPrintOverride is the value DebugPrint is expected to have
	// (default DebugPrintThrottled).
	DebugPrintOverride DebugPrintCallback
}

// DebugAssertAllFoundationVarsUnset returns true if none of the foundation
// library debug variables have been changed (DebugPrint is compared with
// the override, by function identity as Dart's ==), and panics with a
// FlutterError carrying reason otherwise. Used by tests to check that
// debug variables are reset after each test.
func DebugAssertAllFoundationVarsUnset(reason string, opts ...DebugAssertAllFoundationVarsUnsetOptions) bool {
	override := DebugPrintCallback(DebugPrintThrottled)
	if len(opts) > 0 && opts[0].DebugPrintOverride != nil {
		override = opts[0].DebugPrintOverride
	}
	if KDebugMode && (funcIdentity(DebugPrint) != funcIdentity(override) ||
		DebugDefaultTargetPlatformOverride() != nil ||
		DebugDoublePrecision != nil ||
		DebugBrightnessOverride != nil) {
		panic(NewFlutterError(reason))
	}
	return true
}

// funcIdentity is the identity of a function value, Dart's == on functions:
// the same top-level function, or the same closure value, is identical.
// (A Go func literal that captures nothing is one value however often it
// is evaluated; a method value is a new closure each time.)
func funcIdentity[F any](f F) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&f)) }

// DebugInstrumentationEnabled enables the logging of DebugInstrumentAction
// (debug builds only).
var DebugInstrumentationEnabled = false

// DebugInstrumentAction runs action and, when DebugInstrumentationEnabled,
// prints how long it took (`Action "description" took 0:00:00.012345`) once
// the future it returns completes.
//
// Like Dart's async function, the result is a new future: when not
// instrumenting, it is completed with action's future (asynchronously
// chained); when instrumenting, action's future is awaited.
func DebugInstrumentAction[T any](description string, action func() async.Future[T]) async.Future[T] {
	instrument := KDebugMode && DebugInstrumentationEnabled // set in an assert: debug only
	if instrument {
		return async.Async(func() T {
			var sw stopwatch
			sw.start()
			defer func() {
				sw.stop()
				DebugPrint(`Action "` + description + `" took ` + dart.DurationToString(sw.elapsed()))
			}()
			return action().Await()
		})
	}
	// `return action();` in an async function before any await: the result
	// future is completed with action's future, asynchronously.
	result := async.NewCompleter[T]()
	var f async.Future[T]
	func() {
		defer func() {
			if r := recover(); r != nil {
				f = nil
				result.CompleteError(panicAsError(r))
			}
		}()
		f = action()
	}()
	if f != nil {
		result.CompleteFuture(f)
	}
	return result.Future()
}

func panicAsError(r any) error {
	if err, ok := r.(error); ok {
		return err
	}
	return async.PanicError{Value: r}
}

// DebugDoublePrecision configures how many significant digits
// DebugFormatDouble prints (toStringAsPrecision); nil prints one decimal
// (toStringAsFixed(1)).
var DebugDoublePrecision *int

// DebugFormatDouble formats a double to have standard formatting: "null",
// the value with DebugDoublePrecision significant digits, or with one
// decimal.
func DebugFormatDouble(value *float64) string {
	if value == nil {
		return "null"
	}
	if DebugDoublePrecision != nil {
		return dart.ToStringAsPrecision(*value, *DebugDoublePrecision)
	}
	return dart.ToStringAsFixed(*value, 1)
}

// DebugBrightnessOverride, when set, overrides the platform brightness
// (debug builds; set by the brightnessOverride service extension).
var DebugBrightnessOverride *ui.Brightness

// ActiveDevToolsServerAddress is the address of the DevTools server that is
// connected to the app, when there is one (nil otherwise).
var ActiveDevToolsServerAddress *string

// ConnectedVmServiceUri is the URI of the VM service the app is connected
// to, when there is one (nil otherwise).
var ConnectedVmServiceUri *string

// DebugMaybeDispatchCreated dispatches an ObjectCreated event for object, of
// class className in package:flutter/flutterLibrary.dart, to
// FlutterMemoryAllocations when allocations are tracked. It returns true,
// to be used in an assert.
func DebugMaybeDispatchCreated(flutterLibrary, className string, object any) bool {
	if KFlutterMemoryAllocationsEnabled {
		FlutterMemoryAllocationsInstance.DispatchObjectCreated("package:flutter/"+flutterLibrary+".dart", className, object)
	}
	return true
}

// DebugMaybeDispatchDisposed dispatches an ObjectDisposed event for object
// when allocations are tracked. It returns true, to be used in an assert.
func DebugMaybeDispatchDisposed(object any) bool {
	if KFlutterMemoryAllocationsEnabled {
		FlutterMemoryAllocationsInstance.DispatchObjectDisposed(object)
	}
	return true
}

// dartAssert is Dart's assert(condition, message): Go keeps assertions on
// in every build (see API_MAPPING), so a failure panics with an
// AssertionError (nil message for a bare assert).
func dartAssert(condition bool, message any) {
	if !condition {
		panic(dart.NewAssertionError(message))
	}
}
