package painting_test

// Port of packages/flutter/test/painting/image_stream_test.dart (Flutter
// 3.50.1): all 33 tests. tester.idle() is a zero elapse of the fake clock
// (microtasks and zero-duration timers); addTearDown(image.dispose)
// disposes at the end of the test; the leak tracker's memoryEvents(...,
// areCreateAndDispose) listens to FlutterMemoryAllocations directly. The
// hashCode half of "ImageStreamListener hashCode and equals" has no Go
// counterpart (== is Equal).

import (
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// MockCodec hands out frames when the test completes them.
type MockCodec struct {
	frameCount, repetitionCount int
	numFramesAsked              int
	disposed                    bool
	nextFrameCompleter          *async.Completer[*ui.FrameInfo]
}

func newMockCodec(frameCount, repetitionCount int) *MockCodec {
	return &MockCodec{frameCount: frameCount, repetitionCount: repetitionCount, nextFrameCompleter: async.NewCompleter[*ui.FrameInfo]()}
}

func (c *MockCodec) FrameCount() int      { return c.frameCount }
func (c *MockCodec) RepetitionCount() int { return c.repetitionCount }
func (c *MockCodec) GetNextFrame() async.Future[*ui.FrameInfo] {
	if c.disposed {
		panic(dart.StateError{Message: "Codec is disposed"})
	}
	c.numFramesAsked++
	return c.nextFrameCompleter.Future()
}
func (c *MockCodec) completeNextFrame(frameInfo *ui.FrameInfo) {
	c.nextFrameCompleter.Complete(frameInfo)
	c.nextFrameCompleter = async.NewCompleter[*ui.FrameInfo]()
}
func (c *MockCodec) failNextFrame(err string) {
	c.nextFrameCompleter.CompleteError(async.PanicError{Value: err})
}
func (c *MockCodec) Dispose() {
	if c.disposed {
		panic(dart.StateError{Message: "Codec is already disposed"})
	}
	c.disposed = true
}

func fakeFrame(d time.Duration, image *ui.Image) *ui.FrameInfo {
	return &ui.FrameInfo{Duration: d, Image: image}
}

func cloneFrame(f *ui.FrameInfo) *ui.FrameInfo {
	return &ui.FrameInfo{Duration: f.Duration, Image: f.Image.Clone()}
}

// FakeEventReportingImageStreamCompleter reports the chunk events of a
// stream.
type FakeEventReportingImageStreamCompleter struct{ ImageStreamCompleterBase }

func newFakeEventReportingImageStreamCompleter(chunkEvents async.Stream[*ImageChunkEvent]) *FakeEventReportingImageStreamCompleter {
	c := &FakeEventReportingImageStreamCompleter{}
	c.InitImageStreamCompleter(c)
	if chunkEvents != nil {
		chunkEvents.Listen(c.ReportImageChunkEvent)
	}
	return c
}

type streamFixture struct {
	tester       *flutter_test.Tester
	image20x10   *ui.Image
	image200x100 *ui.Image
	toDispose    []*ImageInfo
}

func newStreamFixture(t *testing.T) *streamFixture {
	tester := flutter_test.New(t)
	f := &streamFixture{tester: tester}
	f.image20x10 = createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 20, Height: 10})
	f.image200x100 = createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 200, Height: 100})
	t.Cleanup(func() {
		for _, i := range f.toDispose {
			i.Dispose()
		}
	})
	return f
}

func (f *streamFixture) idle() { f.tester.Loop().Elapse(0) }

// disposing is a listener that disposes its images when the test ends.
func (f *streamFixture) disposing(images *[]*ImageInfo) ImageListener {
	return func(image *ImageInfo, synchronousCall bool) {
		if images != nil {
			*images = append(*images, image)
		}
		f.toDispose = append(f.toDispose, image)
	}
}

func TestImageStreamCodecFutureFails(t *testing.T) {
	f := newStreamFixture(t)
	completer := async.NewCompleter[ui.Codec]()
	NewMultiFrameImageStreamCompleter(completer.Future(), 1.0)
	completer.CompleteError(async.PanicError{Value: "failure message"})
	f.idle()
	if e := f.tester.TakeException(); e != "failure message" {
		t.Errorf("exception %v", e)
	}
}

