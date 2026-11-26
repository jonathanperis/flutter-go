package painting

// Port of borders.dart (Flutter 3.50.1): BorderStyle, BorderSide,
// ShapeBorder, OutlinedBorder, the private _CompoundBorder and paintBorder.
//
// ShapeBorder is an abstract class users extend: it is an interface plus
// the embeddable ShapeBorderBase (self pattern) holding the default
// members. Flutter's own borders are immutable values and implement the
// interface directly (they compare with Go's ==, as Dart's == compares
// their fields; LinearBorder and the compound border have Equal).

import (
	"math"
	"reflect"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// BorderStyle is the style of line to draw for a BorderSide in a Border.
type BorderStyle int

const (
	// BorderStyleNone skips the border.
	BorderStyleNone BorderStyle = iota
	// BorderStyleSolid draws the border as a solid line.
	BorderStyleSolid
)

// BorderStyleValues is BorderStyle.values.
var BorderStyleValues = []BorderStyle{BorderStyleNone, BorderStyleSolid}

// Name is the Dart enum value's name.
func (s BorderStyle) Name() string { return [...]string{"none", "solid"}[s] }

func (s BorderStyle) String() string { return "BorderStyle." + s.Name() }

// BorderSide's strokeAlign constants.
const (
	// BorderSideStrokeAlignInside: the border is drawn fully inside the
	// border path.
	BorderSideStrokeAlignInside = -1.0
	// BorderSideStrokeAlignCenter: the border is drawn on the center of
	// the border path.
	BorderSideStrokeAlignCenter = 0.0
	// BorderSideStrokeAlignOutside: the border is drawn on the outside of
	// the border path.
	BorderSideStrokeAlignOutside = 1.0
)

// BorderSide is a side of a border of a box. It is a comparable value
// (Dart's ==); build it with NewBorderSide (its zero value is not Dart's
// default side).
type BorderSide struct {
	// Color is the color of this side of the border.
	Color ui.Color
	// Width is the width of this side of the border, in logical pixels.
	Width float64
	// Style is the style of this side of the border.
	Style BorderStyle
	// StrokeAlign is the relative position of the stroke on a BorderSide
	// in an OutlinedBorder or Border (-1 inside, 0 center, 1 outside).
	StrokeAlign float64
}

// BorderSideOptions are BorderSide's named parameters; nil takes Dart's
// default (black, 1.0, solid, strokeAlignInside).
type BorderSideOptions struct {
	Color       *ui.Color
	Width       *float64
	Style       *BorderStyle
	StrokeAlign *float64
}

var borderSideDefaultColor = ui.NewColor(0xFF000000)

// NewBorderSide is BorderSide({color, width, style, strokeAlign}); it
// panics when width is negative, as Dart asserts.
func NewBorderSide(opts ...BorderSideOptions) BorderSide {
	var o BorderSideOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := BorderSide{Color: borderSideDefaultColor, Width: 1.0, Style: BorderStyleSolid,
		StrokeAlign: BorderSideStrokeAlignInside}
	if o.Color != nil {
		s.Color = *o.Color
	}
	if o.Width != nil {
		s.Width = *o.Width
	}
	if o.Style != nil {
		s.Style = *o.Style
	}
	if o.StrokeAlign != nil {
		s.StrokeAlign = *o.StrokeAlign
	}
	return s.checked()
}

func (s BorderSide) checked() BorderSide {
	if !(s.Width >= 0.0) {
		panic(dart.NewAssertionError("width >= 0.0"))
	}
	return s
}

// BorderSideNone is BorderSide.none: a hairline black border that is not
// rendered.
var BorderSideNone = BorderSide{Color: borderSideDefaultColor, Width: 0.0, Style: BorderStyleNone,
	StrokeAlign: BorderSideStrokeAlignInside}

func (s BorderSide) isNone() bool { return s.Style == BorderStyleNone && s.Width == 0.0 }

// BorderSideMerge is BorderSide.merge: a side whose width is the sum of
// both. It panics unless BorderSideCanMerge(a, b), as Dart asserts.
func BorderSideMerge(a, b BorderSide) BorderSide {
	if !BorderSideCanMerge(a, b) {
		panic(dart.NewAssertionError("canMerge(a, b)"))
	}
	aIsNone, bIsNone := a.isNone(), b.isNone()
	if aIsNone && bIsNone {
		return BorderSideNone
	}
	if aIsNone {
		return b
	}
	if bIsNone {
		return a
	}
	return BorderSide{Color: a.Color, Width: a.Width + b.Width,
		StrokeAlign: math.Max(a.StrokeAlign, b.StrokeAlign), Style: a.Style}.checked()
}

// BorderSideCopyWithOptions are copyWith's named parameters (nil: keep).
type BorderSideCopyWithOptions = BorderSideOptions

// CopyWith returns a copy of this side with the given fields replaced.
func (s BorderSide) CopyWith(opts ...BorderSideCopyWithOptions) BorderSide {
	var o BorderSideCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Color != nil {
		s.Color = *o.Color
	}
	if o.Width != nil {
		s.Width = *o.Width
	}
	if o.Style != nil {
		s.Style = *o.Style
	}
	if o.StrokeAlign != nil {
		s.StrokeAlign = *o.StrokeAlign
	}
	return s.checked()
}

// Scale returns a copy with the width scaled by t (the style becomes none
// when t <= 0). As in Flutter, the strokeAlign is not kept: it takes the
// default, strokeAlignInside.
func (s BorderSide) Scale(t float64) BorderSide {
	style := s.Style
	if t <= 0.0 {
		style = BorderStyleNone
	}
	return BorderSide{Color: s.Color, Width: math.Max(0.0, s.Width*t), Style: style,
		StrokeAlign: BorderSideStrokeAlignInside}.checked()
}

// ToPaint creates a Paint object that, when used to draw a line, draws it
// as this side describes.
func (s BorderSide) ToPaint() *ui.Paint {
	p := ui.NewPaint()
	switch s.Style {
	case BorderStyleSolid:
		p.SetColor(s.Color)
		p.SetStrokeWidth(s.Width)
		p.SetStyle(ui.PaintingStyleStroke)
	default:
		p.SetColor(ui.NewColor(0x00000000))
		p.SetStrokeWidth(0.0)
		p.SetStyle(ui.PaintingStyleStroke)
	}
	return p
}

// BorderSideCanMerge reports whether the two sides can be merged with
// BorderSideMerge.
func BorderSideCanMerge(a, b BorderSide) bool {
	if a.isNone() || b.isNone() {
		return true
	}
	return a.Style == b.Style && a.Color == b.Color
}

func borderLerpDouble(a, b, t float64) float64 { return *ui.LerpDouble(&a, &b, t) }

func borderLerpColor(a, b ui.Color, t float64) ui.Color { return *ui.ColorLerp(&a, &b, t) }

// BorderSideLerp is BorderSide.lerp: linearly interpolates between two
// border sides.
func BorderSideLerp(a, b BorderSide, t float64) BorderSide {
	if a == b {
		return a
	}
	if t == 0.0 {
		return a
	}
	if t == 1.0 {
		return b
	}
	width := borderLerpDouble(a.Width, b.Width, t)
	if width < 0.0 {
		return BorderSideNone
	}
	if a.Style == b.Style && a.StrokeAlign == b.StrokeAlign {
		return BorderSide{Color: borderLerpColor(a.Color, b.Color, t), Width: width, Style: a.Style,
			StrokeAlign: a.StrokeAlign}.checked()
	}
	colorA, colorB := a.Color, b.Color
	if a.Style == BorderStyleNone {
		colorA = a.Color.WithAlpha(0x00)
	}
	if b.Style == BorderStyleNone {
		colorB = b.Color.WithAlpha(0x00)
	}
	if a.StrokeAlign != b.StrokeAlign {
		return BorderSide{Color: borderLerpColor(colorA, colorB, t), Width: width, Style: BorderStyleSolid,
			StrokeAlign: borderLerpDouble(a.StrokeAlign, b.StrokeAlign, t)}.checked()
	}
	return BorderSide{Color: borderLerpColor(colorA, colorB, t), Width: width, Style: BorderStyleSolid,
		StrokeAlign: a.StrokeAlign}.checked()
}

// StrokeInset is the amount of the stroke width that lies inside of the
// BorderSide.
func (s BorderSide) StrokeInset() float64 { return s.Width * (1 - (1+s.StrokeAlign)/2) }

// StrokeOutset is the amount of the stroke width that lies outside of the
// BorderSide.
func (s BorderSide) StrokeOutset() float64 { return s.Width * (1 + s.StrokeAlign) / 2 }

// StrokeOffset is the offset of the stroke, taking into account the
// stroke alignment.
func (s BorderSide) StrokeOffset() float64 { return s.Width * s.StrokeAlign }

// ToStringShort is 'BorderSide'.
func (s BorderSide) ToStringShort() string { return "BorderSide" }

// String is Diagnosticable's toString.
func (s BorderSide) String() string { return s.ToString() }

// ToString is Diagnosticable's toString({minLevel}).
func (s BorderSide) ToString(opts ...foundation.DiagnosticableToStringOptions) string {
	return diagnosticableValueToString(s, opts)
}

// ToDiagnosticsNode is a DiagnosticableNode describing this side.
func (s BorderSide) ToDiagnosticsNode(opts ...foundation.ToDiagnosticsNodeOptions) foundation.DiagnosticsNode {
	var o foundation.DiagnosticableNodeOptions
	if len(opts) > 0 {
		o = foundation.DiagnosticableNodeOptions(opts[0])
	}
	return foundation.NewDiagnosticableNode[foundation.Diagnosticable](s, o)
}

// DebugFillProperties adds color, width, strokeAlign and style.
func (s BorderSide) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	properties.Add(foundation.NewDiagnosticsProperty("color", s.Color,
		foundation.DiagnosticsPropertyOptions{DefaultValue: borderSideDefaultColor}))
	properties.Add(foundation.NewDoubleProperty("width", s.Width, foundation.DoublePropertyOptions{DefaultValue: 1.0}))
	properties.Add(foundation.NewDoubleProperty("strokeAlign", s.StrokeAlign,
		foundation.DoublePropertyOptions{DefaultValue: BorderSideStrokeAlignInside}))
	properties.Add(foundation.NewEnumProperty("style", s.Style,
		foundation.EnumPropertyOptions{DefaultValue: BorderStyleSolid}))
}

