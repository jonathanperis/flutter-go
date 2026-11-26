package painting_test

// clip.dart (ClipContext) has no test file of its own in Flutter
// (rendering's PaintingContext tests cover it); these check the calls it
// makes for each Clip behavior on a TestRecordingCanvas. An external test
// package, so that it can import flutter_test.

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

type testClipContext struct {
	painting.ClipContextBase
	canvas *flutter_test.TestRecordingCanvas
}

func (c *testClipContext) Canvas() ui.Canvas { return c.canvas }

func newTestClipContext() *testClipContext {
	c := &testClipContext{canvas: flutter_test.NewTestRecordingCanvas()}
	c.SetSelf(c)
	return c
}

func calls(c *flutter_test.TestRecordingCanvas) string {
	var s []string
	for _, inv := range c.Invocations {
		s = append(s, inv.String())
	}
	return strings.Join(s, "; ")
}

func TestClipContext(t *testing.T) {
	r := ui.NewRectFromLTRB(1, 2, 3, 4)
	bounds := ui.NewRectFromLTRB(0, 0, 10, 10)
	rr := ui.NewRRectFromRectXY(r, 1, 1)
	rse := ui.NewRSuperellipseFromRectAndRadius(r, ui.NewRadiusCircular(1))
	path := ui.NewPath()
	paint := ui.NewPaint().String()
	for _, tc := range []struct {
		clip  func(c *testClipContext, b ui.Clip, painter func())
		which string
	}{
		{func(c *testClipContext, b ui.Clip, p func()) { c.ClipRectAndPaint(r, b, bounds, p) }, "clipRect(Rect.fromLTRB(1.0, 2.0, 3.0, 4.0)"},
		{func(c *testClipContext, b ui.Clip, p func()) { c.ClipRRectAndPaint(rr, b, bounds, p) }, "clipRRect(" + rr.String()},
		{func(c *testClipContext, b ui.Clip, p func()) { c.ClipRSuperellipseAndPaint(rse, b, bounds, p) }, "clipRSuperellipse(" + rse.String()},
		{func(c *testClipContext, b ui.Clip, p func()) { c.ClipPathAndPaint(path, b, bounds, p) }, "clipPath(Path"},
	} {
		for _, want := range []struct {
			clip ui.Clip
			want string
		}{
			{ui.ClipNone, "save(); drawPaint(" + paint + "); restore()"},
			{ui.ClipHardEdge, "save(); " + tc.which + ", doAntiAlias: false); drawPaint(" + paint + "); restore()"},
			{ui.ClipAntiAlias, "save(); " + tc.which + ", doAntiAlias: true); drawPaint(" + paint + "); restore()"},
			{ui.ClipAntiAliasWithSaveLayer, "save(); " + tc.which + ", doAntiAlias: true); saveLayer(Rect.fromLTRB(0.0, 0.0, 10.0, 10.0), " +
				paint + "); drawPaint(" + paint + "); restore(); restore()"},
		} {
			c := newTestClipContext()
			tc.clip(c, want.clip, func() { c.canvas.DrawPaint(ui.NewPaint()) })
			if got := calls(c.canvas); got != want.want {
				t.Errorf("%v:\n got  %s\n want %s", want.clip, got, want.want)
			}
		}
	}
}

func TestPaintZigZagCalls(t *testing.T) {
	c := flutter_test.NewTestRecordingCanvas()
	painting.PaintZigZag(c, ui.NewPaint(), ui.NewOffset(10, 20), ui.NewOffset(10, 120), 2, 5)
	// Rotated by atan2(100, 0) = pi/2; length 100, spacing 25.
	flutter_test.Expect(t, func(cv ui.Canvas) {
		painting.PaintZigZag(cv, ui.NewPaint(), ui.NewOffset(10, 20), ui.NewOffset(10, 120), 2, 5)
	},
		flutter_test.Paints().Save().
			Translate(flutter_test.PaintPatternTranslateOptions{X: f(10), Y: f(20)}).
			Rotate(flutter_test.PaintPatternRotateOptions{Angle: f(1.5707963267948966)}).
			Path(flutter_test.PaintPatternPathOptions{}).Restore())
	path := c.Invocations[3].Invocation.PositionalArguments[0].(*ui.Path)
	if b := path.GetBounds(); b != ui.NewRectFromLTRB(0, -5, 100, 5) {
		t.Fatal(b)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("zigs 0 did not assert")
			}
		}()
		painting.PaintZigZag(c, ui.NewPaint(), ui.OffsetZero, ui.NewOffset(1, 1), 0, 1)
	}()
}

func f(v float64) *float64 { return &v }
