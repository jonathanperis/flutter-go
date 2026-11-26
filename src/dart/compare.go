package dart

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
)

// CompareDoubles is double.compareTo: -0.0 sorts before 0.0, and NaN is
// equal to itself and greater than every other value.
func CompareDoubles(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	if a == b {
		if a == 0 {
			aNeg, bNeg := math.Signbit(a), math.Signbit(b)
			if aNeg == bNeg {
				return 0
			}
			if aNeg {
				return -1
			}
			return 1
		}
		return 0
	}
	if math.IsNaN(a) {
		if math.IsNaN(b) {
			return 0
		}
		return 1
	}
	return -1
}

// CompareInts is int.compareTo (Comparable.compare of two ints).
func CompareInts(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// CompareStrings is String.compareTo: a lexicographic comparison of the
// UTF-16 code units (which orders supplementary characters differently
// from a byte comparison of UTF-8), returning -1, 0 or 1.
func CompareStrings(a, b string) int {
	if isASCII(a) && isASCII(b) {
		return strings.Compare(a, b)
	}
	ua := utf16.Encode([]rune(a))
	ub := utf16.Encode([]rune(b))
	n := min(len(ua), len(ub))
	for i := 0; i < n; i++ {
		if ua[i] < ub[i] {
			return -1
		}
		if ua[i] > ub[i] {
			return 1
		}
	}
	return CompareInts(len(ua), len(ub))
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// DoubleTryParse is double.tryParse: after trimming Dart whitespace, an
// optional sign followed by "NaN", "Infinity", or a decimal literal
// (digits with an optional fraction and exponent; "1." and ".5" are
// accepted, "1e", "0x10", "1_000" and "inf" are not). ok is false where Dart
// returns null.
func DoubleTryParse(source string) (value float64, ok bool) {
	s := strings.TrimFunc(source, isDartWhitespaceRune)
	if s == "" {
		return 0, false
	}
	body := s
	negative := false
	if body[0] == '+' || body[0] == '-' {
		negative = body[0] == '-'
		body = body[1:]
	}
	switch body {
	case "NaN":
		return math.NaN(), true
	case "Infinity":
		if negative {
			return math.Inf(-1), true
		}
		return math.Inf(1), true
	}
	i, digits := 0, 0
	for i < len(body) && body[i] >= '0' && body[i] <= '9' {
		i++
		digits++
	}
	if i < len(body) && body[i] == '.' {
		i++
		for i < len(body) && body[i] >= '0' && body[i] <= '9' {
			i++
			digits++
		}
	}
	if digits == 0 {
		return 0, false
	}
	if i < len(body) && (body[i] == 'e' || body[i] == 'E') {
		i++
		if i < len(body) && (body[i] == '+' || body[i] == '-') {
			i++
		}
		expDigits := 0
		for i < len(body) && body[i] >= '0' && body[i] <= '9' {
			i++
			expDigits++
		}
		if expDigits == 0 {
			return 0, false
		}
	}
	if i != len(body) {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		if ne, isNum := err.(*strconv.NumError); !isNum || ne.Err != strconv.ErrRange {
			return 0, false
		}
	}
	return v, true
}

// isDartWhitespaceRune is the whitespace String.trim removes.
func isDartWhitespaceRune(r rune) bool {
	switch {
	case r >= 0x09 && r <= 0x0D, r == 0x20, r == 0x85, r == 0xA0, r == 0x1680,
		r >= 0x2000 && r <= 0x200A, r == 0x2028, r == 0x2029, r == 0x202F, r == 0x205F,
		r == 0x3000, r == 0xFEFF:
		return true
	}
	return false
}
