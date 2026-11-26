// Port of packages/flutter/test/gestures/pointer_router_test.dart (Flutter
// 3.50.1): all 9 tests. "Exceptions include router, route & event" is
// vacuous in Dart (addRoute never throws, so its expectations never run);
// here it routes an event to a throwing route and checks that the reported
// error's information names the router, the route and the event.
package gestures_test

import (
	"slices"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

func TestPointerRouterShouldRoutePointers(t *testing.T) {
	callbackRan := false
	callback := func(PointerEvent) { callbackRan = true }
	pointer2 := newTestPointer(2)
	pointer3 := newTestPointer(3)
	router := NewPointerRouter()
	router.AddRoute(3, callback)
	router.Route(pointer2.down(ui.OffsetZero))
	if callbackRan {
		t.Fatal("ran for pointer 2")
	}
	router.Route(pointer3.down(ui.OffsetZero))
	if !callbackRan {
		t.Fatal("didn't run for pointer 3")
	}
	callbackRan = false
	router.RemoveRoute(3, callback)
	router.Route(pointer3.up())
	if callbackRan {
		t.Fatal("ran after removal")
	}
}

func TestPointerRouterReentrantCancellation(t *testing.T) {
	callbackRan := false
	callback := func(PointerEvent) { callbackRan = true }
	router := NewPointerRouter()
	router.AddRoute(2, func(PointerEvent) { router.RemoveRoute(2, callback) })
	router.AddRoute(2, callback)
	router.Route(newTestPointer(2).down(ui.OffsetZero))
	if callbackRan {
		t.Fatal("removed route ran")
	}
}

func TestPointerRouterGlobalCallbacks(t *testing.T) {
	secondCallbackRan := false
	secondCallback := func(PointerEvent) { secondCallbackRan = true }
	firstCallbackRan := false
	router := NewPointerRouter()
	router.AddGlobalRoute(func(PointerEvent) {
		firstCallbackRan = true
		router.AddGlobalRoute(secondCallback)
	})
	router.Route(newTestPointer(2).down(ui.OffsetZero))
	if !firstCallbackRan || secondCallbackRan {
		t.Fatalf("first %v second %v", firstCallbackRan, secondCallbackRan)
	}
}

func TestPointerRouterReentrantGlobalCancellation(t *testing.T) {
	callbackRan := false
	callback := func(PointerEvent) { callbackRan = true }
	router := NewPointerRouter()
	router.AddGlobalRoute(func(PointerEvent) { router.RemoveGlobalRoute(callback) })
	router.AddGlobalRoute(callback)
	router.Route(newTestPointer(2).down(ui.OffsetZero))
	if callbackRan {
		t.Fatal("removed global route ran")
	}
}

func TestPointerRouterPerPointerCannotReentrantlyAddGlobalRoutes(t *testing.T) {
	callbackRan := false
	callback := func(PointerEvent) { callbackRan = true }
	router := NewPointerRouter()
	perPointerCallbackRan := false
	router.AddRoute(2, func(PointerEvent) {
		perPointerCallbackRan = true
		router.AddGlobalRoute(callback)
	})
	router.Route(newTestPointer(2).down(ui.OffsetZero))
	if !perPointerCallbackRan || callbackRan {
		t.Fatalf("per-pointer %v global %v", perPointerCallbackRan, callbackRan)
	}
}

func TestPointerRouterPerPointerBeforeGlobal(t *testing.T) {
	var log []string
	router := NewPointerRouter()
	router.AddGlobalRoute(func(PointerEvent) { log = append(log, "global 1") })
	router.AddRoute(2, func(PointerEvent) { log = append(log, "per-pointer 1") })
	router.AddGlobalRoute(func(PointerEvent) { log = append(log, "global 2") })
	router.AddRoute(2, func(PointerEvent) { log = append(log, "per-pointer 2") })
	router.Route(newTestPointer(2).down(ui.OffsetZero))
	want := []string{"per-pointer 1", "per-pointer 2", "global 1", "global 2"}
	if !slices.Equal(log, want) {
		t.Fatalf("log %v", log)
	}
}

func TestPointerRouterExceptionsDoNotStopRouting(t *testing.T) {
	var log []string
	router := NewPointerRouter()
	router.AddRoute(2, func(PointerEvent) { log = append(log, "per-pointer 1") })
	router.AddRoute(2, func(PointerEvent) {
		log = append(log, "per-pointer 2")
		panic("Having a bad day!")
	})
	router.AddRoute(2, func(PointerEvent) { log = append(log, "per-pointer 3") })
	previous := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(*foundation.FlutterErrorDetails) { log = append(log, "error report") }
	defer func() { foundation.FlutterErrorOnError = previous }()
	router.Route(newTestPointer(2).down(ui.OffsetZero))
	want := []string{"per-pointer 1", "per-pointer 2", "error report", "per-pointer 3"}
	if !slices.Equal(log, want) {
		t.Fatalf("log %v", log)
	}
}

func TestPointerRouterExceptionsIncludeRouterRouteAndEvent(t *testing.T) {
	router := NewPointerRouter()
	router.AddRoute(2, func(PointerEvent) { panic("Pointer exception") })
	var details *foundation.FlutterErrorDetails
	previous := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { details = d }
	defer func() { foundation.FlutterErrorOnError = previous }()
	router.Route(newTestPointer(2).down(ui.OffsetZero))
	if details == nil {
		t.Fatal("no error reported")
	}
	var info []string
	for node := range details.InformationCollector() {
		info = append(info, node.String())
	}
	text := strings.Join(info, "\n")
	for _, want := range []string{"router: Instance of 'PointerRouter'", "route: Closure", "event: PointerDownEvent#"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	if !strings.Contains(text, "(position: Offset(0.0, 0.0))") {
		t.Errorf("event description: %q", text)
	}
	if got := details.Context.String(); got != "while routing a pointer event" {
		t.Errorf("context %q", got)
	}
}

func TestPointerRouterShouldTransformEvents(t *testing.T) {
	var events, globalEvents []PointerEvent
	router := NewPointerRouter()
	m := vm.NewMatrix4Identity()
	m.ScaleByDouble(1/2.0, 1/2.0, 1.0, 1.0)
	transform := m.Multiplied(vm.NewMatrix4TranslationValues(-10, -30, 0))
	router.AddRoute(1, func(e PointerEvent) { events = append(events, e) }, transform)
	router.AddGlobalRoute(func(e PointerEvent) { globalEvents = append(globalEvents, e) }, transform)

	pointer1 := newTestPointer()
	firstPosition := ui.NewOffset(16, 36)
	router.Route(pointer1.down(firstPosition))
	check := func(list []PointerEvent, position, delta, localPosition, localDelta ui.Offset) {
		t.Helper()
		if len(list) != 1 {
			t.Fatalf("%d events", len(list))
		}
		e := list[0]
		if !e.Transform().Equal(transform) || e.Position() != position || e.Delta() != delta ||
			e.LocalPosition() != localPosition || e.LocalDelta() != localDelta {
			t.Fatalf("event %v local %v %v", e.ToStringFull(), e.LocalPosition(), e.LocalDelta())
		}
	}
	check(events, firstPosition, ui.OffsetZero, ui.NewOffset(3, 3), ui.OffsetZero)
	check(globalEvents, firstPosition, ui.OffsetZero, ui.NewOffset(3, 3), ui.OffsetZero)
	events, globalEvents = nil, nil

	newPosition := ui.NewOffset(20, 40)
	router.Route(pointer1.move(newPosition))
	check(events, newPosition, newPosition.Sub(firstPosition), ui.NewOffset(5, 5), ui.NewOffset(2, 2))
	check(globalEvents, newPosition, newPosition.Sub(firstPosition), ui.NewOffset(5, 5), ui.NewOffset(2, 2))
	if _, ok := events[0].(*PointerMoveEvent); !ok {
		t.Fatalf("a transformed move event is a *PointerMoveEvent, got %T", events[0])
	}
}
