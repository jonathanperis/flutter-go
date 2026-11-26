package painting

// Port of alignment.dart (Flutter 3.50.1): AlignmentGeometry, Alignment,
// AlignmentDirectional, the private _MixedAlignment and TextAlignVertical.
//
// AlignmentGeometry is an interface whose private accessors (Dart's _x,
// _start, _y, which keep it from being implemented outside the library)
// are unexported methods. Alignment and AlignmentDirectional are comparable
// value structs; their operators return their own types (Dart's covariant
// overrides). The operators AlignmentGeometry declares abstractly are the
// functions AlignmentGeometryNeg/Mul/Div/TruncDiv/Mod for a value of the
// interface type. Dart's `==` (equal _x, _start and _y, whatever the
// concrete type) is Equal; Go's == also compares the dynamic type.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AlignmentGeometry is the base type of Alignment, AlignmentDirectional,
// FractionalOffset and the mixed alignments their sums give: a point
// within a rectangle, possibly depending on a TextDirection.
type AlignmentGeometry interface {
	// Add returns the sum of two alignments (Dart's add); the result is a
	// mixed alignment unless both have the same kind.
	Add(other AlignmentGeometry) AlignmentGeometry
	// Resolve converts this alignment to an Alignment for the given
	// direction (nil is Dart's null; directional alignments panic with a
	// FlutterError then).
	Resolve(direction *ui.TextDirection) Alignment
	// Equal is Dart's ==: the same horizontal, start and vertical
	// components, whatever the concrete types.
	Equal(other AlignmentGeometry) bool
	String() string

	gx() float64
	gstart() float64
	gy() float64
	neg() AlignmentGeometry
	mul(other float64) AlignmentGeometry
	div(other float64) AlignmentGeometry
	truncDiv(other float64) AlignmentGeometry
	mod(other float64) AlignmentGeometry
}

// NewAlignmentGeometryXY is AlignmentGeometry.xy: an Alignment.
func NewAlignmentGeometryXY(x, y float64) AlignmentGeometry { return Alignment{x, y} }

// NewAlignmentGeometryDirectional is AlignmentGeometry.directional: an
// AlignmentDirectional.
func NewAlignmentGeometryDirectional(start, y float64) AlignmentGeometry {
	return AlignmentDirectional{start, y}
}

// AlignmentGeometry's static constants.
var (
	AlignmentGeometryTopLeft      AlignmentGeometry = AlignmentTopLeft
	AlignmentGeometryTopCenter    AlignmentGeometry = AlignmentTopCenter
	AlignmentGeometryTopRight     AlignmentGeometry = AlignmentTopRight
	AlignmentGeometryTopStart     AlignmentGeometry = AlignmentDirectionalTopStart
	AlignmentGeometryTopEnd       AlignmentGeometry = AlignmentDirectionalTopEnd
	AlignmentGeometryCenterLeft   AlignmentGeometry = AlignmentCenterLeft
	AlignmentGeometryCenter       AlignmentGeometry = AlignmentCenter
	AlignmentGeometryCenterRight  AlignmentGeometry = AlignmentCenterRight
	AlignmentGeometryCenterStart  AlignmentGeometry = AlignmentDirectionalCenterStart
	AlignmentGeometryCenterEnd    AlignmentGeometry = AlignmentDirectionalCenterEnd
	AlignmentGeometryBottomLeft   AlignmentGeometry = AlignmentBottomLeft
	AlignmentGeometryBottomCenter AlignmentGeometry = AlignmentBottomCenter
	AlignmentGeometryBottomRight  AlignmentGeometry = AlignmentBottomRight
	AlignmentGeometryBottomStart  AlignmentGeometry = AlignmentDirectionalBottomStart
	AlignmentGeometryBottomEnd    AlignmentGeometry = AlignmentDirectionalBottomEnd
)

// geometryAdd is AlignmentGeometry.add's base implementation.
func geometryAdd(a, other AlignmentGeometry) AlignmentGeometry {
	return mixedAlignment{a.gx() + other.gx(), a.gstart() + other.gstart(), a.gy() + other.gy()}
}

