package gestures

// Port of recognizer.dart (Flutter 3.50.1).
//
// The abstract recognizer classes are interfaces (GestureRecognizer,
// OneSequenceGestureRecognizer, PrimaryPointerGestureRecognizer) plus
// embeddable bases (GestureRecognizerBase, ...) that keep `self`, the
// concrete recognizer, and call every overridable method through it. A
// recognizer type embeds the base of its Dart superclass and calls the
// base's Construct(self, options) when created (Dart's super constructor
// call); its overrides call "super" through the embedded field
// (`r.PrimaryPointerGestureRecognizerBase.AcceptGesture(pointer)`).

import (
	"iter"
	"slices"
	"strings"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// RecognizerCallback is the signature of callbacks invoked through
// InvokeCallback (Dart's generic typedef RecognizerCallback<T>).
type RecognizerCallback[T any] func() T

// DragStartBehavior configures where a drag starts: at the pointer-down
// position, or where the drag gesture was detected.
type DragStartBehavior int

const (
	// DragStartBehaviorDown sets the initial offset at the position where
	// the first down event was detected.
	DragStartBehaviorDown DragStartBehavior = iota
	// DragStartBehaviorStart sets the initial position at the position
	// where the drag start event was detected.
	DragStartBehaviorStart
)

// Name is the Dart enum value's name.
func (b DragStartBehavior) Name() string { return [...]string{"down", "start"}[b] }

func (b DragStartBehavior) String() string { return "DragStartBehavior." + b.Name() }

// MultitouchDragStrategy configures how several pointers drag a scrollable.
type MultitouchDragStrategy int

const (
	// MultitouchDragStrategyLatestPointer: only the latest active pointer
	// is tracked.
	MultitouchDragStrategyLatestPointer MultitouchDragStrategy = iota
	// MultitouchDragStrategyAverageBoundaryPointers: the average of the
	// leading and trailing boundary pointers.
	MultitouchDragStrategyAverageBoundaryPointers
	// MultitouchDragStrategySumAllPointers: the sum of all active
	// pointers' movements.
	MultitouchDragStrategySumAllPointers
)

// Name is the Dart enum value's name.
func (s MultitouchDragStrategy) Name() string {
	return [...]string{"latestPointer", "averageBoundaryPointers", "sumAllPointers"}[s]
}

func (s MultitouchDragStrategy) String() string { return "MultitouchDragStrategy." + s.Name() }

// AllowedButtonsFilter decides whether a recognizer accepts a pointer with
// these buttons.
type AllowedButtonsFilter func(buttons int) bool

// GestureRecognizerOptions are GestureRecognizer's constructor parameters
// (nil AllowedButtonsFilter: accept any buttons; nil SupportedDevices: all
// devices).
type GestureRecognizerOptions struct {
	DebugOwner           any
	SupportedDevices     map[ui.PointerDeviceKind]struct{}
	AllowedButtonsFilter AllowedButtonsFilter
}

// GestureRecognizer is the base of gesture recognizers (an abstract class
// in Dart, DiagnosticableTree and GestureArenaMember).
type GestureRecognizer interface {
	GestureArenaMember
	foundation.DiagnosticableTree

	// AddPointerPanZoom registers a new pointer pan/zoom that might be
	// relevant to this recognizer.
	AddPointerPanZoom(event *PointerPanZoomStartEvent)
	// AddAllowedPointerPanZoom is called for allowed pan/zoom pointers
	// (protected).
	AddAllowedPointerPanZoom(event *PointerPanZoomStartEvent)
	// AddPointer registers a new pointer that might be relevant to this
	// recognizer.
	AddPointer(event *PointerDownEvent)
	// AddAllowedPointer is called for allowed pointers (protected).
	AddAllowedPointer(event *PointerDownEvent)
	// HandleNonAllowedPointer is called for pointers not allowed
	// (protected).
	HandleNonAllowedPointer(event *PointerDownEvent)
	// IsPointerAllowed checks the pointer's device kind and buttons
	// (protected).
	IsPointerAllowed(event *PointerDownEvent) bool
	// HandleNonAllowedPointerPanZoom is called for pan/zoom pointers not
	// allowed (protected).
	HandleNonAllowedPointerPanZoom(event *PointerPanZoomStartEvent)
	// IsPointerPanZoomAllowed checks the pan/zoom pointer's device kind
	// (protected).
	IsPointerPanZoomAllowed(event *PointerPanZoomStartEvent) bool
	// GetKindForPointer is the device kind of a pointer seen by AddPointer
	// (protected).
	GetKindForPointer(pointer int) ui.PointerDeviceKind
	// GetButtonsForPointer is the buttons of a pointer seen by AddPointer
	// (protected).
	GetButtonsForPointer(pointer int) int
	// Dispose releases the recognizer's resources; overrides must call it.
	Dispose()
	// DebugDescription is a short description of the gesture recognized
	// ("tap", "long press"...).
	DebugDescription() string

	gestureRecognizer() *GestureRecognizerBase
}

// GestureRecognizerBase is GestureRecognizer's state and default methods,
// to embed.
type GestureRecognizerBase struct {
	foundation.DiagnosticableTreeMixin

	self GestureRecognizer

	// DebugOwner is the recognizer's owner, for debugging (the widget that
	// created it).
	DebugOwner any
	// GestureSettings are the device's gesture settings (touch slop), or
	// nil.
	GestureSettings *DeviceGestureSettings
	// SupportedDevices is the kinds of devices allowed to interact with
	// the recognizer (nil: all).
	SupportedDevices map[ui.PointerDeviceKind]struct{}
	// AllowedButtonsFilter decides which buttons are allowed.
	AllowedButtonsFilter AllowedButtonsFilter

	pointerToEventData map[int]recognizerEventData
}

type recognizerEventData struct {
	kind    ui.PointerDeviceKind
	buttons int
}

func defaultButtonAcceptBehavior(int) bool { return true }

// Construct is GestureRecognizer's constructor: it points the base at self
// (the recognizer embedding it).
func (r *GestureRecognizerBase) Construct(self GestureRecognizer, opts GestureRecognizerOptions) {
	r.self = self
	r.DiagnosticableTreeMixin.SetSelf(self)
	r.DebugOwner = opts.DebugOwner
	r.SupportedDevices = opts.SupportedDevices
	r.AllowedButtonsFilter = opts.AllowedButtonsFilter
	if r.AllowedButtonsFilter == nil {
		r.AllowedButtonsFilter = defaultButtonAcceptBehavior
	}
	r.pointerToEventData = map[int]recognizerEventData{}
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchCreated("gestures", "GestureRecognizer", self)
	}
}

