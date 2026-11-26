package difftest

// Go twin of flutter/test/platform_channels_test.dart: the same scenes on
// services' channels and flutter_test's TestDefaultBinaryMessenger, logged
// against a chain of microtasks and compared line by line.

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/scheduler"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// channelsBinding is the services part of TestWidgetsFlutterBinding.
type channelsBinding struct {
	flutter_test.TestDefaultBinaryMessengerBindingMixin
}

func newChannelsBinding() {
	loop := async.NewLoop()
	defer async.SetCurrent(loop)()
	b := &channelsBinding{}
	base := &foundation.BindingBaseBase{}
	sched := &scheduler.SchedulerBindingMixin{BindingBase: base}
	svc := &services.ServicesBindingMixin{SchedulerBinding: sched}
	b.TestDefaultBinaryMessengerBindingMixin.ServicesBinding = svc
	sched.SetSelf(b)
	svc.SetSelf(b)
	b.TestDefaultBinaryMessengerBindingMixin.SetSelf(b)
	base.Construct(b)
	loop.FlushMicrotasks()
	loop.RunPosted()
}

func TestPlatformChannelsMatchFlutter(t *testing.T) {
	data, err := os.ReadFile("flutter/build/platform_channels.json")
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
	var show func(v any) string
	show = func(v any) string {
		switch x := v.(type) {
		case []byte:
			if x == nil {
				return "null"
			}
			var s []string
			for _, b := range x {
				s = append(s, strconv.Itoa(int(b)))
			}
			return "bytes:" + strings.Join(s, ",")
		case *services.PlatformException:
			return "PlatformException(" + x.Code + ", " + codecNullable(x.Message) + ", " + dart.ToString(x.Details) + ", " +
				codecNullable(x.Stacktrace) + ")"
		}
		return dart.ToString(v)
	}
	track := func(name string, f async.Future[any]) {
		async.OnComplete(f, func(v any) { add(name + " -> " + show(v)) }, func(err error) {
			var pe *services.PlatformException
			if errors.As(err, &pe) {
				add(name + " !! " + show(pe))
				return
			}
			add(name + " !! " + show(err))
		})
	}
	trackBytes := func(name string, f async.Future[[]byte]) {
		track(name, async.EraseFuture(f))
	}
	settle := func() { async.FutureDelayed(time.Millisecond, func() any { return nil }).Await() }
	messenger := func() *flutter_test.TestDefaultBinaryMessenger {
		return flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
	}
	asyncAny := func(f func() any) async.Future[any] { return async.Async(f) }

	body := func() {
		jsonMethod := services.JSONMethodCodec{}
		channel := services.NewMethodChannel("diff/method", services.MethodChannelOptions{Codec: jsonMethod})
		standardChannel := services.NewMethodChannel("diff/standard")
		optional := services.NewOptionalMethodChannel("diff/optional", services.MethodChannelOptions{Codec: jsonMethod})

		// ---- invokeMethod against mock handlers.
		add("scene invoke")
		messenger().SetMockMethodCallHandler(channel, func(call services.MethodCall) async.Future[any] {
			return asyncAny(func() any {
				add("handler " + call.Method + " " + dart.ToString(call.Arguments))
				switch call.Method {
				case "echo":
					return call.Arguments
				case "fail":
					panic(services.NewPlatformException("E", services.PlatformExceptionOptions{Message: ptr("failed"),
						Details: map[string]any{"k": 1}}))
				case "missing":
					panic(services.NewMissingPluginException())
				case "error":
					panic(dart.StateError{Message: "boom"})
				case "later":
					async.FutureDelayed(0, func() any { return nil }).Await()
					return "late"
				}
				return nil
			})
		})
		ticks(12)
		track("echo", channel.InvokeMethod("echo", []any{1, "a"}))
		track("fail", channel.InvokeMethod("fail"))
		track("missing", channel.InvokeMethod("missing"))
		track("error", channel.InvokeMethod("error"))
		track("null", channel.InvokeMethod("other"))
		add("sent")
		settle()
		track("later", channel.InvokeMethod("later"))
		settle()

		// ---- Raw mock message handlers: null replies, sync futures.
		add("scene raw")
		messenger().SetMockMessageHandler("diff/optional", func([]byte) async.Future[[]byte] { return nil })
		messenger().SetMockMessageHandler("diff/standard", func(message []byte) async.Future[[]byte] {
			add("raw handler " + show(message))
			return async.FutureValue(services.StandardMethodCodec{}.EncodeSuccessEnvelope([]any{true}))
		})
		ticks(8)
		track("optional", optional.InvokeMethod("anything"))
		track("standard", standardChannel.InvokeMethod("m", 7))
		track("list", async.EraseFuture(services.MethodChannelInvokeListMethod[any](standardChannel, "m")))
		add("sent")
		settle()
		messenger().SetMockMessageHandler("diff/optional", nil)
		messenger().SetMockMessageHandler("diff/standard", nil)
		messenger().SetMockMethodCallHandler(channel, nil)

		// ---- setMethodCallHandler + handlePlatformMessage.
		add("scene incoming")
		channel.SetMethodCallHandler(func(call services.MethodCall) async.Future[any] {
			return asyncAny(func() any {
				add("incoming " + call.Method + " " + dart.ToString(call.Arguments))
				switch call.Method {
				case "throw":
					panic(dart.ArgumentError{Message: "bad"})
				case "platform":
					panic(services.NewPlatformException("P", services.PlatformExceptionOptions{Message: ptr("pm")}))
				case "missing":
					panic(services.NewMissingPluginException("nope"))
				}
				return dart.ToString(call.Arguments) + "!"
			})
		})
		ticks(10)
		for _, m := range []string{"hi", "throw", "platform", "missing"} {
			trackBytes("handle "+m, messenger().HandlePlatformMessage("diff/method",
				jsonMethod.EncodeMethodCall(services.NewMethodCall(m, "x")), func(reply []byte) {
					var decoded string
					func() {
						defer func() {
							if r := recover(); r != nil {
								decoded = show(r)
							}
						}()
						if reply == nil {
							decoded = "null"
						} else {
							decoded = dart.ToString(jsonMethod.DecodeEnvelope(reply))
						}
					}()
					add("reply " + m + " " + decoded)
				}))
		}
		add("sent")
		settle()
		channel.SetMethodCallHandler(nil)
		trackBytes("no handler", messenger().HandlePlatformMessage("diff/method", nil, func(r []byte) { add("reply none " + show(r)) }))
		settle()

		// ---- BasicMessageChannel both ways.
		add("scene basic")
		basic := services.NewBasicMessageChannel[*string]("diff/basic", services.StringCodec{})
		flutter_test.TestDefaultBinaryMessengerSetMockDecodedMessageHandler(messenger(), basic, func(message *string) async.Future[*string] {
			return async.Async(func() *string {
				add("platform got " + *message)
				return ptr(*message + " back")
			})
		})
		basic.SetMessageHandler(func(message *string) async.Future[*string] {
			return async.Async(func() *string {
				add("framework got " + *message)
				return ptr(*message + " ack")
			})
		})
		ticks(8)
		async.OnComplete(basic.Send(ptr("ping")), func(v *string) { add("basic send -> " + *v) }, nil)
		trackBytes("basic receive", messenger().HandlePlatformMessage("diff/basic", services.StringCodec{}.EncodeMessage(ptr("pong")),
			func(r []byte) { add("basic reply " + *services.StringCodec{}.DecodeMessage(r)) }))
		add("sent")
		settle()
		flutter_test.TestDefaultBinaryMessengerSetMockDecodedMessageHandler[*string](messenger(), basic, nil)
		basic.SetMessageHandler(nil)

		// ---- Channel buffers: messages pushed before the handler exists.
		add("scene buffers")
		buffers := services.ServicesBindingInstance().ChannelBuffers()
		buffers.Push("diff/buffered", []byte("one"), func(r []byte) { add("buffered reply 1 " + show(r)) })
		buffers.Push("diff/buffered", []byte("two"), func(r []byte) { add("buffered reply 2 " + show(r)) })
		ticks(10)
		messenger().SetMessageHandler("diff/buffered", func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				add("buffered got " + string(message))
				return message
			})
		})
		add("handler set")
		settle()
		buffers.Resize("diff/buffered2", 2)
		for _, s := range []string{"a", "b", "c"} {
			buffers.Push("diff/buffered2", []byte(s), func(r []byte) { add("buffered2 reply " + s + " " + show(r)) })
		}
		messenger().SetMessageHandler("diff/buffered2", func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				s := string(message)
				add("buffered2 got " + s)
				if s == "c" {
					panic(dart.StateError{Message: "handler failed"})
				}
				return nil
			})
		})
		previous := foundation.FlutterErrorOnError
		reportTo := func(d *foundation.FlutterErrorDetails) {
			add("reported " + dart.ToString(d.Exception) + " | " + *d.Library + " | " + d.Context.String())
		}
		foundation.FlutterErrorOnError = reportTo
		settle()
		foundation.FlutterErrorOnError = previous
		messenger().SetMessageHandler("diff/buffered", nil)
		messenger().SetMessageHandler("diff/buffered2", nil)

		// ---- EventChannel.
		add("scene events")
		events := services.NewEventChannel("diff/events", services.MethodChannelOptions{Codec: jsonMethod})
		messenger().SetMockStreamHandler(events, flutter_test.NewMockStreamHandlerInline(
			func(arguments any, sink *flutter_test.MockStreamHandlerEventSink) {
				add("listen " + dart.ToString(arguments))
				sink.Success(dart.ToString(arguments) + " 1")
				sink.Error("E", services.PlatformExceptionOptions{Message: ptr("m"), Details: 3})
				sink.Success(dart.ToString(arguments) + " 2")
				sink.EndOfStream()
			},
			flutter_test.MockStreamHandlerInlineOptions{OnCancel: func(arguments any) { add("cancel " + dart.ToString(arguments)) }},
		))
		ticks(16)
		stream := events.ReceiveBroadcastStream("args")
		stream.Listen(func(e any) { add("event " + dart.ToString(e)) }, async.StreamListenOptions{
			OnError: func(e error) { add("event error " + show(e)) },
			OnDone:  func() { add("event done") },
		})
		add("listening")
		settle()
		messenger().SetMockStreamHandler(events, nil)

		// An event channel whose platform side is missing.
		add("scene events missing")
		messenger().SetMockMessageHandler("diff/events2", func([]byte) async.Future[[]byte] {
			return async.Async(func() []byte { return nil })
		})
		foundation.FlutterErrorOnError = reportTo
		events2 := services.NewEventChannel("diff/events2")
		sub := events2.ReceiveBroadcastStream().Listen(func(e any) { add("event2 " + dart.ToString(e)) })
		settle()
		sub.Cancel().Await()
		settle()
		foundation.FlutterErrorOnError = previous
		messenger().SetMockMessageHandler("diff/events2", nil)

		// ---- Pending messages.
		add("scene pending")
		completer := async.NewCompleter[[]byte]()
		messenger().SetMockMessageHandler("diff/pending", func([]byte) async.Future[[]byte] { return completer.Future() })
		trackBytes("pending send", messenger().Send("diff/pending", nil))
		add("pending " + strconv.Itoa(messenger().PendingMessageCount()))
		// Dart's Future.wait completes with the list of results ([null]).
		async.OnComplete(messenger().PlatformMessagesFinished(), func(async.Void) { add("finished -> [null]") }, nil)
		completer.Complete(nil)
		add("pending after complete " + strconv.Itoa(messenger().PendingMessageCount()))
		settle()
		add("pending at end " + strconv.Itoa(messenger().PendingMessageCount()))
		messenger().SetMockMessageHandler("diff/pending", nil)

		// ---- A null mock reply (await null), before other microtasks.
		add("scene await null")
		messenger().SetMockMessageHandler("diff/optional", func([]byte) async.Future[[]byte] { return nil })
		track("optional first", optional.InvokeMethod("x"))
		ticks(4)
		settle()
		ticks(4)
		track("optional second", optional.InvokeMethod("x"))
		settle()
		messenger().SetMockMessageHandler("diff/optional", nil)

		// ---- An event stream closed by the platform.
		add("scene events close")
		messenger().SetMockMessageHandler("diff/events3", func([]byte) async.Future[[]byte] {
			return async.Async(func() []byte { return jsonMethod.EncodeSuccessEnvelope(nil) })
		})
		events3 := services.NewEventChannel("diff/events3", services.MethodChannelOptions{Codec: jsonMethod})
		events3.ReceiveBroadcastStream().Listen(func(e any) { add("event3 " + dart.ToString(e)) },
			async.StreamListenOptions{OnDone: func() { add("event3 done") }})
		settle()
		ticks(8)
		trackBytes("close", messenger().HandlePlatformMessage("diff/events3", nil, nil))
		settle()
		messenger().SetMockMessageHandler("diff/events3", nil)

		// ---- Profiling.
		add("scene profiling")
		previousPrint := foundation.DebugPrint
		foundation.DebugPrint = func(message string, _ ...foundation.DebugPrintOptions) { add("debugPrint " + message) }
		services.DebugProfilePlatformChannels = true
		profiled := services.NewMethodChannel("diff/profiled")
		messenger().SetMockMethodCallHandler(profiled, func(call services.MethodCall) async.Future[any] {
			return async.Async(func() any { return []any{call.Arguments} })
		})
		profiled.InvokeMethod("m", 1).Await()
		profiled.InvokeMethod("m", "two").Await()
		profiled.InvokeMethod("n").Await()
		async.FutureDelayed(1100*time.Millisecond, func() any { return nil }).Await()
		services.DebugProfilePlatformChannels = false
		foundation.DebugPrint = previousPrint
		messenger().SetMockMethodCallHandler(profiled, nil)
	}

	done := false
	var failure error
	async.OnComplete(async.AsyncVoid(body), func(async.Void) { done = true }, func(err error) { failure = err; done = true })
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
}
