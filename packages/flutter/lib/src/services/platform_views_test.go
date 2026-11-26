package services_test

// Port of packages/flutter/test/services/platform_views_test.dart (Flutter
// 3.50.1), all 31 tests, on flutter_test's port of fake_platform_views.dart.
// The two "motion event converter" testWidgets pump no widgets; they run as
// plain async tests. Dart's unorderedEquals on the fakes' views is an
// ordered comparison here (the fakes keep creation order, and every
// expectation lists the views in that order).

import (
	"errors"
	"slices"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
)

func sizep(w, h float64) *ui.Size { s := ui.NewSize(w, h); return &s }
func boolp(b bool) *bool          { return &b }
func intp(i int) *int             { return &i }

func createOpts(w, h float64) services.PlatformViewControllerCreateOptions {
	return services.PlatformViewControllerCreateOptions{Size: sizep(w, h)}
}

func isPlatformException(err error) bool {
	var p *services.PlatformException
	return err != nil && errors.As(err, &p)
}

func expectViews(got, want []flutter_test.FakeAndroidPlatformView) {
	expect(len(got) == len(want), "views %v, want %v", got, want)
	for i := range got {
		expect(got[i].Equal(want[i]), "views %v, want %v", got, want)
	}
}

// androidTest is the Android group: setUp creates the fake platform side.
func androidTest(t *testing.T, body func(views *flutter_test.FakeAndroidPlatformViewsController)) {
	ensureInitialized()
	runAsync(t, func() {
		views := flutter_test.NewFakeAndroidPlatformViewsController()
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, nil)
		body(views)
	})
}

var pvs = services.PlatformViewsService

func TestPlatformViewsAndroidCreateOfUnregisteredType(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		err := awaitError(pvs.InitAndroidView(0, "web", ui.TextDirectionLtr).Create(createOpts(100, 100)))
		expect(isPlatformException(err), "error %v", err)
		views.RegisterViewType("web")
		expect(awaitError(pvs.InitSurfaceAndroidView(0, "web", ui.TextDirectionLtr).Create(createOpts(1, 1))) == nil, "surface")
		expect(awaitError(pvs.InitAndroidView(1, "web", ui.TextDirectionLtr).Create(createOpts(1, 1))) == nil, "texture")
	})
}

func TestPlatformViewsAndroidCreateVDFallbackViews(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr).Create(createOpts(100, 100)).Await()
		pvs.InitAndroidView(1, "webview", ui.TextDirectionRtl).Create(createOpts(200, 300)).Await()
		expectViews(views.Views(), []flutter_test.FakeAndroidPlatformView{
			{ID: 0, Type: "webview", Size: sizep(100, 100), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionLtr},
			{ID: 1, Type: "webview", Size: sizep(200, 300), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionRtl},
		})
	})
}

func TestPlatformViewsAndroidCreateHCFallbackViews(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitSurfaceAndroidView(0, "webview", ui.TextDirectionLtr).Create(createOpts(100, 100)).Await()
		pvs.InitSurfaceAndroidView(1, "webview", ui.TextDirectionRtl).Create(createOpts(200, 300)).Await()
		expectViews(views.Views(), []flutter_test.FakeAndroidPlatformView{
			{ID: 0, Type: "webview", Size: sizep(100, 100), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionLtr, HybridFallback: boolp(true)},
			{ID: 1, Type: "webview", Size: sizep(200, 300), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionRtl, HybridFallback: boolp(true)},
		})
	})
}

func TestPlatformViewsAndroidCreateHCOnlyViews(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitExpensiveAndroidView(0, "webview", ui.TextDirectionLtr).Create(createOpts(100, 100)).Await()
		pvs.InitExpensiveAndroidView(1, "webview", ui.TextDirectionRtl).Create(createOpts(200, 300)).Await()
		expectViews(views.Views(), []flutter_test.FakeAndroidPlatformView{
			{ID: 0, Type: "webview", LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionLtr, Hybrid: boolp(true)},
			{ID: 1, Type: "webview", LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionRtl, Hybrid: boolp(true)},
		})
	})
}

