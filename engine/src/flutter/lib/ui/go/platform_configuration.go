package ui

// The engine side of the frame entry points, which an embedder drives.
//
// Partial port of engine/src/flutter/lib/ui/hooks.dart (the
// `vm:entry-point` functions _beginFrame, _drawFrame and _reportTimings)
// and of lib/ui/window/platform_configuration.cc (Flutter 3.50.1):
// PlatformConfiguration::BeginFrame (begin frame, drain microtasks, draw
// frame), ReportTimings, and PlatformConfigurationNativeApi's
// ScheduleFrame, EndWarmUpFrame, SetNeedsReportTimings and
// RequestDartPerformanceMode, which forward to the embedder's
// PlatformConfigurationClient.
//
// An embedder (src/app's window runner) implements
// PlatformConfigurationClient, installs it with SetPlatformConfiguration,
// and calls BeginFrame when the frame it was asked for is due (vsync). With
// no configuration installed (tests), the natives do nothing, as the
// headless flutter_tester's do not produce frames.

import (
	"log"
	"strconv"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
)

func itoa(i int) string { return strconv.Itoa(i) }

// PlatformConfigurationClient is the part of the engine's
// PlatformConfigurationClient that the ported natives call (partial).
type PlatformConfigurationClient interface {
	// ScheduleFrame requests a frame: the embedder calls BeginFrame at the
	// next vsync.
	ScheduleFrame()
	// EndWarmUpFrame is called after the warm-up frame's draw callback.
	EndWarmUpFrame()
	// SetNeedsReportTimings tells the engine whether to collect frame
	// timings for ReportTimings.
	SetNeedsReportTimings(value bool)
}

// PlatformConfiguration is the engine object that dispatches frames to the
// framework (window/platform_configuration.cc, partial).
type PlatformConfiguration struct {
	client           PlatformConfigurationClient
	lastFrameNumber  uint64
	lastMicroseconds int64

	// platform_message.go
	pendingResponses map[int]PlatformMessageResponse
	nextResponseID   int

	// metrics are the views' metrics (platform_dispatcher_views.go).
	metrics map[int]ViewportMetrics
}

// NewPlatformConfiguration creates the configuration of an engine whose
// client is client.
func NewPlatformConfiguration(client PlatformConfigurationClient) *PlatformConfiguration {
	// The engine adds the implicit view before the isolate runs
	// (RuntimeController::LaunchRootIsolate), with default metrics until
	// the platform reports them.
	return &PlatformConfiguration{client: client, metrics: map[int]ViewportMetrics{kImplicitViewID: NewViewportMetrics()}}
}

var currentPlatformConfiguration *PlatformConfiguration

// SetPlatformConfiguration installs pc as the isolate's configuration
// (UIDartState::Current()->platform_configuration()) and returns a function
// restoring the previous one.
func SetPlatformConfiguration(pc *PlatformConfiguration) (restore func()) {
	prev := currentPlatformConfiguration
	currentPlatformConfiguration = pc
	return func() { currentPlatformConfiguration = prev }
}

// Client is the configuration's client.
func (pc *PlatformConfiguration) Client() PlatformConfigurationClient { return pc.client }

// BeginFrame produces a frame at frameTime (since the embedder's epoch):
// it calls the framework's begin-frame entry point (hooks.dart's
// _beginFrame: PlatformDispatcher.onBeginFrame, then the frame data
// update), drains the microtask queue, then calls the draw-frame entry
// point (onDrawFrame). A frame time older than the last one is clamped, and
// an out-of-order frame number is logged, as the engine does.
func (pc *PlatformConfiguration) BeginFrame(frameTime time.Duration, frameNumber uint64) {
	if pc.lastFrameNumber > frameNumber {
		log.Printf("Frame number is out of order: %d < %d", frameNumber, pc.lastFrameNumber)
	}
	pc.lastFrameNumber = frameNumber

	microseconds := frameTime.Microseconds()
	if pc.lastMicroseconds > microseconds {
		// Do not allow time traveling frametimes
		// (github.com/flutter/flutter/issues/106277).
		log.Printf("Reported frame time is older than the last one; clamping. %d < %d ~= %d",
			microseconds, pc.lastMicroseconds, pc.lastMicroseconds-microseconds)
		microseconds = pc.lastMicroseconds
	}
	pc.lastMicroseconds = microseconds

	hooksBeginFrame(microseconds, int(frameNumber))
	async.Current().FlushMicrotasks()
	hooksDrawFrame()
}

// ReportTimings reports frame timings to the framework: FrameTiming data,
// FramePhase and the info fields for each frame, flattened.
func (pc *PlatformConfiguration) ReportTimings(timings []int64) {
	hooksReportTimings(timings)
}

// hooksBeginFrame is hooks.dart's _beginFrame.
func hooksBeginFrame(microseconds int64, frameNumber int) {
	PlatformDispatcherInstance().beginFrame(microseconds)
	PlatformDispatcherInstance().updateFrameData(frameNumber)
}

// hooksReportTimings is hooks.dart's _reportTimings.
func hooksReportTimings(timings []int64) { PlatformDispatcherInstance().reportTimings(timings) }

// hooksDrawFrame is hooks.dart's _drawFrame.
func hooksDrawFrame() { PlatformDispatcherInstance().drawFrame() }

// ---- PlatformConfigurationNativeApi

func platformConfigurationNativeAPIScheduleFrame() {
	if pc := currentPlatformConfiguration; pc != nil {
		pc.client.ScheduleFrame()
	}
}

func platformConfigurationNativeAPIEndWarmUpFrame() {
	if pc := currentPlatformConfiguration; pc != nil {
		pc.client.EndWarmUpFrame()
	}
}

func platformConfigurationNativeAPISetNeedsReportTimings(value bool) {
	if pc := currentPlatformConfiguration; pc != nil {
		pc.client.SetNeedsReportTimings(value)
	}
}

var currentPerformanceMode = DartPerformanceModeBalanced

// platformConfigurationNativeAPIRequestDartPerformanceMode records the mode
// (PlatformConfigurationNativeApi::RequestDartPerformanceMode). ENGINE GAP:
// Go's runtime has no Dart_SetPerformanceMode; the mode is only recorded
// (see GetDartPerformanceMode).
func platformConfigurationNativeAPIRequestDartPerformanceMode(mode int) int {
	currentPerformanceMode = DartPerformanceMode(mode)
	return 0
}

// GetDartPerformanceMode is the engine's record of the last mode requested
// with PlatformDispatcher.RequestDartPerformanceMode
// (PlatformConfigurationNativeApi::GetDartPerformanceMode).
func GetDartPerformanceMode() DartPerformanceMode { return currentPerformanceMode }
