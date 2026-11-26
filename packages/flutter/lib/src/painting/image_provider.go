package painting

// Port of image_provider.dart (Flutter 3.50.1): ImageConfiguration,
// ImageProvider, AssetBundleImageKey, AssetBundleImageProvider,
// ResizeImageKey/ResizeImage/ResizeImagePolicy, FileImage, MemoryImage,
// ExactAssetImage, WebHtmlElementStrategy, NetworkImageLoadException (and
// the decoder callback types). NetworkImage is in network_image_io.go.
//
// ImageProvider<T> is generic in Dart only over its key type, and every
// consumer uses ImageProvider<Object> (the image cache keys are Objects):
// in Go it is the non-generic interface ImageProvider whose keys are `any`
// (T erased), plus the embeddable ImageProviderBase (self pattern). A
// provider embeds ImageProviderBase (or AssetBundleImageProviderBase),
// calls SetSelf, and implements ObtainKey (and LoadImage). Keys are
// compared as image cache keys are (KeysEqual: an `Equal(any) bool` method,
// else Go's ==); providers whose Dart == differs from Go's define Equal.

import (
	"fmt"
	"iter"
	"math"
	"net/url"
	"slices"
	"unsafe"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/io"
)

// ---------------------------------------------------------------------------
// ImageConfiguration

// ImageConfiguration is the configuration information passed to
// ImageProvider.Resolve to select a specific image. Every field is
// nullable (nil); == is Equal.
type ImageConfiguration struct {
	// Bundle is the preferred AssetBundle to use if the ImageProvider needs
	// one and does not have one already selected.
	Bundle services.AssetBundle
	// DevicePixelRatio is the device pixel ratio where the image will be
	// shown.
	DevicePixelRatio *float64
	// Locale is the language and region for which to select the image.
	Locale *ui.Locale
	// TextDirection is the reading direction of the language for which to
	// select the image.
	TextDirection *ui.TextDirection
	// Size is the size at which the image will be rendered.
	Size *ui.Size
	// Platform is the TargetPlatform for which assets should be used.
	Platform *foundation.TargetPlatform
}

// ImageConfigurationEmpty is ImageConfiguration.empty: an image
// configuration that provides no additional information.
var ImageConfigurationEmpty = ImageConfiguration{}

// CopyWith creates an object holding the same information as this
// configuration, but with the non-nil fields of overrides replaced.
func (c ImageConfiguration) CopyWith(overrides ImageConfiguration) ImageConfiguration {
	if overrides.Bundle != nil {
		c.Bundle = overrides.Bundle
	}
	if overrides.DevicePixelRatio != nil {
		c.DevicePixelRatio = overrides.DevicePixelRatio
	}
	if overrides.Locale != nil {
		c.Locale = overrides.Locale
	}
	if overrides.TextDirection != nil {
		c.TextDirection = overrides.TextDirection
	}
	if overrides.Size != nil {
		c.Size = overrides.Size
	}
	if overrides.Platform != nil {
		c.Platform = overrides.Platform
	}
	return c
}

func ptrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Equal is ==: every field equal (the bundle by identity).
func (c ImageConfiguration) Equal(other any) bool {
	o, ok := other.(ImageConfiguration)
	if !ok {
		return false
	}
	return c.Bundle == o.Bundle && ptrEqual(c.DevicePixelRatio, o.DevicePixelRatio) &&
		ptrEqual(c.Locale, o.Locale) && ptrEqual(c.TextDirection, o.TextDirection) &&
		ptrEqual(c.Size, o.Size) && ptrEqual(c.Platform, o.Platform)
}

func (c ImageConfiguration) String() string {
	result := "ImageConfiguration("
	hasArguments := false
	add := func(s string) {
		if hasArguments {
			result += ", "
		}
		result += s
		hasArguments = true
	}
	if c.Bundle != nil {
		add("bundle: " + c.Bundle.String())
	}
	if c.DevicePixelRatio != nil {
		add("devicePixelRatio: " + dart.ToStringAsFixed(*c.DevicePixelRatio, 1))
	}
	if c.Locale != nil {
		add("locale: " + c.Locale.String())
	}
	if c.TextDirection != nil {
		add("textDirection: " + c.TextDirection.String())
	}
	if c.Size != nil {
		add("size: " + c.Size.String())
	}
	if c.Platform != nil {
		add("platform: " + c.Platform.Name())
	}
	return result + ")"
}

// ---------------------------------------------------------------------------
// Decoder callbacks

// DecoderBufferCallback is the deprecated signature of a function that
// decodes an ImmutableBuffer into a Codec with an optional cache size
// (PaintingBinding.InstantiateImageCodecFromBuffer).
type DecoderBufferCallback func(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecFromBufferOptions) async.Future[ui.Codec]

// ImageDecoderCallback is the signature of a function that decodes an
// ImmutableBuffer into a Codec, at the size GetTargetSize picks
// (PaintingBinding.InstantiateImageCodecWithSize).
type ImageDecoderCallback func(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec]

// simpleDecoderCallback is _SimpleDecoderCallback.
type simpleDecoderCallback func(buffer *ui.ImmutableBuffer) async.Future[ui.Codec]

// ---------------------------------------------------------------------------
// ImageProvider

