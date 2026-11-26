package widgets

import (
	"sort"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
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
	if old := e.scrollCtl; old != nil && old != c {
		old.goIdle() // the old position is disposed with its activity
	}
	e.scrollCtl = c
	list := e.render.(*r.RenderFixedExtentList)
	c.vsync = e.Vsync()
	if c.DevicePixelRatio == 0 {
		c.DevicePixelRatio = e.owner.DevicePixelRatio
	}
	c.attach(list)
	c.platform = e.owner.Platform
	list.Hold = c.Hold
	list.DragStart = c.dragStart
	list.DragUpdate = c.dragUpdate
	list.DragEnd = c.dragEnd
	list.PointerScroll = c.pointerScroll
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
