package impeller

// Bindings for the rest of the SDK's drawing API (paths, paints, color
// sources, filters, textures and the display list builder's operations),
// used by dart:ui's Canvas.

/*
#include <stdlib.h>
#include <string.h>
#include "impeller.h"

static void fg_free(void* p) { free(p); }

static ImpellerTexture fg_texture_new(ImpellerContext ctx, int64_t w, int64_t h, const void* bytes, size_t n) {
  void* copy = malloc(n);
  memcpy(copy, bytes, n);
  ImpellerTextureDescriptor desc = {kImpellerPixelFormatRGBA8888, {w, h}, 1};
  ImpellerMapping m = {(const uint8_t*)copy, n, fg_free};
  ImpellerTexture t = ImpellerTextureCreateWithContentsNew(ctx, &desc, &m, copy);
  if (t == NULL) free(copy);
  return t;
}
*/
import "C"

import (
	"unsafe"
)

// Point is an ImpellerPoint.
type Point struct{ X, Y float32 }

func (p Point) c() C.ImpellerPoint { return C.ImpellerPoint{x: C.float(p.X), y: C.float(p.Y)} }

// Radii are the four corner radii of a rounded rectangle.
type Radii struct{ TopLeft, BottomLeft, TopRight, BottomRight Point }

func (r Radii) c() C.ImpellerRoundingRadii {
	return C.ImpellerRoundingRadii{top_left: r.TopLeft.c(), bottom_left: r.BottomLeft.c(),
		top_right: r.TopRight.c(), bottom_right: r.BottomRight.c()}
}

// Matrix is a 4x4 column-major matrix.
type Matrix [16]float32

func (m *Matrix) c() *C.ImpellerMatrix {
	if m == nil {
		return nil
	}
	var cm C.ImpellerMatrix
	for i, v := range m {
		cm.m[i] = C.float(v)
	}
	return &cm
}

// Enumerations (values are the C API's).
const (
	FillTypeNonZero = 0
	FillTypeOdd     = 1

	ClipDifference = 0
	ClipIntersect  = 1

	DrawStyleFill          = 0
	DrawStyleStroke        = 1
	DrawStyleStrokeAndFill = 2

	SamplingNearest = 0
	SamplingLinear  = 1

	TileClamp  = 0
	TileRepeat = 1
	TileMirror = 2
	TileDecal  = 3
)

// ---------------------------------------------------------------------------
// Paths

type Path struct{ h C.ImpellerPath }

func (p *Path) Release() { C.ImpellerPathRelease(p.h) }

// Bounds is ImpellerPathGetBounds.
func (p *Path) Bounds() Rect {
	var r C.ImpellerRect
	C.ImpellerPathGetBounds(p.h, &r)
	return Rect{float32(r.x), float32(r.y), float32(r.width), float32(r.height)}
}

type PathBuilder struct{ h C.ImpellerPathBuilder }

func NewPathBuilder() *PathBuilder { return &PathBuilder{C.ImpellerPathBuilderNew()} }
func (b *PathBuilder) Release()    { C.ImpellerPathBuilderRelease(b.h) }
func (b *PathBuilder) MoveTo(x, y float32) {
	p := Point{x, y}.c()
	C.ImpellerPathBuilderMoveTo(b.h, &p)
}
func (b *PathBuilder) LineTo(x, y float32) {
	p := Point{x, y}.c()
	C.ImpellerPathBuilderLineTo(b.h, &p)
}
func (b *PathBuilder) QuadTo(x1, y1, x2, y2 float32) {
	p1, p2 := Point{x1, y1}.c(), Point{x2, y2}.c()
	C.ImpellerPathBuilderQuadraticCurveTo(b.h, &p1, &p2)
}
func (b *PathBuilder) CubicTo(x1, y1, x2, y2, x3, y3 float32) {
	p1, p2, p3 := Point{x1, y1}.c(), Point{x2, y2}.c(), Point{x3, y3}.c()
	C.ImpellerPathBuilderCubicCurveTo(b.h, &p1, &p2, &p3)
}
func (b *PathBuilder) Close() { C.ImpellerPathBuilderClose(b.h) }
func (b *PathBuilder) AddRect(r Rect) {
	cr := r.c()
	C.ImpellerPathBuilderAddRect(b.h, &cr)
}
func (b *PathBuilder) AddOval(r Rect) {
	cr := r.c()
	C.ImpellerPathBuilderAddOval(b.h, &cr)
}
func (b *PathBuilder) AddRoundedRect(r Rect, radii Radii) {
	cr, rr := r.c(), radii.c()
	C.ImpellerPathBuilderAddRoundedRect(b.h, &cr, &rr)
}

