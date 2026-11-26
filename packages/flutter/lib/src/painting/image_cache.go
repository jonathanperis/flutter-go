package painting

// Port of image_cache.dart (Flutter 3.50.1): ImageCache and
// ImageCacheStatus.
//
// Keys are `any` (Dart's Object) compared as Dart's map compares them: a
// key with an `Equal(other any) bool` method uses it (the image providers'
// keys whose Dart == is not Go's ==, e.g. MemoryImage's identical bytes),
// other keys use Go's == (a key of an uncomparable type equals nothing but
// itself... and is never found: use comparable keys). The three maps keep
// Dart's insertion order, which decides the least-recently-used image
// evicted first. The timeline events Dart records (TimelineTask) are not
// recorded (no timeline recorder, see src/developer).

import (
	"fmt"
	"reflect"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
)

const (
	kDefaultSize      = 1000
	kDefaultSizeBytes = 100 << 20 // 100 MiB
)

// KeysEqual is Dart's == on image cache keys: Equal(any) bool when the key
// has it, Go's == otherwise (false for uncomparable values).
func KeysEqual(a, b any) (eq bool) {
	if e, ok := a.(interface{ Equal(other any) bool }); ok {
		return e.Equal(b)
	}
	defer func() {
		if recover() != nil {
			eq = false
		}
	}()
	return a == b
}

func typeName(v any) string {
	if v == nil {
		return "Null"
	}
	return reflect.TypeOf(v).String()
}

// orderedMap is a Dart LinkedHashMap with KeysEqual keys.
type orderedMap[V any] struct {
	keys []any
	vals []V
}

func (m *orderedMap[V]) index(key any) int {
	for i, k := range m.keys {
		if KeysEqual(k, key) {
			return i
		}
	}
	return -1
}

func (m *orderedMap[V]) get(key any) (V, bool) {
	if i := m.index(key); i >= 0 {
		return m.vals[i], true
	}
	var zero V
	return zero, false
}

func (m *orderedMap[V]) set(key any, v V) {
	if i := m.index(key); i >= 0 {
		m.vals[i] = v
		return
	}
	m.keys = append(m.keys, key)
	m.vals = append(m.vals, v)
}

func (m *orderedMap[V]) remove(key any) (V, bool) {
	i := m.index(key)
	if i < 0 {
		var zero V
		return zero, false
	}
	v := m.vals[i]
	m.keys = append(m.keys[:i:i], m.keys[i+1:]...)
	m.vals = append(m.vals[:i:i], m.vals[i+1:]...)
	return v, true
}

func (m *orderedMap[V]) length() int { return len(m.keys) }

func (m *orderedMap[V]) clear() { m.keys, m.vals = nil, nil }

// ImageCache is the class for the image cache (PaintingBinding's
// imageCache): it keeps the most recently used images (up to MaximumSize
// images and MaximumSizeBytes bytes), tracks images being loaded (pending)
// and images in use (live).
type ImageCache struct {
	pendingImages orderedMap[*pendingImage]
	cache         orderedMap[*cachedImage]
	liveImages    orderedMap[*liveImage]

	maximumSize      int
	maximumSizeBytes int
	currentSizeBytes int

	self ImageCacheOverrides
}

// ImageCacheOverrides are the ImageCache methods a type extending it may
// override (Flutter's tests and apps subclass ImageCache and return it from
// PaintingBinding.CreateImageCache). The cache and the binding call them
// through Self().
type ImageCacheOverrides interface {
	Clear()
	ClearLiveImages()
	Evict(key any, opts ...ImageCacheEvictOptions) bool
}

// SetSelf points the cache at the value embedding it (call once, when
// creating it); Init sets the defaults of an embedded cache.
func (c *ImageCache) SetSelf(self ImageCacheOverrides) { c.self = self }

// Init sets up an ImageCache embedded in self (Dart's ImageCache()
// constructor for a subclass).
func (c *ImageCache) Init(self ImageCacheOverrides) {
	c.maximumSize, c.maximumSizeBytes, c.self = kDefaultSize, kDefaultSizeBytes, self
}

// Self is the value this cache is embedded in (itself when not extended).
func (c *ImageCache) Self() ImageCacheOverrides {
	if c.self == nil {
		return c
	}
	return c.self
}

