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

// UnsupportedError is dart:core's UnsupportedError(message).
type UnsupportedError struct{ Message string }

func (e UnsupportedError) Error() string { return "Unsupported operation: " + e.Message }

// TruncDiv is Dart's double ~/ operator converted back with toDouble():
// the quotient truncated toward zero, as the int the VM makes of it. An
// infinite or NaN quotient panics with UnsupportedError("Infinity or NaN
// toInt"), as the VM throws; a finite quotient outside int64 saturates to
// int64's bounds (9223372036854775807 converts back to 2^63), as the VM's
// toInt clamps (checked against the 3.50.1 VM, unoptimized and optimized).
// An int has no -0, so a zero quotient is +0.0.
func TruncDiv(a, b float64) float64 {
	q := math.Trunc(a / b)
	switch {
	case math.IsNaN(q) || math.IsInf(q, 0):
		panic(UnsupportedError{"Infinity or NaN toInt"})
	case q >= 9223372036854775808.0:
		return 9223372036854775808.0
	case q <= -9223372036854775808.0:
		return -9223372036854775808.0
	case q == 0:
		return 0 // (-0.5 ~/ 1).toDouble() is 0.0
	}
	return q
}

// UnimplementedError is dart:core's UnimplementedError(message).
type UnimplementedError struct{ Message string }

func (e UnimplementedError) Error() string {
	if e.Message == "" {
		return "UnimplementedError"
	}
	return "UnimplementedError: " + e.Message
}
