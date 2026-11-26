package animation

// Port of packages/flutter/test/animation/tween_test.dart (Flutter 3.50.1):
// 15 of 18 tests.
//
// Skipped (painting's tweens, not ported yet): "BorderTween nullable
// test", "Matrix4Tween" (also needs vector_math's Matrix4),
// "BorderRadiusTween nullable test".
//
// The Object case uses a Go struct named Object (Dart's Object has no
// operators, like any Go struct without Add/Sub/Mul).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

const kApiDocsLink = "See \"Types with special considerations\" at https://api.flutter.dev/flutter/animation/Tween-class.html for more information."

func diagnosticsOf(t *testing.T, f func()) []string {
	t.Helper()
	err := throwsFlutterError(t, f)
	var out []string
	for _, d := range err.Diagnostics {
		out = append(out, d.ToString())
	}
	return out
}

func expectStrings(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("[%d] got %q\nwant %q", i, got[i], want[i])
		}
	}
}

// Object stands for Dart's Object in the first test.
type Object struct{ _ int }

func TestTweenThrowsForObject(t *testing.T) {
	objectTween := NewTween(TweenOptions[Object]{Begin: Object{}, End: Object{}})
	expectStrings(t, diagnosticsOf(t, func() { objectTween.Transform(0.1) }),
		`Cannot lerp between "Instance of 'Object'" and "Instance of 'Object'".`,
		"The type Object might not fully implement `+`, `-`, and/or `*`. "+kApiDocsLink,
		`There may be a dedicated "ObjectTween" for this type, or you may need to create one.`)
}

func TestTweenThrowsForColor(t *testing.T) {
	colorTween := NewTween(TweenOptions[ui.Color]{Begin: ui.NewColor(0xFF000000), End: ui.NewColor(0xFFFFFFFF)})
	expectStrings(t, diagnosticsOf(t, func() { colorTween.Transform(0.1) }),
		`Cannot lerp between "`+ui.NewColor(0xff000000).String()+`" and "`+ui.NewColor(0xffffffff).String()+`".`,
		"The type Color might not fully implement `+`, `-`, and/or `*`. "+kApiDocsLink,
		"To lerp colors, consider ColorTween instead.")
}

func TestTweenThrowsForRect(t *testing.T) {
	rectTween := NewTween(TweenOptions[ui.Rect]{Begin: ui.NewRectFromLTWH(0, 0, 10, 10), End: ui.NewRectFromLTWH(2, 2, 2, 2)})
	expectStrings(t, diagnosticsOf(t, func() { rectTween.Transform(0.1) }),
		`Cannot lerp between "Rect.fromLTRB(0.0, 0.0, 10.0, 10.0)" and "Rect.fromLTRB(2.0, 2.0, 4.0, 4.0)".`,
		"The type Rect might not fully implement `+`, `-`, and/or `*`. "+kApiDocsLink,
		"To lerp rects, consider RectTween instead.")
}

func TestTweenThrowsForInt(t *testing.T) {
	colorTween := NewTween(TweenOptions[int]{Begin: 0, End: 1})
	expectStrings(t, diagnosticsOf(t, func() { colorTween.Transform(0.1) }),
		`Cannot lerp between "0" and "1".`,
		"The type int returned a double after multiplication with a double value. "+kApiDocsLink,
		"To lerp int values, consider IntTween or StepTween instead.")
}

func TestCanChainTweens(t *testing.T) {
	setUp(t)
	tween := NewTween(TweenOptions[float64]{Begin: 0.30, End: 0.50})
	hasOneLineDescription(t, tween)
	chain := tween.Chain(NewTween(TweenOptions[float64]{Begin: 0.50, End: 1.0}))
	controller := newController()
	expect(t, chain.Evaluate(controller), 0.40)
	hasOneLineDescription(t, chain)
}

type stringer string

func (s stringer) String() string { return string(s) }

