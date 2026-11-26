package painting_test

// Port of packages/flutter/test/painting/image_cache_test.dart (Flutter
// 3.50.1): all 25 tests. `await SchedulerBinding.instance.endOfFrame`
// after scheduleFrame is tester.Pump() (the test binding pumps frames
// explicitly); `await null` is a microtask flush.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

func resolveValue(tester *flutter_test.Tester, key, value int) int {
	info := await(tester, extractOneFrame(NewTestImageProvider(key, value, createTestImage(tester)).Resolve(ImageConfigurationEmpty)))
	return info.Self().(*TestImageInfo).Value
}

func TestImageCacheMaintainsCacheSize(t *testing.T) {
	tester := newImageTester(t)
	ImageCacheInstance().SetMaximumSize(3)
	steps := []struct{ key, value, want int }{
		{1, 1, 1}, {1, 2, 1}, {1, 3, 1}, {1, 4, 1}, {1, 5, 1},
		// cache still only has one entry in it: 1(1)
		{2, 7, 7},   // 1(1), 2(7)
		{1, 8, 1},   // 2(7), 1(1)
		{3, 9, 9},   // 2(7), 1(1), 3(9)
		{1, 10, 1},  // 2(7), 3(9), 1(1)
		{4, 11, 11}, // 3(9), 1(1), 4(11)
		{1, 12, 1},  // 3(9), 4(11), 1(1)
		{2, 13, 13}, // 4(11), 1(1), 2(13)
		{3, 14, 14}, // 1(1), 2(13), 3(14)
		{4, 15, 15}, // 2(13), 3(14), 4(15)
		{1, 16, 16}, // 3(14), 4(15), 1(16)
	}
	a := await(tester, extractOneFrame(NewTestImageProvider(1, 1, createTestImage(tester)).Resolve(ImageConfigurationEmpty)))
	f := await(tester, extractOneFrame(NewTestImageProvider(1, 6, createTestImage(tester)).Resolve(ImageConfigurationEmpty)))
	if !f.Self().Equal(a) {
		t.Errorf("f != a: %v %v", f.Self(), a.Self())
	}
	for _, s := range steps {
		if got := resolveValue(tester, s.key, s.value); got != s.want {
			t.Errorf("TestImageProvider(%d, %d): value %d, want %d", s.key, s.value, got, s.want)
		}
	}
}

func TestImageCacheClearRemovesAllImagesAndResetsCacheSize(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	c := ImageCacheInstance()
	if c.CurrentSize() != 0 || c.CurrentSizeBytes() != 0 {
		t.Fatal("not empty")
	}
	await(tester, extractOneFrame(NewTestImageProvider(1, 1, testImage).Resolve(ImageConfigurationEmpty)))
	await(tester, extractOneFrame(NewTestImageProvider(2, 2, testImage).Resolve(ImageConfigurationEmpty)))
	if c.CurrentSize() != 2 || c.CurrentSizeBytes() != 256*2 {
		t.Errorf("size %d, bytes %d", c.CurrentSize(), c.CurrentSizeBytes())
	}
	c.Clear()
	if c.CurrentSize() != 0 || c.CurrentSizeBytes() != 0 {
		t.Errorf("size %d, bytes %d", c.CurrentSize(), c.CurrentSizeBytes())
	}
}

func TestImageCacheEvictsIndividualImages(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	c := ImageCacheInstance()
	await(tester, extractOneFrame(NewTestImageProvider(1, 1, testImage).Resolve(ImageConfigurationEmpty)))
	await(tester, extractOneFrame(NewTestImageProvider(2, 2, testImage).Resolve(ImageConfigurationEmpty)))
	if c.CurrentSize() != 2 || c.CurrentSizeBytes() != 512 {
		t.Errorf("size %d, bytes %d", c.CurrentSize(), c.CurrentSizeBytes())
	}
	if !c.Evict(1) {
		t.Error("evict(1) false")
	}
	if c.CurrentSize() != 1 || c.CurrentSizeBytes() != 256 {
		t.Errorf("size %d, bytes %d", c.CurrentSize(), c.CurrentSizeBytes())
	}
}

func TestImageCacheDoNotCacheLargeImages(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	c := ImageCacheInstance()
	c.SetMaximumSizeBytes(1)
	await(tester, extractOneFrame(NewTestImageProvider(1, 1, testImage).Resolve(ImageConfigurationEmpty)))
	if c.CurrentSize() != 0 || c.CurrentSizeBytes() != 0 || c.MaximumSizeBytes() != 1 {
		t.Errorf("size %d, bytes %d, max %d", c.CurrentSize(), c.CurrentSizeBytes(), c.MaximumSizeBytes())
	}
}

