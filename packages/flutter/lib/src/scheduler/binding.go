// Package scheduler is Flutter's scheduler library: SchedulerBinding
// (frame callbacks, the task queue, frame scheduling through dart:ui's
// PlatformDispatcher), Ticker/TickerFuture, Priority and timeDilation.
package scheduler

// Port of binding.dart (Flutter 3.50.1).
//
// SchedulerBinding is a mixin on BindingBase: here the interface
// SchedulerBinding plus the embeddable SchedulerBindingMixin, whose embedded
// foundation.BindingBase is the binding it is mixed onto ("super"). See
// SchedulerBindingMixin for how a binding is assembled.
//
// The event loop is src/async: Timer.run is async.TimerRun,
// Completers are async.Completer, and the microtasks drained between
// handleBeginFrame and handleDrawFrame are the current async.Loop's (the
// engine drains them in ui.PlatformConfiguration.BeginFrame).

import (
	"iter"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/developer"
)

var timeDilation = 1.0

// TimeDilation slows down animations by this factor to help in
// development (timeDilation).
func TimeDilation() float64 { return timeDilation }

// SetTimeDilation sets TimeDilation. If the SchedulerBinding has been
// initialized, it first calls SchedulerBinding.ResetEpoch so that the time
// stamps consumers of the binding see keep increasing. It is safe to call
// before initializing the binding.
func SetTimeDilation(value float64) {
	dartAssert(value > 0.0, nil)
	if timeDilation == value {
		return
	}
	// If the binding has been created, we need to resetEpoch first so that we
	// capture start of the epoch with the current time dilation.
	if schedulerBindingInstance != nil {
		schedulerBindingInstance.ResetEpoch()
	}
	timeDilation = value
}

// FrameCallback is the signature of frame-related callbacks from the
// scheduler: timeStamp is the time since the beginning of the scheduler's
// epoch, so that all animations are synchronized to a common time base.
type FrameCallback func(timeStamp time.Duration)

// TaskCallback is the signature of ScheduleTask callbacks; T is the task's
// result (async.Void for none). A task that returns a future is a
// ScheduleTaskFuture callback (Dart's FutureOr<T>).
type TaskCallback[T any] func() T

// SchedulingStrategy is the signature of SchedulerBinding's scheduling
// strategy: it reports whether a task of the given priority should run now
// (Dart's required named parameters are positional).
type SchedulingStrategy func(priority int, scheduler SchedulerBinding) bool

// taskEntry is _TaskEntry<T> with T erased, for the queue.
type taskEntry interface {
	priority() int
	run()
	debugStack() dart.StackTrace
}

// taskEntryOf is _TaskEntry<T>.
type taskEntryOf[T any] struct {
	// complete runs the task and completes the completer with its result
	// (completer.complete(task())).
	complete   func(*async.Completer[T])
	prio       int
	debugLabel *string
	flow       *developer.Flow
	stack      dart.StackTrace
	completer  *async.Completer[T]
}

func newTaskEntry[T any](complete func(*async.Completer[T]), priority int, debugLabel *string, flow *developer.Flow) *taskEntryOf[T] {
	e := &taskEntryOf[T]{complete: complete, prio: priority, debugLabel: debugLabel, flow: flow,
		completer: async.NewCompleter[T]()}
	if foundation.KDebugMode {
		e.stack = dart.StackTraceCurrent()
	}
	return e
}

func (e *taskEntryOf[T]) priority() int               { return e.prio }
func (e *taskEntryOf[T]) debugStack() dart.StackTrace { return e.stack }

func (e *taskEntryOf[T]) run() {
	if !foundation.KReleaseMode {
		label := "Scheduled Task"
		if e.debugLabel != nil {
			label = *e.debugLabel
		}
		var opts developer.TimelineOptions
		if e.flow != nil {
			opts.Flow = developer.FlowStep(e.flow.ID)
		}
		developer.TimelineTimeSync(label, func() async.Void {
			e.complete(e.completer)
			return async.Void{}
		}, opts)
	} else {
		e.complete(e.completer)
	}
}

// frameCallbackEntry is _FrameCallbackEntry.
type frameCallbackEntry struct {
	callback   FrameCallback
	debugStack dart.StackTrace
}

// frameCallbackEntryDebugCurrentCallbackStack is
// _FrameCallbackEntry.debugCurrentCallbackStack: the registration stack of
// the transient callback running now (debug builds).
var frameCallbackEntryDebugCurrentCallbackStack dart.StackTrace

func newFrameCallbackEntry(callback FrameCallback, rescheduling bool) *frameCallbackEntry {
	e := &frameCallbackEntry{callback: callback}
	if foundation.KDebugMode {
		if rescheduling {
			if frameCallbackEntryDebugCurrentCallbackStack == nil {
				panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
					foundation.NewErrorSummary("scheduleFrameCallback called with rescheduling true, but no callback is in scope."),
					foundation.NewErrorDescription(`The "rescheduling" argument should only be set to true if the ` +
						`callback is being reregistered from within the callback itself, ` +
						`and only then if the callback itself is entirely synchronous.`),
					foundation.NewErrorHint(`If this is the initial registration of the callback, or if the ` +
						`callback is asynchronous, then do not use the "rescheduling" ` +
						`argument.`),
				}))
			}
			e.debugStack = frameCallbackEntryDebugCurrentCallbackStack
		} else {
			e.debugStack = dart.StackTraceCurrent()
		}
	}
	return e
}

// frameCallbacks is the Map<int, _FrameCallbackEntry> of transient
// callbacks: a LinkedHashMap, iterated in insertion order.
type frameCallbacks struct {
	ids     []int
	entries map[int]*frameCallbackEntry
}

func newFrameCallbacks() *frameCallbacks {
	return &frameCallbacks{entries: map[int]*frameCallbackEntry{}}
}

func (c *frameCallbacks) set(id int, e *frameCallbackEntry) {
	if _, ok := c.entries[id]; !ok {
		c.ids = append(c.ids, id)
	}
	c.entries[id] = e
}

func (c *frameCallbacks) remove(id int) {
	if _, ok := c.entries[id]; !ok {
		return
	}
	delete(c.entries, id)
	if len(c.ids) > 2*len(c.entries)+16 { // drop removed ids now and then
		c.ids = slices.DeleteFunc(c.ids, func(i int) bool { _, ok := c.entries[i]; return !ok })
	}
}

// all iterates the entries in insertion order.
func (c *frameCallbacks) all() iter.Seq2[int, *frameCallbackEntry] {
	return func(yield func(int, *frameCallbackEntry) bool) {
		for _, id := range c.ids {
			if e, ok := c.entries[id]; ok && !yield(id, e) {
				return
			}
		}
	}
}

