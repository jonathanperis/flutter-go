package services_test

// Ports of packages/flutter/test/services/platform_channel_test.dart,
// platform_messages_test.dart, default_binary_messenger_test.dart,
// set_method_call_handler_test.dart,
// use_platform_channel_without_initialization_test.dart,
// channel_buffers_test.dart, binding_lifecycle_test.dart, lifecycle_test.dart
// and the portable parts of binding_test.dart (Flutter 3.50.1).
//
// Skipped / adapted:
//   - "can invoke list/map method and get result": Dart's
//     `invokeMethod<List<String>>` throwing a TypeError (a failed cast of
//     List<dynamic>) has no Go equivalent (InvokeMethod returns any); the
//     rest of each test is ported.
//   - binding_test.dart: the licenses test (rootBundle LICENSES), the memory
//     pressure test (asset caches), the textInput handler test (TextInput),
//     the keyboard state test (HardwareKeyboard) and the view focus test
//     (ViewFocusEvent) need libraries not ported yet.
//   - lifecycle_test.dart: TestPlatformDispatcher's
//     initialLifecycleStateTestValue doesn't exist yet; the two testWidgets
//     are ported by sending the states on flutter/lifecycle, as the engine
//     does after startup (the initial-state path is tested with
//     ui.PlatformConfiguration.UpdateInitialLifecycleState in a child
//     process).

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func bytesHandler(f func(message []byte) []byte) services.MessageHandler {
	return func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte { return f(message) })
	}
}

func TestBasicMessageChannelCanSendStringMessageAndGetReply(t *testing.T) {
	ensureInitialized()
	stringCodec := services.StringCodec{}
	channel := services.NewBasicMessageChannel[*string]("ch", stringCodec)
	messenger().SetMockMessageHandler("ch", bytesHandler(func(message []byte) []byte {
		return stringCodec.EncodeMessage(strp(*stringCodec.DecodeMessage(message) + " world"))
	}))
	defer messenger().SetMockMessageHandler("ch", nil)
	runAsync(t, func() {
		reply := channel.Send(strp("hello")).Await()
		expect(*reply == "hello world", "reply %q", *reply)
	})
}

func TestBasicMessageChannelCanReceiveStringMessageAndSendReply(t *testing.T) {
	ensureInitialized()
	stringCodec := services.StringCodec{}
	channel := services.NewBasicMessageChannel[*string]("ch", stringCodec)
	defer channel.SetMessageHandler(nil)
	runAsync(t, func() {
		channel.SetMessageHandler(func(message *string) async.Future[*string] {
			return async.Async(func() *string { return strp(*message + " world") })
		})
		var reply *string
		messenger().HandlePlatformMessage("ch", services.StringCodec{}.EncodeMessage(strp("hello")), func(replyBinary []byte) {
			reply = stringCodec.DecodeMessage(replyBinary)
		}).Await()
		expect(reply != nil && *reply == "hello world", "reply %v", reply)
	})
}

var (
	jsonMessage = services.JSONMessageCodec{}
	jsonMethod  = services.JSONMethodCodec{}
)

func methodChannels() (*services.MethodChannel, *services.OptionalMethodChannel) {
	return services.NewMethodChannel("ch7", services.MethodChannelOptions{Codec: jsonMethod}),
		services.NewOptionalMethodChannel("ch8", services.MethodChannelOptions{Codec: jsonMethod})
}

// methodTest runs body with Flutter's tearDown (both handlers cleared).
func methodTest(t *testing.T, body func(channel *services.MethodChannel, optional *services.OptionalMethodChannel)) {
	ensureInitialized()
	channel, optional := methodChannels()
	defer func() {
		channel.SetMethodCallHandler(nil)
		optional.SetMethodCallHandler(nil)
		messenger().SetMockMessageHandler("ch7", nil)
		messenger().SetMockMessageHandler("ch8", nil)
	}()
	runAsync(t, func() { body(channel, optional) })
}

func sayHelloHandler(result func(args any) any) services.MessageHandler {
	return bytesHandler(func(message []byte) []byte {
		methodCall := jsonMessage.DecodeMessage(message).(map[string]any)
		if methodCall["method"] == "sayHello" {
			return jsonMessage.EncodeMessage([]any{result(methodCall["args"])})
		}
		return jsonMessage.EncodeMessage([]any{"unknown", nil, nil})
	})
}

