package gestures

// Port of events.dart (Flutter 3.50.1).
//
// PointerEvent (an abstract class in Dart) is an interface; each event
// class is a struct used by pointer (*PointerDownEvent...), so recognizers
// type-switch on them as Dart tests `event is PointerDownEvent`. The
// structs embed PointerEventBase, which holds the fields, implements the
// getters, Diagnosticable (the private _PointerEventDescription mixin) and
// the transform machinery.
//
// Dart's _TransformedPointerXEvent wrappers (which implement PointerXEvent)
// are values of the same Go struct (*PointerXEvent) whose Original() is the
// untransformed event and Transform() the matrix: their getters return the
// original's values, LocalPosition/LocalDelta (and a pan/zoom update's
// LocalPan/LocalPanDelta) are computed when first read, and their runtime
// type (toString, toStringShort) is "_TransformedPointerXEvent".
//
// copyWith is CopyWith(PointerEventCopyWithOptions) returning PointerEvent
// (Go has no covariant return types: assert the concrete type), and
// transformed is Transformed(*Matrix4) PointerEvent. Constructors take a
// per-class options struct whose zero value is Dart's defaults (options
// whose Dart default isn't zero are pointers).

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// The bit of PointerEvent.Buttons that corresponds to a cross-device
// behavior of "primary operation".
const KPrimaryButton = 0x01

// KSecondaryButton is the bit of PointerEvent.Buttons that corresponds to
// a cross-device behavior of "secondary operation".
const KSecondaryButton = 0x02

// KPrimaryMouseButton is the primary mouse button.
const KPrimaryMouseButton = KPrimaryButton

// KSecondaryMouseButton is the secondary mouse button.
const KSecondaryMouseButton = KSecondaryButton

// KStylusContact is the primary button of a stylus (contact with the
// surface).
const KStylusContact = KPrimaryButton

// KPrimaryStylusButton is the primary stylus button.
const KPrimaryStylusButton = KSecondaryButton

// KTertiaryButton is the bit of PointerEvent.Buttons that corresponds to a
// cross-device behavior of "tertiary operation".
const KTertiaryButton = 0x04

// KMiddleMouseButton is the middle mouse button.
const KMiddleMouseButton = KTertiaryButton

// KSecondaryStylusButton is the secondary stylus button.
const KSecondaryStylusButton = KTertiaryButton

// KBackMouseButton is the back mouse button.
const KBackMouseButton = 0x08

// KForwardMouseButton is the forward mouse button.
const KForwardMouseButton = 0x10

// KTouchContact is the bit of a touch device's buttons in contact.
const KTouchContact = KPrimaryButton

// NthMouseButton returns the button of the nth mouse button (1-based).
func NthMouseButton(number int) int {
	return (KPrimaryMouseButton << (number - 1)) & foundation.KMaxUnsignedSMI
}

// NthStylusButton returns the button of the nth stylus button (1-based).
func NthStylusButton(number int) int {
	return (KPrimaryStylusButton << (number - 1)) & foundation.KMaxUnsignedSMI
}

// SmallestButton returns the button with the smallest integer of buttons
// (0 if none).
func SmallestButton(buttons int) int { return buttons & (-buttons) }

// IsSingleButton reports whether buttons contains one and only one button.
func IsSingleButton(buttons int) bool {
	return buttons != 0 && (SmallestButton(buttons) == buttons)
}

// PointerEvent is the base of touch, stylus or mouse events.
type PointerEvent interface {
	foundation.Diagnosticable

	// ViewID is the id of the FlutterView this event was generated for.
	ViewID() int
	// EmbedderID is the unique identifier the embedder gave the event.
	EmbedderID() int
	// TimeStamp is the time the event was dispatched.
	TimeStamp() time.Duration
	// Pointer is the unique identifier of the pointer (not reused).
	Pointer() int
	// Kind is the kind of input device.
	Kind() ui.PointerDeviceKind
	// Device is the unique identifier of the pointing device.
	Device() int
	// Position is the coordinates of the pointer, in logical pixels in the
	// global coordinate space.
	Position() ui.Offset
	// LocalPosition is Position transformed into the event receiver's
	// local coordinate system according to Transform.
	LocalPosition() ui.Offset
	// Delta is the distance the pointer moved since the last move or hover
	// event (global coordinates).
	Delta() ui.Offset
	// LocalDelta is Delta transformed by Transform.
	LocalDelta() ui.Offset
	// Buttons is the bit field of the buttons pressed.
	Buttons() int
	// Down reports whether the pointer is in contact with the device.
	Down() bool
	// Obscured reports whether the pointer was obscured by another window.
	Obscured() bool
	// Pressure is the pressure of the touch.
	Pressure() float64
	// PressureMin is the minimum value Pressure can have.
	PressureMin() float64
	// PressureMax is the maximum value Pressure can have.
	PressureMax() float64
	// Distance is the distance of the detected object from the input
	// surface.
	Distance() float64
	// DistanceMin is the minimum value Distance can have (always 0).
	DistanceMin() float64
	// DistanceMax is the maximum value Distance can have.
	DistanceMax() float64
	// Size is the area of the screen being pressed.
	Size() float64
	// RadiusMajor is the radius of the contact ellipse along the major
	// axis.
	RadiusMajor() float64
	// RadiusMinor is the radius along the minor axis.
	RadiusMinor() float64
	// RadiusMin is the minimum value the radii can have.
	RadiusMin() float64
	// RadiusMax is the maximum value the radii can have.
	RadiusMax() float64
	// Orientation is the angle of the detected object.
	Orientation() float64
	// Tilt is the tilt angle of the detected object.
	Tilt() float64
	// PlatformData is opaque platform-specific data.
	PlatformData() int
	// Synthesized reports whether the event was synthesized by Flutter.
	Synthesized() bool
	// Transform transforms the event from the global coordinate space into
	// the receiver's (nil: none).
	Transform() *vm.Matrix4
	// Original is the original untransformed event, or nil if this is it.
	Original() PointerEvent
	// Transformed returns this event transformed by transform into the
	// receiver's coordinate space (this event if transform is nil or equal
	// to Transform).
	Transformed(transform *vm.Matrix4) PointerEvent
	// CopyWith creates a copy of the event with the given fields replaced
	// (each class keeps the fields its constructor takes).
	CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent
	// ToStringFull returns a string with all the event's properties.
	ToStringFull() string

	pointerEvent() *PointerEventBase
}

// PointerEventCopyWithOptions are copyWith's parameters (nil: keep). Each
// class uses the ones its constructor takes, as in Dart.
type PointerEventCopyWithOptions struct {
	ViewID        *int
	TimeStamp     *time.Duration
	Pointer       *int
	Kind          *ui.PointerDeviceKind
	Device        *int
	Position      *ui.Offset
	LocalPosition *ui.Offset // PointerUpEvent's (unused, as in Dart)
	Delta         *ui.Offset
	Buttons       *int
	Obscured      *bool
	Pressure      *float64
	PressureMin   *float64
	PressureMax   *float64
	Distance      *float64
	DistanceMax   *float64
	Size          *float64
	RadiusMajor   *float64
	RadiusMinor   *float64
	RadiusMin     *float64
	RadiusMax     *float64
	Orientation   *float64
	Tilt          *float64
	Synthesized   *bool
	EmbedderID    *int
	// PointerScrollEvent's.
	OnRespond RespondPointerEventCallback
	// PointerScaleEvent's and PointerPanZoomUpdateEvent's.
	Scale *float64
	// PointerPanZoomUpdateEvent's.
	Pan           *ui.Offset
	LocalPan      *ui.Offset
	PanDelta      *ui.Offset
	LocalPanDelta *ui.Offset
	Rotation      *float64
}

