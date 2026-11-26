package painting_test

// Port of packages/flutter/test/painting/image_provider_test.dart,
// image_info_test.dart, image_decoder_test.dart (with painting_utils.dart's
// PaintingBindingSpy) and shader_warm_up_test.dart (Flutter 3.50.1): all
// 15 tests. package:file's MemoryFileSystem is memoryFile (an io.File
// that is not dart:io's, so FileImage reads it with ReadAsBytes);
// shader_warm_up_test's WidgetsFlutterBinding.ensureInitialized() is a
// painting binding built in the test's own process (WidgetsBinding is not
// ported; the warm-up runs in PaintingBinding.initInstances).

import (
	"regexp"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/io"
)

// memoryFile is a package:file MemoryFileSystem file.
type memoryFile struct {
	path string
	data []byte
}

func (f *memoryFile) Path() string                      { return f.path }
func (f *memoryFile) Length() async.Future[int]         { return async.FutureValue(len(f.data)) }
func (f *memoryFile) ReadAsBytes() async.Future[[]byte] { return async.FutureValue(f.data) }
func (f *memoryFile) String() string                    { return "MemoryFile: '" + f.path + "'" }

var _ io.File = (*memoryFile)(nil)

// newProviderTester sets FlutterError.onError back and clears the cache
// after the test (image_provider_test's setUp/tearDown).
func newProviderTester(t *testing.T) *flutter_test.Tester {
	tester := flutter_test.New(t)
	old := foundation.FlutterErrorOnError
	t.Cleanup(func() {
		foundation.FlutterErrorOnError = old
		ImageCacheInstance().Clear()
		ImageCacheInstance().ClearLiveImages()
	})
	return tester
}

func TestImageProviderObtainKeyErrorsWillBeCaught(t *testing.T) {
	tester := newProviderTester(t)
	imageProvider := NewObtainKeyErrorImageProvider()
	caughtError := async.NewCompleter[bool]()
	foundation.FlutterErrorOnError = func(*foundation.FlutterErrorDetails) { caughtError.Complete(false) }
	stream := imageProvider.Resolve(ImageConfigurationEmpty)
	stream.AddListener(NewImageStreamListener(func(*ImageInfo, bool) { caughtError.Complete(false) },
		ImageStreamListenerOptions{OnError: func(any, dart.StackTrace) { caughtError.Complete(true) }}))
	if !await(tester, caughtError.Future()) {
		t.Error("not caught by the listener")
	}
}

func TestImageProviderObtainKeyErrorsWillBeCaughtCheckLocation(t *testing.T) {
	tester := newProviderTester(t)
	imageProvider := NewObtainKeyErrorImageProvider()
	caughtError := async.NewCompleter[bool]()
	foundation.FlutterErrorOnError = func(*foundation.FlutterErrorDetails) { caughtError.Complete(true) }
	await(tester, imageProvider.ObtainCacheStatus(ImageConfigurationEmpty))
	if !await(tester, caughtError.Future()) {
		t.Error("not caught")
	}
}

func TestFileImageWithEmptyFileThrowsAndEvicts(t *testing.T) {
	tester := newProviderTester(t)
	errC := async.NewCompleter[dart.StateError]()
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		errC.Complete(details.Exception.(dart.StateError))
	}
	provider := NewFileImage(&memoryFile{path: "/empty.png"})
	c := ImageCacheInstance()
	if !c.StatusForKey(provider).Untracked() || c.PendingImageCount() != 0 {
		t.Fatal("tracked")
	}
	provider.Resolve(ImageConfigurationEmpty)
	if !c.StatusForKey(provider).Pending || c.PendingImageCount() != 1 {
		t.Fatal("not pending")
	}
	e := await(tester, errC.Future())
	if e.Message != "MemoryFile: '/empty.png' is empty and cannot be loaded as an image." {
		t.Errorf("error %q", e.Message)
	}
	if !c.StatusForKey(provider).Untracked() || c.PendingImageCount() != 0 {
		t.Error("still tracked")
	}
}

func TestFileImageWithEmptyFileThrowsLoad(t *testing.T) {
	tester := newProviderTester(t)
	errC := async.NewCompleter[dart.StateError]()
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		errC.Complete(details.Exception.(dart.StateError))
	}
	provider := NewFileImage(&memoryFile{path: "/empty.png"})
	completer := provider.LoadBuffer(provider, func(*ui.ImmutableBuffer, ...InstantiateImageCodecFromBufferOptions) async.Future[ui.Codec] {
		return async.FutureValue(createNoOpCodec())
	})
	if _, ok := completer.(*MultiFrameImageStreamCompleter); !ok {
		t.Errorf("%T", completer)
	}
	await(tester, errC.Future())
}

