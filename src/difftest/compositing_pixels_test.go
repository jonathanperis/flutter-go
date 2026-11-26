package difftest

// Pixel differential test of dart:ui's compositing (SceneBuilder, the
// engine's layer tree painted through the LayerStateStack) against
// Flutter's Impeller (flutter_tester --enable-impeller): the same 64×64
// scenes as flutter/test/impeller/compositing_pixels_test.dart, rasterized
// with Scene.toImage. Tolerances follow painting_pixels_test.go (the two
// GPUs' MSAA sample positions differ on anti-aliased edges); scenes not
// listed in compositingPixelTolerance must match within 2 everywhere.

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

func cpic(draw func(c Canvas)) *Picture {
	rec := NewPictureRecorder()
	draw(NewCanvas(rec))
	return rec.EndRecording()
}

func cpaint(color int) *Paint { q := NewPaint(); q.SetColor(NewColor(color)); return q }

func overlapPic() *Picture {
	return cpic(func(c Canvas) {
		c.DrawRect(NewRectFromLTRB(8, 8, 40, 40), cpaint(0xFF0000FF))
		c.DrawRect(NewRectFromLTRB(24, 24, 56, 56), cpaint(0xFFFF0000))
	})
}

func quadrantsPic() *Picture {
	return cpic(func(c Canvas) {
		c.DrawRect(NewRectFromLTRB(0, 0, 32, 32), cpaint(0xFFFF0000))
		c.DrawRect(NewRectFromLTRB(32, 0, 64, 32), cpaint(0xFF00FF00))
		c.DrawRect(NewRectFromLTRB(0, 32, 32, 64), cpaint(0xFF0000FF))
		c.DrawRect(NewRectFromLTRB(32, 32, 64, 64), cpaint(0xFFFFFF00))
	})
}

func squarePic(color int) *Picture {
	return cpic(func(c Canvas) { c.DrawRect(NewRectFromLTRB(16, 16, 48, 48), cpaint(color)) })
}

