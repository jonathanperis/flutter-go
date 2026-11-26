package services_test

// Ports of packages/flutter/test/services/{undo_manager,scribe,
// process_text,spell_check,mouse_tracking}_test.dart (Flutter 3.50.1), all
// tests, plus own tests of mouse_cursor.dart (Flutter has no services test
// for it; its widget tests need MouseRegion), live_text.dart and
// spell_check.dart's service.
//
// Adapted: process_text_test's FakeProcessTextChannel implements
// MethodChannel, a Go struct; it is a MethodChannel on a fake
// BinaryMessenger that answers through the same callback.

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/gestures"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

type fakeUndoManagerClient struct{ latestMethodCall string }

func (c *fakeUndoManagerClient) Undo()         {}
func (c *fakeUndoManagerClient) Redo()         {}
func (c *fakeUndoManagerClient) CanUndo() bool { return false }
func (c *fakeUndoManagerClient) CanRedo() bool { return false }
func (c *fakeUndoManagerClient) HandlePlatformUndo(direction services.UndoDirection) {
	c.latestMethodCall = "handlePlatformUndo(" + direction.String() + ")"
}

func TestUndoManagerClientHandleUndo(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		client := &fakeUndoManagerClient{}
		services.UndoManager.SetClient(client)
		defer services.UndoManager.SetClient(nil)
		expect(client.latestMethodCall == "", "latest %q", client.latestMethodCall)
		send := func(direction string) {
			messageBytes := services.JSONMessageCodec{}.EncodeMessage([]dart.MapEntry[string, any]{
				{Key: "args", Value: []any{direction}},
				{Key: "method", Value: "UndoManagerClient.handleUndo"},
			})
			messenger().HandlePlatformMessage("flutter/undomanager", messageBytes, func([]byte) {}).Await()
		}
		send("undo")
		expect(client.latestMethodCall == "handlePlatformUndo(UndoDirection.undo)", "latest %q", client.latestMethodCall)
		send("redo")
		expect(client.latestMethodCall == "handlePlatformUndo(UndoDirection.redo)", "latest %q", client.latestMethodCall)
		// Not in Flutter's test: an unknown direction is an error reply, an
		// unknown method a null reply (MissingPluginException).
		bad := services.JSONMessageCodec{}.EncodeMessage([]dart.MapEntry[string, any]{
			{Key: "args", Value: []any{"sideways"}}, {Key: "method", Value: "UndoManagerClient.handleUndo"},
		})
		reply := messenger().HandlePlatformMessage("flutter/undomanager", bad, nil).Await()
		expect(strings.Contains(string(reply), "Unknown undo direction: sideways"), "reply %q", reply)
		unknown := services.JSONMessageCodec{}.EncodeMessage([]dart.MapEntry[string, any]{
			{Key: "args", Value: []any{}}, {Key: "method", Value: "UndoManagerClient.other"},
		})
		expect(messenger().HandlePlatformMessage("flutter/undomanager", unknown, nil).Await() == nil, "unknown method replied")
	})
}

func TestUndoManagerSetUndoStateReportsErrorWhenChannelFails(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		oldOnError := foundation.FlutterErrorOnError
		var errs []*foundation.FlutterErrorDetails
		errorReceived := async.NewCompleter[async.Void]()
		foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
			errs = append(errs, details)
			if !errorReceived.IsCompleted() {
				errorReceived.Complete(async.Void{})
			}
		}
		defer func() {
			foundation.FlutterErrorOnError = oldOnError
			services.UndoManager.SetChannel(services.SystemChannels.UndoManager)
		}()
		var log []services.MethodCall
		channel := services.NewMethodChannel("flutter/undomanager")
		services.UndoManager.SetChannel(channel)
		messenger().SetMockMethodCallHandler(channel, func(call services.MethodCall) async.Future[any] {
			return async.Async(func() any {
				log = append(log, call)
				panic(dart.NewException("Channel failed"))
			})
		})
		defer messenger().SetMockMethodCallHandler(channel, nil)
		services.UndoManager.SetUndoState(services.UndoManagerSetUndoStateOptions{CanUndo: true, CanRedo: true})
		errorReceived.Future().Await()
		expect(len(errs) == 1, "errors %d", len(errs))
		var platform *services.PlatformException
		exception, isErr := errs[0].Exception.(error)
		expect(isErr && errors.As(exception, &platform), "exception %T", errs[0].Exception)
		expect(strings.Contains(dart.ToString(errs[0].Exception), "Channel failed"), "exception %v", errs[0].Exception)
		expect(strings.Contains(errs[0].Context.String(), "while sending the UndoManager.setUndoState event"), "context %v", errs[0].Context)
		expect(len(log) == 1 && log[0].Method == "UndoManager.setUndoState", "log %v", log)
	})
}

