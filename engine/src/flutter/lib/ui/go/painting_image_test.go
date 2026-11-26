package ui

// Ports of the engine's image tests (Flutter 3.50.1, run as on Impeller,
// whose decoder dart:ui's Go port follows): testing/dart/codec_test.dart,
// image_descriptor_test.dart, image_dispose_test.dart,
// image_events_test.dart, image_resize_test.dart,
// decode_image_from_pixels_sync_test.dart, encoding_test.dart and
// image_test.dart. The fixtures are read from the Flutter clone at
// ~/Projetos/jonathanperis/flutter (the tests skip without it).
//
// Skipped or adapted:
//   - Skia's resources (third_party/skia/resources/images: alphabetAnim.gif,
//     test640x479.gif, baby_tux.png, flutter_logo.jpg) are not in the clone
//     (a gclient dependency): those tests use the engine fixtures instead
//     (hello_loop_2.gif, four_frame_with_reuse.gif, Horizontal.png,
//     DisplayP3Logo.jpg), with the values real Flutter gives for them (the
//     image_codec difftest).
//   - The golden comparisons of PNG bytes compare the decoded pixels: Go's
//     PNG encoder writes other bytes (ENGINE GAP), the pixels are the same.
//   - "HEIC image" (image_descriptor_test): no HEIC decoder (the engine uses
//     macOS's CoreGraphics; ENGINE GAP).
//   - "decodeImageFromPixelsSync throws if not Impeller", "decodeImageFromPixels
//     float32", "toByteData Unmodified format works with grayscale images" and
//     "toByteData PNG format works with simple image" (encoding_test): skipped
//     on Impeller by the Dart tests themselves.
//   - "Animation metadata" checks test640x479's 4 frames/-1 repetitions on
//     four_frame_with_reuse.gif and alphabetAnim's 0 repetitions on
//     opaque_first_frame_then_alpha.gif (no NETSCAPE extension either).

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func imgFlutterDir(t *testing.T, parts ...string) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	return filepath.Join(append([]string{home, "Projetos/jonathanperis/flutter/engine/src/flutter"}, parts...)...)
}

func imgFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(imgFlutterDir(t, "lib/ui/fixtures"), name))
	if err != nil {
		t.Skipf("no Flutter clone: %v", err)
	}
	return b
}

func imgResource(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(imgFlutterDir(t, "testing/resources"), name))
	if err != nil {
		t.Skipf("no Flutter clone: %v", err)
	}
	return b
}

// imgLoop gives the test its own event loop.
func imgLoop(t *testing.T) {
	restore := async.SetCurrent(async.NewLoop())
	t.Cleanup(restore)
}

// imgResult runs the loop until f completes.
func imgResult[T any](f async.Future[T]) (v T, err error) {
	done := false
	async.OnComplete(f, func(x T) { v, done = x, true }, func(e error) { err, done = e, true })
	l := async.Current()
	l.FlushMicrotasks()
	for !done {
		l.WaitPosted()
	}
	return v, err
}

func imgAwait[T any](t *testing.T, f async.Future[T]) T {
	t.Helper()
	v, err := imgResult(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return v
}

func imgCodec(t *testing.T, data []byte, opts ...InstantiateImageCodecOptions) Codec {
	t.Helper()
	return imgAwait(t, InstantiateImageCodec(data, opts...))
}

func imgBytes(t *testing.T, img *Image, f ImageByteFormat) []byte {
	t.Helper()
	return imgAwait(t, img.ToByteData(ImageToByteDataOptions{Format: f}))
}

func imgUint32BE(b []byte, i int) uint32 { return binary.BigEndian.Uint32(b[i:]) }

func ptr(v int) *int { return &v }

// --- codec_test.dart ---

func TestCodecAnimationMetadata(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "opaque_first_frame_then_alpha.gif"))
	if codec.FrameCount() != 2 || codec.RepetitionCount() != 0 {
		t.Errorf("got %d frames, %d repetitions", codec.FrameCount(), codec.RepetitionCount())
	}
	codec.Dispose()
	codec = imgCodec(t, imgFixture(t, "four_frame_with_reuse.gif"))
	if codec.FrameCount() != 4 || codec.RepetitionCount() != -1 {
		t.Errorf("got %d frames, %d repetitions", codec.FrameCount(), codec.RepetitionCount())
	}
	codec = imgCodec(t, imgFixture(t, "hello_loop_2.gif"))
	if codec.FrameCount() != 20 || codec.RepetitionCount() != 1 {
		t.Errorf("got %d frames, %d repetitions", codec.FrameCount(), codec.RepetitionCount())
	}
}