func TestMethodChannelCanInvokeMethodAndGetResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch7", sayHelloHandler(func(args any) any { return dart.ToString(args) + " world" }))
		result := channel.InvokeMethod("sayHello", "hello").Await()
		expect(result == "hello world", "result %v", result)
	})
}

func TestMethodChannelCanInvokeListMethodAndGetResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch7", sayHelloHandler(func(args any) any { return []string{dart.ToString(args), "world"} }))
		result := services.MethodChannelInvokeListMethod[string](channel, "sayHello", "hello").Await()
		expect(reflect.DeepEqual(result, []string{"hello", "world"}), "result %v", result)
		// Own: an element of another type fails the future.
		err := awaitError(services.MethodChannelInvokeListMethod[int](channel, "sayHello", "hello"))
		var typeError dart.TypeError
		expect(errors.As(err, &typeError) && typeError.Message == "type 'String' is not a subtype of type 'int' in type cast", "err %v", err)
	})
}

func TestMethodChannelCanInvokeListMethodAndGetNullResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch7", sayHelloHandler(func(any) any { return nil }))
		result := services.MethodChannelInvokeListMethod[string](channel, "sayHello", "hello").Await()
		expect(result == nil, "result %v", result)
	})
}

func TestMethodChannelCanInvokeMapMethodAndGetResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch7", sayHelloHandler(func(args any) any { return map[string]string{dart.ToString(args): "world"} }))
		result := services.MethodChannelInvokeMapMethod[string, string](channel, "sayHello", "hello").Await()
		expect(reflect.DeepEqual(result, map[string]string{"hello": "world"}), "result %v", result)
		// Own: a standard-codec map (map[any]any).
		std := services.NewMethodChannel("ch7std")
		messenger().SetMockMethodCallHandler(std, func(services.MethodCall) async.Future[any] {
			return async.FutureValue[any](map[any]any{1: "x"})
		})
		defer messenger().SetMockMethodCallHandler(std, nil)
		r2 := services.MethodChannelInvokeMapMethod[int, string](std, "m").Await()
		expect(reflect.DeepEqual(r2, map[int]string{1: "x"}), "result %v", r2)
		messenger().SetMockMethodCallHandler(std, func(services.MethodCall) async.Future[any] {
			return async.FutureValue[any]("not a map")
		})
		err := awaitError(services.MethodChannelInvokeMapMethod[int, string](std, "m"))
		var typeError dart.TypeError
		expect(errors.As(err, &typeError), "err %v", err)
	})
}

func TestMethodChannelCanInvokeMapMethodAndGetNullResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch7", sayHelloHandler(func(any) any { return nil }))
		result := services.MethodChannelInvokeMapMethod[string, string](channel, "sayHello", "hello").Await()
		expect(result == nil, "result %v", result)
	})
}

func TestMethodChannelCanInvokeMethodAndGetError(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch7", bytesHandler(func([]byte) []byte {
			return jsonMessage.EncodeMessage([]any{"bad", "Something happened", map[string]any{"a": 42, "b": 3.14}})
		}))
		err := awaitError(channel.InvokeMethod("sayHello", "hello"))
		var e *services.PlatformException
		expect(errors.As(err, &e), "err %v", err)
		expect(e.Code == "bad" && *e.Message == "Something happened" &&
			reflect.DeepEqual(e.Details, map[string]any{"a": 42, "b": 3.14}), "e %v", e)
	})
}

func TestMethodChannelCanInvokeUnimplementedMethod(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch7", bytesHandler(func([]byte) []byte { return nil }))
		err := awaitError(channel.InvokeMethod("sayHello", "hello"))
		var e *services.MissingPluginException
		expect(errors.As(err, &e), "err %v", err)
		expect(strings.Contains(*e.Message, "sayHello") && strings.Contains(*e.Message, "ch7"), "message %q", *e.Message)
		expect(*e.Message == "No implementation found for method sayHello on channel ch7", "message %q", *e.Message)
	})
}

