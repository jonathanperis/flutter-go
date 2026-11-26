package gestures

// Port of tap_and_drag.dart (Flutter 3.50.1): the TapDrag* details,
// _TapStatusTrackerMixin, BaseTapAndDragGestureRecognizer (sealed) and its
// subclasses TapAndHorizontalDragGestureRecognizer,
// TapAndPanGestureRecognizer and the deprecated TapAndDragGestureRecognizer.
//
// BaseTapAndDragGestureRecognizer is `OneSequenceGestureRecognizer with
// _TapStatusTrackerMixin`: the mixin's state and methods are part of
// BaseTapAndDragGestureRecognizerBase, its overrides (addAllowedPointer,
// handleEvent, rejectGesture, dispose) being the "super" the base's own
// overrides call. The sealed class's private abstract members are
// unexported interface methods, implemented by the three subclasses.

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

func tapAndDragGlobalDistance(event PointerEvent, originPosition *OffsetPair) float64 {
	dartAssert(originPosition != nil, nil)
	offset := event.Position().Sub(originPosition.Global)
	return offset.Distance()
}

// tapDragState is _DragState.
type tapDragState int

const (
	tapDragStateReady tapDragState = iota
	tapDragStatePossible
	tapDragStateAccepted
)

func intKindProperty(name string, kind *ui.PointerDeviceKind) foundation.DiagnosticsNode {
	return foundation.NewEnumProperty(name, kind)
}

// GestureTapDragDownCallback is the signature for when a pointer that might
// cause a tap or drag has contacted the screen.
type GestureTapDragDownCallback func(details *TapDragDownDetails)

// TapDragDownDetailsOptions are TapDragDownDetails' parameters (Kind nil:
// null).
type TapDragDownDetailsOptions struct {
	GlobalPosition      ui.Offset
	LocalPosition       ui.Offset
	Kind                *ui.PointerDeviceKind
	ConsecutiveTapCount int
}

// TapDragDownDetails are the details for GestureTapDragDownCallback.
type TapDragDownDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Kind is the kind of the device that initiated the event (nil: null).
	Kind *ui.PointerDeviceKind
	// ConsecutiveTapCount is the number of consecutive taps that this
	// down event represents.
	ConsecutiveTapCount int
}

// NewTapDragDownDetails creates details for a GestureTapDragDownCallback.
func NewTapDragDownDetails(opts TapDragDownDetailsOptions) *TapDragDownDetails {
	d := &TapDragDownDetails{positionedDetails: positionedDetails{opts.GlobalPosition, opts.LocalPosition},
		Kind: opts.Kind, ConsecutiveTapCount: opts.ConsecutiveTapCount}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions, kind and tap count.
func (d *TapDragDownDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(intKindProperty("kind", d.Kind))
	properties.Add(foundation.NewIntProperty("consecutiveTapCount", d.ConsecutiveTapCount))
}

// GestureTapDragUpCallback is the signature for when a pointer that will
// trigger a tap has stopped contacting the screen.
type GestureTapDragUpCallback func(details *TapDragUpDetails)

// TapDragUpDetailsOptions are TapDragUpDetails' parameters (all required).
type TapDragUpDetailsOptions struct {
	GlobalPosition      ui.Offset
	LocalPosition       ui.Offset
	Kind                ui.PointerDeviceKind
	ConsecutiveTapCount int
}

// TapDragUpDetails are the details for GestureTapDragUpCallback.
type TapDragUpDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Kind is the kind of the device that initiated the event.
	Kind ui.PointerDeviceKind
	// ConsecutiveTapCount is the number of consecutive taps that this up
	// event represents.
	ConsecutiveTapCount int
}

// NewTapDragUpDetails creates details for a GestureTapDragUpCallback.
func NewTapDragUpDetails(opts TapDragUpDetailsOptions) *TapDragUpDetails {
	d := &TapDragUpDetails{positionedDetails: positionedDetails{opts.GlobalPosition, opts.LocalPosition},
		Kind: opts.Kind, ConsecutiveTapCount: opts.ConsecutiveTapCount}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions, kind and tap count.
func (d *TapDragUpDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	kind := d.Kind
	properties.Add(intKindProperty("kind", &kind))
	properties.Add(foundation.NewIntProperty("consecutiveTapCount", d.ConsecutiveTapCount))
}

// GestureTapDragStartCallback is the signature for when a pointer has
// contacted the screen and has begun to drag.
type GestureTapDragStartCallback func(details *TapDragStartDetails)

// TapDragStartDetailsOptions are TapDragStartDetails' parameters (nil:
// null).
type TapDragStartDetailsOptions struct {
	GlobalPosition      ui.Offset
	LocalPosition       ui.Offset
	SourceTimeStamp     *time.Duration
	Kind                *ui.PointerDeviceKind
	ConsecutiveTapCount int
}

// TapDragStartDetails are the details for GestureTapDragStartCallback.
type TapDragStartDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// SourceTimeStamp is the time stamp of the event that started the drag
	// (nil: null).
	SourceTimeStamp *time.Duration
	// Kind is the kind of the device that initiated the event (nil: null).
	Kind *ui.PointerDeviceKind
	// ConsecutiveTapCount is the number of consecutive taps before the
	// drag.
	ConsecutiveTapCount int
}

