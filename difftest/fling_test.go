package difftest

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

var flingScenes = map[string][2]float64{
	"fling_down":            {0, 1500},
	"fling_up":              {3000, -4000},
	"fling_hits_end":        {48000, 5000},
	"fling_below_tolerance": {1000, 5},
}

func runFling(t *testing.T, start, velocity float64) [][2]float64 {
	tester := widgettest.New(t)
	c := NewScrollController(0)
	tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(*Element, int) Widget { return SizedBox{} }})
	c.JumpTo(start)
	tester.Pump()
	c.Fling(velocity)
	var s [][2]float64
	sample := func() [2]float64 {
		sched := 0.0
		if tester.Owner.Scheduler().HasScheduledFrame() {
			sched = 1
		}
		return [2]float64{c.Offset(), sched}
	}
	tester.Pump()
	s = append(s, sample())
	for i := 0; i < 70; i++ {
		tester.Pump(16 * time.Millisecond)
		s = append(s, sample())
	}
	return s
}

func TestFlingMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/fling.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][][2]float64
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	for name, sc := range flingScenes {
		t.Run(name, func(t *testing.T) {
			ref, got := want[name], runFling(t, sc[0], sc[1])
			maxErr := 0.0
			for i := range ref {
				e := math.Abs(ref[i][0] - got[i][0])
				maxErr = math.Max(maxErr, e)
				// Frame scheduling: Go must never keep animating when Flutter
				// has stopped. Flutter may run one extra frame when a fling
				// ends (it repaints to lift the IgnorePointer it uses during
				// scroll activities), so allow exactly that.
				extraFlutterFrame := ref[i][1] == 1 && got[i][1] == 0 && i > 0 && got[i-1][1] == 1
				if e > 1e-6 || (ref[i][1] != got[i][1] && !extraFlutterFrame) {
					t.Errorf("frame %d: flutter offset %.6f scheduled %v; go %.6f scheduled %v", i, ref[i][0], ref[i][1], got[i][0], got[i][1])
				}
			}
			n := len(got) - 1
			t.Logf("start %v → end %.3f (flutter %.3f), max abs difference %.2g", ref[0][0], got[n][0], ref[n][0], maxErr)
		})
	}
}
