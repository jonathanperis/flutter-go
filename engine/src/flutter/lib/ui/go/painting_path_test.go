package ui

// Port of the engine's testing/dart/path_test.dart (Flutter 3.50.1).
//
// Skipped:
//   - "path combine rect" and "path combine oval": Path.combine needs Skia's
//     PathOps, an ENGINE GAP; TestPathCombineIsAnEngineGap checks that it
//     panics with dart.UnimplementedError instead.
//
// painting_test.dart and canvas_test.dart only use paths to draw (pixels),
// so nothing there is ported here. The difftest (src/difftest/path_test.go)
// compares ~1,900 cases with Flutter bit for bit.

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/dart"
)

func pathExpectPanic(t *testing.T, f func(), check func(any) bool) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil || !check(r) {
			t.Fatalf("expected a panic, got %v", r)
		}
	}()
	f()
}

func pathIsRangeError(r any) bool {
	e, ok := r.(error)
	return ok && strings.HasPrefix(e.Error(), "RangeError: PathMetricIterator is not pointing to a PathMetric.")
}

func TestPathGetBounds(t *testing.T) {
	r := NewRectFromLTRB(1.0, 3.0, 5.0, 7.0)
	p := NewPath()
	p.AddRect(r)
	if p.GetBounds() != r {
		t.Fatal(p.GetBounds())
	}
	p.LineTo(20.0, 15.0)
	if got := p.GetBounds(); got != NewRectFromLTRB(1.0, 3.0, 20.0, 15.0) {
		t.Fatal(got)
	}
}

func TestPathCombineIsAnEngineGap(t *testing.T) {
	c1 := NewRectFromCircle(NewOffset(10.0, 10.0), 10.0)
	c2 := NewRectFromCircle(NewOffset(5.0, 5.0), 10.0)
	p1, p2 := NewPath(), NewPath()
	p1.AddRect(c1)
	p2.AddRect(c2)
	pathExpectPanic(t, func() { PathCombine(PathOperationDifference, p1, p2) }, func(r any) bool {
		var u dart.UnimplementedError
		e, ok := r.(error)
		return ok && errors.As(e, &u) && strings.Contains(u.Message, "ENGINE GAP")
	})
}

func TestPathClone(t *testing.T) {
	p1 := NewPath()
	p1.LineTo(20.0, 20.0)
	p2 := NewPathFrom(p1)
	if p1.GetBounds() != p2.GetBounds() {
		t.Fatal(p1.GetBounds(), p2.GetBounds())
	}
	p1.LineTo(10.0, 30.0)
	if p1.GetBounds().Bottom != p2.GetBounds().Bottom+10 {
		t.Fatal(p1.GetBounds(), p2.GetBounds())
	}
}

func TestPathShift(t *testing.T) {
	bounds := NewRectFromLTRB(0.0, 0.0, 10.0, 10.0)
	p := NewPath()
	p.AddRect(bounds)
	if p.GetBounds() != bounds {
		t.Fatal(p.GetBounds())
	}
	shifted := p.Shift(NewOffset(10, 10))
	if got := shifted.GetBounds(); got != NewRectFromLTRB(10, 10, 20, 20) {
		t.Fatal(got)
	}
}

func TestPathTransformation(t *testing.T) {
	bounds := NewRectFromLTRB(0.0, 0.0, 10.0, 10.0)
	p := NewPath()
	p.AddRect(bounds)
	scaleMatrix := []float64{
		2.5, 0.0, 0.0, 0.0,
		0.0, 0.5, 0.0, 0.0,
		0.0, 0.0, 1.0, 0.0,
		0.0, 0.0, 0.0, 1.0,
	}
	if p.GetBounds() != bounds {
		t.Fatal(p.GetBounds())
	}
	pTransformed := p.Transform(scaleMatrix)
	if got := pTransformed.GetBounds(); got != NewRectFromLTRB(0.0, 0.0, 10*2.5, 10*0.5) {
		t.Fatal(got)
	}
	p2 := NewPath()
	p2.LineTo(10.0, 10.0)
	p.AddPath(p2, NewOffset(10.0, 10.0))
	if got := p.GetBounds(); got != NewRectFromLTRB(0.0, 0.0, 20.0, 20.0) {
		t.Fatal(got)
	}
	p.AddPath(p2, NewOffset(20.0, 20.0), AddPathOptions{Matrix4: scaleMatrix})
	if got := p.GetBounds(); got != NewRectFromLTRB(0.0, 0.0, 20+(10*2.5), 20+(10*.5)) {
		t.Fatal(got)
	}
	p.ExtendWithPath(p2, OffsetZero)
	if got := p.GetBounds(); got != NewRectFromLTRB(0.0, 0.0, 45.0, 25.0) {
		t.Fatal(got)
	}
	p.ExtendWithPath(p2, NewOffset(45.0, 25.0), ExtendWithPathOptions{Matrix4: scaleMatrix})
	if got := p.GetBounds(); got != NewRectFromLTRB(0.0, 0.0, 70.0, 30.0) {
		t.Fatal(got)
	}
}