func TestPlatformViewsAndroidViewComposition(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fallback bool
		init     func() services.AndroidViewController
		want     bool
	}{
		{"default view does not use view composition by default", false, func() services.AndroidViewController {
			return pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr)
		}, false},
		{"default view does not use view composition in fallback mode", true, func() services.AndroidViewController {
			return pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr)
		}, false},
		{"surface view does not use view composition by default", false, func() services.AndroidViewController {
			return pvs.InitSurfaceAndroidView(0, "webview", ui.TextDirectionLtr)
		}, false},
		{"surface view does uses view composition in fallback mode", true, func() services.AndroidViewController {
			return pvs.InitSurfaceAndroidView(0, "webview", ui.TextDirectionLtr)
		}, true},
		{"expensive view uses view composition", false, func() services.AndroidViewController {
			return pvs.InitExpensiveAndroidView(0, "webview", ui.TextDirectionLtr)
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
				views.RegisterViewType("webview")
				views.AllowTextureLayerMode = !tc.fallback
				controller := tc.init()
				controller.Create(createOpts(100, 100)).Await()
				views.AllowTextureLayerMode = true
				expect(controller.RequiresViewComposition() == tc.want, "requiresViewComposition %v", !tc.want)
			})
		})
	}
}

func TestPlatformViewsAndroidReuseViewID(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr).Create(createOpts(100, 100)).Await()
		err := awaitError(pvs.InitAndroidView(0, "web", ui.TextDirectionLtr).Create(createOpts(100, 100)))
		expect(isPlatformException(err), "error %v", err)
	})
}

func TestPlatformViewsAndroidDispose(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr).Create(createOpts(100, 100)).Await()
		viewController := pvs.InitAndroidView(1, "webview", ui.TextDirectionLtr)
		viewController.Create(createOpts(200, 300)).Await()
		viewController.Dispose().Await()
		surfaceViewController := pvs.InitSurfaceAndroidView(1, "webview", ui.TextDirectionLtr)
		surfaceViewController.Create(createOpts(200, 300)).Await()
		surfaceViewController.Dispose().Await()
		expectViews(views.Views(), []flutter_test.FakeAndroidPlatformView{
			{ID: 0, Type: "webview", Size: sizep(100, 100), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionLtr},
		})
	})
}

func TestPlatformViewsAndroidDisposeTwice(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		viewController := pvs.InitAndroidView(1, "webview", ui.TextDirectionLtr)
		viewController.Create(createOpts(200, 300)).Await()
		viewController.Dispose().Await()
		viewController.Dispose().Await()
	})
}

func TestPlatformViewsAndroidDisposeClearsFocusCallbacks(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		didFocus := false
		views.RegisterViewType("webview")
		viewController := pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr,
			services.PlatformViewsServiceInitOptions{OnFocus: func() { didFocus = true }})
		viewController.Create(createOpts(100, 100)).Await()
		viewController.Dispose().Await()
		message := services.SystemChannels.PlatformViews.Codec.EncodeMethodCall(services.MethodCall{Method: "viewFocused", Arguments: 0})
		messenger().HandlePlatformMessage(services.SystemChannels.PlatformViews.Name, message, func([]byte) {}).Await()
		expect(!didFocus, "focused after dispose")
	})
}

// Not in Flutter's test file: the focus callback runs before dispose.
func TestPlatformViewsAndroidViewFocused(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		focused := 0
		views.RegisterViewType("webview")
		viewController := pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr,
			services.PlatformViewsServiceInitOptions{OnFocus: func() { focused++ }})
		viewController.Create(createOpts(100, 100)).Await()
		views.InvokeViewFocused(0)
		async.FutureDelayed[any](0, nil).Await()
		expect(focused == 1, "focused %d", focused)
		viewController.Dispose().Await()
	})
}