func TestCodecFailsWithInvalidData(t *testing.T) {
	imgLoop(t)
	_, err := imgResult(InstantiateImageCodec([]byte{1, 2, 3}))
	if err == nil || !strings.Contains(err.Error(), "Invalid image data") {
		t.Errorf("got %v", err)
	}
}

func TestCodecGetNextFrameFailsWithInvalidData(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "four_frame_with_reuse.gif")[:1200])
	_, err := imgResult(codec.GetNextFrame())
	if err == nil || !strings.Contains(err.Error(), "Could not decompress image.") {
		t.Errorf("got %v", err)
	}
	codec.Dispose()
}

func imgFrameInfos(t *testing.T, codec Codec, n int) [][3]int {
	var out [][3]int
	for i := 0; i < n; i++ {
		fi := imgAwait(t, codec.GetNextFrame())
		out = append(out, [3]int{int(fi.Duration / time.Millisecond), fi.Image.Width, fi.Image.Height})
	}
	return out
}

func TestCodecNextFrame(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "hello_loop_2.gif"))
	got := imgFrameInfos(t, codec, 5)
	codec.Dispose()
	for _, f := range got {
		if f != [3]int{100, 640, 88} {
			t.Errorf("got %v", got)
		}
	}
}

func TestCodecNonAnimatedImage(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "Horizontal.png"))
	got := imgFrameInfos(t, codec, 2)
	codec.Dispose()
	if got[0] != [3]int{0, 300, 100} || got[1] != [3]int{0, 300, 100} {
		t.Errorf("got %v", got)
	}
}

func TestCodecWithSize(t *testing.T) {
	imgLoop(t)
	buffer := imgAwait(t, ImmutableBufferFromUint8List(imgFixture(t, "Horizontal.png")))
	codec := imgAwait(t, InstantiateImageCodecWithSize(buffer, InstantiateImageCodecWithSizeOptions{
		GetTargetSize: func(w, h int) TargetImageSize {
			return NewTargetImageSize(TargetImageSize{Width: ptr(w / 2), Height: ptr(h / 2)})
		},
	}))
	got := imgFrameInfos(t, codec, 2)
	codec.Dispose()
	if got[0] != [3]int{0, 150, 50} || got[1] != [3]int{0, 150, 50} {
		t.Errorf("got %v", got)
	}
	if !buffer.DebugDisposed() {
		t.Error("instantiateImageCodecWithSize didn't dispose the buffer")
	}
}

func TestCodecDisposedDecodedImage(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "DisplayP3Logo.jpg"))
	fi := imgAwait(t, codec.GetNextFrame())
	fi.Image.Dispose()
	_, err := imgResult(codec.GetNextFrame())
	if err == nil || !strings.Contains(err.Error(), "Decoded image has been disposed") {
		t.Errorf("got %v", err)
	}
	codec.Dispose()
}

// imgGoldenPixels decodes a golden PNG's straight RGBA.
func imgGoldenPixels(t *testing.T, name string) []byte {
	codec := imgCodec(t, imgFixture(t, name))
	fi := imgAwait(t, codec.GetNextFrame())
	return imgBytes(t, fi.Image, ImageByteFormatRawStraightRgba)
}

// imgPNGPixels encodes an image to PNG and decodes it back.
func imgPNGPixels(t *testing.T, img *Image) []byte {
	png := imgBytes(t, img, ImageByteFormatPng)
	codec := imgCodec(t, png)
	fi := imgAwait(t, codec.GetNextFrame())
	return imgBytes(t, fi.Image, ImageByteFormatRawStraightRgba)
}

func TestCodecAnimatedGifReusesFrames(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "four_frame_with_reuse.gif"))
	var fi *FrameInfo
	for i := 0; i < 4; i++ {
		fi = imgAwait(t, codec.GetNextFrame())
	}
	codec.Dispose()
	if !bytes.Equal(imgPNGPixels(t, fi.Image), imgGoldenPixels(t, "impeller_four_frame_with_reuse_end.png")) {
		t.Error("last frame differs from the golden")
	}
}

func TestCodecAnimatedWebpAlphaAfterOpaqueFirstFrame(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "opaque_first_frame_then_alpha.webp"))
	b := imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, 0); got != 0xFF0000FF {
		t.Errorf("first frame %08x", got)
	}
	b = imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, 0); got != 0x00008080 {
		t.Errorf("second frame %08x", got)
	}
	codec.Dispose()
}

