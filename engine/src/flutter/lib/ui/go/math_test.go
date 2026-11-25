package ui

import (
	"math"
	"testing"
)

// Port of packages/flutter/test/foundation/math_test.dart (foundation
// re-exports dart:ui's clampDouble).
func TestClampDouble(t *testing.T) {
	inf := math.Inf(1)
	for _, c := range [][4]float64{{-1, 0, 1, 0}, {2, 0, 1, 1}, {inf, 0, 1, 1}, {-inf, 0, 1, 0}, {math.NaN(), 0, inf, inf}} {
		if got := ClampDouble(c[0], c[1], c[2]); got != c[3] {
			t.Errorf("ClampDouble(%v, %v, %v) = %v, want %v", c[0], c[1], c[2], got, c[3])
		}
	}
}
