package ui

// Port of engine/src/flutter/lib/ui/window.dart (Flutter 3.50.1): Display,
// FlutterView, the deprecated SingletonFlutterWindow and its top-level
// `window`, and Brightness. AccessibilityFeatures is in
// platform_dispatcher_semantics.go, FrameData in platform_dispatcher.go and
// GestureSettings in pointer_dispatch.go; a view's configuration
// (_ViewConfiguration) and the engine entry points that change it are in
// platform_dispatcher_views.go.

import (
	"strconv"

	"github.com/jonathanperis/flutter-go/src/dart"
)

// Display is a physical display: its id, device pixel ratio, size in
// physical pixels and refresh rate. Displays are created by the engine
// (PlatformConfiguration.UpdateDisplays); use them by pointer.
type Display struct {
	// ID is the display's unique identifier.
	ID int
	// DevicePixelRatio is the number of device pixels for each logical
	// pixel on this display.
	DevicePixelRatio float64
	// Size is the physical size of the display, in physical pixels.
	Size Size
	// RefreshRate is the refresh rate of the display, in FPS.
	RefreshRate float64
}

func (d *Display) String() string {
	return "Display(id: " + strconv.Itoa(d.ID) + ", size: " + d.Size.String() +
		", devicePixelRatio: " + dart.DoubleToString(d.DevicePixelRatio) +
		", refreshRate: " + dart.DoubleToString(d.RefreshRate) + ")"
}

// FlutterView is a view into which a Flutter Scene is drawn: its metrics
// (physical size, device pixel ratio, paddings and insets, gesture
// settings, display features) come from the engine, and Render shows a
// scene in it. Views are created by the engine
// (PlatformConfiguration.AddView); use them by pointer.
type FlutterView struct {
	viewID             int
	platformDispatcher *PlatformDispatcher
	configuration      *viewConfiguration
	// singleton: SingletonFlutterWindow, which reads the configuration of
	// the view with its id when that view exists.
	singleton bool
}

// viewConfig is _viewConfiguration (SingletonFlutterWindow overrides it).
func (v *FlutterView) viewConfig() *viewConfiguration {
	if v.singleton {
		if view, ok := v.platformDispatcher.views.byID[v.viewID]; ok {
			return view.configuration
		}
	}
	return v.configuration
}

// ViewID is the identifier for this view.
func (v *FlutterView) ViewID() int { return v.viewID }

// PlatformDispatcher is the platform dispatcher this view is registered
// with.
func (v *FlutterView) PlatformDispatcher() *PlatformDispatcher { return v.platformDispatcher }

// Display is the Display this view is rendered on (it panics with Dart's
// assertion error when the engine reported no such display).
func (v *FlutterView) Display() *Display {
	id := v.viewConfig().displayID
	d, ok := v.platformDispatcher.views.displays[id]
	assertNoMessage(ok)
	return d
}

// DevicePixelRatio is the number of device pixels for each logical pixel
// of this view.
func (v *FlutterView) DevicePixelRatio() float64 { return v.viewConfig().devicePixelRatio }

// PhysicalConstraints are the sizing constraints in physical pixels the
// view's content must satisfy.
func (v *FlutterView) PhysicalConstraints() ViewConstraints { return v.viewConfig().viewConstraints }

// PhysicalSize is the size of the rectangle the view occupies, in physical
// pixels.
func (v *FlutterView) PhysicalSize() Size { return v.viewConfig().size }

// ViewInsets are the physical pixels on each side of the display rectangle
// into which the view can render, but over which the operating system
// will likely place system UI, such as the keyboard.
func (v *FlutterView) ViewInsets() ViewPadding { return v.viewConfig().viewInsets }

// ViewPadding are the physical pixels on each side of the display
// rectangle into which the view can render, but which may be partially
// obscured by system UI or hardware intrusions.
func (v *FlutterView) ViewPadding() ViewPadding { return v.viewConfig().viewPadding }

// SystemGestureInsets are the areas where system gestures take priority
// over the application's, in physical pixels.
func (v *FlutterView) SystemGestureInsets() ViewPadding { return v.viewConfig().systemGestureInsets }

// Padding is ViewPadding minus ViewInsets (never negative).
func (v *FlutterView) Padding() ViewPadding { return v.viewConfig().padding }

// GestureSettings are the platform's gesture settings for this view.
func (v *FlutterView) GestureSettings() GestureSettings { return v.viewConfig().gestureSettings }

