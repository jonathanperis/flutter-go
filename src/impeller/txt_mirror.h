// Layout mirrors of the engine's txt and interop classes that the prebuilt
// Impeller SDK (libimpeller.dylib) compiles in but does not export.
//
// The SDK's typography C API is a thin wrapper over the engine's own text
// stack (flutter/txt over SkParagraph, HarfBuzz and ICU): an ImpellerParagraph
// holds a txt::Paragraph, an ImpellerParagraphBuilder a txt::ParagraphBuilder.
// Both are abstract classes whose methods are virtual, so the engine code
// behind them can be reached from outside the library by calling through
// their vtables, provided the declarations below match the engine headers at
// the SDK's revision (IMPELLER_SHA) member for member and virtual for virtual.
// Every declaration names the engine header it mirrors; keep them in sync
// when the pin moves (typography_test.go checks the layouts at run time).
//
// Standard library types: the SDK's libc++ lives in namespace std::__fl but
// has the same object layout as the system libc++ (checked by the tests), and
// both allocate with malloc/free, so a std::string or std::vector built here
// can be read, copied and destroyed there and vice versa.

#ifndef FLUTTER_GO_TXT_MIRROR_H_
#define FLUTTER_GO_TXT_MIRROR_H_

#include <atomic>
#include <cstddef>
#include <cstdint>
#include <map>
#include <memory>
#include <optional>
#include <string>
#include <vector>

namespace fgo {

// --- Skia value types (include/core) -------------------------------------

using SkColor = uint32_t;
struct SkPoint {
  float fX, fY;
};
struct SkRect {
  float fLeft, fTop, fRight, fBottom;
};

// SkRefCntBase (include/core/SkRefCnt.h): a virtual destructor, the private
// virtual internal_dispose, and an atomic int32 count.
class SkRefCntMirror {
 public:
  SkRefCntMirror() : fRefCnt(1) {}
  virtual ~SkRefCntMirror() = default;
  virtual void internal_dispose() const { delete this; }
  void ref() const { fRefCnt.fetch_add(+1, std::memory_order_relaxed); }
  void unref() const {
    if (1 == fRefCnt.fetch_add(-1, std::memory_order_acq_rel)) {
      this->internal_dispose();
    }
  }
  mutable std::atomic<int32_t> fRefCnt;
};

// sk_sp<T> (include/core/SkRefCnt.h): one pointer, non-trivial (so passed
// and returned indirectly, as Skia's is).
template <typename T>
class SkSp {
 public:
  SkSp() = default;
  explicit SkSp(T* p) : fPtr(p) {}
  SkSp(const SkSp& o) : fPtr(o.fPtr) {
    if (fPtr) {
      fPtr->ref();
    }
  }
  SkSp(SkSp&& o) : fPtr(o.fPtr) { o.fPtr = nullptr; }
  ~SkSp() {
    if (fPtr) {
      fPtr->unref();
    }
  }
  SkSp& operator=(const SkSp&) = delete;
  T* get() const { return fPtr; }
  T* release() {
    T* p = fPtr;
    fPtr = nullptr;
    return p;
  }

 private:
  T* fPtr = nullptr;
};

// SkFontStyle (include/core/SkFontStyle.h): one int32.
struct SkFontStyle {
  int32_t fValue;
};

// SkFontStyleSet and SkFontMgr (include/core/SkFontMgr.h): the virtual
// interfaces, in declaration order. The other parameter types are only
// passed through (pointers, or non-trivial values passed indirectly).
class SkFontStyleSetMirror : public SkRefCntMirror {
 public:
  virtual int count() = 0;
  virtual void getStyle(int index, SkFontStyle* style, void* name) = 0;
  virtual SkSp<SkRefCntMirror> createTypeface(int index) = 0;
  virtual SkSp<SkRefCntMirror> matchStyle(const SkFontStyle& pattern) = 0;
};

class SkFontMgrMirror : public SkRefCntMirror {
 public:
  using Typeface = SkSp<SkRefCntMirror>;
  using StyleSet = SkSp<SkFontStyleSetMirror>;
  // std::unique_ptr<SkStreamAsset> and sk_sp<SkData> by value: non-trivial
  // one-pointer types, passed indirectly.
  using StreamArg = std::unique_ptr<SkRefCntMirror>;
  using DataArg = SkSp<SkRefCntMirror>;

