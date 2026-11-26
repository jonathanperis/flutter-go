// Port of packages/flutter/test/gestures/debug_test.dart (Flutter 3.50.1):
// "Gesture details debugFillProperties" (all 22 details). The other four
// tests are in debug_recognizers_test.go.
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
		{NewForcePressDetails(ForcePressDetailsOptions{GlobalPosition: ui.OffsetZero, Pressure: 1.0}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "pressure: 1.0"}},
		{NewLongPressDownDetails(), []string{"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)",
			"kind: null"}},
		{NewLongPressStartDetails(), []string{"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)"}},
		{NewLongPressMoveUpdateDetails(), []string{"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)",
			"offsetFromOrigin: Offset(0.0, 0.0)", "localOffsetFromOrigin: Offset(0.0, 0.0)"}},
		{NewLongPressEndDetails(), []string{"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)",
			"velocity: Velocity(0.0, 0.0)"}},
		{NewSerialTapDownDetails(SerialTapDownDetailsOptions{Kind: ui.PointerDeviceKindUnknown}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "kind: unknown", "buttons: 0", "count: 1"}},
		{NewSerialTapCancelDetails(), []string{"count: 1"}},
		{NewSerialTapUpDetails(SerialTapUpDetailsOptions{}), []string{"globalPosition: Offset(0.0, 0.0)",
			"localPosition: Offset(0.0, 0.0)", "kind: null", "count: 1"}},
		{NewScaleStartDetails(), []string{"focalPoint: Offset(0.0, 0.0)", "localFocalPoint: Offset(0.0, 0.0)",
			"pointerCount: 0", "sourceTimeStamp: null"}},
		{NewScaleUpdateDetails(), []string{"focalPointDelta: Offset(0.0, 0.0)", "focalPoint: Offset(0.0, 0.0)",
			"localFocalPoint: Offset(0.0, 0.0)", "scale: 1.0", "horizontalScale: 1.0", "verticalScale: 1.0",
			"rotation: 0.0", "pointerCount: 0", "sourceTimeStamp: null"}},
		{NewScaleEndDetails(), []string{"velocity: Velocity(0.0, 0.0)", "scaleVelocity: 0.0", "pointerCount: 0"}},
		{NewTapDownDetails(TapDownDetailsOptions{Kind: kindp(ui.PointerDeviceKindUnknown)}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "kind: unknown"}},
		{NewTapUpDetails(TapUpDetailsOptions{Kind: ui.PointerDeviceKindUnknown}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "kind: unknown"}},
		{NewTapDragDownDetails(TapDragDownDetailsOptions{ConsecutiveTapCount: 1}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "kind: null", "consecutiveTapCount: 1"}},
		{NewTapDragUpDetails(TapDragUpDetailsOptions{Kind: ui.PointerDeviceKindUnknown, ConsecutiveTapCount: 1}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "kind: unknown", "consecutiveTapCount: 1"}},
		{NewTapDragStartDetails(TapDragStartDetailsOptions{ConsecutiveTapCount: 1}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "sourceTimeStamp: null", "kind: null",
			"consecutiveTapCount: 1"}},
		{NewTapDragUpdateDetails(TapDragUpdateDetailsOptions{ConsecutiveTapCount: 1}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "sourceTimeStamp: null",
			"delta: Offset(0.0, 0.0)", "primaryDelta: null", "kind: null", "offsetFromOrigin: Offset(0.0, 0.0)",
			"localOffsetFromOrigin: Offset(0.0, 0.0)", "consecutiveTapCount: 1"}},
		{NewTapDragEndDetails(TapDragEndDetailsOptions{ConsecutiveTapCount: 1}), []string{
			"globalPosition: Offset(0.0, 0.0)", "localPosition: Offset(0.0, 0.0)", "velocity: Velocity(0.0, 0.0)",
			"primaryVelocity: null", "consecutiveTapCount: 1"}},
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

func kindp(k ui.PointerDeviceKind) *ui.PointerDeviceKind { return &k }
