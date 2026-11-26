package gestures_test

// Test helpers: a minimal stand-in for flutter_test's TestPointer (down,
// move, up, hover; view id 0, the implicit view), until flutter_test's
// gesture helpers are ported.

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
)

type testPointer struct {
	pointer  int
	kind     ui.PointerDeviceKind
	device   int
	buttons  int
	isDown   bool
	location ui.Offset
}

func newTestPointer(pointer ...int) *testPointer {
	p := 1
	if len(pointer) > 0 {
		p = pointer[0]
	}
	return &testPointer{pointer: p, buttons: KPrimaryButton}
}

func (p *testPointer) down(location ui.Offset, timeStamp ...time.Duration) *PointerDownEvent {
	p.isDown = true
	p.location = location
	return NewPointerDownEvent(PointerDownEventOptions{TimeStamp: optDuration(timeStamp), Kind: p.kind,
		Device: p.device, Pointer: p.pointer, Position: location, Buttons: &p.buttons})
}

func (p *testPointer) move(location ui.Offset, timeStamp ...time.Duration) *PointerMoveEvent {
	delta := location.Sub(p.location)
	p.location = location
	return NewPointerMoveEvent(PointerMoveEventOptions{TimeStamp: optDuration(timeStamp), Kind: p.kind,
		Device: p.device, Pointer: p.pointer, Position: location, Delta: delta, Buttons: &p.buttons})
}

func (p *testPointer) up(timeStamp ...time.Duration) *PointerUpEvent {
	p.isDown = false
	return NewPointerUpEvent(PointerUpEventOptions{TimeStamp: optDuration(timeStamp), Kind: p.kind,
		Device: p.device, Pointer: p.pointer, Position: p.location})
}

func optDuration(d []time.Duration) time.Duration {
	if len(d) > 0 {
		return d[0]
	}
	return 0
}
