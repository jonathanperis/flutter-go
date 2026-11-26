package ui

// Partial port of engine/src/flutter/lib/ui/platform_dispatcher.dart,
// hooks.dart and window.dart (Flutter 3.50.1): the input side of
// PlatformDispatcher (onPointerDataPacket with _unpackPointerDataPacket,
// onKeyData on the flutter/keydata channel with _unpackKeyData), the
// engine entry point _dispatchPointerDataPacket (here
// PlatformConfiguration.DispatchPointerDataPacket), and window.dart's
// GestureSettings with FlutterView's devicePixelRatio/gestureSettings.
//
// ENGINE GAP: views have no metrics yet (_updateWindowMetrics comes with
// the rendering port): a view's devicePixelRatio is 1.0 and its gesture
// settings are empty until an embedder sets them with
// PlatformConfiguration.UpdateViewGestureMetrics.

import (
	"encoding/binary"
	"math"
	"strconv"
	"time"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// PointerDataPacketCallback is the signature of
// PlatformDispatcher.onPointerDataPacket.
type PointerDataPacketCallback func(packet PointerDataPacket)

// KeyDataCallback is the signature of PlatformDispatcher.onKeyData: it
// returns whether the framework handled the event.
type KeyDataCallback func(data KeyData) bool

// platformDispatcherInput is the state of PlatformDispatcher's input
// members.
type platformDispatcherInput struct {
	onPointerDataPacket PointerDataPacketCallback
	onKeyData           KeyDataCallback
}

// kFlutterKeyDataChannel is _kFlutterKeyDataChannel.
const kFlutterKeyDataChannel = "flutter/keydata"

// OnPointerDataPacket is called when pointer data is available.
func (p *PlatformDispatcher) OnPointerDataPacket() PointerDataPacketCallback {
	return p.input.onPointerDataPacket
}

// SetOnPointerDataPacket sets OnPointerDataPacket.
func (p *PlatformDispatcher) SetOnPointerDataPacket(callback PointerDataPacketCallback) {
	p.input.onPointerDataPacket = callback
}

func (p *PlatformDispatcher) dispatchPointerDataPacket(packet []byte) {
	if p.input.onPointerDataPacket != nil {
		p.input.onPointerDataPacket(unpackPointerDataPacket(packet))
	}
}

// kPointerDataFieldCount is _kPointerDataFieldCount (pointer_data.cc).
const kPointerDataFieldCount = 36

// kStride is Int64List.bytesPerElement.
const kStride = 8

// fakeHostEndian is _kFakeHostEndian (Endian.host).
var fakeHostEndian = binary.NativeEndian

// unpackPointerDataPacket is _unpackPointerDataPacket: the engine's packet
// of 36 host-endian 64-bit fields per pointer (pointer_data.h).
func unpackPointerDataPacket(packet []byte) PointerDataPacket {
	const kBytesPerPointerData = kPointerDataFieldCount * kStride
	length := len(packet) / kBytesPerPointerData
	if length*kBytesPerPointerData != len(packet) {
		panic(dart.NewAssertionError(nil))
	}
	data := make([]PointerData, 0, length)
	for i := 0; i < length; i++ {
		offset := i * kPointerDataFieldCount
		i64 := func() int {
			v := int(int64(fakeHostEndian.Uint64(packet[kStride*offset:])))
			offset++
			return v
		}
		f64 := func() float64 {
			v := math.Float64frombits(fakeHostEndian.Uint64(packet[kStride*offset:]))
			offset++
			return v
		}
		var d PointerData
		d.EmbedderID = i64()
		d.TimeStamp = time.Duration(i64()) * time.Microsecond
		d.Change = PointerChangeValues[i64()]
		d.Kind = PointerDeviceKindValues[i64()]
		signalKind := PointerSignalKindValues[i64()]
		d.SignalKind = &signalKind
		d.Device = i64()
		d.PointerIdentifier = i64()
		d.PhysicalX = f64()
		d.PhysicalY = f64()
		d.PhysicalDeltaX = f64()
		d.PhysicalDeltaY = f64()
		d.Buttons = i64()
		d.Obscured = i64() != 0
		d.Synthesized = i64() != 0
		d.Pressure = f64()
		d.PressureMin = f64()
		d.PressureMax = f64()
		d.Distance = f64()
		d.DistanceMax = f64()
		d.Size = f64()
		d.RadiusMajor = f64()
		d.RadiusMinor = f64()
		d.RadiusMin = f64()
		d.RadiusMax = f64()
		d.Orientation = f64()
		d.Tilt = f64()
		d.PlatformData = i64()
		d.ScrollDeltaX = f64()
		d.ScrollDeltaY = f64()
		d.PanX = f64()
		d.PanY = f64()
		d.PanDeltaX = f64()
		d.PanDeltaY = f64()
		d.Scale = f64()
		d.Rotation = f64()
		d.ViewID = i64()
		if offset != (i+1)*kPointerDataFieldCount {
			panic(dart.NewAssertionError(nil))
		}
		data = append(data, d)
	}
	return PointerDataPacket{Data: data}
}

// PackPointerDataPacket is the engine's side of the pointer data packet
// (PointerDataPacket in pointer_data_packet.cc): the bytes
// PlatformConfiguration.DispatchPointerDataPacket unpacks. SignalKind nil
// is packed as none; OnRespond is not packed.
func PackPointerDataPacket(packet PointerDataPacket) []byte {
	out := make([]byte, 0, len(packet.Data)*kPointerDataFieldCount*kStride)
	i64 := func(v int) { out = fakeHostEndian.AppendUint64(out, uint64(int64(v))) }
	f64 := func(v float64) { out = fakeHostEndian.AppendUint64(out, math.Float64bits(v)) }
	b := func(v bool) {
		if v {
			i64(1)
		} else {
			i64(0)
		}
	}
	for _, d := range packet.Data {
		signalKind := PointerSignalKindNone
		if d.SignalKind != nil {
			signalKind = *d.SignalKind
		}
		i64(d.EmbedderID)
		i64(int(d.TimeStamp / time.Microsecond))
		i64(int(d.Change))
		i64(int(d.Kind))
		i64(int(signalKind))
		i64(d.Device)
		i64(d.PointerIdentifier)
		f64(d.PhysicalX)
		f64(d.PhysicalY)
		f64(d.PhysicalDeltaX)
		f64(d.PhysicalDeltaY)
		i64(d.Buttons)
		b(d.Obscured)
		b(d.Synthesized)
		for _, v := range []float64{d.Pressure, d.PressureMin, d.PressureMax, d.Distance, d.DistanceMax,
			d.Size, d.RadiusMajor, d.RadiusMinor, d.RadiusMin, d.RadiusMax, d.Orientation, d.Tilt} {
			f64(v)
		}
		i64(d.PlatformData)
		for _, v := range []float64{d.ScrollDeltaX, d.ScrollDeltaY, d.PanX, d.PanY, d.PanDeltaX, d.PanDeltaY,
			d.Scale, d.Rotation} {
			f64(v)
		}
		i64(d.ViewID)
	}
	return out
}

// DispatchPointerDataPacket is the engine delivering a packet of pointer
// data (hooks.dart's _dispatchPointerDataPacket) to
// PlatformDispatcher.OnPointerDataPacket.
func (pc *PlatformConfiguration) DispatchPointerDataPacket(packet []byte) {
	PlatformDispatcherInstance().dispatchPointerDataPacket(packet)
}

// OnKeyData is called when key data is available; it returns whether the
// framework handled the event.
func (p *PlatformDispatcher) OnKeyData() KeyDataCallback { return p.input.onKeyData }

// SetOnKeyData sets OnKeyData: the callback listens on the flutter/keydata
// channel of ChannelBuffersInstance (nil clears the listener).
func (p *PlatformDispatcher) SetOnKeyData(callback KeyDataCallback) {
	p.input.onKeyData = callback
	if callback != nil {
		ChannelBuffersInstance().SetListener(kFlutterKeyDataChannel, keyDataListener(callback))
	} else {
		ChannelBuffersInstance().ClearListener(kFlutterKeyDataChannel)
	}
}

// keyDataListener is _keyDataListener: it decodes the packet, calls
// onKeyData and replies with one byte (1: handled).
func keyDataListener(onKeyData KeyDataCallback) ChannelCallback {
	return func(packet []byte, callback PlatformMessageResponseCallback) {
		handled := onKeyData(unpackKeyData(packet))
		response := []byte{0}
		if handled {
			response[0] = 1
		}
		callback(response)
	}
}

// kKeyDataFieldCount is _kKeyDataFieldCount (key_data.h).
const kKeyDataFieldCount = 6

// unpackKeyData is _unpackKeyData (key_data_packet.h): the character's
// size, then the fields, then the character's UTF-8 bytes.
func unpackKeyData(packet []byte) KeyData {
	offset := 0
	u64 := func() uint64 {
		v := fakeHostEndian.Uint64(packet[kStride*offset:])
		offset++
		return v
	}
	charDataSize := int(u64())
	var character *string
	if charDataSize != 0 {
		start := kStride * (offset + kKeyDataFieldCount)
		s := dart.UTF8Decode(packet[start : start+charDataSize])
		character = &s
	}
	var k KeyData
	k.TimeStamp = time.Duration(int64(u64())) * time.Microsecond
	k.Type = KeyEventTypeValues[int64(u64())]
	k.Physical = int(int64(u64()))
	k.Logical = int(int64(u64()))
	k.Character = character
	k.Synthesized = u64() != 0
	return k
}

// PackKeyDataPacket is the engine's side of a key data packet
// (KeyDataPacket in key_data_packet.cc): the character's size, the six
// KeyData fields (timestamp, type, physical, logical, synthesized, device
// type) and the character's UTF-8 bytes, as unpackKeyData reads them.
func PackKeyDataPacket(k KeyData) []byte {
	var character []byte
	if k.Character != nil {
		character = dart.UTF8Encode(*k.Character)
	}
	out := make([]byte, 0, kStride*(1+kKeyDataFieldCount)+len(character))
	u64 := func(v uint64) { out = fakeHostEndian.AppendUint64(out, v) }
	u64(uint64(len(character)))
	u64(uint64(k.TimeStamp / time.Microsecond))
	u64(uint64(k.Type))
	u64(uint64(int64(k.Physical)))
	u64(uint64(int64(k.Logical)))
	if k.Synthesized {
		u64(1)
	} else {
		u64(0)
	}
	u64(uint64(k.DeviceType))
	return append(out, character...)
}

// ---- GestureSettings (window.dart)

// GestureSettings are the platform's gesture settings (in physical
// pixels); nil fields are null (no platform value).
type GestureSettings struct {
	PhysicalTouchSlop     *float64
	PhysicalDoubleTapSlop *float64
}

// GestureSettingsCopyWithOptions are copyWith's parameters (nil: keep).
type GestureSettingsCopyWithOptions struct {
	PhysicalTouchSlop     *float64
	PhysicalDoubleTapSlop *float64
}

// CopyWith creates a new GestureSettings with the given fields replaced.
func (g GestureSettings) CopyWith(opts ...GestureSettingsCopyWithOptions) GestureSettings {
	var o GestureSettingsCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	r := g
	if o.PhysicalTouchSlop != nil {
		r.PhysicalTouchSlop = o.PhysicalTouchSlop
	}
	if o.PhysicalDoubleTapSlop != nil {
		r.PhysicalDoubleTapSlop = o.PhysicalDoubleTapSlop
	}
	return r
}

// Equal is ==.
func (g GestureSettings) Equal(other GestureSettings) bool {
	return nullableDoubleEqual(g.PhysicalTouchSlop, other.PhysicalTouchSlop) &&
		nullableDoubleEqual(g.PhysicalDoubleTapSlop, other.PhysicalDoubleTapSlop)
}

func nullableDoubleEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func nullableDoubleString(v *float64) string {
	if v == nil {
		return "null"
	}
	return dart.DoubleToString(*v)
}

func (g GestureSettings) String() string {
	return "GestureSettings(physicalTouchSlop: " + nullableDoubleString(g.PhysicalTouchSlop) +
		", physicalDoubleTapSlop: " + nullableDoubleString(g.PhysicalDoubleTapSlop) + ")"
}

// viewGestureMetrics are the view metrics gestures use, by view id.
var viewGestureMetrics = map[int]struct {
	devicePixelRatio float64
	gestureSettings  GestureSettings
}{}

// DevicePixelRatio is the number of device pixels for each logical pixel
// of the view (1.0 until the embedder reports it; see the file comment).
func (v *FlutterView) DevicePixelRatio() float64 {
	if m, ok := viewGestureMetrics[v.viewID]; ok {
		return m.devicePixelRatio
	}
	return 1.0
}

// GestureSettings are the view's gesture settings.
func (v *FlutterView) GestureSettings() GestureSettings {
	return viewGestureMetrics[v.viewID].gestureSettings
}

// UpdateViewGestureMetrics is the part of the engine's window metrics
// update (hooks.dart's _updateWindowMetrics) gestures read: the view's
// device pixel ratio and gesture settings.
func (pc *PlatformConfiguration) UpdateViewGestureMetrics(viewID int, devicePixelRatio float64, settings GestureSettings) {
	if devicePixelRatio <= 0 {
		panic(dart.NewArgumentErrorValue(devicePixelRatio, "devicePixelRatio", "must be positive ("+strconv.Itoa(viewID)+")"))
	}
	viewGestureMetrics[viewID] = struct {
		devicePixelRatio float64
		gestureSettings  GestureSettings
	}{devicePixelRatio, settings}
}
