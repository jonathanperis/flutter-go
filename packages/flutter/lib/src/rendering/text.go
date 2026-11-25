package rendering

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
)

type TextStyle struct {
	FontSize float64 // 0 means 14, Flutter's default
	Color    Color   // the zero value means opaque black, not transparent
	Bold     bool
}

// Merge returns base (s) overridden by the set fields of o. Go has no
// "unset" for bool, so Bold can be turned on by o but not off.
func (s TextStyle) Merge(o TextStyle) TextStyle {
	if o.FontSize != 0 {
		s.FontSize = o.FontSize
	}
	if o.Color != (Color{}) {
		s.Color = o.Color
	}
	s.Bold = s.Bold || o.Bold
	return s
}

// TextColor returns the color to draw with: an unset (zero) Color would be
// fully transparent, so it means opaque black instead.
func (s TextStyle) TextColor() Color {
	if s.Color == (Color{}) {
		return Color{0, 0, 0, 1}
	}
	return s.Color
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
	// CaretX returns the x position of a caret before UTF-16 offset off.
	CaretX(off int) float64
	// OffsetAtX returns the UTF-16 caret offset nearest to x (single line).
	OffsetAtX(x float64) int
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
	size          float64
}

func (p fixedParagraph) Text() string               { return p.text }
func (p fixedParagraph) Height() float64            { return p.height }
func (p fixedParagraph) MaxIntrinsicWidth() float64 { return p.width }
func (p fixedParagraph) Release()                   {}

// In the test font every grapheme cluster is one square glyph.
func (p fixedParagraph) CaretX(off int) float64 {
	stops := services.GraphemeStops(p.text)
	n := 0
	for i, st := range stops {
		if st <= off {
			n = i
		}
	}
	return float64(n) * p.size
}

func (p fixedParagraph) OffsetAtX(x float64) int {
	stops := services.GraphemeStops(p.text)
	i := int(math.Round(x / p.size))
	return stops[max(0, min(i, len(stops)-1))]
}

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
	return fixedParagraph{text, float64(lines) * size, full, size}
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
	return c.Constrain(ui.NewSize(r.para.MaxIntrinsicWidth(), r.para.Height()))
}

func (r *RenderParagraph) Paint(cv Canvas, offset Offset) { cv.DrawParagraph(r.para, offset) }
func (r *RenderParagraph) HitTestSelf(Offset) bool        { return true }

func (r *RenderParagraph) Dispose() {
	if r.para != nil {
		r.para.Release()
		r.para = nil
	}
}
