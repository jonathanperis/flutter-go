package services_test

// Port of packages/flutter/test/services/system_chrome_test.dart (Flutter
// 3.50.1). All 13 tests are ported. Adapted: hashCode checks are Equal
// checks (Go has no hashCode); `tester.binding.microtaskCount` is the
// current loop's MicrotaskCount; the manual-mode assertion is checked by
// type (a Go AssertionError has no source text to match).

import (
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func colorp(v int) *ui.Color                 { c := ui.NewColor(v); return &c }
func brightp(b ui.Brightness) *ui.Brightness { return &b }
func boolp(b bool) *bool                     { return &b }

func TestSystemChromeOverlayStyle(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		// Start with the overlay style set to dark mode.
		services.SystemChrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleDark)
		idle()
		*log = nil

		// The first call is a cache miss and will queue a microtask.
		services.SystemChrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleLight)
		expect(async.Current().MicrotaskCount() == 1, "microtasks %d", async.Current().MicrotaskCount())

		// Flush all microtasks.
		idle()
		expect(len(*log) == 1, "log %v", *log)
		expect(isMethodCall((*log)[0], "SystemChrome.setSystemUIOverlayStyle", map[string]any{
			"systemNavigationBarColor":            4278190080,
			"systemNavigationBarDividerColor":     nil,
			"systemStatusBarContrastEnforced":     nil,
			"statusBarColor":                      nil,
			"statusBarBrightness":                 "Brightness.dark",
			"statusBarIconBrightness":             "Brightness.light",
			"systemNavigationBarIconBrightness":   "Brightness.light",
			"systemNavigationBarContrastEnforced": nil,
		}), "call %v", (*log)[0])
		*log = nil
		expect(async.Current().MicrotaskCount() == 0, "microtasks")
		expect(len(*log) == 0, "log")

		// The second call with the same value should be a no-op.
		services.SystemChrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleLight)
		expect(async.Current().MicrotaskCount() == 0, "microtasks")

		services.SystemChrome.SetSystemUIOverlayStyle(services.NewSystemUiOverlayStyle(services.SystemUiOverlayStyleOptions{
			SystemStatusBarContrastEnforced:     boolp(false),
			SystemNavigationBarContrastEnforced: boolp(true),
		}))
		expect(async.Current().MicrotaskCount() == 1, "microtasks")
		idle()
		expect(len(*log) == 1, "log %v", *log)
		expect(isMethodCall((*log)[0], "SystemChrome.setSystemUIOverlayStyle", map[string]any{
			"systemNavigationBarColor":            nil,
			"systemNavigationBarDividerColor":     nil,
			"systemStatusBarContrastEnforced":     false,
			"statusBarColor":                      nil,
			"statusBarBrightness":                 nil,
			"statusBarIconBrightness":             nil,
			"systemNavigationBarIconBrightness":   nil,
			"systemNavigationBarContrastEnforced": true,
		}), "call %v", (*log)[0])
	})
}

func TestSystemChromeSetPreferredOrientations(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		services.SystemChrome.SetPreferredOrientations([]services.DeviceOrientation{services.DeviceOrientationPortraitUp}).Await()
		expect(len(*log) == 1, "log")
		expect(isMethodCall((*log)[0], "SystemChrome.setPreferredOrientations", []any{"DeviceOrientation.portraitUp"}), "call %v", (*log)[0])
	})
}

func TestSystemChromeSetApplicationSwitcherDescription(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		label, color := "Example label", 0xFF00FF00
		services.SystemChrome.SetApplicationSwitcherDescription(services.ApplicationSwitcherDescription{Label: &label, PrimaryColor: &color}).Await()
		expect(len(*log) == 1, "log")
		expect(isMethodCall((*log)[0], "SystemChrome.setApplicationSwitcherDescription",
			map[string]any{"label": "Example label", "primaryColor": 4278255360}), "call %v", (*log)[0])
	})
}

func TestSystemChromeSetApplicationSwitcherDescriptionMissingPlugin(t *testing.T) {
	ensureInitialized()
	var log [][]byte
	messenger().SetMockMessageHandler("flutter/platform", func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			log = append(log, message)
			return nil
		})
	})
	defer messenger().SetMockMessageHandler("flutter/platform", nil)
	runAsync(t, func() {
		label, color := "Example label", 0xFF00FF00
		services.SystemChrome.SetApplicationSwitcherDescription(services.ApplicationSwitcherDescription{Label: &label, PrimaryColor: &color}).Await()
		expect(len(log) > 0, "log")
	})
}

func TestSystemChromeSetEnabledSystemUIMode(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		services.SystemChrome.SetEnabledSystemUIMode(services.SystemUiModeLeanBack).Await()
		expect(len(*log) == 1, "log")
		expect(isMethodCall((*log)[0], "SystemChrome.setEnabledSystemUIMode", "SystemUiMode.leanBack"), "call %v", (*log)[0])
	})
}

