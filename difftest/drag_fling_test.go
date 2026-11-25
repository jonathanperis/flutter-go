package difftest

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	r "github.com/jonathanperis/flutter-go/rendering"
	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

var dragFlingScenes = map[string][3]float64{
	"drag_fling_up_1000":      {0, -300, 1000},
	"drag_fling_up_3000":      {0, -200, 3000},
	"drag_fling_down_2000":    {2000, 150, 2000},
	"drag_fling_slow_100":     {0, -100, 100},
	"drag_too_slow_40":        {0, -60, 40},
	"drag_fling_max_velocity": {0, -400, 20000},
	// Items with tap handlers: the drag competes with a tap in the arena.
	"tappable_drag_fling_up_1000": {0, -300, 1000},
}

func TestAcceleratingDragMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/drag_fling.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][]float64
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	tester := widgettest.New(t)
	c := NewScrollController(0)
	tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(*Element, int) Widget { return SizedBox{} }})
	var moves []widgettest.TimedMove
	for i := 1; i <= 15; i++ {
		moves = append(moves, widgettest.TimedMove{T: time.Duration(8*i) * time.Millisecond, DY: -1.5 * float64(i*i)})
	}
	tester.DragTimed(widgettest.ByType[ListViewBuilder](), moves)
	got := []float64{c.Offset()}
	for i := 0; i < 60; i++ {
		tester.Pump(16 * time.Millisecond)
		got = append(got, c.Offset())
	}
	ref := want["accelerating_drag"]
	for i := range ref {
		if math.Abs(ref[i]-got[i]) > 1e-6 {
			t.Fatalf("frame %d: flutter %.6f go %.6f", i, ref[i], got[i])
		}
	}
	t.Logf("release %.3f → end %.3f (flutter %.3f)", got[0], got[len(got)-1], ref[len(ref)-1])
}

func TestDragFlingMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/drag_fling.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][]float64
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	for name, sc := range dragFlingScenes {
		t.Run(name, func(t *testing.T) {
			tester := widgettest.New(t)
			c := NewScrollController(sc[0])
			tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: c,
				ItemBuilder: func(*Element, int) Widget {
					if strings.HasPrefix(name, "tappable") {
						return GestureDetector{OnTap: func() {}, Child: ColoredBox{Color: r.RGB(0xeeeeee)}}
					}
					return SizedBox{}
				}})
			got := []float64{c.Offset()}
			tester.Fling(widgettest.ByType[ListViewBuilder](), sc[1], sc[2])
			got = append(got, c.Offset())
			for i := 0; i < 60; i++ {
				tester.Pump(16 * time.Millisecond)
				got = append(got, c.Offset())
			}
			ref := want[name]
			maxErr := 0.0
			for i := range ref {
				e := math.Abs(ref[i] - got[i])
				maxErr = math.Max(maxErr, e)
				if e > 1e-6 {
					t.Errorf("frame %d: flutter %.6f go %.6f", i, ref[i], got[i])
					if i > 3 {
						break
					}
				}
			}
			t.Logf("release %.3f → end %.3f (flutter %.3f), max diff %.2g", got[1], got[len(got)-1], ref[len(ref)-1], maxErr)
		})
	}
}
