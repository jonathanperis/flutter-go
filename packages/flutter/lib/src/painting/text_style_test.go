package painting

// Port of packages/flutter/test/painting/text_style_test.dart (Flutter
// 3.50.1).
//
// Partly ported: "TextStyle.hashCode" (Go has no hashCode: the styles'
// equality is checked instead) and the hashCode half of "TextStyle
// fontFamily and package". unorderedMatches/anyOf expectations of
// "lerpFontVariations" are checked as Dart states them (any order).

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func tsColor(v int) *ui.Color { c := ui.NewColor(v); return &c }

func tsWeight(w ui.FontWeight) *ui.FontWeight { return &w }

func tsT[T any](v T) *T { return &v }

// tsPropString is _DartUiTextStyleToStringMatcher._propertyToString.
func tsPropString(name string, v any) string {
	return name + ": " + tsValueString(v)
}

func tsValueString(v any) string {
	if v == nil {
		return "unspecified"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return "unspecified"
		}
		if _, ok := v.(*ui.Paint); ok {
			return fmt.Sprint(v)
		}
		return tsValueString(rv.Elem().Interface())
	case reflect.Slice:
		if rv.IsNil() {
			return "unspecified"
		}
		parts := make([]string, rv.Len())
		for i := range parts {
			parts[i] = tsValueString(rv.Index(i).Interface())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case reflect.Float64:
		return dart.DoubleToString(rv.Float())
	}
	return fmt.Sprint(v)
}

// tsMatchesToStringOf is matchesToStringOf: ui.TextStyle.toString reports
// a superset of the painting TextStyle's properties, in order.
func tsMatchesToStringOf(t *testing.T, item *ui.TextStyle, s *TextStyle) {
	t.Helper()
	props := []string{
		tsPropString("color", s.Color),
		tsPropString("decoration", s.Decoration),
		tsPropString("decorationColor", s.DecorationColor),
		tsPropString("decorationStyle", s.DecorationStyle),
		tsPropString("decorationThickness", s.DecorationThickness),
		tsPropString("fontWeight", s.FontWeight),
		tsPropString("fontStyle", s.FontStyle),
		tsPropString("textBaseline", s.TextBaseline),
		tsPropString("fontFamily", s.FontFamily),
		tsPropString("fontFamilyFallback", s.FontFamilyFallback()),
		tsPropString("fontSize", s.FontSize),
		tsPropString("letterSpacing", s.LetterSpacing),
		tsPropString("wordSpacing", s.WordSpacing),
		tsPropString("height", s.Height),
		tsPropString("leadingDistribution", s.LeadingDistribution),
		tsPropString("locale", s.Locale),
		tsPropString("background", s.Background),
		tsPropString("foreground", s.Foreground),
		tsPropString("shadows", s.Shadows),
		tsPropString("fontFeatures", s.FontFeatures),
		tsPropString("fontVariations", s.FontVariations),
	}
	description := item.String()
	if !strings.HasPrefix(description, "TextStyle(") || !strings.HasSuffix(description, ")") {
		t.Fatalf("not a TextStyle description: %s", description)
	}
	desc := description[len("TextStyle(") : len(description)-1]
	start := 0
	for _, p := range props {
		i := strings.Index(desc[start:], p)
		if i < 0 {
			t.Errorf("missing property %q in %s", p, description)
			return
		}
		start += i + len(p)
	}
}

