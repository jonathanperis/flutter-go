package widgets

import (
	"math"
	"sort"
	"time"

	"github.com/jonathanperis/flutter-go/animation"

	r "github.com/jonathanperis/flutter-go/rendering"
)

// ListViewBuilder is a vertical, lazily built list of fixed-height items
// (Flutter's ListView.builder with itemExtent). ItemBuilder is called only
// for items within the visible area plus the cache extent. Each item gets a
// RepaintBoundary, as Flutter's addRepaintBoundaries does, so scrolling
// moves items without repainting them.
type ListViewBuilder struct {
	ItemCount   int
	ItemExtent  float64
	ItemBuilder func(ctx *Element, index int) Widget
	Controller  *ScrollController
}

// ScrollController reads and sets a list's scroll offset.
type ScrollController struct {
	list    *r.RenderFixedExtentList
	initial float64
	sched   *animation.Scheduler
	anim    *animation.Controller
	// DevicePixelRatio sets the fling velocity tolerance (Flutter's
	// toleranceFor). flutter_test uses 3; 0 means 1.
	DevicePixelRatio float64
}

// AnimateTo scrolls to offset over d with curve, like Flutter's
// ScrollPosition.animateTo (a DrivenScrollActivity on an unbounded
// controller). The offset is clamped each frame.
func (c *ScrollController) AnimateTo(offset float64, d time.Duration, curve animation.Curve) {
	if c.list == nil || c.sched == nil {
		c.JumpTo(offset)
		return
	}
	if c.anim != nil {
		c.anim.Dispose()
	}
	c.anim = animation.NewUnboundedController(c.sched, c.list.ScrollOffset())
	a := c.anim
	a.AddListener(func() { c.list.SetScrollOffset(a.Value()) })
	a.AnimateTo(offset, d, curve)
}

// Fling starts a ballistic scroll at velocity (logical px/s, positive scrolls
// content up), using Flutter's ClampingScrollPhysics: below the velocity
// tolerance nothing happens, and the fling stops when it reaches an edge.
func (c *ScrollController) Fling(velocity float64) {
	if c.list == nil || c.sched == nil {
		return
	}
	if c.anim != nil {
		c.anim.Dispose()
		c.anim = nil
	}
	pos, max := c.list.ScrollOffset(), c.list.MaxScrollExtent()
	tolerance := 1.0 / (0.050 * c.dpr())
	if math.Abs(velocity) < tolerance || (velocity > 0 && pos >= max) || (velocity < 0 && pos <= 0) {
		return
	}
	a := animation.NewUnboundedController(c.sched, pos)
	c.anim = a
	a.AddListener(func() {
		v := a.Value()
		c.list.SetScrollOffset(v)
		// Flutter's BallisticScrollActivity goes idle when setPixels overscrolls.
		if v < 0 || v > c.list.MaxScrollExtent() {
			a.Stop()
		}
	})
	a.AnimateWith(animation.NewClampingScrollSimulation(pos, velocity))
}

func (c *ScrollController) dpr() float64 {
	if c.DevicePixelRatio > 0 {
		return c.DevicePixelRatio
	}
	return 1
}

// IsAnimating reports whether an AnimateTo is in progress.
func (c *ScrollController) IsAnimating() bool { return c.anim != nil && c.anim.IsAnimating() }

// NewScrollController starts at initialOffset (clamped on first layout).
func NewScrollController(initialOffset float64) *ScrollController {
	return &ScrollController{initial: initialOffset}
}

func (c *ScrollController) attach(l *r.RenderFixedExtentList) {
	if c.list == nil {
		l.SetScrollOffset(c.initial)
	} else {
		l.SetScrollOffset(c.list.ScrollOffset())
	}
	c.list = l
}

func (c *ScrollController) Offset() float64 {
	if c.list == nil {
		return c.initial
	}
	return c.list.ScrollOffset()
}

func (c *ScrollController) MaxScrollExtent() float64 {
	if c.list == nil {
		return 0
	}
	return c.list.MaxScrollExtent()
}

