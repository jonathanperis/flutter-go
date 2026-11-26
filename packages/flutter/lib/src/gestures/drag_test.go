// Port of packages/flutter/test/gestures/drag_test.dart (Flutter 3.50.1):
// all 43 tests.
package gestures_test

import (
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func at(d time.Duration) flutter_test.TestPointerEventOptions {
	return flutter_test.TestPointerEventOptions{TimeStamp: d}
}

func withButtons(b int) flutter_test.TestPointerEventOptions {
	return flutter_test.TestPointerEventOptions{Buttons: &b}
}

// logDrag makes r append "<name>-down/start/update/end/cancel" to log.
func logDrag(log *[]string, name string, r *DragGestureRecognizerBase) {
	r.OnDown = func(*DragDownDetails) { *log = append(*log, name+"-down") }
	r.OnStart = func(*DragStartDetails) { *log = append(*log, name+"-start") }
	r.OnUpdate = func(*DragUpdateDetails) { *log = append(*log, name+"-update") }
	r.OnEnd = func(*DragEndDetails) { *log = append(*log, name+"-end") }
	r.OnCancel = func() { *log = append(*log, name+"-cancel") }
}

func touch(pointer int) *flutter_test.TestPointer { return tp(pointer, ui.PointerDeviceKindTouch) }

func TestDragMostRecentPointToOnStartByDefault(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		competingDrag := NewVerticalDragGestureRecognizer()
		competingDrag.OnStart = func(*DragStartDetails) {}
		t.Cleanup(drag.Dispose)
		t.Cleanup(competingDrag.Dispose)
		var positionAtOnStart ui.Offset
		drag.OnStart = func(d *DragStartDetails) { positionAtOnStart = d.GlobalPosition() }
		pointer := touch(5)
		down := pointer.Down(off(10, 10))
		drag.AddPointer(down)
		competingDrag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		tester.Route(pointer.Move(off(30, 0)))
		dexpect(t, "positionAtOnStart", positionAtOnStart, off(30, 0))
	})
}

func TestDragMostRecentPointToOnStartWithStartConfiguration(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		competingDrag := NewVerticalDragGestureRecognizer()
		competingDrag.OnStart = func(*DragStartDetails) {}
		t.Cleanup(drag.Dispose)
		t.Cleanup(competingDrag.Dispose)
		var positionAtOnStart, updateOffset *ui.Offset
		drag.OnStart = func(d *DragStartDetails) { p := d.GlobalPosition(); positionAtOnStart = &p }
		drag.OnUpdate = func(d *DragUpdateDetails) { p := d.GlobalPosition(); updateOffset = &p }
		pointer := touch(5)
		down := pointer.Down(off(10, 10))
		drag.AddPointer(down)
		competingDrag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		tester.Route(pointer.Move(off(30, 0)))
		dexpect(t, "positionAtOnStart", *positionAtOnStart, off(30, 0))
		if updateOffset != nil {
			t.Errorf("updateOffset %v, want null", *updateOffset)
		}
	})
}

// dragProbe records what the "Should recognize drag" tests check.
type dragProbe struct {
	didStart, didEnd bool
	updatedDelta     *float64
}

func newDragProbe(r *DragGestureRecognizerBase) *dragProbe {
	p := &dragProbe{}
	r.OnStart = func(*DragStartDetails) { p.didStart = true }
	r.OnUpdate = func(d *DragUpdateDetails) { p.updatedDelta = d.PrimaryDelta }
	r.OnEnd = func(*DragEndDetails) { p.didEnd = true }
	return p
}

func (p *dragProbe) expect(t *testing.T, step string, didStart bool, updatedDelta *float64, didEnd bool) {
	t.Helper()
	if p.didStart != didStart || p.didEnd != didEnd || (p.updatedDelta == nil) != (updatedDelta == nil) ||
		(updatedDelta != nil && *p.updatedDelta != *updatedDelta) {
		t.Errorf("%s: start=%v delta=%v end=%v, want %v %v %v", step, p.didStart, p.updatedDelta, p.didEnd,
			didStart, updatedDelta, didEnd)
	}
}

func TestDragShouldRecognizeDrag(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		p := newDragProbe(&drag.DragGestureRecognizerBase)
		pointer := touch(5)
		down := pointer.Down(off(10, 10))
		drag.AddPointer(down)
		tester.CloseArena(5)
		p.expect(t, "closed", false, nil, false)
		tester.Route(down)
		p.expect(t, "down", true, nil, false)
		tester.Route(pointer.Move(off(20, 25)))
		p.expect(t, "move", true, f64(10), false)
		p.didStart, p.updatedDelta = false, nil
		tester.Route(pointer.Move(off(20, 25)))
		p.expect(t, "move 2", false, f64(0), false)
		p.updatedDelta = nil
		tester.Route(pointer.Up())
		p.expect(t, "up", false, nil, true)
	})
}

func rejectMouse(t *testing.T, vertical bool) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		opts := GestureRecognizerOptions{SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindTouch: {}}}
		var base *DragGestureRecognizerBase
		var r DragGestureRecognizer
		if vertical {
			v := NewVerticalDragGestureRecognizer(opts)
			base, r = &v.DragGestureRecognizerBase, v
		} else {
			h := NewHorizontalDragGestureRecognizer(opts)
			base, r = &h.DragGestureRecognizerBase, h
		}
		base.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(r.Dispose)
		p := newDragProbe(base)
		pointer := tp(5, ui.PointerDeviceKindMouse)
		down := pointer.Down(off(10, 10))
		r.AddPointer(down)
		tester.CloseArena(5)
		p.expect(t, "closed", false, nil, false)
		tester.Route(down)
		p.expect(t, "down", false, nil, false)
		move := off(20, 25)
		if vertical {
			move = off(25, 20)
		}
		tester.Route(pointer.Move(move))
		p.expect(t, "move", false, nil, false)
		tester.Route(pointer.Move(move))
		p.expect(t, "move 2", false, nil, false)
		tester.Route(pointer.Up())
		p.expect(t, "up", false, nil, false)
	})
}

func TestDragRejectMouseHorizontal(t *testing.T) { rejectMouse(t, false) }
func TestDragRejectMouseVertical(t *testing.T)   { rejectMouse(t, true) }

