package flutter_test

// Port of flutter_test's test_default_binary_messenger.dart (Flutter
// 3.50.1), plus the TestDefaultBinaryMessengerBinding mixin of binding.dart
// and an AddTearDown for setMockStreamHandler (see SetCurrentTest).

import (
	"errors"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

// AllMessagesHandler is the signature of
// TestDefaultBinaryMessenger.AllMessagesHandler: it gets every outgoing
// message (with the mock handler registered for its channel, if any) and
// returns the response.
type AllMessagesHandler func(channel string, handler services.MessageHandler, message []byte) async.Future[[]byte]

// TestDefaultBinaryMessenger is a BinaryMessenger subclass that is used as
// the default binary messenger under testing environment: it tracks
// pending messages and lets tests mock the platform's side of channels
// (SetMockMessageHandler...) and send fake platform messages
// (HandlePlatformMessage). Use it by pointer.
type TestDefaultBinaryMessenger struct {
	// Delegate is the messenger that this object wraps (the binding's
	// default messenger).
	Delegate services.BinaryMessenger

	// AllMessagesHandler, when set, intercepts and responds to every
	// outgoing message, instead of the mock handlers and the delegate.
	AllMessagesHandler AllMessagesHandler

	// The handlers for messages from the engine (including fake messages
	// sent by HandlePlatformMessage).
	inboundHandlers map[string]services.MessageHandler
	pendingMessages []async.Future[[]byte]
	// Handlers that intercept and respond to outgoing messages, pretending
	// to be the platform.
	outboundHandlers map[string]services.MessageHandler
	// The outbound callbacks that were actually registered, so that
	// CheckMockMessageHandler can check them.
	outboundHandlerIdentities map[string]any
}

// TestDefaultBinaryMessengerOptions are TestDefaultBinaryMessenger's
// optional named parameters.
type TestDefaultBinaryMessengerOptions struct {
	// OutboundHandlers are mock handlers to start with.
	OutboundHandlers map[string]services.MessageHandler
}

// NewTestDefaultBinaryMessenger is TestDefaultBinaryMessenger(delegate,
// {outboundHandlers}).
func NewTestDefaultBinaryMessenger(delegate services.BinaryMessenger, opts ...TestDefaultBinaryMessengerOptions) *TestDefaultBinaryMessenger {
	m := &TestDefaultBinaryMessenger{
		Delegate:                  delegate,
		inboundHandlers:           map[string]services.MessageHandler{},
		outboundHandlers:          map[string]services.MessageHandler{},
		outboundHandlerIdentities: map[string]any{},
	}
	if len(opts) > 0 {
		for channel, handler := range opts[0].OutboundHandlers {
			m.outboundHandlers[channel] = handler
		}
	}
	return m
}

// HandlePlatformMessage sends a fake message to the handler registered for
// channel (with SetMessageHandler), as if the platform had sent it, and
// calls callback (if not nil) with the handler's response. The returned
// future completes with the response.
func (m *TestDefaultBinaryMessenger) HandlePlatformMessage(channel string, data []byte, callback ui.PlatformMessageResponseCallback) async.Future[[]byte] {
	var result async.Future[[]byte]
	if handler, ok := m.inboundHandlers[channel]; ok {
		result = handler(data)
	}
	if result == nil {
		result = async.FutureValue[[]byte](nil)
	}
	if callback != nil {
		result = async.Then(result, func(result []byte) []byte {
			callback(result)
			return result
		})
	}
	return result
}

// SetMessageHandler sets the handler for messages from the platform on
// channel (both real ones from the engine, through the delegate, and fake
// ones from HandlePlatformMessage); nil removes it.
func (m *TestDefaultBinaryMessenger) SetMessageHandler(channel string, handler services.MessageHandler) {
	if handler == nil {
		delete(m.inboundHandlers, channel)
		m.Delegate.SetMessageHandler(channel, nil)
	} else {
		m.inboundHandlers[channel] = handler           // used to handle fake messages sent via handlePlatformMessage
		m.Delegate.SetMessageHandler(channel, handler) // used to handle real messages from the engine
	}
}

// PendingMessageCount is the number of incomplete futures created by Send.
func (m *TestDefaultBinaryMessenger) PendingMessageCount() int { return len(m.pendingMessages) }

// Send sends message on channel to the AllMessagesHandler, else to the mock
// handler registered for channel, else to the delegate (the engine); the
// response future is tracked until it completes.
func (m *TestDefaultBinaryMessenger) Send(channel string, message []byte) async.Future[[]byte] {
	var resultFuture async.Future[[]byte]
	handler := m.outboundHandlers[channel]
	if m.AllMessagesHandler != nil {
		resultFuture = m.AllMessagesHandler(channel, handler, message)
	} else if handler != nil {
		resultFuture = handler(message)
	} else {
		resultFuture = m.Delegate.Send(channel, message)
	}
	if resultFuture != nil {
		m.pendingMessages = append(m.pendingMessages, resultFuture)
		resultFuture.
			CatchError(func(error) []byte { /* errors are the responsibility of the caller */ return nil }, nil).
			WhenComplete(func() { m.removePending(resultFuture) })
	}
	return resultFuture
}

func (m *TestDefaultBinaryMessenger) removePending(f async.Future[[]byte]) {
	for i, p := range m.pendingMessages {
		if p == f {
			m.pendingMessages = append(m.pendingMessages[:i], m.pendingMessages[i+1:]...)
			return
		}
	}
}

// PlatformMessagesFinished completes when all the futures created by Send
// have completed (Future.wait: if one fails, it fails with the first
// error, once all have completed).
func (m *TestDefaultBinaryMessenger) PlatformMessagesFinished() async.Future[async.Void] {
	pending := append([]async.Future[[]byte](nil), m.pendingMessages...)
	return async.AsyncVoid(func() {
		var first error
		for _, f := range pending {
			func() {
				defer func() {
					if r := recover(); r != nil && first == nil {
						if err, ok := r.(error); ok {
							first = err
						} else {
							first = async.PanicError{Value: r}
						}
					}
				}()
				f.Await()
			}()
		}
		if first != nil {
			panic(first)
		}
	})
}

// SetMockMessageHandler sets a callback that intercepts outgoing messages
// on channel, pretending to be the platform; nil removes it. identity
// (Dart's optional positional [identity], the handler by default) is what
// CheckMockMessageHandler compares with.
func (m *TestDefaultBinaryMessenger) SetMockMessageHandler(channel string, handler services.MessageHandler, identity ...any) {
	if handler == nil {
		delete(m.outboundHandlers, channel)
		delete(m.outboundHandlerIdentities, channel)
		return
	}
	var id any = handler
	if len(identity) > 0 && identity[0] != nil {
		id = identity[0]
	}
	m.outboundHandlers[channel] = handler
	m.outboundHandlerIdentities[channel] = id
}

// TestDefaultBinaryMessengerSetMockDecodedMessageHandler is
// TestDefaultBinaryMessenger.setMockDecodedMessageHandler<T>: a mock
// handler for channel that receives decoded messages and returns the
// response to encode; nil removes it.
func TestDefaultBinaryMessengerSetMockDecodedMessageHandler[T any](m *TestDefaultBinaryMessenger, channel *services.BasicMessageChannel[T], handler func(message T) async.Future[T]) {
	if handler == nil {
		m.SetMockMessageHandler(channel.Name, nil)
		return
	}
	m.SetMockMessageHandler(channel.Name, func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			return channel.Codec.EncodeMessage(handler(channel.Codec.DecodeMessage(message)).Await())
		})
	}, handler)
}

