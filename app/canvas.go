package app

import (
	"math"

	"github.com/jonathanperis/flutter-go/textedit"

	"github.com/jonathanperis/flutter-go/impeller"
	r "github.com/jonathanperis/flutter-go/rendering"
)

// NewCanvas returns a rendering.Canvas that records into b. It needs no GPU.
func NewCanvas(b *impeller.DisplayListBuilder, p *impeller.Paint) r.Canvas {
	return &impellerCanvas{b, p}
}

// NewTextEngine returns an Impeller-backed rendering.TextEngine.
func NewTextEngine(tc *impeller.TypographyContext) r.TextEngine { return impellerText{tc} }

// impellerCanvas implements rendering.Canvas on an Impeller display list.
type impellerCanvas struct {
	b     *impeller.DisplayListBuilder
	paint *impeller.Paint
}

func color(c r.Color) impeller.Color {
	return impeller.Color{R: float32(c.R), G: float32(c.G), B: float32(c.B), A: float32(c.A)}
}

func rect(x r.Rect) impeller.Rect {
	return impeller.Rect{X: float32(x.X), Y: float32(x.Y), W: float32(x.W), H: float32(x.H)}
}

func (c *impellerCanvas) Save()                    { c.b.Save() }
func (c *impellerCanvas) Restore()                 { c.b.Restore() }
func (c *impellerCanvas) Translate(dx, dy float64) { c.b.Translate(float32(dx), float32(dy)) }
func (c *impellerCanvas) ClipRect(x r.Rect)        { c.b.ClipRect(rect(x)) }

func (c *impellerCanvas) DrawRect(x r.Rect, col r.Color) {
	c.paint.SetColor(color(col))
	c.b.DrawRect(rect(x), c.paint)
}

func (c *impellerCanvas) DrawRRect(x r.Rect, radius float64, col r.Color) {
	c.paint.SetColor(color(col))
	c.b.DrawRoundedRect(rect(x), float32(radius), c.paint)
}

type impellerRecording struct {
	b *impeller.DisplayListBuilder
	c *impellerCanvas
}

func (r impellerRecording) Canvas() r.Canvas { return r.c }
func (r impellerRecording) End() r.Picture {
	dl := r.b.Build()
	r.b.Release()
	return dl
}

func (c *impellerCanvas) NewRecording() r.Recording {
	b := impeller.NewDisplayListBuilder()
	return impellerRecording{b, &impellerCanvas{b, c.paint}}
}

func (c *impellerCanvas) DrawPicture(p r.Picture, at r.Offset) {
	c.b.Save()
	c.b.Translate(float32(at.X), float32(at.Y))
	c.b.DrawDisplayList(p.(*impeller.DisplayList), 1)
	c.b.Restore()
}

func (c *impellerCanvas) DrawParagraph(p r.Paragraph, at r.Offset) {
	c.b.DrawParagraph(p.(*paragraph).p, float32(at.X), float32(at.Y))
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
	n := textedit.Len16(p.text)
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
	n := textedit.Len16(p.text)
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
