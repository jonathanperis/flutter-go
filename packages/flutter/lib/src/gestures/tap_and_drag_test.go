package gestures_test

// Port of packages/flutter/test/gestures/tap_and_drag_test.dart (Flutter
// 3.50.1): all 30 tests (none needs widgets).

import (
	"fmt"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// tdConsecutiveTapDelay is kConsecutiveTapDelay (kDoubleTapTimeout ~/ 2).
const tdConsecutiveTapDelay = KDoubleTapTimeout / 2

// tdSetUp is setUpTapAndPanGestureRecognizer /
// setUpTapAndHorizontalDragGestureRecognizer: the recognizer logs into
// events with its drag name ("pan" or "horizontaldrag").
func tdSetUp(t *testing.T, events *strs, horizontal bool, eagerVictoryOnDrag bool) BaseTapAndDragGestureRecognizer {
	var r BaseTapAndDragGestureRecognizer
	var base *BaseTapAndDragGestureRecognizerBase
	name := "pan"
	if horizontal {
		h := NewTapAndHorizontalDragGestureRecognizer()
		r, base, name = h, &h.BaseTapAndDragGestureRecognizerBase, "horizontaldrag"
	} else {
		p := NewTapAndPanGestureRecognizer()
		r, base = p, &p.BaseTapAndDragGestureRecognizerBase
	}
	base.DragStartBehavior = DragStartBehaviorDown
	base.EagerVictoryOnDrag = eagerVictoryOnDrag
	base.MaxConsecutiveTap = intp(3)
	base.OnTapDown = func(d *TapDragDownDetails) { events.add(fmt.Sprintf("down#%d", d.ConsecutiveTapCount)) }
	base.OnTapUp = func(d *TapDragUpDetails) { events.add(fmt.Sprintf("up#%d", d.ConsecutiveTapCount)) }
	base.OnDragStart = func(d *TapDragStartDetails) { events.add(fmt.Sprintf("%sstart#%d", name, d.ConsecutiveTapCount)) }
	base.OnDragUpdate = func(d *TapDragUpdateDetails) { events.add(fmt.Sprintf("%supdate#%d", name, d.ConsecutiveTapCount)) }
	base.OnDragEnd = func(d *TapDragEndDetails) { events.add(fmt.Sprintf("%send#%d", name, d.ConsecutiveTapCount)) }
	base.OnCancel = func() { events.add("cancel") }
	t.Cleanup(r.Dispose)
	return r
}

var (
	tdDown1   = peDown(1, 10, 10)
	tdUp1     = peUp(1, 11, 9)
	tdCancel1 = NewPointerCancelEvent(PointerCancelEventOptions{Pointer: 1})
	tdDown2   = peDown(2, 12, 12)
	tdUp2     = peUp(2, 13, 11)
	tdDown3   = peDown(3, 12, 12)
	tdUp3     = peUp(3, 13, 11)
	tdDown4   = peDown(4, 130, 130)
	tdUp4     = peUp(4, 131, 129)
	tdDown5   = peDown(5, 10, 10)
	tdMove5   = peMove(5, 25, 25)
	tdUp5     = peUp(5, 25, 25)
	tdDown6   = peDown(6, 10, 10, peOpt{Kind: ui.PointerDeviceKindMouse})
	tdMove6   = peMove(6, 15, 15, peOpt{Kind: ui.PointerDeviceKindMouse, Delta: off(5, 5)})
	tdUp6     = peUp(6, 15, 15, peOpt{Kind: ui.PointerDeviceKindMouse})
)

// tdTap adds down to r, closes the arena, routes down and up, sweeps.
func tdTap(tester *flutter_test.GestureTester, r GestureRecognizer, down *PointerDownEvent, up *PointerUpEvent) {
	r.AddPointer(down)
	tester.CloseArena(down.Pointer())
	tester.Route(down)
	tester.Route(up)
	tapArena().Sweep(down.Pointer())
}

func tdTapAt(tester *flutter_test.GestureTester, rs []GestureRecognizer, p *flutter_test.TestPointer, x, y float64, moves ...ui.Offset) {
	down := p.Down(off(x, y))
	for _, r := range rs {
		r.AddPointer(down)
	}
	tester.CloseArena(p.Pointer)
	tester.Route(down)
	for _, m := range moves {
		tester.Route(p.Move(m))
	}
	tester.Route(p.Up())
}

func TestTapAndDragRecognizesConsecutiveTaps(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		tdTap(tester, r, tdDown1, tdUp1)
		expectLog(t, events, "down#1", "up#1")
		events = nil
		tester.Async.Elapse(tdConsecutiveTapDelay)
		tdTap(tester, r, tdDown2, tdUp2)
		expectLog(t, events, "down#2", "up#2")
		events = nil
		tester.Async.Elapse(tdConsecutiveTapDelay)
		tdTap(tester, r, tdDown3, tdUp3)
		expectLog(t, events, "down#3", "up#3")
	})
}

