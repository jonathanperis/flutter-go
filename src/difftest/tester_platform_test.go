package difftest

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

// testerClient is a PlatformConfigurationClient that produces no frames,
// as flutter_tester's headless platform view.
type testerClient struct{}

func (testerClient) ScheduleFrame()             {}
func (testerClient) EndWarmUpFrame()            {}
func (testerClient) SetNeedsReportTimings(bool) {}

// testerMetrics are flutter_tester's implicit view metrics
// (shell/testing/tester_main.cc's ConfigureShell): 2400×1800 physical
// pixels at a device pixel ratio of 3.0, tight constraints, display 0.
func testerMetrics() ui.ViewportMetrics {
	m := ui.NewViewportMetrics()
	m.DevicePixelRatio = 3.0
	m.PhysicalWidth, m.PhysicalHeight = 2400, 1800
	m.PhysicalMinWidthConstraint, m.PhysicalMaxWidthConstraint = 2400, 2400
	m.PhysicalMinHeightConstraint, m.PhysicalMaxHeightConstraint = 1800, 1800
	return m
}

// useTesterPlatform installs an engine configured as flutter_tester's
// (its display and implicit view) and returns a function restoring the
// previous engine and the defaults.
func useTesterPlatform() (*ui.PlatformConfiguration, func()) {
	pc := ui.NewPlatformConfiguration(testerClient{})
	restore := ui.SetPlatformConfiguration(pc)
	pc.UpdateDisplays([]ui.DisplayData{{ID: 0, Width: 2400, Height: 1800, PixelRatio: 3.0, RefreshRate: 60}})
	pc.UpdateViewMetrics(0, testerMetrics())
	// tester_main.cc sends the test locales en_US and zh_CN.
	pc.UpdateLocales([]string{"en", "US", "", "", "zh", "CN", "", ""})
	return pc, func() {
		pc.UpdateLocales(nil)
		pc.UpdateViewMetrics(0, ui.NewViewportMetrics())
		pc.UpdateDisplays(nil)
		restore()
	}
}
