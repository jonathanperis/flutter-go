package ui

// Port of painting.dart's codecs (Flutter 3.50.1): Codec and the native
// codecs (single_frame_codec.cc, multi_frame_codec.cc), instantiateImageCodec
// and its variants, TargetImageSize, decodeImageFromList,
// decodeImageFromPixels(Sync), ImmutableBuffer (immutable_buffer.cc) and
// ImageDescriptor (image_descriptor.cc), with Impeller's decoder
// (image_decoder_impeller.cc). The image formats are decoded in Go
// (painting_image_decoders.go) where the engine uses Skia's SkCodec.

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// ---------------------------------------------------------------------------
// ImmutableBuffer

// ImmutableBuffer is a read-only byte buffer managed by the engine. Call
// Dispose when it is no longer needed.
type ImmutableBuffer struct {
	data          []byte
	length        int
	debugDisposed bool
}

// ImmutableBufferFromUint8List copies list into a new buffer.
func ImmutableBufferFromUint8List(list []byte) async.Future[*ImmutableBuffer] {
	instance := &ImmutableBuffer{length: len(list)}
	f := futurize(func(callback func(async.Void, bool)) string {
		instance.data = append([]byte{}, list...)
		callback(async.Void{}, true)
		return ""
	})
	return async.Then(f, func(async.Void) *ImmutableBuffer { return instance })
}

// ImmutableBufferFromAsset loads the asset with the given key through
// EngineAssetLoader; the future fails with Exception('Asset not found') when
// there is none. The key is URI-encoded as the flutter tool encodes asset
// keys (a space becomes %20).
func ImmutableBufferFromAsset(assetKey string) async.Future[*ImmutableBuffer] {
	encodedKey := encodeAssetKey(assetKey)
	instance := &ImmutableBuffer{}
	f := futurize(func(callback func(int, bool)) string {
		loop := async.Current()
		loader := EngineAssetLoader
		go func() {
			var data []byte
			ok := false
			if loader != nil {
				if d, err := loader(encodedKey); err == nil {
					data, ok = append([]byte{}, d...), true
				}
			}
			loop.Post(func() {
				if !ok {
					callback(-1, true)
					return
				}
				instance.data = data
				callback(len(data), true)
			})
		}()
		return ""
	})
	return async.Then(f, func(length int) *ImmutableBuffer {
		if length == -1 {
			panic(dart.NewException("Asset not found"))
		}
		instance.length = length
		return instance
	})
}

// ImmutableBufferFromFilePath reads the file at path; the future fails with
// Exception('Could not load file at $path.') when it can't be read.
func ImmutableBufferFromFilePath(path string) async.Future[*ImmutableBuffer] {
	instance := &ImmutableBuffer{}
	f := futurize(func(callback func(int, bool)) string {
		loop := async.Current()
		go func() {
			data, err := os.ReadFile(path)
			loop.Post(func() {
				if err != nil {
					callback(-1, true)
					return
				}
				instance.data = data
				callback(len(data), true)
			})
		}()
		return ""
	})
	return async.Then(f, func(length int) *ImmutableBuffer {
		if length == -1 {
			panic(dart.NewException("Could not load file at " + path + "."))
		}
		instance.length = length
		return instance
	})
}

// encodeAssetKey is Uri(path: Uri.encodeFull(assetKey)).path.
func encodeAssetKey(key string) string {
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
			c != 0 && c < 0x80 && contains("!$&'()*+,-./:;=@_~", c) {
			out = append(out, c)
		} else {
			out = append(out, '%', hex[c>>4], hex[c&15])
		}
	}
	return string(out)
}

