package impeller

// Go side of typography_bridge.cc: the engine's text stack (txt over
// SkParagraph) as the Impeller SDK compiles it in. See txt_mirror.h for how
// the SDK's unexported engine objects are reached.

/*
#cgo CXXFLAGS: -std=c++20 -I${SRCDIR}/../third_party/impeller_sdk/include
#include <stdlib.h>
#include "typography_bridge.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

// TxtFontCollection is the engine's font collection (lib/ui/text's
// FontCollection over txt::FontCollection).
type TxtFontCollection struct{ c *C.FgoFontCollection }

// TestFont is a font flutter_tester registers with --use-test-fonts.
type TestFont struct {
	Family string
	Data   []byte
}

// NewTxtFontCollection creates a font collection. With test fonts, every
// family resolves to one of them (the first unless named) and font fallback
// is off, as in flutter_tester; without, the platform's fonts are used.
func NewTxtFontCollection(testFonts []TestFont) *TxtFontCollection {
	n := len(testFonts)
	if n == 0 {
		return &TxtFontCollection{C.fgo_font_collection_new(nil, nil, nil, 0)}
	}
	ptrs := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(uintptr(0))))
	lens := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(C.size_t(0))))
	names := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(uintptr(0))))
	defer C.free(ptrs)
	defer C.free(lens)
	defer C.free(names)
	pa := unsafe.Slice((**C.uint8_t)(ptrs), n)
	la := unsafe.Slice((*C.size_t)(lens), n)
	na := unsafe.Slice((**C.char)(names), n)
	for i, f := range testFonts {
		pa[i] = (*C.uint8_t)(C.CBytes(f.Data))
		la[i] = C.size_t(len(f.Data))
		na[i] = C.CString(f.Family)
	}
	c := C.fgo_font_collection_new((**C.uint8_t)(ptrs), (*C.size_t)(lens), (**C.char)(names), C.size_t(n))
	for i := range testFonts {
		C.free(unsafe.Pointer(pa[i]))
		C.free(unsafe.Pointer(na[i]))
	}
	return &TxtFontCollection{c}
}

// LoadFont registers a font (FontCollection::LoadFontFromList); an empty
// family uses the font's own family name.
func (f *TxtFontCollection) LoadFont(data []byte, family string) bool {
	if len(data) == 0 {
		return false
	}
	p := C.CBytes(data)
	defer C.free(p)
	var cf *C.char
	if family != "" {
		cf = C.CString(family)
		defer C.free(unsafe.Pointer(cf))
	}
	return bool(C.fgo_font_collection_load_font(f.c, (*C.uint8_t)(p), C.size_t(len(data)), cf))
}

// cStrings allocates a C array of C strings; free with freeCStrings.
func cStrings(s []string) (**C.char, func()) {
	if len(s) == 0 {
		return nil, func() {}
	}
	arr := C.malloc(C.size_t(len(s)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	a := unsafe.Slice((**C.char)(arr), len(s))
	for i, x := range s {
		a[i] = C.CString(x)
	}
	return (**C.char)(arr), func() {
		for i := range s {
			C.free(unsafe.Pointer(a[i]))
		}
		C.free(arr)
	}
}

func cBytes(b []byte) (*C.uint8_t, func()) {
	if b == nil {
		return nil, func() {}
	}
	p := C.malloc(C.size_t(len(b)) + 1)
	copy(unsafe.Slice((*byte)(p), len(b)), b)
	return (*C.uint8_t)(p), func() { C.free(p) }
}

// TxtParagraphBuilderArgs are ParagraphBuilder::Create's arguments (the
// encoded paragraph style, as text.dart passes them). StrutData nil is
// Dart's null.
type TxtParagraphBuilderArgs struct {
	Encoded         []int32
	StrutData       []byte
	FontFamily      string
	StrutFamilies   []string
	FontSize        float64
	Height          float64
	Ellipsis        []uint16
	Locale          string
	ImpellerEnabled bool
}

// TxtParagraphBuilder is the engine's ParagraphBuilder.
type TxtParagraphBuilder struct{ b *C.FgoParagraphBuilder }

// NewTxtParagraphBuilder is ParagraphBuilder::Create.
func NewTxtParagraphBuilder(fonts *TxtFontCollection, a TxtParagraphBuilderArgs) *TxtParagraphBuilder {
	enc := C.malloc(C.size_t(len(a.Encoded)*4 + 4))
	defer C.free(enc)
	copy(unsafe.Slice((*int32)(enc), len(a.Encoded)), a.Encoded)
	strut, freeStrut := cBytes(a.StrutData)
	defer freeStrut()
	family := C.CString(a.FontFamily)
	defer C.free(unsafe.Pointer(family))
	families, freeFamilies := cStrings(a.StrutFamilies)
	defer freeFamilies()
	ell := C.malloc(C.size_t(len(a.Ellipsis)*2 + 2))
	defer C.free(ell)
	copy(unsafe.Slice((*uint16)(ell), len(a.Ellipsis)), a.Ellipsis)
	locale := C.CString(a.Locale)
	defer C.free(unsafe.Pointer(locale))
	b := C.fgo_paragraph_builder_new(fonts.c, (*C.int32_t)(enc), C.size_t(len(a.Encoded)),
		strut, C.size_t(len(a.StrutData)), family, families, C.size_t(len(a.StrutFamilies)),
		C.double(a.FontSize), C.double(a.Height), (*C.uint16_t)(ell), C.size_t(len(a.Ellipsis)),
		locale, C.bool(a.ImpellerEnabled))
	pb := &TxtParagraphBuilder{b}
	runtime.SetFinalizer(pb, (*TxtParagraphBuilder).Release)
	return pb
}

// TxtTextStyleArgs are ParagraphBuilder::pushStyle's arguments. Background
// and Foreground are nil for Dart's null (the paint is copied);
// FontFeatures/FontVariations nil are null; Shadows is never null in Dart
// (an empty ByteData).
type TxtTextStyleArgs struct {
	Encoded             []int32
	FontFamilies        []string
	FontSize            float64
	LetterSpacing       float64
	WordSpacing         float64
	Height              float64
	DecorationThickness float64
	Locale              string
	Background          *Paint
	Foreground          *Paint
	Shadows             []byte
	FontFeatures        []byte
	FontVariations      []byte
}

// PushStyle is ParagraphBuilder::pushStyle.
func (b *TxtParagraphBuilder) PushStyle(a TxtTextStyleArgs) {
	enc := C.malloc(C.size_t(len(a.Encoded)*4 + 4))
	defer C.free(enc)
	copy(unsafe.Slice((*int32)(enc), len(a.Encoded)), a.Encoded)
	families, freeFamilies := cStrings(a.FontFamilies)
	defer freeFamilies()
	locale := C.CString(a.Locale)
	defer C.free(unsafe.Pointer(locale))
	var bg, fg C.ImpellerPaint
	if a.Background != nil {
		bg = a.Background.h
	}
	if a.Foreground != nil {
		fg = a.Foreground.h
	}
	shadows, freeShadows := cBytes(append([]byte{}, a.Shadows...))
	defer freeShadows()
	features, freeFeatures := cBytes(a.FontFeatures)
	defer freeFeatures()
	variations, freeVariations := cBytes(a.FontVariations)
	defer freeVariations()
	C.fgo_paragraph_builder_push_style(b.b, (*C.int32_t)(enc), families, C.size_t(len(a.FontFamilies)),
		C.double(a.FontSize), C.double(a.LetterSpacing), C.double(a.WordSpacing), C.double(a.Height),
		C.double(a.DecorationThickness), locale, bg, fg,
		shadows, C.size_t(len(a.Shadows)), features, C.size_t(len(a.FontFeatures)),
		variations, C.size_t(len(a.FontVariations)))
}

// Pop is ParagraphBuilder::pop.
func (b *TxtParagraphBuilder) Pop() { C.fgo_paragraph_builder_pop(b.b) }

// AddText is ParagraphBuilder::addText on well-formed UTF-16.
func (b *TxtParagraphBuilder) AddText(text []uint16) {
	if len(text) == 0 {
		return
	}
	p := C.malloc(C.size_t(len(text) * 2))
	defer C.free(p)
	copy(unsafe.Slice((*uint16)(p), len(text)), text)
	C.fgo_paragraph_builder_add_text(b.b, (*C.uint16_t)(p), C.size_t(len(text)))
}

// AddPlaceholder is ParagraphBuilder::addPlaceholder.
func (b *TxtParagraphBuilder) AddPlaceholder(width, height float64, alignment uint32, baselineOffset float64, baseline uint32) {
	C.fgo_paragraph_builder_add_placeholder(b.b, C.double(width), C.double(height), C.uint32_t(alignment),
		C.double(baselineOffset), C.uint32_t(baseline))
}

// Build is ParagraphBuilder::build; the builder can't be used afterwards.
func (b *TxtParagraphBuilder) Build() *TxtParagraph {
	p := &TxtParagraph{p: C.fgo_paragraph_builder_build(b.b)}
	b.b = nil
	runtime.SetFinalizer(p, (*TxtParagraph).free)
	return p
}

// Release frees a builder that was not built.
func (b *TxtParagraphBuilder) Release() {
	if b.b != nil {
		C.fgo_paragraph_builder_release(b.b)
		b.b = nil
	}
}

// TxtParagraph is the engine's Paragraph (a txt::Paragraph).
type TxtParagraph struct{ p *C.FgoParagraph }

var errDisposed = errors.New("impeller: paragraph used after dispose")

func (p *TxtParagraph) free() { p.Dispose() }

func (p *TxtParagraph) live() *C.FgoParagraph {
	if p.p == nil {
		panic(errDisposed)
	}
	return p.p
}

// TxtParagraphMetrics are the paragraph's metric getters.
type TxtParagraphMetrics struct {
	Width, Height, LongestLine              float64
	MinIntrinsicWidth, MaxIntrinsicWidth    float64
	AlphabeticBaseline, IdeographicBaseline float64
	DidExceedMaxLines                       bool
}

// Metrics reads every metric getter.
func (p *TxtParagraph) Metrics() TxtParagraphMetrics {
	var m C.FgoParagraphMetrics
	C.fgo_paragraph_metrics(p.live(), &m)
	return TxtParagraphMetrics{float64(m.width), float64(m.height), float64(m.longest_line),
		float64(m.min_intrinsic_width), float64(m.max_intrinsic_width),
		float64(m.alphabetic_baseline), float64(m.ideographic_baseline), bool(m.did_exceed_max_lines)}
}

// Layout is Paragraph::layout.
func (p *TxtParagraph) Layout(width float64) { C.fgo_paragraph_layout(p.live(), C.double(width)) }

// Paint is Paragraph::paint into a display list builder.
func (p *TxtParagraph) Paint(b *DisplayListBuilder, x, y float64) {
	if p.p == nil || b == nil {
		return
	}
	C.fgo_paragraph_paint(p.p, b.h, C.double(x), C.double(y))
}

func floats(n C.size_t, out *C.float, per int) []float32 {
	defer C.fgo_free(unsafe.Pointer(out))
	return append([]float32{}, unsafe.Slice((*float32)(unsafe.Pointer(out)), int(n)*per)...)
}

// RectsForRange is Paragraph::getRectsForRange (5 floats per box).
func (p *TxtParagraph) RectsForRange(start, end, heightStyle, widthStyle uint32) []float32 {
	var out *C.float
	n := C.fgo_paragraph_get_rects_for_range(p.live(), C.uint32_t(start), C.uint32_t(end),
		C.uint32_t(heightStyle), C.uint32_t(widthStyle), &out)
	return floats(n, out, 5)
}

// RectsForPlaceholders is Paragraph::getRectsForPlaceholders.
func (p *TxtParagraph) RectsForPlaceholders() []float32 {
	var out *C.float
	n := C.fgo_paragraph_get_rects_for_placeholders(p.live(), &out)
	return floats(n, out, 5)
}

// PositionForOffset is Paragraph::getPositionForOffset (position, affinity).
func (p *TxtParagraph) PositionForOffset(dx, dy float64) (int, int) {
	var out [2]C.size_t
	C.fgo_paragraph_get_position_for_offset(p.live(), C.double(dx), C.double(dy), &out[0])
	return int(out[0]), int(out[1])
}

// TxtGlyphInfo is a found glyph's information (glyphInfoFrom's arguments).
type TxtGlyphInfo struct {
	Left, Top, Right, Bottom float64
	Start, End               int
	IsLTR                    bool
}

func glyphInfo(g C.FgoGlyphInfo) TxtGlyphInfo {
	return TxtGlyphInfo{float64(g.left), float64(g.top), float64(g.right), float64(g.bottom),
		int(g.start), int(g.end), bool(g.is_ltr)}
}

// GlyphInfoAt is Paragraph::getGlyphInfoAt (false: null).
func (p *TxtParagraph) GlyphInfoAt(offset uint32) (TxtGlyphInfo, bool) {
	var g C.FgoGlyphInfo
	if !C.fgo_paragraph_get_glyph_info_at(p.live(), C.uint32_t(offset), &g) {
		return TxtGlyphInfo{}, false
	}
	return glyphInfo(g), true
}

// ClosestGlyphInfo is Paragraph::getClosestGlyphInfo (false: null).
func (p *TxtParagraph) ClosestGlyphInfo(dx, dy float64) (TxtGlyphInfo, bool) {
	var g C.FgoGlyphInfo
	if !C.fgo_paragraph_get_closest_glyph_info(p.live(), C.double(dx), C.double(dy), &g) {
		return TxtGlyphInfo{}, false
	}
	return glyphInfo(g), true
}

// WordBoundary is Paragraph::getWordBoundary.
func (p *TxtParagraph) WordBoundary(offset uint32) (int, int) {
	var out [2]C.size_t
	C.fgo_paragraph_get_word_boundary(p.live(), C.uint32_t(offset), &out[0])
	return int(out[0]), int(out[1])
}

// LineBoundary is Paragraph::getLineBoundary.
func (p *TxtParagraph) LineBoundary(offset uint32) (int, int) {
	var out [2]C.int32_t
	C.fgo_paragraph_get_line_boundary(p.live(), C.uint32_t(offset), &out[0])
	return int(out[0]), int(out[1])
}

// ComputeLineMetrics is Paragraph::computeLineMetrics (9 values per line).
func (p *TxtParagraph) ComputeLineMetrics() []float64 {
	var out *C.double
	n := C.fgo_paragraph_compute_line_metrics(p.live(), &out)
	defer C.fgo_free(unsafe.Pointer(out))
	return append([]float64{}, unsafe.Slice((*float64)(unsafe.Pointer(out)), int(n)*9)...)
}

// LineMetricsAt is Paragraph::getLineMetricsAt (false: null).
func (p *TxtParagraph) LineMetricsAt(lineNumber int32) ([9]float64, bool) {
	var out [9]C.double
	var r [9]float64
	if !C.fgo_paragraph_get_line_metrics_at(p.live(), C.int32_t(lineNumber), &out[0]) {
		return r, false
	}
	for i := range out {
		r[i] = float64(out[i])
	}
	return r, true
}

// NumberOfLines is Paragraph::getNumberOfLines.
func (p *TxtParagraph) NumberOfLines() int {
	return int(C.fgo_paragraph_get_number_of_lines(p.live()))
}

// LineNumberAt is Paragraph::getLineNumberAt.
func (p *TxtParagraph) LineNumberAt(offset uint64) int32 {
	return int32(C.fgo_paragraph_get_line_number_at(p.live(), C.size_t(offset)))
}

// Dispose is Paragraph::dispose.
func (p *TxtParagraph) Dispose() {
	if p.p != nil {
		C.fgo_paragraph_dispose(p.p)
		p.p = nil
	}
}

// TypographySelfCheck verifies the mirrored engine layouts against the SDK.
func TypographySelfCheck() error {
	buf := (*C.char)(C.malloc(1024))
	defer C.free(unsafe.Pointer(buf))
	if !C.fgo_typography_self_check(buf, 1024) {
		return errors.New("impeller: txt layout mismatch: " + C.GoString(buf))
	}
	return nil
}