// ImageProvider identifies an image without committing to the precise
// final asset: Resolve obtains a key for a configuration, and loads (or
// finds in the image cache) the completer for that key.
type ImageProvider interface {
	// Resolve resolves this provider for configuration and returns an
	// ImageStream (Dart: @nonVirtual).
	Resolve(configuration ImageConfiguration) *ImageStream
	// CreateStream is called by Resolve to create the ImageStream it
	// returns (Dart: @protected).
	CreateStream(configuration ImageConfiguration) *ImageStream
	// ObtainCacheStatus returns the cache location for the key this
	// provider creates for configuration (nil: an error occurred).
	ObtainCacheStatus(configuration ImageConfiguration, opts ...ObtainCacheStatusOptions) async.Future[*ImageCacheStatus]
	// ResolveStreamForKey resolves stream for key: from the image cache, or
	// by loading it (Dart: @protected).
	ResolveStreamForKey(configuration ImageConfiguration, stream *ImageStream, key any, handleError ImageErrorListener)
	// Evict evicts this provider's key from the cache (the default cache
	// and an empty configuration when not given).
	Evict(opts ...ImageProviderEvictOptions) async.Future[bool]
	// ObtainKey converts configuration to a key that describes the precise
	// image to load (abstract).
	ObtainKey(configuration ImageConfiguration) async.Future[any]
	// LoadBuffer is the deprecated loader (Dart: @protected).
	LoadBuffer(key any, decode DecoderBufferCallback) ImageStreamCompleter
	// LoadImage converts a key into a completer and starts fetching the
	// image (Dart: @protected).
	LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter
	String() string
}

// ObtainCacheStatusOptions are obtainCacheStatus's optional named
// parameters (configuration, required, is positional).
type ObtainCacheStatusOptions struct {
	HandleError ImageErrorListener
}

// ImageProviderEvictOptions are evict's named parameters (Cache nil: the
// global image cache; Configuration nil: ImageConfigurationEmpty).
type ImageProviderEvictOptions struct {
	Cache         *ImageCache
	Configuration *ImageConfiguration
}

// ImageProviderBase is ImageProvider's default methods.
type ImageProviderBase struct {
	self ImageProvider
}

// SetSelf points the base at the provider embedding it (call once, when
// creating it).
func (p *ImageProviderBase) SetSelf(self ImageProvider) { p.self = self }

func (p *ImageProviderBase) outer() ImageProvider {
	if p.self == nil {
		panic("painting: ImageProviderBase used before SetSelf; call SetSelf with the provider when creating it")
	}
	return p.self
}

func (p *ImageProviderBase) collector(configuration ImageConfiguration, key any) foundation.InformationCollector {
	if !foundation.KDebugMode {
		return nil
	}
	self := p.outer()
	return func() iter.Seq[foundation.DiagnosticsNode] {
		return slices.Values([]foundation.DiagnosticsNode{
			foundation.NewDiagnosticsProperty[ImageProvider]("Image provider", self),
			foundation.NewDiagnosticsProperty("Image configuration", configuration),
			foundation.NewDiagnosticsProperty("Image key", key, foundation.DiagnosticsPropertyOptions{DefaultValue: foundation.Null}),
		})
	}
}

// Resolve creates the stream, obtains the key and resolves the stream for
// it; an error (from ObtainKey, or from resolving) is reported, an event
// later, to the stream's completer (an error completer if none is set).
func (p *ImageProviderBase) Resolve(configuration ImageConfiguration) *ImageStream {
	self := p.outer()
	stream := self.CreateStream(configuration)
	// Load the key (potentially asynchronously), set up an error handling
	// zone, and call resolveStreamForKey.
	p.createErrorHandlerAndKey(configuration,
		func(key any, errorHandler ImageErrorListener) {
			self.ResolveStreamForKey(configuration, stream, key, errorHandler)
		},
		func(key any, exception any, stack dart.StackTrace) async.Future[async.Void] {
			return async.AsyncVoid(func() {
				// wait an event turn in case a listener has been added to the
				// image stream.
				async.FutureValue[any](nil).Await()
				collector := p.collector(configuration, key)
				if stream.Completer() == nil {
					stream.SetCompleter(newErrorImageCompleter())
				}
				stream.Completer().ReportError(exception, ImageStreamCompleterReportErrorOptions{
					Stack:                stack,
					Context:              foundation.NewErrorDescription("while resolving an image"),
					Silent:               true, // could be a network error or whatnot
					InformationCollector: collector,
				})
			})
		})
	return stream
}

// CreateStream returns NewImageStream().
func (p *ImageProviderBase) CreateStream(configuration ImageConfiguration) *ImageStream {
	return NewImageStream()
}

// ObtainCacheStatus completes with the image cache's status for the key,
// or with nil after reporting an error (to HandleError, or to
// FlutterError.reportError).
func (p *ImageProviderBase) ObtainCacheStatus(configuration ImageConfiguration, opts ...ObtainCacheStatusOptions) async.Future[*ImageCacheStatus] {
	var o ObtainCacheStatusOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	completer := async.NewCompleter[*ImageCacheStatus]()
	p.createErrorHandlerAndKey(configuration,
		func(key any, innerHandleError ImageErrorListener) {
			status := PaintingBindingInstance().ImageCache().StatusForKey(key)
			completer.Complete(&status)
		},
		func(key any, exception any, stack dart.StackTrace) async.Future[async.Void] {
			return async.AsyncVoid(func() {
				if o.HandleError != nil {
					o.HandleError(exception, stack)
				} else {
					foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
						Context:              foundation.NewErrorDescription("while checking the cache location of an image"),
						InformationCollector: p.collector(configuration, key),
						Stack:                stack,
					}))
					completer.Complete(nil)
				}
			})
		})
	return completer.Future()
}