// pointerEventData are PointerEvent's fields.
type pointerEventData struct {
	viewID       int
	embedderID   int
	timeStamp    time.Duration
	pointer      int
	kind         ui.PointerDeviceKind
	device       int
	position     ui.Offset
	delta        ui.Offset
	buttons      int
	down         bool
	obscured     bool
	pressure     float64
	pressureMin  float64
	pressureMax  float64
	distance     float64
	distanceMax  float64
	size         float64
	radiusMajor  float64
	radiusMinor  float64
	radiusMin    float64
	radiusMax    float64
	orientation  float64
	tilt         float64
	platformData int
	synthesized  bool
}

// PointerEventBase is PointerEvent's state and methods, embedded by every
// event class. A type extending PointerEvent outside this package embeds
// it, calls InitPointerEvent and implements Transformed and CopyWith.
type PointerEventBase struct {
	foundation.DiagnosticableMixin
	self PointerEvent
	pointerEventData

	transform *vm.Matrix4
	original  PointerEvent

	// Transformed events: localPosition/localDelta are late finals.
	isTransformed bool
	localPosition *ui.Offset
	localDelta    *ui.Offset
}

// PointerEventOptions are PointerEvent's constructor parameters, for types
// extending it (the zero value of each is Dart's default, except the
// pressures, whose Dart default is 1.0: set them with
// PointerEventOptions.Pressure* pointers).
type PointerEventOptions struct {
	ViewID       int
	EmbedderID   int
	TimeStamp    time.Duration
	Pointer      int
	Kind         ui.PointerDeviceKind
	Device       int
	Position     ui.Offset
	Delta        ui.Offset
	Buttons      int
	Down         bool
	Obscured     bool
	Pressure     *float64
	PressureMin  *float64
	PressureMax  *float64
	Distance     float64
	DistanceMax  float64
	Size         float64
	RadiusMajor  float64
	RadiusMinor  float64
	RadiusMin    float64
	RadiusMax    float64
	Orientation  float64
	Tilt         float64
	PlatformData int
	Synthesized  bool
	Transform    *vm.Matrix4
	Original     PointerEvent
}

// InitPointerEvent is the PointerEvent constructor for a type extending
// it: it sets the fields and self.
func (e *PointerEventBase) InitPointerEvent(self PointerEvent, o PointerEventOptions) {
	e.pointerEventData = pointerEventData{
		viewID: o.ViewID, embedderID: o.EmbedderID, timeStamp: o.TimeStamp, pointer: o.Pointer,
		kind: o.Kind, device: o.Device, position: o.Position, delta: o.Delta, buttons: o.Buttons,
		down: o.Down, obscured: o.Obscured, pressure: orDouble(o.Pressure, 1.0),
		pressureMin: orDouble(o.PressureMin, 1.0), pressureMax: orDouble(o.PressureMax, 1.0),
		distance: o.Distance, distanceMax: o.DistanceMax, size: o.Size, radiusMajor: o.RadiusMajor,
		radiusMinor: o.RadiusMinor, radiusMin: o.RadiusMin, radiusMax: o.RadiusMax,
		orientation: o.Orientation, tilt: o.Tilt, platformData: o.PlatformData, synthesized: o.Synthesized,
	}
	e.transform = o.Transform
	e.original = o.Original
	e.setSelf(self)
}

func (e *PointerEventBase) setSelf(self PointerEvent) {
	e.self = self
	e.DiagnosticableMixin.SetSelf(self)
}

func (e *PointerEventBase) outer() PointerEvent {
	if e.self == nil {
		panic("gestures: PointerEventBase used before InitPointerEvent")
	}
	return e.self
}

func (e *PointerEventBase) pointerEvent() *PointerEventBase { return e }

func orDouble(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}

