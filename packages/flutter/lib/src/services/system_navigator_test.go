package services_test

// Ports of packages/flutter/test/services/system_navigator_test.dart (2/2),
// system_sound_test.dart (1/1), haptic_feedback_test.dart (2/2),
// deferred_component_test.dart (2/2) and browser_context_menu_test.dart
// (2/2 of the "not on web" group; the "on web" group is skipped by Flutter
// itself off the web, and Go is never a web build) (Flutter 3.50.1).

import (
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// verify runs test and expects exactly the given calls (method and
// arguments) in log.
func verify(log *[]services.MethodCall, test func() async.Future[async.Void], expectations ...services.MethodCall) {
	*log = nil
	test().Await()
	expect(len(*log) == len(expectations), "log %v, want %v", *log, expectations)
	for i, e := range expectations {
		expect(isMethodCall((*log)[i], e.Method, e.Arguments), "call %v, want %v", (*log)[i], e)
	}
}

func TestSystemNavigatorPlatformMessages(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		verify(log, func() async.Future[async.Void] { return services.SystemNavigator.Pop() },
			services.NewMethodCall("SystemNavigator.pop", nil))
		animated := true
		verify(log, func() async.Future[async.Void] {
			return services.SystemNavigator.Pop(services.SystemNavigatorPopOptions{Animated: &animated})
		}, services.NewMethodCall("SystemNavigator.pop", true))
	})
}

func TestSystemNavigatorSetFrameworkHandlesBack(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	defer foundation.SetDebugDefaultTargetPlatformOverride(nil)
	runAsync(t, func() {
		for _, p := range foundation.TargetPlatformValues {
			foundation.SetDebugDefaultTargetPlatformOverride(&p)
			*log = nil
			services.SystemNavigator.SetFrameworkHandlesBack(true).Await()
			if p == foundation.TargetPlatformAndroid {
				expect(len(*log) == 1 && isMethodCall((*log)[0], "SystemNavigator.setFrameworkHandlesBack", true), "%v: %v", p, *log)
			} else {
				expect(len(*log) == 0, "%v: %v", p, *log)
			}
		}
	})
}

func TestSystemNavigatorNavigationMessages(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Navigation)
	defer done()
	a := "a"
	runAsync(t, func() {
		verify(log, services.SystemNavigator.SelectSingleEntryHistory, services.NewMethodCall("selectSingleEntryHistory", nil))
		verify(log, services.SystemNavigator.SelectMultiEntryHistory, services.NewMethodCall("selectMultiEntryHistory", nil))
		verify(log, func() async.Future[async.Void] {
			return services.SystemNavigator.RouteInformationUpdated(services.RouteInformationUpdatedOptions{Location: &a})
		}, services.NewMethodCall("routeInformationUpdated", map[string]any{"uri": "a", "state": nil, "replace": false}))
		verify(log, func() async.Future[async.Void] {
			return services.SystemNavigator.RouteInformationUpdated(services.RouteInformationUpdatedOptions{Location: &a, State: true})
		}, services.NewMethodCall("routeInformationUpdated", map[string]any{"uri": "a", "state": true, "replace": false}))
		verify(log, func() async.Future[async.Void] {
			return services.SystemNavigator.RouteInformationUpdated(services.RouteInformationUpdatedOptions{Location: &a, State: true, Replace: true})
		}, services.NewMethodCall("routeInformationUpdated", map[string]any{"uri": "a", "state": true, "replace": true}))
	})
}

func TestSystemNavigatorRouteInformationUpdatedNeedsOneOfUriAndLocation(t *testing.T) {
	defer func() {
		if _, ok := recover().(*dart.AssertionError); !ok {
			t.Fatal("no AssertionError")
		}
	}()
	services.SystemNavigator.RouteInformationUpdated(services.RouteInformationUpdatedOptions{})
}

func TestSystemSoundControl(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		services.SystemSound.Play(services.SystemSoundTypeClick).Await()
		expect(len(*log) == 1 && isMethodCall((*log)[0], "SystemSound.play", "SystemSoundType.click"), "log %v", *log)
	})
}

