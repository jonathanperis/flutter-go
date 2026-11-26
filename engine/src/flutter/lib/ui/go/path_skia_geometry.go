package ui

// Curve math from Skia's src/core/SkGeometry.cpp and SkGeometry.h (quad,
// cubic and conic evaluation, chopping and extrema; SkConic's quad
// approximation and BuildUnitArc) and src/core/SkCubicClipper.cpp
// (ChopMonoAtY, bisection build). skvx float2 arithmetic is written lane by
// lane with every product rounded; scalar expressions that clang fuses use
// fma32 (see path_skia_math.go).

import "math"

// lerp2 is SkGeometry.cpp's interp(v0, v1, t) on float2: v0 + (v1 - v0) * t.
func lerp2(a, b skPoint, t float32) skPoint {
	return skPoint{a.x + mul32(b.x-a.x, t), a.y + mul32(b.y-a.y, t)}
}

// mix2 is unchecked_mix(a, b, t) on float2: (b - a) * t + a.
func mix2(a, b skPoint, t float32) skPoint {
	return skPoint{mul32(b.x-a.x, t) + a.x, mul32(b.y-a.y, t) + a.y}
}

func validUnitDivide(numer, denom float32) (float32, bool) {
	if numer < 0 {
		numer = -numer
		denom = -denom
	}
	if denom == 0 || numer == 0 || numer >= denom {
		return 0, false
	}
	r := numer / denom
	if r != r || r == 0 {
		return 0, false
	}
	return r, true
}

// skFindUnitQuadRoots is SkFindUnitQuadRoots.
func skFindUnitQuadRoots(a, b, c float32) []float32 {
	roots := make([]float32, 0, 2)
	if a == 0 {
		if r, ok := validUnitDivide(-c, b); ok {
			roots = append(roots, r)
		}
		return roots
	}
	dr := math.FMA(float64(b), float64(b), -float64(float64(4*float64(a))*float64(c)))
	if dr < 0 {
		return roots
	}
	dr = math.Sqrt(dr)
	r := float32(dr)
	if !isFinite32(r) {
		return roots
	}
	var q float32
	if b < 0 {
		q = -(b - r) / 2
	} else {
		q = -(b + r) / 2
	}
	if v, ok := validUnitDivide(q, a); ok {
		roots = append(roots, v)
	}
	if v, ok := validUnitDivide(c, q); ok {
		roots = append(roots, v)
	}
	if len(roots) == 2 {
		if roots[0] > roots[1] {
			roots[0], roots[1] = roots[1], roots[0]
		} else if roots[0] == roots[1] {
			roots = roots[:1]
		}
	}
	return roots
}

// --- quads ---

// quadCoeffEval is SkQuadCoeff(src).eval(t).
func quadEval(src []skPoint, t float32) skPoint {
	c := src[0]
	b := src[1].sub(c)
	b = b.add(b)
	p1x2 := src[1].add(src[1])
	a := src[2].sub(p1x2).add(c)
	return skPoint{mul32(mul32(a.x, t)+b.x, t) + c.x, mul32(mul32(a.y, t)+b.y, t) + c.y}
}

// quadTangentAt is SkEvalQuadTangentAt.
func quadTangentAt(src []skPoint, t float32) skPoint {
	if (t == 0 && src[0] == src[1]) || (t == 1 && src[1] == src[2]) {
		return src[2].sub(src[0])
	}
	b := src[1].sub(src[0])
	a := src[2].sub(src[1]).sub(b)
	tt := skPoint{mul32(a.x, t) + b.x, mul32(a.y, t) + b.y}
	return tt.add(tt)
}

// chopQuadAt is SkChopQuadAt.
func chopQuadAt(src []skPoint, t float32) [5]skPoint {
	p01 := lerp2(src[0], src[1], t)
	p12 := lerp2(src[1], src[2], t)
	return [5]skPoint{src[0], p01, lerp2(p01, p12, t), p12, src[2]}
}

