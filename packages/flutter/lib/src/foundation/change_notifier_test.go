// Port of packages/flutter/test/foundation/change_notifier_test.dart: all 22
// tests, through foundation's re-exported names (listen_exports.go).
//
// Flutter runs them with the test binding initialized, so package:listen's
// errors go through BindingBase's _initListenable: listener errors are
// reported (tester.takeException) and a disposed notifier throws a
// FlutterError. Here each test installs initListenable's onError and
// collects the reported errors with a replaced FlutterErrorOnError.
package foundation

import (
	"iter"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/pub/listen"
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

type cnA struct{ result bool }

func (a *cnA) Test() { a.result = true }

// cnB is `class B extends A with ChangeNotifier`.
type cnB struct {
	cnA
	ChangeNotifierMixin
}

func newCnB() *cnB {
	b := &cnB{}
	b.SetSelf(b)
	if KFlutterMemoryAllocationsEnabled {
		ChangeNotifierMaybeDispatchObjectCreation(b)
	}
	return b
}

func (b *cnB) Test() {
	b.NotifyListeners()
	b.cnA.Test()
}

// cnCounter is `class Counter with ChangeNotifier`.
type cnCounter struct {
	ChangeNotifierMixin
	value int
}

func newCnCounter() *cnCounter {
	c := &cnCounter{}
	c.SetSelf(c)
	if KFlutterMemoryAllocationsEnabled {
		ChangeNotifierMaybeDispatchObjectCreation(c)
	}
	return c
}

func (c *cnCounter) Value() int { return c.value }
func (c *cnCounter) SetValue(value int) {
	if c.value != value {
		c.value = value
		c.NotifyListeners()
	}
}
func (c *cnCounter) Notify() { c.NotifyListeners() }

// withBindingOnError installs the binding's Listenable.onError and returns
// tester.takeException: the next reported exception (nil when none).
func withBindingOnError(t *testing.T) (takeException func() any) {
	t.Helper()
	originalListen := listen.ListenableOnError
	originalFlutter := FlutterErrorOnError
	var reported []*FlutterErrorDetails
	initListenable()
	FlutterErrorOnError = func(details *FlutterErrorDetails) { reported = append(reported, details) }
	t.Cleanup(func() {
		listen.ListenableOnError = originalListen
		FlutterErrorOnError = originalFlutter
		if len(reported) > 0 {
			t.Errorf("unhandled exceptions: %v", reported[0].Exception)
		}
	})
	return func() any {
		if len(reported) == 0 {
			return nil
		}
		e := reported[0].Exception
		reported = reported[1:]
		return e
	}
}

func cnExpectLog(t *testing.T, log *[]string, want ...string) {
	t.Helper()
	if !slices.Equal(*log, want) && !(len(*log) == 0 && len(want) == 0) {
		t.Fatalf("log = %q, want %q", *log, want)
	}
	*log = nil
}

func cnListenables(l ...Listenable) iter.Seq[Listenable] { return slices.Values(l) }

// throwsFlutterError runs f and returns the FlutterError it panics with.
func throwsFlutterError(t *testing.T, f func()) (err *FlutterError) {
	t.Helper()
	defer func() {
		r := recover()
		e, ok := r.(*FlutterError)
		if !ok {
			t.Fatalf("recovered %#v, want a FlutterError", r)
		}
		err = e
	}()
	f()
	return nil
}

func TestChangeNotifierCanNotDisposeInCallback(t *testing.T) {
	takeException := withBindingOnError(t)
	test := NewTestNotifier()
	callbackDidFinish := false
	foo := func() {
		test.Dispose()
		callbackDidFinish = true
	}

	test.AddListener(foo)

	test.Notify()

	err, ok := takeException().(*dart.AssertionError)
	if !ok || !strings.Contains(err.Error(), "dispose()") {
		t.Fatalf("%#v", err)
	}
	// Make sure it crashes during dispose call.
	if callbackDidFinish {
		t.Fatal("callback finished")
	}
	test.Dispose()
}

func TestChangeNotifier(t *testing.T) {
	takeException := withBindingOnError(t)
	var log []string
	listener := func() { log = append(log, "listener") }
	listener1 := func() { log = append(log, "listener1") }
	listener2 := func() { log = append(log, "listener2") }
	badListener := func() {
		log = append(log, "badListener")
		panic(dart.ArgumentError{})
	}
	isArgumentError := func() {
		t.Helper()
		if _, ok := takeException().(dart.ArgumentError); !ok {
			t.Fatal("no ArgumentError reported")
		}
	}

	test := NewTestNotifier()

	test.AddListener(listener)
	test.AddListener(listener)
	test.Notify()
	cnExpectLog(t, &log, "listener", "listener")

	test.RemoveListener(listener)
	test.Notify()
	cnExpectLog(t, &log, "listener")

	test.RemoveListener(listener)
	test.Notify()
	cnExpectLog(t, &log)

	test.RemoveListener(listener)
	test.Notify()
	cnExpectLog(t, &log)

	test.AddListener(listener)
	test.Notify()
	cnExpectLog(t, &log, "listener")

	test.AddListener(listener1)
	test.Notify()
	cnExpectLog(t, &log, "listener", "listener1")

	test.AddListener(listener2)
	test.Notify()
	cnExpectLog(t, &log, "listener", "listener1", "listener2")

	test.RemoveListener(listener1)
	test.Notify()
	cnExpectLog(t, &log, "listener", "listener2")

	test.AddListener(listener1)
	test.Notify()
	cnExpectLog(t, &log, "listener", "listener2", "listener1")

	test.AddListener(badListener)
	test.Notify()
	cnExpectLog(t, &log, "listener", "listener2", "listener1", "badListener")
	isArgumentError()

	test.AddListener(listener1)
	test.RemoveListener(listener)
	test.RemoveListener(listener1)
	test.RemoveListener(listener2)
	test.AddListener(listener2)
	test.Notify()
	cnExpectLog(t, &log, "badListener", "listener1", "listener2")
	isArgumentError()
	test.Dispose()
}

func TestChangeNotifierWithMutatingListener(t *testing.T) {
	withBindingOnError(t)
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
	cnExpectLog(t, &log, "listener1", "listener2")

	test.Notify()
	cnExpectLog(t, &log, "listener2", "listener4")

	test.Notify()
	cnExpectLog(t, &log, "listener2", "listener4", "listener4")
}

func TestChangeNotifierListenerAddedAndRemovedImmediately(t *testing.T) {
	withBindingOnError(t)
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

	cnExpectLog(t, &log, "listener1")
}

func TestChangeNotifierListenerInTheMiddleRemovesItself(t *testing.T) {
	withBindingOnError(t)
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

	cnExpectLog(t, &log, "listener1", "selfRemovingListener", "listener1")
}

func TestChangeNotifierFirstListenerRemovesItself(t *testing.T) {
	withBindingOnError(t)
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

	cnExpectLog(t, &log, "selfRemovingListener", "listener1")
}

func TestMergingChangeNotifiers(t *testing.T) {
	withBindingOnError(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	source3 := NewTestNotifier()
	var log []string

	merged := ListenableMerge(cnListenables(source1, source2))
	listener1 := func() { log = append(log, "listener1") }
	listener2 := func() { log = append(log, "listener2") }

	merged.AddListener(listener1)
	source1.Notify()
	source2.Notify()
	source3.Notify()
	cnExpectLog(t, &log, "listener1", "listener1")

	merged.RemoveListener(listener1)
	source1.Notify()
	source2.Notify()
	source3.Notify()
	cnExpectLog(t, &log)

	merged.AddListener(listener1)
	merged.AddListener(listener2)
	source1.Notify()
	source2.Notify()
	source3.Notify()
	cnExpectLog(t, &log, "listener1", "listener2", "listener1", "listener2")
}

func TestMergingChangeNotifiersSupportsAnyIterable(t *testing.T) {
	withBindingOnError(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	var log []string

	set := func(yield func(Listenable) bool) { _ = yield(source1) && yield(source2) }
	merged := ListenableMerge(set)
	listener := func() { log = append(log, "listener") }

	merged.AddListener(listener)
	source1.Notify()
	source2.Notify()
	cnExpectLog(t, &log, "listener", "listener")
}

func TestMergingChangeNotifiersIgnoresNull(t *testing.T) {
	withBindingOnError(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	var log []string

	merged := ListenableMerge(cnListenables(nil, source1, nil, source2, nil))
	listener := func() { log = append(log, "listener") }

	merged.AddListener(listener)
	source1.Notify()
	source2.Notify()
	cnExpectLog(t, &log, "listener", "listener")
}

func TestCanRemoveFromMergedNotifier(t *testing.T) {
	withBindingOnError(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	var log []string

	merged := ListenableMerge(cnListenables(source1, source2))
	listener := func() { log = append(log, "listener") }

	merged.AddListener(listener)
	source1.Notify()
	source2.Notify()
	cnExpectLog(t, &log, "listener", "listener")

	merged.RemoveListener(listener)
	source1.Notify()
	source2.Notify()
	cnExpectLog(t, &log)
}

func TestCannotUseADisposedChangeNotifierExceptForRemoveListener(t *testing.T) {
	withBindingOnError(t)
	source := NewTestNotifier()
	source.Dispose()
	throwsFlutterError(t, func() { source.AddListener(func() {}) })
	throwsFlutterError(t, func() { source.Dispose() })
	throwsFlutterError(t, func() { source.Notify() })
}

func TestCanRemoveListenerOnADisposedChangeNotifier(t *testing.T) {
	withBindingOnError(t)
	source := NewTestNotifier()
	source.Dispose()
	source.RemoveListener(func() {})
}

func TestCanCheckHasListenerOnADisposedChangeNotifier(t *testing.T) {
	withBindingOnError(t)
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

func TestValueNotifierNotifiesOnChange(t *testing.T) {
	withBindingOnError(t)
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
	withBindingOnError(t)
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()

	for _, c := range []struct {
		merged Listenable
		want   string
	}{
		{ListenableMerge(cnListenables()), "Listenable.merge([])"},
		{ListenableMerge(cnListenables(nil)), "Listenable.merge([null])"},
		{ListenableMerge(cnListenables(source1)), "Listenable.merge([Instance of 'TestNotifier'])"},
		{ListenableMerge(cnListenables(source1, source2)), "Listenable.merge([Instance of 'TestNotifier', Instance of 'TestNotifier'])"},
		{ListenableMerge(cnListenables(nil, source2)), "Listenable.merge([null, Instance of 'TestNotifier'])"},
	} {
		if got := dartToString(c.merged); got != c.want {
			t.Errorf("toString = %q, want %q", got, c.want)
		}
	}
}

func TestListenableMergeDoesNotLeak(t *testing.T) {
	withBindingOnError(t)
	// Regression test for https://github.com/flutter/flutter/issues/25163.
	source1 := NewTestNotifier()
	source2 := NewTestNotifier()
	fakeListener := func() {}

	listenableUnderTest := ListenableMerge(cnListenables(source1, source2))
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

func TestChangeNotifierHasListeners(t *testing.T) {
	withBindingOnError(t)
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
	withBindingOnError(t)
	// We document that this is a valid way to use this class.
	b := newCnB()
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

const disposedTestNotifierDeep = "FlutterError\n" +
	"   A TestNotifier was used after being disposed.\n" +
	"   Once you have called dispose() on a TestNotifier, it can no\n" +
	"   longer be used.\n"

func TestThrowsFlutterErrorWhenDisposedAndCalled(t *testing.T) {
	withBindingOnError(t)
	testNotifier := NewTestNotifier()
	testNotifier.Dispose()
	err := throwsFlutterError(t, func() { testNotifier.Dispose() })
	if got := err.ToStringDeep(); got != disposedTestNotifierDeep {
		t.Fatalf("toStringDeep = %q", got)
	}
}

func TestCallingDebugAssertNotDisposedWorksAsIntended(t *testing.T) {
	withBindingOnError(t)
	testNotifier := NewTestNotifier()
	if !ChangeNotifierDebugAssertNotDisposed(testNotifier) {
		t.Fatal("false")
	}
	testNotifier.Dispose()
	err := throwsFlutterError(t, func() { ChangeNotifierDebugAssertNotDisposed(testNotifier) })
	if got := err.ToStringDeep(); got != disposedTestNotifierDeep {
		t.Fatalf("toStringDeep = %q", got)
	}
}

func TestNotifyListenerCanBeCalledRecursively(t *testing.T) {
	withBindingOnError(t)
	counter := newCnCounter()
	var log []string

	listener1 := func() {
		log = append(log, "listener1")
		if counter.Value() < 0 {
			counter.SetValue(0)
		}
	}

	counter.AddListener(listener1)
	counter.Notify()
	cnExpectLog(t, &log, "listener1")

	counter.SetValue(3)
	cnExpectLog(t, &log, "listener1")

	counter.SetValue(-2)
	cnExpectLog(t, &log, "listener1", "listener1")
}

func TestRemoveListenersWhileNotifyingOnAListWhichWillNotResize(t *testing.T) {
	withBindingOnError(t)
	test := NewTestNotifier()
	var log []string
	var listeners []func()

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
		listener := func() { log = append(log, "listener"+strconv.Itoa(i)) }
		listeners = append(listeners, listener)
		test.AddListener(listener)
	}

	remainingListenerIndexes := []int{0, 2, 5, 6, 7, 8, 9, 10, 11}
	var expectedLog []string
	for _, i := range remainingListenerIndexes {
		expectedLog = append(expectedLog, "listener"+strconv.Itoa(i))
	}

	test.Notify()
	cnExpectLog(t, &log, expectedLog...)

	// We expect to have the same result after the removal of previous listeners.
	test.Notify()
	cnExpectLog(t, &log, expectedLog...)

	// We remove all other listeners.
	for _, i := range remainingListenerIndexes {
		test.RemoveListener(listeners[i])
	}

	test.Notify()
	cnExpectLog(t, &log)
}

// Own test: a listener error reported through the binding's onError
// carries the library, context and stack Flutter gives it.
func TestListenableErrorReportedThroughBinding(t *testing.T) {
	originalListen := listen.ListenableOnError
	originalFlutter := FlutterErrorOnError
	t.Cleanup(func() { listen.ListenableOnError, FlutterErrorOnError = originalListen, originalFlutter })
	initListenable()
	var reported []*FlutterErrorDetails
	FlutterErrorOnError = func(d *FlutterErrorDetails) { reported = append(reported, d) }

	n := NewTestNotifier()
	n.AddListener(func() { panic("bad") })
	n.Notify()
	if len(reported) != 1 {
		t.Fatal(reported)
	}
	d := reported[0]
	if d.Exception != "bad" || *d.Library != "foundation library" ||
		d.Context.String() != "while dispatching notifications for Listenable" || d.Stack == nil ||
		!strings.Contains(d.Stack.String(), "change_notifier_test.go") {
		t.Fatalf("%v %v %v %v", d.Exception, *d.Library, d.Context, d.Stack)
	}

	// Assertion errors: a FlutterError is rethrown as is; a StateError
	// gives its message; an AssertionError its message (or its toString
	// without one); anything else its toString.
	fe := NewFlutterError("as is")
	for _, c := range []struct {
		err  any
		want string
	}{
		{dart.StateError{Message: "state"}, "state"},
		{dart.NewAssertionError("assert message"), "assert message"},
		{dart.NewAssertionError(nil), "Assertion failed"},
		{dart.ArgumentError{Message: "arg"}, "Invalid argument(s): arg"},
		{"text", "text"},
	} {
		err := throwsFlutterError(t, func() { listen.ListenableOnError(c.err, nil, listen.ErrorContextAssertion) })
		if err.Message() != c.want {
			t.Errorf("%#v: message %q, want %q", c.err, err.Message(), c.want)
		}
	}
	if err := throwsFlutterError(t, func() { listen.ListenableOnError(fe, nil, listen.ErrorContextAssertion) }); err != fe {
		t.Error("FlutterError not rethrown as is")
	}
}

// Own test: a ValueNotifier (also one embedded in another type) is a
// ValueListenable and a ChangeNotifier through foundation's names.
func TestValueNotifierIsAValueListenable(t *testing.T) {
	withBindingOnError(t)
	read := func(v ValueListenable[int]) int { return v.Value() }
	v := NewValueNotifier(4)
	h := NewHasListenersTester(5)
	if read(v) != 4 || read(h) != 5 {
		t.Fatal(read(v), read(h))
	}
	var notifiers []ChangeNotifier = []ChangeNotifier{v, h, NewChangeNotifier(), NewTestNotifier()}
	for _, n := range notifiers {
		n.Dispose()
		if n.HasListeners() {
			t.Fatal("listeners after dispose")
		}
	}
}

// Own test: foundation's describeIdentity and ValueNotifier.toString agree
// (both read Object.hashCode and runtimeType in Dart).
func TestValueNotifierToStringMatchesDescribeIdentity(t *testing.T) {
	v := NewValueNotifier(1)
	if got, want := v.String(), DescribeIdentity(v)+"(1)"; got != want {
		t.Fatalf("%q != %q", got, want)
	}
	if got := ObjectRuntimeType(v, "x"); got != "ValueNotifier<int>" {
		t.Fatal(got)
	}
}

// The binding installs the onError when it initializes its instances.
func TestBindingInstallsListenableOnError(t *testing.T) {
	originalListen := listen.ListenableOnError
	t.Cleanup(func() { listen.ListenableOnError = originalListen })
	listen.ListenableOnError = nil
	resetBindingStatics(t)
	newPlainBinding()
	if listen.ListenableOnError == nil {
		t.Fatal("onError not installed")
	}
	n := NewTestNotifier()
	n.Dispose()
	throwsFlutterError(t, func() { n.Dispose() })
}
