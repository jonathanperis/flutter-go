package ui

// Color management of decoded images: Impeller's decoder asks the codec
// for sRGB pixels (wide gamut is off, as in flutter_tester), and Skia
// converts images with an ICC profile through skcms. This handles the
// matrix/TRC profiles (rXYZ/gXYZ/bXYZ with 'para' or 'curv' curves) that
// PNG iCCP chunks and JPEG APP2 markers carry. ENGINE GAP: skcms
// evaluates curves with polynomial approximations of pow/log/exp; this uses
// exact pow, so a channel can differ by 1.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"math"
)

func inflate(b []byte) []byte {
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	out, _ := io.ReadAll(r)
	return out
}

// iccCurve is a skcms_Curve: a parametric transfer function or a table.
type iccCurve struct {
	g, a, b, c, d, e, f float64
	table               []uint16
}

func (c *iccCurve) eval(x float64) float64 {
	if c.table != nil {
		n := len(c.table)
		ix := float64(math.Max(0, math.Min(1, x)) * float64(n-1))
		lo := int(ix)
		hi := min(lo+1, n-1)
		t := ix - float64(lo)
		l, h := float64(c.table[lo])/65535, float64(c.table[hi])/65535
		return l + float64((h-l)*t)
	}
	sign := 1.0
	if x < 0 {
		sign, x = -1, -x
	}
	if x < c.d {
		return sign * (float64(c.c*x) + c.f)
	}
	return sign * (math.Pow(float64(c.a*x)+c.b, c.g) + c.e)
}

// a = 1/1.055, b = 0.055/1.055, c = 1/12.92 (as doubles).
var srgbCurve = iccCurve{g: 2.4, a: 0.9478672985781991, b: 0.052132701421800945, c: 0.07739938080495357, d: 0.04045}

// srgbToXYZD50 is skcms_sRGB_profile's toXYZD50.
var srgbToXYZD50 = [3][3]float64{
	{0.436065674, 0.385147095, 0.143066406},
	{0.222488403, 0.716873169, 0.060607910},
	{0.013916016, 0.097076416, 0.714096069},
}

// iccTransform converts straight RGBA8 pixels from a profile to sRGB.
type iccTransform struct {
	trc [3]iccCurve
	m   [3][3]float64 // source linear RGB to sRGB linear RGB
}

func parseICCCurve(p []byte) (iccCurve, bool) {
	if len(p) < 12 {
		return iccCurve{}, false
	}
	switch string(p[:4]) {
	case "curv":
		n := int(binary.BigEndian.Uint32(p[8:]))
		if len(p) < 12+2*n {
			return iccCurve{}, false
		}
		switch n {
		case 0:
			return iccCurve{g: 1, a: 1}, true
		case 1:
			return iccCurve{g: float64(binary.BigEndian.Uint16(p[12:])) / 256, a: 1}, true
		}
		t := make([]uint16, n)
		for i := range t {
			t[i] = binary.BigEndian.Uint16(p[12+2*i:])
		}
		return iccCurve{table: t}, true
	case "para":
		typ := binary.BigEndian.Uint16(p[8:])
		counts := [...]int{1, 3, 4, 5, 7}
		if typ > 4 || len(p) < 12+4*counts[typ] {
			return iccCurve{}, false
		}
		v := make([]float64, 7)
		for i := 0; i < counts[typ]; i++ {
			v[i] = float64(int32(binary.BigEndian.Uint32(p[12+4*i:]))) / 65536
		}
		c := iccCurve{g: v[0], a: 1}
		switch typ {
		case 1:
			c.a, c.b, c.d = v[1], v[2], -v[2]/v[1]
		case 2:
			c.a, c.b, c.e, c.d = v[1], v[2], v[3], -v[2]/v[1]
			c.f = v[3]
		case 3:
			c.a, c.b, c.c, c.d = v[1], v[2], v[3], v[4]
		case 4:
			c.a, c.b, c.c, c.d, c.e, c.f = v[1], v[2], v[3], v[4], v[5], v[6]
		}
		return c, true
	}
	return iccCurve{}, false
}