func geometryEqual(a, other AlignmentGeometry) bool {
	return other != nil && other.gx() == a.gx() && other.gstart() == a.gstart() && other.gy() == a.gy()
}

// geometryString is AlignmentGeometry.toString.
func geometryString(a AlignmentGeometry) string {
	if a.gstart() == 0.0 {
		return alignmentStringify(a.gx(), a.gy())
	}
	if a.gx() == 0.0 {
		return alignmentDirectionalStringify(a.gstart(), a.gy())
	}
	return alignmentStringify(a.gx(), a.gy()) + " + " + alignmentDirectionalStringify(a.gstart(), 0.0)
}

// AlignmentGeometryNeg is unary `-` on an AlignmentGeometry.
func AlignmentGeometryNeg(a AlignmentGeometry) AlignmentGeometry { return a.neg() }

// AlignmentGeometryMul is `*` on an AlignmentGeometry.
func AlignmentGeometryMul(a AlignmentGeometry, other float64) AlignmentGeometry { return a.mul(other) }

// AlignmentGeometryDiv is `/` on an AlignmentGeometry.
func AlignmentGeometryDiv(a AlignmentGeometry, other float64) AlignmentGeometry { return a.div(other) }

// AlignmentGeometryTruncDiv is `~/` on an AlignmentGeometry (it panics
// with dart.UnsupportedError where Dart throws).
func AlignmentGeometryTruncDiv(a AlignmentGeometry, other float64) AlignmentGeometry {
	return a.truncDiv(other)
}

// AlignmentGeometryMod is `%` on an AlignmentGeometry.
func AlignmentGeometryMod(a AlignmentGeometry, other float64) AlignmentGeometry { return a.mod(other) }

// identicalGeometry stands for Dart's identical(a, b) on alignments: Go
// values have no identity, so equal values of the same type (what Dart's
// canonicalized const instances are) count as identical.
func identicalGeometry(a, b AlignmentGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a == b
}

// asAlignment reports whether g is an Alignment in Dart's sense (`is
// Alignment`, which includes FractionalOffset).
func asAlignment(g AlignmentGeometry) (Alignment, bool) {
	switch v := g.(type) {
	case Alignment:
		return v, true
	case FractionalOffset:
		return v.Alignment, true
	}
	return Alignment{}, false
}

// AlignmentGeometryLerp linearly interpolates between two alignments (nil
// is Dart's null; the result is nil only when both are).
func AlignmentGeometryLerp(a, b AlignmentGeometry, t float64) AlignmentGeometry {
	if identicalGeometry(a, b) {
		return a
	}
	if a == nil {
		return b.mul(t)
	}
	if b == nil {
		return a.mul(1.0 - t)
	}
	aa, aok := a.(AlignmentLike)
	ba, bok := b.(AlignmentLike)
	if aok && bok {
		return AlignmentLerp(aa, ba, t)
	}
	ad, aok := a.(AlignmentDirectional)
	bd, bok := b.(AlignmentDirectional)
	if aok && bok {
		return *AlignmentDirectionalLerp(&ad, &bd, t)
	}
	return mixedAlignment{lerpD(a.gx(), b.gx(), t), lerpD(a.gstart(), b.gstart(), t), lerpD(a.gy(), b.gy(), t)}
}

// lerpD is ui.lerpDouble(a, b, t)! for non-null a and b.
func lerpD(a, b, t float64) float64 { return *ui.LerpDouble(&a, &b, t) }

// AlignmentLike is Dart's static type Alignment, which includes its
// subclass FractionalOffset: Alignment and FractionalOffset implement it.
type AlignmentLike interface {
	AlignmentGeometry
	// AsAlignment is the value as an Alignment (for a FractionalOffset,
	// its embedded Alignment).
	AsAlignment() Alignment
}

// Alignment is a point within a rectangle: (0, 0) is the center, x and y
// of -1.0 the left and top edges, 1.0 the right and bottom edges. The zero
// value is Alignment.center.
type Alignment struct {
	// X is the distance fraction in the horizontal direction.
	X float64
	// Y is the distance fraction in the vertical direction.
	Y float64
}

// NewAlignment is Alignment(x, y).
func NewAlignment(x, y float64) Alignment { return Alignment{x, y} }

