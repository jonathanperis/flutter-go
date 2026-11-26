package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/src/async"
)

func dartList(v []int) string {
	s := "["
	for i, x := range v {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprint(x)
	}
	return s + "]"
}

// recoverMsg logs a recovered StateError as Dart's e.message.
func recoverMsg(log func(string), prefix string) {
	if r := recover(); r != nil {
		log(prefix + strings.TrimPrefix(msg(r.(error)), "Bad state: "))
	}
}

// Same scenes as flutter/test/stream_test.dart.
var streamScenes = map[string]func(l *Loop, log func(string)){
	"controller_basic": func(l *Loop, log func(string)) {
		c := NewStreamController[int](StreamControllerOptions{
			OnListen: func() { log("onListen") },
			OnPause:  func() { log("onPause") },
			OnResume: func() { log("onResume") },
			OnCancel: func() Future[Void] { log("onCancel"); return nil },
		})
		c.Add(1)
		c.Add(2)
		log(fmt.Sprintf("before listen hasListener=%v isPaused=%v", c.HasListener(), c.IsPaused()))
		c.Stream().Listen(func(v int) { log(fmt.Sprint("data ", v)) }, StreamListenOptions{OnDone: func() { log("done") }})
		log(fmt.Sprintf("after listen hasListener=%v isPaused=%v", c.HasListener(), c.IsPaused()))
		ScheduleMicrotask(func() { log("micro1") })
		c.Add(3)
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		OnComplete(c.Done(), func(Void) { log("done future") }, nil)
		ScheduleMicrotask(func() { log("micro2") })
		log(fmt.Sprintf("sync end isClosed=%v", c.IsClosed()))
		func() {
			defer recoverMsg(log, "add after close: ")
			c.Add(4)
		}()
		func() {
			defer recoverMsg(log, "second listen: ")
			c.Stream().Listen(nil)
		}()
	},
	"pause_resume": func(l *Loop, log func(string)) {
		c := NewStreamController[int](StreamControllerOptions{
			OnPause:  func() { log("onPause") },
			OnResume: func() { log("onResume") },
		})
		var sub StreamSubscription[int]
		sub = c.Stream().Listen(func(v int) {
			log(fmt.Sprint("data ", v))
			if v == 1 {
				sub.Pause(nil)
				log(fmt.Sprintf("paused isPaused=%v controller.isPaused=%v", sub.IsPaused(), c.IsPaused()))
				NewTimer(5*ms, func() {
					log("resuming")
					sub.Resume()
				})
			}
			if v == 3 {
				sub.Pause(FutureDelayed[Void](3*ms, nil))
				sub.Pause(nil)
				log("double paused")
				NewTimer(1*ms, func() {
					log("resume once")
					sub.Resume()
				})
			}
		}, StreamListenOptions{OnDone: func() { log("done") }})
		c.Add(1)
		c.Add(2)
		c.Add(3)
		c.Add(4)
		c.Close()
		ScheduleMicrotask(func() { log("micro") })
		l.Elapse(20 * ms)
		log("end")
	},
	"cancel": func(l *Loop, log func(string)) {
		c := NewStreamController[int](StreamControllerOptions{
			OnCancel: func() Future[Void] {
				log("onCancel")
				return FutureDelayed(2*ms, func() Void { log("onCancel future"); return Void{} })
			},
		})
		var sub StreamSubscription[int]
		sub = c.Stream().Listen(func(v int) {
			log(fmt.Sprint("data ", v))
			if v == 2 {
				OnComplete(sub.Cancel(), func(Void) { log("cancel future") }, nil)
				log(fmt.Sprintf("canceled hasListener=%v", c.HasListener()))
			}
		}, StreamListenOptions{OnDone: func() { log("done (never)") }})
		c.Add(1)
		c.Add(2)
		c.Add(3)
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		l.Elapse(5 * ms)
		func() {
			defer recoverMsg(log, "add after close: ")
			c.Add(4)
		}()
		log(fmt.Sprintf("end isClosed=%v", c.IsClosed()))
	},
	"errors": func(l *Loop, log func(string)) {
		a := NewStreamController[int]()
		a.Stream().Listen(func(v int) { log(fmt.Sprint("a data ", v)) }, StreamListenOptions{
			OnError: func(e error) { log("a error " + msg(e)) },
			OnDone:  func() { log("a done") },
		})
		a.Add(1)
		a.AddError(stateError{"e1"})
		a.Add(2)
		a.Close()
		b := NewStreamController[int](StreamControllerOptions{
			OnCancel: func() Future[Void] {
				return FutureDelayed(1*ms, func() Void { log("b onCancel future"); return Void{} })
			},
		})
		b.Stream().Listen(func(v int) { log(fmt.Sprint("b data ", v)) }, StreamListenOptions{
			OnError:       func(e error) { log("b error " + msg(e)) },
			OnDone:        func() { log("b done (never)") },
			CancelOnError: true,
		})
		b.Add(1)
		b.AddError(stateError{"e2"})
		b.Add(2)
		ScheduleMicrotask(func() { log("micro") })
		l.Elapse(5 * ms)
		log("end")
	},
	"from_iterable": func(l *Loop, log func(string)) {
		var sub StreamSubscription[int]
		sub = StreamFromIterable(slices.Values([]int{1, 2, 3, 4})).Listen(func(v int) {
			log(fmt.Sprint("data ", v))
			if v == 2 {
				sub.Pause(nil)
				NewTimer(1*ms, func() {
					log("resume")
					sub.Resume()
				})
			}
		}, StreamListenOptions{OnDone: func() { log("done") }})
		ScheduleMicrotask(func() { log("micro1") })
		ScheduleMicrotask(func() { ScheduleMicrotask(func() { log("micro2") }) })
		log("sync end")
		l.Elapse(2 * ms)
		StreamFromIterable(slices.Values([]int{5, 6})).Listen(func(v int) { log(fmt.Sprint("data ", v)) }).Cancel()
		StreamFromIterable(slices.Values([]int{})).Listen(nil, StreamListenOptions{OnDone: func() { log("empty iterable done") }})
		log("end")
	},
	"empty": func(l *Loop, log func(string)) {
		StreamEmpty[int]().Listen(nil, StreamListenOptions{OnDone: func() { log("empty done") }})
		paused := StreamEmpty[int]().Listen(nil, StreamListenOptions{OnDone: func() { log("paused empty done") }})
		paused.Pause(nil)
		log(fmt.Sprintf("paused isPaused=%v", paused.IsPaused()))
		ScheduleMicrotask(func() { log("micro") })
		NewTimer(1*ms, func() {
			log("resume")
			paused.Resume()
		})
		StreamEmpty[int]().Listen(nil, StreamListenOptions{OnDone: func() { log("never") }}).Cancel()
		log("sync end")
		l.Elapse(2 * ms)
	},
	"add_stream": func(l *Loop, log func(string)) {
		c := NewStreamController[int](StreamControllerOptions{
			OnListen: func() { log("onListen") },
			OnCancel: func() Future[Void] { log("onCancel"); return nil },
		})
		OnComplete(c.AddStream(StreamFromIterable(slices.Values([]int{1, 2})), false), func(Void) {
			log("addStream 1 done")
			c.Add(3)
			closed := ThenFuture(Then(c.AddStream(StreamFromIterable(slices.Values([]int{4, 5})), false),
				func(Void) Void { log("addStream 2 done"); return Void{} }),
				func(Void) Future[Void] { return c.Close() })
			Then(closed, func(Void) Void { log("closed"); return Void{} })
			func() {
				defer recoverMsg(log, "add while adding: ")
				c.Add(99)
			}()
		}, nil)
		log("before listen")
		ScheduleMicrotask(func() { log("micro") })
		c.Stream().Listen(func(v int) { log(fmt.Sprint("data ", v)) }, StreamListenOptions{OnDone: func() { log("done") }})
		log("sync end")
		l.Elapse(1 * ms)
		errorsEnd := NewStreamController[int]()
		errorsEnd.Stream().Listen(func(v int) { log(fmt.Sprint("ee data ", v)) }, StreamListenOptions{
			OnError: func(e error) { log("ee error " + msg(e)) },
		})
		source := NewStreamController[int]()
		OnComplete(errorsEnd.AddStream(source.Stream(), true), func(Void) { log("ee addStream done") }, nil)
		source.Add(7)
		source.AddError(stateError{"stop"})
		source.Add(8)
		l.Elapse(1 * ms)
		errorsEnd.Add(9)
		l.FlushMicrotasks()
		log("end")
	},
	"pause_while_scheduled": func(l *Loop, log func(string)) {
		c := NewStreamController[int]()
		sub := c.Stream().Listen(func(v int) { log(fmt.Sprint("data ", v)) })
		c.Add(1)
		c.Add(2)
		sub.Pause(nil)
		ScheduleMicrotask(func() { log("micro") })
		NewTimer(1*ms, func() {
			log("resume")
			sub.Resume()
		})
		l.Elapse(2 * ms)
		log("end")
	},
	"add_stream_late_listen": func(l *Loop, log func(string)) {
		c := NewStreamController[int]()
		OnComplete(c.AddStream(StreamFromIterable(slices.Values([]int{1, 2, 3})), false),
			func(Void) { log("addStream done") }, nil)
		ScheduleMicrotask(func() { log("micro") })
		NewTimer(1*ms, func() {
			log("listen")
			c.Stream().Listen(func(v int) { log(fmt.Sprint("data ", v)) })
			ScheduleMicrotask(func() { log("micro after listen") })
			ScheduleMicrotask(func() { ScheduleMicrotask(func() { log("micro 2 after listen") }) })
		})
		l.Elapse(2 * ms)
		log("end")
	},
	"to_list_and_from_future": func(l *Loop, log func(string)) {
		OnComplete(StreamToList(StreamFromIterable(slices.Values([]int{1, 2, 3}))),
			func(v []int) { log("list " + dartList(v)) }, nil)
		c := NewStreamController[int]()
		OnComplete(StreamToList(c.Stream()), func(v []int) { log("controller list " + dartList(v)) }, nil)
		c.Add(7)
		c.Close()
		StreamFromFuture(FutureValue(5)).Listen(func(v int) { log(fmt.Sprint("fromFuture data ", v)) },
			StreamListenOptions{OnDone: func() { log("fromFuture done") }})
		StreamFromFuture(FutureError[int](stateError{"bad"})).Listen(func(int) { log("never") },
			StreamListenOptions{
				OnError: func(e error) { log("fromFuture error " + msg(e)) },
				OnDone:  func() { log("fromFuture error done") },
			})
		failing := NewStreamController[int]()
		OnComplete(StreamToList(failing.Stream()), func(v []int) { log("never " + dartList(v)) },
			func(e error) { log("toList error " + msg(e)) })
		failing.Add(1)
		failing.AddError(stateError{"x"})
		ScheduleMicrotask(func() { log("micro") })
		log("sync end")
	},
}

func TestStreamMatchesDart(t *testing.T) {
	data, err := os.ReadFile("flutter/build/stream.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string][]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != len(streamScenes) {
		t.Fatalf("reference has %d scenes, Go has %d", len(ref), len(streamScenes))
	}
	for name, scene := range streamScenes {
		l := NewLoop()
		restore := SetCurrent(l)
		var got []string
		scene(l, func(s string) { got = append(got, s) })
		l.FlushMicrotasks()
		l.FlushTimers(time.Hour, true)
		restore()
		if !reflect.DeepEqual(got, ref[name]) {
			t.Errorf("%s:\n Go:   %q\n Dart: %q", name, got, ref[name])
		}
	}
}