  virtual int onCountFamilies() const = 0;
  virtual void onGetFamilyName(int index, void* family_name) const = 0;
  virtual StyleSet onCreateStyleSet(int index) const = 0;
  virtual StyleSet onMatchFamily(const char family_name[]) const = 0;
  virtual Typeface onMatchFamilyStyle(const char family_name[],
                                      const SkFontStyle& style) const = 0;
  virtual Typeface onMatchFamilyStyleCharacter(const char family_name[],
                                               const SkFontStyle& style,
                                               const char* bcp47[],
                                               int bcp47_count,
                                               int32_t character) const = 0;
  virtual Typeface onMatch(const void* request) const = 0;
  virtual Typeface onFallback(const void* request) const = 0;
  virtual Typeface onMakeFromData(DataArg data, int ttc_index) const = 0;
  virtual Typeface onMakeFromStreamIndex(StreamArg stream,
                                         int ttc_index) const = 0;
  virtual Typeface onMakeFromStreamArgs(StreamArg stream,
                                        const void* args) const = 0;
  virtual Typeface onMakeFromFile(const char path[], int ttc_index) const = 0;
  virtual Typeface onLegacyMakeTypeface(const char family_name[],
                                        SkFontStyle style) const = 0;
};

// --- display_list/dl_color.h, dl_paint.h ----------------------------------

struct DlColor {
  float alpha, red, green, blue;
  int32_t color_space;
};

struct DlPaint {
  uint32_t bits;  // blend mode, draw style, cap, join, AA, invert (bitfields)
  DlColor color;
  float stroke_width;
  float stroke_miter;
  std::shared_ptr<const void> color_source;
  std::shared_ptr<const void> color_filter;
  std::shared_ptr<void> image_filter;
  std::shared_ptr<const void> mask_filter;
};

// --- txt/src/txt ------------------------------------------------------------

enum class FontStyle { normal, italic };
enum class TextAlign { left, right, center, justify, start, end };
enum class TextDirection { rtl, ltr };

// text_shadow.h
struct TextShadow {
  SkColor color = 0xFF000000;
  SkPoint offset = {0, 0};
  double blur_sigma = 0.0;
};

// font_features.h
struct FontFeatures {
  std::map<std::string, int> feature_map;
};
struct FontVariations {
  std::map<std::string, float> axis_map;
};

// text_style.h (with the header's default member values)
struct TextStyle {
  SkColor color = 0xFFFFFFFF;  // SK_ColorWHITE
  int decoration = 0;
  SkColor decoration_color = 0;  // SK_ColorTRANSPARENT
  int decoration_style = 0;      // kSolid
  double decoration_thickness_multiplier = 1.0;
  int font_weight = 400;
  FontStyle font_style = FontStyle::normal;
  int text_baseline = 0;  // kAlphabetic
  bool half_leading = false;
  std::vector<std::string> font_families;
  double font_size = 14.0;
  double letter_spacing = 0.0;
  double word_spacing = 0.0;
  double height = 1.0;
  bool has_height_override = false;
  std::string locale;
  std::optional<DlPaint> background;
  std::optional<DlPaint> foreground;
  std::vector<TextShadow> text_shadows;
  FontFeatures font_features;
  FontVariations font_variations;
};

// paragraph_style.h
struct ParagraphStyle {
  int font_weight = 400;
  FontStyle font_style = FontStyle::normal;
  std::string font_family = "";
  double font_size = 14;
  double height = 1;
  bool has_height_override = false;
  size_t text_height_behavior = 0;  // kAll

  bool strut_enabled = false;
  int strut_font_weight = 400;
  FontStyle strut_font_style = FontStyle::normal;
  std::vector<std::string> strut_font_families;
  double strut_font_size = 14;
  double strut_height = 1;
  bool strut_has_height_override = false;
  bool strut_half_leading = false;
  double strut_leading = -1;
  bool force_strut_height = false;

  TextAlign text_align = TextAlign::start;
  TextDirection text_direction = TextDirection::ltr;
  size_t max_lines = SIZE_MAX;
  std::u16string ellipsis;
  std::string locale;

  bool render_soft_hyphens = true;
};

// placeholder_run.h
struct PlaceholderRun {
  double width = 0;
  double height = 0;
  int alignment = 0;  // PlaceholderAlignment
  int baseline = 0;   // TextBaseline
  double baseline_offset = 0;
};

// line_metrics.h. run_metrics maps to txt::RunMetrics; only the map object
// itself (whose layout doesn't depend on the value type) is mirrored.
struct LineMetrics {
  size_t start_index = 0;
  size_t end_index = 0;
  size_t end_excluding_whitespace = 0;
  size_t end_including_newline = 0;
  bool hard_break = false;
  double ascent = 0.0;
  double descent = 0.0;
  double unscaled_ascent = 0.0;
  double height = 0.0;
  double width = 0.0;
  double left = 0.0;
  double baseline = 0.0;
  size_t line_number = 0;
  std::map<size_t, int> run_metrics;
};

// modules/skparagraph/include/Metrics.h (skia::textlayout::LineMetrics)
struct SkLineMetrics {
  size_t fStartIndex = 0;
  size_t fEndIndex = 0;
  size_t fEndExcludingWhitespaces = 0;
  size_t fEndIncludingNewline = 0;
  bool fHardBreak = false;
  double fAscent = 0;
  double fDescent = 0;
  double fUnscaledAscent = 0;
  double fHeight = 0.0;
  double fWidth = 0.0;
  double fLeft = 0.0;
  double fBaseline = 0.0;
  size_t fLineNumber = 0;
  std::map<size_t, int> fLineMetrics;
};

// modules/skparagraph/include/Paragraph.h (Paragraph::GlyphInfo)
struct SkGlyphInfo {
  SkRect fGraphemeLayoutBounds;
  size_t fStart, fEnd;  // TextRange fGraphemeClusterTextRange
  int32_t fDirection;   // skia::textlayout::TextDirection (kRtl, kLtr)
  bool fIsEllipsis;
};

// paragraph.h: txt::Paragraph (every method virtual, in declaration order).
class Paragraph {
 public:
  enum Affinity { UPSTREAM, DOWNSTREAM };
  enum class RectHeightStyle {
    kTight,
    kMax,
    kIncludeLineSpacingMiddle,
    kIncludeLineSpacingTop,
    kIncludeLineSpacingBottom,
    kStrut
  };
  enum class RectWidthStyle { kTight, kMax };
  struct PositionWithAffinity {
    size_t position;
    Affinity affinity;
  };
  struct TextBox {
    SkRect rect;
    TextDirection direction;
  };
  struct Range {
    size_t start, end;
  };