// SetMockMethodCallHandler sets a mock handler for method calls on
// channel; nil removes it. The handler's result is sent back as a success
// envelope; a *services.PlatformException becomes an error envelope, a
// *services.MissingPluginException a null reply, any other error an error
// envelope with code "error". The handler may return a nil future (null).
func (m *TestDefaultBinaryMessenger) SetMockMethodCallHandler(channel *services.MethodChannel, handler func(message services.MethodCall) async.Future[any]) {
	if handler == nil {
		m.SetMockMessageHandler(channel.Name, nil)
		return
	}
	m.SetMockMessageHandler(channel.Name, func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			call := channel.Codec.DecodeMethodCall(message)
			return encodeMethodCallResult(channel.Codec, func() any {
				f := handler(call)
				if f == nil {
					f = async.FutureValue[any](nil)
				}
				return f.Await()
			})
		})
	}, handler)
}

// encodeMethodCallResult runs body in Dart's try/catch of
// setMockMethodCallHandler and encodes the outcome.
func encodeMethodCallResult(codec services.MethodCodec, body func() any) (envelope []byte) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		var platform *services.PlatformException
		var missing *services.MissingPluginException
		err, isError := r.(error)
		switch {
		case isError && errors.As(err, &platform):
			envelope = codec.EncodeErrorEnvelope(platform.Code,
				services.EncodeErrorEnvelopeOptions{Message: platform.Message, Details: platform.Details})
		case isError && errors.As(err, &missing):
			envelope = nil
		default:
			message := dart.ToString(r)
			envelope = codec.EncodeErrorEnvelope("error", services.EncodeErrorEnvelopeOptions{Message: &message})
		}
	}()
	return codec.EncodeSuccessEnvelope(body())
}