func TestCodecAnimatedGifAlphaAfterOpaqueFirstFrame(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "opaque_first_frame_then_alpha.gif"))
	b := imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, 0); got != 0xFF0000FF {
		t.Errorf("first frame %08x", got)
	}
	b = imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if imgUint32BE(b, 0) != 0x0000FFFF || imgUint32BE(b, 4) != 0 {
		t.Errorf("second frame %08x %08x", imgUint32BE(b, 0), imgUint32BE(b, 4))
	}
	codec.Dispose()
}

func TestCodecAnimatedWebpReusesFrames(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "heart.webp"))
	var fi *FrameInfo
	for i := 0; i < 69; i++ {
		fi = imgAwait(t, codec.GetNextFrame())
	}
	codec.Dispose()
	if !bytes.Equal(imgPNGPixels(t, fi.Image), imgGoldenPixels(t, "impeller_heart_end.png")) {
		t.Error("last frame differs from the golden")
	}
}

func TestCodecAnimatedApngReusesPrePreFrame(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "2_dispose_op_restore_previous.apng"))
	for i := 0; i < 70; i++ {
		fi := imgAwait(t, codec.GetNextFrame())
		if i >= 67 {
			golden := "impeller_2_dispose_op_restore_previous.apng." + dart.ToString(i) + ".png"
			if !bytes.Equal(imgPNGPixels(t, fi.Image), imgGoldenPixels(t, golden)) {
				t.Errorf("frame %d differs from %s", i, golden)
			}
		}
	}
	codec.Dispose()
}

func TestCodecAnimatedApngAlphaType(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "alpha_animated.apng"))
	b := imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, 0); got != 0x33000033 {
		t.Errorf("first frame %08x", got)
	}
	b = imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, 0); got != 0x99000099 {
		t.Errorf("second frame %08x", got)
	}
	codec.Dispose()
}

func TestCodecAnimatedApngBackgroundColorRestore(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "dispose_op_background.apng"))
	b := imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, 0); got != 0xFF0000FF {
		t.Errorf("first frame %08x", got)
	}
	imgAwait(t, codec.GetNextFrame())
	b = imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, len(b)-4); got != 0x0000FFFF {
		t.Errorf("third frame %08x", got)
	}
	b = imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	if got := imgUint32BE(b, len(b)-4); got != 0 {
		t.Errorf("fourth frame %08x", got)
	}
	codec.Dispose()
}

func TestCodecAnimatedApngInvalidDestinationRegion(t *testing.T) {
	for _, f := range []string{"out_of_bounds.apng", "out_of_bounds_wrapping.apng"} {
		t.Run(f, func(t *testing.T) {
			imgLoop(t)
			codec := imgCodec(t, imgFixture(t, f))
			_, err := imgResult(codec.GetNextFrame())
			if err == nil || !strings.Contains(err.Error(), "Could not decompress image.") {
				t.Errorf("got %v", err)
			}
			codec.Dispose()
		})
	}
}

// --- image_descriptor_test.dart ---

func imgDescriptor(t *testing.T, data []byte) ImageDescriptor {
	buffer := imgAwait(t, ImmutableBufferFromUint8List(data))
	return imgAwait(t, ImageDescriptorEncoded(buffer))
}

func TestImageDescriptorEncoded(t *testing.T) {
	imgLoop(t)
	for _, c := range []struct {
		name          string
		data          []byte
		w, h, bpp, fc int
	}{
		{"greyscale", imgResource(t, "2x2.png"), 2, 2, 1, 1},
		{"square", imgResource(t, "square.png"), 10, 10, 4, 1},
		{"animated", imgFixture(t, "four_frame_with_reuse.gif"), 50, 50, 4, 4},
	} {
		d := imgDescriptor(t, c.data)
		if d.Width() != c.w || d.Height() != c.h || d.BytesPerPixel() != c.bpp {
			t.Errorf("%s: %dx%dx%d", c.name, d.Width(), d.Height(), d.BytesPerPixel())
		}
		codec := imgAwait(t, d.InstantiateCodec())
		if codec.FrameCount() != c.fc {
			t.Errorf("%s: %d frames", c.name, codec.FrameCount())
		}
		if c.name == "animated" && codec.RepetitionCount() != -1 {
			t.Errorf("animated: %d repetitions", codec.RepetitionCount())
		}
		codec.Dispose()
	}
}

func TestImageDescriptorRaw(t *testing.T) {
	imgLoop(t)
	pixels := bytes.Repeat([]byte{0xEF}, 16)
	buffer := imgAwait(t, ImmutableBufferFromUint8List(pixels))
	d := NewImageDescriptorRaw(buffer, 4, 4, PixelFormatRgba8888, ImageDescriptorRawOptions{RowBytes: ptr(16)})
	if d.Width() != 4 || d.Height() != 4 || d.BytesPerPixel() != 4 {
		t.Errorf("%dx%dx%d", d.Width(), d.Height(), d.BytesPerPixel())
	}
	codec := imgAwait(t, d.InstantiateCodec())
	if codec.FrameCount() != 1 {
		t.Errorf("%d frames", codec.FrameCount())
	}
	codec.Dispose()
}

