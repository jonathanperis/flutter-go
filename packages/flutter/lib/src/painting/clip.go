package painting

// Port of clip.dart (Flutter 3.50.1): ClipContext.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// ClipContext is a mixin-like abstract class that provides utility methods
// for clipping: its Canvas getter is abstract. Go: the interface ClipContext
// plus the embeddable ClipContextBase, whose methods reach Canvas through
// self (call SetSelf once, API_MAPPING "Mixins and overridable methods").
type ClipContext interface {
	// Canvas is the canvas on which the clipping and painting happen.
	Canvas() ui.Canvas
	ClipPathAndPaint(path *ui.Path, clipBehavior ui.Clip, bounds ui.Rect, painter func())
	ClipRRectAndPaint(rrect ui.RRect, clipBehavior ui.Clip, bounds ui.Rect, painter func())
	ClipRSuperellipseAndPaint(rse ui.RSuperellipse, clipBehavior ui.Clip, bounds ui.Rect, painter func())
	ClipRectAndPaint(rect ui.Rect, clipBehavior ui.Clip, bounds ui.Rect, painter func())
}

// ClipContextBase holds ClipContext's methods; the embedding type
// implements Canvas.
type ClipContextBase struct {
	self interface{ Canvas() ui.Canvas }
}

// SetSelf points the base at the object embedding it.
func (c *ClipContextBase) SetSelf(self interface{ Canvas() ui.Canvas }) { c.self = self }

func (c *ClipContextBase) canvas() ui.Canvas {
	if c.self == nil {
		panic("painting: ClipContextBase used before SetSelf")
	}
	return c.self.Canvas()
}

func (c *ClipContextBase) clipAndPaint(canvasClipCall func(doAntiAlias bool), clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	c.canvas().Save()
	switch clipBehavior {
	case ui.ClipNone:
	case ui.ClipHardEdge:
		canvasClipCall(false)
	case ui.ClipAntiAlias:
		canvasClipCall(true)
	case ui.ClipAntiAliasWithSaveLayer:
		canvasClipCall(true)
		c.canvas().SaveLayer(&bounds, ui.NewPaint())
	}
	painter()
	if clipBehavior == ui.ClipAntiAliasWithSaveLayer {
		c.canvas().Restore()
	}
	c.canvas().Restore()
}

// ClipPathAndPaint clips the canvas with path according to clipBehavior
// and calls painter; bounds is the saveLayer bounds for
// Clip.antiAliasWithSaveLayer.
func (c *ClipContextBase) ClipPathAndPaint(path *ui.Path, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	c.clipAndPaint(func(doAntiAlias bool) {
		c.canvas().ClipPath(path, ui.CanvasClipOptions{DoAntiAlias: &doAntiAlias})
	}, clipBehavior, bounds, painter)
}

// ClipRRectAndPaint clips the canvas with rrect (see ClipPathAndPaint).
func (c *ClipContextBase) ClipRRectAndPaint(rrect ui.RRect, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	c.clipAndPaint(func(doAntiAlias bool) {
		c.canvas().ClipRRect(rrect, ui.CanvasClipOptions{DoAntiAlias: &doAntiAlias})
	}, clipBehavior, bounds, painter)
}

// ClipRSuperellipseAndPaint clips the canvas with rse (see
// ClipPathAndPaint).
func (c *ClipContextBase) ClipRSuperellipseAndPaint(rse ui.RSuperellipse, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	c.clipAndPaint(func(doAntiAlias bool) {
		c.canvas().ClipRSuperellipse(rse, ui.CanvasClipOptions{DoAntiAlias: &doAntiAlias})
	}, clipBehavior, bounds, painter)
}

// ClipRectAndPaint clips the canvas with rect (see ClipPathAndPaint).
func (c *ClipContextBase) ClipRectAndPaint(rect ui.Rect, clipBehavior ui.Clip, bounds ui.Rect, painter func()) {
	c.clipAndPaint(func(doAntiAlias bool) {
		c.canvas().ClipRect(rect, ui.CanvasClipRectOptions{DoAntiAlias: &doAntiAlias})
	}, clipBehavior, bounds, painter)
}
