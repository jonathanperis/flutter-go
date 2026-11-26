package painting

// Port of border_radius.dart (Flutter 3.50.1): BorderRadiusGeometry,
// BorderRadius, BorderRadiusDirectional and the private _MixedBorderRadius.
//
// As in edge_insets.go, BorderRadiusGeometry is an interface only this
// package implements; the concrete types are comparable value structs
// whose operators return their own type, and the interface carries the
// BorderRadiusGeometry-typed versions (NegGeometry, MulGeometry...). Dart's
// `add` method is AddGeometry. Dart's == requires the same runtime type,
// which is Go's == on two BorderRadiusGeometry interface values.

import (
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// BorderRadiusGeometry is the base type of BorderRadius,
// BorderRadiusDirectional and their mixed sums.
type BorderRadiusGeometry interface {
	// corners returns the eight radii (Dart's private getters): topLeft,
	// topRight, bottomLeft, bottomRight, topStart, topEnd, bottomStart,
	// bottomEnd.
	corners() [8]ui.Radius

	// Subtract is Dart's subtract(other).
	Subtract(other BorderRadiusGeometry) BorderRadiusGeometry
	// AddGeometry is Dart's add(other) (Add is operator + on the
	// concrete types).
	AddGeometry(other BorderRadiusGeometry) BorderRadiusGeometry
	// NegGeometry is unary operator - as a BorderRadiusGeometry.
	NegGeometry() BorderRadiusGeometry
	// MulGeometry is operator * as a BorderRadiusGeometry.
	MulGeometry(other float64) BorderRadiusGeometry
	// DivGeometry is operator / as a BorderRadiusGeometry.
	DivGeometry(other float64) BorderRadiusGeometry
	// TruncDivGeometry is operator ~/ as a BorderRadiusGeometry.
	TruncDivGeometry(other float64) BorderRadiusGeometry
	// ModGeometry is operator % as a BorderRadiusGeometry.
	ModGeometry(other float64) BorderRadiusGeometry
	// Resolve converts the radii into a BorderRadius for a text direction
	// (nil is null: directional and mixed radii panic).
	Resolve(direction *ui.TextDirection) BorderRadius
	String() string
}

// BorderRadiusGeometry's factory constructors.

// NewBorderRadiusGeometryAll is BorderRadiusGeometry.all (BorderRadius.all).
func NewBorderRadiusGeometryAll(radius ui.Radius) BorderRadiusGeometry {
	return NewBorderRadiusAll(radius)
}

// NewBorderRadiusGeometryCircular is BorderRadiusGeometry.circular.
func NewBorderRadiusGeometryCircular(radius float64) BorderRadiusGeometry {
	return NewBorderRadiusCircular(radius)
}

// BorderRadiusGeometryHorizontalOptions are BorderRadiusGeometry.horizontal's
// arguments (nil: not given). Left/Right make a BorderRadius, Start/End a
// BorderRadiusDirectional; they can't be mixed.
type BorderRadiusGeometryHorizontalOptions struct{ Left, Right, Start, End *ui.Radius }

// NewBorderRadiusGeometryHorizontal is BorderRadiusGeometry.horizontal.
func NewBorderRadiusGeometryHorizontal(opts ...BorderRadiusGeometryHorizontalOptions) BorderRadiusGeometry {
	var o BorderRadiusGeometryHorizontalOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if !((o.Left == nil && o.Right == nil) || (o.Start == nil && o.End == nil)) {
		panic(dart.NewAssertionError("The left and right values cannot be used in conjunction with start and end."))
	}
	or := func(p *ui.Radius) ui.Radius {
		if p != nil {
			return *p
		}
		return ui.RadiusZero
	}
	if o.Start != nil || o.End != nil {
		return NewBorderRadiusDirectionalHorizontal(BorderRadiusDirectionalHorizontalOptions{Start: or(o.Start), End: or(o.End)})
	}
	return NewBorderRadiusHorizontal(BorderRadiusHorizontalOptions{Left: or(o.Left), Right: or(o.Right)})
}

// NewBorderRadiusGeometryOnly is BorderRadiusGeometry.only (BorderRadius.only).
func NewBorderRadiusGeometryOnly(r BorderRadius) BorderRadiusGeometry { return r }

// NewBorderRadiusGeometryDirectional is BorderRadiusGeometry.directional
// (BorderRadiusDirectional.only).
func NewBorderRadiusGeometryDirectional(r BorderRadiusDirectional) BorderRadiusGeometry { return r }

// NewBorderRadiusGeometryVertical is BorderRadiusGeometry.vertical
// (BorderRadius.vertical).
func NewBorderRadiusGeometryVertical(opts ...BorderRadiusVerticalOptions) BorderRadiusGeometry {
	return NewBorderRadiusVertical(opts...)
}

// BorderRadiusGeometryZero is BorderRadiusGeometry.zero (BorderRadius.zero).
var BorderRadiusGeometryZero BorderRadiusGeometry = BorderRadiusZero

func radiiSubtract(a, b BorderRadiusGeometry) BorderRadiusGeometry {
	x, y := a.corners(), b.corners()
	var m mixedBorderRadius
	for i := range m {
		m[i] = x[i].Sub(y[i])
	}
	return m
}

func radiiAdd(a, b BorderRadiusGeometry) BorderRadiusGeometry {
	x, y := a.corners(), b.corners()
	var m mixedBorderRadius
	for i := range m {
		m[i] = x[i].Add(y[i])
	}
	return m
}

// BorderRadiusGeometryLerp is BorderRadiusGeometry.lerp: a + (b - a) * t
// with the types' own operators; a nil interface is Dart's null (zero
// radii). Equal values stand for Dart's identical and return a.
func BorderRadiusGeometryLerp(a, b BorderRadiusGeometry, t float64) BorderRadiusGeometry {
	if a == b {
		return a
	}
	if a == nil {
		a = BorderRadiusZero
	}
	if b == nil {
		b = BorderRadiusZero
	}
	return a.AddGeometry(b.Subtract(a).MulGeometry(t))
}

func radiiString(g BorderRadiusGeometry) string {
	c := g.corners()
	zero := ui.RadiusZero
	var visual, logical *string
	part := func(name string, all bool, radii [4]ui.Radius, labels [4]string) *string {
		if all {
			if radii[0] != zero {
				var s string
				if radii[0].X == radii[0].Y {
					s = name + ".circular(" + insetsFixed1(radii[0].X) + ")"
				} else {
					s = name + ".all(" + radii[0].String() + ")"
				}
				return &s
			}
			return nil
		}
		var b strings.Builder
		b.WriteString(name + ".only(")
		comma := false
		for i, r := range radii {
			if r != zero {
				if comma {
					b.WriteString(", ")
				}
				b.WriteString(labels[i] + ": " + r.String())
				comma = true
			}
		}
		b.WriteString(")")
		s := b.String()
		return &s
	}
	tl, tr, bl, br := c[0], c[1], c[2], c[3]
	visual = part("BorderRadius", tl == tr && tr == bl && bl == br, [4]ui.Radius{tl, tr, bl, br},
		[4]string{"topLeft", "topRight", "bottomLeft", "bottomRight"})
	ts, te, bs, be := c[4], c[5], c[6], c[7]
	logical = part("BorderRadiusDirectional", ts == te && te == be && be == bs, [4]ui.Radius{ts, te, bs, be},
		[4]string{"topStart", "topEnd", "bottomStart", "bottomEnd"})
	if visual != nil && logical != nil {
		return *visual + " + " + *logical
	}
	if visual != nil {
		return *visual
	}
	if logical != nil {
		return *logical
	}
	return "BorderRadius.zero"
}

// BorderRadius is an immutable set of radii for each corner of a
// rectangle. The struct literal is BorderRadius.only (every corner
// defaults to Radius.zero).
type BorderRadius struct {
	TopLeft, TopRight, BottomLeft, BottomRight ui.Radius
}

// NewBorderRadiusAll is BorderRadius.all: every corner has radius.
func NewBorderRadiusAll(radius ui.Radius) BorderRadius {
	return BorderRadius{radius, radius, radius, radius}
}

// NewBorderRadiusCircular is BorderRadius.circular: every corner has a
// circular radius.
func NewBorderRadiusCircular(radius float64) BorderRadius {
	return NewBorderRadiusAll(ui.NewRadiusCircular(radius))
}

// BorderRadiusVerticalOptions are BorderRadius.vertical's arguments (zero:
// Radius.zero, the Dart default).
type BorderRadiusVerticalOptions struct{ Top, Bottom ui.Radius }

// NewBorderRadiusVertical is BorderRadius.vertical.
func NewBorderRadiusVertical(opts ...BorderRadiusVerticalOptions) BorderRadius {
	var o BorderRadiusVerticalOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return BorderRadius{o.Top, o.Top, o.Bottom, o.Bottom}
}

// BorderRadiusHorizontalOptions are BorderRadius.horizontal's arguments.
type BorderRadiusHorizontalOptions struct{ Left, Right ui.Radius }

// NewBorderRadiusHorizontal is BorderRadius.horizontal.
func NewBorderRadiusHorizontal(opts ...BorderRadiusHorizontalOptions) BorderRadius {
	var o BorderRadiusHorizontalOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return BorderRadius{o.Left, o.Right, o.Left, o.Right}
}

// NewBorderRadiusOnly is BorderRadius.only (the struct literal works too).
func NewBorderRadiusOnly(r BorderRadius) BorderRadius { return r }

// BorderRadiusZero is BorderRadius.zero.
var BorderRadiusZero = BorderRadius{}

// BorderRadiusCopyWithOptions are BorderRadius.copyWith's arguments (nil:
// keep the current value).
type BorderRadiusCopyWithOptions struct{ TopLeft, TopRight, BottomLeft, BottomRight *ui.Radius }

// CopyWith returns a copy with the given corners replaced.
func (r BorderRadius) CopyWith(opts ...BorderRadiusCopyWithOptions) BorderRadius {
	var o BorderRadiusCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	or := func(p *ui.Radius, d ui.Radius) ui.Radius {
		if p != nil {
			return *p
		}
		return d
	}
	return BorderRadius{or(o.TopLeft, r.TopLeft), or(o.TopRight, r.TopRight),
		or(o.BottomLeft, r.BottomLeft), or(o.BottomRight, r.BottomRight)}
}

func (r BorderRadius) corners() [8]ui.Radius {
	z := ui.RadiusZero
	return [8]ui.Radius{r.TopLeft, r.TopRight, r.BottomLeft, r.BottomRight, z, z, z, z}
}

func (r BorderRadius) String() string { return radiiString(r) }

// ToRRect creates an RRect from the current border radius and a Rect
// (negative radii are clamped to zero).
func (r BorderRadius) ToRRect(rect ui.Rect) ui.RRect {
	zero := ui.RadiusZero
	return ui.NewRRectFromRectAndCorners(rect, ui.RRectCorners{
		TopLeft:     r.TopLeft.Clamp(&zero, nil),
		TopRight:    r.TopRight.Clamp(&zero, nil),
		BottomLeft:  r.BottomLeft.Clamp(&zero, nil),
		BottomRight: r.BottomRight.Clamp(&zero, nil),
	})
}

// ToRSuperellipse creates an RSuperellipse from the current border radius
// and a Rect (negative radii are clamped to zero).
func (r BorderRadius) ToRSuperellipse(rect ui.Rect) ui.RSuperellipse {
	zero := ui.RadiusZero
	return ui.NewRSuperellipseFromRectAndCorners(rect, ui.RRectCorners{
		TopLeft:     r.TopLeft.Clamp(&zero, nil),
		TopRight:    r.TopRight.Clamp(&zero, nil),
		BottomLeft:  r.BottomLeft.Clamp(&zero, nil),
		BottomRight: r.BottomRight.Clamp(&zero, nil),
	})
}

// Subtract is Dart's subtract: a BorderRadius when other is one, mixed
// radii otherwise.
func (r BorderRadius) Subtract(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadius); ok {
		return r.Sub(o)
	}
	return radiiSubtract(r, other)
}

