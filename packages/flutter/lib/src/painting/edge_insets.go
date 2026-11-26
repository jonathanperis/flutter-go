package painting

// Port of edge_insets.dart (Flutter 3.50.1): EdgeInsetsGeometry,
// EdgeInsets, EdgeInsetsDirectional and the private _MixedEdgeInsets.
//
// EdgeInsetsGeometry (abstract) is an interface that only this package's
// types implement. EdgeInsets and EdgeInsetsDirectional are comparable
// value structs whose operators return their own type (Dart's covariant
// overrides); the interface carries the EdgeInsetsGeometry-typed versions
// under Geometry-suffixed names (NegGeometry, MulGeometry, FlippedGeometry...).
// Dart's `add` method is AddGeometry (Add is `operator +`). Dart's `==`
// across the three types (EdgeInsets.zero == EdgeInsetsDirectional.zero is
// true) is EdgeInsetsGeometryEqual; Go's == compares two values of one
// type. ENGINE GAP: dart:ui's ViewPadding is not ported, so
// EdgeInsets.fromViewPadding/fromWindowPadding and
// EdgeInsetsGeometry.fromViewPadding are missing.

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// EdgeInsetsGeometry is the base type of EdgeInsets, EdgeInsetsDirectional
// and their mixed sums: insets with left/right and start/end components.
type EdgeInsetsGeometry interface {
	// lrsetb returns the six components (Dart's private getters).
	lrsetb() (left, right, start, end, top, bottom float64)

	// IsNonNegative reports whether every component is >= 0.
	IsNonNegative() bool
	// Horizontal is the total offset in the horizontal direction.
	Horizontal() float64
	// Vertical is the total offset in the vertical direction.
	Vertical() float64
	// Along is the total offset along the given axis.
	Along(axis Axis) float64
	// CollapsedSize is the size the insets would take with nothing inside.
	CollapsedSize() ui.Size
	// InflateSize returns size grown by the insets.
	InflateSize(size ui.Size) ui.Size
	// DeflateSize returns size shrunk by the insets.
	DeflateSize(size ui.Size) ui.Size
	// Subtract is Dart's subtract(other).
	Subtract(other EdgeInsetsGeometry) EdgeInsetsGeometry
	// AddGeometry is Dart's add(other) (Add is operator + on the
	// concrete types).
	AddGeometry(other EdgeInsetsGeometry) EdgeInsetsGeometry
	// Clamp is Dart's clamp(min, max).
	Clamp(min, max EdgeInsetsGeometry) EdgeInsetsGeometry
	// FlippedGeometry is Dart's flipped as an EdgeInsetsGeometry.
	FlippedGeometry() EdgeInsetsGeometry
	// NegGeometry is unary operator - as an EdgeInsetsGeometry.
	NegGeometry() EdgeInsetsGeometry
	// MulGeometry is operator * as an EdgeInsetsGeometry.
	MulGeometry(other float64) EdgeInsetsGeometry
	// DivGeometry is operator / as an EdgeInsetsGeometry.
	DivGeometry(other float64) EdgeInsetsGeometry
	// TruncDivGeometry is operator ~/ as an EdgeInsetsGeometry.
	TruncDivGeometry(other float64) EdgeInsetsGeometry
	// ModGeometry is operator % as an EdgeInsetsGeometry.
	ModGeometry(other float64) EdgeInsetsGeometry
	// Resolve converts the insets into EdgeInsets for a text direction
	// (nil is null: EdgeInsetsDirectional and mixed insets panic).
	Resolve(direction *ui.TextDirection) EdgeInsets
	String() string
}

// EdgeInsetsGeometry's redirecting factory constructors.

// NewEdgeInsetsGeometryAll is EdgeInsetsGeometry.all (EdgeInsets.all).
func NewEdgeInsetsGeometryAll(value float64) EdgeInsetsGeometry { return NewEdgeInsetsAll(value) }

// NewEdgeInsetsGeometryOnly is EdgeInsetsGeometry.only (EdgeInsets.only).
func NewEdgeInsetsGeometryOnly(e EdgeInsets) EdgeInsetsGeometry { return e }

// NewEdgeInsetsGeometryDirectional is EdgeInsetsGeometry.directional
// (EdgeInsetsDirectional.only).
func NewEdgeInsetsGeometryDirectional(e EdgeInsetsDirectional) EdgeInsetsGeometry { return e }

// NewEdgeInsetsGeometrySymmetric is EdgeInsetsGeometry.symmetric
// (EdgeInsets.symmetric).
func NewEdgeInsetsGeometrySymmetric(opts ...EdgeInsetsSymmetricOptions) EdgeInsetsGeometry {
	return NewEdgeInsetsSymmetric(opts...)
}

