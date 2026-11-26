package gestures

// Port of multitap.dart (Flutter 3.50.1): DoubleTapGestureRecognizer,
// MultiTapGestureRecognizer and SerialTapGestureRecognizer.
//
// Dart's maps keyed by pointer are iterated in insertion order where it is
// observable (clearing trackers, disposing gestures): the Go ports keep
// the insertion order beside the map.

import (
	"slices"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// GestureDoubleTapCallback is the signature for callback when the user
// has tapped the screen at the same location twice in quick succession.
type GestureDoubleTapCallback func()

// GestureMultiTapDownCallback is the signature used by
// MultiTapGestureRecognizer for when a pointer that might cause a tap has
// contacted the screen at a particular location.
type GestureMultiTapDownCallback func(pointer int, details *TapDownDetails)

// GestureMultiTapUpCallback is the signature used by
// MultiTapGestureRecognizer for when a pointer that will trigger a tap has
// stopped contacting the screen at a particular location.
type GestureMultiTapUpCallback func(pointer int, details *TapUpDetails)

// GestureMultiTapCallback is the signature used by
// MultiTapGestureRecognizer for when a tap has occurred.
type GestureMultiTapCallback func(pointer int)

// GestureMultiTapCancelCallback is the signature for when the pointer that
// previously triggered a GestureMultiTapDownCallback will not end up
// causing a tap.
type GestureMultiTapCancelCallback func(pointer int)

// countdownZoned is _CountdownZoned: a flag set when duration has passed.
type countdownZoned struct{ timeout bool }

func newCountdownZoned(duration time.Duration) *countdownZoned {
	c := &countdownZoned{}
	async.NewTimer(duration, func() { c.timeout = true })
	return c
}

// tapTracker is _TapTracker: the state of a pointer down for the
// multi-tap recognizers.
type tapTracker struct {
	gestureSettings             *DeviceGestureSettings
	pointer                     int
	entry                       GestureArenaEntry
	initialGlobalPosition       ui.Offset
	initialButtons              int
	doubleTapMinTimeCountdown   *countdownZoned
	isTrackingPointer           bool
	stopTrackingPointerOverride func(route PointerRoute) // _TapGesture's override
}

func newTapTracker(event *PointerDownEvent, entry GestureArenaEntry, doubleTapMinTime time.Duration, gestureSettings *DeviceGestureSettings) *tapTracker {
	return &tapTracker{
		gestureSettings:           gestureSettings,
		pointer:                   event.Pointer(),
		entry:                     entry,
		initialGlobalPosition:     event.Position(),
		initialButtons:            event.Buttons(),
		doubleTapMinTimeCountdown: newCountdownZoned(doubleTapMinTime),
	}
}

func (t *tapTracker) startTrackingPointer(route PointerRoute, transform *vm.Matrix4) {
	if !t.isTrackingPointer {
		t.isTrackingPointer = true
		GestureBindingInstance().PointerRouter().AddRoute(t.pointer, route, transform)
	}
}

func (t *tapTracker) stopTrackingPointer(route PointerRoute) {
	if t.stopTrackingPointerOverride != nil {
		t.stopTrackingPointerOverride(route)
		return
	}
	t.stopTrackingPointerBase(route)
}

func (t *tapTracker) stopTrackingPointerBase(route PointerRoute) {
	if t.isTrackingPointer {
		t.isTrackingPointer = false
		GestureBindingInstance().PointerRouter().RemoveRoute(t.pointer, route)
	}
}

func (t *tapTracker) isWithinGlobalTolerance(event PointerEvent, tolerance float64) bool {
	offset := event.Position().Sub(t.initialGlobalPosition)
	return offset.Distance() <= tolerance
}

func (t *tapTracker) hasElapsedMinTime() bool { return t.doubleTapMinTimeCountdown.timeout }

func (t *tapTracker) hasSameButton(event *PointerDownEvent) bool {
	return event.Buttons() == t.initialButtons
}

// primaryButtonOnly is the double tap recognizer's default
// allowedButtonsFilter: the primary button alone.
func primaryButtonOnly(buttons int) bool { return buttons == KPrimaryButton }

// DoubleTapGestureRecognizer recognizes when the user has tapped the
// screen at the same location twice in quick succession.
type DoubleTapGestureRecognizer struct {
	GestureRecognizerBase

	OnDoubleTapDown   GestureTapDownCallback
	OnDoubleTap       GestureDoubleTapCallback
	OnDoubleTapCancel GestureTapCancelCallback

	doubleTapTimer   *async.Timer
	firstTap         *tapTracker
	trackers         map[int]*tapTracker
	trackerOrder     []int
	handleEventRoute PointerRoute
}

// NewDoubleTapGestureRecognizer creates a double tap recognizer (a nil
// AllowedButtonsFilter accepts the primary button only).
func NewDoubleTapGestureRecognizer(opts ...GestureRecognizerOptions) *DoubleTapGestureRecognizer {
	var o GestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.AllowedButtonsFilter == nil {
		o.AllowedButtonsFilter = primaryButtonOnly
	}
	r := &DoubleTapGestureRecognizer{trackers: map[int]*tapTracker{}}
	r.Construct(r, o)
	r.handleEventRoute = r.handleEvent
	return r
}

func (r *DoubleTapGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	if r.firstTap == nil {
		if r.OnDoubleTapDown == nil && r.OnDoubleTap == nil && r.OnDoubleTapCancel == nil {
			return false
		}
	}
	// If second tap is not allowed, reset the state.
	isPointerAllowed := r.GestureRecognizerBase.IsPointerAllowed(event)
	if !isPointerAllowed {
		r.reset()
	}
	return isPointerAllowed
}

func (r *DoubleTapGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	if r.firstTap != nil {
		if !r.firstTap.isWithinGlobalTolerance(event, KDoubleTapSlop) {
			// Ignore out-of-bounds second taps.
			return
		} else if !r.firstTap.hasElapsedMinTime() || !r.firstTap.hasSameButton(event) {
			// Restart when the second tap is too close to the first (touch screens
			// often detect touches intermittently), or when buttons mismatch.
			r.reset()
			r.trackTap(event)
			return
		} else if r.OnDoubleTapDown != nil {
			kind := r.GetKindForPointer(event.Pointer())
			local := event.LocalPosition()
			details := NewTapDownDetails(TapDownDetailsOptions{GlobalPosition: event.Position(), LocalPosition: &local, Kind: &kind})
			InvokeVoidCallback(r, "onDoubleTapDown", func() { r.OnDoubleTapDown(details) })
		}
	}
	r.trackTap(event)
}

func (r *DoubleTapGestureRecognizer) setTracker(t *tapTracker) {
	if _, ok := r.trackers[t.pointer]; !ok {
		r.trackerOrder = append(r.trackerOrder, t.pointer)
	}
	r.trackers[t.pointer] = t
}

func (r *DoubleTapGestureRecognizer) removeTracker(pointer int) {
	if _, ok := r.trackers[pointer]; ok {
		delete(r.trackers, pointer)
		r.trackerOrder = slices.DeleteFunc(r.trackerOrder, func(p int) bool { return p == pointer })
	}
}

func (r *DoubleTapGestureRecognizer) trackTap(event *PointerDownEvent) {
	r.stopDoubleTapTimer()
	tracker := newTapTracker(event, GestureBindingInstance().GestureArena().Add(event.Pointer(), r),
		KDoubleTapMinTime, r.GestureSettings)
	r.setTracker(tracker)
	tracker.startTrackingPointer(r.handleEventRoute, event.Transform())
}

func (r *DoubleTapGestureRecognizer) handleEvent(event PointerEvent) {
	tracker := r.trackers[event.Pointer()]
	switch event.(type) {
	case *PointerUpEvent:
		if r.firstTap == nil {
			r.registerFirstTap(tracker)
		} else {
			r.registerSecondTap(tracker)
		}
	case *PointerMoveEvent:
		if !tracker.isWithinGlobalTolerance(event, KDoubleTapTouchSlop) {
			r.reject(tracker)
		}
	case *PointerCancelEvent:
		r.reject(tracker)
	}
}

func (r *DoubleTapGestureRecognizer) AcceptGesture(pointer int) {}

func (r *DoubleTapGestureRecognizer) RejectGesture(pointer int) {
	tracker := r.trackers[pointer]
	// If tracker isn't in the list, check if this is the first tap tracker
	if tracker == nil && r.firstTap != nil && r.firstTap.pointer == pointer {
		tracker = r.firstTap
	}
	// If tracker is still null, we rejected ourselves already
	if tracker != nil {
		r.reject(tracker)
	}
}

func (r *DoubleTapGestureRecognizer) reject(tracker *tapTracker) {
	r.removeTracker(tracker.pointer)
	tracker.entry.Resolve(GestureDispositionRejected)
	r.freezeTracker(tracker)
	if r.firstTap != nil {
		if tracker == r.firstTap {
			r.reset()
		} else {
			r.checkCancel()
			if len(r.trackers) == 0 {
				r.reset()
			}
		}
	}
}

func (r *DoubleTapGestureRecognizer) Dispose() {
	r.reset()
	r.GestureRecognizerBase.Dispose()
}

func (r *DoubleTapGestureRecognizer) reset() {
	r.stopDoubleTapTimer()
	if r.firstTap != nil {
		if len(r.trackers) > 0 {
			r.checkCancel()
		}
		// Note, order is important below in order for the resolve -> reject logic
		// to work properly.
		tracker := r.firstTap
		r.firstTap = nil
		r.reject(tracker)
		GestureBindingInstance().GestureArena().Release(tracker.pointer)
	}
	r.clearTrackers()
}

func (r *DoubleTapGestureRecognizer) registerFirstTap(tracker *tapTracker) {
	r.startDoubleTapTimer()
	GestureBindingInstance().GestureArena().Hold(tracker.pointer)
	// Note, order is important below in order for the clear -> reject logic to
	// work properly.
	r.freezeTracker(tracker)
	r.removeTracker(tracker.pointer)
	r.clearTrackers()
	r.firstTap = tracker
}

func (r *DoubleTapGestureRecognizer) registerSecondTap(tracker *tapTracker) {
	r.firstTap.entry.Resolve(GestureDispositionAccepted)
	tracker.entry.Resolve(GestureDispositionAccepted)
	r.freezeTracker(tracker)
	r.removeTracker(tracker.pointer)
	r.checkUp(tracker.initialButtons)
	r.reset()
}

func (r *DoubleTapGestureRecognizer) clearTrackers() {
	values := make([]*tapTracker, 0, len(r.trackerOrder))
	for _, p := range r.trackerOrder {
		values = append(values, r.trackers[p])
	}
	for _, t := range values {
		r.reject(t)
	}
	dartAssert(len(r.trackers) == 0, nil)
}

func (r *DoubleTapGestureRecognizer) freezeTracker(tracker *tapTracker) {
	tracker.stopTrackingPointer(r.handleEventRoute)
}

func (r *DoubleTapGestureRecognizer) startDoubleTapTimer() {
	if r.doubleTapTimer == nil {
		r.doubleTapTimer = async.NewTimer(KDoubleTapTimeout, r.reset)
	}
}

func (r *DoubleTapGestureRecognizer) stopDoubleTapTimer() {
	if r.doubleTapTimer != nil {
		r.doubleTapTimer.Cancel()
		r.doubleTapTimer = nil
	}
}

func (r *DoubleTapGestureRecognizer) checkUp(buttons int) {
	if r.OnDoubleTap != nil {
		InvokeVoidCallback(r, "onDoubleTap", r.OnDoubleTap)
	}
}

func (r *DoubleTapGestureRecognizer) checkCancel() {
	if r.OnDoubleTapCancel != nil {
		InvokeVoidCallback(r, "onDoubleTapCancel", r.OnDoubleTapCancel)
	}
}

func (r *DoubleTapGestureRecognizer) DebugDescription() string { return "double tap" }

// tapGesture is _TapGesture: one pointer of a MultiTapGestureRecognizer.
type tapGesture struct {
	*tapTracker
	gestureRecognizer *MultiTapGestureRecognizer
	wonArena          bool
	timer             *async.Timer
	lastPosition      OffsetPair
	finalPosition     *OffsetPair
	handleEventRoute  PointerRoute
}

func newTapGesture(gestureRecognizer *MultiTapGestureRecognizer, event *PointerDownEvent, longTapDelay time.Duration, gestureSettings *DeviceGestureSettings) *tapGesture {
	g := &tapGesture{gestureRecognizer: gestureRecognizer, lastPosition: NewOffsetPairFromEventPosition(event)}
	g.tapTracker = newTapTracker(event, GestureBindingInstance().GestureArena().Add(event.Pointer(), gestureRecognizer),
		KDoubleTapMinTime, gestureSettings)
	g.tapTracker.stopTrackingPointerOverride = g.stopTrackingPointer
	g.handleEventRoute = g.handleEvent
	g.startTrackingPointer(g.handleEventRoute, event.Transform())
	if longTapDelay > 0 {
		g.timer = async.NewTimer(longTapDelay, func() {
			g.timer = nil
			gestureRecognizer.dispatchLongTap(event.Pointer(), g.lastPosition)
		})
	}
	return g
}

func (g *tapGesture) handleEvent(event PointerEvent) {
	dartAssert(event.Pointer() == g.pointer, nil)
	switch event.(type) {
	case *PointerMoveEvent:
		if !g.isWithinGlobalTolerance(event, ComputeHitSlop(event.Kind(), g.gestureSettings)) {
			g.cancel()
		} else {
			g.lastPosition = NewOffsetPairFromEventPosition(event)
		}
	case *PointerCancelEvent:
		g.cancel()
	case *PointerUpEvent:
		g.stopTrackingPointer(g.handleEventRoute)
		final := NewOffsetPairFromEventPosition(event)
		g.finalPosition = &final
		g.check()
	}
}

func (g *tapGesture) stopTrackingPointer(route PointerRoute) {
	if g.timer != nil {
		g.timer.Cancel()
	}
	g.timer = nil
	g.tapTracker.stopTrackingPointerBase(route)
}

func (g *tapGesture) accept() {
	g.wonArena = true
	g.check()
}

func (g *tapGesture) reject() {
	g.stopTrackingPointer(g.handleEventRoute)
	g.gestureRecognizer.dispatchCancel(g.pointer)
}

func (g *tapGesture) cancel() {
	// If we won the arena already, then entry is resolved, so resolving
	// again is a no-op. But we still need to clean up our own state.
	if g.wonArena {
		g.reject()
	} else {
		g.entry.Resolve(GestureDispositionRejected) // eventually calls reject()
	}
}

func (g *tapGesture) check() {
	if g.wonArena && g.finalPosition != nil {
		g.gestureRecognizer.dispatchTap(g.pointer, *g.finalPosition)
	}
}

// MultiTapGestureRecognizerOptions are MultiTapGestureRecognizer's
// constructor parameters (LongTapDelay zero: no long taps).
type MultiTapGestureRecognizerOptions struct {
	GestureRecognizerOptions
	LongTapDelay time.Duration
}

// MultiTapGestureRecognizer recognizes taps of several pointers at once,
// each one tracked separately.
type MultiTapGestureRecognizer struct {
	GestureRecognizerBase

	OnTapDown     GestureMultiTapDownCallback
	OnTapUp       GestureMultiTapUpCallback
	OnTap         GestureMultiTapCallback
	OnTapCancel   GestureMultiTapCancelCallback
	OnLongTapDown GestureMultiTapDownCallback
	// LongTapDelay is the time a pointer must be down before OnLongTapDown
	// is called (zero: never).
	LongTapDelay time.Duration

	gestureMap   map[int]*tapGesture
	gestureOrder []int
}

// NewMultiTapGestureRecognizer creates a multi-tap recognizer.
func NewMultiTapGestureRecognizer(opts ...MultiTapGestureRecognizerOptions) *MultiTapGestureRecognizer {
	var o MultiTapGestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r := &MultiTapGestureRecognizer{LongTapDelay: o.LongTapDelay, gestureMap: map[int]*tapGesture{}}
	r.Construct(r, o.GestureRecognizerOptions)
	return r
}

func (r *MultiTapGestureRecognizer) removeGesture(pointer int) {
	delete(r.gestureMap, pointer)
	r.gestureOrder = slices.DeleteFunc(r.gestureOrder, func(p int) bool { return p == pointer })
}

func (r *MultiTapGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	_, exists := r.gestureMap[event.Pointer()]
	dartAssert(!exists, nil)
	g := newTapGesture(r, event, r.LongTapDelay, r.GestureSettings)
	r.gestureMap[event.Pointer()] = g
	r.gestureOrder = append(r.gestureOrder, event.Pointer())
	if r.OnTapDown != nil {
		InvokeVoidCallback(r, "onTapDown", func() {
			kind := event.Kind()
			local := event.LocalPosition()
			r.OnTapDown(event.Pointer(), NewTapDownDetails(TapDownDetailsOptions{
				GlobalPosition: event.Position(), LocalPosition: &local, Kind: &kind}))
		})
	}
}

func (r *MultiTapGestureRecognizer) AcceptGesture(pointer int) {
	g, ok := r.gestureMap[pointer]
	dartAssert(ok, nil)
	g.accept()
}

func (r *MultiTapGestureRecognizer) RejectGesture(pointer int) {
	g, ok := r.gestureMap[pointer]
	dartAssert(ok, nil)
	g.reject()
	_, still := r.gestureMap[pointer]
	dartAssert(!still, nil)
}

func (r *MultiTapGestureRecognizer) dispatchCancel(pointer int) {
	_, ok := r.gestureMap[pointer]
	dartAssert(ok, nil)
	r.removeGesture(pointer)
	if r.OnTapCancel != nil {
		InvokeVoidCallback(r, "onTapCancel", func() { r.OnTapCancel(pointer) })
	}
}

func (r *MultiTapGestureRecognizer) dispatchTap(pointer int, position OffsetPair) {
	_, ok := r.gestureMap[pointer]
	dartAssert(ok, nil)
	r.removeGesture(pointer)
	if r.OnTapUp != nil {
		InvokeVoidCallback(r, "onTapUp", func() {
			local := position.Local
			r.OnTapUp(pointer, NewTapUpDetails(TapUpDetailsOptions{
				Kind: r.GetKindForPointer(pointer), LocalPosition: &local, GlobalPosition: position.Global}))
		})
	}
	if r.OnTap != nil {
		InvokeVoidCallback(r, "onTap", func() { r.OnTap(pointer) })
	}
}

func (r *MultiTapGestureRecognizer) dispatchLongTap(pointer int, lastPosition OffsetPair) {
	_, ok := r.gestureMap[pointer]
	dartAssert(ok, nil)
	if r.OnLongTapDown != nil {
		InvokeVoidCallback(r, "onLongTapDown", func() {
			kind := r.GetKindForPointer(pointer)
			local := lastPosition.Local
			r.OnLongTapDown(pointer, NewTapDownDetails(TapDownDetailsOptions{
				GlobalPosition: lastPosition.Global, LocalPosition: &local, Kind: &kind}))
		})
	}
}

func (r *MultiTapGestureRecognizer) Dispose() {
	localGestures := make([]*tapGesture, 0, len(r.gestureOrder))
	for _, p := range r.gestureOrder {
		localGestures = append(localGestures, r.gestureMap[p])
	}
	for _, g := range localGestures {
		g.cancel()
	}
	// Rejection of each gesture should cause it to be removed from our map
	dartAssert(len(r.gestureMap) == 0, nil)
	r.GestureRecognizerBase.Dispose()
}

func (r *MultiTapGestureRecognizer) DebugDescription() string { return "multitap" }

// GestureSerialTapDownCallback is the signature for
// SerialTapGestureRecognizer.OnSerialTapDown.
type GestureSerialTapDownCallback func(details *SerialTapDownDetails)

// SerialTapDownDetailsOptions are SerialTapDownDetails' parameters (Kind
// is required; Count 0: 1).
type SerialTapDownDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
	Kind           ui.PointerDeviceKind
	Buttons        int
	Count          int
}

