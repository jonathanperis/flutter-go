package gestures

// Port of monodrag.dart (Flutter 3.50.1): DragGestureRecognizer (sealed in
// Dart) and its three subclasses, VerticalDragGestureRecognizer,
// HorizontalDragGestureRecognizer and PanGestureRecognizer.
//
// DragGestureRecognizer is an interface plus the embedded
// DragGestureRecognizerBase holding the state, the callbacks and the
// configuration (Dart's public fields are exported fields: OnStart,
// MinFlingVelocity...). Dart's private abstract members
// (_getDeltaForDetails, _getPrimaryValueFromOffset, _getPrimaryDragAxis)
// are unexported interface methods: the class is sealed, so only this
// package implements it.

import (
	"math"
	"slices"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

type dragState int

const (
	dragStateReady dragState = iota
	dragStatePossible
	dragStateAccepted
)

// GestureDragEndCallback is the signature for when a pointer that was
// previously in contact with the screen and moving is no longer in
// contact with the screen.
type GestureDragEndCallback func(details *DragEndDetails)

// GestureDragCancelCallback is the signature for when the pointer that
// previously triggered a GestureDragDownCallback did not complete.
type GestureDragCancelCallback func()

// GestureVelocityTrackerBuilder is the signature for a function that
// builds a VelocityTracker (a subclass's embedded *VelocityTracker, e.g.
// &NewIOSScrollViewFlingVelocityTracker(kind).VelocityTracker, behaves as
// the subclass).
type GestureVelocityTrackerBuilder func(event PointerEvent) *VelocityTracker

type dragDirection int

const (
	dragDirectionHorizontal dragDirection = iota
	dragDirectionVertical
)

// DragGestureRecognizer recognizes movement (a sealed class in Dart: the
// vertical, horizontal and pan recognizers).
type DragGestureRecognizer interface {
	OneSequenceGestureRecognizer
	// IsFlingGesture determines if a gesture is a fling or not based on
	// velocity.
	IsFlingGesture(estimate VelocityEstimate, kind ui.PointerDeviceKind) bool
	// ConsiderFling returns the DragEndDetails of a fling, or nil if the
	// estimate is not a fling.
	ConsiderFling(estimate VelocityEstimate, kind ui.PointerDeviceKind) *DragEndDetails
	// HasSufficientGlobalDistanceToAccept determines if the drag moved
	// enough to win the arena.
	HasSufficientGlobalDistanceToAccept(pointerDeviceKind ui.PointerDeviceKind, deviceTouchSlop *float64) bool

	getDeltaForDetails(delta ui.Offset) ui.Offset
	getPrimaryValueFromOffset(value ui.Offset) *float64
	getPrimaryDragAxis() *dragDirection
	dragBase() *DragGestureRecognizerBase
}

// DragGestureRecognizerBase is DragGestureRecognizer's state and methods,
// embedded by the three recognizers.
type DragGestureRecognizerBase struct {
	OneSequenceGestureRecognizerBase

	self DragGestureRecognizer

	// DragStartBehavior configures the drag start (default
	// DragStartBehaviorStart).
	DragStartBehavior DragStartBehavior
	// MultitouchDragStrategy configures how several pointers drag (default
	// latestPointer).
	MultitouchDragStrategy MultitouchDragStrategy
	// OnDown is called when a pointer has contacted the screen and might
	// begin to move.
	OnDown GestureDragDownCallback
	// OnStart is called when a pointer has contacted the screen and has
	// begun to move.
	OnStart GestureDragStartCallback
	// OnUpdate is called when a pointer that is in contact with the screen
	// and moving has moved again.
	OnUpdate GestureDragUpdateCallback
	// OnEnd is called when a pointer that was previously in contact with
	// the screen and moving is no longer in contact with the screen.
	OnEnd GestureDragEndCallback
	// OnCancel is called when the pointer that previously triggered OnDown
	// did not complete.
	OnCancel GestureDragCancelCallback
	// MinFlingDistance is the minimum distance an input pointer drag must
	// have moved to be considered a fling gesture (nil: the hit slop).
	MinFlingDistance *float64
	// MinFlingVelocity is the minimum velocity for a fling (nil:
	// kMinFlingVelocity).
	MinFlingVelocity *float64
	// MaxFlingVelocity is the fling velocity's upper bound (nil:
	// kMaxFlingVelocity).
	MaxFlingVelocity *float64
	// OnlyAcceptDragOnThreshold: whether the drag should only be accepted
	// once the threshold is met (even if it won the arena earlier).
	OnlyAcceptDragOnThreshold bool
	// VelocityTrackerBuilder builds the velocity tracker of each pointer
	// (default VelocityTracker.withKind).
	VelocityTrackerBuilder GestureVelocityTrackerBuilder

	state                     dragState
	initialPosition           OffsetPair
	pendingDragOffset         OffsetPair
	lastPosition              OffsetPair
	lastPendingEventTimestamp *time.Duration
	initialButtons            *int
	lastTransform             *vm.Matrix4
	globalDistanceMoved       float64
	hasDragThresholdBeenMet   bool
	velocityTrackers          map[int]*VelocityTracker
	// moveDeltaBeforeFrame is a Map<int, Offset> (insertion order).
	moveDeltaPointers      []int
	moveDeltaBeforeFrame   map[int]ui.Offset
	frameTimeStamp         *time.Duration
	lastUpdatedDeltaForPan ui.Offset
	acceptedActivePointers []int
	activePointer          *int
}

// DragGestureRecognizerOptions are DragGestureRecognizer's constructor
// parameters (nil DragStartBehavior: start; nil VelocityTrackerBuilder:
// VelocityTracker.withKind; nil AllowedButtonsFilter: the primary button
// only).
type DragGestureRecognizerOptions struct {
	GestureRecognizerOptions
	DragStartBehavior         *DragStartBehavior
	MultitouchDragStrategy    MultitouchDragStrategy
	VelocityTrackerBuilder    GestureVelocityTrackerBuilder
	OnlyAcceptDragOnThreshold bool
}

func defaultVelocityTrackerBuilder(event PointerEvent) *VelocityTracker {
	return NewVelocityTrackerWithKind(event.Kind())
}

func dragDefaultButtonAcceptBehavior(buttons int) bool { return buttons == KPrimaryButton }

// Construct is DragGestureRecognizer's constructor.
func (r *DragGestureRecognizerBase) Construct(self DragGestureRecognizer, opts DragGestureRecognizerOptions) {
	if opts.AllowedButtonsFilter == nil {
		opts.AllowedButtonsFilter = dragDefaultButtonAcceptBehavior
	}
	r.OneSequenceGestureRecognizerBase.Construct(self, opts.GestureRecognizerOptions)
	r.self = self
	r.DragStartBehavior = DragStartBehaviorStart
	if opts.DragStartBehavior != nil {
		r.DragStartBehavior = *opts.DragStartBehavior
	}
	r.MultitouchDragStrategy = opts.MultitouchDragStrategy
	r.VelocityTrackerBuilder = opts.VelocityTrackerBuilder
	if r.VelocityTrackerBuilder == nil {
		r.VelocityTrackerBuilder = defaultVelocityTrackerBuilder
	}
	r.OnlyAcceptDragOnThreshold = opts.OnlyAcceptDragOnThreshold
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.moveDeltaBeforeFrame = map[int]ui.Offset{}
}

func (r *DragGestureRecognizerBase) dragBase() *DragGestureRecognizerBase { return r }

// LastPosition is the most recent position of the pointer that is driving
// the drag.
func (r *DragGestureRecognizerBase) LastPosition() OffsetPair { return r.lastPosition }

// GlobalDistanceMoved is the distance moved, in global coordinates, along
// the primary axis (signed), before the drag was accepted.
func (r *DragGestureRecognizerBase) GlobalDistanceMoved() float64 { return r.globalDistanceMoved }

// DebugLastPendingEventTimestamp is the timestamp of the last pending
// event (nil outside debug builds).
func (r *DragGestureRecognizerBase) DebugLastPendingEventTimestamp() *time.Duration {
	if foundation.KDebugMode {
		return r.lastPendingEventTimestamp
	}
	return nil
}

func (r *DragGestureRecognizerBase) IsPointerAllowed(event *PointerDownEvent) bool {
	if r.initialButtons == nil {
		if r.OnDown == nil && r.OnStart == nil && r.OnUpdate == nil && r.OnEnd == nil && r.OnCancel == nil {
			return false
		}
	} else {
		// There can be multiple drags simultaneously. Their effects are
		// combined.
		if event.Buttons() != *r.initialButtons {
			return false
		}
	}
	return r.OneSequenceGestureRecognizerBase.IsPointerAllowed(event)
}

func (r *DragGestureRecognizerBase) addPointer(event PointerEvent) {
	r.velocityTrackers[event.Pointer()] = r.VelocityTrackerBuilder(event)
	switch r.state {
	case dragStateReady:
		r.state = dragStatePossible
		r.initialPosition = OffsetPair{Global: event.Position(), Local: event.LocalPosition()}
		r.lastPosition = r.initialPosition
		r.pendingDragOffset = OffsetPairZero
		r.globalDistanceMoved = 0.0
		ts := event.TimeStamp()
		r.lastPendingEventTimestamp = &ts
		r.lastTransform = event.Transform()
		r.checkDown()
	case dragStatePossible:
	case dragStateAccepted:
		r.self.Resolve(GestureDispositionAccepted)
	}
}

func (r *DragGestureRecognizerBase) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizerBase.AddAllowedPointer(event)
	if r.state == dragStateReady {
		b := event.Buttons()
		r.initialButtons = &b
	}
	r.addPointer(event)
}

