package async

import (
	"fmt"
	"reflect"
	"testing"
)

// No Dart SDK tests ship with Flutter's SDK; these check the rules of
// broadcast_stream_controller.dart directly (the difftest in src/difftest
// compares whole scenes with the Dart VM).

func runLoop(t *testing.T, body func(l *Loop, log func(string))) []string {
	t.Helper()
	l := NewLoop()
	restore := SetCurrent(l)
	defer restore()
	var got []string
	l.UncaughtError = func(err error) { got = append(got, "uncaught "+err.Error()) }
	body(l, func(s string) { got = append(got, s) })
	l.FlushMicrotasks()
	return got
}

func expectLog(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestBroadcastIsBroadcastAndNeverPaused(t *testing.T) {
	runLoop(t, func(l *Loop, log func(string)) {
		c := NewStreamControllerBroadcast[int]()
		s, ok := c.Stream().(interface{ IsBroadcast() bool })
		if !ok || !s.IsBroadcast() {
			t.Error("stream is not broadcast")
		}
		sub := c.Stream().Listen(nil)
		sub.Pause(nil)
		if c.IsPaused() || !sub.IsPaused() {
			t.Error("controller paused or subscription not paused")
		}
	})
}

func TestBroadcastEventsGoToListenersPresentWhenAdded(t *testing.T) {
	got := runLoop(t, func(l *Loop, log func(string)) {
		c := NewStreamControllerBroadcast[int]()
		c.Stream().Listen(func(v int) { log(fmt.Sprint("a ", v)) })
		c.Add(1)
		c.Stream().Listen(func(v int) { log(fmt.Sprint("b ", v)) })
		c.Add(2)
		log("sync")
	})
	expectLog(t, got, "sync", "a 1", "b 2", "a 2")
}

func TestBroadcastOnListenOnCancelPerFirstAndLastListener(t *testing.T) {
	got := runLoop(t, func(l *Loop, log func(string)) {
		c := NewStreamControllerBroadcast[int](StreamControllerBroadcastOptions{
			OnListen: func() { log("listen") },
			OnCancel: func() { log("cancel") },
		})
		a := c.Stream().Listen(nil)
		b := c.Stream().Listen(nil)
		a.Cancel()
		log("a gone")
		b.Cancel()
		log("b gone")
		b.Cancel() // already removed: no second onCancel
		c.Stream().Listen(nil).Cancel()
	})
	expectLog(t, got, "listen", "a gone", "cancel", "b gone", "listen", "cancel")
}

func TestBroadcastSyncDefersRemovalWhileFiring(t *testing.T) {
	got := runLoop(t, func(l *Loop, log func(string)) {
		c := NewStreamControllerBroadcast[int](StreamControllerBroadcastOptions{
			Sync:     true,
			OnCancel: func() { log("cancel") },
		})
		var a, b StreamSubscription[int]
		a = c.Stream().Listen(func(v int) {
			log(fmt.Sprint("a ", v))
			a.Cancel()
			b.Cancel()
			log(fmt.Sprintf("in a hasListener=%v", c.HasListener()))
		})
		b = c.Stream().Listen(func(v int) { log(fmt.Sprint("b ", v)) })
		c.Add(1)
		log(fmt.Sprintf("after hasListener=%v", c.HasListener()))
	})
	expectLog(t, got, "a 1", "in a hasListener=true", "cancel", "after hasListener=false")
}

func TestBroadcastSyncSingleListenerCancelCallsOnCancelAfterEvent(t *testing.T) {
	got := runLoop(t, func(l *Loop, log func(string)) {
		c := NewStreamControllerBroadcast[int](StreamControllerBroadcastOptions{
			Sync:     true,
			OnCancel: func() { log("cancel") },
		})
		var a StreamSubscription[int]
		a = c.Stream().Listen(func(v int) {
			a.Cancel()
			log("canceled in data")
		})
		c.Add(1)
	})
	expectLog(t, got, "canceled in data", "cancel")
}

func TestBroadcastErrorsAndDoneFuture(t *testing.T) {
	got := runLoop(t, func(l *Loop, log func(string)) {
		c := NewStreamControllerBroadcast[int]()
		c.AddError(fmt.Errorf("nobody")) // dropped
		c.Stream().Listen(nil, StreamListenOptions{OnError: func(e error) { log("error " + e.Error()) },
			OnDone: func() { log("done") }})
		c.AddError(fmt.Errorf("x"))
		OnComplete(c.Close(), func(Void) { log("closed") }, nil)
		if c.Close() != c.Done() {
			t.Error("second close is not the done future")
		}
		func() {
			defer func() { log(fmt.Sprint(recover())) }()
			c.Add(1)
		}()
		c.Stream().Listen(nil, StreamListenOptions{OnDone: func() { log("late done") }})
	})
	expectLog(t, got, "Bad state: Cannot add new events after calling close",
		"error x", "late done", "done", "closed")
}

func TestBroadcastAddStreamBlocksEvents(t *testing.T) {
	got := runLoop(t, func(l *Loop, log func(string)) {
		c := NewStreamControllerBroadcast[int]()
		c.Stream().Listen(func(v int) { log(fmt.Sprint("v ", v)) })
		src := NewStreamController[int]()
		OnComplete(c.AddStream(src.Stream(), false), func(Void) { log("addStream done") }, nil)
		func() {
			defer func() { log(fmt.Sprint(recover())) }()
			c.Add(9)
		}()
		src.Add(1)
		src.Close()
		l.FlushMicrotasks()
		c.Add(2)
	})
	expectLog(t, got, "Bad state: Cannot add new events while doing an addStream",
		"v 1", "addStream done", "v 2")
}