func TestPlatformViewsAndroidResize(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr).Create(createOpts(100, 100)).Await()
		androidView := pvs.InitAndroidView(1, "webview", ui.TextDirectionLtr)
		androidView.Create(createOpts(200, 300)).Await()
		got := androidView.SetSize(ui.NewSize(500, 500)).Await()
		expect(got == ui.NewSize(500, 500), "setSize %v", got)
		expectViews(views.Views(), []flutter_test.FakeAndroidPlatformView{
			{ID: 0, Type: "webview", Size: sizep(100, 100), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionLtr},
			{ID: 1, Type: "webview", Size: sizep(500, 500), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionLtr},
		})
	})
}

func TestPlatformViewsAndroidOnPlatformViewCreatedCallback(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		var createdViews []int
		callback := func(id int) { createdViews = append(createdViews, id) }
		controller1 := pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr)
		controller1.AddOnPlatformViewCreatedListener(callback)
		expect(len(createdViews) == 0, "created %v", createdViews)
		controller1.Create(createOpts(100, 100)).Await()
		expect(slices.Equal(createdViews, []int{0}), "created %v", createdViews)
		controller2 := pvs.InitAndroidView(5, "webview", ui.TextDirectionLtr)
		controller2.AddOnPlatformViewCreatedListener(callback)
		expect(slices.Equal(createdViews, []int{0}), "created %v", createdViews)
		controller2.Create(createOpts(100, 200)).Await()
		expect(slices.Equal(createdViews, []int{0, 5}), "created %v", createdViews)
		controller3 := pvs.InitAndroidView(10, "webview", ui.TextDirectionLtr)
		controller3.AddOnPlatformViewCreatedListener(callback)
		expect(slices.Equal(createdViews, []int{0, 5}), "created %v", createdViews)
		create := controller3.Create(createOpts(100, 200))
		dispose := controller3.Dispose()
		create.Await()
		dispose.Await()
		expect(slices.Equal(createdViews, []int{0, 5}), "created %v", createdViews)
	})
}

func TestPlatformViewsAndroidDirectionality(t *testing.T) {
	for _, tc := range []struct {
		name      string
		from, to  ui.TextDirection
		direction int
	}{
		{"change Android view's directionality before creation", ui.TextDirectionRtl, ui.TextDirectionLtr, services.AndroidViewControllerKAndroidLayoutDirectionLtr},
		{"change Android view's directionality after creation", ui.TextDirectionLtr, ui.TextDirectionRtl, services.AndroidViewControllerKAndroidLayoutDirectionRtl},
	} {
		t.Run(tc.name, func(t *testing.T) {
			androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
				views.RegisterViewType("webview")
				viewController := pvs.InitAndroidView(0, "webview", tc.from)
				viewController.SetLayoutDirection(tc.to).Await()
				viewController.Create(createOpts(100, 100)).Await()
				expectViews(views.Views(), []flutter_test.FakeAndroidPlatformView{
					{ID: 0, Type: "webview", Size: sizep(100, 100), LayoutDirection: tc.direction},
				})
			})
		})
	}
}

// Not in Flutter's test file: a direction change after creation is sent.
func TestPlatformViewsAndroidDirectionalityChangeAfterCreate(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		viewController := pvs.InitAndroidView(0, "webview", ui.TextDirectionLtr)
		viewController.Create(createOpts(100, 100)).Await()
		viewController.SetLayoutDirection(ui.TextDirectionRtl).Await()
		expectViews(views.Views(), []flutter_test.FakeAndroidPlatformView{
			{ID: 0, Type: "webview", Size: sizep(100, 100), LayoutDirection: services.AndroidViewControllerKAndroidLayoutDirectionRtl},
		})
	})
}

func TestPlatformViewsAndroidSetOffsetIfCreated(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		viewController := pvs.InitAndroidView(7, "webview", ui.TextDirectionLtr)
		viewController.Create(createOpts(100, 100)).Await()
		viewController.SetOffset(ui.NewOffset(10, 20)).Await()
		expect(len(views.Offsets) == 1 && views.Offsets[7] == ui.NewOffset(10, 20), "offsets %v", views.Offsets)
	})
}

