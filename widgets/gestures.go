package widgets

import (
	"math"

	r "github.com/jonathanperis/flutter-go/rendering"
)

// TouchSlop is how far a pointer must move before a drag starts
// (Flutter's kTouchSlop).
const TouchSlop = 18.0

// PointerRouter turns raw pointer events into taps, drags and scrolls.
//
// Taps: like Flutter's gesture arena for nested tap recognizers, the deepest
// GestureDetector under both the down and the up position wins.
// Drags: a vertical move beyond TouchSlop scrolls the deepest list under the
// down position and cancels the tap. As with DragStartBehavior.start, the
// slop distance itself is not applied to the scroll.
type PointerRouter struct {
	owner    *BuildOwner
	down     []*r.RenderTapRegion
	isDown   bool
	list     *r.RenderFixedExtentList
	start    r.Offset
	last     r.Offset
	dragging bool
}

func NewPointerRouter(o *BuildOwner) *PointerRouter { return &PointerRouter{owner: o} }

func (p *PointerRouter) hit(pos r.Offset) (taps []*r.RenderTapRegion, list *r.RenderFixedExtentList) {
	for _, ro := range r.HitTest(p.owner.RenderRoot(), pos) {
		switch t := ro.(type) {
		case *r.RenderTapRegion:
			taps = append(taps, t)
		case *r.RenderFixedExtentList:
			if list == nil {
				list = t
			}
		}
	}
	return taps, list
}

func (p *PointerRouter) PointerDown(pos r.Offset) {
	p.down, p.list = p.hit(pos)
	p.isDown, p.dragging = true, false
	p.start, p.last = pos, pos
}

func (p *PointerRouter) PointerMove(pos r.Offset) {
	if !p.isDown || p.list == nil {
		return
	}
	if !p.dragging {
		if math.Abs(pos.Y-p.start.Y) <= TouchSlop {
			return
		}
		p.dragging = true
		// Drag starts where the slop was exceeded.
		p.last = r.Offset{X: pos.X, Y: p.start.Y + math.Copysign(TouchSlop, pos.Y-p.start.Y)}
	}
	p.list.ScrollBy(p.last.Y - pos.Y)
	p.last = pos
}

func (p *PointerRouter) PointerUp(pos r.Offset) {
	p.PointerMove(pos)
	down, dragging := p.down, p.dragging
	p.down, p.isDown, p.list, p.dragging = nil, false, nil, false
	if dragging {
		return // a drag is not a tap
	}
	up, _ := p.hit(pos)
	for _, d := range down { // deepest first
		for _, u := range up {
			if d == u {
				if d.OnTap != nil {
					d.OnTap()
				}
				return
			}
		}
	}
}

// Scroll applies a scroll-wheel delta (positive scrolls content up) to the
// deepest list under pos.
func (p *PointerRouter) Scroll(pos r.Offset, dy float64) {
	if _, list := p.hit(pos); list != nil {
		list.ScrollBy(dy)
	}
}
