package painting_test

// Port of packages/flutter/test/painting/paint_image_test.dart (Flutter
// 3.50.1): all 9 tests. TestCanvas (noSuchMethod) is flutter_test's
// TestRecordingCanvas. flutter_tester's views have a device pixel ratio of
// 3.0; flutter-go's view has 1.0 unless the embedder reports one, so the
// tests set it to 3.0 (the testWidgets ones read it back as Dart reads
// tester.view.devicePixelRatio).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

type paintImageImages struct{ image300x300, image300x200, image208x142 *ui.Image }

// newPaintImageTester is the file's setUpAll/setUp: a binding whose view
// has flutter_tester's device pixel ratio, the three images, and the last
// frame's image sizes flushed.
func newPaintImageTester(t *testing.T) (*flutter_test.Tester, paintImageImages) {
	tester := newImageTester(t)
	ui.NewPlatformConfiguration(nil).UpdateViewGestureMetrics(0, 3.0, ui.GestureSettings{})
	t.Cleanup(func() { ui.NewPlatformConfiguration(nil).UpdateViewGestureMetrics(0, 1.0, ui.GestureSettings{}) })
	images := paintImageImages{
		createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 300, Height: 300, NoCache: true}),
		createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 300, Height: 200, NoCache: true}),
		createTestImage(tester, flutter_test.CreateTestImageOptions{Width: 208, Height: 142, NoCache: true}),
	}
	DebugFlushLastFrameImageSizeInfo()
	return tester, images
}

func firstCall(t *testing.T, c *flutter_test.TestRecordingCanvas, name string) []any {
	t.Helper()
	calls := invocationsNamed(c, name)
	if len(calls) == 0 {
		t.Fatalf("no %s call", name)
	}
	return calls[0].PositionalArguments
}

func TestPaintImageCoverAndAlign(t *testing.T) {
	_, img := newPaintImageTester(t)
	canvas := flutter_test.NewTestRecordingCanvas()
	fit := BoxFitCover
	PaintImage(canvas, ui.NewRectFromLTWH(50.0, 75.0, 200.0, 100.0), img.image300x300,
		PaintImageOptions{Fit: &fit, Alignment: AlignmentCenterLeft})
	args := firstCall(t, canvas, "drawImageRect")
	if args[0] != img.image300x300 || args[1] != ui.NewRectFromLTWH(0.0, 75.0, 300.0, 150.0) ||
		args[2] != ui.NewRectFromLTWH(50.0, 75.0, 200.0, 100.0) {
		t.Errorf("args %v", args)
	}
}

func captureErrors(t *testing.T) *[]string {
	messages := &[]string{}
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		*messages = append(*messages, details.ExceptionAsString())
	}
	t.Cleanup(func() { foundation.FlutterErrorOnError = old })
	return messages
}

func TestPaintImageDebugInvertOversizedImages(t *testing.T) {
	_, img := newPaintImageTester(t)
	DebugInvertOversizedImages = true
	defer func() { DebugInvertOversizedImages = false }()
	if PaintingBindingInstance().PlatformDispatcher().ImplicitView().DevicePixelRatio() <= 1.0 {
		t.Fatal("device pixel ratio")
	}
	messages := captureErrors(t)
	canvas := flutter_test.NewTestRecordingCanvas()
	rect := ui.NewRectFromLTWH(50.0, 50.0, 100.0, 50.0)
	label := "TestImage"
	fit := BoxFitFill
	PaintImage(canvas, rect, img.image300x300, PaintImageOptions{DebugImageLabel: &label, Fit: &fit})
	var commands []flutter_test.Invocation
	for _, r := range canvas.Invocations {
		if len(commands) == 0 && r.Invocation.MemberName != "saveLayer" {
			continue
		}
		if len(commands) < 4 {
			commands = append(commands, r.Invocation)
		}
	}
	if len(commands) != 4 {
		t.Fatalf("commands %v", commands)
	}
	if commands[0].PositionalArguments[0] != rect {
		t.Errorf("saveLayer bounds %v", commands[0].PositionalArguments[0])
	}
	want := ui.NewColorFilterMatrix([]float64{-1, 0, 0, 0, 255, 0, -1, 0, 0, 255, 0, 0, -1, 0, 255, 0, 0, 0, 1, 0})
	if !commands[0].PositionalArguments[1].(*ui.Paint).ColorFilter().Equal(want) {
		t.Error("color filter")
	}
	for i, c := range []struct {
		name string
		a, b float64
	}{{"translate", 0.0, 75.0}, {"scale", 1.0, -1.0}, {"translate", 0.0, -75.0}} {
		cmd := commands[i+1]
		if cmd.MemberName != c.name || cmd.PositionalArguments[0] != c.a || cmd.PositionalArguments[1] != c.b {
			t.Errorf("command %d: %v", i+1, cmd)
		}
	}
	if len(*messages) != 1 || (*messages)[0] != "Image TestImage has a display size of 300×150 but a decode size of 300×300, "+
		"which uses an additional 234KB (assuming a device pixel ratio of 3.0).\n\n"+
		"Consider resizing the asset ahead of time, supplying a cacheWidth parameter of 300, a cacheHeight parameter of 150, "+
		"or using a ResizeImage." {
		t.Errorf("messages %q", *messages)
	}
}

