// Port of packages/flutter/test/gestures/transformed_monodrag_test.dart
// (Flutter 3.50.1): 0 of 8 tests as written.
//
// Skipped: all 8 are widget tests (GestureDetector inside Transform,
// tester.drag) — needs widgets (GestureDetector/Listener not ported). What
// they check (local coordinates of transformed events; kTouchSlop measured
// in the global coordinate space when scaled up, scaled down or rotated (rotation: the difftest)
// 45°) is checked here without widgets, by giving the recognizers
// transformed down events as hit testing would, and exactly against
// Flutter by the recognizers_drag difftest's transform scenes.
package gestures_test

import (
	"math"
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// transformedSlop drags vertically by dy (global) with the recognizer's
// events transformed by m, and reports whether the drag started.
func transformedSlop(t *testing.T, m *vm.Matrix4, dy float64) (started bool, local, global []float64) {
	t.Run("drag", func(t *testing.T) {
		testGesture(t, func(tester *flutter_test.GestureTester) {
			drag := NewVerticalDragGestureRecognizer()
			competing := NewHorizontalDragGestureRecognizer()
			competing.OnStart = func(*DragStartDetails) {}
			drag.OnStart = func(d *DragStartDetails) {
				started = true
				local = []float64{d.LocalPosition().Dx, d.LocalPosition().Dy}
				global = []float64{d.GlobalPosition().Dx, d.GlobalPosition().Dy}
			}
			pointer := touch(1)
			down := pointer.Down(off(400, 300))
			drag.AddPointer(down.Transformed(m).(*PointerDownEvent))
			competing.AddPointer(down.Transformed(m).(*PointerDownEvent))
			tester.CloseArena(1)
			tester.Route(down)
			tester.Route(pointer.Move(off(400, 300+dy)))
			tester.Route(pointer.Up())
			drag.Dispose()
			competing.Dispose()
		})
	})
	return
}

func TestTransformedMonodragSlopInGlobalSpace(t *testing.T) {
	scale := func(s float64) *vm.Matrix4 { return vm.NewMatrix4(s, 0, 0, 0, 0, s, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1) }
	for _, m := range []*vm.Matrix4{scale(2), scale(0.5), scale(3.25)} {
		if started, _, _ := transformedSlop(t, m, KTouchSlop-1); started {
			t.Errorf("%v: started below the global slop", m)
		}
		started, local, global := transformedSlop(t, m, KTouchSlop+1)
		if !started {
			t.Errorf("%v: not started above the global slop", m)
		}
		if global[0] != 400 || global[1] != 300+KTouchSlop+1 {
			t.Errorf("global %v", global)
		}
		want := PointerEventTransformPosition(m, off(400, 300+KTouchSlop+1))
		if math.Abs(local[0]-want.Dx) > 1e-9 || math.Abs(local[1]-want.Dy) > 1e-9 {
			t.Errorf("local %v, want %v", local, want)
		}
	}
}