func (c *frameCallbacks) clone() *frameCallbacks {
	n := newFrameCallbacks()
	for id, e := range c.all() {
		n.set(id, e)
	}
	return n
}

// SchedulerPhase is the phase a SchedulerBinding is in during
// HandleBeginFrame/HandleDrawFrame. The values are ordered as the phases
// occur, so they can be compared.
type SchedulerPhase int

const (
	// SchedulerPhaseIdle: no frame is being processed. Tasks, microtasks,
	// timers, event handlers and other callbacks may be executing.
	SchedulerPhaseIdle SchedulerPhase = iota
	// SchedulerPhaseTransientCallbacks: the transient callbacks (scheduled
	// by ScheduleFrameCallback) are executing; typically animations.
	SchedulerPhaseTransientCallbacks
	// SchedulerPhaseMidFrameMicrotasks: microtasks scheduled during the
	// transient callbacks are executing.
	SchedulerPhaseMidFrameMicrotasks
	// SchedulerPhasePersistentCallbacks: the persistent callbacks
	// (AddPersistentFrameCallback) are executing; typically the
	// build/layout/paint pipeline.
	SchedulerPhasePersistentCallbacks
	// SchedulerPhasePostFrameCallbacks: the post-frame callbacks
	// (AddPostFrameCallback) are executing.
	SchedulerPhasePostFrameCallbacks
)

// SchedulerPhaseValues is SchedulerPhase.values.
var SchedulerPhaseValues = []SchedulerPhase{SchedulerPhaseIdle, SchedulerPhaseTransientCallbacks,
	SchedulerPhaseMidFrameMicrotasks, SchedulerPhasePersistentCallbacks, SchedulerPhasePostFrameCallbacks}

// Name is the enum value's name ("transientCallbacks").
func (p SchedulerPhase) Name() string {
	return [...]string{"idle", "transientCallbacks", "midFrameMicrotasks", "persistentCallbacks",
		"postFrameCallbacks"}[p]
}

func (p SchedulerPhase) String() string { return "SchedulerPhase." + p.Name() }

// PerformanceModeRequestHandle is an opaque handle that keeps a request for
// a DartPerformanceMode active until disposed (create one with
// SchedulerBinding.RequestPerformanceMode).
type PerformanceModeRequestHandle struct {
	cleanup func()
}

func newPerformanceModeRequestHandle(cleanup func()) *PerformanceModeRequestHandle {
	h := &PerformanceModeRequestHandle{cleanup: cleanup}
	dartAssert(foundation.DebugMaybeDispatchCreated("scheduler", "PerformanceModeRequestHandle", h), nil)
	return h
}

// Dispose signals to SchedulerBinding that the request is no longer needed.
// It must only be called once per object.
func (h *PerformanceModeRequestHandle) Dispose() {
	dartAssert(h.cleanup != nil, nil)
	dartAssert(foundation.DebugMaybeDispatchDisposed(h), nil)
	h.cleanup()
	h.cleanup = nil
}

// ScheduleFrameCallbackOptions are ScheduleFrameCallback's named
// parameters.
type ScheduleFrameCallbackOptions struct {
	// Rescheduling: the callback is being re-registered from within itself
	// (debug builds keep its first registration's stack).
	Rescheduling bool
	// ScheduleNewFrame: whether to call ScheduleFrame (nil means true).
	ScheduleNewFrame *bool
}

// AddPostFrameCallbackOptions are AddPostFrameCallback's named parameters.
type AddPostFrameCallbackOptions struct {
	// DebugLabel names the callback in the timeline when
	// DebugTracePostFrameCallbacks is set ("" means the default,
	// "callback").
	DebugLabel string
}

// ScheduleTaskOptions are ScheduleTask's named parameters.
type ScheduleTaskOptions struct {
	// DebugLabel and Flow report the task to the timeline.
	DebugLabel *string
	Flow       *developer.Flow
}

