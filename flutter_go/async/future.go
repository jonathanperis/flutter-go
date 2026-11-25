package async

import (
	"errors"
	"time"
)

// Void is the value of a Future<void>.
type Void = struct{}

type futureState int

const (
	incomplete futureState = iota
	pendingComplete
	hasValue
	hasError
)

// Future is dart:async's Future<T>.
type Future[T any] struct {
	loop      *Loop
	state     futureState
	value     T
	err       error
	listeners []func()
	ignored   bool
}

func newFuture[T any]() *Future[T] { return &Future[T]{loop: current} }

func (f *Future[T]) isComplete() bool { return f.state == hasValue || f.state == hasError }

// addListener runs fn once f completes: synchronously during completion if
// f is still pending, in a microtask if it already completed (Dart's
// _addListener).
func (f *Future[T]) addListener(fn func()) {
	if f.isComplete() {
		f.loop.ScheduleMicrotask(fn)
		return
	}
	f.listeners = append(f.listeners, fn)
}

func (f *Future[T]) completeWithValue(v T) {
	f.value, f.state = v, hasValue
	f.propagate()
}

func (f *Future[T]) completeError(err error) {
	f.err, f.state = err, hasError
	f.propagate()
}

func (f *Future[T]) propagate() {
	ls := f.listeners
	f.listeners = nil
	if len(ls) == 0 && f.state == hasError && !f.ignored {
		f.loop.reportUncaught(f.err)
		return
	}
	for _, l := range ls {
		l()
	}
}

func (f *Future[T]) asyncCompleteWithValue(v T) {
	f.state = pendingComplete
	f.loop.ScheduleMicrotask(func() { f.completeWithValue(v) })
}

func (f *Future[T]) asyncCompleteError(err error) {
	f.state = pendingComplete
	f.loop.ScheduleMicrotask(func() { f.completeError(err) })
}

// chain completes f with src's result: synchronously if src is already
// complete and sync is set, otherwise when src completes.
func (f *Future[T]) chain(src *Future[T], sync bool) {
	f.state = pendingComplete
	copyResult := func() {
		if src.state == hasError {
			src.ignored = true
			f.completeError(src.err)
		} else {
			f.completeWithValue(src.value)
		}
	}
	switch {
	case src.isComplete() && sync:
		copyResult()
	case src.isComplete():
		f.loop.ScheduleMicrotask(copyResult)
	default:
		src.ignored = true
		src.listeners = append(src.listeners, copyResult)
	}
}

// ---- constructors ----

// FutureValue is Future.value(v): completes with v in a microtask.
func FutureValue[T any](v T) *Future[T] {
	f := newFuture[T]()
	f.asyncCompleteWithValue(v)
	return f
}

// FutureError is Future.error(err): fails with err in a microtask.
func FutureError[T any](err error) *Future[T] {
	f := newFuture[T]()
	f.asyncCompleteError(err)
	return f
}

// NewFuture is Future(computation): runs it in a zero-duration timer.
func NewFuture[T any](computation func() T) *Future[T] {
	f := newFuture[T]()
	TimerRun(func() { completeFrom(f, computation) })
	return f
}

// FutureMicrotask is Future.microtask(computation).
func FutureMicrotask[T any](computation func() T) *Future[T] {
	f := newFuture[T]()
	ScheduleMicrotask(func() { completeFrom(f, computation) })
	return f
}

// FutureDelayed is Future.delayed(d, computation); computation may be nil
// (the future then completes with the zero value).
func FutureDelayed[T any](d time.Duration, computation func() T) *Future[T] {
	f := newFuture[T]()
	NewTimer(d, func() {
		if computation == nil {
			var zero T
			f.completeWithValue(zero)
			return
		}
		completeFrom(f, computation)
	})
	return f
}

func completeFrom[T any](f *Future[T], computation func() T) {
	v, err := call(computation)
	if err != nil {
		f.completeError(err)
		return
	}
	f.completeWithValue(v)
}

func call[T any](fn func() T) (v T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = asError(r)
		}
	}()
	return fn(), nil
}

// ---- listening ----

// Then is future.then(onValue): a future of onValue's result, or of f's error.
func Then[T, R any](f *Future[T], onValue func(T) R) *Future[R] {
	return ThenOrError(f, onValue, nil)
}

