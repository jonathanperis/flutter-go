package dart

import "time"

// Stopwatch is dart:core's Stopwatch: it measures time while running,
// on a monotonic clock. NewStopwatch measures wall-clock time;
// NewStopwatchWithClock measures the given clock (package:clock's
// Clock.stopwatch(), which flutter_test's fake clock uses).
type Stopwatch struct {
	clock   func() time.Duration
	start   time.Duration // clock reading when started (valid while running)
	acc     time.Duration // elapsed before start
	running bool
}

var stopwatchEpoch = time.Now()

// NewStopwatch creates a stopped stopwatch on the wall clock.
func NewStopwatch() *Stopwatch {
	return NewStopwatchWithClock(func() time.Duration { return time.Since(stopwatchEpoch) })
}

// NewStopwatchWithClock creates a stopped stopwatch measuring clock (the
// time since some epoch; it must not go backwards).
func NewStopwatchWithClock(clock func() time.Duration) *Stopwatch {
	return &Stopwatch{clock: clock}
}

// IsRunning reports whether the stopwatch is running.
func (s *Stopwatch) IsRunning() bool { return s.running }

// Start starts the stopwatch; it does nothing if it is already running.
func (s *Stopwatch) Start() {
	if s.running {
		return
	}
	s.start = s.clock()
	s.running = true
}

// Stop stops the stopwatch; the elapsed time is kept.
func (s *Stopwatch) Stop() {
	if !s.running {
		return
	}
	s.acc += s.clock() - s.start
	s.running = false
}

// Reset resets the elapsed time to zero; a running stopwatch keeps
// running.
func (s *Stopwatch) Reset() {
	s.acc = 0
	if s.running {
		s.start = s.clock()
	}
}

// Elapsed is the elapsed time.
func (s *Stopwatch) Elapsed() time.Duration {
	if s.running {
		return s.acc + s.clock() - s.start
	}
	return s.acc
}

// ElapsedMilliseconds is the elapsed time in whole milliseconds.
func (s *Stopwatch) ElapsedMilliseconds() int { return int(s.Elapsed() / time.Millisecond) }

// ElapsedMicroseconds is the elapsed time in whole microseconds.
func (s *Stopwatch) ElapsedMicroseconds() int { return int(s.Elapsed() / time.Microsecond) }
