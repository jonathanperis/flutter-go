package painting

// Port of notched_shapes.dart (Flutter 3.50.1): NotchedShape,
// CircularNotchedRectangle and AutomaticNotchedShape.

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// NotchedShape is a shape with a notch in its outline (used by a bottom
// app bar around a floating action button). guest nil is Dart's null.
type NotchedShape interface {
	// GetOuterPath creates a path for the outer edge of the host with a
	// notch for guest.
	GetOuterPath(host ui.Rect, guest *ui.Rect) *ui.Path
}

// CircularNotchedRectangle is a rectangle with a smooth circular notch
// (on the top edge, or the bottom one when Inverted).
type CircularNotchedRectangle struct {
	// Inverted puts the notch on the bottom edge.
	Inverted bool
}

// GetOuterPath is the host rectangle with a circular notch fitting guest
// (a circle: its width is the diameter).
func (c CircularNotchedRectangle) GetOuterPath(host ui.Rect, guest *ui.Rect) *ui.Path {
	if guest == nil || !host.Overlaps(*guest) {
		p := ui.NewPath()
		p.AddRect(host)
		return p
	}
	g := *guest
	r := g.Width() / 2.0
	notchRadius := ui.NewRadiusCircular(r)
	invertMultiplier := 1.0
	if c.Inverted {
		invertMultiplier = -1.0
	}
	const s1 = 15.0
	const s2 = 1.0
	a := -r - s2
	edge := host.Top
	if c.Inverted {
		edge = host.Bottom
	}
	b := edge - g.Center().Dy

	n2 := math.Sqrt(b * b * r * r * (float64(a*a) + float64(b*b) - float64(r*r)))
	p2xA := (float64(a*r*r) - n2) / (float64(a*a) + float64(b*b))
	p2xB := (float64(a*r*r) + n2) / (float64(a*a) + float64(b*b))
	p2yA := float64(math.Sqrt(float64(r*r)-float64(p2xA*p2xA)) * invertMultiplier)
	p2yB := float64(math.Sqrt(float64(r*r)-float64(p2xB*p2xB)) * invertMultiplier)

	p := make([]ui.Offset, 6)
	p[0] = ui.NewOffset(a-s1, b)
	p[1] = ui.NewOffset(a, b)
	cmp := 1.0
	if b < 0 {
		cmp = -1.0
	}
	if cmp*p2yA > cmp*p2yB {
		p[2] = ui.NewOffset(p2xA, p2yA)
	} else {
		p[2] = ui.NewOffset(p2xB, p2yB)
	}
	p[3] = ui.NewOffset(-1.0*p[2].Dx, p[2].Dy)
	p[4] = ui.NewOffset(-1.0*p[1].Dx, p[1].Dy)
	p[5] = ui.NewOffset(-1.0*p[0].Dx, p[0].Dy)
	for i := range p {
		p[i] = p[i].Add(g.Center())
	}

	clockwise := false
	path := ui.NewPath()
	path.MoveTo(host.Left, host.Top)
	if !c.Inverted {
		path.LineTo(p[0].Dx, p[0].Dy)
		path.QuadraticBezierTo(p[1].Dx, p[1].Dy, p[2].Dx, p[2].Dy)
		path.ArcToPoint(p[3], ui.ArcToPointOptions{Radius: notchRadius, Clockwise: &clockwise})
		path.QuadraticBezierTo(p[4].Dx, p[4].Dy, p[5].Dx, p[5].Dy)
		path.LineTo(host.Right, host.Top)
		path.LineTo(host.Right, host.Bottom)
		path.LineTo(host.Left, host.Bottom)
	} else {
		path.LineTo(host.Right, host.Top)
		path.LineTo(host.Right, host.Bottom)
		path.LineTo(p[5].Dx, p[5].Dy)
		path.QuadraticBezierTo(p[4].Dx, p[4].Dy, p[3].Dx, p[3].Dy)
		path.ArcToPoint(p[2], ui.ArcToPointOptions{Radius: notchRadius, Clockwise: &clockwise})
		path.QuadraticBezierTo(p[1].Dx, p[1].Dy, p[0].Dx, p[0].Dy)
		path.LineTo(host.Left, host.Bottom)
	}
	path.Close()
	return path
}

// AutomaticNotchedShape is a NotchedShape built from two ShapeBorders:
// the host's outer path minus the guest's (Guest nil: no notch).
type AutomaticNotchedShape struct {
	Host  ShapeBorder
	Guest ShapeBorder
}

// NewAutomaticNotchedShape is AutomaticNotchedShape(host, [guest]).
func NewAutomaticNotchedShape(host ShapeBorder, guest ...ShapeBorder) AutomaticNotchedShape {
	s := AutomaticNotchedShape{Host: host}
	if len(guest) > 0 {
		s.Guest = guest[0]
	}
	return s
}

// GetOuterPath is the host's outer path, minus the guest's when both the
// guest and guestRect are given (Path.combine, an ENGINE GAP: it panics
// with UnimplementedError until PathOps is ported).
func (s AutomaticNotchedShape) GetOuterPath(hostRect ui.Rect, guestRect *ui.Rect) *ui.Path {
	hostPath := s.Host.GetOuterPath(hostRect, nil)
	if s.Guest != nil && guestRect != nil {
		guestPath := s.Guest.GetOuterPath(*guestRect, nil)
		return ui.PathCombine(ui.PathOperationDifference, hostPath, guestPath)
	}
	return hostPath
}
