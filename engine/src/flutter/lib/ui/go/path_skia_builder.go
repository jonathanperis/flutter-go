package ui

// SkPathBuilder, the path the engine's CanvasPath (lib/ui/painting/path.cc)
// keeps: Skia's src/core/SkPathBuilder.cpp and include/core/SkPathBuilder.h
// (verbs, relative verbs, both arcTo forms, addArc/addRect/addOval/addRRect/
// addPolygon, addPath in append and extend modes, offset, transform,
// computeFiniteBounds), the shapes of src/core/SkPathRawShapes.cpp and
// SkPathMakers.h, SkPathPriv::TrimmedBounds and FindLastMoveToIndex, the
// snapshot SkPath (SkPathData::MakeTransform with the identity), and
// SkRRect::setRectRadii (src/core/SkRRect.cpp, SkScaleToSides.h).
// Convexity, isA and segment-mask bookkeeping are left out: nothing the
// Dart API exposes reads them.

import "math"

const (
	verbMove = iota
	verbLine
	verbQuad
	verbConic
	verbCubic
	verbClose
)

// ptsAdvance is the number of points each verb adds.
var ptsAdvance = [6]int{1, 1, 2, 2, 3, 0}

const (
	fillWinding = 0 // SkPathFillType::kWinding
	fillEvenOdd = 1 // SkPathFillType::kEvenOdd
)

type skPathBuilder struct {
	pts           []skPoint
	verbs         []uint8
	weights       []float32
	fillType      int
	lastMoveIndex int
}

func newSkPathBuilder() skPathBuilder { return skPathBuilder{lastMoveIndex: -1} }

func (b *skPathBuilder) clone() skPathBuilder {
	return skPathBuilder{
		pts:           append([]skPoint(nil), b.pts...),
		verbs:         append([]uint8(nil), b.verbs...),
		weights:       append([]float32(nil), b.weights...),
		fillType:      b.fillType,
		lastMoveIndex: b.lastMoveIndex,
	}
}

func (b *skPathBuilder) reset() {
	*b = skPathBuilder{lastMoveIndex: -1, pts: b.pts[:0], verbs: b.verbs[:0], weights: b.weights[:0]}
}

func (b *skPathBuilder) lastVerb() int {
	if len(b.verbs) == 0 {
		return -1
	}
	return int(b.verbs[len(b.verbs)-1])
}

func (b *skPathBuilder) ensureMove() {
	if len(b.verbs) == 0 {
		b.moveTo(skPoint{})
	} else if b.lastVerb() == verbClose {
		b.moveTo(b.pts[b.lastMoveIndex])
	}
}

func (b *skPathBuilder) moveTo(p skPoint) {
	if b.lastVerb() == verbMove {
		b.pts[len(b.pts)-1] = p
		return
	}
	b.lastMoveIndex = len(b.pts)
	b.pts = append(b.pts, p)
	b.verbs = append(b.verbs, verbMove)
}

func (b *skPathBuilder) lineTo(p skPoint) {
	b.ensureMove()
	b.pts = append(b.pts, p)
	b.verbs = append(b.verbs, verbLine)
}

func (b *skPathBuilder) quadTo(p1, p2 skPoint) {
	b.ensureMove()
	b.pts = append(b.pts, p1, p2)
	b.verbs = append(b.verbs, verbQuad)
}

func (b *skPathBuilder) conicTo(p1, p2 skPoint, w float32) {
	b.ensureMove()
	if w <= 0 {
		b.lineTo(p2)
		return
	}
	b.pts = append(b.pts, p1, p2)
	switch {
	case w == 1:
		b.verbs = append(b.verbs, verbQuad)
	case isFinite32(w):
		b.verbs = append(b.verbs, verbConic)
		b.weights = append(b.weights, w)
	default:
		b.verbs = append(b.verbs, verbLine, verbLine)
	}
}

func (b *skPathBuilder) cubicTo(p1, p2, p3 skPoint) {
	b.ensureMove()
	b.pts = append(b.pts, p1, p2, p3)
	b.verbs = append(b.verbs, verbCubic)
}

func (b *skPathBuilder) close() {
	if len(b.verbs) > 0 && b.lastVerb() != verbClose {
		b.ensureMove()
		b.verbs = append(b.verbs, verbClose)
	}
}

