package services

// Port of system_chrome.dart (Flutter 3.50.1).

import (
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// DeviceOrientation specifies a particular device orientation.
type DeviceOrientation int

const (
	// DeviceOrientationPortraitUp: the device is upright.
	DeviceOrientationPortraitUp DeviceOrientation = iota
	// DeviceOrientationLandscapeLeft: rotated 90 degrees clockwise from
	// portraitUp.
	DeviceOrientationLandscapeLeft
	// DeviceOrientationPortraitDown: upside down.
	DeviceOrientationPortraitDown
	// DeviceOrientationLandscapeRight: rotated 90 degrees
	// counter-clockwise from portraitUp.
	DeviceOrientationLandscapeRight
)

// DeviceOrientationValues is DeviceOrientation.values.
var DeviceOrientationValues = []DeviceOrientation{DeviceOrientationPortraitUp, DeviceOrientationLandscapeLeft,
	DeviceOrientationPortraitDown, DeviceOrientationLandscapeRight}

// Name is the enum value's Dart name.
func (o DeviceOrientation) Name() string {
	return [...]string{"portraitUp", "landscapeLeft", "portraitDown", "landscapeRight"}[o]
}

func (o DeviceOrientation) String() string { return "DeviceOrientation." + o.Name() }

// ApplicationSwitcherDescription specifies a description of the
// application that is pertinent to the embedder's application switcher
// (also known as "recent tasks") user interface. A struct literal; nil is
// Dart's null.
type ApplicationSwitcherDescription struct {
	// Label is a label and description of the current state of the
	// application.
	Label *string
	// PrimaryColor is the application's primary color (an ARGB int).
	PrimaryColor *int
}

// SystemUiOverlay specifies a system overlay at a particular location.
type SystemUiOverlay int

const (
	// SystemUiOverlayTop: the status bar.
	SystemUiOverlayTop SystemUiOverlay = iota
	// SystemUiOverlayBottom: the navigation bar.
	SystemUiOverlayBottom
)

// SystemUiOverlayValues is SystemUiOverlay.values.
var SystemUiOverlayValues = []SystemUiOverlay{SystemUiOverlayTop, SystemUiOverlayBottom}

// Name is the enum value's Dart name.
func (o SystemUiOverlay) Name() string { return [...]string{"top", "bottom"}[o] }

func (o SystemUiOverlay) String() string { return "SystemUiOverlay." + o.Name() }

// SystemUiMode describes different display configurations for both
// Android and iOS.
type SystemUiMode int

const (
	// SystemUiModeLeanBack: fullscreen; a tap anywhere restores the
	// overlays.
	SystemUiModeLeanBack SystemUiMode = iota
	// SystemUiModeImmersive: fullscreen; a swipe from the edge restores
	// the overlays.
	SystemUiModeImmersive
	// SystemUiModeImmersiveSticky: like immersive, but the app receives
	// the swipe.
	SystemUiModeImmersiveSticky
	// SystemUiModeEdgeToEdge: the overlays are shown over the app.
	SystemUiModeEdgeToEdge
	// SystemUiModeManual: the overlays given are shown.
	SystemUiModeManual
)

// SystemUiModeValues is SystemUiMode.values.
var SystemUiModeValues = []SystemUiMode{SystemUiModeLeanBack, SystemUiModeImmersive, SystemUiModeImmersiveSticky,
	SystemUiModeEdgeToEdge, SystemUiModeManual}

// Name is the enum value's Dart name.
func (m SystemUiMode) Name() string {
	return [...]string{"leanBack", "immersive", "immersiveSticky", "edgeToEdge", "manual"}[m]
}

func (m SystemUiMode) String() string { return "SystemUiMode." + m.Name() }

// SystemUiOverlayStyle specifies a preference for the style of the system
// overlays. It is immutable and Diagnosticable: it is used by pointer,
// built with NewSystemUiOverlayStyle, and compared with Equal (Dart's ==;
// nil fields are Dart's nulls).
type SystemUiOverlayStyle struct {
	foundation.DiagnosticableMixin
	// SystemNavigationBarColor is the color of the system bottom
	// navigation bar (Android O+).
	SystemNavigationBarColor *ui.Color
	// SystemNavigationBarDividerColor is the color of the divider between
	// the system's bottom navigation bar and the app's content (Android
	// P+).
	SystemNavigationBarDividerColor *ui.Color
	// SystemNavigationBarIconBrightness is the brightness of the system
	// navigation bar icons (Android O+).
	SystemNavigationBarIconBrightness *ui.Brightness
	// SystemNavigationBarContrastEnforced: whether the system should
	// ensure enough contrast for the navigation bar (Android Q+).
	SystemNavigationBarContrastEnforced *bool
	// StatusBarColor is the color of the top status bar (Android M+).
	StatusBarColor *ui.Color
	// StatusBarBrightness is the brightness of the top status bar (iOS).
	StatusBarBrightness *ui.Brightness
	// StatusBarIconBrightness is the brightness of the top status bar
	// icons (Android M+).
	StatusBarIconBrightness *ui.Brightness
	// SystemStatusBarContrastEnforced: whether the system should ensure
	// enough contrast for the status bar (Android Q+).
	SystemStatusBarContrastEnforced *bool
}

// SystemUiOverlayStyleOptions are SystemUiOverlayStyle's named
// parameters (nil: null).
type SystemUiOverlayStyleOptions struct {
	SystemNavigationBarColor            *ui.Color
	SystemNavigationBarDividerColor     *ui.Color
	SystemNavigationBarIconBrightness   *ui.Brightness
	SystemNavigationBarContrastEnforced *bool
	StatusBarColor                      *ui.Color
	StatusBarBrightness                 *ui.Brightness
	StatusBarIconBrightness             *ui.Brightness
	SystemStatusBarContrastEnforced     *bool
}

// NewSystemUiOverlayStyle is SystemUiOverlayStyle({...}).
func NewSystemUiOverlayStyle(opts ...SystemUiOverlayStyleOptions) *SystemUiOverlayStyle {
	var o SystemUiOverlayStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s := &SystemUiOverlayStyle{
		SystemNavigationBarColor:            o.SystemNavigationBarColor,
		SystemNavigationBarDividerColor:     o.SystemNavigationBarDividerColor,
		SystemNavigationBarIconBrightness:   o.SystemNavigationBarIconBrightness,
		SystemNavigationBarContrastEnforced: o.SystemNavigationBarContrastEnforced,
		StatusBarColor:                      o.StatusBarColor,
		StatusBarBrightness:                 o.StatusBarBrightness,
		StatusBarIconBrightness:             o.StatusBarIconBrightness,
		SystemStatusBarContrastEnforced:     o.SystemStatusBarContrastEnforced,
	}
	s.SetSelf(s)
	return s
}

func brightnessPtr(b ui.Brightness) *ui.Brightness { return &b }
func colorPtr(c ui.Color) *ui.Color                { return &c }

// SystemUiOverlayStyleLight is SystemUiOverlayStyle.light: system overlays
// should be drawn with a light color (intended for dark backgrounds).
var SystemUiOverlayStyleLight = NewSystemUiOverlayStyle(SystemUiOverlayStyleOptions{
	SystemNavigationBarColor:          colorPtr(ui.NewColor(0xFF000000)),
	SystemNavigationBarIconBrightness: brightnessPtr(ui.BrightnessLight),
	StatusBarIconBrightness:           brightnessPtr(ui.BrightnessLight),
	StatusBarBrightness:               brightnessPtr(ui.BrightnessDark),
})

// SystemUiOverlayStyleDark is SystemUiOverlayStyle.dark: system overlays
// should be drawn with a dark color (intended for light backgrounds).
var SystemUiOverlayStyleDark = NewSystemUiOverlayStyle(SystemUiOverlayStyleOptions{
	SystemNavigationBarColor:          colorPtr(ui.NewColor(0xFF000000)),
	SystemNavigationBarIconBrightness: brightnessPtr(ui.BrightnessLight),
	StatusBarIconBrightness:           brightnessPtr(ui.BrightnessDark),
	StatusBarBrightness:               brightnessPtr(ui.BrightnessLight),
})

func colorValueOrNull(c *ui.Color) any {
	if c == nil {
		return nil
	}
	return c.Value()
}

func brightnessStringOrNull(b *ui.Brightness) any {
	if b == nil {
		return nil
	}
	return b.String()
}

func boolOrNull(b *bool) any {
	if b == nil {
		return nil
	}
	return *b
}

// toMap is _toMap: the style as the platform receives it (an ordered map,
// as Dart's literal).
func (s *SystemUiOverlayStyle) toMap() []dart.MapEntry[string, any] {
	return []dart.MapEntry[string, any]{
		{Key: "systemNavigationBarColor", Value: colorValueOrNull(s.SystemNavigationBarColor)},
		{Key: "systemNavigationBarDividerColor", Value: colorValueOrNull(s.SystemNavigationBarDividerColor)},
		{Key: "systemStatusBarContrastEnforced", Value: boolOrNull(s.SystemStatusBarContrastEnforced)},
		{Key: "statusBarColor", Value: colorValueOrNull(s.StatusBarColor)},
		{Key: "statusBarBrightness", Value: brightnessStringOrNull(s.StatusBarBrightness)},
		{Key: "statusBarIconBrightness", Value: brightnessStringOrNull(s.StatusBarIconBrightness)},
		{Key: "systemNavigationBarIconBrightness", Value: brightnessStringOrNull(s.SystemNavigationBarIconBrightness)},
		{Key: "systemNavigationBarContrastEnforced", Value: boolOrNull(s.SystemNavigationBarContrastEnforced)},
	}
}

// CopyWith creates a copy of this theme with the given fields replaced
// with new values (nil: keep).
func (s *SystemUiOverlayStyle) CopyWith(opts ...SystemUiOverlayStyleOptions) *SystemUiOverlayStyle {
	var o SystemUiOverlayStyleOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	pick := func(a, b *ui.Color) *ui.Color {
		if a != nil {
			return a
		}
		return b
	}
	pickB := func(a, b *ui.Brightness) *ui.Brightness {
		if a != nil {
			return a
		}
		return b
	}
	pickBool := func(a, b *bool) *bool {
		if a != nil {
			return a
		}
		return b
	}
	return NewSystemUiOverlayStyle(SystemUiOverlayStyleOptions{
		SystemNavigationBarColor:            pick(o.SystemNavigationBarColor, s.SystemNavigationBarColor),
		SystemNavigationBarDividerColor:     pick(o.SystemNavigationBarDividerColor, s.SystemNavigationBarDividerColor),
		SystemNavigationBarContrastEnforced: pickBool(o.SystemNavigationBarContrastEnforced, s.SystemNavigationBarContrastEnforced),
		StatusBarColor:                      pick(o.StatusBarColor, s.StatusBarColor),
		StatusBarIconBrightness:             pickB(o.StatusBarIconBrightness, s.StatusBarIconBrightness),
		StatusBarBrightness:                 pickB(o.StatusBarBrightness, s.StatusBarBrightness),
		SystemStatusBarContrastEnforced:     pickBool(o.SystemStatusBarContrastEnforced, s.SystemStatusBarContrastEnforced),
		SystemNavigationBarIconBrightness:   pickB(o.SystemNavigationBarIconBrightness, s.SystemNavigationBarIconBrightness),
	})
}

func ptrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Equal is SystemUiOverlayStyle's ==.
func (s *SystemUiOverlayStyle) Equal(other *SystemUiOverlayStyle) bool {
	if s == nil || other == nil {
		return s == other
	}
	return ptrEqual(other.SystemNavigationBarColor, s.SystemNavigationBarColor) &&
		ptrEqual(other.SystemNavigationBarDividerColor, s.SystemNavigationBarDividerColor) &&
		ptrEqual(other.SystemNavigationBarContrastEnforced, s.SystemNavigationBarContrastEnforced) &&
		ptrEqual(other.StatusBarColor, s.StatusBarColor) &&
		ptrEqual(other.StatusBarIconBrightness, s.StatusBarIconBrightness) &&
		ptrEqual(other.StatusBarBrightness, s.StatusBarBrightness) &&
		ptrEqual(other.SystemStatusBarContrastEnforced, s.SystemStatusBarContrastEnforced) &&
		ptrEqual(other.SystemNavigationBarIconBrightness, s.SystemNavigationBarIconBrightness)
}

func nullableProperty[T any](name string, v *T) foundation.DiagnosticsNode {
	if v == nil {
		return foundation.NewDiagnosticsProperty[any](name, nil)
	}
	if b, ok := any(*v).(ui.Brightness); ok {
		// ui.Brightness's String() is already Dart's toString
		// ("Brightness.dark") but it has no Name(), which diagnostics
		// expects of such enums: describe it by its String().
		return foundation.NewDiagnosticsProperty(name, describedBrightness{b})
	}
	return foundation.NewDiagnosticsProperty(name, *v)
}

// describedBrightness is a Brightness diagnostics print as Dart does.
type describedBrightness struct{ ui.Brightness }

// DebugFillProperties adds the style's eight properties.
func (s *SystemUiOverlayStyle) DebugFillProperties(properties *foundation.DiagnosticPropertiesBuilder) {
	s.DiagnosticableMixin.DebugFillProperties(properties)
	properties.Add(nullableProperty("systemNavigationBarColor", s.SystemNavigationBarColor))
	properties.Add(nullableProperty("systemNavigationBarDividerColor", s.SystemNavigationBarDividerColor))
	properties.Add(nullableProperty("systemNavigationBarIconBrightness", s.SystemNavigationBarIconBrightness))
	properties.Add(nullableProperty("systemNavigationBarContrastEnforced", s.SystemNavigationBarContrastEnforced))
	properties.Add(nullableProperty("statusBarColor", s.StatusBarColor))
	properties.Add(nullableProperty("statusBarBrightness", s.StatusBarBrightness))
	properties.Add(nullableProperty("statusBarIconBrightness", s.StatusBarIconBrightness))
	properties.Add(nullableProperty("systemStatusBarContrastEnforced", s.SystemStatusBarContrastEnforced))
}

func stringify[T interface{ String() string }](list []T) []any {
	out := make([]any, len(list))
	for i, item := range list {
		out[i] = item.String()
	}
	return out
}

type systemChrome struct {
	pendingStyle *SystemUiOverlayStyle
	latestStyle  *SystemUiOverlayStyle
}

// SystemChrome controls specific aspects of the operating system's
// graphical interface and how it interacts with the application (a class
// of static methods; its private static state lives in this variable).
var SystemChrome = &systemChrome{}

// SetPreferredOrientations specifies the set of orientations the
// application interface can be displayed in (empty: the platform
// default).
func (*systemChrome) SetPreferredOrientations(orientations []DeviceOrientation) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("SystemChrome.setPreferredOrientations", stringify(orientations)).Await()
	})
}

