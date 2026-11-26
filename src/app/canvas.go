package app

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"

	r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// NewCanvas returns a rendering.Canvas that draws through dart:ui's Canvas
// (which records an Impeller display list). It needs no GPU.
func NewCanvas(c ui.Canvas) r.Canvas { return &uiCanvas{c: c, paint: ui.NewPaint()} }

// NewTextEngine returns an Impeller-backed rendering.TextEngine.
func NewTextEngine(tc *impeller.TypographyContext) r.TextEngine { return impellerText{tc} }

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

func (c *uiCanvas) DrawParagraph(p r.Paragraph, at r.Offset) {
	ui.EngineDrawParagraph(c.c, p.(*paragraph).p, at)
}

func color(c r.Color) impeller.Color {
	return impeller.Color{R: float32(c.R), G: float32(c.G), B: float32(c.B), A: float32(c.A)}
}

// impellerText implements rendering.TextEngine with Impeller's typography.
type impellerText struct{ tc *impeller.TypographyContext }

type paragraph struct {
	text string
	p    *impeller.Paragraph
}

func (p *paragraph) Text() string               { return p.text }
func (p *paragraph) Height() float64            { return float64(p.p.Height()) }
func (p *paragraph) MaxIntrinsicWidth() float64 { return float64(p.p.MaxIntrinsicWidth()) }
func (p *paragraph) Release()                   { p.p.Release() }

func (p *paragraph) CaretX(off int) float64 {
	n := services.Len16(p.text)
	if n == 0 {
		return 0
	}
	if off >= n {
		c := p.p.ClusterAt(n - 1)
		return float64(c.Bounds.X + c.Bounds.W)
	}
	return float64(p.p.ClusterAt(off).Bounds.X)
}

func (p *paragraph) OffsetAtX(x float64) int {
	n := services.Len16(p.text)
	if n == 0 || x <= 0 {
		return 0
	}
	c := p.p.ClusterAtPoint(x, float64(p.p.Height())/2)
	if !c.OK {
		return n
	}
	if x < float64(c.Bounds.X+c.Bounds.W/2) {
		return c.Begin
	}
	return c.End
}

func (t impellerText) Layout(text string, s r.TextStyle, maxWidth float64) r.Paragraph {
	w := float32(math.Min(maxWidth, 1e6))
	p, err := impeller.NewStyledParagraph(t.tc, text, float32(s.Size()), color(s.TextColor()), s.Bold, w)
	if err != nil {
		panic(err)
	}
	return &paragraph{text, p}
}
