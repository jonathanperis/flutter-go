// Port of packages/flutter/test/gestures/resampler_test.dart (Flutter
// 3.50.1): all 10 tests, translated mechanically from the Dart file
// (expect(x, y) is expectEq; `is PointerXEvent` is expectType). No skips.
package gestures_test

import (
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
)

func us(n int) time.Duration { return time.Duration(n) * time.Microsecond }

func createAdded(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerAddedEvent(PointerAddedEventOptions{TimeStamp: us(timeStampUs), Position: ui.NewOffset(x, y)})
}

func createRemoved(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerRemovedEvent(PointerRemovedEventOptions{TimeStamp: us(timeStampUs), Position: ui.NewOffset(x, y)})
}

func createDown(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerDownEvent(PointerDownEventOptions{TimeStamp: us(timeStampUs), Position: ui.NewOffset(x, y)})
}

func createMove(timeStampUs int, x, y, deltaX, deltaY float64) PointerEvent {
	return NewPointerMoveEvent(PointerMoveEventOptions{TimeStamp: us(timeStampUs), Position: ui.NewOffset(x, y),
		Delta: ui.NewOffset(deltaX, deltaY)})
}

func createHover(timeStampUs int, x, y, deltaX, deltaY float64) PointerEvent {
	return NewPointerHoverEvent(PointerHoverEventOptions{TimeStamp: us(timeStampUs), Position: ui.NewOffset(x, y),
		Delta: ui.NewOffset(deltaX, deltaY)})
}

func createUp(timeStampUs int, x, y float64) PointerEvent {
	return NewPointerUpEvent(PointerUpEventOptions{TimeStamp: us(timeStampUs), Position: ui.NewOffset(x, y)})
}

func expectEq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func expectType[E PointerEvent](t *testing.T, e PointerEvent) {
	t.Helper()
	if _, ok := e.(E); !ok {
		t.Fatalf("%T is not the expected type", e)
	}
}

func TestResamplerBasic(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 50.0)
	event1 := createHover(2000, 10.0, 40.0, 10.0, -10.0)
	event2 := createDown(2000, 10.0, 40.0)
	event3 := createMove(3000, 20.0, 30.0, 10.0, -10.0)
	event4 := createMove(4000, 30.0, 20.0, 10.0, -10.0)
	event5 := createUp(4000, 30.0, 20.0)
	event6 := createHover(5000, 40.0, 10.0, 10.0, -10.0)
	event7 := createHover(6000, 50.0, 0.0, 10.0, -10.0)
	event8 := createRemoved(6000, 50.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)
	resampler.AddEvent(event6)
	resampler.AddEvent(event7)
	resampler.AddEvent(event8)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(500), 0, add)

	// No pointer event should have been returned yet.
	expectEq(t, len(result), 0)

	resampler.Sample(us(1500), 0, add)

	// Add pointer event should have been returned.
	expectEq(t, len(result), 1)
	expectEq(t, result[0].TimeStamp(), us(1500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 5.0)
	expectEq(t, result[0].Position().Dy, 45.0)

	resampler.Sample(us(2500), 0, add)

	// Hover and down pointer events should have been returned.
	expectEq(t, len(result), 3)
	expectEq(t, result[1].TimeStamp(), us(2500))
	expectType[*PointerHoverEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 15.0)
	expectEq(t, result[1].Position().Dy, 35.0)
	expectEq(t, result[1].Delta().Dx, 10.0)
	expectEq(t, result[1].Delta().Dy, -10.0)
	expectEq(t, result[2].TimeStamp(), us(2500))
	expectType[*PointerDownEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 15.0)
	expectEq(t, result[2].Position().Dy, 35.0)

	resampler.Sample(us(3500), 0, add)

	// Move pointer event should have been returned.
	expectEq(t, len(result), 4)
	expectEq(t, result[3].TimeStamp(), us(3500))
	expectType[*PointerMoveEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 25.0)
	expectEq(t, result[3].Position().Dy, 25.0)
	expectEq(t, result[3].Delta().Dx, 10.0)
	expectEq(t, result[3].Delta().Dy, -10.0)

	resampler.Sample(us(4500), 0, add)

	// Move and up pointer events should have been returned.
	expectEq(t, len(result), 6)
	expectEq(t, result[4].TimeStamp(), us(4500))
	expectType[*PointerMoveEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 35.0)
	expectEq(t, result[4].Position().Dy, 15.0)
	expectEq(t, result[4].Delta().Dx, 10.0)
	expectEq(t, result[4].Delta().Dy, -10.0)
	// buttons field needs to be a valid value
	expectEq(t, result[4].Buttons(), KPrimaryButton)
	expectEq(t, result[5].TimeStamp(), us(4500))
	expectType[*PointerUpEvent](t, result[5])
	expectEq(t, result[5].Position().Dx, 35.0)
	expectEq(t, result[5].Position().Dy, 15.0)

	resampler.Sample(us(5500), 0, add)

	// Hover pointer event should have been returned.
	expectEq(t, len(result), 7)
	expectEq(t, result[6].TimeStamp(), us(5500))
	expectType[*PointerHoverEvent](t, result[6])
	expectEq(t, result[6].Position().Dx, 45.0)
	expectEq(t, result[6].Position().Dy, 5.0)
	expectEq(t, result[6].Delta().Dx, 10.0)
	expectEq(t, result[6].Delta().Dy, -10.0)

	resampler.Sample(us(6500), 0, add)

	// Hover and removed pointer events should have been returned.
	expectEq(t, len(result), 9)
	expectEq(t, result[7].TimeStamp(), us(6500))
	expectType[*PointerHoverEvent](t, result[7])
	expectEq(t, result[7].Position().Dx, 50.0)
	expectEq(t, result[7].Position().Dy, 0.0)
	expectEq(t, result[7].Delta().Dx, 5.0)
	expectEq(t, result[7].Delta().Dy, -5.0)
	expectEq(t, result[8].TimeStamp(), us(6500))
	expectType[*PointerRemovedEvent](t, result[8])
	expectEq(t, result[8].Position().Dx, 50.0)
	expectEq(t, result[8].Position().Dy, 0.0)

	resampler.Sample(us(7500), 0, add)

	// No pointer event should have been returned.
	expectEq(t, len(result), 9)
}

