package difftest

// Go twin of flutter/test/system_channels_calls_test.dart: the platform
// messages SystemChrome, SystemNavigator, SystemSound, HapticFeedback,
// BrowserContextMenu, DeferredComponent and SensitiveContentService send
// (exact bytes), their results and their timing, compared line by line.

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func joinBytes(data []byte) string {
	if data == nil {
		return "null"
	}
	s := make([]string, len(data))
	for i, b := range data {
		s[i] = strconv.Itoa(int(b))
	}
	return strings.Join(s, ",")
}

// dartErrorLine is the Dart side's `${e.runtimeType}: $e` (and
// AssertionError(message)) for the errors these scenes produce.
func dartErrorLine(err error) string {
	var assertion *dart.AssertionError
	if errors.As(err, &assertion) {
		return "AssertionError(" + dart.ToString(assertion.AssertionMessage()) + ")"
	}
	var pe *services.PlatformException
	if errors.As(err, &pe) {
		return "PlatformException: " + pe.Error()
	}
	var ue dart.UnsupportedError
	if errors.As(err, &ue) {
		return "UnsupportedError: " + ue.Error()
	}
	return "?: " + err.Error()
}

// runScenes runs body on a fresh loop until it completes and compares the
// log with the reference's.
func runLoggedScenes(t *testing.T, refFile string, body func(add func(string))) {
	data, err := os.ReadFile(refFile)
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	if !isolated(t) { // it constructs its own binding
		return
	}
	var ref struct{ Log []string }
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	newChannelsBinding()
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	flutter_test.SetCurrentTest(t)
	var log []string
	add := func(s string) { log = append(log, s) }
	done := false
	var failure error
	async.OnComplete(async.AsyncVoid(func() { body(add) }), func(async.Void) { done = true }, func(err error) { failure = err; done = true })
	for i := 0; !done && i < 100000; i++ {
		loop.FlushMicrotasks()
		if done || loop.RunPosted() > 0 {
			continue
		}
		if d, ok := loop.NextDeadline(); ok {
			loop.ElapseTo(d)
		} else {
			break
		}
	}
	if failure != nil {
		t.Fatalf("scenes failed: %v", failure)
	}
	if !done {
		t.Fatal("scenes never completed")
	}
	for i := range max(len(log), len(ref.Log)) {
		var got, want string
		if i < len(log) {
			got = log[i]
		}
		if i < len(ref.Log) {
			want = ref.Log[i]
		}
		if got != want {
			t.Fatalf("line %d differs:\n Go:      %q\n Flutter: %q\ncontext (Go): %q", i, got, want, log[max(0, i-5):min(len(log), i+3)])
		}
	}
	t.Logf("%d lines identical", len(log))
}

