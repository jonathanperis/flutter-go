// animate shows implicit animations and an animated scroll. Click the box
// to toggle it; click the header to animate the list back to the top.
//
// -taps=N toggles the box N times (one per frame) and -frames=N exits after
// N frames, logging frame intervals.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/jonathanperis/flutter-go/flutter_go/app"
	"github.com/jonathanperis/flutter-go/flutter_go/platform/metal"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/animation"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
)

type demo struct{}

func (demo) CreateState() State { return &demoState{scroll: NewScrollController(30_000)} }

type demoState struct {
	StateOf[demo]
	big    bool
	scroll *ScrollController
}

func (s *demoState) Build(*Element) Widget {
	w, h, c := 160.0, 160.0, r.RGB(0x6750a4)
	if s.big {
		w, h, c = 420.0, 260.0, r.RGB(0xe8618c)
	}
	return Row{CrossAxisAlignment: r.CrossStretch, Children: []Widget{
		Expanded{Child: ColoredBox{Color: r.RGB(0xfef7ff), Child: Center{Child: GestureDetector{
			OnTap: func() { s.SetState(func() { s.big = !s.big }) },
			Child: AnimatedBox{Width: w, Height: h, Color: c, Duration: 600 * time.Millisecond,
				Curve: animation.FastOutSlowIn,
				Child: Center{Child: Text{Text: "tap me", Style: r.TextStyle{FontSize: 20, Color: r.RGB(0xffffff)}}}},
		}}}},
		SizedBox{Width: F(260), Child: Column{CrossAxisAlignment: r.CrossStretch, Children: []Widget{
			GestureDetector{
				OnTap: func() { s.scroll.AnimateTo(0, 800*time.Millisecond, animation.EaseInOut) },
				Child: ColoredBox{Color: r.RGB(0x1d1b20), Child: Padding{Padding: r.All(16), Child: Text{
					Text: "↑ animate to top", Style: r.TextStyle{FontSize: 16, Color: r.RGB(0xffffff)}}}},
			},
			Expanded{Child: ColoredBox{Color: r.RGB(0xffffff), Child: ListViewBuilder{ItemCount: 10_000, ItemExtent: 40, Controller: s.scroll,
				ItemBuilder: func(_ *Element, i int) Widget {
					return Padding{Padding: r.Symmetric(16, 10), Child: Text{Text: fmt.Sprint("Row ", i)}}
				}}}},
		}}},
	}}
}

func main() {
	taps := flag.Int("taps", 0, "toggle the box N times")
	frames := flag.Int("frames", 0, "exit after N frames")
	flag.Parse()
	var stamps []time.Time
	var cpus []time.Duration
	err := app.Run(demo{}, app.Options{
		Title: "flutter-go: animate", Width: 900, Height: 600, MaxFrames: *frames,
		InjectTaps: *taps, InjectTapAt: [2]float64{320, 300},
		OnFrame: func(n int, cpu time.Duration) {
			stamps = append(stamps, time.Now())
			cpus = append(cpus, cpu)
		},
		Window: func(w *glfw.Window) { log.Printf("window %d", metal.WindowNumber(w.GetCocoaWindow())) },
	})
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	if len(stamps) > 10 {
		var gaps []time.Duration
		for i := 5; i < len(stamps); i++ { // skip startup frames
			gaps = append(gaps, stamps[i].Sub(stamps[i-1]))
		}
		sort.Slice(gaps, func(i, j int) bool { return gaps[i] < gaps[j] })
		sort.Slice(cpus, func(i, j int) bool { return cpus[i] < cpus[j] })
		log.Printf("%d frames; interval p50 %v p95 %v; cpu p50 %v p95 %v", len(stamps),
			gaps[len(gaps)/2], gaps[len(gaps)*95/100], cpus[len(cpus)/2], cpus[len(cpus)*95/100])
	}
}
