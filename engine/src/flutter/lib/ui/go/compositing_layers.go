package ui

// The engine's layer tree, which SceneBuilder builds and the rasterizer
// paints: a port of engine/src/flutter/flow/layers (Flutter 3.50.1) —
// ContainerLayer, TransformLayer, ClipRect/ClipRRect/ClipRSuperellipse/
// ClipPathLayer (ClipShapeLayer), OpacityLayer, ColorFilterLayer,
// ImageFilterLayer, BackdropFilterLayer, ShaderMaskLayer, DisplayListLayer,
// TextureLayer, PlatformViewLayer, PerformanceOverlayLayer — and of
// LayerStateStack (layer_state_stack.cc), which defers opacity and filters
// to the children that can apply them and otherwise saves a layer, and
// LayerTree::Flatten. Layers paint into a dart:ui Canvas (the Impeller
// SDK's display list builder, the engine's DlCanvas).
//
// Not ported: retained-layer diffing (Diff, AssignOldLayer: the engine
// uses them for partial repaint only), the raster cache (Impeller has
// none), integral transforms (raster cache only).
//
// ENGINE GAP: the SDK doesn't expose a display list's bounds or whether it
// can apply group opacity, so a picture's paint bounds are unknown
// ("unbounded") and a picture never inherits opacity: an ancestor's
// opacity or filter is always applied with a saveLayer (unbounded when the
// bounds are unknown), and layers are never culled against the clip. The
// saveLayer the engine would elide is collapsed by Impeller itself when its
// content can take the opacity, so the pixels are the engine's.

import (
	"errors"
	"runtime"

	"github.com/jonathanperis/flutter-go/src/impeller"
	"github.com/jonathanperis/flutter-go/src/platform/metal"
)

// The LayerStateStack's kCallerCanApply* flags.
const (
	kCallerCanApplyOpacity     = 0x1
	kCallerCanApplyColorFilter = 0x2
	kCallerCanApplyImageFilter = 0x4
	// kSaveLayerRenderFlags is Layer::kSaveLayerRenderFlags.
	kSaveLayerRenderFlags = kCallerCanApplyOpacity | kCallerCanApplyColorFilter | kCallerCanApplyImageFilter
)

// lbounds is a layer's paint bounds: a DlRect, or unbounded when it is
// unknown (a picture's; see the file comment).
type lbounds struct {
	r         dlRect
	unbounded bool
}

var unboundedBounds = lbounds{unbounded: true}

func (b lbounds) isEmpty() bool { return !b.unbounded && b.r.isEmpty() }

// union is TRect::Union.
func (b lbounds) union(o lbounds) lbounds {
	if b.unbounded || o.unbounded {
		return unboundedBounds
	}
	if b.r.isEmpty() {
		return o
	}
	if o.r.isEmpty() {
		return b
	}
	return lbounds{r: dlRect{stdMin32(b.r.l, o.r.l), stdMin32(b.r.t, o.r.t), stdMax32(b.r.r, o.r.r), stdMax32(b.r.b, o.r.b)}}
}

// intersectsWith is TRect::IntersectsWithRect.
func (b lbounds) intersectsWith(o lbounds) bool {
	if b.unbounded || o.unbounded {
		return !b.isEmpty() && !o.isEmpty()
	}
	return b.r.intersectsWithRect(o.r)
}

// intersectionOrEmpty is TRect::IntersectionOrEmpty with a known rect.
func (b lbounds) intersectionOrEmpty(r dlRect) lbounds {
	if b.unbounded {
		return lbounds{r: protectEmpty(r)}
	}
	return lbounds{r: b.r.intersectionOrEmpty(r)}
}

// shift is TRect::Shift.
func (b lbounds) shift(dx, dy float32) lbounds {
	if b.unbounded {
		return b
	}
	return lbounds{r: dlRect{b.r.l + dx, b.r.t + dy, b.r.r + dx, b.r.b + dy}}
}

// transformAndClip is TRect::TransformAndClipBounds.
func (b lbounds) transformAndClip(m *dlMatrix) lbounds {
	if b.unbounded {
		return b
	}
	return lbounds{r: b.r.transformAndClipBounds(m)}
}

