package gestures

// Port of force_press.dart (Flutter 3.50.1).

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// forceState is _ForceState.
type forceState int

const (
	forceStateReady forceState = iota
	forceStatePossible
	forceStateAccepted
	forceStateStarted
	forceStatePeaked
)

// ForcePressDetailsOptions are ForcePressDetails' parameters
// (GlobalPosition and Pressure are required).
type ForcePressDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
	Pressure       float64
}

// ForcePressDetails are the details for the force press callbacks.
type ForcePressDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Pressure is the pressure of the pointer on the screen.
	Pressure float64
}

// NewForcePressDetails creates force press details.
func NewForcePressDetails(opts ForcePressDetailsOptions) *ForcePressDetails {
	d := &ForcePressDetails{positionedDetails: newPositionedDetails(opts.GlobalPosition, opts.LocalPosition), Pressure: opts.Pressure}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions and the pressure.
func (d *ForcePressDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewDoubleProperty("pressure", d.Pressure))
}

// GestureForcePressStartCallback is the signature used by
// ForcePressGestureRecognizer for when a pointer has pressed with at least
// StartPressure.
type GestureForcePressStartCallback func(details *ForcePressDetails)

// GestureForcePressPeakCallback is the signature used by
// ForcePressGestureRecognizer for when a pointer that has pressed with at
// least PeakPressure.
type GestureForcePressPeakCallback func(details *ForcePressDetails)

// GestureForcePressUpdateCallback is the signature used by
// ForcePressGestureRecognizer during the frames after the triggering of a
// GestureForcePressStartCallback.
type GestureForcePressUpdateCallback func(details *ForcePressDetails)

// GestureForcePressEndCallback is the signature for when the pointer that
// previously triggered a GestureForcePressStartCallback is no longer in
// contact with the screen.
type GestureForcePressEndCallback func(details *ForcePressDetails)

// GestureForceInterpolation is the signature for a function that maps a
// pressure between pressureMin and pressureMax to a value between 0 and 1.
type GestureForceInterpolation func(pressureMin, pressureMax, pressure float64) float64

// ForcePressGestureRecognizerOptions are ForcePressGestureRecognizer's
// constructor parameters (nil: 0.4, 0.85, inverse lerp).
type ForcePressGestureRecognizerOptions struct {
	GestureRecognizerOptions
	StartPressure *float64
	PeakPressure  *float64
	Interpolation GestureForceInterpolation
}

// ForcePressGestureRecognizer recognizes a force press on devices with
// pressure sensors.
type ForcePressGestureRecognizer struct {
	OneSequenceGestureRecognizerBase

	OnStart  GestureForcePressStartCallback
	OnUpdate GestureForcePressUpdateCallback
	OnPeak   GestureForcePressPeakCallback
	OnEnd    GestureForcePressEndCallback

	// StartPressure is the pressure (0..1 after interpolation) that starts
	// the gesture.
	StartPressure float64
	// PeakPressure is the pressure that calls OnPeak.
	PeakPressure float64
	// Interpolation maps the pointer's pressure to 0..1.
	Interpolation GestureForceInterpolation

	lastPosition OffsetPair
	lastPressure float64
	state        forceState
}

// NewForcePressGestureRecognizer creates a force press recognizer.
func NewForcePressGestureRecognizer(opts ...ForcePressGestureRecognizerOptions) *ForcePressGestureRecognizer {
	var o ForcePressGestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r := &ForcePressGestureRecognizer{StartPressure: 0.4, PeakPressure: 0.85, Interpolation: inverseLerp}
	if o.StartPressure != nil {
		r.StartPressure = *o.StartPressure
	}
	if o.PeakPressure != nil {
		r.PeakPressure = *o.PeakPressure
	}
	if o.Interpolation != nil {
		r.Interpolation = o.Interpolation
	}
	dartAssert(r.PeakPressure > r.StartPressure, nil)
	r.Construct(r, o.GestureRecognizerOptions)
	return r
}

func (r *ForcePressGestureRecognizer) AddAllowedPointer(event *PointerDownEvent) {
	// If the device has a maximum pressure of less than or equal to 1, it
	// doesn't have touch pressure sensing capabilities. Do not participate
	// in the gesture arena.
	if event.PressureMax() <= 1.0 {
		r.Resolve(GestureDispositionRejected)
	} else {
		r.OneSequenceGestureRecognizerBase.AddAllowedPointer(event)
		if r.state == forceStateReady {
			r.state = forceStatePossible
			r.lastPosition = NewOffsetPairFromEventPosition(event)
		}
	}
}

