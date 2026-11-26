package painting

// Port of box_border.dart (Flutter 3.50.1): BoxShape, BoxBorder, Border
// and BorderDirectional.
//
// BoxBorder.paint takes two more named parameters than ShapeBorder.paint
// (shape, borderRadius); Go can't overload the interface method, so it is
// PaintBox(canvas, rect, BoxBorderPaintOptions), and Paint is PaintBox
// with the defaults. The `top`/`bottom` getters BoxBorder declares are
// methods, so Border and BorderDirectional keep their sides in unexported
// fields read by Top(), Right()... (build them with NewBorder...).

import (
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// BoxShape is the shape to use when rendering a Border or BoxDecoration.
type BoxShape int

const (
	// BoxShapeRectangle is an axis-aligned rectangle, optionally with
	// rounded corners.
	BoxShapeRectangle BoxShape = iota
	// BoxShapeCircle is a circle centered in the middle of the box.
	BoxShapeCircle
)

// BoxShapeValues is BoxShape.values.
var BoxShapeValues = []BoxShape{BoxShapeRectangle, BoxShapeCircle}

// Name is the Dart enum value's name.
func (s BoxShape) Name() string { return [...]string{"rectangle", "circle"}[s] }

func (s BoxShape) String() string { return "BoxShape." + s.Name() }

// BoxBorderPaintOptions are BoxBorder.paint's named parameters.
type BoxBorderPaintOptions struct {
	TextDirection *ui.TextDirection
	Shape         BoxShape
	// BorderRadius nil is null.
	BorderRadius *BorderRadius
}

// BoxBorder is the base of Border and BorderDirectional: borders that
// can be drawn around a box (BoxDecoration).
type BoxBorder interface {
	ShapeBorder
	// Top is the top side of this border.
	Top() BorderSide
	// Bottom is the bottom side of this border.
	Bottom() BorderSide
	// IsUniform reports whether all four sides have the same color, width,
	// style and stroke alignment.
	IsUniform() bool
	// PaintBox is BoxBorder.paint(canvas, rect, {textDirection, shape,
	// borderRadius}).
	PaintBox(canvas ui.Canvas, rect ui.Rect, opts BoxBorderPaintOptions)
}

// BorderOptions are Border's named sides (nil: BorderSide.none).
type BorderOptions struct {
	Top, Right, Bottom, Left *BorderSide
}

// NewBoxBorderFromLTRB is BoxBorder.fromLTRB: a Border.
func NewBoxBorderFromLTRB(opts ...BorderOptions) BoxBorder { return NewBorder(opts...) }

// NewBoxBorderAll is BoxBorder.all: Border.all.
func NewBoxBorderAll(opts ...BorderSideOptions) BoxBorder { return NewBorderAll(opts...) }

// NewBoxBorderFromBorderSide is BoxBorder.fromBorderSide: Border.fromBorderSide.
func NewBoxBorderFromBorderSide(side BorderSide) BoxBorder { return NewBorderFromBorderSide(side) }

// NewBoxBorderSymmetric is BoxBorder.symmetric: Border.symmetric.
func NewBoxBorderSymmetric(opts ...BorderSymmetricOptions) BoxBorder {
	return NewBorderSymmetric(opts...)
}

// BorderDirectionalOptions are BorderDirectional's named sides (nil:
// BorderSide.none).
type BorderDirectionalOptions struct {
	Top, Start, End, Bottom *BorderSide
}

// NewBoxBorderFromSTEB is BoxBorder.fromSTEB: a BorderDirectional.
func NewBoxBorderFromSTEB(opts ...BorderDirectionalOptions) BoxBorder {
	return NewBorderDirectional(opts...)
}

func identicalBoxBorders(a, b BoxBorder) bool {
	var sa, sb ShapeBorder
	if a != nil {
		sa = a
	}
	if b != nil {
		sb = b
	}
	return identicalBorders(sa, sb)
}

// BoxBorderLerp is BoxBorder.lerp: linearly interpolates between two
// box borders (nil is null), including between a Border and a
// BorderDirectional.
func BoxBorderLerp(a, b BoxBorder, t float64) BoxBorder {
	if identicalBoxBorders(a, b) {
		return a
	}
	ab, aIsBorder := a.(Border)
	bb, bIsBorder := b.(Border)
	if (a == nil || aIsBorder) && (b == nil || bIsBorder) {
		var pa, pb *Border
		if aIsBorder {
			pa = &ab
		}
		if bIsBorder {
			pb = &bb
		}
		return *BorderLerp(pa, pb, t)
	}
	ad, aIsDir := a.(BorderDirectional)
	bd, bIsDir := b.(BorderDirectional)
	if (a == nil || aIsDir) && (b == nil || bIsDir) {
		var pa, pb *BorderDirectional
		if aIsDir {
			pa = &ad
		}
		if bIsDir {
			pb = &bd
		}
		return *BorderDirectionalLerp(pa, pb, t)
	}
	if bIsBorder && aIsDir {
		ab, bd = bb, ad
		aIsBorder, bIsDir = true, true
		t = 1.0 - t
	}
	if aIsBorder && bIsDir {
		a, b := ab, bd
		none := BorderSideNone
		if b.start == none && b.end == none {
			return Border{
				top:    BorderSideLerp(a.top, b.top, t),
				right:  BorderSideLerp(a.right, none, t),
				bottom: BorderSideLerp(a.bottom, b.bottom, t),
				left:   BorderSideLerp(a.left, none, t),
			}
		}
		if a.left == none && a.right == none {
			return BorderDirectional{
				top:    BorderSideLerp(a.top, b.top, t),
				start:  BorderSideLerp(none, b.start, t),
				end:    BorderSideLerp(none, b.end, t),
				bottom: BorderSideLerp(a.bottom, b.bottom, t),
			}
		}
		if t < 0.5 {
			return Border{
				top:    BorderSideLerp(a.top, b.top, t),
				right:  BorderSideLerp(a.right, none, t*2.0),
				bottom: BorderSideLerp(a.bottom, b.bottom, t),
				left:   BorderSideLerp(a.left, none, t*2.0),
			}
		}
		return BorderDirectional{
			top:    BorderSideLerp(a.top, b.top, t),
			start:  BorderSideLerp(none, b.start, (t-0.5)*2.0),
			end:    BorderSideLerp(none, b.end, (t-0.5)*2.0),
			bottom: BorderSideLerp(a.bottom, b.bottom, t),
		}
	}
	var result ShapeBorder
	if b != nil {
		result = b.LerpFrom(a, t)
	}
	if result == nil && a != nil {
		result = a.LerpTo(b, t)
	}
	if result != nil {
		r, ok := result.(BoxBorder)
		if !ok {
			panic(dart.NewAssertionError("type '" + shapeBorderTypeName(result) + "' is not a subtype of type 'BoxBorder?' in type cast"))
		}
		return r
	}
	if t < 0.5 {
		return a
	}
	return b
}

// boxBorderInnerPath and boxBorderOuterPath are BoxBorder's getInnerPath
// and getOuterPath.
func boxBorderInnerPath(self BoxBorder, name string, rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	if textDirection == nil {
		panic(dart.NewAssertionError("The textDirection argument to " + name + ".getInnerPath must not be null."))
	}
	p := ui.NewPath()
	p.AddRect(self.Dimensions().Resolve(textDirection).DeflateRect(rect))
	return p
}

func boxBorderOuterPath(name string, rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	if textDirection == nil {
		panic(dart.NewAssertionError("The textDirection argument to " + name + ".getOuterPath must not be null."))
	}
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}

func paintUniformBorderWithRadius(canvas ui.Canvas, rect ui.Rect, side BorderSide, borderRadius BorderRadius) {
	if side.Style == BorderStyleNone {
		panic(dart.NewAssertionError("side.style != BorderStyle.none"))
	}
	paint := ui.NewPaint()
	paint.SetColor(side.Color)
	width := side.Width
	if width == 0.0 {
		paint.SetStyle(ui.PaintingStyleStroke)
		paint.SetStrokeWidth(0.0)
		canvas.DrawRRect(borderRadius.ToRRect(rect), paint)
	} else {
		borderRect := borderRadius.ToRRect(rect)
		inner := borderRect.Deflate(side.StrokeInset())
		outer := borderRect.Inflate(side.StrokeOutset())
		canvas.DrawDRRect(outer, inner, paint)
	}
}

// BoxBorderPaintNonUniformBorderOptions are paintNonUniformBorder's
// optional named parameters (nil sides: BorderSide.none).
type BoxBorderPaintNonUniformBorderOptions struct {
	Shape                    BoxShape
	Top, Right, Bottom, Left *BorderSide
}

// BoxBorderPaintNonUniformBorder is BoxBorder.paintNonUniformBorder:
// paints a border with different widths (one color) around a rounded
// rectangle or circle. Dart's required named borderRadius (nil: null),
// textDirection (nil: null) and color are positional.
func BoxBorderPaintNonUniformBorder(canvas ui.Canvas, rect ui.Rect, borderRadius *BorderRadius,
	textDirection *ui.TextDirection, color ui.Color, opts ...BoxBorderPaintNonUniformBorderOptions) {
	var o BoxBorderPaintNonUniformBorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	top, right, bottom, left := sideOrNone(o.Top), sideOrNone(o.Right), sideOrNone(o.Bottom), sideOrNone(o.Left)
	var borderRect ui.RRect
	switch o.Shape {
	case BoxShapeRectangle:
		r := BorderRadiusZero
		if borderRadius != nil {
			r = *borderRadius
		}
		borderRect = r.Resolve(textDirection).ToRRect(rect)
	case BoxShapeCircle:
		if borderRadius != nil {
			panic(dart.NewAssertionError("A circle cannot have a border radius. Remove either the shape or the borderRadius argument."))
		}
		borderRect = ui.NewRRectFromRectAndRadius(
			ui.NewRectFromCircle(rect.Center(), rect.ShortestSide()/2.0),
			ui.NewRadiusCircular(rect.Width()))
	}
	paint := ui.NewPaint()
	paint.SetColor(color)
	inner := NewEdgeInsetsFromLTRB(left.StrokeInset(), top.StrokeInset(), right.StrokeInset(),
		bottom.StrokeInset()).DeflateRRect(borderRect)
	outer := NewEdgeInsetsFromLTRB(left.StrokeOutset(), top.StrokeOutset(), right.StrokeOutset(),
		bottom.StrokeOutset()).InflateRRect(borderRect)
	canvas.DrawDRRect(outer, inner, paint)
}

func paintUniformBorderWithCircle(canvas ui.Canvas, rect ui.Rect, side BorderSide) {
	if side.Style == BorderStyleNone {
		panic(dart.NewAssertionError("side.style != BorderStyle.none"))
	}
	radius := (rect.ShortestSide() + side.StrokeOffset()) / 2
	canvas.DrawCircle(rect.Center(), radius, side.ToPaint())
}

func paintUniformBorderWithRectangle(canvas ui.Canvas, rect ui.Rect, side BorderSide) {
	if side.Style == BorderStyleNone {
		panic(dart.NewAssertionError("side.style != BorderStyle.none"))
	}
	canvas.DrawRect(rect.Inflate(side.StrokeOffset()/2), side.ToPaint())
}

const circleWithRadiusMessage = "A circle cannot have a border radius. Remove either the shape or the borderRadius argument."

// paintUniform is the uniform case shared by Border.paint and
// BorderDirectional.paint; it reports whether it painted (or skipped) the
// border.
func paintUniform(canvas ui.Canvas, rect ui.Rect, top BorderSide, opts BoxBorderPaintOptions) {
	if top.Style == BorderStyleNone {
		return
	}
	switch opts.Shape {
	case BoxShapeCircle:
		if opts.BorderRadius != nil {
			panic(dart.NewAssertionError(circleWithRadiusMessage))
		}
		paintUniformBorderWithCircle(canvas, rect, top)
	case BoxShapeRectangle:
		if opts.BorderRadius != nil && *opts.BorderRadius != BorderRadiusZero {
			paintUniformBorderWithRadius(canvas, rect, top, *opts.BorderRadius)
			return
		}
		paintUniformBorderWithRectangle(canvas, rect, top)
	}
}

// Border is a border of a box, comprised of four sides: top, right,
// bottom, left. It is a comparable value; build it with NewBorder (its zero
// value is not Dart's Border()).
type Border struct {
	top, right, bottom, left BorderSide
}

// NewBorder is Border({top, right, bottom, left}).
func NewBorder(opts ...BorderOptions) Border {
	var o BorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return Border{sideOrNone(o.Top), sideOrNone(o.Right), sideOrNone(o.Bottom), sideOrNone(o.Left)}
}

// NewBorderFromBorderSide is Border.fromBorderSide: all four sides are side.
func NewBorderFromBorderSide(side BorderSide) Border { return Border{side, side, side, side} }

// BorderSymmetricOptions are Border.symmetric's named parameters (nil:
// BorderSide.none).
type BorderSymmetricOptions struct {
	Vertical, Horizontal *BorderSide
}

// NewBorderSymmetric is Border.symmetric: vertical is the left and right
// sides, horizontal the top and bottom.
func NewBorderSymmetric(opts ...BorderSymmetricOptions) Border {
	var o BorderSymmetricOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	v, h := sideOrNone(o.Vertical), sideOrNone(o.Horizontal)
	return Border{top: h, right: v, bottom: h, left: v}
}

// NewBorderAll is Border.all({color, width, style, strokeAlign}).
func NewBorderAll(opts ...BorderSideOptions) Border {
	return NewBorderFromBorderSide(NewBorderSide(opts...))
}

// BorderMerge is Border.merge: each side merged with BorderSideMerge. It
// panics unless every pair of sides can merge, as Dart asserts.
func BorderMerge(a, b Border) Border {
	for _, p := range [][2]BorderSide{{a.top, b.top}, {a.right, b.right}, {a.bottom, b.bottom}, {a.left, b.left}} {
		if !BorderSideCanMerge(p[0], p[1]) {
			panic(dart.NewAssertionError("BorderSide.canMerge"))
		}
	}
	return Border{
		top:    BorderSideMerge(a.top, b.top),
		right:  BorderSideMerge(a.right, b.right),
		bottom: BorderSideMerge(a.bottom, b.bottom),
		left:   BorderSideMerge(a.left, b.left),
	}
}

// Top is the top side of this border.
func (b Border) Top() BorderSide { return b.top }

// Right is the right side of this border.
func (b Border) Right() BorderSide { return b.right }

// Bottom is the bottom side of this border.
func (b Border) Bottom() BorderSide { return b.bottom }

// Left is the left side of this border.
func (b Border) Left() BorderSide { return b.left }

// Dimensions is the sides' stroke insets.
func (b Border) Dimensions() EdgeInsetsGeometry {
	return NewEdgeInsetsFromLTRB(b.left.StrokeInset(), b.top.StrokeInset(), b.right.StrokeInset(), b.bottom.StrokeInset())
}

// IsUniform reports whether the four sides are the same.
func (b Border) IsUniform() bool {
	return b.colorIsUniform() && b.widthIsUniform() && b.styleIsUniform() && b.strokeAlignIsUniform()
}

func (b Border) colorIsUniform() bool {
	c := b.top.Color
	return b.left.Color == c && b.bottom.Color == c && b.right.Color == c
}

func (b Border) widthIsUniform() bool {
	w := b.top.Width
	return b.left.Width == w && b.bottom.Width == w && b.right.Width == w
}

func (b Border) styleIsUniform() bool {
	s := b.top.Style
	return b.left.Style == s && b.bottom.Style == s && b.right.Style == s
}

func (b Border) strokeAlignIsUniform() bool {
	a := b.top.StrokeAlign
	return b.left.StrokeAlign == a && b.bottom.StrokeAlign == a && b.right.StrokeAlign == a
}

// distinctVisibleColors is a Dart set literal: insertion order, no
// duplicates.
func distinctVisibleColors(sides ...BorderSide) []ui.Color {
	var out []ui.Color
outer:
	for _, s := range sides {
		if s.Style == BorderStyleNone {
			continue
		}
		for _, c := range out {
			if c == s.Color {
				continue outer
			}
		}
		out = append(out, s.Color)
	}
	return out
}

func hasHairlineBorder(sides ...BorderSide) bool {
	for _, s := range sides {
		if s.Style == BorderStyleSolid && s.Width == 0.0 {
			return true
		}
	}
	return false
}

// AddBorder merges other into this border when it is a Border whose
// sides can merge; otherwise nil.
func (b Border) AddBorder(other ShapeBorder, reversed bool) ShapeBorder {
	if o, ok := other.(Border); ok &&
		BorderSideCanMerge(b.top, o.top) && BorderSideCanMerge(b.right, o.right) &&
		BorderSideCanMerge(b.bottom, o.bottom) && BorderSideCanMerge(b.left, o.left) {
		return BorderMerge(b, o)
	}
	return nil
}

// Add is operator +.
func (b Border) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

// Scale scales each side by t.
func (b Border) Scale(t float64) ShapeBorder { return b.ScaleBorder(t) }

// ScaleBorder is Scale with Dart's covariant Border result.
func (b Border) ScaleBorder(t float64) Border {
	return Border{b.top.Scale(t), b.right.Scale(t), b.bottom.Scale(t), b.left.Scale(t)}
}

// LerpFrom interpolates from a Border (or nil).
func (b Border) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(Border); ok {
		return *BorderLerp(&x, &b, t)
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a Border (or nil).
func (b Border) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(Border); ok {
		return *BorderLerp(&b, &x, t)
	}
	return shapeBorderLerpTo(b, other, t)
}

// BorderLerp is Border.lerp (nil is null).
func BorderLerp(a, b *Border, t float64) *Border {
	if a == b || (a != nil && b != nil && *a == *b) {
		return a
	}
	if a == nil {
		r := b.ScaleBorder(t)
		return &r
	}
	if b == nil {
		r := a.ScaleBorder(1.0 - t)
		return &r
	}
	return &Border{
		top:    BorderSideLerp(a.top, b.top, t),
		right:  BorderSideLerp(a.right, b.right, t),
		bottom: BorderSideLerp(a.bottom, b.bottom, t),
		left:   BorderSideLerp(a.left, b.left, t),
	}
}

// GetInnerPath is the rect deflated by the dimensions.
func (b Border) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return boxBorderInnerPath(b, "Border", rect, textDirection)
}