func (b *skPathBuilder) rMoveTo(v skPoint) {
	var last skPoint
	if len(b.pts) > 0 {
		if b.lastVerb() == verbClose {
			last = b.pts[b.lastMoveIndex]
		} else {
			last = b.pts[len(b.pts)-1]
		}
	}
	b.moveTo(last.add(v))
}

func (b *skPathBuilder) back() skPoint { return b.pts[len(b.pts)-1] }

func (b *skPathBuilder) rLineTo(v skPoint) {
	b.ensureMove()
	b.lineTo(b.back().add(v))
}

func (b *skPathBuilder) rQuadTo(p1, p2 skPoint) {
	b.ensureMove()
	base := b.back()
	b.quadTo(base.add(p1), base.add(p2))
}

func (b *skPathBuilder) rConicTo(p1, p2 skPoint, w float32) {
	b.ensureMove()
	base := b.back()
	b.conicTo(base.add(p1), base.add(p2), w)
}

func (b *skPathBuilder) rCubicTo(p1, p2, p3 skPoint) {
	b.ensureMove()
	base := b.back()
	b.cubicTo(base.add(p1), base.add(p2), base.add(p3))
}

func (b *skPathBuilder) lastPt() (skPoint, bool) {
	if len(b.pts) == 0 {
		return skPoint{}, false
	}
	return b.back(), true
}

// --- arcs ---

func arcIsLonePoint(oval skRect, startAngle, sweepAngle float32) (skPoint, bool) {
	if sweepAngle == 0 && (startAngle == 0 || startAngle == 360) {
		return skPoint{oval.r, oval.centerY()}, true
	}
	if oval.width() == 0 && oval.height() == 0 {
		return skPoint{oval.r, oval.t}, true
	}
	return skPoint{}, false
}

func anglesToUnitVectors(startAngle, sweepAngle float32) (startV, stopV skPoint, cw bool) {
	startRad := skDegreesToRadians(startAngle)
	stopRad := skDegreesToRadians(startAngle + sweepAngle)
	s, c := skSinCosSnapToZero(startRad)
	startV = skPoint{c, s}
	s, c = skSinCosSnapToZero(stopRad)
	stopV = skPoint{c, s}
	if startV == stopV {
		sw := fabsf(sweepAngle)
		if sw < 360 && sw > 359 {
			deltaRad := copysignf(float32(1.0/512), sweepAngle)
			for {
				stopRad -= deltaRad
				s, c = skSinCosSnapToZero(stopRad)
				stopV = skPoint{c, s}
				if startV != stopV {
					break
				}
			}
		}
	}
	return startV, stopV, sweepAngle > 0
}

// buildArcConics is SkPathBuilder.cpp's build_arc_conics.
func buildArcConics(oval skRect, start, stop skPoint, cw bool) ([]skConic, skPoint) {
	var m skMatrix
	m.setScale(oval.width()/2, oval.height()/2)
	m.postTranslate(oval.centerX(), oval.centerY())
	conics := buildUnitArc(start, stop, cw, &m)
	if len(conics) == 0 {
		return nil, m.mapPoint(stop)
	}
	return conics, skPoint{}
}

func nearlyEqualPoints(a, b skPoint) bool {
	return skScalarNearlyEqual(a.x, b.x) && skScalarNearlyEqual(a.y, b.y)
}

// arcTo is SkPathBuilder::arcTo(oval, startAngle, sweepAngle, forceMoveTo).
func (b *skPathBuilder) arcTo(oval skRect, startAngle, sweepAngle float32, forceMoveTo bool) {
	if oval.width() < 0 || oval.height() < 0 {
		return
	}
	startAngle = fmodf(startAngle, 360)
	if len(b.verbs) == 0 {
		forceMoveTo = true
	}
	if lone, ok := arcIsLonePoint(oval, startAngle, sweepAngle); ok {
		if forceMoveTo {
			b.moveTo(lone)
		} else {
			b.lineTo(lone)
		}
		return
	}
	startV, stopV, cw := anglesToUnitVectors(startAngle, sweepAngle)
	addPt := func(p skPoint) {
		if forceMoveTo {
			b.moveTo(p)
		} else if !nearlyEqualPoints(b.back(), p) {
			b.lineTo(p)
		}
	}
	if startV == stopV {
		endAngle := skDegreesToRadians(startAngle + sweepAngle)
		radiusX := oval.width() / 2
		radiusY := oval.height() / 2
		s, c := sincosf(endAngle)
		addPt(skPoint{fma32(radiusX, c, oval.centerX()), fma32(radiusY, s, oval.centerY())})
		return
	}
	conics, single := buildArcConics(oval, startV, stopV, cw)
	if len(conics) > 0 {
		addPt(conics[0].pts[0])
		for _, c := range conics {
			b.conicTo(c.pts[1], c.pts[2], c.w)
		}
	} else {
		addPt(single)
	}
}

