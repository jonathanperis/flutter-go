package foundation

// Ports of packages/flutter/test/foundation/{collections,bit_field}_test.dart.

import (
	"math/rand"
	"testing"
)

func set(v ...int) map[int]struct{} {
	m := map[int]struct{}{}
	for _, x := range v {
		m[x] = struct{}{}
	}
	return m
}

func TestListEquals(t *testing.T) {
	a, b, c, d := []int{1, 2, 3}, []int{1, 2, 3}, []int{1, 2}, []int{3, 2, 1}
	for _, x := range []struct {
		a, b []int
		want bool
	}{{nil, nil, true}, {a, nil, false}, {nil, b, false}, {a, a, true}, {a, b, true}, {a, c, false}, {a, d, false},
		{[]int{}, nil, false}, {nil, []int{}, false}} { // beyond Flutter's tests: empty is not null
		if ListEquals(x.a, x.b) != x.want {
			t.Errorf("listEquals(%v, %v) != %v", x.a, x.b, x.want)
		}
	}
}

func TestSetEquals(t *testing.T) {
	a, b, c, d := set(1, 2, 3), set(1, 2, 3), set(1, 2), set(3, 2, 1)
	for _, x := range []struct {
		a, b map[int]struct{}
		want bool
	}{{nil, nil, true}, {a, nil, false}, {nil, b, false}, {a, a, true}, {a, b, true}, {a, c, false}, {a, d, true},
		{set(), nil, false}, {nil, set(), false}} {
		if SetEquals(x.a, x.b) != x.want {
			t.Errorf("setEquals(%v, %v) != %v", x.a, x.b, x.want)
		}
	}
}

func TestMapEquals(t *testing.T) {
	a := map[int]int{1: 1, 2: 2, 3: 3}
	b := map[int]int{1: 1, 2: 2, 3: 3}
	c := map[int]int{1: 1, 2: 2}
	d := map[int]int{3: 3, 2: 2, 1: 1}
	e := map[int]int{3: 1, 2: 2, 1: 3}
	for _, x := range []struct {
		a, b map[int]int
		want bool
	}{{nil, nil, true}, {a, nil, false}, {nil, b, false}, {a, a, true}, {a, b, true}, {a, c, false}, {a, d, true}, {a, e, false},
		{map[int]int{}, nil, false}, {nil, map[int]int{}, false}} {
		if MapEquals(x.a, x.b) != x.want {
			t.Errorf("mapEquals(%v, %v) != %v", x.a, x.b, x.want)
		}
	}
}

func TestBinarySearch(t *testing.T) {
	items := []int{1, 2, 3}
	for v, want := range map[int]int{1: 0, 2: 1, 3: 2, 12: -1} {
		if got := BinarySearch(items, v); got != want {
			t.Errorf("binarySearch(%d) = %d, want %d", v, got, want)
		}
	}
}

func TestMergeSortRandom(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 250; i++ {
		list := make([]int, i)
		for j := range list {
			list[j] = r.Intn(i)
		}
		MergeSort(list)
		for j := 1; j < i; j++ {
			if list[j-1] > list[j] {
				t.Fatalf("size %d not sorted at %d", i, j)
			}
		}
	}
}

type orderedComparable struct{ id, order int }

func (a orderedComparable) CompareTo(b orderedComparable) int { return a.id - b.id }

func TestMergeSortPreservesOrder(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for _, size := range []int{8, 50, 511, 512, 513} {
		list := make([]orderedComparable, size)
		for i := range list {
			list[i] = orderedComparable{r.Intn(size >> 2), i}
		}
		MergeSort(list)
		for i := 1; i < size; i++ {
			p, n := list[i-1], list[i]
			if p.id > n.id || (p.id == n.id && p.order > n.order) {
				t.Fatalf("size %d: not stable at %d", size, i)
			}
		}
		cp := append([]orderedComparable(nil), list...)
		lo, hi := size>>2, size-(size>>2)
		MergeSort(list, MergeSortOptions[orderedComparable]{Start: lo, End: &hi,
			Compare: func(a, b orderedComparable) int { return b.CompareTo(a) }})
		for i := lo + 1; i < hi; i++ {
			p, n := list[i-1], list[i]
			if p.id < n.id || (p.id == n.id && p.order > n.order) {
				t.Fatalf("size %d: reverse range not stable at %d", size, i)
			}
		}
		for i := 0; i < lo; i++ {
			if list[i] != cp[i] {
				t.Fatalf("prefix changed")
			}
		}
		for i := hi; i < size; i++ {
			if list[i] != cp[i] {
				t.Fatalf("suffix changed")
			}
		}
	}
}

func TestMergeSortSpecialCases(t *testing.T) {
	for _, size := range []int{511, 512, 513} {
		list := make([]orderedComparable, size)
		for i := range list {
			list[i] = orderedComparable{0, i}
		}
		MergeSort(list)
		for i := range list {
			if list[i].order != i {
				t.Fatalf("stable equal elements moved")
			}
		}
		list[0] = orderedComparable{1, 0}
		for i := 1; i < size; i++ {
			list[i] = orderedComparable{0, i}
		}
		MergeSort(list)
		for i := 0; i < size-1; i++ {
			if list[i].order != i+1 {
				t.Fatalf("first-largest case at %d", i)
			}
		}
		if list[size-1].order != 0 {
			t.Fatalf("largest not last")
		}
		for i := 0; i < size-1; i++ {
			list[i] = orderedComparable{0, i}
		}
		list[size-1] = orderedComparable{-1, size - 1}
		MergeSort(list)
		if list[0].order != size-1 {
			t.Fatalf("smallest not first")
		}
		for i := 1; i < size; i++ {
			if list[i].order != i-1 {
				t.Fatalf("last-smallest case at %d", i)
			}
		}
		for i := range list {
			list[i] = orderedComparable{size - 1 - i, i}
		}
		MergeSort(list)
		for i := range list {
			if list[i].id != i || list[i].order != size-1-i {
				t.Fatalf("reversed case at %d", i)
			}
		}
	}
}

type testEnum int

const (
	teA testEnum = iota
	teB
	teC
	teD
	teE
)

func TestBitField(t *testing.T) {
	f := NewBitField[testEnum](8)
	if f.Get(teD) {
		t.Fatal("d set initially")
	}
	f.Set(teD, true)
	f.Set(teE, true)
	if f.Get(teC) || !f.Get(teD) || !f.Get(teE) {
		t.Fatal("after setting d, e")
	}
	f.Set(teE, false)
	if f.Get(teC) || !f.Get(teD) || f.Get(teE) {
		t.Fatal("after clearing e")
	}
	f.Reset()
	if f.Get(teC) || f.Get(teD) || f.Get(teE) {
		t.Fatal("after reset()")
	}
	f.Reset(true)
	if !f.Get(teC) || !f.Get(teD) || !f.Get(teE) {
		t.Fatal("after reset(true)")
	}
	if !NewBitFieldFilled[testEnum](8, true).Get(teD) || NewBitFieldFilled[testEnum](8, false).Get(teD) {
		t.Fatal("filled")
	}
}
