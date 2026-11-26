package ui

// Path.addRRect and Path.addRSuperellipse's engine side:
//   - lib/ui/painting/rrect.cc (the Float32List RRect → DlRoundRect) and
//     rsuperellipse.cc (BuildBounds/BuildRadii);
//   - impeller/geometry/round_rect.cc and round_superellipse.cc
//     (MakeRectRadii), rounding_radii.cc (Scaled), rect.h (GetPositive);
//   - display_list/geometry/dl_geometry_conversions.h (ToSkRRect) and
//     dl_path_builder.cc (AddRoundSuperellipse, ConicCurveTo);
//   - impeller/geometry/round_superellipse_param.cc (MakeBoundsRadii and
//     Dispatch: the conics and cubics of a rounded superellipse).
// rsuperellipse_contains.go ports the same parameter math for
// RSuperellipse.contains; this port keeps every fused multiply-add and
// float libm call (powf, asinf, tanf, atan2f) of the engine, which the
// path's coordinates expose bit for bit.

import "math"

type isize struct{ w, h float32 }

func (s isize) isEmpty() bool  { return !(s.w > 0 && s.h > 0) }
func (s isize) isFinite() bool { return isFinite32(s.w) && isFinite32(s.h) }

// roundingRadii is impeller::RoundingRadii (TL, TR, BL, BR).
type roundingRadii struct{ tl, tr, bl, br isize }

func (r roundingRadii) allEmpty() bool {
	return r.tl.isEmpty() && r.tr.isEmpty() && r.bl.isEmpty() && r.br.isEmpty()
}
func (r roundingRadii) isFinite() bool {
	return r.tl.isFinite() && r.tr.isFinite() && r.bl.isFinite() && r.br.isFinite()
}

// positive is impeller TRect::GetPositive.
func (r skRect) positive() skRect {
	if !r.isEmpty() {
		return r
	}
	return skRect{stdMin(r.l, r.r), stdMin(r.t, r.b), stdMax(r.l, r.r), stdMax(r.t, r.b)}
}

func adjustScale(r1, r2, dim float32, scale *float32) {
	if r1+r2 > dim {
		*scale = stdMin(*scale, dim/(r1+r2))
	}
}

// scaled is RoundingRadii::Scaled.
func (r roundingRadii) scaled(inBounds skRect) roundingRadii {
	bounds := inBounds.positive()
	if bounds.isEmpty() || r.allEmpty() || !r.isFinite() {
		return roundingRadii{}
	}
	for _, s := range []*isize{&r.tl, &r.tr, &r.bl, &r.br} {
		if s.isEmpty() {
			*s = isize{}
		}
	}
	w, h := bounds.width(), bounds.height()
	scale := float32(1)
	adjustScale(r.tl.w, r.tr.w, w, &scale)
	adjustScale(r.bl.w, r.br.w, w, &scale)
	adjustScale(r.tl.h, r.bl.h, h, &scale)
	adjustScale(r.tr.h, r.br.h, h, &scale)
	if scale < 1 {
		for _, s := range []*isize{&r.tl, &r.tr, &r.bl, &r.br} {
			*s = isize{mul32(s.w, scale), mul32(s.h, scale)}
		}
	}
	return r
}

// makeRectRadii is RoundRect/RoundSuperellipse::MakeRectRadii: the
// positive bounds and the scaled radii (zero everything when the bounds
// aren't finite).
func makeRectRadii(inBounds skRect, radii roundingRadii) (skRect, roundingRadii) {
	if !inBounds.isFinite() {
		return skRect{}, roundingRadii{}
	}
	b := inBounds.positive()
	return b, radii.scaled(b)
}

