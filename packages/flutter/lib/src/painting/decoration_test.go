package painting_test

// Port of packages/flutter/test/painting/decoration_test.dart (Flutter
// 3.50.1): the tests of Decoration, DecorationImage, its painter and
// paintImage.
//
// box_decoration.dart is ported separately, so:
//   - Deferred to the BoxDecoration port: "Decoration.lerp()" (BoxDecoration
//     colors), "Decoration equality" (BoxDecoration ==), "BoxDecoration
//     backgroundImage clip", "BoxDecoration.lerp - shapes", "BoxDecoration.lerp
//     - gradients".
//   - Ported through the DecorationImage painter that BoxDecoration's painter
//     delegates to (BoxDecoration(image:).createBoxPainter().paint(canvas,
//     offset, configuration) paints the image with
//     createPainter().paint(canvas, offset & size, clipPath: null, ...) for a
//     rectangle): "BoxDecorationImageListenerSync/Async", "...does not change
//     when image is clone", "DecorationImage test", "DecorationImage with
//     null textDirection configuration should throw Error", "DecorationImage
//     scale test".
//   - "Decoration.lerp identical a,b" and "Decoration.lerp with unrelated
//     decorations" use FlutterLogoDecoration and a stand-in decoration
//     instead of BoxDecoration.
// TestCanvas (noSuchMethod) is flutter_test's TestRecordingCanvas.

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// SynchronousTestImageProvider loads a TestImageInfo synchronously.
type SynchronousTestImageProvider struct {
	ImageProviderBase
	image    *ui.Image
	cacheKey int
}

func newSynchronousTestImageProvider(image *ui.Image, cacheKey ...int) *SynchronousTestImageProvider {
	p := &SynchronousTestImageProvider{image: image, cacheKey: 1}
	if len(cacheKey) > 0 {
		p.cacheKey = cacheKey[0]
	}
	p.SetSelf(p)
	return p
}

func (p *SynchronousTestImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p.cacheKey)
}

func (p *SynchronousTestImageProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	return NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(&NewTestImageInfo(key.(int), p.image).ImageInfo))
}

func (p *SynchronousTestImageProvider) String() string { return "SynchronousTestImageProvider()" }

// SynchronousErrorTestImageProvider throws from ObtainKey and LoadImage.
type SynchronousErrorTestImageProvider struct {
	ImageProviderBase
	throwable any
}

func newSynchronousErrorTestImageProvider(throwable any) *SynchronousErrorTestImageProvider {
	p := &SynchronousErrorTestImageProvider{throwable: throwable}
	p.SetSelf(p)
	return p
}

func (p *SynchronousErrorTestImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	panic(p.throwable)
}

func (p *SynchronousErrorTestImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	panic(p.throwable)
}

// AsyncTestImageProvider's key arrives in a later microtask.
type AsyncTestImageProvider struct {
	ImageProviderBase
	image *ui.Image
}

func newAsyncTestImageProvider(image *ui.Image) *AsyncTestImageProvider {
	p := &AsyncTestImageProvider{image: image}
	p.SetSelf(p)
	return p
}

func (p *AsyncTestImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return async.FutureValue[any](2)
}

func (p *AsyncTestImageProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	return NewOneFrameImageStreamCompleter(async.FutureValue(&NewTestImageInfo(key.(int), p.image).ImageInfo))
}

// DelayedImageProvider's image arrives when Complete is called.
type DelayedImageProvider struct {
	ImageProviderBase
	image     *ui.Image
	completer *async.Completer[*ImageInfo]
}

func newDelayedImageProvider(image *ui.Image) *DelayedImageProvider {
	p := &DelayedImageProvider{image: image, completer: async.NewCompleter[*ImageInfo]()}
	p.SetSelf(p)
	return p
}

func (p *DelayedImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *DelayedImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	return NewOneFrameImageStreamCompleter(p.completer.Future())
}

func (p *DelayedImageProvider) Complete() { p.completer.Complete(NewImageInfo(p.image)) }

func (p *DelayedImageProvider) String() string { return foundation.DescribeIdentity(p) + "()" }