// GetOuterPath is the rect.
func (b Border) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return boxBorderOuterPath("Border", rect, textDirection)
}

// HitTest is rect.contains(position).
func (b Border) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return rect.Contains(position)
}

// PaintInterior draws the rect.
func (b Border) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	canvas.DrawRect(rect, paint)
}

// PreferPaintInterior is true.
func (b Border) PreferPaintInterior() bool { return true }

// Paint is PaintBox with a rectangle shape and no radius.
func (b Border) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	b.PaintBox(canvas, rect, BoxBorderPaintOptions{TextDirection: textDirection})
}

// PaintBox paints the border within the given rect.
func (b Border) PaintBox(canvas ui.Canvas, rect ui.Rect, opts BoxBorderPaintOptions) {
	if b.IsUniform() {
		paintUniform(canvas, rect, b.top, opts)
		return
	}
	if b.styleIsUniform() && b.top.Style == BorderStyleNone {
		return
	}
	visibleColors := distinctVisibleColors(b.top, b.right, b.bottom, b.left)
	hairline := hasHairlineBorder(b.top, b.right, b.bottom, b.left)
	shape, borderRadius := opts.Shape, opts.BorderRadius
	hasRadius := borderRadius != nil && *borderRadius != BorderRadiusZero
	if len(visibleColors) == 1 && !hairline && (shape == BoxShapeCircle || hasRadius) {
		top, right, bottom, left := visibleOrNone(b.top), visibleOrNone(b.right), visibleOrNone(b.bottom), visibleOrNone(b.left)
		BoxBorderPaintNonUniformBorder(canvas, rect, borderRadius, opts.TextDirection, visibleColors[0],
			BoxBorderPaintNonUniformBorderOptions{Shape: shape, Top: &top, Right: &right, Bottom: &bottom, Left: &left})
		return
	}
	if hairline && hasRadius {
		panic(dart.NewAssertionError("A hairline border like `BorderSide(width: 0.0, style: BorderStyle.solid)` can only be drawn when BorderRadius is zero or null."))
	}
	notUniform := func() []foundation.DiagnosticsNode {
		if !b.colorIsUniform() {
			return []foundation.DiagnosticsNode{foundation.NewErrorDescription("BorderSide.color")}
		}
		return nil
	}
	if hasRadius {
		panic(foundation.NewFlutterErrorFromParts(append([]foundation.DiagnosticsNode{
			foundation.NewErrorSummary("A borderRadius can only be given on borders with uniform colors."),
			foundation.NewErrorDescription("The following is not uniform:"),
		}, notUniform()...)))
	}
	if shape != BoxShapeRectangle {
		panic(foundation.NewFlutterErrorFromParts(append([]foundation.DiagnosticsNode{
			foundation.NewErrorSummary("A Border can only be drawn as a circle on borders with uniform colors."),
			foundation.NewErrorDescription("The following is not uniform:"),
		}, notUniform()...)))
	}
	if !b.strokeAlignIsUniform() || b.top.StrokeAlign != BorderSideStrokeAlignInside {
		panic(foundation.NewFlutterErrorFromParts([]foundation.DiagnosticsNode{
			foundation.NewErrorSummary("A Border can only draw strokeAlign different than BorderSide.strokeAlignInside on borders with uniform colors."),
		}))
	}
	top, right, bottom, left := b.top, b.right, b.bottom, b.left
	PaintBorder(canvas, rect, PaintBorderOptions{Top: &top, Right: &right, Bottom: &bottom, Left: &left})
}

