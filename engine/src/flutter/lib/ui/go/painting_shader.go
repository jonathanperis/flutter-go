package ui

// Port of painting.dart's Shader, Gradient and ImageShader (Flutter
// 3.50.1), with the engine's gradient.cc and image_shader.cc mapped onto
// the Impeller SDK's color sources.

import (
	"math"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// Shader is a base type for objects that create the color of a paint
// (Gradient, ImageShader, FragmentShader). Dart's base class has a private
// constructor, so only dart:ui's types implement it.
type Shader interface {
	// Dispose releases the shader's resources. Using it afterwards panics
	// in debug builds.
	Dispose()
	// DebugDisposed reports whether Dispose was called (debug builds only).
	DebugDisposed() bool
	// impellerColorSource returns the Impeller color source the shader
	// draws with, owned by the shader (callers don't release it); nil when
	// it can't be created.
	impellerColorSource() *impeller.ColorSource
}

// shaderBase is Dart's Shader base class state.
type shaderBase struct{ debugDisposed bool }

// DebugDisposed is Shader.debugDisposed.
func (s *shaderBase) DebugDisposed() bool { return s.debugDisposed }

func (s *shaderBase) dispose() {
	if debugMode {
		if s.debugDisposed {
			panic(dart.NewAssertionError("A Shader cannot be disposed more than once."))
		}
		s.debugDisposed = true
	}
}

func argumentError(message string) dart.ArgumentError { return dart.ArgumentError{Message: message} }

func dartAssert(cond bool, message string) {
	if !cond {
		panic(dart.NewAssertionError(message))
	}
}

func offsetIsValid(o Offset) {
	dartAssert(!math.IsNaN(o.Dx) && !math.IsNaN(o.Dy), "Offset argument contained a NaN value.")
}

// imageDebugDisposed is image.debugDisposed (false when unknown).
func imageDebugDisposed(image *Image) bool {
	if d, ok := any(image).(interface{ DebugDisposed() bool }); ok {
		return d.DebugDisposed()
	}
	return false
}

func rectIsValid(r Rect) { dartAssert(!r.HasNaN(), "Rect argument contained a NaN value.") }

// ---- Gradient ----

// Gradient is a shader that renders a color gradient (linear, radial,
// two-point conical or sweep).
type Gradient struct {
	shaderBase
	kind     int // 0 linear, 1 radial, 2 conical, 3 sweep
	p0, p1   impeller.Point
	r0, r1   float32
	start    float32 // sweep, degrees
	end      float32
	colors   []impeller.Color
	stops    []float32 // nil: evenly spaced
	tile     TileMode
	matrix   *impeller.Matrix
	native   *impeller.ColorSource
	released bool
}

// GradientLinearOptions are Gradient.linear's optional positional
// arguments [colorStops, tileMode, matrix4].
type GradientLinearOptions struct {
	ColorStops []float64
	TileMode   TileMode
	Matrix4    []float64
}

// GradientRadialOptions are Gradient.radial's optional positional
// arguments [colorStops, tileMode, matrix4, focal, focalRadius].
type GradientRadialOptions struct {
	ColorStops  []float64
	TileMode    TileMode
	Matrix4     []float64
	Focal       *Offset
	FocalRadius float64
}

// GradientSweepOptions are Gradient.sweep's optional positional arguments
// [colorStops, tileMode, startAngle, endAngle, matrix4].
type GradientSweepOptions struct {
	ColorStops []float64
	TileMode   TileMode
	StartAngle float64
	EndAngle   *float64 // nil: 2π
	Matrix4    []float64
}

func validateColorStops(colors []Color, colorStops []float64) {
	if colorStops == nil {
		if len(colors) != 2 {
			panic(argumentError(`"colors" must have length 2 if "colorStops" is omitted.`))
		}
	} else if len(colors) != len(colorStops) {
		panic(argumentError(`"colors" and "colorStops" arguments must have equal length.`))
	}
}

// encodeWideColorList is _encodeWideColorList: colors in extended sRGB,
// as float32.
func encodeWideColorList(colors []Color) []impeller.Color {
	out := make([]impeller.Color, len(colors))
	xr := ColorSpaceExtendedSRGB
	for i, c := range colors {
		c = c.WithValues(ColorWithValuesOptions{ColorSpace: &xr})
		out[i] = impeller.Color{R: float32(c.R), G: float32(c.G), B: float32(c.B), A: float32(c.A),
			Space: int(ColorSpaceExtendedSRGB)}
	}
	return out
}

func encodeStops(stops []float64) []float32 {
	if stops == nil {
		return nil
	}
	out := make([]float32, len(stops))
	for i, s := range stops {
		out[i] = float32(s)
	}
	return out
}

// toImpellerMatrix is ToDlMatrix: a column-major Float64List narrowed to
// float32.
func toImpellerMatrix(m []float64) *impeller.Matrix {
	if m == nil {
		return nil
	}
	var r impeller.Matrix
	for i := range r {
		r[i] = float32(m[i])
	}
	return &r
}

// NewGradientLinear is Gradient.linear(from, to, colors, [colorStops,
// tileMode, matrix4]).
func NewGradientLinear(from, to Offset, colors []Color, opts ...GradientLinearOptions) *Gradient {
	var o GradientLinearOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	offsetIsValid(from)
	offsetIsValid(to)
	if o.Matrix4 != nil {
		matrix4IsValid(o.Matrix4)
	}
	validateColorStops(colors, o.ColorStops)
	return &Gradient{kind: 0, p0: impeller.Point{X: float32(from.Dx), Y: float32(from.Dy)},
		p1: impeller.Point{X: float32(to.Dx), Y: float32(to.Dy)}, colors: encodeWideColorList(colors),
		stops: encodeStops(o.ColorStops), tile: o.TileMode, matrix: toImpellerMatrix(o.Matrix4)}
}

// NewGradientRadial is Gradient.radial(center, radius, colors,
// [colorStops, tileMode, matrix4, focal, focalRadius]); with a focal
// point it is a two-point conical gradient.
func NewGradientRadial(center Offset, radius float64, colors []Color, opts ...GradientRadialOptions) *Gradient {
	var o GradientRadialOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	offsetIsValid(center)
	if o.Matrix4 != nil {
		matrix4IsValid(o.Matrix4)
	}
	validateColorStops(colors, o.ColorStops)
	g := &Gradient{colors: encodeWideColorList(colors), stops: encodeStops(o.ColorStops), tile: o.TileMode,
		matrix: toImpellerMatrix(o.Matrix4)}
	if o.Focal == nil || (*o.Focal == center && o.FocalRadius == 0.0) {
		g.kind = 1
		g.p0 = impeller.Point{X: float32(center.Dx), Y: float32(center.Dy)}
		g.r0 = float32(radius)
	} else {
		dartAssert(center != (Offset{}) || *o.Focal != (Offset{}), "")
		g.kind = 2
		g.p0 = impeller.Point{X: float32(o.Focal.Dx), Y: float32(o.Focal.Dy)}
		g.r0 = float32(o.FocalRadius)
		g.p1 = impeller.Point{X: float32(center.Dx), Y: float32(center.Dy)}
		g.r1 = float32(radius)
	}
	return g
}

// NewGradientSweep is Gradient.sweep(center, colors, [colorStops,
// tileMode, startAngle, endAngle, matrix4]).
func NewGradientSweep(center Offset, colors []Color, opts ...GradientSweepOptions) *Gradient {
	var o GradientSweepOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	end := math.Pi * 2
	if o.EndAngle != nil {
		end = *o.EndAngle
	}
	offsetIsValid(center)
	dartAssert(o.StartAngle < end, "")
	if o.Matrix4 != nil {
		matrix4IsValid(o.Matrix4)
	}
	validateColorStops(colors, o.ColorStops)
	// gradient.cc: SafeNarrow(angle) * 180.0f / static_cast<float>(M_PI).
	deg := func(a float64) float32 { return float32(float32(a)*180.0) / float32(math.Pi) }
	return &Gradient{kind: 3, p0: impeller.Point{X: float32(center.Dx), Y: float32(center.Dy)},
		start: deg(o.StartAngle), end: deg(end), colors: encodeWideColorList(colors),
		stops: encodeStops(o.ColorStops), tile: o.TileMode, matrix: toImpellerMatrix(o.Matrix4)}
}

// Dispose is Shader.dispose.
func (g *Gradient) Dispose() {
	g.dispose()
	if g.native != nil {
		g.native.Release()
		g.native = nil
	}
	g.released = true
}

func (g *Gradient) impellerColorSource() *impeller.ColorSource {
	if g.native != nil || g.released || len(g.colors) == 0 {
		return g.native
	}
	stops := g.stops
	if stops == nil { // evenly spaced, as DlGradientColorSourceBase does without stops
		stops = make([]float32, len(g.colors))
		for i := range stops {
			stops[i] = float32(i) / float32(len(g.colors)-1)
		}
	}
	tile := impellerTileMode(g.tile)
	switch g.kind {
	case 0:
		g.native = impeller.NewLinearGradient(g.p0, g.p1, g.colors, stops, tile, g.matrix)
	case 1:
		g.native = impeller.NewRadialGradient(g.p0, g.r0, g.colors, stops, tile, g.matrix)
	case 2:
		g.native = impeller.NewConicalGradient(g.p0, g.r0, g.p1, g.r1, g.colors, stops, tile, g.matrix)
	case 3:
		g.native = impeller.NewSweepGradient(g.p0, g.start, g.end, g.colors, stops, tile, g.matrix)
	}
	return g.native
}

// ---- ImageShader ----

// ImageShader is a shader that tiles an image.
type ImageShader struct {
	shaderBase
	image         *Image
	tmx, tmy      TileMode
	matrix        impeller.Matrix
	locked        bool // a filterQuality was given
	filterQuality FilterQuality
	native        map[int]*impeller.ColorSource // by sampling
}

// ImageShaderOptions are ImageShader's named arguments.
type ImageShaderOptions struct {
	FilterQuality *FilterQuality // nil: follow the paint's filterQuality
}

// NewImageShader is ImageShader(image, tmx, tmy, matrix4, filterQuality:).
func NewImageShader(image *Image, tmx, tmy TileMode, matrix4 []float64, opts ...ImageShaderOptions) *ImageShader {
	dartAssert(!imageDebugDisposed(image), "")
	if len(matrix4) != 16 {
		panic(argumentError(`"matrix4" must have 16 entries.`))
	}
	s := &ImageShader{image: image, tmx: tmx, tmy: tmy, matrix: *toImpellerMatrix(matrix4)}
	if len(opts) > 0 && opts[0].FilterQuality != nil {
		s.locked, s.filterQuality = true, *opts[0].FilterQuality
	}
	return s
}

// Dispose is ImageShader.dispose.
func (s *ImageShader) Dispose() {
	s.dispose()
	for _, n := range s.native {
		n.Release()
	}
	s.native = nil
}

func (s *ImageShader) impellerColorSource() *impeller.ColorSource {
	return s.colorSourceWithQuality(FilterQualityLow) // kLinear when not locked
}

// colorSourceWithQuality is ImageShader::shader(sampling): the paint's
// filter quality, unless the shader was created with its own.
func (s *ImageShader) colorSourceWithQuality(q FilterQuality) *impeller.ColorSource {
	if s.locked {
		q = s.filterQuality
	}
	sampling := samplingFromQuality(q)
	if n, ok := s.native[sampling]; ok {
		return n
	}
	tex := s.image.impellerTexture()
	if tex == nil {
		return nil
	}
	if s.native == nil {
		s.native = map[int]*impeller.ColorSource{}
	}
	m := s.matrix
	n := impeller.NewImageColorSource(tex, impellerTileMode(s.tmx), impellerTileMode(s.tmy), sampling, &m)
	s.native[sampling] = n
	return n
}
