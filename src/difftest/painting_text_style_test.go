package difftest

// Go side of the painting text-style differential test
// (flutter/test/painting_text_style_test.dart writes
// build/painting_text_style.json): TextStyle, StrutStyle,
// lerpFontVariations and span trees, every line compared exactly.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// tsErr is the Dart test's err(), whose AssertionError message is itself
// guarded (a FlutterError's message renders the styles, which can
// assert).
func tstErr(f func() any) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if a, ok := r.(dart.AssertionFailure); ok {
				s = func() (m string) {
					defer func() {
						if r := recover(); r != nil {
							if _, ok := r.(dart.AssertionFailure); ok {
								m = "AssertionError: (message asserts)"
								return
							}
							panic(r)
						}
					}()
					if msg := a.AssertionMessage(); msg != nil {
						return "AssertionError: " + fmt.Sprint(msg)
					}
					return "AssertionError: null"
				}()
				return
			}
			if e, ok := r.(error); ok {
				s = "error: " + e.Error()
			} else {
				s = "error: " + fmt.Sprint(r)
			}
		}
	}()
	v := f()
	switch x := v.(type) {
	case *TextStyle:
		if x == nil {
			return "null"
		}
		return x.String()
	case *StrutStyle:
		if x == nil {
			return "null"
		}
		return x.String()
	}
	return tpStr(v)
}

func tsLayout(ps *ui.ParagraphStyle, ts *ui.TextStyle) string {
	b := ui.NewParagraphBuilder(ps)
	if ts != nil {
		b.PushStyle(ts)
	}
	b.AddText("Ag xy\nz")
	p := b.Build()
	p.Layout(ui.NewParagraphConstraints(200))
	d := dart.DoubleToString
	return d(p.Width()) + " " + d(p.Height()) + " " + d(p.LongestLine()) + " " + d(p.AlphabeticBaseline()) + " " +
		tpList(p.ComputeLineMetrics())
}

func tsProps(fill func(*foundation.DiagnosticPropertiesBuilder)) string {
	b := foundation.NewDiagnosticPropertiesBuilder()
	fill(b)
	var parts []string
	for _, n := range b.Properties {
		parts = append(parts, n.Name()+"="+n.ToDescription()+"@"+n.Level().String())
	}
	return strings.Join(parts, "; ")
}

func tsSpanName(s InlineSpan) string {
	switch x := s.(type) {
	case nil:
		return "null"
	case *TextSpan:
		return "TextSpan(" + nullableStr(x.Text) + ")"
	}
	return "TestPlaceholder"
}