// Alignment's static constants.
var (
	AlignmentTopLeft      = Alignment{-1.0, -1.0}
	AlignmentTopCenter    = Alignment{0.0, -1.0}
	AlignmentTopRight     = Alignment{1.0, -1.0}
	AlignmentCenterLeft   = Alignment{-1.0, 0.0}
	AlignmentCenter       = Alignment{0.0, 0.0}
	AlignmentCenterRight  = Alignment{1.0, 0.0}
	AlignmentBottomLeft   = Alignment{-1.0, 1.0}
	AlignmentBottomCenter = Alignment{0.0, 1.0}
	AlignmentBottomRight  = Alignment{1.0, 1.0}
)

func (a Alignment) gx() float64     { return a.X }
func (a Alignment) gstart() float64 { return 0.0 }
func (a Alignment) gy() float64     { return a.Y }

// AsAlignment returns a.
func (a Alignment) AsAlignment() Alignment { return a }

// Add returns the sum of a and other: an Alignment when other is one
// (FractionalOffset included), a mixed alignment otherwise.
func (a Alignment) Add(other AlignmentGeometry) AlignmentGeometry {
	if o, ok := asAlignment(other); ok {
		return a.Plus(o)
	}
	return geometryAdd(a, other)
}

// Sub is the binary `-` operator.
func (a Alignment) Sub(other AlignmentLike) Alignment {
	o := other.AsAlignment()
	return Alignment{a.X - o.X, a.Y - o.Y}
}

// Plus is the binary `+` operator (Go's Add is AlignmentGeometry.add).
func (a Alignment) Plus(other AlignmentLike) Alignment {
	o := other.AsAlignment()
	return Alignment{a.X + o.X, a.Y + o.Y}
}

// Neg is unary `-`.
func (a Alignment) Neg() Alignment { return Alignment{-a.X, -a.Y} }

// Mul is `*`.
func (a Alignment) Mul(other float64) Alignment { return Alignment{a.X * other, a.Y * other} }

// Div is `/`.
func (a Alignment) Div(other float64) Alignment { return Alignment{a.X / other, a.Y / other} }

// TruncDiv is `~/` (each component truncated, converted back to double).
func (a Alignment) TruncDiv(other float64) Alignment {
	return Alignment{dart.TruncDiv(a.X, other), dart.TruncDiv(a.Y, other)}
}

// Mod is `%`.
func (a Alignment) Mod(other float64) Alignment {
	return Alignment{dart.Mod(a.X, other), dart.Mod(a.Y, other)}
}

func (a Alignment) neg() AlignmentGeometry          { return a.Neg() }
func (a Alignment) mul(o float64) AlignmentGeometry { return a.Mul(o) }
func (a Alignment) div(o float64) AlignmentGeometry { return a.Div(o) }
func (a Alignment) truncDiv(o float64) AlignmentGeometry {
	return a.TruncDiv(o)
}
func (a Alignment) mod(o float64) AlignmentGeometry { return a.Mod(o) }

// AlongOffset returns the offset that is this fraction in the direction
// of the given offset.
func (a Alignment) AlongOffset(other ui.Offset) ui.Offset {
	centerX := other.Dx / 2.0
	centerY := other.Dy / 2.0
	return ui.NewOffset(centerX+float64(a.X*centerX), centerY+float64(a.Y*centerY))
}

// AlongSize returns the offset that is this fraction within the given size.
func (a Alignment) AlongSize(other ui.Size) ui.Offset {
	centerX := other.Width / 2.0
	centerY := other.Height / 2.0
	return ui.NewOffset(centerX+float64(a.X*centerX), centerY+float64(a.Y*centerY))
}

// WithinRect returns the point that is this fraction within the given rect.
func (a Alignment) WithinRect(rect ui.Rect) ui.Offset {
	halfWidth := rect.Width() / 2.0
	halfHeight := rect.Height() / 2.0
	return ui.NewOffset(rect.Left+halfWidth+float64(a.X*halfWidth), rect.Top+halfHeight+float64(a.Y*halfHeight))
}

