package ui

// Port of engine/src/flutter/lib/ui/lerp.dart (Flutter 3.50.1).

import "math"

// LerpDouble is dart:ui's lerpDouble: linear interpolation between a and b,
// where nil stands for Dart's null (treated as 0 unless both are nil). Like
// Dart in debug mode, it panics on non-finite inputs.
func LerpDouble(a, b *float64, t float64) *float64 {
	if a != nil && b != nil && (*a == *b || math.IsNaN(*a) && math.IsNaN(*b)) {
		v := *a
		return &v
	}
	if a == nil && b == nil {
		return nil
	}
	av, bv := 0.0, 0.0
	if a != nil {
		av = *a
	}
	if b != nil {
		bv = *b
	}
	if !isFinite(av) || !isFinite(bv) {
		panic("ui: Cannot interpolate between finite and non-finite values")
	}
	if !isFinite(t) {
		panic("ui: t must be finite when interpolating between values")
	}
	r := av*(1.0-t) + bv*t
	return &r
}

func lerpDouble(a, b, t float64) float64 { return a*(1.0-t) + b*t }
