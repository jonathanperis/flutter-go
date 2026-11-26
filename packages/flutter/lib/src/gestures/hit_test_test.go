// Port of packages/flutter/test/gestures/hit_test_test.dart (Flutter
// 3.50.1): all 3 tests. No skips. MyHitTestResult's public wrappers of the
// protected push/pop methods are unnecessary (they are exported in Go).
package gestures_test

import (
	"slices"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

type dummyHitTestTarget struct{}

func (*dummyHitTestTarget) HandleEvent(PointerEvent, HitTestEntry) {}

func pathOf(r *HitTestResult) []HitTestEntry { return slices.Collect(r.Path()) }

func TestHitTestWrappedResultSharesPath(t *testing.T) {
	entry1 := NewHitTestEntry(&dummyHitTestTarget{})
	entry2 := NewHitTestEntry(&dummyHitTestTarget{})
	entry3 := NewHitTestEntry(&dummyHitTestTarget{})
	transform := vm.NewMatrix4TranslationValues(40.0, 150.0, 0.0)

	wrapped := NewHitTestResult()
	wrapped.PushTransform(transform)
	wrapped.Add(entry1)
	if !slices.Equal(pathOf(wrapped), []HitTestEntry{entry1}) || !entry1.Transform().Equal(transform) {
		t.Fatal("entry1")
	}
	wrapping := NewHitTestResultWrap(wrapped)
	if !slices.Equal(pathOf(wrapping), []HitTestEntry{entry1}) {
		t.Fatal("wrapping path")
	}
	wrapping.Add(entry2)
	want := []HitTestEntry{entry1, entry2}
	if !slices.Equal(pathOf(wrapping), want) || !slices.Equal(pathOf(wrapped), want) || !entry2.Transform().Equal(transform) {
		t.Fatal("entry2")
	}
	wrapped.Add(entry3)
	want = []HitTestEntry{entry1, entry2, entry3}
	if !slices.Equal(pathOf(wrapping), want) || !slices.Equal(pathOf(wrapped), want) || !entry3.Transform().Equal(transform) {
		t.Fatal("entry3")
	}
	if s := wrapped.String(); !strings.HasPrefix(s, "HitTestResult(HitTestEntry<HitTestTarget>#") {
		t.Errorf("toString %q", s)
	}
	if s := NewHitTestResult().String(); s != "HitTestResult(<empty path>)" {
		t.Errorf("toString %q", s)
	}
}

func currentTransform(r *HitTestResult) *vm.Matrix4 {
	entry := NewHitTestEntry(&dummyHitTestTarget{})
	r.Add(entry)
	return entry.Transform()
}

func expectMatrix(t *testing.T, got, want *vm.Matrix4) {
	t.Helper()
	if !got.Equal(want) {
		t.Fatalf("got\n%v\nwant\n%v", got, want)
	}
}

func TestHitTestPushPopTransforms(t *testing.T) {
	result := NewHitTestResult()
	m1 := vm.NewMatrix4TranslationValues(10, 20, 0)
	m2 := vm.NewMatrix4RotationZ(1)
	m3 := vm.NewMatrix4Diagonal3Values(1.1, 1.2, 1.0)

	result.PushTransform(m1)
	expectMatrix(t, currentTransform(result), m1)
	result.PushTransform(m2)
	expectMatrix(t, currentTransform(result), m2.Multiplied(m1))
	expectMatrix(t, currentTransform(result), m2.Multiplied(m1))

	wrapped := NewHitTestResultWrap(result)
	expectMatrix(t, currentTransform(wrapped), m2.Multiplied(m1))

	result.PushTransform(m3)
	expectMatrix(t, currentTransform(result), m3.Multiplied(m2).Multiplied(m1))
	expectMatrix(t, currentTransform(wrapped), m3.Multiplied(m2).Multiplied(m1))

	result.PopTransform()
	result.PopTransform()
	expectMatrix(t, currentTransform(result), m1)

	result.PopTransform()
	result.PushTransform(m3)
	expectMatrix(t, currentTransform(result), m3)

	result.PushTransform(m2)
	expectMatrix(t, currentTransform(result), m2.Multiplied(m3))
}

func TestHitTestPushPopOffsets(t *testing.T) {
	result := NewHitTestResult()
	m1 := vm.NewMatrix4RotationZ(1)
	m2 := vm.NewMatrix4Diagonal3Values(1.1, 1.2, 1.0)
	o3 := ui.NewOffset(10, 20)
	m3 := vm.NewMatrix4TranslationValues(o3.Dx, o3.Dy, 0.0)

	result.PushOffset(o3)
	expectMatrix(t, currentTransform(result), m3)
	result.PopTransform()

	result.PushOffset(o3)
	result.PushTransform(m1)
	expectMatrix(t, currentTransform(result), m1.Multiplied(m3))
	expectMatrix(t, currentTransform(result), m1.Multiplied(m3))

	wrapped := NewHitTestResultWrap(result)
	expectMatrix(t, currentTransform(wrapped), m1.Multiplied(m3))

	result.PushTransform(m2)
	expectMatrix(t, currentTransform(result), m2.Multiplied(m1).Multiplied(m3))
	expectMatrix(t, currentTransform(wrapped), m2.Multiplied(m1).Multiplied(m3))

	result.PopTransform()
	result.PopTransform()
	result.PopTransform()
	expectMatrix(t, currentTransform(result), vm.NewMatrix4Identity())

	result.PushTransform(m2)
	result.PushOffset(o3)
	result.PushTransform(m1)
	expectMatrix(t, currentTransform(result), m1.Multiplied(m3).Multiplied(m2))

	result.PopTransform()
	expectMatrix(t, currentTransform(result), m3.Multiplied(m2))
}

// Not in Flutter's tests: a matrix with perspective is refused.
func TestHitTestPushTransformRejectsPerspective(t *testing.T) {
	m := vm.NewMatrix4Identity()
	m.SetEntry(3, 2, 0.001)
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(error).Error(), "removePerspectiveTransform") {
			t.Fatalf("recovered %v", r)
		}
	}()
	NewHitTestResult().PushTransform(m)
}
