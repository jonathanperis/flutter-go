package dart

import (
	"testing"
	"time"
)

func TestStopwatch(t *testing.T) {
	now := time.Duration(0)
	s := NewStopwatchWithClock(func() time.Duration { return now })
	if s.IsRunning() || s.Elapsed() != 0 {
		t.Fatal("new stopwatch")
	}
	now = 5 * time.Millisecond
	s.Start()
	now += 3*time.Millisecond + 999*time.Microsecond
	if s.ElapsedMilliseconds() != 3 || s.ElapsedMicroseconds() != 3999 || !s.IsRunning() {
		t.Fatalf("running %v", s.Elapsed())
	}
	s.Stop()
	now += time.Second
	if s.Elapsed() != 3999*time.Microsecond || s.IsRunning() {
		t.Fatalf("stopped %v", s.Elapsed())
	}
	s.Start() // resumes, keeping the elapsed time
	now += time.Millisecond
	if s.Elapsed() != 4999*time.Microsecond {
		t.Fatalf("resumed %v", s.Elapsed())
	}
	s.Reset() // keeps running from zero
	now += 2 * time.Millisecond
	if s.Elapsed() != 2*time.Millisecond || !s.IsRunning() {
		t.Fatalf("reset %v", s.Elapsed())
	}
	s.Stop()
	if s.Elapsed() != 2*time.Millisecond {
		t.Fatalf("stopped again %v", s.Elapsed())
	}
	s.Start()
	now += time.Millisecond
	s.Stop()
	if s.Elapsed() != 3*time.Millisecond {
		t.Fatalf("accumulated %v", s.Elapsed())
	}
	s.Reset()
	if s.Elapsed() != 0 {
		t.Fatal("reset stopped")
	}
	if NewStopwatch().Elapsed() != 0 {
		t.Fatal("wall clock stopwatch")
	}
}
