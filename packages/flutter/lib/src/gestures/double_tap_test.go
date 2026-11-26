package gestures_test

// Port of packages/flutter/test/gestures/double_tap_test.dart (Flutter
// 3.50.1): all tests.

import (
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

// recordingArenaMember is double_tap_test.dart's TestGestureArenaMember.
type recordingArenaMember struct{ accepted, rejected bool }

func (m *recordingArenaMember) AcceptGesture(int) { m.accepted = true }
func (m *recordingArenaMember) RejectGesture(int) { m.rejected = true }

var (
	dDown1 = peDown(1, 10, 10)
	dUp1   = peUp(1, 11, 9)
	dDown2 = peDown(2, 12, 12)
	dUp2   = peUp(2, 13, 11)
	dDown3 = peDown(3, 130, 130)
	dUp3   = peUp(3, 131, 129)
	dDown4 = peDown(4, 10, 10)
	dMove4 = peMove(4, 25, 25)
	dUp4   = peUp(4, 25, 25)
	dDown5 = peDown(5, 10, 10)
	dUp5   = peUp(5, 11, 9)
	dDown6 = peDown(6, 10, 10, peOpt{Buttons: KSecondaryMouseButton})
	dUp6   = peUp(6, 11, 9)
	dDown7 = peDown(7, 10, 10, peOpt{Buttons: KSecondaryMouseButton})
	dUp7   = peUp(7, 11, 9)
)

// dtFixture is the test file's setUp: a double tap recognizer recording
// its callbacks.
type dtFixture struct {
	tap                  *DoubleTapGestureRecognizer
	doubleTapRecognized  bool
	doubleTapDownDetails *TapDownDetails
	doubleTapCanceled    bool
}

func newDTFixture(t *testing.T) *dtFixture {
	f := &dtFixture{tap: NewDoubleTapGestureRecognizer()}
	f.tap.OnDoubleTap = func() {
		if f.doubleTapRecognized {
			t.Error("onDoubleTap twice")
		}
		f.doubleTapRecognized = true
	}
	f.tap.OnDoubleTapDown = func(d *TapDownDetails) {
		if f.doubleTapDownDetails != nil {
			t.Error("onDoubleTapDown twice")
		}
		f.doubleTapDownDetails = d
	}
	f.tap.OnDoubleTapCancel = func() {
		if f.doubleTapCanceled {
			t.Error("onDoubleTapCancel twice")
		}
		f.doubleTapCanceled = true
	}
	t.Cleanup(f.tap.Dispose)
	return f
}

func (f *dtFixture) expectDown(t *testing.T, down *PointerDownEvent) {
	t.Helper()
	if f.doubleTapDownDetails == nil {
		t.Fatal("no doubleTapDownDetails")
	}
	expectDeep(t, f.doubleTapDownDetails.GlobalPosition(), down.Position())
	expectDeep(t, f.doubleTapDownDetails.LocalPosition(), down.LocalPosition())
}

func (f *dtFixture) expectNothing(t *testing.T) {
	t.Helper()
	expectDeep(t, f.doubleTapRecognized, false)
	if f.doubleTapDownDetails != nil {
		t.Fatal("unexpected doubleTapDownDetails")
	}
	expectDeep(t, f.doubleTapCanceled, false)
}

// tapSeq adds, closes, routes down and up, and sweeps pointer.
func tapSeq(tester *flutter_test.GestureTester, r GestureRecognizer, down *PointerDownEvent, up *PointerUpEvent) {
	r.AddPointer(down)
	tester.CloseArena(down.Pointer())
	tester.Route(down)
	tester.Route(up)
	tapArena().Sweep(down.Pointer())
}

func TestDoubleTapShouldRecognizeDoubleTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapSeq(tester, f.tap, dDown1, dUp1)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		tester.Async.Elapse(tapMs(100))
		f.tap.AddPointer(dDown2)
		tester.CloseArena(2)
		f.expectDown(t, dDown2)
		tester.Route(dDown2)
		expectDeep(t, f.doubleTapRecognized, false)
		tester.Route(dUp2)
		expectDeep(t, f.doubleTapRecognized, true)
		tapArena().Sweep(2)
		expectDeep(t, f.doubleTapCanceled, false)
	})
}

