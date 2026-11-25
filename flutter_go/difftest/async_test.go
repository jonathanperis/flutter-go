package difftest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/flutter_go/async"
)

type stateError struct{ msg string }

func (e stateError) Error() string { return "Bad state: " + e.msg }

func msg(err error) string {
	var se stateError
	if errors.As(err, &se) {
		return se.msg
	}
	return err.Error()
}

const ms = time.Millisecond

// Same scenes as flutter/test/async_test.dart.
var asyncScenes = map[string]func(l *Loop, log func(string)){
	"queue_order": func(l *Loop, log func(string)) {
		TimerRun(func() { log("timer.run") })
		ScheduleMicrotask(func() { log("micro1") })
		Then(FutureValue(1), func(v int) Void { log(fmt.Sprint("value ", v)); return Void{} })
		Then(NewFuture(func() int { return 2 }), func(v int) Void { log(fmt.Sprint("future() ", v)); return Void{} })
		Then(FutureMicrotask(func() int { return 3 }), func(v int) Void { log(fmt.Sprint("microtask ", v)); return Void{} })
		ScheduleMicrotask(func() {
			log("micro2")
			ScheduleMicrotask(func() { log("micro2.nested") })
		})
		log("sync end")
		l.FlushMicrotasks()
		log("after flushMicrotasks")
		l.Elapse(0)
		log("after elapse 0")
	},
	"completers": func(l *Loop, log func(string)) {
		a, s := NewCompleter[string](), NewSyncCompleter[string]()
		a.Future().OnComplete(func(v string) { log("async " + v) }, nil)
		s.Future().OnComplete(func(v string) { log("sync " + v) }, nil)
		a.Complete("a")
		log("after a.complete")
		s.Complete("s")
		log("after s.complete")
		s.Future().OnComplete(func(v string) { log("late sync " + v) }, nil)
		log("end")
		l.FlushMicrotasks()
	},
	"listener_order_and_chains": func(l *Loop, log func(string)) {
		c := NewSyncCompleter[int]()
		Then(Then(c.Future(), func(v int) int { return v + 1 }), func(v int) Void { log(fmt.Sprint("chain1 ", v)); return Void{} })
		Then(c.Future(), func(v int) Void { log(fmt.Sprint("direct ", v)); return Void{} })
		Then(ThenFuture(c.Future(), func(v int) *Future[int] { return FutureValue(v * 10) }),
			func(v int) Void { log(fmt.Sprint("chainFuture ", v)); return Void{} })
		c.Future().WhenComplete(func() { log("whenComplete") })
		c.Complete(1)
		log("completed")
		l.FlushMicrotasks()
	},
	"async_functions": func(l *Loop, log func(string)) {
		f := func(name string, dep *Future[int]) *Future[int] {
			return Async(func() int {
				log(name + " start")
				v := dep.Await()
				log(fmt.Sprint(name, " resumed ", v))
				return v + 1
			})
		}
		result := func(name string) func(int) { return func(v int) { log(fmt.Sprint(name, " result ", v)) } }
		gate := NewCompleter[int]()
		f("a", FutureValue(1)).OnComplete(result("a"), nil)
		f("b", gate.Future()).OnComplete(result("b"), nil)
		done := FutureValue(7)
		l.FlushMicrotasks()
		f("c", done).OnComplete(result("c"), nil)
		log("before gate")
		gate.Complete(5)
		log("after gate")
		l.FlushMicrotasks()
	},
	"async_nested": func(l *Loop, log func(string)) {
		inner := func() *Future[int] {
			return Async(func() int {
				log("inner start")
				awaitNull()
				log("inner after await null")
				return 3
			})
		}
		outer := func() *Future[Void] {
			return AsyncVoid(func() {
				log("outer start")
				v := inner().Await()
				log(fmt.Sprint("outer got ", v))
				FutureDelayed[Void](10*ms, nil).Await()
				log("outer after delay")
			})
		}
		outer().OnComplete(func(Void) { log("outer done") }, nil)
		log("sync end")
		l.FlushMicrotasks()
		log("flushed")
		l.Elapse(9 * ms)
		log("elapsed 9")
		l.Elapse(1 * ms)
		log("elapsed 10")
	},
	"errors": func(l *Loop, log func(string)) {
		caught := func(e error) int { log("caught " + msg(e)); return 0 }
		Async(func() int { panic(stateError{"early"}) }).CatchError(caught, nil)
		Async(func() int { FutureValue(Void{}).Await(); panic(stateError{"late"}) }).CatchError(caught, nil)
		Then(Then(Then(FutureError[int](stateError{"passes"}), func(v int) int { return v + 1 }), func(v int) int { return v }).
			CatchError(func(e error) int { log("caught after chain " + msg(e)); return -1 }, nil),
			func(v int) Void { log(fmt.Sprint("recovered ", v)); return Void{} })
		Then(FutureValue(1), func(int) int { panic(stateError{"in then"}) }).CatchError(caught, nil)
		FutureError[int](stateError{"filtered"}).
			CatchError(func(error) int { log("wrong handler"); return 0 }, func(error) bool { return false }).
			CatchError(func(e error) int { log("second handler " + msg(e)); return 0 }, nil)
		awaitsError := Async(func() int {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log("try caught " + msg(r.(error)))
					}
				}()
				FutureError[int](stateError{"awaited"}).Await()
			}()
			return 1
		})
		awaitsError.OnComplete(func(v int) { log(fmt.Sprint("awaitsError ", v)) }, nil)
		log("sync end")
		l.FlushMicrotasks()
	},
	"timers": func(l *Loop, log func(string)) {
		at := func() int64 { return l.Elapsed().Milliseconds() }
		NewTimer(20*ms, func() { log(fmt.Sprint("t20 a ", at())) })
		NewTimer(10*ms, func() {
			log(fmt.Sprint("t10 ", at()))
			ScheduleMicrotask(func() { log("micro in t10") })
			NewTimer(10*ms, func() { log(fmt.Sprint("t10+10 ", at())) })
		})
		NewTimer(20*ms, func() { log(fmt.Sprint("t20 b ", at())) })
		cancelled := NewTimer(15*ms, func() { log("never") })
		NewPeriodicTimer(7*ms, func(t *Timer) {
			log(fmt.Sprint("periodic tick ", t.Tick(), " at ", at()))
			if t.Tick() >= 4 {
				t.Cancel()
			}
		})
		cancelled.Cancel()
		l.Elapse(5 * ms)
		log(fmt.Sprint("at 5 active=", cancelled.IsActive()))
		l.Elapse(25 * ms)
		log(fmt.Sprint("at ", at(), " pending=", len(l.PendingTimers())))
	},
	"periodic_catch_up": func(l *Loop, log func(string)) {
		p := NewPeriodicTimer(10*ms, func(t *Timer) {
			log(fmt.Sprint("tick ", t.Tick(), " at ", l.Elapsed().Milliseconds()))
			if t.Tick() == 1 {
				l.ElapseBlocking(35 * ms)
			}
			if t.Tick() > 6 {
				t.Cancel()
			}
		})
		l.Elapse(100 * ms)
		log(fmt.Sprint("active=", p.IsActive()))
	},
	"flush_timers": func(l *Loop, log func(string)) {
		NewTimer(2*time.Second, func() {
			log(fmt.Sprint("t2 at ", l.Elapsed().Milliseconds()))
			NewTimer(time.Second, func() { log(fmt.Sprint("t3 at ", l.Elapsed().Milliseconds())) })
		})
		l.FlushTimers(time.Hour, true)
		log(fmt.Sprint("done at ", l.Elapsed().Milliseconds()))
	},
}

// awaitNull is Dart's `await null`: one microtask hop.
func awaitNull() { FutureMicrotask(func() Void { return Void{} }).Await() }

func TestAsyncMatchesDart(t *testing.T) {
	data, err := os.ReadFile("flutter/build/async.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string][]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != len(asyncScenes) {
		t.Fatalf("reference has %d scenes, Go has %d", len(ref), len(asyncScenes))
	}
	for name, scene := range asyncScenes {
		l := NewLoop()
		restore := SetCurrent(l)
		var got []string
		scene(l, func(s string) { got = append(got, s) })
		l.FlushMicrotasks()
		restore()
		if !reflect.DeepEqual(got, ref[name]) {
			t.Errorf("%s:\n Go:   %q\n Dart: %q", name, got, ref[name])
		}
	}
}
