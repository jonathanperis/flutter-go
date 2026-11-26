// Package painting is package:flutter/painting.dart.
//
// PARTIAL: only matrix_utils.dart's MatrixUtils methods semantics needs
// (getAsTranslation, getAsScale, matrixEquals, isIdentity, transformPoint),
// debugDescribeTransform and TransformProperty are ported so far.
package painting

import (
	"strings"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

type matrixUtils struct{}

// MatrixUtils is the class of static utility methods for Matrix4
// (PARTIAL, see the package comment): MatrixUtils.GetAsTranslation(m)...
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
