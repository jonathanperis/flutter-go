package difftest

// Go twin of flutter/test/recognizers_tap_and_drag_test.dart: the scenes
// of recognizers_tap_and_drag_scenes.json (scripted pointer sequences
// through GestureBinding's pointer router and arena, as
// gesture_tester.dart does) on the tap-and-drag recognizers, alone and
// competing with the other recognizers; every callback logged with all
// details and the details' toString, plus the callback trace and arena
// diagnostics, compared line by line.

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	g "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var tadLog []string

func tadlog(s string) { tadLog = append(tadLog, s) }

var tadHash = regexp.MustCompile(`#[0-9a-f]{5}`)

func tadHashless(s string) string { return tadHash.ReplaceAllString(s, "#HASH") }

func tadO(v ui.Offset) string {
	return "(" + dart.DoubleToString(v.Dx) + "," + dart.DoubleToString(v.Dy) + ")"
}

func tadTS(v *time.Duration) string {
	if v == nil {
		return "null"
	}
	return strconv.FormatInt(v.Microseconds(), 10)
}

func tadPD(v *float64) string {
	if v == nil {
		return "null"
	}
	return dart.DoubleToString(*v)
}

func tadKind(k *ui.PointerDeviceKind) string {
	if k == nil {
		return "null"
	}
	return k.Name()
}

func tadKindOf(name string) ui.PointerDeviceKind {
	for _, k := range ui.PointerDeviceKindValues {
		if k.Name() == name {
			return k
		}
	}
	panic("kind " + name)
}

func tadDetails(d foundation.Diagnosticable) { tadlog("  " + tadHashless(d.String())) }

type tadOp struct {
	Op      string    `json:"op"`
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Dsb     *string   `json:"dsb"`
	Eager   *bool     `json:"eager"`
	Max     *int      `json:"max"`
	Thr     *int      `json:"throttle"`
	Slop    *float64  `json:"slop"`
	Devices []string  `json:"devices"`
	P       string    `json:"p"`
	Pointer int       `json:"pointer"`
	Kind    string    `json:"kind"`
	Buttons *int      `json:"buttons"`
	X       float64   `json:"x"`
	Y       float64   `json:"y"`
	TS      int       `json:"ts"`
	Add     []string  `json:"add"`
	M       []float64 `json:"m"`
	Ms      int       `json:"ms"`
}

// PassiveRecognizer is the Dart test's PassiveRecognizer: it joins the arena and
// never resolves until it stops tracking (or resigns).
type PassiveRecognizer struct {
	g.OneSequenceGestureRecognizerBase
	name string
}

func (r *PassiveRecognizer) AddPointer(event *g.PointerDownEvent) {
	r.StartTrackingPointer(event.Pointer())
}
func (r *PassiveRecognizer) DebugDescription() string       { return "passive" }
func (r *PassiveRecognizer) DidStopTrackingLastPointer(int) { r.Resolve(g.GestureDispositionRejected) }
func (r *PassiveRecognizer) HandleEvent(event g.PointerEvent) {
	switch event.(type) {
	case *g.PointerUpEvent, *g.PointerCancelEvent:
		r.StopTrackingPointer(event.Pointer())
	}
}
func (r *PassiveRecognizer) AcceptGesture(pointer int) {
	tadlog(r.name + " accept " + strconv.Itoa(pointer))
}
func (r *PassiveRecognizer) RejectGesture(pointer int) {
	tadlog(r.name + " reject " + strconv.Itoa(pointer))
}

