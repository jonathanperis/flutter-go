package gestures

// Port of long_press.dart (Flutter 3.50.1).

import (
	"strconv"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// GestureLongPressDownCallback is the signature for when a pointer that
// might cause a long-press has contacted the screen.
type GestureLongPressDownCallback func(details *LongPressDownDetails)

// GestureLongPressCancelCallback is the signature for when the pointer
// that previously triggered a GestureLongPressDownCallback will not end up
// causing a long-press.
type GestureLongPressCancelCallback func()

// GestureLongPressCallback is the signature for when a long press has been
// recognized.
type GestureLongPressCallback func()

// GestureLongPressUpCallback is the signature for when a pointer stops
// contacting the screen after a long press.
type GestureLongPressUpCallback func()

// GestureLongPressStartCallback is the signature for when a long press
// has been recognized, with details.
type GestureLongPressStartCallback func(details *LongPressStartDetails)

// GestureLongPressMoveUpdateCallback is the signature for when a pointer
// is moving after being held in contact at the same location for a long
// period of time.
type GestureLongPressMoveUpdateCallback func(details *LongPressMoveUpdateDetails)

// GestureLongPressEndCallback is the signature for when a pointer stops
// contacting the screen after a long press, with details.
type GestureLongPressEndCallback func(details *LongPressEndDetails)

// LongPressDownDetailsOptions are LongPressDownDetails' parameters.
type LongPressDownDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
	Kind           *ui.PointerDeviceKind
}

// LongPressDownDetails are the details for GestureLongPressDownCallback.
type LongPressDownDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Kind is the kind of the device that initiated the event, or nil.
	Kind *ui.PointerDeviceKind
}

// NewLongPressDownDetails creates details for
// GestureLongPressDownCallback.
func NewLongPressDownDetails(opts ...LongPressDownDetailsOptions) *LongPressDownDetails {
	var o LongPressDownDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &LongPressDownDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition), Kind: o.Kind}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions and the kind.
func (d *LongPressDownDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewEnumProperty("kind", d.Kind))
}

// LongPressStartDetailsOptions are LongPressStartDetails' parameters.
type LongPressStartDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
}

// LongPressStartDetails are the details for GestureLongPressStartCallback.
type LongPressStartDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
}

// NewLongPressStartDetails creates details for
// GestureLongPressStartCallback.
func NewLongPressStartDetails(opts ...LongPressStartDetailsOptions) *LongPressStartDetails {
	var o LongPressStartDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &LongPressStartDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition)}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions.
func (d *LongPressStartDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
}

// LongPressMoveUpdateDetailsOptions are LongPressMoveUpdateDetails'
// parameters (LocalOffsetFromOrigin nil: OffsetFromOrigin).
type LongPressMoveUpdateDetailsOptions struct {
	GlobalPosition        ui.Offset
	LocalPosition         *ui.Offset
	OffsetFromOrigin      ui.Offset
	LocalOffsetFromOrigin *ui.Offset
}

// LongPressMoveUpdateDetails are the details for
// GestureLongPressMoveUpdateCallback.
type LongPressMoveUpdateDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// OffsetFromOrigin is the displacement from the long press origin.
	OffsetFromOrigin ui.Offset
	// LocalOffsetFromOrigin is OffsetFromOrigin in local coordinates.
	LocalOffsetFromOrigin ui.Offset
}

// NewLongPressMoveUpdateDetails creates details for
// GestureLongPressMoveUpdateCallback.
func NewLongPressMoveUpdateDetails(opts ...LongPressMoveUpdateDetailsOptions) *LongPressMoveUpdateDetails {
	var o LongPressMoveUpdateDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &LongPressMoveUpdateDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition),
		OffsetFromOrigin: o.OffsetFromOrigin, LocalOffsetFromOrigin: o.OffsetFromOrigin}
	if o.LocalOffsetFromOrigin != nil {
		d.LocalOffsetFromOrigin = *o.LocalOffsetFromOrigin
	}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the details' fields.
