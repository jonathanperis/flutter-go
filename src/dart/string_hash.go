package dart

// StringHashCode is String.hashCode as the Dart VM computes it (runtime/vm
// hash.h: each UTF-16 code unit combined with CombineHashes, then
// FinalizeHash to String::kHashBits = 30 bits; 0 becomes 1).
func StringHashCode(s string) int {
	var hash uint32
	for _, u := range utf16Units(s) {
		hash += uint32(u)
		hash += hash << 10
		hash ^= hash >> 6
	}
	hash += hash << 3
	hash ^= hash >> 11
	hash += hash << 15
	hash &= (1 << 30) - 1
	if hash == 0 {
		return 1
	}
	return int(hash)
}
