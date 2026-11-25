package ui

// Port of engine/src/flutter/testing/dart/lerp_test.dart (Flutter 3.50.1).
// Dart's int arguments are the same values as doubles in Go.

import (
	"math"
	"testing"
)

func p(v float64) *float64 { return &v }

func lerpIs(t *testing.T, got *float64, want float64, eps float64) {
	t.Helper()
	switch {
	case got == nil:
		t.Errorf("got nil, want %v", want)
	case math.IsNaN(want):
		if !math.IsNaN(*got) {
			t.Errorf("got %v, want NaN", *got)
		}
	case eps == 0 && *got != want, eps > 0 && math.Abs(*got-want) > eps:
		t.Errorf("got %v, want %v ± %v", *got, want, eps)
	}
}

const lerpTolerance = 1e-4

func TestLerpDoubleNullIffBothNull(t *testing.T) {
	if LerpDouble(nil, nil, 1) != nil {
		t.Error("lerp(null, null) != null")
	}
	if LerpDouble(p(5), nil, 0.25) == nil || LerpDouble(nil, p(5), 0.25) == nil {
		t.Error("one null gave null")
	}
}

func TestLerpDoubleNullIsZero(t *testing.T) {
	lerpIs(t, LerpDouble(nil, p(10), 0.25), 2.5, lerpTolerance)
	lerpIs(t, LerpDouble(p(10), nil, 0.25), 7.5, lerpTolerance)
}

func TestLerpDoubleValues(t *testing.T) {
	for _, c := range [][4]float64{
		{0, 10, -5, -50}, {10, 0, -5, 60}, {0, 10, 0.25, 2.5}, {10, 0, 0.25, 7.5},
		{10, 1e30, 1e-29, 20}, {10, 1e30, 0.5, 5e29}, {0, 10, 5, 50}, {10, 0, 5, -40},
	} {
		lerpIs(t, LerpDouble(p(c[0]), p(c[1]), c[2]), c[3], lerpTolerance)
	}
	// Exact at the ends.
	for _, c := range [][4]float64{
		{2, 10, 0, 2}, {10, 2, 0, 10}, {2, 10, 1, 10}, {10, 2, 1, 2}, {0, 10, 5, 50}, {10, 0, 5, -40},
		{1e30, 10, 1, 10}, {10, 1e30, 0, 10},
	} {
		lerpIs(t, LerpDouble(p(c[0]), p(c[1]), c[2]), c[3], 0)
	}
}

func TestLerpDoubleEqualEnds(t *testing.T) {
	inf, nan := math.Inf(1), math.NaN()
	for _, v := range []float64{10, nan, inf, -inf} {
		for _, tt := range []float64{5, nan, inf, -inf} {
			lerpIs(t, LerpDouble(p(v), p(v), tt), v, 0)
		}
	}
}

func TestLerpDoubleAssertions(t *testing.T) {
	inf, nan := math.Inf(1), math.NaN()
	for _, c := range [][3]float64{
		{0, 10, nan}, {0, 10, inf}, {0, 10, -inf},
		{nan, 10, 5}, {0, nan, 5}, {inf, 10, 5}, {-inf, 10, 5}, {0, inf, 5}, {0, -inf, 5},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("lerpDouble(%v, %v, %v): want panic", c[0], c[1], c[2])
				}
			}()
			LerpDouble(p(c[0]), p(c[1]), c[2])
		}()
	}
}
