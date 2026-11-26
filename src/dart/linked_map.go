package dart

import (
	"iter"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// LinkedMap is dart:core's default Map (a LinkedHashMap<Object?, Object?>,
// what a `{}` literal, json.decode and the platform-channel codecs build):
// a mutable map that keeps its keys in insertion order (a key set again
// keeps its place; a removed key set again goes last), used by pointer so
// that, as in Dart, every holder sees mutations and `==` is identity. Its
// zero value is an empty map.
//
// Keys compare as Dart's == does for the values platform messages carry:
//
//   - null, bools and strings by value;
//   - numbers by value across int and double: an int and an integral double
//     are the same key when Dart's int == double holds and their hashCodes
//     agree (1 and 1.0; 0, 0.0 and -0.0; 2^63 and the largest int, as the
//     VM's double hash saturates), and the first key given is the one kept;
//     NaN is never equal to itself, so each NaN key is a new entry;
//   - lists, typed data and Go maps by identity (Dart's List and Map == is
//     identical): two slices are the same key only when they share their
//     backing array, length and capacity, so two zero-capacity slices are
//     the same key; a *LinkedMap is a pointer, compared by identity;
//   - other Go values with Go's ==. A key Go can't compare otherwise (a
//     struct or array holding a slice) panics with UnsupportedError.
//
// Iterating (Keys, Values, Entries) while the map is changed panics with
// ConcurrentModificationError when the iteration resumes, as Dart's
// iterators throw.
type LinkedMap struct {
	keys   []any
	values []any
	index  map[any]int
	// modifications counts structural changes (adding or removing a key),
	// Dart's _modificationCount.
	modifications int
}

// NewLinkedMap is a map literal: the entries in order (a repeated key
// keeps its first place and its last value).
func NewLinkedMap(entries ...MapEntry[any, any]) *LinkedMap {
	m := &LinkedMap{}
	for _, e := range entries {
		m.Set(e.Key, e.Value)
	}
	return m
}

// NewLinkedMapOf is a map literal from alternating keys and values
// (key, value, key, value...); it panics on an odd count.
func NewLinkedMapOf(keysAndValues ...any) *LinkedMap {
	if len(keysAndValues)%2 != 0 {
		panic(ArgumentError{Message: "an odd number of keys and values"})
	}
	m := &LinkedMap{}
	for i := 0; i < len(keysAndValues); i += 2 {
		m.Set(keysAndValues[i], keysAndValues[i+1])
	}
	return m
}

// identityKey indexes a key Dart compares by identity and Go can't hash: a
// slice (its backing array, length and capacity) or a Go map.
type identityKey struct {
	typ           reflect.Type
	ptr           uintptr
	length, capac int
}

// linkedMapKey is the key a LinkedMap indexes by: an integral double that
// matches an int key as that int (see LinkedMap), -0.0 as 0, a slice or map
// by identity.
func linkedMapKey(key any) any {
	switch k := key.(type) {
	case nil, string, bool, int:
		return key
	case float64:
		if k == math.Trunc(k) && k >= -0x1p63 && k <= 0x1p63 {
			// The VM hashes an integral double as the int it converts to
			// (static_cast<int64_t>, saturating at 2^63); the two keys match
			// when that int also equals the double.
			i := math.MaxInt64
			if k < 0x1p63 {
				i = int(k)
			}
			if float64(i) == k {
				return i
			}
		}
		return key
	}
	rv := reflect.ValueOf(key)
	switch rv.Kind() {
	case reflect.Slice:
		return identityKey{typ: rv.Type(), ptr: rv.Pointer(), length: rv.Len(), capac: rv.Cap()}
	case reflect.Map:
		return identityKey{typ: rv.Type(), ptr: rv.Pointer()}
	}
	if !rv.Type().Comparable() {
		panic(UnsupportedError{Message: "a map key of type " + rv.Type().String() + " can't be compared"})
	}
	return key
}

// Len is Dart's length.
func (m *LinkedMap) Len() int { return len(m.keys) }

// IsEmpty is Dart's isEmpty.
func (m *LinkedMap) IsEmpty() bool { return len(m.keys) == 0 }

// IsNotEmpty is Dart's isNotEmpty.
func (m *LinkedMap) IsNotEmpty() bool { return len(m.keys) != 0 }

// Get returns the value for key and whether the key is present.
func (m *LinkedMap) Get(key any) (value any, ok bool) {
	i, ok := m.index[linkedMapKey(key)]
	if !ok {
		return nil, false
	}
	return m.values[i], true
}

// Lookup is Dart's map[key]: the value, or nil (null) when absent.
func (m *LinkedMap) Lookup(key any) any {
	v, _ := m.Get(key)
	return v
}

// ContainsKey is Dart's containsKey.
func (m *LinkedMap) ContainsKey(key any) bool {
	_, ok := m.index[linkedMapKey(key)]
	return ok
}

// Set is Dart's map[key] = value.
func (m *LinkedMap) Set(key, value any) {
	k := linkedMapKey(key)
	if i, ok := m.index[k]; ok {
		m.values[i] = value
		return
	}
	if m.index == nil {
		m.index = map[any]int{}
	}
	m.index[k] = len(m.keys)
	m.keys = append(m.keys, key)
	m.values = append(m.values, value)
	m.modifications++
}

// PutIfAbsent is Dart's putIfAbsent: the value for key, set to ifAbsent()
// first when absent.
func (m *LinkedMap) PutIfAbsent(key any, ifAbsent func() any) any {
	if v, ok := m.Get(key); ok {
		return v
	}
	v := ifAbsent()
	m.Set(key, v)
	return v
}

// AddAll is Dart's addAll: other's entries set in other's order.
func (m *LinkedMap) AddAll(other *LinkedMap) {
	for k, v := range other.Entries() {
		m.Set(k, v)
	}
}

// Remove is Dart's remove: the removed value (nil when absent) and whether
// the key was present.
func (m *LinkedMap) Remove(key any) (value any, ok bool) {
	k := linkedMapKey(key)
	i, ok := m.index[k]
	if !ok {
		return nil, false
	}
	value = m.values[i]
	delete(m.index, k)
	m.keys = append(m.keys[:i:i], m.keys[i+1:]...)
	m.values = append(m.values[:i:i], m.values[i+1:]...)
	for j := i; j < len(m.keys); j++ {
		if jk := linkedMapKey(m.keys[j]); jk == jk { // NaN keys aren't indexed
			m.index[jk] = j
		}
	}
	m.modifications++
	return value, true
}

// Clear is Dart's clear.
func (m *LinkedMap) Clear() {
	if len(m.keys) > 0 {
		m.modifications++
	}
	m.keys, m.values, m.index = nil, nil, nil
}

// Keys is Dart's keys, in order.
func (m *LinkedMap) Keys() iter.Seq[any] {
	return func(yield func(any) bool) {
		for k := range m.Entries() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values is Dart's values, in key order.
func (m *LinkedMap) Values() iter.Seq[any] {
	return func(yield func(any) bool) {
		for _, v := range m.Entries() {
			if !yield(v) {
				return
			}
		}
	}
}

// Entries is Dart's entries (and forEach), in order, as key-value pairs.
func (m *LinkedMap) Entries() iter.Seq2[any, any] {
	return func(yield func(any, any) bool) {
		modifications := m.modifications
		for i := 0; i < len(m.keys); i++ {
			if !yield(m.keys[i], m.values[i]) {
				return
			}
			if m.modifications != modifications {
				panic(ConcurrentModificationError{ModifiedObject: m})
			}
		}
	}
}

// KeyList is keys.toList(): the keys in order (a copy).
func (m *LinkedMap) KeyList() []any { return append([]any(nil), m.keys...) }

// EntryList is entries.toList(): the entries in order (a copy).
func (m *LinkedMap) EntryList() []MapEntry[any, any] {
	out := make([]MapEntry[any, any], len(m.keys))
	for i, k := range m.keys {
		out[i] = MapEntry[any, any]{Key: k, Value: m.values[i]}
	}
	return out
}

// toStringVisiting are the maps whose toString is running (Dart's
// _toStringVisiting): a map met again inside itself prints as {...}.
var toStringVisiting []*LinkedMap

// String is Dart's Map.toString (MapBase.mapToString): {k: v, ...}.
func (m *LinkedMap) String() string {
	for _, v := range toStringVisiting {
		if v == m {
			return "{...}"
		}
	}
	toStringVisiting = append(toStringVisiting, m)
	defer func() { toStringVisiting = toStringVisiting[:len(toStringVisiting)-1] }()
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(ToString(k))
		b.WriteString(": ")
		b.WriteString(ToString(m.values[i]))
	}
	b.WriteByte('}')
	return b.String()
}

// ConcurrentModificationError is dart:core's ConcurrentModificationError:
// the panic value when a collection is changed while it is iterated.
type ConcurrentModificationError struct {
	ModifiedObject any
}

func (e ConcurrentModificationError) Error() string {
	if e.ModifiedObject == nil {
		return "Concurrent modification during iteration."
	}
	if m, ok := e.ModifiedObject.(*LinkedMap); ok {
		return "Concurrent modification during iteration: _Map len:" + strconv.Itoa(m.Len()) + "."
	}
	return "Concurrent modification during iteration: " + SafeToString(e.ModifiedObject) + "."
}
