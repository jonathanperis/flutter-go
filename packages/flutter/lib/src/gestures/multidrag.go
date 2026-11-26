package gestures

// Port of multidrag.dart (Flutter 3.50.1): MultiDragPointerState,
// MultiDragGestureRecognizer and the immediate, horizontal, vertical and
// delayed multi-drag recognizers.
//
// MultiDragPointerState (abstract) is an interface plus the embeddable
// MultiDragPointerStateBase (self pattern: Construct(self, ...));
// MultiDragGestureRecognizer is an interface plus
// MultiDragGestureRecognizerBase, whose subclasses implement
// CreateNewPointerState.

import (
	"math"
	"slices"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
)

// GestureMultiDragStartCallback is the signature for when a
// MultiDragGestureRecognizer recognizes the start of a drag gesture; it
// returns the Drag receiving the updates, or nil (Dart's null) to ignore
// the drag.
type GestureMultiDragStartCallback func(position ui.Offset) Drag

// MultiDragPointerState is per-pointer state for a
// MultiDragGestureRecognizer.
type MultiDragPointerState interface {
	// Resolve resolves this pointer's arena entry (protected).
	Resolve(disposition GestureDisposition)
	// CheckForResolutionAfterMove is called when the pending delta changed
	// (protected).
	CheckForResolutionAfterMove()
	// Accepted is called when the gesture was accepted (protected,
	// abstract).
	Accepted(starter GestureMultiDragStartCallback)
	// Rejected is called when the gesture was rejected (protected).
	Rejected()
	// Dispose releases the state (protected).
	Dispose()

	multiDragPointerState() *MultiDragPointerStateBase
}

// MultiDragPointerStateBase is MultiDragPointerState's state and default
// methods, to embed.
type MultiDragPointerStateBase struct {
	self MultiDragPointerState

	// GestureSettings are the device's gesture settings, or nil.
	GestureSettings *DeviceGestureSettings
	// InitialPosition is the global coordinates of the pointer when it
	// first contacted the screen.
	InitialPosition ui.Offset
	// Kind is the kind of pointer performing the multi-drag gesture.
	Kind ui.PointerDeviceKind

	velocityTracker           *VelocityTracker
	client                    Drag
	pendingDelta              *ui.Offset
	lastPendingEventTimestamp *time.Duration
	arenaEntry                GestureArenaEntry
}

// Construct is MultiDragPointerState's constructor.
func (s *MultiDragPointerStateBase) Construct(self MultiDragPointerState, initialPosition ui.Offset, kind ui.PointerDeviceKind, gestureSettings *DeviceGestureSettings) {
	s.self = self
	s.InitialPosition = initialPosition
	s.Kind = kind
	s.GestureSettings = gestureSettings
	s.velocityTracker = NewVelocityTrackerWithKind(kind)
	zero := ui.Offset{}
	s.pendingDelta = &zero
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchCreated("gestures", "MultiDragPointerState", self)
	}
}

func (s *MultiDragPointerStateBase) multiDragPointerState() *MultiDragPointerStateBase { return s }

// PendingDelta is the offset of the pointer since it contacted the screen
// that hasn't been given to a Drag yet (nil once accepted or rejected).
func (s *MultiDragPointerStateBase) PendingDelta() *ui.Offset { return s.pendingDelta }

func (s *MultiDragPointerStateBase) setArenaEntry(entry GestureArenaEntry) {
	dartAssert(s.arenaEntry == nil, nil)
	dartAssert(s.pendingDelta != nil, nil)
	dartAssert(s.client == nil, nil)
	s.arenaEntry = entry
}

func (s *MultiDragPointerStateBase) Resolve(disposition GestureDisposition) {
	s.arenaEntry.Resolve(disposition)
}

func (s *MultiDragPointerStateBase) move(event *PointerMoveEvent) {
	dartAssert(s.arenaEntry != nil, nil)
	if !event.Synthesized() {
		s.velocityTracker.AddPosition(event.TimeStamp(), event.Position())
	}
	if s.client != nil {
		dartAssert(s.pendingDelta == nil, nil)
		ts := event.TimeStamp()
		// Call client last to avoid reentrancy.
		s.client.Update(NewDragUpdateDetails(DragUpdateDetailsOptions{
			SourceTimeStamp: &ts, Delta: event.Delta(), GlobalPosition: event.Position(),
		}))
	} else {
		dartAssert(s.pendingDelta != nil, nil)
		d := s.pendingDelta.Add(event.Delta())
		s.pendingDelta = &d
		ts := event.TimeStamp()
		s.lastPendingEventTimestamp = &ts
		s.self.CheckForResolutionAfterMove()
	}
}