func TestDoubleTapShouldRecognizeDoubleTapWithSecondaryButton(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapSecondary := NewDoubleTapGestureRecognizer(GestureRecognizerOptions{
			AllowedButtonsFilter: func(b int) bool { return b == KSecondaryButton }})
		defer tapSecondary.Dispose()
		tapSecondary.OnDoubleTap = func() { f.doubleTapRecognized = true }
		tapSecondary.OnDoubleTapDown = func(d *TapDownDetails) { f.doubleTapDownDetails = d }
		tapSecondary.OnDoubleTapCancel = func() { f.doubleTapCanceled = true }

		tapSeq(tester, tapSecondary, dDown6, dUp6)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		tester.Async.Elapse(tapMs(100))
		tapSecondary.AddPointer(dDown7)
		tester.CloseArena(7)
		f.expectDown(t, dDown7)
		tester.Route(dDown7)
		expectDeep(t, f.doubleTapRecognized, false)
		tester.Route(dUp7)
		expectDeep(t, f.doubleTapRecognized, true)
		tapArena().Sweep(2)
		expectDeep(t, f.doubleTapCanceled, false)
	})
}

func TestDoubleTapInterTapDistanceCancelsDoubleTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapSeq(tester, f.tap, dDown1, dUp1)
		tapSeq(tester, f.tap, dDown3, dUp3)
		f.expectNothing(t)
	})
}

func TestDoubleTapIntraTapDistanceCancelsDoubleTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		f.tap.AddPointer(dDown4)
		tester.CloseArena(4)
		tester.Route(dDown4)
		tester.Route(dMove4)
		tester.Route(dUp4)
		tapArena().Sweep(4)

		f.tap.AddPointer(dDown1)
		tester.CloseArena(1)
		tester.Route(dDown2)
		tester.Route(dUp1)
		tapArena().Sweep(1)
		f.expectNothing(t)
	})
}

func TestDoubleTapInterTapDelayCancelsDoubleTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapSeq(tester, f.tap, dDown1, dUp1)
		tester.Async.Elapse(tapMs(5000))
		tapSeq(tester, f.tap, dDown2, dUp2)
		f.expectNothing(t)
	})
}

func thirdTapIsDoubleTap(t *testing.T, tester *flutter_test.GestureTester, f *dtFixture) {
	t.Helper()
	tester.Async.Elapse(tapMs(100))
	f.tap.AddPointer(dDown5)
	tester.CloseArena(5)
	tester.Route(dDown5)
	expectDeep(t, f.doubleTapRecognized, false)
	f.expectDown(t, dDown5)
	tester.Route(dUp5)
	expectDeep(t, f.doubleTapRecognized, true)
	tapArena().Sweep(5)
	expectDeep(t, f.doubleTapCanceled, false)
}

func TestDoubleTapInterTapDelayResetsDoubleTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapSeq(tester, f.tap, dDown1, dUp1)
		tester.Async.Elapse(tapMs(5000))
		tapSeq(tester, f.tap, dDown2, dUp2)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		thirdTapIsDoubleTap(t, tester, f)
	})
}

func TestDoubleTapIntraTapDelayDoesNotCancelDoubleTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		f.tap.AddPointer(dDown1)
		tester.CloseArena(1)
		tester.Route(dDown1)
		tester.Async.Elapse(tapMs(1000))
		tester.Route(dUp1)
		tapArena().Sweep(1)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		f.tap.AddPointer(dDown2)
		tester.CloseArena(2)
		tester.Route(dDown2)
		expectDeep(t, f.doubleTapRecognized, false)
		f.expectDown(t, dDown2)
		tester.Route(dUp2)
		expectDeep(t, f.doubleTapRecognized, true)
		tapArena().Sweep(2)
		expectDeep(t, f.doubleTapCanceled, false)
	})
}

