package ui

// Port of painting.dart's Canvas (Flutter 3.50.1) and the engine's
// lib/ui/painting/canvas.cc. Flutter's Canvas records into the engine's
// DisplayListBuilder; here every call goes to the Impeller SDK's display
// list builder (the same DisplayListBuilder inside), and the clip bounds
// the builder tracks are recomputed in Go (painting_canvas_state.go)
// because the SDK doesn't expose them.

import (
	"math"
	"strconv"

	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/impeller"
)

// Canvas is an interface for recording graphical operations (Dart's
// abstract class; Flutter's tests implement it, so it is an interface).
// NewCanvas returns the engine's implementation.
type Canvas interface {
	Save()
	SaveLayer(bounds *Rect, paint *Paint)
	Restore()
	RestoreToCount(count int)
	GetSaveCount() int
	Translate(dx, dy float64)
	// Scale is scale(sx, [sy]): sy defaults to sx.
	Scale(sx float64, sy ...float64)
	Rotate(radians float64)
	Skew(sx, sy float64)
	Transform(matrix4 []float64)
	GetTransform() []float64
	ClipRect(rect Rect, opts ...CanvasClipRectOptions)
	ClipRRect(rrect RRect, opts ...CanvasClipOptions)
	ClipRSuperellipse(rsuperellipse RSuperellipse, opts ...CanvasClipOptions)
	ClipPath(path *Path, opts ...CanvasClipOptions)
	GetLocalClipBounds() Rect
	GetDestinationClipBounds() Rect
	DrawColor(color Color, blendMode BlendMode)
	DrawLine(p1, p2 Offset, paint *Paint)
	DrawPaint(paint *Paint)
	DrawRect(rect Rect, paint *Paint)
	DrawRRect(rrect RRect, paint *Paint)
	DrawDRRect(outer, inner RRect, paint *Paint)
	DrawRSuperellipse(rsuperellipse RSuperellipse, paint *Paint)
	DrawOval(rect Rect, paint *Paint)
	DrawCircle(c Offset, radius float64, paint *Paint)
	DrawArc(rect Rect, startAngle, sweepAngle float64, useCenter bool, paint *Paint)
	DrawPath(path *Path, paint *Paint)
	DrawImage(image *Image, offset Offset, paint *Paint)
	DrawImageRect(image *Image, src, dst Rect, paint *Paint)
	DrawImageNine(image *Image, center, dst Rect, paint *Paint)
	DrawPicture(picture *Picture)
	DrawPoints(pointMode PointMode, points []Offset, paint *Paint)
	DrawRawPoints(pointMode PointMode, points []float32, paint *Paint)
	DrawVertices(vertices *Vertices, blendMode BlendMode, paint *Paint)
	// DrawAtlas's colors, blendMode and cullRect are nullable (nil).
	DrawAtlas(atlas *Image, transforms []RSTransform, rects []Rect, colors []Color, blendMode *BlendMode, cullRect *Rect, paint *Paint)
	DrawRawAtlas(atlas *Image, rstTransforms, rects []float32, colors []int32, blendMode *BlendMode, cullRect *Rect, paint *Paint)
	DrawShadow(path *Path, color Color, elevation float64, transparentOccluder bool)
	String() string
}

// CanvasClipRectOptions are clipRect's named arguments.
type CanvasClipRectOptions struct {
	ClipOp      *ClipOp // nil: ClipOp.intersect
	DoAntiAlias *bool   // nil: true
}

// CanvasClipOptions are clipRRect/clipRSuperellipse/clipPath's named
// arguments.
type CanvasClipOptions struct {
	DoAntiAlias *bool // nil: true
}

// EngineDevicePixelRatio is the implicit view's device pixel ratio the
// engine passes to drawShadow (canvas.cc reads the view's metrics, 1.0
// without a view). The embedder sets it.
var EngineDevicePixelRatio = 1.0