func TestResamplerStream(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 50.0)
	event1 := createHover(2000, 10.0, 40.0, 10.0, -10.0)
	event2 := createDown(2000, 10.0, 40.0)
	event3 := createMove(3000, 20.0, 30.0, 10.0, -10.0)
	event4 := createMove(4000, 30.0, 20.0, 10.0, -10.0)
	event5 := createUp(4000, 30.0, 20.0)
	event6 := createHover(5000, 40.0, 10.0, 10.0, -10.0)
	event7 := createHover(6000, 50.0, 0.0, 10.0, -10.0)
	event8 := createRemoved(6000, 50.0, 0.0)

	resampler.AddEvent(event0)

	//
	// Initial sample time a 0.5 ms.
	//

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(500), 0, add)

	// No pointer event should have been returned yet.
	expectEq(t, len(result), 0)

	resampler.AddEvent(event1)
	resampler.AddEvent(event2)

	resampler.Sample(us(500), 0, add)

	// No pointer event should have been returned yet.
	expectEq(t, len(result), 0)

	//
	// Advance sample time to 1.5 ms.
	//

	resampler.Sample(us(1500), 0, add)

	// Added pointer event should have been returned.
	expectEq(t, len(result), 1)
	expectEq(t, result[0].TimeStamp(), us(1500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 5.0)
	expectEq(t, result[0].Position().Dy, 45.0)

	resampler.AddEvent(event3)

	resampler.Sample(us(1500), 0, add)

	// No more pointer events should have been returned.
	expectEq(t, len(result), 1)

	//
	// Advance sample time to 2.5 ms.
	//

	resampler.Sample(us(2500), 0, add)

	// Hover and down pointer events should have been returned.
	expectEq(t, len(result), 3)
	expectEq(t, result[1].TimeStamp(), us(2500))
	expectType[*PointerHoverEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 15.0)
	expectEq(t, result[1].Position().Dy, 35.0)
	expectEq(t, result[1].Delta().Dx, 10.0)
	expectEq(t, result[1].Delta().Dy, -10.0)
	expectEq(t, result[2].TimeStamp(), us(2500))
	expectType[*PointerDownEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 15.0)
	expectEq(t, result[2].Position().Dy, 35.0)

	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	resampler.Sample(us(2500), 0, add)

	// No more pointer events should have been returned.
	expectEq(t, len(result), 3)

	//
	// Advance sample time to 3.5 ms.
	//

	resampler.Sample(us(3500), 0, add)

	// Move pointer event should have been returned.
	expectEq(t, len(result), 4)
	expectEq(t, result[3].TimeStamp(), us(3500))
	expectType[*PointerMoveEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 25.0)
	expectEq(t, result[3].Position().Dy, 25.0)
	expectEq(t, result[3].Delta().Dx, 10.0)
	expectEq(t, result[3].Delta().Dy, -10.0)

	resampler.AddEvent(event6)

	resampler.Sample(us(3500), 0, add)

	// No more pointer events should have been returned.
	expectEq(t, len(result), 4)

	//
	// Advance sample time to 4.5 ms.
	//

	resampler.Sample(us(4500), 0, add)

	// Move and up pointer events should have been returned.
	expectEq(t, len(result), 6)
	expectEq(t, result[4].TimeStamp(), us(4500))
	expectType[*PointerMoveEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 35.0)
	expectEq(t, result[4].Position().Dy, 15.0)
	expectEq(t, result[4].Delta().Dx, 10.0)
	expectEq(t, result[4].Delta().Dy, -10.0)
	expectEq(t, result[5].TimeStamp(), us(4500))
	expectType[*PointerUpEvent](t, result[5])
	expectEq(t, result[5].Position().Dx, 35.0)
	expectEq(t, result[5].Position().Dy, 15.0)

	resampler.AddEvent(event7)
	resampler.AddEvent(event8)

	resampler.Sample(us(4500), 0, add)

	// No more pointer events should have been returned.
	expectEq(t, len(result), 6)

	//
	// Advance sample time to 5.5 ms.
	//

	resampler.Sample(us(5500), 0, add)

	// Hover pointer event should have been returned.
	expectEq(t, len(result), 7)
	expectEq(t, result[6].TimeStamp(), us(5500))
	expectType[*PointerHoverEvent](t, result[6])
	expectEq(t, result[6].Position().Dx, 45.0)
	expectEq(t, result[6].Position().Dy, 5.0)
	expectEq(t, result[6].Delta().Dx, 10.0)
	expectEq(t, result[6].Delta().Dy, -10.0)

	//
	// Advance sample time to 6.5 ms.
	//

	resampler.Sample(us(6500), 0, add)

	// Hover and removed pointer event should have been returned.
	expectEq(t, len(result), 9)
	expectEq(t, result[7].TimeStamp(), us(6500))
	expectType[*PointerHoverEvent](t, result[7])
	expectEq(t, result[7].Position().Dx, 50.0)
	expectEq(t, result[7].Position().Dy, 0.0)
	expectEq(t, result[7].Delta().Dx, 5.0)
	expectEq(t, result[7].Delta().Dy, -5.0)
	expectEq(t, result[8].TimeStamp(), us(6500))
	expectType[*PointerRemovedEvent](t, result[8])
	expectEq(t, result[8].Position().Dx, 50.0)
	expectEq(t, result[8].Position().Dy, 0.0)

	//
	// Advance sample time to 7.5 ms.
	//

	resampler.Sample(us(7500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 9)
}

func TestResamplerQuickTap(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(1000, 0.0, 0.0)
	event2 := createUp(1000, 0.0, 0.0)
	event3 := createRemoved(1000, 0.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(1500), 0, add)

	// All pointer events should have been returned.
	expectEq(t, len(result), 4)
	expectEq(t, result[0].TimeStamp(), us(1500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 0.0)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(1500))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 0.0)
	expectEq(t, result[1].Position().Dy, 0.0)
	expectEq(t, result[2].TimeStamp(), us(1500))
	expectType[*PointerUpEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 0.0)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[3].TimeStamp(), us(1500))
	expectType[*PointerRemovedEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 0.0)
	expectEq(t, result[3].Position().Dy, 0.0)
}

