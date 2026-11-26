// Go-only tests of the binding against the engine (dart:ui's
// PlatformConfiguration): what reaches the embedder, which Dart tests can't
// observe (PlatformDispatcher.scheduleFrame is native there).
package scheduler_test

import (
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
)

type recordingEngine struct{ log *[]string }

func (e recordingEngine) ScheduleFrame()             { *e.log = append(*e.log, "engine.scheduleFrame") }
func (e recordingEngine) EndWarmUpFrame()            { *e.log = append(*e.log, "engine.endWarmUpFrame") }
func (e recordingEngine) SetNeedsReportTimings(bool) {}

func TestEngineDrivesFrames(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := fakeAsync(t)
	var log []string
	pc := ui.NewPlatformConfiguration(recordingEngine{&log})
	t.Cleanup(ui.SetPlatformConfiguration(pc))
	b := newPlainBinding()
	pd := ui.PlatformDispatcherInstance()

	t.Run("an engine frame postponed by the warm-up frame is requested again", func(t *testing.T) {
		ticker := NewTicker(func(e time.Duration) { log = append(log, "tick "+e.String()) })
		t.Cleanup(ticker.Dispose)
		ticker.Start() // ticks in the warm-up frame and reschedules (hasScheduledFrame stays true)
		b.ScheduleFrameCallback(func(time.Duration) {
			async.ScheduleMicrotask(func() {
				b.EnsureFrameCallbacksRegistered()
				pd.OnBeginFrame()(0)
				pd.OnDrawFrame()()
				log = append(log, "engine frame postponed")
			})
		})
		log = nil
		b.ScheduleWarmUpFrame()
		loop.Elapse(0)
		got := strings.Join(log, ",")
		// After the postponed frame, the warm-up's post-frame callback
		// resets hasScheduledFrame and asks the engine for a frame.
		want := "tick 0s,engine.scheduleFrame,engine frame postponed,engine.scheduleFrame,engine.endWarmUpFrame"
		if got != want {
			t.Fatalf("got  %s\nwant %s", got, want)
		}
	})

	t.Run("BeginFrame runs begin, the microtasks, then draw", func(t *testing.T) {
		log = nil
		b.ScheduleFrameCallback(func(ts time.Duration) {
			log = append(log, "transient phase="+b.SchedulerPhase().Name())
			async.ScheduleMicrotask(func() { log = append(log, "microtask phase="+b.SchedulerPhase().Name()) })
		})
		b.AddPostFrameCallback(func(time.Duration) { log = append(log, "post") })
		pc.BeginFrame(time.Second, 5)
		want := "transient phase=transientCallbacks,microtask phase=midFrameMicrotasks,post"
		if got := strings.Join(log, ","); got != want {
			t.Fatalf("got  %s\nwant %s", got, want)
		}
		equal(t, b.CurrentSystemFrameTimeStamp(), time.Second)
		equal(t, pd.FrameData().FrameNumber, 5)
		equal(t, b.HasScheduledFrame(), false)
	})
}
