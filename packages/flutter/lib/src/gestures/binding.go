// Package gestures is Flutter's gestures library: pointer events, the
// pointer router, the gesture arena, hit testing, velocity tracking and
// the gesture recognizers, glued to the engine by GestureBinding.
package gestures

// Port of binding.dart (Flutter 3.50.1): GestureBinding (a binding mixin,
// see foundation/binding_mixins.go), its touch resampler (_Resampler) and
// FlutterErrorDetailsForPointerEventDispatcher. SamplingClock is in
// sampling_clock.go.
//
// Only the implicit view exists in flutter-go's dart:ui, so the device
// pixel ratio of any other view id is unknown (null), as Dart's
// platformDispatcher.view(id:) is for a view that doesn't exist.

import (
	"iter"
	"slices"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// resampler is _Resampler: it resamples the touch events of several
// pointer devices to the frame time.
type resampler struct {
	resamplers              map[int]*PointerEventResampler
	resamplerOrder          []int // the map's insertion order
	frameCallbackScheduled  bool
	frameTime               time.Duration
	frameTimeAge            *dart.Stopwatch
	lastSampleTime          time.Duration
	lastEventTime           time.Duration
	handlePointerEvent      HandleEventCallback
	handleSampleTimeChanged func()
	samplingInterval        time.Duration
	timer                   *async.Timer
}

func newResampler(handlePointerEvent HandleEventCallback, handleSampleTimeChanged func(), samplingInterval time.Duration) *resampler {
	return &resampler{
		resamplers:              map[int]*PointerEventResampler{},
		frameTimeAge:            dart.NewStopwatch(),
		handlePointerEvent:      handlePointerEvent,
		handleSampleTimeChanged: handleSampleTimeChanged,
		samplingInterval:        samplingInterval,
	}
}

func (r *resampler) addOrDispatch(event PointerEvent) {
	if event.Kind() == ui.PointerDeviceKindTouch {
		r.lastEventTime = event.TimeStamp()
		res, ok := r.resamplers[event.Device()]
		if !ok {
			res = NewPointerEventResampler()
			r.resamplers[event.Device()] = res
			r.resamplerOrder = append(r.resamplerOrder, event.Device())
		}
		res.AddEvent(event)
	} else {
		r.handlePointerEvent(event)
	}
}

func (r *resampler) timerActive() bool { return r.timer != nil && r.timer.IsActive() }

func (r *resampler) sample(samplingOffset time.Duration, clock SamplingClock) {
	sched := scheduler.SchedulerBindingInstance()

	if r.frameTime == 0 {
		r.frameTime = time.Duration(clock.Now().UnixMilli()) * time.Millisecond
		r.frameTimeAge = clock.Stopwatch()
		r.frameTimeAge.Start()
	}

	if !r.timerActive() {
		r.timer = async.NewPeriodicTimer(r.samplingInterval, func(*async.Timer) { r.onSampleTimeChanged() })
	}

	samplingIntervalUs := r.samplingInterval.Microseconds()
	elapsedIntervals := int64(r.frameTimeAge.ElapsedMicroseconds()) / samplingIntervalUs
	elapsedUs := elapsedIntervals * samplingIntervalUs
	frameTime := r.frameTime + time.Duration(elapsedUs)*time.Microsecond

	sampleTime := frameTime + samplingOffset
	nextSampleTime := sampleTime + r.samplingInterval

	for _, device := range r.resamplerOrder {
		r.resamplers[device].Sample(sampleTime, nextSampleTime, r.handlePointerEvent)
	}

	r.resamplerOrder = slices.DeleteFunc(r.resamplerOrder, func(device int) bool {
		res := r.resamplers[device]
		if !res.HasPendingEvents() && !res.IsDown() {
			delete(r.resamplers, device)
			return true
		}
		return false
	})

	r.lastSampleTime = sampleTime

	if len(r.resamplers) == 0 {
		r.timer.Cancel()
		return
	}

	if !r.frameCallbackScheduled {
		r.frameCallbackScheduled = true
		sched.AddPostFrameCallback(func(time.Duration) {
			r.frameCallbackScheduled = false
			r.frameTime = sched.CurrentSystemFrameTimeStamp()
			r.frameTimeAge.Reset()
			if r.timer != nil {
				r.timer.Cancel()
			}
			r.timer = async.NewPeriodicTimer(r.samplingInterval, func(*async.Timer) { r.onSampleTimeChanged() })
			r.onSampleTimeChanged()
		}, scheduler.AddPostFrameCallbackOptions{DebugLabel: "Resampler.startTimer"})
	}
}

func (r *resampler) stop() {
	for _, device := range r.resamplerOrder {
		r.resamplers[device].Stop(r.handlePointerEvent)
	}
	r.resamplers = map[int]*PointerEventResampler{}
	r.resamplerOrder = nil
	r.frameTime = 0
	if r.timer != nil {
		r.timer.Cancel()
	}
}

func (r *resampler) onSampleTimeChanged() {
	if foundation.KDebugMode && DebugPrintResamplingMargin {
		resamplingMargin := r.lastEventTime - r.lastSampleTime
		foundation.DebugPrint(dart.DurationToString(resamplingMargin))
	}
	r.handleSampleTimeChanged()
}

// defaultSamplingOffset is _defaultSamplingOffset.
const defaultSamplingOffset = -38 * time.Millisecond

// samplingInterval is _samplingInterval (16667 µs, 60 Hz).
const samplingInterval = 16667 * time.Microsecond

// GestureBinding is the binding for the gesture subsystem (mixin
// GestureBinding on BindingBase implements HitTestable,
// HitTestDispatcher, HitTestTarget): it receives pointer data from the
// engine, hit tests, dispatches the events to the hit test results'
// targets and routes them to the gesture recognizers.
type GestureBinding interface {
	foundation.BindingBase
	HitTestable
	HitTestDispatcher
	HitTestTarget

	// CancelPointer dispatches a PointerCancelEvent for the pointer at the
	// next opportunity.
	CancelPointer(pointer int)
	// PointerRouter is the router that routes pointer events to gesture
	// recognizers.
	PointerRouter() *PointerRouter
	// GestureArena is the gesture arenas used to disambiguate gestures.
	GestureArena() *GestureArenaManager
	// PointerSignalResolver is the resolver that picks one pointer signal
	// handler.
	PointerSignalResolver() *PointerSignalResolver
	// HandlePointerEvent dispatches an event to the targets found by a hit
	// test on its position (resampling touch events if enabled).
	HandlePointerEvent(event PointerEvent)
	// ResetGestureBinding resets the hit tests (protected; for tests).
	ResetGestureBinding()
	// DebugSamplingClock overrides SamplingClock in debug builds
	// (protected; nil by default).
	DebugSamplingClock() SamplingClock
	// SamplingClock is the clock used to sample pointer events.
	SamplingClock() SamplingClock
	// ResamplingEnabled is whether touch events are resampled.
	ResamplingEnabled() bool
	// SetResamplingEnabled sets ResamplingEnabled.
	SetResamplingEnabled(value bool)
	// SamplingOffset is the sampling time relative to the frame time.
	SamplingOffset() time.Duration
	// SetSamplingOffset sets SamplingOffset.
	SetSamplingOffset(value time.Duration)

	gestureBinding() *GestureBindingMixin
}

// GestureBindingMixin is GestureBinding's state and methods, to embed in a
// binding (see foundation/binding_mixins.go): SetSelf, pass it to
// Construct. GestureBinding defines InitInstances and Unlocked.
type GestureBindingMixin struct {
	self GestureBinding

	pendingPointerEvents  []PointerEvent // a Queue
	pointerRouter         *PointerRouter
	gestureArena          *GestureArenaManager
	pointerSignalResolver *PointerSignalResolver
	hitTests              map[int]*HitTestResult
	resamplerValue        *resampler
	resamplingEnabled     bool
	samplingOffset        time.Duration
}

// SetSelf points the mixin at the binding embedding it. Call it once,
// before constructing the binding.
func (m *GestureBindingMixin) SetSelf(self GestureBinding) {
	m.self = self
	m.pointerRouter = NewPointerRouter()
	m.gestureArena = NewGestureArenaManager()
	m.pointerSignalResolver = NewPointerSignalResolver()
	m.hitTests = map[int]*HitTestResult{}
	m.samplingOffset = defaultSamplingOffset
}

func (m *GestureBindingMixin) outer() GestureBinding {
	if m.self == nil {
		panic("gestures: GestureBindingMixin used before SetSelf; call SetSelf with the binding when creating it")
	}
	return m.self
}

func (m *GestureBindingMixin) gestureBinding() *GestureBindingMixin { return m }

var gestureBindingInstance GestureBinding

// GestureBindingInstance is GestureBinding.instance: the current
// GestureBinding (it panics with an explanation if none was created).
func GestureBindingInstance() GestureBinding {
	return foundation.BindingBaseCheckInstance(gestureBindingInstance)
}

func (m *GestureBindingMixin) InitInstances() {
	foundation.BindingBaseSuper(m.outer(), m).InitInstances()
	gestureBindingInstance = m.outer()
	m.outer().PlatformDispatcher().SetOnPointerDataPacket(m.handlePointerDataPacket)
	m.outer().PlatformDispatcher().SetOnHitTest(m.handleHitTest)
	// The velocity trackers read GestureBinding.instance.samplingClock.
	currentSamplingClock = func() SamplingClock { return GestureBindingInstance().SamplingClock() }
}

func (m *GestureBindingMixin) Unlocked() {
	foundation.BindingBaseSuper(m.outer(), m).Unlocked()
	m.flushPointerEventQueue()
}

func (m *GestureBindingMixin) handlePointerDataPacket(packet ui.PointerDataPacket) {
	defer func() {
		if r := recover(); r != nil {
			library := "gestures library"
			foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(r, foundation.FlutterErrorDetailsOptions{
				Stack:   dart.StackTraceFromPanic(),
				Library: &library,
				Context: foundation.NewErrorDescription("while handling a pointer data packet"),
			}))
		}
	}()
	for event := range PointerEventConverterExpand(slices.Values(packet.Data), m.devicePixelRatioForView) {
		m.pendingPointerEvents = append(m.pendingPointerEvents, event)
	}
	if !m.outer().Locked() {
		m.flushPointerEventQueue()
	}
}

