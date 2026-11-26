package services

// Port of platform_channel.dart (Flutter 3.50.1): BasicMessageChannel,
// MethodChannel, OptionalMethodChannel, EventChannel and the platform
// channel profiler (debugProfilePlatformChannels).
//
// Dart's `async` methods are async.Async coroutines, so errors thrown
// before their first await still complete the returned future (in a
// microtask), and every await suspends, as in Dart.

import (
	"errors"
	"slices"
	"strconv"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
	"github.com/jonathanperis/flutter-go/src/developer"
)

// ShouldProfilePlatformChannels tells whether platform channels are being
// profiled: KProfilePlatformChannels, or DebugProfilePlatformChannels in
// non-release builds. Profiled channels log, once a second, the bytes sent
// and received on each channel (through foundation.DebugPrint).
func ShouldProfilePlatformChannels() bool {
	return KProfilePlatformChannels || (!foundation.KReleaseMode && DebugProfilePlatformChannels)
}

// KProfilePlatformChannels turns on platform channel profiling in every
// build mode (a compile-time constant, as in Dart).
const KProfilePlatformChannels = false

var profilePlatformChannelsIsRunning = false

const profilePlatformChannelsRate = time.Second

// profiledBinaryMessengers is the Expando of profiled messengers, keyed by
// channel (Go keys it by the channel's pointer).
var profiledBinaryMessengers = map[any]*profiledBinaryMessenger{}

// profiledBinaryMessenger is _ProfiledBinaryMessenger.
type profiledBinaryMessenger struct {
	proxy           BinaryMessenger
	channelTypeName string
	codecTypeName   string
}

func (m *profiledBinaryMessenger) HandlePlatformMessage(channel string, data []byte, callback ui.PlatformMessageResponseCallback) async.Future[[]byte] {
	return m.proxy.HandlePlatformMessage(channel, data, callback)
}

func (m *profiledBinaryMessenger) sendWithPostfix(channel, postfix string, message []byte) async.Future[[]byte] {
	return async.Async(func() []byte {
		debugRecordUpStream(m.channelTypeName, channel+postfix, m.codecTypeName, message)
		timelineTask := developer.NewTimelineTask()
		timelineTask.Start("Platform Channel send "+channel+postfix, nil)
		var result []byte
		func() {
			defer timelineTask.Finish(nil)
			result = awaitOrNull(m.proxy.Send(channel, message))
		}()
		debugRecordDownStream(m.channelTypeName, channel+postfix, m.codecTypeName, result)
		return result
	})
}

func (m *profiledBinaryMessenger) Send(channel string, message []byte) async.Future[[]byte] {
	return m.sendWithPostfix(channel, "", message)
}

func (m *profiledBinaryMessenger) SetMessageHandler(channel string, handler MessageHandler) {
	m.proxy.SetMessageHandler(channel, handler)
}

// platformChannelStats is _PlatformChannelStats.
type platformChannelStats struct {
	channel, codec, typ  string
	upCount, upBytes     int
	downCount, downBytes int
}

func (s *platformChannelStats) addUpStream(bytes int) {
	s.upCount++
	s.upBytes += bytes
}

func (s *platformChannelStats) addDownStream(bytes int) {
	s.downCount++
	s.downBytes += bytes
}

func (s *platformChannelStats) averageUpPayload() float64 {
	return float64(s.upBytes) / float64(s.upCount)
}

func (s *platformChannelStats) averageDownPayload() float64 {
	return float64(s.downBytes) / float64(s.downCount)
}

// profilePlatformChannelsStats is a Dart Map (insertion-ordered): the
// stats in the order their channels were first used.
var profilePlatformChannelsStats []*platformChannelStats

