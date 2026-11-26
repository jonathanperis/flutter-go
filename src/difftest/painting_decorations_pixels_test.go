package difftest

// Pixel differential test of the painting library's gradients
// (createShader), box shadows (BoxShadow.toPaint, drawn as BoxDecoration
// draws them), paintImage and the Flutter logo against Flutter's own
// Impeller renderer (flutter_tester --enable-impeller): the same 64×64
// scenes as flutter/test/impeller/painting_decorations_pixels_test.dart,
// compared with painting_pixels_test.go's per-scene tolerance scheme.

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func decorationPixelScenes(img *ui.Image) map[string]func(c ui.Canvas) {
	full := ui.NewRectFromLTWH(0, 0, 64, 64)
	col := func(vs ...int) []ui.Color {
		out := make([]ui.Color, len(vs))
		for i, v := range vs {
			out[i] = ui.NewColor(v)
		}
		return out
	}
	rgb := col(0xFFFF0000, 0xFF00FF00, 0xFF0000FF)
	shaderRect := func(c ui.Canvas, g Gradient, rect ui.Rect, d *ui.TextDirection) {
		p := ui.NewPaint()
		p.SetShader(g.CreateShader(rect, d))
		c.DrawRect(full, p)
	}
	shadow := func(c ui.Canvas, s BoxShadow, rect ui.Rect) {
		white := ui.NewPaint()
		white.SetColor(ui.NewColor(0xFFFFFFFF))
		c.DrawRect(full, white)
		c.DrawRect(rect.Shift(s.Offset).Inflate(s.SpreadRadius), s.ToPaint())
	}
	colorP := func(v int) *ui.Color { c := ui.NewColor(v); return &c }
	fit := func(f BoxFit) *BoxFit { return &f }
	rep := func(r ImageRepeat) *ImageRepeat { return &r }
	none := ui.FilterQualityNone
	f := func(v float64) *float64 { return &v }
	rtl := ui.TextDirectionRtl
	logo := func(d *FlutterLogoDecoration, o ui.Offset, s ui.Size) func(c ui.Canvas) {
		return func(c ui.Canvas) { d.CreateBoxPainter(nil).Paint(c, o, ImageConfiguration{Size: &s}) }
	}
	slice := ui.NewRectFromLTRB(4, 4, 12, 12)
	return map[string]func(c ui.Canvas){
		"linear": func(c ui.Canvas) {
			shaderRect(c, NewLinearGradient(LinearGradientOptions{Colors: col(0xFFFF0000, 0xFF0000FF)}), full, nil)
		},
		"linear.rotated.mirror": func(c ui.Canvas) {
			shaderRect(c, NewLinearGradient(LinearGradientOptions{Begin: NewAlignment(-0.2, -0.4), End: NewAlignment(0.1, 0.0),
				Colors: rgb, Stops: []float64{0.1, 0.4, 0.8}, TileMode: ui.TileModeMirror, Transform: NewGradientRotation(0.6)}), full, nil)
		},
		"linear.directional.rtl": func(c ui.Canvas) {
			shaderRect(c, NewLinearGradient(LinearGradientOptions{Begin: AlignmentDirectionalTopStart,
				End: NewAlignmentDirectional(0.5, 1.0), Colors: col(0xFF00FFFF, 0x80FF00FF)}), ui.NewRectFromLTWH(4, 8, 50, 40), &rtl)
		},
		"radial": func(c ui.Canvas) {
			shaderRect(c, NewRadialGradient(RadialGradientOptions{Colors: col(0xFFFFFF00, 0xFF8000FF)}), full, nil)
		},
		"radial.focal.decal": func(c ui.Canvas) {
			shaderRect(c, NewRadialGradient(RadialGradientOptions{Center: NewAlignment(0.2, 0.1), Radius: f(0.6), Colors: rgb,
				TileMode: ui.TileModeDecal, Focal: NewAlignment(-0.3, -0.2), FocalRadius: 0.05,
				Transform: NewGradientRotation(-0.9)}), full, nil)
		},
		"radial.focal.wide": func(c ui.Canvas) {
			shaderRect(c, NewRadialGradient(RadialGradientOptions{Center: NewAlignment(0.1, 0.2), Radius: f(0.9), Colors: rgb,
				Focal: NewAlignment(-0.4, 0.1), FocalRadius: 0.3}), ui.NewRectFromLTWH(0, 12, 64, 30), nil)
		},
		"radial.repeated": func(c ui.Canvas) {
			shaderRect(c, NewRadialGradient(RadialGradientOptions{Radius: f(0.2), Colors: col(0xFFFFFFFF, 0xFF004080),
				Stops: []float64{0.2, 0.9}, TileMode: ui.TileModeRepeated}), full, nil)
		},
		"sweep": func(c ui.Canvas) { shaderRect(c, NewSweepGradient(SweepGradientOptions{Colors: rgb}), full, nil) },
		"sweep.angles.rotated": func(c ui.Canvas) {
			shaderRect(c, NewSweepGradient(SweepGradientOptions{Center: NewAlignment(-0.25, 0.3), StartAngle: 0.4, EndAngle: f(4.2),
				Colors: rgb, Stops: []float64{0.0, 0.3, 1.0}, TileMode: ui.TileModeRepeated,
				Transform: NewGradientRotation(dartPi / 5)}), full, nil)
		},
		"gradient.lerp": func(c ui.Canvas) {
			shaderRect(c, GradientLerp(
				NewLinearGradient(LinearGradientOptions{Colors: col(0xFFFF0000, 0xFF0000FF), Stops: []float64{0.2, 0.7}}),
				NewLinearGradient(LinearGradientOptions{Begin: AlignmentTopCenter, End: AlignmentBottomCenter, Colors: rgb,
					Stops: []float64{0.0, 0.5, 1.0}}), 0.4), full, nil)
		},
		"shadow.normal": func(c ui.Canvas) {
			shadow(c, NewBoxShadow(BoxShadowOptions{Color: colorP(0x99000000), Offset: ui.NewOffset(3, 4), BlurRadius: 6}),
				ui.NewRectFromLTWH(16, 16, 28, 24))
		},
		"shadow.outer.spread": func(c ui.Canvas) {
			shadow(c, NewBoxShadow(BoxShadowOptions{Color: colorP(0xFF3366CC), BlurRadius: 4, SpreadRadius: 3,
				BlurStyle: ui.BlurStyleOuter}), ui.NewRectFromLTWH(18, 18, 28, 28))
		},
		"shadow.inner": func(c ui.Canvas) {
			shadow(c, NewBoxShadow(BoxShadowOptions{Color: colorP(0xFFCC3300), Offset: ui.NewOffset(-2, 1), BlurRadius: 5,
				BlurStyle: ui.BlurStyleInner}), ui.NewRectFromLTWH(10, 12, 40, 36))
		},
		"shadow.solid": func(c ui.Canvas) {
			shadow(c, NewBoxShadow(BoxShadowOptions{Color: colorP(0x80008000), BlurRadius: 3, SpreadRadius: -2,
				BlurStyle: ui.BlurStyleSolid}), ui.NewRectFromLTWH(14, 14, 36, 36))
		},
		"image.cover": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(4, 8, 56, 40), img, PaintImageOptions{Fit: fit(BoxFitCover), FilterQuality: &none})
		},
		"image.contain.bottomRight": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(2, 2, 60, 40), img, PaintImageOptions{Fit: fit(BoxFitContain),
				Alignment: AlignmentBottomRight, FilterQuality: &none})
		},
		"image.none.flip": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(8, 8, 48, 48), img, PaintImageOptions{Fit: fit(BoxFitNone), Alignment: AlignmentTopLeft,
				FlipHorizontally: true})
		},
		"image.repeat": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(3, 5, 58, 50), img, PaintImageOptions{Scale: f(1.5), Repeat: rep(ImageRepeatRepeat),
				Alignment: NewAlignment(0.3, -0.6), FilterQuality: &none})
		},
		"image.repeatX.fitHeight": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(0, 16, 64, 24), img, PaintImageOptions{Fit: fit(BoxFitFitHeight),
				Repeat: rep(ImageRepeatRepeatX), FilterQuality: &none})
		},
		"image.centerSlice": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(4, 4, 56, 50), img, PaintImageOptions{CenterSlice: &slice, FilterQuality: &none})
		},
		"image.invert": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(8, 8, 48, 48), img, PaintImageOptions{Fit: fit(BoxFitFill),
				InvertColors: true, FilterQuality: &none})
		},
		"image.opacity.colorFilter": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(8, 8, 48, 48), img, PaintImageOptions{Fit: fit(BoxFitFill), Opacity: f(0.6),
				ColorFilter: ui.NewColorFilterMode(ui.NewColor(0xFF2040A0), ui.BlendModeModulate), FilterQuality: &none})
		},
		"image.colorFilter.matrix.invert": func(c ui.Canvas) {
			PaintImage(c, ui.NewRectFromLTWH(8, 8, 48, 48), img, PaintImageOptions{Fit: fit(BoxFitFill), InvertColors: true,
				ColorFilter:   ui.NewColorFilterMatrix([]float64{0.5, 0, 0, 0, 20, 0, 1, 0, 0, 0, 0, 0, 0.25, 0, 40, 0, 0, 0, 1, 0}),
				FilterQuality: &none})
		},
		"logo.mark": logo(NewFlutterLogoDecoration(), ui.OffsetZero, ui.NewSize(64, 64)),
		"logo.mark.margin": logo(NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Margin: EdgeInsets{Left: 4, Top: 10, Right: 2, Bottom: 6}}),
			ui.NewOffset(3, 1), ui.NewSize(58, 40)),
		"logo.mark.fading": logo(FlutterLogoDecorationLerp(nil, NewFlutterLogoDecoration(), 0.5), ui.OffsetZero, ui.NewSize(64, 64)),
	}
}