// NewEdgeInsetsGeometryFromLTRB is EdgeInsetsGeometry.fromLTRB.
func NewEdgeInsetsGeometryFromLTRB(left, top, right, bottom float64) EdgeInsetsGeometry {
	return EdgeInsets{left, top, right, bottom}
}

// NewEdgeInsetsGeometryFromSTEB is EdgeInsetsGeometry.fromSTEB.
func NewEdgeInsetsGeometryFromSTEB(start, top, end, bottom float64) EdgeInsetsGeometry {
	return EdgeInsetsDirectional{start, top, end, bottom}
}

// EdgeInsetsGeometryZero is EdgeInsetsGeometry.zero (EdgeInsets.zero).
var EdgeInsetsGeometryZero EdgeInsetsGeometry = EdgeInsetsZero

// EdgeInsetsGeometryInfinity is EdgeInsetsGeometry.infinity: every
// component infinite.
var EdgeInsetsGeometryInfinity EdgeInsetsGeometry = mixedEdgeInsets{math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1)}

// The shared EdgeInsetsGeometry members, on the six components.

func geomIsNonNegative(g EdgeInsetsGeometry) bool {
	l, r, s, e, t, b := g.lrsetb()
	return l >= 0.0 && r >= 0.0 && s >= 0.0 && e >= 0.0 && t >= 0.0 && b >= 0.0
}

func geomHorizontal(g EdgeInsetsGeometry) float64 {
	l, r, s, e, _, _ := g.lrsetb()
	return l + r + s + e
}

func geomVertical(g EdgeInsetsGeometry) float64 {
	_, _, _, _, t, b := g.lrsetb()
	return t + b
}

func geomAlong(g EdgeInsetsGeometry, axis Axis) float64 {
	if axis == AxisHorizontal {
		return g.Horizontal()
	}
	return g.Vertical()
}

func geomFlipped(g EdgeInsetsGeometry) EdgeInsetsGeometry {
	l, r, s, e, t, b := g.lrsetb()
	return mixedEdgeInsets{r, l, e, s, b, t}
}

func geomSubtract(g, other EdgeInsetsGeometry) EdgeInsetsGeometry {
	l, r, s, e, t, b := g.lrsetb()
	ol, or, os, oe, ot, ob := other.lrsetb()
	return mixedEdgeInsets{l - ol, r - or, s - os, e - oe, t - ot, b - ob}
}

func geomAdd(g, other EdgeInsetsGeometry) EdgeInsetsGeometry {
	l, r, s, e, t, b := g.lrsetb()
	ol, or, os, oe, ot, ob := other.lrsetb()
	return mixedEdgeInsets{l + ol, r + or, s + os, e + oe, t + ot, b + ob}
}

func geomClamp(g, min, max EdgeInsetsGeometry) EdgeInsetsGeometry {
	l, r, s, e, t, b := g.lrsetb()
	nl, nr, ns, ne, nt, nb := min.lrsetb()
	xl, xr, xs, xe, xt, xb := max.lrsetb()
	return mixedEdgeInsets{
		ui.ClampDouble(l, nl, xl),
		ui.ClampDouble(r, nr, xr),
		ui.ClampDouble(s, ns, xs),
		ui.ClampDouble(e, ne, xe),
		ui.ClampDouble(t, nt, xt),
		ui.ClampDouble(b, nb, xb),
	}
}

func insetsFixed1(v float64) string { return dart.ToStringAsFixed(v, 1) }

func geomString(g EdgeInsetsGeometry) string {
	l, r, s, e, t, b := g.lrsetb()
	if s == 0.0 && e == 0.0 {
		if l == 0.0 && r == 0.0 && t == 0.0 && b == 0.0 {
			return "EdgeInsets.zero"
		}
		if l == r && r == t && t == b {
			return "EdgeInsets.all(" + insetsFixed1(l) + ")"
		}
		return "EdgeInsets(" + insetsFixed1(l) + ", " + insetsFixed1(t) + ", " + insetsFixed1(r) + ", " + insetsFixed1(b) + ")"
	}
	if l == 0.0 && r == 0.0 {
		return "EdgeInsetsDirectional(" + insetsFixed1(s) + ", " + insetsFixed1(t) + ", " + insetsFixed1(e) + ", " + insetsFixed1(b) + ")"
	}
	return "EdgeInsets(" + insetsFixed1(l) + ", " + insetsFixed1(t) + ", " + insetsFixed1(r) + ", " + insetsFixed1(b) + ")" +
		" + " +
		"EdgeInsetsDirectional(" + insetsFixed1(s) + ", " + "0.0, " + insetsFixed1(e) + ", " + "0.0)"
}

