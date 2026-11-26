package difftest

// Go twin of flutter/test/pointer_events_test.dart: the gestures library's
// base layer (events, transforms, copyWith, the converter, the resampler,
// the arena with its diagnostics, the router, the signal resolver, hit
// test results, drag details, PointerData and KeyData), logged the same
// way and compared exactly (identity hashes masked).

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	g "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var peHash = regexp.MustCompile(`#[0-9a-f]{5}`)

func peMask(s string) string { return peHash.ReplaceAllString(s, "#HASH") }

func peMat(m *vm.Matrix4) string {
	if m == nil {
		return "null"
	}
	parts := make([]string, 16)
	for i, v := range m.Storage() {
		parts[i] = ds(v)
	}
	return strings.Join(parts, ",")
}

func peType(e foundation.Diagnosticable) string { return strings.Split(e.ToStringShort(), "#")[0] }

func peBool(b bool) string { return strconv.FormatBool(b) }

func peFields(e g.PointerEvent) string {
	o := vtOff
	parts := []string{
		peType(e), strconv.Itoa(e.ViewID()), strconv.Itoa(e.EmbedderID()), strconv.Itoa(int(e.TimeStamp() / time.Microsecond)),
		strconv.Itoa(e.Pointer()), e.Kind().Name(), strconv.Itoa(e.Device()), o(e.Position()), o(e.Delta()),
		o(e.LocalPosition()), o(e.LocalDelta()), strconv.Itoa(e.Buttons()), peBool(e.Down()), peBool(e.Obscured()),
		ds(e.Pressure()), ds(e.PressureMin()), ds(e.PressureMax()), ds(e.Distance()), ds(e.DistanceMin()),
		ds(e.DistanceMax()), ds(e.Size()), ds(e.RadiusMajor()), ds(e.RadiusMinor()), ds(e.RadiusMin()),
		ds(e.RadiusMax()), ds(e.Orientation()), ds(e.Tilt()), strconv.Itoa(e.PlatformData()), peBool(e.Synthesized()),
		peMat(e.Transform()),
	}
	if e.Original() == nil {
		parts = append(parts, "orig:null")
	} else {
		parts = append(parts, "orig:"+peType(e.Original()))
	}
	switch x := e.(type) {
	case *g.PointerScrollEvent:
		parts = append(parts, "scrollDelta "+o(x.ScrollDelta()))
	case *g.PointerScaleEvent:
		parts = append(parts, "scale "+ds(x.Scale()))
	case *g.PointerPanZoomUpdateEvent:
		parts = append(parts, "pan "+o(x.Pan())+" "+o(x.LocalPan())+" "+o(x.PanDelta())+" "+o(x.LocalPanDelta())+" "+
			ds(x.Scale())+" "+ds(x.Rotation()))
	}
	if _, ok := e.(g.PointerSignalEvent); ok {
		parts = append(parts, "signal")
	}
	return strings.Join(parts, " ")
}

type peLog struct{ lines []string }

func (l *peLog) log(s string) { l.lines = append(l.lines, s) }

func (l *peLog) describe(label string, e g.PointerEvent) {
	l.log(label + " " + peFields(e))
	l.log("  toString " + peMask(e.String()))
	l.log("  short " + peMask(e.ToStringShort()))
	l.log("  full " + peMask(e.ToStringFull()))
	l.log("  deep " + peMask(e.ToDiagnosticsNode().ToStringDeep()))
	builder := foundation.NewDiagnosticPropertiesBuilder()
	e.DebugFillProperties(builder)
	var props []string
	for _, p := range builder.Properties {
		props = append(props, peMask(p.String())+"/"+p.Level().String()+"/"+peBool(p.IsFiltered(foundation.DiagnosticLevelInfo)))
	}
	l.log("  props " + strings.Join(props, " | "))
}

