package ui

// SkContourMeasureIter and SkContourMeasure (Skia's
// src/core/SkContourMeasure.cpp, with resScale 1 as
// lib/ui/painting/path_measure.cc creates them): contour lengths, getPosTan
// and getSegment, with the same subdivision tolerances.

const kMaxTValue = 0x3FFFFFFF

const (
	segLine = iota
	segQuad
	segCubic
	segConic
)

// tValue2Scalar is t * (1.0f / (float)kMaxTValue); the reciprocal is 2^-30.
func tValue2Scalar(t int) float32 { return mul32(float32(t), float32(1.0/(1<<30))) }

type cmSegment struct {
	distance float32
	ptIndex  int
	tValue   int
	segType  int
}

func (s cmSegment) scalarT() float32 { return tValue2Scalar(s.tValue) }

// skContourMeasure is SkContourMeasure.
type skContourMeasure struct {
	segments []cmSegment
	pts      []skPoint
	length   float32
	isClosed bool
}

// skContourMeasureIter is SkContourMeasureIter (nil data: no contours).
type skContourMeasureIter struct {
	path        *skPathData
	verb, pt, w int // iteration position (SkPathPriv::Iterate)
	tolerance   float32
	forceClosed bool
	segments    []cmSegment
	pts         []skPoint
}

func newContourMeasureIter(path *skPathData, forceClosed bool) *skContourMeasureIter {
	it := &skContourMeasureIter{forceClosed: forceClosed, tolerance: 0.5 /* CHEAP_DIST_LIMIT * (1 / resScale), resScale 1 */}
	for _, p := range path.pts {
		if !p.isFinite() {
			return it // SkContourMeasureIter::reset: a non-finite path has no impl
		}
	}
	it.path = path
	return it
}

func tspanBigEnough(tspan int) int { return tspan >> 10 }

func quadTooCurvy(pts []skPoint, tol float32) bool {
	dx := pts[1].x/2 - ((pts[0].x+pts[2].x)/2)/2
	dy := pts[1].y/2 - ((pts[0].y+pts[2].y)/2)/2
	dist := stdMax(fabsf(dx), fabsf(dy))
	return dist > tol
}

func conicTooCurvy(first, mid, last skPoint, tol float32) bool {
	me := first.add(last)
	me = skPoint{mul32(me.x, 0.5), mul32(me.y, 0.5)}
	d := mid.sub(me)
	return stdMax(fabsf(d.x), fabsf(d.y)) > tol
}

func cheapDistExceedsLimit(p skPoint, x, y, tol float32) bool {
	return stdMax(fabsf(x-p.x), fabsf(y-p.y)) > tol
}

func cubicTooCurvy(pts []skPoint, tol float32) bool {
	return cheapDistExceedsLimit(pts[1], skScalarInterp(pts[0].x, pts[3].x, skOneThird),
		skScalarInterp(pts[0].y, pts[3].y, skOneThird), tol) ||
		cheapDistExceedsLimit(pts[2], skScalarInterp(pts[0].x, pts[3].x, skTwoThirds),
			skScalarInterp(pts[0].y, pts[3].y, skTwoThirds), tol)
}

const kMaxRecursionDepth = 8

func (it *skContourMeasureIter) addSeg(distance float32, d float32, ptIndex, tValue, typ int) float32 {
	prev := distance
	distance += d
	if distance > prev {
		it.segments = append(it.segments, cmSegment{distance, ptIndex, tValue, typ})
	}
	return distance
}

func (it *skContourMeasureIter) computeQuadSegs(pts []skPoint, distance float32, mint, maxt, ptIndex, depth int) float32 {
	if depth < kMaxRecursionDepth && tspanBigEnough(maxt-mint) != 0 && quadTooCurvy(pts, it.tolerance) {
		tmp := chopQuadAt(pts, 0.5)
		halft := (mint + maxt) >> 1
		depth++
		distance = it.computeQuadSegs(tmp[0:3], distance, mint, halft, ptIndex, depth)
		return it.computeQuadSegs(tmp[2:5], distance, halft, maxt, ptIndex, depth)
	}
	return it.addSeg(distance, skDistance(pts[0], pts[2]), ptIndex, maxt, segQuad)
}

func (it *skContourMeasureIter) computeConicSegs(c skConic, distance float32, mint int, minPt skPoint,
	maxt int, maxPt skPoint, ptIndex, depth int) float32 {
	halft := (mint + maxt) >> 1
	halfPt := c.evalAt(tValue2Scalar(halft))
	if !halfPt.isFinite() {
		return distance
	}
	if depth < kMaxRecursionDepth && tspanBigEnough(maxt-mint) != 0 && conicTooCurvy(minPt, halfPt, maxPt, it.tolerance) {
		depth++
		distance = it.computeConicSegs(c, distance, mint, minPt, halft, halfPt, ptIndex, depth)
		return it.computeConicSegs(c, distance, halft, halfPt, maxt, maxPt, ptIndex, depth)
	}
	return it.addSeg(distance, skDistance(minPt, maxPt), ptIndex, maxt, segConic)
}

