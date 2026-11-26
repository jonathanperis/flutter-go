package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/src/async"
)

func broadcastController(log func(string), sync bool) *BroadcastStreamController[int] {
	return NewStreamControllerBroadcast[int](StreamControllerBroadcastOptions{
		OnListen: func() { log("onListen") },
		OnCancel: func() { log("onCancel") },
		Sync:     sync,
	})
}

func broadcastBasic(l *Loop, log func(string), sync bool) {
	c := broadcastController(log, sync)
	c.Add(0) // no listener: dropped
	c.AddError(stateError{"dropped"})
	log(fmt.Sprintf("hasListener=%v isPaused=%v", c.HasListener(), c.IsPaused()))
	c.Stream().Listen(func(v int) { log(fmt.Sprint("a ", v)) }, StreamListenOptions{OnDone: func() { log("a done") }})
	log(fmt.Sprintf("after a hasListener=%v", c.HasListener()))
	c.Add(1)
	c.Stream().Listen(func(v int) { log(fmt.Sprint("b ", v)) }, StreamListenOptions{OnDone: func() { log("b done") }})
	c.Add(2)
	ScheduleMicrotask(func() { log("micro1") })
	c.AddError(stateError{"e"})
	OnComplete(c.Close(), func(Void) { log("close future") }, nil)
	OnComplete(c.Done(), func(Void) { log("done future") }, nil)
	ScheduleMicrotask(func() { log("micro2") })
	log(fmt.Sprintf("sync end isClosed=%v", c.IsClosed()))
	func() {
		defer recoverMsg(log, "add after close: ")
		c.Add(4)
	}()
	func() {
		defer recoverMsg(log, "addError after close: ")
		c.AddError(stateError{"late"})
	}()
	c.Stream().Listen(func(v int) { log(fmt.Sprint("late ", v)) }, StreamListenOptions{OnDone: func() { log("late done") }})
	log(fmt.Sprintf("listened after close hasListener=%v", c.HasListener()))
}

func broadcastCancelInData(l *Loop, log func(string), sync bool) {
	c := broadcastController(log, sync)
	var a, b StreamSubscription[int]
	a = c.Stream().Listen(func(v int) {
		log(fmt.Sprint("a ", v))
		if v == 2 {
			a.Cancel()
			log(fmt.Sprintf("a canceled hasListener=%v", c.HasListener()))
		}
	})
	b = c.Stream().Listen(func(v int) {
		log(fmt.Sprint("b ", v))
		if v == 3 {
			b.Cancel()
			log(fmt.Sprintf("b canceled hasListener=%v", c.HasListener()))
			c.Stream().Listen(func(v int) { log(fmt.Sprint("c ", v)) })
			log("c added")
		}
	})
	c.Add(1)
	c.Add(2)
	c.Add(3)
	c.Add(4)
	ScheduleMicrotask(func() { log("micro") })
	l.FlushMicrotasks()
	c.Add(5)
	l.FlushMicrotasks()
	log("end")
}

func broadcastAddDuringFiring(l *Loop, log func(string), sync bool) {
	c := broadcastController(log, sync)
	c.Stream().Listen(func(v int) {
		log(fmt.Sprint("a ", v))
		if v == 1 {
			func() {
				defer recoverMsg(log, "add during firing: ")
				c.Add(10)
				log("added 10")
			}()
			c.Stream().Listen(func(v int) { log(fmt.Sprint("new ", v)) })
		}
	})
	c.Stream().Listen(func(v int) { log(fmt.Sprint("b ", v)) })
	c.Add(1)
	c.Add(2)
	l.FlushMicrotasks()
	log("end")
}

