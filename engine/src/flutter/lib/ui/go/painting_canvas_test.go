package ui

// Ports of the engine's Dart tests for Paint, filters, gradients, Canvas,
// Picture and Vertices (engine/src/flutter/testing/dart/canvas_test.dart,
// painting_test.dart, image_filter_test.dart, color_filter_test.dart,
// gradient_test.dart, mask_filter_test.dart, picture_test.dart).
//
// Skipped, with reasons:
//   - golden-image comparisons (canvas_test's toImage/gradient/drawAtlas/
//     paragraph/text decoration goldens, painting_test's BackdropFilter and
//     mip-map goldens, image_filter_test's FilterQuality goldens): the
//     engine's goldens are Skia-software renders not in the repository;
//     pixels are compared against Flutter's Impeller renderer in
//     src/difftest/painting_pixels_test.go instead.
//   - "canvas APIs should not crash" passes nulls to every method; Go's
//     types can't express most of them (the nullable ones are covered below).
//   - Paragraph tests (drawParagraph, tab rendering, text decorations):
//     text.dart's Paragraph is not ported.
//   - image_filter_test's blur/matrix/from-color-filter pixel tests: Flutter
//     skips them under Impeller (flutter/flutter#135712), and ColorFilter as
//     an ImageFilter is an ENGINE GAP here.
//   - color_filter_test's linearToSrgbGamma/srgbToLinearGamma: the SDK has
//     no gamma color filters (ENGINE GAP).
//   - toImageSync "too big" (needs the GPU's max texture size error path).
//   - "Canvas preserves perspective data in Matrix4": covered (getTransform
//     after a perspective transform) by the difftest.

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/src/async"
)

func expectPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("no panic, want %q", want)
		}
		var got string
		if e, ok := r.(error); ok {
			got = e.Error()
		} else if s, ok := r.(interface{ String() string }); ok {
			got = s.String()
		}
		if !strings.Contains(got, want) {
			t.Fatalf("panic %q, want %q", got, want)
		}
	}()
	f()
}

// awaitValue runs the current loop until f completes.
func awaitValue[T any](t *testing.T, f async.Future[T]) T {
	t.Helper()
	var v T
	var err error
	done := false
	async.OnComplete(f, func(x T) { v, done = x, true }, func(e error) { err, done = e, true })
	loop := async.Current()
	for !done {
		loop.FlushMicrotasks()
		if !done {
			loop.WaitPosted()
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func needGPU(t *testing.T) {
	if engineContext() == nil {
		t.Skip("no Metal device")
	}
}

// pixelsOf draws with script into a w×h picture and returns its rawRgba
// pixels as little-endian uint32 values (Dart's asUint32List).
func pixelsOf(t *testing.T, w, h int, script func(c Canvas)) []uint32 {
	t.Helper()
	needGPU(t)
	rec := NewPictureRecorder()
	script(NewCanvas(rec))
	img := awaitValue(t, rec.EndRecording().ToImage(w, h))
	b := awaitValue(t, img.ToByteData())
	if len(b) != w*h*4 {
		t.Fatalf("%d bytes, want %d", len(b), w*h*4)
	}
	out := make([]uint32, w*h)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(b[4*i:])
	}
	return out
}

func getBytesForPaint(t *testing.T, p *Paint) []uint32 {
	return pixelsOf(t, 3, 3, func(c Canvas) { c.DrawRect(NewRectFromLTRB(1, 1, 2, 2), p) })
}

func checkBytes(t *testing.T, b []uint32, center, side, corner uint32) {
	t.Helper()
	want := []uint32{corner, side, corner, side, center, side, corner, side, corner}
	for i := range want {
		if b[i] != want[i] {
			t.Fatalf("pixel %d: %#x, want %#x (all: %#x)", i, b[i], want[i], b)
		}
	}
}

// ---- canvas_test.dart ----

func closeTo(a, b []float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-5 {
			return false
		}
	}
	return true
}

func TestCanvasTranslateAffectsGetTransform(t *testing.T) {
	c := NewCanvas(NewPictureRecorder())
	c.Translate(12, 14.5)
	want := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 12, 14.5, 0, 1}
	cur := c.GetTransform()
	if !closeTo(cur, want) {
		t.Fatal(cur)
	}
	c.Translate(10, 10)
	if closeTo(c.GetTransform(), want) || !closeTo(cur, want) {
		t.Fatal("getTransform must return a snapshot")
	}
}

