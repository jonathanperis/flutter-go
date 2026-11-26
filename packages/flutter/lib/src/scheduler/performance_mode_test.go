// Port of packages/flutter/test/scheduler/performance_mode_test.dart
// (Flutter 3.50.1): all 4 tests. WidgetsFlutterBinding is a binding of
// BindingBase and SchedulerBinding (the rest is not ported).
package scheduler_test

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
)

func TestPerformanceModeTestDart(t *testing.T) {
	if !isolated(t) {
		return
	}
	var binding SchedulerBinding = newPlainBinding() // setUpAll

	requested := func() any {
		if m := binding.DebugGetRequestedPerformanceMode(); m != nil {
			return *m
		}
		return nil
	}

	t.Run("PerformanceModeHandler make one request", func(t *testing.T) {
		requestHandle := binding.RequestPerformanceMode(ui.DartPerformanceModeLatency)
		if requestHandle == nil {
			t.Fatal("requestHandle is null")
		}
		equal(t, requested(), ui.DartPerformanceModeLatency)
		requestHandle.Dispose()
		equal(t, requested(), nil)
	})

	t.Run("PerformanceModeHandler make conflicting requests", func(t *testing.T) {
		requestHandle1 := binding.RequestPerformanceMode(ui.DartPerformanceModeLatency)
		if requestHandle1 == nil {
			t.Fatal("requestHandle1 is null")
		}

		requestHandle2 := binding.RequestPerformanceMode(ui.DartPerformanceModeThroughput)
		if requestHandle2 != nil {
			t.Fatal("requestHandle2 is not null")
		}

		equal(t, requested(), ui.DartPerformanceModeLatency)

		requestHandle1.Dispose()
		equal(t, requested(), nil)
	})

	t.Run("PerformanceModeHandler revert only after last requestor disposed", func(t *testing.T) {
		requestHandle1 := binding.RequestPerformanceMode(ui.DartPerformanceModeLatency)
		if requestHandle1 == nil {
			t.Fatal("requestHandle1 is null")
		}

		equal(t, requested(), ui.DartPerformanceModeLatency)

		requestHandle2 := binding.RequestPerformanceMode(ui.DartPerformanceModeLatency)
		if requestHandle2 == nil {
			t.Fatal("requestHandle2 is null")
		}

		equal(t, requested(), ui.DartPerformanceModeLatency)
		requestHandle1.Dispose()
		equal(t, requested(), ui.DartPerformanceModeLatency)
		requestHandle2.Dispose()
		equal(t, requested(), nil)
	})

	t.Run("PerformanceModeRequestHandle dispatches memory events", func(t *testing.T) {
		areCreateAndDispose(t, memoryEvents[*PerformanceModeRequestHandle](func() {
			binding.RequestPerformanceMode(ui.DartPerformanceModeLatency).Dispose()
		}))
	})

	// Go-only: the engine is asked for balanced when the last request goes.
	t.Run("the last dispose requests balanced from the engine", func(t *testing.T) {
		ui.PlatformDispatcherInstance().RequestDartPerformanceMode(ui.DartPerformanceModeMemory)
		h := binding.RequestPerformanceMode(ui.DartPerformanceModeThroughput)
		equal(t, ui.GetDartPerformanceMode(), ui.DartPerformanceModeMemory) // a request alone doesn't reach the engine
		h.Dispose()
		equal(t, ui.GetDartPerformanceMode(), ui.DartPerformanceModeBalanced)
	})
}
