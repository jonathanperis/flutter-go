package dart

import (
	"math"
	"testing"
)

// Expected values are what Dart 3 prints.
func TestDoubleToString(t *testing.T) {
	for v, want := range map[float64]string{
		0: "0.0", 1: "1.0", -2: "-2.0", 0.001: "0.001", 1.5: "1.5", 1e-7: "1e-7", 1.5e-7: "1.5e-7",
		1e21: "1e+21", 123456789.25: "123456789.25", math.Inf(1): "Infinity",
	} {
		if got := DoubleToString(v); got != want {
			t.Errorf("DoubleToString(%v) = %q, want %q", v, got, want)
		}
	}
	a, b := 0.1, 0.2
	if got := DoubleToString(a + b); got != "0.30000000000000004" {
		t.Errorf("0.1+0.2 = %q", got)
	}
}

func TestToStringAsFixed(t *testing.T) {
	for _, c := range []struct {
		v    float64
		d    int
		want string
	}{{0.135, 1, "0.1"}, {9.81, 1, "9.8"}, {-4, 1, "-4.0"}, {2.5, 0, "3"}, {-2.5, 0, "-3"}, {0.125, 2, "0.13"}, {9.95, 1, "9.9"}, {0.5, 0, "1"}, {9.5, 0, "10"}, {1.005, 2, "1.00"}, {math.Inf(-1), 1, "-Infinity"}} {
		if got := ToStringAsFixed(c.v, c.d); got != c.want {
			t.Errorf("ToStringAsFixed(%v, %d) = %q, want %q", c.v, c.d, got, c.want)
		}
	}
}