func orInt(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

func (e *PointerEventBase) ViewID() int                { return e.viewID }
func (e *PointerEventBase) EmbedderID() int            { return e.embedderID }
func (e *PointerEventBase) TimeStamp() time.Duration   { return e.timeStamp }
func (e *PointerEventBase) Pointer() int               { return e.pointer }
func (e *PointerEventBase) Kind() ui.PointerDeviceKind { return e.kind }
func (e *PointerEventBase) Device() int                { return e.device }
func (e *PointerEventBase) Position() ui.Offset        { return e.position }
func (e *PointerEventBase) Delta() ui.Offset           { return e.delta }
func (e *PointerEventBase) Buttons() int               { return e.buttons }
func (e *PointerEventBase) Down() bool                 { return e.down }
func (e *PointerEventBase) Obscured() bool             { return e.obscured }
func (e *PointerEventBase) Pressure() float64          { return e.pressure }
func (e *PointerEventBase) PressureMin() float64       { return e.pressureMin }
func (e *PointerEventBase) PressureMax() float64       { return e.pressureMax }
func (e *PointerEventBase) Distance() float64          { return e.distance }
func (e *PointerEventBase) DistanceMin() float64       { return 0.0 }
func (e *PointerEventBase) DistanceMax() float64       { return e.distanceMax }
func (e *PointerEventBase) Size() float64              { return e.size }
func (e *PointerEventBase) RadiusMajor() float64       { return e.radiusMajor }
func (e *PointerEventBase) RadiusMinor() float64       { return e.radiusMinor }
func (e *PointerEventBase) RadiusMin() float64         { return e.radiusMin }
func (e *PointerEventBase) RadiusMax() float64         { return e.radiusMax }
func (e *PointerEventBase) Orientation() float64       { return e.orientation }
func (e *PointerEventBase) Tilt() float64              { return e.tilt }
func (e *PointerEventBase) PlatformData() int          { return e.platformData }
func (e *PointerEventBase) Synthesized() bool          { return e.synthesized }
func (e *PointerEventBase) Transform() *vm.Matrix4     { return e.transform }
func (e *PointerEventBase) Original() PointerEvent     { return e.original }

// LocalPosition is Position transformed by Transform (computed when first
// read for a transformed event).
func (e *PointerEventBase) LocalPosition() ui.Offset {
	if !e.isTransformed {
		return e.position
	}
	if e.localPosition == nil {
		p := PointerEventTransformPosition(e.transform, e.position)
		e.localPosition = &p
	}
	return *e.localPosition
}

// LocalDelta is Delta transformed by Transform (computed when first read
// for a transformed event).
func (e *PointerEventBase) LocalDelta() ui.Offset {
	if !e.isTransformed {
		return e.delta
	}
	if e.localDelta == nil {
		localPosition := e.LocalPosition()
		d := PointerEventTransformDeltaViaPositions(e.position, &localPosition, e.delta, e.transform)
		e.localDelta = &d
	}
	return *e.localDelta
}

// ToStringShort is the runtime type ("_TransformedPointerXEvent" for a
// transformed event) and the identity hash.
func (e *PointerEventBase) ToStringShort() string {
	name := foundation.ObjectRuntimeType(e.outer(), "PointerEvent")
	if e.isTransformed {
		name = "_Transformed" + name
	}
	return name + "#" + foundation.ShortHash(e.outer())
}

// DebugFillProperties adds the event's properties (the
// _PointerEventDescription mixin): position at info level, the rest at
// debug level.
func (e *PointerEventBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.DiagnosticableMixin.DebugFillProperties(properties)
	self := e.outer()
	debug := foundation.DiagnosticLevelDebug
	lv := &debug
	properties.Add(foundation.NewDiagnosticsProperty("position", self.Position()))
	properties.Add(foundation.NewDiagnosticsProperty("localPosition", self.LocalPosition(),
		foundation.DiagnosticsPropertyOptions{DefaultValue: self.Position(), Level: lv}))
	properties.Add(foundation.NewDiagnosticsProperty("delta", self.Delta(),
		foundation.DiagnosticsPropertyOptions{DefaultValue: ui.OffsetZero, Level: lv}))
	properties.Add(foundation.NewDiagnosticsProperty("localDelta", self.LocalDelta(),
		foundation.DiagnosticsPropertyOptions{DefaultValue: self.Delta(), Level: lv}))
	properties.Add(foundation.NewDiagnosticsProperty("timeStamp", self.TimeStamp(),
		foundation.DiagnosticsPropertyOptions{DefaultValue: time.Duration(0), Level: lv}))
	properties.Add(foundation.NewIntProperty("pointer", self.Pointer(), foundation.IntPropertyOptions{Level: lv}))
	properties.Add(foundation.NewEnumProperty("kind", self.Kind(), foundation.EnumPropertyOptions{Level: lv}))
	properties.Add(foundation.NewIntProperty("device", self.Device(), foundation.IntPropertyOptions{DefaultValue: 0, Level: lv}))
	properties.Add(foundation.NewIntProperty("buttons", self.Buttons(), foundation.IntPropertyOptions{DefaultValue: 0, Level: lv}))
	properties.Add(foundation.NewDiagnosticsProperty("down", self.Down(), foundation.DiagnosticsPropertyOptions{Level: lv}))
	dbl := func(name string, value, def float64) {
		properties.Add(foundation.NewDoubleProperty(name, value, foundation.DoublePropertyOptions{DefaultValue: def, Level: lv}))
	}
	dbl("pressure", self.Pressure(), 1.0)
	dbl("pressureMin", self.PressureMin(), 1.0)
	dbl("pressureMax", self.PressureMax(), 1.0)
	dbl("distance", self.Distance(), 0.0)
	dbl("distanceMin", self.DistanceMin(), 0.0)
	dbl("distanceMax", self.DistanceMax(), 0.0)
	dbl("size", self.Size(), 0.0)
	dbl("radiusMajor", self.RadiusMajor(), 0.0)
	dbl("radiusMinor", self.RadiusMinor(), 0.0)
	dbl("radiusMin", self.RadiusMin(), 0.0)
	dbl("radiusMax", self.RadiusMax(), 0.0)
	dbl("orientation", self.Orientation(), 0.0)
	dbl("tilt", self.Tilt(), 0.0)
	properties.Add(foundation.NewIntProperty("platformData", self.PlatformData(), foundation.IntPropertyOptions{DefaultValue: 0, Level: lv}))
	obscured, synthesized := "obscured", "synthesized"
	properties.Add(foundation.NewFlagProperty("obscured", self.Obscured(), foundation.FlagPropertyOptions{IfTrue: &obscured, Level: lv}))
	properties.Add(foundation.NewFlagProperty("synthesized", self.Synthesized(), foundation.FlagPropertyOptions{IfTrue: &synthesized, Level: lv}))
	properties.Add(foundation.NewIntProperty("embedderId", self.EmbedderID(), foundation.IntPropertyOptions{DefaultValue: 0, Level: lv}))
	properties.Add(foundation.NewIntProperty("viewId", self.ViewID(), foundation.IntPropertyOptions{DefaultValue: 0, Level: lv}))
}

// ToStringFull returns a string representation with all the properties
// (toString at DiagnosticLevel.fine).
func (e *PointerEventBase) ToStringFull() string {
	fine := foundation.DiagnosticLevelFine
	return e.outer().ToString(foundation.DiagnosticableToStringOptions{MinLevel: &fine})
}

// PointerEventTransformPosition is PointerEvent.transformPosition: position
// transformed by transform (position itself if transform is nil).
func PointerEventTransformPosition(transform *vm.Matrix4, position ui.Offset) ui.Offset {
	if transform == nil {
		return position
	}
	position3 := vm.NewVector3(position.Dx, position.Dy, 0.0)
	transformed3 := transform.PerspectiveTransform(position3)
	return ui.NewOffset(transformed3.X(), transformed3.Y())
}

// PointerEventTransformDeltaViaPositions is
// PointerEvent.transformDeltaViaPositions: it transforms the delta by
// transforming its end and start positions (transformedEndPosition nil:
// computed). The required named parameters are positional.
func PointerEventTransformDeltaViaPositions(untransformedEndPosition ui.Offset, transformedEndPosition *ui.Offset,
	untransformedDelta ui.Offset, transform *vm.Matrix4) ui.Offset {
	if transform == nil {
		return untransformedDelta
	}
	// We could transform the delta directly with the transformation matrix.
	// While that is mathematically equivalent, in practice we are seeing a
	// greater precision error with that approach. Instead, we are
	// transforming start and end point of the delta separately and calculate
	// the delta in the new space for greater accuracy.
	var end ui.Offset
	if transformedEndPosition != nil {
		end = *transformedEndPosition
	} else {
		end = PointerEventTransformPosition(transform, untransformedEndPosition)
	}
	transformedStartPosition := PointerEventTransformPosition(transform, untransformedEndPosition.Sub(untransformedDelta))
	return end.Sub(transformedStartPosition)
}

// PointerEventRemovePerspectiveTransform is
// PointerEvent.removePerspectiveTransform: a copy of transform whose third
// row and column are (0, 0, 1, 0), so that transformed points stay under
// the pointer.
func PointerEventRemovePerspectiveTransform(transform *vm.Matrix4) *vm.Matrix4 {
	vector := vm.NewVector4(0, 0, 1, 0)
	m := transform.Clone()
	m.SetColumn(2, vector)
	m.SetRow(2, vector)
	return m
}

// transformedFrom makes e the transformed version (Dart's
// _TransformedPointerXEvent) of original: original's fields, with transform.
func (e *PointerEventBase) transformedFrom(self, original PointerEvent, transform *vm.Matrix4) {
	ob := original.pointerEvent()
	e.pointerEventData = ob.pointerEventData
	e.transform = transform
	e.original = original
	e.isTransformed = true
	e.setSelf(self)
}

// shouldTransform is the common start of transformed(): false when the
// event itself is the answer.
func (e *PointerEventBase) shouldTransform(transform *vm.Matrix4) bool {
	if transform == nil {
		return false
	}
	if e.transform != nil && transform.Equal(e.transform) && !e.isTransformed {
		return false
	}
	return true
}

// transformedBase returns the untransformed event this one derives from:
// Dart's `original as PointerXEvent? ?? this`.
func (e *PointerEventBase) untransformed() PointerEvent {
	if e.original != nil {
		return e.original
	}
	return e.outer()
}

// ---- PointerAddedEvent

// PointerAddedEventOptions are PointerAddedEvent's parameters.
type PointerAddedEventOptions struct {
	ViewID      int
	TimeStamp   time.Duration
	Pointer     int
	Kind        ui.PointerDeviceKind
	Device      int
	Position    ui.Offset
	Obscured    bool
	PressureMin *float64 // default 1.0
	PressureMax *float64 // default 1.0
	Distance    float64
	DistanceMax float64
	RadiusMin   float64
	RadiusMax   float64
	Orientation float64
	Tilt        float64
	EmbedderID  int
}

// PointerAddedEvent is sent when the platform notices that a pointer it
// hasn't seen before is available.
type PointerAddedEvent struct{ PointerEventBase }

// NewPointerAddedEvent creates a PointerAddedEvent.
func NewPointerAddedEvent(opts ...PointerAddedEventOptions) *PointerAddedEvent {
	var o PointerAddedEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerAddedEvent{}
	e.InitPointerEvent(e, PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Pointer: o.Pointer,
		Kind: o.Kind, Device: o.Device, Position: o.Position, Obscured: o.Obscured, Pressure: ptr(0.0),
		PressureMin: o.PressureMin, PressureMax: o.PressureMax, Distance: o.Distance, DistanceMax: o.DistanceMax,
		RadiusMin: o.RadiusMin, RadiusMax: o.RadiusMax, Orientation: o.Orientation, Tilt: o.Tilt, EmbedderID: o.EmbedderID})
	return e
}

func ptr[T any](v T) *T { return &v }

