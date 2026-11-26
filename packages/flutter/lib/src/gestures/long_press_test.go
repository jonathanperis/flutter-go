package gestures_test

// Port of packages/flutter/test/gestures/long_press_test.dart (Flutter
// 3.50.1).
//
// Skipped: "Long press: Drag start delayed by microtask" (needs
// HorizontalDragGestureRecognizer, monodrag.dart, ported separately).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

var (
	lpDown  = peDown(5, 10, 10)
	lpUp    = peUp(5, 11, 9)
	lpMove  = peMove(5, 100, 200)
	lpDown2 = peDown(6, 10, 10)
	lpUp2   = peUp(6, 11, 9)
	lpDown3 = peDown(7, 30, 30, peOpt{Buttons: KSecondaryButton})
)

// lpHandlers is the "Long press" and "long press drag" groups'
// setUpHandlers.
func lpHandlers(gesture *LongPressGestureRecognizer, recognized *strs, dragUpdate *ui.Offset) {
	gesture.OnLongPressDown = func(*LongPressDownDetails) { recognized.add("down") }
	gesture.OnLongPressCancel = func() { recognized.add("cancel") }
	gesture.OnLongPress = func() { recognized.add("start") }
	gesture.OnLongPressMoveUpdate = func(d *LongPressMoveUpdateDetails) {
		recognized.add("move")
		if dragUpdate != nil {
			*dragUpdate = d.GlobalPosition()
		}
	}
	gesture.OnLongPressUp = func() { recognized.add("end") }
}

func newLP(recognized *strs, opts ...LongPressGestureRecognizerOptions) *LongPressGestureRecognizer {
	g := NewLongPressGestureRecognizer(opts...)
	lpHandlers(g, recognized, nil)
	return g
}

func TestLongPressShouldRecognizeLongPress(t *testing.T) {
	for _, group := range []string{"Long press", "long press drag"} {
		t.Run(group, func(t *testing.T) {
			tapTestGesture(t, func(tester *flutter_test.GestureTester) {
				var recognized strs
				gesture := newLP(&recognized)
				gesture.AddPointer(lpDown)
				tester.CloseArena(5)
				expectLog(t, recognized)
				tester.Route(lpDown)
				expectLog(t, recognized, "down")
				tester.Async.Elapse(tapMs(300))
				expectLog(t, recognized, "down")
				tester.Async.Elapse(tapMs(700))
				expectLog(t, recognized, "down", "start")
				gesture.Dispose()
				expectLog(t, recognized, "down", "start")
			})
		})
	}
}

func TestLongPressShouldRecognizeLongPressWithAlteredDuration(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		d := tapMs(100)
		gesture := newLP(&recognized, LongPressGestureRecognizerOptions{Duration: &d})
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		expectLog(t, recognized)
		tester.Route(lpDown)
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(50))
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(50))
		expectLog(t, recognized, "down", "start")
		gesture.Dispose()
		expectLog(t, recognized, "down", "start")
	})
}

func TestLongPressUpCancelsLongPress(t *testing.T) {
	for _, group := range []string{"Long press", "long press drag"} {
		t.Run(group, func(t *testing.T) {
			tapTestGesture(t, func(tester *flutter_test.GestureTester) {
				var recognized strs
				gesture := newLP(&recognized)
				gesture.AddPointer(lpDown)
				tester.CloseArena(5)
				expectLog(t, recognized)
				tester.Route(lpDown)
				expectLog(t, recognized, "down")
				tester.Async.Elapse(tapMs(300))
				expectLog(t, recognized, "down")
				tester.Route(lpUp)
				expectLog(t, recognized, "down", "cancel")
				tester.Async.Elapse(tapMs(1000))
				expectLog(t, recognized, "down", "cancel")
				gesture.Dispose()
				expectLog(t, recognized, "down", "cancel")
			})
		})
	}
}

