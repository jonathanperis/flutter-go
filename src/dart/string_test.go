package dart

import "testing"

func TestStringUTF16Ops(t *testing.T) {
	s := "a😀b" // a, surrogate pair, b
	if StringLength(s) != 4 {
		t.Fatal(StringLength(s))
	}
	hi := Substring(s, 0, 2)
	if StringLength(hi) != 2 || CodeUnitAt(hi, 1) != 0xD83D {
		t.Fatalf("%q", hi)
	}
	if got := Concat(Substring(s, 0, 2), Substring(s, 2)); got != s {
		t.Fatalf("rejoined %q", got)
	}
	if got := ReplaceRange(s, 1, 3, "xy"); got != "axyb" {
		t.Fatal(got)
	}
	if got := StringFromCharCode(0x1F600); got != "😀" {
		t.Fatal(got)
	}
	if got := StringFromCharCodes([]uint16{'a', 0xDC00}); StringLength(got) != 2 || CodeUnitAt(got, 1) != 0xDC00 {
		t.Fatalf("%q", got)
	}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("no panic")
			}
		}()
		CodeUnitAt(s, 4)
	}()
}
