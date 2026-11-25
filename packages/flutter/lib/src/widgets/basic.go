package widgets

import r "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/rendering"

// F returns a pointer to v, for optional fields (Flutter's nullable double).
func F(v float64) *float64 { return &v }

// ColoredBox paints a color behind its child.
type ColoredBox struct {
	Color r.Color
	Child Widget
}

func (w ColoredBox) ChildWidget() Widget             { return w.Child }
func (w ColoredBox) CreateRenderObject() r.RenderBox { return r.NewDecoratedBox(w.Color, 0) }
func (w ColoredBox) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderDecoratedBox).Set(w.Color, 0)
}

// DecoratedBox paints a rounded, colored rectangle behind its child.
type DecoratedBox struct {
	Color        r.Color
	BorderRadius float64
	Child        Widget
}

func (w DecoratedBox) ChildWidget() Widget { return w.Child }
func (w DecoratedBox) CreateRenderObject() r.RenderBox {
	return r.NewDecoratedBox(w.Color, w.BorderRadius)
}
func (w DecoratedBox) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderDecoratedBox).Set(w.Color, w.BorderRadius)
}

// SizedBox forces a width and/or height. A nil dimension is unconstrained.
type SizedBox struct {
	Width, Height *float64
	Child         Widget
}

func (w SizedBox) ChildWidget() Widget { return w.Child }
func (w SizedBox) constraints() r.Constraints {
	c := r.Constraints{MinW: 0, MaxW: r.Inf, MinH: 0, MaxH: r.Inf}
	if w.Width != nil {
		c.MinW, c.MaxW = *w.Width, *w.Width
	}
	if w.Height != nil {
		c.MinH, c.MaxH = *w.Height, *w.Height
	}
	return c
}
func (w SizedBox) CreateRenderObject() r.RenderBox { return r.NewConstrainedBox(w.constraints()) }
func (w SizedBox) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderConstrainedBox).Set(w.constraints())
}

// Padding insets its child.
type Padding struct {
	Padding r.EdgeInsets
	Child   Widget
}

func (w Padding) ChildWidget() Widget             { return w.Child }
func (w Padding) CreateRenderObject() r.RenderBox { return r.NewPadding(w.Padding) }
func (w Padding) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderPadding).Set(w.Padding)
}

// Align positions its child. The zero Alignment is center.
type Align struct {
	Alignment                 r.Alignment
	WidthFactor, HeightFactor *float64
	Child                     Widget
}

func (w Align) ChildWidget() Widget { return w.Child }
func (w Align) CreateRenderObject() r.RenderBox {
	return r.NewPositionedBox(w.Alignment, w.WidthFactor, w.HeightFactor)
}
func (w Align) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderPositionedBox).Set(w.Alignment, w.WidthFactor, w.HeightFactor)
}

// Center centers its child.
type Center struct {
	WidthFactor, HeightFactor *float64
	Child                     Widget
}

func (w Center) ChildWidget() Widget { return w.Child }
func (w Center) CreateRenderObject() r.RenderBox {
	return r.NewPositionedBox(r.Center, w.WidthFactor, w.HeightFactor)
}
func (w Center) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderPositionedBox).Set(r.Center, w.WidthFactor, w.HeightFactor)
}

// Row lays children out horizontally.
type Row struct {
	MainAxisAlignment  r.MainAxisAlignment
	CrossAxisAlignment r.CrossAxisAlignment
	MainAxisSize       r.MainAxisSize
	Children           []Widget
}

func (w Row) ChildWidgets() []Widget { return w.Children }
func (w Row) CreateRenderObject() r.RenderBox {
	return r.NewFlex(r.Horizontal, w.MainAxisAlignment, w.CrossAxisAlignment, w.MainAxisSize)
}
func (w Row) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderFlex).Set(r.Horizontal, w.MainAxisAlignment, w.CrossAxisAlignment, w.MainAxisSize)
}

// Column lays children out vertically.
type Column struct {
	MainAxisAlignment  r.MainAxisAlignment
	CrossAxisAlignment r.CrossAxisAlignment
	MainAxisSize       r.MainAxisSize
	Children           []Widget
}

