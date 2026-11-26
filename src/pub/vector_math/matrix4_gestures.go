package vector_math

// The Matrix4/Vector4 members the gestures library uses (PointerEvent's
// transforms and HitTestResult), ported from vector_math 2.4.4's
// matrix4.dart and vector4.dart in their arithmetic order.

// PerspectiveTransform transforms arg in place, dividing by the resulting w
// (perspectiveTransform), and returns it.
func (m *Matrix4) PerspectiveTransform(arg *Vector3) *Vector3 {
	s := &m.m4storage
	a := &arg.v3storage
	x := float64(s[0]*a[0]) + float64(s[4]*a[1]) + float64(s[8]*a[2]) + s[12]
	y := float64(s[1]*a[0]) + float64(s[5]*a[1]) + float64(s[9]*a[2]) + s[13]
	z := float64(s[2]*a[0]) + float64(s[6]*a[1]) + float64(s[10]*a[2]) + s[14]
	w := 1.0 / (float64(s[3]*a[0]) + float64(s[7]*a[1]) + float64(s[11]*a[2]) + s[15])
	a[0] = float64(x * w)
	a[1] = float64(y * w)
	a[2] = float64(z * w)
	return arg
}

// SetRow sets row row to arg.
func (m *Matrix4) SetRow(row int, arg *Vector4) {
	m.m4storage[m.Index(row, 0)] = arg.v4storage[0]
	m.m4storage[m.Index(row, 1)] = arg.v4storage[1]
	m.m4storage[m.Index(row, 2)] = arg.v4storage[2]
	m.m4storage[m.Index(row, 3)] = arg.v4storage[3]
}

// SetColumn sets column column to arg.
func (m *Matrix4) SetColumn(column int, arg *Vector4) {
	entry := column * 4
	m.m4storage[entry+3] = arg.v4storage[3]
	m.m4storage[entry+2] = arg.v4storage[2]
	m.m4storage[entry+1] = arg.v4storage[1]
	m.m4storage[entry+0] = arg.v4storage[0]
}

// GetColumn returns column column.
func (m *Matrix4) GetColumn(column int) *Vector4 {
	r := NewVector4Zero()
	entry := column * 4
	r.v4storage[3] = m.m4storage[entry+3]
	r.v4storage[2] = m.m4storage[entry+2]
	r.v4storage[1] = m.m4storage[entry+1]
	r.v4storage[0] = m.m4storage[entry+0]
	return r
}

// LeftTranslateByDouble multiplies this matrix from the left by a
// translation by (tx, ty, tz) with w = tw (leftTranslateByDouble).
func (m *Matrix4) LeftTranslateByDouble(tx, ty, tz, tw float64) {
	s := &m.m4storage
	for c := 0; c < 4; c++ {
		r := s[c*4+3]
		s[c*4+0] += float64(tx * r)
		s[c*4+1] += float64(ty * r)
		s[c*4+2] += float64(tz * r)
		s[c*4+3] = float64(tw * r)
	}
}

// Clone returns a copy of the vector.
func (v *Vector4) Clone() *Vector4 { return &Vector4{v4storage: v.v4storage} }

// Sub returns v - other (operator -).
func (v *Vector4) Sub(other *Vector4) *Vector4 {
	r := v.Clone()
	r.v4storage[3] -= other.v4storage[3]
	r.v4storage[2] -= other.v4storage[2]
	r.v4storage[1] -= other.v4storage[1]
	r.v4storage[0] -= other.v4storage[0]
	return r
}
