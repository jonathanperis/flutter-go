package ui

// The engine's image generators (lib/ui/painting/image_generator*.cc and
// image_generator_registry.cc) with the image formats decoded in Go where
// the engine uses Skia's SkCodec: PNG (image/png), APNG (Flutter's own
// demuxer, image_generator_apng.cc, ported), GIF (a GIF decoder with
// Skia's frame dependencies, SkFrameHolder, ported), JPEG (image/jpeg),
// WebP and BMP (golang.org/x/image). Decoded pixels are converted to sRGB
// (ICC matrix/TRC profiles, painting_image_color.go).

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"math"
)

const (
	noFrame           = -1
	infinitePlayCount = math.MaxInt32
)

// disposal is SkCodecAnimation::DisposalMethod (0: unset).
type disposal int

const (
	disposalKeep disposal = iota + 1
	disposalRestoreBG
	disposalRestorePrevious
)

// irect is an SkIRect.
type irect struct{ left, top, right, bottom int }

func (r irect) empty() bool { return r.left >= r.right || r.top >= r.bottom }

func (r irect) intersect(o irect) irect {
	r.left, r.top = max(r.left, o.left), max(r.top, o.top)
	r.right, r.bottom = min(r.right, o.right), min(r.bottom, o.bottom)
	if r.empty() {
		return irect{}
	}
	return r
}

func (r irect) contains(o irect) bool {
	return !r.empty() && !o.empty() && r.left <= o.left && r.top <= o.top && r.right >= o.right && r.bottom >= o.bottom
}

// genInfo is the generator's SkImageInfo.
type genInfo struct {
	width, height int
	format        descFormat
	alpha         alphaType
}

// genFrameInfo is ImageGenerator::FrameInfo.
type genFrameInfo struct {
	requiredFrame int // noFrame when none
	duration      int // milliseconds
	disposal      disposal
	disposalRect  irect
	blendSrc      bool // SkCodecAnimation::Blend::kSrc (else kSrcOver)
}

// imageGenerator is ImageGenerator.
type imageGenerator interface {
	info() genInfo
	frameCount() int
	playCount() int // infinitePlayCount for forever
	frameInfo(i int) genFrameInfo
	scaledDimensions(scale float32) (int, int)
	// getPixels decodes frame i onto dst (RGBA8, info's size; it holds the
	// prior frame when the frame depends on one), premultiplied or not.
	getPixels(dst []byte, premul bool, frame, prior int) bool
	// getScaledPixels decodes the single frame at a scaled size.
	getScaledPixels(dst []byte, w, h int, premul bool) bool
}

