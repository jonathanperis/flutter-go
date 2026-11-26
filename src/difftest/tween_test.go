package difftest

// Go twin of flutter/test/tween_test.dart: the same tweens, sequences,
// styles and messages, logged with Dart's number formatting, compared bit
// for bit.

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/animation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

var twLines []string

func twLog(s string) { twLines = append(twLines, s) }

func col(c *ui.Color) string {
	if c == nil {
		return "null"
	}
	return ds(c.A) + " " + ds(c.R) + " " + ds(c.G) + " " + ds(c.B) + " " + strings.TrimPrefix(c.ColorSpace.String(), "ColorSpace.")
}

func sz(s *ui.Size) string {
	if s == nil {
		return "null"
	}
	return ds(s.Width) + " " + ds(s.Height)
}

func rc(r *ui.Rect) string {
	if r == nil {
		return "null"
	}
	return ds(r.Left) + " " + ds(r.Top) + " " + ds(r.Right) + " " + ds(r.Bottom)
}

func offs(o ui.Offset) string { return ds(o.Dx) + " " + ds(o.Dy) }

func tweenTs() []float64 {
	ts := []float64{0.0, 1.0, 0.5, 0.25, 0.75, 0.1, 0.2, 0.3, 0.7, 0.9, 1.0 / 3, 2.0 / 3, 0.123456789, 0.999999, 1e-9,
		0.4999999999, -0.5, 1.5, -2.0, 3.25}
	for i := 1; i < 40; i++ {
		ts = append(ts, float64(i)/41)
	}
	return ts
}

func unitTs(ts []float64) []float64 {
	var out []float64
	for _, t := range ts {
		if t >= 0 && t <= 1 {
			out = append(out, t)
		}
	}
	return out
}

func errorOf(f func()) (s string) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if e, ok := r.(*foundation.FlutterError); ok {
			var parts []string
			for _, n := range e.Diagnostics {
				parts = append(parts, n.ToString())
			}
			s = strings.Join(parts, " | ")
			return
		}
		panic(r)
	}()
	f()
	return "no error"
}

func cp(v uint32) *ui.Color     { c := ui.NewColor(int(v)); return &c }
func szp(w, h float64) *ui.Size { s := ui.NewSize(w, h); return &s }
func rcp(r ui.Rect) *ui.Rect    { return &r }

func durStr(d *time.Duration) string {
	if d == nil {
		return "null"
	}
	return strconv.FormatInt(d.Microseconds(), 10)
}

func curveStr(c Curve) string {
	if c == nil {
		return "null"
	}
	return c.String()
}

func curveAt(c Curve, u float64) string {
	if c == nil {
		return "null"
	}
	return ds(c.Transform(u))
}