func visibleOrNone(s BorderSide) BorderSide {
	if s.Style == BorderStyleNone {
		return BorderSideNone
	}
	return s
}

func (b Border) String() string {
	if b.IsUniform() {
		return "Border.all(" + b.top.String() + ")"
	}
	var args []string
	for _, s := range []struct {
		name string
		side BorderSide
	}{{"top", b.top}, {"right", b.right}, {"bottom", b.bottom}, {"left", b.left}} {
		if s.side != BorderSideNone {
			args = append(args, s.name+": "+s.side.String())
		}
	}
	return "Border(" + strings.Join(args, ", ") + ")"
}

// BorderDirectional is a border of a box whose left and right sides
// depend on the text direction (start and end). It is a comparable value;
// build it with NewBorderDirectional.
type BorderDirectional struct {
	top, start, end, bottom BorderSide
}

// NewBorderDirectional is BorderDirectional({top, start, end, bottom}).
func NewBorderDirectional(opts ...BorderDirectionalOptions) BorderDirectional {
	var o BorderDirectionalOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return BorderDirectional{sideOrNone(o.Top), sideOrNone(o.Start), sideOrNone(o.End), sideOrNone(o.Bottom)}
}

// BorderDirectionalMerge is BorderDirectional.merge.
func BorderDirectionalMerge(a, b BorderDirectional) BorderDirectional {
	for _, p := range [][2]BorderSide{{a.top, b.top}, {a.start, b.start}, {a.end, b.end}, {a.bottom, b.bottom}} {
		if !BorderSideCanMerge(p[0], p[1]) {
			panic(dart.NewAssertionError("BorderSide.canMerge"))
		}
	}
	return BorderDirectional{
		top:    BorderSideMerge(a.top, b.top),
		start:  BorderSideMerge(a.start, b.start),
		end:    BorderSideMerge(a.end, b.end),
		bottom: BorderSideMerge(a.bottom, b.bottom),
	}
}

