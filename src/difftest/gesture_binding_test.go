package difftest

// Go twin of flutter/test/gesture_binding_test.dart: GestureBinding's
// packet handling, hit testing, dispatch, arena, cancel, locking, errors,
// signals, onHitTest and resampling, and the base recognizers (OneSequence,
// PrimaryPointer, Eager, teams, invokeCallback), compared line by line.

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	g "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var gbLines []string

func gblog(s string) { gbLines = append(gbLines, s) }

var gbHash = regexp.MustCompile(`#[0-9a-f]{5}`)

func gbMask(s string) string { return gbHash.ReplaceAllString(s, "#HASH") }

func gbO(v ui.Offset) string { return dart.DoubleToString(v.Dx) + "," + dart.DoubleToString(v.Dy) }

func gbEv(e g.PointerEvent) string {
	return strings.SplitN(e.ToStringShort(), "#", 2)[0] + " p" + strconv.Itoa(e.Pointer()) +
		" pos=" + gbO(e.Position()) + " local=" + gbO(e.LocalPosition()) +
		" delta=" + gbO(e.Delta()) + " localDelta=" + gbO(e.LocalDelta()) +
		" t=" + strconv.FormatInt(e.TimeStamp().Microseconds(), 10) + " buttons=" + strconv.Itoa(e.Buttons()) +
		" kind=" + e.Kind().Name() + " down=" + strconv.FormatBool(e.Down()) + " device=" + strconv.Itoa(e.Device())
}

type gbTarget struct {
	name        string
	throwOnDown bool
}

func (t *gbTarget) HandleEvent(event g.PointerEvent, entry g.HitTestEntry) {
	gblog(t.name + ".handleEvent " + gbEv(event))
	if _, ok := event.(*g.PointerDownEvent); ok && t.throwOnDown {
		panic(dart.StateError{Message: t.name + " failed"})
	}
	if s, ok := event.(g.PointerSignalEvent); ok {
		g.GestureBindingInstance().PointerSignalResolver().Register(s, func(e g.PointerSignalEvent) {
			gblog(t.name + " resolved signal " + gbEv(e))
		})
	}
}

func (t *gbTarget) String() string { return "Target(" + t.name + ")" }

type gbBinding struct {
	foundation.BindingBaseBase
	g.GestureBindingMixin
	scheduler.SchedulerBindingMixin

	hitTester func(result *g.HitTestResult, position ui.Offset, viewID int)
	loop      *async.Loop
}

func (b *gbBinding) InitInstances()         { b.SchedulerBindingMixin.InitInstances() }
func (b *gbBinding) InitServiceExtensions() { b.SchedulerBindingMixin.InitServiceExtensions() }
func (b *gbBinding) Unlocked()              { b.SchedulerBindingMixin.Unlocked() }

func (b *gbBinding) HitTestInView(result *g.HitTestResult, position ui.Offset, viewID int) {
	gblog("hitTestInView " + gbO(position) + " view=" + strconv.Itoa(viewID) + " dpr=" +
		dart.DoubleToString(b.PlatformDispatcher().ImplicitView().DevicePixelRatio()))
	if b.hitTester != nil {
		b.hitTester(result, position, viewID)
	}
	b.GestureBindingMixin.HitTestInView(result, position, viewID)
}

func (b *gbBinding) HandleEvent(event g.PointerEvent, entry g.HitTestEntry) {
	gblog("binding.handleEvent " + gbEv(event))
	b.GestureBindingMixin.HandleEvent(event, entry)
}

func (b *gbBinding) DebugSamplingClock() g.SamplingClock { return gbClock{b.loop} }

type gbClock struct{ loop *async.Loop }

func (c gbClock) Now() time.Time {
	return time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC).Add(c.loop.Elapsed())
}
func (c gbClock) Stopwatch() *dart.Stopwatch { return dart.NewStopwatchWithClock(c.loop.Elapsed) }

type gbPD struct {
	x, y, scrollX, scrollY      float64
	pointer, device, buttons, ms int
	kind                        ui.PointerDeviceKind
	signal                      *ui.PointerSignalKind
}

