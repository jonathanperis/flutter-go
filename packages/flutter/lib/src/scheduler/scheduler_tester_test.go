// Port of packages/flutter/test/scheduler/scheduler_tester.dart, plus the
// helpers the Go ports of the scheduler tests share.
//
// Dart runs each test file in its own isolate, with its own binding (a
// binding can be constructed once per process). Go runs a package's tests
// in one process, so each ported file's tests are one top-level test that
// runs in a child process of its own (isolated), creating that file's
// binding there.
package scheduler_test

import (
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
)

// tick is scheduler_tester.dart's tick: a frame without running microtasks
// between the two phases (these tests don't use futures).
func tick(duration time.Duration) {
	scheduler.SchedulerBindingInstance().HandleBeginFrame(&duration)
	scheduler.SchedulerBindingInstance().HandleDrawFrame()
}

const isolatedEnv = "FLUTTER_GO_SCHEDULER_TEST_ISOLATED"

// isolated runs the calling top-level test in a process of its own (see
// the file comment): it returns true in the child, where the test body
// runs, and false in the parent once the child has passed.
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
	if testing.Verbose() {
		t.Logf("%s", out)
	}
	return false
}

// fakeAsync gives the test a fresh event loop until it ends.
func fakeAsync(t testing.TB) *async.Loop {
	loop := async.NewLoop()
	t.Cleanup(async.SetCurrent(loop))
	return loop
}

// schedulerOnlyBinding is `BindingBase with SchedulerBinding`; test
// bindings embed it (Dart: extend it) and override its methods.
type schedulerOnlyBinding struct {
	foundation.BindingBaseBase
	scheduler.SchedulerBindingMixin
}

func (b *schedulerOnlyBinding) InitInstances() { b.SchedulerBindingMixin.InitInstances() }
func (b *schedulerOnlyBinding) InitServiceExtensions() {
	b.SchedulerBindingMixin.InitServiceExtensions()
}
func (b *schedulerOnlyBinding) Unlocked() { b.SchedulerBindingMixin.Unlocked() }

// construct runs the construction of self, a binding embedding b.
func construct(self scheduler.SchedulerBinding, b *schedulerOnlyBinding) {
	b.SchedulerBindingMixin.SetSelf(self)
	b.Construct(self, &b.SchedulerBindingMixin)
}

// plainBinding is a binding of BindingBase and SchedulerBinding only (the
// scheduler part of WidgetsFlutterBinding, and animation_test.dart's
// TestSchedulerBinding).
type plainBinding struct {
	schedulerOnlyBinding
}

func newPlainBinding() *plainBinding {
	b := &plainBinding{}
	construct(b, &b.schedulerOnlyBinding)
	return b
}

func expectPanics[E any](t *testing.T, f func()) (err E) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic with %T", err)
		}
		e, ok := r.(E)
		if !ok {
			t.Fatalf("panicked with %T (%v), want %T", r, r, err)
		}
		err = e
	}()
	f()
	return err
}

func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// memoryEvents is leak_tracker's memoryEvents: the FlutterMemoryAllocations
// events callback dispatches for objects of type T.
func memoryEvents[T any](callback func()) []foundation.ObjectEvent {
	var events []foundation.ObjectEvent
	listener := func(event foundation.ObjectEvent) {
		if _, ok := event.Object().(T); ok {
			events = append(events, event)
		}
	}
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	callback()
	return events
}

// areCreateAndDispose is leak_tracker's matcher: one creation, then one
// disposal of the same object.
func areCreateAndDispose(t *testing.T, events []foundation.ObjectEvent) {
	t.Helper()
	if len(events) != 2 {
		t.Fatalf("%d events, want 2", len(events))
	}
	if _, ok := events[0].(*foundation.ObjectCreated); !ok {
		t.Fatalf("first event %T, want ObjectCreated", events[0])
	}
	if _, ok := events[1].(*foundation.ObjectDisposed); !ok {
		t.Fatalf("second event %T, want ObjectDisposed", events[1])
	}
	if events[0].Object() != events[1].Object() {
		t.Fatal("events are for different objects")
	}
}
