// Port of packages/flutter/test/gestures/pointer_signal_resolver_test.dart
// (Flutter 3.50.1): all 5 tests. Dart's `const PointerScrollEvent()` is
// canonicalized (every const event is the same object); the tester's events
// are fresh objects here, which the tests don't distinguish.
package gestures_test

import (
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

type testPointerSignalListener struct {
	t           *testing.T
	event       PointerSignalEvent
	callbackRan bool
}

func (l *testPointerSignalListener) callback(event PointerSignalEvent) {
	if event != l.event {
		l.t.Error("callback got another event")
	}
	if l.callbackRan {
		l.t.Error("callback ran twice")
	}
	l.callbackRan = true
}

type pointerSignalTester struct {
	t        *testing.T
	resolver *PointerSignalResolver
	event    PointerSignalEvent
}

func newPointerSignalTester(t *testing.T) *pointerSignalTester {
	return &pointerSignalTester{t: t, resolver: NewPointerSignalResolver(), event: NewPointerScrollEvent()}
}

func (p *pointerSignalTester) addListener() *testPointerSignalListener {
	l := &testPointerSignalListener{t: p.t, event: p.event}
	p.resolver.Register(p.event, l.callback)
	return l
}

func (p *pointerSignalTester) resolve() {
	p.resolver.Resolve(p.event)
	p.event = NewPointerScrollEvent()
}

func TestPointerSignalResolverNoEntriesIsNoOp(t *testing.T) {
	p := newPointerSignalTester(t)
	p.resolver.Resolve(p.event)
}

func TestPointerSignalResolverNoEntriesNotifiesEngine(t *testing.T) {
	allowedPlatformDefault := false
	p := newPointerSignalTester(t)
	p.event = NewPointerScrollEvent(PointerScrollEventOptions{OnRespond: func(allow bool) { allowedPlatformDefault = allow }})
	p.resolver.Resolve(p.event)
	if !allowedPlatformDefault {
		t.Fatal("Should have called respond with allowPlatformDefault: true")
	}
}

func TestPointerSignalResolverFirstEntryWins(t *testing.T) {
	p := newPointerSignalTester(t)
	first := p.addListener()
	second := p.addListener()
	p.resolve()
	if !first.callbackRan || second.callbackRan {
		t.Fatal("first must win")
	}
}

func TestPointerSignalResolverReuseAfterResolve(t *testing.T) {
	p := newPointerSignalTester(t)
	first := p.addListener()
	second := p.addListener()
	p.resolve()
	if !first.callbackRan || second.callbackRan {
		t.Fatal("first must win")
	}
	newEventListener := p.addListener()
	p.resolve()
	if !newEventListener.callbackRan || !first.callbackRan || second.callbackRan {
		t.Fatal("re-use failed")
	}
}

func TestPointerSignalResolverWorksWithTransformedEvents(t *testing.T) {
	resolver := NewPointerSignalResolver()
	originalEvent := NewPointerScrollEvent()
	transformedEvent := originalEvent.Transformed(vm.NewMatrix4TranslationValues(10.0, 20.0, 0.0)).(PointerSignalEvent)
	anotherTransformedEvent := originalEvent.Transformed(vm.NewMatrix4TranslationValues(30.0, 50.0, 0.0)).(PointerSignalEvent)
	if PointerSignalEvent(originalEvent) == transformedEvent || transformedEvent.Original() != PointerEvent(originalEvent) {
		t.Fatal("transformed event identity")
	}
	if PointerSignalEvent(originalEvent) == anotherTransformedEvent || anotherTransformedEvent.Original() != PointerEvent(originalEvent) {
		t.Fatal("another transformed event identity")
	}
	var events []PointerSignalEvent
	resolver.Register(transformedEvent, func(e PointerSignalEvent) { events = append(events, e) })
	// Registering a second transformed event should not panic.
	resolver.Register(anotherTransformedEvent, func(e PointerSignalEvent) { events = append(events, e) })
	resolver.Resolve(originalEvent)
	if len(events) != 1 || events[0] != transformedEvent {
		t.Fatalf("events %v", events)
	}
}
