package difftest

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/animation"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func animBox(w, h float64, c r.Color, d time.Duration, curve animation.Curve) Widget {
	return Center{Child: AnimatedBox{Width: w, Height: h, Color: c, Duration: d, Curve: curve}}
}

func argb(v uint32) r.Color {
	c := r.RGB(v & 0xffffff)
	c.A = float64(v>>24) / 255
	return c
}

func sampleBox(t *flutter_test.Tester) []float64 {
	el := t.FindAll(flutter_test.ByType[AnimatedBox]())[0]
	size := el.RenderObject().Base().Size
	var color r.Color
	el.Walk(func(e *Element) {
		if cb, ok := e.Widget().(ColoredBox); ok {
			color = cb.Color
		}
	})
	return []float64{size.Width, size.Height, color.R, color.G, color.B, color.A}
}

var animScenes = map[string]func(t *testing.T) [][]float64{
	"container_retarget": func(t *testing.T) [][]float64 {
		d, c := 300*time.Millisecond, animation.EaseInOut
		tester := flutter_test.New(t)
		var s [][]float64
		tester.PumpWidget(animBox(100, 50, argb(0xffff0000), d, c))
		s = append(s, sampleBox(tester))
		tester.PumpWidget(animBox(300, 150, argb(0xff2196f3), d, c))
		s = append(s, sampleBox(tester))
		for i := 0; i < 3; i++ {
			tester.Pump(50 * time.Millisecond)
			s = append(s, sampleBox(tester))
		}
		tester.PumpWidget(animBox(50, 200, argb(0xff4caf50), d, c))
		s = append(s, sampleBox(tester))
		for i := 0; i < 8; i++ {
			tester.Pump(50 * time.Millisecond)
			s = append(s, sampleBox(tester))
		}
		return s
	},
	"container_fast_out_slow_in": func(t *testing.T) [][]float64 {
		d, c := 250*time.Millisecond, animation.FastOutSlowIn
		tester := flutter_test.New(t)
		var s [][]float64
		tester.PumpWidget(animBox(10, 10, argb(0xff000000), d, c))
		tester.PumpWidget(animBox(400, 20, argb(0x80ffffff), d, c))
		for i := 0; i < 20; i++ {
			tester.Pump(16 * time.Millisecond)
			s = append(s, sampleBox(tester))
		}
		return s
	},
	"scroll_animate_to": func(t *testing.T) [][]float64 {
		tester := flutter_test.New(t)
		ctrl := NewScrollController(0)
		tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: ctrl,
			ItemBuilder: func(*Element, int) Widget { return SizedBox{} }})
		var s [][]float64
		ctrl.AnimateTo(2000, 400*time.Millisecond, animation.EaseOut) // outside a frame
		tester.Pump()
		s = append(s, []float64{ctrl.Offset()})
		for i := 0; i < 30; i++ {
			tester.Pump(16 * time.Millisecond)
			s = append(s, []float64{ctrl.Offset()})
		}
		return s
	},
	"scroll_animate_to_after_idle": func(t *testing.T) [][]float64 {
		tester := flutter_test.New(t)
		ctrl := NewScrollController(0)
		tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: ctrl,
			ItemBuilder: func(*Element, int) Widget { return SizedBox{} }})
		tester.Pump(100 * time.Millisecond)
		var s [][]float64
		ctrl.AnimateTo(1000, 200*time.Millisecond, animation.Linear)
		for i := 0; i < 16; i++ {
			tester.Pump(16 * time.Millisecond)
			s = append(s, []float64{ctrl.Offset()})
		}
		return s
	},
}

func TestAnimationsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/animations.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][][]float64
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	for name, run := range animScenes {
		t.Run(name, func(t *testing.T) {
			ref := want[name]
			got := run(t)
			if len(ref) != len(got) {
				t.Fatalf("flutter has %d samples, go %d", len(ref), len(got))
			}
			maxErr := 0.0
			for i := range ref {
				bad := false
				for j := range ref[i] {
					e := math.Abs(ref[i][j] - got[i][j])
					maxErr = math.Max(maxErr, e)
					bad = bad || e > 1e-9
				}
				if bad {
					t.Errorf("sample %d: flutter %v\n           go      %v", i, ref[i], got[i])
				}
			}
			t.Logf("%d samples, max abs difference %.2g", len(ref), maxErr)
		})
	}
}
