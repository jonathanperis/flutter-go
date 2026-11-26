package services_test

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

// testBinding is BindingBase with SchedulerBinding, ServicesBinding,
// TestDefaultBinaryMessengerBinding: the part of TestWidgetsFlutterBinding
// these tests use.
type testBinding struct {
	flutter_test.TestDefaultBinaryMessengerBindingMixin
}

var (
	bindingOnce sync.Once
	theBinding  *testBinding
)

// ensureInitialized is TestWidgetsFlutterBinding.ensureInitialized() for
// these tests: the binding is created once per process.
func ensureInitialized() *testBinding {
	bindingOnce.Do(func() {
		loop := async.NewLoop()
		restore := async.SetCurrent(loop)
		defer restore()
		// Run what construction scheduled (the channel buffers' drains, the
		// engine's reply to initializationComplete), as Dart's event loop would.
		defer func() {
			loop.FlushMicrotasks()
			loop.RunPosted()
		}()
		b := &testBinding{}
		base := &foundation.BindingBaseBase{}
		sched := &scheduler.SchedulerBindingMixin{BindingBase: base}
		svc := &services.ServicesBindingMixin{SchedulerBinding: sched}
		b.TestDefaultBinaryMessengerBindingMixin.ServicesBinding = svc
		sched.SetSelf(b)
		svc.SetSelf(b)
		b.TestDefaultBinaryMessengerBindingMixin.SetSelf(b)
		base.Construct(b)
		theBinding = b
	})
	return theBinding
}

// messenger is TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.
func messenger() *flutter_test.TestDefaultBinaryMessenger {
	return flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
}

// runAsync runs body as a Dart async test body on a fresh event loop,
// running microtasks, posted events (engine replies) and timers until it
// completes; a panic in body fails the test.
func runAsync(t *testing.T, body func()) {
	t.Helper()
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	flutter_test.SetCurrentTest(t)
	var uncaught []error
	loop.UncaughtError = func(err error) { uncaught = append(uncaught, err) }
	done := false
	var failure error
	async.OnComplete(async.AsyncVoid(body), func(async.Void) { done = true }, func(err error) { failure = err; done = true })
	for i := 0; !done && i < 10000; i++ {
		loop.FlushMicrotasks()
		if done || loop.RunPosted() > 0 {
			continue
		}
		if d, ok := loop.NextDeadline(); ok {
			loop.ElapseTo(d)
		} else if !done {
			break
		}
	}
	loop.FlushMicrotasks()
	if failure != nil {
		t.Fatalf("test body failed: %v", failure)
	}
	if !done {
		t.Fatal("test body never completed")
	}
	for _, err := range uncaught {
		t.Errorf("uncaught error: %v", err)
	}
}

// expect panics (failing the async body) unless cond holds.
func expect(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Errorf("expectation failed: "+format, args...))
	}
}

// awaitError awaits f and returns its error (nil if it succeeded).
func awaitError[T any](f async.Future[T]) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	f.Await()
	return nil
}

const isolatedEnv = "FLUTTER_GO_SERVICES_ISOLATED_TEST"

// isolated re-runs the calling test in a child process (a test that needs
// the binding not to exist yet: each Dart test file is its own isolate);
// it returns true in the child, where the test body runs.
func isolated(t *testing.T) bool {
	t.Helper()
	if os.Getenv(isolatedEnv) == t.Name() {
		return true
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), isolatedEnv+"="+t.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s in its own process: %v\n%s", t.Name(), err, out)
	}
	return false
}