// EdgeInsetsGeometryEqual is EdgeInsetsGeometry's ==: the six components
// are equal, whatever the types (nil equals only nil).
func EdgeInsetsGeometryEqual(a, b EdgeInsetsGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	al, ar, as, ae, at, ab := a.lrsetb()
	bl, br, bs, be, bt, bb := b.lrsetb()
	return al == bl && ar == br && as == bs && ae == be && at == bt && ab == bb
}

// lerpD is ui.lerpDouble(a, b, t)! on non-null values.
func insetsLerpDouble(a, b, t float64) float64 { return *ui.LerpDouble(&a, &b, t) }

// EdgeInsetsGeometryLerp is EdgeInsetsGeometry.lerp; a nil interface is
// Dart's null. Equal values stand for Dart's identical (const instances
// are canonicalized) and return a.
func EdgeInsetsGeometryLerp(a, b EdgeInsetsGeometry, t float64) EdgeInsetsGeometry {
	if a == b {
		return a
	}
	if a == nil {
		return b.MulGeometry(t)
	}
	if b == nil {
		return a.MulGeometry(1.0 - t)
	}
	if ea, ok := a.(EdgeInsets); ok {
		if eb, ok := b.(EdgeInsets); ok {
			return *EdgeInsetsLerp(&ea, &eb, t)
		}
	}
	if da, ok := a.(EdgeInsetsDirectional); ok {
		if db, ok := b.(EdgeInsetsDirectional); ok {
			return *EdgeInsetsDirectionalLerp(&da, &db, t)
		}
	}
	al, ar, as, ae, at, ab := a.lrsetb()
	bl, br, bs, be, bt, bb := b.lrsetb()
	return mixedEdgeInsets{
		insetsLerpDouble(al, bl, t),
		insetsLerpDouble(ar, br, t),
		insetsLerpDouble(as, bs, t),
		insetsLerpDouble(ae, be, t),
		insetsLerpDouble(at, bt, t),
		insetsLerpDouble(ab, bb, t),
	}
}

// EdgeInsets is an immutable set of offsets in each of the four cardinal
// directions. The struct literal is EdgeInsets.only (every field defaults
// to 0).
type EdgeInsets struct {
	// Left is the offset from the left.
	Left float64
	// Top is the offset from the top.
	Top float64
	// Right is the offset from the right.
	Right float64
	// Bottom is the offset from the bottom.
	Bottom float64
}

// NewEdgeInsetsFromLTRB is EdgeInsets.fromLTRB.
func NewEdgeInsetsFromLTRB(left, top, right, bottom float64) EdgeInsets {
	return EdgeInsets{left, top, right, bottom}
}

// NewEdgeInsetsAll is EdgeInsets.all: every offset is value.
func NewEdgeInsetsAll(value float64) EdgeInsets { return EdgeInsets{value, value, value, value} }

// NewEdgeInsetsOnly is EdgeInsets.only (the struct literal works too).
func NewEdgeInsetsOnly(e EdgeInsets) EdgeInsets { return e }

// EdgeInsetsSymmetricOptions are EdgeInsets.symmetric's named arguments
// (both default to 0).
type EdgeInsetsSymmetricOptions struct{ Vertical, Horizontal float64 }

// NewEdgeInsetsSymmetric is EdgeInsets.symmetric.
func NewEdgeInsetsSymmetric(opts ...EdgeInsetsSymmetricOptions) EdgeInsets {
	var o EdgeInsetsSymmetricOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return EdgeInsets{o.Horizontal, o.Vertical, o.Horizontal, o.Vertical}
}

// EdgeInsetsZero is EdgeInsets.zero.
var EdgeInsetsZero = EdgeInsets{}

func (e EdgeInsets) lrsetb() (float64, float64, float64, float64, float64, float64) {
	return e.Left, e.Right, 0.0, 0.0, e.Top, e.Bottom
}

func (e EdgeInsets) IsNonNegative() bool                 { return geomIsNonNegative(e) }
func (e EdgeInsets) Horizontal() float64                 { return geomHorizontal(e) }
func (e EdgeInsets) Vertical() float64                   { return geomVertical(e) }
func (e EdgeInsets) Along(axis Axis) float64             { return geomAlong(e, axis) }
func (e EdgeInsets) CollapsedSize() ui.Size              { return ui.NewSize(e.Horizontal(), e.Vertical()) }
func (e EdgeInsets) InflateSize(size ui.Size) ui.Size    { return inflateSize(e, size) }
func (e EdgeInsets) DeflateSize(size ui.Size) ui.Size    { return deflateSize(e, size) }
func (e EdgeInsets) String() string                      { return geomString(e) }
func (e EdgeInsets) FlippedGeometry() EdgeInsetsGeometry { return e.Flipped() }