func (m *GestureBindingMixin) handleHitTest(request ui.HitTestRequest) ui.HitTestResponse {
	result := NewHitTestResult()
	m.outer().HitTestInView(result, request.Offset, request.View.ViewID())
	hasPlatformView := false
	for entry := range result.Path() {
		if _, ok := entry.Target().(NativeHitTestTarget); ok {
			hasPlatformView = true
			break
		}
	}
	return ui.HitTestResponse{HasPlatformView: hasPlatformView}
}

func (m *GestureBindingMixin) devicePixelRatioForView(viewID int) *float64 {
	view := m.outer().PlatformDispatcher().ImplicitView()
	if view == nil || view.ViewID() != viewID {
		return nil
	}
	dpr := view.DevicePixelRatio()
	return &dpr
}

func (m *GestureBindingMixin) CancelPointer(pointer int) {
	if len(m.pendingPointerEvents) == 0 && !m.outer().Locked() {
		async.ScheduleMicrotask(m.flushPointerEventQueue)
	}
	m.pendingPointerEvents = append([]PointerEvent{NewPointerCancelEvent(PointerCancelEventOptions{Pointer: pointer})},
		m.pendingPointerEvents...)
}

func (m *GestureBindingMixin) flushPointerEventQueue() {
	dartAssert(!m.outer().Locked(), nil)
	for len(m.pendingPointerEvents) > 0 {
		event := m.pendingPointerEvents[0]
		m.pendingPointerEvents = m.pendingPointerEvents[1:]
		m.outer().HandlePointerEvent(event)
	}
}

