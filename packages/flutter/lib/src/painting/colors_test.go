package painting

// Port of packages/flutter/test/painting/colors_test.dart (Flutter 3.50.1).
// Skipped: "MaterialColor swatch comparison" (material's MaterialColor and
// Colors are not ported). hashCode checks are dropped (Go values compare
// with ==). within<T>(distance:) uses flutter_test's distance functions
// (max component difference; hue scaled by 1/360; Color by 8-bit
// components), reproduced below.

import (
	"math"
	"slices"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

const doubleColorPrecision = 0.01

func hsvDistance(a, b HSVColor) float64 {
	d := math.Max(math.Abs(a.Saturation-b.Saturation), math.Abs(a.Value-b.Value))
	d = math.Max(d, math.Abs((a.Hue-b.Hue)/360.0))
	return math.Max(d, math.Abs(a.Alpha-b.Alpha))
}

func hslDistance(a, b HSLColor) float64 {
	d := math.Max(math.Abs(a.Saturation-b.Saturation), math.Abs(a.Lightness-b.Lightness))
	d = math.Max(d, math.Abs((a.Hue-b.Hue)/360.0))
	return math.Max(d, math.Abs(a.Alpha-b.Alpha))
}

func colorDistance(a, b ui.Color) float64 {
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}
	d := max(abs(a.Red()-b.Red()), abs(a.Green()-b.Green()), abs(a.Blue()-b.Blue()), abs(a.Alpha()-b.Alpha()))
	return float64(d)
}

func near(a, b float64) bool { return math.Abs(a-b) <= precisionErrorTolerance }

const precisionErrorTolerance = 1e-10

func colorsEqual(t *testing.T, got []ui.Color, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d colors, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != ui.NewColor(want[i]) {
			t.Errorf("[%d] got %v, want %#x", i, got[i], want[i])
		}
	}
}

func TestHSVColorControl(t *testing.T) {
	color := NewHSVColorFromAHSV(0.7, 28.0, 0.3, 0.6)
	if strings.Contains(color.String(), "\n") || color.String() != "HSVColor(0.7, 28.0, 0.3, 0.6)" {
		t.Fatal(color.String())
	}
	if color.WithAlpha(0.8) != NewHSVColorFromAHSV(0.8, 28.0, 0.3, 0.6) ||
		color.WithHue(123.0) != NewHSVColorFromAHSV(0.7, 123.0, 0.3, 0.6) ||
		color.WithSaturation(0.9) != NewHSVColorFromAHSV(0.7, 28.0, 0.9, 0.6) ||
		color.WithValue(0.1) != NewHSVColorFromAHSV(0.7, 28.0, 0.3, 0.1) {
		t.Fatal("with*")
	}
	if color.ToColor() != ui.NewColor(0xb399816b) {
		t.Fatal(color.ToColor())
	}
	other := NewHSVColorFromAHSV(0.3, 128.0, 0.7, 0.2)
	result := HSVColorLerp(&color, &other, 0.25)
	if !near(result.Alpha, 0.6) || !near(result.Hue, 53.0) || !(result.Saturation > 0.3999) ||
		!(result.Saturation < 0.4001) || !near(result.Value, 0.5) {
		t.Fatal(result)
	}
}

func TestHSVColorHueSweep(t *testing.T) {
	var output []ui.Color
	for hue := 0.0; hue <= 360.0; hue += 36.0 {
		hsv := NewHSVColorFromAHSV(1.0, hue, 1.0, 1.0)
		color := hsv.ToColor()
		output = append(output, color)
		if hue != 360.0 && hsvDistance(NewHSVColorFromColor(color), hsv) > doubleColorPrecision {
			t.Errorf("round trip %v", hsv)
		}
	}
	colorsEqual(t, output, []int{0xffff0000, 0xffff9900, 0xffccff00, 0xff33ff00, 0xff00ff66, 0xff00ffff,
		0xff0066ff, 0xff3300ff, 0xffcc00ff, 0xffff0099, 0xffff0000})
}