func TestCanvasScaleAffectsGetTransform(t *testing.T) {
	c := NewCanvas(NewPictureRecorder())
	c.Scale(12, 14.5)
	want := []float64{12, 0, 0, 0, 0, 14.5, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	if !closeTo(c.GetTransform(), want) {
		t.Fatal(c.GetTransform())
	}
	c.Scale(10, 10)
	if closeTo(c.GetTransform(), want) {
		t.Fatal("scale had no effect")
	}
	c2 := NewCanvas(NewPictureRecorder())
	c2.Scale(3) // sy defaults to sx
	if !closeTo(c2.GetTransform(), []float64{3, 0, 0, 0, 0, 3, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}) {
		t.Fatal(c2.GetTransform())
	}
}

func TestCanvasRotateAffectsGetTransform(t *testing.T) {
	c := NewCanvas(NewPictureRecorder())
	c.Rotate(math.Pi)
	want := []float64{-1, 0, 0, 0, 0, -1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	if !closeTo(c.GetTransform(), want) {
		t.Fatal(c.GetTransform())
	}
	c.Rotate(math.Pi / 2)
	if closeTo(c.GetTransform(), want) {
		t.Fatal("rotate had no effect")
	}
}

func TestCanvasSkewAffectsGetTransform(t *testing.T) {
	c := NewCanvas(NewPictureRecorder())
	c.Skew(12, 14.5)
	want := []float64{1, 14.5, 0, 0, 12, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	if !closeTo(c.GetTransform(), want) {
		t.Fatal(c.GetTransform())
	}
	c.Skew(10, 10)
	if closeTo(c.GetTransform(), want) {
		t.Fatal("skew had no effect")
	}
}

func TestCanvasTransformAffectsGetTransform(t *testing.T) {
	c := NewCanvas(NewPictureRecorder())
	m := []float64{12, 0, 0, 0, 0, 14.5, 0, 0, 0, 0, 1, 0, 12, 14.5, 0, 1}
	c.Transform(m)
	if !closeTo(c.GetTransform(), m) {
		t.Fatal(c.GetTransform())
	}
	c.Translate(10, 10)
	if closeTo(c.GetTransform(), m) {
		t.Fatal("translate had no effect")
	}
	expectPanic(t, `"matrix4" must have 16 entries.`, func() { c.Transform(make([]float64, 9)) })
}

func closeToRect(a, b Rect) bool {
	return closeTo([]float64{a.Left, a.Top, a.Right, a.Bottom}, []float64{b.Left, b.Top, b.Right, b.Bottom})
}

func TestCanvasClipRectAffectsGetClipBounds(t *testing.T) {
	testRect := func(clip Rect, aa bool) {
		c := NewCanvas(NewPictureRecorder())
		c.ClipRect(clip, CanvasClipRectOptions{DoAntiAlias: &aa})
		s := NewRectFromLTRB(math.Min(clip.Left, clip.Right), math.Min(clip.Top, clip.Bottom),
			math.Max(clip.Left, clip.Right), math.Max(clip.Top, clip.Bottom))
		exp := s
		if aa {
			exp = NewRectFromLTRB(math.Floor(s.Left), math.Floor(s.Top), math.Ceil(s.Right), math.Ceil(s.Bottom))
		}
		local, dest := c.GetLocalClipBounds(), c.GetDestinationClipBounds()
		if !closeToRect(local, exp) || !closeToRect(dest, exp) {
			t.Fatalf("%v aa=%v: %v %v, want %v", clip, aa, local, dest, exp)
		}
		c.Save()
		c.ClipRect(NewRectFromLTRB(0, 0, 15, 15))
		if closeToRect(c.GetLocalClipBounds(), exp) || closeToRect(c.GetDestinationClipBounds(), exp) {
			t.Fatal("nested clip had no effect")
		}
		c.Restore()
		if c.GetLocalClipBounds() != local || c.GetDestinationClipBounds() != dest {
			t.Fatal("restore didn't restore the clip")
		}
		c.Save()
		c.Scale(2, 2)
		scaled := NewRectFromLTRB(exp.Left/2, exp.Top/2, exp.Right/2, exp.Bottom/2)
		if !closeToRect(c.GetLocalClipBounds(), scaled) || !closeToRect(c.GetDestinationClipBounds(), exp) {
			t.Fatal(c.GetLocalClipBounds(), c.GetDestinationClipBounds())
		}
		c.Restore()
		if c.GetLocalClipBounds() != local || c.GetDestinationClipBounds() != dest {
			t.Fatal("restore didn't restore the clip")
		}
	}
	for _, r := range []Rect{NewRectFromLTRB(10.2, 11.3, 20.4, 25.7), NewRectFromLTRB(20.4, 11.3, 10.2, 25.7),
		NewRectFromLTRB(10.2, 25.7, 20.4, 11.3), NewRectFromLTRB(20.4, 25.7, 10.2, 11.3)} {
		testRect(r, false)
		testRect(r, true)
	}
}

func TestCanvasClipWithMatrixAffectsGetClipBounds(t *testing.T) {
	b1, b2 := NewRectFromLTRB(0, 0, 10, 10), NewRectFromLTRB(10, 10, 20, 20)
	no := false
	kinds := map[string]func(c Canvas, r Rect){
		"rect": func(c Canvas, r Rect) { c.ClipRect(r, CanvasClipRectOptions{DoAntiAlias: &no}) },
		"rrect": func(c Canvas, r Rect) {
			c.ClipRRect(NewRRectFromRectAndRadius(r, NewRadiusCircular(3)), CanvasClipOptions{DoAntiAlias: &no})
		},
		"path": func(c Canvas, r Rect) {
			p := NewPath()
			p.AddRect(r)
			p.AddOval(r)
			c.ClipPath(p, CanvasClipOptions{DoAntiAlias: &no})
		},
	}
	for name, clip := range kinds {
		c := NewCanvas(NewPictureRecorder())
		c.Save()
		clip(c, b1)
		c.Translate(0, 10)
		clip(c, b1)
		if !c.GetDestinationClipBounds().IsEmpty() {
			t.Errorf("%s: %v, want empty", name, c.GetDestinationClipBounds())
		}
		c.Restore()
		c.Save()
		clip(c, b1)
		c.Translate(-10, -10)
		clip(c, b2)
		if c.GetDestinationClipBounds() != b1 {
			t.Errorf("%s: %v, want %v", name, c.GetDestinationClipBounds(), b1)
		}
		c.Restore()
	}
}

func TestCanvasClipRRectAndPathAffectGetClipBounds(t *testing.T) {
	bounds := NewRectFromLTRB(10.2, 11.3, 20.4, 25.7)
	expanded := NewRectFromLTRB(10, 11, 21, 26)
	for _, aa := range []bool{true, false} {
		clips := map[string]func(c Canvas){
			"rrect": func(c Canvas) {
				c.ClipRRect(NewRRectFromRectAndRadius(bounds, NewRadiusCircular(3)), CanvasClipOptions{DoAntiAlias: &aa})
			},
			"path": func(c Canvas) {
				p := NewPath()
				p.AddRect(bounds)
				p.AddOval(bounds)
				c.ClipPath(p, CanvasClipOptions{DoAntiAlias: &aa})
			},
		}
		want, scaled := expanded, NewRectFromLTRB(5, 5.5, 10.5, 13)
		if !aa {
			want, scaled = bounds, NewRectFromLTRB(5.1, 5.65, 10.2, 12.85)
		}
		for name, clip := range clips {
			c := NewCanvas(NewPictureRecorder())
			clip(c)
			local, dest := c.GetLocalClipBounds(), c.GetDestinationClipBounds()
			if !closeToRect(local, want) || !closeToRect(dest, want) {
				t.Errorf("%s aa=%v: %v %v, want %v", name, aa, local, dest, want)
			}
			c.Save()
			c.ClipRect(NewRectFromLTRB(0, 0, 15, 15), CanvasClipRectOptions{DoAntiAlias: &aa})
			if closeToRect(c.GetLocalClipBounds(), want) {
				t.Errorf("%s: nested clip had no effect", name)
			}
			c.Restore()
			c.Save()
			c.Scale(2, 2)
			if !closeToRect(c.GetLocalClipBounds(), scaled) || !closeToRect(c.GetDestinationClipBounds(), want) {
				t.Errorf("%s aa=%v scaled: %v %v", name, aa, c.GetLocalClipBounds(), c.GetDestinationClipBounds())
			}
			c.Restore()
			if c.GetLocalClipBounds() != local || c.GetDestinationClipBounds() != dest {
				t.Errorf("%s: restore didn't restore the clip", name)
			}
		}
	}
}

func TestCanvasClipRectDifferenceDoesNotAffectGetClipBounds(t *testing.T) {
	c := NewCanvas(NewPictureRecorder())
	bounds := NewRectFromLTRB(10.2, 11.3, 20.4, 25.7)
	no := false
	c.ClipRect(bounds, CanvasClipRectOptions{DoAntiAlias: &no})
	local, dest := c.GetLocalClipBounds(), c.GetDestinationClipBounds()
	if !closeToRect(local, bounds) || !closeToRect(dest, bounds) {
		t.Fatal(local, dest)
	}
	diff := ClipOpDifference
	c.ClipRect(NewRectFromLTRB(0, 0, 15, 15), CanvasClipRectOptions{ClipOp: &diff, DoAntiAlias: &no})
	if c.GetLocalClipBounds() != local || c.GetDestinationClipBounds() != dest {
		t.Fatal(c.GetLocalClipBounds(), c.GetDestinationClipBounds())
	}
}

func TestCanvasRestoreToCount(t *testing.T) {
	fresh := func() Canvas {
		c := NewCanvas(NewPictureRecorder())
		for i := 0; i < 5; i++ {
			c.Save()
		}
		if c.GetSaveCount() != 6 {
			t.Fatal(c.GetSaveCount())
		}
		return c
	}
	c := fresh()
	c.RestoreToCount(2)
	if c.GetSaveCount() != 2 {
		t.Fatal(c.GetSaveCount())
	}
	c.Restore()
	if c.GetSaveCount() != 1 {
		t.Fatal(c.GetSaveCount())
	}
	c = fresh()
	c.RestoreToCount(0) // less than 1: the stack is reset
	if c.GetSaveCount() != 1 {
		t.Fatal(c.GetSaveCount())
	}
	c = fresh()
	c.RestoreToCount(c.GetSaveCount() + 1) // greater: nothing happens
	if c.GetSaveCount() != 6 {
		t.Fatal(c.GetSaveCount())
	}
}

func testPaint() *Paint {
	p := NewPaint()
	p.SetColor(NewColor(0xFF0000FF))
	p.SetStrokeWidth(10)
	p.SetStrokeCap(StrokeCapRound)
	p.SetStrokeJoin(StrokeJoinRound)
	p.SetStyle(PaintingStyleStroke)
	p.SetBlendMode(BlendModeSrcOver)
	m := NewMaskFilterBlur(BlurStyleNormal, 10)
	p.SetMaskFilter(&m)
	p.SetFilterQuality(FilterQualityHigh)
	p.SetColorFilter(NewColorFilterMode(NewColor(0xFF00FF00), BlendModeColor))
	p.SetImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 10, SigmaY: 10}))
	return p
}

