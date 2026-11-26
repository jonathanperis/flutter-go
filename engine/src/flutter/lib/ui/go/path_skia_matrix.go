package ui

// The parts of SkMatrix the path code uses: Skia's include/core/SkMatrix.h
// and src/core/SkMatrix.cpp (type masks, set/pre/post scale, translate,
// rotate and concat, the mapPoints procs, mapPoint, mapHomogeneousPoints),
// and lib/ui/painting/matrix.cc's ToSkMatrix.

import "math"

const (
	kMScaleX = iota
	kMSkewX
	kMTransX
	kMSkewY
	kMScaleY
	kMTransY
	kMPersp0
	kMPersp1
	kMPersp2
)

const (
	kIdentityMask          = 0
	kTranslateMask         = 0x01
	kScaleMask             = 0x02
	kAffineMask            = 0x04
	kPerspectiveMask       = 0x08
	kRectStaysRectMask     = 0x10
	kOnlyPerspectiveValid  = 0x40
	kUnknownMask           = 0x80
	kORableMasks           = kTranslateMask | kScaleMask | kAffineMask | kPerspectiveMask
	kAllMasks              = kORableMasks | kRectStaysRectMask
	kRectStaysRectShift    = 4
	kScalar1Int            = 0x3f800000
	skMatrixIdentityMaskRS = kIdentityMask | kRectStaysRectMask
)

type skMatrix struct {
	m    [9]float32
	mask int32
}

func skMatrixIdentity() skMatrix {
	return skMatrix{m: [9]float32{1, 0, 0, 0, 1, 0, 0, 0, 1}, mask: skMatrixIdentityMaskRS}
}

// skMatrixTranslate is SkMatrix::Translate (setTranslate).
func skMatrixTranslate(dx, dy float32) skMatrix {
	mask := int32(kIdentityMask | kRectStaysRectMask)
	if dx != 0 || dy != 0 {
		mask = kTranslateMask | kRectStaysRectMask
	}
	return skMatrix{m: [9]float32{1, 0, dx, 0, 1, dy, 0, 0, 1}, mask: mask}
}

// skMatrixScaleTranslate is SkMatrix::ScaleTranslate.
func skMatrixScaleTranslate(sx, sy, tx, ty float32) skMatrix {
	var mask int32
	if sx != 1 || sy != 1 {
		mask |= kScaleMask
	}
	if tx != 0 || ty != 0 {
		mask |= kTranslateMask
	}
	if sx != 0 && sy != 0 {
		mask |= kRectStaysRectMask
	}
	return skMatrix{m: [9]float32{sx, 0, tx, 0, sy, ty, 0, 0, 1}, mask: mask}
}

// skMatrixFromMatrix4 is matrix.cc's ToSkMatrix: entries 0, 4, 12, 1, 5,
// 13, 3, 7, 15 of the column-major 4x4, each SafeNarrow'd; the type is
// unknown until asked.
func skMatrixFromMatrix4(m4 []float64) skMatrix {
	idx := [9]int{0, 4, 12, 1, 5, 13, 3, 7, 15}
	var m skMatrix
	for i, j := range idx {
		if j < len(m4) {
			m.m[i] = safeNarrow(m4[j])
		}
	}
	m.mask = kUnknownMask
	return m
}

func as2sCompliment(x float32) int32 {
	bits := int32(math.Float32bits(x))
	if bits < 0 {
		bits &= 0x7FFFFFFF
		bits = -bits
	}
	return bits
}

func (m *skMatrix) computePerspectiveTypeMask() int32 {
	if m.m[kMPersp0] != 0 || m.m[kMPersp1] != 0 || m.m[kMPersp2] != 1 {
		return kORableMasks
	}
	return kOnlyPerspectiveValid | kUnknownMask
}

func (m *skMatrix) computeTypeMask() int32 {
	var mask int32
	if m.m[kMPersp0] != 0 || m.m[kMPersp1] != 0 || m.m[kMPersp2] != 1 {
		return kORableMasks
	}
	if m.m[kMTransX] != 0 || m.m[kMTransY] != 0 {
		mask |= kTranslateMask
	}
	m00 := as2sCompliment(m.m[kMScaleX])
	m01 := as2sCompliment(m.m[kMSkewX])
	m10 := as2sCompliment(m.m[kMSkewY])
	m11 := as2sCompliment(m.m[kMScaleY])
	b := func(v bool) int32 {
		if v {
			return 1
		}
		return 0
	}
	if m01|m10 != 0 {
		mask |= kAffineMask | kScaleMask
		dp0 := b(m00|m11 == 0)
		ds1 := b(m01 != 0) & b(m10 != 0)
		mask |= (dp0 & ds1) << kRectStaysRectShift
	} else {
		if (m00^kScalar1Int)|(m11^kScalar1Int) != 0 {
			mask |= kScaleMask
		}
		mask |= (b(m00 != 0) & b(m11 != 0)) << kRectStaysRectShift
	}
	return mask
}

func (m *skMatrix) getType() int32 {
	if m.mask&kUnknownMask != 0 {
		m.mask = m.computeTypeMask()
	}
	return m.mask & 0xF
}

