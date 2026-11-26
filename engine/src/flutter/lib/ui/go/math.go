// Package ui is Flutter's dart:ui: the engine's API (geometry, painting,
// text, compositing, platform dispatch), implemented in Go over the
// Flutter engine's Impeller SDK.
//
// Ported from engine/src/flutter/lib/ui (Flutter 3.50.1).
package ui

import (
	"math"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// ClampDouble is dart:ui's clampDouble: x limited to [min, max], with NaN
// mapped to max. Unlike math.Max/math.Min it never returns NaN.
func ClampDouble(x, min, max float64) float64 {
	// assert(min <= max && !max.isNaN && !min.isNaN), always on in Go.
	if !(min <= max) || math.IsNaN(max) || math.IsNaN(min) {
		panic(dart.NewAssertionError("min <= max && !max.isNaN && !min.isNaN"))
	}
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	if math.IsNaN(x) {
		return max
	}
	return x
}
