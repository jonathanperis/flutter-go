// The engine side of dart:ui's text.dart over the Impeller SDK. Ports of
// lib/ui/text/paragraph_builder.cc, paragraph.cc and font_collection.cc
// (Flutter 3.50.1) and of txt/src/txt/test_font_manager.cc, calling the
// SDK's txt objects through the mirrors in txt_mirror.h.

#include "typography_bridge.h"

#include <cmath>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <vector>

#include "txt_mirror.h"

using namespace fgo;

namespace {

// --- Font collection --------------------------------------------------------

// A copy of font data handed to the SDK as an ImpellerMapping.
ImpellerMapping CopyMapping(const uint8_t* data, size_t length) {
  auto* copy = static_cast<uint8_t*>(malloc(length));
  memcpy(copy, data, length);
  ImpellerMapping mapping = {};
  mapping.data = copy;
  mapping.length = length;
  mapping.on_release = [](void* user_data) { free(user_data); };
  return mapping;
}

// txt::TestFontManager (an AssetFontManager over the test typefaces): every
// family name that is not one of the test fonts' resolves to the first one.
// The typefaces live in an SDK typography context's TypefaceFontProvider.
class TestFontManager final : public SkFontMgrMirror {
 public:
  TestFontManager(SkFontMgrMirror* provider, std::vector<std::string> names)
      : provider_(provider), names_(std::move(names)) {
    provider_->ref();
  }
  ~TestFontManager() override { provider_->unref(); }

  int onCountFamilies() const override { return provider_->onCountFamilies(); }
  void onGetFamilyName(int index, void* family_name) const override {
    provider_->onGetFamilyName(index, family_name);
  }
  StyleSet onCreateStyleSet(int index) const override { return StyleSet(); }

  // TestFontManager::onMatchFamily.
  StyleSet onMatchFamily(const char family_name[]) const override {
    std::string requested_name(family_name);
    std::string sanitized_name = names_[0];
    for (const std::string& test_family : names_) {
      if (requested_name == test_family) {
        sanitized_name = test_family;
      }
    }
    return MatchFamily(sanitized_name);
  }

  // AssetFontManager::onMatchFamilyStyle (no sanitizing).
  Typeface onMatchFamilyStyle(const char family_name[],
                              const SkFontStyle& style) const override {
    StyleSet set = MatchFamily(family_name);
    if (set.get() == nullptr) {
      return Typeface();
    }
    return set.get()->matchStyle(style);
  }

  Typeface onMatchFamilyStyleCharacter(const char[],
                                       const SkFontStyle&,
                                       const char*[],
                                       int,
                                       int32_t) const override {
    return Typeface();
  }
  Typeface onMatch(const void*) const override { return Typeface(); }
  Typeface onFallback(const void*) const override { return Typeface(); }
  Typeface onMakeFromData(DataArg, int) const override { return Typeface(); }
  Typeface onMakeFromStreamIndex(StreamArg, int) const override {
    return Typeface();
  }
  Typeface onMakeFromStreamArgs(StreamArg, const void*) const override {
    return Typeface();
  }
  Typeface onMakeFromFile(const char[], int) const override {
    return Typeface();
  }
  Typeface onLegacyMakeTypeface(const char[], SkFontStyle) const override {
    return Typeface();
  }

 private:
  // FontAssetProvider::MatchFamily canonicalizes names (lower case); the
  // provider was given the names as registered, so match them that way.
  StyleSet MatchFamily(const std::string& name) const {
    std::string lower = Canonical(name);
    for (const std::string& test_family : names_) {
      if (Canonical(test_family) == lower) {
        return provider_->onMatchFamily(test_family.c_str());
      }
    }
    return StyleSet();
  }
  static std::string Canonical(const std::string& s) {
    std::string r = s;
    for (char& c : r) {
      c = static_cast<char>(tolower(static_cast<unsigned char>(c)));
    }
    return r;
  }

  SkFontMgrMirror* provider_;
  std::vector<std::string> names_;
};

}  // namespace

struct FgoFontCollection {
  ImpellerTypographyContext context = nullptr;
  ImpellerTypographyContext test_context = nullptr;

  FontCollection* collection() const {
    return reinterpret_cast<InteropTypographyContext*>(context)
        ->collection.get();
  }

  // FontCollection::ClearFontFamilyCache: drops the SkParagraph collection
  // (with its caches); the next builder makes a new one.
  void ClearFontFamilyCache() {
    FontCollection* fc = collection();
    if (fc->skt_collection != nullptr) {
      fc->skt_collection->unref();
      fc->skt_collection = nullptr;
    }
  }
};

