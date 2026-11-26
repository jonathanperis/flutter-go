package ui

// Port of engine/src/flutter/testing/dart/compositing_test.dart (Flutter
// 3.50.1), all tests, plus checks of the layer tree's painting
// (LayerStateStack) and of what the engine natives record.
//
// Skipped: none. Adapted: Dart's `oldLayer as XEngineLayer?` casts of the
// shared helper are Go type assertions in each push function.

import (
	"math"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestSceneToImageSyncSucceeds(t *testing.T) {
	needGPU(t)
	recorder := NewPictureRecorder()
	canvas := NewCanvas(recorder)
	color := NewColor(0xFF123456)
	p := NewPaint()
	p.SetColor(color)
	canvas.DrawPaint(p)
	picture := recorder.EndRecording()
	builder := NewSceneBuilder()
	builder.PushOffset(10, 10)
	builder.AddPicture(NewOffset(5, 5), picture)
	scene := builder.Build()

	image := scene.ToImageSync(6, 8)
	picture.Dispose()
	scene.Dispose()

	if image.Width != 6 || image.Height != 8 {
		t.Fatalf("size %dx%d", image.Width, image.Height)
	}
	data := awaitValue(t, image.ToByteData())
	if len(data) != 6*8*4 {
		t.Fatalf("length %d", len(data))
	}
	if data[0] != 0x12 || data[1] != 0x34 || data[2] != 0x56 || data[3] != 0xFF {
		t.Fatalf("pixel %v", data[:4])
	}
}

func TestSceneToImageSyncSucceedsWithTextureLayer(t *testing.T) {
	needGPU(t)
	builder := NewSceneBuilder()
	builder.PushOffset(10, 10)
	builder.AddTexture(0, AddTextureOptions{Width: 10, Height: 10})
	scene := builder.Build()
	image := scene.ToImageSync(20, 20)
	scene.Dispose()
	if image.Width != 20 || image.Height != 20 {
		t.Fatalf("size %dx%d", image.Width, image.Height)
	}
	data := awaitValue(t, image.ToByteData())
	if len(data) != 20*20*4 || data[0] != 0 || data[1] != 0 || data[2] != 0 || data[3] != 0 {
		t.Fatalf("pixel %v", data[:4])
	}
}

func isAssertion(r any) bool { _, ok := r.(dart.AssertionFailure); return ok }

func TestAddPictureWithDisposedPictureDoesNotCrash(t *testing.T) {
	recorder := NewPictureRecorder()
	NewCanvas(recorder).DrawPaint(NewPaint())
	picture := recorder.EndRecording()
	picture.Dispose()
	if !picture.DebugDisposed() {
		t.Fatal("debugDisposed")
	}
	builder := NewSceneBuilder()
	if r := panics(func() { builder.AddPicture(OffsetZero, picture) }); !isAssertion(r) {
		t.Fatalf("got %v, want an AssertionError", r)
	}
	builder.Build().Dispose()
}

func TestPushTransformValidatesTheMatrix(t *testing.T) {
	builder := NewSceneBuilder()
	if builder.PushTransform([]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}) == nil {
		t.Fatal("nil layer")
	}
	for name, m := range map[string][]float64{
		"wrong length": {1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0},
		"NaN":          {1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, math.NaN()},
		"infinity":     {1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, math.Inf(1)},
	} {
		if r := panics(func() { builder.PushTransform(m) }); !isAssertion(r) {
			t.Errorf("%s: got %v, want an AssertionError", name, r)
		}
	}
}

func TestSceneBuilderAcceptsTypedLayers(t *testing.T) {
	builder1 := NewSceneBuilder()
	opacity1 := builder1.PushOpacity(100)
	if opacity1 == nil {
		t.Fatal("nil")
	}
	builder1.Pop()
	builder1.Build()

	builder2 := NewSceneBuilder()
	opacity2 := builder2.PushOpacity(200, PushOpacityOptions{OldLayer: opacity1})
	if opacity2 == nil {
		t.Fatal("nil")
	}
	builder2.Pop()
	builder2.Build()
}

// pushFunction is _TestNoSharingFunction.
type pushFunction func(builder SceneBuilder, oldLayer EngineLayer) EngineLayer