func tsCheck(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func tsCheckStyle(t *testing.T, s *TextStyle, family *string, size *float64, weight *ui.FontWeight, height *float64, color *ui.Color) {
	t.Helper()
	tsCheck(t, s.FontFamily, family)
	tsCheck(t, s.FontSize, size)
	tsCheck(t, s.FontWeight, weight)
	tsCheck(t, s.Height, height)
	tsCheck(t, s.Color, color)
}

func TestTextStyleControl(t *testing.T) {
	if got := NewTextStyle(TextStyleOptions{Inherit: tsT(false)}).String(); got != "TextStyle(inherit: false, <no style specified>)" {
		t.Error(got)
	}
	if got := NewTextStyle().String(); got != "TextStyle(<all styles inherited>)" {
		t.Error(got)
	}

	s1 := NewTextStyle(TextStyleOptions{FontSize: f64(10.0), FontWeight: tsWeight(ui.FontWeightW800), Height: f64(123.0)})
	tsCheckStyle(t, s1, nil, f64(10), tsWeight(ui.FontWeightW800), f64(123), nil)
	if !s1.Equal(s1) {
		t.Error("s1 == s1")
	}
	if got := s1.String(); got != "TextStyle(inherit: true, size: 10.0, weight: 800, height: 123.0x)" {
		t.Error(got)
	}
	// Check that the inherit flag can be set with copyWith().
	if got := s1.CopyWith(TextStyleCopyWithOptions{Inherit: tsT(false)}).String(); got != "TextStyle(inherit: false, size: 10.0, weight: 800, height: 123.0x)" {
		t.Error(got)
	}

	s2 := s1.CopyWith(TextStyleCopyWithOptions{Color: tsColor(0xFF00FF00), Height: f64(100.0),
		LeadingDistribution: tsT(ui.TextLeadingDistributionEven)})
	tsCheckStyle(t, s1, nil, f64(10), tsWeight(ui.FontWeightW800), f64(123), nil)
	tsCheckStyle(t, s2, nil, f64(10), tsWeight(ui.FontWeightW800), f64(100), tsColor(0xFF00FF00))
	tsCheck(t, s2.LeadingDistribution, tsT(ui.TextLeadingDistributionEven))
	if s2.Equal(s1) {
		t.Error("s2 != s1")
	}
	want := "TextStyle(inherit: true, color: " + ui.NewColor(0xff00ff00).String() +
		", size: 10.0, weight: 800, height: 100.0x, leadingDistribution: even)"
	if got := s2.String(); got != want {
		t.Errorf("%s\n%s", got, want)
	}

	s3 := s1.Apply(TextStyleApplyOptions{FontSizeFactor: f64(2.0), FontSizeDelta: -2.0, FontWeightDelta: -4})
	tsCheckStyle(t, s1, nil, f64(10), tsWeight(ui.FontWeightW800), f64(123), nil)
	tsCheckStyle(t, s3, nil, f64(18), tsWeight(ui.FontWeightW400), f64(123), nil)
	if s3.Equal(s1) {
		t.Error("s3 != s1")
	}
	tsCheck(t, s1.Apply(TextStyleApplyOptions{FontWeightDelta: -10}).FontWeight, tsWeight(ui.FontWeightW100))
	tsCheck(t, s1.Apply(TextStyleApplyOptions{FontWeightDelta: 2}).FontWeight, tsWeight(ui.FontWeightW900))
	if !s1.Merge(nil).Equal(s1) {
		t.Error("merge(null)")
	}

	s4 := s2.Merge(s1)
	tsCheckStyle(t, s1, nil, f64(10), tsWeight(ui.FontWeightW800), f64(123), nil)
	tsCheckStyle(t, s2, nil, f64(10), tsWeight(ui.FontWeightW800), f64(100), tsColor(0xFF00FF00))
	tsCheck(t, s2.LeadingDistribution, tsT(ui.TextLeadingDistributionEven))
	if s2.Equal(s1) || s2.Equal(s4) {
		t.Error("s2 != s1, s4")
	}
	tsCheckStyle(t, s4, nil, f64(10), tsWeight(ui.FontWeightW800), f64(123), tsColor(0xFF00FF00))
	tsCheck(t, s4.LeadingDistribution, tsT(ui.TextLeadingDistributionEven))

	s5 := TextStyleLerp(s1, s3, 0.25)
	tsCheckStyle(t, s1, nil, f64(10), tsWeight(ui.FontWeightW800), f64(123), nil)
	tsCheckStyle(t, s3, nil, f64(18), tsWeight(ui.FontWeightW400), f64(123), nil)
	if s3.Equal(s1) || s3.Equal(s5) {
		t.Error("s3 != s1, s5")
	}
	tsCheckStyle(t, s5, nil, f64(12), tsWeight(ui.FontWeightW700), f64(123), nil)

	if TextStyleLerp(nil, nil, 0.5) != nil {
		t.Error("lerp(null, null)")
	}

	s6 := TextStyleLerp(nil, s3, 0.25)
	tsCheckStyle(t, s3, nil, f64(18), tsWeight(ui.FontWeightW400), f64(123), nil)
	if s3.Equal(s6) {
		t.Error("s3 != s6")
	}
	tsCheckStyle(t, s6, nil, nil, tsWeight(ui.FontWeightW400), nil, nil)

	s7 := TextStyleLerp(nil, s3, 0.75)
	if !s3.Equal(s7) {
		t.Error("s3 == s7")
	}
	tsCheckStyle(t, s7, nil, f64(18), tsWeight(ui.FontWeightW400), f64(123), nil)

	s8 := TextStyleLerp(s3, nil, 0.25)
	if !s3.Equal(s8) {
		t.Error("s3 == s8")
	}
	tsCheckStyle(t, s8, nil, f64(18), tsWeight(ui.FontWeightW400), f64(123), nil)

	s9 := TextStyleLerp(s3, nil, 0.75)
	if s3.Equal(s9) {
		t.Error("s3 != s9")
	}
	tsCheckStyle(t, s9, nil, nil, tsWeight(ui.FontWeightW400), nil, nil)

	ts5 := s5.GetTextStyle()
	if !ts5.Equal(ui.NewTextStyle(ui.TextStyleOptions{FontWeight: tsWeight(ui.FontWeightW700), FontSize: f64(12.0), Height: f64(123.0)})) {
		t.Errorf("ts5 %v", ts5)
	}
	tsMatchesToStringOf(t, ts5, s5)
	ts2 := s2.GetTextStyle()
	if !ts2.Equal(ui.NewTextStyle(ui.TextStyleOptions{Color: tsColor(0xFF00FF00), FontWeight: tsWeight(ui.FontWeightW800),
		FontSize: f64(10.0), Height: f64(100.0), LeadingDistribution: tsT(ui.TextLeadingDistributionEven)})) {
		t.Errorf("ts2 %v", ts2)
	}
	tsMatchesToStringOf(t, ts2, s2)

	ps2 := s2.GetParagraphStyle(TextStyleGetParagraphStyleOptions{TextAlign: tsT(ui.TextAlignCenter)})
	thb := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{LeadingDistribution: ui.TextLeadingDistributionEven})
	if !ps2.Equal(ui.NewParagraphStyle(ui.ParagraphStyleOptions{TextAlign: tsT(ui.TextAlignCenter), FontWeight: tsWeight(ui.FontWeightW800),
		FontSize: f64(10.0), Height: f64(100.0), TextHeightBehavior: &thb})) {
		t.Errorf("ps2 %v", ps2)
	}
	ps5 := s5.GetParagraphStyle()
	if !ps5.Equal(ui.NewParagraphStyle(ui.ParagraphStyleOptions{FontWeight: tsWeight(ui.FontWeightW700), FontSize: f64(12.0), Height: f64(123.0)})) {
		t.Errorf("ps5 %v", ps5)
	}
}

