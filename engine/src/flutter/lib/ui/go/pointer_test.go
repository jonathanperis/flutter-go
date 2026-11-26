package ui

// Tests of pointer.go, key.go and pointer_dispatch.go. The engine's C++
// pointer_data_packet_unittests (a packed datum reads back) are covered by
// the pack/unpack round trip; there are no Dart tests of these classes in
// the engine (their toString output is checked by the gestures difftest).

import (
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
)

func TestPointerDataToString(t *testing.T) {
	d := PointerData{ViewID: 2, PhysicalX: 1.5, PhysicalY: 3}
	if got := d.String(); got != "PointerData(viewId: 2, x: 1.5, y: 3.0)" {
		t.Errorf("toString %q", got)
	}
	scroll := PointerSignalKindScroll
	d.SignalKind = &scroll
	d.TimeStamp = 1500 * time.Microsecond
	want := "PointerData(embedderId: 0, timeStamp: 0:00:00.001500, change: PointerChange.cancel, " +
		"kind: PointerDeviceKind.touch, signalKind: PointerSignalKind.scroll, device: 0, pointerIdentifier: 0, " +
		"physicalX: 1.5, physicalY: 3.0, physicalDeltaX: 0.0, physicalDeltaY: 0.0, buttons: 0, synthesized: false, " +
		"pressure: 0.0, pressureMin: 0.0, pressureMax: 0.0, distance: 0.0, distanceMax: 0.0, size: 0.0, " +
		"radiusMajor: 0.0, radiusMinor: 0.0, radiusMin: 0.0, radiusMax: 0.0, orientation: 0.0, tilt: 0.0, " +
		"platformData: 0, scrollDeltaX: 0.0, scrollDeltaY: 0.0, panX: 0.0, panY: 0.0, panDeltaX: 0.0, " +
		"panDeltaY: 0.0, scale: 0.0, rotation: 0.0, viewId: 2)"
	if got := d.ToStringFull(); got != want {
		t.Errorf("toStringFull\n%s\nwant\n%s", got, want)
	}
}

func TestPointerDataRespond(t *testing.T) {
	PointerData{}.Respond(true) // no callback: nothing happens
	var got *bool
	PointerData{OnRespond: func(allow bool) { got = &allow }}.Respond(false)
	if got == nil || *got {
		t.Fatal("respond")
	}
}

func TestPointerDataPacketRoundTrip(t *testing.T) {
	scale := PointerSignalKindScale
	in := PointerDataPacket{Data: []PointerData{
		{EmbedderID: 7, TimeStamp: 123456 * time.Microsecond, Change: PointerChangeMove, Kind: PointerDeviceKindStylus,
			Device: 3, PointerIdentifier: 9, PhysicalX: 1.25, PhysicalY: -2.5, PhysicalDeltaX: 0.5, PhysicalDeltaY: 0.75,
			Buttons: 5, Obscured: true, Pressure: 0.3, PressureMin: 0.1, PressureMax: 0.9, Distance: 1, DistanceMax: 2,
			Size: 3, RadiusMajor: 4, RadiusMinor: 5, RadiusMin: 6, RadiusMax: 7, Orientation: 8, Tilt: 9,
			PlatformData: 11, ScrollDeltaX: 12, ScrollDeltaY: 13, PanX: 14, PanY: 15, PanDeltaX: 16, PanDeltaY: 17,
			Scale: 18, Rotation: 19, ViewID: 1},
		{Change: PointerChangePanZoomUpdate, Kind: PointerDeviceKindTrackpad, SignalKind: &scale, Synthesized: true},
	}}
	packet := PackPointerDataPacket(in)
	if len(packet) != 2*36*8 {
		t.Fatalf("packet length %d", len(packet))
	}
	out := unpackPointerDataPacket(packet)
	if len(out.Data) != 2 {
		t.Fatal("length")
	}
	for i := range in.Data {
		a, b := in.Data[i], out.Data[i]
		if b.SignalKind == nil {
			t.Fatal("unpacked signal kind is never null")
		}
		if a.SignalKind == nil && *b.SignalKind != PointerSignalKindNone {
			t.Fatal("signal kind none")
		}
		a.SignalKind, b.SignalKind = nil, nil
		if a.ToStringFull() != b.ToStringFull() {
			t.Errorf("datum %d\n%s\n%s", i, a.ToStringFull(), b.ToStringFull())
		}
	}
}