func TestMethodChannelCanInvokeUnimplementedMethodOptional(t *testing.T) {
	methodTest(t, func(_ *services.MethodChannel, optional *services.OptionalMethodChannel) {
		messenger().SetMockMessageHandler("ch8", bytesHandler(func([]byte) []byte { return nil }))
		result := optional.InvokeMethod("sayHello", "hello").Await()
		expect(result == nil, "result %v", result)
		// Own: through the embedded *MethodChannel (SystemChannels' case).
		result = (&optional.MethodChannel).InvokeMethod("sayHello", "hello").Await()
		expect(result == nil, "result %v", result)
	})
}

func handleOnCh7(call services.MethodCall) []byte {
	var envelope []byte
	messenger().HandlePlatformMessage("ch7", jsonMethod.EncodeMethodCall(call), func(result []byte) {
		envelope = result
	}).Await()
	return envelope
}

func TestMethodChannelCanHandleMethodCallWithNoRegisteredPluginSettingBefore(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		channel.SetMethodCallHandler(nil)
		envelope := handleOnCh7(services.NewMethodCall("sayHello", "hello"))
		async.FutureValue[any](nil).Await() // just in case there's something async happening
		expect(envelope == nil, "envelope %v", envelope)
	})
}

func TestMethodChannelCanHandleMethodCallWithNoRegisteredPluginSettingAfter(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		envelope := handleOnCh7(services.NewMethodCall("sayHello", "hello"))
		channel.SetMethodCallHandler(nil)
		async.FutureValue[any](nil).Await() // just in case there's something async happening
		expect(envelope == nil, "envelope %v", envelope)
	})
}

func asyncHandler(f func(call services.MethodCall) any) func(services.MethodCall) async.Future[any] {
	return func(call services.MethodCall) async.Future[any] {
		return async.Async(func() any { return f(call) })
	}
}

func TestMethodChannelCanHandleMethodCallOfUnimplementedMethod(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		channel.SetMethodCallHandler(asyncHandler(func(services.MethodCall) any { panic(services.NewMissingPluginException()) }))
		envelope := handleOnCh7(services.NewMethodCall("sayHello", "hello"))
		expect(envelope == nil, "envelope %v", envelope)
	})
}

func TestMethodChannelCanHandleMethodCallWithSuccessfulResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		channel.SetMethodCallHandler(asyncHandler(func(call services.MethodCall) any { return dart.ToString(call.Arguments) + ", world" }))
		envelope := handleOnCh7(services.NewMethodCall("sayHello", "hello"))
		expect(jsonMethod.DecodeEnvelope(envelope) == "hello, world", "envelope %s", envelope)
	})
}

func TestMethodChannelCanHandleMethodCallWithExpressiveErrorResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		channel.SetMethodCallHandler(asyncHandler(func(services.MethodCall) any {
			panic(services.NewPlatformException("bad", services.PlatformExceptionOptions{Message: strp("sayHello failed")}))
		}))
		envelope := handleOnCh7(services.NewMethodCall("sayHello", "hello"))
		r := panicOf(func() { jsonMethod.DecodeEnvelope(envelope) })
		e, ok := r.(*services.PlatformException)
		expect(ok && e.Code == "bad" && *e.Message == "sayHello failed", "panic %v", r)
	})
}

func TestMethodChannelCanHandleMethodCallWithOtherErrorResult(t *testing.T) {
	methodTest(t, func(channel *services.MethodChannel, _ *services.OptionalMethodChannel) {
		channel.SetMethodCallHandler(asyncHandler(func(services.MethodCall) any {
			panic(dart.ArgumentError{Message: "bad"})
		}))
		envelope := handleOnCh7(services.NewMethodCall("sayHello", "hello"))
		r := panicOf(func() { jsonMethod.DecodeEnvelope(envelope) })
		e, ok := r.(*services.PlatformException)
		expect(ok && e.Code == "error" && *e.Message == "Invalid argument(s): bad", "panic %v", r)
	})
}

