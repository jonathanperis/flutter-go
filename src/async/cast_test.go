package async

import "testing"

func TestCastFuture(t *testing.T) {
	loop := NewLoop()
	defer SetCurrent(loop)()
	var got []any
	f := FutureValue[any](5)
	OnComplete(CastFuture[Void](f), func(Void) { got = append(got, "void") }, func(err error) { got = append(got, err) })
	OnComplete(CastFuture[int](f), func(v int) { got = append(got, v) }, func(err error) { got = append(got, err) })
	OnComplete(CastFuture[*string](FutureValue[any](nil)), func(v *string) { got = append(got, v == nil) }, nil)
	loop.FlushMicrotasks()
	if len(got) != 3 || got[0] != "void" || got[1] != 5 || got[2] != true {
		t.Fatalf("got %v", got)
	}
	// The cast is the same future: no extra event-loop turn.
	if _, ok := CastFuture[any](f).(*future[any]); !ok {
		t.Fatal("CastFuture[any] wrapped the future")
	}
}
