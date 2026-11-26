package ui

// JPEG, WebP and BMP for dart:ui's codecs. ENGINE GAPS: the engine decodes
// JPEG with libjpeg-turbo (ISLOW IDCT, fancy chroma upsampling, scaled
// IDCTs for 1/8..7/8 sizes) and WebP with libwebp (fancy upsampling, its
// own premultiplication); Go's image/jpeg and x/image/webp round
// differently, and a scaled JPEG decode is approximated by an area
// average of the full decode. Animated WebP frames are composited as
// SkWebpCodec does (SkFrameHolder's required frames, blend and disposal).

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"

	"golang.org/x/image/bmp"
	"golang.org/x/image/webp"
)

// ---------------------------------------------------------------------------
// JPEG

type jpegHeader struct {
	width, height int // as encoded
	gray          bool
	orientation   int
	icc           []byte
}

// readJPEGHeader reads the markers up to the start of scan
// (jpeg_read_header); ok is false when the data ends before it.
func readJPEGHeader(data []byte) (h jpegHeader, ok bool) {
	h.orientation = 1
	pos := 2
	var iccChunks [][]byte
	sof := false
	for {
		for pos < len(data) && data[pos] != 0xff {
			pos++ // libjpeg skips garbage before a marker
		}
		for pos < len(data) && data[pos] == 0xff {
			pos++
		}
		if pos >= len(data) {
			return h, false
		}
		m := data[pos]
		pos++
		if m == 0xd8 || m >= 0xd0 && m <= 0xd7 || m == 0x01 {
			continue
		}
		if m == 0xd9 {
			return h, false
		}
		if pos+2 > len(data) {
			return h, false
		}
		n := int(binary.BigEndian.Uint16(data[pos:]))
		if n < 2 || pos+n > len(data) {
			return h, false
		}
		body := data[pos+2 : pos+n]
		switch {
		case m >= 0xc0 && m <= 0xcf && m != 0xc4 && m != 0xc8 && m != 0xcc:
			if len(body) < 6 {
				return h, false
			}
			h.height = int(binary.BigEndian.Uint16(body[1:]))
			h.width = int(binary.BigEndian.Uint16(body[3:]))
			h.gray = body[5] == 1
			sof = true
		case m == 0xe1 && bytes.HasPrefix(body, []byte("Exif\x00\x00")):
			if o := exifOrientation(body[6:]); o >= 1 && o <= 8 {
				h.orientation = o
			}
		case m == 0xe2 && bytes.HasPrefix(body, []byte("ICC_PROFILE\x00")) && len(body) > 14:
			iccChunks = append(iccChunks, body[14:])
		case m == 0xda:
			for _, c := range iccChunks {
				h.icc = append(h.icc, c...)
			}
			return h, sof && h.width > 0 && h.height > 0
		}
		pos += n
	}
}

// exifOrientation reads tag 0x0112 of a TIFF structure's IFD0.
func exifOrientation(t []byte) int {
	if len(t) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	off := int(bo.Uint32(t[4:]))
	if off < 0 || off+2 > len(t) {
		return 0
	}
	n := int(bo.Uint16(t[off:]))
	for i := 0; i < n; i++ {
		e := off + 2 + 12*i
		if e+12 > len(t) {
			return 0
		}
		if bo.Uint16(t[e:]) == 0x0112 && bo.Uint16(t[e+2:]) == 3 {
			return int(bo.Uint16(t[e+8:]))
		}
	}
	return 0
}

func swapsWidthHeight(orientation int) bool { return orientation >= 5 }

// orient is SkPixmapUtils::Orient: the encoded pixels (w x h) as displayed.
func orient(px []byte, w, h, orientation int) []byte {
	if orientation <= 1 {
		return px
	}
	ow, oh := w, h
	if swapsWidthHeight(orientation) {
		ow, oh = h, w
	}
	out := make([]byte, len(px))
	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			var sx, sy int
			switch orientation {
			case 2:
				sx, sy = w-1-x, y
			case 3:
				sx, sy = w-1-x, h-1-y
			case 4:
				sx, sy = x, h-1-y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, h-1-x
			case 7:
				sx, sy = w-1-y, h-1-x
			case 8:
				sx, sy = w-1-y, x
			}
			copy(out[(y*ow+x)*4:(y*ow+x)*4+4], px[(sy*w+sx)*4:])
		}
	}
	return out
}

