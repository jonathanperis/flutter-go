package difftest

// Go side of the dart:ui text differential test (flutter/test/text_test.dart
// writes the references build/text.json and, under --enable-impeller,
// build/text_impeller.json). Compared exactly.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func textN(v float64) any { return jnum(v) }

func textErr(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if a, ok := r.(dart.AssertionFailure); ok {
				if m := a.AssertionMessage(); m != nil {
					s = "AssertionError: " + fmt.Sprint(m)
				} else {
					s = "AssertionError: null"
				}
				return
			}
			if e, ok := r.(error); ok {
				s = e.Error()
			} else {
				s = fmt.Sprint(r)
			}
		}
	}()
	f()
	return "no error"
}

func textNum(v any) float64 { return v.(float64) }

func textColor(v any) *Color {
	if v == nil {
		return nil
	}
	c := NewColor(int(textNum(v)))
	return &c
}

func textLocale(v any) *Locale {
	if v == nil {
		return nil
	}
	l := v.([]any)
	if len(l) > 1 {
		x := NewLocale(l[0].(string), l[1].(string))
		return &x
	}
	x := NewLocale(l[0].(string))
	return &x
}

func textEnum[T interface{ Name() string }](values []T, name any) *T {
	if name == nil {
		return nil
	}
	for _, v := range values {
		if v.Name() == name.(string) {
			return &v
		}
	}
	panic("no enum value " + name.(string))
}

func textD(v any) *float64 {
	if v == nil {
		return nil
	}
	f := textNum(v)
	return &f
}

func textPaint(v any) *Paint {
	if v == nil {
		return nil
	}
	p := NewPaint()
	p.SetColor(NewColor(int(textNum(v))))
	return p
}

