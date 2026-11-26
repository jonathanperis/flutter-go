// Port of package:listen 1.0.1's test/listen_test.dart: all 25 tests.
//
// "Listenable.onError default implementation rethrows and preserves
// StackTrace" checks the rethrown value only: a Go panic carries no stack
// trace, so the original one is not preserved (API_MAPPING divergence).
package listen

import (
	"errors"
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// TestNotifier is `class TestNotifier extends ChangeNotifier`.
type TestNotifier struct{ ChangeNotifierMixin }

func NewTestNotifier() *TestNotifier {
	n := &TestNotifier{}
	n.SetSelf(n)
	return n
}

func (n *TestNotifier) Notify()            { n.NotifyListeners() }
func (n *TestNotifier) IsListenedTo() bool { return n.HasListeners() }

// HasListenersTester is `class HasListenersTester<T> extends ValueNotifier<T>`.
type HasListenersTester[T any] struct{ ValueNotifier[T] }

func NewHasListenersTester[T any](value T) *HasListenersTester[T] {
	h := &HasListenersTester[T]{}
	h.Construct(h, value)
	return h
}

func (h *HasListenersTester[T]) TestHasListeners() bool { return h.HasListeners() }

type A struct{ result bool }

func (a *A) Test() { a.result = true }

// B is `class B extends A with ChangeNotifier`, overriding test.
type B struct {
	A
	ChangeNotifierMixin
}

func NewB() *B {
	b := &B{}
	b.SetSelf(b)
	return b
}

func (b *B) Test() {
	b.NotifyListeners()
	b.A.Test()
}

// Counter is `class Counter with ChangeNotifier`.
type Counter struct {
	ChangeNotifierMixin
	value int
}

func NewCounter() *Counter {
	c := &Counter{}
	c.SetSelf(c)
	ChangeNotifierMaybeDispatchObjectCreation(c)
	return c
}

func (c *Counter) Value() int { return c.value }
func (c *Counter) SetValue(value int) {
	if c.value != value {
		c.value = value
		c.NotifyListeners()
	}
}
func (c *Counter) Notify() { c.NotifyListeners() }

// formatException stands for dart:core's FormatException.
type formatException struct{ message string }

func (e formatException) Error() string { return "FormatException: " + e.message }

// errorState is the test file's setUp/tearDown: onError records the last
// error, and a test that leaves one unhandled fails.
type errorState struct {
	lastError   any
	lastContext *ErrorContext
}

func setUp(t *testing.T) *errorState {
	t.Helper()
	s := &errorState{}
	original := ListenableOnError
	ListenableOnError = func(err any, _ dart.StackTrace, context ErrorContext) {
		s.lastError = err
		s.lastContext = &context
	}
	t.Cleanup(func() {
		ListenableOnError = original
		if s.lastError != nil {
			t.Errorf("Unexpected error in test: %v", s.lastError)
		}
	})
	return s
}

// replaceOnError is the tests' own `Listenable.onError = ...` with
// addTearDown restoring it.
func replaceOnError(t *testing.T, f ErrorCallback) {
	original := ListenableOnError
	ListenableOnError = f
	t.Cleanup(func() { ListenableOnError = original })
}

func errorString(err any) string {
	if e, ok := err.(error); ok {
		return e.Error()
	}
	return toString(err)
}

func expectLog(t *testing.T, log *[]string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	got := *log
	if got == nil {
		got = []string{}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("log = %q, want %q", got, want)
	}
	*log = nil
}

func TestChangeNotifierCanNotDisposeInCallback(t *testing.T) {
	s := setUp(t)
	test := NewTestNotifier()
	callbackDidFinish := false
	foo := func() {
		test.Dispose()
		callbackDidFinish = true
	}

	test.AddListener(foo)

	test.Notify()

	if _, ok := s.lastError.(*dart.AssertionError); !ok {
		t.Fatalf("lastError = %#v, want an AssertionError", s.lastError)
	}
	if *s.lastContext != ErrorContextListener {
		t.Fatal(s.lastContext)
	}
	if !strings.Contains(errorString(s.lastError), "dispose()") {
		t.Fatal(s.lastError)
	}
	// Make sure it crashes during dispose call.
	if callbackDidFinish {
		t.Fatal("callback finished")
	}
	test.Dispose()
	s.lastError = nil
}

func TestChangeNotifier(t *testing.T) {
	setUp(t)
	var log []string
	listener := func() { log = append(log, "listener") }
	listener1 := func() { log = append(log, "listener1") }
	listener2 := func() { log = append(log, "listener2") }
	badListener := func() {
		log = append(log, "badListener")
		panic(dart.ArgumentError{})
	}

	test := NewTestNotifier()

	var errs []any
	replaceOnError(t, func(err any, _ dart.StackTrace, _ ErrorContext) { errs = append(errs, err) })
	takeArgumentError := func() {
		t.Helper()
		if len(errs) == 0 {
			t.Fatal("no error")
		}
		if _, ok := errs[0].(dart.ArgumentError); !ok {
			t.Fatalf("error %#v is not an ArgumentError", errs[0])
		}
		errs = errs[1:]
	}

	test.AddListener(listener)
	test.AddListener(listener)
	test.Notify()
	expectLog(t, &log, "listener", "listener")

	test.RemoveListener(listener)
	test.Notify()
	expectLog(t, &log, "listener")

	test.RemoveListener(listener)
	test.Notify()
	expectLog(t, &log)

	test.RemoveListener(listener)
	test.Notify()
	expectLog(t, &log)

	test.AddListener(listener)
	test.Notify()
	expectLog(t, &log, "listener")

	test.AddListener(listener1)
	test.Notify()
	expectLog(t, &log, "listener", "listener1")

	test.AddListener(listener2)
	test.Notify()
	expectLog(t, &log, "listener", "listener1", "listener2")

	test.RemoveListener(listener1)
	test.Notify()
	expectLog(t, &log, "listener", "listener2")

	test.AddListener(listener1)
	test.Notify()
	expectLog(t, &log, "listener", "listener2", "listener1")

	test.AddListener(badListener)
	test.Notify()
	expectLog(t, &log, "listener", "listener2", "listener1", "badListener")
	takeArgumentError()

	test.AddListener(listener1)
	test.RemoveListener(listener)
	test.RemoveListener(listener1)
	test.RemoveListener(listener2)
	test.AddListener(listener2)
	test.Notify()
	expectLog(t, &log, "badListener", "listener1", "listener2")
	takeArgumentError()
	test.Dispose()
}

func TestListenableOnErrorPreservesRuntimeExceptionTypes(t *testing.T) {
	setUp(t)
	var errs []any
	replaceOnError(t, func(err any, _ dart.StackTrace, _ ErrorContext) { errs = append(errs, err) })

	notifier := NewTestNotifier()
	notifier.AddListener(func() { panic(formatException{"custom exception"}) })
	notifier.Notify()

	if len(errs) != 1 {
		t.Fatal(errs)
	}
	if e, ok := errs[0].(formatException); !ok || e.message != "custom exception" {
		t.Fatalf("%#v", errs[0])
	}
}

func TestListenableOnErrorDefaultRethrows(t *testing.T) {
	stack := dart.StackTraceFromString("test_stack_trace_marker")
	func() {
		defer func() {
			e := recover()
			if _, ok := e.(formatException); !ok {
				t.Fatalf("recovered %#v, want the FormatException", e)
			}
			// Dart also checks that the rethrown stack is stack: a Go panic
			// carries none (see the file comment).
		}()
		defaultOnError(formatException{"bad format"}, stack, ErrorContextListener)
		t.Fatal("Expected originalOnError to throw")
	}()
}

func TestListenableOnErrorReportsCorrectErrorContext(t *testing.T) {
	setUp(t)
	var contexts []ErrorContext
	replaceOnError(t, func(_ any, _ dart.StackTrace, context ErrorContext) { contexts = append(contexts, context) })
	expectSingle := func(want ErrorContext) {
		t.Helper()
		if len(contexts) != 1 || contexts[0] != want {
			t.Fatalf("contexts = %v, want [%v]", contexts, want)
		}
		contexts = nil
	}

	notifier := NewTestNotifier()
	notifier.AddListener(func() { panic(errors.New("Exception: listener exception")) })
	notifier.Notify()
	expectSingle(ErrorContextListener)

	notifier.Dispose()

	notifier.Dispose()
	expectSingle(ErrorContextAssertion)

	notifier.Notify()
	expectSingle(ErrorContextAssertion)

	notifier.AddListener(func() {})
	expectSingle(ErrorContextAssertion)

	ChangeNotifierDebugAssertNotDisposed(notifier)
	expectSingle(ErrorContextAssertion)
}

func TestChangeNotifierWithMutatingListener(t *testing.T) {
	setUp(t)
	test := NewTestNotifier()
	var log []string

	listener1 := func() { log = append(log, "listener1") }
	listener3 := func() { log = append(log, "listener3") }
	listener4 := func() { log = append(log, "listener4") }
	listener2 := func() {
		log = append(log, "listener2")
		test.RemoveListener(listener1)
		test.RemoveListener(listener3)
		test.AddListener(listener4)
	}

	test.AddListener(listener1)
	test.AddListener(listener2)
	test.AddListener(listener3)
	test.Notify()
	expectLog(t, &log, "listener1", "listener2")

	test.Notify()
	expectLog(t, &log, "listener2", "listener4")

	test.Notify()
	expectLog(t, &log, "listener2", "listener4", "listener4")
}

func TestListenerAddedAndRemovedImmediatelyDuringNotifyListeners(t *testing.T) {
	setUp(t)
	source := NewTestNotifier()
	var log []string

	listener3 := func() { log = append(log, "listener3") }
	listener2 := func() { log = append(log, "listener2") }
	listener1 := func() {
		log = append(log, "listener1")
		source.AddListener(listener2)
		source.RemoveListener(listener2)
		source.AddListener(listener3)
	}

	source.AddListener(listener1)

	source.Notify()

	expectLog(t, &log, "listener1")
}

func TestListenerInTheMiddleRemovingItself(t *testing.T) {
	setUp(t)
	source := NewTestNotifier()
	var log []string

	var selfRemovingListener func()
	selfRemovingListener = func() {
		log = append(log, "selfRemovingListener")
		source.RemoveListener(selfRemovingListener)
	}
	listener1 := func() { log = append(log, "listener1") }

	source.AddListener(listener1)
	source.AddListener(selfRemovingListener)
	source.AddListener(listener1)

	source.Notify()

	expectLog(t, &log, "listener1", "selfRemovingListener", "listener1")
}

func TestFirstListenerRemovingItself(t *testing.T) {
	setUp(t)
	source := NewTestNotifier()
	var log []string

	var selfRemovingListener func()
	selfRemovingListener = func() {
		log = append(log, "selfRemovingListener")
		source.RemoveListener(selfRemovingListener)
	}
	listener1 := func() { log = append(log, "listener1") }

	source.AddListener(selfRemovingListener)
	source.AddListener(listener1)

	source.NotifyListeners()

	expectLog(t, &log, "selfRemovingListener", "listener1")
}

func listenables(l ...Listenable) iter.Seq[Listenable] { return slices.Values(l) }

func TestMergingChangeNotifiers(t *testing.T) {
	setUp(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	source3 := NewTestNotifier()
	var log []string

	merged := ListenableMerge(listenables(source1, source2))
	listener1 := func() { log = append(log, "listener1") }
	listener2 := func() { log = append(log, "listener2") }

	merged.AddListener(listener1)
	source1.Notify()
	source2.Notify()
	source3.Notify()
	expectLog(t, &log, "listener1", "listener1")

	merged.RemoveListener(listener1)
	source1.Notify()
	source2.Notify()
	source3.Notify()
	expectLog(t, &log)

	merged.AddListener(listener1)
	merged.AddListener(listener2)
	source1.Notify()
	source2.Notify()
	source3.Notify()
	expectLog(t, &log, "listener1", "listener2", "listener1", "listener2")
}

func TestMergingChangeNotifiersSupportsAnyIterable(t *testing.T) {
	setUp(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	var log []string

	// Dart passes a Set: any sequence works.
	set := func(yield func(Listenable) bool) {
		_ = yield(source1) && yield(source2)
	}
	merged := ListenableMerge(set)
	listener := func() { log = append(log, "listener") }

	merged.AddListener(listener)
	source1.Notify()
	source2.Notify()
	expectLog(t, &log, "listener", "listener")
}

func TestMergingChangeNotifiersIgnoresNull(t *testing.T) {
	setUp(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	var log []string

	var typedNil *TestNotifier
	merged := ListenableMerge(listenables(nil, source1, typedNil, source2, nil))
	listener := func() { log = append(log, "listener") }

	merged.AddListener(listener)
	source1.Notify()
	source2.Notify()
	expectLog(t, &log, "listener", "listener")
	merged.RemoveListener(listener)
}

func TestCanRemoveFromMergedNotifier(t *testing.T) {
	setUp(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	var log []string

	merged := ListenableMerge(listenables(source1, source2))
	listener := func() { log = append(log, "listener") }

	merged.AddListener(listener)
	source1.Notify()
	source2.Notify()
	expectLog(t, &log, "listener", "listener")

	merged.RemoveListener(listener)
	source1.Notify()
	source2.Notify()
	expectLog(t, &log)
}

func TestCannotUseADisposedChangeNotifierExceptForRemoveListener(t *testing.T) {
	s := setUp(t)
	source := NewTestNotifier()
	source.Dispose()
	const want = "TestNotifier was used after being disposed."

	source.AddListener(func() {})
	if !strings.Contains(errorString(s.lastError), want) {
		t.Fatal(s.lastError)
	}
	s.lastError = nil

	source.Dispose()
	if !strings.Contains(errorString(s.lastError), want) {
		t.Fatal(s.lastError)
	}
	s.lastError = nil

	source.Notify()
	if !strings.Contains(errorString(s.lastError), want) {
		t.Fatal(s.lastError)
	}
	s.lastError = nil
}

func TestCanRemoveListenerOnADisposedChangeNotifier(t *testing.T) {
	setUp(t)
	source := NewTestNotifier()
	source.Dispose()
	source.RemoveListener(func() {})
}

func TestCanCheckHasListenerOnADisposedChangeNotifier(t *testing.T) {
	setUp(t)
	source := NewHasListenersTester(0)
	source.AddListener(func() {})
	if !source.TestHasListeners() {
		t.Fatal("no listeners")
	}
	source.Dispose()
	if source.TestHasListeners() {
		t.Fatal("listeners after dispose")
	}
}

func TestValueNotifier(t *testing.T) {
	setUp(t)
	notifier := NewValueNotifier(2.0)

	var log []float64
	listener := func() { log = append(log, notifier.Value()) }

	notifier.AddListener(listener)
	notifier.SetValue(3.0)

	if !slices.Equal(log, []float64{3.0}) {
		t.Fatal(log)
	}
	log = nil

	notifier.SetValue(3.0)
	if len(log) != 0 {
		t.Fatal(log)
	}
}

func TestListenableMergeToString(t *testing.T) {
	setUp(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()

	for _, c := range []struct {
		merged Listenable
		want   string
	}{
		{ListenableMerge(listenables()), "Listenable.merge([])"},
		{ListenableMerge(listenables(nil)), "Listenable.merge([null])"},
		{ListenableMerge(listenables(source1)), "Listenable.merge([Instance of 'TestNotifier'])"},
		{ListenableMerge(listenables(source1, source2)), "Listenable.merge([Instance of 'TestNotifier', Instance of 'TestNotifier'])"},
		{ListenableMerge(listenables(nil, source2)), "Listenable.merge([null, Instance of 'TestNotifier'])"},
	} {
		if got := toString(c.merged); got != c.want {
			t.Errorf("toString = %q, want %q", got, c.want)
		}
	}
}

func TestListenableMergeDoesNotLeak(t *testing.T) {
	setUp(t)
	// Regression test for https://github.com/flutter/flutter/issues/25163.
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	fakeListener := func() {}

	listenableUnderTest := ListenableMerge(listenables(source1, source2))
	if source1.IsListenedTo() || source2.IsListenedTo() {
		t.Fatal("listened to")
	}
	listenableUnderTest.AddListener(fakeListener)
	if !source1.IsListenedTo() || !source2.IsListenedTo() {
		t.Fatal("not listened to")
	}

	listenableUnderTest.RemoveListener(fakeListener)
	if source1.IsListenedTo() || source2.IsListenedTo() {
		t.Fatal("still listened to")
	}
}

func TestHasListeners(t *testing.T) {
	setUp(t)
	notifier := NewHasListenersTester(true)
	expect := func(want bool) {
		t.Helper()
		if notifier.TestHasListeners() != want {
			t.Fatalf("hasListeners = %v", !want)
		}
	}
	expect(false)
	n := 0
	test1 := func() { n++ }
	test2 := func() { n-- }
	notifier.AddListener(test1)
	expect(true)
	notifier.AddListener(test1)
	expect(true)
	notifier.RemoveListener(test1)
	expect(true)
	notifier.RemoveListener(test1)
	expect(false)
	notifier.AddListener(test1)
	expect(true)
	notifier.AddListener(test2)
	expect(true)
	notifier.RemoveListener(test1)
	expect(true)
	notifier.RemoveListener(test2)
	expect(false)
}

func TestChangeNotifierAsAMixin(t *testing.T) {
	setUp(t)
	// We document that this is a valid way to use this class.
	b := NewB()
	notifications := 0
	b.AddListener(func() { notifications++ })
	if b.result || notifications != 0 {
		t.Fatal("before")
	}
	b.Test()
	if !b.result || notifications != 1 {
		t.Fatal("after")
	}
}

func TestThrowsWhenDisposedAndCalled(t *testing.T) {
	s := setUp(t)
	testNotifier := NewTestNotifier()
	testNotifier.Dispose()

	testNotifier.Dispose()

	if !strings.Contains(errorString(s.lastError), "TestNotifier was used after being disposed.") {
		t.Fatal(s.lastError)
	}
	s.lastError = nil
}

func TestCallingDebugAssertNotDisposedWorksAsIntended(t *testing.T) {
	s := setUp(t)
	testNotifier := NewTestNotifier()
	if !ChangeNotifierDebugAssertNotDisposed(testNotifier) {
		t.Fatal("false")
	}
	testNotifier.Dispose()

	ChangeNotifierDebugAssertNotDisposed(testNotifier)

	if !strings.Contains(errorString(s.lastError), "TestNotifier was used after being disposed.") {
		t.Fatal(s.lastError)
	}
	s.lastError = nil
}

func TestNotifyListenerCanBeCalledRecursively(t *testing.T) {
	setUp(t)
	counter := NewCounter()
	var log []string

	listener1 := func() {
		log = append(log, "listener1")
		if counter.Value() < 0 {
			counter.SetValue(0)
		}
	}

	counter.AddListener(listener1)
	counter.Notify()
	expectLog(t, &log, "listener1")

	counter.SetValue(3)
	expectLog(t, &log, "listener1")

	counter.SetValue(-2)
	expectLog(t, &log, "listener1", "listener1")
}

func TestRemoveListenersWhileNotifyingOnAListWhichWillNotResize(t *testing.T) {
	setUp(t)
	test := NewTestNotifier()
	var log []string
	var listeners []VoidCallback

	var autoRemove func()
	autoRemove = func() {
		// We remove 4 listeners.
		// We will end up with (13-4 = 9) listeners.
		test.RemoveListener(listeners[1])
		test.RemoveListener(listeners[3])
		test.RemoveListener(listeners[4])
		test.RemoveListener(autoRemove)
	}

	test.AddListener(autoRemove)

	// We add 12 more listeners.
	for i := 0; i < 12; i++ {
		listener := func() { log = append(log, "listener"+itoa(i)) }
		listeners = append(listeners, listener)
		test.AddListener(listener)
	}

	remainingListenerIndexes := []int{0, 2, 5, 6, 7, 8, 9, 10, 11}
	var expectedLog []string
	for _, i := range remainingListenerIndexes {
		expectedLog = append(expectedLog, "listener"+itoa(i))
	}

	test.Notify()
	expectLog(t, &log, expectedLog...)

	// We expect to have the same result after the removal of previous listeners.
	test.Notify()
	expectLog(t, &log, expectedLog...)

	// We remove all other listeners.
	for _, i := range remainingListenerIndexes {
		test.RemoveListener(listeners[i])
	}

	test.Notify()
	expectLog(t, &log)
}

func itoa(i int) string { return toString(i) }
