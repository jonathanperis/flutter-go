package painting

// Port of star_border.dart (Flutter 3.50.1): StarBorder and its private
// _StarGenerator.

import (
	"math"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

// dartPiValue keeps math.Pi a variable so expressions with it are rounded
// step by step, as Dart's doubles are (Go folds constant expressions
// exactly).
var dartPiValue = math.Pi

var (
	kRadToDeg = 180 / dartPiValue
	kDegToRad = dartPiValue / 180
)

// StarBorder is a border that fits a star or polygon-shaped border within
// the rectangle of the widget it applies to. It is a comparable value;
// build it with NewStarBorder or NewStarBorderPolygon.
type StarBorder struct {
	// Side is the border outline's color and weight.
	Side BorderSide
	// Points is the number of points in this star (or sides of the
	// polygon); fractional values are allowed.
	Points float64
	// innerRadiusRatio nil: a polygon (cos(pi / points)).
	innerRadiusRatio *float64
	// PointRounding is the amount of rounding on the points (0..1).
	PointRounding float64
	// ValleyRounding is the amount of rounding of the valleys (0..1).
	ValleyRounding float64
	// rotationRadians is the rotation in radians (Rotation() in degrees).
	rotationRadians float64
	// Squash: how much of the aspect ratio of the attached widget to take
	// on (0..1).
	Squash float64
}

// StarBorderOptions are StarBorder's named parameters (nil: Dart's
// defaults: 5 points, innerRadiusRatio 0.4, no rounding, rotation and
// squash 0, BorderSide.none).
type StarBorderOptions struct {
	Side             *BorderSide
	Points           *float64
	InnerRadiusRatio *float64
	PointRounding    float64
	ValleyRounding   float64
	Rotation         float64
	Squash           float64
}

// NewStarBorder is StarBorder({side, points, innerRadiusRatio,
// pointRounding, valleyRounding, rotation, squash}); it panics on Dart's
// assertions.
func NewStarBorder(opts ...StarBorderOptions) StarBorder {
	var o StarBorderOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	points, ratio := 5.0, 0.4
	if o.Points != nil {
		points = *o.Points
	}
	if o.InnerRadiusRatio != nil {
		ratio = *o.InnerRadiusRatio
	}
	return newStarBorder(sideOrNone(o.Side), points, ratio, o.PointRounding, o.ValleyRounding, o.Rotation, o.Squash)
}

func starAssert(ok bool, message string) {
	if !ok {
		panic(dart.NewAssertionError(message))
	}
}

func newStarBorder(side BorderSide, points, innerRadiusRatio, pointRounding, valleyRounding, rotation, squash float64) StarBorder {
	starAssert(squash >= 0, "squash >= 0")
	starAssert(squash <= 1, "squash <= 1")
	starAssert(pointRounding >= 0, "pointRounding >= 0")
	starAssert(pointRounding <= 1, "pointRounding <= 1")
	starAssert(valleyRounding >= 0, "valleyRounding >= 0")
	starAssert(valleyRounding <= 1, "valleyRounding <= 1")
	starAssert(valleyRounding+pointRounding <= 1, "The sum of valleyRounding ("+dart.DoubleToString(valleyRounding)+
		") and pointRounding ("+dart.DoubleToString(pointRounding)+") must not exceed one.")
	starAssert(innerRadiusRatio >= 0, "innerRadiusRatio >= 0")
	starAssert(innerRadiusRatio <= 1, "innerRadiusRatio <= 1")
	starAssert(points >= 2, "points >= 2")
	r := innerRadiusRatio
	return StarBorder{Side: side, Points: points, innerRadiusRatio: &r, PointRounding: pointRounding,
		ValleyRounding: valleyRounding, rotationRadians: rotation * kDegToRad, Squash: squash}
}

// StarBorderPolygonOptions are StarBorder.polygon's named parameters (nil
// Sides: 5).
type StarBorderPolygonOptions struct {
	Side          *BorderSide
	Sides         *float64
	PointRounding float64
	Rotation      float64
	Squash        float64
}

// NewStarBorderPolygon is StarBorder.polygon({side, sides, pointRounding,
// rotation, squash}).
func NewStarBorderPolygon(opts ...StarBorderPolygonOptions) StarBorder {
	var o StarBorderPolygonOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	sides := 5.0
	if o.Sides != nil {
		sides = *o.Sides
	}
	starAssert(o.Squash >= 0, "squash >= 0")
	starAssert(o.Squash <= 1, "squash <= 1")
	starAssert(o.PointRounding >= 0, "pointRounding >= 0")
	starAssert(o.PointRounding <= 1, "pointRounding <= 1")
	starAssert(sides >= 2, "sides >= 2")
	return StarBorder{Side: sideOrNone(o.Side), Points: sides, PointRounding: o.PointRounding,
		rotationRadians: o.Rotation * kDegToRad, Squash: o.Squash}
}

// InnerRadiusRatio is the ratio of the inner radius of a star to its
// outer radius (for a polygon, cos(pi / points)).
func (s StarBorder) InnerRadiusRatio() float64 {
	if s.innerRadiusRatio != nil {
		return *s.innerRadiusRatio
	}
	return dart.Cos(dartPiValue / s.Points)
}

// Rotation is the rotation in clockwise degrees around the center.
func (s StarBorder) Rotation() float64 { return s.rotationRadians * kRadToDeg }

// GetSide is the border's side.
func (s StarBorder) GetSide() BorderSide { return s.Side }

// Dimensions is OutlinedBorder's.
func (s StarBorder) Dimensions() EdgeInsetsGeometry { return outlinedDimensions(s.Side) }

// AddBorder returns nil.
func (s StarBorder) AddBorder(other ShapeBorder, reversed bool) ShapeBorder { return nil }

// Add is operator +.
func (s StarBorder) Add(other ShapeBorder) ShapeBorder { return shapeBorderPlus(s, other) }

// Scale scales the side by t.
func (s StarBorder) Scale(t float64) ShapeBorder {
	return newStarBorder(s.Side.Scale(t), s.Points, s.InnerRadiusRatio(), s.PointRounding, s.ValleyRounding,
		s.Rotation(), s.Squash)
}

func starTwoPhaseLerp(t, split float64, first, second func(t float64) ShapeBorder) ShapeBorder {
	if t < split {
		return first(t * (1 / split))
	}
	t = float64((1 / (1.0 - split)) * (t - split))
	return second(t)
}

func starLerp(a, b, t float64) float64 { return borderLerpDouble(a, b, t) }

// LerpFrom interpolates from a StarBorder, CircleBorder, StadiumBorder,
// RoundedRectangleBorder (the last two in phases through a circle) or nil.
func (s StarBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if t == 0 {
		return a
	}
	if t == 1.0 {
		return s
	}
	if x, ok := a.(StarBorder); ok {
		return newStarBorder(BorderSideLerp(x.Side, s.Side, t), starLerp(x.Points, s.Points, t),
			starLerp(x.InnerRadiusRatio(), s.InnerRadiusRatio(), t), starLerp(x.PointRounding, s.PointRounding, t),
			starLerp(x.ValleyRounding, s.ValleyRounding, t), starLerp(x.rotationRadians, s.rotationRadians, t)*kRadToDeg,
			starLerp(x.Squash, s.Squash, t))
	}
	if x, ok := asCircleBorder(a); ok {
		if s.Points >= 2.5 {
			lerpedPoints := starLerp(math.Round(s.Points), s.Points, t)
			return newStarBorder(BorderSideLerp(x.Side, s.Side, t), lerpedPoints,
				starLerp(dart.Cos(dartPiValue/lerpedPoints), s.InnerRadiusRatio(), t), starLerp(1.0, s.PointRounding, t),
				starLerp(0.0, s.ValleyRounding, t), s.Rotation(), starLerp(x.Eccentricity, s.Squash, t))
		}
		lerpedPoints := starLerp(s.Points, 2, t)
		return newStarBorder(BorderSideLerp(x.Side, s.Side, t), lerpedPoints, starLerp(1, s.InnerRadiusRatio(), t),
			starLerp(0.5, s.PointRounding, t), starLerp(0.5, s.ValleyRounding, t), s.Rotation(), starLerp(x.Eccentricity, s.Squash, t))
	}
	if x, ok := a.(StadiumBorder); ok {
		lerpedSide := BorderSideLerp(x.Side, s.Side, t)
		circle := CircleBorder{Side: lerpedSide}
		return starTwoPhaseLerp(t, 0.5,
			func(t float64) ShapeBorder { return x.LerpTo(circle, t) },
			func(t float64) ShapeBorder { return s.LerpFrom(circle, t) })
	}
	if x, ok := a.(RoundedRectangleBorder); ok {
		lerpedSide := BorderSideLerp(x.Side, s.Side, t)
		circle := CircleBorder{Side: lerpedSide}
		stadium := StadiumBorder{lerpedSide}
		return starTwoPhaseLerp(t, 1.0/3,
			func(t float64) ShapeBorder { return stadium.LerpFrom(x, t) },
			func(t float64) ShapeBorder {
				return starTwoPhaseLerp(t, 0.5,
					func(t float64) ShapeBorder { return stadium.LerpTo(circle, t) },
					func(t float64) ShapeBorder { return s.LerpFrom(circle, t) })
			})
	}
	return shapeBorderLerpFrom(s, a, t)
}

// LerpTo interpolates to a StarBorder, CircleBorder, StadiumBorder,
// RoundedRectangleBorder or nil.
func (s StarBorder) LerpTo(b ShapeBorder, t float64) ShapeBorder {
	if t == 0 {
		return s
	}
	if t == 1.0 {
		return b
	}
	if x, ok := b.(StarBorder); ok {
		return newStarBorder(BorderSideLerp(s.Side, x.Side, t), starLerp(s.Points, x.Points, t),
			starLerp(s.InnerRadiusRatio(), x.InnerRadiusRatio(), t), starLerp(s.PointRounding, x.PointRounding, t),
			starLerp(s.ValleyRounding, x.ValleyRounding, t), starLerp(s.rotationRadians, x.rotationRadians, t)*kRadToDeg,
			starLerp(s.Squash, x.Squash, t))
	}
	if x, ok := asCircleBorder(b); ok {
		if s.Points >= 2.5 {
			lerpedPoints := starLerp(s.Points, math.Round(s.Points), t)
			return newStarBorder(BorderSideLerp(s.Side, x.Side, t), lerpedPoints,
				starLerp(s.InnerRadiusRatio(), dart.Cos(dartPiValue/lerpedPoints), t), starLerp(s.PointRounding, 1.0, t),
				starLerp(s.ValleyRounding, 0.0, t), s.Rotation(), starLerp(s.Squash, x.Eccentricity, t))
		}
		lerpedPoints := starLerp(s.Points, 2, t)
		return newStarBorder(BorderSideLerp(s.Side, x.Side, t), lerpedPoints, starLerp(s.InnerRadiusRatio(), 1, t),
			starLerp(s.PointRounding, 0.5, t), starLerp(s.ValleyRounding, 0.5, t), s.Rotation(), starLerp(s.Squash, x.Eccentricity, t))
	}
	if x, ok := b.(StadiumBorder); ok {
		lerpedSide := BorderSideLerp(s.Side, x.Side, t)
		circle := CircleBorder{Side: lerpedSide}
		return starTwoPhaseLerp(t, 0.5,
			func(t float64) ShapeBorder { return s.LerpTo(circle, t) },
			func(t float64) ShapeBorder { return x.LerpFrom(circle, t) })
	}
	if x, ok := b.(RoundedRectangleBorder); ok {
		lerpedSide := BorderSideLerp(s.Side, x.Side, t)
		circle := CircleBorder{Side: lerpedSide}
		stadium := StadiumBorder{lerpedSide}
		return starTwoPhaseLerp(t, 2.0/3,
			func(t float64) ShapeBorder {
				return starTwoPhaseLerp(t, 0.5,
					func(t float64) ShapeBorder { return s.LerpTo(circle, t) },
					func(t float64) ShapeBorder { return stadium.LerpFrom(circle, t) })
			},
			func(t float64) ShapeBorder { return stadium.LerpTo(x, t) })
	}
	return shapeBorderLerpTo(s, b, t)
}

// StarBorderCopyWithOptions are copyWith's named parameters (nil: keep).
type StarBorderCopyWithOptions struct {
	Side             *BorderSide
	Points           *float64
	InnerRadiusRatio *float64
	PointRounding    *float64
	ValleyRounding   *float64
	Rotation         *float64
	Squash           *float64
}

func floatOr(p *float64, d float64) float64 {
	if p != nil {
		return *p
	}
	return d
}

// CopyWith returns a copy with the given fields replaced (always a star:
// a polygon's computed innerRadiusRatio becomes explicit, as in Dart).
func (s StarBorder) CopyWith(opts ...StarBorderCopyWithOptions) StarBorder {
	var o StarBorderCopyWithOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	side := s.Side
	if o.Side != nil {
		side = *o.Side
	}
	return newStarBorder(side, floatOr(o.Points, s.Points), floatOr(o.InnerRadiusRatio, s.InnerRadiusRatio()),
		floatOr(o.PointRounding, s.PointRounding), floatOr(o.ValleyRounding, s.ValleyRounding),
		floatOr(o.Rotation, s.Rotation()), floatOr(o.Squash, s.Squash))
}

// CopyWithSide is copyWith({side}).
func (s StarBorder) CopyWithSide(side *BorderSide) OutlinedBorder {
	return s.CopyWith(StarBorderCopyWithOptions{Side: side})
}

func (s StarBorder) generator() starGenerator {
	return newStarGenerator(s.Points, s.rotationRadians, s.InnerRadiusRatio(), s.PointRounding, s.ValleyRounding, s.Squash)
}

// GetInnerPath is the star in the rect deflated by the stroke inset.
func (s StarBorder) GetInnerPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return s.generator().generate(rect.Deflate(s.Side.StrokeInset()))
}

