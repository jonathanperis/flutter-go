package widgets

import r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"

// RepaintBoundary gives its subtree its own layer: changes inside don't
// repaint what's outside, and vice versa.
type RepaintBoundary struct{ Child Widget }

func (w RepaintBoundary) ChildWidget() Widget             { return w.Child }
func (w RepaintBoundary) CreateRenderObject() r.RenderBox { return r.NewRepaintBoundary() }
func (w RepaintBoundary) UpdateRenderObject(r.RenderBox)  {}

// Notifier is Flutter's ChangeNotifier: listeners run on Notify.
type Notifier struct{ listeners []func() }

func (n *Notifier) AddListener(f func()) int {
	n.listeners = append(n.listeners, f)
	return len(n.listeners) - 1
}
func (n *Notifier) RemoveListener(id int) { n.listeners[id] = nil }
func (n *Notifier) Notify() {
	for _, f := range n.listeners {
		if f != nil {
			f()
		}
	}
}

// CustomPainter paints directly on the canvas.
type CustomPainter interface {
	Paint(cv r.Canvas, size r.Size)
}

// CustomPaint calls Painter to paint. If Repaint is set, the painter is
// repainted whenever it notifies, without rebuilding or relayout.
type CustomPaint struct {
	Painter CustomPainter
	Repaint *Notifier
	Size    r.Size // used when there's no child
	Child   Widget
}

func (w CustomPaint) ChildWidget() Widget { return w.Child }
func (w CustomPaint) CreateRenderObject() r.RenderBox {
	ro := &renderCustomPaint{}
	w.UpdateRenderObject(ro)
	return ro
}
func (w CustomPaint) UpdateRenderObject(ro r.RenderBox) {
	c := ro.(*renderCustomPaint)
	if c.repaint != w.Repaint {
		if c.repaint != nil {
			c.repaint.RemoveListener(c.listener)
		}
		if w.Repaint != nil {
			c.listener = w.Repaint.AddListener(func() { r.MarkNeedsPaint(c) })
		}
		c.repaint = w.Repaint
	}
	if c.painter != w.Painter || c.size != w.Size {
		c.painter, c.size = w.Painter, w.Size
		r.MarkNeedsLayout(c)
	}
}

type renderCustomPaint struct {
	r.Box
	r.SingleChild
	painter  CustomPainter
	repaint  *Notifier
	listener int
	size     r.Size
}

func (c *renderCustomPaint) PerformLayout(cs r.Constraints) r.Size {
	if ch := c.Child(); ch != nil {
		return r.Layout(ch, cs)
	}
	return cs.Constrain(c.size)
}

func (c *renderCustomPaint) Paint(cv r.Canvas, offset r.Offset) {
	if c.painter != nil {
		cv.Save()
		cv.Translate(offset.X, offset.Y)
		c.painter.Paint(cv, c.Base().Size)
		cv.Restore()
	}
	if ch := c.Child(); ch != nil {
		r.PaintChild(cv, ch, offset)
	}
}