// Not in Flutter's test: the setUndoState message.
func TestUndoManagerSetUndoStateMessage(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var log []services.MethodCall
		messenger().SetMockMethodCallHandler(services.SystemChannels.UndoManager, func(call services.MethodCall) async.Future[any] {
			log = append(log, call)
			return nil
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.UndoManager, nil)
		services.UndoManager.SetUndoState(services.UndoManagerSetUndoStateOptions{CanRedo: true})
		async.FutureDelayed[any](0, nil).Await()
		expect(len(log) == 1 && fmt.Sprint(log[0].Arguments) == "{canUndo: false, canRedo: true}", "log %v", log)
	})
}

func TestScribeUnsupportedMessage(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		messageBytes := services.JSONMessageCodec{}.EncodeMessage([]dart.MapEntry[string, any]{{Key: "method", Value: "Scribe.unsupportedMessage"}})
		response := messenger().HandlePlatformMessage("flutter/scribe", messageBytes, nil).Await()
		expect(response == nil, "response %v", response)
	})
}

func TestScribeCallsThroughToPlatformChannel(t *testing.T) {
	ensureInitialized()
	for _, method := range []string{"isStylusHandwritingAvailable", "isFeatureAvailable"} {
		for _, returnValue := range []any{false, true, nil} {
			t.Run(fmt.Sprint(method, returnValue), func(t *testing.T) {
				runAsync(t, func() {
					var calls []services.MethodCall
					messenger().SetMockMethodCallHandler(services.SystemChannels.Scribe, func(call services.MethodCall) async.Future[any] {
						calls = append(calls, call)
						return async.FutureValue(returnValue)
					})
					defer messenger().SetMockMethodCallHandler(services.SystemChannels.Scribe, nil)
					f := services.Scribe.IsFeatureAvailable
					if method == "isStylusHandwritingAvailable" {
						f = services.Scribe.IsStylusHandwritingAvailable
					}
					if returnValue == nil {
						err := awaitError(f())
						var flutterError *foundation.FlutterError
						expect(errors.As(err, &flutterError), "error %v", err)
					} else {
						got := f().Await()
						expect(got == returnValue, "got %v", got)
					}
					expect(len(calls) == 1 && calls[0].Method == "Scribe."+method, "calls %v", calls)
				})
			})
		}
	}
}

func TestScribeStartStylusHandwriting(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var calls []services.MethodCall
		messenger().SetMockMethodCallHandler(services.SystemChannels.Scribe, func(call services.MethodCall) async.Future[any] {
			calls = append(calls, call)
			return async.FutureValue[any](nil)
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.Scribe, nil)
		services.Scribe.StartStylusHandwriting().Await()
		expect(len(calls) == 1 && calls[0].Method == "Scribe.startStylusHandwriting", "calls %v", calls)
	})
}

func TestLiveText(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var calls []string
		answer := any(nil)
		mock := func(call services.MethodCall) async.Future[any] {
			calls = append(calls, call.Method)
			return async.FutureValue(answer)
		}
		messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, mock)
		messenger().SetMockMethodCallHandler(services.SystemChannels.TextInput, mock)
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.TextInput, nil)
		expect(!services.LiveText.IsLiveTextInputAvailable().Await(), "null is false")
		answer = true
		expect(services.LiveText.IsLiveTextInputAvailable().Await(), "true")
		services.LiveText.StartLiveTextInput().Await()
		expect(slices.Equal(calls, []string{"LiveText.isLiveTextInputAvailable", "LiveText.isLiveTextInputAvailable",
			"TextInput.startLiveTextInput"}), "calls %v", calls)
	})
}