// SchedulerBinding schedules transient callbacks (driven by the engine's
// onBeginFrame, for animations), persistent callbacks (onDrawFrame, for
// the rendering pipeline), post-frame callbacks, and non-rendering tasks
// run between frames by priority.
//
// It is Flutter's `mixin SchedulerBinding on BindingBase`; implement it by
// embedding SchedulerBindingMixin.
type SchedulerBinding interface {
	foundation.BindingBase

	// AddTimingsCallback adds a callback that receives the FrameTimings
	// the engine reports (several callbacks may be added; the same one
	// twice runs twice).
	AddTimingsCallback(callback ui.TimingsCallback)
	// RemoveTimingsCallback removes a callback added by AddTimingsCallback.
	RemoveTimingsCallback(callback ui.TimingsCallback)
	// LifecycleState is whether the application is visible and
	// interactive (nil until the first notification).
	LifecycleState() *ui.AppLifecycleState
	// ResetInternalState lets the test framework reset the lifecycle state
	// and FramesEnabled to their initial values.
	ResetInternalState()
	// HandleAppLifecycleStateChanged is called when the application
	// lifecycle state changes; overrides must call it.
	HandleAppLifecycleStateChanged(state ui.AppLifecycleState)
	// SchedulingStrategy decides whether to run a task
	// (DefaultSchedulingStrategy by default).
	SchedulingStrategy() SchedulingStrategy
	// SetSchedulingStrategy sets SchedulingStrategy.
	SetSchedulingStrategy(strategy SchedulingStrategy)
	// HandleEventLoopCallback runs the highest-priority task if the
	// strategy allows; it returns false if the binding is locked or no
	// task remains, true otherwise (even when nothing ran).
	HandleEventLoopCallback() bool
	// TransientCallbackCount is the number of transient frame callbacks
	// scheduled.
	TransientCallbackCount() int
	// ScheduleFrameCallback schedules a transient frame callback and
	// returns its id (for CancelFrameCallbackWithId). Called during the
	// transient callbacks, it runs in the next frame.
	ScheduleFrameCallback(callback FrameCallback, opts ...ScheduleFrameCallbackOptions) int
	// CancelFrameCallbackWithId cancels a transient frame callback (not
	// the frame request).
	CancelFrameCallbackWithId(id int)
	// DebugAssertNoTransientCallbacks reports an error listing the
	// registration stacks if transient callbacks are registered (debug
	// builds); it returns true.
	DebugAssertNoTransientCallbacks(reason string) bool
	// DebugAssertNoPendingPerformanceModeRequests panics with a
	// FlutterError if a performance mode request is pending (debug
	// builds); it returns true.
	DebugAssertNoPendingPerformanceModeRequests(reason string) bool
	// DebugAssertNoTimeDilation panics with a FlutterError if TimeDilation
	// is not 1 (debug builds); it returns true.
	DebugAssertNoTimeDilation(reason string) bool
	// AddPersistentFrameCallback adds a callback run every frame after the
	// transient callbacks. It does not request a frame and can't be
	// removed.
	AddPersistentFrameCallback(callback FrameCallback)
	// AddPostFrameCallback schedules a callback for the end of this frame
	// (or the next one, if post-frame callbacks have started). It does not
	// request a frame.
	AddPostFrameCallback(callback FrameCallback, opts ...AddPostFrameCallbackOptions)
	// EndOfFrame completes after the current frame, or after the next one
	// (scheduling it when idle).
	EndOfFrame() async.Future[async.Void]
	// HasScheduledFrame reports whether a frame has been requested.
	HasScheduledFrame() bool
	// SchedulerPhase is the phase the scheduler is in.
	SchedulerPhase() SchedulerPhase
	// FramesEnabled reports whether ScheduleFrame schedules frames (it
	// depends on LifecycleState).
	FramesEnabled() bool
	// EnsureFrameCallbacksRegistered registers the binding's handlers as
	// PlatformDispatcher.onBeginFrame/onDrawFrame if unset.
	EnsureFrameCallbacksRegistered()
	// EnsureVisualUpdate schedules a frame unless one is being produced.
	EnsureVisualUpdate()
	// ScheduleFrame requests a frame from the engine
	// (PlatformDispatcher.ScheduleFrame) if needed and frames are enabled.
	ScheduleFrame()
	// ScheduleForcedFrame requests a frame even if frames are disabled.
	ScheduleForcedFrame()
	// ScheduleWarmUpFrame runs a frame as soon as possible, without
	// waiting for vsync, and locks events until it has finished.
	ScheduleWarmUpFrame()
	// ResetEpoch prepares for a non-monotonic change to how time stamps
	// are computed (SetTimeDilation calls it).
	ResetEpoch()
	// CurrentFrameTimeStamp is the time stamp of the frame being
	// processed (only valid during a frame).
	CurrentFrameTimeStamp() time.Duration
	// CurrentSystemFrameTimeStamp is the engine's raw time stamp for the
	// current frame, unadjusted for the epoch and TimeDilation.
	CurrentSystemFrameTimeStamp() time.Duration
	// HandleBeginFrame is called by the engine to prepare a new frame: it
	// runs the transient callbacks. A nil time stamp reuses the last one.
	HandleBeginFrame(rawTimeStamp *time.Duration)
	// RequestPerformanceMode requests a DartPerformanceMode; it returns nil
	// if a different mode is already requested. Dispose the handle when
	// done.
	RequestPerformanceMode(mode ui.DartPerformanceMode) *PerformanceModeRequestHandle
	// DebugGetRequestedPerformanceMode is the mode requested, or nil (always
	// nil in release builds).
	DebugGetRequestedPerformanceMode() *ui.DartPerformanceMode
	// HandleDrawFrame is called by the engine to produce a new frame, after
	// HandleBeginFrame: it runs the persistent callbacks, then the
	// post-frame callbacks.
	HandleDrawFrame()

	schedulerBinding() *SchedulerBindingMixin
}

// SchedulerBindingMixin is SchedulerBinding's state and methods, to embed
// in a binding. Its embedded foundation.BindingBase is the binding it is
// mixed onto (Dart's "super": a *foundation.BindingBaseBase, or the mixin
// below it), whose methods it overrides and calls. A binding embeds the
// topmost mixin, calls SetSelf with itself, then constructs the base:
//
//	type MyBinding struct{ scheduler.SchedulerBindingMixin }
//
//	func NewMyBinding() *MyBinding {
//		b := &MyBinding{}
//		base := &foundation.BindingBaseBase{}
//		b.SchedulerBindingMixin.BindingBase = base
//		b.SetSelf(b)
//		base.Construct(b)
//		return b
//	}
type SchedulerBindingMixin struct {
	foundation.BindingBase

	self SchedulerBinding

	timingsCallbacks        []ui.TimingsCallback
	executeTimingsCallbacks ui.TimingsCallback

	lifecycleState     *ui.AppLifecycleState
	schedulingStrategy SchedulingStrategy

	taskQueue                       heapPriorityQueue
	hasRequestedAnEventLoopCallback bool

	nextFrameCallbackID int // positive
	transientCallbacks  *frameCallbacks
	removedIds          map[int]struct{}

	persistentCallbacks []FrameCallback
	postFrameCallbacks  []FrameCallback
	nextFrameCompleter  *async.Completer[async.Void]

	hasScheduledFrame bool
	schedulerPhase    SchedulerPhase
	framesDisabled    bool // Dart's _framesEnabled, inverted so the zero value is the default
	warmUpFrame       bool

	firstRawTimeStampInEpoch *time.Duration
	epochStart               time.Duration
	lastRawTimeStamp         time.Duration
	currentFrameTimeStamp    *time.Duration

	debugFrameNumber           int
	debugBanner                *string
	rescheduleAfterWarmUpFrame bool
	frameTimelineTaskCreated   bool
	frameTimelineTaskValue     *developer.TimelineTask

	performanceMode            *ui.DartPerformanceMode
	numPerformanceModeRequests int
}

// SetSelf points the mixin at the binding embedding it, whose methods it
// calls so overrides run. Call it once, before constructing the binding.
func (m *SchedulerBindingMixin) SetSelf(self SchedulerBinding) {
	m.self = self
	m.executeTimingsCallbacks = m.executeTimingsCallbacksImpl
}

func (m *SchedulerBindingMixin) outer() SchedulerBinding {
	if m.self == nil {
		panic("scheduler: SchedulerBindingMixin used before SetSelf; call SetSelf with the binding when creating it")
	}
	return m.self
}

func (m *SchedulerBindingMixin) schedulerBinding() *SchedulerBindingMixin { return m }

func (m *SchedulerBindingMixin) InitInstances() {
	m.BindingBase.InitInstances()
	schedulerBindingInstance = m.outer()

	if !foundation.KReleaseMode {
		m.outer().AddTimingsCallback(func(timings []ui.FrameTiming) {
			for _, timing := range timings {
				m.profileFramePostEvent(timing)
			}
		})
	}
}

var schedulerBindingInstance SchedulerBinding

// SchedulerBindingInstance is SchedulerBinding.instance: the current
// SchedulerBinding (it panics with an explanation if none was created).
func SchedulerBindingInstance() SchedulerBinding {
	return foundation.BindingBaseCheckInstance(schedulerBindingInstance)
}

func (m *SchedulerBindingMixin) AddTimingsCallback(callback ui.TimingsCallback) {
	m.timingsCallbacks = append(m.timingsCallbacks, callback)
	pd := m.outer().PlatformDispatcher()
	if len(m.timingsCallbacks) == 1 {
		dartAssert(pd.OnReportTimings() == nil, nil)
		pd.SetOnReportTimings(m.executeTimingsCallbacks)
	}
	dartAssert(funcIdentity(pd.OnReportTimings()) == funcIdentity(m.executeTimingsCallbacks), nil)
}