// SerialTapDownDetails are the details for GestureSerialTapDownCallback.
type SerialTapDownDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Kind is the kind of the device that initiated the event.
	Kind ui.PointerDeviceKind
	// Buttons is the buttons pressed when the pointer went down.
	Buttons int
	// Count is the number of consecutive taps that this "tap down"
	// represents.
	Count int
}

// NewSerialTapDownDetails creates details for GestureSerialTapDownCallback.
func NewSerialTapDownDetails(opts SerialTapDownDetailsOptions) *SerialTapDownDetails {
	count := opts.Count
	if count == 0 {
		count = 1
	}
	dartAssert(count > 0, nil)
	d := &SerialTapDownDetails{positionedDetails: newPositionedDetails(opts.GlobalPosition, opts.LocalPosition),
		Kind: opts.Kind, Buttons: opts.Buttons, Count: count}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the details' fields.
func (d *SerialTapDownDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewEnumProperty("kind", d.Kind))
	properties.Add(foundation.NewIntProperty("buttons", d.Buttons))
	properties.Add(foundation.NewIntProperty("count", d.Count))
}

// GestureSerialTapCancelCallback is the signature for
// SerialTapGestureRecognizer.OnSerialTapCancel.
type GestureSerialTapCancelCallback func(details *SerialTapCancelDetails)