// GetOuterPath is the star.
func (s StarBorder) GetOuterPath(rect ui.Rect, textDirection *ui.TextDirection) *ui.Path {
	return s.generator().generate(rect)
}

// HitTest is ShapeBorder's: inside the outer path.
func (s StarBorder) HitTest(rect ui.Rect, position ui.Offset, textDirection *ui.TextDirection) bool {
	return shapeBorderHitTest(s, rect, position, textDirection)
}

// PaintInterior panics (preferPaintInterior is false), as Dart asserts.
func (s StarBorder) PaintInterior(canvas ui.Canvas, rect ui.Rect, paint *ui.Paint, textDirection *ui.TextDirection) {
	shapeBorderPaintInterior(s)
}

// PreferPaintInterior is false.
func (s StarBorder) PreferPaintInterior() bool { return false }

// Paint strokes the star with the side.
func (s StarBorder) Paint(canvas ui.Canvas, rect ui.Rect, textDirection *ui.TextDirection) {
	if s.Side.Style != BorderStyleSolid {
		return
	}
	adjustedRect := rect.Inflate(s.Side.StrokeOffset() / 2)
	canvas.DrawPath(s.generator().generate(adjustedRect), s.Side.ToPaint())
}

// Equal is Dart's == (Go's == also compares the innerRadiusRatio pointer).
func (s StarBorder) Equal(other ShapeBorder) bool {
	o, ok := other.(StarBorder)
	if !ok {
		return false
	}
	ratioEqual := (o.innerRadiusRatio == nil) == (s.innerRadiusRatio == nil) &&
		(o.innerRadiusRatio == nil || *o.innerRadiusRatio == *s.innerRadiusRatio)
	return o.Side == s.Side && o.Points == s.Points && ratioEqual && o.PointRounding == s.PointRounding &&
		o.ValleyRounding == s.ValleyRounding && o.rotationRadians == s.rotationRadians && o.Squash == s.Squash
}