// The size rules of ImageDescriptor.instantiateCodec and the toString of a
// descriptor whose size hasn't been read.
func TestImageDescriptorTargetSizes(t *testing.T) {
	imgLoop(t)
	d := imgDescriptor(t, imgFixture(t, "Horizontal.png"))
	if s := d.(interface{ String() string }).String(); s != "ImageDescriptor(width: ?, height: ?, bytes per pixel: ?)" {
		t.Errorf("toString %q", s)
	}
	for _, c := range []struct {
		w, h         *int
		wantW, wantH int
	}{
		{ptr(7), nil, 7, 2}, {nil, ptr(7), 21, 7}, {ptr(301), nil, 301, 100}, {nil, ptr(101), 303, 101},
		{ptr(-1), ptr(5), 15, 5}, {ptr(0), ptr(0), 300, 100}, {ptr(10), ptr(10), 10, 10},
	} {
		codec := imgAwait(t, d.InstantiateCodec(ImageDescriptorInstantiateCodecOptions{TargetWidth: c.w, TargetHeight: c.h}))
		fi := imgAwait(t, codec.GetNextFrame())
		if fi.Image.Width != c.wantW || fi.Image.Height != c.wantH {
			t.Errorf("%v %v: %dx%d, want %dx%d", c.w, c.h, fi.Image.Width, fi.Image.Height, c.wantW, c.wantH)
		}
	}
	if s := d.(interface{ String() string }).String(); s != "ImageDescriptor(width: 300, height: 100, bytes per pixel: ?)" {
		t.Errorf("toString %q", s)
	}
}

func TestImageDescriptorInvalid(t *testing.T) {
	imgLoop(t)
	buffer := imgAwait(t, ImmutableBufferFromUint8List([]byte{1, 2, 3}))
	r := panics(func() { ImageDescriptorEncoded(buffer) })
	if e, ok := r.(dart.Exception); !ok || e.Error() != "Exception: Invalid image data" {
		t.Errorf("got %v", r)
	}
}

// --- image_dispose_test.dart ---

func imgFrame(t *testing.T, data []byte) *FrameInfo {
	codec := imgCodec(t, data)
	fi := imgAwait(t, codec.GetNextFrame())
	codec.Dispose()
	return fi
}

func TestImageHandlesAreDistinct(t *testing.T) {
	imgLoop(t)
	frame := imgFrame(t, imgResource(t, "2x2.png"))
	if frame.Image.Width != 2 || frame.Image.Height != 2 {
		t.Fatalf("%dx%d", frame.Image.Width, frame.Image.Height)
	}
	handle1 := frame.Image.Clone()
	if handle1.Width != 2 || handle1.Height != 2 {
		t.Errorf("clone %dx%d", handle1.Width, handle1.Height)
	}
	handle2 := handle1.Clone()
	if handle1 == handle2 || handle1 == frame.Image {
		t.Error("handles are not distinct")
	}
	frame.Image.Dispose()
}

func TestImageCanvasPaintsImageFromHandle(t *testing.T) {
	needGPU(t)
	imgLoop(t)
	frame := imgFrame(t, imgResource(t, "2x2.png"))
	handle1 := frame.Image.Clone()
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	rect := NewRectFromLTRB(0, 0, 2, 2)
	src := BlendModeSrc
	c.DrawImage(handle1, NewOffset(0, 0), NewPaint())
	c.DrawImageRect(handle1, rect, rect, NewPaint())
	c.DrawImageNine(handle1, rect, rect, NewPaint())
	c.DrawAtlas(handle1, nil, nil, nil, &src, &rect, NewPaint())
	c.DrawRawAtlas(handle1, []float32{}, []float32{}, []int32{}, &src, &rect, NewPaint())
	picture := rec.EndRecording()
	rasterized := imgAwait(t, picture.ToImage(2, 2))
	source := imgBytes(t, frame.Image, ImageByteFormatRawRgba)
	handle := imgBytes(t, handle1, ImageByteFormatRawRgba)
	raster := imgBytes(t, rasterized, ImageByteFormatRawRgba)
	if !bytes.Equal(source, handle) || !bytes.Equal(source, raster) {
		t.Errorf("source %v, handle %v, rasterized %v", source, handle, raster)
	}
}

