package gestures

// Port of resampler.dart (Flutter 3.50.1).

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// HandleEventCallback is the signature for a callback that receives
// resampled events.
type HandleEventCallback func(event PointerEvent)

// PointerEventResampler resamples a single pointer's events at given
// sample times (linear interpolation between the events around each
// sample time), producing move/hover events at the sample times.
type PointerEventResampler struct {
	// Events queued for processing.
	queuedEvents []PointerEvent

	// Pointer state required for resampling.
	last              PointerEvent
	next              PointerEvent
	position          ui.Offset
	isTracked         bool
	isDown            bool
	pointerIdentifier int
	hasButtons        int
}

// NewPointerEventResampler creates a resampler.
func NewPointerEventResampler() *PointerEventResampler { return &PointerEventResampler{} }

// toHoverEvent is _toHoverEvent (as in Dart, the event's own buttons are
// used, not the buttons argument).
func (r *PointerEventResampler) toHoverEvent(event PointerEvent, position, delta ui.Offset, timeStamp time.Duration,
	buttons int) PointerEvent {
	return NewPointerHoverEvent(PointerHoverEventOptions{
		ViewID: event.ViewID(), TimeStamp: timeStamp, Kind: event.Kind(), Device: event.Device(),
		Position: position, Delta: delta, Buttons: event.Buttons(), Obscured: event.Obscured(),
		PressureMin: ptr(event.PressureMin()), PressureMax: ptr(event.PressureMax()),
		Distance: event.Distance(), DistanceMax: event.DistanceMax(), Size: event.Size(),
		RadiusMajor: event.RadiusMajor(), RadiusMinor: event.RadiusMinor(), RadiusMin: event.RadiusMin(),
		RadiusMax: event.RadiusMax(), Orientation: event.Orientation(), Tilt: event.Tilt(),
		Synthesized: event.Synthesized(), EmbedderID: event.EmbedderID(),
	})
}

func (r *PointerEventResampler) toMoveEvent(event PointerEvent, position, delta ui.Offset, pointerIdentifier int,
	timeStamp time.Duration, buttons int) PointerEvent {
	return NewPointerMoveEvent(PointerMoveEventOptions{
		ViewID: event.ViewID(), TimeStamp: timeStamp, Pointer: pointerIdentifier, Kind: event.Kind(),
		Device: event.Device(), Position: position, Delta: delta, Buttons: ptr(buttons),
		Obscured: event.Obscured(), Pressure: ptr(event.Pressure()), PressureMin: ptr(event.PressureMin()),
		PressureMax: ptr(event.PressureMax()), DistanceMax: event.DistanceMax(), Size: event.Size(),
		RadiusMajor: event.RadiusMajor(), RadiusMinor: event.RadiusMinor(), RadiusMin: event.RadiusMin(),
		RadiusMax: event.RadiusMax(), Orientation: event.Orientation(), Tilt: event.Tilt(),
		PlatformData: event.PlatformData(), Synthesized: event.Synthesized(), EmbedderID: event.EmbedderID(),
	})
}

func (r *PointerEventResampler) toMoveOrHoverEvent(event PointerEvent, position, delta ui.Offset, pointerIdentifier int,
	timeStamp time.Duration, isDown bool, buttons int) PointerEvent {
	if isDown {
		return r.toMoveEvent(event, position, delta, pointerIdentifier, timeStamp, buttons)
	}
	return r.toHoverEvent(event, position, delta, timeStamp, buttons)
}

func (r *PointerEventResampler) positionAt(sampleTime time.Duration) ui.Offset {
	// Use `next` position by default.
	x, y := 0.0, 0.0
	var nextTimeStamp, lastTimeStamp time.Duration
	if r.next != nil {
		x, y = r.next.Position().Dx, r.next.Position().Dy
		nextTimeStamp = r.next.TimeStamp()
	}
	if r.last != nil {
		lastTimeStamp = r.last.TimeStamp()
	}

	// Resample if `next` time stamp is past `sampleTime`.
	if nextTimeStamp > sampleTime && nextTimeStamp > lastTimeStamp {
		interval := float64((nextTimeStamp - lastTimeStamp) / time.Microsecond)
		scalar := float64((sampleTime-lastTimeStamp)/time.Microsecond) / interval
		lastX, lastY := 0.0, 0.0
		if r.last != nil {
			lastX, lastY = r.last.Position().Dx, r.last.Position().Dy
		}
		x = lastX + float64((x-lastX)*scalar)
		y = lastY + float64((y-lastY)*scalar)
	}

	return ui.NewOffset(x, y)
}

func (r *PointerEventResampler) processPointerEvents(sampleTime time.Duration) {
	for _, event := range r.queuedEvents {
		// Update both `last` and `next` pointer event if time stamp is older
		// or equal to `sampleTime`.
		if event.TimeStamp() <= sampleTime || r.last == nil {
			r.last = event
			r.next = event
			continue
		}

		// Update only `next` pointer event if time stamp is more recent than
		// `sampleTime` and next event is not already more recent.
		var nextTimeStamp time.Duration
		if r.next != nil {
			nextTimeStamp = r.next.TimeStamp()
		}
		if nextTimeStamp < sampleTime {
			r.next = event
			break
		}
	}
}

