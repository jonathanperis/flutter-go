package ui

// The clip-tracking half of the engine's DisplayListBuilder, which backs
// Canvas.getLocalClipBounds/getDestinationClipBounds: a port of
// display_list/utils/dl_matrix_clip_tracker.cc (DisplayListMatrixClipState)
// and the impeller/geometry rect and matrix operations it uses, in float32.
// The current matrix itself comes from the Impeller SDK's builder (the
// same DisplayListBuilder code), so only the clip arithmetic is ported.
//
// Exactness: the engine is built by clang for arm64 with its default
// -ffp-contract=on, so within one C++ expression `a*b + c` becomes a fused
// multiply-add (llvm.fmuladd: the left multiply first, else the right
// one). fmuladd32 reproduces that (a float64 FMA rounded to float32: the
// double rounding differs from a float32 FMA only on exact float32 ties,
// which these inputs don't produce in practice).

import (
	"math"

	"github.com/jonathanperis/flutter-go/src/impeller"
)

// dlRect is impeller's TRect<float>: left, top, right, bottom.
type dlRect struct{ l, t, r, b float32 }

func (r dlRect) isEmpty() bool { return !(r.l < r.r && r.t < r.b) }

func (r dlRect) isFinite() bool {
	return !isInfOrNaN32(r.l) && !isInfOrNaN32(r.t) && !isInfOrNaN32(r.r) && !isInfOrNaN32(r.b)
}

func (r dlRect) toRect() Rect {
	return Rect{Left: float64(r.l), Top: float64(r.t), Right: float64(r.r), Bottom: float64(r.b)}
}

// rectToDl is DlRect::MakeLTRB(SafeNarrow(...)).
func rectToDl(r Rect) dlRect {
	return dlRect{safeNarrow(r.Left), safeNarrow(r.Top), safeNarrow(r.Right), safeNarrow(r.Bottom)}
}

// protectEmpty is ProtectEmpty: an empty rect becomes the zero rect.
func protectEmpty(r dlRect) dlRect {
	if r.isEmpty() {
		return dlRect{}
	}
	return r
}

func stdMin32(a, b float32) float32 { // std::min
	if b < a {
		return b
	}
	return a
}

func stdMax32(a, b float32) float32 { // std::max
	if a < b {
		return b
	}
	return a
}

func (r dlRect) intersectsWithRect(o dlRect) bool {
	return !r.isEmpty() && !o.isEmpty() && r.l < o.r && r.t < o.b && r.r > o.l && r.b > o.t
}

// intersection is TRect::Intersection; ok is false for std::nullopt.
func (r dlRect) intersection(o dlRect) (dlRect, bool) {
	if !r.intersectsWithRect(o) {
		return dlRect{}, false
	}
	return dlRect{stdMax32(r.l, o.l), stdMax32(r.t, o.t), stdMin32(r.r, o.r), stdMin32(r.b, o.b)}, true
}

func (r dlRect) intersectionOrEmpty(o dlRect) dlRect {
	i, _ := r.intersection(o)
	return i
}

// cutoutOrEmpty is TRect::CutoutOrEmpty.
func (r dlRect) cutoutOrEmpty(o dlRect) dlRect {
	if r.isEmpty() {
		return dlRect{}
	}
	if o.l <= r.l && o.r >= r.r {
		if o.t <= r.t && o.b >= r.b {
			return dlRect{}
		}
		if o.t <= r.t && o.b > r.t {
			return dlRect{r.l, o.b, r.r, r.b}
		}
		if o.b >= r.b && o.t < r.b {
			return dlRect{r.l, r.t, r.r, o.t}
		}
	}
	if o.t <= r.t && o.b >= r.b {
		if o.l <= r.l && o.r > r.l {
			return dlRect{o.r, r.t, r.r, r.b}
		}
		if o.r >= r.r && o.l < r.r {
			return dlRect{r.l, r.t, o.l, r.b}
		}
	}
	return r
}

func roundOut(r dlRect) dlRect {
	return dlRect{float32(math.Floor(float64(r.l))), float32(math.Floor(float64(r.t))),
		float32(math.Ceil(float64(r.r))), float32(math.Ceil(float64(r.b)))}
}

