package painting

// Port of box_shadow.dart (Flutter 3.50.1): BoxShadow.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// BoxShadow is a shadow cast by a box: a ui.Shadow with a spread radius
// and a blur style. A comparable value (== is Dart's ==); build it with
// NewBoxShadow, whose unset options take Dart's defaults (opaque black).
type BoxShadow struct {
	ui.Shadow
	// SpreadRadius is the amount the box should be inflated prior to
	// applying the blur.
	SpreadRadius float64
	// BlurStyle is the BlurStyle to use for this shadow.
	BlurStyle ui.BlurStyle
}

// BoxShadowOptions are BoxShadow's named parameters (nil Color: opaque
// black, ui.Shadow's default).
type BoxShadowOptions struct {
	Color        *ui.Color
	Offset       ui.Offset
	BlurRadius   float64
	SpreadRadius float64
	BlurStyle    ui.BlurStyle
}

// NewBoxShadow is BoxShadow({...}); it panics on a negative blur radius,
// as ui.Shadow asserts.
func NewBoxShadow(opts ...BoxShadowOptions) BoxShadow {
	var o BoxShadowOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return BoxShadow{Shadow: ui.NewShadow(ui.ShadowOptions{Color: o.Color, Offset: o.Offset, BlurRadius: o.BlurRadius}),
		SpreadRadius: o.SpreadRadius, BlurStyle: o.BlurStyle}
}

// ToPaint creates the Paint object that corresponds to this shadow
// description (no mask filter while DebugDisableShadows is set).
func (s BoxShadow) ToPaint() *ui.Paint {
	result := ui.NewPaint()
	result.SetColor(s.Color)
	m := ui.NewMaskFilterBlur(s.BlurStyle, s.BlurSigma())
	result.SetMaskFilter(&m)
	if foundation.KDebugMode && DebugDisableShadows {
		result.SetMaskFilter(nil)
	}
	return result
}

// Scale returns a new box shadow with its offset, blurRadius and
// spreadRadius scaled by factor.
func (s BoxShadow) Scale(factor float64) BoxShadow {
	c := s.Color
	return NewBoxShadow(BoxShadowOptions{Color: &c, Offset: s.Offset.Mul(factor), BlurRadius: s.BlurRadius * factor,
		SpreadRadius: s.SpreadRadius * factor, BlurStyle: s.BlurStyle})
}

// BoxShadowCopyWithOptions are copyWith's named parameters (nil keeps the
// current value).
type BoxShadowCopyWithOptions struct {
	Color        *ui.Color
	Offset       *ui.Offset
	BlurRadius   *float64
	SpreadRadius *float64
	BlurStyle    *ui.BlurStyle
}

// CopyWith creates a copy of this object but with the given fields
// replaced with the new values.
func (s BoxShadow) CopyWith(o BoxShadowCopyWithOptions) BoxShadow {
	r := BoxShadowOptions{Color: &s.Color, Offset: s.Offset, BlurRadius: s.BlurRadius, SpreadRadius: s.SpreadRadius,
		BlurStyle: s.BlurStyle}
	if o.Color != nil {
		r.Color = o.Color
	}
	if o.Offset != nil {
		r.Offset = *o.Offset
	}
	if o.BlurRadius != nil {
		r.BlurRadius = *o.BlurRadius
	}
	if o.SpreadRadius != nil {
		r.SpreadRadius = *o.SpreadRadius
	}
	if o.BlurStyle != nil {
		r.BlurStyle = *o.BlurStyle
	}
	return NewBoxShadow(r)
}

// BoxShadowLerp is BoxShadow.lerp (nil is null; the same pointer stands
// for identical).
func BoxShadowLerp(a, b *BoxShadow, t float64) *BoxShadow {
	if a == b {
		return a
	}
	if a == nil {
		r := b.Scale(t)
		return &r
	}
	if b == nil {
		r := a.Scale(1.0 - t)
		return &r
	}
	blurStyle := a.BlurStyle
	if a.BlurStyle == ui.BlurStyleNormal {
		blurStyle = b.BlurStyle
	}
	r := NewBoxShadow(BoxShadowOptions{Color: ui.ColorLerp(&a.Color, &b.Color, t), Offset: *ui.OffsetLerp(&a.Offset, &b.Offset, t),
		BlurRadius: lerpD(a.BlurRadius, b.BlurRadius, t), SpreadRadius: lerpD(a.SpreadRadius, b.SpreadRadius, t),
		BlurStyle: blurStyle})
	return &r
}

// sameSlice is Dart's identical on two lists (nil is null).
func sameSlice[T any](a, b []T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0]) // two empty lists lerp to an empty list either way
}

// BoxShadowLerpList is BoxShadow.lerpList: lerps the common prefix and
// scales the rest (nil is null; the same slice stands for identical).
func BoxShadowLerpList(a, b []BoxShadow, t float64) []BoxShadow {
	if sameSlice(a, b) {
		return a
	}
	commonLength := min(len(a), len(b))
	result := []BoxShadow{}
	for i := 0; i < commonLength; i++ {
		result = append(result, *BoxShadowLerp(&a[i], &b[i], t))
	}
	for i := commonLength; i < len(a); i++ {
		result = append(result, a[i].Scale(1.0-t))
	}
	for i := commonLength; i < len(b); i++ {
		result = append(result, b[i].Scale(t))
	}
	return result
}

func (s BoxShadow) String() string {
	return "BoxShadow(" + s.Color.String() + ", " + s.Offset.String() + ", " + formatDouble(s.BlurRadius) + ", " +
		formatDouble(s.SpreadRadius) + ", " + s.BlurStyle.String() + ")"
}
