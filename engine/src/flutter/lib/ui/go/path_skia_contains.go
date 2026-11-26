package ui

// SkPathBuilder::contains: SkPathPriv::Contains (src/core/SkPathPriv.cpp:
// winding numbers of lines, quads, conics and cubics, on-curve counting and
// the coincident-tangent check) over SkPathEdgeIter (src/core/SkPathPriv.h,
// SkPath.cpp), which closes every contour with a line.

type edge struct {
	verb uint8
	pts  []skPoint
	w    float32
}

// edges is SkPathEdgeIter: every segment, plus the closing line of each
// contour that has segments.
func edges(pts []skPoint, verbs []uint8, weights []float32) []edge {
	var out []edge
	pi, wi := 0, 0
	moveTo := 0
	needsClose := false
	closeLine := func() {
		out = append(out, edge{verbLine, []skPoint{pts[pi-1], pts[moveTo]}, 0})
		needsClose = false
	}
	for _, v := range verbs {
		switch v {
		case verbMove:
			if needsClose {
				closeLine()
			}
			moveTo = pi
			pi++
		case verbClose:
			if needsClose {
				closeLine()
			}
		default:
			n := ptsAdvance[v]
			var w float32
			if v == verbConic {
				w = weights[wi]
				wi++
			}
			needsClose = true
			out = append(out, edge{v, pts[pi-1 : pi+n], w})
			pi += n
		}
	}
	if needsClose {
		closeLine()
	}
	return out
}

func polyEval3(a, b, c, t float32) float32    { return fma32(fma32(a, t, b), t, c) }
func polyEval4(a, b, c, d, t float32) float32 { return fma32(fma32(fma32(a, t, b), t, c), t, d) }

func evalCubicPts(c0, c1, c2, c3, t float32) float32 {
	A := fma32(3, c1-c2, c3) - c0
	B := mul32(3, ((c2-c1)-c1)+c0)
	C := mul32(3, c1-c0)
	return polyEval4(A, B, C, c0, t)
}

func checkOnCurve(x, y float32, start, end skPoint) bool {
	if start.y == end.y {
		return between(start.x, x, end.x) && x != end.x
	}
	return x == start.x && y == start.y
}

func windingMonoCubic(pts []skPoint, x, y float32, onCurve *int) int {
	y0, y3 := pts[0].y, pts[3].y
	dir := 1
	if y0 > y3 {
		y0, y3 = y3, y0
		dir = -1
	}
	if y < y0 || y > y3 {
		return 0
	}
	if checkOnCurve(x, y, pts[0], pts[3]) {
		*onCurve++
		return 0
	}
	if y == y3 {
		return 0
	}
	mn, mx := pts[0].x, pts[0].x
	for i := 1; i < 4; i++ {
		mn = stdMin(mn, pts[i].x)
		mx = stdMax(mx, pts[i].x)
	}
	if x < mn {
		return 0
	}
	if x > mx {
		return dir
	}
	t, ok := chopMonoCubicAtY(pts, y)
	if !ok {
		return 0
	}
	xt := evalCubicPts(pts[0].x, pts[1].x, pts[2].x, pts[3].x, t)
	if skScalarNearlyEqual(xt, x) {
		if x != pts[3].x || y != pts[3].y {
			*onCurve++
			return 0
		}
	}
	if xt < x {
		return dir
	}
	return 0
}

func windingCubic(pts []skPoint, x, y float32, onCurve *int) int {
	n, dst := chopCubicAtYExtrema(pts)
	w := 0
	for i := 0; i <= n; i++ {
		w += windingMonoCubic(dst[i*3:i*3+4], x, y, onCurve)
	}
	return w
}

func conicEvalNumerator(s0, s2, s4, w, t float32) float64 {
	src2w := mul32(s2, w)
	C := s0
	A := fma32(-2, src2w, s4) + C
	B := mul32(2, src2w-C)
	return float64(polyEval3(A, B, C, t))
}

func conicEvalDenominator(w, t float32) float64 {
	B := mul32(2, w-1)
	return float64(polyEval3(-B, B, 1, t))
}

// conicYRoots solves the conic's y(t) = y (winding_mono_conic/tangent_conic).
func conicYRoots(pts []skPoint, w, y float32) []float32 {
	A := pts[2].y
	B := fma32(pts[1].y, w, -mul32(y, w)) + y
	C := pts[0].y
	A += fma32(-2, B, C)
	B -= C
	C -= y
	return skFindUnitQuadRoots(A, mul32(2, B), C)
}