// nativeCanvas is _NativeCanvas plus the engine's Canvas.
type nativeCanvas struct {
	recorder *PictureRecorder
	b        *impeller.DisplayListBuilder
	stack    []clipState // the builder's save stack's global states
	ops      int         // recorded operations (for approximateBytesUsed)
}

// NewCanvas is Canvas(recorder, [cullRect]).
func NewCanvas(recorder *PictureRecorder, cullRect ...Rect) Canvas {
	if recorder.IsRecording() {
		panic(argumentError(`"recorder" must not already be associated with another Canvas.`))
	}
	cull := RectLargest
	if len(cullRect) > 0 {
		cull = cullRect[0]
	}
	c := &nativeCanvas{recorder: recorder}
	recorder.canvas = c
	dl := rectToDl(cull)
	c.b = impeller.NewDisplayListBuilderCull(impRect(dl))
	c.stack = []clipState{newClipState(dl)}
	recorder.cull = protectEmpty(dl)
	return c
}

func impRect(r dlRect) impeller.Rect {
	return impeller.Rect{X: r.l, Y: r.t, W: r.r - r.l, H: r.b - r.t}
}

// errNativePeerCollected is the StateError dart:ui throws when a native
// method is called on a Canvas whose recording has ended (its native peer
// is gone).
var errNativePeerCollected = dart.StateError{Message: "A Dart object attempted to access a native peer, but the native peer has been collected (nullptr). This is usually the result of calling methods on a native-backed object when the native resources have already been disposed."}

func (c *nativeCanvas) builder() *impeller.DisplayListBuilder {
	if c.recorder == nil {
		panic(errNativePeerCollected)
	}
	return c.b
}

func (c *nativeCanvas) top() *clipState { return &c.stack[len(c.stack)-1] }

func (c *nativeCanvas) matrix() dlMatrix { return c.b.GetTransform() }

func (c *nativeCanvas) String() string {
	return "Canvas(recording: " + strconv.FormatBool(c.recorder != nil) + ")"
}

// sorted is _NativeCanvas._sorted.
func sorted(r Rect) Rect {
	if r.IsEmpty() {
		r = NewRectFromLTRB(math.Min(r.Left, r.Right), math.Min(r.Top, r.Bottom),
			math.Max(r.Left, r.Right), math.Max(r.Top, r.Bottom))
	}
	return r
}

func (c *nativeCanvas) Save() {
	if b := c.builder(); b != nil {
		b.Save()
		c.stack = append(c.stack, *c.top())
	}
}

func (c *nativeCanvas) SaveLayer(bounds *Rect, paint *Paint) {
	b := c.builder()
	if b == nil {
		return
	}
	ip := paint.impellerPaint(flagsSaveLayer)
	defer ip.Release()
	st := *c.top()
	m := c.matrix()
	if paint.imageFilter != nil {
		// The layer's filter changes which device pixels can affect the
		// output (DlImageFilter::get_input_device_bounds).
		if in, ok := inputDeviceBounds(paint.imageFilter, roundOut(st.cull), &m); ok {
			st.resetDeviceCullRect(in)
		} else {
			st.resetDeviceCullRect(kMaxCullRect)
		}
	}
	if bounds == nil {
		r := impRect(kMaxCullRect)
		b.SaveLayer(&r, ip, nil)
	} else {
		rectIsValid(*bounds)
		dl := rectToDl(*bounds)
		r := impRect(dl)
		b.SaveLayer(&r, ip, nil)
		st.clipRect(&m, dl, ClipOpIntersect, false)
	}
	c.stack = append(c.stack, st)
	c.ops++
}

func (c *nativeCanvas) Restore() {
	if b := c.builder(); b != nil {
		b.Restore()
		if len(c.stack) > 1 {
			c.stack = c.stack[:len(c.stack)-1]
		}
	}
}