func (s StarBorder) String() string {
	return "StarBorder(" + s.Side.String() + ", points: " + dart.DoubleToString(s.Points) +
		", innerRadiusRatio: " + dart.DoubleToString(s.InnerRadiusRatio()) + ")"
}

type starPointInfo struct {
	valley, point, valleyArc1, pointArc1, pointArc2, valleyArc2 ui.Offset
}

// starGenerator is _StarGenerator.
type starGenerator struct {
	points, innerRadiusRatio, pointRounding, valleyRounding, rotation, squash float64
}

func newStarGenerator(points, rotation, innerRadiusRatio, pointRounding, valleyRounding, squash float64) starGenerator {
	starAssert(points > 1, "points > 1")
	starAssert(innerRadiusRatio <= 1, "innerRadiusRatio <= 1")
	starAssert(innerRadiusRatio >= 0, "innerRadiusRatio >= 0")
	starAssert(squash >= 0, "squash >= 0")
	starAssert(squash <= 1, "squash <= 1")
	starAssert(pointRounding >= 0, "pointRounding >= 0")
	starAssert(pointRounding <= 1, "pointRounding <= 1")
	starAssert(valleyRounding >= 0, "valleyRounding >= 0")
	starAssert(valleyRounding <= 1, "valleyRounding <= 1")
	starAssert(pointRounding+valleyRounding <= 1, "pointRounding + valleyRounding <= 1")
	return starGenerator{points, innerRadiusRatio, pointRounding, valleyRounding, rotation, squash}
}

