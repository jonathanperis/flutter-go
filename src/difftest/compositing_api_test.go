package difftest

// Differential test of dart:ui's compositing API against Flutter 3.50.1
// (flutter/test/compositing_api_test.dart): SceneBuilder/Scene/EngineLayer
// behavior computed without rasterizing — runtime types, the retained-
// layer debug checks and their messages (layer hashes masked), argument
// checks, use after build/dispose, FlutterView.render outside a frame.
// Compared exactly. Not compared: FlutterView.render of a disposed scene,
// which crashes flutter_tester (a null native scene; Go panics with the
// "native peer has been collected" StateError).

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"regexp"
	"testing"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
)

var layerHashRe = regexp.MustCompile(`#[0-9a-f]{5}`)

func maskedErr(f func()) string { return layerHashRe.ReplaceAllString(dartErr(f), "#HASH") }

// runtimeTypeName is Dart's `${layer.runtimeType}`.
func runtimeTypeName(l EngineLayer) string {
	switch l.(type) {
	case *OffsetEngineLayer:
		return "OffsetEngineLayer"
	case *TransformEngineLayer:
		return "TransformEngineLayer"
	case *ClipRectEngineLayer:
		return "ClipRectEngineLayer"
	case *ClipRRectEngineLayer:
		return "ClipRRectEngineLayer"
	case *ClipRSuperellipseEngineLayer:
		return "ClipRSuperellipseEngineLayer"
	case *ClipPathEngineLayer:
		return "ClipPathEngineLayer"
	case *OpacityEngineLayer:
		return "OpacityEngineLayer"
	case *ColorFilterEngineLayer:
		return "ColorFilterEngineLayer"
	case *ImageFilterEngineLayer:
		return "ImageFilterEngineLayer"
	case *BackdropFilterEngineLayer:
		return "BackdropFilterEngineLayer"
	case *ShaderMaskEngineLayer:
		return "ShaderMaskEngineLayer"
	}
	return "?"
}

func smallPicture() *Picture {
	rec := NewPictureRecorder()
	p := NewPaint()
	p.SetColor(NewColor(0xFF00FF00))
	NewCanvas(rec).DrawRect(NewRectFromLTRB(0, 0, 4, 4), p)
	return rec.EndRecording()
}

func old[T any](l EngineLayer) *T {
	if l == nil {
		return nil
	}
	return any(l).(*T)
}

