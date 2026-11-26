package ui

// Port of engine/src/flutter/lib/ui/pointer.dart (Flutter 3.50.1): the raw
// pointer data the engine hands the framework (PointerData, in physical
// pixels), its enums and PointerDataPacket. The engine side
// (PlatformDispatcher.onPointerDataPacket and the packet's byte layout) is
// in pointer_dispatch.go.

import (
	"strconv"
	"time"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// PointerChange is how the pointer has changed since the last report.
type PointerChange int

const (
	// PointerChangeCancel: the input from the pointer is no longer directed
	// towards this receiver.
	PointerChangeCancel PointerChange = iota
	// PointerChangeAdd: the device has started tracking the pointer.
	PointerChangeAdd
	// PointerChangeRemove: the device is no longer tracking the pointer.
	PointerChangeRemove
	// PointerChangeHover: the pointer has moved with respect to the device
	// while not in contact with it.
	PointerChangeHover
	// PointerChangeDown: the pointer has made contact with the device.
	PointerChangeDown
	// PointerChangeMove: the pointer has moved while in contact.
	PointerChangeMove
	// PointerChangeUp: the pointer has stopped making contact.
	PointerChangeUp
	// PointerChangePanZoomStart: a pan/zoom has started on this pointer.
	PointerChangePanZoomStart
	// PointerChangePanZoomUpdate: the pan/zoom on this pointer has updated.
	PointerChangePanZoomUpdate
	// PointerChangePanZoomEnd: the pan/zoom on this pointer has ended.
	PointerChangePanZoomEnd
)

// PointerChangeValues is PointerChange.values.
var PointerChangeValues = []PointerChange{PointerChangeCancel, PointerChangeAdd, PointerChangeRemove,
	PointerChangeHover, PointerChangeDown, PointerChangeMove, PointerChangeUp,
	PointerChangePanZoomStart, PointerChangePanZoomUpdate, PointerChangePanZoomEnd}

// Name is the enum value's name (Dart's .name).
func (c PointerChange) Name() string {
	return [...]string{"cancel", "add", "remove", "hover", "down", "move", "up",
		"panZoomStart", "panZoomUpdate", "panZoomEnd"}[c]
}

func (c PointerChange) String() string { return "PointerChange." + c.Name() }

// PointerDeviceKind is the kind of pointer device.
type PointerDeviceKind int

const (
	// PointerDeviceKindTouch: a touch-based pointer device.
	PointerDeviceKindTouch PointerDeviceKind = iota
	// PointerDeviceKindMouse: a mouse-based pointer device.
	PointerDeviceKindMouse
	// PointerDeviceKindStylus: a pointer device with a stylus.
	PointerDeviceKindStylus
	// PointerDeviceKindInvertedStylus: a stylus with its eraser end.
	PointerDeviceKindInvertedStylus
	// PointerDeviceKindTrackpad: gestures from a touchpad.
	PointerDeviceKindTrackpad
	// PointerDeviceKindUnknown: an unknown pointer device.
	PointerDeviceKindUnknown
)

// PointerDeviceKindValues is PointerDeviceKind.values.
var PointerDeviceKindValues = []PointerDeviceKind{PointerDeviceKindTouch, PointerDeviceKindMouse,
	PointerDeviceKindStylus, PointerDeviceKindInvertedStylus, PointerDeviceKindTrackpad, PointerDeviceKindUnknown}

// Name is the enum value's name (Dart's .name).
func (k PointerDeviceKind) Name() string {
	return [...]string{"touch", "mouse", "stylus", "invertedStylus", "trackpad", "unknown"}[k]
}

func (k PointerDeviceKind) String() string { return "PointerDeviceKind." + k.Name() }

// PointerSignalKind is the kind of pointer signal event.
type PointerSignalKind int

const (
	// PointerSignalKindNone: the event is not associated with a signal.
	PointerSignalKindNone PointerSignalKind = iota
	// PointerSignalKindScroll: a scroll (mouse wheel) event.
	PointerSignalKindScroll
	// PointerSignalKindScrollInertiaCancel: a request to cancel the
	// platform's scroll inertia.
	PointerSignalKindScrollInertiaCancel
	// PointerSignalKindScale: a pinch-to-zoom signal.
	PointerSignalKindScale
	// PointerSignalKindUnknown: an unknown signal.
	PointerSignalKindUnknown
)

// PointerSignalKindValues is PointerSignalKind.values.
var PointerSignalKindValues = []PointerSignalKind{PointerSignalKindNone, PointerSignalKindScroll,
	PointerSignalKindScrollInertiaCancel, PointerSignalKindScale, PointerSignalKindUnknown}

// Name is the enum value's name (Dart's .name).
func (k PointerSignalKind) Name() string {
	return [...]string{"none", "scroll", "scrollInertiaCancel", "scale", "unknown"}[k]
}

func (k PointerSignalKind) String() string { return "PointerSignalKind." + k.Name() }

// PointerDataRespondCallback is the signature of PointerData's respond
// callback (Dart's named allowPlatformDefault is positional).
type PointerDataRespondCallback func(allowPlatformDefault bool)

// PointerData is information about the state of a pointer, in physical
// pixels. Every field's zero value is its Dart default (SignalKind nil is
// null). OnRespond is the constructor's private onRespond.
type PointerData struct {
	ViewID            int
	EmbedderID        int
	TimeStamp         time.Duration
	Change            PointerChange
	Kind              PointerDeviceKind
	SignalKind        *PointerSignalKind
	Device            int
	PointerIdentifier int
	PhysicalX         float64
	PhysicalY         float64
	PhysicalDeltaX    float64
	PhysicalDeltaY    float64
	Buttons           int
	Obscured          bool
	Synthesized       bool
	Pressure          float64
	PressureMin       float64
	PressureMax       float64
	Distance          float64
	DistanceMax       float64
	Size              float64
	RadiusMajor       float64
	RadiusMinor       float64
	RadiusMin         float64
	RadiusMax         float64
	Orientation       float64
	Tilt              float64
	PlatformData      int
	ScrollDeltaX      float64
	ScrollDeltaY      float64
	PanX              float64
	PanY              float64
	PanDeltaX         float64
	PanDeltaY         float64
	Scale             float64
	Rotation          float64
	OnRespond         PointerDataRespondCallback
}

// Respond lets the framework respond to the event that produced this
// datum (whether the platform's default action may run).
func (p PointerData) Respond(allowPlatformDefault bool) {
	if p.OnRespond != nil {
		p.OnRespond(allowPlatformDefault)
	}
}

func dbl(v float64) string { return dart.DoubleToString(v) }

func (p PointerData) String() string {
	return "PointerData(viewId: " + strconv.Itoa(p.ViewID) + ", x: " + dbl(p.PhysicalX) + ", y: " + dbl(p.PhysicalY) + ")"
}

// ToStringFull returns a string representation with all the fields.
func (p PointerData) ToStringFull() string {
	signalKind := "null"
	if p.SignalKind != nil {
		signalKind = p.SignalKind.String()
	}
	return "PointerData(" +
		"embedderId: " + strconv.Itoa(p.EmbedderID) + ", " +
		"timeStamp: " + dart.DurationToString(p.TimeStamp) + ", " +
		"change: " + p.Change.String() + ", " +
		"kind: " + p.Kind.String() + ", " +
		"signalKind: " + signalKind + ", " +
		"device: " + strconv.Itoa(p.Device) + ", " +
		"pointerIdentifier: " + strconv.Itoa(p.PointerIdentifier) + ", " +
		"physicalX: " + dbl(p.PhysicalX) + ", " +
		"physicalY: " + dbl(p.PhysicalY) + ", " +
		"physicalDeltaX: " + dbl(p.PhysicalDeltaX) + ", " +
		"physicalDeltaY: " + dbl(p.PhysicalDeltaY) + ", " +
		"buttons: " + strconv.Itoa(p.Buttons) + ", " +
		"synthesized: " + strconv.FormatBool(p.Synthesized) + ", " +
		"pressure: " + dbl(p.Pressure) + ", " +
		"pressureMin: " + dbl(p.PressureMin) + ", " +
		"pressureMax: " + dbl(p.PressureMax) + ", " +
		"distance: " + dbl(p.Distance) + ", " +
		"distanceMax: " + dbl(p.DistanceMax) + ", " +
		"size: " + dbl(p.Size) + ", " +
		"radiusMajor: " + dbl(p.RadiusMajor) + ", " +
		"radiusMinor: " + dbl(p.RadiusMinor) + ", " +
		"radiusMin: " + dbl(p.RadiusMin) + ", " +
		"radiusMax: " + dbl(p.RadiusMax) + ", " +
		"orientation: " + dbl(p.Orientation) + ", " +
		"tilt: " + dbl(p.Tilt) + ", " +
		"platformData: " + strconv.Itoa(p.PlatformData) + ", " +
		"scrollDeltaX: " + dbl(p.ScrollDeltaX) + ", " +
		"scrollDeltaY: " + dbl(p.ScrollDeltaY) + ", " +
		"panX: " + dbl(p.PanX) + ", " +
		"panY: " + dbl(p.PanY) + ", " +
		"panDeltaX: " + dbl(p.PanDeltaX) + ", " +
		"panDeltaY: " + dbl(p.PanDeltaY) + ", " +
		"scale: " + dbl(p.Scale) + ", " +
		"rotation: " + dbl(p.Rotation) + ", " +
		"viewId: " + strconv.Itoa(p.ViewID) +
		")"
}

// PointerDataPacket is a sequence of reports about the state of pointers.
type PointerDataPacket struct {
	Data []PointerData
}