func TestImageRecordsStackTraces(t *testing.T) {
	imgLoop(t)
	frame := imgFrame(t, imgResource(t, "2x2.png"))
	handle1 := frame.Image.Clone()
	handle2 := handle1.Clone()
	same := func(a, b []dart.StackTrace) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	st := frame.Image.DebugGetOpenHandleStackTraces()
	if len(st) != 3 || !same(st, handle1.DebugGetOpenHandleStackTraces()) {
		t.Errorf("%d stacks", len(st))
	}
	handle1.Dispose()
	st = frame.Image.DebugGetOpenHandleStackTraces()
	if len(st) != 2 || !same(st, handle2.DebugGetOpenHandleStackTraces()) {
		t.Errorf("%d stacks", len(st))
	}
	handle2.Dispose()
	if st = frame.Image.DebugGetOpenHandleStackTraces(); len(st) != 1 {
		t.Errorf("%d stacks", len(st))
	}
	frame.Image.Dispose()
	if st = frame.Image.DebugGetOpenHandleStackTraces(); len(st) != 0 {
		t.Errorf("%d stacks", len(st))
	}
}

func TestImageClonesCanBeCompared(t *testing.T) {
	imgLoop(t)
	data := imgResource(t, "2x2.png")
	frame := imgFrame(t, data)
	handle1 := frame.Image.Clone()
	handle2 := handle1.Clone()
	check := func() {
		if !handle1.IsCloneOf(handle2) || !handle2.IsCloneOf(handle1) || !handle1.IsCloneOf(frame.Image) {
			t.Error("clones aren't clones")
		}
	}
	check()
	handle1.Dispose()
	check()
	frame2 := imgFrame(t, data)
	if frame2.Image.IsCloneOf(frame.Image) {
		t.Error("images of two codecs are clones")
	}
}

func TestImageDebugDisposed(t *testing.T) {
	imgLoop(t)
	frame := imgFrame(t, imgResource(t, "2x2.png"))
	if frame.Image.DebugDisposed() {
		t.Error("disposed before dispose")
	}
	frame.Image.Dispose()
	if !frame.Image.DebugDisposed() {
		t.Error("not disposed after dispose")
	}
}

func TestImageCloneOfDisposedThrows(t *testing.T) {
	imgLoop(t)
	frame := imgFrame(t, imgResource(t, "2x2.png"))
	keep := frame.Image.Clone()
	frame.Image.Dispose()
	r := panics(func() { frame.Image.Clone() })
	if e, ok := r.(dart.StateError); !ok || !strings.HasPrefix(e.Message, "Cannot clone a disposed image.") {
		t.Errorf("got %v", r)
	}
	if s := keep.String(); s != "[2×2]" {
		t.Errorf("toString %q", s)
	}
	keep.Dispose()
}

// --- image_events_test.dart and image_test.dart ---

func imgCreatePicture() *Picture {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	c.ClipRect(NewRectFromLTWH(0, 0, 100, 100))
	return rec.EndRecording()
}

func TestImageToImageSucceeds(t *testing.T) {
	needGPU(t)
	imgLoop(t)
	image := imgAwait(t, imgCreatePicture().ToImage(10, 10))
	if image.Width != 10 || image.Height != 10 {
		t.Errorf("%dx%d", image.Width, image.Height)
	}
	image.Dispose()
}

func TestImageOnCreateAndDispose(t *testing.T) {
	needGPU(t)
	imgLoop(t)
	saveCreate, saveDispose := ImageOnCreate, ImageOnDispose
	t.Cleanup(func() { ImageOnCreate, ImageOnDispose = saveCreate, saveDispose })
	var created, disposed *Image
	createCount, disposeCount := 0, 0
	ImageOnCreate = func(i *Image) { createCount++; created = i }
	ImageOnDispose = func(i *Image) { disposeCount++; disposed = i }
	for n := 1; n <= 2; n++ {
		image := imgAwait(t, imgCreatePicture().ToImage(10, 10))
		image.Dispose()
		if createCount != n || created != image || disposeCount != n || disposed != image {
			t.Errorf("round %d: %d creates, %d disposes", n, createCount, disposeCount)
		}
	}
}

// Clones are handles too: each calls onCreate (Image._ does).
func TestImageCloneCallsOnCreate(t *testing.T) {
	imgLoop(t)
	frame := imgFrame(t, imgResource(t, "2x2.png"))
	saveCreate := ImageOnCreate
	t.Cleanup(func() { ImageOnCreate = saveCreate })
	var created *Image
	ImageOnCreate = func(i *Image) { created = i }
	c := frame.Image.Clone()
	if created != c {
		t.Error("clone didn't call onCreate")
	}
}