func debugLaunchProfilePlatformChannels() {
	async.AsyncVoid(func() {
		if profilePlatformChannelsIsRunning {
			return
		}
		profilePlatformChannelsIsRunning = true
		async.FutureDelayed(profilePlatformChannelsRate, func() any { return nil }).Await()
		profilePlatformChannelsIsRunning = false
		log := "Platform Channel Stats:\n"
		allStats := slices.Clone(profilePlatformChannelsStats)
		// Sort highest combined bandwidth first (Dart's sort is an insertion
		// sort, so stable, for up to 32 channels).
		slices.SortStableFunc(allStats, func(x, y *platformChannelStats) int {
			return (y.upBytes + y.downBytes) - (x.upBytes + x.downBytes)
		})
		for _, stats := range allStats {
			log += `  (name:"` + stats.channel + `" type:"` + stats.typ + `" codec:"` + stats.codec +
				`" upBytes:` + strconv.Itoa(stats.upBytes) +
				` upBytes_avg:` + dart.ToStringAsFixed(stats.averageUpPayload(), 1) +
				` downBytes:` + strconv.Itoa(stats.downBytes) +
				` downBytes_avg:` + dart.ToStringAsFixed(stats.averageDownPayload(), 1) + ")\n"
		}
		foundation.DebugPrint(log)
		profilePlatformChannelsStats = nil
	})
}

func statsFor(channelTypeName, name, codecTypeName string) *platformChannelStats {
	for _, s := range profilePlatformChannelsStats {
		if s.channel == name {
			return s
		}
	}
	s := &platformChannelStats{channel: name, codec: codecTypeName, typ: channelTypeName}
	profilePlatformChannelsStats = append(profilePlatformChannelsStats, s)
	return s
}

func debugRecordUpStream(channelTypeName, name, codecTypeName string, bytes []byte) {
	statsFor(channelTypeName, name, codecTypeName).addUpStream(len(bytes))
	debugLaunchProfilePlatformChannels()
}

func debugRecordDownStream(channelTypeName, name, codecTypeName string, bytes []byte) {
	statsFor(channelTypeName, name, codecTypeName).addDownStream(len(bytes))
	debugLaunchProfilePlatformChannels()
}

// findBinaryMessenger is _findBinaryMessenger: the binding's messenger on
// the root isolate (always, in Go), BackgroundIsolateBinaryMessenger's on
// a background isolate.
func findBinaryMessenger() BinaryMessenger {
	if !foundation.KIsWeb && ServicesBindingRootIsolateToken() == nil {
		return BackgroundIsolateBinaryMessengerInstance()
	}
	return ServicesBindingInstance().DefaultBinaryMessenger()
}

func profiled(channel any, result BinaryMessenger, codec any) BinaryMessenger {
	if !ShouldProfilePlatformChannels() {
		return result
	}
	if m, ok := profiledBinaryMessengers[channel]; ok {
		return m
	}
	m := &profiledBinaryMessenger{proxy: result, channelTypeName: foundation.ObjectRuntimeType(channel, ""),
		codecTypeName: foundation.ObjectRuntimeType(codec, "")}
	profiledBinaryMessengers[channel] = m
	return m
}

// BasicMessageChannel is a named channel for communicating with platform
// plugins using asynchronous message passing, encoded with Codec. Use it by
// pointer.
type BasicMessageChannel[T any] struct {
	// Name is the logical channel on which communication happens.
	Name string
	// Codec is the message codec used by this channel.
	Codec           MessageCodec[T]
	binaryMessenger BinaryMessenger
}

// BasicMessageChannelOptions are BasicMessageChannel's optional named
// parameters: the messenger (nil: the binding's default messenger).
type BasicMessageChannelOptions struct {
	BinaryMessenger BinaryMessenger
}

// NewBasicMessageChannel is BasicMessageChannel(name, codec,
// {binaryMessenger}).
func NewBasicMessageChannel[T any](name string, codec MessageCodec[T], opts ...BasicMessageChannelOptions) *BasicMessageChannel[T] {
	c := &BasicMessageChannel[T]{Name: name, Codec: codec}
	if len(opts) > 0 {
		c.binaryMessenger = opts[0].BinaryMessenger
	}
	return c
}

// BinaryMessenger is the messenger which sends the bytes for this channel.
func (c *BasicMessageChannel[T]) BinaryMessenger() BinaryMessenger {
	result := c.binaryMessenger
	if result == nil {
		result = findBinaryMessenger()
	}
	return profiled(c, result, c.Codec)
}

// Send sends message to the platform plugins on this channel; the future
// completes with the decoded reply.
func (c *BasicMessageChannel[T]) Send(message T) async.Future[T] {
	return async.Async(func() T {
		return c.Codec.DecodeMessage(awaitOrNull(c.BinaryMessenger().Send(c.Name, c.Codec.EncodeMessage(message))))
	})
}