// DisplayFeatures are the areas of the display obstructed by hardware
// features (folds, hinges, cutouts), in logical pixels.
func (v *FlutterView) DisplayFeatures() []DisplayFeature { return v.viewConfig().displayFeatures }

// DisplayCornerRadii are the radii of the display's rounded corners, in
// physical pixels (nil when the platform doesn't report them).
func (v *FlutterView) DisplayCornerRadii() *DisplayCornerRadii {
	return v.viewConfig().displayCornerRadii
}

// FlutterViewRenderOptions are FlutterView.render's named arguments.
type FlutterViewRenderOptions struct {
	// Size is the size of the scene in physical pixels (nil: PhysicalSize).
	Size *Size
}

// Render updates the view's rendering on the GPU with the given scene
// (built with SceneBuilder), of the given size (the view's physical size
// by default). It goes to the engine's PlatformConfigurationNativeApi::
// Render, which hands the scene's layer tree to the embedder (see
// PlatformConfigurationRenderClient).
func (v *FlutterView) Render(scene Scene, opts ...FlutterViewRenderOptions) {
	size := v.PhysicalSize()
	if len(opts) > 0 && opts[0].Size != nil {
		size = *opts[0].Size
	}
	platformConfigurationNativeAPIRender(v.viewID, scene.(*nativeScene), size.Width, size.Height)
}

// UpdateSemantics changes the semantics tree shown by this view (the
// update goes to the engine's UpdateSemantics; see
// PlatformConfigurationSemanticsClient).
func (v *FlutterView) UpdateSemantics(update SemanticsUpdate) {
	platformConfigurationNativeAPIUpdateSemantics(v.viewID, update)
}

func (v *FlutterView) String() string { return "FlutterView(id: " + strconv.Itoa(v.viewID) + ")" }

// SingletonFlutterWindow is the deprecated FlutterView of a single-view
// application (Deprecated: use FlutterView or PlatformDispatcher). Its
// metrics are those of the implicit view; its other members forward to its
// PlatformDispatcher.
type SingletonFlutterWindow struct {
	FlutterView
}

// OnMetricsChanged is platformDispatcher.onMetricsChanged.
func (w *SingletonFlutterWindow) OnMetricsChanged() VoidCallback {
	return w.platformDispatcher.OnMetricsChanged()
}

// SetOnMetricsChanged sets platformDispatcher.onMetricsChanged.
func (w *SingletonFlutterWindow) SetOnMetricsChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnMetricsChanged(callback)
}

// Locale is platformDispatcher.locale.
func (w *SingletonFlutterWindow) Locale() Locale { return w.platformDispatcher.Locale() }

// Locales is platformDispatcher.locales.
func (w *SingletonFlutterWindow) Locales() []Locale { return w.platformDispatcher.Locales() }

// ComputePlatformResolvedLocale is
// platformDispatcher.computePlatformResolvedLocale.
func (w *SingletonFlutterWindow) ComputePlatformResolvedLocale(supportedLocales []Locale) *Locale {
	return w.platformDispatcher.ComputePlatformResolvedLocale(supportedLocales)
}

// OnLocaleChanged is platformDispatcher.onLocaleChanged.
func (w *SingletonFlutterWindow) OnLocaleChanged() VoidCallback {
	return w.platformDispatcher.OnLocaleChanged()
}

// SetOnLocaleChanged sets platformDispatcher.onLocaleChanged.
func (w *SingletonFlutterWindow) SetOnLocaleChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnLocaleChanged(callback)
}

// InitialLifecycleState is platformDispatcher.initialLifecycleState.
func (w *SingletonFlutterWindow) InitialLifecycleState() string {
	return w.platformDispatcher.InitialLifecycleState()
}

// TextScaleFactor is platformDispatcher.textScaleFactor.
func (w *SingletonFlutterWindow) TextScaleFactor() float64 {
	return w.platformDispatcher.TextScaleFactor()
}

// NativeSpellCheckServiceDefined is
// platformDispatcher.nativeSpellCheckServiceDefined.
func (w *SingletonFlutterWindow) NativeSpellCheckServiceDefined() bool {
	return w.platformDispatcher.NativeSpellCheckServiceDefined()
}

// SupportsShowingSystemContextMenu is
// platformDispatcher.supportsShowingSystemContextMenu.
func (w *SingletonFlutterWindow) SupportsShowingSystemContextMenu() bool {
	return w.platformDispatcher.SupportsShowingSystemContextMenu()
}

// BrieflyShowPassword is platformDispatcher.brieflyShowPassword.
func (w *SingletonFlutterWindow) BrieflyShowPassword() bool {
	return w.platformDispatcher.BrieflyShowPassword()
}

