package app

// G7 (Retina scale): the window's device pixel ratio and metrics as the
// runner reports them to dart:ui (no window is opened).

import (
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

func TestDevicePixelRatioIsTheBackingScale(t *testing.T) {
	for _, c := range []struct {
		window, framebuffer int
		layer, want         float64
	}{
		{800, 1600, 2, 2},    // Retina
		{800, 800, 1, 1},     // standard display
		{640, 1920, 3, 3},    // 3x
		{1000, 1500, 2, 1.5}, // a scaled mode: the framebuffer decides
		{0, 0, 2, 2},         // minimized: the layer's scale
		{0, 0, 0, 1},
	} {
		if got := devicePixelRatio(c.window, c.framebuffer, c.layer); got != c.want {
			t.Errorf("devicePixelRatio(%d, %d, %v) = %v, want %v", c.window, c.framebuffer, c.layer, got, c.want)
		}
	}
}

func TestViewportMetrics(t *testing.T) {
	m := viewportMetrics(800, 600, 1600, 1200, 2)
	if m.DevicePixelRatio != 2 || m.PhysicalWidth != 1600 || m.PhysicalHeight != 1200 ||
		m.PhysicalMinWidthConstraint != 1600 || m.PhysicalMaxWidthConstraint != 1600 ||
		m.PhysicalMinHeightConstraint != 1200 || m.PhysicalMaxHeightConstraint != 1200 ||
		m.PhysicalTouchSlop != -1 || m.DisplayID != 0 {
		t.Fatalf("%+v", m)
	}
	// Reported to dart:ui, the view's metrics follow: the logical size is
	// the window's.
	pc := ui.NewPlatformConfiguration(nil)
	view := ui.PlatformDispatcherInstance().ImplicitView()
	pc.UpdateViewMetrics(0, m)
	defer pc.UpdateViewMetrics(0, ui.NewViewportMetrics())
	if view.DevicePixelRatio() != 2 || view.PhysicalSize() != ui.NewSize(1600, 1200) ||
		view.PhysicalConstraints().String() != "ViewConstraints(w=1600.0, h=1200.0)" {
		t.Fatalf("%v %v", view.DevicePixelRatio(), view.PhysicalSize())
	}
	if d := displayData(1512, 982, 120, 2); d.Width != 3024 || d.Height != 1964 || d.PixelRatio != 2 || d.RefreshRate != 120 {
		t.Fatalf("%+v", d)
	}
	if r := rootTransform(2); r[0] != 2 || r[5] != 2 || r[10] != 1 || r[15] != 1 {
		t.Fatal(r)
	}
}
