package flutter_test

// Port of test_pointer.dart's TestPointer (Flutter 3.50.1): a virtual
// pointer that generates the PointerEvents of a touch, mouse, stylus or
// trackpad. TestGesture (which dispatches through the binding's hit
// tester) is not ported: the Tester's gestures build on TestPointer.
//
// Dart's `{FlutterView? view}` parameters are `View *ui.FlutterView`
// options (nil: the implicit view); `timeStamp` defaults to zero.

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// TestPointer is a virtual pointer that generates PointerEvents.
type TestPointer struct {
	// Pointer is the pointer identifier.
	Pointer int
	// Kind is the kind of input device.
	Kind ui.PointerDeviceKind

	device          int
	buttons         int
	isDown          bool
	isPanZoomActive bool
	location        *ui.Offset
	pan             *ui.Offset
}

// TestPointerOptions are TestPointer's optional positional parameters
// (Dart: [pointer = 1, kind = touch, device, buttons = kPrimaryButton]).
type TestPointerOptions struct {
	Pointer *int
	Kind    ui.PointerDeviceKind
	Device  *int
	Buttons *int
}

// NewTestPointer is TestPointer([pointer, kind, device, buttons]).
func NewTestPointer(opts ...TestPointerOptions) *TestPointer {
	var o TestPointerOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &TestPointer{Pointer: 1, Kind: o.Kind, buttons: gestures.KPrimaryButton}
	if o.Pointer != nil {
		p.Pointer = *o.Pointer
	}
	if o.Buttons != nil {
		p.buttons = *o.Buttons
	}
	switch p.Kind {
	case ui.PointerDeviceKindMouse:
		p.device = 1
	default:
		p.device = 0
	}
	if o.Device != nil {
		p.device = *o.Device
	}
	return p
}

// Device is the device identifier.
func (p *TestPointer) Device() int { return p.device }

// Buttons is the buttons currently pressed.
func (p *TestPointer) Buttons() int { return p.buttons }

// IsDown reports whether the pointer is down.
func (p *TestPointer) IsDown() bool { return p.isDown }

// IsPanZoomActive reports whether a pan/zoom is active.
func (p *TestPointer) IsPanZoomActive() bool { return p.isPanZoomActive }

// Location is the position of the last event, or nil.
func (p *TestPointer) Location() *ui.Offset { return p.location }

// Pan is the pan offset of the last pan/zoom event, or nil.
func (p *TestPointer) Pan() *ui.Offset { return p.pan }

// SetDownInfo sets the location (and buttons, if not nil) for the next
// events and updates IsDown from event's type; it returns IsDown.
func (p *TestPointer) SetDownInfo(event gestures.PointerEvent, newLocation ui.Offset, buttons *int) bool {
	p.location = &newLocation
	if buttons != nil {
		p.buttons = *buttons
	}
	// Dart switches on the exact runtimeType: transformed events match
	// nothing.
	if event.Original() == nil || event.Original() == event {
		switch event.(type) {
		case *gestures.PointerDownEvent:
			assert(!p.isDown, nil)
			p.isDown = true
		case *gestures.PointerUpEvent, *gestures.PointerCancelEvent:
			assert(p.isDown, nil)
			p.isDown = false
		}
	}
	return p.isDown
}

func assert(condition bool, message any) {
	if !condition {
		panic(dart.NewAssertionError(message))
	}
}

// TestPointerEventOptions are the named parameters shared by the event
// methods.
type TestPointerEventOptions struct {
	TimeStamp time.Duration
	// Buttons, if not nil, changes the pointer's buttons (down, move).
	Buttons *int
	// View is the view the event is for (nil: the implicit view).
	View *ui.FlutterView
}

func eventOptions(opts []TestPointerEventOptions) TestPointerEventOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return TestPointerEventOptions{}
}

func viewID(view *ui.FlutterView) int {
	if view != nil {
		return view.ViewID()
	}
	return ui.PlatformDispatcherInstance().ImplicitView().ViewID()
}

// Down creates a PointerDownEvent at newLocation.
func (p *TestPointer) Down(newLocation ui.Offset, opts ...TestPointerEventOptions) *gestures.PointerDownEvent {
	o := eventOptions(opts)
	assert(!p.isDown, nil)
	assert(!p.isPanZoomActive, nil)
	p.isDown = true
	p.location = &newLocation
	if o.Buttons != nil {
		p.buttons = *o.Buttons
	}
	buttons := p.buttons
	return gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{
		ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Kind: p.Kind, Device: p.device,
		Pointer: p.Pointer, Position: *p.location, Buttons: &buttons,
	})
}

