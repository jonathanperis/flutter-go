package difftest

// Go twin of flutter/test/recognizers_tap_test.dart: the same scripted
// pointer sequences on the tap, double tap, multi tap, serial tap, long
// press and force press recognizers (with eager recognizers and teams),
// routed as gesture_tester.dart routes them, with the recognizer callback
// trace and arena diagnostics printed; compared line by line.

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	g "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var (
	rtLines []string
	rtLoop  *async.Loop
	rtHash  = regexp.MustCompile(`#[0-9a-f]{5}`)
)

func rtlog(s string) { rtLines = append(rtLines, s) }

func rtmask(o any) string {
	var s string
	if str, ok := o.(string); ok {
		s = str
	} else {
		s = dart.ToString(o)
	}
	return rtHash.ReplaceAllString(s, "#HASH")
}

func rtnow() string { return fmt.Sprint(rtLoop.Elapsed().Microseconds()) }

func rtroute(e g.PointerEvent) {
	g.GestureBindingInstance().PointerRouter().Route(e)
	rtLoop.FlushMicrotasks()
}

func rtdown(rs []g.GestureRecognizer, e *g.PointerDownEvent) {
	rtlog(fmt.Sprintf("[%s] down %d %s buttons=%d kind=%s", rtnow(), e.Pointer(), e.Position(), e.Buttons(), e.Kind().Name()))
	for _, r := range rs {
		r.AddPointer(e)
	}
	rtroute(e)
	g.GestureBindingInstance().GestureArena().Close(e.Pointer())
	rtLoop.FlushMicrotasks()
}

func rtmove(e *g.PointerMoveEvent) {
	rtlog(fmt.Sprintf("[%s] move %d %s buttons=%d", rtnow(), e.Pointer(), e.Position(), e.Buttons()))
	rtroute(e)
}

func rtup(e *g.PointerUpEvent) {
	rtlog(fmt.Sprintf("[%s] up %d %s", rtnow(), e.Pointer(), e.Position()))
	rtroute(e)
	g.GestureBindingInstance().GestureArena().Sweep(e.Pointer())
	rtLoop.FlushMicrotasks()
}

func rtcancel(e *g.PointerCancelEvent) {
	rtlog(fmt.Sprintf("[%s] cancel %d", rtnow(), e.Pointer()))
	rtroute(e)
}

func rtelapse(ms int) {
	rtLoop.Elapse(time.Duration(ms) * time.Millisecond)
	rtlog(fmt.Sprintf("[%s] elapsed %d", rtnow(), ms))
}

func rtshow(label string, o any) { rtlog(label + ": " + rtmask(o)) }

func rtTapRec(name string, buttons ...int) *g.TapGestureRecognizer {
	b := 7
	if len(buttons) > 0 {
		b = buttons[0]
	}
	r := g.NewTapGestureRecognizer()
	if b&1 != 0 {
		r.OnTapDown = func(d *g.TapDownDetails) { rtlog(name + " onTapDown " + rtmask(d)) }
		r.OnTapUp = func(d *g.TapUpDetails) { rtlog(name + " onTapUp " + rtmask(d)) }
		r.OnTap = func() { rtlog(name + " onTap") }
		r.OnTapMove = func(d *g.TapMoveDetails) {
			rtlog(fmt.Sprintf("%s onTapMove %s %s %s %s", name, d.GlobalPosition(), d.LocalPosition(), d.Delta, d.Kind.Name()))
		}
		r.OnTapCancel = func() { rtlog(name + " onTapCancel") }
	}
	if b&2 != 0 {
		r.OnSecondaryTapDown = func(d *g.TapDownDetails) { rtlog(name + " onSecondaryTapDown " + rtmask(d)) }
		r.OnSecondaryTapUp = func(d *g.TapUpDetails) { rtlog(name + " onSecondaryTapUp " + rtmask(d)) }
		r.OnSecondaryTap = func() { rtlog(name + " onSecondaryTap") }
		r.OnSecondaryTapCancel = func() { rtlog(name + " onSecondaryTapCancel") }
	}
	if b&4 != 0 {
		r.OnTertiaryTapDown = func(d *g.TapDownDetails) { rtlog(name + " onTertiaryTapDown " + rtmask(d)) }
		r.OnTertiaryTapUp = func(d *g.TapUpDetails) { rtlog(name + " onTertiaryTapUp " + rtmask(d)) }
		r.OnTertiaryTapCancel = func() { rtlog(name + " onTertiaryTapCancel") }
	}
	return r
}

func rtDoubleTapRec(name string) *g.DoubleTapGestureRecognizer {
	r := g.NewDoubleTapGestureRecognizer()
	r.OnDoubleTapDown = func(d *g.TapDownDetails) { rtlog(name + " onDoubleTapDown " + rtmask(d)) }
	r.OnDoubleTap = func() { rtlog(name + " onDoubleTap") }
	r.OnDoubleTapCancel = func() { rtlog(name + " onDoubleTapCancel") }
	return r
}

