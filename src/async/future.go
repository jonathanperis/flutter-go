package async

import (
	"errors"
	"fmt"
	"time"
)

// Void is the value of a Future<void>.
type Void = struct{}

// Future is dart:async's Future<T>: an interface, as in Dart, so Go code can
// implement it (Flutter's SynchronousFuture and TickerFuture do).
//
// The method set is Dart's Future interface (minus asStream: streams are not
// ported) plus Dart's `await` and the `ignore` extension, because Go has no
// language-level await and no extension methods:
//
//   - ThenAny is Dart's then<R>. Go methods can't take type parameters, so R
//     is erased to any; call the generic functions Then, ThenOrError and
//     ThenFuture, which dispatch to it and restore R. A value returned by
//     onValue or onError that is a Future[any] is a FutureOr<R> holding a
//     future: the result chains to it (Dart's `result is Future<R>`).
//   - CatchError, WhenComplete and Timeout are Dart's methods of those names.
//   - Await is Dart's `await f` inside Async.
//   - Ignore is Dart's `f.ignore()`.
//
// Futures created by this package (FutureValue, Completer.Future, Async...)
// share one implementation, Dart's _Future. A user-defined future
// implements ThenAny, CatchError, WhenComplete and Timeout as its Dart class
// does, and gets Dart's behavior for the other two from AwaitUserDefined and
// IgnoreUserDefined.
type Future[T any] interface {
	ThenAny(onValue func(T) any, onError func(error) any) Future[any]
	CatchError(onError func(error) T, test func(error) bool) Future[T]
	WhenComplete(action func()) Future[T]
	Timeout(timeLimit time.Duration, onTimeout func() T) Future[T]
	Ignore()
	Await() T
}

type futureState int

const (
	incomplete futureState = iota
	pendingComplete
	hasValue
	hasError
)

// future is dart:async's _Future<T>, the implementation behind every future
// this package creates.
type future[T any] struct {
	loop      *Loop
	state     futureState
	value     T
	err       error
	listeners []func()
	ignored   bool
}

func newFuture[T any]() *future[T] { return &future[T]{loop: current} }

func (f *future[T]) isComplete() bool { return f.state == hasValue || f.state == hasError }

// addListener runs fn once f completes: synchronously during completion if
// f is still pending, in a microtask if it already completed (Dart's
// _addListener).
func (f *future[T]) addListener(fn func()) {
	if f.isComplete() {
		f.loop.ScheduleMicrotask(fn)
		return
	}
	f.listeners = append(f.listeners, fn)
}

func (f *future[T]) completeWithValue(v T) {
	f.value, f.state = v, hasValue
	f.propagate()
}

func (f *future[T]) completeError(err error) {
	f.err, f.state = err, hasError
	f.propagate()
}

