package painting

// The rest of packages/flutter/test/painting/matrix_utils_test.dart
// (Flutter 3.50.1); see matrix_utils_test.go's header.

import (
	"math"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

func TestGetAsTranslationRotations(t *testing.T) {
	expectTranslation(t, vm.NewMatrix4RotationX(1.0), nil)
	test := vm.NewMatrix4Identity()
	expectTranslation(t, test, off(0, 0))
	test.RotateX(2.0)
	expectTranslation(t, test, nil)
	test = vm.NewMatrix4Identity()
	test.Scale(2.0)
	expectTranslation(t, test, nil)
	test = vm.NewMatrix4Identity()
	test.Translate(2.0, -2.0)
	expectTranslation(t, test, off(2, -2))
	test.Translate(4.0, 8.0)
	expectTranslation(t, test, off(6, 6))
}

func TestTransformRectVeryLargeFiniteValues(t *testing.T) {
	evilRect := ui.NewRectFromLTRB(0.0, -1.7976931348623157e+308, 800.0, 1.7976931348623157e+308)
	transform := vm.NewMatrix4Identity()
	transform.Translate(10.0)
	if r := MatrixUtils.TransformRect(transform, evilRect); !r.IsFinite() {
		t.Fatal(r)
	}
}

func TestMultiplyInPlace(t *testing.T) {
	originalA := vm.NewMatrix4Identity()
	originalA.Translate(10.0)
	a := originalA.Clone()
	b := vm.NewMatrix4Identity()
	b.Translate(20.0)
	result := a.Multiplied(b)
	MatrixUtils.MultiplyInPlace(a, b)
	if !a.Equal(originalA) || !b.Equal(result) || b.GetTranslation().X() == 20.0 {
		t.Fatal(a, b)
	}
	// A general product, element by element.
	g := vm.NewMatrix4(2, 1, 0.5, 0.1, -1, 3, 0.25, 0.2, 0.3, -0.7, 4, 0.05, 5, -2, 1, 1.5)
	h := vm.NewMatrix4(0.3, 2, -1, 0, 1, 1, 1, 1, 7, -0.5, 2, 0.25, 1, 2, 3, 4)
	want := g.Multiplied(h)
	MatrixUtils.MultiplyInPlace(g, h)
	if !h.Equal(want) {
		t.Fatal(h, want)
	}
}

func TestCylindricalProjectionTransform(t *testing.T) {
	zero := 0.0
	if m := MatrixUtils.CreateCylindricalProjectionTransform(0, 0, CreateCylindricalProjectionTransformOptions{Perspective: &zero}); !m.Equal(vm.NewMatrix4Identity()) {
		t.Fatal("identity", m)
	}
	if m := MatrixUtils.CreateCylindricalProjectionTransform(0, math.Pi/2.0, CreateCylindricalProjectionTransformOptions{Perspective: &zero}); !m.Equal(vm.NewMatrix4RotationX(math.Pi / 2.0)) {
		t.Fatal("rotate with no radius", m)
	}
	if m := MatrixUtils.CreateCylindricalProjectionTransform(1000000.0, 0, CreateCylindricalProjectionTransformOptions{Perspective: &zero}); !m.Equal(vm.NewMatrix4Identity()) {
		t.Fatal("radius does not change scale", m)
	}
	actual := MatrixUtils.CreateCylindricalProjectionTransform(100.0, math.Pi/3.0)
	want := []float64{1.0, 0.0, 0.0, 0.0, 0.0, 0.5, 0.8660254037844386, -0.0008660254037844386,
		0.0, -0.8660254037844386, 0.5, -0.0005, 0.0, -86.60254037844386, -50.0, 1.05}
	for i, w := range want {
		if math.Abs(actual.Storage()[i]-w) > 1e-10 {
			t.Fatalf("[%d] %v want %v", i, actual.Storage()[i], w)
		}
	}
	// Horizontal orientation rotates around Y.
	h := AxisHorizontal
	if m := MatrixUtils.CreateCylindricalProjectionTransform(0, 1.0, CreateCylindricalProjectionTransformOptions{Perspective: &zero, Orientation: &h}); !m.Equal(vm.NewMatrix4RotationY(1.0)) {
		t.Fatal("horizontal", m)
	}
	bad := 1.5
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("perspective > 1 did not assert")
			}
		}()
		MatrixUtils.CreateCylindricalProjectionTransform(1, 1, CreateCylindricalProjectionTransformOptions{Perspective: &bad})
	}()
}

func TestForceToPoint(t *testing.T) {
	forcedOffset := ui.NewOffset(20, -30)
	forced := MatrixUtils.ForceToPoint(forcedOffset)
	for _, p := range []ui.Offset{forcedOffset, ui.OffsetZero, ui.NewOffset(1, 1), ui.NewOffset(-1, -1),
		ui.NewOffset(-20, 30), ui.NewOffset(-1.2344, 1422434.23)} {
		if got := MatrixUtils.TransformPoint(forced, p); got != forcedOffset {
			t.Fatal(p, got)
		}
	}
}

