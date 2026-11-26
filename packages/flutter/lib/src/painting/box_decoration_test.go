package painting_test

// Port of packages/flutter/test/painting/box_decoration_test.dart (Flutter
// 3.50.1), plus the BoxDecoration tests of decoration_test.dart that the
// decoration port deferred ("Decoration.lerp()", "Decoration.lerp
// identical a,b", "Decoration equality", "BoxDecorationImageListenerSync",
// "BoxDecorationImageListenerAsync", "BoxDecorationImageListener does not
// change when image is clone", "BoxDecoration backgroundImage clip",
// "BoxDecoration.lerp - shapes", "BoxDecoration.lerp - gradients",
// "Decoration.lerp with unrelated decorations") and border_test.dart's
// "BoxDecoration.border takes a BorderDirectional".
//
// Skipped: hashCode expectations (Go values have no hashCode; equality is
// Equal). Dart's identical() on canonicalized const decorations is pointer
// identity here: the "identical" tests pass the same pointer.

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// testBoxBorder is the test's _TestBoxBorder: a custom BoxBorder subclass.
type testBoxBorder struct {
	ShapeBorderBase
	width float64
}

func newTestBoxBorder(width float64) *testBoxBorder {
	b := &testBoxBorder{width: width}
	b.SetSelf(b)
	return b
}

func (b *testBoxBorder) Bottom() BorderSide {
	return NewBorderSide(BorderSideOptions{Width: bdF(b.width)})
}
func (b *testBoxBorder) Top() BorderSide {
	return NewBorderSide(BorderSideOptions{Width: bdF(b.width)})
}
func (b *testBoxBorder) Dimensions() EdgeInsetsGeometry {
	return NewEdgeInsetsAll(b.width)
}
func (b *testBoxBorder) IsUniform() bool                                         { return true }
func (b *testBoxBorder) PaintBox(ui.Canvas, ui.Rect, BoxBorderPaintOptions)      {}
func (b *testBoxBorder) Paint(ui.Canvas, ui.Rect, *ui.TextDirection)             {}
func (b *testBoxBorder) GetOuterPath(rect ui.Rect, _ *ui.TextDirection) *ui.Path { return ui.NewPath() }
func (b *testBoxBorder) GetInnerPath(rect ui.Rect, _ *ui.TextDirection) *ui.Path { return ui.NewPath() }
func (b *testBoxBorder) Scale(t float64) ShapeBorder                             { return newTestBoxBorder(b.width * t) }

func (b *testBoxBorder) LerpFrom(a ShapeBorder, t float64) ShapeBorder {
	if a, ok := a.(*testBoxBorder); ok {
		return newTestBoxBorder(*ui.LerpDouble(&a.width, &b.width, t))
	}
	return b.ShapeBorderBase.LerpFrom(a, t)
}

func (b *testBoxBorder) LerpTo(o ShapeBorder, t float64) ShapeBorder {
	if o, ok := o.(*testBoxBorder); ok {
		return newTestBoxBorder(*ui.LerpDouble(&b.width, &o.width, t))
	}
	return b.ShapeBorderBase.LerpTo(o, t)
}

func (b *testBoxBorder) Equal(other ShapeBorder) bool {
	o, ok := other.(*testBoxBorder)
	return ok && o.width == b.width
}

func boxEq(t *testing.T, name string, got, want Decoration) {
	t.Helper()
	g, ok := got.(interface{ Equal(Decoration) bool })
	if !ok || !g.Equal(want) {
		t.Errorf("%s: got %v, want %v", name, got, want)
	}
}

func TestBoxDecorationLerpIdentical(t *testing.T) {
	if BoxDecorationLerp(nil, nil, 0) != nil {
		t.Error("null, null")
	}
	decoration := NewBoxDecoration()
	if BoxDecorationLerp(decoration, decoration, 0.5) != decoration {
		t.Error("identical")
	}
}

func TestBoxDecorationLerpCustomBoxBorder(t *testing.T) {
	d := BoxDecorationLerp(NewBoxDecoration(BoxDecorationOptions{Border: newTestBoxBorder(2.0)}),
		NewBoxDecoration(BoxDecorationOptions{Border: newTestBoxBorder(6.0)}), 0.25)
	if !ShapeBorderEqual(d.Border, newTestBoxBorder(3.0)) {
		t.Errorf("border %v", d.Border)
	}
}

