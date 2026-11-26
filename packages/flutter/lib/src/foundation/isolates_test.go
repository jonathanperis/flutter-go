// Port of packages/flutter/test/foundation/isolates_test.dart.
//
// "compute()": ported the cases with a Go equivalent: a result (test1), a
// throw (test2: the future fails with the thrown value), compute called
// from inside a computation (test1CallCompute, test2CallCompute: the inner
// compute uses its own loop, as an isolate has its own event loop), a
// static method, a closure, an instance method, a null result. Skipped:
// Isolate.exit / Isolate.current.kill (test3-5 and their async and
// CallCompute variants: no Go equivalent), the async callbacks
// (test1Async...: a Go ComputeCallback returns R, since the event loop
// belongs to the calling goroutine), unsendable messages, results and
// errors (ReceivePort: Go shares memory, nothing is unsendable), the
// isolate debugName. Skipped: "compute() closes all ports" (4 tests that
// run _compute_caller*.dart in a separate Dart VM to check it exits).
package foundation

import (
	"errors"
	"testing"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
)

func test1(value int) int { return value + 1 }
func test2(value int) int { panic(2) }

// computeAndWait runs Compute's future to completion, as the inner
// isolate's event loop would.
func computeAndWait[R any](f func() async.Future[R]) (R, error) {
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	var (
		value R
		err   error
		done  bool
	)
	async.OnComplete(f(), func(v R) { value, done = v, true }, func(e error) { err, done = e, true })
	for !done {
		l.WaitPosted()
	}
	return value, err
}

func test1CallCompute(value int) int {
	v, err := computeAndWait(func() async.Future[int] { return Compute(test1, value) })
	if err != nil {
		panic(err)
	}
	return v
}

func test2CallCompute(value int) int {
	v, err := computeAndWait(func() async.Future[int] { return Compute(test2, value) })
	if err != nil {
		panic(err)
	}
	return v
}

type computeTestSubject struct{ base int }

func (s computeTestSubject) method(x int) int { return s.base * x }

func computeTestSubjectStaticMethod(square int) int { return square * square }

func isThrown2(err error) bool {
	var pe async.PanicError
	return errors.As(err, &pe) && pe.Value == 2
}

func TestCompute(t *testing.T) {
	expectValue := func(name string, f func() async.Future[int], want int) {
		t.Helper()
		v, err := computeAndWait(f)
		if err != nil || v != want {
			t.Fatalf("%s: %v, %v", name, v, err)
		}
	}
	expectThrows2 := func(name string, f func() async.Future[int]) {
		t.Helper()
		if _, err := computeAndWait(f); !isThrown2(err) {
			t.Fatalf("%s: err = %#v", name, err)
		}
	}
	expectValue("test1", func() async.Future[int] { return Compute(test1, 0) }, 1)
	expectThrows2("test2", func() async.Future[int] { return Compute(test2, 0) })
	expectValue("test1CallCompute", func() async.Future[int] { return Compute(test1CallCompute, 0) }, 1)
	expectThrows2("test2CallCompute", func() async.Future[int] { return Compute(test2CallCompute, 0) })
	expectValue("static method", func() async.Future[int] { return Compute(computeTestSubjectStaticMethod, 10) }, 100)
	square := 10
	expectValue("closure", func() async.Future[int] {
		return Compute(func(any) int { return square * square }, nil)
	}, 100)
	subject := computeTestSubject{base: 10}
	expectValue("instance method", func() async.Future[int] { return Compute(subject.method, 10) }, 100)
	label := "debug_name"
	expectValue("debugLabel", func() async.Future[int] {
		return Compute(test1, 1, ComputeOptions{DebugLabel: &label})
	}, 2)
	v, err := computeAndWait(func() async.Future[*int] {
		return Compute(func(any) *int { return nil }, nil)
	})
	if err != nil || v != nil {
		t.Fatal("testReturnNull")
	}
}

// Not in Flutter's tests: the result is delivered through the loop, in an
// event of its own, never synchronously.
func TestComputeDeliversThroughTheEventLoop(t *testing.T) {
	l := async.NewLoop()
	defer async.SetCurrent(l)()
	release := make(chan struct{})
	f := Compute(func(int) int { <-release; return 5 }, 0)
	got := 0
	async.OnComplete(f, func(v int) { got = v }, nil)
	l.FlushMicrotasks()
	close(release)
	equal(t, got, 0)
	l.WaitPosted()
	equal(t, got, 5)
}
