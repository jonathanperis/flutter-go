package animation

// Port of packages/flutter/lib/src/animation/listener_helpers.dart.
//
// Dart's four mixins are embeddable structs with the Dart names. Each calls
// the members it leaves abstract (didRegisterListener,
// didStartListening...) through self, which the embedding type sets with
// SetSelf (the animations here do it in their constructors, and define a
// SetSelf of their own that sets every embedded mixin's).
//
// Listeners are Go funcs compared as Dart compares functions
// (dart.FunctionsEqual): AnimationLocalListenersMixin keeps Dart's
// HashedObserverList semantics (a listener added twice is called once per
// notification and must be removed twice), AnimationLocalStatusListenersMixin
// keeps ObserverList's (a listener added twice is called twice).

import (
	"iter"
	"slices"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func unbound(name string) string {
	return "animation: " + name + " used before SetSelf: build animations with their constructors, " +
		"or call SetSelf(outer) once on a type that embeds it"
}

// LazyListenerSelf is what AnimationLazyListenerMixin calls on the object
// it is mixed into.
type LazyListenerSelf interface {
	// DidStartListening is called when the number of listeners changes from
	// zero to one.
	DidStartListening()
	// DidStopListening is called when the number of listeners changes from
	// one to zero.
	DidStopListening()
}

// AnimationLazyListenerMixin is a mixin that helps listen to another object
// only when this object has registered listeners.
type AnimationLazyListenerMixin struct {
	listenerCounter int
	self            LazyListenerSelf
}

// SetSelf points the mixin at the object it is mixed into.
func (m *AnimationLazyListenerMixin) SetSelf(self LazyListenerSelf) { m.self = self }

func (m *AnimationLazyListenerMixin) lazyOuter() LazyListenerSelf {
	if m.self == nil {
		panic(unbound("AnimationLazyListenerMixin"))
	}
	return m.self
}

// DidRegisterListener calls DidStartListening when the first listener is
// added (Dart's @protected didRegisterListener).
func (m *AnimationLazyListenerMixin) DidRegisterListener() {
	assert(m.listenerCounter >= 0, nil)
	if m.listenerCounter == 0 {
		m.lazyOuter().DidStartListening()
	}
	m.listenerCounter++
}

// DidUnregisterListener calls DidStopListening when the last listener is
// removed (Dart's @protected didUnregisterListener).
func (m *AnimationLazyListenerMixin) DidUnregisterListener() {
	assert(m.listenerCounter >= 1, nil)
	m.listenerCounter--
	if m.listenerCounter == 0 {
		m.lazyOuter().DidStopListening()
	}
}

// IsListening reports whether there are any listeners.
func (m *AnimationLazyListenerMixin) IsListening() bool { return m.listenerCounter > 0 }

// AnimationEagerListenerMixin is a mixin that replaces the
// DidRegisterListener/DidUnregisterListener contract with a Dispose
// contract.
type AnimationEagerListenerMixin struct{}

// DidRegisterListener does nothing.
func (*AnimationEagerListenerMixin) DidRegisterListener() {}

// DidUnregisterListener does nothing.
func (*AnimationEagerListenerMixin) DidUnregisterListener() {}

// Dispose releases the resources used by this object (here, nothing;
// types mixing it in override it and call it).
func (*AnimationEagerListenerMixin) Dispose() {}

// ListenerRegistrationSelf is what the local-listener mixins call on the
// object they are mixed into (provided by AnimationLazyListenerMixin or
// AnimationEagerListenerMixin).
type ListenerRegistrationSelf interface {
	DidRegisterListener()
	DidUnregisterListener()
}

// registrationSelf is self when it implements ListenerRegistrationSelf
// (an outer type embedding one of the animations), else fallback.
func registrationSelf(self any, fallback ListenerRegistrationSelf) ListenerRegistrationSelf {
	if s, ok := self.(ListenerRegistrationSelf); ok {
		return s
	}
	return fallback
}

// hashedListeners is Dart's HashedObserverList<VoidCallback> (a
// LinkedHashMap from listener to count) with Dart's function equality.
type hashedListeners struct {
	entries []hashedEntry
}

type hashedEntry struct {
	listener func()
	count    int
}

func (l *hashedListeners) index(listener func()) int {
	return slices.IndexFunc(l.entries, func(e hashedEntry) bool { return dart.FunctionsEqual(e.listener, listener) })
}

func (l *hashedListeners) add(listener func()) {
	if i := l.index(listener); i >= 0 {
		l.entries[i].count++
		return
	}
	l.entries = append(l.entries, hashedEntry{listener, 1})
}

func (l *hashedListeners) remove(listener func()) bool {
	i := l.index(listener)
	if i < 0 {
		return false
	}
	if l.entries[i].count == 1 {
		l.entries = slices.Delete(l.entries, i, i+1)
	} else {
		l.entries[i].count--
	}
	return true
}

func (l *hashedListeners) contains(listener func()) bool { return l.index(listener) >= 0 }

func (l *hashedListeners) toList() []func() {
	out := make([]func(), len(l.entries))
	for i, e := range l.entries {
		out[i] = e.listener
	}
	return out
}

// AnimationLocalListenersMixin is a mixin that implements the AddListener /
// RemoveListener protocol and notifies all the registered listeners when
// NotifyListeners is called.
type AnimationLocalListenersMixin struct {
	listeners hashedListeners
	self      ListenerRegistrationSelf
}

// SetSelf points the mixin at the object it is mixed into.
func (m *AnimationLocalListenersMixin) SetSelf(self ListenerRegistrationSelf) { m.self = self }

func (m *AnimationLocalListenersMixin) listenersOuter() ListenerRegistrationSelf {
	if m.self == nil {
		panic(unbound("AnimationLocalListenersMixin"))
	}
	return m.self
}

// AddListener calls listener every time the value of the animation
// changes.
func (m *AnimationLocalListenersMixin) AddListener(listener func()) {
	if listener == nil {
		panic(dart.NewAssertionError("AddListener(nil): Dart's VoidCallback is not nullable"))
	}
	m.listenersOuter().DidRegisterListener()
	m.listeners.add(listener)
}

// RemoveListener stops calling listener every time the value of the
// animation changes. A listener that is not registered is ignored.
func (m *AnimationLocalListenersMixin) RemoveListener(listener func()) {
	if m.listeners.remove(listener) {
		m.listenersOuter().DidUnregisterListener()
	}
}

// ClearListeners removes all listeners added with AddListener (Dart's
// @protected clearListeners).
func (m *AnimationLocalListenersMixin) ClearListeners() { m.listeners = hashedListeners{} }

// NotifyListeners calls all the listeners (Dart's @protected
// notifyListeners). A listener removed during the notification is not
// called; one added during it is called from the next notification on.
// Errors are reported to FlutterError.reportError.
func (m *AnimationLocalListenersMixin) NotifyListeners() {
	localListeners := m.listeners.toList()
	for _, listener := range localListeners {
		var collector foundation.InformationCollector
		if foundation.KDebugMode {
			self := m.listenersOuter()
			collector = func() iter.Seq[foundation.DiagnosticsNode] {
				style := foundation.DiagnosticsTreeStyleErrorProperty
				return slices.Values([]foundation.DiagnosticsNode{
					foundation.NewDiagnosticsProperty[any]("The "+runtimeType(self)+" notifying listeners was", self,
						foundation.DiagnosticsPropertyOptions{Style: &style}),
				})
			}
		}
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					stack := dart.StackTraceFromPanic()
					library := "animation library"
					foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
						Stack:                stack,
						Library:              &library,
						Context:              foundation.NewErrorDescription("while notifying listeners for " + runtimeType(m.listenersOuter())),
						InformationCollector: collector,
					}))
				}
			}()
			if m.listeners.contains(listener) {
				listener()
			}
		}()
	}
}