func TestResamplerAdvanceSlowly(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(1000, 0.0, 0.0)
	event2 := createMove(2000, 10.0, 0.0, 10.0, 0.0)
	event3 := createMove(3000, 20.0, 0.0, 10.0, 0.0)
	event4 := createUp(3000, 20.0, 0.0)
	event5 := createRemoved(3000, 20.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(1500), 0, add)

	// Added and down pointer events should have been returned.
	expectEq(t, len(result), 2)
	expectEq(t, result[0].TimeStamp(), us(1500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 5.0)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(1500))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 5.0)
	expectEq(t, result[1].Position().Dy, 0.0)

	resampler.Sample(us(1500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 2)

	resampler.Sample(us(1750), 0, add)

	// Move pointer event should have been returned.
	expectEq(t, len(result), 3)
	expectEq(t, result[2].TimeStamp(), us(1750))
	expectType[*PointerMoveEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 7.5)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[2].Delta().Dx, 2.5)
	expectEq(t, result[2].Delta().Dy, 0.0)

	resampler.Sample(us(2000), 0, add)

	// Another move pointer event should have been returned.
	expectEq(t, len(result), 4)
	expectEq(t, result[3].TimeStamp(), us(2000))
	expectType[*PointerMoveEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 10.0)
	expectEq(t, result[3].Position().Dy, 0.0)
	expectEq(t, result[3].Delta().Dx, 2.5)
	expectEq(t, result[3].Delta().Dy, 0.0)

	resampler.Sample(us(3000), 0, add)

	// Move, up and removed pointer events should have been returned.
	expectEq(t, len(result), 7)
	expectEq(t, result[4].TimeStamp(), us(3000))
	expectType[*PointerMoveEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 20.0)
	expectEq(t, result[4].Position().Dy, 0.0)
	expectEq(t, result[4].Delta().Dx, 10.0)
	expectEq(t, result[4].Delta().Dy, 0.0)
	expectEq(t, result[5].TimeStamp(), us(3000))
	expectType[*PointerUpEvent](t, result[5])
	expectEq(t, result[5].Position().Dx, 20.0)
	expectEq(t, result[5].Position().Dy, 0.0)
	expectEq(t, result[6].TimeStamp(), us(3000))
	expectType[*PointerRemovedEvent](t, result[6])
	expectEq(t, result[6].Position().Dx, 20.0)
	expectEq(t, result[6].Position().Dy, 0.0)
}

