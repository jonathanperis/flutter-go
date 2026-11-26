// Port of packages/flutter/test/scheduler/scheduler_test.dart (Flutter
// 3.50.1): 8 of its 9 tests.
//
// Skipped: "Animation frame scheduled in the middle of the warm-up frame",
// skipped upstream too (flaky, flutter/flutter#166470); it also needs a
// zone that intercepts timers to run the warm-up frame's two timers one at
// a time (the Go event loop has no zones), so its rule (an engine frame
// that begins during the warm-up frame is postponed with a post-frame
// callback) is checked by a Go-only test below instead, where the engine
// frame arrives in a microtask between the two timers.
//
// Adaptations: TestSchedulerBinding mixes in only SchedulerBinding
// (ServicesBinding is not ported; these tests don't use it). The two tests
// that intercept timers with a zone ("2 calls to scheduleWarmUpFrame just
// schedules it once") count the loop's pending timers instead and run them
// with the loop. `await null` is awaiting a completed future.
package scheduler_test

import (
	"errors"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
)

type testSchedulerBinding struct {
	SchedulerBindingMixin
	eventsDispatched map[string][]map[string]any

	additionalHandleBeginFrame func()
	additionalHandleDrawFrame  func()
}

func (b *testSchedulerBinding) HandleBeginFrame(rawTimeStamp *time.Duration) {
	if b.additionalHandleBeginFrame != nil {
		b.additionalHandleBeginFrame()
	}
	b.SchedulerBindingMixin.HandleBeginFrame(rawTimeStamp)
}

func (b *testSchedulerBinding) HandleDrawFrame() {
	if b.additionalHandleDrawFrame != nil {
		b.additionalHandleDrawFrame()
	}
	b.SchedulerBindingMixin.HandleDrawFrame()
}

func (b *testSchedulerBinding) PostEvent(eventKind string, eventData map[string]any) {
	b.eventsDispatched[eventKind] = append(b.getEventsDispatched(eventKind), eventData)
}

func (b *testSchedulerBinding) getEventsDispatched(eventKind string) []map[string]any {
	return b.eventsDispatched[eventKind]
}

func (b *testSchedulerBinding) tearDown() {
	b.additionalHandleBeginFrame = nil
	b.additionalHandleDrawFrame = nil
	ui.PlatformDispatcherInstance().SetOnBeginFrame(nil)
	ui.PlatformDispatcherInstance().SetOnDrawFrame(nil)
}

// registerFrameCallbacks ensures the PlatformDispatcher's onBeginFrame and
// onDrawFrame callbacks are registered.
func (b *testSchedulerBinding) registerFrameCallbacks() { b.EnsureFrameCallbacksRegistered() }

func newTestSchedulerBinding() *testSchedulerBinding {
	b := &testSchedulerBinding{eventsDispatched: map[string][]map[string]any{}}
	construct(b, &b.SchedulerBindingMixin)
	return b
}

type testStrategy struct{ allowedPriority int }

func (s *testStrategy) shouldRunTaskWithPriority(priority int, scheduler SchedulerBinding) bool {
	return priority >= s.allowedPriority
}