func TestMethodChannelCanCheckTheMockHandler(t *testing.T) {
	ensureInitialized()
	handler := func(call services.MethodCall) async.Future[any] { return async.FutureValue[any](nil) }
	channel := services.NewMethodChannel("test_handler")
	if !messenger().CheckMockMessageHandler(channel.Name, nil) {
		t.Fatal("nil")
	}
	if messenger().CheckMockMessageHandler(channel.Name, handler) {
		t.Fatal("handler before")
	}
	messenger().SetMockMethodCallHandler(channel, handler)
	defer messenger().SetMockMethodCallHandler(channel, nil)
	if !messenger().CheckMockMessageHandler(channel.Name, handler) {
		t.Fatal("handler after")
	}
}

func TestEventChannelCanReceiveEventStream(t *testing.T) {
	ensureInitialized()
	channel := services.NewEventChannel("ch", services.MethodChannelOptions{Codec: jsonMethod})
	defer messenger().SetMockStreamHandler(channel, nil)
	runAsync(t, func() {
		canceled := false
		messenger().SetMockStreamHandler(channel, flutter_test.NewMockStreamHandlerInline(
			func(arguments any, events *flutter_test.MockStreamHandlerEventSink) {
				events.Success(dart.ToString(arguments) + "1")
				events.Success(dart.ToString(arguments) + "2")
				events.EndOfStream()
			},
			flutter_test.MockStreamHandlerInlineOptions{OnCancel: func(any) { canceled = true }},
		))
		events := async.StreamToList(channel.ReceiveBroadcastStream("hello")).Await()
		expect(reflect.DeepEqual(events, []any{"hello1", "hello2"}), "events %v", events)
		async.FutureDelayed(0, func() any { return nil }).Await()
		expect(canceled, "not canceled")
	})
}

func TestEventChannelCanReceiveErrorEvent(t *testing.T) {
	ensureInitialized()
	channel := services.NewEventChannel("ch", services.MethodChannelOptions{Codec: jsonMethod})
	defer messenger().SetMockStreamHandler(channel, nil)
	runAsync(t, func() {
		messenger().SetMockStreamHandler(channel, flutter_test.NewMockStreamHandlerInline(
			func(arguments any, events *flutter_test.MockStreamHandlerEventSink) {
				events.Error("404", services.PlatformExceptionOptions{Message: strp("Not Found."), Details: arguments})
			}))
		var events []any
		var errs []error
		sub := channel.ReceiveBroadcastStream("hello").Listen(func(e any) { events = append(events, e) },
			async.StreamListenOptions{OnError: func(err error) { errs = append(errs, err) }})
		async.FutureDelayed(0, func() any { return nil }).Await()
		expect(len(events) == 0, "events %v", events)
		expect(len(errs) == 1, "errors %v", errs)
		var e *services.PlatformException
		expect(errors.As(errs[0], &e), "error %v", errs[0])
		expect(e.Code == "404" && *e.Message == "Not Found." && e.Details == "hello", "error %v", e)
		sub.Cancel()
	})
}

// platform_messages_test.dart
func TestMockBinaryMessageHandlerControlTest(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var log [][]byte
		messenger().SetMockMessageHandler("test1", func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				log = append(log, message)
				return nil
			})
		})
		message := []byte{0xAB, 0xCD} // ByteData(2)..setUint16(0, 0xABCD)
		messenger().Send("test1", message).Await()
		expect(reflect.DeepEqual(log, [][]byte{message}), "log %v", log)
		log = nil
		messenger().SetMockMessageHandler("test1", nil)
		// Unmocked: the message goes to the engine, which answers null.
		reply := messenger().Send("test1", message).Await()
		expect(len(log) == 0 && reply == nil, "log %v reply %v", log, reply)
	})
}

// default_binary_messenger_test.dart
func TestDefaultBinaryMessengerCallsCallbackOnce(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		countInbound, countOutbound := 0, 0
		const channel = "foo"
		bar := dart.UTF8Encode("bar")
		done := async.NewCompleter[async.Void]()
		services.ServicesBindingInstance().ChannelBuffers().Push(channel, bar, func(message []byte) {
			expect(message == nil, "message %v", message)
			countOutbound++
			done.Complete(async.Void{})
		})
		expect(countInbound == 0 && countOutbound == 0, "counts %d %d", countInbound, countOutbound)
		services.ServicesBindingInstance().DefaultBinaryMessenger().SetMessageHandler(channel, func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				expect(reflect.DeepEqual(message, bar), "message %v", message)
				countInbound++
				return nil
			})
		})
		defer services.ServicesBindingInstance().DefaultBinaryMessenger().SetMessageHandler(channel, nil)
		expect(countInbound == 0 && countOutbound == 0, "counts %d %d", countInbound, countOutbound)
		done.Future().Await()
		expect(countInbound == 1 && countOutbound == 1, "counts %d %d", countInbound, countOutbound)
	})
}