func (r *GestureRecognizerBase) gestureRecognizer() *GestureRecognizerBase { return r }

func (r *GestureRecognizerBase) outer() GestureRecognizer {
	if r.self == nil {
		panic("gestures: recognizer used before Construct; call Construct with the recognizer when creating it")
	}
	return r.self
}

func (r *GestureRecognizerBase) AddPointerPanZoom(event *PointerPanZoomStartEvent) {
	r.pointerToEventData[event.Pointer()] = recognizerEventData{kind: event.Kind(), buttons: event.Buttons()}
	if r.outer().IsPointerPanZoomAllowed(event) {
		r.outer().AddAllowedPointerPanZoom(event)
	} else {
		r.outer().HandleNonAllowedPointerPanZoom(event)
	}
}

func (r *GestureRecognizerBase) AddAllowedPointerPanZoom(event *PointerPanZoomStartEvent) {}

func (r *GestureRecognizerBase) AddPointer(event *PointerDownEvent) {
	r.pointerToEventData[event.Pointer()] = recognizerEventData{kind: event.Kind(), buttons: event.Buttons()}
	if r.outer().IsPointerAllowed(event) {
		r.outer().AddAllowedPointer(event)
	} else {
		r.outer().HandleNonAllowedPointer(event)
	}
}

func (r *GestureRecognizerBase) AddAllowedPointer(event *PointerDownEvent) {}

func (r *GestureRecognizerBase) HandleNonAllowedPointer(event *PointerDownEvent) {}

func (r *GestureRecognizerBase) IsPointerAllowed(event *PointerDownEvent) bool {
	return r.supportsDevice(event.Kind()) && r.AllowedButtonsFilter(event.Buttons())
}

func (r *GestureRecognizerBase) supportsDevice(kind ui.PointerDeviceKind) bool {
	if r.SupportedDevices == nil {
		return true
	}
	_, ok := r.SupportedDevices[kind]
	return ok
}

