package foundation

// Port of memory_allocations.dart (Flutter 3.50.1).
//
// Not ported: the constructor's hooks into package:listen's
// Listenable.debugMaybeDispatchCreated/Disposed (ChangeNotifier and
// Listenable come from package:listen, not ported yet).

import (
	"iter"
	"slices"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

// kMemoryAllocations is Dart's bool.fromEnvironment('flutter.memory_allocations')
// (false: flutter-go has no such build flag yet).
const kMemoryAllocations = false

// KFlutterMemoryAllocationsEnabled reports whether memory allocations are
// tracked: in debug builds, or with the flutter.memory_allocations flag.
const KFlutterMemoryAllocationsEnabled = kMemoryAllocations || KDebugMode

const dartUILibrary = "dart:ui"

const (
	fieldEventType   = "eventType"
	fieldLibraryName = "libraryName"
	fieldClassName   = "className"
)

// ObjectEvent is a lifecycle event of an instrumented object.
type ObjectEvent interface {
	// Object is the instrumented object.
	Object() any
	// ToMap converts the event to a map that can be passed to the VM
	// service: {object: {fields}}.
	ToMap() map[any]map[string]any
}

// ObjectEventListener receives ObjectEvents.
type ObjectEventListener func(event ObjectEvent)

// ObjectCreated is the event of an instrumented object's creation.
type ObjectCreated struct {
	// Library is the library of the object's class.
	Library string
	// ClassName is the name of the object's class.
	ClassName string
	object    any
}

// NewObjectCreated is ObjectCreated(library:, className:, object:).
func NewObjectCreated(library, className string, object any) *ObjectCreated {
	return &ObjectCreated{Library: library, ClassName: className, object: object}
}

func (e *ObjectCreated) Object() any { return e.object }

func (e *ObjectCreated) ToMap() map[any]map[string]any {
	return map[any]map[string]any{
		e.object: {
			fieldLibraryName: e.Library,
			fieldClassName:   e.ClassName,
			fieldEventType:   "created",
		},
	}
}

// ObjectDisposed is the event of an instrumented object's disposal.
type ObjectDisposed struct{ object any }

// NewObjectDisposed is ObjectDisposed(object:).
func NewObjectDisposed(object any) *ObjectDisposed { return &ObjectDisposed{object: object} }

func (e *ObjectDisposed) Object() any { return e.object }

func (e *ObjectDisposed) ToMap() map[any]map[string]any {
	return map[any]map[string]any{e.object: {fieldEventType: "disposed"}}
}

// MemoryAllocations is FlutterMemoryAllocations (Deprecated: use
// FlutterMemoryAllocations; deprecated after v3.18.0-18.0.pre).
type MemoryAllocations = FlutterMemoryAllocations

// FlutterMemoryAllocations receives events about the creation and disposal
// of Flutter's (and dart:ui's) disposable objects, and dispatches them to
// its listeners (memory-leak tracking tools). Everything is a no-op unless
// KFlutterMemoryAllocationsEnabled.
//
// Listeners are compared by function identity, as Dart's == on functions:
// remove the same function value that was added.
type FlutterMemoryAllocations struct {
	listeners             []ObjectEventListener
	hasListenerList       bool
	activeDispatchLoops   int
	listenersContainNulls bool
}

// FlutterMemoryAllocationsInstance is FlutterMemoryAllocations.instance,
// the singleton.
var FlutterMemoryAllocationsInstance = &FlutterMemoryAllocations{}

// AddListener registers listener; the first one subscribes to dart:ui's
// Image and Picture creation/disposal hooks.
func (m *FlutterMemoryAllocations) AddListener(listener ObjectEventListener) {
	if !KFlutterMemoryAllocationsEnabled {
		return
	}
	if !m.hasListenerList {
		m.listeners = []ObjectEventListener{}
		m.hasListenerList = true
		m.subscribeToSdkObjects()
	}
	m.listeners = append(m.listeners, listener)
}

// RemoveListener unregisters every registration of listener. During a
// dispatch the entries are nulled out and removed afterwards; once no
// listener is left, the dart:ui hooks are unset.
func (m *FlutterMemoryAllocations) RemoveListener(listener ObjectEventListener) {
	if !KFlutterMemoryAllocationsEnabled {
		return
	}
	if !m.hasListenerList {
		return
	}
	id := funcIdentity(listener)
	if m.activeDispatchLoops > 0 {
		for i, l := range m.listeners {
			if l != nil && funcIdentity(l) == id {
				m.listeners[i] = nil
				m.listenersContainNulls = true
			}
		}
	} else {
		kept := m.listeners[:0]
		for _, l := range m.listeners {
			if funcIdentity(l) != id {
				kept = append(kept, l)
			}
		}
		clear(m.listeners[len(kept):])
		m.listeners = kept
		m.checkListenersForEmptiness()
	}
}

func (m *FlutterMemoryAllocations) tryDefragmentListeners() {
	if m.activeDispatchLoops > 0 || !m.listenersContainNulls {
		return
	}
	if m.hasListenerList {
		kept := m.listeners[:0]
		for _, l := range m.listeners {
			if l != nil {
				kept = append(kept, l)
			}
		}
		clear(m.listeners[len(kept):])
		m.listeners = kept
	}
	m.listenersContainNulls = false
	m.checkListenersForEmptiness()
}

func (m *FlutterMemoryAllocations) checkListenersForEmptiness() {
	if m.hasListenerList && len(m.listeners) == 0 {
		m.listeners = nil
		m.hasListenerList = false
		m.unSubscribeFromSdkObjects()
	}
}

// HasListeners reports whether a listener is registered.
func (m *FlutterMemoryAllocations) HasListeners() bool {
	if !KFlutterMemoryAllocationsEnabled {
		return false
	}
	if m.listenersContainNulls {
		// Dart: _listeners?.firstWhere((l) => l != null) != null, which
		// throws a StateError when every entry is null.
		for _, l := range m.listeners {
			if l != nil {
				return true
			}
		}
		panic(dart.StateError{Message: "No element"})
	}
	return m.hasListenerList && len(m.listeners) > 0
}

// DispatchObjectEvent sends event to the listeners registered when the
// dispatch starts (listeners added meanwhile are not called, removed ones
// are skipped). A listener's panic is reported with
// FlutterErrorReportError and the others still run.
func (m *FlutterMemoryAllocations) DispatchObjectEvent(event ObjectEvent) {
	if !KFlutterMemoryAllocationsEnabled {
		return
	}
	if !m.hasListenerList || len(m.listeners) == 0 {
		return
	}
	m.activeDispatchLoops++
	end := len(m.listeners)
	for i := 0; i < end; i++ {
		m.callListener(i, event)
	}
	m.activeDispatchLoops--
	m.tryDefragmentListeners()
}

func (m *FlutterMemoryAllocations) callListener(i int, event ObjectEvent) {
	defer func() {
		if r := recover(); r != nil {
			stack := dart.StackTraceFromPanic()
			typ := dartRuntimeType(event.Object())
			style := DiagnosticsTreeStyleErrorProperty
			FlutterErrorReportError(NewFlutterErrorDetails(r, FlutterErrorDetailsOptions{
				Stack:   stack,
				Library: strp("foundation library"),
				Context: NewErrorDescription("MemoryAllocations while dispatching notifications for " + typ),
				InformationCollector: func() iter.Seq[DiagnosticsNode] {
					return slices.Values([]DiagnosticsNode{
						NewDiagnosticsProperty[any]("The "+typ+" sending notification was", event.Object(),
							DiagnosticsPropertyOptions{Style: &style}),
					})
				},
			}))
		}
	}()
	if l := m.listeners[i]; l != nil {
		l(event)
	}
}

// DispatchObjectCreated dispatches an ObjectCreated event, if anyone
// listens.
func (m *FlutterMemoryAllocations) DispatchObjectCreated(library, className string, object any) {
	if !m.HasListeners() {
		return
	}
	m.DispatchObjectEvent(NewObjectCreated(library, className, object))
}

// DispatchObjectDisposed dispatches an ObjectDisposed event, if anyone
// listens.
func (m *FlutterMemoryAllocations) DispatchObjectDisposed(object any) {
	if !m.HasListeners() {
		return
	}
	m.DispatchObjectEvent(NewObjectDisposed(object))
}

func (m *FlutterMemoryAllocations) subscribeToSdkObjects() {
	dartAssert(ui.ImageOnCreate == nil, nil)
	dartAssert(ui.ImageOnDispose == nil, nil)
	dartAssert(ui.PictureOnCreate == nil, nil)
	dartAssert(ui.PictureOnDispose == nil, nil)
	ui.ImageOnCreate = m.imageOnCreate
	ui.ImageOnDispose = m.imageOnDispose
	ui.PictureOnCreate = m.pictureOnCreate
	ui.PictureOnDispose = m.pictureOnDispose
}

func (m *FlutterMemoryAllocations) unSubscribeFromSdkObjects() {
	// Dart asserts the hooks are still its own tear-offs; Go method values
	// have no identity, so only their presence is checked.
	dartAssert(ui.ImageOnCreate != nil, nil)
	dartAssert(ui.ImageOnDispose != nil, nil)
	dartAssert(ui.PictureOnCreate != nil, nil)
	dartAssert(ui.PictureOnDispose != nil, nil)
	ui.ImageOnCreate = nil
	ui.ImageOnDispose = nil
	ui.PictureOnCreate = nil
	ui.PictureOnDispose = nil
}

func (m *FlutterMemoryAllocations) imageOnCreate(image *ui.Image) {
	m.DispatchObjectEvent(NewObjectCreated(dartUILibrary, "Image", image))
}

func (m *FlutterMemoryAllocations) pictureOnCreate(picture *ui.Picture) {
	m.DispatchObjectEvent(NewObjectCreated(dartUILibrary, "Picture", picture))
}

func (m *FlutterMemoryAllocations) imageOnDispose(image *ui.Image) {
	m.DispatchObjectEvent(NewObjectDisposed(image))
}

func (m *FlutterMemoryAllocations) pictureOnDispose(picture *ui.Picture) {
	m.DispatchObjectEvent(NewObjectDisposed(picture))
}