func (g starGenerator) generate(rect ui.Rect) *ui.Path {
	radius := rect.ShortestSide() / 2
	center := rect.Center()
	const minInnerRadiusRatio = .002
	mappedInnerRadiusRatio := float64(g.innerRadiusRatio*(1.0-minInnerRadiusRatio)) + minInnerRadiusRatio
	var points []starPointInfo
	maxDiameter := float64(2.0 * g.generatePoints(&points, center, radius, radius*mappedInnerRadiusRatio))
	path := ui.NewPath()
	g.drawPoints(path, points)
	scale := ui.NewOffset(rect.Width()/maxDiameter, rect.Height()/maxDiameter)
	if rect.ShortestSide() == rect.Width() {
		scale = ui.NewOffset(scale.Dx, float64(g.squash*scale.Dy)+float64((1-g.squash)*scale.Dx))
	} else {
		scale = ui.NewOffset(float64(g.squash*scale.Dx)+float64((1-g.squash)*scale.Dy), scale.Dy)
	}
	squashMatrix := vm.NewMatrix4TranslationValues(rect.Center().Dx, rect.Center().Dy, 0)
	squashMatrix.Multiply(vm.NewMatrix4Diagonal3Values(scale.Dx, scale.Dy, 1))
	squashMatrix.Multiply(vm.NewMatrix4RotationZ(g.rotation))
	squashMatrix.Multiply(vm.NewMatrix4TranslationValues(-rect.Center().Dx, -rect.Center().Dy, 0))
	return path.Transform(squashMatrix.Storage())
}

