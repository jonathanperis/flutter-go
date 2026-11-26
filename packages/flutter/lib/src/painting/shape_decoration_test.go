package painting_test

// Port of packages/flutter/test/painting/shape_decoration_test.dart (Flutter
// 3.50.1). Every test is ported. Dart's identical() on canonicalized
// const decorations is pointer identity here (the "identical" test passes
// the same pointer); `nonconst` is not needed (Go has no const objects).

import (
	"fmt"
	"math"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func shapeDeco(o ShapeDecorationOptions) *ShapeDecoration { return NewShapeDecoration(o) }

func TestShapeDecorationConstructor(t *testing.T) {
	colorR, colorG := ui.NewColor(0xffff0000), ui.NewColor(0xff00ff00)
	gradient := NewLinearGradient(LinearGradientOptions{Colors: []ui.Color{colorR, colorG}})
	boxEq(t, "Border()", shapeDeco(ShapeDecorationOptions{Shape: NewBorder()}), shapeDeco(ShapeDecorationOptions{Shape: NewBorder()}))
	bdExpectAssertion(t, func() { shapeDeco(ShapeDecorationOptions{Color: &colorR, Gradient: gradient, Shape: NewBorder()}) })
	boxEq(t, "circle", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOptions{Shape: BoxShapeCircle})),
		shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder()}))
	radius := NewBorderRadiusDirectionalAll(ui.NewRadiusCircular(100.0))
	boxEq(t, "rounded", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOptions{BorderRadius: radius})),
		shapeDeco(ShapeDecorationOptions{Shape: NewRoundedRectangleBorder(RRectBorderOptions{BorderRadius: radius})}))
	sideG := NewBorderSide(BorderSideOptions{Color: &colorG})
	boxEq(t, "circle with border", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOptions{
		Shape: BoxShapeCircle, Border: NewBorderAll(BorderSideOptions{Color: &colorG})})),
		shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder(CircleBorderOptions{Side: &sideG})}))
	boxEq(t, "border", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOptions{
		Border: NewBorderAll(BorderSideOptions{Color: &colorR})})),
		shapeDeco(ShapeDecorationOptions{Shape: NewBorderAll(BorderSideOptions{Color: &colorR})}))
	start := NewBorderSide()
	boxEq(t, "directional", NewShapeDecorationFromBoxDecoration(NewBoxDecoration(BoxDecorationOptions{
		Border: NewBorderDirectional(BorderDirectionalOptions{Start: &start})})),
		shapeDeco(ShapeDecorationOptions{Shape: NewBorderDirectional(BorderDirectionalOptions{Start: &start})}))
}

func TestShapeDecorationLerpIdentical(t *testing.T) {
	if ShapeDecorationLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	shape := shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder()})
	if ShapeDecorationLerp(shape, shape, 0.5) != shape {
		t.Error("identical")
	}
}

func TestShapeDecorationLerpNull(t *testing.T) {
	var a Decoration = shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder()})
	var b Decoration = shapeDeco(ShapeDecorationOptions{Shape: NewRoundedRectangleBorder()})
	boxEq(t, "a, null", DecorationLerp(a, nil, 0.0), a)
	boxEq(t, "null, b", DecorationLerp(nil, b, 0.0), b)
	if DecorationLerp(nil, nil, 0.0) != nil {
		t.Error("null, null")
	}
}

func TestShapeDecorationLerpAndHitTest(t *testing.T) {
	var a Decoration = shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder()})
	var b Decoration = shapeDeco(ShapeDecorationOptions{Shape: NewRoundedRectangleBorder()})
	var c Decoration = shapeDeco(ShapeDecorationOptions{Shape: NewOvalBorder()})
	boxEq(t, "ab0", DecorationLerp(a, b, 0.0), a)
	boxEq(t, "ab1", DecorationLerp(a, b, 1.0), b)
	boxEq(t, "ac0", DecorationLerp(a, c, 0.0), a)
	boxEq(t, "ac1", DecorationLerp(a, c, 1.0), c)
	boxEq(t, "bc0", DecorationLerp(b, c, 0.0), b)
	boxEq(t, "bc1", DecorationLerp(b, c, 1.0), c)
	size := ui.NewSize(200.0, 100.0) // at t=0.5, width will be 150 (x=25 to x=175).
	hit := func(name string, d Decoration, x, y float64, want bool) {
		t.Helper()
		if d.HitTest(size, ui.NewOffset(x, y), nil) != want {
			t.Errorf("%s (%v, %v): want %v", name, x, y, want)
		}
	}
	hit("a", a, 20, 50, false)
	hit("c", c, 50, 5, false)
	hit("c", c, 5, 30, false)
	hit("ab.1", DecorationLerp(a, b, 0.1), 20, 50, false)
	hit("ab.5", DecorationLerp(a, b, 0.5), 20, 50, false)
	hit("ab.9", DecorationLerp(a, b, 0.9), 20, 50, true)
	hit("ac.1", DecorationLerp(a, c, 0.1), 30, 50, false)
	hit("ac.5", DecorationLerp(a, c, 0.5), 30, 50, true)
	hit("ac.9", DecorationLerp(a, c, 0.9), 30, 50, true)
	hit("bc.1", DecorationLerp(b, c, 0.1), 45, 10, true)
	hit("bc.5", DecorationLerp(b, c, 0.5), 30, 10, true)
	hit("bc.9", DecorationLerp(b, c, 0.9), 10, 30, true)
	hit("b", b, 20, 50, true)
}