// createErrorHandlerAndKey is _createErrorHandlerAndKey: it obtains the key
// and calls successCallback with it, routing every error (from ObtainKey,
// its future, or successCallback) to errorCallback, once.
func (p *ImageProviderBase) createErrorHandlerAndKey(configuration ImageConfiguration,
	successCallback func(key any, handleError ImageErrorListener),
	errorCallback func(key any, exception any, stack dart.StackTrace) async.Future[async.Void]) {
	var obtainedKey any
	didError := false
	handleError := func(exception any, stack dart.StackTrace) {
		if didError {
			return
		}
		didError = true
		errorCallback(obtainedKey, exception, stack)
	}
	var key async.Future[any]
	if exception, stack, threw := tryCatch(func() { key = p.outer().ObtainKey(configuration) }); threw {
		handleError(exception, stack)
		return
	}
	async.OnComplete(key, func(k any) {
		obtainedKey = k
		if exception, stack, threw := tryCatch(func() { successCallback(k, handleError) }); threw {
			handleError(exception, stack)
		}
	}, func(err error) { handleError(futureErrorValue(err), nil) })
}

// ResolveStreamForKey puts the stream's completer in the cache (when it
// has one), or the cached completer for key, or a new one from LoadImage
// (LoadBuffer for providers that only implement the deprecated loader),
// and sets it on the stream.
func (p *ImageProviderBase) ResolveStreamForKey(configuration ImageConfiguration, stream *ImageStream, key any, handleError ImageErrorListener) {
	self := p.outer()
	cache := PaintingBindingInstance().ImageCache()
	// This is an unusual edge case where someone has told us that they
	// found the image we want before getting to this method. We should
	// avoid calling load again, but still update the image cache with LRU
	// information.
	if stream.Completer() != nil {
		completer := cache.PutIfAbsent(key, func() ImageStreamCompleter { return stream.Completer() },
			ImageCachePutIfAbsentOptions{OnError: handleError})
		dartAssert(completer == stream.Completer())
		return
	}
	completer := cache.PutIfAbsent(key, func() ImageStreamCompleter {
		binding := PaintingBindingInstance()
		result := self.LoadImage(key, binding.InstantiateImageCodecWithSize)
		if _, ok := result.(*abstractImageStreamCompleter); ok {
			result = self.LoadBuffer(key, binding.InstantiateImageCodecFromBuffer)
		}
		return result
	}, ImageCachePutIfAbsentOptions{OnError: handleError})
	if completer != nil {
		stream.SetCompleter(completer)
	}
}

// Evict obtains the key for the configuration and evicts it from cache.
func (p *ImageProviderBase) Evict(opts ...ImageProviderEvictOptions) async.Future[bool] {
	var o ImageProviderEvictOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	configuration := ImageConfigurationEmpty
	if o.Configuration != nil {
		configuration = *o.Configuration
	}
	self := p.outer()
	return async.Async(func() bool {
		cache := o.Cache
		if cache == nil {
			cache = ImageCacheInstance()
		}
		key := self.ObtainKey(configuration).Await()
		return cache.Self().Evict(key)
	})
}

// LoadBuffer is the deprecated loader: by default an abstract completer,
// meaning "not implemented".
func (p *ImageProviderBase) LoadBuffer(key any, decode DecoderBufferCallback) ImageStreamCompleter {
	return newAbstractImageStreamCompleter()
}

// LoadImage returns an abstract completer, meaning "not implemented" (the
// cache then calls LoadBuffer).
func (p *ImageProviderBase) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	return newAbstractImageStreamCompleter()
}

func (p *ImageProviderBase) String() string {
	return foundation.ObjectRuntimeType(p.outer(), "ImageConfiguration") + "()"
}

type abstractImageStreamCompleter struct{ ImageStreamCompleterBase }

func newAbstractImageStreamCompleter() *abstractImageStreamCompleter {
	c := &abstractImageStreamCompleter{}
	c.InitImageStreamCompleter(c)
	return c
}

type errorImageCompleter struct{ ImageStreamCompleterBase }

func newErrorImageCompleter() *errorImageCompleter {
	c := &errorImageCompleter{}
	c.InitImageStreamCompleter(c)
	return c
}

// ---------------------------------------------------------------------------
// AssetBundleImageKey / AssetBundleImageProvider

// AssetBundleImageKey is the key for an image from an AssetBundle
// (comparable with ==).
type AssetBundleImageKey struct {
	// Bundle is the bundle from which the image will be obtained.
	Bundle services.AssetBundle
	// Name is the key to use to obtain the resource from the bundle.
	Name string
	// Scale is the scale to place in the ImageInfo object of the image.
	Scale float64
}

func (k AssetBundleImageKey) String() string {
	bundle := "null"
	if k.Bundle != nil {
		bundle = k.Bundle.String()
	}
	return fmt.Sprintf(`%s(bundle: %s, name: "%s", scale: %s)`, foundation.ObjectRuntimeType(k, "AssetBundleImageKey"),
		bundle, k.Name, dart.DoubleToString(k.Scale))
}

