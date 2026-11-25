package ui

// Port of engine/src/flutter/lib/ui/geometry.dart (Flutter 3.50.1).

import (
	"math"

	"github.com/jonathanperis/flutter-go/flutter_go/dart"
)

func fixed1(v float64) string { return dart.ToStringAsFixed(v, 1) }

// OffsetBase is what Offset and Size share: a pair of doubles that can be
// compared component-wise. Dart's OffsetBase is abstract; in Go it is the
// interface both types implement.
type OffsetBase interface {
	components() (float64, float64)
	IsInfinite() bool
	IsFinite() bool
	Less(other OffsetBase) bool
	LessEq(other OffsetBase) bool
	Greater(other OffsetBase) bool
	GreaterEq(other OffsetBase) bool
}

func isInfinite(a, b float64) bool { return a >= math.Inf(1) || b >= math.Inf(1) }
func isFinite(v float64) bool      { return !math.IsInf(v, 0) && !math.IsNaN(v) }

// Offset is an immutable 2D floating-point offset.
type Offset struct{ Dx, Dy float64 }

var (
	OffsetZero     = Offset{0, 0}
	OffsetInfinite = Offset{math.Inf(1), math.Inf(1)}
)

func NewOffset(dx, dy float64) Offset { return Offset{dx, dy} }

// NewOffsetFromDirection is Offset.fromDirection(direction, [distance = 1]).
func NewOffsetFromDirection(direction float64, distance ...float64) Offset {
	d := 1.0
	if len(distance) > 0 {
		d = distance[0]
	}
	return Offset{d * math.Cos(direction), d * math.Sin(direction)}
}

func (o Offset) components() (float64, float64) { return o.Dx, o.Dy }
func (o Offset) IsInfinite() bool               { return isInfinite(o.Dx, o.Dy) }
func (o Offset) IsFinite() bool                 { return isFinite(o.Dx) && isFinite(o.Dy) }
func (o Offset) Less(b OffsetBase) bool         { x, y := b.components(); return o.Dx < x && o.Dy < y }
func (o Offset) LessEq(b OffsetBase) bool       { x, y := b.components(); return o.Dx <= x && o.Dy <= y }
func (o Offset) Greater(b OffsetBase) bool      { x, y := b.components(); return o.Dx > x && o.Dy > y }
func (o Offset) GreaterEq(b OffsetBase) bool    { x, y := b.components(); return o.Dx >= x && o.Dy >= y }

func (o Offset) Distance() float64        { return math.Sqrt(o.Dx*o.Dx + o.Dy*o.Dy) }
func (o Offset) DistanceSquared() float64 { return o.Dx*o.Dx + o.Dy*o.Dy }
func (o Offset) Direction() float64       { return math.Atan2(o.Dy, o.Dx) }

func (o Offset) Scale(scaleX, scaleY float64) Offset { return Offset{o.Dx * scaleX, o.Dy * scaleY} }
func (o Offset) Translate(translateX, translateY float64) Offset {
	return Offset{o.Dx + translateX, o.Dy + translateY}
}

func (o Offset) Neg() Offset                { return Offset{-o.Dx, -o.Dy} }
func (o Offset) Sub(other Offset) Offset    { return Offset{o.Dx - other.Dx, o.Dy - other.Dy} }
func (o Offset) Add(other Offset) Offset    { return Offset{o.Dx + other.Dx, o.Dy + other.Dy} }
func (o Offset) Mul(operand float64) Offset { return Offset{o.Dx * operand, o.Dy * operand} }
func (o Offset) Div(operand float64) Offset { return Offset{o.Dx / operand, o.Dy / operand} }
func (o Offset) TruncDiv(operand float64) Offset {
	return Offset{dart.TruncDiv(o.Dx, operand), dart.TruncDiv(o.Dy, operand)}
}
func (o Offset) Mod(operand float64) Offset {
	return Offset{dart.Mod(o.Dx, operand), dart.Mod(o.Dy, operand)}
}

// And is `offset & size`: the rectangle at this offset with that size.
func (o Offset) And(other Size) Rect { return NewRectFromLTWH(o.Dx, o.Dy, other.Width, other.Height) }

// OffsetLerp is Offset.lerp; nil stands for Dart's null.
func OffsetLerp(a, b *Offset, t float64) *Offset {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		r := a.Mul(1.0 - t)
		return &r
	case a == nil:
		r := b.Mul(t)
		return &r
	}
	return &Offset{lerpDouble(a.Dx, b.Dx, t), lerpDouble(a.Dy, b.Dy, t)}
}

func (o Offset) String() string { return "Offset(" + fixed1(o.Dx) + ", " + fixed1(o.Dy) + ")" }