func TestSystemChromeSetEnabledSystemUIModeAssertsOverlays(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		err := awaitError(services.SystemChrome.SetEnabledSystemUIMode(services.SystemUiModeManual))
		// Dart checks the VM's text of the failed assertion; Go's
		// AssertionError has no source text (a message-less assert).
		_, isAssertion := err.(*dart.AssertionError)
		expect(isAssertion, "error %v", err)
	})
}

func TestSystemChromeSetEnabledSystemUIModeManualOverlays(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		services.SystemChrome.SetEnabledSystemUIMode(services.SystemUiModeManual,
			services.SetEnabledSystemUIModeOptions{Overlays: []services.SystemUiOverlay{services.SystemUiOverlayTop}}).Await()
		expect(len(*log) == 1, "log")
		expect(isMethodCall((*log)[0], "SystemChrome.setEnabledSystemUIOverlays", []any{"SystemUiOverlay.top"}), "call %v", (*log)[0])
	})
}

func TestSystemChromeSetSystemUIChangeCallback(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		services.SystemChrome.SetSystemUIChangeCallback(nil).Await()
		expect(len(*log) == 0, "log")
		services.SystemChrome.SetSystemUIChangeCallback(func(bool) async.Future[async.Void] {
			return async.AsyncVoid(func() {})
		}).Await()
		expect(len(*log) == 1, "log")
		expect(isMethodCall((*log)[0], "SystemChrome.setSystemUIChangeListener", nil), "call %v", (*log)[0])
	})
	services.ServicesBindingInstance().SetSystemUiChangeCallback(nil)
}

func TestSystemUiOverlayStyleToStringDefaultsNull(t *testing.T) {
	result := services.NewSystemUiOverlayStyle().String()
	if !strings.HasPrefix(result, "SystemUiOverlayStyle#") {
		t.Fatal(result)
	}
	for _, s := range []string{"systemNavigationBarColor: null", "systemNavigationBarDividerColor: null",
		"systemStatusBarContrastEnforced: null", "statusBarColor: null", "statusBarBrightness: null",
		"statusBarIconBrightness: null", "systemNavigationBarIconBrightness: null", "systemNavigationBarContrastEnforced: null"} {
		if !strings.Contains(result, s) {
			t.Errorf("%q lacks %q", result, s)
		}
	}
}

func fullStyle() *services.SystemUiOverlayStyle {
	return services.NewSystemUiOverlayStyle(services.SystemUiOverlayStyleOptions{
		SystemNavigationBarColor:            colorp(0xFF123456),
		SystemNavigationBarDividerColor:     colorp(0xFF654321),
		SystemStatusBarContrastEnforced:     boolp(true),
		StatusBarColor:                      colorp(0xFFABCDEF),
		StatusBarBrightness:                 brightp(ui.BrightnessDark),
		StatusBarIconBrightness:             brightp(ui.BrightnessLight),
		SystemNavigationBarIconBrightness:   brightp(ui.BrightnessDark),
		SystemNavigationBarContrastEnforced: boolp(false),
	})
}

func TestSystemUiOverlayStyleToStringValues(t *testing.T) {
	result := fullStyle().String()
	if !strings.HasPrefix(result, "SystemUiOverlayStyle#") {
		t.Fatal(result)
	}
	for _, s := range []string{
		"systemNavigationBarColor: Color(alpha: 1.0000, red: 0.0706, green: 0.2039, blue: 0.3373, colorSpace: ColorSpace.sRGB)",
		"systemNavigationBarDividerColor: Color(alpha: 1.0000, red: 0.3961, green: 0.2627, blue: 0.1294, colorSpace: ColorSpace.sRGB)",
		"systemStatusBarContrastEnforced: true",
		"statusBarColor: Color(alpha: 1.0000, red: 0.6706, green: 0.8039, blue: 0.9373, colorSpace: ColorSpace.sRGB)",
		"statusBarBrightness: Brightness.dark", "statusBarIconBrightness: Brightness.light",
		"systemNavigationBarIconBrightness: Brightness.dark", "systemNavigationBarContrastEnforced: false",
	} {
		if !strings.Contains(result, s) {
			t.Errorf("%q lacks %q", result, s)
		}
	}
}

func TestSystemUiOverlayStyleEquality(t *testing.T) {
	style := func(c int) *services.SystemUiOverlayStyle {
		return services.NewSystemUiOverlayStyle(services.SystemUiOverlayStyleOptions{
			SystemNavigationBarColor: colorp(c), StatusBarBrightness: brightp(ui.BrightnessDark)})
	}
	style1, style2, style3 := style(0xFF123456), style(0xFF123456), style(0xFF654321)
	if !style1.Equal(style2) || style1.Equal(style3) {
		t.Fatal("==")
	}
}