func rotationM(r float64) []float64 {
	c, s := math.Cos(r), math.Sin(r)
	return []float64{c, s, 0, 0, -s, c, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func compositingScenes() map[string]func(b SceneBuilder) {
	scale2 := []float64{2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	hard, saveLayer := ClipHardEdge, ClipAntiAliasWithSaveLayer
	none := FilterQualityNone
	return map[string]func(b SceneBuilder){
		"picture": func(b SceneBuilder) { b.AddPicture(NewOffset(4, 6), overlapPic()) },
		"offset.nested": func(b SceneBuilder) {
			b.PushOffset(8, 4)
			b.PushOffset(-4, 8)
			b.AddPicture(OffsetZero, squarePic(0xFF336699))
			b.Pop()
			b.Pop()
		},
		"transform.scale": func(b SceneBuilder) {
			b.PushTransform(scale2)
			b.AddPicture(OffsetZero, cpic(func(c Canvas) { c.DrawRect(NewRectFromLTRB(4, 4, 20, 12), cpaint(0xFF884422)) }))
			b.Pop()
		},
		"transform.rotate.offset": func(b SceneBuilder) {
			b.PushOffset(32, 32)
			b.PushTransform(rotationM(math.Pi / 2))
			b.AddPicture(OffsetZero, cpic(func(c Canvas) { c.DrawRect(NewRectFromLTRB(0, 0, 24, 12), cpaint(0xFF22AA44)) }))
			b.Pop()
			b.Pop()
		},
		"transform.zero": func(b SceneBuilder) {
			b.PushTransform(make([]float64, 16))
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"clipRect.hardEdge": func(b SceneBuilder) {
			b.PushClipRect(NewRectFromLTRB(10, 12, 50, 40), PushClipRectOptions{ClipBehavior: &hard})
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"clipRect.antiAlias": func(b SceneBuilder) {
			b.PushClipRect(NewRectFromLTRB(10, 12, 50, 40))
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"clipRect.saveLayer.opacity": func(b SceneBuilder) {
			b.PushOpacity(128)
			b.PushClipRect(NewRectFromLTRB(4, 4, 60, 60), PushClipRectOptions{ClipBehavior: &saveLayer})
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
			b.Pop()
		},
		"clipRRect": func(b SceneBuilder) {
			b.PushClipRRect(NewRRectFromLTRBR(8, 8, 56, 56, NewRadiusCircular(16)))
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"clipRRect.rect": func(b SceneBuilder) {
			b.PushClipRRect(NewRRectFromLTRBR(8, 16, 56, 48, RadiusZero), PushClipRRectOptions{ClipBehavior: &hard})
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"clipRSuperellipse": func(b SceneBuilder) {
			b.PushClipRSuperellipse(NewRSuperellipseFromLTRBR(6, 6, 58, 58, NewRadiusCircular(20)))
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"clipPath.oval": func(b SceneBuilder) {
			path := NewPath()
			path.AddOval(NewRectFromLTRB(4, 12, 60, 52))
			b.PushClipPath(path)
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"clips.nested": func(b SceneBuilder) {
			b.PushClipRect(NewRectFromLTRB(0, 0, 40, 64), PushClipRectOptions{ClipBehavior: &hard})
			b.PushOffset(8, 0)
			b.PushClipRect(NewRectFromLTRB(0, 16, 64, 48), PushClipRectOptions{ClipBehavior: &hard})
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
			b.Pop()
			b.Pop()
		},
		"opacity": func(b SceneBuilder) {
			b.PushOpacity(128)
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
		},
		"opacity.offset": func(b SceneBuilder) {
			b.PushOpacity(200, PushOpacityOptions{Offset: NewOffset(4, -4)})
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
		},
		"opacity.nested": func(b SceneBuilder) {
			b.PushOpacity(160)
			b.AddPicture(OffsetZero, squarePic(0xFF00FF00))
			b.PushOpacity(100)
			b.AddPicture(NewOffset(12, 12), squarePic(0xFF0000FF))
			b.Pop()
			b.Pop()
		},
		"opacity.nested.direct": func(b SceneBuilder) {
			b.PushOpacity(128)
			b.PushOpacity(128)
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
			b.Pop()
		},
		"transform.w2": func(b SceneBuilder) {
			b.PushTransform([]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 8, 4, 0, 2})
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"backdrop.blur.full": func(b SceneBuilder) {
			b.AddPicture(OffsetZero, quadrantsPic())
			b.PushBackdropFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 6, SigmaY: 6}))
			b.Pop()
		},
		"opacity.wrap300": func(b SceneBuilder) {
			b.PushOpacity(300)
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
		},
		"opacity.zero": func(b SceneBuilder) {
			b.AddPicture(OffsetZero, squarePic(0xFF000000))
			b.PushOpacity(0)
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
		},
		"colorFilter.mode": func(b SceneBuilder) {
			b.PushColorFilter(NewColorFilterMode(NewColor(0xFF00AA00), BlendModeModulate))
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"colorFilter.matrix": func(b SceneBuilder) {
			b.PushColorFilter(NewColorFilterMatrix([]float64{0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0,
				0.2126, 0.7152, 0.0722, 0, 0, 0, 0, 0, 1, 0}))
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"colorFilter.opacity.commute": func(b SceneBuilder) {
			b.PushOpacity(128)
			b.PushColorFilter(NewColorFilterMode(NewColor(0xFFFF8800), BlendModeSrcIn))
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
			b.Pop()
		},
		"colorFilter.opacity.noCommute": func(b SceneBuilder) {
			b.PushOpacity(128)
			b.PushColorFilter(NewColorFilterMode(NewColor(0x8000FFFF), BlendModeSrcOver))
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
			b.Pop()
		},
		"imageFilter.blur": func(b SceneBuilder) {
			b.PushImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 3, SigmaY: 3}))
			b.AddPicture(OffsetZero, squarePic(0xFFFF8800))
			b.Pop()
		},
		"imageFilter.blur.offset": func(b SceneBuilder) {
			b.PushImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 2, SigmaY: 4}),
				PushImageFilterOptions{Offset: NewOffset(6, -6)})
			b.AddPicture(OffsetZero, squarePic(0xFF8800FF))
			b.Pop()
		},
		"imageFilter.dilate": func(b SceneBuilder) {
			b.PushImageFilter(NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: 4, RadiusY: 2}))
			b.AddPicture(OffsetZero, squarePic(0xFF0088FF))
			b.Pop()
		},
		"imageFilter.matrix": func(b SceneBuilder) {
			b.PushImageFilter(NewImageFilterMatrix([]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 10, 5, 0, 1},
				ImageFilterMatrixOptions{FilterQuality: &none}))
			b.AddPicture(OffsetZero, squarePic(0xFF00AA88))
			b.Pop()
		},
		"imageFilter.opacity": func(b SceneBuilder) {
			b.PushOpacity(128)
			b.PushImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 2, SigmaY: 2}))
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
			b.Pop()
		},
		"backdrop.blur": func(b SceneBuilder) {
			b.AddPicture(OffsetZero, quadrantsPic())
			b.PushClipRect(NewRectFromLTRB(16, 16, 48, 48), PushClipRectOptions{ClipBehavior: &hard})
			b.PushBackdropFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 4, SigmaY: 4}))
			b.Pop()
			b.Pop()
		},
		"backdrop.blur.child": func(b SceneBuilder) {
			b.AddPicture(OffsetZero, quadrantsPic())
			b.PushClipRect(NewRectFromLTRB(8, 24, 56, 40), PushClipRectOptions{ClipBehavior: &hard})
			b.PushBackdropFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 3, SigmaY: 3}))
			b.AddPicture(OffsetZero, cpic(func(c Canvas) { c.DrawRect(NewRectFromLTRB(28, 0, 36, 64), cpaint(0x80FFFFFF)) }))
			b.Pop()
			b.Pop()
		},
		"backdrop.matrix": func(b SceneBuilder) {
			b.AddPicture(OffsetZero, quadrantsPic())
			b.PushClipRect(NewRectFromLTRB(16, 16, 48, 48), PushClipRectOptions{ClipBehavior: &hard})
			b.PushBackdropFilter(NewImageFilterMatrix([]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 8, 8, 0, 1},
				ImageFilterMatrixOptions{FilterQuality: &none}))
			b.Pop()
			b.Pop()
		},
		"shaderMask.linear.dstIn": func(b SceneBuilder) {
			b.PushShaderMask(NewGradientLinear(OffsetZero, NewOffset(48, 0), []Color{NewColor(0xFF000000), NewColor(0x00000000)}),
				NewRectFromLTRB(8, 8, 56, 56), BlendModeDstIn)
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
		},
		"shaderMask.radial.srcATop": func(b SceneBuilder) {
			b.PushShaderMask(NewGradientRadial(NewOffset(24, 24), 24, []Color{NewColor(0xFFFFFFFF), NewColor(0xFF000000)}),
				NewRectFromLTRB(8, 8, 56, 56), BlendModeSrcATop)
			b.AddPicture(OffsetZero, overlapPic())
			b.Pop()
		},
		"shaderMask.opacity": func(b SceneBuilder) {
			b.PushOpacity(128)
			b.PushShaderMask(NewGradientLinear(OffsetZero, NewOffset(0, 64), []Color{NewColor(0xFF000000), NewColor(0x00000000)}),
				NewRectFromLTRB(0, 0, 64, 64), BlendModeDstIn)
			b.AddPicture(OffsetZero, quadrantsPic())
			b.Pop()
			b.Pop()
		},
		"texture.platformView": func(b SceneBuilder) {
			b.AddPicture(OffsetZero, squarePic(0xFF123456))
			b.AddTexture(1, AddTextureOptions{Offset: NewOffset(4, 4), Width: 20, Height: 20})
			b.AddPlatformView(2, AddPlatformViewOptions{Offset: NewOffset(30, 30), Width: 10, Height: 10})
			b.AddPerformanceOverlay(0, NewRectFromLTRB(0, 0, 64, 64))
		},
		"retained": func(b SceneBuilder) {
			first := NewSceneBuilder()
			layer := first.PushOpacity(128)
			first.PushOffset(8, 8)
			first.AddPicture(OffsetZero, overlapPic())
			first.Pop()
			first.Pop()
			first.Build().Dispose()
			b.AddPicture(OffsetZero, squarePic(0xFF00FF00))
			b.AddRetained(layer)
		},
		"complex": func(b SceneBuilder) {
			b.AddPicture(OffsetZero, quadrantsPic())
			b.PushTransform([]float64{1, 0, 0, 0, 0.25, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1})
			b.PushClipRRect(NewRRectFromLTRBR(4, 4, 52, 52, NewRadiusCircular(8)))
			b.PushOpacity(192)
			b.PushColorFilter(NewColorFilterMode(NewColor(0xFF3366FF), BlendModeModulate))
			b.AddPicture(OffsetZero, overlapPic())
			b.PushOffset(16, 16)
			b.AddPicture(OffsetZero, cpic(func(c Canvas) { c.DrawCircle(NewOffset(8, 8), 8, cpaint(0xFFFFFFFF)) }))
			b.Pop()
			b.Pop()
			b.Pop()
			b.Pop()
			b.Pop()
		},
	}
}

