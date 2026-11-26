package painting

// Tests of placeholder_span.dart (Flutter 3.50.1), which has no test file
// of its own in Flutter: PlaceholderSpan's defaults, plain text, semantics
// information, diagnostics and its debugAssertIsValid (which asserts:
// concrete placeholder spans override it).

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
)

// ptextbBarePlaceholder keeps PlaceholderSpan's debugAssertIsValid.
type ptextbBarePlaceholder struct{ ptextbWidgetSpan }

func (s *ptextbBarePlaceholder) DebugAssertIsValid() bool {
	return s.PlaceholderSpanBase.DebugAssertIsValid()
}

func TestPlaceholderSpanDefaults(t *testing.T) {
	if PlaceholderSpanPlaceholderCodeUnit != 0xFFFC {
		t.Error("placeholderCodeUnit")
	}
	s := newPtextbWidgetSpan(ptextbSizedBox{})
	if s.Alignment() != ui.PlaceholderAlignmentBottom || s.Baseline() != nil || s.Style() != nil {
		t.Errorf("defaults %v %v", s.Alignment(), s.Baseline())
	}
	var _ PlaceholderSpan = s
	top, ideo := ui.PlaceholderAlignmentTop, ui.TextBaselineIdeographic
	style := NewTextStyle()
	s2 := newPtextbWidgetSpan(ptextbSizedBox{}, PlaceholderSpanOptions{Alignment: &top, Baseline: &ideo, Style: style})
	if s2.Alignment() != top || *s2.Baseline() != ideo || s2.Style() != style {
		t.Error("options")
	}
}

func TestPlaceholderSpanPlainTextAndSemantics(t *testing.T) {
	s := newPtextbWidgetSpan(ptextbSizedBox{})
	var b strings.Builder
	s.ComputeToPlainText(&b)
	f := false
	s.ComputeToPlainText(&b, ToPlainTextOptions{IncludePlaceholders: &f})
	s.ComputeToPlainText(&b, ToPlainTextOptions{IncludeSemanticsLabels: &f})
	if b.String() != "￼￼" {
		t.Errorf("%q", b.String())
	}
	collector := []InlineSpanSemanticsInformation{}
	s.ComputeSemanticsInformation(&collector)
	if len(collector) != 1 || !collector[0].Equal(InlineSpanSemanticsInformationPlaceholder) {
		t.Errorf("%v", collector)
	}
}

func TestPlaceholderSpanDebugFillProperties(t *testing.T) {
	describe := func(s InlineSpan) []string {
		b := foundation.NewDiagnosticPropertiesBuilder()
		s.DebugFillProperties(b)
		var out []string
		for _, p := range b.Properties {
			if !p.IsFiltered(foundation.DiagnosticLevelInfo) {
				out = append(out, p.String())
			}
		}
		if b.DefaultDiagnosticsTreeStyle != foundation.DiagnosticsTreeStyleWhitespace {
			t.Error("whitespace tree style")
		}
		return out
	}
	if got := strings.Join(describe(newPtextbWidgetSpan(ptextbSizedBox{})), "; "); got != "alignment: bottom" {
		t.Errorf("got %q", got)
	}
	base, alpha := ui.PlaceholderAlignmentBaseline, ui.TextBaselineAlphabetic
	s := newPtextbWidgetSpan(ptextbSizedBox{}, PlaceholderSpanOptions{Alignment: &base, Baseline: &alpha,
		Style: NewTextStyle(TextStyleOptions{FontSize: f64(12)})})
	if got := strings.Join(describe(s), "; "); got != "inherit: true; size: 12.0; alignment: baseline; baseline: alphabetic" {
		t.Errorf("got %q", got)
	}
}

func TestPlaceholderSpanDebugAssertIsValid(t *testing.T) {
	bare := &ptextbBarePlaceholder{}
	bare.InitPlaceholderSpan(bare)
	if got := tsAssertMessage(func() { bare.DebugAssertIsValid() }); got != "Consider implementing the WidgetSpan interface instead." {
		t.Errorf("got %q", got)
	}
	// A TextSpan containing it is invalid too.
	if got := tsAssertMessage(func() { tspText("a", bare).DebugAssertIsValid() }); got == "no error" {
		t.Error("TextSpan with an invalid child must assert")
	}
}