func TestDefaultBinaryMessengerCanCheckTheMockHandler(t *testing.T) {
	ensureInitialized()
	handler := func(call []byte) async.Future[[]byte] { return async.FutureValue[[]byte](nil) }
	m := messenger()
	if !m.CheckMockMessageHandler("test_channel", nil) {
		t.Fatal("nil")
	}
	if m.CheckMockMessageHandler("test_channel", services.MessageHandler(handler)) {
		t.Fatal("before")
	}
	m.SetMockMessageHandler("test_channel", handler)
	if !m.CheckMockMessageHandler("test_channel", services.MessageHandler(handler)) {
		t.Fatal("after")
	}
	m.SetMockMessageHandler("test_channel", nil)
}

// set_method_call_handler_test.dart
func TestSetMethodCallHandlerThrowsAssertionErrorIffBindingIsNotYetInitialized(t *testing.T) {
	if !isolated(t) {
		return
	}
	methodChannel := services.NewMethodChannel("mock")
	r := panicOf(func() { methodChannel.SetMethodCallHandler(nil) })
	if _, ok := r.(dart.AssertionFailure); !ok {
		t.Fatalf("got %#v", r)
	}
	ensureInitialized()
	if r := panicOf(func() { methodChannel.SetMethodCallHandler(nil) }); r != nil {
		t.Fatalf("got %v", r)
	}
}

// use_platform_channel_without_initialization_test.dart
func TestUsePlatformChannelWithoutInitializationThrowsAssertionError(t *testing.T) {
	if !isolated(t) {
		return
	}
	methodChannel := services.NewMethodChannel("mock")
	r := panicOf(func() { methodChannel.BinaryMessenger() })
	e, ok := r.(dart.AssertionFailure)
	if !ok || !strings.Contains(dart.ToString(e.AssertionMessage()), "WidgetsFlutterBinding.ensureInitialized()") {
		t.Fatalf("got %#v", r)
	}
	ensureInitialized()
	if r := panicOf(func() { methodChannel.BinaryMessenger() }); r != nil {
		t.Fatalf("got %v", r)
	}
}

// channel_buffers_test.dart (services)
func TestDoesDrainChannelBuffers(t *testing.T) {
	binding := ensureInitialized()
	runAsync(t, func() {
		const channel = "foo"
		expect(binding.DefaultBinaryMessenger() != nil, "no messenger")
		didCallCallback := false
		callback := func([]byte) { didCallCallback = true }
		const payload = "bar"
		ui.ChannelBuffersInstance().Push(channel, dart.UTF8Encode(payload), callback)
		didDrainData := false
		binding.DefaultBinaryMessenger().SetMessageHandler(channel, func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				expect(dart.UTF8Decode(message) == payload, "message %v", message)
				didDrainData = true
				return nil
			})
		})
		defer binding.DefaultBinaryMessenger().SetMessageHandler(channel, nil)
		// Flush the event queue.
		async.NewFuture(func() any { return nil }).Await()
		expect(didDrainData, "not drained")
		expect(didCallCallback, "no callback")
	})
}

// binding_lifecycle_test.dart
func TestCanSendMessageOnCompletionOfBindingInitialization(t *testing.T) {
	binding := ensureInitialized()
	runAsync(t, func() {
		called := false
		messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, asyncHandler(func(method services.MethodCall) any {
			if method.Method == "System.initializationComplete" {
				called = true
			}
			return nil
		}))
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
		binding.InitializationComplete().Await()
		expect(called, "not called")
	})
}

