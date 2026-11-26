package gestures

// Port of scale.dart (Flutter 3.50.1): ScaleGestureRecognizer and its
// details.
//
// The recognizer's Map<int, ...> fields are LinkedHashMaps whose iteration
// order decides the order of floating-point sums (focal point, spans), so
// they are kept in insertion order (linkedMap). Math is dart:math's
// (dart.Atan2, dart.Exp: libm).

import (
	"math"
	"slices"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// KDefaultMouseScrollToScaleFactor is the default conversion factor when
// treating mouse scrolling as scaling (kDefaultMouseScrollToScaleFactor).
const KDefaultMouseScrollToScaleFactor = 200.0

// KDefaultTrackpadScrollToScaleFactor is the default conversion factor
// when treating trackpad scrolling as scaling
// (kDefaultTrackpadScrollToScaleFactor).
var KDefaultTrackpadScrollToScaleFactor = ui.NewOffset(0, -1/KDefaultMouseScrollToScaleFactor)

// scaleState is _ScaleState.
type scaleState int

const (
	scaleStateReady scaleState = iota
	scaleStatePossible
	scaleStateAccepted
	scaleStateStarted
)

// linkedMap is a Dart LinkedHashMap<int, V>: iteration in insertion order;
// assigning an existing key keeps its place.
type linkedMap[V any] struct {
	keys   []int
	values map[int]V
}

func newLinkedMap[V any]() *linkedMap[V] { return &linkedMap[V]{values: map[int]V{}} }

func (m *linkedMap[V]) set(k int, v V) {
	if _, ok := m.values[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.values[k] = v
}

func (m *linkedMap[V]) get(k int) (V, bool) { v, ok := m.values[k]; return v, ok }

func (m *linkedMap[V]) remove(k int) {
	if _, ok := m.values[k]; !ok {
		return
	}
	delete(m.values, k)
	m.keys = slices.DeleteFunc(m.keys, func(x int) bool { return x == k })
}

func (m *linkedMap[V]) len() int { return len(m.keys) }

func (m *linkedMap[V]) clear() {
	m.keys = nil
	m.values = map[int]V{}
}

// pointerPanZoomData is _PointerPanZoomData.
type pointerPanZoomData struct {
	parent   *ScaleGestureRecognizer
	position ui.Offset
	pan      ui.Offset
	scale    float64
	rotation float64
}

func pointerPanZoomDataFromStartEvent(parent *ScaleGestureRecognizer, event *PointerPanZoomStartEvent) *pointerPanZoomData {
	return &pointerPanZoomData{parent: parent, position: event.Position(), scale: 1}
}

func pointerPanZoomDataFromUpdateEvent(parent *ScaleGestureRecognizer, event *PointerPanZoomUpdateEvent) *pointerPanZoomData {
	return &pointerPanZoomData{parent: parent, position: event.Position(), pan: event.Pan(),
		scale: event.Scale(), rotation: event.Rotation()}
}

func (p *pointerPanZoomData) focalPoint() ui.Offset {
	if p.parent.TrackpadScrollCausesScale {
		return p.position
	}
	return p.position.Add(p.pan)
}

func (p *pointerPanZoomData) scaleValue() float64 {
	if p.parent.TrackpadScrollCausesScale {
		f := p.parent.TrackpadScrollToScaleFactor
		return p.scale * dart.Exp(float64(p.pan.Dx*f.Dx)+float64(p.pan.Dy*f.Dy))
	}
	return p.scale
}

func (p *pointerPanZoomData) String() string {
	return "_PointerPanZoomData(parent: " + dart.ToString(p.parent) + ", _position: " + p.position.String() +
		", _pan: " + p.pan.String() + ", _scale: " + dart.DoubleToString(p.scale) +
		", _rotation: " + dart.DoubleToString(p.rotation) + ")"
}

// ScaleStartDetailsOptions are ScaleStartDetails' parameters
// (LocalFocalPoint nil: FocalPoint; nil SourceTimeStamp/Kind: null).
type ScaleStartDetailsOptions struct {
	FocalPoint      ui.Offset
	LocalFocalPoint *ui.Offset
	PointerCount    int
	SourceTimeStamp *time.Duration
	Kind            *ui.PointerDeviceKind
}

// ScaleStartDetails are the details for GestureScaleStartCallback.
type ScaleStartDetails struct {
	foundation.DiagnosticableMixin
	// FocalPoint is the initial focal point of the pointers in contact
	// with the screen, in global coordinates.
	FocalPoint ui.Offset
	// LocalFocalPoint is FocalPoint in the receiver's coordinates.
	LocalFocalPoint ui.Offset
	// PointerCount is the number of pointers being tracked.
	PointerCount int
	// SourceTimeStamp is the time stamp of the event that started the
	// scale (nil: null).
	SourceTimeStamp *time.Duration
	// Kind is the kind of the device that initiated the event (nil:
	// null).
	Kind *ui.PointerDeviceKind
}

// NewScaleStartDetails creates details for a GestureScaleStartCallback.
func NewScaleStartDetails(opts ...ScaleStartDetailsOptions) *ScaleStartDetails {
	var o ScaleStartDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &ScaleStartDetails{FocalPoint: o.FocalPoint, LocalFocalPoint: o.FocalPoint, PointerCount: o.PointerCount,
		SourceTimeStamp: o.SourceTimeStamp, Kind: o.Kind}
	if o.LocalFocalPoint != nil {
		d.LocalFocalPoint = *o.LocalFocalPoint
	}
	d.SetSelf(d)
	return d
}

func (d *ScaleStartDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("focalPoint", d.FocalPoint))
	properties.Add(foundation.NewDiagnosticsProperty("localFocalPoint", d.LocalFocalPoint))
	properties.Add(foundation.NewIntProperty("pointerCount", d.PointerCount))
	properties.Add(sourceTimeStampProperty(d.SourceTimeStamp))
}

// ScaleUpdateDetailsOptions are ScaleUpdateDetails' parameters (nil
// scales: 1.0; LocalFocalPoint nil: FocalPoint).
type ScaleUpdateDetailsOptions struct {
	FocalPoint      ui.Offset
	LocalFocalPoint *ui.Offset
	Scale           *float64
	HorizontalScale *float64
	VerticalScale   *float64
	Rotation        float64
	PointerCount    int
	FocalPointDelta ui.Offset
	SourceTimeStamp *time.Duration
}

// ScaleUpdateDetails are the details for GestureScaleUpdateCallback.
type ScaleUpdateDetails struct {
	foundation.DiagnosticableMixin
	// FocalPointDelta is the amount the gesture's focal point has moved
	// in the receiver's coordinates since the previous update.
	FocalPointDelta ui.Offset
	// FocalPoint is the focal point in global coordinates.
	FocalPoint ui.Offset
	// LocalFocalPoint is FocalPoint in the receiver's coordinates.
	LocalFocalPoint ui.Offset
	// Scale is the scale implied by the average distance between the
	// pointers and the focal point.
	Scale float64
	// HorizontalScale is the scale along the horizontal axis.
	HorizontalScale float64
	// VerticalScale is the scale along the vertical axis.
	VerticalScale float64
	// Rotation is the angle (radians) implied by the first two pointers.
	Rotation float64
	// PointerCount is the number of pointers being tracked.
	PointerCount int
	// SourceTimeStamp is the time stamp of the event (nil: null).
	SourceTimeStamp *time.Duration
}

// NewScaleUpdateDetails creates details for a GestureScaleUpdateCallback.
func NewScaleUpdateDetails(opts ...ScaleUpdateDetailsOptions) *ScaleUpdateDetails {
	var o ScaleUpdateDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	orOne := func(p *float64) float64 {
		if p == nil {
			return 1.0
		}
		return *p
	}
	d := &ScaleUpdateDetails{FocalPointDelta: o.FocalPointDelta, FocalPoint: o.FocalPoint, LocalFocalPoint: o.FocalPoint,
		Scale: orOne(o.Scale), HorizontalScale: orOne(o.HorizontalScale), VerticalScale: orOne(o.VerticalScale),
		Rotation: o.Rotation, PointerCount: o.PointerCount, SourceTimeStamp: o.SourceTimeStamp}
	dartAssert(d.Scale >= 0.0, nil)
	dartAssert(d.HorizontalScale >= 0.0, nil)
	dartAssert(d.VerticalScale >= 0.0, nil)
	if o.LocalFocalPoint != nil {
		d.LocalFocalPoint = *o.LocalFocalPoint
	}
	d.SetSelf(d)
	return d
}

func (d *ScaleUpdateDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("focalPointDelta", d.FocalPointDelta))
	properties.Add(foundation.NewDiagnosticsProperty("focalPoint", d.FocalPoint))
	properties.Add(foundation.NewDiagnosticsProperty("localFocalPoint", d.LocalFocalPoint))
	properties.Add(foundation.NewDoubleProperty("scale", d.Scale))
	properties.Add(foundation.NewDoubleProperty("horizontalScale", d.HorizontalScale))
	properties.Add(foundation.NewDoubleProperty("verticalScale", d.VerticalScale))
	properties.Add(foundation.NewDoubleProperty("rotation", d.Rotation))
	properties.Add(foundation.NewIntProperty("pointerCount", d.PointerCount))
	properties.Add(sourceTimeStampProperty(d.SourceTimeStamp))
}