// rect is the bounds for a saveLayer: nil when unbounded.
func (b lbounds) rect() *dlRect {
	if b.unbounded {
		return nil
	}
	r := b.r
	return &r
}

// ---- layers

// layer is flutter::Layer.
type layer interface {
	preroll(ctx *prerollContext)
	paint(ctx *paintContext)
	base() *layerBase
}

// containerLayerNode is a layer SceneBuilder can push (a ContainerLayer).
type containerLayerNode interface {
	layer
	container() *containerLayer
}

// layerBase is flutter::Layer's state.
type layerBase struct {
	paintBounds            lbounds
	subtreeHasPlatformView bool
}

func (l *layerBase) base() *layerBase { return l }

// needsPainting is Layer::needs_painting (without the cull check against
// the clip; see the file comment).
func needsPainting(l layer, ctx *paintContext) bool {
	b := l.base()
	if b.subtreeHasPlatformView {
		return true
	}
	return !ctx.stack.paintingIsNop() && !b.paintBounds.isEmpty()
}

// prerollContext is PrerollContext (the fields that matter here).
type prerollContext struct {
	renderableStateFlags int
	hasPlatformView      bool
	hasTextureLayer      bool
}

// paintContext is PaintContext.
type paintContext struct {
	stack  *layerStateStack
	canvas *nativeCanvas
}

// containerLayer is ContainerLayer.
type containerLayer struct {
	layerBase
	layers                  []layer
	childPaintBounds        lbounds
	childrenRenderableFlags int
}

func newContainerLayer() *containerLayer { return &containerLayer{} }

func (c *containerLayer) container() *containerLayer { return c }

func (c *containerLayer) add(l layer) { c.layers = append(c.layers, l) }

func (c *containerLayer) preroll(ctx *prerollContext) {
	var childPaintBounds lbounds
	c.prerollChildren(ctx, &childPaintBounds)
	c.paintBounds = childPaintBounds
}

func (c *containerLayer) paint(ctx *paintContext) { c.paintChildren(ctx) }

// prerollChildren is ContainerLayer::PrerollChildren.
func (c *containerLayer) prerollChildren(ctx *prerollContext, childPaintBounds *lbounds) {
	childHasPlatformView := false
	childHasTextureLayer := false
	all := kSaveLayerRenderFlags // kCallerCanApplyAnything
	for _, l := range c.layers {
		ctx.hasPlatformView = false
		ctx.hasTextureLayer = false
		ctx.renderableStateFlags = 0
		l.preroll(ctx)
		all &= ctx.renderableStateFlags
		if childPaintBounds.intersectsWith(l.base().paintBounds) {
			// Inheritance works for a linear sequence of non-overlapping
			// children only.
			all = 0
		}
		*childPaintBounds = childPaintBounds.union(l.base().paintBounds)
		childHasPlatformView = childHasPlatformView || ctx.hasPlatformView
		childHasTextureLayer = childHasTextureLayer || ctx.hasTextureLayer
	}
	ctx.hasPlatformView = childHasPlatformView
	ctx.hasTextureLayer = childHasTextureLayer
	ctx.renderableStateFlags = all
	c.subtreeHasPlatformView = childHasPlatformView
	c.childrenRenderableFlags = all
	c.childPaintBounds = *childPaintBounds
}

// paintChildren is ContainerLayer::PaintChildren.
func (c *containerLayer) paintChildren(ctx *paintContext) {
	restore := ctx.stack.applyState(c.childPaintBounds, c.childrenRenderableFlags)
	for _, l := range c.layers {
		if needsPainting(l, ctx) {
			l.paint(ctx)
		}
	}
	restore()
}

// transformLayer is TransformLayer (pushTransform, pushOffset).
type transformLayer struct {
	containerLayer
	transform dlMatrix
}

func newTransformLayer(m dlMatrix) *transformLayer {
	if !matrixIsFinite(&m) {
		engineLog("TransformLayer is constructed with an invalid matrix.")
		m = identityMatrix()
	}
	return &transformLayer{transform: m}
}

func (l *transformLayer) preroll(ctx *prerollContext) {
	var childPaintBounds lbounds
	l.prerollChildren(ctx, &childPaintBounds)
	l.paintBounds = childPaintBounds.transformAndClip(&l.transform)
}

func (l *transformLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	m.transform(l.transform)
	l.paintChildren(ctx)
	m.restore()
}