// TakePath returns the path built so far and resets the builder.
func (b *PathBuilder) TakePath(fillType int) *Path {
	return &Path{C.ImpellerPathBuilderTakePathNew(b.h, C.ImpellerFillType(fillType))}
}

// ---------------------------------------------------------------------------
// Paint

func (c Color) cs() C.ImpellerColor {
	cc := c.c()
	cc.color_space = C.ImpellerColorSpace(c.Space)
	return cc
}

func (p *Paint) SetColorWithSpace(c Color) {
	cc := c.cs()
	C.ImpellerPaintSetColor(p.h, &cc)
}
func (p *Paint) SetBlendMode(m int) { C.ImpellerPaintSetBlendMode(p.h, C.ImpellerBlendMode(m)) }
func (p *Paint) SetDrawStyle(s int) { C.ImpellerPaintSetDrawStyle(p.h, C.ImpellerDrawStyle(s)) }
func (p *Paint) SetStrokeCap(c int) { C.ImpellerPaintSetStrokeCap(p.h, C.ImpellerStrokeCap(c)) }
func (p *Paint) SetStrokeJoin(j int) {
	C.ImpellerPaintSetStrokeJoin(p.h, C.ImpellerStrokeJoin(j))
}
func (p *Paint) SetStrokeWidth(w float32) { C.ImpellerPaintSetStrokeWidth(p.h, C.float(w)) }
func (p *Paint) SetStrokeMiter(m float32) { C.ImpellerPaintSetStrokeMiter(p.h, C.float(m)) }
func (p *Paint) SetColorFilter(f *ColorFilter) {
	C.ImpellerPaintSetColorFilter(p.h, f.h)
}
func (p *Paint) SetColorSource(s *ColorSource) {
	C.ImpellerPaintSetColorSource(p.h, s.h)
}
func (p *Paint) SetImageFilter(f *ImageFilter) {
	C.ImpellerPaintSetImageFilter(p.h, f.h)
}
func (p *Paint) SetMaskFilter(f *MaskFilter) { C.ImpellerPaintSetMaskFilter(p.h, f.h) }

// ---------------------------------------------------------------------------
// Textures

type Texture struct{ h C.ImpellerTexture }

// NewTextureRGBA8 uploads tightly packed, premultiplied RGBA8888 pixels
// (the bytes are copied). It returns nil when Impeller refuses.
func NewTextureRGBA8(ctx *Context, width, height int, pixels []byte) *Texture {
	if len(pixels) == 0 {
		return nil
	}
	h := C.fg_texture_new(ctx.h, C.int64_t(width), C.int64_t(height), unsafe.Pointer(&pixels[0]), C.size_t(len(pixels)))
	if h == nil {
		return nil
	}
	return &Texture{h}
}
func (t *Texture) Release() { C.ImpellerTextureRelease(t.h) }

// ---------------------------------------------------------------------------
// Color sources, color filters, mask filters, image filters

type ColorSource struct{ h C.ImpellerColorSource }

func (s *ColorSource) Release() { C.ImpellerColorSourceRelease(s.h) }

func colorSource(h C.ImpellerColorSource) *ColorSource {
	if h == nil {
		return nil
	}
	return &ColorSource{h}
}

func stopsC(colors []Color, stops []float32) (*C.ImpellerColor, *C.float, C.uint32_t) {
	cc := make([]C.ImpellerColor, len(colors))
	for i, c := range colors {
		cc[i] = c.cs()
	}
	cs := make([]C.float, len(stops))
	for i, s := range stops {
		cs[i] = C.float(s)
	}
	return &cc[0], &cs[0], C.uint32_t(len(colors))
}

func NewLinearGradient(start, end Point, colors []Color, stops []float32, tile int, m *Matrix) *ColorSource {
	s, e := start.c(), end.c()
	cc, cs, n := stopsC(colors, stops)
	return colorSource(C.ImpellerColorSourceCreateLinearGradientNew(&s, &e, n, cc, cs, C.ImpellerTileMode(tile), m.c()))
}

func NewRadialGradient(center Point, radius float32, colors []Color, stops []float32, tile int, m *Matrix) *ColorSource {
	c := center.c()
	cc, cs, n := stopsC(colors, stops)
	return colorSource(C.ImpellerColorSourceCreateRadialGradientNew(&c, C.float(radius), n, cc, cs, C.ImpellerTileMode(tile), m.c()))
}

