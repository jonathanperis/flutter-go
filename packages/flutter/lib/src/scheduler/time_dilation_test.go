// Port of packages/flutter/test/scheduler/time_dilation_test.dart (Flutter
// 3.50.1): its one test, in a process without a binding (as the Dart file
// runs).
package scheduler_test

import (
	"testing"

	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
)

func TestTimeDilationTestDart(t *testing.T) {
	if !isolated(t) {
		return
	}
	t.Run("timeDilation can be set without a binding", func(t *testing.T) {
		equal(t, TimeDilation(), 1.0)
		SetTimeDilation(2.0)
		equal(t, TimeDilation(), 2.0)
	})
}
