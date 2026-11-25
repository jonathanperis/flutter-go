package foundation

import (
	"fmt"
	"reflect"
)

// Key identifies widgets, elements and semantics nodes across rebuilds.
// Keys compare with ==; implementations must be comparable.
type Key interface{ isKey() }

// LocalKey is a key that is unique only among siblings (as opposed to a
// GlobalKey, which lives in widgets).
type LocalKey interface {
	Key
	isLocalKey()
}

// LocalKeyBase is embedded by local key types.
type LocalKeyBase struct{}

func (LocalKeyBase) isKey()      {}
func (LocalKeyBase) isLocalKey() {}

// NewKey is Key(value): a ValueKey<String>.
func NewKey(value string) Key { return ValueKey[string]{Value: value} }

// UniqueKey is equal only to itself (compare *UniqueKey pointers).
type UniqueKey struct {
	LocalKeyBase
	_ byte // non-zero size, so distinct UniqueKeys never share an address
}

func NewUniqueKey() *UniqueKey { return &UniqueKey{} }

func (k *UniqueKey) String() string { return "[#" + ShortHash(k) + "]" }

// ValueKey is a key that uses a value of type T to identify itself; two
// ValueKeys are equal when their types and values are.
type ValueKey[T comparable] struct {
	LocalKeyBase
	Value T
}

func NewValueKey[T comparable](value T) ValueKey[T] { return ValueKey[T]{Value: value} }

func (k ValueKey[T]) String() string {
	if reflect.TypeFor[T]().Kind() == reflect.String {
		return fmt.Sprintf("[<'%v'>]", k.Value)
	}
	return fmt.Sprintf("[<%v>]", k.Value)
}