func TestLongPressMovingBeforeAcceptCancels(t *testing.T) {
	for _, group := range []string{"Long press", "long press drag"} {
		t.Run(group, func(t *testing.T) {
			tapTestGesture(t, func(tester *flutter_test.GestureTester) {
				var recognized strs
				gesture := newLP(&recognized)
				gesture.AddPointer(lpDown)
				tester.CloseArena(5)
				expectLog(t, recognized)
				tester.Route(lpDown)
				expectLog(t, recognized, "down")
				tester.Async.Elapse(tapMs(300))
				expectLog(t, recognized, "down")
				tester.Route(lpMove)
				expectLog(t, recognized, "down", "cancel")
				tester.Async.Elapse(tapMs(1000))
				tester.Route(lpUp)
				tester.Async.Elapse(tapMs(300))
				expectLog(t, recognized, "down", "cancel")
				gesture.Dispose()
				expectLog(t, recognized, "down", "cancel")
			})
		})
	}
}

func TestLongPressMovingAfterAcceptIsOk(t *testing.T) {
	for _, group := range []string{"Long press", "long press drag"} {
		t.Run(group, func(t *testing.T) {
			tapTestGesture(t, func(tester *flutter_test.GestureTester) {
				var recognized strs
				var dragUpdate ui.Offset
				gesture := NewLongPressGestureRecognizer()
				lpHandlers(gesture, &recognized, &dragUpdate)
				gesture.AddPointer(lpDown)
				tester.CloseArena(5)
				expectLog(t, recognized)
				tester.Route(lpDown)
				expectLog(t, recognized, "down")
				tester.Async.Elapse(tapMs(1000))
				expectLog(t, recognized, "down", "start")
				tester.Route(lpMove)
				expectLog(t, recognized, "down", "start", "move")
				expectDeep(t, dragUpdate, ui.NewOffset(100, 200))
				tester.Route(lpUp)
				expectLog(t, recognized, "down", "start", "move", "end")
				tester.Async.Elapse(tapMs(300))
				expectLog(t, recognized, "down", "start", "move", "end")
				gesture.Dispose()
				expectLog(t, recognized, "down", "start", "move", "end")
			})
		})
	}
}

func TestLongPressShouldRecognizeBothTapDownAndLongPress(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newLP(&recognized)
		tap := NewTapGestureRecognizer()
		tap.OnTapDown = func(*TapDownDetails) { recognized.add("tap_down") }

		tap.AddPointer(lpDown)
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		expectLog(t, recognized)
		tester.Route(lpDown)
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(300))
		expectLog(t, recognized, "down", "tap_down")
		tester.Async.Elapse(tapMs(700))
		expectLog(t, recognized, "down", "tap_down", "start")
		tap.Dispose()
		gesture.Dispose()
		expectLog(t, recognized, "down", "tap_down", "start")
	})
}

func TestLongPressShouldRecognizeLongPressUp(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newLP(&recognized)
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown) // kLongPressTimeout = 500;
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(300))
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(700))
		expectLog(t, recognized, "down", "start")
		tester.Route(lpUp)
		expectLog(t, recognized, "down", "start", "end")
		gesture.Dispose()
		expectLog(t, recognized, "down", "start", "end")
	})
}

func TestLongPressShouldNotRecognizeWithMoreThanOneButton(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newLP(&recognized)
		gesture.AddPointer(peDown(5, 10, 10, peOpt{Kind: ui.PointerDeviceKindMouse,
			Buttons: KSecondaryMouseButton | KTertiaryButton}))
		tester.CloseArena(5)
		expectLog(t, recognized)
		tester.Route(lpDown)
		expectLog(t, recognized)
		tester.Async.Elapse(tapMs(1000))
		expectLog(t, recognized)
		tester.Route(lpUp)
		expectLog(t, recognized)
		gesture.Dispose()
		expectLog(t, recognized)
	})
}