func (w Column) ChildWidgets() []Widget { return w.Children }
func (w Column) CreateRenderObject() r.RenderBox {
	return r.NewFlex(r.Vertical, w.MainAxisAlignment, w.CrossAxisAlignment, w.MainAxisSize)
}
func (w Column) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderFlex).Set(r.Vertical, w.MainAxisAlignment, w.CrossAxisAlignment, w.MainAxisSize)
}

// Expanded makes a Row/Column child fill its share of free space.
// Flex 0 means 1.
type Expanded struct {
	Flex  int
	Child Widget
}

func (w Expanded) ChildWidget() Widget { return w.Child }
func (w Expanded) ApplyParentData(b *r.Box) {
	b.Flex, b.Fit = max(w.Flex, 1), r.FitTight
}

// Flexible is like Expanded but lets the child be smaller than its share.
type Flexible struct {
	Flex  int
	Child Widget
}

func (w Flexible) ChildWidget() Widget { return w.Child }
func (w Flexible) ApplyParentData(b *r.Box) {
	b.Flex, b.Fit = max(w.Flex, 1), r.FitLoose
}

// Text displays a string in a single style. Unset style fields come from
// the nearest DefaultTextStyle (like Flutter's Text, which builds a RichText
// with the merged style).
type Text struct {
	Text  string
	Style r.TextStyle
}

func (w Text) Build(ctx *Element) Widget {
	style := w.Style
	if d, ok := DependOn[DefaultTextStyle](ctx); ok {
		style = d.Style.Merge(w.Style)
	}
	return RichText{Text: w.Text, Style: style}
}

// RichText paints text in exactly the given style (no inheritance).
type RichText struct {
	Text  string
	Style r.TextStyle
}

func (w RichText) CreateRenderObject() r.RenderBox { return r.NewParagraph(w.Text, w.Style) }
func (w RichText) UpdateRenderObject(ro r.RenderBox) {
	ro.(*r.RenderParagraph).Set(w.Text, w.Style)
}

// DefaultTextStyle sets the text style for Text widgets below it.
type DefaultTextStyle struct {
	Style r.TextStyle
	Child Widget
}

func (w DefaultTextStyle) ChildWidget() Widget { return w.Child }
func (w DefaultTextStyle) UpdateShouldNotify(old Widget) bool {
	return old.(DefaultTextStyle).Style != w.Style
}

// MediaQuery describes the view: its logical size and pixel density. The
// framework puts one at the root.
type MediaQuery struct {
	Size             r.Size
	DevicePixelRatio float64
	Child            Widget
}

func (w MediaQuery) ChildWidget() Widget { return w.Child }
func (w MediaQuery) UpdateShouldNotify(old Widget) bool {
	o := old.(MediaQuery)
	return o.Size != w.Size || o.DevicePixelRatio != w.DevicePixelRatio
}

// MediaQueryOf returns the nearest MediaQuery and rebuilds ctx when it changes.
func MediaQueryOf(ctx *Element) MediaQuery {
	m, _ := DependOn[MediaQuery](ctx)
	return m
}

// GestureDetector calls OnTap when its subtree is tapped. OnTapAt also gets
// the tap position (global coordinates).
type GestureDetector struct {
	OnTap   func()
	OnTapAt func(global r.Offset)
	Child   Widget
}

func (w GestureDetector) ChildWidget() Widget { return w.Child }
func (w GestureDetector) CreateRenderObject() r.RenderBox {
	t := r.NewTapRegion(w.OnTap)
	t.OnTapAt = w.OnTapAt
	return t
}
func (w GestureDetector) UpdateRenderObject(ro r.RenderBox) {
	t := ro.(*r.RenderTapRegion)
	t.OnTap, t.OnTapAt = w.OnTap, w.OnTapAt
}

// KeyedSubtree attaches a key to a child, for list reconciliation.
type KeyedSubtree struct {
	Key   any
	Child Widget
}

func (w KeyedSubtree) WidgetKey() any        { return w.Key }
func (w KeyedSubtree) Build(*Element) Widget { return w.Child }