// Size holds a 2D floating-point size.
type Size struct{ Width, Height float64 }

var (
	SizeZero     = Size{0, 0}
	SizeInfinite = Size{math.Inf(1), math.Inf(1)}
)

func NewSize(width, height float64) Size    { return Size{width, height} }
func NewSizeCopy(source Size) Size          { return source }
func NewSizeSquare(dimension float64) Size  { return Size{dimension, dimension} }
func NewSizeFromWidth(width float64) Size   { return Size{width, math.Inf(1)} }
func NewSizeFromHeight(height float64) Size { return Size{math.Inf(1), height} }
func NewSizeFromRadius(radius float64) Size { return Size{radius * 2.0, radius * 2.0} }

func (s Size) components() (float64, float64) { return s.Width, s.Height }
func (s Size) IsInfinite() bool               { return isInfinite(s.Width, s.Height) }
func (s Size) IsFinite() bool                 { return isFinite(s.Width) && isFinite(s.Height) }
func (s Size) Less(b OffsetBase) bool         { x, y := b.components(); return s.Width < x && s.Height < y }
func (s Size) LessEq(b OffsetBase) bool       { x, y := b.components(); return s.Width <= x && s.Height <= y }
func (s Size) Greater(b OffsetBase) bool      { x, y := b.components(); return s.Width > x && s.Height > y }
func (s Size) GreaterEq(b OffsetBase) bool {
	x, y := b.components()
	return s.Width >= x && s.Height >= y
}

func (s Size) AspectRatio() float64 {
	switch {
	case s.Height != 0.0:
		return s.Width / s.Height
	case s.Width > 0.0:
		return math.Inf(1)
	case s.Width < 0.0:
		return math.Inf(-1)
	}
	return 0.0
}

func (s Size) IsEmpty() bool { return s.Width <= 0.0 || s.Height <= 0.0 }

// Sub is `size - other`: an Offset when other is a Size, a Size when other
// is an Offset.
func (s Size) Sub(other OffsetBase) OffsetBase {
	switch o := other.(type) {
	case Size:
		return Offset{s.Width - o.Width, s.Height - o.Height}
	case Offset:
		return Size{s.Width - o.Dx, s.Height - o.Dy}
	}
	panic("ui: Size.Sub: argument must be an Offset or a Size")
}

func (s Size) Add(other Offset) Size    { return Size{s.Width + other.Dx, s.Height + other.Dy} }
func (s Size) Mul(operand float64) Size { return Size{s.Width * operand, s.Height * operand} }
func (s Size) Div(operand float64) Size { return Size{s.Width / operand, s.Height / operand} }
func (s Size) TruncDiv(operand float64) Size {
	return Size{dart.TruncDiv(s.Width, operand), dart.TruncDiv(s.Height, operand)}
}
func (s Size) Mod(operand float64) Size {
	return Size{dart.Mod(s.Width, operand), dart.Mod(s.Height, operand)}
}

func (s Size) ShortestSide() float64 { return math.Min(math.Abs(s.Width), math.Abs(s.Height)) }
func (s Size) LongestSide() float64  { return math.Max(math.Abs(s.Width), math.Abs(s.Height)) }

func (s Size) TopLeft(origin Offset) Offset   { return origin }
func (s Size) TopCenter(origin Offset) Offset { return Offset{origin.Dx + s.Width/2.0, origin.Dy} }
func (s Size) TopRight(origin Offset) Offset  { return Offset{origin.Dx + s.Width, origin.Dy} }
func (s Size) CenterLeft(origin Offset) Offset {
	return Offset{origin.Dx, origin.Dy + s.Height/2.0}
}
func (s Size) Center(origin Offset) Offset {
	return Offset{origin.Dx + s.Width/2.0, origin.Dy + s.Height/2.0}
}
func (s Size) CenterRight(origin Offset) Offset {
	return Offset{origin.Dx + s.Width, origin.Dy + s.Height/2.0}
}
func (s Size) BottomLeft(origin Offset) Offset { return Offset{origin.Dx, origin.Dy + s.Height} }
func (s Size) BottomCenter(origin Offset) Offset {
	return Offset{origin.Dx + s.Width/2.0, origin.Dy + s.Height}
}
func (s Size) BottomRight(origin Offset) Offset {
	return Offset{origin.Dx + s.Width, origin.Dy + s.Height}
}

// Contains reports whether offset is inside a rectangle of this size at the
// origin (right and bottom edges excluded).
func (s Size) Contains(offset Offset) bool {
	return offset.Dx >= 0.0 && offset.Dx < s.Width && offset.Dy >= 0.0 && offset.Dy < s.Height
}

