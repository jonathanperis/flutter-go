// Package dart reproduces Dart core-library behavior that Flutter's output
// depends on (number formatting in toString, for example), so Go's output
// matches Flutter's exactly.
package dart

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// DoubleToString is Dart's double.toString().
func DoubleToString(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case math.IsNaN(v):
		return "NaN"
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e21 {
		// Shortest round-trip digits, zero-padded (2^63 is
		// "9223372036854775808.0" exactly but Dart prints
		// "9223372036854776000.0").
		return strconv.FormatFloat(v, 'f', -1, 64) + ".0"
	}
	a := math.Abs(v)
	if a >= 1e-6 && a < 1e21 {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	s := strconv.FormatFloat(v, 'e', -1, 64) // 1.5e-07
	mant, exp, _ := strings.Cut(s, "e")
	sign := exp[:1]
	exp = strings.TrimLeft(exp[1:], "0")
	return mant + "e" + sign + exp
}

// ToStringAsFixed is Dart's double.toStringAsFixed(digits).
func ToStringAsFixed(v float64, digits int) string {
	switch {
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case math.IsNaN(v):
		return "NaN"
	}
	if math.Abs(v) >= 1e21 {
		return DoubleToString(v)
	}
	// Dart rounds an exact tie away from zero; strconv rounds it to even.
	// A tie is only possible when the double's exact decimal expansion ends
	// with a 5 right after the last kept digit.
	exact := new(big.Float).SetFloat64(math.Abs(v)).Text('f', 1100)
	exact = strings.TrimRight(exact, "0")
	if dot := strings.IndexByte(exact, '.'); dot >= 0 && len(exact)-dot-1 == digits+1 && exact[len(exact)-1] == '5' {
		out := roundUpDecimal(exact[:len(exact)-1])
		out = strings.TrimSuffix(out, ".")
		if v < 0 {
			out = "-" + out
		}
		return out
	}
	return strconv.FormatFloat(v, 'f', digits, 64)
}

// roundUpDecimal adds one unit in the last place of a non-negative decimal
// string ("2." → "3.", "0.19" → "0.20", "9.9" → "10.0").
func roundUpDecimal(s string) string {
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		switch {
		case b[i] == '.':
			continue
		case b[i] == '9':
			b[i] = '0'
		default:
			b[i]++
			return string(b)
		}
	}
	return "1" + string(b)
}