// SetMessageHandler sets a callback for receiving messages from the
// platform plugins on this channel; its result is encoded and sent back.
// nil removes the handler.
func (c *BasicMessageChannel[T]) SetMessageHandler(handler func(message T) async.Future[T]) {
	if handler == nil {
		c.BinaryMessenger().SetMessageHandler(c.Name, nil)
		return
	}
	c.BinaryMessenger().SetMessageHandler(c.Name, func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			return c.Codec.EncodeMessage(awaitOrNull(handler(c.Codec.DecodeMessage(message))))
		})
	})
}

// MethodChannel is a named channel for communicating with platform plugins
// using asynchronous method calls, encoded with Codec. Use it by pointer.
type MethodChannel struct {
	// Name is the logical channel on which communication happens.
	Name string
	// Codec is the message codec used by this channel.
	Codec           MethodCodec
	binaryMessenger BinaryMessenger
	// optional is set by NewOptionalMethodChannel: OptionalMethodChannel
	// overrides invokeMethod, and a MethodChannel-typed variable holding an
	// OptionalMethodChannel (SystemChannels.Platform...) dispatches to it.
	optional bool
}

// MethodChannelOptions are MethodChannel's optional positional parameters:
// the codec (nil: StandardMethodCodec) and the messenger (nil: the
// binding's default messenger).
type MethodChannelOptions struct {
	Codec           MethodCodec
	BinaryMessenger BinaryMessenger
}

// NewMethodChannel is MethodChannel(name, [codec = StandardMethodCodec(),
// binaryMessenger]).
func NewMethodChannel(name string, opts ...MethodChannelOptions) *MethodChannel {
	c := &MethodChannel{Name: name, Codec: StandardMethodCodec{}}
	if len(opts) > 0 {
		if opts[0].Codec != nil {
			c.Codec = opts[0].Codec
		}
		c.binaryMessenger = opts[0].BinaryMessenger
	}
	return c
}

// BinaryMessenger is the messenger which sends the bytes for this channel.
func (c *MethodChannel) BinaryMessenger() BinaryMessenger {
	result := c.binaryMessenger
	if result == nil {
		result = findBinaryMessenger()
	}
	return profiled(c, result, c.Codec)
}

func (c *MethodChannel) invokeMethod(method string, missingOk bool, arguments any) async.Future[any] {
	return async.Async(func() any {
		input := c.Codec.EncodeMethodCall(MethodCall{Method: method, Arguments: arguments})
		var result []byte
		if ShouldProfilePlatformChannels() {
			result = c.BinaryMessenger().(*profiledBinaryMessenger).sendWithPostfix(c.Name, "#"+method, input).Await()
		} else {
			result = awaitOrNull(c.BinaryMessenger().Send(c.Name, input))
		}
		if result == nil {
			if missingOk {
				return nil
			}
			panic(NewMissingPluginException("No implementation found for method " + method + " on channel " + c.Name))
		}
		return c.Codec.DecodeEnvelope(result)
	})
}

func optionalArgument(arguments []any) any {
	if len(arguments) > 0 {
		return arguments[0]
	}
	return nil
}

// InvokeMethod invokes method on this channel with the specified arguments
// (Dart's optional positional [arguments]; nil when left out). The future
// completes with the result, or with a *PlatformException if the platform
// plugin failed, or a *MissingPluginException if it isn't implemented (an
// OptionalMethodChannel completes with nil instead). Dart's type argument
// (`invokeMethod<T>`, a cast of the result) is the caller's type assertion.
func (c *MethodChannel) InvokeMethod(method string, arguments ...any) async.Future[any] {
	return c.invokeMethod(method, c.optional, optionalArgument(arguments))
}

// MethodChannelInvokeListMethod is MethodChannel.invokeListMethod<T>: an
// InvokeMethod whose result (a list, or nil) is converted to []T. Dart
// casts the list lazily (an element of another type fails when read); Go
// converts it at once, so a wrong element fails the future.
func MethodChannelInvokeListMethod[T any](c *MethodChannel, method string, arguments ...any) async.Future[[]T] {
	return async.Async(func() []T {
		result := c.InvokeMethod(method, arguments...).Await()
		if result == nil {
			return nil
		}
		list := result.([]any)
		out := make([]T, len(list))
		for i, v := range list {
			out[i] = castElement[T](v)
		}
		return out
	})
}