func expectAssertionContaining(t *testing.T, name, want string, f func()) {
	t.Helper()
	r := panics(f)
	e, ok := r.(dart.AssertionFailure)
	if !ok || !strings.Contains(e.Error(), want) {
		t.Errorf("%s: got %v, want an AssertionError containing %q", name, r, want)
	}
}

func testNoSharing(t *testing.T, name string, push pushFunction) {
	t.Helper()
	const alreadyUsed = "The layer is already being used"
	const previouslyOld = "was previously used as oldLayer"

	// Uses the same layer first as oldLayer then in addRetained.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		push(builder2, layer)
		builder2.Pop()
		expectAssertionContaining(t, name+" pushThenIllegalRetain", alreadyUsed, func() { builder2.AddRetained(layer) })
		builder2.Build()
	}
	// Uses the same layer first in addRetained then as oldLayer.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		builder2.AddRetained(layer)
		expectAssertionContaining(t, name+" addRetainedThenIllegalPush", alreadyUsed, func() { push(builder2, layer) })
		builder2.Build()
	}
	// Retains the same layer twice in the same scene.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		builder2.AddRetained(layer)
		expectAssertionContaining(t, name+" doubleAddRetained", alreadyUsed, func() { builder2.AddRetained(layer) })
		builder2.Build()
	}
	// Uses the same layer as oldLayer twice in the same scene.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		push(builder2, layer)
		expectAssertionContaining(t, name+" pushOldLayerTwice", previouslyOld, func() { push(builder2, layer) })
		builder2.Build()
	}
	// Uses a child of a retained layer as an oldLayer.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		childLayer := builder1.PushOpacity(123)
		builder1.Pop()
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		builder2.AddRetained(layer)
		expectAssertionContaining(t, name+" pushChildLayerOfRetainedLayer", alreadyUsed, func() {
			builder2.PushOpacity(321, PushOpacityOptions{OldLayer: childLayer})
		})
		builder2.Build()
	}
	// Retains a layer whose child is already used as oldLayer.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		childLayer := builder1.PushOpacity(123)
		builder1.Pop()
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		builder2.PushOpacity(234, PushOpacityOptions{OldLayer: childLayer})
		builder2.Pop()
		expectAssertionContaining(t, name+" retainParentLayerOfPushedChild", alreadyUsed, func() { builder2.AddRetained(layer) })
		builder2.Build()
	}
	// Retains a layer used as oldLayer in a previous frame.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		push(builder2, layer)
		builder2.Pop()
		expectAssertionContaining(t, name+" retainOldLayer", previouslyOld, func() {
			builder3 := NewSceneBuilder()
			builder3.AddRetained(layer)
		})
		builder2.Build()
	}
	// Passes as oldLayer a layer used as oldLayer in a previous frame.
	{
		builder1 := NewSceneBuilder()
		layer := push(builder1, nil)
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		push(builder2, layer)
		builder2.Pop()
		expectAssertionContaining(t, name+" pushOldLayer", previouslyOld, func() {
			builder3 := NewSceneBuilder()
			push(builder3, layer)
		})
		builder2.Build()
	}
	// Retains a parent of a layer used as oldLayer in a previous frame.
	{
		builder1 := NewSceneBuilder()
		parentLayer := push(builder1, nil)
		childLayer := builder1.PushOpacity(123)
		builder1.Pop()
		builder1.Pop()
		builder1.Build()
		builder2 := NewSceneBuilder()
		builder2.PushOpacity(321, PushOpacityOptions{OldLayer: childLayer})
		builder2.Pop()
		expectAssertionContaining(t, name+" retainsParentOfOldLayer", previouslyOld, func() {
			builder3 := NewSceneBuilder()
			builder3.AddRetained(parentLayer)
		})
		builder2.Build()
	}
}

// asOld is Dart's `oldLayer as T?`.
func asOld[T any](l EngineLayer) *T {
	if l == nil {
		return nil
	}
	return any(l).(*T)
}

