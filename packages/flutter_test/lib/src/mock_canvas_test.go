package flutter_test

// Ports of packages/flutter_test/test/mock_canvas_test.dart and
// recording_canvas_test.dart (Flutter 3.50.1). Adapted: the Dart tests
// paint through a CustomPaint widget (rendering's RenderCustomPaint wraps
// the painter in save/restore) and match `tester.renderObject(...)`;
// rendering and widgets' CustomPaint are not ported, so each painter is a
// func(ui.Canvas) doing what RenderCustomPaint and the CustomPainter do.
// recording_canvas_test's Material.of(...) targets are the painter itself.
// Plus own tests of every pattern method's match and failure message, the
// other matchers and TestRecordingCanvas.

import (
	"math"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
)

var (
	kTransparent = ui.NewColor(0x00000000)
	kBlue        = ui.NewColor(0xFF0000FF)
)

func pf(v float64) *float64 { return &v }
func pb(v bool) *bool       { return &v }

// customPaint is RenderCustomPaint's painting of a CustomPainter at the
// origin: save, paint, restore.
func customPaint(paint func(c ui.Canvas)) func(ui.Canvas) {
	return func(c ui.Canvas) {
		c.Save()
		paint(c)
		c.Restore()
	}
}

func myPainter(color ui.Color) func(ui.Canvas) {
	return customPaint(func(c ui.Canvas) { c.DrawColor(color, ui.BlendModeColor) })
}

type methodAndArguments struct {
	method    string
	arguments []any
}

func checkMethods(t *testing.T, got []methodAndArguments, want []methodAndArguments) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i].method != want[i].method || len(got[i].arguments) != len(want[i].arguments) {
			t.Fatalf("[%d] got %v, want %v", i, got[i], want[i])
		}
		for j := range want[i].arguments {
			if !dartEquals(got[i].arguments[j], want[i].arguments[j]) {
				t.Fatalf("[%d][%d] got %v, want %v", i, j, got[i].arguments[j], want[i].arguments[j])
			}
		}
	}
}

func matches(object any, m Matcher) (bool, string) {
	state := map[any]any{}
	ok := m.Matches(object, state)
	return ok, m.DescribeMismatch(object, state)
}

func TestSomethingMatchesWhenPredicateReturnsTrue(t *testing.T) {
	var got []methodAndArguments
	Expect(t, myPainter(kTransparent), Paints().Something(func(method string, arguments []any) bool {
		got = append(got, methodAndArguments{method, arguments})
		return method == "drawColor"
	}))
	checkMethods(t, got, []methodAndArguments{{"save", nil}, {"drawColor", []any{kTransparent, ui.BlendModeColor}}})
}

func TestSomethingFailsWhenPredicateAlwaysReturnsFalse(t *testing.T) {
	var got []methodAndArguments
	Expect(t, myPainter(kTransparent), IsNot(Paints().Something(func(method string, arguments []any) bool {
		got = append(got, methodAndArguments{method, arguments})
		return false
	})))
	checkMethods(t, got, []methodAndArguments{{"save", nil}, {"drawColor", []any{kTransparent, ui.BlendModeColor}}, {"restore", nil}})
	// Own: the failure message once every call was tried.
	expectFailureContaining(t, myPainter(kTransparent), Paints().Something(func(string, []any) bool { return false }),
		`It painted methods that the predicate passed to a "something" step, in the paint pattern, none of which were considered correct.`)
}

func TestSomethingFailsWhenPredicateThrows(t *testing.T) {
	var got []methodAndArguments
	Expect(t, myPainter(kTransparent), IsNot(Paints().Something(func(method string, arguments []any) bool {
		got = append(got, methodAndArguments{method, arguments})
		if method == "save" {
			return false
		}
		if method == "drawColor" {
			panic(TestFailure{"fail"})
		}
		return true
	})))
	checkMethods(t, got, []methodAndArguments{{"save", nil}, {"drawColor", []any{kTransparent, ui.BlendModeColor}}})
}

func TestEverythingMatchesWhenPredicateAlwaysReturnsTrue(t *testing.T) {
	var got []methodAndArguments
	Expect(t, myPainter(kTransparent), Paints().Everything(func(method string, arguments []any) bool {
		got = append(got, methodAndArguments{method, arguments})
		return true
	}))
	checkMethods(t, got, []methodAndArguments{{"save", nil}, {"drawColor", []any{kTransparent, ui.BlendModeColor}}, {"restore", nil}})
}