// addArc is SkPathBuilder::addArc.
func (b *skPathBuilder) addArc(oval skRect, startAngle, sweepAngle float32) {
	if oval.isEmpty() || sweepAngle == 0 {
		return
	}
	if sweepAngle >= 360 || sweepAngle <= -360 {
		startOver90 := startAngle / 90
		startOver90I := skFloatRound(startOver90)
		errv := startOver90 - startOver90I
		if skScalarNearlyEqual(errv, 0) {
			startIndex := fmodf(startOver90I+1, 4)
			if startIndex < 0 {
				startIndex += 4
			}
			b.addOval(oval, sweepAngle > 0, uint(startIndex))
			return
		}
	}
	b.arcTo(oval, startAngle, sweepAngle, true)
}

// arcToSVG is SkPathBuilder::arcTo(rad, angle, arcLarge, arcSweep, endPt),
// the SVG arc; cw is SkPathDirection::kCW.
func (b *skPathBuilder) arcToSVG(rad skPoint, angle float32, large, cw bool, endPt skPoint) {
	b.ensureMove()
	src := [2]skPoint{b.back(), endPt}
	if rad.x == 0 || rad.y == 0 {
		b.lineTo(endPt)
		return
	}
	if src[0] == src[1] {
		b.lineTo(endPt)
		return
	}
	rx := fabsf(rad.x)
	ry := fabsf(rad.y)
	mid := src[0].sub(src[1])
	mid = skPoint{mul32(mid.x, 0.5), mul32(mid.y, 0.5)}
	var pointTransform skMatrix
	pointTransform.setRotate(-angle)
	tm := pointTransform.mapPoint(mid)
	squareRx := mul32(rx, rx)
	squareRy := mul32(ry, ry)
	squareX := mul32(tm.x, tm.x)
	squareY := mul32(tm.y, tm.y)
	radiiScale := squareX/squareRx + squareY/squareRy
	if radiiScale > 1 {
		radiiScale = sqrtf(radiiScale)
		rx = mul32(rx, radiiScale)
		ry = mul32(ry, radiiScale)
	}
	pointTransform.setScale(1/rx, 1/ry)
	pointTransform.preRotate(-angle)
	var unit [2]skPoint
	pointTransform.mapPoints(unit[:], src[:])
	delta := unit[1].sub(unit[0])
	d := fma32(delta.x, delta.x, mul32(delta.y, delta.y))
	scaleFactorSquared := stdMax(1/d-0.25, 0)
	scaleFactor := sqrtf(scaleFactorSquared)
	if (!cw) != large {
		scaleFactor = -scaleFactor
	}
	delta = delta.scale(scaleFactor)
	center := unit[0].add(unit[1])
	center = skPoint{mul32(center.x, 0.5), mul32(center.y, 0.5)}
	center = skPoint{center.x + -delta.y, center.y + delta.x}
	unit[0] = unit[0].sub(center)
	unit[1] = unit[1].sub(center)
	theta1 := atan2f(unit[0].y, unit[0].x)
	theta2 := atan2f(unit[1].y, unit[1].x)
	thetaArc := theta2 - theta1
	if thetaArc < 0 && cw {
		thetaArc += skTwoPI
	} else if thetaArc > 0 && !cw {
		thetaArc -= skTwoPI
	}
	if fabsf(thetaArc) < skPIOverMillion {
		b.lineTo(endPt)
		return
	}
	pointTransform.setRotate(angle)
	pointTransform.preScale(rx, ry)
	segments := skScalarCeilToInt(fabsf(thetaArc / skTwoPIOver3))
	thetaWidth := thetaArc / float32(segments)
	t := tanf(mul32(0.5, thetaWidth))
	if !isFinite32(t) {
		return
	}
	startTheta := theta1
	w := sqrtf(fma32(cosf(thetaWidth), 0.5, 0.5))
	isInt := func(v float32) bool { return v == floorf(v) }
	expectIntegers := skScalarNearlyZero(skPIOver2-fabsf(thetaWidth)) && isInt(rx) && isInt(ry) &&
		isInt(endPt.x) && isInt(endPt.y)
	for i := 0; i < segments; i++ {
		endTheta := startTheta + thetaWidth
		sinEnd, cosEnd := skSinCosSnapToZero(endTheta)
		unit[1] = skPoint{cosEnd, sinEnd}.add(center)
		unit[0] = unit[1]
		unit[0] = skPoint{unit[0].x + mul32(t, sinEnd), unit[0].y + mul32(-t, cosEnd)}
		var mapped [2]skPoint
		pointTransform.mapPoints(mapped[:], unit[:])
		if expectIntegers {
			for j := range mapped {
				mapped[j] = skPoint{skFloatRound(mapped[j].x), skFloatRound(mapped[j].y)}
			}
		}
		b.conicTo(mapped[0], mapped[1], w)
		startTheta = endTheta
	}
	b.pts[len(b.pts)-1] = endPt
}

