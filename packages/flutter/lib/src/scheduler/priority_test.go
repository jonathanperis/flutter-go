// Port of packages/flutter/test/scheduler/priority_test.dart (Flutter
// 3.50.1): its one test.
package scheduler_test

import (
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
)

func TestPriorityTestDart(t *testing.T) {
	t.Run("Priority operators control test", func(t *testing.T) {
		priority := PriorityIdle.Add(PriorityKMaxOffset + 100)
		equal(t, priority.Value(), PriorityIdle.Value()+PriorityKMaxOffset)

		priority = PriorityAnimation.Sub(PriorityKMaxOffset + 100)
		equal(t, priority.Value(), PriorityAnimation.Value()-PriorityKMaxOffset)
	})

	// Go-only: offsets within the bound are kept, negative ones too.
	t.Run("offsets within kMaxOffset", func(t *testing.T) {
		equal(t, PriorityTouch.Add(PriorityKMaxOffset).Value(), 210000)
		equal(t, PriorityTouch.Add(-PriorityKMaxOffset-1).Value(), 190000)
		equal(t, PriorityTouch.Sub(-7).Value(), 200007)
		equal(t, PriorityIdle.Add(0).Value(), 0)
	})
}