// opacityLayer is OpacityLayer.
type opacityLayer struct {
	containerLayer
	alpha                    uint8
	dx, dy                   float32
	childrenCanAcceptOpacity bool
}

func newOpacityLayer(alpha uint8, dx, dy float32) *opacityLayer {
	return &opacityLayer{alpha: alpha, dx: dx, dy: dy}
}

// opacity is alpha_ * 1.0f / SK_AlphaOPAQUE.
func (l *opacityLayer) opacity() float32 { return float32(l.alpha) * 1.0 / 255 }

func (l *opacityLayer) preroll(ctx *prerollContext) {
	l.containerLayer.preroll(ctx)
	l.childrenCanAcceptOpacity = ctx.renderableStateFlags&kCallerCanApplyOpacity != 0
	// Parents can always pass opacity to an opacity layer.
	ctx.renderableStateFlags |= kCallerCanApplyOpacity
	l.paintBounds = l.paintBounds.shift(l.dx, l.dy)
}

func (l *opacityLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	m.translate(l.dx, l.dy)
	m.applyOpacity(l.childPaintBounds, l.opacity())
	l.paintChildren(ctx)
	m.restore()
}

// clipShapeLayer is ClipShapeLayer: a clip, with a saveLayer for
// Clip.antiAliasWithSaveLayer.
type clipShapeLayer struct {
	containerLayer
	clipBehavior Clip
	clipBounds   dlRect
	applyClip    func(m *mutatorContext, aa bool)
}

func (l *clipShapeLayer) usesSaveLayer() bool { return l.clipBehavior == ClipAntiAliasWithSaveLayer }

func (l *clipShapeLayer) preroll(ctx *prerollContext) {
	var childPaintBounds lbounds
	l.prerollChildren(ctx, &childPaintBounds)
	l.paintBounds = childPaintBounds.intersectionOrEmpty(l.clipBounds)
	if l.usesSaveLayer() {
		ctx.renderableStateFlags = kSaveLayerRenderFlags
	}
}

func (l *clipShapeLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	l.applyClip(m, l.clipBehavior != ClipHardEdge)
	if l.usesSaveLayer() {
		m.saveLayer(l.paintBounds)
	}
	l.paintChildren(ctx)
	m.restore()
}

func newClipRectLayer(r dlRect, clip Clip) *clipShapeLayer {
	assertNoMessage(clip != ClipNone)
	return &clipShapeLayer{clipBehavior: clip, clipBounds: r,
		applyClip: func(m *mutatorContext, aa bool) { m.clipRect(r, aa) }}
}

func newClipRRectLayer(rr RRect, clip Clip) *clipShapeLayer {
	assertNoMessage(clip != ClipNone)
	bounds := rrectBounds(rr)
	return &clipShapeLayer{clipBehavior: clip, clipBounds: bounds,
		applyClip: func(m *mutatorContext, aa bool) {
			if rrectIsRect(rr) {
				m.clipRect(bounds, aa)
			} else {
				m.clipRRect(rr, aa)
			}
		}}
}

// rrectIsRect is DlRoundRect::IsRect: every corner's radii are empty.
func rrectIsRect(r RRect) bool {
	empty := func(x, y float64) bool { return x <= 0 || y <= 0 }
	return empty(r.TlRadiusX, r.TlRadiusY) && empty(r.TrRadiusX, r.TrRadiusY) &&
		empty(r.BrRadiusX, r.BrRadiusY) && empty(r.BlRadiusX, r.BlRadiusY)
}

func newClipRSuperellipseLayer(rse RSuperellipse, clip Clip) *clipShapeLayer {
	assertNoMessage(clip != ClipNone)
	return &clipShapeLayer{clipBehavior: clip,
		clipBounds: dlRect{safeNarrow(rse.Left), safeNarrow(rse.Top), safeNarrow(rse.Right), safeNarrow(rse.Bottom)},
		applyClip:  func(m *mutatorContext, aa bool) { m.clipRSuperellipse(rse, aa) }}
}

func newClipPathLayer(p *Path, clip Clip) *clipShapeLayer {
	assertNoMessage(clip != ClipNone)
	return &clipShapeLayer{clipBehavior: clip, clipBounds: rectToDl(p.GetBounds()),
		applyClip: func(m *mutatorContext, aa bool) { m.clipPath(p, aa) }}
}