// rArcToSVG is SkPathBuilder::rArcTo.
func (b *skPathBuilder) rArcToSVG(rad skPoint, angle float32, large, cw bool, d skPoint) {
	cur, _ := b.lastPt()
	b.arcToSVG(rad, angle, large, cw, cur.add(d))
}

// --- shapes ---

func rectPoints(r skRect) [4]skPoint {
	return [4]skPoint{{r.l, r.t}, {r.r, r.t}, {r.r, r.b}, {r.l, r.b}}
}

type pointIter struct {
	pts     []skPoint
	current uint
	advance uint
}

func newPointIter(pts []skPoint, cw bool, start uint) *pointIter {
	n := uint(len(pts))
	adv := n - 1
	if cw {
		adv = 1
	}
	return &pointIter{pts, start % n, adv}
}

func (it *pointIter) cur() skPoint { return it.pts[it.current] }
func (it *pointIter) next() skPoint {
	it.current = (it.current + it.advance) % uint(len(it.pts))
	return it.cur()
}

func (b *skPathBuilder) addRect(r skRect, cw bool, index uint) {
	rp := rectPoints(r)
	it := newPointIter(rp[:], cw, index)
	b.moveTo(it.cur())
	b.lineTo(it.next())
	b.lineTo(it.next())
	b.lineTo(it.next())
	b.close()
}

func (b *skPathBuilder) addOval(r skRect, cw bool, index uint) {
	cx, cy := r.centerX(), r.centerY()
	op := [4]skPoint{{cx, r.t}, {r.r, cy}, {cx, r.b}, {r.l, cy}}
	ovalIter := newPointIter(op[:], cw, index)
	rp := rectPoints(r)
	ri := index
	if !cw {
		ri++
	}
	rectIter := newPointIter(rp[:], cw, ri)
	b.moveTo(ovalIter.cur())
	for i := 0; i < 4; i++ {
		c := rectIter.next()
		b.conicTo(c, ovalIter.next(), skScalarRoot2Over2)
	}
	b.close()
}

// skRRect is SkRRect: bounds, radii (UL, UR, LR, LL) and type.
type skRRect struct {
	rect  skRect
	radii [4]skPoint
	typ   int
}

const (
	rrEmpty = iota
	rrRect
	rrOval
	rrSimple
	rrNinePatch
	rrComplex
)

func (rr *skRRect) initializeRect(r skRect) bool {
	if !r.isFinite() {
		*rr = skRRect{}
		return false
	}
	rr.rect = r.makeSorted()
	if rr.rect.isEmpty() {
		rr.radii = [4]skPoint{}
		rr.typ = rrEmpty
		return false
	}
	return true
}

func (rr *skRRect) setRect(r skRect) {
	if !rr.initializeRect(r) {
		return
	}
	rr.radii = [4]skPoint{}
	rr.typ = rrRect
}

func clampToZero(radii *[4]skPoint) bool {
	all := true
	for i := range radii {
		if radii[i].x <= 0 || radii[i].y <= 0 {
			radii[i] = skPoint{}
		} else {
			all = false
		}
	}
	return all
}

func flushToZero(a, b *float32) {
	if *a+*b == *a {
		*b = 0
	} else if *a+*b == *b {
		*a = 0
	}
}

func computeMinScale(rad1, rad2, limit, curMin float64) float64 {
	if rad1+rad2 > limit {
		return min(curMin, limit/(rad1+rad2))
	}
	return curMin
}