extern "C" FgoFontCollection* fgo_font_collection_new(
    const uint8_t* const* test_fonts,
    const size_t* test_font_lengths,
    const char* const* test_font_names,
    size_t test_font_count) {
  auto* result = new FgoFontCollection();
  result->context = ImpellerTypographyContextNew();
  if (test_font_count == 0) {
    return result;
  }
  // FontCollection::RegisterTestFonts.
  result->test_context = ImpellerTypographyContextNew();
  std::vector<std::string> names;
  for (size_t i = 0; i < test_font_count; i++) {
    ImpellerMapping mapping = CopyMapping(test_fonts[i], test_font_lengths[i]);
    ImpellerTypographyContextRegisterFont(result->test_context, &mapping,
                                          const_cast<uint8_t*>(mapping.data), test_font_names[i]);
    names.emplace_back(test_font_names[i]);
  }
  auto* test_provider = static_cast<SkFontMgrMirror*>(
      reinterpret_cast<InteropTypographyContext*>(result->test_context)
          ->asset_font_manager);
  FontCollection* fc = result->collection();
  if (fc->test_font_manager != nullptr) {
    fc->test_font_manager->unref();
  }
  fc->test_font_manager = new TestFontManager(test_provider, names);
  // collection_->DisableFontFallback().
  fc->enable_font_fallback = false;
  result->ClearFontFamilyCache();
  return result;
}

extern "C" bool fgo_font_collection_load_font(FgoFontCollection* collection,
                                              const uint8_t* data,
                                              size_t length,
                                              const char* family) {
  ImpellerMapping mapping = CopyMapping(data, length);
  bool ok = ImpellerTypographyContextRegisterFont(
      collection->context, &mapping, const_cast<uint8_t*>(mapping.data), family);
  collection->ClearFontFamilyCache();
  return ok;
}

// --- ParagraphBuilder (lib/ui/text/paragraph_builder.cc) --------------------