// Move creates a PointerMoveEvent to newLocation.
func (p *TestPointer) Move(newLocation ui.Offset, opts ...TestPointerEventOptions) *gestures.PointerMoveEvent {
	o := eventOptions(opts)
	assert(p.isDown, "Move events can only be generated when the pointer is down. To "+
		"create a movement event simulating a pointer move when the pointer is "+
		"up, use hover() instead.")
	assert(!p.isPanZoomActive, nil)
	delta := newLocation.Sub(*p.location)
	p.location = &newLocation
	if o.Buttons != nil {
		p.buttons = *o.Buttons
	}
	buttons := p.buttons
	return gestures.NewPointerMoveEvent(gestures.PointerMoveEventOptions{
		TimeStamp: o.TimeStamp, ViewID: viewID(o.View), Kind: p.Kind, Device: p.device,
		Pointer: p.Pointer, Position: newLocation, Delta: delta, Buttons: &buttons,
	})
}

// Up creates a PointerUpEvent.
func (p *TestPointer) Up(opts ...TestPointerEventOptions) *gestures.PointerUpEvent {
	o := eventOptions(opts)
	assert(!p.isPanZoomActive, nil)
	assert(p.isDown, nil)
	p.isDown = false
	return gestures.NewPointerUpEvent(gestures.PointerUpEventOptions{
		ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Kind: p.Kind, Device: p.device,
		Pointer: p.Pointer, Position: *p.location,
	})
}

// Cancel creates a PointerCancelEvent.
func (p *TestPointer) Cancel(opts ...TestPointerEventOptions) *gestures.PointerCancelEvent {
	o := eventOptions(opts)
	assert(p.isDown, nil)
	p.isDown = false
	return gestures.NewPointerCancelEvent(gestures.PointerCancelEventOptions{
		TimeStamp: o.TimeStamp, ViewID: viewID(o.View), Kind: p.Kind, Device: p.device,
		Pointer: p.Pointer, Position: *p.location,
	})
}

// TestPointerLocationOptions are addPointer/removePointer's named
// parameters.
type TestPointerLocationOptions struct {
	TimeStamp time.Duration
	Location  *ui.Offset
	View      *ui.FlutterView
}

func (p *TestPointer) locationOrZero() ui.Offset {
	if p.location != nil {
		return *p.location
	}
	return ui.Offset{}
}

// AddPointer creates a PointerAddedEvent.
func (p *TestPointer) AddPointer(opts ...TestPointerLocationOptions) *gestures.PointerAddedEvent {
	var o TestPointerLocationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Location != nil {
		p.location = o.Location
	}
	return gestures.NewPointerAddedEvent(gestures.PointerAddedEventOptions{
		ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Kind: p.Kind, Device: p.device,
		Position: p.locationOrZero(),
	})
}

// RemovePointer creates a PointerRemovedEvent.
func (p *TestPointer) RemovePointer(opts ...TestPointerLocationOptions) *gestures.PointerRemovedEvent {
	var o TestPointerLocationOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Location != nil {
		p.location = o.Location
	}
	return gestures.NewPointerRemovedEvent(gestures.PointerRemovedEventOptions{
		ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Kind: p.Kind, Device: p.device,
		Pointer: p.Pointer, Position: p.locationOrZero(),
	})
}

// Hover creates a PointerHoverEvent to newLocation.
func (p *TestPointer) Hover(newLocation ui.Offset, opts ...TestPointerEventOptions) *gestures.PointerHoverEvent {
	o := eventOptions(opts)
	assert(!p.isDown, "Hover events can only be generated when the pointer is up. To "+
		"simulate movement when the pointer is down, use move() instead.")
	delta := ui.Offset{}
	if p.location != nil {
		delta = newLocation.Sub(*p.location)
	}
	p.location = &newLocation
	return gestures.NewPointerHoverEvent(gestures.PointerHoverEventOptions{
		ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Kind: p.Kind, Device: p.device,
		Pointer: p.Pointer, Position: newLocation, Delta: delta,
	})
}

// TestPointerScrollOptions are scroll's named parameters.
type TestPointerScrollOptions struct {
	TimeStamp time.Duration
	OnRespond gestures.RespondPointerEventCallback
	View      *ui.FlutterView
}