func adjustRadii(limit, scale float64, a, b *float32) {
	*a = float32(float64(*a) * scale)
	*b = float32(float64(*b) * scale)
	if float64(*a+*b) > limit {
		minR, maxR := a, b
		if *minR > *maxR {
			minR, maxR = maxR, minR
		}
		newMin := *minR
		newMax := float32(limit - float64(newMin))
		for float64(newMax+newMin) > limit {
			newMax = nextafterf0(newMax)
		}
		*maxR = newMax
	}
}

// nextafterf0 is nextafterf(x, 0.0f).
func nextafterf0(x float32) float32 {
	if x == 0 || x != x {
		return x
	}
	return math.Float32frombits(math.Float32bits(x) - 1)
}

func (rr *skRRect) scaleRadii() {
	scale := 1.0
	width := float64(rr.rect.r) - float64(rr.rect.l)
	height := float64(rr.rect.b) - float64(rr.rect.t)
	r := &rr.radii
	scale = computeMinScale(float64(r[0].x), float64(r[1].x), width, scale)
	scale = computeMinScale(float64(r[1].y), float64(r[2].y), height, scale)
	scale = computeMinScale(float64(r[2].x), float64(r[3].x), width, scale)
	scale = computeMinScale(float64(r[3].y), float64(r[0].y), height, scale)
	flushToZero(&r[0].x, &r[1].x)
	flushToZero(&r[1].y, &r[2].y)
	flushToZero(&r[2].x, &r[3].x)
	flushToZero(&r[3].y, &r[0].y)
	if scale < 1.0 {
		adjustRadii(width, scale, &r[0].x, &r[1].x)
		adjustRadii(height, scale, &r[1].y, &r[2].y)
		adjustRadii(width, scale, &r[2].x, &r[3].x)
		adjustRadii(height, scale, &r[3].y, &r[0].y)
	}
	clampToZero(r)
	rr.computeType()
}

func radiiAreNinePatch(r *[4]skPoint) bool {
	return r[0].x == r[3].x && r[0].y == r[1].y && r[1].x == r[2].x && r[3].y == r[2].y
}

func (rr *skRRect) computeType() {
	if rr.rect.isEmpty() {
		rr.typ = rrEmpty
		return
	}
	allEqual := true
	allSquare := rr.radii[0].x == 0 || rr.radii[0].y == 0
	for i := 1; i < 4; i++ {
		if rr.radii[i].x != 0 && rr.radii[i].y != 0 {
			allSquare = false
		}
		if rr.radii[i] != rr.radii[i-1] {
			allEqual = false
		}
	}
	if allSquare {
		rr.typ = rrRect
		return
	}
	if allEqual {
		if rr.radii[0].x >= rr.rect.width()/2 && rr.radii[0].y >= rr.rect.height()/2 {
			rr.typ = rrOval
			hw := skFloatMidpoint(-rr.rect.l, rr.rect.r)
			hh := skFloatMidpoint(-rr.rect.t, rr.rect.b)
			for i := range rr.radii {
				rr.radii[i] = skPoint{hw, hh}
			}
		} else {
			rr.typ = rrSimple
		}
		return
	}
	if radiiAreNinePatch(&rr.radii) {
		rr.typ = rrNinePatch
	} else {
		rr.typ = rrComplex
	}
	if !rr.isValid() {
		rr.setRect(rr.rect)
	}
}

// areRadiiPredicatesValid is SkRRect.cpp's are_radii_predicates_valid.
func areRadiiPredicatesValid(radFromMin, radFromMax, minCoord, maxCoord float32) bool {
	if minCoord > maxCoord || radFromMin < 0 || radFromMax < 0 {
		return false
	}
	limit := maxCoord - minCoord
	sum := radFromMin + radFromMax
	ptFromMin := minCoord + radFromMin
	ptFromMax := maxCoord - radFromMax
	sumValid := sum <= limit || skScalarNearlyEqual(sum, limit) || almostEqualsULPs4(sum, limit)
	ptsValid := ptFromMin <= ptFromMax || skScalarNearlyEqual(ptFromMin, ptFromMax) ||
		almostEqualsULPs4(ptFromMin, ptFromMax)
	if !sumValid || !ptsValid {
		return false
	}
	return minCoord <= ptFromMax && ptFromMin <= maxCoord
}

// almostEqualsULPs4 is SkFloatingPoint<float, 4>::AlmostEquals.
func almostEqualsULPs4(a, b float32) bool {
	if a != a || b != b {
		return false
	}
	bias := func(x float32) uint32 {
		bits := math.Float32bits(x)
		const sign = uint32(1) << 31
		if bits&sign != 0 {
			return ^bits + 1
		}
		return sign | bits
	}
	ba, bb := bias(a), bias(b)
	var d uint32
	if ba >= bb {
		d = ba - bb
	} else {
		d = bb - ba
	}
	return d <= 4
}