func roundRect(r dlRect) dlRect { // TRect::Round: std::round, halves away from zero
	return dlRect{float32(math.Round(float64(r.l))), float32(math.Round(float64(r.t))),
		float32(math.Round(float64(r.r))), float32(math.Round(float64(r.b)))}
}

// fmuladd32 is clang's contracted a*b + c in float.
func fmuladd32(a, b, c float32) float32 {
	return float32(math.FMA(float64(a), float64(b), float64(c)))
}

// ---- impeller::Matrix (column-major float32) ----

type dlMatrix = impeller.Matrix

func hasPerspective2D(m *dlMatrix) bool { return m[3] != 0 || m[7] != 0 || m[15] != 1 }

func hasPerspective(m *dlMatrix) bool {
	return m[3] != 0 || m[7] != 0 || m[11] != 0 || m[15] != 1
}

// isAligned2D is Matrix::IsAligned2D() (zero tolerance).
func isAligned2D(m *dlMatrix) bool {
	if hasPerspective2D(m) {
		return false
	}
	if m[1] == 0 && m[4] == 0 {
		return true
	}
	return m[0] == 0 && m[5] == 0
}

// transformPoint is Matrix::operator*(const Point&).
func transformPoint(m *dlMatrix, x, y float32) (float32, float32) {
	w := fmuladd32(x, m[3], float32(y*m[7])) + m[15]
	rx := fmuladd32(x, m[0], float32(y*m[4])) + m[12]
	ry := fmuladd32(x, m[1], float32(y*m[5])) + m[13]
	if w != 0 {
		w = 1 / w
	}
	return rx * w, ry * w
}

// transformBounds is TRect::TransformBounds.
func (r dlRect) transformBounds(m *dlMatrix) dlRect {
	if r.isEmpty() {
		return dlRect{}
	}
	pts := [4][2]float32{{r.l, r.t}, {r.r, r.t}, {r.l, r.b}, {r.r, r.b}}
	var out dlRect
	for i, p := range pts {
		x, y := transformPoint(m, p[0], p[1])
		if i == 0 {
			out = dlRect{x, y, x, y}
			continue
		}
		out = dlRect{stdMin32(out.l, x), stdMin32(out.t, y), stdMax32(out.r, x), stdMax32(out.b, y)}
	}
	return out
}

// transformAndClipBounds is TRect::TransformAndClipBounds. ENGINE GAP
// (ported partially): perspective matrices that clip against the near
// plane fall back to the transformed bounds.
func (r dlRect) transformAndClipBounds(m *dlMatrix) dlRect {
	return r.transformBounds(m)
}