func (s Size) Flipped() Size { return Size{s.Height, s.Width} }

// SizeLerp is Size.lerp; nil stands for Dart's null.
func SizeLerp(a, b *Size, t float64) *Size {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		r := a.Mul(1.0 - t)
		return &r
	case a == nil:
		r := b.Mul(t)
		return &r
	}
	return &Size{lerpDouble(a.Width, b.Width, t), lerpDouble(a.Height, b.Height, t)}
}

func (s Size) String() string { return "Size(" + fixed1(s.Width) + ", " + fixed1(s.Height) + ")" }

// Rect is an immutable, 2D, axis-aligned, floating-point rectangle.
type Rect struct{ Left, Top, Right, Bottom float64 }

const giantScalar = 1.0e+9 // matches kGiantRect from layer.h

var (
	RectZero    = Rect{0, 0, 0, 0}
	RectLargest = Rect{-giantScalar, -giantScalar, giantScalar, giantScalar}
)

func NewRectFromLTRB(left, top, right, bottom float64) Rect { return Rect{left, top, right, bottom} }
func NewRectFromLTWH(left, top, width, height float64) Rect {
	return Rect{left, top, left + width, top + height}
}

// NewRectFromCircle is Rect.fromCircle(center:, radius:).
func NewRectFromCircle(center Offset, radius float64) Rect {
	return NewRectFromCenter(center, radius*2, radius*2)
}

// NewRectFromCenter is Rect.fromCenter(center:, width:, height:).
func NewRectFromCenter(center Offset, width, height float64) Rect {
	return Rect{center.Dx - width/2, center.Dy - height/2, center.Dx + width/2, center.Dy + height/2}
}

func NewRectFromPoints(a, b Offset) Rect {
	return Rect{math.Min(a.Dx, b.Dx), math.Min(a.Dy, b.Dy), math.Max(a.Dx, b.Dx), math.Max(a.Dy, b.Dy)}
}

func (r Rect) Width() float64  { return r.Right - r.Left }
func (r Rect) Height() float64 { return r.Bottom - r.Top }
func (r Rect) Size() Size      { return Size{r.Width(), r.Height()} }
func (r Rect) HasNaN() bool {
	return math.IsNaN(r.Left) || math.IsNaN(r.Top) || math.IsNaN(r.Right) || math.IsNaN(r.Bottom)
}
func (r Rect) IsInfinite() bool {
	inf := math.Inf(1)
	return r.Left >= inf || r.Top >= inf || r.Right >= inf || r.Bottom >= inf
}
func (r Rect) IsFinite() bool {
	return isFinite(r.Left) && isFinite(r.Top) && isFinite(r.Right) && isFinite(r.Bottom)
}
func (r Rect) IsEmpty() bool { return r.Left >= r.Right || r.Top >= r.Bottom }

func (r Rect) Shift(offset Offset) Rect {
	return Rect{r.Left + offset.Dx, r.Top + offset.Dy, r.Right + offset.Dx, r.Bottom + offset.Dy}
}
func (r Rect) Translate(translateX, translateY float64) Rect {
	return Rect{r.Left + translateX, r.Top + translateY, r.Right + translateX, r.Bottom + translateY}
}
func (r Rect) Inflate(delta float64) Rect {
	return Rect{r.Left - delta, r.Top - delta, r.Right + delta, r.Bottom + delta}
}
func (r Rect) Deflate(delta float64) Rect { return r.Inflate(-delta) }
func (r Rect) Intersect(other Rect) Rect {
	return Rect{math.Max(r.Left, other.Left), math.Max(r.Top, other.Top),
		math.Min(r.Right, other.Right), math.Min(r.Bottom, other.Bottom)}
}
func (r Rect) ExpandToInclude(other Rect) Rect {
	return Rect{math.Min(r.Left, other.Left), math.Min(r.Top, other.Top),
		math.Max(r.Right, other.Right), math.Max(r.Bottom, other.Bottom)}
}
func (r Rect) Overlaps(other Rect) bool {
	if r.Right <= other.Left || other.Right <= r.Left {
		return false
	}
	if r.Bottom <= other.Top || other.Bottom <= r.Top {
		return false
	}
	return true
}

func (r Rect) ShortestSide() float64 { return math.Min(math.Abs(r.Width()), math.Abs(r.Height())) }
func (r Rect) LongestSide() float64  { return math.Max(math.Abs(r.Width()), math.Abs(r.Height())) }