// ScaleEndDetailsOptions are ScaleEndDetails' parameters (Velocity
// defaults to VelocityZero).
type ScaleEndDetailsOptions struct {
	Velocity      Velocity
	ScaleVelocity float64
	PointerCount  int
}

// ScaleEndDetails are the details for GestureScaleEndCallback.
type ScaleEndDetails struct {
	foundation.DiagnosticableMixin
	// Velocity is the velocity of the last pointer to be lifted off the
	// screen.
	Velocity Velocity
	// ScaleVelocity is the final velocity of the scale factor reported by
	// the gesture.
	ScaleVelocity float64
	// PointerCount is the number of pointers being tracked.
	PointerCount int
}

// NewScaleEndDetails creates details for a GestureScaleEndCallback.
func NewScaleEndDetails(opts ...ScaleEndDetailsOptions) *ScaleEndDetails {
	var o ScaleEndDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &ScaleEndDetails{Velocity: o.Velocity, ScaleVelocity: o.ScaleVelocity, PointerCount: o.PointerCount}
	d.SetSelf(d)
	return d
}

func (d *ScaleEndDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
	properties.Add(foundation.NewDoubleProperty("scaleVelocity", d.ScaleVelocity))
	properties.Add(foundation.NewIntProperty("pointerCount", d.PointerCount))
}