func TestDragOnStartBehavior(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		var startTimestamp, updatedTimestamp *time.Duration
		var positionAtOnStart, updateDelta *ui.Offset
		drag.OnStart = func(d *DragStartDetails) {
			startTimestamp = d.SourceTimeStamp
			p := d.GlobalPosition()
			positionAtOnStart = &p
		}
		drag.OnUpdate = func(d *DragUpdateDetails) {
			updatedTimestamp = d.SourceTimeStamp
			delta := d.Delta
			updateDelta = &delta
		}
		reset := func() { startTimestamp, positionAtOnStart, updatedTimestamp, updateDelta = nil, nil, nil, nil }
		expectTS := func(name string, got *time.Duration, want any) {
			t.Helper()
			if want == nil {
				if got != nil {
					t.Errorf("%s = %v, want null", name, *got)
				}
				return
			}
			if got == nil || *got != want.(time.Duration) {
				t.Errorf("%s = %v, want %v", name, got, want)
			}
		}
		expectOff := func(name string, got *ui.Offset, want any) {
			t.Helper()
			if want == nil {
				if got != nil {
					t.Errorf("%s = %v, want null", name, *got)
				}
				return
			}
			if got == nil || *got != want.(ui.Offset) {
				t.Errorf("%s = %v, want %v", name, got, want)
			}
		}

		// No competing, dragStartBehavior == DragStartBehavior.down
		pointer := touch(5)
		down := pointer.Down(off(10, 10), at(ms(100)))
		drag.AddPointer(down)
		tester.CloseArena(5)
		expectTS("start", startTimestamp, nil)
		expectOff("position", positionAtOnStart, nil)
		expectTS("updated", updatedTimestamp, nil)
		tester.Route(down)
		expectTS("start", startTimestamp, ms(100))
		expectOff("position", positionAtOnStart, off(10, 10))
		expectTS("updated", updatedTimestamp, nil)
		tester.Route(pointer.Move(off(20, 25), at(ms(200))))
		expectTS("updated", updatedTimestamp, ms(200))
		expectOff("delta", updateDelta, off(10, 0))
		tester.Route(pointer.Move(off(20, 25), at(ms(300))))
		expectTS("updated", updatedTimestamp, ms(300))
		expectOff("delta", updateDelta, off(0, 0))
		tester.Route(pointer.Up())

		// No competing, dragStartBehavior == DragStartBehavior.start
		drag.DragStartBehavior = DragStartBehaviorStart
		reset()
		down = pointer.Down(off(10, 10), at(ms(400)))
		drag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		expectTS("start", startTimestamp, ms(400))
		expectOff("position", positionAtOnStart, off(10, 10))
		expectTS("updated", updatedTimestamp, nil)
		tester.Route(pointer.Move(off(20, 25), at(ms(500))))
		expectTS("updated", updatedTimestamp, ms(500))
		tester.Route(pointer.Up())

		// With competing, dragStartBehavior == DragStartBehavior.start
		reset()
		competingDrag := NewVerticalDragGestureRecognizer()
		competingDrag.OnStart = func(*DragStartDetails) {}
		t.Cleanup(competingDrag.Dispose)
		down = pointer.Down(off(10, 10), at(ms(600)))
		drag.AddPointer(down)
		competingDrag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		expectTS("start", startTimestamp, nil)
		expectOff("position", positionAtOnStart, nil)
		expectTS("updated", updatedTimestamp, nil)
		tester.Route(pointer.Move(off(30, 10), at(ms(700))))
		expectTS("start", startTimestamp, ms(700))
		expectOff("position", positionAtOnStart, off(30, 10))
		expectTS("updated", updatedTimestamp, nil)
		tester.Route(pointer.Up())

		// With competing, dragStartBehavior == DragStartBehavior.down
		drag.DragStartBehavior = DragStartBehaviorDown
		reset()
		down = pointer.Down(off(10, 10), at(ms(800)))
		drag.AddPointer(down)
		competingDrag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		expectTS("start", startTimestamp, nil)
		expectOff("position", positionAtOnStart, nil)
		expectTS("updated", updatedTimestamp, nil)
		tester.Route(pointer.Move(off(30, 10), at(ms(900))))
		expectTS("start", startTimestamp, ms(900))
		expectOff("position", positionAtOnStart, off(10, 10))
		expectTS("updated", updatedTimestamp, ms(900))
		expectOff("delta", updateDelta, off(20, 0))
		tester.Route(pointer.Up())
	})
}

func TestDragShouldReportOriginalTimestamps(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		var startTimestamp, updatedTimestamp *time.Duration
		drag.OnStart = func(d *DragStartDetails) { startTimestamp = d.SourceTimeStamp }
		drag.OnUpdate = func(d *DragUpdateDetails) { updatedTimestamp = d.SourceTimeStamp }
		pointer := touch(5)
		down := pointer.Down(off(10, 10), at(ms(100)))
		drag.AddPointer(down)
		tester.CloseArena(5)
		if startTimestamp != nil {
			t.Error("start not null")
		}
		tester.Route(down)
		dexpect(t, "start", *startTimestamp, ms(100))
		tester.Route(pointer.Move(off(20, 25), at(ms(200))))
		dexpect(t, "updated", *updatedTimestamp, ms(200))
		tester.Route(pointer.Move(off(20, 25), at(ms(300))))
		dexpect(t, "updated", *updatedTimestamp, ms(300))
	})
}

func TestDragInitialDownPointToOnStartWithDownConfiguration(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		competingDrag := NewVerticalDragGestureRecognizer()
		competingDrag.DragStartBehavior = DragStartBehaviorDown
		competingDrag.OnStart = func(*DragStartDetails) {}
		t.Cleanup(drag.Dispose)
		t.Cleanup(competingDrag.Dispose)
		var positionAtOnStart, updateOffset, updateDelta ui.Offset
		drag.OnStart = func(d *DragStartDetails) { positionAtOnStart = d.GlobalPosition() }
		drag.OnUpdate = func(d *DragUpdateDetails) { updateOffset = d.GlobalPosition(); updateDelta = d.Delta }
		pointer := touch(5)
		down := pointer.Down(off(10, 10))
		drag.AddPointer(down)
		competingDrag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		tester.Route(pointer.Move(off(30, 0)))
		dexpect(t, "positionAtOnStart", positionAtOnStart, off(10, 10))
		dexpect(t, "updateOffset", updateOffset, off(30, 10))
		dexpect(t, "updateDelta", updateDelta, off(20, 0))
	})
}

// multiPointerScript is the shared script of the "Drag with multiple
// pointers in down behavior" tests (it returns the log).
func multiPointerScript(t *testing.T, strategy *MultitouchDragStrategy, tail func(tester *flutter_test.GestureTester, p5, p6 *flutter_test.TestPointer, mark func(string))) []string {
	var log []string
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag1 := NewHorizontalDragGestureRecognizer()
		drag1.DragStartBehavior = DragStartBehaviorDown
		drag2 := NewVerticalDragGestureRecognizer()
		drag2.DragStartBehavior = DragStartBehaviorDown
		if strategy != nil {
			drag1.MultitouchDragStrategy = *strategy
			drag2.MultitouchDragStrategy = *strategy
		}
		t.Cleanup(drag1.Dispose)
		t.Cleanup(drag2.Dispose)
		multiPointerAdd = func(down *PointerDownEvent) {
			drag1.AddPointer(down)
			drag2.AddPointer(down)
		}
		logDrag(&log, "drag1", &drag1.DragGestureRecognizerBase)
		logDrag(&log, "drag2", &drag2.DragGestureRecognizerBase)
		mark := func(s string) { log = append(log, s) }

		pointer5 := touch(5)
		down5 := pointer5.Down(off(10, 10))
		drag1.AddPointer(down5)
		drag2.AddPointer(down5)
		tester.CloseArena(5)
		tester.Route(down5)
		mark("-a")
		tester.Route(pointer5.Move(off(100, 0)))
		mark("-b")
		tester.Route(pointer5.Move(off(50, 50)))
		mark("-c")
		pointer6 := touch(6)
		down6 := pointer6.Down(off(20, 20))
		drag1.AddPointer(down6)
		drag2.AddPointer(down6)
		tester.CloseArena(6)
		tester.Route(down6)
		mark("-d")
		tail(tester, pointer5, pointer6, mark)
	})
	return log
}

// multiPointerAdd adds a pointer to the script's two recognizers.
var multiPointerAdd func(down *PointerDownEvent)