func (d *LongPressMoveUpdateDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewDiagnosticsProperty("offsetFromOrigin", d.OffsetFromOrigin))
	properties.Add(foundation.NewDiagnosticsProperty("localOffsetFromOrigin", d.LocalOffsetFromOrigin))
}

// LongPressEndDetailsOptions are LongPressEndDetails' parameters.
type LongPressEndDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
	Velocity       Velocity
}

// LongPressEndDetails are the details for GestureLongPressEndCallback.
type LongPressEndDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Velocity is the pointer's velocity when it stopped contacting the
	// screen.
	Velocity Velocity
}

// NewLongPressEndDetails creates details for GestureLongPressEndCallback.
func NewLongPressEndDetails(opts ...LongPressEndDetailsOptions) *LongPressEndDetails {
	var o LongPressEndDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &LongPressEndDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition), Velocity: o.Velocity}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the details' fields.
func (d *LongPressEndDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewDiagnosticsProperty("velocity", d.Velocity))
}

// LongPressGestureRecognizerOptions are LongPressGestureRecognizer's
// constructor parameters. Duration nil: kLongPressTimeout;
// PostAcceptSlopTolerance nil: null (no tolerance, Dart's default here);
// AllowedButtonsFilter nil: a single primary, secondary or tertiary button.
type LongPressGestureRecognizerOptions struct {
	Duration                *time.Duration
	PostAcceptSlopTolerance *float64
	SupportedDevices        map[ui.PointerDeviceKind]struct{}
	DebugOwner              any
	AllowedButtonsFilter    AllowedButtonsFilter
}

// LongPressGestureRecognizer recognizes long presses.
type LongPressGestureRecognizer struct {
	PrimaryPointerGestureRecognizerBase

	OnLongPressDown       GestureLongPressDownCallback
	OnLongPressCancel     GestureLongPressCancelCallback
	OnLongPress           GestureLongPressCallback
	OnLongPressStart      GestureLongPressStartCallback
	OnLongPressMoveUpdate GestureLongPressMoveUpdateCallback
	OnLongPressUp         GestureLongPressUpCallback
	OnLongPressEnd        GestureLongPressEndCallback

	OnSecondaryLongPressDown       GestureLongPressDownCallback
	OnSecondaryLongPressCancel     GestureLongPressCancelCallback
	OnSecondaryLongPress           GestureLongPressCallback
	OnSecondaryLongPressStart      GestureLongPressStartCallback
	OnSecondaryLongPressMoveUpdate GestureLongPressMoveUpdateCallback
	OnSecondaryLongPressUp         GestureLongPressUpCallback
	OnSecondaryLongPressEnd        GestureLongPressEndCallback

	OnTertiaryLongPressDown       GestureLongPressDownCallback
	OnTertiaryLongPressCancel     GestureLongPressCancelCallback
	OnTertiaryLongPress           GestureLongPressCallback
	OnTertiaryLongPressStart      GestureLongPressStartCallback
	OnTertiaryLongPressMoveUpdate GestureLongPressMoveUpdateCallback
	OnTertiaryLongPressUp         GestureLongPressUpCallback
	OnTertiaryLongPressEnd        GestureLongPressEndCallback

	longPressAccepted bool
	longPressOrigin   *OffsetPair
	initialButtons    *int
	velocityTracker   *VelocityTracker
}

func longPressDefaultButtonAcceptBehavior(buttons int) bool {
	return buttons == KPrimaryButton || buttons == KSecondaryButton || buttons == KTertiaryButton
}