func (c *nativeCanvas) RestoreToCount(count int) {
	if b := c.builder(); b != nil && count < c.GetSaveCount() {
		for count < len(c.stack) && len(c.stack) > 1 {
			b.Restore()
			c.stack = c.stack[:len(c.stack)-1]
		}
	}
}

func (c *nativeCanvas) GetSaveCount() int {
	if b := c.builder(); b != nil {
		return b.SaveCount()
	}
	return 0
}

func (c *nativeCanvas) Translate(dx, dy float64) {
	if b := c.builder(); b != nil {
		b.Translate(safeNarrow(dx), safeNarrow(dy))
	}
}

func (c *nativeCanvas) Scale(sx float64, sy ...float64) {
	y := sx
	if len(sy) > 0 {
		y = sy[0]
	}
	if b := c.builder(); b != nil {
		b.Scale(safeNarrow(sx), safeNarrow(y))
	}
}

// radiansToDegrees is canvas.cc's SafeNarrow(r) * 180.0f / float(M_PI).
func radiansToDegrees(r float64) float32 {
	return float32(safeNarrow(r)*180.0) / float32(math.Pi)
}

func (c *nativeCanvas) Rotate(radians float64) {
	if b := c.builder(); b != nil {
		b.Rotate(radiansToDegrees(radians))
	}
}

func (c *nativeCanvas) Skew(sx, sy float64) {
	if b := c.builder(); b != nil {
		// The SDK has no skew; DisplayListBuilder::Skew multiplies by
		// Matrix::MakeSkew, which is what Transform does with it.
		x, y := safeNarrow(sx), safeNarrow(sy)
		if isInfOrNaN32(x) || isInfOrNaN32(y) || (x == 0 && y == 0) {
			return
		}
		b.Transform(impeller.Matrix{1, y, 0, 0, x, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1})
	}
}

func (c *nativeCanvas) Transform(matrix4 []float64) {
	if len(matrix4) != 16 {
		panic(argumentError(`"matrix4" must have 16 entries.`))
	}
	if b := c.builder(); b != nil {
		var m impeller.Matrix
		for i, v := range matrix4 {
			m[i] = safeNarrow(v)
		}
		b.Transform(m)
	}
}

func (c *nativeCanvas) GetTransform() []float64 {
	out := make([]float64, 16)
	if b := c.builder(); b != nil {
		m := b.GetTransform()
		for i, v := range m {
			out[i] = float64(v)
		}
	}
	return out
}

func boolOpt(p *bool) bool { return p == nil || *p }

func (c *nativeCanvas) ClipRect(rect Rect, opts ...CanvasClipRectOptions) {
	op, aa := ClipOpIntersect, true
	if len(opts) > 0 {
		if opts[0].ClipOp != nil {
			op = *opts[0].ClipOp
		}
		aa = boolOpt(opts[0].DoAntiAlias)
	}
	rectIsValid(rect)
	rect = sorted(rect)
	b := c.builder()
	if b == nil {
		return
	}
	dl := rectToDl(rect)
	if !dl.isFinite() {
		return
	}
	m := c.matrix()
	c.top().clipRect(&m, dl, op, aa)
	impOp := impeller.ClipIntersect
	if op == ClipOpDifference {
		impOp = impeller.ClipDifference
	}
	b.ClipRectOp(impRect(dl), impOp)
	c.ops++
}

// rrectRadii is the RRect's corner radii as the C API's radii.
func rrectRadii(r RRect) impeller.Radii {
	p := func(x, y float64) impeller.Point { return impeller.Point{X: float32(x), Y: float32(y)} }
	return impeller.Radii{TopLeft: p(r.TlRadiusX, r.TlRadiusY), TopRight: p(r.TrRadiusX, r.TrRadiusY),
		BottomLeft: p(r.BlRadiusX, r.BlRadiusY), BottomRight: p(r.BrRadiusX, r.BrRadiusY)}
}

func rrectBounds(r RRect) dlRect {
	return dlRect{float32(r.Left), float32(r.Top), float32(r.Right), float32(r.Bottom)}
}

