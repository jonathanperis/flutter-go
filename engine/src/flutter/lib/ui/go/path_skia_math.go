package ui

// Scalar and point helpers shared by the Skia ports (path_skia_*.go). They
// port include/core/SkScalar.h, include/private/SkFloatingPoint.h,
// include/core/SkPoint.h and src/core/SkPoint.cpp (Skia at the revision
// Flutter 3.50.1 pins), and lib/ui/floating_point.h's SafeNarrow.
//
// Exactness. The engine and Skia compute in float (32-bit) with clang's
// default -ffp-contract=on: inside one C++ expression `a*b + c` is a fused
// multiply-add (the left product first: `a*b + c*d` is fma(a, b, c*d)),
// and separate statements are rounded separately. The reference binary
// (flutter_tester, arm64) was disassembled to confirm it. These ports
// write every fused expression with fma32 (or math.FMA for doubles) and
// round every other product explicitly (float32(a*b)), so Go never fuses
// where C++ doesn't. skvx vector code (float2/float4 operators) is never
// fused: each operator is its own function. Single-precision libm calls
// (sinf, cosf, tanf, atan2f, powf, asinf) go to the C library through cgo,
// as the engine calls them.

// #include <math.h>
// #cgo LDFLAGS: -lm
import "C"

import "math"

func cosf(x float32) float32       { return float32(C.cosf(C.float(x))) }
func tanf(x float32) float32       { return float32(C.tanf(C.float(x))) }
func asinf(x float32) float32      { return float32(C.asinf(C.float(x))) }
func atan2f(y, x float32) float32  { return float32(C.atan2f(C.float(y), C.float(x))) }
func powf(x, y float32) float32    { return float32(C.powf(C.float(x), C.float(y))) }
func sqrtf(x float32) float32      { return float32(math.Sqrt(float64(x))) } // correctly rounded either way
func fabsf(x float32) float32      { return math.Float32frombits(math.Float32bits(x) &^ (1 << 31)) }
func floorf(x float32) float32     { return float32(math.Floor(float64(x))) }
func ceilf(x float32) float32      { return float32(math.Ceil(float64(x))) }
func fmodf(x, y float32) float32   { return float32(math.Mod(float64(x), float64(y))) } // exact
func isFinite32(x float32) bool    { return !math.IsInf(float64(x), 0) && !math.IsNaN(float64(x)) }
func mul32(a, b float32) float32   { return float32(a * b) }
func copysignf(x, y float32) float32 { return float32(math.Copysign(float64(x), float64(y))) }

// sincosf is Apple's __sincosf_stret, which clang calls whenever one
// function takes sinf and cosf of the same value (SkMatrix::setRotate,
// angles_to_unit_vectors, the SVG arc loop, impeller's Point::Rotate). It
// differs from separate sinf/cosf calls in the last bit for about 3% of
// inputs, so the ports call it exactly where the binary does.
func sincosf(x float32) (s, c float32) {
	r := C.__sincosf_stret(C.float(x))
	return float32(r.__sinval), float32(r.__cosval)
}

// fma32 is C's fmaf: a*b + c rounded once to float. The product is exact in
// double; the sum is rounded to double with round-to-odd (TwoSum gives the
// exact error), which makes the final rounding to float correct.
func fma32(a, b, c float32) float32 {
	p := float64(float64(a) * float64(b))
	cc := float64(c)
	s := p + cc
	if math.IsInf(s, 0) || math.IsNaN(s) {
		return float32(s)
	}
	bv := s - p
	e := (p - (s - bv)) + (cc - bv)
	if e != 0 {
		bits := math.Float64bits(s)
		if bits&1 == 0 {
			if (e > 0) == (s > 0) {
				bits++
			} else {
				bits--
			}
			s = math.Float64frombits(bits)
		}
	}
	return float32(s)
}

// fminf/fmaxf are C's fminf/fmaxf (NaN-ignoring; -0 < +0 as arm64's fminnm).
func fminf(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if a == b {
		if math.Signbit(float64(a)) {
			return a
		}
		return b
	}
	if a < b {
		return a
	}
	return b
}

