// Port of "Service extensions - timeDilation" from
// packages/flutter/test/foundation/service_extensions_test.dart (Flutter
// 3.50.1), the scheduler's extension, with that file's
// TestServiceExtensionsBinding hooks: registerServiceExtension into a map,
// postEvent recorded, scheduleFrame recorded (frameScheduled).
package scheduler_test

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
)

type testServiceExtensionsBinding struct {
	SchedulerBindingMixin
	extensions       map[string]foundation.ServiceExtensionCallback
	eventsDispatched map[string][]map[string]any
	frameScheduled   bool
}

func (b *testServiceExtensionsBinding) RegisterServiceExtension(name string, callback foundation.ServiceExtensionCallback) {
	if _, ok := b.extensions[name]; ok {
		panic("extension " + name + " registered twice")
	}
	b.extensions[name] = callback
}

func (b *testServiceExtensionsBinding) PostEvent(eventKind string, eventData map[string]any) {
	b.eventsDispatched[eventKind] = append(b.eventsDispatched[eventKind], eventData)
}

func (b *testServiceExtensionsBinding) ScheduleFrame() { b.frameScheduled = true }

func (b *testServiceExtensionsBinding) getServiceExtensionStateChangedEvents(extensionName string) []map[string]any {
	var out []map[string]any
	for _, e := range b.eventsDispatched["Flutter.ServiceExtensionStateChanged"] {
		if e["extension"] == extensionName {
			out = append(out, e)
		}
	}
	return out
}

func (b *testServiceExtensionsBinding) testExtension(loop *async.Loop, name string, arguments map[string]string) map[string]any {
	var result map[string]any
	async.OnComplete(b.extensions[name](arguments), func(r map[string]any) { result = r }, nil)
	loop.FlushTimers(0, false)
	return result
}

func TestServiceExtensionsTimeDilation(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := fakeAsync(t)
	binding := &testServiceExtensionsBinding{extensions: map[string]foundation.ServiceExtensionCallback{},
		eventsDispatched: map[string][]map[string]any{}}
	construct(binding, &binding.SchedulerBindingMixin)
	name := SchedulerServiceExtensionsTimeDilation.Name()
	events := func() []map[string]any {
		return binding.getServiceExtensionStateChangedEvents("ext.flutter." + name)
	}

	equal(t, binding.frameScheduled, false)
	equal(t, TimeDilation(), 1.0)
	equal(t, binding.testExtension(loop, name, map[string]string{}), map[string]any{name: "1.0"})
	equal(t, TimeDilation(), 1.0)
	equal(t, len(events()), 0)
	equal(t, binding.testExtension(loop, name, map[string]string{name: "100.0"}), map[string]any{name: "100.0"})
	equal(t, TimeDilation(), 100.0)
	equal(t, len(events()), 1)
	equal(t, events()[0]["extension"], "ext.flutter."+name)
	equal(t, events()[0]["value"], "100.0")
	equal(t, binding.testExtension(loop, name, map[string]string{}), map[string]any{name: "100.0"})
	equal(t, TimeDilation(), 100.0)
	equal(t, len(events()), 1)
	equal(t, binding.testExtension(loop, name, map[string]string{name: "1.0"}), map[string]any{name: "1.0"})
	equal(t, TimeDilation(), 1.0)
	equal(t, len(events()), 2)
	equal(t, events()[1]["extension"], "ext.flutter."+name)
	equal(t, events()[1]["value"], "1.0")
	equal(t, binding.testExtension(loop, name, map[string]string{}), map[string]any{name: "1.0"})
	equal(t, TimeDilation(), 1.0)
	equal(t, len(events()), 2)
	equal(t, binding.frameScheduled, false)
}