func (c *nativeCanvas) clipBounds(bounds dlRect, aa bool) {
	m := c.matrix()
	// Every clip shape but a rect clips intersect here, which narrows the
	// cull rect to the shape's bounds (DisplayListMatrixClipState).
	if bounds.isFinite() {
		c.top().adjustCullRect(&m, bounds, ClipOpIntersect, aa)
	}
}

func (c *nativeCanvas) ClipRRect(rrect RRect, opts ...CanvasClipOptions) {
	aa := len(opts) == 0 || boolOpt(opts[0].DoAntiAlias)
	dartAssert(!rrect.HasNaN(), "RRect argument contained a NaN value.")
	b := c.builder()
	if b == nil {
		return
	}
	bounds := rrectBounds(rrect)
	c.clipBounds(bounds, aa)
	b.ClipRoundedRect(impRect(bounds), rrectRadii(rrect), impeller.ClipIntersect)
	c.ops++
}

func (c *nativeCanvas) ClipRSuperellipse(rse RSuperellipse, opts ...CanvasClipOptions) {
	aa := len(opts) == 0 || boolOpt(opts[0].DoAntiAlias)
	dartAssert(!rse.HasNaN(), "RSuperellipse argument contained a NaN value.")
	b := c.builder()
	if b == nil {
		return
	}
	bounds := dlRect{float32(rse.Left), float32(rse.Top), float32(rse.Right), float32(rse.Bottom)}
	c.clipBounds(bounds, aa)
	p := NewPath()
	p.AddRSuperellipse(rse)
	ip := c.impellerPath(p)
	b.ClipPath(ip, impeller.ClipIntersect)
	ip.Release()
	c.ops++
}

func (c *nativeCanvas) ClipPath(path *Path, opts ...CanvasClipOptions) {
	aa := len(opts) == 0 || boolOpt(opts[0].DoAntiAlias)
	b := c.builder()
	if b == nil {
		return
	}
	c.clipBounds(rectToDl(path.GetBounds()), aa)
	ip := c.impellerPath(path)
	b.ClipPath(ip, impeller.ClipIntersect)
	ip.Release()
	c.ops++
}

func (c *nativeCanvas) GetLocalClipBounds() Rect {
	if c.builder() == nil {
		return Rect{}
	}
	m := c.matrix()
	return c.top().localCullCoverage(&m).toRect()
}

func (c *nativeCanvas) GetDestinationClipBounds() Rect {
	if c.builder() == nil {
		return Rect{}
	}
	return c.top().cull.toRect()
}

// impellerPath converts a Path for the SDK, conics to quadratics within a
// tolerance tight enough at the current scale.
func (c *nativeCanvas) impellerPath(p *Path) *impeller.Path {
	pb := impeller.NewPathBuilder()
	defer pb.Release()
	m := c.matrix()
	scale := float32(math.Sqrt(math.Max(float64(float32(m[0]*m[0])+float32(m[1]*m[1])), float64(float32(m[4]*m[4])+float32(m[5]*m[5])))))
	tol := float32(0.1)
	if scale > 1 {
		tol /= scale
	}
	p.emit(pb, tol)
	fill := impeller.FillTypeNonZero
	if p.FillType() == PathFillTypeEvenOdd {
		fill = impeller.FillTypeOdd
	}
	return pb.TakePath(fill)
}

// draw runs f with the paint converted for the SDK.
func (c *nativeCanvas) draw(paint *Paint, flags paintFlags, f func(b *impeller.DisplayListBuilder, p *impeller.Paint)) {
	b := c.builder()
	if b == nil {
		return
	}
	ip := paint.impellerPaint(flags)
	f(b, ip)
	ip.Release()
	c.ops++
}