namespace {

const double kTextHeightNone = 0.0;

// TextStyle
const int kTSLeadingDistributionIndex = 0;
const int kTSColorIndex = 1;
const int kTSTextDecorationIndex = 2;
const int kTSTextDecorationColorIndex = 3;
const int kTSTextDecorationStyleIndex = 4;
const int kTSFontWeightIndex = 5;
const int kTSFontStyleIndex = 6;
const int kTSTextBaselineIndex = 7;
const int kTSTextDecorationThicknessIndex = 8;
const int kTSFontFamilyIndex = 9;
const int kTSFontSizeIndex = 10;
const int kTSLetterSpacingIndex = 11;
const int kTSWordSpacingIndex = 12;
const int kTSHeightIndex = 13;
const int kTSLocaleIndex = 14;
const int kTSBackgroundIndex = 15;
const int kTSForegroundIndex = 16;
const int kTSTextShadowsIndex = 17;
const int kTSFontFeaturesIndex = 18;
const int kTSFontVariationsIndex = 19;

const int kTSLeadingDistributionMask = 1 << kTSLeadingDistributionIndex;
const int kTSColorMask = 1 << kTSColorIndex;
const int kTSTextDecorationMask = 1 << kTSTextDecorationIndex;
const int kTSTextDecorationColorMask = 1 << kTSTextDecorationColorIndex;
const int kTSTextDecorationStyleMask = 1 << kTSTextDecorationStyleIndex;
const int kTSTextDecorationThicknessMask = 1 << kTSTextDecorationThicknessIndex;
const int kTSFontWeightMask = 1 << kTSFontWeightIndex;
const int kTSFontStyleMask = 1 << kTSFontStyleIndex;
const int kTSTextBaselineMask = 1 << kTSTextBaselineIndex;
const int kTSFontFamilyMask = 1 << kTSFontFamilyIndex;
const int kTSFontSizeMask = 1 << kTSFontSizeIndex;
const int kTSLetterSpacingMask = 1 << kTSLetterSpacingIndex;
const int kTSWordSpacingMask = 1 << kTSWordSpacingIndex;
const int kTSHeightMask = 1 << kTSHeightIndex;
const int kTSLocaleMask = 1 << kTSLocaleIndex;
const int kTSBackgroundMask = 1 << kTSBackgroundIndex;
const int kTSForegroundMask = 1 << kTSForegroundIndex;
const int kTSTextShadowsMask = 1 << kTSTextShadowsIndex;
const int kTSFontFeaturesMask = 1 << kTSFontFeaturesIndex;
const int kTSFontVariationsMask = 1 << kTSFontVariationsIndex;

// ParagraphStyle
const int kPSTextAlignIndex = 1;
const int kPSTextDirectionIndex = 2;
const int kPSFontWeightIndex = 3;
const int kPSFontStyleIndex = 4;
const int kPSMaxLinesIndex = 5;
const int kPSTextHeightBehaviorIndex = 6;
const int kPSFontFamilyIndex = 7;
const int kPSFontSizeIndex = 8;
const int kPSHeightIndex = 9;
const int kPSStrutStyleIndex = 10;
const int kPSEllipsisIndex = 11;
const int kPSLocaleIndex = 12;
const int kPSRenderSoftHyphensIndex = 13;
const int kPSRenderSoftHyphensEncodeIndex = 7;

const int kPSTextAlignMask = 1 << kPSTextAlignIndex;
const int kPSTextDirectionMask = 1 << kPSTextDirectionIndex;
const int kPSFontWeightMask = 1 << kPSFontWeightIndex;
const int kPSFontStyleMask = 1 << kPSFontStyleIndex;
const int kPSMaxLinesMask = 1 << kPSMaxLinesIndex;
const int kPSFontFamilyMask = 1 << kPSFontFamilyIndex;
const int kPSFontSizeMask = 1 << kPSFontSizeIndex;
const int kPSHeightMask = 1 << kPSHeightIndex;
const int kPSTextHeightBehaviorMask = 1 << kPSTextHeightBehaviorIndex;
const int kPSStrutStyleMask = 1 << kPSStrutStyleIndex;
const int kPSEllipsisMask = 1 << kPSEllipsisIndex;
const int kPSLocaleMask = 1 << kPSLocaleIndex;
const int kPSRenderSoftHyphensMask = 1 << kPSRenderSoftHyphensIndex;

// TextShadows decoding
constexpr uint32_t kColorDefault = 0xFF000000;
constexpr uint32_t kBytesPerShadow = 16;
constexpr uint32_t kShadowPropertiesCount = 4;
constexpr uint32_t kColorOffset = 0;
constexpr uint32_t kXOffset = 1;
constexpr uint32_t kYOffset = 2;
constexpr uint32_t kBlurOffset = 3;

// FontFeature decoding
constexpr uint32_t kBytesPerFontFeature = 8;
constexpr uint32_t kFontFeatureTagLength = 4;

// FontVariation decoding
constexpr uint32_t kBytesPerFontVariation = 8;
constexpr uint32_t kFontVariationTagLength = 4;

// Strut decoding
const int kSFontWeightIndex = 0;
const int kSFontStyleIndex = 1;
const int kSFontFamilyIndex = 2;
const int kSLeadingDistributionIndex = 3;
const int kSFontSizeIndex = 4;
const int kSHeightIndex = 5;
const int kSLeadingIndex = 6;
const int kSForceStrutHeightIndex = 7;

const int kSFontWeightMask = 1 << kSFontWeightIndex;
const int kSFontStyleMask = 1 << kSFontStyleIndex;
const int kSFontFamilyMask = 1 << kSFontFamilyIndex;
const int kSLeadingDistributionMask = 1 << kSLeadingDistributionIndex;
const int kSFontSizeMask = 1 << kSFontSizeIndex;
const int kSHeightMask = 1 << kSHeightIndex;
const int kSLeadingMask = 1 << kSLeadingIndex;
const int kSForceStrutHeightMask = 1 << kSForceStrutHeightIndex;

void decodeStrut(const uint8_t* strut_data,
                 size_t strut_length,
                 const std::vector<std::string>& strut_font_families,
                 ParagraphStyle& paragraph_style) {
  if (strut_data == nullptr) {
    return;
  }
  if (strut_length == 0) {
    return;
  }
  paragraph_style.strut_enabled = true;

  const int32_t* int32_data = reinterpret_cast<const int32_t*>(strut_data);
  int32_t mask = int32_data[0];

  size_t int32_count = 1;
  if (mask & kSFontWeightMask) {
    paragraph_style.strut_font_weight = int32_data[int32_count++];
  }
  if (mask & kSFontStyleMask) {
    paragraph_style.strut_font_style =
        static_cast<FontStyle>(int32_data[int32_count++]);
  }

  paragraph_style.strut_half_leading = mask & kSLeadingDistributionMask;

  std::vector<float> float_data;
  float_data.resize((strut_length / 4) - int32_count);
  memcpy(float_data.data(), int32_data + int32_count, float_data.size() * 4);
  size_t float_count = 0;
  if (mask & kSFontSizeMask) {
    paragraph_style.strut_font_size = float_data[float_count++];
  }
  if (mask & kSHeightMask) {
    paragraph_style.strut_height = float_data[float_count++];
    paragraph_style.strut_has_height_override = true;
  }
  if (mask & kSLeadingMask) {
    paragraph_style.strut_leading = float_data[float_count++];
  }

  // The boolean is stored as the last bit in the bitmask, as null
  // and false have the same behavior.
  paragraph_style.force_strut_height = mask & kSForceStrutHeightMask;

  if (mask & kSFontFamilyMask) {
    paragraph_style.strut_font_families = strut_font_families;
  } else {
    // Provide an empty font name so that the platform default font will be
    // used.
    paragraph_style.strut_font_families.push_back("");
  }
}

void decodeTextShadows(const uint8_t* data,
                       size_t length,
                       std::vector<TextShadow>& decoded_shadows) {
  decoded_shadows.clear();
  const uint32_t* uint_data = reinterpret_cast<const uint32_t*>(data);
  const float* float_data = reinterpret_cast<const float*>(data);

  size_t shadow_count = length / kBytesPerShadow;
  size_t shadow_count_offset = 0;
  for (size_t shadow_index = 0; shadow_index < shadow_count; ++shadow_index) {
    shadow_count_offset = shadow_index * kShadowPropertiesCount;
    TextShadow shadow;
    shadow.color =
        uint_data[shadow_count_offset + kColorOffset] ^ kColorDefault;
    shadow.offset = {float_data[shadow_count_offset + kXOffset],
                     float_data[shadow_count_offset + kYOffset]};
    shadow.blur_sigma = float_data[shadow_count_offset + kBlurOffset];
    decoded_shadows.push_back(shadow);
  }
}

void decodeFontFeatures(const uint8_t* data,
                        size_t length,
                        FontFeatures& font_features) {
  size_t feature_count = length / kBytesPerFontFeature;
  for (size_t feature_index = 0; feature_index < feature_count;
       ++feature_index) {
    size_t feature_offset = feature_index * kBytesPerFontFeature;
    const char* feature_bytes =
        reinterpret_cast<const char*>(data) + feature_offset;
    std::string tag(feature_bytes, kFontFeatureTagLength);
    int32_t value;
    memcpy(&value, feature_bytes + kFontFeatureTagLength, 4);
    // FontFeatures::SetFeature.
    font_features.feature_map[tag] = value;
  }
}

void decodeFontVariations(const uint8_t* data,
                          size_t length,
                          FontVariations& font_variations) {
  size_t variation_count = length / kBytesPerFontVariation;
  for (size_t variation_index = 0; variation_index < variation_count;
       ++variation_index) {
    size_t variation_offset = variation_index * kBytesPerFontVariation;
    const char* variation_bytes =
        reinterpret_cast<const char*>(data) + variation_offset;
    std::string tag(variation_bytes, kFontVariationTagLength);
    float value;
    memcpy(&value, variation_bytes + kFontVariationTagLength, 4);
    // FontVariations::SetAxisValue.
    font_variations.axis_map[tag] = value;
  }
}

std::vector<std::string> Strings(const char* const* strings, size_t count) {
  std::vector<std::string> result;
  for (size_t i = 0; i < count; i++) {
    result.emplace_back(strings[i]);
  }
  return result;
}

}  // namespace

