package dart

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// The expected orders were printed by the Dart VM (Flutter 3.50.1's SDK):
// List.sort with ties, across the insertion-sort threshold and the
// dual-pivot quicksort's branches.
var dartSortOrders = map[int]string{
	5:   "0,1,2,3,4",
	33:  "0,13,26,7,20,1,14,27,8,21,2,15,28,9,22,3,16,29,10,23,4,17,30,11,24,5,18,31,12,25,6,19,32",
	34:  "0,13,26,33,7,20,1,14,27,21,8,28,15,2,9,22,29,3,16,10,23,30,17,4,24,11,18,5,31,12,25,19,6,32",
	50:  "13,0,26,39,7,46,20,33,1,14,40,27,34,47,8,21,41,2,28,15,9,22,48,35,42,16,29,3,10,23,49,36,17,43,30,4,11,37,24,31,18,44,5,25,12,38,32,19,45,6",
	100: "52,91,13,65,26,78,0,39,33,59,98,20,85,7,72,46,53,92,1,14,27,79,40,66,34,8,99,86,60,73,21,47,2,28,80,93,15,41,67,54,9,87,35,74,22,48,61,81,29,94,55,68,16,3,42,75,36,62,23,10,88,49,4,56,30,69,43,82,17,95,63,50,37,76,24,89,11,70,44,57,31,83,18,5,96,38,90,12,25,64,51,77,58,32,71,45,6,84,97,19",
	257: "91,78,13,182,169,26,247,39,52,208,104,156,221,0,117,234,143,195,130,65,202,241,137,46,72,163,215,85,189,33,176,228,98,20,124,7,111,254,150,59,105,92,131,79,235,66,144,196,209,118,157,53,170,40,27,248,14,183,1,222,86,138,21,190,73,8,47,177,229,203,99,164,34,112,151,60,125,255,242,216,106,2,249,28,67,223,54,93,171,197,236,158,80,119,210,145,15,41,184,132,100,87,165,230,61,113,217,35,9,48,152,74,256,126,243,139,22,204,191,178,120,94,211,146,55,29,250,172,159,198,16,185,237,133,3,81,224,107,68,42,205,140,127,218,153,192,36,244,62,23,75,49,114,88,10,101,231,166,179,147,160,108,238,17,56,43,186,69,121,251,4,225,134,173,212,95,30,82,199,167,180,102,11,206,232,76,50,89,115,24,154,37,219,193,63,141,245,128,5,187,200,174,31,161,148,213,135,122,109,226,96,83,70,57,239,44,18,252,64,129,220,116,103,155,207,90,181,77,233,142,194,168,51,38,246,25,12,201,175,214,240,110,45,188,149,32,71,123,97,19,162,84,58,253,6,136,227",
}

func order(list [][2]int) string {
	parts := make([]string, len(list))
	for i, e := range list {
		parts[i] = strconv.Itoa(e[1])
	}
	return strings.Join(parts, ",")
}

func TestListSortMatchesDart(t *testing.T) {
	for n, want := range dartSortOrders {
		list := make([][2]int, n)
		for i := range list {
			list[i] = [2]int{(i * 7919) % 13, i}
		}
		ListSort(list, func(a, b [2]int) int { return a[0] - b[0] })
		if got := order(list); got != want {
			t.Errorf("n=%d\n got  %s\n want %s", n, got, want)
		}
	}
	// Equal pivots.
	eq := make([][2]int, 60)
	for i := range eq {
		eq[i] = [2]int{(i * 31) % 3, i}
	}
	ListSort(eq, func(a, b [2]int) int { return a[0] - b[0] })
	if got, want := order(eq), "36,3,57,6,9,39,54,12,15,51,18,0,48,21,24,45,27,30,33,42,19,13,22,46,52,25,55,16,28,10,4,31,43,58,34,1,7,37,49,40,38,41,35,32,44,26,23,47,20,29,50,17,14,53,11,8,56,5,2,59"; got != want {
		t.Errorf("equal pivots\n got  %s\n want %s", got, want)
	}
	// sortRange
	r := []int{5, 4, 3, 2, 1}
	ListSortRange(r, 1, 4, CompareInts)
	if r[0] != 5 || r[1] != 2 || r[2] != 3 || r[3] != 4 || r[4] != 1 {
		t.Fatal(r)
	}
}

func TestCompareDoublesAndStrings(t *testing.T) {
	ds := []float64{math.Copysign(0, -1), 0, math.NaN(), 1, -1}
	var rows []string
	for _, a := range ds {
		var row []string
		for _, b := range ds {
			row = append(row, strconv.Itoa(CompareDoubles(a, b)))
		}
		rows = append(rows, strings.Join(row, ","))
	}
	if got := strings.Join(rows, " "); got != "0,-1,-1,-1,1 1,0,-1,-1,1 1,1,0,1,1 1,1,-1,0,1 -1,-1,-1,-1,0" {
		t.Fatal(got)
	}
	ss := []string{"a", "b", "ab", "", "\U0001F600", "￿", "é"}
	rows = nil
	for _, a := range ss {
		var row []string
		for _, b := range ss {
			row = append(row, strconv.Itoa(CompareStrings(a, b)))
		}
		rows = append(rows, strings.Join(row, ","))
	}
	if got := strings.Join(rows, " "); got != "0,-1,-1,1,-1,-1,-1 1,0,1,1,-1,-1,-1 1,-1,0,1,-1,-1,-1 -1,-1,-1,0,-1,-1,-1 1,1,1,1,0,-1,1 1,1,1,1,1,0,1 1,1,1,1,-1,-1,0" {
		t.Fatal(got)
	}
}

// The results of double.tryParse, printed by the Dart VM.
func TestDoubleTryParse(t *testing.T) {
	cases := map[string]string{
		"1": "1.0", " 1 ": "1.0", "1.": "1.0", ".5": "0.5", "-.5": "-0.5", "+1": "1.0", "1e3": "1000.0",
		"1E-3": "0.001", "NaN": "NaN", "-NaN": "NaN", "Infinity": "Infinity", "-Infinity": "-Infinity",
		"+Infinity": "Infinity", "inf": "null", "0x10": "null", "1_000": "null", "": "null", " ": "null",
		"1e": "null", "50%": "null", "1.5.5": "null", " 1": "1.0", "1d": "null", "1f": "null", ".": "null",
		"-": "null", "+.5e+2": "50.0", "00012": "12.0", "1e400": "Infinity", "-0": "-0.0",
	}
	for in, want := range cases {
		got := "null"
		if v, ok := DoubleTryParse(in); ok {
			got = DoubleToString(v)
		}
		if got != want {
			t.Errorf("%q: got %s want %s", in, got, want)
		}
	}
}
