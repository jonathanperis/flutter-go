package rendering

import ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"

// Repaint boundaries and retained layers, after Flutter's PaintingContext,
// OffsetLayer and PictureLayer.
//
// A repaint boundary records its subtree into a Layer: a sequence of
// recorded picture segments and references to child boundaries (with the
// clips and translation active where they sit). Compositing a frame walks
// the layer tree and reuses every boundary whose subtree didn't change, so
// painting work is proportional to what changed, not to the tree size.

// Picture is a recorded, immutable sequence of drawing commands.
type Picture interface{ Release() }

// Recording records drawing commands into a Picture.
type Recording interface {
	Canvas() Canvas
	End() Picture
}

// RepaintBoundary is implemented by render objects that get their own layer.
type RepaintBoundary interface{ IsRepaintBoundary() bool }

func isBoundary(r RenderBox) bool {
	b, ok := r.(RepaintBoundary)
	return ok && b.IsRepaintBoundary()
}

// Layer is a boundary's retained painting.
type Layer struct{ items []layerItem }

type layerItem struct {
	pic   Picture   // a recorded segment, or…
	child RenderBox // …a child repaint boundary
	at    Offset    // child position in this layer's coordinates
	clips []Rect    // clips active at the child, in this layer's coordinates
}

func (l *Layer) release() {
	if l == nil {
		return
	}
	for _, it := range l.items {
		if it.pic != nil {
			it.pic.Release()
		}
	}
}

// op is a state-changing canvas call, kept so a new segment can restore the
// save/translate/clip state after a child layer splits the recording.
type op struct {
	clip   bool
	r      Rect
	dx, dy float64
}

// paintingContext is the Canvas a boundary paints into. It forwards drawing
// to the current segment and splits segments around child boundaries.
type paintingContext struct {
	frame  Canvas // used to create recordings
	layer  *Layer
	rec    Recording
	cur    Canvas
	levels [][]op // ops per save level; levels[0] is the base
}

func newPaintingContext(frame Canvas) *paintingContext {
	pc := &paintingContext{frame: frame, layer: &Layer{}, levels: [][]op{nil}}
	pc.startSegment()
	return pc
}

func (pc *paintingContext) startSegment() {
	pc.rec = pc.frame.NewRecording()
	pc.cur = pc.rec.Canvas()
	for i, level := range pc.levels { // replay the current state
		if i > 0 {
			pc.cur.Save()
		}
		for _, o := range level {
			if o.clip {
				pc.cur.ClipRect(o.r)
			} else {
				pc.cur.Translate(o.dx, o.dy)
			}
		}
	}
}

func (pc *paintingContext) endSegment() {
	pc.layer.items = append(pc.layer.items, layerItem{pic: pc.rec.End()})
}

// state returns the current translation and clips in layer coordinates.
func (pc *paintingContext) state() (Offset, []Rect) {
	var t Offset
	var clips []Rect
	for _, level := range pc.levels {
		for _, o := range level {
			if o.clip {
				clips = append(clips, ui.NewRectFromLTWH(o.r.Left+t.Dx, o.r.Top+t.Dy, o.r.Width(), o.r.Height()))
			} else {
				t = t.Add(ui.NewOffset(o.dx, o.dy))
			}
		}
	}
	return t, clips
}

func (pc *paintingContext) addChildLayer(child RenderBox, offset Offset) {
	pc.endSegment()
	t, clips := pc.state()
	pc.layer.items = append(pc.layer.items, layerItem{child: child, at: offset.Add(t), clips: clips})
	pc.startSegment()
}

func (pc *paintingContext) finish() *Layer {
	pc.endSegment()
	return pc.layer
}

func (pc *paintingContext) Save() {
	pc.levels = append(pc.levels, nil)
	pc.cur.Save()
}

func (pc *paintingContext) Restore() {
	if len(pc.levels) > 1 {
		pc.levels = pc.levels[:len(pc.levels)-1]
	}
	pc.cur.Restore()
}

func (pc *paintingContext) Translate(dx, dy float64) {
	last := len(pc.levels) - 1
	pc.levels[last] = append(pc.levels[last], op{dx: dx, dy: dy})
	pc.cur.Translate(dx, dy)
}

func (pc *paintingContext) ClipRect(r Rect) {
	last := len(pc.levels) - 1
	pc.levels[last] = append(pc.levels[last], op{clip: true, r: r})
	pc.cur.ClipRect(r)
}

func (pc *paintingContext) DrawRect(r Rect, c Color)               { pc.cur.DrawRect(r, c) }
func (pc *paintingContext) DrawRRect(r Rect, rad float64, c Color) { pc.cur.DrawRRect(r, rad, c) }
func (pc *paintingContext) DrawParagraph(p Paragraph, at Offset)   { pc.cur.DrawParagraph(p, at) }
func (pc *paintingContext) DrawPicture(p Picture, at Offset)       { pc.cur.DrawPicture(p, at) }
func (pc *paintingContext) NewRecording() Recording                { return pc.frame.NewRecording() }

// MarkNeedsPaint marks the nearest repaint boundary at or above r dirty and
// requests a frame (Flutter's markNeedsPaint).
func MarkNeedsPaint(r RenderBox) {
	n := r
	for ; n != nil; n = n.Base().parent {
		if isBoundary(n) {
			if n.Base().needsPaint {
				return // already dirty, so a frame is already requested
			}
			n.Base().needsPaint = true
			break
		}
	}
	for ; n != nil && n.Base().parent != nil; n = n.Base().parent {
	}
	if n != nil {
		if f := n.Base().onNeedsFrame; f != nil {
			f()
		}
	}
}

// Composite paints the frame: boundary b (normally the root) and every
// boundary below it, repainting only the dirty ones. It returns how many
// boundaries were repainted.
func Composite(b RenderBox, cv Canvas, at Offset) int {
	repainted := 0
	bb := b.Base()
	if bb.needsPaint || bb.layer == nil {
		pc := newPaintingContext(cv)
		b.Paint(pc, Offset{})
		bb.layer.release()
		bb.layer = pc.finish()
		bb.needsPaint = false
		repainted++
	}
	for _, it := range bb.layer.items {
		if it.pic != nil {
			cv.DrawPicture(it.pic, at)
			continue
		}
		cv.Save()
		for _, c := range it.clips {
			cv.ClipRect(ui.NewRectFromLTWH(c.Left+at.Dx, c.Top+at.Dy, c.Width(), c.Height()))
		}
		repainted += Composite(it.child, cv, at.Add(it.at))
		cv.Restore()
	}
	return repainted
}

// RenderRepaintBoundary gives its subtree its own layer.
type RenderRepaintBoundary struct {
	Box
	SingleChild
}

func NewRepaintBoundary() *RenderRepaintBoundary         { return &RenderRepaintBoundary{} }
func (r *RenderRepaintBoundary) IsRepaintBoundary() bool { return true }
func (r *RenderRepaintBoundary) PerformLayout(c Constraints) Size {
	return layoutProxy(r.child, c)
}
func (r *RenderRepaintBoundary) Paint(cv Canvas, offset Offset) {
	if r.child != nil {
		PaintChild(cv, r.child, offset)
	}
}

func (r *RenderView) IsRepaintBoundary() bool { return true }
