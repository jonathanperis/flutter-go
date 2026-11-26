package painting

// Port of packages/flutter/test/painting/strut_style_test.dart (Flutter
// 3.50.1). Nothing skipped. The extra tests below cover the rest of
// strut_style.dart (merge, inheritFromTextStyle, fromTextStyle, compareTo,
// packages, disabled), which Flutter tests only through rendering.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func TestStrutStyleDiagnostics(t *testing.T) {
	check := func(s *StrutStyle, want string) {
		t.Helper()
		if got := s.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
	serif := strPtr("Serif")
	s0 := NewStrutStyle(StrutStyleOptions{FontFamily: serif, FontSize: f64(14)})
	check(s0, "StrutStyle(family: Serif, size: 14.0)")

	s1 := NewStrutStyle(StrutStyleOptions{FontFamily: serif, FontSize: f64(14), ForceStrutHeight: tsT(true)})
	if *s1.FontFamily != "Serif" || *s1.FontSize != 14.0 || !s1.Equal(s1) {
		t.Error("s1")
	}
	check(s1, "StrutStyle(family: Serif, size: 14.0, <strut height forced>)")

	s2 := NewStrutStyle(StrutStyleOptions{FontFamily: serif, FontSize: f64(14), ForceStrutHeight: tsT(false)})
	check(s2, "StrutStyle(family: Serif, size: 14.0, <strut height normal>)")

	check(NewStrutStyle(), "StrutStyle")
	check(NewStrutStyle(StrutStyleOptions{ForceStrutHeight: tsT(false)}), "StrutStyle(<strut height normal>)")
	check(NewStrutStyle(StrutStyleOptions{ForceStrutHeight: tsT(true)}), "StrutStyle(<strut height forced>)")

	s6 := NewStrutStyle(StrutStyleOptions{Height: f64(14), LeadingDistribution: tsT(ui.TextLeadingDistributionEven)})
	check(s6, "StrutStyle(height: 14.0x, leadingDistribution: even)")
	s7 := NewStrutStyle(StrutStyleOptions{Height: f64(14), LeadingDistribution: tsT(ui.TextLeadingDistributionProportional)})
	check(s7, "StrutStyle(height: 14.0x, leadingDistribution: proportional)")
	if s6.Equal(s7) {
		t.Error("s6 != s7")
	}
}

func TestStrutStyleMergeInheritAndCompare(t *testing.T) {
	base := NewStrutStyle(StrutStyleOptions{FontFamily: strPtr("A"), FontSize: f64(10), Leading: f64(1)})
	other := NewStrutStyle(StrutStyleOptions{FontSize: f64(20), ForceStrutHeight: tsT(true)})
	if base.Merge(nil) != base {
		t.Error("merge(null)")
	}
	m := base.Merge(other)
	if *m.FontFamily != "A" || *m.FontSize != 20 || *m.Leading != 1 || !*m.ForceStrutHeight {
		t.Errorf("merge %v", m)
	}
	if base.CompareTo(base) != RenderComparisonIdentical || base.CompareTo(other) != RenderComparisonLayout {
		t.Error("compareTo")
	}
	if base.CompareTo(NewStrutStyle(StrutStyleOptions{FontFamily: strPtr("A"), FontSize: f64(10), Leading: f64(1)})) != RenderComparisonIdentical {
		t.Error("compareTo equal")
	}
	// leadingDistribution only matters with a height.
	a := NewStrutStyle(StrutStyleOptions{LeadingDistribution: tsT(ui.TextLeadingDistributionEven)})
	if a.CompareTo(NewStrutStyle()) != RenderComparisonIdentical || !a.Equal(NewStrutStyle()) {
		t.Error("leadingDistribution without height")
	}

	ts := NewTextStyle(TextStyleOptions{FontFamily: strPtr("T"), FontSize: f64(30), Height: f64(2),
		LeadingDistribution: tsT(ui.TextLeadingDistributionEven), FontWeight: tsWeight(ui.FontWeightW700),
		DebugLabel: strPtr("ts")})
	if NewStrutStyle().InheritFromTextStyle(nil) == nil {
		t.Error("inherit(null)")
	}
	in := NewStrutStyle(StrutStyleOptions{FontSize: f64(5)}).InheritFromTextStyle(ts)
	if *in.FontFamily != "T" || *in.FontSize != 5 || *in.Height != 2 || *in.LeadingDistribution != ui.TextLeadingDistributionEven ||
		*in.FontWeight != ui.FontWeightW700 || *in.DebugLabel != "ts" {
		t.Errorf("inherit %v", in)
	}
	noHeight := NewStrutStyle().InheritFromTextStyle(NewTextStyle(TextStyleOptions{LeadingDistribution: tsT(ui.TextLeadingDistributionEven)}))
	if noHeight.LeadingDistribution != nil {
		t.Error("leadingDistribution needs a height")
	}

	from := NewStrutStyleFromTextStyle(ts, StrutStyleOptions{FontFamily: strPtr("F"), Package: strPtr("p"), Leading: f64(0.5)})
	// Dart quirk: fromTextStyle prefixes the family and then passes the
	// package on to StrutStyle(), which prefixes it again.
	if *from.FontFamily != "packages/p/packages/p/F" || *from.FontSize != 30 || *from.Height != 2 || *from.Leading != 0.5 {
		t.Errorf("fromTextStyle %v", from)
	}
	pk := NewStrutStyle(StrutStyleOptions{FontFamilyFallback: []string{"a", "b"}, Package: strPtr("q")})
	tsCheck(t, pk.FontFamilyFallback(), []string{"packages/q/a", "packages/q/b"})
	if *pk.FontFamily != "packages/q/null" {
		t.Errorf("family %v", *pk.FontFamily)
	}
	if !StrutStyleDisabled.Equal(NewStrutStyle(StrutStyleOptions{Height: f64(0), Leading: f64(0)})) {
		t.Error("disabled")
	}
	for _, bad := range []func(){
		func() { NewStrutStyle(StrutStyleOptions{FontSize: f64(0)}) },
		func() { NewStrutStyle(StrutStyleOptions{Leading: f64(-1)}) },
		func() { NewStrutStyle(StrutStyleOptions{Package: strPtr("p")}) },
	} {
		if got := tsAssertMessage(bad); got == "no error" || len(got) > 4 && got[:4] == "not " {
			t.Errorf("expected an AssertionError, got %s", got)
		}
	}
}