func TestResamplerAdvanceFast(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(1000, 0.0, 0.0)
	event2 := createMove(2000, 5.0, 0.0, 5.0, 0.0)
	event3 := createMove(3000, 20.0, 0.0, 15.0, 0.0)
	event4 := createUp(4000, 30.0, 0.0)
	event5 := createRemoved(4000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(2500), 0, add)

	// Added and down pointer events should have been returned.
	expectEq(t, len(result), 2)
	expectEq(t, result[0].TimeStamp(), us(2500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 12.5)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(2500))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 12.5)
	expectEq(t, result[1].Position().Dy, 0.0)

	resampler.Sample(us(5500), 0, add)

	// Move, up and removed pointer events should have been returned.
	expectEq(t, len(result), 5)
	expectEq(t, result[2].TimeStamp(), us(5500))
	expectType[*PointerMoveEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 30.0)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[2].Delta().Dx, 17.5)
	expectEq(t, result[2].Delta().Dy, 0.0)
	expectEq(t, result[3].TimeStamp(), us(5500))
	expectType[*PointerUpEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 30.0)
	expectEq(t, result[3].Position().Dy, 0.0)
	expectEq(t, result[4].TimeStamp(), us(5500))
	expectType[*PointerRemovedEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 30.0)
	expectEq(t, result[4].Position().Dy, 0.0)

	resampler.Sample(us(6500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 5)
}