func (r *GestureRecognizerBase) HandleNonAllowedPointerPanZoom(event *PointerPanZoomStartEvent) {}

func (r *GestureRecognizerBase) IsPointerPanZoomAllowed(event *PointerPanZoomStartEvent) bool {
	return r.supportsDevice(event.Kind())
}

func (r *GestureRecognizerBase) GetKindForPointer(pointer int) ui.PointerDeviceKind {
	d, ok := r.pointerToEventData[pointer]
	dartAssert(ok, nil)
	return d.kind
}

func (r *GestureRecognizerBase) GetButtonsForPointer(pointer int) int {
	d, ok := r.pointerToEventData[pointer]
	dartAssert(ok, nil)
	return d.buttons
}

func (r *GestureRecognizerBase) Dispose() {
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchDisposed(r.outer())
	}
}

// InvokeCallback is GestureRecognizer.invokeCallback<T> (a generic method
// in Dart): it calls callback, reporting what it throws to
// FlutterError.reportError, and returns its result (T's zero value, Dart's
// null, when it threw). debugReport, if not nil, adds to the
// debugPrintRecognizerCallbacksTrace line.
func InvokeCallback[T any](r GestureRecognizer, name string, callback RecognizerCallback[T], debugReport ...func() string) (result T) {
	defer func() {
		if exception := recover(); exception != nil {
			stack := dart.StackTraceFromPanic()
			var collector foundation.InformationCollector
			if foundation.KDebugMode {
				collector = func() iter.Seq[foundation.DiagnosticsNode] {
					style := foundation.DiagnosticsTreeStyleErrorProperty
					return slices.Values([]foundation.DiagnosticsNode{
						foundation.NewStringProperty("Handler", name),
						foundation.NewDiagnosticsProperty[GestureRecognizer]("Recognizer", r,
							foundation.DiagnosticsPropertyOptions{Style: &style}),
					})
				}
			}
			library := "gesture"
			foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
				Stack:                stack,
				Library:              &library,
				Context:              foundation.NewErrorDescription("while handling a gesture"),
				InformationCollector: collector,
			}))
			// result keeps T's zero value (Dart's null).
		}
	}()
	if foundation.KDebugMode && DebugPrintRecognizerCallbacksTrace {
		var report string
		hasReport := false
		if len(debugReport) > 0 && debugReport[0] != nil {
			report = debugReport[0]()
			hasReport = true
		}
		// The 19 in the line below is the width of the prefix used by
		// _debugLogDiagnostic in arena.dart.
		prefix := ""
		if DebugPrintGestureArenaDiagnostics {
			prefix = strings.Repeat(" ", 19) + "❙ "
		}
		suffix := ""
		if hasReport && report != "" {
			suffix = " " + report
		}
		foundation.DebugPrint(prefix + dart.ToString(r) + " calling " + name + " callback." + suffix)
	}
	return callback()
}

// InvokeVoidCallback is InvokeCallback for a callback returning nothing
// (Dart's invokeCallback<void>).
func InvokeVoidCallback(r GestureRecognizer, name string, callback func(), debugReport ...func() string) {
	InvokeCallback(r, name, func() struct{} { callback(); return struct{}{} }, debugReport...)
}