// Top is the top side of this border.
func (b BorderDirectional) Top() BorderSide { return b.top }

// Start is the side at the start of the reading direction.
func (b BorderDirectional) Start() BorderSide { return b.start }

// End is the side at the end of the reading direction.
func (b BorderDirectional) End() BorderSide { return b.end }

// Bottom is the bottom side of this border.
func (b BorderDirectional) Bottom() BorderSide { return b.bottom }

// Dimensions is the sides' stroke insets (directional).
func (b BorderDirectional) Dimensions() EdgeInsetsGeometry {
	return NewEdgeInsetsDirectionalFromSTEB(b.start.StrokeInset(), b.top.StrokeInset(), b.end.StrokeInset(), b.bottom.StrokeInset())
}

// IsUniform reports whether the four sides are the same.
func (b BorderDirectional) IsUniform() bool {
	return b.colorIsUniform() && b.widthIsUniform() && b.styleIsUniform() && b.strokeAlignIsUniform()
}

func (b BorderDirectional) colorIsUniform() bool {
	c := b.top.Color
	return b.start.Color == c && b.bottom.Color == c && b.end.Color == c
}

func (b BorderDirectional) widthIsUniform() bool {
	w := b.top.Width
	return b.start.Width == w && b.bottom.Width == w && b.end.Width == w
}