func tadMakeRecognizer(op tadOp) g.GestureRecognizer {
	id := op.ID
	add := func(s string) { tadlog(id + " " + s) }
	switch op.Type {
	case "hd", "pan", "tad":
		var devices map[ui.PointerDeviceKind]struct{}
		if op.Devices != nil {
			devices = map[ui.PointerDeviceKind]struct{}{}
			for _, k := range op.Devices {
				devices[tadKindOf(k)] = struct{}{}
			}
		}
		opts := g.GestureRecognizerOptions{SupportedDevices: devices}
		var r g.GestureRecognizer
		var base *g.BaseTapAndDragGestureRecognizerBase
		switch op.Type {
		case "hd":
			h := g.NewTapAndHorizontalDragGestureRecognizer(opts)
			r, base = h, &h.BaseTapAndDragGestureRecognizerBase
		case "pan":
			p := g.NewTapAndPanGestureRecognizer(opts)
			r, base = p, &p.BaseTapAndDragGestureRecognizerBase
		default:
			p := g.NewTapAndDragGestureRecognizer(opts)
			r, base = p, &p.BaseTapAndDragGestureRecognizerBase
		}
		if op.Dsb != nil {
			if *op.Dsb == "down" {
				base.DragStartBehavior = g.DragStartBehaviorDown
			} else {
				base.DragStartBehavior = g.DragStartBehaviorStart
			}
		}
		if op.Eager != nil {
			base.EagerVictoryOnDrag = *op.Eager
		}
		if op.Max != nil {
			m := *op.Max
			base.MaxConsecutiveTap = &m
		}
		if op.Thr != nil {
			d := time.Duration(*op.Thr) * time.Millisecond
			base.DragUpdateThrottleFrequency = &d
		}
		if op.Slop != nil {
			s := *op.Slop
			base.GestureSettings = &g.DeviceGestureSettings{TouchSlop: &s}
		}
		base.OnTapTrackStart = func() { add("onTapTrackStart") }
		base.OnTapTrackReset = func() { add("onTapTrackReset") }
		base.OnTapDown = func(x *g.TapDragDownDetails) {
			add("onTapDown g=" + tadO(x.GlobalPosition()) + " l=" + tadO(x.LocalPosition()) + " kind=" + tadKind(x.Kind) +
				" count=" + strconv.Itoa(x.ConsecutiveTapCount))
			tadDetails(x)
		}
		base.OnTapUp = func(x *g.TapDragUpDetails) {
			k := x.Kind
			add("onTapUp g=" + tadO(x.GlobalPosition()) + " l=" + tadO(x.LocalPosition()) + " kind=" + tadKind(&k) +
				" count=" + strconv.Itoa(x.ConsecutiveTapCount))
			tadDetails(x)
		}
		base.OnDragStart = func(x *g.TapDragStartDetails) {
			add("onDragStart g=" + tadO(x.GlobalPosition()) + " l=" + tadO(x.LocalPosition()) + " ts=" + tadTS(x.SourceTimeStamp) +
				" kind=" + tadKind(x.Kind) + " count=" + strconv.Itoa(x.ConsecutiveTapCount))
			tadDetails(x)
		}
		base.OnDragUpdate = func(x *g.TapDragUpdateDetails) {
			add("onDragUpdate g=" + tadO(x.GlobalPosition()) + " l=" + tadO(x.LocalPosition()) + " ts=" + tadTS(x.SourceTimeStamp) +
				" delta=" + tadO(x.Delta) + " primary=" + tadPD(x.PrimaryDelta) + " kind=" + tadKind(x.Kind) +
				" origin=" + tadO(x.OffsetFromOrigin) + " localOrigin=" + tadO(x.LocalOffsetFromOrigin) +
				" count=" + strconv.Itoa(x.ConsecutiveTapCount))
			tadDetails(x)
		}
		base.OnDragEnd = func(x *g.TapDragEndDetails) {
			add("onDragEnd g=" + tadO(x.GlobalPosition()) + " l=" + tadO(x.LocalPosition()) +
				" v=" + tadO(x.Velocity.PixelsPerSecond) + " primary=" + tadPD(x.PrimaryVelocity) +
				" count=" + strconv.Itoa(x.ConsecutiveTapCount))
			tadDetails(x)
		}
		base.OnCancel = func() { add("onCancel") }
		return r
	case "tap":
		r := g.NewTapGestureRecognizer()
		r.OnTapDown = func(x *g.TapDownDetails) { add("onTapDown g=" + tadO(x.GlobalPosition())) }
		r.OnTapUp = func(x *g.TapUpDetails) { add("onTapUp g=" + tadO(x.GlobalPosition())) }
		r.OnTap = func() { add("onTap") }
		r.OnTapCancel = func() { add("onTapCancel") }
		return r
	case "hdrag", "vdrag", "panr":
		var r g.GestureRecognizer
		var base *g.DragGestureRecognizerBase
		switch op.Type {
		case "hdrag":
			h := g.NewHorizontalDragGestureRecognizer()
			r, base = h, &h.DragGestureRecognizerBase
		case "vdrag":
			v := g.NewVerticalDragGestureRecognizer()
			r, base = v, &v.DragGestureRecognizerBase
		default:
			p := g.NewPanGestureRecognizer()
			r, base = p, &p.DragGestureRecognizerBase
		}
		base.OnDown = func(x *g.DragDownDetails) { add("onDown g=" + tadO(x.GlobalPosition())) }
		base.OnStart = func(x *g.DragStartDetails) { add("onStart g=" + tadO(x.GlobalPosition())) }
		base.OnUpdate = func(x *g.DragUpdateDetails) { add("onUpdate delta=" + tadO(x.Delta)) }
		base.OnEnd = func(x *g.DragEndDetails) {
			add("onEnd v=" + tadO(x.Velocity.PixelsPerSecond) + " primary=" + tadPD(x.PrimaryVelocity))
		}
		base.OnCancel = func() { add("onCancel") }
		return r
	case "scale":
		r := g.NewScaleGestureRecognizer()
		r.OnStart = func(x *g.ScaleStartDetails) { add("onStart focal=" + tadO(x.FocalPoint)) }
		r.OnUpdate = func(x *g.ScaleUpdateDetails) {
			add("onUpdate focal=" + tadO(x.FocalPoint) + " scale=" + dart.DoubleToString(x.Scale))
		}
		r.OnEnd = func(*g.ScaleEndDetails) { add("onEnd") }
		return r
	case "lp":
		r := g.NewLongPressGestureRecognizer()
		r.OnLongPressStart = func(*g.LongPressStartDetails) { add("onLongPressStart") }
		r.OnLongPressEnd = func(*g.LongPressEndDetails) { add("onLongPressEnd") }
		r.OnLongPressCancel = func() { add("onLongPressCancel") }
		return r
	case "dtap":
		r := g.NewDoubleTapGestureRecognizer()
		r.OnDoubleTapDown = func(*g.TapDownDetails) { add("onDoubleTapDown") }
		r.OnDoubleTap = func() { add("onDoubleTap") }
		r.OnDoubleTapCancel = func() { add("onDoubleTapCancel") }
		return r
	case "eager":
		return g.NewEagerGestureRecognizer()
	case "passive":
		r := &PassiveRecognizer{name: id}
		r.Construct(r, g.GestureRecognizerOptions{})
		return r
	case "tadnone":
		return g.NewTapAndPanGestureRecognizer()
	}
	panic(op.Type)
}

func TestRecognizersTapAndDragMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/recognizers_tap_and_drag.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	scenesData, err := os.ReadFile("recognizers_tap_and_drag_scenes.json")
	if err != nil {
		t.Fatal(err)
	}
	var scenes []struct {
		Name string            `json:"name"`
		Ops  []json.RawMessage `json:"ops"`
	}
	if err := json.Unmarshal(scenesData, &scenes); err != nil {
		t.Fatal(err)
	}
	tadLog = nil
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		oldPrint := foundation.DebugPrint
		foundation.DebugPrint = func(m string, _ ...foundation.DebugPrintOptions) { tadlog("print " + tadHashless(m)) }
		g.DebugPrintRecognizerCallbacksTrace = true
		g.DebugPrintGestureArenaDiagnostics = true
		defer func() {
			g.DebugPrintRecognizerCallbacksTrace = false
			g.DebugPrintGestureArenaDiagnostics = false
			foundation.DebugPrint = oldPrint
		}()
		for _, scene := range scenes {
			tadlog("scene " + scene.Name)
			recognizers := map[string]g.GestureRecognizer{}
			pointers := map[string]*flutter_test.TestPointer{}
			downs := map[string]*g.PointerDownEvent{}
			for _, raw := range scene.Ops {
				var compact bytes.Buffer
				if err := json.Compact(&compact, raw); err != nil {
					t.Fatal(err)
				}
				tadlog("> " + compact.String())
				var op tadOp
				if err := json.Unmarshal(raw, &op); err != nil {
					t.Fatal(err)
				}
				ts := time.Duration(op.TS) * time.Millisecond
				switch op.Op {
				case "new":
					recognizers[op.ID] = tadMakeRecognizer(op)
				case "ptr":
					pointer, buttons := op.Pointer, *op.Buttons
					pointers[op.P] = flutter_test.NewTestPointer(flutter_test.TestPointerOptions{Pointer: &pointer,
						Kind: tadKindOf(op.Kind), Buttons: &buttons})
				case "down":
					down := pointers[op.P].Down(ui.NewOffset(op.X, op.Y),
						flutter_test.TestPointerEventOptions{TimeStamp: ts, Buttons: op.Buttons})
					downs[op.P] = down
					added := down
					if op.M != nil {
						added = down.Transformed(vm.NewMatrix4FromList(op.M)).(*g.PointerDownEvent)
					}
					for _, id := range op.Add {
						recognizers[id].AddPointer(added)
					}
				case "routedown":
					tester.Route(downs[op.P])
				case "close":
					g.GestureBindingInstance().GestureArena().Close(op.Pointer)
				case "sweep":
					g.GestureBindingInstance().GestureArena().Sweep(op.Pointer)
				case "move":
					tester.Route(pointers[op.P].Move(ui.NewOffset(op.X, op.Y),
						flutter_test.TestPointerEventOptions{TimeStamp: ts, Buttons: op.Buttons}))
				case "up":
					tester.Route(pointers[op.P].Up(flutter_test.TestPointerEventOptions{TimeStamp: ts}))
				case "cancel":
					tester.Route(pointers[op.P].Cancel(flutter_test.TestPointerEventOptions{TimeStamp: ts}))
				case "elapse":
					tester.Async.Elapse(time.Duration(op.Ms) * time.Millisecond)
				case "flush":
					tester.Async.FlushMicrotasks()
				case "state":
					r := recognizers[op.ID].(g.BaseTapAndDragGestureRecognizer)
					ptr := func(e g.PointerEvent, isNil bool) string {
						if isNil {
							return "null"
						}
						return strconv.Itoa(e.Pointer())
					}
					tadlog("state count=" + strconv.Itoa(r.ConsecutiveTapCount()) +
						" down=" + ptr(r.CurrentDown(), r.CurrentDown() == nil) +
						" up=" + ptr(r.CurrentUp(), r.CurrentUp() == nil) +
						" " + tadHashless(r.String()) + " " + r.DebugDescription())
				case "resign":
					recognizers[op.ID].(g.OneSequenceGestureRecognizer).Resolve(g.GestureDispositionRejected)
					tester.Async.FlushMicrotasks()
				case "dispose":
					recognizers[op.ID].Dispose()
				default:
					t.Fatalf("op %s", op.Op)
				}
			}
			tester.Async.Elapse(2 * time.Second)
		}
	})
	for i := 0; i < max(len(tadLog), len(ref.Log)); i++ {
		var w, gl string
		if i < len(ref.Log) {
			w = ref.Log[i]
		}
		if i < len(tadLog) {
			gl = tadLog[i]
		}
		if w != gl {
			t.Fatalf("line %d:\nflutter: %s\ngo:      %s", i, w, gl)
		}
	}
	if len(ref.Log) < 10000 {
		t.Fatalf("reference too short: %d lines", len(ref.Log))
	}
}
