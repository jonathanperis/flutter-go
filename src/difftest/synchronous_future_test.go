package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// Same scenes as flutter/test/synchronous_future_test.dart.
var synchronousFutureScenes = map[string]func(l *Loop, log func(string)){
	"then_sync": func(l *Loop, log func(string)) {
		f := foundation.NewSynchronousFuture(1)
		log("before then")
		Then(Then(f, func(v int) int { log(fmt.Sprint("then ", v)); return v + 1 }),
			func(v int) Void { log(fmt.Sprint("chained ", v)); return Void{} })
		log("after then")
		ScheduleMicrotask(func() { log("micro") })
		Then(f, func(v int) Void { log(fmt.Sprint("again ", v)); return Void{} })
		log("sync end")
	},
	"then_returns_future": func(l *Loop, log func(string)) {
		Then(ThenFuture(foundation.NewSynchronousFuture(1), func(v int) Future[int] { return FutureValue(v * 10) }),
			func(v int) Void { log(fmt.Sprint("core result ", v)); return Void{} })
		Then(ThenFuture(foundation.NewSynchronousFuture(2), func(v int) Future[int] { return foundation.NewSynchronousFuture(v * 10) }),
			func(v int) Void { log(fmt.Sprint("sync result ", v)); return Void{} })
		c := NewCompleter[int]()
		Then(ThenFuture(foundation.NewSynchronousFuture(3), func(int) Future[int] { return c.Future() }),
			func(v int) Void { log(fmt.Sprint("completer result ", v)); return Void{} })
		ScheduleMicrotask(func() { log("micro") })
		c.Complete(30)
		log("sync end")
	},
	"then_throws": func(l *Loop, log func(string)) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log("thrown synchronously " + msg(r.(error)))
				}
			}()
			Then(foundation.NewSynchronousFuture(1), func(int) int { panic(stateError{"boom"}) })
			log("not thrown")
		}()
	},
	"await": func(l *Loop, log func(string)) {
		f := func() Future[int] {
			return Async(func() int {
				log("f start")
				v := foundation.NewSynchronousFuture(5).Await()
				log(fmt.Sprint("f resumed ", v))
				w := foundation.NewSynchronousFuture(6).Await()
				log(fmt.Sprint("f resumed2 ", w))
				return v + w
			})
		}
		g := func() Future[int] {
			return Async(func() int {
				log("g start")
				foundation.NewSynchronousFuture(1).Await()
				log("g after sync await")
				FutureValue(Void{}).Await()
				log("g after core await")
				return 2
			})
		}
		h := func() Future[int] {
			return Async(func() int {
				foundation.NewSynchronousFuture(1).Await()
				panic(stateError{"after sync await"})
			})
		}
		k := func() Future[int] {
			return Async(func() int { panic(stateError{"before await"}) })
		}
		Then(f(), func(v int) Void { log(fmt.Sprint("f result ", v)); return Void{} })
		log("after f()")
		Then(g(), func(v int) Void { log(fmt.Sprint("g result ", v)); return Void{} })
		log("after g()")
		// Dart's runZonedGuarded: h fails before anyone listens.
		l.UncaughtError = func(err error) { log("uncaught " + msg(err)) }
		hf := h()
		l.UncaughtError = nil
		ScheduleMicrotask(func() { log("micro") })
		hf.CatchError(func(e error) int { log("h caught " + msg(e)); return 0 }, nil)
		k().CatchError(func(e error) int { log("k caught " + msg(e)); return 0 }, nil)
		log("sync end")
	},
	"completer_chains_foreign": func(l *Loop, log func(string)) {
		a, s := NewCompleter[int](), NewSyncCompleter[int]()
		OnComplete(a.Future(), func(v int) { log(fmt.Sprint("async completer ", v)) }, nil)
		OnComplete(s.Future(), func(v int) { log(fmt.Sprint("sync completer ", v)) }, nil)
		ScheduleMicrotask(func() { log("micro1") })
		a.CompleteFuture(foundation.NewSynchronousFuture(1))
		log(fmt.Sprint("after a.complete isCompleted=", a.IsCompleted()))
		s.CompleteFuture(foundation.NewSynchronousFuture(2))
		log("after s.complete")
		Then(ThenFuture(FutureValue(3), func(v int) Future[int] { return foundation.NewSynchronousFuture(v + 1) }),
			func(v int) Void { log(fmt.Sprint("core then foreign ", v)); return Void{} })
		ScheduleMicrotask(func() { log("micro2") })
		log("sync end")
	},
	"completer_chains_completed_core": func(l *Loop, log func(string)) {
		done := FutureValue(1)
		l.FlushMicrotasks()
		c := NewCompleter[int]()
		c.CompleteFuture(done)
		ScheduleMicrotask(func() { log("micro") })
		OnComplete(c.Future(), func(v int) { log(fmt.Sprint("completer ", v)) }, nil)
		d := NewCompleter[int]()
		OnComplete(d.Future(), func(v int) { log(fmt.Sprint("listened first ", v)) }, nil)
		d.CompleteFuture(done)
		ScheduleMicrotask(func() { log("micro2") })
		log("sync end")
	},
	"when_complete": func(l *Loop, log func(string)) {
		f := foundation.NewSynchronousFuture(4)
		w := f.WhenComplete(func() { log("action") })
		log(fmt.Sprint("same future ", w == Future[int](f)))
		Then(w, func(v int) Void { log(fmt.Sprint("after whenComplete ", v)); return Void{} })
		Then(f.WhenComplete(func() { panic(stateError{"action failed"}) }), func(v int) int { log(fmt.Sprint("never ", v)); return v }).
			CatchError(func(e error) int { log("caught " + msg(e)); return 0 }, nil)
		ScheduleMicrotask(func() { log("micro") })
		log("sync end")
	},
	"catch_error_never_completes": func(l *Loop, log func(string)) {
		Then(foundation.NewSynchronousFuture(1).CatchError(func(error) int { return 0 }, nil),
			func(v int) int { log(fmt.Sprint("never ", v)); return v }).
			WhenComplete(func() { log("never done") })
		l.Elapse(time.Second)
		log("end")
	},
	"timeout": func(l *Loop, log func(string)) {
		at := func() int64 { return l.Elapsed().Milliseconds() }
		Then(foundation.NewSynchronousFuture(7).Timeout(5*ms, nil),
			func(v int) Void { log(fmt.Sprint("sync timeout value ", v)); return Void{} })
		log("after sync timeout then")
		ScheduleMicrotask(func() { log("micro") })
		done := FutureValue(8)
		l.FlushMicrotasks()
		t := done.Timeout(5*ms, nil)
		ScheduleMicrotask(func() { log("micro2") })
		Then(t, func(v int) Void { log(fmt.Sprint("completed timeout ", v)); return Void{} })
		NewCompleter[int]().Future().Timeout(5*ms, nil).
			CatchError(func(e error) int { log(fmt.Sprint("timed out at ", at(), ": ", e)); return 0 }, nil)
		Then(NewCompleter[int]().Future().Timeout(3*ms, func() int { return 9 }),
			func(v int) Void { log(fmt.Sprint("onTimeout ", v, " at ", at())); return Void{} })
		late := NewCompleter[int]()
		Then(late.Future().Timeout(10*ms, nil),
			func(v int) Void { log(fmt.Sprint("in time ", v, " at ", at())); return Void{} })
		NewTimer(4*ms, func() { late.Complete(10) })
		tooLate := NewCompleter[int]()
		tl := tooLate.Future().Timeout(5*ms, func() int { return -1 })
		NewTimer(8*ms, func() { tooLate.Complete(11) })
		failed := FutureError[int](stateError{"failed"})
		failed.CatchError(func(error) int { return 0 }, nil)
		l.Elapse(20 * ms)
		Then(tl, func(v int) Void { log(fmt.Sprint("too late result ", v)); return Void{} })
		tf := failed.Timeout(5*ms, nil)
		ScheduleMicrotask(func() { log("micro3") })
		tf.CatchError(func(e error) int { log("failed timeout " + msg(e)); return 0 }, nil)
		l.FlushMicrotasks()
		c := NewCompleter[int]()
		c.CompleteFuture(failed)
		ScheduleMicrotask(func() { log("micro4") })
		c.Future().CatchError(func(e error) int { log("completer caught " + msg(e)); return 0 }, nil)
		log("end")
	},
	"ignore": func(l *Loop, log func(string)) {
		foundation.NewSynchronousFuture(1).Ignore()
		FutureError[int](stateError{"ignored"}).Ignore()
		log("end")
	},
}

func TestSynchronousFutureMatchesDart(t *testing.T) {
	data, err := os.ReadFile("flutter/build/synchronous_future.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string][]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != len(synchronousFutureScenes) {
		t.Fatalf("reference has %d scenes, Go has %d", len(ref), len(synchronousFutureScenes))
	}
	for name, scene := range synchronousFutureScenes {
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