func TestPlatformViewsAndroidNoOffsetIfNotCreated(t *testing.T) {
	androidTest(t, func(views *flutter_test.FakeAndroidPlatformViewsController) {
		views.RegisterViewType("webview")
		viewController := pvs.InitAndroidView(7, "webview", ui.TextDirectionLtr)
		viewController.SetOffset(ui.NewOffset(10, 20)).Await()
		expect(len(views.Offsets) == 0, "offsets %v", views.Offsets)
	})
}

func logPlatformViews(channel *services.MethodChannel) *[]services.MethodCall {
	var log []services.MethodCall
	messenger().SetMockMethodCallHandler(channel, func(call services.MethodCall) async.Future[any] {
		log = append(log, call)
		return async.FutureValue[any](nil)
	})
	return &log
}

func TestPlatformViewsMotionEventConverterDoesNotDuplicateMoveEvents(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		log := logPlatformViews(services.SystemChannels.PlatformViews)
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, nil)
		viewController := pvs.InitSurfaceAndroidView(7, "web", ui.TextDirectionLtr)
		viewController.SetPointTransformer(func(offset ui.Offset) ui.Offset { return offset })
		const pointerCount = 10
		for i := range pointerCount {
			viewController.DispatchPointerEvent(gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{
				TimeStamp: time.Millisecond, Pointer: i})).Await()
		}
		const kPointerDataFlagMultiple = 2
		for i := range pointerCount {
			viewController.DispatchPointerEvent(gestures.NewPointerMoveEvent(gestures.PointerMoveEventOptions{
				TimeStamp: 2 * time.Millisecond, Pointer: i, PlatformData: kPointerDataFlagMultiple | (pointerCount << 8)})).Await()
		}
		var moveCalls []services.MethodCall
		for _, call := range *log {
			args := call.Arguments.([]any)
			if call.Method == "touch" && args[3] == services.AndroidViewControllerKActionMove {
				moveCalls = append(moveCalls, call)
			}
		}
		expect(len(moveCalls) == 1, "move calls %d", len(moveCalls))
		expect(moveCalls[0].Arguments.([]any)[4] == pointerCount, "pointer count %v", moveCalls[0].Arguments.([]any)[4])
	})
}

func xOf(call services.MethodCall) []float64 {
	var xs []float64
	for _, coords := range call.Arguments.([]any)[6].([]any) {
		xs = append(xs, coords.([]any)[7].(float64))
	}
	return xs
}

func singleTouch(log []services.MethodCall) services.MethodCall {
	var touches []services.MethodCall
	for _, c := range log {
		if c.Method == "touch" {
			touches = append(touches, c)
		}
	}
	expect(len(touches) == 1, "touch calls %v", touches)
	return touches[0]
}

func TestPlatformViewsMotionEventConverterOrdersPointersByAndroidPointerID(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		log := logPlatformViews(services.SystemChannels.PlatformViews)
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, nil)
		viewController := pvs.InitSurfaceAndroidView(7, "web", ui.TextDirectionLtr)
		viewController.SetPointTransformer(func(offset ui.Offset) ui.Offset { return offset })
		down := func(pointer int, x float64) {
			viewController.DispatchPointerEvent(gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{
				Pointer: pointer, Position: ui.NewOffset(x, x)})).Await()
		}
		down(1, 10)
		down(2, 20)
		viewController.DispatchPointerEvent(gestures.NewPointerUpEvent(gestures.PointerUpEventOptions{
			Pointer: 1, Position: ui.NewOffset(10, 10)})).Await()
		*log = nil
		down(3, 30)
		downCall := singleTouch(*log)
		downArgs := downCall.Arguments.([]any)
		expect(downArgs[4] == 2, "pointer count %v", downArgs[4])
		expect(downArgs[3] == services.AndroidViewControllerPointerAction(0, services.AndroidViewControllerKActionPointerDown), "action %v", downArgs[3])
		props := downArgs[5].([]any)
		expect(len(props) == 2 && slices.Equal(props[0].([]any), []any{0, 1}) && slices.Equal(props[1].([]any), []any{1, 1}),
			"properties %v", props)
		expect(slices.Equal(xOf(downCall), []float64{30, 20}), "x %v", xOf(downCall))
		const platformData = 2 | (2 << 8)
		*log = nil
		viewController.DispatchPointerEvent(gestures.NewPointerMoveEvent(gestures.PointerMoveEventOptions{
			Pointer: 3, Position: ui.NewOffset(31, 31), PlatformData: platformData})).Await()
		viewController.DispatchPointerEvent(gestures.NewPointerMoveEvent(gestures.PointerMoveEventOptions{
			Pointer: 2, Position: ui.NewOffset(21, 21), PlatformData: platformData})).Await()
		moveCall := singleTouch(*log)
		expect(moveCall.Arguments.([]any)[3] == services.AndroidViewControllerKActionMove, "action")
		expect(slices.Equal(xOf(moveCall), []float64{31, 21}), "x %v", xOf(moveCall))
	})
}

