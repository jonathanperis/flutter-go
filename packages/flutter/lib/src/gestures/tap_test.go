package gestures_test

// Port of packages/flutter/test/gestures/tap_test.dart (Flutter 3.50.1):
// all 34 tests (the three competing with HorizontalDragGestureRecognizer
// are at the end of the file).

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// The test file's const events.
var (
	tDown1 = peDown(1, 10, 10)
	tUp1   = peUp(1, 11, 9)
	tDown2 = peDown(2, 30, 30)
	tUp2   = peUp(2, 31, 29)
	tDown3 = peDown(3, 10, 10) // intervening motion, more than kTouchSlop (~21px)
	tMove3 = peMove(3, 25, 25)
	tUp3   = peUp(3, 25, 25)
	tDown4 = peDown(4, 10, 10) // intervening motion, less than kTouchSlop (~17px)
	tMove4 = peMove(4, 22, 22)
	tUp4   = peUp(4, 22, 22)
	tDown5 = peDown(5, 20, 20, peOpt{Buttons: KSecondaryButton})
	tUp5   = peUp(5, 20, 20)
	tDown6 = peDown(6, 20, 20, peOpt{Buttons: KTertiaryButton})
	tUp6   = peUp(6, 20, 20)
)

func TestTapShouldRecognizeTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tapRecognized := false
		tap.OnTap = func() { tapRecognized = true }

		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		expectDeep(t, tapRecognized, false)
		tester.Route(tDown1)
		expectDeep(t, tapRecognized, false)

		tester.Route(tUp1)
		expectDeep(t, tapRecognized, true)
		tapArena().Sweep(1)
		expectDeep(t, tapRecognized, true)

		tap.Dispose()
	})
}

func TestTapShouldRecognizeTapForSupportedDevicesOnly(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer(TapGestureRecognizerOptions{GestureRecognizerOptions: GestureRecognizerOptions{
			SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindMouse: {}, ui.PointerDeviceKindStylus: {}},
		}})
		tapRecognized := false
		tap.OnTap = func() { tapRecognized = true }

		touchDown, touchUp := peDown(1, 10, 10), peUp(1, 11, 9)
		tap.AddPointer(touchDown)
		tester.CloseArena(1)
		expectDeep(t, tapRecognized, false)
		tester.Route(touchDown)
		expectDeep(t, tapRecognized, false)
		tester.Route(touchUp)
		expectDeep(t, tapRecognized, false)
		tapArena().Sweep(1)
		expectDeep(t, tapRecognized, false)

		for _, kind := range []ui.PointerDeviceKind{ui.PointerDeviceKindMouse, ui.PointerDeviceKindStylus} {
			tapRecognized = false
			down, up := peDown(1, 10, 10, peOpt{Kind: kind}), peUp(1, 11, 9, peOpt{Kind: kind})
			tap.AddPointer(down)
			tester.CloseArena(1)
			expectDeep(t, tapRecognized, false)
			tester.Route(down)
			expectDeep(t, tapRecognized, false)
			tester.Route(up)
			expectDeep(t, tapRecognized, true)
			tapArena().Sweep(1)
			expectDeep(t, tapRecognized, true)
		}

		tap.Dispose()
	})
}

func TestTapDetailsContainTheCorrectDeviceKind(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		var lastDownDetails *TapDownDetails
		var lastUpDetails *TapUpDetails
		tap.OnTapDown = func(d *TapDownDetails) { lastDownDetails = d }
		tap.OnTapUp = func(d *TapUpDetails) { lastUpDetails = d }

		mouseDown := peDown(1, 0, 0, peOpt{Kind: ui.PointerDeviceKindMouse})
		mouseUp := peUp(1, 0, 0, peOpt{Kind: ui.PointerDeviceKindMouse})
		tap.AddPointer(mouseDown)
		tester.CloseArena(1)
		tester.Route(mouseDown)
		if lastDownDetails == nil || *lastDownDetails.Kind != ui.PointerDeviceKindMouse {
			t.Fatalf("down details %v", lastDownDetails)
		}
		tester.Route(mouseUp)
		if lastUpDetails == nil || lastUpDetails.Kind != ui.PointerDeviceKindMouse {
			t.Fatalf("up details %v", lastUpDetails)
		}
		tap.Dispose()
	})
}

