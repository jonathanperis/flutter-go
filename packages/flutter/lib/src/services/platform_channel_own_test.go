package services_test

// Own tests of platform_channel.go, binding.go and flutter_test's
// TestDefaultBinaryMessenger for what Flutter's tests don't check:
// profiling, nil handler futures, the all-messages handler, the keyboard
// mock, the event channel's close and cancel, the binding's
// initializationComplete message, and the Go map key order.

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestBindingSendsInitializationComplete(t *testing.T) {
	ensureInitialized()
	var methods []string
	for _, m := range initMessages {
		if m.Channel == "flutter/platform" {
			methods = append(methods, jsonMethod.DecodeMethodCall(m.Data).Method)
		}
	}
	if !reflect.DeepEqual(methods, []string{"System.initializationComplete"}) {
		t.Fatalf("got %v", methods)
	}
}

func TestProfilePlatformChannels(t *testing.T) {
	ensureInitialized()
	var printed []string
	prevPrint := foundation.DebugPrint
	foundation.DebugPrint = func(message string, _ ...foundation.DebugPrintOptions) { printed = append(printed, message) }
	defer func() { foundation.DebugPrint = prevPrint }()
	services.DebugProfilePlatformChannels = true
	defer func() { services.DebugProfilePlatformChannels = false }()
	if !services.ShouldProfilePlatformChannels() {
		t.Fatal("ShouldProfilePlatformChannels")
	}
	method := services.NewMethodChannel("prof/method")
	basic := services.NewBasicMessageChannel[*string]("prof/basic", services.StringCodec{})
	messenger().SetMockMethodCallHandler(method, func(services.MethodCall) async.Future[any] {
		return async.FutureValue[any]("result")
	})
	messenger().SetMockMessageHandler("prof/basic", func(m []byte) async.Future[[]byte] { return async.FutureValue(m) })
	defer messenger().SetMockMessageHandler("prof/method", nil)
	defer messenger().SetMockMessageHandler("prof/basic", nil)
	runAsync(t, func() {
		method.InvokeMethod("m", 1).Await()
		method.InvokeMethod("m", 2).Await()
		basic.Send(strp("hello")).Await()
		// The same channel gets the same profiled messenger.
		expect(method.BinaryMessenger() == method.BinaryMessenger(), "messenger cached")
		async.FutureDelayed(2*time.Second, func() any { return nil }).Await()
		// The stats start over after each report.
		method.InvokeMethod("m", 1).Await()
		async.FutureDelayed(2*time.Second, func() any { return nil }).Await()
	})
	want := "Platform Channel Stats:\n" +
		`  (name:"prof/method#m" type:"MethodChannel" codec:"StandardMethodCodec" upBytes:16 upBytes_avg:8.0 downBytes:18 downBytes_avg:9.0)` + "\n" +
		`  (name:"prof/basic" type:"BasicMessageChannel<String>" codec:"StringCodec" upBytes:5 upBytes_avg:5.0 downBytes:5 downBytes_avg:5.0)` + "\n"
	want2 := "Platform Channel Stats:\n" +
		`  (name:"prof/method#m" type:"MethodChannel" codec:"StandardMethodCodec" upBytes:8 upBytes_avg:8.0 downBytes:9 downBytes_avg:9.0)` + "\n"
	if len(printed) != 2 || printed[0] != want || printed[1] != want2 {
		t.Fatalf("printed %q\nwant %q", printed, want)
	}
}

func TestMethodCallHandlerReturningNilFuture(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		channel.SetMethodCallHandler(func(services.MethodCall) async.Future[any] { return nil })
		envelope := handleOnCh7(services.NewMethodCall("x"))
		expect(string(envelope) == "[null]", "envelope %s", envelope)
		// The same through a mock handler.
		other := services.NewMethodChannel("nilfuture", services.MethodChannelOptions{Codec: jsonMethod})
		messenger().SetMockMethodCallHandler(other, func(services.MethodCall) async.Future[any] { return nil })
		defer messenger().SetMockMethodCallHandler(other, nil)
		expect(other.InvokeMethod("x").Await() == nil, "nil")
	})
}