func (r *GestureRecognizerBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	r.DiagnosticableTreeMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty[any]("debugOwner", r.DebugOwner,
		foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
}

// ---- OneSequenceGestureRecognizer

// OneSequenceGestureRecognizer is a recognizer that tracks one gesture
// (sequence of pointers) at a time.
type OneSequenceGestureRecognizer interface {
	GestureRecognizer
	// HandleEvent is called when a pointer event is routed to the
	// recognizer (protected).
	HandleEvent(event PointerEvent)
	// DidStopTrackingLastPointer is called when the number of pointers
	// tracked drops to zero (protected).
	DidStopTrackingLastPointer(pointer int)
	// Resolve resolves this recognizer's participation in every arena it
	// is in (protected).
	Resolve(disposition GestureDisposition)
	// ResolvePointer resolves its participation in one pointer's arena
	// (protected).
	ResolvePointer(pointer int, disposition GestureDisposition)
	// StartTrackingPointer routes the pointer's events to HandleEvent and
	// adds the recognizer to its arena (protected).
	StartTrackingPointer(pointer int, transform ...*vm.Matrix4)
	// StopTrackingPointer stops routing the pointer's events (protected).
	StopTrackingPointer(pointer int)
	// StopTrackingIfPointerNoLongerDown stops tracking the event's pointer
	// on an up, cancel or pan/zoom end event (protected).
	StopTrackingIfPointerNoLongerDown(event PointerEvent)
	// Team is the team this recognizer competes in, or nil.
	Team() *GestureArenaTeam
	// SetTeam sets Team (once, before any pointer is tracked).
	SetTeam(value *GestureArenaTeam)

	oneSequence() *OneSequenceGestureRecognizerBase
}

// OneSequenceGestureRecognizerBase is OneSequenceGestureRecognizer's state
// and default methods, to embed.
type OneSequenceGestureRecognizerBase struct {
	GestureRecognizerBase

	self OneSequenceGestureRecognizer

	// entries is the Map<int, GestureArenaEntry> (insertion order).
	entryPointers   []int
	entries         map[int]GestureArenaEntry
	trackedPointers map[int]struct{}
	trackedOrder    []int
	team            *GestureArenaTeam

	// handleEventRoute is self.HandleEvent as one func value, so the
	// pointer router can remove the route it added (Dart's tear-offs are
	// equal).
	handleEventRoute PointerRoute
}

// Construct is OneSequenceGestureRecognizer's constructor.
func (r *OneSequenceGestureRecognizerBase) Construct(self OneSequenceGestureRecognizer, opts GestureRecognizerOptions) {
	r.GestureRecognizerBase.Construct(self, opts)
	r.self = self
	r.entries = map[int]GestureArenaEntry{}
	r.trackedPointers = map[int]struct{}{}
	r.handleEventRoute = func(event PointerEvent) { r.self.HandleEvent(event) }
}

func (r *OneSequenceGestureRecognizerBase) oneSequence() *OneSequenceGestureRecognizerBase { return r }

func (r *OneSequenceGestureRecognizerBase) AddAllowedPointer(event *PointerDownEvent) {
	r.self.StartTrackingPointer(event.Pointer(), event.Transform())
}

func (r *OneSequenceGestureRecognizerBase) HandleNonAllowedPointer(event *PointerDownEvent) {
	r.self.Resolve(GestureDispositionRejected)
}

func (r *OneSequenceGestureRecognizerBase) AcceptGesture(pointer int) {}

func (r *OneSequenceGestureRecognizerBase) RejectGesture(pointer int) {}

func (r *OneSequenceGestureRecognizerBase) Resolve(disposition GestureDisposition) {
	localEntries := make([]GestureArenaEntry, 0, len(r.entryPointers))
	for _, p := range r.entryPointers {
		localEntries = append(localEntries, r.entries[p])
	}
	r.entries = map[int]GestureArenaEntry{}
	r.entryPointers = nil
	for _, entry := range localEntries {
		entry.Resolve(disposition)
	}
}

func (r *OneSequenceGestureRecognizerBase) ResolvePointer(pointer int, disposition GestureDisposition) {
	if entry, ok := r.entries[pointer]; ok {
		r.removeEntry(pointer)
		entry.Resolve(disposition)
	}
}

func (r *OneSequenceGestureRecognizerBase) removeEntry(pointer int) {
	delete(r.entries, pointer)
	r.entryPointers = slices.DeleteFunc(r.entryPointers, func(p int) bool { return p == pointer })
}

func (r *OneSequenceGestureRecognizerBase) Dispose() {
	r.self.Resolve(GestureDispositionRejected)
	for _, pointer := range r.trackedOrder {
		GestureBindingInstance().PointerRouter().RemoveRoute(pointer, r.handleEventRoute)
	}
	r.trackedPointers = map[int]struct{}{}
	r.trackedOrder = nil
	dartAssert(len(r.entries) == 0, nil)
	r.GestureRecognizerBase.Dispose()
}

func (r *OneSequenceGestureRecognizerBase) Team() *GestureArenaTeam { return r.team }

func (r *OneSequenceGestureRecognizerBase) SetTeam(value *GestureArenaTeam) {
	dartAssert(value != nil, nil)
	dartAssert(len(r.entries) == 0, nil)
	dartAssert(len(r.trackedPointers) == 0, nil)
	dartAssert(r.team == nil, nil)
	r.team = value
}

func (r *OneSequenceGestureRecognizerBase) addPointerToArena(pointer int) GestureArenaEntry {
	if r.team != nil {
		return r.team.Add(pointer, r.self)
	}
	return GestureBindingInstance().GestureArena().Add(pointer, r.self)
}

func (r *OneSequenceGestureRecognizerBase) StartTrackingPointer(pointer int, transform ...*vm.Matrix4) {
	var t *vm.Matrix4
	if len(transform) > 0 {
		t = transform[0]
	}
	GestureBindingInstance().PointerRouter().AddRoute(pointer, r.handleEventRoute, t)
	if _, ok := r.trackedPointers[pointer]; !ok {
		r.trackedPointers[pointer] = struct{}{}
		r.trackedOrder = append(r.trackedOrder, pointer)
	}
	entry := r.addPointerToArena(pointer)
	if _, ok := r.entries[pointer]; !ok {
		r.entryPointers = append(r.entryPointers, pointer)
	}
	r.entries[pointer] = entry
}

func (r *OneSequenceGestureRecognizerBase) StopTrackingPointer(pointer int) {
	if _, ok := r.trackedPointers[pointer]; ok {
		GestureBindingInstance().PointerRouter().RemoveRoute(pointer, r.handleEventRoute)
		delete(r.trackedPointers, pointer)
		r.trackedOrder = slices.DeleteFunc(r.trackedOrder, func(p int) bool { return p == pointer })
		if len(r.trackedPointers) == 0 {
			r.self.DidStopTrackingLastPointer(pointer)
		}
	}
}

func (r *OneSequenceGestureRecognizerBase) StopTrackingIfPointerNoLongerDown(event PointerEvent) {
	switch event.(type) {
	case *PointerUpEvent, *PointerCancelEvent, *PointerPanZoomEndEvent:
		r.self.StopTrackingPointer(event.Pointer())
	}
}

// ---- PrimaryPointerGestureRecognizer

// GestureRecognizerState is the possible states of a
// PrimaryPointerGestureRecognizer.
type GestureRecognizerState int

const (
	// GestureRecognizerStateReady: the recognizer is ready to start
	// recognizing a gesture.
	GestureRecognizerStateReady GestureRecognizerState = iota
	// GestureRecognizerStatePossible: the sequence of pointer events seen
	// thus far is consistent with the gesture.
	GestureRecognizerStatePossible
	// GestureRecognizerStateDefunct: further pointer events cannot cause
	// this recognizer to recognize the gesture until it is reset.
	GestureRecognizerStateDefunct
)

// Name is the Dart enum value's name.
func (s GestureRecognizerState) Name() string {
	return [...]string{"ready", "possible", "defunct"}[s]
}

func (s GestureRecognizerState) String() string { return "GestureRecognizerState." + s.Name() }

// PrimaryPointerGestureRecognizerOptions are
// PrimaryPointerGestureRecognizer's constructor parameters. A nil slop
// tolerance is Dart's default (the device's touch slop, kTouchSlop);
// NoPreAcceptSlopTolerance / NoPostAcceptSlopTolerance pass Dart's
// explicit null (no tolerance).
type PrimaryPointerGestureRecognizerOptions struct {
	GestureRecognizerOptions
	Deadline                  *time.Duration
	PreAcceptSlopTolerance    *float64
	NoPreAcceptSlopTolerance  bool
	PostAcceptSlopTolerance   *float64
	NoPostAcceptSlopTolerance bool
}

// PrimaryPointerGestureRecognizer is a recognizer that tracks a single
// primary pointer (taps, long presses).
type PrimaryPointerGestureRecognizer interface {
	OneSequenceGestureRecognizer
	// HandlePrimaryPointer is called for events of the primary pointer
	// (protected, abstract).
	HandlePrimaryPointer(event PointerEvent)
	// DidExceedDeadline is called when the deadline passes (protected).
	DidExceedDeadline()
	// DidExceedDeadlineWithEvent is called when the deadline passes, with
	// the down event (protected).
	DidExceedDeadlineWithEvent(event *PointerDownEvent)
	// State is the recognizer's state.
	State() GestureRecognizerState
	// PrimaryPointer is the primary pointer, or nil.
	PrimaryPointer() *int
	// InitialPosition is the primary pointer's down position, or nil.
	InitialPosition() *OffsetPair
	// PreAcceptSlopTolerance is the slop before acceptance (nil: none).
	PreAcceptSlopTolerance() *float64
	// PostAcceptSlopTolerance is the slop after acceptance (nil: none).
	PostAcceptSlopTolerance() *float64
}

const unsetTouchSlop = -1.0

// PrimaryPointerGestureRecognizerBase is PrimaryPointerGestureRecognizer's
// state and default methods, to embed.
type PrimaryPointerGestureRecognizerBase struct {
	OneSequenceGestureRecognizerBase

	self PrimaryPointerGestureRecognizer

	// Deadline, if not nil, is when DidExceedDeadline is called after the
	// primary pointer goes down.
	Deadline *time.Duration

	preAcceptSlopTolerance  *float64
	postAcceptSlopTolerance *float64

	state           GestureRecognizerState
	primaryPointer  *int
	initialPosition *OffsetPair
	gestureAccepted bool
	timer           *async.Timer
}

func slopOption(value *float64, null bool, name string) *float64 {
	if null {
		return nil
	}
	if value == nil {
		v := unsetTouchSlop
		return &v
	}
	dartAssert(*value >= 0, "The "+name+" must be unspecified, positive, or null")
	return value
}

// Construct is PrimaryPointerGestureRecognizer's constructor.
func (r *PrimaryPointerGestureRecognizerBase) Construct(self PrimaryPointerGestureRecognizer, opts PrimaryPointerGestureRecognizerOptions) {
	r.preAcceptSlopTolerance = slopOption(opts.PreAcceptSlopTolerance, opts.NoPreAcceptSlopTolerance, "preAcceptSlopTolerance")
	r.postAcceptSlopTolerance = slopOption(opts.PostAcceptSlopTolerance, opts.NoPostAcceptSlopTolerance, "postAcceptSlopTolerance")
	r.OneSequenceGestureRecognizerBase.Construct(self, opts.GestureRecognizerOptions)
	r.self = self
	r.Deadline = opts.Deadline
}

func (r *PrimaryPointerGestureRecognizerBase) defaultTouchSlop() float64 {
	if r.GestureSettings != nil && r.GestureSettings.TouchSlop != nil {
		return *r.GestureSettings.TouchSlop
	}
	return KTouchSlop
}

func (r *PrimaryPointerGestureRecognizerBase) resolveSlop(v *float64) *float64 {
	if v != nil && *v == unsetTouchSlop {
		d := r.defaultTouchSlop()
		return &d
	}
	return v
}

func (r *PrimaryPointerGestureRecognizerBase) PreAcceptSlopTolerance() *float64 {
	return r.resolveSlop(r.preAcceptSlopTolerance)
}

func (r *PrimaryPointerGestureRecognizerBase) PostAcceptSlopTolerance() *float64 {
	return r.resolveSlop(r.postAcceptSlopTolerance)
}

func (r *PrimaryPointerGestureRecognizerBase) State() GestureRecognizerState { return r.state }

func (r *PrimaryPointerGestureRecognizerBase) PrimaryPointer() *int { return r.primaryPointer }

func (r *PrimaryPointerGestureRecognizerBase) InitialPosition() *OffsetPair {
	return r.initialPosition
}

func (r *PrimaryPointerGestureRecognizerBase) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizerBase.AddAllowedPointer(event)
	if r.self.State() == GestureRecognizerStateReady {
		r.state = GestureRecognizerStatePossible
		pointer := event.Pointer()
		r.primaryPointer = &pointer
		r.initialPosition = &OffsetPair{Local: event.LocalPosition(), Global: event.Position()}
		if r.Deadline != nil {
			r.timer = async.NewTimer(*r.Deadline, func() { r.self.DidExceedDeadlineWithEvent(event) })
		}
	}
}