// decorationPixelTolerance is each scene's allowed maximum per-channel
// difference and how many pixels (of 4096) may differ by more than 2;
// scenes not listed allow max 2 and none over 2. Measured on Apple Metal
// (here) vs SwiftShader Vulkan (flutter_tester). Gradients, shadows,
// center slices, repeats and the logo are within 2. The three scenes
// below scale the 16×16 checker by 2.5 with FilterQuality.none (nearest
// sampling): every fifth pixel center falls exactly on a texel boundary
// and the two GPUs round that tie to opposite texels, so those pixels show
// the neighbouring texel (blue steps by 16 per texel in the checker, red
// and green change every 4 texels) — only tie pixels differ, by one texel.
//
// Not compared: invertColors with a translucent paint (opacity < 1) on an
// image. Flutter's Impeller output is opaque with darker colors (its
// inversion composes differently with the paint's alpha) while the SDK's
// is the premultiplied inverted image; a dart:ui ENGINE GAP like the other
// invertColors compositions listed in PLAN.md (paintImage itself sets the
// same Paint, which the value difftest compares exactly). Opaque
// inversion ("image.invert") matches.
var decorationPixelTolerance = map[string]struct{ max, over2 int }{
	"image.contain.bottomRight": {16, 260},
	"image.cover":               {16, 25},
	"image.repeatX.fitHeight":   {16, 45},
}

