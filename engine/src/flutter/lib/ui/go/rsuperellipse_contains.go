package ui

// RSuperellipse.contains, which Dart implements in the engine
// (lib/ui/painting/rsuperellipse.cc → impeller/geometry/
// round_superellipse_param.cc). The prebuilt Impeller SDK doesn't expose
// it, so this is a Go port of that C++ (an ENGINE GAP filled in Go; see
// flutter_go/PLAN.md). The engine computes in float (32-bit), and so does
// this port, including the one step C++ does in double (`* sqrt(2)`).

import "math"

type f32 = float32

type point32 struct{ x, y f32 }

func (a point32) sub(b point32) point32 { return point32{a.x - b.x, a.y - b.y} }
func (a point32) add(b point32) point32 { return point32{a.x + b.x, a.y + b.y} }
func (a point32) div(b point32) point32 { return point32{a.x / b.x, a.y / b.y} }
func (a point32) scale(s f32) point32   { return point32{a.x * s, a.y * s} }
func (a point32) abs() point32          { return point32{abs32(a.x), abs32(a.y)} }
func (a point32) length() f32           { return sqrt32(float32(a.x*a.x) + float32(a.y*a.y)) }
func (a point32) distanceSquared(b point32) f32 {
	d := a.sub(b)
	return float32(d.x*d.x) + float32(d.y*d.y)
}
func (a point32) normalize() point32 {
	l := a.length()
	if l == 0 {
		return point32{1, 0}
	}
	return point32{a.x / l, a.y / l}
}
func (a point32) angleTo(b point32) f32 {
	cross := float32(a.x*b.y) - float32(a.y*b.x)
	dot := float32(a.x*b.x) + float32(a.y*b.y)
	return f32(math.Atan2(float64(cross), float64(dot)))
}

func abs32(v f32) f32    { return f32(math.Abs(float64(v))) }
func sqrt32(v f32) f32   { return f32(math.Sqrt(float64(v))) }
func pow32(x, y f32) f32 { return f32(math.Pow(float64(x), float64(y))) }
func asin32(v f32) f32   { return f32(math.Asin(float64(v))) }
func isNaN32(v f32) bool { return v != v }
func min32(a, b f32) f32 { return f32(math.Min(float64(a), float64(b))) }
func clamp32(v, lo, hi f32) f32 {
	if v < lo {
		return lo
	}
	if hi < v {
		return hi
	}
	return v
}

// safeNarrow is lib/ui/floating_point.h's SafeNarrow.
func safeNarrow(v float64) f32 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return f32(v)
	}
	return clamp32(f32(v), -math.MaxFloat32, math.MaxFloat32)
}

const (
	kEhCloseEnough f32 = 1e-3
	kGapFactor     f32 = 0.29289321881 // 1-cos(pi/4)
)

type seOctant struct {
	offset         point32
	seA, seN       f32
	seMaxTheta     f32
	circleStart    point32
	circleCenter   point32
	circleMaxAngle f32
	circleRadius   f32
}

type seQuadrant struct {
	offset, signedScale point32
	top, right          seOctant
}

type rsuperellipseParam struct {
	topRight, bottomRight, bottomLeft, topLeft seQuadrant
	allCornersSame                             bool
}

func split(left, right, ratioLeft, ratioRight f32) f32 {
	if ratioLeft == 0 && ratioRight == 0 {
		return (left + right) / 2
	}
	return (float32(left*ratioRight) + float32(right*ratioLeft)) / (ratioLeft + ratioRight)
}

var kPrecomputedVariables = [...][2]f32{
	{2.00000000, 1.13276676}, // ratio=2.00
	{2.18349805, 1.20311921}, // ratio=2.10
	{2.33888662, 1.28698796}, // ratio=2.20
	{2.48660575, 1.36351941}, // ratio=2.30
	{2.62226596, 1.44717976}, // ratio=2.40
	{2.75148990, 1.53385819}, // ratio=2.50
	{3.36298265, 1.98288283}, // ratio=3.00
	{4.08649929, 2.23811846}, // ratio=3.50
	{4.85481134, 2.47563463}, // ratio=4.00
	{5.62945551, 2.72948597}, // ratio=4.50
	{6.43023796, 2.98020421}, // ratio=5.00
}

const (
	kMinRatio          f32 = 2.00
	kFirstStepInverse  f32 = 10
	kFirstMaxRatio     f32 = 2.50
	kFirstNumRecords   f32 = 6
	kSecondStepInverse f32 = 2
	kSecondMaxRatio    f32 = 5.00
	kThirdNSlope       f32 = 1.559599389
	kThirdKxjSlope     f32 = 0.522807185
)

