// Port of packages/flutter/test/foundation/debug_test.dart (all 4 tests),
// plus tests of debugAssertAllFoundationVarsUnset and the non-instrumented
// path of debugInstrumentAction, which Flutter covers only indirectly.
package foundation

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

// runLoop runs body as an async test body on a fresh loop until it
// completes (Dart's `test(..., () async {...})`), returning its error.
func runLoop(t *testing.T, body func()) error {
	t.Helper()
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	var result error
	done := false
	async.OnComplete(async.AsyncVoid(body), func(async.Void) { done = true },
		func(err error) { done, result = true, err })
	for i := 0; i < 1000 && !done; i++ {
		l.FlushTimers(time.Hour, true)
		if !done {
			l.RunPosted()
		}
	}
	if !done {
		t.Fatal("async test body did not complete")
	}
	return result
}

func withInstrumentation(t *testing.T) *strings.Builder {
	DebugInstrumentationEnabled = true
	printBuffer := &strings.Builder{}
	original := DebugPrint
	DebugPrint = func(message string, _ ...DebugPrintOptions) { printBuffer.WriteString(message + "\n") }
	t.Cleanup(func() {
		DebugInstrumentationEnabled = false
		DebugPrint = original
	})
	return printBuffer
}

func TestDebugInstrumentActionWorksWithNonFailingActions(t *testing.T) {
	printBuffer := withInstrumentation(t)
	var result int
	err := runLoop(t, func() {
		result = DebugInstrumentAction("no-op", func() async.Future[int] {
			return async.Async(func() int {
				DebugPrint("action()")
				return 1
			})
		}).Await()
	})
	if err != nil {
		t.Fatal(err)
	}
	equal(t, result, 1)
	if !regexp.MustCompile(`(?m)^action\(\)\nAction "no-op" took .+$`).MatchString(printBuffer.String()) {
		t.Fatalf("%q", printBuffer.String())
	}
}

func TestDebugInstrumentActionReturnsFailingFutureIfActionThrows(t *testing.T) {
	printBuffer := withInstrumentation(t)
	err := runLoop(t, func() {
		DebugInstrumentAction("throws", func() async.Future[async.Void] {
			return async.AsyncVoid(func() {
				async.FutureDelayed[async.Void](0, nil).Await()
				panic("Error")
			})
		}).Await()
	})
	var pe async.PanicError
	if !errors.As(err, &pe) || pe.Value != "Error" {
		t.Fatalf("err = %#v", err)
	}
	if !regexp.MustCompile(`^Action "throws" took .+`).MatchString(printBuffer.String()) {
		t.Fatalf("%q", printBuffer.String())
	}
}

func TestDebugInstrumentActionDisabledChainsTheFuture(t *testing.T) {
	printed := false
	original := DebugPrint
	DebugPrint = func(string, ...DebugPrintOptions) { printed = true }
	defer func() { DebugPrint = original }()
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	c := async.NewCompleter[int]()
	f := DebugInstrumentAction("x", func() async.Future[int] { return c.Future() })
	if f == c.Future() {
		t.Fatal("the result must be a new future, as an async function's")
	}
	got := 0
	async.OnComplete(f, func(v int) { got = v }, nil)
	c.Complete(7)
	l.FlushMicrotasks()
	equal(t, got, 7)
	if printed {
		t.Fatal("printed while instrumentation is disabled")
	}
	// A synchronous throw fails the future (in a microtask).
	failed := DebugInstrumentAction("x", func() async.Future[int] { panic("sync") })
	var failure error
	async.OnComplete(failed, func(int) {}, func(err error) { failure = err })
	l.FlushMicrotasks()
	if failure == nil || failure.Error() != "sync" {
		t.Fatalf("failure = %v", failure)
	}
}

func TestMemoryAllocationsDebugMaybeDispatch(t *testing.T) {
	object := []int{0}
	var dispatchedEvent ObjectEvent
	listener := func(event ObjectEvent) {
		if dispatchedEvent != nil {
			t.Fatal("dispatched twice")
		}
		dispatchedEvent = event
	}
	ma := FlutterMemoryAllocationsInstance

	t.Run("debugMaybeDispatchCreated", func(t *testing.T) {
		dispatchedEvent = nil
		ma.AddListener(listener)
		defer ma.RemoveListener(listener)
		DebugMaybeDispatchCreated("library", "class", &object)
		theEvent, ok := dispatchedEvent.(*ObjectCreated)
		if !ok {
			t.Fatal("Expected ObjectCreated event")
		}
		if theEvent.Object() != any(&object) {
			t.Fatal("object")
		}
		equal(t, theEvent.Library, "package:flutter/library.dart")
		equal(t, theEvent.ClassName, "class")
	})
	t.Run("debugMaybeDispatchDisposed", func(t *testing.T) {
		dispatchedEvent = nil
		ma.AddListener(listener)
		defer ma.RemoveListener(listener)
		DebugMaybeDispatchDisposed(&object)
		theEvent, ok := dispatchedEvent.(*ObjectDisposed)
		if !ok {
			t.Fatal("Expected ObjectDisposed event")
		}
		if theEvent.Object() != any(&object) {
			t.Fatal("object")
		}
	})
}

func TestDebugAssertAllFoundationVarsUnset(t *testing.T) {
	equal(t, DebugAssertAllFoundationVarsUnset("reason"), true)
	expectFlutterError := func(name string, change, restore func()) {
		t.Helper()
		change()
		defer restore()
		defer func() {
			t.Helper()
			r := recover()
			fe, ok := r.(*FlutterError)
			if !ok || fe.Error() != "reason "+name {
				t.Fatalf("%s: recovered %#v", name, r)
			}
		}()
		DebugAssertAllFoundationVarsUnset("reason " + name)
	}
	original := DebugPrint
	expectFlutterError("debugPrint", func() { DebugPrint = DebugPrintSynchronously }, func() { DebugPrint = original })
	expectFlutterError("platform", func() { SetDebugDefaultTargetPlatformOverride(ptr(TargetPlatformIOS)) },
		func() { SetDebugDefaultTargetPlatformOverride(nil) })
	expectFlutterError("precision", func() { DebugDoublePrecision = ptr(2) }, func() { DebugDoublePrecision = nil })
	expectFlutterError("brightness", func() { DebugBrightnessOverride = ptr(ui.BrightnessDark) },
		func() { DebugBrightnessOverride = nil })
	// The expected debugPrint can be overridden.
	DebugPrint = DebugPrintSynchronously
	equal(t, DebugAssertAllFoundationVarsUnset("reason", DebugAssertAllFoundationVarsUnsetOptions{
		DebugPrintOverride: DebugPrintSynchronously,
	}), true)
	DebugPrint = original
}