// colorFilterLayer is ColorFilterLayer.
type colorFilterLayer struct {
	containerLayer
	filter *ColorFilter
}

func newColorFilterLayer(f *ColorFilter) *colorFilterLayer { return &colorFilterLayer{filter: f} }

// colorFilterCanCommuteWithOpacity is DlColorFilter::
// can_commute_with_opacity: filter(color) * opacity == filter(color *
// opacity).
func colorFilterCanCommuteWithOpacity(f *ColorFilter) bool {
	switch f.kind {
	case colorFilterMode:
		switch f.blendMode {
		case BlendModeClear, BlendModeDst, BlendModeSrcIn, BlendModeDstIn, BlendModeDstOut,
			BlendModeSrcATop, BlendModeModulate:
			return true
		}
		return false
	case colorFilterMatrix:
		m := f.engineMatrix()
		if !(m[3] == 0 && m[8] == 0 && m[13] == 0) {
			return false
		}
		if !(m[15] == 0 && m[16] == 0 && m[17] == 0 && m[19] == 0) {
			return false
		}
		return true
	}
	return true // the gamma filters
}

func (l *colorFilterLayer) preroll(ctx *prerollContext) {
	l.containerLayer.preroll(ctx)
	if l.filter != nil {
		if colorFilterCanCommuteWithOpacity(l.filter) {
			ctx.renderableStateFlags = kCallerCanApplyOpacity
		} else {
			ctx.renderableStateFlags = 0
		}
	}
}

func (l *colorFilterLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	m.applyColorFilter(l.paintBounds, l.filter)
	l.paintChildren(ctx)
	m.restore()
}

// imageFilterLayer is ImageFilterLayer; the filter is the engine's
// filter(kDecal) (nil for a filter that does nothing).
type imageFilterLayer struct {
	containerLayer
	filter *impeller.ImageFilter
	dx, dy float32
}

func newImageFilterLayer(f ImageFilter, dx, dy float32) *imageFilterLayer {
	return &imageFilterLayer{filter: filterWithTileMode(f, TileModeDecal), dx: dx, dy: dy}
}

func (l *imageFilterLayer) preroll(ctx *prerollContext) {
	var childBounds lbounds
	l.prerollChildren(ctx, &childBounds)
	if l.filter == nil {
		l.paintBounds = childBounds.shift(l.dx, l.dy)
		return
	}
	ctx.renderableStateFlags = kCallerCanApplyOpacity | kCallerCanApplyColorFilter
	// The filter's output bounds (map_device_bounds) are not computed: the
	// SDK doesn't map filter bounds.
	l.paintBounds = unboundedBounds
}

func (l *imageFilterLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	m.translate(l.dx, l.dy)
	m.applyImageFilter(l.childPaintBounds, l.filter)
	l.paintChildren(ctx)
	m.restore()
}

// backdropFilterLayer is BackdropFilterLayer; the filter is the engine's
// filter(kMirror).
type backdropFilterLayer struct {
	containerLayer
	filter     *impeller.ImageFilter
	blendMode  BlendMode
	backdropID *int
}

func newBackdropFilterLayer(f ImageFilter, mode BlendMode, backdropID *int) *backdropFilterLayer {
	return &backdropFilterLayer{filter: filterWithTileMode(f, TileModeMirror), blendMode: mode, backdropID: backdropID}
}

func (l *backdropFilterLayer) preroll(ctx *prerollContext) {
	var childPaintBounds lbounds
	l.prerollChildren(ctx, &childPaintBounds)
	// The children's bounds united with the local cull rect: everything in
	// the clip.
	l.paintBounds = unboundedBounds
	ctx.renderableStateFlags = kSaveLayerRenderFlags
}

func (l *backdropFilterLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	m.applyBackdropFilter(l.paintBounds, l.filter, l.blendMode)
	l.paintChildren(ctx)
	m.restore()
}

// shaderMaskLayer is ShaderMaskLayer.
type shaderMaskLayer struct {
	containerLayer
	shader    Shader
	quality   FilterQuality
	maskRect  dlRect
	blendMode BlendMode
}