func TestHSVColorSaturationSweep(t *testing.T) {
	var output []ui.Color
	for saturation := 0.0; saturation < 1.0; saturation += 0.1 {
		hsv := NewHSVColorFromAHSV(1.0, 0.0, saturation, 1.0)
		color := hsv.ToColor()
		output = append(output, color)
		if hsvDistance(NewHSVColorFromColor(color), hsv) > doubleColorPrecision {
			t.Errorf("round trip %v", hsv)
		}
	}
	colorsEqual(t, output, []int{0xffffffff, 0xffffe6e6, 0xffffcccc, 0xffffb3b3, 0xffff9999, 0xffff8080,
		0xffff6666, 0xffff4d4d, 0xffff3333, 0xffff1a1a, 0xffff0000})
}

func TestHSVColorValueSweep(t *testing.T) {
	var output []ui.Color
	for value := 0.0; value < 1.0; value += 0.1 {
		hsv := NewHSVColorFromAHSV(1.0, 0.0, 1.0, value)
		color := hsv.ToColor()
		output = append(output, color)
		if value >= doubleColorPrecision && value <= 1.0-doubleColorPrecision &&
			hsvDistance(NewHSVColorFromColor(color), hsv) > doubleColorPrecision {
			t.Errorf("round trip %v", hsv)
		}
	}
	colorsEqual(t, output, []int{0xff000000, 0xff1a0000, 0xff330000, 0xff4d0000, 0xff660000, 0xff800000,
		0xff990000, 0xffb30000, 0xffcc0000, 0xffe50000, 0xffff0000})
}

func TestHSVColorLerpIdentical(t *testing.T) {
	if HSVColorLerp(nil, nil, 0) != nil {
		t.Fatal("null")
	}
	color := NewHSVColorFromAHSV(1.0, 0.0, 1.0, 1.0)
	if HSVColorLerp(&color, &color, 0.5) != &color {
		t.Fatal("identical")
	}
}

func TestHSVColorLerpNull(t *testing.T) {
	c := NewHSVColorFromAHSV(0.8, 10, 0.5, 0.5)
	if r := HSVColorLerp(nil, &c, 0.25); *r != c.WithAlpha(0.2) {
		t.Fatal(r)
	}
	if r := HSVColorLerp(&c, nil, 0.25); *r != c.WithAlpha(c.Alpha*(1-0.25)) {
		t.Fatal(r)
	}
}

func lerpHSVColors(start, end HSVColor, from, to float64) []ui.Color {
	var output []ui.Color
	for t := from; t < to; t += 0.1 {
		output = append(output, HSVColorLerp(&start, &end, t).ToColor())
	}
	return output
}

func TestHSVColorLerpsHue(t *testing.T) {
	colorsEqual(t, lerpHSVColors(NewHSVColorFromAHSV(1.0, 0.0, 1.0, 1.0), NewHSVColorFromAHSV(1.0, 180.0, 1.0, 1.0), -0.5, 1.5),
		[]int{0xff8000ff, 0xffcc00ff, 0xffff00e6, 0xffff0099, 0xffff004c, 0xffff0000, 0xffff4c00, 0xffff9900,
			0xffffe600, 0xffccff00, 0xff80ff00, 0xff33ff00, 0xff00ff19, 0xff00ff66, 0xff00ffb2, 0xff00ffff,
			0xff00b3ff, 0xff0066ff, 0xff001aff, 0xff3300ff})
}

type hueCase struct{ start, end, t, want float64 }

var shortestArcCases = []hueCase{
	{300, 60, 0.25, 330}, {300, 60, 0.5, 0}, {300, 60, 0.75, 30},
	{60, 300, 0.25, 30}, {60, 300, 0.5, 0}, {60, 300, 0.75, 330},
	{350, 10, 0.5, 0}, {10, 350, 0.5, 0},
	{20, 200, 0.5, 110}, {0, 90, 0.5, 45},
	{0, 360, 0.5, 0},
	{300, 60, 1.5, 120}, {300, 60, -0.5, 240},
	// Antipodal hues lerp reversibly.
	{0, 180, 0.5, 90}, {180, 0, 0.5, 90}, {60, 240, 0.75, 195}, {240, 60, 0.25, 195},
}