func TestLongPressShouldCancelWhenButtonsChangeBeforeAcceptance(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newLP(&recognized)
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown)
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(300))
		tester.Route(peMove(5, 10, 10, peOpt{Kind: ui.PointerDeviceKindMouse, Buttons: KTertiaryButton}))
		expectLog(t, recognized, "down", "cancel")
		tester.Async.Elapse(tapMs(700))
		expectLog(t, recognized, "down", "cancel")
		tester.Route(lpUp)
		expectLog(t, recognized, "down", "cancel")
		gesture.Dispose()
		expectLog(t, recognized, "down", "cancel")
	})
}

func TestLongPressNonAllowedPointerDoesNotResetRecognizer(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newLP(&recognized, LongPressGestureRecognizerOptions{
			SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindTouch: {}}})

		// Accept a long-press gesture
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown)
		tester.Async.Elapse(tapMs(500))
		expectLog(t, recognized, "down", "start")

		// Add a non-allowed pointer (doesn't match the kind filter)
		gesture.AddPointer(peDown(101, 10, 10, peOpt{Kind: ui.PointerDeviceKindMouse}))
		expectLog(t, recognized, "down", "start")

		// Moving the primary pointer should result in a normal event
		tester.Route(peMove(5, 15, 15))
		expectLog(t, recognized, "down", "start", "move")
		gesture.Dispose()
	})
}

// Enforce consistent-button restriction.

var lpMoveR = peMove(5, 10, 10, peOpt{Buttons: KSecondaryButton})

func newConsistentLP(t *testing.T, recognized *strs) *LongPressGestureRecognizer {
	g := NewLongPressGestureRecognizer()
	g.OnLongPressDown = func(*LongPressDownDetails) { recognized.add("down") }
	g.OnLongPressCancel = func() { recognized.add("cancel") }
	g.OnLongPressStart = func(*LongPressStartDetails) { recognized.add("start") }
	g.OnLongPressEnd = func(*LongPressEndDetails) { recognized.add("end") }
	t.Cleanup(g.Dispose)
	return g
}

func TestLongPressConsistentButtonCancelBeforeAcceptance(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newConsistentLP(t, &recognized)
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown)
		tester.Async.Elapse(tapMs(300))
		tester.Route(lpMoveR)
		expectLog(t, recognized, "down", "cancel")
		tester.Async.Elapse(tapMs(700))
		tester.Route(lpUp)
		expectLog(t, recognized, "down", "cancel")
	})
}

func TestLongPressConsistentButtonChangeBeforeAcceptanceAllowsNext(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newConsistentLP(t, &recognized)
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown)
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(300))
		tester.Route(lpMoveR)
		expectLog(t, recognized, "down", "cancel")
		tester.Async.Elapse(tapMs(700))
		tester.Route(lpUp)
		recognized = nil

		gesture.AddPointer(lpDown2)
		tester.CloseArena(6)
		tester.Route(lpDown2)
		expectLog(t, recognized, "down")
		tester.Async.Elapse(tapMs(1000))
		expectLog(t, recognized, "down", "start")
		recognized = nil
		tester.Route(lpUp2)
		expectLog(t, recognized, "end")
	})
}

func TestLongPressConsistentButtonNoCancelAfterAcceptance(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newConsistentLP(t, &recognized)
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown)
		tester.Async.Elapse(tapMs(1000))
		expectLog(t, recognized, "down", "start")
		recognized = nil
		tester.Route(lpMoveR)
		expectLog(t, recognized)
		tester.Route(lpUp)
		expectLog(t, recognized, "end")
	})
}

func TestLongPressConsistentButtonChangeAfterAcceptanceAllowsNext(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		gesture := newConsistentLP(t, &recognized)
		gesture.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown)
		tester.Async.Elapse(tapMs(1000))
		tester.Route(lpMoveR)
		tester.Route(lpUp)
		recognized = nil

		gesture.AddPointer(lpDown2)
		tester.CloseArena(6)
		tester.Route(lpDown2)
		tester.Async.Elapse(tapMs(1000))
		expectLog(t, recognized, "down", "start")
		recognized = nil
		tester.Route(lpUp2)
		expectLog(t, recognized, "end")
	})
}