func peEvents() []struct {
	label string
	e     g.PointerEvent
} {
	us := func(n int) time.Duration { return time.Duration(n) * time.Microsecond }
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	off := ui.NewOffset
	K := struct{ touch, mouse, stylus, inverted, trackpad, unknown ui.PointerDeviceKind }{
		ui.PointerDeviceKindTouch, ui.PointerDeviceKindMouse, ui.PointerDeviceKindStylus,
		ui.PointerDeviceKindInvertedStylus, ui.PointerDeviceKindTrackpad, ui.PointerDeviceKindUnknown}
	type ev = struct {
		label string
		e     g.PointerEvent
	}
	return []ev{
		{"added", g.NewPointerAddedEvent()},
		{"added full", g.NewPointerAddedEvent(g.PointerAddedEventOptions{ViewID: 3, TimeStamp: us(1234567), Pointer: 4, Kind: K.stylus,
			Device: 5, Position: off(10.5, -20.25), Obscured: true, PressureMin: pf(0.1), PressureMax: pf(2.5), Distance: 1.1,
			DistanceMax: 7.7, RadiusMin: 0.3, RadiusMax: 9.9, Orientation: 0.7, Tilt: 0.2, EmbedderID: 12})},
		{"removed", g.NewPointerRemovedEvent()},
		{"removed full", g.NewPointerRemovedEvent(g.PointerRemovedEventOptions{ViewID: 3, TimeStamp: ms(77), Pointer: 6, Kind: K.mouse,
			Device: 2, Position: off(1.25, 2.5), Obscured: true, PressureMin: pf(0.5), PressureMax: pf(1.5), DistanceMax: 3.3,
			RadiusMin: 0.25, RadiusMax: 4.75, EmbedderID: 8})},
		{"hover", g.NewPointerHoverEvent()},
		{"hover full", g.NewPointerHoverEvent(g.PointerHoverEventOptions{ViewID: 1, TimeStamp: 3 * time.Second, Kind: K.mouse, Pointer: 2,
			Device: 3, Position: off(100.125, 200.5), Delta: off(-1.5, 2.25), Buttons: 6, Obscured: true, PressureMin: pf(0.2),
			PressureMax: pf(3.1), Distance: 0.4, DistanceMax: 5.5, Size: 0.33, RadiusMajor: 6.6, RadiusMinor: 2.2, RadiusMin: 1.1,
			RadiusMax: 8.8, Orientation: -0.5, Tilt: 0.9, Synthesized: true, EmbedderID: 99})},
		{"enter", g.NewPointerEnterEvent()},
		{"enter full", g.NewPointerEnterEvent(g.PointerEnterEventOptions{PointerHoverEventOptions: g.PointerHoverEventOptions{
			TimeStamp: 24 * time.Hour, Pointer: 7, Kind: K.inverted, Device: 1, Position: off(3, 4), Delta: off(0.1, 0.2), Buttons: 1,
			Obscured: true, PressureMin: pf(0.0), PressureMax: pf(0.0), Distance: 2, DistanceMax: 4, Size: 1, RadiusMajor: 2,
			RadiusMinor: 1, RadiusMin: 0.5, RadiusMax: 3, Orientation: 1, Tilt: 1, Synthesized: true, EmbedderID: 4}, Down: true})},
		{"exit", g.NewPointerExitEvent()},
		{"exit full", g.NewPointerExitEvent(g.PointerEnterEventOptions{PointerHoverEventOptions: g.PointerHoverEventOptions{
			TimeStamp: 2 * time.Hour, Pointer: 8, Kind: K.unknown, Device: 9, Position: off(-3, -4), Delta: off(-0.1, -0.2),
			Buttons: 2, Synthesized: true}, Down: true})},
		{"down", g.NewPointerDownEvent()},
		{"down full", g.NewPointerDownEvent(g.PointerDownEventOptions{ViewID: 2, TimeStamp: us(999), Pointer: 11, Kind: K.stylus,
			Device: 4, Position: off(55.5, 66.25), Buttons: pi(3), Obscured: true, Pressure: pf(0.75), PressureMin: pf(0.25),
			PressureMax: pf(1.75), DistanceMax: 2.5, Size: 0.125, RadiusMajor: 3.5, RadiusMinor: 1.5, RadiusMin: 0.5,
			RadiusMax: 4.5, Orientation: 0.3, Tilt: 0.6, EmbedderID: 21})},
		{"move", g.NewPointerMoveEvent()},
		{"move full", g.NewPointerMoveEvent(g.PointerMoveEventOptions{ViewID: 2, TimeStamp: us(16999), Pointer: 11, Kind: K.touch,
			Device: 4, Position: off(57.5, 60.25), Delta: off(2.0, -6.0), Buttons: pi(1), Obscured: true, Pressure: pf(0.85),
			PressureMin: pf(0.25), PressureMax: pf(1.75), DistanceMax: 2.5, Size: 0.125, RadiusMajor: 3.5, RadiusMinor: 1.5,
			RadiusMin: 0.5, RadiusMax: 4.5, Orientation: 0.3, Tilt: 0.6, PlatformData: 42, Synthesized: true, EmbedderID: 22})},
		{"up", g.NewPointerUpEvent()},
		{"up full", g.NewPointerUpEvent(g.PointerUpEventOptions{ViewID: 2, TimeStamp: us(33999), Pointer: 11, Kind: K.touch,
			Device: 4, Position: off(57.5, 60.25), Buttons: 0, Obscured: true, Pressure: 0.05, PressureMin: pf(0.25),
			PressureMax: pf(1.75), Distance: 0.5, DistanceMax: 2.5, Size: 0.125, RadiusMajor: 3.5, RadiusMinor: 1.5,
			RadiusMin: 0.5, RadiusMax: 4.5, Orientation: 0.3, Tilt: 0.6, EmbedderID: 23})},
		{"cancel", g.NewPointerCancelEvent()},
		{"cancel full", g.NewPointerCancelEvent(g.PointerCancelEventOptions{ViewID: 2, TimeStamp: us(1), Pointer: 12, Kind: K.mouse,
			Device: 1, Position: off(1, 1), Buttons: 4, Obscured: true, PressureMin: pf(0.5), PressureMax: pf(0.5), Distance: 1,
			DistanceMax: 2, Size: 3, RadiusMajor: 4, RadiusMinor: 5, RadiusMin: 6, RadiusMax: 7, Orientation: 8, Tilt: 9,
			EmbedderID: 10})},
		{"scroll", g.NewPointerScrollEvent()},
		{"scroll full", g.NewPointerScrollEvent(g.PointerScrollEventOptions{PointerSignalEventOptions: g.PointerSignalEventOptions{
			ViewID: 1, TimeStamp: ms(5), Kind: &K.trackpad, Device: 2, Position: off(30, 40), EmbedderID: 6},
			ScrollDelta: off(-3.5, 120.25)})},
		{"inertia", g.NewPointerScrollInertiaCancelEvent()},
		{"inertia full", g.NewPointerScrollInertiaCancelEvent(g.PointerSignalEventOptions{ViewID: 1, TimeStamp: ms(6),
			Kind: &K.trackpad, Device: 2, Position: off(31, 41), EmbedderID: 7})},
		{"scale", g.NewPointerScaleEvent()},
		{"scale full", g.NewPointerScaleEvent(g.PointerScaleEventOptions{PointerSignalEventOptions: g.PointerSignalEventOptions{
			ViewID: 1, TimeStamp: ms(7), Kind: &K.touch, Device: 2, Position: off(32, 42), EmbedderID: 8}, Scale: pf(1.75)})},
		{"pzstart", g.NewPointerPanZoomStartEvent()},
		{"pzstart full", g.NewPointerPanZoomStartEvent(g.PointerPanZoomEventOptions{ViewID: 1, TimeStamp: ms(8), Device: 3,
			Pointer: 4, Position: off(33, 43), EmbedderID: 9, Synthesized: true})},
		{"pzupdate", g.NewPointerPanZoomUpdateEvent()},
		{"pzupdate full", g.NewPointerPanZoomUpdateEvent(g.PointerPanZoomUpdateEventOptions{PointerPanZoomEventOptions: g.PointerPanZoomEventOptions{
			ViewID: 1, TimeStamp: ms(9), Device: 3, Pointer: 4, Position: off(33, 43), EmbedderID: 10, Synthesized: true},
			Pan: off(12.5, -7.25), PanDelta: off(1.5, -0.75), Scale: pf(1.2), Rotation: 0.35})},
		{"pzend", g.NewPointerPanZoomEndEvent()},
		{"pzend full", g.NewPointerPanZoomEndEvent(g.PointerPanZoomEventOptions{ViewID: 1, TimeStamp: ms(10), Device: 3,
			Pointer: 4, Position: off(34, 44), EmbedderID: 11, Synthesized: true})},
	}
}