func (s *MultiDragPointerStateBase) CheckForResolutionAfterMove() {}

func (s *MultiDragPointerStateBase) Rejected() {
	dartAssert(s.arenaEntry != nil, nil)
	dartAssert(s.client == nil, nil)
	dartAssert(s.pendingDelta != nil, nil)
	s.pendingDelta = nil
	s.lastPendingEventTimestamp = nil
	s.arenaEntry = nil
}

func (s *MultiDragPointerStateBase) startDrag(client Drag) {
	dartAssert(s.arenaEntry != nil, nil)
	dartAssert(s.client == nil, nil)
	dartAssert(s.pendingDelta != nil, nil)
	s.client = client
	details := NewDragUpdateDetails(DragUpdateDetailsOptions{
		SourceTimeStamp: s.lastPendingEventTimestamp, Delta: *s.pendingDelta, GlobalPosition: s.InitialPosition,
	})
	s.pendingDelta = nil
	s.lastPendingEventTimestamp = nil
	// Call client last to avoid reentrancy.
	s.client.Update(details)
}

func (s *MultiDragPointerStateBase) up() {
	dartAssert(s.arenaEntry != nil, nil)
	if s.client != nil {
		dartAssert(s.pendingDelta == nil, nil)
		details := NewDragEndDetails(DragEndDetailsOptions{Velocity: s.velocityTracker.GetVelocity()})
		client := s.client
		s.client = nil
		// Call client last to avoid reentrancy.
		client.End(details)
	} else {
		dartAssert(s.pendingDelta != nil, nil)
		s.pendingDelta = nil
		s.lastPendingEventTimestamp = nil
	}
}

func (s *MultiDragPointerStateBase) cancel() {
	dartAssert(s.arenaEntry != nil, nil)
	if s.client != nil {
		dartAssert(s.pendingDelta == nil, nil)
		client := s.client
		s.client = nil
		// Call client last to avoid reentrancy.
		client.Cancel()
	} else {
		dartAssert(s.pendingDelta != nil, nil)
		s.pendingDelta = nil
		s.lastPendingEventTimestamp = nil
	}
}

func (s *MultiDragPointerStateBase) Dispose() {
	if foundation.KDebugMode {
		foundation.DebugMaybeDispatchDisposed(s.self)
	}
	if s.arenaEntry != nil {
		s.arenaEntry.Resolve(GestureDispositionRejected)
	}
	s.arenaEntry = nil
	if foundation.KDebugMode {
		s.pendingDelta = nil
	}
}

// MultiDragGestureRecognizer recognizes movement on a per-pointer basis.
type MultiDragGestureRecognizer interface {
	GestureRecognizer
	// CreateNewPointerState creates the state of a new pointer (protected,
	// abstract).
	CreateNewPointerState(event *PointerDownEvent) MultiDragPointerState

	multiDragGestureRecognizer() *MultiDragGestureRecognizerBase
}

// MultiDragGestureRecognizerBase is MultiDragGestureRecognizer's state and
// methods, to embed.
type MultiDragGestureRecognizerBase struct {
	GestureRecognizerBase

	self MultiDragGestureRecognizer

	// OnStart is called when this class recognizes the start of a drag
	// gesture.
	OnStart GestureMultiDragStartCallback

	pointers         map[int]MultiDragPointerState // nil once disposed
	pointerOrder     []int
	handleEventRoute PointerRoute
}

// Construct is MultiDragGestureRecognizer's constructor (nil
// AllowedButtonsFilter: the primary button only).
func (r *MultiDragGestureRecognizerBase) Construct(self MultiDragGestureRecognizer, opts GestureRecognizerOptions) {
	if opts.AllowedButtonsFilter == nil {
		opts.AllowedButtonsFilter = dragDefaultButtonAcceptBehavior
	}
	r.GestureRecognizerBase.Construct(self, opts)
	r.self = self
	r.pointers = map[int]MultiDragPointerState{}
	r.handleEventRoute = r.handleEvent
}