func computeNAndXj(ratio f32) (f32, f32) {
	last := kPrecomputedVariables[len(kPrecomputedVariables)-1]
	if ratio > kSecondMaxRatio {
		n := float32(kThirdNSlope*(ratio-kSecondMaxRatio)) + last[0]
		kxj := float32(kThirdKxjSlope*(ratio-kSecondMaxRatio)) + last[1]
		return n, 1 - 1/kxj
	}
	ratio = clamp32(ratio, kMinRatio, kSecondMaxRatio)
	var steps f32
	if ratio < kFirstMaxRatio {
		steps = float32((ratio - kMinRatio) * kFirstStepInverse)
	} else {
		steps = float32((ratio-kFirstMaxRatio)*kSecondStepInverse) + kFirstNumRecords - 1
	}
	left := int(math.Floor(float64(steps)))
	if left < 0 { // size_t clamp: a negative floor wraps to a huge value in C++
		left = len(kPrecomputedVariables) - 2
	}
	if left > len(kPrecomputedVariables)-2 {
		left = len(kPrecomputedVariables) - 2
	}
	frac := steps - f32(left)
	n := float32((1-frac)*kPrecomputedVariables[left][0]) + float32(frac*kPrecomputedVariables[left+1][0])
	kxj := float32((1-frac)*kPrecomputedVariables[left][1]) + float32(frac*kPrecomputedVariables[left+1][1])
	return n, 1 - 1/kxj
}

func findCircleCenter(a, b point32, r f32) point32 {
	aToB := b.sub(a)
	m := a.add(b).scale(0.5)
	cToM := point32{-aToB.y, aToB.x}
	distanceAM := aToB.length() / 2
	distanceCM := sqrt32(float32(r*r) - float32(distanceAM*distanceAM))
	return m.sub(cToM.normalize().scale(distanceCM))
}

func computeOctant(center point32, a, radius f32) seOctant {
	if radius <= kEhCloseEnough {
		return seOctant{offset: center, seA: a, seN: 0, circleStart: point32{a, a}}
	}
	ratio := a * 2 / radius
	g := float32(kGapFactor * radius)
	n, xjRatio := computeNAndXj(ratio)
	xJ := float32(xjRatio * a)
	yJ := float32(pow32(1-pow32(xjRatio, n), 1/n) * a)
	maxTheta := asin32(pow32(xjRatio, n/2))
	tanPhiJ := pow32(xJ/yJ, n-1)
	d := (xJ - float32(tanPhiJ*yJ)) / (1 - tanPhiJ)
	R := f32(float64(a-d-g) * math.Sqrt(2)) // C++: Scalar * sqrt(2) is a double product
	pointM := point32{a - g, a - g}
	pointJ := point32{xJ, yJ}
	circleCenter := pointM
	var circleMaxAngle f32
	if radius != 0 {
		circleCenter = findCircleCenter(pointJ, pointM, R)
		circleMaxAngle = pointM.sub(circleCenter).angleTo(pointJ.sub(circleCenter))
	}
	return seOctant{offset: center, seA: a, seN: n, seMaxTheta: maxTheta, circleStart: pointJ,
		circleCenter: circleCenter, circleMaxAngle: circleMaxAngle, circleRadius: R}
}

func computeQuadrant(center, corner, inRadii, sign point32) seQuadrant {
	cornerVector := corner.sub(center)
	radii := point32{min32(abs32(inRadii.x), abs32(cornerVector.x)), min32(abs32(inRadii.y), abs32(cornerVector.y))}
	normRadius := min32(radii.x, radii.y)
	forwardScale := point32{1, 1}
	if normRadius != 0 {
		forwardScale = point32{radii.x / normRadius, radii.y / normRadius}
	}
	normHalfSize := cornerVector.abs().div(forwardScale)
	signedScale := cornerVector.div(normHalfSize)
	if isNaN32(signedScale.x) {
		signedScale.x = sign.x
	}
	if isNaN32(signedScale.y) {
		signedScale.y = sign.y
	}
	c := normHalfSize.x - normHalfSize.y
	return seQuadrant{offset: center, signedScale: signedScale,
		top:   computeOctant(point32{0, -c}, normHalfSize.x, normRadius),
		right: computeOctant(point32{c, 0}, normHalfSize.y, normRadius)}
}

func nearlyEqual32(x, y, tol f32) bool { return abs32(x-y) <= tol }

