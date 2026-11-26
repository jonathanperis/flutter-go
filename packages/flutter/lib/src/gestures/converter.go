package gestures

// Port of converter.dart (Flutter 3.50.1).

import (
	"iter"
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// synthesiseDownButtons is _synthesiseDownButtons: adds KPrimaryButton to
// buttons when a pointer of certain devices is down.
func synthesiseDownButtons(buttons int, kind ui.PointerDeviceKind) int {
	switch kind {
	case ui.PointerDeviceKindMouse, ui.PointerDeviceKindTrackpad:
		return buttons
	case ui.PointerDeviceKindTouch, ui.PointerDeviceKindStylus, ui.PointerDeviceKindInvertedStylus:
		if buttons == 0 {
			return KPrimaryButton
		}
		return buttons
	}
	// PointerDeviceKind.unknown: we have no information about the device
	// but we know we never want buttons to be 0 when the pointer is down.
	if buttons == 0 {
		return KPrimaryButton
	}
	return buttons
}

// DevicePixelRatioGetter is the signature of a function returning the
// device pixel ratio of a view, or nil if the view doesn't exist.
type DevicePixelRatioGetter func(viewID int) *float64

// PointerEventConverter converts pointer data from the engine to framework
// pointer events (a class of static members: its statics are methods of
// this variable).
var PointerEventConverter pointerEventConverter

type pointerEventConverter struct{}

// Expand is PointerEventConverter.expand: it converts
// the engine's pointer data (physical pixels) to framework pointer events
// (logical pixels). The sequence is lazy, as Dart's Iterable: data are
// converted as it is iterated. Data of unknown signal kinds and of views
// that no longer exist are skipped, as are scroll signals with non-finite
// deltas or a non-positive device pixel ratio.
func (pointerEventConverter) Expand(data iter.Seq[ui.PointerData], devicePixelRatioForView DevicePixelRatioGetter) iter.Seq[PointerEvent] {
	return func(yield func(PointerEvent) bool) {
		for datum := range data {
			if datum.SignalKind != nil && *datum.SignalKind == ui.PointerSignalKindUnknown {
				continue
			}
			event := convertPointerData(datum, devicePixelRatioForView)
			if event == nil {
				continue
			}
			if !yield(event) {
				return
			}
		}
	}
}

func convertPointerData(datum ui.PointerData, devicePixelRatioForView DevicePixelRatioGetter) PointerEvent {
	dpr := devicePixelRatioForView(datum.ViewID)
	if dpr == nil {
		// View doesn't exist anymore.
		return nil
	}
	devicePixelRatio := *dpr
	position := ui.NewOffset(datum.PhysicalX, datum.PhysicalY).Div(devicePixelRatio)
	delta := ui.NewOffset(datum.PhysicalDeltaX, datum.PhysicalDeltaY).Div(devicePixelRatio)
	radiusMinor := toLogicalPixels(datum.RadiusMinor, devicePixelRatio)
	radiusMajor := toLogicalPixels(datum.RadiusMajor, devicePixelRatio)
	radiusMin := toLogicalPixels(datum.RadiusMin, devicePixelRatio)
	radiusMax := toLogicalPixels(datum.RadiusMax, devicePixelRatio)
	timeStamp := datum.TimeStamp
	kind := datum.Kind
	signalKind := ui.PointerSignalKindNone
	if datum.SignalKind != nil {
		signalKind = *datum.SignalKind
	}
	switch signalKind {
	case ui.PointerSignalKindNone:
		switch datum.Change {
		case ui.PointerChangeAdd:
			return NewPointerAddedEvent(PointerAddedEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Kind: kind, Device: datum.Device,
				Position: position, Obscured: datum.Obscured, PressureMin: ptr(datum.PressureMin),
				PressureMax: ptr(datum.PressureMax), Distance: datum.Distance, DistanceMax: datum.DistanceMax,
				RadiusMin: radiusMin, RadiusMax: radiusMax, Orientation: datum.Orientation, Tilt: datum.Tilt,
				EmbedderID: datum.EmbedderID,
			})
		case ui.PointerChangeHover:
			return NewPointerHoverEvent(PointerHoverEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Kind: kind, Device: datum.Device,
				Position: position, Delta: delta, Buttons: datum.Buttons, Obscured: datum.Obscured,
				PressureMin: ptr(datum.PressureMin), PressureMax: ptr(datum.PressureMax),
				Distance: datum.Distance, DistanceMax: datum.DistanceMax, Size: datum.Size,
				RadiusMajor: radiusMajor, RadiusMinor: radiusMinor, RadiusMin: radiusMin, RadiusMax: radiusMax,
				Orientation: datum.Orientation, Tilt: datum.Tilt, Synthesized: datum.Synthesized,
				EmbedderID: datum.EmbedderID,
			})
		case ui.PointerChangeDown:
			return NewPointerDownEvent(PointerDownEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Pointer: datum.PointerIdentifier, Kind: kind,
				Device: datum.Device, Position: position, Buttons: ptr(synthesiseDownButtons(datum.Buttons, kind)),
				Obscured: datum.Obscured, Pressure: ptr(datum.Pressure), PressureMin: ptr(datum.PressureMin),
				PressureMax: ptr(datum.PressureMax), DistanceMax: datum.DistanceMax, Size: datum.Size,
				RadiusMajor: radiusMajor, RadiusMinor: radiusMinor, RadiusMin: radiusMin, RadiusMax: radiusMax,
				Orientation: datum.Orientation, Tilt: datum.Tilt, EmbedderID: datum.EmbedderID,
			})
		case ui.PointerChangeMove:
			return NewPointerMoveEvent(PointerMoveEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Pointer: datum.PointerIdentifier, Kind: kind,
				Device: datum.Device, Position: position, Delta: delta,
				Buttons: ptr(synthesiseDownButtons(datum.Buttons, kind)), Obscured: datum.Obscured,
				Pressure: ptr(datum.Pressure), PressureMin: ptr(datum.PressureMin), PressureMax: ptr(datum.PressureMax),
				DistanceMax: datum.DistanceMax, Size: datum.Size, RadiusMajor: radiusMajor, RadiusMinor: radiusMinor,
				RadiusMin: radiusMin, RadiusMax: radiusMax, Orientation: datum.Orientation, Tilt: datum.Tilt,
				PlatformData: datum.PlatformData, Synthesized: datum.Synthesized, EmbedderID: datum.EmbedderID,
			})
		case ui.PointerChangeUp:
			return NewPointerUpEvent(PointerUpEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Pointer: datum.PointerIdentifier, Kind: kind,
				Device: datum.Device, Position: position, Buttons: datum.Buttons, Obscured: datum.Obscured,
				Pressure: datum.Pressure, PressureMin: ptr(datum.PressureMin), PressureMax: ptr(datum.PressureMax),
				Distance: datum.Distance, DistanceMax: datum.DistanceMax, Size: datum.Size,
				RadiusMajor: radiusMajor, RadiusMinor: radiusMinor, RadiusMin: radiusMin, RadiusMax: radiusMax,
				Orientation: datum.Orientation, Tilt: datum.Tilt, EmbedderID: datum.EmbedderID,
			})
		case ui.PointerChangeCancel:
			return NewPointerCancelEvent(PointerCancelEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Pointer: datum.PointerIdentifier, Kind: kind,
				Device: datum.Device, Position: position, Buttons: datum.Buttons, Obscured: datum.Obscured,
				PressureMin: ptr(datum.PressureMin), PressureMax: ptr(datum.PressureMax),
				Distance: datum.Distance, DistanceMax: datum.DistanceMax, Size: datum.Size,
				RadiusMajor: radiusMajor, RadiusMinor: radiusMinor, RadiusMin: radiusMin, RadiusMax: radiusMax,
				Orientation: datum.Orientation, Tilt: datum.Tilt, EmbedderID: datum.EmbedderID,
			})
		case ui.PointerChangeRemove:
			return NewPointerRemovedEvent(PointerRemovedEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Kind: kind, Device: datum.Device,
				Position: position, Obscured: datum.Obscured, PressureMin: ptr(datum.PressureMin),
				PressureMax: ptr(datum.PressureMax), DistanceMax: datum.DistanceMax,
				RadiusMin: radiusMin, RadiusMax: radiusMax, EmbedderID: datum.EmbedderID,
			})
		case ui.PointerChangePanZoomStart:
			return NewPointerPanZoomStartEvent(PointerPanZoomEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Pointer: datum.PointerIdentifier,
				Device: datum.Device, Position: position, EmbedderID: datum.EmbedderID,
				Synthesized: datum.Synthesized,
			})
		case ui.PointerChangePanZoomUpdate:
			pan := ui.NewOffset(datum.PanX, datum.PanY).Div(devicePixelRatio)
			panDelta := ui.NewOffset(datum.PanDeltaX, datum.PanDeltaY).Div(devicePixelRatio)
			return NewPointerPanZoomUpdateEvent(PointerPanZoomUpdateEventOptions{
				PointerPanZoomEventOptions: PointerPanZoomEventOptions{
					ViewID: datum.ViewID, TimeStamp: timeStamp, Pointer: datum.PointerIdentifier,
					Device: datum.Device, Position: position, EmbedderID: datum.EmbedderID,
					Synthesized: datum.Synthesized,
				},
				Pan: pan, PanDelta: panDelta, Scale: ptr(datum.Scale), Rotation: datum.Rotation,
			})
		case ui.PointerChangePanZoomEnd:
			return NewPointerPanZoomEndEvent(PointerPanZoomEventOptions{
				ViewID: datum.ViewID, TimeStamp: timeStamp, Pointer: datum.PointerIdentifier,
				Device: datum.Device, Position: position, EmbedderID: datum.EmbedderID,
				Synthesized: datum.Synthesized,
			})
		}
	case ui.PointerSignalKindScroll:
		if !isFinite(datum.ScrollDeltaX) || !isFinite(datum.ScrollDeltaY) || devicePixelRatio <= 0 {
			return nil
		}
		scrollDelta := ui.NewOffset(datum.ScrollDeltaX, datum.ScrollDeltaY).Div(devicePixelRatio)
		return NewPointerScrollEvent(PointerScrollEventOptions{
			PointerSignalEventOptions: PointerSignalEventOptions{ViewID: datum.ViewID, TimeStamp: timeStamp,
				Kind: ptr(kind), Device: datum.Device, Position: position, EmbedderID: datum.EmbedderID},
			ScrollDelta: scrollDelta, OnRespond: RespondPointerEventCallback(datum.Respond),
		})
	case ui.PointerSignalKindScrollInertiaCancel:
		return NewPointerScrollInertiaCancelEvent(PointerSignalEventOptions{ViewID: datum.ViewID,
			TimeStamp: timeStamp, Kind: ptr(kind), Device: datum.Device, Position: position,
			EmbedderID: datum.EmbedderID})
	case ui.PointerSignalKindScale:
		return NewPointerScaleEvent(PointerScaleEventOptions{
			PointerSignalEventOptions: PointerSignalEventOptions{ViewID: datum.ViewID, TimeStamp: timeStamp,
				Kind: ptr(kind), Device: datum.Device, Position: position, EmbedderID: datum.EmbedderID},
			Scale: ptr(datum.Scale),
		})
	}
	// PointerSignalKind.unknown is filtered out before.
	panic(dart.StateError{Message: "Unreachable"})
}

func isFinite(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }

func toLogicalPixels(physicalPixels, devicePixelRatio float64) float64 {
	return physicalPixels / devicePixelRatio
}
