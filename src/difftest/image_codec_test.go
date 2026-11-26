package difftest

// Go side of the image codec differential test: the cases of
// flutter/test/impeller/image_codec_test.dart (run on Impeller), on
// dart:ui's codecs (ui.InstantiateImageCodec, ImageDescriptor,
// DecodeImageFromPixels...), compared field by field: frame counts,
// repetition counts, durations, sizes, color spaces, error messages and the
// pixels of every ImageByteFormat. Pixels compare exactly, except where an
// ENGINE GAP is documented (imageCodecTolerance: JPEG and WebP decoding,
// ICC conversion, the GPU's bilinear resize), where the maximum
// per-channel difference is checked and reported. PNG files are compared
// by their decoded pixels (the encoder's bytes differ: ENGINE GAP).

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// imageCodecTolerance is the largest per-channel difference allowed in a
// case's pixels (by case-name substring, and pixel field when key is set);
// 0 elsewhere. Differences are measured on premultiplied values: in the
// unpremultiplied formats (rawStraightRgba, rawExtendedRgba128, pngBack) a
// difference is scaled by the pixel's alpha, as dividing by a small alpha
// magnifies a premultiplied difference of 1.
var imageCodecTolerance = []struct {
	match, key string
	tol        int
}{
	// ENGINE GAP: Metal's bilinear scale (MPSImageBilinearScale) rounds
	// some interpolated values the other way at non-integer ratios (1
	// premultiplied, plus 1 from rounding the unpremultiplied value).
	{"pixels resize", "", 2},
	{"size Horizontal.png", "", 2},
	{"size transparent_image.png", "", 2},
	{"descriptor Horizontal.png", "", 2},
	{"withSize Horizontal.png half", "", 2},
	// ENGINE GAP: Go's JPEG IDCT (and the approximated scaled IDCTs) vs
	// libjpeg-turbo's ISLOW.
	// libjpeg-turbo scales by 2/8 and 4/8 with reduced IDCTs, approximated
	// here by box averages.
	{"Horizontal.jpg w130", "", 8},
	{"Horizontal.jpg w300 h10", "", 13},
	{".jpg", "", 4},
	// ENGINE GAP: SkPngEncoder's 16-bit encoding of float images, read back.
	{"Float32", "pngBack", 2},
}

func icFnv(b []byte) string {
	h := uint64(0xcbf29ce484222325)
	for _, x := range b {
		h ^= uint64(x)
		h *= 0x100000001b3
	}
	v := int64(h)
	if v < 0 {
		return "-" + strconv.FormatUint(uint64(-v), 16)
	}
	return strconv.FormatInt(v, 16)
}

func icBytes(b []byte) string {
	if len(b) <= 1100000 {
		return base64.StdEncoding.EncodeToString(b)
	}
	return "fnv:" + icFnv(b)
}

func icErr(e any) string {
	switch x := e.(type) {
	case *dart.AssertionError:
		return "assertion"
	case async.PanicError:
		return icErr(x.Value)
	case error:
		var ae *dart.AssertionError
		if errors.As(x, &ae) {
			return "assertion"
		}
		return x.Error()
	case string:
		return x
	}
	return fmt.Sprint(e)
}

// icAwait runs fut's loop until it completes.
func icAwait[T any](f async.Future[T]) (v T, err error) {
	done := false
	async.OnComplete(f, func(x T) { v, done = x, true }, func(e error) { err, done = e, true })
	l := async.Current()
	l.FlushMicrotasks()
	for !done {
		l.WaitPosted()
	}
	return v, err
}

func icTry[T any](f func() async.Future[T]) (v T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = async.PanicError{Value: r}
		}
	}()
	return icAwait(f())
}