func TestSchedulerTestDart(t *testing.T) {
	if !isolated(t) {
		return
	}
	scheduler := newTestSchedulerBinding() // setUpAll
	run := func(name string, body func(t *testing.T)) {
		t.Run(name, func(t *testing.T) {
			t.Cleanup(scheduler.tearDown)
			body(t)
		})
	}

	run("Tasks are executed in the right order", func(t *testing.T) {
		fakeAsync(t)
		strategy := &testStrategy{allowedPriority: 10000}
		scheduler.SetSchedulingStrategy(strategy.shouldRunTaskWithPriority)
		input := []int{2, 23, 23, 11, 0, 80, 3}
		var executedTasks []int

		scheduleAddingTask := func(x int) {
			ScheduleTask(scheduler, func() async.Void {
				executedTasks = append(executedTasks, x)
				return async.Void{}
			}, PriorityIdle.Add(x))
		}

		for _, x := range input {
			scheduleAddingTask(x)
		}

		strategy.allowedPriority = 100
		for i := 0; i < 3; i++ {
			equal(t, scheduler.HandleEventLoopCallback(), true)
		}
		equal(t, len(executedTasks) == 0, true)

		strategy.allowedPriority = 50
		for i := 0; i < 3; i++ {
			equal(t, scheduler.HandleEventLoopCallback(), true)
		}
		equal(t, executedTasks, []int{80})
		executedTasks = nil

		strategy.allowedPriority = 20
		for i := 0; i < 3; i++ {
			equal(t, scheduler.HandleEventLoopCallback(), true)
		}
		equal(t, executedTasks, []int{23, 23})
		executedTasks = nil

		scheduleAddingTask(99)
		scheduleAddingTask(19)
		scheduleAddingTask(5)
		scheduleAddingTask(97)
		for i := 0; i < 3; i++ {
			equal(t, scheduler.HandleEventLoopCallback(), true)
		}
		equal(t, executedTasks, []int{99, 97})
		executedTasks = nil

		strategy.allowedPriority = 10
		for i := 0; i < 3; i++ {
			equal(t, scheduler.HandleEventLoopCallback(), true)
		}
		equal(t, executedTasks, []int{19, 11})
		executedTasks = nil

		strategy.allowedPriority = 1
		for i := 0; i < 4; i++ {
			equal(t, scheduler.HandleEventLoopCallback(), true)
		}
		equal(t, executedTasks, []int{5, 3, 2})
		executedTasks = nil

		strategy.allowedPriority = 0
		equal(t, scheduler.HandleEventLoopCallback(), false)
		equal(t, executedTasks, []int{0})
	})

	run("scheduleWarmUpFrame should flush microtasks between callbacks", func(t *testing.T) {
		loop := fakeAsync(t)
		t.Cleanup(func() { scheduler.HandleEventLoopCallback() })

		microtaskDone := false
		drawFrameDone := async.NewCompleter[async.Void]()
		scheduler.additionalHandleBeginFrame = func() {
			equal(t, microtaskDone, false)
			async.ScheduleMicrotask(func() { microtaskDone = true })
		}
		scheduler.additionalHandleDrawFrame = func() {
			equal(t, microtaskDone, true)
			drawFrameDone.Complete(async.Void{})
		}
		scheduler.ScheduleWarmUpFrame()
		done := false
		async.OnComplete(drawFrameDone.Future(), func(async.Void) { done = true }, nil)
		loop.FlushTimers(time.Minute, false) // the event loop runs until the test's await returns
		equal(t, done, true)
	})

	run("2 calls to scheduleWarmUpFrame just schedules it once", func(t *testing.T) {
		loop := fakeAsync(t)
		taskExecuted := false
		// Run it twice without processing the queued tasks.
		scheduler.ScheduleWarmUpFrame()
		scheduler.ScheduleWarmUpFrame()
		ScheduleTask(scheduler, func() async.Void {
			taskExecuted = true
			return async.Void{}
		}, PriorityTouch)

		// scheduleWarmUpFrame scheduled 2 Timers, scheduleTask scheduled 0 because
		// events are locked.
		equal(t, len(loop.PendingTimers()), 2)
		equal(t, taskExecuted, false)

		// Run the timers so that the scheduler is no longer in warm-up state.
		loop.Elapse(0)

		// As events are locked, make scheduleTask execute after the test or it
		// will execute during following tests and risk failure.
		t.Cleanup(func() { scheduler.HandleEventLoopCallback() })
	})

	run("Flutter.Frame event fired", func(t *testing.T) {
		frameNumber := 1991
		SchedulerBindingInstance().PlatformDispatcher().OnReportTimings()([]ui.FrameTiming{
			ui.NewFrameTiming(5000, 10000, 15000, 16000, 20000, 20010, ui.FrameTimingOptions{FrameNumber: &frameNumber}),
		})

		events := scheduler.getEventsDispatched("Flutter.Frame")
		equal(t, len(events), 1)

		event := events[0]
		equal(t, event["number"], 1991)
		equal(t, event["startTime"], 10000)
		equal(t, event["elapsed"], int64(15000))
		equal(t, event["build"], int64(5000))
		equal(t, event["raster"], int64(4000))
		equal(t, event["vsyncOverhead"], int64(5000))
	})

	run("TimingsCallback exceptions are caught", func(t *testing.T) {
		var errorCaught *foundation.FlutterErrorDetails
		previous := foundation.FlutterErrorOnError
		t.Cleanup(func() { foundation.FlutterErrorOnError = previous })
		foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
			errorCaught = details
		}
		SchedulerBindingInstance().AddTimingsCallback(func([]ui.FrameTiming) {
			panic(errors.New("Exception: Test"))
		})
		SchedulerBindingInstance().PlatformDispatcher().OnReportTimings()([]ui.FrameTiming{})
		equal(t, errorCaught.ExceptionAsString(), "Exception: Test")
	})

	run("currentSystemFrameTimeStamp is the raw timestamp", func(t *testing.T) {
		// Undo epoch set by previous tests.
		scheduler.ResetEpoch()

		var lastTimeStamp, lastSystemTimeStamp time.Duration

		frameCallback := func(timeStamp time.Duration) {
			equal(t, timeStamp, scheduler.CurrentFrameTimeStamp())
			lastTimeStamp = scheduler.CurrentFrameTimeStamp()
			lastSystemTimeStamp = scheduler.CurrentSystemFrameTimeStamp()
		}

		scheduler.ScheduleFrameCallback(frameCallback)
		tick(2 * time.Second)
		equal(t, lastTimeStamp, time.Duration(0))
		equal(t, lastSystemTimeStamp, 2*time.Second)

		scheduler.ScheduleFrameCallback(frameCallback)
		tick(4 * time.Second)
		equal(t, lastTimeStamp, 2*time.Second)
		equal(t, lastSystemTimeStamp, 4*time.Second)

		SetTimeDilation(2)
		scheduler.ScheduleFrameCallback(frameCallback)
		tick(6 * time.Second)
		equal(t, lastTimeStamp, 2*time.Second) // timeDilation calls SchedulerBinding.resetEpoch
		equal(t, lastSystemTimeStamp, 6*time.Second)

		scheduler.ScheduleFrameCallback(frameCallback)
		tick(8 * time.Second)
		equal(t, lastTimeStamp, 3*time.Second) // 2s + (8 - 6)s / 2
		equal(t, lastSystemTimeStamp, 8*time.Second)

		SetTimeDilation(1.0) // restore time dilation, or it will affect other tests
	})

	run("Animation frame scheduled in the middle of the warm-up frame", func(t *testing.T) {
		t.Skip("skipped upstream (flaky, flutter/flutter#166470); see the microtask version below")
	})

	run("Can schedule futures to completion", func(t *testing.T) {
		loop := fakeAsync(t)
		isCompleted := false

		result := ScheduleTaskFuture(scheduler, func() async.Future[int] {
			return async.Async(func() int {
				// Yield, so if awaiting `result` did not wait for completion of this
				// task, the assertion on `isCompleted` will fail.
				async.FutureValue[any](nil).Await()
				async.FutureValue[any](nil).Await()

				isCompleted = true
				return 1
			})
		}, PriorityIdle)

		scheduler.HandleEventLoopCallback()
		awaited := false
		async.AsyncVoid(func() {
			result.Await()
			awaited = true
			equal(t, isCompleted, true)
		})
		loop.FlushMicrotasks()
		equal(t, awaited, true)
	})

	run("Can schedule a frame callback with / without scheduling a new frame", func(t *testing.T) {
		scheduler.HandleBeginFrame(nil)
		scheduler.HandleDrawFrame()
		callbackInvoked := false

		equal(t, scheduler.HasScheduledFrame(), false)
		noNewFrame := false
		scheduler.ScheduleFrameCallback(func(time.Duration) { callbackInvoked = true },
			ScheduleFrameCallbackOptions{ScheduleNewFrame: &noNewFrame})
		equal(t, scheduler.HasScheduledFrame(), false)
		scheduler.HandleBeginFrame(nil)
		scheduler.HandleDrawFrame()
		equal(t, callbackInvoked, true)

		equal(t, scheduler.HasScheduledFrame(), false)
		callbackInvoked = false
		scheduler.ScheduleFrameCallback(func(time.Duration) { callbackInvoked = true })
		equal(t, scheduler.HasScheduledFrame(), true)
		scheduler.HandleBeginFrame(nil)
		scheduler.HandleDrawFrame()
		equal(t, callbackInvoked, true)

		equal(t, scheduler.HasScheduledFrame(), false)
	})

	// Go-only: the rule of the skipped warm-up test above, without zones:
	// the engine frame arrives in a microtask of the warm-up begin frame,
	// which runs before the warm-up draw timer.
	run("Animation frame scheduled in the middle of the warm-up frame (microtask)", func(t *testing.T) {
		loop := fakeAsync(t)
		t.Cleanup(func() { scheduler.HandleEventLoopCallback() })
		equal(t, scheduler.SchedulerPhase(), SchedulerPhaseIdle)
		pd := ui.PlatformDispatcherInstance()
		engineFrameRan := false
		scheduler.additionalHandleBeginFrame = func() {
			scheduler.additionalHandleBeginFrame = nil
			async.ScheduleMicrotask(func() {
				scheduler.registerFrameCallbacks()
				// Simulate an animation frame firing between warm-up begin frame and warm-up draw frame.
				equal(t, scheduler.HasScheduledFrame(), false)
				pd.OnBeginFrame()(0)
				equal(t, scheduler.HasScheduledFrame(), false)
				pd.OnDrawFrame()()
				equal(t, scheduler.HasScheduledFrame(), false)
				engineFrameRan = true
			})
		}
		scheduler.ScheduleWarmUpFrame()
		equal(t, len(loop.PendingTimers()), 2)
		// The draw frame part of the warm-up frame will run the post-frame
		// callback that reschedules the engine frame.
		loop.Elapse(0)
		equal(t, engineFrameRan, true)
		equal(t, scheduler.HasScheduledFrame(), true)
	})
}