func TestSceneBuilderDoesNotShareALayerBetweenAddRetainedAndPush(t *testing.T) {
	identity := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	cases := map[string]pushFunction{
		"pushOffset": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushOffset(0, 0, PushOffsetOptions{OldLayer: asOld[OffsetEngineLayer](old)})
		},
		"pushTransform": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushTransform(make([]float64, 16), PushTransformOptions{OldLayer: asOld[TransformEngineLayer](old)})
		},
		"pushClipRect": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushClipRect(RectZero, PushClipRectOptions{OldLayer: asOld[ClipRectEngineLayer](old)})
		},
		"pushClipRRect": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushClipRRect(RRectZero, PushClipRRectOptions{OldLayer: asOld[ClipRRectEngineLayer](old)})
		},
		"pushClipPath": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushClipPath(NewPath(), PushClipPathOptions{OldLayer: asOld[ClipPathEngineLayer](old)})
		},
		"pushOpacity": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushOpacity(100, PushOpacityOptions{OldLayer: asOld[OpacityEngineLayer](old)})
		},
		"pushBackdropFilter": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushBackdropFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1}),
				PushBackdropFilterOptions{OldLayer: asOld[BackdropFilterEngineLayer](old)})
		},
		"pushShaderMask": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushShaderMask(NewGradientRadial(OffsetZero, 10,
				[]Color{NewColorFromARGB(0, 0, 0, 0), NewColorFromARGB(0, 255, 255, 255)}),
				RectZero, BlendModeColor, PushShaderMaskOptions{OldLayer: asOld[ShaderMaskEngineLayer](old)})
		},
		"pushColorFilter.mode": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushColorFilter(NewColorFilterMode(NewColorFromARGB(0, 0, 0, 0), BlendModeColor),
				PushColorFilterOptions{OldLayer: asOld[ColorFilterEngineLayer](old)})
		},
		"pushColorFilter.matrix": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushColorFilter(NewColorFilterMatrix([]float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0}),
				PushColorFilterOptions{OldLayer: asOld[ColorFilterEngineLayer](old)})
		},
		"pushColorFilter.linearToSrgbGamma": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushColorFilter(NewColorFilterLinearToSrgbGamma(), PushColorFilterOptions{OldLayer: asOld[ColorFilterEngineLayer](old)})
		},
		"pushColorFilter.srgbToLinearGamma": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushColorFilter(NewColorFilterSrgbToLinearGamma(), PushColorFilterOptions{OldLayer: asOld[ColorFilterEngineLayer](old)})
		},
		"pushImageFilter.blur": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 10, SigmaY: 10}),
				PushImageFilterOptions{OldLayer: asOld[ImageFilterEngineLayer](old)})
		},
		"pushImageFilter.dilate": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushImageFilter(NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: 10, RadiusY: 10}),
				PushImageFilterOptions{OldLayer: asOld[ImageFilterEngineLayer](old)})
		},
		"pushImageFilter.erode": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushImageFilter(NewImageFilterErode(ImageFilterRadiusOptions{RadiusX: 10, RadiusY: 10}),
				PushImageFilterOptions{OldLayer: asOld[ImageFilterEngineLayer](old)})
		},
		"pushImageFilter.matrix": func(b SceneBuilder, old EngineLayer) EngineLayer {
			return b.PushImageFilter(NewImageFilterMatrix(identity), PushImageFilterOptions{OldLayer: asOld[ImageFilterEngineLayer](old)})
		},
	}
	for name, push := range cases {
		testNoSharing(t, name, push)
	}
}

// ---- beyond the Dart tests: what the natives build