func inflateSize(g EdgeInsetsGeometry, size ui.Size) ui.Size {
	return ui.NewSize(size.Width+g.Horizontal(), size.Height+g.Vertical())
}

func deflateSize(g EdgeInsetsGeometry, size ui.Size) ui.Size {
	return ui.NewSize(size.Width-g.Horizontal(), size.Height-g.Vertical())
}

// TopLeft is an Offset describing the vector from the top left of a
// rectangle to the top left of that rectangle inset by this object.
func (e EdgeInsets) TopLeft() ui.Offset { return ui.NewOffset(e.Left, e.Top) }

// TopRight is the vector from the top right of a rectangle to the top
// right of that rectangle inset by this object.
func (e EdgeInsets) TopRight() ui.Offset { return ui.NewOffset(-e.Right, e.Top) }

// BottomLeft is the vector from the bottom left of a rectangle to the
// bottom left of that rectangle inset by this object.
func (e EdgeInsets) BottomLeft() ui.Offset { return ui.NewOffset(e.Left, -e.Bottom) }

// BottomRight is the vector from the bottom right of a rectangle to the
// bottom right of that rectangle inset by this object.
func (e EdgeInsets) BottomRight() ui.Offset { return ui.NewOffset(-e.Right, -e.Bottom) }

// Flipped is the EdgeInsets with top and bottom as well as left and right
// flipped.
func (e EdgeInsets) Flipped() EdgeInsets { return EdgeInsets{e.Right, e.Bottom, e.Left, e.Top} }

// InflateRect returns a new rect that is bigger than the given rect in each
// direction by the amount of inset in each direction.
func (e EdgeInsets) InflateRect(rect ui.Rect) ui.Rect {
	return ui.NewRectFromLTRB(rect.Left-e.Left, rect.Top-e.Top, rect.Right+e.Right, rect.Bottom+e.Bottom)
}

// DeflateRect returns a new rect that is smaller than the given rect in
// each direction by the amount of inset in each direction.
func (e EdgeInsets) DeflateRect(rect ui.Rect) ui.Rect {
	return ui.NewRectFromLTRB(rect.Left+e.Left, rect.Top+e.Top, rect.Right-e.Right, rect.Bottom-e.Bottom)
}

// InflateRRect returns a new RRect that is bigger than the given one in
// each direction by the amount of inset in each direction.
func (e EdgeInsets) InflateRRect(rect ui.RRect) ui.RRect {
	zero := ui.RadiusZero
	return ui.NewRRectFromLTRBAndCorners(rect.Left-e.Left, rect.Top-e.Top, rect.Right+e.Right, rect.Bottom+e.Bottom,
		ui.RRectCorners{
			TopLeft:     rect.TlRadius().Add(ui.NewRadiusElliptical(e.Left, e.Top)).Clamp(&zero, nil),
			TopRight:    rect.TrRadius().Add(ui.NewRadiusElliptical(e.Right, e.Top)).Clamp(&zero, nil),
			BottomRight: rect.BrRadius().Add(ui.NewRadiusElliptical(e.Right, e.Bottom)).Clamp(&zero, nil),
			BottomLeft:  rect.BlRadius().Add(ui.NewRadiusElliptical(e.Left, e.Bottom)).Clamp(&zero, nil),
		})
}

// DeflateRRect returns a new RRect that is smaller than the given one in
// each direction by the amount of inset in each direction.
func (e EdgeInsets) DeflateRRect(rect ui.RRect) ui.RRect {
	zero := ui.RadiusZero
	return ui.NewRRectFromLTRBAndCorners(rect.Left+e.Left, rect.Top+e.Top, rect.Right-e.Right, rect.Bottom-e.Bottom,
		ui.RRectCorners{
			TopLeft:     rect.TlRadius().Sub(ui.NewRadiusElliptical(e.Left, e.Top)).Clamp(&zero, nil),
			TopRight:    rect.TrRadius().Sub(ui.NewRadiusElliptical(e.Right, e.Top)).Clamp(&zero, nil),
			BottomRight: rect.BrRadius().Sub(ui.NewRadiusElliptical(e.Right, e.Bottom)).Clamp(&zero, nil),
			BottomLeft:  rect.BlRadius().Sub(ui.NewRadiusElliptical(e.Left, e.Bottom)).Clamp(&zero, nil),
		})
}