func rtLongPressRec(name string, opts ...g.LongPressGestureRecognizerOptions) *g.LongPressGestureRecognizer {
	r := g.NewLongPressGestureRecognizer(opts...)
	for _, b := range []string{"", "Secondary", "Tertiary"} {
		l := func(cb string, d ...any) {
			s := name + " on" + b + "LongPress" + cb
			if len(d) > 0 {
				s += " " + rtmask(d[0])
			}
			rtlog(s)
		}
		down := func(d *g.LongPressDownDetails) { l("Down", d) }
		cancel := func() { l("Cancel") }
		press := func() { l("") }
		start := func(d *g.LongPressStartDetails) { l("Start", d) }
		update := func(d *g.LongPressMoveUpdateDetails) { l("MoveUpdate", d) }
		upf := func() { l("Up") }
		end := func(d *g.LongPressEndDetails) { l("End", d) }
		switch b {
		case "":
			r.OnLongPressDown, r.OnLongPressCancel, r.OnLongPress, r.OnLongPressStart = down, cancel, press, start
			r.OnLongPressMoveUpdate, r.OnLongPressUp, r.OnLongPressEnd = update, upf, end
		case "Secondary":
			r.OnSecondaryLongPressDown, r.OnSecondaryLongPressCancel, r.OnSecondaryLongPress = down, cancel, press
			r.OnSecondaryLongPressStart, r.OnSecondaryLongPressMoveUpdate = start, update
			r.OnSecondaryLongPressUp, r.OnSecondaryLongPressEnd = upf, end
		case "Tertiary":
			r.OnTertiaryLongPressDown, r.OnTertiaryLongPressCancel, r.OnTertiaryLongPress = down, cancel, press
			r.OnTertiaryLongPressStart, r.OnTertiaryLongPressMoveUpdate = start, update
			r.OnTertiaryLongPressUp, r.OnTertiaryLongPressEnd = upf, end
		}
	}
	return r
}

func rtMultiTapRec(name string, longTapDelay time.Duration) *g.MultiTapGestureRecognizer {
	r := g.NewMultiTapGestureRecognizer(g.MultiTapGestureRecognizerOptions{LongTapDelay: longTapDelay})
	r.OnTapDown = func(p int, d *g.TapDownDetails) { rtlog(fmt.Sprintf("%s onTapDown %d %s", name, p, rtmask(d))) }
	r.OnTapUp = func(p int, d *g.TapUpDetails) { rtlog(fmt.Sprintf("%s onTapUp %d %s", name, p, rtmask(d))) }
	r.OnTap = func(p int) { rtlog(fmt.Sprintf("%s onTap %d", name, p)) }
	r.OnTapCancel = func(p int) { rtlog(fmt.Sprintf("%s onTapCancel %d", name, p)) }
	r.OnLongTapDown = func(p int, d *g.TapDownDetails) {
		rtlog(fmt.Sprintf("%s onLongTapDown %d %s", name, p, rtmask(d)))
	}
	return r
}

func rtSerialTapRec(name string) *g.SerialTapGestureRecognizer {
	r := g.NewSerialTapGestureRecognizer()
	r.OnSerialTapDown = func(d *g.SerialTapDownDetails) { rtlog(name + " onSerialTapDown " + rtmask(d)) }
	r.OnSerialTapUp = func(d *g.SerialTapUpDetails) { rtlog(name + " onSerialTapUp " + rtmask(d)) }
	r.OnSerialTapCancel = func(d *g.SerialTapCancelDetails) { rtlog(name + " onSerialTapCancel " + rtmask(d)) }
	return r
}

func rtForcePressRec(name string) *g.ForcePressGestureRecognizer {
	r := g.NewForcePressGestureRecognizer()
	r.OnStart = func(d *g.ForcePressDetails) { rtlog(name + " onStart " + rtmask(d)) }
	r.OnPeak = func(d *g.ForcePressDetails) { rtlog(name + " onPeak " + rtmask(d)) }
	r.OnUpdate = func(d *g.ForcePressDetails) { rtlog(name + " onUpdate " + rtmask(d)) }
	r.OnEnd = func(d *g.ForcePressDetails) { rtlog(name + " onEnd " + rtmask(d)) }
	return r
}

// rtEv holds the optional event parameters.
type rtEv struct {
	buttons     int
	kind        ui.PointerDeviceKind
	ms          int
	pressure    *float64
	pressureMax *float64
	dx, dy      float64
}

func (o rtEv) b() *int {
	b := o.buttons
	if b == 0 {
		b = g.KPrimaryButton
	}
	return &b
}

