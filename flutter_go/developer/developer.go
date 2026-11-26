// Package developer is the subset of dart:developer that Flutter's
// foundation library uses: service extensions (registerExtension,
// postEvent, ServiceExtensionResponse) and the timeline (Timeline,
// TimelineTask, Flow), ported from the Dart SDK that ships with Flutter
// 3.50.1 (lib/developer/extension.dart, timeline.dart).
//
// ENGINE GAP: Dart's versions talk to the VM service and the VM's timeline
// recorder. flutter-go has neither: an extension is registered in this
// package's registry (LookupExtension is the hook a future VM-service
// transport, or a test, uses to invoke it), postEvent behaves as Dart's does
// when no client listens to the Extension stream (it returns), and timeline
// events are dropped, as when the VM's Dart timeline stream is disabled. The
// argument checks and the StateErrors of unbalanced timeline calls are
// Dart's.
package developer

import (
	"strings"
	"time"

	"github.com/jonathanperis/flutter-go/flutter_go/async"
	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

// ---- service extensions (extension.dart) ----

// ServiceExtensionResponse is the response to a service extension call:
// either a JSON result or an error code and detail.
type ServiceExtensionResponse struct {
	Result      *string
	ErrorCode   *int
	ErrorDetail *string
}

// Error codes of ServiceExtensionResponse.error.
const (
	ServiceExtensionResponseInvalidParams     = -32602
	ServiceExtensionResponseExtensionError    = -32000
	ServiceExtensionResponseExtensionErrorMax = -32000
	ServiceExtensionResponseExtensionErrorMin = -32016
)

// NewServiceExtensionResponseResult is ServiceExtensionResponse.result.
func NewServiceExtensionResponseResult(result string) *ServiceExtensionResponse {
	return &ServiceExtensionResponse{Result: &result}
}

// NewServiceExtensionResponseError is ServiceExtensionResponse.error; it
// panics with an ArgumentError for a code outside the allowed ranges.
func NewServiceExtensionResponseError(errorCode int, errorDetail string) *ServiceExtensionResponse {
	validateErrorCode(errorCode)
	return &ServiceExtensionResponse{ErrorCode: &errorCode, ErrorDetail: &errorDetail}
}

func validateErrorCode(errorCode int) {
	if errorCode == ServiceExtensionResponseInvalidParams {
		return
	}
	if errorCode >= ServiceExtensionResponseExtensionErrorMin && errorCode <= ServiceExtensionResponseExtensionErrorMax {
		return
	}
	panic(dart.ArgumentError{Message: "Out of range", Name: "errorCode", Value: errorCode, HasValue: true})
}

// IsError reports whether the response is an error.
func (r *ServiceExtensionResponse) IsError() bool { return r.ErrorCode != nil && r.ErrorDetail != nil }

// ServiceExtensionHandler is the callback of a service extension.
type ServiceExtensionHandler func(method string, parameters map[string]string) async.Future[*ServiceExtensionResponse]

var extensions = map[string]ServiceExtensionHandler{}

// RegisterExtension registers handler for the service extension method,
// which must start with "ext." and not be registered yet (both are
// ArgumentErrors, as in Dart).
func RegisterExtension(method string, handler ServiceExtensionHandler) {
	if !strings.HasPrefix(method, "ext.") {
		panic(dart.ArgumentError{Message: "Must begin with ext.", Name: "method", Value: method, HasValue: true})
	}
	if _, ok := extensions[method]; ok {
		panic(dart.ArgumentError{Message: "Extension already registered: " + method})
	}
	extensions[method] = handler
}

// LookupExtension returns the handler registered for method, or nil. It is
// the VM's _lookupExtension: the VM service calls it to dispatch a request;
// in flutter-go, tests (and a future transport) do.
func LookupExtension(method string) ServiceExtensionHandler { return extensions[method] }

// ExtensionStreamHasListener reports whether a VM-service client listens to
// the Extension stream: never, in flutter-go (ENGINE GAP).
func ExtensionStreamHasListener() bool { return false }

// PostEventOptions are postEvent's optional parameters.
type PostEventOptions struct {
	Stream string // default "Extension"
}

var protectedStreams = []string{"VM", "Isolate", "Debug", "GC", "_Echo", "HeapSnapshot", "Logging", "Timer", "Timeline", "Profiler"}

// PostEvent posts an event of eventKind to the VM service's stream (by
// default "Extension"). With no listener, as always in flutter-go, it only
// checks its arguments.
func PostEvent(eventKind string, eventData map[string]any, opts ...PostEventOptions) {
	stream := "Extension"
	if len(opts) > 0 && opts[0].Stream != "" {
		stream = opts[0].Stream
	}
	for _, p := range protectedStreams {
		if stream == p {
			panic(dart.ArgumentError{Message: "Cannot be a protected stream.", Name: "stream", Value: stream, HasValue: true})
		}
	}
	if strings.HasPrefix(stream, "_") {
		panic(dart.ArgumentError{Message: "Cannot start with an underscore.", Name: "stream", Value: stream, HasValue: true})
	}
	if !ExtensionStreamHasListener() {
		return
	}
}

// ---- timeline (timeline.dart) ----

// Flow is a flow event attached to a synchronous timeline block.
type Flow struct {
	typ int
	ID  int
}

const (
	flowBegin = 9
	flowStep  = 10
	flowEnd   = 11
)

var nextTaskID = 0

func getNextTaskID() int {
	nextTaskID++
	return nextTaskID
}

// FlowBegin is Flow.begin({id}): id may be nil for a fresh one.
func FlowBegin(id *int) *Flow {
	if id == nil {
		n := getNextTaskID()
		id = &n
	}
	return &Flow{typ: flowBegin, ID: *id}
}

// FlowStep is Flow.step(id).
func FlowStep(id int) *Flow { return &Flow{typ: flowStep, ID: id} }

// FlowEnd is Flow.end(id).
func FlowEnd(id int) *Flow { return &Flow{typ: flowEnd, ID: id} }

// TimelineOptions are Timeline.startSync's optional parameters.
type TimelineOptions struct {
	Arguments map[string]any
	Flow      *Flow
}

// isDartStreamEnabled is the VM's _isDartStreamEnabled: false, flutter-go
// has no timeline recorder (ENGINE GAP), so blocks are pushed as null.
func isDartStreamEnabled() bool { return false }

var syncStack []*struct{}

// TimelineStartSync is Timeline.startSync: starts a synchronous block.
func TimelineStartSync(name string, opts ...TimelineOptions) {
	if !isDartStreamEnabled() {
		syncStack = append(syncStack, nil)
		return
	}
}

// TimelineFinishSync is Timeline.finishSync: finishes the last block, and
// panics with a StateError if none is open.
func TimelineFinishSync() {
	if len(syncStack) == 0 {
		panic(dart.StateError{Message: "Uneven calls to startSync and finishSync"})
	}
	syncStack = syncStack[:len(syncStack)-1]
}

// TimelineInstantSync is Timeline.instantSync.
func TimelineInstantSync(name string, arguments map[string]any) {}

// TimelineTimeSync is Timeline.timeSync: function inside a block.
func TimelineTimeSync[T any](name string, function func() T, opts ...TimelineOptions) T {
	TimelineStartSync(name, opts...)
	defer TimelineFinishSync()
	return function()
}

var traceClockStart = time.Now()

// TimelineNow is Timeline.now: the trace clock, in microseconds (monotonic).
func TimelineNow() int { return int(time.Since(traceClockStart).Microseconds()) }

// TimelineTaskOptions are TimelineTask's optional parameters.
type TimelineTaskOptions struct {
	Parent    *TimelineTask
	FilterKey *string
}

// TimelineTask is an asynchronous timeline task.
type TimelineTask struct {
	parent    *TimelineTask
	filterKey *string
	taskID    int
	stack     []*struct{}
}

// NewTimelineTask is TimelineTask({parent, filterKey}).
func NewTimelineTask(opts ...TimelineTaskOptions) *TimelineTask {
	t := &TimelineTask{taskID: getNextTaskID()}
	if len(opts) > 0 {
		t.parent, t.filterKey = opts[0].Parent, opts[0].FilterKey
	}
	return t
}

// NewTimelineTaskWithTaskID is TimelineTask.withTaskId.
func NewTimelineTaskWithTaskID(taskID int, filterKey *string) *TimelineTask {
	return &TimelineTask{taskID: taskID, filterKey: filterKey}
}

// Start begins an operation of the task.
func (t *TimelineTask) Start(name string, arguments map[string]any) {
	if !isDartStreamEnabled() {
		t.stack = append(t.stack, nil)
		return
	}
}

// Instant emits an instant event of the task.
func (t *TimelineTask) Instant(name string, arguments map[string]any) {}

// Finish ends the last operation started, and panics with a StateError if
// none is open.
func (t *TimelineTask) Finish(arguments map[string]any) {
	if len(t.stack) == 0 {
		panic(dart.StateError{Message: "Uneven calls to start and finish"})
	}
	t.stack = t.stack[:len(t.stack)-1]
}

// Pass returns the task id, to continue the task elsewhere; it panics if an
// operation is still open.
func (t *TimelineTask) Pass() int {
	if len(t.stack) > 0 {
		panic(dart.StateError{Message: "You cannot pass a TimelineTask without finishing all started operations"})
	}
	return t.taskID
}