// SetApplicationSwitcherDescription specifies the description of the
// current state of the application as it pertains to the application
// switcher.
func (*systemChrome) SetApplicationSwitcherDescription(description ApplicationSwitcherDescription) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("SystemChrome.setApplicationSwitcherDescription", []dart.MapEntry[string, any]{
			{Key: "label", Value: description.Label}, {Key: "primaryColor", Value: description.PrimaryColor},
		}).Await()
	})
}

// SetEnabledSystemUIModeOptions are setEnabledSystemUIMode's named
// parameters.
type SetEnabledSystemUIModeOptions struct {
	// Overlays are the overlays shown in SystemUiModeManual (nil: null).
	Overlays []SystemUiOverlay
}

// SetEnabledSystemUIMode specifies the SystemUiMode to have visible when
// the application is running; SystemUiModeManual requires Overlays.
func (*systemChrome) SetEnabledSystemUIMode(mode SystemUiMode, opts ...SetEnabledSystemUIModeOptions) async.Future[async.Void] {
	var o SetEnabledSystemUIModeOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return async.AsyncVoid(func() {
		if mode != SystemUiModeManual {
			SystemChannels.Platform.InvokeMethod("SystemChrome.setEnabledSystemUIMode", mode.String()).Await()
		} else {
			if !(mode == SystemUiModeManual && o.Overlays != nil) {
				// assert(mode == SystemUiMode.manual && overlays != null)
				panic(dart.NewAssertionError(nil))
			}
			SystemChannels.Platform.InvokeMethod("SystemChrome.setEnabledSystemUIOverlays", stringify(o.Overlays)).Await()
		}
	})
}