func (rr *skRRect) isValid() bool {
	r := rr.rect
	if !r.isFinite() || !(r.l <= r.r && r.t <= r.b) {
		return false
	}
	rd := &rr.radii
	if !(areRadiiPredicatesValid(rd[0].x, rd[1].x, r.l, r.r) && areRadiiPredicatesValid(rd[3].x, rd[2].x, r.l, r.r) &&
		areRadiiPredicatesValid(rd[0].y, rd[3].y, r.t, r.b) && areRadiiPredicatesValid(rd[1].y, rd[2].y, r.t, r.b)) {
		return false
	}
	allZero := rd[0].x == 0 && rd[0].y == 0
	allSquare := rd[0].x == 0 || rd[0].y == 0
	allSame := true
	for i := 1; i < 4; i++ {
		if rd[i].x != 0 || rd[i].y != 0 {
			allZero = false
		}
		if rd[i] != rd[i-1] {
			allSame = false
		}
		if rd[i].x != 0 && rd[i].y != 0 {
			allSquare = false
		}
	}
	nine := radiiAreNinePatch(rd)
	switch rr.typ {
	case rrEmpty:
		return r.isEmpty() && allZero && allSame && allSquare
	case rrRect:
		return !r.isEmpty() && allZero && allSame && allSquare
	case rrOval, rrSimple:
		// The oval radii are set to exactly half the size by computeType.
		return !r.isEmpty() && !allZero && allSame && !allSquare
	case rrNinePatch:
		return !r.isEmpty() && !allZero && !allSame && !allSquare && nine
	default:
		return !r.isEmpty() && !allZero && !allSame && !allSquare && !nine
	}
}

// setRectRadii is SkRRect::setRectRadii.
func (rr *skRRect) setRectRadii(r skRect, radii [4]skPoint) {
	if !rr.initializeRect(r) {
		return
	}
	for _, p := range radii {
		if !p.isFinite() {
			rr.setRect(r)
			return
		}
	}
	rr.radii = radii
	if clampToZero(&rr.radii) {
		rr.setRect(r)
		return
	}
	rr.scaleRadii()
	if !rr.isValid() {
		rr.setRect(r)
	}
}

// addRRect is SkPathBuilder::addRRect(rrect, dir, start).
func (b *skPathBuilder) addRRect(rr skRRect, cw bool, index uint) {
	bounds := rr.rect
	if rr.typ == rrRect || rr.typ == rrEmpty {
		b.addRect(bounds, cw, (index+1)/2)
		return
	}
	if rr.typ == rrOval {
		b.addOval(bounds, cw, index/2)
		return
	}
	startsWithConic := (index&1 == 1) == cw
	L, T, R, B := bounds.l, bounds.t, bounds.r, bounds.b
	rd := rr.radii
	rrp := [8]skPoint{
		{L + rd[0].x, T}, {R - rd[1].x, T}, {R, T + rd[1].y}, {R, B - rd[2].y},
		{R - rd[2].x, B}, {L + rd[3].x, B}, {L, B - rd[3].y}, {L, T + rd[0].y},
	}
	rrIter := newPointIter(rrp[:], cw, index)
	rectStart := index / 2
	if !cw {
		rectStart++
	}
	rp := rectPoints(bounds)
	rectIter := newPointIter(rp[:], cw, rectStart)
	w := skScalarRoot2Over2
	b.moveTo(rrIter.cur())
	if startsWithConic {
		for i := 0; i < 3; i++ {
			c := rectIter.next()
			b.conicTo(c, rrIter.next(), w)
			b.lineTo(rrIter.next())
		}
		c := rectIter.next()
		b.conicTo(c, rrIter.next(), w)
	} else {
		for i := 0; i < 4; i++ {
			b.lineTo(rrIter.next())
			c := rectIter.next()
			b.conicTo(c, rrIter.next(), w)
		}
	}
	b.close()
}

// addPolygon is SkPathBuilder::addPolygon.
func (b *skPathBuilder) addPolygon(pts []skPoint, closed bool) {
	if len(pts) == 0 {
		return
	}
	b.moveTo(pts[0])
	if len(pts) > 1 {
		b.ensureMove()
		for _, p := range pts[1:] {
			b.pts = append(b.pts, p)
			b.verbs = append(b.verbs, verbLine)
		}
	}
	if closed {
		b.close()
	}
}

