package gestures_test

// Port of packages/flutter/test/gestures/multitap_test.dart (Flutter
// 3.50.1): both tests.

import (
	"fmt"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func testPointerN(pointer int, kind ...ui.PointerDeviceKind) *flutter_test.TestPointer {
	o := flutter_test.TestPointerOptions{Pointer: &pointer}
	if len(kind) > 0 {
		o.Kind = kind[0]
	}
	return flutter_test.NewTestPointer(o)
}

func loggingMultiTap(log *strs, opts MultiTapGestureRecognizerOptions) *MultiTapGestureRecognizer {
	tap := NewMultiTapGestureRecognizer(opts)
	tap.OnTapDown = func(p int, _ *TapDownDetails) { log.add(fmt.Sprintf("tap-down %d", p)) }
	tap.OnTapUp = func(p int, _ *TapUpDetails) { log.add(fmt.Sprintf("tap-up %d", p)) }
	tap.OnTap = func(p int) { log.add(fmt.Sprintf("tap %d", p)) }
	tap.OnLongTapDown = func(p int, _ *TapDownDetails) { log.add(fmt.Sprintf("long-tap-down %d", p)) }
	tap.OnTapCancel = func(p int) { log.add(fmt.Sprintf("tap-cancel %d", p)) }
	return tap
}

func TestMultiTapShouldRecognizePan(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var log strs
		tap := loggingMultiTap(&log, MultiTapGestureRecognizerOptions{LongTapDelay: KLongPressTimeout})

		pointer5 := testPointerN(5)
		down5 := pointer5.Down(ui.NewOffset(10, 10))
		tap.AddPointer(down5)
		tester.CloseArena(5)
		expectLog(t, log, "tap-down 5")
		log = nil
		tester.Route(down5)
		expectLog(t, log)

		pointer6 := testPointerN(6)
		down6 := pointer6.Down(ui.NewOffset(15, 15))
		tap.AddPointer(down6)
		tester.CloseArena(6)
		expectLog(t, log, "tap-down 6")
		log = nil
		tester.Route(down6)
		expectLog(t, log)

		tester.Route(pointer5.Move(ui.NewOffset(11, 12)))
		expectLog(t, log)
		tester.Route(pointer6.Move(ui.NewOffset(14, 13)))
		expectLog(t, log)

		tester.Route(pointer5.Up())
		expectLog(t, log, "tap-up 5", "tap 5")
		log = nil

		tester.Async.Elapse(KLongPressTimeout + KPressTimeout)
		expectLog(t, log, "long-tap-down 6")
		log = nil

		tester.Route(pointer6.Move(ui.NewOffset(40, 30))) // move more than kTouchSlop from 15.0,15.0
		expectLog(t, log, "tap-cancel 6")
		log = nil

		tester.Route(pointer6.Up())
		expectLog(t, log)
		tap.Dispose()
	})
}

func TestMultiTapCanFilterBasedOnDeviceKind(t *testing.T) {
	tapTestGesture(t, func(tester *flutter_test.GestureTester) {
		var log strs
		tap := loggingMultiTap(&log, MultiTapGestureRecognizerOptions{LongTapDelay: KLongPressTimeout,
			GestureRecognizerOptions: GestureRecognizerOptions{
				SupportedDevices: map[ui.PointerDeviceKind]struct{}{ui.PointerDeviceKindTouch: {}}}})

		touchPointer5 := testPointerN(5)
		down5 := touchPointer5.Down(ui.NewOffset(10, 10))
		tap.AddPointer(down5)
		tester.CloseArena(5)
		expectLog(t, log, "tap-down 5")
		log = nil
		tester.Route(down5)
		expectLog(t, log)

		mousePointer6 := testPointerN(6, ui.PointerDeviceKindMouse)
		down6 := mousePointer6.Down(ui.NewOffset(20, 20))
		tap.AddPointer(down6)
		tester.CloseArena(6)
		// Mouse down should be ignored by the recognizer.
		expectLog(t, log)

		touchPointer7 := testPointerN(7)
		down7 := touchPointer7.Down(ui.NewOffset(15, 15))
		tap.AddPointer(down7)
		tester.CloseArena(7)
		expectLog(t, log, "tap-down 7")
		log = nil
		tester.Route(down7)
		expectLog(t, log)

		tester.Route(touchPointer5.Move(ui.NewOffset(11, 12)))
		expectLog(t, log)

		// Move within the kTouchSlop range, then beyond it: neither triggers
		// any event because they originate from a mouse.
		tester.Route(mousePointer6.Move(ui.NewOffset(21, 18)))
		tester.Route(mousePointer6.Move(ui.NewOffset(50, 40)))
		expectLog(t, log)

		tester.Route(touchPointer7.Move(ui.NewOffset(14, 13)))
		expectLog(t, log)

		tester.Route(touchPointer5.Up())
		expectLog(t, log, "tap-up 5", "tap 5")
		log = nil

		// Mouse up should be ignored.
		tester.Route(mousePointer6.Up())
		expectLog(t, log)

		tester.Async.Elapse(KLongPressTimeout + KPressTimeout)
		// Only the touch pointer (7) triggers a long-tap, not the mouse pointer (6).
		expectLog(t, log, "long-tap-down 7")
		log = nil

		tester.Route(touchPointer7.Move(ui.NewOffset(40, 30))) // move more than kTouchSlop from 15.0,15.0
		expectLog(t, log, "tap-cancel 7")
		tap.Dispose()
	})
}
