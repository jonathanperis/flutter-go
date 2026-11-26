package ui

// Port of painting.dart's PathFillType, PathOperation and Path (Flutter
// 3.50.1). The engine's CanvasPath (lib/ui/painting/path.cc) wraps Skia's
// SkPathBuilder; path_skia_*.go port that C++ in float as the engine
// computes it, so bounds, hit tests and metrics match Flutter bit for bit.

import (
	"github.com/jonathanperis/flutter-go/src/dart"
)

// PathFillType determines the winding rule that decides how the interior
// of a Path is calculated.
type PathFillType int

const (
	// PathFillTypeNonZero: the interior is defined by a non-zero sum of
	// signed edge crossings.
	PathFillTypeNonZero PathFillType = iota
	// PathFillTypeEvenOdd: the interior is defined by an odd number of edge
	// crossings.
	PathFillTypeEvenOdd
)

var pathFillTypeNames = [...]string{"nonZero", "evenOdd"}

// PathFillTypeValues is PathFillType.values.
func PathFillTypeValues() []PathFillType { return []PathFillType{PathFillTypeNonZero, PathFillTypeEvenOdd} }

func (t PathFillType) Name() string   { return pathFillTypeNames[t] }
func (t PathFillType) String() string { return "PathFillType." + t.Name() }

// PathOperation is a strategy for combining paths (Path.combine).
type PathOperation int

const (
	PathOperationDifference PathOperation = iota
	PathOperationIntersect
	PathOperationUnion
	PathOperationXor
	PathOperationReverseDifference
)

var pathOperationNames = [...]string{"difference", "intersect", "union", "xor", "reverseDifference"}

// PathOperationValues is PathOperation.values.
func PathOperationValues() []PathOperation {
	return []PathOperation{PathOperationDifference, PathOperationIntersect, PathOperationUnion,
		PathOperationXor, PathOperationReverseDifference}
}

func (o PathOperation) Name() string   { return pathOperationNames[o] }
func (o PathOperation) String() string { return "PathOperation." + o.Name() }

// Path is a complex, one-dimensional subset of a plane: a list of
// subpaths of lines and curves, filled or stroked by Canvas.drawPath and
// used by Canvas.clipPath. Use it by pointer (NewPath, NewPathFrom).
type Path struct {
	b skPathBuilder
}

// NewPath is Dart's Path(): an empty path.
func NewPath() *Path { return &Path{b: newSkPathBuilder()} }

// NewPathFrom is Path.from(source): a copy of source; later changes to
// either don't affect the other.
func NewPathFrom(source *Path) *Path { return &Path{b: source.b.clone()} }

// FillType is how the interior of the path is computed.
func (p *Path) FillType() PathFillType { return PathFillType(p.b.fillType) }

// SetFillType is the fillType setter.
func (p *Path) SetFillType(v PathFillType) { p.b.fillType = int(v) }

// assertions of painting.dart (_offsetIsValid, _rectIsValid...): Go keeps
// assertions on in every build.
func assertOffsetValid(o Offset) {
	if o.Dx != o.Dx || o.Dy != o.Dy {
		panic(dart.NewAssertionError("Offset argument contained a NaN value."))
	}
}

func assertRectValid(r Rect) {
	if r.HasNaN() {
		panic(dart.NewAssertionError("Rect argument contained a NaN value."))
	}
}

func assertRadiusValid(r Radius) {
	if r.X != r.X || r.Y != r.Y {
		panic(dart.NewAssertionError("Radius argument contained a NaN value."))
	}
}

func assertMatrix4Valid(m []float64) {
	if len(m) != 16 {
		panic(dart.NewAssertionError("Matrix4 must have 16 entries."))
	}
	for _, v := range m {
		if !isFinite(v) {
			panic(dart.NewAssertionError("Matrix4 entries must be finite."))
		}
	}
}

func pt(x, y float64) skPoint { return skPoint{safeNarrow(x), safeNarrow(y)} }

func skRectOf(r Rect) skRect {
	return skRect{safeNarrow(r.Left), safeNarrow(r.Top), safeNarrow(r.Right), safeNarrow(r.Bottom)}
}

// MoveTo starts a new subpath at the given coordinate.
func (p *Path) MoveTo(x, y float64) { p.b.moveTo(pt(x, y)) }

// RelativeMoveTo starts a new subpath at the given offset from the current
// point.
func (p *Path) RelativeMoveTo(dx, dy float64) { p.b.rMoveTo(pt(dx, dy)) }