func TestTextStyleWithTextDirection(t *testing.T) {
	ps6 := NewTextStyle().GetParagraphStyle(TextStyleGetParagraphStyleOptions{TextDirection: tsT(ui.TextDirectionLtr)})
	if !ps6.Equal(ui.NewParagraphStyle(ui.ParagraphStyleOptions{TextDirection: tsT(ui.TextDirectionLtr), FontSize: f64(14.0)})) {
		t.Errorf("ps6 %v", ps6)
	}
	ps7 := NewTextStyle().GetParagraphStyle(TextStyleGetParagraphStyleOptions{TextDirection: tsT(ui.TextDirectionRtl)})
	if !ps7.Equal(ui.NewParagraphStyle(ui.ParagraphStyleOptions{TextDirection: tsT(ui.TextDirectionRtl), FontSize: f64(14.0)})) {
		t.Errorf("ps7 %v", ps7)
	}
}

func tsAssertMessage(f func()) (msg string) {
	defer func() {
		r := recover()
		if a, ok := r.(dart.AssertionFailure); ok {
			msg = fmt.Sprint(a.AssertionMessage())
			return
		}
		if r != nil {
			msg = fmt.Sprintf("not an AssertionError: %v", r)
		}
	}()
	f()
	return "no error"
}

func TestTextStyleRejectsNaNHeight(t *testing.T) {
	nan := f64(0)
	*nan = *nan / *nan
	if got := tsAssertMessage(func() { NewTextStyle(TextStyleOptions{Height: nan}) }); !strings.Contains(got, "TextStyle.height must not be NaN.") {
		t.Error(got)
	}
	if got := tsAssertMessage(func() {
		NewTextStyle().GetParagraphStyle(TextStyleGetParagraphStyleOptions{Height: nan})
	}); !strings.Contains(got, "TextStyle.height must not be NaN.") {
		t.Error(got)
	}
}

func TestTextStyleUsingPackageFont(t *testing.T) {
	s6 := NewTextStyle(TextStyleOptions{FontFamily: strPtr("test")})
	tsCheck(t, *s6.FontFamily, "test")
	tsMatchesToStringOf(t, s6.GetTextStyle(), s6)

	s7 := NewTextStyle(TextStyleOptions{FontFamily: strPtr("test"), Package: strPtr("p")})
	tsCheck(t, *s7.FontFamily, "packages/p/test")
	tsMatchesToStringOf(t, s7.GetTextStyle(), s7)

	s8 := NewTextStyle(TextStyleOptions{FontFamilyFallback: []string{"test", "test2"}, Package: strPtr("p")})
	tsCheck(t, s8.FontFamilyFallback(), []string{"packages/p/test", "packages/p/test2"})

	s9 := NewTextStyle(TextStyleOptions{Package: strPtr("p")})
	if s9.FontFamilyFallback() != nil {
		t.Error("s9 fallback")
	}

	s10 := NewTextStyle(TextStyleOptions{FontFamilyFallback: []string{}, Package: strPtr("p")})
	tsCheck(t, s10.FontFamilyFallback(), []string{})

	// Ensure that package prefix is not duplicated after copying.
	s11 := s8.CopyWith()
	tsCheck(t, s11.FontFamilyFallback(), []string{"packages/p/test", "packages/p/test2"})
	if !s8.Equal(s11) {
		t.Error("s8 == s11")
	}

	// Ensure that package prefix is not duplicated after applying.
	s12 := s8.Apply()
	tsCheck(t, s12.FontFamilyFallback(), []string{"packages/p/test", "packages/p/test2"})
	if !s8.Equal(s12) {
		t.Error("s8 == s12")
	}
}