// AddGeometry is Dart's add: a BorderRadius when other is one, mixed radii
// otherwise.
func (r BorderRadius) AddGeometry(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadius); ok {
		return r.Add(o)
	}
	return radiiAdd(r, other)
}

// Sub is operator -(BorderRadius).
func (r BorderRadius) Sub(o BorderRadius) BorderRadius {
	return BorderRadius{r.TopLeft.Sub(o.TopLeft), r.TopRight.Sub(o.TopRight),
		r.BottomLeft.Sub(o.BottomLeft), r.BottomRight.Sub(o.BottomRight)}
}

// Add is operator +(BorderRadius).
func (r BorderRadius) Add(o BorderRadius) BorderRadius {
	return BorderRadius{r.TopLeft.Add(o.TopLeft), r.TopRight.Add(o.TopRight),
		r.BottomLeft.Add(o.BottomLeft), r.BottomRight.Add(o.BottomRight)}
}

// Neg is unary operator -.
func (r BorderRadius) Neg() BorderRadius {
	return BorderRadius{r.TopLeft.Neg(), r.TopRight.Neg(), r.BottomLeft.Neg(), r.BottomRight.Neg()}
}

// Mul is operator *.
func (r BorderRadius) Mul(o float64) BorderRadius {
	return BorderRadius{r.TopLeft.Mul(o), r.TopRight.Mul(o), r.BottomLeft.Mul(o), r.BottomRight.Mul(o)}
}