func TestFileImageSetsTag(t *testing.T) {
	newProviderTester(t)
	file := &memoryFile{path: "/blue.png", data: kBlueSquarePng}
	provider := NewFileImage(file)
	completer := provider.LoadBuffer(provider, noOpDecoderBufferCallback).(*MultiFrameImageStreamCompleter)
	if l := completer.DebugLabel(); l == nil || *l != file.Path() {
		t.Errorf("label %v", l)
	}
}

func TestMemoryImageSetsTag(t *testing.T) {
	newProviderTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	provider := NewMemoryImage(bytes)
	completer := provider.LoadBuffer(provider, noOpDecoderBufferCallback).(*MultiFrameImageStreamCompleter)
	want := "MemoryImage(Uint8List#" + foundation.ShortHash(&bytes[0]) + ")"
	if l := completer.DebugLabel(); l == nil || *l != want {
		t.Errorf("label %v, want %s", *l, want)
	}
}

// testAssetBundle is image_provider_test's _TestAssetBundle.
type testAssetBundle struct {
	services.CachingAssetBundle
	data []byte
}

func newTestAssetBundle(data []byte) *testAssetBundle {
	b := &testAssetBundle{data: data}
	b.SetSelf(b)
	return b
}

func (b *testAssetBundle) Load(key string) async.Future[[]byte] {
	return async.Async(func() []byte { return append([]byte{}, b.data...) })
}

func TestAssetImageSetsTag(t *testing.T) {
	tester := newProviderTester(t)
	const asset = "images/blue.png"
	provider := NewExactAssetImage(asset, ExactAssetImageOptions{Bundle: newTestAssetBundle(kBlueSquarePng)})
	key := await(tester, provider.ObtainKey(ImageConfigurationEmpty))
	completer := provider.LoadBuffer(key, noOpDecoderBufferCallback).(*MultiFrameImageStreamCompleter)
	if l := completer.DebugLabel(); l == nil || *l != asset {
		t.Errorf("label %v", l)
	}
}

func TestResizeImageSetsTag(t *testing.T) {
	tester := newProviderTester(t)
	bytes := append([]byte{}, kBlueSquarePng...)
	w, h := 40, 40
	provider := NewResizeImage(NewMemoryImage(bytes), ResizeImageOptions{Width: &w, Height: &h})
	completer := provider.LoadBuffer(await(tester, provider.ObtainKey(ImageConfigurationEmpty)), noOpDecoderBufferCallback).(*MultiFrameImageStreamCompleter)
	want := "MemoryImage(Uint8List#" + foundation.ShortHash(&bytes[0]) + ") - Resized(40×40)"
	if l := completer.DebugLabel(); l == nil || *l != want {
		t.Errorf("label %q, want %q", *l, want)
	}
}

func TestFileImageThrowsForNonImageFile(t *testing.T) {
	tester := newProviderTester(t)
	errC := async.NewCompleter[error]()
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		errC.Complete(details.Exception.(error))
	}
	provider := NewFileImage(io.NewFile("image_provider.go"))
	c := ImageCacheInstance()
	if !c.StatusForKey(provider).Untracked() || c.PendingImageCount() != 0 {
		t.Fatal("tracked")
	}
	provider.Resolve(ImageConfigurationEmpty)
	if !c.StatusForKey(provider).Pending || c.PendingImageCount() != 1 {
		t.Fatal("not pending")
	}
	if e := await(tester, errC.Future()); !strings.Contains(e.Error(), "Invalid image data") {
		t.Errorf("error %v", e)
	}
	// Invalid images are marked as pending so that we do not attempt to
	// reload them.
	if c.StatusForKey(provider).Untracked() || c.PendingImageCount() != 1 {
		t.Error("untracked")
	}
}

func TestImageProviderToStrings(t *testing.T) {
	newProviderTester(t)
	scale := 1.21
	if s := NewNetworkImage("test", NetworkImageOptions{Scale: &scale}).String(); s !=
		`NetworkImage("test", scale: 1.2, webHtmlElementStrategy: never, headers: null)` {
		t.Error(s)
	}
	if s := NewExactAssetImage("test", ExactAssetImageOptions{Scale: &scale}).String(); s !=
		`ExactAssetImage(name: "test", scale: 1.2, bundle: null)` {
		t.Error(s)
	}
	s := NewMemoryImage(make([]byte, 0), MemoryImageOptions{Scale: &scale}).String()
	if regexp.MustCompile(`#[0-9a-f]{5}`).ReplaceAllString(s, "#00000") != "MemoryImage(Uint8List#00000, scale: 1.2)" {
		t.Error(s)
	}
}