// NewTapDragStartDetails creates details for a GestureTapDragStartCallback.
func NewTapDragStartDetails(opts TapDragStartDetailsOptions) *TapDragStartDetails {
	d := &TapDragStartDetails{positionedDetails: positionedDetails{opts.GlobalPosition, opts.LocalPosition},
		SourceTimeStamp: opts.SourceTimeStamp, Kind: opts.Kind, ConsecutiveTapCount: opts.ConsecutiveTapCount}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions, time stamp, kind and tap count.
func (d *TapDragStartDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(sourceTimeStampProperty(d.SourceTimeStamp))
	properties.Add(intKindProperty("kind", d.Kind))
	properties.Add(foundation.NewIntProperty("consecutiveTapCount", d.ConsecutiveTapCount))
}

// GestureTapDragUpdateCallback is the signature for when a pointer that is
// in contact with the screen and moving has moved again.
type GestureTapDragUpdateCallback func(details *TapDragUpdateDetails)

// TapDragUpdateDetailsOptions are TapDragUpdateDetails' parameters (nil:
// null).
type TapDragUpdateDetailsOptions struct {
	GlobalPosition        ui.Offset
	LocalPosition         ui.Offset
	SourceTimeStamp       *time.Duration
	Delta                 ui.Offset
	PrimaryDelta          *float64
	Kind                  *ui.PointerDeviceKind
	OffsetFromOrigin      ui.Offset
	LocalOffsetFromOrigin ui.Offset
	ConsecutiveTapCount   int
}

// TapDragUpdateDetails are the details for GestureTapDragUpdateCallback.
type TapDragUpdateDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// SourceTimeStamp is the time stamp of the event that triggered the
	// update (nil: null).
	SourceTimeStamp *time.Duration
	// Delta is the amount the pointer has moved in the local coordinate
	// space since the previous update.
	Delta ui.Offset
	// PrimaryDelta is the amount moved along the primary axis (nil: null).
	PrimaryDelta *float64
	// Kind is the kind of the device that initiated the event (nil: null).
	Kind *ui.PointerDeviceKind
	// OffsetFromOrigin is the global offset from the drag's start.
	OffsetFromOrigin ui.Offset
	// LocalOffsetFromOrigin is the local offset from the drag's start.
	LocalOffsetFromOrigin ui.Offset
	// ConsecutiveTapCount is the number of consecutive taps before the
	// drag.
	ConsecutiveTapCount int
}

// NewTapDragUpdateDetails creates details for a
// GestureTapDragUpdateCallback. A PrimaryDelta must be one of Delta's
// components, the other being zero.
func NewTapDragUpdateDetails(opts TapDragUpdateDetailsOptions) *TapDragUpdateDetails {
	if p := opts.PrimaryDelta; p != nil {
		dartAssert((*p == opts.Delta.Dx && opts.Delta.Dy == 0.0) || (*p == opts.Delta.Dy && opts.Delta.Dx == 0.0), nil)
	}
	d := &TapDragUpdateDetails{positionedDetails: positionedDetails{opts.GlobalPosition, opts.LocalPosition},
		SourceTimeStamp: opts.SourceTimeStamp, Delta: opts.Delta, PrimaryDelta: opts.PrimaryDelta, Kind: opts.Kind,
		OffsetFromOrigin: opts.OffsetFromOrigin, LocalOffsetFromOrigin: opts.LocalOffsetFromOrigin,
		ConsecutiveTapCount: opts.ConsecutiveTapCount}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds every field.
func (d *TapDragUpdateDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(sourceTimeStampProperty(d.SourceTimeStamp))
	properties.Add(foundation.NewDiagnosticsProperty("delta", d.Delta))
	properties.Add(foundation.NewDoubleProperty("primaryDelta", d.PrimaryDelta))
	properties.Add(intKindProperty("kind", d.Kind))
	properties.Add(foundation.NewDiagnosticsProperty("offsetFromOrigin", d.OffsetFromOrigin))
	properties.Add(foundation.NewDiagnosticsProperty("localOffsetFromOrigin", d.LocalOffsetFromOrigin))
	properties.Add(foundation.NewIntProperty("consecutiveTapCount", d.ConsecutiveTapCount))
}

// GestureTapDragEndCallback is the signature for when a pointer that was
// previously in contact with the screen and moving is no longer in
// contact.
type GestureTapDragEndCallback func(endDetails *TapDragEndDetails)

// TapDragEndDetailsOptions are TapDragEndDetails' parameters
// (LocalPosition nil: GlobalPosition; PrimaryVelocity nil: null; the zero
// Velocity is Velocity.zero).
type TapDragEndDetailsOptions struct {
	GlobalPosition      ui.Offset
	LocalPosition       *ui.Offset
	Velocity            Velocity
	PrimaryVelocity     *float64
	ConsecutiveTapCount int
}

// TapDragEndDetails are the details for GestureTapDragEndCallback.
type TapDragEndDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Velocity is the velocity the pointer was moving at when it stopped
	// contacting the screen.
	Velocity Velocity
	// PrimaryVelocity is the velocity along the primary axis (nil: null).
	PrimaryVelocity *float64
	// ConsecutiveTapCount is the number of consecutive taps before the
	// drag.
	ConsecutiveTapCount int
}

