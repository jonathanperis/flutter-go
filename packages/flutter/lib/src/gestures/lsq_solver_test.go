// Port of packages/flutter/test/gestures/lsq_solver_test.dart (Flutter
// 3.50.1): all 5 tests. No skips.
package gestures_test

import (
	"math"
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
)

func approx(value, expectation float64) bool { return math.Abs(value-expectation) < 1e-6 }

func ones(n int) []float64 {
	r := make([]float64, n)
	for i := range r {
		r[i] = 1.0
	}
	return r
}

func TestLeastSquaresFit(t *testing.T) {
	x := []float64{0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
	flat := ones(9)
	sloped := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0}
	cases := []struct {
		name   string
		y      []float64
		degree int
		want   []float64
	}{
		{"linear polynomial to line", flat, 1, []float64{1, 0}},
		{"linear polynomial to sloped line", sloped, 1, []float64{1, 1}},
		{"quadratic polynomial to line", flat, 2, []float64{1, 0, 0}},
		{"quadratic polynomial to sloped line", sloped, 2, []float64{1, 1, 0}},
	}
	for _, c := range cases {
		fit := NewLeastSquaresSolver(x, c.y, ones(9)).Solve(c.degree)
		if fit == nil || len(fit.Coefficients) != len(c.want) {
			t.Fatalf("%s: fit %v", c.name, fit)
		}
		for i, w := range c.want {
			if !approx(fit.Coefficients[i], w) {
				t.Errorf("%s: coefficient %d = %v, want %v", c.name, i, fit.Coefficients[i], w)
			}
		}
		if !approx(fit.Confidence, 1.0) {
			t.Errorf("%s: confidence %v", c.name, fit.Confidence)
		}
	}
}

func TestLeastSquaresFitToString(t *testing.T) {
	fit := NewPolynomialFit(2)
	fit.Coefficients[0] = 123.45
	fit.Coefficients[1] = 54.321
	fit.Coefficients[2] = 1.3579
	fit.Confidence = 0.9876
	if got := fit.String(); got != "PolynomialFit([123, 54.3, 1.36], confidence: 0.988)" {
		t.Errorf("got %q", got)
	}
}

func TestLeastSquaresNotEnoughData(t *testing.T) {
	if NewLeastSquaresSolver([]float64{1}, []float64{1}, []float64{1}).Solve(2) != nil {
		t.Error("degree > len(x) must give nil")
	}
	// Linearly dependent columns (all x equal): no solution.
	if NewLeastSquaresSolver([]float64{2, 2, 2}, []float64{1, 2, 3}, ones(3)).Solve(1) != nil {
		t.Error("degenerate data must give nil")
	}
}