// --- image_resize_test.dart ---

func imgSize(t *testing.T, opts InstantiateImageCodecOptions) (int, int) {
	codec := imgCodec(t, imgCodecData(t), opts)
	fi := imgAwait(t, codec.GetNextFrame())
	codec.Dispose()
	return fi.Image.Width, fi.Image.Height
}

func imgCodecData(t *testing.T) []byte { return imgResource(t, "2x2.png") }

func TestImageResizeEncoded(t *testing.T) {
	imgLoop(t)
	for _, c := range []struct {
		name         string
		opts         InstantiateImageCodecOptions
		wantW, wantH int
	}{
		{"no resize by default", InstantiateImageCodecOptions{}, 2, 2},
		{"resize width with constrained height", InstantiateImageCodecOptions{TargetHeight: ptr(1)}, 1, 1},
		{"resize height with constrained width", InstantiateImageCodecOptions{TargetWidth: ptr(1)}, 1, 1},
		{"upscale image by 5x", InstantiateImageCodecOptions{TargetWidth: ptr(10)}, 10, 10},
		{"upscale image by 5x - no upscaling", InstantiateImageCodecOptions{TargetWidth: ptr(10), DisallowUpscaling: true}, 2, 2},
		{"upscale image varying width and height", InstantiateImageCodecOptions{TargetWidth: ptr(10), TargetHeight: ptr(1)}, 10, 1},
		{"upscale image varying width and height - no upscaling", InstantiateImageCodecOptions{TargetWidth: ptr(10), TargetHeight: ptr(1), DisallowUpscaling: true}, 2, 1},
	} {
		if w, h := imgSize(t, c.opts); w != c.wantW || h != c.wantH {
			t.Errorf("%s: %dx%d", c.name, w, h)
		}
	}
}

func imgBlackSquare(t *testing.T, opts DecodeImageFromPixelsOptions) *Image {
	c := async.NewCompleter[*Image]()
	DecodeImageFromPixels(make([]byte, 2*2*4), 2, 2, PixelFormatRgba8888, func(i *Image) { c.Complete(i) }, opts)
	return imgAwait(t, c.Future())
}

func TestImageResizePixels(t *testing.T) {
	imgLoop(t)
	for _, c := range []struct {
		name         string
		opts         DecodeImageFromPixelsOptions
		wantW, wantH int
	}{
		{"no resize by default", DecodeImageFromPixelsOptions{DisallowUpscaling: true}, 2, 2},
		{"resize width with constrained height", DecodeImageFromPixelsOptions{TargetHeight: ptr(1), DisallowUpscaling: true}, 1, 1},
		{"resize height with constrained width", DecodeImageFromPixelsOptions{TargetWidth: ptr(1), DisallowUpscaling: true}, 1, 1},
		{"upscale image by 5x", DecodeImageFromPixelsOptions{TargetWidth: ptr(10)}, 10, 10},
		{"upscale image varying width and height", DecodeImageFromPixelsOptions{TargetHeight: ptr(1), TargetWidth: ptr(10)}, 10, 1},
		{"large negative dimensions", DecodeImageFromPixelsOptions{TargetHeight: ptr(-100), TargetWidth: ptr(-99999), DisallowUpscaling: true}, 2, 2},
	} {
		img := imgBlackSquare(t, c.opts)
		if img.Width != c.wantW || img.Height != c.wantH {
			t.Errorf("%s: %dx%d", c.name, img.Width, img.Height)
		}
	}
	for _, opts := range []DecodeImageFromPixelsOptions{
		{TargetHeight: ptr(10), DisallowUpscaling: true},
		{TargetHeight: ptr(10), TargetWidth: ptr(1), DisallowUpscaling: true},
	} {
		r := panics(func() {
			DecodeImageFromPixels(make([]byte, 16), 2, 2, PixelFormatRgba8888, func(*Image) {}, opts)
		})
		if _, ok := r.(*dart.AssertionError); !ok {
			t.Errorf("no upscaling: got %v, want an AssertionError", r)
		}
	}
}

// --- decode_image_from_pixels_sync_test.dart ---

func TestDecodeImageFromPixelsSyncRGBA8888(t *testing.T) {
	imgLoop(t)
	pixels := bytes.Repeat([]byte{255, 0, 0, 255}, 4)
	image := DecodeImageFromPixelsSync(pixels, 2, 2, PixelFormatRgba8888)
	if image.Width != 2 || image.Height != 2 {
		t.Errorf("%dx%d", image.Width, image.Height)
	}
	if got := imgBytes(t, image, ImageByteFormatRawRgba); !bytes.Equal(got, pixels) {
		t.Errorf("got %v", got)
	}
	image.Dispose()
}

