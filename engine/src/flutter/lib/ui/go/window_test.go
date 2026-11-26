package ui

// Ports of engine/src/flutter/testing/dart/window_test.dart,
// platform_dispatcher_test.dart, gesture_settings_test.dart and
// platform_view_test.dart (Flutter 3.50.1), plus tests of the hooks.dart
// entry points (views, metrics, displays, locales, user settings, focus)
// that the engine's Dart tests can't reach.
//
// Skipped: window_test's "window.sendPlatformMessage preserves callback
// zone" (Go has no zones; platform_message_test.go checks the reply is
// delivered on the sender's loop). FrameTiming.toString and
// scheduleWarmUpFrame are tested in platform_configuration_test.go.

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// withTesterEngine installs an engine configured as flutter_tester's
// (tester_main.cc) for the duration of the test.
func withTesterEngine(t *testing.T) *PlatformConfiguration {
	t.Helper()
	pc := NewPlatformConfiguration(noopClient{})
	restore := SetPlatformConfiguration(pc)
	pd := PlatformDispatcherInstance()
	savedConfig, savedView := pd.configuration, pd.ImplicitView().configuration
	pc.UpdateDisplays([]DisplayData{{ID: 0, Width: 2400, Height: 1800, PixelRatio: 3, RefreshRate: 60}})
	m := NewViewportMetrics()
	m.DevicePixelRatio, m.PhysicalWidth, m.PhysicalHeight = 3, 2400, 1800
	pc.UpdateViewMetrics(0, m)
	t.Cleanup(func() {
		pd.configuration = savedConfig
		pd.ImplicitView().configuration = savedView
		pd.updateDisplays(nil)
		pd.settings = platformDispatcherSettings{brieflyShowPassword: true}
		pd.views.onMetricsChanged = nil
		restore()
	})
	return pc
}

func TestComputePlatformResolvedLocaleBasic(t *testing.T) {
	hans, cn, fr, us := "Hans", "CN", "FR", "US"
	supported := []Locale{
		NewLocaleFromSubtags(LocaleFromSubtagsOptions{LanguageCode: "zh", ScriptCode: &hans, CountryCode: &cn}),
		NewLocaleFromSubtags(LocaleFromSubtagsOptions{LanguageCode: "fr", CountryCode: &fr}),
		NewLocaleFromSubtags(LocaleFromSubtagsOptions{LanguageCode: "en", CountryCode: &us}),
		NewLocaleFromSubtags(LocaleFromSubtagsOptions{LanguageCode: "en"}),
	}
	// The default implementation returns null due to lack of a real platform.
	if r := PlatformDispatcherInstance().ComputePlatformResolvedLocale(supported); r != nil {
		t.Fatalf("got %v", r)
	}
}

type localeClient struct {
	noopClient
	data []*string
	set  string
}

func (c *localeClient) SetApplicationLocale(l string) { c.set = l }
func (c *localeClient) ComputePlatformResolvedLocale(d []*string) []string {
	c.data = d
	return []string{"fr", "", "Latn"}
}

func TestComputePlatformResolvedLocaleWithAClient(t *testing.T) {
	c := &localeClient{}
	defer SetPlatformConfiguration(NewPlatformConfiguration(c))()
	us := "US"
	pd := PlatformDispatcherInstance()
	r := pd.ComputePlatformResolvedLocale([]Locale{NewLocaleFromSubtags(LocaleFromSubtagsOptions{LanguageCode: "en", CountryCode: &us})})
	if r == nil || r.String() != "fr_Latn" || r.CountryCode() != nil {
		t.Fatalf("got %v", r)
	}
	if len(c.data) != 3 || *c.data[0] != "en" || *c.data[1] != "US" || c.data[2] != nil {
		t.Fatalf("data %v", c.data)
	}
	pd.SetApplicationLocale(NewLocale("pt", "BR"))
	if c.set != "pt-BR" {
		t.Fatalf("set %q", c.set)
	}
}

