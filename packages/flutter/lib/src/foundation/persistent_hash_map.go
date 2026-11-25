package foundation

// Port of persistent_hash_map.dart: an immutable hash array mapped trie.
// Keys hash with maphash.Comparable, unless they provide their own
// HashCode() (Dart's hashCode override), which tests use to force
// collisions.

import (
	"hash/maphash"
	"math/bits"
	"reflect"
)

// Hasher is implemented by keys that define their own hash (Dart's
// hashCode). Equal keys must return equal hashes.
type Hasher interface{ HashCode() int }

var mapSeed = maphash.MakeSeed()

func hashOf[K comparable](key K) int {
	if h, ok := any(key).(Hasher); ok {
		return h.HashCode() & 0xFFFFFFFF
	}
	return int(maphash.Comparable(mapSeed, key) & 0xFFFFFFFF)
}

// PersistentHashMap is an immutable map; Put returns a new map (or the same
// one when nothing changed) and leaves the original untouched.
type PersistentHashMap[K comparable, V any] struct{ root trieNode[K, V] }

// NewPersistentHashMapEmpty is PersistentHashMap.empty().
func NewPersistentHashMapEmpty[K comparable, V any]() PersistentHashMap[K, V] {
	return PersistentHashMap[K, V]{}
}

func (m PersistentHashMap[K, V]) Put(key K, value V) PersistentHashMap[K, V] {
	root := m.root
	if root == nil {
		root = emptyCompressed[K, V]()
	}
	newRoot := root.put(0, key, hashOf(key), value)
	if newRoot == m.root {
		return m
	}
	return PersistentHashMap[K, V]{newRoot}
}

// Get is the [] operator: the value and whether the key is present (Dart
// returns null when absent).
func (m PersistentHashMap[K, V]) Get(key K) (V, bool) {
	if m.root == nil {
		var zero V
		return zero, false
	}
	return m.root.get(0, key, hashOf(key))
}

const (
	hashBitsPerLevel     = 5
	hashBitsPerLevelMask = (1 << hashBitsPerLevel) - 1
	fullNodeElements     = 1 << hashBitsPerLevel
)

func trieIndex(hash, bitIndex int) int { return (hash >> bitIndex) & hashBitsPerLevelMask }

type trieNode[K comparable, V any] interface {
	put(bitIndex int, key K, keyHash int, value V) trieNode[K, V]
	get(bitIndex int, key K, keyHash int) (V, bool)
}

// identical is Dart's identical() for values: pointer identity for
// reference-like kinds, == otherwise.
func identical(a, b any) bool {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if !va.IsValid() || !vb.IsValid() {
		return !va.IsValid() && !vb.IsValid()
	}
	if va.Type() != vb.Type() {
		return false
	}
	switch va.Kind() {
	case reflect.Slice, reflect.Map, reflect.Func, reflect.Pointer, reflect.Chan, reflect.UnsafePointer:
		return va.Pointer() == vb.Pointer()
	}
	if va.Type().Comparable() {
		return a == b
	}
	return false
}

type fullNode[K comparable, V any] struct {
	descendants [fullNodeElements]trieNode[K, V]
}

func (n *fullNode[K, V]) put(bitIndex int, key K, keyHash int, value V) trieNode[K, V] {
	index := trieIndex(keyHash, bitIndex)
	node := n.descendants[index]
	if node == nil {
		node = emptyCompressed[K, V]()
	}
	newNode := node.put(bitIndex+hashBitsPerLevel, key, keyHash, value)
	if newNode == node {
		return n
	}
	c := *n
	c.descendants[index] = newNode
	return &c
}

func (n *fullNode[K, V]) get(bitIndex int, key K, keyHash int) (V, bool) {
	if node := n.descendants[trieIndex(keyHash, bitIndex)]; node != nil {
		return node.get(bitIndex+hashBitsPerLevel, key, keyHash)
	}
	var zero V
	return zero, false
}

// entry is a key/value pair, or (key absent) a pointer to a sub-node:
// Dart stores null in the key slot for that case.
type entry[K comparable, V any] struct {
	hasKey bool
	key    K
	value  V
	node   trieNode[K, V]
}

type compressedNode[K comparable, V any] struct {
	occupiedIndices int
	entries         []entry[K, V]
}

func emptyCompressed[K comparable, V any]() *compressedNode[K, V] { return &compressedNode[K, V]{} }

func (n *compressedNode[K, V]) compressedIndex(bit int) int {
	return bits.OnesCount32(uint32(n.occupiedIndices & (bit - 1)))
}