func (r Rect) TopLeft() Offset      { return Offset{r.Left, r.Top} }
func (r Rect) TopCenter() Offset    { return Offset{r.Left + r.Width()/2.0, r.Top} }
func (r Rect) TopRight() Offset     { return Offset{r.Right, r.Top} }
func (r Rect) CenterLeft() Offset   { return Offset{r.Left, r.Top + r.Height()/2.0} }
func (r Rect) Center() Offset       { return Offset{r.Left + r.Width()/2.0, r.Top + r.Height()/2.0} }
func (r Rect) CenterRight() Offset  { return Offset{r.Right, r.Top + r.Height()/2.0} }
func (r Rect) BottomLeft() Offset   { return Offset{r.Left, r.Bottom} }
func (r Rect) BottomCenter() Offset { return Offset{r.Left + r.Width()/2.0, r.Bottom} }
func (r Rect) BottomRight() Offset  { return Offset{r.Right, r.Bottom} }

// Contains reports whether offset is inside (right and bottom edges excluded).
func (r Rect) Contains(offset Offset) bool {
	return offset.Dx >= r.Left && offset.Dx < r.Right && offset.Dy >= r.Top && offset.Dy < r.Bottom
}

// RectLerp is Rect.lerp; nil stands for Dart's null.
func RectLerp(a, b *Rect, t float64) *Rect {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		k := 1.0 - t
		return &Rect{a.Left * k, a.Top * k, a.Right * k, a.Bottom * k}
	case a == nil:
		return &Rect{b.Left * t, b.Top * t, b.Right * t, b.Bottom * t}
	}
	return &Rect{lerpDouble(a.Left, b.Left, t), lerpDouble(a.Top, b.Top, t),
		lerpDouble(a.Right, b.Right, t), lerpDouble(a.Bottom, b.Bottom, t)}
}

func (r Rect) String() string {
	return "Rect.fromLTRB(" + fixed1(r.Left) + ", " + fixed1(r.Top) + ", " + fixed1(r.Right) + ", " + fixed1(r.Bottom) + ")"
}

// Radius is a radius for either circular or elliptical shapes.
type Radius struct{ X, Y float64 }

var RadiusZero = Radius{0, 0}

func NewRadiusCircular(radius float64) Radius { return Radius{radius, radius} }
func NewRadiusElliptical(x, y float64) Radius { return Radius{x, y} }

// Clamp is Radius.clamp({minimum, maximum}); nil bounds are unbounded.
func (r Radius) Clamp(minimum, maximum *Radius) Radius {
	lo, hi := Radius{math.Inf(-1), math.Inf(-1)}, Radius{math.Inf(1), math.Inf(1)}
	if minimum != nil {
		lo = *minimum
	}
	if maximum != nil {
		hi = *maximum
	}
	return Radius{ClampDouble(r.X, lo.X, hi.X), ClampDouble(r.Y, lo.Y, hi.Y)}
}

// RadiusClampValuesOptions are Radius.clampValues' named arguments; nil
// bounds are unbounded.
type RadiusClampValuesOptions struct{ MinimumX, MinimumY, MaximumX, MaximumY *float64 }

func (r Radius) ClampValues(o RadiusClampValuesOptions) Radius {
	or := func(p *float64, d float64) float64 {
		if p != nil {
			return *p
		}
		return d
	}
	inf := math.Inf(1)
	return Radius{ClampDouble(r.X, or(o.MinimumX, -inf), or(o.MaximumX, inf)),
		ClampDouble(r.Y, or(o.MinimumY, -inf), or(o.MaximumY, inf))}
}

func (r Radius) Neg() Radius                { return Radius{-r.X, -r.Y} }
func (r Radius) Sub(other Radius) Radius    { return Radius{r.X - other.X, r.Y - other.Y} }
func (r Radius) Add(other Radius) Radius    { return Radius{r.X + other.X, r.Y + other.Y} }
func (r Radius) Mul(operand float64) Radius { return Radius{r.X * operand, r.Y * operand} }
func (r Radius) Div(operand float64) Radius { return Radius{r.X / operand, r.Y / operand} }
func (r Radius) TruncDiv(operand float64) Radius {
	return Radius{dart.TruncDiv(r.X, operand), dart.TruncDiv(r.Y, operand)}
}
func (r Radius) Mod(operand float64) Radius {
	return Radius{dart.Mod(r.X, operand), dart.Mod(r.Y, operand)}
}

// RadiusLerp is Radius.lerp; nil stands for Dart's null.
func RadiusLerp(a, b *Radius, t float64) *Radius {
	switch {
	case a == nil && b == nil:
		return nil
	case b == nil:
		k := 1.0 - t
		return &Radius{a.X * k, a.Y * k}
	case a == nil:
		return &Radius{b.X * t, b.Y * t}
	}
	return &Radius{lerpDouble(a.X, b.X, t), lerpDouble(a.Y, b.Y, t)}
}