func TestImageStreamDecodingStartsWhenListenerAddedAfterCodecReady(t *testing.T) {
	f := newStreamFixture(t)
	completer := async.NewCompleter[ui.Codec]()
	mockCodec := newMockCodec(1, 0)
	imageStream := NewMultiFrameImageStreamCompleter(completer.Future(), 1.0)
	completer.Complete(mockCodec)
	f.idle()
	if mockCodec.numFramesAsked != 0 {
		t.Error("asked")
	}
	imageStream.AddListener(NewImageStreamListener(f.disposing(nil)))
	f.idle()
	if mockCodec.numFramesAsked != 1 || mockCodec.disposed {
		t.Errorf("asked %d disposed %v", mockCodec.numFramesAsked, mockCodec.disposed)
	}
}

func TestImageStreamDecodingStartsWhenCodecReadyAfterListenerAdded(t *testing.T) {
	f := newStreamFixture(t)
	completer := async.NewCompleter[ui.Codec]()
	mockCodec := newMockCodec(1, 0)
	imageStream := NewMultiFrameImageStreamCompleter(completer.Future(), 1.0)
	imageStream.AddListener(NewImageStreamListener(f.disposing(nil)))
	f.idle()
	if mockCodec.numFramesAsked != 0 {
		t.Error("asked")
	}
	completer.Complete(mockCodec)
	f.idle()
	if mockCodec.numFramesAsked != 1 || mockCodec.disposed {
		t.Errorf("asked %d disposed %v", mockCodec.numFramesAsked, mockCodec.disposed)
	}
}

func TestImageStreamDecodingDoesNotCrashWhenDisposed(t *testing.T) {
	f := newStreamFixture(t)
	completer := async.NewCompleter[ui.Codec]()
	mockCodec := newMockCodec(1, 0)
	imageStream := NewMultiFrameImageStreamCompleter(completer.Future(), 1.0)
	completer.Complete(mockCodec)
	f.idle()
	streamListener := NewImageStreamListener(f.disposing(nil))
	imageStream.AddListener(streamListener)
	f.idle()
	if mockCodec.numFramesAsked != 1 {
		t.Error("asked")
	}
	mockCodec.completeNextFrame(fakeFrame(200*time.Millisecond, f.image20x10))
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.RemoveListener(streamListener)
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
	f.idle()
}

func chunk(cumulative, total int) *ImageChunkEvent { return NewImageChunkEvent(cumulative, &total) }

func collectChunks(chunks *[]*ImageChunkEvent) ImageStreamListener {
	return NewImageStreamListener(func(*ImageInfo, bool) {}, ImageStreamListenerOptions{
		OnChunk: func(e *ImageChunkEvent) { *chunks = append(*chunks, e) },
	})
}

func checkChunks(t *testing.T, chunks []*ImageChunkEvent, want ...int) {
	t.Helper()
	if len(chunks) != len(want) {
		t.Fatalf("%d chunks, want %d", len(chunks), len(want))
	}
	for i, c := range chunks {
		if c.CumulativeBytesLoaded != want[i] || *c.ExpectedTotalBytes != 3 {
			t.Errorf("chunk %d: %d/%d", i, c.CumulativeBytesLoaded, *c.ExpectedTotalBytes)
		}
	}
}

func TestImageStreamChunkEventsOfBaseCompleterAreDelivered(t *testing.T) {
	f := newStreamFixture(t)
	var chunks []*ImageChunkEvent
	streamController := async.NewStreamController[*ImageChunkEvent]()
	imageStream := newFakeEventReportingImageStreamCompleter(streamController.Stream())
	imageStream.AddListener(collectChunks(&chunks))
	streamController.Add(chunk(1, 3))
	streamController.Add(chunk(2, 3))
	f.idle()
	checkChunks(t, chunks, 1, 2)
}

func TestImageStreamChunkEventsOfBaseCompleterAreNotBuffered(t *testing.T) {
	f := newStreamFixture(t)
	var chunks []*ImageChunkEvent
	streamController := async.NewStreamController[*ImageChunkEvent]()
	imageStream := newFakeEventReportingImageStreamCompleter(streamController.Stream())
	streamController.Add(chunk(1, 3))
	f.idle()
	imageStream.AddListener(collectChunks(&chunks))
	streamController.Add(chunk(2, 3))
	f.idle()
	checkChunks(t, chunks, 2)
}

func TestImageStreamChunkEventsOfMultiFrameCompleterAreDelivered(t *testing.T) {
	f := newStreamFixture(t)
	var chunks []*ImageChunkEvent
	completer := async.NewCompleter[ui.Codec]()
	streamController := async.NewStreamController[*ImageChunkEvent]()
	imageStream := NewMultiFrameImageStreamCompleter(completer.Future(), 1.0,
		MultiFrameImageStreamCompleterOptions{ChunkEvents: streamController.Stream()})
	imageStream.AddListener(collectChunks(&chunks))
	streamController.Add(chunk(1, 3))
	streamController.Add(chunk(2, 3))
	f.idle()
	checkChunks(t, chunks, 1, 2)
}