var outerPathCount, hitTestCount int

// pathHitTestBorder is _PathHitTestBorder.
type pathHitTestBorder struct{ ShapeBorderBase }

func newPathHitTestBorder() *pathHitTestBorder { b := &pathHitTestBorder{}; b.SetSelf(b); return b }

func (b *pathHitTestBorder) Dimensions() EdgeInsetsGeometry { return EdgeInsetsZero }
func (b *pathHitTestBorder) Scale(float64) ShapeBorder      { return b }
func (b *pathHitTestBorder) GetInnerPath(ui.Rect, *ui.TextDirection) *ui.Path {
	panic(dart.StateError{Message: "ShapeBorder.hitTest should not call getInnerPath."})
}
func (b *pathHitTestBorder) GetOuterPath(rect ui.Rect, _ *ui.TextDirection) *ui.Path {
	outerPathCount++
	p := ui.NewPath()
	p.AddRect(rect)
	return p
}
func (b *pathHitTestBorder) Paint(ui.Canvas, ui.Rect, *ui.TextDirection) {}

// hitTestBorder is _HitTestBorder.
type hitTestBorder struct{ ShapeBorderBase }

func newHitTestBorder() *hitTestBorder { b := &hitTestBorder{}; b.SetSelf(b); return b }

func (b *hitTestBorder) Dimensions() EdgeInsetsGeometry { return EdgeInsetsZero }
func (b *hitTestBorder) Scale(float64) ShapeBorder      { return b }
func (b *hitTestBorder) GetInnerPath(ui.Rect, *ui.TextDirection) *ui.Path {
	panic(dart.StateError{Message: "ShapeDecoration.hitTest should not call getInnerPath."})
}
func (b *hitTestBorder) GetOuterPath(ui.Rect, *ui.TextDirection) *ui.Path {
	panic(dart.StateError{Message: "hitTest should not call getOuterPath directly."})
}
func (b *hitTestBorder) HitTest(ui.Rect, ui.Offset, *ui.TextDirection) bool {
	hitTestCount++
	return false
}
func (b *hitTestBorder) Paint(ui.Canvas, ui.Rect, *ui.TextDirection) {}

func TestShapeBorderHitTestDefaultsToGetOuterPath(t *testing.T) {
	outerPathCount = 0
	border := newPathHitTestBorder()
	rect := ui.NewRectFromLTWH(10.0, 20.0, 100.0, 50.0)
	if !border.HitTest(rect, ui.NewOffset(20.0, 30.0), nil) || border.HitTest(rect, ui.OffsetZero, nil) {
		t.Error("hitTest")
	}
	if outerPathCount != 2 {
		t.Errorf("outerPathCount %d", outerPathCount)
	}
}

func TestShapeDecorationHitTestDelegates(t *testing.T) {
	hitTestCount = 0
	decoration := shapeDeco(ShapeDecorationOptions{Shape: newHitTestBorder()})
	if decoration.HitTest(ui.NewSize(100.0, 100.0), ui.NewOffset(50.0, 50.0), nil) {
		t.Error("hit")
	}
	if hitTestCount != 1 {
		t.Errorf("hitTestCount %d", hitTestCount)
	}
}

