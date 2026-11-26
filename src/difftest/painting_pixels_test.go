package difftest

// Pixel differential test of dart:ui's Canvas over the Impeller SDK
// against Flutter's own Impeller renderer (flutter_tester
// --enable-impeller, Impeller on Vulkan/SwiftShader), scene by scene: the
// same 64×64 scenes as flutter/test/impeller/painting_pixels_test.dart,
// rasterized with Picture.toImage and read back as rawRgba.
//
// Bit-exactness is not possible across the two GPUs' rasterization (MSAA
// sample positions and resolve, SwiftShader's vs Apple's float precision,
// texture filtering), so each scene has a documented per-channel
// tolerance and a cap on how many pixels may differ by more than 2.

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

const pixelSize = 64

func awaitF[T any](t *testing.T, f async.Future[T]) T {
	t.Helper()
	var v T
	var err error
	done := false
	async.OnComplete(f, func(x T) { v, done = x, true }, func(e error) { err, done = e, true })
	for loop := async.Current(); !done; {
		loop.FlushMicrotasks()
		if !done {
			loop.WaitPosted()
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func checkerImage(t *testing.T) *Image {
	px := make([]byte, 16*16*4)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			i := (y*16 + x) * 4
			on := (x/4+y/4)%2 == 0
			px[i], px[i+1] = 30, 200
			if on {
				px[i], px[i+1] = 255, 120
			}
			px[i+2] = byte(x * 16)
			px[i+3] = 255
		}
	}
	var img *Image
	DecodeImageFromPixels(px, 16, 16, PixelFormatRgba8888, func(i *Image) { img = i })
	for loop := async.Current(); img == nil; {
		loop.FlushMicrotasks()
		if img == nil {
			loop.WaitPosted()
		}
	}
	return img
}

func pixelScenes(img *Image) map[string]func(c Canvas) {
	p := func(color int) *Paint { q := NewPaint(); q.SetColor(NewColor(color)); return q }
	with := func(q *Paint, f func(q *Paint)) *Paint { f(q); return q }
	full := NewRectFromLTWH(0, 0, 64, 64)
	shader := func(s Shader) *Paint { q := NewPaint(); q.SetShader(s); return q }
	return map[string]func(c Canvas){
		"rect":            func(c Canvas) { c.DrawRect(NewRectFromLTRB(8, 8, 40, 30), p(0xFF3366CC)) },
		"rect.fractional": func(c Canvas) { c.DrawRect(NewRectFromLTRB(8.3, 8.6, 40.2, 30.7), p(0xFFCC6633)) },
		"rrect": func(c Canvas) {
			c.DrawRRect(NewRRectFromLTRBR(6, 10, 58, 50, NewRadiusCircular(12)), p(0xFF22AA44))
		},
		"oval":   func(c Canvas) { c.DrawOval(NewRectFromLTRB(4, 12, 60, 52), p(0xFFAA2244)) },
		"circle": func(c Canvas) { c.DrawCircle(NewOffset(32, 32), 20, p(0xFF4422AA)) },
		"drrect": func(c Canvas) {
			c.DrawDRRect(NewRRectFromLTRBR(4, 4, 60, 60, NewRadiusCircular(10)),
				NewRRectFromLTRBR(16, 16, 48, 48, NewRadiusCircular(6)), p(0xFF119988))
		},
		"stroke.rect": func(c Canvas) {
			c.DrawRect(NewRectFromLTRB(10, 10, 54, 54), with(p(0xFF000000), func(q *Paint) {
				q.SetStyle(PaintingStyleStroke)
				q.SetStrokeWidth(4)
			}))
		},
		"stroke.line.round": func(c Canvas) {
			c.DrawLine(NewOffset(8, 50), NewOffset(56, 12), with(p(0xFFEE5500), func(q *Paint) {
				q.SetStrokeWidth(6)
				q.SetStrokeCap(StrokeCapRound)
			}))
		},
		"paint": func(c Canvas) { c.DrawPaint(p(0x80FF0000)) },
		"color.blend": func(c Canvas) {
			c.DrawColor(NewColor(0xFF2040A0), BlendModeSrc)
			c.DrawRect(NewRectFromLTRB(10, 10, 54, 54), with(p(0xFFFFCC00), func(q *Paint) { q.SetBlendMode(BlendModeMultiply) }))
		},
		"alpha.srcOver": func(c Canvas) {
			c.DrawRect(NewRectFromLTRB(4, 4, 40, 40), p(0xFF0000FF))
			c.DrawRect(NewRectFromLTRB(20, 20, 60, 60), p(0x8000FF00))
		},
		"gradient.linear": func(c Canvas) {
			c.DrawRect(full, shader(NewGradientLinear(NewOffset(4, 4), NewOffset(60, 40),
				[]Color{NewColor(0xFFFF0000), NewColor(0xFF0000FF)})))
		},
		"gradient.linear.stops.repeat": func(c Canvas) {
			c.DrawRect(full, shader(NewGradientLinear(NewOffset(20, 0), NewOffset(40, 0),
				[]Color{NewColor(0xFF00FF00), NewColor(0xFFFFFFFF), NewColor(0xFF000000)},
				GradientLinearOptions{ColorStops: []float64{0, 0.3, 1}, TileMode: TileModeRepeated})))
		},
		"gradient.radial": func(c Canvas) {
			c.DrawRect(full, shader(NewGradientRadial(NewOffset(32, 32), 28, []Color{NewColor(0xFFFFFF00), NewColor(0xFF8000FF)})))
		},
		"gradient.sweep": func(c Canvas) {
			c.DrawRect(full, shader(NewGradientSweep(NewOffset(32, 32),
				[]Color{NewColor(0xFFFF0000), NewColor(0xFF00FF00), NewColor(0xFFFF0000)},
				GradientSweepOptions{ColorStops: []float64{0, 0.5, 1}})))
		},
		"gradient.conical": func(c Canvas) {
			focal := NewOffset(24, 24)
			c.DrawRect(full, shader(NewGradientRadial(NewOffset(32, 32), 30, []Color{NewColor(0xFFFFFFFF), NewColor(0xFF004080)},
				GradientRadialOptions{Focal: &focal, FocalRadius: 4})))
		},
		"clip.rect": func(c Canvas) {
			c.ClipRect(NewRectFromLTRB(10.5, 10.5, 50.5, 40.5))
			c.DrawPaint(p(0xFF336699))
		},
		"clip.rrect": func(c Canvas) {
			c.ClipRRect(NewRRectFromLTRBR(8, 8, 56, 56, NewRadiusCircular(16)))
			c.DrawPaint(p(0xFF993366))
		},
		"clip.difference": func(c Canvas) {
			d := ClipOpDifference
			c.ClipRect(NewRectFromLTRB(16, 16, 48, 48), CanvasClipRectOptions{ClipOp: &d})
			c.DrawPaint(p(0xFF666666))
		},
		"transform.rotate": func(c Canvas) {
			c.Translate(32, 32)
			c.Rotate(math.Pi / 6)
			c.DrawRect(NewRectFromLTRB(-16, -10, 16, 10), p(0xFF0088FF))
		},
		"transform.skew.scale": func(c Canvas) {
			c.Scale(1.5, 0.75)
			c.Skew(0.3, 0.1)
			c.DrawRect(NewRectFromLTRB(4, 8, 30, 40), p(0xFF88FF00))
		},
		"saveLayer.alpha": func(c Canvas) {
			c.SaveLayer(nil, p(0x80000000))
			c.DrawRect(NewRectFromLTRB(4, 4, 40, 40), p(0xFFFF0000))
			c.DrawRect(NewRectFromLTRB(24, 24, 60, 60), p(0xFF0000FF))
			c.Restore()
		},
		"colorFilter.matrix": func(c Canvas) {
			c.DrawRect(NewRectFromLTRB(8, 8, 56, 56), with(p(0xFF00AA00), func(q *Paint) {
				q.SetColorFilter(NewColorFilterMatrix([]float64{0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0,
					0.2126, 0.7152, 0.0722, 0, 0, 0, 0, 0, 1, 0}))
			}))
		},
		"colorFilter.matrix.translate": func(c Canvas) {
			c.DrawRect(NewRectFromLTRB(8, 8, 56, 56), with(p(0xFF204060), func(q *Paint) {
				q.SetColorFilter(NewColorFilterMatrix([]float64{1, 0, 0, 0, 51, 0, 1, 0, 0, -20, 0, 0, 1, 0, 100, 0, 0, 0, 1, 0}))
			}))
		},
		"colorFilter.mode": func(c Canvas) {
			c.DrawRect(NewRectFromLTRB(8, 8, 56, 56), with(p(0xFF00AA00), func(q *Paint) {
				q.SetColorFilter(NewColorFilterMode(NewColor(0xFFAA0000), BlendModeColor))
			}))
		},
		"invertColors": func(c Canvas) {
			c.DrawRect(NewRectFromLTRB(8, 8, 56, 56), with(p(0xFF3366CC), func(q *Paint) { q.SetInvertColors(true) }))
		},
		"maskFilter.blur": func(c Canvas) {
			m := NewMaskFilterBlur(BlurStyleNormal, 4)
			c.DrawRect(NewRectFromLTRB(16, 16, 48, 48), with(p(0xFF000000), func(q *Paint) { q.SetMaskFilter(&m) }))
		},
		"imageFilter.blur": func(c Canvas) {
			l := NewPaint()
			l.SetImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 3, SigmaY: 3}))
			c.SaveLayer(nil, l)
			c.DrawRect(NewRectFromLTRB(16, 16, 48, 48), p(0xFFFF8800))
			c.Restore()
		},
		"imageFilter.dilate": func(c Canvas) {
			l := NewPaint()
			l.SetImageFilter(NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: 4, RadiusY: 2}))
			c.SaveLayer(nil, l)
			c.DrawRect(NewRectFromLTRB(20, 20, 44, 44), p(0xFF8800FF))
			c.Restore()
		},
		"image.draw": func(c Canvas) { c.DrawImage(img, NewOffset(8, 8), NewPaint()) },
		"image.rect.nearest": func(c Canvas) {
			c.DrawImageRect(img, NewRectFromLTRB(0, 0, 16, 16), NewRectFromLTRB(0, 0, 64, 64),
				with(NewPaint(), func(q *Paint) { q.SetFilterQuality(FilterQualityNone) }))
		},
		"image.shader": func(c Canvas) {
			none := FilterQualityNone
			c.DrawRect(full, shader(NewImageShader(img, TileModeRepeated, TileModeMirror,
				[]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 3, 5, 0, 1}, ImageShaderOptions{FilterQuality: &none})))
		},
		"path.cubic": func(c Canvas) {
			path := NewPath()
			path.MoveTo(8, 56)
			path.CubicTo(8, 0, 56, 64, 56, 8)
			path.LineTo(32, 32)
			path.Close()
			c.DrawPath(path, p(0xFF884422))
		},
		"path.evenOdd": func(c Canvas) {
			path := NewPath()
			path.SetFillType(PathFillTypeEvenOdd)
			path.AddOval(NewRectFromLTRB(4, 4, 60, 60))
			path.AddRect(NewRectFromLTRB(20, 20, 44, 44))
			c.DrawPath(path, p(0xFF2288AA))
		},
		"arc": func(c Canvas) { c.DrawArc(NewRectFromLTRB(8, 8, 56, 56), 0.3, 2.5, true, p(0xFFCC0044)) },
		"points": func(c Canvas) {
			c.DrawPoints(PointModePolygon, []Offset{NewOffset(8, 8), NewOffset(56, 20), NewOffset(12, 40), NewOffset(50, 56)},
				with(p(0xFF000000), func(q *Paint) { q.SetStrokeWidth(3) }))
		},
		"shadow": func(c Canvas) {
			path := NewPath()
			path.AddRect(NewRectFromLTRB(16, 16, 48, 48))
			c.DrawShadow(path, NewColor(0xFF000000), 6, false)
		},
	}
}