func TestImageStreamChunkEventsOfMultiFrameCompleterAreNotBuffered(t *testing.T) {
	f := newStreamFixture(t)
	var chunks []*ImageChunkEvent
	completer := async.NewCompleter[ui.Codec]()
	streamController := async.NewStreamController[*ImageChunkEvent]()
	imageStream := NewMultiFrameImageStreamCompleter(completer.Future(), 1.0,
		MultiFrameImageStreamCompleterOptions{ChunkEvents: streamController.Stream()})
	streamController.Add(chunk(1, 3))
	f.idle()
	imageStream.AddListener(collectChunks(&chunks))
	streamController.Add(chunk(2, 3))
	f.idle()
	checkChunks(t, chunks, 2)
}

func TestImageStreamChunkErrorsAreReported(t *testing.T) {
	f := newStreamFixture(t)
	var chunks []*ImageChunkEvent
	completer := async.NewCompleter[ui.Codec]()
	streamController := async.NewStreamController[*ImageChunkEvent]()
	imageStream := NewMultiFrameImageStreamCompleter(completer.Future(), 1.0,
		MultiFrameImageStreamCompleterOptions{ChunkEvents: streamController.Stream()})
	imageStream.AddListener(collectChunks(&chunks))
	streamController.AddError(dartError{})
	streamController.Add(chunk(2, 3))
	f.idle()
	if f.tester.TakeException() == nil {
		t.Error("no exception")
	}
	checkChunks(t, chunks, 2)
}

func TestImageStreamGetNextFrameFutureFails(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(1, 0)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	imageStream.AddListener(NewImageStreamListener(f.disposing(nil)))
	codecCompleter.Complete(mockCodec)
	// MultiFrameImageStreamCompleter only sets an error handler for the next
	// frame future after the codec future has completed.
	f.idle()
	mockCodec.failNextFrame("frame completion error")
	f.idle()
	if e := f.tester.TakeException(); e != "frame completion error" {
		t.Errorf("exception %v", e)
	}
	if mockCodec.disposed {
		t.Error("disposed")
	}
}

func TestImageStreamEmitsFrameStaticImage(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(1, 0)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	var emitted []*ImageInfo
	listener := NewImageStreamListener(f.disposing(&emitted))
	imageStream.AddListener(listener)
	codecCompleter.Complete(mockCodec)
	f.idle()
	frame := fakeFrame(200*time.Millisecond, f.image20x10)
	mockCodec.completeNextFrame(frame)
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	f.idle()
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
	for _, info := range emitted {
		if !info.Image.IsCloneOf(frame.Image) {
			t.Error("not a clone")
		}
	}
	imageStream.RemoveListener(listener)
	ImageCacheInstance().Clear()
}

func TestImageStreamEmitsFramesAnimated(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	var emitted []*ImageInfo
	listener := NewImageStreamListener(f.disposing(&emitted))
	imageStream.AddListener(listener)
	codecCompleter.Complete(mockCodec)
	f.idle()
	frame1 := fakeFrame(200*time.Millisecond, f.image20x10)
	mockCodec.completeNextFrame(frame1)
	f.idle()
	// We are waiting for the next animation tick, so at this point no
	// frames should have been emitted.
	if len(emitted) != 0 {
		t.Fatal("emitted early")
	}
	f.tester.Pump()
	if len(emitted) != 1 || !emitted[0].Image.IsCloneOf(frame1.Image) {
		t.Fatal("first frame")
	}
	frame2 := fakeFrame(400*time.Millisecond, f.image200x100)
	mockCodec.completeNextFrame(frame2)
	f.tester.Pump(100 * time.Millisecond)
	// The duration for the current frame was 200ms, so we don't emit the
	// next frame yet even though it is ready.
	if len(emitted) != 1 {
		t.Fatal("emitted too early")
	}
	f.tester.Pump(100 * time.Millisecond)
	if len(emitted) != 2 || !emitted[0].Image.IsCloneOf(frame1.Image) || !emitted[1].Image.IsCloneOf(frame2.Image) {
		t.Fatal("second frame")
	}
	// Let the pending timer for the next frame to complete so we can
	// cleanly quit the test without pending timers.
	f.tester.Pump(400 * time.Millisecond)
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.RemoveListener(listener)
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
	ImageCacheInstance().Clear()
}

