package vector_math

// Vector4 is a 4D column vector (PARTIAL: storage, accessors, toString and
// ==).
type Vector4 struct {
	v4storage [4]float64
}

// NewVector4 is Vector4(x, y, z, w).
func NewVector4(x, y, z, w float64) *Vector4 {
	return &Vector4{v4storage: [4]float64{x, y, z, w}}
}

// NewVector4Zero is Vector4.zero.
func NewVector4Zero() *Vector4 { return &Vector4{} }

// Storage is the components of the vector, aliasing them.
func (v *Vector4) Storage() []float64 { return v.v4storage[:] }

// X is the x component.
func (v *Vector4) X() float64 { return v.v4storage[0] }

// Y is the y component.
func (v *Vector4) Y() float64 { return v.v4storage[1] }

// Z is the z component.
func (v *Vector4) Z() float64 { return v.v4storage[2] }

// W is the w component.
func (v *Vector4) W() float64 { return v.v4storage[3] }

// String is "[x,y,z,w]".
func (v *Vector4) String() string { return vectorString(v.v4storage[:]) }

// Equal is operator ==.
func (v *Vector4) Equal(other *Vector4) bool {
	if v == nil || other == nil {
		return v == other
	}
	return v.v4storage == other.v4storage
}
