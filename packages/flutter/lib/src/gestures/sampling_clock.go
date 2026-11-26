package gestures

// SamplingClock from binding.dart (Flutter 3.50.1), and the hook through
// which the velocity trackers reach GestureBinding.instance.samplingClock.

import (
	"time"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// SamplingClock provides the time and stopwatches used to sample pointer
// events (velocity tracking, resampling). flutter_test's binding replaces
// it with a clock on the test's fake time.
type SamplingClock interface {
	// Now returns the current time (DateTime.now()).
	Now() time.Time
	// Stopwatch returns a new, stopped stopwatch on this clock.
	Stopwatch() *dart.Stopwatch
}

// SamplingClockBase is the default SamplingClock (the wall clock), to use
// as is (NewSamplingClock) or to embed.
type SamplingClockBase struct{}

// NewSamplingClock is SamplingClock().
func NewSamplingClock() *SamplingClockBase { return &SamplingClockBase{} }

// Now is DateTime.now().
func (*SamplingClockBase) Now() time.Time { return time.Now() }

// Stopwatch is Stopwatch().
func (*SamplingClockBase) Stopwatch() *dart.Stopwatch { return dart.NewStopwatch() }

// currentSamplingClock is GestureBinding.instance.samplingClock as the
// velocity trackers read it. GestureBinding points it at its sampling clock
// when it is initialized; until then (no binding) it is the wall clock.
var currentSamplingClock = func() SamplingClock { return NewSamplingClock() }

// DebugSetSamplingClock replaces the sampling clock the velocity trackers
// use (GestureBinding.instance.samplingClock) until restore is called, for
// tests that run without a GestureBinding. Not a Flutter API.
func DebugSetSamplingClock(clock func() SamplingClock) (restore func()) {
	previous := currentSamplingClock
	currentSamplingClock = clock
	return func() { currentSamplingClock = previous }
}