func TestTapAndDragResetsIfTimesOutInBetweenTaps(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		tdTap(tester, r, tdDown1, tdUp1)
		expectLog(t, events, "down#1", "up#1")
		events = nil
		tester.Async.Elapse(ms(1000))
		tdTap(tester, r, tdDown2, tdUp2)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragResetsIfTapsAreFarApart(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		tdTap(tester, r, tdDown1, tdUp1)
		expectLog(t, events, "down#1", "up#1")
		events = nil
		tester.Async.Elapse(ms(100))
		tdTap(tester, r, tdDown4, tdUp4)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragResetsAtMaxConsecutiveTap(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		tdTap(tester, r, tdDown1, tdUp1)
		expectLog(t, events, "down#1", "up#1")
		events = nil
		tdTap(tester, r, tdDown2, tdUp2)
		expectLog(t, events, "down#2", "up#2")
		events = nil
		tdTap(tester, r, tdDown3, tdUp3)
		expectLog(t, events, "down#3", "up#3")
		events = nil
		tdTap(tester, r, tdDown3, tdUp3)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragShouldRecognizeDrag(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		tdTapAt(tester, []GestureRecognizer{r}, testPointerN(5), 10, 10, off(40, 45))
		tapArena().Sweep(5)
		expectLog(t, events, "down#1", "panstart#1", "panupdate#1", "panend#1")
	})
}

func TestTapAndDragRecognizesConsecutiveTapsPlusDrag(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		p := testPointerN(5)
		tdTapAt(tester, []GestureRecognizer{r}, p, 10, 10)
		tapArena().Sweep(5)
		tester.Async.Elapse(tdConsecutiveTapDelay)
		tdTapAt(tester, []GestureRecognizer{r}, p, 10, 10)
		tapArena().Sweep(5)
		tester.Async.Elapse(tdConsecutiveTapDelay)
		tdTapAt(tester, []GestureRecognizer{r}, p, 10, 10, off(40, 45))
		expectLog(t, events, "down#1", "up#1", "down#2", "up#2", "down#3", "panstart#3", "panupdate#3", "panend#3")
	})
}

// tdTwoPointers runs the FIFO/FILO tests: both downs added (before or after
// the first is routed), then the ups in order.
func tdTwoPointers(t *testing.T, addBoth, upFirstFirst bool) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		r.AddPointer(tdDown1)
		if addBoth {
			r.AddPointer(tdDown2)
		}
		tester.CloseArena(1)
		tester.Route(tdDown1)
		if !addBoth {
			r.AddPointer(tdDown2)
		}
		tester.CloseArena(2)
		tester.Route(tdDown2)
		if upFirstFirst {
			tester.Route(tdUp1)
			tapArena().Sweep(1)
			tester.Route(tdUp2)
			tapArena().Sweep(2)
		} else {
			tester.Route(tdUp2)
			tapArena().Sweep(2)
			tester.Route(tdUp1)
			tapArena().Sweep(1)
		}
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragRejectsNonPrimaryFIFOBeforeAcceptance(t *testing.T) { tdTwoPointers(t, true, true) }
func TestTapAndDragRejectsNonPrimaryFILOBeforeAcceptance(t *testing.T) { tdTwoPointers(t, true, false) }
func TestTapAndDragRejectsNonPrimaryFIFOAfterAcceptance(t *testing.T)  { tdTwoPointers(t, false, true) }
func TestTapAndDragRejectsNonPrimaryFILOAfterAcceptance(t *testing.T)  { tdTwoPointers(t, false, false) }