func TestDisplayIsConfiguredForTheImplicitView(t *testing.T) {
	withTesterEngine(t)
	view := PlatformDispatcherInstance().ImplicitView()
	d := view.Display()
	if d.ID != 0 || d.DevicePixelRatio != view.DevicePixelRatio() || d.RefreshRate != 60 || d.Size != view.PhysicalSize() {
		t.Fatalf("%v", d)
	}
	if d.String() != "Display(id: 0, size: Size(2400.0, 1800.0), devicePixelRatio: 3.0, refreshRate: 60.0)" {
		t.Fatal(d.String())
	}
}

func TestDisplayAssertsWithoutDisplays(t *testing.T) {
	if r := panics(func() { PlatformDispatcherInstance().ImplicitView().Display() }); r == nil {
		t.Fatal("no assertion")
	}
}

func TestFlutterViewToStringContainsTheViewID(t *testing.T) {
	v := PlatformDispatcherInstance().ImplicitView()
	if v.ViewID() != 0 || v.String() != "FlutterView(id: 0)" {
		t.Fatal(v)
	}
}

func TestScaleFontSizeIsTheIdentityByDefault(t *testing.T) {
	pd := PlatformDispatcherInstance()
	for _, v := range []float64{0, 1, 2, 3, 3.4} {
		if got := pd.ScaleFontSize(v); got != v {
			t.Errorf("scaleFontSize(%v) = %v", v, got)
		}
	}
}

func TestViewConstraintsTight(t *testing.T) {
	c := NewViewConstraintsTight(NewSize(200, 300))
	if c.MinWidth != 200 || c.MaxWidth != 200 || c.MinHeight != 300 || c.MaxHeight != 300 {
		t.Fatal(c)
	}
	if !c.IsTight() || !c.IsSatisfiedBy(NewSize(200, 300)) || c.IsSatisfiedBy(NewSize(400, 500)) ||
		!c.Div(2).Equal(NewViewConstraintsTight(NewSize(100, 150))) {
		t.Fatal(c)
	}
}

func TestViewConstraintsUnconstrained(t *testing.T) {
	c := NewViewConstraints()
	if c.MinWidth != 0 || c.MaxWidth != math.Inf(1) || c.MinHeight != 0 || c.MaxHeight != math.Inf(1) {
		t.Fatal(c)
	}
	if c.IsTight() || !c.IsSatisfiedBy(NewSize(200, 300)) || !c.IsSatisfiedBy(NewSize(400, 500)) || !c.Div(2).Equal(NewViewConstraints()) {
		t.Fatal(c)
	}
}

func TestViewConstraints(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	c := NewViewConstraints(ViewConstraintsOptions{MinWidth: f(100), MaxWidth: f(200), MinHeight: f(300), MaxHeight: f(400)})
	if c.MinWidth != 100 || c.MaxWidth != 200 || c.MinHeight != 300 || c.MaxHeight != 400 || c.IsTight() ||
		!c.IsSatisfiedBy(NewSize(200, 300)) || c.IsSatisfiedBy(NewSize(400, 500)) {
		t.Fatal(c)
	}
	if !c.Div(2).Equal(NewViewConstraints(ViewConstraintsOptions{MinWidth: f(50), MaxWidth: f(100), MinHeight: f(150), MaxHeight: f(200)})) {
		t.Fatal(c.Div(2))
	}
	for want, c := range map[string]ViewConstraints{
		"ViewConstraints(unconstrained)":                    NewViewConstraints(),
		"ViewConstraints(biggest)":                          NewViewConstraints(ViewConstraintsOptions{MinWidth: f(math.Inf(1)), MinHeight: f(math.Inf(1))}),
		"ViewConstraints(100.0<=w<=200.0, 300.0<=h<=400.0)": c,
		"ViewConstraints(w=66.7, h=1.0)":                    NewViewConstraintsTight(NewSize(66.66, 1)),
	} {
		if c.String() != want {
			t.Errorf("%q, want %q", c.String(), want)
		}
	}
}

