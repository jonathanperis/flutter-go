package ui

// Port of engine/src/flutter/testing/dart/text_test.dart (Flutter 3.50.1).
//
// Skipped: "FontWeight is applied to variable fonts" compares a rendering
// with the engine's golden image (text_test_fontWeightVariable.png, Skia
// Gold); the same paragraph's layout is checked instead (the weights widen
// the text), and drawParagraph is covered by TestTextDrawParagraphRecords.

import (
	"math"
	"os"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
)

func readResource(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../testing/resources/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func loadFont(t *testing.T, data []byte, family string) {
	t.Helper()
	done := false
	async.OnComplete(LoadFontFromList(data, LoadFontFromListOptions{FontFamily: &family}),
		func(async.Void) { done = true }, func(err error) { t.Fatal(err) })
	async.Current().FlushMicrotasks()
	if !done {
		t.Fatal("loadFontFromList did not complete")
	}
}

// testFontWeight

func TestFontWeightLerpWorksWithNonNullValues(t *testing.T) {
	eq(t, "lerp", *FontWeightLerp(&FontWeightW400, &FontWeightW600, .5), FontWeightW500)
}

func TestFontWeightLerpReturnsNullIfAAndBAreNull(t *testing.T) {
	if FontWeightLerp(nil, nil, 0) != nil {
		t.Error("not null")
	}
}

func TestFontWeightLerpReturnsW400IfAIsNull(t *testing.T) {
	eq(t, "lerp", *FontWeightLerp(nil, &FontWeightW400, 0), FontWeightW400)
}

func TestFontWeightLerpReturnsW400IfBIsNull(t *testing.T) {
	eq(t, "lerp", *FontWeightLerp(&FontWeightW400, nil, 1), FontWeightW400)
}

func TestFontWeightsHaveTheCorrectValue(t *testing.T) {
	for i, w := range []FontWeight{FontWeightW100, FontWeightW200, FontWeightW300, FontWeightW400, FontWeightW500,
		FontWeightW600, FontWeightW700, FontWeightW800, FontWeightW900} {
		eq(t, w.String(), w.Value, (i+1)*100)
	}
}

// testParagraphStyle

func TestParagraphStyleToStringWorks(t *testing.T) {
	ps0 := NewParagraphStyle(ParagraphStyleOptions{TextDirection: ptrTo(TextDirectionLtr), FontSize: ptrTo(14.0)})
	ps1 := NewParagraphStyle(ParagraphStyleOptions{TextDirection: ptrTo(TextDirectionRtl), FontSize: ptrTo(14.0)})
	ps2 := NewParagraphStyle(ParagraphStyleOptions{TextAlign: ptrTo(TextAlignCenter), FontWeight: ptrTo(FontWeightW800),
		FontSize: ptrTo(10.0), Height: ptrTo(100.0)})
	ps3 := NewParagraphStyle(ParagraphStyleOptions{FontWeight: ptrTo(FontWeightW700), FontSize: ptrTo(12.0), Height: ptrTo(123.0)})
	ps4 := NewParagraphStyle(ParagraphStyleOptions{FontWeight: ptrTo(FontWeightW700), FontSize: ptrTo(12.0), Height: ptrTo(KTextHeightNone)})
	ps5 := NewParagraphStyle(ParagraphStyleOptions{Hyphens: ptrTo(HyphensHidden)})
	eq(t, "ps0", ps0.String(), "ParagraphStyle(textAlign: unspecified, textDirection: TextDirection.ltr, fontWeight: unspecified, fontStyle: unspecified, maxLines: unspecified, textHeightBehavior: unspecified, fontFamily: unspecified, fontSize: 14.0, height: unspecified, strutStyle: unspecified, ellipsis: unspecified, locale: unspecified, hyphens: unspecified)")
	eq(t, "ps1", ps1.String(), "ParagraphStyle(textAlign: unspecified, textDirection: TextDirection.rtl, fontWeight: unspecified, fontStyle: unspecified, maxLines: unspecified, textHeightBehavior: unspecified, fontFamily: unspecified, fontSize: 14.0, height: unspecified, strutStyle: unspecified, ellipsis: unspecified, locale: unspecified, hyphens: unspecified)")
	eq(t, "ps2", ps2.String(), "ParagraphStyle(textAlign: TextAlign.center, textDirection: unspecified, fontWeight: FontWeight.w800, fontStyle: unspecified, maxLines: unspecified, textHeightBehavior: unspecified, fontFamily: unspecified, fontSize: 10.0, height: 100.0x, strutStyle: unspecified, ellipsis: unspecified, locale: unspecified, hyphens: unspecified)")
	eq(t, "ps3", ps3.String(), "ParagraphStyle(textAlign: unspecified, textDirection: unspecified, fontWeight: FontWeight.w700, fontStyle: unspecified, maxLines: unspecified, textHeightBehavior: unspecified, fontFamily: unspecified, fontSize: 12.0, height: 123.0x, strutStyle: unspecified, ellipsis: unspecified, locale: unspecified, hyphens: unspecified)")
	eq(t, "ps4", ps4.String(), "ParagraphStyle(textAlign: unspecified, textDirection: unspecified, fontWeight: FontWeight.w700, fontStyle: unspecified, maxLines: unspecified, textHeightBehavior: unspecified, fontFamily: unspecified, fontSize: 12.0, height: unspecified, strutStyle: unspecified, ellipsis: unspecified, locale: unspecified, hyphens: unspecified)")
	eq(t, "ps5", ps5.String(), "ParagraphStyle(textAlign: unspecified, textDirection: unspecified, fontWeight: unspecified, fontStyle: unspecified, maxLines: unspecified, textHeightBehavior: unspecified, fontFamily: unspecified, fontSize: unspecified, height: unspecified, strutStyle: unspecified, ellipsis: unspecified, locale: unspecified, hyphens: Hyphens.hidden)")
}

// testTextStyle

func TestTextStyleToStringWorks(t *testing.T) {
	ts0 := NewTextStyle(TextStyleOptions{FontWeight: ptrTo(FontWeightW700), FontSize: ptrTo(12.0), Height: ptrTo(123.0)})
	ts1 := NewTextStyle(TextStyleOptions{Color: ptrTo(NewColor(0xFF00FF00)), FontWeight: ptrTo(FontWeightW800),
		FontSize: ptrTo(10.0), Height: ptrTo(100.0)})
	ts2 := NewTextStyle(TextStyleOptions{FontFamily: ptrTo("test")})
	ts3 := NewTextStyle(TextStyleOptions{FontFamily: ptrTo("foo"), FontFamilyFallback: []string{"Roboto", "test"}})
	ts4 := NewTextStyle(TextStyleOptions{LeadingDistribution: ptrTo(TextLeadingDistributionEven)})
	ts5 := NewTextStyle(TextStyleOptions{Height: ptrTo(KTextHeightNone)})
	eq(t, "ts0", ts0.String(), "TextStyle(color: unspecified, decoration: unspecified, decorationColor: unspecified, decorationStyle: unspecified, decorationThickness: unspecified, fontWeight: FontWeight.w700, fontStyle: unspecified, textBaseline: unspecified, fontFamily: unspecified, fontFamilyFallback: unspecified, fontSize: 12.0, letterSpacing: unspecified, wordSpacing: unspecified, height: 123.0x, leadingDistribution: unspecified, locale: unspecified, background: unspecified, foreground: unspecified, shadows: unspecified, fontFeatures: unspecified, fontVariations: unspecified)")
	eq(t, "ts1", ts1.String(), "TextStyle(color: "+NewColor(0xFF00FF00).String()+", decoration: unspecified, decorationColor: unspecified, decorationStyle: unspecified, decorationThickness: unspecified, fontWeight: FontWeight.w800, fontStyle: unspecified, textBaseline: unspecified, fontFamily: unspecified, fontFamilyFallback: unspecified, fontSize: 10.0, letterSpacing: unspecified, wordSpacing: unspecified, height: 100.0x, leadingDistribution: unspecified, locale: unspecified, background: unspecified, foreground: unspecified, shadows: unspecified, fontFeatures: unspecified, fontVariations: unspecified)")
	eq(t, "ts2", ts2.String(), "TextStyle(color: unspecified, decoration: unspecified, decorationColor: unspecified, decorationStyle: unspecified, decorationThickness: unspecified, fontWeight: unspecified, fontStyle: unspecified, textBaseline: unspecified, fontFamily: test, fontFamilyFallback: unspecified, fontSize: unspecified, letterSpacing: unspecified, wordSpacing: unspecified, height: unspecified, leadingDistribution: unspecified, locale: unspecified, background: unspecified, foreground: unspecified, shadows: unspecified, fontFeatures: unspecified, fontVariations: unspecified)")
	eq(t, "ts3", ts3.String(), "TextStyle(color: unspecified, decoration: unspecified, decorationColor: unspecified, decorationStyle: unspecified, decorationThickness: unspecified, fontWeight: unspecified, fontStyle: unspecified, textBaseline: unspecified, fontFamily: foo, fontFamilyFallback: [Roboto, test], fontSize: unspecified, letterSpacing: unspecified, wordSpacing: unspecified, height: unspecified, leadingDistribution: unspecified, locale: unspecified, background: unspecified, foreground: unspecified, shadows: unspecified, fontFeatures: unspecified, fontVariations: unspecified)")
	eq(t, "ts4", ts4.String(), "TextStyle(color: unspecified, decoration: unspecified, decorationColor: unspecified, decorationStyle: unspecified, decorationThickness: unspecified, fontWeight: unspecified, fontStyle: unspecified, textBaseline: unspecified, fontFamily: unspecified, fontFamilyFallback: unspecified, fontSize: unspecified, letterSpacing: unspecified, wordSpacing: unspecified, height: unspecified, leadingDistribution: TextLeadingDistribution.even, locale: unspecified, background: unspecified, foreground: unspecified, shadows: unspecified, fontFeatures: unspecified, fontVariations: unspecified)")
	eq(t, "ts5", ts5.String(), "TextStyle(color: unspecified, decoration: unspecified, decorationColor: unspecified, decorationStyle: unspecified, decorationThickness: unspecified, fontWeight: unspecified, fontStyle: unspecified, textBaseline: unspecified, fontFamily: unspecified, fontFamilyFallback: unspecified, fontSize: unspecified, letterSpacing: unspecified, wordSpacing: unspecified, height: kTextHeightNone, leadingDistribution: unspecified, locale: unspecified, background: unspecified, foreground: unspecified, shadows: unspecified, fontFeatures: unspecified, fontVariations: unspecified)")
}

// testTextHeightBehavior

func TestTextHeightBehavior(t *testing.T) {
	behavior0 := NewTextHeightBehavior()
	behavior1 := NewTextHeightBehavior(TextHeightBehaviorOptions{ApplyHeightToFirstAscent: ptrTo(false), ApplyHeightToLastDescent: ptrTo(false)})
	behavior2 := NewTextHeightBehavior(TextHeightBehaviorOptions{ApplyHeightToFirstAscent: ptrTo(false)})
	behavior3 := NewTextHeightBehavior(TextHeightBehaviorOptions{ApplyHeightToLastDescent: ptrTo(false)})
	behavior4 := NewTextHeightBehavior(TextHeightBehaviorOptions{ApplyHeightToLastDescent: ptrTo(false), LeadingDistribution: TextLeadingDistributionEven})
	t.Run("default constructor works", func(t *testing.T) {
		eq(t, "0 first", behavior0.ApplyHeightToFirstAscent, true)
		eq(t, "0 last", behavior0.ApplyHeightToLastDescent, true)
		eq(t, "1 first", behavior1.ApplyHeightToFirstAscent, false)
		eq(t, "1 last", behavior1.ApplyHeightToLastDescent, false)
		eq(t, "2 first", behavior2.ApplyHeightToFirstAscent, false)
		eq(t, "2 last", behavior2.ApplyHeightToLastDescent, true)
		eq(t, "3 first", behavior3.ApplyHeightToFirstAscent, true)
		eq(t, "3 last", behavior3.ApplyHeightToLastDescent, false)
		eq(t, "4 last", behavior4.ApplyHeightToLastDescent, false)
		eq(t, "4 leading", behavior4.LeadingDistribution, TextLeadingDistributionEven)
	})
	t.Run("toString works", func(t *testing.T) {
		eq(t, "0", behavior0.String(), "TextHeightBehavior(applyHeightToFirstAscent: true, applyHeightToLastDescent: true, leadingDistribution: TextLeadingDistribution.proportional)")
		eq(t, "1", behavior1.String(), "TextHeightBehavior(applyHeightToFirstAscent: false, applyHeightToLastDescent: false, leadingDistribution: TextLeadingDistribution.proportional)")
		eq(t, "2", behavior2.String(), "TextHeightBehavior(applyHeightToFirstAscent: false, applyHeightToLastDescent: true, leadingDistribution: TextLeadingDistribution.proportional)")
		eq(t, "3", behavior3.String(), "TextHeightBehavior(applyHeightToFirstAscent: true, applyHeightToLastDescent: false, leadingDistribution: TextLeadingDistribution.proportional)")
		eq(t, "4", behavior4.String(), "TextHeightBehavior(applyHeightToFirstAscent: true, applyHeightToLastDescent: false, leadingDistribution: TextLeadingDistribution.even)")
	})
}

// testTextRange

func TestTextRangeEngine(t *testing.T) {
	t.Run("empty ranges are correct", func(t *testing.T) {
		r := NewTextRange(-1, -1)
		eq(t, "collapsed", r, NewTextRangeCollapsed(-1))
		eq(t, "empty", r, TextRangeEmpty)
	})
	t.Run("isValid works", func(t *testing.T) {
		eq(t, "empty", TextRangeEmpty.IsValid(), false)
		eq(t, "0,0", NewTextRange(0, 0).IsValid(), true)
		eq(t, "0,10", NewTextRange(0, 10).IsValid(), true)
		eq(t, "10,10", NewTextRange(10, 10).IsValid(), true)
		eq(t, "-1,10", NewTextRange(-1, 10).IsValid(), false)
		eq(t, "10,0", NewTextRange(10, 0).IsValid(), true)
		eq(t, "10,-1", NewTextRange(10, -1).IsValid(), false)
	})
	t.Run("isCollapsed works", func(t *testing.T) {
		eq(t, "empty", TextRangeEmpty.IsCollapsed(), true)
		eq(t, "0,0", NewTextRange(0, 0).IsCollapsed(), true)
		eq(t, "0,10", NewTextRange(0, 10).IsCollapsed(), false)
		eq(t, "10,10", NewTextRange(10, 10).IsCollapsed(), true)
		eq(t, "-1,10", NewTextRange(-1, 10).IsCollapsed(), false)
		eq(t, "10,0", NewTextRange(10, 0).IsCollapsed(), false)
		eq(t, "10,-1", NewTextRange(10, -1).IsCollapsed(), false)
	})
	t.Run("isNormalized works", func(t *testing.T) {
		eq(t, "empty", TextRangeEmpty.IsNormalized(), true)
		eq(t, "0,0", NewTextRange(0, 0).IsNormalized(), true)
		eq(t, "0,10", NewTextRange(0, 10).IsNormalized(), true)
		eq(t, "10,10", NewTextRange(10, 10).IsNormalized(), true)
		eq(t, "-1,10", NewTextRange(-1, 10).IsNormalized(), true)
		eq(t, "10,0", NewTextRange(10, 0).IsNormalized(), false)
		eq(t, "10,-1", NewTextRange(10, -1).IsNormalized(), false)
	})
	t.Run("textBefore works", func(t *testing.T) {
		eq(t, "0,0", NewTextRange(0, 0).TextBefore("hello"), "")
		eq(t, "1,1", NewTextRange(1, 1).TextBefore("hello"), "h")
		eq(t, "1,2", NewTextRange(1, 2).TextBefore("hello"), "h")
		eq(t, "5,5", NewTextRange(5, 5).TextBefore("hello"), "hello")
		eq(t, "0,5", NewTextRange(0, 5).TextBefore("hello"), "")
	})
	t.Run("textAfter works", func(t *testing.T) {
		eq(t, "0,0", NewTextRange(0, 0).TextAfter("hello"), "hello")
		eq(t, "1,1", NewTextRange(1, 1).TextAfter("hello"), "ello")
		eq(t, "1,2", NewTextRange(1, 2).TextAfter("hello"), "llo")
		eq(t, "5,5", NewTextRange(5, 5).TextAfter("hello"), "")
		eq(t, "0,5", NewTextRange(0, 5).TextAfter("hello"), "")
	})
	t.Run("textInside works", func(t *testing.T) {
		eq(t, "0,0", NewTextRange(0, 0).TextInside("hello"), "")
		eq(t, "1,1", NewTextRange(1, 1).TextInside("hello"), "")
		eq(t, "1,2", NewTextRange(1, 2).TextInside("hello"), "e")
		eq(t, "5,5", NewTextRange(5, 5).TextInside("hello"), "")
		eq(t, "0,5", NewTextRange(0, 5).TextInside("hello"), "hello")
	})
}

// testGlyphInfo

func TestGlyphInfoConstructor(t *testing.T) {
	testRect := NewRectFromLTWH(1, 2, 3, 4)
	testRange := NewTextRange(5, 6)
	info := NewGlyphInfo(testRect, testRange, TextDirectionLtr)
	eq(t, "bounds", info.GraphemeClusterLayoutBounds, testRect)
	eq(t, "range", info.GraphemeClusterCodeUnitRange, testRange)
	eq(t, "direction", info.WritingDirection, TextDirectionLtr)
}

// testTextDecoration

func TestTextDecorationMaskValue(t *testing.T) {
	eq(t, "none", TextDecorationNone.MaskValue(), 0x0)
	eq(t, "underline", TextDecorationUnderline.MaskValue(), 0x1)
	eq(t, "overline", TextDecorationOverline.MaskValue(), 0x2)
	eq(t, "lineThrough", TextDecorationLineThrough.MaskValue(), 0x4)
	eq(t, "combined", TextDecorationCombine([]TextDecoration{TextDecorationUnderline, TextDecorationLineThrough}).MaskValue(), 0x1|0x4)
	eq(t, "all", TextDecorationCombine([]TextDecoration{TextDecorationUnderline, TextDecorationOverline,
		TextDecorationLineThrough}).MaskValue(), 0x1|0x2|0x4)
}

// testLoadFontFromList

func TestLoadFontFromListSendsPlatformMessageAfterFontIsLoaded(t *testing.T) {
	useTestFonts(t)
	var message string
	ChannelBuffersInstance().SetListener("flutter/system", func(data []byte, callback PlatformMessageResponseCallback) {
		message = string(data)
	})
	defer ChannelBuffersInstance().ClearListener("flutter/system")
	loadFont(t, []byte{}, "fake")
	eq(t, "message", message, `{"type":"fontsChange"}`)
}

// testFontFeatureClass

func TestFontFeatureClass(t *testing.T) {
	for i, c := range [][2]FontFeature{
		{NewFontFeatureAlternative(1), NewFontFeature("aalt")},
		{NewFontFeatureAlternative(5), NewFontFeature("aalt", 5)},
		{NewFontFeatureAlternativeFractions(), NewFontFeature("afrc")},
		{NewFontFeatureContextualAlternates(), NewFontFeature("calt")},
		{NewFontFeatureCaseSensitiveForms(), NewFontFeature("case")},
		{NewFontFeatureCharacterVariant(1), NewFontFeature("cv01")},
		{NewFontFeatureCharacterVariant(18), NewFontFeature("cv18")},
		{NewFontFeatureCharacterVariant(99), NewFontFeature("cv99")},
		{NewFontFeatureDenominator(), NewFontFeature("dnom")},
		{NewFontFeatureFractions(), NewFontFeature("frac")},
		{NewFontFeatureHistoricalForms(), NewFontFeature("hist")},
		{NewFontFeatureHistoricalLigatures(), NewFontFeature("hlig")},
		{NewFontFeatureLiningFigures(), NewFontFeature("lnum")},
		{NewFontFeatureLocaleAware(), NewFontFeature("locl")},
		{NewFontFeatureLocaleAware(), NewFontFeature("locl")},
		{NewFontFeatureLocaleAware(FontFeatureLocaleAwareOptions{Enable: ptrTo(false)}), NewFontFeature("locl", 0)},
		{NewFontFeatureNotationalForms(), NewFontFeature("nalt")},
		{NewFontFeatureNotationalForms(5), NewFontFeature("nalt", 5)},
		{NewFontFeatureNumerators(), NewFontFeature("numr")},
		{NewFontFeatureOldstyleFigures(), NewFontFeature("onum")},
		{NewFontFeatureOrdinalForms(), NewFontFeature("ordn")},
		{NewFontFeatureProportionalFigures(), NewFontFeature("pnum")},
		{NewFontFeatureRandomize(), NewFontFeature("rand")},
		{NewFontFeatureStylisticAlternates(), NewFontFeature("salt")},
		{NewFontFeatureScientificInferiors(), NewFontFeature("sinf")},
		{NewFontFeatureStylisticSet(1), NewFontFeature("ss01")},
		{NewFontFeatureStylisticSet(18), NewFontFeature("ss18")},
		{NewFontFeatureSubscripts(), NewFontFeature("subs")},
		{NewFontFeatureSuperscripts(), NewFontFeature("sups")},
		{NewFontFeatureSwash(), NewFontFeature("swsh")},
		{NewFontFeatureSwash(0), NewFontFeature("swsh", 0)},
		{NewFontFeatureSwash(5), NewFontFeature("swsh", 5)},
		{NewFontFeatureTabularFigures(), NewFontFeature("tnum")},
		{NewFontFeatureSlashedZero(), NewFontFeature("zero")},
		{NewFontFeatureEnable("TEST"), NewFontFeature("TEST")},
		{NewFontFeatureDisable("TEST"), NewFontFeature("TEST", 0)},
	} {
		if c[0] != c[1] {
			t.Errorf("%d: %v != %v", i, c[0], c[1])
		}
	}
	f := NewFontFeature("FEAT", 1000)
	eq(t, "feature", f.Feature, "FEAT")
	eq(t, "value", f.Value, 1000)
	eq(t, "toString", f.String(), "FontFeature('FEAT', 1000)")
}

// testFontVariation

func minIntrinsicWidthOf(t *testing.T, variations []FontVariation, weight *FontWeight) float64 {
	b := NewParagraphBuilder(NewParagraphStyle(ParagraphStyleOptions{FontFamily: ptrTo("RobotoSerif"), FontSize: ptrTo(40.0)}))
	if variations != nil || weight != nil {
		b.PushStyle(NewTextStyle(TextStyleOptions{FontFamily: ptrTo("RobotoSerif"), FontSize: ptrTo(40.0),
			FontVariations: variations, FontWeight: weight}))
	}
	b.AddText("Hello")
	p := b.Build()
	p.Layout(NewParagraphConstraints(math.Inf(1)))
	return p.MinIntrinsicWidth()
}

func TestFontVariation(t *testing.T) {
	useTestFonts(t)
	loadFont(t, readResource(t, "RobotoSlab-VariableFont_wght.ttf"), "RobotoSerif")
	baseWidth := minIntrinsicWidthOf(t, nil, nil)
	wideWidth := minIntrinsicWidthOf(t, []FontVariation{NewFontVariation("wght", 900.0)}, nil)
	if !(wideWidth > baseWidth) {
		t.Errorf("wide %v should be wider than base %v", wideWidth, baseWidth)
	}
}

func TestFontVariationConstructors(t *testing.T) {
	for _, c := range []struct {
		v     FontVariation
		axis  string
		value float64
	}{
		{NewFontVariationWeight(123.0), "wght", 123.0},
		{NewFontVariationWidth(123.0), "wdth", 123.0},
		{NewFontVariationSlant(45.0), "slnt", 45.0},
		{NewFontVariationOpticalSize(67.0), "opsz", 67.0},
		{NewFontVariationItalic(0.8), "ital", 0.8},
	} {
		eq(t, "axis", c.v.Axis, c.axis)
		eq(t, "value", c.v.Value, c.value)
	}
}

func TestFontVariationLerp(t *testing.T) {
	lerp := func(a, b FontVariation, t float64) FontVariation { return *FontVariationLerp(&a, &b, t) }
	eq(t, "weight", lerp(NewFontVariationWeight(100.0), NewFontVariationWeight(300.0), 0.5), NewFontVariationWeight(200.0))
	eq(t, "slant", lerp(NewFontVariationSlant(0.0), NewFontVariationSlant(-80.0), 0.25), NewFontVariationSlant(-20.0))
	eq(t, "width/italic 0.1", lerp(NewFontVariationWidth(90.0), NewFontVariationItalic(0.2), 0.1), NewFontVariationWidth(90.0))
	eq(t, "width/italic 0.9", lerp(NewFontVariationWidth(90.0), NewFontVariationItalic(0.2), 0.9), NewFontVariationItalic(0.2))
}

// The engine compares this paragraph's rendering with a golden image; here
// the weights must change its layout (the variable font's wght axis).
func TestFontWeightIsAppliedToVariableFonts(t *testing.T) {
	useTestFonts(t)
	loadFont(t, readResource(t, "RobotoSlab-VariableFont_wght.ttf"), "RobotoSerif")
	thin := minIntrinsicWidthOf(t, nil, &FontWeightW100)
	normal := minIntrinsicWidthOf(t, nil, &FontWeightNormal)
	black := minIntrinsicWidthOf(t, nil, &FontWeightW900)
	if !(thin < normal && normal < black) {
		t.Errorf("widths by weight: thin %v, normal %v, black %v", thin, normal, black)
	}
}

// testGetWordBoundary

func TestGetWordBoundary(t *testing.T) {
	useTestFonts(t)
	loadFont(t, readResource(t, "RobotoSlab-VariableFont_wght.ttf"), "RobotoSerif")
	p := paragraphOf(NewParagraphStyle(ParagraphStyleOptions{FontFamily: ptrTo("RobotoSerif"), FontSize: ptrTo(40.0)}),
		"Hello team", math.Inf(1))
	eq(t, "upstream", p.GetWordBoundary(NewTextPosition(5, TextAffinityUpstream)), NewTextRange(0, 5))
	eq(t, "downstream", p.GetWordBoundary(NewTextPosition(5)), NewTextRange(5, 6))
}

// Canvas.drawParagraph records the paragraph in the picture (the engine's
// Paragraph::paint into the canvas's display list builder).
func TestTextDrawParagraphRecords(t *testing.T) {
	useTestFonts(t)
	p := paragraphOf(NewParagraphStyle(), "Hello", 800)
	recorder := NewPictureRecorder()
	canvas := NewCanvas(recorder)
	before := recorder.canvas.ops
	canvas.DrawParagraph(p, NewOffset(10, 20))
	if recorder.canvas.ops != before+1 {
		t.Errorf("ops %d -> %d", before, recorder.canvas.ops)
	}
	recorder.EndRecording()
}