func TestShapeBorderHitTestMatchesGetOuterPath(t *testing.T) {
	rect := ui.NewRectFromLTWH(0.0, 0.0, 120.0, 80.0)
	textDirection := bdTD(ui.TextDirectionLtr)
	positions := []ui.Offset{
		ui.NewOffset(-10.0, 40.0), // Outside every shape.
		ui.NewOffset(1.0, 1.0),    // Distinguishes square and rounded corners.
		ui.NewOffset(0.5, 14.5),   // Distinguishes RRect and superellipse corners.
		ui.NewOffset(1.0, 35.0),   // Exercises the left edge of wide shapes.
		ui.NewOffset(1.0, 68.0),   // Exercises the directional bottom-left corner.
		ui.NewOffset(11.0, 35.0),  // Exercises CircleBorder eccentricity.
		ui.NewOffset(14.5, 13.5),  // Distinguishes interpolated outer shapes.
		ui.NewOffset(21.0, 35.0),  // Inside every shape.
	}
	r18 := NewBorderRadiusAll(ui.NewRadiusCircular(18.0))
	borders := []ShapeBorder{
		NewBorderAll(),
		NewCircleBorder(),
		NewCircleBorder(CircleBorderOptions{Eccentricity: 0.5}),
		NewOvalBorder(),
		NewRoundedRectangleBorder(),
		NewRoundedRectangleBorder(RRectBorderOptions{BorderRadius: r18}),
		NewRoundedRectangleBorder(RRectBorderOptions{BorderRadius: BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(18.0)}}),
		NewRoundedSuperellipseBorder(),
		NewRoundedSuperellipseBorder(RRectBorderOptions{BorderRadius: r18}),
		NewStadiumBorder(),
		ShapeBorderLerp(NewCircleBorder(), NewRoundedRectangleBorder(RRectBorderOptions{BorderRadius: r18}), 0.5),
		ShapeBorderLerp(NewCircleBorder(), NewRoundedSuperellipseBorder(RRectBorderOptions{BorderRadius: r18}), 0.5),
		ShapeBorderLerp(NewStadiumBorder(), NewCircleBorder(), 0.5),
		ShapeBorderLerp(NewStadiumBorder(), NewRoundedRectangleBorder(RRectBorderOptions{BorderRadius: r18}), 0.5),
	}
	for _, border := range borders {
		for _, position := range positions {
			if border.HitTest(rect, position, textDirection) != border.GetOuterPath(rect, textDirection).Contains(position) {
				t.Errorf("%v at %v", border, position)
			}
		}
	}
}

func TestCompoundBorderHitTestPreservesChildHitTest(t *testing.T) {
	hitTestCount = 0
	compoundBorder := NewRoundedRectangleBorder().Add(newHitTestBorder())
	if compoundBorder.HitTest(ui.NewRectFromLTWH(0.0, 0.0, 100.0, 100.0), ui.OffsetZero, nil) {
		t.Error("hit")
	}
	if hitTestCount != 1 {
		t.Errorf("hitTestCount %d", hitTestCount)
	}
}

func TestShapeDecorationLerpGradientAndColor(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/93953
	colorR, colorG := ui.NewColor(0xffff0000), ui.NewColor(0xff00ff00)
	gradient := NewLinearGradient(LinearGradientOptions{Colors: []ui.Color{colorR, colorG}})
	colorDecoration := shapeDeco(ShapeDecorationOptions{Color: &colorR, Shape: NewCircleBorder()})
	gradientDecoration := shapeDeco(ShapeDecorationOptions{Gradient: gradient, Shape: NewRoundedRectangleBorder()})
	boxEq(t, "0", ShapeDecorationLerp(colorDecoration, gradientDecoration, 0.0), colorDecoration)
	boxEq(t, "1", ShapeDecorationLerp(colorDecoration, gradientDecoration, 1.0), gradientDecoration)
	for _, x := range []float64{0.1, 0.25, 0.49, 0.5, 0.51, 0.75, 0.9} {
		for _, d := range []*ShapeDecoration{ShapeDecorationLerp(colorDecoration, gradientDecoration, x),
			ShapeDecorationLerp(gradientDecoration, colorDecoration, x)} {
			if d.Color != nil {
				t.Errorf("t=%v: color %v", x, d.Color)
			}
			if _, ok := d.Gradient.(*LinearGradient); !ok {
				t.Errorf("t=%v: gradient %T", x, d.Gradient)
			}
		}
	}
	near := ShapeDecorationLerp(colorDecoration, gradientDecoration, 0.001).Gradient.(*LinearGradient)
	if len(near.Colors()) != 2 {
		t.Fatalf("colors %v", near.Colors())
	}
	for _, c := range near.Colors() {
		if math.Abs(c.R-colorR.R) > 0.05 || math.Abs(c.G-colorR.G) > 0.05 || math.Abs(c.B-colorR.B) > 0.05 {
			t.Errorf("color %v", c)
		}
	}
}