func TestHSVColorLerpsHueShortestArc(t *testing.T) {
	for _, c := range shortestArcCases {
		a, b := NewHSVColorFromAHSV(1.0, c.start, 1.0, 1.0), NewHSVColorFromAHSV(1.0, c.end, 1.0, 1.0)
		if got := HSVColorLerp(&a, &b, c.t).Hue; !near(got, c.want) {
			t.Errorf("%v: %v", c, got)
		}
	}
}

func TestHSVColorLerpsSaturation(t *testing.T) {
	colorsEqual(t, lerpHSVColors(NewHSVColorFromAHSV(1.0, 0.0, 0.0, 1.0), NewHSVColorFromAHSV(1.0, 0.0, 1.0, 1.0), -0.1, 1.1),
		[]int{0xffffffff, 0xffffffff, 0xffffe6e6, 0xffffcccc, 0xffffb3b3, 0xffff9999, 0xffff8080, 0xffff6666,
			0xffff4d4d, 0xffff3333, 0xffff1a1a, 0xffff0000, 0xffff0000})
}

func TestHSVColorLerpsValue(t *testing.T) {
	colorsEqual(t, lerpHSVColors(NewHSVColorFromAHSV(1.0, 0.0, 1.0, 0.0), NewHSVColorFromAHSV(1.0, 0.0, 1.0, 1.0), -0.1, 1.1),
		[]int{0xff000000, 0xff000000, 0xff1a0000, 0xff330000, 0xff4d0000, 0xff660000, 0xff800000, 0xff990000,
			0xffb30000, 0xffcc0000, 0xffe50000, 0xffff0000, 0xffff0000})
}

func TestHSLColorControl(t *testing.T) {
	color := NewHSLColorFromAHSL(0.7, 28.0, 0.3, 0.6)
	if color.String() != "HSLColor(0.7, 28.0, 0.3, 0.6)" {
		t.Fatal(color.String())
	}
	if color.WithAlpha(0.8) != NewHSLColorFromAHSL(0.8, 28.0, 0.3, 0.6) ||
		color.WithHue(123.0) != NewHSLColorFromAHSL(0.7, 123.0, 0.3, 0.6) ||
		color.WithSaturation(0.9) != NewHSLColorFromAHSL(0.7, 28.0, 0.9, 0.6) ||
		color.WithLightness(0.1) != NewHSLColorFromAHSL(0.7, 28.0, 0.3, 0.1) {
		t.Fatal("with*")
	}
	if color.ToColor() != ui.NewColor(0xb3b8977a) {
		t.Fatal(color.ToColor())
	}
	other := NewHSLColorFromAHSL(0.3, 128.0, 0.7, 0.2)
	result := HSLColorLerp(&color, &other, 0.25)
	if !near(result.Alpha, 0.6) || !near(result.Hue, 53.0) || !(result.Saturation > 0.3999) ||
		!(result.Saturation < 0.4001) || !near(result.Lightness, 0.5) {
		t.Fatal(result)
	}
}

func TestHSLColorHueSweep(t *testing.T) {
	var output []ui.Color
	for hue := 0.0; hue <= 360.0; hue += 36.0 {
		hsl := NewHSLColorFromAHSL(1.0, hue, 0.5, 0.5)
		color := hsl.ToColor()
		output = append(output, color)
		if hue != 360.0 && hslDistance(NewHSLColorFromColor(color), hsl) > doubleColorPrecision {
			t.Errorf("round trip %v", hsl)
		}
	}
	colorsEqual(t, output, []int{0xffbf4040, 0xffbf8c40, 0xffa6bf40, 0xff59bf40, 0xff40bf73, 0xff40bfbf,
		0xff4073bf, 0xff5940bf, 0xffa640bf, 0xffbf408c, 0xffbf4040})
}