func TestTextStylePackageFontMerge(t *testing.T) {
	s1 := NewTextStyle(TextStyleOptions{Package: strPtr("p"), FontFamily: strPtr("font1"), FontFamilyFallback: []string{"fallback1"}})
	s2 := NewTextStyle(TextStyleOptions{Package: strPtr("p"), FontFamily: strPtr("font2"), FontFamilyFallback: []string{"fallback2"}})

	emptyMerge := NewTextStyle().Merge(s1)
	tsCheck(t, *emptyMerge.FontFamily, "packages/p/font1")
	tsCheck(t, emptyMerge.FontFamilyFallback(), []string{"packages/p/fallback1"})

	lerp1 := TextStyleLerp(s1, s2, 0)
	tsCheck(t, *lerp1.FontFamily, "packages/p/font1")
	tsCheck(t, lerp1.FontFamilyFallback(), []string{"packages/p/fallback1"})

	lerp2 := TextStyleLerp(s1, s2, 1.0)
	tsCheck(t, *lerp2.FontFamily, "packages/p/font2")
	tsCheck(t, lerp2.FontFamilyFallback(), []string{"packages/p/fallback2"})
}

func TestTextStyleFontFamilyFallback(t *testing.T) {
	s1 := NewTextStyle(TextStyleOptions{FontFamilyFallback: []string{"Roboto", "test"}})
	tsCheck(t, s1.FontFamilyFallback(), []string{"Roboto", "test"})

	s2 := NewTextStyle(TextStyleOptions{FontFamily: strPtr("foo"), FontFamilyFallback: []string{"Roboto", "test"}})
	tsCheck(t, s2.FontFamilyFallback(), []string{"Roboto", "test"})
	tsCheck(t, *s2.FontFamily, "foo")

	s3 := NewTextStyle(TextStyleOptions{FontFamily: strPtr("foo")})
	tsCheck(t, *s3.FontFamily, "foo")
	if s3.FontFamilyFallback() != nil {
		t.Error("s3 fallback")
	}

	s4 := NewTextStyle(TextStyleOptions{FontFamily: strPtr("foo"), FontFamilyFallback: []string{}})
	tsCheck(t, *s4.FontFamily, "foo")
	tsCheck(t, s4.FontFamilyFallback(), []string{})

	tsMatchesToStringOf(t, s2.GetTextStyle(), s2)

	tsCheck(t, *s2.Apply().FontFamily, "foo")
	tsCheck(t, s2.Apply().FontFamilyFallback(), []string{"Roboto", "test"})
	tsCheck(t, *s2.Apply(TextStyleApplyOptions{FontFamily: strPtr("bar")}).FontFamily, "bar")
	tsCheck(t, s2.Apply(TextStyleApplyOptions{FontFamilyFallback: []string{"Banana"}}).FontFamilyFallback(), []string{"Banana"})
}

func TestTextStyleDebugLabel(t *testing.T) {
	unknown := NewTextStyle()
	foo := NewTextStyle(TextStyleOptions{DebugLabel: strPtr("foo"), FontSize: f64(1.0)})
	bar := NewTextStyle(TextStyleOptions{DebugLabel: strPtr("bar"), FontSize: f64(2.0)})
	baz := NewTextStyle(TextStyleOptions{DebugLabel: strPtr("baz"), FontSize: f64(3.0)})
	label := func(s *TextStyle) any {
		if s.DebugLabel == nil {
			return nil
		}
		return *s.DebugLabel
	}

	tsCheck(t, label(unknown), nil)
	tsCheck(t, unknown.String(), "TextStyle(<all styles inherited>)")
	tsCheck(t, label(unknown.CopyWith()), nil)
	tsCheck(t, label(unknown.CopyWith(TextStyleCopyWithOptions{DebugLabel: strPtr("123")})), "123")
	tsCheck(t, label(unknown.Apply()), nil)

	tsCheck(t, label(foo), "foo")
	tsCheck(t, foo.String(), "TextStyle(debugLabel: foo, inherit: true, size: 1.0)")
	tsCheck(t, label(foo.Merge(bar)), "(foo).merge(bar)")
	tsCheck(t, label(foo.Merge(bar).Merge(baz)), "((foo).merge(bar)).merge(baz)")
	tsCheck(t, label(foo.CopyWith()), "(foo).copyWith")
	tsCheck(t, label(foo.Apply()), "(foo).apply")
	tsCheck(t, label(TextStyleLerp(foo, bar, 0.5)), "lerp(foo ⎯0.5→ bar)")
	tsCheck(t, label(TextStyleLerp(foo.Merge(bar), baz, 0.51).CopyWith()), "(lerp((foo).merge(bar) ⎯0.5→ baz)).copyWith")
}