func runTweenScene() {
	ts := tweenTs()
	unit := unitTs(ts)
	doubles := []*Tween[float64]{
		NewTween(TweenOptions[float64]{Begin: 0.3, End: 0.5}),
		NewTween(TweenOptions[float64]{Begin: -100, End: 250.75}),
		NewTween(TweenOptions[float64]{Begin: 1e10, End: -3e-7}),
		NewTween(TweenOptions[float64]{Begin: 0.1, End: 0.1}),
		NewTween(TweenOptions[float64]{Begin: 5, End: inf()}),
	}
	for _, tw := range doubles {
		twLog(tw.String())
		for _, t := range ts {
			twLog("  " + ds(t) + " " + ds(tw.Transform(t)) + " " + ds(tw.Lerp(t)))
		}
	}
	offsets := NewTween(TweenOptions[ui.Offset]{Begin: ui.NewOffset(1.5, -2), End: ui.NewOffset(-30.25, 7.125)})
	twLog(offsets.String())
	for _, t := range ts {
		twLog("  " + ds(t) + " " + offs(offsets.Transform(t)))
	}
	sizes := NewTween(TweenOptions[ui.Size]{Begin: ui.NewSize(10, 20), End: ui.NewSize(3.5, 99.25)})
	twLog(sizes.String())
	for _, t := range ts {
		s := sizes.Transform(t)
		twLog("  " + ds(t) + " " + sz(&s))
	}
	p3 := ui.NewColorFrom(ui.ColorFromOptions{Alpha: 1, Red: 0.1, Green: 0.2, Blue: 0.3, ColorSpace: ui.ColorSpaceDisplayP3})
	colors := []*ColorTween{
		NewColorTween(ColorTweenOptions{Begin: cp(0xff000000), End: cp(0xffffffff)}),
		NewColorTween(ColorTweenOptions{Begin: cp(0x802196f3), End: cp(0xff4caf50)}),
		NewColorTween(ColorTweenOptions{Begin: cp(0xff2196f3)}),
		NewColorTween(ColorTweenOptions{End: cp(0x40ff0000)}),
		NewColorTween(),
		NewColorTween(ColorTweenOptions{Begin: &p3, End: cp(0xffff8800)}),
	}
	for _, tw := range colors {
		twLog(tw.String())
		for _, t := range unit {
			twLog("  " + ds(t) + " " + col(tw.Transform(t)))
		}
	}
	sizeTweens := []*SizeTween{
		NewSizeTween(SizeTweenOptions{Begin: szp(0, 0), End: szp(20, 30)}),
		NewSizeTween(SizeTweenOptions{Begin: szp(1.25, 7), End: szp(-3, 1e5)}),
		NewSizeTween(SizeTweenOptions{Begin: szp(4, 8)}),
		NewSizeTween(SizeTweenOptions{End: szp(4, 8)}),
	}
	for _, tw := range sizeTweens {
		twLog(tw.String())
		for _, t := range ts {
			twLog("  " + ds(t) + " " + sz(tw.Transform(t)))
		}
	}
	rectTweens := []*RectTween{
		NewRectTween(RectTweenOptions{Begin: rcp(ui.NewRectFromLTWH(5, 3, 7, 11)), End: rcp(ui.NewRectFromLTWH(8, 12, 14, 18))}),
		NewRectTween(RectTweenOptions{Begin: rcp(ui.NewRectFromLTRB(-1.5, 0, 2.25, 100)), End: rcp(ui.RectZero)}),
		NewRectTween(RectTweenOptions{Begin: rcp(ui.NewRectFromLTWH(10, 10, 10, 10))}),
		NewRectTween(RectTweenOptions{End: rcp(ui.NewRectFromLTWH(10, 10, 10, 10))}),
	}
	for _, tw := range rectTweens {
		twLog(tw.String())
		for _, t := range ts {
			twLog("  " + ds(t) + " " + rc(tw.Transform(t)))
		}
	}
	ints := []Animatable[int]{
		NewIntTween(IntTweenOptions{Begin: 5, End: 9}),
		NewIntTween(IntTweenOptions{Begin: -7, End: 12}),
		NewIntTween(IntTweenOptions{Begin: 100, End: -100}),
		NewStepTween(StepTweenOptions{Begin: 5, End: 9}),
		NewStepTween(StepTweenOptions{Begin: -7, End: 12}),
		NewStepTween(StepTweenOptions{Begin: 100, End: -100}),
		NewReverseTween(&NewIntTween(IntTweenOptions{Begin: 5, End: 9}).Tween),
		NewReverseTween(&NewStepTween(StepTweenOptions{Begin: -3, End: 4}).Tween),
		NewConstantTween(42),
	}
	for _, tw := range ints {
		twLog(tw.String())
		for _, t := range ts {
			twLog("  " + ds(t) + " " + strconv.Itoa(tw.Transform(t)))
		}
	}
	reverseDouble := NewReverseTween(NewTween(TweenOptions[float64]{Begin: 0.25, End: 4}))
	twLog(reverseDouble.String() + " " + ds(reverseDouble.Begin) + " " + ds(reverseDouble.End))
	for _, t := range ts {
		twLog("  " + ds(t) + " " + ds(reverseDouble.Transform(t)))
	}
	constant := NewConstantTween(100.0)
	twLog(constant.String() + " " + ds(constant.Transform(0.3)))
	curves := []*CurveTween{
		NewCurveTween(Curves.EaseIn),
		NewCurveTween(Curves.ElasticInOut),
		NewCurveTween(NewInterval(0.25, 0.75, IntervalOptions{Curve: Curves.BounceOut})),
		NewCurveTween(NewSawTooth(4)),
	}
	for _, tw := range curves {
		twLog(tw.String())
		for _, t := range unit {
			twLog("  " + ds(t) + " " + ds(tw.Transform(t)))
		}
	}
	chain := NewTween(TweenOptions[float64]{Begin: 10, End: 20}).Chain(NewCurveTween(Curves.EaseOutCubic))
	chain2 := NewTween(TweenOptions[ui.Offset]{Begin: ui.OffsetZero, End: ui.NewOffset(5, 9)}).
		Chain(NewTween(TweenOptions[float64]{Begin: 0.2, End: 0.6}).Chain(NewCurveTween(Curves.FastOutSlowIn)))
	twLog(chain.String())
	twLog(chain2.String())
	for _, t := range unit {
		twLog("  " + ds(t) + " " + ds(chain.Transform(t)) + " " + offs(chain2.Transform(t)))
	}
	callback := NewAnimatableFromCallback(func(v float64) ui.Offset { return ui.NewOffset(float64(v*3), -v) })
	twLog("callback " + offs(callback.Transform(0.25)) + " " + foundation.ObjectRuntimeType(callback, ""))
	twLog("callback " + callback.String())

	controller := NewAnimationController(vs, AnimationControllerOptions{Value: pf(0.37)})
	animated := NewTween(TweenOptions[float64]{Begin: 0.3, End: 0.5}).Animate(controller)
	driven := Drive(controller, NewSizeTween(SizeTweenOptions{Begin: szp(1, 2), End: szp(3, 5)}).Chain(NewCurveTween(Curves.Ease)))
	twLog("animated " + mask(animated) + " | " + animated.ToStringDetails() + " | " + ds(animated.Value()) + " " + animated.Status().Name())
	twLog("driven " + sz(driven.Value()) + " " + driven.Status().Name())
	for _, v := range unit {
		controller.SetValue(v)
		twLog("  " + ds(v) + " " + ds(animated.Value()) + " " + sz(driven.Value()) + " " + ds(chain.Evaluate(controller)))
	}
	controller.Dispose()

	seqs := []Animatable[float64]{
		NewTweenSequence([]TweenSequenceItem[float64]{
			NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 5.0, End: 10.0}), 4.0),
			NewTweenSequenceItem[float64](NewConstantTween(10.0), 2.0),
			NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 10.0, End: 5.0}), 4.0),
		}),
		NewTweenSequence([]TweenSequenceItem[float64]{
			NewTweenSequenceItem(NewTween(TweenOptions[float64]{Begin: 0, End: 1}).Chain(NewCurveTween(Curves.EaseIn)), 0.7),
			NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 1, End: -1}), 1e-3),
			NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: -1, End: 0.5}), 3),
			NewTweenSequenceItem[float64](NewCurveTween(Curves.BounceIn), 1.0/3),
		}),
		NewFlippedTweenSequence([]TweenSequenceItem[float64]{
			NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 0, End: 0.3}), 1),
			NewTweenSequenceItem(NewTween(TweenOptions[float64]{Begin: 0.3, End: 1}).Chain(NewCurveTween(Curves.Decelerate)), 2.5),
		}),
	}
	for _, s := range seqs {
		twLog(s.String())
		for _, t := range unit {
			twLog("  " + ds(t) + " " + ds(s.Transform(t)))
		}
	}
	colorSeq := NewTweenSequence([]TweenSequenceItem[*ui.Color]{
		NewTweenSequenceItem[*ui.Color](NewColorTween(ColorTweenOptions{Begin: cp(0xffff0000), End: cp(0xff00ff00)}), 1),
		NewTweenSequenceItem[*ui.Color](NewColorTween(ColorTweenOptions{Begin: cp(0xff00ff00), End: cp(0x000000ff)}), 2),
	})
	for _, t := range unit {
		twLog("  color " + ds(t) + " " + col(colorSeq.Transform(t)))
	}

	a := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Ease, Duration: pd(time.Second),
		ReverseCurve: Curves.BounceIn, ReverseDuration: pd(333 * time.Millisecond)})
	b := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Linear, Duration: pd(2000001 * time.Microsecond)})
	for _, t := range []float64{0, 0.1, 0.25, 1.0 / 3, 0.5, 0.77, 1, 1.5} {
		s := AnimationStyleLerp(a, b, t)
		twLog("style " + ds(t) + " " + durStr(s.Duration) + " " + durStr(s.ReverseDuration) + " " + curveStr(s.Curve) + " " + curveStr(s.ReverseCurve))
		for _, u := range []float64{0.1, 0.5, 0.9} {
			twLog("  " + ds(u) + " " + curveAt(s.Curve, u) + " " + curveAt(s.ReverseCurve, u))
		}
	}
	twLog("style null " + durStr(AnimationStyleLerp(nil, b, 0.5).Duration) + " " + curveStr(AnimationStyleLerp(a, nil, 0.25).Curve))
	c1 := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Ease, Duration: pd(time.Second)})
	c2 := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Ease, Duration: pd(time.Second)})
	same := AnimationStyleLerp(c1, c2, 0.4)
	twLog("style same " + curveStr(same.Curve) + " " + durStr(same.Duration) + " " + curveStr(same.ReverseCurve) + " " + durStr(same.ReverseDuration))
	other := AnimationStyleLerp(AnimationStyleLerp(a, b, 0.5), AnimationStyleLerp(NewAnimationStyle(AnimationStyleOptions{Curve: Curves.EaseIn}), b, 0.5), 0.3)
	twLog("style other first " + curveStr(other.Curve))
	for _, p := range [][2]float64{{0.5, 0.5}, {0.5, 0.25}} {
		l := AnimationStyleLerp(AnimationStyleLerp(a, b, p[0]), AnimationStyleLerp(a, b, p[1]), 0.3)
		twLog("style lerped " + ds(p[0]) + " " + ds(p[1]) + " " + curveStr(l.Curve) + " " + curveStr(l.ReverseCurve) + " " +
			strconv.FormatBool(AnimationStyleLerp(a, b, p[0]).Equal(AnimationStyleLerp(a, b, p[1]))))
	}

	twLog(errorOf(func() {
		NewTween(TweenOptions[ui.Color]{Begin: ui.NewColor(0xff000000), End: ui.NewColor(0xffffffff)}).Transform(0.1)
	}))
	twLog(errorOf(func() {
		NewTween(TweenOptions[ui.Rect]{Begin: ui.NewRectFromLTWH(0, 0, 10, 10), End: ui.NewRectFromLTWH(2, 2, 2, 2)}).Transform(0.1)
	}))
	twLog(errorOf(func() { NewTween(TweenOptions[int]{Begin: 0, End: 1}).Transform(0.1) }))
	twLog(errorOf(func() { NewTween(TweenOptions[int]{Begin: 0, End: 1}).Transform(1.0) }))
}

func inf() float64 { z := 0.0; return 1 / z }

func TestTweenMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/tween.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	flutter_test.New(t) // the binding the controller's ticker needs
	twLines = nil
	runTweenScene()
	compareLines(t, ref.Log, twLines)
}