func TestSceneBuilderLayerTree(t *testing.T) {
	b := NewSceneBuilder().(*nativeSceneBuilder)
	off := b.PushOffset(3, 4)
	b.PushClipRect(NewRectFromLTRB(0, 0, 10, 10), PushClipRectOptions{ClipBehavior: ptrTo(ClipHardEdge)})
	b.AddTexture(7, AddTextureOptions{Offset: NewOffset(1, 2), Width: 5, Height: 6})
	b.Pop()
	b.Pop()
	b.Pop() // the root is never popped
	b.AddPlatformView(9, AddPlatformViewOptions{Width: 1, Height: 1})
	scene := b.Build().(*nativeScene)
	root := scene.root
	if len(root.layers) != 2 {
		t.Fatalf("root has %d layers", len(root.layers))
	}
	tl, ok := root.layers[0].(*transformLayer)
	if !ok || tl.transform[12] != 3 || tl.transform[13] != 4 || off.nativeLayer.layer != tl {
		t.Fatalf("offset layer %#v", root.layers[0])
	}
	cl := tl.layers[0].(*clipShapeLayer)
	if cl.clipBehavior != ClipHardEdge || cl.clipBounds != (dlRect{0, 0, 10, 10}) {
		t.Fatalf("clip %#v", cl)
	}
	tex := cl.layers[0].(*textureLayer)
	if tex.rect != (dlRect{1, 2, 6, 8}) || tex.textureID != 7 {
		t.Fatalf("texture %#v", tex)
	}
	if pv := root.layers[1].(*platformViewLayer); pv.viewID != 9 {
		t.Fatalf("platform view %#v", pv)
	}
	// Preroll: the texture's bounds through the clip and the offset.
	var ctx prerollContext
	root.preroll(&ctx)
	if tl.paintBounds.r != (dlRect{4, 6, 9, 12}) || tl.paintBounds.unbounded {
		t.Fatalf("paint bounds %#v", tl.paintBounds)
	}
	// After build, the builder's native peer is gone.
	expectPanic(t, "native peer has been collected", func() { b.PushOffset(0, 0) })
	expectPanic(t, "native peer has been collected", func() { b.Build() })
	if b.String() != "SceneBuilder" || scene.String() != "Scene" {
		t.Fatal("toString")
	}
}

func TestSceneDisposeAndErrors(t *testing.T) {
	scene := NewSceneBuilder().Build()
	expectPanic(t, "Invalid image dimensions.", func() { scene.ToImageSync(0, 1) })
	expectPanic(t, "Invalid image dimensions.", func() { scene.ToImage(1, -1) })
	scene.Dispose()
	expectPanic(t, "native peer has been collected", func() { scene.ToImageSync(1, 1) })
	expectPanic(t, "native peer has been collected", func() { scene.ToImage(1, 1) })
	expectPanic(t, "native peer has been collected", func() { scene.Dispose() })
	// Dimensions are checked before the native peer.
	expectPanic(t, "Invalid image dimensions.", func() { scene.ToImageSync(1, 0) })
}

func TestEngineLayerDispose(t *testing.T) {
	b := NewSceneBuilder()
	l := b.PushOpacity(10)
	l.Dispose()
	if l.nativeLayer != nil {
		t.Fatal("the wrapper keeps its native layer")
	}
	expectAssertionContaining(t, "second dispose", "Object disposed", l.Dispose)
	expectAssertionContaining(t, "oldLayer after dispose", "Object disposed", func() {
		NewSceneBuilder().PushOpacity(10, PushOpacityOptions{OldLayer: l})
	})
}

func TestUsedOnceMessages(t *testing.T) {
	b1 := NewSceneBuilder()
	layer := b1.PushOpacity(1)
	b1.Pop()
	b1.Build()
	b2 := NewSceneBuilder()
	b2.PushOpacity(2, PushOpacityOptions{OldLayer: layer})
	r := panics(func() { b2.AddRetained(layer) })
	want := "Layer OpacityEngineLayer already used.\nThe layer is already being used as oldLayer in pushOpacity in this scene.\nA layer may only be used once in a given scene."
	if e, ok := r.(dart.AssertionFailure); !ok || e.AssertionMessage() != want {
		t.Fatalf("got %v", r)
	}
	r = panics(func() { NewSceneBuilder().AddRetained(layer) })
	msg, _ := r.(dart.AssertionFailure).AssertionMessage().(string)
	if !strings.HasPrefix(msg, "Layer OpacityEngineLayer#") || !strings.Contains(msg, " was previously used as oldLayer.\nOnce a layer is used as oldLayer, it may not be used again. Instead, after calling one of the SceneBuilder.push* methods and passing an oldLayer to it, use the layer returned by the method as oldLayer in subsequent frames.") {
		t.Fatalf("got %q", msg)
	}
	hash := msg[len("Layer OpacityEngineLayer#"):strings.Index(msg, " was")]
	if len(hash) != 5 {
		t.Fatalf("hash %q", hash)
	}
	for _, clip := range []Clip{ClipNone} {
		if r := panics(func() { NewSceneBuilder().PushClipRect(RectZero, PushClipRectOptions{ClipBehavior: &clip}) }); !isAssertion(r) {
			t.Fatalf("Clip.none: %v", r)
		}
	}
}

