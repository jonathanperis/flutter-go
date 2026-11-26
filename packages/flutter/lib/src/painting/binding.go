package painting

// Port of binding.dart (Flutter 3.50.1): PaintingBinding (mixin
// PaintingBinding on BindingBase, ServicesBinding), composed as API_MAPPING
// "Bindings" describes, and the top-level imageCache getter.

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PaintingBinding binds the painting library to the engine: it owns the
// image cache, the image codec entry points and the system fonts
// notifier.
type PaintingBinding interface {
	services.ServicesBinding

	// ImageCache is the singleton that implements the Flutter framework's
	// image cache.
	ImageCache() *ImageCache
	// CreateImageCache creates the ImageCache (InitInstances calls it once;
	// bindings override it to use a subclass).
	CreateImageCache() *ImageCache
	// InstantiateImageCodecFromBuffer calls ui.InstantiateImageCodecFromBuffer
	// (deprecated in Flutter: use InstantiateImageCodecWithSize).
	InstantiateImageCodecFromBuffer(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecFromBufferOptions) async.Future[ui.Codec]
	// InstantiateImageCodecWithSize calls ui.InstantiateImageCodecWithSize.
	InstantiateImageCodecWithSize(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec]
	// SystemFonts notifies its listeners when the system fonts change.
	SystemFonts() foundation.Listenable
}

// InstantiateImageCodecFromBufferOptions are
// instantiateImageCodecFromBuffer's named parameters (Dart's
// allowUpscaling = false).
type InstantiateImageCodecFromBufferOptions struct {
	CacheWidth, CacheHeight *int
	AllowUpscaling          bool
}

// InstantiateImageCodecWithSizeOptions are instantiateImageCodecWithSize's
// named parameters.
type InstantiateImageCodecWithSizeOptions struct {
	GetTargetSize ui.TargetImageSizeCallback
}

// PaintingBindingMixin is PaintingBinding's state and methods, to embed in
// a binding after the ServicesBindingMixin (see foundation's
// binding_mixins.go). A binding embedding it defines InitInstances, Evict,
// HandleMemoryPressure and HandleSystemMessage, calling the last mixin in
// `with` order that defines each (PaintingBindingMixin's, unless a later
// mixin overrides them).
type PaintingBindingMixin struct {
	self        PaintingBinding
	imageCache  *ImageCache
	systemFonts *systemFontsNotifier
}

// SetSelf points the mixin at the binding embedding it.
func (m *PaintingBindingMixin) SetSelf(self PaintingBinding) {
	m.self = self
	m.systemFonts = &systemFontsNotifier{}
}

func (m *PaintingBindingMixin) outer() PaintingBinding {
	if m.self == nil {
		panic("painting: PaintingBindingMixin used before SetSelf; call SetSelf with the binding when creating it")
	}
	return m.self
}

var paintingBindingInstance PaintingBinding

// PaintingBindingInstance is PaintingBinding.instance.
func PaintingBindingInstance() PaintingBinding {
	return foundation.BindingBaseCheckInstance(paintingBindingInstance)
}

// PaintingBindingShaderWarmUp is PaintingBinding.shaderWarmUp: the
// ShaderWarmUp executed when the binding is initialized (nil: none, the
// default).
var PaintingBindingShaderWarmUp ShaderWarmUp

func (m *PaintingBindingMixin) InitInstances() {
	foundation.BindingBaseSuper(m.outer(), m).InitInstances()
	paintingBindingInstance = m.outer()
	m.imageCache = m.outer().CreateImageCache()
	if PaintingBindingShaderWarmUp != nil {
		PaintingBindingShaderWarmUp.Execute()
	}
}

func (m *PaintingBindingMixin) ImageCache() *ImageCache { return m.imageCache }

func (m *PaintingBindingMixin) CreateImageCache() *ImageCache { return NewImageCache() }

func (m *PaintingBindingMixin) InstantiateImageCodecFromBuffer(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecFromBufferOptions) async.Future[ui.Codec] {
	var o InstantiateImageCodecFromBufferOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(o.CacheWidth == nil || *o.CacheWidth > 0)
	dartAssert(o.CacheHeight == nil || *o.CacheHeight > 0)
	return ui.InstantiateImageCodecFromBuffer(buffer, ui.InstantiateImageCodecOptions{
		TargetWidth: o.CacheWidth, TargetHeight: o.CacheHeight, DisallowUpscaling: !o.AllowUpscaling,
	})
}

func (m *PaintingBindingMixin) InstantiateImageCodecWithSize(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec] {
	var o InstantiateImageCodecWithSizeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return ui.InstantiateImageCodecWithSize(buffer, ui.InstantiateImageCodecWithSizeOptions{GetTargetSize: o.GetTargetSize})
}

// Evict also clears the image cache and its live images.
func (m *PaintingBindingMixin) Evict(asset string) {
	foundation.BindingSuper[interface{ Evict(string) }](m.outer(), m).Evict(asset)
	m.ImageCache().Self().Clear()
	m.ImageCache().Self().ClearLiveImages()
}

// HandleMemoryPressure also clears the image cache.
func (m *PaintingBindingMixin) HandleMemoryPressure() {
	foundation.BindingSuper[interface{ HandleMemoryPressure() }](m.outer(), m).HandleMemoryPressure()
	m.ImageCache().Self().Clear()
}

func (m *PaintingBindingMixin) SystemFonts() foundation.Listenable { return m.systemFonts }

// HandleSystemMessage also notifies SystemFonts on a "fontsChange" message.
func (m *PaintingBindingMixin) HandleSystemMessage(systemMessage any) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		foundation.BindingSuper[interface {
			HandleSystemMessage(any) async.Future[async.Void]
		}](m.outer(), m).HandleSystemMessage(systemMessage).Await()
		message := systemMessage.(map[string]any)
		typ := message["type"].(string)
		switch typ {
		case "fontsChange":
			m.systemFonts.notifyListeners()
		}
	})
}

// systemFontsNotifier is _SystemFontsNotifier: a Listenable with a set of
// callbacks (insertion order, no duplicates).
type systemFontsNotifier struct {
	callbacks []func()
}

func (n *systemFontsNotifier) notifyListeners() {
	for _, callback := range append([]func(){}, n.callbacks...) {
		callback()
	}
}

func (n *systemFontsNotifier) AddListener(listener func()) {
	for _, c := range n.callbacks {
		if dart.FunctionsEqual(c, listener) {
			return
		}
	}
	n.callbacks = append(n.callbacks, listener)
}

func (n *systemFontsNotifier) RemoveListener(listener func()) {
	for i, c := range n.callbacks {
		if dart.FunctionsEqual(c, listener) {
			n.callbacks = append(n.callbacks[:i:i], n.callbacks[i+1:]...)
			return
		}
	}
}

// ImageCacheInstance is the top-level imageCache getter:
// PaintingBinding.instance.imageCache.
func ImageCacheInstance() *ImageCache { return PaintingBindingInstance().ImageCache() }