// SetMockStreamHandler sets a mock stream handler for channel: the
// platform's side of an EventChannel ("listen" and "cancel" go to handler,
// and the events it sends reach the channel's stream); nil removes it. The
// stream it creates is closed when the current test ends (AddTearDown).
func (m *TestDefaultBinaryMessenger) SetMockStreamHandler(channel *services.EventChannel, handler MockStreamHandler) {
	if handler == nil {
		m.SetMockMessageHandler(channel.Name, nil)
		return
	}
	controller := async.NewStreamController[any]()
	AddTearDown(func() { controller.Close() })
	m.SetMockMethodCallHandler(services.NewMethodChannel(channel.Name, services.MethodChannelOptions{Codec: channel.Codec}),
		func(call services.MethodCall) async.Future[any] {
			return async.Async(func() any {
				switch call.Method {
				case "listen":
					handler.OnListen(call.Arguments, NewMockStreamHandlerEventSink(controller))
				case "cancel":
					handler.OnCancel(call.Arguments)
				default:
					panic(dart.UnimplementedError{Message: "Method " + call.Method + " not implemented"})
				}
				return nil
			})
		})
	sub := controller.Stream().Listen(func(e any) {
		m.HandlePlatformMessage(channel.Name, channel.Codec.EncodeSuccessEnvelope(e), nil)
	}, async.StreamListenOptions{
		OnError: func(e error) {
			var platform *services.PlatformException
			if !errors.As(e, &platform) {
				panic(dart.ArgumentError{Message: "Stream error must be a PlatformException"})
			}
			m.HandlePlatformMessage(channel.Name, channel.Codec.EncodeErrorEnvelope(platform.Code,
				services.EncodeErrorEnvelopeOptions{Message: platform.Message, Details: platform.Details}), nil)
		},
		OnDone: func() { m.HandlePlatformMessage(channel.Name, nil, nil) },
	})
	AddTearDown(func() { sub.Cancel() })
}

// CheckMockMessageHandler reports whether handler (compared with
// dart.FunctionsEqual when it is a function, with == otherwise) is the
// identity of the mock handler registered for channel (nil when none).
func (m *TestDefaultBinaryMessenger) CheckMockMessageHandler(channel string, handler any) bool {
	return identityEqual(m.outboundHandlerIdentities[channel], handler)
}

// identityEqual is Dart's == on the identity objects (functions compare as
// Dart's tear-offs do).
func identityEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if fa, ok := a.(services.MessageHandler); ok {
		fb, ok := b.(services.MessageHandler)
		return ok && dart.FunctionsEqual(fa, fb)
	}
	if fa, ok := a.(func(services.MethodCall) async.Future[any]); ok {
		fb, ok := b.(func(services.MethodCall) async.Future[any])
		return ok && dart.FunctionsEqual(fa, fb)
	}
	defer func() { recover() }() // uncomparable values are unequal
	return a == b
}

// ---- mock_event_channel.dart

// MockStreamHandler is a mock stream handler for an EventChannel
// (TestDefaultBinaryMessenger.SetMockStreamHandler).
type MockStreamHandler interface {
	// OnListen handles the "listen" call: events sent to events reach the
	// channel's stream.
	OnListen(arguments any, events *MockStreamHandlerEventSink)
	// OnCancel handles the "cancel" call.
	OnCancel(arguments any)
}

// MockStreamHandlerOnListenCallback is the signature of
// MockStreamHandler.OnListen for NewMockStreamHandlerInline.
type MockStreamHandlerOnListenCallback func(arguments any, events *MockStreamHandlerEventSink)

// MockStreamHandlerOnCancelCallback is the signature of
// MockStreamHandler.OnCancel for NewMockStreamHandlerInline.
type MockStreamHandlerOnCancelCallback func(arguments any)

// MockStreamHandlerInlineOptions are MockStreamHandler.inline's optional
// named parameters.
type MockStreamHandlerInlineOptions struct {
	OnCancel MockStreamHandlerOnCancelCallback
}

// NewMockStreamHandlerInline is MockStreamHandler.inline(onListen:,
// onCancel:) (the required onListen is positional).
func NewMockStreamHandlerInline(onListen MockStreamHandlerOnListenCallback, opts ...MockStreamHandlerInlineOptions) MockStreamHandler {
	h := &inlineMockStreamHandler{onListen: onListen}
	if len(opts) > 0 {
		h.onCancel = opts[0].OnCancel
	}
	return h
}

// inlineMockStreamHandler is _InlineMockStreamHandler.
type inlineMockStreamHandler struct {
	onListen MockStreamHandlerOnListenCallback
	onCancel MockStreamHandlerOnCancelCallback
}

func (h *inlineMockStreamHandler) OnListen(arguments any, events *MockStreamHandlerEventSink) {
	h.onListen(arguments, events)
}