func (m *skMatrix) isIdentity() bool { return m.getType() == 0 }

func (m *skMatrix) hasPerspective() bool {
	if m.mask&kUnknownMask != 0 && m.mask&kOnlyPerspectiveValid == 0 {
		m.mask = m.computePerspectiveTypeMask()
	}
	return m.mask&0xF&kPerspectiveMask != 0
}

func (m *skMatrix) rectStaysRect() bool {
	if m.mask&kUnknownMask != 0 {
		m.mask = m.computeTypeMask()
	}
	return m.mask&kRectStaysRectMask != 0
}

func (m *skMatrix) isTriviallyIdentity() bool {
	if m.mask&kUnknownMask != 0 {
		return false
	}
	return m.mask&0xF == 0
}

func (m *skMatrix) updateTranslateMask() {
	if m.m[kMTransX] != 0 || m.m[kMTransY] != 0 {
		m.mask |= kTranslateMask
	} else {
		m.mask &^= kTranslateMask
	}
}

// setScale is SkMatrix::setScale(sx, sy).
func (m *skMatrix) setScale(sx, sy float32) {
	var rectMask int32 = kRectStaysRectMask
	if sx == 0 || sy == 0 {
		rectMask = 0
	}
	mask := kScaleMask | rectMask
	if sx == 1 && sy == 1 {
		mask = kIdentityMask | rectMask
	}
	*m = skMatrix{m: [9]float32{sx, 0, 0, 0, sy, 0, 0, 0, 1}, mask: mask}
}

// postTranslate is SkMatrix::postTranslate.
func (m *skMatrix) postTranslate(dx, dy float32) {
	if m.hasPerspective() {
		t := skMatrixTranslate(dx, dy)
		m.postConcat(&t)
		return
	}
	m.m[kMTransX] += dx
	m.m[kMTransY] += dy
	m.updateTranslateMask()
}

// preScale is SkMatrix::preScale(sx, sy).
func (m *skMatrix) preScale(sx, sy float32) {
	if sx == 1 && sy == 1 {
		return
	}
	m.m[kMScaleX] = mul32(m.m[kMScaleX], sx)
	m.m[kMSkewY] = mul32(m.m[kMSkewY], sx)
	m.m[kMPersp0] = mul32(m.m[kMPersp0], sx)
	m.m[kMSkewX] = mul32(m.m[kMSkewX], sy)
	m.m[kMScaleY] = mul32(m.m[kMScaleY], sy)
	m.m[kMPersp1] = mul32(m.m[kMPersp1], sy)
	if m.m[kMScaleX] == 1 && m.m[kMScaleY] == 1 && m.mask&(kPerspectiveMask|kAffineMask) == 0 {
		m.mask &^= kScaleMask
	} else {
		m.mask |= kScaleMask
		if sx == 0 || sy == 0 {
			m.mask &^= kRectStaysRectMask
		}
	}
}

// setSinCos is SkMatrix::setSinCos(sin, cos).
func (m *skMatrix) setSinCos(sinV, cosV float32) {
	*m = skMatrix{m: [9]float32{cosV, -sinV, 0, sinV, cosV, 0, 0, 0, 1},
		mask: kUnknownMask | kOnlyPerspectiveValid}
}

// setRotate is SkMatrix::setRotate(degrees).
func (m *skMatrix) setRotate(degrees float32) {
	rad := skDegreesToRadians(degrees)
	m.setSinCos(skSinCosSnapToZero(rad))
}

// preRotate is SkMatrix::preRotate(degrees).
func (m *skMatrix) preRotate(degrees float32) {
	var r skMatrix
	r.setRotate(degrees)
	m.preConcat(&r)
}

func (m *skMatrix) preConcat(o *skMatrix) {
	if !o.isIdentity() {
		a := *m
		m.setConcat(&a, o)
	}
}

func (m *skMatrix) postConcat(o *skMatrix) {
	if !o.isIdentity() {
		b := *m
		m.setConcat(o, &b)
	}
}

// muladdmul is SkMatrix.cpp's muladdmul: (float)((double)a*b + (double)c*d).
func muladdmul(a, b, c, d float32) float32 {
	return float32(float64(float64(a)*float64(b)) + float64(float64(c)*float64(d)))
}

// rowcol3 is row[0]*col[0] + row[1]*col[3] + row[2]*col[6], fused.
func rowcol3(row, col []float32) float32 {
	return fma32(row[2], col[6], fma32(row[0], col[0], mul32(row[1], col[3])))
}