// Div is operator /.
func (r BorderRadius) Div(o float64) BorderRadius {
	return BorderRadius{r.TopLeft.Div(o), r.TopRight.Div(o), r.BottomLeft.Div(o), r.BottomRight.Div(o)}
}

// TruncDiv is operator ~/.
func (r BorderRadius) TruncDiv(o float64) BorderRadius {
	return BorderRadius{r.TopLeft.TruncDiv(o), r.TopRight.TruncDiv(o), r.BottomLeft.TruncDiv(o), r.BottomRight.TruncDiv(o)}
}

// Mod is operator %.
func (r BorderRadius) Mod(o float64) BorderRadius {
	return BorderRadius{r.TopLeft.Mod(o), r.TopRight.Mod(o), r.BottomLeft.Mod(o), r.BottomRight.Mod(o)}
}

func (r BorderRadius) NegGeometry() BorderRadiusGeometry               { return r.Neg() }
func (r BorderRadius) MulGeometry(o float64) BorderRadiusGeometry      { return r.Mul(o) }
func (r BorderRadius) DivGeometry(o float64) BorderRadiusGeometry      { return r.Div(o) }
func (r BorderRadius) TruncDivGeometry(o float64) BorderRadiusGeometry { return r.TruncDiv(o) }
func (r BorderRadius) ModGeometry(o float64) BorderRadiusGeometry      { return r.Mod(o) }

