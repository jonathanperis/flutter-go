package painting_test

// Port of packages/flutter/test/painting/mocks_for_image_cache.dart,
// fake_codec.dart, fake_image_provider.dart, image_test_utils.dart and
// no_op_codec.dart (Flutter 3.50.1), the helpers of the image tests, plus
// the Go test harness (newImageTester: flutter_test's binding with the
// image cache reset after each test, as the Dart files' tearDown do).

import (
	"fmt"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// newImageTester starts a test in flutter_test's binding (Dart:
// TestRenderingFlutterBinding.ensureInitialized()) and resets the image
// cache when it ends (the image cache tests' tearDown).
func newImageTester(t *testing.T) *flutter_test.Tester {
	tester := flutter_test.New(t)
	t.Cleanup(func() {
		c := ImageCacheInstance()
		c.Clear()
		c.ClearLiveImages()
		c.SetMaximumSize(1000)
		c.SetMaximumSizeBytes(10485760)
	})
	return tester
}

func await[T any](tester *flutter_test.Tester, f async.Future[T]) T {
	return flutter_test.AwaitFuture(tester, f)
}

func createTestImage(tester *flutter_test.Tester, opts ...flutter_test.CreateTestImageOptions) *ui.Image {
	return await(tester, flutter_test.CreateTestImage(opts...))
}

// dartError stands for an instance of Dart's Error.
type dartError struct{}

func (dartError) Error() string { return "Instance of 'Error'" }

// TestImageInfo is an ImageInfo with a value (mocks_for_image_cache.dart).
type TestImageInfo struct {
	ImageInfo
	Value int
}

func NewTestImageInfo(value int, image *ui.Image, opts ...ImageInfoOptions) *TestImageInfo {
	i := &TestImageInfo{Value: value}
	i.Init(i, image, opts...)
	return i
}

func (i *TestImageInfo) String() string {
	return fmt.Sprintf("%s(%d)", foundation.ObjectRuntimeType(i, "TestImageInfo"), i.Value)
}

func (i *TestImageInfo) Clone() *ImageInfo {
	scale := i.Scale
	return &NewTestImageInfo(i.Value, i.Image.Clone(), ImageInfoOptions{Scale: &scale, DebugLabel: i.DebugLabel}).ImageInfo
}

func (i *TestImageInfo) Equal(other *ImageInfo) bool {
	o, ok := other.Self().(*TestImageInfo)
	return ok && o.Value == i.Value && o.Image.IsCloneOf(i.Image) && o.Scale == i.Scale &&
		((o.DebugLabel == nil) == (i.DebugLabel == nil)) && (o.DebugLabel == nil || *o.DebugLabel == *i.DebugLabel)
}

// TestImageProvider has an int key and loads a TestImageInfo.
type TestImageProvider struct {
	ImageProviderBase
	Key        int
	ImageValue int
	Image      *ui.Image
}

func NewTestImageProvider(key, imageValue int, image *ui.Image) *TestImageProvider {
	p := &TestImageProvider{Key: key, ImageValue: imageValue, Image: image}
	p.SetSelf(p)
	return p
}

func (p *TestImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return async.FutureValue[any](p.Key)
}

func (p *TestImageProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	return NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(&NewTestImageInfo(p.ImageValue, p.Image.Clone()).ImageInfo))
}

func (p *TestImageProvider) String() string {
	return fmt.Sprintf("%s(%d, %d)", foundation.ObjectRuntimeType(p, "TestImageProvider"), p.Key, p.ImageValue)
}

// FailingTestImageProvider's image fails to load with 'loading failed!'.
type FailingTestImageProvider struct{ TestImageProvider }

func NewFailingTestImageProvider(key, imageValue int, image *ui.Image) *FailingTestImageProvider {
	p := &FailingTestImageProvider{TestImageProvider{Key: key, ImageValue: imageValue, Image: image}}
	p.SetSelf(p)
	return p
}

func (p *FailingTestImageProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	return NewOneFrameImageStreamCompleter(async.FutureError[*ImageInfo](async.PanicError{Value: "loading failed!"}))
}

