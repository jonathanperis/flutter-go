package ui

// Tests of engine behaviors of dart:ui's image codecs that neither the
// engine's Dart tests nor the image_codec difftest's fixtures reach: the
// rounding rules of the pixel conversions, GIF interlacing, APNG frame
// timing, blending and bounds, PNG validity, 16-bit PNGs and the EXIF
// orientations of JPEGs. Inputs are built here (or patched from the
// engine's fixtures); the expected values follow the engine's C++ code.

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"strings"
	"testing"
)

func TestPremul8RoundsToNearest(t *testing.T) {
	for c := 0; c < 256; c++ {
		for a := 0; a < 256; a++ {
			want := byte(math.Floor(float64(c*a)/255 + 0.5))
			if got := premul8(byte(c), byte(a)); got != want {
				t.Fatalf("premul8(%d, %d) = %d, want %d", c, a, got, want)
			}
		}
	}
}

func TestToUnorm8(t *testing.T) {
	for _, c := range []struct {
		v    float32
		want byte
	}{
		{float32(math.NaN()), 0}, {-1, 0}, {2, 255}, {0.5, 128}, {float32(0.5) / 255 * 3, 2},
		{float32(76.5) / 255, 76}, {float32(77.5) / 255, 78},
	} {
		if got := toUnorm8(c.v); got != c.want {
			t.Errorf("toUnorm8(%v) = %d, want %d", c.v, got, c.want)
		}
	}
}

// patchPNGChunk rewrites the n-th chunk of type typ through edit and fixes
// its CRC.
func patchPNGChunk(t *testing.T, data []byte, typ string, n int, edit func(body []byte)) []byte {
	t.Helper()
	out := append([]byte(nil), data...)
	pos := 8
	for pos+8 <= len(out) {
		l := int(binary.BigEndian.Uint32(out[pos:]))
		if string(out[pos+4:pos+8]) == typ {
			if n == 0 {
				edit(out[pos+8 : pos+8+l])
				binary.BigEndian.PutUint32(out[pos+8+l:], crc32.ChecksumIEEE(out[pos+4:pos+8+l]))
				return out
			}
			n--
		}
		pos += 12 + l
	}
	t.Fatalf("no %s chunk", typ)
	return nil
}

func TestGIFInterlaced(t *testing.T) {
	imgLoop(t)
	pal := color.Palette{}
	for i := 0; i < 8; i++ {
		pal = append(pal, color.RGBA{uint8(i * 30), 0, 0, 255})
	}
	m := image.NewPaletted(image.Rect(0, 0, 3, 8), pal)
	for y := 0; y < 8; y++ {
		for x := 0; x < 3; x++ {
			m.SetColorIndex(x, y, uint8(y)) // stored row k has color k
		}
	}
	var buf bytes.Buffer
	if err := gif.Encode(&buf, m, nil); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	pos := 13 + 3*(2<<(data[10]&7))
	for data[pos] == 0x21 { // skip extensions
		pos += 2
		for data[pos] != 0 {
			pos += 1 + int(data[pos])
		}
		pos++
	}
	if data[pos] != 0x2c {
		t.Fatal("no image descriptor")
	}
	data[pos+9] |= 0x40
	img := imgFrame(t, data).Image
	px := imgBytes(t, img, ImageByteFormatRawRgba)
	order := []int{0, 4, 2, 6, 1, 3, 5, 7}
	for k, row := range order {
		if got := px[row*3*4]; got != uint8(k*30) {
			t.Errorf("row %d: red %d, want %d (stored row %d)", row, got, k*30, k)
		}
	}
}

func TestAPNGDurationIsComputedInFloat(t *testing.T) {
	imgLoop(t)
	data := patchPNGChunk(t, imgFixture(t, "alpha_animated.apng"), "fcTL", 0, func(b []byte) {
		binary.BigEndian.PutUint16(b[20:], 65533)
		binary.BigEndian.PutUint16(b[22:], 3)
	})
	codec := imgCodec(t, data)
	fi := imgAwait(t, codec.GetNextFrame())
	// 65533 * 1000.f / 3 is 21844333.33, which rounds to 21844334 in float.
	if got := fi.Duration.Milliseconds(); got != 21844334 {
		t.Errorf("duration %d ms", got)
	}
}

