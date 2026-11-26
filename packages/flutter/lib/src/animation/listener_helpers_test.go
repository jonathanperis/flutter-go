package animation

// Ports of packages/flutter/test/animation/ (Flutter 3.50.1):
//   - listener_helpers_test.dart: 2/2
//   - animation_controller_listener_test.dart: 1/1
//   - animation_from_listener_test.dart: 3/3
//   - iteration_patterns_test.dart: 4/4 (the two testWidgets read
//     tester.takeException: here the reported errors are collected by setUp)

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// --- listener_helpers_test.dart

type testAnimationLocalStatusListeners struct {
	AnimationLocalStatusListenersMixin
	AnimationLazyListenerMixin
	callsToStart, callsToStop int
}

func newTestAnimationLocalStatusListeners() *testAnimationLocalStatusListeners {
	u := &testAnimationLocalStatusListeners{}
	u.AnimationLocalStatusListenersMixin.SetSelf(u)
	u.AnimationLazyListenerMixin.SetSelf(u)
	return u
}

func (u *testAnimationLocalStatusListeners) DidStartListening() { u.callsToStart++ }
func (u *testAnimationLocalStatusListeners) DidStopListening()  { u.callsToStop++ }

type testAnimationLocalListeners struct {
	AnimationLocalListenersMixin
	AnimationLazyListenerMixin
	callsToStart, callsToStop int
}

func newTestAnimationLocalListeners() *testAnimationLocalListeners {
	u := &testAnimationLocalListeners{}
	u.AnimationLocalListenersMixin.SetSelf(u)
	u.AnimationLazyListenerMixin.SetSelf(u)
	return u
}

func (u *testAnimationLocalListeners) DidStartListening() { u.callsToStart++ }
func (u *testAnimationLocalListeners) DidStopListening()  { u.callsToStop++ }

func TestRemovingUnregisteredStatusListenerIsNoOp(t *testing.T) {
	uut := newTestAnimationLocalStatusListeners()
	fakeListener := func(AnimationStatus) {}
	uut.RemoveStatusListener(fakeListener)
	expect(t, uut.callsToStart, 0)
	expect(t, uut.callsToStop, 0)
}

func TestRemovingUnregisteredListenerIsNoOp(t *testing.T) {
	uut := newTestAnimationLocalListeners()
	fakeListener := func() {}
	uut.RemoveListener(fakeListener)
	expect(t, uut.callsToStart, 0)
	expect(t, uut.callsToStop, 0)
}

// Own: the lazy mixin starts on the first listener and stops on the last,
// counting value and status listeners together.
func TestLazyListenerStartsAndStops(t *testing.T) {
	uut := newTestAnimationLocalListeners()
	l1, l2 := func() {}, func() { _ = 1 }
	uut.AddListener(l1)
	uut.AddListener(l2)
	expect(t, uut.callsToStart, 1)
	expect(t, uut.IsListening(), true)
	uut.RemoveListener(l1)
	expect(t, uut.callsToStop, 0)
	uut.RemoveListener(l2)
	expect(t, uut.callsToStop, 1)
	expect(t, uut.IsListening(), false)
	throwsAssertionError(t, func() { uut.DidUnregisterListener() })
}

// Own: HashedObserverList semantics (a listener added twice is called once
// and must be removed twice; re-adding moves it to the end) and
// ObserverList semantics for status listeners (called twice).
func TestListenerListSemantics(t *testing.T) {
	uut := newTestAnimationLocalListeners()
	var log []string
	a := func() { log = append(log, "a") }
	b := func() { log = append(log, "b") }
	uut.AddListener(a)
	uut.AddListener(b)
	uut.AddListener(a)
	uut.NotifyListeners()
	expectStrings(t, log, "a", "b")
	log = nil
	uut.RemoveListener(a)
	uut.NotifyListeners()
	expectStrings(t, log, "a", "b")
	log = nil
	uut.RemoveListener(a)
	uut.AddListener(a)
	uut.NotifyListeners()
	expectStrings(t, log, "b", "a")
	log = nil
	uut.ClearListeners()
	uut.NotifyListeners()
	expectStrings(t, log)

	s := newTestAnimationLocalStatusListeners()
	sa := func(AnimationStatus) { log = append(log, "sa") }
	s.AddStatusListener(sa)
	s.AddStatusListener(sa)
	s.NotifyStatusListeners(AnimationStatusForward)
	expectStrings(t, log, "sa", "sa")
	log = nil
	s.RemoveStatusListener(sa)
	s.NotifyStatusListeners(AnimationStatusForward)
	expectStrings(t, log, "sa")
	log = nil
	s.ClearStatusListeners()
	s.NotifyStatusListeners(AnimationStatusForward)
	expectStrings(t, log)
}