// Inscribe returns a rect of the given size, aligned within given rect as
// specified by this alignment.
func (a Alignment) Inscribe(size ui.Size, rect ui.Rect) ui.Rect {
	halfWidthDelta := (rect.Width() - size.Width) / 2.0
	halfHeightDelta := (rect.Height() - size.Height) / 2.0
	return ui.NewRectFromLTWH(
		rect.Left+halfWidthDelta+float64(a.X*halfWidthDelta),
		rect.Top+halfHeightDelta+float64(a.Y*halfHeightDelta),
		size.Width,
		size.Height,
	)
}

// AlignmentLerp linearly interpolates between two alignments (nil is
// Dart's null). As in Dart, identical inputs return the first as is (a
// FractionalOffset stays one); otherwise the result is an Alignment.
func AlignmentLerp(a, b AlignmentLike, t float64) AlignmentLike {
	if a == nil && b == nil {
		return nil
	}
	if a != nil && b != nil && a == b {
		return a
	}
	if a == nil {
		bv := b.AsAlignment()
		return Alignment{lerpD(0.0, bv.X, t), lerpD(0.0, bv.Y, t)}
	}
	av := a.AsAlignment()
	if b == nil {
		return Alignment{lerpD(av.X, 0.0, t), lerpD(av.Y, 0.0, t)}
	}
	bv := b.AsAlignment()
	return Alignment{lerpD(av.X, bv.X, t), lerpD(av.Y, bv.Y, t)}
}

// Resolve returns a (an Alignment needs no direction).
func (a Alignment) Resolve(direction *ui.TextDirection) Alignment { return a }

// Equal is Dart's ==.
func (a Alignment) Equal(other AlignmentGeometry) bool { return geometryEqual(a, other) }

func alignmentStringify(x, y float64) string {
	switch {
	case x == -1.0 && y == -1.0:
		return "Alignment.topLeft"
	case x == 0.0 && y == -1.0:
		return "Alignment.topCenter"
	case x == 1.0 && y == -1.0:
		return "Alignment.topRight"
	case x == -1.0 && y == 0.0:
		return "Alignment.centerLeft"
	case x == 0.0 && y == 0.0:
		return "Alignment.center"
	case x == 1.0 && y == 0.0:
		return "Alignment.centerRight"
	case x == -1.0 && y == 1.0:
		return "Alignment.bottomLeft"
	case x == 0.0 && y == 1.0:
		return "Alignment.bottomCenter"
	case x == 1.0 && y == 1.0:
		return "Alignment.bottomRight"
	}
	return "Alignment(" + dart.ToStringAsFixed(x, 1) + ", " + dart.ToStringAsFixed(y, 1) + ")"
}

func (a Alignment) String() string { return alignmentStringify(a.X, a.Y) }

// AlignmentDirectional is an offset that is expressed as a fraction of a
// Size, but whose horizontal component is dependent on the writing
// direction: Start -1.0 is the start edge (left in ltr, right in rtl).
type AlignmentDirectional struct {
	// Start is the distance fraction in the horizontal direction.
	Start float64
	// Y is the distance fraction in the vertical direction.
	Y float64
}

// NewAlignmentDirectional is AlignmentDirectional(start, y).
func NewAlignmentDirectional(start, y float64) AlignmentDirectional {
	return AlignmentDirectional{start, y}
}

// AlignmentDirectional's static constants.
var (
	AlignmentDirectionalTopStart     = AlignmentDirectional{-1.0, -1.0}
	AlignmentDirectionalTopCenter    = AlignmentDirectional{0.0, -1.0}
	AlignmentDirectionalTopEnd       = AlignmentDirectional{1.0, -1.0}
	AlignmentDirectionalCenterStart  = AlignmentDirectional{-1.0, 0.0}
	AlignmentDirectionalCenter       = AlignmentDirectional{0.0, 0.0}
	AlignmentDirectionalCenterEnd    = AlignmentDirectional{1.0, 0.0}
	AlignmentDirectionalBottomStart  = AlignmentDirectional{-1.0, 1.0}
	AlignmentDirectionalBottomCenter = AlignmentDirectional{0.0, 1.0}
	AlignmentDirectionalBottomEnd    = AlignmentDirectional{1.0, 1.0}
)