// AssetBundleImageProvider is a subclass of ImageProvider that knows about
// AssetBundles (abstract: AssetImage and ExactAssetImage implement
// ObtainKey).
type AssetBundleImageProvider interface {
	ImageProvider
	assetBundleImageProvider()
}

// AssetBundleImageProviderBase is AssetBundleImageProvider's methods.
type AssetBundleImageProviderBase struct {
	ImageProviderBase
}

func (p *AssetBundleImageProviderBase) assetBundleImageProvider() {}

func (p *AssetBundleImageProviderBase) assetCollector(key AssetBundleImageKey) foundation.InformationCollector {
	if !foundation.KDebugMode {
		return nil
	}
	self := p.outer()
	return func() iter.Seq[foundation.DiagnosticsNode] {
		return slices.Values([]foundation.DiagnosticsNode{
			foundation.NewDiagnosticsProperty[ImageProvider]("Image provider", self),
			foundation.NewDiagnosticsProperty("Image key", key),
		})
	}
}

// LoadImage loads the bundle's data for key and decodes it.
func (p *AssetBundleImageProviderBase) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	k := key.(AssetBundleImageKey)
	name := k.Name
	return NewMultiFrameImageStreamCompleter(p.loadAsync(k, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{DebugLabel: &name, InformationCollector: p.assetCollector(k)})
}

// LoadBuffer is LoadImage with the deprecated decoder.
func (p *AssetBundleImageProviderBase) LoadBuffer(key any, decode DecoderBufferCallback) ImageStreamCompleter {
	k := key.(AssetBundleImageKey)
	name := k.Name
	return NewMultiFrameImageStreamCompleter(p.loadAsync(k, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{DebugLabel: &name, InformationCollector: p.assetCollector(k)})
}

// loadAsync fetches the image from the asset bundle, decodes it, and
// returns a corresponding Codec; a FlutterError from the bundle evicts key
// from the image cache.
func (p *AssetBundleImageProviderBase) loadAsync(key AssetBundleImageKey, decode simpleDecoderCallback) async.Future[ui.Codec] {
	return async.Async(func() ui.Codec {
		var buffer *ui.ImmutableBuffer
		func() {
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(*foundation.FlutterError); ok {
						PaintingBindingInstance().ImageCache().Evict(key)
					}
					panic(r)
				}
			}()
			buffer = key.Bundle.LoadBuffer(key.Name).Await()
		}()
		return decode(buffer).Await()
	})
}

// ---------------------------------------------------------------------------
// ResizeImage

// ResizeImageKey is the key used by ResizeImage to keep the resized
// image's cache entry separate from the original's.
type ResizeImageKey struct {
	providerCacheKey any
	policy           ResizeImagePolicy
	width            *int
	height           *int
	allowUpscaling   bool
}

// Equal is ==.
func (k ResizeImageKey) Equal(other any) bool {
	o, ok := other.(ResizeImageKey)
	return ok && KeysEqual(k.providerCacheKey, o.providerCacheKey) && k.policy == o.policy &&
		ptrEqual(k.width, o.width) && ptrEqual(k.height, o.height) && k.allowUpscaling == o.allowUpscaling
}

// ResizeImagePolicy configures the behavior for ResizeImage.
type ResizeImagePolicy int

const (
	// ResizeImagePolicyExact sizes the image to the exact width and height
	// given (or, with only one given, keeps the aspect ratio).
	ResizeImagePolicyExact ResizeImagePolicy = iota
	// ResizeImagePolicyFit scales the image so that it fits within the
	// width and height, keeping its aspect ratio.
	ResizeImagePolicyFit
)

// ResizeImagePolicyValues is ResizeImagePolicy.values.
var ResizeImagePolicyValues = []ResizeImagePolicy{ResizeImagePolicyExact, ResizeImagePolicyFit}

// Name is the Dart enum value's name.
func (p ResizeImagePolicy) Name() string { return [...]string{"exact", "fit"}[p] }

func (p ResizeImagePolicy) String() string { return "ResizeImagePolicy." + p.Name() }

// ResizeImage instructs Flutter to decode the image at the specified
// dimensions instead of its native size.
type ResizeImage struct {
	ImageProviderBase
	// ImageProvider is the ImageProvider that this class wraps.
	ImageProvider ImageProvider
	// Width and Height are the size to decode at (at least one is set).
	Width, Height *int
	// Policy is the policy determining how Width and Height are used.
	Policy ResizeImagePolicy
	// AllowUpscaling is whether the width and height may exceed the
	// intrinsic size.
	AllowUpscaling bool
	// UseLogicalSize: Width and Height are logical pixels, multiplied by
	// the configuration's device pixel ratio.
	UseLogicalSize bool
}

// ResizeImageOptions are ResizeImage's named parameters.
type ResizeImageOptions struct {
	Width, Height  *int
	Policy         ResizeImagePolicy
	AllowUpscaling bool
	UseLogicalSize bool
}

