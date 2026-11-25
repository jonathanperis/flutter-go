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
// Drags: the deepest list under the down position scrolls. If a tap
// recognizer competes, the drag is accepted only after moving beyond
// TouchSlop (cancelling the tap) and, as with DragStartBehavior.start,
// begins where it was accepted; otherwise it is accepted on down. Release velocity comes from a port of
// Flutter's VelocityTracker.
type PointerRouter struct {
	owner    *BuildOwner
	down     []*r.RenderTapRegion
	isDown   bool
	list     *r.RenderFixedExtentList
	start    r.Offset
	last     r.Offset
	dragging bool
	tracker  VelocityTracker
	// Now returns the current pointer event's timestamp in seconds.
	Now func() float64
	// Clock returns the sampling clock in seconds (defaults to Now). Flutter
	// separates the two: event timestamps for the fit, a clock to detect a
	// pointer that stopped before lifting.
	Clock func() float64
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
	p.tracker = VelocityTracker{}
	p.record(pos)
	// Gesture arena: with no tap recognizer competing, the drag is the only
	// member and wins at once (Flutter resolves single-member arenas by
	// default), so the whole movement scrolls. With a competitor, the drag
	// must first move beyond TouchSlop.
	if p.list != nil && len(p.down) == 0 {
		p.dragging = true
	}
	if p.list != nil && p.list.Hold != nil && p.list.Hold() {
		// Touching a scrolling list only stops it: like Flutter's
		// IgnorePointer during scroll activities, items don't get the tap.
		p.down = nil
	}
}

func (p *PointerRouter) record(pos r.Offset) {
	if p.Now == nil {
		return
	}
	clock := p.Now
	if p.Clock != nil {
		clock = p.Clock
	}
	p.tracker.AddPosition(p.Now(), clock(), pos.Y)
}

// releaseVelocity is Flutter's VerticalDragGestureRecognizer.considerFling
// followed by ScrollDragController.end: a fling needs more than
// MinFlingVelocity and more than TouchSlop of distance in the samples; the
// velocity is clamped to MaxFlingVelocity and negated (finger up = scroll down).
func (p *PointerRouter) releaseVelocity() float64 {
	if p.Now == nil {
		return 0
	}
	clock := p.Now
	if p.Clock != nil {
		clock = p.Clock
	}
	vy, dy, ok := p.tracker.Estimate(clock())
	if !ok || math.Abs(vy) <= MinFlingVelocity || math.Abs(dy) <= TouchSlop {
		return 0
	}
	return -math.Max(-MaxFlingVelocity, math.Min(MaxFlingVelocity, vy))
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
		// DragStartBehavior.start: the drag begins at the pointer position
		// where it was accepted, so this move itself scrolls nothing.
		p.last = pos
	}
	p.list.ScrollBy(p.last.Y - pos.Y)
	p.last = pos
}

func (p *PointerRouter) PointerUp(pos r.Offset) {
	if pos != p.last && p.isDown {
		p.PointerMove(pos)
	}
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
				if d.OnTapAt != nil {
					d.OnTapAt(pos)
				}
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