func contains(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// Length is the length of the data in bytes.
func (b *ImmutableBuffer) Length() int { return b.length }

// DebugDisposed is whether Dispose has been called (asserts only).
func (b *ImmutableBuffer) DebugDisposed() bool {
	if !debugMode {
		panic(dart.StateError{Message: "ImmutableBuffer.debugDisposed is only available when asserts are enabled."})
	}
	return b.debugDisposed
}

// Dispose releases the buffer. Its memory stays alive while an
// ImageDescriptor created from it does.
func (b *ImmutableBuffer) Dispose() {
	if debugMode {
		dartAssert(!b.debugDisposed, "")
		b.debugDisposed = true
	}
	b.data = nil
}

// ---------------------------------------------------------------------------
// ImageDescriptor

// ImageDescriptor describes image data that a Codec can decode: its size
// and bytes per pixel, before decoding.
type ImageDescriptor interface {
	Width() int
	Height() int
	BytesPerPixel() int
	Dispose()
	InstantiateCodec(opts ...ImageDescriptorInstantiateCodecOptions) async.Future[Codec]
}

// ImageDescriptorInstantiateCodecOptions are ImageDescriptor.instantiateCodec's
// named arguments (nil: not given; a size <= 0 counts as not given).
type ImageDescriptorInstantiateCodecOptions struct {
	TargetWidth, TargetHeight *int
	TargetFormat              TargetPixelFormat
}

// ImageDescriptorRawOptions are ImageDescriptor.raw's optional arguments.
type ImageDescriptorRawOptions struct {
	RowBytes *int // default width * bytes per pixel
}

// descFormat is ImageDescriptor::PixelFormat.
type descFormat int

const (
	descUnknown descFormat = iota
	descRGBA8888
	descBGRA8888
	descRGBAFloat32
	descR32Float
	descGray8
)

// alphaType is SkAlphaType.
type alphaType int

const (
	alphaOpaque alphaType = iota
	alphaPremul
	alphaUnpremul
)

type nativeImageDescriptor struct {
	data     []byte
	gen      imageGenerator
	width    int
	height   int
	format   descFormat
	alpha    alphaType
	rowBytes int // -1: default

	cachedWidth, cachedHeight, cachedBPP *int
}

// NewImageDescriptorRaw is ImageDescriptor.raw: pixels of the given
// format, rows of rowBytes bytes (the required named arguments width,
// height and pixelFormat are positional).
func NewImageDescriptorRaw(buffer *ImmutableBuffer, width, height int, pixelFormat PixelFormat, opts ...ImageDescriptorRawOptions) ImageDescriptor {
	rowBytes := -1
	if len(opts) > 0 && opts[0].RowBytes != nil {
		rowBytes = *opts[0].RowBytes
	}
	d := &nativeImageDescriptor{}
	w, h, bpp := width, height, 4 // Dart: "We only support 4 byte pixel formats"
	d.cachedWidth, d.cachedHeight, d.cachedBPP = &w, &h, &bpp
	format := [...]descFormat{descRGBA8888, descBGRA8888, descRGBAFloat32, descR32Float}[pixelFormat]
	d.data = buffer.data
	d.width, d.height, d.format, d.rowBytes = width, height, format, rowBytes
	d.alpha = alphaPremul
	if format == descRGBAFloat32 {
		d.alpha = alphaUnpremul
	}
	return d
}

// ImageDescriptorEncoded is ImageDescriptor.encoded: a descriptor of
// encoded image data in a supported format. It throws Exception('Invalid
// image data') synchronously when no decoder recognizes the data.
func ImageDescriptorEncoded(buffer *ImmutableBuffer) async.Future[ImageDescriptor] {
	d := &nativeImageDescriptor{rowBytes: -1}
	f := futurize(func(callback func(async.Void, bool)) string {
		if buffer == nil {
			return "Buffer parameter must not be null"
		}
		gen := createCompatibleGenerator(buffer.data)
		if gen == nil {
			return "Invalid image data"
		}
		info := gen.info()
		d.data, d.gen = buffer.data, gen
		d.width, d.height, d.format, d.alpha = info.width, info.height, info.format, info.alpha
		callback(async.Void{}, true)
		return ""
	})
	return async.Then(f, func(async.Void) ImageDescriptor { return d })
}

func (d *nativeImageDescriptor) Width() int {
	if d.cachedWidth == nil {
		w := d.width
		d.cachedWidth = &w
	}
	return *d.cachedWidth
}

func (d *nativeImageDescriptor) Height() int {
	if d.cachedHeight == nil {
		h := d.height
		d.cachedHeight = &h
	}
	return *d.cachedHeight
}

func (d *nativeImageDescriptor) BytesPerPixel() int {
	if d.cachedBPP == nil {
		b := d.format.bytesPerPixel()
		d.cachedBPP = &b
	}
	return *d.cachedBPP
}

func (f descFormat) bytesPerPixel() int {
	switch f {
	case descUnknown:
		return 0
	case descGray8:
		return 1
	case descRGBAFloat32:
		return 16
	}
	return 4
}

func (d *nativeImageDescriptor) Dispose() {}

func (d *nativeImageDescriptor) String() string {
	q := func(p *int) string {
		if p == nil {
			return "?"
		}
		return dart.ToString(*p)
	}
	return "ImageDescriptor(width: " + q(d.cachedWidth) + ", height: " + q(d.cachedHeight) +
		", bytes per pixel: " + q(d.cachedBPP) + ")"
}

func (d *nativeImageDescriptor) InstantiateCodec(opts ...ImageDescriptorInstantiateCodecOptions) async.Future[Codec] {
	var o ImageDescriptorInstantiateCodecOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return async.Async(func() Codec {
		tw, th := o.TargetWidth, o.TargetHeight
		if tw != nil && *tw <= 0 {
			tw = nil
		}
		if th != nil && *th <= 0 {
			th = nil
		}
		var w, h int
		switch {
		case tw == nil && th == nil:
			w, h = d.Width(), d.Height()
		case tw == nil:
			h = *th
			w = int(math.Round(float64(h) * (float64(d.Width()) / float64(d.Height()))))
		case th == nil:
			w = *tw
			h = int(dart.TruncDiv(float64(w), float64(d.Width())/float64(d.Height())))
		default:
			w, h = *tw, *th
		}
		return d.instantiateCodec(w, h, o.TargetFormat)
	})
}

// instantiateCodec is ImageDescriptor::instantiateCodec: a single frame
// codec, unless the data has several frames (then the target size is
// ignored).
func (d *nativeImageDescriptor) instantiateCodec(w, h int, format TargetPixelFormat) Codec {
	if d.gen == nil || d.gen.frameCount() == 1 {
		return &nativeCodec{single: &singleFrameCodec{descriptor: d, targetWidth: w, targetHeight: h, targetFormat: format}}
	}
	return &nativeCodec{multi: newMultiFrameCodec(d.gen)}
}

// ---------------------------------------------------------------------------
// Codec

// Codec decodes the frames of an image.
type Codec interface {
	// FrameCount is the number of frames in the image.
	FrameCount() int
	// RepetitionCount is how many times the animation repeats after the
	// first play: 0 plays once, -1 repeats forever.
	RepetitionCount() int
	// GetNextFrame decodes the next frame (cycling back to the first).
	GetNextFrame() async.Future[*FrameInfo]
	// Dispose releases the codec's resources.
	Dispose()
}

// nativeCodec is _NativeCodec over the engine's SingleFrameCodec or
// MultiFrameCodec.
type nativeCodec struct {
	single *singleFrameCodec
	multi  *multiFrameCodec

	cachedFrameCount, cachedRepetitionCount *int
}

func (c *nativeCodec) FrameCount() int {
	if c.cachedFrameCount == nil {
		n := 1
		if c.multi != nil {
			n = c.multi.frameCount
		}
		c.cachedFrameCount = &n
	}
	return *c.cachedFrameCount
}

func (c *nativeCodec) RepetitionCount() int {
	if c.cachedRepetitionCount == nil {
		n := 0
		if c.multi != nil {
			n = c.multi.repetitionCount
		}
		c.cachedRepetitionCount = &n
	}
	return *c.cachedRepetitionCount
}

// nextFrameCallback is the native callback: the image (nil on failure),
// its duration in milliseconds and the decode error.
type nextFrameCallback func(image *imageData, durationMs int, decodeError string)

func (c *nativeCodec) GetNextFrame() async.Future[*FrameInfo] {
	return async.Async(func() *FrameInfo {
		completer := async.NewSyncCompleter[*FrameInfo]()
		cb := func(image *imageData, durationMs int, decodeError string) {
			if image == nil {
				if decodeError == "" {
					decodeError = "Codec failed to produce an image, possibly due to invalid image data."
				}
				completer.CompleteError(dart.NewException(decodeError))
				return
			}
			completer.Complete(&FrameInfo{
				Image:    wrapImage(image, image.width, image.height),
				Duration: time.Duration(durationMs) * time.Millisecond,
			})
		}
		var errMsg string
		if c.multi != nil {
			errMsg = c.multi.getNextFrame(cb)
		} else {
			errMsg = c.single.getNextFrame(cb)
		}
		if errMsg != "" {
			panic(dart.NewException(errMsg))
		}
		return completer.Future().Await()
	})
}

func (c *nativeCodec) Dispose() {
	if c.multi != nil {
		c.multi.dispose()
	}
}

func (c *nativeCodec) String() string {
	if c.cachedFrameCount == nil {
		return "Codec()"
	}
	return "Codec(" + dart.ToString(*c.cachedFrameCount) + " frames)"
}

// singleFrameCodec is SingleFrameCodec: decodes once (Impeller's decoder)
// and hands the same image to every call.
type singleFrameCodec struct {
	descriptor                *nativeImageDescriptor
	targetWidth, targetHeight int
	targetFormat              TargetPixelFormat

	status  int // 0 new, 1 in progress, 2 complete
	cached  *imageData
	pending []nextFrameCallback
}

func (c *singleFrameCodec) getNextFrame(callback nextFrameCallback) string {
	if c.status == 2 {
		if c.cached == nil {
			return "Image failed to decode"
		}
		if c.cached.released {
			return "Decoded image has been disposed"
		}
		callback(c.cached, 0, "")
		return ""
	}
	c.pending = append(c.pending, callback)
	if c.status == 1 {
		return ""
	}
	loop := async.Current()
	d := c.descriptor
	w, h, f := c.targetWidth, c.targetHeight, c.targetFormat
	go func() {
		px, decodeError := decompress(d, w, h, f)
		loop.Post(func() {
			if px != nil {
				c.cached = &imageData{pixmap: *px}
			}
			c.status = 2
			for _, cb := range c.pending {
				cb(c.cached, 0, decodeError)
			}
			c.pending = nil
		})
	}()
	c.descriptor = nil
	c.status = 1
	return ""
}

// multiFrameCodec is MultiFrameCodec: composites each frame on the frames
// it depends on, in order, on a serial worker (the IO task runner).
type multiFrameCodec struct {
	frameCount, repetitionCount int

	mu       sync.Mutex // the state, used by the worker
	gen      imageGenerator
	disposed bool
	next     int
	last     []byte // lastRequiredFrame_ (premultiplied RGBA8)
	restore  *irect // restoreBGColorRect_
	queue    chan func()
}

func newMultiFrameCodec(gen imageGenerator) *multiFrameCodec {
	c := &multiFrameCodec{gen: gen, frameCount: gen.frameCount()}
	if pc := gen.playCount(); pc == infinitePlayCount {
		c.repetitionCount = -1
	} else {
		c.repetitionCount = pc - 1
	}
	return c
}

func (c *multiFrameCodec) dispose() {
	c.mu.Lock()
	c.disposed = true
	c.mu.Unlock()
}

func (c *multiFrameCodec) getNextFrame(callback nextFrameCallback) string {
	loop := async.Current()
	if c.frameCount == 0 {
		loop.Post(func() { callback(nil, 0, "Could not provide any frame.") })
		return ""
	}
	if c.queue == nil {
		c.queue = make(chan func(), 1024)
		go func() {
			for job := range c.queue {
				job()
			}
		}()
	}
	c.queue <- func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.disposed {
			return // the callback is cleared, never called
		}
		img, decodeError := c.nextFrameImage()
		duration := 0
		var data *imageData
		if img != nil {
			data = &imageData{pixmap: *img}
			duration = c.gen.frameInfo(c.next).duration
		}
		c.next = (c.next + 1) % c.frameCount
		loop.Post(func() { callback(data, duration, decodeError) })
	}
	return ""
}