func isNotMonotonic(a, b, c float32) bool {
	ab := a - b
	bc := b - c
	if ab < 0 {
		bc = -bc
	}
	return ab == 0 || bc < 0
}

// chopQuadAtYExtrema is SkChopQuadAtYExtrema: 0 or 1 chops in dst.
func chopQuadAtYExtrema(src []skPoint) (int, [5]skPoint) {
	a, b, c := src[0].y, src[1].y, src[2].y
	if isNotMonotonic(a, b, c) {
		if t, ok := validUnitDivide(a-b, a-b-b+c); ok {
			dst := chopQuadAt(src, t)
			dst[1].y = dst[2].y
			dst[3].y = dst[2].y
			return 1, dst
		}
		if fabsf(a-b) < fabsf(b-c) {
			b = a
		} else {
			b = c
		}
	}
	return 0, [5]skPoint{{src[0].x, a}, {src[1].x, b}, {src[2].x, c}}
}

// --- cubics ---

// cubicEval is SkCubicCoeff(src).eval(t).
func cubicEval(src []skPoint, t float32) skPoint {
	ax := (src[3].x + mul32(3, src[1].x-src[2].x)) - src[0].x
	ay := (src[3].y + mul32(3, src[1].y-src[2].y)) - src[0].y
	bx := mul32(3, (src[2].x-(src[1].x+src[1].x))+src[0].x)
	by := mul32(3, (src[2].y-(src[1].y+src[1].y))+src[0].y)
	cx := mul32(3, src[1].x-src[0].x)
	cy := mul32(3, src[1].y-src[0].y)
	return skPoint{
		mul32(mul32(mul32(ax, t)+bx, t)+cx, t) + src[0].x,
		mul32(mul32(mul32(ay, t)+by, t)+cy, t) + src[0].y,
	}
}

// cubicDerivative is SkGeometry.cpp's eval_cubic_derivative.
func cubicDerivative(src []skPoint, t float32) skPoint {
	ax := (src[3].x + mul32(3, src[1].x-src[2].x)) - src[0].x
	ay := (src[3].y + mul32(3, src[1].y-src[2].y)) - src[0].y
	bx := (src[2].x - (src[1].x + src[1].x)) + src[0].x
	by := (src[2].y - (src[1].y + src[1].y)) + src[0].y
	bx, by = bx+bx, by+by
	cx, cy := src[1].x-src[0].x, src[1].y-src[0].y
	return skPoint{mul32(mul32(ax, t)+bx, t) + cx, mul32(mul32(ay, t)+by, t) + cy}
}

// cubicTangentAt is SkEvalCubicAt's tangent.
func cubicTangentAt(src []skPoint, t float32) skPoint {
	if (t == 0 && src[0] == src[1]) || (t == 1 && src[2] == src[3]) {
		var tan skPoint
		if t == 0 {
			tan = src[2].sub(src[0])
		} else {
			tan = src[3].sub(src[1])
		}
		if tan.x == 0 && tan.y == 0 {
			tan = src[3].sub(src[0])
		}
		return tan
	}
	return cubicDerivative(src, t)
}

// skFindCubicExtrema is SkFindCubicExtrema.
func skFindCubicExtrema(a, b, c, d float32) []float32 {
	A := fma32(3, b-c, d-a)
	B := mul32(2, ((a-b)-b)+c)
	C := b - a
	return skFindUnitQuadRoots(A, B, C)
}

// chopCubicAt is SkChopCubicAt(src, dst[7], t).
func chopCubicAt(src []skPoint, t float32) [7]skPoint {
	if t == 1 {
		return [7]skPoint{src[0], src[1], src[2], src[3], src[3], src[3], src[3]}
	}
	ab := mix2(src[0], src[1], t)
	bc := mix2(src[1], src[2], t)
	cd := mix2(src[2], src[3], t)
	abc := mix2(ab, bc, t)
	bcd := mix2(bc, cd, t)
	abcd := mix2(abc, bcd, t)
	return [7]skPoint{src[0], ab, abc, abcd, bcd, cd, src[3]}
}

