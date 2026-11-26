// Package widgettest is a headless harness in the spirit of flutter_test's
// WidgetTester: pump widgets, find them, tap them. It uses the fixed test
// font and a recording canvas, so no GPU is needed.
package flutter_test

import (
	"math"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/widgets"
)

type Tester struct {
	t      testing.TB
	Owner  *widgets.BuildOwner
	router *widgets.PointerRouter
	Size   r.Size
	// Canvas holds the last frame's paint commands.
	Canvas *r.Recorder
	// eventTime, if not nil, is the time stamp of the next pointer events
	// (otherwise the fake clock's time).
	eventTime *time.Duration

	binding     *AutomatedTestWidgetsFlutterBinding
	loop        *async.Loop // the test's fake event loop (FakeAsync)
	restoreLoop func()
	lastSize    ui.Size // Size when last pumped (a change is a metrics change)

	previousOnError  foundation.FlutterExceptionHandler
	pendingException *foundation.FlutterErrorDetails
}

// TakeException returns the exception reported (to FlutterErrorOnError)
// since the last call, and clears it; nil if there was none.
func (t *Tester) TakeException() any {
	e := t.pendingException
	t.pendingException = nil
	if e == nil {
		return nil
	}
	return e.Exception
}

// New creates a tester with Flutter's default test surface (800×600), in
// AutomatedTestWidgetsFlutterBinding with a fresh fake event loop (made
// current until the test ends, as testWidgets runs the body in its own
// FakeAsync zone). When the test ends, the tree is unmounted and
// flutter_test's post-test invariants are checked.
func New(t testing.TB) *Tester {
	// `flutter test` runs flutter_tester with --use-test-fonts.
	ui.EngineUseTestFonts()
	r.SetTextEngine(r.UITextEngine{})
	binding := AutomatedTestWidgetsFlutterBindingEnsureInitialized()
	// flutter_test's default view: 800×600 logical pixels at DPR 3.
	o := &widgets.BuildOwner{DevicePixelRatio: 3}
	tt := &Tester{t: t, Owner: o, router: widgets.NewPointerRouter(o), Size: ui.Size{Width: 800, Height: 600},
		binding: binding, loop: async.NewLoop()}
	o.SetViewSize(tt.Size)
	tt.router.TimeStamp = func() time.Duration {
		if tt.eventTime != nil {
			return *tt.eventTime
		}
		return tt.Now()
	}
	// flutter_test awaits each event it sends, running its microtasks (the
	// gesture arena resolves a lone member in one).
	tt.router.AfterEvent = func() { tt.loop.FlushMicrotasks() }
	o.Clipboard = &MemoryClipboard{}
	o.OnNeedsFrame = binding.EnsureVisualUpdate
	binding.startTest(tt)
	t.Cleanup(func() { binding.postTest(tt) })
	return tt
}

// Binding is the binding the test runs in.
func (t *Tester) Binding() *AutomatedTestWidgetsFlutterBinding { return t.binding }

// Loop is the test's fake event loop.
func (t *Tester) Loop() *async.Loop { return t.loop }

// Now is the fake clock: the time elapsed since the test started.
func (t *Tester) Now() time.Duration { return t.loop.Elapsed() }

// PumpWidget mounts w as the root, schedules a frame and pumps it
// (flutter_test's pumpWidget).
func (t *Tester) PumpWidget(w widgets.Widget) {
	t.Owner.RunApp(w)
	t.binding.ScheduleFrame()
	t.Pump()
}

// Pump advances the fake clock by d (optional, default 0), firing due
// timers, and draws a frame if one is scheduled, like flutter_test's
// tester.pump(duration) (see AutomatedTestWidgetsFlutterBinding.Pump).
func (t *Tester) Pump(d ...time.Duration) { t.binding.Pump(d...) }