func newShaderMaskLayer(shader Shader, quality FilterQuality, maskRect dlRect, mode BlendMode) *shaderMaskLayer {
	return &shaderMaskLayer{shader: shader, quality: quality, maskRect: maskRect, blendMode: mode}
}

func (l *shaderMaskLayer) preroll(ctx *prerollContext) {
	l.containerLayer.preroll(ctx)
	ctx.renderableStateFlags = kSaveLayerRenderFlags
}

func (l *shaderMaskLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	m.saveLayer(l.paintBounds)
	l.paintChildren(ctx)
	// DlPaint with anti-aliasing off (the SDK's paints have no AA flag;
	// rects are not anti-aliased there either), the blend mode and the
	// shader.
	ip := impeller.NewPaint()
	ip.SetBlendMode(int(l.blendMode))
	var src *impeller.ColorSource
	if is, ok := l.shader.(*ImageShader); ok {
		src = is.colorSourceWithQuality(l.quality)
	} else if l.shader != nil {
		src = l.shader.impellerColorSource()
	}
	if src != nil {
		ip.SetColorSource(src)
	}
	c := ctx.canvas
	b := c.builder()
	b.Translate(l.maskRect.l, l.maskRect.t)
	b.DrawRect(impeller.Rect{W: l.maskRect.r - l.maskRect.l, H: l.maskRect.b - l.maskRect.t}, ip)
	c.ops++
	ip.Release()
	m.restore()
}

// displayListLayer is DisplayListLayer: a picture at an offset. It holds a
// reference to the picture's display list (released when the layer is
// collected), as the engine's sk_sp<DisplayList> does.
type displayListLayer struct {
	layerBase
	dx, dy float32
	dl     *retainedDisplayList
}

type retainedDisplayList struct{ dl *impeller.DisplayList }

func newDisplayListLayer(dx, dy float32, dl *impeller.DisplayList, isComplex, willChange bool) *displayListLayer {
	dl.Retain()
	r := &retainedDisplayList{dl}
	runtime.SetFinalizer(r, func(r *retainedDisplayList) { r.dl.Release() })
	return &displayListLayer{dx: dx, dy: dy, dl: r}
}

func (l *displayListLayer) preroll(ctx *prerollContext) {
	// ENGINE GAP: can_apply_group_opacity and the display list's bounds
	// are unknown (see the file comment).
	l.paintBounds = unboundedBounds
}

func (l *displayListLayer) paint(ctx *paintContext) {
	m := ctx.stack.save()
	m.translate(l.dx, l.dy)
	opacity := ctx.stack.outstanding.opacity
	ctx.canvas.builder().DrawDisplayList(l.dl.dl, opacity)
	ctx.canvas.ops++
	m.restore()
	runtime.KeepAlive(l.dl)
}

// textureLayer is TextureLayer. ENGINE GAP: there is no texture registry,
// so it paints nothing (the engine's behavior for an unknown texture).
type textureLayer struct {
	layerBase
	rect      dlRect
	textureID int
	freeze    bool
	quality   FilterQuality
}

func newTextureLayer(dx, dy, w, h float32, id int, freeze bool, q FilterQuality) *textureLayer {
	return &textureLayer{rect: dlRect{dx, dy, dx + w, dy + h}, textureID: id, freeze: freeze, quality: q}
}

func (l *textureLayer) preroll(ctx *prerollContext) {
	l.paintBounds = lbounds{r: l.rect}
	ctx.hasTextureLayer = true
	ctx.renderableStateFlags = kCallerCanApplyOpacity
}

func (l *textureLayer) paint(ctx *paintContext) {}

// platformViewLayer is PlatformViewLayer. ENGINE GAP: there is no external
// view embedder, so it paints nothing (as the engine without one).
type platformViewLayer struct {
	layerBase
	rect   dlRect
	viewID int
}

func newPlatformViewLayer(dx, dy, w, h float32, viewID int) *platformViewLayer {
	return &platformViewLayer{rect: dlRect{dx, dy, dx + w, dy + h}, viewID: viewID}
}

func (l *platformViewLayer) preroll(ctx *prerollContext) { l.paintBounds = lbounds{r: l.rect} }

func (l *platformViewLayer) paint(ctx *paintContext) {}