// Own: errors are reported with the animation library's details.
func TestListenerErrorsAreReported(t *testing.T) {
	setUp(t)
	var details []*foundation.FlutterErrorDetails
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { details = append(details, d) }
	controller := newController()
	controller.AddListener(func() { panic("value") })
	controller.AddStatusListener(func(AnimationStatus) { panic("status") })
	controller.SetValue(0.5)
	if len(details) != 2 {
		t.Fatalf("%d reports", len(details))
	}
	expect(t, details[0].Exception, any("value"))
	expect(t, *details[0].Library, "animation library")
	expect(t, details[0].Context.ToString(), "while notifying listeners for AnimationController")
	expect(t, details[1].Exception, any("status"))
	expect(t, *details[1].Library, "animation library")
	expect(t, details[1].Context.ToString(), "while notifying status listeners for AnimationController")
	var info []string
	for n := range details[0].InformationCollector() {
		info = append(info, n.ToString())
	}
	equalsIgnoringHashCodes(t, info[0], "The AnimationController notifying listeners was: "+
		"AnimationController#00000(▶ 0.500; paused)")
	info = nil
	for n := range details[1].InformationCollector() {
		info = append(info, n.ToString())
	}
	equalsIgnoringHashCodes(t, info[0], "The AnimationController notifying status listeners was: "+
		"AnimationController#00000(▶ 0.500; paused)")
}

// --- animation_controller_listener_test.dart

type testAnimationController struct{ AnimationController }

func (c *testAnimationController) publicNotifyListeners() { c.NotifyListeners() }
func (c *testAnimationController) publicNotifyStatusListeners(s AnimationStatus) {
	c.NotifyStatusListeners(s)
}

func TestDisposingControllerRemovesListeners(t *testing.T) {
	setUp(t)
	controller := &testAnimationController{}
	controller.Construct(controller, vsync, AnimationControllerOptions{Duration: dur(100 * ms)})
	statusListener, listener := 0, 0
	controller.AddListener(func() { listener++ })
	controller.AddStatusListener(func(AnimationStatus) { statusListener++ })
	expect(t, statusListener, 0)
	expect(t, listener, 0)
	controller.publicNotifyListeners()
	controller.publicNotifyStatusListeners(AnimationStatusCompleted)
	expect(t, statusListener, 1)
	expect(t, listener, 1)
	controller.Dispose()
	controller.publicNotifyListeners()
	controller.publicNotifyStatusListeners(AnimationStatusCompleted)
	expect(t, statusListener, 1)
	expect(t, listener, 1)
}

// --- animation_from_listener_test.dart

func TestAnimationCreatedFromValueListenable(t *testing.T) {
	listenable := foundation.NewValueNotifier(0.0)
	animation := NewAnimationFromValueListenable[float64](listenable)
	expect(t, animation.Status(), AnimationStatusForward)
	expect(t, animation.Value(), 0.0)
	listenable.SetValue(1.0)
	expect(t, animation.Value(), 1.0)
	listenerCalled := false
	listener := func() { listenerCalled = true }
	animation.AddListener(listener)
	listenable.SetValue(0.5)
	expect(t, listenerCalled, true)
	listenerCalled = false
	animation.RemoveListener(listener)
	listenable.SetValue(0.2)
	expect(t, listenerCalled, false)
}

func TestAnimationCreatedFromValueListenableCanTransformValue(t *testing.T) {
	listenable := foundation.NewValueNotifier(0.0)
	animation := NewAnimationFromValueListenable[float64](listenable,
		AnimationFromValueListenableOptions[float64]{Transformer: func(input float64) float64 { return input / 10 }})
	expect(t, animation.Status(), AnimationStatusForward)
	expect(t, animation.Value(), 0.0)
	listenable.SetValue(10.0)
	expect(t, animation.Value(), 1.0)
}

