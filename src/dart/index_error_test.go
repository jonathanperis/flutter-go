package dart

import "testing"

func TestIndexErrorString(t *testing.T) {
	cases := []struct {
		e    IndexError
		want string
	}{
		{IndexError{Index: 3, Length: 3, Name: "index"}, "RangeError (index): Index out of range: index should be less than 3: 3"},
		{IndexError{Index: -1, Length: 3, Name: "index"}, "RangeError (index): Index out of range: index must not be negative: -1"},
		{IndexError{Index: 0, Length: 0, Name: "index"}, "RangeError (index): Index out of range: no indices are valid: 0"},
		{IndexError{Index: 2, Length: 1, Message: "m"}, "RangeError: m: index should be less than 1: 2"},
	}
	for _, c := range cases {
		if got := c.e.Error(); got != c.want {
			t.Errorf("%q, want %q", got, c.want)
		}
	}
	for _, c := range []struct {
		e    RangeError
		want string
	}{
		{NewRangeErrorRange(10, 0, 9, "length", ""), "RangeError (length): Invalid value: Not in inclusive range 0..9: 10"},
		{NewRangeErrorRange(1, 0, 0, "length", ""), "RangeError (length): Invalid value: Only valid value is 0: 1"},
		{NewRangeErrorRange(0, 0, -1, "length", ""), "RangeError (length): Invalid value: Valid value range is empty: 0"},
	} {
		if got := c.e.Error(); got != c.want {
			t.Errorf("%q, want %q", got, c.want)
		}
	}
}