func TestLongPressCanFilterBasedOnDeviceKind(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		mouseLongPress := NewLongPressGestureRecognizer(LongPressGestureRecognizerOptions{
			SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindMouse: {}}})
		mouseLongPressDown := false
		mouseLongPress.OnLongPress = func() { mouseLongPressDown = true }
		mouseDown := peDown(5, 10, 10, peOpt{Kind: ui.PointerDeviceKindMouse})
		touchDown := peDown(5, 10, 10)

		// Touch events shouldn't be recognized.
		mouseLongPress.AddPointer(touchDown)
		tester.CloseArena(5)
		expectDeep(t, mouseLongPressDown, false)
		tester.Route(touchDown)
		expectDeep(t, mouseLongPressDown, false)
		tester.Async.Elapse(tapMs(2000))
		expectDeep(t, mouseLongPressDown, false)

		// Mouse events are still recognized.
		mouseLongPress.AddPointer(mouseDown)
		tester.CloseArena(5)
		expectDeep(t, mouseLongPressDown, false)
		tester.Route(mouseDown)
		expectDeep(t, mouseLongPressDown, false)
		tester.Async.Elapse(tapMs(2000))
		expectDeep(t, mouseLongPressDown, true)
		mouseLongPress.Dispose()
	})
}

type lpCompetition struct {
	tapPrimary, tapSecondary *TapGestureRecognizer
	longPress                *LongPressGestureRecognizer
}

func newLPCompetition(t *testing.T, recognized *strs) lpCompetition {
	c := lpCompetition{NewTapGestureRecognizer(), NewTapGestureRecognizer(), NewLongPressGestureRecognizer()}
	c.tapPrimary.OnTapDown = func(*TapDownDetails) { recognized.add("tapPrimary") }
	c.tapSecondary.OnSecondaryTapDown = func(*TapDownDetails) { recognized.add("tapSecondary") }
	c.longPress.OnLongPressStart = func(*LongPressStartDetails) { recognized.add("longPress") }
	t.Cleanup(func() {
		c.tapPrimary.Dispose()
		c.tapSecondary.Dispose()
		c.longPress.Dispose()
	})
	return c
}

func TestLongPressPrimaryDoesNotCompeteWithSecondaryTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		c := newLPCompetition(t, &recognized)
		c.longPress.AddPointer(lpDown3)
		c.tapSecondary.AddPointer(lpDown3)
		tester.CloseArena(7)
		tester.Route(lpDown3)
		expectLog(t, recognized, "tapSecondary")
	})
}

func TestLongPressPrimaryFormsCompetitionWithPrimaryTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		c := newLPCompetition(t, &recognized)
		c.longPress.AddPointer(lpDown)
		c.tapPrimary.AddPointer(lpDown)
		tester.CloseArena(5)
		tester.Route(lpDown)
		expectLog(t, recognized)
		tester.Route(lpUp)
		expectLog(t, recognized, "tapPrimary")
	})
}

func primaryLP(recognized *strs) *LongPressGestureRecognizer {
	g := NewLongPressGestureRecognizer()
	g.OnLongPressStart = func(*LongPressStartDetails) { recognized.add("primaryStart") }
	g.OnLongPress = func() { recognized.add("primary") }
	g.OnLongPressMoveUpdate = func(*LongPressMoveUpdateDetails) { recognized.add("primaryUpdate") }
	g.OnLongPressEnd = func(*LongPressEndDetails) { recognized.add("primaryEnd") }
	g.OnLongPressUp = func() { recognized.add("primaryUp") }
	return g
}