// performanceOverlayLayer is PerformanceOverlayLayer. ENGINE GAP: no frame
// statistics are collected, so it paints nothing.
type performanceOverlayLayer struct {
	layerBase
	options uint64
}

func newPerformanceOverlayLayer(options uint64, bounds dlRect) *performanceOverlayLayer {
	l := &performanceOverlayLayer{options: options}
	l.paintBounds = lbounds{r: bounds}
	return l
}

func (l *performanceOverlayLayer) preroll(ctx *prerollContext) {}

func (l *performanceOverlayLayer) paint(ctx *paintContext) {}

// ---- LayerStateStack

// renderingAttributes is LayerStateStack::RenderingAttributes: the
// opacity and filters waiting to be applied by a saveLayer or by the
// children.
type renderingAttributes struct {
	saveLayerBounds lbounds
	opacity         float32
	colorFilter     *ColorFilter
	imageFilter     *impeller.ImageFilter
}

func newRenderingAttributes() renderingAttributes { return renderingAttributes{opacity: 1} }

// fill is RenderingAttributes::fill: the saveLayer paint, nil when it
// would be the default.
func (a renderingAttributes) fill(mode BlendMode) *impeller.Paint {
	needed := false
	ip := impeller.NewPaint()
	opacity := float32(1)
	if a.opacity < 1 {
		opacity = stdMax32(a.opacity, 0)
		needed = true
	}
	ip.SetColorWithSpace(impeller.Color{A: opacity, Space: int(ColorSpaceSRGB)})
	if a.colorFilter != nil {
		if cf := a.colorFilter.impellerColorFilter(); cf != nil {
			ip.SetColorFilter(cf)
		}
		needed = true
	}
	if a.imageFilter != nil {
		ip.SetImageFilter(a.imageFilter)
		needed = true
	}
	ip.SetBlendMode(int(mode))
	if mode != BlendModeSrcOver {
		needed = true
	}
	if !needed {
		ip.Release()
		return nil
	}
	return ip
}

// layerStateStack is LayerStateStack with the DlCanvas delegate.
type layerStateStack struct {
	canvas      *nativeCanvas
	entries     []stateEntry
	outstanding renderingAttributes
}

func newLayerStateStack(c *nativeCanvas) *layerStateStack {
	return &layerStateStack{canvas: c, outstanding: newRenderingAttributes()}
}

// stateEntry is LayerStateStack::StateEntry: apply happened when it was
// pushed; restore undoes it.
type stateEntry interface{ restore(s *layerStateStack) }

type saveEntry struct{}

func (saveEntry) restore(s *layerStateStack) { s.canvas.Restore() }

type saveLayerEntry struct{ old renderingAttributes }

func (e saveLayerEntry) restore(s *layerStateStack) {
	s.canvas.Restore()
	s.outstanding = e.old
}

type attributeEntry struct{ old renderingAttributes }

func (e attributeEntry) restore(s *layerStateStack) { s.outstanding = e.old }

type transformEntry struct{}

func (transformEntry) restore(*layerStateStack) {}

func (s *layerStateStack) paintingIsNop() bool { return s.outstanding.opacity <= 0 }

func (s *layerStateStack) restoreToCount(n int) {
	for len(s.entries) > n {
		s.entries[len(s.entries)-1].restore(s)
		s.entries = s.entries[:len(s.entries)-1]
	}
}

// applyState is LayerStateStack::applyState: a saveLayer when the
// children can't apply what is outstanding; the returned function is the
// AutoRestore's destructor.
func (s *layerStateStack) applyState(bounds lbounds, flags int) func() {
	n := len(s.entries)
	if s.needsSaveLayer(flags) {
		s.saveLayer(bounds)
	}
	return func() { s.restoreToCount(n) }
}

func (s *layerStateStack) needsSaveLayer(flags int) bool {
	if s.outstanding.opacity < 1 && flags&kCallerCanApplyOpacity == 0 {
		return true
	}
	if s.outstanding.imageFilter != nil && flags&kCallerCanApplyImageFilter == 0 {
		return true
	}
	if s.outstanding.colorFilter != nil && flags&kCallerCanApplyColorFilter == 0 {
		return true
	}
	return false
}