// chopCubicAt2 is SkChopCubicAt(src, dst[10], t0, t1).
func chopCubicAt2(src []skPoint, t0, t1 float32) [10]skPoint {
	var dst [10]skPoint
	if t1 == 1 {
		d := chopCubicAt(src, t0)
		copy(dst[:], d[:])
		dst[7], dst[8], dst[9] = src[3], src[3], src[3]
		return dst
	}
	// Lanes lo (t0) and hi (t1).
	ab0, ab1 := mix2(src[0], src[1], t0), mix2(src[0], src[1], t1)
	bc0, bc1 := mix2(src[1], src[2], t0), mix2(src[1], src[2], t1)
	cd0, cd1 := mix2(src[2], src[3], t0), mix2(src[2], src[3], t1)
	abc0, abc1 := mix2(ab0, bc0, t0), mix2(ab1, bc1, t1)
	bcd0, bcd1 := mix2(bc0, cd0, t0), mix2(bc1, cd1, t1)
	abcd0, abcd1 := mix2(abc0, bcd0, t0), mix2(abc1, bcd1, t1)
	// middle = unchecked_mix(abc, bcd, shuffle<2,3,0,1>(T)): lo uses t1, hi t0.
	mid0, mid1 := mix2(abc0, bcd0, t1), mix2(abc1, bcd1, t0)
	return [10]skPoint{src[0], ab0, abc0, abcd0, mid0, mid1, abcd1, bcd1, cd1, src[3]}
}

// chopCubicAtValues is SkChopCubicAt(src, dst, tValues, tCount).
func chopCubicAtValues(src []skPoint, tValues []float32) []skPoint {
	dst := make([]skPoint, 3*len(tValues)+4)
	if len(tValues) == 0 {
		copy(dst, src[:4])
		return dst
	}
	cur := src
	out := 0
	i := 0
	for ; i < len(tValues)-1; i += 2 {
		t0, t1 := tValues[i], tValues[i+1]
		if i != 0 {
			last := tValues[i-1]
			t0 = skPin01((t0 - last) / (1 - last))
			t1 = skPin01((t1 - last) / (1 - last))
		}
		d := chopCubicAt2(cur, t0, t1)
		copy(dst[out:], d[:])
		out += 6
		cur = dst[out:]
	}
	if i < len(tValues) {
		t := tValues[i]
		if i != 0 {
			last := tValues[i-1]
			t = (t - last) / (1 - last)
			t = skPin01(t)
		}
		d := chopCubicAt(cur, t)
		copy(dst[out:], d[:])
	}
	return dst
}

// skPin01 is skvx::pin / SkTPin(x, 0, 1): max(lo, min(x, hi)), so NaN gives 0.
func skPin01(x float32) float32 {
	m := x
	if 1 < x {
		m = 1
	}
	if 0 < m {
		return m
	}
	return 0
}

// chopCubicAtYExtrema is SkChopCubicAtYExtrema: the number of chops and dst.
func chopCubicAtYExtrema(src []skPoint) (int, []skPoint) {
	t := skFindCubicExtrema(src[0].y, src[1].y, src[2].y, src[3].y)
	dst := chopCubicAtValues(src, t)
	if len(t) > 0 {
		dst[2].y = dst[3].y
		dst[4].y = dst[3].y
		if len(t) == 2 {
			dst[5].y = dst[6].y
			dst[7].y = dst[6].y
		}
	}
	return len(t), dst
}

