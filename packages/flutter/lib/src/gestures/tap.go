package gestures

// Port of tap.dart (Flutter 3.50.1).
//
// The details classes are Diagnosticable objects used by pointer
// (positions are methods, as PositionedGestureDetails); a nullable kind is
// a *ui.PointerDeviceKind. BaseTapGestureRecognizer's required named
// parameters (handleTapDown({required down}) ...) are positional.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// TapDownDetailsOptions are TapDownDetails' parameters (LocalPosition nil:
// GlobalPosition; Kind nil: null).
type TapDownDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
	Kind           *ui.PointerDeviceKind
}

// TapDownDetails are the details for GestureTapDownCallback.
type TapDownDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Kind is the kind of the device that initiated the event, or nil.
	Kind *ui.PointerDeviceKind
}

// NewTapDownDetails creates details for a GestureTapDownCallback.
func NewTapDownDetails(opts ...TapDownDetailsOptions) *TapDownDetails {
	var o TapDownDetailsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	d := &TapDownDetails{positionedDetails: newPositionedDetails(o.GlobalPosition, o.LocalPosition), Kind: o.Kind}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions and the kind.
func (d *TapDownDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewEnumProperty("kind", d.Kind))
}

// GestureTapDownCallback is the signature for when a pointer that might
// cause a tap has contacted the screen.
type GestureTapDownCallback func(details *TapDownDetails)

// TapUpDetailsOptions are TapUpDetails' parameters (Kind is required).
type TapUpDetailsOptions struct {
	GlobalPosition ui.Offset
	LocalPosition  *ui.Offset
	Kind           ui.PointerDeviceKind
}

// TapUpDetails are the details for GestureTapUpCallback.
type TapUpDetails struct {
	foundation.DiagnosticableMixin
	positionedDetails
	// Kind is the kind of the device that initiated the event.
	Kind ui.PointerDeviceKind
}

// NewTapUpDetails creates details for a GestureTapUpCallback.
func NewTapUpDetails(opts TapUpDetailsOptions) *TapUpDetails {
	d := &TapUpDetails{positionedDetails: newPositionedDetails(opts.GlobalPosition, opts.LocalPosition), Kind: opts.Kind}
	d.SetSelf(d)
	return d
}

// DebugFillProperties adds the positions and the kind.
func (d *TapUpDetails) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	d.DiagnosticableMixin.DebugFillProperties(properties)
	d.fillPositions(properties)
	properties.Add(foundation.NewEnumProperty("kind", d.Kind))
}

// TapMoveDetailsOptions are TapMoveDetails' parameters (Kind is required).
type TapMoveDetailsOptions struct {
	Kind           ui.PointerDeviceKind
	GlobalPosition ui.Offset
	Delta          ui.Offset
	LocalPosition  *ui.Offset
}

// TapMoveDetails are the details for GestureTapMoveCallback.
type TapMoveDetails struct {
	positionedDetails
	// Kind is the kind of the device that initiated the event.
	Kind ui.PointerDeviceKind
	// Delta is the amount the pointer moved since the previous update.
	Delta ui.Offset
}

// NewTapMoveDetails creates details for a GestureTapMoveCallback.
func NewTapMoveDetails(opts TapMoveDetailsOptions) *TapMoveDetails {
	return &TapMoveDetails{positionedDetails: newPositionedDetails(opts.GlobalPosition, opts.LocalPosition),
		Kind: opts.Kind, Delta: opts.Delta}
}

// GestureTapUpCallback is the signature for when a pointer that will
// trigger a tap has stopped contacting the screen.
type GestureTapUpCallback func(details *TapUpDetails)

// GestureTapCallback is the signature for when a tap has occurred.
type GestureTapCallback func()

// GestureTapMoveCallback is the signature for when a pointer that might
// cause a tap has moved.
type GestureTapMoveCallback func(details *TapMoveDetails)

// GestureTapCancelCallback is the signature for when the pointer that
// previously triggered a GestureTapDownCallback will not end up causing a
// tap.
type GestureTapCancelCallback func()

// BaseTapGestureRecognizerOptions are BaseTapGestureRecognizer's
// constructor parameters (see PrimaryPointerGestureRecognizerOptions for
// the slop tolerances).
type BaseTapGestureRecognizerOptions struct {
	GestureRecognizerOptions
	PreAcceptSlopTolerance    *float64
	NoPreAcceptSlopTolerance  bool
	PostAcceptSlopTolerance   *float64
	NoPostAcceptSlopTolerance bool
}