// GestureScaleStartCallback is the signature for when the pointers have
// established a focal point and initial scale of 1.0.
type GestureScaleStartCallback func(details *ScaleStartDetails)

// GestureScaleUpdateCallback is the signature for when the pointers
// indicate a new focal point and/or scale.
type GestureScaleUpdateCallback func(details *ScaleUpdateDetails)

// GestureScaleEndCallback is the signature for when the pointers are no
// longer in contact with the screen.
type GestureScaleEndCallback func(details *ScaleEndDetails)

func isScaleFlingGesture(velocity Velocity) bool {
	speedSquared := velocity.PixelsPerSecond.DistanceSquared()
	return speedSquared > KMinFlingVelocity*KMinFlingVelocity
}

// lineBetweenPointers is _LineBetweenPointers.
type lineBetweenPointers struct {
	pointerStartLocation ui.Offset
	pointerStartID       int
	pointerEndLocation   ui.Offset
	pointerEndID         int
}

func newLineBetweenPointers(startID int, start ui.Offset, endID int, end ui.Offset) *lineBetweenPointers {
	dartAssert(startID != endID, nil)
	return &lineBetweenPointers{pointerStartLocation: start, pointerStartID: startID,
		pointerEndLocation: end, pointerEndID: endID}
}

// ScaleGestureRecognizerOptions are ScaleGestureRecognizer's constructor
// parameters (nil DragStartBehavior: down; nil
// TrackpadScrollToScaleFactor: KDefaultTrackpadScrollToScaleFactor).
type ScaleGestureRecognizerOptions struct {
	GestureRecognizerOptions
	DragStartBehavior           *DragStartBehavior
	TrackpadScrollCausesScale   bool
	TrackpadScrollToScaleFactor *ui.Offset
}

