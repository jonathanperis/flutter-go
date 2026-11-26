package gestures_test

// Port of packages/flutter/test/gestures/serial_tap_test.dart (Flutter
// 3.50.1): all tests, with its WinningGestureRecognizer and
// ReleaseGestureRecognizer.

import (
	"fmt"
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// kSerialTapDelay: anything longer than kDoubleTapTimeout resets the
// serial tap count.
const kSerialTapDelay = KDoubleTapTimeout / 2

var (
	sDown1   = peDown(1, 10, 10)
	sCancel1 = peCancel(1, 0, 0)
	sUp1     = peUp(1, 11, 9)
	sDown2   = peDown(2, 12, 12)
	sUp2     = peUp(2, 13, 11)
	sDown3   = peDown(3, 12, 12)
	sUp3     = peUp(3, 13, 11)
	sDown4   = peDown(4, 130, 130)
	sUp4     = peUp(4, 131, 129)
	sDown5   = peDown(5, 10, 10)
	sMove5   = peMove(5, 25, 25)
	sUp5     = peUp(5, 25, 25)
	sDown6   = peDown(6, 10, 10, peOpt{Buttons: KSecondaryMouseButton})
	sUp6     = peUp(6, 11, 9)
)

type winningGestureRecognizer struct {
	PrimaryPointerGestureRecognizerBase
}

func newWinningGestureRecognizer() *winningGestureRecognizer {
	r := &winningGestureRecognizer{}
	r.Construct(r, PrimaryPointerGestureRecognizerOptions{})
	return r
}

func (r *winningGestureRecognizer) DebugDescription() string { return "winner" }
func (r *winningGestureRecognizer) HandlePrimaryPointer(PointerEvent) {
	r.Resolve(GestureDispositionAccepted)
}

type releaseGestureRecognizer struct {
	PrimaryPointerGestureRecognizerBase
	onRelease func()
}

func newReleaseGestureRecognizer() *releaseGestureRecognizer {
	r := &releaseGestureRecognizer{}
	r.Construct(r, PrimaryPointerGestureRecognizerOptions{})
	return r
}

func (r *releaseGestureRecognizer) DebugDescription() string { return "release" }
func (r *releaseGestureRecognizer) HandlePrimaryPointer(event PointerEvent) {
	if _, ok := event.(*PointerUpEvent); ok {
		r.Resolve(GestureDispositionAccepted)
		if r.onRelease != nil {
			r.onRelease()
		}
	}
}

// newSerial is the test file's setUp.
func newSerial(t *testing.T, events *strs) *SerialTapGestureRecognizer {
	serial := NewSerialTapGestureRecognizer()
	serial.OnSerialTapDown = func(d *SerialTapDownDetails) { events.add(fmt.Sprintf("down#%d", d.Count)) }
	serial.OnSerialTapCancel = func(d *SerialTapCancelDetails) { events.add(fmt.Sprintf("cancel#%d", d.Count)) }
	serial.OnSerialTapUp = func(d *SerialTapUpDetails) { events.add(fmt.Sprintf("up#%d", d.Count)) }
	t.Cleanup(serial.Dispose)
	return serial
}

// seq adds down to the recognizers in order, closes the arena, routes down
// and end, and sweeps.
func seq(tester *flutter_test.GestureTester, rs []GestureRecognizer, down *PointerDownEvent, end ...PointerEvent) {
	for _, r := range rs {
		r.AddPointer(down)
	}
	tester.CloseArena(down.Pointer())
	tester.Route(down)
	for _, e := range end {
		tester.Route(e)
	}
	tapArena().Sweep(down.Pointer())
}

func gr(r ...GestureRecognizer) []GestureRecognizer { return r }

func TestSerialTapRecognizesSerialTaps(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		seq(tester, gr(serial), sDown1, sUp1)
		expectLog(t, events, "down#1", "up#1")
		events = nil
		tester.Async.Elapse(kSerialTapDelay)
		seq(tester, gr(serial), sDown2, sUp2)
		expectLog(t, events, "down#2", "up#2")
		events = nil
		tester.Async.Elapse(kSerialTapDelay)
		seq(tester, gr(serial), sDown3, sUp3)
		expectLog(t, events, "down#3", "up#3")
	})
}

func testSerialWinsOverTap(t *testing.T, serialFirst bool) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		recognizedSingleTap, canceledSingleTap := false, false
		singleTap := NewTapGestureRecognizer()
		singleTap.OnTap = func() { recognizedSingleTap = true }
		singleTap.OnTapCancel = func() { canceledSingleTap = true }
		defer singleTap.Dispose()

		if serialFirst {
			serial.AddPointer(sDown1)
			singleTap.AddPointer(sDown1)
		} else {
			singleTap.AddPointer(sDown1)
			serial.AddPointer(sDown1)
		}
		tester.CloseArena(1)
		tester.Route(sDown1)
		tester.Async.Elapse(KPressTimeout) // To register the possible single tap.
		tester.Route(sUp1)
		tapArena().Sweep(1)
		expectLog(t, events, "down#1", "up#1")
		expectDeep(t, recognizedSingleTap, false)
		expectDeep(t, canceledSingleTap, true)
	})
}

func TestSerialTapWinsOverTapGestureBelowIt(t *testing.T) { testSerialWinsOverTap(t, false) }
func TestSerialTapWinsOverTapGestureAboveIt(t *testing.T) { testSerialWinsOverTap(t, true) }