// SerialTapCancelDetails are the details for
// GestureSerialTapCancelCallback.
type SerialTapCancelDetails struct {
	foundation.DiagnosticableMixin
	// Count is the number of consecutive taps that were in progress when
	// the gesture was interrupted.
	Count int
}

// NewSerialTapCancelDetails creates details for
// GestureSerialTapCancelCallback (count 0 or omitted: 1).
func NewSerialTapCancelDetails(count ...int) *SerialTapCancelDetails {
	c := 1
	if len(count) > 0 && count[0] != 0 {
		c = count[0]
	}
	dartAssert(c > 0, nil)
	d := &SerialTapCancelDetails{Count: c}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the count.
func (d *SerialTapCancelDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewIntProperty("count", d.Count))
}

// GestureSerialTapUpCallback is the signature for
// SerialTapGestureRecognizer.OnSerialTapUp.
type GestureSerialTapUpCallback func(details *SerialTapUpDetails)

// SerialTapUpDetailsOptions are SerialTapUpDetails' parameters (Kind nil:
// null; Count 0: 1).
type SerialTapUpDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
	Kind           *ui.PointerDeviceKind
	Count          int
}

// SerialTapUpDetails are the details for GestureSerialTapUpCallback.
type SerialTapUpDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Kind is the kind of the device that initiated the event, or nil.
	Kind *ui.PointerDeviceKind
	// Count is the number of consecutive taps that this "tap up"
	// represents.
	Count int
}