func TestTapAndDragCallsTapUpWhenAcceptedBeforeHandleEvent(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		r.AddPointer(tdDown1)
		tester.CloseArena(1)
		tapArena().Sweep(1)
		tester.Route(tdDown1)
		tester.Route(tdUp1)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragDetectsTapWithinTolerance(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		tdTap(tester, r, tdDown1, tdUp1)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragDetectsDragPastTapToleranceButNotDragMinimum(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		r.AddPointer(tdDown5)
		tester.CloseArena(5)
		tester.Route(tdDown5)
		tester.Route(tdMove5)
		tester.Route(tdUp5)
		tapArena().Sweep(5)
		expectLog(t, events, "down#1", "panstart#1", "panend#1")
	})
}

func tdTaps(t *testing.T, events *strs) *TapGestureRecognizer {
	taps := NewTapGestureRecognizer()
	taps.OnTapDown = func(*TapDownDetails) { events.add("tapdown") }
	taps.OnTapUp = func(*TapUpDetails) { events.add("tapup") }
	taps.OnTapCancel = func() { events.add("tapscancel") }
	t.Cleanup(taps.Dispose)
	return taps
}

// tdDragLogger sets the drag callbacks of a monodrag recognizer.
func tdDragLogger(t *testing.T, events *strs, prefix string, start *GestureDragStartCallback, update *GestureDragUpdateCallback,
	end *GestureDragEndCallback, cancel *GestureDragCancelCallback, dispose func()) {
	*start = func(*DragStartDetails) { events.add(prefix + "start") }
	*update = func(*DragUpdateDetails) { events.add(prefix + "update") }
	*end = func(*DragEndDetails) { events.add(prefix + "end") }
	*cancel = func() { events.add(prefix + "cancel") }
	t.Cleanup(dispose)
}

func tdPans(t *testing.T, events *strs) *PanGestureRecognizer {
	pans := NewPanGestureRecognizer()
	tdDragLogger(t, events, "pan", &pans.OnStart, &pans.OnUpdate, &pans.OnEnd, &pans.OnCancel, pans.Dispose)
	return pans
}

func tdVertical(t *testing.T, events *strs) *VerticalDragGestureRecognizer {
	v := NewVerticalDragGestureRecognizer()
	tdDragLogger(t, events, "vertical", &v.OnStart, &v.OnUpdate, &v.OnEnd, &v.OnCancel, v.Dispose)
	return v
}

func TestTapAndDragBeatsTapWhenMouseMovesPastPrecisePanSlop(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		taps := tdTaps(t, &events)
		r.AddPointer(tdDown6)
		taps.AddPointer(tdDown6)
		tester.CloseArena(6)
		tester.Route(tdDown6)
		tester.Route(tdMove6)
		tester.Route(tdUp6)
		tapArena().Sweep(6)
		expectLog(t, events, "down#1", "panstart#1", "panupdate#1", "panend#1")
	})
}

func TestTapAndDragDeclaresSelfVictoryInNonEmptyArena(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		pans := tdPans(t, &events)
		tdTapAt(tester, []GestureRecognizer{r, pans}, testPointerN(5), 10, 10, off(40, 45))
		expectLog(t, events, "pancancel", "down#1", "panstart#1", "panupdate#1", "panend#1")
	})
}

func TestTapAndHorizontalDragAcceptsPanWhenArenaWon(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, true, true)
		tdTapAt(tester, []GestureRecognizer{r}, testPointerN(5), 10, 10, off(25, 45))
		expectLog(t, events, "down#1", "horizontaldragstart#1", "horizontaldragupdate#1", "horizontaldragend#1")
	})
}

func TestTapAndDragLosesToVerticalDragOnVerticalDrag(t *testing.T) {
	for _, horizontal := range []bool{true, false} {
		t.Run(fmt.Sprint(horizontal), func(t *testing.T) {
			testGesture(t, func(tester *flutter_test.GestureTester) {
				var events strs
				r := tdSetUp(t, &events, horizontal, true)
				v := tdVertical(t, &events)
				tdTapAt(tester, []GestureRecognizer{r, v}, testPointerN(5), 10, 10, off(10, 45), off(10, 100))
				expectLog(t, events, "verticalstart", "verticalupdate", "verticalend")
			})
		})
	}
}

