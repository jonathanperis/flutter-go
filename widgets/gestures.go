package widgets

import r "github.com/jonathanperis/flutter-go/rendering"

// PointerRouter turns pointer down/up into taps. Like Flutter's gesture arena
// for nested tap recognizers, the deepest GestureDetector under both the down
// and the up position wins.
type PointerRouter struct {
	owner *BuildOwner
	down  []*r.RenderTapRegion
}

func NewPointerRouter(o *BuildOwner) *PointerRouter { return &PointerRouter{owner: o} }

func (p *PointerRouter) tapRegions(pos r.Offset) []*r.RenderTapRegion {
	var out []*r.RenderTapRegion
	for _, ro := range r.HitTest(p.owner.RenderRoot(), pos) {
		if t, ok := ro.(*r.RenderTapRegion); ok {
			out = append(out, t)
		}
	}
	return out
}

func (p *PointerRouter) PointerDown(pos r.Offset) { p.down = p.tapRegions(pos) }

func (p *PointerRouter) PointerUp(pos r.Offset) {
	up := p.tapRegions(pos)
	down := p.down
	p.down = nil
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
