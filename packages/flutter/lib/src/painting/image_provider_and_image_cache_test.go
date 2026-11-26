package painting_test

// Port of packages/flutter/test/painting/image_provider_and_image_cache_test.dart
// and binding_test.dart (Flutter 3.50.1): all 7 tests. binding_test's
// TestBindingBase (a BindingBase whose members do nothing) is the
// painting test binding in its own process.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func basicDecoder(bytes *ui.ImmutableBuffer, opts ...InstantiateImageCodecFromBufferOptions) async.Future[ui.Codec] {
	return PaintingBindingInstance().InstantiateImageCodecFromBuffer(bytes, opts...)
}

func TestAssetImageProviderEvictsOnFailureToLoad(t *testing.T) {
	tester := newProviderTester(t)
	errC := async.NewCompleter[*foundation.FlutterError]()
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		errC.Complete(details.Exception.(*foundation.FlutterError))
	}
	provider := NewExactAssetImage("does-not-exist")
	key := await(tester, provider.ObtainKey(ImageConfigurationEmpty))
	c := ImageCacheInstance()
	if !c.StatusForKey(provider).Untracked() || c.PendingImageCount() != 0 {
		t.Fatal("tracked")
	}
	provider.Resolve(ImageConfigurationEmpty)
	if !c.StatusForKey(key).Pending || c.PendingImageCount() != 1 {
		t.Fatal("not pending")
	}
	await(tester, errC.Future())
	if !c.StatusForKey(provider).Untracked() || c.PendingImageCount() != 0 {
		t.Error("still tracked")
	}
}

func TestImageProviderCanEvictImages(t *testing.T) {
	tester := newProviderTester(t)
	bytes := append([]byte{}, kTransparentImage...)
	imageProvider := NewMemoryImage(bytes)
	stream := imageProvider.Resolve(ImageConfigurationEmpty)
	completer := async.NewCompleter[async.Void]()
	stream.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { completer.Complete(async.Void{}) }))
	await(tester, completer.Future())
	if ImageCacheInstance().CurrentSize() != 1 {
		t.Fatal("not cached")
	}
	if !await(tester, NewMemoryImage(bytes).Evict()) {
		t.Error("evict false")
	}
	if ImageCacheInstance().CurrentSize() != 0 {
		t.Error("still cached")
	}
}

func TestImageProviderEvictRespectsProvidedCache(t *testing.T) {
	tester := newProviderTester(t)
	otherCache := NewImageCache()
	bytes := append([]byte{}, kTransparentImage...)
	imageProvider := NewMemoryImage(bytes)
	cacheStream := otherCache.PutIfAbsent(imageProvider, func() ImageStreamCompleter {
		return imageProvider.LoadBuffer(imageProvider, basicDecoder)
	})
	stream := imageProvider.Resolve(ImageConfigurationEmpty)
	completer := async.NewCompleter[async.Void]()
	cacheCompleter := async.NewCompleter[async.Void]()
	stream.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { completer.Complete(async.Void{}) }))
	cacheStream.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { cacheCompleter.Complete(async.Void{}) }))
	await(tester, completer.Future())
	await(tester, cacheCompleter.Future())
	if otherCache.CurrentSize() != 1 || ImageCacheInstance().CurrentSize() != 1 {
		t.Fatal("sizes")
	}
	if !await(tester, imageProvider.Evict(ImageProviderEvictOptions{Cache: otherCache})) {
		t.Error("evict false")
	}
	if otherCache.CurrentSize() != 0 || ImageCacheInstance().CurrentSize() != 1 {
		t.Error("sizes after evict")
	}
}

func TestImageProviderErrorsCanAlwaysBeCaught(t *testing.T) {
	tester := newProviderTester(t)
	imageProvider := NewErrorImageProvider()
	caughtError := async.NewCompleter[bool]()
	foundation.FlutterErrorOnError = func(*foundation.FlutterErrorDetails) { caughtError.Complete(false) }
	stream := imageProvider.Resolve(ImageConfigurationEmpty)
	stream.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { caughtError.Complete(false) },
		ImageStreamListenerOptions{OnError: func(any, dart.StackTrace) { caughtError.Complete(true) }}))
	if !await(tester, caughtError.Future()) {
		t.Error("not caught by the listener")
	}
}

func TestDidHaveMemoryPressureClearsImageCache(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester)
	ImageCacheInstance().PutIfAbsent(1, func() ImageStreamCompleter {
		return NewOneFrameImageStreamCompleter(async.FutureValue(NewImageInfo(image)))
	})
	tester.Loop().Elapse(0)
	if ImageCacheInstance().CurrentSize() != 1 {
		t.Fatal("not cached")
	}
	message := services.JSONMessageCodec{}.EncodeMessage(map[string]any{"type": "memoryPressure"})
	await(tester, tester.Binding().DefaultBinaryMessenger().HandlePlatformMessage("flutter/system", message, func([]byte) {}))
	if ImageCacheInstance().CurrentSize() != 0 {
		t.Error("not cleared")
	}
}

// fakeImageCache counts clears.
type fakeImageCache struct {
	ImageCache
	clearCount, liveClearCount int
}

func (c *fakeImageCache) Clear() {
	c.clearCount++
	c.ImageCache.Clear()
}

func (c *fakeImageCache) ClearLiveImages() {
	c.liveClearCount++
	c.ImageCache.ClearLiveImages()
}

type evictTestBinding struct {
	paintingTestBinding
	imageCache *fakeImageCache
}

func (b *evictTestBinding) CreateImageCache() *ImageCache {
	b.imageCache = &fakeImageCache{}
	b.imageCache.Init(b.imageCache)
	return &b.imageCache.ImageCache
}

func TestEvictClearsLiveReferences(t *testing.T) {
	if !isolated(t) {
		return
	}
	defer async.SetCurrent(async.NewLoop())()
	b := &evictTestBinding{}
	newPaintingBinding(b, &b.paintingTestBinding)
	if b.imageCache.clearCount != 0 || b.imageCache.liveClearCount != 0 {
		t.Fatal("cleared")
	}
	b.Evict("/path/to/asset.png")
	if b.imageCache.clearCount != 1 || b.imageCache.liveClearCount != 1 {
		t.Errorf("clear %d, live %d", b.imageCache.clearCount, b.imageCache.liveClearCount)
	}
}

func TestShaderWarmUpIsNullByDefault(t *testing.T) {
	if PaintingBindingShaderWarmUp != nil {
		t.Error("not null")
	}
}