func TestProcessTextQueryTextActionsEmitsCorrectMethodCall(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var log []services.MethodCall
		messenger().SetMockMethodCallHandler(services.SystemChannels.ProcessText, func(call services.MethodCall) async.Future[any] {
			log = append(log, call)
			return async.FutureValue[any](nil)
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.ProcessText, nil)
		actions := services.NewDefaultProcessTextService().QueryTextActions().Await()
		expect(actions != nil && len(actions) == 0, "actions %v", actions)
		expect(len(log) == 1 && log[0].Method == "ProcessText.queryTextActions" && log[0].Arguments == nil, "log %v", log)
	})
}

func TestProcessTextProcessTextActionEmitsCorrectMethodCall(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var log []services.MethodCall
		messenger().SetMockMethodCallHandler(services.SystemChannels.ProcessText, func(call services.MethodCall) async.Future[any] {
			log = append(log, call)
			return async.FutureValue[any](nil)
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.ProcessText, nil)
		services.NewDefaultProcessTextService().ProcessTextAction("fakeActivity.fakeAction", "Flutter", false).Await()
		expect(len(log) == 1 && log[0].Method == "ProcessText.processTextAction" &&
			fmt.Sprint(log[0].Arguments) == "[fakeActivity.fakeAction Flutter false]", "log %v", log)
	})
}

// fakeMessenger answers a channel's method calls with outgoing (Dart's
// FakeProcessTextChannel).
type fakeMessenger struct {
	codec    services.MethodCodec
	outgoing func(call services.MethodCall) any
	calls    []services.MethodCall
}

func (m *fakeMessenger) HandlePlatformMessage(string, []byte, ui.PlatformMessageResponseCallback) async.Future[[]byte] {
	panic(dart.UnimplementedError{})
}
func (m *fakeMessenger) SetMessageHandler(string, services.MessageHandler) {}
func (m *fakeMessenger) Send(_ string, message []byte) async.Future[[]byte] {
	call := m.codec.DecodeMethodCall(message)
	m.calls = append(m.calls, call)
	return async.FutureValue(m.codec.EncodeSuccessEnvelope(m.outgoing(call)))
}

func TestProcessTextHandlesEngineAnswersOverTheChannel(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		const action1ID, action2ID = "fakeActivity.fakeAction1", "fakeActivity.fakeAction2"
		fake := &fakeMessenger{codec: services.StandardMethodCodec{}, outgoing: func(call services.MethodCall) any {
			switch call.Method {
			case "ProcessText.queryTextActions":
				return []dart.MapEntry[string, string]{{Key: action1ID, Value: "Action1"}, {Key: action2ID, Value: "Action2"}}
			case "ProcessText.processTextAction":
				args := call.Arguments.([]any)
				if args[0] == action1ID {
					return args[1].(string) + "!!!"
				}
			}
			return nil
		}}
		service := services.NewDefaultProcessTextService()
		service.SetChannel(services.NewMethodChannel("flutter/processtext", services.MethodChannelOptions{BinaryMessenger: fake}))
		actions := service.QueryTextActions().Await()
		expect(slices.Equal(actions, []services.ProcessTextAction{{ID: action1ID, Label: "Action1"}, {ID: action2ID, Label: "Action2"}}),
			"actions %v", actions)
		processed := service.ProcessTextAction(action1ID, "Flutter", false).Await()
		expect(processed != nil && *processed == "Flutter!!!", "processed %v", processed)
		processed = service.ProcessTextAction(action2ID, "Flutter", false).Await()
		expect(processed == nil, "processed %v", processed)
	})
}

