// Package physics is Flutter's physics library (package:flutter/physics.dart):
// simple one-dimensional simulations used by animations and scrolling.
//
// Ported from packages/flutter/lib/src/physics at the pinned Flutter commit.
package physics

import "fmt"

const epsilonDefault = 1e-3

// Tolerance describes the precision a simulation needs: differences smaller
// than these are treated as equal.
type Tolerance struct {
	Distance float64 // in logical pixels
	Time     float64 // in seconds
	Velocity float64 // in logical pixels per second
}

// DefaultTolerance is Tolerance.defaultTolerance: 1e-3 for every field.
var DefaultTolerance = Tolerance{Distance: epsilonDefault, Time: epsilonDefault, Velocity: epsilonDefault}

// ToleranceOptions are the optional arguments of Flutter's Tolerance
// constructor; nil fields take the default of 1e-3.
type ToleranceOptions struct {
	Distance, Time, Velocity *float64
}

// NewTolerance is Flutter's Tolerance({distance, time, velocity}).
func NewTolerance(opts ...ToleranceOptions) Tolerance {
	t := DefaultTolerance
	for _, o := range opts {
		if o.Distance != nil {
			t.Distance = *o.Distance
		}
		if o.Time != nil {
			t.Time = *o.Time
		}
		if o.Velocity != nil {
			t.Velocity = *o.Velocity
		}
	}
	return t
}

func (t Tolerance) String() string {
	return fmt.Sprintf("Tolerance(distance: ±%s, time: ±%s, velocity: ±%s)",
		dartDouble(t.Distance), dartDouble(t.Time), dartDouble(t.Velocity))
}