func peMatrices() []struct {
	name string
	m    *vm.Matrix4
} {
	perspective := vm.NewMatrix4Identity()
	perspective.SetEntry(3, 0, 0.001)
	perspective.SetEntry(3, 1, -0.002)
	rotate := vm.NewMatrix4RotationZ(0.7)
	rotate.SetTranslationRaw(3, 4, 0)
	composite := vm.NewMatrix4Identity()
	composite.ScaleByDouble(1/3.0, 1/7.0, 1.0, 1.0)
	return []struct {
		name string
		m    *vm.Matrix4
	}{
		{"translate", vm.NewMatrix4TranslationValues(10, -20, 0)},
		{"scale", vm.NewMatrix4Diagonal3Values(2.5, 0.5, 1)},
		{"rotate", rotate},
		{"perspective", perspective},
		{"composite", composite.Multiplied(vm.NewMatrix4TranslationValues(-10.5, 30.25, 0))},
	}
}

type peMember struct {
	l     *peLog
	name  string
	entry g.GestureArenaEntry
}

func (m *peMember) AcceptGesture(pointer int) {
	m.l.log("  " + m.name + " accept " + strconv.Itoa(pointer))
}
func (m *peMember) RejectGesture(pointer int) {
	m.l.log("  " + m.name + " reject " + strconv.Itoa(pointer))
}
func (m *peMember) String() string { return "Member(" + m.name + ")" }

type HtTarget struct{ name string }

func (*HtTarget) HandleEvent(g.PointerEvent, g.HitTestEntry) {}
func (t *HtTarget) String() string                           { return "HtTarget(" + t.name + ")" }