// Not in Flutter's test: a failing query is an empty list.
func TestProcessTextQueryFailureIsEmpty(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		messenger().SetMockMethodCallHandler(services.SystemChannels.ProcessText, func(call services.MethodCall) async.Future[any] {
			panic(services.NewPlatformException("busy"))
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.ProcessText, nil)
		actions := services.NewDefaultProcessTextService().QueryTextActions().Await()
		expect(actions != nil && len(actions) == 0, "actions %v", actions)
	})
}

func TestSpellCheckToString(t *testing.T) {
	span := services.NewSuggestionSpan(ui.NewTextRange(12, 17), []string{"weird"})
	if got := span.String(); got != "SuggestionSpan(range: TextRange(start: 12, end: 17), suggestions: [weird])" {
		t.Error(got)
	}
	results := services.NewSpellCheckResults("i before e except after c is so wierd.", []services.SuggestionSpan{span})
	if got := results.String(); got != "SpellCheckResults(spellCheckText: i before e except after c is so wierd., suggestionSpans: [SuggestionSpan(range: TextRange(start: 12, end: 17), suggestions: [weird])])" {
		t.Error(got)
	}
}

func spanList(spans ...services.SuggestionSpan) string {
	parts := make([]string, len(spans))
	for i, s := range spans {
		parts[i] = s.String()
	}
	return strings.Join(parts, ";")
}

// Own test: fetchSpellCheckSuggestions parses, merges and records results.
func TestDefaultSpellCheckService(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var answer any
		var calls []services.MethodCall
		messenger().SetMockMethodCallHandler(services.SystemChannels.SpellCheck, func(call services.MethodCall) async.Future[any] {
			calls = append(calls, call)
			return async.FutureValue(answer)
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.SpellCheck, nil)
		span := func(start, end int, s ...string) map[string]any {
			list := make([]any, len(s))
			for i, x := range s {
				list[i] = x
			}
			return map[string]any{"startIndex": start, "endIndex": end, "suggestions": list}
		}
		service := services.NewDefaultSpellCheckService()
		answer = []any{span(0, 3, "the"), span(4, 8, "word")}
		got := service.FetchSpellCheckSuggestions(ui.NewLocale("en", "US"), "teh wrod").Await()
		expect(spanList(got...) == "SuggestionSpan(range: TextRange(start: 0, end: 3), suggestions: [the]);SuggestionSpan(range: TextRange(start: 4, end: 8), suggestions: [word])", "got %v", got)
		expect(len(calls) == 1 && fmt.Sprint(calls[0].Arguments) == "[en-US teh wrod]", "calls %v", calls)
		// Same text and same spans: merged with the saved results.
		got = service.FetchSpellCheckSuggestions(ui.NewLocale("en", "US"), "teh wrod").Await()
		expect(len(got) == 2, "got %v", got)
		// A failure (pending request) is nil; the saved results stay.
		answer = "not a list"
		got = service.FetchSpellCheckSuggestions(ui.NewLocale("en"), "teh wrod").Await()
		expect(got == nil && service.LastSavedResults.SpellCheckedText == "teh wrod", "got %v", got)
		merged := services.DefaultSpellCheckServiceMergeResults(
			[]services.SuggestionSpan{services.NewSuggestionSpan(ui.NewTextRange(0, 2), []string{"a"}), services.NewSuggestionSpan(ui.NewTextRange(5, 6), []string{"b"})},
			[]services.SuggestionSpan{services.NewSuggestionSpan(ui.NewTextRange(0, 2), []string{"x"}), services.NewSuggestionSpan(ui.NewTextRange(3, 4), []string{"c"}), services.NewSuggestionSpan(ui.NewTextRange(9, 9), []string{"d"})},
		)
		expect(spanList(merged...) == "SuggestionSpan(range: TextRange(start: 0, end: 2), suggestions: [a]);SuggestionSpan(range: TextRange(start: 3, end: 4), suggestions: [c]);SuggestionSpan(range: TextRange(start: 5, end: 6), suggestions: [b]);SuggestionSpan(range: TextRange(start: 9, end: 9), suggestions: [d])", "merged %v", merged)
	})
}