var multiPointerPrefix = []string{"drag1-down", "drag2-down", "-a", "drag2-cancel", "drag1-start", "drag1-update",
	"-b", "drag1-update", "-c", "drag2-down", "drag2-cancel", "-d"}

func TestDragMultiplePointersDownSumAllPointers(t *testing.T) {
	s := MultitouchDragStrategySumAllPointers
	log := multiPointerScript(t, &s, func(tester *flutter_test.GestureTester, p5, p6 *flutter_test.TestPointer, mark func(string)) {
		tester.Route(p5.Move(off(0, 100)))
		mark("-e")
		tester.Route(p5.Move(off(70, 70)))
		mark("-f")
		tester.Route(p6.Move(off(0, 100)))
		mark("-g")
		tester.Route(p6.Move(off(70, 70)))
		mark("-h")
		tester.Route(p5.Up())
		tester.Route(p6.Up())
	})
	dexpect(t, "log", log, append(append([]string{}, multiPointerPrefix...), "drag1-update", "-e", "drag1-update", "-f",
		"drag1-update", "-g", "drag1-update", "-h", "drag1-end"))
}

func defaultOrLatest(t *testing.T, strategy *MultitouchDragStrategy) {
	log := multiPointerScript(t, strategy, func(tester *flutter_test.GestureTester, p5, p6 *flutter_test.TestPointer, mark func(string)) {
		tester.Route(p5.Move(off(0, 100)))
		mark("-e")
		tester.Route(p5.Move(off(70, 70)))
		mark("-f")
		tester.Route(p6.Move(off(0, 100)))
		mark("-g")
		tester.Route(p6.Move(off(70, 70)))
		mark("-h")
		tester.Route(p6.Up())
		mark("-i")
		tester.Route(p5.Move(off(0, 100)))
		mark("-j")
		tester.Route(p5.Move(off(70, 70)))
		mark("-k")
		tester.Route(p5.Up())
	})
	dexpect(t, "log", log, append(append([]string{}, multiPointerPrefix...), "-e", "-f", "drag1-update", "-g",
		"drag1-update", "-h", "-i", "drag1-update", "-j", "drag1-update", "-k", "drag1-end"))
}

func TestDragMultiplePointersDownDefault(t *testing.T) { defaultOrLatest(t, nil) }

// averageBoundaryScript is the script of the three "drag with multiple
// pointers - averageBoundaryPointers" tests; pos maps the scalar position
// of the Dart test onto the recognizer's axis.
func averageBoundaryScript(t *testing.T, r DragGestureRecognizer, pos func(v float64) ui.Offset) []string {
	var log []string
	testGesture(t, func(tester *flutter_test.GestureTester) {
		base := dragBaseOf(r)
		base.MultitouchDragStrategy = MultitouchDragStrategyAverageBoundaryPointers
		base.OnUpdate = func(d *DragUpdateDetails) { log = append(log, "drag-update ("+d.Delta.String()+")") }
		pointers := map[int]*flutter_test.TestPointer{}
		down := func(n int) {
			p := touch(n)
			pointers[n] = p
			e := p.Down(off(0, 0))
			r.AddPointer(e)
			tester.CloseArena(n)
			tester.Route(e)
		}
		move := func(n int, v float64) { tester.Route(pointers[n].Move(pos(v))) }
		mark := func(s string) { log = append(log, s) }
		frame := func(d time.Duration) {
			flutter_test.AutomatedTestWidgetsFlutterBindingEnsureInitialized().HandleBeginFrame(&d)
			flutter_test.AutomatedTestWidgetsFlutterBindingEnsureInitialized().HandleDrawFrame()
		}
		down(5)
		mark("-a")
		move(5, 100)
		down(6)
		mark("-b")
		move(6, 110)
		down(7)
		mark("-c")
		move(7, -100)
		down(8)
		mark("-d")
		move(8, -110)
		mark("-e")
		move(5, 120)
		mark("-f")
		move(7, -120)
		frame(ms(100))
		mark("-g")
		move(6, 120)
		mark("-h")
		move(8, -120)
		mark("-i")
		move(5, 130)
		mark("-j")
		move(7, -130)
		for _, n := range []int{5, 6, 7, 8} {
			tester.Route(pointers[n].Up())
		}
		frame(0)
		r.Dispose()
	})
	return log
}

func dragBaseOf(r DragGestureRecognizer) *DragGestureRecognizerBase {
	switch d := r.(type) {
	case *VerticalDragGestureRecognizer:
		return &d.DragGestureRecognizerBase
	case *HorizontalDragGestureRecognizer:
		return &d.DragGestureRecognizerBase
	case *PanGestureRecognizer:
		return &d.DragGestureRecognizerBase
	}
	panic("unknown recognizer")
}

func TestDragHorizontalAverageBoundaryPointers(t *testing.T) {
	log := averageBoundaryScript(t, NewHorizontalDragGestureRecognizer(), func(v float64) ui.Offset { return off(v, 0) })
	dexpect(t, "log", log, []string{"-a", "drag-update (Offset(100.0, 0.0))", "-b", "drag-update (Offset(10.0, 0.0))",
		"-c", "drag-update (Offset(-100.0, 0.0))", "-d", "drag-update (Offset(-10.0, 0.0))", "-e",
		"drag-update (Offset(10.0, 0.0))", "-f", "drag-update (Offset(-10.0, 0.0))", "-g",
		"drag-update (Offset(10.0, 0.0))", "-h", "drag-update (Offset(-10.0, 0.0))", "-i",
		"drag-update (Offset(0.0, 0.0))", "-j", "drag-update (Offset(0.0, 0.0))"})
}

func TestDragVerticalAverageBoundaryPointers(t *testing.T) {
	log := averageBoundaryScript(t, NewVerticalDragGestureRecognizer(), func(v float64) ui.Offset { return off(0, v) })
	dexpect(t, "log", log, []string{"-a", "drag-update (Offset(0.0, 100.0))", "-b", "drag-update (Offset(0.0, 10.0))",
		"-c", "drag-update (Offset(0.0, -100.0))", "-d", "drag-update (Offset(0.0, -10.0))", "-e",
		"drag-update (Offset(0.0, 10.0))", "-f", "drag-update (Offset(0.0, -10.0))", "-g",
		"drag-update (Offset(0.0, 10.0))", "-h", "drag-update (Offset(0.0, -10.0))", "-i",
		"drag-update (Offset(0.0, 0.0))", "-j", "drag-update (Offset(0.0, 0.0))"})
}

func TestDragPanAverageBoundaryPointers(t *testing.T) {
	log := averageBoundaryScript(t, NewPanGestureRecognizer(), func(v float64) ui.Offset { return off(v, v) })
	dexpect(t, "log", log, []string{"-a", "drag-update (Offset(100.0, 100.0))", "-b", "drag-update (Offset(5.0, 5.0))",
		"-c", "drag-update (Offset(-68.3, -68.3))", "-d", "drag-update (Offset(-36.7, -36.7))", "-e",
		"drag-update (Offset(5.0, 5.0))", "-f", "drag-update (Offset(-5.0, -5.0))", "-g",
		"drag-update (Offset(2.5, 2.5))", "-h", "drag-update (Offset(-2.5, -2.5))", "-i",
		"drag-update (Offset(2.5, 2.5))", "-j", "drag-update (Offset(-2.5, -2.5))"})
}

func near(a, b, eps float64) bool { return a-b <= eps && b-a <= eps }

