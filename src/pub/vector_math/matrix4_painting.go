package vector_math

// The Matrix4 members the painting library and its tests use
// (MatrixUtils.createCylindricalProjectionTransform, matrix_utils_test),
// ported from vector_math 2.4.4's matrix4.dart in its arithmetic order
// (no fused multiply-adds: each product is rounded, as in Dart).

import "github.com/jonathanperis/flutter-go/src/dart"

// NewMatrix4RotationX is Matrix4.rotationX: a rotation of radians around X.
func NewMatrix4RotationX(radians float64) *Matrix4 {
	m := NewMatrix4Zero()
	m.m4storage[15] = 1.0
	m.SetRotationX(radians)
	return m
}

// NewMatrix4RotationY is Matrix4.rotationY: a rotation of radians around Y.
func NewMatrix4RotationY(radians float64) *Matrix4 {
	m := NewMatrix4Zero()
	m.m4storage[15] = 1.0
	m.SetRotationY(radians)
	return m
}

// SetRotationX sets the upper 3x3 to a rotation of radians around X.
func (m *Matrix4) SetRotationX(radians float64) {
	c := dart.Cos(radians)
	s := dart.Sin(radians)
	m.m4storage[0] = 1.0
	m.m4storage[1] = 0.0
	m.m4storage[2] = 0.0
	m.m4storage[4] = 0.0
	m.m4storage[5] = c
	m.m4storage[6] = s
	m.m4storage[8] = 0.0
	m.m4storage[9] = -s
	m.m4storage[10] = c
	m.m4storage[3] = 0.0
	m.m4storage[7] = 0.0
	m.m4storage[11] = 0.0
}

// SetRotationY sets the upper 3x3 to a rotation of radians around Y.
func (m *Matrix4) SetRotationY(radians float64) {
	c := dart.Cos(radians)
	s := dart.Sin(radians)
	m.m4storage[0] = c
	m.m4storage[1] = 0.0
	m.m4storage[2] = -s
	m.m4storage[4] = 0.0
	m.m4storage[5] = 1.0
	m.m4storage[6] = 0.0
	m.m4storage[8] = s
	m.m4storage[9] = 0.0
	m.m4storage[10] = c
	m.m4storage[3] = 0.0
	m.m4storage[7] = 0.0
	m.m4storage[11] = 0.0
}

// rotate2 rotates the column pairs (a, b) of the storage: a' = a*c + b*s,
// b' = a*-s + b*c (rotateX/Y/Z's shared shape).
func (m *Matrix4) rotate2(a, b int, c, s float64) {
	st := &m.m4storage
	t1 := float64(st[a]*c) + float64(st[b]*s)
	t2 := float64(st[a+1]*c) + float64(st[b+1]*s)
	t3 := float64(st[a+2]*c) + float64(st[b+2]*s)
	t4 := float64(st[a+3]*c) + float64(st[b+3]*s)
	t5 := float64(st[a]*-s) + float64(st[b]*c)
	t6 := float64(st[a+1]*-s) + float64(st[b+1]*c)
	t7 := float64(st[a+2]*-s) + float64(st[b+2]*c)
	t8 := float64(st[a+3]*-s) + float64(st[b+3]*c)
	st[a], st[a+1], st[a+2], st[a+3] = t1, t2, t3, t4
	st[b], st[b+1], st[b+2], st[b+3] = t5, t6, t7, t8
}

// RotateX rotates this matrix angle radians around X.
func (m *Matrix4) RotateX(angle float64) {
	m.rotate2(4, 8, dart.Cos(angle), dart.Sin(angle))
}

// RotateY rotates this matrix angle radians around Y.
func (m *Matrix4) RotateY(angle float64) {
	cosAngle := dart.Cos(angle)
	sinAngle := dart.Sin(angle)
	st := &m.m4storage
	t1 := float64(st[0]*cosAngle) + float64(st[8]*-sinAngle)
	t2 := float64(st[1]*cosAngle) + float64(st[9]*-sinAngle)
	t3 := float64(st[2]*cosAngle) + float64(st[10]*-sinAngle)
	t4 := float64(st[3]*cosAngle) + float64(st[11]*-sinAngle)
	t5 := float64(st[0]*sinAngle) + float64(st[8]*cosAngle)
	t6 := float64(st[1]*sinAngle) + float64(st[9]*cosAngle)
	t7 := float64(st[2]*sinAngle) + float64(st[10]*cosAngle)
	t8 := float64(st[3]*sinAngle) + float64(st[11]*cosAngle)
	st[0], st[1], st[2], st[3] = t1, t2, t3, t4
	st[8], st[9], st[10], st[11] = t5, t6, t7, t8
}

// RotateZ rotates this matrix angle radians around Z.
func (m *Matrix4) RotateZ(angle float64) {
	m.rotate2(0, 4, dart.Cos(angle), dart.Sin(angle))
}

// GetTranslation returns the translation vector of this homogeneous
// transformation matrix.
func (m *Matrix4) GetTranslation() *Vector3 {
	return NewVector3(m.m4storage[12], m.m4storage[13], m.m4storage[14])
}

// Mul is operator * with a Matrix4 argument: the product this * arg
// (multiplied). Dart's operator also takes a double, Vector3 or Vector4:
// use ScaledByDouble-style methods for those.
func (m *Matrix4) Mul(arg *Matrix4) *Matrix4 { return m.Multiplied(arg) }

// Translate is the deprecated translate(x, [y, z]) with double arguments
// (translateByDouble(x, y, z, 1)); y and z default to 0.
func (m *Matrix4) Translate(x float64, yz ...float64) {
	y, z := 0.0, 0.0
	if len(yz) > 0 {
		y = yz[0]
	}
	if len(yz) > 1 {
		z = yz[1]
	}
	m.TranslateByDouble(x, y, z, 1.0)
}

// Scale is the deprecated scale(x, [y, z]) with double arguments
// (scaleByDouble(x, y ?? x, z ?? x, 1)).
func (m *Matrix4) Scale(x float64, yz ...float64) {
	y, z := x, x
	if len(yz) > 0 {
		y = yz[0]
	}
	if len(yz) > 1 {
		z = yz[1]
	}
	m.ScaleByDouble(x, y, z, 1.0)
}
