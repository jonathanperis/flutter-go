package ui

// Port of painting.dart's Tangent, PathMetrics, PathMetricIterator,
// PathMetric and _PathMeasure (Flutter 3.50.1). The engine side,
// lib/ui/painting/path_measure.cc over SkContourMeasureIter, is ported in
// path_skia_contour_measure.go.

import (
	"iter"
	"strconv"
	"strings"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// Tangent is the location and direction of a point along a path: Vector
// is the (normalized) tangent at Position.
type Tangent struct {
	Position Offset
	Vector   Offset
}

// NewTangent is Tangent(position, vector).
func NewTangent(position, vector Offset) Tangent { return Tangent{position, vector} }

// NewTangentFromAngle is Tangent.fromAngle: the vector is (cos, sin) of
// angle (radians).
func NewTangentFromAngle(position Offset, angle float64) Tangent {
	return Tangent{position, Offset{dart.Cos(angle), dart.Sin(angle)}}
}

// Angle is the angle of the tangent, in radians, clockwise from the
// positive x axis negated to match Path.arcTo's sweepAngle:
// -atan2(vector.dy, vector.dx).
func (t Tangent) Angle() float64 { return -dart.Atan2(t.Vector.Dy, t.Vector.Dx) }

// pathMeasure is painting.dart's _PathMeasure with the engine's
// CanvasPathMeasure.
type pathMeasure struct {
	iter                *skContourMeasureIter
	measures            []*skContourMeasure
	currentContourIndex int
}

func newPathMeasure(path *Path, forceClosed bool) *pathMeasure {
	return &pathMeasure{iter: newContourMeasureIter(path.b.snapshot(), forceClosed), currentContourIndex: -1}
}

func (m *pathMeasure) assertAdvanced(contourIndex int) {
	if contourIndex > m.currentContourIndex {
		panic(dart.NewAssertionError("Iterator must be advanced before index " + strconv.Itoa(contourIndex) +
			" can be used."))
	}
}

func (m *pathMeasure) valid(i int) bool { return i >= 0 && i < len(m.measures) }

func (m *pathMeasure) length(contourIndex int) float64 {
	m.assertAdvanced(contourIndex)
	if m.valid(contourIndex) {
		return float64(m.measures[contourIndex].length)
	}
	return -1
}

func (m *pathMeasure) isClosed(contourIndex int) bool {
	m.assertAdvanced(contourIndex)
	return m.valid(contourIndex) && m.measures[contourIndex].isClosed
}

func (m *pathMeasure) getTangentForOffset(contourIndex int, distance float64) *Tangent {
	m.assertAdvanced(contourIndex)
	if !m.valid(contourIndex) {
		return nil
	}
	pos, tan, ok := m.measures[contourIndex].getPosTan(safeNarrow(distance))
	if !ok {
		return nil
	}
	return &Tangent{Offset{float64(pos.x), float64(pos.y)}, Offset{float64(tan.x), float64(tan.y)}}
}

func (m *pathMeasure) extractPath(contourIndex int, start, end float64, startWithMoveTo bool) *Path {
	m.assertAdvanced(contourIndex)
	p := NewPath()
	if !m.valid(contourIndex) {
		return p
	}
	dst := newSkPathBuilder()
	if m.measures[contourIndex].getSegment(safeNarrow(start), safeNarrow(end), &dst, startWithMoveTo) {
		p.b.assign(dst.snapshot(), fillWinding)
	}
	return p
}

func (m *pathMeasure) nextContour() bool {
	if cm := m.iter.next(); cm != nil {
		m.measures = append(m.measures, cm)
		m.currentContourIndex++
		return true
	}
	return false
}

// PathMetrics is an iterable of the PathMetric of each non-empty contour
// of a path (Path.ComputeMetrics). As in Dart, it has a single iterator:
// iterating consumes it, and later iterations see no more metrics.
type PathMetrics struct {
	iterator *PathMetricIterator
}

func newPathMetrics(path *Path, forceClosed bool) *PathMetrics {
	return &PathMetrics{iterator: &PathMetricIterator{measure: newPathMeasure(path, forceClosed)}}
}

// Iterator is the iterator getter: always the same PathMetricIterator.
func (m *PathMetrics) Iterator() *PathMetricIterator { return m.iterator }

// All iterates the metrics through Iterator (Dart's for-in).
func (m *PathMetrics) All() iter.Seq[*PathMetric] {
	return func(yield func(*PathMetric) bool) {
		it := m.Iterator()
		for it.MoveNext() {
			if !yield(it.Current()) {
				return
			}
		}
	}
}

// String is IterableBase.toString (Iterable.iterableToShortString): the
// metrics in parentheses, elided when long. Like Dart's, it advances the
// shared iterator.
func (m *PathMetrics) String() string {
	var parts []string
	iterablePartsToStrings(m.Iterator(), &parts)
	return "(" + strings.Join(parts, ", ") + ")"
}

// iterablePartsToStrings is dart:core's _iterablePartsToStrings.
func iterablePartsToStrings(it *PathMetricIterator, parts *[]string) {
	const (
		lengthLimit  = 80
		headCount    = 3
		tailCount    = 2
		maxCount     = 100
		overhead     = 2
		ellipsisSize = 3
	)
	removeLast := func() string {
		s := (*parts)[len(*parts)-1]
		*parts = (*parts)[:len(*parts)-1]
		return s
	}
	length, count := 0, 0
	for length < lengthLimit || count < headCount {
		if !it.MoveNext() {
			return
		}
		next := it.Current().String()
		*parts = append(*parts, next)
		length += len(next) + overhead
		count++
	}
	var penultimateString, ultimateString string
	if !it.MoveNext() {
		if count <= headCount+tailCount {
			return
		}
		ultimateString = removeLast()
		penultimateString = removeLast()
	} else {
		penultimate := it.Current()
		count++
		if !it.MoveNext() {
			if count <= headCount+1 {
				*parts = append(*parts, penultimate.String())
				return
			}
			ultimateString = penultimate.String()
			penultimateString = removeLast()
			length += len(ultimateString) + overhead
		} else {
			ultimate := it.Current()
			count++
			for it.MoveNext() {
				penultimate = ultimate
				ultimate = it.Current()
				count++
				if count > maxCount {
					for length > lengthLimit-ellipsisSize-overhead && count > headCount {
						length -= len(removeLast()) + overhead
						count--
					}
					*parts = append(*parts, "...")
					return
				}
			}
			penultimateString = penultimate.String()
			ultimateString = ultimate.String()
			length += len(ultimateString) + len(penultimateString) + 2*overhead
		}
	}
	elision := ""
	if count > len(*parts)+tailCount {
		elision = "..."
		length += ellipsisSize + overhead
	}
	for length > lengthLimit && len(*parts) > headCount {
		length -= len(removeLast()) + overhead
		if elision == "" {
			elision = "..."
			length += ellipsisSize + overhead
		}
	}
	if elision != "" {
		*parts = append(*parts, elision)
	}
	*parts = append(*parts, penultimateString, ultimateString)
}

// PathMetricIterator iterates the contours of a path (PathMetrics'
// iterator).
type PathMetricIterator struct {
	pathMetric *PathMetric
	measure    *pathMeasure
}

// pathRangeError is the RangeError(message) Current throws.
type pathRangeError struct{ message string }

func (e pathRangeError) Error() string { return "RangeError: " + e.message }

// Current is the metric the iterator points at. It panics with a
// RangeError before the first MoveNext and after MoveNext returned false.
func (it *PathMetricIterator) Current() *PathMetric {
	if it.pathMetric == nil {
		panic(pathRangeError{"PathMetricIterator is not pointing to a PathMetric. This can happen in two situations:\n" +
			"- The iteration has not started yet. If so, call \"moveNext\" to start iteration.\n" +
			"- The iterator ran out of elements. If so, check that \"moveNext\" returns true prior to calling \"current\"."})
	}
	return it.pathMetric
}

// MoveNext advances to the next contour, returning false at the end.
func (it *PathMetricIterator) MoveNext() bool {
	if it.measure.nextContour() {
		it.pathMetric = newPathMetric(it.measure)
		return true
	}
	it.pathMetric = nil
	return false
}

// PathMetric is a snapshot of one contour's measurements: Length,
// IsClosed and ContourIndex are fixed when the iterator reaches it.
type PathMetric struct {
	// Length is the contour's total length.
	Length float64
	// IsClosed is whether the contour was closed (or computeMetrics was
	// asked to force it closed).
	IsClosed bool
	// ContourIndex is the zero-based index of the contour among the
	// path's non-empty contours.
	ContourIndex int
	measure      *pathMeasure
}

func newPathMetric(m *pathMeasure) *PathMetric {
	return &PathMetric{
		Length:       m.length(m.currentContourIndex),
		IsClosed:     m.isClosed(m.currentContourIndex),
		ContourIndex: m.currentContourIndex,
		measure:      m,
	}
}

// GetTangentForOffset is the position and tangent at distance along the
// contour (clamped to [0, Length]); nil if it can't be computed (a NaN
// distance).
func (pm *PathMetric) GetTangentForOffset(distance float64) *Tangent {
	return pm.measure.getTangentForOffset(pm.ContourIndex, distance)
}

// ExtractPathOptions are extractPath's named parameters. A nil
// StartWithMoveTo is Dart's default, true.
type ExtractPathOptions struct{ StartWithMoveTo *bool }

// ExtractPath returns the segment of the contour from start to end
// (distances, clamped); an empty path when start > end. The segment
// begins with a moveTo unless StartWithMoveTo is false.
func (pm *PathMetric) ExtractPath(start, end float64, opts ...ExtractPathOptions) *Path {
	move := true
	if len(opts) > 0 && opts[0].StartWithMoveTo != nil {
		move = *opts[0].StartWithMoveTo
	}
	return pm.measure.extractPath(pm.ContourIndex, start, end, move)
}

// String is Dart's toString().
func (pm *PathMetric) String() string {
	return "PathMetric(length: " + dart.DoubleToString(pm.Length) + ", isClosed: " +
		strconv.FormatBool(pm.IsClosed) + ", contourIndex: " + strconv.Itoa(pm.ContourIndex) + ")"
}