func (r *MultiDragGestureRecognizerBase) multiDragGestureRecognizer() *MultiDragGestureRecognizerBase {
	return r
}

func (r *MultiDragGestureRecognizerBase) AddAllowedPointer(event *PointerDownEvent) {
	dartAssert(r.pointers != nil, nil)
	_, exists := r.pointers[event.Pointer()]
	dartAssert(!exists, nil)
	state := r.self.CreateNewPointerState(event)
	r.pointers[event.Pointer()] = state
	r.pointerOrder = append(r.pointerOrder, event.Pointer())
	GestureBindingInstance().PointerRouter().AddRoute(event.Pointer(), r.handleEventRoute)
	state.multiDragPointerState().setArenaEntry(GestureBindingInstance().GestureArena().Add(event.Pointer(), r.self))
}

func (r *MultiDragGestureRecognizerBase) handleEvent(event PointerEvent) {
	dartAssert(r.pointers != nil, nil)
	state, ok := r.pointers[event.Pointer()]
	dartAssert(ok, nil)
	base := state.multiDragPointerState()
	switch e := event.(type) {
	case *PointerMoveEvent:
		base.move(e)
		// We might be disposed here.
	case *PointerUpEvent:
		dartAssert(event.Delta() == (ui.Offset{}), nil)
		base.up()
		// We might be disposed here.
		r.removeState(event.Pointer())
	case *PointerCancelEvent:
		dartAssert(event.Delta() == (ui.Offset{}), nil)
		base.cancel()
		// We might be disposed here.
		r.removeState(event.Pointer())
	case *PointerDownEvent:
		// We get the PointerDownEvent that resulted in our addPointer
		// getting called since we add ourselves to the pointer router then
		// (before the pointer router has heard of the event).
	default:
		dartAssert(false, nil)
	}
}

func (r *MultiDragGestureRecognizerBase) AcceptGesture(pointer int) {
	dartAssert(r.pointers != nil, nil)
	state, ok := r.pointers[pointer]
	if !ok {
		return // We might already have canceled this drag if the up comes before the accept.
	}
	state.Accepted(func(initialPosition ui.Offset) Drag { return r.startDrag(initialPosition, pointer) })
}

func (r *MultiDragGestureRecognizerBase) startDrag(initialPosition ui.Offset, pointer int) Drag {
	dartAssert(r.pointers != nil, nil)
	state := r.pointers[pointer]
	dartAssert(state.multiDragPointerState().pendingDelta != nil, nil)
	var drag Drag
	if r.OnStart != nil {
		drag = InvokeCallback(r.self, "onStart", func() Drag { return r.OnStart(initialPosition) })
	}
	if drag != nil {
		state.multiDragPointerState().startDrag(drag)
	} else {
		r.removeState(pointer)
	}
	return drag
}

func (r *MultiDragGestureRecognizerBase) RejectGesture(pointer int) {
	dartAssert(r.pointers != nil, nil)
	if state, ok := r.pointers[pointer]; ok {
		state.Rejected()
		r.removeState(pointer)
	} // else we already preemptively forgot about it (e.g. we got an up event)
}

func (r *MultiDragGestureRecognizerBase) removeState(pointer int) {
	if r.pointers == nil {
		// We've already been disposed. It's harmless to skip removing the
		// state for the given pointer because dispose() has already
		// removed it.
		return
	}
	state, ok := r.pointers[pointer]
	dartAssert(ok, nil)
	GestureBindingInstance().PointerRouter().RemoveRoute(pointer, r.handleEventRoute)
	delete(r.pointers, pointer)
	r.pointerOrder = slices.DeleteFunc(r.pointerOrder, func(p int) bool { return p == pointer })
	state.Dispose()
}

func (r *MultiDragGestureRecognizerBase) Dispose() {
	for _, pointer := range slices.Clone(r.pointerOrder) {
		r.removeState(pointer)
	}
	dartAssert(len(r.pointers) == 0, nil)
	r.pointers = nil
	r.GestureRecognizerBase.Dispose()
}

