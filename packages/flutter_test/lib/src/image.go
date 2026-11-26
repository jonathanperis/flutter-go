package flutter_test

// Port of flutter_test's image.dart (Flutter 3.50.1): createTestImage and
// precacheTestImage, plus RunAsync/AwaitFuture, the Go stand-ins for an
// async test body (Dart's `test(() async {...})` and tester.runAsync).

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

var testImageCache = map[[2]int]*ui.Image{}

// CreateTestImageOptions are createTestImage's named parameters (Width and
// Height 0: 1; NoCache is cache: false).
type CreateTestImageOptions struct {
	Width, Height int
	NoCache       bool
}

// CreateTestImage creates an arbitrary (transparent) image for testing,
// cached by size unless NoCache: a cached image is returned as a clone.
func CreateTestImage(opts ...CreateTestImageOptions) async.Future[*ui.Image] {
	var o CreateTestImageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	width, height := o.Width, o.Height
	if width == 0 {
		width = 1
	}
	if height == 0 {
		height = 1
	}
	return async.Async(func() *ui.Image {
		if width <= 0 || height <= 0 {
			panic("createTestImage: width and height must be positive")
		}
		cacheKey := [2]int{width, height}
		if cached, ok := testImageCache[cacheKey]; ok && !o.NoCache {
			return cached.Clone()
		}
		completer := async.NewCompleter[*ui.Image]()
		ui.DecodeImageFromPixels(make([]byte, width*height*4), width, height, ui.PixelFormatRgba8888,
			func(image *ui.Image) { completer.Complete(image) })
		image := completer.Future().Await()
		if !o.NoCache {
			testImageCache[cacheKey] = image.Clone()
		}
		return image
	})
}

// PrecacheTestImageOptions are precacheTestImage's named parameters
// (Configuration nil: ImageConfigurationEmpty).
type PrecacheTestImageOptions struct {
	Configuration *painting.ImageConfiguration
	OnError       painting.ImageErrorListener
}

// PrecacheTestImage resolves provider (outside the fake clock, as Dart's
// runAsync does) so that its image is in the image cache; an error is
// passed to OnError, or panics.
func (t *Tester) PrecacheTestImage(provider painting.ImageProvider, opts ...PrecacheTestImageOptions) {
	var o PrecacheTestImageOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	configuration := painting.ImageConfigurationEmpty
	if o.Configuration != nil {
		configuration = *o.Configuration
	}
	var caughtError any
	t.RunAsync(func() {
		completer := async.NewSyncCompleter[async.Void]()
		stream := provider.Resolve(configuration)
		var listener painting.ImageStreamListener
		listener = painting.NewImageStreamListener(func(info *painting.ImageInfo, syncCall bool) {
			info.Dispose()
			if !completer.IsCompleted() {
				stream.RemoveListener(listener)
				completer.Complete(async.Void{})
			}
		}, painting.ImageStreamListenerOptions{OnError: func(err any, _ dart.StackTrace) {
			if !completer.IsCompleted() {
				stream.RemoveListener(listener)
				if o.OnError != nil {
					o.OnError(err, nil)
				} else {
					caughtError = err
				}
				completer.Complete(async.Void{})
			}
		}})
		stream.AddListener(listener)
		completer.Future().Await()
	})
	if caughtError != nil {
		panic(caughtError)
	}
}

// RunAsync runs body as an async function (it may Await futures) on the
// test's event loop and drives the loop until it returns: microtasks, then
// events posted from other goroutines (image decoding, isolates), then —
// when nothing else is pending — the fake clock jumps to the next timer.
// A panic in body is re-raised.
func (t *Tester) RunAsync(body func()) {
	AwaitFuture(t, async.AsyncVoid(body))
}

// AwaitFuture drives the test's event loop (see RunAsync) until f
// completes, and returns its value (or panics with its error).
func AwaitFuture[T any](t *Tester, f async.Future[T]) T {
	return awaitOnLoop(t.loop, f)
}

func awaitOnLoop[T any](loop *async.Loop, f async.Future[T]) T {
	var v T
	var err error
	done := false
	async.OnComplete(f, func(x T) { v, done = x, true }, func(e error) { err, done = e, true })
	for {
		loop.FlushMicrotasks()
		if done {
			break
		}
		if loop.RunPosted() > 0 {
			continue
		}
		if deadline, ok := loop.NextDeadline(); ok {
			// Give work in flight on other goroutines a moment to post its
			// result, as it would arrive before a timer in real time.
			posted := false
			for range 20 {
				time.Sleep(time.Millisecond)
				if loop.RunPosted() > 0 {
					posted = true
					break
				}
			}
			if !posted {
				loop.ElapseTo(deadline)
			}
			continue
		}
		loop.WaitPosted()
	}
	if err != nil {
		panic(err)
	}
	return v
}