// nextFrameImage is MultiFrameCodec::State::GetNextFrameImage.
func (c *multiFrameCodec) nextFrameImage() (*pixmap, string) {
	info := c.gen.info()
	w, h := info.width, info.height
	bitmap := make([]byte, w*h*4)
	fi := c.gen.frameInfo(c.next)
	required := fi.requiredFrame
	if required != noFrame && c.last != nil {
		copy(bitmap, c.last)
		if c.restore != nil {
			eraseRect(bitmap, w, h, *c.restore)
		}
	}
	if !c.gen.getPixels(bitmap, true, c.next, required) {
		return nil, fmt.Sprintf("Could not getPixels for frame %d", c.next)
	}
	keep := fi.disposal == disposalKeep
	restorePrevious := fi.disposal == disposalRestorePrevious
	if keep || (c.last != nil && !restorePrevious) {
		c.last = bitmap
	}
	if fi.disposal == disposalRestoreBG {
		r := fi.disposalRect
		c.restore = &r
	} else {
		c.restore = nil
	}
	return &pixmap{width: w, height: h, format: texRGBA8, pix8: append([]byte(nil), bitmap...)}, ""
}

// eraseRect is SkBitmap::erase(transparent, rect).
func eraseRect(px []byte, w, h int, r irect) {
	r = r.intersect(irect{0, 0, w, h})
	for y := r.top; y < r.bottom; y++ {
		clear(px[(y*w+r.left)*4 : (y*w+r.right)*4])
	}
}