// LineTo adds a straight line segment from the current point to the given
// point.
func (p *Path) LineTo(x, y float64) { p.b.lineTo(pt(x, y)) }

// RelativeLineTo adds a straight line segment from the current point to
// the point at the given offset from the current point.
func (p *Path) RelativeLineTo(dx, dy float64) { p.b.rLineTo(pt(dx, dy)) }

// QuadraticBezierTo adds a quadratic bezier segment that curves from the
// current point to (x2, y2), using (x1, y1) as the control point.
func (p *Path) QuadraticBezierTo(x1, y1, x2, y2 float64) { p.b.quadTo(pt(x1, y1), pt(x2, y2)) }

// RelativeQuadraticBezierTo is QuadraticBezierTo with offsets from the
// current point.
func (p *Path) RelativeQuadraticBezierTo(x1, y1, x2, y2 float64) { p.b.rQuadTo(pt(x1, y1), pt(x2, y2)) }

// CubicTo adds a cubic bezier segment that curves from the current point
// to (x3, y3), using the control points (x1, y1) and (x2, y2).
func (p *Path) CubicTo(x1, y1, x2, y2, x3, y3 float64) {
	p.b.cubicTo(pt(x1, y1), pt(x2, y2), pt(x3, y3))
}

// RelativeCubicTo is CubicTo with offsets from the current point.
func (p *Path) RelativeCubicTo(x1, y1, x2, y2, x3, y3 float64) {
	p.b.rCubicTo(pt(x1, y1), pt(x2, y2), pt(x3, y3))
}

// ConicTo adds a bezier segment that curves from the current point to
// (x2, y2) using the control point (x1, y1) and weight w: an elliptical
// arc for w < 1, a quadratic for w == 1, a hyperbola for w > 1.
func (p *Path) ConicTo(x1, y1, x2, y2, w float64) {
	p.b.conicTo(pt(x1, y1), pt(x2, y2), safeNarrow(w))
}

// RelativeConicTo is ConicTo with offsets from the current point.
func (p *Path) RelativeConicTo(x1, y1, x2, y2, w float64) {
	p.b.rConicTo(pt(x1, y1), pt(x2, y2), safeNarrow(w))
}

// radiansToDegrees32 is path.cc's SafeNarrow(angle) * 180.0f / (float)M_PI.
func radiansToDegrees32(angle float64) float32 {
	return mul32(safeNarrow(angle), 180) / mPIf
}

// ArcTo adds a segment of the oval inscribed in rect, from startAngle
// (radians, clockwise from the positive x axis) sweeping sweepAngle
// radians. With forceMoveTo false, a line connects the current point to
// the arc's start.
func (p *Path) ArcTo(rect Rect, startAngle, sweepAngle float64, forceMoveTo bool) {
	assertRectValid(rect)
	p.b.arcTo(skRectOf(rect), radiansToDegrees32(startAngle), radiansToDegrees32(sweepAngle), forceMoveTo)
}

// ArcToPointOptions are arcToPoint's named parameters. A nil Clockwise is
// Dart's default, true.
type ArcToPointOptions struct {
	Radius    Radius
	Rotation  float64
	LargeArc  bool
	Clockwise *bool
}

// RelativeArcToPointOptions are relativeArcToPoint's named parameters.
type RelativeArcToPointOptions = ArcToPointOptions

func arcOpts(opts []ArcToPointOptions) (ArcToPointOptions, bool) {
	var o ArcToPointOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	cw := true
	if o.Clockwise != nil {
		cw = *o.Clockwise
	}
	return o, cw
}

// ArcToPoint appends up to four conic curves weighted to describe an oval
// of the given radius and rotation (degrees), from the current point to
// arcEnd (SVG's elliptical arc). A zero radius, or arcEnd equal to the
// current point, gives a straight line.
func (p *Path) ArcToPoint(arcEnd Offset, opts ...ArcToPointOptions) {
	o, cw := arcOpts(opts)
	assertOffsetValid(arcEnd)
	assertRadiusValid(o.Radius)
	p.b.arcToSVG(pt(o.Radius.X, o.Radius.Y), safeNarrow(o.Rotation), o.LargeArc, cw, pt(arcEnd.Dx, arcEnd.Dy))
}