// NewImageCache is ImageCache().
func NewImageCache() *ImageCache {
	return &ImageCache{maximumSize: kDefaultSize, maximumSizeBytes: kDefaultSizeBytes}
}

// MaximumSize is the maximum number of images to cache (1000 by default).
func (c *ImageCache) MaximumSize() int { return c.maximumSize }

// SetMaximumSize changes the maximum cache size; 0 clears the cache, a
// smaller size evicts the least recently used images.
func (c *ImageCache) SetMaximumSize(value int) {
	dartAssert(value >= 0)
	if value == c.maximumSize {
		return
	}
	c.maximumSize = value
	if c.maximumSize == 0 {
		c.Self().Clear()
	} else {
		c.checkCacheSize()
	}
}

// CurrentSize is the current number of cached entries.
func (c *ImageCache) CurrentSize() int { return c.cache.length() }

// MaximumSizeBytes is the maximum size of the cache in bytes (100 MiB by
// default).
func (c *ImageCache) MaximumSizeBytes() int { return c.maximumSizeBytes }

// SetMaximumSizeBytes changes the maximum cache size in bytes; 0 clears
// the cache.
func (c *ImageCache) SetMaximumSizeBytes(value int) {
	dartAssert(value >= 0)
	if value == c.maximumSizeBytes {
		return
	}
	c.maximumSizeBytes = value
	if c.maximumSizeBytes == 0 {
		c.Self().Clear()
	} else {
		c.checkCacheSize()
	}
}

// CurrentSizeBytes is the current size of the cache in bytes.
func (c *ImageCache) CurrentSizeBytes() int { return c.currentSizeBytes }

// Clear evicts all pending and keepAlive entries from the cache (live
// images stay tracked).
func (c *ImageCache) Clear() {
	for _, image := range c.cache.vals {
		image.dispose()
	}
	c.cache.clear()
	for _, p := range c.pendingImages.vals {
		p.removeListener()
	}
	c.pendingImages.clear()
	c.currentSizeBytes = 0
}

// ImageCacheEvictOptions are evict's named parameters (IncludeLive nil:
// true).
type ImageCacheEvictOptions struct {
	IncludeLive *bool
}

// Evict removes the key and its image from the cache (and from the live
// images unless IncludeLive is false); it reports whether the key was
// pending or cached.
func (c *ImageCache) Evict(key any, opts ...ImageCacheEvictOptions) bool {
	includeLive := len(opts) == 0 || opts[0].IncludeLive == nil || *opts[0].IncludeLive
	if includeLive {
		// Remove from live images - the cache will not be able to mark it
		// as complete, and it might be getting evicted because it will
		// never complete, e.g. it was loaded in a FakeAsync zone. In such a
		// case, we need to make sure subsequent calls to the provider are
		// able to get the image.
		if image, ok := c.liveImages.remove(key); ok {
			image.dispose()
		}
	}
	if p, ok := c.pendingImages.remove(key); ok {
		p.removeListener()
		return true
	}
	if image, ok := c.cache.remove(key); ok {
		c.currentSizeBytes -= *image.sizeBytes
		image.dispose()
		return true
	}
	return false
}

// touch adds the image to the cache's keepAlive images when it fits,
// disposes it otherwise.
func (c *ImageCache) touch(key any, image *cachedImage) {
	if image.sizeBytes != nil && *image.sizeBytes <= c.maximumSizeBytes && c.maximumSize > 0 {
		c.currentSizeBytes += *image.sizeBytes
		c.cache.set(key, image)
		c.checkCacheSize()
	} else {
		image.dispose()
	}
}

func (c *ImageCache) trackLiveImage(key any, completer ImageStreamCompleter, sizeBytes *int) {
	// Avoid adding unnecessary callbacks to the completer.
	image, ok := c.liveImages.get(key)
	if !ok {
		image = newLiveImage(completer, func() { c.liveImages.remove(key) })
		c.liveImages.set(key, image)
	}
	// We may be transitioning from a pending to a live image, so update the
	// size if it is still unknown.
	if image.sizeBytes == nil {
		image.sizeBytes = sizeBytes
	}
}