// decompress is ImageDecoderImpeller::DecompressTexture followed by the
// upload (and the GPU resize): the decoded texture's pixels.
func decompress(d *nativeImageDescriptor, targetW, targetH int, targetFormat TargetPixelFormat) (*pixmap, string) {
	srcW, srcH := d.width, d.height
	compressed := d.gen != nil
	if !compressed && srcW == targetW && srcH == targetH &&
		d.format == descR32Float && targetFormat == TargetPixelFormatRFloat32 {
		px := &pixmap{width: srcW, height: srcH, format: texRF32, pixF: make([]float32, srcW*srcH)}
		readFloats(d.data, px.pixF)
		return px, ""
	}
	decodeW, decodeH := srcW, srcH
	if compressed {
		scale := float32(targetW) / float32(srcW)
		if s := float32(targetH) / float32(srcH); s > scale {
			scale = s
		}
		decodeW, decodeH = d.gen.scaledDimensions(scale)
	}
	// CreateImageInfo: the decoded color type.
	var out texFormat
	switch {
	case targetFormat == TargetPixelFormatRgbaFloat32:
		out = texRGBAF32
	case targetFormat != TargetPixelFormatDontCare:
		return nil, "unsupported target pixel format"
	case d.format == descRGBAFloat32:
		out = texRGBAF16
	default:
		out = texRGBA8
	}
	var px *pixmap
	if compressed {
		bitmap := make([]byte, decodeW*decodeH*4)
		var ok bool
		if decodeW == srcW && decodeH == srcH {
			ok = d.gen.getPixels(bitmap, d.alpha != alphaUnpremul, 0, noFrame)
		} else {
			ok = d.gen.getScaledPixels(bitmap, decodeW, decodeH, d.alpha != alphaUnpremul)
		}
		if !ok {
			return nil, "Could not decompress image."
		}
		if d.alpha == alphaUnpremul { // HandlePremultiplication
			premultiply8(bitmap)
		}
		px = &pixmap{width: decodeW, height: decodeH, format: texRGBA8, pix8: bitmap}
		if out == texRGBAF32 {
			px = &pixmap{width: decodeW, height: decodeH, format: texRGBAF32, pixF: floatsOf(px)}
		}
	} else {
		px = convertRaw(d, out)
	}
	if px.width != targetW || px.height != targetH {
		px = resizeBilinear(px, targetW, targetH)
	}
	return px, ""
}