// ---- the four recognizers

func optionsOf(opts []GestureRecognizerOptions) GestureRecognizerOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return GestureRecognizerOptions{}
}

type immediatePointerState struct{ MultiDragPointerStateBase }

func (s *immediatePointerState) CheckForResolutionAfterMove() {
	dartAssert(s.pendingDelta != nil, nil)
	if s.pendingDelta.Distance() > ComputeHitSlop(s.Kind, s.GestureSettings) {
		s.Resolve(GestureDispositionAccepted)
	}
}

func (s *immediatePointerState) Accepted(starter GestureMultiDragStartCallback) {
	starter(s.InitialPosition)
}

// ImmediateMultiDragGestureRecognizer recognizes movement in any direction
// on a per-pointer basis, starting at once.
type ImmediateMultiDragGestureRecognizer struct {
	MultiDragGestureRecognizerBase
}

// NewImmediateMultiDragGestureRecognizer is
// ImmediateMultiDragGestureRecognizer({debugOwner, supportedDevices,
// allowedButtonsFilter}).
func NewImmediateMultiDragGestureRecognizer(opts ...GestureRecognizerOptions) *ImmediateMultiDragGestureRecognizer {
	r := &ImmediateMultiDragGestureRecognizer{}
	r.Construct(r, optionsOf(opts))
	return r
}