// diagnosticableValueToString is Diagnosticable.toString for value types
// that implement Diagnosticable without the mixin (asserts are always on,
// so the single-line description is used).
func diagnosticableValueToString(d foundation.Diagnosticable, opts []foundation.DiagnosticableToStringOptions) string {
	var o foundation.DiagnosticableToStringOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	style := foundation.DiagnosticsTreeStyleSingleLine
	return d.ToDiagnosticsNode(foundation.ToDiagnosticsNodeOptions{Style: &style}).
		ToString(foundation.DiagnosticsNodeToStringOptions{MinLevel: o.MinLevel})
}

// ShapeBorder is the base of shape outlines, including Border (used by
// BoxDecoration), CircleBorder, RoundedRectangleBorder and the others in
// this package. Embed ShapeBorderBase (and call SetSelf) to get Dart's
// default members.
//
// Dart's optional named textDirection is a positional *ui.TextDirection
// (nil is null). Dart's protected add({reversed}) is AddBorder; operator +
// is Add. A nil ShapeBorder (or a nil result) is Dart's null.
type ShapeBorder interface {
	// Dimensions is the widths of the sides of this border, as insets.
	Dimensions() EdgeInsetsGeometry
	// AddBorder is Dart's add(other, {reversed}): a border combining this
	// one and other, or nil when they can't be merged into one.
	AddBorder(other ShapeBorder, reversed bool) ShapeBorder
	// Add is operator +: AddBorder, or a compound border of both.
	Add(other ShapeBorder) ShapeBorder
	// Scale returns a copy of this border, scaled by t.
	Scale(t float64) ShapeBorder
	// LerpFrom interpolates from a (nil: null) to this border, or returns
	// nil when it doesn't know how.
	LerpFrom(a ShapeBorder, t float64) ShapeBorder
	// LerpTo interpolates from this border to b, or returns nil.
	LerpTo(b ShapeBorder, t float64) ShapeBorder
	// GetOuterPath is the outer edge of the border.
	GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path
	// GetInnerPath is the inner edge of the border.
	GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path
	// HitTest reports whether position is inside the outer path.
	HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool
	// PaintInterior fills the outer path with paint (when
	// PreferPaintInterior is true).
	PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection)
	// PreferPaintInterior reports whether PaintInterior is implemented and
	// preferred to filling GetOuterPath.
	PreferPaintInterior() bool
	// Paint paints the border within the given rect.
	Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection)
	String() string
}