// MultiFrameImageProvider hands out a test completer.
type MultiFrameImageProvider struct {
	ImageProviderBase
	completer *TestImageStreamCompleter
}

func newMultiFrameImageProvider(c *TestImageStreamCompleter) *MultiFrameImageProvider {
	p := &MultiFrameImageProvider{completer: c}
	p.SetSelf(p)
	return p
}

func (p *MultiFrameImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *MultiFrameImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	return p.completer
}

func invocationsNamed(c *flutter_test.TestRecordingCanvas, name string) []flutter_test.Invocation {
	var out []flutter_test.Invocation
	for _, r := range c.Invocations {
		if r.Invocation.MemberName == name {
			out = append(out, r.Invocation)
		}
	}
	return out
}

func lastDrawnImage(t *testing.T, c *flutter_test.TestRecordingCanvas) *ui.Image {
	t.Helper()
	calls := invocationsNamed(c, "drawImageRect")
	if len(calls) == 0 {
		t.Fatal("no drawImageRect")
	}
	return calls[len(calls)-1].PositionalArguments[0].(*ui.Image)
}

func sizeConfig(w, h float64) ImageConfiguration {
	s := ui.NewSize(w, h)
	return ImageConfiguration{Size: &s}
}

func TestDecorationImageListenerSync(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	painter := NewDecorationImage(DecorationImageOptions{Image: newSynchronousTestImageProvider(image)}).
		CreatePainter(func() { t.Error("onChanged called during paint") })
	painter.Paint(flutter_test.NewTestRecordingCanvas(), ui.RectZero, nil, sizeConfig(0, 0))
}

func TestDecorationImageListenerAsync(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10})
	onChangedCalled := false
	painter := NewDecorationImage(DecorationImageOptions{Image: newAsyncTestImageProvider(image)}).
		CreatePainter(func() { onChangedCalled = true })
	painter.Paint(flutter_test.NewTestRecordingCanvas(), ui.RectZero, nil, sizeConfig(0, 0))
	if onChangedCalled {
		t.Error("called synchronously")
	}
	tester.Loop().FlushMicrotasks()
	if !onChangedCalled {
		t.Error("not called after the microtasks")
	}
}

func TestDecorationImageListenerClone(t *testing.T) {
	tester := newImageTester(t)
	image1 := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10, NoCache: true})
	image2 := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10, NoCache: true})
	completer := NewTestImageStreamCompleter()
	onChangedCalled := false
	painter := NewDecorationImage(DecorationImageOptions{Image: newMultiFrameImageProvider(completer)}).
		CreatePainter(func() { onChangedCalled = true })
	painter.Paint(flutter_test.NewTestRecordingCanvas(), ui.RectZero, nil, sizeConfig(0, 0))
	if onChangedCalled {
		t.Error("called synchronously")
	}
	completer.TestSetImage(image1.Clone())
	tester.Loop().FlushMicrotasks()
	if !onChangedCalled {
		t.Error("first image")
	}
	onChangedCalled = false
	completer.TestSetImage(image1.Clone())
	tester.Loop().FlushMicrotasks()
	if onChangedCalled {
		t.Error("clone of the same image")
	}
	completer.TestSetImage(image2.Clone())
	tester.Loop().FlushMicrotasks()
	if !onChangedCalled {
		t.Error("new image")
	}
}