func TestDispatchPointerDataPacket(t *testing.T) {
	pd := PlatformDispatcherInstance()
	defer pd.SetOnPointerDataPacket(nil)
	var got []PointerDataPacket
	pd.SetOnPointerDataPacket(func(p PointerDataPacket) { got = append(got, p) })
	pc := NewPlatformConfiguration(nil)
	pc.DispatchPointerDataPacket(PackPointerDataPacket(PointerDataPacket{Data: []PointerData{{PhysicalX: 4}}}))
	if len(got) != 1 || got[0].Data[0].PhysicalX != 4 {
		t.Fatalf("got %v", got)
	}
	pd.SetOnPointerDataPacket(nil)
	pc.DispatchPointerDataPacket(PackPointerDataPacket(PointerDataPacket{Data: []PointerData{{}}}))
	if len(got) != 1 {
		t.Fatal("dispatched without a callback")
	}
}

func TestKeyDataToString(t *testing.T) {
	a := "a"
	k := KeyData{TimeStamp: time.Second, Type: KeyEventTypeDown, Physical: 0x70004, Logical: 0x61, Character: &a}
	if got := k.String(); got != `KeyData(Key Down, physical: 0x70004, logical: 0x61 (Unicode), character: "a" (0x61)` {
		t.Errorf("toString %q", got)
	}
	nl := "\n"
	k = KeyData{Type: KeyEventTypeRepeat, DeviceType: KeyEventDeviceTypeGamepad, Physical: 0x100070028,
		Logical: 0x200000100, Character: &nl, Synthesized: true}
	if got := k.String(); got != `KeyData(Key Repeat, physical: 0x100070028, logical: 0x200000100 (Flutter), character: "\n" (0x0a), synthesized` {
		t.Errorf("toString %q", got)
	}
	if got := k.ToStringFull(); got != `KeyData(type: Key Repeat, deviceType: Gamepad, timeStamp: 0:00:00.000000, physical: 0x100070028, logical: 0x200000100, character: "\n", synthesized: true)` {
		t.Errorf("toStringFull %q", got)
	}
	k = KeyData{Type: KeyEventTypeUp, Logical: 0x1400000000}
	if got := k.String(); got != `KeyData(Key Up, physical: 0x0, logical: 0x1400000000 (macOS), character: <none>` {
		t.Errorf("toString %q", got)
	}
	// A surrogate pair prints both code units.
	smile := "😀"
	if got := (KeyData{Character: &smile}).String(); got != `KeyData(Key Down, physical: 0x0, logical: 0x0 (Unicode), character: "😀" (0xd83d de00)` {
		t.Errorf("toString %q", got)
	}
}

func TestOnKeyData(t *testing.T) {
	pd := PlatformDispatcherInstance()
	defer pd.SetOnKeyData(nil)
	var got []KeyData
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	pd.SetOnKeyData(func(k KeyData) bool { got = append(got, k); return k.Logical == 0x61 })
	loop.FlushMicrotasks() // the channel drains its (empty) buffer in a microtask
	a := "é"
	var replies [][]byte
	in := KeyData{TimeStamp: 42 * time.Microsecond, Type: KeyEventTypeUp, Physical: 0x70004, Logical: 0x61, Character: &a,
		Synthesized: true, DeviceType: KeyEventDeviceTypeHdmi}
	ChannelBuffersInstance().Push(kFlutterKeyDataChannel, PackKeyDataPacket(in), func(r []byte) { replies = append(replies, r) })
	ChannelBuffersInstance().Push(kFlutterKeyDataChannel, PackKeyDataPacket(KeyData{Logical: 0x62}), func(r []byte) { replies = append(replies, r) })
	if len(got) != 2 || len(replies) != 2 || replies[0][0] != 1 || replies[1][0] != 0 {
		t.Fatalf("got %v replies %v", got, replies)
	}
	// Dart's unpacking doesn't read the device type.
	in.DeviceType = KeyEventDeviceTypeKeyboard
	if got[0].ToStringFull() != in.ToStringFull() || got[1].Character != nil {
		t.Errorf("unpacked %s", got[0].ToStringFull())
	}
}

func TestGestureSettings(t *testing.T) {
	one := 1.0
	g := GestureSettings{}.CopyWith(GestureSettingsCopyWithOptions{PhysicalTouchSlop: &one})
	if g.String() != "GestureSettings(physicalTouchSlop: 1.0, physicalDoubleTapSlop: null)" ||
		!g.Equal(GestureSettings{PhysicalTouchSlop: &one}) || g.Equal(GestureSettings{}) {
		t.Fatalf("%v", g)
	}
	v := PlatformDispatcherInstance().ImplicitView()
	if v.DevicePixelRatio() != 1.0 || !v.GestureSettings().Equal(GestureSettings{}) {
		t.Fatal("defaults")
	}
	saved := v.configuration
	defer func() { v.configuration = saved }()
	m := NewViewportMetrics()
	m.DevicePixelRatio, m.PhysicalTouchSlop = 3, 1
	NewPlatformConfiguration(nil).UpdateViewMetrics(0, m)
	if v.DevicePixelRatio() != 3 || !v.GestureSettings().Equal(g) {
		t.Fatal("updated metrics")
	}
}