func (it *skContourMeasureIter) computeCubicSegs(pts []skPoint, distance float32, mint, maxt, ptIndex, depth int) float32 {
	if depth < kMaxRecursionDepth && tspanBigEnough(maxt-mint) != 0 && cubicTooCurvy(pts, it.tolerance) {
		tmp := chopCubicAt(pts, 0.5)
		halft := (mint + maxt) >> 1
		depth++
		distance = it.computeCubicSegs(tmp[0:4], distance, mint, halft, ptIndex, depth)
		return it.computeCubicSegs(tmp[3:7], distance, halft, maxt, ptIndex, depth)
	}
	return it.addSeg(distance, skDistance(pts[0], pts[3]), ptIndex, maxt, segCubic)
}

func (it *skContourMeasureIter) hasNext() bool { return it.path != nil && it.verb < len(it.path.verbs) }

// buildSegments is SkContourMeasureIter::Impl::buildSegments.
func (it *skContourMeasureIter) buildSegments() *skContourMeasure {
	ptIndex := -1
	var distance float32
	haveSeenClose := it.forceClosed
	haveSeenMoveTo := false
	it.segments = nil
	it.pts = nil
	d := it.path
	for ; it.verb < len(d.verbs); it.verb++ {
		v := d.verbs[it.verb]
		if haveSeenMoveTo && v == verbMove {
			break
		}
		var pts []skPoint
		if v == verbMove {
			pts = d.pts[it.pt : it.pt+1]
		} else {
			pts = d.pts[it.pt-1 : it.pt+ptsAdvance[v]]
		}
		switch v {
		case verbMove:
			ptIndex++
			it.pts = append(it.pts, pts[0])
			haveSeenMoveTo = true
		case verbLine:
			prev := distance
			distance = it.addSeg(distance, skDistance(pts[0], pts[1]), ptIndex, kMaxTValue, segLine)
			if distance > prev {
				it.pts = append(it.pts, pts[1])
				ptIndex++
			}
		case verbQuad:
			prev := distance
			distance = it.computeQuadSegs(pts, distance, 0, kMaxTValue, ptIndex, 0)
			if distance > prev {
				it.pts = append(it.pts, pts[1], pts[2])
				ptIndex += 2
			}
		case verbConic:
			c := newConic(pts[0], pts[1], pts[2], d.weights[it.w])
			prev := distance
			distance = it.computeConicSegs(c, distance, 0, c.pts[0], kMaxTValue, c.pts[2], ptIndex, 0)
			if distance > prev {
				it.pts = append(it.pts, skPoint{c.w, 0}, pts[1], pts[2])
				ptIndex += 3
			}
		case verbCubic:
			prev := distance
			distance = it.computeCubicSegs(pts, distance, 0, kMaxTValue, ptIndex, 0)
			if distance > prev {
				it.pts = append(it.pts, pts[1], pts[2], pts[3])
				ptIndex += 3
			}
		case verbClose:
			haveSeenClose = true
		}
		it.pt += ptsAdvance[v]
		if v == verbConic {
			it.w++
		}
	}
	if !isFinite32(distance) || len(it.segments) == 0 {
		return nil
	}
	if haveSeenClose {
		prev := distance
		first := it.pts[0]
		distance = it.addSeg(distance, skDistance(it.pts[ptIndex], first), ptIndex, kMaxTValue, segLine)
		if distance > prev {
			it.pts = append(it.pts, first)
		}
	}
	return &skContourMeasure{segments: it.segments, pts: it.pts, length: distance, isClosed: haveSeenClose}
}

// next is SkContourMeasureIter::next.
func (it *skContourMeasureIter) next() *skContourMeasure {
	for it.hasNext() {
		if cm := it.buildSegments(); cm != nil {
			return cm
		}
	}
	return nil
}