func TestDecorationImagePaint(t *testing.T) {
	tester := newImageTester(t)
	colorFilter := ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeSrc)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	fit, repeat, opacity, fq := BoxFitContain, ImageRepeatRepeatY, 0.5, ui.FilterQualityHigh
	slice := ui.NewRectFromLTWH(10.0, 20.0, 30.0, 40.0)
	backgroundImage := NewDecorationImage(DecorationImageOptions{Image: newSynchronousTestImageProvider(image),
		ColorFilter: colorFilter, Fit: &fit, Alignment: AlignmentBottomLeft, CenterSlice: &slice, Repeat: &repeat,
		Opacity: &opacity, FilterQuality: &fq, InvertColors: true, IsAntiAlias: true})
	canvas := flutter_test.NewTestRecordingCanvas()
	backgroundImage.CreatePainter(func() { t.Error("onChanged") }).
		Paint(canvas, ui.NewRectFromLTWH(0, 0, 100, 100), nil, sizeConfig(100, 100))
	calls := invocationsNamed(canvas, "drawImageNine")
	if len(calls) != 1 || len(calls[0].PositionalArguments) != 4 {
		t.Fatalf("drawImageNine calls %v", calls)
	}
	args := calls[0].PositionalArguments
	if _, ok := args[0].(*ui.Image); !ok {
		t.Error("image")
	}
	if args[1] != ui.NewRectFromLTRB(10.0, 20.0, 40.0, 60.0) || args[2] != ui.NewRectFromLTRB(0.0, 0.0, 100.0, 100.0) {
		t.Errorf("rects %v %v", args[1], args[2])
	}
	paint := args[3].(*ui.Paint)
	if !paint.ColorFilter().Equal(colorFilter) || !sameColorAs(paint.Color(), ui.NewColor(0x7F000000)) ||
		paint.FilterQuality() != ui.FilterQualityHigh || !paint.IsAntiAlias() || !paint.InvertColors() {
		t.Errorf("paint %v", paint)
	}
}