// recordingCanvasOps paints a scene's tree into a display list and counts
// the canvas operations, checking the LayerStateStack's decisions.
func TestLayerStateStackSavesLayersOnlyWhenNeeded(t *testing.T) {
	rec := NewPictureRecorder()
	NewCanvas(rec).DrawPaint(NewPaint())
	pic := rec.EndRecording()
	defer pic.Dispose()

	count := func(build func(b SceneBuilder)) (saveCount int, ops int) {
		b := NewSceneBuilder()
		build(b)
		tree := b.Build().(*nativeScene).buildLayerTree(10, 10)
		r := NewPictureRecorder()
		c := NewCanvas(r, NewRectFromLTWH(0, 0, 10, 10)).(*nativeCanvas)
		var pre prerollContext
		tree.root.preroll(&pre)
		stack := newLayerStateStack(c)
		maxDepth := 0
		ctx := &paintContext{stack: stack, canvas: c}
		tree.root.paint(ctx)
		maxDepth = c.ops
		r.EndRecording().Dispose()
		return len(stack.entries), maxDepth
	}
	// Opacity over a picture: one saveLayer and the picture.
	if entries, ops := count(func(b SceneBuilder) {
		b.PushOpacity(128)
		b.AddPicture(OffsetZero, pic)
	}); entries != 0 || ops != 2 {
		t.Errorf("opacity: entries %d, ops %d", entries, ops)
	}
	// Opacity over a texture: the texture takes the opacity, no saveLayer.
	if _, ops := count(func(b SceneBuilder) {
		b.PushOpacity(128)
		b.AddTexture(1, AddTextureOptions{Width: 1, Height: 1})
	}); ops != 0 {
		t.Errorf("opacity over texture: ops %d", ops)
	}
	// Two nested opacities over a picture: one saveLayer with the product.
	if _, ops := count(func(b SceneBuilder) {
		b.PushOpacity(128)
		b.PushOpacity(128)
		b.AddPicture(OffsetZero, pic)
	}); ops != 2 {
		t.Errorf("nested opacity: ops %d", ops)
	}
	// A clip with Clip.antiAliasWithSaveLayer under an opacity: the clip,
	// then the clip's saveLayer takes the opacity.
	if _, ops := count(func(b SceneBuilder) {
		b.PushOpacity(128)
		b.PushClipRect(NewRectFromLTRB(1, 1, 5, 5), PushClipRectOptions{ClipBehavior: ptrTo(ClipAntiAliasWithSaveLayer)})
		b.AddPicture(OffsetZero, pic)
	}); ops != 3 {
		t.Errorf("clip with save layer: ops %d", ops)
	}
	// A blend-mode color filter that doesn't commute with opacity: a
	// saveLayer for the opacity, one for the filter.
	if _, ops := count(func(b SceneBuilder) {
		b.PushOpacity(128)
		b.PushColorFilter(NewColorFilterMode(NewColor(0xFF00FF00), BlendModeSrcOver))
		b.AddPicture(OffsetZero, pic)
	}); ops != 3 {
		t.Errorf("non-commuting color filter: ops %d", ops)
	}
	// One that commutes: a single saveLayer with both.
	if _, ops := count(func(b SceneBuilder) {
		b.PushOpacity(128)
		b.PushColorFilter(NewColorFilterMode(NewColor(0xFF00FF00), BlendModeSrcIn))
		b.AddPicture(OffsetZero, pic)
	}); ops != 2 {
		t.Errorf("commuting color filter: ops %d", ops)
	}
	// An opacity of 0: nothing is painted.
	if _, ops := count(func(b SceneBuilder) {
		b.PushOpacity(0)
		b.AddPicture(OffsetZero, pic)
	}); ops != 2 {
		t.Errorf("zero opacity: ops %d", ops)
	}
}

func TestClipBehaviorDefaultsToAntiAlias(t *testing.T) {
	b := NewSceneBuilder()
	layers := []EngineLayer{
		b.PushClipRect(RectZero), b.PushClipRRect(RRectZero),
		b.PushClipRSuperellipse(RSuperellipseZero), b.PushClipPath(NewPath()),
	}
	for i, l := range layers {
		var w *engineLayerWrapper
		switch x := l.(type) {
		case *ClipRectEngineLayer:
			w = &x.engineLayerWrapper
		case *ClipRRectEngineLayer:
			w = &x.engineLayerWrapper
		case *ClipRSuperellipseEngineLayer:
			w = &x.engineLayerWrapper
		case *ClipPathEngineLayer:
			w = &x.engineLayerWrapper
		}
		if c := w.nativeLayer.layer.(*clipShapeLayer).clipBehavior; c != ClipAntiAlias {
			t.Errorf("layer %d: %v", i, c)
		}
	}
}

