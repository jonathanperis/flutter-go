package painting_test

// The "Flutter" label of FlutterLogoDecoration's horizontal and stacked
// styles (flutter_logo.dart, 3.50.1). Flutter tests it only through the
// "Flutter Logo golden test" (a Container golden, skipped in
// flutter_logo_test.go; the label's pixels are checked by flutter-go's
// Impeller difftest). These tests record the painter's canvas calls
// (flutter_test's TestRecordingCanvas, hence the external package) and
// check the label's clip, transform, paragraph and fade against Dart's
// formulas, with flutter_tester's test fonts.

import (
	"math"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	. "github.com/jonathanperis/flutter-go/packages/flutter/lib/src/painting"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	ft "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
)

// logoLabelBox is the painter's _textBoundingRect: the box of "Flutter"
// laid out at font size 100 * 350 / 247 (the label's TextPainter).
func logoLabelBox(t *testing.T) ui.Rect {
	t.Helper()
	text, family := "Flutter", "Roboto"
	size, weight, baseline := 100.0*350.0/247.0, ui.FontWeightW300, ui.TextBaselineAlphabetic
	ltr := ui.TextDirectionLtr
	p := NewTextPainter(TextPainterOptions{
		Text: NewTextSpan(TextSpanOptions{Text: &text, Style: NewTextStyle(TextStyleOptions{
			FontFamily: &family, FontSize: &size, FontWeight: &weight, TextBaseline: &baseline})}),
		TextDirection: &ltr,
	})
	defer p.Dispose()
	p.Layout()
	boxes := p.GetBoxesForSelection(services.NewTextSelection(0, 7))
	if len(boxes) != 1 {
		t.Fatalf("%d boxes", len(boxes))
	}
	return boxes[0].ToRect()
}

func logoLabelPaint(d *FlutterLogoDecoration, size ui.Size) *ft.TestRecordingCanvas {
	c := ft.NewTestRecordingCanvas()
	bp := d.CreateBoxPainter(func() {})
	bp.Paint(c, ui.OffsetZero, ImageConfiguration{Size: &size})
	bp.Dispose()
	return c
}

// logoLabelCalls is the recorded calls before the mark (whose first call is
// its save + translate + scale to 202x202).
func logoLabelCalls(c *ft.TestRecordingCanvas) []ft.RecordedInvocation {
	for i, inv := range c.Invocations {
		if inv.Invocation.MemberName == "drawParagraph" {
			for j := i + 1; j < len(c.Invocations); j++ {
				if c.Invocations[j].Invocation.MemberName == "restore" {
					return c.Invocations[:j+1]
				}
			}
		}
	}
	return nil
}

func logoLabelNames(calls []ft.RecordedInvocation) string {
	var names []string
	for _, c := range calls {
		names = append(names, c.Invocation.MemberName)
	}
	return strings.Join(names, " ")
}

func logoLabelNear(t *testing.T, what string, got any, want float64) {
	t.Helper()
	if g, ok := got.(float64); !ok || math.Abs(g-want) > 1e-9 {
		t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

func TestFlutterLogoHorizontalLabel(t *testing.T) {
	ui.EngineUseTestFonts()
	box := logoLabelBox(t)
	size := ui.NewSize(820, 232)

	c := logoLabelPaint(NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleHorizontal}), size)
	calls := logoLabelCalls(c)
	if got := logoLabelNames(calls); got != "save translate scale drawParagraph restore" {
		t.Fatalf("calls: %s", got)
	}
	fontSize := 2.0 / 3.0 * 232.0 * (1 - (10.4*2.0)/202.0)
	scale := fontSize / 100.0
	logoLabelNear(t, "dx", calls[1].Invocation.PositionalArguments[0], (256.4/820.0)*820.0-(32.0/350.0)*fontSize)
	logoLabelNear(t, "dy", calls[1].Invocation.PositionalArguments[1], (232.0-box.Height()*scale)/2.0)
	logoLabelNear(t, "scale x", calls[2].Invocation.PositionalArguments[0], scale)
	logoLabelNear(t, "scale y", calls[2].Invocation.PositionalArguments[1], scale)
	if off := calls[3].Invocation.PositionalArguments[1].(ui.Offset); off != ui.OffsetZero {
		t.Errorf("paragraph offset %v", off)
	}

	// Mid-transition from the mark only, the label is clipped to a wedge.
	c = logoLabelPaint(FlutterLogoDecorationLerp(NewFlutterLogoDecoration(),
		NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleHorizontal}), 0.75), size)
	if got := logoLabelNames(logoLabelCalls(c)); got != "save clipPath translate scale drawParagraph restore" {
		t.Errorf("transition calls: %s", got)
	}
}

func TestFlutterLogoStackedLabel(t *testing.T) {
	ui.EngineUseTestFonts()
	box := logoLabelBox(t)
	size := ui.NewSize(252, 306)

	c := logoLabelPaint(NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleStacked}), size)
	calls := logoLabelCalls(c)
	if got := logoLabelNames(calls); got != "save translate scale drawParagraph restore" {
		t.Fatalf("calls: %s", got)
	}
	logoHeight := 306.0 * 191.0 / 306.0
	fontSize := 0.35 * logoHeight * (1 - (10.4*2.0)/202.0)
	scale := fontSize / 100.0
	logoLabelNear(t, "dx", calls[1].Invocation.PositionalArguments[0], (252.0-logoHeight)/2.0+logoHeight/2.0-box.Width()*scale/2.0)
	logoLabelNear(t, "dy", calls[1].Invocation.PositionalArguments[1], logoHeight)
	logoLabelNear(t, "scale", calls[2].Invocation.PositionalArguments[0], scale)

	// Mid-transition, the label is faded in by a modulating gradient in a
	// layer bounded by the text.
	c = logoLabelPaint(FlutterLogoDecorationLerp(NewFlutterLogoDecoration(),
		NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleStacked}), 0.75), size)
	calls = logoLabelCalls(c)
	if got := logoLabelNames(calls); got != "saveLayer translate scale drawParagraph drawRect restore" {
		t.Fatalf("transition calls: %s", got)
	}
	if b := calls[0].Invocation.PositionalArguments[0].(ui.Rect); b != box {
		t.Errorf("layer bounds %v, want %v", b, box)
	}
	if r := calls[4].Invocation.PositionalArguments[0].(ui.Rect); r != box.Inflate(box.Width()*0.5) {
		t.Errorf("fade rect %v", r)
	}
	if p := calls[4].Invocation.PositionalArguments[1].(*ui.Paint); p.BlendMode() != ui.BlendModeModulate || p.Shader() == nil {
		t.Errorf("fade paint %v", p)
	}
}

func TestFlutterLogoMarkOnlyHasNoLabel(t *testing.T) {
	ui.EngineUseTestFonts()
	c := logoLabelPaint(NewFlutterLogoDecoration(), ui.NewSize(100, 100))
	for _, inv := range c.Invocations {
		if inv.Invocation.MemberName == "drawParagraph" {
			t.Fatal("the mark-only logo draws no label")
		}
	}
}