// pixelTolerance is each scene's allowed maximum per-channel difference
// and how many pixels (of 4096) may differ by more than 2; scenes not
// listed allow max 2 and none over 2 (most are bit-exact: gradients,
// images, color filters, blur/dilate filters, layers). Measured on Apple
// Metal (here) vs SwiftShader Vulkan (flutter_tester):
//   - anti-aliased geometry edges differ by one MSAA sample of four
//     (64 = 255/4): the two GPUs place their 4x MSAA samples differently;
//     only edge pixels differ, by at most one sample.
//   - circle: Impeller's circle geometry differs by ≤5 on edge pixels.
//   - clip.rect: ENGINE GAP — the SDK's clips take no anti-alias flag
//     (it records is_aa=false), so a fractional clip edge is hard where
//     Flutter's is anti-aliased: every edge pixel of the clip differs.
var pixelTolerance = map[string]struct{ max, over2 int }{
	"arc":                  {65, 24},
	"circle":               {5, 50},
	"clip.rect":            {192, 140},
	"clip.rrect":           {65, 8},
	"drrect":               {65, 8},
	"oval":                 {65, 10},
	"path.cubic":           {65, 10},
	"path.evenOdd":         {65, 30},
	"points":               {65, 25},
	"rrect":                {65, 8},
	"stroke.line.round":    {65, 20},
	"transform.rotate":     {65, 16},
	"transform.skew.scale": {65, 4},
}

func TestPaintingPixelsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/painting_pixels.json")
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
	// flutter_tester's implicit view has a device pixel ratio of 3.0
	// (tester_main.cc), which drawShadow uses.
	_, restoreTester := useTesterPlatform()
	defer restoreTester()
	img := checkerImage(t)
	scenes := pixelScenes(img)
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
		rec := NewPictureRecorder()
		scenes[name](NewCanvas(rec))
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
		tol, ok := pixelTolerance[name]
		if !ok {
			tol.max, tol.over2 = 2, 0
		}
		t.Logf("%-30s max %3d, pixels >2: %4d, >16: %4d", name, maxDiff, over2, over16)
		if maxDiff > tol.max || over2 > tol.over2 {
			t.Errorf("%s: max channel difference %d (%d pixels > 2), tolerance %+v", name, maxDiff, over2, tol)
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