func (m *GestureBindingMixin) PointerRouter() *PointerRouter { return m.pointerRouter }

func (m *GestureBindingMixin) GestureArena() *GestureArenaManager { return m.gestureArena }

func (m *GestureBindingMixin) PointerSignalResolver() *PointerSignalResolver {
	return m.pointerSignalResolver
}

func (m *GestureBindingMixin) HandlePointerEvent(event PointerEvent) {
	dartAssert(!m.outer().Locked(), nil)
	if m.outer().ResamplingEnabled() {
		m.resampler().addOrDispatch(event)
		m.resampler().sample(m.outer().SamplingOffset(), m.outer().SamplingClock())
		return
	}
	// Stop resampler if resampling is not enabled. This is a no-op if
	// resampling was never enabled.
	m.resampler().stop()
	m.handlePointerEventImmediately(event)
}

func (m *GestureBindingMixin) handlePointerEventImmediately(event PointerEvent) {
	var hitTestResult *HitTestResult
	_, isSignal := event.(PointerSignalEvent)
	switch event.(type) {
	case *PointerDownEvent, *PointerHoverEvent, *PointerPanZoomStartEvent:
		isSignal = true
	}
	switch {
	case isSignal:
		if _, ok := m.hitTests[event.Pointer()]; ok {
			minLevel := foundation.DiagnosticLevelDebug
			panic(dart.NewAssertionError("Pointer of " + event.ToString(foundation.DiagnosticableToStringOptions{MinLevel: &minLevel}) +
				" unexpectedly has a HitTestResult associated with it."))
		}
		hitTestResult = NewHitTestResult()
		m.outer().HitTestInView(hitTestResult, event.Position(), event.ViewID())
		switch event.(type) {
		case *PointerDownEvent, *PointerPanZoomStartEvent:
			m.hitTests[event.Pointer()] = hitTestResult
		}
		if foundation.KDebugMode && DebugPrintHitTestResults {
			minLevel := foundation.DiagnosticLevelDebug
			foundation.DebugPrint(event.ToString(foundation.DiagnosticableToStringOptions{MinLevel: &minLevel}) + ": " + hitTestResult.String())
		}
	default:
		switch event.(type) {
		case *PointerUpEvent, *PointerCancelEvent, *PointerPanZoomEndEvent:
			hitTestResult = m.hitTests[event.Pointer()]
			delete(m.hitTests, event.Pointer())
		default:
			_, isPanZoomUpdate := event.(*PointerPanZoomUpdateEvent)
			if event.Down() || isPanZoomUpdate {
				hitTestResult = m.hitTests[event.Pointer()]
			}
		}
	}
	if foundation.KDebugMode && DebugPrintMouseHoverEvents {
		if _, ok := event.(*PointerHoverEvent); ok {
			foundation.DebugPrint(dart.ToString(event))
		}
	}
	_, isAdded := event.(*PointerAddedEvent)
	_, isRemoved := event.(*PointerRemovedEvent)
	if hitTestResult != nil || isAdded || isRemoved {
		m.outer().DispatchEvent(event, hitTestResult)
	}
}

