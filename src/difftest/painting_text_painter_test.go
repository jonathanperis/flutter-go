package difftest

// Go side of the TextPainter differential test
// (flutter/test/painting_text_painter_test.dart writes
// build/painting_text_painter.json): the scenes of
// painting_text_painter_scenes.json laid out at each width in turn by one
// TextPainter and queried exhaustively, plus state changes and the static
// helpers. Every line is compared exactly.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// tpErr is the Dart test's err(): the value's toString, or the error.
func tpErr(f func() any) (s string) {
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
				s = "error: " + e.Error()
			} else {
				s = "error: " + fmt.Sprint(r)
			}
		}
	}()
	return tpStr(f())
}

func tpStr(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case *int:
		if x == nil {
			return "null"
		}
		return strconv.Itoa(*x)
	case float64:
		return dart.DoubleToString(x)
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprint(v)
}

func tpList[T fmt.Stringer](l []T) string {
	parts := make([]string, len(l))
	for i, x := range l {
		parts[i] = x.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func tpDbl(v any) float64 {
	if s, ok := v.(string); ok {
		f, err := strconv.ParseFloat(s, 64)
		if s == "Infinity" {
			return math.Inf(1)
		}
		if err != nil {
			panic(err)
		}
		return f
	}
	return v.(float64)
}

// tpPlaceholder is the Dart test's TestPlaceholder: a placeholder span
// built as widgets' WidgetSpan builds one.
type tpPlaceholder struct{ PlaceholderSpanBase }

func newTPPlaceholder() *tpPlaceholder {
	p := &tpPlaceholder{}
	p.InitPlaceholderSpan(p)
	return p
}

func (p *tpPlaceholder) Build(builder ui.ParagraphBuilder, opts ...InlineSpanBuildOptions) {
	o := opts[0]
	scaler := o.TextScaler
	if scaler == nil {
		scaler = TextScalerNoScaling
	}
	if p.Style() != nil {
		builder.PushStyle(p.Style().GetTextStyle(TextStyleGetTextStyleOptions{TextScaler: scaler}))
	}
	d := o.Dimensions[builder.PlaceholderCount()]
	builder.AddPlaceholder(d.Size.Width, d.Size.Height, d.Alignment,
		ui.AddPlaceholderOptions{Baseline: d.Baseline, BaselineOffset: d.BaselineOffset})
	if p.Style() != nil {
		builder.Pop()
	}
}

func (p *tpPlaceholder) VisitChildren(visitor InlineSpanVisitor) bool { return visitor(p) }
func (p *tpPlaceholder) VisitDirectChildren(InlineSpanVisitor) bool   { return true }

func (p *tpPlaceholder) GetSpanForPositionVisitor(position ui.TextPosition, offset *Accumulator) InlineSpan {
	if position.Offset == offset.Value() {
		return p
	}
	offset.Increment(1)
	return nil
}

func (p *tpPlaceholder) CodeUnitAtVisitor(index int, offset *Accumulator) *int {
	expectedOffset := index-offset.Value() == 0
	offset.Increment(1)
	if expectedOffset {
		v := PlaceholderSpanPlaceholderCodeUnit
		return &v
	}
	return nil
}

func (p *tpPlaceholder) CompareTo(other InlineSpan) RenderComparison {
	if InlineSpan(p) == other {
		return RenderComparisonIdentical
	}
	return RenderComparisonLayout
}

func (p *tpPlaceholder) DebugAssertIsValid() bool { return true }
func (p *tpPlaceholder) ToStringShort() string    { return "TestPlaceholder" }

func tpStyle(v any) *TextStyle {
	if v == nil {
		return nil
	}
	s := v.(map[string]any)
	o := TextStyleOptions{}
	num := func(k string) *float64 {
		if s[k] == nil {
			return nil
		}
		f := tpDbl(s[k])
		return &f
	}
	o.FontSize, o.LetterSpacing, o.WordSpacing, o.Height = num("size"), num("ls"), num("ws"), num("height")
	if f, ok := s["family"].(string); ok {
		o.FontFamily = &f
	}
	if l, ok := s["lead"].(string); ok {
		o.LeadingDistribution = textEnum(ui.TextLeadingDistributionValues, l)
	}
	return NewTextStyle(o)
}

func tpSpan(v map[string]any) InlineSpan {
	if v["ph"] == true {
		return newTPPlaceholder()
	}
	o := TextSpanOptions{Style: tpStyle(v["style"])}
	if t, ok := v["text"].(string); ok {
		o.Text = &t
	}
	if l, ok := v["label"].(string); ok {
		o.SemanticsLabel = &l
	}
	if c, ok := v["children"].([]any); ok {
		o.Children = []InlineSpan{}
		for _, x := range c {
			o.Children = append(o.Children, tpSpan(x.(map[string]any)))
		}
	}
	return NewTextSpan(o)
}

func tpPainter(scene map[string]any) *TextPainter {
	p, _ := scene["painter"].(map[string]any)
	if p == nil {
		p = map[string]any{}
	}
	str := func(k, def string) string {
		if s, ok := p[k].(string); ok {
			return s
		}
		return def
	}
	dir := *textEnum(ui.TextDirectionValues, str("dir", "ltr"))
	var painter *TextPainter
	if p["scaleFactor"] != nil {
		f := tpDbl(p["scaleFactor"])
		painter = NewTextPainter(TextPainterOptions{Text: tpSpan(scene["span"].(map[string]any)), TextDirection: &dir,
			TextScaleFactor: &f})
	} else {
		o := TextPainterOptions{
			Text:           tpSpan(scene["span"].(map[string]any)),
			TextAlign:      textEnum(ui.TextAlignValues, str("align", "start")),
			TextDirection:  &dir,
			TextScaler:     TextScalerNoScaling,
			TextWidthBasis: *textEnum(TextWidthBasisValues, str("basis", "parent")),
			Hyphens:        *textEnum(ui.HyphensValues, str("hyphens", "manual")),
		}
		if sc, ok := p["scaler"].([]any); ok {
			if sc[0] == "linear" {
				o.TextScaler = NewTextScalerLinear(tpDbl(sc[1]))
			} else {
				lo, hi := tpDbl(sc[2]), tpDbl(sc[3])
				o.TextScaler = NewTextScalerLinear(tpDbl(sc[1])).Clamp(TextScalerClampOptions{MinScaleFactor: &lo, MaxScaleFactor: &hi})
			}
		}
		if st, ok := p["strut"].(map[string]any); ok {
			if st["disabled"] == true {
				o.StrutStyle = StrutStyleDisabled
			} else {
				so := StrutStyleOptions{}
				num := func(k string) *float64 {
					if st[k] == nil {
						return nil
					}
					f := tpDbl(st[k])
					return &f
				}
				so.FontSize, so.Height, so.Leading = num("size"), num("height"), num("leading")
				if f, ok := st["family"].(string); ok {
					so.FontFamily = &f
				}
				if f, ok := st["force"].(bool); ok {
					so.ForceStrutHeight = &f
				}
				o.StrutStyle = NewStrutStyle(so)
			}
		}
		if m, ok := p["maxLines"].(float64); ok {
			n := int(m)
			o.MaxLines = &n
		}
		if e, ok := p["ellipsis"].(string); ok {
			o.Ellipsis = &e
		}
		if l, ok := p["locale"].([]any); ok {
			loc := ui.NewLocale(l[0].(string), l[1].(string))
			o.Locale = &loc
		}
		if thb, ok := p["thb"].(map[string]any); ok {
			first, last := thb["first"].(bool), thb["last"].(bool)
			b := ui.NewTextHeightBehavior(ui.TextHeightBehaviorOptions{ApplyHeightToFirstAscent: &first, ApplyHeightToLastDescent: &last})
			o.TextHeightBehavior = &b
		}
		painter = NewTextPainter(o)
	}
	if phs, ok := scene["placeholders"].([]any); ok {
		var dims []PlaceholderDimensions
		for _, x := range phs {
			ph := x.([]any)
			d := PlaceholderDimensions{Size: ui.NewSize(tpDbl(ph[0]), tpDbl(ph[1])),
				Alignment: *textEnum(ui.PlaceholderAlignmentValues, ph[2].(string))}
			if ph[3] != nil {
				d.Baseline = textEnum(ui.TextBaselineValues, ph[3].(string))
			}
			if ph[4] != nil {
				f := tpDbl(ph[4])
				d.BaselineOffset = &f
			}
			dims = append(dims, d)
		}
		painter.SetPlaceholderDimensions(dims)
	}
	return painter
}

func tpGlyph(g *ui.GlyphInfo) string {
	if g == nil {
		return "null"
	}
	return g.GraphemeClusterLayoutBounds.String() + " " + g.GraphemeClusterCodeUnitRange.String() + " " + g.WritingDirection.String()
}

func tpDump(out *[]string, key string, p *TextPainter) {
	add := func(s string) { *out = append(*out, s) }
	d := dart.DoubleToString
	plain := p.PlainText()
	n := dart.StringLength(plain)
	add(key + " size " + tpErr(func() any { return p.Size() }) + " min " + d(p.MinIntrinsicWidth()) + " max " +
		d(p.MaxIntrinsicWidth()) + " exceeded " + strconv.FormatBool(p.DidExceedMaxLines()) + " alpha " +
		d(p.ComputeDistanceToActualBaseline(ui.TextBaselineAlphabetic)) + " ideo " +
		d(p.ComputeDistanceToActualBaseline(ui.TextBaselineIdeographic)) + " preferred " + d(p.PreferredLineHeight()))
	add(key + " lines " + tpList(p.ComputeLineMetrics()))
	add(key + " placeholders " + tpList(p.InlinePlaceholderBoxes()))
	caret := ui.NewRectFromLTWH(0, 0, 2, 10)
	for i := 0; i <= n+1; i++ {
		for _, a := range []ui.TextAffinity{ui.TextAffinityUpstream, ui.TextAffinityDownstream} {
			pos := ui.NewTextPosition(i, a)
			add(fmt.Sprintf("%s caret %d %s %s %s", key, i, a.Name(), tpErr(func() any { return p.GetOffsetForCaret(pos, caret) }),
				tpErr(func() any { return p.GetFullHeightForCaret(pos, caret) })))
		}
	}
	full := n <= 24
	for i := 0; i <= n; i++ {
		for j := i; j <= n; j++ {
			if !full && j-i > 3 && j != n {
				continue
			}
			add(fmt.Sprintf("%s boxes %d %d %s", key, i, j, tpList(p.GetBoxesForSelection(services.NewTextSelection(i, j)))))
		}
	}
	if n > 1 {
		for _, h := range ui.BoxHeightStyleValues {
			for _, w := range ui.BoxWidthStyleValues {
				o := GetBoxesForSelectionOptions{BoxHeightStyle: h, BoxWidthStyle: w}
				add(fmt.Sprintf("%s boxes %s %s %s %s", key, h.Name(), w.Name(),
					tpList(p.GetBoxesForSelection(services.NewTextSelection(1, n), o)),
					tpList(p.GetBoxesForSelection(services.NewTextSelection(n, 0), o))))
			}
		}
	}
	width := p.Width()
	if math.IsInf(width, 0) || math.IsNaN(width) {
		width = 100
	}
	for yi := 0; yi <= 6; yi++ {
		y := -4 + (p.Height()+8)*float64(yi)/6
		var row []string
		for xi := 0; xi <= 8; xi++ {
			x := -4 + (width+8)*float64(xi)/8
			o := ui.NewOffset(x, y)
			row = append(row, p.GetPositionForOffset(o).String()+" "+tpGlyph(p.GetClosestGlyphForOffset(o)))
		}
		add(fmt.Sprintf("%s grid %d %s", key, yi, strings.Join(row, " | ")))
	}
	words := p.WordBoundaries()
	move := words.MoveByWordBoundary()
	for i := -1; i <= n+1; i++ {
		add(fmt.Sprintf("%s words %d %s %s %s %s %s %s %s %s %s", key, i,
			tpErr(func() any { return p.GetWordBoundary(ui.NewTextPosition(i)) }),
			tpErr(func() any { return p.GetWordBoundary(ui.NewTextPosition(i, ui.TextAffinityUpstream)) }),
			tpErr(func() any { return p.GetLineBoundary(ui.NewTextPosition(i)) }),
			tpErr(func() any { return p.GetLineBoundary(ui.NewTextPosition(i, ui.TextAffinityUpstream)) }),
			tpErr(func() any { return words.GetTextBoundaryAt(i) }),
			tpErr(func() any { return move.GetLeadingTextBoundaryAt(i) }),
			tpErr(func() any { return move.GetTrailingTextBoundaryAt(i) }),
			tpErr(func() any { return p.GetOffsetAfter(i) }),
			tpErr(func() any { return p.GetOffsetBefore(i) })))
	}
	canvas := flutter_test.NewTestRecordingCanvas()
	p.Paint(canvas, ui.NewOffset(3, 5))
	var calls []string
	for _, inv := range canvas.Invocations {
		var args []string
		for _, a := range inv.Invocation.PositionalArguments[1:] {
			args = append(args, tpStr(a))
		}
		calls = append(calls, `Symbol("`+inv.Invocation.MemberName+`") [`+strings.Join(args, ", ")+"]")
	}
	add(key + " paint [" + strings.Join(calls, ", ") + "]")
}

func paintingTextPainterLines(t *testing.T) []string {
	EngineUseTestFontsAndRoboto(t)
	var out []string
	add := func(s string) { out = append(out, s) }
	data, err := os.ReadFile("painting_text_painter_scenes.json")
	if err != nil {
		t.Fatal(err)
	}
	var scenes []map[string]any
	if err := json.Unmarshal(data, &scenes); err != nil {
		t.Fatal(err)
	}
	for _, scene := range scenes {
		p := tpPainter(scene)
		js, _ := json.Marshal(p.PlainText())
		add(scene["name"].(string) + " plain " + dartJSONString(string(js)))
		for _, w := range scene["widths"].([]any) {
			minW, maxW := tpDbl(w.([]any)[0]), tpDbl(w.([]any)[1])
			key := fmt.Sprintf("%s [%s,%s]", scene["name"], dart.DoubleToString(minW), dart.DoubleToString(maxW))
			if e := tpErr(func() any {
				p.Layout(TextPainterLayoutOptions{MinWidth: minW, MaxWidth: &maxW})
				return "ok"
			}); e != "ok" {
				add(key + " layout " + e)
				continue
			}
			tpDump(&out, key, p)
		}
		p.Dispose()
	}

	// State changes.
	text := func(s string, color *ui.Color) InlineSpan {
		size := 10.0
		return NewTextSpan(TextSpanOptions{Text: &s, Style: NewTextStyle(TextStyleOptions{FontSize: &size, Color: color})})
	}
	black, red := ui.NewColor(0xFF000000), ui.NewColor(0xFFFF0000)
	ltr, rtl := ui.TextDirectionLtr, ui.TextDirectionRtl
	f := func(v float64) *float64 { return &v }
	p := NewTextPainter(TextPainterOptions{Text: text("state", &black), TextDirection: &ltr})
	add("never laid out: " + tpErr(func() any { return p.Width() }))
	add("paint before layout: " + tpErr(func() any { p.Paint(flutter_test.NewTestRecordingCanvas(), ui.OffsetZero); return nil }))
	p.Layout()
	add("size " + p.Size().String())
	p.SetText(text("state", &red))
	add("after color change size " + p.Size().String())
	c1 := flutter_test.NewTestRecordingCanvas()
	p.Paint(c1, ui.OffsetZero)
	add(fmt.Sprintf("painted %d", len(c1.Invocations)))
	p.SetTextWidthBasis(TextWidthBasisLongestLine)
	add("width after basis change: " + tpErr(func() any { return p.Width() }))
	add("height after basis change: " + tpErr(func() any { return p.Height() }))
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(30)})
	add("size " + p.Size().String())
	p.SetTextAlign(ui.TextAlignRight)
	alignErr := strings.SplitN(tpErr(func() any { return p.Width() }), "\n", 3)
	add("after align: " + strings.Join(alignErr[:min(2, len(alignErr))], "\n"))
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(100)})
	add("right " + p.Size().String() + " " + p.GetOffsetForCaret(ui.NewTextPosition(0), ui.RectZero).String())
	p.SetText(text("state longer", nil))
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(100)})
	add("text change " + p.Size().String() + " " + p.PlainText())
	p.SetTextScaler(NewTextScalerLinear(2))
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(100)})
	add("scaler " + p.Size().String() + " " + d2s(p.PreferredLineHeight()) + " " + tpStr(p.TextScaler()))
	p.SetTextScaleFactor(1.5)
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(100)})
	add("scale factor " + p.Size().String() + " " + d2s(p.TextScaleFactor()))
	one, ell := 1, "…"
	p.SetMaxLines(&one)
	p.SetEllipsis(&ell)
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(40)})
	add("ellipsis " + p.Size().String() + " " + strconv.FormatBool(p.DidExceedMaxLines()))
	force := true
	p.SetStrutStyle(NewStrutStyle(StrutStyleOptions{FontSize: f(30), ForceStrutHeight: &force}))
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(40)})
	add("strut " + p.Size().String() + " " + d2s(p.GetFullHeightForCaret(ui.NewTextPosition(1), ui.RectZero)))
	p.SetTextDirection(&rtl)
	p.Layout(TextPainterLayoutOptions{MaxWidth: f(200)})
	add("rtl " + p.Size().String() + " " + p.GetOffsetForCaret(ui.NewTextPosition(2), ui.RectZero).String())
	p.SetTextDirection(nil)
	add("null direction: " + tpErr(func() any { p.Layout(); return nil }))
	p.SetTextDirection(&ltr)
	p.SetText(nil)
	add("null text: " + tpErr(func() any { p.Layout(); return nil }))
	zero, empty := 0, ""
	add("maxLines 0: " + tpErr(func() any { p.SetMaxLines(&zero); return nil }))
	add("empty ellipsis: " + tpErr(func() any { p.SetEllipsis(&empty); return nil }))
	p.Dispose()
	add("debugDisposed " + strconv.FormatBool(p.DebugDisposed()))

	for _, max := range []float64{math.Inf(1), 50, 20} {
		t := text("compute me now", nil)
		add("compute " + d2s(max) + " " +
			d2s(TextPainterComputeWidth(t, ltr, TextPainterComputeOptions{MaxWidth: &max})) + " " +
			d2s(TextPainterComputeMaxIntrinsicWidth(t, rtl, TextPainterComputeOptions{MaxWidth: &max, MinWidth: 5})) + " " +
			d2s(TextPainterComputeWidth(t, ltr, TextPainterComputeOptions{MaxWidth: &max, TextScaler: NewTextScalerLinear(1.5),
				TextWidthBasis: TextWidthBasisLongestLine})))
	}
	add(fmt.Sprintf("surrogates %t %t %t %s", TextPainterIsHighSurrogate(0xD83D), TextPainterIsLowSurrogate(0xDE00),
		TextPainterIsHighSurrogate(0x41), tpErr(func() any { return TextPainterIsLowSurrogate(0x100000) })))
	mid := PlaceholderDimensions{Size: ui.NewSize(1, 2), Alignment: ui.PlaceholderAlignmentMiddle}
	add("dims " + PlaceholderDimensionsEmpty.String() + " " +
		PlaceholderDimensions{Size: ui.NewSize(1, 2), Alignment: ui.PlaceholderAlignmentBaseline, BaselineOffset: f(3)}.String() + " " +
		PlaceholderDimensions{Size: ui.NewSize(1, 2), Alignment: ui.PlaceholderAlignmentBaseline}.String() + " " +
		strconv.FormatBool(mid.Equal(PlaceholderDimensions{Size: ui.NewSize(1, 2), Alignment: ui.PlaceholderAlignmentMiddle})))
	return out
}