func TestDecorationImageToString(t *testing.T) {
	tester := newImageTester(t)
	opacity, scale := 0.99, 2.01
	got := NewDecorationImage(DecorationImageOptions{
		Image:   newSynchronousTestImageProvider(createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})),
		Opacity: &opacity, Scale: &scale}).String()
	if want := "DecorationImage(SynchronousTestImageProvider(), Alignment.center, scale 2.0, opacity 1.0, FilterQuality.medium)"; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestDecorationImageNullTextDirection(t *testing.T) {
	tester := newImageTester(t)
	colorFilter := ui.NewColorFilterMode(ui.NewColor(0xFF00FF00), ui.BlendModeSrc)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	fit, repeat, scale, opacity := BoxFitContain, ImageRepeatRepeatY, 0.5, 0.5
	slice := ui.NewRectFromLTWH(10.0, 20.0, 30.0, 40.0)
	backgroundImage := NewDecorationImage(DecorationImageOptions{Image: newSynchronousTestImageProvider(image),
		ColorFilter: colorFilter, Fit: &fit, CenterSlice: &slice, Repeat: &repeat, MatchTextDirection: true,
		Scale: &scale, Opacity: &opacity, InvertColors: true, IsAntiAlias: true})
	var err *foundation.FlutterError
	func() {
		defer func() { err, _ = recover().(*foundation.FlutterError) }()
		backgroundImage.CreatePainter(func() { t.Error("onChanged") }).
			Paint(flutter_test.NewTestRecordingCanvas(), ui.NewRectFromLTWH(0, 0, 100, 100), nil, sizeConfig(100, 100))
	}()
	if err == nil {
		t.Fatal("no FlutterError")
	}
	if len(err.Diagnostics) != 4 {
		t.Fatalf("%d diagnostics", len(err.Diagnostics))
	}
	if _, ok := err.Diagnostics[2].(*foundation.DiagnosticsProperty[DecorationImage]); !ok {
		t.Errorf("diagnostics[2] is %T", err.Diagnostics[2])
	}
	if _, ok := err.Diagnostics[3].(*foundation.DiagnosticsProperty[ImageConfiguration]); !ok {
		t.Errorf("diagnostics[3] is %T", err.Diagnostics[3])
	}
	w := 640
	want := "FlutterError\n" +
		"   DecorationImage.matchTextDirection can only be used when a TextDirection is available.\n" +
		"   When DecorationImagePainter.paint() was called, there was no text direction provided in the ImageConfiguration object to match.\n" +
		"   The DecorationImage was:\n" +
		"     DecorationImage(SynchronousTestImageProvider(), ColorFilter.mode(" + ui.NewColor(0xff00ff00).String() +
		", BlendMode.src), BoxFit.contain, Alignment.center, centerSlice: Rect.fromLTRB(10.0, 20.0, 40.0, 60.0), " +
		"ImageRepeat.repeatY, match text direction, scale 0.5, opacity 0.5, FilterQuality.medium, invert colors, use anti-aliasing)\n" +
		"   The ImageConfiguration was:\n" +
		"     ImageConfiguration(size: Size(100.0, 100.0))\n"
	if got := err.ToStringDeep(foundation.DiagnosticableTreeToStringDeepOptions{WrapWidth: &w}); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestDecorationImageErrorListener(t *testing.T) {
	tester := newImageTester(t)
	exception := ""
	backgroundImage := NewDecorationImage(DecorationImageOptions{Image: newSynchronousErrorTestImageProvider("threw"),
		OnError: func(err any, _ dart.StackTrace) { exception = err.(string) }})
	backgroundImage.CreatePainter(func() {}).Paint(flutter_test.NewTestRecordingCanvas(), ui.RectLargest, ui.NewPath(),
		ImageConfigurationEmpty)
	tester.Loop().FlushMicrotasks()
	if exception != "threw" {
		t.Errorf("exception %q", exception)
	}
}

func TestDecorationLerpIdenticalAndUnrelated(t *testing.T) {
	if DecorationLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	logo := NewFlutterLogoDecoration()
	if DecorationLerp(logo, logo, 0.5) != Decoration(logo) {
		t.Error("identical")
	}
	other := newPlainDecoration()
	for _, c := range []struct {
		t    float64
		logo bool
	}{{0.0, true}, {0.25, true}, {0.75, false}, {1.0, false}} {
		_, isLogo := DecorationLerp(logo, other, c.t).(*FlutterLogoDecoration)
		if isLogo != c.logo {
			t.Errorf("t=%v: logo %v", c.t, isLogo)
		}
	}
}

// plainDecoration is a Decoration that lerps with nothing (BoxDecoration's
// stand-in).
type plainDecoration struct{ DecorationBase }

func newPlainDecoration() *plainDecoration {
	d := &plainDecoration{}
	d.SetSelf(d)
	return d
}

func (d *plainDecoration) CreateBoxPainter(onChanged func()) BoxPainter { return nil }

func paintImageCall(t *testing.T, opts PaintImageOptions, rect ui.Rect, image *ui.Image) []any {
	t.Helper()
	canvas := flutter_test.NewTestRecordingCanvas()
	PaintImage(canvas, rect, image, opts)
	calls := invocationsNamed(canvas, "drawImageRect")
	if len(calls) == 0 || len(calls[0].PositionalArguments) != 4 {
		t.Fatalf("drawImageRect calls %v", calls)
	}
	return calls[0].PositionalArguments
}

func TestPaintImageScale(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	imageSize := ui.NewSize(100, 100)
	outputRect := ui.NewRectFromLTWH(30.0, 30.0, 250.0, 250.0)
	for _, fit := range []BoxFit{BoxFitNone, BoxFitScaleDown} {
		for scale := 1.0; scale <= 4.0; scale += 1.0 {
			f, s := fit, scale
			args := paintImageCall(t, PaintImageOptions{Scale: &s, Alignment: AlignmentBottomRight, Fit: &f}, outputRect, image)
			if args[1] != ui.OffsetZero.And(imageSize) {
				t.Errorf("%v scale %v: source %v %#v", fit, scale, args[1], args[1])
			}
			tile := imageSize.Div(scale)
			br := outputRect.BottomRight()
			if want := ui.NewRectFromPoints(br.Translate(-tile.Width, -tile.Height), br); args[2] != want {
				t.Errorf("%v scale %v: dst %v, want %v", fit, scale, args[2], want)
			}
		}
	}
	// container height (20 px) < scaled image height (50 px)
	f, s := BoxFitScaleDown, 2.0
	outputRect = ui.NewRectFromLTWH(30.0, 30.0, 250.0, 20.0)
	args := paintImageCall(t, PaintImageOptions{Scale: &s, Alignment: AlignmentBottomRight, Fit: &f}, outputRect, image)
	br := outputRect.BottomRight()
	if want := ui.NewRectFromPoints(br.Translate(-20, -20), br); args[2] != want || args[1] != ui.OffsetZero.And(imageSize) {
		t.Errorf("scaleDown: %v %v", args[1], args[2])
	}
}

func TestPaintImageFitScaleAlignment(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	outputRect := ui.NewRectFromLTWH(30.0, 30.0, 250.0, 250.0)
	for _, fit := range []BoxFit{BoxFitContain, BoxFitCover, BoxFitFitWidth, BoxFitFitWidth, BoxFitFitHeight, BoxFitNone, BoxFitScaleDown} {
		f, s := fit, 3.0
		args := paintImageCall(t, PaintImageOptions{Scale: &s, Fit: &f}, outputRect, image)
		if args[2].(ui.Rect).Center() != outputRect.Center() {
			t.Errorf("%v: %v", fit, args[2])
		}
	}
}

func TestPaintImageRepeat(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	imageSize := ui.NewSize(100, 100)
	for _, c := range []struct {
		rect   ui.Rect
		fit    BoxFit
		repeat ImageRepeat
		want   []ui.Rect
	}{
		{ui.NewRectFromLTWH(30.0, 30.0, 400.0, 200.0), BoxFitFitHeight, ImageRepeatRepeatX,
			[]ui.Rect{ui.NewRectFromLTWH(30.0, 30.0, 200.0, 200.0), ui.NewRectFromLTWH(230.0, 30.0, 200.0, 200.0)}},
		{ui.NewRectFromLTWH(30.0, 30.0, 200.0, 400.0), BoxFitFitWidth, ImageRepeatRepeatY,
			[]ui.Rect{ui.NewRectFromLTWH(30.0, 30.0, 200.0, 200.0), ui.NewRectFromLTWH(30.0, 230.0, 200.0, 200.0)}},
	} {
		canvas := flutter_test.NewTestRecordingCanvas()
		PaintImage(canvas, c.rect, image, PaintImageOptions{Alignment: AlignmentTopLeft, Fit: &c.fit, Repeat: &c.repeat})
		calls := invocationsNamed(canvas, "drawImageRect")
		if len(calls) != 2 {
			t.Fatalf("%v: %d calls", c.repeat, len(calls))
		}
		tiles := map[ui.Rect]bool{}
		for _, call := range calls {
			if call.PositionalArguments[1] != ui.OffsetZero.And(imageSize) {
				t.Errorf("source %v", call.PositionalArguments[1])
			}
			tiles[call.PositionalArguments[2].(ui.Rect)] = true
		}
		for _, w := range c.want {
			if !tiles[w] {
				t.Errorf("%v: missing tile %v in %v", c.repeat, w, tiles)
			}
		}
	}
}

func TestDecorationImageScale(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	scale := 4.0
	canvas := flutter_test.NewTestRecordingCanvas()
	NewDecorationImage(DecorationImageOptions{Image: newSynchronousTestImageProvider(image), Scale: &scale,
		Alignment: AlignmentTopLeft}).CreatePainter(func() { t.Error("onChanged") }).
		Paint(canvas, ui.NewRectFromLTWH(0, 0, 100, 100), nil, sizeConfig(100, 100))
	calls := invocationsNamed(canvas, "drawImageRect")
	if len(calls) == 0 || calls[0].PositionalArguments[2] != ui.NewRectFromLTRB(0.0, 0.0, 25.0, 25.0) {
		t.Errorf("calls %v", calls)
	}
}

func TestDecorationImagePlaceholder(t *testing.T) {
	tester := newImageTester(t)
	placeholder := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10})
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	delayed := newDelayedImageProvider(image)
	onChangedCallCount := 0
	painter := NewDecorationImage(DecorationImageOptions{Image: delayed,
		Placeholder: newSynchronousTestImageProvider(placeholder, 3)}).CreatePainter(func() { onChangedCallCount++ })
	defer painter.Dispose()
	rect := ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)
	canvas := flutter_test.NewTestRecordingCanvas()
	painter.Paint(canvas, rect, nil, ImageConfigurationEmpty)
	if d := lastDrawnImage(t, canvas); d.Width != 10 || d.Height != 10 {
		t.Errorf("drew %v", d)
	}
	delayed.Complete()
	tester.Loop().FlushMicrotasks()
	if onChangedCallCount != 1 {
		t.Errorf("onChanged %d", onChangedCallCount)
	}
	canvas = flutter_test.NewTestRecordingCanvas()
	painter.Paint(canvas, rect, nil, ImageConfigurationEmpty)
	if d := lastDrawnImage(t, canvas); d.Width != 100 || d.Height != 100 {
		t.Errorf("drew %v", d)
	}
}