// --- bounds and snapshots ---

// computeFiniteBounds is SkPathPriv::TrimmedBounds of the points: a
// trailing move counts only when it is the only verb; false if not finite.
func (b *skPathBuilder) computeFiniteBounds() (skRect, bool) {
	pts := b.pts
	if len(b.verbs) > 1 && b.verbs[len(b.verbs)-1] == verbMove {
		if !pts[len(pts)-1].isFinite() {
			return skRect{}, false
		}
		pts = pts[:len(pts)-1]
	}
	return skRectBounds(pts)
}

// skPathData is the immutable SkPath a snapshot makes (DlPath's SkPath).
type skPathData struct {
	pts      []skPoint
	verbs    []uint8
	weights  []float32
	fillType int
	bounds   skRect
}

// snapshot is SkPathBuilder::snapshot(): the same verbs and points, or an
// empty path when the bounds are not finite.
func (b *skPathBuilder) snapshot() *skPathData {
	d := &skPathData{fillType: b.fillType}
	bounds, ok := b.computeFiniteBounds()
	if !ok || len(b.verbs) == 0 {
		return d
	}
	d.pts = append([]skPoint(nil), b.pts...)
	d.verbs = append([]uint8(nil), b.verbs...)
	d.weights = append([]float32(nil), b.weights...)
	d.bounds = bounds
	return d
}

// findLastMoveToIndex is SkPathPriv::FindLastMoveToIndex.
func findLastMoveToIndex(verbs []uint8, ptCount int) int {
	index := ptCount
	for i := len(verbs) - 1; i >= 0; i-- {
		v := verbs[i]
		index -= ptsAdvance[v]
		if v == verbMove {
			return index
		}
	}
	return -1
}

// assign is SkPathBuilder::operator=(const SkPath&), keeping fill type ft.
func (b *skPathBuilder) assign(src *skPathData, ft int) {
	*b = skPathBuilder{
		pts:      append([]skPoint(nil), src.pts...),
		verbs:    append([]uint8(nil), src.verbs...),
		weights:  append([]float32(nil), src.weights...),
		fillType: ft,
	}
	b.lastMoveIndex = findLastMoveToIndex(b.verbs, len(b.pts))
}

// iterate calls f for each verb with its points (pts[0] is the previous
// point for every verb but move; SkPathPriv::Iterate) and conic weight.
func (d *skPathData) iterate(f func(verb uint8, pts []skPoint, w float32)) {
	pi, wi := 0, 0
	for _, v := range d.verbs {
		var w float32
		switch v {
		case verbMove:
			f(v, d.pts[pi:pi+1], 0)
		case verbClose:
			f(v, d.pts[pi-1:pi], 0)
		default:
			if v == verbConic {
				w = d.weights[wi]
				wi++
			}
			f(v, d.pts[pi-1:pi+ptsAdvance[v]], w)
		}
		pi += ptsAdvance[v]
	}
}

// addPath is SkPathBuilder::addPath(src, matrix, mode); extend is
// kExtend_AddPathMode.
func (b *skPathBuilder) addPath(src *skPathData, m skMatrix, extend bool) {
	if len(src.verbs) == 0 {
		return
	}
	canReplace := (!extend && len(b.verbs) <= 1) || len(b.verbs) == 0
	if canReplace && m.isIdentity() {
		b.assign(src, b.fillType)
		return
	}
	if !extend && !m.hasPerspective() {
		if b.lastVerb() == verbMove {
			b.verbs = b.verbs[:len(b.verbs)-1]
			b.pts = b.pts[:len(b.pts)-1]
		}
		b.lastMoveIndex = findLastMoveToIndex(src.verbs, len(src.pts)) + len(b.pts)
		b.verbs = append(b.verbs, src.verbs...)
		start := len(b.pts)
		b.pts = append(b.pts, make([]skPoint, len(src.pts))...)
		m.mapPoints(b.pts[start:], src.pts)
		b.weights = append(b.weights, src.weights...)
		return
	}
	first := true
	src.iterate(func(v uint8, pts []skPoint, w float32) {
		var mp [3]skPoint
		switch v {
		case verbMove:
			m.mapPoints(mp[:1], pts[:1])
			if first && extend && len(b.verbs) != 0 {
				b.ensureMove()
				if last, ok := b.lastPt(); !ok || last != mp[0] {
					b.lineTo(mp[0])
				}
			} else {
				b.moveTo(mp[0])
			}
		case verbLine:
			m.mapPoints(mp[:1], pts[1:2])
			b.lineTo(mp[0])
		case verbQuad:
			m.mapPoints(mp[:2], pts[1:3])
			b.quadTo(mp[0], mp[1])
		case verbConic:
			m.mapPoints(mp[:2], pts[1:3])
			b.conicTo(mp[0], mp[1], w)
		case verbCubic:
			m.mapPoints(mp[:3], pts[1:4])
			b.cubicTo(mp[0], mp[1], mp[2])
		case verbClose:
			b.close()
		}
		first = false
	})
}

