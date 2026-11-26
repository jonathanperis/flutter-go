package app

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// devicePixelRatio is the window's backing scale factor (2.0 on Retina
// displays): framebuffer pixels per window point, as the macOS embedder
// reports the view's device pixel ratio (FlutterViewController's
// backingScaleFactor). GLFW gives both sizes; a zero-sized (minimized)
// window falls back to the layer's contents scale.
func devicePixelRatio(windowWidth, framebufferWidth int, layerScale float64) float64 {
	if windowWidth > 0 && framebufferWidth > 0 {
		return float64(framebufferWidth) / float64(windowWidth)
	}
	if layerScale > 0 {
		return layerScale
	}
	return 1.0
}

// viewportMetrics are the window's metrics as the macOS embedder sends them
// (FlutterViewController's updateWindowMetrics): the framebuffer's size in
// physical pixels, the device pixel ratio, tight constraints (a desktop
// window is the size it is) and display 0. ENGINE GAP: GLFW reports no
// padding, insets, system gesture insets, display features or corner
// radii, so they keep the engine's defaults (zero/unset), as on desktop.
func viewportMetrics(windowWidth, windowHeight, framebufferWidth, framebufferHeight int, layerScale float64) ui.ViewportMetrics {
	m := ui.NewViewportMetrics()
	m.DevicePixelRatio = devicePixelRatio(windowWidth, framebufferWidth, layerScale)
	m.PhysicalWidth = float64(framebufferWidth)
	m.PhysicalHeight = float64(framebufferHeight)
	m.PhysicalMinWidthConstraint = m.PhysicalWidth
	m.PhysicalMaxWidthConstraint = m.PhysicalWidth
	m.PhysicalMinHeightConstraint = m.PhysicalHeight
	m.PhysicalMaxHeightConstraint = m.PhysicalHeight
	m.DisplayID = 0
	return m
}

// displayData is the display the window is on (the macOS embedder's
// FlutterEngine display update): its size in physical pixels (the video
// mode is in points), its scale and refresh rate.
func displayData(modeWidth, modeHeight, refreshRate int, contentScale float64) ui.DisplayData {
	if contentScale <= 0 {
		contentScale = 1
	}
	return ui.DisplayData{ID: 0, Width: float64(modeWidth) * contentScale, Height: float64(modeHeight) * contentScale,
		PixelRatio: contentScale, RefreshRate: float64(refreshRate)}
}

// rootTransform is RenderView's root transform: logical pixels to
// physical (a scale by the device pixel ratio), column-major.
func rootTransform(devicePixelRatio float64) []float64 {
	return []float64{devicePixelRatio, 0, 0, 0, 0, devicePixelRatio, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}