// NewResizeImage is ResizeImage(imageProvider, {width, height, policy,
// allowUpscaling, useLogicalSize}).
func NewResizeImage(imageProvider ImageProvider, opts ResizeImageOptions) *ResizeImage {
	dartAssert(opts.Width != nil || opts.Height != nil)
	r := &ResizeImage{ImageProvider: imageProvider, Width: opts.Width, Height: opts.Height, Policy: opts.Policy,
		AllowUpscaling: opts.AllowUpscaling, UseLogicalSize: opts.UseLogicalSize}
	r.SetSelf(r)
	return r
}

// ResizeImageResizeIfNeededOptions are resizeIfNeeded's named parameters.
type ResizeImageResizeIfNeededOptions struct {
	UseLogicalSize bool
}

// ResizeImageResizeIfNeeded is ResizeImage.resizeIfNeeded: provider
// wrapped in a ResizeImage when cacheWidth or cacheHeight is set.
func ResizeImageResizeIfNeeded(cacheWidth, cacheHeight *int, provider ImageProvider, opts ...ResizeImageResizeIfNeededOptions) ImageProvider {
	var o ResizeImageResizeIfNeededOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if cacheWidth != nil || cacheHeight != nil {
		return NewResizeImage(provider, ResizeImageOptions{Width: cacheWidth, Height: cacheHeight, UseLogicalSize: o.UseLogicalSize})
	}
	return provider
}

func intString(p *int) string {
	if p == nil {
		return "null"
	}
	return dart.ToString(*p)
}

func debugLabelString(p *string) string {
	if p == nil {
		return "null"
	}
	return *p
}

// LoadBuffer loads the wrapped provider's image with the deprecated
// decoder, at the key's size.
func (r *ResizeImage) LoadBuffer(key any, decode DecoderBufferCallback) ImageStreamCompleter {
	k := key.(ResizeImageKey)
	decodeResize := func(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecFromBufferOptions) async.Future[ui.Codec] {
		if len(opts) > 0 && (opts[0].CacheWidth != nil || opts[0].CacheHeight != nil || opts[0].AllowUpscaling) {
			panic(dart.NewAssertionError("ResizeImage cannot be composed with another ImageProvider that applies " +
				"cacheWidth, cacheHeight, or allowUpscaling."))
		}
		return decode(buffer, InstantiateImageCodecFromBufferOptions{CacheWidth: k.width, CacheHeight: k.height, AllowUpscaling: r.AllowUpscaling})
	}
	completer := r.ImageProvider.LoadBuffer(k.providerCacheKey, decodeResize)
	if !foundation.KReleaseMode {
		label := debugLabelString(completer.DebugLabel()) + " - Resized(" + intString(k.width) + "×" + intString(k.height) + ")"
		completer.SetDebugLabel(&label)
	}
	r.configureErrorListener(completer, k)
	return completer
}

// ResizeImageTargetSize is the size ResizeImage decodes at for an image of
// the given intrinsic size (the getTargetSize LoadImage passes to the
// decoder). It is a flutter-go export of the Dart closure, for tests.
func ResizeImageTargetSize(policy ResizeImagePolicy, width, height *int, allowUpscaling bool, intrinsicWidth, intrinsicHeight int) ui.TargetImageSize {
	effectiveWidth, effectiveHeight := width, height
	switch policy {
	case ResizeImagePolicyExact:
		targetWidth, targetHeight := effectiveWidth, effectiveHeight
		if !allowUpscaling {
			if targetWidth != nil && *targetWidth > intrinsicWidth {
				targetWidth = &intrinsicWidth
			}
			if targetHeight != nil && *targetHeight > intrinsicHeight {
				targetHeight = &intrinsicHeight
			}
		}
		return ui.NewTargetImageSize(ui.TargetImageSize{Width: targetWidth, Height: targetHeight})
	default: // ResizeImagePolicyFit
		aspectRatio := float64(intrinsicWidth) / float64(intrinsicHeight)
		maxWidth, maxHeight := intrinsicWidth, intrinsicHeight
		if effectiveWidth != nil {
			maxWidth = *effectiveWidth
		}
		if effectiveHeight != nil {
			maxHeight = *effectiveHeight
		}
		targetWidth, targetHeight := intrinsicWidth, intrinsicHeight
		if targetWidth > maxWidth {
			targetWidth = maxWidth
			targetHeight = intTruncDiv(targetWidth, aspectRatio)
		}
		if targetHeight > maxHeight {
			targetHeight = maxHeight
			targetWidth = dartFloor(float64(targetHeight) * aspectRatio)
		}
		if allowUpscaling {
			if effectiveWidth == nil {
				dartAssert(effectiveHeight != nil)
				targetHeight = *effectiveHeight
				targetWidth = dartFloor(float64(targetHeight) * aspectRatio)
			} else if effectiveHeight == nil {
				targetWidth = *effectiveWidth
				targetHeight = intTruncDiv(targetWidth, aspectRatio)
			} else {
				derivedMaxWidth := dartFloor(float64(maxHeight) * aspectRatio)
				derivedMaxHeight := intTruncDiv(maxWidth, aspectRatio)
				targetWidth = min(maxWidth, derivedMaxWidth)
				targetHeight = min(maxHeight, derivedMaxHeight)
			}
		}
		return ui.NewTargetImageSize(ui.TargetImageSize{Width: &targetWidth, Height: &targetHeight})
	}
}