func (m *SchedulerBindingMixin) RemoveTimingsCallback(callback ui.TimingsCallback) {
	i := slices.IndexFunc(m.timingsCallbacks, func(c ui.TimingsCallback) bool {
		return funcIdentity(c) == funcIdentity(callback)
	})
	dartAssert(i >= 0, nil)
	if i >= 0 {
		m.timingsCallbacks = slices.Delete(m.timingsCallbacks, i, i+1)
	}
	if len(m.timingsCallbacks) == 0 {
		m.outer().PlatformDispatcher().SetOnReportTimings(nil)
	}
}

func (m *SchedulerBindingMixin) executeTimingsCallbacksImpl(timings []ui.FrameTiming) {
	clonedCallbacks := slices.Clone(m.timingsCallbacks)
	for _, callback := range clonedCallbacks {
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					stack := dart.StackTraceFromPanic()
					var collector foundation.InformationCollector
					if foundation.KDebugMode {
						collector = func() iter.Seq[foundation.DiagnosticsNode] {
							style := foundation.DiagnosticsTreeStyleErrorProperty
							return slices.Values([]foundation.DiagnosticsNode{
								foundation.NewDiagnosticsProperty("The TimingsCallback that gets executed was", callback,
									foundation.DiagnosticsPropertyOptions{Style: &style}),
							})
						}
					}
					foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
						Stack:                stack,
						Context:              foundation.NewErrorDescription("while executing callbacks for FrameTiming"),
						InformationCollector: collector,
					}))
				}
			}()
			if slices.ContainsFunc(m.timingsCallbacks, func(c ui.TimingsCallback) bool {
				return funcIdentity(c) == funcIdentity(callback)
			}) {
				callback(timings)
			}
		}()
	}
}

func (m *SchedulerBindingMixin) InitServiceExtensions() {
	m.BindingBase.InitServiceExtensions()

	if !foundation.KReleaseMode {
		m.outer().RegisterNumericServiceExtension(SchedulerServiceExtensionsTimeDilation.Name(),
			func() async.Future[float64] { return async.Async(func() float64 { return timeDilation }) },
			func(value float64) async.Future[async.Void] {
				return async.AsyncVoid(func() { SetTimeDilation(value) })
			})
	}
}

func (m *SchedulerBindingMixin) LifecycleState() *ui.AppLifecycleState { return m.lifecycleState }

func (m *SchedulerBindingMixin) ResetInternalState() {
	m.lifecycleState = nil
	m.framesDisabled = false
}

func (m *SchedulerBindingMixin) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	if current := m.outer().LifecycleState(); current != nil && *current == state {
		return
	}
	m.lifecycleState = &state
	switch state {
	case ui.AppLifecycleStateResumed, ui.AppLifecycleStateInactive:
		m.setFramesEnabledState(true)
	case ui.AppLifecycleStateHidden, ui.AppLifecycleStatePaused, ui.AppLifecycleStateDetached:
		m.setFramesEnabledState(false)
	}
}

func (m *SchedulerBindingMixin) SchedulingStrategy() SchedulingStrategy {
	if m.schedulingStrategy == nil {
		return DefaultSchedulingStrategy
	}
	return m.schedulingStrategy
}

func (m *SchedulerBindingMixin) SetSchedulingStrategy(strategy SchedulingStrategy) {
	m.schedulingStrategy = strategy
}

// taskSorter is _taskSorter: higher priorities first.
func taskSorter(e1, e2 taskEntry) int { return -compareInts(e1.priority(), e2.priority()) }

func compareInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// ScheduleTask is SchedulerBinding.scheduleTask (a generic method in Dart):
// it schedules task with priority, to run between frames in priority order
// (skipping tasks the scheduling strategy refuses), and returns a future of
// its result. Tasks should be short (up to a millisecond).
func ScheduleTask[T any](s SchedulerBinding, task TaskCallback[T], priority Priority, opts ...ScheduleTaskOptions) async.Future[T] {
	return scheduleTask(s, func(c *async.Completer[T]) { c.Complete(task()) }, priority, opts)
}

// ScheduleTaskFuture is ScheduleTask for a task that returns a future: the
// returned future completes with that future's result.
func ScheduleTaskFuture[T any](s SchedulerBinding, task func() async.Future[T], priority Priority, opts ...ScheduleTaskOptions) async.Future[T] {
	return scheduleTask(s, func(c *async.Completer[T]) { c.CompleteFuture(task()) }, priority, opts)
}

func scheduleTask[T any](s SchedulerBinding, complete func(*async.Completer[T]), priority Priority, opts []ScheduleTaskOptions) async.Future[T] {
	var o ScheduleTaskOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	m := s.schedulerBinding()
	isFirstTask := m.taskQueue.isEmpty()
	entry := newTaskEntry(complete, priority.Value(), o.DebugLabel, o.Flow)
	m.taskQueue.add(entry)
	if isFirstTask && !m.outer().Locked() {
		m.ensureEventLoopCallback()
	}
	return entry.completer.Future()
}

func (m *SchedulerBindingMixin) Unlocked() {
	m.BindingBase.Unlocked()
	if m.taskQueue.isNotEmpty() {
		m.ensureEventLoopCallback()
	}
}

// ensureEventLoopCallback makes sure the scheduler services a task
// scheduled by ScheduleTask.
func (m *SchedulerBindingMixin) ensureEventLoopCallback() {
	dartAssert(!m.outer().Locked(), nil)
	dartAssert(m.taskQueue.isNotEmpty(), nil)
	if m.hasRequestedAnEventLoopCallback {
		return
	}
	m.hasRequestedAnEventLoopCallback = true
	async.TimerRun(m.runTasks)
}

// runTasks is scheduled by ensureEventLoopCallback.
func (m *SchedulerBindingMixin) runTasks() {
	m.hasRequestedAnEventLoopCallback = false
	if m.outer().HandleEventLoopCallback() {
		m.ensureEventLoopCallback()
	} // runs next task when there's time
}

func (m *SchedulerBindingMixin) HandleEventLoopCallback() bool {
	self := m.outer()
	if m.taskQueue.isEmpty() || self.Locked() {
		return false
	}
	entry := m.taskQueue.first()
	if self.SchedulingStrategy()(entry.priority(), self) {
		func() {
			defer func() {
				if exception := recover(); exception != nil {
					exceptionStack := dart.StackTraceFromPanic()
					var callbackStack dart.StackTrace
					if foundation.KDebugMode {
						callbackStack = entry.debugStack()
					}
					foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
						Stack:                exceptionStack,
						Library:              strp("scheduler library"),
						Context:              foundation.NewErrorDescription("during a task callback"),
						InformationCollector: registrationStackCollector(callbackStack),
					}))
				}
			}()
			m.taskQueue.removeFirst()
			entry.run()
		}()
		return m.taskQueue.isNotEmpty()
	}
	return true
}