// Same scenes as flutter/test/broadcast_stream_test.dart.
var broadcastStreamScenes = map[string]func(l *Loop, log func(string)){
	"basic_async":             func(l *Loop, log func(string)) { broadcastBasic(l, log, false) },
	"basic_sync":              func(l *Loop, log func(string)) { broadcastBasic(l, log, true) },
	"cancel_in_data_async":    func(l *Loop, log func(string)) { broadcastCancelInData(l, log, false) },
	"cancel_in_data_sync":     func(l *Loop, log func(string)) { broadcastCancelInData(l, log, true) },
	"add_during_firing_async": func(l *Loop, log func(string)) { broadcastAddDuringFiring(l, log, false) },
	"add_during_firing_sync":  func(l *Loop, log func(string)) { broadcastAddDuringFiring(l, log, true) },
	"add_during_firing_sync_single": func(l *Loop, log func(string)) {
		c := broadcastController(log, true)
		c.Stream().Listen(func(v int) {
			log(fmt.Sprint("a ", v))
			if v == 1 {
				func() {
					defer recoverMsg(log, "add during firing: ")
					c.Add(10)
					log("added 10")
				}()
				func() {
					defer recoverMsg(log, "close during firing: ")
					c.Close()
				}()
			}
		})
		c.Add(1)
		c.Add(2)
		log("end")
	},
	"relisten": func(l *Loop, log func(string)) {
		c := broadcastController(log, false)
		s1 := c.Stream().Listen(func(v int) { log(fmt.Sprint("s1 ", v)) })
		s2 := c.Stream().Listen(func(v int) { log(fmt.Sprint("s2 ", v)) })
		c.Add(1)
		s1.Cancel()
		log(fmt.Sprintf("s1 canceled hasListener=%v", c.HasListener()))
		s2.Cancel()
		log(fmt.Sprintf("s2 canceled hasListener=%v", c.HasListener()))
		s2.Cancel()
		log("s2 canceled twice")
		c.Add(2)
		l.FlushMicrotasks()
		c.Stream().Listen(func(v int) { log(fmt.Sprint("s3 ", v)) }, StreamListenOptions{OnDone: func() { log("s3 done") }})
		c.Add(3)
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		l.FlushMicrotasks()
		log("end")
	},
	"pause_one_listener": func(l *Loop, log func(string)) {
		c := broadcastController(log, false)
		var a StreamSubscription[int]
		a = c.Stream().Listen(func(v int) {
			log(fmt.Sprint("a ", v))
			if v == 1 {
				a.Pause(nil)
				log(fmt.Sprintf("a paused isPaused=%v controller.isPaused=%v", a.IsPaused(), c.IsPaused()))
				NewTimer(3*ms, func() {
					log("resume a")
					a.Resume()
				})
			}
		}, StreamListenOptions{OnDone: func() { log("a done") }})
		c.Stream().Listen(func(v int) { log(fmt.Sprint("b ", v)) }, StreamListenOptions{OnDone: func() { log("b done") }})
		c.Add(1)
		c.Add(2)
		c.Add(3)
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		l.Elapse(1 * ms)
		log("after 1ms")
		l.Elapse(5 * ms)
		log("end")
	},
	"pause_with_resume_signal": func(l *Loop, log func(string)) {
		c := broadcastController(log, false)
		a := c.Stream().Listen(func(v int) { log(fmt.Sprint("a ", v)) })
		a.Pause(FutureDelayed[Void](2*ms, nil))
		c.Add(1)
		c.Add(2)
		l.FlushMicrotasks()
		log("paused flushed")
		l.Elapse(3 * ms)
		a.Cancel()
		log("end")
	},
	"close_without_listeners": func(l *Loop, log func(string)) {
		c := broadcastController(log, false)
		OnComplete(c.Done(), func(Void) { log("done future") }, nil)
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		log("closed")
		again := c.Close()
		log(fmt.Sprintf("close again identical=%v", again == c.Done()))
		ScheduleMicrotask(func() { log("micro") })
		c.Stream().Listen(func(v int) { log(fmt.Sprint("never ", v)) }, StreamListenOptions{OnDone: func() { log("late done") }})
		l.FlushMicrotasks()
		log("end")
	},
	"errors": func(l *Loop, log func(string)) {
		c := broadcastController(log, false)
		c.Stream().Listen(func(v int) { log(fmt.Sprint("a ", v)) }, StreamListenOptions{
			OnError: func(e error) { log("a error " + msg(e)) },
			OnDone:  func() { log("a done") },
		})
		c.Stream().Listen(func(v int) { log(fmt.Sprint("b ", v)) }, StreamListenOptions{
			OnError:       func(e error) { log("b error " + msg(e)) },
			OnDone:        func() { log("b done") },
			CancelOnError: true,
		})
		c.Add(1)
		c.AddError(stateError{"boom"})
		c.Add(2)
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		l.FlushMicrotasks()
		log("end")
	},
	"to_list": func(l *Loop, log func(string)) {
		c := broadcastController(log, false)
		OnComplete(StreamToList(c.Stream()), func(v []int) { log("list1 " + dartList(v)) }, nil)
		c.Add(1)
		OnComplete(StreamToList(c.Stream()), func(v []int) { log("list2 " + dartList(v)) }, nil)
		c.Add(2)
		c.Close()
		f := broadcastController(log, false)
		OnComplete(StreamToList(f.Stream()), func(v []int) { log("never " + dartList(v)) },
			func(e error) { log("toList error " + msg(e)) })
		f.Add(1)
		f.AddError(stateError{"x"})
		f.Add(2)
		log("sync end")
	},
	"add_stream": func(l *Loop, log func(string)) {
		c := broadcastController(log, false)
		c.Stream().Listen(func(v int) { log(fmt.Sprint("a ", v)) }, StreamListenOptions{
			OnError: func(e error) { log("a error " + msg(e)) },
		})
		OnComplete(c.AddStream(StreamFromIterable(slices.Values([]int{1, 2, 3})), false),
			func(Void) { log("addStream done") }, nil)
		func() {
			defer recoverMsg(log, "add during addStream: ")
			c.Add(9)
		}()
		func() {
			defer recoverMsg(log, "close during addStream: ")
			c.Close()
		}()
		l.FlushMicrotasks()
		src := NewStreamController[int]()
		OnComplete(c.AddStream(src.Stream(), true), func(Void) { log("addStream2 done") }, nil)
		src.Add(4)
		src.AddError(stateError{"stop"})
		src.Add(5)
		l.FlushMicrotasks()
		c.Add(6)
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		l.FlushMicrotasks()
		log("end")
	},
	"cancel_on_done_sync_onCancel_order": func(l *Loop, log func(string)) {
		c := broadcastController(log, true)
		c.Stream().Listen(func(v int) { log(fmt.Sprint("a ", v)) }, StreamListenOptions{OnDone: func() { log("a done") }})
		c.Stream().Listen(func(v int) { log(fmt.Sprint("b ", v)) }, StreamListenOptions{OnDone: func() { log("b done") }})
		OnComplete(c.Done(), func(Void) { log("done future") }, nil)
		c.Add(1)
		log("after add")
		OnComplete(c.Close(), func(Void) { log("close future") }, nil)
		log(fmt.Sprintf("after close hasListener=%v", c.HasListener()))
		l.FlushMicrotasks()
		log("end")
	},
}

func TestBroadcastStreamMatchesDart(t *testing.T) {
	data, err := os.ReadFile("flutter/build/broadcast_stream.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string][]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != len(broadcastStreamScenes) {
		t.Fatalf("reference has %d scenes, Go has %d", len(ref), len(broadcastStreamScenes))
	}
	for name, scene := range broadcastStreamScenes {
		l := NewLoop()
		restore := SetCurrent(l)
		var got []string
		l.UncaughtError = func(err error) { got = append(got, "uncaught "+msg(err)) }
		scene(l, func(s string) { got = append(got, s) })
		l.FlushMicrotasks()
		l.FlushTimers(time.Hour, true)
		restore()
		if !reflect.DeepEqual(got, ref[name]) {
			t.Errorf("%s:\n Go:   %q\n Dart: %q", name, got, ref[name])
		}
	}
}