// rrectToSk is rrect.cc's conversion followed by ToSkRRect: the float
// values Dart's _getValue32 stores, made positive, scaled, then
// SkRRect::setRectRadii.
func rrectToSk(v [12]float32) skRRect {
	raw := skRect{v[0], v[1], v[2], v[3]}
	radii := roundingRadii{tl: isize{v[4], v[5]}, tr: isize{v[6], v[7]}, bl: isize{v[10], v[11]}, br: isize{v[8], v[9]}}
	b, r := makeRectRadii(raw.positive(), radii)
	var rr skRRect
	rr.setRectRadii(b, [4]skPoint{{r.tl.w, r.tl.h}, {r.tr.w, r.tr.h}, {r.br.w, r.br.h}, {r.bl.w, r.bl.h}})
	return rr
}

// --- RoundSuperellipseParam ---

type rseOctant struct {
	offset                    skPoint
	seA, seN, seMaxTheta      float32
	circleStart, circleCenter skPoint
	circleMaxAngle            float32
	circleRadius              float32
}

type rseQuadrant struct {
	offset, signedScale skPoint
	top, right          rseOctant
}

type rseParam struct {
	topRight, bottomRight, bottomLeft, topLeft rseQuadrant
	allCornersSame                             bool
}

var rsePrecomputed = [11][2]float32{
	{2.00000000, 1.13276676}, {2.18349805, 1.20311921}, {2.33888662, 1.28698796}, {2.48660575, 1.36351941},
	{2.62226596, 1.44717976}, {2.75148990, 1.53385819}, {3.36298265, 1.98288283}, {4.08649929, 2.23811846},
	{4.85481134, 2.47563463}, {5.62945551, 2.72948597}, {6.43023796, 2.98020421},
}

func rseSplit(left, right, ratioLeft, ratioRight float32) float32 {
	if ratioLeft == 0 && ratioRight == 0 {
		return (left + right) / 2
	}
	return fma32(left, ratioRight, mul32(right, ratioLeft)) / (ratioLeft + ratioRight)
}

func sizeTFloor(v float32, maxIndex int) int {
	f := math.Floor(float64(v))
	if f < 0 || f > float64(maxIndex) { // size_t: negative wraps to huge
		return maxIndex
	}
	return int(f)
}

func rseComputeNAndXj(ratio float32) (float32, float32) {
	const n = len(rsePrecomputed)
	if ratio > 5 {
		nn := fma32(float32(1.559599389), ratio-5, rsePrecomputed[n-1][0])
		kxj := fma32(float32(0.522807185), ratio-5, rsePrecomputed[n-1][1])
		return nn, 1 - 1/kxj
	}
	ratio = clamp32(ratio, 2, 5)
	var steps float32
	if ratio < 2.5 {
		steps = mul32(ratio-2, 10)
	} else {
		steps = fma32(ratio-2.5, 2, 6) - 1
	}
	left := sizeTFloor(steps, n-2)
	frac := steps - float32(left)
	nn := fma32(1-frac, rsePrecomputed[left][0], mul32(frac, rsePrecomputed[left+1][0]))
	kxj := fma32(1-frac, rsePrecomputed[left][1], mul32(frac, rsePrecomputed[left+1][1]))
	return nn, 1 - 1/kxj
}

// impeller Point helpers.
func ipLength(p skPoint) float32 {
	return sqrtf(float32(float64(float64(p.x)*float64(p.x)) + float64(float64(p.y)*float64(p.y))))
}

func ipNormalize(p skPoint) skPoint {
	l := ipLength(p)
	if l == 0 {
		return skPoint{1, 0}
	}
	return skPoint{p.x / l, p.y / l}
}

func ipCross(a, b skPoint) float32 { return fma32(a.x, b.y, -mul32(a.y, b.x)) }
func ipDot(a, b skPoint) float32   { return fma32(a.x, b.x, mul32(a.y, b.y)) }

func ipRotate(p skPoint, rad float32) skPoint {
	s, c := sincosf(rad)
	return skPoint{fma32(p.x, c, -mul32(p.y, s)), fma32(p.x, s, mul32(p.y, c))}
}