func isMoveOrHover(event PointerEvent) bool {
	switch event.(type) {
	case *PointerMoveEvent, *PointerHoverEvent:
		return true
	}
	return false
}

func (r *PointerEventResampler) dequeueAndSampleNonHoverOrMovePointerEventsUntil(sampleTime, nextSampleTime time.Duration,
	callback HandleEventCallback) {
	endTime := sampleTime
	// Scan queued events to determine end time.
	for _, event := range r.queuedEvents {
		// Potentially stop dispatching events if more recent than
		// `sampleTime`.
		if event.TimeStamp() > sampleTime {
			// Definitely stop if more recent than `nextSampleTime`.
			if event.TimeStamp() >= nextSampleTime {
				break
			}

			// Update `endTime` to allow early processing of up and removed
			// events as this improves resampling of these events, which is
			// important for fling animations.
			switch event.(type) {
			case *PointerUpEvent, *PointerRemovedEvent:
				endTime = event.TimeStamp()
				continue
			}

			// Stop if event is not move or hover.
			if !isMoveOrHover(event) {
				break
			}
		}
	}

	for len(r.queuedEvents) > 0 {
		event := r.queuedEvents[0]

		// Stop dispatching events if more recent than `endTime`.
		if event.TimeStamp() > endTime {
			break
		}

		wasTracked := r.isTracked
		wasDown := r.isDown
		hadButtons := r.hasButtons

		// Update pointer state.
		_, removed := event.(*PointerRemovedEvent)
		r.isTracked = !removed
		r.isDown = event.Down()
		r.hasButtons = event.Buttons()

		// Position at `sampleTime`.
		position := r.positionAt(sampleTime)

		// Initialize position if we are starting to track this pointer.
		if r.isTracked && !wasTracked {
			r.position = position
		}

		// Current pointer identifier.
		pointerIdentifier := event.Pointer()

		// Initialize pointer identifier for `move` events. Identifier is
		// expected to be the same while `down`.
		dartAssert(!wasDown || r.pointerIdentifier == pointerIdentifier, nil)
		r.pointerIdentifier = pointerIdentifier

		// Skip `move` and `hover` events as they are automatically generated
		// when the position has changed.
		if !isMoveOrHover(event) {
			// Add synthetics `move` or `hover` event if position has changed.
			//
			// Devices without `hover` events are expected to always have `add`
			// and `down` events with the same position and this logic will
			// therefore never produce `hover` events.
			if position != r.position {
				delta := position.Sub(r.position)
				callback(r.toMoveOrHoverEvent(event, position, delta, r.pointerIdentifier, sampleTime, wasDown, hadButtons))
				r.position = position
			}
			zero := ui.OffsetZero
			callback(event.CopyWith(PointerEventCopyWithOptions{
				Position: &position, Delta: &zero, Pointer: &pointerIdentifier, TimeStamp: &sampleTime,
			}))
		}

		r.queuedEvents = r.queuedEvents[1:]
	}
}

func (r *PointerEventResampler) samplePointerPosition(sampleTime time.Duration, callback HandleEventCallback) {
	// Position at `sampleTime`.
	position := r.positionAt(sampleTime)

	// Add `move` or `hover` events if position has changed.
	next := r.next
	if position != r.position && next != nil {
		delta := position.Sub(r.position)
		callback(r.toMoveOrHoverEvent(next, position, delta, r.pointerIdentifier, sampleTime, r.isDown, r.hasButtons))
		r.position = position
	}
}

// AddEvent enqueues an event for resampling.
func (r *PointerEventResampler) AddEvent(event PointerEvent) {
	r.queuedEvents = append(r.queuedEvents, event)
}

// Sample dispatches the resampled events at sampleTime to callback
// (nextSampleTime lets up/remove events before it be processed early).
func (r *PointerEventResampler) Sample(sampleTime, nextSampleTime time.Duration, callback HandleEventCallback) {
	r.processPointerEvents(sampleTime)

	// Dequeue and sample pointer events until `sampleTime`.
	r.dequeueAndSampleNonHoverOrMovePointerEventsUntil(sampleTime, nextSampleTime, callback)

	// Dispatch resampled pointer location event if tracked.
	if r.isTracked {
		r.samplePointerPosition(sampleTime, callback)
	}
}

// Stop dispatches the queued events to callback and resets the state.
func (r *PointerEventResampler) Stop(callback HandleEventCallback) {
	for len(r.queuedEvents) > 0 {
		event := r.queuedEvents[0]
		r.queuedEvents = r.queuedEvents[1:]
		callback(event)
	}
	r.pointerIdentifier = 0
	r.isDown = false
	r.isTracked = false
	r.position = ui.OffsetZero
	r.next = nil
	r.last = nil
}

// HasPendingEvents reports whether events are queued.
func (r *PointerEventResampler) HasPendingEvents() bool { return len(r.queuedEvents) > 0 }

// IsTracked reports whether the pointer is tracked (added, not removed).
func (r *PointerEventResampler) IsTracked() bool { return r.isTracked }

// IsDown reports whether the pointer is down.
func (r *PointerEventResampler) IsDown() bool { return r.isDown }
