package foundation

// Port of isolates.dart with _isolates_io.dart (Flutter 3.50.1).
//
// Go has no isolates: Compute runs the callback on a new goroutine (in
// parallel, like an isolate) and delivers its result through the calling
// event loop, as an event posted to async.Loop (Isolate.run's result port).
// Differences, all from the absence of isolates: memory is shared, so
// there is no "unsendable message" error; the callback must not use the
// event loop (async), which belongs to the calling goroutine (a Dart
// isolate has its own), so ComputeCallback returns R rather than
// FutureOr<R>; Isolate.exit/kill and the isolate's debugName have no Go
// equivalent.

import (
	"github.com/jonathanperis/flutter-go/src/async"
)

// ComputeCallback is the signature of a function Compute runs.
type ComputeCallback[M, R any] func(message M) R

// ComputeOptions are Compute's optional parameters.
type ComputeOptions struct {
	// DebugLabel names the computation (Dart: the isolate's debugName);
	// default "compute" in release builds, the callback's description
	// otherwise. Go has no isolate to show it.
	DebugLabel *string
}

// ComputeImpl is the signature of Compute.
type ComputeImpl[M, R any] func(callback ComputeCallback[M, R], message M, opts ...ComputeOptions) async.Future[R]

// Compute runs callback(message) on another goroutine and returns a future
// of its result, delivered through the current event loop: the loop's
// owner must run its posted events (async.Loop.RunPosted / WaitPosted).
// A panic in callback fails the future with that error.
func Compute[M, R any](callback ComputeCallback[M, R], message M, opts ...ComputeOptions) async.Future[R] {
	// The body of the Dart `async` function: it returns Isolate.run's
	// future without awaiting it.
	var debugLabel string
	if len(opts) > 0 && opts[0].DebugLabel != nil {
		debugLabel = *opts[0].DebugLabel
	} else if KReleaseMode {
		debugLabel = "compute"
	} else {
		debugLabel = dartToString(callback)
	}
	_ = debugLabel
	result := async.NewCompleter[R]()
	result.CompleteFuture(isolateRun(func() R { return callback(message) }))
	return result.Future()
}

// isolateRun is Isolate.run: computation runs elsewhere; its result (or
// error) completes an asynchronous completer from an event of this loop.
func isolateRun[R any](computation func() R) async.Future[R] {
	loop := async.Current()
	result := async.NewCompleter[R]()
	go func() {
		var value R
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = panicAsError(r)
				}
			}()
			value = computation()
		}()
		loop.Post(func() {
			if err != nil {
				result.CompleteError(err)
				return
			}
			result.Complete(value)
		})
	}()
	return result.Future()
}