// RelativeArcToPoint is ArcToPoint with arcEndDelta an offset from the
// current point.
func (p *Path) RelativeArcToPoint(arcEndDelta Offset, opts ...RelativeArcToPointOptions) {
	o, cw := arcOpts(opts)
	assertOffsetValid(arcEndDelta)
	assertRadiusValid(o.Radius)
	p.b.rArcToSVG(pt(o.Radius.X, o.Radius.Y), safeNarrow(o.Rotation), o.LargeArc, cw,
		pt(arcEndDelta.Dx, arcEndDelta.Dy))
}

// AddRect adds a new subpath of four lines outlining rect.
func (p *Path) AddRect(rect Rect) {
	assertRectValid(rect)
	p.b.addRect(skRectOf(rect), true, 0)
}

// AddOval adds a new subpath of the ellipse inscribed in oval.
func (p *Path) AddOval(oval Rect) {
	assertRectValid(oval)
	p.b.addOval(skRectOf(oval), true, 1)
}

// AddArc adds a new subpath of an arc of the oval inscribed in oval, from
// startAngle sweeping sweepAngle (radians).
func (p *Path) AddArc(oval Rect, startAngle, sweepAngle float64) {
	assertRectValid(oval)
	p.b.addArc(skRectOf(oval), radiansToDegrees32(startAngle), radiansToDegrees32(sweepAngle))
}

// AddPolygon adds a new subpath of line segments through points, closed
// when close is true.
func (p *Path) AddPolygon(points []Offset, close bool) {
	pts := make([]skPoint, len(points))
	for i, o := range points {
		assertOffsetValid(o)
		pts[i] = skPoint{float32(o.Dx), float32(o.Dy)} // Float32List storage
	}
	p.b.addPolygon(pts, close)
}

// AddRRect adds a new subpath outlining the rounded rectangle.
func (p *Path) AddRRect(rrect RRect) {
	if rrect.HasNaN() {
		panic(dart.NewAssertionError("RRect argument contained a NaN value."))
	}
	r := rrect.rrectLike
	v := [12]float32{float32(r.Left), float32(r.Top), float32(r.Right), float32(r.Bottom),
		float32(r.TlRadiusX), float32(r.TlRadiusY), float32(r.TrRadiusX), float32(r.TrRadiusY),
		float32(r.BrRadiusX), float32(r.BrRadiusY), float32(r.BlRadiusX), float32(r.BlRadiusY)}
	p.b.addRRect(rrectToSk(v), true, 6)
}

// AddRSuperellipse adds a new subpath outlining the rounded superellipse.
func (p *Path) AddRSuperellipse(rsuperellipse RSuperellipse) {
	if rsuperellipse.HasNaN() {
		panic(dart.NewAssertionError("RSuperellipse argument contained a NaN value."))
	}
	p.b.addPath(rsuperellipsePath(rsuperellipse), skMatrixIdentity(), false)
}

// AddPathOptions are addPath's named parameters: Matrix4, when not nil,
// is a column-major 4x4 matrix applied to path (then offset).
type AddPathOptions struct{ Matrix4 []float64 }

// ExtendWithPathOptions are extendWithPath's named parameters.
type ExtendWithPathOptions = AddPathOptions

func (p *Path) addPathMode(path *Path, offset Offset, opts []AddPathOptions, extend bool) {
	assertOffsetValid(offset)
	var m skMatrix
	if len(opts) > 0 && opts[0].Matrix4 != nil {
		assertMatrix4Valid(opts[0].Matrix4)
		m = skMatrixFromMatrix4(opts[0].Matrix4)
		m.m[kMTransX] = m.m[kMTransX] + safeNarrow(offset.Dx)
		m.m[kMTransY] = m.m[kMTransY] + safeNarrow(offset.Dy)
		m.mask = kUnknownMask
	} else {
		m = skMatrixTranslate(safeNarrow(offset.Dx), safeNarrow(offset.Dy))
	}
	p.b.addPath(path.b.snapshot(), m, extend)
}

// AddPath adds the subpaths of path, offset by offset (and transformed by
// the optional matrix first).
func (p *Path) AddPath(path *Path, offset Offset, opts ...AddPathOptions) {
	p.addPathMode(path, offset, opts, false)
}

// ExtendWithPath adds path's subpaths, offset by offset (and transformed
// by the optional matrix), connecting its first point to this path's
// current point with a line.
func (p *Path) ExtendWithPath(path *Path, offset Offset, opts ...ExtendWithPathOptions) {
	p.addPathMode(path, offset, opts, true)
}

