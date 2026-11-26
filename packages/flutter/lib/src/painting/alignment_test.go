package painting

// Port of packages/flutter/test/painting/alignment_test.dart (Flutter
// 3.50.1). Every test is ported; `hasOneLineDescription` checks that the
// String() is one non-empty line. Dart's `expect(x, y)` on alignments is
// Dart's == (Equal here); `identical` is Go value equality of the same
// type (see identicalGeometry).

import (
	"errors"
	"math"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

func expectAlign(t *testing.T, got, want AlignmentGeometry) {
	t.Helper()
	if got == nil || want == nil {
		if got != nil || want != nil {
			t.Errorf("got %v, want %v", got, want)
		}
		return
	}
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func expectNotAlign(t *testing.T, got, want AlignmentGeometry) {
	t.Helper()
	if got.Equal(want) {
		t.Errorf("got %v, want not equal to %v", got, want)
	}
}

func approxExpect(t *testing.T, a, b Alignment) {
	t.Helper()
	if math.Abs(a.X-b.X) > 1e-10 || math.Abs(a.Y-b.Y) > 1e-10 {
		t.Errorf("got %v (%v, %v), want approximately %v (%v, %v)", a, a.X, a.Y, b, b.X, b.Y)
	}
}

func dir(d ui.TextDirection) *ui.TextDirection { return &d }

var (
	ltr = dir(ui.TextDirectionLtr)
	rtl = dir(ui.TextDirectionRtl)
)

func hasOneLineDescription(t *testing.T, s string) {
	t.Helper()
	if s == "" || strings.Contains(s, "\n") || strings.Contains(s, "Instance of ") {
		t.Errorf("%q is not a one-line description", s)
	}
}

func TestAlignmentControl(t *testing.T) {
	alignment := NewAlignment(0.5, 0.25)
	hasOneLineDescription(t, alignment.String())
	if alignment != NewAlignment(0.5, 0.25) {
		t.Error("hashCode/==")
	}
	expectAlign(t, alignment.Div(2.0), NewAlignment(0.25, 0.125))
	expectAlign(t, alignment.TruncDiv(2.0), AlignmentCenter)
	expectAlign(t, alignment.Mod(5.0), NewAlignment(0.5, 0.25))
}

func TestAlignmentLerp(t *testing.T) {
	a, b := AlignmentTopLeft, AlignmentTopCenter
	expectAlign(t, AlignmentLerp(a, b, 0.25), NewAlignment(-0.75, -1.0))
	if AlignmentLerp(nil, nil, 0.25) != nil {
		t.Error("lerp(null, null) is not null")
	}
	expectAlign(t, AlignmentLerp(nil, b, 0.25), NewAlignment(0.0, -0.25))
	expectAlign(t, AlignmentLerp(a, nil, 0.25), NewAlignment(-0.75, -0.75))
}

func TestAlignmentLerpIdentical(t *testing.T) {
	if AlignmentLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	alignment := AlignmentTopLeft
	if AlignmentLerp(alignment, alignment, 0.5) != AlignmentLike(alignment) {
		t.Error("identical")
	}
	// A FractionalOffset lerped with itself stays one (Dart returns a).
	if _, ok := AlignmentLerp(FractionalOffsetCenter, FractionalOffsetCenter, 0.5).(FractionalOffset); !ok {
		t.Error("identical FractionalOffset")
	}
}

func TestAlignmentGeometryLerpIdentical(t *testing.T) {
	if AlignmentGeometryLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	var alignment AlignmentGeometry = AlignmentTopLeft
	if AlignmentGeometryLerp(alignment, alignment, 0.5) != alignment {
		t.Error("identical")
	}
}

func TestAlignmentDirectionalLerpIdentical(t *testing.T) {
	if AlignmentDirectionalLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	alignment := AlignmentDirectionalTopStart
	if AlignmentDirectionalLerp(&alignment, &alignment, 0.5) != &alignment {
		t.Error("identical")
	}
}

func TestAlignmentGeometryInvariants(t *testing.T) {
	topStart := AlignmentDirectionalTopStart
	topEnd := AlignmentDirectionalTopEnd
	center := AlignmentCenter
	topLeft := AlignmentTopLeft
	topRight := AlignmentTopRight
	numbers := []float64{0.0, 1.0, -1.0, 2.0, 0.25, 0.5, 100.0, -999.75}

	expectAlign(t, topEnd.Mul(0.0).Add(topRight.Mul(0.0)), center)
	expectAlign(t, AlignmentGeometryMul(topEnd.Add(topRight), 0.0), topEnd.Mul(0.0).Add(topRight.Mul(0.0)))
	expectAlign(t, topStart.Add(topLeft), topLeft.Add(topStart))
	expectAlign(t, topStart.Add(topLeft).Resolve(ltr), topStart.Resolve(ltr).Plus(topLeft))
	expectAlign(t, topStart.Add(topLeft).Resolve(rtl), topStart.Resolve(rtl).Plus(topLeft))
	expectAlign(t, topStart.Add(topLeft).Resolve(ltr), topStart.Resolve(ltr).Add(topLeft))
	expectAlign(t, topStart.Add(topLeft).Resolve(rtl), topStart.Resolve(rtl).Add(topLeft))
	expectAlign(t, topStart.Resolve(ltr), topLeft)
	expectAlign(t, topStart.Resolve(rtl), topRight)
	expectAlign(t, topEnd.Mul(0.0), center)
	expectAlign(t, topLeft.Mul(0.0), center)
	expectAlign(t, topStart.Mul(1.0), topStart)
	expectAlign(t, topEnd.Mul(1.0), topEnd)
	expectAlign(t, topLeft.Mul(1.0), topLeft)
	expectAlign(t, topRight.Mul(1.0), topRight)
	for _, n := range numbers {
		expectAlign(t, topStart.Mul(n).Add(topStart), topStart.Mul(n+1.0))
		expectAlign(t, topEnd.Mul(n).Add(topEnd), topEnd.Mul(n+1.0))
		for _, m := range numbers {
			expectAlign(t, topStart.Mul(n).Add(topStart.Mul(m)), topStart.Mul(n+m))
		}
	}
	expectAlign(t, topStart.Plus(topStart).Plus(topStart), topStart.Mul(3.0)) // without using "add"
	for _, d := range ui.TextDirectionValues {
		x := dir(d)
		expectAlign(t, topEnd.Mul(0.0).Add(topRight.Mul(0.0)).Resolve(x), center.Add(center).Resolve(x))
		expectAlign(t, topEnd.Mul(0.0).Add(topLeft).Resolve(x), center.Add(topLeft).Resolve(x))
		expectAlign(t, topEnd.Mul(0.0).Resolve(x).Add(topLeft.Resolve(x)), center.Resolve(x).Add(topLeft.Resolve(x)))
		expectAlign(t, topEnd.Mul(0.0).Resolve(x).Add(topLeft), center.Resolve(x).Add(topLeft))
		expectAlign(t, topEnd.Mul(0.0).Resolve(x), center.Resolve(x))
	}
	expectNotAlign(t, topStart, topLeft)
	expectNotAlign(t, topEnd, topLeft)
	expectNotAlign(t, topStart, topRight)
	expectNotAlign(t, topEnd, topRight)
	expectNotAlign(t, topStart.Add(topLeft), topLeft)
	expectNotAlign(t, topStart.Add(topLeft), topStart)
}

func TestAlignmentGeometryResolve(t *testing.T) {
	cases := []struct {
		start, y float64
		d        *ui.TextDirection
		want     Alignment
	}{
		{0.25, 0.3, ltr, NewAlignment(0.25, 0.3)},
		{0.25, 0.3, rtl, NewAlignment(-0.25, 0.3)},
		{-0.25, 0.3, ltr, NewAlignment(-0.25, 0.3)},
		{-0.25, 0.3, rtl, NewAlignment(0.25, 0.3)},
		{1.25, 0.3, ltr, NewAlignment(1.25, 0.3)},
		{1.25, 0.3, rtl, NewAlignment(-1.25, 0.3)},
		{0.5, -0.3, ltr, NewAlignment(0.5, -0.3)},
		{0.5, -0.3, rtl, NewAlignment(-0.5, -0.3)},
	}
	for _, c := range cases {
		expectAlign(t, NewAlignmentDirectional(c.start, c.y).Resolve(c.d), c.want)
	}
	expectAlign(t, AlignmentDirectionalCenter.Resolve(ltr), AlignmentCenter)
	expectAlign(t, AlignmentDirectionalCenter.Resolve(rtl), AlignmentCenter)
	expectAlign(t, AlignmentDirectionalBottomEnd.Resolve(ltr), AlignmentBottomRight)
	expectAlign(t, AlignmentDirectionalBottomEnd.Resolve(rtl), AlignmentBottomLeft)
	expectAlign(t, NewAlignmentDirectional(1.0, 2.0), NewAlignmentDirectional(1.0, 2.0))
	expectNotAlign(t, NewAlignmentDirectional(1.0, 2.0), NewAlignmentDirectional(2.0, 1.0))
	expectAlign(t, AlignmentDirectionalCenterStart.Resolve(ltr), AlignmentDirectionalCenterEnd.Resolve(rtl))
	expectNotAlign(t, AlignmentDirectionalCenterStart.Resolve(ltr), AlignmentDirectionalCenterEnd.Resolve(ltr))
	expectNotAlign(t, AlignmentDirectionalCenterEnd.Resolve(ltr), AlignmentDirectionalCenterEnd.Resolve(rtl))
}

func TestAlignmentGeometryLerpAdHoc(t *testing.T) {
	mixed1 := NewAlignment(10.0, 20.0).Add(NewAlignmentDirectional(30.0, 50.0))
	mixed2 := NewAlignment(70.0, 110.0).Add(NewAlignmentDirectional(130.0, 170.0))
	mixed3 := NewAlignment(25.0, 42.5).Add(NewAlignmentDirectional(55.0, 80.0))
	for _, d := range ui.TextDirectionValues {
		direction := dir(d)
		expectAlign(t, AlignmentGeometryLerp(mixed1, mixed2, 0.0).Resolve(direction), mixed1.Resolve(direction))
		expectAlign(t, AlignmentGeometryLerp(mixed1, mixed2, 1.0).Resolve(direction), mixed2.Resolve(direction))
		expectAlign(t, AlignmentGeometryLerp(mixed1, mixed2, 0.25).Resolve(direction), mixed3.Resolve(direction))
	}
}

func TestAlignmentLerpCommutesWithResolve(t *testing.T) {
	offsets := []AlignmentGeometry{
		AlignmentTopLeft, AlignmentTopCenter, AlignmentTopRight,
		AlignmentDirectionalTopStart, AlignmentDirectionalTopCenter, AlignmentDirectionalTopEnd,
		AlignmentCenterLeft, AlignmentCenter, AlignmentCenterRight,
		AlignmentDirectionalCenterStart, AlignmentDirectionalCenter, AlignmentDirectionalCenterEnd,
		AlignmentBottomLeft, AlignmentBottomCenter, AlignmentBottomRight,
		AlignmentDirectionalBottomStart, AlignmentDirectionalBottomCenter, AlignmentDirectionalBottomEnd,
		NewAlignment(-1.0, 0.65),
		NewAlignmentDirectional(-1.0, 0.45),
		NewAlignmentDirectional(0.125, 0.625),
		NewAlignment(0.25, 0.875),
		NewAlignment(0.0625, 0.5625).Add(NewAlignmentDirectional(0.1875, 0.6875)),
		NewAlignmentDirectional(2.0, 3.0),
		NewAlignment(2.0, 3.0),
		NewAlignment(2.0, 3.0).Add(NewAlignmentDirectional(5.0, 3.0)),
		NewAlignment(10.0, 20.0).Add(NewAlignmentDirectional(30.0, 50.0)),
		NewAlignment(70.0, 110.0).Add(NewAlignmentDirectional(130.0, 170.0)),
		NewAlignment(25.0, 42.5).Add(NewAlignmentDirectional(55.0, 80.0)),
		nil,
	}
	times := []float64{0.25, 0.5, 0.75}
	orDefault := func(g, def AlignmentGeometry) AlignmentGeometry {
		if g == nil {
			return def
		}
		return g
	}
	for _, d := range ui.TextDirectionValues {
		direction := dir(d)
		defaultValue := AlignmentDirectionalCenter.Resolve(direction)
		for _, a := range offsets {
			resolvedA := orDefault(a, defaultValue).Resolve(direction)
			for _, b := range offsets {
				resolvedB := orDefault(b, defaultValue).Resolve(direction)
				approxExpect(t, AlignmentLerp(resolvedA, resolvedB, 0.0).AsAlignment(), resolvedA)
				approxExpect(t, AlignmentLerp(resolvedA, resolvedB, 1.0).AsAlignment(), resolvedB)
				approxExpect(t, orDefault(AlignmentGeometryLerp(a, b, 0.0), defaultValue).Resolve(direction), resolvedA)
				approxExpect(t, orDefault(AlignmentGeometryLerp(a, b, 1.0), defaultValue).Resolve(direction), resolvedB)
				for _, tt := range times {
					value := orDefault(AlignmentGeometryLerp(a, b, tt), defaultValue).Resolve(direction)
					approxExpect(t, value, AlignmentLerp(resolvedA, resolvedB, tt).AsAlignment())
					if value.X < math.Min(resolvedA.X, resolvedB.X) || value.X > math.Max(resolvedA.X, resolvedB.X) ||
						value.Y < math.Min(resolvedA.Y, resolvedB.Y) || value.Y > math.Max(resolvedA.Y, resolvedB.Y) {
						t.Errorf("%v out of range", value)
					}
				}
			}
		}
	}
}

func TestAlignmentGeometryAddSubtract(t *testing.T) {
	var directional AlignmentGeometry = NewAlignmentDirectional(1.0, 2.0)
	var normal AlignmentGeometry = NewAlignment(3.0, 5.0)
	expectAlign(t, directional.Add(normal).Resolve(ltr), NewAlignment(4.0, 7.0))
	expectAlign(t, directional.Add(normal).Resolve(rtl), NewAlignment(2.0, 7.0))
	expectAlign(t, normal.Add(normal), AlignmentGeometryMul(normal, 2.0))
	expectAlign(t, directional.Add(directional), AlignmentGeometryMul(directional, 2.0))
}

func TestAlignmentGeometryOperators(t *testing.T) {
	ad := NewAlignmentDirectional(1.0, 2.0)
	expectAlign(t, ad.Mul(2.0), NewAlignmentDirectional(2.0, 4.0))
	expectAlign(t, ad.Div(2.0), NewAlignmentDirectional(0.5, 1.0))
	expectAlign(t, ad.Mod(2.0), AlignmentDirectionalCenterEnd)
	expectAlign(t, ad.TruncDiv(2.0), AlignmentDirectionalBottomCenter)
	for _, d := range ui.TextDirectionValues {
		direction := dir(d)
		expectAlign(t, AlignmentCenter.Add(ad.Mul(2.0)).Resolve(direction), NewAlignmentDirectional(2.0, 4.0).Resolve(direction))
		expectAlign(t, AlignmentCenter.Add(ad.Div(2.0)).Resolve(direction), NewAlignmentDirectional(0.5, 1.0).Resolve(direction))
		expectAlign(t, AlignmentCenter.Add(ad.Mod(2.0)).Resolve(direction), AlignmentDirectionalCenterEnd.Resolve(direction))
		expectAlign(t, AlignmentCenter.Add(ad.TruncDiv(2.0)).Resolve(direction), AlignmentDirectionalBottomCenter.Resolve(direction))
	}
	a := NewAlignment(1.0, 2.0)
	expectAlign(t, a.Mul(2.0), NewAlignment(2.0, 4.0))
	expectAlign(t, a.Div(2.0), NewAlignment(0.5, 1.0))
	expectAlign(t, a.Mod(2.0), AlignmentCenterRight)
	expectAlign(t, a.TruncDiv(2.0), AlignmentBottomCenter)
}

func TestAlignmentGeometryBinaryOperators(t *testing.T) {
	expectAlign(t, NewAlignment(1.0, 2.0).Plus(NewAlignment(3.0, 5.0)), NewAlignment(4.0, 7.0))
	expectAlign(t, NewAlignment(1.0, 2.0).Sub(NewAlignment(3.0, 5.0)), NewAlignment(-2.0, -3.0))
	expectAlign(t, NewAlignmentDirectional(1.0, 2.0).Plus(NewAlignmentDirectional(3.0, 5.0)), NewAlignmentDirectional(4.0, 7.0))
	expectAlign(t, NewAlignmentDirectional(1.0, 2.0).Sub(NewAlignmentDirectional(3.0, 5.0)), NewAlignmentDirectional(-2.0, -3.0))
}

func TestAlignmentGeometryToString(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{NewAlignment(1.0001, 2.0001).String(), "Alignment(1.0, 2.0)"},
		{AlignmentCenter.String(), "Alignment.center"},
		{AlignmentBottomLeft.Add(AlignmentDirectionalCenterEnd).String(), "Alignment.bottomLeft + AlignmentDirectional.centerEnd"},
		{NewAlignment(0.0001, 0.0001).String(), "Alignment(0.0, 0.0)"},
		{AlignmentDirectionalCenter.String(), "AlignmentDirectional.center"},
		{AlignmentBottomRight.Add(AlignmentDirectionalBottomEnd).String(), "Alignment(1.0, 2.0) + AlignmentDirectional.centerEnd"},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

func TestAlignmentGeometryFactories(t *testing.T) {
	expectAlign(t, NewAlignmentGeometryXY(4, 5), NewAlignment(4, 5))
	expectAlign(t, NewAlignmentGeometryDirectional(4, 5), NewAlignmentDirectional(4, 5))
	if _, ok := NewAlignmentGeometryXY(4, 5).(Alignment); !ok {
		t.Error("xy is not an Alignment")
	}
	if _, ok := NewAlignmentGeometryDirectional(4, 5).(AlignmentDirectional); !ok {
		t.Error("directional is not an AlignmentDirectional")
	}
}

func TestAlignmentGeometryStaticMembers(t *testing.T) {
	for _, c := range [][2]AlignmentGeometry{
		{AlignmentGeometryTopLeft, AlignmentTopLeft},
		{AlignmentGeometryTopCenter, AlignmentTopCenter},
		{AlignmentGeometryTopRight, AlignmentTopRight},
		{AlignmentGeometryTopStart, AlignmentDirectionalTopStart},
		{AlignmentGeometryTopEnd, AlignmentDirectionalTopEnd},
		{AlignmentGeometryCenterLeft, AlignmentCenterLeft},
		{AlignmentGeometryCenter, AlignmentCenter},
		{AlignmentGeometryCenterRight, AlignmentCenterRight},
		{AlignmentGeometryCenterStart, AlignmentDirectionalCenterStart},
		{AlignmentGeometryCenterEnd, AlignmentDirectionalCenterEnd},
		{AlignmentGeometryBottomLeft, AlignmentBottomLeft},
		{AlignmentGeometryBottomCenter, AlignmentBottomCenter},
		{AlignmentGeometryBottomRight, AlignmentBottomRight},
		{AlignmentGeometryBottomStart, AlignmentDirectionalBottomStart},
		{AlignmentGeometryBottomEnd, AlignmentDirectionalBottomEnd},
	} {
		if c[0] != c[1] {
			t.Errorf("%v != %v", c[0], c[1])
		}
	}
}

func expectNoTextDirection(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		e, ok := r.(error)
		var fe *foundation.FlutterError
		if !ok || !errors.As(e, &fe) {
			t.Fatalf("got %v, want a FlutterError", r)
		}
		if m := fe.Message(); !strings.Contains(m, "No TextDirection found.") ||
			!strings.Contains(m, "without a Directionality ancestor") {
			t.Errorf("message %q", m)
		}
	}()
	f()
}

func TestAlignmentDirectionalResolveNullTextDirection(t *testing.T) {
	expectNoTextDirection(t, func() { NewAlignmentDirectional(1.0, 2.0).Resolve(nil) })
}

func TestMixedAlignmentResolveNullTextDirection(t *testing.T) {
	expectNoTextDirection(t, func() { NewAlignment(5.0, 6.0).Add(NewAlignmentDirectional(15.0, 16.0)).Resolve(nil) })
}

// Not in Flutter's tests: TextAlignVertical (alignment.dart).
func TestTextAlignVertical(t *testing.T) {
	if s := TextAlignVerticalTop.String(); s != "TextAlignVertical(y: -1.0)" {
		t.Error(s)
	}
	if s := NewTextAlignVertical(0.5).String(); s != "TextAlignVertical(y: 0.5)" {
		t.Error(s)
	}
	if TextAlignVerticalCenter.Y != 0 || TextAlignVerticalBottom.Y != 1 {
		t.Error("constants")
	}
	for _, y := range []float64{-1.0000001, 1.0000001, math.NaN()} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewTextAlignVertical(%v) did not assert", y)
				}
			}()
			NewTextAlignVertical(y)
		}()
	}
}