func nullableStr(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func tsInfo(i InlineSpanSemanticsInformation) string {
	return fmt.Sprintf("{%s|%s|%s|%t|%t|%t|%s}", i.Text, nullableStr(i.SemanticsLabel), nullableStr(i.SemanticsIdentifier),
		i.Recognizer != nil, i.IsPlaceholder, i.RequiresOwnNode, tpList(i.StringAttributes))
}

func tsInfos(l []InlineSpanSemanticsInformation) string {
	parts := make([]string, len(l))
	for i, x := range l {
		parts[i] = tsInfo(x)
	}
	return strings.Join(parts, " ")
}

// tsBarePlaceholder is the Dart test's _BarePlaceholder: a PlaceholderSpan
// that keeps PlaceholderSpan's asserting debugAssertIsValid.
type tsBarePlaceholder struct{ PlaceholderSpanBase }

func (p *tsBarePlaceholder) Build(ui.ParagraphBuilder, ...InlineSpanBuildOptions) {}
func (p *tsBarePlaceholder) VisitChildren(InlineSpanVisitor) bool                 { return true }
func (p *tsBarePlaceholder) VisitDirectChildren(InlineSpanVisitor) bool           { return true }
func (p *tsBarePlaceholder) GetSpanForPositionVisitor(ui.TextPosition, *Accumulator) InlineSpan {
	return nil
}
func (p *tsBarePlaceholder) CodeUnitAtVisitor(int, *Accumulator) *int { return nil }
func (p *tsBarePlaceholder) CompareTo(InlineSpan) RenderComparison    { return RenderComparisonLayout }

func paintingTextStyleLines(t *testing.T) []string {
	EngineUseTestFontsAndRoboto(t)
	var out []string
	add := func(s string) { out = append(out, s) }
	f := func(v float64) *float64 { return &v }
	s := func(v string) *string { return &v }
	c := func(v int) *ui.Color { x := ui.NewColor(v); return &x }
	b := func(v bool) *bool { return &v }
	loc := func(l string, cc ...string) *ui.Locale { x := ui.NewLocale(l, cc...); return &x }
	fw := func(w ui.FontWeight) *ui.FontWeight { return &w }
	fsItalic, fsNormal := ui.FontStyleItalic, ui.FontStyleNormal
	ideo, alpha := ui.TextBaselineIdeographic, ui.TextBaselineAlphabetic
	even, prop := ui.TextLeadingDistributionEven, ui.TextLeadingDistributionProportional
	under, over, through := ui.TextDecorationUnderline, ui.TextDecorationOverline, ui.TextDecorationLineThrough
	dashed, wavy, dotted := ui.TextDecorationStyleDashed, ui.TextDecorationStyleWavy, ui.TextDecorationStyleDotted
	ellipsisOverflow, fade, visible, clip := TextOverflowEllipsis, TextOverflowFade, TextOverflowVisible, TextOverflowClip
	fg := ui.NewPaint()
	fg.SetColor(ui.NewColor(0xFF112233))
	bg := ui.NewPaint()
	bg.SetColor(ui.NewColor(0xFF445566))

	styles := []*TextStyle{
		nil,
		NewTextStyle(),
		NewTextStyle(TextStyleOptions{Inherit: b(false)}),
		NewTextStyle(TextStyleOptions{Color: c(0xFF00FF00), FontSize: f(12), FontWeight: fw(ui.FontWeightW700),
			FontStyle: &fsItalic, DebugLabel: s("a")}),
		NewTextStyle(TextStyleOptions{BackgroundColor: c(0x80FF0000), LetterSpacing: f(1.5), WordSpacing: f(-2),
			TextBaseline: &ideo, Height: f(1.25), LeadingDistribution: &even, Locale: loc("fr", "CA")}),
		NewTextStyle(TextStyleOptions{Foreground: fg, Background: bg, FontSize: f(20)}),
		NewTextStyle(TextStyleOptions{
			Shadows:        []ui.Shadow{ui.NewShadow(ui.ShadowOptions{Color: c(0xFF000000), Offset: ui.NewOffset(1, 2), BlurRadius: 3})},
			FontFeatures:   []ui.FontFeature{ui.NewFontFeatureTabularFigures()},
			FontVariations: []ui.FontVariation{ui.NewFontVariation("wght", 300), ui.NewFontVariation("wdth", 80)},
		}),
		NewTextStyle(TextStyleOptions{Decoration: &under, DecorationColor: c(0xFF0000FF), DecorationStyle: &dashed,
			DecorationThickness: f(2)}),
		NewTextStyle(TextStyleOptions{DecorationThickness: f(0.5)}),
		NewTextStyle(TextStyleOptions{FontFamily: s("Roboto"), FontFamilyFallback: []string{"Noto", "Emoji"}, Package: s("pkg"),
			Overflow: &ellipsisOverflow, DebugLabel: s("pkg")}),
		NewTextStyle(TextStyleOptions{FontFamily: s("Ahem"), FontFamilyFallback: []string{"A"}, Inherit: b(false), FontSize: f(8),
			Color: c(0xFF123456), Height: f(0), FontWeight: fw(ui.FontWeightW100)}),
		NewTextStyle(TextStyleOptions{Shadows: []ui.Shadow{}, FontFeatures: []ui.FontFeature{},
			FontVariations: []ui.FontVariation{ui.NewFontVariation("wght", 900), ui.NewFontVariation("opsz", 12), ui.NewFontVariation("slnt", -5)},
			FontSize:       f(30), LetterSpacing: f(0), WordSpacing: f(4), Height: f(2), DecorationThickness: f(1), Color: c(0xFFABCDEF)}),
		NewTextStyle(TextStyleOptions{Inherit: b(false), BackgroundColor: c(0xFF00FFFF), FontSize: f(14), LetterSpacing: f(1),
			WordSpacing: f(1), Height: f(1), DecorationColor: c(0xFF111111), DecorationThickness: f(3), Color: c(0x01020304)}),
		NewTextStyle(TextStyleOptions{Package: s("only"), Decoration: &through}),
	}

	center, rtl := ui.TextAlignCenter, ui.TextDirectionRtl
	hidden := ui.HyphensHidden
	three := 3
	for i, st := range styles {
		if st == nil {
			continue
		}
		pre := fmt.Sprintf("style %d", i)
		add(pre + ": " + tstErr(func() any { return st }))
		add(pre + " deep: " + tstErr(func() any { return st.ToDiagnosticsNode().ToStringDeep() }))
		fine := foundation.DiagnosticLevelFine
		add(pre + " fine: " + tstErr(func() any {
			return st.ToDiagnosticsNode().ToStringDeep(foundation.DiagnosticsNodeToStringDeepOptions{MinLevel: &fine})
		}))
		add(pre + " props: " + tstErr(func() any {
			return tsProps(func(b *foundation.DiagnosticPropertiesBuilder) { st.DebugFillPropertiesWithPrefix(b, "p.") })
		}))
		add(pre + " fallback: " + tsStrings(st.FontFamilyFallback()) + " family: " + nullableStr(st.FontFamily))
		add(pre + " ui: " + st.GetTextStyle().String())
		add(pre + " ui x2: " + st.GetTextStyle(TextStyleGetTextStyleOptions{TextScaler: NewTextScalerLinear(2)}).String())
		add(pre + " ui f1.5: " + st.GetTextStyle(TextStyleGetTextStyleOptions{TextScaleFactor: f(1.5)}).String())
		add(pre + " ui both: " + tstErr(func() any {
			return st.GetTextStyle(TextStyleGetTextStyleOptions{TextScaleFactor: f(1.5), TextScaler: NewTextScalerLinear(2)})
		}))
		add(pre + " para: " + st.GetParagraphStyle().String())
		first := false
		thb := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{ApplyHeightToFirstAscent: &first})
		ps := st.GetParagraphStyle(TextStyleGetParagraphStyleOptions{
			TextAlign: &center, TextDirection: &rtl, TextScaler: NewTextScalerLinear(1.5), Ellipsis: s("…"), MaxLines: &three,
			TextHeightBehavior: &thb, Hyphens: &hidden, Locale: loc("ja"), FontFamily: s("Ahem"), FontSize: f(9),
			FontWeight: fw(ui.FontWeightW500), FontStyle: &fsItalic, Height: f(1.7),
			StrutStyle: NewStrutStyle(StrutStyleOptions{FontSize: f(10), Height: f(2), Leading: f(0.5), ForceStrutHeight: b(true)}),
		})
		add(pre + " para2: " + ps.String())
		strut3 := func(force *bool) *StrutStyle {
			return NewStrutStyle(StrutStyleOptions{FontFamily: s("Ahem"), FontSize: f(20), LeadingDistribution: &even, Height: f(1),
				ForceStrutHeight: force})
		}
		add(pre + " para3: " + st.GetParagraphStyle(TextStyleGetParagraphStyleOptions{StrutStyle: strut3(nil),
			TextScaler: NewTextScalerLinear(0.5)}).String())
		zero := 0
		add(pre + " maxLines0: " + tstErr(func() any { return st.GetParagraphStyle(TextStyleGetParagraphStyleOptions{MaxLines: &zero}) }))
		add(pre + " layout: " + tsLayout(st.GetParagraphStyle(), st.GetTextStyle()))
		add(pre + " layout2: " + tsLayout(ps, st.GetTextStyle(TextStyleGetTextStyleOptions{TextScaler: NewTextScalerLinear(1.5)})))
		add(pre + " layout3: " + tsLayout(st.GetParagraphStyle(TextStyleGetParagraphStyleOptions{StrutStyle: strut3(b(true))}), nil))
		add(pre + " copy: " + tstErr(func() any { return st.CopyWith() }))
		add(pre + " copy2: " + tstErr(func() any {
			return st.CopyWith(TextStyleCopyWithOptions{Color: c(0xFF0000AA), BackgroundColor: c(0xFF00AA00), FontSize: f(33), DebugLabel: s("cw")})
		}))
		add(pre + " copy3: " + tstErr(func() any {
			return st.CopyWith(TextStyleCopyWithOptions{Foreground: bg, Background: fg, Inherit: b(false), FontFamily: s("Z"),
				Package: s("q"), FontFamilyFallback: []string{"f"}, Overflow: &fade})
		}))
		add(pre + " copy4: " + tstErr(func() any { return st.CopyWith(TextStyleCopyWithOptions{Color: c(0xFF000000), Foreground: fg}) }))
		add(pre + " apply: " + tstErr(func() any { return st.Apply() }))
		add(pre + " apply2: " + tstErr(func() any {
			return st.Apply(TextStyleApplyOptions{Color: c(0xFF999999), BackgroundColor: c(0xFF888888), Decoration: &over,
				DecorationColor: c(0xFF777777), DecorationStyle: &wavy, FontFamily: s("F"), FontFamilyFallback: []string{"G"},
				FontStyle: &fsNormal, TextBaseline: &alpha, LeadingDistribution: &prop, Locale: loc("es"),
				Shadows: []ui.Shadow{ui.NewShadow(ui.ShadowOptions{BlurRadius: 1})}, FontFeatures: []ui.FontFeature{ui.NewFontFeatureEnable("smcp")},
				FontVariations: []ui.FontVariation{ui.NewFontVariation("wght", 1)}, Package: s("ap"), Overflow: &visible})
		}))
		add(pre + " apply3: " + tstErr(func() any { return st.Apply(TextStyleApplyOptions{FontSizeFactor: f(1.1), FontSizeDelta: 0.3}) }))
		add(pre + " apply4: " + tstErr(func() any { return st.Apply(TextStyleApplyOptions{FontWeightDelta: 3}) }))
		add(pre + " apply5: " + tstErr(func() any { return st.Apply(TextStyleApplyOptions{FontWeightDelta: -10}) }))
		add(pre + " apply6: " + tstErr(func() any {
			return st.Apply(TextStyleApplyOptions{LetterSpacingFactor: f(0.3), LetterSpacingDelta: 0.1, WordSpacingFactor: f(3),
				WordSpacingDelta: -0.7})
		}))
		add(pre + " apply7: " + tstErr(func() any { return st.Apply(TextStyleApplyOptions{HeightFactor: f(1.1), HeightDelta: 0.2}) }))
		add(pre + " apply8: " + tstErr(func() any {
			return st.Apply(TextStyleApplyOptions{DecorationThicknessFactor: f(0.7), DecorationThicknessDelta: 0.1})
		}))
	}
	for i, a := range styles {
		for j, bb := range styles {
			pre := fmt.Sprintf("pair %d %d", i, j)
			if a != nil {
				add(pre + " merge: " + tstErr(func() any { return a.Merge(bb) }))
				if bb != nil {
					add(fmt.Sprintf("%s eq: %t cmp: %s", pre, a.Equal(bb), a.CompareTo(bb)))
				}
			}
			for _, tt := range []float64{0, 0.3, 0.5, 0.75, 1} {
				add(pre + " lerp " + dart.DoubleToString(tt) + ": " + tstErr(func() any { return TextStyleLerp(a, bb, tt) }))
			}
		}
	}
	base := NewTextStyle(TextStyleOptions{FontSize: f(10), Color: c(0xFF000000)})
	for _, o := range []TextStyleCopyWithOptions{
		{}, {Color: c(0xFF000001)}, {BackgroundColor: c(0xFF000001)}, {Decoration: &under}, {DecorationColor: c(0xFF000001)},
		{DecorationStyle: &dotted}, {DecorationThickness: f(2)}, {FontSize: f(11)}, {FontWeight: fw(ui.FontWeightW300)},
		{FontStyle: &fsItalic}, {LetterSpacing: f(1)}, {WordSpacing: f(1)}, {TextBaseline: &alpha}, {Height: f(1)},
		{LeadingDistribution: &even}, {Locale: loc("en")}, {Shadows: []ui.Shadow{}}, {FontFeatures: []ui.FontFeature{}},
		{FontVariations: []ui.FontVariation{}}, {FontFamily: s("f")}, {FontFamilyFallback: []string{}}, {Overflow: &clip},
		{Inherit: b(false)}, {DebugLabel: s("x")}, {Package: s("p")}, {Foreground: fg}, {Background: bg},
	} {
		other := base.CopyWith(o)
		add(fmt.Sprintf("cmp %s %s %t", base.CompareTo(other), other.CompareTo(base), base.Equal(other)))
	}

	add(fmt.Sprintf("pkg eq %t %s", NewTextStyle(TextStyleOptions{FontFamily: s("packages/p/f")}).Equal(NewTextStyle(TextStyleOptions{FontFamily: s("f"), Package: s("p")})),
		NewTextStyle(TextStyleOptions{FontFamily: s("packages/p/f")}).CompareTo(NewTextStyle(TextStyleOptions{FontFamily: s("f"), Package: s("p")}))))
	ltrDir := ui.TextDirectionLtr
	badText := dart.StringFromCharCodes([]uint16{0xD800, 0x78})
	bad := NewTextPainter(TextPainterOptions{Text: NewTextSpan(TextSpanOptions{Text: &badText, Style: NewTextStyle(TextStyleOptions{FontSize: f(10)})}),
		TextDirection: &ltrDir})
	bad.Layout()
	add(fmt.Sprintf("malformed %s %d", bad.Size(), dart.StringLength(bad.PlainText())))
	bad.Dispose()

	fv := ui.NewFontVariation
	variations := [][]ui.FontVariation{
		nil, {}, {fv("wght", 100)}, {fv("wght", 200), fv("wdth", 50)}, {fv("wdth", 50), fv("wght", 200)},
		{fv("opsz", 10), fv("slnt", -10), fv("wght", 400), fv("ital", 1)},
		{fv("wght", 900), fv("GRAD", 50), fv("abcd", 3), fv("efgh", 4), fv("ijkl", 5), fv("mnop", 6), fv("qrst", 7)},
		{fv("wght", 300), fv("opsz", 20)},
		{fv("xxxx", 1), fv("xxxx", 2), fv("yyyy", 3)},
	}
	for i := range variations {
		for j := range variations {
			for _, tt := range []float64{0, 0.25, 0.5, 0.6, 1} {
				r := LerpFontVariations(variations[i], variations[j], tt)
				str := "null"
				if r != nil {
					str = tpList(r)
				}
				add(fmt.Sprintf("fv %d %d %s: %s", i, j, dart.DoubleToString(tt), str))
			}
		}
	}

	struts := []*StrutStyle{
		nil,
		NewStrutStyle(),
		StrutStyleDisabled,
		NewStrutStyle(StrutStyleOptions{FontFamily: s("Roboto"), FontSize: f(12), Height: f(1.5), LeadingDistribution: &even,
			Leading: f(0.25), FontWeight: fw(ui.FontWeightW600), FontStyle: &fsItalic, ForceStrutHeight: b(true), DebugLabel: s("s")}),
		NewStrutStyle(StrutStyleOptions{FontFamily: s("A"), FontFamilyFallback: []string{"B", "C"}, Package: s("pk")}),
		NewStrutStyle(StrutStyleOptions{ForceStrutHeight: b(false)}),
		NewStrutStyle(StrutStyleOptions{Height: f(2)}),
		NewStrutStyle(StrutStyleOptions{LeadingDistribution: &prop}),
		NewStrutStyle(StrutStyleOptions{FontFamilyFallback: []string{"X"}, Package: s("p")}),
		NewStrutStyle(StrutStyleOptions{Height: f(2), LeadingDistribution: &prop}),
	}
	for i, st := range struts {
		if st == nil {
			continue
		}
		pre := fmt.Sprintf("strut %d", i)
		add(pre + ": " + st.String())
		add(pre + " deep: " + st.ToDiagnosticsNode().ToStringDeep())
		add(pre + " props: " + tsProps(func(b *foundation.DiagnosticPropertiesBuilder) { st.DebugFillPropertiesWithPrefix(b, "q.") }))
		add(pre + " fallback: " + tsStrings(st.FontFamilyFallback()) + " family: " + nullableStr(st.FontFamily))
		for k := range styles {
			add(fmt.Sprintf("%s inherit %d: %s %s", pre, k, tstErr(func() any { return st.InheritFromTextStyle(styles[k]) }), tpStr(ptrF(st.InheritFromTextStyle(styles[k]).Leading))))
		}
	}
	for i, a := range struts {
		for j, bb := range struts {
			if a != nil {
				line := fmt.Sprintf("strut pair %d %d merge: %s", i, j, tstErr(func() any { return a.Merge(bb) }))
				if bb != nil {
					line += fmt.Sprintf(" eq: %t cmp: %s", a.Equal(bb), a.CompareTo(bb))
				}
				add(line)
			}
		}
	}
	for k := 1; k < len(styles); k++ {
		add(fmt.Sprintf("fromTextStyle %d: %s", k, tstErr(func() any { return NewStrutStyleFromTextStyle(styles[k]) })))
		add(fmt.Sprintf("fromTextStyle2 %d: %s", k, tstErr(func() any {
			return NewStrutStyleFromTextStyle(styles[k], StrutStyleOptions{FontFamily: s("F"), FontFamilyFallback: []string{"G"},
				FontSize: f(7), Height: f(3), LeadingDistribution: &even, Leading: f(1), FontWeight: fw(ui.FontWeightW900),
				FontStyle: &fsNormal, ForceStrutHeight: b(true), DebugLabel: s("ft"), Package: s("fp")})
		})))
	}
	add("strut errors: " + tstErr(func() any { return NewStrutStyle(StrutStyleOptions{FontSize: f(0)}) }) + " " +
		tstErr(func() any { return NewStrutStyle(StrutStyleOptions{Leading: f(-1)}) }) + " " +
		tstErr(func() any { return NewStrutStyle(StrutStyleOptions{Package: s("p")}) }))
	add("style errors: " + tstErr(func() any { return NewTextStyle(TextStyleOptions{Color: c(0xFF000000), Foreground: fg}) }) + " " +
		tstErr(func() any { return NewTextStyle(TextStyleOptions{BackgroundColor: c(0xFF000000), Background: bg}) }) + " " +
		tstErr(func() any { return NewTextStyle(TextStyleOptions{Height: f(math.NaN())}) }))

	// Spans.
	tap := gestures.NewTapGestureRecognizer()
	enter := func(*gestures.PointerEnterEvent) {}
	span := func(o TextSpanOptions) *TextSpan { return NewTextSpan(o) }
	xy := func(st *TextStyle) TextSpanOptions {
		return TextSpanOptions{Text: s("x"), Style: st, Children: []InlineSpan{span(TextSpanOptions{Text: s("y")})}}
	}
	black := NewTextStyle(TextStyleOptions{Color: c(0xFF000000)})
	spans := []InlineSpan{
		span(TextSpanOptions{Text: s("Hello "), Style: NewTextStyle(TextStyleOptions{FontSize: f(10)}), Children: []InlineSpan{
			span(TextSpanOptions{Text: s("world"), SemanticsLabel: s("WORLD"), Locale: loc("en"), SpellOut: b(true)}),
			newTPPlaceholder(),
			span(TextSpanOptions{Children: []InlineSpan{span(TextSpanOptions{Text: s("😀x"), Recognizer: tap})}}),
			span(TextSpanOptions{Text: s(""), SpellOut: b(false)}),
		}}),
		span(TextSpanOptions{}),
		span(TextSpanOptions{Text: s("a"), SemanticsIdentifier: s("id"), Locale: loc("ja", "JP"), Children: []InlineSpan{
			span(TextSpanOptions{Text: s("b"), SpellOut: b(true), Children: []InlineSpan{span(TextSpanOptions{Text: s("c"), Locale: loc("de")})}}),
			span(TextSpanOptions{Text: s("d"), SemanticsLabel: s("DD")}),
		}}),
		span(TextSpanOptions{Text: s("hover"), OnEnter: enter, MouseCursor: services.SystemMouseCursors.Text, SemanticsLabel: s("")}),
		span(xy(black)),
		span(xy(NewTextStyle(TextStyleOptions{Color: c(0xFF000001)}))),
		span(xy(NewTextStyle(TextStyleOptions{FontSize: f(3)}))),
		span(xy(nil)),
		span(TextSpanOptions{Text: s("x"), Style: black, Children: []InlineSpan{
			span(TextSpanOptions{Text: s("y"), Style: NewTextStyle(TextStyleOptions{Decoration: &under})})}}),
		span(TextSpanOptions{Text: s("x"), Style: black, Recognizer: tap, Children: []InlineSpan{span(TextSpanOptions{Text: s("y")})}}),
		span(TextSpanOptions{Text: s("x"), Style: black, Children: []InlineSpan{span(TextSpanOptions{Text: s("z")})}}),
		span(TextSpanOptions{Text: s("x"), Style: black, Children: []InlineSpan{span(TextSpanOptions{Text: s("y")}), span(TextSpanOptions{Text: s("y")})}}),
		newTPPlaceholder(),
		span(xy(NewTextStyle(TextStyleOptions{Color: c(0xFF000000)}))),
		span(TextSpanOptions{Text: s("\U0001F600\U0001F600"), Children: []InlineSpan{
			span(TextSpanOptions{Text: s("q"), SemanticsLabel: s("Q"), SpellOut: b(true)})}}),
		span(TextSpanOptions{Text: s(""), Locale: loc("en"), SpellOut: b(true)}),
	}
	fls := false
	for i, sp := range spans {
		pre := fmt.Sprintf("span %d", i)
		js := func(v string) string { x, _ := json.Marshal(v); return dartJSONString(string(x)) }
		add(pre + " plain: " + js(sp.ToPlainText()) + " " + js(sp.ToPlainText(ToPlainTextOptions{IncludeSemanticsLabels: &fls})) + " " +
			js(sp.ToPlainText(ToPlainTextOptions{IncludePlaceholders: &fls})) + " " +
			js(sp.ToPlainText(ToPlainTextOptions{IncludeSemanticsLabels: &fls, IncludePlaceholders: &fls})))
		n := dart.StringLength(sp.ToPlainText(ToPlainTextOptions{IncludeSemanticsLabels: &fls}))
		var units []string
		for k := -1; k <= n+1; k++ {
			units = append(units, tpStr(sp.CodeUnitAt(k)))
		}
		add(pre + " units: " + strings.Join(units, ","))
		var at []string
		for k := 0; k <= n+1; k++ {
			for _, a := range []ui.TextAffinity{ui.TextAffinityUpstream, ui.TextAffinityDownstream} {
				at = append(at, tsSpanName(sp.GetSpanForPosition(ui.NewTextPosition(k, a))))
			}
		}
		add(pre + " at: " + strings.Join(at, ","))
		var visited []string
		r := sp.VisitChildren(func(c InlineSpan) bool { visited = append(visited, tsSpanName(c)); return true })
		add(fmt.Sprintf("%s visit: %t %s", pre, r, tsStrings(visited, true)))
		var visited2 []string
		r = sp.VisitChildren(func(c InlineSpan) bool { visited2 = append(visited2, tsSpanName(c)); return len(visited2) < 2 })
		add(fmt.Sprintf("%s visit2: %t %s", pre, r, tsStrings(visited2, true)))
		var direct []string
		r = sp.VisitDirectChildren(func(c InlineSpan) bool { direct = append(direct, tsSpanName(c)); return true })
		add(fmt.Sprintf("%s direct: %t %s", pre, r, tsStrings(direct, true)))
		sem := sp.GetSemanticsInformation()
		add(pre + " semantics: " + tsInfos(sem))
		add(pre + " combined: " + tsInfos(CombineSemanticsInfo(sem)))
		add(pre + " deep: " + sp.ToStringDeep())
		add(fmt.Sprintf("%s valid: %t", pre, sp.DebugAssertIsValid()))
	}
	for i := range spans {
		var eq, cmp []string
		for _, o := range spans {
			eq = append(eq, map[bool]string{true: "1", false: "0"}[spans[i].Equal(o)])
			cmp = append(cmp, strconv.Itoa(int(spans[i].CompareTo(o))))
		}
		add(fmt.Sprintf("span eq %d: %s", i, strings.Join(eq, "")))
		add(fmt.Sprintf("span cmp %d: %s", i, strings.Join(cmp, "")))
	}
	acc := NewAccumulator(3)
	acc.Increment(4)
	add(fmt.Sprintf("accumulator %d %d %s", NewAccumulator().Value(), acc.Value(), tstErr(func() any { acc.Increment(-1); return nil })))
	add("placeholder info " + InlineSpanSemanticsInformationPlaceholder.String() + " " + tsInfo(InlineSpanSemanticsInformationPlaceholder))
	infos := []InlineSpanSemanticsInformation{
		NewInlineSpanSemanticsInformation("ab", InlineSpanSemanticsInformationOptions{SemanticsLabel: s("AB"),
			StringAttributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 2))}}),
		NewInlineSpanSemanticsInformation("cd"),
		NewInlineSpanSemanticsInformation("ef", InlineSpanSemanticsInformationOptions{
			StringAttributes: []ui.StringAttribute{ui.NewLocaleStringAttribute(ui.NewTextRange(1, 2), ui.NewLocale("fr"))}}),
		NewInlineSpanSemanticsInformation("id", InlineSpanSemanticsInformationOptions{SemanticsIdentifier: s("i")}),
		InlineSpanSemanticsInformationPlaceholder,
		NewInlineSpanSemanticsInformation("gh", InlineSpanSemanticsInformationOptions{SemanticsLabel: s("G"),
			StringAttributes: []ui.StringAttribute{ui.NewSpellOutStringAttribute(ui.NewTextRange(0, 1))}}),
		NewInlineSpanSemanticsInformation("😀"),
	}
	var combined []string
	for _, i := range CombineSemanticsInfo(infos) {
		combined = append(combined, i.String()+" "+tsInfo(i))
	}
	add("combine " + strings.Join(combined, " "))
	add("combine empty " + tsInfos(CombineSemanticsInfo([]InlineSpanSemanticsInformation{})))
	add(fmt.Sprintf("info eq %t %t %s", NewInlineSpanSemanticsInformation("a").Equal(NewInlineSpanSemanticsInformation("a")),
		NewInlineSpanSemanticsInformation("a", InlineSpanSemanticsInformationOptions{SemanticsLabel: s("b")}).Equal(NewInlineSpanSemanticsInformation("a")),
		tstErr(func() any {
			return NewInlineSpanSemanticsInformation("a", InlineSpanSemanticsInformationOptions{IsPlaceholder: true})
		})))
	bare := &tsBarePlaceholder{}
	bare.InitPlaceholderSpan(bare)
	add("placeholder assert: " + tstErr(func() any {
		if !bare.DebugAssertIsValid() {
			panic(dart.NewAssertionError(nil))
		}
		return bare.ToPlainText()
	}))
	return out
}

// tsStrings is a Dart List<String>'s toString (nil: null; asList: an
// empty list for nil).
func tsStrings(l []string, asList ...bool) string {
	if l == nil && len(asList) == 0 {
		return "null"
	}
	return "[" + strings.Join(l, ", ") + "]"
}

func TestPaintingTextStyleMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_text_style.json")
	if err != nil {
		t.Fatalf("no reference (%v); run the Flutter side: flutter test test/painting_text_style_test.dart", err)
	}
	var want []string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	got := paintingTextStyleLines(t)
	if len(got) != len(want) {
		t.Errorf("%d lines, Flutter has %d", len(got), len(want))
	}
	bad := 0
	for i := 0; i < len(want) && i < len(got); i++ {
		if got[i] != want[i] {
			if bad++; bad <= 30 {
				t.Errorf("line %d:\n flutter %s\n go      %s", i, want[i], got[i])
			}
		}
	}
	t.Logf("%d lines, %d differ", len(want), bad)
}

// ptrF prints a nullable double as Dart does.
func ptrF(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}