func TestAllMessagesHandlerAndKeyboardMock(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		m := messenger()
		var seen []string
		m.AllMessagesHandler = func(channel string, handler services.MessageHandler, message []byte) async.Future[[]byte] {
			seen = append(seen, channel)
			expect((handler != nil) == (channel == "flutter/keyboard"), "handler for %s", channel)
			return async.FutureValue([]byte("all"))
		}
		reply := m.Send("any", nil).Await()
		reply2 := m.Send("flutter/keyboard", nil).Await()
		m.AllMessagesHandler = nil
		expect(string(reply) == "all" && string(reply2) == "all" && reflect.DeepEqual(seen, []string{"any", "flutter/keyboard"}),
			"replies %q %q seen %v", reply, reply2, seen)
		// flutter/keyboard's getKeyboardState mock answers an empty map.
		state := services.MethodChannelInvokeMapMethod[int, int](services.SystemChannels.Keyboard, "getKeyboardState").Await()
		expect(state != nil && len(state) == 0, "state %v", state)
	})
}

func TestEventChannelCloseAndCancel(t *testing.T) {
	ensureInitialized()
	channel := services.NewEventChannel("events/close", services.MethodChannelOptions{Codec: jsonMethod})
	messenger().SetMockMethodCallHandler(services.NewMethodChannel("events/close", services.MethodChannelOptions{Codec: jsonMethod}),
		func(services.MethodCall) async.Future[any] { return async.FutureValue[any](nil) })
	defer messenger().SetMockMessageHandler("events/close", nil)
	runAsync(t, func() {
		var log []string
		sub := channel.ReceiveBroadcastStream().Listen(func(e any) { log = append(log, "event "+dart.ToString(e)) },
			async.StreamListenOptions{OnDone: func() { log = append(log, "done") }})
		async.FutureDelayed(0, func() any { return nil }).Await()
		messenger().HandlePlatformMessage("events/close", jsonMethod.EncodeSuccessEnvelope(1), nil).Await()
		// A null message closes the stream; the handler replies once the
		// close has completed (after the done event).
		messenger().HandlePlatformMessage("events/close", nil, nil).Await()
		log = append(log, "closed")
		expect(reflect.DeepEqual(log, []string{"event 1", "done", "closed"}), "log %v", log)
		sub.Cancel()

		// After the last listener cancels, the channel's handler is removed.
		log = nil
		sub = channel.ReceiveBroadcastStream().Listen(func(e any) { log = append(log, "event "+dart.ToString(e)) })
		async.FutureDelayed(0, func() any { return nil }).Await()
		sub.Cancel().Await()
		async.FutureDelayed(0, func() any { return nil }).Await()
		// (the channel buffers now buffer the platform's messages again)
		replied := false
		services.ServicesBindingInstance().ChannelBuffers().Push("events/close", jsonMethod.EncodeSuccessEnvelope(3), func([]byte) { replied = true })
		async.FutureDelayed(0, func() any { return nil }).Await()
		expect(!replied, "a handler answered after cancel")
		messenger().HandlePlatformMessage("events/close", jsonMethod.EncodeSuccessEnvelope(2), nil).Await()
		async.FutureDelayed(0, func() any { return nil }).Await()
		expect(len(log) == 0, "log %v", log)
	})
}

func TestGoMapKeyOrder(t *testing.T) {
	standard := services.NewStandardMessageCodec()
	// Large ints sort numerically (their doubles tie, and their texts sort
	// the other way).
	a, b := 999999999999999999, 1000000000000000000
	got := standard.EncodeMessage(map[any]any{b: 0, a: 0})
	want := standard.EncodeMessage(entries(a, 0, b, 0))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got % x", got)
	}
	_ = strings.Repeat
	_ = flutter_test.AddTearDown
}