// intTruncDiv is Dart's int ~/ double: the double quotient, truncated
// (UnsupportedError for infinity or NaN).
func intTruncDiv(a int, b float64) int {
	q := math.Trunc(float64(a) / b)
	if math.IsInf(q, 0) || math.IsNaN(q) {
		panic(dart.UnsupportedError{Message: "Infinity or NaN toInt"})
	}
	return int(q)
}

// dartFloor is double.floor() (UnsupportedError for infinity or NaN).
func dartFloor(v float64) int {
	f := math.Floor(v)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		panic(dart.UnsupportedError{Message: "Infinity or NaN toInt"})
	}
	return int(f)
}

// dartCeil is double.ceil() (UnsupportedError for infinity or NaN).
func dartCeil(v float64) int {
	f := math.Ceil(v)
	if math.IsInf(f, 0) || math.IsNaN(f) {
		panic(dart.UnsupportedError{Message: "Infinity or NaN toInt"})
	}
	return int(f)
}

// LoadImage loads the wrapped provider's image, decoding it at the size
// ResizeImageTargetSize picks.
func (r *ResizeImage) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	k := key.(ResizeImageKey)
	decodeResize := func(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec] {
		if len(opts) > 0 && opts[0].GetTargetSize != nil {
			panic(dart.NewAssertionError("ResizeImage cannot be composed with another ImageProvider that applies " +
				"getTargetSize."))
		}
		return decode(buffer, InstantiateImageCodecWithSizeOptions{
			GetTargetSize: func(intrinsicWidth, intrinsicHeight int) ui.TargetImageSize {
				return ResizeImageTargetSize(r.Policy, k.width, k.height, r.AllowUpscaling, intrinsicWidth, intrinsicHeight)
			},
		})
	}
	completer := r.ImageProvider.LoadImage(k.providerCacheKey, decodeResize)
	if !foundation.KReleaseMode {
		label := debugLabelString(completer.DebugLabel()) + " - Resized(" + intString(k.width) + "×" + intString(k.height) + ")"
		completer.SetDebugLabel(&label)
	}
	r.configureErrorListener(completer, k)
	return completer
}

func (r *ResizeImage) configureErrorListener(completer ImageStreamCompleter, key ResizeImageKey) {
	completer.AddEphemeralErrorListener(func(exception any, stackTrace dart.StackTrace) {
		// The microtask is scheduled because of the same reason as
		// NetworkImage: Depending on where the exception is thrown, the
		// image cache may not be updated yet.
		async.ScheduleMicrotask(func() { PaintingBindingInstance().ImageCache().Evict(key) })
	})
}

// ObtainKey obtains the wrapped provider's key and wraps it with the
// (device-pixel-scaled, when UseLogicalSize) size; synchronously when the
// wrapped key is.
func (r *ResizeImage) ObtainKey(configuration ImageConfiguration) async.Future[any] {
	devicePixelRatio := 1.0
	if r.UseLogicalSize {
		if configuration.DevicePixelRatio != nil {
			devicePixelRatio = *configuration.DevicePixelRatio
		}
	}
	var effectiveWidth, effectiveHeight *int
	if r.Width != nil {
		w := dartCeil(float64(*r.Width) * devicePixelRatio)
		effectiveWidth = &w
	}
	if r.Height != nil {
		h := dartCeil(float64(*r.Height) * devicePixelRatio)
		effectiveHeight = &h
	}
	var completer *async.Completer[any]
	// If the imageProvider.obtainKey future is synchronous, then we will be
	// able to fill in result with a value before completer is initialized
	// below.
	var result async.Future[any]
	async.Then(r.ImageProvider.ObtainKey(configuration), func(key any) any {
		k := ResizeImageKey{key, r.Policy, effectiveWidth, effectiveHeight, r.AllowUpscaling}
		if completer == nil {
			// This future has completed synchronously (completer was never
			// assigned), so we can directly create the synchronous result to
			// return.
			result = foundation.NewSynchronousFuture[any](k)
		} else {
			// This future did not synchronously complete.
			completer.Complete(k)
		}
		return nil
	})
	if result != nil {
		return result
	}
	// If the code reaches here, it means the imageProvider.obtainKey was
	// not completed sync, so we initialize the completer for completion
	// later.
	completer = async.NewCompleter[any]()
	return completer.Future()
}

// Equal is ==.
func (r *ResizeImage) Equal(other any) bool {
	o, ok := other.(*ResizeImage)
	if !ok {
		return false
	}
	if r == o {
		return true
	}
	return KeysEqual(r.ImageProvider, o.ImageProvider) && ptrEqual(r.Width, o.Width) && ptrEqual(r.Height, o.Height) &&
		r.Policy == o.Policy && r.AllowUpscaling == o.AllowUpscaling && r.UseLogicalSize == o.UseLogicalSize
}

// ---------------------------------------------------------------------------
// WebHtmlElementStrategy

// WebHtmlElementStrategy is how NetworkImage uses <img> elements on the
// web (unused elsewhere).
type WebHtmlElementStrategy int

const (
	// WebHtmlElementStrategyNever: only use the image codec.
	WebHtmlElementStrategyNever WebHtmlElementStrategy = iota
	// WebHtmlElementStrategyFallback: use an <img> element if the codec
	// fails.
	WebHtmlElementStrategyFallback
	// WebHtmlElementStrategyPrefer: use an <img> element when possible.
	WebHtmlElementStrategyPrefer
)

