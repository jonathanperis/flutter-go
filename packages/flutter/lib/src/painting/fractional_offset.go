package painting

// Port of fractional_offset.dart (Flutter 3.50.1).
//
// FractionalOffset extends Alignment in Dart: here it embeds it (the
// stored X and Y are Alignment's, -1..1, computed as Dart's constructor
// does), and overrides the operators. Both implement AlignmentLike, Go's
// stand-in for Dart's static type Alignment.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// FractionalOffset is an offset that's expressed as a fraction of a Size:
// (0, 0) is the top left, (1, 1) the bottom right.
type FractionalOffset struct {
	Alignment
}

// NewFractionalOffset is FractionalOffset(dx, dy).
func NewFractionalOffset(dx, dy float64) FractionalOffset {
	return FractionalOffset{Alignment{float64(dx*2.0) - 1.0, float64(dy*2.0) - 1.0}}
}

// NewFractionalOffsetFromOffsetAndSize is FractionalOffset.fromOffsetAndSize.
func NewFractionalOffsetFromOffsetAndSize(offset ui.Offset, size ui.Size) FractionalOffset {
	return NewFractionalOffset(offset.Dx/size.Width, offset.Dy/size.Height)
}

// NewFractionalOffsetFromOffsetAndRect is FractionalOffset.fromOffsetAndRect.
func NewFractionalOffsetFromOffsetAndRect(offset ui.Offset, rect ui.Rect) FractionalOffset {
	return NewFractionalOffsetFromOffsetAndSize(offset.Sub(rect.TopLeft()), rect.Size())
}

// Dx is the distance fraction in the horizontal direction.
func (f FractionalOffset) Dx() float64 { return (f.X + 1.0) / 2.0 }

// Dy is the distance fraction in the vertical direction.
func (f FractionalOffset) Dy() float64 { return (f.Y + 1.0) / 2.0 }

// FractionalOffset's static constants.
var (
	FractionalOffsetTopLeft      = NewFractionalOffset(0.0, 0.0)
	FractionalOffsetTopCenter    = NewFractionalOffset(0.5, 0.0)
	FractionalOffsetTopRight     = NewFractionalOffset(1.0, 0.0)
	FractionalOffsetCenterLeft   = NewFractionalOffset(0.0, 0.5)
	FractionalOffsetCenter       = NewFractionalOffset(0.5, 0.5)
	FractionalOffsetCenterRight  = NewFractionalOffset(1.0, 0.5)
	FractionalOffsetBottomLeft   = NewFractionalOffset(0.0, 1.0)
	FractionalOffsetBottomCenter = NewFractionalOffset(0.5, 1.0)
	FractionalOffsetBottomRight  = NewFractionalOffset(1.0, 1.0)
)

// Add is Alignment.add as inherited: `this + other` (FractionalOffset's +)
// when other is an Alignment, a mixed alignment otherwise.
func (f FractionalOffset) Add(other AlignmentGeometry) AlignmentGeometry {
	if o, ok := other.(AlignmentLike); ok {
		return f.Plus(o)
	}
	return geometryAdd(f, other)
}

// Sub is the binary `-`: a FractionalOffset when other is one, an
// Alignment otherwise.
func (f FractionalOffset) Sub(other AlignmentLike) AlignmentLike {
	o, ok := other.(FractionalOffset)
	if !ok {
		return f.Alignment.Sub(other)
	}
	return NewFractionalOffset(f.Dx()-o.Dx(), f.Dy()-o.Dy())
}

// Plus is the binary `+`: a FractionalOffset when other is one, an
// Alignment otherwise.
func (f FractionalOffset) Plus(other AlignmentLike) AlignmentLike {
	o, ok := other.(FractionalOffset)
	if !ok {
		return f.Alignment.Plus(other)
	}
	return NewFractionalOffset(f.Dx()+o.Dx(), f.Dy()+o.Dy())
}

// Neg is unary `-`.
func (f FractionalOffset) Neg() FractionalOffset { return NewFractionalOffset(-f.Dx(), -f.Dy()) }

// Mul is `*`.
func (f FractionalOffset) Mul(other float64) FractionalOffset {
	return NewFractionalOffset(f.Dx()*other, f.Dy()*other)
}

// Div is `/`.
func (f FractionalOffset) Div(other float64) FractionalOffset {
	return NewFractionalOffset(f.Dx()/other, f.Dy()/other)
}

// TruncDiv is `~/`.
func (f FractionalOffset) TruncDiv(other float64) FractionalOffset {
	return NewFractionalOffset(dart.TruncDiv(f.Dx(), other), dart.TruncDiv(f.Dy(), other))
}

// Mod is `%`.
func (f FractionalOffset) Mod(other float64) FractionalOffset {
	return NewFractionalOffset(dart.Mod(f.Dx(), other), dart.Mod(f.Dy(), other))
}

func (f FractionalOffset) neg() AlignmentGeometry          { return f.Neg() }
func (f FractionalOffset) mul(o float64) AlignmentGeometry { return f.Mul(o) }
func (f FractionalOffset) div(o float64) AlignmentGeometry { return f.Div(o) }
func (f FractionalOffset) truncDiv(o float64) AlignmentGeometry {
	return f.TruncDiv(o)
}
func (f FractionalOffset) mod(o float64) AlignmentGeometry { return f.Mod(o) }

// AsAlignment returns the embedded Alignment.
func (f FractionalOffset) AsAlignment() Alignment { return f.Alignment }

// Equal is Dart's ==.
func (f FractionalOffset) Equal(other AlignmentGeometry) bool { return geometryEqual(f, other) }

// Resolve returns the embedded Alignment (Dart returns this, typed
// Alignment).
func (f FractionalOffset) Resolve(direction *ui.TextDirection) Alignment { return f.Alignment }

// FractionalOffsetLerp linearly interpolates between two fractional
// offsets (nil is Dart's null, the center for the other end).
func FractionalOffsetLerp(a, b *FractionalOffset, t float64) *FractionalOffset {
	if a == b || a != nil && b != nil && *a == *b {
		return a
	}
	var r FractionalOffset
	switch {
	case a == nil:
		r = NewFractionalOffset(lerpD(0.5, b.Dx(), t), lerpD(0.5, b.Dy(), t))
	case b == nil:
		r = NewFractionalOffset(lerpD(a.Dx(), 0.5, t), lerpD(a.Dy(), 0.5, t))
	default:
		r = NewFractionalOffset(lerpD(a.Dx(), b.Dx(), t), lerpD(a.Dy(), b.Dy(), t))
	}
	return &r
}

func (f FractionalOffset) String() string {
	return "FractionalOffset(" + dart.ToStringAsFixed(f.Dx(), 1) + ", " + dart.ToStringAsFixed(f.Dy(), 1) + ")"
}