// JumpTo moves to offset immediately, clamped to the scrollable range.
func (c *ScrollController) JumpTo(offset float64) {
	if c.list == nil {
		c.initial = offset
		return
	}
	c.list.SetScrollOffset(offset)
}

// listManager implements rendering.ChildManager for a lazy element.
type listManager struct{ e *Element }

func (m listManager) CreateChild(i int) r.RenderBox {
	e := m.e
	w := e.widget.(ListViewBuilder)
	if i < 0 || i >= w.ItemCount {
		return nil
	}
	child := e.updateChild(nil, RepaintBoundary{Child: w.ItemBuilder(e, i)})
	if child == nil {
		return nil
	}
	e.lazy[i] = child
	e.syncLazyChildren()
	return childRender(child)
}

func (m listManager) RemoveChild(i int) {
	e := m.e
	if c, ok := e.lazy[i]; ok {
		c.unmount()
		delete(e.lazy, i)
		e.syncLazyChildren()
	}
}

func (e *Element) mountLazy() {
	w := e.widget.(ListViewBuilder)
	e.lazy = map[int]*Element{}
	list := r.NewFixedExtentList(w.ItemExtent, w.ItemCount, listManager{e})
	e.render = list
	e.attachController(w.Controller)
}

// attachController connects c (or an internal controller when c is nil,
// as Flutter's Scrollable does) to the list and its fling/hold hooks.
func (e *Element) attachController(c *ScrollController) {
	if c == nil {
		if e.ownScroll == nil {
			e.ownScroll = NewScrollController(0)
		}
		c = e.ownScroll
	}
	list := e.render.(*r.RenderFixedExtentList)
	c.sched = e.Scheduler()
	if c.DevicePixelRatio == 0 {
		c.DevicePixelRatio = e.owner.DevicePixelRatio
	}
	c.attach(list)
	list.Fling = c.Fling
	list.Hold = c.Hold
}

// Hold stops any running scroll animation and reports whether one was
// running.
func (c *ScrollController) Hold() bool {
	if c.anim != nil && c.anim.IsAnimating() {
		c.anim.Stop()
		return true
	}
	return false
}

// updateLazy applies a new ListViewBuilder: live items are rebuilt with the
// new builder (as Flutter's SliverMultiBoxAdaptorElement does).
func (e *Element) updateLazy(old ListViewBuilder) {
	w := e.widget.(ListViewBuilder)
	list := e.render.(*r.RenderFixedExtentList)
	if w.Controller != old.Controller {
		e.attachController(w.Controller)
	}
	list.Set(w.ItemExtent, w.ItemCount)
	for _, i := range e.lazyIndices() {
		if i >= w.ItemCount {
			continue // removed by the next layout
		}
		e.lazy[i] = e.updateChild(e.lazy[i], RepaintBoundary{Child: w.ItemBuilder(e, i)})
	}
	e.syncLazyChildren()
	e.syncLazy()
	r.MarkNeedsLayout(list)
}

func (e *Element) lazyIndices() []int {
	idx := make([]int, 0, len(e.lazy))
	for i := range e.lazy {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	return idx
}

// syncLazyChildren keeps e.children (used by Walk and unmount) in index order.
func (e *Element) syncLazyChildren() {
	e.children = e.children[:0]
	for _, i := range e.lazyIndices() {
		e.children = append(e.children, e.lazy[i])
	}
}

// syncLazy hands rebuilt items' render objects to the list.
func (e *Element) syncLazy() {
	list := e.render.(*r.RenderFixedExtentList)
	for i, c := range e.lazy {
		if ro := childRender(c); ro != nil {
			list.ReplaceChild(i, ro)
		}
	}
}

// LazyIndex returns the item index of a direct child of a ListViewBuilder
// element, or -1.
func (e *Element) LazyIndex(child *Element) int {
	for i, c := range e.lazy {
		if c == child {
			return i
		}
	}
	return -1
}