// PumpAndSettle pumps 16ms frames until no frame is scheduled, like
// flutter_test's pumpAndSettle. It fails after 10 seconds of fake time.
func (t *Tester) PumpAndSettle() {
	t.t.Helper()
	t.Pump()
	for i := 0; t.binding.HasScheduledFrame() || t.Owner.NeedsBuild(); i++ {
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
	return r.GlobalOffset(ro).And(ro.Base().Size)
}

// Tap taps the center of the single element matched by f.
func (t *Tester) Tap(f Finder) {
	t.t.Helper()
	rc := t.Rect(f)
	c := ui.Offset{Dx: rc.Left + rc.Width()/2, Dy: rc.Top + rc.Height()/2}
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
	p := ui.Offset{Dx: rc.Left + rc.Width()/2, Dy: rc.Top + rc.Height()/2}
	t.router.PointerDown(p)
	for moved := 0.0; moved != dy; {
		step := dy - moved
		if step > 10 {
			step = 10
		} else if step < -10 {
			step = -10
		}
		moved += step
		t.router.PointerMove(ui.Offset{Dx: p.Dx, Dy: p.Dy + moved})
	}
	t.router.PointerUp(ui.Offset{Dx: p.Dx, Dy: p.Dy + dy})
}

// Scroll sends a mouse-wheel delta at the center of the element matched by f.
func (t *Tester) Scroll(f Finder, dy float64) {
	t.t.Helper()
	rc := t.Rect(f)
	t.router.Scroll(ui.Offset{Dx: rc.Left + rc.Width()/2, Dy: rc.Top + rc.Height()/2}, dy)
}

// FlingGesture drags by dy over duration in 16ms steps (pumping a frame each
// step, so time passes) and releases, like flutter_test's tester.fling.
func (t *Tester) FlingGesture(f Finder, dy float64, duration time.Duration) {
	t.t.Helper()
	rc := t.Rect(f)
	p := ui.Offset{Dx: rc.Left + rc.Width()/2, Dy: rc.Top + rc.Height()/2}
	steps := int(duration / (16 * time.Millisecond))
	if steps < 1 {
		steps = 1
	}
	t.router.PointerDown(p)
	for i := 1; i <= steps; i++ {
		t.Pump(16 * time.Millisecond)
		t.router.PointerMove(ui.Offset{Dx: p.Dx, Dy: p.Dy + dy*float64(i)/float64(steps)})
	}
	t.router.PointerUp(ui.Offset{Dx: p.Dx, Dy: p.Dy + dy})
}

// TapAt presses and releases at a global position.
func (t *Tester) TapAt(x, y float64) {
	t.router.PointerDown(ui.Offset{Dx: x, Dy: y})
	t.router.PointerUp(ui.Offset{Dx: x, Dy: y})
}

// MemoryClipboard is an in-memory clipboard for tests.
type MemoryClipboard struct{ Text string }

func (c *MemoryClipboard) Get() string  { return c.Text }
func (c *MemoryClipboard) Set(s string) { c.Text = s }

// SendKey sends a key to the focused widget and pumps a frame.
func (t *Tester) SendKey(ev services.KeyEvent) {
	t.Owner.DispatchKey(ev)
	t.Pump()
}

// EnterText types s into the focused widget (replacing the selection, as a
// keyboard does) and pumps a frame.
func (t *Tester) EnterText(s string) {
	t.Owner.DispatchText(s)
	t.Pump()
}

// Fling is a port of flutter_test's flingFrom: from the center of f, it
// drags by dy at speed px/s using 50 moves with synthetic event timestamps,
// pumping a frame whenever more than 16 ms of event time has passed, then
// releases.
func (t *Tester) Fling(f Finder, dy, speed float64) {
	t.t.Helper()
	rc := t.Rect(f)
	start := ui.Offset{Dx: rc.Left + rc.Width()/2, Dy: rc.Top + rc.Height()/2}
	const moveCount = 50
	const frameIntervalUs = 16000.0
	delta := 1e6 * ui.Offset{Dx: 0, Dy: dy}.Distance() / (moveCount * speed) // µs between moves
	ts, last := 0.0, 0.0
	at := func(us float64) { d := time.Duration(math.Round(us)) * time.Microsecond; t.eventTime = &d }
	defer func() { t.eventTime = nil }()
	at(ts)
	t.router.PointerDown(start)
	offset := ui.Offset{Dx: 0, Dy: dy}
	for i := 0; i <= moveCount; i++ {
		at(ts)
		// startLocation + Offset.lerp(Offset.zero, offset, i / kMoveCount)
		t.router.PointerMove(start.Add(*ui.OffsetLerp(&ui.Offset{}, &offset, float64(i)/moveCount)))
		ts += delta
		if ts-last > frameIntervalUs {
			t.Pump(time.Duration(math.Trunc(ts-last)) * time.Microsecond)
			last = ts
		}
	}
	at(ts)
	t.router.PointerUp(ui.Offset{Dx: start.Dx, Dy: start.Dy + dy})
}

// TimedMove is a pointer position (dy from the start) at an event time.
type TimedMove struct {
	T  time.Duration
	DY float64
}

// DragTimed presses at the center of f at event time 0, moves through
// moves with their explicit timestamps (no frames in between), and releases
// at the last move's time — like flutter_test's TestGesture with timeStamps.
func (t *Tester) DragTimed(f Finder, moves []TimedMove) {
	t.t.Helper()
	rc := t.Rect(f)
	start := ui.Offset{Dx: rc.Left + rc.Width()/2, Dy: rc.Top + rc.Height()/2}
	defer func() { t.eventTime = nil }()
	zero := time.Duration(0)
	t.eventTime = &zero
	t.router.PointerDown(start)
	pos := start
	for _, m := range moves {
		at := m.T
		t.eventTime = &at
		pos = ui.Offset{Dx: start.Dx, Dy: start.Dy + m.DY}
		t.router.PointerMove(pos)
	}
	t.router.PointerUp(pos)
}

// Gesture is a pointer controlled step by step with explicit event
// timestamps, like flutter_test's TestGesture.
type Gesture struct {
	t     *Tester
	start r.Offset
	pos   r.Offset
}

// StartGesture presses at the center of f at event time 0.
func (t *Tester) StartGesture(f Finder) *Gesture {
	t.t.Helper()
	rc := t.Rect(f)
	g := &Gesture{t: t, start: ui.Offset{Dx: rc.Left + rc.Width()/2, Dy: rc.Top + rc.Height()/2}}
	g.pos = g.start
	zero := time.Duration(0)
	t.eventTime = &zero
	t.router.PointerDown(g.pos)
	t.eventTime = nil
	return g
}

// MoveTo moves the pointer to start + (0, dy) at event time at.
func (g *Gesture) MoveTo(dy float64, at time.Duration) {
	g.t.eventTime = &at
	g.pos = ui.Offset{Dx: g.start.Dx, Dy: g.start.Dy + dy}
	g.t.router.PointerMove(g.pos)
	g.t.eventTime = nil
}

// Up releases the pointer at event time at.
func (g *Gesture) Up(at time.Duration) {
	g.t.eventTime = &at
	g.t.router.PointerUp(g.pos)
	g.t.eventTime = nil
}