func TestEverythingFailsWhenPredicateReturnsFalse(t *testing.T) {
	var got []methodAndArguments
	Expect(t, myPainter(kTransparent), IsNot(Paints().Everything(func(method string, arguments []any) bool {
		got = append(got, methodAndArguments{method, arguments})
		return method == "restore" || method == "save"
	})))
	checkMethods(t, got, []methodAndArguments{{"save", nil}, {"drawColor", []any{kTransparent, ui.BlendModeColor}}})
}

func TestEverythingFailsIfPredicateEverThrows(t *testing.T) {
	var got []methodAndArguments
	Expect(t, myPainter(kTransparent), IsNot(Paints().Everything(func(method string, arguments []any) bool {
		got = append(got, methodAndArguments{method, arguments})
		if method == "drawColor" {
			panic(TestFailure{"failed "})
		}
		return true
	})))
	checkMethods(t, got, []methodAndArguments{{"save", nil}, {"drawColor", []any{kTransparent, ui.BlendModeColor}}})
}

func expectFailureContaining(t *testing.T, object any, m Matcher, want string) {
	t.Helper()
	ok, msg := matches(object, m)
	if ok {
		t.Fatalf("matched; want a failure containing %q", want)
	}
	if !strings.Contains(msg, want) {
		t.Fatalf("failure %q does not contain %q", msg, want)
	}
}

func TestArc(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 50, 50)
	startAngle, sweepAngle, useCenter := math.Pi/4, math.Pi/2, false
	paint := ui.NewPaint()
	paint.SetColor(kBlue)
	painter := customPaint(func(c ui.Canvas) { c.DrawArc(rect, startAngle, sweepAngle, useCenter, paint) })

	Expect(t, painter, Paints().Arc(PaintPatternArcOptions{Rect: &rect}))
	deflated := rect.Deflate(10)
	expectFailureContaining(t, painter, Paints().Arc(PaintPatternArcOptions{Rect: &deflated}),
		"It called drawArc with a paint whose rect, Rect.fromLTRB(0.0, 0.0, 50.0, 50.0), was not exactly the "+
			"expected rect (Rect.fromLTRB(10.0, 10.0, 40.0, 40.0)).")
	Expect(t, painter, Paints().Arc(PaintPatternArcOptions{StartAngle: &startAngle}))
	expectFailureContaining(t, painter, Paints().Arc(PaintPatternArcOptions{StartAngle: pf(startAngle * 2)}),
		"It called drawArc with a start angle, 0.7853981633974483, which was not exactly the expected start angle (1.5707963267948966).")
	Expect(t, painter, Paints().Arc(PaintPatternArcOptions{SweepAngle: &sweepAngle}))
	expectFailureContaining(t, painter, Paints().Arc(PaintPatternArcOptions{SweepAngle: pf(sweepAngle * 2)}),
		"It called drawArc with a sweep angle, 1.5707963267948966, which was not exactly the expected sweep angle (3.141592653589793).")
	Expect(t, painter, Paints().Arc(PaintPatternArcOptions{UseCenter: &useCenter}))
	expectFailureContaining(t, painter, Paints().Arc(PaintPatternArcOptions{UseCenter: pb(!useCenter)}),
		"It called drawArc with a useCenter value, false, which was not exactly the expected value (true)")
	// Own: the strokeCap check (arc only).
	round := ui.StrokeCapRound
	expectFailureContaining(t, painter, Paints().Arc(PaintPatternArcOptions{StrokeCap: &round}),
		"It called drawArc with a paint whose strokeCap, StrokeCap.butt, was not exactly the expected strokeCap (StrokeCap.round).")
}

func TestRSuperellipse(t *testing.T) {
	rse := ui.NewRSuperellipseFromRectAndRadius(ui.NewRectFromLTWH(0, 0, 50, 50), ui.NewRadiusCircular(5))
	paint := ui.NewPaint()
	paint.SetColor(kBlue)
	painter := customPaint(func(c ui.Canvas) { c.DrawRSuperellipse(rse, paint) })
	Expect(t, painter, Paints().RSuperellipse(PaintPatternRSuperellipseOptions{RSuperellipse: &rse}))
	deflated := rse.Deflate(10)
	expectFailureContaining(t, painter, Paints().RSuperellipse(PaintPatternRSuperellipseOptions{RSuperellipse: &deflated}),
		"It called drawRSuperellipse with a rounded superellipse, RSuperellipse.fromLTRBR(0.0, 0.0, 50.0, 50.0, 5.0), which was "+
			"not exactly the expected rounded superellipse (RSuperellipse.fromLTRBR(10.0, 10.0, 40.0, 40.0, 0.0))")
}