func rseFindCircleCenter(a, b skPoint, r float32) skPoint {
	aToB := b.sub(a)
	m := a.add(b)
	m = skPoint{m.x / 2, m.y / 2}
	cToM := skPoint{-aToB.y, aToB.x}
	distAM := ipLength(aToB) / 2
	distCM := sqrtf(fma32(r, r, -mul32(distAM, distAM)))
	n := ipNormalize(cToM)
	return m.sub(skPoint{mul32(distCM, n.x), mul32(distCM, n.y)})
}

const rseCloseEnough = float32(1e-3)

func rseComputeOctant(center skPoint, a, radius float32) rseOctant {
	if radius <= rseCloseEnough {
		return rseOctant{offset: center, seA: a, circleStart: skPoint{a, a}}
	}
	ratio := mul32(a, 2) / radius
	g := mul32(float32(0.29289321881), radius)
	n, xjRatio := rseComputeNAndXj(ratio)
	xJ := mul32(xjRatio, a)
	yJ := mul32(powf(1-powf(xjRatio, n), 1/n), a)
	maxTheta := asinf(powf(xjRatio, n/2))
	tanPhiJ := powf(xJ/yJ, n-1)
	d := fma32(-tanPhiJ, yJ, xJ) / (1 - tanPhiJ)
	R := float32(float64(a-d-g) * math.Sqrt(2))
	pointM := skPoint{a - g, a - g}
	pointJ := skPoint{xJ, yJ}
	cc := rseFindCircleCenter(pointJ, pointM, R)
	u, v := pointM.sub(cc), pointJ.sub(cc)
	maxAngle := atan2f(ipCross(u, v), ipDot(u, v))
	return rseOctant{offset: center, seA: a, seN: n, seMaxTheta: maxTheta, circleStart: pointJ,
		circleCenter: cc, circleMaxAngle: maxAngle, circleRadius: R}
}

func rseComputeQuadrant(center, corner skPoint, in isize, sign skPoint) rseQuadrant {
	cv := corner.sub(center)
	radii := isize{stdMin(fabsf(in.w), fabsf(cv.x)), stdMin(fabsf(in.h), fabsf(cv.y))}
	normRadius := stdMin(radii.w, radii.h)
	fwd := isize{1, 1}
	if normRadius != 0 {
		fwd = isize{radii.w / normRadius, radii.h / normRadius}
	}
	nhs := skPoint{fabsf(cv.x) / fwd.w, fabsf(cv.y) / fwd.h}
	ss := skPoint{cv.x / nhs.x, cv.y / nhs.y}
	if ss.x != ss.x {
		ss.x = sign.x
	}
	if ss.y != ss.y {
		ss.y = sign.y
	}
	c := nhs.x - nhs.y
	return rseQuadrant{offset: center, signedScale: ss,
		top:   rseComputeOctant(skPoint{0, -c}, nhs.x, normRadius),
		right: rseComputeOctant(skPoint{c, 0}, nhs.y, normRadius)}
}

func nearlyEqualTol(a, b, tol float32) bool { return fabsf(a-b) <= tol }