func (m *GestureBindingMixin) HitTestInView(result *HitTestResult, position ui.Offset, viewID int) {
	result.Add(NewHitTestEntry(m.outer()))
}

// HitTest is the deprecated HitTestable.hitTest (Deprecated: use
// HitTestInView and specify the view to hit test; deprecated after
// v3.11.0-20.0.pre).
func (m *GestureBindingMixin) HitTest(result *HitTestResult, position ui.Offset) {
	m.outer().HitTestInView(result, position, m.outer().PlatformDispatcher().ImplicitView().ViewID())
}

func (m *GestureBindingMixin) DispatchEvent(event PointerEvent, hitTestResult *HitTestResult) {
	dartAssert(!m.outer().Locked(), nil)
	errorStyle := foundation.DiagnosticsTreeStyleErrorProperty
	library := "gesture library"
	if hitTestResult == nil {
		switch event.(type) {
		case *PointerAddedEvent, *PointerRemovedEvent:
		default:
			panic(dart.NewAssertionError(nil))
		}
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					stack := dart.StackTraceFromPanic()
					foundation.FlutterErrorReportError(NewFlutterErrorDetailsForPointerEventDispatcher(exception,
						FlutterErrorDetailsForPointerEventDispatcherOptions{
							FlutterErrorDetailsOptions: foundation.FlutterErrorDetailsOptions{
								Stack:   stack,
								Library: &library,
								Context: foundation.NewErrorDescription("while dispatching a non-hit-tested pointer event"),
								InformationCollector: func() iter.Seq[foundation.DiagnosticsNode] {
									return slices.Values([]foundation.DiagnosticsNode{
										foundation.NewDiagnosticsProperty[PointerEvent]("Event", event,
											foundation.DiagnosticsPropertyOptions{Style: &errorStyle}),
									})
								},
							},
							Event: event,
						}).FlutterErrorDetails)
				}
			}()
			m.pointerRouter.Route(event)
		}()
		return
	}
	for entry := range hitTestResult.Path() {
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					stack := dart.StackTraceFromPanic()
					foundation.FlutterErrorReportError(NewFlutterErrorDetailsForPointerEventDispatcher(exception,
						FlutterErrorDetailsForPointerEventDispatcherOptions{
							FlutterErrorDetailsOptions: foundation.FlutterErrorDetailsOptions{
								Stack:   stack,
								Library: &library,
								Context: foundation.NewErrorDescription("while dispatching a pointer event"),
								InformationCollector: func() iter.Seq[foundation.DiagnosticsNode] {
									return slices.Values([]foundation.DiagnosticsNode{
										foundation.NewDiagnosticsProperty[PointerEvent]("Event", event,
											foundation.DiagnosticsPropertyOptions{Style: &errorStyle}),
										foundation.NewDiagnosticsProperty[HitTestTarget]("Target", entry.Target(),
											foundation.DiagnosticsPropertyOptions{Style: &errorStyle}),
									})
								},
							},
							Event:        event,
							HitTestEntry: entry,
						}).FlutterErrorDetails)
				}
			}()
			entry.Target().HandleEvent(event.Transformed(entry.Transform()), entry)
		}()
	}
}

