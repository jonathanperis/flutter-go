package gestures

// Port of velocity_tracker.dart (Flutter 3.50.1).
//
// VelocityTracker is a class Flutter extends (IOSScrollViewFlingVelocity-
// Tracker, MacOSScrollViewFlingVelocityTracker): here a struct whose
// AddPosition, GetVelocityEstimate and GetVelocity dispatch through self,
// so a *VelocityTracker embedded in a subclass (as recognizers hold
// trackers) behaves as the subclass. A type embedding it calls SetSelf.
//
// The stopwatch measuring the time since the last sample comes from
// GestureBinding.instance.samplingClock (currentSamplingClock in
// sampling_clock.go), created on first use as in Dart.

import (
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// Velocity is a velocity in two dimensions (a comparable value: Dart's ==).
type Velocity struct {
	// PixelsPerSecond is the number of pixels per second of velocity in the
	// x and y directions.
	PixelsPerSecond ui.Offset
}

// VelocityZero is a velocity that isn't moving at all (Velocity.zero).
var VelocityZero = Velocity{PixelsPerSecond: ui.OffsetZero}

// Neg is the unary minus.
func (v Velocity) Neg() Velocity { return Velocity{PixelsPerSecond: v.PixelsPerSecond.Neg()} }

// Sub is v - other.
func (v Velocity) Sub(other Velocity) Velocity {
	return Velocity{PixelsPerSecond: v.PixelsPerSecond.Sub(other.PixelsPerSecond)}
}

// Add is v + other.
func (v Velocity) Add(other Velocity) Velocity {
	return Velocity{PixelsPerSecond: v.PixelsPerSecond.Add(other.PixelsPerSecond)}
}

// ClampMagnitude returns a velocity with the same direction whose magnitude
// is clamped to [minValue, maxValue].
func (v Velocity) ClampMagnitude(minValue, maxValue float64) Velocity {
	dartAssert(minValue >= 0.0, nil)
	dartAssert(maxValue >= 0.0 && maxValue >= minValue, nil)
	valueSquared := v.PixelsPerSecond.DistanceSquared()
	if valueSquared > float64(maxValue*maxValue) {
		return Velocity{PixelsPerSecond: v.PixelsPerSecond.Div(v.PixelsPerSecond.Distance()).Mul(maxValue)}
	}
	if valueSquared < float64(minValue*minValue) {
		return Velocity{PixelsPerSecond: v.PixelsPerSecond.Div(v.PixelsPerSecond.Distance()).Mul(minValue)}
	}
	return v
}

func (v Velocity) String() string {
	return "Velocity(" + dart.ToStringAsFixed(v.PixelsPerSecond.Dx, 1) + ", " +
		dart.ToStringAsFixed(v.PixelsPerSecond.Dy, 1) + ")"
}

// VelocityEstimate is a two dimensional velocity estimate (a comparable
// value).
type VelocityEstimate struct {
	// PixelsPerSecond is the number of pixels per second of velocity in the
	// x and y directions.
	PixelsPerSecond ui.Offset
	// Confidence is the confidence the fit had (0 to 1).
	Confidence float64
	// Duration is the time that elapsed between the first and last samples
	// used.
	Duration time.Duration
	// Offset is the difference between the first and last samples used.
	Offset ui.Offset
}

func (e VelocityEstimate) String() string {
	return "VelocityEstimate(" + dart.ToStringAsFixed(e.PixelsPerSecond.Dx, 1) + ", " +
		dart.ToStringAsFixed(e.PixelsPerSecond.Dy, 1) + "; offset: " + e.Offset.String() +
		", duration: " + dart.DurationToString(e.Duration) +
		", confidence: " + dart.ToStringAsFixed(e.Confidence, 1) + ")"
}

// pointAtTime is _PointAtTime.
type pointAtTime struct {
	time  time.Duration
	point ui.Offset
}

func (p *pointAtTime) String() string {
	if p == nil {
		return "null"
	}
	return "_PointAtTime(" + p.point.String() + " at " + dart.DurationToString(p.time) + ")"
}

const (
	assumePointerMoveStoppedMilliseconds = 40
	velocityHistorySize                  = 20
	velocityHorizonMilliseconds          = 100
	velocityMinSampleSize                = 3
)

// velocityTrackerOverrides are the methods subclasses override.
type velocityTrackerOverrides interface {
	AddPosition(time time.Duration, position ui.Offset)
	GetVelocityEstimate() *VelocityEstimate
}

// VelocityTracker computes a pointer's velocity from position samples: a
// least-squares quadratic fit over the last 100 ms (up to 20 samples).
type VelocityTracker struct {
	self velocityTrackerOverrides

	// Kind is the kind of pointer this tracker is for.
	Kind ui.PointerDeviceKind

	stopwatch *dart.Stopwatch
	// Circular buffer; current sample at index.
	samples [velocityHistorySize]*pointAtTime
	index   int
}

// NewVelocityTrackerWithKind is VelocityTracker.withKind.
func NewVelocityTrackerWithKind(kind ui.PointerDeviceKind) *VelocityTracker {
	v := &VelocityTracker{Kind: kind}
	v.self = v
	return v
}

// SetSelf points the tracker at the object embedding it, whose
// AddPosition/GetVelocityEstimate overrides it calls.
func (v *VelocityTracker) SetSelf(self velocityTrackerOverrides) { v.self = self }

func (v *VelocityTracker) outer() velocityTrackerOverrides {
	if v.self == nil {
		panic("gestures: VelocityTracker used before SetSelf; create it with NewVelocityTrackerWithKind, or call SetSelf when embedding it")
	}
	return v.self
}

func (v *VelocityTracker) overridden() bool {
	return v.outer() != velocityTrackerOverrides(v)
}

// sinceLastSample is _sinceLastSample: the time since the last sample was
// added, on GestureBinding.instance.samplingClock.
func (v *VelocityTracker) sinceLastSample() *dart.Stopwatch {
	if v.stopwatch == nil {
		v.stopwatch = currentSamplingClock().Stopwatch()
	}
	return v.stopwatch
}

// AddPosition adds a position as of the given time to the tracker.
func (v *VelocityTracker) AddPosition(time time.Duration, position ui.Offset) {
	if v.overridden() {
		v.self.AddPosition(time, position)
		return
	}
	v.sinceLastSample().Start()
	v.sinceLastSample().Reset()
	v.index++
	if v.index == velocityHistorySize {
		v.index = 0
	}
	v.samples[v.index] = &pointAtTime{point: position, time: time}
}

// GetVelocityEstimate returns an estimate of the velocity of the object
// being tracked, or nil if there is no data on which to base an estimate.
func (v *VelocityTracker) GetVelocityEstimate() *VelocityEstimate {
	if v.overridden() {
		return v.self.GetVelocityEstimate()
	}
	// Has user recently moved since last sample?
	if v.sinceLastSample().ElapsedMilliseconds() > assumePointerMoveStoppedMilliseconds {
		return &VelocityEstimate{PixelsPerSecond: ui.OffsetZero, Confidence: 1.0}
	}

	var x, y, w, tm []float64
	sampleCount := 0
	index := v.index

	newestSample := v.samples[index]
	if newestSample == nil {
		return nil
	}

	previousSample := newestSample
	oldestSample := newestSample

	// Starting with the most recent PointAtTime sample, iterate backwards
	// while the samples represent continuous motion.
	for {
		sample := v.samples[index]
		if sample == nil {
			break
		}

		age := float64((newestSample.time-sample.time)/time.Microsecond) / 1000
		delta := float64(absInt(int((sample.time-previousSample.time)/time.Microsecond))) / 1000
		previousSample = sample
		if age > velocityHorizonMilliseconds || delta > assumePointerMoveStoppedMilliseconds {
			break
		}

		oldestSample = sample
		position := sample.point
		x = append(x, position.Dx)
		y = append(y, position.Dy)
		w = append(w, 1.0)
		tm = append(tm, -age)
		if index == 0 {
			index = velocityHistorySize
		}
		index--

		sampleCount++
		if sampleCount >= velocityHistorySize {
			break
		}
	}

	if sampleCount >= velocityMinSampleSize {
		// Dart's late finals: yFit is solved only when xFit isn't null.
		xFit := NewLeastSquaresSolver(tm, x, w).Solve(2)
		if xFit != nil {
			yFit := NewLeastSquaresSolver(tm, y, w).Solve(2)
			if yFit != nil {
				return &VelocityEstimate{
					// convert from pixels/ms to pixels/s
					PixelsPerSecond: ui.NewOffset(float64(xFit.Coefficients[1]*1000), float64(yFit.Coefficients[1]*1000)),
					Confidence:      xFit.Confidence * yFit.Confidence,
					Duration:        newestSample.time - oldestSample.time,
					Offset:          newestSample.point.Sub(oldestSample.point),
				}
			}
		}
	}

	// We're unable to make a velocity estimate but we did have at least one
	// valid pointer position.
	return &VelocityEstimate{
		PixelsPerSecond: ui.OffsetZero,
		Confidence:      1.0,
		Duration:        newestSample.time - oldestSample.time,
		Offset:          newestSample.point.Sub(oldestSample.point),
	}
}

// GetVelocity computes the velocity of the pointer being tracked (zero if
// there is no estimate).
func (v *VelocityTracker) GetVelocity() Velocity {
	estimate := v.outer().GetVelocityEstimate()
	if estimate == nil || estimate.PixelsPerSecond == ui.OffsetZero {
		return VelocityZero
	}
	return Velocity{PixelsPerSecond: estimate.PixelsPerSecond}
}

func absInt(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

const iosSampleSize = 20

// IOSScrollViewFlingVelocityTracker is a VelocityTracker that approximates
// the scroll velocity of an iOS UIScrollView at the moment the touch is
// released (a weighted average of the last three samples' velocities).
type IOSScrollViewFlingVelocityTracker struct {
	VelocityTracker
	touchSamples [iosSampleSize]*pointAtTime
}

// NewIOSScrollViewFlingVelocityTracker creates an iOS fling velocity
// tracker for the pointer kind.
func NewIOSScrollViewFlingVelocityTracker(kind ui.PointerDeviceKind) *IOSScrollViewFlingVelocityTracker {
	t := &IOSScrollViewFlingVelocityTracker{VelocityTracker: VelocityTracker{Kind: kind}}
	t.SetSelf(t)
	return t
}

// AddPosition adds a position as of the given time; times must not go
// backwards (a FlutterError in debug builds).
func (t *IOSScrollViewFlingVelocityTracker) AddPosition(time time.Duration, position ui.Offset) {
	t.sinceLastSample().Start()
	t.sinceLastSample().Reset()
	if foundation.KDebugMode {
		previousPoint := t.touchSamples[t.index]
		if previousPoint != nil && previousPoint.time > time {
			panic(foundation.NewFlutterError("The position being added (" + position.String() +
				") has a smaller timestamp (" + dart.DurationToString(time) + ") " +
				"than its predecessor: " + previousPoint.String() + "."))
		}
	}
	t.index = (t.index + 1) % iosSampleSize
	t.touchSamples[t.index] = &pointAtTime{point: position, time: time}
}

// previousVelocityAt is _previousVelocityAt: the velocity between two
// adjacent samples; index 0 uses the latest sample and the one before it,
// smaller indices go back in history.
func (t *IOSScrollViewFlingVelocityTracker) previousVelocityAt(index int) ui.Offset {
	endIndex := dartModInt(t.index+index, iosSampleSize)
	startIndex := dartModInt(t.index+index-1, iosSampleSize)
	end := t.touchSamples[endIndex]
	start := t.touchSamples[startIndex]

	if end == nil || start == nil {
		return ui.OffsetZero
	}

	dt := int((end.time - start.time) / time.Microsecond)
	dartAssert(dt >= 0, nil)

	if dt > 0 {
		// Convert dt to milliseconds to preserve floating point precision.
		return end.point.Sub(start.point).Mul(1000).Div(float64(dt) / 1000)
	}
	return ui.OffsetZero
}

// dartModInt is Dart's int % (never negative for a positive divisor).
func dartModInt(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}

// GetVelocityEstimate estimates the velocity at release.
func (t *IOSScrollViewFlingVelocityTracker) GetVelocityEstimate() *VelocityEstimate {
	return t.flingEstimate(0.6, 0.35, 0.05)
}

func (t *IOSScrollViewFlingVelocityTracker) flingEstimate(w2, w1, w0 float64) *VelocityEstimate {
	// Has user recently moved since last sample?
	if t.sinceLastSample().ElapsedMilliseconds() > assumePointerMoveStoppedMilliseconds {
		return &VelocityEstimate{PixelsPerSecond: ui.OffsetZero, Confidence: 1.0}
	}

	// The velocity estimated using this expression is an approximation of
	// the scroll velocity of a scroll view at the moment the user touch was
	// released.
	estimatedVelocity := t.previousVelocityAt(-2).Mul(w2).
		Add(t.previousVelocityAt(-1).Mul(w1)).
		Add(t.previousVelocityAt(0).Mul(w0))

	newestSample := t.touchSamples[t.index]
	var oldestNonNullSample *pointAtTime

	for i := 1; i <= iosSampleSize; i++ {
		oldestNonNullSample = t.touchSamples[(t.index+i)%iosSampleSize]
		if oldestNonNullSample != nil {
			break
		}
	}

	if oldestNonNullSample == nil || newestSample == nil {
		panic(dart.NewAssertionError("There must be at least 1 point in _touchSamples: " + t.touchSamplesString()))
	}
	return &VelocityEstimate{
		PixelsPerSecond: estimatedVelocity,
		Confidence:      1.0,
		Duration:        newestSample.time - oldestNonNullSample.time,
		Offset:          newestSample.point.Sub(oldestNonNullSample.point),
	}
}

func (t *IOSScrollViewFlingVelocityTracker) touchSamplesString() string {
	s := "["
	for i, p := range t.touchSamples {
		if i > 0 {
			s += ", "
		}
		s += p.String()
	}
	return s + "]"
}

// MacOSScrollViewFlingVelocityTracker is a VelocityTracker that
// approximates the scroll velocity of a macOS NSScrollView at the moment
// the touch is released.
type MacOSScrollViewFlingVelocityTracker struct {
	IOSScrollViewFlingVelocityTracker
}

// NewMacOSScrollViewFlingVelocityTracker creates a macOS fling velocity
// tracker for the pointer kind.
func NewMacOSScrollViewFlingVelocityTracker(kind ui.PointerDeviceKind) *MacOSScrollViewFlingVelocityTracker {
	t := &MacOSScrollViewFlingVelocityTracker{}
	t.Kind = kind
	t.SetSelf(t)
	return t
}

// GetVelocityEstimate estimates the velocity at release.
func (t *MacOSScrollViewFlingVelocityTracker) GetVelocityEstimate() *VelocityEstimate {
	return t.flingEstimate(0.15, 0.65, 0.2)
}

func dartAssert(condition bool, message any) {
	if !condition {
		panic(dart.NewAssertionError(message))
	}
}
