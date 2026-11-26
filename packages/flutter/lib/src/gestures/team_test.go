package gestures_test

// Port of packages/flutter/test/gestures/team_test.dart (Flutter 3.50.1):
// both tests.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// PassiveGestureRecognizer is the test file's PassiveGestureRecognizer.
type PassiveGestureRecognizer struct {
	OneSequenceGestureRecognizerBase
	onGestureAccepted func()
}

func newPassiveGestureRecognizer() *PassiveGestureRecognizer {
	r := &PassiveGestureRecognizer{}
	r.Construct(r, GestureRecognizerOptions{})
	return r
}

func (r *PassiveGestureRecognizer) AddPointer(event *PointerDownEvent) {
	r.StartTrackingPointer(event.Pointer())
}

func (r *PassiveGestureRecognizer) DebugDescription() string { return "passive" }

func (r *PassiveGestureRecognizer) DidStopTrackingLastPointer(pointer int) {
	r.Resolve(GestureDispositionRejected)
}

func (r *PassiveGestureRecognizer) HandleEvent(event PointerEvent) {
	switch event.(type) {
	case *PointerUpEvent, *PointerCancelEvent:
		r.StopTrackingPointer(event.Pointer())
	}
}

func (r *PassiveGestureRecognizer) AcceptGesture(pointer int) {
	if r.onGestureAccepted != nil {
		r.onGestureAccepted()
	}
}

func (r *PassiveGestureRecognizer) RejectGesture(pointer int) {}

// teamRun is the tests' local test(delta): a pointer down at (10, 10),
// moved by delta and released, added to recognizers.
func teamRun(t *testing.T, tester *flutter_test.GestureTester, log *strs, delta ui.Offset, recognizers ...GestureRecognizer) {
	t.Helper()
	origin := ui.NewOffset(10, 10)
	pointer := testPointerN(5)
	down := pointer.Down(origin)
	for _, r := range recognizers {
		r.AddPointer(down)
	}
	expectLog(t, *log)
	tester.CloseArena(5)
	expectLog(t, *log)
	tester.Route(down)
	expectLog(t, *log)
	tester.Route(pointer.Move(origin.Add(delta)))
	tester.Route(pointer.Up())
}

func TestGestureArenaTeamRejection(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		team := NewGestureArenaTeam()
		horizontalDrag := NewHorizontalDragGestureRecognizer()
		horizontalDrag.SetTeam(team)
		verticalDrag := NewVerticalDragGestureRecognizer()
		verticalDrag.SetTeam(team)
		tap := NewTapGestureRecognizer()
		if horizontalDrag.Team() != team || verticalDrag.Team() != team || tap.Team() != nil {
			t.Fatal("teams")
		}
		var log strs
		horizontalDrag.OnStart = func(*DragStartDetails) { log.add("horizontal-drag-start") }
		verticalDrag.OnStart = func(*DragStartDetails) { log.add("vertical-drag-start") }
		tap.OnTap = func() { log.add("tap") }

		teamRun(t, tester, &log, ui.Offset{}, horizontalDrag, verticalDrag, tap)
		expectLog(t, log, "tap")
		log = nil
		teamRun(t, tester, &log, ui.NewOffset(0, 30), horizontalDrag, verticalDrag, tap)
		expectLog(t, log, "vertical-drag-start")
		log = nil

		horizontalDrag.Dispose()
		verticalDrag.Dispose()
		tap.Dispose()
	})
}

func TestGestureArenaTeamCaptain(t *testing.T) {
	testGesture(t, func(tester *flutter_test.GestureTester) {
		team := NewGestureArenaTeam()
		captain := newPassiveGestureRecognizer()
		captain.SetTeam(team)
		horizontalDrag := NewHorizontalDragGestureRecognizer()
		horizontalDrag.SetTeam(team)
		verticalDrag := NewVerticalDragGestureRecognizer()
		verticalDrag.SetTeam(team)
		tap := NewTapGestureRecognizer()
		team.Captain = captain

		var log strs
		captain.onGestureAccepted = func() { log.add("captain accepted gesture") }
		horizontalDrag.OnStart = func(*DragStartDetails) { log.add("horizontal-drag-start") }
		verticalDrag.OnStart = func(*DragStartDetails) { log.add("vertical-drag-start") }
		tap.OnTap = func() { log.add("tap") }

		teamRun(t, tester, &log, ui.Offset{}, captain, horizontalDrag, verticalDrag, tap)
		expectLog(t, log, "tap")
		log = nil
		teamRun(t, tester, &log, ui.NewOffset(0, 30), captain, horizontalDrag, verticalDrag, tap)
		expectLog(t, log, "captain accepted gesture")
		log = nil

		horizontalDrag.Dispose()
		verticalDrag.Dispose()
		tap.Dispose()
		captain.Dispose()
	})
}
