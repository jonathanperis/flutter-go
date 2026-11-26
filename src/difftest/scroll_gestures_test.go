package difftest

// Go twin of flutter/test/scroll_gestures_test.dart: real pointer gestures
// on a list through the gestures library (widgets' gesture targets) on the
// Android, iOS and macOS scroll configurations, compared exactly.

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func sgPlatform(name string) services.Platform {
	switch {
	case strings.HasPrefix(name, "ios"):
		return services.IOS
	case strings.HasPrefix(name, "mac"):
		return services.MacOS
	}
	return services.Android
}

func sgList(c *ScrollController, log *[]string) Widget {
	return ListViewBuilder{ItemCount: 1000, ItemExtent: 50, Controller: c,
		ItemBuilder: func(_ *Element, i int) Widget {
			if log == nil {
				return SizedBox{}
			}
			return GestureDetector{
				OnTapAt: func(g r.Offset) {
					*log = append(*log, "tapUp "+strconv.Itoa(i)+" "+dart.DoubleToString(g.Dx)+" "+dart.DoubleToString(g.Dy))
				},
				OnTap: func() { *log = append(*log, "tap "+strconv.Itoa(i)) },
				Child: ColoredBox{Color: r.RGB(0xeeeeee)},
			}
		}}
}

func sgD(v float64) string { return dart.DoubleToString(v) }

func TestScrollGesturesMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/scroll_gestures.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var want map[string][]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	compare := func(t *testing.T, name string, got []string) {
		t.Helper()
		ref := want[name]
		if len(ref) == 0 {
			t.Fatalf("no reference for %s", name)
		}
		for i := 0; i < max(len(ref), len(got)); i++ {
			var w, g string
			if i < len(ref) {
				w = ref[i]
			}
			if i < len(got) {
				g = got[i]
			}
			if w != g {
				t.Fatalf("%s line %d: flutter %s go %s", name, i, w, g)
			}
		}
	}
	flings := map[string][2]float64{
		"android_fling_up_1500": {-300, 1500},
		"ios_fling_up_1000":     {-300, 1000},
		"ios_fling_up_75":       {-100, 75},
		"ios_fling_down_2000":   {150, 2000},
		"mac_fling_up_1500":     {-300, 1500},
		"mac_fling_up_150":      {-100, 150},
	}
	for name, f := range flings {
		t.Run(name, func(t *testing.T) {
			tester := flutter_test.New(t)
			tester.Owner.Platform = sgPlatform(name)
			start := 0.0
			if strings.Contains(name, "down") {
				start = 2000
			}
			c := NewScrollController(start)
			tester.PumpWidget(sgList(c, nil))
			s := []string{sgD(c.Offset())}
			tester.Fling(flutter_test.ByType[ListViewBuilder](), f[0], f[1])
			s = append(s, sgD(c.Offset()))
			for range 90 {
				tester.Pump(16 * time.Millisecond)
				s = append(s, sgD(c.Offset()))
			}
			compare(t, name, s)
		})
	}
	for _, name := range []string{"android_drag_pause", "ios_drag_pause", "mac_drag_pause"} {
		t.Run(name, func(t *testing.T) {
			tester := flutter_test.New(t)
			tester.Owner.Platform = sgPlatform(name)
			c := NewScrollController(500)
			tester.PumpWidget(sgList(c, nil))
			var s []string
			g := tester.StartGesture(flutter_test.ByType[ListViewBuilder]())
			moves := [][2]float64{{8, -1}, {16, -3}, {24, -4}, {104, -5}, {112, -6}, {120, -8}, {128, -12}, {136, -20}, {144, -40}}
			for _, m := range moves {
				g.MoveTo(m[1], time.Duration(m[0])*time.Millisecond)
				s = append(s, sgD(c.Offset()))
			}
			g.Up(152 * time.Millisecond)
			s = append(s, sgD(c.Offset()))
			for range 60 {
				tester.Pump(16 * time.Millisecond)
				s = append(s, sgD(c.Offset()))
			}
			compare(t, name, s)
		})
	}
	t.Run("android_tap_item", func(t *testing.T) {
		tester := flutter_test.New(t)
		var log []string
		c := NewScrollController(30)
		tester.PumpWidget(sgList(c, &log))
		tester.TapAt(400, 125.5)
		tester.Pump()
		tester.TapAt(17.25, 590)
		tester.Pump()
		log = append(log, "offset "+sgD(c.Offset()))
		compare(t, "android_tap_item", log)
	})
	t.Run("ios_tappable_drag_stop", func(t *testing.T) {
		tester := flutter_test.New(t)
		tester.Owner.Platform = services.IOS
		var log []string
		c := NewScrollController(500)
		tester.PumpWidget(sgList(c, &log))
		g := tester.StartGesture(flutter_test.ByType[ListViewBuilder]())
		moves := [][2]float64{{50, -10}, {100, -25}, {120, -25}, {130, -27}, {140, -30}, {150, -34}, {260, -34}, {270, -35}, {280, -37}, {290, -45}}
		for _, m := range moves {
			g.MoveTo(m[1], time.Duration(m[0])*time.Millisecond)
			log = append(log, sgD(c.Offset()))
		}
		g.Up(300 * time.Millisecond)
		for range 30 {
			tester.Pump(16 * time.Millisecond)
			log = append(log, sgD(c.Offset()))
		}
		compare(t, "ios_tappable_drag_stop", log)
	})
	t.Run("ios_hold_overscroll_under_outer_tap", func(t *testing.T) {
		tester := flutter_test.New(t)
		tester.Owner.Platform = services.IOS
		var log []string
		c := NewScrollController(0)
		tester.PumpWidget(GestureDetector{OnTap: func() { log = append(log, "outer tap") }, Child: sgList(c, nil)})
		c.JumpTo(-100)
		tester.Pump()
		tester.Pump(48 * time.Millisecond)
		log = append(log, "springing "+sgD(c.Offset()))
		g := tester.StartGesture(flutter_test.ByType[ListViewBuilder]())
		for range 3 {
			tester.Pump(16 * time.Millisecond)
			log = append(log, "held "+sgD(c.Offset()))
		}
		g.Up(0)
		for range 30 {
			tester.Pump(16 * time.Millisecond)
			log = append(log, "after "+sgD(c.Offset()))
		}
		compare(t, "ios_hold_overscroll_under_outer_tap", log)
	})
	t.Run("android_hold_under_outer_tap", func(t *testing.T) {
		tester := flutter_test.New(t)
		var log []string
		c := NewScrollController(0)
		tester.PumpWidget(GestureDetector{OnTap: func() { log = append(log, "outer tap") }, Child: sgList(c, nil)})
		c.Fling(3000)
		tester.Pump()
		tester.Pump(100 * time.Millisecond)
		log = append(log, "flinging "+sgD(c.Offset()))
		g := tester.StartGesture(flutter_test.ByType[ListViewBuilder]())
		for range 5 {
			tester.Pump(16 * time.Millisecond)
			log = append(log, "held "+sgD(c.Offset()))
		}
		g.Up(0)
		for range 5 {
			tester.Pump(16 * time.Millisecond)
			log = append(log, "after "+sgD(c.Offset()))
		}
		compare(t, "android_hold_under_outer_tap", log)
	})
}
