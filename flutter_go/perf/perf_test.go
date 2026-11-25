// Package perf measures frame cost against the 16.6 ms (60 Hz) budget.
//
//	PERF=1 go test ./perf -v -run Budget   # per-frame percentiles + GC pauses
//	go test ./perf -bench . -run '^$'      # Go benchmarks
package perf

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/flutter_go/app"
	"github.com/jonathanperis/flutter-go/flutter_go/impeller"
	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
)

const rows, cols = 100, 100

// grid is a 100×100 grid of colored cells plus a text label per row.
// Each tick changes every cell's color; in layout mode it also changes sizes.
type grid struct{ layoutMode bool }

func (g grid) CreateState() State { return &gridState{} }

type gridState struct {
	StateOf[grid]
	tick int
}

func (s *gridState) Build(*Element) Widget {
	side := 5.0
	if s.Widget().layoutMode && s.tick%2 == 1 {
		side = 4
	}
	rowWidgets := make([]Widget, rows)
	for i := range rowWidgets {
		cells := make([]Widget, 0, cols+1)
		cells = append(cells, SizedBox{Width: F(40), Child: Text{Text: fmt.Sprint(i), Style: r.TextStyle{FontSize: 4}}})
		for j := 0; j < cols; j++ {
			c := r.Color{R: float64((i+j+s.tick)%256) / 255, G: 0.5, B: 0.5, A: 1}
			cells = append(cells, ColoredBox{Color: c, Child: SizedBox{Width: F(side), Height: F(side)}})
		}
		rowWidgets[i] = Row{MainAxisSize: r.MainMin, Children: cells}
	}
	return Column{CrossAxisAlignment: r.CrossStart, Children: rowWidgets}
}

type nopCanvas struct{}

func (nopCanvas) Save()                               {}
func (nopCanvas) Restore()                            {}
func (nopCanvas) Translate(float64, float64)          {}
func (nopCanvas) ClipRect(r.Rect)                     {}
func (nopCanvas) DrawRect(r.Rect, r.Color)            {}
func (nopCanvas) DrawRRect(r.Rect, float64, r.Color)  {}
func (nopCanvas) DrawParagraph(r.Paragraph, r.Offset) {}
func (nopCanvas) DrawPicture(r.Picture, r.Offset)     {}
func (nopCanvas) NewRecording() r.Recording           { return nopRecording{} }

type nopRecording struct{}
type nopPicture struct{}

func (nopRecording) Canvas() r.Canvas { return nopCanvas{} }
func (nopRecording) End() r.Picture   { return nopPicture{} }
func (nopPicture) Release()           {}

type harness struct {
	owner *BuildOwner
	state *gridState
	tc    *impeller.TypographyContext
	paint *impeller.Paint
}

func newHarness(layoutMode bool) *harness {
	h := &harness{owner: &BuildOwner{}, tc: impeller.NewTypographyContext(), paint: impeller.NewPaint()}
	r.SetTextEngine(app.NewTextEngine(h.tc))
	h.owner.RunApp(grid{layoutMode})
	h.owner.Root().Walk(func(e *Element) {
		if s, ok := StateOfElement(e).(*gridState); ok {
			h.state = s
		}
	})
	return h
}

func (h *harness) release() { r.Dispose(h.owner.RenderRoot()); h.paint.Release(); h.tc.Release() }

func (h *harness) elements() int {
	n := 0
	h.owner.Root().Walk(func(*Element) { n++ })
	return n
}

// frame: setState → build → layout → paint into a real Impeller display list
// (or a no-op canvas, to separate framework cost from cgo cost).
func (h *harness) frame(useImpeller bool) {
	h.state.SetState(func() { h.state.tick++ })
	size := r.Size{W: 800, H: 600}
	if !useImpeller {
		h.owner.DrawFrame(size, nopCanvas{})
		return
	}
	b := impeller.NewDisplayListBuilder()
	h.owner.DrawFrame(size, app.NewCanvas(b, h.paint))
	b.Build().Release()
	b.Release()
}

func BenchmarkFrame(b *testing.B) {
	for _, layoutMode := range []bool{false, true} {
		for _, useImpeller := range []bool{false, true} {
			name := fmt.Sprintf("layout=%v/impeller=%v", layoutMode, useImpeller)
			b.Run(name, func(b *testing.B) {
				h := newHarness(layoutMode)
				defer h.release()
				h.frame(useImpeller)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					h.frame(useImpeller)
				}
			})
		}
	}
}

// TestFrameBudget renders 300 frames per mode and reports percentiles and GC
// pauses. It fails only if p99 exceeds the 60 Hz budget.
func TestFrameBudget(t *testing.T) {
	if os.Getenv("PERF") == "" {
		t.Skip("timing-sensitive; run alone with PERF=1")
	}
	for _, layoutMode := range []bool{false, true} {
		h := newHarness(layoutMode)
		h.frame(true)
		t.Logf("layout=%v: %d elements per frame", layoutMode, h.elements())

		runtime.GC()
		var before runtime.MemStats
		runtime.ReadMemStats(&before)
		times := make([]time.Duration, 300)
		for i := range times {
			start := time.Now()
			h.frame(true)
			times[i] = time.Since(start)
		}
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		h.release()

		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
		pct := func(p float64) time.Duration { return times[int(p*float64(len(times)-1))] }
		var gcStats debug.GCStats
		debug.ReadGCStats(&gcStats)
		maxPause := time.Duration(0)
		for i := before.NumGC; i < after.NumGC; i++ {
			if p := time.Duration(after.PauseNs[(i+255)%256]); p > maxPause {
				maxPause = p
			}
		}
		t.Logf("layout=%v: p50 %v  p90 %v  p99 %v  max %v | GCs %d, max pause %v, %.1f MB alloc/frame",
			layoutMode, pct(.5), pct(.9), pct(.99), times[len(times)-1],
			after.NumGC-before.NumGC, maxPause,
			float64(after.TotalAlloc-before.TotalAlloc)/float64(len(times))/1e6)
		if p99 := pct(.99); p99 > 16600*time.Microsecond {
			t.Errorf("p99 %v exceeds the 16.6 ms budget", p99)
		}
	}
}

// BenchmarkScrollFrame scrolls a 1,000,000-item list by 7px per frame. The
// cost should depend on the number of visible items, not the item count.
func BenchmarkScrollFrame(b *testing.B) {
	for _, count := range []int{1_000, 1_000_000} {
		b.Run(fmt.Sprintf("items=%d", count), func(b *testing.B) {
			r.SetTextEngine(r.FixedTextEngine{})
			c := NewScrollController(0)
			owner := &BuildOwner{}
			owner.RunApp(ListViewBuilder{ItemCount: count, ItemExtent: 56, Controller: c,
				ItemBuilder: func(_ *Element, i int) Widget {
					return ColoredBox{Color: r.RGB(0xffffff), Child: Text{Text: fmt.Sprint("Row ", i)}}
				}})
			size := r.Size{W: 800, H: 600}
			owner.DrawFrame(size, nopCanvas{})
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				off := float64(i*7) - float64(i*7/50_000)*50_000 // wrap within the first 50k px
				c.JumpTo(off)
				owner.DrawFrame(size, nopCanvas{})
			}
		})
	}
}