// saveLayer is save_layer: a SaveLayerEntry applying the outstanding
// attributes.
func (s *layerStateStack) saveLayer(bounds lbounds) {
	s.saveLayerWith(bounds, BlendModeSrcOver, nil)
}

func (s *layerStateStack) saveLayerWith(bounds lbounds, mode BlendMode, backdrop *impeller.ImageFilter) {
	old := s.outstanding
	ip := s.outstanding.fill(mode)
	s.canvas.saveLayerEngine(bounds.rect(), ip, backdrop)
	if ip != nil {
		ip.Release()
	}
	s.outstanding = newRenderingAttributes()
	s.entries = append(s.entries, saveLayerEntry{old})
}

func (s *layerStateStack) doSave() {
	s.canvas.Save()
	s.entries = append(s.entries, saveEntry{})
}

func (s *layerStateStack) maybeSaveLayerForTransformOrClip(saveNeeded bool) {
	if s.outstanding.imageFilter != nil {
		s.saveLayer(s.outstanding.saveLayerBounds)
	} else if saveNeeded {
		s.doSave()
	}
}

// save is LayerStateStack::save: a MutatorContext.
func (s *layerStateStack) save() *mutatorContext {
	return &mutatorContext{stack: s, restoreCount: len(s.entries), saveNeeded: true}
}

// mutatorContext is LayerStateStack::MutatorContext; restore is its
// destructor.
type mutatorContext struct {
	stack        *layerStateStack
	restoreCount int
	saveNeeded   bool
}

func (m *mutatorContext) restore() { m.stack.restoreToCount(m.restoreCount) }

func (m *mutatorContext) saveLayer(bounds lbounds) { m.stack.saveLayer(bounds) }

func (m *mutatorContext) applyOpacity(bounds lbounds, opacity float32) {
	if opacity < 1 {
		s := m.stack
		// maybe_save_layer(opacity)
		if s.outstanding.imageFilter != nil {
			s.saveLayer(s.outstanding.saveLayerBounds)
		}
		s.entries = append(s.entries, attributeEntry{s.outstanding})
		s.outstanding.saveLayerBounds = bounds
		s.outstanding.opacity = float32(s.outstanding.opacity * opacity)
	}
}

func (m *mutatorContext) applyImageFilter(bounds lbounds, filter *impeller.ImageFilter) {
	if filter != nil {
		s := m.stack
		if s.outstanding.imageFilter != nil {
			s.saveLayer(s.outstanding.saveLayerBounds)
		}
		s.entries = append(s.entries, attributeEntry{s.outstanding})
		s.outstanding.saveLayerBounds = bounds
		s.outstanding.imageFilter = filter
	}
}

func (m *mutatorContext) applyColorFilter(bounds lbounds, filter *ColorFilter) {
	if filter != nil {
		s := m.stack
		if s.outstanding.colorFilter != nil || s.outstanding.imageFilter != nil ||
			(s.outstanding.opacity < 1 && !colorFilterCanCommuteWithOpacity(filter)) {
			s.saveLayer(s.outstanding.saveLayerBounds)
		}
		s.entries = append(s.entries, attributeEntry{s.outstanding})
		s.outstanding.saveLayerBounds = bounds
		s.outstanding.colorFilter = filter
	}
}

// applyBackdropFilter is push_backdrop: a saveLayer with the outstanding
// attributes, the blend mode and the backdrop filter.
func (m *mutatorContext) applyBackdropFilter(bounds lbounds, filter *impeller.ImageFilter, mode BlendMode) {
	m.stack.saveLayerWith(bounds, mode, filter)
}

func (m *mutatorContext) translate(tx, ty float32) {
	if !(tx == 0 && ty == 0) {
		m.stack.maybeSaveLayerForTransformOrClip(m.saveNeeded)
		m.saveNeeded = false
		m.stack.canvas.builder().Translate(tx, ty)
		m.stack.entries = append(m.stack.entries, transformEntry{})
	}
}

func (m *mutatorContext) transform(matrix dlMatrix) {
	if isTranslationOnly(&matrix) {
		m.translate(matrix[12], matrix[13])
	} else if matrix != identityMatrix() {
		m.stack.maybeSaveLayerForTransformOrClip(m.saveNeeded)
		m.saveNeeded = false
		m.stack.canvas.builder().Transform(matrix)
		m.stack.entries = append(m.stack.entries, transformEntry{})
	}
}