// NewSerialTapUpDetails creates details for GestureSerialTapUpCallback.
func NewSerialTapUpDetails(opts SerialTapUpDetailsOptions) *SerialTapUpDetails {
	count := opts.Count
	if count == 0 {
		count = 1
	}
	dartAssert(count > 0, nil)
	d := &SerialTapUpDetails{positionedDetails: newPositionedDetails(opts.GlobalPosition, opts.LocalPosition),
		Kind: opts.Kind, Count: count}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the details' fields.
func (d *SerialTapUpDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewEnumProperty("kind", d.Kind))
	properties.Add(foundation.NewIntProperty("count", d.Count))
}

// SerialTapGestureRecognizer recognizes serial taps (taps in a series).
type SerialTapGestureRecognizer struct {
	GestureRecognizerBase

	OnSerialTapDown   GestureSerialTapDownCallback
	OnSerialTapCancel GestureSerialTapCancelCallback
	OnSerialTapUp     GestureSerialTapUpCallback

	serialTapTimer     *async.Timer
	completedTaps      []*tapTracker
	gestureResolutions map[int]GestureDisposition
	pendingTap         *tapTracker
	handleEventRoute   PointerRoute
}

// NewSerialTapGestureRecognizer creates a serial tap recognizer.
func NewSerialTapGestureRecognizer(opts ...GestureRecognizerOptions) *SerialTapGestureRecognizer {
	var o GestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r := &SerialTapGestureRecognizer{gestureResolutions: map[int]GestureDisposition{}}
	r.Construct(r, o)
	r.handleEventRoute = r.handleEvent
	return r
}