// "TextStyle.hashCode": Go has no hashCode; the styles' equality is
// checked instead.
func TestTextStyleHashCodeEquality(t *testing.T) {
	a := NewTextStyle(TextStyleOptions{FontFamilyFallback: []string{"Roboto"}, Shadows: []ui.Shadow{ui.NewShadow()},
		FontFeatures: []ui.FontFeature{ui.NewFontFeature("abcd")}, FontVariations: []ui.FontVariation{ui.NewFontVariation("wght", 123.0)}})
	b := NewTextStyle(TextStyleOptions{FontFamilyFallback: []string{"Noto"}, Shadows: []ui.Shadow{ui.NewShadow()},
		FontFeatures: []ui.FontFeature{ui.NewFontFeature("abcd")}, FontVariations: []ui.FontVariation{ui.NewFontVariation("wght", 123.0)}})
	if !a.Equal(a) || a.Equal(b) {
		t.Error("a/b")
	}
	c := NewTextStyle(TextStyleOptions{LeadingDistribution: tsT(ui.TextLeadingDistributionEven)})
	d := NewTextStyle(TextStyleOptions{LeadingDistribution: tsT(ui.TextLeadingDistributionProportional)})
	if !c.Equal(c) || c.Equal(d) {
		t.Error("c/d")
	}
}

func TestTextStyleShadows(t *testing.T) {
	shadow1 := ui.NewShadow(ui.ShadowOptions{BlurRadius: 1.0, Offset: ui.NewOffset(1.0, 1.0)})
	shadow2 := ui.NewShadow(ui.ShadowOptions{BlurRadius: 2.0, Color: tsColor(0xFF111111), Offset: ui.NewOffset(2.0, 2.0)})
	shadow3 := ui.NewShadow(ui.ShadowOptions{BlurRadius: 3.0, Color: tsColor(0xFF222222), Offset: ui.NewOffset(3.0, 3.0)})
	shadow4 := ui.NewShadow(ui.ShadowOptions{BlurRadius: 4.0, Color: tsColor(0xFF333333), Offset: ui.NewOffset(4.0, 4.0)})
	s1 := NewTextStyle(TextStyleOptions{Shadows: []ui.Shadow{shadow1, shadow2}})
	s2 := NewTextStyle(TextStyleOptions{Shadows: []ui.Shadow{shadow3, shadow4}})
	lerp12 := TextStyleLerp(s1, s2, 0.5)
	if len(lerp12.Shadows) != 2 {
		t.Fatalf("%d shadows", len(lerp12.Shadows))
	}
	for i, pair := range [][2]ui.Shadow{{shadow1, shadow3}, {shadow2, shadow4}} {
		got := lerp12.Shadows[i]
		tsCheck(t, got.BlurRadius, *ui.LerpDouble(&pair[0].BlurRadius, &pair[1].BlurRadius, 0.5))
		tsCheck(t, got.Color, *ui.ColorLerp(&pair[0].Color, &pair[1].Color, 0.5))
		tsCheck(t, got.Offset, *ui.OffsetLerp(&pair[0].Offset, &pair[1].Offset, 0.5))
	}
}

func tsPaint(c ui.Color) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(c)
	return p
}

func TestTextStyleForegroundAndColorCombos(t *testing.T) {
	red := ui.NewColorFromARGB(255, 255, 0, 0)
	blue := ui.NewColorFromARGB(255, 0, 0, 255)
	redTextStyle := NewTextStyle(TextStyleOptions{Color: &red})
	blueTextStyle := NewTextStyle(TextStyleOptions{Color: &blue})
	redPaintTextStyle := NewTextStyle(TextStyleOptions{Foreground: tsPaint(red)})
	bluePaintTextStyle := NewTextStyle(TextStyleOptions{Foreground: tsPaint(blue)})

	// merge/copyWith
	m := redTextStyle.Merge(blueTextStyle)
	tsCheck(t, m.Color, &blue)
	if m.Foreground != nil {
		t.Error("foreground")
	}
	m = redPaintTextStyle.Merge(bluePaintTextStyle)
	if m.Color != nil || m.Foreground != bluePaintTextStyle.Foreground {
		t.Error("paint merged")
	}
	m = redPaintTextStyle.Merge(blueTextStyle)
	if m.Color != nil || m.Foreground != redPaintTextStyle.Foreground {
		t.Error("paint+color merged")
	}
	m = blueTextStyle.Merge(redPaintTextStyle)
	if m.Color != nil || m.Foreground != redPaintTextStyle.Foreground {
		t.Error("color+paint merged")
	}

	// apply
	if redPaintTextStyle.Apply(TextStyleApplyOptions{Color: &blue}).Color != nil {
		t.Error("apply color")
	}
	tsCheck(t, redPaintTextStyle.Apply(TextStyleApplyOptions{Color: &blue}).Foreground.Color(), red)
	tsCheck(t, redTextStyle.Apply(TextStyleApplyOptions{Color: &blue}).Color, &blue)

	// lerp
	tsCheck(t, TextStyleLerp(redTextStyle, blueTextStyle, .25).Color, ui.ColorLerp(&red, &blue, .25))
	if TextStyleLerp(redTextStyle, bluePaintTextStyle, .25).Color != nil {
		t.Error("lerp color")
	}
	tsCheck(t, TextStyleLerp(redTextStyle, bluePaintTextStyle, .25).Foreground.Color(), red)
	tsCheck(t, TextStyleLerp(redTextStyle, bluePaintTextStyle, .75).Foreground.Color(), blue)
	if TextStyleLerp(redPaintTextStyle, bluePaintTextStyle, .25).Color != nil {
		t.Error("lerp paint color")
	}
	tsCheck(t, TextStyleLerp(redPaintTextStyle, bluePaintTextStyle, .25).Foreground.Color(), red)
	tsCheck(t, TextStyleLerp(redPaintTextStyle, bluePaintTextStyle, .75).Foreground.Color(), blue)
}

