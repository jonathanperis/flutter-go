// Port of packages/flutter/test/foundation/memory_allocations_test.dart.
//
// Ported: 6 of 7 tests. Skipped: "publishers in Flutter dispatch events in
// debug mode" (it creates ValueNotifier/ChangeNotifier, from
// package:listen, and real dart:ui Pictures and Images, neither ported).
// The PrintOverrideTestBinding / tester.takeException plumbing becomes a
// replaced FlutterErrorOnError that counts the reported errors.
//
// Found: in "does not invoke concurrently removed listeners", Flutter's
// `expect(ma.hasListeners, isFalse)` inside listener1 is false in fact
// (listener1 is still registered): the failed expect throws inside the
// dispatch, which reports it through FlutterError.reportError instead of
// failing the test. The Go test checks the real value (true).
package foundation

import (
	"slices"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func checkSdkHandlersSet(t *testing.T) {
	t.Helper()
	if ui.ImageOnCreate == nil || ui.PictureOnCreate == nil || ui.ImageOnDispose == nil || ui.PictureOnDispose == nil {
		t.Fatal("SDK handlers not set")
	}
}

func checkSdkHandlersNotSet(t *testing.T) {
	t.Helper()
	if ui.ImageOnCreate != nil || ui.PictureOnCreate != nil || ui.ImageOnDispose != nil || ui.PictureOnDispose != nil {
		t.Fatal("SDK handlers set")
	}
}

func memoryAllocationsSetUp(t *testing.T) *FlutterMemoryAllocations {
	t.Helper()
	ma := FlutterMemoryAllocationsInstance
	if ma.HasListeners() {
		t.Fatal("listeners left by another test")
	}
	checkSdkHandlersNotSet(t)
	return ma
}

func TestMemoryAllocationsAddListenerAndRemoveListener(t *testing.T) {
	ma := memoryAllocationsSetUp(t)
	var event ObjectEvent = NewObjectDisposed("object")
	var receivedEvent ObjectEvent
	listener := func(e ObjectEvent) { receivedEvent = e }
	equal(t, ma.HasListeners(), false)

	ma.AddListener(listener)
	checkSdkHandlersSet(t)
	ma.DispatchObjectEvent(event)
	if receivedEvent != event {
		t.Fatal("event not received")
	}
	equal(t, ma.HasListeners(), true)
	receivedEvent = nil

	ma.RemoveListener(listener)
	ma.DispatchObjectEvent(event)
	if receivedEvent != nil {
		t.Fatal("received after removal")
	}
	equal(t, ma.HasListeners(), false)
	checkSdkHandlersNotSet(t)
}

func TestMemoryAllocationsDispatchHandlesBadListeners(t *testing.T) {
	ma := memoryAllocationsSetUp(t)
	event := NewObjectDisposed("object")
	var log []string
	var reported []*FlutterErrorDetails
	originalOnError := FlutterErrorOnError
	FlutterErrorOnError = func(details *FlutterErrorDetails) { reported = append(reported, details) }
	defer func() { FlutterErrorOnError = originalOnError }()

	badListener1 := func(ObjectEvent) { log = append(log, "badListener1"); panic(argumentError{}) }
	listener1 := func(ObjectEvent) { log = append(log, "listener1") }
	badListener2 := func(ObjectEvent) { log = append(log, "badListener2"); panic(argumentError{}) }
	listener2 := func(ObjectEvent) { log = append(log, "listener2") }

	ma.AddListener(badListener1)
	checkSdkHandlersSet(t)
	ma.AddListener(listener1)
	ma.AddListener(badListener2)
	ma.AddListener(listener2)

	ma.DispatchObjectEvent(event)
	if !slices.Equal(log, []string{"badListener1", "listener1", "badListener2", "listener2"}) {
		t.Fatal(log)
	}
	// Flutter: tester.takeException() contains 'Multiple exceptions (2)'.
	equal(t, len(reported), 2)
	equal(t, *reported[0].Library, "foundation library")
	equal(t, reported[0].Context.String(), "MemoryAllocations while dispatching notifications for String")

	ma.RemoveListener(badListener1)
	checkSdkHandlersSet(t)
	ma.RemoveListener(listener1)
	ma.RemoveListener(badListener2)
	ma.RemoveListener(listener2)
	checkSdkHandlersNotSet(t)

	log = nil
	equal(t, ma.HasListeners(), false)
	ma.DispatchObjectEvent(event)
	equal(t, len(log), 0)
}

func TestMemoryAllocationsDoesNotInvokeConcurrentlyAddedListeners(t *testing.T) {
	ma := memoryAllocationsSetUp(t)
	event := NewObjectDisposed("object")
	var log []string
	listener2 := func(ObjectEvent) { log = append(log, "listener2") }
	listener1 := func(ObjectEvent) {
		log = append(log, "listener1")
		ma.AddListener(listener2)
	}

	ma.AddListener(listener1)
	checkSdkHandlersSet(t)

	ma.DispatchObjectEvent(event)
	if !slices.Equal(log, []string{"listener1"}) {
		t.Fatal(log)
	}
	log = nil

	ma.DispatchObjectEvent(event)
	if !slices.Equal(log, []string{"listener1", "listener2"}) {
		t.Fatal(log)
	}
	log = nil

	ma.RemoveListener(listener1)
	ma.RemoveListener(listener2)
	checkSdkHandlersNotSet(t)

	equal(t, ma.HasListeners(), false)
	ma.DispatchObjectEvent(event)
	equal(t, len(log), 0)
}

func TestMemoryAllocationsDoesNotInvokeConcurrentlyRemovedListeners(t *testing.T) {
	ma := memoryAllocationsSetUp(t)
	event := NewObjectDisposed("object")
	var log []string
	var hasListenersInside []bool
	listener2 := func(ObjectEvent) { log = append(log, "listener2") }
	listener1 := func(ObjectEvent) {
		log = append(log, "listener1")
		ma.RemoveListener(listener2)
		hasListenersInside = append(hasListenersInside, ma.HasListeners())
	}

	ma.AddListener(listener1)
	ma.AddListener(listener2)

	ma.DispatchObjectEvent(event)
	if !slices.Equal(log, []string{"listener1"}) {
		t.Fatal(log)
	}
	// See the note at the top: Flutter expects false here, silently.
	if !slices.Equal(hasListenersInside, []bool{true}) {
		t.Fatal(hasListenersInside)
	}
	log = nil

	ma.RemoveListener(listener1)
	checkSdkHandlersNotSet(t)
	equal(t, ma.HasListeners(), false)
}

func TestMemoryAllocationsLastRemoveListenerUnsubscribes(t *testing.T) {
	ma := memoryAllocationsSetUp(t)
	listener1 := func(ObjectEvent) {}
	listener2 := func(ObjectEvent) {}

	ma.AddListener(listener1)
	checkSdkHandlersSet(t)
	ma.AddListener(listener2)
	checkSdkHandlersSet(t)
	ma.RemoveListener(listener1)
	checkSdkHandlersSet(t)
	ma.RemoveListener(listener2)
	checkSdkHandlersNotSet(t)
}

func TestKFlutterMemoryAllocationsEnabledInDebugMode(t *testing.T) {
	equal(t, KFlutterMemoryAllocationsEnabled, true)
}

// Not in Flutter's tests.

func TestMemoryAllocationsSdkHooksAndEvents(t *testing.T) {
	ma := memoryAllocationsSetUp(t)
	var events []ObjectEvent
	listener := func(e ObjectEvent) { events = append(events, e) }
	ma.AddListener(listener)
	image, picture := &ui.Image{}, &ui.Picture{}
	ui.ImageOnCreate(image)
	ui.PictureOnCreate(picture)
	ui.ImageOnDispose(image)
	ui.PictureOnDispose(picture)
	ma.RemoveListener(listener)
	equal(t, len(events), 4)
	created := events[0].(*ObjectCreated)
	equal(t, created.Library, "dart:ui")
	equal(t, created.ClassName, "Image")
	equal(t, events[1].(*ObjectCreated).ClassName, "Picture")
	if events[2].Object() != any(image) || events[3].Object() != any(picture) {
		t.Fatal("disposed objects")
	}
	m := created.ToMap()[image]
	equal(t, m["libraryName"], any("dart:ui"))
	equal(t, m["className"], any("Image"))
	equal(t, m["eventType"], any("created"))
	equal(t, events[2].ToMap()[image]["eventType"], any("disposed"))
	equal(t, len(events[2].ToMap()[image]), 1)

	// The same listener added twice is removed by one RemoveListener; a
	// different closure is not removed.
	other := func(ObjectEvent) {}
	ma.AddListener(listener)
	ma.AddListener(listener)
	ma.AddListener(other)
	ma.RemoveListener(listener)
	equal(t, ma.HasListeners(), true)
	ma.RemoveListener(other)
	equal(t, ma.HasListeners(), false)

	// Removing every listener during a dispatch: hasListeners throws
	// (firstWhere finds no element) until the dispatch ends.
	var inside any
	var self func(ObjectEvent)
	self = func(ObjectEvent) {
		ma.RemoveListener(self)
		func() {
			defer func() { inside = recover() }()
			ma.HasListeners()
		}()
	}
	ma.AddListener(self)
	ma.DispatchObjectEvent(NewObjectDisposed("x"))
	if !isStateError(inside) {
		t.Fatalf("inside = %#v", inside)
	}
	equal(t, ma.HasListeners(), false)
	checkSdkHandlersNotSet(t)

	// No listener: dispatchObjectCreated does nothing.
	ma.DispatchObjectCreated("lib", "C", "o")
}