// NewLongPressGestureRecognizer creates a long-press recognizer.
func NewLongPressGestureRecognizer(opts ...LongPressGestureRecognizerOptions) *LongPressGestureRecognizer {
	var o LongPressGestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	deadline := KLongPressTimeout
	if o.Duration != nil {
		deadline = *o.Duration
	}
	filter := o.AllowedButtonsFilter
	if filter == nil {
		filter = longPressDefaultButtonAcceptBehavior
	}
	r := &LongPressGestureRecognizer{}
	r.Construct(r, PrimaryPointerGestureRecognizerOptions{
		GestureRecognizerOptions: GestureRecognizerOptions{DebugOwner: o.DebugOwner,
			SupportedDevices: o.SupportedDevices, AllowedButtonsFilter: filter},
		Deadline:                  &deadline,
		PostAcceptSlopTolerance:   o.PostAcceptSlopTolerance,
		NoPostAcceptSlopTolerance: o.PostAcceptSlopTolerance == nil,
	})
	return r
}

func (r *LongPressGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	switch event.Buttons() {
	case KPrimaryButton:
		if r.OnLongPressDown == nil && r.OnLongPressCancel == nil && r.OnLongPressStart == nil &&
			r.OnLongPress == nil && r.OnLongPressMoveUpdate == nil && r.OnLongPressEnd == nil &&
			r.OnLongPressUp == nil {
			return false
		}
	case KSecondaryButton:
		if r.OnSecondaryLongPressDown == nil && r.OnSecondaryLongPressCancel == nil &&
			r.OnSecondaryLongPressStart == nil && r.OnSecondaryLongPress == nil &&
			r.OnSecondaryLongPressMoveUpdate == nil && r.OnSecondaryLongPressEnd == nil &&
			r.OnSecondaryLongPressUp == nil {
			return false
		}
	case KTertiaryButton:
		if r.OnTertiaryLongPressDown == nil && r.OnTertiaryLongPressCancel == nil &&
			r.OnTertiaryLongPressStart == nil && r.OnTertiaryLongPress == nil &&
			r.OnTertiaryLongPressMoveUpdate == nil && r.OnTertiaryLongPressEnd == nil &&
			r.OnTertiaryLongPressUp == nil {
			return false
		}
	default:
		return false
	}
	return r.PrimaryPointerGestureRecognizerBase.IsPointerAllowed(event)
}

func (r *LongPressGestureRecognizer) DidExceedDeadline() {
	// Exceeding the deadline puts the gesture in the accepted state.
	r.Resolve(GestureDispositionAccepted)
	r.longPressAccepted = true
	r.PrimaryPointerGestureRecognizerBase.AcceptGesture(*r.PrimaryPointer())
	r.checkLongPressStart()
}

func (r *LongPressGestureRecognizer) HandlePrimaryPointer(event PointerEvent) {
	if !event.Synthesized() {
		switch event.(type) {
		case *PointerDownEvent:
			r.velocityTracker = NewVelocityTrackerWithKind(event.Kind())
			r.velocityTracker.AddPosition(event.TimeStamp(), event.LocalPosition())
		case *PointerMoveEvent:
			dartAssert(r.velocityTracker != nil, nil)
			r.velocityTracker.AddPosition(event.TimeStamp(), event.LocalPosition())
		}
	}
	switch e := event.(type) {
	case *PointerUpEvent:
		if r.longPressAccepted {
			r.checkLongPressEnd(event)
		} else {
			// Pointer is lifted before timeout.
			r.Resolve(GestureDispositionRejected)
		}
		r.reset()
	case *PointerCancelEvent:
		r.checkLongPressCancel()
		r.reset()
	case *PointerDownEvent:
		// The first touch.
		origin := NewOffsetPairFromEventPosition(event)
		r.longPressOrigin = &origin
		buttons := event.Buttons()
		r.initialButtons = &buttons
		r.checkLongPressDown(e)
	case *PointerMoveEvent:
		if !r.sameInitialButtons(event.Buttons()) && !r.longPressAccepted {
			r.Resolve(GestureDispositionRejected)
			r.StopTrackingPointer(*r.PrimaryPointer())
		} else if r.longPressAccepted {
			r.checkLongPressMoveUpdate(event)
		}
	}
}

func (r *LongPressGestureRecognizer) sameInitialButtons(buttons int) bool {
	return r.initialButtons != nil && *r.initialButtons == buttons
}

