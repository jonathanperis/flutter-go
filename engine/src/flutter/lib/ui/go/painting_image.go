package ui

// Port of painting.dart's Image (Flutter 3.50.1): ImageByteFormat,
// PixelFormat, TargetPixelFormat, ImageEventCallback, Image (handles over a
// shared _Image), FrameInfo and ImageDecoderCallback; the engine side is
// lib/ui/painting/image.cc and image_encoding*.cc (Impeller). Codecs and
// decoding are in painting_codec.go, the pixel conversions in
// painting_image_pixels.go.

import (
	"sync"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// ImageByteFormat is the format in which Image.toByteData returns bytes.
type ImageByteFormat int

const (
	// ImageByteFormatRawRgba is RGBA, 8 bits per channel, premultiplied.
	ImageByteFormatRawRgba ImageByteFormat = iota
	// ImageByteFormatRawStraightRgba is RGBA, 8 bits per channel, not premultiplied.
	ImageByteFormatRawStraightRgba
	// ImageByteFormatRawUnmodified is the image's own pixel format.
	ImageByteFormatRawUnmodified
	// ImageByteFormatRawExtendedRgba128 is RGBA as 32-bit floats, not premultiplied.
	ImageByteFormatRawExtendedRgba128
	// ImageByteFormatPng is a PNG file.
	ImageByteFormatPng
)

var imageByteFormatNames = [...]string{"rawRgba", "rawStraightRgba", "rawUnmodified", "rawExtendedRgba128", "png"}

// ImageByteFormatValues is ImageByteFormat.values.
func ImageByteFormatValues() []ImageByteFormat {
	return []ImageByteFormat{0, 1, 2, 3, 4}
}

func (f ImageByteFormat) Name() string   { return imageByteFormatNames[f] }
func (f ImageByteFormat) String() string { return "ImageByteFormat." + f.Name() }

// PixelFormat is the format of pixel data given to decodeImageFromPixels.
type PixelFormat int

const (
	PixelFormatRgba8888 PixelFormat = iota
	PixelFormatBgra8888
	PixelFormatRgbaFloat32
	PixelFormatRFloat32
)

var pixelFormatNames = [...]string{"rgba8888", "bgra8888", "rgbaFloat32", "rFloat32"}

// PixelFormatValues is PixelFormat.values.
func PixelFormatValues() []PixelFormat { return []PixelFormat{0, 1, 2, 3} }

func (f PixelFormat) Name() string   { return pixelFormatNames[f] }
func (f PixelFormat) String() string { return "PixelFormat." + f.Name() }

// TargetPixelFormat is the pixel format a decoded image should have.
type TargetPixelFormat int

const (
	TargetPixelFormatDontCare TargetPixelFormat = iota
	TargetPixelFormatRgbaFloat32
	TargetPixelFormatRFloat32
)

var targetPixelFormatNames = [...]string{"dontCare", "rgbaFloat32", "rFloat32"}

// TargetPixelFormatValues is TargetPixelFormat.values.
func TargetPixelFormatValues() []TargetPixelFormat { return []TargetPixelFormat{0, 1, 2} }

func (f TargetPixelFormat) Name() string   { return targetPixelFormatNames[f] }
func (f TargetPixelFormat) String() string { return "TargetPixelFormat." + f.Name() }

// ImageEventCallback is called with an Image when it is created or disposed.
type ImageEventCallback func(image *Image)

// ImageOnCreate and ImageOnDispose are Image.onCreate and Image.onDispose,
// called for every handle created (including clones) and disposed.
var ImageOnCreate, ImageOnDispose ImageEventCallback

// texFormat is the Impeller pixel format of an image's texture.
type texFormat int

const (
	texRGBA8   texFormat = iota // R8G8B8A8UNormInt, premultiplied
	texRGBAF16                  // R16G16B16A16Float, premultiplied (values held as float32)
	texRGBAF32                  // R32G32B32A32Float, premultiplied
	texRF32                     // R32Float
)

// imageData is dart:ui's _Image with the engine's CanvasImage and the
// DlImage it holds: the pixels shared by every handle (clone) of an image.
type imageData struct {
	pixmap

	handles  []*Image
	disposed bool // _Image._disposed (Dart side)
	released bool // CanvasImage::dispose: image_.reset()

	texMu sync.Mutex
	tex   *impeller.Texture
}

func (d *imageData) String() string {
	return "[" + dart.ToString(d.width) + "×" + dart.ToString(d.height) + "]"
}

// dispose is _Image.dispose followed by the native Image::dispose.
func (d *imageData) dispose() {
	dartAssert(!d.disposed, "")
	dartAssert(len(d.handles) == 0, "Attempted to dispose of an Image object that has "+
		dart.ToString(len(d.handles))+" open handles.\n"+
		"If you see this, it is a bug in dart:ui. Please file an issue at "+
		"https://github.com/flutter/flutter/issues/new.")
	d.disposed = true
	d.release()
}

// release drops the pixels and the texture (CanvasImage::dispose).
func (d *imageData) release() {
	d.texMu.Lock()
	if d.tex != nil {
		d.tex.Release()
		d.tex = nil
	}
	d.released = true
	d.pix8, d.pixF = nil, nil
	d.texMu.Unlock()
}

// colorSpace is Image::colorSpace (ImageEncodingImpeller::GetColorSpace):
// extended sRGB for half-float textures.
func (d *imageData) colorSpace() int {
	if d.format == texRGBAF16 {
		return 1
	}
	return 0
}

// Image is an opaque handle to decoded image data (pixels). Each handle
// must be disposed; the data is released when its last handle is.
type Image struct {
	// Width and Height are the image's dimensions in pixels.
	Width, Height int

	image      *imageData
	debugStack dart.StackTrace
	disposed   bool
}

// wrapImage is Image._(image, width, height): a new handle to image.
func wrapImage(data *imageData, width, height int) *Image {
	img := &Image{Width: width, Height: height, image: data}
	if debugMode {
		img.debugStack = dart.StackTraceCurrent()
	}
	data.handles = append(data.handles, img)
	if ImageOnCreate != nil {
		ImageOnCreate(img)
	}
	return img
}

// newImageFromRGBA8 wraps premultiplied RGBA8 pixels (width*height*4 bytes,
// row by row) in a new Image handle (Picture.toImage's result).
func newImageFromRGBA8(width, height int, premul []byte) *Image {
	return wrapImage(&imageData{pixmap: pixmap{width: width, height: height, format: texRGBA8, pix8: premul}}, width, height)
}

// Dispose releases this handle; the pixels are released with the last one.
func (img *Image) Dispose() {
	if ImageOnDispose != nil {
		ImageOnDispose(img)
	}
	dartAssert(!img.disposed && !img.image.disposed, "")
	i := img.handleIndex()
	dartAssert(i >= 0, "")
	img.disposed = true
	h := img.image.handles
	img.image.handles = append(h[:i:i], h[i+1:]...)
	if len(img.image.handles) == 0 {
		img.image.dispose()
	}
}

func (img *Image) handleIndex() int {
	for i, h := range img.image.handles {
		if h == img {
			return i
		}
	}
	return -1
}

// DebugDisposed is whether this handle has been disposed (asserts-only in
// Dart: it throws a StateError in release builds).
func (img *Image) DebugDisposed() bool {
	if !debugMode {
		panic(dart.StateError{Message: "Image.debugDisposed is only available when asserts are enabled."})
	}
	return img.disposed
}

// ImageToByteDataOptions are Image.toByteData's named arguments.
type ImageToByteDataOptions struct {
	Format ImageByteFormat // default rawRgba
}

// ToByteData converts the image to bytes in the given format. The future
// completes with nil or an error (Exception) when the encoding fails; it is
// delivered as an event of the current async.Loop.
func (img *Image) ToByteData(opts ...ImageToByteDataOptions) async.Future[[]byte] {
	var o ImageToByteDataOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	dartAssert(!img.disposed && !img.image.disposed, "")
	data := img.image
	return futurizeWithError(func(callback func([]byte, string)) string {
		loop := async.Current()
		// The engine snapshots the DlImage now and encodes on the IO thread.
		snap := data.pixmap
		go func() {
			encoded, errMsg := encodeImage(&snap, o.Format)
			loop.Post(func() {
				if errMsg == "" && encoded != nil {
					callback(encoded, "")
				} else {
					callback(nil, errMsg)
				}
			})
		}()
		return ""
	})
}

// ColorSpace is the color space of the image's pixels.
func (img *Image) ColorSpace() ColorSpace {
	switch v := img.image.colorSpace(); v {
	case 0:
		return ColorSpaceSRGB
	case 1:
		return ColorSpaceExtendedSRGB
	default:
		panic(dart.UnsupportedError{Message: "Unrecognized color space: " + dart.ToString(v)})
	}
}

// DebugGetOpenHandleStackTraces returns the stack traces of the creation of
// every open handle to this image's data (nil in release builds).
func (img *Image) DebugGetOpenHandleStackTraces() []dart.StackTrace {
	if !debugMode {
		return nil
	}
	stacks := make([]dart.StackTrace, 0, len(img.image.handles))
	for _, h := range img.image.handles {
		stacks = append(stacks, h.debugStack)
	}
	return stacks
}

// Clone creates a new handle to the same image data.
func (img *Image) Clone() *Image {
	if img.disposed {
		panic(dart.StateError{Message: "Cannot clone a disposed image.\n" +
			"The clone() method of a previously-disposed Image was called. Once an " +
			"Image object has been disposed, it can no longer be used to create " +
			"handles, as the underlying data may have been released."})
	}
	dartAssert(!img.image.disposed, "")
	return wrapImage(img.image, img.Width, img.Height)
}

// IsCloneOf is whether other is a handle to the same image data.
func (img *Image) IsCloneOf(other *Image) bool { return other.image == img.image }

func (img *Image) String() string { return img.image.String() }

// impellerTexture returns the image's pixels as an Impeller texture in the
// engine context (uploaded once, owned by the image data and released with
// it); nil when there is no GPU context or the image is empty. The SDK only
// has RGBA8888 textures, so float images are converted.
func (img *Image) impellerTexture() *impeller.Texture {
	d := img.image
	if d == nil {
		return nil
	}
	d.texMu.Lock()
	defer d.texMu.Unlock()
	if d.tex == nil && !d.released {
		if ctx := engineContext(); ctx != nil {
			px := d.pix8
			if d.format != texRGBA8 {
				px, _ = encodeImage(&d.pixmap, ImageByteFormatRawRgba)
			}
			if px != nil {
				d.tex = impeller.NewTextureRGBA8(ctx, d.width, d.height, px)
			}
		}
	}
	return d.tex
}

// ImageDecoderCallback is called with the decoded image.
type ImageDecoderCallback func(result *Image)

// FrameInfo is a frame of an animated image: its image and how long to show
// it.
type FrameInfo struct {
	Duration time.Duration
	Image    *Image
}

// futurizeWithError is painting.dart's _futurizeWithError: callbacker
// starts an operation that calls back with a value or an error message; a
// callback with a nil value completes the future with Exception(error), or
// throws it when called synchronously; a non-empty returned string throws
// Exception(string).
func futurizeWithError[T any](callbacker func(callback func(T, string)) string) async.Future[T] {
	completer := async.NewSyncCompleter[T]()
	isSync := true
	errMsg := callbacker(func(t T, e string) {
		if !isNil(t) {
			completer.Complete(t)
			return
		}
		if e == "" {
			e = "operation failed"
		}
		if isSync {
			panic(dart.NewException(e))
		}
		completer.CompleteError(dart.NewException(e))
	})
	isSync = false
	if errMsg != "" {
		panic(dart.NewException(errMsg))
	}
	return completer.Future()
}

// futurize is painting.dart's _futurize (a nil value fails with
// "operation failed").
func futurize[T any](callbacker func(callback func(T, bool)) string) async.Future[T] {
	completer := async.NewSyncCompleter[T]()
	isSync := true
	errMsg := callbacker(func(t T, ok bool) {
		if ok {
			completer.Complete(t)
			return
		}
		if isSync {
			panic(dart.NewException("operation failed"))
		}
		completer.CompleteError(dart.NewException("operation failed"))
	})
	isSync = false
	if errMsg != "" {
		panic(dart.NewException(errMsg))
	}
	return completer.Future()
}

func isNil(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case []byte:
		return x == nil
	case *imageData:
		return x == nil
	}
	return false
}
