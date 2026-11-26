// Port of packages/flutter/test/gestures/debug_test.dart (Flutter 3.50.1):
// 1 of 5 tests, partially.
//
// Skipped (needs GestureBinding and the recognizers, which the parent
// ports): debugPrintGestureArenaDiagnostics, debugPrintRecognizerCallbacks-
// Trace, both together, and "TapGestureRecognizer _sentTapDown toString".
// "Gesture details debugFillProperties" is ported for the drag details
// (drag_details.dart); the force press, long press, serial tap, scale, tap
// and tap-and-drag details come with their recognizers.
package gestures_test

import (
	"slices"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
)

func TestGestureDetailsDebugFillProperties(t *testing.T) {
	pairs := []struct {
		detail   foundation.Diagnosticable
		expected []string
	}{
		{NewDragDownDetails(), []string{"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)"}},
		{NewDragStartDetails(), []string{"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)",
			"sourceTimeStamp: null", "kind: null"}},
		{NewDragUpdateDetails(DragUpdateDetailsOptions{GlobalPosition: ui.OffsetZero}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "sourceTimeStamp: null",
			"delta: Offset(0.0, 0.0)", "primaryDelta: null"}},
		{NewDragEndDetails(), []string{"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)",
			"velocity: Velocity(0.0, 0.0)", "primaryVelocity: null"}},
	}
	for _, p := range pairs {
		builder := foundation.NewDiagnosticPropertiesBuilder()
		p.detail.DebugFillProperties(builder)
		var description []string
		for _, node := range builder.Properties {
			if !node.IsFiltered(foundation.DiagnosticLevelInfo) {
				description = append(description, node.String())
			}
		}
		if !slices.Equal(description, p.expected) {
			t.Errorf("%T: %q", p.detail, description)
		}
	}
}

// Not in Flutter's tests: DebugAssertAllGesturesVarsUnset.
func TestDebugAssertAllGesturesVarsUnset(t *testing.T) {
	if !DebugAssertAllGesturesVarsUnset("unset") {
		t.Fatal("false")
	}
	DebugPrintGestureArenaDiagnostics = true
	defer func() { DebugPrintGestureArenaDiagnostics = false }()
	defer func() {
		if recover() == nil {
			t.Fatal("no panic with a variable set")
		}
	}()
	DebugAssertAllGesturesVarsUnset("set")
}