func readFloats(data []byte, out []float32) {
	for i := range out {
		if 4*i+4 <= len(data) {
			out[i] = math.Float32frombits(uint32(data[4*i]) | uint32(data[4*i+1])<<8 | uint32(data[4*i+2])<<16 | uint32(data[4*i+3])<<24)
		}
	}
}

// convertRaw is DecodeToBitmap for raw pixels (readPixels from the
// descriptor's format and alpha type into the decoded color type, keeping
// the alpha type) followed by HandlePremultiplication.
func convertRaw(d *nativeImageDescriptor, out texFormat) *pixmap {
	w, h := d.width, d.height
	bpp := d.format.bytesPerPixel()
	rowBytes := d.rowBytes
	if rowBytes < 0 {
		rowBytes = w * bpp
	}
	// The source as floats (4 per pixel) or bytes.
	src8 := make([]byte, w*h*4)
	srcF := []float32(nil)
	if rowBytes*(h-1)+w*bpp <= len(d.data) || h == 0 {
		for y := 0; y < h; y++ {
			row := d.data[y*rowBytes:]
			switch d.format {
			case descRGBA8888:
				copy(src8[y*w*4:(y+1)*w*4], row[:w*4])
			case descBGRA8888:
				for x := 0; x < w; x++ {
					p := row[x*4:]
					copy(src8[(y*w+x)*4:], []byte{p[2], p[1], p[0], p[3]})
				}
			case descRGBAFloat32:
				if srcF == nil {
					srcF = make([]float32, w*h*4)
				}
				readFloats(row[:w*16], srcF[y*w*4:(y+1)*w*4])
			}
		}
	}
	px := &pixmap{width: w, height: h, format: out}
	switch {
	case out == texRGBA8:
		px.pix8 = src8
		if d.alpha == alphaUnpremul {
			premultiply8(px.pix8)
		}
		return px
	case srcF == nil:
		px.pixF = floatsOf(&pixmap{format: texRGBA8, pix8: src8})
	default:
		px.pixF = srcF
	}
	if out == texRGBAF16 {
		for i, v := range px.pixF {
			px.pixF[i] = quantizeHalf(v)
		}
	}
	if d.alpha == alphaUnpremul {
		f := px.pixF
		for i := 0; i+3 < len(f); i += 4 {
			a := f[i+3]
			f[i], f[i+1], f[i+2] = float32(f[i]*a), float32(f[i+1]*a), float32(f[i+2]*a)
			if out == texRGBAF16 {
				f[i], f[i+1], f[i+2] = quantizeHalf(f[i]), quantizeHalf(f[i+1]), quantizeHalf(f[i+2])
			}
		}
	}
	return px
}

