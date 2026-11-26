// Package vector_math is package:vector_math's vector_math_64 library
// (version 2.4.4, the one in the pub cache; Flutter 3.50.1's pubspec.lock
// pins 2.4.3, whose Matrix4/Vector3/Vector4 code ported here is the same),
// re-exported by Flutter (`semantics.dart` exports Matrix4).
//
// PARTIAL: only what flutter-go's ports use so far: Matrix4's storage,
// constructors (zero, identity, copy, inverted, fromList, translationValues,
// diagonal3Values, rotationZ, the 16-value constructor), entry access,
// setValues/setFrom/setZero/setIdentity/setTranslationRaw/setRotationZ,
// multiply/multiplied, invert/copyInverse/tryInvert, determinant,
// translateByDouble, scaleByDouble, transform3, getRow, isIdentity, isZero,
// toString and ==; Vector3 and Vector4 as values with storage, toString and
// ==. The rest of the library (Matrix2/3, quaternions, the other Matrix4
// operations, colors, geometry) is not ported yet.
//
// Like the Dart classes, Matrix4 and the vectors are mutable objects used by
// pointer; Storage aliases the values (Dart's Float64List storage).
//
//nolint:revive // the package keeps the Dart package's name.
package vector_math

import (
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Matrix4 is a 4D matrix. Values are stored in column major order.
type Matrix4 struct {
	m4storage [16]float64
}

// NewMatrix4 constructs a new mat4 from its 16 values in column major
// order (Dart's Matrix4(arg0, ..., arg15)).
func NewMatrix4(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15 float64) *Matrix4 {
	m := NewMatrix4Zero()
	m.SetValues(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15)
	return m
}

// NewMatrix4FromList is Matrix4.fromList: a matrix from 16 values in column
// major order.
func NewMatrix4FromList(values []float64) *Matrix4 {
	return NewMatrix4(values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7],
		values[8], values[9], values[10], values[11], values[12], values[13], values[14], values[15])
}

// NewMatrix4Zero is Matrix4.zero: the zero matrix.
func NewMatrix4Zero() *Matrix4 { return &Matrix4{} }

// NewMatrix4Identity is Matrix4.identity: the identity matrix.
func NewMatrix4Identity() *Matrix4 {
	m := NewMatrix4Zero()
	m.m4storage[0] = 1.0
	m.m4storage[5] = 1.0
	m.m4storage[10] = 1.0
	m.m4storage[15] = 1.0
	return m
}

// NewMatrix4Copy is Matrix4.copy: a copy of other.
func NewMatrix4Copy(other *Matrix4) *Matrix4 {
	m := NewMatrix4Zero()
	m.SetFrom(other)
	return m
}

// NewMatrix4Inverted is Matrix4.inverted: the inverse of other. It panics
// with Dart's ArgumentError when other can't be inverted.
func NewMatrix4Inverted(other *Matrix4) *Matrix4 {
	r := NewMatrix4Zero()
	determinant := r.CopyInverse(other)
	if determinant == 0.0 {
		panic(dart.NewArgumentErrorValue(other, "other", "Matrix cannot be inverted"))
	}
	return r
}

// NewMatrix4RotationZ is Matrix4.rotationZ: a rotation of radians around Z.
func NewMatrix4RotationZ(radians float64) *Matrix4 {
	m := NewMatrix4Zero()
	m.m4storage[15] = 1.0
	m.SetRotationZ(radians)
	return m
}

// NewMatrix4TranslationValues is Matrix4.translationValues: a translation
// matrix.
func NewMatrix4TranslationValues(x, y, z float64) *Matrix4 {
	m := NewMatrix4Zero()
	m.SetIdentity()
	m.SetTranslationRaw(x, y, z)
	return m
}

// NewMatrix4Translation is Matrix4.translation: a translation matrix.
func NewMatrix4Translation(translation *Vector3) *Matrix4 {
	m := NewMatrix4Zero()
	m.SetIdentity()
	m.SetTranslationRaw(translation.X(), translation.Y(), translation.Z())
	return m
}