func TestImageCacheReturnsNullIfAnErrorIsCaught(t *testing.T) {
	newImageTester(t)
	basicDecoder := func(bytes *ui.ImmutableBuffer, opts ...InstantiateImageCodecFromBufferOptions) asyncCodec {
		return PaintingBindingInstance().InstantiateImageCodecFromBuffer(bytes, opts...)
	}
	errorImage := NewErrorImageProvider()
	r := mustPanic(t, func() {
		ImageCacheInstance().PutIfAbsent(errorImage, func() ImageStreamCompleter { return errorImage.LoadBuffer(errorImage, basicDecoder) })
	})
	if _, ok := r.(dartError); !ok {
		t.Errorf("threw %v", r)
	}
	caughtError := false
	result := ImageCacheInstance().PutIfAbsent(errorImage, func() ImageStreamCompleter { return errorImage.LoadBuffer(errorImage, basicDecoder) },
		ImageCachePutIfAbsentOptions{OnError: func(any, dartStack) { caughtError = true }})
	if result != nil || !caughtError {
		t.Errorf("result %v, caught %v", result, caughtError)
	}
}

func TestImageCacheAlreadyPendingImageIsReturned(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1, completer2 := NewTestImageStreamCompleter(), NewTestImageStreamCompleter()
	r1 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	r2 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer2 })
	if r1 != completer1 || r2 != completer1 {
		t.Error("not the pending completer")
	}
}

func status(key any) ImageCacheStatus { return ImageCacheInstance().StatusForKey(key) }

func TestImageCachePendingImageRemovedWhenCacheCleared(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1, completer2 := NewTestImageStreamCompleter(), NewTestImageStreamCompleter()
	r1 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	// Make the image seem live.
	completer1.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}))
	if s := status(testImage); !s.Pending || !s.Live {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().Clear()
	if s := status(testImage); s.Pending || !s.Live {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().ClearLiveImages()
	if s := status(testImage); s.Pending || s.Live {
		t.Errorf("%v", s)
	}
	r2 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer2 })
	if r1 != completer1 || r2 != completer2 {
		t.Error("wrong completers")
	}
}

func TestImageCachePendingImageRemovedWhenEvicted(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1, completer2 := NewTestImageStreamCompleter(), NewTestImageStreamCompleter()
	r1 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	ImageCacheInstance().Evict(testImage)
	r2 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer2 })
	if r1 != completer1 || r2 != completer2 {
		t.Error("wrong completers")
	}
}

func TestImageCacheFailedImageCanBeRemovedFromPending(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	called := false
	NewFailingTestImageProvider(1, 1, testImage).Resolve(ImageConfigurationEmpty).AddListener(NewImageStreamListener(
		func(*ImageInfo, bool) { t.Error("Image should not complete successfully") },
		ImageStreamListenerOptions{OnError: func(exception any, _ dartStack) {
			called = true
			if !ImageCacheInstance().Evict(1) {
				t.Error("eviction failed")
			}
		}}))
	// yield an event turn so that async work can complete.
	tester.Loop().FlushMicrotasks()
	if !called {
		t.Error("onError not called")
	}
}

func TestImageCacheContainsKeyPending(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	r1 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if r1 != completer1 || !ImageCacheInstance().ContainsKey(testImage) {
		t.Error("containsKey")
	}
}

func TestImageCacheContainsKeyCompleted(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	r1 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	// Mark as complete
	completer1.TestSetImage(testImage)
	if r1 != completer1 || !ImageCacheInstance().ContainsKey(testImage) {
		t.Error("containsKey")
	}
}

func TestImageCachePutIfAbsentUpdatesLRUOfLiveImage(t *testing.T) {
	tester := newImageTester(t)
	ImageCacheInstance().SetMaximumSize(1)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	testImage2 := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10})
	completer1 := NewTestImageStreamCompleter()
	completer1.TestSetImage(testImage)
	completer2 := NewTestImageStreamCompleter()
	completer2.TestSetImage(testImage2)
	completer1.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}))
	r1 := ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{KeepAlive: true, Live: true}) {
		t.Errorf("%v", s)
	}
	if !status(testImage2).Untracked() {
		t.Error("testImage2 tracked")
	}
	r2 := ImageCacheInstance().PutIfAbsent(testImage2, func() ImageStreamCompleter { return completer2 })
	if s := status(testImage); s != (ImageCacheStatus{Live: true}) { // evicted
		t.Errorf("%v", s)
	}
	if s := status(testImage2); s != (ImageCacheStatus{KeepAlive: true}) { // took the LRU spot, no listeners
		t.Errorf("%v", s)
	}
	if r1 != completer1 || r2 != completer2 {
		t.Error("wrong completers")
	}
}