// ---- recording_canvas_test.dart ----

var startColor = ui.NewColor(0xff00ff00)

// mutantPainter paints with a paint whose color it changes afterwards: the
// recorded paint must be a copy.
func mutantPainter(painter func(c ui.Canvas, p *ui.Paint)) func(ui.Canvas) {
	return customPaint(func(c ui.Canvas) {
		p := ui.NewPaint()
		p.SetColor(startColor)
		painter(c, p)
		p.SetColor(ui.NewColor(0xFF2196F3)) // Colors.blue
	})
}

func testImage(t *testing.T) *ui.Image {
	px := make([]byte, 8*8*4)
	var img *ui.Image
	ui.DecodeImageFromPixels(px, 8, 8, ui.PixelFormatRgba8888, func(i *ui.Image) { img = i })
	for loop := async.Current(); img == nil; {
		loop.FlushMicrotasks()
		if img == nil {
			loop.WaitPosted()
		}
	}
	return img
}

func paintColorIs(method string, index int) PaintPatternPredicate {
	return func(methodName string, arguments []any) bool {
		if methodName != method {
			return false
		}
		return arguments[index].(*ui.Paint).Color() == startColor
	}
}

func TestPaintsNotAffectedByMutatedColors(t *testing.T) {
	image := testImage(t)
	rect := ui.NewRectFromLTRB(0, 0, 100, 100)
	c := startColor
	po := PaintOptions{Color: &c}
	for _, tc := range []struct {
		name    string
		painter func(c ui.Canvas, p *ui.Paint)
		pattern *PaintPattern
	}{
		{"circle", func(c ui.Canvas, p *ui.Paint) { c.DrawCircle(ui.OffsetZero, 10.0, p) }, Paints().Circle(PaintPatternCircleOptions{PaintOptions: po})},
		{"rect", func(c ui.Canvas, p *ui.Paint) { c.DrawRect(rect, p) }, Paints().Rect(PaintPatternRectOptions{PaintOptions: po})},
		{"drawRRect", func(c ui.Canvas, p *ui.Paint) { c.DrawRRect(ui.NewRRectFromRectXY(rect, 4, 4), p) }, Paints().RRect(PaintPatternRRectOptions{PaintOptions: po})},
		{"drawDRRect", func(c ui.Canvas, p *ui.Paint) {
			c.DrawDRRect(ui.NewRRectFromRectXY(rect, 4, 4), ui.NewRRectFromRectXY(ui.NewRectFromLTRB(10, 10, 80, 80), 4, 4), p)
		}, Paints().DRRect(PaintPatternDRRectOptions{PaintOptions: po})},
		{"drawPath", func(c ui.Canvas, p *ui.Paint) { c.DrawPath(ui.NewPath(), p) }, Paints().Path(PaintPatternPathOptions{PaintOptions: po})},
		{"drawLine", func(c ui.Canvas, p *ui.Paint) { c.DrawLine(ui.OffsetZero, ui.NewOffset(10, 10), p) }, Paints().Line(PaintPatternLineOptions{PaintOptions: po})},
		{"drawArc", func(c ui.Canvas, p *ui.Paint) { c.DrawArc(rect, 10, 10, true, p) }, Paints().Arc(PaintPatternArcOptions{PaintOptions: po})},
		{"drawPaint", func(c ui.Canvas, p *ui.Paint) { c.DrawPaint(p) }, Paints().Something(paintColorIs("drawPaint", 0))},
		{"drawRSuperellipse", func(c ui.Canvas, p *ui.Paint) {
			c.DrawRSuperellipse(ui.NewRSuperellipseFromLTRBR(0, 0, 100, 100, ui.NewRadiusCircular(10)), p)
		}, Paints().RSuperellipse(PaintPatternRSuperellipseOptions{Color: &c})},
		{"drawOval", func(c ui.Canvas, p *ui.Paint) { c.DrawOval(rect, p) }, Paints().Something(paintColorIs("drawOval", 1))},
		{"image", func(c ui.Canvas, p *ui.Paint) { c.DrawImage(image, ui.OffsetZero, p) }, Paints().Something(paintColorIs("drawImage", 2))},
		{"drawImageRect", func(c ui.Canvas, p *ui.Paint) { c.DrawImageRect(image, rect, rect, p) }, Paints().DrawImageRect(PaintPatternDrawImageRectOptions{PaintOptions: po})},
		{"drawImageNine", func(c ui.Canvas, p *ui.Paint) { c.DrawImageNine(image, rect, rect, p) }, Paints().Something(paintColorIs("drawImageNine", 3))},
		{"drawPoints", func(c ui.Canvas, p *ui.Paint) {
			c.DrawPoints(ui.PointModeLines, []ui.Offset{ui.OffsetZero, ui.NewOffset(10, 10)}, p)
		}, Paints().Something(paintColorIs("drawPoints", 2))},
		{"drawRawPoints", func(c ui.Canvas, p *ui.Paint) { c.DrawRawPoints(ui.PointModeLines, []float32{0, 0, 10, 10}, p) },
			Paints().Something(paintColorIs("drawRawPoints", 2))},
		{"drawVertices", func(c ui.Canvas, p *ui.Paint) {
			c.DrawVertices(ui.NewVertices(ui.VertexModeTriangles, []ui.Offset{ui.OffsetZero, ui.NewOffset(0, 10), ui.NewOffset(10, 10)}), ui.BlendModeSrc, p)
		}, Paints().Something(paintColorIs("drawVertices", 2))},
		{"drawAtlas", func(c ui.Canvas, p *ui.Paint) {
			src := ui.BlendModeSrc
			c.DrawAtlas(image, []ui.RSTransform{}, []ui.Rect{}, []ui.Color{}, &src, &rect, p)
		}, Paints().Something(paintColorIs("drawAtlas", 6))},
		{"drawRawAtlas", func(c ui.Canvas, p *ui.Paint) {
			src := ui.BlendModeSrc
			c.DrawRawAtlas(image, []float32{}, []float32{}, []int32{}, &src, &rect, p)
		}, Paints().Something(paintColorIs("drawRawAtlas", 6))},
	} {
		t.Run(tc.name, func(t *testing.T) { Expect(t, mutantPainter(tc.painter), tc.pattern) })
	}
}