// IsTrackingPointer reports whether a pointer is being tracked.
func (r *SerialTapGestureRecognizer) IsTrackingPointer() bool { return r.pendingTap != nil }

func (r *SerialTapGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	if r.OnSerialTapDown == nil && r.OnSerialTapCancel == nil && r.OnSerialTapUp == nil {
		return false
	}
	return r.GestureRecognizerBase.IsPointerAllowed(event)
}

func (r *SerialTapGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	if (len(r.completedTaps) > 0 && !r.representsSameSeries(r.completedTaps[len(r.completedTaps)-1], event)) ||
		r.pendingTap != nil {
		r.reset()
	}
	r.trackTap(event)
}

func (r *SerialTapGestureRecognizer) representsSameSeries(tap *tapTracker, event *PointerDownEvent) bool {
	return tap.hasElapsedMinTime() && // touch screens often detect touches intermittently
		tap.hasSameButton(event) &&
		tap.isWithinGlobalTolerance(event, KDoubleTapSlop)
}

func (r *SerialTapGestureRecognizer) trackTap(event *PointerDownEvent) {
	r.stopSerialTapTimer()
	if r.OnSerialTapDown != nil {
		local := event.LocalPosition()
		details := NewSerialTapDownDetails(SerialTapDownDetailsOptions{
			GlobalPosition: event.Position(), LocalPosition: &local, Kind: r.GetKindForPointer(event.Pointer()),
			Buttons: event.Buttons(), Count: len(r.completedTaps) + 1,
		})
		InvokeVoidCallback(r, "onSerialTapDown", func() { r.OnSerialTapDown(details) })
	}
	tracker := newTapTracker(event, GestureBindingInstance().GestureArena().Add(event.Pointer(), r),
		KDoubleTapMinTime, r.GestureSettings)
	dartAssert(r.pendingTap == nil, nil)
	r.pendingTap = tracker
	tracker.startTrackingPointer(r.handleEventRoute, event.Transform())
}

