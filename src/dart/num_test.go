package dart

import (
	"math"
	"testing"
)

// Expected values are what Dart 3.50.1's SDK prints.
func TestMod(t *testing.T) {
	inf := math.Inf(1)
	for _, c := range [][3]float64{{5.5, 2, 1.5}, {-5.5, 2, 0.5}, {5.5, -2, 1.5}, {-5.5, -2, 0.5}, {math.Copysign(0, -1), 3, 0},
		{3, inf, 3}, {-3, inf, inf}} {
		if got := Mod(c[0], c[1]); got != c[2] || math.Signbit(got) {
			t.Errorf("Mod(%v, %v) = %v, want %v", c[0], c[1], got, c[2])
		}
	}
	for _, c := range [][2]float64{{7, 0}, {inf, 2}} {
		if got := Mod(c[0], c[1]); !math.IsNaN(got) {
			t.Errorf("Mod(%v, %v) = %v, want NaN", c[0], c[1], got)
		}
	}
}

func TestTruncDiv(t *testing.T) {
	// Values checked against the Dart VM (`(a ~/ b).toDouble()`).
	for _, c := range [][3]float64{{5.5, 2, 2}, {-5.5, 2, -2}, {7, -2, -3},
		{-1e300, 1, -9223372036854775808.0}, {9.3e18, 1, 9223372036854775808.0}} {
		if got := TruncDiv(c[0], c[1]); got != c[2] {
			t.Errorf("TruncDiv(%v, %v) = %v, want %v", c[0], c[1], got, c[2])
		}
	}
	for _, c := range [][2]float64{{-0.25, 2}, {math.Copysign(0, -1), 1}} {
		if got := TruncDiv(c[0], c[1]); got != 0 || math.Signbit(got) {
			t.Errorf("TruncDiv(%v, %v) = %v, want +0.0", c[0], c[1], got)
		}
	}
	for _, c := range [][2]float64{{1e300, 1e-10}, {1, 0}, {math.NaN(), 1}, {math.Inf(1), 1}} {
		func() {
			defer func() {
				if e, ok := recover().(UnsupportedError); !ok || e.Error() != "Unsupported operation: Infinity or NaN toInt" {
					t.Errorf("TruncDiv(%v, %v): want UnsupportedError panic", c[0], c[1])
				}
			}()
			TruncDiv(c[0], c[1])
		}()
	}
}