// chopMonoCubicAtY is SkCubicClipper::ChopMonoAtY (bisection).
func chopMonoCubicAtY(pts []skPoint, y float32) (float32, bool) {
	ycrv := [4]float32{pts[0].y - y, pts[1].y - y, pts[2].y - y, pts[3].y - y}
	var tNeg, tPos float32
	switch {
	case ycrv[0] < 0:
		if ycrv[3] < 0 {
			return 0, false
		}
		tNeg, tPos = 0, 1
	case ycrv[0] > 0:
		if ycrv[3] > 0 {
			return 0, false
		}
		tNeg, tPos = 1, 0
	default:
		return 0, true
	}
	tol := float32(1.0 / 65536)
	for {
		tMid := (tPos + tNeg) / 2
		y01 := skScalarInterp(ycrv[0], ycrv[1], tMid)
		y12 := skScalarInterp(ycrv[1], ycrv[2], tMid)
		y23 := skScalarInterp(ycrv[2], ycrv[3], tMid)
		y012 := skScalarInterp(y01, y12, tMid)
		y123 := skScalarInterp(y12, y23, tMid)
		y0123 := skScalarInterp(y012, y123, tMid)
		if y0123 == 0 {
			return tMid, true
		}
		if y0123 < 0 {
			tNeg = tMid
		} else {
			tPos = tMid
		}
		if fabsf(tPos-tNeg) <= tol {
			break
		}
	}
	return (tNeg + tPos) / 2, true
}

// --- conics ---

// skConic is SkConic.
type skConic struct {
	pts [3]skPoint
	w   float32
}

func newConic(p0, p1, p2 skPoint, w float32) skConic {
	c := skConic{pts: [3]skPoint{p0, p1, p2}}
	if w > 0 && isFinite32(w) {
		c.w = w
	} else {
		c.w = 1
	}
	return c
}

// eval is SkConicCoeff(conic).eval(t): numer / denom lane-wise.
func (c skConic) evalAt(t float32) skPoint {
	p0, p1, p2 := c.pts[0], c.pts[1], c.pts[2]
	p1w := p1.scale(c.w)
	nA := p2.sub(p1w.add(p1w)).add(p0)
	nB := p1w.sub(p0)
	nB = nB.add(nB)
	nC := p0
	dB := c.w - 1
	dB = dB + dB
	dA := 0 - dB
	numer := skPoint{mul32(mul32(nA.x, t)+nB.x, t) + nC.x, mul32(mul32(nA.y, t)+nB.y, t) + nC.y}
	denom := mul32(mul32(dA, t)+dB, t) + 1
	return skPoint{numer.x / denom, numer.y / denom}
}

// evalTangentAt is SkConic::evalTangentAt.
func (c skConic) evalTangentAt(t float32) skPoint {
	if (t == 0 && c.pts[0] == c.pts[1]) || (t == 1 && c.pts[1] == c.pts[2]) {
		return c.pts[2].sub(c.pts[0])
	}
	p20 := c.pts[2].sub(c.pts[0])
	p10 := c.pts[1].sub(c.pts[0])
	C := skPoint{mul32(c.w, p10.x), mul32(c.w, p10.y)}
	A := skPoint{mul32(c.w, p20.x) - p20.x, mul32(c.w, p20.y) - p20.y}
	B := p20.sub(C).sub(C)
	return skPoint{mul32(mul32(A.x, t)+B.x, t) + C.x, mul32(mul32(A.y, t)+B.y, t) + C.y}
}

// p3dInterp is SkGeometry.cpp's p3d_interp on one coordinate.
func p3dInterp(s0, s3, s6, t float32) (float32, float32, float32) {
	ab := skScalarInterp(s0, s3, t)
	bc := skScalarInterp(s3, s6, t)
	return ab, skScalarInterp(ab, bc, t), bc
}