func (m *GestureBindingMixin) HandleEvent(event PointerEvent, entry HitTestEntry) {
	m.pointerRouter.Route(event)
	switch e := event.(type) {
	case *PointerDownEvent, *PointerPanZoomStartEvent:
		m.gestureArena.Close(event.Pointer())
	case *PointerUpEvent, *PointerPanZoomEndEvent:
		m.gestureArena.Sweep(event.Pointer())
	case PointerSignalEvent:
		m.pointerSignalResolver.Resolve(e)
	}
}

func (m *GestureBindingMixin) ResetGestureBinding() { m.hitTests = map[int]*HitTestResult{} }

func (m *GestureBindingMixin) handleSampleTimeChanged() {
	if !m.outer().Locked() {
		if m.outer().ResamplingEnabled() {
			m.resampler().sample(m.outer().SamplingOffset(), m.outer().SamplingClock())
		} else {
			m.resampler().stop()
		}
	}
}

func (m *GestureBindingMixin) DebugSamplingClock() SamplingClock { return nil }

func (m *GestureBindingMixin) SamplingClock() SamplingClock {
	var value SamplingClock = NewSamplingClock()
	if foundation.KDebugMode {
		if debugValue := m.outer().DebugSamplingClock(); debugValue != nil {
			value = debugValue
		}
	}
	return value
}

// resampler is the late final _resampler.
func (m *GestureBindingMixin) resampler() *resampler {
	if m.resamplerValue == nil {
		m.resamplerValue = newResampler(m.handlePointerEventImmediately, m.handleSampleTimeChanged, samplingInterval)
	}
	return m.resamplerValue
}

func (m *GestureBindingMixin) ResamplingEnabled() bool { return m.resamplingEnabled }

func (m *GestureBindingMixin) SetResamplingEnabled(value bool) { m.resamplingEnabled = value }

func (m *GestureBindingMixin) SamplingOffset() time.Duration { return m.samplingOffset }

func (m *GestureBindingMixin) SetSamplingOffset(value time.Duration) { m.samplingOffset = value }

// FlutterErrorDetailsForPointerEventDispatcherOptions are the named
// parameters of FlutterErrorDetailsForPointerEventDispatcher's
// constructor.
type FlutterErrorDetailsForPointerEventDispatcherOptions struct {
	foundation.FlutterErrorDetailsOptions
	// Event is the pointer event being routed when the exception was
	// raised.
	Event PointerEvent
	// HitTestEntry is the hit test result entry whose target threw (nil if
	// the event was not hit tested).
	HitTestEntry HitTestEntry
}

// FlutterErrorDetailsForPointerEventDispatcher is the FlutterErrorDetails
// of an exception thrown while dispatching a pointer event.
type FlutterErrorDetailsForPointerEventDispatcher struct {
	*foundation.FlutterErrorDetails
	// Event is the pointer event being routed when the exception was
	// raised.
	Event PointerEvent
	// HitTestEntry is the hit test result entry whose target threw, or
	// nil.
	HitTestEntry HitTestEntry
}

// NewFlutterErrorDetailsForPointerEventDispatcher creates the details.
func NewFlutterErrorDetailsForPointerEventDispatcher(exception any, opts FlutterErrorDetailsForPointerEventDispatcherOptions) *FlutterErrorDetailsForPointerEventDispatcher {
	return &FlutterErrorDetailsForPointerEventDispatcher{
		FlutterErrorDetails: foundation.NewFlutterErrorDetails(exception, opts.FlutterErrorDetailsOptions),
		Event:               opts.Event,
		HitTestEntry:        opts.HitTestEntry,
	}
}
