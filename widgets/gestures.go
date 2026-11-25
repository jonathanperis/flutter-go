package widgets

import (
	"math"

	r "github.com/jonathanperis/flutter-go/rendering"
)

// Fling velocity limits (Flutter's kMinFlingVelocity and kMaxFlingVelocity).
const (
	MinFlingVelocity = 50.0
	MaxFlingVelocity = 8000.0
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
	// recent pointer positions during a drag, for release velocity
	samples []sample
	// Now returns the current time in seconds (for velocity estimation).
	Now func() float64
}

type sample struct{ t, y float64 }

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
	p.samples = p.samples[:0]
	p.record(pos)
	if p.list != nil && p.list.Hold != nil && p.list.Hold() {
		// Touching a scrolling list only stops it: like Flutter's
		// IgnorePointer during scroll activities, items don't get the tap.
		p.down = nil
	}
}

func (p *PointerRouter) record(pos r.Offset) {
	if p.Now != nil {
		p.samples = append(p.samples, sample{p.Now(), pos.Y})
	}
}

// releaseVelocity estimates scroll velocity (px/s) from the last 100ms of
// samples with a straight-line fit. Flutter's VelocityTracker uses a
// least-squares polynomial; this simpler estimate is not Flutter-exact.
func (p *PointerRouter) releaseVelocity() float64 {
	if len(p.samples) < 2 {
		return 0
	}
	last := p.samples[len(p.samples)-1]
	first := last
	for i := len(p.samples) - 1; i >= 0 && last.t-p.samples[i].t <= 0.1; i-- {
		first = p.samples[i]
	}
	if last.t-first.t < 1e-3 {
		return 0
	}
	v := -(last.y - first.y) / (last.t - first.t) // finger up = content up = positive
	if math.Abs(v) < MinFlingVelocity {
		return 0
	}
	return math.Max(-MaxFlingVelocity, math.Min(MaxFlingVelocity, v))
}

func (p *PointerRouter) PointerMove(pos r.Offset) {
	if !p.isDown || p.list == nil {
		return
	}
	p.record(pos)
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
	down, dragging, list := p.down, p.dragging, p.list
	p.down, p.isDown, p.list, p.dragging = nil, false, nil, false
	if dragging {
		if v := p.releaseVelocity(); v != 0 && list.Fling != nil {
			list.Fling(v)
		}
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