// NewMatrix4Diagonal3 is Matrix4.diagonal3: a scale matrix.
func NewMatrix4Diagonal3(scale *Vector3) *Matrix4 {
	m := NewMatrix4Zero()
	m.m4storage[15] = 1.0
	m.m4storage[10] = scale.Z()
	m.m4storage[5] = scale.Y()
	m.m4storage[0] = scale.X()
	return m
}

// NewMatrix4Diagonal3Values is Matrix4.diagonal3Values: a scale matrix.
func NewMatrix4Diagonal3Values(x, y, z float64) *Matrix4 {
	m := NewMatrix4Zero()
	m.m4storage[15] = 1.0
	m.m4storage[10] = z
	m.m4storage[5] = y
	m.m4storage[0] = x
	return m
}

// Matrix4TryInvert is Matrix4.tryInvert: the inverse of other, or nil when
// other is not invertible.
func Matrix4TryInvert(other *Matrix4) *Matrix4 {
	r := NewMatrix4Zero()
	determinant := r.CopyInverse(other)
	if determinant == 0.0 {
		return nil
	}
	return r
}

// Storage is the components of the matrix (column major), aliasing them.
func (m *Matrix4) Storage() []float64 { return m.m4storage[:] }

// Dimension is the dimension of the matrix (4).
func (m *Matrix4) Dimension() int { return 4 }

// Index is the index in storage of the value at row, col.
func (m *Matrix4) Index(row, col int) int { return (col * 4) + row }

// Entry is the value at row, col.
func (m *Matrix4) Entry(row, col int) float64 {
	if row < 0 || row >= m.Dimension() || col < 0 || col >= m.Dimension() {
		panic(dart.NewAssertionError(nil))
	}
	return m.m4storage[m.Index(row, col)]
}

// SetEntry sets the value at row, col to v.
func (m *Matrix4) SetEntry(row, col int, v float64) {
	if row < 0 || row >= m.Dimension() || col < 0 || col >= m.Dimension() {
		panic(dart.NewAssertionError(nil))
	}
	m.m4storage[m.Index(row, col)] = v
}

// At is operator [](i): the element at storage index i.
func (m *Matrix4) At(i int) float64 { return m.m4storage[i] }

// SetAt is operator []=(i, v).
func (m *Matrix4) SetAt(i int, v float64) { m.m4storage[i] = v }

// SetValues sets the matrix with 16 values in column major order.
func (m *Matrix4) SetValues(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15 float64) {
	m.m4storage = [16]float64{arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10, arg11, arg12, arg13, arg14, arg15}
}

// SetFrom copies the values of arg.
func (m *Matrix4) SetFrom(arg *Matrix4) { m.m4storage = arg.m4storage }

// Clone is a copy of this.
func (m *Matrix4) Clone() *Matrix4 { return NewMatrix4Copy(m) }

// String is the printable form: one "[i] x,y,z,w" line per row (each value
// in Dart's double.toString), each line ending with a newline.
func (m *Matrix4) String() string {
	return "[0] " + m.GetRow(0).String() + "\n[1] " + m.GetRow(1).String() + "\n" +
		"[2] " + m.GetRow(2).String() + "\n[3] " + m.GetRow(3).String() + "\n"
}

// Equal is operator ==: the 16 values are equal (NaN is never equal; -0.0
// equals 0.0). A nil matrix equals only nil.
func (m *Matrix4) Equal(other *Matrix4) bool {
	if m == nil || other == nil {
		return m == other
	}
	return m.m4storage == other.m4storage
}

// GetRow is the row of the matrix.
func (m *Matrix4) GetRow(row int) *Vector4 {
	r := NewVector4Zero()
	r.v4storage[0] = m.m4storage[m.Index(row, 0)]
	r.v4storage[1] = m.m4storage[m.Index(row, 1)]
	r.v4storage[2] = m.m4storage[m.Index(row, 2)]
	r.v4storage[3] = m.m4storage[m.Index(row, 3)]
	return r
}