func TestImageStreamAnimationWrapsBack(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	var emitted []*ImageInfo
	listener := NewImageStreamListener(f.disposing(&emitted))
	imageStream.AddListener(listener)
	codecCompleter.Complete(mockCodec)
	f.idle()
	frame1 := fakeFrame(200*time.Millisecond, f.image20x10)
	frame2 := fakeFrame(400*time.Millisecond, f.image200x100)
	mockCodec.completeNextFrame(cloneFrame(frame1))
	f.idle()
	f.tester.Pump()
	mockCodec.completeNextFrame(cloneFrame(frame2))
	f.idle()
	f.tester.Pump(200 * time.Millisecond)
	mockCodec.completeNextFrame(cloneFrame(frame1))
	f.idle()
	f.tester.Pump(400 * time.Millisecond)
	if len(emitted) != 3 || !emitted[0].Image.IsCloneOf(frame1.Image) || !emitted[1].Image.IsCloneOf(frame2.Image) ||
		!emitted[2].Image.IsCloneOf(frame1.Image) {
		t.Fatalf("emitted %d", len(emitted))
	}
	f.tester.Pump(200 * time.Millisecond)
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.RemoveListener(listener)
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
	ImageCacheInstance().Clear()
}

func TestImageStreamAnimationDoesNotRepeatMoreThanSpecified(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, 0)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	var emitted []*ImageInfo
	listener := NewImageStreamListener(f.disposing(&emitted))
	imageStream.AddListener(listener)
	codecCompleter.Complete(mockCodec)
	f.idle()
	frame1 := fakeFrame(200*time.Millisecond, f.image20x10)
	frame2 := fakeFrame(400*time.Millisecond, f.image200x100)
	mockCodec.completeNextFrame(frame1)
	f.idle()
	f.tester.Pump()
	mockCodec.completeNextFrame(frame2)
	f.idle()
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	f.tester.Pump(200 * time.Millisecond)
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
	mockCodec.completeNextFrame(frame1)
	// allow another frame to complete (but we shouldn't be asking for it as
	// this animation should not repeat.
	f.idle()
	f.tester.Pump(400 * time.Millisecond)
	if len(emitted) != 2 || !emitted[0].Image.IsCloneOf(frame1.Image) || !emitted[1].Image.IsCloneOf(frame2.Image) {
		t.Fatalf("emitted %d", len(emitted))
	}
	imageStream.RemoveListener(listener)
	ImageCacheInstance().Clear()
}

func TestImageStreamFramesOnlyDecodedWithListeners(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	listener := NewImageStreamListener(f.disposing(nil))
	imageStream.AddListener(listener)
	handle := imageStream.KeepAlive()
	codecCompleter.Complete(mockCodec)
	f.idle()
	frame1 := fakeFrame(200*time.Millisecond, f.image20x10)
	frame2 := fakeFrame(400*time.Millisecond, f.image200x100)
	mockCodec.completeNextFrame(frame1)
	f.idle()
	f.tester.Pump()
	mockCodec.completeNextFrame(frame2)
	imageStream.RemoveListener(listener)
	f.idle()
	f.tester.Pump(400 * time.Millisecond)
	// Decoding of the 3rd frame should not start as there are no
	// registered listeners to the stream
	if mockCodec.numFramesAsked != 2 {
		t.Errorf("asked %d", mockCodec.numFramesAsked)
	}
	imageStream.AddListener(listener)
	f.idle()
	if mockCodec.numFramesAsked != 3 {
		t.Errorf("asked %d", mockCodec.numFramesAsked)
	}
	handle.Dispose()
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.RemoveListener(listener)
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
	ImageCacheInstance().Clear()
}

func TestImageStreamReAddingListenerWhileFrameCallbackPending(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	var emitted []*ImageInfo
	listener := NewImageStreamListener(f.disposing(&emitted))
	imageStream.AddListener(listener)
	// Mirrors ImageCache keeping the completer alive with zero listeners.
	handle := imageStream.KeepAlive()
	codecCompleter.Complete(mockCodec)
	f.idle()
	// Decoding the first frame schedules a frame callback.
	mockCodec.completeNextFrame(fakeFrame(200*time.Millisecond, f.image20x10))
	f.idle()
	if mockCodec.numFramesAsked != 1 {
		t.Errorf("asked %d", mockCodec.numFramesAsked)
	}
	// Before that callback runs, drop to zero listeners and re-add one.
	imageStream.RemoveListener(listener)
	imageStream.AddListener(listener)
	if mockCodec.numFramesAsked != 2 {
		t.Errorf("asked %d", mockCodec.numFramesAsked)
	}
	// The stale callback must not touch the discarded frame.
	f.tester.Pump()
	if e := f.tester.TakeException(); e != nil || len(emitted) != 0 {
		t.Errorf("exception %v, emitted %d", e, len(emitted))
	}
	// When the restarted decode completes, the animation resumes.
	mockCodec.completeNextFrame(fakeFrame(200*time.Millisecond, f.image200x100))
	f.idle()
	f.tester.Pump()
	if len(emitted) != 1 || emitted[0].Image.Width != 200 {
		t.Errorf("emitted %d", len(emitted))
	}
	handle.Dispose()
	imageStream.RemoveListener(listener)
	ImageCacheInstance().Clear()
}