func pathCloseTo(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestPathMetrics(t *testing.T) {
	simpleHorizontalLine := NewPath()
	simpleHorizontalLine.LineTo(10.0, 0.0)

	// basic tests on horizontal line
	simpleHorizontalMetrics := simpleHorizontalLine.ComputeMetrics()
	pathExpectPanic(t, func() { simpleHorizontalMetrics.Iterator().Current() }, pathIsRangeError)
	if !simpleHorizontalMetrics.Iterator().MoveNext() {
		t.Fatal("moveNext")
	}
	cur := simpleHorizontalMetrics.Iterator().Current()
	if cur == nil || cur.Length != 10.0 || cur.IsClosed {
		t.Fatal(cur)
	}
	simpleExtract := cur.ExtractPath(1.0, 9.0)
	if got := simpleExtract.GetBounds(); got != NewRectFromLTRB(1.0, 0.0, 9.0, 0.0) {
		t.Fatal(got)
	}
	posTan := cur.GetTangentForOffset(1.0)
	if posTan.Position != NewOffset(1.0, 0.0) || posTan.Angle() != 0.0 {
		t.Fatal(posTan)
	}
	if simpleHorizontalMetrics.Iterator().MoveNext() {
		t.Fatal("moveNext at end")
	}
	pathExpectPanic(t, func() { simpleHorizontalMetrics.Iterator().Current() }, pathIsRangeError)

	// test with forceClosed
	simpleMetricsClosed := simpleHorizontalLine.ComputeMetrics(ComputeMetricsOptions{ForceClosed: true})
	pathExpectPanic(t, func() { simpleHorizontalMetrics.Iterator().Current() }, pathIsRangeError)
	if !simpleMetricsClosed.Iterator().MoveNext() {
		t.Fatal("moveNext")
	}
	cur = simpleMetricsClosed.Iterator().Current()
	if cur.Length != 20.0 || !cur.IsClosed { // because we forced close
		t.Fatal(cur)
	}
	simpleExtract2 := cur.ExtractPath(1.0, 9.0)
	if got := simpleExtract2.GetBounds(); got != NewRectFromLTRB(1.0, 0.0, 9.0, 0.0) {
		t.Fatal(got)
	}
	if simpleMetricsClosed.Iterator().MoveNext() {
		t.Fatal("moveNext at end")
	}

	// test getTangentForOffset with vertical line
	simpleVerticalLine := NewPath()
	simpleVerticalLine.LineTo(0.0, 10.0)
	simpleMetricsVertical := simpleVerticalLine.ComputeMetrics()
	simpleMetricsVertical.Iterator().MoveNext()
	posTanVertical := simpleMetricsVertical.Iterator().Current().GetTangentForOffset(5.0)
	if posTanVertical.Position != NewOffset(0.0, 5.0) || !pathCloseTo(posTanVertical.Angle(), -1.5708, .0001) {
		t.Fatal(posTanVertical)
	}

	// test getTangentForOffset with diagonal line
	simpleDiagonalLine := NewPath()
	simpleDiagonalLine.LineTo(10.0, 10.0)
	simpleMetricsDiagonal := simpleDiagonalLine.ComputeMetrics()
	simpleMetricsDiagonal.Iterator().MoveNext()
	midPoint := simpleMetricsDiagonal.Iterator().Current().Length / 2
	posTanDiagonal := simpleMetricsDiagonal.Iterator().Current().GetTangentForOffset(midPoint)
	if posTanDiagonal.Position != NewOffset(5.0, 5.0) || !pathCloseTo(posTanDiagonal.Angle(), -0.7853981633974483, .00001) {
		t.Fatal(posTanDiagonal)
	}

	// test a multi-contour path
	multiContour := NewPath()
	multiContour.LineTo(0.0, 10.0)
	multiContour.MoveTo(10.0, 10.0)
	multiContour.LineTo(10.0, 15.0)
	multiContourMetric := multiContour.ComputeMetrics()
	pathExpectPanic(t, func() { multiContourMetric.Iterator().Current() }, pathIsRangeError)
	if !multiContourMetric.Iterator().MoveNext() || multiContourMetric.Iterator().Current().Length != 10.0 {
		t.Fatal("first contour")
	}
	if !multiContourMetric.Iterator().MoveNext() || multiContourMetric.Iterator().Current().Length != 5.0 {
		t.Fatal("second contour")
	}
	if multiContourMetric.Iterator().MoveNext() {
		t.Fatal("moveNext at end")
	}
	pathExpectPanic(t, func() { multiContourMetric.Iterator().Current() }, pathIsRangeError)
}

func pathFirstLength(p *Path) float64 {
	for m := range p.ComputeMetrics().All() {
		return m.Length
	}
	panic("no metrics")
}

func TestPathMetricsRememberLengthsAndIsClosed(t *testing.T) {
	path := NewPath()
	path.LineTo(0, 10)
	path.Close()
	path.MoveTo(0, 15)
	path.LineTo(10, 15)
	metrics := slices.Collect(path.ComputeMetrics().All())
	if len(metrics) != 2 {
		t.Fatal(len(metrics))
	}
	if metrics[0].Length != 20 || !metrics[0].IsClosed ||
		metrics[0].GetTangentForOffset(4.0).Vector != NewOffset(0.0, 1.0) ||
		pathFirstLength(metrics[0].ExtractPath(4.0, 10.0)) != 6.0 {
		t.Fatal(metrics[0])
	}
	if metrics[1].Length != 10 || metrics[1].IsClosed ||
		metrics[1].GetTangentForOffset(4.0).Vector != NewOffset(1.0, 0.0) ||
		pathFirstLength(metrics[1].ExtractPath(4.0, 6.0)) != 2.0 {
		t.Fatal(metrics[1])
	}
}

func pathMetricsEmpty(m *PathMetrics) bool {
	for range m.All() {
		return false
	}
	return true
}

func TestPathMetricsOnAMutatedPath(t *testing.T) {
	path := NewPath()
	path.LineTo(0, 10)
	metrics := path.ComputeMetrics()
	var firstMetric *PathMetric
	for m := range metrics.All() {
		firstMetric = m
		break
	}
	// We've consumed the iterator.
	if !pathMetricsEmpty(metrics) {
		t.Fatal("metrics not empty")
	}
	check := func(m *PathMetric, length float64, closed bool) {
		t.Helper()
		if m.Length != length || m.IsClosed != closed || m.GetTangentForOffset(4.0).Vector != NewOffset(0.0, 1.0) ||
			pathFirstLength(m.ExtractPath(4.0, 10.0)) != 6.0 {
			t.Fatal(m)
		}
	}
	check(firstMetric, 10, false)

	path.LineTo(10, 10)
	path.LineTo(10, 0)
	path.Close()
	// mutating the path shouldn't have added anything to the iterator.
	if !pathMetricsEmpty(metrics) {
		t.Fatal("metrics not empty")
	}
	check(firstMetric, 10, false)

	// getting a new iterator should update us.
	newMetrics := path.ComputeMetrics()
	var newFirstMetric *PathMetric
	for m := range newMetrics.All() {
		newFirstMetric = m
		break
	}
	if !pathMetricsEmpty(newMetrics) {
		t.Fatal("new metrics not empty")
	}
	check(newFirstMetric, 40, true)
}

func TestPathMetricsToString(t *testing.T) {
	path := NewPath()
	path.LineTo(0, 30)
	path.LineTo(40, 30)
	path.MoveTo(100, 0)
	path.LineTo(100, 30)
	path.LineTo(140, 30)
	path.Close()
	want := "(PathMetric(length: 70.0, isClosed: false, contourIndex: 0), " +
		"PathMetric(length: 120.0, isClosed: true, contourIndex: 1))"
	if got := path.ComputeMetrics().String(); got != want {
		t.Fatal(got)
	}
}

func pathCheckPointWithOffset(t *testing.T, path *Path, inPoint, outwardOffset Offset) {
	t.Helper()
	if !path.Contains(inPoint) {
		t.Errorf("%v should be inside", inPoint)
	}
	if path.Contains(inPoint.Add(outwardOffset)) {
		t.Errorf("%v should be outside", inPoint.Add(outwardOffset))
	}
}

func TestRSuperellipsePathSlimDiagonal(t *testing.T) {
	rse := NewRSuperellipseFromLTRBAndCorners(-50, -50, 50, 50, RRectCorners{
		TopLeft: NewRadiusCircular(1.0), TopRight: NewRadiusCircular(99.0),
		BottomLeft: NewRadiusCircular(99.0), BottomRight: NewRadiusCircular(1.0),
	})
	path := NewPath()
	path.AddRSuperellipse(rse)
	path.Close()

	if !path.Contains(OffsetZero) {
		t.Error("origin")
	}
	for _, p := range []Offset{NewOffset(-49.999, -49.999), NewOffset(-49.999, 49.999),
		NewOffset(49.999, 49.999), NewOffset(49.999, -49.999)} {
		if path.Contains(p) {
			t.Errorf("%v should be outside", p)
		}
	}
	// The pointy ends at the NE and SW corners
	pathCheckPointWithOffset(t, path, NewOffset(-49.70, -49.70), NewOffset(-0.02, -0.02))
	pathCheckPointWithOffset(t, path, NewOffset(49.70, 49.70), NewOffset(0.02, 0.02))

	// Checks two points symmetrical to the origin.
	checkDiagonalPoints := func(p Offset) {
		pathCheckPointWithOffset(t, path, p, NewOffset(0.02, -0.02))
		pathCheckPointWithOffset(t, path, NewOffset(-p.Dx, -p.Dy), NewOffset(-0.02, 0.02))
	}
	// A few other points along the edge
	checkDiagonalPoints(NewOffset(-40.0, -49.58))
	checkDiagonalPoints(NewOffset(-20.0, -45.64))
	checkDiagonalPoints(NewOffset(0.0, -37.01))
	checkDiagonalPoints(NewOffset(20.0, -21.96))
	checkDiagonalPoints(NewOffset(21.05, -20.92))
	checkDiagonalPoints(NewOffset(40.0, 5.68))
}

// --- flutter-go's own checks of the Go API ---

func TestPathEnums(t *testing.T) {
	if PathFillTypeEvenOdd.String() != "PathFillType.evenOdd" || PathFillTypeNonZero.Name() != "nonZero" ||
		len(PathFillTypeValues()) != 2 {
		t.Fatal(PathFillTypeEvenOdd)
	}
	if PathOperationReverseDifference.String() != "PathOperation.reverseDifference" || len(PathOperationValues()) != 5 {
		t.Fatal(PathOperationReverseDifference)
	}
	p := NewPath()
	if p.FillType() != PathFillTypeNonZero || p.String() != "Path" {
		t.Fatal(p.FillType())
	}
	p.SetFillType(PathFillTypeEvenOdd)
	if NewPathFrom(p).FillType() != PathFillTypeEvenOdd {
		t.Fatal("Path.from keeps the fill type")
	}
	p.Reset()
	if p.FillType() != PathFillTypeNonZero {
		t.Fatal("reset restores nonZero (SkPathBuilder::reset)")
	}
}

func TestTangent(t *testing.T) {
	tan := NewTangentFromAngle(NewOffset(1, 2), math.Pi/2)
	if tan.Position != NewOffset(1, 2) || tan.Vector.Dx != dart.Cos(math.Pi/2) || tan.Vector.Dy != 1 {
		t.Fatal(tan)
	}
	if got := NewTangent(OffsetZero, NewOffset(0, 1)).Angle(); got != -math.Pi/2 {
		t.Fatal(got)
	}
}

func TestPathAssertions(t *testing.T) {
	isAssert := func(msg string) func(any) bool {
		return func(r any) bool {
			e, ok := r.(*dart.AssertionError)
			return ok && e.AssertionMessage() == msg
		}
	}
	p := NewPath()
	nan := math.NaN()
	pathExpectPanic(t, func() { p.AddRect(NewRectFromLTRB(nan, 0, 1, 1)) }, isAssert("Rect argument contained a NaN value."))
	pathExpectPanic(t, func() { p.Contains(NewOffset(0, nan)) }, isAssert("Offset argument contained a NaN value."))
	pathExpectPanic(t, func() { p.ArcToPoint(OffsetZero, ArcToPointOptions{Radius: NewRadiusCircular(nan)}) },
		isAssert("Radius argument contained a NaN value."))
	pathExpectPanic(t, func() { p.Transform([]float64{1, 2, 3}) }, isAssert("Matrix4 must have 16 entries."))
	m := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, math.Inf(1)}
	pathExpectPanic(t, func() { p.Transform(m) }, isAssert("Matrix4 entries must be finite."))
}