func rectWithin(t *testing.T, got, want ui.Rect, distance float64) {
	t.Helper()
	if math.Abs(got.Left-want.Left) > distance || math.Abs(got.Top-want.Top) > distance ||
		math.Abs(got.Right-want.Right) > distance || math.Abs(got.Bottom-want.Bottom) > distance {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestTransformRectNoPerspective(t *testing.T) {
	r := ui.NewRectFromLTRB(10, 20, 30, 40)
	if got := MatrixUtils.TransformRect(vm.NewMatrix4Identity(), r); got != r {
		t.Fatal(got)
	}
	if got := MatrixUtils.TransformRect(vm.NewMatrix4Diagonal3Values(2, 2, 2), r); got != ui.NewRectFromLTRB(20, 40, 60, 80) {
		t.Fatal(got)
	}
	rectWithin(t, MatrixUtils.TransformRect(vm.NewMatrix4RotationZ(math.Pi/2.0), r), ui.NewRectFromLTRB(-40.0, 10.0, -20.0, 30.0), 0.00001)
	// Negative scales flip which corner is the minimum.
	if got := MatrixUtils.TransformRect(vm.NewMatrix4Diagonal3Values(-1, -2, 1), r); got != ui.NewRectFromLTRB(-30, -80, -10, -40) {
		t.Fatal(got)
	}
	skew := vm.NewMatrix4Identity()
	skew.SetEntry(0, 1, -0.5) // x' = x - 0.5y
	skew.SetEntry(1, 0, 0.25) // y' = y + 0.25x
	if got := MatrixUtils.TransformRect(skew, r); got != ui.NewRectFromLTRB(-10, 22.5, 20, 47.5) {
		t.Fatal(got)
	}
}

func transformPointVectorWise(m *vm.Matrix4, p ui.Offset) ui.Offset {
	v := m.PerspectiveTransform(vm.NewVector3(p.Dx, p.Dy, 0))
	return ui.NewOffset(v.X(), v.Y())
}

func TestTransformRectWithPerspective(t *testing.T) {
	transform := MatrixUtils.CreateCylindricalProjectionTransform(10.0, math.Pi/8.0,
		CreateCylindricalProjectionTransformOptions{Perspective: ptrF(0.3)})
	for i := 1; i < 10000; i++ {
		fi := float64(i)
		rect := ui.NewRectFromLTRB(11.0*fi, 12.0*fi, 15.0*fi, 18.0*fi)
		p1, p2 := transformPointVectorWise(transform, rect.TopLeft()), transformPointVectorWise(transform, rect.TopRight())
		p3, p4 := transformPointVectorWise(transform, rect.BottomLeft()), transformPointVectorWise(transform, rect.BottomRight())
		golden := ui.NewRectFromLTRB(math.Min(p1.Dx, math.Min(p2.Dx, math.Min(p3.Dx, p4.Dx))),
			math.Min(p1.Dy, math.Min(p2.Dy, math.Min(p3.Dy, p4.Dy))),
			math.Max(p1.Dx, math.Max(p2.Dx, math.Max(p3.Dx, p4.Dx))),
			math.Max(p1.Dy, math.Max(p2.Dy, math.Max(p3.Dy, p4.Dy))))
		rectWithin(t, MatrixUtils.TransformRect(transform, rect), golden, 0.00001)
	}
}

func ptrF(v float64) *float64 { return &v }

func TestTransformRectNonFinite(t *testing.T) {
	// An infinite width goes through the corner-by-corner path.
	r := ui.NewRectFromLTRB(math.Inf(-1), 0, 10, 10)
	m := vm.NewMatrix4TranslationValues(5, 5, 0)
	// 0 * -Infinity is NaN in the y sums, as in Dart.
	if got := MatrixUtils.TransformRect(m, r); got.Left != math.Inf(-1) || got.Right != 15 || !math.IsNaN(got.Top) || !math.IsNaN(got.Bottom) {
		t.Fatal(got)
	}
	p := vm.NewMatrix4Identity()
	p.SetEntry(3, 1, 0.5) // w = 0.5y + 1
	// The right corners' w is 0 * Infinity = NaN, so they never win a
	// comparison (Dart's behavior too): only the left corners count.
	if got := MatrixUtils.TransformRect(p, ui.NewRectFromLTRB(0, 0, math.Inf(1), 2)); got != ui.NewRectFromLTRB(0, 0, 0, 1) {
		t.Fatal(got)
	}
}

func TestInverseTransformRect(t *testing.T) {
	r := ui.NewRectFromLTRB(10, 20, 30, 40)
	if got := MatrixUtils.InverseTransformRect(vm.NewMatrix4Identity(), r); got != r {
		t.Fatal(got)
	}
	m := vm.NewMatrix4Diagonal3Values(2, 4, 1)
	before := m.Clone()
	if got := MatrixUtils.InverseTransformRect(m, r); got != ui.NewRectFromLTRB(5, 5, 15, 10) || !m.Equal(before) {
		t.Fatal(got, m)
	}
}
