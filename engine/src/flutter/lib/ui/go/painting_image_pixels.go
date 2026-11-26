package ui

// Pixel conversions of dart:ui's images, reproducing the engine's: Skia's
// raster pipeline (SkConvertPixels: from_byte, premul, unpremul,
// to_unorm with round-to-nearest-even as on arm64 NEON, half floats),
// SkOpts' 8-bit premultiplication, the GPU's bilinear resize and the PNG
// encoding of Image.toByteData.

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"math"
)

// pixmap is the pixel content of an image (the texture's contents).
type pixmap struct {
	width, height int
	format        texFormat
	pix8          []byte    // texRGBA8
	pixF          []float32 // the float formats (4 per pixel, 1 for texRF32)
}

const inv255 = float32(1.0 / 255)

// fromByte is the pipeline's from_byte: v * (1/255.0f).
func fromByte(b byte) float32 { return float32(b) * inv255 }

// toUnorm8 is the pipeline's to_unorm(v, 255): clamp to [0, 1], scale and
// round to nearest even (vcvtnq_u32_f32).
func toUnorm8(v float32) byte {
	if !(v > 0) { // NaN too: max(0, NaN) is 0 in the pipeline
		v = 0
	}
	if v > 1 {
		v = 1
	}
	x := float32(v * 255)
	return byte(math.RoundToEven(float64(x)))
}

// unpremulScale is the pipeline's unpremul scale: 1/a, or 0 when that is
// not finite.
func unpremulScale(a float32) float32 {
	s := 1 / a
	if s < float32(math.Inf(1)) {
		return s
	}
	return 0
}

// premul8 is SkOpts::RGBA_to_rgbA's rounding: (c*a + 127) / 255.
func premul8(c, a byte) byte { return byte((int(c)*int(a) + 127) / 255) }

// premultiply8 premultiplies straight RGBA8 pixels in place.
func premultiply8(px []byte) {
	for i := 0; i+3 < len(px); i += 4 {
		a := px[i+3]
		if a == 255 {
			continue
		}
		px[i] = premul8(px[i], a)
		px[i+1] = premul8(px[i+1], a)
		px[i+2] = premul8(px[i+2], a)
	}
}

// float32ToHalf converts to IEEE half precision with round to nearest even
// (vcvt_f16_f32).
func float32ToHalf(f float32) uint16 {
	b := math.Float32bits(f)
	sign := uint16(b>>16) & 0x8000
	exp := int(b>>23) & 0xff
	mant := b & 0x7fffff
	if exp == 0xff {
		if mant != 0 {
			return sign | 0x7e00
		}
		return sign | 0x7c00
	}
	e := exp - 127 + 15
	if e >= 0x1f {
		return sign | 0x7c00
	}
	if e <= 0 {
		if e < -10 {
			return sign
		}
		m := mant | 0x800000
		shift := uint(14 - e)
		h := m >> shift
		rem := m & (1<<shift - 1)
		half := uint32(1) << (shift - 1)
		if rem > half || (rem == half && h&1 == 1) {
			h++
		}
		return sign | uint16(h)
	}
	h := uint32(e)<<10 | mant>>13
	rem := mant & 0x1fff
	if rem > 0x1000 || (rem == 0x1000 && h&1 == 1) {
		h++ // may carry into the exponent, which is right
	}
	return sign | uint16(h)
}

// halfToFloat32 converts an IEEE half to float32 (exactly).
func halfToFloat32(h uint16) float32 {
	sign := uint32(h&0x8000) << 16
	exp := int(h>>10) & 0x1f
	mant := uint32(h & 0x3ff)
	switch {
	case exp == 0x1f:
		return math.Float32frombits(sign | 0x7f800000 | mant<<13)
	case exp == 0:
		if mant == 0 {
			return math.Float32frombits(sign)
		}
		v := float32(mant) / 16777216 // mant * 2^-24
		if sign != 0 {
			return -v
		}
		return v
	}
	return math.Float32frombits(sign | uint32(exp-15+127)<<23 | mant<<13)
}