func TestImageInfoSizeBytes(t *testing.T) {
	tester := newImageTester(t)
	for _, c := range []struct{ w, h, want int }{{10, 20, 800}, {20, 100, 8000}, {100, 200, 80000}} {
		image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: c.w, Height: c.h})
		if n := NewImageInfo(image).SizeBytes(); n != c.want {
			t.Errorf("%dx%d: %d", c.w, c.h, n)
		}
	}
}

// paintingBindingSpy is painting_utils.dart's PaintingBindingSpy.
type paintingBindingSpy struct {
	paintingTestBinding
	counter int
}

func (b *paintingBindingSpy) InstantiateImageCodecWithSize(buffer *ui.ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[ui.Codec] {
	b.counter++
	var o InstantiateImageCodecWithSizeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return ui.InstantiateImageCodecWithSize(buffer, ui.InstantiateImageCodecWithSizeOptions{GetTargetSize: o.GetTargetSize})
}

func newPaintingBinding[B interface {
	foundation.BindingBase
	PaintingBinding
}](b B, base *paintingTestBinding) B {
	base.SchedulerBindingMixin.SetSelf(b)
	base.ServicesBindingMixin.SetSelf(b)
	base.PaintingBindingMixin.SetSelf(b)
	base.Construct(b, &base.SchedulerBindingMixin, &base.ServicesBindingMixin, &base.PaintingBindingMixin)
	return b
}

func TestImageDecoderControlTest(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	b := &paintingBindingSpy{}
	newPaintingBinding(b, &b.paintingTestBinding)
	if b.counter != 0 {
		t.Fatal("called")
	}
	image := awaitLoop(loop, DecodeImageFromList(append([]byte{}, kTransparentImage...)))
	if image == nil || image.Width != 1 || image.Height != 1 || b.counter != 1 {
		t.Errorf("image %v, calls %d", image, b.counter)
	}
}

func awaitLoop[T any](loop *async.Loop, f async.Future[T]) T {
	var v T
	var err error
	done := false
	async.OnComplete(f, func(x T) { v, done = x, true }, func(e error) { err, done = e, true })
	for !done {
		loop.FlushMicrotasks()
		if !done && loop.RunPosted() == 0 {
			if d, ok := loop.NextDeadline(); ok {
				loop.ElapseTo(d)
			} else {
				loop.WaitPosted()
			}
		}
	}
	if err != nil {
		panic(err)
	}
	return v
}

// fakeShaderWarmUp is shader_warm_up_test's FakeShaderWarmUp.
type fakeShaderWarmUp struct {
	ShaderWarmUpBase
	ranWarmUp bool
}

func (w *fakeShaderWarmUp) WarmUpOnCanvas(canvas ui.Canvas) async.Future[async.Void] {
	w.ranWarmUp = true
	return async.FutureDelayed(0, func() async.Void { return async.Void{} })
}

func TestShaderWarmUp(t *testing.T) {
	if !isolated(t) {
		return
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	shaderWarmUp := &fakeShaderWarmUp{}
	shaderWarmUp.SetSelf(shaderWarmUp)
	PaintingBindingShaderWarmUp = shaderWarmUp
	captured := 0
	DebugCaptureShaderWarmUpImage = func(*ui.Image) bool { captured++; return true }
	pb := &paintingTestBinding{}
	newPaintingBinding(pb, pb)
	if !shaderWarmUp.ranWarmUp {
		t.Error("warm-up did not run")
	}
	// expectAsync1: the capture callback is called once, when the picture
	// has been rasterized.
	for i := 0; captured == 0 && i < 1000; i++ {
		loop.FlushMicrotasks()
		if loop.RunPosted() == 0 {
			if d, ok := loop.NextDeadline(); ok {
				loop.ElapseTo(d)
			} else {
				loop.WaitPosted()
			}
		}
	}
	if captured != 1 {
		t.Errorf("captured %d", captured)
	}
}

// twiceErrorProvider reports an error twice from ResolveStreamForKey.
type twiceErrorProvider struct{ ImageProviderBase }

func (p *twiceErrorProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *twiceErrorProvider) ResolveStreamForKey(_ ImageConfiguration, _ *ImageStream, _ any, handleError ImageErrorListener) {
	handleError("first", nil)
	handleError("second", nil)
}

// Not in Flutter's tests (found by mutation testing): resolve reports only
// the first error of a provider.
func TestImageProviderReportsOnlyFirstError(t *testing.T) {
	tester := newProviderTester(t)
	p := &twiceErrorProvider{}
	p.SetSelf(p)
	var errs []any
	p.Resolve(ImageConfigurationEmpty).AddListener(NewImageStreamListener(func(*ImageInfo, bool) {},
		ImageStreamListenerOptions{OnError: func(e any, _ dart.StackTrace) { errs = append(errs, e) }}))
	tester.Loop().Elapse(0)
	if len(errs) != 1 || errs[0] != "first" {
		t.Errorf("errors %v", errs)
	}
}