func TestTapAndHorizontalDragBeatsVerticalDragOnHorizontalDrag(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, true, true)
		v := tdVertical(t, &events)
		tdTapAt(tester, []GestureRecognizer{r, v}, testPointerN(5), 10, 10, off(45, 10))
		expectLog(t, events, "verticalcancel", "down#1", "horizontaldragstart#1", "horizontaldragupdate#1", "horizontaldragend#1")
	})
}

func TestTapAndPanBeatsVerticalDragOnHorizontalPan(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		v := tdVertical(t, &events)
		tdTapAt(tester, []GestureRecognizer{r, v}, testPointerN(5), 10, 10, off(45, 25))
		expectLog(t, events, "verticalcancel", "down#1", "panstart#1", "panupdate#1", "panend#1")
	})
}

func TestTapAndDragLosesToDragWithoutEagerVictory(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, false)
		pans := tdPans(t, &events)
		tdTapAt(tester, []GestureRecognizer{r, pans}, testPointerN(5), 10, 10, off(40, 45))
		expectLog(t, events, "panstart", "panend")
	})
}

func tdLongPress(t *testing.T, events *strs) *LongPressGestureRecognizer {
	lp := NewLongPressGestureRecognizer()
	lp.OnLongPressStart = func(*LongPressStartDetails) { events.add("longpressstart") }
	lp.OnLongPressMoveUpdate = func(*LongPressMoveUpdateDetails) { events.add("longpressmoveupdate") }
	lp.OnLongPressEnd = func(*LongPressEndDetails) { events.add("longpressend") }
	lp.OnLongPressCancel = func() { events.add("longpresscancel") }
	t.Cleanup(lp.Dispose)
	return lp
}

func tdCaptureErrors(t *testing.T) *[]*foundation.FlutterErrorDetails {
	var errs []*foundation.FlutterErrorDetails
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { errs = append(errs, d) }
	t.Cleanup(func() { foundation.FlutterErrorOnError = old })
	return &errs
}

func TestTapAndDragStateResetAfterLosingArena(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, true, false)
		h := NewHorizontalDragGestureRecognizer()
		tdDragLogger(t, &events, "basichorizontal", &h.OnStart, &h.OnUpdate, &h.OnEnd, &h.OnCancel, h.Dispose)
		lp := tdLongPress(t, &events)
		errs := tdCaptureErrors(t)
		p := testPointerN(5)
		tdTapAt(tester, []GestureRecognizer{r, h, lp}, p, 10, 10, off(28.1, 10))
		expectLog(t, events, "basichorizontalstart", "basichorizontalend")
		tdTapAt(tester, []GestureRecognizer{r, h, lp}, p, 10, 10)
		if len(*errs) != 0 {
			t.Fatalf("errors: %v", (*errs)[0].Exception)
		}
	})
}

func TestTapAndDragBeatsLongPressOnConsecutiveTap(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		lp := tdLongPress(t, &events)
		p := testPointerN(5)
		tdTapAt(tester, []GestureRecognizer{r, lp}, p, 10, 10)
		tapArena().Sweep(5)
		tester.Async.Elapse(tdConsecutiveTapDelay)
		downB := p.Down(off(10, 10))
		r.AddPointer(downB)
		lp.AddPointer(downB)
		tester.CloseArena(5)
		tester.Route(downB)
		tester.Async.Elapse(ms(500))
		tester.Route(p.Move(off(40, 45)))
		tester.Route(p.Up())
		expectLog(t, events, "longpresscancel", "down#1", "up#1", "down#2", "panstart#2", "panupdate#2", "panend#2")
	})
}