// ---------------------------------------------------------------------------
// instantiateImageCodec and friends

// InstantiateImageCodecOptions are instantiateImageCodec's (and
// instantiateImageCodecFromBuffer's) named arguments. DisallowUpscaling is
// Dart's allowUpscaling: false.
type InstantiateImageCodecOptions struct {
	TargetWidth, TargetHeight *int
	DisallowUpscaling         bool
}

// InstantiateImageCodec decodes encoded image data into a Codec.
func InstantiateImageCodec(list []byte, opts ...InstantiateImageCodecOptions) async.Future[Codec] {
	return async.Async(func() Codec {
		buffer := ImmutableBufferFromUint8List(list).Await()
		return InstantiateImageCodecFromBuffer(buffer, opts...).Await()
	})
}

// InstantiateImageCodecFromBuffer is instantiateImageCodec for data in an
// ImmutableBuffer, which it disposes.
func InstantiateImageCodecFromBuffer(buffer *ImmutableBuffer, opts ...InstantiateImageCodecOptions) async.Future[Codec] {
	var o InstantiateImageCodecOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	tw, th := o.TargetWidth, o.TargetHeight
	return InstantiateImageCodecWithSize(buffer, InstantiateImageCodecWithSizeOptions{
		GetTargetSize: func(intrinsicWidth, intrinsicHeight int) TargetImageSize {
			if o.DisallowUpscaling {
				if tw != nil && *tw > intrinsicWidth {
					tw = &intrinsicWidth
				}
				if th != nil && *th > intrinsicHeight {
					th = &intrinsicHeight
				}
			}
			return NewTargetImageSize(TargetImageSize{Width: tw, Height: th})
		},
	})
}