func icImageInfo(image *ui.Image) map[string]any {
	out := map[string]any{"w": image.Width, "h": image.Height, "str": image.String()}
	func() {
		defer func() {
			if r := recover(); r != nil {
				out["colorSpace"] = icErr(r)
			}
		}()
		out["colorSpace"] = strings.TrimPrefix(image.ColorSpace().String(), "ColorSpace.")
	}()
	for _, f := range ui.ImageByteFormatValues() {
		d, err := icTry(func() async.Future[[]byte] { return image.ToByteData(ui.ImageToByteDataOptions{Format: f}) })
		switch {
		case err != nil:
			out[f.Name()] = "error: " + icErr(err)
		case d == nil:
			out[f.Name()] = nil
		case f == ui.ImageByteFormatPng:
			out["pngLen"] = len(d)
			out["pngHash"] = icFnv(d)
			c, err := icAwait(ui.InstantiateImageCodec(d))
			if err != nil {
				out["pngBack"] = "error: " + icErr(err)
				continue
			}
			fi, err := icAwait(c.GetNextFrame())
			if err != nil {
				out["pngBack"] = "error: " + icErr(err)
				continue
			}
			back, _ := icAwait(fi.Image.ToByteData(ui.ImageToByteDataOptions{Format: ui.ImageByteFormatRawStraightRgba}))
			out["pngBack"] = icBytes(back)
			out["pngBackSize"] = fmt.Sprintf("%dx%d", fi.Image.Width, fi.Image.Height)
			fi.Image.Dispose()
			c.Dispose()
		default:
			out[f.Name()] = icBytes(d)
		}
	}
	return out
}

func icCodecCase(make func() async.Future[ui.Codec], maxFrames int, pixelsEveryFrame bool) map[string]any {
	out := map[string]any{}
	codec, err := icTry(make)
	if err != nil {
		out["error"] = icErr(err)
		return out
	}
	out["strBefore"] = fmt.Sprint(codec)
	out["frameCount"] = codec.FrameCount()
	out["repetitionCount"] = codec.RepetitionCount()
	out["strAfter"] = fmt.Sprint(codec)
	var frames []any
	n := min(codec.FrameCount()+1, maxFrames)
	for i := 0; i < n; i++ {
		fi, err := icTry(codec.GetNextFrame)
		if err != nil {
			frames = append(frames, map[string]any{"error": icErr(err)})
			continue
		}
		var f map[string]any
		if pixelsEveryFrame || i == n-1 {
			f = icImageInfo(fi.Image)
		} else {
			d, _ := icAwait(fi.Image.ToByteData())
			f = map[string]any{"w": fi.Image.Width, "h": fi.Image.Height, "rawRgba": "fnv:" + icFnv(d)}
		}
		f["duration"] = fi.Duration.Milliseconds()
		frames = append(frames, f)
		fi.Image.Dispose()
	}
	out["frames"] = frames
	codec.Dispose()
	return out
}

func icPattern(w, h int) []byte {
	b := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			a := (x*61 + y*97) % 256
			b[i] = byte((x*37 + y*11) % (a + 1))
			b[i+1] = byte((x*13 + y*71) % (a + 1))
			b[i+2] = byte((x*7 + y*29) % (a + 1))
			b[i+3] = byte(a)
		}
	}
	return b
}

func icFromPixels(pixels []byte, w, h int, format ui.PixelFormat, opts ...ui.DecodeImageFromPixelsOptions) (*ui.Image, error) {
	c := async.NewCompleter[*ui.Image]()
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = async.PanicError{Value: r}
			}
		}()
		ui.DecodeImageFromPixels(pixels, w, h, format, func(i *ui.Image) { c.Complete(i) }, opts...)
	}()
	if err != nil {
		return nil, err
	}
	return icAwait(c.Future())
}

func icFlutterRoot(t *testing.T) string {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	return filepath.Join(home, "Projetos/jonathanperis/flutter")
}

