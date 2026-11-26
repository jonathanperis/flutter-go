package painting_test

// Go-only tests of decoration_image.dart's painter (not in Flutter's test
// suite; they cover rules mutation testing found unchecked).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

// directionalProvider's key (and image) depends on the configuration's
// text direction.
type directionalProvider struct {
	ImageProviderBase
	ltr, rtl *ui.Image
}

func (p *directionalProvider) ObtainKey(c ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](*c.TextDirection)
}

func (p *directionalProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	img := p.ltr
	if key.(ui.TextDirection) == ui.TextDirectionRtl {
		img = p.rtl
	}
	return NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(NewImageInfo(img.Clone())))
}

// The painter resolves the image again when the configuration gives a new
// key (and keeps its stream when the key is the same).
func TestDecorationImagePainterNewKey(t *testing.T) {
	tester := newImageTester(t)
	p := &directionalProvider{ltr: createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10}),
		rtl: createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 20, Height: 20})}
	p.SetSelf(p)
	painter := NewDecorationImage(DecorationImageOptions{Image: p}).CreatePainter(func() {})
	defer painter.Dispose()
	rect := ui.NewRectFromLTWH(0, 0, 100, 100)
	for _, c := range []struct {
		dir  ui.TextDirection
		want int
	}{{ui.TextDirectionLtr, 10}, {ui.TextDirectionLtr, 10}, {ui.TextDirectionRtl, 20}} {
		dir := c.dir
		canvas := flutter_test.NewTestRecordingCanvas()
		painter.Paint(canvas, rect, nil, ImageConfiguration{TextDirection: &dir})
		if got := lastDrawnImage(t, canvas).Width; got != c.want {
			t.Errorf("%v: drew a %d-wide image, want %d", dir, got, c.want)
		}
	}
}

// recordingProvider loads its image synchronously and keeps the completer
// it returned.
type recordingProvider struct {
	ImageProviderBase
	image     *ui.Image
	key       int
	completer ImageStreamCompleter
}

func (p *recordingProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p.key)
}

func (p *recordingProvider) LoadImage(key any, decode ImageDecoderCallback) ImageStreamCompleter {
	p.completer = NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(NewImageInfo(p.image.Clone())))
	return p.completer
}

// Once the image has loaded, the painter stops listening to the
// placeholder, and painting again does not resolve it again.
func TestDecorationImagePlaceholderNotResolvedAfterImage(t *testing.T) {
	tester := newImageTester(t)
	placeholder := &recordingProvider{image: createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10}), key: 9}
	placeholder.SetSelf(placeholder)
	delayed := newDelayedImageProvider(createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100}))
	painter := NewDecorationImage(DecorationImageOptions{Image: delayed, Placeholder: placeholder}).CreatePainter(func() {})
	defer painter.Dispose()
	rect := ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0)
	painter.Paint(flutter_test.NewTestRecordingCanvas(), rect, nil, ImageConfigurationEmpty)
	if placeholder.completer == nil || !placeholder.completer.HasListeners() {
		t.Fatal("placeholder not listened to while it is painted")
	}
	delayed.Complete()
	tester.Loop().FlushMicrotasks()
	painter.Paint(flutter_test.NewTestRecordingCanvas(), rect, nil, ImageConfigurationEmpty)
	if placeholder.completer.HasListeners() {
		t.Error("placeholder listened to after the image loaded")
	}
}