func TestGestureSettingsToStringAndEquality(t *testing.T) {
	one, two, three := 1.0, 2.0, 3.0
	gs := GestureSettings{PhysicalDoubleTapSlop: &two, PhysicalTouchSlop: &one}
	if gs.String() != "GestureSettings(physicalTouchSlop: 1.0, physicalDoubleTapSlop: 2.0)" {
		t.Fatal(gs.String())
	}
	value := 2.0
	a := GestureSettings{PhysicalDoubleTapSlop: &value, PhysicalTouchSlop: &one}
	b := GestureSettings{PhysicalDoubleTapSlop: &value, PhysicalTouchSlop: &three}
	c := GestureSettings{PhysicalDoubleTapSlop: &value, PhysicalTouchSlop: &one}
	if !a.Equal(c) || !c.Equal(a) || a.Equal(b) || c.Equal(b) || b.Equal(a) || b.Equal(c) {
		t.Fatal("equality")
	}
	copyA := GestureSettings{PhysicalDoubleTapSlop: &one, PhysicalTouchSlop: &one}.CopyWith()
	if *copyA.PhysicalDoubleTapSlop != 1 || *copyA.PhysicalTouchSlop != 1 {
		t.Fatal("copyWith")
	}
	copyB := copyA.CopyWith(GestureSettingsCopyWithOptions{PhysicalDoubleTapSlop: &two, PhysicalTouchSlop: &two})
	if *copyB.PhysicalDoubleTapSlop != 2 || *copyB.PhysicalTouchSlop != 2 {
		t.Fatal("copyWith")
	}
	if s := (GestureSettings{}); s.PhysicalDoubleTapSlop != nil || s.PhysicalTouchSlop != nil {
		t.Fatal("defaults")
	}
}

// platform_view_test.dart: rendering a platform view layer in a frame
// emits no error.
type renderClient struct {
	noopClient
	trees []*EngineLayerTree
	dprs  []float64
}

func (c *renderClient) Render(viewID int, tree *EngineLayerTree, dpr float64) {
	c.trees = append(c.trees, tree)
	c.dprs = append(c.dprs, dpr)
}

func TestPlatformViewLayersDoNotEmitErrors(t *testing.T) {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	c := &renderClient{}
	pc := NewPlatformConfiguration(c)
	defer SetPlatformConfiguration(pc)()
	var logged []string
	defer func(l func(string)) { engineLog = l }(engineLog)
	engineLog = func(m string) { logged = append(logged, m) }
	builder := NewSceneBuilder()
	builder.AddPlatformView(1)
	pd := PlatformDispatcherInstance()
	defer pd.SetOnBeginFrame(nil)
	pd.SetOnBeginFrame(func(time.Duration) {
		scene := builder.Build()
		pd.ImplicitView().Render(scene)
		scene.Dispose()
	})
	pd.ScheduleFrame()
	pc.BeginFrame(time.Millisecond, 1)
	if len(c.trees) != 1 || len(logged) != 0 {
		t.Fatalf("trees %d, log %v", len(c.trees), logged)
	}
	if dl := c.trees[0].Flatten(); dl == nil {
		t.Fatal("no display list")
	} else {
		dl.Release()
	}
}

func TestRenderGoesToTheClientWithTheViewMetrics(t *testing.T) {
	c := &renderClient{}
	pc := NewPlatformConfiguration(c)
	defer SetPlatformConfiguration(pc)()
	view := PlatformDispatcherInstance().ImplicitView()
	saved := view.configuration
	defer func() { view.configuration = saved }()
	m := NewViewportMetrics()
	m.DevicePixelRatio, m.PhysicalWidth, m.PhysicalHeight = 2, 300.7, 200
	pc.UpdateViewMetrics(0, m)
	scene := NewSceneBuilder().Build()
	view.Render(scene)
	size := NewSize(10, 20)
	view.Render(scene, FlutterViewRenderOptions{Size: &size})
	if len(c.trees) != 2 || c.dprs[0] != 2 {
		t.Fatalf("%d trees", len(c.trees))
	}
	if w, h := c.trees[0].FrameSize(); w != 300 || h != 200 {
		t.Fatalf("frame %dx%d", w, h)
	}
	if w, h := c.trees[1].FrameSize(); w != 10 || h != 20 {
		t.Fatalf("frame %dx%d", w, h)
	}
	// A view the engine has no metrics for drops the frame.
	other := &FlutterView{viewID: 5, platformDispatcher: PlatformDispatcherInstance(), configuration: newViewConfiguration()}
	other.Render(scene)
	if len(c.trees) != 2 {
		t.Fatal("rendered an unknown view")
	}
}