func (c *nativeCanvas) DrawColor(color Color, blendMode BlendMode) {
	// canvas.cc: DrawColor(DlColor(color.value), mode) — 8-bit color.
	p := NewPaint()
	p.SetColor(NewColor(color.Value()))
	p.SetBlendMode(blendMode)
	c.draw(p, paintFlags{}, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) { b.DrawPaint(ip) })
}

func impPt(o Offset) impeller.Point { return impeller.Point{X: safeNarrow(o.Dx), Y: safeNarrow(o.Dy)} }

func (c *nativeCanvas) DrawLine(p1, p2 Offset, paint *Paint) {
	offsetIsValid(p1)
	offsetIsValid(p2)
	c.draw(paint, flagsDrawLine, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		b.DrawLine(impPt(p1), impPt(p2), ip)
	})
}

func (c *nativeCanvas) DrawPaint(paint *Paint) {
	c.draw(paint, flagsDrawPaint, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) { b.DrawPaint(ip) })
}

func (c *nativeCanvas) DrawRect(rect Rect, paint *Paint) {
	rectIsValid(rect)
	rect = sorted(rect)
	if paint.Style() != PaintingStyleFill || !rect.IsEmpty() {
		c.draw(paint, flagsDrawGeometry, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
			b.DrawRect(impRect(rectToDl(rect)), ip)
		})
	}
}

func (c *nativeCanvas) DrawRRect(rrect RRect, paint *Paint) {
	dartAssert(!rrect.HasNaN(), "RRect argument contained a NaN value.")
	c.draw(paint, flagsDrawGeometry, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		b.DrawRoundedRectRadii(impRect(rrectBounds(rrect)), rrectRadii(rrect), ip)
	})
}

func (c *nativeCanvas) DrawDRRect(outer, inner RRect, paint *Paint) {
	dartAssert(!outer.HasNaN(), "RRect argument contained a NaN value.")
	dartAssert(!inner.HasNaN(), "RRect argument contained a NaN value.")
	c.draw(paint, flagsDrawGeometry, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		b.DrawRoundedRectDifference(impRect(rrectBounds(outer)), rrectRadii(outer),
			impRect(rrectBounds(inner)), rrectRadii(inner), ip)
	})
}

func (c *nativeCanvas) DrawRSuperellipse(rse RSuperellipse, paint *Paint) {
	dartAssert(!rse.HasNaN(), "RSuperellipse argument contained a NaN value.")
	p := NewPath()
	p.AddRSuperellipse(rse)
	c.DrawPath(p, paint)
}

func (c *nativeCanvas) DrawOval(rect Rect, paint *Paint) {
	rectIsValid(rect)
	rect = sorted(rect)
	if paint.Style() != PaintingStyleFill || !rect.IsEmpty() {
		c.draw(paint, flagsDrawGeometry, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
			b.DrawOval(impRect(rectToDl(rect)), ip)
		})
	}
}

func (c *nativeCanvas) DrawCircle(center Offset, radius float64, paint *Paint) {
	offsetIsValid(center)
	x, y, r := safeNarrow(center.Dx), safeNarrow(center.Dy), safeNarrow(radius)
	c.draw(paint, flagsDrawGeometry, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		b.DrawOval(impRect(dlRect{x - r, y - r, x + r, y + r}), ip)
	})
}

func (c *nativeCanvas) DrawArc(rect Rect, startAngle, sweepAngle float64, useCenter bool, paint *Paint) {
	rectIsValid(rect)
	// The SDK has no arc; the arc becomes a path (DlPath's arc geometry).
	p := NewPath()
	if useCenter {
		p.MoveTo(rect.Center().Dx, rect.Center().Dy)
		p.ArcTo(rect, startAngle, sweepAngle, false)
		p.Close()
	} else {
		p.AddArc(rect, startAngle, sweepAngle)
	}
	c.DrawPath(p, paint)
}

func (c *nativeCanvas) DrawPath(path *Path, paint *Paint) {
	c.draw(paint, flagsDrawGeometry, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		p := c.impellerPath(path)
		b.DrawPath(p, ip)
		p.Release()
	})
}

