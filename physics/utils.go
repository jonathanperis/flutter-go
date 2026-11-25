package physics

// NearEqual reports whether a is within epsilon of b. Flutter's nearEqual
// also accepts null; Go floats can't be null (see docs/API_MAPPING.md).
func NearEqual(a, b, epsilon float64) bool {
	if epsilon < 0 {
		panic("physics: epsilon must be >= 0")
	}
	return (a > (b-epsilon)) && (a < (b+epsilon)) || a == b
}

// NearZero reports whether a is within epsilon of zero.
func NearZero(a, epsilon float64) bool { return NearEqual(a, 0.0, epsilon) }
