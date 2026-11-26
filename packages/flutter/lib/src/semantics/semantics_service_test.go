package semantics_test

// Port of packages/flutter/test/semantics/semantics_service_test.dart and
// semantics_node_send_event_test.dart (Flutter 3.50.1), plus the events'
// toMap/toString.
//
// Adapted: flutter_test's TestDefaultBinaryMessenger.
// setMockDecodedMessageHandler is mockMessenger below, installed as
// services.DefaultBinaryMessenger (the stand-in for ServicesBinding.
// instance.defaultBinaryMessenger until ServicesBinding is ported);
// tester.view is the implicit view. pumpEventQueue is running the loop's
// microtasks.

import (
	"errors"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/semantics"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type semanticsCodec struct{}

func (semanticsCodec) encode(v any) []byte { return services.StandardMessageCodec{}.EncodeMessage(v) }

// mockMessenger is a BinaryMessenger whose channels have mock handlers that
// receive decoded messages (StandardMessageCodec).
type mockMessenger struct {
	handlers map[string]func(message any) async.Future[any]
}

func (m *mockMessenger) HandlePlatformMessage(string, []byte, func([]byte)) async.Future[async.Void] {
	return async.FutureValue(async.Void{})
}

func (m *mockMessenger) Send(channel string, message []byte) async.Future[[]byte] {
	handler := m.handlers[channel]
	if handler == nil {
		return nil
	}
	codec := services.StandardMessageCodec{}
	return async.Async(func() []byte {
		return codec.EncodeMessage(handler(codec.DecodeMessage(message)).Await())
	})
}

func (m *mockMessenger) SetMessageHandler(string, services.MessageHandler) {}

func installMockMessenger(t *testing.T, channel string, handler func(message any) async.Future[any]) {
	t.Helper()
	prev := services.DefaultBinaryMessenger
	services.DefaultBinaryMessenger = &mockMessenger{handlers: map[string]func(any) async.Future[any]{channel: handler}}
	t.Cleanup(func() { services.DefaultBinaryMessenger = prev })
}

func TestSemanticAnnouncement(t *testing.T) {
	ensureBinding(t)
	loop := async.NewLoop()
	t.Cleanup(async.SetCurrent(loop))
	var log []any
	installMockMessenger(t, "flutter/accessibility", func(message any) async.Future[any] {
		log = append(log, message)
		return async.FutureValue[any](nil)
	})
	view := ui.PlatformDispatcherInstance().ImplicitView()
	semantics.SemanticsService.SendAnnouncement(view, "announcement 1", ui.TextDirectionLtr)
	semantics.SemanticsService.SendAnnouncement(view, "announcement 2", ui.TextDirectionRtl,
		semantics.SemanticsServiceAnnounceOptions{Assertiveness: semantics.AssertivenessAssertive})
	loop.FlushMicrotasks()
	expectEqual(t, len(log), 2)
	expectString(t, dart.JSONEncode(toStringKeyed(log[0])),
		`{"type":"announce","data":{"viewId":0,"message":"announcement 1","textDirection":1}}`)
	expectString(t, dart.JSONEncode(toStringKeyed(log[1])),
		`{"type":"announce","data":{"viewId":0,"message":"announcement 2","textDirection":0,"assertiveness":1}}`)
}

// toStringKeyed turns a decoded map ([]MapEntry[any, any]) into the
// string-keyed ordered map JSONEncode writes.
func toStringKeyed(v any) any {
	m, ok := v.([]dart.MapEntry[any, any])
	if !ok {
		return v
	}
	out := []dart.MapEntry[string, any]{}
	for _, e := range m {
		out = append(out, dart.MapEntry[string, any]{Key: e.Key.(string), Value: toStringKeyed(e.Value)})
	}
	return out
}

func TestSendEventReportsErrorToFlutterError(t *testing.T) {
	ensureBinding(t)
	loop := async.NewLoop()
	t.Cleanup(async.SetCurrent(loop))
	var errorDetails []*foundation.FlutterErrorDetails
	oldHandler := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		errorDetails = append(errorDetails, details)
	}
	defer func() { foundation.FlutterErrorOnError = oldHandler }()

	node := semantics.NewSemanticsNode()
	owner := semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {})
	node.Attach(owner)
	expectEqual(t, node.Attached(), true)

	// Mock accessibility channel to throw error.
	installMockMessenger(t, "flutter/accessibility", func(any) async.Future[any] {
		return async.Async(func() any { panic(errors.New("Exception: Test exception")) })
	})
	node.SendEvent(semantics.NewAnnounceSemanticsEvent("hello", ui.TextDirectionLtr, 0))
	// Wait for the async error handler to run.
	loop.FlushMicrotasks()

	expectEqual(t, len(errorDetails), 1)

	expectEqual(t, contains(errorDetails[0].Exception.(error).Error(), "Test exception"), true)
	expectEqual(t, *errorDetails[0].Library, "semantics library")
	expectEqual(t, contains(errorDetails[0].Context.String(), "while sending accessibility event"), true)
}

