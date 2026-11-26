package dart

import (
	"math"
	"reflect"
	"slices"
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
	keys := slices.Collect(m.Keys())
	if !reflect.DeepEqual(keys, []any{1, "a", -0.0, "b", "c"}) || !reflect.DeepEqual(m.KeyList(), keys) {
		t.Fatal(keys)
	}
	if values := slices.Collect(m.Values()); !reflect.DeepEqual(values, []any{"double", 2, "zero", 4, 9}) {
		t.Fatal(values)
	}
	var entries []MapEntry[any, any]
	for k, v := range m.Entries() {
		entries = append(entries, MapEntry[any, any]{Key: k, Value: v})
	}
	if !reflect.DeepEqual(entries, m.EntryList()) || len(entries) != 5 || entries[4].Key != "c" {
		t.Fatal(entries)
	}
	if !m.IsNotEmpty() {
		t.Fatal("not empty")
	}
	m.Clear()
	if !m.IsEmpty() || m.IsNotEmpty() || m.Lookup("a") != nil || m.Len() != 0 {
		t.Fatal("clear")
	}
	func() {
		defer func() {
			if _, ok := recover().(UnsupportedError); !ok {
				t.Fatal("an uncomparable struct key doesn't panic")
			}
		}()
		m.Set(struct{ s []int }{}, 1)
	}()
}

// TestLinkedMapRemoveKeepsOrder removes from the front, middle and end, and
// checks lookups and order after each removal.
func TestLinkedMapRemoveKeepsOrder(t *testing.T) {
	m := NewLinkedMapOf("a", 1, "b", 2, "c", 3, "d", 4, "e", 5)
	for _, step := range []struct {
		remove string
		want   string
	}{
		{"c", "{a: 1, b: 2, d: 4, e: 5}"},
		{"a", "{b: 2, d: 4, e: 5}"},
		{"e", "{b: 2, d: 4}"},
	} {
		m.Remove(step.remove)
		if got := m.String(); got != step.want {
			t.Fatalf("after removing %s: %s, want %s", step.remove, got, step.want)
		}
		for k, v := range m.Entries() {
			if got := m.Lookup(k); got != v {
				t.Fatalf("lookup %v = %v, want %v", k, got, v)
			}
		}
	}
	m.Set("a", 6)
	if got := m.String(); got != "{b: 2, d: 4, a: 6}" {
		t.Fatal(got)
	}
	other := NewLinkedMapOf("d", 7, "z", 8)
	m.AddAll(other)
	if got := m.String(); got != "{b: 2, d: 7, a: 6, z: 8}" {
		t.Fatal(got)
	}
}

// TestLinkedMapKeySemantics: numbers merge as Dart's == and hashCode do,
// NaN never matches, lists and maps are keys by identity.
func TestLinkedMapKeySemantics(t *testing.T) {
	m := &LinkedMap{}
	m.Set(math.MaxInt64, "max")
	m.Set(0x1p63, "2^63") // == maxInt and hashes as it (saturating cast)
	m.Set(1<<53+1, "int")
	m.Set(0x1p53, "double") // == (1<<53+1) as a double, but hashes apart
	m.Set(-0x1p63, "min")
	m.Set(math.MinInt64, "minInt")
	m.Set(1.5, "x")
	m.Set(math.NaN(), 1)
	m.Set(math.NaN(), 2)
	m.Set(nil, "null")
	m.Set(true, "t")
	m.Set("1", "s")
	if got := m.String(); got != "{9223372036854775807: 2^63, 9007199254740993: int, 9007199254740992.0: double, -9223372036854776000.0: minInt, 1.5: x, NaN: 1, NaN: 2, null: null, true: t, 1: s}" {
		t.Fatal(got)
	}
	if m.ContainsKey(math.NaN()) || m.Lookup(nil) != "null" || m.Lookup(1) != nil {
		t.Fatal("lookups")
	}
	m.Remove(1.5) // reindexes past the NaN keys
	if m.Lookup(nil) != "null" || m.Lookup("1") != "s" || m.Len() != 9 {
		t.Fatal("after removal", m)
	}

	list1, list2 := []any{1}, []any{1}
	typed := []byte{1}
	inner := map[string]int{"a": 1}
	l := NewLinkedMapOf(list1, "first", list2, "second", typed, "bytes", inner, "map")
	l.Set(list1, "again")
	if l.Len() != 4 || l.Lookup(list1) != "again" || l.Lookup(list2) != "second" || l.Lookup([]any{1}) != nil ||
		l.Lookup(typed) != "bytes" || l.Lookup(inner) != "map" || l.Lookup(map[string]int{"a": 1}) != nil {
		t.Fatal(l)
	}
	if got := l.String(); got != "{[1]: again, [1]: second, [1]: bytes, {a: 1}: map}" {
		t.Fatal(got)
	}
	if l.Lookup(list1[:0]) != nil { // a different length is a different list
		t.Fatal("sub-slice")
	}
	self := &LinkedMap{}
	self.Set("self", self)
	self.Set(self, 1)
	if got := self.String(); got != "{self: {...}, {...}: 1}" {
		t.Fatal(got)
	}
}

func TestLinkedMapConcurrentModification(t *testing.T) {
	expectCME := func(name string, f func()) {
		t.Helper()
		defer func() {
			if e, ok := recover().(ConcurrentModificationError); !ok ||
				!strings.HasPrefix(e.Error(), "Concurrent modification during iteration: _Map len:") {
				t.Errorf("%s: no ConcurrentModificationError", name)
			}
		}()
		f()
	}
	m := NewLinkedMapOf("a", 1, "b", 2)
	expectCME("add", func() {
		for k := range m.Keys() {
			m.Set(k.(string)+"x", 0)
		}
	})
	m = NewLinkedMapOf("a", 1, "b", 2)
	expectCME("remove", func() {
		for range m.Values() {
			m.Remove("b")
		}
	})
	m = NewLinkedMapOf("a", 1, "b", 2)
	expectCME("remove last", func() {
		for k := range m.Entries() {
			if k == "b" {
				m.Remove("a")
			}
		}
	})
	m = NewLinkedMapOf("a", 1, "b", 2)
	expectCME("clear", func() {
		for range m.Entries() {
			m.Clear()
		}
	})
	// Replacing a value is not a modification; breaking out is fine.
	m = NewLinkedMapOf("a", 1, "b", 2)
	for k := range m.Keys() {
		m.Set(k, 3)
	}
	for range m.Entries() {
		break
	}
	if got := m.String(); got != "{a: 3, b: 3}" {
		t.Fatal(got)
	}
	func() {
		defer func() {
			if _, ok := recover().(ArgumentError); !ok {
				t.Error("odd NewLinkedMapOf")
			}
		}()
		NewLinkedMapOf("a")
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