struct FgoParagraphBuilder {
  ParagraphBuilder* builder;
};

struct FgoParagraph {
  Paragraph* paragraph;
};

extern "C" FgoParagraphBuilder* fgo_paragraph_builder_new(
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
    bool impeller_enabled) {
  (void)encoded_length;
  (void)impeller_enabled;  // The SDK's builders are always Impeller's.
  int32_t mask = 0;
  ParagraphStyle style;
  {
    mask = encoded[0];

    if (mask & kPSTextAlignMask) {
      style.text_align = static_cast<TextAlign>(encoded[kPSTextAlignIndex]);
    }

    if (mask & kPSTextDirectionMask) {
      style.text_direction =
          static_cast<TextDirection>(encoded[kPSTextDirectionIndex]);
    }

    if (mask & kPSFontWeightMask) {
      style.font_weight = encoded[kPSFontWeightIndex];
    }

    if (mask & kPSFontStyleMask) {
      style.font_style = static_cast<FontStyle>(encoded[kPSFontStyleIndex]);
    }

    if (mask & kPSFontFamilyMask) {
      style.font_family = font_family;
    }

    if (mask & kPSFontSizeMask) {
      style.font_size = font_size;
    }

    if (mask & kPSHeightMask) {
      style.height = height;
      style.has_height_override = true;
    }

    if (mask & kPSTextHeightBehaviorMask) {
      style.text_height_behavior = encoded[kPSTextHeightBehaviorIndex];
    }

    if (mask & kPSMaxLinesMask) {
      style.max_lines = encoded[kPSMaxLinesIndex];
    }

    if (mask & kPSRenderSoftHyphensMask) {
      // Hyphens.manual (0) renders the soft hyphen glyph; Hyphens.hidden (1)
      // suppresses it.
      style.render_soft_hyphens =
          encoded[kPSRenderSoftHyphensEncodeIndex] == 0;
    }
  }

  if (mask & kPSStrutStyleMask) {
    decodeStrut(strut_data, strut_length,
                Strings(strut_families, strut_family_count), style);
  }

  if (mask & kPSEllipsisMask) {
    style.ellipsis = std::u16string(
        reinterpret_cast<const char16_t*>(ellipsis), ellipsis_length);
  }

  if (mask & kPSLocaleMask) {
    style.locale = locale;
  }

  // txt::ParagraphBuilder::CreateSkiaBuilder(style, font_collection, ...):
  // the SDK's builder creates its ParagraphBuilderSkia from the style of the
  // first style pushed, so push one carrying this paragraph style and pop it
  // (SkParagraph's builder is then in its initial state: one empty block
  // with the paragraph's text style).
  ImpellerParagraphStyle sdk_style = ImpellerParagraphStyleNew();
  reinterpret_cast<InteropParagraphStyle*>(sdk_style)->style = style;
  ImpellerParagraphBuilder sdk_builder =
      ImpellerParagraphBuilderNew(collection->context);
  ImpellerParagraphBuilderPushStyle(sdk_builder, sdk_style);
  auto* result = new FgoParagraphBuilder();
  result->builder = reinterpret_cast<InteropParagraphBuilder*>(sdk_builder)
                        ->lazy_builder.release();
  ImpellerParagraphBuilderRelease(sdk_builder);
  ImpellerParagraphStyleRelease(sdk_style);
  result->builder->Pop();
  return result;
}