// NewTapDragEndDetails creates details for a GestureTapDragEndCallback.
func NewTapDragEndDetails(opts TapDragEndDetailsOptions) *TapDragEndDetails {
	if p := opts.PrimaryVelocity; p != nil {
		v := opts.Velocity.PixelsPerSecond
		dartAssert(*p == v.Dx || *p == v.Dy, nil)
	}
	d := &TapDragEndDetails{positionedDetails: newPositionedDetails(opts.GlobalPosition, opts.LocalPosition),
		Velocity: opts.Velocity, PrimaryVelocity: opts.PrimaryVelocity, ConsecutiveTapCount: opts.ConsecutiveTapCount}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions, velocities and tap count.
func (d *TapDragEndDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
	properties.Add(foundation.NewDoubleProperty("primaryVelocity", d.PrimaryVelocity))
	properties.Add(foundation.NewIntProperty("consecutiveTapCount", d.ConsecutiveTapCount))
}

// GestureCancelCallback is the signature for when the pointer that
// previously triggered a down event will not end up causing a tap.
type GestureCancelCallback func()

// BaseTapAndDragGestureRecognizer recognizes taps along with movement in
// the horizontal or both directions (sealed in Dart: implemented by
// TapAndHorizontalDragGestureRecognizer and TapAndPanGestureRecognizer).
type BaseTapAndDragGestureRecognizer interface {
	OneSequenceGestureRecognizer
	// CurrentDown is the down event of the current tap, or nil.
	CurrentDown() *PointerDownEvent
	// CurrentUp is the up event of the current tap, or nil.
	CurrentUp() *PointerUpEvent
	// ConsecutiveTapCount is the number of consecutive taps that the
	// recognizer has tracked.
	ConsecutiveTapCount() int

	getDeltaForDetails(delta ui.Offset) ui.Offset
	getPrimaryValueFromOffset(value ui.Offset) *float64
	hasSufficientGlobalDistanceToAccept(pointerDeviceKind ui.PointerDeviceKind) bool
}

// BaseTapAndDragGestureRecognizerBase is BaseTapAndDragGestureRecognizer's
// state and methods (with _TapStatusTrackerMixin's), embedded by the
// concrete recognizers.
type BaseTapAndDragGestureRecognizerBase struct {
	OneSequenceGestureRecognizerBase

	self BaseTapAndDragGestureRecognizer

	// DragStartBehavior determines the point at which the drag begins
	// (default DragStartBehaviorStart).
	DragStartBehavior DragStartBehavior
	// DragUpdateThrottleFrequency, if not nil, throttles onDragUpdate to
	// at most one call per that duration.
	DragUpdateThrottleFrequency *time.Duration
	// MaxConsecutiveTap is the maximum number of consecutive taps (nil:
	// unlimited).
	MaxConsecutiveTap *int
	// EagerVictoryOnDrag: the recognizer declares victory as soon as it
	// detects a drag (default true).
	EagerVictoryOnDrag bool

	OnTapDown    GestureTapDragDownCallback
	OnTapUp      GestureTapDragUpCallback
	OnDragStart  GestureTapDragStartCallback
	OnDragUpdate GestureTapDragUpdateCallback
	OnDragEnd    GestureTapDragEndCallback
	OnCancel     GestureCancelCallback

	// OnTapTrackStart is called when a tap is tracked (the mixin's
	// onTapTrackStart).
	OnTapTrackStart func()
	// OnTapTrackReset is called when the tap tracking is reset.
	OnTapTrackReset func()

	// _TapStatusTrackerMixin's state.
	down                *PointerDownEvent
	up                  *PointerUpEvent
	consecutiveTapCount int
	originPosition      *OffsetPair
	previousButtons     *int
	consecutiveTapTimer *async.Timer
	lastTapOffset       *ui.Offset

	pastSlopTolerance          bool
	sentTapDown                bool
	wonArenaForPrimaryPointer  bool
	primaryPointer             *int
	deadlineTimer              *async.Timer
	deadline                   time.Duration
	dragState                  tapDragState
	start                      PointerEvent
	initialPosition            OffsetPair
	currentPosition            OffsetPair
	globalDistanceMoved        float64
	globalDistanceMovedAllAxes float64
	lastDragUpdateDetails      *TapDragUpdateDetails
	dragUpdateThrottleTimer    *async.Timer
	acceptedActivePointers     map[int]struct{}
}

// Construct is BaseTapAndDragGestureRecognizer's constructor (Dart's
// eagerVictoryOnDrag defaults to true).
func (r *BaseTapAndDragGestureRecognizerBase) Construct(self BaseTapAndDragGestureRecognizer, opts GestureRecognizerOptions) {
	r.OneSequenceGestureRecognizerBase.Construct(self, opts)
	r.self = self
	r.EagerVictoryOnDrag = true
	r.deadline = KPressTimeout
	r.DragStartBehavior = DragStartBehaviorStart
	r.acceptedActivePointers = map[int]struct{}{}
}

// ---- _TapStatusTrackerMixin

func (r *BaseTapAndDragGestureRecognizerBase) CurrentDown() *PointerDownEvent { return r.down }

func (r *BaseTapAndDragGestureRecognizerBase) CurrentUp() *PointerUpEvent { return r.up }

func (r *BaseTapAndDragGestureRecognizerBase) ConsecutiveTapCount() int { return r.consecutiveTapCount }

func (r *BaseTapAndDragGestureRecognizerBase) trackerAddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizerBase.AddAllowedPointer(event)
	if r.consecutiveTapTimer != nil && !r.consecutiveTapTimer.IsActive() {
		r.tapTrackerReset()
	}
	if r.MaxConsecutiveTap != nil && *r.MaxConsecutiveTap == r.consecutiveTapCount {
		r.tapTrackerReset()
	}
	r.up = nil
	if r.down != nil && !r.representsSameSeries(event) {
		r.consecutiveTapCount = 1
	} else {
		r.consecutiveTapCount++
	}
	r.consecutiveTapTimerStop()
	r.trackTap(event)
}