func TestHSLColorSaturationSweep(t *testing.T) {
	var output []ui.Color
	for saturation := 0.0; saturation < 1.0; saturation += 0.1 {
		hsl := NewHSLColorFromAHSL(1.0, 0.0, saturation, 0.5)
		color := hsl.ToColor()
		output = append(output, color)
		if hslDistance(NewHSLColorFromColor(color), hsl) > doubleColorPrecision {
			t.Errorf("round trip %v", hsl)
		}
	}
	colorsEqual(t, output, []int{0xff808080, 0xff8c7373, 0xff996666, 0xffa65959, 0xffb34d4d, 0xffbf4040,
		0xffcc3333, 0xffd92626, 0xffe51a1a, 0xfff20d0d, 0xffff0000})
}

func TestHSLColorLightnessSweep(t *testing.T) {
	var output []ui.Color
	for lightness := 0.0; lightness < 1.0; lightness += 0.1 {
		hsl := NewHSLColorFromAHSL(1.0, 0.0, 0.5, lightness)
		color := hsl.ToColor()
		output = append(output, color)
		if lightness >= doubleColorPrecision && lightness <= 1.0-doubleColorPrecision &&
			hslDistance(NewHSLColorFromColor(color), hsl) > doubleColorPrecision {
			t.Errorf("round trip %v", hsl)
		}
	}
	colorsEqual(t, output, []int{0xff000000, 0xff260d0d, 0xff4d1a1a, 0xff732626, 0xff993333, 0xffbf4040,
		0xffcc6666, 0xffd98c8c, 0xffe6b3b3, 0xfff2d9d9, 0xffffffff})
}

func TestHSLColorFromColor(t *testing.T) {
	pink := NewHSLColorFromColor(ui.NewColorFromARGB(255, 255, 51, 152))
	if pink.Alpha != 1.0 || math.Abs(pink.Hue-330) > .3 || pink.Saturation != 1.0 || math.Abs(pink.Lightness-0.6) > doubleColorPrecision {
		t.Error("pink", pink)
	}
	if w := NewHSLColorFromColor(ui.NewColor(0xffffffff)); w != (HSLColor{1, 0, 0, 1}) {
		t.Error("white", w)
	}
	if b := NewHSLColorFromColor(ui.NewColor(0xff000000)); b != (HSLColor{1, 0, 0, 0}) {
		t.Error("black", b)
	}
	if g := NewHSLColorFromColor(ui.NewColor(0xff808080)); g.Alpha != 1 || g.Hue != 0 || g.Saturation != 0 ||
		math.Abs(g.Lightness-0.5) > doubleColorPrecision {
		t.Error("gray", g)
	}
}

func TestHSLColorLerpIdentical(t *testing.T) {
	if HSLColorLerp(nil, nil, 0) != nil {
		t.Fatal("null")
	}
	color := NewHSLColorFromAHSL(1.0, 0.0, 0.5, 0.5)
	if HSLColorLerp(&color, &color, 0.5) != &color {
		t.Fatal("identical")
	}
}

func lerpHSLColors(start, end HSLColor, from, to float64) []ui.Color {
	var output []ui.Color
	for t := from; t < to; t += 0.1 {
		output = append(output, HSLColorLerp(&start, &end, t).ToColor())
	}
	return output
}

func TestHSLColorLerpsHue(t *testing.T) {
	colorsEqual(t, lerpHSLColors(NewHSLColorFromAHSL(1.0, 0.0, 0.5, 0.5), NewHSLColorFromAHSL(1.0, 180.0, 0.5, 0.5), -0.5, 1.5),
		[]int{0xff8040bf, 0xffa640bf, 0xffbf40b3, 0xffbf408c, 0xffbf4066, 0xffbf4040, 0xffbf6640, 0xffbf8c40,
			0xffbfb340, 0xffa6bf40, 0xff80bf40, 0xff59bf40, 0xff40bf4c, 0xff40bf73, 0xff40bf99, 0xff40bfbf,
			0xff4099bf, 0xff4073bf, 0xff404dbf, 0xff5940bf})
}

