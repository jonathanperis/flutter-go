package animation

// Port of packages/flutter/lib/src/animation/tween.dart.
//
// Animatable is Dart's abstract class: an interface plus the embeddable
// AnimatableBase (evaluate/animate/chain call Transform through self).
// Tween is a concrete class Flutter extends (ColorTween, IntTween...): a
// struct whose Transform calls the Lerp override through self, so a type
// embedding Tween calls SetSelf once, as with Ticker.
//
// Tween's begin and end are Dart's `T?`: the fields are T, so a nullable
// tween (ColorTween is a Tween<Color?>) has a pointer type argument
// (Tween[*ui.Color]), and nil is null. With a non-pointer T (Tween<double>)
// there is no null begin/end.
//
// Tween.lerp evaluates Dart's dynamic `begin + (end - begin) * t`: on
// float64 directly, and on other types through their Add, Sub and Mul
// methods (API_MAPPING's operator methods: ui.Offset, ui.Size...), found
// by reflection as Dart finds the operators at run time. A type without
// them is Dart's NoSuchMethodError, a result of another type Dart's
// TypeError, reported as Flutter reports them (FlutterError with hints).

import (
	"errors"
	"reflect"
	"strconv"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AnimatableCallback is the signature of a function that receives a value
// in [0, 1] and returns a value of type T (see NewAnimatableFromCallback).
type AnimatableCallback[T any] func(value float64) T

// Animatable is an object that can produce a value of type T given an
// Animation[float64] as input. Implementations embed AnimatableBase and
// define Transform.
type Animatable[T any] interface {
	// Transform returns the value of the object at point t (usually in
	// [0, 1]).
	Transform(t float64) T
	// Evaluate is Transform(animation.Value()).
	Evaluate(animation Animation[float64]) T
	// Animate returns an Animation driven by parent, with this object's
	// value.
	Animate(parent Animation[float64]) Animation[T]
	// Chain returns an Animatable that first applies parent, then this
	// object.
	Chain(parent Animatable[float64]) Animatable[T]
	String() string
}

// AnimatableBase is the embeddable implementation of Animatable's concrete
// members. The embedding type defines Transform and calls SetSelf once.
type AnimatableBase[T any] struct{ self Animatable[T] }

// SetSelf points the base at the animatable that embeds it.
func (a *AnimatableBase[T]) SetSelf(self Animatable[T]) { a.self = self }

func (a *AnimatableBase[T]) outer() Animatable[T] {
	if a.self == nil {
		panic(unbound("AnimatableBase"))
	}
	return a.self
}

// Transform is abstract in Dart; an embedding type that doesn't define it
// gets UnimplementedError.
func (a *AnimatableBase[T]) Transform(float64) T { panic(dart.UnimplementedError{}) }

// Evaluate is Transform(animation.Value()).
func (a *AnimatableBase[T]) Evaluate(animation Animation[float64]) T {
	return a.outer().Transform(animation.Value())
}

// Animate returns an Animation driven by parent, with this object's value.
func (a *AnimatableBase[T]) Animate(parent Animation[float64]) Animation[T] {
	e := &_AnimatedEvaluation[T]{parent: parent, evaluatable: a.outer()}
	e.SetSelf(e)
	return e
}

// Chain returns an Animatable that first applies parent, then this object.
func (a *AnimatableBase[T]) Chain(parent Animatable[float64]) Animatable[T] {
	c := &_ChainedEvaluation[T]{parent: parent, evaluatable: a.outer()}
	c.SetSelf(c)
	return c
}

// String is Object.toString ("Instance of 'Type'").
func (a *AnimatableBase[T]) String() string {
	return "Instance of '" + runtimeType(a.outer()) + "'"
}

// NewAnimatableFromCallback is Animatable.fromCallback: an Animatable whose
// Transform is callback.
func NewAnimatableFromCallback[T any](callback AnimatableCallback[T]) Animatable[T] {
	a := &_CallbackAnimatable[T]{callback: callback}
	a.SetSelf(a)
	return a
}

// _CallbackAnimatable is Animatable.fromCallback's type.
type _CallbackAnimatable[T any] struct {
	AnimatableBase[T]
	callback AnimatableCallback[T]
}

func (a *_CallbackAnimatable[T]) Transform(t float64) T { return a.callback(t) }

// _AnimatedEvaluation is Animatable.animate's animation.
type _AnimatedEvaluation[T any] struct {
	AnimationBase[T]
	AnimationWithParentMixin[float64]
	parent      Animation[float64]
	evaluatable Animatable[T]
}

func (e *_AnimatedEvaluation[T]) SetSelf(self *_AnimatedEvaluation[T]) {
	e.AnimationBase.SetSelf(self)
	e.AnimationWithParentMixin.SetSelf(self)
}

func (e *_AnimatedEvaluation[T]) Parent() Animation[float64] { return e.parent }

func (e *_AnimatedEvaluation[T]) Value() T { return e.evaluatable.Evaluate(e.parent) }

func (e *_AnimatedEvaluation[T]) String() string {
	return e.parent.String() + "➩" + e.evaluatable.String() + "➩" + dartString(e.Value())
}

func (e *_AnimatedEvaluation[T]) ToStringDetails() string {
	return e.AnimationBase.ToStringDetails() + " " + e.evaluatable.String()
}

// _ChainedEvaluation is Animatable.chain's animatable.
type _ChainedEvaluation[T any] struct {
	AnimatableBase[T]
	parent      Animatable[float64]
	evaluatable Animatable[T]
}

func (c *_ChainedEvaluation[T]) Transform(t float64) T {
	return c.evaluatable.Transform(c.parent.Transform(t))
}

func (c *_ChainedEvaluation[T]) String() string {
	return c.parent.String() + "➩" + c.evaluatable.String()
}

// TweenSelf is the set of Tween members a type embedding Tween may
// override; Tween calls them through self.
type TweenSelf[T any] interface {
	Animatable[T]
	// Lerp returns the value this variable has at the given animation
	// clock value (Dart's @protected lerp).
	Lerp(t float64) T
}

// Tween is a linear interpolation between a beginning and ending value:
// Transform(t) is Begin at 0, End at 1 and Lerp(t) in between.
type Tween[T any] struct {
	AnimatableBase[T]
	// Begin is the value this variable has at the beginning of the
	// animation.
	Begin T
	// End is the value this variable has at the end of the animation.
	End T

	self TweenSelf[T]
}

// TweenOptions are Tween's named parameters.
type TweenOptions[T any] struct {
	Begin, End T
}

// NewTween creates a tween from Begin to End.
func NewTween[T any](opts ...TweenOptions[T]) *Tween[T] {
	tw := &Tween[T]{}
	if len(opts) > 0 {
		tw.Begin, tw.End = opts[0].Begin, opts[0].End
	}
	tw.SetSelf(tw)
	return tw
}

// SetSelf points the tween at the object embedding it, whose Lerp (and
// String...) override the tween's.
func (tw *Tween[T]) SetSelf(self TweenSelf[T]) {
	tw.self = self
	tw.AnimatableBase.SetSelf(self)
}

func (tw *Tween[T]) tweenOuter() TweenSelf[T] {
	if tw.self == nil {
		panic(unbound("Tween"))
	}
	return tw.self
}

// Lerp is `begin + (end - begin) * t` (Dart's dynamic operators; see the
// file comment). Begin and End must not be null.
func (tw *Tween[T]) Lerp(t float64) T {
	assert(!isNull(tw.Begin), nil)
	assert(!isNull(tw.End), nil)
	result, failure := dynamicLerp[T](tw.Begin, tw.End, t)
	if failure != lerpOK {
		if foundation.KDebugMode {
			panic(lerpError(tw.Begin, tw.End, result, failure))
		}
		panic(lerpRawError(tw.Begin, tw.End, result, failure))
	}
	return result.(T)
}

// Transform returns Begin at 0, End at 1, and Lerp(t) otherwise.
func (tw *Tween[T]) Transform(t float64) T {
	if t == 0.0 {
		return tw.Begin
	}
	if t == 1.0 {
		return tw.End
	}
	return tw.tweenOuter().Lerp(t)
}

func (tw *Tween[T]) String() string {
	return foundation.ObjectRuntimeType(tw.tweenOuter(), "Animatable") + "(" + dartString(tw.Begin) + " → " +
		dartString(tw.End) + ")"
}

type lerpFailure int

const (
	lerpOK lerpFailure = iota
	lerpNoSuchMethod
	lerpTypeError
)

// dynamicLerp evaluates `begin + (end - begin) * t` as Dart does with
// dynamic receivers, then `result as T`. result is the sum (nil when an
// operator call failed).
func dynamicLerp[T any](begin, end any, t float64) (result any, failure lerpFailure) {
	switch b := begin.(type) {
	case float64:
		result = b + float64((end.(float64)-b)*t)
	case int:
		// int - int is an int, times a double a double, plus an int a
		// double: `as int` fails.
		result = float64(b) + float64(float64(end.(int)-b)*t)
	default:
		failure = func() (failure lerpFailure) {
			defer func() {
				if r := recover(); r != nil {
					p, ok := r.(lerpPanic)
					if !ok {
						panic(r)
					}
					failure = p.failure
				}
			}()
			diff := callOperator(end, "Sub", begin)
			product := callOperator(diff, "Mul", t)
			result = callOperator(begin, "Add", product)
			return lerpOK
		}()
		if failure != lerpOK {
			return nil, failure
		}
	}
	if _, ok := result.(T); !ok {
		return result, lerpTypeError
	}
	return result, lerpOK
}

type lerpPanic struct{ failure lerpFailure }

// callOperator calls receiver's operator method name with arg, as a
// dynamic Dart call: a missing method is NoSuchMethodError, an argument of
// the wrong type a TypeError.
func callOperator(receiver any, name string, arg any) any {
	rv := reflect.ValueOf(receiver)
	if !rv.IsValid() {
		panic(lerpPanic{lerpNoSuchMethod})
	}
	m := rv.MethodByName(name)
	if !m.IsValid() || m.Type().NumIn() != 1 || m.Type().NumOut() != 1 {
		panic(lerpPanic{lerpNoSuchMethod})
	}
	av := reflect.ValueOf(arg)
	if !av.IsValid() || !av.Type().AssignableTo(m.Type().In(0)) {
		panic(lerpPanic{lerpTypeError})
	}
	out := m.Call([]reflect.Value{av})[0]
	if out.Kind() == reflect.Interface {
		out = out.Elem()
	}
	if !out.IsValid() {
		return nil
	}
	return out.Interface()
}

// lerpError is the FlutterError Tween.lerp's debug check throws.
func lerpError(begin, end, result any, failure lerpFailure) *foundation.FlutterError {
	apiDocs := "See \"Types with special considerations\" at https://api.flutter.dev/flutter/animation/Tween-class.html " +
		"for more information."
	parts := []foundation.DiagnosticsNode{
		foundation.NewErrorSummary("Cannot lerp between \"" + dartString(begin) + "\" and \"" + dartString(end) + "\"."),
	}
	beginType := runtimeTypeOf(begin)
	if failure == lerpNoSuchMethod {
		parts = append(parts, foundation.NewErrorDescription(
			"The type "+beginType+" might not fully implement `+`, `-`, and/or `*`. "+apiDocs))
		switch {
		case isColor(begin) || isColor(end):
			parts = append(parts, foundation.NewErrorHint("To lerp colors, consider ColorTween instead."))
		case isRect(begin) || isRect(end):
			parts = append(parts, foundation.NewErrorHint("To lerp rects, consider RectTween instead."))
		default:
			parts = append(parts, foundation.NewErrorHint("There may be a dedicated \""+beginType+
				"Tween\" for this type, or you may need to create one."))
		}
	} else {
		parts = append(parts, foundation.NewErrorDescription("The type "+beginType+" returned a "+
			runtimeTypeOf(result)+" after multiplication with a double value. "+apiDocs))
		if isInt(begin) || isInt(end) {
			parts = append(parts, foundation.NewErrorHint("To lerp int values, consider IntTween or StepTween instead."))
		} else {
			parts = append(parts, foundation.NewErrorHint("There may be a dedicated \""+beginType+
				"Tween\" for this type, or you may need to create one."))
		}
	}
	return foundation.NewFlutterErrorFromParts(parts)
}

// lerpRawError is what Dart throws without the debug check (release
// builds): the NoSuchMethodError or TypeError itself.
func lerpRawError(begin, end, result any, failure lerpFailure) error {
	if failure == lerpNoSuchMethod {
		return errors.New("NoSuchMethodError: Class '" + runtimeTypeOf(begin) +
			"' has no instance method for the tween's operators")
	}
	return errors.New("TypeError: Instance of '" + runtimeTypeOf(result) + "' is not a subtype of the tween's type")
}

func isColor(v any) bool {
	switch v.(type) {
	case ui.Color, *ui.Color:
		return !isNull(v)
	}
	return false
}

func isRect(v any) bool {
	switch v.(type) {
	case ui.Rect, *ui.Rect:
		return !isNull(v)
	}
	return false
}

func isInt(v any) bool {
	switch v.(type) {
	case int, *int:
		return !isNull(v)
	}
	return false
}

// runtimeTypeOf is Dart's `value.runtimeType` as text ("Null" for null).
func runtimeTypeOf(v any) string {
	if isNull(v) {
		return "Null"
	}
	return runtimeType(v)
}

// ReverseTween is a Tween that evaluates its Parent in reverse.
type ReverseTween[T any] struct {
	Tween[T]
	// Parent is the tween this one reverses: the Tween embedded in it
	// (`&intTween.Tween`), whose Lerp override is used.
	Parent *Tween[T]
}

// NewReverseTween creates a tween from parent's End to its Begin.
func NewReverseTween[T any](parent *Tween[T]) *ReverseTween[T] {
	tw := &ReverseTween[T]{Parent: parent}
	tw.Begin, tw.End = parent.End, parent.Begin
	tw.SetSelf(tw)
	return tw
}

func (tw *ReverseTween[T]) Lerp(t float64) T { return tw.Parent.tweenOuter().Lerp(1.0 - t) }

// ColorTween is an interpolation between two colors (ui.ColorLerp); nil
// is Dart's null (a null end fades begin out, and vice versa).
type ColorTween struct{ Tween[*ui.Color] }

// ColorTweenOptions are ColorTween's named parameters.
type ColorTweenOptions struct{ Begin, End *ui.Color }

// NewColorTween creates a color tween.
func NewColorTween(opts ...ColorTweenOptions) *ColorTween {
	tw := &ColorTween{}
	if len(opts) > 0 {
		tw.Begin, tw.End = opts[0].Begin, opts[0].End
	}
	tw.SetSelf(tw)
	return tw
}

func (tw *ColorTween) Lerp(t float64) *ui.Color { return ui.ColorLerp(tw.Begin, tw.End, t) }

// SizeTween is an interpolation between two sizes (ui.SizeLerp).
type SizeTween struct{ Tween[*ui.Size] }

// SizeTweenOptions are SizeTween's named parameters.
type SizeTweenOptions struct{ Begin, End *ui.Size }

// NewSizeTween creates a size tween.
func NewSizeTween(opts ...SizeTweenOptions) *SizeTween {
	tw := &SizeTween{}
	if len(opts) > 0 {
		tw.Begin, tw.End = opts[0].Begin, opts[0].End
	}
	tw.SetSelf(tw)
	return tw
}

func (tw *SizeTween) Lerp(t float64) *ui.Size { return ui.SizeLerp(tw.Begin, tw.End, t) }

// RectTween is an interpolation between two rectangles (ui.RectLerp).
type RectTween struct{ Tween[*ui.Rect] }

// RectTweenOptions are RectTween's named parameters.
type RectTweenOptions struct{ Begin, End *ui.Rect }

// NewRectTween creates a rect tween.
func NewRectTween(opts ...RectTweenOptions) *RectTween {
	tw := &RectTween{}
	if len(opts) > 0 {
		tw.Begin, tw.End = opts[0].Begin, opts[0].End
	}
	tw.SetSelf(tw)
	return tw
}

func (tw *RectTween) Lerp(t float64) *ui.Rect { return ui.RectLerp(tw.Begin, tw.End, t) }

// IntTween is an interpolation between two integers that rounds.
type IntTween struct{ Tween[int] }

// IntTweenOptions are IntTween's named parameters.
type IntTweenOptions struct{ Begin, End int }

// NewIntTween creates an integer tween.
func NewIntTween(opts ...IntTweenOptions) *IntTween {
	tw := &IntTween{}
	if len(opts) > 0 {
		tw.Begin, tw.End = opts[0].Begin, opts[0].End
	}
	tw.SetSelf(tw)
	return tw
}

// Lerp is (begin + (end - begin) * t).round().
func (tw *IntTween) Lerp(t float64) int {
	return dartRound(float64(tw.Begin) + float64(float64(tw.End-tw.Begin)*t))
}

// StepTween is an interpolation between two integers that floors.
type StepTween struct{ Tween[int] }

// StepTweenOptions are StepTween's named parameters.
type StepTweenOptions struct{ Begin, End int }

// NewStepTween creates a step tween.
func NewStepTween(opts ...StepTweenOptions) *StepTween {
	tw := &StepTween{}
	if len(opts) > 0 {
		tw.Begin, tw.End = opts[0].Begin, opts[0].End
	}
	tw.SetSelf(tw)
	return tw
}

// Lerp is (begin + (end - begin) * t).floor().
func (tw *StepTween) Lerp(t float64) int {
	return dartFloor(float64(tw.Begin) + float64(float64(tw.End-tw.Begin)*t))
}

// ConstantTween is a tween with a constant value.
type ConstantTween[T any] struct{ Tween[T] }

// NewConstantTween creates a tween whose Begin and End are value.
func NewConstantTween[T any](value T) *ConstantTween[T] {
	tw := &ConstantTween[T]{}
	tw.Begin, tw.End = value, value
	tw.SetSelf(tw)
	return tw
}

func (tw *ConstantTween[T]) Lerp(float64) T { return tw.Begin }

func (tw *ConstantTween[T]) String() string {
	return foundation.ObjectRuntimeType(tw.tweenOuter(), "ConstantTween") + "(value: " + dartString(tw.Begin) + ")"
}

// CurveTween transforms the value of the given animation by the given
// curve.
type CurveTween struct {
	AnimatableBase[float64]
	// Curve is the curve to use when transforming the value of the
	// animation.
	Curve Curve
}

// NewCurveTween creates a curve tween (curve is Dart's required named
// parameter).
func NewCurveTween(curve Curve) *CurveTween {
	tw := &CurveTween{Curve: curve}
	tw.SetSelf(tw)
	return tw
}

func (tw *CurveTween) Transform(t float64) float64 {
	if t == 0.0 || t == 1.0 {
		assert(float64(dartRound(tw.Curve.Transform(t))) == t, nil)
		return t
	}
	return tw.Curve.Transform(t)
}

func (tw *CurveTween) String() string {
	return foundation.ObjectRuntimeType(tw.outer(), "CurveTween") + "(curve: " + tw.Curve.String() + ")"
}

// Helpers.

// dartString is Dart's string interpolation of a value ("$v"): null for
// nil, Dart number formatting, String() otherwise.
func dartString(v any) string {
	if isNull(v) {
		return "null"
	}
	switch x := v.(type) {
	case float64:
		return dart.DoubleToString(x)
	case *float64:
		return dart.DoubleToString(*x)
	case int:
		return strconv.Itoa(x)
	case *int:
		return strconv.Itoa(*x)
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case interface{ String() string }:
		return x.String()
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if s, ok := rv.Elem().Interface().(interface{ String() string }); ok {
			return s.String()
		}
	}
	return "Instance of '" + runtimeType(v) + "'"
}

// isNull reports whether v is Dart's null: nil, or a nil pointer, slice,
// map, func or interface.
func isNull(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return rv.IsNil()
	}
	return false
}

// dartFloor is double.floor() (Dart throws for NaN and infinities).
func dartFloor(v float64) int {
	if v != v || v > 9.3e18 || v < -9.3e18 {
		panic(dart.UnsupportedError{Message: "Infinity or NaN toInt"})
	}
	f := int(v)
	if float64(f) > v {
		f--
	}
	return f
}