func TestDragClampMaxVelocity(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		var velocity Velocity
		var primaryVelocity *float64
		drag.OnEnd = func(d *DragEndDetails) { velocity, primaryVelocity = d.Velocity, d.PrimaryVelocity }
		pointer := touch(5)
		down := pointer.Down(off(10, 25), at(ms(10)))
		drag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		for i := 0; i <= 10; i++ {
			tester.Route(pointer.Move(off(float64(20+10*i), 25), at(ms(10+i))))
		}
		tester.Route(pointer.Up(at(ms(20))))
		dx := velocity.PixelsPerSecond.Dx
		expectTrue(t, "dx in range", dx >= 0.99*KMaxFlingVelocity && dx <= KMaxFlingVelocity)
		expectTrue(t, "dy ~ 0", near(velocity.PixelsPerSecond.Dy, 0, 1e-10))
		dexpect(t, "primaryVelocity", *primaryVelocity, dx)
	})
}

// performDragToEnd drags the pointer at the given velocity and returns the
// details the recognizer passes to onEnd (it sets OnEnd).
func performDragToEnd(tester *flutter_test.GestureTester, recognizer DragGestureRecognizer, pointerVelocity ui.Offset) *DragEndDetails {
	var actual *DragEndDetails
	dragBaseOf(recognizer).OnEnd = func(d *DragEndDetails) { actual = d }
	pointer := flutter_test.NewTestPointer()
	down := pointer.Down(off(0, 0))
	recognizer.AddPointer(down)
	tester.CloseArena(pointer.Pointer)
	tester.Route(down)
	tester.Route(pointer.Move(pointerVelocity.Mul(0.025), at(ms(25))))
	tester.Route(pointer.Move(pointerVelocity.Mul(0.050), at(ms(50))))
	tester.Route(pointer.Up(at(ms(50))))
	return actual
}

func TestDragClampMaxPanVelocityIsotropically(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		recognizer := NewPanGestureRecognizer()
		t.Cleanup(recognizer.Dispose)
		check := func(v, want ui.Offset) {
			t.Helper()
			actual := performDragToEnd(tester, recognizer, v)
			p := actual.Velocity.PixelsPerSecond
			expectTrue(t, "velocity "+p.String()+" ~ "+want.String(), near(p.Dx, want.Dx, 0.1) && near(p.Dy, want.Dy, 0.1))
			expectTrue(t, "primaryVelocity null", actual.PrimaryVelocity == nil)
		}
		check(off(400, 400), off(400, 400))
		check(off(2000, -2000), off(2000, -2000))
		check(off(-8000, -8000), off(-5656.9, -5656.9))
		check(off(-8000, 6000), off(-6400, 4800))
		check(off(-9000, 0), off(-8000, 0))
		check(off(-9000, -1000), off(-7951.1, -883.5))
		check(off(-1000, 9000), off(-883.5, 7951.1))
		check(off(0, 9000), off(0, 8000))
	})
}

func clampAxis(t *testing.T, vertical bool) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var recognizer DragGestureRecognizer
		if vertical {
			recognizer = NewVerticalDragGestureRecognizer()
		} else {
			recognizer = NewHorizontalDragGestureRecognizer()
		}
		t.Cleanup(recognizer.Dispose)
		check := func(v ui.Offset, want float64) {
			t.Helper()
			actual := performDragToEnd(tester, recognizer, v)
			expectTrue(t, "primary", near(*actual.PrimaryVelocity, want, 0.1))
			p := actual.Velocity.PixelsPerSecond
			if vertical {
				expectTrue(t, "dx 0, dy primary", p.Dx == 0 && p.Dy == *actual.PrimaryVelocity)
			} else {
				expectTrue(t, "dy 0, dx primary", p.Dy == 0 && p.Dx == *actual.PrimaryVelocity)
			}
		}
		if vertical {
			check(off(500, 400), 400)
			check(off(3000, -2000), -2000)
			check(off(-9000, -9000), -8000)
			check(off(-9000, 0), 0)
			check(off(-9000, 1000), 1000)
			check(off(-1000, -9000), -8000)
			check(off(0, -9000), -8000)
		} else {
			check(off(500, 400), 500)
			check(off(3000, -2000), 3000)
			check(off(-9000, -9000), -8000)
			check(off(-9000, 0), -8000)
			check(off(-9000, 1000), -8000)
			check(off(-1000, -9000), -1000)
			check(off(0, -9000), 0)
		}
	})
}

func TestDragClampMaxVerticalVelocity(t *testing.T)   { clampAxis(t, true) }
func TestDragClampMaxHorizontalVelocity(t *testing.T) { clampAxis(t, false) }

func TestDragSynthesizedEventsIgnoredForVelocity(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		var velocity Velocity
		drag.OnEnd = func(d *DragEndDetails) { velocity = d.Velocity }
		pointer := flutter_test.NewTestPointer()
		down := pointer.Down(off(10, 25), at(ms(10)))
		drag.AddPointer(down)
		tester.CloseArena(1)
		tester.Route(down)
		for i := 2; i <= 5; i++ {
			tester.Route(pointer.Move(off(float64(10*i), 25), at(ms(10*i))))
		}
		tester.Route(NewPointerMoveEvent(PointerMoveEventOptions{Pointer: 1, Position: off(51, 26),
			TimeStamp: ms(60), Synthesized: true}))
		tester.Route(pointer.Up(at(ms(70))))
		expectTrue(t, "dx", near(velocity.PixelsPerSecond.Dx, 1000, 1e-10))
		expectTrue(t, "dy", near(velocity.PixelsPerSecond.Dy, 0, 1e-10))
	})
}

func TestDragQuickFlicksHaveVelocity(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		var velocity Velocity
		drag.OnEnd = func(d *DragEndDetails) { velocity = d.Velocity }
		pointer := flutter_test.NewTestPointer()
		down := pointer.Down(off(10, 25), at(ms(10)))
		drag.AddPointer(down)
		tester.CloseArena(1)
		tester.Route(down)
		tester.Route(pointer.Move(off(20, 25), at(ms(20))))
		tester.Route(pointer.Move(off(30, 25), at(ms(30))))
		tester.Route(pointer.Up(at(ms(40))))
		expectTrue(t, "dx", near(velocity.PixelsPerSecond.Dx, 1000, 1e-10))
		expectTrue(t, "dy", near(velocity.PixelsPerSecond.Dy, 0, 1e-10))
	})
}

func TestDragDetailsHaveOneLineDescription(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		for _, s := range []string{NewDragDownDetails().String(), NewDragStartDetails().String(),
			NewDragUpdateDetails(DragUpdateDetailsOptions{}).String(), NewDragEndDetails().String()} {
			expectTrue(t, "one line: "+s, s != "" && !containsNewline(s))
		}
	})
}

func containsNewline(s string) bool {
	for _, c := range s {
		if c == '\n' {
			return true
		}
	}
	return false
}

