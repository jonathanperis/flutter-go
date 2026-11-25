package dart

// MapEntry is dart:core's MapEntry<K, V>. A slice of entries stands in for
// a Dart Map where its insertion order matters (Go maps have no order).
type MapEntry[K comparable, V any] struct {
	Key   K
	Value V
}