func (b BorderDirectional) styleIsUniform() bool {
	s := b.top.Style
	return b.start.Style == s && b.bottom.Style == s && b.end.Style == s
}

func (b BorderDirectional) strokeAlignIsUniform() bool {
	a := b.top.StrokeAlign
	return b.start.StrokeAlign == a && b.bottom.StrokeAlign == a && b.end.StrokeAlign == a
}

// AddBorder merges other into this border when possible (a
// BorderDirectional, or a Border without conflicting sides); otherwise nil.
func (b BorderDirectional) AddBorder(other ShapeBorder, reversed bool) ShapeBorder {
	switch o := other.(type) {
	case BorderDirectional:
		if BorderSideCanMerge(b.top, o.top) && BorderSideCanMerge(b.start, o.start) &&
			BorderSideCanMerge(b.end, o.end) && BorderSideCanMerge(b.bottom, o.bottom) {
			return BorderDirectionalMerge(b, o)
		}
		return nil
	case Border:
		if !BorderSideCanMerge(o.top, b.top) || !BorderSideCanMerge(o.bottom, b.bottom) {
			return nil
		}
		if b.start != BorderSideNone || b.end != BorderSideNone {
			if o.left != BorderSideNone || o.right != BorderSideNone {
				return nil
			}
			return BorderDirectional{
				top:    BorderSideMerge(o.top, b.top),
				start:  b.start,
				end:    b.end,
				bottom: BorderSideMerge(o.bottom, b.bottom),
			}
		}
		return Border{
			top:    BorderSideMerge(o.top, b.top),
			right:  o.right,
			bottom: BorderSideMerge(o.bottom, b.bottom),
			left:   o.left,
		}
	}
	return nil
}