func TestLongPressSecondaryShouldNotTriggerPrimary(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		longPress := primaryLP(&recognized)
		down2 := peDown(2, 30, 30, peOpt{Buttons: KSecondaryButton})
		move2 := peMove(2, 100, 200, peOpt{Buttons: KSecondaryButton})
		up2 := peUp(2, 100, 201)
		longPress.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		tester.Async.Elapse(tapMs(700))
		tester.Route(move2)
		tester.Route(up2)
		expectLog(t, recognized)
		longPress.Dispose()
	})
}

func TestLongPressTertiaryShouldNotTriggerPrimaryOrSecondary(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		longPress := primaryLP(&recognized)
		longPress.OnSecondaryLongPressStart = func(*LongPressStartDetails) { recognized.add("secondaryStart") }
		longPress.OnSecondaryLongPress = func() { recognized.add("secondary") }
		longPress.OnSecondaryLongPressMoveUpdate = func(*LongPressMoveUpdateDetails) { recognized.add("secondaryUpdate") }
		longPress.OnSecondaryLongPressEnd = func(*LongPressEndDetails) { recognized.add("secondaryEnd") }
		longPress.OnSecondaryLongPressUp = func() { recognized.add("secondaryUp") }
		down2 := peDown(2, 30, 30, peOpt{Buttons: KTertiaryButton})
		move2 := peMove(2, 100, 200, peOpt{Buttons: KTertiaryButton})
		up2 := peUp(2, 100, 201)
		longPress.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		tester.Async.Elapse(tapMs(700))
		tester.Route(move2)
		tester.Route(up2)
		expectLog(t, recognized)
		longPress.Dispose()
	})
}

func startEndLP(recognized *strs) *LongPressGestureRecognizer {
	g := NewLongPressGestureRecognizer()
	g.OnLongPressStart = func(*LongPressStartDetails) { recognized.add("primaryStart") }
	g.OnLongPressEnd = func(*LongPressEndDetails) { recognized.add("primaryEnd") }
	return g
}

func TestLongPressSwitchingButtonsMidStreamSendsEnd(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		longPress := startEndLP(&recognized)
		down4 := peDown(8, 10, 10)
		move4 := peMove(8, 100, 200, peOpt{Buttons: KPrimaryButton | KSecondaryButton})
		up4 := peUp(8, 100, 200, peOpt{Buttons: KSecondaryButton})
		longPress.AddPointer(down4)
		tester.CloseArena(4)
		tester.Route(down4)
		tester.Async.Elapse(tapMs(1000))
		recognized.add("two seconds later...")
		tester.Route(move4)
		tester.Async.Elapse(tapMs(1000))
		recognized.add("two more seconds later...")
		tester.Route(up4)
		tester.Async.Elapse(tapMs(1000))
		expectLog(t, recognized, "primaryStart", "two seconds later...", "two more seconds later...", "primaryEnd")
		longPress.Dispose()
	})
}

func TestLongPressSwitchingButtonsMidStreamSendsEndAlternative(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		longPress := startEndLP(&recognized)
		down5 := peDown(9, 10, 10)
		move5a := peMove(9, 100, 200, peOpt{Buttons: 3}) // add 2
		move5b := peMove(9, 100, 200, peOpt{Buttons: 2}) // remove 1
		up5 := peUp(9, 100, 200)
		longPress.AddPointer(down5)
		tester.CloseArena(4)
		tester.Route(down5)
		tester.Async.Elapse(tapMs(1000))
		recognized.add("two seconds later...")
		tester.Route(move5a)
		tester.Async.Elapse(tapMs(1000))
		recognized.add("two more seconds later...")
		tester.Route(move5b)
		tester.Async.Elapse(tapMs(1000))
		recognized.add("two more seconds later still...")
		tester.Route(up5)
		tester.Async.Elapse(tapMs(1000))
		expectLog(t, recognized, "primaryStart", "two seconds later...", "two more seconds later...",
			"two more seconds later still...", "primaryEnd")
		longPress.Dispose()
	})
}