// paintOps prerolls and paints a scene's tree and returns the canvas
// operations recorded.
func paintOps(build func(b SceneBuilder)) (*containerLayer, int) {
	b := NewSceneBuilder()
	build(b)
	root := b.Build().(*nativeScene).root
	r := NewPictureRecorder()
	c := NewCanvas(r, NewRectFromLTWH(0, 0, 10, 10)).(*nativeCanvas)
	var pre prerollContext
	root.preroll(&pre)
	ctx := &paintContext{stack: newLayerStateStack(c), canvas: c}
	if needsPainting(root, ctx) {
		root.paint(ctx)
	}
	ops := c.ops
	r.EndRecording().Dispose()
	return root, ops
}

func TestOpacityInheritanceAndCulling(t *testing.T) {
	tex := func(b SceneBuilder, x float64) {
		b.AddTexture(1, AddTextureOptions{Offset: NewOffset(x, 0), Width: 4, Height: 4})
	}
	// Non-overlapping textures take the opacity: no saveLayer.
	if _, ops := paintOps(func(b SceneBuilder) { b.PushOpacity(100); tex(b, 0); tex(b, 5) }); ops != 0 {
		t.Errorf("non-overlapping: %d ops", ops)
	}
	// Overlapping ones can't: a saveLayer.
	if _, ops := paintOps(func(b SceneBuilder) { b.PushOpacity(100); tex(b, 0); tex(b, 2) }); ops != 1 {
		t.Errorf("overlapping: %d ops", ops)
	}
	// The opacity layer's bounds are its children's, offset.
	root, _ := paintOps(func(b SceneBuilder) {
		b.PushOpacity(100, PushOpacityOptions{Offset: NewOffset(3, 7)})
		tex(b, 0)
	})
	if pb := root.layers[0].base().paintBounds; pb.unbounded || pb.r != (dlRect{3, 7, 7, 11}) {
		t.Errorf("opacity bounds %#v", pb)
	}
	// A clip with an empty rect culls its picture.
	rec := NewPictureRecorder()
	NewCanvas(rec).DrawPaint(NewPaint())
	pic := rec.EndRecording()
	defer pic.Dispose()
	if _, ops := paintOps(func(b SceneBuilder) { b.PushClipRect(RectZero); b.AddPicture(OffsetZero, pic) }); ops != 0 {
		t.Errorf("culled clip: %d ops", ops)
	}
	if _, ops := paintOps(func(b SceneBuilder) { b.PushClipRect(NewRectFromLTRB(0, 0, 1, 1)); b.AddPicture(OffsetZero, pic) }); ops != 2 {
		t.Errorf("clip: %d ops", ops)
	}
}

func TestColorFilterCanCommuteWithOpacity(t *testing.T) {
	for mode, want := range map[BlendMode]bool{BlendModeClear: true, BlendModeDst: true, BlendModeSrcIn: true,
		BlendModeDstIn: true, BlendModeDstOut: true, BlendModeSrcATop: true, BlendModeModulate: true,
		BlendModeSrcOver: false, BlendModeSrc: false, BlendModeMultiply: false} {
		if got := colorFilterCanCommuteWithOpacity(NewColorFilterMode(NewColor(0xFF000000), mode)); got != want {
			t.Errorf("%v: %v", mode, got)
		}
	}
	m := []float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0}
	if !colorFilterCanCommuteWithOpacity(NewColorFilterMatrix(m)) {
		t.Error("identity matrix")
	}
	for _, i := range []int{3, 8, 13, 15, 16, 17, 19} {
		mm := append([]float64(nil), m...)
		mm[i] = 0.5
		if colorFilterCanCommuteWithOpacity(NewColorFilterMatrix(mm)) {
			t.Errorf("matrix[%d] != 0 commutes", i)
		}
	}
	if !colorFilterCanCommuteWithOpacity(NewColorFilterLinearToSrgbGamma()) {
		t.Error("gamma")
	}
}
