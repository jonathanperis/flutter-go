package async

// CastFuture is Dart's implicit downcast of a `Future<dynamic>` to a
// `Future<R>` (`f as Future<R>`) for a future whose value is an R (or
// null): the same future seen with its static type, without the extra
// asynchronous step a Then would add. A future that EraseFuture wrapped
// comes back as it was.
func CastFuture[R any](f Future[any]) Future[R] { return castFuture[R](f) }