func TestDragShouldRecognizeDragDeltas(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		didStart, didEnd := false, false
		var updateDelta *ui.Offset
		var updatePrimaryDelta *float64
		drag.OnStart = func(*DragStartDetails) { didStart = true }
		drag.OnUpdate = func(d *DragUpdateDetails) {
			delta := d.Delta
			updateDelta = &delta
			updatePrimaryDelta = d.PrimaryDelta
		}
		drag.OnEnd = func(*DragEndDetails) { didEnd = true }
		pointer := touch(5)
		down := pointer.Down(off(10, 10))
		drag.AddPointer(down)
		tester.CloseArena(5)
		expectTrue(t, "nothing yet", !didStart && updateDelta == nil && updatePrimaryDelta == nil && !didEnd)
		tester.Route(down)
		expectTrue(t, "started", didStart && updateDelta == nil && !didEnd)
		didStart = false
		tester.Route(pointer.Move(off(20, 25)))
		expectTrue(t, "update 1", !didStart && *updateDelta == off(10, 0) && *updatePrimaryDelta == 10 && !didEnd)
		updateDelta, updatePrimaryDelta = nil, nil
		tester.Route(pointer.Move(off(20, 25)))
		expectTrue(t, "update 2", !didStart && *updateDelta == off(0, 0) && *updatePrimaryDelta == 0 && !didEnd)
		updateDelta, updatePrimaryDelta = nil, nil
		tester.Route(pointer.Up())
		expectTrue(t, "ended", !didStart && updateDelta == nil && updatePrimaryDelta == nil && didEnd)
	})
}

func TestDragShouldRecognizeDragGlobalPositions(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer()
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		var latestGlobalPosition ui.Offset
		var latestDelta *ui.Offset
		drag.OnStart = func(d *DragStartDetails) { latestGlobalPosition = d.GlobalPosition() }
		drag.OnUpdate = func(d *DragUpdateDetails) {
			latestGlobalPosition = d.GlobalPosition()
			delta := d.Delta
			latestDelta = &delta
		}
		pointer := touch(5)
		down := pointer.Down(off(10, 10))
		drag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		dexpect(t, "position", latestGlobalPosition, off(10, 10))
		expectTrue(t, "no delta", latestDelta == nil)
		tester.Route(pointer.Move(off(20, 25)))
		dexpect(t, "position", latestGlobalPosition, off(20, 25))
		dexpect(t, "delta", *latestDelta, off(10, 0))
		tester.Route(pointer.Move(off(0, 45)))
		dexpect(t, "position", latestGlobalPosition, off(0, 45))
		dexpect(t, "delta", *latestDelta, off(-20, 0))
		tester.Route(pointer.Up())
	})
}

func TestDragCanFilterByDeviceKind(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		drag := NewHorizontalDragGestureRecognizer(GestureRecognizerOptions{
			SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindMouse: {}}})
		drag.DragStartBehavior = DragStartBehaviorDown
		t.Cleanup(drag.Dispose)
		p := newDragProbe(&drag.DragGestureRecognizerBase)
		touchPointer := touch(5)
		touchDown := touchPointer.Down(off(10, 10))
		drag.AddPointer(touchDown)
		tester.CloseArena(5)
		p.expect(t, "touch closed", false, nil, false)
		tester.Route(touchDown)
		p.expect(t, "touch down", false, nil, false)
		tester.Route(touchPointer.Move(off(20, 25)))
		p.expect(t, "touch move", false, nil, false)
		tester.Route(touchPointer.Up())
		p.expect(t, "touch up", false, nil, false)

		mousePointer := tp(5, ui.PointerDeviceKindMouse)
		mouseDown := mousePointer.Down(off(10, 10))
		drag.AddPointer(mouseDown)
		tester.CloseArena(5)
		p.expect(t, "mouse closed", false, nil, false)
		tester.Route(mouseDown)
		p.expect(t, "mouse down", true, nil, false)
		p.didStart = false
		tester.Route(mousePointer.Move(off(20, 25)))
		p.expect(t, "mouse move", false, f64(10), false)
		p.updatedDelta = nil
		tester.Route(mousePointer.Up())
		p.expect(t, "mouse up", false, nil, true)
	})
}

// competitor is the tests' `TapGestureRecognizer()..onTap = () {}`, only
// a competitor in the arena.
type competitor struct{ tap *TapGestureRecognizer }

func newCompetitor(t *testing.T) *competitor {
	tap := NewTapGestureRecognizer()
	tap.OnTap = func() {}
	t.Cleanup(tap.Dispose)
	return &competitor{tap: tap}
}

func (c *competitor) addPointer(down *PointerDownEvent) { c.tap.AddPointer(down) }

func consistentButtonSetUp(t *testing.T) (*PanGestureRecognizer, *competitor, *[]string) {
	logs := []string{}
	pan := NewPanGestureRecognizer()
	pan.OnStart = func(*DragStartDetails) { logs = append(logs, "start") }
	pan.OnDown = func(*DragDownDetails) { logs = append(logs, "down") }
	pan.OnUpdate = func(*DragUpdateDetails) { logs = append(logs, "update") }
	pan.OnCancel = func() { logs = append(logs, "cancel") }
	pan.OnEnd = func(*DragEndDetails) { logs = append(logs, "end") }
	return pan, newCompetitor(t), &logs
}

func TestDragButtonChangeBeforeAcceptanceCancels(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan, tap, logs := consistentButtonSetUp(t)
		t.Cleanup(pan.Dispose)
		pointer := tp(5, ui.PointerDeviceKindMouse, KPrimaryButton)
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tap.addPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		dexpect(t, "logs", *logs, []string{"down"})
		tester.Route(pointer.Move(off(30, 30), withButtons(KSecondaryButton)))
		dexpect(t, "logs", *logs, []string{"down", "cancel"})
		tester.Route(pointer.Up())
	})
}

func TestDragButtonChangeBeforeAcceptanceAllowsNextDrag(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan, tap, logs := consistentButtonSetUp(t)
		t.Cleanup(pan.Dispose)
		pointer := tp(5, ui.PointerDeviceKindMouse, KPrimaryButton)
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tap.addPointer(down)
		tester.CloseArena(down.Pointer())
		tester.Route(down)
		tester.Route(pointer.Move(off(10, 10), withButtons(KSecondaryButton)))
		tester.Route(pointer.Up())
		dexpect(t, "logs", *logs, []string{"down", "cancel"})
		*logs = (*logs)[:0]

		pointer2 := tp(6, ui.PointerDeviceKindMouse, KPrimaryButton)
		down2 := pointer2.Down(off(10, 10))
		pan.AddPointer(down2)
		tap.addPointer(down2)
		tester.CloseArena(down2.Pointer())
		tester.Route(down2)
		dexpect(t, "logs", *logs, []string{"down"})
		tester.Route(pointer2.Move(off(30, 30)))
		dexpect(t, "logs", *logs, []string{"down", "start"})
		tester.Route(pointer2.Up())
		dexpect(t, "logs", *logs, []string{"down", "start", "end"})
	})
}

func TestDragButtonChangeAfterAcceptanceEnds(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan, tap, logs := consistentButtonSetUp(t)
		t.Cleanup(pan.Dispose)
		pointer := tp(5, ui.PointerDeviceKindMouse, KPrimaryButton)
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tap.addPointer(down)
		tester.CloseArena(down.Pointer())
		tester.Route(down)
		dexpect(t, "logs", *logs, []string{"down"})
		tester.Route(pointer.Move(off(30, 30)))
		dexpect(t, "logs", *logs, []string{"down", "start"})
		tester.Route(pointer.Move(off(30, 30), withButtons(KSecondaryButton)))
		dexpect(t, "logs", *logs, []string{"down", "start", "end"})
		tester.Route(pointer.Move(off(50, 50)))
		dexpect(t, "logs", *logs, []string{"down", "start", "end"})
		tester.Route(pointer.Up())
	})
}

