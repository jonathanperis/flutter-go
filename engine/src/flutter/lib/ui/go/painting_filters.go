package ui

// Port of painting.dart's MaskFilter, ColorFilter and ImageFilter
// (Flutter 3.50.1), with the engine's color_filter.cc and image_filter.cc
// mapped onto the Impeller SDK.

import (
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// ImageFilter is a filter applied to the pixels of a layer or a draw
// (blur, dilate, erode, matrix, compose, shader; a ColorFilter is one too).
// Dart's abstract class has a private constructor, so only dart:ui's types
// implement it.
type ImageFilter interface {
	String() string
	// DebugShortDescription is ImageFilter.debugShortDescription.
	DebugShortDescription() string
	// impellerImageFilter returns the Impeller filter, owned by the filter
	// (callers don't release it); nil when it can't be created.
	impellerImageFilter() *impeller.ImageFilter
	// equalImageFilter is Dart's ==.
	equalImageFilter(other ImageFilter) bool
}

// ImageFilterEqual is Dart's == on two image filters (nil is null).
func ImageFilterEqual(a, b ImageFilter) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.equalImageFilter(b)
}

// ---- MaskFilter ----

// MaskFilter is a filter applied to the coverage of a shape (a blur).
// Comparable: == is Dart's ==.
type MaskFilter struct {
	style BlurStyle
	sigma float64
}

// NewMaskFilterBlur is MaskFilter.blur(style, sigma).
func NewMaskFilterBlur(style BlurStyle, sigma float64) MaskFilter {
	return MaskFilter{style: style, sigma: sigma}
}

func (m MaskFilter) String() string {
	return "MaskFilter.blur(" + m.style.String() + ", " + dart.ToStringAsFixed(m.sigma, 1) + ")"
}

// ---- ColorFilter ----

type colorFilterKind int

const (
	colorFilterMode colorFilterKind = iota + 1
	colorFilterMatrix
	colorFilterLinearToSrgbGamma
	colorFilterSrgbToLinearGamma
)

// ColorFilter is a filter applied to the colors of a draw. It is also an
// ImageFilter. Use it by pointer; == is Equal.
type ColorFilter struct {
	kind      colorFilterKind
	color     Color
	blendMode BlendMode
	matrix    []float64
	native    *impeller.ColorFilter
	nativeInv *impeller.ColorFilter
	nativeIF  *impeller.ImageFilter
}

// NewColorFilterMode is ColorFilter.mode(color, blendMode).
func NewColorFilterMode(color Color, blendMode BlendMode) *ColorFilter {
	return &ColorFilter{kind: colorFilterMode, color: color, blendMode: blendMode}
}

// NewColorFilterMatrix is ColorFilter.matrix(matrix): a 5x4 row-major
// matrix whose fifth column (the translation) is in 0..255.
func NewColorFilterMatrix(matrix []float64) *ColorFilter {
	return &ColorFilter{kind: colorFilterMatrix, matrix: matrix}
}

// NewColorFilterLinearToSrgbGamma is ColorFilter.linearToSrgbGamma().
func NewColorFilterLinearToSrgbGamma() *ColorFilter {
	return &ColorFilter{kind: colorFilterLinearToSrgbGamma}
}

// NewColorFilterSrgbToLinearGamma is ColorFilter.srgbToLinearGamma().
func NewColorFilterSrgbToLinearGamma() *ColorFilter {
	return &ColorFilter{kind: colorFilterSrgbToLinearGamma}
}

// NewColorFilterSaturation is ColorFilter.saturation(saturation).
func NewColorFilterSaturation(saturation float64) *ColorFilter {
	const rLuminance = 0.2126
	const gLuminance = 0.7152
	const bLuminance = 0.0722
	invSat := 1 - saturation
	// float64(...) keeps each product rounded (no FMA), as in Dart.
	return NewColorFilterMatrix([]float64{
		float64(invSat*rLuminance) + saturation, float64(invSat * gLuminance), float64(invSat * bLuminance), 0, 0,
		float64(invSat * rLuminance), float64(invSat*gLuminance) + saturation, float64(invSat * bLuminance), 0, 0,
		float64(invSat * rLuminance), float64(invSat * gLuminance), float64(invSat*bLuminance) + saturation, 0, 0,
		0, 0, 0, 1, 0,
	})
}

