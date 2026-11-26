// Package listen is package:listen 1.0.1 (lib/src/listen.dart), the pure
// Dart package Flutter 3.50.1's foundation library re-exports ChangeNotifier,
// Listenable, ValueListenable and ValueNotifier from. Flutter users reach
// those names through foundation (see foundation/listen_exports.go); this
// package holds the whole public API of package:listen.
//
// Listeners are func() values compared as Dart compares functions
// (dart.FunctionsEqual): remove the same function value that was added, or
// a method value of the same receiver pointer (x.M, as Dart's tear-offs).
//
// Dart's assert(() {...}()) blocks (the disposed check, creation and
// disposal dispatch) run in debug builds only; plain assertions panic in
// every build (see API_MAPPING.md).
package listen

import (
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// VoidCallback is the signature of callbacks that have no arguments and
// return no data (an alias, so func() values and interface methods taking
// func() match it).
type VoidCallback = func()

// ErrorContext identifies the context in which an error was reported to
// ListenableOnError.
type ErrorContext int

const (
	// ErrorContextListener: the error was thrown by a listener callback
	// during Listenable notification.
	ErrorContextListener ErrorContext = iota
	// ErrorContextAssertion: the error was reported by a debug assertion or
	// lifecycle check (e.g. using a disposed notifier).
	ErrorContextAssertion
)

var errorContextNames = [...]string{"listener", "assertion"}

// String is the Dart enum name.
func (c ErrorContext) String() string { return errorContextNames[c] }

// ErrorCallback is the signature of callbacks that are called when an error
// is reported by a listener or assertion. error is Dart's `Object error`
// (any value a listener panicked with); stackTrace may be nil.
type ErrorCallback func(error any, stackTrace dart.StackTrace, context ErrorContext)

// ObjectCreatedCallback is the signature of callbacks that are called when
// an object is created.
type ObjectCreatedCallback func(className string, object any)

// ObjectDisposedCallback is the signature of callbacks that are called when
// an object is disposed.
type ObjectDisposedCallback func(object any)

// Listenable is an object that maintains a list of listeners, typically
// used to notify clients that the object has been updated.
type Listenable interface {
	// AddListener registers a closure to be called when the object
	// notifies its listeners.
	AddListener(listener VoidCallback)
	// RemoveListener removes a previously registered closure from the list
	// of closures that the object notifies.
	RemoveListener(listener VoidCallback)
}

// ListenableOnError is Listenable.onError: called when an error is thrown
// by a listener or reported by an assertion. Flutter's BindingBase replaces
// it. The default rethrows the error (Go divergence: a panic carries no
// stack trace, so the original stackTrace is not preserved).
var ListenableOnError ErrorCallback = defaultOnError

func defaultOnError(error any, stackTrace dart.StackTrace, context ErrorContext) {
	panic(error)
}

// ListenableDebugMaybeDispatchCreated is Listenable.debugMaybeDispatchCreated:
// called when a new object is created (debug builds only). Flutter's
// FlutterMemoryAllocations replaces it.
var ListenableDebugMaybeDispatchCreated ObjectCreatedCallback = func(string, any) {}

// ListenableDebugMaybeDispatchDisposed is
// Listenable.debugMaybeDispatchDisposed: called when an object is disposed
// (debug builds only).
var ListenableDebugMaybeDispatchDisposed ObjectDisposedCallback = func(any) {}

// ListenableMerge is Listenable.merge: a Listenable that triggers when any
// of the given Listenables trigger. The sequence is iterated each time a
// listener is added or removed, so it must not change once merged. Nil
// entries (Dart's nulls) are ignored.
func ListenableMerge(listenables iter.Seq[Listenable]) Listenable {
	return &mergingListenable{children: listenables}
}

// mergingListenable is _MergingListenable.
type mergingListenable struct {
	children iter.Seq[Listenable]
}

func (m *mergingListenable) AddListener(listener VoidCallback) {
	for child := range m.children {
		if !isNull(child) {
			child.AddListener(listener)
		}
	}
}

func (m *mergingListenable) RemoveListener(listener VoidCallback) {
	for child := range m.children {
		if !isNull(child) {
			child.RemoveListener(listener)
		}
	}
}

func (m *mergingListenable) String() string {
	var parts []string
	for child := range m.children {
		parts = append(parts, toString(child))
	}
	return "Listenable.merge([" + strings.Join(parts, ", ") + "])"
}

// ValueListenable is a Listenable that exposes a current value.
type ValueListenable[T any] interface {
	Listenable
	// Value is the current value of the object. When it changes, the
	// callbacks registered with AddListener are invoked.
	Value() T
}

// ChangeNotifier is the type of Dart's `mixin class ChangeNotifier`: a
// Listenable that can be extended or mixed in, with the change notification
// API. Embed ChangeNotifierMixin to implement it (as Dart's `extends
// ChangeNotifier` / `with ChangeNotifier`); NewChangeNotifier is Dart's
// `ChangeNotifier()`.
type ChangeNotifier interface {
	Listenable
	// Dispose discards any resources used by the object.
	Dispose()
	// NotifyListeners calls all the registered listeners (protected).
	NotifyListeners()
	// HasListeners reports whether any listeners are registered
	// (protected).
	HasListeners() bool
	changeNotifier() *ChangeNotifierMixin
}

// ChangeNotifierMixin is the state and methods of ChangeNotifier. A type
// embedding it calls SetSelf(outer) once, when it is created, so that
// error messages name the outer type, the memory-allocation events carry
// the outer object, and overrides of NotifyListeners are called (see
// API_MAPPING's self pattern). It is used by pointer.
//
// Adding a listener is O(1); removing one and notifying are O(N). Listeners
// added during a notification are not called by it; listeners removed
// during one are not called after their removal.
type ChangeNotifierMixin struct {
	self ChangeNotifier

	count int
	// listeners is Dart's fixed-length _listeners: len(listeners) is its
	// length, and nil entries are removed listeners (during a
	// notification) or free slots.
	listeners                   []VoidCallback
	notificationCallStackDepth  int
	reentrantlyRemovedListeners int
	debugDisposed               bool
	// debugCreationDispatched is whether the creation of this instance was
	// dispatched to ListenableDebugMaybeDispatchCreated (on the first
	// AddListener, ChangeNotifier having no constructor).
	debugCreationDispatched bool
}

// NewChangeNotifier is ChangeNotifier(): a plain notifier.
func NewChangeNotifier() *ChangeNotifierMixin {
	n := &ChangeNotifierMixin{}
	n.SetSelf(n)
	return n
}

// SetSelf points the mixin at the object it is mixed into.
func (n *ChangeNotifierMixin) SetSelf(self ChangeNotifier) { n.self = self }

func (n *ChangeNotifierMixin) changeNotifier() *ChangeNotifierMixin { return n }

func (n *ChangeNotifierMixin) outer() ChangeNotifier {
	if n.self == nil {
		panic("listen: ChangeNotifierMixin used before SetSelf; call SetSelf with the outer object when creating it")
	}
	return n.self
}

// String is Object.toString for the outer object: "Instance of 'Type'".
func (n *ChangeNotifierMixin) String() string {
	return "Instance of '" + runtimeTypeName(n.outer()) + "'"
}

// ChangeNotifierDebugAssertNotDisposed is ChangeNotifier.debugAssertNotDisposed:
// in debug builds, reports a StateError to ListenableOnError (context
// assertion) when notifier has been disposed. It returns true, so it can be
// used in an assertion.
func ChangeNotifierDebugAssertNotDisposed(notifier ChangeNotifier) bool {
	if debugMode && notifier.changeNotifier().debugDisposed {
		typ := runtimeTypeName(notifier)
		ListenableOnError(
			dart.StateError{Message: "A " + typ + " was used after being disposed.\n" +
				"Once you have called dispose() on a " + typ + ", it can no longer be used."},
			dart.StackTraceCurrent(),
			ErrorContextAssertion,
		)
	}
	return true
}

// HasListeners reports whether any listeners are currently registered
// (protected). It is false once Dispose has been called.
func (n *ChangeNotifierMixin) HasListeners() bool { return n.count > 0 }

// ChangeNotifierMaybeDispatchObjectCreation is
// ChangeNotifier.maybeDispatchObjectCreation (protected): in debug builds,
// dispatches object's creation to ListenableDebugMaybeDispatchCreated, once.
// Call it from the constructor of a type embedding ChangeNotifierMixin, so
// leak trackers see the creation's stack rather than the first
// AddListener's.
func ChangeNotifierMaybeDispatchObjectCreation(object ChangeNotifier) {
	if debugMode {
		m := object.changeNotifier()
		if !m.debugCreationDispatched {
			ListenableDebugMaybeDispatchCreated("ChangeNotifier", object)
			m.debugCreationDispatched = true
		}
	}
}

// AddListener registers a closure to be called when the object changes.
// A closure registered twice is called twice, and must be removed twice.
// It must not be called after Dispose. listener must not be nil (Dart's
// VoidCallback is non-nullable).
func (n *ChangeNotifierMixin) AddListener(listener VoidCallback) {
	if listener == nil {
		panic("listen: AddListener(nil): a listener can't be null")
	}
	self := n.outer()
	ChangeNotifierDebugAssertNotDisposed(self)
	if debugMode {
		ChangeNotifierMaybeDispatchObjectCreation(self)
	}

	if n.count == len(n.listeners) {
		if n.count == 0 {
			n.listeners = make([]VoidCallback, 1)
		} else {
			newListeners := make([]VoidCallback, len(n.listeners)*2)
			for i := 0; i < n.count; i++ {
				newListeners[i] = n.listeners[i]
			}
			n.listeners = newListeners
		}
	}
	n.listeners[n.count] = listener
	n.count++
}

func (n *ChangeNotifierMixin) removeAt(index int) {
	// The list holding the listeners is not growable for performance
	// reasons; it is shrunk when the real number of listeners is half its
	// length.
	n.count--
	if n.count*2 <= len(n.listeners) {
		newListeners := make([]VoidCallback, n.count)

		// Listeners before the index are at the same place.
		for i := 0; i < index; i++ {
			newListeners[i] = n.listeners[i]
		}

		// Listeners after the index move towards the start of the list.
		for i := index; i < n.count; i++ {
			newListeners[i] = n.listeners[i+1]
		}

		n.listeners = newListeners
	} else {
		// Otherwise only shift the listeners, to avoid reallocating.
		for i := index; i < n.count; i++ {
			n.listeners[i] = n.listeners[i+1]
		}
		n.listeners[n.count] = nil
	}
}

// RemoveListener removes the first registration of listener (compared with
// dart.FunctionsEqual); it does nothing if listener is not registered. It
// may be called after Dispose.
//
// During a notification the entry is cleared and the list is compacted
// once every (nested) notification is over; so a listener registered twice
// and removed once during a notification is still called again.
func (n *ChangeNotifierMixin) RemoveListener(listener VoidCallback) {
	for i := 0; i < n.count; i++ {
		listenerAtIndex := n.listeners[i]
		if listenerAtIndex != nil && dart.FunctionsEqual(listenerAtIndex, listener) {
			if n.notificationCallStackDepth > 0 {
				// Not resized during notifyListeners iterations: the entry
				// is cleared, and the list compacted at the end of all
				// notifyListeners iterations.
				n.listeners[i] = nil
				n.reentrantlyRemovedListeners++
			} else {
				// Outside the notifyListeners iterations the list can be
				// shrunk right away.
				n.removeAt(i)
			}
			break
		}
	}
}

// Dispose discards any resources used by the object: it clears the
// listeners without notifying them. Afterwards the object is not usable
// (AddListener, NotifyListeners and Dispose report errors in debug builds).
// Overrides must call it.
func (n *ChangeNotifierMixin) Dispose() {
	self := n.outer()
	ChangeNotifierDebugAssertNotDisposed(self)
	if n.notificationCallStackDepth != 0 {
		panic(dart.NewAssertionError(`The "dispose()" method on ` + toString(self) + ` was called during the call to ` +
			`"notifyListeners()". This is likely to cause errors since it modifies ` +
			`the list of listeners while the list is being used.`))
	}
	if debugMode {
		n.debugDisposed = true
		if n.debugCreationDispatched {
			ListenableDebugMaybeDispatchDisposed(self)
		}
	}
	n.listeners = nil
	n.count = 0
}

// NotifyListeners calls all the registered listeners (protected; visible
// for testing). Listeners added during the call are not called; listeners
// removed during it are not called after their removal. A listener's panic
// is recovered and reported to ListenableOnError (context listener), and
// the other listeners still run. It must not be called after Dispose.
func (n *ChangeNotifierMixin) NotifyListeners() {
	ChangeNotifierDebugAssertNotDisposed(n.outer())
	if n.count == 0 {
		return
	}

	// Listeners removed during this iteration are set to nil, and the list
	// is not shrunk until every (re-entrant) iteration is over, tracked by
	// notificationCallStackDepth.
	n.notificationCallStackDepth++

	end := n.count
	for i := 0; i < end; i++ {
		n.callListener(i)
	}

	n.notificationCallStackDepth--

	if n.notificationCallStackDepth == 0 && n.reentrantlyRemovedListeners > 0 {
		// The listeners are really removed when all notifications are done.
		newLength := n.count - n.reentrantlyRemovedListeners
		if newLength*2 <= len(n.listeners) {
			// As in removeAt, the list is shrunk only when the real number
			// of listeners is half its length.
			newListeners := make([]VoidCallback, newLength)

			newIndex := 0
			for i := 0; i < n.count; i++ {
				if listener := n.listeners[i]; listener != nil {
					newListeners[newIndex] = listener
					newIndex++
				}
			}

			n.listeners = newListeners
		} else {
			// Otherwise the nil entries are moved to the end.
			for i := 0; i < newLength; i++ {
				if n.listeners[i] == nil {
					// Swap this item with the next non-nil item.
					swapIndex := i + 1
					for n.listeners[swapIndex] == nil {
						swapIndex++
					}
					n.listeners[i] = n.listeners[swapIndex]
					n.listeners[swapIndex] = nil
				}
			}
		}

		n.reentrantlyRemovedListeners = 0
		n.count = newLength
	}
}

// callListener is one iteration of notifyListeners' try/catch: a panic is
// Dart's exception, reported with the panic's stack. If ListenableOnError
// panics (the default rethrows), the notification is abandoned without
// restoring the call depth, as in Dart (no finally).
func (n *ChangeNotifierMixin) callListener(i int) {
	defer func() {
		if r := recover(); r != nil {
			stack := dart.StackTraceFromPanic()
			ListenableOnError(r, stack, ErrorContextListener)
		}
	}()
	if listener := n.listeners[i]; listener != nil {
		listener()
	}
}

// ValueNotifier is a ChangeNotifier that holds a single value, and notifies
// its listeners when the value is replaced with one that is not equal to
// the old one (Dart's ==: an Equal(T) bool method when T has one, Go's ==
// otherwise, identity for slices, maps and functions).
type ValueNotifier[T any] struct {
	ChangeNotifierMixin
	value T
}

// NewValueNotifier is ValueNotifier(value).
func NewValueNotifier[T any](value T) *ValueNotifier[T] {
	n := &ValueNotifier[T]{}
	n.Construct(n, value)
	return n
}

// Construct is Dart's ValueNotifier(value) constructor for a type that
// embeds ValueNotifier (Dart's `super(value)`): it sets self, the value,
// and (in debug builds) dispatches the creation of self.
func (n *ValueNotifier[T]) Construct(self ChangeNotifier, value T) {
	n.SetSelf(self)
	n.value = value
	if debugMode {
		ChangeNotifierMaybeDispatchObjectCreation(self)
	}
}

// Value is the current value stored in this notifier.
func (n *ValueNotifier[T]) Value() T { return n.value }

// SetValue replaces the value and notifies the listeners, unless newValue
// equals the current value.
func (n *ValueNotifier[T]) SetValue(newValue T) {
	if dartEquals(n.value, newValue) {
		return
	}
	n.value = newValue
	n.outer().NotifyListeners()
}

// String is "ValueNotifier<T>#hash(value)" (the runtime type of the outer
// object and its Value()).
func (n *ValueNotifier[T]) String() string {
	self := n.outer()
	value := any(n.value)
	if v, ok := self.(ValueListenable[T]); ok {
		value = v.Value()
	}
	return describeIdentity(self) + "(" + toString(value) + ")"
}

// ---- Dart object model helpers (package:listen can't use foundation's) ----

// dartEquals is Dart's a == b for ValueNotifier's values.
func dartEquals[T any](a, b T) bool {
	if eq, ok := any(a).(interface{ Equal(T) bool }); ok {
		return eq.Equal(b)
	}
	return valuesEqual(any(a), any(b))
}

func valuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return isNull(a) && isNull(b)
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb {
		return false
	}
	if ta.Comparable() {
		return a == b
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	switch ta.Kind() {
	case reflect.Slice: // a Dart List compares by identity
		return va.Len() == vb.Len() && (va.Len() == 0 && va.IsNil() == vb.IsNil() || va.Pointer() == vb.Pointer())
	case reflect.Map:
		return va.Pointer() == vb.Pointer()
	case reflect.Func:
		return dart.FunctionsEqual(a, b) || va.Pointer() == 0 && vb.Pointer() == 0
	}
	// A struct holding uncomparable fields: Dart's default == is identity,
	// and a Go value has none.
	return false
}

// isNull reports whether v is Dart's null: nil, or a nil pointer, map,
// slice, func or channel in an interface.
func isNull(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

// describeIdentity is foundation's describeIdentity (debug builds: the
// runtime type; otherwise "<optimized out>") and the short hash.
func describeIdentity(object any) string {
	typ := "<optimized out>"
	if debugMode {
		typ = runtimeTypeName(object)
	}
	return typ + "#" + fmt.Sprintf("%05x", dart.IdentityHashCode(object)&0xFFFFF)
}

// toString is Dart's `object.toString()` (string interpolation).
func toString(v any) string {
	if isNull(v) {
		return "null"
	}
	switch x := v.(type) {
	case time.Duration:
		return dart.DurationToString(x)
	case fmt.Stringer:
		if isEnum(v) {
			return runtimeTypeName(v) + "." + x.String()
		}
		return x.String()
	case error:
		return x.Error()
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String()
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return dart.DoubleToString(rv.Float())
	case reflect.Pointer:
		if rv.Elem().Kind() != reflect.Struct {
			return toString(rv.Elem().Interface()) // a nullable value
		}
	case reflect.Slice, reflect.Array:
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = toString(rv.Index(i).Interface())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case reflect.Func:
		return "Closure: " + runtimeTypeName(v)
	}
	return "Instance of '" + runtimeTypeName(v) + "'"
}

// isEnum reports whether v is a Go enum per API_MAPPING: a named integer
// type with a String method.
func isEnum(v any) bool {
	t := reflect.TypeOf(v)
	if t.PkgPath() == "" {
		return false
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		_, ok := v.(fmt.Stringer)
		return ok
	}
	return false
}

// runtimeTypeName is Dart's `object.runtimeType.toString()`: pointers are
// dropped, generic arguments use <> and Dart's names for builtin types.
// ChangeNotifierMixin is Dart's ChangeNotifier.
func runtimeTypeName(v any) string {
	if v == nil {
		return "Null"
	}
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeFor[ChangeNotifierMixin]() {
		return "ChangeNotifier"
	}
	if t.Name() == "" || t.PkgPath() == "" {
		return dartTypeString(t.String())
	}
	return dartTypeString(t.Name())
}

// dartTypeString converts the textual form of a Go type to Dart's spelling.
func dartTypeString(s string) string {
	s = strings.TrimLeft(strings.TrimSpace(s), "*")
	switch s {
	case "float64", "float32":
		return "double"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return "int"
	case "string":
		return "String"
	case "bool":
		return "bool"
	case "interface {}", "any":
		return "Object"
	}
	switch {
	case strings.HasPrefix(s, "func("):
		return "Function"
	case strings.HasPrefix(s, "[]"):
		return "List<" + dartTypeString(s[2:]) + ">"
	}
	head, args := s, ""
	if i := strings.IndexByte(s, '['); i >= 0 && strings.HasSuffix(s, "]") {
		head, args = s[:i], s[i+1:len(s)-1]
	}
	if i := strings.LastIndexByte(head, '/'); i >= 0 {
		head = head[i+1:]
	}
	if i := strings.IndexByte(head, '.'); i >= 0 {
		head = head[i+1:]
	}
	if args == "" {
		return head
	}
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, dartTypeString(args[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, dartTypeString(args[start:]))
	return head + "<" + strings.Join(out, ", ") + ">"
}