func (r *PrimaryPointerGestureRecognizerBase) HandleNonAllowedPointer(event *PointerDownEvent) {
	if !r.gestureAccepted {
		r.OneSequenceGestureRecognizerBase.HandleNonAllowedPointer(event)
	}
}

func (r *PrimaryPointerGestureRecognizerBase) isPrimary(pointer int) bool {
	p := r.self.PrimaryPointer()
	return p != nil && *p == pointer
}

func (r *PrimaryPointerGestureRecognizerBase) HandleEvent(event PointerEvent) {
	dartAssert(r.self.State() != GestureRecognizerStateReady, nil)
	if r.self.State() == GestureRecognizerStatePossible && r.isPrimary(event.Pointer()) {
		pre := r.self.PreAcceptSlopTolerance()
		post := r.self.PostAcceptSlopTolerance()
		isPreAcceptSlopPastTolerance := !r.gestureAccepted && pre != nil && r.getGlobalDistance(event) > *pre
		isPostAcceptSlopPastTolerance := r.gestureAccepted && post != nil && r.getGlobalDistance(event) > *post
		if _, isMove := event.(*PointerMoveEvent); isMove && (isPreAcceptSlopPastTolerance || isPostAcceptSlopPastTolerance) {
			r.self.Resolve(GestureDispositionRejected)
			r.self.StopTrackingPointer(*r.self.PrimaryPointer())
		} else {
			r.self.HandlePrimaryPointer(event)
		}
	}
	r.self.StopTrackingIfPointerNoLongerDown(event)
}