// ScaleGestureRecognizer recognizes a scale gesture: pointers in contact
// with the screen establishing a focal point and initial scale of 1.0,
// then moving.
type ScaleGestureRecognizer struct {
	OneSequenceGestureRecognizerBase

	// DragStartBehavior determines what point is used as the starting
	// point for the gesture.
	DragStartBehavior DragStartBehavior
	// OnStart is called when the pointers have established a focal point
	// and initial scale.
	OnStart GestureScaleStartCallback
	// OnUpdate is called when the pointers indicate a new focal point
	// and/or scale.
	OnUpdate GestureScaleUpdateCallback
	// OnEnd is called when the pointers are no longer in contact.
	OnEnd GestureScaleEndCallback
	// TrackpadScrollCausesScale: trackpad scrolling scales instead of
	// panning.
	TrackpadScrollCausesScale bool
	// TrackpadScrollToScaleFactor converts trackpad scrolling to scale.
	TrackpadScrollToScaleFactor ui.Offset

	state         scaleState
	lastTransform *vm.Matrix4

	initialFocalPoint     ui.Offset
	currentFocalPoint     *ui.Offset
	initialSpan           float64
	currentSpan           float64
	initialHorizontalSpan float64
	currentHorizontalSpan float64
	initialVerticalSpan   float64
	currentVerticalSpan   float64
	localFocalPoint       ui.Offset
	initialLine           *lineBetweenPointers
	currentLine           *lineBetweenPointers
	pointerLocations      *linkedMap[ui.Offset]
	pointerQueue          []int
	velocityTrackers      map[int]*VelocityTracker
	scaleVelocityTracker  *VelocityTracker
	delta                 ui.Offset
	pointerPanZooms       *linkedMap[*pointerPanZoomData]

	initialPanZoomScaleFactor    float64
	initialPanZoomRotationFactor float64
	initialEventTimestamp        *time.Duration
}

// NewScaleGestureRecognizer creates a scale recognizer.
func NewScaleGestureRecognizer(opts ...ScaleGestureRecognizerOptions) *ScaleGestureRecognizer {
	r := &ScaleGestureRecognizer{}
	r.Construct(r, opts...)
	return r
}

// Construct is ScaleGestureRecognizer's constructor, for a type extending
// it (self embeds r).
func (r *ScaleGestureRecognizer) Construct(self OneSequenceGestureRecognizer, opts ...ScaleGestureRecognizerOptions) {
	var o ScaleGestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r.DragStartBehavior = DragStartBehaviorDown
	if o.DragStartBehavior != nil {
		r.DragStartBehavior = *o.DragStartBehavior
	}
	r.TrackpadScrollCausesScale = o.TrackpadScrollCausesScale
	r.TrackpadScrollToScaleFactor = KDefaultTrackpadScrollToScaleFactor
	if o.TrackpadScrollToScaleFactor != nil {
		r.TrackpadScrollToScaleFactor = *o.TrackpadScrollToScaleFactor
	}
	r.pointerLocations = newLinkedMap[ui.Offset]()
	r.velocityTrackers = map[int]*VelocityTracker{}
	r.pointerPanZooms = newLinkedMap[*pointerPanZoomData]()
	r.initialPanZoomScaleFactor = 1
	r.initialPanZoomRotationFactor = 0
	r.OneSequenceGestureRecognizerBase.Construct(self, o.GestureRecognizerOptions)
}

// PointerCount is the number of pointers being tracked (a trackpad
// pan/zoom counts as two).
func (r *ScaleGestureRecognizer) PointerCount() int {
	return 2*r.pointerPanZooms.len() + len(r.pointerQueue)
}

func (r *ScaleGestureRecognizer) pointerScaleFactor() float64 {
	if r.initialSpan > 0.0 {
		return r.currentSpan / r.initialSpan
	}
	return 1.0
}

func (r *ScaleGestureRecognizer) pointerHorizontalScaleFactor() float64 {
	if r.initialHorizontalSpan > 0.0 {
		return r.currentHorizontalSpan / r.initialHorizontalSpan
	}
	return 1.0
}

func (r *ScaleGestureRecognizer) pointerVerticalScaleFactor() float64 {
	if r.initialVerticalSpan > 0.0 {
		return r.currentVerticalSpan / r.initialVerticalSpan
	}
	return 1.0
}