func (r *SerialTapGestureRecognizer) handleEvent(event PointerEvent) {
	dartAssert(r.pendingTap != nil, nil)
	dartAssert(r.pendingTap.pointer == event.Pointer(), nil)
	tracker := r.pendingTap
	switch e := event.(type) {
	case *PointerUpEvent:
		r.registerTap(e, tracker)
	case *PointerMoveEvent:
		if !tracker.isWithinGlobalTolerance(event, KDoubleTapTouchSlop) {
			r.reset()
		}
	case *PointerCancelEvent:
		r.reset()
	}
}

func (r *SerialTapGestureRecognizer) AcceptGesture(pointer int) {
	dartAssert(r.pendingTap != nil, nil)
	dartAssert(r.pendingTap.pointer == pointer, nil)
	r.gestureResolutions[pointer] = GestureDispositionAccepted
}

func (r *SerialTapGestureRecognizer) RejectGesture(pointer int) {
	r.gestureResolutions[pointer] = GestureDispositionRejected
	r.reset()
}

func (r *SerialTapGestureRecognizer) rejectPendingTap() {
	dartAssert(r.pendingTap != nil, nil)
	tracker := r.pendingTap
	r.pendingTap = nil
	// Order is important here; the `resolve` call can yield a re-entrant
	// `reset()`, so we need to check cancel here while we can trust the
	// length of our _completedTaps list.
	r.checkCancel(len(r.completedTaps) + 1)
	if _, ok := r.gestureResolutions[tracker.pointer]; !ok {
		tracker.entry.Resolve(GestureDispositionRejected)
	}
	r.stopTrackingPointer(tracker)
}

