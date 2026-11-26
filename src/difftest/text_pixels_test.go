package difftest

// Pixel differential test of Canvas.drawParagraph against flutter_tester
// --enable-impeller (flutter/test/impeller/text_pixels_test.dart): the same
// paragraphs drawn into 96×48 pictures and read back as rawRgba. As in
// painting_pixels_test.go, the GPUs differ (Metal here, Vulkan/SwiftShader
// there), so each scene has a measured tolerance.

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

const textPixelWidth, textPixelHeight = 96, 48

func textPara(text string, style *ParagraphStyle, textStyle *TextStyle, width float64, more func(b ParagraphBuilder)) Paragraph {
	if style == nil {
		style = NewParagraphStyle()
	}
	if textStyle == nil {
		textStyle = NewTextStyle(TextStyleOptions{FontSize: ptr(12.0), Color: ptr(NewColor(0xFF000000))})
	}
	b := NewParagraphBuilder(style)
	b.PushStyle(textStyle)
	b.AddText(text)
	if more != nil {
		more(b)
	}
	p := b.Build()
	p.Layout(NewParagraphConstraints(width))
	return p
}

func colorPaint(c int) *Paint {
	p := NewPaint()
	p.SetColor(NewColor(c))
	return p
}

func textPixelScenes() map[string]func(c Canvas) {
	const w = textPixelWidth - 8.0
	return map[string]func(c Canvas){
		"test_font": func(c Canvas) { c.DrawParagraph(textPara("Hello", nil, nil, w, nil), NewOffset(4, 4)) },
		"test_font.fractional": func(c Canvas) {
			c.DrawParagraph(textPara("Hi there", nil, nil, w, nil), NewOffset(4.3, 6.6))
		},
		"test_font.wrap": func(c Canvas) {
			c.DrawParagraph(textPara("ab cd ef gh", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(10.0),
				Color: ptr(NewColor(0xFF2040C0))}), 40, nil), NewOffset(2, 2))
		},
		"colors": func(c Canvas) {
			c.DrawParagraph(textPara("red", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(12.0),
				Color: ptr(NewColor(0xFFFF0000))}), w, func(b ParagraphBuilder) {
				b.PushStyle(NewTextStyle(TextStyleOptions{Color: ptr(NewColor(0x8000AA00)), FontSize: ptr(16.0)}))
				b.AddText("green")
				b.Pop()
			}), NewOffset(2, 8))
		},
		"decorations": func(c Canvas) {
			c.DrawParagraph(textPara("deco", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(14.0),
				Color:               ptr(NewColor(0xFF000000)),
				Decoration:          ptr(TextDecorationCombine([]TextDecoration{TextDecorationUnderline, TextDecorationLineThrough})),
				DecorationColor:     ptr(NewColor(0xFFFF00FF)),
				DecorationThickness: ptr(2.0)}), w, nil), NewOffset(4, 10))
		},
		"decorations.wavy": func(c Canvas) {
			c.DrawParagraph(textPara("wavy", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(14.0),
				Color: ptr(NewColor(0xFF000000)), Decoration: ptr(TextDecorationOverline),
				DecorationStyle: ptr(TextDecorationStyleWavy), DecorationColor: ptr(NewColor(0xFF0080FF))}), w, nil),
				NewOffset(4, 10))
		},
		"background": func(c Canvas) {
			c.DrawParagraph(textPara("bg", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(16.0),
				Background: colorPaint(0xFFFFCC00)}), w, nil), NewOffset(6, 6))
		},
		"shadow": func(c Canvas) {
			c.DrawParagraph(textPara("sh", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(16.0),
				Color:   ptr(NewColor(0xFF000000)),
				Shadows: []Shadow{NewShadow(ShadowOptions{Color: ptr(NewColor(0xFFFF0000)), Offset: NewOffset(3, 2)})}}),
				w, nil), NewOffset(6, 6))
		},
		"foreground.gradient": func(c Canvas) {
			fg := NewPaint()
			fg.SetShader(NewGradientLinear(OffsetZero, NewOffset(72, 0), []Color{NewColor(0xFFFF0000), NewColor(0xFF0000FF)}))
			c.DrawParagraph(textPara("grad", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(18.0), Foreground: fg}), w, nil),
				NewOffset(4, 4))
		},
		"foreground.stroke": func(c Canvas) {
			fg := NewPaint()
			fg.SetStyle(PaintingStyleStroke)
			fg.SetStrokeWidth(2)
			fg.SetColor(NewColor(0xFF008000))
			c.DrawParagraph(textPara("st", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(24.0), Foreground: fg}), w, nil),
				NewOffset(4, 4))
		},
		"transform": func(c Canvas) {
			c.Translate(20, 4)
			c.Rotate(0.3)
			c.Scale(1.5, 1)
			c.DrawParagraph(textPara("rot", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(10.0),
				Color: ptr(NewColor(0xFF000000))}), w, nil), OffsetZero)
		},
		"rtl_ellipsis": func(c Canvas) {
			c.DrawParagraph(textPara("שלום עולם טוב", NewParagraphStyle(ParagraphStyleOptions{
				TextDirection: ptr(TextDirectionRtl), MaxLines: ptr(1), Ellipsis: ptr("...")}),
				NewTextStyle(TextStyleOptions{FontSize: ptr(8.0), Color: ptr(NewColor(0xFF000000))}), 60, nil), NewOffset(2, 2))
		},
		"roboto": func(c Canvas) {
			c.DrawParagraph(textPara("Agile", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(20.0), FontFamily: ptr("Roboto"),
				Color: ptr(NewColor(0xFF000000))}), w, nil), NewOffset(4, 4))
		},
		"roboto.small": func(c Canvas) {
			c.DrawParagraph(textPara("Small text, Roboto at 9px", nil, NewTextStyle(TextStyleOptions{FontSize: ptr(9.0),
				FontFamily: ptr("Roboto"), Color: ptr(NewColor(0xFF203040))}), w, nil), NewOffset(2, 2))
		},
	}
}

