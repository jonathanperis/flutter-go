package dart

import (
	"math"
	"testing"
)

// Expected values are what Dart 3.50.1 prints for math.pow (macOS arm64).
// Go's math.Pow gives 0.03310476657088506 and 1.1407667734226012 here.
func TestPowMatchesDart(t *testing.T) {
	base := 51.0 / 255
	base = (base + 0.055) / 1.055 // computed at run time, as in Color.computeLuminance
	for _, c := range []struct{ x, y, want float64 }{
		{base, 2.4, 0.033104766570885055},
		{0.9, -1.25, 1.140766773422601},
		{0.135, 0.5, 0.3674234614174767},
		{1.3, 2.4, 1.8769990185500045},
		{0.3, 3, 0.027}, // the VM's own x*x*x; C's pow gives 0.026999999999999996
		{1.7, 5, 14.198569999999998}, {0.135, 4, 0.0003321506250000001}, {0.7, 2, 0.48999999999999994},
		{1.1, 3, 1.3310000000000004}, {0.3, 0.5, 0.5477225575051661}, {0.3, -1, 3.3333333333333335},
		{0.3, -2, 11.111111111111112}, {0.3, 4, 0.0081}, {math.Inf(-1), 0.5, math.Inf(1)},
		{1, math.NaN(), 1}, {math.NaN(), 0, 1},
	} {
		if got := Pow(c.x, c.y); got != c.want {
			t.Errorf("Pow(%v, %v) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}