// ---- own tests ----

func TestPaintPatternDrawCommands(t *testing.T) {
	r := ui.NewRectFromLTRB(1, 2, 30, 40)
	rr := ui.NewRRectFromRectXY(r, 4, 4)
	inner := ui.NewRRectFromRectXY(ui.NewRectFromLTRB(5, 5, 10, 10), 1, 1)
	stroke := ui.NewPaint()
	stroke.SetColor(kBlue)
	stroke.SetStyle(ui.PaintingStyleStroke)
	stroke.SetStrokeWidth(2)
	blur := ui.NewMaskFilterBlur(ui.BlurStyleNormal, 3)
	blurred := ui.NewPaint()
	blurred.SetMaskFilter(&blur)
	path := ui.NewPath()
	path.AddRect(ui.NewRectFromLTRB(0, 0, 10, 10))
	painter := func(c ui.Canvas) {
		c.Translate(5, 6)
		c.DrawRect(r, stroke)
		c.DrawRRect(rr, blurred)
		c.DrawDRRect(rr, inner, stroke)
		c.DrawCircle(ui.NewOffset(3, 4), 5, stroke)
		c.DrawPath(path, stroke)
		c.DrawLine(ui.NewOffset(1, 1), ui.NewOffset(2, 2), stroke)
		c.ClipRect(r)
		c.ClipRRect(rr)
		c.Scale(2, 3)
		c.Rotate(0.5)
		c.DrawShadow(path, kBlue, 4, true)
	}
	blue := kBlue
	stroked := ui.PaintingStyleStroke
	fill := ui.PaintingStyleFill
	Expect(t, painter, Paints().
		Translate(PaintPatternTranslateOptions{X: pf(5), Y: pf(6)}).
		Rect(PaintPatternRectOptions{Rect: &r, PaintOptions: PaintOptions{Color: &blue, StrokeWidth: pf(2), Style: &stroked, HasMaskFilter: pb(false)}}).
		RRect(PaintPatternRRectOptions{RRect: &rr, PaintOptions: PaintOptions{HasMaskFilter: pb(true)}}).
		DRRect(PaintPatternDRRectOptions{Outer: &rr, Inner: &inner}).
		Circle(PaintPatternCircleOptions{X: pf(3), Y: pf(4), Radius: pf(5)}).
		Path(PaintPatternPathOptions{Includes: []ui.Offset{ui.NewOffset(5, 5)}, Excludes: []ui.Offset{ui.NewOffset(15, 5)}}).
		Line(PaintPatternLineOptions{P1: &ui.Offset{Dx: 1, Dy: 1}, P2: &ui.Offset{Dx: 2, Dy: 2}}).
		ClipRect(PaintPatternClipRectOptions{Rect: &r}).
		ClipRRect(PaintPatternClipRRectOptions{RRect: &rr}).
		Scale(PaintPatternScaleOptions{X: pf(2), Y: pf(3)}).
		Rotate(PaintPatternRotateOptions{Angle: pf(0.5)}).
		Shadow(PaintPatternShadowOptions{Includes: []ui.Offset{ui.NewOffset(5, 5)}, Color: &blue, Elevation: pf(4), TransparentOccluder: pb(true)}))
	// Skipping calls is allowed; order matters.
	Expect(t, painter, Paints().Circle().Line())
	Expect(t, painter, IsNot(Paints().Line().Circle()))

	other := ui.NewRectFromLTRB(0, 0, 1, 1)
	for _, c := range []struct {
		pattern *PaintPattern
		want    string
	}{
		{Paints().Rect(PaintPatternRectOptions{Rect: &other}),
			"It called drawRect with Rect, Rect.fromLTRB(1.0, 2.0, 30.0, 40.0), which was not exactly the expected Rect (Rect.fromLTRB(0.0, 0.0, 1.0, 1.0))."},
		{Paints().Rect(PaintPatternRectOptions{PaintOptions: PaintOptions{StrokeWidth: pf(3)}}),
			"It called drawRect with a paint whose strokeWidth, 2.0, was not exactly the expected strokeWidth (3.0)."},
		{Paints().Rect(PaintPatternRectOptions{PaintOptions: PaintOptions{Style: &fill}}),
			"It called drawRect with a paint whose style, PaintingStyle.stroke, was not exactly the expected style (PaintingStyle.fill)."},
		{Paints().Rect(PaintPatternRectOptions{PaintOptions: PaintOptions{HasMaskFilter: pb(true)}}),
			"It called drawRect with a paint that did not have a mask filter, despite expecting one."},
		{Paints().RRect(PaintPatternRRectOptions{PaintOptions: PaintOptions{HasMaskFilter: pb(false)}}),
			"It called drawRRect with a paint that had a mask filter, despite not expecting one."},
		{Paints().Rect(PaintPatternRectOptions{PaintOptions: PaintOptions{Color: &kTransparent}}),
			"It called drawRect with a paint whose color, " + kBlue.String() + ", was not exactly the expected color (" + kTransparent.String() + ")."},
		{Paints().RRect(PaintPatternRRectOptions{RRect: &inner}),
			"It called drawRRect with RRect, " + rr.String() + ", which was not exactly the expected RRect (" + inner.String() + ")."},
		{Paints().DRRect(PaintPatternDRRectOptions{Inner: &rr}),
			"It called drawDRRect with its second argument (a RRect), " + inner.String() + ", which was not exactly the expected RRect (" + rr.String() + ")."},
		{Paints().DRRect(PaintPatternDRRectOptions{Outer: &inner}),
			"It called drawDRRect with its first argument (a RRect), " + rr.String() + ", which was not exactly the expected RRect (" + inner.String() + ")."},
		{Paints().Circle(PaintPatternCircleOptions{X: pf(3), Y: pf(5)}),
			"It called drawCircle with a center coordinate, Offset(3.0, 4.0), which was not exactly the expected coordinate (Offset(3.0, 5.0))."},
		{Paints().Circle(PaintPatternCircleOptions{X: pf(9)}),
			"It called drawCircle with a center coordinate, Offset(3.0, 4.0), whose x-coordinate was not exactly the expected coordinate (9.0)."},
		{Paints().Circle(PaintPatternCircleOptions{Y: pf(9)}),
			"It called drawCircle with a center coordinate, Offset(3.0, 4.0), whose y-coordinate was not exactly the expected coordinate (9.0)."},
		{Paints().Circle(PaintPatternCircleOptions{Radius: pf(6)}),
			"It called drawCircle with radius, 5.0, which was not exactly the expected radius (6.0)."},
		{Paints().Path(PaintPatternPathOptions{Includes: []ui.Offset{ui.NewOffset(15, 5)}}),
			"It called drawPath with a path that unexpectedly did not contain Offset(15.0, 5.0)."},
		{Paints().Path(PaintPatternPathOptions{Excludes: []ui.Offset{ui.NewOffset(5, 5)}}),
			"It called drawPath with a path that unexpectedly contained Offset(5.0, 5.0)."},
		{Paints().Line(PaintPatternLineOptions{P1: &ui.Offset{Dx: 0, Dy: 1}}),
			"It called drawLine with p1 endpoint, Offset(1.0, 1.0), which was not exactly the expected endpoint (Offset(0.0, 1.0))."},
		{Paints().Line(PaintPatternLineOptions{P2: &ui.Offset{Dx: 0, Dy: 1}}),
			"It called drawLine with p2 endpoint, Offset(2.0, 2.0), which was not exactly the expected endpoint (Offset(0.0, 1.0))."},
		{Paints().Translate(PaintPatternTranslateOptions{X: pf(7)}),
			"It called translate with argument 0 having value 5.0 when 7.0 was expected."},
		{Paints().Scale(PaintPatternScaleOptions{Y: pf(7)}),
			"It called scale with argument 1 having value 3.0 when 7.0 was expected."},
		{Paints().Shadow(PaintPatternShadowOptions{Elevation: pf(1)}),
			"It called drawShadow with an elevation, 4.0, which was not exactly the expected value (1.0)."},
		{Paints().Shadow(PaintPatternShadowOptions{TransparentOccluder: pb(false)}),
			"It called drawShadow with a transparentOccluder value, true, which was not exactly the expected value (false)."},
		{Paints().Shadow(PaintPatternShadowOptions{Color: &kTransparent}),
			"It called drawShadow with a color, " + kBlue.String() + ", which was not exactly the expected color (" + kTransparent.String() + ")."},
		{Paints().Shadow(PaintPatternShadowOptions{Excludes: []ui.Offset{ui.NewOffset(5, 5)}}),
			"It called drawShadow with a path that unexpectedly contained Offset(5.0, 5.0)."},
		{Paints().Rotate().Translate(),
			"It called 1 other method on the canvas, the first of which was drawShadow(Path, " + kBlue.String() +
				", 4.0, true), but did not call translate() at the time where translate(..., ...) was expected."},
		{Paints().Arc(),
			"It called 12 other methods on the canvas, the first of which was translate(5.0, 6.0), but did not call drawArc() at the time where an arc was expected."},
		{Paints().Shadow().Rect(), "type 'Null' is not a subtype of type 'RecordedInvocation' in type cast"},
	} {
		expectFailureContaining(t, painter, c.pattern, c.want)
	}
}

