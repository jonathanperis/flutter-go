package dart

import "testing"

type identityReceiver struct{ n int }

func (r *identityReceiver) m1() { r.n++ }
func (r *identityReceiver) m2() { r.n-- }
func (r identityReceiver) v()   {}

type genericReceiver[T any] struct{ v T }

func (g *genericReceiver[T]) m() {}

func topLevel1() {}
func topLevel2() {}

func TestFunctionsEqual(t *testing.T) {
	a, b := &identityReceiver{}, &identityReceiver{}
	g1, g2 := &genericReceiver[int]{}, &genericReceiver[int]{}
	n := 0
	closure := func() func() { return func() { n++ } }
	c1, c2 := closure(), closure()
	var nilFunc func()
	f := a.m1
	for _, c := range []struct {
		name string
		x, y func()
		want bool
	}{
		{"same top-level function", topLevel1, topLevel1, true},
		{"different top-level functions", topLevel1, topLevel2, false},
		{"same closure value", c1, c1, true},
		{"two evaluations of a capturing literal", c1, c2, false},
		{"tear-offs of the same receiver", a.m1, a.m1, true},
		{"the same method value", f, f, true},
		{"different receivers", a.m1, b.m1, false},
		{"different methods", a.m1, a.m2, false},
		{"generic receiver", g1.m, g1.m, true},
		{"different generic receivers", g1.m, g2.m, false},
		{"value receivers", a.v, a.v, false},
		{"method value and closure", a.m1, c1, false},
		{"nil and nil", nilFunc, nil, true},
		{"nil and a function", nilFunc, topLevel1, false},
		{"a function and nil", a.m1, nil, false},
	} {
		if got := FunctionsEqual(c.x, c.y); got != c.want {
			t.Errorf("%s: FunctionsEqual = %v, want %v", c.name, got, c.want)
		}
	}
	// Other func types work too.
	h := func(int) string { return "" }
	if !FunctionsEqual(h, h) || FunctionsEqual(h, func(int) string { return "x" }) {
		t.Error("func(int) string")
	}
}

func TestFunctionsEqualRejectsNonFunctions(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("no panic")
		}
	}()
	FunctionsEqual(1, 1)
}

func TestIdentityHashCode(t *testing.T) {
	a, b := &identityReceiver{}, &identityReceiver{}
	if IdentityHashCode(a) != IdentityHashCode(a) || IdentityHashCode(a) == IdentityHashCode(b) {
		t.Fatal("identity")
	}
	if IdentityHashCode("x") != IdentityHashCode("x") || IdentityHashCode("x") == IdentityHashCode("y") {
		t.Fatal("content")
	}
	if h := IdentityHashCode(a); h < 0 || h > 0x3FFFFFFF {
		t.Fatal(h)
	}
}
