package dart

// #include <math.h>
// #cgo LDFLAGS: -lm
import "C"

import "math"

// Dart's dart:math functions on doubles call the C library (libm). Go's
// math package can differ from it in the last bit (for example
// pow(0.24170616113744078, 2.4)), so these call the same C functions to
// stay bit-identical with Flutter on the same platform. sqrt is exact in
// both and needs no wrapper.

// Pow is dart:math's pow on doubles. The VM handles a few exponents itself
// before calling the C library (observed against Dart 3.50.1: pow(0.3, 3.0)
// is 0.3*0.3*0.3 = 0.027, while C's pow gives 0.026999999999999996).
func Pow(x, y float64) float64 {
	switch {
	case y == 0:
		return 1
	case y == 1:
		return x
	case y == 2:
		return x * x
	case y == 3:
		return x * x * x
	case x == 1:
		return 1
	case math.IsNaN(x) || math.IsNaN(y):
		return math.NaN()
	case !math.IsInf(x, -1) && y == 0.5:
		if x == 0 {
			return 0
		}
		return math.Sqrt(x)
	}
	return float64(C.pow(C.double(x), C.double(y)))
}
func Exp(x float64) float64      { return float64(C.exp(C.double(x))) }
func Log(x float64) float64      { return float64(C.log(C.double(x))) }
func Sin(x float64) float64      { return float64(C.sin(C.double(x))) }
func Cos(x float64) float64      { return float64(C.cos(C.double(x))) }
func Tan(x float64) float64      { return float64(C.tan(C.double(x))) }
func Asin(x float64) float64     { return float64(C.asin(C.double(x))) }
func Acos(x float64) float64     { return float64(C.acos(C.double(x))) }
func Atan(x float64) float64     { return float64(C.atan(C.double(x))) }
func Atan2(y, x float64) float64 { return float64(C.atan2(C.double(y), C.double(x))) }