func compositingAPICases() map[string]any {
	out := map[string]any{}
	identity := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	type push func(b SceneBuilder, o EngineLayer) EngineLayer
	pushes := map[string]push{
		"pushOffset": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushOffset(1, 2, PushOffsetOptions{OldLayer: old[OffsetEngineLayer](o)})
		},
		"pushTransform": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushTransform(identity, PushTransformOptions{OldLayer: old[TransformEngineLayer](o)})
		},
		"pushClipRect": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushClipRect(RectZero, PushClipRectOptions{OldLayer: old[ClipRectEngineLayer](o)})
		},
		"pushClipRRect": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushClipRRect(RRectZero, PushClipRRectOptions{OldLayer: old[ClipRRectEngineLayer](o)})
		},
		"pushClipRSuperellipse": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushClipRSuperellipse(RSuperellipseZero, PushClipRSuperellipseOptions{OldLayer: old[ClipRSuperellipseEngineLayer](o)})
		},
		"pushClipPath": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushClipPath(NewPath(), PushClipPathOptions{OldLayer: old[ClipPathEngineLayer](o)})
		},
		"pushOpacity": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushOpacity(100, PushOpacityOptions{OldLayer: old[OpacityEngineLayer](o)})
		},
		"pushColorFilter": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushColorFilter(NewColorFilterMode(NewColor(0x80000000), BlendModeSrcIn),
				PushColorFilterOptions{OldLayer: old[ColorFilterEngineLayer](o)})
		},
		"pushImageFilter": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 2}),
				PushImageFilterOptions{OldLayer: old[ImageFilterEngineLayer](o)})
		},
		"pushBackdropFilter": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushBackdropFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 2}),
				PushBackdropFilterOptions{OldLayer: old[BackdropFilterEngineLayer](o)})
		},
		"pushShaderMask": func(b SceneBuilder, o EngineLayer) EngineLayer {
			return b.PushShaderMask(NewGradientLinear(OffsetZero, NewOffset(1, 0), []Color{NewColor(0xFF000000), NewColor(0x00000000)}),
				RectZero, BlendModeDstIn, PushShaderMaskOptions{OldLayer: old[ShaderMaskEngineLayer](o)})
		},
	}
	for name, push := range pushes {
		var log []any
		b1 := NewSceneBuilder()
		layer := push(b1, nil)
		child := b1.PushOpacity(10)
		b1.Pop()
		b1.Pop()
		b1.Build()
		log = append(log, runtimeTypeName(layer))
		b2 := NewSceneBuilder()
		log = append(log, maskedErr(func() { b2.AddRetained(layer) }))
		log = append(log, maskedErr(func() { push(b2, layer) }))
		log = append(log, maskedErr(func() { b2.PushOpacity(1, PushOpacityOptions{OldLayer: child}) }))
		log = append(log, maskedErr(func() { b2.AddRetained(layer) }))
		b2.Build()
		b3 := NewSceneBuilder()
		log = append(log, maskedErr(func() { push(b3, layer) }))
		log = append(log, maskedErr(func() { b3.AddRetained(layer) }))
		log = append(log, maskedErr(func() { push(b3, layer) }))
		b3.Build()
		log = append(log, maskedErr(func() { NewSceneBuilder().AddRetained(layer) }))
		log = append(log, maskedErr(func() { push(NewSceneBuilder(), layer) }))
		b4 := NewSceneBuilder()
		fresh := push(b4, nil)
		b4.Build()
		log = append(log, maskedErr(fresh.Dispose))
		log = append(log, maskedErr(fresh.Dispose))
		log = append(log, maskedErr(func() { push(NewSceneBuilder(), fresh) }))
		log = append(log, maskedErr(func() { NewSceneBuilder().AddRetained(fresh) }))
		out["layer."+name] = log
	}

	builder := NewSceneBuilder()
	out["sceneBuilder.toString"] = builder.(interface{ String() string }).String()
	nanM := append([]float64(nil), identity...)
	nanM[15] = math.NaN()
	infM := append([]float64(nil), identity...)
	infM[12] = inf()
	out["sceneBuilder.matrix"] = []any{
		dartErr(func() { builder.PushTransform(make([]float64, 15)) }),
		dartErr(func() { builder.PushTransform(nanM) }),
		dartErr(func() { builder.PushTransform(infM) }),
		dartErr(func() { builder.PushTransform(make([]float64, 16)) }),
	}
	none, hard := ClipNone, ClipHardEdge
	out["sceneBuilder.clipNone"] = []any{
		dartErr(func() { builder.PushClipRect(RectZero, PushClipRectOptions{ClipBehavior: &none}) }),
		dartErr(func() { builder.PushClipRRect(RRectZero, PushClipRRectOptions{ClipBehavior: &none}) }),
		dartErr(func() {
			builder.PushClipRSuperellipse(RSuperellipseZero, PushClipRSuperellipseOptions{ClipBehavior: &none})
		}),
		dartErr(func() { builder.PushClipPath(NewPath(), PushClipPathOptions{ClipBehavior: &none}) }),
		dartErr(func() { builder.PushClipRect(RectZero, PushClipRectOptions{ClipBehavior: &hard}) }),
	}
	disposed := smallPicture()
	disposed.Dispose()
	out["sceneBuilder.leaves"] = []any{
		dartErr(func() { builder.AddPicture(OffsetZero, disposed) }),
		dartErr(func() {
			builder.AddPicture(NewOffset(1, 2), smallPicture(), AddPictureOptions{IsComplexHint: true, WillChangeHint: true})
		}),
		dartErr(func() {
			builder.AddTexture(3, AddTextureOptions{Offset: NewOffset(1, 1), Width: 2, Height: 2, Freeze: true})
		}),
		dartErr(func() { builder.AddPlatformView(4, AddPlatformViewOptions{Width: 5, Height: 6}) }),
		dartErr(func() { builder.AddPerformanceOverlay(15, NewRectFromLTRB(0, 0, 10, 10)) }),
		dartErr(builder.Pop),
		dartErr(builder.Pop),
		dartErr(builder.Pop),
	}
	scene := builder.Build()
	out["scene.toString"] = scene.(interface{ String() string }).String()
	out["sceneBuilder.afterBuild"] = []any{
		dartErr(func() { builder.PushOffset(0, 0) }),
		dartErr(builder.Pop),
		dartErr(func() { builder.AddPicture(OffsetZero, smallPicture()) }),
		dartErr(func() { builder.Build() }),
		builder.(interface{ String() string }).String(),
	}
	out["scene.errors"] = []any{
		dartErr(func() { scene.ToImageSync(0, 1) }),
		dartErr(func() { scene.ToImageSync(1, -1) }),
		dartErr(func() { scene.ToImage(0, 0) }),
	}
	image := scene.ToImageSync(3, 2)
	out["scene.toImageSync"] = []any{image.Width, image.Height}
	view := PlatformDispatcherInstance().ImplicitView()
	size := NewSize(10, 10)
	out["render.outsideFrame"] = []any{
		dartErr(func() { view.Render(scene) }),
		dartErr(func() { view.Render(scene, FlutterViewRenderOptions{Size: &size}) }),
	}
	scene.Dispose()
	out["scene.afterDispose"] = []any{
		dartErr(func() { scene.ToImageSync(1, 1) }),
		dartErr(func() { scene.ToImage(1, 1) }),
		dartErr(scene.Dispose),
		dartErr(func() { scene.ToImageSync(0, 1) }),
		scene.(interface{ String() string }).String(),
	}
	return out
}

func TestCompositingAPIMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/compositing_api.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if EngineImpellerContext() == nil {
		t.Fatal("no Metal device: Scene.toImageSync needs one")
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	_, restore := useTesterPlatform()
	defer restore()
	cases := compositingAPICases()
	if len(ref) != len(cases) {
		t.Errorf("reference has %d cases, Go has %d", len(ref), len(cases))
	}
	for name, got := range cases {
		want, ok := ref[name]
		if !ok {
			t.Errorf("%s: missing from the reference", name)
			continue
		}
		if g := toJSONShape(t, got); !reflect.DeepEqual(g, want) { // exact
			t.Errorf("%s:\n Go:      %v\n Flutter: %v", name, g, want)
		}
	}
}