func TestDragButtonChangeAfterAcceptanceAllowsNextDrag(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan, tap, logs := consistentButtonSetUp(t)
		t.Cleanup(pan.Dispose)
		pointer := tp(5, ui.PointerDeviceKindMouse, KPrimaryButton)
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tap.addPointer(down)
		tester.CloseArena(down.Pointer())
		tester.Route(down)
		tester.Route(pointer.Move(off(30, 30)))
		tester.Route(pointer.Move(off(30, 31), withButtons(KSecondaryButton)))
		tester.Route(pointer.Up())
		dexpect(t, "logs", *logs, []string{"down", "start", "end"})
		*logs = (*logs)[:0]

		pointer2 := tp(6, ui.PointerDeviceKindMouse, KPrimaryButton)
		down2 := pointer2.Down(off(10, 10))
		pan.AddPointer(down2)
		tap.addPointer(down2)
		tester.CloseArena(down2.Pointer())
		tester.Route(down2)
		dexpect(t, "logs", *logs, []string{"down"})
		tester.Route(pointer2.Move(off(30, 30)))
		dexpect(t, "logs", *logs, []string{"down", "start"})
		tester.Route(pointer2.Up())
		dexpect(t, "logs", *logs, []string{"down", "start", "end"})
	})
}

func TestDragSecondaryDragShouldNotTriggerPrimary(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized []string
		tap := newCompetitor(t)
		pan := NewPanGestureRecognizer()
		logDrag(&recognized, "primary", &pan.DragGestureRecognizerBase)
		t.Cleanup(pan.Dispose)
		pointer := flutter_test.NewTestPointer(flutter_test.TestPointerOptions{Pointer: intp(5), Device: intp(0),
			Buttons: intp(KSecondaryButton)})
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tap.addPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		tester.Route(pointer.Move(off(20, 30)))
		tester.Route(pointer.Move(off(20, 25)))
		tester.Route(pointer.Up())
		dexpect(t, "recognized", recognized, []string(nil))
	})
}

// panProbe records what the pan tests check.
type panProbe struct {
	didStart, didEnd bool
	delta            *ui.Offset
}

func newPanProbe(r *PanGestureRecognizer) *panProbe {
	p := &panProbe{}
	r.OnStart = func(*DragStartDetails) { p.didStart = true }
	r.OnUpdate = func(d *DragUpdateDetails) { delta := d.Delta; p.delta = &delta }
	r.OnEnd = func(*DragEndDetails) { p.didEnd = true }
	return p
}

func (p *panProbe) expect(t *testing.T, step string, didStart bool, delta *ui.Offset, didEnd bool) {
	t.Helper()
	if p.didStart != didStart || p.didEnd != didEnd || (p.delta == nil) != (delta == nil) || (delta != nil && *p.delta != *delta) {
		t.Errorf("%s: start=%v delta=%v end=%v, want %v %v %v", step, p.didStart, p.delta, p.didEnd, didStart, delta, didEnd)
	}
}

func offp(x, y float64) *ui.Offset { o := off(x, y); return &o }

func panUpdate(p *flutter_test.TestPointer, pan ui.Offset) *PointerPanZoomUpdateEvent {
	return p.PanZoomUpdate(off(10, 10), flutter_test.TestPointerPanZoomUpdateOptions{Pan: pan})
}

func TestDragPanGesturesFromPlatform(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan := NewPanGestureRecognizer()
		competingPan := NewPanGestureRecognizer()
		t.Cleanup(pan.Dispose)
		t.Cleanup(competingPan.Dispose)
		p := newPanProbe(pan)
		pointer := tp(2, ui.PointerDeviceKindTrackpad)
		start := pointer.PanZoomStart(off(10, 10))
		pan.AddPointerPanZoom(start)
		competingPan.AddPointerPanZoom(start)
		tester.CloseArena(2)
		p.expect(t, "closed", false, nil, false)
		tester.Route(start)
		p.expect(t, "start", false, nil, false)
		tester.Route(panUpdate(pointer, off(20, 20)))
		p.expect(t, "28 < 36", false, nil, false)
		tester.Route(panUpdate(pointer, off(30, 30)))
		p.expect(t, "42 > 36", true, nil, false)
		p.didStart = false
		tester.Route(panUpdate(pointer, off(30, 25)))
		p.expect(t, "update", false, offp(0, -5), false)
		p.delta = nil
		tester.Route(pointer.PanZoomEnd())
		p.expect(t, "end", false, nil, true)
	})
}

func TestDragPanZoomAllowsTouchesToJoin(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan := NewPanGestureRecognizer()
		competingPan := NewPanGestureRecognizer()
		t.Cleanup(pan.Dispose)
		t.Cleanup(competingPan.Dispose)
		p := newPanProbe(pan)
		panZoomPointer := tp(2, ui.PointerDeviceKindTrackpad)
		touchPointer := touch(3)
		start := panZoomPointer.PanZoomStart(off(10, 10))
		pan.AddPointerPanZoom(start)
		competingPan.AddPointerPanZoom(start)
		tester.CloseArena(2)
		p.expect(t, "closed", false, nil, false)
		tester.Route(start)
		p.expect(t, "start", false, nil, false)
		tester.Route(panUpdate(panZoomPointer, off(20, 20)))
		p.expect(t, "28 < 36", false, nil, false)
		tester.Route(panUpdate(panZoomPointer, off(30, 30)))
		p.expect(t, "42 > 36", true, nil, false)
		p.didStart = false
		tester.Route(panUpdate(panZoomPointer, off(30, 25)))
		p.expect(t, "update", false, offp(0, -5), false)
		p.delta = nil
		touchDown := touchPointer.Down(off(20, 20))
		pan.AddPointer(touchDown)
		competingPan.AddPointer(touchDown)
		tester.CloseArena(3)
		p.expect(t, "touch closed", false, nil, false)
		tester.Route(touchDown)
		p.expect(t, "touch down", false, nil, false)
		tester.Route(touchPointer.Move(off(25, 25)))
		p.expect(t, "touch move", false, offp(5, 5), false)
		p.delta = nil
		tester.Route(touchPointer.Up())
		p.expect(t, "touch up", false, nil, false)
		tester.Route(panZoomPointer.PanZoomEnd())
		p.expect(t, "pan end", false, nil, true)
	})
}

func TestDragTouchAllowsPanZoomsToJoin(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan := NewPanGestureRecognizer()
		competingPan := NewPanGestureRecognizer()
		t.Cleanup(pan.Dispose)
		t.Cleanup(competingPan.Dispose)
		p := newPanProbe(pan)
		panZoomPointer := tp(2, ui.PointerDeviceKindTrackpad)
		touchPointer := touch(3)
		touchDown := touchPointer.Down(off(20, 20))
		pan.AddPointer(touchDown)
		competingPan.AddPointer(touchDown)
		tester.CloseArena(3)
		p.expect(t, "closed", false, nil, false)
		tester.Route(touchPointer.Move(off(60, 60)))
		p.expect(t, "move", true, nil, false)
		p.didStart = false
		tester.Route(touchPointer.Move(off(70, 70)))
		p.expect(t, "move 2", false, offp(10, 10), false)
		p.delta = nil
		start := panZoomPointer.PanZoomStart(off(10, 10))
		pan.AddPointerPanZoom(start)
		competingPan.AddPointerPanZoom(start)
		tester.CloseArena(2)
		p.expect(t, "pz closed", false, nil, false)
		tester.Route(start)
		p.expect(t, "pz start", false, nil, false)
		tester.Route(panUpdate(panZoomPointer, off(20, 20)))
		p.expect(t, "pz update", false, offp(20, 20), false)
		p.delta = nil
		tester.Route(panUpdate(panZoomPointer, off(30, 30)))
		p.expect(t, "pz update 2", false, offp(10, 10), false)
		p.delta = nil
		tester.Route(panZoomPointer.PanZoomEnd())
		p.expect(t, "pz end", false, nil, false)
		tester.Route(touchPointer.Up())
		p.expect(t, "touch up", false, nil, true)
	})
}