// isTranslationOnly is Matrix::IsTranslationOnly (a z translation is
// ignored, as in the engine).
func isTranslationOnly(m *dlMatrix) bool {
	return m[0] == 1 && m[1] == 0 && m[2] == 0 && m[3] == 0 &&
		m[4] == 0 && m[5] == 1 && m[6] == 0 && m[7] == 0 &&
		m[8] == 0 && m[9] == 0 && m[10] == 1 && m[11] == 0 && m[15] == 1
}

func (m *mutatorContext) beforeClip() {
	m.stack.maybeSaveLayerForTransformOrClip(m.saveNeeded)
	m.saveNeeded = false
	m.stack.entries = append(m.stack.entries, transformEntry{})
}

func (m *mutatorContext) clipRect(r dlRect, aa bool) {
	m.beforeClip()
	m.stack.canvas.ClipRect(r.toRect(), CanvasClipRectOptions{DoAntiAlias: &aa})
}

func (m *mutatorContext) clipRRect(r RRect, aa bool) {
	m.beforeClip()
	m.stack.canvas.ClipRRect(r, CanvasClipOptions{DoAntiAlias: &aa})
}

func (m *mutatorContext) clipRSuperellipse(r RSuperellipse, aa bool) {
	m.beforeClip()
	m.stack.canvas.ClipRSuperellipse(r, CanvasClipOptions{DoAntiAlias: &aa})
}

func (m *mutatorContext) clipPath(p *Path, aa bool) {
	m.beforeClip()
	m.stack.canvas.ClipPath(p, CanvasClipOptions{DoAntiAlias: &aa})
}

// saveLayerEngine is DlCanvas::SaveLayer for the compositor: bounds nil is
// unbounded, the paint may be nil, and backdrop is a backdrop filter.
func (c *nativeCanvas) saveLayerEngine(bounds *dlRect, ip *impeller.Paint, backdrop *impeller.ImageFilter) {
	b := c.builder()
	st := *c.top()
	if bounds == nil {
		r := impRect(kMaxCullRect)
		b.SaveLayer(&r, ip, backdrop)
	} else {
		r := impRect(*bounds)
		b.SaveLayer(&r, ip, backdrop)
		m := c.matrix()
		st.clipRect(&m, *bounds, ClipOpIntersect, false)
	}
	c.stack = append(c.stack, st)
	c.ops++
}

// ---- LayerTree

// EngineLayerTree is the engine's LayerTree: the root layer of a scene and
// the frame size, in physical pixels, that FlutterView.Render hands to the
// embedder (PlatformConfigurationRenderClient).
type EngineLayerTree struct {
	root          *containerLayer
	width, height int
}

// FrameSize is the tree's frame size in physical pixels.
func (t *EngineLayerTree) FrameSize() (width, height int) { return t.width, t.height }

// Flatten paints the tree into a display list (LayerTree::Flatten: preroll
// then paint, culled to the frame). The caller releases it.
func (t *EngineLayerTree) Flatten() *impeller.DisplayList {
	rec := NewPictureRecorder()
	c := NewCanvas(rec, NewRectFromLTWH(0, 0, float64(t.width), float64(t.height))).(*nativeCanvas)
	if t.root != nil {
		var pre prerollContext
		t.root.preroll(&pre)
		ctx := &paintContext{stack: newLayerStateStack(c), canvas: c}
		if needsPainting(t.root, ctx) {
			t.root.paint(ctx)
		}
		ctx.stack.restoreToCount(0)
	}
	dl := c.b.Build()
	c.b.Release()
	c.recorder = nil
	rec.canvas = nil
	return dl
}

// rasterize renders the tree into a premultiplied RGBA8 image of its frame
// size (Picture::RasterizeLayerTreeToImage).
func (t *EngineLayerTree) rasterize() (*Image, error) {
	ctx := engineContext()
	if ctx == nil {
		return nil, errors.New("no GPU context")
	}
	dl := t.Flatten()
	defer dl.Release()
	px, err := metal.RenderOffscreen(ctx.Handle(), dl.Handle(), t.width, t.height)
	if err != nil {
		return nil, err
	}
	return newImageFromRGBA8(t.width, t.height, px), nil
}
