package dart

import (
	"math"
	"testing"
	"time"
)

// Expected values are Dart's (json.encode, Duration.toString).
func TestJSONEncode(t *testing.T) {
	for _, c := range []struct {
		v    any
		want string
	}{
		{nil, "null"},
		{true, "true"},
		{"a<b>&\"\\\n é", `"a<b>&\"\\\n` + " é\""},
		{1.0, "1.0"},
		{1e300, "1e+300"},
		{0.1, "0.1"},
		{-0.0, "0.0"},
		{math.Copysign(0, -1), "-0.0"},
		{42, "42"},
		{[]any{1, "x", nil}, `[1,"x",null]`},
		{map[string]any{"b": 1, "a": []int{}}, `{"a":[],"b":1}`},
		{[]MapEntry[string, any]{{"z", 1.5}, {"a", false}}, `{"z":1.5,"a":false}`},
	} {
		if got := JSONEncode(c.v); got != c.want {
			t.Errorf("JSONEncode(%#v) = %s, want %s", c.v, got, c.want)
		}
	}
	defer func() {
		if _, ok := recover().(JSONUnsupportedObjectError); !ok {
			t.Fatal("NaN must not encode")
		}
	}()
	JSONEncode(math.NaN())
}

func TestDurationToString(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00:00.000000"},
		{1500 * time.Microsecond, "0:00:00.001500"},
		{5 * time.Millisecond, "0:00:00.005000"},
		{3*time.Hour + 4*time.Minute + 5*time.Second + 6*time.Microsecond, "3:04:05.000006"},
		{100 * time.Hour, "100:00:00.000000"},
		{-1500 * time.Millisecond, "-0:00:01.500000"},
	} {
		if got := DurationToString(c.d); got != c.want {
			t.Errorf("%v: %s, want %s", c.d, got, c.want)
		}
	}
}