func TestBoxDecorationScaleCustomBoxBorder(t *testing.T) {
	d := NewBoxDecoration(BoxDecorationOptions{Border: newTestBoxBorder(8.0)}).Scale(0.25)
	if !ShapeBorderEqual(d.Border, newTestBoxBorder(2.0)) {
		t.Errorf("border %v", d.Border)
	}
}

func TestBoxDecorationBorderRadiusDirectional(t *testing.T) {
	decoration := NewBoxDecoration(BoxDecorationOptions{Color: bdCol(0xFF000000),
		BorderRadius: BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(100.0)}})
	painter := decoration.CreateBoxPainter(nil)
	size := ui.NewSize(1000.0, 1000.0)
	rtl, ltr := bdTD(ui.TextDirectionRtl), bdTD(ui.TextDirectionLtr)
	want := ui.NewRRectFromRectAndCorners(ui.OffsetZero.And(size), ui.RRectCorners{TopRight: ui.NewRadiusCircular(100.0)})
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.OffsetZero, ImageConfiguration{Size: &size, TextDirection: rtl})
	}, ft.Paints().RRect(ft.PaintPatternRRectOptions{RRect: &want}))
	if !decoration.HitTest(size, ui.NewOffset(10.0, 10.0), rtl) || decoration.HitTest(size, ui.NewOffset(990.0, 10.0), rtl) {
		t.Error("hitTest rtl")
	}
	want = ui.NewRRectFromRectAndCorners(ui.OffsetZero.And(size), ui.RRectCorners{TopLeft: ui.NewRadiusCircular(100.0)})
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.OffsetZero, ImageConfiguration{Size: &size, TextDirection: ltr})
	}, ft.Paints().RRect(ft.PaintPatternRRectOptions{RRect: &want}))
	if decoration.HitTest(size, ui.NewOffset(10.0, 10.0), ltr) || !decoration.HitTest(size, ui.NewOffset(990.0, 10.0), ltr) {
		t.Error("hitTest ltr")
	}
}

func TestBoxDecorationLinearGradientDirectional(t *testing.T) {
	decoration := NewBoxDecoration(BoxDecorationOptions{Color: bdCol(0xFF000000),
		Gradient: NewLinearGradient(LinearGradientOptions{Begin: AlignmentDirectionalCenterStart, End: AlignmentDirectionalBottomEnd,
			Colors: []ui.Color{ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF)}})})
	painter := decoration.CreateBoxPainter(nil)
	size := ui.NewSize(1000.0, 1000.0)
	r := ui.OffsetZero.And(size)
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.OffsetZero, ImageConfiguration{Size: &size, TextDirection: bdTD(ui.TextDirectionRtl)})
	}, ft.Paints().Rect(ft.PaintPatternRectOptions{Rect: &r}))
}

func TestBoxDecorationGetClipPathBorderRadius(t *testing.T) {
	decoration := NewBoxDecoration(BoxDecorationOptions{BorderRadius: NewBorderRadiusAll(ui.NewRadiusCircular(10))})
	clipPath := decoration.GetClipPath(ui.NewRectFromLTWH(0.0, 0.0, 100.0, 20.0), ui.TextDirectionLtr)
	ft.Expect(t, clipPath, ft.IsPathThat(ft.IsPathThatOptions{
		Includes: []ui.Offset{ui.NewOffset(30.0, 10.0), ui.NewOffset(50.0, 10.0)},
		Excludes: []ui.Offset{ui.NewOffset(1.0, 1.0), ui.NewOffset(99.0, 19.0)}}))
}

func TestBoxDecorationGetClipPathCircle(t *testing.T) {
	decoration := NewBoxDecoration(BoxDecorationOptions{Shape: BoxShapeCircle})
	clipPath := decoration.GetClipPath(ui.NewRectFromLTWH(0.0, 0.0, 100.0, 20.0), ui.TextDirectionLtr)
	ft.Expect(t, clipPath, ft.IsPathThat(ft.IsPathThatOptions{
		Includes: []ui.Offset{ui.NewOffset(50.0, 0.0), ui.NewOffset(40.0, 10.0)},
		Excludes: []ui.Offset{ui.NewOffset(40.0, 0.0), ui.NewOffset(10.0, 10.0)}}))
}

func TestBoxDecorationHitTestCircle(t *testing.T) {
	decoration := NewBoxDecoration(BoxDecorationOptions{Shape: BoxShapeCircle})
	size := ui.NewSize(100.0, 20.0)
	for _, c := range []struct {
		x, y float64
		want bool
	}{{50, 0, true}, {40, 10, true}, {40, 0, false}, {10, 10, false}} {
		if decoration.HitTest(size, ui.NewOffset(c.x, c.y), nil) != c.want {
			t.Errorf("(%v, %v)", c.x, c.y)
		}
	}
}