// Transformed returns the event transformed by transform.
func (e *PointerAddedEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerAddedEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (as Dart's, `pointer` is not kept).
func (e *PointerAddedEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerAddedEvent(PointerAddedEventOptions{
		ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Kind: orKind(o.Kind, e.Kind()), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), Obscured: orBool(o.Obscured, e.Obscured()),
		PressureMin: ptr(orDouble(o.PressureMin, e.PressureMin())), PressureMax: ptr(orDouble(o.PressureMax, e.PressureMax())),
		Distance: orDouble(o.Distance, e.Distance()), DistanceMax: orDouble(o.DistanceMax, e.DistanceMax()),
		RadiusMin: orDouble(o.RadiusMin, e.RadiusMin()), RadiusMax: orDouble(o.RadiusMax, e.RadiusMax()),
		Orientation: orDouble(o.Orientation, e.Orientation()), Tilt: orDouble(o.Tilt, e.Tilt()),
		EmbedderID: orInt(o.EmbedderID, e.EmbedderID()),
	}).Transformed(e.Transform())
}

func copyOpts(opts []PointerEventCopyWithOptions) PointerEventCopyWithOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return PointerEventCopyWithOptions{}
}

func orDuration(v *time.Duration, def time.Duration) time.Duration {
	if v == nil {
		return def
	}
	return *v
}

func orKind(v *ui.PointerDeviceKind, def ui.PointerDeviceKind) ui.PointerDeviceKind {
	if v == nil {
		return def
	}
	return *v
}

func orOffset(v *ui.Offset, def ui.Offset) ui.Offset {
	if v == nil {
		return def
	}
	return *v
}

func orBool(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

// ---- PointerRemovedEvent

// PointerRemovedEventOptions are PointerRemovedEvent's parameters.
type PointerRemovedEventOptions struct {
	ViewID      int
	TimeStamp   time.Duration
	Pointer     int
	Kind        ui.PointerDeviceKind
	Device      int
	Position    ui.Offset
	Obscured    bool
	PressureMin *float64 // default 1.0
	PressureMax *float64 // default 1.0
	DistanceMax float64
	RadiusMin   float64
	RadiusMax   float64
	Original    *PointerRemovedEvent
	EmbedderID  int
}

// PointerRemovedEvent is sent when a pointer that was previously available
// is no longer available.
type PointerRemovedEvent struct{ PointerEventBase }

// NewPointerRemovedEvent creates a PointerRemovedEvent.
func NewPointerRemovedEvent(opts ...PointerRemovedEventOptions) *PointerRemovedEvent {
	var o PointerRemovedEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerRemovedEvent{}
	var original PointerEvent
	if o.Original != nil {
		original = o.Original
	}
	e.InitPointerEvent(e, PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Pointer: o.Pointer,
		Kind: o.Kind, Device: o.Device, Position: o.Position, Obscured: o.Obscured, Pressure: ptr(0.0),
		PressureMin: o.PressureMin, PressureMax: o.PressureMax, DistanceMax: o.DistanceMax,
		RadiusMin: o.RadiusMin, RadiusMax: o.RadiusMax, Original: original, EmbedderID: o.EmbedderID})
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerRemovedEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerRemovedEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (as Dart's, `pointer` is not kept).
func (e *PointerRemovedEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerRemovedEvent(PointerRemovedEventOptions{
		ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Kind: orKind(o.Kind, e.Kind()), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), Obscured: orBool(o.Obscured, e.Obscured()),
		PressureMin: ptr(orDouble(o.PressureMin, e.PressureMin())), PressureMax: ptr(orDouble(o.PressureMax, e.PressureMax())),
		DistanceMax: orDouble(o.DistanceMax, e.DistanceMax()),
		RadiusMin:   orDouble(o.RadiusMin, e.RadiusMin()), RadiusMax: orDouble(o.RadiusMax, e.RadiusMax()),
		EmbedderID: orInt(o.EmbedderID, e.EmbedderID()),
	}).Transformed(e.Transform())
}

// ---- hover-like events (hover, enter, exit)

// PointerHoverEventOptions are PointerHoverEvent's parameters.
type PointerHoverEventOptions struct {
	ViewID      int
	TimeStamp   time.Duration
	Kind        ui.PointerDeviceKind
	Pointer     int
	Device      int
	Position    ui.Offset
	Delta       ui.Offset
	Buttons     int
	Obscured    bool
	PressureMin *float64 // default 1.0
	PressureMax *float64 // default 1.0
	Distance    float64
	DistanceMax float64
	Size        float64
	RadiusMajor float64
	RadiusMinor float64
	RadiusMin   float64
	RadiusMax   float64
	Orientation float64
	Tilt        float64
	Synthesized bool
	EmbedderID  int
}

func (o PointerHoverEventOptions) base(down bool) PointerEventOptions {
	return PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Kind: o.Kind, Pointer: o.Pointer,
		Device: o.Device, Position: o.Position, Delta: o.Delta, Buttons: o.Buttons, Obscured: o.Obscured,
		Down: down, Pressure: ptr(0.0), PressureMin: o.PressureMin, PressureMax: o.PressureMax,
		Distance: o.Distance, DistanceMax: o.DistanceMax, Size: o.Size, RadiusMajor: o.RadiusMajor,
		RadiusMinor: o.RadiusMinor, RadiusMin: o.RadiusMin, RadiusMax: o.RadiusMax,
		Orientation: o.Orientation, Tilt: o.Tilt, Synthesized: o.Synthesized, EmbedderID: o.EmbedderID}
}

// hoverCopy is the copyWith of the hover, enter and exit events (pointer
// and down are not kept, as in Dart).
func hoverCopy(e PointerEvent, o PointerEventCopyWithOptions) PointerHoverEventOptions {
	return PointerHoverEventOptions{
		ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Kind: orKind(o.Kind, e.Kind()), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), Delta: orOffset(o.Delta, e.Delta()),
		Buttons: orInt(o.Buttons, e.Buttons()), Obscured: orBool(o.Obscured, e.Obscured()),
		PressureMin: ptr(orDouble(o.PressureMin, e.PressureMin())), PressureMax: ptr(orDouble(o.PressureMax, e.PressureMax())),
		Distance: orDouble(o.Distance, e.Distance()), DistanceMax: orDouble(o.DistanceMax, e.DistanceMax()),
		Size: orDouble(o.Size, e.Size()), RadiusMajor: orDouble(o.RadiusMajor, e.RadiusMajor()),
		RadiusMinor: orDouble(o.RadiusMinor, e.RadiusMinor()), RadiusMin: orDouble(o.RadiusMin, e.RadiusMin()),
		RadiusMax: orDouble(o.RadiusMax, e.RadiusMax()), Orientation: orDouble(o.Orientation, e.Orientation()),
		Tilt: orDouble(o.Tilt, e.Tilt()), Synthesized: orBool(o.Synthesized, e.Synthesized()),
		EmbedderID: orInt(o.EmbedderID, e.EmbedderID()),
	}
}

// PointerHoverEvent is sent when the pointer has moved without being in
// contact with the device.
type PointerHoverEvent struct{ PointerEventBase }