// Close closes the last subpath, as if a straight line had been drawn
// from the current point to the first point of the subpath.
func (p *Path) Close() { p.b.close() }

// Reset clears the path of all subpaths and points. As in the engine
// (SkPathBuilder::reset), the fill type reverts to nonZero.
func (p *Path) Reset() { p.b.reset() }

// Contains tests whether point is within the area the path encloses, by
// its fill type. Points on the edge may or may not be inside.
func (p *Path) Contains(point Offset) bool {
	assertOffsetValid(point)
	return p.b.contains(pt(point.Dx, point.Dy))
}

// Shift returns a copy of the path with every segment translated by
// offset.
func (p *Path) Shift(offset Offset) *Path {
	assertOffsetValid(offset)
	n := &Path{b: p.b.clone()}
	n.b.offset(safeNarrow(offset.Dx), safeNarrow(offset.Dy))
	return n
}

// Transform returns a copy of the path with every segment transformed by
// the column-major 4x4 matrix4.
func (p *Path) Transform(matrix4 []float64) *Path {
	assertMatrix4Valid(matrix4)
	n := &Path{b: p.b.clone()}
	n.b.transform(skMatrixFromMatrix4(matrix4))
	return n
}

// GetBounds returns the bounds of the path's control points (not the
// tight bounds of its curves); Rect.zero when it has none or one isn't
// finite.
func (p *Path) GetBounds() Rect {
	r, ok := p.b.computeFiniteBounds()
	if !ok {
		r = skRect{}
	}
	return Rect{float64(r.l), float64(r.t), float64(r.r), float64(r.b)}
}

// unimplementedPathError reports a Path feature whose engine code is not
// ported (an ENGINE GAP).
func unimplementedPathError(what string) dart.UnimplementedError {
	return dart.UnimplementedError{Message: "ENGINE GAP: " + what}
}

// PathCombine is Path.combine(operation, path1, path2): the boolean
// operation of two paths. ENGINE GAP: the engine runs Skia's PathOps
// (SkPathOps, ~18,000 lines of C++), which is not ported; this panics with
// dart.UnimplementedError.
func PathCombine(operation PathOperation, path1, path2 *Path) *Path {
	panic(unimplementedPathError("Path.combine needs Skia's PathOps (SkPathOps), which is not ported"))
}

// ComputeMetricsOptions are computeMetrics' named parameters.
type ComputeMetricsOptions struct{ ForceClosed bool }

// ComputeMetrics creates a PathMetrics over the path's contours as it is
// now (later changes to the path don't affect it). With ForceClosed each
// contour is measured as if closed.
func (p *Path) ComputeMetrics(opts ...ComputeMetricsOptions) *PathMetrics {
	force := len(opts) > 0 && opts[0].ForceClosed
	return newPathMetrics(p, force)
}

// String is Dart's toString(): "Path".
func (p *Path) String() string { return "Path" }

// emit sends the path's verbs to s in order, as the engine's DlPath holds
// them (an empty path when a point isn't finite); each conic is converted
// to quadratics (SkAutoConicToQuads) within tol path units.
func (p *Path) emit(s pathSink, tol float32) {
	d := p.b.snapshot()
	d.iterate(func(v uint8, pts []skPoint, w float32) {
		switch v {
		case verbMove:
			s.MoveTo(pts[0].x, pts[0].y)
		case verbLine:
			s.LineTo(pts[1].x, pts[1].y)
		case verbQuad:
			s.QuadTo(pts[1].x, pts[1].y, pts[2].x, pts[2].y)
		case verbConic:
			q := conicToQuads(pts[0], pts[1], pts[2], w, tol)
			for i := 1; i+1 < len(q); i += 2 {
				s.QuadTo(q[i].x, q[i].y, q[i+1].x, q[i+1].y)
			}
		case verbCubic:
			s.CubicTo(pts[1].x, pts[1].y, pts[2].x, pts[2].y, pts[3].x, pts[3].y)
		case verbClose:
			s.Close()
		}
	})
}

// pathSink receives a path's outline for drawing (impeller.PathBuilder
// satisfies it).
type pathSink interface {
	MoveTo(x, y float32)
	LineTo(x, y float32)
	QuadTo(x1, y1, x2, y2 float32)
	CubicTo(x1, y1, x2, y2, x3, y3 float32)
	Close()
}