func TestResamplerSkip(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(1000, 0.0, 0.0)
	event2 := createMove(2000, 10.0, 0.0, 10.0, 0.0)
	event3 := createUp(3000, 10.0, 0.0)
	event4 := createHover(4000, 20.0, 0.0, 10.0, 0.0)
	event5 := createDown(4000, 20.0, 0.0)
	event6 := createMove(5000, 30.0, 0.0, 10.0, 0.0)
	event7 := createUp(5000, 30.0, 0.0)
	event8 := createRemoved(5000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)
	resampler.AddEvent(event6)
	resampler.AddEvent(event7)
	resampler.AddEvent(event8)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(1500), 0, add)

	// Added and down pointer events should have been returned.
	expectEq(t, len(result), 2)
	expectEq(t, result[0].TimeStamp(), us(1500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 5.0)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(1500))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 5.0)
	expectEq(t, result[1].Position().Dy, 0.0)

	resampler.Sample(us(5500), 0, add)

	// All remaining pointer events should have been returned.
	expectEq(t, len(result), 7)
	expectEq(t, result[2].TimeStamp(), us(5500))
	expectType[*PointerMoveEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 30.0)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[2].Delta().Dx, 25.0)
	expectEq(t, result[2].Delta().Dy, 0.0)
	expectEq(t, result[3].TimeStamp(), us(5500))
	expectType[*PointerUpEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 30.0)
	expectEq(t, result[3].Position().Dy, 0.0)
	expectEq(t, result[4].TimeStamp(), us(5500))
	expectType[*PointerDownEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 30.0)
	expectEq(t, result[4].Position().Dy, 0.0)
	expectEq(t, result[5].TimeStamp(), us(5500))
	expectType[*PointerUpEvent](t, result[5])
	expectEq(t, result[5].Position().Dx, 30.0)
	expectEq(t, result[5].Position().Dy, 0.0)
	expectEq(t, result[6].TimeStamp(), us(5500))
	expectType[*PointerRemovedEvent](t, result[6])
	expectEq(t, result[6].Position().Dx, 30.0)
	expectEq(t, result[6].Position().Dy, 0.0)

	resampler.Sample(us(6500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 7)
}

func TestResamplerSkipAll(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(1000, 0.0, 0.0)
	event2 := createMove(4000, 30.0, 0.0, 30.0, 0.0)
	event3 := createUp(4000, 30.0, 0.0)
	event4 := createRemoved(4000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 0)

	resampler.Sample(us(5500), 0, add)

	// All remaining pointer events should have been returned.
	expectEq(t, len(result), 4)
	expectEq(t, result[0].TimeStamp(), us(5500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 30.0)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(5500))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 30.0)
	expectEq(t, result[1].Position().Dy, 0.0)
	expectEq(t, result[2].TimeStamp(), us(5500))
	expectType[*PointerUpEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 30.0)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[3].TimeStamp(), us(5500))
	expectType[*PointerRemovedEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 30.0)
	expectEq(t, result[3].Position().Dy, 0.0)

	resampler.Sample(us(6500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 4)
}

func TestResamplerStop(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(2000, 0.0, 0.0)
	event2 := createMove(3000, 10.0, 0.0, 10.0, 0.0)
	event3 := createMove(4000, 20.0, 0.0, 10.0, 0.0)
	event4 := createUp(4000, 20.0, 0.0)
	event5 := createRemoved(5000, 20.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 0)

	resampler.Stop(add)

	// All pointer events should have been returned with original
	// time stamps and positions.
	expectEq(t, len(result), 6)
	expectEq(t, result[0].TimeStamp(), us(1000))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 0.0)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(2000))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 0.0)
	expectEq(t, result[1].Position().Dy, 0.0)
	expectEq(t, result[2].TimeStamp(), us(3000))
	expectType[*PointerMoveEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 10.0)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[2].Delta().Dx, 10.0)
	expectEq(t, result[2].Delta().Dy, 0.0)
	expectEq(t, result[3].TimeStamp(), us(4000))
	expectType[*PointerMoveEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 20.0)
	expectEq(t, result[3].Position().Dy, 0.0)
	expectEq(t, result[3].Delta().Dx, 10.0)
	expectEq(t, result[3].Delta().Dy, 0.0)
	expectEq(t, result[4].TimeStamp(), us(4000))
	expectType[*PointerUpEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 20.0)
	expectEq(t, result[4].Position().Dy, 0.0)
	expectEq(t, result[5].TimeStamp(), us(5000))
	expectType[*PointerRemovedEvent](t, result[5])
	expectEq(t, result[5].Position().Dx, 20.0)
	expectEq(t, result[5].Position().Dy, 0.0)

	resampler.Sample(us(10000), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 6)
}

