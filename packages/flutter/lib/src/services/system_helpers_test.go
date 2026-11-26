package services_test

// Helpers for the ports of the system services tests (system chrome,
// navigator, sound, haptics, restoration...): method call logs, and the
// pump/idle of testWidgets' tester for tests running in this package's
// binding (BindingBase with Scheduler, Services, TestDefaultBinaryMessenger).

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
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
	if ea, ok := mapEntriesOf(a); ok {
		eb, ok := mapEntriesOf(b)
		if !ok || len(ea) != len(eb) {
			return false
		}
		for _, e := range ea {
			w, ok := services.MapLookup(b, e.Key)
			if !ok || !decodedEqual(e.Value, w) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

// mapEntriesOf lists the entries of a map value (a *dart.LinkedMap, a Go
// map or a []dart.MapEntry): ok is false for other values.
func mapEntriesOf(v any) (entries []dart.MapEntry[any, any], ok bool) {
	if m, isLinked := v.(*dart.LinkedMap); isLinked {
		return m.EntryList(), m != nil
	}
	rv := reflect.ValueOf(v)
	switch {
	case rv.Kind() == reflect.Map:
		for it := rv.MapRange(); it.Next(); {
			entries = append(entries, dart.MapEntry[any, any]{Key: it.Key().Interface(), Value: it.Value().Interface()})
		}
		return entries, true
	case rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Struct &&
		strings.HasPrefix(rv.Type().Elem().Name(), "MapEntry["):
		for i := range rv.Len() {
			entries = append(entries, dart.MapEntry[any, any]{Key: rv.Index(i).Field(0).Interface(), Value: rv.Index(i).Field(1).Interface()})
		}
		return entries, true
	}
	return nil, false
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