func (r Radius) String() string {
	if r.X == r.Y {
		return "Radius.circular(" + fixed1(r.X) + ")"
	}
	return "Radius.elliptical(" + fixed1(r.X) + ", " + fixed1(r.Y) + ")"
}

// rrectLike holds what RRect and RSuperellipse share (Dart's _RRectLike).
type rrectLike struct {
	Left, Top, Right, Bottom float64
	TlRadiusX, TlRadiusY     float64
	TrRadiusX, TrRadiusY     float64
	BrRadiusX, BrRadiusY     float64
	BlRadiusX, BlRadiusY     float64
}

func (r rrectLike) checked() rrectLike {
	if r.TlRadiusX < 0 || r.TlRadiusY < 0 || r.TrRadiusX < 0 || r.TrRadiusY < 0 ||
		r.BrRadiusX < 0 || r.BrRadiusY < 0 || r.BlRadiusX < 0 || r.BlRadiusY < 0 {
		panic("ui: corner radii must be >= 0")
	}
	return r
}

func uniform(left, top, right, bottom, rx, ry float64) rrectLike {
	return rrectLike{left, top, right, bottom, rx, ry, rx, ry, rx, ry, rx, ry}.checked()
}

// RRectCorners are the named corner arguments of fromLTRBAndCorners and
// fromRectAndCorners; unset corners are Radius.zero.
type RRectCorners struct{ TopLeft, TopRight, BottomRight, BottomLeft Radius }

func corners(left, top, right, bottom float64, c RRectCorners) rrectLike {
	return rrectLike{left, top, right, bottom,
		c.TopLeft.X, c.TopLeft.Y, c.TopRight.X, c.TopRight.Y,
		c.BottomRight.X, c.BottomRight.Y, c.BottomLeft.X, c.BottomLeft.Y}.checked()
}

func (r rrectLike) TlRadius() Radius { return Radius{r.TlRadiusX, r.TlRadiusY} }
func (r rrectLike) TrRadius() Radius { return Radius{r.TrRadiusX, r.TrRadiusY} }
func (r rrectLike) BrRadius() Radius { return Radius{r.BrRadiusX, r.BrRadiusY} }
func (r rrectLike) BlRadius() Radius { return Radius{r.BlRadiusX, r.BlRadiusY} }

func (r rrectLike) shift(offset Offset) rrectLike {
	r.Left, r.Top, r.Right, r.Bottom = r.Left+offset.Dx, r.Top+offset.Dy, r.Right+offset.Dx, r.Bottom+offset.Dy
	return r
}

func (r rrectLike) inflate(delta float64) rrectLike {
	g := func(v float64) float64 { return math.Max(0, v+delta) }
	return rrectLike{r.Left - delta, r.Top - delta, r.Right + delta, r.Bottom + delta,
		g(r.TlRadiusX), g(r.TlRadiusY), g(r.TrRadiusX), g(r.TrRadiusY),
		g(r.BrRadiusX), g(r.BrRadiusY), g(r.BlRadiusX), g(r.BlRadiusY)}
}

func (r rrectLike) Width() float64  { return r.Right - r.Left }
func (r rrectLike) Height() float64 { return r.Bottom - r.Top }
func (r rrectLike) OuterRect() Rect { return Rect{r.Left, r.Top, r.Right, r.Bottom} }

func (r rrectLike) SafeInnerRect() Rect {
	const kInsetFactor = 0.29289321881 // 1-cos(pi/4)
	leftRadius := math.Max(r.BlRadiusX, r.TlRadiusX)
	topRadius := math.Max(r.TlRadiusY, r.TrRadiusY)
	rightRadius := math.Max(r.TrRadiusX, r.BrRadiusX)
	bottomRadius := math.Max(r.BrRadiusY, r.BlRadiusY)
	return Rect{r.Left + leftRadius*kInsetFactor, r.Top + topRadius*kInsetFactor,
		r.Right - rightRadius*kInsetFactor, r.Bottom - bottomRadius*kInsetFactor}
}

func (r rrectLike) MiddleRect() Rect {
	leftRadius := math.Max(r.BlRadiusX, r.TlRadiusX)
	topRadius := math.Max(r.TlRadiusY, r.TrRadiusY)
	rightRadius := math.Max(r.TrRadiusX, r.BrRadiusX)
	bottomRadius := math.Max(r.BrRadiusY, r.BlRadiusY)
	return Rect{r.Left + leftRadius, r.Top + topRadius, r.Right - rightRadius, r.Bottom - bottomRadius}
}

