package gestures

// Port of drag_details.dart (Flutter 3.50.1).
//
// The details are Diagnosticable objects used by pointer, built with their
// constructors. globalPosition/localPosition are methods
// (PositionedGestureDetails); the other final fields are struct fields.

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// positionedDetails holds globalPosition and localPosition.
type positionedDetails struct {
	globalPosition ui.Offset
	localPosition  ui.Offset
}

// GlobalPosition is the global position at which the pointer interacted.
func (d *positionedDetails) GlobalPosition() ui.Offset { return d.globalPosition }

// LocalPosition is GlobalPosition in the receiver's coordinate system.
func (d *positionedDetails) LocalPosition() ui.Offset { return d.localPosition }

func newPositionedDetails(global ui.Offset, local *ui.Offset) positionedDetails {
	if local == nil {
		return positionedDetails{global, global}
	}
	return positionedDetails{global, *local}
}

func (d *positionedDetails) fillPositions(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.Add(foundation.NewDiagnosticsProperty("globalPosition", d.globalPosition))
	properties.Add(foundation.NewDiagnosticsProperty("localPosition", d.localPosition))
}

// sourceTimeStampProperty is DiagnosticsProperty<Duration?>: the value is
// the Duration itself (formatted as Dart's Duration) or nil (null).
func sourceTimeStampProperty(sourceTimeStamp *time.Duration) *foundation.DiagnosticsProperty[any] {
	var value any
	if sourceTimeStamp != nil {
		value = *sourceTimeStamp
	}
	return foundation.NewDiagnosticsProperty("sourceTimeStamp", value)
}

// DragDownDetailsOptions are DragDownDetails' parameters (LocalPosition nil:
// GlobalPosition).
type DragDownDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
}

// DragDownDetails are the details for GestureDragDownCallback.
type DragDownDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
}

// NewDragDownDetails creates details for a GestureDragDownCallback.
func NewDragDownDetails(opts ...DragDownDetailsOptions) *DragDownDetails {
	var o DragDownDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &DragDownDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition)}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions.
func (d *DragDownDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
}

// GestureDragDownCallback is the signature for when a pointer has
// contacted the screen and might begin to move.
type GestureDragDownCallback func(details *DragDownDetails)

// DragStartDetailsOptions are DragStartDetails' parameters (nil: null).
type DragStartDetailsOptions struct {
	GlobalPosition  ui.Offset
	LocalPosition   *ui.Offset
	SourceTimeStamp *time.Duration
	Kind            *ui.PointerDeviceKind
}

// DragStartDetails are the details for GestureDragStartCallback.
type DragStartDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// SourceTimeStamp is the time stamp of the event that started the drag
	// (nil: null).
	SourceTimeStamp *time.Duration
	// Kind is the kind of the device that initiated the event (nil: null).
	Kind *ui.PointerDeviceKind
}

// NewDragStartDetails creates details for a GestureDragStartCallback.
func NewDragStartDetails(opts ...DragStartDetailsOptions) *DragStartDetails {
	var o DragStartDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &DragStartDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition),
		SourceTimeStamp: o.SourceTimeStamp, Kind: o.Kind}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions, time stamp and kind.
func (d *DragStartDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(sourceTimeStampProperty(d.SourceTimeStamp))
	properties.Add(foundation.NewEnumProperty("kind", d.Kind))
}

// GestureDragStartCallback is the signature for when a pointer has
// contacted the screen and has begun to move.
type GestureDragStartCallback func(details *DragStartDetails)

// DragUpdateDetailsOptions are DragUpdateDetails' parameters (nil: null).
type DragUpdateDetailsOptions struct {
	GlobalPosition  ui.Offset
	LocalPosition   *ui.Offset
	SourceTimeStamp *time.Duration
	Delta           ui.Offset
	PrimaryDelta    *float64
	Kind            *ui.PointerDeviceKind
}

// DragUpdateDetails are the details for GestureDragUpdateCallback.
type DragUpdateDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// SourceTimeStamp is the time stamp of the event that triggered the
	// update (nil: null).
	SourceTimeStamp *time.Duration
	// Delta is the amount the pointer has moved since the previous update.
	Delta ui.Offset
	// PrimaryDelta is the amount moved along the primary axis, for
	// one-dimensional drags (nil: null).
	PrimaryDelta *float64
	// Kind is the kind of the device that triggered the event (nil: null).
	Kind *ui.PointerDeviceKind
}

// NewDragUpdateDetails creates details for a GestureDragUpdateCallback.
// GlobalPosition is required (Dart). A PrimaryDelta must be one of Delta's
// components, the other being zero.
func NewDragUpdateDetails(opts DragUpdateDetailsOptions) *DragUpdateDetails {
	if p := opts.PrimaryDelta; p != nil {
		dartAssert((*p == opts.Delta.Dx && opts.Delta.Dy == 0.0) || (*p == opts.Delta.Dy && opts.Delta.Dx == 0.0), nil)
	}
	d := &DragUpdateDetails{positionedDetails: newPositionedDetails(opts.GlobalPosition, opts.LocalPosition),
		SourceTimeStamp: opts.SourceTimeStamp, Delta: opts.Delta, PrimaryDelta: opts.PrimaryDelta, Kind: opts.Kind}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions, time stamp and deltas.
func (d *DragUpdateDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(sourceTimeStampProperty(d.SourceTimeStamp))
	properties.Add(foundation.NewDiagnosticsProperty("delta", d.Delta))
	properties.Add(foundation.NewDoubleProperty("primaryDelta", d.PrimaryDelta))
}

// GestureDragUpdateCallback is the signature for when a pointer that is in
// contact with the screen and moving has moved again.
type GestureDragUpdateCallback func(details *DragUpdateDetails)

// DragEndDetailsOptions are DragEndDetails' parameters (nil: null).
type DragEndDetailsOptions struct {
	GlobalPosition  ui.Offset
	LocalPosition   *ui.Offset
	Velocity        Velocity
	PrimaryVelocity *float64
}

// DragEndDetails are the details for GestureDragEndCallback.
type DragEndDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Velocity is the velocity the pointer was moving when it stopped
	// contacting the screen.
	Velocity Velocity
	// PrimaryVelocity is the velocity along the primary axis, for
	// one-dimensional drags (nil: null).
	PrimaryVelocity *float64
}

// NewDragEndDetails creates details for a GestureDragEndCallback. A
// PrimaryVelocity must be one of Velocity's components, the other being
// zero.
func NewDragEndDetails(opts ...DragEndDetailsOptions) *DragEndDetails {
	var o DragEndDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if p := o.PrimaryVelocity; p != nil {
		v := o.Velocity.PixelsPerSecond
		dartAssert((*p == v.Dx && v.Dy == 0) || (*p == v.Dy && v.Dx == 0), nil)
	}
	d := &DragEndDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition),
		Velocity: o.Velocity, PrimaryVelocity: o.PrimaryVelocity}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions and velocities.
func (d *DragEndDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
	properties.Add(foundation.NewDoubleProperty("primaryVelocity", d.PrimaryVelocity))
}