// jpegScaledDimensions is SkJpegCodec::onGetScaledDimensions: libjpeg-turbo
// scales by num/8.
func jpegScaledDimensions(w, h int, scale float32) (int, int) {
	num := 1
	switch {
	case scale >= 0.9375:
		num = 8
	case scale >= 0.8125:
		num = 7
	case scale >= 0.6875:
		num = 6
	case scale >= 0.5625:
		num = 5
	case scale >= 0.4375:
		num = 4
	case scale >= 0.3125:
		num = 3
	case scale >= 0.1875:
		num = 2
	}
	return (w*num + 7) / 8, (h*num + 7) / 8
}

func newJPEGGenerator(data []byte) imageGenerator {
	h, ok := readJPEGHeader(data)
	if !ok {
		return nil
	}
	inf := genInfo{width: h.width, height: h.height, format: descRGBA8888, alpha: alphaOpaque}
	if h.gray {
		inf.format = descGray8
	}
	if swapsWidthHeight(h.orientation) {
		inf.width, inf.height = inf.height, inf.width
	}
	var cached []byte
	var done, good bool
	decodeAt := func(num int) ([]byte, bool) {
		m, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, false
		}
		px := jpegRGBA(m, num)
		if xf := iccTransformFromProfile(h.icc); xf != nil && !h.gray {
			xf.apply(px)
		}
		w, ht := (h.width*num+7)/8, (h.height*num+7)/8
		return orient(px, w, ht, h.orientation), true
	}
	decode := func() ([]byte, bool) {
		if !done {
			done = true
			cached, good = decodeAt(8)
		}
		return cached, good
	}
	return &stillGenerator{
		inf:    inf,
		decode: decode,
		scale: func(scale float32) (int, int) {
			w, ht := jpegScaledDimensions(h.width, h.height, scale)
			if swapsWidthHeight(h.orientation) {
				return ht, w
			}
			return w, ht
		},
		scaled: func(w, ht int) (int, int, []byte, bool) {
			ew := w
			if swapsWidthHeight(h.orientation) {
				ew = ht
			}
			for num := 1; num <= 8; num++ {
				if (h.width*num+7)/8 == ew {
					px, ok := decodeAt(num)
					return w, ht, px, ok
				}
			}
			return 0, 0, nil, false
		},
	}
}

