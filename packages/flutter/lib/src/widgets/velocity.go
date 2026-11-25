package widgets

import "math"

// VelocityTracker is a port of Flutter's VelocityTracker
// (gestures/velocity_tracker.dart) and LeastSquaresSolver (lsq_solver.dart):
// it fits a quadratic to the last 100 ms of pointer positions and reports
// the slope at the newest sample. Times are in seconds; only y is tracked.
type VelocityTracker struct {
	samples [velocityHistory]*pointAtTime
	index   int
	// lastSampleClock is when (on the sampling clock) the last sample was
	// added; Flutter uses a Stopwatch for this.
	lastSampleClock float64
}

const (
	velocityHistory         = 20
	assumePointerStoppedMs  = 40.0
	velocityHorizonMs       = 100.0
	velocityMinSamples      = 3
	precisionErrorTolerance = 1e-10
)

type pointAtTime struct {
	y      float64
	timeUs int64 // event timestamp in microseconds
}

// AddPosition records a sample. eventTime is the pointer event's timestamp;
// clock is the sampling clock now.
func (v *VelocityTracker) AddPosition(eventTime, clock float64, y float64) {
	v.lastSampleClock = clock
	v.index++
	if v.index == velocityHistory {
		v.index = 0
	}
	v.samples[v.index] = &pointAtTime{y: y, timeUs: int64(math.Round(eventTime * 1e6))}
}

// Estimate returns the velocity (px/s) and the distance covered by the
// samples used (Flutter's VelocityEstimate.pixelsPerSecond.dy and .offset.dy).
func (v *VelocityTracker) Estimate(clock float64) (velocity, offset float64, ok bool) {
	if (clock-v.lastSampleClock)*1000 > assumePointerStoppedMs {
		return 0, 0, true
	}
	newest := v.samples[v.index]
	if newest == nil {
		return 0, 0, false
	}
	var x, y, w []float64
	previous, oldest := newest, newest
	index, count := v.index, 0
	for count < velocityHistory {
		s := v.samples[index]
		if s == nil {
			break
		}
		age := float64(newest.timeUs-s.timeUs) / 1000
		delta := math.Abs(float64(s.timeUs-previous.timeUs)) / 1000
		previous = s
		if age > velocityHorizonMs || delta > assumePointerStoppedMs {
			break
		}
		oldest = s
		y = append(y, s.y)
		w = append(w, 1)
		x = append(x, -age)
		if index == 0 {
			index = velocityHistory
		}
		index--
		count++
	}
	offset = newest.y - oldest.y
	if count >= velocityMinSamples {
		if coeffs, ok := leastSquares(x, y, w, 2); ok {
			return coeffs[1] * 1000, offset, true
		}
	}
	return 0, offset, true
}

// leastSquares fits a polynomial of the given degree (Flutter's
// LeastSquaresSolver.solve, Gram-Schmidt QR decomposition).
func leastSquares(x, y, w []float64, degree int) ([]float64, bool) {
	if degree > len(x) {
		return nil, false
	}
	m, n := len(x), degree+1
	a := make([][]float64, n)
	for i := range a {
		a[i] = make([]float64, m)
	}
	for h := 0; h < m; h++ {
		a[0][h] = w[h]
		for i := 1; i < n; i++ {
			a[i][h] = a[i-1][h] * x[h]
		}
	}
	dot := func(p, q []float64) float64 {
		s := 0.0
		for i := range p {
			s += p[i] * q[i]
		}
		return s
	}
	q := make([][]float64, n)
	r := make([][]float64, n)
	for j := 0; j < n; j++ {
		q[j] = append([]float64(nil), a[j]...)
		r[j] = make([]float64, n)
		for i := 0; i < j; i++ {
			d := dot(q[j], q[i])
			for h := 0; h < m; h++ {
				q[j][h] -= d * q[i][h]
			}
		}
		norm := math.Sqrt(dot(q[j], q[j]))
		if norm < precisionErrorTolerance {
			return nil, false
		}
		inv := 1.0 / norm
		for h := 0; h < m; h++ {
			q[j][h] *= inv
		}
		for i := 0; i < n; i++ {
			if i >= j {
				r[j][i] = dot(q[j], a[i])
			}
		}
	}
	wy := make([]float64, m)
	for h := 0; h < m; h++ {
		wy[h] = y[h] * w[h]
	}
	coeffs := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		coeffs[i] = dot(q[i], wy)
		for j := n - 1; j > i; j-- {
			coeffs[i] -= r[i][j] * coeffs[j]
		}
		coeffs[i] /= r[i][i]
	}
	return coeffs, true
}