func TestImageStreamMultipleStreamListeners(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	var emitted1, emitted2 []*ImageInfo
	listener1 := NewImageStreamListener(f.disposing(&emitted1))
	listener2 := NewImageStreamListener(f.disposing(&emitted2))
	imageStream.AddListener(listener1)
	imageStream.AddListener(listener2)
	codecCompleter.Complete(mockCodec)
	f.idle()
	frame1 := fakeFrame(200*time.Millisecond, f.image20x10)
	frame2 := fakeFrame(400*time.Millisecond, f.image200x100)
	mockCodec.completeNextFrame(frame1)
	f.idle()
	f.tester.Pump()
	if len(emitted1) != 1 || !emitted1[0].Image.IsCloneOf(frame1.Image) || len(emitted2) != 1 || !emitted2[0].Image.IsCloneOf(frame1.Image) {
		t.Fatal("first frame")
	}
	mockCodec.completeNextFrame(frame2)
	f.idle()
	f.tester.Pump() // next app frame will schedule a timer.
	imageStream.RemoveListener(listener1)
	f.tester.Pump(400 * time.Millisecond) // emit 2nd frame.
	if len(emitted1) != 1 || len(emitted2) != 2 || !emitted2[1].Image.IsCloneOf(frame2.Image) {
		t.Fatal("second frame")
	}
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.RemoveListener(listener2)
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
}

func TestImageStreamTimerCanceledWhenListenersRemoved(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	listener := f.disposing(nil)
	imageStream.AddListener(NewImageStreamListener(listener))
	codecCompleter.Complete(mockCodec)
	f.idle()
	mockCodec.completeNextFrame(fakeFrame(200*time.Millisecond, f.image20x10))
	f.idle()
	f.tester.Pump()
	mockCodec.completeNextFrame(fakeFrame(400*time.Millisecond, f.image200x100))
	f.idle()
	f.tester.Pump()
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.RemoveListener(NewImageStreamListener(listener))
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
	// The test framework fails this if there are pending timers.
}

func TestImageStreamTimeDilationAffectsFrameTimers(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	listener := NewImageStreamListener(f.disposing(nil))
	imageStream.AddListener(listener)
	codecCompleter.Complete(mockCodec)
	f.idle()
	mockCodec.completeNextFrame(fakeFrame(200*time.Millisecond, f.image20x10))
	f.idle()
	f.tester.Pump()
	scheduler.SetTimeDilation(2.0)
	mockCodec.completeNextFrame(fakeFrame(400*time.Millisecond, f.image200x100))
	f.idle()
	f.tester.Pump()                       // schedule next app frame
	f.tester.Pump(200 * time.Millisecond) // emit 2nd frame.
	// Decoding of the 3rd frame should not start after 200 ms, as time is
	// dilated by a factor of 2.
	if mockCodec.numFramesAsked != 2 {
		t.Errorf("asked %d", mockCodec.numFramesAsked)
	}
	f.tester.Pump(200 * time.Millisecond)
	if mockCodec.numFramesAsked != 3 {
		t.Errorf("asked %d", mockCodec.numFramesAsked)
	}
	scheduler.SetTimeDilation(1.0)
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.RemoveListener(listener)
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
}

func TestImageStreamErrorHandlersCanInterceptErrors(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(1, 0)
	codecCompleter := async.NewCompleter[ui.Codec]()
	streamUnderTest := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	var captured any
	streamUnderTest.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {},
		ImageStreamListenerOptions{OnError: func(exception any, _ dart.StackTrace) { captured = exception }}))
	codecCompleter.Complete(mockCodec)
	f.idle()
	mockCodec.failNextFrame("frame completion error")
	f.idle()
	// No exception is passed up.
	if e := f.tester.TakeException(); e != nil {
		t.Errorf("exception %v", e)
	}
	if captured != "frame completion error" || mockCodec.disposed {
		t.Errorf("captured %v, disposed %v", captured, mockCodec.disposed)
	}
}

