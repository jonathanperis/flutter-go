package painting

// Port of packages/flutter/test/painting/flutter_logo_test.dart (Flutter
// 3.50.1). Changed: "lerpFrom and lerpTo" lerps against a stand-in
// decoration (otherDecoration) instead of BoxDecoration (box_decoration.dart
// is ported separately); what it checks is Decoration.lerp's handling of
// unrelated decorations, which is the same. Skipped: "Flutter Logo golden
// test" (a Container golden; the painting_decorations_pixels difftest draws
// the mark against Flutter's Impeller output).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// otherDecoration is a Decoration that lerps with nothing.
type otherDecoration struct{ DecorationBase }

func newOtherDecoration() *otherDecoration {
	d := &otherDecoration{}
	d.SetSelf(d)
	return d
}

func (d *otherDecoration) CreateBoxPainter(onChanged func()) BoxPainter { return nil }

func logoStartEnd() (*FlutterLogoDecoration, *FlutterLogoDecoration) {
	c1, c2 := ui.NewColor(0xFFD4F144), ui.NewColor(0xFF81D4FA)
	start := NewFlutterLogoDecoration(FlutterLogoDecorationOptions{TextColor: &c1, Style: FlutterLogoStyleStacked,
		Margin: EdgeInsets{10, 10, 10, 10}})
	end := NewFlutterLogoDecoration(FlutterLogoDecorationOptions{TextColor: &c2, Style: FlutterLogoStyleStacked,
		Margin: EdgeInsets{10, 10, 10, 10}})
	return start, end
}

func TestFlutterLogoLerpNullNull(t *testing.T) {
	if FlutterLogoDecorationLerp(nil, nil, 0.5) != nil || FlutterLogoDecorationLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	logo := NewFlutterLogoDecoration()
	if !FlutterLogoDecorationLerp(logo, logo, 0.5).Equal(logo) {
		t.Error("identical")
	}
}

func TestFlutterLogoLerpMargin(t *testing.T) {
	start, end := logoStartEnd()
	logo := FlutterLogoDecorationLerp(start, nil, 0.4)
	if logo.TextColor != start.TextColor || logo.Style != start.Style || logo.Margin != start.Margin.Mul(0.4) {
		t.Errorf("to null: %v", logo)
	}
	logo = FlutterLogoDecorationLerp(nil, end, 0.6)
	if logo.TextColor != end.TextColor || logo.Style != end.Style || logo.Margin != end.Margin.Mul(0.6) {
		t.Errorf("from null: %v", logo)
	}
	logo = FlutterLogoDecorationLerp(start, end, 0.5)
	if logo.TextColor != *ui.ColorLerp(&start.TextColor, &end.TextColor, 0.5) ||
		logo.Margin != *EdgeInsetsLerp(&start.Margin, &end.Margin, 0.5) {
		t.Errorf("between: %v", logo)
	}
}

func TestFlutterLogoLerpFromAndTo(t *testing.T) {
	start, end := logoStartEnd()
	other := newOtherDecoration()
	if DecorationLerp(start, other, 0.0) != Decoration(start) || DecorationLerp(start, other, 1.0) != Decoration(other) ||
		DecorationLerp(other, end, 0.0) != Decoration(other) || DecorationLerp(other, end, 1.0) != Decoration(end) {
		t.Error("lerp with unrelated decoration")
	}
}

func TestFlutterLogoLerpStyleSwitch(t *testing.T) {
	start, _ := logoStartEnd()
	h := NewFlutterLogoDecoration(FlutterLogoDecorationOptions{Style: FlutterLogoStyleHorizontal})
	if FlutterLogoDecorationLerp(start, h, 0.4).Style != start.Style || FlutterLogoDecorationLerp(start, h, 0.5).Style != h.Style {
		t.Error("style switch")
	}
}

func TestFlutterLogoToString(t *testing.T) {
	start, end := logoStartEnd()
	if got, want := start.String(), "FlutterLogoDecoration(textColor: "+ui.NewColor(0xffd4f144).String()+", style: stacked)"; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if got, want := FlutterLogoDecorationLerp(nil, end, 0.5).String(),
		"FlutterLogoDecoration(textColor: "+ui.NewColor(0xff81d4fa).String()+", style: stacked, transition -1.0:0.5)"; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// Go-only: isComplex, debugAssertIsValid, getClipPath and equality.
func TestFlutterLogoMembers(t *testing.T) {
	start, end := logoStartEnd()
	if !start.IsComplex() || FlutterLogoDecorationLerp(nil, end, 0.5).IsComplex() ||
		FlutterLogoDecorationLerp(start, NewFlutterLogoDecoration(), 0.5).IsComplex() {
		t.Error("isComplex")
	}
	if !start.Equal(NewFlutterLogoDecoration(FlutterLogoDecorationOptions{TextColor: &start.TextColor,
		Style: FlutterLogoStyleStacked})) || start.Equal(end) || start.Equal(newOtherDecoration()) {
		t.Error("==")
	}
	if !NewFlutterLogoDecoration().GetClipPath(ui.NewRectFromLTWH(1, 2, 3, 4), ui.TextDirectionLtr).Contains(ui.NewOffset(2, 3)) {
		t.Error("getClipPath")
	}
	bad := newFlutterLogoDecoration(start.TextColor, start.Style, start.Margin, 0, 1.5)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("debugAssertIsValid")
			}
		}()
		bad.DebugAssertIsValid()
	}()
}