// InstantiateImageCodecWithSizeOptions are instantiateImageCodecWithSize's
// named arguments (a nil GetTargetSize keeps the intrinsic size).
type InstantiateImageCodecWithSizeOptions struct {
	GetTargetSize TargetImageSizeCallback
}

// InstantiateImageCodecWithSize decodes the buffer's data into a Codec of
// the size GetTargetSize picks from the intrinsic size; it disposes the
// buffer.
func InstantiateImageCodecWithSize(buffer *ImmutableBuffer, opts ...InstantiateImageCodecWithSizeOptions) async.Future[Codec] {
	var getTargetSize TargetImageSizeCallback
	if len(opts) > 0 {
		getTargetSize = opts[0].GetTargetSize
	}
	if getTargetSize == nil {
		getTargetSize = func(int, int) TargetImageSize { return TargetImageSize{} }
	}
	return async.Async(func() Codec {
		descriptor := ImageDescriptorEncoded(buffer).Await()
		defer buffer.Dispose()
		targetSize := getTargetSize(descriptor.Width(), descriptor.Height())
		dartAssert(targetSize.Width == nil || *targetSize.Width > 0, "")
		dartAssert(targetSize.Height == nil || *targetSize.Height > 0, "")
		return descriptor.InstantiateCodec(ImageDescriptorInstantiateCodecOptions{
			TargetWidth: targetSize.Width, TargetHeight: targetSize.Height,
		}).Await()
	})
}

// TargetImageSizeCallback picks the size to decode an image at from its
// intrinsic size.
type TargetImageSizeCallback func(intrinsicWidth, intrinsicHeight int) TargetImageSize

// TargetImageSize is the size to decode an image at; a nil dimension keeps
// the aspect ratio (or the intrinsic size when both are nil). Build it with
// NewTargetImageSize to get Dart's assertions.
type TargetImageSize struct {
	Width, Height *int
}

// NewTargetImageSize is TargetImageSize({width, height}): it asserts that
// the given dimensions are positive.
func NewTargetImageSize(s TargetImageSize) TargetImageSize {
	dartAssert(s.Width == nil || *s.Width > 0, "")
	dartAssert(s.Height == nil || *s.Height > 0, "")
	return s
}

func (s TargetImageSize) String() string {
	q := func(p *int) string {
		if p == nil {
			return "null"
		}
		return dart.ToString(*p)
	}
	return "TargetImageSize(" + q(s.Width) + " x " + q(s.Height) + ")"
}

// DecodeImageFromList decodes the first frame of encoded image data and
// calls callback with it.
func DecodeImageFromList(list []byte, callback ImageDecoderCallback) {
	async.AsyncVoid(func() {
		codec := InstantiateImageCodec(list).Await()
		var frameInfo *FrameInfo
		func() {
			defer codec.Dispose()
			frameInfo = codec.GetNextFrame().Await()
		}()
		callback(frameInfo.Image)
	})
}

// DecodeImageFromPixelsOptions are decodeImageFromPixels' named arguments.
// DisallowUpscaling is Dart's allowUpscaling: false.
type DecodeImageFromPixelsOptions struct {
	RowBytes, TargetWidth, TargetHeight *int
	DisallowUpscaling                   bool
	TargetFormat                        TargetPixelFormat
}

