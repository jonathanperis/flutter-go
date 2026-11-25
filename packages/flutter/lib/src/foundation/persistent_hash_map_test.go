package foundation

// Port of packages/flutter/test/foundation/persistent_hash_map_test.dart.

import (
	"fmt"
	"testing"
)

type mockKey struct {
	hash    int
	payload string
}

func (k mockKey) HashCode() int { return k.hash }

func TestPersistentHashMapSimple(t *testing.T) {
	maps := []PersistentHashMap[string, int]{NewPersistentHashMapEmpty[string, int]()}
	for i := 0; i < 50; i++ {
		maps = append(maps, maps[len(maps)-1].Put(fmt.Sprintf("key:%d", i), i))
	}
	for i := 1; i < len(maps); i++ {
		for j := 0; j < i; j++ {
			if v, ok := maps[i].Get(fmt.Sprintf("key:%d", j)); !ok || v != j {
				t.Fatalf("map %d key %d = %v, %v", i, j, v, ok)
			}
		}
	}
}

func TestPersistentHashMapCollisions(t *testing.T) {
	k1, k2, k3, k4 := mockKey{1, "key:1"}, mockKey{0 | 1<<5, "key:2"}, mockKey{1, "key:3"}, mockKey{1 | 1<<5, "key:4"}
	m := NewPersistentHashMapEmpty[mockKey, string]().Put(k1, "a").Put(k2, "b").Put(k3, "c")
	for k, want := range map[mockKey]string{k1: "a", k2: "b", k3: "c"} {
		if v, _ := m.Get(k); v != want {
			t.Errorf("%v = %q, want %q", k, v, want)
		}
	}
	m2 := m.Put(k4, "d")
	if v, _ := m2.Get(k4); v != "d" {
		t.Errorf("k4 = %q", v)
	}
	m3 := m2.Put(k1, "updated(a)").Put(k2, "updated(b)").Put(k3, "updated(c)").Put(k4, "updated(d)")
	for k, want := range map[mockKey]string{k1: "updated(a)", k2: "updated(b)", k3: "updated(c)", k4: "updated(d)"} {
		if v, _ := m3.Get(k); v != want {
			t.Errorf("%v = %q, want %q", k, v, want)
		}
	}
	// Persistence: the older maps are untouched.
	if v, _ := m.Get(k1); v != "a" {
		t.Errorf("original map changed: %q", v)
	}
	if _, ok := m.Get(k4); ok {
		t.Error("original map gained k4")
	}
}

func TestPersistentHashMapInflation(t *testing.T) {
	maps := []PersistentHashMap[mockKey, int]{NewPersistentHashMapEmpty[mockKey, int]()}
	for i := 0; i < 32*32; i++ {
		maps = append(maps, maps[len(maps)-1].Put(mockKey{i, fmt.Sprint(i)}, i))
	}
	for i := 1; i < len(maps); i++ {
		for j := 0; j < i; j++ {
			if v, ok := maps[i].Get(mockKey{j, fmt.Sprint(j)}); !ok || v != j {
				t.Fatalf("map %d key %d = %v, %v", i, j, v, ok)
			}
		}
	}
}

func TestPersistentHashMapPutSameValueKeepsMap(t *testing.T) {
	m := NewPersistentHashMapEmpty[string, int]().Put("a", 1)
	if m2 := m.Put("a", 1); m2.root != m.root {
		t.Error("putting an identical value must return the same map")
	}
}

// Beyond Flutter's tests: every intermediate map keeps exactly its own
// entries, through collisions, collision nodes and inflation.
func TestPersistentHashMapEveryVersionIsPersistent(t *testing.T) {
	keys := []mockKey{{1, "a"}, {1, "b"}, {1 | 1<<5, "c"}, {1, "d"}, {33, "e"}}
	for i := 0; i < 40; i++ {
		keys = append(keys, mockKey{i % 7, fmt.Sprint("x", i)})
	}
	versions := []PersistentHashMap[mockKey, int]{NewPersistentHashMapEmpty[mockKey, int]()}
	for i, k := range keys {
		versions = append(versions, versions[len(versions)-1].Put(k, i))
	}
	for v, m := range versions {
		for i, k := range keys {
			got, ok := m.Get(k)
			if i < v && (!ok || got != i) {
				t.Fatalf("version %d lost %v (got %v, %v)", v, k, got, ok)
			}
			if i >= v && ok {
				t.Fatalf("version %d has %v from a later put", v, k)
			}
		}
	}
}