func NewConicalGradient(startCenter Point, startRadius float32, endCenter Point, endRadius float32, colors []Color, stops []float32, tile int, m *Matrix) *ColorSource {
	s, e := startCenter.c(), endCenter.c()
	cc, cs, n := stopsC(colors, stops)
	return colorSource(C.ImpellerColorSourceCreateConicalGradientNew(&s, C.float(startRadius), &e, C.float(endRadius), n, cc, cs, C.ImpellerTileMode(tile), m.c()))
}

// NewSweepGradient takes its angles in degrees.
func NewSweepGradient(center Point, start, end float32, colors []Color, stops []float32, tile int, m *Matrix) *ColorSource {
	c := center.c()
	cc, cs, n := stopsC(colors, stops)
	return colorSource(C.ImpellerColorSourceCreateSweepGradientNew(&c, C.float(start), C.float(end), n, cc, cs, C.ImpellerTileMode(tile), m.c()))
}

func NewImageColorSource(t *Texture, tileX, tileY, sampling int, m *Matrix) *ColorSource {
	return colorSource(C.ImpellerColorSourceCreateImageNew(t.h, C.ImpellerTileMode(tileX), C.ImpellerTileMode(tileY), C.ImpellerTextureSampling(sampling), m.c()))
}

type ColorFilter struct{ h C.ImpellerColorFilter }

func (f *ColorFilter) Release() { C.ImpellerColorFilterRelease(f.h) }

func NewBlendColorFilter(c Color, mode int) *ColorFilter {
	cc := c.cs()
	h := C.ImpellerColorFilterCreateBlendNew(&cc, C.ImpellerBlendMode(mode))
	if h == nil {
		return nil
	}
	return &ColorFilter{h}
}

// NewMatrixColorFilter takes a row-major 4x5 matrix whose translation column
// is in 0..255 space (the C API's convention).
func NewMatrixColorFilter(m [20]float32) *ColorFilter {
	var cm C.ImpellerColorMatrix
	for i, v := range m {
		cm.m[i] = C.float(v)
	}
	h := C.ImpellerColorFilterCreateColorMatrixNew(&cm)
	if h == nil {
		return nil
	}
	return &ColorFilter{h}
}

type MaskFilter struct{ h C.ImpellerMaskFilter }

func (f *MaskFilter) Release() { C.ImpellerMaskFilterRelease(f.h) }

func NewBlurMaskFilter(style int, sigma float32) *MaskFilter {
	h := C.ImpellerMaskFilterCreateBlurNew(C.ImpellerBlurStyle(style), C.float(sigma))
	if h == nil {
		return nil
	}
	return &MaskFilter{h}
}

type ImageFilter struct{ h C.ImpellerImageFilter }

func (f *ImageFilter) Release() { C.ImpellerImageFilterRelease(f.h) }

func imageFilter(h C.ImpellerImageFilter) *ImageFilter {
	if h == nil {
		return nil
	}
	return &ImageFilter{h}
}

func NewBlurImageFilter(sx, sy float32, tile int) *ImageFilter {
	return imageFilter(C.ImpellerImageFilterCreateBlurNew(C.float(sx), C.float(sy), C.ImpellerTileMode(tile)))
}
func NewDilateImageFilter(rx, ry float32) *ImageFilter {
	return imageFilter(C.ImpellerImageFilterCreateDilateNew(C.float(rx), C.float(ry)))
}
func NewErodeImageFilter(rx, ry float32) *ImageFilter {
	return imageFilter(C.ImpellerImageFilterCreateErodeNew(C.float(rx), C.float(ry)))
}
func NewMatrixImageFilter(m Matrix, sampling int) *ImageFilter {
	return imageFilter(C.ImpellerImageFilterCreateMatrixNew(m.c(), C.ImpellerTextureSampling(sampling)))
}
func NewComposeImageFilter(outer, inner *ImageFilter) *ImageFilter {
	return imageFilter(C.ImpellerImageFilterCreateComposeNew(outer.h, inner.h))
}

// ---------------------------------------------------------------------------
// Display list builder

// NewDisplayListBuilderCull is NewDisplayListBuilder with a cull rect.
func NewDisplayListBuilderCull(cull Rect) *DisplayListBuilder {
	cr := cull.c()
	return &DisplayListBuilder{C.ImpellerDisplayListBuilderNew(&cr)}
}