// AlwaysUse24HourFormat is platformDispatcher.alwaysUse24HourFormat.
func (w *SingletonFlutterWindow) AlwaysUse24HourFormat() bool {
	return w.platformDispatcher.AlwaysUse24HourFormat()
}

// OnTextScaleFactorChanged is platformDispatcher.onTextScaleFactorChanged.
func (w *SingletonFlutterWindow) OnTextScaleFactorChanged() VoidCallback {
	return w.platformDispatcher.OnTextScaleFactorChanged()
}

// SetOnTextScaleFactorChanged sets
// platformDispatcher.onTextScaleFactorChanged.
func (w *SingletonFlutterWindow) SetOnTextScaleFactorChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnTextScaleFactorChanged(callback)
}

// PlatformBrightness is platformDispatcher.platformBrightness.
func (w *SingletonFlutterWindow) PlatformBrightness() Brightness {
	return w.platformDispatcher.PlatformBrightness()
}

// OnPlatformBrightnessChanged is
// platformDispatcher.onPlatformBrightnessChanged.
func (w *SingletonFlutterWindow) OnPlatformBrightnessChanged() VoidCallback {
	return w.platformDispatcher.OnPlatformBrightnessChanged()
}

// SetOnPlatformBrightnessChanged sets
// platformDispatcher.onPlatformBrightnessChanged.
func (w *SingletonFlutterWindow) SetOnPlatformBrightnessChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnPlatformBrightnessChanged(callback)
}

// SystemFontFamily is platformDispatcher.systemFontFamily.
func (w *SingletonFlutterWindow) SystemFontFamily() *string {
	return w.platformDispatcher.SystemFontFamily()
}

// OnSystemFontFamilyChanged is platformDispatcher.onSystemFontFamilyChanged.
func (w *SingletonFlutterWindow) OnSystemFontFamilyChanged() VoidCallback {
	return w.platformDispatcher.OnSystemFontFamilyChanged()
}

// SetOnSystemFontFamilyChanged sets
// platformDispatcher.onSystemFontFamilyChanged.
func (w *SingletonFlutterWindow) SetOnSystemFontFamilyChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnSystemFontFamilyChanged(callback)
}

// OnBeginFrame is platformDispatcher.onBeginFrame.
func (w *SingletonFlutterWindow) OnBeginFrame() FrameCallback {
	return w.platformDispatcher.OnBeginFrame()
}

// SetOnBeginFrame sets platformDispatcher.onBeginFrame.
func (w *SingletonFlutterWindow) SetOnBeginFrame(callback FrameCallback) {
	w.platformDispatcher.SetOnBeginFrame(callback)
}

// OnDrawFrame is platformDispatcher.onDrawFrame.
func (w *SingletonFlutterWindow) OnDrawFrame() VoidCallback {
	return w.platformDispatcher.OnDrawFrame()
}

// SetOnDrawFrame sets platformDispatcher.onDrawFrame.
func (w *SingletonFlutterWindow) SetOnDrawFrame(callback VoidCallback) {
	w.platformDispatcher.SetOnDrawFrame(callback)
}

// OnReportTimings is platformDispatcher.onReportTimings.
func (w *SingletonFlutterWindow) OnReportTimings() TimingsCallback {
	return w.platformDispatcher.OnReportTimings()
}

// SetOnReportTimings sets platformDispatcher.onReportTimings.
func (w *SingletonFlutterWindow) SetOnReportTimings(callback TimingsCallback) {
	w.platformDispatcher.SetOnReportTimings(callback)
}

// OnPointerDataPacket is platformDispatcher.onPointerDataPacket.
func (w *SingletonFlutterWindow) OnPointerDataPacket() PointerDataPacketCallback {
	return w.platformDispatcher.OnPointerDataPacket()
}

// SetOnPointerDataPacket sets platformDispatcher.onPointerDataPacket.
func (w *SingletonFlutterWindow) SetOnPointerDataPacket(callback PointerDataPacketCallback) {
	w.platformDispatcher.SetOnPointerDataPacket(callback)
}

// OnKeyData is platformDispatcher.onKeyData.
func (w *SingletonFlutterWindow) OnKeyData() KeyDataCallback { return w.platformDispatcher.OnKeyData() }

// SetOnKeyData sets platformDispatcher.onKeyData.
func (w *SingletonFlutterWindow) SetOnKeyData(callback KeyDataCallback) {
	w.platformDispatcher.SetOnKeyData(callback)
}

