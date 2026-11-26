// C interface to the engine's text stack inside the Impeller SDK (see
// txt_mirror.h). It is the engine side of dart:ui's text.dart: the bridge
// functions port lib/ui/text/paragraph_builder.cc, paragraph.cc and
// font_collection.cc (Flutter 3.50.1), calling txt::ParagraphBuilder and
// txt::Paragraph through their vtables.

#ifndef FLUTTER_GO_TYPOGRAPHY_BRIDGE_H_
#define FLUTTER_GO_TYPOGRAPHY_BRIDGE_H_

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#include "impeller.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct FgoFontCollection FgoFontCollection;
typedef struct FgoParagraphBuilder FgoParagraphBuilder;
typedef struct FgoParagraph FgoParagraph;

// fgo_font_collection_new creates the engine's FontCollection. With test
// fonts (flutter_tester's --use-test-fonts), they are registered under their
// names and every other family resolves to the first one
// (txt::TestFontManager), with font fallback disabled; without, the
// platform's fonts are the fallback.
FgoFontCollection* fgo_font_collection_new(const uint8_t* const* test_fonts,
                                           const size_t* test_font_lengths,
                                           const char* const* test_font_names,
                                           size_t test_font_count);

// FontCollection::LoadFontFromList: family may be NULL (the font's own
// family name). Returns false if the data is not a font.
bool fgo_font_collection_load_font(FgoFontCollection* collection,
                                   const uint8_t* data,
                                   size_t length,
                                   const char* family);

// ParagraphBuilder::Create. strut_data is NULL for Dart's null;
// strut_families is used when the strut's font family mask bit is set.
FgoParagraphBuilder* fgo_paragraph_builder_new(
    FgoFontCollection* collection,
    const int32_t* encoded,
    size_t encoded_length,
    const uint8_t* strut_data,
    size_t strut_length,
    const char* font_family,
    const char* const* strut_families,
    size_t strut_family_count,
    double font_size,
    double height,
    const uint16_t* ellipsis,
    size_t ellipsis_length,
    const char* locale,
    bool impeller_enabled);

// ParagraphBuilder::pushStyle. background/foreground are NULL for Dart's
// null; their DlPaints are copied. shadows is never NULL (an empty
// ByteData); font_features/font_variations are NULL for null.
void fgo_paragraph_builder_push_style(FgoParagraphBuilder* builder,
                                      const int32_t* encoded,
                                      const char* const* font_families,
                                      size_t font_family_count,
                                      double font_size,
                                      double letter_spacing,
                                      double word_spacing,
                                      double height,
                                      double decoration_thickness,
                                      const char* locale,
                                      ImpellerPaint background,
                                      ImpellerPaint foreground,
                                      const uint8_t* shadows,
                                      size_t shadows_length,
                                      const uint8_t* font_features,
                                      size_t font_features_length,
                                      const uint8_t* font_variations,
                                      size_t font_variations_length);

void fgo_paragraph_builder_pop(FgoParagraphBuilder* builder);

// ParagraphBuilder::addText (the text is validated by the caller).
void fgo_paragraph_builder_add_text(FgoParagraphBuilder* builder,
                                    const uint16_t* text,
                                    size_t length);

void fgo_paragraph_builder_add_placeholder(FgoParagraphBuilder* builder,
                                           double width,
                                           double height,
                                           uint32_t alignment,
                                           double baseline_offset,
                                           uint32_t baseline);

// ParagraphBuilder::build: the builder is consumed (freed).
FgoParagraph* fgo_paragraph_builder_build(FgoParagraphBuilder* builder);

// Frees a builder that was never built.
void fgo_paragraph_builder_release(FgoParagraphBuilder* builder);

typedef struct FgoParagraphMetrics {
  double width;
  double height;
  double longest_line;
  double min_intrinsic_width;
  double max_intrinsic_width;
  double alphabetic_baseline;
  double ideographic_baseline;
  bool did_exceed_max_lines;
} FgoParagraphMetrics;

void fgo_paragraph_metrics(FgoParagraph* paragraph, FgoParagraphMetrics* out);
void fgo_paragraph_layout(FgoParagraph* paragraph, double width);
void fgo_paragraph_paint(FgoParagraph* paragraph,
                         ImpellerDisplayListBuilder builder,
                         double x,
                         double y);

// getRectsForRange / getRectsForPlaceholders: *out is a malloc'ed array of
// 5 floats per box (left, top, right, bottom, direction); returns the box
// count. Free with fgo_free.
size_t fgo_paragraph_get_rects_for_range(FgoParagraph* paragraph,
                                         uint32_t start,
                                         uint32_t end,
                                         uint32_t height_style,
                                         uint32_t width_style,
                                         float** out);
size_t fgo_paragraph_get_rects_for_placeholders(FgoParagraph* paragraph,
                                                float** out);

void fgo_paragraph_get_position_for_offset(FgoParagraph* paragraph,
                                           double dx,
                                           double dy,
                                           size_t out[2]);

typedef struct FgoGlyphInfo {
  float left, top, right, bottom;
  size_t start, end;
  bool is_ltr;
} FgoGlyphInfo;

bool fgo_paragraph_get_glyph_info_at(FgoParagraph* paragraph,
                                     uint32_t offset,
                                     FgoGlyphInfo* out);
bool fgo_paragraph_get_closest_glyph_info(FgoParagraph* paragraph,
                                          double dx,
                                          double dy,
                                          FgoGlyphInfo* out);

void fgo_paragraph_get_word_boundary(FgoParagraph* paragraph,
                                     uint32_t offset,
                                     size_t out[2]);
void fgo_paragraph_get_line_boundary(FgoParagraph* paragraph,
                                     uint32_t offset,
                                     int32_t out[2]);

// computeLineMetrics: *out is a malloc'ed array of 9 doubles per line.
size_t fgo_paragraph_compute_line_metrics(FgoParagraph* paragraph,
                                          double** out);

// getLineMetricsAt: the 9 values (hardBreak as 0/1, lineNumber as double).
bool fgo_paragraph_get_line_metrics_at(FgoParagraph* paragraph,
                                       int32_t line_number,
                                       double out[9]);

size_t fgo_paragraph_get_number_of_lines(FgoParagraph* paragraph);
int32_t fgo_paragraph_get_line_number_at(FgoParagraph* paragraph,
                                         size_t utf16_offset);

// Paragraph::dispose (frees the paragraph).
void fgo_paragraph_dispose(FgoParagraph* paragraph);

void fgo_free(void* p);

// Layout checks for typography_test.go: fills sizes/offsets of the mirrored
// engine objects as seen through the SDK's public setters.
bool fgo_typography_self_check(char* message, size_t message_size);

#ifdef __cplusplus
}
#endif

#endif  // FLUTTER_GO_TYPOGRAPHY_BRIDGE_H_