// Subtract is Dart's subtract: an EdgeInsets when other is one, mixed
// insets otherwise.
func (e EdgeInsets) Subtract(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsets); ok {
		return e.Sub(o)
	}
	return geomSubtract(e, other)
}

// AddGeometry is Dart's add: an EdgeInsets when other is one, mixed insets
// otherwise.
func (e EdgeInsets) AddGeometry(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsets); ok {
		return e.Add(o)
	}
	return geomAdd(e, other)
}

// Clamp returns EdgeInsets with each side clamped to min's and max's
// (start and end are ignored).
func (e EdgeInsets) Clamp(min, max EdgeInsetsGeometry) EdgeInsetsGeometry {
	nl, nr, _, _, nt, nb := min.lrsetb()
	xl, xr, _, _, xt, xb := max.lrsetb()
	return EdgeInsets{
		ui.ClampDouble(e.Left, nl, xl),
		ui.ClampDouble(e.Top, nt, xt),
		ui.ClampDouble(e.Right, nr, xr),
		ui.ClampDouble(e.Bottom, nb, xb),
	}
}

// Sub is operator -(EdgeInsets).
func (e EdgeInsets) Sub(other EdgeInsets) EdgeInsets {
	return EdgeInsets{e.Left - other.Left, e.Top - other.Top, e.Right - other.Right, e.Bottom - other.Bottom}
}

// Add is operator +(EdgeInsets).
func (e EdgeInsets) Add(other EdgeInsets) EdgeInsets {
	return EdgeInsets{e.Left + other.Left, e.Top + other.Top, e.Right + other.Right, e.Bottom + other.Bottom}
}

// Neg is unary operator -.
func (e EdgeInsets) Neg() EdgeInsets { return EdgeInsets{-e.Left, -e.Top, -e.Right, -e.Bottom} }

// Mul is operator *.
func (e EdgeInsets) Mul(other float64) EdgeInsets {
	return EdgeInsets{e.Left * other, e.Top * other, e.Right * other, e.Bottom * other}
}

// Div is operator /.
func (e EdgeInsets) Div(other float64) EdgeInsets {
	return EdgeInsets{e.Left / other, e.Top / other, e.Right / other, e.Bottom / other}
}

// TruncDiv is operator ~/ (each side's integer quotient, as a double).
func (e EdgeInsets) TruncDiv(other float64) EdgeInsets {
	return EdgeInsets{dart.TruncDiv(e.Left, other), dart.TruncDiv(e.Top, other),
		dart.TruncDiv(e.Right, other), dart.TruncDiv(e.Bottom, other)}
}

// Mod is operator %.
func (e EdgeInsets) Mod(other float64) EdgeInsets {
	return EdgeInsets{dart.Mod(e.Left, other), dart.Mod(e.Top, other), dart.Mod(e.Right, other), dart.Mod(e.Bottom, other)}
}

func (e EdgeInsets) NegGeometry() EdgeInsetsGeometry               { return e.Neg() }
func (e EdgeInsets) MulGeometry(o float64) EdgeInsetsGeometry      { return e.Mul(o) }
func (e EdgeInsets) DivGeometry(o float64) EdgeInsetsGeometry      { return e.Div(o) }
func (e EdgeInsets) TruncDivGeometry(o float64) EdgeInsetsGeometry { return e.TruncDiv(o) }
func (e EdgeInsets) ModGeometry(o float64) EdgeInsetsGeometry      { return e.Mod(o) }

// Resolve returns e (EdgeInsets are already resolved).
func (e EdgeInsets) Resolve(direction *ui.TextDirection) EdgeInsets { return e }

// EdgeInsetsCopyWithOptions are EdgeInsets.copyWith's arguments (nil:
// keep the current value).
type EdgeInsetsCopyWithOptions struct{ Left, Top, Right, Bottom *float64 }

// CopyWith returns a copy with the given fields replaced.
func (e EdgeInsets) CopyWith(opts ...EdgeInsetsCopyWithOptions) EdgeInsets {
	var o EdgeInsetsCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return EdgeInsets{insetsOrFloat(o.Left, e.Left), insetsOrFloat(o.Top, e.Top), insetsOrFloat(o.Right, e.Right), insetsOrFloat(o.Bottom, e.Bottom)}
}

func insetsOrFloat(p *float64, d float64) float64 {
	if p != nil {
		return *p
	}
	return d
}