func TestMouseTrackerAnnotationToString(t *testing.T) {
	annotation1 := services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{
		OnEnter: func(*gestures.PointerEnterEvent) {}, OnExit: func(*gestures.PointerExitEvent) {}})
	if got, want := annotation1.String(), "MouseTrackerAnnotation#"+foundation.ShortHash(annotation1)+"(callbacks: [enter, exit])"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	annotation2 := services.NewMouseTrackerAnnotation()
	if got, want := annotation2.String(), "MouseTrackerAnnotation#"+foundation.ShortHash(annotation2)+"(callbacks: <none>)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	annotation3 := services.NewMouseTrackerAnnotation(services.MouseTrackerAnnotationOptions{
		OnEnter: func(*gestures.PointerEnterEvent) {}, Cursor: services.SystemMouseCursors.Grab})
	if got, want := annotation3.String(), "MouseTrackerAnnotation#"+foundation.ShortHash(annotation3)+"(callbacks: [enter], cursor: SystemMouseCursor(grab))"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !annotation2.ValidForMouseTracker() || annotation2.Cursor() != services.MouseCursorDefer {
		t.Error("defaults")
	}
}

func TestMouseCursorManager(t *testing.T) {
	ensureInitialized()
	runAsync(t, func() {
		var log []string
		messenger().SetMockMethodCallHandler(services.SystemChannels.MouseCursor, func(call services.MethodCall) async.Future[any] {
			log = append(log, call.Method+" "+fmt.Sprint(call.Arguments))
			return async.FutureValue[any](nil)
		})
		defer messenger().SetMockMethodCallHandler(services.SystemChannels.MouseCursor, nil)
		c := services.SystemMouseCursors
		m := services.NewMouseCursorManager(c.Basic)
		update := func(device int, event gestures.PointerEvent, cursors ...services.MouseCursor) {
			m.HandleDeviceCursorUpdate(device, event, slices.Values(cursors))
		}
		update(1, nil, services.MouseCursorDefer, c.Click, c.Text)
		update(1, nil, c.Click)                   // same: nothing
		update(1, nil, services.MouseCursorDefer) // fallback
		update(2, nil, services.MouseCursorUncontrolled)
		update(2, nil, c.Grab)
		update(1, gestures.NewPointerRemovedEvent(), c.Text) // ends the session
		expect(m.DebugDeviceActiveCursor(1) == nil, "removed device has %v", m.DebugDeviceActiveCursor(1))
		update(1, nil, c.Basic)
		expect(m.DebugDeviceActiveCursor(2) == services.MouseCursor(c.Grab), "device 2 %v", m.DebugDeviceActiveCursor(2))
		async.FutureDelayed[any](0, nil).Await()
		expect(slices.Equal(log, []string{
			"activateSystemCursor {device: 1, kind: click}",
			"activateSystemCursor {device: 1, kind: basic}",
			"activateSystemCursor {device: 2, kind: grab}",
			"activateSystemCursor {device: 1, kind: basic}",
		}), "log %q", log)
	})
}

func TestMouseCursorToString(t *testing.T) {
	if got := services.SystemMouseCursors.Click.String(); got != "SystemMouseCursor(click)" {
		t.Error(got)
	}
	if got := services.MouseCursorDefer.String(); got != "defer" {
		t.Error(got)
	}
	if got := services.MouseCursorUncontrolled.String(); got != "uncontrolled" {
		t.Error(got)
	}
	debug := foundation.DiagnosticLevelDebug
	if got := services.SystemMouseCursors.ZoomIn.ToString(foundation.DiagnosticableToStringOptions{MinLevel: &debug}); got != "SystemMouseCursor#"+foundation.ShortHash(services.SystemMouseCursors.ZoomIn)+"(kind: zoomIn)" {
		t.Error(got)
	}
	if !services.SystemMouseCursors.Text.Equal(services.SystemMouseCursors.Text) || services.SystemMouseCursors.Text.Equal(services.SystemMouseCursors.Cell) {
		t.Error("Equal")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("defer created a session")
			}
		}()
		services.MouseCursorDefer.CreateSession(0)
	}()
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a defer fallback was accepted")
			}
		}()
		services.NewMouseCursorManager(services.MouseCursorDefer)
	}()
}