func checkTestPaint(t *testing.T, p *Paint) {
	t.Helper()
	m := NewMaskFilterBlur(BlurStyleNormal, 10)
	if p.Color() != NewColor(0xFF0000FF) || p.StrokeWidth() != 10 || p.StrokeCap() != StrokeCapRound ||
		p.StrokeJoin() != StrokeJoinRound || p.Style() != PaintingStyleStroke || p.BlendMode() != BlendModeSrcOver ||
		*p.MaskFilter() != m || p.FilterQuality() != FilterQualityHigh ||
		!p.ColorFilter().Equal(NewColorFilterMode(NewColor(0xFF00FF00), BlendModeColor)) ||
		!ImageFilterEqual(p.ImageFilter(), NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 10, SigmaY: 10})) {
		t.Fatal(p)
	}
}

func TestPaintCopyHasEquivalentFields(t *testing.T) { checkTestPaint(t, NewPaintFrom(testPaint())) }

func TestPaintCopyDoesNotMutateOriginal(t *testing.T) {
	p := testPaint()
	c := NewPaintFrom(p)
	c.SetColor(NewColor(0xFFFF0000))
	c.SetStrokeWidth(20)
	c.SetStrokeCap(StrokeCapButt)
	c.SetStrokeJoin(StrokeJoinBevel)
	c.SetStyle(PaintingStyleFill)
	c.SetBlendMode(BlendModeSrc)
	m := NewMaskFilterBlur(BlurStyleNormal, 20)
	c.SetMaskFilter(&m)
	c.SetFilterQuality(FilterQualityLow)
	c.SetColorFilter(NewColorFilterMode(NewColor(0xFFFF0000), BlendModeColor))
	c.SetImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 20, SigmaY: 20}))
	checkTestPaint(t, p)
}