// invert is Matrix::Invert, with the engine's contracted float arithmetic.
func invert(m *dlMatrix) (dlMatrix, bool) {
	// t is one cofactor: s1*a1*b1 ± s2*a2*b2 ± ... with the signs of the
	// C++ source; each triple product is (a*b)*c and each +/- of a product
	// onto the running sum is fused (the first pair fuses its left term).
	type term struct {
		neg     bool
		a, b, c int
	}
	cof := func(terms ...term) float32 {
		prod := func(t term) (float32, float32) { // (a*b) and c, sign on a
			a := m[t.a]
			if t.neg {
				a = -a
			}
			return float32(a * m[t.b]), m[t.c]
		}
		ab0, c0 := prod(terms[0])
		ab1, c1 := prod(terms[1])
		// T0 + T1 (signs folded into the products): fmuladd(ab0, c0, ab1*c1).
		s := fmuladd32(ab0, c0, float32(ab1*c1))
		for _, t := range terms[2:] {
			ab, c := prod(t)
			s = fmuladd32(ab, c, s)
		}
		return s
	}
	T := func(neg bool, a, b, c int) term { return term{neg, a, b, c} }
	// Signs: "x - y*z*w" is fmuladd(-(y*z), w, x), i.e. a negated first
	// factor; a leading unary minus is (-m[i]) * m[j] * m[k].
	tmp := dlMatrix{
		cof(T(false, 5, 10, 15), T(true, 5, 11, 14), T(true, 9, 6, 15), T(false, 9, 7, 14), T(false, 13, 6, 11), T(true, 13, 7, 10)),
		cof(T(true, 1, 10, 15), T(false, 1, 11, 14), T(false, 9, 2, 15), T(true, 9, 3, 14), T(true, 13, 2, 11), T(false, 13, 3, 10)),
		cof(T(false, 1, 6, 15), T(true, 1, 7, 14), T(true, 5, 2, 15), T(false, 5, 3, 14), T(false, 13, 2, 7), T(true, 13, 3, 6)),
		cof(T(true, 1, 6, 11), T(false, 1, 7, 10), T(false, 5, 2, 11), T(true, 5, 3, 10), T(true, 9, 2, 7), T(false, 9, 3, 6)),
		cof(T(true, 4, 10, 15), T(false, 4, 11, 14), T(false, 8, 6, 15), T(true, 8, 7, 14), T(true, 12, 6, 11), T(false, 12, 7, 10)),
		cof(T(false, 0, 10, 15), T(true, 0, 11, 14), T(true, 8, 2, 15), T(false, 8, 3, 14), T(false, 12, 2, 11), T(true, 12, 3, 10)),
		cof(T(true, 0, 6, 15), T(false, 0, 7, 14), T(false, 4, 2, 15), T(true, 4, 3, 14), T(true, 12, 2, 7), T(false, 12, 3, 6)),
		cof(T(false, 0, 6, 11), T(true, 0, 7, 10), T(true, 4, 2, 11), T(false, 4, 3, 10), T(false, 8, 2, 7), T(true, 8, 3, 6)),
		cof(T(false, 4, 9, 15), T(true, 4, 11, 13), T(true, 8, 5, 15), T(false, 8, 7, 13), T(false, 12, 5, 11), T(true, 12, 7, 9)),
		cof(T(true, 0, 9, 15), T(false, 0, 11, 13), T(false, 8, 1, 15), T(true, 8, 3, 13), T(true, 12, 1, 11), T(false, 12, 3, 9)),
		cof(T(false, 0, 5, 15), T(true, 0, 7, 13), T(true, 4, 1, 15), T(false, 4, 3, 13), T(false, 12, 1, 7), T(true, 12, 3, 5)),
		cof(T(true, 0, 5, 11), T(false, 0, 7, 9), T(false, 4, 1, 11), T(true, 4, 3, 9), T(true, 8, 1, 7), T(false, 8, 3, 5)),
		cof(T(true, 4, 9, 14), T(false, 4, 10, 13), T(false, 8, 5, 14), T(true, 8, 6, 13), T(true, 12, 5, 10), T(false, 12, 6, 9)),
		cof(T(false, 0, 9, 14), T(true, 0, 10, 13), T(true, 8, 1, 14), T(false, 8, 2, 13), T(false, 12, 1, 10), T(true, 12, 2, 9)),
		cof(T(true, 0, 5, 14), T(false, 0, 6, 13), T(false, 4, 1, 14), T(true, 4, 2, 13), T(true, 12, 1, 6), T(false, 12, 2, 5)),
		cof(T(false, 0, 5, 10), T(true, 0, 6, 9), T(true, 4, 1, 10), T(false, 4, 2, 9), T(false, 8, 1, 6), T(true, 8, 2, 5)),
	}
	det := fmuladd32(m[3], tmp[12], fmuladd32(m[2], tmp[8], fmuladd32(m[0], tmp[0], float32(m[1]*tmp[4]))))
	if det == 0 {
		return dlMatrix{}, false
	}
	det = float32(1.0 / float64(det)) // `det = 1.0 / det`: a double division
	var r dlMatrix
	for i := range r {
		r[i] = float32(tmp[i] * det)
	}
	return r, true
}

