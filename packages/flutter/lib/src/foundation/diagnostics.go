package foundation

// Partial port of diagnostics.dart: shortHash and describeIdentity so far.

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	identityMu     sync.Mutex
	identityHashes = map[uintptr]int{}
	identityNext   = 0
)

// identityHashCode stands in for Dart's identityHashCode: a stable,
// arbitrary-looking number per object (pointer-like values; other values
// hash by content).
func identityHashCode(object any) int {
	v := reflect.ValueOf(object)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		identityMu.Lock()
		defer identityMu.Unlock()
		p := v.Pointer()
		h, ok := identityHashes[p]
		if !ok {
			identityNext++
			h = (identityNext * 0x9E3779B1) & 0x3FFFFFFF // spread consecutive ids
			identityHashes[p] = h
		}
		return h
	}
	return int(hashString(fmt.Sprintf("%#v", object)))
}

func hashString(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h & 0x3FFFFFFF
}

// ShortHash is shortHash: five hex digits identifying the object.
func ShortHash(object any) string {
	return fmt.Sprintf("%05x", identityHashCode(object)&0xFFFFF)
}

// DescribeIdentity is describeIdentity: the runtime type and short hash.
func DescribeIdentity(object any) string {
	return ObjectRuntimeType(object, "<optimized out>") + "#" + ShortHash(object)
}