func (r *PrimaryPointerGestureRecognizerBase) DidExceedDeadline() {
	dartAssert(r.Deadline == nil, nil)
}

func (r *PrimaryPointerGestureRecognizerBase) DidExceedDeadlineWithEvent(event *PointerDownEvent) {
	r.self.DidExceedDeadline()
}

func (r *PrimaryPointerGestureRecognizerBase) AcceptGesture(pointer int) {
	if r.isPrimary(pointer) {
		r.stopTimer()
		r.gestureAccepted = true
	}
}

func (r *PrimaryPointerGestureRecognizerBase) RejectGesture(pointer int) {
	if r.isPrimary(pointer) && r.self.State() == GestureRecognizerStatePossible {
		r.stopTimer()
		r.state = GestureRecognizerStateDefunct
	}
}

func (r *PrimaryPointerGestureRecognizerBase) DidStopTrackingLastPointer(pointer int) {
	dartAssert(r.self.State() != GestureRecognizerStateReady, nil)
	r.stopTimer()
	r.state = GestureRecognizerStateReady
	r.initialPosition = nil
	r.gestureAccepted = false
}

func (r *PrimaryPointerGestureRecognizerBase) Dispose() {
	r.stopTimer()
	r.OneSequenceGestureRecognizerBase.Dispose()
}