func TestImageStreamRemoveAndAddListener(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(3, 0)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	listener := f.disposing(nil)
	imageStream.AddListener(NewImageStreamListener(listener))
	codecCompleter.Complete(mockCodec)
	f.idle()
	imageStream.AddListener(NewImageStreamListener(listener))
	imageStream.RemoveListener(NewImageStreamListener(listener))
	mockCodec.completeNextFrame(fakeFrame(200*time.Millisecond, f.image20x10))
	f.idle()
	f.tester.Pump()
	f.tester.Pump(200 * time.Millisecond)
	if mockCodec.disposed {
		t.Error("disposed")
	}
	// (leaking by design: the stream still has a listener; drop it so the
	// test ends without pending frame timers.)
	imageStream.RemoveListener(NewImageStreamListener(listener))
}

func handleImage(*ImageInfo, bool)            {}
func handleImageDifferently(*ImageInfo, bool) {}
func handleError(any, dart.StackTrace)        {}
func handleChunk(*ImageChunkEvent)            {}

func TestImageStreamListenerEquals(t *testing.T) {
	newImageTester(t)
	compare := func(onImage1, onImage2 ImageListener, onChunk1, onChunk2 ImageChunkListener, onError1, onError2 ImageErrorListener, areEqual bool) {
		t.Helper()
		l1 := NewImageStreamListener(onImage1, ImageStreamListenerOptions{OnChunk: onChunk1, OnError: onError1})
		l2 := NewImageStreamListener(onImage2, ImageStreamListenerOptions{OnChunk: onChunk2, OnError: onError2})
		if l1.Equal(l2) != areEqual {
			t.Errorf("equal %v, want %v", l1.Equal(l2), areEqual)
		}
	}
	compare(handleImage, handleImage, nil, nil, nil, nil, true)
	compare(handleImage, handleImageDifferently, nil, nil, nil, nil, false)
	compare(handleImage, handleImage, handleChunk, handleChunk, nil, nil, true)
	compare(handleImage, handleImage, handleChunk, handleChunk, handleError, handleError, true)
	compare(handleImage, handleImage, handleChunk, nil, nil, nil, false)
	compare(handleImage, handleImage, handleChunk, nil, handleError, nil, false)
	compare(handleImage, handleImage, handleChunk, handleChunk, handleError, nil, false)
	compare(handleImage, handleImage, handleChunk, nil, handleError, handleError, false)
}

func TestImageStreamKeepAliveHandlesDoNotDriveFrames(t *testing.T) {
	f := newStreamFixture(t)
	image10x10 := createTestImage(f.tester, flutter_test.CreateTestImageOptions{Width: 10, Height: 10})
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	onImageCount := 0
	activeListener := func(image *ImageInfo, synchronousCall bool) {
		onImageCount++
		f.toDispose = append(f.toDispose, image)
	}
	lastListenerDropped := false
	imageStream.AddOnLastListenerRemovedCallback(func() { lastListenerDropped = true })
	handle := imageStream.KeepAlive()
	if lastListenerDropped {
		t.Fatal("dropped")
	}
	b := scheduler.SchedulerBindingInstance()
	noTransient := func() {
		t.Helper()
		if n := b.TransientCallbackCount(); n != 0 {
			t.Fatalf("Only passive listeners: %d transient callbacks", n)
		}
	}
	noTransient()
	codecCompleter.Complete(mockCodec)
	f.idle()
	if onImageCount != 0 {
		t.Fatal("onImage")
	}
	frame1 := fakeFrame(0, f.image20x10)
	mockCodec.completeNextFrame(frame1)
	f.idle()
	noTransient()
	f.tester.Pump()
	if onImageCount != 0 {
		t.Fatal("onImage")
	}
	imageStream.AddListener(NewImageStreamListener(activeListener))
	frame2 := fakeFrame(0, image10x10)
	mockCodec.completeNextFrame(frame2)
	f.idle()
	if b.TransientCallbackCount() != 1 {
		t.Fatalf("transient %d", b.TransientCallbackCount())
	}
	f.tester.Pump()
	if onImageCount != 1 {
		t.Fatalf("onImage %d", onImageCount)
	}
	imageStream.RemoveListener(NewImageStreamListener(activeListener))
	if !lastListenerDropped {
		t.Fatal("not dropped")
	}
	mockCodec.completeNextFrame(frame1)
	f.idle()
	if b.TransientCallbackCount() != 1 {
		t.Fatalf("transient %d", b.TransientCallbackCount())
	}
	f.tester.Pump()
	if onImageCount != 1 {
		t.Fatalf("onImage %d", onImageCount)
	}
	noTransient()
	mockCodec.completeNextFrame(frame2)
	f.idle()
	noTransient()
	f.tester.Pump()
	if onImageCount != 1 || mockCodec.disposed {
		t.Fatalf("onImage %d", onImageCount)
	}
	handle.Dispose()
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
}