func TestPaintOriginalChangeDoesNotMutateCopy(t *testing.T) {
	p := testPaint()
	c := NewPaintFrom(p)
	p.SetColor(NewColor(0xFFFF0000))
	p.SetStrokeWidth(20)
	p.SetImageFilter(NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 20, SigmaY: 20}))
	checkTestPaint(t, c)
}

func TestDrawAtlasDataLengthsMustMatch(t *testing.T) {
	needGPU(t)
	img := newImageFromRGBA8(2, 2, make([]byte, 16))
	c := NewCanvas(NewPictureRecorder())
	rect := NewRectFromLTWH(0, 0, 100, 100)
	tr := NewRSTransform(1, 0, 0, 0)
	src := BlendModeSrc
	p := NewPaint()
	// These don't throw (null and empty colors are allowed).
	c.DrawAtlas(img, []RSTransform{tr}, []Rect{rect}, []Color{{}}, &src, &rect, p)
	c.DrawAtlas(img, []RSTransform{tr}, []Rect{rect}, []Color{}, nil, &rect, p)
	c.DrawAtlas(img, []RSTransform{tr}, []Rect{rect}, nil, nil, &rect, p)
	c.DrawRawAtlas(img, []float32{}, []float32{}, []int32{}, &src, &rect, p)
	c.DrawRawAtlas(img, make([]float32, 4), make([]float32, 4), make([]int32, 1), &src, &rect, p)
	c.DrawRawAtlas(img, make([]float32, 4), make([]float32, 4), nil, nil, &rect, p)
	expectPanic(t, `"transforms" and "rects" lengths must match.`, func() {
		c.DrawAtlas(img, []RSTransform{tr}, []Rect{rect, rect}, []Color{{}}, &src, &rect, p)
	})
	expectPanic(t, `If non-null, "colors" length must match that of "transforms" and "rects".`, func() {
		c.DrawAtlas(img, []RSTransform{tr}, []Rect{rect}, []Color{{}, {}}, &src, &rect, p)
	})
	expectPanic(t, `"rstTransforms" and "rects" lengths must match.`, func() {
		c.DrawRawAtlas(img, make([]float32, 4), make([]float32, 8), nil, nil, &rect, p)
	})
	expectPanic(t, `"rstTransforms" and "rects" lengths must be a multiple of four.`, func() {
		c.DrawRawAtlas(img, make([]float32, 3), make([]float32, 3), nil, nil, &rect, p)
	})
	expectPanic(t, `If non-null, "colors" length must be one fourth the length of "rstTransforms" and "rects".`, func() {
		c.DrawRawAtlas(img, make([]float32, 4), make([]float32, 4), make([]int32, 2), &src, &rect, p)
	})
}