// quantizeHalf rounds a float32 to the nearest half.
func quantizeHalf(f float32) float32 { return halfToFloat32(float32ToHalf(f)) }

// floatsOf returns the image's RGBA pixels as premultiplied floats (the
// pipeline's load_8888 / load_f16 / load_f32).
func floatsOf(p *pixmap) []float32 {
	if p.format != texRGBA8 {
		return p.pixF
	}
	out := make([]float32, len(p.pix8))
	for i, b := range p.pix8 {
		out[i] = fromByte(b)
	}
	return out
}

// encodeImage is EncodeImage (image_encoding.cc) after Impeller's
// readback: the texture's pixels as premultiplied pixels of its format,
// converted to the requested format. It returns the bytes or an error
// message.
func encodeImage(p *pixmap, format ImageByteFormat) ([]byte, string) {
	if p.format == texRF32 {
		return nil, "Failed to get color type from pixel format."
	}
	if p.width == 0 || p.height == 0 || (p.pix8 == nil && p.pixF == nil) {
		return nil, "Image dimensions were empty."
	}
	switch format {
	case ImageByteFormatRawRgba:
		if p.format == texRGBA8 {
			return append([]byte(nil), p.pix8...), ""
		}
		out := make([]byte, len(p.pixF))
		for i, v := range p.pixF {
			out[i] = toUnorm8(v)
		}
		return out, ""
	case ImageByteFormatRawStraightRgba:
		return straightRGBA8(p), ""
	case ImageByteFormatRawUnmodified:
		switch p.format {
		case texRGBA8:
			return append([]byte(nil), p.pix8...), ""
		case texRGBAF16:
			out := make([]byte, 2*len(p.pixF))
			for i, v := range p.pixF {
				binary.LittleEndian.PutUint16(out[2*i:], float32ToHalf(v))
			}
			return out, ""
		default:
			return float32Bytes(p.pixF), ""
		}
	case ImageByteFormatRawExtendedRgba128:
		f := floatsOf(p)
		out := make([]float32, len(f))
		for i := 0; i+3 < len(f); i += 4 {
			s := unpremulScale(f[i+3])
			out[i], out[i+1], out[i+2], out[i+3] = float32(f[i]*s), float32(f[i+1]*s), float32(f[i+2]*s), f[i+3]
		}
		return float32Bytes(out), ""
	case ImageByteFormatPng:
		return encodePNG(p)
	}
	return nil, "Unknown error encoding image."
}

// straightRGBA8 converts premultiplied pixels to unpremultiplied RGBA8
// (load, unpremul, store_8888).
func straightRGBA8(p *pixmap) []byte {
	f := floatsOf(p)
	out := make([]byte, len(f))
	for i := 0; i+3 < len(f); i += 4 {
		s := unpremulScale(f[i+3])
		out[i] = toUnorm8(f[i] * s)
		out[i+1] = toUnorm8(f[i+1] * s)
		out[i+2] = toUnorm8(f[i+2] * s)
		out[i+3] = toUnorm8(f[i+3])
	}
	return out
}

func float32Bytes(f []float32) []byte {
	out := make([]byte, 4*len(f))
	for i, v := range f {
		binary.LittleEndian.PutUint32(out[4*i:], math.Float32bits(v))
	}
	return out
}

