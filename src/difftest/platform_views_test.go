package difftest

// Go twin of flutter/test/platform_views_test.dart: the same scenes on
// services' PlatformViewsService and controllers, logging the exact bytes
// of every method call and the controllers' states.

import (
	"math"
	"strconv"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	vm "github.com/jonathanperis/flutter-go/src/pub/vector_math"
)

func TestPlatformViewsMatchFlutter(t *testing.T) {
	runChannelScenes(t, "platform_views", platformViewsScenes)
}

func guardValue(f func() any) (v any) {
	defer func() {
		if r := recover(); r != nil {
			v = "threw " + showValue(r)
		}
	}()
	return f()
}

func showSize(s ui.Size) string {
	return "Size(" + dart.DoubleToString(s.Width) + ", " + dart.DoubleToString(s.Height) + ")"
}

func platformViewsScenes(l *channelLog) {
	replies := map[string][]any{}
	codec := services.StandardMethodCodec{}
	installPlatform := func(channel string) {
		testMessenger().SetMockMessageHandler(channel, func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				call := codec.DecodeMethodCall(message)
				l.add(channel + " " + call.Method + " [" + bytesList(message) + "]")
				var reply any
				if q := replies[call.Method]; len(q) > 0 {
					reply, replies[call.Method] = q[0], q[1:]
				}
				if pe, ok := reply.(*services.PlatformException); ok {
					return codec.EncodeErrorEnvelope(pe.Code, services.EncodeErrorEnvelopeOptions{Message: pe.Message})
				}
				return codec.EncodeSuccessEnvelope(reply)
			})
		})
	}
	state := func(c services.AndroidViewController) string {
		texture := guardValue(func() any {
			if id := c.TextureID(); id != nil {
				return *id
			}
			return nil
		})
		return "id " + strconv.Itoa(c.ViewID()) + " created " + strconv.FormatBool(c.IsCreated()) + " awaiting " +
			strconv.FormatBool(c.AwaitingCreation()) + " composition " + strconv.FormatBool(c.RequiresViewComposition()) +
			" texture " + dart.ToString(texture)
	}
	track := l.trackVoid
	pvs := services.PlatformViewsService
	params := services.NewStandardMessageCodec()
	sz := func(w, h float64) services.PlatformViewControllerCreateOptions {
		s := ui.NewSize(w, h)
		return services.PlatformViewControllerCreateOptions{Size: &s}
	}

	installPlatform("flutter/platform_views")
	installPlatform("flutter/platform_views_2")

	l.add("scene registry")
	first := services.PlatformViewsRegistryInstance().GetNextPlatformViewID()
	l.add("next " + strconv.Itoa(services.PlatformViewsRegistryInstance().GetNextPlatformViewID()-first))

	// ---- Texture view: create, resize, offset, direction, focus, dispose.
	l.add("scene texture")
	replies["create"] = []any{7}
	replies["resize"] = []any{[]dart.MapEntry[string, any]{{Key: "width", Value: 250.5}, {Key: "height", Value: 120.0}}}
	texture := pvs.InitAndroidView(1, "webview", ui.TextDirectionRtl, services.PlatformViewsServiceInitOptions{
		CreationParams: []dart.MapEntry[string, any]{
			{Key: "url", Value: "https://flutter.dev"}, {Key: "zoom", Value: 1.5}, {Key: "list", Value: []any{1, 2}},
		},
		CreationParamsCodec: params,
		OnFocus:             func() { l.add("focus 1") },
	})
	var createdLog []int
	onCreated := func(id int) { createdLog = append(createdLog, id) }
	texture.AddOnPlatformViewCreatedListener(onCreated)
	texture.AddOnPlatformViewCreatedListener(func(id int) { l.add("created callback " + strconv.Itoa(id)) })
	l.add(state(texture))
	track("offset before create", texture.SetOffset(ui.NewOffset(1, 2)))
	track("clearFocus before create", texture.ClearFocus())
	track("direction before create", texture.SetLayoutDirection(ui.TextDirectionLtr))
	l.ticks(10)
	track("create without size", texture.Create())
	l.add("sent")
	settle()
	l.add(state(texture))
	l.ticks(12)
	pos := ui.NewOffset(10, 20)
	size := ui.NewSize(200, 100)
	track("create", texture.Create(services.PlatformViewControllerCreateOptions{Size: &size, Position: &pos}))
	l.add(state(texture))
	settle()
	l.add(state(texture))
	l.add("created " + dart.ToString(createdLog))
	texture.RemoveOnPlatformViewCreatedListener(onCreated)
	l.add("callbacks " + strconv.Itoa(len(texture.CreatedCallbacks())))
	l.ticks(10)
	trackFuture(l, "setSize", texture.SetSize(ui.NewSize(250, 120)), showSize)
	track("setOffset", texture.SetOffset(ui.NewOffset(3.25, -4)))
	track("setOffset same", texture.SetOffset(ui.NewOffset(3.25, -4)))
	track("direction", texture.SetLayoutDirection(ui.TextDirectionRtl))
	track("direction same", texture.SetLayoutDirection(ui.TextDirectionRtl))
	track("clearFocus", texture.ClearFocus())
	track("reject", texture.RejectGesture())
	id77 := 77
	track("reject id", texture.RejectGesture(services.PlatformViewControllerRejectGestureOptions{GestureID: &id77}))
	settle()
	for _, call := range []services.MethodCall{
		services.NewMethodCall("viewFocused", 1),
		services.NewMethodCall("viewFocused", 99),
		services.NewMethodCall("unknown", 1),
	} {
		testMessenger().HandlePlatformMessage("flutter/platform_views", codec.EncodeMethodCall(call), func(reply []byte) {
			l.add("reply " + call.Method + " [" + bytesList(reply) + "]")
		}).Await()
	}
	l.ticks(6)
	track("dispose", texture.Dispose())
	l.add(state(texture))
	settle()
	track("dispose again", texture.Dispose())
	settle()

	// ---- Motion events through a texture view.
	l.add("scene motion")
	replies["create"] = []any{3}
	touch := pvs.InitAndroidView(2, "webview", ui.TextDirectionLtr)
	touch.SetPointTransformer(func(o ui.Offset) ui.Offset { return o.Sub(ui.NewOffset(5, 7.5)) })
	touch.Create(sz(100, 100)).Await()
	f64 := func(v float64) *float64 { return &v }
	ms := time.Millisecond
	type named struct {
		name  string
		event gestures.PointerEvent
	}
	down := func(o gestures.PointerDownEventOptions) named {
		return named{"PointerDownEvent", gestures.NewPointerDownEvent(o)}
	}
	move := func(o gestures.PointerMoveEventOptions) named {
		return named{"PointerMoveEvent", gestures.NewPointerMoveEvent(o)}
	}
	up := func(o gestures.PointerUpEventOptions) named {
		return named{"PointerUpEvent", gestures.NewPointerUpEvent(o)}
	}
	events := []named{
		down(gestures.PointerDownEventOptions{TimeStamp: 1500 * time.Microsecond, Pointer: 11, Position: ui.NewOffset(10, 20),
			Pressure: f64(0.75), Size: 0.125, RadiusMajor: 3, RadiusMinor: 2, Orientation: 0.5, EmbedderID: 900}),
		move(gestures.PointerMoveEventOptions{TimeStamp: 3 * ms, Pointer: 11, Position: ui.NewOffset(12, 22)}),
		{"PointerHoverEvent", gestures.NewPointerHoverEvent(gestures.PointerHoverEventOptions{Pointer: 11, Position: ui.NewOffset(1, 1)})},
		down(gestures.PointerDownEventOptions{TimeStamp: 4 * ms, Pointer: 12, Kind: ui.PointerDeviceKindStylus, Position: ui.NewOffset(50, 60)}),
		move(gestures.PointerMoveEventOptions{TimeStamp: 5 * ms, Pointer: 12, Position: ui.NewOffset(51, 61), PlatformData: 1}),
		move(gestures.PointerMoveEventOptions{TimeStamp: 5 * ms, Pointer: 11, Position: ui.NewOffset(13, 23), PlatformData: 2 | (2 << 8)}),
		move(gestures.PointerMoveEventOptions{TimeStamp: 5 * ms, Pointer: 12, Kind: ui.PointerDeviceKindStylus, Position: ui.NewOffset(52, 62),
			PlatformData: 2 | (2 << 8)}),
		up(gestures.PointerUpEventOptions{TimeStamp: 6 * ms, Pointer: 11, Position: ui.NewOffset(13, 23)}),
		down(gestures.PointerDownEventOptions{TimeStamp: 7 * ms, Pointer: 13, Kind: ui.PointerDeviceKindInvertedStylus, Position: ui.NewOffset(70, 80)}),
		{"PointerCancelEvent", gestures.NewPointerCancelEvent(gestures.PointerCancelEventOptions{TimeStamp: 8 * ms, Pointer: 12})},
		up(gestures.PointerUpEventOptions{TimeStamp: 9 * ms, Pointer: 13, Kind: ui.PointerDeviceKindInvertedStylus}),
		down(gestures.PointerDownEventOptions{TimeStamp: 20 * ms, Pointer: 14, Kind: ui.PointerDeviceKindMouse, Position: ui.NewOffset(-1, -2)}),
		up(gestures.PointerUpEventOptions{TimeStamp: 21 * ms, Pointer: 14, Kind: ui.PointerDeviceKindMouse}),
		down(gestures.PointerDownEventOptions{TimeStamp: 22 * ms, Pointer: 15, Kind: ui.PointerDeviceKindUnknown}),
		up(gestures.PointerUpEventOptions{TimeStamp: 23 * ms, Pointer: 15, Kind: ui.PointerDeviceKindUnknown}),
	}
	for _, e := range events {
		l.ticks(4)
		track("dispatch "+e.name+" "+strconv.Itoa(e.event.Pointer()), touch.DispatchPointerEvent(e.event))
		settle()
	}
	touch.DispatchPointerEvent(gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{Pointer: 16, Position: ui.NewOffset(1, 1)}).
		Transformed(vm.NewMatrix4TranslationValues(9, 9, 0))).Await()
	touch.DispatchPointerEvent(gestures.NewPointerUpEvent(gestures.PointerUpEventOptions{Pointer: 16, Position: ui.NewOffset(2, 2)})).Await()
	touch.Dispose().Await()
	settle()

	// ---- Texture view falling back to hybrid composition (create => null).
	l.add("scene texture fallback")
	fallback := pvs.InitAndroidView(3, "webview", ui.TextDirectionLtr)
	fallback.SetSize(ui.NewSize(30, 40)).Await()
	l.add(state(fallback))
	trackFuture(l, "setSize", fallback.SetSize(ui.NewSize(31, 41)), showSize)
	track("setOffset", fallback.SetOffset(ui.NewOffset(1, 1)))
	settle()
	fallback.SetPointTransformer(func(o ui.Offset) ui.Offset { return o })
	fallback.DispatchPointerEvent(gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{Pointer: 1, Position: ui.NewOffset(4, 4)})).Await()
	fallback.Dispose().Await()
	settle()

	// ---- Surface view: texture, then hybrid fallback.
	l.add("scene surface")
	replies["create"] = []any{12}
	surface := pvs.InitSurfaceAndroidView(4, "map", ui.TextDirectionLtr, services.PlatformViewsServiceInitOptions{
		CreationParams: "params", CreationParamsCodec: params})
	surface.Create(sz(10, 10)).Await()
	l.add(state(surface))
	surface.Dispose().Await()
	surface2 := pvs.InitSurfaceAndroidView(5, "map", ui.TextDirectionRtl)
	pos12 := ui.NewOffset(1, 2)
	size10 := ui.NewSize(10, 10)
	surface2.Create(services.PlatformViewControllerCreateOptions{Size: &size10, Position: &pos12}).Await()
	l.add(state(surface2))
	trackFuture(l, "setSize", surface2.SetSize(ui.NewSize(20, 20)), showSize)
	l.add("setOffset " + dart.ToString(guardValue(func() any { return surface2.SetOffset(ui.OffsetZero) })))
	id3 := 3
	track("reject", surface2.RejectGesture(services.PlatformViewControllerRejectGestureOptions{GestureID: &id3}))
	settle()
	surface2.Dispose().Await()
	settle()

	// ---- Expensive (hybrid) view.
	l.add("scene expensive")
	expensive := pvs.InitExpensiveAndroidView(6, "map", ui.TextDirectionRtl, services.PlatformViewsServiceInitOptions{
		CreationParams: []any{1, "two", nil}, CreationParamsCodec: params})
	l.ticks(8)
	pos45 := ui.NewOffset(4, 5)
	track("create", expensive.Create(services.PlatformViewControllerCreateOptions{Position: &pos45}))
	settle()
	l.add(state(expensive))
	trackFuture(l, "setSize", expensive.SetSize(ui.NewSize(1, 1)), showSize)
	l.add("setOffset " + dart.ToString(guardValue(func() any { return expensive.SetOffset(ui.OffsetZero) })))
	expensive.SetPointTransformer(func(o ui.Offset) ui.Offset { return o.Mul(2) })
	expensive.DispatchPointerEvent(gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{Pointer: 1, Position: ui.NewOffset(4, 4)})).Await()
	expensive.DispatchPointerEvent(gestures.NewPointerUpEvent(gestures.PointerUpEventOptions{Pointer: 1, Position: ui.NewOffset(4, 4)})).Await()
	expensive.Dispose().Await()
	settle()

	// ---- Hybrid (platform_views_2) view.
	l.add("scene hybrid")
	replies["isSurfaceControlEnabled"] = []any{true}
	trackFuture(l, "supported", services.HybridAndroidViewControllerCheckIfSupported(), strconv.FormatBool)
	hybrid := pvs.InitHybridAndroidView(7, "map", ui.TextDirectionLtr)
	hybrid.Create(sz(5, 5)).Await()
	l.add(state(hybrid))
	hybrid.SetPointTransformer(func(o ui.Offset) ui.Offset { return o })
	hybrid.DispatchPointerEvent(gestures.NewPointerDownEvent(gestures.PointerDownEventOptions{Pointer: 1, Position: ui.NewOffset(4, 4)})).Await()
	hybrid.DispatchPointerEvent(gestures.NewPointerCancelEvent(gestures.PointerCancelEventOptions{Pointer: 1})).Await()
	hybrid.RejectGesture().Await()
	id8 := 8
	hybrid.RejectGesture(services.PlatformViewControllerRejectGestureOptions{GestureID: &id8}).Await()
	hybrid.ClearFocus().Await()
	hybrid.SetLayoutDirection(ui.TextDirectionRtl).Await()
	hybrid.Dispose().Await()
	settle()

	// ---- Create and dispose at once.
	l.add("scene create+dispose")
	replies["create"] = []any{1}
	racing := pvs.InitAndroidView(8, "webview", ui.TextDirectionLtr)
	racing.AddOnPlatformViewCreatedListener(func(id int) { l.add("racing created " + strconv.Itoa(id)) })
	l.ticks(10)
	track("create", racing.Create(sz(1, 1)))
	track("dispose", racing.Dispose())
	l.add(state(racing))
	settle()
	l.add(state(racing))

	never := pvs.InitAndroidView(10, "webview", ui.TextDirectionLtr)
	track("dispose uncreated", never.Dispose())
	settle()
	l.add(state(never))

	// ---- Platform errors.
	l.add("scene errors")
	nope := "nope"
	replies["create"] = []any{services.NewPlatformException("error", services.PlatformExceptionOptions{Message: &nope})}
	failing := pvs.InitAndroidView(9, "x", ui.TextDirectionLtr)
	l.ticks(10)
	track("create", failing.Create(sz(1, 1)))
	settle()
	l.add(state(failing))

	// ---- UiKit and AppKit.
	l.add("scene darwin")
	for _, policy := range services.UiKitViewGestureBlockingPolicyValues {
		l.ticks(6)
		p := policy
		trackFuture(l, "uikit "+policy.String(), async.Then(pvs.InitUiKitView(20+int(policy), "ios", ui.TextDirectionLtr,
			services.InitUiKitViewOptions{GestureBlockingPolicy: &p}), func(c *services.UiKitViewController) int { return c.ID }),
			strconv.Itoa)
		settle()
	}
	uikit := pvs.InitUiKitView(30, "ios", ui.TextDirectionLtr, services.InitUiKitViewOptions{
		CreationParams: []dart.MapEntry[string, any]{{Key: "a", Value: true}}, CreationParamsCodec: params,
		OnFocus: func() { l.add("focus 30") },
	}).Await()
	testMessenger().HandlePlatformMessage("flutter/platform_views", codec.EncodeMethodCall(services.NewMethodCall("viewFocused", 30)),
		func(reply []byte) { l.add("reply [" + bytesList(reply) + "]") }).Await()
	l.ticks(6)
	track("accept", uikit.AcceptGesture())
	track("reject", uikit.RejectGesture())
	track("direction", uikit.SetLayoutDirection(ui.TextDirectionRtl))
	track("dispose", uikit.Dispose())
	settle()
	uikitDefault := pvs.InitUiKitView(31, "ios", ui.TextDirectionRtl).Await()
	l.add("uikit default " + strconv.Itoa(uikitDefault.ID))
	appkit := pvs.InitAppKitView(40, "mac", ui.TextDirectionLtr, services.PlatformViewsServiceInitOptions{
		CreationParams: 42, CreationParamsCodec: params, OnFocus: func() { l.add("focus 40") },
	}).Await()
	testMessenger().HandlePlatformMessage("flutter/platform_views", codec.EncodeMethodCall(services.NewMethodCall("viewFocused", 40)), nil).Await()
	appkit.AcceptGesture().Await()
	appkit.Dispose().Await()
	testMessenger().HandlePlatformMessage("flutter/platform_views", codec.EncodeMethodCall(services.NewMethodCall("viewFocused", 40)), nil).Await()
	pvs.InitAppKitView(41, "mac", ui.TextDirectionRtl).Await()
	settle()

	// ---- toString.
	l.add("scene toString")
	l.add(services.AndroidPointerProperties{ID: 3, ToolType: services.AndroidPointerPropertiesKToolTypeStylus}.String())
	nan, inf := dartNaN, math.Inf(1)
	l.add(services.AndroidPointerCoords{Orientation: 0.1, Pressure: 1e21, Size: math.Copysign(0, -1), ToolMajor: 1.0 / 3, ToolMinor: 2.5e-7,
		TouchMajor: 100, TouchMinor: inf, X: nan, Y: -12345.678}.String())
	l.add((&services.AndroidMotionEvent{DownTime: 1, EventTime: 2,
		Action:       services.AndroidViewControllerPointerAction(3, services.AndroidViewControllerKActionPointerUp),
		PointerCount: 1, PointerProperties: []services.AndroidPointerProperties{{ID: 0, ToolType: 1}},
		PointerCoords: []services.AndroidPointerCoords{{Pressure: 1, X: 1.5, Y: 2}}, XPrecision: 1, YPrecision: 1,
		Source: services.AndroidViewControllerKInputDeviceSourceTouchScreen, MotionEventID: 5}).String())
	for _, pointer := range []int{0, 1, 255, 256, 300} {
		l.add("pointerAction " + strconv.Itoa(pointer) + " " + strconv.Itoa(services.AndroidViewControllerPointerAction(pointer, 6)))
	}
	l.add(dart.ToString(services.UiKitViewGestureBlockingPolicyValues))
}