func TestResamplerSyntheticMove(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(2000, 0.0, 0.0)
	event2 := createMove(3000, 10.0, 0.0, 10.0, 0.0)
	event3 := createUp(4000, 10.0, 0.0)
	event4 := createRemoved(5000, 10.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	resampler.Sample(us(500), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 0)

	resampler.Sample(us(2000), 0, add)

	// Added and down pointer events should have been returned.
	expectEq(t, len(result), 2)
	expectEq(t, result[0].TimeStamp(), us(2000))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 0.0)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(2000))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 0.0)
	expectEq(t, result[1].Position().Dy, 0.0)

	resampler.Sample(us(5000), 0, add)

	// All remaining pointer events and a synthetic move event should
	// have been returned.
	expectEq(t, len(result), 5)
	expectEq(t, result[2].TimeStamp(), us(5000))
	expectType[*PointerMoveEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 10.0)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[2].Delta().Dx, 10.0)
	expectEq(t, result[2].Delta().Dy, 0.0)
	expectEq(t, result[3].TimeStamp(), us(5000))
	expectType[*PointerUpEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 10.0)
	expectEq(t, result[3].Position().Dy, 0.0)
	expectEq(t, result[4].TimeStamp(), us(5000))
	expectType[*PointerRemovedEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 10.0)
	expectEq(t, result[4].Position().Dy, 0.0)

	resampler.Sample(us(10000), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 5)
}

func TestResamplerNextSampleTime(t *testing.T) {
	resampler := NewPointerEventResampler()
	event0 := createAdded(1000, 0.0, 0.0)
	event1 := createDown(1000, 0.0, 0.0)
	event2 := createMove(2000, 10.0, 0.0, 10.0, 0.0)
	event3 := createMove(3000, 20.0, 0.0, 10.0, 0.0)
	event4 := createUp(3000, 20.0, 0.0)
	event5 := createHover(4000, 30.0, 0.0, 10.0, 0.0)
	event6 := createRemoved(4000, 30.0, 0.0)

	resampler.AddEvent(event0)
	resampler.AddEvent(event1)
	resampler.AddEvent(event2)
	resampler.AddEvent(event3)
	resampler.AddEvent(event4)
	resampler.AddEvent(event5)
	resampler.AddEvent(event6)

	var result []PointerEvent
	add := func(e PointerEvent) { result = append(result, e) }

	sampleTime := us(500)
	nextSampleTime := us(1500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 0)

	sampleTime = nextSampleTime
	nextSampleTime = us(2500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// Added and down pointer events should have been returned.
	expectEq(t, len(result), 2)
	expectEq(t, result[0].TimeStamp(), us(1500))
	expectType[*PointerAddedEvent](t, result[0])
	expectEq(t, result[0].Position().Dx, 5.0)
	expectEq(t, result[0].Position().Dy, 0.0)
	expectEq(t, result[1].TimeStamp(), us(1500))
	expectType[*PointerDownEvent](t, result[1])
	expectEq(t, result[1].Position().Dx, 5.0)
	expectEq(t, result[1].Position().Dy, 0.0)

	sampleTime = nextSampleTime
	nextSampleTime = us(3500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// Move and up pointer events should have been returned.
	expectEq(t, len(result), 4)
	expectEq(t, result[2].TimeStamp(), us(2500))
	expectType[*PointerMoveEvent](t, result[2])
	expectEq(t, result[2].Position().Dx, 15.0)
	expectEq(t, result[2].Position().Dy, 0.0)
	expectEq(t, result[2].Delta().Dx, 10.0)
	expectEq(t, result[2].Delta().Dy, 0.0)
	expectEq(t, result[3].TimeStamp(), us(2500))
	expectType[*PointerUpEvent](t, result[3])
	expectEq(t, result[3].Position().Dx, 15.0)
	expectEq(t, result[3].Position().Dy, 0.0)

	sampleTime = nextSampleTime
	nextSampleTime = us(4500)
	resampler.Sample(sampleTime, nextSampleTime, add)

	// All remaining pointer events should have been returned.
	expectEq(t, len(result), 6)
	expectEq(t, result[4].TimeStamp(), us(3500))
	expectType[*PointerHoverEvent](t, result[4])
	expectEq(t, result[4].Position().Dx, 25.0)
	expectEq(t, result[4].Position().Dy, 0.0)
	expectEq(t, result[4].Delta().Dx, 10.0)
	expectEq(t, result[4].Delta().Dy, 0.0)
	expectEq(t, result[5].TimeStamp(), us(3500))
	expectType[*PointerRemovedEvent](t, result[5])
	expectEq(t, result[5].Position().Dx, 25.0)
	expectEq(t, result[5].Position().Dy, 0.0)

	resampler.Sample(us(10000), 0, add)

	// No pointer events should have been returned.
	expectEq(t, len(result), 6)
}
