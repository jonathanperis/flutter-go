package dart

// The Dart VM's String.hashCode and Null.hashCode (runtime/vm/hash.h's
// CombineHashes/FinalizeHash over the string's UTF-16 code units, 30 bits:
// String::kHashBits), for the few places Flutter puts a hash code into a
// platform message (text_input.dart's IOSSystemContextMenuItemData).

// NullHashCode is null.hashCode in the Dart VM.
const NullHashCode = 2011

// StringHashCode is String.hashCode as the Dart VM computes it.
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