// makeRSEParam is RoundSuperellipseParam::MakeBoundsRadii.
func makeRSEParam(b skRect, r roundingRadii) rseParam {
	same := nearlyEqualTol(r.tl.w, r.tr.w, rseCloseEnough) && nearlyEqualTol(r.tl.w, r.br.w, rseCloseEnough) &&
		nearlyEqualTol(r.tl.w, r.bl.w, rseCloseEnough) && nearlyEqualTol(r.tl.h, r.tr.h, rseCloseEnough) &&
		nearlyEqualTol(r.tl.h, r.br.h, rseCloseEnough) && nearlyEqualTol(r.tl.h, r.bl.h, rseCloseEnough)
	center := skPoint{avgScalar(b.l, b.r), avgScalar(b.t, b.b)}
	if same && !r.tl.isEmpty() {
		return rseParam{topRight: rseComputeQuadrant(center, skPoint{b.r, b.t}, r.tr, skPoint{-1, 1}), allCornersSame: true}
	}
	top := rseSplit(b.l, b.r, r.tl.w, r.tr.w)
	right := rseSplit(b.t, b.b, r.tr.h, r.br.h)
	bottom := rseSplit(b.l, b.r, r.bl.w, r.br.w)
	left := rseSplit(b.t, b.b, r.tl.h, r.bl.h)
	return rseParam{
		topRight:    rseComputeQuadrant(skPoint{top, right}, skPoint{b.r, b.t}, r.tr, skPoint{1, -1}),
		bottomRight: rseComputeQuadrant(skPoint{bottom, right}, skPoint{b.r, b.b}, r.br, skPoint{1, 1}),
		bottomLeft:  rseComputeQuadrant(skPoint{bottom, left}, skPoint{b.l, b.b}, r.bl, skPoint{-1, 1}),
		topLeft:     rseComputeQuadrant(skPoint{top, left}, skPoint{b.l, b.t}, r.tl, skPoint{-1, -1}),
	}
}

// avgScalar is saturated::AverageScalar for floats:
// (float)(scalbn(a, -1) + scalbn(b, -1)), computed in double.
func avgScalar(a, b float32) float32 {
	return float32(math.Ldexp(float64(a), -1) + math.Ldexp(float64(b), -1))
}

// imat is an impeller::Matrix (column-major, 16 floats).
type imat [16]float32

// imatTranslateScale is Matrix::MakeTranslateScale(s, t) with 2D points
// (Vector3(Point) has z = 0).
func imatTranslateScale(s, t skPoint) imat {
	return imat{s.x, 0, 0, 0, 0, s.y, 0, 0, 0, 0, 0, 0, t.x, t.y, 0, 1}
}

// imatTranslation is Matrix::MakeTranslation(Vector3(t)).
func imatTranslation(t skPoint) imat {
	return imat{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, t.x, t.y, 0, 1}
}

var rseFlip = imat{0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}

// mul is Matrix::Multiply: each entry a[i]*o[0] + a[4+i]*o[1] + a[8+i]*o[2]
// + a[12+i]*o[3], fused left to right (the binary's fmla chain).
func (a imat) mul(o imat) imat {
	var r imat
	for j := 0; j < 4; j++ {
		for i := 0; i < 4; i++ {
			r[j*4+i] = fma32(a[12+i], o[j*4+3], fma32(a[8+i], o[j*4+2], fma32(a[i], o[j*4], mul32(a[4+i], o[j*4+1]))))
		}
	}
	return r
}

// apply is Matrix * Point.
func (a imat) apply(p skPoint) skPoint {
	w := fma32(p.x, a[3], mul32(p.y, a[7])) + a[15]
	x := fma32(p.x, a[0], mul32(p.y, a[4])) + a[12]
	y := fma32(p.x, a[1], mul32(p.y, a[5])) + a[13]
	if w != 0 {
		w = 1 / w
	}
	return skPoint{mul32(x, w), mul32(y, w)}
}

// rseSink receives the superellipse outline (impeller::PathReceiver).
type rseSink interface {
	moveTo(p skPoint)
	lineTo(p skPoint)
	conicTo(cp, p skPoint, w float32)
	cubicTo(cp1, cp2, p skPoint)
	close()
}

type rseConic struct {
	p1, c, p2 skPoint
	w         float32
}

var rseBezierFactors = [13][2]float32{
	{0.7078, 8.3194}, {0.7895, 2.4523}, {0.8379, 1.8528}, {0.8701, 1.6891}, {0.8932, 1.5806},
	{0.9107, 1.5043}, {0.9244, 1.4470}, {0.9355, 1.4037}, {0.9448, 1.3701}, {0.9526, 1.3431},
	{0.9594, 1.3212}, {0.9653, 1.3032}, {0.9705, 1.2880},
}

