package animation

import (
	"math"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/physics"
)

// AnimateWith drives the controller with a simulation (Flutter's
// animateWith): each tick sets value = sim.X(elapsed seconds).
func (c *Controller) AnimateWith(sim physics.Simulation) {
	c.stopTicker(true)
	c.sim = sim
	c.setStatus(Forward)
	c.value = math.Max(c.lower, math.Min(c.upper, sim.X(0)))
	c.ticker.Start()
	c.notify()
}

// Stop halts the controller where it is.
func (c *Controller) Stop() { c.stopTicker(true); c.sim = nil }