// registrationStackCollector is the informationCollector Flutter attaches
// to errors in scheduler callbacks: the stack the callback was registered
// with (nil when there is none).
func registrationStackCollector(callbackStack dart.StackTrace) foundation.InformationCollector {
	if callbackStack == nil {
		return nil
	}
	return func() iter.Seq[foundation.DiagnosticsNode] {
		return slices.Values([]foundation.DiagnosticsNode{
			foundation.NewDiagnosticsStackTrace(
				"\nThis exception was thrown in the context of a scheduler callback. "+
					"When the scheduler callback was _registered_ (as opposed to when the "+
					"exception was thrown), this was the stack",
				callbackStack,
			),
		})
	}
}

func (m *SchedulerBindingMixin) callbacks() *frameCallbacks {
	if m.transientCallbacks == nil {
		m.transientCallbacks = newFrameCallbacks()
	}
	return m.transientCallbacks
}

func (m *SchedulerBindingMixin) TransientCallbackCount() int { return len(m.callbacks().entries) }

func (m *SchedulerBindingMixin) ScheduleFrameCallback(callback FrameCallback, opts ...ScheduleFrameCallbackOptions) int {
	var o ScheduleFrameCallbackOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.ScheduleNewFrame == nil || *o.ScheduleNewFrame {
		m.outer().ScheduleFrame()
	}
	m.nextFrameCallbackID++
	m.callbacks().set(m.nextFrameCallbackID, newFrameCallbackEntry(callback, o.Rescheduling))
	return m.nextFrameCallbackID
}

func (m *SchedulerBindingMixin) CancelFrameCallbackWithId(id int) {
	dartAssert(id > 0, nil)
	m.callbacks().remove(id)
	if m.removedIds == nil {
		m.removedIds = map[int]struct{}{}
	}
	m.removedIds[id] = struct{}{}
}

func (m *SchedulerBindingMixin) DebugAssertNoTransientCallbacks(reason string) bool {
	if foundation.KDebugMode {
		if m.outer().TransientCallbackCount() > 0 {
			// We cache the values so that we can produce them later
			// even if the information collector is called after
			// the problem has been resolved.
			count := m.outer().TransientCallbackCount()
			callbacks := m.callbacks().clone()
			foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(reason, foundation.FlutterErrorDetailsOptions{
				Library: strp("scheduler library"),
				InformationCollector: func() iter.Seq[foundation.DiagnosticsNode] {
					var nodes []foundation.DiagnosticsNode
					if count == 1 {
						nodes = append(nodes, foundation.NewErrorDescription(
							"There was one transient callback left. "+
								"The stack trace for when it was registered is as follows:"))
					} else {
						nodes = append(nodes, foundation.NewErrorDescription(
							"There were "+strconv.Itoa(count)+" transient callbacks left. "+
								"The stack traces for when they were registered are as follows:"))
					}
					showSeparator := false
					for id, entry := range callbacks.all() {
						nodes = append(nodes, foundation.NewDiagnosticsStackTrace("── callback "+strconv.Itoa(id)+" ──",
							entry.debugStack, foundation.DiagnosticsStackTraceOptions{ShowSeparator: &showSeparator}))
					}
					return slices.Values(nodes)
				},
			}))
		}
	}
	return true
}

func (m *SchedulerBindingMixin) DebugAssertNoPendingPerformanceModeRequests(reason string) bool {
	if foundation.KDebugMode && m.performanceMode != nil {
		panic(foundation.NewFlutterError(reason))
	}
	return true
}

func (m *SchedulerBindingMixin) DebugAssertNoTimeDilation(reason string) bool {
	if foundation.KDebugMode && timeDilation != 1.0 {
		panic(foundation.NewFlutterError(reason))
	}
	return true
}

// SchedulerBindingDebugPrintTransientCallbackRegistrationStack is
// SchedulerBinding.debugPrintTransientCallbackRegistrationStack: in debug
// builds, during a transient callback, it prints the stack where that
// callback was registered; elsewhere it says no transient callback is
// executing.
func SchedulerBindingDebugPrintTransientCallbackRegistrationStack() {
	if foundation.KDebugMode {
		if frameCallbackEntryDebugCurrentCallbackStack != nil {
			foundation.DebugPrint("When the current transient callback was registered, this was the stack:")
			lines := strings.Split(strings.TrimRight(
				foundation.FlutterErrorDemangleStackTrace(frameCallbackEntryDebugCurrentCallbackStack).String(), " \t\n\r"), "\n")
			foundation.DebugPrint(strings.Join(slices.Collect(foundation.FlutterErrorDefaultStackFilter(slices.Values(lines))), "\n"))
		} else {
			foundation.DebugPrint("No transient callback is currently executing.")
		}
	}
}

func (m *SchedulerBindingMixin) AddPersistentFrameCallback(callback FrameCallback) {
	m.persistentCallbacks = append(m.persistentCallbacks, callback)
}

func (m *SchedulerBindingMixin) AddPostFrameCallback(callback FrameCallback, opts ...AddPostFrameCallbackOptions) {
	debugLabel := "callback"
	if len(opts) > 0 && opts[0].DebugLabel != "" {
		debugLabel = opts[0].DebugLabel
	}
	if foundation.KDebugMode && DebugTracePostFrameCallbacks {
		originalCallback := callback
		callback = func(timeStamp time.Duration) {
			developer.TimelineStartSync(debugLabel)
			defer developer.TimelineFinishSync()
			originalCallback(timeStamp)
		}
	}
	m.postFrameCallbacks = append(m.postFrameCallbacks, callback)
}

func (m *SchedulerBindingMixin) EndOfFrame() async.Future[async.Void] {
	self := m.outer()
	if m.nextFrameCompleter == nil {
		if self.SchedulerPhase() == SchedulerPhaseIdle {
			self.ScheduleFrame()
		}
		m.nextFrameCompleter = async.NewCompleter[async.Void]()
		self.AddPostFrameCallback(func(time.Duration) {
			m.nextFrameCompleter.Complete(async.Void{})
			m.nextFrameCompleter = nil
		}, AddPostFrameCallbackOptions{DebugLabel: "SchedulerBinding.completeFrame"})
	}
	return m.nextFrameCompleter.Future()
}

func (m *SchedulerBindingMixin) HasScheduledFrame() bool { return m.hasScheduledFrame }

func (m *SchedulerBindingMixin) SchedulerPhase() SchedulerPhase { return m.schedulerPhase }

