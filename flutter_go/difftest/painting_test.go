package difftest

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func comps(x Color) []any { return []any{x.A, x.R, x.G, x.B, float64(x.ColorSpace)} }

func cs(v ColorSpace) *ColorSpace { return &v }

// Same cases as flutter/test/painting_test.dart.
func paintingCases() map[string]any {
	from := func(a, r, g, b float64, s ColorSpace) Color {
		return NewColorFrom(ColorFromOptions{Alpha: a, Red: r, Green: g, Blue: b, ColorSpace: s})
	}
	p3red := from(1, 1, 0, 0, ColorSpaceDisplayP3)
	ext := from(0.8, 1.2, -0.1, 0.5, ColorSpaceExtendedSRGB)
	half := from(0.5, 0.25, 0.75, 0.125, ColorSpaceSRGB)
	with := func(c Color, o ColorWithValuesOptions) []any { return comps(c.WithValues(o)) }
	return map[string]any{
		"toString":    fmt.Sprint(NewColor(0x12345678)),
		"toString.p3": fmt.Sprint(p3red),
		"fromRGBO":    comps(NewColorFromRGBO(10, 300, -1, 0.3)),
		"argb.half":   float64(from(0.5/255, 1.5/255, 2.5/255, 254.5/255, ColorSpaceSRGB).ToARGB32()),
		"alphaFromOpacity": []any{float64(ColorGetAlphaFromOpacity(0.5)), float64(ColorGetAlphaFromOpacity(0.002)),
			float64(ColorGetAlphaFromOpacity(1.5))},
		"withOpacity":          comps(NewColor(0xFF102030).WithOpacity(0.5)),
		"withAlpha":            comps(NewColor(0x80102030).WithAlpha(0x1ff)),
		"withValues.red":       with(half, ColorWithValuesOptions{Red: fp(0.9)}),
		"p3.to.srgb":           with(p3red, ColorWithValuesOptions{ColorSpace: cs(ColorSpaceSRGB)}),
		"p3.to.ext":            with(p3red, ColorWithValuesOptions{ColorSpace: cs(ColorSpaceExtendedSRGB)}),
		"srgb.to.p3":           with(half, ColorWithValuesOptions{ColorSpace: cs(ColorSpaceDisplayP3)}),
		"srgb.to.ext":          with(half, ColorWithValuesOptions{ColorSpace: cs(ColorSpaceExtendedSRGB)}),
		"ext.to.srgb":          with(ext, ColorWithValuesOptions{ColorSpace: cs(ColorSpaceSRGB)}),
		"ext.to.p3":            with(ext, ColorWithValuesOptions{ColorSpace: cs(ColorSpaceDisplayP3)}),
		"ext.values.and.space": with(ext, ColorWithValuesOptions{Alpha: fp(0.3), ColorSpace: cs(ColorSpaceSRGB)}),
		"lerp.mixed":           comps(*ColorLerp(&half, &p3red, 0.3)),
		"lerp.null":            comps(*ColorLerp(nil, &half, 0.4)),
		"lerp.null.b":          comps(*ColorLerp(&half, nil, 0.4)),
		"alphaBlend":           comps(ColorAlphaBlend(NewColor(0x80336699), NewColor(0x40FFCC00))),
		"luminance":            []any{NewColor(0xFF336699).ComputeLuminance(), half.ComputeLuminance(), p3red.ComputeLuminance()},
		"opacity":              NewColor(0x80000000).Opacity(),
	}
}

func TestPaintingMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	cases := paintingCases()
	if len(ref) != len(cases) {
		t.Fatalf("reference has %d cases, Go has %d", len(ref), len(cases))
	}
	for name, got := range cases {
		if !reflect.DeepEqual(got, ref[name]) { // exact, bit for bit
			t.Errorf("%s:\n Go:      %v\n Flutter: %v", name, got, ref[name])
		}
	}
}