// MethodChannelInvokeMapMethod is MethodChannel.invokeMapMethod<K, V>: an
// InvokeMethod whose result (a map, or nil) is cast to Map<K, V>. The
// result is the decoded *dart.LinkedMap itself, in the platform's order;
// Dart casts its entries lazily (one of another type fails when read), Go
// checks every key and value against K and V at once, so a wrong entry
// fails the future.
func MethodChannelInvokeMapMethod[K, V any](c *MethodChannel, method string, arguments ...any) async.Future[*dart.LinkedMap] {
	return async.Async(func() *dart.LinkedMap {
		result := c.InvokeMethod(method, arguments...).Await()
		if result == nil {
			return nil
		}
		m, ok := result.(*dart.LinkedMap)
		if !ok {
			panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(result, "Object") +
				"' is not a subtype of type 'Map<dynamic, dynamic>?' in type cast"})
		}
		for k, v := range m.Entries() {
			castElement[K](k)
			castElement[V](v)
		}
		return m
	})
}

// castElement is Dart's `as T` of a decoded value (nil is T's zero value,
// Dart's null).
func castElement[T any](v any) T {
	if v == nil {
		var zero T
		return zero
	}
	t, ok := v.(T)
	if !ok {
		var zero T
		panic(dart.TypeError{Message: "type '" + foundation.ObjectRuntimeType(v, "Object") +
			"' is not a subtype of type '" + foundation.ObjectRuntimeType(&zero, "Object") + "' in type cast"})
	}
	return t
}

// SetMethodCallHandler sets a callback for receiving method calls on this
// channel; nil removes it. The handler's result is sent back as a success
// envelope; a *PlatformException becomes an error envelope, a
// *MissingPluginException a null reply (not implemented), and any other
// error an error envelope with code "error" and the error's text. In debug
// builds it panics with an AssertionError if neither a messenger was given
// nor a binding initialized.
func (c *MethodChannel) SetMethodCallHandler(handler func(call MethodCall) async.Future[any]) {
	if c.binaryMessenger == nil && foundation.BindingBaseDebugBindingType() == nil {
		panic(dart.NewAssertionError("Cannot set the method call handler before the binary messenger has been initialized. " +
			"This happens when you call setMethodCallHandler() before the WidgetsFlutterBinding " +
			"has been initialized. You can fix this by either calling WidgetsFlutterBinding.ensureInitialized() " +
			"before this or by passing a custom BinaryMessenger instance to MethodChannel()."))
	}
	if handler == nil {
		c.BinaryMessenger().SetMessageHandler(c.Name, nil)
		return
	}
	c.BinaryMessenger().SetMessageHandler(c.Name, func(message []byte) async.Future[[]byte] {
		return c.handleAsMethodCall(message, handler)
	})
}

func (c *MethodChannel) handleAsMethodCall(message []byte, handler func(call MethodCall) async.Future[any]) async.Future[[]byte] {
	return async.Async(func() []byte {
		call := c.Codec.DecodeMethodCall(message)
		return encodeMethodCallResult(c.Codec, func() any { return awaitOrNull(handler(call)) })
	})
}

// encodeMethodCallResult runs a method call handler's body inside a Dart
// try/catch and encodes the outcome: the result as a success envelope, a
// *PlatformException as an error envelope (code, message, details), a
// *MissingPluginException as nil, any other panic as an error envelope with
// code "error" and the error's toString.
func encodeMethodCallResult(codec MethodCodec, body func() any) (envelope []byte) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		var platform *PlatformException
		var missing *MissingPluginException
		err, isError := r.(error)
		switch {
		case isError && errors.As(err, &platform):
			envelope = codec.EncodeErrorEnvelope(platform.Code,
				EncodeErrorEnvelopeOptions{Message: platform.Message, Details: platform.Details})
		case isError && errors.As(err, &missing):
			envelope = nil
		default:
			message := dart.ToString(r)
			envelope = codec.EncodeErrorEnvelope("error", EncodeErrorEnvelopeOptions{Message: &message})
		}
	}()
	return codec.EncodeSuccessEnvelope(body())
}