// BaseTapGestureRecognizer is a base class for gesture recognizers that
// recognize taps (an abstract class).
type BaseTapGestureRecognizer interface {
	PrimaryPointerGestureRecognizer
	// HandleTapDown is called when a tap is possible (protected, abstract).
	HandleTapDown(down *PointerDownEvent)
	// HandleTapUp is called when a tap is recognized (protected,
	// abstract).
	HandleTapUp(down *PointerDownEvent, up *PointerUpEvent)
	// HandleTapMove is called when the pointer moves (protected).
	HandleTapMove(move *PointerMoveEvent)
	// HandleTapCancel is called when a tap is canceled after
	// HandleTapDown (protected, abstract); cancel is nil unless a
	// PointerCancelEvent caused it.
	HandleTapCancel(down *PointerDownEvent, cancel *PointerCancelEvent, reason string)
}

// BaseTapGestureRecognizerBase is BaseTapGestureRecognizer's state and
// default methods, to embed.
type BaseTapGestureRecognizerBase struct {
	PrimaryPointerGestureRecognizerBase

	self BaseTapGestureRecognizer

	sentTapDown               bool
	wonArenaForPrimaryPointer bool
	down                      *PointerDownEvent
	up                        *PointerUpEvent
}

// Construct is BaseTapGestureRecognizer's constructor (deadline:
// kPressTimeout).
func (r *BaseTapGestureRecognizerBase) Construct(self BaseTapGestureRecognizer, opts BaseTapGestureRecognizerOptions) {
	deadline := KPressTimeout
	r.PrimaryPointerGestureRecognizerBase.Construct(self, PrimaryPointerGestureRecognizerOptions{
		GestureRecognizerOptions:  opts.GestureRecognizerOptions,
		Deadline:                  &deadline,
		PreAcceptSlopTolerance:    opts.PreAcceptSlopTolerance,
		NoPreAcceptSlopTolerance:  opts.NoPreAcceptSlopTolerance,
		PostAcceptSlopTolerance:   opts.PostAcceptSlopTolerance,
		NoPostAcceptSlopTolerance: opts.NoPostAcceptSlopTolerance,
	})
	r.self = self
}

func (r *BaseTapGestureRecognizerBase) HandleTapMove(move *PointerMoveEvent) {}

func (r *BaseTapGestureRecognizerBase) AddAllowedPointer(event *PointerDownEvent) {
	if r.self.State() == GestureRecognizerStateReady {
		// If there is no result in the previous gesture arena,
		// we ignore them and prepare to accept a new pointer.
		if r.down != nil && r.up != nil {
			dartAssert(r.down.Pointer() == r.up.Pointer(), nil)
			r.reset()
		}
		dartAssert(r.down == nil && r.up == nil, nil)
		r.down = event
	}
	if r.down != nil {
		r.PrimaryPointerGestureRecognizerBase.AddAllowedPointer(event)
	}
}

func (r *BaseTapGestureRecognizerBase) StartTrackingPointer(pointer int, transform ...*vm.Matrix4) {
	// The recognizer should never track any pointers when `_down` is null,
	// because calling `_checkDown` in this state will throw exception.
	dartAssert(r.down != nil, nil)
	r.PrimaryPointerGestureRecognizerBase.StartTrackingPointer(pointer, transform...)
}

func (r *BaseTapGestureRecognizerBase) HandlePrimaryPointer(event PointerEvent) {
	switch e := event.(type) {
	case *PointerUpEvent:
		r.up = e
		r.checkUp()
		return
	case *PointerCancelEvent:
		r.self.Resolve(GestureDispositionRejected)
		if r.sentTapDown {
			r.checkCancel(e, "")
		}
		r.reset()
		return
	}
	if event.Buttons() != r.down.Buttons() {
		r.self.Resolve(GestureDispositionRejected)
		r.self.StopTrackingPointer(*r.self.PrimaryPointer())
	} else if move, ok := event.(*PointerMoveEvent); ok {
		r.checkMove(move)
	}
}

func (r *BaseTapGestureRecognizerBase) Resolve(disposition GestureDisposition) {
	if r.wonArenaForPrimaryPointer && disposition == GestureDispositionRejected {
		// This can happen if the gesture has been canceled. For example, when
		// the pointer has exceeded the touch slop, the buttons have been changed,
		// or if the recognizer is disposed.
		dartAssert(r.sentTapDown, nil)
		r.checkCancel(nil, "spontaneous")
		r.reset()
	}
	r.PrimaryPointerGestureRecognizerBase.Resolve(disposition)
}

func (r *BaseTapGestureRecognizerBase) DidExceedDeadline() { r.checkDown() }