func (n *compressedNode[K, V]) put(bitIndex int, key K, keyHash int, value V) trieNode[K, V] {
	bit := 1 << trieIndex(keyHash, bitIndex)
	index := n.compressedIndex(bit)
	if n.occupiedIndices&bit != 0 {
		e := n.entries[index]
		if !e.hasKey {
			newNode := e.node.put(bitIndex+hashBitsPerLevel, key, keyHash, value)
			if newNode == e.node {
				return n
			}
			c := n.clone()
			c.entries[index].node = newNode
			return c
		}
		if key == e.key {
			if identical(value, e.value) {
				return n
			}
			c := n.clone()
			c.entries[index].value = value
			return c
		}
		newNode := resolveCollision(bitIndex+hashBitsPerLevel, e.key, e.value, key, keyHash, value)
		c := n.clone()
		c.entries[index] = entry[K, V]{node: newNode}
		return c
	}
	occupied := bits.OnesCount32(uint32(n.occupiedIndices))
	if occupied >= 16 {
		f := n.inflate(bitIndex)
		f.descendants[trieIndex(keyHash, bitIndex)] =
			emptyCompressed[K, V]().put(bitIndex+hashBitsPerLevel, key, keyHash, value)
		return f
	}
	entries := make([]entry[K, V], 0, len(n.entries)+1)
	entries = append(entries, n.entries[:index]...)
	entries = append(entries, entry[K, V]{hasKey: true, key: key, value: value})
	entries = append(entries, n.entries[index:]...)
	return &compressedNode[K, V]{n.occupiedIndices | bit, entries}
}

func (n *compressedNode[K, V]) clone() *compressedNode[K, V] {
	return &compressedNode[K, V]{n.occupiedIndices, append([]entry[K, V](nil), n.entries...)}
}

func (n *compressedNode[K, V]) get(bitIndex int, key K, keyHash int) (V, bool) {
	var zero V
	bit := 1 << trieIndex(keyHash, bitIndex)
	if n.occupiedIndices&bit == 0 {
		return zero, false
	}
	e := n.entries[n.compressedIndex(bit)]
	if !e.hasKey {
		return e.node.get(bitIndex+hashBitsPerLevel, key, keyHash)
	}
	if key == e.key {
		return e.value, true
	}
	return zero, false
}

func (n *compressedNode[K, V]) inflate(bitIndex int) *fullNode[K, V] {
	f := &fullNode[K, V]{}
	src := 0
	for dst := 0; dst < fullNodeElements; dst++ {
		if (n.occupiedIndices>>dst)&1 != 0 {
			e := n.entries[src]
			if !e.hasKey {
				f.descendants[dst] = e.node
			} else {
				f.descendants[dst] = emptyCompressed[K, V]().put(bitIndex+hashBitsPerLevel, e.key, hashOf(e.key), e.value)
			}
			src++
		}
	}
	return f
}

func resolveCollision[K comparable, V any](bitIndex int, existingKey K, existingValue V, key K, keyHash int, value V) trieNode[K, V] {
	existingHash := hashOf(existingKey)
	if existingHash == keyHash {
		return &hashCollisionNode[K, V]{keyHash, []entry[K, V]{
			{hasKey: true, key: existingKey, value: existingValue}, {hasKey: true, key: key, value: value}}}
	}
	return emptyCompressed[K, V]().put(bitIndex, existingKey, existingHash, existingValue).put(bitIndex, key, keyHash, value)
}

type hashCollisionNode[K comparable, V any] struct {
	hash    int
	entries []entry[K, V]
}

func (n *hashCollisionNode[K, V]) indexOf(key K) int {
	for i, e := range n.entries {
		if e.key == key {
			return i
		}
	}
	return -1
}

func (n *hashCollisionNode[K, V]) put(bitIndex int, key K, keyHash int, value V) trieNode[K, V] {
	if keyHash == n.hash {
		if i := n.indexOf(key); i != -1 {
			if identical(n.entries[i].value, value) {
				return n
			}
			c := &hashCollisionNode[K, V]{keyHash, append([]entry[K, V](nil), n.entries...)}
			c.entries[i].value = value
			return c
		}
		entries := append(append([]entry[K, V](nil), n.entries...), entry[K, V]{hasKey: true, key: key, value: value})
		return &hashCollisionNode[K, V]{keyHash, entries}
	}
	single := &compressedNode[K, V]{1 << trieIndex(n.hash, bitIndex), []entry[K, V]{{node: n}}}
	return single.put(bitIndex, key, keyHash, value)
}

func (n *hashCollisionNode[K, V]) get(bitIndex int, key K, keyHash int) (V, bool) {
	if i := n.indexOf(key); i >= 0 {
		return n.entries[i].value, true
	}
	var zero V
	return zero, false
}