func TestTapAndDragBeatsTapAndDoubleTapWhenFirstAndNotMoved(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		taps := tdTaps(t, &events)
		doubleTaps := NewDoubleTapGestureRecognizer()
		doubleTaps.OnDoubleTapDown = func(*TapDownDetails) { events.add("doubletapdown") }
		doubleTaps.OnDoubleTap = func() { events.add("doubletapup") }
		doubleTaps.OnDoubleTapCancel = func() { events.add("doubletapcancel") }
		t.Cleanup(doubleTaps.Dispose)
		r.AddPointer(tdDown1)
		taps.AddPointer(tdDown1)
		doubleTaps.AddPointer(tdDown1)
		tester.CloseArena(1)
		tester.Route(tdDown1)
		tester.Route(tdUp1)
		tapArena().Sweep(1)
		tester.Async.Elapse(KDoubleTapTimeout)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragBeatsTapWhenFirstAndNotMoved(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		taps := tdTaps(t, &events)
		r.AddPointer(tdDown1)
		taps.AddPointer(tdDown1)
		tester.CloseArena(1)
		tester.Route(tdDown1)
		tester.Route(tdUp1)
		tapArena().Sweep(1)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragBeatsTapPastSlop(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		taps := tdTaps(t, &events)
		r.AddPointer(tdDown5)
		taps.AddPointer(tdDown5)
		tester.CloseArena(5)
		tester.Route(tdDown5)
		tester.Route(tdMove5)
		tester.Route(tdUp5)
		tapArena().Sweep(5)
		expectLog(t, events, "down#1", "panstart#1", "panend#1")
		events = nil
		tester.Async.Elapse(ms(1000))
		taps.AddPointer(tdDown1)
		r.AddPointer(tdDown1)
		tester.CloseArena(1)
		tester.Route(tdDown1)
		tester.Route(tdUp1)
		tapArena().Sweep(1)
		expectLog(t, events, "tapdown", "tapup")
	})
}

func TestTapAndDragTiesWithPanBelowDragDistance(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		pans := tdPans(t, &events)
		r.AddPointer(tdDown5)
		pans.AddPointer(tdDown5)
		tester.CloseArena(5)
		tester.Route(tdDown5)
		tester.Route(tdMove5)
		tester.Route(tdUp5)
		tapArena().Sweep(5)
		expectLog(t, events, "pancancel")
	})
}

func TestTapAndDragDefaultsToDragPastSlop(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		r.AddPointer(tdDown5)
		tester.CloseArena(5)
		tester.Route(tdDown5)
		tester.Route(tdMove5)
		tester.Route(tdUp5)
		tapArena().Sweep(5)
		expectLog(t, events, "down#1", "panstart#1", "panend#1")
		events = nil
		tester.Async.Elapse(ms(1000))
		tdTap(tester, r, tdDown1, tdUp1)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragFiresCancelAndResetsOnPointerCancel(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := tdSetUp(t, &events, false, true)
		r.AddPointer(tdDown1)
		tester.CloseArena(1)
		tester.Route(tdDown1)
		tester.Route(tdCancel1)
		tapArena().Sweep(1)
		expectLog(t, events, "down#1", "cancel")
		events = nil
		tester.Async.Elapse(ms(100))
		tdTap(tester, r, tdDown2, tdUp2)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestTapAndDragDoesNotCallOnDragEndIfNotProvided(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		r := NewTapAndDragGestureRecognizer()
		r.DragStartBehavior = DragStartBehaviorDown
		r.MaxConsecutiveTap = intp(3)
		r.OnTapDown = func(d *TapDragDownDetails) { events.add(fmt.Sprintf("down#%d", d.ConsecutiveTapCount)) }
		t.Cleanup(r.Dispose)
		errs := tdCaptureErrors(t)
		r.AddPointer(tdDown5)
		tester.CloseArena(5)
		tester.Route(tdDown5)
		tester.Route(tdMove5)
		tester.Route(tdUp5)
		tapArena().Sweep(5)
		expectLog(t, events, "down#1")
		if len(*errs) != 0 {
			t.Fatalf("errors: %v", (*errs)[0].Exception)
		}
		events = nil
		tester.Async.Elapse(ms(1000))
		tdTap(tester, r, tdDown1, tdUp1)
		expectLog(t, events, "down#1")
	})
}

func TestTapAndDragDragEndDetailsPositions(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		var end *TapDragEndDetails
		r := NewTapAndHorizontalDragGestureRecognizer()
		r.DragStartBehavior = DragStartBehaviorDown
		r.EagerVictoryOnDrag = true
		r.MaxConsecutiveTap = intp(3)
		r.OnDragEnd = func(d *TapDragEndDetails) { end = d }
		t.Cleanup(r.Dispose)
		tdTapAt(tester, []GestureRecognizer{r}, testPointerN(5), 10, 10, off(50, 20), off(90, 30), off(120, 45))
		if end == nil || end.GlobalPosition() != off(120, 45) {
			t.Fatalf("end details %v", end)
		}
	})
}