// chopAt is SkConic::chopAt(t, dst[2]); false if a value isn't finite.
func (c skConic) chopAt(t float32) ([2]skConic, bool) {
	// ratquad_mapTo3D: (x*1, y*1, 1), (x*w, y*w, w), (x*1, y*1, 1).
	x0, x3, x6 := p3dInterp(c.pts[0].x, mul32(c.pts[1].x, c.w), c.pts[2].x, t)
	y0, y3, y6 := p3dInterp(c.pts[0].y, mul32(c.pts[1].y, c.w), c.pts[2].y, t)
	z0, z3, z6 := p3dInterp(1, c.w, 1, t)
	var dst [2]skConic
	dst[0].pts[0] = c.pts[0]
	dst[0].pts[1] = skPoint{x0 / z0, y0 / z0}
	dst[0].pts[2] = skPoint{x3 / z3, y3 / z3}
	dst[1].pts[0] = dst[0].pts[2]
	dst[1].pts[1] = skPoint{x6 / z6, y6 / z6}
	dst[1].pts[2] = c.pts[2]
	root := sqrtf(z3)
	dst[0].w = z0 / root
	dst[1].w = z6 / root
	for _, d := range dst {
		for _, p := range d.pts {
			if !p.isFinite() {
				return dst, false
			}
		}
		if !isFinite32(d.w) {
			return dst, false
		}
	}
	return dst, true
}

// chopAt2 is SkConic::chopAt(t1, t2, dst).
func (c skConic) chopAt2(t1, t2 float32) skConic {
	if t1 == 0 || t2 == 1 {
		if t1 == 0 && t2 == 1 {
			return c
		}
		t := t2
		if t1 != 0 {
			t = t1
		}
		if pair, ok := c.chopAt(t); ok {
			if t1 != 0 {
				return pair[1]
			}
			return pair[0]
		}
	}
	p0, p1, p2 := c.pts[0], c.pts[1], c.pts[2]
	p1w := p1.scale(c.w)
	nA := p2.sub(p1w.add(p1w)).add(p0)
	nB := p1w.sub(p0)
	nB = nB.add(nB)
	dB := c.w - 1
	dB = dB + dB
	dA := 0 - dB
	numer := func(t float32) skPoint {
		return skPoint{mul32(mul32(nA.x, t)+nB.x, t) + p0.x, mul32(mul32(nA.y, t)+nB.y, t) + p0.y}
	}
	denom := func(t float32) float32 { return mul32(mul32(dA, t)+dB, t) + 1 }
	aXY, aZZ := numer(t1), denom(t1)
	mid := (t1 + t2) / 2
	dXY, dZZ := numer(mid), denom(mid)
	cXY, cZZ := numer(t2), denom(t2)
	bXY := skPoint{(dXY.x + dXY.x) - mul32(aXY.x+cXY.x, 0.5), (dXY.y + dXY.y) - mul32(aXY.y+cXY.y, 0.5)}
	bZZ := (dZZ + dZZ) - mul32(aZZ+cZZ, 0.5)
	var dst skConic
	dst.pts[0] = skPoint{aXY.x / aZZ, aXY.y / aZZ}
	dst.pts[1] = skPoint{bXY.x / bZZ, bXY.y / bZZ}
	dst.pts[2] = skPoint{cXY.x / cZZ, cXY.y / cZZ}
	dst.w = bZZ / sqrtf(mul32(aZZ, cZZ))
	return dst
}

// chop is SkConic::chop (the non-legacy version Flutter builds).
func (c skConic) chop() [2]skConic {
	scale := 1 / (1 + c.w)
	t0 := c.pts[0].scale(scale)
	t1 := c.pts[1].scale(mul32(c.w, scale))
	t2 := c.pts[2].scale(scale)
	p1 := t0.add(t1)
	p3 := t1.add(t2)
	p2 := skPoint{(mul32(0.5, t0.x) + t1.x) + mul32(0.5, t2.x), (mul32(0.5, t0.y) + t1.y) + mul32(0.5, t2.y)}
	w := sqrtf(fma32(c.w, 0.5, 0.5))
	return [2]skConic{{pts: [3]skPoint{c.pts[0], p1, p2}, w: w}, {pts: [3]skPoint{p2, p3, c.pts[2]}, w: w}}
}

