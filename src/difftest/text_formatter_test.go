package difftest

// The text formatter difftest (flutter/test/text_formatter_test.dart):
// replays every case Flutter recorded in build/text_formatter.json —
// Pattern.allMatches (dart.RegExp and String patterns), the filtering and
// length-limiting formatters on many edited values, and
// TextEditingValue.replaced/toJSON — and compares the results exactly
// (strings as UTF-16 code units, toString included).

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type tfValue struct {
	T  []uint16 `json:"t"`
	B  int      `json:"b"`
	E  int      `json:"e"`
	A  string   `json:"a"`
	D  bool     `json:"d"`
	CS int      `json:"cs"`
	CE int      `json:"ce"`
}

func (v tfValue) value() services.TextEditingValue {
	aff := ui.TextAffinityDownstream
	if v.A == "upstream" {
		aff = ui.TextAffinityUpstream
	}
	sel := services.NewTextSelection(v.B, v.E, services.TextSelectionOptions{Affinity: &aff, IsDirectional: v.D})
	comp := ui.NewTextRange(v.CS, v.CE)
	return services.NewTextEditingValue(services.TextEditingValueOptions{Text: dart.StringFromCharCodes(v.T),
		Selection: &sel, Composing: &comp})
}

func encodeTF(v services.TextEditingValue) tfValue {
	return tfValue{T: dart.StringCodeUnits(v.Text), B: v.Selection.BaseOffset, E: v.Selection.ExtentOffset,
		A: v.Selection.Affinity.Name(), D: v.Selection.IsDirectional, CS: v.Composing.Start, CE: v.Composing.End}
}

func sameTF(a, b tfValue) bool {
	return slices.Equal(a.T, b.T) && a.B == b.B && a.E == b.E && a.A == b.A && a.D == b.D && a.CS == b.CS && a.CE == b.CE
}

type textFormatterRef struct {
	AllMatches []struct {
		P []uint16 `json:"p"`
		R bool     `json:"r"`
		F string   `json:"f"`
		T []uint16 `json:"t"`
		S int      `json:"s"`
		M [][]any  `json:"m"`
	} `json:"allMatches"`
	FilterSpecs [][]any `json:"filterSpecs"`
	Filtering   []struct {
		Spec int      `json:"spec"`
		New  tfValue  `json:"new"`
		Out  tfValue  `json:"out"`
		Str  []uint16 `json:"str"`
	} `json:"filtering"`
	LengthLimiting []struct {
		Platform string   `json:"platform"`
		Max      *int     `json:"max"`
		Enf      *string  `json:"enf"`
		Old      tfValue  `json:"old"`
		New      tfValue  `json:"new"`
		Out      tfValue  `json:"out"`
		Str      []uint16 `json:"str"`
	} `json:"lengthLimiting"`
	Defaults map[string]string `json:"defaults"`
	Replaced []struct {
		In             tfValue  `json:"in"`
		RS             int      `json:"rs"`
		RE             int      `json:"re"`
		Rep            []uint16 `json:"rep"`
		Out            tfValue  `json:"out"`
		JSON           []uint16 `json:"json"`
		ComposingValid bool     `json:"composingValid"`
	} `json:"replaced"`
}

var (
	textFormatterOnce sync.Once
	textFormatterData *textFormatterRef
	textFormatterErr  error
)

func loadTextFormatterRef(t *testing.T) *textFormatterRef {
	textFormatterOnce.Do(func() {
		data, err := os.ReadFile("flutter/build/text_formatter.json")
		if err != nil {
			textFormatterErr = err
			return
		}
		textFormatterData = &textFormatterRef{}
		textFormatterErr = json.Unmarshal(data, textFormatterData)
	})
	if textFormatterData == nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", textFormatterErr)
	}
	if textFormatterErr != nil {
		t.Fatal(textFormatterErr)
	}
	return textFormatterData
}

func goPattern(p []uint16, isRegExp bool, flags string) dart.Pattern {
	src := dart.StringFromCharCodes(p)
	if !isRegExp {
		return dart.StringPattern(src)
	}
	cs := !strings.Contains(flags, "i")
	return dart.NewRegExp(src, dart.RegExpOptions{MultiLine: strings.Contains(flags, "m"), CaseSensitive: &cs,
		Unicode: strings.Contains(flags, "u"), DotAll: strings.Contains(flags, "s")})
}

func describeMatches(ms []dart.Match) string {
	var parts []string
	for _, m := range ms {
		s := fmt.Sprint(m.Start, ",", m.End)
		for g := 1; g <= m.GroupCount(); g++ {
			if v := m.Group(g); v != nil {
				s += fmt.Sprint(",", dart.StringCodeUnits(*v))
			} else {
				s += ",null"
			}
		}
		parts = append(parts, "["+s+"]")
	}
	return strings.Join(parts, " ")
}