// Add is operator +.
func (b BorderDirectional) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(b, other) }

// Scale scales each side by t.
func (b BorderDirectional) Scale(t float64) ShapeBorder { return b.ScaleBorder(t) }

// ScaleBorder is Scale with Dart's covariant BorderDirectional result.
func (b BorderDirectional) ScaleBorder(t float64) BorderDirectional {
	return BorderDirectional{b.top.Scale(t), b.start.Scale(t), b.end.Scale(t), b.bottom.Scale(t)}
}

// LerpFrom interpolates from a BorderDirectional (or nil).
func (b BorderDirectional) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if x, ok := a.(BorderDirectional); ok {
		return *BorderDirectionalLerp(&x, &b, t)
	}
	return shapeBorderLerpFrom(b, a, t)
}

// LerpTo interpolates to a BorderDirectional (or nil).
func (b BorderDirectional) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	if x, ok := other.(BorderDirectional); ok {
		return *BorderDirectionalLerp(&b, &x, t)
	}
	return shapeBorderLerpTo(b, other, t)
}

// BorderDirectionalLerp is BorderDirectional.lerp (nil is null).
func BorderDirectionalLerp(a, b *BorderDirectional, t float64) *BorderDirectional {
	if a == b || (a != nil && b != nil && *a == *b) {
		return a
	}
	if a == nil {
		r := b.ScaleBorder(t)
		return &r
	}
	if b == nil {
		r := a.ScaleBorder(1.0 - t)
		return &r
	}
	return &BorderDirectional{
		top:    BorderSideLerp(a.top, b.top, t),
		end:    BorderSideLerp(a.end, b.end, t),
		bottom: BorderSideLerp(a.bottom, b.bottom, t),
		start:  BorderSideLerp(a.start, b.start, t),
	}
}

