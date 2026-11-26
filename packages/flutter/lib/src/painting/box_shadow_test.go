package painting

// Port of the BoxShadow tests of packages/flutter/test/painting/
// box_painter_test.dart and decoration_test.dart's "BoxShadow.copyWith"
// (Flutter 3.50.1). Skipped: the BorderSide/Border tests (borders.dart and
// box_border.dart, ported separately) and the eight "BoxShadow BoxStyle.*"
// golden tests (they paint a BoxDecoration in widgets; the
// painting_decorations_pixels difftest compares BoxShadow.toPaint's
// rendering with Flutter's Impeller output). hashCode checks are dropped;
// hasOneLineDescription is a one-line, non-empty String().

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func TestBoxShadowControl(t *testing.T) {
	shadow1 := NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0})
	shadow2 := *BoxShadowLerp(nil, &shadow1, 0.25)
	shadow3 := *BoxShadowLerp(&shadow1, nil, 0.25)
	if s := shadow1.String(); s == "" || strings.Contains(s, "\n") {
		t.Errorf("description %q", s)
	}
	if shadow1 != NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0}) {
		t.Error("==")
	}
	if shadow2.BlurRadius != 1.0 || shadow3.BlurRadius != 3.0 {
		t.Errorf("blur radii %v %v", shadow2.BlurRadius, shadow3.BlurRadius)
	}
	shadow4 := *BoxShadowLerp(&shadow2, &shadow3, 0.5)
	if shadow4.BlurRadius != 2.0 {
		t.Errorf("shadow4 %v", shadow4.BlurRadius)
	}
	want := []BoxShadow{shadow4, shadow1.Scale(0.5)}
	for _, got := range [][]BoxShadow{
		BoxShadowLerpList([]BoxShadow{shadow2, shadow1}, []BoxShadow{shadow3}, 0.5),
		BoxShadowLerpList([]BoxShadow{shadow2}, []BoxShadow{shadow3, shadow1}, 0.5),
	} {
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("lerpList %v", got)
		}
	}
}

func TestBoxShadowLerpIdentical(t *testing.T) {
	if BoxShadowLerp(nil, nil, 0) != nil {
		t.Error("null")
	}
	s := NewBoxShadow()
	if BoxShadowLerp(&s, &s, 0.5) != &s {
		t.Error("identical")
	}
	if BoxShadowLerpList(nil, nil, 0) != nil {
		t.Error("list null")
	}
	list := []BoxShadow{NewBoxShadow()}
	if got := BoxShadowLerpList(list, list, 0.5); &got[0] != &list[0] {
		t.Error("list identical")
	}
}

func TestBoxShadowBlurStyle(t *testing.T) {
	shadow1 := NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0})
	shadow2 := NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0, BlurStyle: ui.BlurStyleOuter})
	shadow3 := *BoxShadowLerp(&shadow1, nil, 0.25)
	shadow4 := *BoxShadowLerp(nil, &shadow1, 0.25)
	shadow5 := *BoxShadowLerp(&shadow1, &shadow2, 0.25)
	solid := NewBoxShadow(BoxShadowOptions{BlurStyle: ui.BlurStyleSolid})
	shadow6 := *BoxShadowLerp(&solid, &shadow2, 0.25)
	for i, c := range []struct{ got, want ui.BlurStyle }{
		{shadow1.BlurStyle, ui.BlurStyleNormal}, {shadow2.BlurStyle, ui.BlurStyleOuter},
		{shadow3.BlurStyle, ui.BlurStyleNormal}, {shadow4.BlurStyle, ui.BlurStyleNormal},
		{shadow5.BlurStyle, ui.BlurStyleOuter}, {shadow6.BlurStyle, ui.BlurStyleSolid},
	} {
		if c.got != c.want {
			t.Errorf("[%d] %v, want %v", i, c.got, c.want)
		}
	}
	for i, c := range []struct {
		a, b []BoxShadow
		want [2]ui.BlurStyle
	}{
		{[]BoxShadow{shadow2, shadow1}, []BoxShadow{shadow3}, [2]ui.BlurStyle{ui.BlurStyleOuter, ui.BlurStyleNormal}},
		{[]BoxShadow{shadow6}, []BoxShadow{shadow3, shadow1}, [2]ui.BlurStyle{ui.BlurStyleSolid, ui.BlurStyleNormal}},
		{[]BoxShadow{shadow3}, []BoxShadow{shadow6, shadow1}, [2]ui.BlurStyle{ui.BlurStyleSolid, ui.BlurStyleNormal}},
		{[]BoxShadow{shadow3}, []BoxShadow{shadow2, shadow1}, [2]ui.BlurStyle{ui.BlurStyleOuter, ui.BlurStyleNormal}},
	} {
		l := BoxShadowLerpList(c.a, c.b, 0.5)
		if l[0].BlurStyle != c.want[0] || l[1].BlurStyle != c.want[1] {
			t.Errorf("list %d: %v %v", i, l[0].BlurStyle, l[1].BlurStyle)
		}
	}
}

func TestBoxShadowToString(t *testing.T) {
	black := ui.NewColor(0xff000000).String()
	if got := NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0}).String(); got != "BoxShadow("+black+", Offset(0.0, 0.0), 4.0, 0.0, BlurStyle.normal)" {
		t.Error(got)
	}
	if got := NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0, BlurStyle: ui.BlurStyleSolid}).String(); got != "BoxShadow("+black+", Offset(0.0, 0.0), 4.0, 0.0, BlurStyle.solid)" {
		t.Error(got)
	}
}

func TestBoxShadowCopyWith(t *testing.T) {
	c := ui.NewColor(0xFF112233)
	o := ui.NewOffset(1.0, 2.0)
	f := 123.0
	outer := ui.BlurStyleOuter
	for i, p := range []struct {
		copy BoxShadowCopyWithOptions
		opts BoxShadowOptions
	}{
		{BoxShadowCopyWithOptions{Color: &c}, BoxShadowOptions{Color: &c}},
		{BoxShadowCopyWithOptions{Offset: &o}, BoxShadowOptions{Offset: o}},
		{BoxShadowCopyWithOptions{BlurRadius: &f}, BoxShadowOptions{BlurRadius: f}},
		{BoxShadowCopyWithOptions{SpreadRadius: &f}, BoxShadowOptions{SpreadRadius: f}},
		{BoxShadowCopyWithOptions{BlurStyle: &outer}, BoxShadowOptions{BlurStyle: outer}},
	} {
		want := NewBoxShadow(p.opts)
		if NewBoxShadow() == want {
			t.Errorf("[%d] default equals %v", i, want)
		}
		if got := NewBoxShadow().CopyWith(p.copy); got != want {
			t.Errorf("[%d] got %v, want %v", i, got, want)
		}
	}
}

// Go-only: toPaint's mask filter, and debugDisableShadows.
func TestBoxShadowToPaint(t *testing.T) {
	s := NewBoxShadow(BoxShadowOptions{BlurRadius: 4.0, BlurStyle: ui.BlurStyleInner})
	p := s.ToPaint()
	want := ui.NewMaskFilterBlur(ui.BlurStyleInner, ui.ShadowConvertRadiusToSigma(4.0))
	if p.MaskFilter() == nil || p.MaskFilter().String() != want.String() || p.Color() != s.Color {
		t.Errorf("paint %v", p)
	}
	defer func(old bool) { DebugDisableShadows = old }(DebugDisableShadows)
	DebugDisableShadows = true
	if s.ToPaint().MaskFilter() != nil {
		t.Error("debugDisableShadows")
	}
}