func (r *LongPressGestureRecognizer) buttonsCase() int {
	if r.initialButtons == nil {
		return -1
	}
	return *r.initialButtons
}

func (r *LongPressGestureRecognizer) unhandledButton() {
	s := "null"
	if r.initialButtons != nil {
		s = strconv.Itoa(*r.initialButtons)
	}
	dartAssert(false, "Unhandled button "+s)
}

func (r *LongPressGestureRecognizer) checkLongPressDown(event *PointerDownEvent) {
	dartAssert(r.longPressOrigin != nil, nil)
	kind := r.GetKindForPointer(event.Pointer())
	local := r.longPressOrigin.Local
	details := NewLongPressDownDetails(LongPressDownDetailsOptions{GlobalPosition: r.longPressOrigin.Global,
		LocalPosition: &local, Kind: &kind})
	switch r.buttonsCase() {
	case KPrimaryButton:
		if r.OnLongPressDown != nil {
			InvokeVoidCallback(r, "onLongPressDown", func() { r.OnLongPressDown(details) })
		}
	case KSecondaryButton:
		if r.OnSecondaryLongPressDown != nil {
			InvokeVoidCallback(r, "onSecondaryLongPressDown", func() { r.OnSecondaryLongPressDown(details) })
		}
	case KTertiaryButton:
		if r.OnTertiaryLongPressDown != nil {
			InvokeVoidCallback(r, "onTertiaryLongPressDown", func() { r.OnTertiaryLongPressDown(details) })
		}
	default:
		r.unhandledButton()
	}
}

func (r *LongPressGestureRecognizer) checkLongPressCancel() {
	if r.State() == GestureRecognizerStatePossible {
		switch r.buttonsCase() {
		case KPrimaryButton:
			if r.OnLongPressCancel != nil {
				InvokeVoidCallback(r, "onLongPressCancel", r.OnLongPressCancel)
			}
		case KSecondaryButton:
			if r.OnSecondaryLongPressCancel != nil {
				InvokeVoidCallback(r, "onSecondaryLongPressCancel", r.OnSecondaryLongPressCancel)
			}
		case KTertiaryButton:
			if r.OnTertiaryLongPressCancel != nil {
				InvokeVoidCallback(r, "onTertiaryLongPressCancel", r.OnTertiaryLongPressCancel)
			}
		default:
			r.unhandledButton()
		}
	}
}

func (r *LongPressGestureRecognizer) startDetails() *LongPressStartDetails {
	local := r.longPressOrigin.Local
	return NewLongPressStartDetails(LongPressStartDetailsOptions{GlobalPosition: r.longPressOrigin.Global, LocalPosition: &local})
}

func (r *LongPressGestureRecognizer) checkLongPressStart() {
	switch r.buttonsCase() {
	case KPrimaryButton:
		if r.OnLongPressStart != nil {
			details := r.startDetails()
			InvokeVoidCallback(r, "onLongPressStart", func() { r.OnLongPressStart(details) })
		}
		if r.OnLongPress != nil {
			InvokeVoidCallback(r, "onLongPress", r.OnLongPress)
		}
	case KSecondaryButton:
		if r.OnSecondaryLongPressStart != nil {
			details := r.startDetails()
			InvokeVoidCallback(r, "onSecondaryLongPressStart", func() { r.OnSecondaryLongPressStart(details) })
		}
		if r.OnSecondaryLongPress != nil {
			InvokeVoidCallback(r, "onSecondaryLongPress", r.OnSecondaryLongPress)
		}
	case KTertiaryButton:
		if r.OnTertiaryLongPressStart != nil {
			details := r.startDetails()
			InvokeVoidCallback(r, "onTertiaryLongPressStart", func() { r.OnTertiaryLongPressStart(details) })
		}
		if r.OnTertiaryLongPress != nil {
			InvokeVoidCallback(r, "onTertiaryLongPress", r.OnTertiaryLongPress)
		}
	default:
		r.unhandledButton()
	}
}