func TestPaintPatternDescriptions(t *testing.T) {
	r := ui.NewRectFromLTRB(1, 2, 3, 4)
	rr := ui.NewRRectFromRectXY(r, 1, 1)
	blue := kBlue
	stroked := ui.PaintingStyleStroke
	p := Paints().
		Rect(PaintPatternRectOptions{Rect: &r, PaintOptions: PaintOptions{Color: &blue, StrokeWidth: pf(2), HasMaskFilter: pb(false), Style: &stroked}}).
		RRect(PaintPatternRRectOptions{RRect: &rr}).
		DRRect(PaintPatternDRRectOptions{Outer: &rr}).
		Circle(PaintPatternCircleOptions{X: pf(1), Radius: pf(2)}).
		Circle(PaintPatternCircleOptions{X: pf(1), Y: pf(2)}).
		Path(PaintPatternPathOptions{Includes: []ui.Offset{ui.NewOffset(1, 2)}, Excludes: []ui.Offset{}}).
		Line(PaintPatternLineOptions{P1: &ui.Offset{Dx: 1, Dy: 2}}).
		Arc(PaintPatternArcOptions{StartAngle: pf(1), UseCenter: pb(true)}).
		Shadow(PaintPatternShadowOptions{Elevation: pf(3)}).
		Translate(PaintPatternTranslateOptions{X: pf(1)}).
		SaveRestore().
		Something(func(string, []any) bool { return true }).
		Everything(func(string, []any) bool { return true })
	want := "Object or closure painting:\n" +
		"a rectangle with " + kBlue.String() + ", strokeWidth: 2.0, no mask filter, PaintingStyle.stroke, Rect.fromLTRB(1.0, 2.0, 3.0, 4.0)\n" +
		"a rounded rectangle with RRect: " + rr.String() + "\n" +
		"a rounded rectangle outline with " + rr.String() + "\n" +
		"a circle with x-coordinate 1.0, radius 2.0\n" +
		"a circle with point Offset(1.0, 2.0)\n" +
		"a path with that contains [Offset(1.0, 2.0)] and does not contain []\n" +
		"a line with end point p1: Offset(1.0, 2.0)\n" +
		"an arc with startAngle 1.0, useCenter true\n" +
		"drawShadow with elevation: 3.0\n" +
		"translate(1.0, ...)\n" +
		"a matching save/restore pair\n" +
		`a "something" step` + "\n" +
		`an "everything" step`
	if got := p.Describe(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if got := Paints().Describe(); got != "An object or closure and a paint pattern." {
		t.Fatal(got)
	}
}

func TestPaintPatternEmptyAndUnsupported(t *testing.T) {
	expectFailureContaining(t, func(ui.Canvas) {}, Paints().Rect(), "did not match the pattern.\nIt painted nothing.")
	expectFailureContaining(t, myPainter(kBlue), Paints(), "It painted something, but you must now add a pattern")
	expectFailureContaining(t, 42, Paints(), `was not one of the supported objects for the "paints" matcher.`)
	// The complete display list is listed on failure.
	expectFailureContaining(t, myPainter(kBlue), Paints().Rect(),
		"The complete display list was:\n  * save()\n  * drawColor("+kBlue.String()+", BlendMode.color)\n  * restore()")
	// A painter that throws.
	expectFailureContaining(t, func(ui.Canvas) { panic("boom") }, Paints().Rect(), "threw the following exception:\nboom\n")
}

func TestSaveRestore(t *testing.T) {
	painter := func(c ui.Canvas) {
		c.Save()
		c.Save()
		c.DrawPaint(ui.NewPaint())
		c.Restore()
		c.Restore()
		c.DrawRect(ui.NewRectFromLTRB(0, 0, 1, 1), ui.NewPaint())
	}
	Expect(t, painter, Paints().SaveRestore().Rect())
	Expect(t, painter, IsNot(Paints().SaveRestore().Restore()))
	Expect(t, painter, Paints().Save().Save().Restore().Restore())
	expectFailureContaining(t, func(c ui.Canvas) { c.Save(); c.DrawPaint(ui.NewPaint()) }, Paints().SaveRestore(),
		"It did not have a matching restore() for the save() that was found where a matching save/restore pair was expected.")
}

func TestClipPathAndTransform(t *testing.T) {
	path := ui.NewPath()
	path.AddRect(ui.NewRectFromLTRB(0, 0, 10, 10))
	m := []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 3, 4, 0, 1}
	painter := func(c ui.Canvas) {
		c.ClipPath(path)
		c.Transform(m)
	}
	Expect(t, painter, Paints().
		ClipPath(PaintPatternClipPathOptions{PathMatcher: IsPathThat(IsPathThatOptions{Includes: []ui.Offset{ui.NewOffset(5, 5)}})}).
		Transform(PaintPatternTransformOptions{Matrix4: m}))
	expectFailureContaining(t, painter,
		Paints().ClipPath(PaintPatternClipPathOptions{PathMatcher: IsPathThat(IsPathThatOptions{Excludes: []ui.Offset{ui.NewOffset(5, 5)}})}),
		"Offset(5.0, 5.0) should be outside the path, but is not.")
	expectFailureContaining(t, painter, Paints().Transform(PaintPatternTransformOptions{Matrix4: make([]float64, 16)}),
		"It called transform with argument 0 having value")
}