func TestHSLColorLerpsHueShortestArc(t *testing.T) {
	for _, c := range shortestArcCases {
		a, b := NewHSLColorFromAHSL(1.0, c.start, 0.5, 0.5), NewHSLColorFromAHSL(1.0, c.end, 0.5, 0.5)
		if got := HSLColorLerp(&a, &b, c.t).Hue; !near(got, c.want) {
			t.Errorf("%v: %v", c, got)
		}
	}
}

func TestHSLColorLerpsSaturation(t *testing.T) {
	colorsEqual(t, lerpHSLColors(NewHSLColorFromAHSL(1.0, 0.0, 0.0, 0.5), NewHSLColorFromAHSL(1.0, 0.0, 1.0, 0.5), -0.1, 1.1),
		[]int{0xff808080, 0xff808080, 0xff8c7373, 0xff996666, 0xffa65959, 0xffb34d4d, 0xffbf4040, 0xffcc3333,
			0xffd92626, 0xffe51a1a, 0xfff20d0d, 0xffff0000, 0xffff0000})
}

func TestHSLColorLerpsLightness(t *testing.T) {
	colorsEqual(t, lerpHSLColors(NewHSLColorFromAHSL(1.0, 0.0, 0.5, 0.0), NewHSLColorFromAHSL(1.0, 0.0, 0.5, 1.0), -0.1, 1.1),
		[]int{0xff000000, 0xff000000, 0xff260d0d, 0xff4d1a1a, 0xff732626, 0xff993333, 0xffbf4040, 0xffcc6666,
			0xffd98c8c, 0xffe6b3b3, 0xfff2d9d9, 0xffffffff, 0xffffffff})
}

func TestWikipediaExamplesTable(t *testing.T) {
	type row struct {
		rgb                            int
		r, g, b, hue, v, l, sHSV, sHSL float64
		name                           string
	}
	for _, c := range []row{
		{0xFFFFFFFF, 1.0, 1.0, 1.0, 0.0, 1.0, 1.0, 0.0, 0.0, "white"},
		{0xFF808080, 0.5, 0.5, 0.5, 0.0, 0.5, 0.5, 0.0, 0.0, "gray"},
		{0xFF000000, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, "black"},
		{0xFFFF0000, 1.0, 0.0, 0.0, 0.0, 1.0, 0.5, 1.0, 1.0, "red"},
		{0xFFBFBF00, .75, .75, 0.0, 60, .75, .375, 1.0, 1.0, "lime"},
		{0xFF008000, 0.0, 0.5, 0.0, 120, 0.5, .25, 1.0, 1.0, "green"},
		{0xFF80FFFF, 0.5, 1.0, 1.0, 180, 1.0, 0.75, 0.5, 1, "cyan"},
		{0xFF8080FF, 0.5, 0.5, 1.0, 240, 1.0, 0.75, 0.5, 1, "light purple"},
		{0xFFBF40BF, 0.75, 0.25, 0.75, 300, .75, .5, 2.0 / 3, .5, "mute magenta"},
	} {
		color := ui.NewColorFrom(ui.ColorFromOptions{Alpha: 1.0, Red: c.r, Green: c.g, Blue: c.b})
		intColor := ui.NewColor(c.rgb)
		if math.Abs(intColor.R-c.r) > doubleColorPrecision || math.Abs(intColor.G-c.g) > doubleColorPrecision ||
			math.Abs(intColor.B-c.b) > doubleColorPrecision {
			t.Errorf("%s: components", c.name)
		}
		hsv := NewHSVColorFromAHSV(1.0, c.hue, c.sHSV, c.v)
		hsl := NewHSLColorFromAHSL(1.0, c.hue, c.sHSL, c.l)
		if colorDistance(color, intColor) > doubleColorPrecision || colorDistance(hsv.ToColor(), color) > doubleColorPrecision ||
			colorDistance(hsl.ToColor(), color) > doubleColorPrecision {
			t.Errorf("%s: colors", c.name)
		}
		if hsvDistance(NewHSVColorFromColor(color), hsv) > doubleColorPrecision ||
			hslDistance(NewHSLColorFromColor(color), hsl) > doubleColorPrecision {
			t.Errorf("%s: fromColor", c.name)
		}
	}
}