func TestDragMultiplePointersDownLatestPointer(t *testing.T) {
	s := MultitouchDragStrategyLatestPointer
	log := multiPointerScript(t, &s, func(tester *flutter_test.GestureTester, p5, p6 *flutter_test.TestPointer, mark func(string)) {
		tester.Route(p5.Move(off(0, 100)))
		mark("-e")
		tester.Route(p5.Move(off(70, 70)))
		mark("-f")
		tester.Route(p6.Move(off(0, 100)))
		mark("-g")
		tester.Route(p6.Move(off(70, 70)))
		mark("-h")
		pointer7 := touch(7)
		down7 := pointer7.Down(off(20, 20))
		multiPointerAdd(down7)
		tester.CloseArena(7)
		tester.Route(down7)
		mark("-i")
		tester.Route(pointer7.Up())
		mark("-j")
		tester.Route(p5.Move(off(0, 100)))
		mark("-k")
		tester.Route(p5.Move(off(70, 70)))
		mark("-l")
		tester.Route(p5.Up())
		tester.Route(p6.Up())
	})
	dexpect(t, "log", log, append(append([]string{}, multiPointerPrefix...), "-e", "-f", "drag1-update", "-g",
		"drag1-update", "-h", "drag2-down", "drag2-cancel", "-i", "-j", "drag1-update", "-k", "drag1-update", "-l",
		"drag1-end"))
}

func TestDragShouldRecognizePan(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		pan := NewPanGestureRecognizer()
		tap := NewTapGestureRecognizer()
		t.Cleanup(pan.Dispose)
		t.Cleanup(tap.Dispose)
		didStartPan, didEndPan, didTap := false, false, false
		var updatedScrollDelta *ui.Offset
		pan.OnStart = func(*DragStartDetails) { didStartPan = true }
		pan.OnUpdate = func(d *DragUpdateDetails) { delta := d.Delta; updatedScrollDelta = &delta }
		pan.OnEnd = func(*DragEndDetails) { didEndPan = true }
		tap.OnTap = func() { didTap = true }
		check := func(step string, start bool, delta *ui.Offset, end bool) {
			t.Helper()
			if didStartPan != start || didEndPan != end || didTap || (updatedScrollDelta == nil) != (delta == nil) ||
				(delta != nil && *updatedScrollDelta != *delta) {
				t.Fatalf("%s: start=%v delta=%v end=%v tap=%v", step, didStartPan, updatedScrollDelta, didEndPan, didTap)
			}
		}
		pointer := testPointerN(5)
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tap.AddPointer(down)
		tester.CloseArena(5)
		check("closed", false, nil, false)
		tester.Route(down)
		check("down", false, nil, false)
		tester.Route(pointer.Move(off(20, 20)))
		check("14px", false, nil, false)
		tester.Route(pointer.Move(off(20, 30)))
		if !didStartPan {
			t.Fatal("pan not started at 22px")
		}
		didStartPan = false
		updatedScrollDelta = nil
		tester.Route(pointer.Move(off(20, 25)))
		check("move", false, offp(0, -5), false)
		updatedScrollDelta = nil
		tester.Route(pointer.Up())
		check("up", false, nil, true)
	})
}

// buttonCompetition is the group "Recognizers listening on different
// buttons do not form competition".
func buttonCompetition(t *testing.T) (tapPrimary, tapSecondary *TapGestureRecognizer, pan *PanGestureRecognizer, recognized *[]string) {
	recognized = &[]string{}
	tapPrimary = NewTapGestureRecognizer()
	tapPrimary.OnTapDown = func(*TapDownDetails) { *recognized = append(*recognized, "tapPrimary") }
	tapSecondary = NewTapGestureRecognizer()
	tapSecondary.OnSecondaryTapDown = func(*TapDownDetails) { *recognized = append(*recognized, "tapSecondary") }
	pan = NewPanGestureRecognizer()
	pan.OnStart = func(*DragStartDetails) { *recognized = append(*recognized, "drag") }
	t.Cleanup(func() { tapPrimary.Dispose(); tapSecondary.Dispose(); pan.Dispose() })
	return
}

func TestDragPrimaryPanDoesNotCompeteWithSecondaryTap(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		_, tapSecondary, pan, recognized := buttonCompetition(t)
		pointer := flutter_test.NewTestPointer(flutter_test.TestPointerOptions{Pointer: intp(1), Device: intp(0),
			Buttons: intp(KSecondaryButton)})
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tapSecondary.AddPointer(down)
		tester.CloseArena(down.Pointer())
		tester.Route(down)
		dexpect(t, "recognized", *recognized, []string{"tapSecondary"})
	})
}

func TestDragPrimaryPanCompetesWithPrimaryTap(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		tapPrimary, _, pan, recognized := buttonCompetition(t)
		// TestPointer(1, touch, kPrimaryButton): the third positional
		// parameter is the device.
		pointer := flutter_test.NewTestPointer(flutter_test.TestPointerOptions{Pointer: intp(1), Device: intp(KPrimaryButton)})
		down := pointer.Down(off(10, 10))
		pan.AddPointer(down)
		tapPrimary.AddPointer(down)
		tester.CloseArena(down.Pointer())
		tester.Route(down)
		dexpect(t, "recognized", *recognized, []string{})
		tester.Route(pointer.Up())
		dexpect(t, "recognized", *recognized, []string{"tapPrimary"})
	})
}

// multiPointerLogs builds the "On multiple pointers" tests' logged
// horizontal drag and competing tap.
func multiPointerLogs(t *testing.T) (*HorizontalDragGestureRecognizer, *TapGestureRecognizer, *[]string) {
	logs := &[]string{}
	add := func(s string) { *logs = append(*logs, s) }
	hori := NewHorizontalDragGestureRecognizer()
	hori.OnDown = func(*DragDownDetails) { add("downH") }
	hori.OnStart = func(*DragStartDetails) { add("startH") }
	hori.OnUpdate = func(*DragUpdateDetails) { add("updateH") }
	hori.OnEnd = func(*DragEndDetails) { add("endH") }
	hori.OnCancel = func() { add("cancelH") }
	vert := NewTapGestureRecognizer()
	vert.OnTapDown = func(*TapDownDetails) { add("downT") }
	vert.OnTapUp = func(*TapUpDetails) { add("upT") }
	vert.OnTapCancel = func() {}
	t.Cleanup(hori.Dispose)
	t.Cleanup(vert.Dispose)
	return hori, vert, logs
}

func expectAndClear(t *testing.T, logs *[]string, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	dexpect(t, "logs", append([]string{}, *logs...), want)
	*logs = (*logs)[:0]
}

