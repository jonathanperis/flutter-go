package difftest

// Differential test of dart:ui's window/platform API against Flutter
// 3.50.1 (flutter/test/window_api_test.dart): the implicit view and
// PlatformDispatcher as flutter_tester configures them (the Go side
// installs an engine with flutter_tester's display, metrics and locales,
// tester_platform_test.go), the deprecated window, and the value types.
// Compared exactly.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/isolate"
)

// dartErr is the Dart tests' err(): AssertionError's message, or the
// error's toString, or "no error".
func dartErr(f func()) (s string) {
	defer func() {
		if r := recover(); r != nil {
			if a, ok := r.(dart.AssertionFailure); ok {
				m := a.AssertionMessage()
				if m == nil {
					s = "AssertionError: null"
				} else {
					s = "AssertionError: " + fmt.Sprint(m)
				}
				return
			}
			if e, ok := r.(error); ok {
				s = e.Error()
				return
			}
			s = fmt.Sprint(r)
		}
	}()
	f()
	return "no error"
}

func topLevelCallback() string { return "top" }

func nullStr[T any](p *T) string {
	if p == nil {
		return "null"
	}
	return fmt.Sprint(*p)
}

func dstr(v float64) string { return dart.DoubleToString(v) }

func featuresString(fs []DisplayFeature) string {
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = f.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func viewState(v *FlutterView) map[string]any {
	radii := "null"
	if r := v.DisplayCornerRadii(); r != nil {
		radii = r.String()
	}
	return map[string]any{
		"toString":            v.String(),
		"viewId":              v.ViewID(),
		"devicePixelRatio":    v.DevicePixelRatio(),
		"physicalSize":        v.PhysicalSize().String(),
		"physicalConstraints": v.PhysicalConstraints().String(),
		"padding":             v.Padding().String(),
		"viewInsets":          v.ViewInsets().String(),
		"viewPadding":         v.ViewPadding().String(),
		"systemGestureInsets": v.SystemGestureInsets().String(),
		"gestureSettings":     v.GestureSettings().String(),
		"displayFeatures":     featuresString(v.DisplayFeatures()),
		"displayCornerRadii":  radii,
		"display":             v.Display().String(),
	}
}

func localesString(ls []Locale) []any {
	out := []any{}
	for _, l := range ls {
		out = append(out, l.String())
	}
	return out
}

func callbackString(c VoidCallback) string {
	if c == nil {
		return "null"
	}
	return "Closure"
}

func windowAPICases() map[string]any {
	out := map[string]any{}
	pd := PlatformDispatcherInstance()
	view := pd.ImplicitView()

	out["view"] = viewState(view)
	views := []any{}
	for v := range pd.Views() {
		views = append(views, v.String())
	}
	out["views"] = views
	out["view.0.identical"] = pd.View(0) == view
	out["view.1"] = fmt.Sprint(map[bool]string{true: "null", false: "?"}[pd.View(1) == nil])
	displays := []any{}
	for d := range pd.Displays() {
		displays = append(displays, d.String())
	}
	out["displays"] = displays
	d := view.Display()
	out["display.fields"] = []any{d.ID, d.DevicePixelRatio, d.Size.String(), d.RefreshRate}

	en := NewLocale("en", "US")
	hans, cn := "Hans", "CN"
	resolved := func(l *Locale) string {
		if l == nil {
			return "null"
		}
		return l.String()
	}
	out["dispatcher"] = map[string]any{
		"locale":                           pd.Locale().String(),
		"locale.languageTag":               pd.Locale().ToLanguageTag(),
		"locales":                          localesString(pd.Locales()),
		"textScaleFactor":                  pd.TextScaleFactor(),
		"alwaysUse24HourFormat":            pd.AlwaysUse24HourFormat(),
		"platformBrightness":               pd.PlatformBrightness().String(),
		"systemFontFamily":                 nullStr(pd.SystemFontFamily()),
		"nativeSpellCheckServiceDefined":   pd.NativeSpellCheckServiceDefined(),
		"supportsShowingSystemContextMenu": pd.SupportsShowingSystemContextMenu(),
		"brieflyShowPassword":              pd.BrieflyShowPassword(),
		"defaultRouteName":                 pd.DefaultRouteName(),
		"semanticsEnabled":                 pd.SemanticsEnabled(),
		"accessibilityFeatures":            pd.AccessibilityFeatures().String(),
		"frameData.frameNumber":            pd.FrameData().FrameNumber,
		"lineHeightScaleFactorOverride":    nullStr(pd.LineHeightScaleFactorOverride()),
		"letterSpacingOverride":            nullStr(pd.LetterSpacingOverride()),
		"wordSpacingOverride":              nullStr(pd.WordSpacingOverride()),
		"paragraphSpacingOverride":         nullStr(pd.ParagraphSpacingOverride()),
		"persistentIsolateData":            map[bool]string{true: "null", false: "data"}[pd.GetPersistentIsolateData() == nil],
		"onMetricsChanged":                 callbackString(pd.OnMetricsChanged()),
		"onLocaleChanged":                  callbackString(pd.OnLocaleChanged()),
		"onError":                          map[bool]string{true: "null", false: "Closure"}[pd.OnError() == nil],
		"onViewFocusChange":                map[bool]string{true: "null", false: "Closure"}[pd.OnViewFocusChange() == nil],
		"computePlatformResolvedLocale": resolved(pd.ComputePlatformResolvedLocale([]Locale{
			NewLocaleFromSubtags(LocaleFromSubtagsOptions{LanguageCode: "zh", ScriptCode: &hans, CountryCode: &cn}), en})),
		"computePlatformResolvedLocale.empty": resolved(pd.ComputePlatformResolvedLocale(nil)),
	}
	sizes := []any{}
	for _, v := range []float64{0, 1, 2, 3.4, 14.5, 1000.25} {
		sizes = append(sizes, pd.ScaleFontSize(v))
	}
	out["scaleFontSize"] = sizes
	out["scaleFontSize.errors"] = []any{
		dartErr(func() { pd.ScaleFontSize(-1) }),
		dartErr(func() { pd.ScaleFontSize(math.Inf(1)) }),
	}
	out["natives"] = []any{
		dartErr(func() { pd.RequestViewFocusChange(0, ViewFocusStateFocused, ViewFocusDirectionForward) }),
		dartErr(func() { pd.SetApplicationLocale(NewLocale("fr", "CA")) }),
		dartErr(func() { pd.SetIsolateDebugName("difftest") }),
		dartErr(func() { DartPluginRegistrant.EnsureInitialized() }),
	}

	w := Window
	ws := viewState(&w.FlutterView)
	ws["locale"] = w.Locale().String()
	ws["locales"] = "[" + strings.Join(func() []string {
		var s []string
		for _, l := range w.Locales() {
			s = append(s, l.String())
		}
		return s
	}(), ", ") + "]"
	ws["textScaleFactor"] = w.TextScaleFactor()
	ws["alwaysUse24HourFormat"] = w.AlwaysUse24HourFormat()
	ws["platformBrightness"] = w.PlatformBrightness().String()
	ws["systemFontFamily"] = nullStr(w.SystemFontFamily())
	ws["defaultRouteName"] = w.DefaultRouteName()
	ws["semanticsEnabled"] = w.SemanticsEnabled()
	ws["accessibilityFeatures"] = w.AccessibilityFeatures().String()
	ws["brieflyShowPassword"] = w.BrieflyShowPassword()
	ws["nativeSpellCheckServiceDefined"] = w.NativeSpellCheckServiceDefined()
	ws["supportsShowingSystemContextMenu"] = w.SupportsShowingSystemContextMenu()
	ws["frameData.frameNumber"] = w.FrameData().FrameNumber
	ws["computePlatformResolvedLocale"] = resolved(w.ComputePlatformResolvedLocale([]Locale{NewLocale("en")}))
	ws["identicalDispatcher"] = w.PlatformDispatcher() == pd
	out["window"] = ws

	out["viewPadding.zero"] = ViewPaddingZero.String()
	out["windowPadding.zero"] = WindowPadding(ViewPaddingZero).String()
	z := ViewPaddingZero
	out["viewPadding.fields"] = []any{z.Left, z.Top, z.Right, z.Bottom}

	f := func(v float64) *float64 { return &v }
	inf, nan := math.Inf(1), math.NaN()
	constraints := []ViewConstraints{
		NewViewConstraints(),
		NewViewConstraintsTight(NewSize(200, 300)),
		NewViewConstraints(ViewConstraintsOptions{MinWidth: f(100), MaxWidth: f(200), MinHeight: f(300), MaxHeight: f(400)}),
		NewViewConstraints(ViewConstraintsOptions{MinWidth: f(inf), MinHeight: f(inf)}),
		NewViewConstraints(ViewConstraintsOptions{MinWidth: f(inf), MaxHeight: f(10)}),
		NewViewConstraints(ViewConstraintsOptions{MaxWidth: f(1.25), MaxHeight: f(0.05)}),
		NewViewConstraints(ViewConstraintsOptions{MinWidth: f(2.35), MaxWidth: f(2.45), MinHeight: f(0.15), MaxHeight: f(0.15)}),
		NewViewConstraints(ViewConstraintsOptions{MinWidth: f(0), MaxWidth: f(inf), MinHeight: f(5)}),
		NewViewConstraints(ViewConstraintsOptions{MaxWidth: f(0), MaxHeight: f(0)}),
		NewViewConstraintsTight(SizeZero),
		NewViewConstraints(ViewConstraintsOptions{MinWidth: f(nan), MaxWidth: f(nan)}),
	}
	cs := []any{}
	for _, c := range constraints {
		cs = append(cs, []any{c.String(), dstr(c.MinWidth), dstr(c.MaxWidth), dstr(c.MinHeight), dstr(c.MaxHeight),
			c.IsTight(), c.IsSatisfiedBy(NewSize(200, 300)), c.IsSatisfiedBy(NewSize(400, 500)), c.IsSatisfiedBy(SizeZero),
			c.Div(2).String(), c.Div(3).String(), c.Div(0).String(),
			c.Equal(NewViewConstraintsTight(NewSize(200, 300))), c.Equal(NewViewConstraints())})
	}
	out["viewConstraints"] = cs

	fold := NewDisplayFeature(NewRectFromLTRB(10, 0, 12, 600), DisplayFeatureTypeFold, DisplayFeatureStatePostureHalfOpened)
	out["displayFeature"] = []any{
		fold.String(),
		fold.Equal(NewDisplayFeature(NewRectFromLTRB(10, 0, 12, 600), DisplayFeatureTypeFold, DisplayFeatureStatePostureHalfOpened)),
		fold.Equal(NewDisplayFeature(NewRectFromLTRB(10, 0, 12, 600), DisplayFeatureTypeHinge, DisplayFeatureStatePostureHalfOpened)),
		dartErr(func() { NewDisplayFeature(RectZero, DisplayFeatureTypeValues[3], DisplayFeatureStateValues[1]) }),
		NewDisplayFeature(RectZero, DisplayFeatureTypeCutout, DisplayFeatureStateUnknown).String(),
	}
	var types, states []any
	for _, t := range DisplayFeatureTypeValues {
		types = append(types, t.String())
	}
	for _, s := range DisplayFeatureStateValues {
		states = append(states, s.String())
	}
	out["displayFeature.enums"] = []any{types, states}
	radii := NewDisplayCornerRadii(1, 2.5, 0, 4)
	out["displayCornerRadii"] = []any{
		radii.String(),
		radii.Equal(NewDisplayCornerRadii(1, 2.5, 0, 4)),
		radii.Equal(NewDisplayCornerRadii(1, 2.5, 0, 5)),
		dartErr(func() { NewDisplayCornerRadii(-1, 0, 0, 0) }),
	}

	gs := GestureSettings{PhysicalDoubleTapSlop: f(2), PhysicalTouchSlop: f(1)}
	out["gestureSettings"] = []any{
		gs.String(),
		GestureSettings{}.String(),
		gs.CopyWith().String(),
		gs.CopyWith(GestureSettingsCopyWithOptions{PhysicalTouchSlop: f(3)}).String(),
		GestureSettings{}.CopyWith(GestureSettingsCopyWithOptions{PhysicalDoubleTapSlop: f(0.5)}).String(),
		gs.Equal(GestureSettings{PhysicalDoubleTapSlop: f(2), PhysicalTouchSlop: f(1)}),
		gs.Equal(GestureSettings{PhysicalTouchSlop: f(1)}),
	}

	out["viewFocusEvent"] = ViewFocusEvent{ViewID: 3, State: ViewFocusStateUnfocused, Direction: ViewFocusDirectionBackward}.String()
	var fstates, fdirs, brightness []any
	for _, s := range ViewFocusStateValues {
		fstates = append(fstates, s.String())
	}
	for _, s := range ViewFocusDirectionValues {
		fdirs = append(fdirs, s.String())
	}
	for _, s := range BrightnessValues {
		brightness = append(brightness, s.String())
	}
	out["enums"] = []any{fstates, fdirs, brightness}

	black := NewColor(0xFF000000)
	out["systemColor"] = []any{
		SystemColorPlatformProvidesSystemColors(),
		SystemColorLight.Brightness.String(),
		SystemColorDark.Brightness.String(),
		dartErr(func() { SystemColorLight.AccentColor() }),
		dartErr(func() { SystemColorDark.VisitedText() }),
		SystemColor{Name: "x"}.IsSupported(),
		SystemColor{Name: "y", Value: &black}.IsSupported(),
		SystemColor{Name: "z"}.Name,
	}

	hTop := PluginUtilities.GetCallbackHandle(topLevelCallback)
	anon := func(a, b int) int { return a + b }
	out["plugins"] = []any{
		hTop != nil,
		*PluginUtilities.GetCallbackHandle(topLevelCallback) == *hTop,
		PluginUtilities.GetCallbackFromHandle(*hTop).(func() string)(),
		PluginUtilities.GetCallbackHandle(anon) == nil,
		NewCallbackHandleFromRawHandle(5) == NewCallbackHandleFromRawHandle(5),
		NewCallbackHandleFromRawHandle(5) == NewCallbackHandleFromRawHandle(6),
		NewCallbackHandleFromRawHandle(7).ToRawHandle(),
		map[bool]string{true: "null", false: "?"}[PluginUtilities.GetCallbackFromHandle(NewCallbackHandleFromRawHandle(12345)) == nil],
	}

	const portName = "window_api_port"
	r1, r2 := isolate.NewReceivePort(), isolate.NewReceivePort()
	ins := IsolateNameServer
	out["isolateNameServer"] = []any{
		ins.LookupPortByName(portName) == nil,
		ins.RemovePortNameMapping(portName),
		ins.RegisterPortWithName(r1.SendPort(), portName),
		ins.LookupPortByName(portName) == r1.SendPort(),
		ins.RegisterPortWithName(r2.SendPort(), portName),
		ins.LookupPortByName(portName) == r1.SendPort(),
		ins.RemovePortNameMapping(portName),
		ins.LookupPortByName(portName) == nil,
		ins.RegisterPortWithName(r2.SendPort(), portName),
		ins.LookupPortByName(portName) == r2.SendPort(),
		ins.RemovePortNameMapping(portName),
	}
	r1.Close()
	r2.Close()

	out["platformIsolate"] = []any{
		IsRunningOnPlatformThread,
		dartErr(func() { RunOnPlatformThread(func() int { return 1 }) }),
	}
	out["keepToString"] = []any{KeepToString.Name, nullStr[any](nil)}
	return out
}

func TestWindowAPIMatchesFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/window_api.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string]any
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	pc, restore := useTesterPlatform()
	defer restore()
	// flutter_tester has produced its first frame when the test runs.
	pc.BeginFrame(time.Millisecond, 1)
	defer func(d string) { PlatformDispatcherInstance().SetIsolateDebugName(d) }(EngineIsolateDebugName())
	cases := windowAPICases()
	loop.FlushTimers(time.Second, false) // the runOnPlatformThread computation
	if len(ref) != len(cases) {
		t.Errorf("reference has %d cases, Go has %d", len(ref), len(cases))
	}
	for name, got := range cases {
		want, ok := ref[name]
		if !ok {
			t.Errorf("%s: missing from the reference", name)
			continue
		}
		if g := toJSONShape(t, got); !reflect.DeepEqual(g, want) { // exact
			t.Errorf("%s:\n Go:      %v\n Flutter: %v", name, g, want)
		}
	}
}