extern "C" void fgo_paragraph_builder_push_style(
    FgoParagraphBuilder* builder,
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
    size_t font_variations_length) {
  int32_t mask = encoded[0];

  // Set to use the properties of the previous style if the property is not
  // explicitly given.
  TextStyle style = builder->builder->PeekStyle();

  style.half_leading = mask & kTSLeadingDistributionMask;
  // Only change the style property from the previous value if a new explicitly
  // set value is available
  if (mask & kTSColorMask) {
    style.color = encoded[kTSColorIndex];
  }

  if (mask & kTSTextDecorationMask) {
    style.decoration = encoded[kTSTextDecorationIndex];
  }

  if (mask & kTSTextDecorationColorMask) {
    style.decoration_color = encoded[kTSTextDecorationColorIndex];
  }

  if (mask & kTSTextDecorationStyleMask) {
    style.decoration_style = encoded[kTSTextDecorationStyleIndex];
  }

  if (mask & kTSTextDecorationThicknessMask) {
    style.decoration_thickness_multiplier = decoration_thickness;
  }

  if (mask & kTSTextBaselineMask) {
    // TODO(abarth): Implement TextBaseline. The CSS version of this
    // property wasn't wired up either.
  }

  if (mask & (kTSFontWeightMask | kTSFontStyleMask | kTSFontSizeMask |
              kTSLetterSpacingMask | kTSWordSpacingMask)) {
    if (mask & kTSFontWeightMask) {
      style.font_weight = encoded[kTSFontWeightIndex];
    }

    if (mask & kTSFontStyleMask) {
      style.font_style = static_cast<FontStyle>(encoded[kTSFontStyleIndex]);
    }

    if (mask & kTSFontSizeMask) {
      style.font_size = font_size;
    }

    if (mask & kTSLetterSpacingMask) {
      style.letter_spacing = letter_spacing;
    }

    if (mask & kTSWordSpacingMask) {
      style.word_spacing = word_spacing;
    }
  }

  if (mask & kTSHeightMask) {
    style.height = height;
    style.has_height_override = style.height != kTextHeightNone;
  }

  if (mask & kTSLocaleMask) {
    style.locale = locale;
  }

  // Paint::paint(dl_paint, kDrawParagraphFlags, kDecal) is done by the
  // caller, which hands over the resulting SDK paint.
  if (mask & kTSBackgroundMask) {
    if (background != nullptr) {
      style.background = reinterpret_cast<InteropPaint*>(background)->paint;
    }
  }

  if (mask & kTSForegroundMask) {
    if (foreground != nullptr) {
      style.foreground = reinterpret_cast<InteropPaint*>(foreground)->paint;
    }
  }

  if (mask & kTSTextShadowsMask) {
    decodeTextShadows(shadows, shadows_length, style.text_shadows);
  }

  if (mask & kTSFontFamilyMask) {
    // The child style's font families override the parent's font families.
    // If the child's fonts are not available, then the font collection will
    // use the system fallback fonts (not the parent's fonts).
    style.font_families = Strings(font_families, font_family_count);
  }

  if (mask & kTSFontFeaturesMask) {
    decodeFontFeatures(font_features, font_features_length,
                       style.font_features);
  }

  if (mask & kTSFontVariationsMask) {
    decodeFontVariations(font_variations, font_variations_length,
                         style.font_variations);
  }

  builder->builder->PushStyle(style);
}