func overlappingTaps(tester *flutter_test.GestureTester, f *dtFixture) {
	f.tap.AddPointer(dDown1)
	tester.CloseArena(1)
	tester.Route(dDown1)
	f.tap.AddPointer(dDown2)
	tester.CloseArena(2)
	tester.Route(dDown1)
	tester.Route(dUp1)
	tapArena().Sweep(1)
	tester.Route(dUp2)
	tapArena().Sweep(2)
}

func TestDoubleTapShouldNotRecognizeTwoOverlappingTaps(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		overlappingTaps(tester, f)
		f.expectNothing(t)
	})
}

func TestDoubleTapShouldRecognizeOneTapOfGroupFollowedBySecondTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		overlappingTaps(tester, f)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		tester.Async.Elapse(tapMs(100))
		f.tap.AddPointer(dDown1)
		tester.CloseArena(1)
		tester.Route(dDown1)
		expectDeep(t, f.doubleTapRecognized, false)
		f.expectDown(t, dDown1)
		tester.Route(dUp1)
		expectDeep(t, f.doubleTapRecognized, true)
		tapArena().Sweep(1)
		expectDeep(t, f.doubleTapCanceled, false)
	})
}

func TestDoubleTapShouldCancelOnArenaRejectDuringFirstTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		f.tap.AddPointer(dDown1)
		member := &recordingArenaMember{}
		entry := tapArena().Add(1, member)
		tester.CloseArena(1)
		tester.Route(dDown1)
		tester.Route(dUp1)
		entry.Resolve(GestureDispositionAccepted)
		expectDeep(t, member.accepted, true)
		tapArena().Sweep(1)

		tapSeq(tester, f.tap, dDown2, dUp2)
		f.expectNothing(t)
	})
}

func TestDoubleTapShouldCancelOnArenaRejectBetweenTaps(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		f.tap.AddPointer(dDown1)
		member := &recordingArenaMember{}
		entry := tapArena().Add(1, member)
		tester.CloseArena(1)
		tester.Route(dDown1)
		tester.Route(dUp1)
		tapArena().Sweep(1)
		entry.Resolve(GestureDispositionAccepted)
		expectDeep(t, member.accepted, true)

		tapSeq(tester, f.tap, dDown2, dUp2)
		f.expectNothing(t)
	})
}

func TestDoubleTapShouldCancelOnArenaRejectDuringLastTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		f.tap.AddPointer(dDown1)
		member := &recordingArenaMember{}
		entry := tapArena().Add(1, member)
		tester.CloseArena(1)
		tester.Route(dDown1)
		tester.Route(dUp1)
		tapArena().Sweep(1)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		tester.Async.Elapse(tapMs(100))
		f.tap.AddPointer(dDown2)
		tester.CloseArena(2)
		tester.Route(dDown2)
		f.expectDown(t, dDown2)
		expectDeep(t, f.doubleTapCanceled, false)

		entry.Resolve(GestureDispositionAccepted)
		expectDeep(t, member.accepted, true)
		expectDeep(t, f.doubleTapCanceled, true)

		tester.Route(dUp2)
		tapArena().Sweep(2)
		expectDeep(t, f.doubleTapRecognized, false)
	})
}

func TestDoubleTapPassiveGestureShouldTriggerOnDoubleTapCancel(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		// FakeAsync().run: a fake event loop of its own.
		loop := async.NewLoop()
		restore := async.SetCurrent(loop)
		defer restore()
		f.tap.AddPointer(dDown1)
		member := &recordingArenaMember{}
		tapArena().Add(1, member)
		tester.CloseArena(1)
		tester.Route(dDown1)
		tester.Route(dUp1)
		tapArena().Sweep(1)

		expectDeep(t, member.accepted, false)
		loop.Elapse(tapMs(5000))
		expectDeep(t, member.accepted, true)
		f.expectNothing(t)
	})
}