// createCompatibleGenerator is ImageGeneratorRegistry::CreateCompatibleGenerator:
// the APNG demuxer first, then Skia's codecs.
func createCompatibleGenerator(data []byte) imageGenerator {
	if g := newAPNGGenerator(data); g != nil {
		return g
	}
	switch {
	case isPNG(data):
		if g := newPNGGenerator(data); g != nil {
			return g
		}
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		if g := newJPEGGenerator(data); g != nil {
			return g
		}
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		if g := newGIFGenerator(data); g != nil {
			return g
		}
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		if g := newWebPGenerator(data); g != nil {
			return g
		}
	case len(data) >= 2 && data[0] == 'B' && data[1] == 'M':
		if g := newBMPGenerator(data); g != nil {
			return g
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Still images (a single frame decoded on demand)

// stillGenerator is BuiltinSkiaCodecImageGenerator over a single-frame
// codec: decode returns straight RGBA8 in sRGB.
type stillGenerator struct {
	inf    genInfo
	decode func() (straight []byte, ok bool)
	scaled func(w, h int) (int, int, []byte, bool) // nil: no scaled decoding
	scale  func(scale float32) (int, int)
}

func (g *stillGenerator) info() genInfo   { return g.inf }
func (g *stillGenerator) frameCount() int { return 1 }
func (g *stillGenerator) playCount() int  { return 1 }
func (g *stillGenerator) frameInfo(int) genFrameInfo {
	return genFrameInfo{requiredFrame: noFrame, disposal: disposalKeep}
}

func (g *stillGenerator) scaledDimensions(scale float32) (int, int) {
	if g.scale != nil {
		return g.scale(scale)
	}
	return g.inf.width, g.inf.height
}

func (g *stillGenerator) getPixels(dst []byte, premul bool, _, _ int) bool {
	px, ok := g.decode()
	if !ok || len(px) != len(dst) {
		return false
	}
	copy(dst, px)
	if premul {
		premultiply8(dst)
	}
	return true
}

func (g *stillGenerator) getScaledPixels(dst []byte, w, h int, premul bool) bool {
	if g.scaled == nil {
		return false
	}
	sw, sh, px, ok := g.scaled(w, h)
	if !ok || sw != w || sh != h || len(px) != len(dst) {
		return false
	}
	copy(dst, px)
	if premul {
		premultiply8(dst)
	}
	return true
}

// straightRGBA converts a decoded Go image to straight RGBA8 (16-bit
// channels are scaled as skcms does: v/65535*255, rounded half up).
func straightRGBA(m image.Image) []byte {
	b := m.Bounds()
	out := make([]byte, 0, b.Dx()*b.Dy()*4)
	conv16 := func(v uint16) byte {
		f := float32(float32(v) * float32(1.0/65535))
		return byte(int32(float32(f*255) + 0.5))
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			switch c := m.At(x, y).(type) {
			case color.NRGBA:
				out = append(out, c.R, c.G, c.B, c.A)
			case color.RGBA:
				if c.A == 255 || c.A == 0 && c.R|c.G|c.B == 0 {
					out = append(out, c.R, c.G, c.B, c.A)
				} else {
					n := color.NRGBAModel.Convert(c).(color.NRGBA)
					out = append(out, n.R, n.G, n.B, n.A)
				}
			case color.Gray:
				out = append(out, c.Y, c.Y, c.Y, 255)
			case color.Gray16:
				g := conv16(c.Y)
				out = append(out, g, g, g, 255)
			case color.NRGBA64:
				out = append(out, conv16(c.R), conv16(c.G), conv16(c.B), conv16(c.A))
			case color.RGBA64:
				n := color.NRGBA64Model.Convert(c).(color.NRGBA64)
				out = append(out, conv16(n.R), conv16(n.G), conv16(n.B), conv16(n.A))
			default:
				n := color.NRGBAModel.Convert(c).(color.NRGBA)
				out = append(out, n.R, n.G, n.B, n.A)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// PNG

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func isPNG(data []byte) bool { return len(data) >= 8 && bytes.Equal(data[:8], pngSignature) }

// pngHeader is what SkPngCodec reads before the first IDAT.
type pngHeader struct {
	width, height       int
	bitDepth, colorType byte
	hasTRNS             bool
	iccp                []byte // the compressed profile
	srgb                bool
}

// readPNGHeader parses the chunks up to the first IDAT (png_read_info);
// ok is false when the data ends before it or the header is invalid.
func readPNGHeader(data []byte) (h pngHeader, ok bool) {
	if !isPNG(data) {
		return h, false
	}
	pos := 8
	first := true
	for {
		if pos+8 > len(data) {
			return h, false
		}
		n := int(binary.BigEndian.Uint32(data[pos:]))
		typ := string(data[pos+4 : pos+8])
		if typ == "IDAT" {
			return h, !first && h.width > 0
		}
		if n < 0 || pos+12+n > len(data) {
			return h, false
		}
		body := data[pos+8 : pos+8+n]
		if first {
			if typ != "IHDR" || n != 13 {
				return h, false
			}
			h.width = int(binary.BigEndian.Uint32(body))
			h.height = int(binary.BigEndian.Uint32(body[4:]))
			h.bitDepth, h.colorType = body[8], body[9]
			if h.width <= 0 || h.height <= 0 || h.width > 1<<24 || h.height > 1<<24 {
				return h, false
			}
			first = false
		}
		switch typ {
		case "tRNS":
			h.hasTRNS = true
		case "iCCP":
			if i := bytes.IndexByte(body, 0); i >= 0 && i+2 <= len(body) {
				h.iccp = body[i+2:]
			}
		case "sRGB":
			h.srgb = true
		case "IEND":
			return h, false
		}
		pos += 12 + n
	}
}

func (h pngHeader) genInfo() genInfo {
	inf := genInfo{width: h.width, height: h.height, format: descRGBA8888, alpha: alphaOpaque}
	if h.colorType == 0 && h.bitDepth <= 8 && !h.hasTRNS {
		inf.format = descGray8
	}
	if h.colorType == 4 || h.colorType == 6 || h.hasTRNS {
		inf.alpha = alphaUnpremul
	}
	return inf
}

// decodePNG decodes a whole PNG to straight sRGB RGBA8.
func decodePNG(data []byte, h pngHeader) ([]byte, bool) {
	m, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	px := straightRGBA(m)
	if !h.srgb && h.iccp != nil {
		if xf := iccTransformFromProfile(inflate(h.iccp)); xf != nil {
			xf.apply(px)
		}
	}
	return px, true
}

func newPNGGenerator(data []byte) imageGenerator {
	h, ok := readPNGHeader(data)
	if !ok {
		return nil
	}
	inf := h.genInfo()
	if inf.alpha == alphaUnpremul { // getInfoIncludingExif prefers premul
		inf.alpha = alphaPremul
	}
	var cached []byte
	var done, good bool
	return &stillGenerator{inf: inf, decode: func() ([]byte, bool) {
		if !done {
			cached, good = decodePNG(data, h)
			done = true
		}
		return cached, good
	}}
}

// ---------------------------------------------------------------------------
// APNG (image_generator_apng.cc)

type apngImage struct {
	png        []byte // the demuxed frame as a PNG file
	header     pngHeader
	frameInfo  *genFrameInfo
	xOff, yOff uint32
	pixels     []byte // decoded straight RGBA8, on demand
	decoded    bool
	decodeOK   bool
}

type apngGenerator struct {
	data       []byte
	inf        genInfo
	frames     int
	plays      int
	firstIndex int
	nextChunk  int // -1: none
	header     []byte
	images     []*apngImage
}

// pngChunkValid is APNGImageGenerator::IsValidChunkHeader.
func pngChunkValid(data []byte, pos int) bool {
	if pos < 0 || pos >= len(data) {
		return false
	}
	remaining := len(data) - pos
	if remaining < 8 {
		return false
	}
	n := int(binary.BigEndian.Uint32(data[pos:]))
	if remaining-8 < n || remaining-8-n < 4 {
		return false
	}
	for _, c := range data[pos+4 : pos+8] {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z') {
			return false
		}
	}
	return true
}

func pngChunkType(data []byte, pos int) string { return string(data[pos+4 : pos+8]) }
func pngChunkLen(data []byte, pos int) int     { return int(binary.BigEndian.Uint32(data[pos:])) }
func pngChunkSize(data []byte, pos int) int    { return 12 + pngChunkLen(data, pos) }

func pngNextChunk(data []byte, pos int) int {
	next := pos + pngChunkSize(data, pos)
	if !pngChunkValid(data, next) {
		return -1
	}
	return next
}

func newAPNGGenerator(data []byte) imageGenerator {
	if len(data) < 8+8 || !isPNG(data) {
		return nil
	}
	chunk := 8
	if !pngChunkValid(data, chunk) || pngChunkLen(data, chunk) != 13 || pngChunkType(data, chunk) != "IHDR" {
		return nil
	}
	for {
		chunk = pngNextChunk(data, chunk)
		if chunk < 0 || pngChunkType(data, chunk) == "IDAT" {
			return nil
		}
		if pngChunkType(data, chunk) == "acTL" {
			break
		}
	}
	if pngChunkLen(data, chunk) < 8 {
		return nil
	}
	numFrames := binary.BigEndian.Uint32(data[chunk+8:])
	numPlays := binary.BigEndian.Uint32(data[chunk+12:])
	// ExtractHeader.
	header := append([]byte(nil), pngSignature...)
	c := 8
	for {
		if pngChunkType(data, c) != "acTL" {
			header = append(header, data[c:c+pngChunkSize(data, c)]...)
		}
		c = pngNextChunk(data, c)
		if c < 0 {
			return nil
		}
		if t := pngChunkType(data, c); t == "fcTL" || t == "IDAT" || t == "fdAT" {
			break
		}
	}
	def, next := apngDemuxNext(data, header, c)
	if def == nil {
		return nil
	}
	plays := int(numPlays)
	if plays == 0 {
		plays = infinitePlayCount
	}
	inf := def.header.genInfo()
	g := &apngGenerator{data: data, inf: inf, frames: int(numFrames), plays: plays, nextChunk: next,
		header: header, images: []*apngImage{def}}
	if def.frameInfo == nil {
		g.firstIndex = 1
	}
	return g
}

// apngDemuxNext is DemuxNextImage: the image starting at chunk pos, and
// the chunk after it (-1 at the end).
func apngDemuxNext(data, header []byte, pos int) (*apngImage, int) {
	if !pngChunkValid(data, pos) {
		return nil, -1
	}
	typ := pngChunkType(data, pos)
	if typ != "fcTL" && typ != "IDAT" {
		return nil, -1
	}
	img := &apngImage{}
	var control []byte
	if typ == "fcTL" {
		if pngChunkLen(data, pos) < 26 {
			return nil, -1
		}
		control = data[pos+8 : pos+8+26]
		w, h := binary.BigEndian.Uint32(control[4:]), binary.BigEndian.Uint32(control[8:])
		x, y := binary.BigEndian.Uint32(control[12:]), binary.BigEndian.Uint32(control[16:])
		num, den := binary.BigEndian.Uint16(control[20:]), binary.BigEndian.Uint16(control[22:])
		fi := genFrameInfo{requiredFrame: noFrame}
		switch control[25] {
		case 0:
			fi.blendSrc = true
		case 1:
		default:
			return nil, -1
		}
		rect := irect{int(int32(x)), int(int32(y)), int(int32(x)) + int(int32(w)), int(int32(y)) + int(int32(h))}
		switch control[24] {
		case 0:
			fi.disposal = disposalKeep
		case 1:
			fi.disposal = disposalRestoreBG
			fi.disposalRect = rect
		case 2:
			fi.disposal = disposalRestorePrevious
		default:
			return nil, -1
		}
		if den == 0 {
			den = 100
		}
		fi.duration = int(float32(float32(num)*1000) / float32(den))
		img.frameInfo = &fi
		img.xOff, img.yOff = x, y
	}
	var body []byte
	for {
		t := pngChunkType(data, pos)
		if t != "fcTL" {
			if t == "fdAT" {
				n := pngChunkLen(data, pos)
				if n < 4 {
					return nil, -1
				}
				body = appendChunk(body, "IDAT", data[pos+12:pos+8+n])
			} else {
				body = append(body, data[pos:pos+pngChunkSize(data, pos)]...)
			}
		}
		pos = pngNextChunk(data, pos)
		if pos < 0 || pngChunkType(data, pos) == "fcTL" || pngChunkType(data, pos) == "IEND" {
			break
		}
	}
	buf := append([]byte(nil), header...)
	if control != nil {
		copy(buf[16:24], control[4:12])
		binary.BigEndian.PutUint32(buf[29:], crc32.ChecksumIEEE(buf[12:29]))
	}
	buf = append(buf, body...)
	buf = append(buf, 0, 0, 0, 0, 'I', 'E', 'N', 'D', 0xAE, 0x42, 0x60, 0x82)
	h, ok := readPNGHeader(buf)
	if !ok {
		return nil, -1
	}
	img.png, img.header = buf, h
	if pos >= 0 && pngChunkType(data, pos) == "IEND" {
		pos = -1
	}
	return img, pos
}

func appendChunk(out []byte, typ string, body []byte) []byte {
	var hdr [8]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(body)))
	copy(hdr[4:], typ)
	out = append(out, hdr[:]...)
	out = append(out, body...)
	crc := crc32.NewIEEE()
	crc.Write(hdr[4:])
	crc.Write(body)
	return binary.BigEndian.AppendUint32(out, crc.Sum32())
}

func (g *apngGenerator) info() genInfo   { return g.inf }
func (g *apngGenerator) frameCount() int { return g.frames }
func (g *apngGenerator) playCount() int {
	if g.frames > 1 {
		return g.plays
	}
	return 1
}

func (g *apngGenerator) scaledDimensions(float32) (int, int) { return g.inf.width, g.inf.height }

func (g *apngGenerator) getScaledPixels([]byte, int, int, bool) bool { return false }

func (g *apngGenerator) frameInfo(i int) genFrameInfo {
	idx := g.firstIndex + i
	if !g.demuxTo(idx) || g.images[idx].frameInfo == nil {
		return genFrameInfo{requiredFrame: noFrame}
	}
	return *g.images[idx].frameInfo
}

// demuxNextInternal is DemuxNextImageInternal.
func (g *apngGenerator) demuxNextInternal() bool {
	if g.nextChunk < 0 {
		return false
	}
	var img *apngImage
	img, g.nextChunk = apngDemuxNext(g.data, g.header, g.nextChunk)
	if img == nil || img.frameInfo == nil {
		return false
	}
	last := g.images[len(g.images)-1].frameInfo
	if last == nil {
		return false
	}
	n := len(g.images)
	if n > g.firstIndex && (last.disposal == disposalKeep || last.disposal == disposalRestoreBG) {
		img.frameInfo.requiredFrame = n - 1
	} else if n > g.firstIndex+1 && last.disposal == disposalRestorePrevious {
		img.frameInfo.requiredFrame = n - 2
	}
	g.images = append(g.images, img)
	return (img.header.colorType == 0 && img.header.bitDepth <= 8 && !img.header.hasTRNS) ==
		(g.images[0].header.colorType == 0 && g.images[0].header.bitDepth <= 8 && !g.images[0].header.hasTRNS)
}

func (g *apngGenerator) demuxTo(idx int) bool {
	if idx >= len(g.images) {
		for g.demuxNextInternal() && idx >= len(g.images) {
		}
		if idx >= len(g.images) {
			return false
		}
	}
	return true
}

func (im *apngImage) decode() bool {
	if !im.decoded {
		im.pixels, im.decodeOK = decodePNG(im.png, im.header)
		im.decoded = true
	}
	return im.decodeOK
}

func (g *apngGenerator) getPixels(dst []byte, premul bool, frame, _ int) bool {
	idx := g.firstIndex + frame
	if !g.demuxTo(idx) {
		return g.renderDefault(dst, premul)
	}
	f := g.images[idx]
	fw, fh := f.header.width, f.header.height
	if !f.decode() {
		return g.renderDefault(dst, premul)
	}
	if f.frameInfo == nil {
		return false
	}
	w, h := g.inf.width, g.inf.height
	if f.xOff > math.MaxUint32-uint32(fw) || f.yOff > math.MaxUint32-uint32(fh) ||
		uint64(f.xOff)+uint64(fw) > uint64(w) || uint64(f.yOff)+uint64(fh) > uint64(h) {
		return false
	}
	if f.header.genInfo().format != descRGBA8888 {
		return false // the frame isn't kN32
	}
	if f.frameInfo.blendSrc {
		for y := 0; y < fh; y++ {
			for x := 0; x < fw; x++ {
				s := f.pixels[(y*fw+x)*4:]
				d := dst[((y+int(f.yOff))*w+x+int(f.xOff))*4:]
				copy(d[:4], s[:4])
				if premul && s[3] != 255 {
					d[0], d[1], d[2] = premul8(s[0], s[3]), premul8(s[1], s[3]), premul8(s[2], s[3])
				}
			}
		}
		return true
	}
	srcUnpremul := f.header.genInfo().alpha == alphaUnpremul
	for y := 0; y < fh; y++ {
		for x := 0; x < fw; x++ {
			var src, d [4]byte
			copy(src[:], f.pixels[(y*fw+x)*4:])
			dp := dst[((y+int(f.yOff))*w+x+int(f.xOff))*4:]
			copy(d[:], dp)
			if !premul {
				apngPremultiply(&d)
			}
			if srcUnpremul {
				apngPremultiply(&src)
			}
			for i := 0; i < 4; i++ {
				d[i] = byte(int(src[i]) + int(d[i])*(0xFF-int(src[3]))/0xFF)
			}
			if !premul {
				apngUnpremultiply(&d)
			}
			copy(dp, d[:])
		}
	}
	return true
}

func apngPremultiply(p *[4]byte) {
	for i := 0; i < 3; i++ {
		p[i] = byte(int(p[i]) * int(p[3]) / 0xFF)
	}
}

func apngUnpremultiply(p *[4]byte) {
	if p[3] == 0 {
		p[0], p[1], p[2] = 0, 0, 0
		return
	}
	for i := 0; i < 3; i++ {
		p[i] = byte(int(p[i]) * 0xFF / int(p[3]))
	}
}

// renderDefault is RenderDefaultImage: the default image's pixels.
func (g *apngGenerator) renderDefault(dst []byte, premul bool) bool {
	f := g.images[0]
	if f.header.width > g.inf.width || f.header.height > g.inf.height {
		return false
	}
	if !f.decode() || len(f.pixels) > len(dst) {
		return false
	}
	if f.header.width != g.inf.width {
		return false
	}
	copy(dst, f.pixels)
	if premul {
		premultiply8(dst[:len(f.pixels)])
	}
	return true
}
