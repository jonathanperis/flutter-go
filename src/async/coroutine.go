package async

// Async runs body as a Dart async function: synchronously until its first
// Await, then resumed by the loop when the awaited future completes. The
// returned future carries body's result, or its panic as an error.
//
// body runs on its own goroutine, but control is handed over explicitly, so
// it never runs at the same time as the loop or another coroutine. A
// coroutine that awaits a future which never completes stays parked (Dart
// would garbage-collect it).
func Async[T any](body func() T) Future[T] {
	f := newFuture[T]()
	co := &coroutine{resume: make(chan struct{}), yield: make(chan struct{})}
	go func() {
		<-co.resume
		defer func() {
			if r := recover(); r != nil {
				// Before the first Await Dart completes the error in a
				// microtask; afterwards, synchronously.
				if co.suspended {
					f.completeError(asError(r))
				} else {
					f.asyncCompleteError(asError(r))
				}
			}
			co.done = true
			co.yield <- struct{}{}
		}()
		f.completeWithValue(body())
	}()
	co.step()
	return f
}

// AsyncVoid is Async for a body without a result (Future<void>).
func AsyncVoid(body func()) Future[Void] {
	return Async(func() Void { body(); return Void{} })
}

type coroutine struct {
	resume, yield chan struct{}
	suspended     bool // has awaited at least once
	done          bool
}

// running is the coroutine that currently has control (nil: the loop).
// Only the goroutine holding control touches it.
var running *coroutine

func (co *coroutine) step() {
	prev := running
	running = co
	co.resume <- struct{}{}
	<-co.yield
	running = prev
}

// Await suspends the calling coroutine until f completes and returns its
// value, or panics with its error (Dart's await). It must be called inside
// Async.
func (f *future[T]) Await() T {
	co := running
	if co == nil {
		panic("async: Await called outside Async")
	}
	co.suspended = true
	f.ignored = true
	f.addListener(co.step)
	co.yield <- struct{}{}
	<-co.resume
	if f.state == hasError {
		panic(f.err)
	}
	return f.value
}