func TestDoubleTapShouldNotRecognizeTwoOverRapidTaps(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapSeq(tester, f.tap, dDown1, dUp1)
		tester.Async.Elapse(tapMs(10))
		tapSeq(tester, f.tap, dDown2, dUp2)
		f.expectNothing(t)
	})
}

func TestDoubleTapOverRapidTapsResetsDoubleTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapSeq(tester, f.tap, dDown1, dUp1)
		tester.Async.Elapse(tapMs(10))
		tapSeq(tester, f.tap, dDown2, dUp2)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		thirdTapIsDoubleTap(t, tester, f)
	})
}

func TestDoubleTapButtonChangeShouldInterruptExistingSequence(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		interval := tapMs(100)
		tapSeq(tester, f.tap, dDown1, dUp1)
		tester.Async.Elapse(interval)
		tapSeq(tester, f.tap, dDown6, dUp6)
		tester.Async.Elapse(interval)
		expectDeep(t, f.doubleTapRecognized, false)
		tapSeq(tester, f.tap, dDown2, dUp2)
		f.expectNothing(t)
	})
}

func TestDoubleTapButtonChangeWithAllowedButtonsFilterShouldInterruptExistingSequence(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		tapPrimary := NewDoubleTapGestureRecognizer(GestureRecognizerOptions{
			AllowedButtonsFilter: func(b int) bool { return b == KPrimaryButton }})
		defer tapPrimary.Dispose()
		tapPrimary.OnDoubleTap = func() { f.doubleTapRecognized = true }
		tapPrimary.OnDoubleTapDown = func(d *TapDownDetails) { f.doubleTapDownDetails = d }
		tapPrimary.OnDoubleTapCancel = func() { f.doubleTapCanceled = true }

		interval := tapMs(100)
		tapSeq(tester, tapPrimary, dDown1, dUp1)
		tester.Async.Elapse(interval)
		tapSeq(tester, tapPrimary, dDown6, dUp6)
		tester.Async.Elapse(interval)
		expectDeep(t, f.doubleTapRecognized, false)
		tapSeq(tester, tapPrimary, dDown2, dUp2)
		f.expectNothing(t)
	})
}

func TestDoubleTapButtonChangeShouldStartAValidSequence(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		f := newDTFixture(t)
		interval := tapMs(100)
		tapSeq(tester, f.tap, dDown6, dUp6)
		tester.Async.Elapse(interval)
		tapSeq(tester, f.tap, dDown1, dUp1)
		expectDeep(t, f.doubleTapRecognized, false)
		if f.doubleTapDownDetails != nil {
			t.Fatal("details")
		}
		tester.Async.Elapse(interval)

		f.tap.AddPointer(dDown2)
		tester.CloseArena(2)
		tester.Route(dDown2)
		f.expectDown(t, dDown2)
		tester.Route(dUp2)
		tapArena().Sweep(2)
		expectDeep(t, f.doubleTapRecognized, true)
		expectDeep(t, f.doubleTapCanceled, false)
	})
}

type dtCompetition struct {
	tapPrimary, tapSecondary *TapGestureRecognizer
	doubleTap                *DoubleTapGestureRecognizer
}

func newDTCompetition(t *testing.T, recognized *strs) dtCompetition {
	c := dtCompetition{NewTapGestureRecognizer(), NewTapGestureRecognizer(), NewDoubleTapGestureRecognizer()}
	c.tapPrimary.OnTapDown = func(*TapDownDetails) { recognized.add("tapPrimary") }
	c.tapSecondary.OnSecondaryTapDown = func(*TapDownDetails) { recognized.add("tapSecondary") }
	c.doubleTap.OnDoubleTap = func() { recognized.add("doubleTap") }
	t.Cleanup(func() {
		c.tapPrimary.Dispose()
		c.tapSecondary.Dispose()
		c.doubleTap.Dispose()
	})
	return c
}