func d2s(v float64) string { return dart.DoubleToString(v) }

// dartJSONString re-encodes Go's JSON string as Dart's jsonEncode does
// (Go escapes <, > and & and U+2028/U+2029; Dart doesn't).
func dartJSONString(s string) string {
	for _, r := range []struct{ from, to string }{{`\u003c`, "<"}, {`\u003e`, ">"}, {`\u0026`, "&"},
		{`\u2028`, "\u2028"}, {`\u2029`, "\u2029"}} {
		s = strings.ReplaceAll(s, r.from, r.to)
	}
	return s
}

// EngineUseTestFontsAndRoboto turns on flutter_tester's test fonts and
// loads Roboto as the Dart side does.
func EngineUseTestFontsAndRoboto(t *testing.T) {
	ui.EngineUseTestFonts()
	ui.ChannelBuffersInstance().SetListener("flutter/system", func([]byte, ui.PlatformMessageResponseCallback) {})
	defer ui.ChannelBuffersInstance().ClearListener("flutter/system")
	data, err := os.ReadFile("../../engine/src/flutter/txt/third_party/fonts/Roboto-Regular.ttf")
	if err != nil {
		t.Fatal(err)
	}
	family := "Roboto"
	ui.LoadFontFromList(data, ui.LoadFontFromListOptions{FontFamily: &family})
	async.Current().FlushMicrotasks()
}

func TestPaintingTextPainterMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_text_painter.json")
	if err != nil {
		t.Fatalf("no reference (%v); run the Flutter side: flutter test test/painting_text_painter_test.dart", err)
	}
	var want []string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	got := paintingTextPainterLines(t)
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