func TestTapNoDuplicateTapEvents(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tapsRecognized := 0
		tap.OnTap = func() { tapsRecognized++ }

		for i := 1; i <= 2; i++ {
			tap.AddPointer(tDown1)
			tester.CloseArena(1)
			expectDeep(t, tapsRecognized, i-1)
			tester.Route(tDown1)
			expectDeep(t, tapsRecognized, i-1)
			tester.Route(tUp1)
			expectDeep(t, tapsRecognized, i)
			tapArena().Sweep(1)
			expectDeep(t, tapsRecognized, i)
		}
		tap.Dispose()
	})
}

func testOverlapping(t *testing.T, fifo bool) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tapsRecognized := 0
		tap.OnTap = func() { tapsRecognized++ }

		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		expectDeep(t, tapsRecognized, 0)
		tester.Route(tDown1)
		expectDeep(t, tapsRecognized, 0)

		tap.AddPointer(tDown2)
		tester.CloseArena(2)
		expectDeep(t, tapsRecognized, 0)
		tester.Route(tDown1)
		expectDeep(t, tapsRecognized, 0)

		if fifo {
			tester.Route(tUp1)
			expectDeep(t, tapsRecognized, 1)
			tapArena().Sweep(1)
			expectDeep(t, tapsRecognized, 1)
			tester.Route(tUp2)
			expectDeep(t, tapsRecognized, 1)
			tapArena().Sweep(2)
			expectDeep(t, tapsRecognized, 1)
		} else {
			tester.Route(tUp2)
			expectDeep(t, tapsRecognized, 0)
			tapArena().Sweep(2)
			expectDeep(t, tapsRecognized, 0)
			tester.Route(tUp1)
			expectDeep(t, tapsRecognized, 1)
			tapArena().Sweep(1)
			expectDeep(t, tapsRecognized, 1)
		}
		tap.Dispose()
	})
}

func TestTapShouldNotRecognizeTwoOverlappingTapsFIFO(t *testing.T) { testOverlapping(t, true) }
func TestTapShouldNotRecognizeTwoOverlappingTapsFILO(t *testing.T) { testOverlapping(t, false) }

func testDistance(t *testing.T, down *PointerDownEvent, move *PointerMoveEvent, up *PointerUpEvent, canceled bool) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tapRecognized, tapCanceled := false, false
		tap.OnTap = func() { tapRecognized = true }
		tap.OnTapCancel = func() { tapCanceled = true }
		p := down.Pointer()

		tap.AddPointer(down)
		tester.CloseArena(p)
		expectDeep(t, [2]bool{tapRecognized, tapCanceled}, [2]bool{false, false})
		tester.Route(down)
		expectDeep(t, [2]bool{tapRecognized, tapCanceled}, [2]bool{false, false})

		tester.Route(move)
		expectDeep(t, [2]bool{tapRecognized, tapCanceled}, [2]bool{false, canceled})
		tester.Route(up)
		expectDeep(t, [2]bool{tapRecognized, tapCanceled}, [2]bool{!canceled, canceled})
		tapArena().Sweep(p)
		expectDeep(t, [2]bool{tapRecognized, tapCanceled}, [2]bool{!canceled, canceled})
		tap.Dispose()
	})
}

func TestTapDistanceCancelsTap(t *testing.T) { testDistance(t, tDown3, tMove3, tUp3, true) }
func TestTapShortDistanceDoesNotCancelTap(t *testing.T) {
	testDistance(t, tDown4, tMove4, tUp4, false)
}

func TestTapTimeoutDoesNotCancelTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tapRecognized := false
		tap.OnTap = func() { tapRecognized = true }

		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		tester.Route(tDown1)
		expectDeep(t, tapRecognized, false)
		tester.Async.Elapse(tapMs(500))
		expectDeep(t, tapRecognized, false)
		tester.Route(tUp1)
		expectDeep(t, tapRecognized, true)
		tapArena().Sweep(1)
		expectDeep(t, tapRecognized, true)
		tap.Dispose()
	})
}

func testHeldArena(t *testing.T, disposition GestureDisposition) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tapRecognized := false
		tap.OnTap = func() { tapRecognized = true }

		tap.AddPointer(tDown1)
		entry := tapArena().Add(1, nopArenaMember{})
		tapArena().Hold(1)
		tester.CloseArena(1)
		expectDeep(t, tapRecognized, false)
		tester.Route(tDown1)
		expectDeep(t, tapRecognized, false)
		tester.Route(tUp1)
		expectDeep(t, tapRecognized, false)
		tapArena().Sweep(1)
		expectDeep(t, tapRecognized, false)

		entry.Resolve(disposition)
		tester.Async.FlushMicrotasks()
		expectDeep(t, tapRecognized, disposition == GestureDispositionRejected)
		tap.Dispose()
	})
}

