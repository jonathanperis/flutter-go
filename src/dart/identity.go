package dart

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// Object identity: dart:core's identityHashCode and `==` on functions,
// shared by every package that stands for a Dart library (foundation's
// shortHash and package:listen's ValueNotifier.toString must agree, as both
// read the same Object.hashCode in Dart).

var (
	identityMu     sync.Mutex
	identityHashes = map[uintptr]int{}
	identityNext   = 0
)

// IdentityHashCode stands in for Dart's identityHashCode (and the default
// Object.hashCode): a stable, arbitrary-looking number per object.
// Pointer-like values hash by address; other values hash by content.
func IdentityHashCode(object any) int {
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

// FunctionsEqual is Dart's `==` on two function values (what
// `removeListener(listener)` compares):
//
//   - the same top-level function, or the same closure value (the same
//     evaluation of a func literal), is equal to itself; two evaluations of
//     a func literal that captures variables are different closures, as in
//     Dart (one that captures nothing is a single value in Go);
//   - a method value of a pointer receiver, `x.M`, equals another `x.M` of
//     the same receiver pointer and method, as Dart's tear-offs `x.m == x.m`
//     (Go allocates a new closure for each evaluation, so this compares the
//     method and the receiver it captured);
//   - other method values (value receivers, interface method values) are
//     equal only to the same closure value: keep the value in a variable to
//     remove it later.
//
// nil equals only nil.
func FunctionsEqual[F any](a, b F) bool {
	pa, pb := closurePointer(a), closurePointer(b)
	if pa == pb {
		return true
	}
	if pa == nil || pb == nil {
		return false
	}
	code := *(*uintptr)(pa)
	if code != *(*uintptr)(pb) || !isPointerMethodValue(code) {
		return false
	}
	// A method value's closure is {code, receiver}: compare the receivers.
	const word = unsafe.Sizeof(uintptr(0))
	return *(*unsafe.Pointer)(unsafe.Add(pa, word)) == *(*unsafe.Pointer)(unsafe.Add(pb, word))
}

// closurePointer is the closure a function value points to (nil for a nil
// function). F must be a func type.
func closurePointer[F any](f F) unsafe.Pointer {
	if reflect.TypeFor[F]().Kind() != reflect.Func {
		panic("dart.FunctionsEqual: not a function type: " + reflect.TypeFor[F]().String())
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(&f))
}

var pointerMethodValues sync.Map // code pointer → bool

// isPointerMethodValue reports whether code is the wrapper the compiler
// generates for a method value of a pointer receiver: "pkg.(*T).M-fm".
func isPointerMethodValue(code uintptr) bool {
	if v, ok := pointerMethodValues.Load(code); ok {
		return v.(bool)
	}
	is := false
	if fn := runtime.FuncForPC(code); fn != nil {
		name := fn.Name()
		is = strings.HasSuffix(name, "-fm") && strings.Contains(name, ".(*")
	}
	pointerMethodValues.Store(code, is)
	return is
}