func (m *SchedulerBindingMixin) FramesEnabled() bool { return !m.framesDisabled }

func (m *SchedulerBindingMixin) setFramesEnabledState(enabled bool) {
	if !m.framesDisabled == enabled {
		return
	}
	m.framesDisabled = !enabled
	if enabled {
		m.outer().ScheduleFrame()
	}
}

func (m *SchedulerBindingMixin) EnsureFrameCallbacksRegistered() {
	pd := m.outer().PlatformDispatcher()
	if pd.OnBeginFrame() == nil {
		pd.SetOnBeginFrame(m.handleBeginFrameFromEngine)
	}
	if pd.OnDrawFrame() == nil {
		pd.SetOnDrawFrame(m.handleDrawFrameFromEngine)
	}
}

func (m *SchedulerBindingMixin) EnsureVisualUpdate() {
	switch m.outer().SchedulerPhase() {
	case SchedulerPhaseIdle, SchedulerPhasePostFrameCallbacks:
		m.outer().ScheduleFrame()
		return
	case SchedulerPhaseTransientCallbacks, SchedulerPhaseMidFrameMicrotasks, SchedulerPhasePersistentCallbacks:
		return
	}
}

func (m *SchedulerBindingMixin) ScheduleFrame() {
	self := m.outer()
	if m.hasScheduledFrame || !self.FramesEnabled() {
		return
	}
	if foundation.KDebugMode && DebugPrintScheduleFrameStacks {
		foundation.DebugPrintStack(foundation.DebugPrintStackOptions{
			Label: strp("scheduleFrame() called. Current phase is " + self.SchedulerPhase().String() + "."),
		})
	}
	self.EnsureFrameCallbacksRegistered()
	self.PlatformDispatcher().ScheduleFrame()
	m.hasScheduledFrame = true
}

func (m *SchedulerBindingMixin) ScheduleForcedFrame() {
	self := m.outer()
	if m.hasScheduledFrame {
		return
	}
	if foundation.KDebugMode && DebugPrintScheduleFrameStacks {
		foundation.DebugPrintStack(foundation.DebugPrintStackOptions{
			Label: strp("scheduleForcedFrame() called. Current phase is " + self.SchedulerPhase().String() + "."),
		})
	}
	self.EnsureFrameCallbacksRegistered()
	self.PlatformDispatcher().ScheduleFrame()
	m.hasScheduledFrame = true
}

func (m *SchedulerBindingMixin) ScheduleWarmUpFrame() {
	self := m.outer()
	if m.warmUpFrame || self.SchedulerPhase() != SchedulerPhaseIdle {
		return
	}

	m.warmUpFrame = true
	var debugTimelineTask *developer.TimelineTask
	if !foundation.KReleaseMode {
		debugTimelineTask = developer.NewTimelineTask()
		debugTimelineTask.Start("Warm-up frame", nil)
	}
	hadScheduledFrame := m.hasScheduledFrame
	ui.PlatformDispatcherInstance().ScheduleWarmUpFrame(
		func() {
			dartAssert(m.warmUpFrame, nil)
			self.HandleBeginFrame(nil)
		},
		func() {
			dartAssert(m.warmUpFrame, nil)
			self.HandleDrawFrame()
			// We call resetEpoch after this frame so that, in the hot reload case,
			// the very next frame pretends to have occurred immediately after this
			// warm-up frame. The warm-up frame's timestamp will typically be far in
			// the past (the time of the last real frame), so if we didn't reset the
			// epoch we would see a sudden jump from the old time in the warm-up frame
			// to the new time in the "real" frame.
			self.ResetEpoch()
			m.warmUpFrame = false
			if hadScheduledFrame {
				self.ScheduleFrame()
			}
		},
	)

	// Lock events so touch events etc don't insert themselves until the
	// scheduled frame has finished.
	self.LockEvents(func() async.Future[async.Void] {
		return async.AsyncVoid(func() {
			self.EndOfFrame().Await()
			if !foundation.KReleaseMode {
				debugTimelineTask.Finish(nil)
			}
		})
	})
}

func (m *SchedulerBindingMixin) ResetEpoch() {
	m.epochStart = m.adjustForEpoch(m.lastRawTimeStamp)
	m.firstRawTimeStampInEpoch = nil
}

// adjustForEpoch adjusts rawTimeStamp into the current epoch: offset by
// when the epoch started (in raw time and in the epoch's own time line)
// and scaled by the current TimeDilation.
func (m *SchedulerBindingMixin) adjustForEpoch(rawTimeStamp time.Duration) time.Duration {
	var rawDurationSinceEpoch time.Duration
	if m.firstRawTimeStampInEpoch != nil {
		rawDurationSinceEpoch = rawTimeStamp - *m.firstRawTimeStampInEpoch
	}
	// (int / double).round(): round half away from zero, as math.Round.
	scaled := int64(math.Round(float64(rawDurationSinceEpoch.Microseconds()) / timeDilation))
	return time.Duration(scaled+m.epochStart.Microseconds()) * time.Microsecond
}

func (m *SchedulerBindingMixin) CurrentFrameTimeStamp() time.Duration {
	dartAssert(m.currentFrameTimeStamp != nil, nil)
	return *m.currentFrameTimeStamp
}

func (m *SchedulerBindingMixin) CurrentSystemFrameTimeStamp() time.Duration {
	return m.lastRawTimeStamp
}

// handleBeginFrameFromEngine is _handleBeginFrame, the
// PlatformDispatcher.onBeginFrame handler: an engine frame that begins
// during the warm-up frame is postponed until after it.
func (m *SchedulerBindingMixin) handleBeginFrameFromEngine(rawTimeStamp time.Duration) {
	if m.warmUpFrame {
		// "begin frame" and "draw frame" must strictly alternate. Therefore
		// _rescheduleAfterWarmUpFrame cannot possibly be true here as it is
		// reset by _handleDrawFrame.
		dartAssert(!m.rescheduleAfterWarmUpFrame, nil)
		m.rescheduleAfterWarmUpFrame = true
		return
	}
	m.outer().HandleBeginFrame(&rawTimeStamp)
}

// handleDrawFrameFromEngine is _handleDrawFrame, the
// PlatformDispatcher.onDrawFrame handler.
func (m *SchedulerBindingMixin) handleDrawFrameFromEngine() {
	if m.rescheduleAfterWarmUpFrame {
		m.rescheduleAfterWarmUpFrame = false
		// Reschedule in a post-frame callback to allow the draw-frame phase of
		// the warm-up frame to finish.
		m.outer().AddPostFrameCallback(func(time.Duration) {
			// Force an engine frame: the original engine frame was cancelled,
			// so handleBeginFrame did not reset _hasScheduledFrame.
			m.hasScheduledFrame = false
			m.outer().ScheduleFrame()
		}, AddPostFrameCallbackOptions{DebugLabel: "SchedulerBinding.scheduleFrame"})
		return
	}
	m.outer().HandleDrawFrame()
}

