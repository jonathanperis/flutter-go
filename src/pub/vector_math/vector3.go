package vector_math

// Vector3 is a 3D column vector (PARTIAL: storage, accessors, toString and
// ==; see the package comment).
type Vector3 struct {
	v3storage [3]float64
}

// NewVector3 is Vector3(x, y, z).
func NewVector3(x, y, z float64) *Vector3 {
	v := NewVector3Zero()
	v.SetValues(x, y, z)
	return v
}

// NewVector3Zero is Vector3.zero.
func NewVector3Zero() *Vector3 { return &Vector3{} }

// NewVector3Copy is Vector3.copy.
func NewVector3Copy(other *Vector3) *Vector3 { return &Vector3{v3storage: other.v3storage} }

// Storage is the components of the vector, aliasing them.
func (v *Vector3) Storage() []float64 { return v.v3storage[:] }

// SetValues sets the values.
func (v *Vector3) SetValues(x, y, z float64) {
	v.v3storage[0] = x
	v.v3storage[1] = y
	v.v3storage[2] = z
}

// SetZero zeros the vector.
func (v *Vector3) SetZero() { v.v3storage = [3]float64{} }

// X is the x component.
func (v *Vector3) X() float64 { return v.v3storage[0] }

// Y is the y component.
func (v *Vector3) Y() float64 { return v.v3storage[1] }

// Z is the z component.
func (v *Vector3) Z() float64 { return v.v3storage[2] }

// SetX sets x.
func (v *Vector3) SetX(x float64) { v.v3storage[0] = x }

// SetY sets y.
func (v *Vector3) SetY(y float64) { v.v3storage[1] = y }

// SetZ sets z.
func (v *Vector3) SetZ(z float64) { v.v3storage[2] = z }

// String is "[x,y,z]".
func (v *Vector3) String() string { return vectorString(v.v3storage[:]) }

// Equal is operator ==.
func (v *Vector3) Equal(other *Vector3) bool {
	if v == nil || other == nil {
		return v == other
	}
	return v.v3storage == other.v3storage
}