// transform is SkPathBuilder::transform(matrix).
func (b *skPathBuilder) transform(m skMatrix) {
	if m.isIdentity() || len(b.verbs) == 0 {
		return
	}
	if m.hasPerspective() {
		src := b.snapshot()
		ft := b.fillType
		b.reset()
		b.fillType = ft
		src = perspectiveClip(src, &m)
		src.iterate(func(v uint8, pts []skPoint, w float32) {
			switch v {
			case verbMove:
				b.moveTo(pts[0])
			case verbLine:
				b.lineTo(pts[1])
			case verbQuad:
				b.conicTo(pts[1], pts[2], conicTransformW(pts, 1, &m))
			case verbConic:
				b.conicTo(pts[1], pts[2], conicTransformW(pts, w, &m))
			case verbCubic:
				b.subdivideCubicTo(pts, 2)
			case verbClose:
				b.close()
			}
		})
	}
	m.mapPoints(b.pts, b.pts)
}

func (b *skPathBuilder) subdivideCubicTo(pts []skPoint, level int) {
	level--
	if level >= 0 {
		tmp := chopCubicAt(pts, 0.5)
		b.subdivideCubicTo(tmp[0:4], level)
		b.subdivideCubicTo(tmp[3:7], level)
	} else {
		b.cubicTo(pts[1], pts[2], pts[3])
	}
}

// offset is SkPathBuilder::offset.
func (b *skPathBuilder) offset(dx, dy float32) {
	for i := range b.pts {
		b.pts[i] = skPoint{b.pts[i].x + dx, b.pts[i].y + dy}
	}
}

// perspectiveClip is SkPathPriv::PerspectiveClip (src/core/SkPath.cpp)
// followed by the caller's choice: the path itself when it lies entirely
// in front of the w = 1/16384 plane, an empty path when it lies behind it
// (or the plane is degenerate). ENGINE GAP: a path crossing the plane is
// clipped by Skia's SkEdgeClipper, which is not ported; that case panics.
func perspectiveClip(src *skPathData, m *skMatrix) *skPathData {
	a, b, c := float64(m.m[kMPersp0]), float64(m.m[kMPersp1]), float64(m.m[kMPersp2]-float32(1.0/16384))
	empty := &skPathData{fillType: src.fillType}
	dmag := math.Sqrt(float64(a*a) + float64(b*b))
	var fa, fb, fc float32
	if dmag == 0 {
		return src // normalize gives the plane 0x + 0y + 1: all positive, no clip
	}
	ds := 1.0 / dmag
	a, b, c = float64(a*ds), float64(b*ds), float64(c*ds)
	if math.IsInf(a, 0) || math.IsInf(b, 0) || math.IsInf(c, 0) || a != a || b != b || c != c || (a == 0 && b == 0) {
		return empty
	}
	fa, fb, fc = float32(a), float32(b), float32(c)
	bounds := src.bounds
	var minX, maxX, minY, maxY float32
	if fa >= 0 {
		minX, maxX = bounds.l, bounds.r
	} else {
		minX, maxX = bounds.r, bounds.l
	}
	if fb >= 0 {
		minY, maxY = bounds.t, bounds.b
	} else {
		minY, maxY = bounds.b, bounds.t
	}
	eval := func(x, y float32) float32 { return fma32(fa, x, mul32(fb, y)) + fc }
	test := eval(minX, minY)
	sign := mul32(test, eval(maxX, maxY))
	if sign > 0 {
		if test < 0 {
			return empty
		}
		return src
	}
	panic(unimplementedPathError("Path.transform with a perspective matrix whose w = 0 plane crosses the path " +
		"(Skia's SkEdgeClipper path clipping is not ported)"))
}