func badConicW(w float32) bool { return w < 0 || !isFinite32(w) }

// computeQuadPOW2 is SkConic::computeQuadPOW2.
func (c skConic) computeQuadPOW2(tol float32) int {
	if tol < 0 || !isFinite32(tol) || !c.pts[0].isFinite() || !c.pts[1].isFinite() || !c.pts[2].isFinite() ||
		badConicW(c.w) {
		return 0
	}
	a := c.w - 1
	k := a / mul32(4, 2+a)
	x := mul32(k, fma32(-2, c.pts[1].x, c.pts[0].x)+c.pts[2].x)
	y := mul32(k, fma32(-2, c.pts[1].y, c.pts[0].y)+c.pts[2].y)
	err := sqrtf(fma32(x, x, mul32(y, y)))
	pow2 := 0
	for ; pow2 < 5; pow2++ {
		if err <= tol {
			break
		}
		err = mul32(err, 0.25)
	}
	return pow2
}

func between(a, b, c float32) bool { return mul32(a-b, c-b) <= 0 }

func conicSubdivide(src skConic, pts []skPoint, level int) []skPoint {
	if level == 0 {
		pts[0], pts[1] = src.pts[1], src.pts[2]
		return pts[2:]
	}
	dst := src.chop()
	startY := src.pts[0].y
	endY := src.pts[2].y
	if between(startY, src.pts[1].y, endY) {
		midY := dst[0].pts[2].y
		if !between(startY, midY, endY) {
			closerY := endY
			if fabsf(midY-startY) < fabsf(midY-endY) {
				closerY = startY
			}
			dst[0].pts[2].y = closerY
			dst[1].pts[0].y = closerY
		}
		if !between(startY, dst[0].pts[1].y, dst[0].pts[2].y) {
			dst[0].pts[1].y = startY
		}
		if !between(dst[1].pts[0].y, dst[1].pts[1].y, endY) {
			dst[1].pts[1].y = endY
		}
	}
	level--
	pts = conicSubdivide(dst[0], pts, level)
	return conicSubdivide(dst[1], pts, level)
}

// chopIntoQuadsPOW2 is SkConic::chopIntoQuadsPOW2: 1 + 2*(1<<pow2) points.
func (c skConic) chopIntoQuadsPOW2(pow2 int) []skPoint {
	if badConicW(c.w) {
		pow2 = 0
	}
	pts := make([]skPoint, 1+2*(1<<pow2))
	pts[0] = c.pts[0]
	done := false
	if pow2 == 5 {
		dst := c.chop()
		if skEqualsWithinTolerance(dst[0].pts[1], dst[0].pts[2]) && skEqualsWithinTolerance(dst[1].pts[0], dst[1].pts[1]) {
			pts = pts[:5]
			pts[1], pts[2], pts[3] = dst[0].pts[1], dst[0].pts[1], dst[0].pts[1]
			pts[4] = dst[1].pts[2]
			pow2 = 1
			done = true
		}
	}
	if !done {
		conicSubdivide(c, pts[1:], pow2)
	}
	for _, p := range pts {
		if !p.isFinite() {
			for i := 1; i < len(pts)-1; i++ {
				pts[i] = c.pts[1]
			}
			break
		}
	}
	return pts
}

// conicToQuads is SkAutoConicToQuads::computeQuads(pts, weight, tol): the
// quads share end points (quad i is pts[2i..2i+2]).
func conicToQuads(p0, p1, p2 skPoint, weight, tol float32) []skPoint {
	if weight <= 0 {
		return []skPoint{p0, {mul32(p0.x+p2.x, 0.5), mul32(p0.y+p2.y, 0.5)}, p2}
	}
	c := newConic(p0, p1, p2, weight)
	return c.chopIntoQuadsPOW2(c.computeQuadPOW2(tol))
}