func TestSimpleToImage(t *testing.T) {
	needGPU(t)
	rec := NewPictureRecorder()
	c := NewCanvas(rec, NewRectFromLTRB(0, 0, 100, 100))
	p := NewPath()
	p.AddOval(NewRectFromCircle(NewOffset(40, 40), 20))
	paint := NewPaint()
	paint.SetIsAntiAlias(false)
	c.DrawPath(p, paint)
	img := awaitValue(t, rec.EndRecording().ToImage(100, 100))
	if img.Width != 100 || img.Height != 100 {
		t.Fatal(img.Width, img.Height)
	}
	px := awaitValue(t, img.ToByteData())
	at := func(x, y int) []byte { return px[(y*100+x)*4 : (y*100+x)*4+4] }
	if a := at(40, 40); a[3] != 255 || a[0] != 0 {
		t.Fatalf("center %v, want opaque black", a)
	}
	if a := at(5, 5); a[3] != 0 {
		t.Fatalf("outside %v, want transparent", a)
	}
}

func TestToImageSyncAndToImageHaveIdenticalContents(t *testing.T) {
	needGPU(t)
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	p := NewPaint()
	p.SetColor(NewColor(0xFF2196F3))
	c.DrawCircle(NewOffset(20, 20), 15, p)
	pic := rec.EndRecording()
	a := awaitValue(t, pic.ToImageSync(40, 40).ToByteData())
	b := awaitValue(t, awaitValue(t, pic.ToImage(40, 40)).ToByteData())
	if string(a) != string(b) {
		t.Fatal("toImage and toImageSync differ")
	}
	expectPanic(t, "Invalid image dimensions.", func() { pic.ToImageSync(-1, 2) })
	expectPanic(t, "Invalid image dimensions.", func() { pic.ToImage(2, 0) })
}

func TestDrawRectOvalClipRectWithUnsortedRectangles(t *testing.T) {
	// canvas_test.dart: unsorted rects render as their sorted forms.
	draw := func(sortedRects bool) []uint32 {
		return pixelsOf(t, 20, 20, func(c Canvas) {
			r := NewRectFromLTRB(15, 15, 5, 5)
			if sortedRects {
				r = NewRectFromLTRB(5, 5, 15, 15)
			}
			p := NewPaint()
			c.ClipRect(r)
			c.DrawRect(r, p)
			p.SetColor(NewColor(0xFF00FF00))
			c.DrawOval(r, p)
		})
	}
	a, b := draw(false), draw(true)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("pixel %d: %#x vs %#x", i, a[i], b[i])
		}
	}
}

// ---- painting_test.dart ----

func TestVerticesChecks(t *testing.T) {
	zero3 := []Offset{{}, {}, {}}
	expectPanic(t, `Invalid argument(s): "indices" values must be valid indices in the positions list (i.e. numbers in the range 0..2), but indices[2] is 5, which is too big.`,
		func() { NewVertices(VertexModeTriangles, zero3, VerticesOptions{Indices: []int{0, 2, 5}}) })
	NewVertices(VertexModeTriangles, []Offset{{}}).Dispose()
	NewVertices(VertexModeTriangles, zero3, VerticesOptions{Indices: []int{0, 2, 1, 2, 0, 1, 2, 0}}).Dispose()
}