// WebHtmlElementStrategyValues is WebHtmlElementStrategy.values.
var WebHtmlElementStrategyValues = []WebHtmlElementStrategy{WebHtmlElementStrategyNever,
	WebHtmlElementStrategyFallback, WebHtmlElementStrategyPrefer}

// Name is the Dart enum value's name.
func (s WebHtmlElementStrategy) Name() string { return [...]string{"never", "fallback", "prefer"}[s] }

func (s WebHtmlElementStrategy) String() string { return "WebHtmlElementStrategy." + s.Name() }

// ---------------------------------------------------------------------------
// FileImage

// FileImage decodes the given File object as an image, associating it
// with the given scale. Its key is itself.
type FileImage struct {
	ImageProviderBase
	// File is the file to decode into an image.
	File io.File
	// Scale is the scale to place in the ImageInfo object of the image.
	Scale float64
}

// FileImageOptions are FileImage's named parameters (Scale nil: 1.0).
type FileImageOptions struct{ Scale *float64 }

// NewFileImage is FileImage(file, {scale}).
func NewFileImage(file io.File, opts ...FileImageOptions) *FileImage {
	p := &FileImage{File: file, Scale: 1.0}
	if len(opts) > 0 && opts[0].Scale != nil {
		p.Scale = *opts[0].Scale
	}
	p.SetSelf(p)
	return p
}

// ObtainKey returns the provider itself, synchronously.
func (p *FileImage) ObtainKey(configuration ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *FileImage) fileCollector() foundation.InformationCollector {
	return func() iter.Seq[foundation.DiagnosticsNode] {
		return slices.Values([]foundation.DiagnosticsNode{foundation.NewErrorDescription("Path: " + p.File.Path())})
	}
}