func fmaxf(a, b float32) float32 {
	if a != a {
		return b
	}
	if b != b {
		return a
	}
	if a == b {
		if math.Signbit(float64(a)) {
			return b
		}
		return a
	}
	if a > b {
		return a
	}
	return b
}

// stdMin/stdMax are std::min/std::max: (b < a) ? b : a and (a < b) ? b : a.
func stdMin(a, b float32) float32 {
	if b < a {
		return b
	}
	return a
}

func stdMax(a, b float32) float32 {
	if a < b {
		return b
	}
	return a
}

// Skia scalar constants, computed in float step by step as C++ folds them
// (see init).
var (
	skFloatPI          = float32(3.14159265)
	skScalarRoot2Over2 = float32(0.707106781)
	skNearlyZero       = float32(1.0 / 4096)  // SK_ScalarNearlyZero, exact
	skSinCosNearlyZero = float32(1.0 / 65536) // SK_ScalarSinCosNearlyZero, exact
	mPIf               = float32(math.Pi)     // static_cast<float>(M_PI)

	skPIOver180     float32 // SK_ScalarPI / 180
	skTwoPI         float32 // SK_ScalarPI * 2
	skTwoPIOver3    float32 // 2 * SK_ScalarPI / 3
	skPIOverMillion float32 // SK_ScalarPI / (1000 * 1000)
	skPIOver2       float32 // SK_ScalarPI / 2
	skOneThird      float32 // SK_Scalar1 / 3
	skTwoThirds     float32 // SK_Scalar1 * 2 / 3
)

func init() {
	one, two, three := float32(1), float32(2), float32(3)
	skPIOver180 = skFloatPI / float32(180)
	skTwoPI = float32(skFloatPI * two)
	skTwoPIOver3 = float32(two*skFloatPI) / three
	skPIOverMillion = skFloatPI / float32(1000000)
	skPIOver2 = skFloatPI / two
	skOneThird = one / three
	skTwoThirds = float32(one*two) / three
}

func skNearlyZeroTol(x, tol float32) bool { return fabsf(x) <= tol }
func skScalarNearlyZero(x float32) bool   { return fabsf(x) <= skNearlyZero }
func skScalarNearlyEqual(x, y float32) bool {
	return fabsf(x-y) <= skNearlyZero
}

func skScalarSignAsInt(x float32) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

func skDegreesToRadians(deg float32) float32 { return float32(deg * skPIOver180) }

// skSinCosSnapToZero is SkScalarSinSnapToZero(rad) and
// SkScalarCosSnapToZero(rad) (one sincosf call, as compiled).
func skSinCosSnapToZero(rad float32) (float32, float32) {
	s, c := sincosf(rad)
	return skSnapToZero(s), skSnapToZero(c)
}

func skSnapToZero(v float32) float32 {
	if skNearlyZeroTol(v, skSinCosNearlyZero) {
		return 0
	}
	return v
}

// skScalarInterp is SkScalarInterp: A + (B - A) * t, fused.
func skScalarInterp(a, b, t float32) float32 { return fma32(b-a, t, a) }

// skFloatRound is sk_float_round: floor((double)x + 0.5).
func skFloatRound(x float32) float32 { return float32(math.Floor(float64(x) + 0.5)) }

// skFloatSaturate2Int is sk_float_saturate2int.
func skFloatSaturate2Int(x float32) int {
	const maxS32FitsInFloat = 2147483520
	if x != x {
		return maxS32FitsInFloat
	}
	if !(x < maxS32FitsInFloat) {
		x = maxS32FitsInFloat
	}
	if !(x > -maxS32FitsInFloat) {
		x = -maxS32FitsInFloat
	}
	return int(x)
}

func skScalarCeilToInt(x float32) int { return skFloatSaturate2Int(ceilf(x)) }

// safeNarrow32 is lib/ui/floating_point.h's SafeNarrow (rsuperellipse_contains.go
// has the same function as safeNarrow).
func safeNarrow32(v float64) float32 { return safeNarrow(v) }

// skPoint is SkPoint / SkVector.
type skPoint struct{ x, y float32 }

