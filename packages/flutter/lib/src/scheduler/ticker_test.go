// Port of packages/flutter/test/scheduler/ticker_test.dart (Flutter
// 3.50.1): all 7 tests, in flutter_test's binding (testWidgets is
// flutter_test.New + Tester.Pump).
//
// Adaptations: setAppLifeCycleState sends the state through the
// flutter/lifecycle channel in Dart; services is not ported, so the helper
// does what ServicesBinding does with the message: it calls
// handleAppLifecycleStateChanged for each generated state transition
// (_generateStateTransitions, ported below). "Ticker control test" checks
// that the creation stack names the creating function, as Dart's
// `contains('testFunction')`. memoryEvents/areCreateAndDispose
// (leak_tracker) are reproduced with FlutterMemoryAllocations' listener.
package scheduler_test

import (
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// generateStateTransitions is ServicesBinding._generateStateTransitions.
func generateStateTransitions(previousState *ui.AppLifecycleState, state ui.AppLifecycleState) []ui.AppLifecycleState {
	if previousState != nil && *previousState == state {
		return nil
	}
	var stateChanges []ui.AppLifecycleState
	values := ui.AppLifecycleStateValues
	if previousState == nil {
		// If there was no previous state, just jump directly to the new state.
		stateChanges = append(stateChanges, state)
	} else {
		previousStateIndex, stateIndex := int(*previousState), int(state)
		if state == ui.AppLifecycleStateDetached {
			for i := previousStateIndex + 1; i < len(values); i++ {
				stateChanges = append(stateChanges, values[i])
			}
			stateChanges = append(stateChanges, ui.AppLifecycleStateDetached)
		} else if previousStateIndex > stateIndex {
			for i := stateIndex; i < previousStateIndex; i++ {
				stateChanges = append([]ui.AppLifecycleState{values[i]}, stateChanges...)
			}
		} else {
			for i := previousStateIndex + 1; i <= stateIndex; i++ {
				stateChanges = append(stateChanges, values[i])
			}
		}
	}
	return stateChanges
}

func setAppLifeCycleState(state ui.AppLifecycleState) {
	binding := SchedulerBindingInstance()
	for _, stateChange := range generateStateTransitions(binding.LifecycleState(), state) {
		binding.HandleAppLifecycleStateChanged(stateChange)
	}
}

func TestTickerTestDart(t *testing.T) {
	if !isolated(t) {
		return
	}

	t.Run("Ticker mute control test", func(t *testing.T) {
		tester := flutter_test.New(t)
		tickCount := 0
		handleTick := func(time.Duration) { tickCount++ }

		ticker := NewTicker(handleTick)
		t.Cleanup(ticker.Dispose)

		equal(t, ticker.IsTicking(), false)
		equal(t, ticker.IsActive(), false)

		ticker.Start()

		equal(t, ticker.IsTicking(), true)
		equal(t, ticker.IsActive(), true)
		equal(t, tickCount, 0)

		err := expectPanics[*foundation.FlutterError](t, func() { ticker.Start() })
		equal(t, len(err.Diagnostics), 3)
		if _, ok := err.Diagnostics[2].(*foundation.DiagnosticsProperty[*Ticker]); !ok {
			t.Fatalf("last diagnostic is %T, want DiagnosticsProperty<Ticker>", err.Diagnostics[2])
		}
		want := "FlutterError\n" +
			"   A ticker was started twice.\n" +
			"   A ticker that is already active cannot be started again without\n" +
			"   first stopping it.\n" +
			"   The affected ticker was:\n" +
			"     Ticker()\n"
		if got := err.ToStringDeep(); !strings.HasPrefix(got, want) {
			t.Fatalf("toStringDeep:\n%s\nwant prefix:\n%s", got, want)
		}

		tester.Pump(10 * time.Millisecond)

		equal(t, tickCount, 1)

		ticker.SetMuted(true)
		tester.Pump(10 * time.Millisecond)

		equal(t, tickCount, 1)
		equal(t, ticker.IsTicking(), false)
		equal(t, ticker.IsActive(), true)

		ticker.SetMuted(false)
		tester.Pump(10 * time.Millisecond)

		equal(t, tickCount, 2)
		equal(t, ticker.IsTicking(), true)
		equal(t, ticker.IsActive(), true)

		ticker.SetMuted(true)
		tester.Pump(10 * time.Millisecond)

		equal(t, tickCount, 2)
		equal(t, ticker.IsTicking(), false)
		equal(t, ticker.IsActive(), true)

		ticker.Stop()

		equal(t, tickCount, 2)
		equal(t, ticker.IsTicking(), false)
		equal(t, ticker.IsActive(), false)

		ticker.SetMuted(false)

		equal(t, tickCount, 2)
		equal(t, ticker.IsTicking(), false)
		equal(t, ticker.IsActive(), false)

		tester.Pump(10 * time.Millisecond)

		equal(t, tickCount, 2)
		equal(t, ticker.IsTicking(), false)
		equal(t, ticker.IsActive(), false)
	})

	t.Run("Ticker control test", func(t *testing.T) {
		flutter_test.New(t)
		var ticker *Ticker
		t.Cleanup(func() { ticker.Dispose() })

		testFunction := func() {
			ticker = NewTicker(func(time.Duration) {})
		}

		testFunction()

		// hasOneLineDescription
		if s := ticker.String(); s == "" || strings.Contains(s, "\n") || strings.HasPrefix(s, "Instance of ") {
			t.Fatalf("not a one-line description: %q", s)
		}
		// The creation stack names the function that created the ticker (a
		// closure of this test).
		if s := ticker.ToString(TickerToStringOptions{DebugIncludeStack: true}); !strings.Contains(s, "TestTickerTestDart") {
			t.Fatalf("stack doesn't name the creator:\n%s", s)
		}
	})

	t.Run("Ticker can be sped up with time dilation", func(t *testing.T) {
		tester := flutter_test.New(t)
		SetTimeDilation(0.5) // Move twice as fast.
		var lastDuration time.Duration
		handleTick := func(duration time.Duration) { lastDuration = duration }

		ticker := NewTicker(handleTick)
		ticker.Start()
		tester.Pump(10 * time.Millisecond)
		tester.Pump(10 * time.Millisecond)
		equal(t, lastDuration, 20*time.Millisecond)

		ticker.Dispose()

		SetTimeDilation(1.0) // restore time dilation, or it will affect other tests
	})

	t.Run("Ticker can be slowed down with time dilation", func(t *testing.T) {
		tester := flutter_test.New(t)
		SetTimeDilation(2.0) // Move half as fast.
		var lastDuration time.Duration
		handleTick := func(duration time.Duration) { lastDuration = duration }

		ticker := NewTicker(handleTick)
		ticker.Start()
		tester.Pump(10 * time.Millisecond)
		tester.Pump(10 * time.Millisecond)
		equal(t, lastDuration, 5*time.Millisecond)

		ticker.Dispose()

		SetTimeDilation(1.0) // restore time dilation, or it will affect other tests
	})

	t.Run("Ticker stops ticking when application is paused", func(t *testing.T) {
		flutter_test.New(t)
		tickCount := 0
		handleTick := func(time.Duration) { tickCount++ }

		ticker := NewTicker(handleTick)
		t.Cleanup(ticker.Dispose)
		ticker.Start()

		equal(t, ticker.IsTicking(), true)
		equal(t, ticker.IsActive(), true)
		equal(t, tickCount, 0)

		setAppLifeCycleState(ui.AppLifecycleStatePaused)

		equal(t, ticker.IsTicking(), false)
		equal(t, ticker.IsActive(), true)

		ticker.Stop()

		setAppLifeCycleState(ui.AppLifecycleStateResumed)
	})

	t.Run("Ticker can be created before application unpauses", func(t *testing.T) {
		tester := flutter_test.New(t)
		setAppLifeCycleState(ui.AppLifecycleStatePaused)

		tickCount := 0
		handleTick := func(time.Duration) { tickCount++ }

		ticker := NewTicker(handleTick)
		t.Cleanup(ticker.Dispose)
		ticker.Start()

		equal(t, tickCount, 0)
		equal(t, ticker.IsTicking(), false)

		tester.Pump(10 * time.Millisecond)

		equal(t, tickCount, 0)
		equal(t, ticker.IsTicking(), false)

		setAppLifeCycleState(ui.AppLifecycleStateResumed)

		tester.Pump(10 * time.Millisecond)

		equal(t, tickCount, 1)
		equal(t, ticker.IsTicking(), true)

		ticker.Stop()
	})

	t.Run("Ticker dispatches memory events", func(t *testing.T) {
		flutter_test.AutomatedTestWidgetsFlutterBindingEnsureInitialized()
		areCreateAndDispose(t, memoryEvents[*Ticker](func() { NewTicker(func(time.Duration) {}).Dispose() }))
	})
}
