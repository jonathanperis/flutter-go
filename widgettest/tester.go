// Package widgettest is a headless harness in the spirit of flutter_test's
// WidgetTester: pump widgets, find them, tap them. It uses the fixed test
// font and a recording canvas, so no GPU is needed.
package widgettest

import (
	"testing"
	"time"

	r "github.com/jonathanperis/flutter-go/rendering"
	"github.com/jonathanperis/flutter-go/widgets"
)

type Tester struct {
	t      testing.TB
	Owner  *widgets.BuildOwner
	router *widgets.PointerRouter
	Size   r.Size
	// Canvas holds the last frame's paint commands.
	Canvas *r.Recorder
	// Now is the fake clock (frame timestamp).
	Now time.Duration
}

// New creates a tester with Flutter's default test surface (800×600).
func New(t testing.TB) *Tester {
	r.SetTextEngine(r.FixedTextEngine{})
	o := &widgets.BuildOwner{}
	return &Tester{t: t, Owner: o, router: widgets.NewPointerRouter(o), Size: r.Size{W: 800, H: 600}}
}

// PumpWidget mounts w as the root and draws a frame.
func (t *Tester) PumpWidget(w widgets.Widget) {
	t.Owner.RunApp(w)
	t.Pump()
}

// Pump advances the fake clock by d (optional, default 0) and draws a frame,
// like flutter_test's tester.pump(duration).
func (t *Tester) Pump(d ...time.Duration) {
	t.Canvas = &r.Recorder{}
	if len(d) > 0 {
		t.Now += d[0]
	}
	t.Owner.DrawFrameAt(t.Now, t.Size, t.Canvas)
}

// PumpAndSettle pumps 16ms frames until no animation is scheduled, like
// flutter_test's pumpAndSettle. It fails after 10 seconds of fake time.
func (t *Tester) PumpAndSettle() {
	t.t.Helper()
	t.Pump()
	for i := 0; t.Owner.Scheduler().HasScheduledFrame() || t.Owner.NeedsBuild(); i++ {
		if i > 625 {
			t.t.Fatal("PumpAndSettle timed out")
		}
		t.Pump(16 * time.Millisecond)
	}
}

// Finder selects elements.
type Finder func(e *widgets.Element) bool

func Text(s string) Finder {
	return func(e *widgets.Element) bool {
		w, ok := e.Widget().(widgets.Text)
		return ok && w.Text == s
	}
}

func ByType[W widgets.Widget]() Finder {
	return func(e *widgets.Element) bool { _, ok := e.Widget().(W); return ok }
}

// FindAll returns matching elements in tree order.
func (t *Tester) FindAll(f Finder) []*widgets.Element {
	var out []*widgets.Element
	t.Owner.Root().Walk(func(e *widgets.Element) {
		if f(e) {
			out = append(out, e)
		}
	})
	return out
}

// Count returns how many elements match.
func (t *Tester) Count(f Finder) int { return len(t.FindAll(f)) }

// Rect returns the global rect of the single element matched by f.
func (t *Tester) Rect(f Finder) r.Rect {
	t.t.Helper()
	ro := t.one(f).RenderObject()
	return r.RectFrom(r.GlobalOffset(ro), ro.Base().Size)
}

// Tap taps the center of the single element matched by f.
func (t *Tester) Tap(f Finder) {
	t.t.Helper()
	rc := t.Rect(f)
	c := r.Offset{X: rc.X + rc.W/2, Y: rc.Y + rc.H/2}
	t.router.PointerDown(c)
	t.router.PointerUp(c)
}

func (t *Tester) one(f Finder) *widgets.Element {
	t.t.Helper()
	all := t.FindAll(f)
	if len(all) != 1 {
		t.t.Fatalf("expected exactly one match, found %d", len(all))
	}
	return all[0]
}

// Drag presses at the center of the element matched by f, moves by dy in
// steps of at most 10px, and releases.
func (t *Tester) Drag(f Finder, dy float64) {
	t.t.Helper()
	rc := t.Rect(f)
	p := r.Offset{X: rc.X + rc.W/2, Y: rc.Y + rc.H/2}
	t.router.PointerDown(p)
	for moved := 0.0; moved != dy; {
		step := dy - moved
		if step > 10 {
			step = 10
		} else if step < -10 {
			step = -10
		}
		moved += step
		t.router.PointerMove(r.Offset{X: p.X, Y: p.Y + moved})
	}
	t.router.PointerUp(r.Offset{X: p.X, Y: p.Y + dy})
}

// Scroll sends a mouse-wheel delta at the center of the element matched by f.
func (t *Tester) Scroll(f Finder, dy float64) {
	t.t.Helper()
	rc := t.Rect(f)
	t.router.Scroll(r.Offset{X: rc.X + rc.W/2, Y: rc.Y + rc.H/2}, dy)
}
