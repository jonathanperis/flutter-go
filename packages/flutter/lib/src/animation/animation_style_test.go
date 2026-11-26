package animation

// Port of packages/flutter/test/animation/animation_style_test.dart
// (Flutter 3.50.1): 8/8 tests. hashCode has no Go counterpart (Equal is
// Dart's ==), so the first test checks equality only.

import (
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

func TestAnimationStyleCopyWithEqualityBasics(t *testing.T) {
	expect(t, NewAnimationStyle().Equal(NewAnimationStyle().CopyWith()), true)
	expect(t, NewAnimationStyle().Equal(NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Ease})), false)
}

func TestAnimationStyleCopyWithOverridesAllProperties(t *testing.T) {
	original := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Ease, Duration: dur(time.Second),
		ReverseCurve: Curves.Ease, ReverseDuration: dur(time.Second)})
	copy := original.CopyWith(CopyWithOptions{Curve: Curves.Linear, Duration: dur(2 * time.Second),
		ReverseCurve: Curves.Linear, ReverseDuration: dur(2 * time.Second)})
	expect(t, copy.Curve, Curves.Linear)
	expect(t, *copy.Duration, 2*time.Second)
	expect(t, copy.ReverseCurve, Curves.Linear)
	expect(t, *copy.ReverseDuration, 2*time.Second)
}

func TestAnimationStyleMergeFillsInNullProperties(t *testing.T) {
	original := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Ease, Duration: dur(time.Second),
		ReverseCurve: Curves.Ease, ReverseDuration: dur(time.Second)})
	other := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Linear, Duration: dur(2 * time.Second)})
	merged := original.Merge(other)
	expect(t, merged.Curve, Curves.Linear)
	expect(t, *merged.Duration, 2*time.Second)
	expect(t, merged.ReverseCurve, Curves.Ease)
	expect(t, *merged.ReverseDuration, time.Second)
	expect(t, original.Merge(nil), original)
}

func TestAnimationStyleLerpIdentical(t *testing.T) {
	if AnimationStyleLerp(nil, nil, 0) != nil {
		t.Fatal("lerp(null, null) is not null")
	}
	data := NewAnimationStyle()
	expect(t, AnimationStyleLerp(data, data, 0.5), data)
}

func TestAnimationStyleLerpSmoothlyTransitionsAllValues(t *testing.T) {
	a := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Ease, Duration: dur(time.Second),
		ReverseCurve: Curves.Ease, ReverseDuration: dur(time.Second)})
	b := NewAnimationStyle(AnimationStyleOptions{Curve: Curves.Linear, Duration: dur(2 * time.Second),
		ReverseCurve: Curves.Linear, ReverseDuration: dur(2 * time.Second)})
	aForward, aReverse := float64(a.Duration.Microseconds()), float64(a.ReverseDuration.Microseconds())
	bForward, bReverse := float64(b.Duration.Microseconds()), float64(b.ReverseDuration.Microseconds())

	expect(t, AnimationStyleLerp(a, b, 0).Equal(a), true)
	expect(t, AnimationStyleLerp(a, b, 1.0).Equal(b), true)

	const styleSteps, curveSteps = 5, 5
	for styleStep := 0; styleStep < styleSteps; styleStep++ {
		styleTransition := float64(styleStep+1) / (styleSteps + 1)
		lerpedStyle := AnimationStyleLerp(a, b, styleTransition)
		expect(t, int(lerpedStyle.Duration.Microseconds()), dartRound(*ui.LerpDouble(&aForward, &bForward, styleTransition)))
		expect(t, int(lerpedStyle.ReverseDuration.Microseconds()), dartRound(*ui.LerpDouble(&aReverse, &bReverse, styleTransition)))
		for curveStep := 0; curveStep < curveSteps; curveStep++ {
			tt := float64(curveStep+1) / (curveSteps + 1)
			aResult := Curves.Ease.Transform(tt)
			bResult := Curves.Linear.Transform(tt)
			lerpResult := lerpedStyle.Curve.Transform(tt)
			expect(t, lerpResult, *ui.LerpDouble(&aResult, &bResult, styleTransition))
		}
	}
}

func TestAnimationStyleLerpWithNullProperty(t *testing.T) {
	a := NewAnimationStyle(AnimationStyleOptions{Duration: dur(2 * time.Second)})
	b := NewAnimationStyle()
	// Assuming null is treated as Duration.zero for interpolation.
	expect(t, *AnimationStyleLerp(a, b, 1.0/2).Duration, time.Second)
	expect(t, *AnimationStyleLerp(a, b, 1.0/4).Duration, 1500*ms)
}

func describedProperties(s *AnimationStyle) []string {
	builder := &foundation.DiagnosticPropertiesBuilder{}
	s.DebugFillProperties(builder)
	var description []string
	for _, node := range builder.Properties {
		if !node.IsFiltered(foundation.DiagnosticLevelInfo) {
			description = append(description, node.ToString())
		}
	}
	return description
}

func TestDefaultAnimationStyleDebugFillProperties(t *testing.T) {
	expectStrings(t, describedProperties(NewAnimationStyle()))
}

func TestAnimationStyleImplementsDebugFillProperties(t *testing.T) {
	expectStrings(t, describedProperties(NewAnimationStyle(AnimationStyleOptions{Curve: Curves.EaseInOut,
		Duration: dur(time.Second), ReverseCurve: Curves.BounceInOut, ReverseDuration: dur(2 * time.Second)})),
		"curve: Cubic(0.42, 0.00, 0.58, 1.00)",
		"duration: 0:00:01.000000",
		"reverseCurve: _BounceInOutCurve",
		"reverseDuration: 0:00:02.000000")
}