// TranslateByDouble translates this matrix by (tx, ty, tz, tw).
func (m *Matrix4) TranslateByDouble(tx, ty, tz, tw float64) {
	s := &m.m4storage
	t1 := float64(s[0]*tx) + float64(s[4]*ty) + float64(s[8]*tz) + float64(s[12]*tw)
	s[12] = t1
	t2 := float64(s[1]*tx) + float64(s[5]*ty) + float64(s[9]*tz) + float64(s[13]*tw)
	s[13] = t2
	t3 := float64(s[2]*tx) + float64(s[6]*ty) + float64(s[10]*tz) + float64(s[14]*tw)
	s[14] = t3
	t4 := float64(s[3]*tx) + float64(s[7]*ty) + float64(s[11]*tz) + float64(s[15]*tw)
	s[15] = t4
}

// ScaleByDouble scales this matrix by (sx, sy, sz, sw).
func (m *Matrix4) ScaleByDouble(sx, sy, sz, sw float64) {
	s := &m.m4storage
	for i := 0; i < 4; i++ {
		s[i] = float64(s[i] * sx)
		s[4+i] = float64(s[4+i] * sy)
		s[8+i] = float64(s[8+i] * sz)
		s[12+i] = float64(s[12+i] * sw)
	}
}

// SetZero zeros this.
func (m *Matrix4) SetZero() { m.m4storage = [16]float64{} }

// SetIdentity makes this into the identity matrix.
func (m *Matrix4) SetIdentity() {
	m.m4storage = [16]float64{1.0, 0, 0, 0, 0, 1.0, 0, 0, 0, 0, 1.0, 0, 0, 0, 0, 1.0}
}

// SetTranslationRaw sets the translation vector in this homogeneous
// transformation matrix.
func (m *Matrix4) SetTranslationRaw(x, y, z float64) {
	m.m4storage[14] = z
	m.m4storage[13] = y
	m.m4storage[12] = x
}

// SetRotationZ sets the upper 3x3 to a rotation of radians around Z.
func (m *Matrix4) SetRotationZ(radians float64) {
	c := dart.Cos(radians)
	s := dart.Sin(radians)
	m.m4storage[0] = c
	m.m4storage[1] = s
	m.m4storage[2] = 0.0
	m.m4storage[4] = -s
	m.m4storage[5] = c
	m.m4storage[6] = 0.0
	m.m4storage[8] = 0.0
	m.m4storage[9] = 0.0
	m.m4storage[10] = 1.0
	m.m4storage[3] = 0.0
	m.m4storage[7] = 0.0
	m.m4storage[11] = 0.0
}

// Determinant is the determinant of this matrix.
func (m *Matrix4) Determinant() float64 {
	s := &m.m4storage
	det2_01_01 := float64(s[0]*s[5]) - float64(s[1]*s[4])
	det2_01_02 := float64(s[0]*s[6]) - float64(s[2]*s[4])
	det2_01_03 := float64(s[0]*s[7]) - float64(s[3]*s[4])
	det2_01_12 := float64(s[1]*s[6]) - float64(s[2]*s[5])
	det2_01_13 := float64(s[1]*s[7]) - float64(s[3]*s[5])
	det2_01_23 := float64(s[2]*s[7]) - float64(s[3]*s[6])
	det3_201_012 := float64(s[8]*det2_01_12) - float64(s[9]*det2_01_02) + float64(s[10]*det2_01_01)
	det3_201_013 := float64(s[8]*det2_01_13) - float64(s[9]*det2_01_03) + float64(s[11]*det2_01_01)
	det3_201_023 := float64(s[8]*det2_01_23) - float64(s[10]*det2_01_03) + float64(s[11]*det2_01_02)
	det3_201_123 := float64(s[9]*det2_01_23) - float64(s[10]*det2_01_13) + float64(s[11]*det2_01_12)
	return float64(-det3_201_123*s[12]) + float64(det3_201_023*s[13]) - float64(det3_201_013*s[14]) +
		float64(det3_201_012*s[15])
}

// Invert inverts this and returns the determinant (0 when this is not
// invertible, in which case it is left unchanged).
func (m *Matrix4) Invert() float64 { return m.CopyInverse(m) }