func TestTapShouldYieldToOtherArenaMembers(t *testing.T) {
	testHeldArena(t, GestureDispositionAccepted)
}

func TestTapShouldTriggerOnReleaseOfHeldArena(t *testing.T) {
	testHeldArena(t, GestureDispositionRejected)
}

func TestTapShouldLogExceptionsFromCallbacks(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tap.OnTap = func() { panic("test") }

		previous := foundation.FlutterErrorOnError
		gotError := false
		foundation.FlutterErrorOnError = func(*foundation.FlutterErrorDetails) { gotError = true }
		defer func() { foundation.FlutterErrorOnError = previous }()

		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		tester.Route(tDown1)
		expectDeep(t, gotError, false)
		tester.Route(tUp1)
		expectDeep(t, gotError, true)
		tap.Dispose()
	})
}

func TestTapOnTapCancelShouldShowReasonInTheProperFormat(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tap := NewTapGestureRecognizer()
		tap.OnTapCancel = func() { panic("test") }

		previous := foundation.FlutterErrorOnError
		gotError := false
		foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
			if !strings.Contains(details.String(), `"spontaneous onTapCancel"`) {
				t.Errorf("details: %s", details.String())
			}
			gotError = true
		}
		defer func() { foundation.FlutterErrorOnError = previous }()

		tap.AddPointer(peDown(1, 0, 0))
		tester.CloseArena(1)
		tester.Async.Elapse(tapMs(500))
		tester.Route(peCancel(1, 0, 0))
		expectDeep(t, gotError, true)
		tap.Dispose()
	})
}

func TestTapNoDuplicateTapEventsTwoRecognizers(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		tapA := NewTapGestureRecognizer()
		tapB := NewTapGestureRecognizer()
		var log strs
		for _, r := range []struct {
			n   string
			rec *TapGestureRecognizer
		}{{"tapA", tapA}, {"tapB", tapB}} {
			n := r.n
			r.rec.OnTapDown = func(*TapDownDetails) { log.add(n + " onTapDown") }
			r.rec.OnTapUp = func(*TapUpDetails) { log.add(n + " onTapUp") }
			r.rec.OnTap = func() { log.add(n + " onTap") }
			r.rec.OnTapCancel = func() { log.add(n + " onTapCancel") }
		}

		log.add("start")
		tapA.AddPointer(tDown1)
		log.add("added 1 to A")
		tapB.AddPointer(tDown1)
		log.add("added 1 to B")
		tester.CloseArena(1)
		log.add("closed 1")
		tester.Route(tDown1)
		log.add("routed 1 down")
		tester.Route(tUp1)
		log.add("routed 1 up")
		tapArena().Sweep(1)
		log.add("swept 1")
		tapA.AddPointer(tDown2)
		log.add("down 2 to A")
		tapB.AddPointer(tDown2)
		log.add("down 2 to B")
		tester.CloseArena(2)
		log.add("closed 2")
		tester.Route(tDown2)
		log.add("routed 2 down")
		tester.Route(tUp2)
		log.add("routed 2 up")
		tapArena().Sweep(2)
		log.add("swept 2")
		tapA.Dispose()
		log.add("disposed A")
		tapB.Dispose()
		log.add("disposed B")

		expectLog(t, log, "start", "added 1 to A", "added 1 to B", "closed 1", "routed 1 down", "routed 1 up",
			"tapA onTapDown", "tapA onTapUp", "tapA onTap", "swept 1", "down 2 to A", "down 2 to B", "closed 2",
			"routed 2 down", "routed 2 up", "tapA onTapDown", "tapA onTapUp", "tapA onTap", "swept 2",
			"disposed A", "disposed B")
	})
}

func recordingTap(log *strs) *TapGestureRecognizer {
	tap := NewTapGestureRecognizer()
	tap.OnTapDown = func(*TapDownDetails) { log.add("down") }
	tap.OnTapUp = func(*TapUpDetails) { log.add("up") }
	tap.OnTap = func() { log.add("tap") }
	tap.OnTapCancel = func() { log.add("cancel") }
	return tap
}

func TestTapPointerCancelEventCancelsTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		tap := recordingTap(&recognized)
		tap.AddPointer(peDown(5, 10, 10))
		tester.CloseArena(5)
		tester.Async.Elapse(tapMs(5000))
		expectLog(t, recognized, "down")
		tester.Route(peCancel(5, 10, 10))
		expectLog(t, recognized, "down", "cancel")
		tap.Dispose()
	})
}

func TestTapNonPrimaryPointersDoesNotTriggerTimeout(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		tap := recordingTap(&recognized)

		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		tap.AddPointer(tDown2)
		tester.CloseArena(2)
		expectLog(t, recognized)

		tester.Route(tUp1)
		tapArena().Sweep(1)
		expectLog(t, recognized, "down", "up", "tap")
		recognized = nil

		// If regression happens, the following step will throw error
		tester.Async.Elapse(tapMs(200))
		expectLog(t, recognized)

		tester.Route(tUp2)
		tapArena().Sweep(2)
		expectLog(t, recognized)
		tap.Dispose()
	})
}

// Enforce consistent-button restriction.

var (
	tMove1lr = peMove(1, 10, 10, peOpt{Buttons: KPrimaryMouseButton | KSecondaryMouseButton})
	tMove1r  = peMove(1, 10, 10, peOpt{Buttons: KSecondaryMouseButton})
)

func consistentButtonTap(log *strs) *TapGestureRecognizer {
	tap := NewTapGestureRecognizer()
	tap.OnTapDown = func(*TapDownDetails) { log.add("down") }
	tap.OnTapUp = func(*TapUpDetails) { log.add("up") }
	tap.OnTapCancel = func() { log.add("cancel") }
	return tap
}

func TestTapChangingButtonsBeforeTapDownCancelsWithoutCancel(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		tap := consistentButtonTap(&recognized)
		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		expectLog(t, recognized)
		tester.Route(tMove1lr)
		expectLog(t, recognized)
		tester.Route(tMove1r)
		expectLog(t, recognized)
		tester.Route(tUp1)
		expectLog(t, recognized)
		tap.Dispose()
	})
}

func TestTapChangingButtonsBeforeTapDownDoesNotPreventNextTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		tap := consistentButtonTap(&recognized)
		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		tester.Route(tMove1lr)
		tester.Route(tMove1r)
		tester.Route(tUp1)
		expectLog(t, recognized)

		tap.AddPointer(tDown2)
		tester.CloseArena(2)
		tester.Async.Elapse(tapMs(1000))
		tester.Route(tUp2)
		expectLog(t, recognized, "down", "up")
		tap.Dispose()
	})
}

func TestTapChangingButtonsAfterTapDownCancelsAndSendsCancel(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		tap := consistentButtonTap(&recognized)
		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		expectLog(t, recognized)
		tester.Async.Elapse(tapMs(1000))
		expectLog(t, recognized, "down")
		tester.Route(tMove1lr)
		expectLog(t, recognized, "down", "cancel")
		tester.Route(tMove1r)
		expectLog(t, recognized, "down", "cancel")
		tester.Route(tUp1)
		expectLog(t, recognized, "down", "cancel")
		tap.Dispose()
	})
}

func TestTapChangingButtonsAfterTapDownDoesNotPreventNextTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		tap := consistentButtonTap(&recognized)
		tap.AddPointer(tDown1)
		tester.CloseArena(1)
		tester.Async.Elapse(tapMs(1000))
		tester.Route(tMove1lr)
		tester.Route(tMove1r)
		tester.Route(tUp1)
		tapArena().Sweep(1)
		expectLog(t, recognized, "down", "cancel")

		tap.AddPointer(tDown2)
		tester.CloseArena(2)
		tester.Async.Elapse(tapMs(1000))
		tester.Route(tUp2)
		tapArena().Sweep(2)
		expectLog(t, recognized, "down", "cancel", "down", "up")
		tap.Dispose()
	})
}

// Recognizers listening on different buttons do not form competition.

type buttonTaps struct {
	primary, primary2, secondary, tertiary *TapGestureRecognizer
}