func TestCanAnimateTweens(t *testing.T) {
	setUp(t)
	tween := NewTween(TweenOptions[float64]{Begin: 0.30, End: 0.50})
	controller := newController()
	animation := tween.Animate(controller)
	controller.SetValue(0.50)
	expect(t, animation.Value(), 0.40)
	hasOneLineDescription(t, animation)
	hasOneLineDescription(t, stringer(animation.ToStringDetails()))
}

func TestCanDriveTweens(t *testing.T) {
	setUp(t)
	tween := NewTween(TweenOptions[float64]{Begin: 0.30, End: 0.50})
	controller := newController()
	animation := Drive[float64](controller, tween)
	controller.SetValue(0.50)
	expect(t, animation.Value(), 0.40)
	hasOneLineDescription(t, animation)
	hasOneLineDescription(t, stringer(animation.ToStringDetails()))
}

func TestSizeTween(t *testing.T) {
	zero, end := ui.SizeZero, ui.NewSize(20.0, 30.0)
	tween := NewSizeTween(SizeTweenOptions{Begin: &zero, End: &end})
	expect(t, *tween.Lerp(0.5), ui.NewSize(10.0, 15.0))
	hasOneLineDescription(t, tween)
}

func TestIntTween(t *testing.T) {
	tween := NewIntTween(IntTweenOptions{Begin: 5, End: 9})
	expect(t, tween.Lerp(0.5), 7)
	expect(t, tween.Lerp(0.7), 8)
}

func TestRectTween(t *testing.T) {
	a := ui.NewRectFromLTWH(5.0, 3.0, 7.0, 11.0)
	b := ui.NewRectFromLTWH(8.0, 12.0, 14.0, 18.0)
	tween := NewRectTween(RectTweenOptions{Begin: &a, End: &b})
	expect(t, *tween.Lerp(0.5), *ui.RectLerp(&a, &b, 0.5))
	hasOneLineDescription(t, tween)
}

func TestConstantTween(t *testing.T) {
	tween := NewConstantTween(100.0)
	expect(t, tween.Begin, 100.0)
	expect(t, tween.End, 100.0)
	expect(t, tween.Lerp(0.0), 100.0)
	expect(t, tween.Lerp(0.5), 100.0)
	expect(t, tween.Lerp(1.0), 100.0)
}

func TestReverseTween(t *testing.T) {
	tween := NewReverseTween(&NewIntTween(IntTweenOptions{Begin: 5, End: 9}).Tween)
	expect(t, tween.Lerp(0.5), 7)
	expect(t, tween.Lerp(0.7), 6)
}

// isSameColorAs is flutter_test's matcher: same color space, components
// within colorEpsilon (0.004).
func isSameColorAs(t *testing.T, got *ui.Color, want ui.Color) {
	t.Helper()
	const eps = 0.004
	abs := func(v float64) float64 {
		if v < 0 {
			return -v
		}
		return v
	}
	if got == nil || got.ColorSpace != want.ColorSpace || abs(got.A-want.A) > eps || abs(got.R-want.R) > eps ||
		abs(got.G-want.G) > eps || abs(got.B-want.B) > eps {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestColorTween(t *testing.T) {
	b, e := ui.NewColor(0xff000000), ui.NewColor(0xffffffff)
	tween := NewColorTween(ColorTweenOptions{Begin: &b, End: &e})
	isSameColorAs(t, tween.Lerp(0.0), ui.NewColor(0xff000000))
	isSameColorAs(t, tween.Lerp(0.5), ui.NewColor(0xff7f7f7f))
	isSameColorAs(t, tween.Lerp(0.7), ui.NewColor(0xffb2b2b2))
	isSameColorAs(t, tween.Lerp(1.0), ui.NewColor(0xffffffff))
}

func TestStepTween(t *testing.T) {
	tween := NewStepTween(StepTweenOptions{Begin: 5, End: 9})
	expect(t, tween.Lerp(0.5), 7)
	expect(t, tween.Lerp(0.7), 7)
}

func TestCurveTween(t *testing.T) {
	tween := NewCurveTween(Curves.EaseIn)
	expect(t, tween.Transform(0.0), 0.0)
	expect(t, tween.Transform(0.5), 0.31640625)
	expect(t, tween.Transform(1.0), 1.0)
}
