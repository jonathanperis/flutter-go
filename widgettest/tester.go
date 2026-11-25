// Package widgettest is a headless harness in the spirit of flutter_test's
// WidgetTester: pump widgets, find them, tap them. It uses the fixed test
// font and a recording canvas, so no GPU is needed.
package widgettest

import (
	"testing"

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

// Pump draws a frame (rebuilding anything dirty).
func (t *Tester) Pump() {
	t.Canvas = &r.Recorder{}
	t.Owner.DrawFrame(t.Size, t.Canvas)
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