// DefaultRouteName is platformDispatcher.defaultRouteName.
func (w *SingletonFlutterWindow) DefaultRouteName() string {
	return w.platformDispatcher.DefaultRouteName()
}

// ScheduleFrame is platformDispatcher.scheduleFrame.
func (w *SingletonFlutterWindow) ScheduleFrame() { w.platformDispatcher.ScheduleFrame() }

// SemanticsEnabled is platformDispatcher.semanticsEnabled.
func (w *SingletonFlutterWindow) SemanticsEnabled() bool {
	return w.platformDispatcher.SemanticsEnabled()
}

// OnSemanticsEnabledChanged is platformDispatcher.onSemanticsEnabledChanged.
func (w *SingletonFlutterWindow) OnSemanticsEnabledChanged() VoidCallback {
	return w.platformDispatcher.OnSemanticsEnabledChanged()
}

// SetOnSemanticsEnabledChanged sets
// platformDispatcher.onSemanticsEnabledChanged.
func (w *SingletonFlutterWindow) SetOnSemanticsEnabledChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnSemanticsEnabledChanged(callback)
}

// FrameData is platformDispatcher.frameData.
func (w *SingletonFlutterWindow) FrameData() FrameData { return w.platformDispatcher.FrameData() }

// OnFrameDataChanged is platformDispatcher.onFrameDataChanged.
func (w *SingletonFlutterWindow) OnFrameDataChanged() VoidCallback {
	return w.platformDispatcher.OnFrameDataChanged()
}

// SetOnFrameDataChanged sets platformDispatcher.onFrameDataChanged.
func (w *SingletonFlutterWindow) SetOnFrameDataChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnFrameDataChanged(callback)
}

// AccessibilityFeatures is platformDispatcher.accessibilityFeatures.
func (w *SingletonFlutterWindow) AccessibilityFeatures() AccessibilityFeatures {
	return w.platformDispatcher.AccessibilityFeatures()
}

// OnAccessibilityFeaturesChanged is
// platformDispatcher.onAccessibilityFeaturesChanged.
func (w *SingletonFlutterWindow) OnAccessibilityFeaturesChanged() VoidCallback {
	return w.platformDispatcher.OnAccessibilityFeaturesChanged()
}

// SetOnAccessibilityFeaturesChanged sets
// platformDispatcher.onAccessibilityFeaturesChanged.
func (w *SingletonFlutterWindow) SetOnAccessibilityFeaturesChanged(callback VoidCallback) {
	w.platformDispatcher.SetOnAccessibilityFeaturesChanged(callback)
}

// SendPlatformMessage is platformDispatcher.sendPlatformMessage.
func (w *SingletonFlutterWindow) SendPlatformMessage(name string, data []byte, callback PlatformMessageResponseCallback) {
	w.platformDispatcher.SendPlatformMessage(name, data, callback)
}

// OnPlatformMessage is platformDispatcher.onPlatformMessage (deprecated).
func (w *SingletonFlutterWindow) OnPlatformMessage() PlatformMessageCallback {
	return w.platformDispatcher.OnPlatformMessage()
}

// SetOnPlatformMessage sets platformDispatcher.onPlatformMessage
// (deprecated).
func (w *SingletonFlutterWindow) SetOnPlatformMessage(callback PlatformMessageCallback) {
	w.platformDispatcher.SetOnPlatformMessage(callback)
}

// SetIsolateDebugName is PlatformDispatcher.instance.setIsolateDebugName.
func (w *SingletonFlutterWindow) SetIsolateDebugName(name string) {
	PlatformDispatcherInstance().SetIsolateDebugName(name)
}

// Brightness describes the contrast of a theme or color palette.
type Brightness int

const (
	// BrightnessDark: the color is dark and needs a light text color.
	BrightnessDark Brightness = iota
	// BrightnessLight: the color is light and needs a dark text color.
	BrightnessLight
)

// BrightnessValues is Brightness.values.
var BrightnessValues = []Brightness{BrightnessDark, BrightnessLight}

// Name is the enum value's name ("dark").
func (b Brightness) Name() string { return [...]string{"dark", "light"}[b] }

func (b Brightness) String() string { return "Brightness." + b.Name() }

// Window is dart:ui's top-level window (Deprecated: look up the FlutterView
// or use PlatformDispatcher.instance): the SingletonFlutterWindow of the
// implicit view.
var Window = &SingletonFlutterWindow{FlutterView{viewID: kImplicitViewID,
	platformDispatcher: PlatformDispatcherInstance(), configuration: newViewConfiguration(), singleton: true}}