func (c *nativeCanvas) imageTexture(image *Image) *impeller.Texture {
	dartAssert(!imageDebugDisposed(image), "")
	return image.impellerTexture()
}

func (c *nativeCanvas) DrawImage(image *Image, offset Offset, paint *Paint) {
	tex := c.imageTexture(image)
	offsetIsValid(offset)
	if tex == nil {
		return
	}
	c.draw(paint, flagsDrawImage, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		b.DrawTexture(tex, impPt(offset), samplingFromQuality(paint.FilterQuality()), ip)
	})
}

func (c *nativeCanvas) DrawImageRect(image *Image, src, dst Rect, paint *Paint) {
	tex := c.imageTexture(image)
	rectIsValid(src)
	rectIsValid(dst)
	if tex == nil {
		return
	}
	c.draw(paint, flagsDrawImage, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		b.DrawTextureRect(tex, impRect(rectToDl(src)), impRect(rectToDl(dst)),
			samplingFromQuality(paint.FilterQuality()), ip)
	})
}

// DrawImageNine draws the nine patches as Impeller's NinePatchConverter
// does (the SDK has no nine-patch call).
func (c *nativeCanvas) DrawImageNine(image *Image, center, dst Rect, paint *Paint) {
	tex := c.imageTexture(image)
	rectIsValid(center)
	rectIsValid(dst)
	if tex == nil {
		return
	}
	w, h := float32(image.Width), float32(image.Height)
	ctr, d := rectToDl(center), rectToDl(dst)
	xs := ninePatchStops(0, ctr.l, ctr.r, w, d.l, d.r)
	ys := ninePatchStops(0, ctr.t, ctr.b, h, d.t, d.b)
	sampling := samplingFromQuality(paint.FilterQuality())
	c.draw(paint, flagsDrawImage, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		for yi := 0; yi < 3; yi++ {
			for xi := 0; xi < 3; xi++ {
				src := dlRect{xs[xi].src, ys[yi].src, xs[xi+1].src, ys[yi+1].src}
				dr := dlRect{xs[xi].dst, ys[yi].dst, xs[xi+1].dst, ys[yi+1].dst}
				if src.isEmpty() || dr.isEmpty() {
					continue
				}
				b.DrawTextureRect(tex, impRect(src), impRect(dr), sampling, ip)
			}
		}
	})
}

type ninePatchStop struct{ src, dst float32 }

// ninePatchStops is NinePatchConverter::InitSlices: the source and
// destination stops of one axis; when the destination is smaller than the
// fixed edges, the edges shrink proportionally.
func ninePatchStops(start, centerStart, centerEnd, size, dstStart, dstEnd float32) [4]ninePatchStop {
	centerStart = stdMax32(start, stdMin32(centerStart, size))
	centerEnd = stdMax32(centerStart, stdMin32(centerEnd, size))
	first, last := centerStart-start, size-centerEnd
	dstSize := dstEnd - dstStart
	fixed := first + last
	scale := float32(1)
	if fixed > dstSize && fixed > 0 {
		scale = dstSize / fixed
	}
	return [4]ninePatchStop{
		{start, dstStart},
		{centerStart, dstStart + float32(first*scale)},
		{centerEnd, dstEnd - float32(last*scale)},
		{size, dstEnd},
	}
}

func (c *nativeCanvas) DrawPicture(picture *Picture) {
	dartAssert(!picture.DebugDisposed(), "")
	b := c.builder()
	if b == nil || picture.dl == nil {
		return
	}
	b.DrawDisplayList(picture.dl, 1)
	c.ops++
}

func (c *nativeCanvas) DrawPoints(pointMode PointMode, points []Offset, paint *Paint) {
	raw := make([]float32, 0, 2*len(points))
	for _, p := range points {
		offsetIsValid(p)
		raw = append(raw, float32(p.Dx), float32(p.Dy))
	}
	c.drawPoints(pointMode, raw, paint)
}