func TestSystemChannelsCallsMatchFlutter(t *testing.T) {
	runLoggedScenes(t, "flutter/build/system_channels_calls.json", func(add func(string)) {
		messenger := func() *flutter_test.TestDefaultBinaryMessenger {
			return flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
		}
		ticks := func(n int) {
			i := 0
			var step func()
			step = func() {
				add("tick " + strconv.Itoa(i))
				i++
				if i < n {
					async.ScheduleMicrotask(step)
				}
			}
			async.ScheduleMicrotask(step)
		}
		replies := map[string]func([]byte) []byte{}
		logChannel := func(name string, isJSON bool) {
			messenger().SetMockMessageHandler(name, func(message []byte) async.Future[[]byte] {
				return async.Async(func() []byte {
					if isJSON {
						add(name + " " + string(message))
					} else {
						add(name + " " + joinBytes(message))
					}
					if r := replies[name]; r != nil {
						return r(message)
					}
					return nil
				})
			})
		}
		step := func(name string, f func() async.Future[any]) {
			var err error
			var result any
			func() {
				defer func() {
					if r := recover(); r != nil {
						if e, ok := r.(error); ok {
							err = e
						} else {
							err = errors.New(dart.ToString(r))
						}
					}
				}()
				result = f().Await()
			}()
			if err != nil {
				add(name + " !! " + dartErrorLine(err))
				return
			}
			add(name + " -> " + dart.ToString(result))
		}
		void := func(f func() async.Future[async.Void]) func() async.Future[any] {
			return func() async.Future[any] {
				return async.Then(f(), func(async.Void) any { return nil })
			}
		}
		yield := func() { async.FutureDelayed(0, func() any { return nil }).Await() }

		logChannel("flutter/platform", true)
		logChannel("flutter/navigation", true)
		logChannel("flutter/deferredcomponent", false)
		logChannel("flutter/sensitivecontent", false)
		logChannel("flutter/contextmenu", true)
		chrome := services.SystemChrome

		add("scene orientations")
		step("none", void(func() async.Future[async.Void] {
			return chrome.SetPreferredOrientations([]services.DeviceOrientation{})
		}))
		step("up", void(func() async.Future[async.Void] {
			return chrome.SetPreferredOrientations([]services.DeviceOrientation{services.DeviceOrientationPortraitUp})
		}))
		step("all", void(func() async.Future[async.Void] {
			return chrome.SetPreferredOrientations(services.DeviceOrientationValues)
		}))

		add("scene switcher")
		label, color := `Example "label"`, 0xFF00FF00
		step("both", void(func() async.Future[async.Void] {
			return chrome.SetApplicationSwitcherDescription(services.ApplicationSwitcherDescription{Label: &label, PrimaryColor: &color})
		}))
		step("none", void(func() async.Future[async.Void] {
			return chrome.SetApplicationSwitcherDescription(services.ApplicationSwitcherDescription{})
		}))
		unicode := "ünïcødé ✓"
		step("label", void(func() async.Future[async.Void] {
			return chrome.SetApplicationSwitcherDescription(services.ApplicationSwitcherDescription{Label: &unicode})
		}))
		minusOne := -1
		step("color", void(func() async.Future[async.Void] {
			return chrome.SetApplicationSwitcherDescription(services.ApplicationSwitcherDescription{PrimaryColor: &minusOne})
		}))

		add("scene ui mode")
		for _, mode := range services.SystemUiModeValues {
			if mode != services.SystemUiModeManual {
				step(mode.String(), void(func() async.Future[async.Void] { return chrome.SetEnabledSystemUIMode(mode) }))
			}
		}
		manual := func(overlays []services.SystemUiOverlay) func() async.Future[any] {
			return void(func() async.Future[async.Void] {
				return chrome.SetEnabledSystemUIMode(services.SystemUiModeManual, services.SetEnabledSystemUIModeOptions{Overlays: overlays})
			})
		}
		step("manual none", manual([]services.SystemUiOverlay{}))
		step("manual top", manual([]services.SystemUiOverlay{services.SystemUiOverlayTop}))
		step("manual both", manual([]services.SystemUiOverlay{services.SystemUiOverlayBottom, services.SystemUiOverlayTop}))
		step("manual null", manual(nil))
		step("restore", void(chrome.RestoreSystemUIOverlays))

		add("scene ui change callback")
		step("null callback", void(func() async.Future[async.Void] { return chrome.SetSystemUIChangeCallback(nil) }))
		step("callback", void(func() async.Future[async.Void] {
			return chrome.SetSystemUIChangeCallback(func(visible bool) async.Future[async.Void] {
				return async.AsyncVoid(func() { add("systemUIChange " + strconv.FormatBool(visible)) })
			})
		}))
		for _, visible := range []bool{true, false} {
			messenger().HandlePlatformMessage("flutter/platform",
				services.JSONMethodCodec{}.EncodeMethodCall(services.NewMethodCall("SystemChrome.systemUIChange", []any{visible})),
				func(reply []byte) { add("systemUIChange reply " + joinBytes(reply)) }).Await()
		}
		step("clear callback", void(func() async.Future[async.Void] { return chrome.SetSystemUIChangeCallback(nil) }))

		add("scene overlay style")
		// Make sure no style is recorded as sent.
		chrome.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached)
		yield()
		ticks(4)
		chrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleDark)
		add("latest " + strconv.FormatBool(chrome.LatestStyle().Equal(services.SystemUiOverlayStyleDark)))
		chrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleLight)
		add("set twice")
		yield()
		add("latest " + strconv.FormatBool(chrome.LatestStyle().Equal(services.SystemUiOverlayStyleLight)))
		ticks(3)
		chrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleLight)
		add("same again")
		yield()
		colorp := func(v int) *ui.Color { c := ui.NewColor(v); return &c }
		brightp := func(b ui.Brightness) *ui.Brightness { return &b }
		boolp := func(b bool) *bool { return &b }
		custom := services.NewSystemUiOverlayStyle(services.SystemUiOverlayStyleOptions{
			SystemNavigationBarColor:            colorp(0x80123456),
			SystemNavigationBarDividerColor:     colorp(0x00000000),
			SystemNavigationBarIconBrightness:   brightp(ui.BrightnessDark),
			SystemNavigationBarContrastEnforced: boolp(false),
			StatusBarColor:                      colorp(0xFFFFFFFF),
			StatusBarBrightness:                 brightp(ui.BrightnessLight),
			StatusBarIconBrightness:             brightp(ui.BrightnessDark),
			SystemStatusBarContrastEnforced:     boolp(true),
		})
		copied := func() *services.SystemUiOverlayStyle {
			return custom.CopyWith(services.SystemUiOverlayStyleOptions{StatusBarColor: colorp(0xFF000001)})
		}
		chrome.SetSystemUIOverlayStyle(custom)
		chrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleLight)
		chrome.SetSystemUIOverlayStyle(custom)
		yield()
		chrome.SetSystemUIOverlayStyle(copied())
		yield()
		chrome.SetSystemUIOverlayStyle(copied())
		yield()
		add("detached")
		chrome.HandleAppLifecycleStateChanged(ui.AppLifecycleStateDetached)
		chrome.SetSystemUIOverlayStyle(copied())
		yield()
		chrome.SetSystemUIOverlayStyle(copied())
		yield()
		add("resumed")
		chrome.HandleAppLifecycleStateChanged(ui.AppLifecycleStateResumed)
		chrome.SetSystemUIOverlayStyle(services.NewSystemUiOverlayStyle())
		yield()

		add("scene navigator")
		nav := services.SystemNavigator
		step("pop", void(func() async.Future[async.Void] { return nav.Pop() }))
		step("pop animated", void(func() async.Future[async.Void] {
			return nav.Pop(services.SystemNavigatorPopOptions{Animated: boolp(true)})
		}))
		step("pop not animated", void(func() async.Future[async.Void] {
			return nav.Pop(services.SystemNavigatorPopOptions{Animated: boolp(false)})
		}))
		for _, platform := range foundation.TargetPlatformValues {
			foundation.SetDebugDefaultTargetPlatformOverride(&platform)
			step("handles back "+platform.String(), void(func() async.Future[async.Void] { return nav.SetFrameworkHandlesBack(true) }))
			step("no back "+platform.String(), void(func() async.Future[async.Void] { return nav.SetFrameworkHandlesBack(false) }))
		}
		foundation.SetDebugDefaultTargetPlatformOverride(nil)
		step("single", void(nav.SelectSingleEntryHistory))
		step("multi", void(nav.SelectMultiEntryHistory))
		route := func(opts services.RouteInformationUpdatedOptions) func() async.Future[any] {
			return void(func() async.Future[async.Void] { return nav.RouteInformationUpdated(opts) })
		}
		strp := func(s string) *string { return &s }
		mustURL := func(s string) *url.URL { u, _ := url.Parse(s); return u }
		step("location", route(services.RouteInformationUpdatedOptions{Location: strp("/a/b")}))
		step("location state", route(services.RouteInformationUpdatedOptions{Location: strp("a"),
			State: []dart.MapEntry[string, any]{{Key: "k", Value: []any{1, 2.5, nil, "x"}}}, Replace: true}))
		step("uri", route(services.RouteInformationUpdatedOptions{URI: mustURL("https://flutter.dev/docs?q=1#frag")}))
		step("uri path", route(services.RouteInformationUpdatedOptions{URI: mustURL("/p/q?x=y"), State: 42}))
		step("both", route(services.RouteInformationUpdatedOptions{Location: strp("a"), URI: mustURL("b")}))

		add("scene sounds")
		for _, typ := range services.SystemSoundTypeValues {
			step(typ.String(), void(func() async.Future[async.Void] { return services.SystemSound.Play(typ) }))
		}

		add("scene haptics")
		haptics := services.HapticFeedback
		step("vibrate", void(haptics.Vibrate))
		step("light", void(haptics.LightImpact))
		step("medium", void(haptics.MediumImpact))
		step("heavy", void(haptics.HeavyImpact))
		step("selection", void(haptics.SelectionClick))
		step("success", void(haptics.SuccessNotification))
		step("warning", void(haptics.WarningNotification))
		step("error", void(haptics.ErrorNotification))

		add("scene context menu")
		step("disable", void(services.BrowserContextMenu.DisableContextMenu))
		step("enable", void(services.BrowserContextMenu.EnableContextMenu))
		add("enabled " + strconv.FormatBool(services.BrowserContextMenu.Enabled()))

		add("scene deferred components")
		step("install", void(func() async.Future[async.Void] { return services.DeferredComponent.InstallDeferredComponent("comp") }))
		step("uninstall", void(func() async.Future[async.Void] { return services.DeferredComponent.UninstallDeferredComponent("ünï") }))

		add("scene sensitive content")
		service := services.NewSensitiveContentService()
		for _, sensitivity := range services.ContentSensitivityValues[:3] {
			step("set "+sensitivity.String(), void(func() async.Future[async.Void] { return service.SetContentSensitivity(sensitivity) }))
		}
		for i := 0; i < 4; i++ {
			replies["flutter/sensitivecontent"] = func([]byte) []byte { return services.NewStandardMethodCodec().EncodeSuccessEnvelope(i) }
			step("get "+strconv.Itoa(i), func() async.Future[any] {
				return async.EraseFuture(service.GetContentSensitivity())
			})
		}
		replies["flutter/sensitivecontent"] = func([]byte) []byte { return services.NewStandardMethodCodec().EncodeSuccessEnvelope(true) }
		android, iOS := foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS
		foundation.SetDebugDefaultTargetPlatformOverride(&android)
		step("supported android", func() async.Future[any] { return async.EraseFuture(service.IsSupported()) })
		foundation.SetDebugDefaultTargetPlatformOverride(&iOS)
		step("supported iOS", func() async.Future[any] { return async.EraseFuture(service.IsSupported()) })
		foundation.SetDebugDefaultTargetPlatformOverride(nil)
		delete(replies, "flutter/sensitivecontent")

		add("scene replies")
		message := "no haptics"
		replies["flutter/platform"] = func([]byte) []byte {
			return services.JSONMethodCodec{}.EncodeErrorEnvelope("denied", services.EncodeErrorEnvelopeOptions{Message: &message})
		}
		step("haptic error", void(haptics.Vibrate))
		step("sound error", void(func() async.Future[async.Void] { return services.SystemSound.Play(services.SystemSoundTypeAlert) }))
		replies["flutter/platform"] = func([]byte) []byte {
			return services.JSONMethodCodec{}.EncodeSuccessEnvelope(map[string]any{"ok": 1})
		}
		step("chrome value", void(chrome.RestoreSystemUIOverlays))
		delete(replies, "flutter/platform")
	})
}