// shapeTestImageProvider is the test's TestImageProvider.
type shapeTestImageProvider struct {
	ImageProviderBase
	image *ui.Image
}

func newShapeTestImageProvider(image *ui.Image) *shapeTestImageProvider {
	p := &shapeTestImageProvider{image: image}
	p.SetSelf(p)
	return p
}

func (p *shapeTestImageProvider) ObtainKey(ImageConfiguration) async.Future[any] {
	return foundation.NewSynchronousFuture[any](p)
}

func (p *shapeTestImageProvider) LoadImage(any, ImageDecoderCallback) ImageStreamCompleter {
	return NewOneFrameImageStreamCompleter(foundation.NewSynchronousFuture(NewImageInfo(p.image)))
}

func TestShapeDecorationImageRTL(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, ft.CreateTestImageOptions{Width: 100, Height: 200})
	var log []int
	decoration := shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder(),
		Image: NewDecorationImage(DecorationImageOptions{Image: newShapeTestImageProvider(image),
			Alignment: AlignmentDirectionalBottomEnd})})
	painter := decoration.CreateBoxPainter(func() { log = append(log, 0) })
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.OffsetZero, sizeConfig(100.0, 100.0))
	}, ft.PaintsAssertion())
	size := ui.NewSize(1000.0, 1000.0)
	src := ui.NewRectFromLTRB(0.0, 0.0, 100.0, 200.0)
	dst := ui.NewRectFromLTRB(20.0, 1000.0-40.0-200.0, 20.0+100.0, 1000.0-40.0)
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.NewOffset(20.0, -40.0), ImageConfiguration{Size: &size, TextDirection: bdTD(ui.TextDirectionRtl)})
	}, ft.Paints().DrawImageRect(ft.PaintPatternDrawImageRectOptions{Source: &src, Destination: &dst}))
	size2 := ui.NewSize(100.0, 200.0)
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.OffsetZero, ImageConfiguration{Size: &size2, TextDirection: bdTD(ui.TextDirectionLtr)})
	}, ft.IsNot(ft.Paints().Image())) // we always use drawImageRect
	if len(log) != 0 {
		t.Errorf("log %v", log)
	}
}

func TestShapeDecorationGetClipPath(t *testing.T) {
	decoration := shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder()})
	clipPath := decoration.GetClipPath(ui.NewRectFromLTWH(0.0, 0.0, 100.0, 20.0), ui.TextDirectionLtr)
	ft.Expect(t, clipPath, ft.IsPathThat(ft.IsPathThatOptions{
		Includes: []ui.Offset{ui.NewOffset(50.0, 10.0)},
		Excludes: []ui.Offset{ui.NewOffset(1.0, 1.0), ui.NewOffset(30.0, 10.0), ui.NewOffset(99.0, 19.0)}}))
}

func TestShapeDecorationGetClipPathOval(t *testing.T) {
	decoration := shapeDeco(ShapeDecorationOptions{Shape: NewOvalBorder()})
	clipPath := decoration.GetClipPath(ui.NewRectFromLTWH(0.0, 0.0, 100.0, 50.0), ui.TextDirectionLtr)
	ft.Expect(t, clipPath, ft.IsPathThat(ft.IsPathThatOptions{
		Includes: []ui.Offset{ui.NewOffset(50.0, 10.0)},
		Excludes: []ui.Offset{ui.NewOffset(1.0, 1.0), ui.NewOffset(15.0, 1.0), ui.NewOffset(99.0, 19.0)}}))
}

// Go-only: createBoxPainter's null check (Dart's onChanged! throws even
// without an image), padding, isComplex and the lerp's null-shape check.
func TestShapeDecorationMembers(t *testing.T) {
	d := shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder(CircleBorderOptions{Side: bdSide(NewBorderSide(BorderSideOptions{Width: bdF(4)}))})})
	func() {
		defer func() {
			if r := recover(); fmt.Sprint(r) != "Null check operator used on a null value" {
				t.Errorf("createBoxPainter(null): %v", r)
			}
		}()
		d.CreateBoxPainter(nil)
	}()
	if d.Padding() != EdgeInsetsGeometry(NewEdgeInsetsAll(4)) {
		t.Errorf("padding %v", d.Padding())
	}
	if d.IsComplex() || !shapeDeco(ShapeDecorationOptions{Shape: NewCircleBorder(), Shadows: []BoxShadow{}}).IsComplex() {
		t.Error("isComplex")
	}
}