func listEquals(a, b []float64) bool {
	if a == nil {
		return b == nil
	}
	if b == nil || len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Equal is Dart's ==.
func (f *ColorFilter) Equal(other *ColorFilter) bool {
	if f == nil || other == nil {
		return f == other
	}
	return other.kind == f.kind && listEquals(other.matrix, f.matrix) && other.color == f.color &&
		other.blendMode == f.blendMode
}

func (f *ColorFilter) equalImageFilter(other ImageFilter) bool {
	o, ok := other.(*ColorFilter)
	return ok && f.Equal(o)
}

func doubleListString(l []float64) string {
	parts := make([]string, len(l))
	for i, v := range l {
		parts[i] = dart.DoubleToString(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// DebugShortDescription is ColorFilter.debugShortDescription.
func (f *ColorFilter) DebugShortDescription() string {
	switch f.kind {
	case colorFilterMode, colorFilterMatrix, colorFilterLinearToSrgbGamma, colorFilterSrgbToLinearGamma:
		return f.String()
	}
	return "unknow ColorFilter"
}

func (f *ColorFilter) String() string {
	switch f.kind {
	case colorFilterMode:
		return "ColorFilter.mode(" + f.color.String() + ", " + f.blendMode.String() + ")"
	case colorFilterMatrix:
		if f.matrix == nil {
			return "ColorFilter.matrix(null)"
		}
		return "ColorFilter.matrix(" + doubleListString(f.matrix) + ")"
	case colorFilterLinearToSrgbGamma:
		return "ColorFilter.linearToSrgbGamma()"
	case colorFilterSrgbToLinearGamma:
		return "ColorFilter.srgbToLinearGamma()"
	}
	return "Unknown ColorFilter type. This is an error. If you're seeing this, please file an issue at https://github.com/flutter/flutter/issues/new."
}

// engineMatrix is color_filter.cc's initMatrix: the Dart matrix narrowed
// to float32 with the translation column scaled from 0..255 to 0..1
// (`matrix[4] *= 1.0f / 255`, in float).
func (f *ColorFilter) engineMatrix() [20]float32 {
	if len(f.matrix) != 20 {
		panic(dart.NewAssertionError("Color Matrix must have 20 entries."))
	}
	var m [20]float32
	for i, v := range f.matrix {
		m[i] = float32(v)
	}
	one, d255 := float32(1.0), float32(255)
	inv := one / d255 // C++ folds 1.0f / 255 in float: the same rounding
	for _, i := range []int{4, 9, 14, 19} {
		m[i] = float32(m[i] * inv)
	}
	return m
}

// impellerColorFilter is the engine's DlColorFilter as an Impeller color
// filter. ENGINE GAP: the SDK has no gamma color filters, so
// linearToSrgbGamma and srgbToLinearGamma draw unfiltered.
func (f *ColorFilter) impellerColorFilter() *impeller.ColorFilter {
	if f.native != nil {
		return f.native
	}
	switch f.kind {
	case colorFilterMode:
		// initMode passes color.value (8-bit ARGB) to DlColor.
		v := uint32(f.color.Value())
		c := impeller.Color{A: float32(v>>24) / 255, R: float32(v>>16&0xff) / 255,
			G: float32(v>>8&0xff) / 255, B: float32(v&0xff) / 255}
		f.native = impeller.NewBlendColorFilter(c, int(f.blendMode))
	case colorFilterMatrix:
		f.native = impeller.NewMatrixColorFilter(f.engineMatrix())
	}
	return f.native
}

// invertedImpellerColorFilter is the matrix filter followed by
// Paint.invertColors' inversion, as one matrix (inversion × matrix).
func (f *ColorFilter) invertedImpellerColorFilter() *impeller.ColorFilter {
	if f.nativeInv == nil {
		f.nativeInv = impeller.NewMatrixColorFilter(concatColorMatrix(invertColorMatrix, f.engineMatrix()))
	}
	return f.nativeInv
}

// concatColorMatrix returns a∘b (b applied first) for 4x5 row-major color
// matrices with an implicit fifth row (0 0 0 0 1).
func concatColorMatrix(a, b [20]float32) [20]float32 {
	var r [20]float32
	for row := 0; row < 4; row++ {
		for col := 0; col < 5; col++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += float32(a[row*5+k] * b[k*5+col])
			}
			if col == 4 {
				s += a[row*5+4]
			}
			r[row*5+col] = s
		}
	}
	return r
}

func (f *ColorFilter) impellerImageFilter() *impeller.ImageFilter {
	// ENGINE GAP: the SDK has no color-filter image filter
	// (DlColorFilterImageFilter); a ColorFilter used as an image filter
	// draws unfiltered.
	return nil
}

// ---- ImageFilter implementations ----

type matrixImageFilter struct {
	data          []float64
	filterQuality FilterQuality
	native        *impeller.ImageFilter
}

type gaussianBlurImageFilter struct {
	sigmaX, sigmaY float64
	tileMode       *TileMode
	bounds         *Rect
	native         *impeller.ImageFilter
	nativeTile     map[TileMode]*impeller.ImageFilter
}

type dilateImageFilter struct {
	radiusX, radiusY float64
	native           *impeller.ImageFilter
}

type erodeImageFilter struct {
	radiusX, radiusY float64
	native           *impeller.ImageFilter
}

type composeImageFilter struct {
	inner, outer ImageFilter
	native       *impeller.ImageFilter
}

// ImageFilterBlurOptions are ImageFilter.blur's named arguments.
type ImageFilterBlurOptions struct {
	SigmaX, SigmaY float64
	TileMode       *TileMode // nil: unspecified (the engine picks per use)
	Bounds         *Rect
}

// NewImageFilterBlur is ImageFilter.blur(sigmaX:, sigmaY:, tileMode:, bounds:).
func NewImageFilterBlur(opts ...ImageFilterBlurOptions) ImageFilter {
	var o ImageFilterBlurOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &gaussianBlurImageFilter{sigmaX: o.SigmaX, sigmaY: o.SigmaY, tileMode: o.TileMode, bounds: o.Bounds}
}

// ImageFilterRadiusOptions are ImageFilter.dilate/erode's named arguments.
type ImageFilterRadiusOptions struct{ RadiusX, RadiusY float64 }

// NewImageFilterDilate is ImageFilter.dilate(radiusX:, radiusY:).
func NewImageFilterDilate(opts ...ImageFilterRadiusOptions) ImageFilter {
	var o ImageFilterRadiusOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &dilateImageFilter{radiusX: o.RadiusX, radiusY: o.RadiusY}
}

// NewImageFilterErode is ImageFilter.erode(radiusX:, radiusY:).
func NewImageFilterErode(opts ...ImageFilterRadiusOptions) ImageFilter {
	var o ImageFilterRadiusOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return &erodeImageFilter{radiusX: o.RadiusX, radiusY: o.RadiusY}
}

// ImageFilterMatrixOptions are ImageFilter.matrix's named arguments.
type ImageFilterMatrixOptions struct {
	FilterQuality *FilterQuality // nil: FilterQuality.medium
}

// NewImageFilterMatrix is ImageFilter.matrix(matrix4, filterQuality:).
func NewImageFilterMatrix(matrix4 []float64, opts ...ImageFilterMatrixOptions) ImageFilter {
	if len(matrix4) != 16 {
		panic(argumentError(`"matrix4" must have 16 entries.`))
	}
	fq := FilterQualityMedium
	if len(opts) > 0 && opts[0].FilterQuality != nil {
		fq = *opts[0].FilterQuality
	}
	return &matrixImageFilter{data: append([]float64(nil), matrix4...), filterQuality: fq}
}

// NewImageFilterCompose is ImageFilter.compose(outer:, inner:).
func NewImageFilterCompose(outer, inner ImageFilter) ImageFilter {
	return &composeImageFilter{inner: inner, outer: outer}
}

// ImageFilterIsShaderFilterSupported is ImageFilter.isShaderFilterSupported
// (true: rendering is always Impeller).
func ImageFilterIsShaderFilterSupported() bool { return true }

func (f *matrixImageFilter) DebugShortDescription() string {
	return "matrix(" + doubleListString(f.data) + ", " + f.filterQuality.String() + ")"
}
func (f *matrixImageFilter) String() string {
	return "ImageFilter.matrix(" + doubleListString(f.data) + ", " + f.filterQuality.String() + ")"
}
func (f *matrixImageFilter) equalImageFilter(other ImageFilter) bool {
	o, ok := other.(*matrixImageFilter)
	return ok && o.filterQuality == f.filterQuality && listEquals(o.data, f.data)
}

func (f *gaussianBlurImageFilter) modeString() string {
	if f.tileMode == nil {
		return "unspecified"
	}
	return f.tileMode.Name()
}
func (f *gaussianBlurImageFilter) boundsString() string {
	if f.bounds == nil {
		return ""
	}
	return ", bounds: " + f.bounds.String()
}
func (f *gaussianBlurImageFilter) DebugShortDescription() string {
	return "blur(" + dart.DoubleToString(f.sigmaX) + ", " + dart.DoubleToString(f.sigmaY) + ", " +
		f.modeString() + f.boundsString() + ")"
}
func (f *gaussianBlurImageFilter) String() string {
	return "ImageFilter.blur(" + dart.DoubleToString(f.sigmaX) + ", " + dart.DoubleToString(f.sigmaY) + ", " +
		f.modeString() + f.boundsString() + ")"
}
func (f *gaussianBlurImageFilter) equalImageFilter(other ImageFilter) bool {
	o, ok := other.(*gaussianBlurImageFilter)
	if !ok || o.sigmaX != f.sigmaX || o.sigmaY != f.sigmaY {
		return false
	}
	if (o.bounds == nil) != (f.bounds == nil) || (o.bounds != nil && *o.bounds != *f.bounds) {
		return false
	}
	return (o.tileMode == nil) == (f.tileMode == nil) && (o.tileMode == nil || *o.tileMode == *f.tileMode)
}

func (f *dilateImageFilter) DebugShortDescription() string {
	return "dilate(" + dart.DoubleToString(f.radiusX) + ", " + dart.DoubleToString(f.radiusY) + ")"
}
func (f *dilateImageFilter) String() string {
	return "ImageFilter.dilate(" + dart.DoubleToString(f.radiusX) + ", " + dart.DoubleToString(f.radiusY) + ")"
}
func (f *dilateImageFilter) equalImageFilter(other ImageFilter) bool {
	o, ok := other.(*dilateImageFilter)
	return ok && o.radiusX == f.radiusX && o.radiusY == f.radiusY
}

func (f *erodeImageFilter) DebugShortDescription() string {
	return "erode(" + dart.DoubleToString(f.radiusX) + ", " + dart.DoubleToString(f.radiusY) + ")"
}
func (f *erodeImageFilter) String() string {
	return "ImageFilter.erode(" + dart.DoubleToString(f.radiusX) + ", " + dart.DoubleToString(f.radiusY) + ")"
}
func (f *erodeImageFilter) equalImageFilter(other ImageFilter) bool {
	o, ok := other.(*erodeImageFilter)
	return ok && o.radiusX == f.radiusX && o.radiusY == f.radiusY
}

func (f *composeImageFilter) DebugShortDescription() string {
	return f.inner.DebugShortDescription() + " -> " + f.outer.DebugShortDescription()
}
func (f *composeImageFilter) String() string {
	return "ImageFilter.compose(source -> " + f.DebugShortDescription() + " -> result)"
}
func (f *composeImageFilter) equalImageFilter(other ImageFilter) bool {
	o, ok := other.(*composeImageFilter)
	return ok && ImageFilterEqual(o.inner, f.inner) && ImageFilterEqual(o.outer, f.outer)
}

// ---- the engine side (image_filter.cc) ----

// samplingFromQuality is ImageFilter::SamplingFromIndex mapped onto the
// SDK's two samplings (ENGINE GAP: no mipmaps or cubic sampling, so medium
// and high sample linearly).
func samplingFromQuality(q FilterQuality) int {
	if q == FilterQualityNone {
		return impeller.SamplingNearest
	}
	return impeller.SamplingLinear
}

func impellerTileMode(m TileMode) int {
	switch m {
	case TileModeRepeated:
		return impeller.TileRepeat
	case TileModeMirror:
		return impeller.TileMirror
	case TileModeDecal:
		return impeller.TileDecal
	}
	return impeller.TileClamp
}

// filterWithTileMode is ImageFilter::filter(tile_mode): a blur without a
// tile mode takes the one the use supplies (decal for draws and layers,
// clamp for backdrops and composition).
func filterWithTileMode(f ImageFilter, tile TileMode) *impeller.ImageFilter {
	if b, ok := f.(*gaussianBlurImageFilter); ok && b.tileMode == nil {
		return b.withTile(tile)
	}
	return f.impellerImageFilter()
}

func (f *gaussianBlurImageFilter) withTile(tile TileMode) *impeller.ImageFilter {
	// DlBlurImageFilter::Make returns null (no filter) for non-finite sigmas
	// or when both are nearly zero, and zeroes a nearly-zero one. ENGINE
	// GAP: the SDK's blur has no bounds.
	sx, sy := float32(f.sigmaX), float32(f.sigmaY)
	if isInfOrNaN32(sx) || isInfOrNaN32(sy) || (sx < scalarNearlyZero && sy < scalarNearlyZero) {
		return nil
	}
	if sx < scalarNearlyZero {
		sx = 0
	}
	if sy < scalarNearlyZero {
		sy = 0
	}
	if f.nativeTile == nil {
		f.nativeTile = map[TileMode]*impeller.ImageFilter{}
	}
	if n, ok := f.nativeTile[tile]; ok {
		return n
	}
	n := impeller.NewBlurImageFilter(sx, sy, impellerTileMode(tile))
	f.nativeTile[tile] = n
	return n
}

func (f *gaussianBlurImageFilter) impellerImageFilter() *impeller.ImageFilter {
	tile := TileModeClamp
	if f.tileMode != nil {
		tile = *f.tileMode
	}
	return f.withTile(tile)
}

func (f *dilateImageFilter) impellerImageFilter() *impeller.ImageFilter {
	if f.native == nil {
		rx, ry := float32(f.radiusX), float32(f.radiusY)
		if morphologyRadiusValid(rx, ry) {
			f.native = impeller.NewDilateImageFilter(rx, ry)
		}
	}
	return f.native
}

func (f *erodeImageFilter) impellerImageFilter() *impeller.ImageFilter {
	if f.native == nil {
		rx, ry := float32(f.radiusX), float32(f.radiusY)
		if morphologyRadiusValid(rx, ry) {
			f.native = impeller.NewErodeImageFilter(rx, ry)
		}
	}
	return f.native
}

// scalarNearlyZero is SK_ScalarNearlyZero (1 / 4096).
const scalarNearlyZero = float32(1.0 / (1 << 12))

// morphologyRadiusValid is DlDilateImageFilter/DlErodeImageFilter::Make's
// check: both radii finite and greater than SK_ScalarNearlyZero.
func morphologyRadiusValid(rx, ry float32) bool {
	return !isInfOrNaN32(rx) && rx > scalarNearlyZero && !isInfOrNaN32(ry) && ry > scalarNearlyZero
}

func (f *matrixImageFilter) impellerImageFilter() *impeller.ImageFilter {
	if f.native == nil {
		var m impeller.Matrix
		for i, v := range f.data {
			m[i] = float32(v)
		}
		f.native = impeller.NewMatrixImageFilter(m, samplingFromQuality(f.filterQuality))
	}
	return f.native
}

func (f *composeImageFilter) impellerImageFilter() *impeller.ImageFilter {
	if f.native == nil {
		outer := filterWithTileMode(f.outer, TileModeClamp)
		inner := filterWithTileMode(f.inner, TileModeClamp)
		switch {
		case outer == nil:
			return inner
		case inner == nil:
			return outer
		}
		f.native = impeller.NewComposeImageFilter(outer, inner)
	}
	return f.native
}