// Resolve returns r (a BorderRadius is already resolved).
func (r BorderRadius) Resolve(direction *ui.TextDirection) BorderRadius { return r }

// BorderRadiusLerp is BorderRadius.lerp; nil is Dart's null. Equal values
// stand for Dart's identical and return a.
func BorderRadiusLerp(a, b *BorderRadius, t float64) *BorderRadius {
	if a == b || a != nil && b != nil && *a == *b {
		return a
	}
	var r BorderRadius
	switch {
	case a == nil:
		r = b.Mul(t)
	case b == nil:
		r = a.Mul(1.0 - t)
	default:
		r = BorderRadius{
			*ui.RadiusLerp(&a.TopLeft, &b.TopLeft, t),
			*ui.RadiusLerp(&a.TopRight, &b.TopRight, t),
			*ui.RadiusLerp(&a.BottomLeft, &b.BottomLeft, t),
			*ui.RadiusLerp(&a.BottomRight, &b.BottomRight, t),
		}
	}
	return &r
}

// BorderRadiusDirectional is an immutable set of radii for each corner of
// a rectangle, whose horizontal corners depend on the text direction. The
// struct literal is BorderRadiusDirectional.only.
type BorderRadiusDirectional struct {
	TopStart, TopEnd, BottomStart, BottomEnd ui.Radius
}

// NewBorderRadiusDirectionalAll is BorderRadiusDirectional.all.
func NewBorderRadiusDirectionalAll(radius ui.Radius) BorderRadiusDirectional {
	return BorderRadiusDirectional{radius, radius, radius, radius}
}

// NewBorderRadiusDirectionalCircular is BorderRadiusDirectional.circular.
func NewBorderRadiusDirectionalCircular(radius float64) BorderRadiusDirectional {
	return NewBorderRadiusDirectionalAll(ui.NewRadiusCircular(radius))
}

// NewBorderRadiusDirectionalVertical is BorderRadiusDirectional.vertical.
func NewBorderRadiusDirectionalVertical(opts ...BorderRadiusVerticalOptions) BorderRadiusDirectional {
	var o BorderRadiusVerticalOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return BorderRadiusDirectional{o.Top, o.Top, o.Bottom, o.Bottom}
}

// BorderRadiusDirectionalHorizontalOptions are
// BorderRadiusDirectional.horizontal's arguments.
type BorderRadiusDirectionalHorizontalOptions struct{ Start, End ui.Radius }