func (r *BaseTapGestureRecognizerBase) AcceptGesture(pointer int) {
	r.PrimaryPointerGestureRecognizerBase.AcceptGesture(pointer)
	if r.isPrimary(pointer) {
		r.checkDown()
		r.wonArenaForPrimaryPointer = true
		r.checkUp()
	}
}

func (r *BaseTapGestureRecognizerBase) RejectGesture(pointer int) {
	r.PrimaryPointerGestureRecognizerBase.RejectGesture(pointer)
	if r.isPrimary(pointer) {
		// Another gesture won the arena.
		dartAssert(r.self.State() != GestureRecognizerStatePossible, nil)
		if r.sentTapDown {
			r.checkCancel(nil, "forced")
		}
		r.reset()
	}
}

func (r *BaseTapGestureRecognizerBase) checkDown() {
	if r.sentTapDown {
		return
	}
	r.self.HandleTapDown(r.down)
	r.sentTapDown = true
}

func (r *BaseTapGestureRecognizerBase) checkUp() {
	if !r.wonArenaForPrimaryPointer || r.up == nil {
		return
	}
	dartAssert(r.up.Pointer() == r.down.Pointer(), nil)
	r.self.HandleTapUp(r.down, r.up)
	r.reset()
}

func (r *BaseTapGestureRecognizerBase) checkCancel(event *PointerCancelEvent, note string) {
	r.self.HandleTapCancel(r.down, event, note)
}

func (r *BaseTapGestureRecognizerBase) checkMove(event *PointerMoveEvent) {
	dartAssert(event.Pointer() == r.down.Pointer(), nil)
	r.self.HandleTapMove(event)
}

func (r *BaseTapGestureRecognizerBase) reset() {
	r.sentTapDown = false
	r.wonArenaForPrimaryPointer = false
	r.up = nil
	r.down = nil
}

func (r *BaseTapGestureRecognizerBase) DebugDescription() string { return "base tap" }