func TestSerialTapLosesToReleaseGestureBelowIt(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		recognizedRelease := false
		release := newReleaseGestureRecognizer()
		release.onRelease = func() { recognizedRelease = true }
		defer release.Dispose()
		seq(tester, gr(release, serial), sDown1, sUp1)
		expectLog(t, events, "down#1", "cancel#1")
		expectDeep(t, recognizedRelease, true)
	})
}

func TestSerialTapWinsOverReleaseGestureAboveIt(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		recognizedRelease := false
		release := newReleaseGestureRecognizer()
		release.onRelease = func() { recognizedRelease = true }
		defer release.Dispose()
		seq(tester, gr(serial, release), sDown1, sUp1)
		expectLog(t, events, "down#1", "up#1")
		expectDeep(t, recognizedRelease, false)
	})
}

func TestSerialTapFiresCancelIfCompetingRecognizerDeclaresVictory(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		winner := newWinningGestureRecognizer()
		defer winner.Dispose()
		seq(tester, gr(winner, serial), sDown1, sUp1)
		expectLog(t, events, "down#1", "cancel#1")
	})
}

func testSerialWinsOverDoubleTap(t *testing.T, serialFirst bool) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		recognizedDoubleTap := false
		doubleTap := NewDoubleTapGestureRecognizer()
		doubleTap.OnDoubleTap = func() { recognizedDoubleTap = true }
		defer doubleTap.Dispose()
		order := gr(doubleTap, serial)
		if serialFirst {
			order = gr(serial, doubleTap)
		}

		seq(tester, order, sDown1, sUp1)
		expectLog(t, events, "down#1", "up#1")
		expectDeep(t, recognizedDoubleTap, false)

		events = nil
		tester.Async.Elapse(kSerialTapDelay)
		seq(tester, order, sDown2, sUp2)
		expectLog(t, events, "down#2", "up#2")
		expectDeep(t, recognizedDoubleTap, false)

		events = nil
		tester.Async.Elapse(kSerialTapDelay)
		if serialFirst {
			seq(tester, order, sDown3, sUp3)
		} else {
			seq(tester, gr(serial), sDown3, sUp3)
		}
		expectLog(t, events, "down#3", "up#3")
	})
}

func TestSerialTapWinsOverDoubleTapBelowIt(t *testing.T) { testSerialWinsOverDoubleTap(t, false) }
func TestSerialTapWinsOverDoubleTapAboveIt(t *testing.T) { testSerialWinsOverDoubleTap(t, true) }

func TestSerialTapFiresCancelAndResetsForPointerCancelEvent(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		seq(tester, gr(serial), sDown1, sCancel1)
		expectLog(t, events, "down#1", "cancel#1")
		events = nil
		tester.Async.Elapse(tapMs(100))
		seq(tester, gr(serial), sDown2, sUp2)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestSerialTapFiresCancelAndResetsWhenDraggedPastSlop(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		seq(tester, gr(serial), sDown5, sMove5, sUp5)
		expectLog(t, events, "down#1", "cancel#1")
		events = nil
		tester.Async.Elapse(tapMs(1000))
		seq(tester, gr(serial), sDown1, sUp1)
		expectLog(t, events, "down#1", "up#1")
	})
}

func testSerialResets(t *testing.T, delay int, down *PointerDownEvent, up *PointerUpEvent) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		seq(tester, gr(serial), sDown1, sUp1)
		expectLog(t, events, "down#1", "up#1")
		events = nil
		tester.Async.Elapse(tapMs(delay))
		seq(tester, gr(serial), down, up)
		expectLog(t, events, "down#1", "up#1")
	})
}

func TestSerialTapResetsIfTimesOutInBetweenTaps(t *testing.T) {
	testSerialResets(t, 1000, sDown2, sUp2)
}
func TestSerialTapResetsIfTapsAreFarApart(t *testing.T) { testSerialResets(t, 100, sDown4, sUp4) }
func TestSerialTapDifferentButtonsStartANewSequence(t *testing.T) {
	testSerialResets(t, 1000, sDown6, sUp6)
}

func TestSerialTapInterleavingTapsCancelFirstSequence(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		serial.AddPointer(sDown1)
		tester.CloseArena(1)
		tester.Route(sDown1)
		serial.AddPointer(sDown2)
		tester.CloseArena(2)
		tester.Route(sDown2)
		tester.Route(sUp1)
		tapArena().Sweep(1)
		tester.Route(sUp2)
		tapArena().Sweep(2)
		expectLog(t, events, "down#1", "cancel#1", "down#1", "up#1")
	})
}

func TestSerialTapIsNoOpIfNoCallbacksAreSpecified(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		newSerial(t, &events)
		serial := NewSerialTapGestureRecognizer()
		defer serial.Dispose()
		serial.AddPointer(sDown1)
		tester.CloseArena(1)
		tester.Route(sDown1)
		expectDeep(t, serial.IsTrackingPointer(), false)
		tester.Route(sUp1)
		tapArena().Sweep(1)
		expectLog(t, events)
	})
}

func TestSerialTapWorksForNonPrimaryButton(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var events strs
		serial := newSerial(t, &events)
		seq(tester, gr(serial), sDown6, sUp6)
		expectLog(t, events, "down#1", "up#1")
	})
}
