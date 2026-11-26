package services_test

// Helpers for the ports of the system services tests (system chrome,
// navigator, sound, haptics, restoration...): method call logs, and the
// pump/idle of testWidgets' tester for tests running in this package's
// binding (BindingBase with Scheduler, Services, TestDefaultBinaryMessenger).

import (
	"reflect"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

// logMethodCalls installs a mock handler on channel that records the calls
// it gets and answers null (Dart: `log.add(methodCall); return null;`); it
// returns the log and a function removing the handler.
func logMethodCalls(channel *services.MethodChannel) (*[]services.MethodCall, func()) {
	log := &[]services.MethodCall{}
	messenger().SetMockMethodCallHandler(channel, func(call services.MethodCall) async.Future[any] {
		return async.Async(func() any {
			*log = append(*log, call)
			return nil
		})
	})
	return log, func() { messenger().SetMockMethodCallHandler(channel, nil) }
}

// isMethodCall is flutter_test's isMethodCall(name, arguments:) for decoded
// arguments: maps compare by content in any order (Dart's equals on maps).
func isMethodCall(call services.MethodCall, method string, arguments any) bool {
	return call.Method == method && decodedEqual(call.Arguments, arguments)
}

func decodedEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Kind() == reflect.Map || vb.Kind() == reflect.Map {
		if va.Kind() != reflect.Map || vb.Kind() != reflect.Map || va.Len() != vb.Len() {
			return false
		}
		for _, k := range va.MapKeys() {
			if !k.Type().AssignableTo(vb.Type().Key()) && !(k.Kind() == reflect.Interface) {
				return false
			}
			key := reflect.ValueOf(k.Interface())
			if !key.Type().AssignableTo(vb.Type().Key()) {
				return false
			}
			w := vb.MapIndex(key)
			if !w.IsValid() || !decodedEqual(va.MapIndex(k).Interface(), w.Interface()) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

// idle is tester.idle() inside a runAsync body: the event loop runs its
// microtasks and the timers due now.
func idle() {
	async.FutureDelayed(0, func() async.Void { return async.Void{} }).Await()
}

// pump is tester.pump() inside a runAsync body: a frame is produced if one
// is scheduled (handleBeginFrame, microtasks, handleDrawFrame), then the
// microtasks run.
func pump() {
	b := ensureInitialized()
	idle()
	if b.HasScheduledFrame() {
		stamp := async.Current().Elapsed() + time.Second
		b.HandleBeginFrame(&stamp)
		idle()
		b.HandleDrawFrame()
	}
	idle()
}

// runAsyncWaiting is runAsync for a body awaiting events from other
// goroutines (an HTTP request, compute): when the loop has nothing left to
// run, it waits for a posted event (up to 10 s).
func runAsyncWaiting(t *testing.T, body func()) {
	t.Helper()
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	done := false
	var failure error
	async.OnComplete(async.AsyncVoid(body), func(async.Void) { done = true }, func(err error) { failure = err; done = true })
	deadline := time.Now().Add(10 * time.Second)
	for !done && time.Now().Before(deadline) {
		loop.FlushMicrotasks()
		if done || loop.RunPosted() > 0 {
			continue
		}
		if d, ok := loop.NextDeadline(); ok {
			loop.ElapseTo(d)
			continue
		}
		loop.WaitPosted()
	}
	loop.FlushMicrotasks()
	if failure != nil {
		t.Fatalf("test body failed: %v", failure)
	}
	if !done {
		t.Fatal("test body never completed")
	}
}
