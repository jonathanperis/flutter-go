package difftest

// Go twin of flutter/test/recognizers_drag_test.dart: the scenes of
// recognizers_drag_scenes.json (scripted pointer sequences through
// GestureBinding's pointer router and arena, as gesture_tester.dart does)
// on the drag and multi-drag recognizers, every callback logged with all
// details, compared line by line.

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
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var rdLog []string

func rdlog(s string) { rdLog = append(rdLog, s) }

func rdOff(v ui.Offset) string {
	return "(" + dart.DoubleToString(v.Dx) + "," + dart.DoubleToString(v.Dy) + ")"
}

func rdTS(v *time.Duration) string {
	if v == nil {
		return "null"
	}
	return strconv.FormatInt(v.Microseconds(), 10)
}

func rdPD(v *float64) string {
	if v == nil {
		return "null"
	}
	return dart.DoubleToString(*v)
}

func rdKind(k *ui.PointerDeviceKind) string {
	if k == nil {
		return "null"
	}
	return k.Name()
}

var rdHash = regexp.MustCompile(`#[0-9a-f]{5}`)

func rdKindOf(name string) ui.PointerDeviceKind {
	for _, k := range ui.PointerDeviceKindValues {
		if k.Name() == name {
			return k
		}
	}
	panic("kind " + name)
}

type rdDrag struct{ name string }

func (d *rdDrag) Update(x *gestures.DragUpdateDetails) {
	rdlog(d.name + " drag.update g=" + rdOff(x.GlobalPosition()) + " l=" + rdOff(x.LocalPosition()) +
		" delta=" + rdOff(x.Delta) + " primary=" + rdPD(x.PrimaryDelta) + " ts=" + rdTS(x.SourceTimeStamp))
}

func (d *rdDrag) End(x *gestures.DragEndDetails) {
	rdlog(d.name + " drag.end v=" + rdOff(x.Velocity.PixelsPerSecond) + " primary=" + rdPD(x.PrimaryVelocity) +
		" g=" + rdOff(x.GlobalPosition()) + " l=" + rdOff(x.LocalPosition()))
}

func (d *rdDrag) Cancel() { rdlog(d.name + " drag.cancel") }

type rdSpec map[string]any

func (s rdSpec) str(k string) string { v, _ := s[k].(string); return v }

