package ui

// Port of engine/src/flutter/testing/dart/color_test.dart (Flutter 3.50.1).
// Not ported: cases that need Paint (not yet ported), Dart's arbitrary
// precision int literals (`1 << 1000000`), and subclasses of Color (Go has
// no inheritance; see flutter_go/docs/API_MAPPING.md).

import (
	"math"
	"testing"
)

func colorMatches(t *testing.T, name string, got, want Color, threshold float64) {
	t.Helper()
	if got.ColorSpace != want.ColorSpace || math.Abs(got.A-want.A) > threshold || math.Abs(got.R-want.R) > threshold ||
		math.Abs(got.G-want.G) > threshold || math.Abs(got.B-want.B) > threshold {
		t.Errorf("%s = %v, want %v ± %v", name, got, want, threshold)
	}
}

func TestColorAccessors(t *testing.T) {
	foo := NewColor(0x12345678)
	check(t, "alpha", foo.Alpha(), 0x12)
	check(t, "red", foo.Red(), 0x34)
	check(t, "green", foo.Green(), 0x56)
	check(t, "blue", foo.Blue(), 0x78)
	check(t, "equal", NewColor(0x12345678) == NewColor(0x12345678), true)
	check(t, "not equal", NewColor(0x12345678) == NewColor(0x87654321), false)
	_ = NewColor(0x100 << 24) // out of bounds value is fine
}

func TestColorLerp(t *testing.T) {
	c0, c1 := NewColor(0x00000000), NewColor(0xFFFFFFFF)
	check(t, "0", *ColorLerp(&c0, &c1, 0), c0)
	colorMatches(t, "0.5", *ColorLerp(&c0, &c1, 0.5), NewColor(0x7F7F7F7F), 1.0/255)
	check(t, "1", *ColorLerp(&c0, &c1, 1), c1)
	check(t, "-0.1", *ColorLerp(&c0, &c1, -0.1), c0)
	check(t, "1.1", *ColorLerp(&c0, &c1, 1.1), c1)
	check(t, "same", *ColorLerp(&c1, &c1, 0.04), c1)
}

func p3(a, r, g, b float64) Color {
	return NewColorFrom(ColorFromOptions{Alpha: a, Red: r, Green: g, Blue: b, ColorSpace: ColorSpaceDisplayP3})
}
func srgb(a, r, g, b float64) Color {
	return NewColorFrom(ColorFromOptions{Alpha: a, Red: r, Green: g, Blue: b})
}
func inSpace(c Color, cs ColorSpace) Color {
	return c.WithValues(ColorWithValuesOptions{ColorSpace: &cs})
}

func TestColorLerpColorSpaces(t *testing.T) {
	a, b := p3(1, 0, 0, 0), p3(1, 1, 0, 0)
	check(t, "same space", *ColorLerp(&a, &b, 0.2), p3(1, 0.2, 0, 0))

	x, y := p3(1, 1, 0, 0), srgb(1, 0, 0, 1)
	check(t, "mixed → p3", ColorLerp(&x, &y, 0).ColorSpace, ColorSpaceDisplayP3)

	red, p3Green := srgb(1, 1, 0, 0), p3(1, 0, 1, 0)
	r := *ColorLerp(&red, &p3Green, 0)
	check(t, "t=0 space", r.ColorSpace, ColorSpaceDisplayP3)
	colorMatches(t, "t=0", r, inSpace(red, ColorSpaceDisplayP3), 1.0/255)
	r = *ColorLerp(&p3Green, &red, 1)
	colorMatches(t, "t=1", r, inSpace(red, ColorSpaceDisplayP3), 1.0/255)

	r = *ColorLerp(&red, &p3Green, 0.5)
	rp := inSpace(red, ColorSpaceDisplayP3)
	colorMatches(t, "mid", r, p3((rp.A+p3Green.A)/2, (rp.R+p3Green.R)/2, (rp.G+p3Green.G)/2, (rp.B+p3Green.B)/2), 1e-4)

	half, blue := srgb(0.5, 1, 0, 0), p3(1, 0, 0, 1)
	if a := ColorLerp(&half, &blue, 0.5).A; math.Abs(a-0.75) > 1e-4 {
		t.Errorf("alpha = %v, want 0.75", a)
	}
}