func (r *PrimaryPointerGestureRecognizerBase) stopTimer() {
	if r.timer != nil {
		r.timer.Cancel()
		r.timer = nil
	}
}

func (r *PrimaryPointerGestureRecognizerBase) getGlobalDistance(event PointerEvent) float64 {
	offset := event.Position().Sub(r.self.InitialPosition().Global)
	return offset.Distance()
}

func (r *PrimaryPointerGestureRecognizerBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	r.OneSequenceGestureRecognizerBase.DebugFillProperties(properties)
	properties.Add(foundation.NewEnumProperty("state", r.self.State()))
}

// ---- OffsetPair

// OffsetPair is a container for a local and a global Offset.
type OffsetPair struct {
	// Local is the offset in the local coordinate space.
	Local ui.Offset
	// Global is the offset in the global coordinate space, after applying
	// all transforms.
	Global ui.Offset
}

// OffsetPairZero is OffsetPair.zero.
var OffsetPairZero = OffsetPair{}

// NewOffsetPairFromEventPosition is OffsetPair.fromEventPosition.
func NewOffsetPairFromEventPosition(event PointerEvent) OffsetPair {
	return OffsetPair{Local: event.LocalPosition(), Global: event.Position()}
}

// NewOffsetPairFromEventDelta is OffsetPair.fromEventDelta.
func NewOffsetPairFromEventDelta(event PointerEvent) OffsetPair {
	return OffsetPair{Local: event.LocalDelta(), Global: event.Delta()}
}

// Add is `+`.
func (p OffsetPair) Add(other OffsetPair) OffsetPair {
	return OffsetPair{Local: p.Local.Add(other.Local), Global: p.Global.Add(other.Global)}
}

// Sub is `-`.
func (p OffsetPair) Sub(other OffsetPair) OffsetPair {
	return OffsetPair{Local: p.Local.Sub(other.Local), Global: p.Global.Sub(other.Global)}
}

func (p OffsetPair) String() string {
	return "OffsetPair(local: " + p.Local.String() + ", global: " + p.Global.String() + ")"
}