func TestDecodeImageFromPixelsSyncThrows(t *testing.T) {
	imgLoop(t)
	r := panics(func() { DecodeImageFromPixelsSync(make([]byte, 4), 0, 1, PixelFormatRgba8888) })
	if s, ok := r.(string); !ok || s != "Image dimensions must be greater than zero." {
		t.Errorf("got %#v", r)
	}
	r = panics(func() { DecodeImageFromPixelsSync(make([]byte, 4), 1, 1, PixelFormatRFloat32) })
	if s, ok := r.(string); !ok || s != "Unsupported pixel format." {
		t.Errorf("got %#v", r)
	}
}

// --- encoding_test.dart ---

const imgKWidth, imgKRadius = 10, 2

func imgSquare4x4(t *testing.T) *Image {
	rec := NewPictureRecorder()
	c := NewCanvas(rec, NewRectFromLTWH(0, 0, imgKWidth, imgKWidth))
	black := NewPaint()
	black.SetStrokeWidth(1)
	black.SetColor(NewColor(0xFF000000))
	green := NewPaint()
	green.SetStrokeWidth(1)
	green.SetColor(NewColor(0xFF00FF00))
	c.DrawRect(NewRectFromLTWH(0, 0, imgKWidth, imgKWidth), black)
	inner := float64(imgKWidth - 2*imgKRadius)
	c.DrawRect(NewRectFromLTWH(imgKRadius, imgKRadius, inner, inner), green)
	return imgAwait(t, rec.EndRecording().ToImage(imgKWidth, imgKWidth))
}

func imgSquare4x4Bytes() []byte {
	out := make([]byte, imgKWidth*imgKWidth*4)
	for i := 0; i < imgKWidth*imgKWidth; i++ {
		x, y := i%imgKWidth, i/imgKWidth
		if x >= imgKRadius && x < imgKWidth-imgKRadius && y >= imgKRadius && y < imgKWidth-imgKRadius {
			copy(out[4*i:], []byte{0, 255, 0, 255})
		} else {
			copy(out[4*i:], []byte{0, 0, 0, 255})
		}
	}
	return out
}

func TestEncodingRGBASimpleImage(t *testing.T) {
	needGPU(t)
	imgLoop(t)
	image := imgSquare4x4(t)
	if got := imgBytes(t, image, ImageByteFormatRawRgba); !bytes.Equal(got, imgSquare4x4Bytes()) {
		t.Errorf("got %v", got)
	}
	if got := imgBytes(t, image, ImageByteFormatRawUnmodified); !bytes.Equal(got, imgSquare4x4Bytes()) {
		t.Errorf("unmodified: got %v", got)
	}
}

func imgFromList(t *testing.T, data []byte) *Image {
	c := async.NewCompleter[*Image]()
	DecodeImageFromList(data, func(i *Image) { c.Complete(i) })
	return imgAwait(t, c.Future())
}