func TestPaintImageDebugInvertOversizedImagesUnderAllowance(t *testing.T) {
	_, img := newPaintImageTester(t)
	DebugInvertOversizedImages = true
	defer func() { DebugInvertOversizedImages = false }()
	messages := captureErrors(t)
	label := "TestImage"
	fit := BoxFitFill
	PaintImage(flutter_test.NewTestRecordingCanvas(), ui.NewRectFromLTWH(50.0, 50.0, 290.0, 290.0), img.image300x300,
		PaintImageOptions{DebugImageLabel: &label, Fit: &fit})
	if len(*messages) != 0 {
		t.Errorf("messages %q", *messages)
	}
}

func TestPaintImageCenterSliceScale(t *testing.T) {
	_, img := newPaintImageTester(t)
	canvas := flutter_test.NewTestRecordingCanvas()
	scale := 2.0
	slice := ui.NewRectFromLTRB(50, 40, 250, 260)
	PaintImage(canvas, ui.NewRectFromLTRB(10, 20, 430, 420), img.image300x300, PaintImageOptions{Scale: &scale, CenterSlice: &slice})
	args := firstCall(t, canvas, "drawImageNine")
	if args[0] != img.image300x300 || args[1] != ui.NewRectFromLTRB(100.0, 80.0, 500.0, 520.0) ||
		args[2] != ui.NewRectFromLTRB(20.0, 40.0, 860.0, 840.0) {
		t.Errorf("args %v", args)
	}
}

func TestPaintImageCenterSliceRounding(t *testing.T) {
	_, img := newPaintImageTester(t)
	scale := 3.0
	for _, c := range []struct{ rect, slice ui.Rect }{
		{ui.NewRectFromLTWH(0.0, 0.0, 200.0, 100.0), ui.NewRectFromLTRB(15.0, 15.0, 17.0, 17.0)},
		{ui.NewRectFromLTWH(0.0, 0.0, 100.0, 46.0), ui.NewRectFromLTRB(15.5, 15.5, 16.5, 16.5)},
	} {
		canvas := flutter_test.NewTestRecordingCanvas()
		PaintImage(canvas, c.rect, img.image208x142, PaintImageOptions{Scale: &scale, CenterSlice: &c.slice})
		if args := firstCall(t, canvas, "drawImageNine"); args[0] != img.image208x142 {
			t.Errorf("image %v", args[0])
		}
	}
}

func TestPaintImageReportsImagePainting(t *testing.T) {
	tester, img := newPaintImageTester(t)
	var info ImageSizeInfo
	count := 0
	DebugOnPaintImage = func(i ImageSizeInfo) { count++; info = i }
	defer func() { DebugOnPaintImage = nil }()
	dpr := PaintingBindingInstance().PlatformDispatcher().ImplicitView().DevicePixelRatio()
	label := "test.png"
	canvas := flutter_test.NewTestRecordingCanvas()
	PaintImage(canvas, ui.NewRectFromLTWH(50.0, 75.0, 200.0, 100.0), img.image300x300, PaintImageOptions{DebugImageLabel: &label})
	if count != 1 || info.Source != "test.png" || info.ImageSize != ui.NewSize(300, 300) ||
		info.DisplaySize != ui.NewSize(200, 100).Mul(dpr) {
		t.Errorf("count %d, info %v", count, info)
	}
	// Make sure that we don't report an identical image size info if we
	// redraw in the next frame.
	tester.Binding().ScheduleForcedFrame()
	tester.Pump()
	PaintImage(canvas, ui.NewRectFromLTWH(50.0, 75.0, 200.0, 100.0), img.image300x300, PaintImageOptions{DebugImageLabel: &label})
	if count != 1 {
		t.Errorf("count %d", count)
	}
}

func TestPaintImageReportsChangePerFrame(t *testing.T) {
	tester, img := newPaintImageTester(t)
	var info ImageSizeInfo
	count := 0
	DebugOnPaintImage = func(i ImageSizeInfo) { count++; info = i }
	defer func() { DebugOnPaintImage = nil }()
	dpr := PaintingBindingInstance().PlatformDispatcher().ImplicitView().DevicePixelRatio()
	label := "test.png"
	canvas := flutter_test.NewTestRecordingCanvas()
	PaintImage(canvas, ui.NewRectFromLTWH(50.0, 75.0, 200.0, 100.0), img.image300x300, PaintImageOptions{DebugImageLabel: &label})
	if count != 1 || info.DisplaySize != ui.NewSize(200, 100).Mul(dpr) {
		t.Errorf("count %d, info %v", count, info)
	}
	tester.Binding().ScheduleForcedFrame()
	tester.Pump()
	PaintImage(canvas, ui.NewRectFromLTWH(50.0, 75.0, 200.0, 150.0), img.image300x300, PaintImageOptions{DebugImageLabel: &label})
	if count != 2 || info.Source != "test.png" || info.ImageSize != ui.NewSize(300, 300) ||
		info.DisplaySize != ui.NewSize(200, 150).Mul(dpr) {
		t.Errorf("count %d, info %v", count, info)
	}
}

func TestPaintImageReportsNoDebugLabel(t *testing.T) {
	_, img := newPaintImageTester(t)
	var info ImageSizeInfo
	count := 0
	DebugOnPaintImage = func(i ImageSizeInfo) { count++; info = i }
	defer func() { DebugOnPaintImage = nil }()
	dpr := PaintingBindingInstance().PlatformDispatcher().ImplicitView().DevicePixelRatio()
	PaintImage(flutter_test.NewTestRecordingCanvas(), ui.NewRectFromLTWH(50.0, 75.0, 200.0, 100.0), img.image300x200)
	if count != 1 || info.Source != "<Unknown Image(300×200)>" || info.ImageSize != ui.NewSize(300, 200) ||
		info.DisplaySize != ui.NewSize(200, 100).Mul(dpr) {
		t.Errorf("count %d, info %v", count, info)
	}
}