func windingMonoConic(c skConic, x, y float32, onCurve *int) int {
	pts := c.pts[:]
	y0, y2 := pts[0].y, pts[2].y
	dir := 1
	if y0 > y2 {
		y0, y2 = y2, y0
		dir = -1
	}
	if y < y0 || y > y2 {
		return 0
	}
	if checkOnCurve(x, y, pts[0], pts[2]) {
		*onCurve++
		return 0
	}
	if y == y2 {
		return 0
	}
	roots := conicYRoots(pts, c.w, y)
	var xt float32
	if len(roots) == 0 {
		xt = pts[1-dir].x
	} else {
		t := roots[0]
		xt = float32(conicEvalNumerator(pts[0].x, pts[1].x, pts[2].x, c.w, t) / conicEvalDenominator(c.w, t))
	}
	if skScalarNearlyEqual(xt, x) {
		if x != pts[2].x || y != pts[2].y {
			*onCurve++
			return 0
		}
	}
	if xt < x {
		return dir
	}
	return 0
}

func isMonoQuad(y0, y1, y2 float32) bool {
	if y0 == y1 {
		return true
	}
	if y0 < y1 {
		return y1 <= y2
	}
	return y1 >= y2
}

func windingConic(pts []skPoint, x, y, weight float32, onCurve *int) int {
	c := newConic(pts[0], pts[1], pts[2], weight)
	isMono := isMonoQuad(pts[0].y, pts[1].y, pts[2].y)
	var chopped [2]skConic
	if !isMono {
		var ok bool
		chopped, ok = c.chopAtYExtrema()
		isMono = !ok
	}
	if isMono {
		return windingMonoConic(c, x, y, onCurve)
	}
	return windingMonoConic(chopped[0], x, y, onCurve) + windingMonoConic(chopped[1], x, y, onCurve)
}

func quadYRoots(pts []skPoint, y float32) []float32 {
	return skFindUnitQuadRoots(fma32(-2, pts[1].y, pts[0].y)+pts[2].y, mul32(2, pts[1].y-pts[0].y), pts[0].y-y)
}

func quadXAt(pts []skPoint, t float32) float32 {
	C := pts[0].x
	A := fma32(-2, pts[1].x, pts[2].x) + C
	B := mul32(2, pts[1].x-C)
	return polyEval3(A, B, C, t)
}

func windingMonoQuad(pts []skPoint, x, y float32, onCurve *int) int {
	y0, y2 := pts[0].y, pts[2].y
	dir := 1
	if y0 > y2 {
		y0, y2 = y2, y0
		dir = -1
	}
	if y < y0 || y > y2 {
		return 0
	}
	if checkOnCurve(x, y, pts[0], pts[2]) {
		*onCurve++
		return 0
	}
	if y == y2 {
		return 0
	}
	roots := quadYRoots(pts, y)
	var xt float32
	if len(roots) == 0 {
		xt = pts[1-dir].x
	} else {
		xt = quadXAt(pts, roots[0])
	}
	if skScalarNearlyEqual(xt, x) {
		if x != pts[2].x || y != pts[2].y {
			*onCurve++
			return 0
		}
	}
	if xt < x {
		return dir
	}
	return 0
}

func windingQuad(pts []skPoint, x, y float32, onCurve *int) int {
	span := pts
	n := 0
	if !isMonoQuad(pts[0].y, pts[1].y, pts[2].y) {
		var dst [5]skPoint
		n, dst = chopQuadAtYExtrema(pts)
		span = dst[:]
	}
	w := windingMonoQuad(span[:3], x, y, onCurve)
	if n > 0 {
		w += windingMonoQuad(span[2:5], x, y, onCurve)
	}
	return w
}

func windingLine(pts []skPoint, x, y float32, onCurve *int) int {
	x0, y0, x1, y1 := pts[0].x, pts[0].y, pts[1].x, pts[1].y
	dy := y1 - y0
	dir := 1
	if y0 > y1 {
		y0, y1 = y1, y0
		dir = -1
	}
	if y < y0 || y > y1 {
		return 0
	}
	if checkOnCurve(x, y, pts[0], pts[1]) {
		*onCurve++
		return 0
	}
	if y == y1 {
		return 0
	}
	cross := fma32(x1-x0, y-pts[0].y, -mul32(dy, x-x0))
	if cross == 0 {
		if x != x1 || y != pts[1].y {
			*onCurve++
		}
		dir = 0
	} else if skScalarSignAsInt(cross) == dir {
		dir = 0
	}
	return dir
}

func tangentCubic(pts []skPoint, x, y float32, tangents *[]skPoint) {
	if !between(pts[0].y, y, pts[1].y) && !between(pts[1].y, y, pts[2].y) && !between(pts[2].y, y, pts[3].y) {
		return
	}
	if !between(pts[0].x, x, pts[1].x) && !between(pts[1].x, x, pts[2].x) && !between(pts[2].x, x, pts[3].x) {
		return
	}
	n, dst := chopCubicAtYExtrema(pts)
	for i := 0; i <= n; i++ {
		c := dst[i*3 : i*3+4]
		t, ok := chopMonoCubicAtY(c, y)
		if !ok {
			continue
		}
		xt := evalCubicPts(c[0].x, c[1].x, c[2].x, c[3].x, t)
		if !skScalarNearlyEqual(x, xt) {
			continue
		}
		*tangents = append(*tangents, cubicTangentAt(c, t))
	}
}