// compositingPixelTolerance, measured (Metal here vs SwiftShader in
// flutter_tester):
//   - clipRRect, clipPath.oval: anti-aliased clip edges differ by one MSAA
//     sample of four (64), as rrect/oval draws in painting_pixels.
//   - clipRSuperellipse: the SDK has no superellipse clip, so the clip is
//     dart:ui's RSuperellipse path, whose flattening differs from
//     Impeller's superellipse geometry on edge pixels by up to two samples
//     (128).
//   - complex: the white circle under a skew differs by ≤4 on 5 edge
//     pixels (Impeller's circle geometry, as "circle" in painting_pixels).
//
// Every other scene (transforms, hard and anti-aliased rect clips, clip
// save layers, opacity incl. nested/offset/0/300, color filters with and
// without opacity, image filters, backdrop filters, shader masks,
// retained layers, textures/platform views) is within 1.
var compositingPixelTolerance = map[string]struct{ max, over2 int }{
	"clipPath.oval":     {65, 30},
	"clipRRect":         {65, 8},
	"clipRSuperellipse": {129, 50},
	"complex":           {5, 8},
}

func TestCompositingPixelsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/compositing_pixels.json")
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
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	scenes := compositingScenes()
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
		b := NewSceneBuilder()
		scenes[name](b)
		scene := b.Build()
		got := awaitF(t, awaitF(t, scene.ToImage(pixelSize, pixelSize)).ToByteData())
		scene.Dispose()
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
		tol, ok := compositingPixelTolerance[name]
		if !ok {
			tol.max, tol.over2 = 2, 0
		}
		t.Logf("%-32s max %3d, pixels >2: %4d, >16: %4d", name, maxDiff, over2, over16)
		if maxDiff > tol.max || over2 > tol.over2 {
			t.Errorf("%s: max channel difference %d (%d pixels > 2), tolerance %+v", name, maxDiff, over2, tol)
		}
	}
}