extern "C" void fgo_paragraph_builder_pop(FgoParagraphBuilder* builder) {
  builder->builder->Pop();
}

extern "C" void fgo_paragraph_builder_add_text(FgoParagraphBuilder* builder,
                                               const uint16_t* text,
                                               size_t length) {
  if (length == 0) {
    return;
  }
  builder->builder->AddText(
      std::u16string(reinterpret_cast<const char16_t*>(text), length));
}

extern "C" void fgo_paragraph_builder_add_placeholder(
    FgoParagraphBuilder* builder,
    double width,
    double height,
    uint32_t alignment,
    double baseline_offset,
    uint32_t baseline) {
  PlaceholderRun placeholder_run;
  placeholder_run.width = width;
  placeholder_run.height = height;
  placeholder_run.alignment = static_cast<int>(alignment);
  placeholder_run.baseline = static_cast<int>(baseline);
  placeholder_run.baseline_offset = baseline_offset;
  builder->builder->AddPlaceholder(placeholder_run);
}

extern "C" FgoParagraph* fgo_paragraph_builder_build(
    FgoParagraphBuilder* builder) {
  auto* result = new FgoParagraph();
  result->paragraph = builder->builder->Build().release();
  fgo_paragraph_builder_release(builder);
  return result;
}

extern "C" void fgo_paragraph_builder_release(FgoParagraphBuilder* builder) {
  delete builder->builder;
  delete builder;
}

// --- Paragraph (lib/ui/text/paragraph.cc) -----------------------------------

extern "C" void fgo_paragraph_metrics(FgoParagraph* p,
                                      FgoParagraphMetrics* out) {
  Paragraph* m = p->paragraph;
  out->width = m->GetMaxWidth();
  out->height = m->GetHeight();
  out->longest_line = m->GetLongestLine();
  out->min_intrinsic_width = m->GetMinIntrinsicWidth();
  out->max_intrinsic_width = m->GetMaxIntrinsicWidth();
  out->alphabetic_baseline = m->GetAlphabeticBaseline();
  out->ideographic_baseline = m->GetIdeographicBaseline();
  out->did_exceed_max_lines = m->DidExceedMaxLines();
}

extern "C" void fgo_paragraph_layout(FgoParagraph* p, double width) {
  p->paragraph->Layout(width);
}

extern "C" void fgo_paragraph_paint(FgoParagraph* p,
                                    ImpellerDisplayListBuilder builder,
                                    double x,
                                    double y) {
  if (p->paragraph == nullptr || builder == nullptr) {
    // disposed.
    return;
  }
  p->paragraph->Paint(
      reinterpret_cast<InteropDisplayListBuilder*>(builder)->builder, x, y);
}

namespace {

size_t EncodeTextBoxes(const std::vector<Paragraph::TextBox>& boxes,
                       float** out) {
  // Layout: boxes.size() groups of 5 which are LTRBD, where D is the text
  // direction index.
  float* result = static_cast<float*>(malloc((boxes.size() * 5 + 1) * 4));
  uint64_t position = 0;
  for (uint64_t i = 0; i < boxes.size(); i++) {
    const Paragraph::TextBox& box = boxes[i];
    result[position++] = box.rect.fLeft;
    result[position++] = box.rect.fTop;
    result[position++] = box.rect.fRight;
    result[position++] = box.rect.fBottom;
    result[position++] = static_cast<float>(box.direction);
  }
  *out = result;
  return boxes.size();
}

void GlyphInfoFrom(const SkGlyphInfo& glyph_info, FgoGlyphInfo* out) {
  out->left = glyph_info.fGraphemeLayoutBounds.fLeft;
  out->top = glyph_info.fGraphemeLayoutBounds.fTop;
  out->right = glyph_info.fGraphemeLayoutBounds.fRight;
  out->bottom = glyph_info.fGraphemeLayoutBounds.fBottom;
  out->start = glyph_info.fStart;
  out->end = glyph_info.fEnd;
  out->is_ltr = glyph_info.fDirection == 1;  // TextDirection::kLtr
}

}  // namespace

