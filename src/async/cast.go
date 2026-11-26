package async

// CastFuture is Dart's cast of a Future<dynamic> to Future<R> (as a method
// returning `Future<void>` returns an `invokeMethod` future): the same
// future, with no extra event-loop turn; its value reads as R (null as R's
// zero value), and as Void whatever it is (a `Future<void>` ignores its
// value).
func CastFuture[R any](f Future[any]) Future[R] { return castFuture[R](f) }
