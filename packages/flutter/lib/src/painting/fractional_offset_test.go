package painting

// Port of packages/flutter/test/painting/fractional_offset_test.dart
// (Flutter 3.50.1). Every test is ported.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func TestFractionalOffsetControl(t *testing.T) {
	a := NewFractionalOffset(0.5, 0.25)
	b := NewFractionalOffset(1.25, 0.75)
	hasOneLineDescription(t, a.String())
	if a != NewFractionalOffset(0.5, 0.25) {
		t.Error("hashCode/==")
	}
	if a.String() != "FractionalOffset(0.5, 0.3)" {
		t.Error(a.String())
	}
	expectAlign(t, a.Neg(), NewFractionalOffset(-0.5, -0.25))
	expectAlign(t, a.Sub(b), NewFractionalOffset(-0.75, -0.5))
	expectAlign(t, a.Plus(b), NewFractionalOffset(1.75, 1.0))
	expectAlign(t, a.Mul(2.0), FractionalOffsetCenterRight)
	expectAlign(t, a.Div(2.0), NewFractionalOffset(0.25, 0.125))
	expectAlign(t, a.TruncDiv(2.0), FractionalOffsetTopLeft)
	expectAlign(t, a.Mod(5.0), NewFractionalOffset(0.5, 0.25))
	// Not in Flutter's test: the sum and difference with an Alignment are
	// Alignment arithmetic (Dart's `super + other`).
	if got, ok := a.Plus(AlignmentCenter).(Alignment); !ok || got != a.Alignment {
		t.Errorf("FractionalOffset + Alignment = %v", got)
	}
	if got, ok := a.Sub(AlignmentTopLeft).(Alignment); !ok || got != NewAlignment(1.0, 0.5) {
		t.Errorf("FractionalOffset - Alignment = %v", got)
	}
	if _, ok := a.Add(b).(FractionalOffset); !ok {
		t.Error("add of two FractionalOffsets is not one")
	}
}

func TestFractionalOffsetLerp(t *testing.T) {
	a, b := FractionalOffsetTopLeft, FractionalOffsetTopCenter
	expectAlign(t, *FractionalOffsetLerp(&a, &b, 0.25), NewFractionalOffset(0.125, 0.0))
	if FractionalOffsetLerp(nil, nil, 0.25) != nil {
		t.Error("lerp(null, null)")
	}
	expectAlign(t, *FractionalOffsetLerp(nil, &b, 0.25), NewFractionalOffset(0.5, 0.5-0.125))
	expectAlign(t, *FractionalOffsetLerp(&a, nil, 0.25), NewFractionalOffset(0.125, 0.125))
}

func TestFractionalOffsetLerpIdentical(t *testing.T) {
	if FractionalOffsetLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	decoration := NewFractionalOffset(1, 2)
	if FractionalOffsetLerp(&decoration, &decoration, 0.5) != &decoration {
		t.Error("identical")
	}
}

func TestFractionalOffsetFromOffsetAndSize(t *testing.T) {
	a := NewFractionalOffsetFromOffsetAndSize(ui.NewOffset(100.0, 100.0), ui.NewSize(200.0, 400.0))
	expectAlign(t, a, NewFractionalOffset(0.5, 0.25))
}

func TestFractionalOffsetFromOffsetAndRect(t *testing.T) {
	a := NewFractionalOffsetFromOffsetAndRect(ui.NewOffset(150.0, 120.0), ui.NewRectFromLTWH(50.0, 20.0, 200.0, 400.0))
	expectAlign(t, a, NewFractionalOffset(0.5, 0.25))
}