// CopyInverse sets this matrix to the inverse of arg and returns arg's
// determinant; when it is 0, this is set to arg.
func (m *Matrix4) CopyInverse(arg *Matrix4) float64 {
	a := &arg.m4storage
	a00, a01, a02, a03 := a[0], a[1], a[2], a[3]
	a10, a11, a12, a13 := a[4], a[5], a[6], a[7]
	a20, a21, a22, a23 := a[8], a[9], a[10], a[11]
	a30, a31, a32, a33 := a[12], a[13], a[14], a[15]
	b00 := float64(a00*a11) - float64(a01*a10)
	b01 := float64(a00*a12) - float64(a02*a10)
	b02 := float64(a00*a13) - float64(a03*a10)
	b03 := float64(a01*a12) - float64(a02*a11)
	b04 := float64(a01*a13) - float64(a03*a11)
	b05 := float64(a02*a13) - float64(a03*a12)
	b06 := float64(a20*a31) - float64(a21*a30)
	b07 := float64(a20*a32) - float64(a22*a30)
	b08 := float64(a20*a33) - float64(a23*a30)
	b09 := float64(a21*a32) - float64(a22*a31)
	b10 := float64(a21*a33) - float64(a23*a31)
	b11 := float64(a22*a33) - float64(a23*a32)
	det := float64(b00*b11) - float64(b01*b10) + float64(b02*b09) + float64(b03*b08) - float64(b04*b07) +
		float64(b05*b06)
	if det == 0.0 {
		m.SetFrom(arg)
		return 0.0
	}
	invDet := 1.0 / det
	s := &m.m4storage
	s[0] = float64((float64(a11*b11) - float64(a12*b10) + float64(a13*b09)) * invDet)
	s[1] = float64((float64(-a01*b11) + float64(a02*b10) - float64(a03*b09)) * invDet)
	s[2] = float64((float64(a31*b05) - float64(a32*b04) + float64(a33*b03)) * invDet)
	s[3] = float64((float64(-a21*b05) + float64(a22*b04) - float64(a23*b03)) * invDet)
	s[4] = float64((float64(-a10*b11) + float64(a12*b08) - float64(a13*b07)) * invDet)
	s[5] = float64((float64(a00*b11) - float64(a02*b08) + float64(a03*b07)) * invDet)
	s[6] = float64((float64(-a30*b05) + float64(a32*b02) - float64(a33*b01)) * invDet)
	s[7] = float64((float64(a20*b05) - float64(a22*b02) + float64(a23*b01)) * invDet)
	s[8] = float64((float64(a10*b10) - float64(a11*b08) + float64(a13*b06)) * invDet)
	s[9] = float64((float64(-a00*b10) + float64(a01*b08) - float64(a03*b06)) * invDet)
	s[10] = float64((float64(a30*b04) - float64(a31*b02) + float64(a33*b00)) * invDet)
	s[11] = float64((float64(-a20*b04) + float64(a21*b02) - float64(a23*b00)) * invDet)
	s[12] = float64((float64(-a10*b09) + float64(a11*b07) - float64(a12*b06)) * invDet)
	s[13] = float64((float64(a00*b09) - float64(a01*b07) + float64(a02*b06)) * invDet)
	s[14] = float64((float64(-a30*b03) + float64(a31*b01) - float64(a32*b00)) * invDet)
	s[15] = float64((float64(a20*b03) - float64(a21*b01) + float64(a22*b00)) * invDet)
	return det
}