// ImageCachePutIfAbsentOptions are putIfAbsent's named parameters.
type ImageCachePutIfAbsentOptions struct {
	OnError ImageErrorListener
}

// PutIfAbsent returns the completer for key: the pending, cached or live
// one, or loader's (which is then tracked, and cached once its first image
// arrives). If loader panics, OnError is called with the exception (and
// PutIfAbsent returns nil), or the panic propagates when OnError is nil.
func (c *ImageCache) PutIfAbsent(key any, loader func() ImageStreamCompleter, opts ...ImageCachePutIfAbsentOptions) ImageStreamCompleter {
	var o ImageCachePutIfAbsentOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if p, ok := c.pendingImages.get(key); ok && p.completer != nil {
		return p.completer
	}
	// Remove the provider from the list so that we can move it to the
	// recently used position below. Don't use _touch here, which would
	// trigger a check on cache size that is not needed since this is just
	// moving an existing cache entry to the head of the list.
	if image, ok := c.cache.remove(key); ok {
		// The image might have been keptAlive but had no listeners (so not
		// live). Make sure the cache starts tracking it as live again.
		c.trackLiveImage(key, image.completer, image.sizeBytes)
		c.cache.set(key, image)
		return image.completer
	}
	if live, ok := c.liveImages.get(key); ok {
		c.touch(key, newCachedImage(live.completer, live.sizeBytes))
		return live.completer
	}
	var result ImageStreamCompleter
	if exception, stack, threw := tryCatch(func() {
		result = loader()
		c.trackLiveImage(key, result, nil)
	}); threw {
		if o.OnError != nil {
			o.OnError(exception, stack)
			return nil
		}
		panic(exception)
	}
	// The listener we add here never removes itself; we remove it with
	// pendingImage.removeListener once the first image (or error) arrives,
	// or when the key is evicted while pending.
	listenedOnce := false
	// We shouldn't use the _pendingImages map if the cache is disabled, but
	// we will have to listen to the image at least once so we don't leak
	// it in the live image tracking.
	trackPendingImage := c.maximumSize > 0 && c.maximumSizeBytes > 0
	var pending *pendingImage
	listener := func(info *ImageInfo, syncCall bool) {
		var sizeBytes *int
		if info != nil {
			n := info.SizeBytes()
			sizeBytes = &n
			info.Dispose()
		}
		image := newCachedImage(result, sizeBytes)
		c.trackLiveImage(key, result, sizeBytes)
		// Only touch if the cache was enabled when resolve was initially
		// called.
		if trackPendingImage {
			c.touch(key, image)
		} else {
			image.dispose()
		}
		c.pendingImages.remove(key)
		if !listenedOnce {
			pending.removeListener()
		}
		listenedOnce = true
	}
	streamListener := NewImageStreamListener(listener)
	pending = &pendingImage{completer: result, listener: streamListener}
	if trackPendingImage {
		c.pendingImages.set(key, pending)
	}
	// Listener is removed in [_PendingImage.removeListener].
	result.AddListener(streamListener)
	return result
}

// StatusForKey is the ImageCacheStatus of key.
func (c *ImageCache) StatusForKey(key any) ImageCacheStatus {
	_, pending := c.pendingImages.get(key)
	_, keepAlive := c.cache.get(key)
	_, live := c.liveImages.get(key)
	return newImageCacheStatus(pending, keepAlive, live)
}

// ContainsKey is whether key is pending or cached (not live only).
func (c *ImageCache) ContainsKey(key any) bool {
	p, ok := c.pendingImages.get(key)
	if ok && p != nil {
		return true
	}
	im, ok := c.cache.get(key)
	return ok && im != nil
}

// LiveImageCount is the number of live images (with listeners).
func (c *ImageCache) LiveImageCount() int { return c.liveImages.length() }

// PendingImageCount is the number of images being loaded.
func (c *ImageCache) PendingImageCount() int { return c.pendingImages.length() }

// ClearLiveImages stops tracking the live images (they may then be
// reloaded rather than shared).
func (c *ImageCache) ClearLiveImages() {
	for _, image := range c.liveImages.vals {
		image.dispose()
	}
	c.liveImages.clear()
}