// NewPointerHoverEvent creates a PointerHoverEvent.
func NewPointerHoverEvent(opts ...PointerHoverEventOptions) *PointerHoverEvent {
	var o PointerHoverEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerHoverEvent{}
	e.InitPointerEvent(e, o.base(false))
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerHoverEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerHoverEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (as Dart's, `pointer` is not kept).
func (e *PointerHoverEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	return NewPointerHoverEvent(hoverCopy(e, copyOpts(opts))).Transformed(e.Transform())
}

// PointerEnterEventOptions are PointerEnterEvent's and PointerExitEvent's
// parameters.
type PointerEnterEventOptions struct {
	PointerHoverEventOptions
	Down bool
}

// PointerEnterEvent is sent when the pointer has entered an annotated
// region (synthesized by the mouse tracker).
type PointerEnterEvent struct{ PointerEventBase }

// NewPointerEnterEvent creates a PointerEnterEvent (not for trackpads).
func NewPointerEnterEvent(opts ...PointerEnterEventOptions) *PointerEnterEvent {
	var o PointerEnterEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.Kind != ui.PointerDeviceKindTrackpad, nil)
	e := &PointerEnterEvent{}
	e.InitPointerEvent(e, o.base(o.Down))
	return e
}

// fromMouseEventOptions are a hover-like event's fields from event (Dart's
// fromMouseEvent factories: no embedderId).
func fromMouseEventOptions(event PointerEvent) PointerEnterEventOptions {
	return PointerEnterEventOptions{PointerHoverEventOptions: PointerHoverEventOptions{
		ViewID: event.ViewID(), TimeStamp: event.TimeStamp(), Pointer: event.Pointer(), Kind: event.Kind(),
		Device: event.Device(), Position: event.Position(), Delta: event.Delta(), Buttons: event.Buttons(),
		Obscured: event.Obscured(), PressureMin: ptr(event.PressureMin()), PressureMax: ptr(event.PressureMax()),
		Distance: event.Distance(), DistanceMax: event.DistanceMax(), Size: event.Size(),
		RadiusMajor: event.RadiusMajor(), RadiusMinor: event.RadiusMinor(), RadiusMin: event.RadiusMin(),
		RadiusMax: event.RadiusMax(), Orientation: event.Orientation(), Tilt: event.Tilt(),
		Synthesized: event.Synthesized(),
	}, Down: event.Down()}
}

// NewPointerEnterEventFromMouseEvent is PointerEnterEvent.fromMouseEvent:
// an enter event with event's fields, transformed by its transform.
func NewPointerEnterEventFromMouseEvent(event PointerEvent) PointerEvent {
	return NewPointerEnterEvent(fromMouseEventOptions(event)).Transformed(event.Transform())
}

// Transformed returns the event transformed by transform.
func (e *PointerEnterEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerEnterEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (as Dart's, `pointer` and `down` are not kept).
func (e *PointerEnterEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	return NewPointerEnterEvent(PointerEnterEventOptions{PointerHoverEventOptions: hoverCopy(e, copyOpts(opts))}).
		Transformed(e.Transform())
}

// PointerExitEvent is sent when the pointer has exited an annotated region
// (synthesized by the mouse tracker).
type PointerExitEvent struct{ PointerEventBase }

// NewPointerExitEvent creates a PointerExitEvent (not for trackpads).
func NewPointerExitEvent(opts ...PointerEnterEventOptions) *PointerExitEvent {
	var o PointerEnterEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.Kind != ui.PointerDeviceKindTrackpad, nil)
	e := &PointerExitEvent{}
	e.InitPointerEvent(e, o.base(o.Down))
	return e
}

// NewPointerExitEventFromMouseEvent is PointerExitEvent.fromMouseEvent.
func NewPointerExitEventFromMouseEvent(event PointerEvent) PointerEvent {
	return NewPointerExitEvent(fromMouseEventOptions(event)).Transformed(event.Transform())
}

// Transformed returns the event transformed by transform.
func (e *PointerExitEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerExitEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (as Dart's, `pointer` and `down` are not kept).
func (e *PointerExitEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	return NewPointerExitEvent(PointerEnterEventOptions{PointerHoverEventOptions: hoverCopy(e, copyOpts(opts))}).
		Transformed(e.Transform())
}

// ---- PointerDownEvent

// PointerDownEventOptions are PointerDownEvent's parameters.
type PointerDownEventOptions struct {
	ViewID      int
	TimeStamp   time.Duration
	Pointer     int
	Kind        ui.PointerDeviceKind
	Device      int
	Position    ui.Offset
	Buttons     *int // default KPrimaryButton
	Obscured    bool
	Pressure    *float64 // default 1.0
	PressureMin *float64 // default 1.0
	PressureMax *float64 // default 1.0
	DistanceMax float64
	Size        float64
	RadiusMajor float64
	RadiusMinor float64
	RadiusMin   float64
	RadiusMax   float64
	Orientation float64
	Tilt        float64
	EmbedderID  int
}

// PointerDownEvent is sent when the pointer has made contact with the
// device.
type PointerDownEvent struct{ PointerEventBase }

// NewPointerDownEvent creates a PointerDownEvent (not for trackpads).
func NewPointerDownEvent(opts ...PointerDownEventOptions) *PointerDownEvent {
	var o PointerDownEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.Kind != ui.PointerDeviceKindTrackpad, nil)
	e := &PointerDownEvent{}
	e.InitPointerEvent(e, PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Pointer: o.Pointer,
		Kind: o.Kind, Device: o.Device, Position: o.Position, Buttons: orInt(o.Buttons, KPrimaryButton),
		Obscured: o.Obscured, Pressure: o.Pressure, PressureMin: o.PressureMin, PressureMax: o.PressureMax,
		DistanceMax: o.DistanceMax, Size: o.Size, RadiusMajor: o.RadiusMajor, RadiusMinor: o.RadiusMinor,
		RadiusMin: o.RadiusMin, RadiusMax: o.RadiusMax, Orientation: o.Orientation, Tilt: o.Tilt,
		EmbedderID: o.EmbedderID, Down: true, Distance: 0.0})
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerDownEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerDownEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event.
func (e *PointerDownEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerDownEvent(PointerDownEventOptions{
		ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Pointer: orInt(o.Pointer, e.Pointer()), Kind: orKind(o.Kind, e.Kind()), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), Buttons: ptr(orInt(o.Buttons, e.Buttons())),
		Obscured: orBool(o.Obscured, e.Obscured()), Pressure: ptr(orDouble(o.Pressure, e.Pressure())),
		PressureMin: ptr(orDouble(o.PressureMin, e.PressureMin())), PressureMax: ptr(orDouble(o.PressureMax, e.PressureMax())),
		DistanceMax: orDouble(o.DistanceMax, e.DistanceMax()), Size: orDouble(o.Size, e.Size()),
		RadiusMajor: orDouble(o.RadiusMajor, e.RadiusMajor()), RadiusMinor: orDouble(o.RadiusMinor, e.RadiusMinor()),
		RadiusMin: orDouble(o.RadiusMin, e.RadiusMin()), RadiusMax: orDouble(o.RadiusMax, e.RadiusMax()),
		Orientation: orDouble(o.Orientation, e.Orientation()), Tilt: orDouble(o.Tilt, e.Tilt()),
		EmbedderID: orInt(o.EmbedderID, e.EmbedderID()),
	}).Transformed(e.Transform())
}

// ---- PointerMoveEvent

// PointerMoveEventOptions are PointerMoveEvent's parameters.
type PointerMoveEventOptions struct {
	ViewID       int
	TimeStamp    time.Duration
	Pointer      int
	Kind         ui.PointerDeviceKind
	Device       int
	Position     ui.Offset
	Delta        ui.Offset
	Buttons      *int // default KPrimaryButton
	Obscured     bool
	Pressure     *float64 // default 1.0
	PressureMin  *float64 // default 1.0
	PressureMax  *float64 // default 1.0
	DistanceMax  float64
	Size         float64
	RadiusMajor  float64
	RadiusMinor  float64
	RadiusMin    float64
	RadiusMax    float64
	Orientation  float64
	Tilt         float64
	PlatformData int
	Synthesized  bool
	EmbedderID   int
}

// PointerMoveEvent is sent when the pointer has moved while in contact
// with the device.
type PointerMoveEvent struct{ PointerEventBase }

// NewPointerMoveEvent creates a PointerMoveEvent (not for trackpads).
func NewPointerMoveEvent(opts ...PointerMoveEventOptions) *PointerMoveEvent {
	var o PointerMoveEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.Kind != ui.PointerDeviceKindTrackpad, nil)
	e := &PointerMoveEvent{}
	e.InitPointerEvent(e, PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Pointer: o.Pointer,
		Kind: o.Kind, Device: o.Device, Position: o.Position, Delta: o.Delta, Buttons: orInt(o.Buttons, KPrimaryButton),
		Obscured: o.Obscured, Pressure: o.Pressure, PressureMin: o.PressureMin, PressureMax: o.PressureMax,
		DistanceMax: o.DistanceMax, Size: o.Size, RadiusMajor: o.RadiusMajor, RadiusMinor: o.RadiusMinor,
		RadiusMin: o.RadiusMin, RadiusMax: o.RadiusMax, Orientation: o.Orientation, Tilt: o.Tilt,
		PlatformData: o.PlatformData, Synthesized: o.Synthesized, EmbedderID: o.EmbedderID,
		Down: true, Distance: 0.0})
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerMoveEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerMoveEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (as Dart's, `platformData` is not kept).
func (e *PointerMoveEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerMoveEvent(PointerMoveEventOptions{
		ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Pointer: orInt(o.Pointer, e.Pointer()), Kind: orKind(o.Kind, e.Kind()), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), Delta: orOffset(o.Delta, e.Delta()),
		Buttons: ptr(orInt(o.Buttons, e.Buttons())), Obscured: orBool(o.Obscured, e.Obscured()),
		Pressure:    ptr(orDouble(o.Pressure, e.Pressure())),
		PressureMin: ptr(orDouble(o.PressureMin, e.PressureMin())), PressureMax: ptr(orDouble(o.PressureMax, e.PressureMax())),
		DistanceMax: orDouble(o.DistanceMax, e.DistanceMax()), Size: orDouble(o.Size, e.Size()),
		RadiusMajor: orDouble(o.RadiusMajor, e.RadiusMajor()), RadiusMinor: orDouble(o.RadiusMinor, e.RadiusMinor()),
		RadiusMin: orDouble(o.RadiusMin, e.RadiusMin()), RadiusMax: orDouble(o.RadiusMax, e.RadiusMax()),
		Orientation: orDouble(o.Orientation, e.Orientation()), Tilt: orDouble(o.Tilt, e.Tilt()),
		Synthesized: orBool(o.Synthesized, e.Synthesized()), EmbedderID: orInt(o.EmbedderID, e.EmbedderID()),
	}).Transformed(e.Transform())
}

// ---- PointerUpEvent

// PointerUpEventOptions are PointerUpEvent's parameters.
type PointerUpEventOptions struct {
	ViewID      int
	TimeStamp   time.Duration
	Pointer     int
	Kind        ui.PointerDeviceKind
	Device      int
	Position    ui.Offset
	Buttons     int
	Obscured    bool
	Pressure    float64  // default 0.0 (an up event can carry a pressure)
	PressureMin *float64 // default 1.0
	PressureMax *float64 // default 1.0
	Distance    float64
	DistanceMax float64
	Size        float64
	RadiusMajor float64
	RadiusMinor float64
	RadiusMin   float64
	RadiusMax   float64
	Orientation float64
	Tilt        float64
	EmbedderID  int
}

// PointerUpEvent is sent when the pointer has stopped contacting the
// device.
type PointerUpEvent struct{ PointerEventBase }

// NewPointerUpEvent creates a PointerUpEvent (not for trackpads).
func NewPointerUpEvent(opts ...PointerUpEventOptions) *PointerUpEvent {
	var o PointerUpEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.Kind != ui.PointerDeviceKindTrackpad, nil)
	e := &PointerUpEvent{}
	e.InitPointerEvent(e, PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Pointer: o.Pointer,
		Kind: o.Kind, Device: o.Device, Position: o.Position, Buttons: o.Buttons, Obscured: o.Obscured,
		Pressure: ptr(o.Pressure), PressureMin: o.PressureMin, PressureMax: o.PressureMax,
		Distance: o.Distance, DistanceMax: o.DistanceMax, Size: o.Size, RadiusMajor: o.RadiusMajor,
		RadiusMinor: o.RadiusMinor, RadiusMin: o.RadiusMin, RadiusMax: o.RadiusMax,
		Orientation: o.Orientation, Tilt: o.Tilt, EmbedderID: o.EmbedderID, Down: false})
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerUpEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerUpEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event.
func (e *PointerUpEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerUpEvent(PointerUpEventOptions{
		ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Pointer: orInt(o.Pointer, e.Pointer()), Kind: orKind(o.Kind, e.Kind()), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), Buttons: orInt(o.Buttons, e.Buttons()),
		Obscured: orBool(o.Obscured, e.Obscured()), Pressure: orDouble(o.Pressure, e.Pressure()),
		PressureMin: ptr(orDouble(o.PressureMin, e.PressureMin())), PressureMax: ptr(orDouble(o.PressureMax, e.PressureMax())),
		Distance: orDouble(o.Distance, e.Distance()), DistanceMax: orDouble(o.DistanceMax, e.DistanceMax()),
		Size: orDouble(o.Size, e.Size()), RadiusMajor: orDouble(o.RadiusMajor, e.RadiusMajor()),
		RadiusMinor: orDouble(o.RadiusMinor, e.RadiusMinor()), RadiusMin: orDouble(o.RadiusMin, e.RadiusMin()),
		RadiusMax: orDouble(o.RadiusMax, e.RadiusMax()), Orientation: orDouble(o.Orientation, e.Orientation()),
		Tilt: orDouble(o.Tilt, e.Tilt()), EmbedderID: orInt(o.EmbedderID, e.EmbedderID()),
	}).Transformed(e.Transform())
}