// GetInnerPath is the rect deflated by the resolved dimensions.
func (b BorderDirectional) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return boxBorderInnerPath(b, "BorderDirectional", rect, textDirection)
}

// GetOuterPath is the rect.
func (b BorderDirectional) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return boxBorderOuterPath("BorderDirectional", rect, textDirection)
}

// HitTest is rect.contains(position).
func (b BorderDirectional) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return rect.Contains(position)
}

// PaintInterior draws the rect.
func (b BorderDirectional) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	canvas.DrawRect(rect, paint)
}

// PreferPaintInterior is true.
func (b BorderDirectional) PreferPaintInterior() bool { return true }

// Paint is PaintBox with a rectangle shape and no radius.
func (b BorderDirectional) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	b.PaintBox(canvas, rect, BoxBorderPaintOptions{TextDirection: textDirection})
}

// PaintBox paints the border within the given rect; non-uniform borders
// need a text direction.
func (b BorderDirectional) PaintBox(canvas ui.Canvas, rect ui.Rect, opts BoxBorderPaintOptions) {
	if b.IsUniform() {
		paintUniform(canvas, rect, b.top, opts)
		return
	}
	if b.styleIsUniform() && b.top.Style == BorderStyleNone {
		return
	}
	if opts.TextDirection == nil {
		panic(dart.NewAssertionError("Non-uniform BorderDirectional objects require a TextDirection when painting."))
	}
	left, right := b.start, b.end
	if *opts.TextDirection == ui.TextDirectionRtl {
		left, right = b.end, b.start
	}
	visibleColors := distinctVisibleColors(b.top, b.end, b.bottom, b.start)
	hairline := hasHairlineBorder(b.top, b.end, b.bottom, b.start)
	shape, borderRadius := opts.Shape, opts.BorderRadius
	hasRadius := borderRadius != nil && *borderRadius != BorderRadiusZero
	if len(visibleColors) == 1 && !hairline && (shape == BoxShapeCircle || hasRadius) {
		t, r, bo, l := visibleOrNone(b.top), visibleOrNone(right), visibleOrNone(b.bottom), visibleOrNone(left)
		BoxBorderPaintNonUniformBorder(canvas, rect, borderRadius, opts.TextDirection, visibleColors[0],
			BoxBorderPaintNonUniformBorderOptions{Shape: shape, Top: &t, Right: &r, Bottom: &bo, Left: &l})
		return
	}
	if hairline && hasRadius {
		panic(dart.NewAssertionError("A side like `BorderSide(width: 0.0, style: BorderStyle.solid)` can only be drawn when BorderRadius is zero or null."))
	}
	if borderRadius != nil {
		panic(dart.NewAssertionError("A borderRadius can only be given for borders with uniform colors."))
	}
	if shape != BoxShapeRectangle {
		panic(dart.NewAssertionError("A Border can only be drawn as a circle on borders with uniform colors."))
	}
	if !(b.strokeAlignIsUniform() && b.top.StrokeAlign == BorderSideStrokeAlignInside) {
		panic(dart.NewAssertionError("A Border can only draw strokeAlign different than strokeAlignInside on borders with uniform colors."))
	}
	top, bottom := b.top, b.bottom
	PaintBorder(canvas, rect, PaintBorderOptions{Top: &top, Left: &left, Bottom: &bottom, Right: &right})
}

func (b BorderDirectional) String() string {
	var args []string
	for _, s := range []struct {
		name string
		side BorderSide
	}{{"top", b.top}, {"start", b.start}, {"end", b.end}, {"bottom", b.bottom}} {
		if s.side != BorderSideNone {
			args = append(args, s.name+": "+s.side.String())
		}
	}
	return "BorderDirectional(" + strings.Join(args, ", ") + ")"
}