// binding_test.dart
func TestCallingExitApplicationSendsAMethodCallToTheEngine(t *testing.T) {
	binding := ensureInitialized()
	runAsync(t, func() {
		sentMessage := false
		var methodCall *services.MethodCall
		messenger().SetMockMessageHandler("flutter/platform", bytesHandler(func(message []byte) []byte {
			call := jsonMethod.DecodeMethodCall(message)
			methodCall = &call
			sentMessage = true
			return jsonMethod.EncodeSuccessEnvelope(map[string]string{"response": "cancel"})
		}))
		defer messenger().SetMockMessageHandler("flutter/platform", nil)
		response := binding.ExitApplication(ui.AppExitTypeRequired).Await()
		expect(sentMessage, "not sent")
		expect(methodCall != nil, "no call")
		expect(methodCall.Arguments.(map[string]any)["type"] == "required", "args %v", methodCall.Arguments)
		expect(methodCall.Arguments.(map[string]any)["exitCode"] == 0, "args %v", methodCall.Arguments)
		expect(response == ui.AppExitResponseCancel, "response %v", response)
		// Own: no reply (null) is cancel; another response is exit.
		messenger().SetMockMessageHandler("flutter/platform", bytesHandler(func([]byte) []byte { return jsonMethod.EncodeSuccessEnvelope(nil) }))
		expect(binding.ExitApplication(ui.AppExitTypeCancelable, 3).Await() == ui.AppExitResponseCancel, "null")
		messenger().SetMockMessageHandler("flutter/platform", bytesHandler(func([]byte) []byte {
			return jsonMethod.EncodeSuccessEnvelope(map[string]string{"response": "exit"})
		}))
		expect(binding.ExitApplication(ui.AppExitTypeCancelable).Await() == ui.AppExitResponseExit, "exit")
	})
}

func TestDefaultHandleRequestAppExitReturnsExit(t *testing.T) {
	binding := ensureInitialized()
	runAsync(t, func() {
		incomingCall := services.NewMethodCall("System.requestAppExit", []any{map[string]any{"type": "cancelable"}})
		receivedReply := false
		var result map[string]any
		binding.DefaultBinaryMessenger().HandlePlatformMessage("flutter/platform", jsonMethod.EncodeMethodCall(incomingCall),
			func(message []byte) {
				result = jsonMessage.DecodeMessage(message).([]any)[0].(map[string]any)
				receivedReply = true
			}).Await()
		expect(receivedReply, "no reply")
		expect(result != nil && result["response"] == "exit", "result %v", result)
	})
}

// lifecycle_test.dart
func TestAppLifecycleStateValuesAreInTheRightOrder(t *testing.T) {
	want := []ui.AppLifecycleState{ui.AppLifecycleStateDetached, ui.AppLifecycleStateResumed,
		ui.AppLifecycleStateInactive, ui.AppLifecycleStateHidden, ui.AppLifecycleStatePaused}
	if !reflect.DeepEqual(ui.AppLifecycleStateValues, want) {
		t.Fatal(ui.AppLifecycleStateValues)
	}
}

func sendLifecycle(state string) {
	messenger().HandlePlatformMessage("flutter/lifecycle", services.StringCodec{}.EncodeMessage(&state), nil).Await()
}

func TestHandlesAllOfTheAllowedStatesOfAppLifecycleState(t *testing.T) {
	binding := ensureInitialized()
	runAsync(t, func() {
		for _, state := range ui.AppLifecycleStateValues {
			binding.ResetInternalState()
			sendLifecycle(state.String())
			expect(services.ServicesBindingInstance().LifecycleState().String() == state.String(),
				"state %v, want %v", binding.LifecycleState(), state)
		}
		binding.ResetInternalState()
	})
}

func TestInitialLifecycleStateIsUsedToInitStatePaused(t *testing.T) {
	if !isolated(t) {
		return
	}
	pc := ui.NewPlatformConfiguration(nil)
	pc.UpdateInitialLifecycleState("AppLifecycleState.paused")
	binding := ensureInitialized()
	if got := binding.LifecycleState(); got == nil || got.String() != "AppLifecycleState.paused" {
		t.Fatalf("got %v", got)
	}
	// Once read, the initial state no longer follows the engine.
	pc.UpdateInitialLifecycleState("AppLifecycleState.resumed")
	if got := binding.PlatformDispatcher().InitialLifecycleState(); got != "AppLifecycleState.paused" {
		t.Fatalf("got %q", got)
	}
}

// ---- own tests of the binding