func (r rrectLike) WideMiddleRect() Rect {
	topRadius := math.Max(r.TlRadiusY, r.TrRadiusY)
	bottomRadius := math.Max(r.BrRadiusY, r.BlRadiusY)
	return Rect{r.Left, r.Top + topRadius, r.Right, r.Bottom - bottomRadius}
}

func (r rrectLike) TallMiddleRect() Rect {
	leftRadius := math.Max(r.BlRadiusX, r.TlRadiusX)
	rightRadius := math.Max(r.TrRadiusX, r.BrRadiusX)
	return Rect{r.Left + leftRadius, r.Top, r.Right - rightRadius, r.Bottom}
}

func (r rrectLike) IsEmpty() bool { return r.Left >= r.Right || r.Top >= r.Bottom }
func (r rrectLike) IsFinite() bool {
	return isFinite(r.Left) && isFinite(r.Top) && isFinite(r.Right) && isFinite(r.Bottom)
}
func (r rrectLike) IsRect() bool {
	return (r.TlRadiusX == 0.0 || r.TlRadiusY == 0.0) && (r.TrRadiusX == 0.0 || r.TrRadiusY == 0.0) &&
		(r.BlRadiusX == 0.0 || r.BlRadiusY == 0.0) && (r.BrRadiusX == 0.0 || r.BrRadiusY == 0.0)
}
func (r rrectLike) sameCorners() bool {
	return r.TlRadius() == r.TrRadius() && r.TrRadius() == r.BrRadius() && r.BrRadius() == r.BlRadius()
}
func (r rrectLike) IsStadium() bool {
	return r.sameCorners() && (r.Width() <= 2.0*r.TlRadiusX || r.Height() <= 2.0*r.TlRadiusY)
}
func (r rrectLike) IsEllipse() bool {
	return r.sameCorners() && r.Width() <= 2.0*r.TlRadiusX && r.Height() <= 2.0*r.TlRadiusY
}
func (r rrectLike) IsCircle() bool { return r.Width() == r.Height() && r.IsEllipse() }
func (r rrectLike) ShortestSide() float64 {
	return math.Min(math.Abs(r.Width()), math.Abs(r.Height()))
}
func (r rrectLike) LongestSide() float64 { return math.Max(math.Abs(r.Width()), math.Abs(r.Height())) }
func (r rrectLike) HasNaN() bool {
	for _, v := range []float64{r.Left, r.Top, r.Right, r.Bottom, r.TrRadiusX, r.TrRadiusY, r.TlRadiusX,
		r.TlRadiusY, r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY} {
		if math.IsNaN(v) {
			return true
		}
	}
	return false
}
func (r rrectLike) Center() Offset {
	return Offset{r.Left + r.Width()/2.0, r.Top + r.Height()/2.0}
}

func getMin(min, radius1, radius2, limit float64) float64 {
	sum := radius1 + radius2
	if sum > limit && sum != 0.0 {
		return math.Min(min, limit/sum)
	}
	return min
}

func (r rrectLike) scaleRadii() rrectLike {
	scale := 1.0
	scale = getMin(scale, r.BlRadiusY, r.TlRadiusY, r.Height())
	scale = getMin(scale, r.TlRadiusX, r.TrRadiusX, r.Width())
	scale = getMin(scale, r.TrRadiusY, r.BrRadiusY, r.Height())
	scale = getMin(scale, r.BrRadiusX, r.BlRadiusX, r.Width())
	if scale < 1.0 {
		return rrectLike{r.Left, r.Top, r.Right, r.Bottom,
			r.TlRadiusX * scale, r.TlRadiusY * scale, r.TrRadiusX * scale, r.TrRadiusY * scale,
			r.BrRadiusX * scale, r.BrRadiusY * scale, r.BlRadiusX * scale, r.BlRadiusY * scale}
	}
	return r
}

func (r rrectLike) lerpTo(b *rrectLike, t float64) rrectLike {
	if b == nil {
		k := 1.0 - t
		m := func(v float64) float64 { return math.Max(0, v*k) }
		return rrectLike{r.Left * k, r.Top * k, r.Right * k, r.Bottom * k,
			m(r.TlRadiusX), m(r.TlRadiusY), m(r.TrRadiusX), m(r.TrRadiusY),
			m(r.BrRadiusX), m(r.BrRadiusY), m(r.BlRadiusX), m(r.BlRadiusY)}
	}
	l := func(a, b float64) float64 { return lerpDouble(a, b, t) }
	m := func(a, b float64) float64 { return math.Max(0, lerpDouble(a, b, t)) }
	return rrectLike{l(r.Left, b.Left), l(r.Top, b.Top), l(r.Right, b.Right), l(r.Bottom, b.Bottom),
		m(r.TlRadiusX, b.TlRadiusX), m(r.TlRadiusY, b.TlRadiusY), m(r.TrRadiusX, b.TrRadiusX),
		m(r.TrRadiusY, b.TrRadiusY), m(r.BrRadiusX, b.BrRadiusX), m(r.BrRadiusY, b.BrRadiusY),
		m(r.BlRadiusX, b.BlRadiusX), m(r.BlRadiusY, b.BlRadiusY)}
}

