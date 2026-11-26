// Package painting is package:flutter/painting.dart.
package painting

// Port of matrix_utils.dart (Flutter 3.50.1): MatrixUtils,
// debugDescribeTransform and TransformProperty.

import (
	"math"
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

type matrixUtils struct{}

// MatrixUtils is the class of static utility methods for Matrix4:
// MatrixUtils.GetAsTranslation(m)...
var MatrixUtils matrixUtils

// GetAsTranslation returns the given transform's offset if it is a pure 2D
// translation, or nil.
func (matrixUtils) GetAsTranslation(transform *vm.Matrix4) *ui.Offset {
	s := transform.Storage()
	// Values are stored in column-major order, so the appearance of dx is
	// transposed from the top-right to the bottom-left.
	if s[0] == 1.0 && s[1] == 0.0 && s[2] == 0.0 && s[3] == 0.0 &&
		s[4] == 0.0 && s[5] == 1.0 && s[6] == 0.0 && s[7] == 0.0 &&
		s[8] == 0.0 && s[9] == 0.0 && s[10] == 1.0 && s[11] == 0.0 &&
		s[14] == 0.0 && s[15] == 1.0 {
		o := ui.NewOffset(s[12], s[13])
		return &o
	}
	return nil
}

// GetAsScale returns the given transform's scale if it is a uniform 2D
// scale, or nil.
func (matrixUtils) GetAsScale(transform *vm.Matrix4) *float64 {
	s := transform.Storage()
	// Values are stored in column-major order (but this symmetric matrix is
	// unaffected).
	if s[1] == 0.0 && s[2] == 0.0 && s[3] == 0.0 &&
		s[4] == 0.0 && s[6] == 0.0 && s[7] == 0.0 &&
		s[8] == 0.0 && s[9] == 0.0 && s[10] == 1.0 && s[11] == 0.0 &&
		s[12] == 0.0 && s[13] == 0.0 && s[14] == 0.0 && s[15] == 1.0 &&
		s[0] == s[5] {
		d := s[0]
		return &d
	}
	return nil
}

// MatrixEquals reports whether the two matrices are equal (nil is the
// identity; at most one may be nil, as Dart asserts).
func (m matrixUtils) MatrixEquals(a, b *vm.Matrix4) bool {
	if a == b {
		return true
	}
	if a == nil {
		return m.IsIdentity(b)
	}
	if b == nil {
		return m.IsIdentity(a)
	}
	sa, sb := a.Storage(), b.Storage()
	for i := 0; i < 16; i++ {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

// IsIdentity reports whether the given matrix is the identity matrix.
func (matrixUtils) IsIdentity(a *vm.Matrix4) bool {
	s := a.Storage()
	return s[0] == 1.0 && s[1] == 0.0 && s[2] == 0.0 && s[3] == 0.0 &&
		s[4] == 0.0 && s[5] == 1.0 && s[6] == 0.0 && s[7] == 0.0 &&
		s[8] == 0.0 && s[9] == 0.0 && s[10] == 1.0 && s[11] == 0.0 &&
		s[12] == 0.0 && s[13] == 0.0 && s[14] == 0.0 && s[15] == 1.0
}

// TransformPoint applies the given matrix as a perspective transform to
// the given point (z = 0, the resulting z dropped).
func (matrixUtils) TransformPoint(transform *vm.Matrix4, point ui.Offset) ui.Offset {
	s := transform.Storage()
	x := point.Dx
	y := point.Dy
	// Directly simulate the transform of the vector (x, y, 0, 1), dropping
	// the resulting Z coordinate, and normalizing only if needed.
	rx := float64(s[0]*x) + float64(s[4]*y) + s[12]
	ry := float64(s[1]*x) + float64(s[5]*y) + s[13]
	rw := float64(s[3]*x) + float64(s[7]*y) + s[15]
	if rw == 1.0 {
		return ui.NewOffset(rx, ry)
	}
	return ui.NewOffset(rx/rw, ry/rw)
}

// MultiplyInPlace computes a * b and stores the result in b (a is left
// unchanged).
func (matrixUtils) MultiplyInPlace(a, b *vm.Matrix4) {
	as := a.Storage()
	m00, m01, m02, m03 := as[0], as[4], as[8], as[12]
	m10, m11, m12, m13 := as[1], as[5], as[9], as[13]
	m20, m21, m22, m23 := as[2], as[6], as[10], as[14]
	m30, m31, m32, m33 := as[3], as[7], as[11], as[15]
	bs := b.Storage()
	n00, n01, n02, n03 := bs[0], bs[4], bs[8], bs[12]
	n10, n11, n12, n13 := bs[1], bs[5], bs[9], bs[13]
	n20, n21, n22, n23 := bs[2], bs[6], bs[10], bs[14]
	n30, n31, n32, n33 := bs[3], bs[7], bs[11], bs[15]
	bs[0] = float64(m00*n00) + float64(m01*n10) + float64(m02*n20) + float64(m03*n30)
	bs[4] = float64(m00*n01) + float64(m01*n11) + float64(m02*n21) + float64(m03*n31)
	bs[8] = float64(m00*n02) + float64(m01*n12) + float64(m02*n22) + float64(m03*n32)
	bs[12] = float64(m00*n03) + float64(m01*n13) + float64(m02*n23) + float64(m03*n33)
	bs[1] = float64(m10*n00) + float64(m11*n10) + float64(m12*n20) + float64(m13*n30)
	bs[5] = float64(m10*n01) + float64(m11*n11) + float64(m12*n21) + float64(m13*n31)
	bs[9] = float64(m10*n02) + float64(m11*n12) + float64(m12*n22) + float64(m13*n32)
	bs[13] = float64(m10*n03) + float64(m11*n13) + float64(m12*n23) + float64(m13*n33)
	bs[2] = float64(m20*n00) + float64(m21*n10) + float64(m22*n20) + float64(m23*n30)
	bs[6] = float64(m20*n01) + float64(m21*n11) + float64(m22*n21) + float64(m23*n31)
	bs[10] = float64(m20*n02) + float64(m21*n12) + float64(m22*n22) + float64(m23*n32)
	bs[14] = float64(m20*n03) + float64(m21*n13) + float64(m22*n23) + float64(m23*n33)
	bs[3] = float64(m30*n00) + float64(m31*n10) + float64(m32*n20) + float64(m33*n30)
	bs[7] = float64(m30*n01) + float64(m31*n11) + float64(m32*n21) + float64(m33*n31)
	bs[11] = float64(m30*n02) + float64(m31*n12) + float64(m32*n22) + float64(m33*n32)
	bs[15] = float64(m30*n03) + float64(m31*n13) + float64(m32*n23) + float64(m33*n33)
}

// safeTransformRect transforms the four corners one by one (used when the
// rect's width or height is not finite).
func safeTransformRect(transform *vm.Matrix4, rect ui.Rect) ui.Rect {
	storage := transform.Storage()
	isAffine := storage[3] == 0.0 && storage[7] == 0.0 && storage[15] == 1.0
	var minMax [4]float64
	accumulate(&minMax, storage, rect.Left, rect.Top, true, isAffine)
	accumulate(&minMax, storage, rect.Right, rect.Top, false, isAffine)
	accumulate(&minMax, storage, rect.Left, rect.Bottom, false, isAffine)
	accumulate(&minMax, storage, rect.Right, rect.Bottom, false, isAffine)
	return ui.NewRectFromLTRB(minMax[0], minMax[1], minMax[2], minMax[3])
}

func accumulate(minMax *[4]float64, m []float64, x, y float64, first, isAffine bool) {
	w := 1.0
	if !isAffine {
		w = 1.0 / (float64(m[3]*x) + float64(m[7]*y) + m[15])
	}
	tx := float64((float64(m[0]*x) + float64(m[4]*y) + m[12]) * w)
	ty := float64((float64(m[1]*x) + float64(m[5]*y) + m[13]) * w)
	if first {
		minMax[0], minMax[2] = tx, tx
		minMax[1], minMax[3] = ty, ty
		return
	}
	if tx < minMax[0] {
		minMax[0] = tx
	}
	if ty < minMax[1] {
		minMax[1] = ty
	}
	if tx > minMax[2] {
		minMax[2] = tx
	}
	if ty > minMax[3] {
		minMax[3] = ty
	}
}

// TransformRect returns a rect that bounds the result of applying the
// given matrix as a perspective transform to the given rect.
func (matrixUtils) TransformRect(transform *vm.Matrix4, rect ui.Rect) ui.Rect {
	storage := transform.Storage()
	x := rect.Left
	y := rect.Top
	w := rect.Right - x
	h := rect.Bottom - y
	// We want to avoid turning a finite rect into an infinite one if we can.
	if math.IsInf(w, 0) || math.IsNaN(w) || math.IsInf(h, 0) || math.IsNaN(h) {
		return safeTransformRect(transform, rect)
	}
	// See matrix_utils.dart for the derivation: the corners are the
	// transformed top-left corner plus the delta-transformed width and
	// height vectors.
	wx := float64(storage[0] * w)
	hx := float64(storage[4] * h)
	rx := float64(storage[0]*x) + float64(storage[4]*y) + storage[12]
	wy := float64(storage[1] * w)
	hy := float64(storage[5] * h)
	ry := float64(storage[1]*x) + float64(storage[5]*y) + storage[13]
	if storage[3] == 0.0 && storage[7] == 0.0 && storage[15] == 1.0 {
		left, right := rx, rx
		if wx < 0 {
			left += wx
		} else {
			right += wx
		}
		if hx < 0 {
			left += hx
		} else {
			right += hx
		}
		top, bottom := ry, ry
		if wy < 0 {
			top += wy
		} else {
			bottom += wy
		}
		if hy < 0 {
			top += hy
		} else {
			bottom += hy
		}
		return ui.NewRectFromLTRB(left, top, right, bottom)
	}
	ww := float64(storage[3] * w)
	hw := float64(storage[7] * h)
	rw := float64(storage[3]*x) + float64(storage[7]*y) + storage[15]
	ulx := rx / rw
	uly := ry / rw
	urx := (rx + wx) / (rw + ww)
	ury := (ry + wy) / (rw + ww)
	llx := (rx + hx) / (rw + hw)
	lly := (ry + hy) / (rw + hw)
	lrx := (rx + wx + hx) / (rw + ww + hw)
	lry := (ry + wy + hy) / (rw + ww + hw)
	return ui.NewRectFromLTRB(min4(ulx, urx, llx, lrx), min4(uly, ury, lly, lry),
		max4(ulx, urx, llx, lrx), max4(uly, ury, lly, lry))
}

func min4(a, b, c, d float64) float64 {
	e := b
	if a < b {
		e = a
	}
	f := d
	if c < d {
		f = c
	}
	if e < f {
		return e
	}
	return f
}

func max4(a, b, c, d float64) float64 {
	e := b
	if a > b {
		e = a
	}
	f := d
	if c > d {
		f = c
	}
	if e > f {
		return e
	}
	return f
}

// InverseTransformRect returns a rect that bounds the result of applying
// the inverse of the given matrix as a perspective transform to the given
// rect (the rect itself for the identity).
func (m matrixUtils) InverseTransformRect(transform *vm.Matrix4, rect ui.Rect) ui.Rect {
	if m.IsIdentity(transform) {
		return rect
	}
	transform = vm.NewMatrix4Copy(transform)
	transform.Invert()
	return m.TransformRect(transform, rect)
}

// CreateCylindricalProjectionTransformOptions are
// createCylindricalProjectionTransform's optional named parameters (nil:
// Dart's defaults, perspective 0.001 and Axis.vertical).
type CreateCylindricalProjectionTransformOptions struct {
	Perspective *float64
	Orientation *Axis
}

// CreateCylindricalProjectionTransform returns a matrix that transforms
// every point to a cylindrical projection: radius and angle (Dart's
// required named parameters) are positional.
func (matrixUtils) CreateCylindricalProjectionTransform(radius, angle float64,
	opts ...CreateCylindricalProjectionTransformOptions) *vm.Matrix4 {
	perspective, orientation := 0.001, AxisVertical
	if len(opts) > 0 {
		if opts[0].Perspective != nil {
			perspective = *opts[0].Perspective
		}
		if opts[0].Orientation != nil {
			orientation = *opts[0].Orientation
		}
	}
	if !(perspective >= 0 && perspective <= 1.0) {
		panic(dart.NewAssertionError("perspective >= 0 && perspective <= 1.0"))
	}
	// Pre-multiplied matrix of a projection matrix and a view matrix (see
	// matrix_utils.dart).
	result := vm.NewMatrix4Identity()
	result.SetEntry(3, 2, -perspective)
	result.SetEntry(2, 3, -radius)
	result.SetEntry(3, 3, float64(perspective*radius)+1.0)
	// Model matrix by first translating the object from the origin of the
	// world by radius in the z axis and then rotating against the world.
	var rotation *vm.Matrix4
	if orientation == AxisHorizontal {
		rotation = vm.NewMatrix4RotationY(angle)
	} else {
		rotation = vm.NewMatrix4RotationX(angle)
	}
	// Essentially perspective * view * model.
	return result.Mul(rotation.Mul(vm.NewMatrix4TranslationValues(0.0, 0.0, radius)))
}

// ForceToPoint returns a matrix that transforms every point to offset.
func (matrixUtils) ForceToPoint(offset ui.Offset) *vm.Matrix4 {
	result := vm.NewMatrix4Zero()
	storage := result.Storage()
	// Matrix4 is column-major. The first two columns (indices 0, 1, 4, 5)
	// are zero, so x and y are ignored.
	storage[10] = 1 // z' = z
	storage[12] = offset.Dx
	storage[13] = offset.Dy
	storage[15] = 1 // w' = w
	return result
}

func formatEntry(m *vm.Matrix4, row, col int) string {
	v := m.Entry(row, col)
	return foundation.DebugFormatDouble(&v)
}

func describeRow(m *vm.Matrix4, row int) string {
	return formatEntry(m, row, 0) + "," + formatEntry(m, row, 1) + "," + formatEntry(m, row, 2) + "," +
		formatEntry(m, row, 3)
}

// DebugDescribeTransform returns a list of strings representing the given
// transform in a format useful for TransformProperty (["null"] for nil).
func DebugDescribeTransform(transform *vm.Matrix4) []string {
	if transform == nil {
		return []string{"null"}
	}
	return []string{
		"[0] " + describeRow(transform, 0),
		"[1] " + describeRow(transform, 1),
		"[2] " + describeRow(transform, 2),
		"[3] " + describeRow(transform, 3),
	}
}

// TransformProperty is a property describing a Matrix4 (nil: null).
type TransformProperty struct {
	foundation.DiagnosticsProperty[*vm.Matrix4]
}

// TransformPropertyOptions are TransformProperty's named parameters.
type TransformPropertyOptions struct {
	ShowName     *bool
	DefaultValue any
	Level        *foundation.DiagnosticLevel
}

// NewTransformProperty is TransformProperty(name, value, {showName,
// defaultValue, level}).
func NewTransformProperty(name string, value *vm.Matrix4, opts ...TransformPropertyOptions) *TransformProperty {
	var o TransformPropertyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	p := &TransformProperty{DiagnosticsProperty: *foundation.NewDiagnosticsProperty(name, value,
		foundation.DiagnosticsPropertyOptions{ShowName: o.ShowName, DefaultValue: o.DefaultValue, Level: o.Level})}
	p.SetSelf(p)
	return p
}

// ValueToString is the four rows, on one line when the parent's style
// puts properties on one line.
func (p *TransformProperty) ValueToString(opts ...foundation.ValueToStringOptions) string {
	value := p.TypedValue()
	if len(opts) > 0 && opts[0].ParentConfiguration != nil && !opts[0].ParentConfiguration.LineBreakProperties {
		// Format the value on a single line to be compatible with the parent's
		// style.
		values := []string{describeRow(value, 0), describeRow(value, 1), describeRow(value, 2), describeRow(value, 3)}
		return "[" + strings.Join(values, "; ") + "]"
	}
	return strings.Join(DebugDescribeTransform(value), "\n")
}