func TestPaintingDecorationsPixelsMatchFlutter(t *testing.T) {
	// Shadows paint as in an app (the Dart side sets debugDisableShadows =
	// false; a test binding created earlier in this process sets it true).
	defer func(old bool) { DebugDisableShadows = old }(DebugDisableShadows)
	DebugDisableShadows = false
	data, err := os.ReadFile("flutter/build/painting_decorations_pixels.json")
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
	flutter_test.New(t) // paintImage reads PaintingBinding.instance
	ui.NewPlatformConfiguration(nil).UpdateViewGestureMetrics(0, 3.0, ui.GestureSettings{})
	defer ui.NewPlatformConfiguration(nil).UpdateViewGestureMetrics(0, 1.0, ui.GestureSettings{})
	img := checkerImage(t)
	scenes := decorationPixelScenes(img)
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
		if err != nil || len(want) != pixelSize*pixelSize*4 {
			t.Errorf("%s: bad reference", name)
			continue
		}
		rec := ui.NewPictureRecorder()
		scenes[name](ui.NewCanvas(rec))
		got := awaitF(t, awaitF(t, rec.EndRecording().ToImage(pixelSize, pixelSize)).ToByteData())
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
		tol, ok := decorationPixelTolerance[name]
		if !ok {
			tol.max, tol.over2 = 2, 0
		}
		t.Logf("%-30s max %3d, pixels >2: %4d, >16: %4d", name, maxDiff, over2, over16)
		if maxDiff > tol.max || over2 > tol.over2 {
			t.Errorf("%s: max channel difference %d (%d pixels > 2), tolerance %+v", name, maxDiff, over2, tol)
		}
	}
}