// frameTimelineTask is _frameTimelineTask (nil in release builds).
func (m *SchedulerBindingMixin) frameTimelineTask() *developer.TimelineTask {
	if !m.frameTimelineTaskCreated {
		m.frameTimelineTaskCreated = true
		if !foundation.KReleaseMode {
			m.frameTimelineTaskValue = developer.NewTimelineTask()
		}
	}
	return m.frameTimelineTaskValue
}

func (m *SchedulerBindingMixin) HandleBeginFrame(rawTimeStamp *time.Duration) {
	self := m.outer()
	if task := m.frameTimelineTask(); task != nil {
		task.Start("Frame", nil)
	}
	if m.firstRawTimeStampInEpoch == nil && rawTimeStamp != nil {
		first := *rawTimeStamp
		m.firstRawTimeStampInEpoch = &first
	}
	raw := m.lastRawTimeStamp
	if rawTimeStamp != nil {
		raw = *rawTimeStamp
	}
	current := m.adjustForEpoch(raw)
	m.currentFrameTimeStamp = &current
	if rawTimeStamp != nil {
		m.lastRawTimeStamp = *rawTimeStamp
	}

	if foundation.KDebugMode {
		m.debugFrameNumber++

		if DebugPrintBeginFrameBanner || DebugPrintEndFrameBanner {
			var frameTimeStampDescription strings.Builder
			if rawTimeStamp != nil {
				debugDescribeTimeStamp(*m.currentFrameTimeStamp, &frameTimeStampDescription)
			} else {
				frameTimeStampDescription.WriteString("(warm-up frame)")
			}
			banner := "▄▄▄▄▄▄▄▄ Frame " + padRight(strconv.Itoa(m.debugFrameNumber), 7) + "   " +
				padLeft(frameTimeStampDescription.String(), 18) + " ▄▄▄▄▄▄▄▄"
			m.debugBanner = &banner
			if DebugPrintBeginFrameBanner {
				foundation.DebugPrint(banner)
			}
		}
	}

	dartAssert(self.SchedulerPhase() == SchedulerPhaseIdle, nil)
	m.hasScheduledFrame = false
	func() {
		defer func() { m.schedulerPhase = SchedulerPhaseMidFrameMicrotasks }()
		// TRANSIENT FRAME CALLBACKS
		if task := m.frameTimelineTask(); task != nil {
			task.Start("Animate", nil)
		}
		m.schedulerPhase = SchedulerPhaseTransientCallbacks
		callbacks := m.callbacks()
		m.transientCallbacks = newFrameCallbacks()
		for id, callbackEntry := range callbacks.all() {
			if _, removed := m.removedIds[id]; !removed {
				m.invokeFrameCallback(callbackEntry.callback, *m.currentFrameTimeStamp, callbackEntry.debugStack)
			}
		}
		clear(m.removedIds)
	}()
}

func (m *SchedulerBindingMixin) RequestPerformanceMode(mode ui.DartPerformanceMode) *PerformanceModeRequestHandle {
	// conflicting requests are not allowed.
	if m.performanceMode != nil && *m.performanceMode != mode {
		return nil
	}

	if m.performanceMode != nil && *m.performanceMode == mode {
		dartAssert(m.numPerformanceModeRequests > 0, nil)
		m.numPerformanceModeRequests++
	} else if m.performanceMode == nil {
		dartAssert(m.numPerformanceModeRequests == 0, nil)
		m.performanceMode = &mode
		m.numPerformanceModeRequests = 1
	}

	return newPerformanceModeRequestHandle(m.disposePerformanceModeRequest)
}

// disposePerformanceModeRequest removes a request; when none is left, the
// engine reverts to DartPerformanceModeBalanced.
func (m *SchedulerBindingMixin) disposePerformanceModeRequest() {
	m.numPerformanceModeRequests--
	if m.numPerformanceModeRequests == 0 {
		m.performanceMode = nil
		ui.PlatformDispatcherInstance().RequestDartPerformanceMode(ui.DartPerformanceModeBalanced)
	}
}

func (m *SchedulerBindingMixin) DebugGetRequestedPerformanceMode() *ui.DartPerformanceMode {
	if !(foundation.KDebugMode || foundation.KProfileMode) {
		return nil
	}
	return m.performanceMode
}

func (m *SchedulerBindingMixin) HandleDrawFrame() {
	dartAssert(m.schedulerPhase == SchedulerPhaseMidFrameMicrotasks, nil)
	if task := m.frameTimelineTask(); task != nil {
		task.Finish(nil) // end the "Animate" phase
	}
	defer func() {
		m.schedulerPhase = SchedulerPhaseIdle
		if task := m.frameTimelineTask(); task != nil {
			task.Finish(nil) // end the Frame
		}
		if foundation.KDebugMode {
			if DebugPrintEndFrameBanner {
				foundation.DebugPrint(strings.Repeat("▀", len(utf16.Encode([]rune(*m.debugBanner)))))
			}
			m.debugBanner = nil
		}
		m.currentFrameTimeStamp = nil
	}()

	// PERSISTENT FRAME CALLBACKS
	m.schedulerPhase = SchedulerPhasePersistentCallbacks
	for _, callback := range slices.Clone(m.persistentCallbacks) {
		m.invokeFrameCallback(callback, *m.currentFrameTimeStamp, nil)
	}

	// POST-FRAME CALLBACKS
	m.schedulerPhase = SchedulerPhasePostFrameCallbacks
	localPostFrameCallbacks := slices.Clone(m.postFrameCallbacks)
	m.postFrameCallbacks = nil
	if !foundation.KReleaseMode {
		foundation.FlutterTimeline.StartSync("POST_FRAME")
	}
	func() {
		defer func() {
			if !foundation.KReleaseMode {
				foundation.FlutterTimeline.FinishSync()
			}
		}()
		for _, callback := range localPostFrameCallbacks {
			m.invokeFrameCallback(callback, *m.currentFrameTimeStamp, nil)
		}
	}()
}

func (m *SchedulerBindingMixin) profileFramePostEvent(frameTiming ui.FrameTiming) {
	m.outer().PostEvent("Flutter.Frame", map[string]any{
		"number":        frameTiming.FrameNumber(),
		"startTime":     frameTiming.TimestampInMicroseconds(ui.FramePhaseBuildStart),
		"elapsed":       frameTiming.TotalSpan().Microseconds(),
		"build":         frameTiming.BuildDuration().Microseconds(),
		"raster":        frameTiming.RasterDuration().Microseconds(),
		"vsyncOverhead": frameTiming.VsyncOverhead().Microseconds(),
	})
}