func runPointerEventsScene() []string {
	l := &peLog{}
	log := l.log
	savedPrint := foundation.DebugPrint
	foundation.DebugPrint = func(message string, opts ...foundation.DebugPrintOptions) { log("debugPrint " + message) }
	defer func() { foundation.DebugPrint = savedPrint }()

	// ---- events, transformed, copyWith
	for _, ev := range peEvents() {
		label, e := ev.label, ev.e
		l.describe(label, e)
		log("  transformed(null) identical " + peBool(e.Transformed(nil) == e))
		for _, mm := range peMatrices() {
			t := e.Transformed(mm.m)
			l.describe(label+" * "+mm.name, t)
			log("  original identical " + peBool(t.Original() == e) + " again " + peBool(t.Transformed(mm.m) == t) +
				" back " + peBool(t.Transformed(nil) == e) + " twice-original " +
				peBool(t.Transformed(vm.NewMatrix4Identity()).Original() == e))
			l.describe(label+" * "+mm.name+" copy", t.CopyWith())
			ts := 4242 * time.Microsecond
			pos, delta := ui.NewOffset(1.5, 2.5), ui.NewOffset(0.25, -0.5)
			f, tr := false, true
			l.describe(label+" * "+mm.name+" copy2", t.CopyWith(g.PointerEventCopyWithOptions{
				ViewID: pi(5), TimeStamp: &ts, Pointer: pi(77), Device: pi(13), Position: &pos, Delta: &delta, Buttons: pi(2),
				Obscured: &f, Pressure: pf(0.6), PressureMin: pf(0.3), PressureMax: pf(0.9), Distance: pf(4.4),
				DistanceMax: pf(8.8), Size: pf(0.7), RadiusMajor: pf(2.2), RadiusMinor: pf(1.1), RadiusMin: pf(0.6),
				RadiusMax: pf(3.3), Orientation: pf(0.15), Tilt: pf(0.25), Synthesized: &tr, EmbedderID: pi(31)}))
		}
		l.describe(label+" copy", e.CopyWith())
	}
	hover := g.NewPointerHoverEvent(g.PointerHoverEventOptions{Position: ui.NewOffset(5, 6), Delta: ui.NewOffset(1, 1),
		Kind: ui.PointerDeviceKindMouse, Device: 3, Buttons: 1, EmbedderID: 4})
	tm := vm.NewMatrix4TranslationValues(-1, -2, 0)
	l.describe("enter.fromMouseEvent", g.NewPointerEnterEventFromMouseEvent(hover))
	l.describe("exit.fromMouseEvent", g.NewPointerExitEventFromMouseEvent(hover))
	l.describe("enter.fromMouseEvent transformed", g.NewPointerEnterEventFromMouseEvent(hover.Transformed(tm)))
	l.describe("exit.fromMouseEvent move", g.NewPointerExitEventFromMouseEvent(g.NewPointerMoveEvent(g.PointerMoveEventOptions{
		Position: ui.NewOffset(9, 9), Pointer: 3})))
	l.describe("removed with original", g.NewPointerRemovedEvent(g.PointerRemovedEventOptions{
		Original: g.NewPointerRemovedEvent(g.PointerRemovedEventOptions{Position: ui.NewOffset(1, 2)})}))
	l.describe("removed with original transformed", g.NewPointerRemovedEvent(g.PointerRemovedEventOptions{Position: ui.NewOffset(7, 8),
		Original: g.NewPointerRemovedEvent(g.PointerRemovedEventOptions{Position: ui.NewOffset(1, 2)})}).Transformed(tm))

	scroll := g.NewPointerScrollEvent(g.PointerScrollEventOptions{OnRespond: func(allow bool) { log("respond " + peBool(allow)) }})
	scroll.Respond(true)
	scroll.Transformed(tm).(*g.PointerScrollEvent).Respond(false)
	scroll.CopyWith().(*g.PointerScrollEvent).Respond(true)
	scroll.Transformed(tm).CopyWith().(*g.PointerScrollEvent).Respond(false)
	scroll.CopyWith(g.PointerEventCopyWithOptions{OnRespond: func(allow bool) { log("other " + peBool(allow)) }}).(*g.PointerScrollEvent).Respond(true)

	// ---- statics
	for _, mm := range peMatrices() {
		end, delta := ui.NewOffset(12.5, -7.25), ui.NewOffset(3.5, 1.25)
		log("transformPosition " + mm.name + " " + vtOff(g.PointerEventTransformPosition(mm.m, end)))
		hundred := ui.NewOffset(100, 100)
		log("transformDelta " + mm.name + " " + vtOff(g.PointerEventTransformDeltaViaPositions(end, nil, delta, mm.m)) + " " +
			vtOff(g.PointerEventTransformDeltaViaPositions(end, &hundred, delta, mm.m)))
		log("removePerspective " + mm.name + " " + peMat(g.PointerEventRemovePerspectiveTransform(mm.m)))
	}
	log("transformPosition null " + vtOff(g.PointerEventTransformPosition(nil, ui.NewOffset(1, 2))))
	slop := 7.5
	withSlop, empty := &g.DeviceGestureSettings{TouchSlop: &slop}, &g.DeviceGestureSettings{}
	for _, kind := range ui.PointerDeviceKindValues {
		log("slops " + kind.Name() + " " + ds(g.ComputeHitSlop(kind, nil)) + " " + ds(g.ComputePanSlop(kind, nil)) + " " +
			ds(g.ComputeScaleSlop(kind)) + " " + ds(g.ComputeHitSlop(kind, withSlop)) + " " + ds(g.ComputePanSlop(kind, withSlop)) +
			" " + ds(g.ComputeHitSlop(kind, empty)) + " " + ds(g.ComputePanSlop(kind, empty)))
	}
	nullable := func(v *float64) string {
		if v == nil {
			return "null"
		}
		return ds(*v)
	}
	log("settings " + withSlop.String() + " " + empty.String() + " " + nullable(withSlop.PanSlop()) + " " +
		nullable(empty.PanSlop()) + " " + peBool(withSlop.Equal(g.DeviceGestureSettings{TouchSlop: &slop})) + " " +
		peBool(withSlop.Equal(*empty)))
	var mouseB, stylusB, single []string
	for i := 1; i <= 8; i++ {
		mouseB = append(mouseB, strconv.Itoa(g.NthMouseButton(i)))
		stylusB = append(stylusB, strconv.Itoa(g.NthStylusButton(i)))
	}
	for _, b := range []int{0, 1, 2, 3, 6, 0x220, 0x400, -4} {
		single = append(single, strconv.Itoa(g.SmallestButton(b))+"/"+peBool(g.IsSingleButton(b)))
	}
	log("buttons [" + strings.Join(mouseB, ", ") + "] [" + strings.Join(stylusB, ", ") + "] [" + strings.Join(single, ", ") + "]")

	// ---- converter
	var data []ui.PointerData
	for _, change := range ui.PointerChangeValues {
		for _, kind := range ui.PointerDeviceKindValues {
			if kind == ui.PointerDeviceKindTrackpad && slices.Contains([]ui.PointerChange{ui.PointerChangeDown,
				ui.PointerChangeMove, ui.PointerChangeUp, ui.PointerChangeCancel}, change) {
				continue
			}
			viewID := 0
			if kind == ui.PointerDeviceKindUnknown {
				viewID = 99
			}
			ci, ki := int(change), int(kind)
			data = append(data, ui.PointerData{ViewID: viewID, EmbedderID: ci*10 + ki,
				TimeStamp: time.Duration(1000+ci*37+ki) * time.Microsecond, Change: change, Kind: kind, Device: ki,
				PointerIdentifier: ci + 1, PhysicalX: 301.5 + float64(ki), PhysicalY: 207.25 - float64(ci),
				PhysicalDeltaX: 3.75, PhysicalDeltaY: -1.5, Buttons: ci % 3, Obscured: ki%2 == 1, Synthesized: ci%2 == 0,
				Pressure: 0.55, PressureMin: 0.05, PressureMax: 1.95, Distance: 0.7, DistanceMax: 1.4, Size: 0.21,
				RadiusMajor: 7.5, RadiusMinor: 3.25, RadiusMin: 1.5, RadiusMax: 10.5, Orientation: 0.12, Tilt: 0.34,
				PlatformData: 77, ScrollDeltaX: 9.5, ScrollDeltaY: -3.5, PanX: 21.5, PanY: -14.25, PanDeltaX: 1.5,
				PanDeltaY: -2.5, Scale: 1.3, Rotation: 0.4})
		}
	}
	for _, signal := range ui.PointerSignalKindValues {
		s := signal
		data = append(data, ui.PointerData{SignalKind: &s, Change: ui.PointerChangeHover, Kind: ui.PointerDeviceKindMouse,
			PhysicalX: 50, PhysicalY: 70, ScrollDeltaX: 30.5, ScrollDeltaY: -60.25, Scale: 0.8, EmbedderID: 500 + int(signal),
			OnRespond: func(allow bool) { log("datum respond " + s.Name() + " " + peBool(allow)) }})
	}
	scrollKind := ui.PointerSignalKindScroll
	zero := 0.0
	data = append(data,
		ui.PointerData{SignalKind: &scrollKind, ScrollDeltaX: 1 / zero, EmbedderID: 600},
		ui.PointerData{SignalKind: &scrollKind, ScrollDeltaY: zero / zero, EmbedderID: 601},
		ui.PointerData{Change: ui.PointerChangeDown, Kind: ui.PointerDeviceKindMouse, EmbedderID: 602},
		ui.PointerData{Change: ui.PointerChangeDown, Kind: ui.PointerDeviceKindStylus, EmbedderID: 603},
		ui.PointerData{Change: ui.PointerChangeMove, Kind: ui.PointerDeviceKindInvertedStylus, Buttons: 4, EmbedderID: 604},
	)
	for _, dpr := range []float64{1, 2, 3.5, 0} {
		d := dpr
		out := slices.Collect(g.PointerEventConverter.Expand(slices.Values(data), func(viewID int) *float64 {
			if viewID == 99 {
				return nil
			}
			return &d
		}))
		log("expand dpr " + ds(dpr) + ": " + strconv.Itoa(len(out)))
		for _, e := range out {
			log("  " + peFields(e))
			if s, ok := e.(*g.PointerScrollEvent); ok {
				s.Respond(true)
			}
		}
	}
	log("datum " + data[0].String() + " " + data[3].ToStringFull())

	// ---- resampler
	script := func(n int) []g.PointerEvent {
		us := func(n int) time.Duration { return time.Duration(n) * time.Microsecond }
		off := ui.NewOffset
		if n == 0 {
			return []g.PointerEvent{
				g.NewPointerAddedEvent(g.PointerAddedEventOptions{TimeStamp: us(1000), Position: off(0, 50)}),
				g.NewPointerHoverEvent(g.PointerHoverEventOptions{TimeStamp: us(2000), Position: off(10, 40), Delta: off(10, -10)}),
				g.NewPointerDownEvent(g.PointerDownEventOptions{TimeStamp: us(2000), Position: off(10, 40), Pointer: 3}),
				g.NewPointerMoveEvent(g.PointerMoveEventOptions{TimeStamp: us(3333), Position: off(20.5, 30.25), Delta: off(10.5, -9.75), Pointer: 3}),
				g.NewPointerMoveEvent(g.PointerMoveEventOptions{TimeStamp: us(4111), Position: off(31, 19), Delta: off(10.5, -11.25), Pointer: 3, Buttons: pi(3)}),
				g.NewPointerUpEvent(g.PointerUpEventOptions{TimeStamp: us(4999), Position: off(31, 19), Pointer: 3}),
				g.NewPointerHoverEvent(g.PointerHoverEventOptions{TimeStamp: us(6001), Position: off(40, 10), Delta: off(9, -9)}),
				g.NewPointerRemovedEvent(g.PointerRemovedEventOptions{TimeStamp: us(7003), Position: off(40, 10)}),
			}
		}
		if n == 2 {
			return []g.PointerEvent{
				g.NewPointerUpEvent(g.PointerUpEventOptions{TimeStamp: us(1000), Position: off(5, 7), Pointer: 1}),
				g.NewPointerRemovedEvent(g.PointerRemovedEventOptions{TimeStamp: us(1200), Position: off(5, 7)}),
			}
		}
		out := []g.PointerEvent{g.NewPointerDownEvent(g.PointerDownEventOptions{TimeStamp: us(10000), Position: off(100, 100), Pointer: 9})}
		for i := 1; i <= 12; i++ {
			fi := float64(i)
			out = append(out, g.NewPointerMoveEvent(g.PointerMoveEventOptions{TimeStamp: us(10000 + i*4100),
				Position: off(100+float64(fi*3.7), 100-float64(float64(i*i)*0.3)), Delta: off(3.7, -float64(float64(2*i-1)*0.3)), Pointer: 9, Pressure: pf(0.5 + float64(i)/100)}))
		}
		return append(out, g.NewPointerUpEvent(g.PointerUpEventOptions{TimeStamp: us(60000), Position: off(144.4, 56.8), Pointer: 9}))
	}
	for _, n := range []int{0, 1, 2} {
		for _, c := range [][3]int{{500, 1000, 0}, {0, 16667, 16667}, {3000, 7000, 3500}, {9000, 2500, 2500}, {500, 1500, 1500}, {999, 1000, 1000}, {14100, 4100, 4100}} {
			start, step, nextOffset := c[0], c[1], c[2]
			resampler := g.NewPointerEventResampler()
			for _, e := range script(n) {
				resampler.AddEvent(e)
			}
			log("resample " + strconv.Itoa(n) + " " + strconv.Itoa(start) + " " + strconv.Itoa(step) + " " + strconv.Itoa(nextOffset))
			t := start
			for k := 0; k < 40 && (resampler.HasPendingEvents() || k < 3); k++ {
				tt := t
				resampler.Sample(time.Duration(t)*time.Microsecond, time.Duration(t+nextOffset)*time.Microsecond,
					func(e g.PointerEvent) { log("  " + strconv.Itoa(tt) + " " + peFields(e)) })
				log("  state " + strconv.Itoa(t) + " " + peBool(resampler.HasPendingEvents()) + " " + peBool(resampler.IsTracked()) +
					" " + peBool(resampler.IsDown()))
				t += step
			}
			resampler.Stop(func(e g.PointerEvent) { log("  stop " + peFields(e)) })
		}
	}
	stopper := g.NewPointerEventResampler()
	for _, e := range script(1) {
		stopper.AddEvent(e)
	}
	stopper.Sample(20000*time.Microsecond, 0, func(e g.PointerEvent) { log("  pre-stop " + peFields(e)) })
	stopper.Stop(func(e g.PointerEvent) { log("  stop " + peFields(e)) })
	log("  after stop " + peBool(stopper.HasPendingEvents()) + " " + peBool(stopper.IsTracked()) + " " + peBool(stopper.IsDown()))

	// ---- arena
	g.DebugPrintGestureArenaDiagnostics = true
	arenaScript := func(name string, body func(arena *g.GestureArenaManager, m []*peMember, flush func())) {
		loop := async.NewLoop()
		restore := async.SetCurrent(loop)
		log("arena " + name)
		arena := g.NewGestureArenaManager()
		m := []*peMember{{l: l, name: "a"}, {l: l, name: "b"}, {l: l, name: "c"}}
		body(arena, m, loop.FlushMicrotasks)
		loop.FlushMicrotasks()
		log("  end")
		restore()
	}
	addAll := func(arena *g.GestureArenaManager, m []*peMember, pointer int) {
		for _, x := range m {
			x.entry = arena.Add(pointer, x)
		}
	}
	acc, rej := g.GestureDispositionAccepted, g.GestureDispositionRejected
	arenaScript("sweep", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m, 1)
		arena.Close(1)
		arena.Sweep(1)
		arena.Sweep(1)
	})
	arenaScript("accept closed", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m, 2)
		arena.Close(2)
		m[1].entry.Resolve(acc)
		m[0].entry.Resolve(acc)
	})
	arenaScript("eager", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m, 3)
		m[2].entry.Resolve(acc)
		m[0].entry.Resolve(acc)
		arena.Close(3)
	})
	arenaScript("reject to one", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m, 4)
		arena.Close(4)
		m[0].entry.Resolve(rej)
		m[2].entry.Resolve(rej)
		log("  before flush")
		flush()
	})
	arenaScript("reject all", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m, 5)
		arena.Close(5)
		for _, x := range m {
			x.entry.Resolve(rej)
		}
	})
	arenaScript("hold", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m[:2], 6)
		arena.Close(6)
		arena.Hold(6)
		arena.Sweep(6)
		log("  held")
		arena.Release(6)
		arena.Release(6)
	})
	arenaScript("hold accept", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m[:2], 7)
		arena.Hold(7)
		arena.Close(7)
		m[1].entry.Resolve(acc)
		arena.Release(7)
	})
	arenaScript("single default", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		m[0].entry = arena.Add(8, m[0])
		arena.Close(8)
		log("  closed")
		flush()
		m[0].entry.Resolve(acc)
	})
	arenaScript("single then sweep", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		m[0].entry = arena.Add(9, m[0])
		arena.Close(9)
		arena.Sweep(9)
	})
	arenaScript("eager rejected open", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m, 10)
		m[0].entry.Resolve(acc)
		m[0].entry.Resolve(rej)
		m[1].entry.Resolve(acc)
		arena.Close(10)
	})
	arenaScript("two pointers", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		m[0].entry = arena.Add(11, m[0])
		m[1].entry = arena.Add(12, m[1])
		m[2].entry = arena.Add(11, m[2])
		arena.Close(12)
		arena.Close(11)
		arena.Close(13)
		arena.Sweep(13)
		arena.Hold(13)
		arena.Release(13)
		flush()
		arena.Sweep(11)
	})
	arenaScript("accept during reject", func(arena *g.GestureArenaManager, m []*peMember, flush func()) {
		addAll(arena, m, 14)
		arena.Close(14)
		m[1].entry.Resolve(rej)
		m[2].entry.Resolve(acc)
		m[0].entry.Resolve(acc)
	})
	g.DebugPrintGestureArenaDiagnostics = false

	// ---- router
	router := g.NewPointerRouter()
	rm := vm.NewMatrix4Identity()
	rm.ScaleByDouble(0.5, 0.25, 1.0, 1.0)
	rm = rm.Multiplied(vm.NewMatrix4TranslationValues(-10, -30, 0))
	var r1, r2 g.PointerRoute
	r1 = func(e g.PointerEvent) { log("  r1 " + peFields(e)) }
	r2 = func(e g.PointerEvent) {
		log("  r2 " + peFields(e))
		router.RemoveRoute(2, r1)
		router.AddRoute(2, r1)
	}
	g1 := func(e g.PointerEvent) { log("  g1 " + peFields(e)) }
	bad := func(g.PointerEvent) { panic(dart.StateError{Message: "route failed"}) }
	router.AddRoute(2, r1, rm)
	router.AddRoute(2, r2)
	router.AddGlobalRoute(g1, vm.NewMatrix4RotationZ(0.3))
	router.AddRoute(3, bad)
	saved := foundation.FlutterErrorOnError
	onError := func(d *foundation.FlutterErrorDetails) {
		var info []string
		for n := range d.InformationCollector() {
			info = append(info, peMask(n.String()))
		}
		log("  error " + dart.ToString(d.Exception) + " | " + *d.Library + " | " + d.Context.String() + " | " + strings.Join(info, " ; "))
	}
	foundation.FlutterErrorOnError = onError
	log("route 2")
	router.Route(g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: 2, Position: ui.NewOffset(16, 36)}))
	log("route 2 again")
	router.Route(g.NewPointerMoveEvent(g.PointerMoveEventOptions{Pointer: 2, Position: ui.NewOffset(20, 40), Delta: ui.NewOffset(4, 4)}))
	log("route 3")
	router.Route(g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: 3, Position: ui.NewOffset(1, 1)}))
	router.RemoveRoute(2, r1)
	router.RemoveRoute(2, r2)
	router.RemoveGlobalRoute(g1)
	log("route 2 removed")
	router.Route(g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: 2}))
	log("global count " + strconv.Itoa(router.DebugGlobalRouteCount()))
	foundation.FlutterErrorOnError = saved

	// ---- signal resolver
	resolver := g.NewPointerSignalResolver()
	original := g.NewPointerScrollEvent(g.PointerScrollEventOptions{ScrollDelta: ui.NewOffset(0, 10),
		OnRespond: func(allow bool) { log("  scroll respond " + peBool(allow)) }})
	resolver.Resolve(original)
	t1 := original.Transformed(vm.NewMatrix4TranslationValues(5, 5, 0)).(g.PointerSignalEvent)
	t2 := original.Transformed(vm.NewMatrix4TranslationValues(7, 7, 0)).(g.PointerSignalEvent)
	resolver.Register(t1, func(e g.PointerSignalEvent) { log("  first " + peFields(e)) })
	resolver.Register(t2, func(e g.PointerSignalEvent) { log("  second " + peFields(e)) })
	resolver.Resolve(original)
	resolver.Register(original, func(g.PointerSignalEvent) { panic(dart.StateError{Message: "resolve failed"}) })
	foundation.FlutterErrorOnError = onError
	resolver.Resolve(original)
	foundation.FlutterErrorOnError = saved
	resolver.Resolve(original)

	// ---- hit test
	result := g.NewHitTestResult()
	result.Add(g.NewHitTestEntry(&HtTarget{"root"}))
	result.PushOffset(ui.NewOffset(10, 20))
	result.Add(g.NewHitTestEntry(&HtTarget{"offset"}))
	rot := vm.NewMatrix4RotationZ(0.25)
	rot.SetTranslationRaw(1, 2, 0)
	result.PushTransform(rot)
	wrapped := g.NewHitTestResultWrap(result)
	wrapped.Add(g.NewHitTestEntry(&HtTarget{"rotated"}))
	result.PushOffset(ui.NewOffset(-3.5, 4.25))
	result.PopTransform()
	result.PushTransform(vm.NewMatrix4Diagonal3Values(2, 3, 1))
	result.Add(g.NewHitTestEntry(&HtTarget{"scaled"}))
	result.PopTransform()
	result.PopTransform()
	result.Add(g.NewHitTestEntry(&HtTarget{"back"}))
	for e := range wrapped.Path() {
		log("hit " + peMask(e.String()) + " " + peMat(e.Transform()))
	}
	log("hit result " + peMask(result.String()))
	log("hit empty " + g.NewHitTestResult().String())

	// ---- details, PointerData, KeyData
	local := ui.NewOffset(3, 4)
	ts12, ts3 := 12*time.Millisecond, 3*time.Microsecond
	stylus, touch := ui.PointerDeviceKindStylus, ui.PointerDeviceKindTouch
	for _, d := range []foundation.Diagnosticable{
		g.NewDragDownDetails(g.DragDownDetailsOptions{GlobalPosition: ui.NewOffset(1, 2)}),
		g.NewDragDownDetails(g.DragDownDetailsOptions{GlobalPosition: ui.NewOffset(1, 2), LocalPosition: &local}),
		g.NewDragStartDetails(g.DragStartDetailsOptions{GlobalPosition: ui.NewOffset(5, 6), SourceTimeStamp: &ts12, Kind: &stylus}),
		g.NewDragUpdateDetails(g.DragUpdateDetailsOptions{GlobalPosition: ui.NewOffset(7, 8), Delta: ui.NewOffset(0, 2.5),
			PrimaryDelta: pf(2.5), SourceTimeStamp: &ts3, Kind: &touch}),
		g.NewDragEndDetails(g.DragEndDetailsOptions{Velocity: g.Velocity{PixelsPerSecond: ui.NewOffset(100.25, 0)}, PrimaryVelocity: pf(100.25)}),
		g.NewDragEndDetails(),
	} {
		log("details " + peMask(d.String()) + " | " + peMask(d.ToDiagnosticsNode().ToStringDeep()))
	}
	scaleKind := ui.PointerSignalKindScale
	log("pointerdata " + ui.PointerData{ViewID: 4, PhysicalX: 1.5, PhysicalY: 2.25}.String() + " " +
		ui.PointerData{SignalKind: &scaleKind, TimeStamp: 61 * time.Second}.ToStringFull())
	str := func(s string) *string { return &s }
	for _, k := range []ui.KeyData{
		{TimeStamp: 1500 * time.Millisecond, Type: ui.KeyEventTypeDown, Physical: 0x70004, Logical: 0x61, Character: str("a")},
		{Type: ui.KeyEventTypeRepeat, Physical: 0x100070028, Logical: 0x200000100, Character: str("\n"), Synthesized: true,
			DeviceType: ui.KeyEventDeviceTypeJoystick},
		{TimeStamp: 7 * time.Microsecond, Type: ui.KeyEventTypeUp, Logical: 0x1100000042, DeviceType: ui.KeyEventDeviceTypeDirectionalPad},
		{TimeStamp: 8 * time.Microsecond, Type: ui.KeyEventTypeDown, Physical: 1, Logical: 0x1800000001, Character: str("\té"),
			DeviceType: ui.KeyEventDeviceTypeHdmi},
		{TimeStamp: 9 * time.Microsecond, Type: ui.KeyEventTypeDown, Physical: 1, Logical: 0x3300000001, Character: str("\U0001F600"),
			DeviceType: ui.KeyEventDeviceTypeGamepad},
	} {
		log("key " + k.String() + " | " + k.ToStringFull())
	}
	return l.lines
}

func TestPointerEventsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/pointer_events.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	compareLines(t, ref.Log, runPointerEventsScene())
}