// checkCacheSize evicts the least recently used images until the cache
// fits its limits.
func (c *ImageCache) checkCacheSize() {
	for c.currentSizeBytes > c.maximumSizeBytes || c.cache.length() > c.maximumSize {
		key := c.cache.keys[0]
		image := c.cache.vals[0]
		c.currentSizeBytes -= *image.sizeBytes
		image.dispose()
		c.cache.remove(key)
	}
	dartAssert(c.currentSizeBytes >= 0)
	dartAssert(c.cache.length() <= c.maximumSize)
	dartAssert(c.currentSizeBytes <= c.maximumSizeBytes)
}

// ImageCacheStatus is the information about how the ImageCache is
// tracking an image (comparable with ==).
type ImageCacheStatus struct {
	// Pending: the image is being loaded.
	Pending bool
	// KeepAlive: the image is cached (not necessarily used).
	KeepAlive bool
	// Live: the image has listeners.
	Live bool
}

func newImageCacheStatus(pending, keepAlive, live bool) ImageCacheStatus {
	dartAssert(!pending || !keepAlive)
	return ImageCacheStatus{Pending: pending, KeepAlive: keepAlive, Live: live}
}

// Tracked is whether the cache tracks the image in any way.
func (s ImageCacheStatus) Tracked() bool { return s.Pending || s.KeepAlive || s.Live }

// Untracked is whether the cache doesn't track the image.
func (s ImageCacheStatus) Untracked() bool { return !s.Pending && !s.KeepAlive && !s.Live }

func (s ImageCacheStatus) String() string {
	return fmt.Sprintf("%s(pending: %t, live: %t, keepAlive: %t)",
		foundation.ObjectRuntimeType(s, "ImageCacheStatus"), s.Pending, s.Live, s.KeepAlive)
}

// cachedImageBase is _CachedImageBase: an entry holding a keep-alive
// handle on its completer, released after the next frame when disposed.
type cachedImageBase struct {
	completer ImageStreamCompleter
	sizeBytes *int
	handle    *ImageStreamCompleterHandle
}

func (b *cachedImageBase) init(completer ImageStreamCompleter, sizeBytes *int) {
	b.completer, b.sizeBytes, b.handle = completer, sizeBytes, completer.KeepAlive()
}

func (b *cachedImageBase) disposeBase(owner any) {
	dartAssert(b.handle != nil)
	dartAssert(foundation.DebugMaybeDispatchDisposed(owner))
	// Give any interested parties a chance to listen to the stream before
	// we potentially dispose it.
	scheduler.SchedulerBindingInstance().AddPostFrameCallback(func(timeStamp time.Duration) {
		dartAssert(b.handle != nil)
		if b.handle != nil {
			b.handle.Dispose()
		}
		b.handle = nil
	}, scheduler.AddPostFrameCallbackOptions{DebugLabel: "CachedImage.disposeHandle"})
}

type cachedImage struct{ cachedImageBase }

func newCachedImage(completer ImageStreamCompleter, sizeBytes *int) *cachedImage {
	i := &cachedImage{}
	i.init(completer, sizeBytes)
	dartAssert(foundation.DebugMaybeDispatchCreated("painting", "_CachedImageBase", i))
	return i
}

func (i *cachedImage) dispose() { i.disposeBase(i) }

type liveImage struct {
	cachedImageBase
	handleRemove func()
}

func newLiveImage(completer ImageStreamCompleter, handleRemove func()) *liveImage {
	i := &liveImage{}
	i.init(completer, nil)
	dartAssert(foundation.DebugMaybeDispatchCreated("painting", "_CachedImageBase", i))
	i.handleRemove = func() {
		handleRemove()
		i.dispose()
	}
	completer.AddOnLastListenerRemovedCallback(i.handleRemove)
	return i
}

func (i *liveImage) dispose() {
	i.completer.RemoveOnLastListenerRemovedCallback(i.handleRemove)
	i.disposeBase(i)
}

func (i *liveImage) String() string { return foundation.DescribeIdentity(i) }

type pendingImage struct {
	completer ImageStreamCompleter
	listener  ImageStreamListener
}

func (p *pendingImage) removeListener() { p.completer.RemoveListener(p.listener) }
