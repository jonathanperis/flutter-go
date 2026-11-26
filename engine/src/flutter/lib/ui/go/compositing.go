package ui

// Port of engine/src/flutter/lib/ui/compositing.dart (Flutter 3.50.1),
// with painting.dart's EngineLayer and the natives of
// lib/ui/compositing/scene_builder.cc and scene.cc: Scene, SceneBuilder,
// EngineLayer and its typed wrappers (TransformEngineLayer ...
// ShaderMaskEngineLayer) with the debug checks of retained layers. The
// layer tree the natives build, and how it is painted, is in
// compositing_layers.go.

import (
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ---- Scene

// Scene is an opaque object representing a composited scene: build one
// with SceneBuilder and show it with FlutterView.Render, or rasterize it
// with ToImage/ToImageSync.
type Scene interface {
	// ToImageSync synchronously creates a handle to an image of this scene,
	// width×height physical pixels.
	ToImageSync(width, height int) *Image
	// ToImage creates a raster image of this scene, width×height physical
	// pixels.
	ToImage(width, height int) async.Future[*Image]
	// Dispose releases the scene's resources; the scene can't be used
	// afterwards.
	Dispose()
}

// nativeScene is _NativeScene with the engine's Scene: the root layer of
// the tree SceneBuilder built (nil once disposed). Detached is the Dart
// wrapper's cleared native peer (dispose calls ClearDartWrapper).
type nativeScene struct {
	root     *containerLayer
	detached bool
}

func (s *nativeScene) native() *nativeScene {
	if s.detached {
		panic(errNativePeerCollected)
	}
	return s
}

// valid is Scene::valid.
func (s *nativeScene) valid() bool { return s.root != nil }

// ToImageSync is Scene.toImageSync. ENGINE GAP: as Picture.toImageSync,
// the scene is rendered now (the engine defers it to a texture filled on
// the raster thread).
func (s *nativeScene) ToImageSync(width, height int) *Image {
	if width <= 0 || height <= 0 {
		panic(dart.NewException("Invalid image dimensions."))
	}
	n := s.native()
	if !n.valid() {
		panic(PictureRasterizationException{Message: "Scene has been disposed."})
	}
	img, err := n.buildLayerTree(width, height).rasterize()
	if err != nil {
		panic(PictureRasterizationException{Message: err.Error()})
	}
	return img
}

// ToImage is Scene.toImage: the future completes in a later event of the
// current loop (the engine completes it from the raster thread).
func (s *nativeScene) ToImage(width, height int) async.Future[*Image] {
	if width <= 0 || height <= 0 {
		panic(dart.NewException("Invalid image dimensions."))
	}
	n := s.native()
	if !n.valid() {
		panic(dart.NewException("Scene has been disposed."))
	}
	c := async.NewSyncCompleter[*Image]()
	img, err := n.buildLayerTree(width, height).rasterize()
	async.Current().Post(func() {
		if err != nil || img == nil {
			c.CompleteError(dart.NewException("operation failed"))
			return
		}
		c.Complete(img)
	})
	return c.Future()
}

// Dispose is Scene::dispose: the layer tree is released and the Dart
// wrapper detached.
func (s *nativeScene) Dispose() {
	n := s.native()
	n.root = nil
	n.detached = true
}

func (s *nativeScene) String() string { return "Scene" }

// buildLayerTree is Scene::BuildLayerTree (nil when disposed).
func (s *nativeScene) buildLayerTree(width, height int) *EngineLayerTree {
	if !s.valid() {
		return nil
	}
	return &EngineLayerTree{root: s.root, width: width, height: height}
}

// takeLayerTree is Scene::takeLayerTree (the engine's uint64 sizes become
// the tree's uint32 frame size).
func (s *nativeScene) takeLayerTree(width, height uint64) *EngineLayerTree {
	if s.detached {
		panic(errNativePeerCollected)
	}
	return s.buildLayerTree(int(uint32(width)), int(uint32(height)))
}

// ---- EngineLayer

// EngineLayer is a handle to a layer the engine retains between frames: a
// SceneBuilder push method returns one, and passing it back as oldLayer
// (or to AddRetained) lets the engine reuse it.
type EngineLayer interface {
	// Dispose releases the layer's resources.
	Dispose()
}

// nativeEngineLayer is _NativeEngineLayer with the engine's EngineLayer:
// the retained layer (nil once disposed).
type nativeEngineLayer struct {
	layer layer
}

// Dispose is EngineLayer::dispose.
func (l *nativeEngineLayer) Dispose() { l.layer = nil }

// engineLayerWrapper is _EngineLayerWrapper, embedded by every typed
// layer.
type engineLayerWrapper struct {
	nativeLayer *nativeEngineLayer
	// debugChildren are the layers pushed while this one was the top of the
	// builder's stack (debug builds only).
	debugChildren []*engineLayerWrapper
	// debugWasUsedAsOldLayer is whether the layer was passed as oldLayer.
	debugWasUsedAsOldLayer bool
	// runtimeType is the Dart class name of the typed layer.
	runtimeType string
}

func (w *engineLayerWrapper) wrapper() *engineLayerWrapper { return w }

// Dispose is _EngineLayerWrapper.dispose.
func (w *engineLayerWrapper) Dispose() {
	dartAssert(w.nativeLayer != nil, "Object disposed")
	w.nativeLayer.Dispose()
	if debugMode {
		w.nativeLayer = nil
	}
}

// shortHash is the framework's shortHash of the layer: its identity hash's
// low 20 bits in hex, padded to 5 digits.
func (w *engineLayerWrapper) shortHash() string {
	s := strconv.FormatInt(int64(dart.IdentityHashCode(w)&0xFFFFF), 16)
	return strings.Repeat("0", max(0, 5-len(s))) + s
}

// debugCheckNotUsedAsOldLayer is _debugCheckNotUsedAsOldLayer.
func (w *engineLayerWrapper) debugCheckNotUsedAsOldLayer() bool {
	dartAssert(!w.debugWasUsedAsOldLayer,
		"Layer "+w.runtimeType+"#"+w.shortHash()+" was previously used as oldLayer.\n"+
			"Once a layer is used as oldLayer, it may not be used again. Instead, "+
			"after calling one of the SceneBuilder.push* methods and passing an oldLayer "+
			"to it, use the layer returned by the method as oldLayer in subsequent "+
			"frames.")
	return true
}

type engineLayerWrapperHolder interface{ wrapper() *engineLayerWrapper }

// TransformEngineLayer is the layer SceneBuilder.PushTransform returns.
type TransformEngineLayer struct{ engineLayerWrapper }

// OffsetEngineLayer is the layer SceneBuilder.PushOffset returns.
type OffsetEngineLayer struct{ engineLayerWrapper }

// ClipRectEngineLayer is the layer SceneBuilder.PushClipRect returns.
type ClipRectEngineLayer struct{ engineLayerWrapper }

// ClipRRectEngineLayer is the layer SceneBuilder.PushClipRRect returns.
type ClipRRectEngineLayer struct{ engineLayerWrapper }

// ClipRSuperellipseEngineLayer is the layer
// SceneBuilder.PushClipRSuperellipse returns.
type ClipRSuperellipseEngineLayer struct{ engineLayerWrapper }

// ClipPathEngineLayer is the layer SceneBuilder.PushClipPath returns.
type ClipPathEngineLayer struct{ engineLayerWrapper }

// OpacityEngineLayer is the layer SceneBuilder.PushOpacity returns.
type OpacityEngineLayer struct{ engineLayerWrapper }

// ColorFilterEngineLayer is the layer SceneBuilder.PushColorFilter
// returns.
type ColorFilterEngineLayer struct{ engineLayerWrapper }

// ImageFilterEngineLayer is the layer SceneBuilder.PushImageFilter
// returns.
type ImageFilterEngineLayer struct{ engineLayerWrapper }

// BackdropFilterEngineLayer is the layer SceneBuilder.PushBackdropFilter
// returns.
type BackdropFilterEngineLayer struct{ engineLayerWrapper }

// ShaderMaskEngineLayer is the layer SceneBuilder.PushShaderMask returns.
type ShaderMaskEngineLayer struct{ engineLayerWrapper }

// ---- SceneBuilder

// SceneBuilder builds a Scene: push* methods push a layer that applies an
// effect (transform, clip, opacity, filter...) to everything added until
// the matching Pop; add* methods add leaf content (pictures, textures,
// platform views, retained layers).
type SceneBuilder interface {
	PushTransform(matrix4 []float64, opts ...PushTransformOptions) *TransformEngineLayer
	PushOffset(dx, dy float64, opts ...PushOffsetOptions) *OffsetEngineLayer
	PushClipRect(rect Rect, opts ...PushClipRectOptions) *ClipRectEngineLayer
	PushClipRRect(rrect RRect, opts ...PushClipRRectOptions) *ClipRRectEngineLayer
	PushClipRSuperellipse(rsuperellipse RSuperellipse, opts ...PushClipRSuperellipseOptions) *ClipRSuperellipseEngineLayer
	PushClipPath(path *Path, opts ...PushClipPathOptions) *ClipPathEngineLayer
	PushOpacity(alpha int, opts ...PushOpacityOptions) *OpacityEngineLayer
	PushColorFilter(filter *ColorFilter, opts ...PushColorFilterOptions) *ColorFilterEngineLayer
	PushImageFilter(filter ImageFilter, opts ...PushImageFilterOptions) *ImageFilterEngineLayer
	PushBackdropFilter(filter ImageFilter, opts ...PushBackdropFilterOptions) *BackdropFilterEngineLayer
	PushShaderMask(shader Shader, maskRect Rect, blendMode BlendMode, opts ...PushShaderMaskOptions) *ShaderMaskEngineLayer
	Pop()
	AddRetained(retainedLayer EngineLayer)
	AddPerformanceOverlay(enabledOptions int, bounds Rect)
	AddPicture(offset Offset, picture *Picture, opts ...AddPictureOptions)
	AddTexture(textureID int, opts ...AddTextureOptions)
	AddPlatformView(viewID int, opts ...AddPlatformViewOptions)
	Build() Scene
}

// PushTransformOptions are pushTransform's named arguments.
type PushTransformOptions struct{ OldLayer *TransformEngineLayer }

// PushOffsetOptions are pushOffset's named arguments.
type PushOffsetOptions struct{ OldLayer *OffsetEngineLayer }

// PushClipRectOptions are pushClipRect's named arguments (nil
// ClipBehavior: Clip.antiAlias).
type PushClipRectOptions struct {
	ClipBehavior *Clip
	OldLayer     *ClipRectEngineLayer
}

// PushClipRRectOptions are pushClipRRect's named arguments.
type PushClipRRectOptions struct {
	ClipBehavior *Clip
	OldLayer     *ClipRRectEngineLayer
}

// PushClipRSuperellipseOptions are pushClipRSuperellipse's named
// arguments.
type PushClipRSuperellipseOptions struct {
	ClipBehavior *Clip
	OldLayer     *ClipRSuperellipseEngineLayer
}

// PushClipPathOptions are pushClipPath's named arguments.
type PushClipPathOptions struct {
	ClipBehavior *Clip
	OldLayer     *ClipPathEngineLayer
}

// PushOpacityOptions are pushOpacity's named arguments (the zero Offset is
// Dart's default Offset.zero).
type PushOpacityOptions struct {
	Offset   Offset
	OldLayer *OpacityEngineLayer
}

// PushColorFilterOptions are pushColorFilter's named arguments.
type PushColorFilterOptions struct{ OldLayer *ColorFilterEngineLayer }

// PushImageFilterOptions are pushImageFilter's named arguments.
type PushImageFilterOptions struct {
	Offset   Offset
	OldLayer *ImageFilterEngineLayer
}

// PushBackdropFilterOptions are pushBackdropFilter's named arguments (nil
// BlendMode: BlendMode.srcOver; nil BackdropID: null).
type PushBackdropFilterOptions struct {
	BlendMode  *BlendMode
	OldLayer   *BackdropFilterEngineLayer
	BackdropID *int
}

// PushShaderMaskOptions are pushShaderMask's named arguments (nil
// FilterQuality: FilterQuality.low).
type PushShaderMaskOptions struct {
	OldLayer      *ShaderMaskEngineLayer
	FilterQuality *FilterQuality
}

// AddPictureOptions are addPicture's named arguments.
type AddPictureOptions struct {
	IsComplexHint  bool
	WillChangeHint bool
}

// AddTextureOptions are addTexture's named arguments (nil FilterQuality:
// FilterQuality.low).
type AddTextureOptions struct {
	Offset        Offset
	Width, Height float64
	Freeze        bool
	FilterQuality *FilterQuality
}

// AddPlatformViewOptions are addPlatformView's named arguments.
type AddPlatformViewOptions struct {
	Offset        Offset
	Width, Height float64
}

// nativeSceneBuilder is _NativeSceneBuilder with the engine's SceneBuilder
// (layerStack, the native side; nil once built: the wrapper is cleared).
type nativeSceneBuilder struct {
	// usedLayers are the layers used in this scene and what for
	// ("oldLayer in pushOpacity", "retained layer"), debug builds only.
	usedLayers      map[*engineLayerWrapper]string
	debugLayerStack []*engineLayerWrapper

	layerStack []*containerLayer
}

// NewSceneBuilder is SceneBuilder(): a builder whose root is a container
// layer, so add* calls are always valid.
func NewSceneBuilder() SceneBuilder {
	b := &nativeSceneBuilder{usedLayers: map[*engineLayerWrapper]string{}}
	b.pushLayer(newContainerLayer())
	return b
}

func (b *nativeSceneBuilder) String() string { return "SceneBuilder" }

// native is the engine object; it panics once the scene is built (the
// Dart wrapper's native peer is gone).
func (b *nativeSceneBuilder) native() *nativeSceneBuilder {
	if b.layerStack == nil {
		panic(errNativePeerCollected)
	}
	return b
}

// debugCheckUsedOnce is _debugCheckUsedOnce.
func (b *nativeSceneBuilder) debugCheckUsedOnce(layer *engineLayerWrapper, usage string) bool {
	if debugMode {
		used, ok := b.usedLayers[layer]
		dartAssert(!ok, "Layer "+layer.runtimeType+" already used.\n"+
			"The layer is already being used as "+used+" in this scene.\n"+
			"A layer may only be used once in a given scene.")
		b.usedLayers[layer] = usage
	}
	return true
}

// debugCheckCanBeUsedAsOldLayer is _debugCheckCanBeUsedAsOldLayer.
func (b *nativeSceneBuilder) debugCheckCanBeUsedAsOldLayer(layer *engineLayerWrapper, methodName string) {
	if debugMode && layer != nil {
		dartAssert(layer.nativeLayer != nil, "Object disposed")
		layer.debugCheckNotUsedAsOldLayer()
		b.debugCheckUsedOnce(layer, "oldLayer in "+methodName)
		layer.debugWasUsedAsOldLayer = true
	}
}

// debugPushLayer is _debugPushLayer.
func (b *nativeSceneBuilder) debugPushLayer(newLayer *engineLayerWrapper) {
	if debugMode {
		if n := len(b.debugLayerStack); n > 0 {
			current := b.debugLayerStack[n-1]
			current.debugChildren = append(current.debugChildren, newLayer)
		}
		b.debugLayerStack = append(b.debugLayerStack, newLayer)
	}
}

// oldWrapper is oldLayer as an _EngineLayerWrapper (nil for null).
func oldWrapper[T any, P interface {
	*T
	engineLayerWrapperHolder
}](old P) *engineLayerWrapper {
	if old == nil {
		return nil
	}
	return old.wrapper()
}

// push records a pushed layer: the engine layer handle (EngineLayer::
// MakeRetained) and the typed wrapper.
func (b *nativeSceneBuilder) push(l containerLayerNode, runtimeType string) engineLayerWrapper {
	n := b.native()
	n.pushLayer(l)
	return engineLayerWrapper{nativeLayer: &nativeEngineLayer{layer: l}, runtimeType: runtimeType}
}

// PushTransform pushes a transform (a 4×4 column-major matrix) onto the
// operation stack.
func (b *nativeSceneBuilder) PushTransform(matrix4 []float64, opts ...PushTransformOptions) *TransformEngineLayer {
	var old *TransformEngineLayer
	if len(opts) > 0 {
		old = opts[0].OldLayer
	}
	matrix4IsValid(matrix4)
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(old), "pushTransform")
	var m dlMatrix
	for i, v := range matrix4 {
		m[i] = safeNarrow(v)
	}
	l := &TransformEngineLayer{b.push(newTransformLayer(m), "TransformEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushOffset pushes an offset onto the operation stack (a translation).
func (b *nativeSceneBuilder) PushOffset(dx, dy float64, opts ...PushOffsetOptions) *OffsetEngineLayer {
	var old *OffsetEngineLayer
	if len(opts) > 0 {
		old = opts[0].OldLayer
	}
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(old), "pushOffset")
	m := identityMatrix()
	m[12], m[13] = safeNarrow(dx), safeNarrow(dy)
	l := &OffsetEngineLayer{b.push(newTransformLayer(m), "OffsetEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

func clipOpt(c *Clip) Clip {
	if c == nil {
		return ClipAntiAlias
	}
	return *c
}

// PushClipRect pushes a rectangular clip onto the operation stack.
func (b *nativeSceneBuilder) PushClipRect(rect Rect, opts ...PushClipRectOptions) *ClipRectEngineLayer {
	var o PushClipRectOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	clip := clipOpt(o.ClipBehavior)
	assertNoMessage(clip != ClipNone)
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushClipRect")
	r := dlRect{safeNarrow(rect.Left), safeNarrow(rect.Top), safeNarrow(rect.Right), safeNarrow(rect.Bottom)}
	l := &ClipRectEngineLayer{b.push(newClipRectLayer(r, clip), "ClipRectEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushClipRRect pushes a rounded-rectangular clip onto the operation
// stack.
func (b *nativeSceneBuilder) PushClipRRect(rrect RRect, opts ...PushClipRRectOptions) *ClipRRectEngineLayer {
	var o PushClipRRectOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	clip := clipOpt(o.ClipBehavior)
	assertNoMessage(clip != ClipNone)
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushClipRRect")
	l := &ClipRRectEngineLayer{b.push(newClipRRectLayer(rrectValue32(rrect), clip), "ClipRRectEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// rrectValue32 is RRect._getValue32: the rounded rectangle as the engine
// receives it, in float32.
func rrectValue32(r RRect) RRect {
	f := func(v float64) float64 { return float64(float32(v)) }
	l := r.rrectLike
	l.Left, l.Top, l.Right, l.Bottom = f(l.Left), f(l.Top), f(l.Right), f(l.Bottom)
	l.TlRadiusX, l.TlRadiusY, l.TrRadiusX, l.TrRadiusY = f(l.TlRadiusX), f(l.TlRadiusY), f(l.TrRadiusX), f(l.TrRadiusY)
	l.BrRadiusX, l.BrRadiusY, l.BlRadiusX, l.BlRadiusY = f(l.BrRadiusX), f(l.BrRadiusY), f(l.BlRadiusX), f(l.BlRadiusY)
	return RRect{l}
}

// PushClipRSuperellipse pushes a rounded-superellipse clip onto the
// operation stack.
func (b *nativeSceneBuilder) PushClipRSuperellipse(rsuperellipse RSuperellipse, opts ...PushClipRSuperellipseOptions) *ClipRSuperellipseEngineLayer {
	var o PushClipRSuperellipseOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	clip := clipOpt(o.ClipBehavior)
	assertNoMessage(clip != ClipNone)
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushClipRSuperellipse")
	l := &ClipRSuperellipseEngineLayer{b.push(newClipRSuperellipseLayer(rsuperellipse, clip), "ClipRSuperellipseEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushClipPath pushes a path clip onto the operation stack.
func (b *nativeSceneBuilder) PushClipPath(path *Path, opts ...PushClipPathOptions) *ClipPathEngineLayer {
	var o PushClipPathOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	clip := clipOpt(o.ClipBehavior)
	assertNoMessage(clip != ClipNone)
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushClipPath")
	l := &ClipPathEngineLayer{b.push(newClipPathLayer(NewPathFrom(path), clip), "ClipPathEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushOpacity pushes an opacity operation onto the operation stack: alpha
// 0 is fully transparent, 255 opaque; the layer's content is offset by
// Offset.
func (b *nativeSceneBuilder) PushOpacity(alpha int, opts ...PushOpacityOptions) *OpacityEngineLayer {
	var o PushOpacityOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushOpacity")
	// The native takes an Int32 and the engine's layer a uint8_t.
	l := &OpacityEngineLayer{b.push(newOpacityLayer(uint8(int32(alpha)),
		safeNarrow(o.Offset.Dx), safeNarrow(o.Offset.Dy)), "OpacityEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushColorFilter pushes a color filter onto the operation stack.
func (b *nativeSceneBuilder) PushColorFilter(filter *ColorFilter, opts ...PushColorFilterOptions) *ColorFilterEngineLayer {
	var old *ColorFilterEngineLayer
	if len(opts) > 0 {
		old = opts[0].OldLayer
	}
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(old), "pushColorFilter")
	l := &ColorFilterEngineLayer{b.push(newColorFilterLayer(filter), "ColorFilterEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushImageFilter pushes an image filter onto the operation stack (the
// layer's content is offset by Offset).
func (b *nativeSceneBuilder) PushImageFilter(filter ImageFilter, opts ...PushImageFilterOptions) *ImageFilterEngineLayer {
	var o PushImageFilterOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushImageFilter")
	l := &ImageFilterEngineLayer{b.push(newImageFilterLayer(filter,
		safeNarrow(o.Offset.Dx), safeNarrow(o.Offset.Dy)), "ImageFilterEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushBackdropFilter pushes a backdrop filter onto the operation stack:
// the filter applies to what was painted before the layer, within its
// clip.
func (b *nativeSceneBuilder) PushBackdropFilter(filter ImageFilter, opts ...PushBackdropFilterOptions) *BackdropFilterEngineLayer {
	var o PushBackdropFilterOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushBackdropFilter")
	mode := BlendModeSrcOver
	if o.BlendMode != nil {
		mode = *o.BlendMode
	}
	l := &BackdropFilterEngineLayer{b.push(newBackdropFilterLayer(filter, mode, o.BackdropID), "BackdropFilterEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// PushShaderMask pushes a shader mask onto the operation stack: the
// content is blended with the shader over maskRect.
func (b *nativeSceneBuilder) PushShaderMask(shader Shader, maskRect Rect, blendMode BlendMode, opts ...PushShaderMaskOptions) *ShaderMaskEngineLayer {
	var o PushShaderMaskOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	b.debugCheckCanBeUsedAsOldLayer(oldWrapper(o.OldLayer), "pushShaderMask")
	quality := FilterQualityLow
	if o.FilterQuality != nil {
		quality = *o.FilterQuality
	}
	r := dlRect{safeNarrow(maskRect.Left), safeNarrow(maskRect.Top), safeNarrow(maskRect.Right), safeNarrow(maskRect.Bottom)}
	l := &ShaderMaskEngineLayer{b.push(newShaderMaskLayer(shader, quality, r, blendMode), "ShaderMaskEngineLayer")}
	b.debugPushLayer(&l.engineLayerWrapper)
	return l
}

// Pop ends the effect of the most recent push (the root is never popped).
func (b *nativeSceneBuilder) Pop() {
	if n := len(b.debugLayerStack); n > 0 {
		b.debugLayerStack = b.debugLayerStack[:n-1]
	}
	b.native().popLayer()
}

// AddRetained adds a layer retained from a previous frame (with its
// subtree) to the scene.
func (b *nativeSceneBuilder) AddRetained(retainedLayer EngineLayer) {
	h, ok := retainedLayer.(engineLayerWrapperHolder)
	assertNoMessage(ok)
	layer := h.wrapper()
	if debugMode {
		assertNoMessage(layer.nativeLayer != nil)
		var recursivelyCheckChildrenUsedOnce func(parent *engineLayerWrapper)
		recursivelyCheckChildrenUsedOnce = func(parent *engineLayerWrapper) {
			b.debugCheckUsedOnce(parent, "retained layer")
			parent.debugCheckNotUsedAsOldLayer()
			for _, child := range parent.debugChildren {
				recursivelyCheckChildrenUsedOnce(child)
			}
		}
		recursivelyCheckChildrenUsedOnce(layer)
	}
	if layer.nativeLayer == nil {
		panic(dart.TypeError{Message: "Null check operator used on a null value"})
	}
	n := b.native()
	if l := layer.nativeLayer.layer; l != nil {
		n.addLayer(l)
	}
}

// AddPerformanceOverlay adds a performance overlay (enabledOptions are
// the engine's PerformanceOverlayLayer option bits) in bounds. ENGINE GAP:
// the overlay is recorded but paints nothing (no frame statistics are
// collected).
func (b *nativeSceneBuilder) AddPerformanceOverlay(enabledOptions int, bounds Rect) {
	r := dlRect{safeNarrow(bounds.Left), safeNarrow(bounds.Top), safeNarrow(bounds.Right), safeNarrow(bounds.Bottom)}
	b.native().addLayer(newPerformanceOverlayLayer(uint64(enabledOptions), r))
}

// AddPicture adds a picture at offset.
func (b *nativeSceneBuilder) AddPicture(offset Offset, picture *Picture, opts ...AddPictureOptions) {
	var o AddPictureOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if debugMode {
		assertNoMessage(!picture.disposed)
	}
	n := b.native()
	// SceneBuilder::addPicture: a disposed picture has no display list.
	if picture.dl != nil {
		n.addLayer(newDisplayListLayer(safeNarrow(offset.Dx), safeNarrow(offset.Dy), picture.dl,
			o.IsComplexHint, o.WillChangeHint))
	}
}

// AddTexture adds the backend texture textureID at offset, width×height.
// ENGINE GAP: flutter-go has no texture registry (external textures), so
// the layer paints nothing, as the engine does for an unknown texture id.
func (b *nativeSceneBuilder) AddTexture(textureID int, opts ...AddTextureOptions) {
	var o AddTextureOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	quality := FilterQualityLow
	if o.FilterQuality != nil {
		quality = *o.FilterQuality
	}
	b.native().addLayer(newTextureLayer(safeNarrow(o.Offset.Dx), safeNarrow(o.Offset.Dy),
		safeNarrow(o.Width), safeNarrow(o.Height), textureID, o.Freeze, quality))
}

// AddPlatformView adds the platform view viewID at offset, width×height.
// ENGINE GAP: flutter-go's embedders have no platform views (no external
// view embedder), so the layer paints nothing, as the engine does without
// one (it logs an error in debug builds).
func (b *nativeSceneBuilder) AddPlatformView(viewID int, opts ...AddPlatformViewOptions) {
	var o AddPlatformViewOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	b.native().addLayer(newPlatformViewLayer(safeNarrow(o.Offset.Dx), safeNarrow(o.Offset.Dy),
		safeNarrow(o.Width), safeNarrow(o.Height), viewID))
}

// Build finishes the scene; the builder can't be used afterwards.
func (b *nativeSceneBuilder) Build() Scene {
	scene := &nativeScene{}
	n := b.native()
	scene.root = n.layerStack[0]
	n.layerStack = nil // layer_stack_.clear(); ClearDartWrapper()
	return scene
}

// ---- the engine's SceneBuilder layer stack

func (b *nativeSceneBuilder) addLayer(l layer) {
	if len(b.layerStack) > 0 {
		top := b.layerStack[len(b.layerStack)-1]
		top.add(l)
	}
}

func (b *nativeSceneBuilder) pushLayer(l containerLayerNode) {
	b.addLayer(l)
	b.layerStack = append(b.layerStack, l.container())
}

func (b *nativeSceneBuilder) popLayer() {
	// The root layer is never popped, so add operations are always valid.
	if len(b.layerStack) > 1 {
		b.layerStack = b.layerStack[:len(b.layerStack)-1]
	}
}