extern "C" size_t fgo_paragraph_get_rects_for_range(FgoParagraph* p,
                                                    uint32_t start,
                                                    uint32_t end,
                                                    uint32_t height_style,
                                                    uint32_t width_style,
                                                    float** out) {
  std::vector<Paragraph::TextBox> boxes = p->paragraph->GetRectsForRange(
      start, end, static_cast<Paragraph::RectHeightStyle>(height_style),
      static_cast<Paragraph::RectWidthStyle>(width_style));
  return EncodeTextBoxes(boxes, out);
}

extern "C" size_t fgo_paragraph_get_rects_for_placeholders(FgoParagraph* p,
                                                           float** out) {
  std::vector<Paragraph::TextBox> boxes =
      p->paragraph->GetRectsForPlaceholders();
  return EncodeTextBoxes(boxes, out);
}

extern "C" void fgo_paragraph_get_position_for_offset(FgoParagraph* p,
                                                      double dx,
                                                      double dy,
                                                      size_t out[2]) {
  Paragraph::PositionWithAffinity pos =
      p->paragraph->GetGlyphPositionAtCoordinate(dx, dy);
  out[0] = pos.position;
  out[1] = static_cast<size_t>(pos.affinity);
}

extern "C" bool fgo_paragraph_get_glyph_info_at(FgoParagraph* p,
                                                uint32_t offset,
                                                FgoGlyphInfo* out) {
  SkGlyphInfo glyph_info;
  const bool found = p->paragraph->GetGlyphInfoAt(offset, &glyph_info);
  if (found) {
    GlyphInfoFrom(glyph_info, out);
  }
  return found;
}

extern "C" bool fgo_paragraph_get_closest_glyph_info(FgoParagraph* p,
                                                     double dx,
                                                     double dy,
                                                     FgoGlyphInfo* out) {
  SkGlyphInfo glyph_info;
  const bool found =
      p->paragraph->GetClosestGlyphInfoAtCoordinate(dx, dy, &glyph_info);
  if (found) {
    GlyphInfoFrom(glyph_info, out);
  }
  return found;
}

extern "C" void fgo_paragraph_get_word_boundary(FgoParagraph* p,
                                                uint32_t offset,
                                                size_t out[2]) {
  Paragraph::Range point = p->paragraph->GetWordBoundary(offset);
  out[0] = point.start;
  out[1] = point.end;
}

extern "C" void fgo_paragraph_get_line_boundary(FgoParagraph* p,
                                                uint32_t utf16_offset,
                                                int32_t out[2]) {
  std::vector<LineMetrics> metrics = p->paragraph->GetLineMetrics();
  int line_start = -1;
  int line_end = -1;
  for (LineMetrics& line : metrics) {
    if (utf16_offset >= line.start_index && utf16_offset <= line.end_index) {
      line_start = line.start_index;
      line_end = line.end_index;
      break;
    }
  }
  out[0] = line_start;
  out[1] = line_end;
}

extern "C" size_t fgo_paragraph_compute_line_metrics(FgoParagraph* p,
                                                     double** out) {
  std::vector<LineMetrics>& metrics = p->paragraph->GetLineMetrics();
  double* result = static_cast<double*>(malloc((metrics.size() * 9 + 1) * 8));
  uint64_t position = 0;
  for (uint64_t i = 0; i < metrics.size(); i++) {
    const LineMetrics& line = metrics[i];
    result[position++] = static_cast<double>(line.hard_break);
    result[position++] = line.ascent;
    result[position++] = line.descent;
    result[position++] = line.unscaled_ascent;
    // We add then round to get the height. The
    // definition of height here is different
    // than the one in LibTxt.
    result[position++] = round(line.ascent + line.descent);
    result[position++] = line.width;
    result[position++] = line.left;
    result[position++] = line.baseline;
    result[position++] = static_cast<double>(line.line_number);
  }
  *out = result;
  return metrics.size();
}