// iccTransformFromProfile parses an RGB matrix/TRC profile; nil when it
// can't, or when it is (approximately) sRGB, where Skia doesn't convert.
func iccTransformFromProfile(p []byte) *iccTransform {
	if len(p) < 132 || string(p[16:20]) != "RGB " {
		return nil
	}
	n := int(binary.BigEndian.Uint32(p[128:]))
	tags := map[string][]byte{}
	for i := 0; i < n && 144+12*i <= len(p); i++ {
		e := p[132+12*i:]
		off, size := int(binary.BigEndian.Uint32(e[4:])), int(binary.BigEndian.Uint32(e[8:]))
		if off < 0 || size < 0 || off+size > len(p) {
			return nil
		}
		tags[string(e[:4])] = p[off : off+size]
	}
	var xf iccTransform
	var toXYZ [3][3]float64
	for i, name := range []string{"r", "g", "b"} {
		x := tags[name+"XYZ"]
		if len(x) < 20 || string(x[:4]) != "XYZ " {
			return nil
		}
		for j := 0; j < 3; j++ {
			toXYZ[j][i] = float64(int32(binary.BigEndian.Uint32(x[8+4*j:]))) / 65536
		}
		c, ok := parseICCCurve(tags[name+"TRC"])
		if !ok {
			return nil
		}
		xf.trc[i] = c
	}
	inv, ok := invert3(srgbToXYZD50)
	if !ok {
		return nil
	}
	xf.m = mul3(inv, toXYZ)
	if xf.nearSRGB() {
		return nil
	}
	return &xf
}

func (xf *iccTransform) nearSRGB() bool {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			want := 0.0
			if i == j {
				want = 1
			}
			if math.Abs(xf.m[i][j]-want) > 0.01 {
				return false
			}
		}
		for k := 0; k <= 255; k += 15 {
			x := float64(k) / 255
			if math.Abs(xf.trc[i].eval(x)-srgbCurve.eval(x)) > 0.005 {
				return false
			}
		}
	}
	return true
}

// encodeSRGB is the inverse of the sRGB curve.
func encodeSRGB(x float64) float64 {
	if x <= 0.0031308 {
		return x * 12.92
	}
	return float64(1.055*math.Pow(x, 1/2.4)) - 0.055
}

// apply converts straight RGBA8 pixels in place (store: clamp, v*255 +
// 0.5 truncated, as skcms' to_fixed).
func (xf *iccTransform) apply(px []byte) {
	cache := map[[3]byte][3]byte{}
	for i := 0; i+3 < len(px); i += 4 {
		k := [3]byte{px[i], px[i+1], px[i+2]}
		out, ok := cache[k]
		if !ok {
			var lin [3]float64
			for c := 0; c < 3; c++ {
				lin[c] = xf.trc[c].eval(float64(k[c]) / 255)
			}
			for c := 0; c < 3; c++ {
				v := float64(xf.m[c][0]*lin[0]) + float64(xf.m[c][1]*lin[1]) + float64(xf.m[c][2]*lin[2])
				v = encodeSRGB(math.Max(0, math.Min(1, v)))
				out[c] = byte(math.Floor(float64(v*255) + 0.5))
			}
			cache[k] = out
		}
		px[i], px[i+1], px[i+2] = out[0], out[1], out[2]
	}
}

func mul3(a, b [3][3]float64) (m [3][3]float64) {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			m[i][j] = float64(a[i][0]*b[0][j]) + float64(a[i][1]*b[1][j]) + float64(a[i][2]*b[2][j])
		}
	}
	return m
}

func invert3(a [3][3]float64) ([3][3]float64, bool) {
	var m [3][3]float64
	det := float64(a[0][0]*(float64(a[1][1]*a[2][2])-float64(a[1][2]*a[2][1]))) -
		float64(a[0][1]*(float64(a[1][0]*a[2][2])-float64(a[1][2]*a[2][0]))) +
		float64(a[0][2]*(float64(a[1][0]*a[2][1])-float64(a[1][1]*a[2][0])))
	if det == 0 {
		return m, false
	}
	m[0][0] = (float64(a[1][1]*a[2][2]) - float64(a[1][2]*a[2][1])) / det
	m[0][1] = (float64(a[0][2]*a[2][1]) - float64(a[0][1]*a[2][2])) / det
	m[0][2] = (float64(a[0][1]*a[1][2]) - float64(a[0][2]*a[1][1])) / det
	m[1][0] = (float64(a[1][2]*a[2][0]) - float64(a[1][0]*a[2][2])) / det
	m[1][1] = (float64(a[0][0]*a[2][2]) - float64(a[0][2]*a[2][0])) / det
	m[1][2] = (float64(a[0][2]*a[1][0]) - float64(a[0][0]*a[1][2])) / det
	m[2][0] = (float64(a[1][0]*a[2][1]) - float64(a[1][1]*a[2][0])) / det
	m[2][1] = (float64(a[0][1]*a[2][0]) - float64(a[0][0]*a[2][1])) / det
	m[2][2] = (float64(a[0][0]*a[1][1]) - float64(a[0][1]*a[1][0])) / det
	return m, true
}
