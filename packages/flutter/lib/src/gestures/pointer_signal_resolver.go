package gestures

// Port of pointer_signal_resolver.dart (Flutter 3.50.1).

import (
	"iter"
	"slices"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PointerSignalResolvedCallback is the callback to register with a
// PointerSignalResolver to express interest in a pointer signal event.
type PointerSignalResolvedCallback func(event PointerSignalEvent)

// isSameEvent is _isSameEvent: the same event, or transformed versions of
// the same event (events compare by identity).
func isSameEvent(event1, event2 PointerSignalEvent) bool {
	var e1, e2 PointerEvent = event1, event2
	if o := event1.Original(); o != nil {
		e1 = o
	}
	if o := event2.Original(); o != nil {
		e2 = o
	}
	return e1 == e2
}

// PointerSignalResolver mediates disputes over which listener should
// handle pointer signal events when several are interested: the first
// registered callback wins.
type PointerSignalResolver struct {
	firstRegisteredCallback PointerSignalResolvedCallback
	currentEvent            PointerSignalEvent
}

// NewPointerSignalResolver creates a resolver.
func NewPointerSignalResolver() *PointerSignalResolver { return &PointerSignalResolver{} }

// Register registers interest in resolving event; only the first callback
// registered for an event is called by Resolve.
func (r *PointerSignalResolver) Register(event PointerSignalEvent, callback PointerSignalResolvedCallback) {
	dartAssert(r.currentEvent == nil || isSameEvent(r.currentEvent, event), nil)
	if r.firstRegisteredCallback != nil {
		return
	}
	r.currentEvent = event
	r.firstRegisteredCallback = callback
}

// Resolve calls the first registered callback with the event it was
// registered with, then resets. With no registered callback, it lets the
// platform run its default action (event.Respond(true)). Exceptions
// thrown by the callback are reported to FlutterErrorReportError.
func (r *PointerSignalResolver) Resolve(event PointerSignalEvent) {
	if r.firstRegisteredCallback == nil {
		dartAssert(r.currentEvent == nil, nil)
		// Nothing in the framework/app wants to handle the `event`. Allow the
		// platform to trigger any default native actions.
		event.Respond(true)
		return
	}
	dartAssert(isSameEvent(r.currentEvent, event), nil)
	func() {
		defer func() {
			if exception := recover(); exception != nil {
				stack := dart.StackTraceFromPanic()
				var collector foundation.InformationCollector
				if foundation.KDebugMode {
					style := foundation.DiagnosticsTreeStyleErrorProperty
					collector = func() iter.Seq[foundation.DiagnosticsNode] {
						return slices.Values([]foundation.DiagnosticsNode{
							foundation.NewDiagnosticsProperty[PointerSignalEvent]("Event", event,
								foundation.DiagnosticsPropertyOptions{Style: &style}),
						})
					}
				}
				library := "gesture library"
				foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
					Stack:                stack,
					Library:              &library,
					Context:              foundation.NewErrorDescription("while resolving a PointerSignalEvent"),
					InformationCollector: collector,
				}))
			}
		}()
		r.firstRegisteredCallback(r.currentEvent)
	}()
	r.firstRegisteredCallback = nil
	r.currentEvent = nil
}