// LoadBuffer is LoadImage with the deprecated decoder.
func (p *FileImage) LoadBuffer(key any, decode DecoderBufferCallback) ImageStreamCompleter {
	k := key.(*FileImage)
	path := k.File.Path()
	return NewMultiFrameImageStreamCompleter(p.loadAsync(k, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{DebugLabel: &path, InformationCollector: p.fileCollector()})
}

// LoadImage reads and decodes the file.
func (p *FileImage) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	k := key.(*FileImage)
	path := k.File.Path()
	return NewMultiFrameImageStreamCompleter(p.loadAsync(k, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{DebugLabel: &path, InformationCollector: p.fileCollector()})
}

func (p *FileImage) loadAsync(key *FileImage, decode simpleDecoderCallback) async.Future[ui.Codec] {
	return async.Async(func() ui.Codec {
		dartAssert(key.Equal(p))
		lengthInBytes := p.File.Length().Await()
		if lengthInBytes == 0 {
			// The file may become available later.
			PaintingBindingInstance().ImageCache().Evict(key)
			panic(dart.StateError{Message: p.File.String() + " is empty and cannot be loaded as an image."})
		}
		if _, ok := p.File.(*io.LocalFile); ok {
			return decode(ui.ImmutableBufferFromFilePath(p.File.Path()).Await()).Await()
		}
		return decode(ui.ImmutableBufferFromUint8List(p.File.ReadAsBytes().Await()).Await()).Await()
	})
}

// Equal is ==: same file path and scale.
func (p *FileImage) Equal(other any) bool {
	o, ok := other.(*FileImage)
	return ok && o.File.Path() == p.File.Path() && o.Scale == p.Scale
}

func (p *FileImage) String() string {
	return fmt.Sprintf(`%s("%s", scale: %s)`, foundation.ObjectRuntimeType(p, "FileImage"), p.File.Path(), dart.ToStringAsFixed(p.Scale, 1))
}

// ---------------------------------------------------------------------------
// MemoryImage

// MemoryImage decodes the given bytes as an image, associating it with the
// given scale. Its key is itself; two MemoryImages are equal when they
// hold the same byte list (identity, as Dart's Uint8List ==) and scale.
type MemoryImage struct {
	ImageProviderBase
	// Bytes is the bytes to decode into an image.
	Bytes []byte
	// Scale is the scale to place in the ImageInfo object of the image.
	Scale float64
}

// MemoryImageOptions are MemoryImage's named parameters (Scale nil: 1.0).
type MemoryImageOptions struct{ Scale *float64 }

// NewMemoryImage is MemoryImage(bytes, {scale}).
func NewMemoryImage(bytes []byte, opts ...MemoryImageOptions) *MemoryImage {
	p := &MemoryImage{Bytes: bytes, Scale: 1.0}
	if len(opts) > 0 && opts[0].Scale != nil {
		p.Scale = *opts[0].Scale
	}
	p.SetSelf(p)
	return p
}

// ObtainKey returns the provider itself, synchronously.
func (p *MemoryImage) ObtainKey(configuration ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

// describeBytes is describeIdentity(bytes) for a Uint8List.
func describeBytes(b []byte) string {
	return "Uint8List#" + foundation.ShortHash(unsafe.SliceData(b))
}

// LoadBuffer is LoadImage with the deprecated decoder.
func (p *MemoryImage) LoadBuffer(key any, decode DecoderBufferCallback) ImageStreamCompleter {
	k := key.(*MemoryImage)
	dartAssert(k.Equal(p))
	label := "MemoryImage(" + describeBytes(k.Bytes) + ")"
	return NewMultiFrameImageStreamCompleter(p.loadAsync(k, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{DebugLabel: &label})
}

// LoadImage decodes the bytes.
func (p *MemoryImage) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	k := key.(*MemoryImage)
	label := "MemoryImage(" + describeBytes(k.Bytes) + ")"
	return NewMultiFrameImageStreamCompleter(p.loadAsync(k, func(b *ui.ImmutableBuffer) async.Future[ui.Codec] { return decode(b) }),
		k.Scale, MultiFrameImageStreamCompleterOptions{DebugLabel: &label})
}

func (p *MemoryImage) loadAsync(key *MemoryImage, decode simpleDecoderCallback) async.Future[ui.Codec] {
	return async.Async(func() ui.Codec {
		dartAssert(key.Equal(p))
		return decode(ui.ImmutableBufferFromUint8List(p.Bytes).Await()).Await()
	})
}

func sameBytes(a, b []byte) bool {
	return unsafe.SliceData(a) == unsafe.SliceData(b) && len(a) == len(b) && (a == nil) == (b == nil)
}

// Equal is ==: the same byte list and scale.
func (p *MemoryImage) Equal(other any) bool {
	o, ok := other.(*MemoryImage)
	return ok && sameBytes(o.Bytes, p.Bytes) && o.Scale == p.Scale
}

func (p *MemoryImage) String() string {
	return fmt.Sprintf("%s(%s, scale: %s)", foundation.ObjectRuntimeType(p, "MemoryImage"), describeBytes(p.Bytes), dart.ToStringAsFixed(p.Scale, 1))
}

// ---------------------------------------------------------------------------
// ExactAssetImage

// ExactAssetImage fetches an image from an AssetBundle, with an explicit
// scale (no variant resolution).
type ExactAssetImage struct {
	AssetBundleImageProviderBase
	// AssetName is the name of the asset.
	AssetName string
	// Scale is the scale to place in the ImageInfo object of the image.
	Scale float64
	// Bundle is the bundle from which the image will be obtained (nil: the
	// configuration's, or RootBundle).
	Bundle services.AssetBundle
	// Package is the name of the package from which the image is included
	// (nil: none).
	Package *string
}

// ExactAssetImageOptions are ExactAssetImage's named parameters (Scale
// nil: 1.0).
type ExactAssetImageOptions struct {
	Scale   *float64
	Bundle  services.AssetBundle
	Package *string
}

// NewExactAssetImage is ExactAssetImage(assetName, {scale, bundle,
// package}).
func NewExactAssetImage(assetName string, opts ...ExactAssetImageOptions) *ExactAssetImage {
	var o ExactAssetImageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &ExactAssetImage{AssetName: assetName, Scale: 1.0, Bundle: o.Bundle, Package: o.Package}
	if o.Scale != nil {
		p.Scale = *o.Scale
	}
	p.SetSelf(p)
	return p
}

// KeyName is the key to use to obtain the resource from the bundle
// ("packages/<package>/<assetName>" for a package's asset).
func (p *ExactAssetImage) KeyName() string {
	if p.Package == nil {
		return p.AssetName
	}
	return "packages/" + *p.Package + "/" + p.AssetName
}

// ObtainKey returns the asset's key, synchronously.
func (p *ExactAssetImage) ObtainKey(configuration ImageConfiguration) async.Future[any] {
	bundle := p.Bundle
	if bundle == nil {
		bundle = configuration.Bundle
	}
	if bundle == nil {
		bundle = services.RootBundle
	}
	return foundation.NewSynchronousFuture[any](AssetBundleImageKey{Bundle: bundle, Name: p.KeyName(), Scale: p.Scale})
}

// Equal is ==: same key name, scale and bundle.
func (p *ExactAssetImage) Equal(other any) bool {
	o, ok := other.(*ExactAssetImage)
	return ok && o.KeyName() == p.KeyName() && o.Scale == p.Scale && o.Bundle == p.Bundle
}

func (p *ExactAssetImage) String() string {
	bundle := "null"
	if p.Bundle != nil {
		bundle = p.Bundle.String()
	}
	return fmt.Sprintf(`%s(name: "%s", scale: %s, bundle: %s)`, foundation.ObjectRuntimeType(p, "ExactAssetImage"),
		p.KeyName(), dart.ToStringAsFixed(p.Scale, 1), bundle)
}

// ---------------------------------------------------------------------------
// NetworkImageLoadException

// NetworkImageLoadException is the exception thrown when a NetworkImage's
// HTTP request fails.
type NetworkImageLoadException struct {
	// StatusCode is the HTTP status code from the server.
	StatusCode int
	// URI is the requested URI.
	URI     *url.URL
	message string
}

// NewNetworkImageLoadException is NetworkImageLoadException(statusCode:,
// uri:).
func NewNetworkImageLoadException(statusCode int, uri *url.URL) *NetworkImageLoadException {
	return &NetworkImageLoadException{StatusCode: statusCode, URI: uri,
		message: "HTTP request failed, statusCode: " + dart.ToString(statusCode) + ", " + uri.String()}
}

func (e *NetworkImageLoadException) Error() string  { return e.message }
func (e *NetworkImageLoadException) String() string { return e.message }
