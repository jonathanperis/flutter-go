package gestures

// Port of pointer_router.dart (Flutter 3.50.1).
//
// Dart keys its route maps by the route function (LinkedHashMaps:
// insertion order); here they are ordered lists of entries compared with
// dart.FunctionsEqual (a method value of a pointer receiver equals another
// of the same receiver and method, as Dart's tear-offs do).

import (
	"iter"
	"slices"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// PointerRoute is a callback that receives a PointerEvent.
type PointerRoute func(event PointerEvent)

type routeEntry struct {
	route     PointerRoute
	transform *vm.Matrix4
}

// routeMap is a Map<PointerRoute, Matrix4?> object (insertion order; used
// by pointer, as the Dart map's identity matters while routing). Its
// entries slice is never mutated in place, so a slice of it is a copy.
type routeMap struct{ entries []routeEntry }

func (m *routeMap) index(route PointerRoute) int {
	return slices.IndexFunc(m.entries, func(e routeEntry) bool { return dart.FunctionsEqual(e.route, route) })
}

func (m *routeMap) containsKey(route PointerRoute) bool { return m.index(route) >= 0 }

func (m *routeMap) add(route PointerRoute, transform *vm.Matrix4) {
	m.entries = append(slices.Clone(m.entries), routeEntry{route, transform})
}

func (m *routeMap) remove(i int) { m.entries = slices.Delete(slices.Clone(m.entries), i, i+1) }

// PointerRouter is a routing table for PointerEvent events.
type PointerRouter struct {
	routeMap     map[int]*routeMap
	globalRoutes *routeMap
}

// NewPointerRouter creates an empty router.
func NewPointerRouter() *PointerRouter {
	return &PointerRouter{routeMap: map[int]*routeMap{}, globalRoutes: &routeMap{}}
}

// AddRoute adds a route to the routing table: whenever the router routes a
// PointerEvent for pointer, it calls route with the event, transformed by
// transform (optional, Dart's [Matrix4? transform]). Routes added while
// routing an event are not called for it.
func (r *PointerRouter) AddRoute(pointer int, route PointerRoute, transform ...*vm.Matrix4) {
	routes, ok := r.routeMap[pointer]
	if !ok {
		routes = &routeMap{}
		r.routeMap[pointer] = routes
	}
	dartAssert(!routes.containsKey(route), nil)
	routes.add(route, optTransform(transform))
}

func optTransform(transform []*vm.Matrix4) *vm.Matrix4 {
	if len(transform) > 0 {
		return transform[0]
	}
	return nil
}

// RemoveRoute removes a route from the routing table; it must be there.
// Routes removed while routing an event are not called for it.
func (r *PointerRouter) RemoveRoute(pointer int, route PointerRoute) {
	routes, ok := r.routeMap[pointer]
	dartAssert(ok, nil)
	i := routes.index(route)
	dartAssert(i >= 0, nil)
	routes.remove(i)
	if len(routes.entries) == 0 {
		delete(r.routeMap, pointer)
	}
}

// AddGlobalRoute adds a route that receives every pointer event,
// transformed by transform (optional).
func (r *PointerRouter) AddGlobalRoute(route PointerRoute, transform ...*vm.Matrix4) {
	dartAssert(!r.globalRoutes.containsKey(route), nil)
	r.globalRoutes.add(route, optTransform(transform))
}

// RemoveGlobalRoute removes a route added by AddGlobalRoute.
func (r *PointerRouter) RemoveGlobalRoute(route PointerRoute) {
	i := r.globalRoutes.index(route)
	dartAssert(i >= 0, nil)
	r.globalRoutes.remove(i)
}

// DebugGlobalRouteCount is the number of global routes (debug builds; it
// panics with UnsupportedError in release builds).
func (r *PointerRouter) DebugGlobalRouteCount() int {
	if !foundation.KDebugMode {
		panic(dart.UnsupportedError{Message: "debugGlobalRouteCount is not supported in release builds"})
	}
	return len(r.globalRoutes.entries)
}

func (r *PointerRouter) dispatch(event PointerEvent, route PointerRoute, transform *vm.Matrix4) {
	defer func() {
		if exception := recover(); exception != nil {
			stack := dart.StackTraceFromPanic()
			var collector foundation.InformationCollector
			if foundation.KDebugMode {
				debug := foundation.DiagnosticLevelDebug
				ev := event
				collector = func() iter.Seq[foundation.DiagnosticsNode] {
					return slices.Values([]foundation.DiagnosticsNode{
						foundation.NewDiagnosticsProperty("router", r, foundation.DiagnosticsPropertyOptions{Level: &debug}),
						foundation.NewDiagnosticsProperty("route", route, foundation.DiagnosticsPropertyOptions{Level: &debug}),
						foundation.NewDiagnosticsProperty("event", ev, foundation.DiagnosticsPropertyOptions{Level: &debug}),
					})
				}
			}
			library := "gesture library"
			foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
				Stack:                stack,
				Library:              &library,
				Context:              foundation.NewErrorDescription("while routing a pointer event"),
				InformationCollector: collector,
			}))
		}
	}()
	event = event.Transformed(transform)
	route(event)
}

// Route calls the routes registered for event's pointer, then the global
// routes. Exceptions thrown by routes are reported to
// FlutterErrorReportError.
func (r *PointerRouter) Route(event PointerEvent) {
	routes := r.routeMap[event.Pointer()]
	copiedGlobalRoutes := r.globalRoutes.entries
	if routes != nil {
		r.dispatchEventToRoutes(event, routes, routes.entries)
	}
	r.dispatchEventToRoutes(event, r.globalRoutes, copiedGlobalRoutes)
}

// dispatchEventToRoutes calls the copied routes still in the reference
// map (routes may be removed meanwhile).
func (r *PointerRouter) dispatchEventToRoutes(event PointerEvent, referenceRoutes *routeMap, copiedRoutes []routeEntry) {
	for _, entry := range copiedRoutes {
		if referenceRoutes.containsKey(entry.route) {
			r.dispatch(event, entry.route, entry.transform)
		}
	}
}
