package ui

// Partial port of engine/src/flutter/lib/ui/platform_dispatcher.dart
// (Flutter 3.50.1): the PlatformDispatcher singleton, the part of its
// platform configuration that foundation reads (platformBrightness), and
// what the scheduler needs: frame scheduling (onBeginFrame, onDrawFrame,
// scheduleFrame, scheduleWarmUpFrame, frameData), frame timings
// (onReportTimings, FrameTiming, FramePhase), requestDartPerformanceMode
// (DartPerformanceMode) and AppLifecycleState. Views, messages, locales,
// accessibility... come with the services and rendering ports.
//
// Dart's PlatformDispatcher records the zone each callback was set in and
// invokes the callback there (hooks.dart's _invoke). Go has no zones, so
// callbacks are called directly (see API_MAPPING "Bindings, zones and
// statics").
//
// The `external` natives (scheduleFrame, endWarmUpFrame,
// setNeedsReportTimings, requestDartPerformanceMode) go to the engine's
// PlatformConfiguration (platform_configuration.go), which the embedder
// drives.

import (
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// VoidCallback is the signature of callbacks that have no arguments and
// return no data.
type VoidCallback func()

// FrameCallback is the signature of PlatformDispatcher.onBeginFrame: the
// duration is the point at which the current frame interval began, since
// some epoch that is the same for all frames.
type FrameCallback func(duration time.Duration)

// TimingsCallback is the signature of PlatformDispatcher.onReportTimings:
// the timings of recently rasterized frames, earliest first.
type TimingsCallback func(timings []FrameTiming)

// PlatformDispatcher is the most basic interface to the host operating
// system's interface.
type PlatformDispatcher struct {
	configuration platformConfiguration

	onBeginFrame       FrameCallback
	onDrawFrame        VoidCallback
	onReportTimings    TimingsCallback
	frameData          FrameData
	onFrameDataChanged VoidCallback

	// platform_message.go
	onPlatformMessage             PlatformMessageCallback
	initialLifecycleState         string
	initialLifecycleStateAccessed bool
	semantics                     platformDispatcherSemantics // platform_dispatcher_semantics.go
}

// platformConfiguration is _PlatformConfiguration (the fields ported so far)
// with its defaults.
type platformConfiguration struct {
	platformBrightness    Brightness
	accessibilityFeatures AccessibilityFeatures
	semanticsEnabled      bool
}

var platformDispatcherInstance = &PlatformDispatcher{
	configuration: platformConfiguration{platformBrightness: BrightnessLight},
	frameData:     FrameData{FrameNumber: -1},
}

// PlatformDispatcherInstance is PlatformDispatcher.instance, the singleton.
func PlatformDispatcherInstance() *PlatformDispatcher { return platformDispatcherInstance }

// PlatformBrightness is the setting indicating the current brightness mode
// of the host platform (Brightness.light when it has no preference).
func (p *PlatformDispatcher) PlatformBrightness() Brightness {
	return p.configuration.platformBrightness
}

// OnBeginFrame is the callback invoked when any view begins a frame: an
// appropriate time to provide a scene. When possible it is driven by the
// hardware VSync signal, and only called if ScheduleFrame has been called
// since the last time it was invoked.
func (p *PlatformDispatcher) OnBeginFrame() FrameCallback { return p.onBeginFrame }

// SetOnBeginFrame sets OnBeginFrame.
func (p *PlatformDispatcher) SetOnBeginFrame(callback FrameCallback) { p.onBeginFrame = callback }

// beginFrame is called from the engine, via hooks.go.
func (p *PlatformDispatcher) beginFrame(microseconds int64) {
	if p.onBeginFrame != nil {
		p.onBeginFrame(time.Duration(microseconds) * time.Microsecond)
	}
}

// OnDrawFrame is the callback invoked for each frame after OnBeginFrame has
// completed and after the microtask queue has been drained.
func (p *PlatformDispatcher) OnDrawFrame() VoidCallback { return p.onDrawFrame }

// SetOnDrawFrame sets OnDrawFrame.
func (p *PlatformDispatcher) SetOnDrawFrame(callback VoidCallback) { p.onDrawFrame = callback }

// drawFrame is called from the engine, via hooks.go.
func (p *PlatformDispatcher) drawFrame() {
	if p.onDrawFrame != nil {
		p.onDrawFrame()
	}
}

// OnReportTimings is the callback invoked to report the FrameTiming of
// recently rasterized frames (prefer SchedulerBinding.AddTimingsCallback,
// which allows several callbacks).
func (p *PlatformDispatcher) OnReportTimings() TimingsCallback { return p.onReportTimings }

// SetOnReportTimings sets OnReportTimings; the engine only collects timings
// while it is set.
func (p *PlatformDispatcher) SetOnReportTimings(callback TimingsCallback) {
	if (callback == nil) != (p.onReportTimings == nil) {
		p.setNeedsReportTimings(callback != nil)
	}
	p.onReportTimings = callback
}

// setNeedsReportTimings is _setNeedsReportTimings, which the constructor
// sets to _nativeSetNeedsReportTimings.
func (p *PlatformDispatcher) setNeedsReportTimings(value bool) {
	platformConfigurationNativeAPISetNeedsReportTimings(value)
}

// reportTimings is called from the engine, via hooks.go.
func (p *PlatformDispatcher) reportTimings(timings []int64) {
	if len(timings)%frameTimingDataLength != 0 {
		panic(dart.NewAssertionError(nil))
	}
	frameTimings := []FrameTiming{}
	for i := 0; i < len(timings); i += frameTimingDataLength {
		frameTimings = append(frameTimings, newFrameTimingFromData(timings[i:i+frameTimingDataLength]))
	}
	if p.onReportTimings != nil {
		p.onReportTimings(frameTimings)
	}
}

// RequestDartPerformanceMode requests the Dart VM to adjust the GC
// heuristics based on mode. The request may be ignored.
func (p *PlatformDispatcher) RequestDartPerformanceMode(mode DartPerformanceMode) {
	platformConfigurationNativeAPIRequestDartPerformanceMode(int(mode))
}

// ScheduleFrame requests that, at the next appropriate opportunity, the
// OnBeginFrame and OnDrawFrame callbacks be invoked.
func (p *PlatformDispatcher) ScheduleFrame() { platformConfigurationNativeAPIScheduleFrame() }

// ScheduleWarmUpFrame schedules a frame to run as soon as possible, rather
// than waiting for the engine to request one, using beginFrame and
// drawFrame instead of OnBeginFrame and OnDrawFrame. Two timers are used so
// that microtasks flush in between.
func (p *PlatformDispatcher) ScheduleWarmUpFrame(beginFrame, drawFrame VoidCallback) {
	async.TimerRun(beginFrame)
	async.TimerRun(func() {
		drawFrame()
		platformConfigurationNativeAPIEndWarmUpFrame()
	})
}

// updateFrameData is called from the engine via hooks.go.
func (p *PlatformDispatcher) updateFrameData(frameNumber int) {
	previous := p.frameData
	if previous.FrameNumber == frameNumber {
		return
	}
	p.frameData = FrameData{FrameNumber: frameNumber}
	if p.onFrameDataChanged != nil {
		p.onFrameDataChanged()
	}
}

// FrameData is the FrameData of the current frame.
func (p *PlatformDispatcher) FrameData() FrameData { return p.frameData }

// OnFrameDataChanged is invoked when the window updates the FrameData.
func (p *PlatformDispatcher) OnFrameDataChanged() VoidCallback { return p.onFrameDataChanged }

// SetOnFrameDataChanged sets OnFrameDataChanged.
func (p *PlatformDispatcher) SetOnFrameDataChanged(callback VoidCallback) {
	p.onFrameDataChanged = callback
}

// FramePhase is the phase of a frame that FrameTiming reports a timestamp
// for (the order matters: FrameTiming's data is indexed by it).
type FramePhase int

const (
	// FramePhaseVsyncStart is the timestamp of the vsync signal given by the
	// operating system.
	FramePhaseVsyncStart FramePhase = iota
	// FramePhaseBuildStart is when the UI thread starts building a frame.
	FramePhaseBuildStart
	// FramePhaseBuildFinish is when the UI thread finishes building a frame.
	FramePhaseBuildFinish
	// FramePhaseRasterStart is when the raster thread starts rasterizing a
	// frame.
	FramePhaseRasterStart
	// FramePhaseRasterFinish is when the raster thread finishes rasterizing
	// a frame.
	FramePhaseRasterFinish
	// FramePhaseRasterFinishWallTime is when the raster thread finished
	// rasterizing a frame, in wall time.
	FramePhaseRasterFinishWallTime
)

// FramePhaseValues is FramePhase.values.
var FramePhaseValues = []FramePhase{FramePhaseVsyncStart, FramePhaseBuildStart, FramePhaseBuildFinish,
	FramePhaseRasterStart, FramePhaseRasterFinish, FramePhaseRasterFinishWallTime}

func (p FramePhase) String() string {
	return "FramePhase." + [...]string{"vsyncStart", "buildStart", "buildFinish", "rasterStart",
		"rasterFinish", "rasterFinishWallTime"}[p]
}

// frameTimingInfo is _FrameTimingInfo: the non-timestamp data of a
// FrameTiming, after the phases.
type frameTimingInfo int

const (
	frameTimingInfoLayerCacheCount frameTimingInfo = iota
	frameTimingInfoLayerCacheBytes
	frameTimingInfoPictureCacheCount
	frameTimingInfoPictureCacheBytes
	frameTimingInfoFrameNumber
	frameTimingInfoCount
)

// frameTimingDataLength is FrameTiming._dataLength.
var frameTimingDataLength = len(FramePhaseValues) + int(frameTimingInfoCount)

// FrameTiming holds time-related performance metrics of a frame (raw
// timestamps in microseconds, cache counts and sizes, the frame number).
type FrameTiming struct {
	data []int64 // some elements in microseconds, some in bytes, some are counts
}

// FrameTimingOptions are FrameTiming's optional parameters.
type FrameTimingOptions struct {
	LayerCacheCount   int
	LayerCacheBytes   int
	PictureCacheCount int
	PictureCacheBytes int
	// FrameNumber defaults to -1.
	FrameNumber *int
}

// NewFrameTiming constructs a FrameTiming with raw timestamps in
// microseconds (Dart's required named parameters are positional).
func NewFrameTiming(vsyncStart, buildStart, buildFinish, rasterStart, rasterFinish, rasterFinishWallTime int, opts ...FrameTimingOptions) FrameTiming {
	var o FrameTimingOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	frameNumber := -1
	if o.FrameNumber != nil {
		frameNumber = *o.FrameNumber
	}
	return newFrameTimingFromData([]int64{
		int64(vsyncStart), int64(buildStart), int64(buildFinish), int64(rasterStart), int64(rasterFinish),
		int64(rasterFinishWallTime), int64(o.LayerCacheCount), int64(o.LayerCacheBytes),
		int64(o.PictureCacheCount), int64(o.PictureCacheBytes), int64(frameNumber),
	})
}

// newFrameTimingFromData is FrameTiming._.
func newFrameTimingFromData(data []int64) FrameTiming {
	if len(data) != frameTimingDataLength {
		panic(dart.NewAssertionError(nil))
	}
	return FrameTiming{data: append([]int64(nil), data...)}
}

// TimestampInMicroseconds is the timestamp of phase, in microseconds.
func (f FrameTiming) TimestampInMicroseconds(phase FramePhase) int { return int(f.data[phase]) }

func (f FrameTiming) rawDuration(phase FramePhase) time.Duration {
	return time.Duration(f.data[phase]) * time.Microsecond
}

func (f FrameTiming) rawInfo(info frameTimingInfo) int {
	return int(f.data[len(FramePhaseValues)+int(info)])
}

// BuildDuration is the duration to build the frame on the UI thread.
func (f FrameTiming) BuildDuration() time.Duration {
	return f.rawDuration(FramePhaseBuildFinish) - f.rawDuration(FramePhaseBuildStart)
}

// RasterDuration is the duration to rasterize the frame on the raster
// thread.
func (f FrameTiming) RasterDuration() time.Duration {
	return f.rawDuration(FramePhaseRasterFinish) - f.rawDuration(FramePhaseRasterStart)
}

// VsyncOverhead is the duration between receiving the vsync signal and
// starting building the frame.
func (f FrameTiming) VsyncOverhead() time.Duration {
	return f.rawDuration(FramePhaseBuildStart) - f.rawDuration(FramePhaseVsyncStart)
}

// TotalSpan is the timespan between the vsync start and the raster finish.
func (f FrameTiming) TotalSpan() time.Duration {
	return f.rawDuration(FramePhaseRasterFinish) - f.rawDuration(FramePhaseVsyncStart)
}

// LayerCacheCount is the number of layers stored in the raster cache.
func (f FrameTiming) LayerCacheCount() int { return f.rawInfo(frameTimingInfoLayerCacheCount) }

// LayerCacheBytes is the number of bytes used to cache layers.
func (f FrameTiming) LayerCacheBytes() int { return f.rawInfo(frameTimingInfoLayerCacheBytes) }

// LayerCacheMegabytes is LayerCacheBytes in megabytes.
func (f FrameTiming) LayerCacheMegabytes() float64 {
	return float64(f.LayerCacheBytes()) / 1024.0 / 1024.0
}

// PictureCacheCount is the number of pictures stored in the raster cache.
func (f FrameTiming) PictureCacheCount() int { return f.rawInfo(frameTimingInfoPictureCacheCount) }

// PictureCacheBytes is the number of bytes used to cache pictures.
func (f FrameTiming) PictureCacheBytes() int { return f.rawInfo(frameTimingInfoPictureCacheBytes) }

// PictureCacheMegabytes is PictureCacheBytes in megabytes.
func (f FrameTiming) PictureCacheMegabytes() float64 {
	return float64(f.PictureCacheBytes()) / 1024.0 / 1024.0
}

// FrameNumber is the frame key associated with this frame timing.
func (f FrameTiming) FrameNumber() int { return int(f.data[len(f.data)-1]) }

func formatMS(d time.Duration) string {
	return dart.DoubleToString(float64(d.Microseconds())*0.001) + "ms"
}

func (f FrameTiming) String() string {
	return "FrameTiming(buildDuration: " + formatMS(f.BuildDuration()) + ", " +
		"rasterDuration: " + formatMS(f.RasterDuration()) + ", " +
		"vsyncOverhead: " + formatMS(f.VsyncOverhead()) + ", " +
		"totalSpan: " + formatMS(f.TotalSpan()) + ", " +
		"layerCacheCount: " + itoa(f.LayerCacheCount()) + ", " +
		"layerCacheBytes: " + itoa(f.LayerCacheBytes()) + ", " +
		"pictureCacheCount: " + itoa(f.PictureCacheCount()) + ", " +
		"pictureCacheBytes: " + itoa(f.PictureCacheBytes()) + ", " +
		"frameNumber: " + itoa(int(f.data[len(f.data)-1])) + ")"
}

// AppLifecycleState is the state of the application: whether it is
// visible, and whether it has input focus.
type AppLifecycleState int

const (
	// AppLifecycleStateDetached: still hosted by a Flutter engine but
	// detached from any host views (the default before the app runs).
	AppLifecycleStateDetached AppLifecycleState = iota
	// AppLifecycleStateResumed: visible and has input focus.
	AppLifecycleStateResumed
	// AppLifecycleStateInactive: at least one view is visible, none has
	// input focus.
	AppLifecycleStateInactive
	// AppLifecycleStateHidden: all views are hidden.
	AppLifecycleStateHidden
	// AppLifecycleStatePaused: not visible and not responding to input; it
	// may be suspended at any time.
	AppLifecycleStatePaused
)

// AppLifecycleStateValues is AppLifecycleState.values.
var AppLifecycleStateValues = []AppLifecycleState{AppLifecycleStateDetached, AppLifecycleStateResumed,
	AppLifecycleStateInactive, AppLifecycleStateHidden, AppLifecycleStatePaused}

// Name is the enum value's name ("resumed").
func (s AppLifecycleState) Name() string {
	return [...]string{"detached", "resumed", "inactive", "hidden", "paused"}[s]
}

func (s AppLifecycleState) String() string { return "AppLifecycleState." + s.Name() }

// DartPerformanceMode is a hint to the Dart VM's GC heuristics.
type DartPerformanceMode int

const (
	// DartPerformanceModeBalanced is the default mode.
	DartPerformanceModeBalanced DartPerformanceMode = iota
	// DartPerformanceModeLatency optimizes for low latency, at the expense of
	// throughput and memory overhead.
	DartPerformanceModeLatency
	// DartPerformanceModeThroughput optimizes for high throughput, at the
	// expense of latency and memory overhead.
	DartPerformanceModeThroughput
	// DartPerformanceModeMemory optimizes for low memory, at the expense of
	// throughput and latency.
	DartPerformanceModeMemory
)

func (m DartPerformanceMode) String() string {
	return "DartPerformanceMode." + [...]string{"balanced", "latency", "throughput", "memory"}[m]
}

// FrameData is frame-related data (window.dart's FrameData); FrameNumber
// is -1 before the first frame.
type FrameData struct {
	FrameNumber int
}