func makeRSuperellipseParam(l, t, r, b f32, tl, tr, br, bl point32) rsuperellipseParam {
	sameCorners := nearlyEqual32(tl.x, tr.x, kEhCloseEnough) && nearlyEqual32(tl.x, br.x, kEhCloseEnough) &&
		nearlyEqual32(tl.x, bl.x, kEhCloseEnough) && nearlyEqual32(tl.y, tr.y, kEhCloseEnough) &&
		nearlyEqual32(tl.y, br.y, kEhCloseEnough) && nearlyEqual32(tl.y, bl.y, kEhCloseEnough)
	tlEmpty := !(tl.x > 0 && tl.y > 0)
	center := point32{(l + r) / 2, (t + b) / 2}
	if sameCorners && !tlEmpty {
		return rsuperellipseParam{topRight: computeQuadrant(center, point32{r, t}, tr, point32{-1, 1}), allCornersSame: true}
	}
	topSplit := split(l, r, tl.x, tr.x)
	rightSplit := split(t, b, tr.y, br.y)
	bottomSplit := split(l, r, bl.x, br.x)
	leftSplit := split(t, b, tl.y, bl.y)
	return rsuperellipseParam{
		topRight:    computeQuadrant(point32{topSplit, rightSplit}, point32{r, t}, tr, point32{1, -1}),
		bottomRight: computeQuadrant(point32{bottomSplit, rightSplit}, point32{r, b}, br, point32{1, 1}),
		bottomLeft:  computeQuadrant(point32{bottomSplit, leftSplit}, point32{l, b}, bl, point32{-1, 1}),
		topLeft:     computeQuadrant(point32{topSplit, leftSplit}, point32{l, t}, tl, point32{-1, -1}),
	}
}

func octantContains(o seOctant, p point32) bool {
	if p.x < 0 || p.y < 0 || p.y < p.x {
		return true
	}
	if p.x <= o.circleStart.x {
		pse := point32{p.x / o.seA, p.y / o.seA}
		return pow32(pse.x, o.seN)+pow32(pse.y, o.seN) <= 1
	}
	circleRadius := o.circleStart.distanceSquared(o.circleCenter)
	pc := p.sub(o.circleCenter)
	return pc.distanceSquared(point32{}) < circleRadius
}

func cornerContains(q seQuadrant, p point32, checkQuadrant bool) bool {
	norm := p.sub(q.offset).div(q.signedScale)
	if checkQuadrant {
		if norm.x < 0 || norm.y < 0 {
			return true
		}
	} else {
		norm = norm.abs()
	}
	if q.top.seN < 2 || q.right.seN < 2 {
		xDelta := q.right.offset.x + q.right.seA - norm.x
		yDelta := q.top.offset.y + q.top.seA - norm.y
		xWithin := xDelta > 0 || (xDelta == 0 && q.signedScale.x < 0)
		yWithin := yDelta > 0 || (yDelta == 0 && q.signedScale.y < 0)
		return xWithin && yWithin
	}
	flip := func(a point32) point32 { return point32{a.y, a.x} }
	return octantContains(q.top, norm.sub(q.top.offset)) && octantContains(q.right, flip(norm.sub(q.right.offset)))
}

func (prm rsuperellipseParam) contains(p point32) bool {
	if prm.allCornersSame {
		return cornerContains(prm.topRight, p, false)
	}
	return cornerContains(prm.topRight, p, true) && cornerContains(prm.bottomRight, p, true) &&
		cornerContains(prm.bottomLeft, p, true) && cornerContains(prm.topLeft, p, true)
}

// Contains reports whether point is inside the rounded superellipse, as
// the engine computes it.
func (r RSuperellipse) Contains(point Offset) bool {
	l, t, rt, b := safeNarrow(r.Left), safeNarrow(r.Top), safeNarrow(r.Right), safeNarrow(r.Bottom)
	// DlRect::GetPositive: an empty (or inverted) rect is reordered.
	if !(l < rt && t < b) {
		l, rt = min32(l, rt), f32(math.Max(float64(l), float64(rt)))
		t, b = min32(t, b), f32(math.Max(float64(t), float64(b)))
	}
	p := point32{safeNarrow(point.Dx), safeNarrow(point.Dy)}
	if !(l < rt && t < b) || !(p.x >= l && p.y >= t && p.x < rt && p.y < b) {
		return false
	}
	n := func(x, y float64) point32 { return point32{safeNarrow(x), safeNarrow(y)} }
	prm := makeRSuperellipseParam(l, t, rt, b, n(r.TlRadiusX, r.TlRadiusY), n(r.TrRadiusX, r.TrRadiusY),
		n(r.BrRadiusX, r.BrRadiusY), n(r.BlRadiusX, r.BlRadiusY))
	return prm.contains(p)
}