// OutlinedBorder is a ShapeBorder that draws an outline with the width
// and color specified by its side. Its `side` getter is GetSide (Flutter's
// borders keep it in a field), and copyWith({side}) is CopyWithSide (each
// border's own CopyWith returns its concrete type).
type OutlinedBorder interface {
	ShapeBorder
	// GetSide is the border outline's color and weight.
	GetSide() BorderSide
	// CopyWithSide is copyWith({side}) (nil: keep the side).
	CopyWithSide(side *BorderSide) OutlinedBorder
}

// identicalBorders is Dart's identical(a, b) for shape borders: Go's ==
// when the dynamic type is comparable (Flutter's const borders are
// canonicalized, so equal values are identical), never panicking.
func identicalBorders(a, b ShapeBorder) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	if ta != tb || !ta.Comparable() {
		return false
	}
	return a == b
}

// ShapeBorderEqual is Dart's == on two shape borders (nil is null): the
// border's Equal(ShapeBorder) bool when it has one, Go's == otherwise.
func ShapeBorderEqual(a, b ShapeBorder) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if e, ok := a.(interface{ Equal(ShapeBorder) bool }); ok {
		return e.Equal(b)
	}
	return identicalBorders(a, b)
}

// shapeBorderPlus is ShapeBorder's operator +.
func shapeBorderPlus(self, other ShapeBorder) ShapeBorder {
	if r := self.AddBorder(other, false); r != nil {
		return r
	}
	if r := other.AddBorder(self, true); r != nil {
		return r
	}
	return newCompoundBorder([]ShapeBorder{other, self})
}

