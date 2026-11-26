package painting

// durationTimes is Dart's Duration * num (MultiFrameImageStreamCompleter's
// frame delay times timeDilation): microseconds times the factor, rounded
// half away from zero (Duration(microseconds: 3) * 0.5 is 2µs in Dart).

import (
	"testing"
	"time"
)

func TestDurationTimes(t *testing.T) {
	for _, c := range []struct {
		d      time.Duration
		factor float64
		want   time.Duration
	}{
		{3 * time.Microsecond, 0.5, 2 * time.Microsecond},
		{5 * time.Microsecond, 0.5, 3 * time.Microsecond},
		{-3 * time.Microsecond, 0.5, -2 * time.Microsecond},
		{160 * time.Millisecond, 0.123456, 19753 * time.Microsecond},
		{100 * time.Millisecond, 2.0, 200 * time.Millisecond},
		{1999 * time.Nanosecond, 1.0, 1 * time.Microsecond}, // Duration's resolution is 1µs
	} {
		if got := durationTimes(c.d, c.factor); got != c.want {
			t.Errorf("%v * %v = %v, want %v", c.d, c.factor, got, c.want)
		}
	}
}