// ThenOrError is future.then(onValue, onError: onError); onError may be nil.
func ThenOrError[T, R any](f *Future[T], onValue func(T) R, onError func(error) R) *Future[R] {
	result := newFuture[R]()
	f.addListener(func() {
		if f.state == hasError {
			if onError == nil {
				result.completeError(f.err)
				return
			}
			completeFrom(result, func() R { return onError(f.err) })
			return
		}
		completeFrom(result, func() R { return onValue(f.value) })
	})
	return result
}

// ThenFuture is future.then(onValue) where onValue returns a Future: the
// result completes when that future does.
func ThenFuture[T, R any](f *Future[T], onValue func(T) *Future[R]) *Future[R] {
	result := newFuture[R]()
	f.addListener(func() {
		if f.state == hasError {
			result.completeError(f.err)
			return
		}
		next, err := call(func() *Future[R] { return onValue(f.value) })
		if err != nil {
			result.completeError(err)
			return
		}
		result.chain(next, true)
	})
	return result
}

// OnComplete listens for f's value or error without creating a result
// future (then with void callbacks). onError may be nil, in which case an
// error is uncaught.
func (f *Future[T]) OnComplete(onValue func(T), onError func(error)) {
	f.addListener(func() {
		if f.state == hasError {
			if onError == nil {
				f.loop.reportUncaught(f.err)
				return
			}
			onError(f.err)
			return
		}
		onValue(f.value)
	})
}

// CatchError is future.catchError(onError): recovers from an error with
// onError's value. test, when non-nil, selects which errors to handle.
func (f *Future[T]) CatchError(onError func(error) T, test func(error) bool) *Future[T] {
	result := newFuture[T]()
	f.addListener(func() {
		if f.state == hasError && (test == nil || test(f.err)) {
			completeFrom(result, func() T { return onError(f.err) })
			return
		}
		if f.state == hasError {
			result.completeError(f.err)
			return
		}
		result.completeWithValue(f.value)
	})
	return result
}

// WhenComplete is future.whenComplete(action).
func (f *Future[T]) WhenComplete(action func()) *Future[T] {
	result := newFuture[T]()
	f.addListener(func() {
		if _, err := call(func() Void { action(); return Void{} }); err != nil {
			result.completeError(err)
			return
		}
		if f.state == hasError {
			result.completeError(f.err)
			return
		}
		result.completeWithValue(f.value)
	})
	return result
}

// Ignore marks f's error, if any, as handled (future.ignore()).
func (f *Future[T]) Ignore() { f.ignored = true }

// ---- Completer ----

// Completer is dart:async's Completer<T>.
type Completer[T any] struct {
	future *Future[T]
	sync   bool
}

// NewCompleter is Completer(): listeners hear the result in a microtask.
func NewCompleter[T any]() *Completer[T] { return &Completer[T]{future: newFuture[T]()} }

// NewSyncCompleter is Completer.sync(): listeners hear the result
// immediately.
func NewSyncCompleter[T any]() *Completer[T] {
	return &Completer[T]{future: newFuture[T](), sync: true}
}

func (c *Completer[T]) Future() *Future[T] { return c.future }

// IsCompleted reports whether Complete or CompleteError was called.
func (c *Completer[T]) IsCompleted() bool { return c.future.state != incomplete }

// ErrAlreadyCompleted is the StateError of completing twice.
var ErrAlreadyCompleted = errors.New("Bad state: Future already completed")

func (c *Completer[T]) Complete(v T) {
	if c.IsCompleted() {
		panic(ErrAlreadyCompleted)
	}
	if c.sync {
		c.future.completeWithValue(v)
	} else {
		c.future.asyncCompleteWithValue(v)
	}
}

// CompleteFuture completes with another future's result.
func (c *Completer[T]) CompleteFuture(f *Future[T]) {
	if c.IsCompleted() {
		panic(ErrAlreadyCompleted)
	}
	c.future.chain(f, c.sync)
}

func (c *Completer[T]) CompleteError(err error) {
	if c.IsCompleted() {
		panic(ErrAlreadyCompleted)
	}
	if c.sync {
		c.future.completeError(err)
	} else {
		c.future.asyncCompleteError(err)
	}
}