// shapeBorderLerpFrom and shapeBorderLerpTo are ShapeBorder's (and
// OutlinedBorder's) default lerpFrom and lerpTo.
func shapeBorderLerpFrom(self, a ShapeBorder, t float64) ShapeBorder {
	if a == nil {
		return self.Scale(t)
	}
	return nil
}

func shapeBorderLerpTo(self, b ShapeBorder, t float64) ShapeBorder {
	if b == nil {
		return self.Scale(1.0 - t)
	}
	return nil
}

func firstNonNil(fs ...func() ShapeBorder) ShapeBorder {
	for _, f := range fs {
		if r := f(); r != nil {
			return r
		}
	}
	return nil
}

// ShapeBorderLerp is ShapeBorder.lerp: linearly interpolates between two
// shape borders (nil is null), through LerpFrom/LerpTo, or switches from
// a to b at t = 0.5 when neither knows how.
func ShapeBorderLerp(a, b ShapeBorder, t float64) ShapeBorder {
	if identicalBorders(a, b) {
		return a
	}
	result := shapeBorderLerpChain(a, b, t)
	if result != nil {
		return result
	}
	if t < 0.5 {
		return a
	}
	return b
}

func shapeBorderLerpChain(a, b ShapeBorder, t float64) ShapeBorder {
	return firstNonNil(
		func() ShapeBorder {
			if b == nil {
				return nil
			}
			return b.LerpFrom(a, t)
		},
		func() ShapeBorder {
			if a == nil {
				return nil
			}
			return a.LerpTo(b, t)
		},
		func() ShapeBorder {
			if b == nil {
				return nil
			}
			return b.LerpTo(a, 1.0-t)
		},
		func() ShapeBorder {
			if a == nil {
				return nil
			}
			return a.LerpFrom(b, 1.0-t)
		})
}