func TestBoxDecorationsBlendModesNotEqual(t *testing.T) {
	color, m1, m2 := ui.NewColor(0x00000000), ui.BlendModeColor, ui.BlendModeDifference
	one := NewBoxDecoration(BoxDecorationOptions{Color: &color, BackgroundBlendMode: &m1})
	two := NewBoxDecoration(BoxDecorationOptions{Color: &color, BackgroundBlendMode: &m2})
	if one.Equal(two) {
		t.Error("equal")
	}
}

// decoration_test.dart

func TestDecorationLerpBoxDecorationColors(t *testing.T) {
	a := NewBoxDecoration(BoxDecorationOptions{Color: bdCol(0xFFFFFFFF)})
	b := NewBoxDecoration(BoxDecorationOptions{Color: bdCol(0x00000000)})
	c := DecorationLerp(a, b, 0.0).(*BoxDecoration)
	if *c.Color != *a.Color {
		t.Error("0.0")
	}
	c = DecorationLerp(a, b, 0.25).(*BoxDecoration)
	if *c.Color != *ui.ColorLerp(bdCol(0xFFFFFFFF), bdCol(0x00000000), 0.25) {
		t.Error("0.25")
	}
	c = DecorationLerp(a, b, 1.0).(*BoxDecoration)
	if *c.Color != *b.Color {
		t.Error("1.0")
	}
}

func TestDecorationLerpIdenticalBoxDecoration(t *testing.T) {
	if DecorationLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	var decoration Decoration = NewBoxDecoration()
	if DecorationLerp(decoration, decoration, 0.5) != decoration {
		t.Error("identical")
	}
}

func TestDecorationEqualityBoxDecoration(t *testing.T) {
	a := NewBoxDecoration(BoxDecorationOptions{Color: bdCol(0xFFFFFFFF), BoxShadow: []BoxShadow{NewBoxShadow()}})
	b := NewBoxDecoration(BoxDecorationOptions{Color: bdCol(0xFFFFFFFF), BoxShadow: []BoxShadow{NewBoxShadow()}})
	boxEq(t, "equality", a, b)
}

func TestBoxDecorationImageListenerSync(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, ft.CreateTestImageOptions{Width: 100, Height: 100})
	boxDecoration := NewBoxDecoration(BoxDecorationOptions{
		Image: NewDecorationImage(DecorationImageOptions{Image: newSynchronousTestImageProvider(image)})})
	onChangedCalled := false
	boxPainter := boxDecoration.CreateBoxPainter(func() { onChangedCalled = true })
	boxPainter.Paint(ft.NewTestRecordingCanvas(), ui.OffsetZero, sizeConfig(0, 0))
	if onChangedCalled {
		t.Error("onChanged called during paint")
	}
}

func TestBoxDecorationImageListenerAsync(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, ft.CreateTestImageOptions{Width: 10, Height: 10})
	boxDecoration := NewBoxDecoration(BoxDecorationOptions{
		Image: NewDecorationImage(DecorationImageOptions{Image: newAsyncTestImageProvider(image)})})
	onChangedCalled := false
	boxPainter := boxDecoration.CreateBoxPainter(func() { onChangedCalled = true })
	boxPainter.Paint(ft.NewTestRecordingCanvas(), ui.OffsetZero, sizeConfig(0, 0))
	if onChangedCalled {
		t.Error("called synchronously")
	}
	tester.Loop().FlushMicrotasks()
	if !onChangedCalled {
		t.Error("not called after the microtasks")
	}
}

func TestBoxDecorationImageListenerClone(t *testing.T) {
	tester := newImageTester(t)
	image1 := createTestImage(tester, ft.CreateTestImageOptions{Width: 10, Height: 10, NoCache: true})
	image2 := createTestImage(tester, ft.CreateTestImageOptions{Width: 10, Height: 10, NoCache: true})
	completer := NewTestImageStreamCompleter()
	boxDecoration := NewBoxDecoration(BoxDecorationOptions{
		Image: NewDecorationImage(DecorationImageOptions{Image: newMultiFrameImageProvider(completer)})})
	onChangedCalled := false
	boxPainter := boxDecoration.CreateBoxPainter(func() { onChangedCalled = true })
	boxPainter.Paint(ft.NewTestRecordingCanvas(), ui.OffsetZero, sizeConfig(0, 0))
	if onChangedCalled {
		t.Error("called synchronously")
	}
	completer.TestSetImage(image1.Clone())
	tester.Loop().FlushMicrotasks()
	if !onChangedCalled {
		t.Error("first image")
	}
	onChangedCalled = false
	completer.TestSetImage(image1.Clone())
	tester.Loop().FlushMicrotasks()
	if onChangedCalled {
		t.Error("clone of the same image")
	}
	completer.TestSetImage(image2.Clone())
	tester.Loop().FlushMicrotasks()
	if !onChangedCalled {
		t.Error("new image")
	}
}