func (r *ForcePressGestureRecognizer) details(pressure float64, global, local ui.Offset) *ForcePressDetails {
	return NewForcePressDetails(ForcePressDetailsOptions{Pressure: pressure, GlobalPosition: global, LocalPosition: &local})
}

func (r *ForcePressGestureRecognizer) HandleEvent(event PointerEvent) {
	dartAssert(r.state != forceStateReady, nil)
	// A static pointer with changes in pressure creates PointerMoveEvent events.
	switch event.(type) {
	case *PointerMoveEvent, *PointerDownEvent:
		pressure := r.Interpolation(event.PressureMin(), event.PressureMax(), event.Pressure())
		dartAssert((pressure >= 0.0 && pressure <= 1.0) || math.IsNaN(pressure), nil)
		r.lastPosition = NewOffsetPairFromEventPosition(event)
		r.lastPressure = pressure

		if r.state == forceStatePossible {
			if pressure > r.StartPressure {
				r.state = forceStateStarted
				r.Resolve(GestureDispositionAccepted)
			} else if event.Delta().DistanceSquared() > ComputeHitSlop(event.Kind(), r.GestureSettings) {
				r.Resolve(GestureDispositionRejected)
			}
		}
		// In case this is the only gesture detector we still don't want to start
		// the gesture until the pressure is greater than the startPressure.
		if pressure > r.StartPressure && r.state == forceStateAccepted {
			r.state = forceStateStarted
			if r.OnStart != nil {
				InvokeVoidCallback(r, "onStart", func() {
					r.OnStart(r.details(pressure, r.lastPosition.Global, r.lastPosition.Local))
				})
			}
		}
		if r.OnPeak != nil && pressure > r.PeakPressure && r.state == forceStateStarted {
			r.state = forceStatePeaked
			if r.OnPeak != nil {
				InvokeVoidCallback(r, "onPeak", func() {
					r.OnPeak(r.details(pressure, event.Position(), event.LocalPosition()))
				})
			}
		}
		if r.OnUpdate != nil && !math.IsNaN(pressure) &&
			(r.state == forceStateStarted || r.state == forceStatePeaked) {
			if r.OnUpdate != nil {
				InvokeVoidCallback(r, "onUpdate", func() {
					r.OnUpdate(r.details(pressure, event.Position(), event.LocalPosition()))
				})
			}
		}
	}
	r.StopTrackingIfPointerNoLongerDown(event)
}

func (r *ForcePressGestureRecognizer) AcceptGesture(pointer int) {
	if r.state == forceStatePossible {
		r.state = forceStateAccepted
	}
	if r.OnStart != nil && r.state == forceStateStarted {
		InvokeVoidCallback(r, "onStart", func() {
			r.OnStart(r.details(r.lastPressure, r.lastPosition.Global, r.lastPosition.Local))
		})
	}
}

func (r *ForcePressGestureRecognizer) DidStopTrackingLastPointer(pointer int) {
	wasAccepted := r.state == forceStateStarted || r.state == forceStatePeaked
	if r.state == forceStatePossible {
		r.Resolve(GestureDispositionRejected)
		return
	}
	if wasAccepted && r.OnEnd != nil {
		if r.OnEnd != nil {
			InvokeVoidCallback(r, "onEnd", func() {
				r.OnEnd(r.details(0.0, r.lastPosition.Global, r.lastPosition.Local))
			})
		}
	}
	r.state = forceStateReady
}

func (r *ForcePressGestureRecognizer) RejectGesture(pointer int) {
	r.StopTrackingPointer(pointer)
	r.DidStopTrackingLastPointer(pointer)
}

// inverseLerp is ForcePressGestureRecognizer._inverseLerp.
func inverseLerp(min, max, t float64) float64 {
	dartAssert(min <= max, nil)
	value := (t - min) / (max - min)
	// If the device incorrectly reports a pressure outside of pressureMin
	// and pressureMax, we still want this recognizer to respond normally.
	if !math.IsNaN(value) {
		value = clampDouble(value, 0.0, 1.0)
	}
	return value
}

// clampDouble is foundation's clampDouble.
func clampDouble(x, min, max float64) float64 {
	dartAssert(min <= max && !math.IsNaN(max) && !math.IsNaN(min), nil)
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	if math.IsNaN(x) {
		return max
	}
	return x
}

func (r *ForcePressGestureRecognizer) DebugDescription() string { return "force press" }