// lifecycleTransitions sends state after starting from previous and
// returns the states the binding went through.
func lifecycleTransitions(binding *testBinding, previous *ui.AppLifecycleState, state ui.AppLifecycleState) []ui.AppLifecycleState {
	binding.ResetInternalState()
	if previous != nil {
		sendLifecycle(previous.String())
	}
	var seen []ui.AppLifecycleState
	listener := func(s ui.AppLifecycleState) { seen = append(seen, s) }
	lifecycleListener = listener
	defer func() { lifecycleListener = nil }()
	sendLifecycle(state.String())
	return seen
}

var lifecycleListener func(ui.AppLifecycleState)

func (b *testBinding) HandleAppLifecycleStateChanged(state ui.AppLifecycleState) {
	b.TestDefaultBinaryMessengerBindingMixin.HandleAppLifecycleStateChanged(state)
	if lifecycleListener != nil {
		lifecycleListener(state)
	}
}

func TestLifecycleStateTransitions(t *testing.T) {
	binding := ensureInitialized()
	d, r, i, h, p := ui.AppLifecycleStateDetached, ui.AppLifecycleStateResumed, ui.AppLifecycleStateInactive,
		ui.AppLifecycleStateHidden, ui.AppLifecycleStatePaused
	cases := []struct {
		previous *ui.AppLifecycleState
		state    ui.AppLifecycleState
		want     []ui.AppLifecycleState
	}{
		{nil, h, []ui.AppLifecycleState{h}},
		{&r, r, nil},
		{&r, p, []ui.AppLifecycleState{i, h, p}},
		{&p, r, []ui.AppLifecycleState{h, i, r}},
		{&r, d, []ui.AppLifecycleState{i, h, p, d}},
		{&d, r, []ui.AppLifecycleState{r}},
		{&d, p, []ui.AppLifecycleState{r, i, h, p}},
		{&h, i, []ui.AppLifecycleState{i}},
		{&i, h, []ui.AppLifecycleState{h}},
		{&p, d, []ui.AppLifecycleState{d}},
		{&h, d, []ui.AppLifecycleState{p, d}},
	}
	runAsync(t, func() {
		for _, c := range cases {
			got := lifecycleTransitions(binding, c.previous, c.state)
			expect(reflect.DeepEqual(got, c.want), "%v → %v: got %v, want %v", c.previous, c.state, got, c.want)
		}
		binding.ResetInternalState()
		// An unknown state fails the handler (Dart's null check): a fake
		// message's future fails with it...
		state := "AppLifecycleState.bogus"
		err := awaitError(messenger().HandlePlatformMessage("flutter/lifecycle", services.StringCodec{}.EncodeMessage(&state), nil))
		var typeError dart.TypeError
		expect(errors.As(err, &typeError) && typeError.Message == "Null check operator used on a null value", "err %v", err)
		// ...and a real one (through the channel buffers) is reported, with
		// a null reply.
		var reported []*foundation.FlutterErrorDetails
		previous := foundation.FlutterErrorOnError
		foundation.FlutterErrorOnError = func(d *foundation.FlutterErrorDetails) { reported = append(reported, d) }
		defer func() { foundation.FlutterErrorOnError = previous }()
		replied := false
		ui.ChannelBuffersInstance().Push("flutter/lifecycle", services.StringCodec{}.EncodeMessage(&state), func(r []byte) {
			expect(r == nil, "reply %v", r)
			replied = true
		})
		async.NewFuture(func() any { return nil }).Await()
		expect(replied, "no reply (reported %d)", len(reported))
		expect(len(reported) == 1 && *reported[0].Library == "services library" &&
			reported[0].Context.String() == "during a platform message callback", "reported %v", reported)
	})
}

func TestSystemAndAccessibilityChannels(t *testing.T) {
	binding := ensureInitialized()
	runAsync(t, func() {
		var reply []byte = []byte{1}
		messenger().HandlePlatformMessage("flutter/system", jsonMessage.EncodeMessage(map[string]any{"type": "memoryPressure"}),
			func(r []byte) { reply = r }).Await()
		expect(reply == nil, "reply %v", reply)
		messenger().HandlePlatformMessage("flutter/accessibility",
			services.NewStandardMessageCodec().EncodeMessage(map[any]any{"type": "didGainFocus", "nodeId": 12}), nil).Await()
		focus := binding.AccessibilityFocus().Value()
		expect(focus != nil && *focus == 12, "focus %v", focus)
		messenger().HandlePlatformMessage("flutter/accessibility",
			services.NewStandardMessageCodec().EncodeMessage(map[any]any{"type": "other", "nodeId": 13}), nil).Await()
		expect(*binding.AccessibilityFocus().Value() == 12, "focus changed")
	})
}

