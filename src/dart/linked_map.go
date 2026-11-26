package dart

import (
	"math"
	"reflect"
	"strings"
)

// LinkedMap is dart:core's default Map (a LinkedHashMap<Object?, Object?>):
// a mutable map that keeps its keys in insertion order (a key set again
// keeps its place; a removed key set again goes last), used by pointer so
// that, as in Dart, every holder sees mutations. Keys compare as Dart's ==
// does for the values platform messages carry: Go's == on comparable
// values, except that an int and an integral double are the same key
// (1 == 1.0, 0 == -0.0), and the first key given is the one kept. A key Go
// can't compare (a slice, a map) panics with UnsupportedError.
type LinkedMap struct {
	keys   []any
	values []any
	index  map[any]int
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

// linkedMapKey is the key a LinkedMap indexes by: integral doubles that an
// int can hold as ints, -0.0 as 0.
func linkedMapKey(key any) any {
	if key != nil && !reflect.TypeOf(key).Comparable() {
		panic(UnsupportedError{Message: "a map key of type " + reflect.TypeOf(key).String() + " can't be a Go map key"})
	}
	if f, ok := key.(float64); ok && f == math.Trunc(f) && f >= -0x1p63 && f < 0x1p63 {
		return int(f)
	}
	return key
}

// Len is the number of entries (Dart's length).
func (m *LinkedMap) Len() int { return len(m.keys) }

// IsEmpty is Dart's isEmpty.
func (m *LinkedMap) IsEmpty() bool { return len(m.keys) == 0 }

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
		m.index[linkedMapKey(m.keys[j])] = j
	}
	return value, true
}

// Clear is Dart's clear.
func (m *LinkedMap) Clear() {
	m.keys, m.values, m.index = nil, nil, nil
}

// Keys are the keys in order (a copy).
func (m *LinkedMap) Keys() []any { return append([]any(nil), m.keys...) }

// Values are the values in key order (a copy).
func (m *LinkedMap) Values() []any { return append([]any(nil), m.values...) }

// Entries are the entries in order (a copy).
func (m *LinkedMap) Entries() []MapEntry[any, any] {
	out := make([]MapEntry[any, any], len(m.keys))
	for i, k := range m.keys {
		out[i] = MapEntry[any, any]{Key: k, Value: m.values[i]}
	}
	return out
}

// String is Dart's Map.toString: {k: v, ...}.
func (m *LinkedMap) String() string {
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