// EdgeInsetsLerp is EdgeInsets.lerp; nil is Dart's null. Equal values
// stand for Dart's identical and return a.
func EdgeInsetsLerp(a, b *EdgeInsets, t float64) *EdgeInsets {
	if a == b || a != nil && b != nil && *a == *b {
		return a
	}
	var r EdgeInsets
	switch {
	case a == nil:
		r = b.Mul(t)
	case b == nil:
		r = a.Mul(1.0 - t)
	default:
		r = EdgeInsets{
			insetsLerpDouble(a.Left, b.Left, t),
			insetsLerpDouble(a.Top, b.Top, t),
			insetsLerpDouble(a.Right, b.Right, t),
			insetsLerpDouble(a.Bottom, b.Bottom, t),
		}
	}
	return &r
}

// EdgeInsetsDirectional is an immutable set of offsets in each of the four
// cardinal directions, whose horizontal components depend on the writing
// direction. The struct literal is EdgeInsetsDirectional.only.
type EdgeInsetsDirectional struct {
	// Start is the offset from the start side (left in LTR, right in RTL).
	Start float64
	// Top is the offset from the top.
	Top float64
	// End is the offset from the end side (right in LTR, left in RTL).
	End float64
	// Bottom is the offset from the bottom.
	Bottom float64
}

// NewEdgeInsetsDirectionalFromSTEB is EdgeInsetsDirectional.fromSTEB.
func NewEdgeInsetsDirectionalFromSTEB(start, top, end, bottom float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{start, top, end, bottom}
}

// NewEdgeInsetsDirectionalOnly is EdgeInsetsDirectional.only (the struct
// literal works too).
func NewEdgeInsetsDirectionalOnly(e EdgeInsetsDirectional) EdgeInsetsDirectional { return e }

// NewEdgeInsetsDirectionalSymmetric is EdgeInsetsDirectional.symmetric.
func NewEdgeInsetsDirectionalSymmetric(opts ...EdgeInsetsSymmetricOptions) EdgeInsetsDirectional {
	var o EdgeInsetsSymmetricOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return EdgeInsetsDirectional{o.Horizontal, o.Vertical, o.Horizontal, o.Vertical}
}

// NewEdgeInsetsDirectionalAll is EdgeInsetsDirectional.all.
func NewEdgeInsetsDirectionalAll(value float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{value, value, value, value}
}

// EdgeInsetsDirectionalZero is EdgeInsetsDirectional.zero.
var EdgeInsetsDirectionalZero = EdgeInsetsDirectional{}

func (e EdgeInsetsDirectional) lrsetb() (float64, float64, float64, float64, float64, float64) {
	return 0.0, 0.0, e.Start, e.End, e.Top, e.Bottom
}

// IsNonNegative reports whether every side is >= 0.
func (e EdgeInsetsDirectional) IsNonNegative() bool {
	return e.Start >= 0.0 && e.Top >= 0.0 && e.End >= 0.0 && e.Bottom >= 0.0
}

func (e EdgeInsetsDirectional) Horizontal() float64     { return geomHorizontal(e) }
func (e EdgeInsetsDirectional) Vertical() float64       { return geomVertical(e) }
func (e EdgeInsetsDirectional) Along(axis Axis) float64 { return geomAlong(e, axis) }
func (e EdgeInsetsDirectional) CollapsedSize() ui.Size {
	return ui.NewSize(e.Horizontal(), e.Vertical())
}
func (e EdgeInsetsDirectional) InflateSize(size ui.Size) ui.Size { return inflateSize(e, size) }
func (e EdgeInsetsDirectional) DeflateSize(size ui.Size) ui.Size { return deflateSize(e, size) }
func (e EdgeInsetsDirectional) String() string                   { return geomString(e) }
func (e EdgeInsetsDirectional) Clamp(min, max EdgeInsetsGeometry) EdgeInsetsGeometry {
	return geomClamp(e, min, max)
}
func (e EdgeInsetsDirectional) FlippedGeometry() EdgeInsetsGeometry { return e.Flipped() }

// Flipped is the insets with top and bottom as well as start and end
// flipped.
func (e EdgeInsetsDirectional) Flipped() EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.End, e.Bottom, e.Start, e.Top}
}

// Subtract is Dart's subtract: an EdgeInsetsDirectional when other is one,
// mixed insets otherwise.
func (e EdgeInsetsDirectional) Subtract(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsetsDirectional); ok {
		return e.Sub(o)
	}
	return geomSubtract(e, other)
}

// AddGeometry is Dart's add: an EdgeInsetsDirectional when other is one,
// mixed insets otherwise.
func (e EdgeInsetsDirectional) AddGeometry(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	if o, ok := other.(EdgeInsetsDirectional); ok {
		return e.Add(o)
	}
	return geomAdd(e, other)
}

