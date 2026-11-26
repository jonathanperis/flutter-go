package difftest

// Go twin of flutter/test/velocity_tracker_test.dart: the same trackers,
// motions, stopped-pointer timings, Velocity operations and least-squares
// fits, compared exactly. The fake sampling clock stands for testWidgets'
// (tester.pump advances it).

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type vtSample struct {
	us   int
	x, y float64
}

type vtFakeClock struct{ now time.Duration }

func (c *vtFakeClock) Now() time.Time { return time.Unix(0, 0).Add(c.now) }
func (c *vtFakeClock) Stopwatch() *dart.Stopwatch {
	return dart.NewStopwatchWithClock(func() time.Duration { return c.now })
}

func vtOff(o ui.Offset) string { return ds(o.Dx) + " " + ds(o.Dy) }

func vtEst(e *gestures.VelocityEstimate) string {
	if e == nil {
		return "null"
	}
	return vtOff(e.PixelsPerSecond) + " " + ds(e.Confidence) + " " + strconv.Itoa(int(e.Duration/time.Microsecond)) +
		" " + vtOff(e.Offset) + " | " + e.String()
}

// vtTracker is a tracker with its Dart runtime type name.
type vtTracker struct {
	name string
	*gestures.VelocityTracker
}

func vtTrackers() []vtTracker {
	return []vtTracker{
		{"VelocityTracker", gestures.NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch)},
		{"IOSScrollViewFlingVelocityTracker", &gestures.NewIOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindTouch).VelocityTracker},
		{"MacOSScrollViewFlingVelocityTracker", &gestures.NewMacOSScrollViewFlingVelocityTracker(ui.PointerDeviceKindMouse).VelocityTracker},
	}
}

type vtMotion struct {
	name    string
	samples []vtSample
}

func vtGen(n int, t func(int) int, x, y func(int) float64) []vtSample {
	out := make([]vtSample, n)
	for i := range out {
		out[i] = vtSample{t(i), x(i), y(i)}
	}
	return out
}

func vtMotions() []vtMotion {
	fl := func(i int) float64 { return float64(i) }
	var out []vtMotion
	add := func(name string, s []vtSample) { out = append(out, vtMotion{name, s}) }
	add("single", []vtSample{{1000, 10.0, 20.0}})
	add("two", []vtSample{{1000, 10.0, 20.0}, {9000, 13.5, 27.25}})
	add("three linear", vtGen(3, func(i int) int { return 1000 + 8000*i }, func(i int) float64 { return 10.0 * fl(i) },
		func(i int) float64 { return -5.0 * fl(i) }))
	add("uniform 8ms", vtGen(12, func(i int) int { return 16000 + 8000*i }, func(i int) float64 { return 3.0 * fl(i) },
		func(i int) float64 { return 100 + 7.5*fl(i) }))
	add("accelerating", vtGen(15, func(i int) int { return 5000 + 7000*i }, func(i int) float64 { return float64(0.5*fl(i)) * fl(i) },
		func(i int) float64 { return float64(2.0*fl(i))*fl(i) - fl(i) }))
	add("decelerating", vtGen(15, func(i int) int { return 7000 * i },
		func(i int) float64 { return 100 - float64(float64((14-i)*(14-i))*0.3) },
		func(i int) float64 { return 50.0 * math.Sqrt(fl(i)) }))
	add("irregular times", vtGen(14, func(i int) int { return 6000*i + (i*i*731)%5000 },
		func(i int) float64 { return float64(1.7*fl(i)) + dart.Sin(fl(i)) },
		func(i int) float64 { return 400 - float64(9.1*fl(i)) }))
	first5 := vtGen(5, func(i int) int { return 8000 * i }, func(i int) float64 { return 4.0 * fl(i) },
		func(i int) float64 { return 2.0 * fl(i) })
	add("gap over 40ms", append(append([]vtSample{}, first5...), vtGen(5, func(i int) int { return 32000 + 41000 + 8000*i },
		func(i int) float64 { return 20 + float64(6.0*fl(i)) }, func(i int) float64 { return 10 + float64(1.0*fl(i)) })...))
	add("gap exactly 40ms", append(append([]vtSample{}, first5...), vtGen(5, func(i int) int { return 32000 + 40000 + 8000*i },
		func(i int) float64 { return 20 + float64(6.0*fl(i)) }, func(i int) float64 { return 10 + float64(1.0*fl(i)) })...))
	add("past horizon", vtGen(20, func(i int) int { return 9000 * i }, func(i int) float64 { return 3.0 * fl(i) },
		func(i int) float64 { return -2.0 * fl(i) }))
	add("horizon edge", vtGen(11, func(i int) int { return 10000 * i }, func(i int) float64 { return float64(1.0*fl(i)) * fl(i) },
		func(i int) float64 { return 3.0 * fl(i) }))
	add("wrap 25", vtGen(25, func(i int) int { return 4000 * i }, func(i int) float64 { return float64(2.5*fl(i)) + fl(i%3) },
		func(i int) float64 { return float64(7.0*fl(i)) - fl(i%4) }))
	add("wrap 47", vtGen(47, func(i int) int { return 3000 * i }, func(i int) float64 { return 100 * dart.Cos(fl(i)/7) },
		func(i int) float64 { return 100 * dart.Sin(fl(i)/5) }))
	add("equal times", []vtSample{{1000, 1.0, 1.0}, {1000, 2.0, 3.0}, {9000, 4.0, 4.0}, {9000, 6.0, 9.0}, {17000, 7.0, 10.0}})
	add("stationary", vtGen(10, func(i int) int { return 8000 * i }, func(int) float64 { return 50.0 },
		func(int) float64 { return 60.0 }))
	add("vertical only", vtGen(10, func(i int) int { return 8000 * i }, func(int) float64 { return 0.0 },
		func(i int) float64 { return -13.25 * fl(i) }))
	add("large values", vtGen(10, func(i int) int { return 1000000000 + 8000*i }, func(i int) float64 { return 1e6 + float64(123.456*fl(i)) },
		func(i int) float64 { return -1e5 - float64(float64(0.001*fl(i))*fl(i)) }))
	add("tiny steps", vtGen(10, func(i int) int { return 8000 * i }, func(i int) float64 { return 1e-7 * fl(i) },
		func(i int) float64 { return float64(3e-9*fl(i)) * fl(i) }))
	add("fast fling", vtGen(8, func(i int) int { return 4000 * i }, func(int) float64 { return 0.0 },
		func(i int) float64 { return 300 - float64(42.0*fl(i)) }))
	add("micro times", vtGen(9, func(i int) int { return 1 + i*1337 }, func(i int) float64 { return fl(i) * 0.75 },
		func(i int) float64 { return fl(i) * -1.25 }))
	return out
}

