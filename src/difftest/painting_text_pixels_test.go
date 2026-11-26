package difftest

// Pixel differential test of painting's TextPainter and the Flutter logo's
// label against Flutter's own Impeller renderer (flutter_tester
// --enable-impeller): the same 128×64 scenes as
// flutter/test/impeller/painting_text_pixels_test.dart, compared with the
// per-scene tolerance scheme of the other pixel difftests.

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
)

const ptpWidth, ptpHeight = 128, 64

type ptpPaintOptions struct {
	align       *ui.TextAlign
	dir         ui.TextDirection
	maxWidth    float64
	maxLines    *int
	ellipsis    *string
	strut       *StrutStyle
	scaler      TextScaler
	placeholder []PlaceholderDimensions
	offset      *ui.Offset
}

func ptpPainted(text InlineSpan, o ptpPaintOptions) func(c ui.Canvas) {
	return func(c ui.Canvas) {
		if o.maxWidth == 0 {
			o.maxWidth = 120
		}
		if o.scaler == nil {
			o.scaler = TextScalerNoScaling
		}
		offset := ui.NewOffset(4, 4)
		if o.offset != nil {
			offset = *o.offset
		}
		dir := o.dir
		p := NewTextPainter(TextPainterOptions{Text: text, TextAlign: o.align, TextDirection: &dir, MaxLines: o.maxLines,
			Ellipsis: o.ellipsis, StrutStyle: o.strut, TextScaler: o.scaler})
		if o.placeholder != nil {
			p.SetPlaceholderDimensions(o.placeholder)
		}
		p.Layout(TextPainterLayoutOptions{MaxWidth: &o.maxWidth})
		p.Paint(c, offset)
		p.Dispose()
	}
}

// ptpOnEdge reports whether the reference pixel (x, y) is on an edge: some
// channel spans more than 16 over its 3x3 neighbourhood.
func ptpOnEdge(img []byte, x, y int) bool {
	for k := 0; k < 4; k++ {
		lo, hi := 255, 0
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				xx, yy := x+dx, y+dy
				if xx < 0 || yy < 0 || xx >= ptpWidth || yy >= ptpHeight {
					continue
				}
				v := int(img[(yy*ptpWidth+xx)*4+k])
				lo, hi = min(lo, v), max(hi, v)
			}
		}
		if hi-lo > 16 {
			return true
		}
	}
	return false
}

func ptpLogo(d *FlutterLogoDecoration, o ui.Offset, s ui.Size) func(c ui.Canvas) {
	return func(c ui.Canvas) { d.CreateBoxPainter(nil).Paint(c, o, ImageConfiguration{Size: &s}) }
}

func paintingTextPixelScenes() map[string]func(c ui.Canvas) {
	s := func(v string) *string { return &v }
	f := func(v float64) *float64 { return &v }
	c := func(v int) *ui.Color { x := ui.NewColor(v); return &x }
	span := func(o TextSpanOptions) *TextSpan { return NewTextSpan(o) }
	black := NewTextStyle(TextStyleOptions{FontSize: f(10), Color: c(0xFF000000)})
	center, right := ui.TextAlignCenter, ui.TextAlignRight
	one, force := 1, true
	under := ui.TextDecorationUnderline
	off := func(x, y float64) *ui.Offset { o := ui.NewOffset(x, y); return &o }
	gap := func() InlineSpan { return newTPPlaceholder() }
	horizontal := NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleHorizontal})
	stacked := NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleStacked})
	return map[string]func(c ui.Canvas){
		"plain": ptpPainted(span(TextSpanOptions{Text: s("Hello world"), Style: black}),
			ptpPaintOptions{dir: ui.TextDirectionLtr, offset: off(4.3, 6.6)}),
		"wrap.center": ptpPainted(span(TextSpanOptions{Text: s("one two three four five"), Style: black}),
			ptpPaintOptions{dir: ui.TextDirectionLtr, align: &center, maxWidth: 70}),
		"rtl.right": ptpPainted(span(TextSpanOptions{Text: s("שלום עולם abc"), Style: black}),
			ptpPaintOptions{dir: ui.TextDirectionRtl, align: &right}),
		"ellipsis": ptpPainted(span(TextSpanOptions{Text: s("the quick brown fox jumps over"), Style: black}),
			ptpPaintOptions{dir: ui.TextDirectionLtr, maxLines: &one, ellipsis: s("…"), maxWidth: 90}),
		"spans": ptpPainted(span(TextSpanOptions{Style: black, Children: []InlineSpan{
			span(TextSpanOptions{Text: s("red "), Style: NewTextStyle(TextStyleOptions{Color: c(0xFFFF0000), FontSize: f(14)})}),
			span(TextSpanOptions{Text: s("bg"), Style: NewTextStyle(TextStyleOptions{BackgroundColor: c(0xFF80C0FF)})}),
			span(TextSpanOptions{Text: s(" under"), Style: NewTextStyle(TextStyleOptions{Decoration: &under, DecorationColor: c(0xFF00A000)})}),
			span(TextSpanOptions{Text: s(" big"), Style: NewTextStyle(TextStyleOptions{FontSize: f(18), Color: c(0x80000080)})}),
		}}), ptpPaintOptions{dir: ui.TextDirectionLtr}),
		"strut": ptpPainted(span(TextSpanOptions{Text: s("strut a\nb c"), Style: black}), ptpPaintOptions{dir: ui.TextDirectionLtr,
			strut: NewStrutStyle(StrutStyleOptions{FontSize: f(14), Height: f(1.4), ForceStrutHeight: &force})}),
		"roboto.scaled": ptpPainted(span(TextSpanOptions{Text: s("Roboto text"),
			Style: NewTextStyle(TextStyleOptions{FontFamily: s("Roboto"), FontSize: f(12), Color: c(0xFF202020)})}),
			ptpPaintOptions{dir: ui.TextDirectionLtr, scaler: NewTextScalerLinear(1.5)}),
		"placeholders": ptpPainted(span(TextSpanOptions{Style: black, Children: []InlineSpan{
			span(TextSpanOptions{Text: s("a")}), gap(), span(TextSpanOptions{Text: s("b")}), gap(), span(TextSpanOptions{Text: s("c")}),
		}}), ptpPaintOptions{dir: ui.TextDirectionLtr, placeholder: []PlaceholderDimensions{
			{Size: ui.NewSize(20, 8), Alignment: ui.PlaceholderAlignmentMiddle},
			{Size: ui.NewSize(6, 20), Alignment: ui.PlaceholderAlignmentBottom},
		}}),
		"logo.horizontal": ptpLogo(horizontal, ui.OffsetZero, ui.NewSize(128, 64)),
		"logo.stacked": ptpLogo(NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleStacked,
			TextColor: c(0xFF8000C0)}), ui.NewOffset(32, 0), ui.NewSize(64, 64)),
		"logo.horizontal.transition": ptpLogo(FlutterLogoDecorationLerp(NewFlutterLogoDecoration(), horizontal, 0.5),
			ui.OffsetZero, ui.NewSize(128, 64)),
		"logo.stacked.transition": ptpLogo(FlutterLogoDecorationLerp(NewFlutterLogoDecoration(), stacked, 0.4),
			ui.NewOffset(32, 0), ui.NewSize(64, 64)),
		"logo.horizontal.margin.fading": ptpLogo(FlutterLogoDecorationLerp(nil, NewFlutterLogoDecoration(FlutterLogoDecorationOptions{
			Style: FlutterLogoStyleHorizontal, Margin: EdgeInsets{Left: 6, Top: 4, Right: 2, Bottom: 8}}), 0.7),
			ui.NewOffset(2, 3), ui.NewSize(120, 56)),
	}
}