func (r rrectLike) toString(className string) string {
	rect := fixed1(r.Left) + ", " + fixed1(r.Top) + ", " + fixed1(r.Right) + ", " + fixed1(r.Bottom)
	if r.sameCorners() {
		if r.TlRadius().X == r.TlRadius().Y {
			return className + ".fromLTRBR(" + rect + ", " + fixed1(r.TlRadius().X) + ")"
		}
		return className + ".fromLTRBXY(" + rect + ", " + fixed1(r.TlRadius().X) + ", " + fixed1(r.TlRadius().Y) + ")"
	}
	return className + ".fromLTRBAndCorners(" + rect + ", topLeft: " + r.TlRadius().String() +
		", topRight: " + r.TrRadius().String() + ", bottomRight: " + r.BrRadius().String() +
		", bottomLeft: " + r.BlRadius().String() + ")"
}

// RRect is an immutable rounded rectangle with custom radii for all four
// corners.
type RRect struct{ rrectLike }

var RRectZero = RRect{}

func NewRRectFromLTRBXY(left, top, right, bottom, radiusX, radiusY float64) RRect {
	return RRect{uniform(left, top, right, bottom, radiusX, radiusY)}
}
func NewRRectFromLTRBR(left, top, right, bottom float64, radius Radius) RRect {
	return RRect{uniform(left, top, right, bottom, radius.X, radius.Y)}
}
func NewRRectFromRectXY(rect Rect, radiusX, radiusY float64) RRect {
	return RRect{uniform(rect.Left, rect.Top, rect.Right, rect.Bottom, radiusX, radiusY)}
}
func NewRRectFromRectAndRadius(rect Rect, radius Radius) RRect {
	return RRect{uniform(rect.Left, rect.Top, rect.Right, rect.Bottom, radius.X, radius.Y)}
}
func NewRRectFromLTRBAndCorners(left, top, right, bottom float64, c RRectCorners) RRect {
	return RRect{corners(left, top, right, bottom, c)}
}
func NewRRectFromRectAndCorners(rect Rect, c RRectCorners) RRect {
	return RRect{corners(rect.Left, rect.Top, rect.Right, rect.Bottom, c)}
}

func (r RRect) Shift(offset Offset) RRect   { return RRect{r.shift(offset)} }
func (r RRect) Inflate(delta float64) RRect { return RRect{r.inflate(delta)} }
func (r RRect) Deflate(delta float64) RRect { return RRect{r.inflate(-delta)} }
func (r RRect) ScaleRadii() RRect           { return RRect{r.scaleRadii()} }

// Contains reports whether point is inside the rounded rectangle.
func (r RRect) Contains(point Offset) bool {
	if point.Dx < r.Left || point.Dx >= r.Right || point.Dy < r.Top || point.Dy >= r.Bottom {
		return false // outside bounding box
	}
	s := r.ScaleRadii()
	var x, y, radiusX, radiusY float64
	switch {
	case point.Dx < r.Left+s.TlRadiusX && point.Dy < r.Top+s.TlRadiusY:
		x, y, radiusX, radiusY = point.Dx-r.Left-s.TlRadiusX, point.Dy-r.Top-s.TlRadiusY, s.TlRadiusX, s.TlRadiusY
	case point.Dx > r.Right-s.TrRadiusX && point.Dy < r.Top+s.TrRadiusY:
		x, y, radiusX, radiusY = point.Dx-r.Right+s.TrRadiusX, point.Dy-r.Top-s.TrRadiusY, s.TrRadiusX, s.TrRadiusY
	case point.Dx > r.Right-s.BrRadiusX && point.Dy > r.Bottom-s.BrRadiusY:
		x, y, radiusX, radiusY = point.Dx-r.Right+s.BrRadiusX, point.Dy-r.Bottom+s.BrRadiusY, s.BrRadiusX, s.BrRadiusY
	case point.Dx < r.Left+s.BlRadiusX && point.Dy > r.Bottom-s.BlRadiusY:
		x, y, radiusX, radiusY = point.Dx-r.Left-s.BlRadiusX, point.Dy-r.Bottom+s.BlRadiusY, s.BlRadiusX, s.BlRadiusY
	default:
		return true // inside and not within the rounded corner area
	}
	x /= radiusX
	y /= radiusY
	return !(x*x+y*y > 1.0)
}