func newButtonTaps(log *strs) buttonTaps {
	primary := func(n string) *TapGestureRecognizer {
		r := NewTapGestureRecognizer()
		r.OnTapDown = func(*TapDownDetails) { log.add(n + "Down") }
		r.OnTapUp = func(*TapUpDetails) { log.add(n + "Up") }
		r.OnTapCancel = func() { log.add(n + "Cancel") }
		return r
	}
	secondary := NewTapGestureRecognizer()
	secondary.OnSecondaryTapDown = func(*TapDownDetails) { log.add("secondaryDown") }
	secondary.OnSecondaryTapUp = func(*TapUpDetails) { log.add("secondaryUp") }
	secondary.OnSecondaryTapCancel = func() { log.add("secondaryCancel") }
	tertiary := NewTapGestureRecognizer()
	tertiary.OnTertiaryTapDown = func(*TapDownDetails) { log.add("tertiaryDown") }
	tertiary.OnTertiaryTapUp = func(*TapUpDetails) { log.add("tertiaryUp") }
	tertiary.OnTertiaryTapCancel = func() { log.add("tertiaryCancel") }
	return buttonTaps{primary("primary"), primary("primary2"), secondary, tertiary}
}

func (b buttonTaps) dispose() {
	b.primary.Dispose()
	b.primary2.Dispose()
	b.secondary.Dispose()
	b.tertiary.Dispose()
}

func TestTapPrimaryDoesNotCompeteWithSecondaryOrTertiary(t *testing.T) {
	for _, other := range []string{"secondary", "tertiary"} {
		t.Run(other, func(t *testing.T) {
			tapTestGesture(t, func(tester *flutter_test.GestureTester) {
				var recognized strs
				b := newButtonTaps(&recognized)
				defer b.dispose()
				b.primary.AddPointer(tDown1)
				if other == "secondary" {
					b.secondary.AddPointer(tDown1)
				} else {
					b.tertiary.AddPointer(tDown1)
				}
				tester.CloseArena(1)
				tester.Route(tDown1)
				expectLog(t, recognized, "primaryDown")
				recognized = nil
				tester.Route(tUp1)
				expectLog(t, recognized, "primaryUp")
			})
		})
	}
}

func TestTapPrimaryFormsCompetitionWithAnotherPrimary(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		b := newButtonTaps(&recognized)
		defer b.dispose()
		b.primary.AddPointer(tDown1)
		b.primary2.AddPointer(tDown1)
		tester.CloseArena(1)
		tester.Route(tDown1)
		expectLog(t, recognized)
		tester.Async.Elapse(tapMs(500))
		expectLog(t, recognized, "primaryDown", "primary2Down")
	})
}

// Gestures of different buttons trigger correct callbacks.

func allButtonsTap(log *strs) *TapGestureRecognizer {
	tap := NewTapGestureRecognizer()
	tap.OnTapDown = func(*TapDownDetails) { log.add("primaryDown") }
	tap.OnTap = func() { log.add("primary") }
	tap.OnTapUp = func(*TapUpDetails) { log.add("primaryUp") }
	tap.OnTapCancel = func() { log.add("primaryCancel") }
	tap.OnSecondaryTapDown = func(*TapDownDetails) { log.add("secondaryDown") }
	tap.OnSecondaryTapUp = func(*TapUpDetails) { log.add("secondaryUp") }
	tap.OnSecondaryTapCancel = func() { log.add("secondaryCancel") }
	tap.OnTertiaryTapDown = func(*TapDownDetails) { log.add("tertiaryDown") }
	tap.OnTertiaryTapUp = func(*TapUpDetails) { log.add("tertiaryUp") }
	tap.OnTertiaryTapCancel = func() { log.add("tertiaryCancel") }
	return tap
}

func TestTapDifferentButtonsTriggerCorrectCallbacks(t *testing.T) {
	cases := []struct {
		name  string
		down  *PointerDownEvent
		end   PointerEvent
		want1 string
		want2 []string
	}{
		{"primary tap", tDown1, tUp1, "primaryDown", []string{"primaryUp", "primary"}},
		{"primary cancel", tDown1, peCancel(1, 0, 0), "primaryDown", []string{"primaryCancel"}},
		{"secondary tap", tDown5, tUp5, "secondaryDown", []string{"secondaryUp"}},
		{"tertiary tap", tDown6, tUp6, "tertiaryDown", []string{"tertiaryUp"}},
		{"secondary cancel", tDown5, peCancel(5, 0, 0), "secondaryDown", []string{"secondaryCancel"}},
		{"tertiary cancel", tDown6, peCancel(6, 0, 0), "tertiaryDown", []string{"tertiaryCancel"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tapTestGesture(t, func(tester *flutter_test.GestureTester) {
				var recognized strs
				tap := allButtonsTap(&recognized)
				defer tap.Dispose()
				tap.AddPointer(c.down)
				tester.CloseArena(c.down.Pointer())
				expectLog(t, recognized)
				tester.Async.Elapse(tapMs(500))
				expectLog(t, recognized, c.want1)
				recognized = nil
				tester.Route(c.end)
				tapArena().Sweep(c.down.Pointer())
				expectLog(t, recognized, c.want2...)
			})
		})
	}
}