func describeRefMatches(ms [][]any) string {
	var parts []string
	for _, m := range ms {
		s := fmt.Sprint(m[0], ",", m[1])
		for _, g := range m[2:] {
			if g == nil {
				s += ",null"
				continue
			}
			var u []uint16
			for _, x := range g.([]any) {
				u = append(u, uint16(x.(float64)))
			}
			s += fmt.Sprint(",", u)
		}
		parts = append(parts, "["+s+"]")
	}
	return strings.Join(parts, " ")
}

func TestTextFormatterAllMatchesMatchFlutter(t *testing.T) {
	ref := loadTextFormatterRef(t)
	if len(ref.AllMatches) < 1000 {
		t.Fatalf("only %d cases", len(ref.AllMatches))
	}
	failures := 0
	for _, c := range ref.AllMatches {
		got := describeMatches(dart.AllMatches(goPattern(c.P, c.R, c.F), dart.StringFromCharCodes(c.T), c.S))
		if want := describeRefMatches(c.M); got != want && failures < 20 {
			failures++
			t.Errorf("%q (regexp %v %q) on %x from %d: flutter %s, go %s", dart.StringFromCharCodes(c.P), c.R, c.F, c.T, c.S, want, got)
		}
	}
}

func TestTextFormatterFilteringMatchesFlutter(t *testing.T) {
	ref := loadTextFormatterRef(t)
	var formatters []*services.FilteringTextInputFormatter
	for _, s := range ref.FilterSpecs {
		toUnits := func(x any) []uint16 {
			var u []uint16
			for _, e := range x.([]any) {
				u = append(u, uint16(e.(float64)))
			}
			return u
		}
		formatters = append(formatters, services.NewFilteringTextInputFormatter(
			goPattern(toUnits(s[1]), s[2].(bool), s[4].(string)), s[0] == "allow",
			services.FilteringTextInputFormatterOptions{ReplacementString: dart.StringFromCharCodes(toUnits(s[3]))}))
	}
	failures := 0
	for _, c := range ref.Filtering {
		out := formatters[c.Spec].FormatEditUpdate(services.TextEditingValueEmpty, c.New.value())
		if (!sameTF(encodeTF(out), c.Out) || !slices.Equal(dart.StringCodeUnits(out.String()), c.Str)) && failures < 20 {
			failures++
			t.Errorf("spec %v on %+v:\nflutter %s\ngo      %s", ref.FilterSpecs[c.Spec], c.New,
				dart.StringFromCharCodes(c.Str), out)
		}
	}
	if len(ref.Filtering) < 1000 {
		t.Fatalf("only %d cases", len(ref.Filtering))
	}
}

func TestTextFormatterLengthLimitingMatchesFlutter(t *testing.T) {
	ref := loadTextFormatterRef(t)
	defer foundation.SetDebugDefaultTargetPlatformOverride(nil)
	failures := 0
	for _, c := range ref.LengthLimiting {
		p := foundation.TargetPlatformAndroid
		if c.Platform == "iOS" {
			p = foundation.TargetPlatformIOS
		}
		foundation.SetDebugDefaultTargetPlatformOverride(&p)
		var opts services.LengthLimitingTextInputFormatterOptions
		if c.Enf != nil {
			for _, e := range services.MaxLengthEnforcementValues {
				if e.Name() == *c.Enf {
					opts.MaxLengthEnforcement = &e
				}
			}
		}
		f := services.NewLengthLimitingTextInputFormatter(c.Max, opts)
		out := f.FormatEditUpdate(c.Old.value(), c.New.value())
		if (!sameTF(encodeTF(out), c.Out) || !slices.Equal(dart.StringCodeUnits(out.String()), c.Str)) && failures < 20 {
			failures++
			t.Errorf("%s max %v enf %v old %+v new %+v:\nflutter %s\ngo      %s", c.Platform, c.Max, c.Enf, c.Old, c.New,
				dart.StringFromCharCodes(c.Str), out)
		}
	}
	for _, p := range foundation.TargetPlatformValues {
		if got := services.LengthLimitingTextInputFormatterGetDefaultMaxLengthEnforcement(p).String(); got != ref.Defaults[p.Name()] {
			t.Errorf("default for %v: flutter %s, go %s", p, ref.Defaults[p.Name()], got)
		}
	}
	if len(ref.LengthLimiting) < 1000 {
		t.Fatalf("only %d cases", len(ref.LengthLimiting))
	}
}

func TestTextEditingValueReplacedMatchesFlutter(t *testing.T) {
	ref := loadTextFormatterRef(t)
	failures := 0
	for _, c := range ref.Replaced {
		out := c.In.value().Replaced(ui.NewTextRange(c.RS, c.RE), dart.StringFromCharCodes(c.Rep))
		js := dart.StringCodeUnits(dart.JSONEncode(out.ToJSON()))
		if (!sameTF(encodeTF(out), c.Out) || !slices.Equal(js, c.JSON) || out.IsComposingRangeValid() != c.ComposingValid) && failures < 20 {
			failures++
			t.Errorf("%+v replaced (%d,%d) %x: flutter %+v %s, go %+v %s", c.In, c.RS, c.RE, c.Rep, c.Out,
				dart.StringFromCharCodes(c.JSON), encodeTF(out), dart.StringFromCharCodes(js))
		}
	}
}