func TestTextStyleBackgroundColor(t *testing.T) {
	s1 := NewTextStyle()
	if s1.BackgroundColor != nil {
		t.Error("s1")
	}
	tsCheck(t, s1.String(), "TextStyle(<all styles inherited>)")
	s2 := NewTextStyle(TextStyleOptions{BackgroundColor: tsColor(0xFF00FF00)})
	tsCheck(t, s2.BackgroundColor, tsColor(0xFF00FF00))
	tsCheck(t, s2.String(), "TextStyle(inherit: true, backgroundColor: "+ui.NewColor(0xff00ff00).String()+")")
	ts2 := s2.GetTextStyle()
	expectedPaint := tsPaint(ui.NewColor(0xFF00FF00))
	if !strings.Contains(ts2.String(), "background: "+expectedPaint.String()) {
		t.Error(ts2.String())
	}
}

func TestTextStyleBackgroundAndBackgroundColorCombos(t *testing.T) {
	red := ui.NewColorFromARGB(255, 255, 0, 0)
	blue := ui.NewColorFromARGB(255, 0, 0, 255)
	redTextStyle := NewTextStyle(TextStyleOptions{BackgroundColor: &red})
	blueTextStyle := NewTextStyle(TextStyleOptions{BackgroundColor: &blue})
	redPaintTextStyle := NewTextStyle(TextStyleOptions{Background: tsPaint(red)})
	bluePaintTextStyle := NewTextStyle(TextStyleOptions{Background: tsPaint(blue)})

	m := redTextStyle.Merge(blueTextStyle)
	tsCheck(t, m.BackgroundColor, &blue)
	if m.Foreground != nil {
		t.Error("foreground")
	}
	m = redPaintTextStyle.Merge(bluePaintTextStyle)
	if m.BackgroundColor != nil || m.Background != bluePaintTextStyle.Background {
		t.Error("paint merged")
	}
	m = redPaintTextStyle.Merge(blueTextStyle)
	if m.BackgroundColor != nil || m.Background != redPaintTextStyle.Background {
		t.Error("paint+color merged")
	}
	m = blueTextStyle.Merge(redPaintTextStyle)
	if m.BackgroundColor != nil || m.Background != redPaintTextStyle.Background {
		t.Error("color+paint merged")
	}

	if redPaintTextStyle.Apply(TextStyleApplyOptions{BackgroundColor: &blue}).BackgroundColor != nil {
		t.Error("apply")
	}
	tsCheck(t, redPaintTextStyle.Apply(TextStyleApplyOptions{BackgroundColor: &blue}).Background.Color(), red)
	tsCheck(t, redTextStyle.Apply(TextStyleApplyOptions{BackgroundColor: &blue}).BackgroundColor, &blue)

	tsCheck(t, TextStyleLerp(redTextStyle, blueTextStyle, .25).BackgroundColor, ui.ColorLerp(&red, &blue, .25))
	if TextStyleLerp(redTextStyle, bluePaintTextStyle, .25).BackgroundColor != nil {
		t.Error("lerp")
	}
	tsCheck(t, TextStyleLerp(redTextStyle, bluePaintTextStyle, .25).Background.Color(), red)
	tsCheck(t, TextStyleLerp(redTextStyle, bluePaintTextStyle, .75).Background.Color(), blue)
	if TextStyleLerp(redPaintTextStyle, bluePaintTextStyle, .25).BackgroundColor != nil {
		t.Error("lerp paint")
	}
	tsCheck(t, TextStyleLerp(redPaintTextStyle, bluePaintTextStyle, .25).Background.Color(), red)
	tsCheck(t, TextStyleLerp(redPaintTextStyle, bluePaintTextStyle, .75).Background.Color(), blue)
}

func TestTextStyleStrutTextScaler(t *testing.T) {
	paragraphStyle0 := NewTextStyle(TextStyleOptions{FontSize: f64(10)}).GetParagraphStyle(
		TextStyleGetParagraphStyleOptions{TextScaler: NewTextScalerLinear(2.5)})
	paragraphStyle1 := NewTextStyle(TextStyleOptions{FontSize: f64(25)}).GetParagraphStyle()
	if !paragraphStyle0.Equal(paragraphStyle1) {
		t.Errorf("%v != %v", paragraphStyle0, paragraphStyle1)
	}
}

