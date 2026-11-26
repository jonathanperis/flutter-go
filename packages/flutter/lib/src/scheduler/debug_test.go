// Port of packages/flutter/test/scheduler/debug_test.dart (Flutter 3.50.1):
// its one test.
package scheduler_test

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
)

func throwsFlutterError(f func()) (threw bool) {
	defer func() {
		if r := recover(); r != nil {
			_, threw = r.(*foundation.FlutterError)
		}
	}()
	f()
	return false
}

func TestDebugTestDart(t *testing.T) {
	t.Run("debugAssertAllSchedulerVarsUnset control test", func(t *testing.T) {
		equal(t, throwsFlutterError(func() { DebugAssertAllSchedulerVarsUnset("Example test") }), false)

		DebugPrintBeginFrameBanner = true

		equal(t, throwsFlutterError(func() { DebugAssertAllSchedulerVarsUnset("Example test") }), true)

		DebugPrintBeginFrameBanner = false
		DebugPrintEndFrameBanner = true

		equal(t, throwsFlutterError(func() { DebugAssertAllSchedulerVarsUnset("Example test") }), true)

		DebugPrintEndFrameBanner = false

		equal(t, throwsFlutterError(func() { DebugAssertAllSchedulerVarsUnset("Example test") }), false)
	})
}