func iosTest(t *testing.T, body func(views *flutter_test.FakeIosPlatformViewsController)) {
	ensureInitialized()
	runAsync(t, func() {
		views := flutter_test.NewFakeIosPlatformViewsController()
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, nil)
		body(views)
	})
}

func expectUiKitViews(got, want []flutter_test.FakeUiKitView) {
	expect(len(got) == len(want), "views %v, want %v", got, want)
	for i := range got {
		expect(got[i].Equal(want[i]), "views %v, want %v", got, want)
	}
}

func TestPlatformViewsIOSCreateOfUnregisteredType(t *testing.T) {
	iosTest(t, func(*flutter_test.FakeIosPlatformViewsController) {
		err := awaitError(pvs.InitUiKitView(0, "web", ui.TextDirectionLtr))
		expect(isPlatformException(err), "error %v", err)
	})
}

func TestPlatformViewsIOSCreate(t *testing.T) {
	iosTest(t, func(views *flutter_test.FakeIosPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitUiKitView(0, "webview", ui.TextDirectionLtr).Await()
		pvs.InitUiKitView(1, "webview", ui.TextDirectionRtl).Await()
		expectUiKitViews(views.Views(), []flutter_test.FakeUiKitView{{ID: 0, Type: "webview"}, {ID: 1, Type: "webview"}})
	})
}

func TestPlatformViewsIOSReuseViewID(t *testing.T) {
	iosTest(t, func(views *flutter_test.FakeIosPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitUiKitView(0, "webview", ui.TextDirectionLtr).Await()
		err := awaitError(pvs.InitUiKitView(0, "web", ui.TextDirectionLtr))
		expect(isPlatformException(err), "error %v", err)
	})
}

func TestPlatformViewsIOSDispose(t *testing.T) {
	iosTest(t, func(views *flutter_test.FakeIosPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitUiKitView(0, "webview", ui.TextDirectionLtr).Await()
		viewController := pvs.InitUiKitView(1, "webview", ui.TextDirectionLtr).Await()
		viewController.Dispose().Await()
		expectUiKitViews(views.Views(), []flutter_test.FakeUiKitView{{ID: 0, Type: "webview"}})
	})
}

func TestPlatformViewsIOSDisposeInexisting(t *testing.T) {
	iosTest(t, func(views *flutter_test.FakeIosPlatformViewsController) {
		views.RegisterViewType("webview")
		pvs.InitUiKitView(0, "webview", ui.TextDirectionLtr).Await()
		viewController := pvs.InitUiKitView(1, "webview", ui.TextDirectionLtr).Await()
		viewController.Dispose().Await()
		err := awaitError(viewController.Dispose())
		expect(isPlatformException(err), "error %v", err)
	})
}

// Not in Flutter's test file: gestures and creation params on macOS.
func TestPlatformViewsMacOSGesturesAndParams(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		views := flutter_test.NewFakeMacosPlatformViewsController()
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.PlatformViews, nil)
		views.RegisterViewType("map")
		controller := pvs.InitAppKitView(3, "map", ui.TextDirectionLtr, services.PlatformViewsServiceInitOptions{
			CreationParams: "zoom", CreationParamsCodec: services.NewStandardMessageCodec()}).Await()
		controller.AcceptGesture().Await()
		controller.AcceptGesture().Await()
		controller.RejectGesture().Await()
		expect(views.GesturesAccepted[3] == 2 && views.GesturesRejected[3] == 1, "gestures %v %v", views.GesturesAccepted, views.GesturesRejected)
		got := views.Views()
		expect(len(got) == 1 && slices.Equal(got[0].CreationParams, services.NewStandardMessageCodec().EncodeMessage("zoom")), "views %v", got)
		controller.Dispose().Await()
		expect(len(views.Views()) == 0, "views %v", views.Views())
	})
}