func textStrings(v any) []string {
	if v == nil {
		return nil
	}
	out := []string{}
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func textStyleFrom(s map[string]any) *TextStyle {
	o := TextStyleOptions{
		Color:               textColor(s["color"]),
		DecorationColor:     textColor(s["decorationColor"]),
		DecorationStyle:     textEnum(TextDecorationStyleValues, s["decorationStyle"]),
		DecorationThickness: textD(s["decorationThickness"]),
		FontStyle:           textEnum(FontStyleValues, s["fontStyle"]),
		TextBaseline:        textEnum(TextBaselineValues, s["textBaseline"]),
		FontFamilyFallback:  textStrings(s["fontFamilyFallback"]),
		FontSize:            textD(s["fontSize"]),
		LetterSpacing:       textD(s["letterSpacing"]),
		WordSpacing:         textD(s["wordSpacing"]),
		Height:              textD(s["height"]),
		LeadingDistribution: textEnum(TextLeadingDistributionValues, s["leadingDistribution"]),
		Locale:              textLocale(s["locale"]),
		Background:          textPaint(s["background"]),
		Foreground:          textPaint(s["foreground"]),
	}
	if d, ok := s["decoration"]; ok {
		var ds []TextDecoration
		for _, x := range d.([]any) {
			switch x {
			case "underline":
				ds = append(ds, TextDecorationUnderline)
			case "overline":
				ds = append(ds, TextDecorationOverline)
			case "lineThrough":
				ds = append(ds, TextDecorationLineThrough)
			default:
				ds = append(ds, TextDecorationNone)
			}
		}
		o.Decoration = ptr(TextDecorationCombine(ds))
	}
	if w, ok := s["fontWeight"]; ok {
		o.FontWeight = ptr(NewFontWeight(int(textNum(w))))
	}
	if f, ok := s["fontFamily"]; ok {
		o.FontFamily = ptr(f.(string))
	}
	if sh, ok := s["shadows"]; ok {
		o.Shadows = []Shadow{}
		for _, x := range sh.([]any) {
			l := x.([]any)
			o.Shadows = append(o.Shadows, NewShadow(ShadowOptions{Color: textColor(l[0]),
				Offset: NewOffset(textNum(l[1]), textNum(l[2])), BlurRadius: textNum(l[3])}))
		}
	}
	if ff, ok := s["fontFeatures"]; ok {
		o.FontFeatures = []FontFeature{}
		for _, x := range ff.([]any) {
			l := x.([]any)
			o.FontFeatures = append(o.FontFeatures, NewFontFeature(l[0].(string), int(textNum(l[1]))))
		}
	}
	if fv, ok := s["fontVariations"]; ok {
		o.FontVariations = []FontVariation{}
		for _, x := range fv.([]any) {
			l := x.([]any)
			o.FontVariations = append(o.FontVariations, NewFontVariation(l[0].(string), textNum(l[1])))
		}
	}
	return NewTextStyle(o)
}

func paragraphStyleFrom(p map[string]any) *ParagraphStyle {
	o := ParagraphStyleOptions{
		TextAlign:     textEnum(TextAlignValues, p["textAlign"]),
		TextDirection: textEnum(TextDirectionValues, p["textDirection"]),
		FontSize:      textD(p["fontSize"]),
		Height:        textD(p["height"]),
		FontStyle:     textEnum(FontStyleValues, p["fontStyle"]),
		Locale:        textLocale(p["locale"]),
		Hyphens:       textEnum(HyphensValues, p["hyphens"]),
	}
	if v, ok := p["maxLines"]; ok {
		o.MaxLines = ptr(int(textNum(v)))
	}
	if v, ok := p["fontFamily"]; ok {
		o.FontFamily = ptr(v.(string))
	}
	if v, ok := p["fontWeight"]; ok {
		o.FontWeight = ptr(NewFontWeight(int(textNum(v))))
	}
	if v, ok := p["ellipsis"]; ok {
		o.Ellipsis = ptr(v.(string))
	}
	if v, ok := p["textHeightBehavior"]; ok {
		m := v.(map[string]any)
		ld := TextLeadingDistributionProportional
		if l := textEnum(TextLeadingDistributionValues, m["leading"]); l != nil {
			ld = *l
		}
		o.TextHeightBehavior = ptr(NewTextHeightBehavior(TextHeightBehaviorOptions{
			ApplyHeightToFirstAscent: ptr(m["first"].(bool)),
			ApplyHeightToLastDescent: ptr(m["last"].(bool)),
			LeadingDistribution:      ld,
		}))
	}
	if v, ok := p["strut"]; ok {
		m := v.(map[string]any)
		so := StrutStyleOptions{
			FontSize:            textD(m["fontSize"]),
			Height:              textD(m["height"]),
			LeadingDistribution: textEnum(TextLeadingDistributionValues, m["leadingDistribution"]),
			Leading:             textD(m["leading"]),
		}
		so.FontFamilyFallback = textStrings(m["fontFamilyFallback"])
		if f, ok := m["fontFamily"]; ok {
			so.FontFamily = ptr(f.(string))
		}
		if f, ok := m["forceStrutHeight"]; ok {
			so.ForceStrutHeight = ptr(f.(bool))
		}
		o.StrutStyle = NewStrutStyle(so)
	}
	return NewParagraphStyle(o)
}

func buildTextScene(scene map[string]any) (Paragraph, int) {
	ps := map[string]any{}
	if p, ok := scene["paragraph"]; ok {
		ps = p.(map[string]any)
	}
	b := NewParagraphBuilder(paragraphStyleFrom(ps))
	length := 0
	for _, opv := range scene["ops"].([]any) {
		op := opv.([]any)
		switch op[0] {
		case "push":
			b.PushStyle(textStyleFrom(op[1].(map[string]any)))
		case "pop":
			b.Pop()
		case "text":
			b.AddText(op[1].(string))
			length += len(dart.UTF16CodeUnits(op[1].(string)))
		case "placeholder":
			o := map[string]any{}
			if len(op) > 4 {
				o = op[4].(map[string]any)
			}
			scale := 1.0
			if s := textD(o["scale"]); s != nil {
				scale = *s
			}
			b.AddPlaceholder(textNum(op[1]), textNum(op[2]), *textEnum(PlaceholderAlignmentValues, op[3]),
				AddPlaceholderOptions{Scale: &scale, BaselineOffset: textD(o["baselineOffset"]),
					Baseline: textEnum(TextBaselineValues, o["baseline"])})
			length++
		}
	}
	return b.Build(), length
}

func textBox(b TextBox) []any {
	return []any{textN(b.Left), textN(b.Top), textN(b.Right), textN(b.Bottom), int(b.Direction)}
}

func textBoxes(l []TextBox) []any {
	out := []any{}
	for _, b := range l {
		out = append(out, textBox(b))
	}
	return out
}

func textGlyph(g *GlyphInfo) any {
	if g == nil {
		return nil
	}
	r := g.GraphemeClusterLayoutBounds
	return []any{textN(r.Left), textN(r.Top), textN(r.Right), textN(r.Bottom),
		g.GraphemeClusterCodeUnitRange.Start, g.GraphemeClusterCodeUnitRange.End, int(g.WritingDirection)}
}

func textLine(m *LineMetrics) any {
	if m == nil {
		return nil
	}
	return []any{m.HardBreak, textN(m.Ascent), textN(m.Descent), textN(m.UnscaledAscent), textN(m.Height),
		textN(m.Width), textN(m.Left), textN(m.Baseline), m.LineNumber}
}

func textGrid(extent, step float64) []float64 {
	out := []float64{-5, 0}
	if math.IsInf(extent, 0) || math.IsNaN(extent) {
		extent = 300
	}
	for v := 1.5; v <= extent+5; v += step {
		out = append(out, v)
	}
	return out
}

func dumpParagraph(p Paragraph, length int) map[string]any {
	out := map[string]any{}
	out["metrics"] = []any{textN(p.Width()), textN(p.Height()), textN(p.LongestLine()), textN(p.MinIntrinsicWidth()),
		textN(p.MaxIntrinsicWidth()), textN(p.AlphabeticBaseline()), textN(p.IdeographicBaseline()),
		p.DidExceedMaxLines(), p.NumberOfLines()}
	lines := []any{}
	for _, m := range p.ComputeLineMetrics() {
		lines = append(lines, textLine(&m))
	}
	out["lines"] = lines
	lineAt := []any{}
	for i := -1; i <= p.NumberOfLines(); i++ {
		lineAt = append(lineAt, textLine(p.GetLineMetricsAt(i)))
	}
	out["lineAt"] = lineAt
	lineNumberAt := []any{}
	for i := -1; i <= length+1; i++ {
		if n := p.GetLineNumberAt(i); n != nil {
			lineNumberAt = append(lineNumberAt, *n)
		} else {
			lineNumberAt = append(lineNumberAt, nil)
		}
	}
	out["lineNumberAt"] = lineNumberAt
	boxes1 := []any{}
	for i := 0; i < length; i++ {
		boxes1 = append(boxes1, textBoxes(p.GetBoxesForRange(i, i+1)))
	}
	out["boxes1"] = boxes1
	ranges := [][2]int{{0, length}, {length / 3, length - length/3}, {length - 2, length + 3}, {3, 1}}
	styled := map[string]any{}
	for _, r := range ranges {
		for _, hs := range BoxHeightStyleValues {
			for _, ws := range BoxWidthStyleValues {
				key := strconv.Itoa(r[0]) + "," + strconv.Itoa(r[1]) + "," + strconv.Itoa(int(hs)) + "," + strconv.Itoa(int(ws))
				styled[key] = textBoxes(p.GetBoxesForRange(r[0], r[1], GetBoxesForRangeOptions{BoxHeightStyle: hs, BoxWidthStyle: ws}))
			}
		}
	}
	out["boxes"] = styled
	out["placeholders"] = textBoxes(p.GetBoxesForPlaceholders())
	glyphAt := []any{}
	for i := -1; i <= length; i++ {
		glyphAt = append(glyphAt, textGlyph(p.GetGlyphInfoAt(i)))
	}
	out["glyphAt"] = glyphAt
	positions, closest := []any{}, []any{}
	for _, y := range textGrid(p.Height(), 7) {
		for _, x := range textGrid(p.Width(), 6.25) {
			pos := p.GetPositionForOffset(NewOffset(x, y))
			positions = append(positions, []any{pos.Offset, int(pos.Affinity)})
			closest = append(closest, textGlyph(p.GetClosestGlyphInfoForOffset(NewOffset(x, y))))
		}
	}
	out["positions"] = positions
	out["closest"] = closest
	words, lineBounds := []any{}, []any{}
	for i := 0; i <= length; i++ {
		for _, a := range TextAffinityValues {
			w := p.GetWordBoundary(NewTextPosition(i, a))
			words = append(words, []any{w.Start, w.End})
			l := p.GetLineBoundary(NewTextPosition(i, a))
			lineBounds = append(lineBounds, []any{l.Start, l.End})
		}
	}
	out["words"] = words
	out["lineBounds"] = lineBounds
	out["string"] = p.String()
	return out
}

var textFontsLoaded bool

// textLateFont is the lateFont case, computed once per process.
var (
	textLateLoaded bool
	textLateFont   []any
)

// loadTextFonts loads the difftest's fonts once (as the Dart side does at
// the start of its run); their fontsChange messages go to a listener, as
// flutter_test's binding takes them.
func loadTextFonts(t *testing.T) {
	if textFontsLoaded {
		return
	}
	textFontsLoaded = true
	ChannelBuffersInstance().SetListener("flutter/system", func([]byte, PlatformMessageResponseCallback) {})
	defer ChannelBuffersInstance().ClearListener("flutter/system")
	for _, f := range []struct{ file, family string }{
		{"Roboto-Regular.ttf", "Roboto"},
		{"NotoNaskhArabic-Regular.ttf", "NotoNaskhArabic"},
		{"RobotoSlab-VariableFont_wght.ttf", "RobotoSlab"},
	} {
		data, err := os.ReadFile("../../engine/src/flutter/txt/third_party/fonts/" + f.file)
		if err != nil {
			t.Fatal(err)
		}
		LoadFontFromList(data, LoadFontFromListOptions{FontFamily: ptr(f.family)})
	}
	async.Current().FlushMicrotasks()
}

func textStringsOf[T fmt.Stringer](l ...T) []any {
	out := []any{}
	for _, x := range l {
		out = append(out, x.String())
	}
	return out
}

func boolMatrix[T any](l []T, eq func(a, b T) bool) []any {
	out := []any{}
	for i := range l {
		row := []any{}
		for j := range l {
			row = append(row, eq(l[i], l[j]))
		}
		out = append(out, row)
	}
	return out
}

func textNullable[T fmt.Stringer](v *T) string {
	if v == nil {
		return "null"
	}
	return (*v).String()
}

func textAPICases() map[string]any {
	out := map[string]any{}
	out["fontWeight.values"] = textStringsOf(FontWeightValues...)
	out["fontWeight.odd"] = textStringsOf(NewFontWeight(1), NewFontWeight(450), NewFontWeight(1000))
	out["fontWeight.index"] = []any{NewFontWeight(1).Index(), NewFontWeight(450).Index(), NewFontWeight(1000).Index()}
	lerps := []any{}
	for _, t := range []float64{-1, 0, 0.25, 0.5, 0.55, 1, 2, 1.0 / 3, 0.123, 0.6789} {
		lerps = append(lerps, textNullable(FontWeightLerp(&FontWeightW100, &FontWeightW900, t))+"|"+
			textNullable(FontWeightLerp(nil, &FontWeightW100, t))+"|"+textNullable(FontWeightLerp(&FontWeightW300, nil, t))+"|"+
			textNullable(FontWeightLerp(nil, nil, t)))
	}
	out["fontWeight.lerp"] = lerps
	out["fontWeight.errors"] = []any{textErr(func() { NewFontWeight(0) }), textErr(func() { NewFontWeight(1001) })}
	out["fontWeight.equal"] = []any{NewFontWeight(400) == FontWeightNormal, NewFontWeight(700) == FontWeightBold, NewFontWeight(701) == FontWeightBold}
	out["fontFeature.ctors"] = textStringsOf(
		NewFontFeature("liga"), NewFontFeature("liga", 0), NewFontFeatureEnable("smcp"), NewFontFeatureDisable("kern"),
		NewFontFeatureAlternative(3), NewFontFeatureAlternativeFractions(), NewFontFeatureContextualAlternates(),
		NewFontFeatureCaseSensitiveForms(), NewFontFeatureCharacterVariant(7), NewFontFeatureCharacterVariant(42),
		NewFontFeatureDenominator(), NewFontFeatureFractions(), NewFontFeatureHistoricalForms(),
		NewFontFeatureHistoricalLigatures(), NewFontFeatureLiningFigures(), NewFontFeatureLocaleAware(),
		NewFontFeatureLocaleAware(FontFeatureLocaleAwareOptions{Enable: ptr(false)}), NewFontFeatureNotationalForms(),
		NewFontFeatureNotationalForms(4), NewFontFeatureNumerators(), NewFontFeatureOldstyleFigures(),
		NewFontFeatureOrdinalForms(), NewFontFeatureProportionalFigures(), NewFontFeatureRandomize(),
		NewFontFeatureStylisticAlternates(), NewFontFeatureScientificInferiors(), NewFontFeatureStylisticSet(3),
		NewFontFeatureStylisticSet(20), NewFontFeatureSubscripts(), NewFontFeatureSuperscripts(), NewFontFeatureSwash(),
		NewFontFeatureSwash(2), NewFontFeatureTabularFigures(), NewFontFeatureSlashedZero())
	out["fontFeature.errors"] = []any{
		textErr(func() { NewFontFeature("abc") }), textErr(func() { NewFontFeature("abcde") }),
		textErr(func() { NewFontFeature("liga", -1) }), textErr(func() { NewFontFeatureCharacterVariant(0) }),
		textErr(func() { NewFontFeatureCharacterVariant(100) }), textErr(func() { NewFontFeatureStylisticSet(21) }),
	}
	out["fontFeature.equal"] = []any{NewFontFeature("liga") == NewFontFeatureEnable("liga"), NewFontFeature("liga") == NewFontFeature("liga", 2)}
	out["fontVariation.ctors"] = textStringsOf(NewFontVariation("wght", 500), NewFontVariation("wdth", -1.5),
		NewFontVariationItalic(0.5), NewFontVariationOpticalSize(12), NewFontVariationSlant(-10),
		NewFontVariationWidth(75), NewFontVariationWeight(1000))
	out["fontVariation.errors"] = []any{
		textErr(func() { NewFontVariation("wg", 1) }), textErr(func() { NewFontVariation("wght", 32768) }),
		textErr(func() { NewFontVariation("wght", -32769) }), textErr(func() { NewFontVariationItalic(2) }),
		textErr(func() { NewFontVariationSlant(90) }), textErr(func() { NewFontVariationWeight(0) }),
	}
	vl := []any{}
	for _, t := range []float64{-1, 0, 0.3, 0.5, 1, 1e6} {
		a, b, c := NewFontVariation("wght", 100), NewFontVariation("wght", 900), NewFontVariation("wdth", 900)
		e := NewFontVariation("wdth", 9)
		vl = append(vl, textNullable(FontVariationLerp(&a, &b, t))+"|"+textNullable(FontVariationLerp(&a, &c, t))+"|"+
			textNullable(FontVariationLerp(nil, &e, t))+"|"+textNullable(FontVariationLerp(nil, nil, t)))
	}
	out["fontVariation.lerp"] = vl
	ul := TextDecorationCombine([]TextDecoration{TextDecorationUnderline, TextDecorationOverline})
	out["decoration"] = []any{
		TextDecorationNone.String(), TextDecorationUnderline.String(), TextDecorationOverline.String(),
		TextDecorationLineThrough.String(),
		TextDecorationCombine([]TextDecoration{TextDecorationUnderline, TextDecorationLineThrough}).String(),
		TextDecorationCombine([]TextDecoration{TextDecorationUnderline, TextDecorationOverline, TextDecorationLineThrough}).String(),
		TextDecorationCombine([]TextDecoration{}).String(),
		ul.Contains(TextDecorationOverline), TextDecorationUnderline.Contains(TextDecorationOverline),
		TextDecorationUnderline.Contains(TextDecorationNone), ul.MaskValue(),
		TextDecorationCombine([]TextDecoration{TextDecorationUnderline}) == TextDecorationUnderline,
	}
	out["textHeightBehavior"] = []any{
		NewTextHeightBehavior().String(),
		NewTextHeightBehavior(TextHeightBehaviorOptions{ApplyHeightToFirstAscent: ptr(false), LeadingDistribution: TextLeadingDistributionEven}).String(),
		NewTextHeightBehavior() == NewTextHeightBehavior(TextHeightBehaviorOptions{ApplyHeightToLastDescent: ptr(true)}),
		NewTextHeightBehavior() == NewTextHeightBehavior(TextHeightBehaviorOptions{ApplyHeightToLastDescent: ptr(false)}),
	}
	enums := []any{}
	enums = append(enums, textStringsOf(FontStyleValues...)...)
	enums = append(enums, textStringsOf(TextAlignValues...)...)
	enums = append(enums, textStringsOf(HyphensValues...)...)
	enums = append(enums, textStringsOf(TextBaselineValues...)...)
	enums = append(enums, textStringsOf(TextDecorationStyleValues...)...)
	enums = append(enums, textStringsOf(TextLeadingDistributionValues...)...)
	enums = append(enums, textStringsOf(BoxHeightStyleValues...)...)
	enums = append(enums, textStringsOf(BoxWidthStyleValues...)...)
	enums = append(enums, textStringsOf(PlaceholderAlignmentValues...)...)
	out["enums"] = enums
	g := NewGlyphInfo(NewRectFromLTRB(1, 2, 3, 4), NewTextRange(5, 6), TextDirectionRtl)
	out["valueTypes"] = []any{
		NewTextBoxFromLTRBD(1.25, 2.05, 3.35, 4.45, TextDirectionRtl).String(),
		NewTextBoxFromLTRBD(1, 2, 3, 4, TextDirectionRtl).Start(),
		NewTextBoxFromLTRBD(1, 2, 3, 4, TextDirectionRtl).End(),
		NewTextBoxFromLTRBD(1, 2, 3, 4, TextDirectionLtr).Start(),
		NewTextBoxFromLTRBD(1, 2, 3, 4, TextDirectionLtr).ToRect().String(),
		NewParagraphConstraints(12.5).String(),
		NewParagraphConstraints(math.Inf(1)).String(),
		LineMetrics{HardBreak: true, Ascent: 1, Descent: 2.5, UnscaledAscent: 3, Height: 4, Width: 5, Left: 6, Baseline: 7, LineNumber: 8}.String(),
		g.String(),
		g == NewGlyphInfo(NewRectFromLTRB(1, 2, 3, 4), NewTextRange(5, 6), TextDirectionRtl),
	}
	bg := NewPaint()
	bg.SetColor(NewColor(0xFF00FF00))
	fg := NewPaint()
	fg.SetColor(NewColor(0xFF0000FF))
	bg2 := NewPaint()
	bg2.SetColor(NewColor(0xFF00FF00))
	styles := []*TextStyle{
		NewTextStyle(),
		NewTextStyle(TextStyleOptions{Color: ptr(NewColor(0xFF123456)), FontSize: ptr(12.5)}),
		NewTextStyle(TextStyleOptions{
			Decoration:          ptr(TextDecorationCombine([]TextDecoration{TextDecorationUnderline, TextDecorationOverline})),
			DecorationColor:     ptr(NewColor(0x80FF0000)),
			DecorationStyle:     ptr(TextDecorationStyleDashed),
			DecorationThickness: ptr(2.0),
			FontWeight:          ptr(NewFontWeight(450)),
			FontStyle:           ptr(FontStyleItalic),
			TextBaseline:        ptr(TextBaselineIdeographic),
			FontFamily:          ptr("Roboto"),
			FontFamilyFallback:  []string{"Noto", "Emoji"},
			FontSize:            ptr(10.0),
			LetterSpacing:       ptr(1.5),
			WordSpacing:         ptr(-2.0),
			Height:              ptr(1.25),
			LeadingDistribution: ptr(TextLeadingDistributionEven),
			Locale:              ptr(NewLocale("fr", "CA")),
			Background:          bg,
			Foreground:          fg,
			Shadows:             []Shadow{NewShadow(ShadowOptions{Color: ptr(NewColor(0xFF00FF00)), Offset: NewOffset(1, 2), BlurRadius: 3})},
			FontFeatures:        []FontFeature{NewFontFeatureTabularFigures(), NewFontFeature("liga", 0)},
			FontVariations:      []FontVariation{NewFontVariation("wght", 300.5)},
		}),
		NewTextStyle(TextStyleOptions{Height: ptr(KTextHeightNone)}),
		NewTextStyle(TextStyleOptions{FontFamily: ptr("")}),
		NewTextStyle(TextStyleOptions{FontFamilyFallback: []string{}}),
		NewTextStyle(TextStyleOptions{FontFamilyFallback: []string{"Only"}}),
		NewTextStyle(TextStyleOptions{Shadows: []Shadow{}, FontFeatures: []FontFeature{}, FontVariations: []FontVariation{}}),
		NewTextStyle(TextStyleOptions{Background: bg}),
		NewTextStyle(TextStyleOptions{Background: bg}),
		NewTextStyle(TextStyleOptions{Background: bg2}),
		NewTextStyle(TextStyleOptions{FontSize: ptr(12.5), Color: ptr(NewColor(0xFF123456))}),
	}
	ss := []any{}
	for _, s := range styles {
		ss = append(ss, s.String())
	}
	out["textStyle.string"] = ss
	out["textStyle.equal"] = boolMatrix(styles, (*TextStyle).Equal)
	out["textStyle.errors"] = []any{textErr(func() {
		NewTextStyle(TextStyleOptions{Color: ptr(NewColor(0xFF000000)), Foreground: NewPaint()})
	})}
	pushed := NewTextStyle(TextStyleOptions{FontSize: ptr(10.0)})
	twin := NewTextStyle(TextStyleOptions{FontSize: ptr(10.0)})
	evenBuilder := NewParagraphBuilder(NewParagraphStyle(ParagraphStyleOptions{
		TextHeightBehavior: ptr(NewTextHeightBehavior(TextHeightBehaviorOptions{LeadingDistribution: TextLeadingDistributionEven}))}))
	out["textStyle.pushMutates.before"] = pushed.Equal(twin)
	evenBuilder.PushStyle(pushed)
	out["textStyle.pushMutates.after"] = []any{pushed.Equal(twin), pushed.String()}
	evenBuilder.Build().Dispose()
	strut := NewStrutStyle(StrutStyleOptions{FontSize: ptr(10.0), Height: ptr(1.5)})
	pstyles := []*ParagraphStyle{
		NewParagraphStyle(),
		NewParagraphStyle(ParagraphStyleOptions{
			TextAlign:     ptr(TextAlignJustify),
			TextDirection: ptr(TextDirectionRtl),
			MaxLines:      ptr(3),
			FontFamily:    ptr("Roboto"),
			FontSize:      ptr(13.0),
			Height:        ptr(1.5),
			TextHeightBehavior: ptr(NewTextHeightBehavior(TextHeightBehaviorOptions{
				ApplyHeightToLastDescent: ptr(false), LeadingDistribution: TextLeadingDistributionEven})),
			FontWeight: ptr(FontWeightW600),
			FontStyle:  ptr(FontStyleItalic),
			StrutStyle: strut,
			Ellipsis:   ptr("…"),
			Locale:     ptr(NewLocale("en", "US")),
			Hyphens:    ptr(HyphensHidden),
		}),
		NewParagraphStyle(ParagraphStyleOptions{Height: ptr(KTextHeightNone)}),
		NewParagraphStyle(ParagraphStyleOptions{Height: ptr(0.5)}),
		NewParagraphStyle(ParagraphStyleOptions{MaxLines: ptr(0), Ellipsis: ptr("")}),
		NewParagraphStyle(ParagraphStyleOptions{StrutStyle: NewStrutStyle(StrutStyleOptions{FontSize: ptr(10.0), Height: ptr(1.5)})}),
		NewParagraphStyle(ParagraphStyleOptions{StrutStyle: strut}),
		NewParagraphStyle(ParagraphStyleOptions{StrutStyle: NewStrutStyle()}),
		NewParagraphStyle(ParagraphStyleOptions{TextHeightBehavior: ptr(NewTextHeightBehavior())}),
	}
	ps := []any{}
	for _, s := range pstyles {
		ps = append(ps, s.String())
	}
	out["paragraphStyle.string"] = ps
	out["paragraphStyle.equal"] = boolMatrix(pstyles, (*ParagraphStyle).Equal)
	struts := []*StrutStyle{
		NewStrutStyle(),
		NewStrutStyle(StrutStyleOptions{Height: ptr(KTextHeightNone)}),
		NewStrutStyle(StrutStyleOptions{ForceStrutHeight: ptr(false)}),
		NewStrutStyle(StrutStyleOptions{FontFamily: ptr("A")}),
		NewStrutStyle(StrutStyleOptions{FontFamilyFallback: []string{"A"}}),
		NewStrutStyle(StrutStyleOptions{FontFamilyFallback: []string{}}),
		NewStrutStyle(StrutStyleOptions{FontWeight: ptr(FontWeightW300), FontStyle: ptr(FontStyleItalic), FontSize: ptr(12.0), Height: ptr(2.0), Leading: ptr(0.5)}),
		NewStrutStyle(StrutStyleOptions{FontWeight: ptr(FontWeightW300), FontStyle: ptr(FontStyleItalic), FontSize: ptr(12.0), Height: ptr(2.0), Leading: ptr(0.5)}),
		NewStrutStyle(StrutStyleOptions{LeadingDistribution: ptr(TextLeadingDistributionEven)}),
		NewStrutStyle(StrutStyleOptions{FontSize: ptr(12.000001)}),
		NewStrutStyle(StrutStyleOptions{FontSize: ptr(12.0000001)}),
	}
	out["strutStyle.equal"] = boolMatrix(struts, (*StrutStyle).Equal)
	out["strutStyle.string"] = struts[0].String()
	usedStrut := NewStrutStyle(StrutStyleOptions{FontSize: ptr(10.0)})
	twinStrut := NewStrutStyle(StrutStyleOptions{FontSize: ptr(10.0)})
	NewParagraphBuilder(NewParagraphStyle(ParagraphStyleOptions{StrutStyle: usedStrut,
		TextHeightBehavior: ptr(NewTextHeightBehavior(TextHeightBehaviorOptions{LeadingDistribution: TextLeadingDistributionEven}))})).Build().Dispose()
	out["strutStyle.buildMutates"] = usedStrut.Equal(twinStrut)
	builder := NewParagraphBuilder(NewParagraphStyle())
	out["builder.string"] = builder.String()
	builder.AddPlaceholder(10, 10, PlaceholderAlignmentMiddle, AddPlaceholderOptions{Scale: ptr(2.0)})
	builder.AddPlaceholder(10, 10, PlaceholderAlignmentTop)
	scales := []any{}
	for _, s := range builder.PlaceholderScales() {
		scales = append(scales, s)
	}
	out["builder.placeholders"] = []any{builder.PlaceholderCount(), scales}
	out["builder.errors"] = []any{
		textErr(func() { builder.AddText("bad \xed\xa0\x80 surrogate") }),
		textErr(func() { builder.AddText("bad \xed\xb0\x80") }),
		textErr(func() { builder.AddText("trailing \xed\xa0\xbd") }),
		textErr(func() { builder.AddPlaceholder(1, 1, PlaceholderAlignmentBaseline) }),
		textErr(func() { builder.AddPlaceholder(1, 1, PlaceholderAlignmentAboveBaseline) }),
		textErr(func() { builder.AddPlaceholder(1, 1, PlaceholderAlignmentBelowBaseline) }),
		textErr(func() { builder.AddText("ok 👍") }),
	}
	paragraph := builder.Build()
	strs := []any{paragraph.String()}
	paragraph.Layout(NewParagraphConstraints(100))
	strs = append(strs, paragraph.String())
	out["paragraph.debugDisposed"] = paragraph.DebugDisposed()
	paragraph.Dispose()
	strs = append(strs, paragraph.String())
	out["paragraph.string"] = strs
	out["paragraph.disposed"] = []any{paragraph.DebugDisposed(), textErr(paragraph.Dispose)}
	dirty := NewParagraphBuilder(NewParagraphStyle()).Build()
	dirty.Dispose()
	out["paragraph.disposedDirty"] = dirty.String()
	built := NewParagraphBuilder(NewParagraphStyle())
	built.Build()
	afterBuildStyle := NewTextStyle(TextStyleOptions{FontSize: ptr(3.0)})
	out["builder.afterBuild"] = []any{
		textErr(func() { built.PushStyle(afterBuildStyle) }),
		afterBuildStyle.String(),
		textErr(built.Pop),
		textErr(func() { built.AddText("x") }),
		textErr(func() { built.AddText("") }),
		textErr(func() { built.AddPlaceholder(1, 1, PlaceholderAlignmentTop) }),
		built.PlaceholderCount(),
		textErr(func() { built.Build() }),
	}
	gone := NewParagraphBuilder(NewParagraphStyle()).Build()
	gone.Layout(NewParagraphConstraints(10))
	gone.Dispose()
	out["paragraph.afterDispose"] = []any{
		textErr(func() { gone.Width() }),
		textErr(func() { gone.DidExceedMaxLines() }),
		textErr(func() { gone.Layout(NewParagraphConstraints(10)) }),
		textErr(func() { gone.GetBoxesForRange(0, 1) }),
		textErr(func() { gone.GetBoxesForPlaceholders() }),
		textErr(func() { gone.GetPositionForOffset(OffsetZero) }),
		textErr(func() { gone.GetGlyphInfoAt(0) }),
		textErr(func() { gone.GetClosestGlyphInfoForOffset(OffsetZero) }),
		textErr(func() { gone.GetWordBoundary(NewTextPosition(0)) }),
		textErr(func() { gone.GetLineBoundary(NewTextPosition(0)) }),
		textErr(func() { gone.ComputeLineMetrics() }),
		textErr(func() { gone.GetLineMetricsAt(0) }),
		textErr(func() { gone.NumberOfLines() }),
		textErr(func() { gone.GetLineNumberAt(0) }),
	}
	recorder := NewPictureRecorder()
	canvas := NewCanvas(recorder)
	unlaid := NewParagraphBuilder(NewParagraphStyle()).Build()
	out["drawParagraph.errors"] = []any{
		textErr(func() { canvas.DrawParagraph(unlaid, OffsetZero) }),
		textErr(func() {
			unlaid.Layout(NewParagraphConstraints(10))
			canvas.DrawParagraph(unlaid, NewOffset(math.NaN(), 0))
		}),
		textErr(func() { canvas.DrawParagraph(unlaid, NewOffset(1, 2)) }),
		textErr(func() {
			unlaid.Dispose()
			canvas.DrawParagraph(unlaid, OffsetZero)
		}),
	}
	recorder.EndRecording()
	return out
}

// textCases builds every case of the reference.
func textCases(t *testing.T) map[string]any {
	EngineUseTestFonts()
	loadTextFonts(t)
	data, err := os.ReadFile("text_scenes.json")
	if err != nil {
		t.Fatal(err)
	}
	var scenes []map[string]any
	if err := json.Unmarshal(data, &scenes); err != nil {
		t.Fatal(err)
	}
	out := map[string]any{}
	for _, scene := range scenes {
		p, length := buildTextScene(scene)
		perWidth := []any{}
		for _, w := range scene["widths"].([]any) {
			width := math.Inf(1)
			if w != "Infinity" {
				width = textNum(w)
			}
			p.Layout(NewParagraphConstraints(width))
			perWidth = append(perWidth, dumpParagraph(p, length))
		}
		p.Dispose()
		out["scene."+scene["name"].(string)] = perWidth
	}
	lateWidth := func() float64 {
		b := NewParagraphBuilder(NewParagraphStyle())
		b.PushStyle(NewTextStyle(TextStyleOptions{FontFamily: ptr("LateFamily"), FontSize: ptr(20.0)}))
		b.AddText("late font")
		p := b.Build()
		p.Layout(NewParagraphConstraints(1000))
		w := p.MaxIntrinsicWidth()
		p.Dispose()
		return w
	}
	if !textLateLoaded {
		before := lateWidth()
		data, err := os.ReadFile("../../engine/src/flutter/txt/third_party/fonts/Roboto-Regular.ttf")
		if err != nil {
			t.Fatal(err)
		}
		ChannelBuffersInstance().SetListener("flutter/system", func([]byte, PlatformMessageResponseCallback) {})
		LoadFontFromList(data, LoadFontFromListOptions{FontFamily: ptr("LateFamily")})
		async.Current().FlushMicrotasks()
		ChannelBuffersInstance().ClearListener("flutter/system")
		textLateFont = []any{before, lateWidth()}
		textLateLoaded = true
	}
	out["lateFont"] = textLateFont
	for k, v := range textAPICases() {
		out[k] = v
	}
	return out
}

func TestTextMatchesFlutter(t *testing.T) {
	var refs []string
	for _, f := range []string{"flutter/build/text.json", "flutter/build/text_impeller.json"} {
		if _, err := os.Stat(f); err == nil {
			refs = append(refs, f)
		}
	}
	if len(refs) == 0 {
		t.Skip("no reference; run scripts/difftest.sh")
	}
	got := toJSONShape(t, textCases(t)).(map[string]any)
	for _, f := range refs {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var ref map[string]any
		if err := json.Unmarshal(data, &ref); err != nil {
			t.Fatal(err)
		}
		if len(ref) != len(got) {
			t.Errorf("%s: reference has %d cases, Go has %d", f, len(ref), len(got))
		}
		names := make([]string, 0, len(ref))
		for k := range ref {
			names = append(names, k)
		}
		sort.Strings(names)
		failures := 0
		for _, name := range names {
			g, ok := got[name]
			if !ok {
				t.Errorf("%s: %s missing in Go", f, name)
				continue
			}
			if !reflect.DeepEqual(g, ref[name]) {
				failures++
				if failures <= 20 {
					t.Errorf("%s: %s differs: %s", f, name, textFirstDiff("", ref[name], g))
				}
			}
		}
		if failures > 20 {
			t.Errorf("%s: %d cases differ", f, failures)
		}
	}
}

// firstDiff describes the first difference between two JSON values.
func textFirstDiff(path string, want, got any) string {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s: Flutter %v, Go %v", path, want, got)
		}
		keys := make([]string, 0, len(w))
		for k := range w {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !reflect.DeepEqual(w[k], g[k]) {
				return textFirstDiff(path+"."+k, w[k], g[k])
			}
		}
		return fmt.Sprintf("%s: Go has extra keys", path)
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return fmt.Sprintf("%s: Flutter %v, Go %v", path, want, got)
		}
		for i := range w {
			if !reflect.DeepEqual(w[i], g[i]) {
				return textFirstDiff(path+"["+strconv.Itoa(i)+"]", w[i], g[i])
			}
		}
	}
	return fmt.Sprintf("%s: Flutter %v, Go %v", path, want, got)
}