// NewBorderRadiusDirectionalHorizontal is BorderRadiusDirectional.horizontal.
func NewBorderRadiusDirectionalHorizontal(opts ...BorderRadiusDirectionalHorizontalOptions) BorderRadiusDirectional {
	var o BorderRadiusDirectionalHorizontalOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return BorderRadiusDirectional{o.Start, o.End, o.Start, o.End}
}

// NewBorderRadiusDirectionalOnly is BorderRadiusDirectional.only (the
// struct literal works too).
func NewBorderRadiusDirectionalOnly(r BorderRadiusDirectional) BorderRadiusDirectional { return r }

// BorderRadiusDirectionalZero is BorderRadiusDirectional.zero.
var BorderRadiusDirectionalZero = BorderRadiusDirectional{}

func (r BorderRadiusDirectional) corners() [8]ui.Radius {
	z := ui.RadiusZero
	return [8]ui.Radius{z, z, z, z, r.TopStart, r.TopEnd, r.BottomStart, r.BottomEnd}
}

func (r BorderRadiusDirectional) String() string { return radiiString(r) }

// Subtract is Dart's subtract: a BorderRadiusDirectional when other is
// one, mixed radii otherwise.
func (r BorderRadiusDirectional) Subtract(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadiusDirectional); ok {
		return r.Sub(o)
	}
	return radiiSubtract(r, other)
}

// AddGeometry is Dart's add: a BorderRadiusDirectional when other is one,
// mixed radii otherwise.
func (r BorderRadiusDirectional) AddGeometry(other BorderRadiusGeometry) BorderRadiusGeometry {
	if o, ok := other.(BorderRadiusDirectional); ok {
		return r.Add(o)
	}
	return radiiAdd(r, other)
}

// Sub is operator -(BorderRadiusDirectional).
func (r BorderRadiusDirectional) Sub(o BorderRadiusDirectional) BorderRadiusDirectional {
	return BorderRadiusDirectional{r.TopStart.Sub(o.TopStart), r.TopEnd.Sub(o.TopEnd),
		r.BottomStart.Sub(o.BottomStart), r.BottomEnd.Sub(o.BottomEnd)}
}

// Add is operator +(BorderRadiusDirectional).
func (r BorderRadiusDirectional) Add(o BorderRadiusDirectional) BorderRadiusDirectional {
	return BorderRadiusDirectional{r.TopStart.Add(o.TopStart), r.TopEnd.Add(o.TopEnd),
		r.BottomStart.Add(o.BottomStart), r.BottomEnd.Add(o.BottomEnd)}
}

// Neg is unary operator -.
func (r BorderRadiusDirectional) Neg() BorderRadiusDirectional {
	return BorderRadiusDirectional{r.TopStart.Neg(), r.TopEnd.Neg(), r.BottomStart.Neg(), r.BottomEnd.Neg()}
}

// Mul is operator *.
func (r BorderRadiusDirectional) Mul(o float64) BorderRadiusDirectional {
	return BorderRadiusDirectional{r.TopStart.Mul(o), r.TopEnd.Mul(o), r.BottomStart.Mul(o), r.BottomEnd.Mul(o)}
}

// Div is operator /.
func (r BorderRadiusDirectional) Div(o float64) BorderRadiusDirectional {
	return BorderRadiusDirectional{r.TopStart.Div(o), r.TopEnd.Div(o), r.BottomStart.Div(o), r.BottomEnd.Div(o)}
}

// TruncDiv is operator ~/.
func (r BorderRadiusDirectional) TruncDiv(o float64) BorderRadiusDirectional {
	return BorderRadiusDirectional{r.TopStart.TruncDiv(o), r.TopEnd.TruncDiv(o), r.BottomStart.TruncDiv(o), r.BottomEnd.TruncDiv(o)}
}

// Mod is operator %.
func (r BorderRadiusDirectional) Mod(o float64) BorderRadiusDirectional {
	return BorderRadiusDirectional{r.TopStart.Mod(o), r.TopEnd.Mod(o), r.BottomStart.Mod(o), r.BottomEnd.Mod(o)}
}