func rseSuperellipseBezierFactors(n, xJOverA, yJOverA float32) (float32, float32, float32) {
	const records = len(rseBezierFactors)
	maxN := float32(2 + (records - 1))
	if n >= maxN {
		n = maxN
	}
	steps := clamp32((n-2)/1, 0, float32(records-1))
	left := sizeTFloor(steps, records-2)
	frac := steps - float32(left)
	sn := sqrtf(n)
	w1 := fma32(1-frac, rseBezierFactors[left][0], mul32(mul32(frac, rseBezierFactors[left+1][0]), sn))
	w2 := fma32(1-frac, rseBezierFactors[left][1], mul32(mul32(frac, rseBezierFactors[left+1][1]), xJOverA))
	yHp := sqrtf(n)
	yH := fma32(1, yHp, yJOverA) / (yHp + 1)
	return w1, w2, yH
}

func rseIntersection(p1 skPoint, k1 float32, p2 skPoint, k2 float32) skPoint {
	if fabsf(k1-k2) < rseCloseEnough {
		s := p1.add(p2)
		return skPoint{s.x / 2, s.y / 2}
	}
	x := ((fma32(k1, p1.x, -mul32(k2, p2.x)) + p2.y) - p1.y) / (k1 - k2)
	y := fma32(k1, x-p1.x, p1.y)
	return skPoint{x, y}
}

func rseSuperellipseArc(o rseOctant) (rseConic, rseConic) {
	A := skPoint{0, o.seA}
	J := o.circleStart
	w1, w2, yHOverA := rseSuperellipseBezierFactors(o.seN, J.x/o.seA, J.y/o.seA)
	H := skPoint{mul32(powf(1-powf(yHOverA, o.seN), 1/o.seN), o.seA), mul32(yHOverA, o.seA)}
	var kA float32
	kJ := -powf(J.x/J.y, o.seN-1)
	kH := -powf(H.x/H.y, o.seN-1)
	return rseConic{A, rseIntersection(A, kA, H, kH), H, w1}, rseConic{H, rseIntersection(H, kH, J, kJ), J, w2}
}

func rseCircularArc(o rseOctant) [4]skPoint {
	sv := o.circleStart.sub(o.circleCenter)
	ev := ipRotate(sv, -o.circleMaxAngle)
	circleEnd := o.circleCenter.add(ev)
	st := ipNormalize(skPoint{sv.y, -sv.x})
	et := ipNormalize(skPoint{-ev.y, ev.x})
	bf := mul32(tanf(o.circleMaxAngle/4), 4) / 3
	radius := ipLength(sv)
	return [4]skPoint{
		o.circleStart,
		o.circleStart.add(skPoint{mul32(mul32(st.x, bf), radius), mul32(mul32(st.y, bf), radius)}),
		circleEnd.add(skPoint{mul32(mul32(et.x, bf), radius), mul32(mul32(et.y, bf), radius)}),
		circleEnd,
	}
}

func rseAddOctant(s rseSink, o rseOctant, reverse, flip bool, ext imat) {
	t := ext.mul(imatTranslation(o.offset))
	if flip {
		t = t.mul(rseFlip)
	}
	cp := rseCircularArc(o)
	c1, c2 := rseSuperellipseArc(o)
	if !reverse {
		s.conicTo(t.apply(c1.c), t.apply(c1.p2), c1.w)
		s.conicTo(t.apply(c2.c), t.apply(c2.p2), c2.w)
		s.cubicTo(t.apply(cp[1]), t.apply(cp[2]), t.apply(cp[3]))
	} else {
		s.cubicTo(t.apply(cp[2]), t.apply(cp[1]), t.apply(cp[0]))
		s.conicTo(t.apply(c2.c), t.apply(c2.p1), c2.w)
		s.conicTo(t.apply(c1.c), t.apply(c1.p1), c1.w)
	}
}