func (r *LongPressGestureRecognizer) checkLongPressMoveUpdate(event PointerEvent) {
	local := event.LocalPosition()
	localOffset := event.LocalPosition().Sub(r.longPressOrigin.Local)
	details := NewLongPressMoveUpdateDetails(LongPressMoveUpdateDetailsOptions{
		GlobalPosition: event.Position(), LocalPosition: &local,
		OffsetFromOrigin: event.Position().Sub(r.longPressOrigin.Global), LocalOffsetFromOrigin: &localOffset,
	})
	switch r.buttonsCase() {
	case KPrimaryButton:
		if r.OnLongPressMoveUpdate != nil {
			InvokeVoidCallback(r, "onLongPressMoveUpdate", func() { r.OnLongPressMoveUpdate(details) })
		}
	case KSecondaryButton:
		if r.OnSecondaryLongPressMoveUpdate != nil {
			InvokeVoidCallback(r, "onSecondaryLongPressMoveUpdate", func() { r.OnSecondaryLongPressMoveUpdate(details) })
		}
	case KTertiaryButton:
		if r.OnTertiaryLongPressMoveUpdate != nil {
			InvokeVoidCallback(r, "onTertiaryLongPressMoveUpdate", func() { r.OnTertiaryLongPressMoveUpdate(details) })
		}
	default:
		r.unhandledButton()
	}
}

func (r *LongPressGestureRecognizer) checkLongPressEnd(event PointerEvent) {
	estimate := r.velocityTracker.GetVelocityEstimate()
	velocity := VelocityZero
	if estimate != nil {
		velocity = Velocity{PixelsPerSecond: estimate.PixelsPerSecond}
	}
	local := event.LocalPosition()
	details := NewLongPressEndDetails(LongPressEndDetailsOptions{GlobalPosition: event.Position(),
		LocalPosition: &local, Velocity: velocity})
	r.velocityTracker = nil
	switch r.buttonsCase() {
	case KPrimaryButton:
		if r.OnLongPressEnd != nil {
			InvokeVoidCallback(r, "onLongPressEnd", func() { r.OnLongPressEnd(details) })
		}
		if r.OnLongPressUp != nil {
			InvokeVoidCallback(r, "onLongPressUp", r.OnLongPressUp)
		}
	case KSecondaryButton:
		if r.OnSecondaryLongPressEnd != nil {
			InvokeVoidCallback(r, "onSecondaryLongPressEnd", func() { r.OnSecondaryLongPressEnd(details) })
		}
		if r.OnSecondaryLongPressUp != nil {
			InvokeVoidCallback(r, "onSecondaryLongPressUp", r.OnSecondaryLongPressUp)
		}
	case KTertiaryButton:
		if r.OnTertiaryLongPressEnd != nil {
			InvokeVoidCallback(r, "onTertiaryLongPressEnd", func() { r.OnTertiaryLongPressEnd(details) })
		}
		if r.OnTertiaryLongPressUp != nil {
			InvokeVoidCallback(r, "onTertiaryLongPressUp", r.OnTertiaryLongPressUp)
		}
	default:
		r.unhandledButton()
	}
}

func (r *LongPressGestureRecognizer) reset() {
	r.longPressAccepted = false
	r.longPressOrigin = nil
	r.initialButtons = nil
	r.velocityTracker = nil
}

func (r *LongPressGestureRecognizer) Resolve(disposition GestureDisposition) {
	if disposition == GestureDispositionRejected {
		if r.longPressAccepted {
			// This can happen if the gesture has been canceled. For example when
			// the buttons have changed.
			r.reset()
		} else {
			r.checkLongPressCancel()
		}
	}
	r.PrimaryPointerGestureRecognizerBase.Resolve(disposition)
}

// AcceptGesture does nothing: winning the arena isn't important here since
// it may happen from a sweep; exceeding the deadline puts the gesture in
// the accepted state.
func (r *LongPressGestureRecognizer) AcceptGesture(pointer int) {}

func (r *LongPressGestureRecognizer) DebugDescription() string { return "long press" }
