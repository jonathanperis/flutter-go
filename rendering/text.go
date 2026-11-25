package rendering

import (
	"math"
	"strings"
	"unicode/utf8"
)

type TextStyle struct {
	FontSize float64 // 0 means 14, Flutter's default
	Color    Color
	Bold     bool
}

func (s TextStyle) Size() float64 {
	if s.FontSize == 0 {
		return 14
	}
	return s.FontSize
}

// Paragraph is laid-out text.
type Paragraph interface {
	Text() string
	Height() float64
	MaxIntrinsicWidth() float64 // width of the longest line without wrapping
	Release()
}

// TextEngine shapes and lays out text. The app installs an Impeller-backed
// engine; tests use FixedTextEngine.
type TextEngine interface {
	Layout(text string, style TextStyle, maxWidth float64) Paragraph
}

var textEngine TextEngine = FixedTextEngine{}

// SetTextEngine installs the process-wide text engine.
func SetTextEngine(e TextEngine) { textEngine = e }

// FixedTextEngine mimics Flutter's test font: every glyph is a
// fontSize × fontSize square, and lines break greedily at spaces (a word
// longer than the line breaks between glyphs). Trailing spaces do not count
// toward a line's width.
type FixedTextEngine struct{}

type fixedParagraph struct {
	text          string
	height, width float64
}

func (p fixedParagraph) Text() string               { return p.text }
func (p fixedParagraph) Height() float64            { return p.height }
func (p fixedParagraph) MaxIntrinsicWidth() float64 { return p.width }
func (p fixedParagraph) Release()                   {}

func (FixedTextEngine) Layout(text string, style TextStyle, maxWidth float64) Paragraph {
	size := style.Size()
	words := strings.Split(text, " ")
	full := float64(utf8.RuneCountInString(strings.TrimRight(text, " "))) * size
	perLine := math.Max(1, math.Floor(maxWidth/size)) // glyphs per line
	lines, used := 1, 0.0                             // glyphs used on current line
	for i, w := range words {
		n := float64(utf8.RuneCountInString(w))
		if i > 0 {
			if used+1+n <= perLine { // fits with its leading space
				used += 1 + n
				continue
			}
			lines++
			used = 0
		}
		for n > perLine { // break an over-long word
			lines++
			n -= perLine
		}
		used = n
	}
	return fixedParagraph{text, float64(lines) * size, full}
}

// RenderParagraph displays a single-style run of text.
type RenderParagraph struct {
	Box
	text  string
	style TextStyle
	para  Paragraph
}

func NewParagraph(text string, style TextStyle) *RenderParagraph {
	return &RenderParagraph{text: text, style: style}
}

func (r *RenderParagraph) Children() []RenderBox { return nil }
func (r *RenderParagraph) Text() string          { return r.text }

func (r *RenderParagraph) Set(text string, style TextStyle) {
	if text != r.text || style != r.style {
		r.text, r.style = text, style
		MarkNeedsLayout(r)
	}
}

func (r *RenderParagraph) PerformLayout(c Constraints) Size {
	if r.para != nil {
		r.para.Release()
	}
	r.para = textEngine.Layout(r.text, r.style, c.MaxW)
	// Flutter's TextWidthBasis.parent: unwrapped width clamped to constraints.
	return c.Constrain(Size{r.para.MaxIntrinsicWidth(), r.para.Height()})
}

func (r *RenderParagraph) Paint(cv Canvas, offset Offset) { cv.DrawParagraph(r.para, offset) }
func (r *RenderParagraph) HitTestSelf(Offset) bool        { return true }

func (r *RenderParagraph) Dispose() {
	if r.para != nil {
		r.para.Release()
		r.para = nil
	}
}