func (r *ScaleGestureRecognizer) panZoomScaled(scale float64) float64 {
	for _, k := range r.pointerPanZooms.keys {
		p := r.pointerPanZooms.values[k]
		scale = float64(scale * (p.scaleValue() / r.initialPanZoomScaleFactor))
	}
	return scale
}

func (r *ScaleGestureRecognizer) scaleFactor() float64 {
	return r.panZoomScaled(r.pointerScaleFactor())
}

func (r *ScaleGestureRecognizer) horizontalScaleFactor() float64 {
	return r.panZoomScaled(r.pointerHorizontalScaleFactor())
}

func (r *ScaleGestureRecognizer) verticalScaleFactor() float64 {
	return r.panZoomScaled(r.pointerVerticalScaleFactor())
}

func (r *ScaleGestureRecognizer) computeRotationFactor() float64 {
	factor := 0.0
	if r.initialLine != nil && r.currentLine != nil {
		fx := r.initialLine.pointerStartLocation.Dx
		fy := r.initialLine.pointerStartLocation.Dy
		sx := r.initialLine.pointerEndLocation.Dx
		sy := r.initialLine.pointerEndLocation.Dy
		nfx := r.currentLine.pointerStartLocation.Dx
		nfy := r.currentLine.pointerStartLocation.Dy
		nsx := r.currentLine.pointerEndLocation.Dx
		nsy := r.currentLine.pointerEndLocation.Dy
		angle1 := dart.Atan2(fy-sy, fx-sx)
		angle2 := dart.Atan2(nfy-nsy, nfx-nsx)
		factor = angle2 - angle1
	}
	for _, k := range r.pointerPanZooms.keys {
		factor += r.pointerPanZooms.values[k].rotation
	}
	factor -= r.initialPanZoomRotationFactor
	return factor
}

func (r *ScaleGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	r.OneSequenceGestureRecognizerBase.AddAllowedPointer(event)
	r.velocityTrackers[event.Pointer()] = NewVelocityTrackerWithKind(event.Kind())
	ts := event.TimeStamp()
	r.initialEventTimestamp = &ts
	if r.state == scaleStateReady {
		r.state = scaleStatePossible
		r.initialSpan = 0.0
		r.currentSpan = 0.0
		r.initialHorizontalSpan = 0.0
		r.currentHorizontalSpan = 0.0
		r.initialVerticalSpan = 0.0
		r.currentVerticalSpan = 0.0
	}
}

func (r *ScaleGestureRecognizer) IsPointerPanZoomAllowed(event *PointerPanZoomStartEvent) bool {
	return true
}

func (r *ScaleGestureRecognizer) AddAllowedPointerPanZoom(event *PointerPanZoomStartEvent) {
	r.OneSequenceGestureRecognizerBase.AddAllowedPointerPanZoom(event)
	r.StartTrackingPointer(event.Pointer(), event.Transform())
	r.velocityTrackers[event.Pointer()] = NewVelocityTrackerWithKind(event.Kind())
	ts := event.TimeStamp()
	r.initialEventTimestamp = &ts
	if r.state == scaleStateReady {
		r.state = scaleStatePossible
		r.initialPanZoomScaleFactor = 1.0
		r.initialPanZoomRotationFactor = 0.0
	}
}