// ---- PointerSignalEvent

// PointerSignalEvent is an event that corresponds to a discrete pointer
// signal (scroll, scroll inertia cancel, scale). Dispatch it to a
// PointerSignalResolver.
type PointerSignalEvent interface {
	PointerEvent
	// Respond lets the platform know whether its default action may run
	// for the event (the _RespondablePointerEvent mixin; it does nothing
	// unless the event carries a callback).
	Respond(allowPlatformDefault bool)
	pointerSignalEvent()
}

// RespondPointerEventCallback is the signature of a PointerScrollEvent's
// onRespond (Dart's required named allowPlatformDefault is positional).
type RespondPointerEventCallback func(allowPlatformDefault bool)

// signalEventOptions are PointerSignalEvent's parameters (Kind nil: mouse).
func signalEventOptions(viewID int, timeStamp time.Duration, kind *ui.PointerDeviceKind, device int,
	position ui.Offset, embedderID int) PointerEventOptions {
	k := ui.PointerDeviceKindMouse
	if kind != nil {
		k = *kind
	}
	return PointerEventOptions{ViewID: viewID, TimeStamp: timeStamp, Kind: k, Device: device,
		Position: position, EmbedderID: embedderID}
}

// PointerSignalEventOptions are the parameters of PointerScrollEvent,
// PointerScrollInertiaCancelEvent and PointerScaleEvent.
type PointerSignalEventOptions struct {
	ViewID     int
	TimeStamp  time.Duration
	Kind       *ui.PointerDeviceKind // default mouse
	Device     int
	Position   ui.Offset
	EmbedderID int
}

func (o PointerSignalEventOptions) base() PointerEventOptions {
	return signalEventOptions(o.ViewID, o.TimeStamp, o.Kind, o.Device, o.Position, o.EmbedderID)
}

func signalCopy(e PointerEvent, o PointerEventCopyWithOptions) PointerSignalEventOptions {
	return PointerSignalEventOptions{ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Kind: ptr(orKind(o.Kind, e.Kind())), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), EmbedderID: orInt(o.EmbedderID, e.EmbedderID())}
}

// PointerScrollEventOptions are PointerScrollEvent's parameters.
type PointerScrollEventOptions struct {
	PointerSignalEventOptions
	ScrollDelta ui.Offset
	OnRespond   RespondPointerEventCallback
}