  virtual ~Paragraph() = default;
  virtual double GetMaxWidth() = 0;
  virtual double GetHeight() = 0;
  virtual double GetLongestLine() = 0;
  virtual double GetMinIntrinsicWidth() = 0;
  virtual double GetMaxIntrinsicWidth() = 0;
  virtual double GetAlphabeticBaseline() = 0;
  virtual double GetIdeographicBaseline() = 0;
  virtual bool DidExceedMaxLines() = 0;
  virtual void Layout(double width) = 0;
  virtual bool Paint(void* display_list_builder, double x, double y) = 0;
  virtual std::vector<TextBox> GetRectsForRange(
      size_t start,
      size_t end,
      RectHeightStyle rect_height_style,
      RectWidthStyle rect_width_style) = 0;
  virtual std::vector<TextBox> GetRectsForPlaceholders() = 0;
  virtual PositionWithAffinity GetGlyphPositionAtCoordinate(double dx,
                                                            double dy) = 0;
  virtual bool GetGlyphInfoAt(unsigned offset,
                              SkGlyphInfo* glyph_info) const = 0;
  virtual bool GetClosestGlyphInfoAtCoordinate(
      double dx,
      double dy,
      SkGlyphInfo* glyph_info) const = 0;
  virtual Range GetWordBoundary(size_t offset) = 0;
  virtual std::vector<LineMetrics>& GetLineMetrics() = 0;
  virtual bool GetLineMetricsAt(int line_number,
                                SkLineMetrics* line_metrics) const = 0;
  virtual size_t GetNumberOfLines() const = 0;
  virtual int GetLineNumberAt(size_t utf16_offset) const = 0;
};

// paragraph_builder.h: txt::ParagraphBuilder.
class ParagraphBuilder {
 public:
  virtual ~ParagraphBuilder() = default;
  virtual void PushStyle(const TextStyle& style) = 0;
  virtual void Pop() = 0;
  virtual const TextStyle& PeekStyle() = 0;
  virtual void AddText(const std::u16string& text) = 0;
  virtual void AddText(const uint8_t* utf8_data, size_t byte_length) = 0;
  virtual void AddPlaceholder(PlaceholderRun& span) = 0;
  virtual std::unique_ptr<Paragraph> Build() = 0;
};

// font_collection.h: txt::FontCollection (enable_shared_from_this first).
struct FontCollection {
  std::weak_ptr<void> weak_this;
  SkRefCntMirror* default_font_manager;
  SkRefCntMirror* asset_font_manager;
  SkRefCntMirror* dynamic_font_manager;
  SkRefCntMirror* test_font_manager;
  bool enable_font_fallback;
  SkRefCntMirror* skt_collection;
};

// --- impeller/toolkit/interop --------------------------------------------

// object.h: ObjectBase (vtable, atomic_uint64_t ref count).
struct InteropObject {
  void* vtable;
  std::atomic<uint64_t> ref_count;
};

// typography_context.h
struct InteropTypographyContext : InteropObject {
  std::shared_ptr<FontCollection> collection;
  SkRefCntMirror* asset_font_manager;
};

// paragraph_style.h (the members this bridge touches come first).
struct InteropParagraphStyle : InteropObject {
  ParagraphStyle style;
};

// paragraph_builder.h
struct InteropParagraphBuilder : InteropObject {
  InteropTypographyContext* context;
  std::unique_ptr<ParagraphBuilder> lazy_builder;
};

// paint.h
struct InteropPaint : InteropObject {
  DlPaint paint;
};

// dl_builder.h: the flutter::DisplayListBuilder member's address is what
// txt::Paragraph::Paint takes.
struct InteropDisplayListBuilder : InteropObject {
  char builder[1];
};

}  // namespace fgo

#endif  // FLUTTER_GO_TXT_MIRROR_H_