func rdMakeRecognizer(spec rdSpec) gestures.GestureRecognizer {
	id := spec.str("id")
	var opts gestures.GestureRecognizerOptions
	if devs, ok := spec["devices"].([]any); ok {
		opts.SupportedDevices = map[ui.PointerDeviceKind]struct{}{}
		for _, k := range devs {
			opts.SupportedDevices[rdKindOf(k.(string))] = struct{}{}
		}
	}
	switch spec.str("type") {
	case "eager":
		return gestures.NewEagerGestureRecognizer(opts)
	case "immediate", "hmulti", "vmulti", "delayed":
		var r gestures.MultiDragGestureRecognizer
		var base *gestures.MultiDragGestureRecognizerBase
		switch spec.str("type") {
		case "immediate":
			m := gestures.NewImmediateMultiDragGestureRecognizer(opts)
			r, base = m, &m.MultiDragGestureRecognizerBase
		case "hmulti":
			m := gestures.NewHorizontalMultiDragGestureRecognizer(opts)
			r, base = m, &m.MultiDragGestureRecognizerBase
		case "vmulti":
			m := gestures.NewVerticalMultiDragGestureRecognizer(opts)
			r, base = m, &m.MultiDragGestureRecognizerBase
		default:
			o := gestures.DelayedMultiDragGestureRecognizerOptions{GestureRecognizerOptions: opts}
			if d, ok := spec["delay"].(float64); ok {
				delay := time.Duration(d) * time.Millisecond
				o.Delay = &delay
			}
			m := gestures.NewDelayedMultiDragGestureRecognizer(o)
			r, base = m, &m.MultiDragGestureRecognizerBase
		}
		nullDrag := spec["nullDrag"] == true
		base.OnStart = func(position ui.Offset) gestures.Drag {
			rdlog(id + " onStart " + rdOff(position))
			if nullDrag {
				return nil
			}
			return &rdDrag{name: id}
		}
		return r
	}
	var r gestures.DragGestureRecognizer
	var b *gestures.DragGestureRecognizerBase
	if spec["anyButtons"] == true {
		opts.AllowedButtonsFilter = func(int) bool { return true }
	}
	switch spec.str("type") {
	case "vertical":
		v := gestures.NewVerticalDragGestureRecognizer(opts)
		r, b = v, &v.DragGestureRecognizerBase
	case "horizontal":
		h := gestures.NewHorizontalDragGestureRecognizer(opts)
		r, b = h, &h.DragGestureRecognizerBase
	default:
		p := gestures.NewPanGestureRecognizer(opts)
		r, b = p, &p.DragGestureRecognizerBase
	}
	if spec.str("start") == "down" {
		b.DragStartBehavior = gestures.DragStartBehaviorDown
	}
	switch spec.str("strategy") {
	case "sumAllPointers":
		b.MultitouchDragStrategy = gestures.MultitouchDragStrategySumAllPointers
	case "averageBoundaryPointers":
		b.MultitouchDragStrategy = gestures.MultitouchDragStrategyAverageBoundaryPointers
	case "latestPointer":
		b.MultitouchDragStrategy = gestures.MultitouchDragStrategyLatestPointer
	}
	num := func(k string) *float64 {
		if v, ok := spec[k].(float64); ok {
			return &v
		}
		return nil
	}
	b.MinFlingDistance = num("minFlingDistance")
	b.MinFlingVelocity = num("minFlingVelocity")
	b.MaxFlingVelocity = num("maxFlingVelocity")
	if spec["onlyAcceptDragOnThreshold"] == true {
		b.OnlyAcceptDragOnThreshold = true
	}
	if slop := num("touchSlop"); slop != nil {
		b.GestureSettings = &gestures.DeviceGestureSettings{TouchSlop: slop}
	}
	switch spec.str("tracker") {
	case "ios":
		b.VelocityTrackerBuilder = func(e gestures.PointerEvent) *gestures.VelocityTracker {
			return &gestures.NewIOSScrollViewFlingVelocityTracker(e.Kind()).VelocityTracker
		}
	case "macos":
		b.VelocityTrackerBuilder = func(e gestures.PointerEvent) *gestures.VelocityTracker {
			return &gestures.NewMacOSScrollViewFlingVelocityTracker(e.Kind()).VelocityTracker
		}
	}
	b.OnDown = func(x *gestures.DragDownDetails) {
		rdlog(id + " onDown g=" + rdOff(x.GlobalPosition()) + " l=" + rdOff(x.LocalPosition()))
	}
	b.OnStart = func(x *gestures.DragStartDetails) {
		rdlog(id + " onStart g=" + rdOff(x.GlobalPosition()) + " l=" + rdOff(x.LocalPosition()) +
			" ts=" + rdTS(x.SourceTimeStamp) + " kind=" + rdKind(x.Kind))
	}
	b.OnUpdate = func(x *gestures.DragUpdateDetails) {
		rdlog(id + " onUpdate g=" + rdOff(x.GlobalPosition()) + " l=" + rdOff(x.LocalPosition()) +
			" delta=" + rdOff(x.Delta) + " primary=" + rdPD(x.PrimaryDelta) + " ts=" + rdTS(x.SourceTimeStamp) +
			" kind=" + rdKind(x.Kind))
	}
	b.OnEnd = func(x *gestures.DragEndDetails) {
		rdlog(id + " onEnd v=" + rdOff(x.Velocity.PixelsPerSecond) + " primary=" + rdPD(x.PrimaryVelocity) +
			" g=" + rdOff(x.GlobalPosition()) + " l=" + rdOff(x.LocalPosition()) +
			" global=" + dart.DoubleToString(b.GlobalDistanceMoved()) + " last=" + rdOff(b.LastPosition().Global))
	}
	b.OnCancel = func() { rdlog(id + " onCancel") }
	return r
}

func rdInt(v any) int { return int(v.(float64)) }

func TestRecognizersDragMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/recognizers_drag.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	sceneData, err := os.ReadFile("recognizers_drag_scenes.json")
	if err != nil {
		t.Fatal(err)
	}
	var scenes []struct {
		Name        string
		Recognizers []rdSpec
		Ops         [][]any
		Trace       bool
	}
	if err := json.Unmarshal(sceneData, &scenes); err != nil {
		t.Fatal(err)
	}

	rdLog = nil
	flutter_test.New(t)
	previousPrint := foundation.DebugPrint
	foundation.DebugPrint = func(message string, opts ...foundation.DebugPrintOptions) {
		rdlog("print " + rdHash.ReplaceAllString(message, "#HASH"))
	}
	defer func() {
		foundation.DebugPrint = previousPrint
		gestures.DebugPrintRecognizerCallbacksTrace = false
	}()
	binding := flutter_test.AutomatedTestWidgetsFlutterBindingEnsureInitialized()
	for _, scene := range scenes {
		rdlog("== " + scene.Name)
		gestures.DebugPrintRecognizerCallbacksTrace = scene.Trace
		flutter_test.RunTestGestureInCurrentTest(func(tester *flutter_test.GestureTester) {
			var ids []string
			recognizers := map[string]gestures.GestureRecognizer{}
			for _, spec := range scene.Recognizers {
				id := spec.str("id")
				ids = append(ids, id)
				recognizers[id] = rdMakeRecognizer(spec)
			}
			pointers := map[int]*flutter_test.TestPointer{}
			events := map[int]gestures.PointerEvent{}
			var transform *vm.Matrix4
			route := func(e gestures.PointerEvent) { tester.Route(e) }
			us := func(v any) flutter_test.TestPointerEventOptions {
				return flutter_test.TestPointerEventOptions{TimeStamp: time.Duration(rdInt(v)) * time.Microsecond}
			}
			for _, op := range scene.Ops {
				p := 0
				if len(op) > 1 {
					if f, ok := op[1].(float64); ok {
						p = int(f)
					}
				}
				if op[0] != "transform" {
					rdlog("> " + op[0].(string) + " " + strconv.Itoa(p))
				}
				switch op[0] {
				case "transform":
					var m []float64
					for _, v := range op[1].([]any) {
						m = append(m, v.(float64))
					}
					transform = vm.NewMatrix4FromList(m)
				case "down":
					buttons := gestures.KPrimaryButton
					if op[6] != nil {
						buttons = rdInt(op[6])
					}
					pointer := flutter_test.NewTestPointer(flutter_test.TestPointerOptions{Pointer: &p,
						Kind: rdKindOf(op[2].(string)), Buttons: &buttons})
					pointers[p] = pointer
					down := pointer.Down(ui.NewOffset(op[3].(float64), op[4].(float64)), us(op[5]))
					events[p] = down
					for _, id := range op[7].([]any) {
						recognizers[id.(string)].AddPointer(down.Transformed(transform).(*gestures.PointerDownEvent))
					}
				case "pzstart":
					pointer := flutter_test.NewTestPointer(flutter_test.TestPointerOptions{Pointer: &p,
						Kind: ui.PointerDeviceKindTrackpad})
					pointers[p] = pointer
					start := pointer.PanZoomStart(ui.NewOffset(op[2].(float64), op[3].(float64)), us(op[4]))
					events[p] = start
					for _, id := range op[5].([]any) {
						recognizers[id.(string)].AddPointerPanZoom(start.Transformed(transform).(*gestures.PointerPanZoomStartEvent))
					}
				case "close":
					gestures.GestureBindingInstance().GestureArena().Close(p)
				case "sweep":
					gestures.GestureBindingInstance().GestureArena().Sweep(p)
				case "route_down", "route_pz":
					route(events[p])
				case "move":
					o := us(op[4])
					if op[5] != nil {
						b := rdInt(op[5])
						o.Buttons = &b
					}
					route(pointers[p].Move(ui.NewOffset(op[2].(float64), op[3].(float64)), o))
				case "synth":
					route(gestures.NewPointerMoveEvent(gestures.PointerMoveEventOptions{Pointer: p,
						Position:  ui.NewOffset(op[2].(float64), op[3].(float64)),
						TimeStamp: time.Duration(rdInt(op[4])) * time.Microsecond, Synthesized: true}))
				case "up":
					route(pointers[p].Up(us(op[2])))
				case "cancel":
					route(pointers[p].Cancel(us(op[2])))
				case "pzupdate":
					route(pointers[p].PanZoomUpdate(*pointers[p].Location(), flutter_test.TestPointerPanZoomUpdateOptions{
						Pan: ui.NewOffset(op[2].(float64), op[3].(float64)), TimeStamp: time.Duration(rdInt(op[4])) * time.Microsecond}))
				case "pzend":
					route(pointers[p].PanZoomEnd(us(op[2])))
				case "frame":
					ts := time.Duration(rdInt(op[1])) * time.Microsecond
					binding.HandleBeginFrame(&ts)
					binding.HandleDrawFrame()
				case "elapse":
					tester.Async.Elapse(time.Duration(rdInt(op[1])) * time.Millisecond)
				default:
					t.Fatalf("unknown op %v", op[0])
				}
			}
			for _, id := range ids {
				r := recognizers[id]
				builder := foundation.NewDiagnosticPropertiesBuilder()
				r.DebugFillProperties(builder)
				var props []string
				for _, n := range builder.Properties {
					props = append(props, n.String())
				}
				rdlog(id + " " + r.DebugDescription() + " props=" + strings.Join(props, "; "))
				r.Dispose()
			}
			tester.Async.FlushMicrotasks()
		})
	}

	got := rdLog
	for i := 0; i < max(len(got), len(ref.Log)); i++ {
		var w, g string
		if i < len(ref.Log) {
			w = ref.Log[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			start := i
			for start > 0 && !strings.HasPrefix(ref.Log[start], "== ") {
				start--
			}
			t.Fatalf("line %d (scene %s):\nflutter: %s\ngo:      %s", i, ref.Log[start], w, g)
		}
	}
	t.Logf("%d lines match", len(got))
}
