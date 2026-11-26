package ui

// Tests of the frame entry points (platform_configuration.go) and the
// PlatformDispatcher members the scheduler uses. The engine has no Dart
// tests for these (they are exercised through the framework); the
// scheduler difftest covers the framework side.

import (
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
)

type fakeClient struct{ log *[]string }

func (c fakeClient) ScheduleFrame()  { *c.log = append(*c.log, "scheduleFrame") }
func (c fakeClient) EndWarmUpFrame() { *c.log = append(*c.log, "endWarmUpFrame") }
func (c fakeClient) SetNeedsReportTimings(v bool) {
	*c.log = append(*c.log, "setNeedsReportTimings "+map[bool]string{true: "true", false: "false"}[v])
}

func TestBeginFrameDrainsMicrotasksBetweenBeginAndDraw(t *testing.T) {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	var log []string
	pd := PlatformDispatcherInstance()
	defer func() { pd.SetOnBeginFrame(nil); pd.SetOnDrawFrame(nil); pd.SetOnFrameDataChanged(nil) }()
	pd.SetOnBeginFrame(func(d time.Duration) {
		log = append(log, "begin "+d.String())
		async.ScheduleMicrotask(func() { log = append(log, "microtask") })
	})
	pd.SetOnDrawFrame(func() { log = append(log, "draw") })
	pd.SetOnFrameDataChanged(func() { log = append(log, "frameData "+itoa(pd.FrameData().FrameNumber)) })
	pc := NewPlatformConfiguration(fakeClient{&log})
	defer SetPlatformConfiguration(pc)()

	pc.BeginFrame(16*time.Millisecond, 1)
	pc.BeginFrame(10*time.Millisecond, 2) // time travel: clamped to 16ms
	pc.BeginFrame(20*time.Millisecond, 2) // same frame number: no frame data change
	want := "begin 16ms,frameData 1,microtask,draw," +
		"begin 16ms,frameData 2,microtask,draw," +
		"begin 20ms,microtask,draw"
	if got := strings.Join(log, ","); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestNativesReachTheClient(t *testing.T) {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	var log []string
	pd := PlatformDispatcherInstance()
	pd.ScheduleFrame() // no configuration: nothing happens
	defer SetPlatformConfiguration(NewPlatformConfiguration(fakeClient{&log}))()
	pd.ScheduleFrame()
	cb := func([]FrameTiming) {}
	pd.SetOnReportTimings(cb)
	pd.SetOnReportTimings(cb) // already set: not reported again
	pd.SetOnReportTimings(nil)
	pd.ScheduleWarmUpFrame(func() { log = append(log, "warm begin") }, func() { log = append(log, "warm draw") })
	if len(loop.PendingTimers()) != 2 {
		t.Fatalf("%d timers, want 2", len(loop.PendingTimers()))
	}
	async.ScheduleMicrotask(func() { log = append(log, "microtask") })
	loop.Elapse(0)
	want := "scheduleFrame,setNeedsReportTimings true,setNeedsReportTimings false,microtask,warm begin,warm draw,endWarmUpFrame"
	if got := strings.Join(log, ","); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	pd.RequestDartPerformanceMode(DartPerformanceModeLatency)
	if GetDartPerformanceMode() != DartPerformanceModeLatency {
		t.Fatal("performance mode not recorded")
	}
	pd.RequestDartPerformanceMode(DartPerformanceModeBalanced)
}

func TestReportTimings(t *testing.T) {
	var got []FrameTiming
	pd := PlatformDispatcherInstance()
	pd.SetOnReportTimings(func(timings []FrameTiming) { got = timings })
	defer pd.SetOnReportTimings(nil)
	pc := NewPlatformConfiguration(nil)
	pc.ReportTimings([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 10, 20, 30, 40, 50, 60, 0, 0, 0, 0, -1})
	if len(got) != 2 || got[0].FrameNumber() != 11 || got[1].TotalSpan() != 40*time.Microsecond ||
		got[0].LayerCacheBytes() != 8 || got[0].TimestampInMicroseconds(FramePhaseRasterFinishWallTime) != 6 {
		t.Fatalf("%v", got)
	}
	if s := got[1].String(); s != "FrameTiming(buildDuration: 0.01ms, rasterDuration: 0.01ms, vsyncOverhead: 0.01ms, totalSpan: 0.04ms, layerCacheCount: 0, layerCacheBytes: 0, pictureCacheCount: 0, pictureCacheBytes: 0, frameNumber: -1)" {
		t.Fatal(s)
	}
	if AppLifecycleStatePaused.String() != "AppLifecycleState.paused" || DartPerformanceModeMemory.String() != "DartPerformanceMode.memory" ||
		FramePhaseBuildFinish.String() != "FramePhase.buildFinish" {
		t.Fatal("enum names")
	}
}
