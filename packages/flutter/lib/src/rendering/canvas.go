package rendering

import (
	"fmt"
	"strings"
)

// Canvas is what render objects paint to. The app implements it with Impeller;
// tests use Recorder.
type Canvas interface {
	Save()
	Restore()
	Translate(dx, dy float64)
	ClipRect(r Rect)
	DrawRect(r Rect, c Color)
	DrawRRect(r Rect, radius float64, c Color)
	DrawParagraph(p Paragraph, at Offset)
	// NewRecording starts recording a separate Picture.
	NewRecording() Recording
	// DrawPicture replays a recorded picture at an offset.
	DrawPicture(p Picture, at Offset)
}

// PaintChild paints child at its parent-assigned offset relative to offset.
// A child repaint boundary is not painted here; it becomes a child layer.
func PaintChild(cv Canvas, child RenderBox, offset Offset) {
	at := offset.Add(child.Base().Offset)
	if pc, ok := cv.(*paintingContext); ok && isBoundary(child) {
		pc.addChildLayer(child, at)
		return
	}
	child.Paint(cv, at)
}

// Recorder is a Canvas that records commands as text, for tests.
type Recorder struct{ Ops []string }

func (r *Recorder) add(f string, a ...any) { r.Ops = append(r.Ops, fmt.Sprintf(f, a...)) }

func (r *Recorder) Save()                    { r.add("save") }
func (r *Recorder) Restore()                 { r.add("restore") }
func (r *Recorder) Translate(dx, dy float64) { r.add("translate(%g, %g)", dx, dy) }
func (r *Recorder) ClipRect(x Rect)          { r.add("clipRect(%g, %g, %g, %g)", x.X, x.Y, x.W, x.H) }
func (r *Recorder) DrawRect(x Rect, c Color) {
	r.add("rect(%g, %g, %g, %g) %s", x.X, x.Y, x.W, x.H, hex(c))
}
func (r *Recorder) DrawRRect(x Rect, rad float64, c Color) {
	r.add("rrect(%g, %g, %g, %g, r=%g) %s", x.X, x.Y, x.W, x.H, rad, hex(c))
}
func (r *Recorder) DrawParagraph(p Paragraph, at Offset) {
	r.add("text(%q at %g, %g)", p.Text(), at.X, at.Y)
}
func (r *Recorder) String() string { return strings.Join(r.Ops, "\n") }

// Recorder pictures are recorders; drawing one replays its ops.
func (r *Recorder) Release() {}

type recorderRecording struct{ r *Recorder }

func (rr recorderRecording) Canvas() Canvas { return rr.r }
func (rr recorderRecording) End() Picture   { return rr.r }

func (r *Recorder) NewRecording() Recording { return recorderRecording{&Recorder{}} }

func (r *Recorder) DrawPicture(p Picture, at Offset) {
	ops := p.(*Recorder).Ops
	if len(ops) == 0 {
		return
	}
	if at != (Offset{}) {
		r.add("save")
		r.add("translate(%g, %g)", at.X, at.Y)
	}
	r.Ops = append(r.Ops, ops...)
	if at != (Offset{}) {
		r.add("restore")
	}
}

func hex(c Color) string {
	return fmt.Sprintf("#%02x%02x%02x", int(c.R*255+0.5), int(c.G*255+0.5), int(c.B*255+0.5))
}