// OutlinedBorderLerp is OutlinedBorder.lerp (nil is null); like
// ShapeBorderLerp, but the result must be an OutlinedBorder (Dart's cast
// throws otherwise; Go panics).
func OutlinedBorderLerp(a, b OutlinedBorder, t float64) OutlinedBorder {
	var sa, sb ShapeBorder
	if a != nil {
		sa = a
	}
	if b != nil {
		sb = b
	}
	if identicalBorders(sa, sb) {
		return a
	}
	if result := shapeBorderLerpChain(sa, sb, t); result != nil {
		o, ok := result.(OutlinedBorder)
		if !ok {
			panic(dart.NewAssertionError("type '" + shapeBorderTypeName(result) + "' is not a subtype of type 'OutlinedBorder?' in type cast"))
		}
		return o
	}
	if t < 0.5 {
		return a
	}
	return b
}

// shapeBorderTypeName is a border's Dart runtimeType name.
func shapeBorderTypeName(b ShapeBorder) string {
	switch x := b.(type) {
	case *compoundBorder:
		return "_CompoundBorder"
	case shapeToCircleBorder:
		if x.superellipse {
			return "_RoundedSuperellipseToCircleBorder"
		}
		return "_RoundedRectangleToCircleBorder"
	case stadiumToCircleBorder:
		return "_StadiumToCircleBorder"
	case stadiumToRoundedRectangleBorder:
		return "_StadiumToRoundedRectangleBorder"
	}
	t := reflect.TypeOf(b)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

// shapeBorderHitTest is ShapeBorder's default hitTest.
func shapeBorderHitTest(self ShapeBorder, rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return self.GetOuterPath(rect, textDirection).Contains(position)
}

// shapeBorderPaintInterior is ShapeBorder's default paintInterior, which
// asserts it is never called.
func shapeBorderPaintInterior(self ShapeBorder) {
	name := shapeBorderTypeName(self)
	if self.PreferPaintInterior() {
		panic(dart.NewAssertionError(name + ".preferPaintInterior returns true but " + name +
			".paintInterior is not implemented."))
	}
	panic(dart.NewAssertionError(name + ".preferPaintInterior returns false, so it is an error to call its paintInterior method."))
}

// ShapeBorderBase is the embeddable base of ShapeBorder implementations
// outside this package (Dart: extending ShapeBorder). Call SetSelf with the
// outer border when creating it; the outer type implements Dimensions,
// Scale, GetOuterPath, GetInnerPath and Paint, and may override the rest.
type ShapeBorderBase struct{ self ShapeBorder }

// SetSelf points the base at the border it is embedded in.
func (b *ShapeBorderBase) SetSelf(self ShapeBorder) { b.self = self }

func (b *ShapeBorderBase) outer() ShapeBorder {
	if b.self == nil {
		panic("painting: ShapeBorderBase used before SetSelf; call SetSelf with the outer border when creating it")
	}
	return b.self
}

// AddBorder returns nil (Dart's default add).
func (b *ShapeBorderBase) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (b *ShapeBorderBase) Add(other ShapeBorder) ShapeBorder {
	return shapeBorderPlus(b.outer(), other)
}

// LerpFrom scales this border by t when a is nil, else returns nil.
func (b *ShapeBorderBase) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	return shapeBorderLerpFrom(b.outer(), a, t)
}

// LerpTo scales this border by 1 - t when b is nil, else returns nil.
func (b *ShapeBorderBase) LerpTo(other ShapeBorder, t float64) ShapeBorder {
	return shapeBorderLerpTo(b.outer(), other, t)
}

// HitTest reports whether position is inside GetOuterPath.
func (b *ShapeBorderBase) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return shapeBorderHitTest(b.outer(), rect, position, textDirection)
}

// PaintInterior panics: borders that prefer painting their interior
// override it.
func (b *ShapeBorderBase) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	shapeBorderPaintInterior(b.outer())
}

// PreferPaintInterior is false.
func (b *ShapeBorderBase) PreferPaintInterior() bool { return false }

// String is objectRuntimeType(this, 'ShapeBorder') + '()'.
func (b *ShapeBorderBase) String() string {
	return foundation.ObjectRuntimeType(b.outer(), "ShapeBorder") + "()"
}

