package dart

import "math"

// Mod is Dart's double % operator: the Euclidean remainder, never negative
// (unlike math.Mod, which takes the dividend's sign).
func Mod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r == 0 {
		return 0 // Dart returns +0.0, even for -0.0 % b
	}
	if r < 0 {
		r += math.Abs(b)
	}
	return r
}

// UnsupportedError is what Dart throws for an integer division whose result
// isn't a finite 64-bit integer.
type UnsupportedError struct{ Message string }

func (e UnsupportedError) Error() string { return "Unsupported operation: " + e.Message }

// TruncDiv is Dart's double ~/ operator converted back with toDouble():
// the quotient truncated toward zero. It panics with UnsupportedError when
// the quotient is infinite, NaN or outside int64, as Dart throws.
func TruncDiv(a, b float64) float64 {
	q := math.Trunc(a / b)
	if math.IsNaN(q) || math.IsInf(q, 0) || q >= 9223372036854775808.0 || q < -9223372036854775808.0 {
		panic(UnsupportedError{"Result of truncating division is " + DoubleToString(a/b) + ": " +
			DoubleToString(a) + " ~/ " + DoubleToString(b)})
	}
	return q
}
