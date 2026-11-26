package painting

// Port of gradient.dart (Flutter 3.50.1): GradientTransform,
// GradientRotation, Gradient, LinearGradient, RadialGradient and
// SweepGradient.
//
// Gradient is an abstract class users extend: it is the interface
// Gradient plus the embeddable GradientBase (colors, stops, transform and
// the default fromColor/lerpFrom/lerpTo, which reach the outer gradient
// through self). The three concrete gradients are used by pointer and built
// with NewLinearGradient(LinearGradientOptions{...}) etc.

import (
	"math"
	"sort"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// gradientSample is _sample: the color at t of a gradient's colors/stops.
func gradientSample(colors []ui.Color, stops []float64, t float64) ui.Color {
	if len(colors) == 0 || len(stops) == 0 {
		panic(dart.NewAssertionError(nil))
	}
	if t <= stops[0] {
		return colors[0]
	}
	if t >= stops[len(stops)-1] {
		return colors[len(colors)-1]
	}
	index := -1
	for i := len(stops) - 1; i >= 0; i-- {
		if stops[i] <= t {
			index = i
			break
		}
	}
	if index == -1 {
		panic(dart.NewAssertionError(nil))
	}
	return *ui.ColorLerp(&colors[index], &colors[index+1], (t-stops[index])/(stops[index+1]-stops[index]))
}

// interpolateColorsAndStops is _interpolateColorsAndStops: both gradients
// sampled at the union of their stops (a SplayTreeSet: sorted, without
// duplicates), lerped at t.
func interpolateColorsAndStops(aColors []ui.Color, aStops []float64, bColors []ui.Color, bStops []float64,
	t float64) ([]ui.Color, []float64) {
	if len(aColors) < 2 || len(bColors) < 2 || len(aStops) != len(aColors) || len(bStops) != len(bColors) {
		panic(dart.NewAssertionError(nil))
	}
	// SplayTreeSet<double> orders with compareTo (-0.0 < 0.0, NaN last)
	// and treats compareTo == 0 as the same element.
	var stops []float64
	for _, s := range append(append([]float64{}, aStops...), bStops...) {
		i := sort.Search(len(stops), func(i int) bool { return dart.CompareDoubles(stops[i], s) >= 0 })
		if i < len(stops) && dart.CompareDoubles(stops[i], s) == 0 {
			continue
		}
		stops = append(stops, 0)
		copy(stops[i+1:], stops[i:])
		stops[i] = s
	}
	colors := make([]ui.Color, len(stops))
	for i, stop := range stops {
		a, b := gradientSample(aColors, aStops, stop), gradientSample(bColors, bStops, stop)
		colors[i] = *ui.ColorLerp(&a, &b, t)
	}
	return colors, stops
}

// GradientTransform is a base for gradient transforms: Transform returns
// the matrix applied to a gradient's shader for the given bounds (nil:
// none). Dart's named textDirection is a positional *ui.TextDirection (nil
// is null).
type GradientTransform interface {
	Transform(bounds ui.Rect, textDirection *ui.TextDirection) *vm.Matrix4
}

// GradientRotation is a GradientTransform that rotates the gradient
// around the center of its bounds. Comparable.
type GradientRotation struct {
	// Radians is the angle, clockwise, of the rotation.
	Radians float64
}

// NewGradientRotation is GradientRotation(radians).
func NewGradientRotation(radians float64) GradientRotation { return GradientRotation{radians} }

// Transform is the rotation about bounds' center.
func (g GradientRotation) Transform(bounds ui.Rect, textDirection *ui.TextDirection) *vm.Matrix4 {
	sinRadians := dart.Sin(g.Radians)
	oneMinusCosRadians := 1 - dart.Cos(g.Radians)
	center := bounds.Center()
	originX := float64(sinRadians*center.Dy) + float64(oneMinusCosRadians*center.Dx)
	originY := float64(-sinRadians*center.Dx) + float64(oneMinusCosRadians*center.Dy)
	m := vm.NewMatrix4Identity()
	m.TranslateByDouble(originX, originY, 0, 1)
	m.RotateZ(g.Radians)
	return m
}

func (g GradientRotation) String() string {
	r := g.Radians
	return foundation.ObjectRuntimeType(g, "GradientRotation") + "(radians: " + foundation.DebugFormatDouble(&r) + ")"
}

// gradientTransformsEqual is == on GradientTransform? values: an Equal
// method, else Go's == (identity for pointers; uncomparable values are
// unequal unless the same interface value).
func gradientTransformsEqual(a, b GradientTransform) bool {
	if e, ok := a.(interface {
		Equal(other GradientTransform) bool
	}); ok {
		return e.Equal(b)
	}
	return sameObject(a, b)
}

// Gradient is a 2D gradient: the interface of Dart's abstract Gradient. A
// gradient Go users write embeds GradientBase.
type Gradient interface {
	// Colors are the colors the gradient should obtain at each of the stops.
	Colors() []ui.Color
	// Stops are the stops (0.0 to 1.0) of the colors; nil is null (evenly
	// spaced).
	Stops() []float64
	// Transform is the transform applied to the shader (nil: none).
	Transform() GradientTransform
	// CreateShader creates a shader filling rect (textDirection nil is
	// null).
	CreateShader(rect ui.Rect, textDirection *ui.TextDirection) ui.Shader
	// Scale returns a gradient with each color scaled by factor (0: fully
	// transparent).
	Scale(factor float64) Gradient
	// WithOpacity returns a gradient with each color's opacity set.
	WithOpacity(opacity float64) Gradient
	// FromColor returns a gradient of the same type with every color
	// replaced by color.
	FromColor(color ui.Color) Gradient
	// LerpFrom is Dart's @protected lerpFrom (a nil: null); nil result is
	// null.
	LerpFrom(a Gradient, t float64) Gradient
	// LerpTo is Dart's @protected lerpTo.
	LerpTo(b Gradient, t float64) Gradient
	String() string
}

// GradientBase is the state and default methods of Dart's abstract
// Gradient. A type embedding it calls Init (Dart's super constructor).
type GradientBase struct {
	colors    []ui.Color
	stops     []float64
	transform GradientTransform
	self      Gradient
}

// Init sets up the base: self is the outer gradient, through which the
// defaults call Scale.
func (g *GradientBase) Init(self Gradient, colors []ui.Color, stops []float64, transform GradientTransform) {
	g.self, g.colors, g.stops, g.transform = self, colors, stops, transform
}

func (g *GradientBase) outer() Gradient {
	if g.self == nil {
		panic("painting: GradientBase used before Init")
	}
	return g.self
}

// Colors are the gradient's colors.
func (g *GradientBase) Colors() []ui.Color { return g.colors }

// Stops are the gradient's stops (nil is null).
func (g *GradientBase) Stops() []float64 { return g.stops }

// Transform is the gradient's transform (nil is null).
func (g *GradientBase) Transform() GradientTransform { return g.transform }

// impliedStops is _impliedStops: the stops, or evenly spaced ones.
func (g *GradientBase) impliedStops() []float64 {
	if g.stops != nil {
		return g.stops
	}
	if len(g.colors) < 2 {
		panic(dart.NewAssertionError("colors list must have at least two colors"))
	}
	separation := 1.0 / float64(len(g.colors)-1)
	out := make([]float64, len(g.colors))
	for i := range out {
		out[i] = float64(float64(i) * separation)
	}
	return out
}

// resolveTransform is _resolveTransform: the transform's storage, or nil.
func (g *GradientBase) resolveTransform(bounds ui.Rect, textDirection *ui.TextDirection) []float64 {
	if g.transform == nil {
		return nil
	}
	m := g.transform.Transform(bounds, textDirection)
	if m == nil {
		return nil
	}
	return m.Storage()
}

// FromColor is Gradient.fromColor's default: a LinearGradient of the same
// stops and transform with every color replaced.
func (g *GradientBase) FromColor(color ui.Color) Gradient {
	return NewLinearGradient(LinearGradientOptions{Colors: filledColors(len(g.colors), color), Stops: g.stops,
		Transform: g.transform})
}

// LerpFrom is the default lerpFrom: the outer gradient scaled by t when a
// is null, else null.
func (g *GradientBase) LerpFrom(a Gradient, t float64) Gradient {
	if a == nil {
		return g.outer().Scale(t)
	}
	return nil
}

// LerpTo is the default lerpTo: the outer gradient scaled by 1 - t when b
// is null, else null.
func (g *GradientBase) LerpTo(b Gradient, t float64) Gradient {
	if b == nil {
		return g.outer().Scale(1.0 - t)
	}
	return nil
}

func filledColors(n int, c ui.Color) []ui.Color {
	out := make([]ui.Color, n)
	for i := range out {
		out[i] = c
	}
	return out
}

func scaledColors(colors []ui.Color, factor float64) []ui.Color {
	out := make([]ui.Color, len(colors))
	for i := range colors {
		out[i] = *ui.ColorLerp(nil, &colors[i], factor)
	}
	return out
}

func isNilGradient(g Gradient) bool { return isNilValue(g) }

// GradientLerp is Gradient.lerp: linearly interpolates between two
// gradients (nil is null).
func GradientLerp(a, b Gradient, t float64) Gradient {
	if isNilGradient(a) {
		a = nil
	}
	if isNilGradient(b) {
		b = nil
	}
	if sameObject(a, b) {
		return a
	}
	var result Gradient
	if b != nil {
		result = b.LerpFrom(a, t) // if a is null, this must return non-null
	}
	if isNilGradient(result) && a != nil {
		result = a.LerpTo(b, t) // if b is null, this must return non-null
	}
	if !isNilGradient(result) {
		return result
	}
	if a == nil || b == nil {
		panic(dart.NewAssertionError(nil))
	}
	if t < 0.5 {
		return a.Scale(1.0 - float64((t * 2.0)))
	}
	return b.Scale((t - 0.5) * 2.0)
}

func colorsString(colors []ui.Color) string {
	parts := make([]string, len(colors))
	for i, c := range colors {
		parts[i] = c.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func doublesString(values []float64) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = dart.DoubleToString(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func transformString(t GradientTransform) string {
	if s, ok := t.(interface{ String() string }); ok {
		return s.String()
	}
	return "Instance of '" + foundation.ObjectRuntimeType(t, "GradientTransform") + "'"
}

func colorListsEqual(a, b []ui.Color) bool { return foundation.ListEquals(a, b) }

func alignmentGeometryEqual(a, b AlignmentGeometry) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func formatDouble(v float64) string { return foundation.DebugFormatDouble(&v) }

// ---- LinearGradient ----

// LinearGradient is a 2D linear gradient.
type LinearGradient struct {
	GradientBase
	// Begin is the offset at which stop 0.0 of the gradient is placed.
	Begin AlignmentGeometry
	// End is the offset at which stop 1.0 of the gradient is placed.
	End AlignmentGeometry
	// TileMode is how the gradient repeats outside begin..end.
	TileMode ui.TileMode
}

// LinearGradientOptions are LinearGradient's named parameters (nil Begin:
// Alignment.centerLeft; nil End: Alignment.centerRight; zero TileMode:
// TileMode.clamp).
type LinearGradientOptions struct {
	Begin     AlignmentGeometry
	End       AlignmentGeometry
	Colors    []ui.Color
	Stops     []float64
	TileMode  ui.TileMode
	Transform GradientTransform
}

// NewLinearGradient is LinearGradient({...}).
func NewLinearGradient(o LinearGradientOptions) *LinearGradient {
	g := &LinearGradient{Begin: o.Begin, End: o.End, TileMode: o.TileMode}
	if g.Begin == nil {
		g.Begin = AlignmentCenterLeft
	}
	if g.End == nil {
		g.End = AlignmentCenterRight
	}
	g.Init(g, o.Colors, o.Stops, o.Transform)
	return g
}

// CreateShader is ui.Gradient.linear from begin to end resolved in rect.
func (g *LinearGradient) CreateShader(rect ui.Rect, textDirection *ui.TextDirection) ui.Shader {
	return ui.NewGradientLinear(g.Begin.Resolve(textDirection).WithinRect(rect), g.End.Resolve(textDirection).WithinRect(rect),
		g.colors, ui.GradientLinearOptions{ColorStops: g.impliedStops(), TileMode: g.TileMode,
			Matrix4: g.resolveTransform(rect, textDirection)})
}

func (g *LinearGradient) with(colors []ui.Color) *LinearGradient {
	return NewLinearGradient(LinearGradientOptions{Begin: g.Begin, End: g.End, Colors: colors, Stops: g.stops,
		TileMode: g.TileMode, Transform: g.transform})
}

// ScaleLinear is Scale with Dart's covariant LinearGradient result.
func (g *LinearGradient) ScaleLinear(factor float64) *LinearGradient {
	return g.with(scaledColors(g.colors, factor))
}

// Scale returns the gradient with each color lerped from transparent by
// factor.
func (g *LinearGradient) Scale(factor float64) Gradient { return g.ScaleLinear(factor) }

// FromColor returns the gradient with every color replaced by color.
func (g *LinearGradient) FromColor(color ui.Color) Gradient {
	return g.with(filledColors(len(g.colors), color))
}

// WithOpacity returns the gradient with each color's alpha set to opacity
// (Color.withValues).
func (g *LinearGradient) WithOpacity(opacity float64) Gradient {
	colors := make([]ui.Color, len(g.colors))
	for i, c := range g.colors {
		colors[i] = c.WithValues(ui.ColorWithValuesOptions{Alpha: &opacity})
	}
	return g.with(colors)
}

// LerpFrom lerps from a when it is a LinearGradient or null.
func (g *LinearGradient) LerpFrom(a Gradient, t float64) Gradient {
	if isNilGradient(a) {
		return nilIfNilLinear(LinearGradientLerp(nil, g, t))
	}
	if la, ok := a.(*LinearGradient); ok {
		return nilIfNilLinear(LinearGradientLerp(la, g, t))
	}
	return g.GradientBase.LerpFrom(a, t)
}

// LerpTo lerps to b when it is a LinearGradient or null.
func (g *LinearGradient) LerpTo(b Gradient, t float64) Gradient {
	if isNilGradient(b) {
		return nilIfNilLinear(LinearGradientLerp(g, nil, t))
	}
	if lb, ok := b.(*LinearGradient); ok {
		return nilIfNilLinear(LinearGradientLerp(g, lb, t))
	}
	return g.GradientBase.LerpTo(b, t)
}

func nilIfNilLinear(g *LinearGradient) Gradient {
	if g == nil {
		return nil
	}
	return g
}

// LinearGradientLerp is LinearGradient.lerp (nil is null).
func LinearGradientLerp(a, b *LinearGradient, t float64) *LinearGradient {
	if a == b {
		return a
	}
	if a == nil {
		return b.ScaleLinear(t)
	}
	if b == nil {
		return a.ScaleLinear(1.0 - t)
	}
	colors, stops := interpolateColorsAndStops(a.colors, a.impliedStops(), b.colors, b.impliedStops(), t)
	tileMode, transform := b.TileMode, b.transform
	if t < 0.5 {
		tileMode, transform = a.TileMode, a.transform
	}
	return NewLinearGradient(LinearGradientOptions{Begin: AlignmentGeometryLerp(a.Begin, b.Begin, t),
		End: AlignmentGeometryLerp(a.End, b.End, t), Colors: colors, Stops: stops, TileMode: tileMode,
		Transform: transform})
}

// Equal is ==.
func (g *LinearGradient) Equal(other Gradient) bool {
	o, ok := other.(*LinearGradient)
	if !ok || o == nil {
		return false
	}
	return g == o || (alignmentGeometryEqual(o.Begin, g.Begin) && alignmentGeometryEqual(o.End, g.End) &&
		o.TileMode == g.TileMode && gradientTransformsEqual(o.transform, g.transform) &&
		colorListsEqual(o.colors, g.colors) && foundation.ListEquals(o.stops, g.stops))
}

func (g *LinearGradient) String() string {
	d := []string{"begin: " + g.Begin.String(), "end: " + g.End.String(), "colors: " + colorsString(g.colors)}
	if g.stops != nil {
		d = append(d, "stops: "+doublesString(g.stops))
	}
	d = append(d, "tileMode: "+g.TileMode.String())
	if g.transform != nil {
		d = append(d, "transform: "+transformString(g.transform))
	}
	return foundation.ObjectRuntimeType(g, "LinearGradient") + "(" + strings.Join(d, ", ") + ")"
}

// ---- RadialGradient ----

// RadialGradient is a 2D radial gradient.
type RadialGradient struct {
	GradientBase
	// Center is the center of the gradient.
	Center AlignmentGeometry
	// Radius is the radius, as a fraction of the shortest side of the
	// painted box.
	Radius float64
	// TileMode is how the gradient repeats outside the radius.
	TileMode ui.TileMode
	// Focal is the focal point of the gradient (nil: the center).
	Focal AlignmentGeometry
	// FocalRadius is the radius of the focal point, as a fraction of the
	// shortest side.
	FocalRadius float64
}

// RadialGradientOptions are RadialGradient's named parameters (nil Center:
// Alignment.center; nil Radius: 0.5).
type RadialGradientOptions struct {
	Center      AlignmentGeometry
	Radius      *float64
	Colors      []ui.Color
	Stops       []float64
	TileMode    ui.TileMode
	Focal       AlignmentGeometry
	FocalRadius float64
	Transform   GradientTransform
}

// NewRadialGradient is RadialGradient({...}).
func NewRadialGradient(o RadialGradientOptions) *RadialGradient {
	g := &RadialGradient{Center: o.Center, Radius: 0.5, TileMode: o.TileMode, Focal: o.Focal, FocalRadius: o.FocalRadius}
	if g.Center == nil {
		g.Center = AlignmentCenter
	}
	if o.Radius != nil {
		g.Radius = *o.Radius
	}
	g.Init(g, o.Colors, o.Stops, o.Transform)
	return g
}

// CreateShader is ui.Gradient.radial (two-point conical with a focal
// point) resolved in rect.
func (g *RadialGradient) CreateShader(rect ui.Rect, textDirection *ui.TextDirection) ui.Shader {
	var focal *ui.Offset
	if g.Focal != nil {
		f := g.Focal.Resolve(textDirection).WithinRect(rect)
		focal = &f
	}
	return ui.NewGradientRadial(g.Center.Resolve(textDirection).WithinRect(rect), g.Radius*rect.ShortestSide(),
		g.colors, ui.GradientRadialOptions{ColorStops: g.impliedStops(), TileMode: g.TileMode,
			Matrix4: g.resolveTransform(rect, textDirection), Focal: focal,
			FocalRadius: g.FocalRadius * rect.ShortestSide()})
}

func (g *RadialGradient) with(colors []ui.Color) *RadialGradient {
	r := g.Radius
	return NewRadialGradient(RadialGradientOptions{Center: g.Center, Radius: &r, Colors: colors, Stops: g.stops,
		TileMode: g.TileMode, Focal: g.Focal, FocalRadius: g.FocalRadius, Transform: g.transform})
}

// ScaleRadial is Scale with Dart's covariant RadialGradient result.
func (g *RadialGradient) ScaleRadial(factor float64) *RadialGradient {
	return g.with(scaledColors(g.colors, factor))
}

// Scale returns the gradient with each color lerped from transparent by
// factor.
func (g *RadialGradient) Scale(factor float64) Gradient { return g.ScaleRadial(factor) }

// FromColor returns the gradient with every color replaced by color.
func (g *RadialGradient) FromColor(color ui.Color) Gradient {
	return g.with(filledColors(len(g.colors), color))
}

// WithOpacity returns the gradient with each color's opacity set
// (the deprecated Color.withOpacity, as Flutter does).
func (g *RadialGradient) WithOpacity(opacity float64) Gradient {
	colors := make([]ui.Color, len(g.colors))
	for i, c := range g.colors {
		colors[i] = c.WithOpacity(opacity)
	}
	return g.with(colors)
}

// LerpFrom lerps from a when it is a RadialGradient or null.
func (g *RadialGradient) LerpFrom(a Gradient, t float64) Gradient {
	if isNilGradient(a) {
		return nilIfNilRadial(RadialGradientLerp(nil, g, t))
	}
	if ra, ok := a.(*RadialGradient); ok {
		return nilIfNilRadial(RadialGradientLerp(ra, g, t))
	}
	return g.GradientBase.LerpFrom(a, t)
}

// LerpTo lerps to b when it is a RadialGradient or null.
func (g *RadialGradient) LerpTo(b Gradient, t float64) Gradient {
	if isNilGradient(b) {
		return nilIfNilRadial(RadialGradientLerp(g, nil, t))
	}
	if rb, ok := b.(*RadialGradient); ok {
		return nilIfNilRadial(RadialGradientLerp(g, rb, t))
	}
	return g.GradientBase.LerpTo(b, t)
}

func nilIfNilRadial(g *RadialGradient) Gradient {
	if g == nil {
		return nil
	}
	return g
}

// RadialGradientLerp is RadialGradient.lerp (nil is null).
func RadialGradientLerp(a, b *RadialGradient, t float64) *RadialGradient {
	if a == b {
		return a
	}
	if a == nil {
		return b.ScaleRadial(t)
	}
	if b == nil {
		return a.ScaleRadial(1.0 - t)
	}
	colors, stops := interpolateColorsAndStops(a.colors, a.impliedStops(), b.colors, b.impliedStops(), t)
	tileMode, transform := b.TileMode, b.transform
	if t < 0.5 {
		tileMode, transform = a.TileMode, a.transform
	}
	radius := math.Max(0.0, lerpD(a.Radius, b.Radius, t))
	return NewRadialGradient(RadialGradientOptions{Center: AlignmentGeometryLerp(a.Center, b.Center, t),
		Radius: &radius, Colors: colors, Stops: stops, TileMode: tileMode,
		Focal: AlignmentGeometryLerp(a.Focal, b.Focal, t), FocalRadius: math.Max(0.0, lerpD(a.FocalRadius, b.FocalRadius, t)),
		Transform: transform})
}

// Equal is ==.
func (g *RadialGradient) Equal(other Gradient) bool {
	o, ok := other.(*RadialGradient)
	if !ok || o == nil {
		return false
	}
	return g == o || (alignmentGeometryEqual(o.Center, g.Center) && o.Radius == g.Radius &&
		o.TileMode == g.TileMode && gradientTransformsEqual(o.transform, g.transform) &&
		colorListsEqual(o.colors, g.colors) && foundation.ListEquals(o.stops, g.stops) &&
		alignmentGeometryEqual(o.Focal, g.Focal) && o.FocalRadius == g.FocalRadius)
}

func (g *RadialGradient) String() string {
	d := []string{"center: " + g.Center.String(), "radius: " + formatDouble(g.Radius), "colors: " + colorsString(g.colors)}
	if g.stops != nil {
		d = append(d, "stops: "+doublesString(g.stops))
	}
	d = append(d, "tileMode: "+g.TileMode.String())
	if g.Focal != nil {
		d = append(d, "focal: "+g.Focal.String())
	}
	d = append(d, "focalRadius: "+formatDouble(g.FocalRadius))
	if g.transform != nil {
		d = append(d, "transform: "+transformString(g.transform))
	}
	return foundation.ObjectRuntimeType(g, "RadialGradient") + "(" + strings.Join(d, ", ") + ")"
}

// ---- SweepGradient ----

// SweepGradient is a 2D sweep gradient (a conic gradient).
type SweepGradient struct {
	GradientBase
	// Center is the center of the gradient.
	Center AlignmentGeometry
	// StartAngle is the angle in radians at which stop 0.0 is placed.
	StartAngle float64
	// EndAngle is the angle in radians at which stop 1.0 is placed.
	EndAngle float64
	// TileMode is how the gradient repeats outside the angles.
	TileMode ui.TileMode
}

// SweepGradientOptions are SweepGradient's named parameters (nil Center:
// Alignment.center; nil EndAngle: math.pi * 2).
type SweepGradientOptions struct {
	Center     AlignmentGeometry
	StartAngle float64
	EndAngle   *float64
	Colors     []ui.Color
	Stops      []float64
	TileMode   ui.TileMode
	Transform  GradientTransform
}

// NewSweepGradient is SweepGradient({...}).
func NewSweepGradient(o SweepGradientOptions) *SweepGradient {
	g := &SweepGradient{Center: o.Center, StartAngle: o.StartAngle, EndAngle: math.Pi * 2, TileMode: o.TileMode}
	if g.Center == nil {
		g.Center = AlignmentCenter
	}
	if o.EndAngle != nil {
		g.EndAngle = *o.EndAngle
	}
	g.Init(g, o.Colors, o.Stops, o.Transform)
	return g
}

// CreateShader is ui.Gradient.sweep resolved in rect.
func (g *SweepGradient) CreateShader(rect ui.Rect, textDirection *ui.TextDirection) ui.Shader {
	end := g.EndAngle
	return ui.NewGradientSweep(g.Center.Resolve(textDirection).WithinRect(rect), g.colors,
		ui.GradientSweepOptions{ColorStops: g.impliedStops(), TileMode: g.TileMode, StartAngle: g.StartAngle,
			EndAngle: &end, Matrix4: g.resolveTransform(rect, textDirection)})
}

func (g *SweepGradient) with(colors []ui.Color) *SweepGradient {
	end := g.EndAngle
	return NewSweepGradient(SweepGradientOptions{Center: g.Center, StartAngle: g.StartAngle, EndAngle: &end,
		Colors: colors, Stops: g.stops, TileMode: g.TileMode, Transform: g.transform})
}

// ScaleSweep is Scale with Dart's covariant SweepGradient result.
func (g *SweepGradient) ScaleSweep(factor float64) *SweepGradient {
	return g.with(scaledColors(g.colors, factor))
}

// Scale returns the gradient with each color lerped from transparent by
// factor.
func (g *SweepGradient) Scale(factor float64) Gradient { return g.ScaleSweep(factor) }

// FromColor returns the gradient with every color replaced by color.
func (g *SweepGradient) FromColor(color ui.Color) Gradient {
	return g.with(filledColors(len(g.colors), color))
}

// WithOpacity returns the gradient with each color's opacity set
// (the deprecated Color.withOpacity, as Flutter does).
func (g *SweepGradient) WithOpacity(opacity float64) Gradient {
	colors := make([]ui.Color, len(g.colors))
	for i, c := range g.colors {
		colors[i] = c.WithOpacity(opacity)
	}
	return g.with(colors)
}

// LerpFrom lerps from a when it is a SweepGradient or null.
func (g *SweepGradient) LerpFrom(a Gradient, t float64) Gradient {
	if isNilGradient(a) {
		return nilIfNilSweep(SweepGradientLerp(nil, g, t))
	}
	if sa, ok := a.(*SweepGradient); ok {
		return nilIfNilSweep(SweepGradientLerp(sa, g, t))
	}
	return g.GradientBase.LerpFrom(a, t)
}

// LerpTo lerps to b when it is a SweepGradient or null.
func (g *SweepGradient) LerpTo(b Gradient, t float64) Gradient {
	if isNilGradient(b) {
		return nilIfNilSweep(SweepGradientLerp(g, nil, t))
	}
	if sb, ok := b.(*SweepGradient); ok {
		return nilIfNilSweep(SweepGradientLerp(g, sb, t))
	}
	return g.GradientBase.LerpTo(b, t)
}

func nilIfNilSweep(g *SweepGradient) Gradient {
	if g == nil {
		return nil
	}
	return g
}

// SweepGradientLerp is SweepGradient.lerp (nil is null).
func SweepGradientLerp(a, b *SweepGradient, t float64) *SweepGradient {
	if a == b {
		return a
	}
	if a == nil {
		return b.ScaleSweep(t)
	}
	if b == nil {
		return a.ScaleSweep(1.0 - t)
	}
	colors, stops := interpolateColorsAndStops(a.colors, a.impliedStops(), b.colors, b.impliedStops(), t)
	tileMode, transform := b.TileMode, b.transform
	if t < 0.5 {
		tileMode, transform = a.TileMode, a.transform
	}
	end := math.Max(0.0, lerpD(a.EndAngle, b.EndAngle, t))
	return NewSweepGradient(SweepGradientOptions{Center: AlignmentGeometryLerp(a.Center, b.Center, t),
		StartAngle: math.Max(0.0, lerpD(a.StartAngle, b.StartAngle, t)), EndAngle: &end, Colors: colors,
		Stops: stops, TileMode: tileMode, Transform: transform})
}

// Equal is ==.
func (g *SweepGradient) Equal(other Gradient) bool {
	o, ok := other.(*SweepGradient)
	if !ok || o == nil {
		return false
	}
	return g == o || (alignmentGeometryEqual(o.Center, g.Center) && o.StartAngle == g.StartAngle &&
		o.EndAngle == g.EndAngle && o.TileMode == g.TileMode && gradientTransformsEqual(o.transform, g.transform) &&
		colorListsEqual(o.colors, g.colors) && foundation.ListEquals(o.stops, g.stops))
}

func (g *SweepGradient) String() string {
	d := []string{"center: " + g.Center.String(), "startAngle: " + formatDouble(g.StartAngle),
		"endAngle: " + formatDouble(g.EndAngle), "colors: " + colorsString(g.colors)}
	if g.stops != nil {
		d = append(d, "stops: "+doublesString(g.stops))
	}
	d = append(d, "tileMode: "+g.TileMode.String())
	if g.transform != nil {
		d = append(d, "transform: "+transformString(g.transform))
	}
	return foundation.ObjectRuntimeType(g, "SweepGradient") + "(" + strings.Join(d, ", ") + ")"
}

// GradientsEqual is == on Gradient? values: the gradient's Equal when it
// has one, else identity.
func GradientsEqual(a, b Gradient) bool {
	if isNilGradient(a) || isNilGradient(b) {
		return isNilGradient(a) && isNilGradient(b)
	}
	if e, ok := a.(interface{ Equal(other Gradient) bool }); ok {
		return e.Equal(b)
	}
	return sameObject(a, b)
}