// distanceToSegment is SkContourMeasure::distanceToSegment.
func (cm *skContourMeasure) distanceToSegment(distance float32) (int, float32) {
	segs := cm.segments
	// SkTKSearch
	lo, hi := 0, len(segs)-1
	for lo < hi {
		mid := (hi + lo) >> 1
		if segs[mid].distance < distance {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	index := hi
	if segs[hi].distance < distance {
		index = hi + 1 // ~(hi+1) xor'ed back
	}
	seg := segs[index]
	var startT, startD float32
	if index > 0 {
		startD = segs[index-1].distance
		if segs[index-1].ptIndex == seg.ptIndex {
			startT = segs[index-1].scalarT()
		}
	}
	t := startT + mul32(seg.scalarT()-startT, distance-startD)/(seg.distance-startD)
	return index, t
}

// computePosTan is SkContourMeasure.cpp's compute_pos_tan.
func computePosTan(pts []skPoint, segType int, t float32) (pos, tan skPoint) {
	switch segType {
	case segLine:
		pos = skPoint{skScalarInterp(pts[0].x, pts[1].x, t), skScalarInterp(pts[0].y, pts[1].y, t)}
		tan, _ = skSetLength(pts[1].x-pts[0].x, pts[1].y-pts[0].y, 1)
	case segQuad:
		pos = quadEval(pts, t)
		tan = quadTangentAt(pts, t).normalize()
	case segConic:
		c := newConic(pts[0], pts[2], pts[3], pts[1].x)
		pos = c.evalAt(t)
		tan = c.evalTangentAt(t).normalize()
	case segCubic:
		pos = cubicEval(pts, t)
		tan = cubicTangentAt(pts, t).normalize()
	}
	return pos, tan
}

// getPosTan is SkContourMeasure::getPosTan.
func (cm *skContourMeasure) getPosTan(distance float32) (skPoint, skPoint, bool) {
	if distance != distance {
		return skPoint{}, skPoint{}, false
	}
	if distance < 0 {
		distance = 0
	} else if distance > cm.length {
		distance = cm.length
	}
	i, t := cm.distanceToSegment(distance)
	if t != t {
		return skPoint{}, skPoint{}, false
	}
	seg := cm.segments[i]
	pos, tan := computePosTan(cm.pts[seg.ptIndex:], seg.segType, t)
	return pos, tan, true
}

// segTo is SkContourMeasure_segTo.
func segTo(pts []skPoint, segType int, startT, stopT float32, dst *skPathBuilder) {
	if startT == stopT {
		if len(dst.verbs) != 0 {
			last, _ := dst.lastPt()
			dst.lineTo(last)
		}
		return
	}
	switch segType {
	case segLine:
		if stopT == 1 {
			dst.lineTo(pts[1])
		} else {
			dst.lineTo(skPoint{skScalarInterp(pts[0].x, pts[1].x, stopT), skScalarInterp(pts[0].y, pts[1].y, stopT)})
		}
	case segQuad:
		if startT == 0 {
			if stopT == 1 {
				dst.quadTo(pts[1], pts[2])
			} else {
				tmp := chopQuadAt(pts, stopT)
				dst.quadTo(tmp[1], tmp[2])
			}
		} else {
			tmp0 := chopQuadAt(pts, startT)
			if stopT == 1 {
				dst.quadTo(tmp0[3], tmp0[4])
			} else {
				tmp1 := chopQuadAt(tmp0[2:5], (stopT-startT)/(1-startT))
				dst.quadTo(tmp1[1], tmp1[2])
			}
		}
	case segConic:
		c := newConic(pts[0], pts[2], pts[3], pts[1].x)
		if startT == 0 {
			if stopT == 1 {
				dst.conicTo(c.pts[1], c.pts[2], c.w)
			} else if tmp, ok := c.chopAt(stopT); ok {
				dst.conicTo(tmp[0].pts[1], tmp[0].pts[2], tmp[0].w)
			}
		} else {
			if stopT == 1 {
				if tmp, ok := c.chopAt(startT); ok {
					dst.conicTo(tmp[1].pts[1], tmp[1].pts[2], tmp[1].w)
				}
			} else {
				tmp := c.chopAt2(startT, stopT)
				dst.conicTo(tmp.pts[1], tmp.pts[2], tmp.w)
			}
		}
	case segCubic:
		if startT == 0 {
			if stopT == 1 {
				dst.cubicTo(pts[1], pts[2], pts[3])
			} else {
				tmp := chopCubicAt(pts, stopT)
				dst.cubicTo(tmp[1], tmp[2], tmp[3])
			}
		} else {
			tmp0 := chopCubicAt(pts, startT)
			if stopT == 1 {
				dst.cubicTo(tmp0[4], tmp0[5], tmp0[6])
			} else {
				tmp1 := chopCubicAt(tmp0[3:7], (stopT-startT)/(1-startT))
				dst.cubicTo(tmp1[1], tmp1[2], tmp1[3])
			}
		}
	}
}

// getSegment is SkContourMeasure::getSegment.
func (cm *skContourMeasure) getSegment(startD, stopD float32, dst *skPathBuilder, startWithMoveTo bool) bool {
	if startD < 0 {
		startD = 0
	}
	if stopD > cm.length {
		stopD = cm.length
	}
	if !(startD <= stopD) || len(cm.segments) == 0 {
		return false
	}
	si, startT := cm.distanceToSegment(startD)
	if !isFinite32(startT) {
		return false
	}
	ei, stopT := cm.distanceToSegment(stopD)
	if !isFinite32(stopT) {
		return false
	}
	seg, stopSeg := cm.segments[si], cm.segments[ei]
	if startWithMoveTo {
		p, _ := computePosTan(cm.pts[seg.ptIndex:], seg.segType, startT)
		dst.moveTo(p)
	}
	if seg.ptIndex == stopSeg.ptIndex {
		segTo(cm.pts[seg.ptIndex:], seg.segType, startT, stopT, dst)
	} else {
		for {
			segTo(cm.pts[seg.ptIndex:], seg.segType, startT, 1, dst)
			// Segment::Next: skip the segments sharing this point index.
			pi := seg.ptIndex
			for cm.segments[si].ptIndex == pi {
				si++
			}
			seg = cm.segments[si]
			startT = 0
			if seg.ptIndex >= stopSeg.ptIndex {
				break
			}
		}
		segTo(cm.pts[seg.ptIndex:], seg.segType, 0, stopT, dst)
	}
	return true
}