// RRectLerp is RRect.lerp; nil stands for Dart's null.
func RRectLerp(a, b *RRect, t float64) *RRect {
	switch {
	case a == nil && b == nil:
		return nil
	case a == nil:
		return &RRect{b.lerpTo(nil, 1-t)}
	case b == nil:
		return &RRect{a.lerpTo(nil, t)}
	}
	return &RRect{a.lerpTo(&b.rrectLike, t)}
}

func (r RRect) String() string { return r.toString("RRect") }

// RSuperellipse is an immutable rounded superellipse. Its Contains is in
// rsuperellipse_contains.go: a Go port of the engine's C++, because Dart
// calls into the engine for it and the Impeller SDK doesn't expose it.
type RSuperellipse struct{ rrectLike }

var RSuperellipseZero = RSuperellipse{}

func NewRSuperellipseFromLTRBXY(left, top, right, bottom, radiusX, radiusY float64) RSuperellipse {
	return RSuperellipse{uniform(left, top, right, bottom, radiusX, radiusY)}
}
func NewRSuperellipseFromLTRBR(left, top, right, bottom float64, radius Radius) RSuperellipse {
	return RSuperellipse{uniform(left, top, right, bottom, radius.X, radius.Y)}
}
func NewRSuperellipseFromRectXY(rect Rect, radiusX, radiusY float64) RSuperellipse {
	return RSuperellipse{uniform(rect.Left, rect.Top, rect.Right, rect.Bottom, radiusX, radiusY)}
}
func NewRSuperellipseFromRectAndRadius(rect Rect, radius Radius) RSuperellipse {
	return RSuperellipse{uniform(rect.Left, rect.Top, rect.Right, rect.Bottom, radius.X, radius.Y)}
}
func NewRSuperellipseFromLTRBAndCorners(left, top, right, bottom float64, c RRectCorners) RSuperellipse {
	return RSuperellipse{corners(left, top, right, bottom, c)}
}
func NewRSuperellipseFromRectAndCorners(rect Rect, c RRectCorners) RSuperellipse {
	return RSuperellipse{corners(rect.Left, rect.Top, rect.Right, rect.Bottom, c)}
}

func (r RSuperellipse) Shift(offset Offset) RSuperellipse   { return RSuperellipse{r.shift(offset)} }
func (r RSuperellipse) Inflate(delta float64) RSuperellipse { return RSuperellipse{r.inflate(delta)} }
func (r RSuperellipse) Deflate(delta float64) RSuperellipse { return RSuperellipse{r.inflate(-delta)} }
func (r RSuperellipse) ScaleRadii() RSuperellipse           { return RSuperellipse{r.scaleRadii()} }

// RSuperellipseLerp is RSuperellipse.lerp; nil stands for Dart's null.
func RSuperellipseLerp(a, b *RSuperellipse, t float64) *RSuperellipse {
	switch {
	case a == nil && b == nil:
		return nil
	case a == nil:
		return &RSuperellipse{b.lerpTo(nil, 1-t)}
	case b == nil:
		return &RSuperellipse{a.lerpTo(nil, t)}
	}
	return &RSuperellipse{a.lerpTo(&b.rrectLike, t)}
}

func (r RSuperellipse) String() string { return r.toString("RSuperellipse") }

// RSTransform is a rotation-and-uniform-scale plus translation. Dart stores
// it in a Float32List, so its components read back rounded to float32.
type RSTransform struct{ value [4]float32 }

func NewRSTransform(scos, ssin, tx, ty float64) RSTransform {
	return RSTransform{[4]float32{float32(scos), float32(ssin), float32(tx), float32(ty)}}
}

// RSTransformComponents are RSTransform.fromComponents' named arguments.
type RSTransformComponents struct {
	Rotation, Scale, AnchorX, AnchorY, TranslateX, TranslateY float64
}

func NewRSTransformFromComponents(c RSTransformComponents) RSTransform {
	scos := math.Cos(c.Rotation) * c.Scale
	ssin := math.Sin(c.Rotation) * c.Scale
	tx := c.TranslateX + -scos*c.AnchorX + ssin*c.AnchorY
	ty := c.TranslateY + -ssin*c.AnchorX - scos*c.AnchorY
	return NewRSTransform(scos, ssin, tx, ty)
}

func (t RSTransform) Scos() float64 { return float64(t.value[0]) }
func (t RSTransform) Ssin() float64 { return float64(t.value[1]) }
func (t RSTransform) Tx() float64   { return float64(t.value[2]) }
func (t RSTransform) Ty() float64   { return float64(t.value[3]) }