// SetSystemUIChangeCallback sets the callback method for responding to
// changes in the system UI (nil: none).
func (*systemChrome) SetSystemUIChangeCallback(callback SystemUiChangeCallback) async.Future[async.Void] {
	return async.AsyncVoid(func() {
		ServicesBindingInstance().SetSystemUiChangeCallback(callback)
		// Skip setting up the listener if there is no callback.
		if callback != nil {
			SystemChannels.Platform.InvokeMethod("SystemChrome.setSystemUIChangeListener").Await()
		}
	})
}

// RestoreSystemUIOverlays restores the system overlays to the last
// settings provided via SetEnabledSystemUIMode.
func (*systemChrome) RestoreSystemUIOverlays() async.Future[async.Void] {
	return async.AsyncVoid(func() {
		SystemChannels.Platform.InvokeMethod("SystemChrome.restoreSystemUIOverlays").Await()
	})
}

// SetSystemUIOverlayStyle specifies the style to use for the system
// overlays; calls in the same task are coalesced (the last style is sent
// in a microtask, and only if it differs from the last one sent).
func (c *systemChrome) SetSystemUIOverlayStyle(style *SystemUiOverlayStyle) {
	if c.pendingStyle != nil {
		// The microtask has already been queued; just update the pending
		// value.
		c.pendingStyle = style
		return
	}
	if style.Equal(c.latestStyle) {
		// Trivial success: no microtask has been queued and the given style
		// is already in effect, so no need to queue a microtask.
		return
	}
	c.pendingStyle = style
	async.ScheduleMicrotask(func() {
		if c.pendingStyle == nil {
			panic(dart.NewAssertionError(nil))
		}
		if !c.pendingStyle.Equal(c.latestStyle) {
			async.OnComplete(SystemChannels.Platform.InvokeMethod("SystemChrome.setSystemUIOverlayStyle", c.pendingStyle.toMap()),
				func(any) {},
				func(err error) {
					library := "services library"
					foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(err, foundation.FlutterErrorDetailsOptions{
						Library: &library,
						Context: foundation.NewErrorDescription("while setting the system UI overlay style"),
					}))
				})
			c.latestStyle = c.pendingStyle
		}
		c.pendingStyle = nil
	})
}

// HandleAppLifecycleStateChanged is called by the binding when the app's
// lifecycle state changes: when the app is detached, the record of the
// style sent to the host is cleared (in a microtask) so that it will be
// sent again when the app is reattached.
func (c *systemChrome) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	if state == ui.AppLifecycleStateDetached {
		async.ScheduleMicrotask(func() {
			c.latestStyle = nil
		})
	}
}

// LatestStyle is the last style sent to the host (@visibleForTesting);
// nil when none.
func (c *systemChrome) LatestStyle() *SystemUiOverlayStyle { return c.latestStyle }