// setConcat is SkMatrix::setConcat(a, b).
func (m *skMatrix) setConcat(a, b *skMatrix) {
	aType, bType := a.getType(), b.getType()
	switch {
	case a.isTriviallyIdentity():
		*m = *b
	case b.isTriviallyIdentity():
		*m = *a
	case (aType|bType)&(kAffineMask|kPerspectiveMask) == 0:
		*m = skMatrixScaleTranslate(mul32(a.m[kMScaleX], b.m[kMScaleX]), mul32(a.m[kMScaleY], b.m[kMScaleY]),
			fma32(a.m[kMScaleX], b.m[kMTransX], a.m[kMTransX]), fma32(a.m[kMScaleY], b.m[kMTransY], a.m[kMTransY]))
	case (aType|bType)&kPerspectiveMask != 0:
		var t skMatrix
		for r := 0; r < 3; r++ {
			for c := 0; c < 3; c++ {
				t.m[r*3+c] = rowcol3(a.m[r*3:], b.m[c:])
			}
		}
		t.mask = kUnknownMask
		*m = t
	default:
		var t skMatrix
		t.m[kMScaleX] = muladdmul(a.m[kMScaleX], b.m[kMScaleX], a.m[kMSkewX], b.m[kMSkewY])
		t.m[kMSkewX] = muladdmul(a.m[kMScaleX], b.m[kMSkewX], a.m[kMSkewX], b.m[kMScaleY])
		t.m[kMTransX] = muladdmul(a.m[kMScaleX], b.m[kMTransX], a.m[kMSkewX], b.m[kMTransY]) + a.m[kMTransX]
		t.m[kMSkewY] = muladdmul(a.m[kMSkewY], b.m[kMScaleX], a.m[kMScaleY], b.m[kMSkewY])
		t.m[kMScaleY] = muladdmul(a.m[kMSkewY], b.m[kMSkewX], a.m[kMScaleY], b.m[kMScaleY])
		t.m[kMTransY] = muladdmul(a.m[kMSkewY], b.m[kMTransX], a.m[kMScaleY], b.m[kMTransY]) + a.m[kMTransY]
		t.m[kMPersp2] = 1
		t.mask = kUnknownMask | kOnlyPerspectiveValid
		*m = t
	}
}

// sdot is SkMatrix.cpp's sdot(a, b, c, d): a*b + c*d, fused.
func sdot(a, b, c, d float32) float32 { return fma32(a, b, mul32(c, d)) }

// mapPoint is SkMatrix::mapPoint (mapPointAffine or mapPointPerspective).
func (m *skMatrix) mapPoint(p skPoint) skPoint {
	if m.hasPerspective() {
		return m.mapPointPerspective(p)
	}
	return skPoint{fma32(p.x, m.m[0], mul32(p.y, m.m[1])) + m.m[2], fma32(p.x, m.m[3], mul32(p.y, m.m[4])) + m.m[5]}
}

func (m *skMatrix) mapPointPerspective(p skPoint) skPoint {
	x := sdot(p.x, m.m[kMScaleX], p.y, m.m[kMSkewX]) + m.m[kMTransX]
	y := sdot(p.x, m.m[kMSkewY], p.y, m.m[kMScaleY]) + m.m[kMTransY]
	z := sdot(p.x, m.m[kMPersp0], p.y, m.m[kMPersp1]) + m.m[kMPersp2]
	if z != 0 {
		z = 1 / z
	}
	return skPoint{mul32(x, z), mul32(y, z)}
}

// mapPoints is SkMatrix::mapPoints(dst, src) through the type's proc
// (Identity_pts, Trans_pts, Scale_pts, Affine_vpts or Persp_pts; the
// vector procs are lane-wise float operations, never fused).
func (m *skMatrix) mapPoints(dst, src []skPoint) {
	switch t := m.getType(); {
	case t == 0:
		copy(dst, src)
	case t == kTranslateMask:
		tx, ty := m.m[kMTransX], m.m[kMTransY]
		for i, p := range src {
			dst[i] = skPoint{p.x + tx, p.y + ty}
		}
	case t&(kAffineMask|kPerspectiveMask) == 0:
		tx, ty, sx, sy := m.m[kMTransX], m.m[kMTransY], m.m[kMScaleX], m.m[kMScaleY]
		for i, p := range src {
			dst[i] = skPoint{mul32(p.x, sx) + tx, mul32(p.y, sy) + ty}
		}
	case t&kPerspectiveMask == 0:
		tx, ty, sx, sy := m.m[kMTransX], m.m[kMTransY], m.m[kMScaleX], m.m[kMScaleY]
		kx, ky := m.m[kMSkewX], m.m[kMSkewY]
		for i, p := range src {
			dst[i] = skPoint{(mul32(p.x, sx) + mul32(p.y, kx)) + tx, (mul32(p.y, sy) + mul32(p.x, ky)) + ty}
		}
	default:
		for i, p := range src {
			dst[i] = m.mapPointPerspective(p)
		}
	}
}

// mapHomogeneous is MapHomogeneousPointsWithStride for one point.
func (m *skMatrix) mapHomogeneous(p [3]float32) [3]float32 {
	if m.isIdentity() {
		return p
	}
	sdot3 := func(a, b, c, d, e, f float32) float32 { return fma32(e, f, fma32(a, b, mul32(c, d))) }
	return [3]float32{
		sdot3(p[0], m.m[kMScaleX], p[1], m.m[kMSkewX], p[2], m.m[kMTransX]),
		sdot3(p[0], m.m[kMSkewY], p[1], m.m[kMScaleY], p[2], m.m[kMTransY]),
		sdot3(p[0], m.m[kMPersp0], p[1], m.m[kMPersp1], p[2], m.m[kMPersp2]),
	}
}
