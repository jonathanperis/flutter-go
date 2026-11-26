// Port of packages/flutter/test/scheduler/animation_test.dart (Flutter
// 3.50.1): both tests. TestSchedulerBinding mixes in only SchedulerBinding
// (ServicesBinding is not ported; the tests don't use it).
package scheduler_test

import (
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
)

func TestAnimationTestDart(t *testing.T) {
	if !isolated(t) {
		return
	}
	var scheduler SchedulerBinding = newPlainBinding()

	t.Run("Check for a time dilation being in effect", func(t *testing.T) {
		equal(t, TimeDilation(), 1.0)
	})

	t.Run("Can cancel queued callback", func(t *testing.T) {
		var secondID int

		firstCallbackRan := false
		secondCallbackRan := false

		firstCallback := func(timeStamp time.Duration) {
			equal(t, firstCallbackRan, false)
			equal(t, secondCallbackRan, false)
			equal(t, timeStamp.Milliseconds(), int64(0))
			firstCallbackRan = true
			scheduler.CancelFrameCallbackWithId(secondID)
		}

		secondCallback := func(timeStamp time.Duration) {
			equal(t, firstCallbackRan, true)
			equal(t, secondCallbackRan, false)
			equal(t, timeStamp.Milliseconds(), int64(0))
			secondCallbackRan = true
		}

		scheduler.ScheduleFrameCallback(firstCallback)
		secondID = scheduler.ScheduleFrameCallback(secondCallback)

		tick(16 * time.Millisecond)

		equal(t, firstCallbackRan, true)
		equal(t, secondCallbackRan, false)

		firstCallbackRan = false
		secondCallbackRan = false

		tick(32 * time.Millisecond)

		equal(t, firstCallbackRan, false)
		equal(t, secondCallbackRan, false)
	})
}