func (a AlignmentDirectional) gx() float64     { return 0.0 }
func (a AlignmentDirectional) gstart() float64 { return a.Start }
func (a AlignmentDirectional) gy() float64     { return a.Y }

// Add returns the sum of a and other: an AlignmentDirectional when other
// is one, a mixed alignment otherwise.
func (a AlignmentDirectional) Add(other AlignmentGeometry) AlignmentGeometry {
	if o, ok := other.(AlignmentDirectional); ok {
		return a.Plus(o)
	}
	return geometryAdd(a, other)
}

// Sub is the binary `-` operator.
func (a AlignmentDirectional) Sub(other AlignmentDirectional) AlignmentDirectional {
	return AlignmentDirectional{a.Start - other.Start, a.Y - other.Y}
}

// Plus is the binary `+` operator.
func (a AlignmentDirectional) Plus(other AlignmentDirectional) AlignmentDirectional {
	return AlignmentDirectional{a.Start + other.Start, a.Y + other.Y}
}

// Neg is unary `-`.
func (a AlignmentDirectional) Neg() AlignmentDirectional { return AlignmentDirectional{-a.Start, -a.Y} }

// Mul is `*`.
func (a AlignmentDirectional) Mul(other float64) AlignmentDirectional {
	return AlignmentDirectional{a.Start * other, a.Y * other}
}

// Div is `/`.
func (a AlignmentDirectional) Div(other float64) AlignmentDirectional {
	return AlignmentDirectional{a.Start / other, a.Y / other}
}

// TruncDiv is `~/`.
func (a AlignmentDirectional) TruncDiv(other float64) AlignmentDirectional {
	return AlignmentDirectional{dart.TruncDiv(a.Start, other), dart.TruncDiv(a.Y, other)}
}

// Mod is `%`.
func (a AlignmentDirectional) Mod(other float64) AlignmentDirectional {
	return AlignmentDirectional{dart.Mod(a.Start, other), dart.Mod(a.Y, other)}
}

func (a AlignmentDirectional) neg() AlignmentGeometry          { return a.Neg() }
func (a AlignmentDirectional) mul(o float64) AlignmentGeometry { return a.Mul(o) }
func (a AlignmentDirectional) div(o float64) AlignmentGeometry { return a.Div(o) }
func (a AlignmentDirectional) truncDiv(o float64) AlignmentGeometry {
	return a.TruncDiv(o)
}
func (a AlignmentDirectional) mod(o float64) AlignmentGeometry { return a.Mod(o) }

// AlignmentDirectionalLerp linearly interpolates between two directional
// alignments (nil is Dart's null).
func AlignmentDirectionalLerp(a, b *AlignmentDirectional, t float64) *AlignmentDirectional {
	if a == b || a != nil && b != nil && *a == *b {
		return a
	}
	var r AlignmentDirectional
	switch {
	case a == nil:
		r = AlignmentDirectional{lerpD(0.0, b.Start, t), lerpD(0.0, b.Y, t)}
	case b == nil:
		r = AlignmentDirectional{lerpD(a.Start, 0.0, t), lerpD(a.Y, 0.0, t)}
	default:
		r = AlignmentDirectional{lerpD(a.Start, b.Start, t), lerpD(a.Y, b.Y, t)}
	}
	return &r
}

// Resolve converts a to an Alignment for the given direction; nil panics
// with Flutter's "No TextDirection found." FlutterError.
func (a AlignmentDirectional) Resolve(direction *ui.TextDirection) Alignment {
	checkCanResolveTextDirection(direction, "AlignmentDirectional")
	if *direction == ui.TextDirectionRtl {
		return Alignment{-a.Start, a.Y}
	}
	return Alignment{a.Start, a.Y}
}

// Equal is Dart's ==.
func (a AlignmentDirectional) Equal(other AlignmentGeometry) bool { return geometryEqual(a, other) }