func (r *ScaleGestureRecognizer) HandleEvent(event PointerEvent) {
	dartAssert(r.state != scaleStateReady, nil)
	didChangeConfiguration := false
	shouldStartIfAccepted := false
	switch e := event.(type) {
	case *PointerMoveEvent:
		tracker := r.velocityTrackers[e.Pointer()]
		if !e.Synthesized() {
			tracker.AddPosition(e.TimeStamp(), e.Position())
		}
		r.pointerLocations.set(e.Pointer(), e.Position())
		shouldStartIfAccepted = true
		r.lastTransform = e.Transform()
	case *PointerDownEvent:
		r.pointerLocations.set(e.Pointer(), e.Position())
		r.pointerQueue = append(r.pointerQueue, e.Pointer())
		didChangeConfiguration = true
		shouldStartIfAccepted = true
		r.lastTransform = e.Transform()
	case *PointerUpEvent, *PointerCancelEvent:
		r.pointerLocations.remove(event.Pointer())
		r.removeFromQueue(event.Pointer())
		didChangeConfiguration = true
		r.lastTransform = event.Transform()
	case *PointerPanZoomStartEvent:
		_, exists := r.pointerPanZooms.get(e.Pointer())
		dartAssert(!exists, nil)
		r.pointerPanZooms.set(e.Pointer(), pointerPanZoomDataFromStartEvent(r, e))
		didChangeConfiguration = true
		shouldStartIfAccepted = true
		r.lastTransform = e.Transform()
	case *PointerPanZoomUpdateEvent:
		_, exists := r.pointerPanZooms.get(e.Pointer())
		dartAssert(exists, nil)
		if !e.Synthesized() && !r.TrackpadScrollCausesScale {
			r.velocityTrackers[e.Pointer()].AddPosition(e.TimeStamp(), e.Pan())
		}
		r.pointerPanZooms.set(e.Pointer(), pointerPanZoomDataFromUpdateEvent(r, e))
		r.lastTransform = e.Transform()
		shouldStartIfAccepted = true
	case *PointerPanZoomEndEvent:
		_, exists := r.pointerPanZooms.get(e.Pointer())
		dartAssert(exists, nil)
		r.pointerPanZooms.remove(e.Pointer())
		didChangeConfiguration = true
	}
	r.updateLines()
	r.update()
	if !didChangeConfiguration || r.reconfigure(event.Pointer()) {
		r.advanceStateMachine(shouldStartIfAccepted, event)
	}
	r.StopTrackingIfPointerNoLongerDown(event)
}

// removeFromQueue is List.remove: the first occurrence.
func (r *ScaleGestureRecognizer) removeFromQueue(pointer int) {
	if i := slices.Index(r.pointerQueue, pointer); i >= 0 {
		r.pointerQueue = slices.Delete(r.pointerQueue, i, i+1)
	}
}

func (r *ScaleGestureRecognizer) update() {
	previousFocalPoint := r.currentFocalPoint

	// Compute the focal point.
	focalPoint := ui.Offset{}
	for _, pointer := range r.pointerLocations.keys {
		focalPoint = focalPoint.Add(r.pointerLocations.values[pointer])
	}
	for _, k := range r.pointerPanZooms.keys {
		focalPoint = focalPoint.Add(r.pointerPanZooms.values[k].focalPoint())
	}
	current := focalPoint.Div(float64(max(1, r.pointerLocations.len()+r.pointerPanZooms.len())))
	r.currentFocalPoint = &current

	if previousFocalPoint == nil {
		r.localFocalPoint = PointerEventTransformPosition(r.lastTransform, current)
		r.delta = ui.Offset{}
	} else {
		localPreviousFocalPoint := r.localFocalPoint
		r.localFocalPoint = PointerEventTransformPosition(r.lastTransform, current)
		r.delta = r.localFocalPoint.Sub(localPreviousFocalPoint)
	}

	count := r.pointerLocations.len()

	pointerFocalPoint := ui.Offset{}
	for _, pointer := range r.pointerLocations.keys {
		pointerFocalPoint = pointerFocalPoint.Add(r.pointerLocations.values[pointer])
	}
	if count > 0 {
		pointerFocalPoint = pointerFocalPoint.Div(float64(count))
	}

	totalDeviation := 0.0
	totalHorizontalDeviation := 0.0
	totalVerticalDeviation := 0.0
	for _, pointer := range r.pointerLocations.keys {
		location := r.pointerLocations.values[pointer]
		totalDeviation += pointerFocalPoint.Sub(location).Distance()
		totalHorizontalDeviation += math.Abs(pointerFocalPoint.Dx - location.Dx)
		totalVerticalDeviation += math.Abs(pointerFocalPoint.Dy - location.Dy)
	}
	if count > 0 {
		r.currentSpan = totalDeviation / float64(count)
		r.currentHorizontalSpan = totalHorizontalDeviation / float64(count)
		r.currentVerticalSpan = totalVerticalDeviation / float64(count)
	} else {
		r.currentSpan = 0.0
		r.currentHorizontalSpan = 0.0
		r.currentVerticalSpan = 0.0
	}
}