func TestDecorationImageNoPlaceholderWhenAvailable(t *testing.T) {
	tester := newImageTester(t)
	placeholder := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10})
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	painter := NewDecorationImage(DecorationImageOptions{Image: newSynchronousTestImageProvider(image, 4),
		Placeholder: newSynchronousTestImageProvider(placeholder, 5)}).CreatePainter(func() { t.Error("onChanged") })
	defer painter.Dispose()
	canvas := flutter_test.NewTestRecordingCanvas()
	painter.Paint(canvas, ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0), nil, ImageConfigurationEmpty)
	if d := lastDrawnImage(t, canvas); d.Width != 100 {
		t.Errorf("drew %v", d)
	}
}

func TestDecorationImagePainterReleasesPlaceholder(t *testing.T) {
	tester := newImageTester(t)
	placeholder := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10})
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	delayed := newDelayedImageProvider(image)
	painter := NewDecorationImage(DecorationImageOptions{Image: delayed,
		Placeholder: newSynchronousTestImageProvider(placeholder, 7)}).CreatePainter(func() {})
	defer painter.Dispose()
	painter.Paint(flutter_test.NewTestRecordingCanvas(), ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0), nil, ImageConfigurationEmpty)
	handles := len(placeholder.DebugGetOpenHandleStackTraces())
	delayed.Complete()
	tester.Loop().FlushMicrotasks()
	if got := len(placeholder.DebugGetOpenHandleStackTraces()); got != handles-1 {
		t.Errorf("handles %d, want %d", got, handles-1)
	}
}