func TestVerticesRawChecks(t *testing.T) {
	expectPanic(t, `Invalid argument(s): "positions" must have an even number of entries (each coordinate is an x,y pair).`,
		func() { NewVerticesRaw(VertexModeTriangles, make([]float32, 5)) })
	expectPanic(t, `Invalid argument(s): "indices" values must be valid indices in the positions list (i.e. numbers in the range 0..2), but indices[2] is 5, which is too big.`,
		func() {
			NewVerticesRaw(VertexModeTriangles, make([]float32, 6), VerticesRawOptions{Indices: []uint16{0, 2, 5}})
		})
	NewVerticesRaw(VertexModeTriangles, make([]float32, 2)).Dispose()
	NewVerticesRaw(VertexModeTriangles, make([]float32, 6), VerticesRawOptions{Indices: []uint16{0, 2, 1, 2, 0, 1, 2, 0}}).Dispose()
}

func TestImageFilterMatrixDefaultsToFilterQualityMedium(t *testing.T) {
	f := NewImageFilterMatrix([]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1})
	if !strings.Contains(f.String(), "FilterQuality.medium") {
		t.Fatal(f)
	}
}

// ---- image_filter_test.dart ----

const (
	testRed   = 0xFFAA0000
	testGreen = 0xFF00AA00
)

func makeScale(scX, scY, trX, trY float64, q FilterQuality) ImageFilter {
	trX *= 1.0 - scX
	trY *= 1.0 - scY
	return NewImageFilterMatrix([]float64{scX, 0, 0, 0, 0, scY, 0, 0, 0, 0, 1, 0, trX, trY, 0, 1},
		ImageFilterMatrixOptions{FilterQuality: &q})
}

func makeFilterList() []ImageFilter {
	blur := func(sx, sy float64, tm *TileMode) ImageFilter {
		return NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: sx, SigmaY: sy, TileMode: tm})
	}
	t := func(m TileMode) *TileMode { return &m }
	dil := func(x, y float64) ImageFilter {
		return NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: x, RadiusY: y})
	}
	ero := func(x, y float64) ImageFilter {
		return NewImageFilterErode(ImageFilterRadiusOptions{RadiusX: x, RadiusY: y})
	}
	gray := []float64{0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0, 0, 0, 0, 1, 0}
	return []ImageFilter{
		blur(10, 10, nil), blur(10, 10, t(TileModeDecal)), blur(10, 10, t(TileModeClamp)),
		blur(10, 10, t(TileModeMirror)), blur(10, 10, t(TileModeRepeated)), blur(10, 20, nil), blur(20, 20, nil),
		dil(10, 20), dil(20, 20), dil(20, 10), ero(10, 20), ero(20, 20), ero(20, 10),
		makeScale(10, 10, 0, 0, FilterQualityLow), makeScale(10, 20, 0, 0, FilterQualityLow),
		makeScale(20, 10, 0, 0, FilterQualityLow), makeScale(10, 10, 1, 1, FilterQualityLow),
		makeScale(10, 10, 0, 0, FilterQualityMedium), makeScale(10, 10, 0, 0, FilterQualityHigh),
		makeScale(10, 10, 0, 0, FilterQualityNone),
		NewColorFilterMode(NewColor(testGreen), BlendModeColor), NewColorFilterMode(NewColor(testRed), BlendModeColor),
		NewColorFilterMode(NewColor(testRed), BlendModeScreen), NewColorFilterMatrix(gray),
		NewColorFilterLinearToSrgbGamma(), NewColorFilterSrgbToLinearGamma(),
	}
}

func composedFilters(a, b []ImageFilter) []ImageFilter {
	var out []ImageFilter
	for _, x := range a {
		for _, y := range b {
			out = append(out, NewImageFilterCompose(x, y))
		}
	}
	return out
}

func TestImageFilterEquals(t *testing.T) {
	check := func(a, b []ImageFilter) {
		t.Helper()
		for i := range a {
			for j := range a {
				eq := ImageFilterEqual(a[i], b[j])
				sameStr := a[i].String() == b[j].String()
				if (i == j) != eq || (i == j) != sameStr {
					t.Fatalf("%d,%d: equal=%v sameString=%v (%v vs %v)", i, j, eq, sameStr, a[i], b[j])
				}
			}
		}
	}
	A, B := makeFilterList(), makeFilterList()
	check(A, A)
	check(A, B)
	check(B, B)
	check(composedFilters(A, A), composedFilters(A, A))
	check(composedFilters(A, B), composedFilters(B, A))
}

func TestImageFilterBlurToString(t *testing.T) {
	tm := func(m TileMode) *TileMode { return &m }
	for want, f := range map[string]ImageFilter{
		"ImageFilter.blur(1.9, 2.1, unspecified)": NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1.9, SigmaY: 2.1}),
		"ImageFilter.blur(1.9, 2.1, decal)":       NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1.9, SigmaY: 2.1, TileMode: tm(TileModeDecal)}),
		"ImageFilter.blur(1.9, 2.1, clamp)":       NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1.9, SigmaY: 2.1, TileMode: tm(TileModeClamp)}),
		"ImageFilter.blur(1.9, 2.1, mirror)":      NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1.9, SigmaY: 2.1, TileMode: tm(TileModeMirror)}),
		"ImageFilter.blur(1.9, 2.1, repeated)":    NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: 1.9, SigmaY: 2.1, TileMode: tm(TileModeRepeated)}),
	} {
		if f.String() != want {
			t.Errorf("%q, want %q", f.String(), want)
		}
	}
}