func (r *ScaleGestureRecognizer) lineOfQueue() *lineBetweenPointers {
	return newLineBetweenPointers(r.pointerQueue[0], r.pointerLocations.values[r.pointerQueue[0]],
		r.pointerQueue[1], r.pointerLocations.values[r.pointerQueue[1]])
}

func (r *ScaleGestureRecognizer) updateLines() {
	count := r.pointerLocations.len()
	dartAssert(len(r.pointerQueue) >= count, nil)
	if count < 2 {
		r.initialLine = r.currentLine
	} else if r.initialLine != nil &&
		r.initialLine.pointerStartID == r.pointerQueue[0] &&
		r.initialLine.pointerEndID == r.pointerQueue[1] {
		r.currentLine = r.lineOfQueue()
	} else {
		r.initialLine = r.lineOfQueue()
		r.currentLine = r.initialLine
	}
}

func (r *ScaleGestureRecognizer) panZoomRotationSum() float64 {
	// .map(rotation).reduce(+): the first value, then the others added.
	sum := 0.0
	for i, k := range r.pointerPanZooms.keys {
		if i == 0 {
			sum = r.pointerPanZooms.values[k].rotation
		} else {
			sum = sum + r.pointerPanZooms.values[k].rotation
		}
	}
	return sum
}

func (r *ScaleGestureRecognizer) resetInitial() {
	r.initialFocalPoint = *r.currentFocalPoint
	r.initialSpan = r.currentSpan
	r.initialLine = r.currentLine
	r.initialHorizontalSpan = r.currentHorizontalSpan
	r.initialVerticalSpan = r.currentVerticalSpan
	if r.pointerPanZooms.len() == 0 {
		r.initialPanZoomScaleFactor = 1.0
		r.initialPanZoomRotationFactor = 0.0
	} else {
		r.initialPanZoomScaleFactor = r.scaleFactor() / r.pointerScaleFactor()
		r.initialPanZoomRotationFactor = r.panZoomRotationSum()
	}
}

func (r *ScaleGestureRecognizer) scaleVelocity() float64 {
	if r.scaleVelocityTracker == nil {
		return -1
	}
	return r.scaleVelocityTracker.GetVelocity().PixelsPerSecond.Dx
}

func (r *ScaleGestureRecognizer) reconfigure(pointer int) bool {
	r.resetInitial()
	if r.state == scaleStateStarted {
		if r.OnEnd != nil {
			tracker := r.velocityTrackers[pointer]
			velocity := tracker.GetVelocity()
			if isScaleFlingGesture(velocity) {
				pixelsPerSecond := velocity.PixelsPerSecond
				if pixelsPerSecond.DistanceSquared() > KMaxFlingVelocity*KMaxFlingVelocity {
					velocity = Velocity{PixelsPerSecond: pixelsPerSecond.Div(pixelsPerSecond.Distance()).Mul(KMaxFlingVelocity)}
				}
				InvokeVoidCallback(r, "onEnd", func() {
					r.OnEnd(NewScaleEndDetails(ScaleEndDetailsOptions{Velocity: velocity,
						ScaleVelocity: r.scaleVelocity(), PointerCount: r.PointerCount()}))
				})
			} else {
				InvokeVoidCallback(r, "onEnd", func() {
					r.OnEnd(NewScaleEndDetails(ScaleEndDetailsOptions{Velocity: VelocityZero,
						ScaleVelocity: r.scaleVelocity(), PointerCount: r.PointerCount()}))
				})
			}
		}
		r.state = scaleStateAccepted
		r.scaleVelocityTracker = NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch) // arbitrary PointerDeviceKind
		return false
	}
	r.scaleVelocityTracker = NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch) // arbitrary PointerDeviceKind
	return true
}