func TestTextStyleApply(t *testing.T) {
	style := NewTextStyle(TextStyleOptions{
		FontSize: f64(10), Shadows: []ui.Shadow{}, FontStyle: tsT(ui.FontStyleNormal), FontFeatures: []ui.FontFeature{},
		FontVariations: []ui.FontVariation{}, TextBaseline: tsT(ui.TextBaselineAlphabetic),
		LeadingDistribution: tsT(ui.TextLeadingDistributionEven),
	})
	tsCheck(t, style.Apply().Shadows, []ui.Shadow{})
	sh := []ui.Shadow{ui.NewShadow(ui.ShadowOptions{BlurRadius: 2.0})}
	tsCheck(t, style.Apply(TextStyleApplyOptions{Shadows: sh}).Shadows, sh)
	tsCheck(t, style.Apply().FontStyle, tsT(ui.FontStyleNormal))
	tsCheck(t, style.Apply(TextStyleApplyOptions{FontStyle: tsT(ui.FontStyleItalic)}).FontStyle, tsT(ui.FontStyleItalic))
	if style.Apply().Locale != nil {
		t.Error("locale")
	}
	es := ui.NewLocaleFromSubtags(ui.LocaleFromSubtagsOptions{LanguageCode: "es"})
	if l := style.Apply(TextStyleApplyOptions{Locale: &es}).Locale; l == nil || !l.Equal(es) {
		t.Error("es")
	}
	tsCheck(t, style.Apply().FontFeatures, []ui.FontFeature{})
	ff := []ui.FontFeature{ui.NewFontFeatureEnable("test")}
	tsCheck(t, style.Apply(TextStyleApplyOptions{FontFeatures: ff}).FontFeatures, ff)
	tsCheck(t, style.Apply().FontVariations, []ui.FontVariation{})
	fv := []ui.FontVariation{ui.NewFontVariation("test", 100.0)}
	tsCheck(t, style.Apply(TextStyleApplyOptions{FontVariations: fv}).FontVariations, fv)
	tsCheck(t, style.Apply().TextBaseline, tsT(ui.TextBaselineAlphabetic))
	tsCheck(t, style.Apply(TextStyleApplyOptions{TextBaseline: tsT(ui.TextBaselineIdeographic)}).TextBaseline, tsT(ui.TextBaselineIdeographic))
	tsCheck(t, style.Apply().LeadingDistribution, tsT(ui.TextLeadingDistributionEven))
	tsCheck(t, style.Apply(TextStyleApplyOptions{LeadingDistribution: tsT(ui.TextLeadingDistributionProportional)}).LeadingDistribution,
		tsT(ui.TextLeadingDistributionProportional))

	tsCheck(t, NewTextStyle(TextStyleOptions{Height: f64(ui.KTextHeightNone)}).Apply(
		TextStyleApplyOptions{HeightFactor: f64(1000), HeightDelta: 1000}).Height, f64(ui.KTextHeightNone))
}

// The hashCode comparison of "TextStyle fontFamily and package" has no Go
// equivalent; the rest is ported.
func TestTextStyleFontFamilyAndPackage(t *testing.T) {
	ff := strPtr("fontFamily")
	if NewTextStyle(TextStyleOptions{FontFamily: ff, Package: strPtr("foo")}).Equal(NewTextStyle(TextStyleOptions{FontFamily: ff, Package: strPtr("bar")})) {
		t.Error("!=")
	}
	tsCheck(t, *NewTextStyle(TextStyleOptions{FontFamily: ff}).FontFamily, "fontFamily")
	tsCheck(t, *NewTextStyle(TextStyleOptions{FontFamily: ff}).CopyWith(TextStyleCopyWithOptions{Package: strPtr("bar")}).FontFamily, "packages/bar/fontFamily")
	tsCheck(t, *NewTextStyle(TextStyleOptions{FontFamily: ff, Package: strPtr("foo")}).FontFamily, "packages/foo/fontFamily")
	tsCheck(t, *NewTextStyle(TextStyleOptions{FontFamily: ff, Package: strPtr("foo")}).CopyWith(TextStyleCopyWithOptions{Package: strPtr("bar")}).FontFamily, "packages/bar/fontFamily")
	tsCheck(t, *NewTextStyle().Merge(NewTextStyle(TextStyleOptions{FontFamily: ff, Package: strPtr("bar")})).FontFamily, "packages/bar/fontFamily")
	tsCheck(t, *NewTextStyle().Apply(TextStyleApplyOptions{FontFamily: ff, Package: strPtr("foo")}).FontFamily, "packages/foo/fontFamily")
	tsCheck(t, *NewTextStyle(TextStyleOptions{FontFamily: ff, Package: strPtr("foo")}).Apply(TextStyleApplyOptions{FontFamily: ff, Package: strPtr("bar")}).FontFamily, "packages/bar/fontFamily")
}

func TestTextStyleLerpIdentical(t *testing.T) {
	if TextStyleLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	style := NewTextStyle()
	if TextStyleLerp(style, style, 0.5) != style {
		t.Error("identical")
	}
}

func TestTextStyleLerpThrowsOnInheritMismatch(t *testing.T) {
	fromStyle := NewTextStyle()
	toStyle := NewTextStyle(TextStyleOptions{Inherit: tsT(false)})
	func() {
		defer func() {
			if _, ok := recover().(*foundation.FlutterError); !ok {
				t.Error("expected a FlutterError")
			}
		}()
		TextStyleLerp(fromStyle, toStyle, 0.5)
	}()
	if !TextStyleLerp(fromStyle, fromStyle, 0.5).Equal(fromStyle) {
		t.Error("from")
	}
}

