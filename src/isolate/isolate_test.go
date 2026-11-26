package isolate

import (
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
)

func TestReceivePortDeliversOnItsLoopInOrder(t *testing.T) {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	r := NewReceivePort()
	r.SendPort().Send(1) // buffered until Listen
	var got []any
	done := false
	r.Listen(func(m any) { got = append(got, m) }, func() { done = true })
	finished := make(chan struct{})
	go func() {
		for i := 2; i <= 5; i++ {
			r.SendPort().Send(i)
		}
		close(finished)
	}()
	<-finished
	if len(got) != 0 {
		t.Fatal("delivered synchronously")
	}
	for len(got) < 5 {
		loop.WaitPosted()
		loop.RunPosted()
	}
	for i, m := range got {
		if m != i+1 {
			t.Fatalf("order %v", got)
		}
	}
	r.Close()
	r.SendPort().Send(6)
	r.Close()
	loop.RunPosted()
	if !done || len(got) != 5 {
		t.Fatalf("after close: %v, done %v", got, done)
	}
	if r.SendPort() != r.SendPort() || r.SendPort() == NewReceivePort().SendPort() {
		t.Fatal("equality")
	}
}