func TestEncodingRGBAConvertsGrayscale(t *testing.T) {
	imgLoop(t)
	got := imgBytes(t, imgFromList(t, imgResource(t, "2x2.png")), ImageByteFormatRawRgba)
	want := []byte{255, 255, 255, 255, 127, 127, 127, 255, 127, 127, 127, 255, 0, 0, 0, 255}
	if !bytes.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestEncodingTransparentImage(t *testing.T) {
	imgLoop(t)
	image := imgFromList(t, imgResource(t, "transparent_image.png"))
	premul := []byte{
		255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 136, 136, 136, 255,
		127, 0, 0, 127, 0, 127, 0, 127, 0, 0, 127, 127, 67, 67, 67, 127,
		63, 0, 0, 63, 0, 63, 0, 63, 0, 0, 63, 63, 33, 33, 33, 63,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	straight := []byte{
		255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 136, 136, 136, 255,
		255, 0, 0, 127, 0, 255, 0, 127, 0, 0, 255, 127, 135, 135, 135, 127,
		255, 0, 0, 63, 0, 255, 0, 63, 0, 0, 255, 63, 134, 134, 134, 63,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	if got := imgBytes(t, image, ImageByteFormatRawRgba); !bytes.Equal(got, premul) {
		t.Errorf("rawRgba: got %v", got)
	}
	if got := imgBytes(t, image, ImageByteFormatRawStraightRgba); !bytes.Equal(got, straight) {
		t.Errorf("rawStraightRgba: got %v", got)
	}
}

func TestEncodingExtendedRGBA128(t *testing.T) {
	needGPU(t)
	imgLoop(t)
	image := imgSquare4x4(t)
	data := imgBytes(t, image, ImageByteFormatRawExtendedRgba128)
	if image.Width != imgKWidth || image.Height != imgKWidth || len(data) != imgKWidth*imgKWidth*16 {
		t.Fatalf("%dx%d, %d bytes", image.Width, image.Height, len(data))
	}
	for i, want := range []float32{0, 0, 0, 1} {
		if got := math.Float32frombits(binary.LittleEndian.Uint32(data[4*i:])); got != want {
			t.Errorf("float %d: %v", i, got)
		}
	}
	if image.ColorSpace() != ColorSpaceSRGB {
		t.Errorf("color space %v", image.ColorSpace())
	}
}

// --- engine behavior the Dart tests don't reach directly ---

func TestImmutableBufferFromAssetAndFile(t *testing.T) {
	imgLoop(t)
	save := EngineAssetLoader
	t.Cleanup(func() { EngineAssetLoader = save })
	var asked string
	EngineAssetLoader = func(key string) ([]byte, error) {
		asked = key
		if key == "a%20b.png" {
			return []byte{1, 2, 3}, nil
		}
		return nil, os.ErrNotExist
	}
	b := imgAwait(t, ImmutableBufferFromAsset("a b.png"))
	if b.Length() != 3 || asked != "a%20b.png" {
		t.Errorf("length %d, key %q", b.Length(), asked)
	}
	b.Dispose()
	if !b.DebugDisposed() {
		t.Error("not disposed")
	}
	if _, err := imgResult(ImmutableBufferFromAsset("missing")); err == nil || err.Error() != "Exception: Asset not found" {
		t.Errorf("got %v", err)
	}
	if _, err := imgResult(ImmutableBufferFromFilePath("/nonexistent")); err == nil || err.Error() != "Exception: Could not load file at /nonexistent." {
		t.Errorf("got %v", err)
	}
	if got := encodeAssetKey("x/é#?%"); got != "x/%C3%A9%23%3F%25" {
		t.Errorf("encoded key %q", got)
	}
}

func TestTargetImageSizeString(t *testing.T) {
	if s := (TargetImageSize{}).String(); s != "TargetImageSize(null x null)" {
		t.Error(s)
	}
	if s := NewTargetImageSize(TargetImageSize{Width: ptr(3), Height: ptr(4)}).String(); s != "TargetImageSize(3 x 4)" {
		t.Error(s)
	}
	if _, ok := panics(func() { NewTargetImageSize(TargetImageSize{Width: ptr(0)}) }).(*dart.AssertionError); !ok {
		t.Error("width 0 doesn't assert")
	}
}

func TestCodecToString(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "four_frame_with_reuse.gif"))
	s := codec.(interface{ String() string })
	if s.String() != "Codec()" {
		t.Error(s.String())
	}
	codec.FrameCount()
	if s.String() != "Codec(4 frames)" {
		t.Error(s.String())
	}
}

// A multi-frame codec's frames wrap around, and a disposed codec never
// delivers the frames asked for after its disposal.
func TestCodecFramesWrapAround(t *testing.T) {
	imgLoop(t)
	codec := imgCodec(t, imgFixture(t, "alpha_animated.apng"))
	var firsts []uint32
	for i := 0; i < 3; i++ {
		fi := imgAwait(t, codec.GetNextFrame())
		firsts = append(firsts, imgUint32BE(imgBytes(t, fi.Image, ImageByteFormatRawRgba), 0))
	}
	if firsts[0] != 0x33000033 || firsts[1] != 0x99000099 || firsts[2] != 0x33000033 {
		t.Errorf("got %08x", firsts)
	}
}

func TestHalfFloatConversions(t *testing.T) {
	for _, c := range []struct {
		f float32
		h uint16
	}{
		{0, 0}, {1, 0x3c00}, {-2, 0xc000}, {0.5, 0x3800}, {65504, 0x7bff}, {65520, 0x7c00},
		{float32(math.Inf(-1)), 0xfc00}, {5.960464477539063e-08, 0x0001}, {2.98e-08, 0},
		{0.3, 0x34cd}, {0.21000001, 0x32b8}, {1 + 1.0/2048, 0x3c00}, {1 + 3.0/2048, 0x3c02},
	} {
		if got := float32ToHalf(c.f); got != c.h {
			t.Errorf("float32ToHalf(%v) = %04x, want %04x", c.f, got, c.h)
		}
		if c.h&0x7c00 != 0x7c00 {
			if back := float32ToHalf(halfToFloat32(c.h)); back != c.h {
				t.Errorf("round trip %04x = %04x", c.h, back)
			}
		}
	}
}