func TestDisplayP3ToExtendedSRGB(t *testing.T) {
	ext := ui.ColorSpaceExtendedSRGB
	p3 := ui.NewColorFrom(ui.ColorFromOptions{Alpha: 1.0, Red: 30 / 255.0, Green: 202 / 255.0, Blue: 211 / 255.0, ColorSpace: ui.ColorSpaceDisplayP3})
	e := p3.WithValues(ui.ColorWithValuesOptions{ColorSpace: &ext})
	if e.ColorSpace != ext || math.Abs(e.R - -0.376) > 0.01 || math.Abs(e.G-0.807) > 0.01 || math.Abs(e.B-0.837) > 0.01 {
		t.Error(e)
	}
	green := ui.NewColorFrom(ui.ColorFromOptions{Alpha: 1.0, Green: 1.0, ColorSpace: ui.ColorSpaceDisplayP3})
	e = green.WithValues(ui.ColorWithValuesOptions{ColorSpace: &ext})
	if math.Abs(e.R - -0.512) > 0.01 || math.Abs(e.G-1.018) > 0.01 || math.Abs(e.B - -0.311) > 0.01 {
		t.Error(e)
	}
	p3s := ui.ColorSpaceDisplayP3
	srgb := ui.NewColorFrom(ui.ColorFromOptions{Alpha: 1.0, Red: 0.5, Green: 0.3, Blue: 0.8})
	back := srgb.WithValues(ui.ColorWithValuesOptions{ColorSpace: &p3s}).WithValues(ui.ColorWithValuesOptions{ColorSpace: &ext})
	if math.Abs(back.R-srgb.R) > 0.01 || math.Abs(back.G-srgb.G) > 0.01 || math.Abs(back.B-srgb.B) > 0.01 {
		t.Error(back)
	}
}

func greens() []dart.MapEntry[string, ui.Color] {
	return []dart.MapEntry[string, ui.Color]{
		{Key: "2259 C", Value: ui.NewColor(0xFF027223)},
		{Key: "2273 C", Value: ui.NewColor(0xFF257226)},
		{Key: "2426 XGC", Value: ui.NewColor(0xFF00932F)},
		{Key: "7732 XGC", Value: ui.NewColor(0xFF007940)},
	}
}

func TestColorSwatch(t *testing.T) {
	greens1 := NewColorSwatch(0xFF027223, greens())
	greens2 := NewColorSwatch(0xFF027223, greens())
	if !greens1.Equal(greens2) {
		t.Fatal("equal")
	}
	if c := greens1.Get("2259 C"); c == nil || *c != ui.NewColor(0xFF027223) || greens1.Get("nope") != nil {
		t.Fatal(c)
	}
	if greens1.Value() != 0xFF027223 {
		t.Fatal(greens1.Value())
	}
	if !slices.Equal(slices.Collect(greens1.Keys()), slices.Collect(greens2.Keys())) ||
		!slices.Equal(slices.Collect(greens1.Keys()), []string{"2259 C", "2273 C", "2426 XGC", "7732 XGC"}) {
		t.Fatal("keys")
	}
	// Own: a different primary or swatch is unequal; order doesn't matter.
	reordered := greens()
	reordered[0], reordered[1] = reordered[1], reordered[0]
	changed := greens()
	changed[3].Value = ui.NewColor(0xFF007941)
	if !greens1.Equal(NewColorSwatch(0xFF027223, reordered)) || greens1.Equal(NewColorSwatch(0xFF027224, greens())) ||
		greens1.Equal(NewColorSwatch(0xFF027223, changed)) || greens1.Equal(NewColorSwatch(0xFF027223, greens()[:3])) ||
		greens1.Equal(nil) || NewColorSwatch(0xFF027223, greens()[:3]).Equal(greens1) {
		t.Fatal("inequality")
	}
	if got := NewColorSwatch(0xFF027223, greens()).String(); got != "ColorSwatch<String>(primary value: "+ui.NewColor(0xFF027223).String()+")" {
		t.Fatal(got)
	}
}