// OptionalMethodChannel is a MethodChannel that ignores missing platform
// plugins: when a method isn't implemented, InvokeMethod completes with nil
// instead of a *MissingPluginException. Use it by pointer; its embedded
// MethodChannel (&c.MethodChannel) behaves the same.
type OptionalMethodChannel struct {
	MethodChannel
}

// NewOptionalMethodChannel is OptionalMethodChannel(name, [codec,
// binaryMessenger]).
func NewOptionalMethodChannel(name string, opts ...MethodChannelOptions) *OptionalMethodChannel {
	c := &OptionalMethodChannel{MethodChannel: *NewMethodChannel(name, opts...)}
	c.optional = true
	return c
}

// EventChannel is a named channel for communicating with platform plugins
// using event streams. Use it by pointer.
type EventChannel struct {
	// Name is the logical channel on which communication happens.
	Name string
	// Codec is the message codec used by this channel.
	Codec           MethodCodec
	binaryMessenger BinaryMessenger
}

// NewEventChannel is EventChannel(name, [codec = StandardMethodCodec(),
// binaryMessenger]).
func NewEventChannel(name string, opts ...MethodChannelOptions) *EventChannel {
	c := &EventChannel{Name: name, Codec: StandardMethodCodec{}}
	if len(opts) > 0 {
		if opts[0].Codec != nil {
			c.Codec = opts[0].Codec
		}
		c.binaryMessenger = opts[0].BinaryMessenger
	}
	return c
}

// BinaryMessenger is the messenger used by this channel to send platform
// messages (not profiled, as in Flutter).
func (c *EventChannel) BinaryMessenger() BinaryMessenger {
	if c.binaryMessenger != nil {
		return c.binaryMessenger
	}
	return findBinaryMessenger()
}

// ReceiveBroadcastStream sets up a broadcast stream for receiving events on
// this channel from the platform plugin, with optional arguments (Dart's
// [arguments]). Events are decoded as success envelopes; error envelopes
// become *PlatformException error events; a null message closes the
// stream. The platform is sent "listen" when the first listener comes and
// "cancel" when the last one leaves; their failures are reported to
// foundation.FlutterErrorReportError.
func (c *EventChannel) ReceiveBroadcastStream(arguments ...any) async.Stream[any] {
	args := optionalArgument(arguments)
	methodChannel := NewMethodChannel(c.Name, MethodChannelOptions{Codec: c.Codec})
	var controller *async.BroadcastStreamController[any]
	controller = async.NewStreamControllerBroadcast[any](async.StreamControllerBroadcastOptions{
		OnListen: func() {
			async.AsyncVoid(func() {
				c.BinaryMessenger().SetMessageHandler(c.Name, func(reply []byte) async.Future[[]byte] {
					return async.Async(func() []byte {
						if reply == nil {
							controller.Close().Await()
						} else {
							func() {
								defer func() {
									if r := recover(); r != nil {
										var platform *PlatformException
										if err, ok := r.(error); ok && errors.As(err, &platform) {
											controller.AddError(platform)
											return
										}
										panic(r)
									}
								}()
								controller.Add(c.Codec.DecodeEnvelope(reply))
							}()
						}
						return nil
					})
				})
				reportPlatformStreamError(func() { methodChannel.InvokeMethod("listen", args).Await() },
					"while activating platform stream on channel "+c.Name)
			})
		},
		OnCancel: func() {
			async.AsyncVoid(func() {
				c.BinaryMessenger().SetMessageHandler(c.Name, nil)
				reportPlatformStreamError(func() { methodChannel.InvokeMethod("cancel", args).Await() },
					"while de-activating platform stream on channel "+c.Name)
			})
		},
	})
	return controller.Stream()
}

// reportPlatformStreamError runs body, reporting what it throws to
// FlutterErrorReportError with the services library and context.
func reportPlatformStreamError(body func(), context string) {
	defer func() {
		if exception := recover(); exception != nil {
			stack := dart.StackTraceFromPanic()
			library := "services library"
			foundation.FlutterErrorReportError(foundation.NewFlutterErrorDetails(exception, foundation.FlutterErrorDetailsOptions{
				Stack:   stack,
				Library: &library,
				Context: foundation.NewErrorDescription(context),
			}))
		}
	}()
	body()
}