func TestPathTransformPerspectiveCrossingPlaneIsAnEngineGap(t *testing.T) {
	p := NewPath()
	p.AddRect(NewRectFromLTRB(-100, -100, 100, 100))
	persp := []float64{1, 0, 0, 0.1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	pathExpectPanic(t, func() { p.Transform(persp) }, func(r any) bool {
		_, ok := r.(dart.UnimplementedError)
		return ok
	})
	// Entirely behind the w = 0 plane: clipped out (empty).
	behind := []float64{1, 0, 0, -0.1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	q := NewPath()
	q.AddRect(NewRectFromLTRB(20, 0, 30, 10))
	if got := q.Transform(behind).GetBounds(); got != RectZero {
		t.Fatal(got)
	}
}

type pathRecordingSink struct{ ops []string }

func (s *pathRecordingSink) MoveTo(x, y float32) { s.ops = append(s.ops, "M") }
func (s *pathRecordingSink) LineTo(x, y float32) { s.ops = append(s.ops, "L") }
func (s *pathRecordingSink) QuadTo(x1, y1, x2, y2 float32) {
	s.ops = append(s.ops, "Q")
}
func (s *pathRecordingSink) CubicTo(x1, y1, x2, y2, x3, y3 float32) { s.ops = append(s.ops, "C") }
func (s *pathRecordingSink) Close()                                 { s.ops = append(s.ops, "Z") }

func TestPathEmit(t *testing.T) {
	p := NewPath()
	p.MoveTo(0, 0)
	p.LineTo(10, 0)
	p.QuadraticBezierTo(10, 10, 0, 10)
	p.CubicTo(1, 2, 3, 4, 5, 6)
	p.ConicTo(10, 20, 20, 0, 0.5) // a conic: split into 1 << pow2 quads
	p.Close()
	var s pathRecordingSink
	p.emit(&s, 0.25)
	got := strings.Join(s.ops, "")
	if !strings.HasPrefix(got, "MLQC") || !strings.HasSuffix(got, "Z") {
		t.Fatal(got)
	}
	quads := strings.Count(got, "Q") - 1
	if quads != 4 { // error 2.08 > 0.25 → 0.52 > 0.25 → 0.13: pow2 = 2
		t.Fatal(got)
	}
	// A path with a non-finite point is empty for drawing, as DlPath is.
	q := NewPath()
	q.LineTo(math.Inf(1), 0)
	var e pathRecordingSink
	q.emit(&e, 0.25)
	if len(e.ops) != 0 {
		t.Fatal(e.ops)
	}
}

func TestFma32(t *testing.T) {
	// Values whose float32 fma differs from the separately rounded form.
	a, b, c := float32(1+1.0/(1<<12)), float32(1+1.0/(1<<12)), float32(-(1 + 1.0/(1<<11)))
	if got := fma32(a, b, c); got != 1.0/(1<<24) {
		t.Fatal(got)
	}
	if got := float32(a*b) + c; got != 0 {
		t.Fatal("expected the unfused form to round to 0", got)
	}
	// Round-to-odd: 24929 * 673 = 2^24 + 1 exactly, a float32 tie; the
	// tiny addend breaks the tie upward, but a double sum rounds back onto
	// the tie and then to even (16777216). fmaf gives 16777218.
	x, y, z := float32(24929), float32(673), float32(1.0/(1<<30))
	if got := fma32(x, y, z); got != 16777218 {
		t.Fatal(got)
	}
	if dbl := float32(math.FMA(float64(x), float64(y), float64(z))); dbl != 16777216 {
		t.Fatal("expected double rounding to differ", dbl)
	}
}
