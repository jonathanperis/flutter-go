package foundation

// foundation.dart's
//
//	export 'package:listen/listen.dart' show ChangeNotifier, Listenable, ValueListenable, ValueNotifier;
//
// Go has no re-exports, so the four names (and the statics and
// constructors that come with them) are aliases and wrappers of package
// listen (src/pub/listen, package:listen 1.0.1), as Flutter users
// reach them through foundation. The assignable statics stay in listen (a
// Go variable can't be aliased): Listenable.onError is
// listen.ListenableOnError, Listenable.debugMaybeDispatchCreated/Disposed
// are listen.ListenableDebugMaybeDispatchCreated/Disposed.

import (
	"iter"

	"github.com/jonathanperis/flutter-go/src/pub/listen"
)

type (
	// Listenable is listen.Listenable.
	Listenable = listen.Listenable
	// ValueListenable is listen.ValueListenable.
	ValueListenable[T any] = listen.ValueListenable[T]
	// ChangeNotifier is listen.ChangeNotifier.
	ChangeNotifier = listen.ChangeNotifier
	// ChangeNotifierMixin is listen.ChangeNotifierMixin, the struct to embed
	// for Dart's `extends ChangeNotifier` / `with ChangeNotifier`.
	ChangeNotifierMixin = listen.ChangeNotifierMixin
	// ValueNotifier is listen.ValueNotifier.
	ValueNotifier[T any] = listen.ValueNotifier[T]
)

// NewChangeNotifier is ChangeNotifier() (listen.NewChangeNotifier).
func NewChangeNotifier() *ChangeNotifierMixin { return listen.NewChangeNotifier() }

// NewValueNotifier is ValueNotifier(value) (listen.NewValueNotifier).
func NewValueNotifier[T any](value T) *ValueNotifier[T] { return listen.NewValueNotifier(value) }

// ListenableMerge is Listenable.merge (listen.ListenableMerge).
func ListenableMerge(listenables iter.Seq[Listenable]) Listenable {
	return listen.ListenableMerge(listenables)
}

// ChangeNotifierDebugAssertNotDisposed is ChangeNotifier.debugAssertNotDisposed
// (listen.ChangeNotifierDebugAssertNotDisposed).
func ChangeNotifierDebugAssertNotDisposed(notifier ChangeNotifier) bool {
	return listen.ChangeNotifierDebugAssertNotDisposed(notifier)
}

// ChangeNotifierMaybeDispatchObjectCreation is
// ChangeNotifier.maybeDispatchObjectCreation
// (listen.ChangeNotifierMaybeDispatchObjectCreation).
func ChangeNotifierMaybeDispatchObjectCreation(object ChangeNotifier) {
	listen.ChangeNotifierMaybeDispatchObjectCreation(object)
}