func TestMultiFrameImageStreamCompleterOneFrameDecodedOnce(t *testing.T) {
	tester := newImageTester(t)
	oneFrameCodec := await(tester, FakeCodecFromData(kTransparentImage))
	codecCompleter := async.NewCompleter[ui.Codec]()
	decodeCompleter := async.NewCompleter[async.Void]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	imageListener := NewImageStreamListener(func(*ImageInfo, bool) { decodeCompleter.Complete(async.Void{}) })
	imageStream.KeepAlive() // do not dispose
	imageStream.AddListener(imageListener)
	codecCompleter.Complete(oneFrameCodec)
	await(tester, decodeCompleter.Future())
	imageStream.RemoveListener(imageListener)
	if oneFrameCodec.NumFramesAsked != 1 {
		t.Errorf("asked %d", oneFrameCodec.NumFramesAsked)
	}
	// Adding a new listener for decoded imageSteam, the one frame image
	// should not be decoded again.
	imageStream.AddListener(NewImageStreamListener(func(*ImageInfo, bool) {}))
	if oneFrameCodec.NumFramesAsked != 1 {
		t.Errorf("asked %d", oneFrameCodec.NumFramesAsked)
	}
}

func TestMultiFrameCompleterUnsubscribesFromChunkEventsWhenDisposed(t *testing.T) {
	tester := newImageTester(t)
	codec := await(tester, FakeCodecFromData(kTransparentImage))
	chunkStream := async.NewStreamController[*ImageChunkEvent]()
	completer := NewMultiFrameImageStreamCompleter(async.FutureValue[ui.Codec](codec), 1.0,
		MultiFrameImageStreamCompleterOptions{ChunkEvents: chunkStream.Stream()})
	if !chunkStream.HasListener() {
		t.Error("no listener")
	}
	chunkStream.Add(chunk(1, 3))
	listener := NewImageStreamListener(func(*ImageInfo, bool) {})
	// Cause the completer to dispose.
	completer.AddListener(listener)
	completer.RemoveListener(listener)
	if chunkStream.HasListener() {
		t.Error("still listening")
	}
	chunkStream.Add(chunk(2, 3))
	tester.Loop().FlushMicrotasks()
}

func TestImageStreamSetCompleterBeforeAddListenerIsSynchronous(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	imageStreamCompleter := NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(&NewTestImageInfo(1, image).ImageInfo))
	imageStream := NewImageStream()
	imageStream.SetCompleter(imageStreamCompleter)
	var synchronouslyCalled *bool
	imageStream.AddListener(NewImageStreamListener(func(_ *ImageInfo, synchronousCall bool) { synchronouslyCalled = &synchronousCall }))
	if synchronouslyCalled == nil || !*synchronouslyCalled {
		t.Error("not synchronous")
	}
}

func TestImageStreamSetCompleterAfterAddListenerIsNotSynchronous(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 100, Height: 100})
	imageStreamCompleter := NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(&NewTestImageInfo(1, image).ImageInfo))
	imageStream := NewImageStream()
	var synchronouslyCalled *bool
	imageStream.AddListener(NewImageStreamListener(func(_ *ImageInfo, synchronousCall bool) { synchronouslyCalled = &synchronousCall }))
	imageStream.SetCompleter(imageStreamCompleter)
	if synchronouslyCalled == nil || *synchronouslyCalled {
		t.Error("synchronous")
	}
}

// memoryEvents records the creation and disposal events of objects of
// type T while f runs (leak_tracker's memoryEvents + areCreateAndDispose).
func checkCreateAndDispose[T any](t *testing.T, f func()) {
	t.Helper()
	var created, disposed int
	listener := func(e foundation.ObjectEvent) {
		if _, ok := e.Object().(T); !ok {
			return
		}
		switch e.(type) {
		case *foundation.ObjectCreated:
			created++
		case *foundation.ObjectDisposed:
			disposed++
		}
	}
	foundation.FlutterMemoryAllocationsInstance.AddListener(listener)
	defer foundation.FlutterMemoryAllocationsInstance.RemoveListener(listener)
	f()
	if foundation.KFlutterMemoryAllocationsEnabled && (created != 1 || disposed != 1) {
		t.Errorf("created %d, disposed %d", created, disposed)
	}
}

