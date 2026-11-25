package difftest

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/textedit"
	. "github.com/jonathanperis/flutter-go/widgets"
	"github.com/jonathanperis/flutter-go/widgettest"
)

func bounceTester(t *testing.T, platform textedit.Platform) (*widgettest.Tester, *ScrollController) {
	tester := widgettest.New(t)
	tester.Owner.Platform = platform
	c := NewScrollController(0)
	tester.PumpWidget(ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(*Element, int) Widget { return SizedBox{} }})
	return tester, c
}

func frames(tester *widgettest.Tester, c *ScrollController, s []float64, n int) []float64 {
	for i := 0; i < n; i++ {
		tester.Pump(16 * time.Millisecond)
		s = append(s, c.Offset())
	}
	return s
}

func plat(p float64) textedit.Platform {
	if p == 0 {
		return textedit.IOS
	}
	return textedit.MacOS
}

var bounceScenes = map[string]func(t *testing.T) []float64{}

func init() {
	for name, sc := range map[string][3]float64{
		"ios_fling_within":          {0, 0, 1500},
		"ios_fling_overshoot_end":   {0, 48000, 5000},
		"ios_fling_overshoot_start": {0, 300, -4000},
		"mac_fling_within":          {1, 0, 3000},
		"mac_fling_overshoot_end":   {1, 48000, 6000},
	} {
		sc := sc
		bounceScenes[name] = func(t *testing.T) []float64 {
			tester, c := bounceTester(t, plat(sc[0]))
			c.JumpTo(sc[1])
			tester.Pump()
			c.Fling(sc[2])
			tester.Pump()
			return frames(tester, c, []float64{c.Offset()}, 90)
		}
	}
	bounceScenes["ios_jump_out_of_range"] = func(t *testing.T) []float64 {
		tester, c := bounceTester(t, textedit.IOS)
		c.JumpTo(-150)
		s := []float64{c.Offset()}
		tester.Pump()
		s = append(s, c.Offset())
		return frames(tester, c, s, 40)
	}
	for name, p := range map[string]float64{"ios_drag_overscroll_top": 0, "mac_drag_overscroll_top": 1} {
		p := p
		bounceScenes[name] = func(t *testing.T) []float64 {
			tester, c := bounceTester(t, plat(p))
			g := tester.StartGesture(widgettest.ByType[ListViewBuilder]())
			var s []float64
			for i := 1; i <= 20; i++ {
				g.MoveTo(10*float64(i), time.Duration(8*i)*time.Millisecond)
				s = append(s, c.Offset())
			}
			tester.Pump(100 * time.Millisecond)
			g.Up(260 * time.Millisecond)
			tester.Pump()
			s = append(s, c.Offset())
			return frames(tester, c, s, 40)
		}
	}
}

func TestBouncingMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/bouncing.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][]float64
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	for name, run := range bounceScenes {
		t.Run(name, func(t *testing.T) {
			ref, got := want[name], run(t)
			if len(ref) != len(got) {
				t.Fatalf("flutter %d samples, go %d", len(ref), len(got))
			}
			maxErr := 0.0
			for i := range ref {
				e := math.Abs(ref[i] - got[i])
				maxErr = math.Max(maxErr, e)
				if e > 1e-6 {
					t.Errorf("sample %d: flutter %.6f go %.6f", i, ref[i], got[i])
					if i > 4 {
						break
					}
				}
			}
			t.Logf("max abs difference %.2g", maxErr)
		})
	}
}