// PointerScrollEvent is a pointer scroll (mouse wheel) event.
type PointerScrollEvent struct {
	PointerEventBase
	scrollDelta ui.Offset
	onRespond   RespondPointerEventCallback
}

// NewPointerScrollEvent creates a PointerScrollEvent.
func NewPointerScrollEvent(opts ...PointerScrollEventOptions) *PointerScrollEvent {
	var o PointerScrollEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerScrollEvent{scrollDelta: o.ScrollDelta, onRespond: o.OnRespond}
	e.InitPointerEvent(e, o.base())
	return e
}

// ScrollDelta is the amount to scroll, in logical pixels.
func (e *PointerScrollEvent) ScrollDelta() ui.Offset { return e.scrollDelta }

func (e *PointerScrollEvent) pointerSignalEvent() {}

// Respond calls the event's onRespond (a transformed event responds
// through its original).
func (e *PointerScrollEvent) Respond(allowPlatformDefault bool) {
	if e.isTransformed {
		e.original.(*PointerScrollEvent).Respond(allowPlatformDefault)
		return
	}
	if e.onRespond != nil {
		e.onRespond(allowPlatformDefault)
	}
}

// DebugFillProperties adds scrollDelta.
func (e *PointerScrollEvent) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	e.PointerEventBase.DebugFillProperties(properties)
	properties.Add(foundation.NewDiagnosticsProperty("scrollDelta", e.scrollDelta))
}