// textPixelTolerance is the measured difference per scene (Metal vs
// SwiftShader); scenes not listed must be within 2 everywhere (and are
// bit-exact today: glyphs, colors, shadows, gradient and stroke
// foregrounds, Roboto). The decorations differ on one anti-aliased row
// only (the line's fractional edge: one MSAA sample, 64, as for lines in
// painting_pixels_test.go); rotated glyphs differ by ≤13 on their edges.
var textPixelTolerance = map[string]struct{ max, over2 int }{
	"decorations":      {64, 56},
	"decorations.wavy": {33, 56},
	"transform":        {13, 71},
}

func TestTextPixelsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/text_pixels.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if EngineImpellerContext() == nil {
		t.Fatal("no Metal device: the pixel difftest needs one")
	}
	EngineUseTestFonts()
	loadTextFonts(t)
	scenes := textPixelScenes()
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
		if err != nil || len(want) != textPixelWidth*textPixelHeight*4 {
			t.Errorf("%s: bad reference", name)
			continue
		}
		rec := NewPictureRecorder()
		canvas := NewCanvas(rec)
		canvas.DrawColor(NewColor(0xFFFFFFFF), BlendModeSrc)
		scenes[name](canvas)
		got := awaitF(t, awaitF(t, rec.EndRecording().ToImage(textPixelWidth, textPixelHeight)).ToByteData())
		maxDiff, over2, over16 := 0, 0, 0
		for i := 0; i < len(want); i += 4 {
			d := 0
			for k := 0; k < 4; k++ {
				d = max(d, absInt(int(got[i+k])-int(want[i+k])))
			}
			maxDiff = max(maxDiff, d)
			if d > 2 {
				over2++
			}
			if d > 16 {
				over16++
			}
		}
		tol, ok := textPixelTolerance[name]
		if !ok {
			tol.max, tol.over2 = 2, 0
		}
		t.Logf("%-24s max %3d, pixels >2: %4d, >16: %4d", name, maxDiff, over2, over16)
		if maxDiff > tol.max || over2 > tol.over2 {
			t.Errorf("%s: max channel difference %d (%d pixels > 2), tolerance %+v", name, maxDiff, over2, tol)
		}
	}
}
