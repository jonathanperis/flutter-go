package dart

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// RangeError is dart:core's RangeError for an integer argument outside an
// inclusive range (RangeError.range).
type RangeError struct {
	Value, Start, End int
	Name              string
	Message           string // "" means Dart's default, "Invalid value"
}

// NewRangeErrorRange is RangeError.range(invalidValue, start, end, name, message).
func NewRangeErrorRange(invalidValue, start, end int, name, message string) RangeError {
	return RangeError{Value: invalidValue, Start: start, End: end, Name: name, Message: message}
}

func (e RangeError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "Invalid value"
	}
	s := "RangeError"
	if e.Name != "" {
		s += " (" + e.Name + ")"
	}
	var explanation string
	switch {
	case e.End > e.Start:
		explanation = ": Not in inclusive range " + strconv.Itoa(e.Start) + ".." + strconv.Itoa(e.End)
	case e.End < e.Start:
		explanation = ": Valid value range is empty"
	default:
		explanation = ": Only valid value is " + strconv.Itoa(e.Start)
	}
	return s + ": " + msg + explanation + ": " + strconv.Itoa(e.Value)
}

// ToStringAsPrecision is Dart's double.toStringAsPrecision(precision): the
// value rounded to precision significant digits, in exponential notation
// when the exponent is below -6 or not below precision (ECMAScript's
// toPrecision, as the VM's double-conversion ToPrecision implements it:
// exact ties round away from zero, "e+" for positive exponents, -0.0 keeps
// its sign). It panics with a RangeError unless 1 <= precision <= 21.
func ToStringAsPrecision(v float64, precision int) string {
	if precision < 1 || precision > 21 {
		panic(RangeError{Value: precision, Start: 1, End: 21, Name: "precision"})
	}
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	}
	sign := ""
	if math.Signbit(v) {
		sign = "-"
	}
	a := math.Abs(v)
	mant, exp, _ := strings.Cut(strconv.FormatFloat(a, 'e', precision-1, 64), "e")
	digits := strings.Replace(mant, ".", "", 1)
	exponent, _ := strconv.Atoi(exp)
	// Dart rounds an exact tie away from zero; strconv rounds it to even. A
	// tie is only possible when the exact expansion has precision+1
	// significant digits, the last one a 5.
	if a != 0 {
		exactMant, exactExp, _ := strings.Cut(new(big.Float).SetFloat64(a).Text('e', 1100), "e")
		exact := strings.TrimRight(strings.Replace(exactMant, ".", "", 1), "0")
		if len(exact) == precision+1 && exact[precision] == '5' {
			exponent, _ = strconv.Atoi(exactExp)
			digits = roundUpDecimal(exact[:precision])
			if len(digits) > precision { // 9.5 → 10
				digits = digits[:precision]
				exponent++
			}
		}
	}
	decimalPoint := exponent + 1
	if -decimalPoint+1 > 6 || decimalPoint > precision {
		out := digits[:1]
		if precision > 1 {
			out += "." + digits[1:]
		}
		if exponent < 0 {
			return sign + out + "e-" + strconv.Itoa(-exponent)
		}
		return sign + out + "e+" + strconv.Itoa(exponent)
	}
	switch {
	case decimalPoint <= 0:
		return sign + "0." + strings.Repeat("0", -decimalPoint) + digits
	case decimalPoint == precision:
		return sign + digits
	default:
		return sign + digits[:decimalPoint] + "." + digits[decimalPoint:]
	}
}