func TestAnimationCreatedFromValueListenableCanBeTransformedViaDrive(t *testing.T) {
	listenable := foundation.NewValueNotifier(0.0)
	animation := NewAnimationFromValueListenable[float64](listenable)
	offset := Drive(animation, NewAnimatableFromCallback(func(value float64) ui.Offset {
		return ui.NewOffset(0.0, value)
	}))
	expect(t, offset.Value(), ui.OffsetZero)
	expect(t, offset.Status(), AnimationStatusForward)
	listenable.SetValue(10)
	expect(t, offset.Value(), ui.NewOffset(0.0, 10.0))
	listenerCalled := false
	listener := func() { listenerCalled = true }
	offset.AddListener(listener)
	listenable.SetValue(0.5)
	expect(t, listenerCalled, true)
	listenerCalled = false
	offset.RemoveListener(listener)
	listenable.SetValue(0.2)
	expect(t, listenerCalled, false)
}

// Own: a transformer returning null falls back to the listenable's value.
func TestFromValueListenableNullTransformerResult(t *testing.T) {
	v := 3.0
	listenable := foundation.NewValueNotifier(&v)
	animation := NewAnimationFromValueListenable[*float64](listenable,
		AnimationFromValueListenableOptions[*float64]{Transformer: func(*float64) *float64 { return nil }})
	expect(t, animation.Value(), &v)
}

// --- iteration_patterns_test.dart

func TestAnimationControllerWithMutatingListener(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	var log []string
	listener1 := func() { log = append(log, "listener1") }
	listener3 := func() { log = append(log, "listener3") }
	listener4 := func() { log = append(log, "listener4") }
	listener2 := func() {
		log = append(log, "listener2")
		controller.RemoveListener(listener1)
		controller.RemoveListener(listener3)
		controller.AddListener(listener4)
	}
	controller.AddListener(listener1)
	controller.AddListener(listener2)
	controller.AddListener(listener3)
	controller.SetValue(0.2)
	expectStrings(t, log, "listener1", "listener2")
	log = nil
	controller.SetValue(0.3)
	expectStrings(t, log, "listener2", "listener4")
	log = nil
	controller.SetValue(0.4)
	expectStrings(t, log, "listener2", "listener4")
}

func TestAnimationControllerWithMutatingStatusListener(t *testing.T) {
	setUp(t)
	controller := newController(ms100())
	var log []string
	listener1 := func(AnimationStatus) { log = append(log, "listener1") }
	listener3 := func(AnimationStatus) { log = append(log, "listener3") }
	listener4 := func(AnimationStatus) { log = append(log, "listener4") }
	listener2 := func(AnimationStatus) {
		log = append(log, "listener2")
		controller.RemoveStatusListener(listener1)
		controller.RemoveStatusListener(listener3)
		controller.AddStatusListener(listener4)
	}
	controller.AddStatusListener(listener1)
	controller.AddStatusListener(listener2)
	controller.AddStatusListener(listener3)
	controller.Forward()
	expectStrings(t, log, "listener1", "listener2")
	log = nil
	controller.Reverse()
	expectStrings(t, log, "listener2", "listener4")
	log = nil
	controller.Forward()
	expectStrings(t, log, "listener2", "listener4", "listener4")
	log = nil
	controller.Dispose()
}

func isArgumentError(t *testing.T, e any) {
	t.Helper()
	if _, ok := e.(dart.ArgumentError); !ok {
		t.Fatalf("want an ArgumentError, got %T %v", e, e)
	}
}

func TestAnimationControllerWithThrowingListener(t *testing.T) {
	st := setUp(t)
	controller := newController(ms100())
	t.Cleanup(controller.Dispose)
	var log []string
	listener1 := func() { log = append(log, "listener1") }
	badListener := func() {
		log = append(log, "badListener")
		panic(dart.ArgumentError{})
	}
	listener2 := func() { log = append(log, "listener2") }
	controller.AddListener(listener1)
	controller.AddListener(badListener)
	controller.AddListener(listener2)
	controller.SetValue(0.2)
	expectStrings(t, log, "listener1", "badListener", "listener2")
	isArgumentError(t, st.takeException())
}

func TestAnimationControllerWithThrowingStatusListener(t *testing.T) {
	st := setUp(t)
	controller := newController(ms100())
	var log []string
	listener1 := func(AnimationStatus) { log = append(log, "listener1") }
	badListener := func(AnimationStatus) {
		log = append(log, "badListener")
		panic(dart.ArgumentError{})
	}
	listener2 := func(AnimationStatus) { log = append(log, "listener2") }
	controller.AddStatusListener(listener1)
	controller.AddStatusListener(badListener)
	controller.AddStatusListener(listener2)
	controller.Forward()
	expectStrings(t, log, "listener1", "badListener", "listener2")
	isArgumentError(t, st.takeException())
	controller.Dispose()
}