func (r *SerialTapGestureRecognizer) Dispose() {
	r.reset()
	r.GestureRecognizerBase.Dispose()
}

func (r *SerialTapGestureRecognizer) reset() {
	if r.pendingTap != nil {
		r.rejectPendingTap()
	}
	r.pendingTap = nil
	r.completedTaps = nil
	r.gestureResolutions = map[int]GestureDisposition{}
	r.stopSerialTapTimer()
}

func (r *SerialTapGestureRecognizer) registerTap(event *PointerUpEvent, tracker *tapTracker) {
	dartAssert(tracker == r.pendingTap, nil)
	dartAssert(tracker.pointer == event.Pointer(), nil)
	r.startSerialTapTimer()
	if d, ok := r.gestureResolutions[event.Pointer()]; ok {
		dartAssert(d != GestureDispositionRejected, nil)
	}
	if _, ok := r.gestureResolutions[event.Pointer()]; !ok {
		tracker.entry.Resolve(GestureDispositionAccepted)
	}
	dartAssert(r.gestureResolutions[event.Pointer()] == GestureDispositionAccepted, nil)
	r.stopTrackingPointer(tracker)
	// Note, order is important below in order for the clear -> reject logic to
	// work properly.
	r.pendingTap = nil
	r.checkUp(event, tracker)
	r.completedTaps = append(r.completedTaps, tracker)
}

