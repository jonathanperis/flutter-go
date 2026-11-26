// packages/flutter/test/scheduler/benchmarks_test.dart (Flutter 3.50.1) is
// not ported: its one test ("test pumpBenchmark() only runs one frame")
// needs LiveTestWidgetsFlutterBinding, benchmarkWidgets and
// tester.pumpBenchmark (flutter_test's live binding), which are not ported.
// Its rule (microtasks scheduled in handleBeginFrame run before
// handleDrawFrame) is checked in flutter_test's binding by
// TestBenchmarkRuleMicrotasksBeforeDrawFrame.
package scheduler_test

import (
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

func TestBenchmarksTestDart(t *testing.T) {
	t.Run("test pumpBenchmark() only runs one frame", func(t *testing.T) {
		t.Skip("needs LiveTestWidgetsFlutterBinding and benchmarkWidgets (not ported)")
	})
}

func TestBenchmarkRuleMicrotasksBeforeDrawFrame(t *testing.T) {
	if !isolated(t) {
		return
	}
	tester := flutter_test.New(t)
	b := tester.Binding()
	framesBegun, framesDrawn := 0, 0
	handleBeginFrameMicrotaskRun := false
	var errs []string
	b.AddPersistentFrameCallback(func(time.Duration) {
		if !handleBeginFrameMicrotaskRun {
			errs = append(errs, "Microtasks scheduled by 'handledBeginFrame' must be run before 'handleDrawFrame'.")
		}
		framesDrawn++
	})
	b.ScheduleFrameCallback(func(time.Duration) {
		framesBegun++
		async.FutureMicrotask(func() async.Void { handleBeginFrameMicrotaskRun = true; return async.Void{} })
	})
	tester.PumpWidget(widgets.SizedBox{})
	equal(t, errs, []string(nil))
	equal(t, framesBegun, 1)
	equal(t, framesDrawn, 1)
	equal(t, b.SchedulerPhase(), scheduler.SchedulerPhaseIdle)
	// flutter_test's frame time stamps: DateTime.utc(2015) on the fake
	// clock, in microseconds since the epoch.
	tester.Pump(16 * time.Millisecond)
	b.ScheduleFrame()
	tester.Pump(time.Millisecond)
	equal(t, b.CurrentSystemFrameTimeStamp(), 1420070400*time.Second+17*time.Millisecond)
}
