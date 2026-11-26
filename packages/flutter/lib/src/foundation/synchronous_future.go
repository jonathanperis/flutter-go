package foundation

import (
	"time"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
)

// SynchronousFuture is a Future whose then calls the callback synchronously
// (synchronous_future.dart).
//
// This is useful when a function returns a Future but its value is already
// known: callers that only use then (or Then) see the value immediately,
// while callers that await it are resumed at once as well (the Dart VM's
// await on a user-defined future calls then). Like Flutter's, it is
// "dangerous": code that relies on asynchrony breaks with it.
//
// Divergences: asStream is not ported (no streams yet); WhenComplete's
// action can't return a Future in Go, so the branch that waits for one does
// not exist.
type SynchronousFuture[T any] struct {
	value T
}

// NewSynchronousFuture creates a future that completes synchronously with
// value (Dart's SynchronousFuture(value)).
func NewSynchronousFuture[T any](value T) *SynchronousFuture[T] {
	return &SynchronousFuture[T]{value: value}
}

var _ async.Future[int] = (*SynchronousFuture[int])(nil)

// ThenAny is then: onValue runs now; a future it returns is returned as is,
// any other value is wrapped in a SynchronousFuture. onError is never used.
// Call async.Then/ThenFuture rather than this method.
func (s *SynchronousFuture[T]) ThenAny(onValue func(T) any, onError func(error) any) async.Future[any] {
	result := onValue(s.value)
	if f, ok := result.(async.Future[any]); ok {
		return f
	}
	return NewSynchronousFuture(result)
}

// CatchError returns a future that never completes: a SynchronousFuture has
// no error to catch (Dart returns Completer<T>().future).
func (s *SynchronousFuture[T]) CatchError(onError func(error) T, test func(error) bool) async.Future[T] {
	return async.NewCompleter[T]().Future()
}

// Timeout is Future<T>.value(value).timeout(timeLimit, onTimeout: onTimeout).
func (s *SynchronousFuture[T]) Timeout(timeLimit time.Duration, onTimeout func() T) async.Future[T] {
	return async.FutureValue(s.value).Timeout(timeLimit, onTimeout)
}

// WhenComplete runs action now and returns s, or a failed future if action
// panics (Dart: Future<T>.error(e, stack)).
func (s *SynchronousFuture[T]) WhenComplete(action func()) (result async.Future[T]) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = async.PanicError{Value: r}
			}
			result = async.FutureError[T](err)
		}
	}()
	action()
	return s
}

// Ignore is Dart's ignore() on a user-defined future.
func (s *SynchronousFuture[T]) Ignore() { async.IgnoreUserDefined[T](s) }

// Await is Dart's await: the VM listens with then, which calls back at once,
// so the awaiting coroutine continues without suspending.
func (s *SynchronousFuture[T]) Await() T { return async.AwaitUserDefined[T](s) }
