package difftest

// Go twin of flutter/test/recognizers_scale_test.dart: it replays the
// script the Dart side recorded (recognizers, test pointers, events,
// arena closes and sweeps, routes in gesture_tester's way) on the Go
// ScaleGestureRecognizer and compares the callback/arena log line by line,
// exactly (doubles in Dart's shortest form).

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

var rsLog []string

func rslog(s string) { rsLog = append(rsLog, s) }

var rsHash = regexp.MustCompile(`#[0-9a-f]{5}`)

func rsMask(s string) string { return rsHash.ReplaceAllString(s, "#") }
func rsD(v float64) string   { return dart.DoubleToString(v) }
func rsO(v ui.Offset) string { return "(" + rsD(v.Dx) + ", " + rsD(v.Dy) + ")" }
func rsTS(t *time.Duration) string {
	if t == nil {
		return "null"
	}
	return strconv.FormatInt(t.Microseconds(), 10)
}

type rsLogScale struct {
	gestures.ScaleGestureRecognizer
	id int
}

func (r *rsLogScale) AcceptGesture(pointer int) {
	rslog(fmt.Sprintf("r%d accept %d", r.id, pointer))
	r.ScaleGestureRecognizer.AcceptGesture(pointer)
}

func (r *rsLogScale) RejectGesture(pointer int) {
	rslog(fmt.Sprintf("r%d reject %d", r.id, pointer))
	r.ScaleGestureRecognizer.RejectGesture(pointer)
}

type rsLogEager struct {
	gestures.EagerGestureRecognizer
	id int
}

func (r *rsLogEager) AcceptGesture(pointer int) {
	rslog(fmt.Sprintf("r%d accept %d", r.id, pointer))
	r.EagerGestureRecognizer.AcceptGesture(pointer)
}

func (r *rsLogEager) RejectGesture(pointer int) {
	rslog(fmt.Sprintf("r%d reject %d", r.id, pointer))
	r.EagerGestureRecognizer.RejectGesture(pointer)
}

func rsKind(name string) ui.PointerDeviceKind {
	for _, k := range ui.PointerDeviceKindValues {
		if k.Name() == name {
			return k
		}
	}
	panic("unknown kind " + name)
}

func rsEventName(e gestures.PointerEvent) string {
	return strings.TrimPrefix(fmt.Sprintf("%T", e), "*gestures.")
}

type rsEntry struct {
	id int
	r  gestures.GestureRecognizer
}

func TestRecognizersScaleMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/recognizers_scale.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct {
		Script []map[string]any
		Log    []string
	}
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	transforms := []*vm.Matrix4{
		vm.NewMatrix4TranslationValues(-350, -250, 0),
		vm.NewMatrix4Diagonal3Values(2, 0.5, 1),
		vm.NewMatrix4FromList([]float64{0.8, 0.6, 0, 0, -0.6, 0.8, 0, 0, 0, 0, 1, 0, 10, 20, 0, 1}),
	}
	rsLog = nil
	flutter_test.RunTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognizers []rsEntry // a Dart LinkedHashMap (insertion order)
		find := func(id int) gestures.GestureRecognizer {
			for _, e := range recognizers {
				if e.id == id {
					return e.r
				}
			}
			panic("no recognizer " + strconv.Itoa(id))
		}
		pointers := map[int]*flutter_test.TestPointer{}
		lastEvent := map[int]gestures.PointerEvent{}
		for _, op := range ref.Script {
			i := func(k string) int { return int(op[k].(float64)) }
			n := func(k string) float64 { return op[k].(float64) }
			ts := func() time.Duration {
				if v, ok := op["ts"].(float64); ok {
					return time.Duration(v) * time.Microsecond
				}
				return 0
			}
			tso := func() flutter_test.TestPointerEventOptions {
				return flutter_test.TestPointerEventOptions{TimeStamp: ts()}
			}
			switch op["op"].(string) {
			case "scene":
				rslog("== " + op["name"].(string))
			case "scale":
				id := i("id")
				var o gestures.ScaleGestureRecognizerOptions
				if op["start"] == true {
					b := gestures.DragStartBehaviorStart
					o.DragStartBehavior = &b
				}
				o.TrackpadScrollCausesScale = op["tscs"] == true
				if f, ok := op["factor"].([]any); ok {
					v := ui.NewOffset(f[0].(float64), f[1].(float64))
					o.TrackpadScrollToScaleFactor = &v
				}
				if ds, ok := op["devices"].([]any); ok {
					o.SupportedDevices = map[ui.PointerDeviceKind]struct{}{}
					for _, k := range ds {
						o.SupportedDevices[rsKind(k.(string))] = struct{}{}
					}
				}
				s := &rsLogScale{id: id}
				s.Construct(s, o)
				s.OnStart = func(x *gestures.ScaleStartDetails) {
					kind := "null"
					if x.Kind != nil {
						kind = x.Kind.Name()
					}
					rslog(fmt.Sprintf("r%d start focal=%s local=%s count=%d ts=%s kind=%s | %s", id, rsO(x.FocalPoint),
						rsO(x.LocalFocalPoint), x.PointerCount, rsTS(x.SourceTimeStamp), kind, rsMask(x.String())))
				}
				s.OnUpdate = func(x *gestures.ScaleUpdateDetails) {
					rslog(fmt.Sprintf("r%d update focal=%s local=%s delta=%s scale=%s h=%s v=%s rot=%s count=%d ts=%s | %s", id,
						rsO(x.FocalPoint), rsO(x.LocalFocalPoint), rsO(x.FocalPointDelta), rsD(x.Scale), rsD(x.HorizontalScale),
						rsD(x.VerticalScale), rsD(x.Rotation), x.PointerCount, rsTS(x.SourceTimeStamp), rsMask(x.String())))
				}
				s.OnEnd = func(x *gestures.ScaleEndDetails) {
					rslog(fmt.Sprintf("r%d end v=%s sv=%s count=%d | %s", id, rsO(x.Velocity.PixelsPerSecond),
						rsD(x.ScaleVelocity), x.PointerCount, rsMask(x.String())))
				}
				recognizers = append(recognizers, rsEntry{id, s})
			case "eager":
				e := &rsLogEager{id: i("id")}
				e.Construct(e, gestures.GestureRecognizerOptions{})
				recognizers = append(recognizers, rsEntry{i("id"), e})
			case "factor":
				find(i("id")).(*rsLogScale).TrackpadScrollToScaleFactor = ui.NewOffset(n("x"), n("y"))
			case "pointer":
				p := i("p")
				pointers[p] = flutter_test.NewTestPointer(flutter_test.TestPointerOptions{Pointer: &p, Kind: rsKind(op["kind"].(string))})
			case "down":
				lastEvent[i("p")] = pointers[i("p")].Down(ui.NewOffset(n("x"), n("y")), tso())
			case "move":
				lastEvent[i("p")] = pointers[i("p")].Move(ui.NewOffset(n("x"), n("y")), tso())
			case "synthMove":
				p := pointers[i("p")]
				to := ui.NewOffset(n("x"), n("y"))
				e := gestures.NewPointerMoveEvent(gestures.PointerMoveEventOptions{TimeStamp: ts(), Kind: p.Kind,
					Device: p.Device(), Pointer: p.Pointer, Position: to, Delta: to.Sub(*p.Location()), Synthesized: true})
				lastEvent[i("p")] = e
				p.SetDownInfo(e, to, nil)
			case "up":
				lastEvent[i("p")] = pointers[i("p")].Up(tso())
			case "cancel":
				lastEvent[i("p")] = pointers[i("p")].Cancel(tso())
			case "pzStart":
				lastEvent[i("p")] = pointers[i("p")].PanZoomStart(ui.NewOffset(n("x"), n("y")), tso())
			case "pzUpdate":
				scale := n("scale")
				lastEvent[i("p")] = pointers[i("p")].PanZoomUpdate(ui.NewOffset(n("x"), n("y")),
					flutter_test.TestPointerPanZoomUpdateOptions{Pan: ui.NewOffset(n("px"), n("py")), Scale: &scale,
						Rotation: n("rot"), TimeStamp: ts()})
			case "pzEnd":
				lastEvent[i("p")] = pointers[i("p")].PanZoomEnd(tso())
			case "add":
				e := lastEvent[i("p")]
				if op["t"] != nil {
					e = e.Transformed(transforms[i("t")])
				}
				r := find(i("id"))
				if pz, ok := e.(*gestures.PointerPanZoomStartEvent); ok {
					r.AddPointerPanZoom(pz)
				} else {
					r.AddPointer(e.(*gestures.PointerDownEvent))
				}
			case "close":
				gestures.GestureBindingInstance().GestureArena().Close(i("p"))
			case "sweep":
				gestures.GestureBindingInstance().GestureArena().Sweep(i("p"))
				tester.Async.FlushMicrotasks()
			case "route":
				tester.Route(lastEvent[i("p")])
				var counts []string
				for _, e := range recognizers {
					if s, ok := e.r.(*rsLogScale); ok {
						counts = append(counts, fmt.Sprintf("r%d=%d", e.id, s.PointerCount()))
					}
				}
				rslog(fmt.Sprintf("routed %s p%d %s", rsEventName(lastEvent[i("p")]), i("p"), strings.Join(counts, " ")))
			case "disposeAll":
				for _, e := range recognizers {
					e.r.Dispose()
				}
				recognizers = nil
				pointers = map[int]*flutter_test.TestPointer{}
				lastEvent = map[int]gestures.PointerEvent{}
			default:
				t.Fatalf("unknown op %v", op["op"])
			}
		}
	})
	for i := 0; i < max(len(rsLog), len(ref.Log)); i++ {
		var w, g string
		if i < len(ref.Log) {
			w = ref.Log[i]
		}
		if i < len(rsLog) {
			g = rsLog[i]
		}
		if w != g {
			t.Fatalf("line %d:\nflutter: %s\ngo:      %s", i, w, g)
		}
	}
	t.Logf("%d script ops, %d log lines identical", len(ref.Script), len(ref.Log))
}