func swatch1(v int) *ColorSwatch[int] {
	return NewColorSwatch(v, []dart.MapEntry[int, ui.Color]{{Key: 1, Value: ui.NewColor(v)}})
}

func isSameColorSwatch(a, b *ColorSwatch[int]) bool {
	// isSameColorAs: the primary colors' components (and the swatch
	// colors') within 1/255.
	same := func(x, y ui.Color) bool {
		return math.Abs(x.A-y.A) < 1/255.0 && math.Abs(x.R-y.R) < 1/255.0 && math.Abs(x.G-y.G) < 1/255.0 &&
			math.Abs(x.B-y.B) < 1/255.0
	}
	return same(a.Color, b.Color) && same(*a.Get(1), *b.Get(1))
}

func TestColorSwatchLerp(t *testing.T) {
	a, b := swatch1(0x00000000), swatch1(0xFFFFFFFF)
	for _, c := range []struct {
		t    float64
		want int
	}{{0.0, 0x00000000}, {0.5, 0x7F7F7F7F}, {1.0, 0xFFFFFFFF}, {-0.1, 0x00000000}, {1.1, 0xFFFFFFFF}} {
		if got := ColorSwatchLerp(a, b, c.t); !isSameColorSwatch(got, swatch1(c.want)) {
			t.Errorf("%v: %v", c.t, got)
		}
	}
	if ColorSwatchLerp[any](nil, nil, 0) != nil {
		t.Fatal("null")
	}
	if ColorSwatchLerp(a, a, 0.5) != a {
		t.Fatal("identical")
	}
	// Own: a null end fades the other.
	// The primary color goes through Color.value (8 bits per channel).
	if got := ColorSwatchLerp(nil, b, 0.25); got.Get(1).A != 0.25 || got.A != 64/255.0 {
		t.Fatal(got)
	}
	if got := ColorSwatchLerp(b, nil, 0.25); got.Get(1).A != 0.75 || got.A != 191/255.0 {
		t.Fatal(got)
	}
}

func TestColorDiagnosticsPropertyJSON(t *testing.T) {
	property := NewColorProperty("foo", ui.NewColorFromARGB(10, 20, 30, 40))
	vp := property.ToJsonMap(foundation.NewDiagnosticsSerializationDelegate())["valueProperties"].(map[string]any)
	if vp["alpha"] != 10 || vp["red"] != 20 || vp["green"] != 30 || vp["blue"] != 40 {
		t.Fatal(vp)
	}
	property = NewColorProperty[*ui.Color]("foo", nil)
	if _, ok := property.ToJsonMap(foundation.NewDiagnosticsSerializationDelegate())["valueProperties"]; ok {
		t.Fatal("null has valueProperties")
	}
	if got := NewColorProperty("foo", ui.NewColor(0xFF102030)).String(); got != "foo: "+ui.NewColor(0xFF102030).String() {
		t.Fatal(got)
	}
	if got := NewColorProperty[*ui.Color]("foo", nil).String(); got != "foo: null" {
		t.Fatal(got)
	}
}

func TestHSVAssertions(t *testing.T) {
	for _, f := range []func(){
		func() { NewHSVColorFromAHSV(-0.1, 0, 0, 0) }, func() { NewHSVColorFromAHSV(1.1, 0, 0, 0) },
		func() { NewHSVColorFromAHSV(1, -1, 0, 0) }, func() { NewHSVColorFromAHSV(1, 361, 0, 0) },
		func() { NewHSVColorFromAHSV(1, 0, 2, 0) }, func() { NewHSVColorFromAHSV(1, 0, 0, -1) },
		func() { NewHSLColorFromAHSL(1, 0, 0, 1.5) }, func() { NewHSLColorFromAHSL(1, 400, 0, 0) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("no assertion")
				}
			}()
			f()
		}()
	}
}