func TestImageCacheLiveImageCacheAvoidsLeaks(t *testing.T) {
	tester := newImageTester(t)
	ImageCacheInstance().SetMaximumSize(3)
	for i := 1; i <= 6; i++ {
		NewTestImageProvider(i, i, createTestImage(tester)).Resolve(ImageConfigurationEmpty)
	}
	// wait an event loop to let image resolution process.
	tester.Loop().FlushMicrotasks()
	if ImageCacheInstance().CurrentSize() != 3 || ImageCacheInstance().LiveImageCount() != 0 {
		t.Errorf("size %d, live %d", ImageCacheInstance().CurrentSize(), ImageCacheInstance().LiveImageCount())
	}
}

func TestImageCacheDisabledDoesNotLeakLiveImages(t *testing.T) {
	tester := newImageTester(t)
	ImageCacheInstance().SetMaximumSize(0)
	for i := 1; i <= 6; i++ {
		NewTestImageProvider(i, i, createTestImage(tester)).Resolve(ImageConfigurationEmpty)
	}
	tester.Loop().FlushMicrotasks()
	if ImageCacheInstance().CurrentSize() != 0 || ImageCacheInstance().LiveImageCount() != 0 {
		t.Errorf("size %d, live %d", ImageCacheInstance().CurrentSize(), ImageCacheInstance().LiveImageCount())
	}
}

func TestImageCacheClearingDoesNotLeakLiveImages(t *testing.T) {
	tester := newImageTester(t)
	ImageCacheInstance().SetMaximumSize(1)
	testImage1 := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	testImage2 := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10})
	completer1 := NewTestImageStreamCompleter()
	completer2 := NewTestImageStreamCompleter()
	completer2.TestSetImage(testImage2)
	ImageCacheInstance().PutIfAbsent(testImage1, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage1); !s.Pending || !s.Live {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().Clear()
	if s := status(testImage1); s.Pending || s.Live {
		t.Errorf("%v", s)
	}
	completer1.TestSetImage(testImage1)
	if s := status(testImage1); s.KeepAlive || s.Live {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().PutIfAbsent(testImage2, func() ImageStreamCompleter { return completer2 })
	if status(testImage1).Tracked() || !status(testImage2).Tracked() {
		t.Error("tracking")
	}
}

func TestImageCacheEvictingPendingClearsLiveByDefault(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{Pending: true, Live: true}) {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().Evict(testImage)
	if !status(testImage).Untracked() {
		t.Error("tracked")
	}
}

var noLive = false

func TestImageCacheEvictingPendingIncludeLiveFalseOnlyCacheListening(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{Pending: true, Live: true}) {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().Evict(testImage, ImageCacheEvictOptions{IncludeLive: &noLive})
	if s := status(testImage); s != (ImageCacheStatus{}) {
		t.Errorf("%v", s)
	}
}

func TestImageCacheEvictingPendingIncludeLiveFalseOtherListener(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	completer1.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}))
	ImageCacheInstance().Evict(testImage, ImageCacheEvictOptions{IncludeLive: &noLive})
	if s := status(testImage); s != (ImageCacheStatus{Live: true}) {
		t.Errorf("%v", s)
	}
}

func TestImageCacheEvictingCompletedClearsLiveByDefault(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	completer1.TestSetImage(testImage)
	completer1.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}))
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{Live: true, KeepAlive: true}) {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().Evict(testImage)
	if !status(testImage).Untracked() {
		t.Error("tracked")
	}
}

func TestImageCacheEvictingCompletedIncludeLiveFalse(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	completer1.TestSetImage(testImage)
	completer1.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}))
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	ImageCacheInstance().Evict(testImage, ImageCacheEvictOptions{IncludeLive: &noLive})
	if s := status(testImage); s != (ImageCacheStatus{Live: true}) {
		t.Errorf("%v", s)
	}
}

func TestImageCacheClearingLiveImagesRemovesCallbacks(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	listener := NewImageStreamListener(func(*ImageInfo, bool) {})
	completer1 := NewTestImageStreamCompleter()
	completer1.TestSetImage(testImage)
	completer1.AddListener(listener)
	completer2 := NewTestImageStreamCompleter()
	completer2.TestSetImage(testImage)
	completer2.AddListener(listener)
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{Live: true, KeepAlive: true}) {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().Clear()
	ImageCacheInstance().ClearLiveImages()
	if s := status(testImage); s != (ImageCacheStatus{}) {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer2 })
	if s := status(testImage); s != (ImageCacheStatus{Live: true, KeepAlive: true}) {
		t.Errorf("%v", s)
	}
	completer1.RemoveListener(listener)
	if s := status(testImage); s != (ImageCacheStatus{Live: true, KeepAlive: true}) {
		t.Errorf("%v", s)
	}
}

