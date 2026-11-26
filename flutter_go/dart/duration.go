package dart

import (
	"strconv"
	"time"
)

// DurationToString is Dart's Duration.toString(): "H:MM:SS.ffffff"
// (microsecond precision; hours are not wrapped; negative durations start
// with "-").
func DurationToString(d time.Duration) string {
	us := d.Microseconds()
	sign := ""
	if us < 0 {
		sign, us = "-", -us
	}
	h := us / 3_600_000_000
	us -= h * 3_600_000_000
	m := us / 60_000_000
	us -= m * 60_000_000
	s := us / 1_000_000
	us -= s * 1_000_000
	pad := func(v int64, n int) string {
		t := strconv.FormatInt(v, 10)
		for len(t) < n {
			t = "0" + t
		}
		return t
	}
	return sign + strconv.FormatInt(h, 10) + ":" + pad(m, 2) + ":" + pad(s, 2) + "." + pad(us, 6)
}