func (c *nativeCanvas) DrawRawPoints(pointMode PointMode, points []float32, paint *Paint) {
	if len(points)%2 != 0 {
		panic(argumentError(`"points" must have an even number of values.`))
	}
	c.drawPoints(pointMode, points, paint)
}

// drawPoints is DlCanvas::DrawPoints as Impeller draws it: points are
// squares (round caps: circles) of the stroke width (a hairline is one
// pixel wide), lines are pairs of points, a polygon is an open polyline.
func (c *nativeCanvas) drawPoints(mode PointMode, pts []float32, paint *Paint) {
	n := len(pts) / 2
	c.draw(paint, flagsDrawLine, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		switch mode {
		case PointModePoints:
			r := float32(paint.StrokeWidth()) / 2
			if r == 0 {
				r = 0.5
			}
			fp := NewPaintFrom(paint)
			fp.SetStyle(PaintingStyleFill)
			fip := fp.impellerPaint(flagsDrawGeometry)
			defer fip.Release()
			for i := 0; i < n; i++ {
				x, y := pts[2*i], pts[2*i+1]
				rect := impRect(dlRect{x - r, y - r, x + r, y + r})
				if paint.StrokeCap() == StrokeCapRound {
					b.DrawOval(rect, fip)
				} else {
					b.DrawRect(rect, fip)
				}
			}
		case PointModeLines:
			for i := 0; i+1 < n; i += 2 {
				b.DrawLine(impeller.Point{X: pts[2*i], Y: pts[2*i+1]},
					impeller.Point{X: pts[2*i+2], Y: pts[2*i+3]}, ip)
			}
		case PointModePolygon:
			// Impeller's DlDispatcher::drawPoints: one line per segment.
			for i := 1; i < n; i++ {
				b.DrawLine(impeller.Point{X: pts[2*i-2], Y: pts[2*i-1]}, impeller.Point{X: pts[2*i], Y: pts[2*i+1]}, ip)
			}
		}
	})
}

// DrawVertices draws the triangles of vertices filled with the paint.
// ENGINE GAP: the SDK has no vertices API, so per-vertex colors and
// texture coordinates are ignored (each triangle is a path).
func (c *nativeCanvas) DrawVertices(vertices *Vertices, blendMode BlendMode, paint *Paint) {
	dartAssert(!vertices.DebugDisposed(), "")
	c.draw(paint, flagsDrawGeometry, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		pb := impeller.NewPathBuilder()
		defer pb.Release()
		for _, t := range vertices.triangles() {
			pb.MoveTo(t[0], t[1])
			pb.LineTo(t[2], t[3])
			pb.LineTo(t[4], t[5])
			pb.Close()
		}
		path := pb.TakePath(impeller.FillTypeNonZero)
		b.DrawPath(path, ip)
		path.Release()
	})
}

func (c *nativeCanvas) DrawAtlas(atlas *Image, transforms []RSTransform, rects []Rect, colors []Color, blendMode *BlendMode, cullRect *Rect, paint *Paint) {
	dartAssert(!imageDebugDisposed(atlas), "")
	dartAssert(len(colors) == 0 || blendMode != nil, "")
	n := len(rects)
	if len(transforms) != n {
		panic(argumentError(`"transforms" and "rects" lengths must match.`))
	}
	if len(colors) != 0 && len(colors) != n {
		panic(argumentError(`If non-null, "colors" length must match that of "transforms" and "rects".`))
	}
	rst := make([]float32, 0, 4*n)
	rs := make([]float32, 0, 4*n)
	for i := 0; i < n; i++ {
		t, r := transforms[i], rects[i]
		rectIsValid(r)
		rst = append(rst, float32(t.Scos()), float32(t.Ssin()), float32(t.Tx()), float32(t.Ty()))
		rs = append(rs, float32(r.Left), float32(r.Top), float32(r.Right), float32(r.Bottom))
	}
	var cs []int32
	if len(colors) != 0 {
		cs = make([]int32, n)
		for i, col := range colors {
			cs[i] = int32(uint32(col.Value()))
		}
	}
	c.drawAtlas(atlas, rst, rs, cs, blendMode, paint)
}

