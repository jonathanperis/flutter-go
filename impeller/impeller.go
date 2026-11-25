// Package impeller is a minimal Go binding to the Impeller Standalone SDK C API
// (third_party/impeller_sdk/include/impeller.h).
//
// Every type wraps an Impeller handle and must be released with Release.
package impeller

/*
#cgo CFLAGS: -I${SRCDIR}/../third_party/impeller_sdk/include
#cgo LDFLAGS: -L${SRCDIR}/../third_party/impeller_sdk/lib -limpeller -Wl,-rpath,${SRCDIR}/../third_party/impeller_sdk/lib
#include <stdlib.h>
#include "impeller.h"

static uint32_t fg_impeller_version() { return IMPELLER_VERSION; }
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Version is the C API version this binding was compiled against.
func Version() uint32 { return uint32(C.fg_impeller_version()) }

type Color struct{ R, G, B, A float32 }

func (c Color) c() C.ImpellerColor {
	return C.ImpellerColor{red: C.float(c.R), green: C.float(c.G), blue: C.float(c.B), alpha: C.float(c.A)}
}

type Rect struct{ X, Y, W, H float32 }

func (r Rect) c() C.ImpellerRect {
	return C.ImpellerRect{x: C.float(r.X), y: C.float(r.Y), width: C.float(r.W), height: C.float(r.H)}
}

// ---------------------------------------------------------------------------
// Context

type Context struct{ h C.ImpellerContext }

// NewMetalContext creates a Metal-backed rendering context.
func NewMetalContext() (*Context, error) {
	h := C.ImpellerContextCreateMetalNew(C.fg_impeller_version())
	if h == nil {
		return nil, errors.New("impeller: could not create Metal context")
	}
	return &Context{h}, nil
}

// Handle returns the raw ImpellerContext for platform shims.
func (c *Context) Handle() unsafe.Pointer { return unsafe.Pointer(c.h) }
func (c *Context) Release()               { C.ImpellerContextRelease(c.h) }

// ---------------------------------------------------------------------------
// Paint

type Paint struct{ h C.ImpellerPaint }

func NewPaint() *Paint { return &Paint{C.ImpellerPaintNew()} }

func (p *Paint) SetColor(c Color) {
	cc := c.c()
	C.ImpellerPaintSetColor(p.h, &cc)
}
func (p *Paint) Release() { C.ImpellerPaintRelease(p.h) }

// ---------------------------------------------------------------------------
// Typography

type TypographyContext struct{ h C.ImpellerTypographyContext }

func NewTypographyContext() *TypographyContext {
	return &TypographyContext{C.ImpellerTypographyContextNew()}
}
func (t *TypographyContext) Release() { C.ImpellerTypographyContextRelease(t.h) }

type Paragraph struct{ h C.ImpellerParagraph }

func (p *Paragraph) Release() { C.ImpellerParagraphRelease(p.h) }

// Metrics of a laid-out paragraph.
func (p *Paragraph) Height() float32 { return float32(C.ImpellerParagraphGetHeight(p.h)) }
func (p *Paragraph) MaxIntrinsicWidth() float32 {
	return float32(C.ImpellerParagraphGetMaxIntrinsicWidth(p.h))
}

// NewParagraph lays out a single-style paragraph of text within width.
func NewParagraph(tc *TypographyContext, text string, size float32, color Color, width float32) (*Paragraph, error) {
	return NewStyledParagraph(tc, text, size, color, false, width)
}

// NewStyledParagraph is NewParagraph with an optional bold weight.
func NewStyledParagraph(tc *TypographyContext, text string, size float32, color Color, bold bool, width float32) (*Paragraph, error) {
	style := C.ImpellerParagraphStyleNew()
	defer C.ImpellerParagraphStyleRelease(style)
	fg := NewPaint()
	defer fg.Release()
	fg.SetColor(color)
	C.ImpellerParagraphStyleSetForeground(style, fg.h)
	C.ImpellerParagraphStyleSetFontSize(style, C.float(size))
	if bold {
		C.ImpellerParagraphStyleSetFontWeight(style, C.kImpellerFontWeight700)
	}

	b := C.ImpellerParagraphBuilderNew(tc.h)
	if b == nil {
		return nil, errors.New("impeller: could not create paragraph builder")
	}
	defer C.ImpellerParagraphBuilderRelease(b)
	C.ImpellerParagraphBuilderPushStyle(b, style)
	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))
	C.ImpellerParagraphBuilderAddText(b, (*C.uint8_t)(unsafe.Pointer(ctext)), C.uint32_t(len(text)))
	h := C.ImpellerParagraphBuilderBuildParagraphNew(b, C.float(width))
	if h == nil {
		return nil, errors.New("impeller: could not build paragraph")
	}
	return &Paragraph{h}, nil
}

// ---------------------------------------------------------------------------
// Display lists

type DisplayList struct{ h C.ImpellerDisplayList }

func (d *DisplayList) Handle() unsafe.Pointer { return unsafe.Pointer(d.h) }
func (d *DisplayList) Release()               { C.ImpellerDisplayListRelease(d.h) }

// DisplayListBuilder records drawing commands entirely on the C side, so a
// frame costs one cgo call per command and none at present time.
type DisplayListBuilder struct{ h C.ImpellerDisplayListBuilder }

func NewDisplayListBuilder() *DisplayListBuilder {
	return &DisplayListBuilder{C.ImpellerDisplayListBuilderNew(nil)}
}
func (b *DisplayListBuilder) Release() { C.ImpellerDisplayListBuilderRelease(b.h) }

func (b *DisplayListBuilder) Save()    { C.ImpellerDisplayListBuilderSave(b.h) }
func (b *DisplayListBuilder) Restore() { C.ImpellerDisplayListBuilderRestore(b.h) }
func (b *DisplayListBuilder) Translate(x, y float32) {
	C.ImpellerDisplayListBuilderTranslate(b.h, C.float(x), C.float(y))
}
func (b *DisplayListBuilder) Scale(x, y float32) {
	C.ImpellerDisplayListBuilderScale(b.h, C.float(x), C.float(y))
}

func (b *DisplayListBuilder) ClipRect(r Rect) {
	cr := r.c()
	C.ImpellerDisplayListBuilderClipRect(b.h, &cr, C.kImpellerClipOperationIntersect)
}

func (b *DisplayListBuilder) DrawPaint(p *Paint) { C.ImpellerDisplayListBuilderDrawPaint(b.h, p.h) }

func (b *DisplayListBuilder) DrawRect(r Rect, p *Paint) {
	cr := r.c()
	C.ImpellerDisplayListBuilderDrawRect(b.h, &cr, p.h)
}

func (b *DisplayListBuilder) DrawRoundedRect(r Rect, radius float32, p *Paint) {
	cr := r.c()
	pt := C.ImpellerPoint{x: C.float(radius), y: C.float(radius)}
	radii := C.ImpellerRoundingRadii{top_left: pt, bottom_left: pt, top_right: pt, bottom_right: pt}
	C.ImpellerDisplayListBuilderDrawRoundedRect(b.h, &cr, &radii, p.h)
}

func (b *DisplayListBuilder) DrawParagraph(p *Paragraph, x, y float32) {
	pt := C.ImpellerPoint{x: C.float(x), y: C.float(y)}
	C.ImpellerDisplayListBuilderDrawParagraph(b.h, p.h, &pt)
}

func (b *DisplayListBuilder) Build() *DisplayList {
	return &DisplayList{C.ImpellerDisplayListBuilderCreateDisplayListNew(b.h)}
}