// multiPointerStart adds P1 then P2 to the drag and the tap, routing each
// down before closing its arena, as the Dart tests do.
func multiPointerStart(t *testing.T, tester *flutter_test.GestureTester, hori, vert GestureRecognizer, logs *[]string,
	p1, p2 *flutter_test.TestPointer) {
	down1 := p1.Down(off(10, 10))
	down2 := p2.Down(off(11, 10))
	hori.AddPointer(down1)
	vert.AddPointer(down1)
	tester.Route(down1)
	tester.CloseArena(p1.Pointer)
	expectAndClear(t, logs, "downH")
	hori.AddPointer(down2)
	vert.AddPointer(down2)
	tester.Route(down2)
	tester.CloseArena(p2.Pointer)
	expectAndClear(t, logs)
}

func TestDragMultiplePointersCanceledWhenAllCanceledFIFO(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		hori, vert, logs := multiPointerLogs(t)
		p1, p2 := testPointerN(4), testPointerN(5)
		multiPointerStart(t, tester, hori, vert, logs, p1, p2)
		tester.Route(p1.Up())
		tapArena().Sweep(p1.Pointer)
		expectAndClear(t, logs, "downT", "upT")
		tester.Route(p2.Up())
		tapArena().Sweep(p2.Pointer)
		expectAndClear(t, logs, "cancelH")
	})
}

func TestDragMultiplePointersCanceledWhenAllCanceledFILO(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		hori, vert, logs := multiPointerLogs(t)
		p1, p2 := testPointerN(4), testPointerN(5)
		multiPointerStart(t, tester, hori, vert, logs, p1, p2)
		tester.Route(p2.Up())
		tapArena().Sweep(p2.Pointer)
		expectAndClear(t, logs)
		tester.Route(p1.Up())
		tapArena().Sweep(p1.Pointer)
		expectAndClear(t, logs, "cancelH", "downT", "upT")
	})
}

func TestDragMultiplePointersAcceptedWhenFirstAccepted(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		hori, vert, logs := multiPointerLogs(t)
		p1, p2 := testPointerN(4), testPointerN(5)
		multiPointerStart(t, tester, hori, vert, logs, p1, p2)
		tester.Route(p1.Move(off(100, 100)))
		expectAndClear(t, logs, "startH")
		tester.Route(p2.Up())
		tapArena().Sweep(p2.Pointer)
		expectAndClear(t, logs)
		tester.Route(p1.Up())
		tapArena().Sweep(p1.Pointer)
		expectAndClear(t, logs, "endH")
	})
}

func TestDragMultiplePointersCanceledByUpDoNotPreventLater(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		hori, vert, logs := multiPointerLogs(t)
		p1, p2 := testPointerN(4), testPointerN(5)
		multiPointerStart(t, tester, hori, vert, logs, p1, p2)
		tester.Route(p1.Up())
		tapArena().Sweep(p1.Pointer)
		expectAndClear(t, logs, "downT", "upT")
		tester.Route(p2.Move(off(100, 100)))
		expectAndClear(t, logs, "startH")
		tester.Route(p2.Up())
		tapArena().Sweep(p2.Pointer)
		expectAndClear(t, logs, "endH")
	})
}

func TestDragMultiplePointersCanceledByButtonsDoNotPreventLater(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		hori, vert, logs := multiPointerLogs(t)
		p1, p2 := testPointerN(1), testPointerN(2)
		down1 := p1.Down(off(10, 10))
		down2 := p2.Down(off(11, 10))
		hori.AddPointer(down1)
		vert.AddPointer(down1)
		tester.Route(down1)
		tester.CloseArena(p1.Pointer)
		hori.AddPointer(down2)
		vert.AddPointer(down2)
		tester.Route(down2)
		tester.CloseArena(p2.Pointer)
		expectAndClear(t, logs, "downH")
		tester.Route(p1.Move(off(9.9, 9.9), withButtons(KSecondaryButton)))
		expectAndClear(t, logs, "startH")
		tester.Route(p2.Move(off(100, 100)))
		expectAndClear(t, logs, "updateH")
		tester.Route(p2.Up())
		tapArena().Sweep(p2.Pointer)
		expectAndClear(t, logs, "endH")
	})
}

func TestDragMultiplePointersLastTrackingPointerRejectedByResolvePointer(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		logs := &[]string{}
		add := func(s string) { *logs = append(*logs, s) }
		drag := NewVerticalDragGestureRecognizer()
		drag.OnDown = func(*DragDownDetails) { add("downD") }
		drag.OnStart = func(*DragStartDetails) { add("startD") }
		drag.OnUpdate = func(*DragUpdateDetails) { add("updateD") }
		drag.OnEnd = func(*DragEndDetails) { add("endD") }
		drag.OnCancel = func() { add("cancelD") }
		tap := NewTapGestureRecognizer()
		tap.OnTapDown = func(*TapDownDetails) { add("downT") }
		tap.OnTapUp = func(*TapUpDetails) { add("upT") }
		tap.OnTapCancel = func() {}
		t.Cleanup(tap.Dispose)
		t.Cleanup(drag.Dispose)
		p1, p2, p3, p4 := testPointerN(1), testPointerN(2), testPointerN(3), testPointerN(4)
		down1, down2 := p1.Down(off(10, 10)), p2.Down(off(11, 11))
		down3, down4 := p3.Down(off(12, 12)), p4.Down(off(13, 13))
		tap.AddPointer(down1)
		drag.AddPointer(down1)
		tester.CloseArena(1)
		tester.Route(down1)
		expectAndClear(t, logs, "downD")
		tap.AddPointer(down2)
		drag.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		expectAndClear(t, logs)
		tap.AddPointer(down3)
		drag.AddPointer(down3)
		tester.CloseArena(3)
		tester.Route(down3)
		expectAndClear(t, logs)
		drag.AddPointer(down4)
		tester.CloseArena(4)
		tester.Route(down4)
		expectAndClear(t, logs, "startD")
		tester.Route(p2.Up())
		tapArena().Sweep(2)
		expectAndClear(t, logs)
		tester.Route(p4.Cancel())
		expectAndClear(t, logs)
		tester.Route(p3.Cancel())
		expectAndClear(t, logs)
		tester.Route(p1.Cancel())
		expectAndClear(t, logs, "endD")
	})
}

func TestDragDoesNotCrashWhenOneOfTwoPointersWinsByDefaultAndIsReleased(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		didStartDrag := false
		drag := NewHorizontalDragGestureRecognizer()
		drag.OnStart = func(*DragStartDetails) { didStartDrag = true }
		drag.OnEnd = func(*DragEndDetails) {} // Crash triggers at onEnd.
		drag.DragStartBehavior = DragStartBehaviorDown
		tap := NewTapGestureRecognizer()
		tap.OnTap = func() {}
		tap2 := NewTapGestureRecognizer()
		tap2.OnTap = func() {}
		p1 := testPointerN(5)
		down1 := p1.Down(off(10, 10))
		drag.AddPointer(down1)
		tap.AddPointer(down1)
		tester.CloseArena(p1.Pointer)
		tester.Route(down1)
		p2 := testPointerN(6)
		down2 := p2.Down(off(10, 10))
		drag.AddPointer(down2)
		tap2.AddPointer(down2)
		tester.CloseArena(p2.Pointer)
		tester.Route(down2)
		tap.Dispose()
		tester.Async.FlushMicrotasks()
		if !didStartDrag {
			t.Fatal("drag did not start")
		}
		tester.Route(p1.Up())
		drag.Dispose()
		tap2.Dispose()
	})
}