func rseAddQuadrant(s rseSink, q rseQuadrant, reverse bool, scaleSign skPoint) {
	t := imatTranslateScale(skPoint{mul32(q.signedScale.x, scaleSign.x), mul32(q.signedScale.y, scaleSign.y)}, q.offset)
	if q.top.seN < 2 || q.right.seN < 2 {
		s.lineTo(t.apply(q.top.offset.add(skPoint{q.top.seA, q.top.seA})))
		if !reverse {
			s.lineTo(t.apply(q.right.offset.add(skPoint{q.right.seA, 0})))
		} else {
			s.lineTo(t.apply(q.top.offset.add(skPoint{0, q.top.seA})))
		}
		return
	}
	if !reverse {
		rseAddOctant(s, q.top, false, false, t)
		rseAddOctant(s, q.right, true, true, t)
	} else {
		rseAddOctant(s, q.right, false, true, t)
		rseAddOctant(s, q.top, true, false, t)
	}
}

// dispatch is RoundSuperellipseParam::Dispatch.
func (p rseParam) dispatch(s rseSink) {
	tr := p.topRight
	v := tr.top.offset.add(skPoint{0, tr.top.seA})
	start := tr.offset.add(skPoint{mul32(tr.signedScale.x, v.x), mul32(tr.signedScale.y, v.y)})
	s.moveTo(start)
	if p.allCornersSame {
		rseAddQuadrant(s, tr, false, skPoint{1, 1})
		rseAddQuadrant(s, tr, true, skPoint{1, -1})
		rseAddQuadrant(s, tr, false, skPoint{-1, -1})
		rseAddQuadrant(s, tr, true, skPoint{-1, 1})
	} else {
		rseAddQuadrant(s, p.topRight, false, skPoint{1, 1})
		rseAddQuadrant(s, p.bottomRight, true, skPoint{1, 1})
		rseAddQuadrant(s, p.bottomLeft, false, skPoint{1, 1})
		rseAddQuadrant(s, p.topLeft, true, skPoint{1, 1})
	}
	s.lineTo(start)
	s.close()
}

// dlBuilder is DlPathBuilder over an SkPathBuilder (the receiver
// AddRoundSuperellipse dispatches to).
type dlBuilder struct{ b *skPathBuilder }

func (d dlBuilder) moveTo(p skPoint) { d.b.moveTo(p) }
func (d dlBuilder) lineTo(p skPoint) { d.b.lineTo(p) }
func (d dlBuilder) conicTo(cp, p skPoint, w float32) {
	if !(w > 0) {
		d.b.lineTo(p)
	} else if !isFinite32(w) {
		d.b.lineTo(cp)
		d.b.lineTo(p)
	} else {
		d.b.conicTo(cp, p, w)
	}
}
func (d dlBuilder) cubicTo(cp1, cp2, p skPoint) { d.b.cubicTo(cp1, cp2, p) }
func (d dlBuilder) close()                      { d.b.close() }

// rsuperellipsePath is CanvasPath::addRSuperellipse's DlPathBuilder: the
// RSuperellipse's values SafeNarrow'd (BuildBounds/BuildRadii), made a
// DlRoundSuperellipse (MakeRectRadii), dispatched, and taken as an SkPath.
func rsuperellipsePath(r RSuperellipse) *skPathData {
	n := safeNarrow
	bounds := skRect{n(r.Left), n(r.Top), n(r.Right), n(r.Bottom)}.positive()
	radii := roundingRadii{
		tl: isize{n(r.TlRadiusX), n(r.TlRadiusY)}, tr: isize{n(r.TrRadiusX), n(r.TrRadiusY)},
		bl: isize{n(r.BlRadiusX), n(r.BlRadiusY)}, br: isize{n(r.BrRadiusX), n(r.BrRadiusY)},
	}
	b, rr := makeRectRadii(bounds, radii)
	builder := newSkPathBuilder()
	makeRSEParam(b, rr).dispatch(dlBuilder{&builder})
	return builder.snapshot()
}