func gbData(change ui.PointerChange, o gbPD) ui.PointerData {
	return ui.PointerData{Change: change, PhysicalX: o.x, PhysicalY: o.y, PointerIdentifier: o.pointer,
		Device: o.device, Buttons: o.buttons, TimeStamp: time.Duration(o.ms) * time.Millisecond, Kind: o.kind,
		SignalKind: o.signal, ScrollDeltaX: o.scrollX, ScrollDeltaY: o.scrollY}
}

func gbSend(data ...ui.PointerData) {
	var names []string
	for _, d := range data {
		names = append(names, d.Change.Name())
	}
	gblog("packet " + strings.Join(names, ","))
	g.GestureBindingInstance().PlatformDispatcher().OnPointerDataPacket()(ui.PointerDataPacket{Data: data})
}

type gbPassive struct {
	g.OneSequenceGestureRecognizerBase
	name string
}

func newGbPassive(name string) *gbPassive {
	r := &gbPassive{name: name}
	r.Construct(r, g.GestureRecognizerOptions{})
	return r
}

func (r *gbPassive) AddPointer(event *g.PointerDownEvent) { r.StartTrackingPointer(event.Pointer()) }
func (r *gbPassive) DebugDescription() string             { return "passive" }
func (r *gbPassive) DidStopTrackingLastPointer(pointer int) {
	gblog(r.name + ".didStopTrackingLastPointer " + strconv.Itoa(pointer))
	r.Resolve(g.GestureDispositionRejected)
}
func (r *gbPassive) HandleEvent(event g.PointerEvent) {
	gblog(r.name + ".handleEvent " + gbEv(event))
	switch event.(type) {
	case *g.PointerUpEvent, *g.PointerCancelEvent:
		r.StopTrackingPointer(event.Pointer())
	}
}
func (r *gbPassive) AcceptGesture(pointer int) { gblog(r.name + ".accept " + strconv.Itoa(pointer)) }
func (r *gbPassive) RejectGesture(pointer int) { gblog(r.name + ".reject " + strconv.Itoa(pointer)) }

// Primary has the Dart class's name, as its toString shows it.
type Primary struct {
	g.PrimaryPointerGestureRecognizerBase
	name        string
	resolveOnUp *g.GestureDisposition
}

func newGbPrimary(name string, opts g.PrimaryPointerGestureRecognizerOptions) *Primary {
	r := &Primary{name: name}
	r.Construct(r, opts)
	return r
}

func (r *Primary) state() string { return r.State().Name() }

func (r *Primary) HandlePrimaryPointer(event g.PointerEvent) {
	gblog(r.name + ".handlePrimaryPointer " + gbEv(event) + " state=" + r.state())
	if _, ok := event.(*g.PointerUpEvent); ok && r.resolveOnUp != nil {
		r.Resolve(*r.resolveOnUp)
	}
}
func (r *Primary) DidExceedDeadline() {
	gblog(r.name + ".didExceedDeadline state=" + r.state())
	r.Resolve(g.GestureDispositionAccepted)
}
func (r *Primary) HandleNonAllowedPointer(event *g.PointerDownEvent) {
	gblog(r.name + ".handleNonAllowedPointer " + gbEv(event))
	r.PrimaryPointerGestureRecognizerBase.HandleNonAllowedPointer(event)
}
func (r *Primary) AcceptGesture(pointer int) {
	r.PrimaryPointerGestureRecognizerBase.AcceptGesture(pointer)
	gblog(r.name + ".accept " + strconv.Itoa(pointer) + " state=" + r.state())
}
func (r *Primary) RejectGesture(pointer int) {
	r.PrimaryPointerGestureRecognizerBase.RejectGesture(pointer)
	gblog(r.name + ".reject " + strconv.Itoa(pointer) + " state=" + r.state())
}
func (r *Primary) DidStopTrackingLastPointer(pointer int) {
	gblog(r.name + ".didStopTrackingLastPointer " + strconv.Itoa(pointer) + " state=" + r.state())
	r.PrimaryPointerGestureRecognizerBase.DidStopTrackingLastPointer(pointer)
}
func (r *Primary) DebugDescription() string { return "primary" }

type gbAdder struct{ recognizers []g.GestureRecognizer }