// vtRecorded splits the recorded events of the gestures velocity tests
// (velocity_tracker_data.dart, as the Go gestures tests have them) per
// pointer.
func vtRecorded() []vtMotion {
	var out []vtMotion
	for _, set := range []struct {
		name string
		data []vtDatum
	}{{"velocityEventData", vtVelocityEventData}, {"interruptedVelocityEventData", vtInterruptedVelocityEventData}} {
		var current []vtSample
		n := 0
		for _, d := range set.data {
			switch d.kind {
			case "down":
				current = []vtSample{{d.ms * 1000, d.x, d.y}}
			case "move":
				current = append(current, vtSample{d.ms * 1000, d.x, d.y})
			case "up":
				out = append(out, vtMotion{set.name + " " + strconv.Itoa(n), current})
				n++
			}
		}
	}
	return out
}

func runVelocityTrackerScene() []string {
	var lines []string
	log := func(s string) { lines = append(lines, s) }
	clock := &vtFakeClock{}
	restore := gestures.DebugSetSamplingClock(func() gestures.SamplingClock { return clock })
	defer restore()

	for _, m := range append(vtRecorded(), vtMotions()...) {
		log("motion " + m.name + " (" + strconv.Itoa(len(m.samples)) + ")")
		for _, tracker := range vtTrackers() {
			var ests []string
			for _, s := range m.samples {
				tracker.AddPosition(time.Duration(s.us)*time.Microsecond, ui.NewOffset(s.x, s.y))
				ests = append(ests, vtEst(tracker.GetVelocityEstimate()))
			}
			log("  " + tracker.name)
			for _, e := range ests {
				log("    " + e)
			}
			v := tracker.GetVelocity()
			log("    velocity " + vtOff(v.PixelsPerSecond) + " " + v.String())
			for _, c := range [][2]float64{{0, 8000}, {50, 8000}, {0, 100}, {500, 600}, {0, 0}} {
				log("    clamp " + ds(c[0]) + " " + ds(c[1]) + " " + vtOff(v.ClampMagnitude(c[0], c[1]).PixelsPerSecond))
			}
		}
	}

	for _, waitUs := range []int{0, 39999, 40000, 40999, 41000, 41001, 100000} {
		ts := vtTrackers()
		for _, tracker := range ts {
			tracker.AddPosition(0, ui.OffsetZero)
			tracker.AddPosition(8*time.Millisecond, ui.NewOffset(5, 10))
			tracker.AddPosition(16*time.Millisecond, ui.NewOffset(12, 20))
		}
		clock.now += time.Duration(waitUs) * time.Microsecond // tester.pump
		for _, tracker := range ts {
			log("stopped " + strconv.Itoa(waitUs) + " " + tracker.name + " " + vtEst(tracker.GetVelocityEstimate()))
		}
		for _, tracker := range ts {
			tracker.AddPosition(24*time.Millisecond, ui.NewOffset(20, 31))
			log("restarted " + strconv.Itoa(waitUs) + " " + tracker.name + " " + vtEst(tracker.GetVelocityEstimate()))
		}
	}

	log("empty " + vtEst(gestures.NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch).GetVelocityEstimate()) + " " +
		gestures.NewVelocityTrackerWithKind(ui.PointerDeviceKindTouch).GetVelocity().String())

	a := gestures.Velocity{PixelsPerSecond: ui.NewOffset(3.25, -4.75)}
	b := gestures.Velocity{PixelsPerSecond: ui.NewOffset(-0.05, 0.15)}
	log("ops " + vtOff(a.Neg().PixelsPerSecond) + " " + vtOff(a.Add(b).PixelsPerSecond) + " " + vtOff(a.Sub(b).PixelsPerSecond) +
		" " + strconv.FormatBool(a == gestures.Velocity{PixelsPerSecond: ui.NewOffset(3.25, -4.75)}) + " " +
		strconv.FormatBool(a == b) + " " + a.String() + " " + b.String() + " " + gestures.VelocityZero.String())
	for _, v := range []gestures.Velocity{a, b, {PixelsPerSecond: ui.NewOffset(3000, 4000)}, {PixelsPerSecond: ui.NewOffset(1e-3, -2e-3)},
		{PixelsPerSecond: ui.NewOffset(0.05, -0.05)}, {PixelsPerSecond: ui.NewOffset(-0.25, 0.35)}} {
		for _, c := range [][2]float64{{0, 5}, {5, 10}, {1, 1}, {0.001, 0.002}, {6000, 9000}} {
			log("clamp " + vtOff(v.PixelsPerSecond) + " " + ds(c[0]) + " " + ds(c[1]) + " " + vtOff(v.ClampMagnitude(c[0], c[1]).PixelsPerSecond))
		}
	}
	log(gestures.VelocityEstimate{PixelsPerSecond: ui.NewOffset(1.25, -3.35), Confidence: 0.45,
		Duration: 12345 * time.Microsecond, Offset: ui.NewOffset(7.5, 8.25)}.String())

	xs := make([]float64, 9)
	for i := range xs {
		xs[i] = float64(-float64(i)*7.5) + float64(float64(i%2)*0.3)
	}
	fill := func(f func(i int) float64) []float64 {
		out := make([]float64, 9)
		for i := range out {
			out[i] = f(i)
		}
		return out
	}
	ones := fill(func(int) float64 { return 1.0 })
	sets := []struct {
		name    string
		x, y, w []float64
	}{
		{"flat", xs, ones, ones},
		{"line", xs, fill(func(i int) float64 { return float64(2.5*xs[i]) - 3 }), ones},
		{"noisy", xs, fill(func(i int) float64 {
			return float64(float64(0.3*xs[i])*xs[i]) + float64(dart.Sin(float64(i)*1.3)*4)
		}), ones},
		{"weighted", xs, fill(func(i int) float64 { return dart.Cos(float64(i)/2) * 10 }),
			fill(func(i int) float64 { return 0.5 + float64(i)/3 })},
		{"two points", []float64{0, 1}, []float64{3, 5}, []float64{1, 1}},
		{"same x", []float64{2, 2, 2, 2}, []float64{1, 2, 3, 4}, []float64{1, 1, 1, 1}},
		{"zero weights", []float64{0, 1, 2}, []float64{1, 2, 3}, []float64{0, 0, 0}},
		{"one point", []float64{4}, []float64{9}, []float64{2}},
		{"cubic", []float64{-3, -2, -1, 0, 1, 2, 3, 4}, []float64{-27, -8, -1, 0, 1, 8, 27, 64.5}, []float64{1, 1, 1, 1, 1, 1, 1, 1}},
	}
	for _, s := range sets {
		for degree := 0; degree <= 3; degree++ {
			fit := gestures.NewLeastSquaresSolver(s.x, s.y, s.w).Solve(degree)
			text := "null"
			if fit != nil {
				parts := make([]string, len(fit.Coefficients))
				for i, c := range fit.Coefficients {
					parts[i] = ds(c)
				}
				text = strings.Join(parts, " ") + " " + ds(fit.Confidence) + " " + fit.String()
			}
			log("lsq " + s.name + " " + strconv.Itoa(degree) + " " + text)
		}
	}
	return lines
}

func TestVelocityTrackerMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/velocity_tracker.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	compareLines(t, ref.Log, runVelocityTrackerScene())
}