func TestDrawShadowUsesTheImplicitViewsDevicePixelRatio(t *testing.T) {
	if implicitViewDevicePixelRatio() != 1 {
		t.Fatal("default")
	}
	pc := withTesterEngine(t)
	if implicitViewDevicePixelRatio() != 3 {
		t.Fatal("tester")
	}
	_ = pc
}

// ---- hooks.dart

func TestBuildViewConfiguration(t *testing.T) {
	m := NewViewportMetrics()
	m.DevicePixelRatio, m.PhysicalWidth, m.PhysicalHeight = 2, 800, 600
	m.PhysicalPaddingTop, m.PhysicalPaddingRight, m.PhysicalPaddingBottom, m.PhysicalPaddingLeft = 40, 10, 30, 0
	m.PhysicalViewInsetTop, m.PhysicalViewInsetRight, m.PhysicalViewInsetBottom, m.PhysicalViewInsetLeft = 10, 20, 0, 5
	m.PhysicalSystemGestureInsetTop, m.PhysicalSystemGestureInsetRight = -3, 4
	m.PhysicalSystemGestureInsetBottom, m.PhysicalSystemGestureInsetLeft = 0, -1
	m.PhysicalTouchSlop = 16
	m.PhysicalDisplayFeaturesBounds = []float64{100, 0, 104, 600, 0, 0, 20, 10}
	m.PhysicalDisplayFeaturesType = []int{1, 3}
	m.PhysicalDisplayFeaturesState = []int{2, 7}
	m.DisplayID = 4
	m.PhysicalMinWidthConstraint, m.PhysicalMaxWidthConstraint = 1, 2
	m.PhysicalMinHeightConstraint, m.PhysicalMaxHeightConstraint = 3, 4
	m.PhysicalDisplayCornerRadiusTopLeft, m.PhysicalDisplayCornerRadiusTopRight = 1, 2
	m.PhysicalDisplayCornerRadiusBottomRight, m.PhysicalDisplayCornerRadiusBottomLeft = 3, 4
	c := buildViewConfiguration(m)
	check := func(name, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
	check("size", c.size.String(), "Size(800.0, 600.0)")
	check("viewPadding", c.viewPadding.String(), "ViewPadding(left: 0.0, top: 40.0, right: 10.0, bottom: 30.0)")
	check("viewInsets", c.viewInsets.String(), "ViewPadding(left: 5.0, top: 10.0, right: 20.0, bottom: 0.0)")
	check("padding", c.padding.String(), "ViewPadding(left: 0.0, top: 30.0, right: 0.0, bottom: 30.0)")
	check("systemGestureInsets", c.systemGestureInsets.String(), "ViewPadding(left: 0.0, top: 0.0, right: 4.0, bottom: 0.0)")
	check("gestureSettings", c.gestureSettings.String(), "GestureSettings(physicalTouchSlop: 16.0, physicalDoubleTapSlop: null)")
	check("displayFeatures", c.displayFeatures[0].String()+"|"+c.displayFeatures[1].String(),
		"DisplayFeature(rect: Rect.fromLTRB(50.0, 0.0, 52.0, 300.0), type: DisplayFeatureType.fold, state: DisplayFeatureState.postureHalfOpened)|"+
			"DisplayFeature(rect: Rect.fromLTRB(0.0, 0.0, 10.0, 5.0), type: DisplayFeatureType.cutout, state: DisplayFeatureState.unknown)")
	check("constraints", c.viewConstraints.String(), "ViewConstraints(1.0<=w<=2.0, 3.0<=h<=4.0)")
	check("corner radii", c.displayCornerRadii.String(), "DisplayCornerRadii(topLeft: 1.0, topRight: 2.0, bottomRight: 3.0, bottomLeft: 4.0)")
	if c.displayID != 4 || c.devicePixelRatio != 2 {
		t.Fatal("id/dpr")
	}
	// Defaults: unset touch slop and radii are null.
	d := buildViewConfiguration(NewViewportMetrics())
	if d.gestureSettings.PhysicalTouchSlop != nil || d.displayCornerRadii != nil || len(d.displayFeatures) != 0 {
		t.Fatal("defaults")
	}
	// Inconsistent radii are an ArgumentError (debug builds).
	m2 := NewViewportMetrics()
	m2.PhysicalDisplayCornerRadiusTopLeft = 2
	r := panics(func() { buildViewConfiguration(m2) })
	if e, ok := r.(dart.ArgumentError); !ok || !strings.Contains(e.Error(), "The display corner radii must be either all set or all unset.\nProvided values were inconsistent:\n  TopLeft: 2.0\n  TopRight: -1.0") {
		t.Fatalf("got %v", r)
	}
	// Bounds must be four values per feature.
	m3 := NewViewportMetrics()
	m3.PhysicalDisplayFeaturesBounds = []float64{1, 2, 3}
	m3.PhysicalDisplayFeaturesType = []int{1}
	m3.PhysicalDisplayFeaturesState = []int{0}
	expectPanic(t, "Bounds are rectangles, requiring 4 measurements each", func() { buildViewConfiguration(m3) })
}

func TestViewsAddRemoveAndMetrics(t *testing.T) {
	pc := NewPlatformConfiguration(noopClient{})
	pd := PlatformDispatcherInstance()
	calls := 0
	pd.SetOnMetricsChanged(func() { calls++ })
	defer pd.SetOnMetricsChanged(nil)
	var logged []string
	defer func(l func(string)) { engineLog = l }(engineLog)
	engineLog = func(m string) { logged = append(logged, m) }

	m := NewViewportMetrics()
	m.DevicePixelRatio = 2
	if !pc.AddView(7, m) || pc.AddView(7, m) {
		t.Fatal("add")
	}
	if pd.View(7) == nil || pd.View(7).DevicePixelRatio() != 2 || calls != 1 {
		t.Fatal("view 7")
	}
	var ids []int
	for v := range pd.Views() {
		ids = append(ids, v.ViewID())
	}
	if len(ids) != 2 || ids[0] != 0 || ids[1] != 7 {
		t.Fatal(ids)
	}
	m.DevicePixelRatio = 3
	if !pc.UpdateViewMetrics(7, m) || pd.View(7).DevicePixelRatio() != 3 || calls != 2 {
		t.Fatal("update")
	}
	if pc.UpdateViewMetrics(8, m) || calls != 2 {
		t.Fatal("update of an unknown view")
	}
	if !pc.RemoveView(7) || pc.RemoveView(7) || pd.View(7) != nil || calls != 3 {
		t.Fatal("remove")
	}
	if strings.Join(logged, "|") != "View #7 already exists.|View #7 doesn't exist." {
		t.Fatalf("log %v", logged)
	}
	expectPanic(t, "The implicit view #0 cannot be removed.", func() { pc.RemoveView(0) })
	// The dispatcher's own checks (engine and framework out of sync).
	expectPanic(t, "View ID 0 already exists.", func() { pd.addView(0, newViewConfiguration()) })
	expectPanic(t, "View ID 9 does not exist.", func() { pd.removeView(9) })
	expectPanic(t, "The implicit view #0 can not be removed.", func() { pd.removeView(0) })
	expectPanic(t, "View 9 does not exist.", func() { pd.updateWindowMetrics(9, newViewConfiguration()) })
}

func TestSingletonWindowFollowsTheImplicitView(t *testing.T) {
	withTesterEngine(t)
	if Window.DevicePixelRatio() != 3 || Window.PhysicalSize() != NewSize(2400, 1800) || Window.ViewID() != 0 {
		t.Fatal("window metrics")
	}
	calls := 0
	Window.SetOnMetricsChanged(func() { calls++ })
	if PlatformDispatcherInstance().OnMetricsChanged() == nil || Window.OnMetricsChanged() == nil {
		t.Fatal("forwarding")
	}
	Window.SetOnMetricsChanged(nil)
}

func TestUpdateDisplaysAndFocus(t *testing.T) {
	pc := NewPlatformConfiguration(noopClient{})
	pd := PlatformDispatcherInstance()
	defer pd.updateDisplays(nil)
	pc.UpdateDisplays([]DisplayData{{ID: 2, Width: 10, Height: 20, PixelRatio: 2, RefreshRate: 120}, {ID: 1}})
	var got []string
	for d := range pd.Displays() {
		got = append(got, d.String())
	}
	if strings.Join(got, "|") != "Display(id: 2, size: Size(10.0, 20.0), devicePixelRatio: 2.0, refreshRate: 120.0)|Display(id: 1, size: Size(0.0, 0.0), devicePixelRatio: 0.0, refreshRate: 0.0)" {
		t.Fatal(got)
	}
	var events []ViewFocusEvent
	pd.SetOnViewFocusChange(func(e ViewFocusEvent) { events = append(events, e) })
	defer pd.SetOnViewFocusChange(nil)
	pc.SendViewFocusEvent(ViewFocusEvent{ViewID: 0, State: ViewFocusStateFocused, Direction: ViewFocusDirectionBackward})
	if len(events) != 1 || events[0].String() != "ViewFocusEvent(viewId: 0, state: ViewFocusState.focused, direction: ViewFocusDirection.backward)" {
		t.Fatal(events)
	}
	if pd.EngineID() != nil {
		t.Fatal("engine id before set")
	}
	pc.SetEngineID(42)
	if *pd.EngineID() != 42 {
		t.Fatal("engine id")
	}
	pd.views.engineID = nil
}

type focusClient struct {
	noopClient
	got []string
}

func (c *focusClient) RequestViewFocusChange(id int, s ViewFocusState, d ViewFocusDirection) {
	c.got = append(c.got, s.String()+" "+d.String())
}

func TestRequestViewFocusChangeReachesTheClient(t *testing.T) {
	c := &focusClient{}
	defer SetPlatformConfiguration(NewPlatformConfiguration(c))()
	PlatformDispatcherInstance().RequestViewFocusChange(0, ViewFocusStateUnfocused, ViewFocusDirectionForward)
	if len(c.got) != 1 || c.got[0] != "ViewFocusState.unfocused ViewFocusDirection.forward" {
		t.Fatal(c.got)
	}
}

func TestUpdateLocales(t *testing.T) {
	pc := withTesterEngine(t)
	pd := PlatformDispatcherInstance()
	var log []string
	pd.SetOnLocaleChanged(func() { log = append(log, "locale") })
	pd.SetOnPlatformConfigurationChanged(func() { log = append(log, "config") })
	defer pd.SetOnPlatformConfigurationChanged(nil)
	if pd.Locale().String() != "und" {
		t.Fatal(pd.Locale())
	}
	pc.UpdateLocales([]string{"en", "US", "", "", "zh", "", "Hant", "x"})
	pc.UpdateLocales([]string{"en", "US", "", "", "zh", "", "Hant", "y"}) // the same locales
	if strings.Join(log, ",") != "config,locale" {
		t.Fatal(log)
	}
	if pd.Locale().String() != "en_US" || pd.Locales()[1].String() != "zh_Hant" || pc.LocaleClosure() != "en_US" {
		t.Fatal(pd.Locales())
	}
	pc.UpdateLocales([]string{"en", "US", "", ""}) // fewer locales
	if len(pd.Locales()) != 1 || len(log) != 4 {
		t.Fatal(log)
	}
}

func TestUpdateUserSettingsData(t *testing.T) {
	pc := withTesterEngine(t)
	pd := PlatformDispatcherInstance()
	var log []string
	pd.SetOnPlatformConfigurationChanged(func() { log = append(log, "config") })
	defer pd.SetOnPlatformConfigurationChanged(nil)
	pd.SetOnTextScaleFactorChanged(func() { log = append(log, "textScale") })
	pd.SetOnPlatformBrightnessChanged(func() { log = append(log, "brightness") })
	pd.SetOnSystemFontFamilyChanged(func() { log = append(log, "font") })

	pc.UpdateUserSettingsData("{}") // empty: ignored
	pc.UpdateUserSettingsData(`{"textScaleFactor": 1, "alwaysUse24HourFormat": false, "platformBrightness": "light"}`)
	if len(log) != 0 {
		t.Fatal(log) // nothing changed
	}
	pc.UpdateUserSettingsData(`{"textScaleFactor": 1.5, "alwaysUse24HourFormat": true, "platformBrightness": "dark",
		"systemFontFamily": "Inter", "nativeSpellCheckServiceDefined": true, "supportsShowingSystemContextMenu": true,
		"brieflyShowPassword": false}`)
	if strings.Join(log, ",") != "config,textScale,brightness,font" {
		t.Fatal(log)
	}
	if pd.TextScaleFactor() != 1.5 || !pd.AlwaysUse24HourFormat() || pd.PlatformBrightness() != BrightnessDark ||
		*pd.SystemFontFamily() != "Inter" || !pd.NativeSpellCheckServiceDefined() || !pd.SupportsShowingSystemContextMenu() ||
		pd.BrieflyShowPassword() {
		t.Fatal("settings")
	}
	// A null font family is a change, but copyWith keeps the old family;
	// the optional flags fall back to false, brieflyShowPassword is kept.
	log = nil
	pc.UpdateUserSettingsData(`{"textScaleFactor": 1.5, "alwaysUse24HourFormat": true, "platformBrightness": "dark"}`)
	if strings.Join(log, ",") != "config,font" || *pd.SystemFontFamily() != "Inter" ||
		pd.NativeSpellCheckServiceDefined() || pd.SupportsShowingSystemContextMenu() || pd.BrieflyShowPassword() {
		t.Fatal(log)
	}
	// A configuration id alone triggers the update.
	log = nil
	pc.UpdateUserSettingsData(`{"textScaleFactor": 1.5, "alwaysUse24HourFormat": true, "platformBrightness": "dark", "systemFontFamily": "Inter", "configurationId": 3}`)
	if strings.Join(log, ",") != "config" || *pd.configuration.configurationID != 3 {
		t.Fatal(log)
	}
	expectPanic(t, "Bad state: purple is not a valid platformBrightness.", func() {
		pc.UpdateUserSettingsData(`{"textScaleFactor": 1, "alwaysUse24HourFormat": false, "platformBrightness": "purple"}`)
	})
	expectPanic(t, "Null check operator used on a null value", func() {
		pc.UpdateUserSettingsData(`{"alwaysUse24HourFormat": false, "platformBrightness": "light"}`)
	})
}

type fontScaleClient struct {
	noopClient
	calls int
}

func (c *fontScaleClient) GetScaledFontSize(size float64, id int) float64 {
	c.calls++
	if id != 9 {
		return -1
	}
	return size*size/10 + 1
}

func TestScaleFontSizeWithAConfigurationID(t *testing.T) {
	withTesterEngine(t)
	c := &fontScaleClient{}
	pc := NewPlatformConfiguration(c)
	defer SetPlatformConfiguration(pc)()
	pd := PlatformDispatcherInstance()
	// A text scale factor of 1 is the identity, even with a configuration.
	pc.UpdateUserSettingsData(`{"textScaleFactor": 1, "alwaysUse24HourFormat": false, "platformBrightness": "light", "configurationId": 9}`)
	if pd.ScaleFontSize(10) != 10 || c.calls != 0 {
		t.Fatal("identity")
	}
	pd.configuration.configurationID = nil
	pc.UpdateUserSettingsData(`{"textScaleFactor": 2, "alwaysUse24HourFormat": false, "platformBrightness": "light"}`)
	if pd.ScaleFontSize(10.5) != 21 || c.calls != 0 {
		t.Fatal("linear scaling without a configuration id")
	}
	pc.UpdateUserSettingsData(`{"textScaleFactor": 2, "alwaysUse24HourFormat": false, "platformBrightness": "light", "configurationId": 9}`)
	if got := pd.ScaleFontSize(10); got != 11 {
		t.Fatalf("10: %v", got)
	}
	// 10.5 interpolates between 10 (cached) and 11.
	if got, want := pd.ScaleFontSize(10.5), 11+(13.1-11)*0.5; got != want || c.calls != 2 {
		t.Fatalf("10.5: %v (%d calls)", got, c.calls)
	}
	// A new text scale factor clears the cache.
	pc.UpdateUserSettingsData(`{"textScaleFactor": 3, "alwaysUse24HourFormat": false, "platformBrightness": "light", "configurationId": 9}`)
	pd.ScaleFontSize(10)
	if c.calls != 3 {
		t.Fatalf("%d calls", c.calls)
	}
	pc.UpdateUserSettingsData(`{"textScaleFactor": 3, "alwaysUse24HourFormat": false, "platformBrightness": "light", "configurationId": 4}`)
	expectPanic(t, "Flutter Error: incorrect configuration id: 4.", func() { pd.ScaleFontSize(12) })
}

func TestDispatchError(t *testing.T) {
	pc := NewPlatformConfiguration(noopClient{})
	pd := PlatformDispatcherInstance()
	if pc.DispatchError("x", nil) {
		t.Fatal("handled without onError")
	}
	var got []string
	pd.SetOnError(func(e any, s dart.StackTrace) bool { got = append(got, dart.ToString(e)+"|"+s.String()); return true })
	defer pd.SetOnError(nil)
	if !pc.DispatchError("boom", nil) || len(got) != 1 || got[0] != "boom|" {
		t.Fatal(got)
	}
}

type miscClient struct {
	noopClient
	reset int
}

func (c *miscClient) DefaultRouteName() string         { return "/home" }
func (c *miscClient) GetPersistentIsolateData() []byte { return []byte{1} }
func (c *miscClient) ResetInternalState()              { c.reset++ }

func TestMiscNatives(t *testing.T) {
	pd := PlatformDispatcherInstance()
	if pd.DefaultRouteName() != "/" || pd.GetPersistentIsolateData() != nil {
		t.Fatal("defaults")
	}
	c := &miscClient{}
	defer SetPlatformConfiguration(NewPlatformConfiguration(c))()
	pd.ResetInternalState()
	if pd.DefaultRouteName() != "/home" || len(pd.GetPersistentIsolateData()) != 1 || c.reset != 1 {
		t.Fatal("client")
	}
	defer pd.SetIsolateDebugName(EngineIsolateDebugName())
	pd.SetIsolateDebugName("x")
	if EngineIsolateDebugName() != "x" {
		t.Fatal("debug name")
	}
}

func TestSystemColor(t *testing.T) {
	if SystemColorPlatformProvidesSystemColors() || SystemColorLight.Brightness != BrightnessLight || SystemColorDark.Brightness != BrightnessDark {
		t.Fatal("statics")
	}
	expectPanic(t, "Unsupported operation: SystemColor not supported on the current platform.", func() { SystemColorLight.Canvas() })
	c := NewColor(0xFF000000)
	if (SystemColor{Name: "a"}).IsSupported() || !(SystemColor{Name: "a", Value: &c}).IsSupported() {
		t.Fatal("isSupported")
	}
}

func TestRunOnPlatformThread(t *testing.T) {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	if !IsRunningOnPlatformThread {
		t.Fatal("not on the platform thread")
	}
	f := RunOnPlatformThread(func() int { return 42 })
	var got int
	async.OnComplete(f, func(v int) { got = v }, func(error) {})
	if got != 0 {
		t.Fatal("ran synchronously")
	}
	loop.FlushTimers(time.Second, false)
	loop.FlushMicrotasks()
	if got != 42 {
		t.Fatal(got)
	}
	g := RunOnPlatformThread(func() int { panic(dart.StateError{Message: "x"}) })
	var err error
	async.OnComplete(g, func(int) {}, func(e error) { err = e })
	loop.FlushTimers(time.Second, false)
	loop.FlushMicrotasks()
	if err == nil || err.Error() != "Bad state: x" {
		t.Fatal(err)
	}
}

func TestKeepToString(t *testing.T) {
	if KeepToString.Name != "flutter:keep-to-string" || KeepToString.Options != nil {
		t.Fatal(KeepToString)
	}
}

func TestDeclaredTypes(t *testing.T) {
	var onError ErrorCallback = func(any, dart.StackTrace) bool { return true }
	var onFocus ViewFocusChangeCallback = func(ViewFocusEvent) {}
	var palette *SystemColorPalette = SystemColorLight
	var window *SingletonFlutterWindow = Window
	var handle CallbackHandle = NewCallbackHandleFromRawHandle(1)
	if onError == nil || onFocus == nil || palette == nil || window.ViewID() != 0 || handle.ToRawHandle() != 1 {
		t.Fatal("types")
	}
	e := PictureRasterizationException{Message: "Scene has been disposed."}
	if e.Error() != "Failed to rasterize a picture: Scene has been disposed.." {
		t.Fatal(e.Error())
	}
}