func TestIsPathThat(t *testing.T) {
	path := ui.NewPath()
	path.AddRect(ui.NewRectFromLTRB(0, 0, 10, 10))
	m := IsPathThat(IsPathThatOptions{Includes: []ui.Offset{ui.NewOffset(5, 5)}, Excludes: []ui.Offset{ui.NewOffset(20, 5), ui.NewOffset(5, 20)}})
	Expect(t, path, m)
	if m.Describe() != "A Path that contains one particular point but does not contain 2 particular points." {
		t.Fatal(m.Describe())
	}
	ok, msg := matches(path, IsPathThat(IsPathThatOptions{Includes: []ui.Offset{ui.NewOffset(20, 5)}}))
	if ok || msg != "Not all the given points were inside or outside the path as expected:\n  Offset Offset(20.0, 5.0) should be inside the path, but is not." {
		t.Fatal(msg)
	}
	if ok, msg := matches(3, m); ok || msg != "The given object (3) was not a Path." {
		t.Fatal(msg)
	}
}

func TestPaintsNothingCountAssertion(t *testing.T) {
	Expect(t, func(c ui.Canvas) { c.Save(); c.Restore() }, PaintsNothing())
	expectFailureContaining(t, myPainter(kBlue), PaintsNothing(), "painted something, the first call having the following stack:")
	painter := func(c ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTRB(0, 0, 1, 1), ui.NewPaint())
		c.DrawRect(ui.NewRectFromLTRB(0, 0, 1, 1), ui.NewPaint())
	}
	Expect(t, painter, PaintsExactlyCountTimes("drawRect", 2))
	expectFailureContaining(t, painter, PaintsExactlyCountTimes("drawRect", 3), `It painted Symbol("drawRect") 2 times instead of 3 times.`)
	if got := PaintsExactlyCountTimes("drawRect", 3).Describe(); got != `Object or closure painting Symbol("drawRect") exactly 3 times` {
		t.Fatal(got)
	}
	Expect(t, func(ui.Canvas) { panic(foundation.NewFlutterError("bad")) }, PaintsAssertion())
	expectFailureContaining(t, painter, PaintsAssertion(), "did not assert.")
	expectFailureContaining(t, func(ui.Canvas) { panic("x") }, PaintsAssertion(), "threw the following exception:\nx")
}