// extractOneFrame completes with the stream's first image.
func extractOneFrame(stream *ImageStream) async.Future[*ImageInfo] {
	completer := async.NewCompleter[*ImageInfo]()
	var listener ImageStreamListener
	listener = NewImageStreamListener(func(image *ImageInfo, synchronousCall bool) {
		completer.Complete(image)
		stream.RemoveListener(listener)
	})
	stream.AddListener(listener)
	return completer.Future()
}

// ErrorImageProvider throws from LoadImage and LoadBuffer.
type ErrorImageProvider struct{ ImageProviderBase }

func NewErrorImageProvider() *ErrorImageProvider {
	p := &ErrorImageProvider{}
	p.SetSelf(p)
	return p
}

func (p *ErrorImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	panic(dartError{})
}
func (p *ErrorImageProvider) LoadBuffer(any, DecoderBufferCallback) ImageStreamCompleter {
	panic(dartError{})
}
func (p *ErrorImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

// ObtainKeyErrorImageProvider throws from ObtainKey.
type ObtainKeyErrorImageProvider struct{ ImageProviderBase }

func NewObtainKeyErrorImageProvider() *ObtainKeyErrorImageProvider {
	p := &ObtainKeyErrorImageProvider{}
	p.SetSelf(p)
	return p
}

func (p *ObtainKeyErrorImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	panic(dartError{})
}
func (p *ObtainKeyErrorImageProvider) LoadBuffer(any, DecoderBufferCallback) ImageStreamCompleter {
	panic(dart.UnimplementedError{})
}
func (p *ObtainKeyErrorImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	panic(dartError{})
}

// LoadErrorImageProvider throws from LoadImage.
type LoadErrorImageProvider struct{ ImageProviderBase }

func NewLoadErrorImageProvider() *LoadErrorImageProvider {
	p := &LoadErrorImageProvider{}
	p.SetSelf(p)
	return p
}

func (p *LoadErrorImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	panic(dartError{})
}
func (p *LoadErrorImageProvider) LoadBuffer(any, DecoderBufferCallback) ImageStreamCompleter {
	panic(dart.UnimplementedError{})
}
func (p *LoadErrorImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

// LoadErrorCompleterImageProvider's completer fails synchronously.
type LoadErrorCompleterImageProvider struct{ ImageProviderBase }

func NewLoadErrorCompleterImageProvider() *LoadErrorCompleterImageProvider {
	p := &LoadErrorCompleterImageProvider{}
	p.SetSelf(p)
	return p
}

func (p *LoadErrorCompleterImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	completer := async.NewSyncCompleter[*ImageInfo]()
	completer.CompleteError(dartError{})
	return NewOneFrameImageStreamCompleter(completer.Future())
}
func (p *LoadErrorCompleterImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

// TestImageStreamCompleter exposes SetImage.
type TestImageStreamCompleter struct{ ImageStreamCompleterBase }

func NewTestImageStreamCompleter() *TestImageStreamCompleter {
	c := &TestImageStreamCompleter{}
	c.InitImageStreamCompleter(c)
	return c
}

func (c *TestImageStreamCompleter) TestSetImage(image *ui.Image) { c.SetImage(NewImageInfo(image)) }

// FakeCodec replays the frames of a real codec (fake_codec.dart).
type FakeCodec struct {
	frameCount, repetitionCount int
	frameInfos                  []*ui.FrameInfo
	nextFrame                   int
	NumFramesAsked              int
}

// FakeCodecFromData decodes data and keeps its frames.
func FakeCodecFromData(data []byte) async.Future[*FakeCodec] {
	return async.Async(func() *FakeCodec {
		codec := ui.InstantiateImageCodec(data).Await()
		frameCount := codec.FrameCount()
		var frameInfos []*ui.FrameInfo
		for i := 0; i < frameCount; i++ {
			frameInfos = append(frameInfos, codec.GetNextFrame().Await())
		}
		repetitionCount := codec.RepetitionCount()
		codec.Dispose()
		return &FakeCodec{frameCount: frameCount, repetitionCount: repetitionCount, frameInfos: frameInfos}
	})
}

func (c *FakeCodec) FrameCount() int      { return c.frameCount }
func (c *FakeCodec) RepetitionCount() int { return c.repetitionCount }
func (c *FakeCodec) GetNextFrame() async.Future[*ui.FrameInfo] {
	c.NumFramesAsked++
	result := async.FutureValue(c.frameInfos[c.nextFrame])
	c.nextFrame = (c.nextFrame + 1) % c.frameCount
	return result
}
func (c *FakeCodec) Dispose() {}

// FakeImageProvider loads a MultiFrameImageStreamCompleter over a codec.
type FakeImageProvider struct {
	ImageProviderBase
	codec ui.Codec
	Scale float64
}

func NewFakeImageProvider(codec ui.Codec, scale ...float64) *FakeImageProvider {
	p := &FakeImageProvider{codec: codec, Scale: 1.0}
	if len(scale) > 0 {
		p.Scale = scale[0]
	}
	p.SetSelf(p)
	return p
}

func (p *FakeImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *FakeImageProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	if key != p {
		panic(dart.NewAssertionError(nil))
	}
	return NewMultiFrameImageStreamCompleter(foundation.NewSynchronousFuture(p.codec), p.Scale)
}

// UtilsTestImageProvider is image_test_utils.dart's TestImageProvider: it
// completes with testImage when Complete is called.
type UtilsTestImageProvider struct {
	ImageProviderBase
	testImage     *ui.Image
	completer     *async.Completer[*ImageInfo]
	Configuration *ImageConfiguration
	LoadCallCount int
}

func NewUtilsTestImageProvider(testImage *ui.Image) *UtilsTestImageProvider {
	p := &UtilsTestImageProvider{testImage: testImage, completer: async.NewSyncCompleter[*ImageInfo]()}
	p.SetSelf(p)
	return p
}

func (p *UtilsTestImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *UtilsTestImageProvider) ResolveStreamForKey(config ImageConfiguration, stream *ImageStream, key any, handleError ImageErrorListener) {
	p.Configuration = &config
	p.ImageProviderBase.ResolveStreamForKey(config, stream, key, handleError)
}

func (p *UtilsTestImageProvider) LoadBuffer(any, DecoderBufferCallback) ImageStreamCompleter {
	panic(dart.UnsupportedError{Message: "Use ImageProvider.loadImage instead."})
}

func (p *UtilsTestImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	p.LoadCallCount++
	return NewOneFrameImageStreamCompleter(p.completer.Future())
}

func (p *UtilsTestImageProvider) Complete() *ImageInfo {
	imageInfo := NewImageInfo(p.testImage)
	p.completer.Complete(imageInfo)
	return imageInfo
}

func (p *UtilsTestImageProvider) String() string { return foundation.DescribeIdentity(p) + "()" }

// noOpCodec is no_op_codec.dart's _NoOpCodec.
type noOpCodec struct{}

func (noOpCodec) Dispose()             {}
func (noOpCodec) FrameCount() int      { panic(dart.UnimplementedError{}) }
func (noOpCodec) RepetitionCount() int { panic(dart.UnimplementedError{}) }
func (noOpCodec) GetNextFrame() async.Future[*ui.FrameInfo] {
	panic(dart.UnimplementedError{})
}

func createNoOpCodec() ui.Codec { return noOpCodec{} }

func noOpDecoderBufferCallback(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecFromBufferOptions) async.Future[ui.Codec] {
	return async.Async(func() ui.Codec { return noOpCodec{} })
}

// mustPanic returns what f panics with (failing the test if it doesn't).
func mustPanic(t *testing.T, f func()) (r any) {
	t.Helper()
	defer func() { r = recover() }()
	f()
	t.Fatal("expected a panic")
	return nil
}

type (
	asyncCodec = async.Future[ui.Codec]
	dartStack  = dart.StackTrace
)
