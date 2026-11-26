package gestures

// Port of lsq_solver.dart (Flutter 3.50.1), in Dart's arithmetic order.

import (
	"math"
	"strings"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// lsqVector is _Vector: a view of length elements of a slice from offset.
type lsqVector struct {
	offset   int
	length   int
	elements []float64
}

func newLsqVector(size int) lsqVector {
	return lsqVector{length: size, elements: make([]float64, size)}
}

func (v lsqVector) at(i int) float64     { return v.elements[i+v.offset] }
func (v lsqVector) set(i int, x float64) { v.elements[i+v.offset] = x }

// dot is operator *.
func (v lsqVector) dot(a lsqVector) float64 {
	result := 0.0
	for i := 0; i < v.length; i++ {
		result += float64(v.at(i) * a.at(i))
	}
	return result
}

func (v lsqVector) norm() float64 { return math.Sqrt(v.dot(v)) }

// lsqMatrix is _Matrix, row-major.
type lsqMatrix struct {
	columns  int
	elements []float64
}

func newLsqMatrix(rows, cols int) lsqMatrix {
	return lsqMatrix{columns: cols, elements: make([]float64, rows*cols)}
}

func (m lsqMatrix) get(row, col int) float64    { return m.elements[row*m.columns+col] }
func (m lsqMatrix) set(row, col int, x float64) { m.elements[row*m.columns+col] = x }
func (m lsqMatrix) getRow(row int) lsqVector {
	return lsqVector{offset: row * m.columns, length: m.columns, elements: m.elements}
}

// PolynomialFit is an nth degree polynomial fit to a dataset.
type PolynomialFit struct {
	// Coefficients are the polynomial coefficients of the fit, constant
	// term first (Coefficients[i] multiplies x^i).
	Coefficients []float64
	// Confidence is an indicator of the quality of the fit: 1 is a perfect
	// fit, smaller is worse (Dart's late field, set by the solver).
	Confidence float64
}

// NewPolynomialFit creates a polynomial fit of the given degree (all
// coefficients zero).
func NewPolynomialFit(degree int) *PolynomialFit {
	return &PolynomialFit{Coefficients: make([]float64, degree+1)}
}

func (f *PolynomialFit) String() string {
	parts := make([]string, len(f.Coefficients))
	for i, c := range f.Coefficients {
		parts[i] = dart.ToStringAsPrecision(c, 3)
	}
	return foundation.ObjectRuntimeType(f, "PolynomialFit") + "([" + strings.Join(parts, ", ") +
		"], confidence: " + dart.ToStringAsFixed(f.Confidence, 3) + ")"
}

// LeastSquaresSolver uses the least-squares algorithm to fit a polynomial
// to a set of data (x, y, w: the weights).
type LeastSquaresSolver struct {
	X []float64
	Y []float64
	W []float64
}

// NewLeastSquaresSolver creates a solver for the data; the three lists
// must have the same length.
func NewLeastSquaresSolver(x, y, w []float64) *LeastSquaresSolver {
	if len(x) != len(y) || len(y) != len(w) {
		panic(dart.NewAssertionError(nil))
	}
	return &LeastSquaresSolver{X: x, Y: y, W: w}
}

// Solve fits a polynomial of the given degree to the data; it returns nil
// if there is not enough data or the data are degenerate.
func (s *LeastSquaresSolver) Solve(degree int) *PolynomialFit {
	x, y, w := s.X, s.Y, s.W
	if degree > len(x) {
		// Not enough data to fit a curve.
		return nil
	}

	result := NewPolynomialFit(degree)

	// Shorthands for the purpose of notation equivalence to original C++ code.
	m := len(x)
	n := degree + 1

	// Expand the X vector to a matrix A, pre-multiplied by the weights.
	a := newLsqMatrix(n, m)
	for h := 0; h < m; h++ {
		a.set(0, h, w[h])
		for i := 1; i < n; i++ {
			a.set(i, h, a.get(i-1, h)*x[h])
		}
	}

	// Apply the Gram-Schmidt process to A to obtain its QR decomposition.

	// Orthonormal basis, column-major order.
	q := newLsqMatrix(n, m)
	// Upper triangular matrix, row-major order.
	r := newLsqMatrix(n, n)
	for j := 0; j < n; j++ {
		for h := 0; h < m; h++ {
			q.set(j, h, a.get(j, h))
		}
		for i := 0; i < j; i++ {
			dot := q.getRow(j).dot(q.getRow(i))
			for h := 0; h < m; h++ {
				q.set(j, h, q.get(j, h)-float64(dot*q.get(i, h)))
			}
		}

		norm := q.getRow(j).norm()
		if norm < foundation.PrecisionErrorTolerance {
			// Vectors are linearly dependent or zero so no solution.
			return nil
		}

		inverseNorm := 1.0 / norm
		for h := 0; h < m; h++ {
			q.set(j, h, q.get(j, h)*inverseNorm)
		}
		for i := 0; i < n; i++ {
			if i < j {
				r.set(j, i, 0.0)
			} else {
				r.set(j, i, q.getRow(j).dot(a.getRow(i)))
			}
		}
	}

	// Solve R B = Qt W Y to find B. This is easy because R is upper
	// triangular. We just work from bottom-right to top-left calculating B's
	// coefficients.
	wy := newLsqVector(m)
	for h := 0; h < m; h++ {
		wy.set(h, y[h]*w[h])
	}
	for i := n - 1; i >= 0; i-- {
		result.Coefficients[i] = q.getRow(i).dot(wy)
		for j := n - 1; j > i; j-- {
			result.Coefficients[i] -= float64(r.get(i, j) * result.Coefficients[j])
		}
		result.Coefficients[i] /= r.get(i, i)
	}

	// Calculate the coefficient of determination (confidence) as:
	//   1 - (sumSquaredError / sumSquaredTotal)
	// ...where sumSquaredError is the residual sum of squares (variance of
	// the error), and sumSquaredTotal is the total sum of squares (variance
	// of the data) where each has been weighted.
	yMean := 0.0
	for h := 0; h < m; h++ {
		yMean += y[h]
	}
	yMean /= float64(m)

	sumSquaredError := 0.0
	sumSquaredTotal := 0.0
	for h := 0; h < m; h++ {
		term := 1.0
		err := y[h] - result.Coefficients[0]
		for i := 1; i < n; i++ {
			term = float64(term * x[h])
			err -= float64(term * result.Coefficients[i])
		}
		sumSquaredError += float64(float64(float64(w[h]*w[h])*err) * err)
		v := y[h] - yMean
		sumSquaredTotal += float64(float64(float64(w[h]*w[h])*v) * v)
	}

	if sumSquaredTotal <= foundation.PrecisionErrorTolerance {
		result.Confidence = 1.0
	} else {
		result.Confidence = 1.0 - (sumSquaredError / sumSquaredTotal)
	}
	return result
}