extern "C" bool fgo_paragraph_get_line_metrics_at(FgoParagraph* p,
                                                  int32_t line_number,
                                                  double out[9]) {
  SkLineMetrics line;
  const bool found = p->paragraph->GetLineMetricsAt(line_number, &line);
  if (!found) {
    return false;
  }
  out[0] = line.fHardBreak ? 1 : 0;
  out[1] = line.fAscent;
  out[2] = line.fDescent;
  out[3] = line.fUnscaledAscent;
  // We add then round to get the height. The
  // definition of height here is different
  // than the one in LibTxt.
  out[4] = round(line.fAscent + line.fDescent);
  out[5] = line.fWidth;
  out[6] = line.fLeft;
  out[7] = line.fBaseline;
  out[8] = static_cast<double>(line.fLineNumber);
  return true;
}

extern "C" size_t fgo_paragraph_get_number_of_lines(FgoParagraph* p) {
  return p->paragraph->GetNumberOfLines();
}

extern "C" int32_t fgo_paragraph_get_line_number_at(FgoParagraph* p,
                                                    size_t utf16_offset) {
  return p->paragraph->GetLineNumberAt(utf16_offset);
}

extern "C" void fgo_paragraph_dispose(FgoParagraph* p) {
  delete p->paragraph;
  delete p;
}

extern "C" void fgo_free(void* p) {
  free(p);
}

// --- Layout checks -----------------------------------------------------------

extern "C" bool fgo_typography_self_check(char* message, size_t size) {
  bool ok = true;
  std::string report;
  auto check = [&](bool cond, const char* what) {
    if (!cond) {
      ok = false;
      report += what;
      report += "; ";
    }
  };

  // ParagraphStyle through the SDK's setters.
  ImpellerParagraphStyle sdk_style = ImpellerParagraphStyleNew();
  ImpellerParagraphStyleSetFontFamily(sdk_style,
                                      "A family name longer than SSO");
  ImpellerParagraphStyleSetFontSize(sdk_style, 21);
  ImpellerParagraphStyleSetHeight(sdk_style, 1.5);
  ImpellerParagraphStyleSetFontWeight(sdk_style, kImpellerFontWeight700);
  ImpellerParagraphStyleSetTextAlignment(sdk_style,
                                         kImpellerTextAlignmentCenter);
  ImpellerParagraphStyleSetTextDirection(sdk_style, kImpellerTextDirectionRTL);
  ImpellerParagraphStyleSetMaxLines(sdk_style, 7);
  ImpellerParagraphStyleSetLocale(sdk_style, "fr_CA");
  ImpellerParagraphStyleSetEllipsis(sdk_style, "\xE2\x80\xA6!");
  const ParagraphStyle& s =
      reinterpret_cast<InteropParagraphStyle*>(sdk_style)->style;
  check(s.font_family == "A family name longer than SSO", "font_family");
  check(s.font_size == 21, "font_size");
  check(s.height == 1.5 && s.has_height_override, "height");
  check(s.font_weight == 700, "font_weight");
  check(s.text_align == TextAlign::center, "text_align");
  check(s.text_direction == TextDirection::rtl, "text_direction");
  check(s.max_lines == 7, "max_lines");
  check(s.locale == "fr_CA", "locale");
  check(s.ellipsis == u"…!", "ellipsis");
  check(s.render_soft_hyphens, "render_soft_hyphens");
  check(s.strut_leading == -1 && s.strut_font_size == 14, "strut defaults");

  // TextStyle through ParagraphStyle::GetTextStyle (the builder's base).
  ImpellerTypographyContext context = ImpellerTypographyContextNew();
  check(reinterpret_cast<InteropTypographyContext*>(context)
                ->collection.get()
                ->asset_font_manager ==
            reinterpret_cast<InteropTypographyContext*>(context)
                ->asset_font_manager,
        "font collection");
  ImpellerParagraphBuilder sdk_builder = ImpellerParagraphBuilderNew(context);
  ImpellerParagraphBuilderPushStyle(sdk_builder, sdk_style);
  ParagraphBuilder* b =
      reinterpret_cast<InteropParagraphBuilder*>(sdk_builder)
          ->lazy_builder.get();
  const TextStyle& t = b->PeekStyle();
  check(t.font_families.size() == 1 &&
            t.font_families[0] == "A family name longer than SSO",
        "text style font_families");
  check(t.font_size == 21 && t.height == 1.5 && t.has_height_override,
        "text style size/height");
  check(t.font_weight == 700 && t.locale == "fr_CA", "text style weight");
  check(t.decoration_thickness_multiplier == 1.0 && t.letter_spacing == 0 &&
            !t.background.has_value(),
        "text style defaults");
  check(t.foreground.has_value() == false, "text style foreground");
  ImpellerParagraphBuilderRelease(sdk_builder);
  ImpellerParagraphStyleRelease(sdk_style);
  ImpellerTypographyContextRelease(context);

  snprintf(message, size, "%s", report.c_str());
  return ok;
}