func TestImageFilterDilate(t *testing.T) {
	p := NewPaint()
	p.SetColor(NewColor(testGreen))
	p.SetImageFilter(NewImageFilterDilate(ImageFilterRadiusOptions{RadiusX: 1, RadiusY: 1}))
	checkBytes(t, getBytesForPaint(t, p), 0xFF00AA00, 0xFF00AA00, 0xFF00AA00)
}

func TestImageFilterErode(t *testing.T) {
	p := NewPaint()
	p.SetColor(NewColor(testGreen))
	p.SetImageFilter(NewImageFilterErode(ImageFilterRadiusOptions{RadiusX: 1, RadiusY: 1}))
	checkBytes(t, getBytesForPaint(t, p), 0, 0, 0)
}

func TestImageFilterMatrixCopiesTheList(t *testing.T) {
	m := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
	f := NewImageFilterMatrix(m)
	orig := f.String()
	m[0] = 12345
	if !strings.Contains(f.String(), "[1.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 1.0]") ||
		f.String() != orig {
		t.Fatal(f)
	}
}

func TestCompositeImageFilterToString(t *testing.T) {
	tm := func(m TileMode) *TileMode { return &m }
	blur := func(s float64, m *TileMode) ImageFilter {
		return NewImageFilterBlur(ImageFilterBlurOptions{SigmaX: s, SigmaY: s, TileMode: m})
	}
	if s := NewImageFilterCompose(blur(20, tm(TileModeDecal)), blur(10, nil)).String(); !strings.Contains(s,
		"blur(10.0, 10.0, unspecified) -> blur(20.0, 20.0, decal)") {
		t.Error(s)
	}
	s := NewImageFilterCompose(
		NewImageFilterCompose(blur(30, tm(TileModeMirror)), blur(20, tm(TileModeRepeated))),
		NewImageFilterCompose(NewColorFilterMode(NewColor(0xFFABCDEF), BlendModeColor), makeScale(10, 10, 0, 0, FilterQualityLow)),
	).String()
	want := "matrix([10.0, 0.0, 0.0, 0.0, 0.0, 10.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, -0.0, -0.0, 0.0, 1.0], FilterQuality.low) -> " +
		"ColorFilter.mode(" + NewColor(0xFFABCDEF).String() + ", BlendMode.color) -> blur(20.0, 20.0, repeated) -> blur(30.0, 30.0, mirror)"
	if !strings.Contains(s, want) {
		t.Errorf("%s\nwant %s", s, want)
	}
}

// ---- color_filter_test.dart ----

func colorFilterBytes(t *testing.T, f *ColorFilter, c uint32) []uint32 {
	p := NewPaint()
	p.SetColor(NewColor(int(c)))
	p.SetColorFilter(f)
	return pixelsOf(t, 1, 1, func(cv Canvas) { cv.DrawPaint(p) })
}

func TestColorFilterMode(t *testing.T) {
	// green under ColorFilter.mode(red, color) is greenRedColorBlend. The
	// invertColors half is disabled under Impeller in Flutter (#135699).
	if b := colorFilterBytes(t, NewColorFilterMode(NewColor(testRed), BlendModeColor), testGreen); b[0] != 0xFF3131DB {
		t.Fatalf("%#x", b[0])
	}
}

// nopLayer is the NOP tests' scene: a saveLayer whose paint has the filter
// (Flutter renders it through a SceneBuilder; a picture's toImage is the
// same rasterization here).
func nopLayer(t *testing.T, f *ColorFilter) {
	p := NewPaint()
	p.SetColor(NewColor(testGreen))
	p.SetColorFilter(f)
	pixelsOf(t, 100, 100, func(c Canvas) {
		b := NewRectFromLTRB(-100, -100, 200, 200)
		c.SaveLayer(&b, p)
		c.DrawRect(NewRectFromLTRB(0, 0, 100, 100), NewPaint())
		c.Restore()
	})
}

func TestColorFilterNOPModeDoesNotCrash(t *testing.T) {
	nopLayer(t, NewColorFilterMode(NewColor(0), BlendModeSrcOver))
}

func TestColorFilterMatrix(t *testing.T) {
	gray := []float64{0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0, 0.2126, 0.7152, 0.0722, 0, 0, 0, 0, 0, 1, 0}
	if b := colorFilterBytes(t, NewColorFilterMatrix(gray), testGreen); b[0] != 0xFF7A7A7A { // greenGreyscaled
		t.Fatalf("%#x", b[0])
	}
}