func (r *ImmediateMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerState {
	s := &immediatePointerState{}
	s.Construct(s, event.Position(), event.Kind(), r.GestureSettings)
	return s
}

func (r *ImmediateMultiDragGestureRecognizer) DebugDescription() string { return "multidrag" }

type horizontalPointerState struct{ MultiDragPointerStateBase }

func (s *horizontalPointerState) CheckForResolutionAfterMove() {
	dartAssert(s.pendingDelta != nil, nil)
	if math.Abs(s.pendingDelta.Dx) > ComputeHitSlop(s.Kind, s.GestureSettings) {
		s.Resolve(GestureDispositionAccepted)
	}
}

func (s *horizontalPointerState) Accepted(starter GestureMultiDragStartCallback) {
	starter(s.InitialPosition)
}

// HorizontalMultiDragGestureRecognizer recognizes horizontal movement on a
// per-pointer basis.
type HorizontalMultiDragGestureRecognizer struct {
	MultiDragGestureRecognizerBase
}

// NewHorizontalMultiDragGestureRecognizer is
// HorizontalMultiDragGestureRecognizer({debugOwner, supportedDevices,
// allowedButtonsFilter}).
func NewHorizontalMultiDragGestureRecognizer(opts ...GestureRecognizerOptions) *HorizontalMultiDragGestureRecognizer {
	r := &HorizontalMultiDragGestureRecognizer{}
	r.Construct(r, optionsOf(opts))
	return r
}

func (r *HorizontalMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerState {
	s := &horizontalPointerState{}
	s.Construct(s, event.Position(), event.Kind(), r.GestureSettings)
	return s
}

func (r *HorizontalMultiDragGestureRecognizer) DebugDescription() string {
	return "horizontal multidrag"
}

type verticalPointerState struct{ MultiDragPointerStateBase }

func (s *verticalPointerState) CheckForResolutionAfterMove() {
	dartAssert(s.pendingDelta != nil, nil)
	if math.Abs(s.pendingDelta.Dy) > ComputeHitSlop(s.Kind, s.GestureSettings) {
		s.Resolve(GestureDispositionAccepted)
	}
}

func (s *verticalPointerState) Accepted(starter GestureMultiDragStartCallback) {
	starter(s.InitialPosition)
}

// VerticalMultiDragGestureRecognizer recognizes vertical movement on a
// per-pointer basis.
type VerticalMultiDragGestureRecognizer struct {
	MultiDragGestureRecognizerBase
}

// NewVerticalMultiDragGestureRecognizer is
// VerticalMultiDragGestureRecognizer({debugOwner, supportedDevices,
// allowedButtonsFilter}).
func NewVerticalMultiDragGestureRecognizer(opts ...GestureRecognizerOptions) *VerticalMultiDragGestureRecognizer {
	r := &VerticalMultiDragGestureRecognizer{}
	r.Construct(r, optionsOf(opts))
	return r
}

func (r *VerticalMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerState {
	s := &verticalPointerState{}
	s.Construct(s, event.Position(), event.Kind(), r.GestureSettings)
	return s
}

func (r *VerticalMultiDragGestureRecognizer) DebugDescription() string { return "vertical multidrag" }

type delayedPointerState struct {
	MultiDragPointerStateBase
	timer   *async.Timer
	starter GestureMultiDragStartCallback
}

func newDelayedPointerState(initialPosition ui.Offset, delay time.Duration, kind ui.PointerDeviceKind, gestureSettings *DeviceGestureSettings) *delayedPointerState {
	s := &delayedPointerState{}
	s.Construct(s, initialPosition, kind, gestureSettings)
	s.timer = async.NewTimer(delay, s.delayPassed)
	return s
}

func (s *delayedPointerState) delayPassed() {
	dartAssert(s.timer != nil, nil)
	dartAssert(s.pendingDelta != nil, nil)
	dartAssert(s.pendingDelta.Distance() <= ComputeHitSlop(s.Kind, s.GestureSettings), nil)
	s.timer = nil
	if s.starter != nil {
		s.starter(s.InitialPosition)
		s.starter = nil
	} else {
		s.Resolve(GestureDispositionAccepted)
	}
	dartAssert(s.starter == nil, nil)
}

func (s *delayedPointerState) ensureTimerStopped() {
	if s.timer != nil {
		s.timer.Cancel()
	}
	s.timer = nil
}

func (s *delayedPointerState) Accepted(starter GestureMultiDragStartCallback) {
	dartAssert(s.starter == nil, nil)
	if s.timer == nil {
		starter(s.InitialPosition)
	} else {
		s.starter = starter
	}
}

func (s *delayedPointerState) CheckForResolutionAfterMove() {
	if s.timer == nil {
		// If we've been accepted by the gesture arena but the pointer moves
		// too much before the timer fires, we end up a state where the
		// timer is stopped but we keep getting calls to this function
		// because we never actually started the drag. In this case,
		// _starter will be non-null because we're essentially waiting
		// forever to start the drag.
		dartAssert(s.starter != nil, nil)
		return
	}
	dartAssert(s.pendingDelta != nil, nil)
	if s.pendingDelta.Distance() > ComputeHitSlop(s.Kind, s.GestureSettings) {
		s.Resolve(GestureDispositionRejected)
		s.ensureTimerStopped()
	}
}

func (s *delayedPointerState) Dispose() {
	s.ensureTimerStopped()
	s.MultiDragPointerStateBase.Dispose()
}

// DelayedMultiDragGestureRecognizerOptions are
// DelayedMultiDragGestureRecognizer's parameters (nil Delay:
// kLongPressTimeout).
type DelayedMultiDragGestureRecognizerOptions struct {
	GestureRecognizerOptions
	Delay *time.Duration
}

// DelayedMultiDragGestureRecognizer recognizes movement in any direction
// on a per-pointer basis, after the pointer has been held still for Delay.
type DelayedMultiDragGestureRecognizer struct {
	MultiDragGestureRecognizerBase
	// Delay is the amount of time the pointer must remain in the same
	// place for the drag to be recognized.
	Delay time.Duration
}

// NewDelayedMultiDragGestureRecognizer is
// DelayedMultiDragGestureRecognizer({delay, debugOwner, supportedDevices,
// allowedButtonsFilter}).
func NewDelayedMultiDragGestureRecognizer(opts ...DelayedMultiDragGestureRecognizerOptions) *DelayedMultiDragGestureRecognizer {
	var o DelayedMultiDragGestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r := &DelayedMultiDragGestureRecognizer{Delay: KLongPressTimeout}
	if o.Delay != nil {
		r.Delay = *o.Delay
	}
	r.Construct(r, o.GestureRecognizerOptions)
	return r
}

func (r *DelayedMultiDragGestureRecognizer) CreateNewPointerState(event *PointerDownEvent) MultiDragPointerState {
	return newDelayedPointerState(event.Position(), r.Delay, event.Kind(), r.GestureSettings)
}

func (r *DelayedMultiDragGestureRecognizer) DebugDescription() string { return "long multidrag" }