func (r BorderRadiusDirectional) NegGeometry() BorderRadiusGeometry          { return r.Neg() }
func (r BorderRadiusDirectional) MulGeometry(o float64) BorderRadiusGeometry { return r.Mul(o) }
func (r BorderRadiusDirectional) DivGeometry(o float64) BorderRadiusGeometry { return r.Div(o) }
func (r BorderRadiusDirectional) TruncDivGeometry(o float64) BorderRadiusGeometry {
	return r.TruncDiv(o)
}
func (r BorderRadiusDirectional) ModGeometry(o float64) BorderRadiusGeometry { return r.Mod(o) }

// Resolve converts the radii to a BorderRadius for the given direction;
// nil panics with Flutter's "No TextDirection found." error.
func (r BorderRadiusDirectional) Resolve(direction *ui.TextDirection) BorderRadius {
	checkCanResolveTextDirection(direction, "BorderRadiusDirectional")
	if *direction == ui.TextDirectionRtl {
		return BorderRadius{TopLeft: r.TopEnd, TopRight: r.TopStart, BottomLeft: r.BottomEnd, BottomRight: r.BottomStart}
	}
	return BorderRadius{TopLeft: r.TopStart, TopRight: r.TopEnd, BottomLeft: r.BottomStart, BottomRight: r.BottomEnd}
}

// BorderRadiusDirectionalLerp is BorderRadiusDirectional.lerp; nil is
// Dart's null. Equal values stand for Dart's identical and return a.
func BorderRadiusDirectionalLerp(a, b *BorderRadiusDirectional, t float64) *BorderRadiusDirectional {
	if a == b || a != nil && b != nil && *a == *b {
		return a
	}
	var r BorderRadiusDirectional
	switch {
	case a == nil:
		r = b.Mul(t)
	case b == nil:
		r = a.Mul(1.0 - t)
	default:
		r = BorderRadiusDirectional{
			*ui.RadiusLerp(&a.TopStart, &b.TopStart, t),
			*ui.RadiusLerp(&a.TopEnd, &b.TopEnd, t),
			*ui.RadiusLerp(&a.BottomStart, &b.BottomStart, t),
			*ui.RadiusLerp(&a.BottomEnd, &b.BottomEnd, t),
		}
	}
	return &r
}

// mixedBorderRadius is Dart's _MixedBorderRadius, in corners() order.
type mixedBorderRadius [8]ui.Radius

func (m mixedBorderRadius) corners() [8]ui.Radius { return m }
func (m mixedBorderRadius) String() string        { return radiiString(m) }
func (m mixedBorderRadius) Subtract(other BorderRadiusGeometry) BorderRadiusGeometry {
	return radiiSubtract(m, other)
}
func (m mixedBorderRadius) AddGeometry(other BorderRadiusGeometry) BorderRadiusGeometry {
	return radiiAdd(m, other)
}

func (m mixedBorderRadius) each(f func(ui.Radius) ui.Radius) BorderRadiusGeometry {
	var r mixedBorderRadius
	for i, v := range m {
		r[i] = f(v)
	}
	return r
}

func (m mixedBorderRadius) NegGeometry() BorderRadiusGeometry {
	return m.each(ui.Radius.Neg)
}
func (m mixedBorderRadius) MulGeometry(o float64) BorderRadiusGeometry {
	return m.each(func(r ui.Radius) ui.Radius { return r.Mul(o) })
}
func (m mixedBorderRadius) DivGeometry(o float64) BorderRadiusGeometry {
	return m.each(func(r ui.Radius) ui.Radius { return r.Div(o) })
}
func (m mixedBorderRadius) TruncDivGeometry(o float64) BorderRadiusGeometry {
	return m.each(func(r ui.Radius) ui.Radius { return r.TruncDiv(o) })
}
func (m mixedBorderRadius) ModGeometry(o float64) BorderRadiusGeometry {
	return m.each(func(r ui.Radius) ui.Radius { return r.Mod(o) })
}

func (m mixedBorderRadius) Resolve(direction *ui.TextDirection) BorderRadius {
	checkCanResolveTextDirection(direction, "_MixedBorderRadius")
	tl, tr, bl, br, ts, te, bs, be := m[0], m[1], m[2], m[3], m[4], m[5], m[6], m[7]
	if *direction == ui.TextDirectionRtl {
		return BorderRadius{TopLeft: tl.Add(te), TopRight: tr.Add(ts), BottomLeft: bl.Add(be), BottomRight: br.Add(bs)}
	}
	return BorderRadius{TopLeft: tl.Add(ts), TopRight: tr.Add(te), BottomLeft: bl.Add(bs), BottomRight: br.Add(be)}
}