func TestSendEventWithoutMessengerReportsError(t *testing.T) {
	// Without a ServicesBinding (no default messenger), Flutter's async send
	// fails, and sendEvent reports it.
	ensureBinding(t)
	loop := async.NewLoop()
	t.Cleanup(async.SetCurrent(loop))
	var reported int
	old := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(*foundation.FlutterErrorDetails) { reported++ }
	defer func() { foundation.FlutterErrorOnError = old }()
	prev := services.DefaultBinaryMessenger
	services.DefaultBinaryMessenger = nil
	defer func() { services.DefaultBinaryMessenger = prev }()

	node := semantics.NewSemanticsNode()
	node.SendEvent(semantics.NewTapSemanticEvent()) // not attached: nothing
	loop.FlushMicrotasks()
	expectEqual(t, reported, 0)
	node.Attach(semantics.NewSemanticsOwner(func(ui.SemanticsUpdate) {}))
	node.SendEvent(semantics.NewTapSemanticEvent())
	loop.FlushMicrotasks()
	expectEqual(t, reported, 1)
}

func TestSemanticsEventToMapAndString(t *testing.T) {
	id := 42
	cases := []struct {
		event     semantics.SemanticsEvent
		toMap     string
		withNode  string
		toStringV string
	}{
		{semantics.NewAnnounceSemanticsEvent("hi", ui.TextDirectionRtl, 3), `{"type":"announce","data":{"viewId":3,"message":"hi","textDirection":0}}`,
			`{"type":"announce","data":{"viewId":3,"message":"hi","textDirection":0},"nodeId":42}`,
			"AnnounceSemanticsEvent(message: hi, textDirection: 0, viewId: 3)"},
		{semantics.NewAnnounceSemanticsEvent("x", ui.TextDirectionLtr, 0, semantics.AnnounceSemanticsEventOptions{Assertiveness: semantics.AssertivenessAssertive}),
			`{"type":"announce","data":{"viewId":0,"message":"x","textDirection":1,"assertiveness":1}}`,
			`{"type":"announce","data":{"viewId":0,"message":"x","textDirection":1,"assertiveness":1},"nodeId":42}`,
			"AnnounceSemanticsEvent(assertiveness: 1, message: x, textDirection: 1, viewId: 0)"},
		{semantics.NewTooltipSemanticsEvent("tip"), `{"type":"tooltip","data":{"message":"tip"}}`,
			`{"type":"tooltip","data":{"message":"tip"},"nodeId":42}`, "TooltipSemanticsEvent(message: tip)"},
		{semantics.NewLongPressSemanticsEvent(), `{"type":"longPress","data":{}}`, `{"type":"longPress","data":{},"nodeId":42}`, "LongPressSemanticsEvent()"},
		{semantics.NewTapSemanticEvent(), `{"type":"tap","data":{}}`, `{"type":"tap","data":{},"nodeId":42}`, "TapSemanticEvent()"},
		{semantics.NewFocusSemanticEvent(), `{"type":"focus","data":{}}`, `{"type":"focus","data":{},"nodeId":42}`, "FocusSemanticEvent()"},
	}
	for _, c := range cases {
		expectString(t, dart.JSONEncode(c.event.ToMap()), c.toMap)
		expectString(t, dart.JSONEncode(c.event.ToMap(semantics.SemanticsEventToMapOptions{NodeID: &id})), c.withNode)
		expectString(t, c.event.String(), c.toStringV)
	}
}
