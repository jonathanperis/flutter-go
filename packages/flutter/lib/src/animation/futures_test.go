package animation

// Port of packages/flutter/test/animation/futures_test.dart (Flutter
// 3.50.1): 6/6 tests, driven by setUp's fake event loop and flutter_test's
// pump (scheduler_tester_test.go). "TickerFuture is a Future" checks
// catchError/then/whenComplete/timeout; Dart's asStream is not ported (see
// API_MAPPING "Futures"). tester.idle() is elapse(Duration.zero).

import (
	"slices"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
)

func expectLog(t *testing.T, log []string, want ...string) {
	t.Helper()
	if !slices.Equal(log, want) && !(len(log) == 0 && len(want) == 0) {
		t.Fatalf("log %q, want %q", log, want)
	}
}

func awaitingControllers(t *testing.T, orCancel bool) {
	st := setUp(t)
	c1 := newController(AnimationControllerOptions{Duration: dur(100 * ms)})
	t.Cleanup(c1.Dispose)
	c2 := newController(AnimationControllerOptions{Duration: dur(600 * ms)})
	t.Cleanup(c2.Dispose)
	c3 := newController(AnimationControllerOptions{Duration: dur(300 * ms)})
	t.Cleanup(c3.Dispose)
	var log []string
	await := func(f *scheduler.TickerFuture) {
		if orCancel {
			f.OrCancel().Await()
		} else {
			f.Await()
		}
	}
	runTest := func() async.Future[async.Void] {
		return async.AsyncVoid(func() {
			log = append(log, "a") // t=0
			await(c1.Forward())    // starts at t=0 again
			log = append(log, "b") // wants to end at t=100 but missed frames until t=150
			await(c2.Forward())    // starts at t=150
			log = append(log, "c") // wants to end at t=750 but missed frames until t=799
			await(c3.Forward())    // starts at t=799
			log = append(log, "d") // wants to end at t=1099 but missed frames until t=1200
		})
	}
	log = append(log, "start")
	async.OnComplete(runTest(), func(async.Void) { log = append(log, "end") }, nil)
	st.pump() // t=0
	expectLog(t, log, "start", "a")
	st.pump() // t=0 again
	expectLog(t, log, "start", "a")
	st.pump(50 * ms) // t=50
	expectLog(t, log, "start", "a")
	st.pump(100 * ms) // t=150
	expectLog(t, log, "start", "a", "b")
	st.pump(50 * ms) // t=200
	expectLog(t, log, "start", "a", "b")
	st.pump(400 * ms) // t=600
	expectLog(t, log, "start", "a", "b")
	st.pump(199 * ms) // t=799
	expectLog(t, log, "start", "a", "b", "c")
	st.pump(51 * ms) // t=850
	expectLog(t, log, "start", "a", "b", "c")
	st.pump(400 * ms) // t=1200
	expectLog(t, log, "start", "a", "b", "c", "d", "end")
	st.pump(400 * ms) // t=1600
	expectLog(t, log, "start", "a", "b", "c", "d", "end")
}

func TestAwaitingAnimationControllersUsingDirectFuture(t *testing.T) { awaitingControllers(t, false) }

func TestAwaitingAnimationControllersUsingOrCancel(t *testing.T) { awaitingControllers(t, true) }

func TestAwaitingAnimationControllersAndFailing(t *testing.T) {
	st := setUp(t)
	c1 := newController(AnimationControllerOptions{Duration: dur(100 * ms)})
	var log []string
	runTest := func() async.Future[async.Void] {
		return async.AsyncVoid(func() {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(scheduler.TickerCanceled); !ok {
						panic(r)
					}
					log = append(log, "caught")
				}
			}()
			log = append(log, "start")
			c1.Forward().OrCancel().Await()
			log = append(log, "fail")
		})
	}
	async.OnComplete(runTest(), func(async.Void) { log = append(log, "end") }, nil)
	st.pump() // start ticker
	expectLog(t, log, "start")
	st.pump(50 * ms)
	expectLog(t, log, "start")
	c1.Dispose()
	expectLog(t, log, "start")
	st.loop.Elapse(0) // tester.idle()
	expectLog(t, log, "start", "caught", "end")
}

func TestCreatingOrCancelFutureLater(t *testing.T) {
	st := setUp(t)
	c1 := newController(AnimationControllerOptions{Duration: dur(100 * ms)})
	t.Cleanup(c1.Dispose)
	f := c1.Forward()
	st.pump()         // start ticker
	st.pump(200 * ms) // end ticker
	reached := false
	async.AsyncVoid(func() {
		f.Await()            // should be a no-op
		f.OrCancel().Await() // should create a resolved future
		reached = true
	})
	st.loop.FlushMicrotasks()
	expect(t, reached, true)
}

func TestCreatingOrCancelFutureLaterAfterCancel(t *testing.T) {
	st := setUp(t)
	c1 := newController(AnimationControllerOptions{Duration: dur(100 * ms)})
	t.Cleanup(c1.Dispose)
	f := c1.Forward()
	st.pump() // start ticker
	c1.Stop() // cancel ticker
	ok := false
	async.AsyncVoid(func() {
		defer func() {
			if _, canceled := recover().(scheduler.TickerCanceled); canceled {
				ok = true
			}
		}()
		f.OrCancel().Await() // should create a resolved future
	})
	st.loop.FlushMicrotasks()
	expect(t, ok, true)
}

func TestTickerFutureIsAFuture(t *testing.T) {
	st := setUp(t)
	c1 := newController(AnimationControllerOptions{Duration: dur(100 * ms)})
	t.Cleanup(c1.Dispose)
	f := c1.Forward()
	st.pump()         // start ticker
	st.pump(200 * ms) // end ticker
	var thenResult *bool
	async.AsyncVoid(func() {
		f.CatchError(func(error) async.Void { panic("do not reach") }, nil).Await()
		v := async.Then[async.Void, bool](f, func(async.Void) bool { return true }).Await()
		thenResult = &v
	})
	_ = f.WhenComplete(func() {})
	timeout := f.Timeout(5*time.Second, nil)
	st.loop.FlushMicrotasks()
	if thenResult == nil || !*thenResult {
		t.Fatal("then did not complete with true")
	}
	done := false
	async.OnComplete(timeout, func(async.Void) { done = true }, nil)
	st.loop.FlushMicrotasks()
	expect(t, done, true)
}