// debugDescribeTimeStamp is _debugDescribeTimeStamp, with Dart Duration's
// truncating in* getters.
func debugDescribeTimeStamp(timeStamp time.Duration, buffer *strings.Builder) {
	us := timeStamp.Microseconds()
	inMilliseconds := us / 1000
	inSeconds := us / 1000000
	inMinutes := inSeconds / 60
	inHours := inMinutes / 60
	inDays := inHours / 24
	if inDays > 0 {
		buffer.WriteString(strconv.FormatInt(inDays, 10) + "d ")
	}
	if inHours > 0 {
		buffer.WriteString(strconv.FormatInt(inHours-inDays*24, 10) + "h ")
	}
	if inMinutes > 0 {
		buffer.WriteString(strconv.FormatInt(inMinutes-inHours*60, 10) + "m ")
	}
	if inSeconds > 0 {
		buffer.WriteString(strconv.FormatInt(inSeconds-inMinutes*60, 10) + "s ")
	}
	buffer.WriteString(strconv.FormatInt(inMilliseconds-inSeconds*1000, 10))
	microseconds := us - inMilliseconds*1000
	if microseconds > 0 {
		buffer.WriteString("." + padLeftWith(strconv.FormatInt(microseconds, 10), 3, "0"))
	}
	buffer.WriteString("ms")
}

// invokeFrameCallback calls callback with timeStamp, reporting what it
// panics with to FlutterErrorReportError (with the registration stack,
// when given).
func (m *SchedulerBindingMixin) invokeFrameCallback(callback FrameCallback, timeStamp time.Duration, callbackStack dart.StackTrace) {
	dartAssert(frameCallbackEntryDebugCurrentCallbackStack == nil, nil)
	if foundation.KDebugMode {
		frameCallbackEntryDebugCurrentCallbackStack = callbackStack
	}
	func() {
		defer func() {
			if exception := recover(); exception != nil {
				exceptionStack := dart.StackTraceFromPanic()
				foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
					Stack:                exceptionStack,
					Library:              strp("scheduler library"),
					Context:              foundation.NewErrorDescription("during a scheduler callback"),
					InformationCollector: registrationStackCollector(callbackStack),
				}))
			}
		}()
		callback(timeStamp)
	}()
	if foundation.KDebugMode {
		frameCallbackEntryDebugCurrentCallbackStack = nil
	}
}

// DefaultSchedulingStrategy is the default SchedulingStrategy: while frame
// callbacks are registered, only tasks of PriorityAnimation or higher run;
// otherwise all tasks run.
func DefaultSchedulingStrategy(priority int, scheduler SchedulerBinding) bool {
	if scheduler.TransientCallbackCount() > 0 {
		return priority >= PriorityAnimation.Value()
	}
	return true
}

// ---- helpers

// dartAssert is Dart's assert(condition, message): assertions are on in
// every build (API_MAPPING).
func dartAssert(condition bool, message any) {
	if !condition {
		panic(dart.NewAssertionError(message))
	}
}

func strp(s string) *string { return &s }

// funcIdentity is Dart's identical() on functions: the closure pointer.
func funcIdentity[F any](f F) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&f)) }

// padLeft and padRight are String.padLeft/padRight with spaces (lengths
// in UTF-16 code units).
func padLeft(s string, width int) string { return padLeftWith(s, width, " ") }

func padLeftWith(s string, width int, padding string) string {
	n := len(utf16.Encode([]rune(s)))
	if n >= width {
		return s
	}
	return strings.Repeat(padding, width-n) + s
}

func padRight(s string, width int) string {
	n := len(utf16.Encode([]rune(s)))
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// ---- package:collection's HeapPriorityQueue (collection 1.19.1), the
// parts the task queue uses, with the comparison fixed to taskSorter.

type heapPriorityQueue struct {
	queue  []taskEntry
	length int
}

const heapInitialCapacity = 7

func (q *heapPriorityQueue) isEmpty() bool    { return q.length == 0 }
func (q *heapPriorityQueue) isNotEmpty() bool { return q.length != 0 }

func (q *heapPriorityQueue) add(element taskEntry) {
	if q.length == len(q.queue) {
		q.grow()
	}
	q.bubbleUp(element, q.length)
	q.length++
}

func (q *heapPriorityQueue) first() taskEntry {
	if q.length == 0 {
		panic(dart.StateError{Message: "No element"})
	}
	return q.queue[0]
}

func (q *heapPriorityQueue) removeFirst() taskEntry {
	if q.length == 0 {
		panic(dart.StateError{Message: "No element"})
	}
	result := q.queue[0]
	last := q.removeLast()
	if q.length > 0 {
		q.bubbleDown(last, 0)
	}
	return result
}

func (q *heapPriorityQueue) removeLast() taskEntry {
	newLength := q.length - 1
	last := q.queue[newLength]
	q.queue[newLength] = nil
	q.length = newLength
	return last
}

func (q *heapPriorityQueue) bubbleUp(element taskEntry, index int) {
	for index > 0 {
		parentIndex := (index - 1) / 2
		parent := q.queue[parentIndex]
		if taskSorter(element, parent) > 0 {
			break
		}
		q.queue[index] = parent
		index = parentIndex
	}
	q.queue[index] = element
}

func (q *heapPriorityQueue) bubbleDown(element taskEntry, index int) {
	rightChildIndex := index*2 + 2
	for rightChildIndex < q.length {
		leftChildIndex := rightChildIndex - 1
		leftChild := q.queue[leftChildIndex]
		rightChild := q.queue[rightChildIndex]
		comp := taskSorter(leftChild, rightChild)
		var minChildIndex int
		var minChild taskEntry
		if comp < 0 {
			minChild = leftChild
			minChildIndex = leftChildIndex
		} else {
			minChild = rightChild
			minChildIndex = rightChildIndex
		}
		comp = taskSorter(element, minChild)
		if comp <= 0 {
			q.queue[index] = element
			return
		}
		q.queue[index] = minChild
		index = minChildIndex
		rightChildIndex = index*2 + 2
	}
	leftChildIndex := rightChildIndex - 1
	if leftChildIndex < q.length {
		child := q.queue[leftChildIndex]
		comp := taskSorter(element, child)
		if comp > 0 {
			q.queue[index] = child
			index = leftChildIndex
		}
	}
	q.queue[index] = element
}

func (q *heapPriorityQueue) grow() {
	newCapacity := len(q.queue)*2 + 1
	if newCapacity < heapInitialCapacity {
		newCapacity = heapInitialCapacity
	}
	newQueue := make([]taskEntry, newCapacity)
	copy(newQueue, q.queue[:q.length])
	q.queue = newQueue
}