func (r *BaseTapGestureRecognizerBase) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	r.PrimaryPointerGestureRecognizerBase.DebugFillProperties(properties)
	wonArena := "won arena"
	properties.Add(foundation.NewFlagProperty("wonArenaForPrimaryPointer", r.wonArenaForPrimaryPointer,
		foundation.FlagPropertyOptions{IfTrue: &wonArena}))
	if r.up != nil {
		p, l := r.up.Position(), r.up.LocalPosition()
		properties.Add(foundation.NewDiagnosticsProperty("finalPosition", p,
			foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
		properties.Add(foundation.NewDiagnosticsProperty("finalLocalPosition", l,
			foundation.DiagnosticsPropertyOptions{DefaultValue: p}))
	} else {
		properties.Add(foundation.NewDiagnosticsProperty[*ui.Offset]("finalPosition", nil,
			foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
		properties.Add(foundation.NewDiagnosticsProperty[*ui.Offset]("finalLocalPosition", nil,
			foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}))
	}
	var button *int
	if r.down != nil {
		b := r.down.Buttons()
		button = &b
	}
	properties.Add(foundation.NewIntProperty("button", button, foundation.IntPropertyOptions{DefaultValue: foundation.Null}))
	sentTapDown := "sent tap down"
	properties.Add(foundation.NewFlagProperty("sentTapDown", r.sentTapDown,
		foundation.FlagPropertyOptions{IfTrue: &sentTapDown}))
}

// TapGestureRecognizerOptions are TapGestureRecognizer's constructor
// parameters.
type TapGestureRecognizerOptions = BaseTapGestureRecognizerOptions

// TapGestureRecognizer recognizes taps (primary, secondary and tertiary
// buttons).
type TapGestureRecognizer struct {
	BaseTapGestureRecognizerBase

	OnTapDown            GestureTapDownCallback
	OnTapUp              GestureTapUpCallback
	OnTap                GestureTapCallback
	OnTapMove            GestureTapMoveCallback
	OnTapCancel          GestureTapCancelCallback
	OnSecondaryTap       GestureTapCallback
	OnSecondaryTapDown   GestureTapDownCallback
	OnSecondaryTapUp     GestureTapUpCallback
	OnSecondaryTapCancel GestureTapCancelCallback
	OnTertiaryTapDown    GestureTapDownCallback
	OnTertiaryTapUp      GestureTapUpCallback
	OnTertiaryTapCancel  GestureTapCancelCallback
}

// NewTapGestureRecognizer creates a tap gesture recognizer.
func NewTapGestureRecognizer(opts ...TapGestureRecognizerOptions) *TapGestureRecognizer {
	var o TapGestureRecognizerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r := &TapGestureRecognizer{}
	r.Construct(r, o)
	return r
}

func (r *TapGestureRecognizer) IsPointerAllowed(event *PointerDownEvent) bool {
	switch event.Buttons() {
	case KPrimaryButton:
		if r.OnTapDown == nil && r.OnTap == nil && r.OnTapUp == nil && r.OnTapCancel == nil && r.OnTapMove == nil {
			return false
		}
	case KSecondaryButton:
		if r.OnSecondaryTap == nil && r.OnSecondaryTapDown == nil && r.OnSecondaryTapUp == nil &&
			r.OnSecondaryTapCancel == nil {
			return false
		}
	case KTertiaryButton:
		if r.OnTertiaryTapDown == nil && r.OnTertiaryTapUp == nil && r.OnTertiaryTapCancel == nil {
			return false
		}
	default:
		return false
	}
	return r.BaseTapGestureRecognizerBase.IsPointerAllowed(event)
}

func (r *TapGestureRecognizer) HandleTapDown(down *PointerDownEvent) {
	kind := r.GetKindForPointer(down.Pointer())
	local := down.LocalPosition()
	details := NewTapDownDetails(TapDownDetailsOptions{GlobalPosition: down.Position(), LocalPosition: &local, Kind: &kind})
	switch down.Buttons() {
	case KPrimaryButton:
		if r.OnTapDown != nil {
			InvokeVoidCallback(r, "onTapDown", func() { r.OnTapDown(details) })
		}
	case KSecondaryButton:
		if r.OnSecondaryTapDown != nil {
			InvokeVoidCallback(r, "onSecondaryTapDown", func() { r.OnSecondaryTapDown(details) })
		}
	case KTertiaryButton:
		if r.OnTertiaryTapDown != nil {
			InvokeVoidCallback(r, "onTertiaryTapDown", func() { r.OnTertiaryTapDown(details) })
		}
	}
}

func (r *TapGestureRecognizer) HandleTapUp(down *PointerDownEvent, up *PointerUpEvent) {
	local := up.LocalPosition()
	details := NewTapUpDetails(TapUpDetailsOptions{Kind: up.Kind(), GlobalPosition: up.Position(), LocalPosition: &local})
	switch down.Buttons() {
	case KPrimaryButton:
		if r.OnTapUp != nil {
			InvokeVoidCallback(r, "onTapUp", func() { r.OnTapUp(details) })
		}
		if r.OnTap != nil {
			InvokeVoidCallback(r, "onTap", r.OnTap)
		}
	case KSecondaryButton:
		if r.OnSecondaryTapUp != nil {
			InvokeVoidCallback(r, "onSecondaryTapUp", func() { r.OnSecondaryTapUp(details) })
		}
		if r.OnSecondaryTap != nil {
			InvokeVoidCallback(r, "onSecondaryTap", func() { r.OnSecondaryTap() })
		}
	case KTertiaryButton:
		if r.OnTertiaryTapUp != nil {
			InvokeVoidCallback(r, "onTertiaryTapUp", func() { r.OnTertiaryTapUp(details) })
		}
	}
}

func (r *TapGestureRecognizer) HandleTapMove(move *PointerMoveEvent) {
	if r.OnTapMove != nil && move.Buttons() == KPrimaryButton {
		local := move.LocalPosition()
		details := NewTapMoveDetails(TapMoveDetailsOptions{GlobalPosition: move.Position(), LocalPosition: &local,
			Kind: r.GetKindForPointer(move.Pointer()), Delta: move.Delta()})
		InvokeVoidCallback(r, "onTapMove", func() { r.OnTapMove(details) })
	}
}

func (r *TapGestureRecognizer) HandleTapCancel(down *PointerDownEvent, cancel *PointerCancelEvent, reason string) {
	note := reason
	if reason != "" {
		note = reason + " "
	}
	switch down.Buttons() {
	case KPrimaryButton:
		if r.OnTapCancel != nil {
			InvokeVoidCallback(r, note+"onTapCancel", r.OnTapCancel)
		}
	case KSecondaryButton:
		if r.OnSecondaryTapCancel != nil {
			InvokeVoidCallback(r, note+"onSecondaryTapCancel", r.OnSecondaryTapCancel)
		}
	case KTertiaryButton:
		if r.OnTertiaryTapCancel != nil {
			InvokeVoidCallback(r, note+"onTertiaryTapCancel", r.OnTertiaryTapCancel)
		}
	}
}

func (r *TapGestureRecognizer) DebugDescription() string { return "tap" }
