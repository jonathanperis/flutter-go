package app

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
)

// NewCanvas returns a rendering.Canvas that draws through dart:ui's Canvas
// (which records an Impeller display list). It needs no GPU.
func NewCanvas(c ui.Canvas) r.Canvas { return &uiCanvas{c: c, paint: ui.NewPaint()} }

// uiCanvas implements rendering.Canvas (flutter-go's interim render-object
// canvas) on dart:ui's Canvas, as Flutter's PaintingContext paints into a
// ui.Canvas.
type uiCanvas struct {
	c     ui.Canvas
	paint *ui.Paint
}

func (c *uiCanvas) Save()                    { c.c.Save() }
func (c *uiCanvas) Restore()                 { c.c.Restore() }
func (c *uiCanvas) Translate(dx, dy float64) { c.c.Translate(dx, dy) }
func (c *uiCanvas) ClipRect(x r.Rect)        { c.c.ClipRect(x) }

func (c *uiCanvas) DrawRect(x r.Rect, col r.Color) {
	c.paint.SetColor(col)
	c.c.DrawRect(x, c.paint)
}

func (c *uiCanvas) DrawRRect(x r.Rect, radius float64, col r.Color) {
	c.paint.SetColor(col)
	c.c.DrawRRect(ui.NewRRectFromRectXY(x, radius, radius), c.paint)
}

// uiRecording records a separate picture (a repaint boundary's layer).
type uiRecording struct {
	rec *ui.PictureRecorder
	c   *uiCanvas
}

func (rc uiRecording) Canvas() r.Canvas { return rc.c }
func (rc uiRecording) End() r.Picture   { return picture{rc.rec.EndRecording()} }

// picture adapts *ui.Picture to rendering.Picture.
type picture struct{ p *ui.Picture }

func (p picture) Release() { p.p.Dispose() }

func (c *uiCanvas) NewRecording() r.Recording {
	rec := ui.NewPictureRecorder()
	return uiRecording{rec, &uiCanvas{c: ui.NewCanvas(rec), paint: c.paint}}
}

func (c *uiCanvas) DrawPicture(p r.Picture, at r.Offset) {
	c.c.Save()
	c.c.Translate(at.Dx, at.Dy)
	c.c.DrawPicture(p.(picture).p)
	c.c.Restore()
}

// DrawParagraph paints a rendering.UIParagraph with its TextPainter.
func (c *uiCanvas) DrawParagraph(p r.Paragraph, at r.Offset) {
	p.(*r.UIParagraph).Painter.Paint(c.c, at)
}
