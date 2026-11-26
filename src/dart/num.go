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
// the VM's `(a / b).truncate()`, an int. It panics with UnsupportedError
// ("Infinity or NaN toInt") when the quotient is infinite or NaN, as Dart
// throws; a quotient outside int64 clamps to it (the VM's toInt), and a
// zero is +0.0 (an int has no sign).
func TruncDiv(a, b float64) float64 {
	q := a / b
	if math.IsNaN(q) || math.IsInf(q, 0) {
		panic(UnsupportedError{"Infinity or NaN toInt"})
	}
	q = math.Trunc(q)
	switch {
	case q >= 9223372036854775808.0:
		return float64(math.MaxInt64)
	case q < -9223372036854775808.0:
		return float64(math.MinInt64)
	}
	return float64(int64(q))
}

// UnimplementedError is dart:core's UnimplementedError(message).
type UnimplementedError struct{ Message string }

func (e UnimplementedError) Error() string {
	if e.Message == "" {
		return "UnimplementedError"
	}
	return "UnimplementedError: " + e.Message
}