func (g starGenerator) generatePoints(pointList *[]starPointInfo, center ui.Offset, radius, innerRadius float64) float64 {
	step := dartPiValue / g.points
	angle := -dartPiValue/2 - step
	valley := ui.NewOffset(center.Dx+float64(dart.Cos(angle)*innerRadius), center.Dy+float64(dart.Sin(angle)*innerRadius))

	getCurveMidpoint := func(a, b, c, a1, c1 ui.Offset) ui.Offset {
		angle := starGetAngle(a, b, c)
		w := starGetWeight(angle) / 2
		return a1.Div(4).Add(b.Mul(w)).Add(c1.Div(4)).Div(0.5 + w)
	}

	addPoint := func(pointAngle, pointStep, pointRadius, pointInnerRadius float64) float64 {
		pointAngle += pointStep
		point := ui.NewOffset(center.Dx+float64(dart.Cos(pointAngle)*pointRadius),
			center.Dy+float64(dart.Sin(pointAngle)*pointRadius))
		pointAngle += pointStep
		nextValley := ui.NewOffset(center.Dx+float64(dart.Cos(pointAngle)*pointInnerRadius),
			center.Dy+float64(dart.Sin(pointAngle)*pointInnerRadius))
		valleyArc1 := valley.Add(point.Sub(valley).Mul(g.valleyRounding))
		pointArc1 := point.Add(valley.Sub(point).Mul(g.pointRounding))
		pointArc2 := point.Add(nextValley.Sub(point).Mul(g.pointRounding))
		valleyArc2 := nextValley.Add(point.Sub(nextValley).Mul(g.valleyRounding))
		*pointList = append(*pointList, starPointInfo{valley: valley, point: point, valleyArc1: valleyArc1,
			pointArc1: pointArc1, pointArc2: pointArc2, valleyArc2: valleyArc2})
		valley = nextValley
		return pointAngle
	}

	remainder := g.points - math.Trunc(g.points)
	hasIntegerSides := remainder < 1e-6
	wholeSides := g.points
	if !hasIntegerSides {
		wholeSides = g.points - 1
	}
	for i := 0; float64(i) < wholeSides; i++ {
		angle = addPoint(angle, step, radius, innerRadius)
	}

	thisPoint, nextPoint := (*pointList)[0], (*pointList)[1]
	pointMidpoint := getCurveMidpoint(thisPoint.valley, thisPoint.point, nextPoint.valley, thisPoint.pointArc1, thisPoint.pointArc2)
	valleyMidpoint := getCurveMidpoint(thisPoint.point, nextPoint.valley, nextPoint.point, thisPoint.valleyArc2, nextPoint.valleyArc1)
	valleyRadius := valleyMidpoint.Sub(center).Distance()
	pointRadius := pointMidpoint.Sub(center).Distance()

	if !hasIntegerSides {
		effectiveInnerRadius := math.Max(valleyRadius, innerRadius)
		endingRadius := effectiveInnerRadius + float64(remainder*(radius-effectiveInnerRadius))
		addPoint(angle, step*remainder, endingRadius, innerRadius)
	}
	return ui.ClampDouble(math.Max(valleyRadius, pointRadius), math.SmallestNonzeroFloat64, math.MaxFloat64)
}