func (r *BaseTapAndDragGestureRecognizerBase) trackerHandleEvent(event PointerEvent) {
	switch e := event.(type) {
	case *PointerMoveEvent:
		computedSlop := ComputeHitSlop(e.Kind(), r.GestureSettings)
		isSlopPastTolerance := tapAndDragGlobalDistance(event, r.originPosition) > computedSlop
		if isSlopPastTolerance {
			r.consecutiveTapTimerStop()
			r.previousButtons = nil
			r.lastTapOffset = nil
		}
	case *PointerUpEvent:
		r.up = e
		if r.down != nil {
			r.consecutiveTapTimerStop()
			r.consecutiveTapTimerStart()
		}
	case *PointerCancelEvent:
		r.tapTrackerReset()
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) trackTap(event *PointerDownEvent) {
	r.down = event
	buttons := event.Buttons()
	r.previousButtons = &buttons
	position := event.Position()
	r.lastTapOffset = &position
	r.originPosition = &OffsetPair{Local: event.LocalPosition(), Global: event.Position()}
	if r.OnTapTrackStart != nil {
		r.OnTapTrackStart()
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) hasSameButton(buttons int) bool {
	dartAssert(r.previousButtons != nil, nil)
	return buttons == *r.previousButtons
}

func (r *BaseTapAndDragGestureRecognizerBase) isWithinConsecutiveTapTolerance(secondTapOffset ui.Offset) bool {
	if r.lastTapOffset == nil {
		return false
	}
	difference := secondTapOffset.Sub(*r.lastTapOffset)
	return difference.Distance() <= KDoubleTapSlop
}

func (r *BaseTapAndDragGestureRecognizerBase) representsSameSeries(event *PointerDownEvent) bool {
	return r.consecutiveTapTimer != nil &&
		r.isWithinConsecutiveTapTolerance(event.Position()) &&
		r.hasSameButton(event.Buttons())
}

func (r *BaseTapAndDragGestureRecognizerBase) consecutiveTapTimerStart() {
	if r.consecutiveTapTimer == nil {
		r.consecutiveTapTimer = async.NewTimer(KDoubleTapTimeout, r.consecutiveTapTimerTimeout)
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) consecutiveTapTimerStop() {
	if r.consecutiveTapTimer != nil {
		r.consecutiveTapTimer.Cancel()
		r.consecutiveTapTimer = nil
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) consecutiveTapTimerTimeout() {
	// The consecutive tap timer may time out before a tap down/tap up
	// event is fired. In this case we should not reset the tap tracker
	// state immediately. Instead we should reset the tap tracker on the
	// next call to addAllowedPointer, if the timer is no longer active.
}

func (r *BaseTapAndDragGestureRecognizerBase) tapTrackerReset() {
	r.consecutiveTapTimerStop()
	r.previousButtons = nil
	r.originPosition = nil
	r.lastTapOffset = nil
	r.consecutiveTapCount = 0
	r.down = nil
	r.up = nil
	if r.OnTapTrackReset != nil {
		r.OnTapTrackReset()
	}
}

// ---- BaseTapAndDragGestureRecognizer

func (r *BaseTapAndDragGestureRecognizerBase) handleDragUpdateThrottled() {
	dartAssert(r.lastDragUpdateDetails != nil, nil)
	if r.OnDragUpdate != nil {
		details := r.lastDragUpdateDetails
		InvokeVoidCallback(r.self, "onDragUpdate", func() { r.OnDragUpdate(details) })
	}
	r.dragUpdateThrottleTimer = nil
	r.lastDragUpdateDetails = nil
}

func (r *BaseTapAndDragGestureRecognizerBase) IsPointerAllowed(event *PointerDownEvent) bool {
	if r.primaryPointer == nil {
		switch event.Buttons() {
		case KPrimaryButton:
			if r.OnTapDown == nil && r.OnDragStart == nil && r.OnDragUpdate == nil && r.OnDragEnd == nil &&
				r.OnTapUp == nil && r.OnCancel == nil {
				return false
			}
		default:
			return false
		}
	} else if event.Pointer() != *r.primaryPointer {
		return false
	}
	return r.OneSequenceGestureRecognizerBase.IsPointerAllowed(event)
}

func (r *BaseTapAndDragGestureRecognizerBase) AddAllowedPointer(event *PointerDownEvent) {
	if r.dragState == tapDragStateReady {
		r.trackerAddAllowedPointer(event)
		pointer := event.Pointer()
		r.primaryPointer = &pointer
		r.globalDistanceMoved = 0.0
		r.globalDistanceMovedAllAxes = 0.0
		r.dragState = tapDragStatePossible
		r.initialPosition = OffsetPair{Global: event.Position(), Local: event.LocalPosition()}
		r.currentPosition = r.initialPosition
		r.deadlineTimer = async.NewTimer(r.deadline, func() { r.didExceedDeadlineWithEvent(event) })
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) HandleNonAllowedPointer(event *PointerDownEvent) {
	// There can be multiple drags simultaneously. Their effects are
	// combined.
	if event.Buttons() != KPrimaryButton {
		if !r.wonArenaForPrimaryPointer {
			r.OneSequenceGestureRecognizerBase.HandleNonAllowedPointer(event)
		}
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) isPrimary(pointer int) bool {
	return r.primaryPointer != nil && *r.primaryPointer == pointer
}

func (r *BaseTapAndDragGestureRecognizerBase) removeAccepted(pointer int) bool {
	if _, ok := r.acceptedActivePointers[pointer]; ok {
		delete(r.acceptedActivePointers, pointer)
		return true
	}
	return false
}

func (r *BaseTapAndDragGestureRecognizerBase) AcceptGesture(pointer int) {
	if !r.isPrimary(pointer) {
		return
	}
	r.stopDeadlineTimer()
	_, already := r.acceptedActivePointers[pointer]
	dartAssert(!already, nil)
	r.acceptedActivePointers[pointer] = struct{}{}
	if r.self.CurrentDown() != nil {
		r.checkTapDown(r.self.CurrentDown())
	}
	r.wonArenaForPrimaryPointer = true
	if r.start != nil && r.EagerVictoryOnDrag {
		dartAssert(r.dragState == tapDragStateAccepted, nil)
		dartAssert(r.self.CurrentUp() == nil, nil)
		r.acceptDrag(r.start)
	}
	if r.start != nil && !r.EagerVictoryOnDrag {
		dartAssert(r.dragState == tapDragStatePossible, nil)
		dartAssert(r.self.CurrentUp() == nil, nil)
		r.dragState = tapDragStateAccepted
		r.acceptDrag(r.start)
	}
	if r.self.CurrentUp() != nil {
		r.checkTapUp(r.self.CurrentUp())
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) DidStopTrackingLastPointer(pointer int) {
	switch r.dragState {
	case tapDragStateReady:
		r.checkCancel()
		r.self.Resolve(GestureDispositionRejected)
	case tapDragStatePossible:
		if r.pastSlopTolerance {
			// This means the pointer was not accepted as a tap.
			if r.wonArenaForPrimaryPointer {
				// If the recognizer has already won the arena for the
				// primary pointer being tracked but the pointer has
				// exceeded the tap tolerance, then the pointer is accepted
				// as a drag gesture.
				if r.self.CurrentDown() != nil {
					if !r.removeAccepted(pointer) {
						r.self.ResolvePointer(pointer, GestureDispositionRejected)
					}
					r.dragState = tapDragStateAccepted
					r.acceptDrag(r.self.CurrentDown())
					r.checkDragEnd()
				}
			} else {
				r.checkCancel()
				r.self.Resolve(GestureDispositionRejected)
			}
		} else {
			// The pointer is accepted as a tap.
			if r.self.CurrentUp() != nil {
				r.checkTapUp(r.self.CurrentUp())
			}
		}
	case tapDragStateAccepted:
		// For the case when the pointer has been accepted as a drag.
		// Meaning _checkTapDown and _checkDragStart have already been
		// called.
		r.checkDragEnd()
	}
	r.stopDeadlineTimer()
	r.start = nil
	r.dragState = tapDragStateReady
	r.pastSlopTolerance = false
}

func (r *BaseTapAndDragGestureRecognizerBase) HandleEvent(event PointerEvent) {
	if !r.isPrimary(event.Pointer()) {
		return
	}
	r.trackerHandleEvent(event)
	switch e := event.(type) {
	case *PointerMoveEvent:
		// Receiving a PointerMoveEvent does not mean the pointer is past
		// the slop.
		computedSlop := ComputeHitSlop(e.Kind(), r.GestureSettings)
		r.pastSlopTolerance = r.pastSlopTolerance || tapAndDragGlobalDistance(event, &r.initialPosition) > computedSlop
		if r.dragState == tapDragStateAccepted {
			r.currentPosition = NewOffsetPairFromEventPosition(event)
			r.checkDragUpdate(event, nil)
		} else if r.dragState == tapDragStatePossible {
			if r.start == nil {
				// Only check for a drag if the start of a drag was not
				// already identified.
				r.checkDrag(e)
			}
			// This can occur when the recognizer is accepted before a
			// PointerMoveEvent has been received that moves the pointer
			// a sufficient global distance to be considered a drag.
			if r.start != nil && r.wonArenaForPrimaryPointer {
				r.dragState = tapDragStateAccepted
				r.acceptDrag(r.start)
			}
		}
	case *PointerUpEvent:
		if r.dragState == tapDragStatePossible {
			// The drag has not been accepted before a PointerUpEvent,
			// therefore the recognizer attempts to recognize a tap.
			r.self.StopTrackingIfPointerNoLongerDown(event)
		} else if r.dragState == tapDragStateAccepted {
			r.giveUpPointer(event.Pointer())
		}
	case *PointerCancelEvent:
		r.dragState = tapDragStateReady
		r.giveUpPointer(event.Pointer())
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) RejectGesture(pointer int) {
	if !r.isPrimary(pointer) {
		return
	}
	// _TapStatusTrackerMixin.rejectGesture (which doesn't call its super).
	r.tapTrackerReset()
	r.stopDeadlineTimer()
	r.giveUpPointer(pointer)
	r.resetTaps()
	r.resetDragUpdateThrottle()
}

func (r *BaseTapAndDragGestureRecognizerBase) Dispose() {
	r.stopDeadlineTimer()
	r.resetDragUpdateThrottle()
	// _TapStatusTrackerMixin.dispose.
	r.tapTrackerReset()
	r.OneSequenceGestureRecognizerBase.Dispose()
}

func (r *BaseTapAndDragGestureRecognizerBase) DebugDescription() string { return "tap_and_drag" }

func (r *BaseTapAndDragGestureRecognizerBase) acceptDrag(event PointerEvent) {
	dartAssert(r.dragState == tapDragStateAccepted, nil)
	if !r.wonArenaForPrimaryPointer {
		return
	}
	if r.DragStartBehavior == DragStartBehaviorStart {
		r.initialPosition = r.initialPosition.Add(OffsetPair{Global: event.Delta(), Local: event.LocalDelta()})
		r.currentPosition = r.initialPosition
	}
	r.checkDragStart(event)
	localDelta := event.LocalDelta()
	if localDelta != (ui.Offset{}) {
		r.currentPosition = NewOffsetPairFromEventPosition(event)
		correctedLocalPosition := r.initialPosition.Local.Add(localDelta)
		var localToGlobalTransform *vm.Matrix4
		if event.Transform() != nil {
			localToGlobalTransform = vm.Matrix4TryInvert(event.Transform())
		}
		globalUpdateDelta := PointerEventTransformDeltaViaPositions(correctedLocalPosition, nil, localDelta, localToGlobalTransform)
		updateDelta := OffsetPair{Local: localDelta, Global: globalUpdateDelta}
		corrected := r.initialPosition.Add(updateDelta)
		r.checkDragUpdate(event, &corrected)
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) checkDrag(event *PointerMoveEvent) {
	var localToGlobalTransform *vm.Matrix4
	if event.Transform() != nil {
		localToGlobalTransform = vm.Matrix4TryInvert(event.Transform())
	}
	movedLocally := r.self.getDeltaForDetails(event.LocalDelta())
	sign := 1.0
	if v := r.self.getPrimaryValueFromOffset(movedLocally); v != nil {
		sign = dartSign(*v)
	}
	r.globalDistanceMoved += float64(PointerEventTransformDeltaViaPositions(event.LocalPosition(), nil, movedLocally,
		localToGlobalTransform).Distance() * sign)
	r.globalDistanceMovedAllAxes += float64(PointerEventTransformDeltaViaPositions(event.LocalPosition(), nil, event.LocalDelta(),
		localToGlobalTransform).Distance() * 1)
	if r.self.hasSufficientGlobalDistanceToAccept(event.Kind()) ||
		(r.wonArenaForPrimaryPointer && absFloat(r.globalDistanceMovedAllAxes) > ComputePanSlop(event.Kind(), r.GestureSettings)) {
		r.start = event
		if r.EagerVictoryOnDrag {
			r.dragState = tapDragStateAccepted
			if !r.wonArenaForPrimaryPointer {
				r.self.Resolve(GestureDispositionAccepted)
			}
		}
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func (r *BaseTapAndDragGestureRecognizerBase) checkTapDown(event *PointerDownEvent) {
	if r.sentTapDown {
		return
	}
	kind := r.self.GetKindForPointer(event.Pointer())
	details := NewTapDragDownDetails(TapDragDownDetailsOptions{
		GlobalPosition:      event.Position(),
		LocalPosition:       event.LocalPosition(),
		Kind:                &kind,
		ConsecutiveTapCount: r.self.ConsecutiveTapCount(),
	})
	if r.OnTapDown != nil {
		InvokeVoidCallback(r.self, "onTapDown", func() { r.OnTapDown(details) })
	}
	r.sentTapDown = true
}

func (r *BaseTapAndDragGestureRecognizerBase) checkTapUp(event *PointerUpEvent) {
	if !r.wonArenaForPrimaryPointer {
		return
	}
	upDetails := NewTapDragUpDetails(TapDragUpDetailsOptions{
		Kind:                event.Kind(),
		GlobalPosition:      event.Position(),
		LocalPosition:       event.LocalPosition(),
		ConsecutiveTapCount: r.self.ConsecutiveTapCount(),
	})
	if r.OnTapUp != nil {
		InvokeVoidCallback(r.self, "onTapUp", func() { r.OnTapUp(upDetails) })
	}
	r.resetTaps()
	if !r.removeAccepted(event.Pointer()) {
		r.self.ResolvePointer(event.Pointer(), GestureDispositionRejected)
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) checkDragStart(event PointerEvent) {
	if r.OnDragStart != nil {
		ts := event.TimeStamp()
		kind := r.self.GetKindForPointer(event.Pointer())
		local := r.initialPosition.Local
		details := NewTapDragStartDetails(TapDragStartDetailsOptions{
			SourceTimeStamp:     &ts,
			GlobalPosition:      r.initialPosition.Global,
			LocalPosition:       local,
			Kind:                &kind,
			ConsecutiveTapCount: r.self.ConsecutiveTapCount(),
		})
		InvokeVoidCallback(r.self, "onDragStart", func() { r.OnDragStart(details) })
	}
	r.start = nil
}

func (r *BaseTapAndDragGestureRecognizerBase) checkDragUpdate(event PointerEvent, corrected *OffsetPair) {
	globalPosition := event.Position()
	localPosition := event.LocalPosition()
	if corrected != nil {
		globalPosition = corrected.Global
		localPosition = corrected.Local
	}
	ts := event.TimeStamp()
	kind := r.self.GetKindForPointer(event.Pointer())
	details := NewTapDragUpdateDetails(TapDragUpdateDetailsOptions{
		SourceTimeStamp:       &ts,
		Delta:                 event.LocalDelta(),
		GlobalPosition:        globalPosition,
		Kind:                  &kind,
		LocalPosition:         localPosition,
		OffsetFromOrigin:      globalPosition.Sub(r.initialPosition.Global),
		LocalOffsetFromOrigin: localPosition.Sub(r.initialPosition.Local),
		ConsecutiveTapCount:   r.self.ConsecutiveTapCount(),
	})
	if r.DragUpdateThrottleFrequency != nil {
		r.lastDragUpdateDetails = details
		// Only schedule a new timer if there's not one pending.
		if r.dragUpdateThrottleTimer == nil {
			r.dragUpdateThrottleTimer = async.NewTimer(*r.DragUpdateThrottleFrequency, r.handleDragUpdateThrottled)
		}
	} else if r.OnDragUpdate != nil {
		InvokeVoidCallback(r.self, "onDragUpdate", func() { r.OnDragUpdate(details) })
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) checkDragEnd() {
	globalPosition := r.currentPosition.Global
	localPosition := r.currentPosition.Local
	if r.dragUpdateThrottleTimer != nil {
		// If there's already an update scheduled, trigger it immediately
		// and cancel the timer.
		r.dragUpdateThrottleTimer.Cancel()
		r.handleDragUpdateThrottled()
	}
	zero := 0.0
	endDetails := NewTapDragEndDetails(TapDragEndDetailsOptions{
		GlobalPosition:      globalPosition,
		LocalPosition:       &localPosition,
		PrimaryVelocity:     &zero,
		ConsecutiveTapCount: r.self.ConsecutiveTapCount(),
	})
	if r.OnDragEnd != nil {
		InvokeVoidCallback(r.self, "onDragEnd", func() { r.OnDragEnd(endDetails) })
	}
	r.resetTaps()
	r.resetDragUpdateThrottle()
}

func (r *BaseTapAndDragGestureRecognizerBase) checkCancel() {
	if !r.sentTapDown {
		// Do not fire tap cancel if [onTapDown] was never called.
		return
	}
	if r.OnCancel != nil {
		InvokeVoidCallback(r.self, "onCancel", func() { r.OnCancel() })
	}
	r.resetDragUpdateThrottle()
	r.resetTaps()
}

func (r *BaseTapAndDragGestureRecognizerBase) didExceedDeadlineWithEvent(event *PointerDownEvent) {
	r.didExceedDeadline()
}

func (r *BaseTapAndDragGestureRecognizerBase) didExceedDeadline() {
	if r.self.CurrentDown() != nil {
		r.checkTapDown(r.self.CurrentDown())
		if r.self.ConsecutiveTapCount() > 1 {
			// If consecutive tap count is greater than 1, i.e. is a double
			// tap or greater, then this recognizer declares victory to
			// prevent the tap from being rejected by other recognizers.
			r.self.Resolve(GestureDispositionAccepted)
		}
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) giveUpPointer(pointer int) {
	r.self.StopTrackingPointer(pointer)
	// If the pointer was never accepted, then it is rejected since this
	// recognizer is no longer interested in winning the gesture arena for
	// it.
	if !r.removeAccepted(pointer) {
		r.self.ResolvePointer(pointer, GestureDispositionRejected)
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) resetTaps() {
	r.sentTapDown = false
	r.wonArenaForPrimaryPointer = false
	r.primaryPointer = nil
}

func (r *BaseTapAndDragGestureRecognizerBase) resetDragUpdateThrottle() {
	if r.DragUpdateThrottleFrequency == nil {
		return
	}
	r.lastDragUpdateDetails = nil
	if r.dragUpdateThrottleTimer != nil {
		r.dragUpdateThrottleTimer.Cancel()
		r.dragUpdateThrottleTimer = nil
	}
}

func (r *BaseTapAndDragGestureRecognizerBase) stopDeadlineTimer() {
	if r.deadlineTimer != nil {
		r.deadlineTimer.Cancel()
		r.deadlineTimer = nil
	}
}

// tapAndDragOptions keeps the parameters the concrete recognizers take
// (debugOwner and supportedDevices; their allowedButtonsFilter is the
// default).
func tapAndDragOptions(opts []GestureRecognizerOptions) GestureRecognizerOptions {
	var o GestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return GestureRecognizerOptions{DebugOwner: o.DebugOwner, SupportedDevices: o.SupportedDevices}
}

// TapAndHorizontalDragGestureRecognizer recognizes taps along with
// horizontal drags.
type TapAndHorizontalDragGestureRecognizer struct {
	BaseTapAndDragGestureRecognizerBase
}

// NewTapAndHorizontalDragGestureRecognizer is
// TapAndHorizontalDragGestureRecognizer({debugOwner, supportedDevices}).
func NewTapAndHorizontalDragGestureRecognizer(opts ...GestureRecognizerOptions) *TapAndHorizontalDragGestureRecognizer {
	r := &TapAndHorizontalDragGestureRecognizer{}
	r.Construct(r, tapAndDragOptions(opts))
	return r
}

func (r *TapAndHorizontalDragGestureRecognizer) hasSufficientGlobalDistanceToAccept(kind ui.PointerDeviceKind) bool {
	return absFloat(r.globalDistanceMoved) > ComputeHitSlop(kind, r.GestureSettings)
}

func (r *TapAndHorizontalDragGestureRecognizer) getDeltaForDetails(delta ui.Offset) ui.Offset {
	return ui.NewOffset(delta.Dx, 0.0)
}

func (r *TapAndHorizontalDragGestureRecognizer) getPrimaryValueFromOffset(value ui.Offset) *float64 {
	v := value.Dx
	return &v
}

func (r *TapAndHorizontalDragGestureRecognizer) DebugDescription() string {
	return "tap and horizontal drag"
}

// TapAndPanGestureRecognizer recognizes taps along with movement in any
// direction.
type TapAndPanGestureRecognizer struct {
	BaseTapAndDragGestureRecognizerBase
}

// NewTapAndPanGestureRecognizer is TapAndPanGestureRecognizer({debugOwner,
// supportedDevices}).
func NewTapAndPanGestureRecognizer(opts ...GestureRecognizerOptions) *TapAndPanGestureRecognizer {
	r := &TapAndPanGestureRecognizer{}
	r.Construct(r, tapAndDragOptions(opts))
	return r
}

func (r *TapAndPanGestureRecognizer) hasSufficientGlobalDistanceToAccept(kind ui.PointerDeviceKind) bool {
	return absFloat(r.globalDistanceMoved) > ComputePanSlop(kind, r.GestureSettings)
}

func (r *TapAndPanGestureRecognizer) getDeltaForDetails(delta ui.Offset) ui.Offset { return delta }

func (r *TapAndPanGestureRecognizer) getPrimaryValueFromOffset(value ui.Offset) *float64 { return nil }

func (r *TapAndPanGestureRecognizer) DebugDescription() string { return "tap and pan" }

// TapAndDragGestureRecognizer is TapAndPanGestureRecognizer under its old
// name.
//
// Deprecated: Use TapAndPanGestureRecognizer instead (deprecated after
// v3.9.0-19.0.pre).
type TapAndDragGestureRecognizer struct {
	BaseTapAndDragGestureRecognizerBase
}

// NewTapAndDragGestureRecognizer is TapAndDragGestureRecognizer.
//
// Deprecated: Use NewTapAndPanGestureRecognizer instead.
func NewTapAndDragGestureRecognizer(opts ...GestureRecognizerOptions) *TapAndDragGestureRecognizer {
	r := &TapAndDragGestureRecognizer{}
	r.Construct(r, tapAndDragOptions(opts))
	return r
}

func (r *TapAndDragGestureRecognizer) hasSufficientGlobalDistanceToAccept(kind ui.PointerDeviceKind) bool {
	return absFloat(r.globalDistanceMoved) > ComputePanSlop(kind, r.GestureSettings)
}

func (r *TapAndDragGestureRecognizer) getDeltaForDetails(delta ui.Offset) ui.Offset { return delta }

func (r *TapAndDragGestureRecognizer) getPrimaryValueFromOffset(value ui.Offset) *float64 { return nil }

func (r *TapAndDragGestureRecognizer) DebugDescription() string { return "tap and pan" }