func (a skPoint) add(b skPoint) skPoint   { return skPoint{a.x + b.x, a.y + b.y} }
func (a skPoint) sub(b skPoint) skPoint   { return skPoint{a.x - b.x, a.y - b.y} }
func (a skPoint) scale(s float32) skPoint { return skPoint{mul32(a.x, s), mul32(a.y, s)} }
func (a skPoint) isFinite() bool          { return isFinite32(a.x) && isFinite32(a.y) }

// skDot and skCross are SkPoint::DotProduct/CrossProduct (fused).
func skDot(a, b skPoint) float32   { return fma32(a.x, b.x, mul32(a.y, b.y)) }
func skCross(a, b skPoint) float32 { return fma32(a.x, b.y, -mul32(a.y, b.x)) }

// skLength is SkPoint::Length.
func skLength(dx, dy float32) float32 {
	mag2 := fma32(dx, dx, mul32(dy, dy))
	if isFinite32(mag2) {
		return sqrtf(mag2)
	}
	xx, yy := float64(dx), float64(dy)
	return float32(math.Sqrt(float64(xx*xx) + float64(yy*yy)))
}

func skDistance(a, b skPoint) float32 { return skLength(a.x-b.x, a.y-b.y) }

// skSetLength is SkPoint::setLength(x, y, length) (set_point_length): the
// result and whether it succeeded ((0, 0) and false when it didn't).
func skSetLength(x, y, length float32) (skPoint, bool) {
	xx, yy := float64(x), float64(y)
	dmag := math.Sqrt(float64(xx*xx) + float64(yy*yy))
	dscale := float64(length) / dmag
	x = float32(float64(x) * dscale)
	y = float32(float64(y) * dscale)
	if !isFinite32(x) || !isFinite32(y) || (x == 0 && y == 0) {
		return skPoint{}, false
	}
	return skPoint{x, y}, true
}

func (a skPoint) normalize() skPoint { p, _ := skSetLength(a.x, a.y, 1); return p }

// skEqualsWithinTolerance is SkPointPriv::EqualsWithinTolerance(p1, p2):
// !CanNormalize(p1 - p2).
func skEqualsWithinTolerance(p1, p2 skPoint) bool {
	dx, dy := p1.x-p2.x, p1.y-p2.y
	return !(isFinite32(dx) && isFinite32(dy) && (dx != 0 || dy != 0))
}

// skRect is SkRect (left, top, right, bottom).
type skRect struct{ l, t, r, b float32 }

func (r skRect) width() float32  { return r.r - r.l }
func (r skRect) height() float32 { return r.b - r.t }
func (r skRect) isEmpty() bool   { return !(r.l < r.r && r.t < r.b) }
func (r skRect) isFinite() bool {
	return isFinite32(r.l) && isFinite32(r.t) && isFinite32(r.r) && isFinite32(r.b)
}
func (r skRect) centerX() float32 { return skFloatMidpoint(r.l, r.r) }
func (r skRect) centerY() float32 { return skFloatMidpoint(r.t, r.b) }
func (r skRect) makeSorted() skRect {
	return skRect{stdMin(r.l, r.r), stdMin(r.t, r.b), stdMax(r.l, r.r), stdMax(r.t, r.b)}
}

// skFloatMidpoint is sk_float_midpoint: (float)(0.5 * ((double)a + b)).
func skFloatMidpoint(a, b float32) float32 {
	return float32(0.5 * float64(float64(a)+float64(b)))
}

// skRectBounds is SkRect::Bounds: the bounds of pts, false if any is not
// finite. An empty span gives the empty rect.
func skRectBounds(pts []skPoint) (skRect, bool) {
	if len(pts) == 0 {
		return skRect{}, true
	}
	l, t, r, b := pts[0].x, pts[0].y, pts[0].x, pts[0].y
	var nx, ny float32
	for _, p := range pts {
		l = fminf(p.x, l)
		t = fminf(p.y, t)
		r = fmaxf(p.x, r)
		b = fmaxf(p.y, b)
		nx = mul32(nx, p.x)
		ny = mul32(ny, p.y)
	}
	if nx == 0 && ny == 0 {
		return skRect{l, t, r, b}, true
	}
	return skRect{}, false
}