func TestColorAlphaBlend(t *testing.T) {
	for _, c := range []struct {
		fg, bg, want int
		approx       bool
	}{
		{0x00000000, 0x00000000, 0x00000000, false}, {0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, false},
		{0xFFFFFFFF, 0x00000000, 0xFFFFFFFF, false}, {0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, false},
		{0x80FFFFFF, 0xFF000000, 0xFF808080, false}, {0x80808080, 0xFFFFFFFF, 0xFFBFBFBF, true},
		{0x80808080, 0xFF000000, 0xFF404040, true}, {0x01020304, 0xFF000000, 0xFF000000, true},
		{0x11223344, 0xFF000000, 0xFF020304, true}, {0x11223344, 0x80000000, 0x88040608, true},
	} {
		got := ColorAlphaBlend(NewColor(c.fg), NewColor(c.bg))
		if c.approx {
			colorMatches(t, "alphaBlend", got, NewColor(c.want), 1.0/255)
		} else {
			check(t, "alphaBlend", got, NewColor(c.want))
		}
	}
	check(t, "keeps colorspace", ColorAlphaBlend(p3(0.5, 1, 1, 1), p3(1, 0, 0, 0)), p3(1, 0.5, 0.5, 0.5))
}

func TestColorLuminance(t *testing.T) {
	check(t, "gray", NewColor(0xFF333333).ComputeLuminance(), 0.033104766570885055)
	check(t, "red", NewColor(0xFFFF3B30).ComputeLuminance(), 0.24601329637099723)
}

func TestColorFromAndAccessors(t *testing.T) {
	c := srgb(0.1, 0.2, 0.3, 0.4)
	check(t, "components", [4]float64{c.A, c.R, c.G, c.B}, [4]float64{0.1, 0.2, 0.3, 0.4})
	check(t, "space", c.ColorSpace, ColorSpaceSRGB)
	check(t, "8-bit", [4]int{c.Alpha(), c.Red(), c.Green(), c.Blue()}, [4]int{26, 51, 77, 102})
	check(t, "value", c.Value(), 0x1a334d66)
	check(t, "toARGB32", c.ToARGB32(), 0x1a334d66)
	a := NewColorFromARGB(10, 20, 35, 47)
	check(t, "fromARGB", [4]int{a.Alpha(), a.Red(), a.Green(), a.Blue()}, [4]int{10, 20, 35, 47})
	b := NewColor(0xffeeddcc)
	check(t, "constructor", [4]int{b.Alpha(), b.Red(), b.Green(), b.Blue()}, [4]int{0xff, 0xee, 0xdd, 0xcc})
}

func TestColorSpaceConversions(t *testing.T) {
	e := inSpace(p3(1, 1, 0, 0), ColorSpaceExtendedSRGB)
	colorMatches(t, "p3 → extended", e, NewColorFrom(ColorFromOptions{Alpha: 1, Red: 1.0931, Green: -0.22684034705162098,
		Blue: -0.15007957816123998, ColorSpace: ColorSpaceExtendedSRGB}), 1e-4)
	check(t, "p3 → extended alpha", e.A, 1.0)
	s := inSpace(p3(1, 1, 0, 0), ColorSpaceSRGB)
	colorMatches(t, "p3 → srgb", s, srgb(1, 1, 0, 0), 1e-4)
	x := NewColorFrom(ColorFromOptions{Alpha: 1, Red: 1.0931, Green: -0.2268, Blue: -0.1501, ColorSpace: ColorSpaceExtendedSRGB})
	colorMatches(t, "extended → p3", inSpace(x, ColorSpaceDisplayP3), p3(1, 1, 0, 0), 1e-4)
	c := inSpace(NewColorFrom(ColorFromOptions{Alpha: 1, Red: 2, ColorSpace: ColorSpaceExtendedSRGB}), ColorSpaceDisplayP3)
	for _, v := range []float64{c.R, c.G, c.B} {
		if v < 0 || v > 1 {
			t.Errorf("extended → p3 not clamped: %v", c)
		}
	}
	check(t, "equality considers colorspace", srgb(1, 1, 0, 0) == p3(1, 1, 0, 0), false)
}