func tangentConic(pts []skPoint, x, y, w float32, tangents *[]skPoint) {
	if !between(pts[0].y, y, pts[1].y) && !between(pts[1].y, y, pts[2].y) {
		return
	}
	if !between(pts[0].x, x, pts[1].x) && !between(pts[1].x, x, pts[2].x) {
		return
	}
	for _, t := range conicYRoots(pts, w, y) {
		xt := float32(conicEvalNumerator(pts[0].x, pts[1].x, pts[2].x, w, t) / conicEvalDenominator(w, t))
		if !skScalarNearlyEqual(x, xt) {
			continue
		}
		c := newConic(pts[0], pts[1], pts[2], w)
		*tangents = append(*tangents, c.evalTangentAt(t))
	}
}

func tangentQuad(pts []skPoint, x, y float32, tangents *[]skPoint) {
	if !between(pts[0].y, y, pts[1].y) && !between(pts[1].y, y, pts[2].y) {
		return
	}
	if !between(pts[0].x, x, pts[1].x) && !between(pts[1].x, x, pts[2].x) {
		return
	}
	for _, t := range quadYRoots(pts, y) {
		xt := quadXAt(pts, t)
		if !skScalarNearlyEqual(x, xt) {
			continue
		}
		*tangents = append(*tangents, quadTangentAt(pts, t))
	}
}

func tangentLine(pts []skPoint, x, y float32, tangents *[]skPoint) {
	y0, y1 := pts[0].y, pts[1].y
	if !between(y0, y, y1) {
		return
	}
	x0, x1 := pts[0].x, pts[1].x
	if !between(x0, x, x1) {
		return
	}
	dx, dy := x1-x0, y1-y0
	if !skScalarNearlyEqual(mul32(x-x0, dy), mul32(dx, y-y0)) {
		return
	}
	*tangents = append(*tangents, skPoint{dx, dy})
}

// contains is SkPathBuilder::contains(p).
func (b *skPathBuilder) contains(p skPoint) bool {
	bounds, ok := b.computeFiniteBounds()
	if !ok {
		return false
	}
	if len(b.verbs) == 0 {
		return false
	}
	if !(bounds.l <= p.x && p.x <= bounds.r && bounds.t <= p.y && p.y <= bounds.b) {
		return false
	}
	es := edges(b.pts, b.verbs, b.weights)
	w, onCurve := 0, 0
	for _, e := range es {
		switch e.verb {
		case verbLine:
			w += windingLine(e.pts, p.x, p.y, &onCurve)
		case verbQuad:
			w += windingQuad(e.pts, p.x, p.y, &onCurve)
		case verbConic:
			w += windingConic(e.pts, p.x, p.y, e.w, &onCurve)
		case verbCubic:
			w += windingCubic(e.pts, p.x, p.y, &onCurve)
		}
	}
	evenOdd := b.fillType == fillEvenOdd
	if evenOdd {
		w &= 1
	}
	if w != 0 {
		return true
	}
	if onCurve <= 1 {
		return onCurve != 0
	}
	if onCurve&1 != 0 || evenOdd {
		return onCurve&1 != 0
	}
	var tangents []skPoint
	for _, e := range es {
		old := len(tangents)
		switch e.verb {
		case verbLine:
			tangentLine(e.pts, p.x, p.y, &tangents)
		case verbQuad:
			tangentQuad(e.pts, p.x, p.y, &tangents)
		case verbConic:
			tangentConic(e.pts, p.x, p.y, e.w, &tangents)
		case verbCubic:
			tangentCubic(e.pts, p.x, p.y, &tangents)
		}
		if len(tangents) > old {
			last := len(tangents) - 1
			tangent := tangents[last]
			if skScalarNearlyZero(skDot(tangent, tangent)) {
				tangents = tangents[:last]
			} else {
				for index := 0; index < last; index++ {
					test := tangents[index]
					if skScalarNearlyZero(skCross(test, tangent)) && skScalarSignAsInt(mul32(tangent.x, test.x)) <= 0 &&
						skScalarSignAsInt(mul32(tangent.y, test.y)) <= 0 {
						tangents = tangents[:last]
						// removeShuffle(index): the last element moves into index.
						tangents[index] = tangents[len(tangents)-1]
						tangents = tangents[:len(tangents)-1]
						break
					}
				}
			}
		}
	}
	return len(tangents) != 0
}