// encodePNG is SkPngEncoder on the readback image: unpremultiplied RGBA,
// 8 bits per channel for 8-bit textures and 16 for float ones. ENGINE GAP:
// the pixels are the same, the bytes are Go's encoder's (filters and zlib
// stream differ from libpng/zlib's).
func encodePNG(p *pixmap) ([]byte, string) {
	var buf bytes.Buffer
	var err error
	if p.format == texRGBA8 {
		m := &image.NRGBA{Pix: straightRGBA8(p), Stride: 4 * p.width, Rect: image.Rect(0, 0, p.width, p.height)}
		err = (&png.Encoder{CompressionLevel: png.DefaultCompression}).Encode(&buf, nrgbaAlways{m})
	} else {
		m := image.NewNRGBA64(image.Rect(0, 0, p.width, p.height))
		f := p.pixF
		for i := 0; i+3 < len(f); i += 4 {
			s := unpremulScale(f[i+3])
			for c := 0; c < 4; c++ {
				v := f[i+c]
				if c < 3 {
					v = float32(v * s)
				}
				binary.BigEndian.PutUint16(m.Pix[2*(i+c):], toUnorm16(v))
			}
		}
		err = png.Encode(&buf, m)
	}
	if err != nil {
		return nil, "Could not convert raster image to PNG."
	}
	return buf.Bytes(), ""
}

func toUnorm16(v float32) uint16 {
	if !(v > 0) {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return uint16(math.RoundToEven(float64(v * 65535)))
}

// nrgbaAlways hides Opaque so that Go's encoder writes RGBA, as Skia's
// does for a premultiplied image.
type nrgbaAlways struct{ *image.NRGBA }

func (nrgbaAlways) Opaque() bool { return false }

// resizeBilinear is the GPU texture resize (the blit pass's
// ResizeTexture, Metal's bilinear scale): bilinear sampling at pixel
// centers with clamped edges, on premultiplied values; 8-bit results are
// rounded to nearest even from the exactly interpolated byte values.
func resizeBilinear(p *pixmap, w, h int) *pixmap {
	ch := 4
	if p.format == texRF32 {
		ch = 1
	}
	src := make([]float64, p.width*p.height*ch)
	if p.format == texRGBA8 {
		for i, b := range p.pix8 {
			src[i] = float64(b)
		}
	} else {
		for i, v := range p.pixF {
			src[i] = float64(v)
		}
	}
	out := make([]float64, w*h*ch)
	sx := float64(p.width) / float64(w)
	sy := float64(p.height) / float64(h)
	for y := 0; y < h; y++ {
		fy := float64((float64(y)+0.5)*sy) - 0.5
		y0, y1, ty := bilinearTaps(fy, p.height)
		for x := 0; x < w; x++ {
			fx := float64((float64(x)+0.5)*sx) - 0.5
			x0, x1, tx := bilinearTaps(fx, p.width)
			for c := 0; c < ch; c++ {
				a := src[(y0*p.width+x0)*ch+c]
				b := src[(y0*p.width+x1)*ch+c]
				cc := src[(y1*p.width+x0)*ch+c]
				d := src[(y1*p.width+x1)*ch+c]
				top := float64(a*(1-tx)) + float64(b*tx)
				bot := float64(cc*(1-tx)) + float64(d*tx)
				out[(y*w+x)*ch+c] = float64(top*(1-ty)) + float64(bot*ty)
			}
		}
	}
	r := &pixmap{width: w, height: h, format: p.format}
	if p.format == texRGBA8 {
		r.pix8 = make([]byte, len(out))
		for i, v := range out {
			r.pix8[i] = byte(math.RoundToEven(math.Max(0, math.Min(255, v))))
		}
		return r
	}
	r.pixF = make([]float32, len(out))
	for i, v := range out {
		r.pixF[i] = float32(v)
		if p.format == texRGBAF16 {
			r.pixF[i] = quantizeHalf(r.pixF[i])
		}
	}
	return r
}

func bilinearTaps(f float64, n int) (i0, i1 int, t float64) {
	if f <= 0 {
		return 0, 0, 0
	}
	fl := math.Floor(f)
	i0 = int(fl)
	t = f - fl
	i1 = i0 + 1
	if i0 >= n-1 {
		return n - 1, n - 1, 0
	}
	return i0, i1, t
}