// Sub is operator -(EdgeInsetsDirectional).
func (e EdgeInsetsDirectional) Sub(other EdgeInsetsDirectional) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start - other.Start, e.Top - other.Top, e.End - other.End, e.Bottom - other.Bottom}
}

// Add is operator +(EdgeInsetsDirectional).
func (e EdgeInsetsDirectional) Add(other EdgeInsetsDirectional) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start + other.Start, e.Top + other.Top, e.End + other.End, e.Bottom + other.Bottom}
}

// Neg is unary operator -.
func (e EdgeInsetsDirectional) Neg() EdgeInsetsDirectional {
	return EdgeInsetsDirectional{-e.Start, -e.Top, -e.End, -e.Bottom}
}

// Mul is operator *.
func (e EdgeInsetsDirectional) Mul(other float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start * other, e.Top * other, e.End * other, e.Bottom * other}
}

// Div is operator /.
func (e EdgeInsetsDirectional) Div(other float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{e.Start / other, e.Top / other, e.End / other, e.Bottom / other}
}

// TruncDiv is operator ~/.
func (e EdgeInsetsDirectional) TruncDiv(other float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{dart.TruncDiv(e.Start, other), dart.TruncDiv(e.Top, other),
		dart.TruncDiv(e.End, other), dart.TruncDiv(e.Bottom, other)}
}

// Mod is operator %.
func (e EdgeInsetsDirectional) Mod(other float64) EdgeInsetsDirectional {
	return EdgeInsetsDirectional{dart.Mod(e.Start, other), dart.Mod(e.Top, other),
		dart.Mod(e.End, other), dart.Mod(e.Bottom, other)}
}

func (e EdgeInsetsDirectional) NegGeometry() EdgeInsetsGeometry               { return e.Neg() }
func (e EdgeInsetsDirectional) MulGeometry(o float64) EdgeInsetsGeometry      { return e.Mul(o) }
func (e EdgeInsetsDirectional) DivGeometry(o float64) EdgeInsetsGeometry      { return e.Div(o) }
func (e EdgeInsetsDirectional) TruncDivGeometry(o float64) EdgeInsetsGeometry { return e.TruncDiv(o) }
func (e EdgeInsetsDirectional) ModGeometry(o float64) EdgeInsetsGeometry      { return e.Mod(o) }

// Resolve converts the insets to EdgeInsets for the given direction; nil
// panics with Flutter's "No TextDirection found." error.
func (e EdgeInsetsDirectional) Resolve(direction *ui.TextDirection) EdgeInsets {
	checkCanResolveTextDirection(direction, "EdgeInsetsDirectional")
	if *direction == ui.TextDirectionRtl {
		return EdgeInsets{e.End, e.Top, e.Start, e.Bottom}
	}
	return EdgeInsets{e.Start, e.Top, e.End, e.Bottom}
}

// EdgeInsetsDirectionalCopyWithOptions are EdgeInsetsDirectional.copyWith's
// arguments (nil: keep the current value).
type EdgeInsetsDirectionalCopyWithOptions struct{ Start, Top, End, Bottom *float64 }

// CopyWith returns a copy with the given fields replaced.
func (e EdgeInsetsDirectional) CopyWith(opts ...EdgeInsetsDirectionalCopyWithOptions) EdgeInsetsDirectional {
	var o EdgeInsetsDirectionalCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return EdgeInsetsDirectional{insetsOrFloat(o.Start, e.Start), insetsOrFloat(o.Top, e.Top), insetsOrFloat(o.End, e.End), insetsOrFloat(o.Bottom, e.Bottom)}
}

// EdgeInsetsDirectionalLerp is EdgeInsetsDirectional.lerp; nil is Dart's
// null. Equal values stand for Dart's identical and return a.
func EdgeInsetsDirectionalLerp(a, b *EdgeInsetsDirectional, t float64) *EdgeInsetsDirectional {
	if a == b || a != nil && b != nil && *a == *b {
		return a
	}
	var r EdgeInsetsDirectional
	switch {
	case a == nil:
		r = b.Mul(t)
	case b == nil:
		r = a.Mul(1.0 - t)
	default:
		r = EdgeInsetsDirectional{
			insetsLerpDouble(a.Start, b.Start, t),
			insetsLerpDouble(a.Top, b.Top, t),
			insetsLerpDouble(a.End, b.End, t),
			insetsLerpDouble(a.Bottom, b.Bottom, t),
		}
	}
	return &r
}

// mixedEdgeInsets is Dart's _MixedEdgeInsets: the sum of EdgeInsets and
// EdgeInsetsDirectional components.
type mixedEdgeInsets struct{ left, right, start, end, top, bottom float64 }