// paintingTextPixelTolerance is each scene's allowed maximum per-channel
// difference and how many pixels (of 8192) may differ by more than 2;
// scenes not listed allow max 2 and none over 2 (Apple Metal here vs
// SwiftShader Vulkan in flutter_tester). Every text scene is bit-exact,
// and so is the label in each logo scene: the logo scenes listed differ
// only on the anti-aliased edges of the mark's beams at these
// non-integer scales (one MSAA sample of four, 64, or a fraction of it
// under the fade layer), as the mark-only logo and the other paths in
// painting_pixels do; every pixel differing by more than 2 must lie on an
// edge of the reference (its 3x3 neighbourhood spans more than 16).
var paintingTextPixelTolerance = map[string]struct{ max, over2 int }{
	"logo.stacked":                  {64, 11},
	"logo.stacked.transition":       {43, 64},
	"logo.horizontal.transition":    {9, 1},
	"logo.horizontal.margin.fading": {30, 15},
}

func TestPaintingTextPixelsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_text_pixels.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if ui.EngineImpellerContext() == nil {
		t.Fatal("no Metal device: the pixel difftest needs one")
	}
	EngineUseTestFontsAndRoboto(t)
	scenes := paintingTextPixelScenes()
	if len(ref) != len(scenes) {
		t.Errorf("reference has %d scenes, Go has %d", len(ref), len(scenes))
	}
	names := make([]string, 0, len(scenes))
	for n := range scenes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		want, err := base64.StdEncoding.DecodeString(ref[name])
		if err != nil || len(want) != ptpWidth*ptpHeight*4 {
			t.Errorf("%s: bad reference", name)
			continue
		}
		rec := ui.NewPictureRecorder()
		canvas := ui.NewCanvas(rec)
		canvas.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrc)
		scenes[name](canvas)
		got := awaitF(t, awaitF(t, rec.EndRecording().ToImage(ptpWidth, ptpHeight)).ToByteData())
		if os.Getenv("PTEXT_DUMP") != "" {
			os.WriteFile(os.Getenv("PTEXT_DUMP")+"/"+name+".rgba", got, 0o644)
		}
		maxDiff, over2, over16 := 0, 0, 0
		for i := 0; i < len(want); i += 4 {
			d := 0
			for k := 0; k < 4; k++ {
				d = max(d, absInt(int(got[i+k])-int(want[i+k])))
			}
			maxDiff = max(maxDiff, d)
			if d > 2 {
				over2++
				if px := i / 4; !ptpOnEdge(want, px%ptpWidth, px/ptpWidth) {
					t.Errorf("%s: pixel (%d, %d) differs by %d off an edge", name, px%ptpWidth, px/ptpWidth, d)
				}
			}
			if d > 16 {
				over16++
			}
		}
		tol, ok := paintingTextPixelTolerance[name]
		if !ok {
			tol.max, tol.over2 = 2, 0
		}
		t.Logf("%-30s max %3d, pixels >2: %4d, >16: %4d", name, maxDiff, over2, over16)
		if maxDiff > tol.max || over2 > tol.over2 {
			t.Errorf("%s: max channel difference %d (%d pixels > 2), tolerance %+v", name, maxDiff, over2, tol)
		}
	}
}
