package animation

// flutter-go's own: the Dart class hierarchy, checked at compile time
// (every animation is an Animation, every tween an Animatable, the
// controller is a Listenable/ValueListenable as in Dart), and the typedefs'
// signatures.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

var (
	_ Animation[float64]                  = (*AnimationController)(nil)
	_ foundation.ValueListenable[float64] = (*AnimationController)(nil)
	_ Animation[float64]                  = (*ProxyAnimation)(nil)
	_ Animation[float64]                  = (*ReverseAnimation)(nil)
	_ Animation[float64]                  = (*CurvedAnimation)(nil)
	_ Animation[float64]                  = (*TrainHoppingAnimation)(nil)
	_ Animation[float64]                  = (*AnimationMean)(nil)
	_ Animation[int]                      = (*AnimationMax[int])(nil)
	_ Animation[float64]                  = (*AnimationMin[float64])(nil)
	_ Animation[ui.Offset]                = (*AlwaysStoppedAnimation[ui.Offset])(nil)
	_ ParentSelf[float64]                 = (*CurvedAnimation)(nil)
	_ LazyListenerSelf                    = (*ProxyAnimation)(nil)
	_ LazyListenerSelf                    = (*CompoundAnimation[float64])(nil)
	_ ListenerRegistrationSelf            = (*AnimationEagerListenerMixin)(nil)
	_ ListenerRegistrationSelf            = (*AnimationLazyListenerMixin)(nil)

	_ Animatable[float64]   = (*Tween[float64])(nil)
	_ TweenSelf[*ui.Color]  = (*ColorTween)(nil)
	_ TweenSelf[*ui.Size]   = (*SizeTween)(nil)
	_ TweenSelf[*ui.Rect]   = (*RectTween)(nil)
	_ TweenSelf[int]        = (*IntTween)(nil)
	_ TweenSelf[int]        = (*StepTween)(nil)
	_ TweenSelf[int]        = (*ReverseTween[int])(nil)
	_ TweenSelf[float64]    = (*ConstantTween[float64])(nil)
	_ Animatable[float64]   = (*CurveTween)(nil)
	_ Animatable[ui.Offset] = (*TweenSequence[ui.Offset])(nil)
	_ Animatable[float64]   = (*FlippedTweenSequence)(nil)

	_ foundation.Diagnosticable = (*AnimationStyle)(nil)
)

// The embedded mixins and bases expose the members Dart's mixins do.
var (
	_ = (*AnimationWithParentMixin[float64]).Status
	_ = (*AnimationEagerListenerMixin).Dispose
	_ = (*CompoundAnimation[float64]).Construct
	_ = (*AnimatableBase[float64]).Chain
	_ = (*AnimationBase[float64]).ToStringDetails
)

func TestTypedefSignatures(t *testing.T) {
	var status AnimationStatusListener = func(AnimationStatus) {}
	var transformer ValueListenableTransformer[float64] = func(v float64) float64 { return v * 2 }
	var callback AnimatableCallback[ui.Offset] = func(v float64) ui.Offset { return ui.NewOffset(v, v) }
	status(AnimationStatusForward)
	expect(t, transformer(2), 4.0)
	expect(t, NewAnimatableFromCallback(callback).Transform(0.5), ui.NewOffset(0.5, 0.5))
	expect(t, NewAlwaysStoppedAnimation(ui.NewOffset(1, 2)).Value(), ui.NewOffset(1, 2))
	expect(t, NewReverseTween(NewTween(TweenOptions[float64]{Begin: 1, End: 3})).Transform(0.25), 2.5)
	expect(t, NewFlippedTweenSequence([]TweenSequenceItem[float64]{
		NewTweenSequenceItem[float64](NewTween(TweenOptions[float64]{Begin: 0, End: 1}), 1),
	}).Transform(0.25), 0.25)
	seq := NewTweenSequence([]TweenSequenceItem[ui.Offset]{
		NewTweenSequenceItem(NewAnimatableFromCallback(func(v float64) ui.Offset { return ui.NewOffset(v, 0) }), 1),
		NewTweenSequenceItem(NewAnimatableFromCallback(func(v float64) ui.Offset { return ui.NewOffset(1, v) }), 1),
	})
	expect(t, seq.Transform(0.75), ui.NewOffset(1, 0.5))
	throwsAssertionError(t, func() { NewTweenSequenceItem[float64](NewTween[float64](), 0) })
	throwsAssertionError(t, func() { NewTweenSequence[float64](nil) })
}