func (m mixedEdgeInsets) lrsetb() (float64, float64, float64, float64, float64, float64) {
	return m.left, m.right, m.start, m.end, m.top, m.bottom
}

func (m mixedEdgeInsets) IsNonNegative() bool                 { return geomIsNonNegative(m) }
func (m mixedEdgeInsets) Horizontal() float64                 { return geomHorizontal(m) }
func (m mixedEdgeInsets) Vertical() float64                   { return geomVertical(m) }
func (m mixedEdgeInsets) Along(axis Axis) float64             { return geomAlong(m, axis) }
func (m mixedEdgeInsets) CollapsedSize() ui.Size              { return ui.NewSize(m.Horizontal(), m.Vertical()) }
func (m mixedEdgeInsets) InflateSize(size ui.Size) ui.Size    { return inflateSize(m, size) }
func (m mixedEdgeInsets) DeflateSize(size ui.Size) ui.Size    { return deflateSize(m, size) }
func (m mixedEdgeInsets) String() string                      { return geomString(m) }
func (m mixedEdgeInsets) FlippedGeometry() EdgeInsetsGeometry { return geomFlipped(m) }
func (m mixedEdgeInsets) Subtract(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	return geomSubtract(m, other)
}
func (m mixedEdgeInsets) AddGeometry(other EdgeInsetsGeometry) EdgeInsetsGeometry {
	return geomAdd(m, other)
}
func (m mixedEdgeInsets) Clamp(min, max EdgeInsetsGeometry) EdgeInsetsGeometry {
	return geomClamp(m, min, max)
}

func (m mixedEdgeInsets) NegGeometry() EdgeInsetsGeometry {
	return mixedEdgeInsets{-m.left, -m.right, -m.start, -m.end, -m.top, -m.bottom}
}

func (m mixedEdgeInsets) MulGeometry(o float64) EdgeInsetsGeometry {
	return mixedEdgeInsets{m.left * o, m.right * o, m.start * o, m.end * o, m.top * o, m.bottom * o}
}

func (m mixedEdgeInsets) DivGeometry(o float64) EdgeInsetsGeometry {
	return mixedEdgeInsets{m.left / o, m.right / o, m.start / o, m.end / o, m.top / o, m.bottom / o}
}

func (m mixedEdgeInsets) TruncDivGeometry(o float64) EdgeInsetsGeometry {
	return mixedEdgeInsets{dart.TruncDiv(m.left, o), dart.TruncDiv(m.right, o), dart.TruncDiv(m.start, o),
		dart.TruncDiv(m.end, o), dart.TruncDiv(m.top, o), dart.TruncDiv(m.bottom, o)}
}

func (m mixedEdgeInsets) ModGeometry(o float64) EdgeInsetsGeometry {
	return mixedEdgeInsets{dart.Mod(m.left, o), dart.Mod(m.right, o), dart.Mod(m.start, o),
		dart.Mod(m.end, o), dart.Mod(m.top, o), dart.Mod(m.bottom, o)}
}

func (m mixedEdgeInsets) Resolve(direction *ui.TextDirection) EdgeInsets {
	checkCanResolveTextDirection(direction, "_MixedEdgeInsets")
	if *direction == ui.TextDirectionRtl {
		return EdgeInsets{m.end + m.left, m.top, m.start + m.right, m.bottom}
	}
	return EdgeInsets{m.start + m.left, m.top, m.end + m.right, m.bottom}
}

// checkCanResolveTextDirection is debug.dart's
// debugCheckCanResolveTextDirection (assertions are always on in Go): it
// panics with Flutter's FlutterError when direction is nil.
// TODO(painting debug.dart port): call DebugCheckCanResolveTextDirection.
func checkCanResolveTextDirection(direction *ui.TextDirection, target string) {
	if direction != nil {
		return
	}
	panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
		foundation.NewErrorSummary("No TextDirection found."),
		foundation.NewErrorDescription("To resolve " + target + " properties, it must be provided with a TextDirection."),
		foundation.NewErrorHint("This error usually occurs when " + target + " is used in a widget without " +
			"a Directionality ancestor."),
		foundation.NewErrorHint("Typically, the Directionality widget is introduced by the MaterialApp " +
			"or WidgetsApp widget at the top of your application widget tree. It " +
			"determines the ambient reading direction and is used, for example, to " +
			"determine how to lay out text, how to interpret \"start\" and \"end\" " +
			"values, and to resolve EdgeInsetsDirectional, " +
			"AlignmentDirectional, and other *Directional objects."),
	}))
}