// findYExtrema is SkConic::findYExtrema (conic_find_extrema).
func (c skConic) findYExtrema() (float32, bool) {
	p20 := c.pts[2].y - c.pts[0].y
	p10 := c.pts[1].y - c.pts[0].y
	wP10 := mul32(c.w, p10)
	c0 := fma32(c.w, p20, -p20)
	c1 := fma32(-2, wP10, p20)
	roots := skFindUnitQuadRoots(c0, c1, wP10)
	if len(roots) == 1 {
		return roots[0], true
	}
	return 0, false
}

// chopAtYExtrema is SkConic::chopAtYExtrema.
func (c skConic) chopAtYExtrema() ([2]skConic, bool) {
	t, ok := c.findYExtrema()
	if !ok {
		return [2]skConic{}, false
	}
	dst, ok := c.chopAt(t)
	if !ok {
		return dst, false
	}
	v := dst[0].pts[2].y
	dst[0].pts[1].y = v
	dst[1].pts[0].y = v
	dst[1].pts[1].y = v
	return dst, true
}

// conicTransformW is SkConic::TransformW.
func conicTransformW(pts []skPoint, w float32, m *skMatrix) float32 {
	if !m.hasPerspective() {
		return w
	}
	d0 := m.mapHomogeneous([3]float32{mul32(pts[0].x, 1), mul32(pts[0].y, 1), 1})
	d1 := m.mapHomogeneous([3]float32{mul32(pts[1].x, w), mul32(pts[1].y, w), w})
	d2 := m.mapHomogeneous([3]float32{mul32(pts[2].x, 1), mul32(pts[2].y, 1), 1})
	w0, w1, w2 := float64(d0[2]), float64(d1[2]), float64(d2[2])
	return float32(math.Sqrt(float64(w1*w1) / float64(w0*w2)))
}

// buildUnitArc is SkConic::BuildUnitArc (dir: true for clockwise).
func buildUnitArc(uStart, uStop skPoint, cw bool, user *skMatrix) []skConic {
	x := skDot(uStart, uStop)
	y := skCross(uStart, uStop)
	absY := fabsf(y)
	if absY <= skNearlyZero && x > 0 && ((y >= 0 && cw) || (y <= 0 && !cw)) {
		return nil
	}
	if !cw {
		y = -y
	}
	quadrant := 0
	switch {
	case y == 0:
		quadrant = 2
	case x == 0:
		if y > 0 {
			quadrant = 1
		} else {
			quadrant = 3
		}
	default:
		if y < 0 {
			quadrant += 2
		}
		if (x < 0) != (y < 0) {
			quadrant++
		}
	}
	quadrantPts := [8]skPoint{{1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}}
	dst := make([]skConic, 0, 5)
	for i := 0; i < quadrant; i++ {
		dst = append(dst, newConic(quadrantPts[i*2], quadrantPts[i*2+1], quadrantPts[(i*2+2)%8], skScalarRoot2Over2))
	}
	finalP := skPoint{x, y}
	lastQ := quadrantPts[quadrant*2]
	dot := skDot(lastQ, finalP)
	if dot != dot {
		return nil
	}
	if dot < 1 {
		offCurve := skPoint{lastQ.x + x, lastQ.y + y}
		cosThetaOver2 := sqrtf((1 + dot) / 2)
		offCurve, _ = skSetLength(offCurve.x, offCurve.y, 1/cosThetaOver2)
		if !skEqualsWithinTolerance(lastQ, offCurve) {
			dst = append(dst, newConic(lastQ, offCurve, finalP, cosThetaOver2))
		}
	}
	var m skMatrix
	m.setSinCos(uStart.y, uStart.x)
	if !cw {
		m.preScale(1, -1)
	}
	if user != nil {
		m.postConcat(user)
	}
	for i := range dst {
		m.mapPoints(dst[i].pts[:], dst[i].pts[:])
	}
	return dst
}