func (b *DisplayListBuilder) SaveLayer(bounds *Rect, p *Paint, backdrop *ImageFilter) {
	var cr C.ImpellerRect
	if bounds != nil {
		cr = bounds.c()
	} else {
		// The C API needs bounds; a huge rect stands for "unbounded".
		cr = Rect{-1e9, -1e9, 2e9, 2e9}.c()
	}
	var ph C.ImpellerPaint
	if p != nil {
		ph = p.h
	}
	var bh C.ImpellerImageFilter
	if backdrop != nil {
		bh = backdrop.h
	}
	C.ImpellerDisplayListBuilderSaveLayer(b.h, &cr, ph, bh)
}
func (b *DisplayListBuilder) Rotate(degrees float32) {
	C.ImpellerDisplayListBuilderRotate(b.h, C.float(degrees))
}
func (b *DisplayListBuilder) Transform(m Matrix) { C.ImpellerDisplayListBuilderTransform(b.h, m.c()) }
func (b *DisplayListBuilder) SetTransform(m Matrix) {
	C.ImpellerDisplayListBuilderSetTransform(b.h, m.c())
}
func (b *DisplayListBuilder) GetTransform() Matrix {
	var cm C.ImpellerMatrix
	C.ImpellerDisplayListBuilderGetTransform(b.h, &cm)
	var m Matrix
	for i := range m {
		m[i] = float32(cm.m[i])
	}
	return m
}
func (b *DisplayListBuilder) ResetTransform() { C.ImpellerDisplayListBuilderResetTransform(b.h) }
func (b *DisplayListBuilder) SaveCount() int {
	return int(C.ImpellerDisplayListBuilderGetSaveCount(b.h))
}
func (b *DisplayListBuilder) RestoreToCount(n int) {
	C.ImpellerDisplayListBuilderRestoreToCount(b.h, C.uint32_t(n))
}
func (b *DisplayListBuilder) ClipRectOp(r Rect, op int) {
	cr := r.c()
	C.ImpellerDisplayListBuilderClipRect(b.h, &cr, C.ImpellerClipOperation(op))
}
func (b *DisplayListBuilder) ClipOval(r Rect, op int) {
	cr := r.c()
	C.ImpellerDisplayListBuilderClipOval(b.h, &cr, C.ImpellerClipOperation(op))
}
func (b *DisplayListBuilder) ClipRoundedRect(r Rect, radii Radii, op int) {
	cr, rr := r.c(), radii.c()
	C.ImpellerDisplayListBuilderClipRoundedRect(b.h, &cr, &rr, C.ImpellerClipOperation(op))
}
func (b *DisplayListBuilder) ClipPath(p *Path, op int) {
	C.ImpellerDisplayListBuilderClipPath(b.h, p.h, C.ImpellerClipOperation(op))
}
func (b *DisplayListBuilder) DrawLine(from, to Point, p *Paint) {
	f, t := from.c(), to.c()
	C.ImpellerDisplayListBuilderDrawLine(b.h, &f, &t, p.h)
}
func (b *DisplayListBuilder) DrawOval(r Rect, p *Paint) {
	cr := r.c()
	C.ImpellerDisplayListBuilderDrawOval(b.h, &cr, p.h)
}
func (b *DisplayListBuilder) DrawRoundedRectRadii(r Rect, radii Radii, p *Paint) {
	cr, rr := r.c(), radii.c()
	C.ImpellerDisplayListBuilderDrawRoundedRect(b.h, &cr, &rr, p.h)
}
func (b *DisplayListBuilder) DrawRoundedRectDifference(outer Rect, outerRadii Radii, inner Rect, innerRadii Radii, p *Paint) {
	o, or, i, ir := outer.c(), outerRadii.c(), inner.c(), innerRadii.c()
	C.ImpellerDisplayListBuilderDrawRoundedRectDifference(b.h, &o, &or, &i, &ir, p.h)
}
func (b *DisplayListBuilder) DrawPath(path *Path, p *Paint) {
	C.ImpellerDisplayListBuilderDrawPath(b.h, path.h, p.h)
}
func (b *DisplayListBuilder) DrawShadow(path *Path, c Color, elevation float32, transparentOccluder bool, dpr float32) {
	cc := c.cs()
	C.ImpellerDisplayListBuilderDrawShadow(b.h, path.h, &cc, C.float(elevation), C.bool(transparentOccluder), C.float(dpr))
}
func (b *DisplayListBuilder) DrawTexture(t *Texture, at Point, sampling int, p *Paint) {
	pt := at.c()
	var ph C.ImpellerPaint
	if p != nil {
		ph = p.h
	}
	C.ImpellerDisplayListBuilderDrawTexture(b.h, t.h, &pt, C.ImpellerTextureSampling(sampling), ph)
}
func (b *DisplayListBuilder) DrawTextureRect(t *Texture, src, dst Rect, sampling int, p *Paint) {
	s, d := src.c(), dst.c()
	var ph C.ImpellerPaint
	if p != nil {
		ph = p.h
	}
	C.ImpellerDisplayListBuilderDrawTextureRect(b.h, t.h, &s, &d, C.ImpellerTextureSampling(sampling), ph)
}