func TestImagePatternAlwaysFails(t *testing.T) {
	image := testImage(t)
	// As in Flutter 3.50.1: the image predicate casts the image to an
	// Offset (shape_decoration_test relies on isNot(paints..image())).
	painter := func(c ui.Canvas) { c.DrawImage(image, ui.OffsetZero, ui.NewPaint()) }
	expectFailureContaining(t, painter, Paints().Image(), "type 'Image' is not a subtype of type 'Offset' in type cast")
	Expect(t, painter, IsNot(Paints().Image()))
	rect := ui.NewRectFromLTRB(0, 0, 8, 8)
	other := ui.NewRectFromLTRB(0, 0, 4, 4)
	p2 := func(c ui.Canvas) { c.DrawImageRect(image, rect, rect, ui.NewPaint()) }
	Expect(t, p2, Paints().DrawImageRect(PaintPatternDrawImageRectOptions{Image: image, Source: &rect, Destination: &rect}))
	expectFailureContaining(t, p2, Paints().DrawImageRect(PaintPatternDrawImageRectOptions{Source: &other}), "with a source rectangle")
	expectFailureContaining(t, p2, Paints().DrawImageRect(PaintPatternDrawImageRectOptions{Destination: &other}), "with a destination rectangle")
	expectFailureContaining(t, p2, Paints().DrawImageRect(PaintPatternDrawImageRectOptions{Image: testImage(t)}), "with an image")
}

