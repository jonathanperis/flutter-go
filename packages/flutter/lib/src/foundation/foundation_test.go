package foundation

// Ports of packages/flutter/test/foundation/{constants,basic_types,
// observer_list,caching_iterable}_test.dart (Flutter 3.50.1).
// CachingIterable's inherited Iterable helpers (where, map, skip, take, ...)
// are iter.Seq compositions in Go; the laziness and caching they check is
// tested here through All, Length and ElementAt.

import (
	"iter"
	"slices"
	"testing"
	"time"
)

func TestKIsWeb(t *testing.T) {
	if KIsWeb {
		t.Error("kIsWeb must be false")
	}
}

func TestLerpDuration(t *testing.T) {
	s, ms := time.Second, time.Millisecond
	for _, c := range []struct {
		a, b time.Duration
		t    float64
		want time.Duration
	}{
		{1 * s, 2 * s, 0.5, 1500 * ms}, {-1 * s, -2 * s, 0.5, -1500 * ms}, {-1 * s, 2 * s, 0.5, 500 * ms},
		{1 * s, 2 * s, 0, 1 * s}, {1 * s, 2 * s, 1, 2 * s}, {1 * s, 2 * s, 5, 6 * s}, {-1 * s, -2 * s, 5, -6 * s},
		// Beyond Flutter's tests: halves round away from zero, as Dart's round().
		{0, 3 * time.Microsecond, 0.5, 2 * time.Microsecond}, {0, -3 * time.Microsecond, 0.5, -2 * time.Microsecond},
	} {
		if got := LerpDuration(c.a, c.b, c.t); got != c.want {
			t.Errorf("lerpDuration(%v, %v, %v) = %v, want %v", c.a, c.b, c.t, got, c.want)
		}
	}
}

func prefix[T any](seq iter.Seq[T], n int) []T {
	var out []T
	if n == 0 {
		return out
	}
	for v := range seq {
		out = append(out, v)
		if len(out) == n {
			break
		}
	}
	return out
}

func TestObserverList(t *testing.T) {
	var l ObserverList[int]
	for i := 0; i < 10; i++ {
		l.Add(i)
	}
	if got := l.ToList(); !slices.Equal(got, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Fatalf("list = %v", got)
	}
	for i := 9; i >= 0; i-- {
		if !l.Remove(i) {
			t.Fatalf("remove(%d) = false", i)
		}
		for j, v := range prefix(l.All(), i) {
			if v != j {
				t.Fatalf("after removing %d: [%d] = %d", i, j, v)
			}
		}
	}
}

func TestHashedObserverList(t *testing.T) {
	var l HashedObserverList[int]
	for i := 0; i < 10; i++ {
		l.Add(i)
	}
	for i := 9; i >= 0; i-- {
		if !l.Remove(i) {
			t.Fatalf("remove(%d) = false", i)
		}
		if got := l.ToList(); !slices.Equal(got, prefix(slices.Values([]int{0, 1, 2, 3, 4, 5, 6, 7, 8}), i)) {
			t.Fatalf("after removing %d: %v", i, got)
		}
	}
	l.Add(0)
	for i := 0; i < 10; i++ {
		l.Add(1)
	}
	l.Add(2)
	if got := l.ToList(); !slices.Equal(got, []int{0, 1, 2}) {
		t.Fatalf("list = %v", got)
	}
	for i := 2; i >= 0; i-- {
		if !l.Remove(i) {
			t.Fatalf("remove(%d) = false", i)
		}
		for j, v := range l.ToList() {
			want := j
			if i == 0 {
				want = 1
			}
			if v != want || j >= 3 {
				t.Fatalf("after removing %d: [%d] = %d", i, j, v)
			}
		}
	}
	if got := l.ToList(); !slices.Equal(got, []int{1}) || l.IsEmpty() {
		t.Fatalf("list = %v", got)
	}
	for i := 0; i < 9; i++ {
		if !l.Remove(1) {
			t.Fatalf("remove(1) #%d = false", i)
		}
	}
	if !l.IsEmpty() {
		t.Fatal("not empty")
	}
}

func countingRange(count *int, start, end int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := start; i <= end; i++ {
			*count++
			if !yield(i) {
				return
			}
		}
	}
}

func TestCachingIterableLengthCaches(t *testing.T) {
	n := 0
	c := NewCachingIterable(countingRange(&n, 1, 5))
	if n != 0 {
		t.Fatalf("eager: %d", n)
	}
	if c.Length() != 5 || n != 5 || c.Length() != 5 || n != 5 {
		t.Fatalf("length/yields = %d/%d", c.Length(), n)
	}
	if got := slices.Collect(c.All()); !slices.Equal(got, []int{1, 2, 3, 4, 5}) || n != 5 {
		t.Fatalf("all = %v (yields %d)", got, n)
	}
}

func TestCachingIterableLaziness(t *testing.T) {
	n := 0
	c := NewCachingIterable(countingRange(&n, 1, 5))
	if got := prefix(c.All(), 1); got[0] != 1 || n != 1 {
		t.Fatalf("first = %v, yields %d", got, n)
	}
	for v := range c.All() {
		if v == 3 {
			break
		}
	}
	if n != 3 {
		t.Fatalf("firstWhere yields = %d", n)
	}
	all := slices.Collect(c.All())
	if all[len(all)-1] != 5 || n != 5 {
		t.Fatalf("last yields = %d", n)
	}
}

func TestCachingIterableElementAt(t *testing.T) {
	n := 0
	c := NewCachingIterable(countingRange(&n, 1, 5))
	func() {
		defer func() {
			if recover() == nil {
				t.Error("elementAt(-1) should panic")
			}
		}()
		c.ElementAt(-1)
	}()
	if c.ElementAt(1) != 2 || n != 2 {
		t.Fatalf("elementAt(1), yields %d", n)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("elementAt(9) should panic")
			}
		}()
		c.ElementAt(9)
	}()
	if c.ElementAt(4) != 5 || n != 5 {
		t.Fatalf("elementAt(4), yields %d", n)
	}
}

func TestKeys(t *testing.T) {
	if NewValueKey("a") != NewValueKey("a") || NewValueKey(1) == NewValueKey(2) {
		t.Error("ValueKey equality")
	}
	var a, b Key = NewValueKey(1), NewValueKey(int64(1))
	if a == b {
		t.Error("ValueKeys of different types must differ")
	}
	u1, u2 := NewUniqueKey(), NewUniqueKey()
	if Key(u1) == Key(u2) || Key(u1) != Key(u1) {
		t.Error("UniqueKey identity")
	}
	if NewKey("x") != Key(NewValueKey("x")) {
		t.Error("Key(value) is a ValueKey<String>")
	}
	if s := NewValueKey("x").String(); s != "[<'x'>]" {
		t.Errorf("string key = %s", s)
	}
	if s := NewValueKey(3).String(); s != "[<3>]" {
		t.Errorf("int key = %s", s)
	}
	if s := u1.String(); len(s) != 8 || s[:2] != "[#" {
		t.Errorf("unique key = %s", s)
	}
}

func TestDefaultTargetPlatform(t *testing.T) {
	t.Setenv("FLUTTER_TEST", "true")
	if DefaultTargetPlatform() != TargetPlatformAndroid {
		t.Error("tests run as android, like flutter test")
	}
	ios := TargetPlatformIOS
	SetDebugDefaultTargetPlatformOverride(&ios)
	defer SetDebugDefaultTargetPlatformOverride(nil)
	if DefaultTargetPlatform() != TargetPlatformIOS {
		t.Error("override ignored")
	}
}
