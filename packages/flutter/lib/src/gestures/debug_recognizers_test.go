package gestures_test

// The rest of packages/flutter/test/gestures/debug_test.dart (Flutter
// 3.50.1) that needs GestureBinding and TapGestureRecognizer:
// debugPrintGestureArenaDiagnostics, debugPrintRecognizerCallbacksTrace,
// both together, and "TapGestureRecognizer _sentTapDown toString". (The
// details part of "Gesture details debugFillProperties" is in
// debug_test.go.)

import (
	"regexp"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

var hashCode = regexp.MustCompile(`#[0-9a-f]{5}`)

// ignoringHashCodes is equalsIgnoringHashCodes' normalization.
func ignoringHashCodes(s string) string { return hashCode.ReplaceAllString(s, "#00000") }

func captureDebugPrint(t *testing.T) *[]string {
	log := &[]string{}
	old := foundation.DebugPrint
	foundation.DebugPrint = func(s string, _ ...foundation.DebugPrintOptions) { *log = append(*log, s) }
	t.Cleanup(func() { foundation.DebugPrint = old })
	return log
}

func newDebugTap() *TapGestureRecognizer {
	tap := NewTapGestureRecognizer()
	tap.OnTapDown = func(*TapDownDetails) {}
	tap.OnTapUp = func(*TapUpDetails) {}
	tap.OnTap = func() {}
	tap.OnTapCancel = func() {}
	return tap
}

func expectLines(t *testing.T, log []string, want ...string) {
	t.Helper()
	if len(log) != len(want) {
		t.Fatalf("log %q, want %q", log, want)
	}
	for i := range want {
		if ignoringHashCodes(log[i]) != want[i] {
			t.Errorf("line %d: %q, want %q", i, log[i], want[i])
		}
	}
}

func runDebugTapSequence(t *testing.T, log *[]string, afterDown, afterClose, afterSweep []string) {
	tap := newDebugTap()
	expectLines(t, *log)
	var event PointerEvent = NewPointerDownEvent(PointerDownEventOptions{Pointer: 1, Position: ui.NewOffset(10, 10)})
	tap.AddPointer(event.(*PointerDownEvent))
	expectLines(t, *log, afterDown...)
	*log = nil
	GestureBindingInstance().GestureArena().Close(1)
	expectLines(t, *log, afterClose...)
	*log = nil
	GestureBindingInstance().PointerRouter().Route(event)
	expectLines(t, *log)
	event = NewPointerUpEvent(PointerUpEventOptions{Pointer: 1, Position: ui.NewOffset(12, 8)})
	GestureBindingInstance().PointerRouter().Route(event)
	expectLines(t, *log)
	GestureBindingInstance().GestureArena().Sweep(1)
	expectLines(t, *log, afterSweep...)
	*log = nil
	tap.Dispose()
	expectLines(t, *log)
}

const (
	arenaOpen   = "Gesture arena 1    ❙ ★ Opening new gesture arena."
	arenaAdd    = "Gesture arena 1    ❙ Adding: TapGestureRecognizer#00000(state: ready, button: 1)"
	arenaClose  = "Gesture arena 1    ❙ Closing with 1 member."
	arenaSweep  = "Gesture arena 1    ❙ Sweeping with 1 member."
	arenaWinner = "Gesture arena 1    ❙ Winner: TapGestureRecognizer#00000(state: ready, finalPosition: Offset(12.0, 8.0), button: 1)"
	traceDown   = "TapGestureRecognizer#00000(state: ready, finalPosition: Offset(12.0, 8.0), button: 1) calling onTapDown callback."
	traceUp     = "TapGestureRecognizer#00000(state: ready, won arena, finalPosition: Offset(12.0, 8.0), button: 1, sent tap down) calling onTapUp callback."
	traceTap    = "TapGestureRecognizer#00000(state: ready, won arena, finalPosition: Offset(12.0, 8.0), button: 1, sent tap down) calling onTap callback."
)

func TestDebugPrintGestureArenaDiagnostics(t *testing.T) {
	flutter_test.New(t)
	DebugPrintGestureArenaDiagnostics = true
	defer func() { DebugPrintGestureArenaDiagnostics = false }()
	log := captureDebugPrint(t)
	runDebugTapSequence(t, log, []string{arenaOpen, arenaAdd}, []string{arenaClose}, []string{arenaSweep, arenaWinner})
}

func TestDebugPrintRecognizerCallbacksTrace(t *testing.T) {
	flutter_test.New(t)
	DebugPrintRecognizerCallbacksTrace = true
	defer func() { DebugPrintRecognizerCallbacksTrace = false }()
	log := captureDebugPrint(t)
	runDebugTapSequence(t, log, nil, nil, []string{traceDown, traceUp, traceTap})
}

func TestDebugPrintGestureArenaDiagnosticsAndRecognizerCallbacksTrace(t *testing.T) {
	flutter_test.New(t)
	DebugPrintGestureArenaDiagnostics = true
	DebugPrintRecognizerCallbacksTrace = true
	defer func() {
		DebugPrintGestureArenaDiagnostics = false
		DebugPrintRecognizerCallbacksTrace = false
	}()
	log := captureDebugPrint(t)
	prefix := strings.Repeat(" ", 19) + "❙ "
	runDebugTapSequence(t, log, []string{arenaOpen, arenaAdd}, []string{arenaClose},
		[]string{arenaSweep, arenaWinner, prefix + traceDown, prefix + traceUp, prefix + traceTap})
}

func TestTapGestureRecognizerSentTapDownToString(t *testing.T) {
	flutter_test.New(t)
	tap := NewTapGestureRecognizer()
	tap.OnTap = func() {}
	if s := ignoringHashCodes(tap.String()); s != "TapGestureRecognizer#00000(state: ready)" {
		t.Error(s)
	}
	tap.AddPointer(NewPointerDownEvent(PointerDownEventOptions{Pointer: 1, Position: ui.NewOffset(10, 10)}))
	tap.DidExceedDeadline()
	if s := ignoringHashCodes(tap.String()); s != "TapGestureRecognizer#00000(state: possible, button: 1, sent tap down)" {
		t.Error(s)
	}
	GestureBindingInstance().GestureArena().Close(1)
	tap.Dispose()
}