type menuClient struct{ log []string }

func (c *menuClient) HandleSystemHide()                         { c.log = append(c.log, "hide") }
func (c *menuClient) HandleCustomContextMenuAction(id string) { c.log = append(c.log, "action "+id) }

func TestPlatformChannelHandler(t *testing.T) {
	binding := ensureInitialized()
	runAsync(t, func() {
		call := func(method string, args any) (any, error) {
			var envelope []byte
			messenger().HandlePlatformMessage("flutter/platform", jsonMethod.EncodeMethodCall(services.NewMethodCall(method, args)),
				func(r []byte) { envelope = r }).Await()
			var result any
			err, _ := panicOf(func() { result = jsonMethod.DecodeEnvelope(envelope) }).(error)
			return result, err
		}
		var visible []bool
		binding.SetSystemUiChangeCallback(func(v bool) async.Future[async.Void] {
			return async.AsyncVoid(func() { visible = append(visible, v) })
		})
		_, err := call("SystemChrome.systemUIChange", []any{true})
		expect(err == nil && reflect.DeepEqual(visible, []bool{true}), "visible %v err %v", visible, err)
		binding.SetSystemUiChangeCallback(nil)
		_, err = call("SystemChrome.systemUIChange", []any{false})
		expect(err == nil && len(visible) == 1, "visible %v", visible)

		client := &menuClient{}
		services.ServicesBindingSetSystemContextMenuClient(client)
		_, err = call("ContextMenu.onPerformCustomAction", []any{0, "copy"})
		expect(err == nil, "err %v", err)
		_, err = call("ContextMenu.onDismissSystemContextMenu", nil)
		expect(err == nil && reflect.DeepEqual(client.log, []string{"action copy", "hide"}), "log %v", client.log)
		// The client is cleared on dismiss: both calls now assert.
		_, err = call("ContextMenu.onDismissSystemContextMenu", nil)
		var pe *services.PlatformException
		expect(errors.As(err, &pe) && pe.Code == "error" &&
			*pe.Message == "Assertion failed: \"Platform sent onDismissSystemContextMenu when no SystemContextMenuClient was registered.\"",
			"err %v", err)
		_, err = call("ContextMenu.onPerformCustomAction", []any{0, "x"})
		expect(errors.As(err, &pe) && strings.HasPrefix(*pe.Message, "Assertion failed: \"Platform sent onPerformCustomAction"), "err %v", err)
		_, err = call("Bogus", nil)
		expect(errors.As(err, &pe) && *pe.Message == "Assertion failed: \"Method \\\"Bogus\\\" not handled.\"", "err %v", err)
	})
}

func TestServicesBindingInstanceAndChannelBuffers(t *testing.T) {
	binding := ensureInitialized()
	if services.ServicesBindingInstance() != services.ServicesBinding(binding) {
		t.Fatal("instance")
	}
	if binding.ChannelBuffers() != ui.ChannelBuffersInstance() {
		t.Fatal("channel buffers")
	}
	if services.ServicesBindingRootIsolateToken() == nil {
		t.Fatal("root isolate token")
	}
	if _, ok := panicOf(func() { services.BackgroundIsolateBinaryMessengerInstance() }).(dart.StateError); !ok {
		t.Fatal("background instance")
	}
	services.BackgroundIsolateBinaryMessengerEnsureInitialized(services.ServicesBindingRootIsolateToken())
	bg := services.BackgroundIsolateBinaryMessengerInstance()
	if _, ok := panicOf(func() { bg.SetMessageHandler("x", nil) }).(dart.UnsupportedError); !ok {
		t.Fatal("background setMessageHandler")
	}
	if _, ok := panicOf(func() { bg.HandlePlatformMessage("x", nil, nil) }).(dart.UnimplementedError); !ok {
		t.Fatal("background handlePlatformMessage")
	}
}