func (r *SerialTapGestureRecognizer) stopTrackingPointer(tracker *tapTracker) {
	tracker.stopTrackingPointer(r.handleEventRoute)
}

func (r *SerialTapGestureRecognizer) startSerialTapTimer() {
	if r.serialTapTimer == nil {
		r.serialTapTimer = async.NewTimer(KDoubleTapTimeout, r.reset)
	}
}

func (r *SerialTapGestureRecognizer) stopSerialTapTimer() {
	if r.serialTapTimer != nil {
		r.serialTapTimer.Cancel()
		r.serialTapTimer = nil
	}
}

func (r *SerialTapGestureRecognizer) checkUp(event *PointerUpEvent, tracker *tapTracker) {
	if r.OnSerialTapUp != nil {
		kind := r.GetKindForPointer(tracker.pointer)
		local := event.LocalPosition()
		details := NewSerialTapUpDetails(SerialTapUpDetailsOptions{
			GlobalPosition: event.Position(), LocalPosition: &local, Kind: &kind, Count: len(r.completedTaps) + 1,
		})
		InvokeVoidCallback(r, "onSerialTapUp", func() { r.OnSerialTapUp(details) })
	}
}

func (r *SerialTapGestureRecognizer) checkCancel(count int) {
	if r.OnSerialTapCancel != nil {
		details := NewSerialTapCancelDetails(count)
		InvokeVoidCallback(r, "onSerialTapCancel", func() { r.OnSerialTapCancel(details) })
	}
}

func (r *SerialTapGestureRecognizer) DebugDescription() string { return "serial tap" }