func TestTestRecordingCanvas(t *testing.T) {
	c := NewTestRecordingCanvas()
	c.Save()
	b := ui.NewRectFromLTRB(0, 0, 1, 2)
	c.SaveLayer(&b, ui.NewPaint())
	c.SaveLayer(nil, ui.NewPaint())
	if c.GetSaveCount() != 3 {
		t.Fatal(c.GetSaveCount())
	}
	aa := false
	c.ClipRect(b, ui.CanvasClipRectOptions{DoAntiAlias: &aa})
	c.ClipPath(ui.NewPath())
	c.Scale(2)
	c.Rotate(0.25)
	c.Restore()
	c.Restore()
	c.Restore()
	var got []string
	for _, inv := range c.Invocations {
		got = append(got, inv.String())
	}
	want := "save()|saveLayer(Rect.fromLTRB(0.0, 0.0, 1.0, 2.0), " + ui.NewPaint().String() + ")|saveLayer(null, " +
		ui.NewPaint().String() + ")|clipRect(Rect.fromLTRB(0.0, 0.0, 1.0, 2.0), doAntiAlias: false)|clipPath(Path)|scale(2.0)|rotate(0.3)|restore()|restore()|restore()"
	if strings.Join(got, "|") != want {
		t.Fatalf("got  %s\nwant %s", strings.Join(got, "|"), want)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("restore below zero did not assert")
			}
		}()
		c.Restore()
	}()
	if s := c.Invocations[0].StackToString(StackToStringOptions{Indent: "  "}); !strings.HasPrefix(s, "  #") {
		t.Fatalf("%q", s)
	}
}