func (p *TestPointer) signalOptions(timeStamp time.Duration, view *ui.FlutterView) gestures.PointerSignalEventOptions {
	assert(p.Kind != ui.PointerDeviceKindTouch, "Touch pointers can't generate pointer signal events")
	assert(p.location != nil, nil)
	kind := p.Kind
	return gestures.PointerSignalEventOptions{ViewID: viewID(view), TimeStamp: timeStamp, Kind: &kind,
		Device: p.device, Position: *p.location}
}

// Scroll creates a PointerScrollEvent.
func (p *TestPointer) Scroll(scrollDelta ui.Offset, opts ...TestPointerScrollOptions) *gestures.PointerScrollEvent {
	var o TestPointerScrollOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return gestures.NewPointerScrollEvent(gestures.PointerScrollEventOptions{
		PointerSignalEventOptions: p.signalOptions(o.TimeStamp, o.View),
		ScrollDelta:               scrollDelta, OnRespond: o.OnRespond,
	})
}

// ScrollInertiaCancel creates a PointerScrollInertiaCancelEvent.
func (p *TestPointer) ScrollInertiaCancel(opts ...TestPointerEventOptions) *gestures.PointerScrollInertiaCancelEvent {
	o := eventOptions(opts)
	return gestures.NewPointerScrollInertiaCancelEvent(p.signalOptions(o.TimeStamp, o.View))
}

// Scale creates a PointerScaleEvent.
func (p *TestPointer) Scale(scale float64, opts ...TestPointerEventOptions) *gestures.PointerScaleEvent {
	o := eventOptions(opts)
	return gestures.NewPointerScaleEvent(gestures.PointerScaleEventOptions{
		PointerSignalEventOptions: p.signalOptions(o.TimeStamp, o.View), Scale: &scale,
	})
}

// PanZoomStart creates a PointerPanZoomStartEvent at location.
func (p *TestPointer) PanZoomStart(location ui.Offset, opts ...TestPointerEventOptions) *gestures.PointerPanZoomStartEvent {
	o := eventOptions(opts)
	assert(!p.isPanZoomActive, nil)
	assert(p.Kind == ui.PointerDeviceKindTrackpad, nil)
	p.location = &location
	zero := ui.Offset{}
	p.pan = &zero
	p.isPanZoomActive = true
	return gestures.NewPointerPanZoomStartEvent(gestures.PointerPanZoomEventOptions{
		ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Device: p.device, Pointer: p.Pointer, Position: location,
	})
}

// TestPointerPanZoomUpdateOptions are panZoomUpdate's named parameters
// (Scale nil: 1).
type TestPointerPanZoomUpdateOptions struct {
	Pan       ui.Offset
	Scale     *float64
	Rotation  float64
	TimeStamp time.Duration
	View      *ui.FlutterView
}

// PanZoomUpdate creates a PointerPanZoomUpdateEvent.
func (p *TestPointer) PanZoomUpdate(location ui.Offset, opts ...TestPointerPanZoomUpdateOptions) *gestures.PointerPanZoomUpdateEvent {
	var o TestPointerPanZoomUpdateOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	assert(p.isPanZoomActive, nil)
	assert(p.Kind == ui.PointerDeviceKindTrackpad, nil)
	p.location = &location
	panDelta := o.Pan.Sub(*p.pan)
	pan := o.Pan
	p.pan = &pan
	scale := 1.0
	if o.Scale != nil {
		scale = *o.Scale
	}
	return gestures.NewPointerPanZoomUpdateEvent(gestures.PointerPanZoomUpdateEventOptions{
		PointerPanZoomEventOptions: gestures.PointerPanZoomEventOptions{
			ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Device: p.device, Pointer: p.Pointer, Position: location,
		},
		Pan: o.Pan, PanDelta: panDelta, Scale: &scale, Rotation: o.Rotation,
	})
}

// PanZoomEnd creates a PointerPanZoomEndEvent.
func (p *TestPointer) PanZoomEnd(opts ...TestPointerEventOptions) *gestures.PointerPanZoomEndEvent {
	o := eventOptions(opts)
	assert(p.isPanZoomActive, nil)
	assert(p.Kind == ui.PointerDeviceKindTrackpad, nil)
	p.isPanZoomActive = false
	p.pan = nil
	return gestures.NewPointerPanZoomEndEvent(gestures.PointerPanZoomEventOptions{
		ViewID: viewID(o.View), TimeStamp: o.TimeStamp, Device: p.device, Pointer: p.Pointer, Position: *p.location,
	})
}