func (f *future[T]) propagate() {
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

func (f *future[T]) asyncCompleteWithValue(v T) {
	f.state = pendingComplete
	f.loop.ScheduleMicrotask(func() { f.completeWithValue(v) })
}

func (f *future[T]) asyncCompleteError(err error) {
	f.state = pendingComplete
	f.loop.ScheduleMicrotask(func() { f.completeError(err) })
}

// chain completes f with src's result (Dart's _chainCoreFuture when src is
// one of this package's futures, _chainForeignFuture otherwise).
func (f *future[T]) chain(src Future[T], sync bool) {
	if core, ok := src.(*future[T]); ok {
		chainCore(f, core, sync, func(v T) T { return v })
		return
	}
	if c, ok := src.(coreCast[T]); ok {
		chainCore(f, c.inner, sync, castValue[T])
		return
	}
	f.chainForeign(src)
}

// chainCore completes f with src's result, converted by conv: synchronously
// if src is already complete and sync is set, otherwise when src completes.
func chainCore[T, S any](f *future[T], src *future[S], sync bool, conv func(S) T) {
	f.state = pendingComplete
	copyResult := func() {
		if src.state == hasError {
			src.ignored = true
			f.completeError(src.err)
		} else {
			f.completeWithValue(conv(src.value))
		}
	}
	switch {
	case src.isComplete() && (sync || (len(f.listeners) == 0 && (src.state != hasError || f.ignored))):
		// Dart completes synchronously when allowed, or when no one listens yet
		// (an un-ignored error is still delayed, to give time to add handlers).
		copyResult()
	case src.isComplete():
		f.loop.ScheduleMicrotask(copyResult)
	default:
		src.ignored = true
		src.listeners = append(src.listeners, copyResult)
	}
}

// chainForeign is Dart's _chainForeignFuture: a user-defined future is
// listened to through its then, and f completes synchronously in the
// callback.
func (f *future[T]) chainForeign(src Future[T]) {
	f.state = pendingComplete
	_, err := call(func() Future[any] {
		return src.ThenAny(
			func(v T) any { f.completeWithValue(v); return nil },
			func(err error) any { f.completeError(err); return nil },
		)
	})
	if err != nil {
		// A non-conforming then that throws synchronously.
		ScheduleMicrotask(func() { f.completeError(err) })
	}
}

// completeWithResult completes f with a then callback's result r, which is
// either a value or a future of one (FutureOr<T>).
func (f *future[T]) completeWithResult(r any, err error) {
	if err != nil {
		f.completeError(err)
		return
	}
	if fut, ok := r.(Future[any]); ok {
		f.chain(castFuture[T](fut), true)
		return
	}
	f.completeWithValue(castValue[T](r))
}

// ---- constructors ----

// FutureValue is Future.value(v): completes with v in a microtask.
func FutureValue[T any](v T) Future[T] {
	f := newFuture[T]()
	f.asyncCompleteWithValue(v)
	return f
}

// FutureError is Future.error(err): fails with err in a microtask.
func FutureError[T any](err error) Future[T] {
	f := newFuture[T]()
	f.asyncCompleteError(err)
	return f
}

// NewFuture is Future(computation): runs it in a zero-duration timer.
func NewFuture[T any](computation func() T) Future[T] {
	f := newFuture[T]()
	TimerRun(func() { completeFrom(f, computation) })
	return f
}

// FutureMicrotask is Future.microtask(computation).
func FutureMicrotask[T any](computation func() T) Future[T] {
	f := newFuture[T]()
	ScheduleMicrotask(func() { completeFrom(f, computation) })
	return f
}

// FutureDelayed is Future.delayed(d, computation); computation may be nil
// (the future then completes with the zero value).
func FutureDelayed[T any](d time.Duration, computation func() T) Future[T] {
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

func completeFrom[T any](f *future[T], computation func() T) {
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
func Then[T, R any](f Future[T], onValue func(T) R) Future[R] {
	return ThenOrError(f, onValue, nil)
}

// ThenOrError is future.then(onValue, onError: onError); onError may be nil.
func ThenOrError[T, R any](f Future[T], onValue func(T) R, onError func(error) R) Future[R] {
	if core, ok := f.(*future[T]); ok {
		result := newFuture[R]()
		core.addListener(func() {
			if core.state == hasError {
				if onError == nil {
					result.completeError(core.err)
					return
				}
				completeFrom(result, func() R { return onError(core.err) })
				return
			}
			completeFrom(result, func() R { return onValue(core.value) })
		})
		return result
	}
	var onErrorAny func(error) any
	if onError != nil {
		onErrorAny = func(err error) any { return onError(err) }
	}
	return castFuture[R](f.ThenAny(func(v T) any { return onValue(v) }, onErrorAny))
}

// ThenFuture is future.then(onValue) where onValue returns a Future: the
// result completes when that future does.
func ThenFuture[T, R any](f Future[T], onValue func(T) Future[R]) Future[R] {
	if core, ok := f.(*future[T]); ok {
		result := newFuture[R]()
		core.addListener(func() {
			if core.state == hasError {
				result.completeError(core.err)
				return
			}
			next, err := call(func() Future[R] { return onValue(core.value) })
			if err != nil {
				result.completeError(err)
				return
			}
			result.chain(next, true)
		})
		return result
	}
	return castFuture[R](f.ThenAny(func(v T) any { return EraseFuture(onValue(v)) }, nil))
}

// OnComplete is future.then(onValue, onError: onError) with callbacks
// whose results are discarded, without a result future to listen to. onError
// may be nil, in which case an error is uncaught.
func OnComplete[T any](f Future[T], onValue func(T), onError func(error)) {
	if core, ok := f.(*future[T]); ok {
		core.addListener(func() {
			if core.state == hasError {
				if onError == nil {
					core.loop.reportUncaught(core.err)
					return
				}
				onError(core.err)
				return
			}
			onValue(core.value)
		})
		return
	}
	var onErrorAny func(error) any
	if onError != nil {
		onErrorAny = func(err error) any { onError(err); return nil }
	}
	f.ThenAny(func(v T) any { onValue(v); return nil }, onErrorAny)
}

// ThenAny is _Future.then with R erased (see Future).
func (f *future[T]) ThenAny(onValue func(T) any, onError func(error) any) Future[any] {
	result := newFuture[any]()
	f.addListener(func() {
		if f.state == hasError {
			if onError == nil {
				result.completeError(f.err)
				return
			}
			result.completeWithResult(call(func() any { return onError(f.err) }))
			return
		}
		result.completeWithResult(call(func() any { return onValue(f.value) }))
	})
	return result
}

// CatchError is future.catchError(onError): recovers from an error with
// onError's value. test, when non-nil, selects which errors to handle.
func (f *future[T]) CatchError(onError func(error) T, test func(error) bool) Future[T] {
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
func (f *future[T]) WhenComplete(action func()) Future[T] {
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

// Timeout is future.timeout(timeLimit, onTimeout: onTimeout): a future of
// f's result, or, if f takes longer than timeLimit, of onTimeout's result
// (a TimeoutException when onTimeout is nil).
func (f *future[T]) Timeout(timeLimit time.Duration, onTimeout func() T) Future[T] {
	result := newFuture[T]()
	if f.isComplete() {
		// _Future.immediate(this): chains without completing synchronously.
		chainCore(result, f, false, func(v T) T { return v })
		return result
	}
	var timer *Timer
	if onTimeout == nil {
		timer = NewTimer(timeLimit, func() {
			result.completeError(TimeoutException{Message: "Future not completed", Duration: timeLimit})
		})
	} else {
		timer = NewTimer(timeLimit, func() { completeFrom(result, onTimeout) })
	}
	OnComplete[T](f, func(v T) {
		if timer.IsActive() {
			timer.Cancel()
			result.completeWithValue(v)
		}
	}, func(err error) {
		if timer.IsActive() {
			timer.Cancel()
			result.completeError(err)
		}
	})
	return result
}

// Ignore marks f's error, if any, as handled (future.ignore()).
func (f *future[T]) Ignore() { f.ignored = true }

// TimeoutException is dart:async's TimeoutException.
type TimeoutException struct {
	Message  string // "" for null
	Duration time.Duration
}

func (e TimeoutException) Error() string {
	s := "TimeoutException after " + dartDuration(e.Duration)
	if e.Message != "" {
		s += ": " + e.Message
	}
	return s
}

// dartDuration is Duration.toString: "H:MM:SS.ffffff".
func dartDuration(d time.Duration) string {
	sign := ""
	us := d.Microseconds()
	if us < 0 {
		sign, us = "-", -us
	}
	h := us / 3_600_000_000
	us -= h * 3_600_000_000
	m := us / 60_000_000
	us -= m * 60_000_000
	s := us / 1_000_000
	us -= s * 1_000_000
	return fmt.Sprintf("%s%d:%02d:%02d.%06d", sign, h, m, s, us)
}

// ---- user-defined futures ----

// AwaitUserDefined is Dart's `await` on a future that is not one of this
// package's (the VM's _awaitUserDefinedFuture): it listens with f.ThenAny
// and resumes the coroutine from the callback. If the callback runs
// synchronously (SynchronousFuture), the coroutine continues without
// suspending, as the VM resumes it within then. Implementations of Future
// return it from Await.
func AwaitUserDefined[T any](f Future[T]) T {
	co := running
	if co == nil {
		panic("async: Await called outside Async")
	}
	co.suspended = true
	var (
		value   T
		err     error
		done    bool
		waiting bool
	)
	resume := func() {
		done = true
		if waiting {
			co.step()
		}
	}
	f.ThenAny(
		func(v T) any { value = v; resume(); return nil },
		func(e error) any { err = e; resume(); return nil },
	)
	if !done {
		waiting = true
		co.yield <- struct{}{}
		<-co.resume
	}
	if err != nil {
		panic(err)
	}
	return value
}

// IgnoreUserDefined is Dart's ignore() on a user-defined future:
// f.then(_ignore, onError: _ignore). Implementations of Future call it
// from Ignore.
func IgnoreUserDefined[T any](f Future[T]) {
	f.ThenAny(func(T) any { return nil }, func(error) any { return nil })
}

// ---- type erasure for ThenAny ----

// EraseFuture views f as a Future[any], for returning a future from a
// ThenAny callback (Dart's `FutureOr<R>` holding a `Future<R>`).
func EraseFuture[T any](f Future[T]) Future[any] {
	switch f := f.(type) {
	case Future[any]:
		return f
	case userCast[T]:
		return f.inner
	case coreCast[T]:
		return f.inner
	}
	return erased[T]{f}
}

// castFuture views a Future[any] produced by ThenAny as a Future[R]. A
// future that EraseFuture wrapped is returned as it was; this package's
// futures stay core futures (coreCast), so chaining them keeps
// _chainCoreFuture's timing.
func castFuture[R any](f Future[any]) Future[R] {
	switch f := f.(type) {
	case Future[R]:
		return f
	case erased[R]:
		return f.inner
	case *future[any]:
		return coreCast[R]{f}
	}
	return userCast[R]{f}
}

func castValue[R any](v any) R {
	var zero R
	if _, isVoid := any(zero).(Void); isVoid {
		return zero // Future<void> accepts (and ignores) any value
	}
	if v == nil {
		var zero R
		return zero
	}
	return v.(R)
}

// erased is a Future[T] seen as a Future[any].
type erased[T any] struct{ inner Future[T] }

func (e erased[T]) ThenAny(onValue func(any) any, onError func(error) any) Future[any] {
	return e.inner.ThenAny(func(v T) any { return onValue(v) }, onError)
}

func (e erased[T]) CatchError(onError func(error) any, test func(error) bool) Future[any] {
	return EraseFuture(e.inner.CatchError(func(err error) T { return castValue[T](onError(err)) }, test))
}

func (e erased[T]) WhenComplete(action func()) Future[any] {
	return EraseFuture(e.inner.WhenComplete(action))
}

func (e erased[T]) Timeout(timeLimit time.Duration, onTimeout func() any) Future[any] {
	var t func() T
	if onTimeout != nil {
		t = func() T { return castValue[T](onTimeout()) }
	}
	return EraseFuture(e.inner.Timeout(timeLimit, t))
}

func (e erased[T]) Ignore()    { e.inner.Ignore() }
func (e erased[T]) Await() any { return e.inner.Await() }

// userCast is a user-defined Future[any] seen as a Future[R].
type userCast[R any] struct{ inner Future[any] }

func (c userCast[R]) ThenAny(onValue func(R) any, onError func(error) any) Future[any] {
	return c.inner.ThenAny(func(v any) any { return onValue(castValue[R](v)) }, onError)
}

func (c userCast[R]) CatchError(onError func(error) R, test func(error) bool) Future[R] {
	return castFuture[R](c.inner.CatchError(func(err error) any { return onError(err) }, test))
}

func (c userCast[R]) WhenComplete(action func()) Future[R] {
	return castFuture[R](c.inner.WhenComplete(action))
}

func (c userCast[R]) Timeout(timeLimit time.Duration, onTimeout func() R) Future[R] {
	var t func() any
	if onTimeout != nil {
		t = func() any { return onTimeout() }
	}
	return castFuture[R](c.inner.Timeout(timeLimit, t))
}

func (c userCast[R]) Ignore()  { c.inner.Ignore() }
func (c userCast[R]) Await() R { return castValue[R](c.inner.Await()) }

// coreCast is one of this package's Future[any] seen as a Future[R]; the
// package's functions treat it as a core future.
type coreCast[R any] struct{ inner *future[any] }

func (c coreCast[R]) ThenAny(onValue func(R) any, onError func(error) any) Future[any] {
	return c.inner.ThenAny(func(v any) any { return onValue(castValue[R](v)) }, onError)
}

func (c coreCast[R]) CatchError(onError func(error) R, test func(error) bool) Future[R] {
	return castFuture[R](c.inner.CatchError(func(err error) any { return onError(err) }, test))
}

func (c coreCast[R]) WhenComplete(action func()) Future[R] {
	return castFuture[R](c.inner.WhenComplete(action))
}

func (c coreCast[R]) Timeout(timeLimit time.Duration, onTimeout func() R) Future[R] {
	var t func() any
	if onTimeout != nil {
		t = func() any { return onTimeout() }
	}
	return castFuture[R](c.inner.Timeout(timeLimit, t))
}

func (c coreCast[R]) Ignore()  { c.inner.Ignore() }
func (c coreCast[R]) Await() R { return castValue[R](c.inner.Await()) }

// ---- Completer ----

// Completer is dart:async's Completer<T>.
type Completer[T any] struct {
	future *future[T]
	sync   bool
}

// NewCompleter is Completer(): listeners hear the result in a microtask.
func NewCompleter[T any]() *Completer[T] { return &Completer[T]{future: newFuture[T]()} }

// NewSyncCompleter is Completer.sync(): listeners hear the result
// immediately.
func NewSyncCompleter[T any]() *Completer[T] {
	return &Completer[T]{future: newFuture[T](), sync: true}
}

func (c *Completer[T]) Future() Future[T] { return c.future }

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
func (c *Completer[T]) CompleteFuture(f Future[T]) {
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