func alignmentDirectionalStringify(start, y float64) string {
	switch {
	case start == -1.0 && y == -1.0:
		return "AlignmentDirectional.topStart"
	case start == 0.0 && y == -1.0:
		return "AlignmentDirectional.topCenter"
	case start == 1.0 && y == -1.0:
		return "AlignmentDirectional.topEnd"
	case start == -1.0 && y == 0.0:
		return "AlignmentDirectional.centerStart"
	case start == 0.0 && y == 0.0:
		return "AlignmentDirectional.center"
	case start == 1.0 && y == 0.0:
		return "AlignmentDirectional.centerEnd"
	case start == -1.0 && y == 1.0:
		return "AlignmentDirectional.bottomStart"
	case start == 0.0 && y == 1.0:
		return "AlignmentDirectional.bottomCenter"
	case start == 1.0 && y == 1.0:
		return "AlignmentDirectional.bottomEnd"
	}
	return "AlignmentDirectional(" + dart.ToStringAsFixed(start, 1) + ", " + dart.ToStringAsFixed(y, 1) + ")"
}

func (a AlignmentDirectional) String() string { return alignmentDirectionalStringify(a.Start, a.Y) }

// mixedAlignment is Dart's _MixedAlignment: the sum of an Alignment and an
// AlignmentDirectional.
type mixedAlignment struct{ x, start, y float64 }

func (m mixedAlignment) gx() float64     { return m.x }
func (m mixedAlignment) gstart() float64 { return m.start }
func (m mixedAlignment) gy() float64     { return m.y }

func (m mixedAlignment) Add(other AlignmentGeometry) AlignmentGeometry { return geometryAdd(m, other) }
func (m mixedAlignment) Equal(other AlignmentGeometry) bool            { return geometryEqual(m, other) }
func (m mixedAlignment) String() string                                { return geometryString(m) }

func (m mixedAlignment) neg() AlignmentGeometry { return mixedAlignment{-m.x, -m.start, -m.y} }
func (m mixedAlignment) mul(o float64) AlignmentGeometry {
	return mixedAlignment{m.x * o, m.start * o, m.y * o}
}
func (m mixedAlignment) div(o float64) AlignmentGeometry {
	return mixedAlignment{m.x / o, m.start / o, m.y / o}
}
func (m mixedAlignment) truncDiv(o float64) AlignmentGeometry {
	return mixedAlignment{dart.TruncDiv(m.x, o), dart.TruncDiv(m.start, o), dart.TruncDiv(m.y, o)}
}
func (m mixedAlignment) mod(o float64) AlignmentGeometry {
	return mixedAlignment{dart.Mod(m.x, o), dart.Mod(m.start, o), dart.Mod(m.y, o)}
}

func (m mixedAlignment) Resolve(direction *ui.TextDirection) Alignment {
	checkCanResolveTextDirection(direction, "_MixedAlignment")
	if *direction == ui.TextDirectionRtl {
		return Alignment{m.x - m.start, m.y}
	}
	return Alignment{m.x + m.start, m.y}
}

// checkCanResolveTextDirection is debug.dart's
// debugCheckCanResolveTextDirection (asserts are always on in Go).
// TODO(painting debug.dart port): route through DebugCheckCanResolveTextDirection.
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

// TextAlignVertical is the vertical alignment of text within an input
// box: Y from -1.0 (top) to 1.0 (bottom).
type TextAlignVertical struct {
	// Y is a value ranging from -1.0 to 1.0 that defines the topmost and
	// bottommost locations of the top and bottom of the input box.
	Y float64
}

// NewTextAlignVertical is TextAlignVertical(y:); it panics unless
// -1.0 <= y <= 1.0, as Dart asserts.
func NewTextAlignVertical(y float64) TextAlignVertical {
	if !(y >= -1.0 && y <= 1.0) {
		panic(dart.NewAssertionError("y >= -1.0 && y <= 1.0"))
	}
	return TextAlignVertical{y}
}

// TextAlignVertical's static constants.
var (
	TextAlignVerticalTop    = TextAlignVertical{-1.0}
	TextAlignVerticalCenter = TextAlignVertical{0.0}
	TextAlignVerticalBottom = TextAlignVertical{1.0}
)

func (v TextAlignVertical) String() string {
	return foundation.ObjectRuntimeType(v, "TextAlignVertical") + "(y: " + dart.DoubleToString(v.Y) + ")"
}