func TestAPNGBlendOverTruncates(t *testing.T) {
	imgLoop(t)
	data := patchPNGChunk(t, imgFixture(t, "alpha_animated.apng"), "tRNS", 0, func(b []byte) { b[1] = 0x34 })
	data = patchPNGChunk(t, data, "fcTL", 1, func(b []byte) { b[25] = 1 })
	codec := imgCodec(t, data)
	first := imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	second := imgBytes(t, imgAwait(t, codec.GetNextFrame()).Image, ImageByteFormatRawRgba)
	// 255*153/255 + 52*(255-153)/255 = 153 + 20 (integer division).
	if !bytes.Equal(first[:4], []byte{52, 0, 0, 52}) || !bytes.Equal(second[:4], []byte{173, 0, 0, 173}) {
		t.Errorf("first %v, second %v", first[:4], second[:4])
	}
}

func TestAPNGFrameOutsideCanvas(t *testing.T) {
	imgLoop(t)
	data := patchPNGChunk(t, imgFixture(t, "out_of_bounds.apng"), "fcTL", 0, func(b []byte) {
		binary.BigEndian.PutUint32(b[16:], 1) // y offset 1: the 2x2 frame sticks out by a row
	})
	codec := imgCodec(t, data)
	if _, err := imgResult(codec.GetNextFrame()); err == nil || !strings.Contains(err.Error(), "Could not decompress image.") {
		t.Errorf("got %v", err)
	}
}

func TestPNGWithoutIDATIsInvalid(t *testing.T) {
	imgLoop(t)
	data := append([]byte(nil), pngSignature...)
	data = appendChunk(data, "IHDR", []byte{0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0})
	data = appendChunk(data, "IEND", nil)
	if _, err := imgResult(InstantiateImageCodec(data)); err == nil || !strings.Contains(err.Error(), "Invalid image data") {
		t.Errorf("got %v", err)
	}
}

func TestPNG16BitChannels(t *testing.T) {
	imgLoop(t)
	m := image.NewNRGBA64(image.Rect(0, 0, 1, 1))
	// 25854/65535*255 = 100.6 (rounds up), 25600/65535*255 = 99.6, 0x7fff = 127.498.
	m.SetNRGBA64(0, 0, color.NRGBA64{25854, 25600, 0x7fff, 0xffff})
	var buf bytes.Buffer
	if err := png.Encode(&buf, m); err != nil {
		t.Fatal(err)
	}
	got := imgBytes(t, imgFrame(t, buf.Bytes()).Image, ImageByteFormatRawRgba)
	if !bytes.Equal(got, []byte{101, 100, 127, 255}) {
		t.Errorf("got %v", got)
	}
}

// jpegWithOrientation encodes m as a JPEG carrying an EXIF orientation.
func jpegWithOrientation(t *testing.T, m image.Image, orientation int) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, m, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x01\x01\x12\x00\x03\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00")
	tiff[19] = byte(orientation)
	app1 := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(app1)+2))
	data := buf.Bytes()
	out := append([]byte{}, data[:2]...)
	out = append(out, seg...)
	out = append(out, app1...)
	return append(out, data[2:]...)
}

func TestJPEGOrientations(t *testing.T) {
	imgLoop(t)
	const w, h = 24, 16
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.Set(x, y, color.RGBA{uint8(x * 10), uint8(y * 15), 128, 255})
		}
	}
	base := imgBytes(t, imgFrame(t, jpegWithOrientation(t, m, 1)).Image, ImageByteFormatRawRgba)
	// Where the EXIF orientation puts the source pixel (sx, sy).
	dest := map[int]func(sx, sy int) (int, int){
		2: func(sx, sy int) (int, int) { return w - 1 - sx, sy },
		3: func(sx, sy int) (int, int) { return w - 1 - sx, h - 1 - sy },
		4: func(sx, sy int) (int, int) { return sx, h - 1 - sy },
		5: func(sx, sy int) (int, int) { return sy, sx },
		6: func(sx, sy int) (int, int) { return h - 1 - sy, sx },
		7: func(sx, sy int) (int, int) { return h - 1 - sy, w - 1 - sx },
		8: func(sx, sy int) (int, int) { return sy, w - 1 - sx },
	}
	for o := 2; o <= 8; o++ {
		img := imgFrame(t, jpegWithOrientation(t, m, o)).Image
		ow := w
		if o >= 5 {
			ow = h
		}
		if img.Width != ow || img.Width*img.Height != w*h {
			t.Errorf("orientation %d: %dx%d", o, img.Width, img.Height)
			continue
		}
		px := imgBytes(t, img, ImageByteFormatRawRgba)
	check:
		for sy := 0; sy < h; sy++ {
			for sx := 0; sx < w; sx++ {
				dx, dy := dest[o](sx, sy)
				if !bytes.Equal(px[(dy*ow+dx)*4:][:4], base[(sy*w+sx)*4:][:4]) {
					t.Errorf("orientation %d: source pixel (%d, %d) isn't at (%d, %d)", o, sx, sy, dx, dy)
					break check
				}
			}
		}
	}
}
