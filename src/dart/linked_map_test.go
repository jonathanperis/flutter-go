package dart

import (
	"math"
	"strings"
	"testing"
)

func TestLinkedMapOrderAndKeys(t *testing.T) {
	m := NewLinkedMap(MapEntry[any, any]{Key: "b", Value: 1}, MapEntry[any, any]{Key: 1, Value: "int"})
	m.Set(1.0, "double") // the same key as 1: keeps its place and first key
	m.Set("a", 2)
	m.Set("b", 3) // a key set again keeps its place
	if got := m.String(); got != "{b: 3, 1: double, a: 2}" {
		t.Fatal(got)
	}
	if v, ok := m.Get(1.0); !ok || v != "double" {
		t.Fatal(v, ok)
	}
	if m.ContainsKey(0.0) || m.ContainsKey(0) || m.ContainsKey(2) {
		t.Fatal("zero / absent keys")
	}
	m.Set(math.Copysign(0, -1), "zero")
	if m.Lookup(0) != "zero" {
		t.Fatal("-0.0 is 0")
	}
	if v, ok := m.Remove("b"); !ok || v != 3 {
		t.Fatal(v, ok)
	}
	if _, ok := m.Remove("b"); ok {
		t.Fatal("removed twice")
	}
	m.Set("b", 4) // a removed key set again goes last
	if got := m.String(); got != "{1: double, a: 2, -0.0: zero, b: 4}" {
		t.Fatal(got)
	}
	if m.Lookup("a") != 2 || m.Lookup(1) != "double" || m.Len() != 4 {
		t.Fatal("index after removal")
	}
	if got := m.PutIfAbsent("a", func() any { return 9 }); got != 2 {
		t.Fatal(got)
	}
	if got := m.PutIfAbsent("c", func() any { return 9 }); got != 9 || m.Lookup("c") != 9 {
		t.Fatal(got)
	}
	keys := m.Keys()
	if len(keys) != 5 || keys[4] != "c" || len(m.Values()) != 5 || len(m.Entries()) != 5 {
		t.Fatal(keys)
	}
	m.Clear()
	if !m.IsEmpty() || m.Lookup("a") != nil {
		t.Fatal("clear")
	}
	func() {
		defer func() {
			if _, ok := recover().(UnsupportedError); !ok {
				t.Fatal("a slice key doesn't panic")
			}
		}()
		m.Set([]int{1}, 1)
	}()
}

func TestURIEncodeFullAndPath(t *testing.T) {
	for _, c := range [][2]string{
		{"assets/sub dir/a.png", "assets/sub%20dir/a.png"},
		{"q?x=1#frag", "q?x=1#frag"},
		{"ü", "%C3%BC"},
		{"%41", "%2541"},
		{`a\b[]`, "a%5Cb%5B%5D"},
		{"!$&'()*+,;=:@/~_.-", "!$&'()*+,;=:@/~_.-"},
	} {
		if got := URIEncodeFull(c[0]); got != c[1] {
			t.Errorf("encodeFull(%q) = %q, want %q", c[0], got, c[1])
		}
	}
	for _, c := range [][2]string{
		{"q?x=1#frag", "q%3Fx=1%23frag"},
		{"./dot/../a/./b", "a/b"},
		{"../up", "../up"},
		{"a:b/c", "a%3Ab/c"},
		{"1a:b", "1a:b"},
		{"/abs/../x", "/x"},
		{"%41%7e%7E", "A~~"},
		{"%zz", "%25zz"},
		{"", ""},
	} {
		if got := URIPath(c[0]); got != c[1] {
			t.Errorf("Uri(path: %q).path = %q, want %q", c[0], got, c[1])
		}
	}
	if !strings.HasPrefix(URIPath("a b"), "a%20b") {
		t.Error("space")
	}
}