func (o rtEv) p() (*float64, *float64) {
	one := 1.0
	p, m := &one, &one
	if o.pressure != nil {
		p = o.pressure
	}
	if o.pressureMax != nil {
		m = o.pressureMax
	}
	return p, m
}

func rtF(v float64) *float64 { return &v }

func rtPDown(pointer int, x, y float64, opts ...rtEv) *g.PointerDownEvent {
	var o rtEv
	if len(opts) > 0 {
		o = opts[0]
	}
	p, m := o.p()
	return g.NewPointerDownEvent(g.PointerDownEventOptions{Pointer: pointer, Position: ui.NewOffset(x, y), Buttons: o.b(),
		Kind: o.kind, TimeStamp: time.Duration(o.ms) * time.Millisecond, Pressure: p, PressureMax: m})
}

func rtPMove(pointer int, x, y float64, opts ...rtEv) *g.PointerMoveEvent {
	var o rtEv
	if len(opts) > 0 {
		o = opts[0]
	}
	p, m := o.p()
	return g.NewPointerMoveEvent(g.PointerMoveEventOptions{Pointer: pointer, Position: ui.NewOffset(x, y), Buttons: o.b(),
		Kind: o.kind, TimeStamp: time.Duration(o.ms) * time.Millisecond, Pressure: p, PressureMax: m,
		Delta: ui.NewOffset(o.dx, o.dy)})
}

func rtPUp(pointer int, x, y float64, opts ...rtEv) *g.PointerUpEvent {
	var o rtEv
	if len(opts) > 0 {
		o = opts[0]
	}
	return g.NewPointerUpEvent(g.PointerUpEventOptions{Pointer: pointer, Position: ui.NewOffset(x, y), Kind: o.kind,
		TimeStamp: time.Duration(o.ms) * time.Millisecond})
}

func rtPCancel(pointer int) *g.PointerCancelEvent {
	return g.NewPointerCancelEvent(g.PointerCancelEventOptions{Pointer: pointer})
}

func rtRs(r ...g.GestureRecognizer) []g.GestureRecognizer { return r }

func rtscene(name string, body func()) {
	rtlog("== " + name)
	body()
	rtLoop.FlushMicrotasks()
}

func rtMs(n int) *time.Duration { d := time.Duration(n) * time.Millisecond; return &d }