func (h *inlineMockStreamHandler) OnCancel(arguments any) {
	if h.onCancel != nil {
		h.onCancel(arguments)
	}
}

// MockStreamHandlerEventSink is the sink a MockStreamHandler sends its
// events to.
type MockStreamHandlerEventSink struct {
	sink *async.StreamController[any]
}

// NewMockStreamHandlerEventSink is MockStreamHandlerEventSink(sink).
func NewMockStreamHandlerEventSink(sink *async.StreamController[any]) *MockStreamHandlerEventSink {
	return &MockStreamHandlerEventSink{sink: sink}
}

// Success sends a success event.
func (s *MockStreamHandlerEventSink) Success(event any) { s.sink.Add(event) }

// Error sends an error event (a *services.PlatformException; Dart's
// required named `code:` is positional).
func (s *MockStreamHandlerEventSink) Error(code string, opts ...services.PlatformExceptionOptions) {
	var o services.PlatformExceptionOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	s.sink.AddError(services.NewPlatformException(code, services.PlatformExceptionOptions{Message: o.Message, Details: o.Details}))
}

// EndOfStream ends the stream.
func (s *MockStreamHandlerEventSink) EndOfStream() { s.sink.Close() }

// ---- addTearDown

var currentTest testing.TB

// SetCurrentTest makes t the test AddTearDown registers with, until t
// ends. (Dart's test framework knows the running test; Go's doesn't, so a
// test calls this first, as flutter_test's testWidgets/test wrappers would.)
func SetCurrentTest(t testing.TB) {
	previous := currentTest
	currentTest = t
	t.Cleanup(func() { currentTest = previous })
}

// AddTearDown registers callback to run after the current test (package
// test_api's addTearDown, which flutter_test re-exports); it panics with a
// StateError outside a test.
func AddTearDown(callback func()) {
	if currentTest == nil {
		panic(dart.StateError{Message: "addTearDown() may only be called within a test."})
	}
	currentTest.Cleanup(callback)
}

// ---- TestDefaultBinaryMessengerBinding (binding.dart)

// TestDefaultBinaryMessengerBinding is a ServicesBinding whose default
// messenger is a *TestDefaultBinaryMessenger (mixin
// TestDefaultBinaryMessengerBinding on BindingBase, ServicesBinding).
// Dart's covariant `TestDefaultBinaryMessenger get defaultBinaryMessenger`
// is DefaultBinaryMessenger().(*TestDefaultBinaryMessenger).
type TestDefaultBinaryMessengerBinding interface {
	services.ServicesBinding
	testDefaultBinaryMessengerBinding() *TestDefaultBinaryMessengerBindingMixin
}

// TestDefaultBinaryMessengerBindingMixin is the mixin's state and methods;
// its embedded services.ServicesBinding is the binding it is mixed onto
// (see services.ServicesBindingMixin for the construction).
type TestDefaultBinaryMessengerBindingMixin struct {
	services.ServicesBinding
	self TestDefaultBinaryMessengerBinding
}

// SetSelf points the mixin at the binding embedding it.
func (m *TestDefaultBinaryMessengerBindingMixin) SetSelf(self TestDefaultBinaryMessengerBinding) {
	m.self = self
}

func (m *TestDefaultBinaryMessengerBindingMixin) testDefaultBinaryMessengerBinding() *TestDefaultBinaryMessengerBindingMixin {
	return m
}

var testDefaultBinaryMessengerBindingInstance TestDefaultBinaryMessengerBinding

// TestDefaultBinaryMessengerBindingInstance is
// TestDefaultBinaryMessengerBinding.instance.
func TestDefaultBinaryMessengerBindingInstance() TestDefaultBinaryMessengerBinding {
	return foundation.BindingBaseCheckInstance(testDefaultBinaryMessengerBindingInstance)
}

func (m *TestDefaultBinaryMessengerBindingMixin) InitInstances() {
	m.ServicesBinding.InitInstances()
	testDefaultBinaryMessengerBindingInstance = m.self
}

// CreateBinaryMessenger wraps the default messenger in a
// TestDefaultBinaryMessenger that answers flutter/keyboard's
// getKeyboardState with an empty map.
func (m *TestDefaultBinaryMessengerBindingMixin) CreateBinaryMessenger() services.BinaryMessenger {
	keyboardHandler := func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			return services.StandardMethodCodec{}.EncodeSuccessEnvelope(map[int]int{})
		})
	}
	return NewTestDefaultBinaryMessenger(m.ServicesBinding.CreateBinaryMessenger(), TestDefaultBinaryMessengerOptions{
		OutboundHandlers: map[string]services.MessageHandler{"flutter/keyboard": keyboardHandler},
	})
}