// Multiply multiplies this by arg (this = this * arg).
func (m *Matrix4) Multiply(arg *Matrix4) {
	s := &m.m4storage
	m00, m01, m02, m03 := s[0], s[4], s[8], s[12]
	m10, m11, m12, m13 := s[1], s[5], s[9], s[13]
	m20, m21, m22, m23 := s[2], s[6], s[10], s[14]
	m30, m31, m32, m33 := s[3], s[7], s[11], s[15]
	a := &arg.m4storage
	n00, n01, n02, n03 := a[0], a[4], a[8], a[12]
	n10, n11, n12, n13 := a[1], a[5], a[9], a[13]
	n20, n21, n22, n23 := a[2], a[6], a[10], a[14]
	n30, n31, n32, n33 := a[3], a[7], a[11], a[15]
	s[0] = float64(m00*n00) + float64(m01*n10) + float64(m02*n20) + float64(m03*n30)
	s[4] = float64(m00*n01) + float64(m01*n11) + float64(m02*n21) + float64(m03*n31)
	s[8] = float64(m00*n02) + float64(m01*n12) + float64(m02*n22) + float64(m03*n32)
	s[12] = float64(m00*n03) + float64(m01*n13) + float64(m02*n23) + float64(m03*n33)
	s[1] = float64(m10*n00) + float64(m11*n10) + float64(m12*n20) + float64(m13*n30)
	s[5] = float64(m10*n01) + float64(m11*n11) + float64(m12*n21) + float64(m13*n31)
	s[9] = float64(m10*n02) + float64(m11*n12) + float64(m12*n22) + float64(m13*n32)
	s[13] = float64(m10*n03) + float64(m11*n13) + float64(m12*n23) + float64(m13*n33)
	s[2] = float64(m20*n00) + float64(m21*n10) + float64(m22*n20) + float64(m23*n30)
	s[6] = float64(m20*n01) + float64(m21*n11) + float64(m22*n21) + float64(m23*n31)
	s[10] = float64(m20*n02) + float64(m21*n12) + float64(m22*n22) + float64(m23*n32)
	s[14] = float64(m20*n03) + float64(m21*n13) + float64(m22*n23) + float64(m23*n33)
	s[3] = float64(m30*n00) + float64(m31*n10) + float64(m32*n20) + float64(m33*n30)
	s[7] = float64(m30*n01) + float64(m31*n11) + float64(m32*n21) + float64(m33*n31)
	s[11] = float64(m30*n02) + float64(m31*n12) + float64(m32*n22) + float64(m33*n32)
	s[15] = float64(m30*n03) + float64(m31*n13) + float64(m32*n23) + float64(m33*n33)
}

// Multiplied is a copy of this multiplied by arg.
func (m *Matrix4) Multiplied(arg *Matrix4) *Matrix4 {
	r := m.Clone()
	r.Multiply(arg)
	return r
}

// Transform3 transforms arg (as a point) in place and returns it.
func (m *Matrix4) Transform3(arg *Vector3) *Vector3 {
	s := &m.m4storage
	a := &arg.v3storage
	x := float64(s[0]*a[0]) + float64(s[4]*a[1]) + float64(s[8]*a[2]) + s[12]
	y := float64(s[1]*a[0]) + float64(s[5]*a[1]) + float64(s[9]*a[2]) + s[13]
	z := float64(s[2]*a[0]) + float64(s[6]*a[1]) + float64(s[10]*a[2]) + s[14]
	a[0] = x
	a[1] = y
	a[2] = z
	return arg
}

// IsIdentity reports whether this is the identity matrix.
func (m *Matrix4) IsIdentity() bool {
	s := &m.m4storage
	return s[0] == 1.0 && s[1] == 0.0 && s[2] == 0.0 && s[3] == 0.0 &&
		s[4] == 0.0 && s[5] == 1.0 && s[6] == 0.0 && s[7] == 0.0 &&
		s[8] == 0.0 && s[9] == 0.0 && s[10] == 1.0 && s[11] == 0.0 &&
		s[12] == 0.0 && s[13] == 0.0 && s[14] == 0.0 && s[15] == 1.0
}

// IsZero reports whether this is the zero matrix.
func (m *Matrix4) IsZero() bool {
	for _, v := range m.m4storage {
		if v != 0.0 {
			return false
		}
	}
	return true
}

// vectorString is "[a,b,...]" with Dart's double.toString of each value.
func vectorString(values []float64) string {
	s := "["
	for i, v := range values {
		if i > 0 {
			s += ","
		}
		s += dart.DoubleToString(v)
	}
	return s + "]"
}