func (r *ScaleGestureRecognizer) advanceStateMachine(shouldStartIfAccepted bool, event PointerEvent) {
	if r.state == scaleStateReady {
		r.state = scaleStatePossible
	}

	if r.state == scaleStatePossible {
		spanDelta := math.Abs(r.currentSpan - r.initialSpan)
		focalPointDelta := r.currentFocalPoint.Sub(r.initialFocalPoint).Distance()
		if spanDelta > ComputeScaleSlop(event.Kind()) ||
			focalPointDelta > ComputePanSlop(event.Kind(), r.GestureSettings) ||
			dartMax(r.scaleFactor()/r.pointerScaleFactor(), r.pointerScaleFactor()/r.scaleFactor()) > 1.05 {
			r.Resolve(GestureDispositionAccepted)
		}
	} else if r.state >= scaleStateAccepted {
		r.Resolve(GestureDispositionAccepted)
	}

	if r.state == scaleStateAccepted && shouldStartIfAccepted {
		ts := event.TimeStamp()
		r.initialEventTimestamp = &ts
		r.state = scaleStateStarted
		r.dispatchOnStartCallbackIfNeeded()
	}

	if r.state == scaleStateStarted {
		if r.scaleVelocityTracker != nil {
			r.scaleVelocityTracker.AddPosition(event.TimeStamp(), ui.NewOffset(r.scaleFactor(), 0))
		}
		if r.OnUpdate != nil {
			InvokeVoidCallback(r, "onUpdate", func() {
				scale, hScale, vScale := r.scaleFactor(), r.horizontalScaleFactor(), r.verticalScaleFactor()
				local := r.localFocalPoint
				ts := event.TimeStamp()
				r.OnUpdate(NewScaleUpdateDetails(ScaleUpdateDetailsOptions{
					Scale:           &scale,
					HorizontalScale: &hScale,
					VerticalScale:   &vScale,
					FocalPoint:      *r.currentFocalPoint,
					LocalFocalPoint: &local,
					Rotation:        r.computeRotationFactor(),
					PointerCount:    r.PointerCount(),
					FocalPointDelta: r.delta,
					SourceTimeStamp: &ts,
				}))
			})
		}
	}
}

func (r *ScaleGestureRecognizer) dispatchOnStartCallbackIfNeeded() {
	dartAssert(r.state == scaleStateStarted, nil)
	if r.OnStart != nil {
		InvokeVoidCallback(r, "onStart", func() {
			var kind *ui.PointerDeviceKind
			if len(r.pointerQueue) > 0 {
				k := r.GetKindForPointer(r.pointerQueue[0])
				kind = &k
			} else if r.pointerPanZooms.len() > 0 {
				k := r.GetKindForPointer(r.pointerPanZooms.keys[0])
				kind = &k
			}
			local := r.localFocalPoint
			r.OnStart(NewScaleStartDetails(ScaleStartDetailsOptions{
				FocalPoint:      *r.currentFocalPoint,
				LocalFocalPoint: &local,
				PointerCount:    r.PointerCount(),
				SourceTimeStamp: r.initialEventTimestamp,
				Kind:            kind,
			}))
		})
	}
	r.initialEventTimestamp = nil
}

func (r *ScaleGestureRecognizer) AcceptGesture(pointer int) {
	if r.state == scaleStatePossible {
		r.state = scaleStateStarted
		r.dispatchOnStartCallbackIfNeeded()
		if r.DragStartBehavior == DragStartBehaviorStart {
			r.resetInitial()
		}
	}
}

func (r *ScaleGestureRecognizer) RejectGesture(pointer int) {
	r.pointerPanZooms.remove(pointer)
	r.pointerLocations.remove(pointer)
	r.removeFromQueue(pointer)
	r.StopTrackingPointer(pointer)
}

func (r *ScaleGestureRecognizer) DidStopTrackingLastPointer(pointer int) {
	switch r.state {
	case scaleStatePossible:
		r.Resolve(GestureDispositionRejected)
	case scaleStateReady:
		dartAssert(false, nil) // We should have not seen a pointer yet
	case scaleStateAccepted:
	case scaleStateStarted:
		dartAssert(false, nil) // We should be in the accepted state when user is done
	}
	r.state = scaleStateReady
}

func (r *ScaleGestureRecognizer) Dispose() {
	clear(r.velocityTrackers)
	r.OneSequenceGestureRecognizerBase.Dispose()
}

func (r *ScaleGestureRecognizer) DebugDescription() string { return "scale" }