func TestHapticFeedbackControl(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		services.HapticFeedback.Vibrate().Await()
		expect(len(*log) == 1 && isMethodCall((*log)[0], "HapticFeedback.vibrate", nil), "log %v", *log)
	})
}

func TestHapticFeedbackVariations(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		check := func(hapticFunction func() async.Future[async.Void], argument string) {
			*log = nil
			hapticFunction().Await()
			expect(len(*log) == 1 && isMethodCall((*log)[0], "HapticFeedback.vibrate", argument), "log %v", *log)
		}
		check(services.HapticFeedback.LightImpact, "HapticFeedbackType.lightImpact")
		check(services.HapticFeedback.MediumImpact, "HapticFeedbackType.mediumImpact")
		check(services.HapticFeedback.HeavyImpact, "HapticFeedbackType.heavyImpact")
		check(services.HapticFeedback.SelectionClick, "HapticFeedbackType.selectionClick")
		check(services.HapticFeedback.SuccessNotification, "HapticFeedbackType.successNotification")
		check(services.HapticFeedback.WarningNotification, "HapticFeedbackType.warningNotification")
		check(services.HapticFeedback.ErrorNotification, "HapticFeedbackType.errorNotification")
	})
}

func TestDeferredComponentInstallAndUninstall(t *testing.T) {
	ensureInitialized()
	log, done := logMethodCalls(services.SystemChannels.DeferredComponent)
	defer done()
	runAsync(t, func() {
		verify(log, func() async.Future[async.Void] {
			return services.DeferredComponent.InstallDeferredComponent("testComponentName")
		}, services.NewMethodCall("installDeferredComponent", map[any]any{"loadingUnitId": -1, "componentName": "testComponentName"}))
		verify(log, func() async.Future[async.Void] {
			return services.DeferredComponent.UninstallDeferredComponent("testComponentName")
		}, services.NewMethodCall("uninstallDeferredComponent", map[any]any{"loadingUnitId": -1, "componentName": "testComponentName"}))
	})
}

func TestBrowserContextMenuAssertsOffTheWeb(t *testing.T) {
	for _, f := range []func() async.Future[async.Void]{
		services.BrowserContextMenu.DisableContextMenu, services.BrowserContextMenu.EnableContextMenu,
	} {
		func() {
			defer func() {
				if _, ok := recover().(*dart.AssertionError); !ok {
					t.Fatal("no AssertionError")
				}
			}()
			f()
		}()
	}
	if !services.BrowserContextMenu.Enabled() {
		t.Fatal("disabled")
	}
}

// Own test (no upstream test file): SensitiveContentService's calls and
// results.
func TestSensitiveContentService(t *testing.T) {
	ensureInitialized()
	var reply any
	var calls []services.MethodCall
	messenger().SetMockMethodCallHandler(services.SystemChannels.SensitiveContent, func(call services.MethodCall) async.Future[any] {
		calls = append(calls, call)
		return async.FutureValue(reply)
	})
	defer messenger().SetMockMethodCallHandler(services.SystemChannels.SensitiveContent, nil)
	defer foundation.SetDebugDefaultTargetPlatformOverride(nil)
	runAsync(t, func() {
		service := services.NewSensitiveContentService()
		service.SetContentSensitivity(services.ContentSensitivitySensitive).Await()
		expect(len(calls) == 1 && isMethodCall(calls[0], "SensitiveContent.setContentSensitivity", 1), "calls %v", calls)
		reply = 2
		expect(service.GetContentSensitivity().Await() == services.ContentSensitivityNotSensitive, "get")
		reply = 3
		_, unsupported := awaitError(service.GetContentSensitivity()).(dart.UnsupportedError)
		expect(unsupported, "unknown mode")
		reply = true
		ios := foundation.TargetPlatformIOS
		foundation.SetDebugDefaultTargetPlatformOverride(&ios)
		n := len(calls)
		expect(!service.IsSupported().Await() && len(calls) == n, "iOS")
		android := foundation.TargetPlatformAndroid
		foundation.SetDebugDefaultTargetPlatformOverride(&android)
		expect(service.IsSupported().Await() && len(calls) == n+1, "android")
		expect(services.ContentSensitivityAutoSensitive.String() == "ContentSensitivity.autoSensitive", "name")
	})
}