func TestTapOnTapMoveWorks(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var tapMoveDetails *TapMoveDetails
		tap := NewTapGestureRecognizer(TapGestureRecognizerOptions{NoPostAcceptSlopTolerance: true})
		tap.OnTapMove = func(d *TapMoveDetails) { tapMoveDetails = d }
		defer tap.Dispose()

		pointer1 := flutter_test.NewTestPointer()
		down := pointer1.Down(ui.Offset{})
		tap.AddPointer(down)
		tester.CloseArena(1)
		tester.Route(down)
		tester.Route(pointer1.Move(ui.NewOffset(50, 0)))
		if tapMoveDetails == nil {
			t.Fatal("no move details")
		}
		expectDeep(t, tapMoveDetails.GlobalPosition(), ui.NewOffset(50, 0))
		expectDeep(t, tapMoveDetails.Delta, ui.NewOffset(50, 0))
		tapMoveDetails = nil

		tester.Route(pointer1.Move(ui.NewOffset(60, 10)))
		if tapMoveDetails == nil {
			t.Fatal("no move details")
		}
		expectDeep(t, tapMoveDetails.GlobalPosition(), ui.NewOffset(60, 10))
		expectDeep(t, tapMoveDetails.Delta, ui.NewOffset(10, 10))
	})
}

func tapLogged(t *testing.T, recognized *strs) *TapGestureRecognizer {
	tap := NewTapGestureRecognizer()
	t.Cleanup(tap.Dispose)
	tap.OnTapDown = func(*TapDownDetails) { recognized.add("down") }
	tap.OnTapUp = func(*TapUpDetails) { recognized.add("up") }
	tap.OnTap = func() { recognized.add("tap") }
	tap.OnTapCancel = func() { recognized.add("cancel") }
	return tap
}

func TestTapPointerCancelAfterExceedingDeadlineCancelsTap(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		down := peDown(5, 10, 10)
		cancel := peCancel(5, 10, 10)
		var recognized strs
		tap := tapLogged(t, &recognized)
		drag := NewHorizontalDragGestureRecognizer()
		drag.OnStart = func(*DragStartDetails) {} // Need a callback to compete
		t.Cleanup(drag.Dispose)
		tap.AddPointer(down)
		drag.AddPointer(down)
		tester.CloseArena(5)
		tester.Route(down)
		expectLog(t, recognized)
		tester.Async.Elapse(ms(1000))
		expectLog(t, recognized, "down")
		tester.Route(cancel)
		expectLog(t, recognized, "down", "cancel")
	})
}

func TestTapLosingRecognizerDoesNotSendOnTapCancel(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var recognized strs
		tap := tapLogged(t, &recognized)
		drag := NewHorizontalDragGestureRecognizer()
		t.Cleanup(drag.Dispose)
		tap.AddPointer(tDown3)
		drag.AddPointer(tDown3)
		tester.CloseArena(3)
		tester.Route(tMove3)
		tapArena().Sweep(3)
		expectLog(t, recognized)
	})
}

func TestTapSecondTapAfterRejectionIsIgnored(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		didTap := false
		tap := NewTapGestureRecognizer()
		tap.OnTap = func() { didTap = true }
		t.Cleanup(tap.Dispose)
		drag := NewHorizontalDragGestureRecognizer()
		drag.OnStart = func(*DragStartDetails) {}
		t.Cleanup(drag.Dispose)
		pointer1 := testPointerN(1)
		down := pointer1.Down(ui.Offset{})
		drag.AddPointer(down)
		tap.AddPointer(down)
		tester.CloseArena(1)
		tester.Route(down)
		tester.Route(pointer1.Move(ui.NewOffset(50, 0)))
		pointer2 := testPointerN(2)
		down2 := pointer2.Down(ui.NewOffset(10, 20))
		drag.AddPointer(down2)
		tap.AddPointer(down2)
		tester.CloseArena(2)
		tester.Route(down2)
		if didTap {
			t.Fatal("tapped")
		}
	})
}