func (c *nativeCanvas) DrawRawAtlas(atlas *Image, rstTransforms, rects []float32, colors []int32, blendMode *BlendMode, cullRect *Rect, paint *Paint) {
	dartAssert(colors == nil || blendMode != nil, "")
	n := len(rects)
	if len(rstTransforms) != n {
		panic(argumentError(`"rstTransforms" and "rects" lengths must match.`))
	}
	if n%4 != 0 {
		panic(argumentError(`"rstTransforms" and "rects" lengths must be a multiple of four.`))
	}
	if colors != nil && len(colors)*4 != n {
		panic(argumentError(`If non-null, "colors" length must be one fourth the length of "rstTransforms" and "rects".`))
	}
	c.drawAtlas(atlas, rstTransforms, rects, colors, blendMode, paint)
}

// drawAtlas draws each sprite as a texture rect under its RSTransform. A
// sprite color is blended with the sprite by a blend color filter (color
// as source, sprite as destination), which is the atlas blend.
func (c *nativeCanvas) drawAtlas(atlas *Image, rst, rects []float32, colors []int32, blendMode *BlendMode, paint *Paint) {
	tex := atlas.impellerTexture()
	if tex == nil {
		return
	}
	mode := BlendModeSrc
	if blendMode != nil {
		mode = *blendMode
	}
	sampling := samplingFromQuality(paint.FilterQuality())
	c.draw(paint, flagsDrawImage, func(b *impeller.DisplayListBuilder, ip *impeller.Paint) {
		for i := 0; i+3 < len(rects); i += 4 {
			scos, ssin, tx, ty := rst[i], rst[i+1], rst[i+2], rst[i+3]
			src := dlRect{rects[i], rects[i+1], rects[i+2], rects[i+3]}
			b.Save()
			b.Transform(impeller.Matrix{scos, ssin, 0, 0, -ssin, scos, 0, 0, 0, 0, 1, 0, tx, ty, 0, 1})
			sp := ip
			if colors != nil {
				p := NewPaintFrom(paint)
				p.SetColorFilter(NewColorFilterMode(NewColor(int(uint32(colors[i/4]))), mode))
				sp = p.impellerPaint(flagsDrawImage)
			}
			b.DrawTextureRect(tex, impRect(src), impRect(dlRect{0, 0, src.r - src.l, src.b - src.t}), sampling, sp)
			if sp != ip {
				sp.Release()
			}
			b.Restore()
		}
	})
}

func (c *nativeCanvas) DrawShadow(path *Path, color Color, elevation float64, transparentOccluder bool) {
	b := c.builder()
	if b == nil {
		return
	}
	v := uint32(color.Value())
	ic := impeller.Color{A: float32(v>>24) / 255, R: float32(v>>16&0xff) / 255, G: float32(v>>8&0xff) / 255,
		B: float32(v&0xff) / 255}
	p := c.impellerPath(path)
	b.DrawShadow(p, ic, safeNarrow(elevation), transparentOccluder, float32(EngineDevicePixelRatio))
	p.Release()
	c.ops++
}

// EngineDrawParagraph draws a paragraph laid out by the Impeller SDK's
// typography. It stands in for Canvas.drawParagraph until text.dart's
// Paragraph is ported.
func EngineDrawParagraph(canvas Canvas, p *impeller.Paragraph, offset Offset) {
	c, ok := canvas.(*nativeCanvas)
	if !ok {
		panic(argumentError("EngineDrawParagraph needs the engine Canvas"))
	}
	if b := c.builder(); b != nil {
		b.DrawParagraph(p, float32(offset.Dx), float32(offset.Dy))
		c.ops++
	}
}