func TestTextStyleLerpInheritMismatchFullySpecified(t *testing.T) {
	fromStyle := NewTextStyle()
	toStyle := NewTextStyle(TextStyleOptions{Inherit: tsT(false), Color: tsColor(0x87654321), BackgroundColor: tsColor(0x12345678),
		FontSize: f64(20), LetterSpacing: f64(1), WordSpacing: f64(1), Height: f64(20), DecorationColor: tsColor(0x11111111),
		DecorationThickness: f64(5)})
	if got := TextStyleLerp(fromStyle, toStyle, 1); !got.Equal(toStyle) {
		t.Errorf("%v", got)
	}
}

func TestLerpFontVariations(t *testing.T) {
	type fvs = []ui.FontVariation
	empty := fvs{}
	eq := func(got, want fvs) {
		t.Helper()
		if (got == nil) != (want == nil) || !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	unordered := func(got, want fvs) {
		t.Helper()
		if len(got) != len(want) {
			t.Errorf("got %v, want %v (any order)", got, want)
			return
		}
		for _, w := range want {
			if !slices.Contains(got, w) {
				t.Errorf("got %v, want %v (any order)", got, want)
				return
			}
		}
	}
	// nil cases
	eq(LerpFontVariations(empty, empty, 0.0), empty)
	eq(LerpFontVariations(empty, empty, 0.5), empty)
	eq(LerpFontVariations(empty, empty, 1.0), empty)
	eq(LerpFontVariations(nil, empty, 0.0), nil)
	eq(LerpFontVariations(empty, nil, 0.0), empty)
	eq(LerpFontVariations(nil, nil, 0.0), nil)
	eq(LerpFontVariations(nil, empty, 0.5), empty)
	eq(LerpFontVariations(empty, nil, 0.5), nil)
	eq(LerpFontVariations(nil, nil, 0.5), nil)
	eq(LerpFontVariations(nil, empty, 1.0), empty)
	eq(LerpFontVariations(empty, nil, 1.0), nil)
	eq(LerpFontVariations(nil, nil, 1.0), nil)

	w := ui.NewFontVariationWeight
	w100, w120, w150, w200, w300, w1000 := w(100), w(120), w(150), w(200), w(300), w(1000)

	// one axis
	eq(LerpFontVariations(fvs{w100}, fvs{w200}, 0.0), fvs{w100})
	eq(LerpFontVariations(fvs{w100}, fvs{w200}, 0.2), fvs{w120})
	eq(LerpFontVariations(fvs{w100}, fvs{w200}, 0.5), fvs{w150})
	eq(LerpFontVariations(fvs{w100}, fvs{w200}, 2.0), fvs{w300})

	// weird one axis cases
	eq(LerpFontVariations(fvs{w100, w1000}, fvs{w300}, 0.0), fvs{w100, w1000})
	eq(LerpFontVariations(fvs{w100, w1000}, fvs{w300}, 0.5), fvs{w200})
	eq(LerpFontVariations(fvs{w100, w1000}, fvs{w300}, 1.0), fvs{w300})
	eq(LerpFontVariations(fvs{w100, w1000}, empty, 0.5), empty)

	s := ui.NewFontVariationSlant
	sn80, sn40, s0, sp40, sp80 := s(-80), s(-40), s(0), s(40), s(80)

	// two axis matched order
	eq(LerpFontVariations(fvs{w100, sn80}, fvs{w300, sp80}, 0.5), fvs{w200, s0})

	// two axis unmatched order
	eq(LerpFontVariations(fvs{sn80, w100}, fvs{w300, sp80}, 0.0), fvs{sn80, w100})
	unordered(LerpFontVariations(fvs{sn80, w100}, fvs{w300, sp80}, 0.5), fvs{s0, w200})
	eq(LerpFontVariations(fvs{sn80, w100}, fvs{w300, sp80}, 1.0), fvs{w300, sp80})

	// two axis with duplicates
	unordered(LerpFontVariations(fvs{sn80, w100, sp80}, fvs{w300, sp80}, 0.5), fvs{sp80, w200})

	// mixed axis counts
	eq(LerpFontVariations(fvs{sn80, w100}, fvs{w300}, 0.5), fvs{w200})
	eq(LerpFontVariations(fvs{sn80}, fvs{w300}, 0.0), fvs{sn80})
	eq(LerpFontVariations(fvs{sn80}, fvs{w300}, 0.1), fvs{sn80})
	eq(LerpFontVariations(fvs{sn80}, fvs{w300}, 0.9), fvs{w300})
	eq(LerpFontVariations(fvs{sn80}, fvs{w300}, 1.0), fvs{w300})
	got := LerpFontVariations(fvs{sn40, s0, w100}, fvs{sp40, w300, sp80}, 0.5)
	if !slices.Equal(got, fvs{s0, w200, sp40}) && !slices.Equal(got, fvs{s0, sp40, w200}) {
		t.Errorf("got %v", got)
	}
}