func rtScenes() {
	rtscene("tap alone", func() {
		t := rtTapRec("tap")
		rtshow("before", t)
		rtdown(rtRs(t), rtPDown(1, 10, 10))
		rtshow("after down", t)
		rtmove(rtPMove(1, 12, 13, rtEv{dx: 2, dy: 3}))
		rtup(rtPUp(1, 12, 13))
		rtshow("after up", t)
		t.Dispose()
	})
	rtscene("tap vs long press: deadline then up", func() {
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		rtdown(rtRs(t, lp), rtPDown(1, 10, 10))
		rtelapse(50)
		rtelapse(60)
		rtshow("tap after deadline", t)
		rtshow("lp", lp)
		rtup(rtPUp(1, 10, 10))
		t.Dispose()
		lp.Dispose()
	})
	rtscene("tap vs long press: long press wins", func() {
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		rtdown(rtRs(t, lp), rtPDown(1, 10, 10, rtEv{ms: 0}))
		rtelapse(300)
		rtmove(rtPMove(1, 11, 10, rtEv{ms: 300, dx: 1}))
		rtelapse(250)
		rtmove(rtPMove(1, 20, 15, rtEv{ms: 560, dx: 9, dy: 5}))
		rtmove(rtPMove(1, 40, 25, rtEv{ms: 580, dx: 20, dy: 10}))
		rtmove(rtPMove(1, 60, 35, rtEv{ms: 600, dx: 20, dy: 10}))
		rtup(rtPUp(1, 60, 35, rtEv{ms: 610}))
		t.Dispose()
		lp.Dispose()
	})
	rtscene("long press too short", func() {
		lp := rtLongPressRec("lp")
		t := rtTapRec("tap")
		rtdown(rtRs(lp, t), rtPDown(1, 5, 5))
		rtelapse(300)
		rtup(rtPUp(1, 5, 5))
		rtelapse(500)
		lp.Dispose()
		t.Dispose()
	})
	rtscene("tap slop rejection before deadline", func() {
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		rtdown(rtRs(t, lp), rtPDown(1, 10, 10))
		rtmove(rtPMove(1, 35, 10, rtEv{dx: 25}))
		rtelapse(600)
		rtup(rtPUp(1, 35, 10))
		t.Dispose()
		lp.Dispose()
	})
	rtscene("tap won then slop: spontaneous cancel", func() {
		t := rtTapRec("tap")
		rtdown(rtRs(t), rtPDown(1, 10, 10))
		rtmove(rtPMove(1, 10, 40, rtEv{dy: 30}))
		rtup(rtPUp(1, 10, 40))
		t.Dispose()
	})
	rtscene("tap after deadline then slop: forced cancel", func() {
		t := rtTapRec("tap")
		t2 := rtTapRec("tap2")
		rtdown(rtRs(t, t2), rtPDown(1, 10, 10))
		rtelapse(150)
		rtmove(rtPMove(1, 10, 40, rtEv{dy: 30}))
		rtup(rtPUp(1, 10, 40))
		t.Dispose()
		t2.Dispose()
	})
	rtscene("two taps: first wins on sweep", func() {
		a := rtTapRec("a")
		b := rtTapRec("b")
		rtdown(rtRs(a, b), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		a.Dispose()
		b.Dispose()
	})
	rtscene("secondary and tertiary", func() {
		t := rtTapRec("tap")
		t2 := rtTapRec("tap2")
		rtdown(rtRs(t, t2), rtPDown(1, 10, 10, rtEv{buttons: g.KSecondaryButton}))
		rtelapse(120)
		rtup(rtPUp(1, 10, 10))
		rtdown(rtRs(t, t2), rtPDown(2, 20, 20, rtEv{buttons: g.KTertiaryButton}))
		rtup(rtPUp(2, 20, 20))
		rtdown(rtRs(t), rtPDown(3, 20, 20, rtEv{buttons: g.KSecondaryButton | g.KTertiaryButton}))
		rtup(rtPUp(3, 20, 20))
		onlyPrimary := rtTapRec("primary", 1)
		rtdown(rtRs(onlyPrimary, t), rtPDown(4, 30, 30, rtEv{buttons: g.KSecondaryButton}))
		rtup(rtPUp(4, 30, 30))
		t.Dispose()
		t2.Dispose()
		onlyPrimary.Dispose()
	})
	rtscene("buttons change during press", func() {
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		rtdown(rtRs(t, lp), rtPDown(1, 10, 10))
		rtelapse(150)
		rtmove(rtPMove(1, 10, 10, rtEv{buttons: g.KPrimaryButton | g.KSecondaryButton}))
		rtup(rtPUp(1, 10, 10))
		t.Dispose()
		lp.Dispose()
	})
	rtscene("cancel event", func() {
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		rtdown(rtRs(t, lp), rtPDown(1, 10, 10))
		rtelapse(150)
		rtcancel(rtPCancel(1))
		rtelapse(500)
		t.Dispose()
		lp.Dispose()
	})
	rtscene("dispose while won", func() {
		t := rtTapRec("tap")
		rtdown(rtRs(t), rtPDown(1, 10, 10))
		t.Dispose()
		rtup(rtPUp(1, 10, 10))
	})
	rtscene("double tap", func() {
		dt := rtDoubleTapRec("dt")
		t := rtTapRec("tap")
		rtdown(rtRs(dt, t), rtPDown(1, 10, 10))
		rtelapse(10)
		rtup(rtPUp(1, 11, 10))
		rtelapse(100)
		rtshow("dt between", dt)
		rtdown(rtRs(dt, t), rtPDown(2, 12, 12))
		rtelapse(10)
		rtup(rtPUp(2, 12, 12))
		rtelapse(400)
		dt.Dispose()
		t.Dispose()
	})
	rtscene("double tap timeout", func() {
		dt := rtDoubleTapRec("dt")
		t := rtTapRec("tap")
		rtdown(rtRs(dt, t), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		rtelapse(400)
		rtdown(rtRs(dt, t), rtPDown(2, 10, 10))
		rtup(rtPUp(2, 10, 10))
		rtelapse(400)
		dt.Dispose()
		t.Dispose()
	})
	rtscene("double tap: second tap too soon, too far, moved", func() {
		dt := rtDoubleTapRec("dt")
		rtdown(rtRs(dt), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		rtelapse(20)
		rtdown(rtRs(dt), rtPDown(2, 10, 10))
		rtup(rtPUp(2, 10, 10))
		rtelapse(100)
		rtdown(rtRs(dt), rtPDown(3, 200, 200))
		rtup(rtPUp(3, 200, 200))
		rtelapse(50)
		rtdown(rtRs(dt), rtPDown(4, 12, 10))
		rtmove(rtPMove(4, 40, 10, rtEv{dx: 28}))
		rtup(rtPUp(4, 40, 10))
		rtelapse(400)
		dt.Dispose()
	})
	rtscene("double tap: cancel second and buttons mismatch", func() {
		dt := rtDoubleTapRec("dt")
		rtdown(rtRs(dt), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		rtelapse(100)
		rtdown(rtRs(dt), rtPDown(2, 10, 10))
		rtcancel(rtPCancel(2))
		rtelapse(100)
		rtdown(rtRs(dt), rtPDown(3, 10, 10))
		rtup(rtPUp(3, 10, 10))
		rtelapse(100)
		dt2 := g.NewDoubleTapGestureRecognizer(g.GestureRecognizerOptions{AllowedButtonsFilter: func(int) bool { return true }})
		dt2.OnDoubleTap = func() { rtlog("dt2 onDoubleTap") }
		dt2.OnDoubleTapCancel = func() { rtlog("dt2 onDoubleTapCancel") }
		rtdown(rtRs(dt2), rtPDown(4, 10, 10))
		rtup(rtPUp(4, 10, 10))
		rtelapse(100)
		rtdown(rtRs(dt2), rtPDown(5, 10, 10, rtEv{buttons: g.KSecondaryButton}))
		rtup(rtPUp(5, 10, 10))
		rtelapse(400)
		dt.Dispose()
		dt2.Dispose()
	})
	rtscene("double tap: two pointers in first tap", func() {
		dt := rtDoubleTapRec("dt")
		rtdown(rtRs(dt), rtPDown(1, 10, 10))
		rtdown(rtRs(dt), rtPDown(2, 12, 10))
		rtup(rtPUp(1, 10, 10))
		rtup(rtPUp(2, 12, 10))
		rtelapse(400)
		dt.Dispose()
	})
	rtscene("eager vs tap", func() {
		e := g.NewEagerGestureRecognizer()
		t := rtTapRec("tap")
		rtdown(rtRs(t, e), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		e.Dispose()
		t.Dispose()
	})
	rtscene("team with captain", func() {
		team := g.NewGestureArenaTeam()
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		other := rtTapRec("other")
		t.SetTeam(team)
		lp.SetTeam(team)
		team.Captain = lp
		rtdown(rtRs(t, lp, other), rtPDown(1, 10, 10))
		rtelapse(120)
		rtup(rtPUp(1, 10, 10))
		t.Dispose()
		lp.Dispose()
		other.Dispose()
	})
	rtscene("team without captain, long press accepts", func() {
		team := g.NewGestureArenaTeam()
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		other := rtTapRec("other")
		t.SetTeam(team)
		lp.SetTeam(team)
		rtdown(rtRs(t, lp, other), rtPDown(1, 10, 10))
		rtelapse(600)
		rtup(rtPUp(1, 10, 10))
		t.Dispose()
		lp.Dispose()
		other.Dispose()
	})
	rtscene("long press secondary, tertiary, custom duration, post-accept slop", func() {
		lp := rtLongPressRec("lp", g.LongPressGestureRecognizerOptions{Duration: rtMs(200)})
		t := rtTapRec("tap")
		rtdown(rtRs(lp, t), rtPDown(1, 10, 10, rtEv{buttons: g.KSecondaryButton, ms: 0}))
		rtelapse(250)
		rtmove(rtPMove(1, 15, 10, rtEv{buttons: g.KSecondaryButton, ms: 260, dx: 5}))
		rtup(rtPUp(1, 15, 10, rtEv{ms: 270}))
		rtdown(rtRs(lp), rtPDown(2, 10, 10, rtEv{buttons: g.KTertiaryButton, ms: 300}))
		rtelapse(250)
		rtup(rtPUp(2, 10, 10, rtEv{ms: 560}))
		lp2 := rtLongPressRec("lp2", g.LongPressGestureRecognizerOptions{PostAcceptSlopTolerance: rtF(5)})
		rtdown(rtRs(lp2), rtPDown(3, 10, 10, rtEv{ms: 600}))
		rtelapse(600)
		rtmove(rtPMove(3, 14, 10, rtEv{ms: 1210, dx: 4}))
		rtmove(rtPMove(3, 30, 10, rtEv{ms: 1220, dx: 16}))
		rtup(rtPUp(3, 30, 10, rtEv{ms: 1230}))
		rtshow("lp2", lp2)
		lp.Dispose()
		lp2.Dispose()
		t.Dispose()
	})
	rtscene("long press buttons change and slop", func() {
		lp := rtLongPressRec("lp")
		rtdown(rtRs(lp), rtPDown(1, 10, 10))
		rtmove(rtPMove(1, 10, 10, rtEv{buttons: 3}))
		rtelapse(600)
		rtdown(rtRs(lp), rtPDown(2, 10, 10))
		rtmove(rtPMove(2, 40, 10, rtEv{dx: 30}))
		rtelapse(600)
		rtup(rtPUp(2, 40, 10))
		rtdown(rtRs(lp), rtPDown(3, 10, 10))
		rtelapse(600)
		rtmove(rtPMove(3, 10, 10, rtEv{buttons: 3}))
		rtup(rtPUp(3, 10, 10))
		lp.Dispose()
	})
	rtscene("multitap", func() {
		m := rtMultiTapRec("m", 300*time.Millisecond)
		mouse := ui.PointerDeviceKindMouse
		rtdown(rtRs(m), rtPDown(1, 10, 10))
		rtdown(rtRs(m), rtPDown(2, 50, 50))
		rtmove(rtPMove(1, 15, 12, rtEv{dx: 5, dy: 2}))
		rtelapse(350)
		rtup(rtPUp(1, 15, 12))
		rtmove(rtPMove(2, 90, 50, rtEv{dx: 40}))
		rtup(rtPUp(2, 90, 50))
		rtdown(rtRs(m), rtPDown(3, 10, 10, rtEv{kind: mouse}))
		rtmove(rtPMove(3, 12, 10, rtEv{kind: mouse, dx: 2}))
		rtup(rtPUp(3, 12, 10, rtEv{kind: mouse}))
		t := rtTapRec("tap")
		rtdown(rtRs(t, m), rtPDown(4, 10, 10))
		rtcancel(rtPCancel(4))
		rtdown(rtRs(m), rtPDown(5, 10, 10))
		m.Dispose()
		t.Dispose()
	})
	rtscene("serial tap", func() {
		s := rtSerialTapRec("s")
		t := rtTapRec("tap")
		for i := 1; i <= 3; i++ {
			rtdown(rtRs(s, t), rtPDown(i, 10, 10))
			rtup(rtPUp(i, 10, 10))
			rtelapse(100)
		}
		rtelapse(300)
		rtdown(rtRs(s), rtPDown(4, 10, 10))
		rtup(rtPUp(4, 10, 10))
		rtelapse(50)
		rtdown(rtRs(s), rtPDown(5, 10, 10))
		rtmove(rtPMove(5, 40, 10, rtEv{dx: 30}))
		rtelapse(50)
		rtdown(rtRs(s), rtPDown(6, 10, 10))
		rtup(rtPUp(6, 10, 10))
		rtelapse(10)
		rtdown(rtRs(s), rtPDown(7, 10, 10))
		rtup(rtPUp(7, 10, 10))
		rtelapse(100)
		rtdown(rtRs(s), rtPDown(8, 300, 10))
		rtcancel(rtPCancel(8))
		s.Dispose()
		t.Dispose()
	})
	rtscene("force press", func() {
		fp := rtForcePressRec("f")
		t := rtTapRec("tap")
		six := rtF(6)
		rtdown(rtRs(fp, t), rtPDown(1, 10, 10, rtEv{pressure: rtF(0), pressureMax: six}))
		rtmove(rtPMove(1, 10, 10, rtEv{pressure: rtF(1), pressureMax: six}))
		rtmove(rtPMove(1, 10, 10, rtEv{pressure: rtF(3), pressureMax: six}))
		rtmove(rtPMove(1, 11, 10, rtEv{pressure: rtF(5.5), pressureMax: six, dx: 1}))
		rtmove(rtPMove(1, 11, 10, rtEv{pressure: rtF(6), pressureMax: six}))
		rtup(rtPUp(1, 11, 10))
		rtdown(rtRs(fp, t), rtPDown(2, 10, 10, rtEv{pressure: rtF(0), pressureMax: rtF(1)}))
		rtup(rtPUp(2, 10, 10))
		rtdown(rtRs(fp), rtPDown(3, 10, 10, rtEv{pressure: rtF(0), pressureMax: six}))
		rtmove(rtPMove(3, 10, 10, rtEv{pressure: rtF(2), pressureMax: six}))
		rtmove(rtPMove(3, 10, 10, rtEv{pressure: rtF(4), pressureMax: six}))
		rtup(rtPUp(3, 10, 10))
		rtdown(rtRs(fp, t), rtPDown(4, 10, 10, rtEv{pressure: rtF(0), pressureMax: six}))
		rtmove(rtPMove(4, 30, 10, rtEv{pressure: rtF(1), pressureMax: six, dx: 5}))
		rtup(rtPUp(4, 30, 10))
		rtdown(rtRs(fp), rtPDown(5, 10, 10, rtEv{pressure: rtF(7), pressureMax: six}))
		rtup(rtPUp(5, 10, 10))
		fp.Dispose()
		t.Dispose()
	})
	rtscene("supported devices and slops", func() {
		mouse := ui.PointerDeviceKindMouse
		t := g.NewTapGestureRecognizer(g.TapGestureRecognizerOptions{GestureRecognizerOptions: g.GestureRecognizerOptions{
			SupportedDevices: map[ui.PointerDeviceKind]struct{}{mouse: {}}}})
		t.OnTap = func() { rtlog("mouseOnly onTap") }
		t.OnTapCancel = func() { rtlog("mouseOnly onTapCancel") }
		t2 := rtTapRec("any")
		rtdown(rtRs(t, t2), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		rtdown(rtRs(t, t2), rtPDown(2, 10, 10, rtEv{kind: mouse}))
		rtup(rtPUp(2, 10, 10, rtEv{kind: mouse}))
		loose := g.NewTapGestureRecognizer(g.TapGestureRecognizerOptions{NoPreAcceptSlopTolerance: true, PostAcceptSlopTolerance: rtF(50)})
		loose.OnTap = func() { rtlog("loose onTap") }
		loose.OnTapCancel = func() { rtlog("loose onTapCancel") }
		other := rtTapRec("other")
		rtdown(rtRs(loose, other), rtPDown(3, 10, 10))
		rtmove(rtPMove(3, 100, 10, rtEv{dx: 90}))
		rtup(rtPUp(3, 100, 10))
		tight := g.NewTapGestureRecognizer(g.TapGestureRecognizerOptions{PreAcceptSlopTolerance: rtF(2), PostAcceptSlopTolerance: rtF(3)})
		tight.OnTap = func() { rtlog("tight onTap") }
		tight.OnTapCancel = func() { rtlog("tight onTapCancel") }
		tight.OnTapDown = func(*g.TapDownDetails) { rtlog("tight onTapDown") }
		rtdown(rtRs(tight), rtPDown(4, 10, 10))
		rtmove(rtPMove(4, 12, 11, rtEv{dx: 2, dy: 1}))
		rtmove(rtPMove(4, 14, 11, rtEv{dx: 2}))
		rtup(rtPUp(4, 14, 11))
		rtshow("tight", tight)
		t.Dispose()
		t2.Dispose()
		loose.Dispose()
		other.Dispose()
		tight.Dispose()
	})
	rtscene("multiple pointers on tap", func() {
		t := rtTapRec("tap")
		lp := rtLongPressRec("lp")
		rtdown(rtRs(t, lp), rtPDown(1, 10, 10))
		rtdown(rtRs(t, lp), rtPDown(2, 20, 20))
		rtshow("tap two pointers", t)
		rtup(rtPUp(2, 20, 20))
		rtelapse(120)
		rtup(rtPUp(1, 10, 10))
		t.Dispose()
		lp.Dispose()
	})
	rtscene("secondary-only callbacks, move with secondary button", func() {
		s := g.NewTapGestureRecognizer()
		s.OnSecondaryTap = func() { rtlog("s onSecondaryTap") }
		rtdown(rtRs(s), rtPDown(1, 10, 10, rtEv{buttons: g.KSecondaryButton}))
		rtup(rtPUp(1, 10, 10))
		t := rtTapRec("tap")
		rtdown(rtRs(t), rtPDown(2, 10, 10, rtEv{buttons: g.KSecondaryButton}))
		rtmove(rtPMove(2, 12, 10, rtEv{buttons: g.KSecondaryButton, dx: 2}))
		rtup(rtPUp(2, 12, 10))
		s.Dispose()
		t.Dispose()
	})
	rtscene("transformed tap", func() {
		t := rtTapRec("tap")
		m := vm.NewMatrix4TranslationValues(5, -3, 0)
		d := rtPDown(1, 10, 10).Transformed(m).(*g.PointerDownEvent)
		rtdown(rtRs(t), d)
		rtmove(rtPMove(1, 12, 11, rtEv{dx: 2, dy: 1}))
		rtup(rtPUp(1, 12, 11))
		t.Dispose()
	})
	rtscene("double tap within double-tap slop, multitap without long tap", func() {
		dt := rtDoubleTapRec("dt")
		rtdown(rtRs(dt), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		rtelapse(100)
		rtdown(rtRs(dt), rtPDown(2, 70, 10))
		rtup(rtPUp(2, 70, 10))
		rtelapse(400)
		dt.Dispose()
		m := rtMultiTapRec("m0", 0)
		rtdown(rtRs(m), rtPDown(3, 10, 10))
		rtelapse(1000)
		rtup(rtPUp(3, 10, 10))
		m.Dispose()
	})
	rtscene("serial taps with different buttons", func() {
		s := rtSerialTapRec("s")
		rtdown(rtRs(s), rtPDown(1, 10, 10))
		rtup(rtPUp(1, 10, 10))
		rtelapse(100)
		rtdown(rtRs(s), rtPDown(2, 10, 10, rtEv{buttons: g.KSecondaryButton}))
		rtup(rtPUp(2, 10, 10))
		rtelapse(400)
		s.Dispose()
	})
	rtscene("long press with synthesized moves", func() {
		lp := rtLongPressRec("lp")
		rtdown(rtRs(lp), rtPDown(1, 10, 10, rtEv{ms: 0}))
		rtelapse(600)
		rtmove(g.NewPointerMoveEvent(g.PointerMoveEventOptions{Pointer: 1, Position: ui.NewOffset(30, 10),
			Delta: ui.NewOffset(20, 0), TimeStamp: 610 * time.Millisecond, Synthesized: true}))
		rtmove(rtPMove(1, 31, 10, rtEv{ms: 620, dx: 1}))
		rtmove(rtPMove(1, 33, 10, rtEv{ms: 630, dx: 2}))
		rtup(rtPUp(1, 33, 10, rtEv{ms: 640}))
		lp.Dispose()
	})
	rtscene("force press thresholds and rejection", func() {
		fp := rtForcePressRec("f")
		t := rtTapRec("tap")
		six := rtF(6)
		rtdown(rtRs(fp, t), rtPDown(1, 10, 10, rtEv{pressure: rtF(0), pressureMax: six}))
		rtmove(rtPMove(1, 10, 10, rtEv{pressure: rtF(2.7), pressureMax: six}))
		rtmove(rtPMove(1, 10, 10, rtEv{pressure: rtF(3.2), pressureMax: six}))
		rtup(rtPUp(1, 10, 10))
		rtdown(rtRs(fp, t), rtPDown(2, 10, 10, rtEv{pressure: rtF(0), pressureMax: six}))
		rtup(rtPUp(2, 10, 10))
		fp.Dispose()
		t.Dispose()
	})
	rtscene("details defaults", func() {
		stylus := ui.PointerDeviceKindStylus
		_ = stylus
		rtshow("TapDownDetails", g.NewTapDownDetails())
		rtshow("TapUpDetails", g.NewTapUpDetails(g.TapUpDetailsOptions{Kind: ui.PointerDeviceKindMouse, GlobalPosition: ui.NewOffset(1, 2)}))
		rtshow("LongPressDownDetails", g.NewLongPressDownDetails())
		rtshow("LongPressStartDetails", g.NewLongPressStartDetails(g.LongPressStartDetailsOptions{GlobalPosition: ui.NewOffset(3, 4)}))
		rtshow("LongPressMoveUpdateDetails", g.NewLongPressMoveUpdateDetails(g.LongPressMoveUpdateDetailsOptions{OffsetFromOrigin: ui.NewOffset(1, 1)}))
		rtshow("LongPressEndDetails", g.NewLongPressEndDetails(g.LongPressEndDetailsOptions{Velocity: g.Velocity{PixelsPerSecond: ui.NewOffset(5, 6)}}))
		rtshow("SerialTapDownDetails", g.NewSerialTapDownDetails(g.SerialTapDownDetailsOptions{Kind: ui.PointerDeviceKindStylus}))
		rtshow("SerialTapUpDetails", g.NewSerialTapUpDetails(g.SerialTapUpDetailsOptions{Count: 3}))
		rtshow("SerialTapCancelDetails", g.NewSerialTapCancelDetails(2))
		rtshow("ForcePressDetails", g.NewForcePressDetails(g.ForcePressDetailsOptions{GlobalPosition: ui.NewOffset(1, 1), Pressure: 0.5}))
		t := g.NewTapGestureRecognizer(g.TapGestureRecognizerOptions{GestureRecognizerOptions: g.GestureRecognizerOptions{DebugOwner: "owner"}})
		rtshow("tap with owner", t)
		rtlog("descriptions " + strings.Join([]string{t.DebugDescription(), rtDoubleTapRec("x").DebugDescription(),
			rtMultiTapRec("x", 0).DebugDescription(), rtSerialTapRec("x").DebugDescription(),
			rtLongPressRec("x").DebugDescription(), rtForcePressRec("x").DebugDescription()}, " "))
		t.Dispose()
	})
}

func TestRecognizersTapMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/recognizers_tap.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	rtLines = nil
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		rtLoop = tester.Async
		previous := foundation.DebugPrint
		foundation.DebugPrint = func(message string, opts ...foundation.DebugPrintOptions) { rtlog("print: " + rtmask(message)) }
		g.DebugPrintRecognizerCallbacksTrace = true
		g.DebugPrintGestureArenaDiagnostics = true
		defer func() {
			g.DebugPrintRecognizerCallbacksTrace = false
			g.DebugPrintGestureArenaDiagnostics = false
			foundation.DebugPrint = previous
		}()
		rtScenes()
	})
	for i := 0; i < max(len(rtLines), len(ref.Log)); i++ {
		var w, gl string
		if i < len(ref.Log) {
			w = ref.Log[i]
		}
		if i < len(rtLines) {
			gl = rtLines[i]
		}
		if w != gl {
			t.Fatalf("line %d:\nflutter: %s\ngo:      %s", i, w, gl)
		}
	}
	t.Logf("%d lines match", len(ref.Log))
}