func TestImageCacheLiveImageGetsSizeUpdated(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	const testImageSize = 8 * 8 * 4
	listener := NewImageStreamListener(func(*ImageInfo, bool) {})
	completer1 := NewTestImageStreamCompleter()
	completer1.AddListener(listener)
	c := ImageCacheInstance()
	c.PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{Pending: true, Live: true}) || c.CurrentSizeBytes() != 0 {
		t.Errorf("%v %d", s, c.CurrentSizeBytes())
	}
	completer1.TestSetImage(testImage)
	if s := status(testImage); s != (ImageCacheStatus{Live: true, KeepAlive: true}) || c.CurrentSizeBytes() != testImageSize {
		t.Errorf("%v %d", s, c.CurrentSizeBytes())
	}
	c.Evict(testImage, ImageCacheEvictOptions{IncludeLive: &noLive})
	if s := status(testImage); s != (ImageCacheStatus{Live: true}) || c.CurrentSizeBytes() != 0 {
		t.Errorf("%v %d", s, c.CurrentSizeBytes())
	}
	c.PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{Live: true, KeepAlive: true}) || c.CurrentSizeBytes() != testImageSize {
		t.Errorf("%v %d", s, c.CurrentSizeBytes())
	}
}

func endOfFrame(tester *flutter_test.Tester) {
	scheduler.SchedulerBindingInstance().ScheduleFrame()
	tester.Pump()
}

func TestImageCacheImageObtainedAndDisposedProperly(t *testing.T) {
	tester := newImageTester(t)
	const key = 1
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8, NoCache: true})
	handles := func() int { return len(testImage.DebugGetOpenHandleStackTraces()) }
	if handles() != 1 {
		t.Fatalf("handles %d", handles())
	}
	var imageInfo *ImageInfo
	listener := NewImageStreamListener(func(info *ImageInfo, syncCall bool) { imageInfo = info })
	completer := NewTestImageStreamCompleter()
	completer.AddListener(listener)
	ImageCacheInstance().PutIfAbsent(key, func() ImageStreamCompleter { return completer })
	if handles() != 1 {
		t.Errorf("handles %d", handles())
	}
	// This should cause keepAlive to be set to true.
	completer.TestSetImage(testImage)
	if imageInfo == nil || handles() != 2 { // +1 ImageStreamCompleter
		t.Errorf("handles %d", handles())
	}
	completer.RemoveListener(listener)
	endOfFrame(tester)
	if handles() != 2 {
		t.Errorf("handles %d", handles())
	}
	if !ImageCacheInstance().Evict(key) {
		t.Error("evict")
	}
	endOfFrame(tester)
	// -1 _CachedImage, -1 ImageStreamCompleter
	if handles() != 1 {
		t.Errorf("handles %d", handles())
	}
	imageInfo.Dispose()
	if handles() != 0 {
		t.Errorf("handles %d", handles())
	}
}

func TestImageCacheImageDisposedProperlyWhenListenerStillActive(t *testing.T) {
	tester := newImageTester(t)
	const key = 1
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8, NoCache: true})
	handles := func() int { return len(testImage.DebugGetOpenHandleStackTraces()) }
	var imageInfo *ImageInfo
	listener := NewImageStreamListener(func(info *ImageInfo, syncCall bool) { imageInfo = info })
	completer := NewTestImageStreamCompleter()
	completer.AddListener(listener)
	ImageCacheInstance().PutIfAbsent(key, func() ImageStreamCompleter { return completer })
	completer.TestSetImage(testImage)
	if imageInfo == nil || handles() != 2 {
		t.Errorf("handles %d", handles())
	}
	if !ImageCacheInstance().Evict(key) {
		t.Error("evict")
	}
	endOfFrame(tester)
	// Live image still around since there's still a listener, and the
	// listener should be holding a handle.
	if handles() != 2 {
		t.Errorf("handles %d", handles())
	}
	completer.RemoveListener(listener)
	if handles() != 1 {
		t.Errorf("handles %d", handles())
	}
	imageInfo.Dispose()
	if handles() != 0 {
		t.Errorf("handles %d", handles())
	}
}

func TestImageCacheClearDoesNotLeavePendingImagesStuck(t *testing.T) {
	tester := newImageTester(t)
	testImage := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 8, Height: 8})
	completer1 := NewTestImageStreamCompleter()
	ImageCacheInstance().PutIfAbsent(testImage, func() ImageStreamCompleter { return completer1 })
	if s := status(testImage); s != (ImageCacheStatus{Pending: true, Live: true}) {
		t.Errorf("%v", s)
	}
	ImageCacheInstance().Clear()
	// No one else is listening to the completer. It should not be
	// considered live anymore.
	if s := status(testImage); s != (ImageCacheStatus{}) {
		t.Errorf("%v", s)
	}
}