func TestColorFilterNOPMatrixDoesNotCrash(t *testing.T) {
	nopLayer(t, NewColorFilterMatrix([]float64{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0}))
}

// ---- gradient_test.dart ----

func TestGradientRadial(t *testing.T) {
	colors := []Color{NewColor(0xFFFFFFFF), NewColor(0xFF000000)}
	NewGradientRadial(Offset{}, 5, colors)                                                          // no focal point
	NewGradientRadial(Offset{}, 5, colors, GradientRadialOptions{Focal: &Offset{}})                 // center == focal == zero, focalRadius 0: ok
	NewGradientRadial(Offset{}, 5, colors, GradientRadialOptions{Focal: &Offset{Dx: 2.5, Dy: 2.5}}) // center != focal: ok
	expectPanic(t, "", func() {                                                                     // center == focal == zero with a focal radius asserts
		NewGradientRadial(Offset{}, 5, colors, GradientRadialOptions{Focal: &Offset{}, FocalRadius: 1})
	})
}

// ---- mask_filter_test.dart ----

func TestMaskFilterNOPBlurDoesNotCrash(t *testing.T) {
	p := NewPaint()
	m := NewMaskFilterBlur(BlurStyleNormal, 0)
	p.SetMaskFilter(&m)
	pixelsOf(t, 1, 1, func(c Canvas) { c.DrawRect(NewRectFromLTRB(0, 0, 1, 1), p) })
}

// ---- picture_test.dart ----

func createPicture() *Picture {
	rec := NewPictureRecorder()
	c := NewCanvas(rec)
	c.ClipRect(NewRectFromLTRB(0, 0, 10, 10))
	return rec.EndRecording()
}

func TestPictureOnCreateOnce(t *testing.T) {
	n := 0
	var created *Picture
	PictureOnCreate = func(p *Picture) { n++; created = p }
	defer func() { PictureOnCreate = nil }()
	p1 := createPicture()
	if n != 1 || created != p1 {
		t.Fatal(n)
	}
	p2 := createPicture()
	if n != 2 || created != p2 {
		t.Fatal(n)
	}
}

func TestApproximateBytesUsedAvailableForOnCreate(t *testing.T) {
	size := -1
	PictureOnCreate = func(p *Picture) { size = p.ApproximateBytesUsed() }
	defer func() { PictureOnCreate = nil }()
	createPicture()
	if size < 0 {
		t.Fatal(size)
	}
}

func TestPictureDisposeInvokesOnDisposeOnce(t *testing.T) {
	n := 0
	var disposed *Picture
	PictureOnDispose = func(p *Picture) { n++; disposed = p }
	defer func() { PictureOnDispose = nil }()
	p1 := createPicture()
	p1.Dispose()
	if n != 1 || disposed != p1 {
		t.Fatal(n)
	}
	p2 := createPicture()
	p2.Dispose()
	if n != 2 || disposed != p2 {
		t.Fatal(n)
	}
}

// ---- flutter-go's own checks ----

func TestPaintStoredValuesAndToString(t *testing.T) {
	p := NewPaint()
	if p.String() != "Paint()" || p.StrokeMiterLimit() != 0 || !p.IsAntiAlias() || p.Color() != NewColor(0xFF000000) {
		t.Fatal(p, p.StrokeMiterLimit())
	}
	p.SetStrokeWidth(0.1)
	if p.StrokeWidth() != float64(float32(0.1)) {
		t.Fatal(p.StrokeWidth())
	}
	p.SetStrokeMiterLimit(10)
	p.SetStyle(PaintingStyleStroke)
	if p.String() != "Paint(PaintingStyle.stroke 0.1 StrokeJoin.miter up to 6.0)" {
		t.Fatal(p)
	}
}

func TestClipBoundsArithmetic(t *testing.T) {
	// The inverse of a rotation, with the engine's fused arithmetic.
	m := dlMatrix{0.9553365, 0.29552022, 0, 0, -0.29552022, 0.9553365, 0, 0, 0, 0, 1, 0, 100, 50, 0, 1}
	inv, ok := invert(&m)
	if !ok {
		t.Fatal("not invertible")
	}
	r := dlRect{68, 34, 132, 98}.transformBounds(&inv)
	if r.l > -35 || r.l < -36 || r.r < 44 || r.r > 45 {
		t.Fatal(r)
	}
	if d := determinant(&m); math.Abs(float64(d)-1) > 1e-6 {
		t.Fatal(d)
	}
	if got := (dlRect{0, 0, 10, 10}).cutoutOrEmpty(dlRect{-1, -1, 11, 4}); got != (dlRect{0, 4, 10, 10}) {
		t.Fatal(got)
	}
}