// areaAverage downsamples straight RGBA8 pixels by averaging the source
// pixels each destination pixel covers.
func areaAverage(px []byte, sw, sh, dw, dh int) []byte {
	out := make([]byte, dw*dh*4)
	for y := 0; y < dh; y++ {
		y0, y1 := y*sh/dh, max((y+1)*sh/dh, y*sh/dh+1)
		for x := 0; x < dw; x++ {
			x0, x1 := x*sw/dw, max((x+1)*sw/dw, x*sw/dw+1)
			var sum [4]int
			for yy := y0; yy < min(y1, sh); yy++ {
				for xx := x0; xx < min(x1, sw); xx++ {
					for c := 0; c < 4; c++ {
						sum[c] += int(px[(yy*sw+xx)*4+c])
					}
				}
			}
			n := (min(y1, sh) - y0) * (min(x1, sw) - x0)
			for c := 0; c < 4; c++ {
				out[(y*dw+x)*4+c] = byte((sum[c] + n/2) / n)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// BMP

func newBMPGenerator(data []byte) imageGenerator {
	cfg, err := bmp.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil
	}
	return newStill(cfg, data, func(r *bytes.Reader) (image.Image, error) { return bmp.Decode(r) })
}

func newStill(cfg image.Config, data []byte, dec func(*bytes.Reader) (image.Image, error)) imageGenerator {
	var cached []byte
	var done, good bool
	return &stillGenerator{
		inf: genInfo{width: cfg.Width, height: cfg.Height, format: descRGBA8888, alpha: alphaPremul},
		decode: func() ([]byte, bool) {
			if !done {
				done = true
				if m, err := dec(bytes.NewReader(data)); err == nil {
					cached, good = straightRGBA(m), true
				}
			}
			return cached, good
		},
	}
}

// ---------------------------------------------------------------------------
// WebP

type webpGenerator struct {
	width, height int
	loopCount     int
	frames        []*gifFrame // the SkFrame state (rect, blend, disposal...)
	chunks        [][]byte    // each frame's ALPH/VP8/VP8L chunks
}

func newWebPGenerator(data []byte) imageGenerator {
	chunks, ok := riffChunks(data[12:])
	if !ok || len(chunks) == 0 {
		return nil
	}
	if string(chunks[0][:4]) != "VP8X" || len(chunks[0]) < 18 || chunks[0][8]&0x02 == 0 {
		cfg, err := webp.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			return nil
		}
		return newStill(cfg, data, func(r *bytes.Reader) (image.Image, error) { return webp.Decode(r) })
	}
	x := chunks[0][8:]
	g := &webpGenerator{width: int(le24(x[4:])) + 1, height: int(le24(x[7:])) + 1}
	for _, c := range chunks[1:] {
		body := c[8:]
		switch string(c[:4]) {
		case "ANIM":
			if len(body) >= 6 {
				g.loopCount = int(binary.LittleEndian.Uint16(body[4:]))
			}
		case "ANMF":
			if len(body) < 16 {
				return nil
			}
			fx, fy := 2*int(le24(body)), 2*int(le24(body[3:]))
			fw, fh := int(le24(body[6:]))+1, int(le24(body[9:]))+1
			f := &gifFrame{rect: irect{fx, fy, fx + fw, fy + fh}, delayMs: int(le24(body[12:])), disposal: disposalKeep}
			if body[15]&1 != 0 {
				f.disposal = disposalRestoreBG
			}
			sub, ok := riffChunks(body[16:])
			if !ok {
				return nil
			}
			var frameData []byte
			for _, s := range sub {
				switch string(s[:4]) {
				case "ALPH":
					f.reportsAlpha = true
					frameData = append(frameData, s...)
				case "VP8L":
					if len(s) >= 13 && s[12]&0x10 != 0 { // the alpha_is_used bit
						f.reportsAlpha = true
					}
					frameData = append(frameData, s...)
				case "VP8 ":
					frameData = append(frameData, s...)
				}
			}
			f.blendSrc = body[15]&2 != 0 || !f.reportsAlpha
			g.frames = append(g.frames, f)
			g.chunks = append(g.chunks, frameData)
			(&gifGenerator{width: g.width, height: g.height, frames: g.frames}).setAlphaAndRequiredFrame(f, len(g.frames)-1)
		}
	}
	if len(g.frames) == 0 {
		return nil
	}
	return g
}

func le24(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 }

// riffChunks splits RIFF chunks (each with its 8-byte header and padded
// body).
func riffChunks(b []byte) ([][]byte, bool) {
	var out [][]byte
	for len(b) >= 8 {
		n := int(binary.LittleEndian.Uint32(b[4:]))
		end := 8 + n + n&1
		if n < 0 || 8+n > len(b) {
			return out, len(out) > 0
		}
		out = append(out, b[:min(end, len(b))])
		b = b[min(end, len(b)):]
	}
	return out, true
}

func (g *webpGenerator) info() genInfo {
	return genInfo{width: g.width, height: g.height, format: descRGBA8888, alpha: alphaPremul}
}

func (g *webpGenerator) frameCount() int { return len(g.frames) }

func (g *webpGenerator) playCount() int {
	if g.loopCount == 0 {
		return infinitePlayCount
	}
	return g.loopCount
}

func (g *webpGenerator) frameInfo(i int) genFrameInfo {
	if i < 0 || i >= len(g.frames) {
		return genFrameInfo{}
	}
	f := g.frames[i]
	return genFrameInfo{requiredFrame: f.requiredFrame, duration: f.delayMs, disposal: f.disposal, blendSrc: f.blendSrc}
}

func (g *webpGenerator) scaledDimensions(float32) (int, int)         { return g.width, g.height }
func (g *webpGenerator) getScaledPixels([]byte, int, int, bool) bool { return false }

// decodeFrame decodes a frame's bitstream as a still WebP file.
func (g *webpGenerator) decodeFrame(i int) ([]byte, bool) {
	f := g.frames[i]
	fw, fh := f.rect.right-f.rect.left, f.rect.bottom-f.rect.top
	var body []byte
	if f.reportsAlpha && bytes.HasPrefix(g.chunks[i], []byte("ALPH")) {
		vp8x := make([]byte, 18)
		copy(vp8x, "VP8X")
		binary.LittleEndian.PutUint32(vp8x[4:], 10)
		vp8x[8] = 0x10
		putLE24(vp8x[12:], uint32(fw-1))
		putLE24(vp8x[15:], uint32(fh-1))
		body = append(vp8x, g.chunks[i]...)
	} else {
		body = g.chunks[i]
	}
	file := append([]byte("RIFF\x00\x00\x00\x00WEBP"), body...)
	binary.LittleEndian.PutUint32(file[4:], uint32(len(file)-8))
	m, err := webp.Decode(bytes.NewReader(file))
	if err != nil || m.Bounds().Dx() != fw || m.Bounds().Dy() != fh {
		return nil, false
	}
	return webpRGBA(m), true
}

func putLE24(b []byte, v uint32) { b[0], b[1], b[2] = byte(v), byte(v>>8), byte(v>>16) }

func (g *webpGenerator) getPixels(dst []byte, premul bool, i, prior int) bool {
	if i < 0 || i >= len(g.frames) {
		return false
	}
	f := g.frames[i]
	w, h := g.width, g.height
	if f.requiredFrame == noFrame {
		clear(dst)
	} else if prior == f.requiredFrame && g.frames[prior].disposal == disposalRestoreBG {
		eraseRect(dst, w, h, g.frames[prior].rect)
	}
	px, ok := g.decodeFrame(i)
	if !ok {
		return false
	}
	webpPremultiply(px)
	fw := f.rect.right - f.rect.left
	r := f.rect.intersect(irect{0, 0, w, h})
	for y := r.top; y < r.bottom; y++ {
		for x := r.left; x < r.right; x++ {
			s := px[((y-f.rect.top)*fw+x-f.rect.left)*4:]
			d := dst[(y*w+x)*4:]
			if f.blendSrc || s[3] == 255 {
				copy(d[:4], s[:4])
				continue
			}
			inv := 255 - int(s[3])
			for c := 0; c < 4; c++ {
				d[c] = byte(int(s[c]) + (int(d[c])*inv+127)/255)
			}
		}
	}
	if !premul {
		unpremultiply8(dst)
	}
	return true
}

func unpremultiply8(px []byte) {
	p := &pixmap{width: len(px) / 4, height: 1, format: texRGBA8, pix8: px}
	copy(px, straightRGBA8(p))
}

// jpegRGBA converts a decoded JPEG to straight RGBA8 as libjpeg-turbo
// does (see ycbcrToRGBA). num/8 is the scale of a scaled decode: the
// planes are then box-averaged, as libjpeg's DC-only (1/8) IDCT does
// (an approximation for the other scales).
func jpegRGBA(m image.Image, num int) []byte {
	y, ok := m.(*image.YCbCr)
	var hs, vs int // chroma subsampling factors
	if ok {
		switch y.SubsampleRatio {
		case image.YCbCrSubsampleRatio444:
			hs, vs = 1, 1
		case image.YCbCrSubsampleRatio422:
			hs, vs = 2, 1
		case image.YCbCrSubsampleRatio420:
			hs, vs = 2, 2
		}
	}
	if hs == 0 {
		px := straightRGBA(m)
		if num == 8 {
			return px
		}
		b := m.Bounds()
		return areaAverage(px, b.Dx(), b.Dy(), (b.Dx()*num+7)/8, (b.Dy()*num+7)/8)
	}
	w, h := y.Rect.Dx(), y.Rect.Dy()
	cw, ch := (w+hs-1)/hs, (h+vs-1)/vs
	yp, ys := y.Y, y.YStride
	cb, cr, cs := y.Cb, y.Cr, y.CStride
	if num != 8 {
		// libjpeg-turbo gives subsampled components a larger IDCT so that
		// they come out at the luma's scaled size (no upsampling).
		sw, sh := (w*num+7)/8, (h*num+7)/8
		yp = boxAverage(yp, ys, w, h, sw, sh, 8, num)
		cb = boxAverage(cb, cs, cw, ch, sw, sh, 8, num*hs)
		cr = boxAverage(cr, cs, cw, ch, sw, sh, 8, num*hs)
		w, h, cw, ch, ys, cs, hs, vs = sw, sh, sw, sh, sw, sw, 1, 1
	}
	return ycbcrToRGBA(yp, ys, w, h, cb, cr, cs, cw, ch, hs, vs)
}

// boxAverage scales a plane to dw x dh, each output sample the rounded
// mean of the den/num x den/num input samples it covers.
func boxAverage(p []byte, stride, w, h, dw, dh, den, num int) []byte {
	out := make([]byte, dw*dh)
	for y := 0; y < dh; y++ {
		y0, y1 := y*den/num, min((y+1)*den/num, h)
		for x := 0; x < dw; x++ {
			x0, x1 := x*den/num, min((x+1)*den/num, w)
			sum, n := 0, 0
			for yy := y0; yy < max(y1, y0+1); yy++ {
				for xx := x0; xx < max(x1, x0+1); xx++ {
					sum += int(p[min(yy, h-1)*stride+min(xx, w-1)])
					n++
				}
			}
			out[y*dw+x] = byte((sum + n/2) / n)
		}
	}
	return out
}

// ycbcrToRGBA is libjpeg-turbo's output stage: fancy (triangle) chroma
// upsampling for 4:2:0 and 4:2:2 (jdsample.c) and jdcolor.c's fixed-point
// YCbCr to RGB tables.
func ycbcrToRGBA(yp []byte, ys, w, h int, cbp, crp []byte, cs, cw, ch, hs, vs int) []byte {
	upsample := func(plane []byte) []int32 {
		out := make([]int32, w*h)
		at := func(cx, cy int) int32 {
			return int32(plane[min(max(cy, 0), ch-1)*cs+min(max(cx, 0), cw-1)])
		}
		row := make([]int32, 2*cw+2)
		for oy := 0; oy < h; oy++ {
			cy := oy / vs
			switch {
			case hs == 1:
				for x := 0; x < w; x++ {
					out[oy*w+x] = at(x, cy)
				}
				continue
			case vs == 1: // h2v1
				for i := 0; i < cw; i++ {
					c := at(i, cy)
					if i == 0 {
						row[0] = c
					} else {
						row[2*i] = (c*3 + at(i-1, cy) + 1) >> 2
					}
					if i == cw-1 {
						row[2*i+1] = c
					} else {
						row[2*i+1] = (c*3 + at(i+1, cy) + 2) >> 2
					}
				}
			default: // h2v2
				nb := cy - 1
				if oy%2 == 1 {
					nb = cy + 1
				}
				col := func(i int) int32 { return at(i, cy)*3 + at(i, nb) }
				for i := 0; i < cw; i++ {
					c := col(i)
					if i == 0 {
						row[0] = (c*4 + 8) >> 4
					} else {
						row[2*i] = (c*3 + col(i-1) + 8) >> 4
					}
					if i == cw-1 {
						row[2*i+1] = (c*4 + 7) >> 4
					} else {
						row[2*i+1] = (c*3 + col(i+1) + 7) >> 4
					}
				}
			}
			copy(out[oy*w:(oy+1)*w], row[:w])
		}
		return out
	}
	cb, cr := upsample(cbp), upsample(crp)
	const scaleBits = 16
	fix := func(x float64) int32 { return int32(float64(x*(1<<scaleBits)) + 0.5) }
	half := int32(1) << (scaleBits - 1)
	clamp := func(v int32) byte { return byte(min(max(v, 0), 255)) }
	out := make([]byte, w*h*4)
	for oy := 0; oy < h; oy++ {
		for ox := 0; ox < w; ox++ {
			i := oy*w + ox
			yy := int32(yp[oy*ys+ox])
			b, r := cb[i]-128, cr[i]-128
			out[4*i] = clamp(yy + (fix(1.40200)*r+half)>>scaleBits)
			out[4*i+1] = clamp(yy + (-fix(0.34414)*b+half-fix(0.71414)*r)>>scaleBits)
			out[4*i+2] = clamp(yy + (fix(1.77200)*b+half)>>scaleBits)
			out[4*i+3] = 255
		}
	}
	return out
}

// webpRGBA converts a decoded WebP to straight RGBA8 as libwebp does for a
// lossy bitstream: fancy upsampling of the 4:2:0 chroma
// (UpsampleRgbaLinePair) and VP8YUVToR/G/B's 14-bit fixed point.
func webpRGBA(m image.Image) []byte {
	var yc *image.YCbCr
	var alpha []byte
	var astride int
	switch v := m.(type) {
	case *image.NYCbCrA:
		yc, alpha, astride = &v.YCbCr, v.A, v.AStride
	case *image.YCbCr:
		yc = v
	}
	if yc == nil || yc.SubsampleRatio != image.YCbCrSubsampleRatio420 {
		return straightRGBA(m)
	}
	w, h := yc.Rect.Dx(), yc.Rect.Dy()
	cw, ch := (w+1)/2, (h+1)/2
	out := make([]byte, w*h*4)
	chroma := func(plane []byte, row int) []int32 {
		r := make([]int32, cw)
		row = min(row, ch-1)
		for i := range r {
			r[i] = int32(plane[row*yc.CStride+i])
		}
		return r
	}
	// upsampleRow is one output row of UPSAMPLE_FUNC: near is the chroma
	// row closer to it (weight 3), far the other one.
	upsampleRow := func(near, far []int32) []int32 {
		r := make([]int32, 2*cw)
		r[0] = (3*near[0] + far[0] + 2) >> 2
		for x := 1; x < cw; x++ {
			tl, t, l, c := near[x-1], near[x], far[x-1], far[x]
			avg := tl + t + l + c + 8
			diag12 := (avg + 2*(t+l)) >> 3
			diag03 := (avg + 2*(tl+c)) >> 3
			r[2*x-1] = (diag12 + tl) >> 1
			r[2*x] = (diag03 + t) >> 1
		}
		if w&1 == 0 {
			r[w-1] = (3*near[cw-1] + far[cw-1] + 2) >> 2
		}
		return r
	}
	clip8 := func(v int32) byte {
		if v&^16383 == 0 {
			return byte(v >> 6)
		}
		if v < 0 {
			return 0
		}
		return 255
	}
	multHi := func(v, c int32) int32 { return (v * c) >> 8 }
	for y := 0; y < h; y++ {
		var nearRow, farRow int
		switch {
		case y == 0:
			nearRow, farRow = 0, 0
		case y%2 == 1:
			nearRow, farRow = (y-1)/2, (y+1)/2
		default:
			nearRow, farRow = y/2, y/2-1
		}
		if (y+1)/2 >= ch { // the last row of an even height
			nearRow, farRow = ch-1, ch-1
		}
		us := upsampleRow(chroma(yc.Cb, nearRow), chroma(yc.Cb, farRow))
		vs := upsampleRow(chroma(yc.Cr, nearRow), chroma(yc.Cr, farRow))
		for x := 0; x < w; x++ {
			yy := multHi(int32(yc.Y[y*yc.YStride+x]), 19077)
			u, v := us[x], vs[x]
			i := (y*w + x) * 4
			out[i] = clip8(yy + multHi(v, 26149) - 14234)
			out[i+1] = clip8(yy - multHi(u, 6419) - multHi(v, 13320) + 8708)
			out[i+2] = clip8(yy + multHi(u, 33050) - 17685)
			out[i+3] = 255
			if alpha != nil {
				out[i+3] = alpha[y*astride+x]
			}
		}
	}
	return out
}

// webpPremultiply is libwebp's WebPApplyAlphaMultiply on arm64 (NEON):
// (p + (p >> 8) + 1) >> 8 with p = c * a.
func webpPremultiply(px []byte) {
	for i := 0; i+3 < len(px); i += 4 {
		a := int(px[i+3])
		if a == 255 {
			continue
		}
		for c := 0; c < 3; c++ {
			p := int(px[i+c]) * a
			px[i+c] = byte((p + p>>8 + 1) >> 8)
		}
	}
}