func TestDoubleTapPrimaryDoesNotCompeteWithSecondaryTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		newDTFixture(t)
		var recognized strs
		c := newDTCompetition(t, &recognized)
		c.doubleTap.AddPointer(dDown6)
		c.tapSecondary.AddPointer(dDown6)
		tester.CloseArena(6)
		tester.Route(dDown6)
		expectLog(t, recognized, "tapSecondary")
	})
}

func TestDoubleTapPrimaryFormsCompetitionWithPrimaryTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		newDTFixture(t)
		var recognized strs
		c := newDTCompetition(t, &recognized)
		c.doubleTap.AddPointer(dDown1)
		c.tapPrimary.AddPointer(dDown1)
		tester.CloseArena(1)
		tester.Route(dDown1)
		expectLog(t, recognized)
		tester.Async.Elapse(tapMs(300))
		expectLog(t, recognized, "tapPrimary")
	})
}

func TestDoubleTapSecondaryDoubleTapShouldNotTriggerPrimary(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		newDTFixture(t)
		var recognized strs
		doubleTap := NewDoubleTapGestureRecognizer()
		doubleTap.OnDoubleTap = func() { recognized.add("primary") }
		defer doubleTap.Dispose()
		tapSeq(tester, doubleTap, dDown6, dUp6)
		tester.Async.Elapse(tapMs(100))
		doubleTap.AddPointer(dDown7)
		tester.CloseArena(7)
		tester.Route(dDown7)
		tester.Route(dUp7)
		expectLog(t, recognized)
	})
}

func TestDoubleTapButtonsFilterShouldCancelInvalidTaps(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		newDTFixture(t)
		var recognized strs
		doubleTap := NewDoubleTapGestureRecognizer(GestureRecognizerOptions{AllowedButtonsFilter: func(int) bool { return false }})
		doubleTap.OnDoubleTap = func() { recognized.add("primary") }
		defer doubleTap.Dispose()
		down7, up7 := peDown(7, 10, 10), peUp(7, 11, 9)
		tapSeq(tester, doubleTap, down7, up7)
		tester.Async.Elapse(tapMs(100))
		doubleTap.AddPointer(dDown6)
		tester.CloseArena(6)
		tester.Route(dDown6)
		tester.Route(dUp6)
		expectLog(t, recognized)
	})
}

func TestDoubleTapUnfinishedDoubleTapDoesNotPreventCompetingTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		newDTFixture(t)
		tapCount := 0
		doubleTap := NewDoubleTapGestureRecognizer()
		doubleTap.OnDoubleTap = func() {}
		defer doubleTap.Dispose()
		tap := NewTapGestureRecognizer()
		tap.OnTap = func() { tapCount++ }
		defer tap.Dispose()

		// Open a arena with 2 members and holding.
		doubleTap.AddPointer(dDown1)
		tap.AddPointer(dDown1)
		tester.CloseArena(1)
		tester.Route(dDown1)
		tester.Route(dUp1)
		tapArena().Sweep(1)

		// Open a new arena with only one TapGestureRecognizer.
		tester.Async.Elapse(tapMs(100))
		tap.AddPointer(dDown2)
		tester.CloseArena(2)
		tester.Route(dDown2)
		tester.Route(NewPointerMoveEvent(PointerMoveEventOptions{Pointer: 2, Position: dDown2.Position()}))
		tester.Route(dUp2)
		expectDeep(t, tapCount, 1) // The second tap will win immediately.
		tapArena().Sweep(2)

		// Finish the previous gesture arena.
		tester.Async.Elapse(tapMs(300))
		expectDeep(t, tapCount, 1) // The first tap should not trigger onTap though it wins the arena.
	})
}
