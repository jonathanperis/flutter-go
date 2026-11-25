package rendering

// Proxy-style objects: one optional child that gets the same constraints.

func layoutProxy(child RenderBox, c Constraints) Size {
	if child == nil {
		return c.Smallest()
	}
	child.Base().Offset = Offset{}
	return Layout(child, c)
}

// RenderDecoratedBox paints a (possibly rounded) colored rectangle behind its
// child. It covers both Flutter's ColoredBox and a minimal DecoratedBox.
type RenderDecoratedBox struct {
	Box
	SingleChild
	color  Color
	radius float64
}

func NewDecoratedBox(color Color, radius float64) *RenderDecoratedBox {
	return &RenderDecoratedBox{color: color, radius: radius}
}

func (r *RenderDecoratedBox) Set(color Color, radius float64) {
	if color != r.color || radius != r.radius {
		r.color, r.radius = color, radius
		MarkNeedsPaint(r)
	}
}

func (r *RenderDecoratedBox) PerformLayout(c Constraints) Size { return layoutProxy(r.child, c) }
func (r *RenderDecoratedBox) HitTestSelf(Offset) bool          { return true }

func (r *RenderDecoratedBox) Paint(cv Canvas, offset Offset) {
	rect := RectFrom(offset, r.Size)
	if r.radius > 0 {
		cv.DrawRRect(rect, r.radius, r.color)
	} else {
		cv.DrawRect(rect, r.color)
	}
	if r.child != nil {
		PaintChild(cv, r.child, offset)
	}
}

// RenderConstrainedBox imposes additional constraints (SizedBox, ConstrainedBox).
type RenderConstrainedBox struct {
	Box
	SingleChild
	additional Constraints
}

func NewConstrainedBox(additional Constraints) *RenderConstrainedBox {
	return &RenderConstrainedBox{additional: additional}
}

func (r *RenderConstrainedBox) Set(additional Constraints) {
	if additional != r.additional {
		r.additional = additional
		MarkNeedsLayout(r)
	}
}

func (r *RenderConstrainedBox) PerformLayout(c Constraints) Size {
	enforced := r.additional.Enforce(c)
	if r.child == nil {
		return enforced.Smallest()
	}
	r.child.Base().Offset = Offset{}
	return Layout(r.child, enforced)
}

func (r *RenderConstrainedBox) Paint(cv Canvas, offset Offset) {
	if r.child != nil {
		PaintChild(cv, r.child, offset)
	}
}

// RenderPadding insets its child.
type RenderPadding struct {
	Box
	SingleChild
	padding EdgeInsets
}

func NewPadding(p EdgeInsets) *RenderPadding { return &RenderPadding{padding: p} }

func (r *RenderPadding) Set(p EdgeInsets) {
	if p != r.padding {
		r.padding = p
		MarkNeedsLayout(r)
	}
}

func (r *RenderPadding) PerformLayout(c Constraints) Size {
	if r.child == nil {
		return c.Constrain(r.padding.Inflate(Size{}))
	}
	childSize := Layout(r.child, c.Deflate(r.padding))
	r.child.Base().Offset = r.padding.TopLeft()
	return c.Constrain(r.padding.Inflate(childSize))
}

func (r *RenderPadding) Paint(cv Canvas, offset Offset) {
	if r.child != nil {
		PaintChild(cv, r.child, offset)
	}
}

// RenderPositionedBox aligns its child (Align, Center). A nil factor means
// "expand to fill"; a non-nil factor sizes to child × factor.
type RenderPositionedBox struct {
	Box
	SingleChild
	alignment                 Alignment
	widthFactor, heightFactor *float64
}

func NewPositionedBox(a Alignment, widthFactor, heightFactor *float64) *RenderPositionedBox {
	return &RenderPositionedBox{alignment: a, widthFactor: widthFactor, heightFactor: heightFactor}
}

func eqPtr(a, b *float64) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }

func (r *RenderPositionedBox) Set(a Alignment, widthFactor, heightFactor *float64) {
	if a != r.alignment || !eqPtr(widthFactor, r.widthFactor) || !eqPtr(heightFactor, r.heightFactor) {
		r.alignment, r.widthFactor, r.heightFactor = a, widthFactor, heightFactor
		MarkNeedsLayout(r)
	}
}

func (r *RenderPositionedBox) PerformLayout(c Constraints) Size {
	shrinkW := r.widthFactor != nil || c.MaxW == Inf
	shrinkH := r.heightFactor != nil || c.MaxH == Inf
	factor := func(f *float64) float64 {
		if f == nil {
			return 1
		}
		return *f
	}
	if r.child == nil {
		w, h := Inf, Inf
		if shrinkW {
			w = 0
		}
		if shrinkH {
			h = 0
		}
		return c.Constrain(Size{w, h})
	}
	cs := Layout(r.child, c.Loosen())
	w, h := Inf, Inf
	if shrinkW {
		w = cs.W * factor(r.widthFactor)
	}
	if shrinkH {
		h = cs.H * factor(r.heightFactor)
	}
	size := c.Constrain(Size{w, h})
	r.child.Base().Offset = r.alignment.Along(size, cs)
	return size
}

func (r *RenderPositionedBox) Paint(cv Canvas, offset Offset) {
	if r.child != nil {
		PaintChild(cv, r.child, offset)
	}
}

// RenderTapRegion reports taps on its subtree (the render half of
// GestureDetector). It is hit only through its children.
type RenderTapRegion struct {
	Box
	SingleChild
	OnTap func()
}

func NewTapRegion(onTap func()) *RenderTapRegion { return &RenderTapRegion{OnTap: onTap} }

func (r *RenderTapRegion) PerformLayout(c Constraints) Size { return layoutProxy(r.child, c) }
func (r *RenderTapRegion) Paint(cv Canvas, offset Offset) {
	if r.child != nil {
		PaintChild(cv, r.child, offset)
	}
}

// RenderView is the root of the render tree. Like Flutter's RenderView, it
// gives its child tight constraints equal to the screen size.
type RenderView struct {
	Box
	SingleChild
}

func NewView() *RenderView { return &RenderView{} }

func (r *RenderView) PerformLayout(c Constraints) Size {
	if r.child != nil {
		r.child.Base().Offset = Offset{}
		Layout(r.child, Tight(c.Constrain(Size{c.MaxW, c.MaxH})))
	}
	return c.Constrain(Size{c.MaxW, c.MaxH})
}

func (r *RenderView) Paint(cv Canvas, offset Offset) {
	if r.child != nil {
		PaintChild(cv, r.child, offset)
	}
}