// Transformed returns the event transformed by transform.
func (e *PointerScrollEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerScrollEvent{scrollDelta: e.scrollDelta}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (as Dart's, scrollDelta is kept; the copy
// responds through this event unless OnRespond is given).
func (e *PointerScrollEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	onRespond := o.OnRespond
	if onRespond == nil {
		onRespond = e.Respond
	}
	return NewPointerScrollEvent(PointerScrollEventOptions{PointerSignalEventOptions: signalCopy(e, o),
		ScrollDelta: e.scrollDelta, OnRespond: onRespond}).Transformed(e.Transform())
}

// PointerScrollInertiaCancelEvent is a request to cancel the platform's
// scroll inertia (macOS trackpads).
type PointerScrollInertiaCancelEvent struct{ PointerEventBase }

// NewPointerScrollInertiaCancelEvent creates a
// PointerScrollInertiaCancelEvent.
func NewPointerScrollInertiaCancelEvent(opts ...PointerSignalEventOptions) *PointerScrollInertiaCancelEvent {
	var o PointerSignalEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerScrollInertiaCancelEvent{}
	e.InitPointerEvent(e, o.base())
	return e
}

func (e *PointerScrollInertiaCancelEvent) pointerSignalEvent() {}

// Respond does nothing.
func (e *PointerScrollInertiaCancelEvent) Respond(allowPlatformDefault bool) {}

// Transformed returns the event transformed by transform.
func (e *PointerScrollInertiaCancelEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerScrollInertiaCancelEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event.
func (e *PointerScrollInertiaCancelEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	return NewPointerScrollInertiaCancelEvent(signalCopy(e, copyOpts(opts))).Transformed(e.Transform())
}

// PointerScaleEventOptions are PointerScaleEvent's parameters.
type PointerScaleEventOptions struct {
	PointerSignalEventOptions
	Scale *float64 // default 1.0
}

// PointerScaleEvent is a pointer signal to scale (zoom) the content.
type PointerScaleEvent struct {
	PointerEventBase
	scale float64
}

// NewPointerScaleEvent creates a PointerScaleEvent.
func NewPointerScaleEvent(opts ...PointerScaleEventOptions) *PointerScaleEvent {
	var o PointerScaleEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerScaleEvent{scale: orDouble(o.Scale, 1.0)}
	e.InitPointerEvent(e, o.base())
	return e
}

// Scale is the scale factor (1.0: no change).
func (e *PointerScaleEvent) Scale() float64 { return e.scale }

func (e *PointerScaleEvent) pointerSignalEvent() {}

// Respond does nothing.
func (e *PointerScaleEvent) Respond(allowPlatformDefault bool) {}

// Transformed returns the event transformed by transform.
func (e *PointerScaleEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerScaleEvent{scale: e.scale}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event.
func (e *PointerScaleEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerScaleEvent(PointerScaleEventOptions{PointerSignalEventOptions: signalCopy(e, o),
		Scale: ptr(orDouble(o.Scale, e.scale))}).Transformed(e.Transform())
}

// ---- pan/zoom events (trackpads)

// PointerPanZoomEventOptions are the parameters of PointerPanZoomStartEvent
// and PointerPanZoomEndEvent (the kind is always trackpad).
type PointerPanZoomEventOptions struct {
	ViewID      int
	TimeStamp   time.Duration
	Device      int
	Pointer     int
	Position    ui.Offset
	EmbedderID  int
	Synthesized bool
}

func (o PointerPanZoomEventOptions) base() PointerEventOptions {
	return PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Device: o.Device, Pointer: o.Pointer,
		Position: o.Position, EmbedderID: o.EmbedderID, Synthesized: o.Synthesized, Kind: ui.PointerDeviceKindTrackpad}
}

// panZoomCopy is the pan/zoom events' copyWith (pointer and synthesized are
// not kept, as in Dart).
func panZoomCopy(e PointerEvent, o PointerEventCopyWithOptions) PointerPanZoomEventOptions {
	dartAssert(o.Kind == nil || *o.Kind == ui.PointerDeviceKindTrackpad, nil)
	return PointerPanZoomEventOptions{ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Device: orInt(o.Device, e.Device()), Position: orOffset(o.Position, e.Position()),
		EmbedderID: orInt(o.EmbedderID, e.EmbedderID())}
}

// PointerPanZoomStartEvent is sent when a pan/zoom gesture starts on a
// trackpad.
type PointerPanZoomStartEvent struct{ PointerEventBase }

// NewPointerPanZoomStartEvent creates a PointerPanZoomStartEvent.
func NewPointerPanZoomStartEvent(opts ...PointerPanZoomEventOptions) *PointerPanZoomStartEvent {
	var o PointerPanZoomEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerPanZoomStartEvent{}
	e.InitPointerEvent(e, o.base())
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerPanZoomStartEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerPanZoomStartEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (Kind, if given, must be trackpad).
func (e *PointerPanZoomStartEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	return NewPointerPanZoomStartEvent(panZoomCopy(e, copyOpts(opts))).Transformed(e.Transform())
}

// PointerPanZoomUpdateEventOptions are PointerPanZoomUpdateEvent's
// parameters.
type PointerPanZoomUpdateEventOptions struct {
	PointerPanZoomEventOptions
	Pan      ui.Offset
	PanDelta ui.Offset
	Scale    *float64 // default 1.0
	Rotation float64
}

// PointerPanZoomUpdateEvent is sent when a trackpad pan/zoom gesture
// updates.
type PointerPanZoomUpdateEvent struct {
	PointerEventBase
	pan      ui.Offset
	panDelta ui.Offset
	scale    float64
	rotation float64
	// Transformed events: late finals.
	localPan      *ui.Offset
	localPanDelta *ui.Offset
}

// NewPointerPanZoomUpdateEvent creates a PointerPanZoomUpdateEvent.
func NewPointerPanZoomUpdateEvent(opts ...PointerPanZoomUpdateEventOptions) *PointerPanZoomUpdateEvent {
	var o PointerPanZoomUpdateEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerPanZoomUpdateEvent{pan: o.Pan, panDelta: o.PanDelta, scale: orDouble(o.Scale, 1.0), rotation: o.Rotation}
	e.InitPointerEvent(e, o.base())
	return e
}

// Pan is the total pan offset of the gesture (global coordinates).
func (e *PointerPanZoomUpdateEvent) Pan() ui.Offset { return e.pan }

// LocalPan is Pan transformed by Transform.
func (e *PointerPanZoomUpdateEvent) LocalPan() ui.Offset {
	if !e.isTransformed {
		return e.pan
	}
	if e.localPan == nil {
		p := PointerEventTransformPosition(e.transform, e.pan)
		e.localPan = &p
	}
	return *e.localPan
}

// PanDelta is the amount the pan offset changed since the last event.
func (e *PointerPanZoomUpdateEvent) PanDelta() ui.Offset { return e.panDelta }

// LocalPanDelta is PanDelta transformed by Transform.
func (e *PointerPanZoomUpdateEvent) LocalPanDelta() ui.Offset {
	if !e.isTransformed {
		return e.panDelta
	}
	if e.localPanDelta == nil {
		localPan := e.LocalPan()
		d := PointerEventTransformDeltaViaPositions(e.pan, &localPan, e.panDelta, e.transform)
		e.localPanDelta = &d
	}
	return *e.localPanDelta
}

// Scale is the scale (zoom) factor since the start of the gesture.
func (e *PointerPanZoomUpdateEvent) Scale() float64 { return e.scale }

// Rotation is the rotation in radians since the start of the gesture.
func (e *PointerPanZoomUpdateEvent) Rotation() float64 { return e.rotation }

// Transformed returns the event transformed by transform.
func (e *PointerPanZoomUpdateEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerPanZoomUpdateEvent{pan: e.pan, panDelta: e.panDelta, scale: e.scale, rotation: e.rotation}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (Kind, if given, must be trackpad; LocalPan and
// LocalPanDelta are ignored, as in Dart).
func (e *PointerPanZoomUpdateEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerPanZoomUpdateEvent(PointerPanZoomUpdateEventOptions{
		PointerPanZoomEventOptions: panZoomCopy(e, o),
		Pan:                        orOffset(o.Pan, e.pan), PanDelta: orOffset(o.PanDelta, e.panDelta),
		Scale: ptr(orDouble(o.Scale, e.scale)), Rotation: orDouble(o.Rotation, e.rotation),
	}).Transformed(e.Transform())
}

// PointerPanZoomEndEvent is sent when a trackpad pan/zoom gesture ends.
type PointerPanZoomEndEvent struct{ PointerEventBase }

// NewPointerPanZoomEndEvent creates a PointerPanZoomEndEvent.
func NewPointerPanZoomEndEvent(opts ...PointerPanZoomEventOptions) *PointerPanZoomEndEvent {
	var o PointerPanZoomEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	e := &PointerPanZoomEndEvent{}
	e.InitPointerEvent(e, o.base())
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerPanZoomEndEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerPanZoomEndEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event (Kind, if given, must be trackpad).
func (e *PointerPanZoomEndEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	return NewPointerPanZoomEndEvent(panZoomCopy(e, copyOpts(opts))).Transformed(e.Transform())
}

// ---- PointerCancelEvent

// PointerCancelEventOptions are PointerCancelEvent's parameters.
type PointerCancelEventOptions struct {
	ViewID      int
	TimeStamp   time.Duration
	Pointer     int
	Kind        ui.PointerDeviceKind
	Device      int
	Position    ui.Offset
	Buttons     int
	Obscured    bool
	PressureMin *float64 // default 1.0
	PressureMax *float64 // default 1.0
	Distance    float64
	DistanceMax float64
	Size        float64
	RadiusMajor float64
	RadiusMinor float64
	RadiusMin   float64
	RadiusMax   float64
	Orientation float64
	Tilt        float64
	EmbedderID  int
}

// PointerCancelEvent is sent when the input from a pointer is no longer
// directed towards this app.
type PointerCancelEvent struct{ PointerEventBase }

// NewPointerCancelEvent creates a PointerCancelEvent (not for trackpads).
func NewPointerCancelEvent(opts ...PointerCancelEventOptions) *PointerCancelEvent {
	var o PointerCancelEventOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.Kind != ui.PointerDeviceKindTrackpad, nil)
	e := &PointerCancelEvent{}
	e.InitPointerEvent(e, PointerEventOptions{ViewID: o.ViewID, TimeStamp: o.TimeStamp, Pointer: o.Pointer,
		Kind: o.Kind, Device: o.Device, Position: o.Position, Buttons: o.Buttons, Obscured: o.Obscured,
		PressureMin: o.PressureMin, PressureMax: o.PressureMax, Distance: o.Distance, DistanceMax: o.DistanceMax,
		Size: o.Size, RadiusMajor: o.RadiusMajor, RadiusMinor: o.RadiusMinor, RadiusMin: o.RadiusMin,
		RadiusMax: o.RadiusMax, Orientation: o.Orientation, Tilt: o.Tilt, EmbedderID: o.EmbedderID,
		Down: false, Pressure: ptr(0.0)})
	return e
}

// Transformed returns the event transformed by transform.
func (e *PointerCancelEvent) Transformed(transform *vm.Matrix4) PointerEvent {
	if e.isTransformed {
		return e.original.Transformed(transform)
	}
	if !e.shouldTransform(transform) {
		return e
	}
	t := &PointerCancelEvent{}
	t.transformedFrom(t, e.untransformed(), transform)
	return t
}

// CopyWith copies the event.
func (e *PointerCancelEvent) CopyWith(opts ...PointerEventCopyWithOptions) PointerEvent {
	o := copyOpts(opts)
	return NewPointerCancelEvent(PointerCancelEventOptions{
		ViewID: orInt(o.ViewID, e.ViewID()), TimeStamp: orDuration(o.TimeStamp, e.TimeStamp()),
		Pointer: orInt(o.Pointer, e.Pointer()), Kind: orKind(o.Kind, e.Kind()), Device: orInt(o.Device, e.Device()),
		Position: orOffset(o.Position, e.Position()), Buttons: orInt(o.Buttons, e.Buttons()),
		Obscured:    orBool(o.Obscured, e.Obscured()),
		PressureMin: ptr(orDouble(o.PressureMin, e.PressureMin())), PressureMax: ptr(orDouble(o.PressureMax, e.PressureMax())),
		Distance: orDouble(o.Distance, e.Distance()), DistanceMax: orDouble(o.DistanceMax, e.DistanceMax()),
		Size: orDouble(o.Size, e.Size()), RadiusMajor: orDouble(o.RadiusMajor, e.RadiusMajor()),
		RadiusMinor: orDouble(o.RadiusMinor, e.RadiusMinor()), RadiusMin: orDouble(o.RadiusMin, e.RadiusMin()),
		RadiusMax: orDouble(o.RadiusMax, e.RadiusMax()), Orientation: orDouble(o.Orientation, e.Orientation()),
		Tilt: orDouble(o.Tilt, e.Tilt()), EmbedderID: orInt(o.EmbedderID, e.EmbedderID()),
	}).Transformed(e.Transform())
}

// ---- slops

// ComputeHitSlop determines the hit slop of a pointer kind (settings nil:
// the defaults).
func ComputeHitSlop(kind ui.PointerDeviceKind, settings *DeviceGestureSettings) float64 {
	switch kind {
	case ui.PointerDeviceKindMouse:
		return KPrecisePointerHitSlop
	}
	if settings != nil && settings.TouchSlop != nil {
		return *settings.TouchSlop
	}
	return KTouchSlop
}

// ComputePanSlop determines the pan slop of a pointer kind (settings nil:
// the defaults).
func ComputePanSlop(kind ui.PointerDeviceKind, settings *DeviceGestureSettings) float64 {
	switch kind {
	case ui.PointerDeviceKindMouse:
		return KPrecisePointerPanSlop
	}
	if settings != nil {
		if s := settings.PanSlop(); s != nil {
			return *s
		}
	}
	return KPanSlop
}

// ComputeScaleSlop determines the scale slop of a pointer kind.
func ComputeScaleSlop(kind ui.PointerDeviceKind) float64 {
	switch kind {
	case ui.PointerDeviceKindMouse:
		return KPrecisePointerScaleSlop
	}
	return KScaleSlop
}