// determinant is Matrix::GetDeterminant (IsInvertible is det != 0).
func determinant(m *dlMatrix) float32 {
	e := func(c, r int) float32 { return m[c*4+r] }
	a00, a01, a02, a03 := e(0, 0), e(0, 1), e(0, 2), e(0, 3)
	a10, a11, a12, a13 := e(1, 0), e(1, 1), e(1, 2), e(1, 3)
	a20, a21, a22, a23 := e(2, 0), e(2, 1), e(2, 2), e(2, 3)
	a30, a31, a32, a33 := e(3, 0), e(3, 1), e(3, 2), e(3, 3)
	// x*y - z*w is fmuladd(x, y, -(z*w)).
	d := func(x, y, z, w float32) float32 { return fmuladd32(x, y, -float32(z*w)) }
	b00, b01, b02 := d(a00, a11, a01, a10), d(a00, a12, a02, a10), d(a00, a13, a03, a10)
	b03, b04, b05 := d(a01, a12, a02, a11), d(a01, a13, a03, a11), d(a02, a13, a03, a12)
	b06, b07, b08 := d(a20, a31, a21, a30), d(a20, a32, a22, a30), d(a20, a33, a23, a30)
	b09, b10, b11 := d(a21, a32, a22, a31), d(a21, a33, a23, a31), d(a22, a33, a23, a32)
	// b00*b11 - b01*b10 + b02*b09 + b03*b08 - b04*b07 + b05*b06
	s := fmuladd32(b00, b11, -float32(b01*b10))
	s = fmuladd32(b02, b09, s)
	s = fmuladd32(b03, b08, s)
	s = fmuladd32(-b04, b07, s)
	s = fmuladd32(b05, b06, s)
	return s
}

// ---- DisplayListMatrixClipState ----

// clipState is the builder's global DisplayListMatrixClipState minus its
// matrix (read from the SDK's builder when needed).
type clipState struct{ cull dlRect }

// kMaxCullRect is DisplayListBuilder::kMaxCullRect.
var kMaxCullRect = dlRect{-1e9, -1e9, 1e9, 1e9}

func newClipState(cull dlRect) clipState { return clipState{cull: protectEmpty(cull)} }

// mapRect is DisplayListMatrixClipState::mapRect: the mapped bounds, and
// whether the matrix keeps rects axis-aligned.
func mapRect(m *dlMatrix, src dlRect) (dlRect, bool) {
	return src.transformAndClipBounds(m), isAligned2D(m)
}

// adjustCullRect is DisplayListMatrixClipState::adjustCullRect.
func (s *clipState) adjustCullRect(m *dlMatrix, clip dlRect, op ClipOp, isAA bool) {
	if s.cull.isEmpty() {
		return
	}
	if hasPerspective(m) {
		return
	}
	switch op {
	case ClipOpIntersect:
		if clip.isEmpty() {
			s.cull = dlRect{}
			break
		}
		rect, _ := mapRect(m, clip)
		if isAA {
			rect = roundOut(rect)
		}
		s.cull = s.cull.intersectionOrEmpty(rect)
	case ClipOpDifference:
		if clip.isEmpty() {
			break
		}
		if rect, ok := mapRect(m, clip); ok {
			if isAA {
				rect = roundRect(rect)
				if rect.isEmpty() {
					break
				}
			}
			s.cull = s.cull.cutoutOrEmpty(rect)
		}
	}
}

// clipRect is DisplayListMatrixClipState::clipRect.
func (s *clipState) clipRect(m *dlMatrix, rect dlRect, op ClipOp, isAA bool) {
	if rect.isFinite() {
		s.adjustCullRect(m, rect, op, isAA)
	}
}

// localCullCoverage is GetLocalCullCoverage.
func (s *clipState) localCullCoverage(m *dlMatrix) dlRect {
	if s.cull.isEmpty() {
		return dlRect{}
	}
	if determinant(m) == 0 {
		return dlRect{}
	}
	if hasPerspective2D(m) {
		return kMaxCullRect
	}
	inv, _ := invert(m)
	return s.cull.transformBounds(&inv)
}

// resetDeviceCullRect is DisplayListMatrixClipState::resetDeviceCullRect.
func (s *clipState) resetDeviceCullRect(r dlRect) {
	if r.isEmpty() {
		s.cull = dlRect{}
	} else {
		s.cull = r
	}
}
