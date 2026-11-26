// Port of packages/flutter/test/foundation/synchronous_future_test.dart.
//
// Skipped parts of "SynchronousFuture control test":
//   - asStream: streams are not ported.
//   - the whenComplete action returning Future<int>.value(31): a Go action
//     can't return a future (the test only checks that the value is not
//     propagated, which holds trivially).
package foundation

import (
	"errors"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
)

type argumentError struct{}

func (argumentError) Error() string { return "Invalid argument(s)" }

func TestSynchronousFutureControl(t *testing.T) {
	l := async.NewLoop()
	defer async.SetCurrent(l)()

	done := async.AsyncVoid(func() {
		var future async.Future[int] = NewSynchronousFuture(42)

		var result *int
		async.Then(future, func(v int) async.Void { result = &v; return async.Void{} })
		if result == nil || *result != 42 {
			t.Fatalf("then did not run synchronously: %v", result)
		}
		result = nil

		futureWithTimeout := future.Timeout(time.Millisecond, nil)
		async.Then(futureWithTimeout, func(v int) async.Void { result = &v; return async.Void{} })
		if result != nil {
			t.Fatalf("timeout's then ran synchronously")
		}
		futureWithTimeout.Await()
		if result == nil || *result != 42 {
			t.Fatalf("after await: %v", result)
		}
		result = nil

		ranAction := false
		completeResult := future.WhenComplete(func() { ranAction = true })
		if !ranAction {
			t.Fatal("whenComplete did not run its action synchronously")
		}
		ranAction = false
		if got := completeResult.Await(); got != 42 {
			t.Fatalf("whenComplete result = %d", got)
		}

		var exception any
		func() {
			defer func() { exception = recover() }()
			future.WhenComplete(func() { panic(argumentError{}) }).Await()
			t.Fatal("unreached")
		}()
		if !errors.As(exception.(error), new(argumentError)) {
			t.Fatalf("exception = %#v", exception)
		}
	})
	failed := true
	async.OnComplete(done, func(async.Void) { failed = false }, func(err error) { t.Fatal(err) })
	l.FlushTimers(time.Hour, true)
	if failed {
		t.Fatal("test body did not complete")
	}
}

// Not in Flutter's test: the rest of the class's contract.
func TestSynchronousFutureContract(t *testing.T) {
	l := async.NewLoop()
	defer async.SetCurrent(l)()

	f := NewSynchronousFuture("a")
	// then returning a non-future gives a SynchronousFuture (chains stay
	// synchronous); returning a future gives that future.
	var log []string
	async.Then(async.Then(f, func(v string) string { return v + "b" }), func(v string) async.Void {
		log = append(log, v)
		return async.Void{}
	})
	if len(log) != 1 || log[0] != "ab" {
		t.Fatalf("chain: %q", log)
	}
	c := async.NewCompleter[string]()
	if got := async.ThenFuture(f, func(string) async.Future[string] { return c.Future() }); got != c.Future() {
		t.Fatalf("then returning a future should return it as is: %#v", got)
	}

	// catchError never completes.
	caught := false
	async.OnComplete(f.CatchError(func(error) string { return "x" }, nil),
		func(string) { caught = true }, func(error) { caught = true })
	l.FlushTimers(time.Hour, true)
	if caught {
		t.Fatal("catchError's future completed")
	}

	// whenComplete returns the same future.
	if got := f.WhenComplete(func() {}); got != async.Future[string](f) {
		t.Fatalf("whenComplete returned %#v", got)
	}

	// await doesn't suspend.
	order := []string{}
	async.AsyncVoid(func() {
		order = append(order, "before")
		order = append(order, f.Await())
	})
	order = append(order, "after")
	if len(order) != 3 || order[1] != "a" || order[2] != "after" {
		t.Fatalf("await order: %q", order)
	}
}