func TestSystemUiOverlayStyleCopyWith(t *testing.T) {
	style1 := services.NewSystemUiOverlayStyle(services.SystemUiOverlayStyleOptions{
		SystemNavigationBarColor: colorp(0xFF123456), StatusBarBrightness: brightp(ui.BrightnessDark),
		SystemStatusBarContrastEnforced: boolp(true)})
	style2 := style1.CopyWith(services.SystemUiOverlayStyleOptions{
		SystemNavigationBarColor: colorp(0xFF654321), StatusBarIconBrightness: brightp(ui.BrightnessLight)})
	if *style2.SystemNavigationBarColor != ui.NewColor(0xFF654321) || *style2.StatusBarBrightness != ui.BrightnessDark ||
		!*style2.SystemStatusBarContrastEnforced || *style2.StatusBarIconBrightness != ui.BrightnessLight ||
		style2.SystemNavigationBarDividerColor != nil {
		t.Fatal(style2)
	}
}

func TestSystemUiOverlayStyleDebugFillProperties(t *testing.T) {
	builder := foundation.NewDiagnosticPropertiesBuilder()
	services.NewSystemUiOverlayStyle(services.SystemUiOverlayStyleOptions{
		SystemNavigationBarColor:            colorp(0xFF123456),
		SystemNavigationBarDividerColor:     colorp(0xFF654321),
		SystemNavigationBarIconBrightness:   brightp(ui.BrightnessLight),
		SystemNavigationBarContrastEnforced: boolp(true),
		StatusBarColor:                      colorp(0xFFABCDEF),
		StatusBarBrightness:                 brightp(ui.BrightnessDark),
		StatusBarIconBrightness:             brightp(ui.BrightnessLight),
		SystemStatusBarContrastEnforced:     boolp(false),
	}).DebugFillProperties(builder)
	var description []string
	for _, node := range builder.Properties {
		if !node.IsFiltered(foundation.DiagnosticLevelInfo) {
			description = append(description, node.String())
		}
	}
	want := []string{
		"systemNavigationBarColor: Color(alpha: 1.0000, red: 0.0706, green: 0.2039, blue: 0.3373, colorSpace: ColorSpace.sRGB)",
		"systemNavigationBarDividerColor: Color(alpha: 1.0000, red: 0.3961, green: 0.2627, blue: 0.1294, colorSpace: ColorSpace.sRGB)",
		"systemNavigationBarIconBrightness: Brightness.light",
		"systemNavigationBarContrastEnforced: true",
		"statusBarColor: Color(alpha: 1.0000, red: 0.6706, green: 0.8039, blue: 0.9373, colorSpace: ColorSpace.sRGB)",
		"statusBarBrightness: Brightness.dark",
		"statusBarIconBrightness: Brightness.light",
		"systemStatusBarContrastEnforced: false",
	}
	if strings.Join(description, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s", strings.Join(description, "\n"))
	}
}

func TestSystemChromeHandlesDetachedLifecycleState(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		systemUiOverlayStyle := services.NewSystemUiOverlayStyle()
		services.SystemChrome.SetSystemUIOverlayStyle(systemUiOverlayStyle)
		idle()
		expect(len(*log) == 1, "log %d", len(*log))

		// Setting the same state should not send another message to the
		// host.
		services.SystemChrome.SetSystemUIOverlayStyle(systemUiOverlayStyle)
		idle()
		expect(len(*log) == 1, "log %d", len(*log))

		// The state should be sent again if the app was detached.
		services.SystemChrome.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached)
		idle()
		services.SystemChrome.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
		services.SystemChrome.SetSystemUIOverlayStyle(systemUiOverlayStyle)
		idle()
		expect(len(*log) == 2, "log %d", len(*log))
	})
}

func TestSystemChromeReportsSetSystemUIOverlayStyleErrors(t *testing.T) {
	ensureInitialized()
	var errs []*foundation.FlutterErrorDetails
	original := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) { errs = append(errs, details) }
	defer func() { foundation.FlutterErrorOnError = original }()
	messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, func(call services.MethodCall) async.Future[any] {
		return async.Async(func() any {
			if call.Method == "SystemChrome.setSystemUIOverlayStyle" {
				panic("Failed to set system UI overlay style")
			}
			return nil
		})
	})
	defer messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
	runAsync(t, func() {
		// Make sure the light style isn't the latest one sent.
		services.SystemChrome.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached)
		idle()
		services.SystemChrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleLight)
		idle()
		expect(len(errs) == 1, "errors %v", errs)
		pe, ok := errs[0].Exception.(*services.PlatformException)
		expect(ok && pe.Message != nil && *pe.Message == "Failed to set system UI overlay style", "exception %v", errs[0].Exception)
		expect(strings.Contains(errs[0].Context.String(), "while setting the system UI overlay style"), "context %v", errs[0].Context)
	})
}
