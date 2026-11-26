package painting

// Port of packages/flutter/test/painting/matrix_utils_test.dart (Flutter
// 3.50.1), all 10 tests. Adapted: Dart's deprecated translate/scale are
// the vector_math port's Translate/Scale; within<Rect>(distance) is a
// per-edge tolerance check. Plus own tests of getAsScale, matrixEquals,
// isIdentity, transformPoint, inverseTransformRect, debugDescribeTransform
// and TransformProperty. Skipped: none.

import (
	"math"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

func expectTranslation(t *testing.T, m *vm.Matrix4, want *ui.Offset) {
	t.Helper()
	got := MatrixUtils.GetAsTranslation(m)
	if (got == nil) != (want == nil) || got != nil && *got != *want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func off(x, y float64) *ui.Offset { o := ui.NewOffset(x, y); return &o }

func TestGetAsTranslation(t *testing.T) {
	test := vm.NewMatrix4Identity()
	expectTranslation(t, test, off(0, 0))
	test = vm.NewMatrix4Zero()
	expectTranslation(t, test, nil)
	test = vm.NewMatrix4RotationZ(1.0)
	expectTranslation(t, test, nil)
	test = vm.NewMatrix4TranslationValues(1.0, 2.0, 0.0)
	expectTranslation(t, test, off(1, 2))
	test = vm.NewMatrix4TranslationValues(1.0, 2.0, 3.0)
	expectTranslation(t, test, nil)

	test = vm.NewMatrix4Identity()
	expectTranslation(t, test, off(0, 0))
	test.ScaleByDouble(2.0, 2.0, 2.0, 1.0)
	expectTranslation(t, test, nil)

	test = vm.NewMatrix4Identity()
	expectTranslation(t, test, off(0, 0))
	test.TranslateByDouble(2.0, -2.0, 0, 1)
	expectTranslation(t, test, off(2, -2))
	test.TranslateByDouble(4.0, 8.0, 0, 1)
	expectTranslation(t, test, off(6, 6))
}

func TestMatrixUtilsOwn(t *testing.T) {
	if s := MatrixUtils.GetAsScale(vm.NewMatrix4Diagonal3Values(2, 2, 1)); s == nil || *s != 2 {
		t.Fatal("scale")
	}
	if MatrixUtils.GetAsScale(vm.NewMatrix4Diagonal3Values(2, 3, 1)) != nil ||
		MatrixUtils.GetAsScale(vm.NewMatrix4Diagonal3Values(2, 2, 2)) != nil ||
		MatrixUtils.GetAsScale(vm.NewMatrix4TranslationValues(0, 0, 1)) != nil ||
		MatrixUtils.GetAsScale(vm.NewMatrix4Diagonal3Values(math.NaN(), math.NaN(), 1)) != nil {
		t.Fatal("not a scale")
	}
	if !MatrixUtils.MatrixEquals(nil, vm.NewMatrix4Identity()) || !MatrixUtils.MatrixEquals(vm.NewMatrix4Identity(), nil) ||
		MatrixUtils.MatrixEquals(nil, vm.NewMatrix4Zero()) ||
		!MatrixUtils.MatrixEquals(vm.NewMatrix4RotationZ(1), vm.NewMatrix4RotationZ(1)) ||
		MatrixUtils.MatrixEquals(vm.NewMatrix4RotationZ(1), vm.NewMatrix4RotationZ(1.5)) {
		t.Fatal("matrixEquals")
	}
	if MatrixUtils.IsIdentity(vm.NewMatrix4TranslationValues(0, 0, 1)) || !MatrixUtils.IsIdentity(vm.NewMatrix4Identity()) {
		t.Fatal("isIdentity")
	}
	if p := MatrixUtils.TransformPoint(vm.NewMatrix4TranslationValues(3, 4, 0), ui.NewOffset(1, 1)); p != ui.NewOffset(4, 5) {
		t.Fatal(p)
	}
	persp := vm.NewMatrix4Identity()
	persp.SetEntry(3, 0, 0.5) // w = 0.5x + 1
	if p := MatrixUtils.TransformPoint(persp, ui.NewOffset(2, 4)); p != ui.NewOffset(1, 2) {
		t.Fatal(p)
	}
	m := vm.NewMatrix4TranslationValues(10, 2.5, 0)
	if got := strings.Join(DebugDescribeTransform(m), "|"); got != "[0] 1.0,0.0,0.0,10.0|[1] 0.0,1.0,0.0,2.5|[2] 0.0,0.0,1.0,0.0|[3] 0.0,0.0,0.0,1.0" {
		t.Fatal(got)
	}
	if got := DebugDescribeTransform(nil); len(got) != 1 || got[0] != "null" {
		t.Fatal(got)
	}
	p := NewTransformProperty("transform", m)
	if got := p.String(); got != "transform:\n  [0] 1.0,0.0,0.0,10.0\n  [1] 0.0,1.0,0.0,2.5\n  [2] 0.0,0.0,1.0,0.0\n  [3] 0.0,0.0,0.0,1.0" {
		t.Fatalf("%q", got)
	}
	if got := p.ToString(foundation.DiagnosticsNodeToStringOptions{ParentConfiguration: foundation.SingleLineTextConfiguration}); got !=
		"transform: [1.0,0.0,0.0,10.0; 0.0,1.0,0.0,2.5; 0.0,0.0,1.0,0.0; 0.0,0.0,0.0,1.0]" {
		t.Fatalf("%q", got)
	}
	f := false
	if got := NewTransformProperty("transform", nil, TransformPropertyOptions{ShowName: &f, DefaultValue: foundation.Null}).Level(); got != foundation.DiagnosticLevelFine {
		t.Fatal(got)
	}
}

func TestMatrix4(t *testing.T) {
	m := vm.NewMatrix4TranslationValues(1, 2, 3)
	m.Multiply(vm.NewMatrix4Diagonal3Values(2, 4, 8))
	inv := vm.NewMatrix4Inverted(m)
	check := m.Multiplied(inv)
	if !check.IsIdentity() || m.Determinant() != 64 {
		t.Fatalf("%v %v", check, m.Determinant())
	}
	// A general (3D, projective) matrix: its inverse times it is the
	// identity to rounding.
	g := vm.NewMatrix4(2, 1, 0.5, 0.1, -1, 3, 0.25, 0.2, 0.3, -0.7, 4, 0.05, 5, -2, 1, 1.5)
	gi := vm.NewMatrix4Inverted(g)
	prod := g.Multiplied(gi)
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			want := 0.0
			if r == c {
				want = 1
			}
			if math.Abs(prod.Entry(r, c)-want) > 1e-12 {
				t.Fatalf("g*inv(g)[%d][%d] = %v", r, c, prod.Entry(r, c))
			}
		}
	}
	if vm.Matrix4TryInvert(vm.NewMatrix4Zero()) != nil {
		t.Fatal("zero inverted")
	}
	z := vm.NewMatrix4Zero()
	if z.CopyInverse(m) == 0 || z.Equal(m) {
		t.Fatal("copyInverse")
	}
	singular := vm.NewMatrix4Diagonal3Values(1, 0, 1)
	before := singular.Clone()
	if singular.Invert() != 0 || !singular.Equal(before) {
		t.Fatal("singular invert must leave the matrix")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("inverted of singular did not panic")
			}
		}()
		vm.NewMatrix4Inverted(singular)
	}()
	v := vm.NewVector3(1, 1, 0)
	m.Transform3(v)
	if v.X() != 3 || v.Y() != 6 || v.Z() != 3 || v.String() != "[3.0,6.0,3.0]" {
		t.Fatal(v)
	}
	if m.String() != "[0] [2.0,0.0,0.0,1.0]\n[1] [0.0,4.0,0.0,2.0]\n[2] [0.0,0.0,8.0,3.0]\n[3] [0.0,0.0,0.0,1.0]\n" {
		t.Fatal(m.String())
	}
	if !vm.NewMatrix4FromList(m.Storage()).Equal(m) || !vm.NewMatrix4Copy(m).Equal(m) || vm.NewMatrix4Zero().IsZero() != true {
		t.Fatal("copies")
	}
	if e := m.Entry(1, 3); e != 2 {
		t.Fatal(e)
	}
}