func TestBoxDecorationBackgroundImageClip(t *testing.T) {
	tester := newImageTester(t)
	image := createTestImage(tester, ft.CreateTestImageOptions{Width: 100, Height: 100})
	testDecoration := func(shape BoxShape, borderRadius BorderRadiusGeometry, expectClip bool) {
		t.Helper()
		imageProvider := newDelayedImageProvider(image)
		boxDecoration := NewBoxDecoration(BoxDecorationOptions{Shape: shape, BorderRadius: borderRadius,
			Image: NewDecorationImage(DecorationImageOptions{Image: imageProvider})})
		canvas := ft.NewTestRecordingCanvas()
		config := sizeConfig(100.0, 100.0)
		onChangedCalled := false
		boxPainter := boxDecoration.CreateBoxPainter(func() { onChangedCalled = true })
		// The painter resolves the background image and listens to its stream.
		boxPainter.Paint(canvas, ui.OffsetZero, config)
		imageProvider.Complete()
		tester.Loop().FlushMicrotasks()
		if !onChangedCalled {
			t.Fatal("onChanged not called")
		}
		boxPainter.Paint(canvas, ui.OffsetZero, config)
		var commands []string
		for _, r := range canvas.Invocations {
			if n := r.Invocation.MemberName; n == "clipPath" || n == "drawImageRect" {
				commands = append(commands, n)
			}
		}
		want := []string{"drawImageRect"}
		if expectClip {
			want = []string{"clipPath", "drawImageRect"}
		}
		if len(commands) != len(want) {
			t.Fatalf("commands %v, want %v", commands, want)
		}
		for i := range want {
			if commands[i] != want[i] {
				t.Errorf("commands %v, want %v", commands, want)
			}
		}
	}
	testDecoration(BoxShapeCircle, nil, true)
	testDecoration(BoxShapeRectangle, NewBorderRadiusAll(ui.NewRadiusCircular(16.0)), true)
	testDecoration(BoxShapeRectangle, nil, false)
}

func TestBoxDecorationLerpShapes(t *testing.T) {
	// We don't lerp the shape, we just switch from one to the other at t=0.5.
	rect := NewBoxDecoration()
	circle := NewBoxDecoration(BoxDecorationOptions{Shape: BoxShapeCircle})
	for _, c := range []struct {
		t    float64
		want *BoxDecoration
	}{{-1.0, rect}, {0.0, rect}, {0.25, rect}, {0.75, circle}, {1.0, circle}, {2.0, circle}} {
		boxEq(t, dart.DoubleToString(c.t), BoxDecorationLerp(NewBoxDecoration(),
			NewBoxDecoration(BoxDecorationOptions{Shape: BoxShapeCircle}), c.t), c.want)
	}
}

func TestBoxDecorationLerpGradients(t *testing.T) {
	gradient := NewLinearGradient(LinearGradientOptions{Colors: []ui.Color{ui.NewColor(0x00000000), ui.NewColor(0xFFFFFFFF)}})
	withGradient := func(second int) *BoxDecoration {
		return NewBoxDecoration(BoxDecorationOptions{Gradient: NewLinearGradient(LinearGradientOptions{
			Colors: []ui.Color{ui.NewColor(0x00000000), ui.NewColor(second)}})})
	}
	for _, c := range []struct {
		t    float64
		want *BoxDecoration
	}{
		{-1.0, withGradient(0x00FFFFFF)},
		{0.0, NewBoxDecoration()},
		{0.2, withGradient(0x33FFFFFF)},
		{1.0 / 3.0, withGradient(0x55FFFFFF)},
		{1.0, NewBoxDecoration(BoxDecorationOptions{Gradient: gradient})},
		{2.0, NewBoxDecoration(BoxDecorationOptions{Gradient: gradient})},
	} {
		got := BoxDecorationLerp(NewBoxDecoration(), NewBoxDecoration(BoxDecorationOptions{Gradient: gradient}), c.t)
		boxEq(t, dart.DoubleToString(c.t), got, c.want)
	}
}