func TestImageStreamCompleterHandleDispatchesMemoryEvents(t *testing.T) {
	newImageTester(t)
	checkCreateAndDispose[*ImageStreamCompleterHandle](t, func() {
		streamController := async.NewStreamController[*ImageChunkEvent]()
		defer streamController.Close()
		handle := newFakeEventReportingImageStreamCompleter(streamController.Stream()).KeepAlive()
		handle.Dispose()
	})
}

func TestImageInfoDispatchesMemoryEvents(t *testing.T) {
	f := newStreamFixture(t)
	checkCreateAndDispose[*ImageInfo](t, func() {
		info := NewImageInfo(f.image20x10)
		info.Dispose()
	})
}

func TestMultiFrameCompleterImageCallbackCanRemoveListener(t *testing.T) {
	f := newStreamFixture(t)
	completer := async.NewCompleter[ui.Codec]()
	mockCodec := newMockCodec(1, 0)
	imageStream := NewMultiFrameImageStreamCompleter(completer.Future(), 1.0)
	completer.Complete(mockCodec)
	f.idle()
	if mockCodec.numFramesAsked != 0 {
		t.Error("asked")
	}
	var streamListener ImageStreamListener
	streamListener = NewImageStreamListener(func(image *ImageInfo, synchronousCall bool) {
		f.toDispose = append(f.toDispose, image)
		imageStream.RemoveListener(streamListener)
	})
	imageStream.AddListener(streamListener)
	f.idle()
	if mockCodec.numFramesAsked != 1 {
		t.Error("asked")
	}
	mockCodec.completeNextFrame(fakeFrame(200*time.Millisecond, f.image20x10))
	f.idle()
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
}

func TestImageStreamNeverListenedCanBeDisposed(t *testing.T) {
	f := newStreamFixture(t)
	mockCodec := newMockCodec(2, -1)
	codecCompleter := async.NewCompleter[ui.Codec]()
	imageStream := NewMultiFrameImageStreamCompleter(codecCompleter.Future(), 1.0)
	codecCompleter.Complete(mockCodec)
	f.idle()
	if mockCodec.disposed {
		t.Error("disposed early")
	}
	imageStream.MaybeDispose()
	if !mockCodec.disposed {
		t.Error("not disposed")
	}
}

func TestImageInfoIsCloneOf(t *testing.T) {
	f := newStreamFixture(t)
	image := f.image20x10.Clone()
	defer image.Dispose()
	two := 2.0
	info1 := NewImageInfo(image.Clone())
	info2 := NewImageInfo(image.Clone(), ImageInfoOptions{Scale: &two})
	// These should NOT be considered clones because their scales differ
	if info1.IsCloneOf(info2) {
		t.Error("clones with different scales")
	}
	info3 := NewImageInfo(image.Clone(), ImageInfoOptions{Scale: &two})
	// These should be considered clones because all properties match
	if !info2.IsCloneOf(info3) {
		t.Error("not clones")
	}
	info1.Dispose()
	info2.Dispose()
	info3.Dispose()
}

func TestImageStreamErrorsNotReportedAfterRemovalWhenReportErrorsFalse(t *testing.T) {
	newImageTester(t)
	completer := newFakeEventReportingImageStreamCompleter(nil)
	var reported []*foundation.FlutterErrorDetails
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { reported = append(reported, d) }
	defer func() { foundation.FlutterErrorOnError = old }()
	no := false
	listener := NewImageStreamListener(func(*ImageInfo, bool) {},
		ImageStreamListenerOptions{OnError: func(any, dart.StackTrace) {}, ReportErrors: &no})
	completer.AddListener(listener)
	completer.RemoveListener(listener)
	completer.ReportError(dart.NewException("test error"), ImageStreamCompleterReportErrorOptions{
		Context: foundation.NewErrorDescription("test"), Silent: true})
	if len(reported) != 0 {
		t.Errorf("reported %d", len(reported))
	}
}

func TestImageStreamErrorsReportedAfterListenerRemovalByDefault(t *testing.T) {
	newImageTester(t)
	completer := newFakeEventReportingImageStreamCompleter(nil)
	var reported []*foundation.FlutterErrorDetails
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { reported = append(reported, d) }
	defer func() { foundation.FlutterErrorOnError = old }()
	listener := NewImageStreamListener(func(*ImageInfo, bool) {})
	completer.AddListener(listener)
	completer.RemoveListener(listener)
	completer.ReportError(dart.NewException("test error"), ImageStreamCompleterReportErrorOptions{
		Context: foundation.NewErrorDescription("test"), Silent: true})
	if len(reported) != 1 {
		t.Errorf("reported %d", len(reported))
	}
}
