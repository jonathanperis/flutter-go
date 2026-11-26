// Package rendering is the render-tree layer: box layout (constraints go down,
// sizes go up), painting to a Canvas, and hit testing. It is pure Go and never
// touches the GPU directly.
package rendering

import (
	"fmt"
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
)

// Inf is an unbounded extent (Flutter's double.infinity).
var Inf = math.Inf(1)

const epsilon = 1e-9

// Offset, Size and Rect are dart:ui's (rendering re-exports them in Flutter).
// The aliases are transitional: callers should use the ui types.
type (
	Offset = ui.Offset
	Size   = ui.Size
	Rect   = ui.Rect
)

// Color is dart:ui's Color (straight, non-premultiplied alpha).
type Color = ui.Color

// RGB builds an opaque color from a 0xRRGGBB value (Color(0xFFRRGGBB)).
// It is a flutter-go convenience, not Flutter API.
func RGB(hex uint32) Color { return ui.NewColor(int(0xFF000000 | hex)) }

// Constraints are Flutter's BoxConstraints.
type Constraints struct{ MinW, MaxW, MinH, MaxH float64 }

func Tight(s Size) Constraints { return Constraints{s.Width, s.Width, s.Height, s.Height} }
func Loose(s Size) Constraints { return Constraints{0, s.Width, 0, s.Height} }

func (c Constraints) String() string {
	return fmt.Sprintf("BoxConstraints(%g<=w<=%g, %g<=h<=%g)", c.MinW, c.MaxW, c.MinH, c.MaxH)
}

func (c Constraints) Loosen() Constraints { return Constraints{0, c.MaxW, 0, c.MaxH} }

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func (c Constraints) ConstrainW(w float64) float64 { return clamp(w, c.MinW, c.MaxW) }
func (c Constraints) ConstrainH(h float64) float64 { return clamp(h, c.MinH, c.MaxH) }
func (c Constraints) Constrain(s Size) Size {
	return ui.NewSize(c.ConstrainW(s.Width), c.ConstrainH(s.Height))
}
func (c Constraints) Smallest() Size { return ui.NewSize(c.ConstrainW(0), c.ConstrainH(0)) }

func (c Constraints) IsSatisfiedBy(s Size) bool {
	return s.Width >= c.MinW-1e-6 && s.Width <= c.MaxW+1e-6 && s.Height >= c.MinH-1e-6 && s.Height <= c.MaxH+1e-6
}

// Enforce clamps c into the range allowed by outer (Flutter's enforce).
func (c Constraints) Enforce(outer Constraints) Constraints {
	return Constraints{
		clamp(c.MinW, outer.MinW, outer.MaxW), clamp(c.MaxW, outer.MinW, outer.MaxW),
		clamp(c.MinH, outer.MinH, outer.MaxH), clamp(c.MaxH, outer.MinH, outer.MaxH),
	}
}

// Deflate shrinks constraints by insets (Flutter's deflate).
func (c Constraints) Deflate(e painting.EdgeInsets) Constraints {
	h, v := e.Horizontal(), e.Vertical()
	minW, minH := math.Max(0, c.MinW-h), math.Max(0, c.MinH-v)
	return Constraints{minW, math.Max(minW, c.MaxW-h), minH, math.Max(minH, c.MaxH-v)}
}