func TestPlatformViewsToString(t *testing.T) {
	if got := (services.AndroidPointerProperties{ID: 0, ToolType: 0}).String(); got != "AndroidPointerProperties(id: 0, toolType: 0)" {
		t.Error(got)
	}
	if got := (services.AndroidPointerCoords{}).String(); got != "AndroidPointerCoords(orientation: 0.0, pressure: 0.0, size: 0.0, toolMajor: 0.0, toolMinor: 0.0, touchMajor: 0.0, touchMinor: 0.0, x: 0.0, y: 0.0)" {
		t.Error(got)
	}
	event := &services.AndroidMotionEvent{PointerProperties: []services.AndroidPointerProperties{}, PointerCoords: []services.AndroidPointerCoords{}}
	if got := event.String(); got != "AndroidPointerEvent(downTime: 0, eventTime: 0, action: 0, pointerCount: 0, pointerProperties: [], pointerCoords: [], metaState: 0, buttonState: 0, xPrecision: 0.0, yPrecision: 0.0, deviceId: 0, edgeFlags: 0, source: 0, flags: 0, motionEventId: 0)" {
		t.Error(got)
	}
}

func TestPlatformViewsRejectGesture(t *testing.T) {
	for _, tc := range []struct {
		name    string
		channel *services.MethodChannel
		init    func() services.AndroidViewController
		withID  bool
	}{
		{"HybridAndroidViewController.rejectGesture invokes rejectGesture on platform_views_2", services.SystemChannels.PlatformViews2,
			func() services.AndroidViewController {
				return pvs.InitHybridAndroidView(42, "test_view", ui.TextDirectionLtr)
			}, true},
		{"ExpensiveAndroidViewController.rejectGesture invokes rejectGesture on platform_views", services.SystemChannels.PlatformViews,
			func() services.AndroidViewController {
				return pvs.InitExpensiveAndroidView(43, "test_view", ui.TextDirectionLtr)
			}, false},
		{"TextureAndroidViewController.rejectGesture invokes rejectGesture on platform_views", services.SystemChannels.PlatformViews,
			func() services.AndroidViewController {
				return pvs.InitAndroidView(44, "test_view", ui.TextDirectionLtr)
			}, false},
		{"SurfaceAndroidViewController.rejectGesture invokes rejectGesture on platform_views", services.SystemChannels.PlatformViews,
			func() services.AndroidViewController {
				return pvs.InitSurfaceAndroidView(45, "test_view", ui.TextDirectionLtr)
			}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ensureInitialized()
			runAsync(t, func() {
				calls := logPlatformViews(tc.channel)
				defer messenger().SetMockMethodCallHandler(tc.channel, nil)
				controller := tc.init()
				controller.RejectGesture().Await()
				expect(len(*calls) == 1 && (*calls)[0].Method == "rejectGesture", "calls %v", *calls)
				args := (*calls)[0].Arguments.(map[any]any)
				expect(len(args) == 1 && args["id"] == controller.ViewID(), "args %v", args)
				if tc.withID {
					*calls = nil
					controller.RejectGesture(services.PlatformViewControllerRejectGestureOptions{GestureID: intp(999)}).Await()
					args := (*calls)[0].Arguments.(map[any]any)
					expect(len(*calls) == 1 && len(args) == 2 && args["id"] == 42 && args["gestureId"] == 999, "args %v", args)
				}
			})
		})
	}
}

func TestPlatformViewsRegistry(t *testing.T) {
	r := services.PlatformViewsRegistryInstance()
	a := r.GetNextPlatformViewID()
	if b := r.GetNextPlatformViewID(); b != a+1 {
		t.Fatalf("ids %d %d", a, b)
	}
}
