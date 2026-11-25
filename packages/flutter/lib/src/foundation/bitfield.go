package foundation

// KMaxUnsignedSMI is the largest unsigned value the Dart VM stores as a
// small integer; BitField uses 62 bits, as Flutter's VM implementation.
const KMaxUnsignedSMI = 0x3FFFFFFFFFFFFFFF

const smiBits = 62

// BitField is a set of boolean flags indexed by an enum (any ~int type).
type BitField[T ~int] struct {
	length int
	bits   int
}

// NewBitField is BitField(length): all flags false.
func NewBitField[T ~int](length int) *BitField[T] {
	if length > smiBits {
		panic("foundation: BitField supports at most 62 flags")
	}
	return &BitField[T]{length: length}
}

// NewBitFieldFilled is BitField.filled(length, value).
func NewBitFieldFilled[T ~int](length int, value bool) *BitField[T] {
	b := NewBitField[T](length)
	b.Reset(value)
	return b
}

// Get is the [] operator.
func (b *BitField[T]) Get(index T) bool {
	i := int(index)
	if i >= b.length {
		panic("foundation: BitField index out of range")
	}
	return b.bits&(1<<i) > 0
}

// Set is the []= operator.
func (b *BitField[T]) Set(index T, value bool) {
	i := int(index)
	if i >= b.length {
		panic("foundation: BitField index out of range")
	}
	if value {
		b.bits |= 1 << i
	} else {
		b.bits &^= 1 << i
	}
}

// Reset sets every flag to value (default false).
func (b *BitField[T]) Reset(value ...bool) {
	b.bits = 0
	if len(value) > 0 && value[0] {
		b.bits = KMaxUnsignedSMI
	}
}
