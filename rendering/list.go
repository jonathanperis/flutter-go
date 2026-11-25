package rendering

import (
	"fmt"
	"math"
)

// DefaultCacheExtent is how far beyond the visible area items are kept built
// (Flutter's RenderAbstractViewport.defaultCacheExtent).
const DefaultCacheExtent = 250.0

// ChildManager creates and removes list children on demand, during layout.
// The widgets layer implements it by building item widgets.
type ChildManager interface {
	// CreateChild builds item index and returns its render object.
	CreateChild(index int) RenderBox
	RemoveChild(index int)
}

// RenderFixedExtentList is a vertical scrolling viewport whose items all have
// the same height. It is the render half of ListViewBuilder and corresponds
// to Flutter's RenderViewport + RenderSliverFixedExtentList.
//
// Only items within the visible area plus DefaultCacheExtent on each side
// exist; others are created and removed as the scroll offset changes.
type RenderFixedExtentList struct {
	Box
	itemExtent float64
	itemCount  int
	offset     float64
	maxScroll  float64
	manager    ChildManager
	live       map[int]RenderBox
	kids       []RenderBox // live children in index order
	first      int         // index of kids[0]
	// OnScroll is called when the scroll offset changes.
	OnScroll func(offset float64)
	// Input hooks, set by the widgets layer (Flutter's ScrollPosition):
	// Hold stops any scroll animation (a pointer landed) and reports whether
	// one was running; drags report finger deltas and timestamps (seconds);
	// DragEnd gets the raw release velocity and the distance its samples
	// covered; PointerScroll is a mouse wheel delta.
	Hold          func() bool
	DragStart     func(ts float64)
	DragUpdate    func(delta, ts float64)
	DragEnd       func(velocity, distance float64)
	PointerScroll func(delta float64)
}

func NewFixedExtentList(itemExtent float64, itemCount int, m ChildManager) *RenderFixedExtentList {
	return &RenderFixedExtentList{itemExtent: itemExtent, itemCount: itemCount, manager: m, live: map[int]RenderBox{}}
}

func (r *RenderFixedExtentList) Set(itemExtent float64, itemCount int) {
	if itemExtent != r.itemExtent || itemCount != r.itemCount {
		r.itemExtent, r.itemCount = itemExtent, itemCount
		MarkNeedsLayout(r)
	}
}

func (r *RenderFixedExtentList) Children() []RenderBox       { return r.kids }
func (r *RenderFixedExtentList) ScrollOffset() float64       { return r.offset }
func (r *RenderFixedExtentList) MaxScrollExtent() float64    { return r.maxScroll }
func (r *RenderFixedExtentList) HitTestSelf(Offset) bool     { return true }
func (r *RenderFixedExtentList) LiveIndices() (first, n int) { return r.first, len(r.kids) }

// SetScrollOffset sets the offset as is (Flutter's forcePixels); scroll
// physics in the widgets layer decide whether out-of-range is allowed.
func (r *RenderFixedExtentList) SetScrollOffset(v float64) {
	if v != r.offset {
		r.offset = v
		MarkNeedsLayout(r)
		if r.OnScroll != nil {
			r.OnScroll(v)
		}
	}
}

// ScrollBy scrolls by delta pixels (positive moves content up).
func (r *RenderFixedExtentList) ScrollBy(delta float64) { r.SetScrollOffset(r.offset + delta) }

// ReplaceChild swaps the render object of a live item (when its widget
// subtree rebuilt into a different render object).
func (r *RenderFixedExtentList) ReplaceChild(index int, child RenderBox) {
	if old, ok := r.live[index]; ok && old != child {
		r.live[index] = child
		child.Base().parent = r
		r.kids[index-r.first] = child
		MarkNeedsLayout(r)
	}
}

// indexRange returns the inclusive range of items that must exist, following
// RenderSliverFixedExtentBoxAdaptor: from the cache-extended start offset to
// the cache-extended end offset.
func (r *RenderFixedExtentList) indexRange(viewport float64) (int, int) {
	start := math.Max(0, r.offset-DefaultCacheExtent)
	end := r.offset + viewport + DefaultCacheExtent
	first := int(math.Floor(start/r.itemExtent + 1e-10))
	last := int(math.Ceil(end/r.itemExtent-1e-10)) - 1
	return first, min(last, r.itemCount-1)
}

func (r *RenderFixedExtentList) PerformLayout(c Constraints) Size {
	if c.MaxH == Inf || c.MaxW == Inf {
		panic(fmt.Sprintf("RenderFixedExtentList was given unbounded constraints %v; "+
			"put it inside Expanded or a SizedBox (Flutter: \"Vertical viewport was given unbounded height\")", c))
	}
	size := Size{c.MaxW, c.MaxH}
	r.maxScroll = math.Max(0, float64(r.itemCount)*r.itemExtent-size.H)

	first, last := r.indexRange(size.H)
	for i := range r.live {
		if i < first || i > last {
			delete(r.live, i)
			r.manager.RemoveChild(i)
		}
	}
	r.kids = r.kids[:0]
	r.first = first
	for i := first; i <= last; i++ {
		child, ok := r.live[i]
		if !ok {
			if child = r.manager.CreateChild(i); child == nil {
				continue
			}
			child.Base().parent = r
			r.live[i] = child
		}
		r.kids = append(r.kids, child)
		Layout(child, Tight(Size{size.W, r.itemExtent}))
		child.Base().Offset = Offset{0, float64(i)*r.itemExtent - r.offset}
	}
	return size
}

func (r *RenderFixedExtentList) Paint(cv Canvas, offset Offset) {
	cv.Save()
	cv.ClipRect(RectFrom(offset, r.Size))
	// Like RenderSliverMultiBoxAdaptor, paint only children that overlap
	// the visible area; children in the cache extent are laid out, not painted.
	for _, c := range r.kids {
		y := c.Base().Offset.Y
		if y+r.itemExtent > 0 && y < r.Size.H {
			PaintChild(cv, c, offset)
		}
	}
	cv.Restore()
}

// ViewportExtent is the visible main-axis size.
func (r *RenderFixedExtentList) ViewportExtent() float64 { return r.Size.H }