func TestDecorationImagePlaceholderEquality(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	provider := newSynchronousTestImageProvider(image)
	placeholder := newSynchronousTestImageProvider(image, 6)
	with := NewDecorationImage(DecorationImageOptions{Image: provider, Placeholder: placeholder})
	if with.Equal(NewDecorationImage(DecorationImageOptions{Image: provider})) {
		t.Error("equal without placeholder")
	}
	if !with.Equal(NewDecorationImage(DecorationImageOptions{Image: provider, Placeholder: placeholder})) {
		t.Error("not equal to itself")
	}
	if !strings.Contains(with.String(), "placeholder: SynchronousTestImageProvider()") {
		t.Error(with.String())
	}
}

func TestDecorationImagePainterDisposesImage(t *testing.T) {
	tester := newImageTester(t)
	provider := NewMemoryImage(kTransparentImage)
	stream := provider.Resolve(ImageConfigurationEmpty)
	info := await(tester, extractOneFrame(stream))
	baseline := len(info.Image.DebugGetOpenHandleStackTraces())
	painter := NewDecorationImage(DecorationImageOptions{Image: provider}).CreatePainter(func() {})
	painter.Paint(flutter_test.NewTestRecordingCanvas(), ui.RectZero, ui.NewPath(), ImageConfigurationEmpty)
	if got := len(info.Image.DebugGetOpenHandleStackTraces()); got != baseline+1 {
		t.Errorf("handles after paint %d, want %d", got, baseline+1)
	}
	painter.Dispose()
	if got := len(info.Image.DebugGetOpenHandleStackTraces()); got != baseline {
		t.Errorf("handles after dispose %d, want %d", got, baseline)
	}
	info.Dispose()
}

// sameColorAs is flutter_test's isSameColorAs (components within
// colorEpsilon, 0.004).
func sameColorAs(a, b ui.Color) bool {
	const eps = 0.004
	d := func(x, y float64) bool { return x-y < eps && y-x < eps }
	return d(a.A, b.A) && d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B) && a.ColorSpace == b.ColorSpace
}