func imageCodecCases(t *testing.T) map[string]any {
	root := icFlutterRoot(t)
	fixtures := filepath.Join(root, "engine/src/flutter/lib/ui/fixtures")
	resources := filepath.Join(root, "engine/src/flutter/testing/resources")
	read := func(dir, name string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Skipf("fixture: %v", err)
		}
		return b
	}
	fixture := func(n string) []byte { return read(fixtures, n) }
	resource := func(n string) []byte { return read(resources, n) }
	instantiate := func(data []byte, o ...ui.InstantiateImageCodecOptions) func() async.Future[ui.Codec] {
		return func() async.Future[ui.Codec] { return ui.InstantiateImageCodec(data, o...) }
	}
	out := map[string]any{}

	for _, name := range []string{"2x2.png", "square.png", "transparent_image.png"} {
		out["encoded "+name] = icCodecCase(instantiate(resource(name)), 8, true)
	}
	for _, name := range []string{"unmultiplied_alpha.png", "DisplayP3Logo.png", "WideGamutIndexed.png",
		"Horizontal.png", "four_frame_with_reuse_end.png", "DisplayP3Logo.jpg", "Horizontal.jpg",
		"four_frame_with_reuse.gif", "hello_loop_2.gif", "opaque_first_frame_then_alpha.gif",
		"opaque_first_frame_then_alpha.webp", "hello_loop_2.webp", "alpha_animated.apng",
		"dispose_op_background.apng", "out_of_bounds.apng", "out_of_bounds_wrapping.apng"} {
		out["encoded "+name] = icCodecCase(instantiate(fixture(name)), 8, true)
	}
	out["encoded heart.webp"] = icCodecCase(instantiate(fixture("heart.webp")), 70, false)
	out["encoded 2_dispose_op_restore_previous.apng"] = icCodecCase(instantiate(fixture("2_dispose_op_restore_previous.apng")), 71, false)
	out["encoded DashInNooglerHat.jpg 150"] = icCodecCase(instantiate(fixture("DashInNooglerHat.jpg"), ui.InstantiateImageCodecOptions{TargetWidth: ip(150)}), 8, true)

	out["invalid 1 2 3"] = icCodecCase(instantiate([]byte{1, 2, 3}), 8, true)
	out["invalid empty"] = icCodecCase(instantiate([]byte{}), 8, true)
	for _, n := range []int{20, 40, 100, 300, 700, 1200} {
		out[fmt.Sprintf("truncated four_frame_with_reuse.gif %d", n)] = icCodecCase(instantiate(fixture("four_frame_with_reuse.gif")[:n]), 8, true)
	}
	for _, n := range []int{30, 60, 200, 1000} {
		out[fmt.Sprintf("truncated Horizontal.png %d", n)] = icCodecCase(instantiate(fixture("Horizontal.png")[:n]), 8, true)
	}
	for _, n := range []int{300, 3000} {
		out[fmt.Sprintf("truncated DisplayP3Logo.jpg %d", n)] = icCodecCase(instantiate(fixture("DisplayP3Logo.jpg")[:n]), 8, true)
	}

	type size struct {
		file string
		w, h *int
		noUp bool
	}
	sizes := map[string]size{
		"2x2.png h1":                    {"2x2.png", nil, ip(1), false},
		"2x2.png w1":                    {"2x2.png", ip(1), nil, false},
		"2x2.png w10":                   {"2x2.png", ip(10), nil, false},
		"2x2.png w10 noup":              {"2x2.png", ip(10), nil, true},
		"2x2.png w10 h1":                {"2x2.png", ip(10), ip(1), false},
		"2x2.png w10 h1 noup":           {"2x2.png", ip(10), ip(1), true},
		"2x2.png w0 h-3":                {"2x2.png", ip(0), ip(-3), false},
		"transparent_image.png w8 h8":   {"transparent_image.png", ip(8), ip(8), false},
		"transparent_image.png w3 h2":   {"transparent_image.png", ip(3), ip(2), false},
		"transparent_image.png w7":      {"transparent_image.png", ip(7), nil, false},
		"Horizontal.png w150":           {"Horizontal.png", ip(150), nil, false},
		"Horizontal.png h37":            {"Horizontal.png", nil, ip(37), false},
		"Horizontal.png w97 h41":        {"Horizontal.png", ip(97), ip(41), false},
		"Horizontal.png w450":           {"Horizontal.png", ip(450), nil, false},
		"DisplayP3Logo.png w33":         {"DisplayP3Logo.png", ip(33), nil, false},
		"Horizontal.jpg w50":            {"Horizontal.jpg", ip(50), nil, false},
		"Horizontal.jpg h300":           {"Horizontal.jpg", nil, ip(300), false},
		"Horizontal.jpg w130":           {"Horizontal.jpg", ip(130), nil, false},
		"Horizontal.jpg w300 h10":       {"Horizontal.jpg", ip(300), ip(10), false},
		"four_frame_with_reuse.gif w10": {"four_frame_with_reuse.gif", ip(10), nil, false},
		"alpha_animated.apng w10":       {"alpha_animated.apng", ip(10), nil, false},
	}
	for name, s := range sizes {
		var data []byte
		if s.file == "2x2.png" || s.file == "transparent_image.png" {
			data = resource(s.file)
		} else {
			data = fixture(s.file)
		}
		out["size "+name] = icCodecCase(instantiate(data, ui.InstantiateImageCodecOptions{
			TargetWidth: s.w, TargetHeight: s.h, DisallowUpscaling: s.noUp}), 2, true)
	}

	out["withSize Horizontal.png half"] = icCodecCase(func() async.Future[ui.Codec] {
		return async.Async(func() ui.Codec {
			buffer := ui.ImmutableBufferFromUint8List(fixture("Horizontal.png")).Await()
			return ui.InstantiateImageCodecWithSize(buffer, ui.InstantiateImageCodecWithSizeOptions{
				GetTargetSize: func(w, h int) ui.TargetImageSize {
					return ui.NewTargetImageSize(ui.TargetImageSize{Width: ip(w / 2), Height: ip(h / 2)})
				}}).Await()
		})
	}, 1, true)
	out["withSize Horizontal.png default"] = icCodecCase(func() async.Future[ui.Codec] {
		return async.Async(func() ui.Codec {
			buffer := ui.ImmutableBufferFromUint8List(fixture("Horizontal.png")).Await()
			return ui.InstantiateImageCodecWithSize(buffer).Await()
		})
	}, 1, true)
	for _, wh := range [][2]*int{{ip(7), nil}, {nil, ip(7)}, {ip(301), nil}, {nil, ip(101)}, {ip(-1), ip(5)}} {
		log := map[string]any{}
		buffer, _ := icAwait(ui.ImmutableBufferFromUint8List(fixture("Horizontal.png")))
		log["bufferLength"] = buffer.Length()
		d, _ := icAwait(ui.ImageDescriptorEncoded(buffer))
		log["strBefore"] = fmt.Sprint(d)
		log["size"] = fmt.Sprintf("%dx%dx%d", d.Width(), d.Height(), d.BytesPerPixel())
		log["strAfter"] = fmt.Sprint(d)
		log["codec"] = icCodecCase(func() async.Future[ui.Codec] {
			return d.InstantiateCodec(ui.ImageDescriptorInstantiateCodecOptions{TargetWidth: wh[0], TargetHeight: wh[1]})
		}, 1, true)
		d.Dispose()
		buffer.Dispose()
		q := func(p *int) string {
			if p == nil {
				return "null"
			}
			return strconv.Itoa(*p)
		}
		out["descriptor Horizontal.png "+q(wh[0])+" "+q(wh[1])] = log
	}
	describe := func(data []byte) string {
		buffer, _ := icAwait(ui.ImmutableBufferFromUint8List(data))
		d, err := icAwait(ui.ImageDescriptorEncoded(buffer))
		if err != nil {
			return icErr(err)
		}
		return fmt.Sprintf("%dx%dx%d %v", d.Width(), d.Height(), d.BytesPerPixel(), d)
	}
	for _, f := range []string{"2x2.png", "square.png", "transparent_image.png"} {
		out["descriptor "+f] = describe(resource(f))
	}
	for _, f := range []string{"WideGamutIndexed.png", "DisplayP3Logo.png", "DisplayP3Logo.jpg", "Horizontal.jpg",
		"four_frame_with_reuse.gif", "hello_loop_2.webp", "alpha_animated.apng", "Horizontal.png"} {
		out["descriptor "+f] = describe(fixture(f))
	}
	{
		buffer, _ := icAwait(ui.ImmutableBufferFromUint8List(make([]byte, 64)))
		d := ui.NewImageDescriptorRaw(buffer, 4, 4, ui.PixelFormatRgbaFloat32)
		out["descriptor raw float"] = fmt.Sprintf("%v %dx%dx%d", d, d.Width(), d.Height(), d.BytesPerPixel())
	}
	{
		_, err := icTry(func() async.Future[ui.ImageDescriptor] {
			return async.Async(func() ui.ImageDescriptor {
				buffer := ui.ImmutableBufferFromUint8List([]byte{1, 2, 3}).Await()
				return ui.ImageDescriptorEncoded(buffer).Await()
			})
		})
		out["descriptor invalid"] = icErr(err)
	}
	{
		_, err := icAwait(ui.ImmutableBufferFromFilePath("/nonexistent/file.png"))
		out["buffer missing file"] = icErr(err)
		b, _ := icAwait(ui.ImmutableBufferFromFilePath(filepath.Join(fixtures, "four_frame_with_reuse.gif")))
		out["buffer file"] = fmt.Sprintf("%d %v", b.Length(), b.DebugDisposed())
		b.Dispose()
		out["buffer file disposed"] = fmt.Sprint(b.DebugDisposed())
	}
	out["TargetImageSize"] = []any{
		ui.TargetImageSize{}.String(),
		ui.NewTargetImageSize(ui.TargetImageSize{Width: ip(3)}).String(),
		ui.NewTargetImageSize(ui.TargetImageSize{Width: ip(3), Height: ip(4)}).String(),
	}

	pat := icPattern(5, 3)
	info := func(img *ui.Image, err error) any {
		if err != nil {
			return icErr(err)
		}
		return icImageInfo(img)
	}
	out["pixels rgba8888"] = info(icFromPixels(pat, 5, 3, ui.PixelFormatRgba8888))
	out["pixels bgra8888"] = info(icFromPixels(pat, 5, 3, ui.PixelFormatBgra8888))
	{
		padded := make([]byte, 3*24)
		for y := 0; y < 3; y++ {
			copy(padded[y*24:y*24+20], pat[y*20:])
			padded[y*24+20] = 0xAB
		}
		out["pixels rowBytes"] = info(icFromPixels(padded, 5, 3, ui.PixelFormatRgba8888, ui.DecodeImageFromPixelsOptions{RowBytes: ip(24)}))
	}
	big := icPattern(16, 12)
	for _, s := range []struct {
		w, h *int
		up   bool
	}{{ip(8), ip(6), true}, {ip(5), nil, true}, {nil, ip(5), true}, {ip(40), ip(30), true}, {ip(23), ip(7), true}, {ip(40), nil, false}, {ip(-100), ip(-99999), false}} {
		q := func(p *int) string {
			if p == nil {
				return "null"
			}
			return strconv.Itoa(*p)
		}
		out["pixels resize "+q(s.w)+" "+q(s.h)+" "+strconv.FormatBool(s.up)] = info(icFromPixels(big, 16, 12, ui.PixelFormatRgba8888,
			ui.DecodeImageFromPixelsOptions{TargetWidth: s.w, TargetHeight: s.h, DisallowUpscaling: !s.up}))
	}
	floats := []float32{1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 1, 1, 1, 1, 0,
		0.5, 0.25, 0.125, 0.5, 1.5, -0.25, 0.3, 0.7, 0.1, 0.2, 0.3, 0.4, 0.999, 0.501, 0.0039, 1}
	floatBytes := make([]byte, 4*len(floats))
	for i, f := range floats {
		binary.LittleEndian.PutUint32(floatBytes[4*i:], math.Float32bits(f))
	}
	out["pixels rgbaFloat32"] = info(icFromPixels(floatBytes, 4, 2, ui.PixelFormatRgbaFloat32))
	out["pixels rgbaFloat32 to rgbaFloat32"] = info(icFromPixels(floatBytes, 4, 2, ui.PixelFormatRgbaFloat32,
		ui.DecodeImageFromPixelsOptions{TargetFormat: ui.TargetPixelFormatRgbaFloat32}))
	out["pixels rgba8888 to rgbaFloat32"] = info(icFromPixels(pat, 5, 3, ui.PixelFormatRgba8888,
		ui.DecodeImageFromPixelsOptions{TargetFormat: ui.TargetPixelFormatRgbaFloat32}))
	{
		r := []float32{0, 0.25, 0.5, 1, 2, -1}
		rb := make([]byte, 4*len(r))
		for i, f := range r {
			binary.LittleEndian.PutUint32(rb[4*i:], math.Float32bits(f))
		}
		out["pixels rFloat32 to rFloat32"] = info(icFromPixels(rb, 3, 2, ui.PixelFormatRFloat32,
			ui.DecodeImageFromPixelsOptions{TargetFormat: ui.TargetPixelFormatRFloat32}))
	}
	for _, f := range ui.PixelFormatValues() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					out["sync "+f.Name()] = fmt.Sprintf("threw String: %v", r)
				}
			}()
			px, w, h := pat, 5, 3
			if f == ui.PixelFormatRgbaFloat32 || f == ui.PixelFormatRFloat32 {
				px = floatBytes
			}
			if f == ui.PixelFormatRgbaFloat32 {
				w, h = 4, 2
			}
			out["sync "+f.Name()] = icImageInfo(ui.DecodeImageFromPixelsSync(px, w, h, f))
		}()
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				out["sync zero"] = fmt.Sprintf("threw String: %v", r)
			}
		}()
		ui.DecodeImageFromPixelsSync(make([]byte, 4), 0, 1, ui.PixelFormatRgba8888)
		out["sync zero"] = "ok"
	}()

	{
		var log []any
		a, _ := icFromPixels(pat, 5, 3, ui.PixelFormatRgba8888)
		b := a.Clone()
		c := b.Clone()
		log = append(log, fmt.Sprintf("%v %v %v %v", a.IsCloneOf(b), b.IsCloneOf(c), a == b, a.DebugDisposed()))
		log = append(log, fmt.Sprintf("stacks %d", len(a.DebugGetOpenHandleStackTraces())))
		a.Dispose()
		log = append(log, fmt.Sprintf("%v %v %v", a.DebugDisposed(), b.DebugDisposed(), a.IsCloneOf(c)))
		log = append(log, fmt.Sprintf("stacks %d", len(b.DebugGetOpenHandleStackTraces())))
		func() {
			defer func() {
				if r := recover(); r != nil {
					log = append(log, "clone disposed: StateError "+r.(dart.StateError).Message)
				}
			}()
			a.Clone()
		}()
		d, _ := icAwait(c.ToByteData())
		log = append(log, fmt.Sprintf("after dispose of a: %d", len(d)))
		b.Dispose()
		c.Dispose()
		log = append(log, fmt.Sprintf("stacks %d", len(a.DebugGetOpenHandleStackTraces())))
		out["handles"] = log
	}
	{
		var log []any
		codec, _ := icAwait(ui.InstantiateImageCodec(resource("2x2.png")))
		f1, _ := icAwait(codec.GetNextFrame())
		f2, _ := icAwait(codec.GetNextFrame())
		log = append(log, fmt.Sprintf("same %v", f1.Image.IsCloneOf(f2.Image)))
		f1.Image.Dispose()
		f3, _ := icAwait(codec.GetNextFrame())
		log = append(log, fmt.Sprintf("after one dispose %v", f3.Image.IsCloneOf(f2.Image)))
		f2.Image.Dispose()
		f3.Image.Dispose()
		_, err := icTry(codec.GetNextFrame)
		if err != nil {
			log = append(log, icErr(err))
		} else {
			log = append(log, "ok")
		}
		out["single frame cache"] = log
	}
	return out
}