// AnimationLocalStatusListenersMixin is a mixin that implements the
// AddStatusListener / RemoveStatusListener protocol and notifies all the
// registered listeners when NotifyStatusListeners is called.
type AnimationLocalStatusListenersMixin struct {
	statusListeners []AnimationStatusListener // Dart's ObserverList
	self            ListenerRegistrationSelf
}

// SetSelf points the mixin at the object it is mixed into.
func (m *AnimationLocalStatusListenersMixin) SetSelf(self ListenerRegistrationSelf) { m.self = self }

func (m *AnimationLocalStatusListenersMixin) statusOuter() ListenerRegistrationSelf {
	if m.self == nil {
		panic(unbound("AnimationLocalStatusListenersMixin"))
	}
	return m.self
}

func (m *AnimationLocalStatusListenersMixin) statusIndex(listener AnimationStatusListener) int {
	return slices.IndexFunc(m.statusListeners, func(l AnimationStatusListener) bool {
		return dart.FunctionsEqual(l, listener)
	})
}

// AddStatusListener calls listener every time the status of the animation
// changes.
func (m *AnimationLocalStatusListenersMixin) AddStatusListener(listener AnimationStatusListener) {
	if listener == nil {
		panic(dart.NewAssertionError("AddStatusListener(nil): Dart's AnimationStatusListener is not nullable"))
	}
	m.statusOuter().DidRegisterListener()
	m.statusListeners = append(m.statusListeners, listener)
}

// RemoveStatusListener stops calling listener every time the status of the
// animation changes. A listener that is not registered is ignored.
func (m *AnimationLocalStatusListenersMixin) RemoveStatusListener(listener AnimationStatusListener) {
	i := m.statusIndex(listener)
	if i < 0 {
		return
	}
	m.statusListeners = slices.Delete(m.statusListeners, i, i+1)
	m.statusOuter().DidUnregisterListener()
}

// ClearStatusListeners removes all listeners added with AddStatusListener
// (Dart's @protected clearStatusListeners).
func (m *AnimationLocalStatusListenersMixin) ClearStatusListeners() { m.statusListeners = nil }

// NotifyStatusListeners calls all the status listeners (Dart's @protected
// notifyStatusListeners). Errors are reported to FlutterError.reportError.
func (m *AnimationLocalStatusListenersMixin) NotifyStatusListeners(status AnimationStatus) {
	localListeners := slices.Clone(m.statusListeners)
	for _, listener := range localListeners {
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					stack := dart.StackTraceFromPanic()
					var collector foundation.InformationCollector
					if foundation.KDebugMode {
						self := m.statusOuter()
						collector = func() iter.Seq[foundation.DiagnosticsNode] {
							style := foundation.DiagnosticsTreeStyleErrorProperty
							return slices.Values([]foundation.DiagnosticsNode{
								foundation.NewDiagnosticsProperty[any]("The "+runtimeType(self)+" notifying status listeners was", self,
									foundation.DiagnosticsPropertyOptions{Style: &style}),
							})
						}
					}
					library := "animation library"
					foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
						Stack:                stack,
						Library:              &library,
						Context:              foundation.NewErrorDescription("while notifying status listeners for " + runtimeType(m.statusOuter())),
						InformationCollector: collector,
					}))
				}
			}()
			if m.statusIndex(listener) >= 0 {
				listener(status)
			}
		}()
	}
}