func TestDecorationLerpUnrelatedBoxDecoration(t *testing.T) {
	for _, c := range []struct {
		t    float64
		logo bool
	}{{0.0, true}, {0.25, true}, {0.75, false}, {1.0, false}} {
		r := DecorationLerp(NewFlutterLogoDecoration(), NewBoxDecoration(), c.t)
		_, isLogo := r.(*FlutterLogoDecoration)
		_, isBox := r.(*BoxDecoration)
		if isLogo != c.logo || isBox == c.logo {
			t.Errorf("t=%v: %T", c.t, r)
		}
	}
}

// border_rtl_test.dart: 'BoxDecoration.border takes a BorderDirectional'.
func TestBoxDecorationBorderTakesBorderDirectional(t *testing.T) {
	deco := func(w float64) *BoxDecoration {
		side := NewBorderSide(BorderSideOptions{Width: bdF(w)})
		return NewBoxDecoration(BoxDecorationOptions{Border: NewBorderDirectional(BorderDirectionalOptions{Start: &side})})
	}
	decoration2, decoration4, decoration6 := deco(2.0), deco(4.0), deco(6.0)
	painter := decoration2.CreateBoxPainter(nil)
	size := ui.NewSize(20.0, 20.0)
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.NewOffset(30.0, 0.0), ImageConfiguration{Size: &size, TextDirection: bdTD(ui.TextDirectionRtl)})
	}, ft.Paints().Path(ft.PaintPatternPathOptions{Includes: []ui.Offset{ui.NewOffset(49.0, 10.0)},
		Excludes: []ui.Offset{ui.NewOffset(31.0, 10.0)}}))
	ft.Expect(t, func(c ui.Canvas) {
		painter.Paint(c, ui.NewOffset(30.0, 0.0), ImageConfiguration{Size: &size, TextDirection: bdTD(ui.TextDirectionLtr)})
	}, ft.Paints().Path(ft.PaintPatternPathOptions{Includes: []ui.Offset{ui.NewOffset(31.0, 10.0)},
		Excludes: []ui.Offset{ui.NewOffset(49.0, 10.0)}}))
	if decoration2.Padding() != EdgeInsetsGeometry(NewEdgeInsetsDirectionalFromSTEB(2.0, 0.0, 0.0, 0.0)) {
		t.Errorf("padding %v", decoration2.Padding())
	}
	boxEq(t, "scale", decoration2.Scale(2.0), decoration4)
	boxEq(t, "lerp", BoxDecorationLerp(decoration2, decoration6, 0.5), decoration4)
}

// Go-only: the constructor's and DebugAssertIsValid's assertions, the
// createBoxPainter assertion, copyWith, padding and isComplex.
func TestBoxDecorationAssertionsAndMembers(t *testing.T) {
	m := ui.BlendModeMultiply
	bdExpectAssertion(t, func() { NewBoxDecoration(BoxDecorationOptions{BackgroundBlendMode: &m}) })
	bdExpectAssertion(t, func() {
		NewBoxDecoration(BoxDecorationOptions{Shape: BoxShapeCircle, BorderRadius: BorderRadiusZero}).DebugAssertIsValid()
	})
	if !NewBoxDecoration().DebugAssertIsValid() {
		t.Error("valid")
	}
	image := NewDecorationImage(DecorationImageOptions{Image: newSynchronousTestImageProvider(nil)})
	bdExpectAssertion(t, func() { NewBoxDecoration(BoxDecorationOptions{Image: image}).CreateBoxPainter(nil) })
	d := NewBoxDecoration(BoxDecorationOptions{Color: bdCol(0xFF00FF00), Border: NewBorderAll(BorderSideOptions{Width: bdF(3)})})
	c := d.CopyWith(BoxDecorationCopyWithOptions{Shape: func() *BoxShape { s := BoxShapeCircle; return &s }()})
	if c.Shape != BoxShapeCircle || *c.Color != *d.Color || !ShapeBorderEqual(c.Border, d.Border) {
		t.Errorf("copyWith %v", c)
	}
	if d.Padding() != EdgeInsetsGeometry(NewEdgeInsetsAll(3)) || NewBoxDecoration().Padding() != EdgeInsetsGeometry(EdgeInsetsZero) {
		t.Error("padding")
	}
	if d.IsComplex() || !NewBoxDecoration(BoxDecorationOptions{BoxShadow: []BoxShadow{}}).IsComplex() {
		t.Error("isComplex")
	}
}