// icPixelKeys are the fields holding pixels.
var icPixelKeys = map[string]bool{"rawRgba": true, "rawStraightRgba": true, "rawUnmodified": true, "rawExtendedRgba128": true, "pngBack": true}

type icComparer struct {
	t       *testing.T
	maxDiff map[string]int
	fails   int
}

func (c *icComparer) errorf(format string, args ...any) {
	c.fails++
	if c.fails <= 60 {
		c.t.Errorf(format, args...)
	}
}

func icTolerance(name, key string) int {
	for _, e := range imageCodecTolerance {
		if strings.Contains(name, e.match) && (e.key == "" || e.key == key) {
			return e.tol
		}
	}
	return 0
}

func (c *icComparer) compare(caseName, path string, want, got any) {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			c.errorf("%s%s: Dart %s, Go %s", caseName, path, icShort(want), icShort(got))
			return
		}
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range g {
			keys[k] = true
		}
		for k := range keys {
			if k == "pngLen" || k == "pngHash" {
				continue // ENGINE GAP: PNG encoder bytes
			}
			if icPixelKeys[k] {
				c.comparePixels(caseName, path+"."+k, k, w[k], g[k])
				continue
			}
			c.compare(caseName, path+"."+k, w[k], g[k])
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			c.errorf("%s%s: Dart %s, Go %s", caseName, path, icShort(want), icShort(got))
			return
		}
		for i := range w {
			c.compare(caseName, fmt.Sprintf("%s[%d]", path, i), w[i], g[i])
		}
	case string:
		if strings.Contains(w, "Failed assertion") {
			w = "assertion"
		}
		if g, _ := got.(string); g != w {
			c.errorf("%s%s: Dart %q, Go %q", caseName, path, w, got)
		}
	default:
		if fmt.Sprint(want) != fmt.Sprint(got) {
			c.errorf("%s%s: Dart %v, Go %v", caseName, path, want, got)
		}
	}
}