// DecodeImageFromPixels makes an Image from raw pixels of the given format
// (8-bit formats premultiplied, rgbaFloat32 not) and calls callback with
// it.
func DecodeImageFromPixels(pixels []byte, width, height int, format PixelFormat, callback ImageDecoderCallback, opts ...DecodeImageFromPixelsOptions) {
	var o DecodeImageFromPixelsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	tw, th := o.TargetWidth, o.TargetHeight
	if tw != nil {
		dartAssert(!o.DisallowUpscaling || *tw <= width, "")
	}
	if th != nil {
		dartAssert(!o.DisallowUpscaling || *th <= height, "")
	}
	async.OnComplete(ImmutableBufferFromUint8List(pixels), func(buffer *ImmutableBuffer) {
		descriptor := NewImageDescriptorRaw(buffer, width, height, format, ImageDescriptorRawOptions{RowBytes: o.RowBytes})
		if o.DisallowUpscaling {
			if tw != nil && *tw > descriptor.Width() {
				w := descriptor.Width()
				tw = &w
			}
			if th != nil && *th > descriptor.Height() {
				h := descriptor.Height()
				th = &h
			}
		}
		frame := async.ThenFuture(descriptor.InstantiateCodec(ImageDescriptorInstantiateCodecOptions{
			TargetWidth: tw, TargetHeight: th, TargetFormat: o.TargetFormat,
		}), func(codec Codec) async.Future[*FrameInfo] {
			frameInfo := codec.GetNextFrame()
			codec.Dispose()
			return frameInfo
		})
		async.OnComplete(frame, func(frameInfo *FrameInfo) {
			buffer.Dispose()
			descriptor.Dispose()
			callback(frameInfo.Image)
		}, nil)
	}, nil)
}

// DecodeImageFromPixelsSync makes an Image from raw pixels synchronously
// (Impeller only). It panics with the engine's error message (a string,
// as the engine throws a String) for empty dimensions or an unsupported
// format.
func DecodeImageFromPixelsSync(pixels []byte, width, height int, format PixelFormat) *Image {
	data := &imageData{}
	image := wrapImage(data, width, height)
	if errMsg := decodeImageFromPixelsSync(pixels, width, height, format, data); errMsg != "" {
		panic(errMsg)
	}
	return image
}

// decodeImageFromPixelsSync is DoDecodeImageFromPixelsSync: an SkImage of
// the pixels (8-bit formats premultiplied, rgbaFloat32 unpremultiplied)
// turned into an RGBA8 texture by the raster thread.
func decodeImageFromPixelsSync(pixels []byte, width, height int, format PixelFormat, out *imageData) string {
	if width <= 0 || height <= 0 {
		return "Image dimensions must be greater than zero."
	}
	if format < 0 || format > PixelFormatRFloat32 {
		return "Invalid pixel format."
	}
	if pixels == nil {
		return "Pixels must not be null."
	}
	if format == PixelFormatRFloat32 {
		return "Unsupported pixel format."
	}
	bpp := 4
	if format == PixelFormatRgbaFloat32 {
		bpp = 16
	}
	if len(pixels) < width*height*bpp {
		return "Failed to create image from pixels."
	}
	px := &pixmap{width: width, height: height, format: texRGBA8, pix8: make([]byte, width*height*4)}
	switch format {
	case PixelFormatRgbaFloat32:
		f := make([]float32, width*height*4)
		readFloats(pixels, f)
		for i := 0; i+3 < len(f); i += 4 {
			a := f[i+3]
			px.pix8[i] = toUnorm8(f[i] * a)
			px.pix8[i+1] = toUnorm8(f[i+1] * a)
			px.pix8[i+2] = toUnorm8(f[i+2] * a)
			px.pix8[i+3] = toUnorm8(a)
		}
	case PixelFormatBgra8888:
		for i := 0; i < width*height*4; i += 4 {
			px.pix8[i], px.pix8[i+1], px.pix8[i+2], px.pix8[i+3] = pixels[i+2], pixels[i+1], pixels[i], pixels[i+3]
		}
	default:
		copy(px.pix8, pixels)
	}
	out.pixmap = *px
	return ""
}
