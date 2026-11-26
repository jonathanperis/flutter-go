package animation

import (
	"math"
	"testing"
	"time"
)

const ms = time.Millisecond

func frame(s *Scheduler, ts time.Duration) { s.BeginFrame(ts); s.EndFrame() }

func TestForwardFromIdleStartsAtFirstTick(t *testing.T) {
	s := NewScheduler()
	frame(s, 100*ms) // an idle frame; time passes
	c := NewController(s, 200*ms)
	var statuses []Status
	c.AddStatusListener(func(st Status) { statuses = append(statuses, st) })
	c.Forward() // outside a frame
	frame(s, 150*ms)
	if c.Value() != 0 {
		t.Fatalf("first tick defines the start time, value = %v", c.Value())
	}
	frame(s, 250*ms)
	if math.Abs(c.Value()-0.5) > 1e-12 {
		t.Fatalf("value after 100ms of 200ms = %v", c.Value())
	}
	frame(s, 400*ms)
	if c.Value() != 1 || c.IsAnimating() {
		t.Fatalf("value %v animating %v", c.Value(), c.IsAnimating())
	}
	if len(statuses) != 2 || statuses[0] != Forward || statuses[1] != Completed {
		t.Fatalf("statuses %v", statuses)
	}
	if s.HasScheduledFrame() {
		t.Fatal("a finished animation must stop requesting frames")
	}
}

func TestStartedInsideFrameUsesFrameTimestamp(t *testing.T) {
	s := NewScheduler()
	c := NewController(s, 100*ms)
	s.BeginFrame(1000 * ms)
	c.Forward() // e.g. from didUpdateWidget during build
	s.EndFrame()
	frame(s, 1050*ms)
	if math.Abs(c.Value()-0.5) > 1e-12 {
		t.Fatalf("value = %v, want 0.5", c.Value())
	}
}

func TestReverseDurationScalesWithDistance(t *testing.T) {
	s := NewScheduler()
	c := NewController(s, 400*ms)
	c.SetValue(0.25)
	c.Reverse() // 0.25 of the range → 100ms
	frame(s, 0)
	frame(s, 50*ms)
	if math.Abs(c.Value()-0.125) > 1e-12 {
		t.Fatalf("value = %v, want 0.125", c.Value())
	}
	frame(s, 100*ms)
	if c.Value() != 0 || c.Status() != Dismissed {
		t.Fatalf("value %v status %v", c.Value(), c.Status())
	}
}

func TestCubicEndpointsAndMonotonic(t *testing.T) {
	for _, c := range []Curve{Curves.Ease, Curves.EaseIn, Curves.EaseOut, Curves.EaseInOut, Curves.FastOutSlowIn} {
		if c.Transform(0) != 0 || c.Transform(1) != 1 {
			t.Fatalf("%v endpoints", c)
		}
		prev := 0.0
		for i := 1; i < 100; i++ {
			v := c.Transform(float64(i) / 100)
			if v < prev-1e-3 {
				t.Fatalf("%v not monotonic at %d", c, i)
			}
			prev = v
		}
	}
	// Flutter: Curves.easeInOut.transform(0.5) == 0.5 (symmetric curve).
	if v := Curves.EaseInOut.Transform(0.5); math.Abs(v-0.5) > 1e-3 {
		t.Fatalf("easeInOut(0.5) = %v", v)
	}
}