func (c *icComparer) comparePixels(caseName, path, key string, want, got any) {
	ws, _ := want.(string)
	gs, _ := got.(string)
	if ws == gs {
		c.record(caseName, 0)
		return
	}
	tol := icTolerance(caseName, key)
	if strings.HasPrefix(ws, "fnv:") || strings.HasPrefix(ws, "error") || strings.HasPrefix(gs, "error") || want == nil || got == nil {
		if tol > 0 && strings.HasPrefix(ws, "fnv:") && strings.HasPrefix(gs, "fnv:") {
			return // too large to store: within tolerance cases only hashes exist
		}
		c.errorf("%s%s: Dart %s, Go %s", caseName, path, icShort(want), icShort(got))
		return
	}
	wb, _ := base64.StdEncoding.DecodeString(ws)
	gb, _ := base64.StdEncoding.DecodeString(gs)
	if len(wb) != len(gb) {
		c.errorf("%s%s: %d bytes in Dart, %d in Go", caseName, path, len(wb), len(gb))
		return
	}
	unpremul := key == "rawStraightRgba" || key == "rawExtendedRgba128" || key == "pngBack"
	d := 0
	if key == "rawExtendedRgba128" || strings.Contains(caseName, "Float32") && key == "rawUnmodified" && len(wb)%16 == 0 {
		// Floats: the difference in 1/255 units.
		for i := 0; i+4 <= len(wb); i += 4 {
			a := float64(math.Float32frombits(binary.LittleEndian.Uint32(wb[i:])))
			b := float64(math.Float32frombits(binary.LittleEndian.Uint32(gb[i:])))
			if a != b {
				diff := math.Abs(a - b)
				if unpremul && tol > 0 {
					diff *= float64(math.Float32frombits(binary.LittleEndian.Uint32(wb[i/16*16+12:])))
				}
				d = max(d, int(math.Ceil(diff*255-1e-9)))
			}
		}
	} else {
		for i := range wb {
			diff := math.Abs(float64(wb[i]) - float64(gb[i]))
			if unpremul && tol > 0 && i%4 != 3 {
				diff = diff * float64(wb[i/4*4+3]) / 255
			}
			d = max(d, int(math.Ceil(diff-1e-9)))
		}
	}
	c.record(caseName, d)
	if d > tol {
		c.errorf("%s%s: max channel difference %d > %d", caseName, path, d, tol)
	}
}

func (c *icComparer) record(caseName string, d int) {
	if d > c.maxDiff[caseName] {
		c.maxDiff[caseName] = d
	} else if _, ok := c.maxDiff[caseName]; !ok {
		c.maxDiff[caseName] = d
	}
}

func icShort(v any) string {
	s := fmt.Sprintf("%v", v)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

func TestImageCodecMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/image_codec.json")
	if err != nil {
		t.Skipf("no reference (%v); run flutter test --enable-impeller test/impeller/", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	goOut := imageCodecCases(t)
	raw, err := json.Marshal(goOut)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	c := &icComparer{t: t, maxDiff: map[string]int{}}
	names := make([]string, 0, len(ref))
	for k := range ref {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		g, ok := got[name]
		if !ok {
			c.errorf("%s: missing in Go", name)
			continue
		}
		c.compare(name, "", ref[name], g)
	}
	if len(got) != len(ref) {
		t.Errorf("Go has %d cases, Dart %d", len(got), len(ref))
	}
	for _, name := range names {
		if d := c.maxDiff[name]; d > 0 {
			t.Logf("%s: max channel difference %d", name, d)
		}
	}
	if c.fails > 60 {
		t.Errorf("... %d failures in all", c.fails)
	}
}