// outlinedDimensions is OutlinedBorder's dimensions.
func outlinedDimensions(side BorderSide) EdgeInsetsGeometry {
	return NewEdgeInsetsAll(math.Max(side.StrokeInset(), 0))
}

// compoundBorder is the private _CompoundBorder: borders painted one
// inside the other (a + b is [b, a]). It is used by pointer and compares
// with Equal (Dart's listEquals).
type compoundBorder struct{ borders []ShapeBorder }

func newCompoundBorder(borders []ShapeBorder) *compoundBorder {
	if len(borders) < 2 {
		panic(dart.NewAssertionError("borders.length >= 2"))
	}
	for _, b := range borders {
		if _, ok := b.(*compoundBorder); ok {
			panic(dart.NewAssertionError("!borders.any((ShapeBorder border) => border is _CompoundBorder)"))
		}
	}
	return &compoundBorder{borders}
}

func (c *compoundBorder) Dimensions() EdgeInsetsGeometry {
	var d EdgeInsetsGeometry = EdgeInsetsZero
	for _, b := range c.borders {
		d = d.AddGeometry(b.Dimensions())
	}
	return d
}

func (c *compoundBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder {
	otherCompound, otherIsCompound := other.(*compoundBorder)
	if !otherIsCompound {
		ours := c.borders[0]
		if reversed {
			ours = c.borders[len(c.borders)-1]
		}
		merged := ours.AddBorder(other, reversed)
		if merged == nil {
			merged = other.AddBorder(ours, !reversed)
		}
		if merged != nil {
			result := append([]ShapeBorder(nil), c.borders...)
			if reversed {
				result[len(result)-1] = merged
			} else {
				result[0] = merged
			}
			return newCompoundBorder(result)
		}
	}
	var merged []ShapeBorder
	if reversed {
		merged = append(merged, c.borders...)
	}
	if otherIsCompound {
		merged = append(merged, otherCompound.borders...)
	} else {
		merged = append(merged, other)
	}
	if !reversed {
		merged = append(merged, c.borders...)
	}
	return newCompoundBorder(merged)
}

func (c *compoundBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(c, other) }

func (c *compoundBorder) Scale(t float64) ShapeBorder {
	out := make([]ShapeBorder, len(c.borders))
	for i, b := range c.borders {
		out[i] = b.Scale(t)
	}
	return newCompoundBorder(out)
}

func (c *compoundBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	return compoundBorderLerp(a, c, t)
}

func (c *compoundBorder) LerpTo(b ShapeBorder, t float64) ShapeBorder {
	return compoundBorderLerp(c, b, t)
}

func compoundBorderLerp(a, b ShapeBorder, t float64) *compoundBorder {
	ac, aok := a.(*compoundBorder)
	bc, bok := b.(*compoundBorder)
	if !aok && !bok {
		panic(dart.NewAssertionError("a is _CompoundBorder || b is _CompoundBorder"))
	}
	aList := []ShapeBorder{a}
	if aok {
		aList = ac.borders
	}
	bList := []ShapeBorder{b}
	if bok {
		bList = bc.borders
	}
	var results []ShapeBorder
	length := max(len(aList), len(bList))
	for index := 0; index < length; index++ {
		var localA, localB ShapeBorder
		if index < len(aList) {
			localA = aList[index]
		}
		if index < len(bList) {
			localB = bList[index]
		}
		if localA != nil && localB != nil {
			localResult := localA.LerpTo(localB, t)
			if localResult == nil {
				localResult = localB.LerpFrom(localA, t)
			}
			if localResult != nil {
				results = append(results, localResult)
				continue
			}
		}
		if localB != nil {
			results = append(results, localB.Scale(t))
		}
		if localA != nil {
			results = append(results, localA.Scale(1.0-t))
		}
	}
	return newCompoundBorder(results)
}

func (c *compoundBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	for index := 0; index < len(c.borders)-1; index++ {
		rect = c.borders[index].Dimensions().Resolve(textDirection).DeflateRect(rect)
	}
	return c.borders[len(c.borders)-1].GetInnerPath(rect, textDirection)
}

func (c *compoundBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return c.borders[0].GetOuterPath(rect, textDirection)
}

func (c *compoundBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return c.borders[0].HitTest(rect, position, textDirection)
}

func (c *compoundBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	c.borders[0].PaintInterior(canvas, rect, paint, textDirection)
}

func (c *compoundBorder) PreferPaintInterior() bool {
	for _, b := range c.borders {
		if !b.PreferPaintInterior() {
			return false
		}
	}
	return true
}

func (c *compoundBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	for _, b := range c.borders {
		b.Paint(canvas, rect, textDirection)
		rect = b.Dimensions().Resolve(textDirection).DeflateRect(rect)
	}
}

// Equal is Dart's ==: the same borders in the same order.
func (c *compoundBorder) Equal(other ShapeBorder) bool {
	o, ok := other.(*compoundBorder)
	if !ok {
		return false
	}
	if c == o {
		return true
	}
	if len(c.borders) != len(o.borders) {
		return false
	}
	for i := range c.borders {
		if !ShapeBorderEqual(o.borders[i], c.borders[i]) {
			return false
		}
	}
	return true
}

func (c *compoundBorder) String() string {
	parts := make([]string, len(c.borders))
	for i, b := range c.borders {
		parts[len(c.borders)-1-i] = b.String()
	}
	return strings.Join(parts, " + ")
}

// PaintBorderOptions are paintBorder's named sides (nil: BorderSide.none).
type PaintBorderOptions struct {
	Top, Right, Bottom, Left *BorderSide
}

func sideOrNone(s *BorderSide) BorderSide {
	if s == nil {
		return BorderSideNone
	}
	return *s
}

// PaintBorder paints a border around the given rectangle on the canvas:
// each solid side as a filled trapezoid, or a hairline stroke when its
// width is 0.
func PaintBorder(canvas ui.Canvas, rect ui.Rect, opts ...PaintBorderOptions) {
	var o PaintBorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	top, right, bottom, left := sideOrNone(o.Top), sideOrNone(o.Right), sideOrNone(o.Bottom), sideOrNone(o.Left)
	paint := ui.NewPaint()
	paint.SetStrokeWidth(0.0)
	path := ui.NewPath()

	if top.Style == BorderStyleSolid {
		paint.SetColor(top.Color)
		path.Reset()
		path.MoveTo(rect.Left, rect.Top)
		path.LineTo(rect.Right, rect.Top)
		if top.Width == 0.0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Right-right.Width, rect.Top+top.Width)
			path.LineTo(rect.Left+left.Width, rect.Top+top.Width)
		}
		canvas.DrawPath(path, paint)
	}
	if right.Style == BorderStyleSolid {
		paint.SetColor(right.Color)
		path.Reset()
		path.MoveTo(rect.Right, rect.Top)
		path.LineTo(rect.Right, rect.Bottom)
		if right.Width == 0.0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Right-right.Width, rect.Bottom-bottom.Width)
			path.LineTo(rect.Right-right.Width, rect.Top+top.Width)
		}
		canvas.DrawPath(path, paint)
	}
	if bottom.Style == BorderStyleSolid {
		paint.SetColor(bottom.Color)
		path.Reset()
		path.MoveTo(rect.Right, rect.Bottom)
		path.LineTo(rect.Left, rect.Bottom)
		if bottom.Width == 0.0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Left+left.Width, rect.Bottom-bottom.Width)
			path.LineTo(rect.Right-right.Width, rect.Bottom-bottom.Width)
		}
		canvas.DrawPath(path, paint)
	}
	if left.Style == BorderStyleSolid {
		paint.SetColor(left.Color)
		path.Reset()
		path.MoveTo(rect.Left, rect.Bottom)
		path.LineTo(rect.Left, rect.Top)
		if left.Width == 0.0 {
			paint.SetStyle(ui.PaintingStyleStroke)
		} else {
			paint.SetStyle(ui.PaintingStyleFill)
			path.LineTo(rect.Left+left.Width, rect.Top+top.Width)
			path.LineTo(rect.Left+left.Width, rect.Bottom-bottom.Width)
		}
		canvas.DrawPath(path, paint)
	}
}