func (r *DragGestureRecognizerBase) AddAllowedPointerPanZoom(event *PointerPanZoomStartEvent) {
	r.OneSequenceGestureRecognizerBase.AddAllowedPointerPanZoom(event)
	r.self.StartTrackingPointer(event.Pointer(), event.Transform())
	if r.state == dragStateReady {
		b := KPrimaryButton
		r.initialButtons = &b
	}
	r.addPointer(event)
}

func (r *DragGestureRecognizerBase) shouldTrackMoveEvent(pointer int) bool {
	switch r.MultitouchDragStrategy {
	case MultitouchDragStrategySumAllPointers, MultitouchDragStrategyAverageBoundaryPointers:
		return true
	default: // latestPointer
		return r.activePointer == nil || pointer == *r.activePointer
	}
}

func durationPtrEqual(a, b *time.Duration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func currentSystemFrameTimeStamp() *time.Duration {
	ts := scheduler.SchedulerBindingInstance().CurrentSystemFrameTimeStamp()
	return &ts
}

func (r *DragGestureRecognizerBase) recordMoveDeltaForMultitouch(pointer int, localDelta ui.Offset) {
	if r.MultitouchDragStrategy != MultitouchDragStrategyAverageBoundaryPointers {
		dartAssert(r.frameTimeStamp == nil, nil)
		dartAssert(len(r.moveDeltaBeforeFrame) == 0, nil)
		return
	}
	dartAssert(durationPtrEqual(r.frameTimeStamp, currentSystemFrameTimeStamp()), nil)
	if r.state != dragStateAccepted || localDelta == (ui.Offset{}) {
		return
	}
	if offset, ok := r.moveDeltaBeforeFrame[pointer]; ok {
		r.moveDeltaBeforeFrame[pointer] = offset.Add(localDelta)
	} else {
		r.moveDeltaBeforeFrame[pointer] = localDelta
		r.moveDeltaPointers = append(r.moveDeltaPointers, pointer)
	}
}

func (r *DragGestureRecognizerBase) clearMoveDeltas() {
	r.moveDeltaBeforeFrame = map[int]ui.Offset{}
	r.moveDeltaPointers = nil
}

func (r *DragGestureRecognizerBase) getSumDelta(pointer int, positive bool, axis dragDirection) float64 {
	sum := 0.0
	offset, ok := r.moveDeltaBeforeFrame[pointer]
	if !ok {
		return sum
	}
	if positive {
		if axis == dragDirectionVertical {
			sum = dartMax(offset.Dy, 0.0)
		} else {
			sum = dartMax(offset.Dx, 0.0)
		}
	} else {
		if axis == dragDirectionVertical {
			sum = dartMin(offset.Dy, 0.0)
		} else {
			sum = dartMin(offset.Dx, 0.0)
		}
	}
	return sum
}

// dartMax and dartMin are dart:math's max/min on doubles (NaN wins; -0.0
// is less than 0.0).
func dartMax(a, b float64) float64 {
	switch {
	case math.IsNaN(a) || math.IsNaN(b):
		return math.NaN()
	case a > b:
		return a
	case a < b:
		return b
	case math.Signbit(a):
		return b
	}
	return a
}

func dartMin(a, b float64) float64 {
	switch {
	case math.IsNaN(a) || math.IsNaN(b):
		return math.NaN()
	case a < b:
		return a
	case a > b:
		return b
	case math.Signbit(a):
		return a
	}
	return b
}

func (r *DragGestureRecognizerBase) getMaxSumDeltaPointer(positive bool, axis dragDirection) *int {
	if len(r.moveDeltaBeforeFrame) == 0 {
		return nil
	}
	var ret *int
	var maxValue float64
	for _, pointer := range r.moveDeltaPointers {
		sum := r.getSumDelta(pointer, positive, axis)
		if ret == nil {
			p := pointer
			ret = &p
			maxValue = sum
		} else if positive {
			if sum > maxValue {
				p := pointer
				ret = &p
				maxValue = sum
			}
		} else {
			if sum < maxValue {
				p := pointer
				ret = &p
				maxValue = sum
			}
		}
	}
	dartAssert(ret != nil, nil)
	return ret
}

func (r *DragGestureRecognizerBase) resolveLocalDeltaForMultitouch(pointer int, localDelta ui.Offset) ui.Offset {
	if r.MultitouchDragStrategy != MultitouchDragStrategyAverageBoundaryPointers {
		if r.frameTimeStamp != nil {
			r.clearMoveDeltas()
			r.frameTimeStamp = nil
			r.lastUpdatedDeltaForPan = ui.Offset{}
		}
		return localDelta
	}
	current := currentSystemFrameTimeStamp()
	if !durationPtrEqual(r.frameTimeStamp, current) {
		r.clearMoveDeltas()
		r.lastUpdatedDeltaForPan = ui.Offset{}
		r.frameTimeStamp = current
	}
	dartAssert(durationPtrEqual(r.frameTimeStamp, currentSystemFrameTimeStamp()), nil)
	axis := r.self.getPrimaryDragAxis()
	if r.state != dragStateAccepted || localDelta == (ui.Offset{}) || (len(r.moveDeltaBeforeFrame) == 0 && axis != nil) {
		return localDelta
	}
	var dx, dy float64
	if axis != nil && *axis == dragDirectionHorizontal {
		dx = r.resolveDelta(pointer, dragDirectionHorizontal, localDelta)
		dartAssert(math.Abs(dx) <= math.Abs(localDelta.Dx), nil)
		dy = 0.0
	} else if axis != nil && *axis == dragDirectionVertical {
		dx = 0.0
		dy = r.resolveDelta(pointer, dragDirectionVertical, localDelta)
		dartAssert(math.Abs(dy) <= math.Abs(localDelta.Dy), nil)
	} else {
		averageX := r.resolveDeltaForPanGesture(dragDirectionHorizontal, localDelta)
		averageY := r.resolveDeltaForPanGesture(dragDirectionVertical, localDelta)
		updatedDelta := ui.NewOffset(averageX, averageY).Sub(r.lastUpdatedDeltaForPan)
		r.lastUpdatedDeltaForPan = ui.NewOffset(averageX, averageY)
		dx = updatedDelta.Dx
		dy = updatedDelta.Dy
	}
	return ui.NewOffset(dx, dy)
}

func (r *DragGestureRecognizerBase) resolveDelta(pointer int, axis dragDirection, localDelta ui.Offset) float64 {
	var positive bool
	var delta float64
	if axis == dragDirectionHorizontal {
		positive = localDelta.Dx > 0
		delta = localDelta.Dx
	} else {
		positive = localDelta.Dy > 0
		delta = localDelta.Dy
	}
	maxSumDeltaPointer := r.getMaxSumDeltaPointer(positive, axis)
	dartAssert(maxSumDeltaPointer != nil, nil)
	if *maxSumDeltaPointer == pointer {
		return delta
	}
	maxSumDelta := r.getSumDelta(*maxSumDeltaPointer, positive, axis)
	curPointerSumDelta := r.getSumDelta(pointer, positive, axis)
	if positive {
		if curPointerSumDelta+delta > maxSumDelta {
			return curPointerSumDelta + delta - maxSumDelta
		}
		return 0.0
	}
	if curPointerSumDelta+delta < maxSumDelta {
		return curPointerSumDelta + delta - maxSumDelta
	}
	return 0.0
}

func (r *DragGestureRecognizerBase) resolveDeltaForPanGesture(axis dragDirection, localDelta ui.Offset) float64 {
	delta := localDelta.Dy
	if axis == dragDirectionHorizontal {
		delta = localDelta.Dx
	}
	pointerCount := len(r.acceptedActivePointers)
	dartAssert(pointerCount >= 1, nil)
	sum := delta
	for _, pointer := range r.moveDeltaPointers {
		offset := r.moveDeltaBeforeFrame[pointer]
		if axis == dragDirectionHorizontal {
			sum += offset.Dx
		} else {
			sum += offset.Dy
		}
	}
	return sum / float64(pointerCount)
}

// dartSign is double.sign: -1.0, 1.0, or the value itself (±0.0, NaN).
func dartSign(v float64) float64 {
	switch {
	case v > 0:
		return 1.0
	case v < 0:
		return -1.0
	}
	return v
}

func (r *DragGestureRecognizerBase) HandleEvent(event PointerEvent) {
	dartAssert(r.state != dragStateReady, nil)
	if !event.Synthesized() {
		var position ui.Offset
		track := true
		switch e := event.(type) {
		case *PointerPanZoomStartEvent:
			position = ui.Offset{}
		case *PointerPanZoomUpdateEvent:
			position = e.Pan()
		case *PointerDownEvent, *PointerMoveEvent:
			position = event.LocalPosition()
		default:
			track = false
		}
		if track {
			r.velocityTrackers[event.Pointer()].AddPosition(event.TimeStamp(), position)
		}
	}
	if move, ok := event.(*PointerMoveEvent); ok && (r.initialButtons == nil || move.Buttons() != *r.initialButtons) {
		r.giveUpPointer(event.Pointer())
		return
	}
	move, isMove := event.(*PointerMoveEvent)
	panZoom, isPanZoomUpdate := event.(*PointerPanZoomUpdateEvent)
	if (isMove || isPanZoomUpdate) && r.shouldTrackMoveEvent(event.Pointer()) {
		var delta, localDelta, position, localPosition ui.Offset
		if isMove {
			delta = move.Delta()
			localDelta = move.LocalDelta()
			position = move.Position()
			localPosition = move.LocalPosition()
		} else {
			delta = panZoom.PanDelta()
			localDelta = panZoom.LocalPanDelta()
			position = event.Position().Add(panZoom.Pan())
			localPosition = event.LocalPosition().Add(panZoom.LocalPan())
		}
		r.lastPosition = OffsetPair{Local: localPosition, Global: position}
		resolvedDelta := r.resolveLocalDeltaForMultitouch(event.Pointer(), localDelta)
		switch r.state {
		case dragStateReady, dragStatePossible:
			r.pendingDragOffset = r.pendingDragOffset.Add(OffsetPair{Local: localDelta, Global: delta})
			ts := event.TimeStamp()
			r.lastPendingEventTimestamp = &ts
			r.lastTransform = event.Transform()
			movedLocally := r.self.getDeltaForDetails(localDelta)
			var localToGlobalTransform *vm.Matrix4
			if event.Transform() != nil {
				localToGlobalTransform = vm.Matrix4TryInvert(event.Transform())
			}
			sign := 1.0
			if v := r.self.getPrimaryValueFromOffset(movedLocally); v != nil {
				sign = *v
			}
			r.globalDistanceMoved += float64(PointerEventTransformDeltaViaPositions(localPosition, nil, movedLocally,
				localToGlobalTransform).Distance() * dartSign(sign))
			var touchSlop *float64
			if r.GestureSettings != nil {
				touchSlop = r.GestureSettings.TouchSlop
			}
			if r.self.HasSufficientGlobalDistanceToAccept(event.Kind(), touchSlop) {
				r.hasDragThresholdBeenMet = true
				if slices.Contains(r.acceptedActivePointers, event.Pointer()) {
					r.checkDrag(event.Pointer())
				} else {
					r.self.Resolve(GestureDispositionAccepted)
				}
			}
		case dragStateAccepted:
			ts := event.TimeStamp()
			r.checkUpdate(&ts, r.self.getDeltaForDetails(resolvedDelta), r.self.getPrimaryValueFromOffset(resolvedDelta),
				position, &localPosition, event.Pointer())
		}
		r.recordMoveDeltaForMultitouch(event.Pointer(), localDelta)
	}
	switch event.(type) {
	case *PointerUpEvent, *PointerCancelEvent, *PointerPanZoomEndEvent:
		r.giveUpPointer(event.Pointer())
	}
}

func (r *DragGestureRecognizerBase) AcceptGesture(pointer int) {
	dartAssert(!slices.Contains(r.acceptedActivePointers, pointer), nil)
	r.acceptedActivePointers = append(r.acceptedActivePointers, pointer)
	p := pointer
	r.activePointer = &p
	if !r.OnlyAcceptDragOnThreshold || r.hasDragThresholdBeenMet {
		r.checkDrag(pointer)
	}
}

func (r *DragGestureRecognizerBase) RejectGesture(pointer int) { r.giveUpPointer(pointer) }

func (r *DragGestureRecognizerBase) DidStopTrackingLastPointer(pointer int) {
	dartAssert(r.state != dragStateReady, nil)
	switch r.state {
	case dragStateReady:
	case dragStatePossible:
		r.self.Resolve(GestureDispositionRejected)
		r.checkCancel()
	case dragStateAccepted:
		r.checkEnd(pointer)
	}
	r.hasDragThresholdBeenMet = false
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.initialButtons = nil
	r.state = dragStateReady
}

func (r *DragGestureRecognizerBase) giveUpPointer(pointer int) {
	r.self.StopTrackingPointer(pointer)
	// If we never accepted the pointer, we reject it since we are no longer
	// interested in winning the gesture arena for it.
	if i := slices.Index(r.acceptedActivePointers, pointer); i >= 0 {
		r.acceptedActivePointers = slices.Delete(r.acceptedActivePointers, i, i+1)
	} else {
		r.self.ResolvePointer(pointer, GestureDispositionRejected)
	}
	if _, ok := r.moveDeltaBeforeFrame[pointer]; ok {
		delete(r.moveDeltaBeforeFrame, pointer)
		r.moveDeltaPointers = slices.DeleteFunc(r.moveDeltaPointers, func(p int) bool { return p == pointer })
	}
	if r.activePointer != nil && *r.activePointer == pointer {
		if len(r.acceptedActivePointers) > 0 {
			p := r.acceptedActivePointers[0]
			r.activePointer = &p
		} else {
			r.activePointer = nil
		}
	}
}

func (r *DragGestureRecognizerBase) checkDown() {
	if r.OnDown != nil {
		local := r.initialPosition.Local
		details := NewDragDownDetails(DragDownDetailsOptions{GlobalPosition: r.initialPosition.Global, LocalPosition: &local})
		InvokeVoidCallback(r.self, "onDown", func() { r.OnDown(details) })
	}
}

func (r *DragGestureRecognizerBase) checkDrag(pointer int) {
	if r.state == dragStateAccepted {
		return
	}
	r.state = dragStateAccepted
	delta := r.pendingDragOffset
	timestamp := r.lastPendingEventTimestamp
	transform := r.lastTransform
	var localUpdateDelta ui.Offset
	switch r.DragStartBehavior {
	case DragStartBehaviorStart:
		r.initialPosition = r.initialPosition.Add(delta)
		localUpdateDelta = ui.Offset{}
	case DragStartBehaviorDown:
		localUpdateDelta = r.self.getDeltaForDetails(delta.Local)
	}
	r.pendingDragOffset = OffsetPairZero
	r.lastPendingEventTimestamp = nil
	r.lastTransform = nil
	r.checkStart(timestamp, pointer)
	if localUpdateDelta != (ui.Offset{}) && r.OnUpdate != nil {
		var localToGlobal *vm.Matrix4
		if transform != nil {
			localToGlobal = vm.Matrix4TryInvert(transform)
		}
		correctedLocalPosition := r.initialPosition.Local.Add(localUpdateDelta)
		globalUpdateDelta := PointerEventTransformDeltaViaPositions(correctedLocalPosition, nil, localUpdateDelta, localToGlobal)
		updateDelta := OffsetPair{Local: localUpdateDelta, Global: globalUpdateDelta}
		correctedPosition := r.initialPosition.Add(updateDelta) // Only adds delta for down behaviour
		local := correctedPosition.Local
		r.checkUpdate(timestamp, localUpdateDelta, r.self.getPrimaryValueFromOffset(localUpdateDelta),
			correctedPosition.Global, &local, pointer)
	}
	// This acceptGesture might have been called only for one pointer,
	// instead of all pointers. Resolve all pointers to `accepted`. This
	// won't cause infinite recursion because an accepted pointer won't be
	// accepted again.
	r.self.Resolve(GestureDispositionAccepted)
}

func (r *DragGestureRecognizerBase) checkStart(timestamp *time.Duration, pointer int) {
	if r.OnStart != nil {
		local := r.initialPosition.Local
		kind := r.self.GetKindForPointer(pointer)
		details := NewDragStartDetails(DragStartDetailsOptions{
			SourceTimeStamp: timestamp, GlobalPosition: r.initialPosition.Global, LocalPosition: &local, Kind: &kind,
		})
		InvokeVoidCallback(r.self, "onStart", func() { r.OnStart(details) })
	}
}

func (r *DragGestureRecognizerBase) checkUpdate(sourceTimeStamp *time.Duration, delta ui.Offset, primaryDelta *float64,
	globalPosition ui.Offset, localPosition *ui.Offset, pointer int) {
	if r.OnUpdate != nil {
		kind := r.self.GetKindForPointer(pointer)
		details := NewDragUpdateDetails(DragUpdateDetailsOptions{
			SourceTimeStamp: sourceTimeStamp, Delta: delta, PrimaryDelta: primaryDelta,
			GlobalPosition: globalPosition, LocalPosition: localPosition, Kind: &kind,
		})
		InvokeVoidCallback(r.self, "onUpdate", func() { r.OnUpdate(details) })
	}
}

func (r *DragGestureRecognizerBase) checkEnd(pointer int) {
	if r.OnEnd == nil {
		return
	}
	tracker := r.velocityTrackers[pointer]
	estimate := tracker.GetVelocityEstimate()
	var details *DragEndDetails
	var debugReport func() string
	if estimate == nil {
		debugReport = func() string { return "Could not estimate velocity." }
	} else {
		details = r.self.ConsiderFling(*estimate, tracker.Kind)
		if details != nil {
			d := details
			debugReport = func() string { return estimate.String() + "; fling at " + d.Velocity.String() + "." }
		} else {
			debugReport = func() string { return estimate.String() + "; judged to not be a fling." }
		}
	}
	if details == nil {
		zero := 0.0
		local := r.lastPosition.Local
		details = NewDragEndDetails(DragEndDetailsOptions{
			PrimaryVelocity: &zero, GlobalPosition: r.lastPosition.Global, LocalPosition: &local,
		})
	}
	InvokeVoidCallback(r.self, "onEnd", func() { r.OnEnd(details) }, debugReport)
}

func (r *DragGestureRecognizerBase) checkCancel() {
	if r.OnCancel != nil {
		InvokeVoidCallback(r.self, "onCancel", r.OnCancel)
	}
}

func (r *DragGestureRecognizerBase) Dispose() {
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.OneSequenceGestureRecognizerBase.Dispose()
}

func (r *DragGestureRecognizerBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	r.OneSequenceGestureRecognizerBase.DebugFillProperties(properties)
	properties.Add(foundation.NewEnumProperty("start behavior", r.DragStartBehavior))
}

func (r *DragGestureRecognizerBase) getPrimaryDragAxis() *dragDirection { return nil }

func (r *DragGestureRecognizerBase) minVelocity() float64 {
	if r.MinFlingVelocity != nil {
		return *r.MinFlingVelocity
	}
	return KMinFlingVelocity
}

func (r *DragGestureRecognizerBase) maxVelocity() float64 {
	if r.MaxFlingVelocity != nil {
		return *r.MaxFlingVelocity
	}
	return KMaxFlingVelocity
}

func (r *DragGestureRecognizerBase) minDistance(kind ui.PointerDeviceKind) float64 {
	if r.MinFlingDistance != nil {
		return *r.MinFlingDistance
	}
	return ComputeHitSlop(kind, r.GestureSettings)
}

func floatp(v float64) *float64 { return &v }

// ---- VerticalDragGestureRecognizer

// VerticalDragGestureRecognizer recognizes movement in the vertical
// direction.
type VerticalDragGestureRecognizer struct {
	DragGestureRecognizerBase
}

// NewVerticalDragGestureRecognizer is VerticalDragGestureRecognizer({debugOwner,
// supportedDevices, allowedButtonsFilter}).
func NewVerticalDragGestureRecognizer(opts ...GestureRecognizerOptions) *VerticalDragGestureRecognizer {
	r := &VerticalDragGestureRecognizer{}
	var o GestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r.Construct(r, DragGestureRecognizerOptions{GestureRecognizerOptions: o})
	return r
}

func (r *VerticalDragGestureRecognizer) IsFlingGesture(estimate VelocityEstimate, kind ui.PointerDeviceKind) bool {
	return math.Abs(estimate.PixelsPerSecond.Dy) > r.minVelocity() && math.Abs(estimate.Offset.Dy) > r.minDistance(kind)
}

func (r *VerticalDragGestureRecognizer) ConsiderFling(estimate VelocityEstimate, kind ui.PointerDeviceKind) *DragEndDetails {
	if !r.self.IsFlingGesture(estimate, kind) {
		return nil
	}
	maxVelocity := r.maxVelocity()
	dy := ui.ClampDouble(estimate.PixelsPerSecond.Dy, -maxVelocity, maxVelocity)
	local := r.LastPosition().Local
	return NewDragEndDetails(DragEndDetailsOptions{
		Velocity: Velocity{PixelsPerSecond: ui.NewOffset(0, dy)}, PrimaryVelocity: floatp(dy),
		GlobalPosition: r.LastPosition().Global, LocalPosition: &local,
	})
}

func (r *VerticalDragGestureRecognizer) HasSufficientGlobalDistanceToAccept(pointerDeviceKind ui.PointerDeviceKind, deviceTouchSlop *float64) bool {
	return math.Abs(r.GlobalDistanceMoved()) > ComputeHitSlop(pointerDeviceKind, r.GestureSettings)
}

func (r *VerticalDragGestureRecognizer) getDeltaForDetails(delta ui.Offset) ui.Offset {
	return ui.NewOffset(0.0, delta.Dy)
}

func (r *VerticalDragGestureRecognizer) getPrimaryValueFromOffset(value ui.Offset) *float64 {
	return floatp(value.Dy)
}

func (r *VerticalDragGestureRecognizer) getPrimaryDragAxis() *dragDirection {
	d := dragDirectionVertical
	return &d
}

func (r *VerticalDragGestureRecognizer) DebugDescription() string { return "vertical drag" }

// ---- HorizontalDragGestureRecognizer

// HorizontalDragGestureRecognizer recognizes movement in the horizontal
// direction.
type HorizontalDragGestureRecognizer struct {
	DragGestureRecognizerBase
}

// NewHorizontalDragGestureRecognizer is HorizontalDragGestureRecognizer({debugOwner,
// supportedDevices, allowedButtonsFilter}).
func NewHorizontalDragGestureRecognizer(opts ...GestureRecognizerOptions) *HorizontalDragGestureRecognizer {
	r := &HorizontalDragGestureRecognizer{}
	var o GestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r.Construct(r, DragGestureRecognizerOptions{GestureRecognizerOptions: o})
	return r
}

func (r *HorizontalDragGestureRecognizer) IsFlingGesture(estimate VelocityEstimate, kind ui.PointerDeviceKind) bool {
	return math.Abs(estimate.PixelsPerSecond.Dx) > r.minVelocity() && math.Abs(estimate.Offset.Dx) > r.minDistance(kind)
}

func (r *HorizontalDragGestureRecognizer) ConsiderFling(estimate VelocityEstimate, kind ui.PointerDeviceKind) *DragEndDetails {
	if !r.self.IsFlingGesture(estimate, kind) {
		return nil
	}
	maxVelocity := r.maxVelocity()
	dx := ui.ClampDouble(estimate.PixelsPerSecond.Dx, -maxVelocity, maxVelocity)
	local := r.lastPosition.Local
	return NewDragEndDetails(DragEndDetailsOptions{
		Velocity: Velocity{PixelsPerSecond: ui.NewOffset(dx, 0)}, PrimaryVelocity: floatp(dx),
		GlobalPosition: r.lastPosition.Global, LocalPosition: &local,
	})
}

func (r *HorizontalDragGestureRecognizer) HasSufficientGlobalDistanceToAccept(pointerDeviceKind ui.PointerDeviceKind, deviceTouchSlop *float64) bool {
	return math.Abs(r.GlobalDistanceMoved()) > ComputeHitSlop(pointerDeviceKind, r.GestureSettings)
}

func (r *HorizontalDragGestureRecognizer) getDeltaForDetails(delta ui.Offset) ui.Offset {
	return ui.NewOffset(delta.Dx, 0.0)
}

func (r *HorizontalDragGestureRecognizer) getPrimaryValueFromOffset(value ui.Offset) *float64 {
	return floatp(value.Dx)
}

func (r *HorizontalDragGestureRecognizer) getPrimaryDragAxis() *dragDirection {
	d := dragDirectionHorizontal
	return &d
}

func (r *HorizontalDragGestureRecognizer) DebugDescription() string { return "horizontal drag" }

// ---- PanGestureRecognizer

// PanGestureRecognizer recognizes movement both horizontally and
// vertically.
type PanGestureRecognizer struct {
	DragGestureRecognizerBase
}

// NewPanGestureRecognizer is PanGestureRecognizer({debugOwner,
// supportedDevices, allowedButtonsFilter}).
func NewPanGestureRecognizer(opts ...GestureRecognizerOptions) *PanGestureRecognizer {
	r := &PanGestureRecognizer{}
	var o GestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r.Construct(r, DragGestureRecognizerOptions{GestureRecognizerOptions: o})
	return r
}

func (r *PanGestureRecognizer) IsFlingGesture(estimate VelocityEstimate, kind ui.PointerDeviceKind) bool {
	minVelocity := r.minVelocity()
	minDistance := r.minDistance(kind)
	return estimate.PixelsPerSecond.DistanceSquared() > float64(minVelocity*minVelocity) &&
		estimate.Offset.DistanceSquared() > float64(minDistance*minDistance)
}

func (r *PanGestureRecognizer) ConsiderFling(estimate VelocityEstimate, kind ui.PointerDeviceKind) *DragEndDetails {
	if !r.self.IsFlingGesture(estimate, kind) {
		return nil
	}
	velocity := Velocity{PixelsPerSecond: estimate.PixelsPerSecond}.ClampMagnitude(r.minVelocity(), r.maxVelocity())
	local := r.lastPosition.Local
	return NewDragEndDetails(DragEndDetailsOptions{
		Velocity: velocity, GlobalPosition: r.lastPosition.Global, LocalPosition: &local,
	})
}

func (r *PanGestureRecognizer) HasSufficientGlobalDistanceToAccept(pointerDeviceKind ui.PointerDeviceKind, deviceTouchSlop *float64) bool {
	return math.Abs(r.GlobalDistanceMoved()) > ComputePanSlop(pointerDeviceKind, r.GestureSettings)
}

func (r *PanGestureRecognizer) getDeltaForDetails(delta ui.Offset) ui.Offset { return delta }

func (r *PanGestureRecognizer) getPrimaryValueFromOffset(value ui.Offset) *float64 { return nil }

func (r *PanGestureRecognizer) DebugDescription() string { return "pan" }