func (a *gbAdder) HandleEvent(event g.PointerEvent, entry g.HitTestEntry) {
	if down, ok := event.(*g.PointerDownEvent); ok {
		for _, r := range a.recognizers {
			r.AddPointer(down)
		}
	}
}

func TestGestureBindingMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/gesture_binding.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	if !isolated(t) { // it constructs its own binding
		return
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	// flutter_test's test view: device pixel ratio 3.
	ui.NewPlatformConfiguration(nil).UpdateViewGestureMetrics(0, 3, ui.GestureSettings{})

	binding := &gbBinding{loop: loop}
	binding.GestureBindingMixin.SetSelf(binding)
	binding.SchedulerBindingMixin.SetSelf(binding)
	binding.Construct(binding, &binding.GestureBindingMixin, &binding.SchedulerBindingMixin)

	a, b, c := &gbTarget{name: "A"}, &gbTarget{name: "B"}, &gbTarget{name: "C"}
	binding.hitTester = func(result *g.HitTestResult, position ui.Offset, viewID int) {
		result.Add(g.NewHitTestEntry(a))
		result.PushOffset(ui.NewOffset(-10, -20))
		result.Add(g.NewHitTestEntry(b))
		result.PushTransform(vm.NewMatrix4Diagonal3Values(2, 3, 1))
		result.Add(g.NewHitTestEntry(c))
		result.PopTransform()
		result.PopTransform()
	}
	var errors []*foundation.FlutterErrorDetails
	previousOnError := foundation.FlutterErrorOnError
	defer func() { foundation.FlutterErrorOnError = previousOnError }()
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { errors = append(errors, d) }
	lib := func(d *foundation.FlutterErrorDetails) string {
		if d.Library == nil {
			return "null"
		}
		return *d.Library
	}
	flushErrors := func() {
		for _, d := range errors {
			pd := g.AsFlutterErrorDetailsForPointerEventDispatcher(d)
			extra := ""
			if pd != nil {
				extra = "event=" + gbMask(dart.ToString(pd.Event)) + " entry=" + gbMask(dart.ToString(pd.HitTestEntry))
			}
			gblog("error " + dart.ToString(d.Exception) + " library=" + lib(d) + " context=" + dart.ToString(d.Context) +
				" dispatcher=" + strconv.FormatBool(pd != nil) + " " + extra)
		}
		errors = nil
	}
	var routed []g.PointerEvent
	globalRoute := func(e g.PointerEvent) { routed = append(routed, e) }
	g.GestureBindingInstance().PointerRouter().AddGlobalRoute(globalRoute)
	flushRoute := func() {
		for _, e := range routed {
			gblog("global route " + gbEv(e))
		}
		routed = nil
	}
	mouse := ui.PointerDeviceKindMouse

	// 1.
	gbSend(gbData(ui.PointerChangeAdd, gbPD{x: 5, y: 6, pointer: 1}),
		gbData(ui.PointerChangeDown, gbPD{x: 5, y: 6, pointer: 1, ms: 1}),
		gbData(ui.PointerChangeMove, gbPD{x: 7, y: 9, pointer: 1, ms: 2}),
		gbData(ui.PointerChangeUp, gbPD{x: 7, y: 9, pointer: 1, ms: 3}))
	flushRoute()
	flushErrors()

	// 2.
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 1, y: 1, pointer: 2, device: 1}),
		gbData(ui.PointerChangeDown, gbPD{x: 50, y: 60, pointer: 3, device: 2}),
		gbData(ui.PointerChangeMove, gbPD{x: 2, y: 3, pointer: 2, device: 1}),
		gbData(ui.PointerChangeUp, gbPD{x: 50, y: 60, pointer: 3, device: 2}),
		gbData(ui.PointerChangeCancel, gbPD{x: 2, y: 3, pointer: 2, device: 1}),
		gbData(ui.PointerChangeHover, gbPD{x: 8, y: 8, device: 5, kind: mouse}),
		gbData(ui.PointerChangeRemove, gbPD{x: 8, y: 8, device: 5, kind: mouse}))
	flushRoute()

	// 3.
	b.throwOnDown = true
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 3, y: 4, pointer: 4}), gbData(ui.PointerChangeUp, gbPD{x: 3, y: 4, pointer: 4}))
	b.throwOnDown = false
	flushErrors()
	flushRoute()

	// 4.
	bad := func(g.PointerEvent) { panic(dart.StateError{Message: "route failed"}) }
	g.GestureBindingInstance().PointerRouter().AddGlobalRoute(bad)
	gbSend(gbData(ui.PointerChangeAdd, gbPD{x: 1, y: 2, device: 9}))
	g.GestureBindingInstance().PointerRouter().RemoveGlobalRoute(bad)
	flushErrors()
	flushRoute()

	// 5.
	binding.LockEvents(func() async.Future[async.Void] {
		return async.AsyncVoid(func() {
			gbSend(gbData(ui.PointerChangeDown, gbPD{x: 9, y: 9, pointer: 5}), gbData(ui.PointerChangeUp, gbPD{x: 9, y: 9, pointer: 5}))
			gblog("sent while locked")
			async.FutureDelayed[async.Void](5*time.Millisecond, nil).Await()
			gblog("unlocking")
		})
	})
	loop.Elapse(10 * time.Millisecond)
	flushRoute()

	// 6.
	canceler := func(e g.PointerEvent) {
		if _, ok := e.(*g.PointerDownEvent); ok {
			gblog("cancelPointer " + strconv.Itoa(e.Pointer()))
			binding.CancelPointer(e.Pointer())
		}
	}
	g.GestureBindingInstance().PointerRouter().AddGlobalRoute(canceler)
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 4, y: 4, pointer: 6}))
	gblog("after packet")
	loop.FlushMicrotasks()
	g.GestureBindingInstance().PointerRouter().RemoveGlobalRoute(canceler)
	flushRoute()

	// 7.
	scroll := ui.PointerSignalKindScroll
	gbSend(gbData(ui.PointerChangeAdd, gbPD{x: 10, y: 10, device: 7, kind: mouse}),
		gbData(ui.PointerChangeHover, gbPD{x: 10, y: 10, device: 7, kind: mouse, signal: &scroll, scrollX: 3, scrollY: -4}))
	flushRoute()

	// 8.
	response := binding.PlatformDispatcher().OnHitTest()(ui.HitTestRequest{View: binding.PlatformDispatcher().ImplicitView(), Offset: ui.NewOffset(3, 3)})
	gblog("onHitTest hasPlatformView=" + strconv.FormatBool(response.HasPlatformView))
	flushRoute()

	// 9.
	binding.hitTester = nil
	team := g.NewGestureArenaTeam()
	p1 := newGbPassive("P1")
	p1.SetTeam(team)
	eager := g.NewEagerGestureRecognizer()
	deadline := 100 * time.Millisecond
	prim := newGbPrimary("Prim", g.PrimaryPointerGestureRecognizerOptions{Deadline: &deadline})
	downWith := func(rs []g.GestureRecognizer, pointer int, x float64) {
		down := g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: pointer, Position: ui.NewOffset(x, x)})
		for _, r := range rs {
			r.AddPointer(down)
		}
		binding.PointerRouter().Route(down)
		binding.GestureArena().Close(pointer)
		loop.FlushMicrotasks()
	}
	up := func(pointer int, x float64) {
		binding.PointerRouter().Route(g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: pointer, Position: ui.NewOffset(x, x)}))
		binding.GestureArena().Sweep(pointer)
	}
	downWith([]g.GestureRecognizer{p1, prim}, 20, 5)
	gblog("elapse 50")
	loop.Elapse(50 * time.Millisecond)
	gblog("elapse 50")
	loop.Elapse(50 * time.Millisecond)
	gblog("elapse 10")
	loop.Elapse(10 * time.Millisecond)
	up(20, 5)
	gblog("prim state " + prim.state() + " " + gbMask(prim.String()))

	captainTeam := g.NewGestureArenaTeam()
	p2, p3 := newGbPassive("P2"), newGbPassive("P3")
	p2.SetTeam(captainTeam)
	p3.SetTeam(captainTeam)
	captainTeam.Captain = p3
	downWith([]g.GestureRecognizer{p2, p3, eager}, 21, 1)
	up(21, 1)

	slopTolerance := 18.0
	slop := newGbPrimary("Slop", g.PrimaryPointerGestureRecognizerOptions{PreAcceptSlopTolerance: &slopTolerance})
	accepted := g.GestureDispositionAccepted
	slop.resolveOnUp = &accepted
	other := newGbPassive("P4")
	downWith([]g.GestureRecognizer{slop, other}, 22, 0)
	binding.PointerRouter().Route(g.NewPointerMoveEvent(g.PointerMoveEventOptions{Pointer: 22, Position: ui.NewOffset(10, 10), Delta: ui.NewOffset(10, 10)}))
	binding.PointerRouter().Route(g.NewPointerMoveEvent(g.PointerMoveEventOptions{Pointer: 22, Position: ui.NewOffset(13, 13), Delta: ui.NewOffset(3, 3)}))
	up(22, 13)

	filtered := newGbPrimary("Filtered", g.PrimaryPointerGestureRecognizerOptions{GestureRecognizerOptions: g.GestureRecognizerOptions{
		SupportedDevices:     map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindMouse: {}},
		AllowedButtonsFilter: func(buttons int) bool { return buttons == g.KSecondaryButton },
	}})
	filtered.AddPointer(g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: 23}))
	secondary := g.KSecondaryButton
	filtered.AddPointer(g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: 24, Kind: mouse, Buttons: &secondary}))
	gblog("filtered state " + filtered.state() + " primary=" + strconv.Itoa(*filtered.PrimaryPointer()))
	gblog("filtered kind " + filtered.GetKindForPointer(23).Name() + " " + strconv.Itoa(filtered.GetButtonsForPointer(24)))
	binding.GestureArena().Close(23)
	binding.GestureArena().Close(24)
	loop.FlushMicrotasks()

	mouseDown := func(pointer int, buttons *int) *g.PointerDownEvent {
		return g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: pointer, Kind: mouse, Buttons: buttons})
	}
	touchDown := func(pointer int, buttons *int) *g.PointerDownEvent {
		return g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: pointer, Buttons: buttons})
	}
	filtered.AddPointer(mouseDown(25, nil))
	filtered.AddPointer(touchDown(26, &secondary))
	mouseOnly := map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindMouse: {}}
	f2 := newGbPrimary("F2", g.PrimaryPointerGestureRecognizerOptions{GestureRecognizerOptions: g.GestureRecognizerOptions{
		SupportedDevices: mouseOnly, AllowedButtonsFilter: func(buttons int) bool { return buttons == g.KSecondaryButton }}})
	f2rival := newGbPassive("F2rival")
	f2down := mouseDown(27, &secondary)
	f2.AddPointer(f2down)
	f2rival.AddPointer(f2down)
	f2.AddPointer(touchDown(28, nil))
	gblog("f2 state " + f2.state())
	f3 := newGbPrimary("F3", g.PrimaryPointerGestureRecognizerOptions{GestureRecognizerOptions: g.GestureRecognizerOptions{SupportedDevices: mouseOnly}})
	f3.AddPointer(mouseDown(29, nil))
	binding.GestureArena().Close(29)
	loop.FlushMicrotasks()
	f3.AddPointer(mouseDown(44, nil))
	f3rival := newGbPassive("F3rival")
	f3rival.AddPointer(touchDown(44, nil))
	f3.AddPointer(touchDown(32, nil))
	gblog("f3 state " + f3.state())
	binding.GestureArena().Close(44)
	binding.GestureArena().Sweep(44)
	f3rival.Dispose()

	canceled := newGbPrimary("Canceled", g.PrimaryPointerGestureRecognizerOptions{})
	canceled.AddPointer(touchDown(33, nil))
	binding.PointerRouter().Route(g.NewPointerCancelEvent(g.PointerCancelEventOptions{Pointer: 33}))
	gblog("canceled state " + canceled.state())
	binding.GestureArena().Close(33)
	loop.FlushMicrotasks()

	far := newGbPrimary("Far", g.PrimaryPointerGestureRecognizerOptions{PreAcceptSlopTolerance: &slopTolerance})
	far.resolveOnUp = &accepted
	farRival := newGbPassive("FarRival")
	downWith([]g.GestureRecognizer{far, farRival}, 34, 0)
	binding.PointerRouter().Route(g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: 34, Position: ui.NewOffset(30, 30)}))
	binding.GestureArena().Sweep(34)

	early := newGbPrimary("Early", g.PrimaryPointerGestureRecognizerOptions{Deadline: &deadline})
	downWith([]g.GestureRecognizer{early}, 35, 0)
	loop.Elapse(150 * time.Millisecond)
	binding.PointerRouter().Route(g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: 35}))

	capR := newGbPassive("Cap")
	tp := newGbPrimary("TP", g.PrimaryPointerGestureRecognizerOptions{})
	tp.resolveOnUp = &accepted
	team3 := g.NewGestureArenaTeam()
	team3.Captain = capR
	capR.SetTeam(team3)
	tp.SetTeam(team3)
	out := newGbPassive("Out")
	downWith([]g.GestureRecognizer{out, tp, capR}, 36, 0)
	binding.PointerRouter().Route(g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: 36}))
	binding.GestureArena().Sweep(36)
	cap2, p7 := newGbPassive("Cap2"), newGbPassive("P7")
	team4 := g.NewGestureArenaTeam()
	team4.Captain = cap2
	cap2.SetTeam(team4)
	p7.SetTeam(team4)
	out2 := newGbPassive("Out2")
	down37 := touchDown(37, nil)
	p7.AddPointer(down37)
	cap2.AddPointer(down37)
	out2.AddPointer(down37)
	binding.GestureArena().Close(37)
	binding.GestureArena().Sweep(37)
	binding.PointerRouter().Route(g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: 37}))

	multi, multiRival := newGbPassive("Multi"), newGbPassive("MultiRival")
	for _, p := range []int{38, 39} {
		d := touchDown(p, nil)
		multi.AddPointer(d)
		multiRival.AddPointer(d)
		binding.GestureArena().Close(p)
	}
	multi.ResolvePointer(38, g.GestureDispositionRejected)
	multi.Resolve(g.GestureDispositionAccepted)
	loop.FlushMicrotasks()
	for _, p := range []int{38, 39} {
		binding.PointerRouter().Route(g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: p}))
		binding.GestureArena().Sweep(p)
	}

	g.DebugPrintRecognizerCallbacksTrace = true
	var printed []string
	oldPrint := foundation.DebugPrint
	foundation.DebugPrint = func(m string, _ ...foundation.DebugPrintOptions) { printed = append(printed, m) }
	r1 := g.InvokeCallback(prim, "onThing", func() *int { v := 7; return &v }, func() string { return "report" })
	r2 := g.InvokeCallback(prim, "onBad", func() *int { panic(dart.StateError{Message: "bad callback"}) })
	g.InvokeVoidCallback(prim, "onEmpty", func() {}, func() string { return "" })
	g.DebugPrintRecognizerCallbacksTrace = false
	foundation.DebugPrint = oldPrint
	for _, p := range printed {
		gblog("print " + gbMask(p))
	}
	nullable := func(p *int) string {
		if p == nil {
			return "null"
		}
		return strconv.Itoa(*p)
	}
	gblog("invokeCallback results " + nullable(r1) + " " + nullable(r2))
	for _, d := range errors {
		gblog("error " + dart.ToString(d.Exception) + " library=" + lib(d) + " context=" + dart.ToString(d.Context))
		var info []string
		for n := range d.InformationCollector() {
			info = append(info, gbMask(n.String()))
		}
		gblog("  info " + strings.Join(info, " | "))
	}
	errors = nil

	// 10.
	binding.SetResamplingEnabled(true)
	binding.SetSamplingOffset(-5 * time.Millisecond)
	binding.hitTester = func(result *g.HitTestResult, position ui.Offset, viewID int) { result.Add(g.NewHitTestEntry(a)) }
	nowMs := int(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()) + int(loop.Elapsed().Milliseconds())
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 0, pointer: 30, ms: nowMs + 1}),
		gbData(ui.PointerChangeMove, gbPD{x: 10, pointer: 30, ms: nowMs + 11}),
		gbData(ui.PointerChangeMove, gbPD{x: 20, pointer: 30, ms: nowMs + 21}),
		gbData(ui.PointerChangeUp, gbPD{x: 20, pointer: 30, ms: nowMs + 31}))
	for range 6 {
		gblog("elapse 10 (frame " + strconv.FormatBool(binding.HasScheduledFrame()) + ")")
		loop.Elapse(10 * time.Millisecond)
	}
	binding.SetResamplingEnabled(false)
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 1, pointer: 31, ms: 1}))
	gbSend(gbData(ui.PointerChangeUp, gbPD{x: 1, pointer: 31, ms: 2}))
	loop.Elapse(50 * time.Millisecond)
	flushRoute()
	flushErrors()
	// 11.
	q1, q2 := newGbPassive("Q1"), newGbPrimary("Q2", g.PrimaryPointerGestureRecognizerOptions{})
	binding.hitTester = func(result *g.HitTestResult, position ui.Offset, viewID int) {
		result.Add(g.NewHitTestEntry(&gbAdder{[]g.GestureRecognizer{q1, q2}}))
	}
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 6, y: 6, pointer: 40}))
	gblog("after down")
	loop.FlushMicrotasks()
	gbSend(gbData(ui.PointerChangeUp, gbPD{x: 6, y: 6, pointer: 40}))
	gblog("after up")
	loop.FlushMicrotasks()
	solo := newGbPrimary("Solo", g.PrimaryPointerGestureRecognizerOptions{})
	binding.hitTester = func(result *g.HitTestResult, position ui.Offset, viewID int) {
		result.Add(g.NewHitTestEntry(&gbAdder{[]g.GestureRecognizer{solo}}))
	}
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 6, y: 6, pointer: 41}))
	gblog("solo before microtasks")
	loop.FlushMicrotasks()
	gblog("solo after microtasks")
	gbSend(gbData(ui.PointerChangeUp, gbPD{x: 6, y: 6, pointer: 41}))

	// 12.
	binding.hitTester = nil
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 2, y: 2, pointer: 42}))
	binding.CancelPointer(42)
	gblog("cancel requested")
	loop.FlushMicrotasks()
	gblog("cancel flushed")

	// 13.
	binding.SetResamplingEnabled(true)
	now2 := int(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()) + int(loop.Elapsed().Milliseconds())
	gbSend(gbData(ui.PointerChangeDown, gbPD{x: 0, pointer: 43, ms: now2 + 1}),
		gbData(ui.PointerChangeMove, gbPD{x: 6, pointer: 43, ms: now2 + 2}))
	elapse := func(n int) {
		for range n {
			gblog("elapse 10")
			loop.Elapse(10 * time.Millisecond)
		}
	}
	elapse(5)
	gbSend(gbData(ui.PointerChangeMove, gbPD{x: 30, pointer: 43, ms: now2 + 55}))
	elapse(4)
	gbSend(gbData(ui.PointerChangeUp, gbPD{x: 30, pointer: 43, ms: now2 + 95}))
	elapse(4)
	binding.SetResamplingEnabled(false)
	flushRoute()

	// 14.
	tracker := g.NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch)
	tracker.AddPosition(0, ui.Offset{})
	tracker.AddPosition(10*time.Millisecond, ui.NewOffset(0, 10))
	tracker.AddPosition(20*time.Millisecond, ui.NewOffset(0, 25))
	gblog("velocity now " + tracker.GetVelocityEstimate().String())
	loop.Elapse(41 * time.Millisecond)
	gblog("velocity after 41ms " + tracker.GetVelocityEstimate().String())

	gblog("timers " + strconv.Itoa(len(loop.PendingTimers())))

	for i := 0; i < max(len(gbLines), len(ref.Log)); i++ {
		var w, gl string
		if i < len(ref.Log) {
			w = ref.Log[i]
		}
		if i < len(gbLines) {
			gl = gbLines[i]
		}
		if w != gl {
			t.Fatalf("line %d:\nflutter: %s\ngo:      %s", i, w, gl)
		}
	}
	if len(ref.Log) < 150 {
		t.Fatalf("reference too short: %d lines", len(ref.Log))
	}
}