func (g starGenerator) drawPoints(path *ui.Path, points []starPointInfo) {
	startingPoint := points[0].pointArc1
	path.MoveTo(startingPoint.Dx, startingPoint.Dy)
	pointAngle := starGetAngle(points[0].valley, points[0].point, points[1].valley)
	pointWeight := starGetWeight(pointAngle)
	valleyAngle := starGetAngle(points[1].point, points[1].valley, points[0].point)
	valleyWeight := starGetWeight(valleyAngle)

	for i := range points {
		point := points[i]
		nextPoint := points[(i+1)%len(points)]
		path.LineTo(point.pointArc1.Dx, point.pointArc1.Dy)
		if pointAngle != 180 && pointAngle != 0 {
			path.ConicTo(point.point.Dx, point.point.Dy, point.pointArc2.Dx, point.pointArc2.Dy, pointWeight)
		} else {
			path.LineTo(point.pointArc2.Dx, point.pointArc2.Dy)
		}
		path.LineTo(point.valleyArc2.Dx, point.valleyArc2.Dy)
		if valleyAngle != 180 && valleyAngle != 0 {
			path.ConicTo(nextPoint.valley.Dx, nextPoint.valley.Dy, nextPoint.valleyArc1.Dx, nextPoint.valleyArc1.Dy, valleyWeight)
		} else {
			path.LineTo(nextPoint.valleyArc1.Dx, nextPoint.valleyArc1.Dy)
		}
	}
	path.Close()
}

func starGetWeight(angle float64) float64 {
	return dart.Cos(dart.Mod(angle/2, dartPiValue/2))
}

func starGetAngle(a, b, c ui.Offset) float64 {
	if a == c || b == c || b == a {
		return 0
	}
	u := a.Sub(b)
	v := c.Sub(b)
	dot := float64(u.Dx*v.Dx) + float64(u.Dy*v.Dy)
	m1 := math.Inf(1)
	if b.Dx != a.Dx {
		m1 = -u.Dy / -u.Dx
	}
	m2 := math.Inf(1)
	if b.Dx != c.Dx {
		m2 = -v.Dy / -v.Dx
	}
	angle := math.Abs(dart.Atan2(m1-m2, 1+float64(m1*m2)))
	if dot < 0 {
		angle += dartPiValue
	}
	return angle
}
