// Port of packages/flutter/test/scheduler/binding_test.dart (Flutter
// 3.50.1): all 5 tests. WidgetsFlutterBinding.ensureInitialized() is a
// binding of BindingBase and SchedulerBinding (the rest of
// WidgetsFlutterBinding is not ported; these tests only use the
// scheduler). The tearDown that awaits the forced-frame ticker's future
// runs the frame that completes it.
package scheduler_test

import (
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
)

func TestBindingTestDart(t *testing.T) {
	if !isolated(t) {
		return
	}
	newPlainBinding() // WidgetsFlutterBinding.ensureInitialized()

	t.Run("scheduleForcedFrame sets up frame callbacks", func(t *testing.T) {
		SchedulerBindingInstance().ScheduleForcedFrame()
		if SchedulerBindingInstance().PlatformDispatcher().OnBeginFrame() == nil {
			t.Fatal("onBeginFrame is null")
		}
	})

	t.Run("Ticker.forceFrames requests forced frames", func(t *testing.T) {
		loop := fakeAsync(t)
		ticker := NewTicker(func(time.Duration) {})
		ticker.ForceFrames = true
		f := ticker.Start()
		t.Cleanup(func() {
			ticker.Stop()
			done := false
			async.AsyncVoid(func() { f.Await(); done = true })
			loop.FlushMicrotasks()
			equal(t, done, true)
		})
		// A forced frame should be scheduled even if frames are otherwise disabled.
		equal(t, SchedulerBindingInstance().HasScheduledFrame(), true)
	})

	t.Run("debugAssertNoTimeDilation does not throw if time dilate already reset", func(t *testing.T) {
		SetTimeDilation(2.0)
		SetTimeDilation(1.0)
		SchedulerBindingInstance().DebugAssertNoTimeDilation("reason") // no error
	})

	t.Run("debugAssertNoTimeDilation throw if time dilate not reset", func(t *testing.T) {
		SetTimeDilation(3.0)
		err := expectPanics[*foundation.FlutterError](t, func() {
			SchedulerBindingInstance().DebugAssertNoTimeDilation("reason")
		})
		equal(t, err.Message(), "reason")
		SetTimeDilation(1.0)
	})

	t.Run("Adding a persistent frame callback during a persistent frame callback", func(t *testing.T) {
		calledBack := false
		SchedulerBindingInstance().AddPersistentFrameCallback(func(time.Duration) {
			if !calledBack {
				SchedulerBindingInstance().AddPersistentFrameCallback(func(time.Duration) {
					calledBack = true
				})
			}
		})
		SchedulerBindingInstance().HandleBeginFrame(nil)
		SchedulerBindingInstance().HandleDrawFrame()
		equal(t, calledBack, false)
		SchedulerBindingInstance().HandleBeginFrame(nil)
		SchedulerBindingInstance().HandleDrawFrame()
		equal(t, calledBack, true)
	})
}
