package difftest

// Go twin of flutter/test/undo_scribe_misc_test.dart: UndoManager, Scribe,
// LiveText, ProcessText and SpellCheck traffic, results and errors.

import (
	"strconv"
	"strings"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestUndoScribeMiscMatchFlutter(t *testing.T) {
	runChannelScenes(t, "undo_scribe_misc", undoScribeMiscScenes)
}

type noReply struct{}

type undoClient struct{ l *channelLog }

func (c undoClient) HandlePlatformUndo(direction services.UndoDirection) {
	c.l.add("handlePlatformUndo " + direction.String())
}
func (undoClient) Undo()         {}
func (undoClient) Redo()         {}
func (undoClient) CanUndo() bool { return true }
func (undoClient) CanRedo() bool { return false }

func showSpans(spans []services.SuggestionSpan) string {
	if spans == nil {
		return "null"
	}
	parts := make([]string, len(spans))
	for i, s := range spans {
		parts[i] = s.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func undoScribeMiscScenes(l *channelLog) {
	replies := map[string][]any{}
	installPlatform := func(channel string, codec services.MethodCodec) {
		testMessenger().SetMockMessageHandler(channel, func(message []byte) async.Future[[]byte] {
			return async.Async(func() []byte {
				call := codec.DecodeMethodCall(message)
				l.add(channel + " " + call.Method + " [" + bytesList(message) + "]")
				var reply any
				if q := replies[call.Method]; len(q) > 0 {
					reply, replies[call.Method] = q[0], q[1:]
				}
				switch r := reply.(type) {
				case *services.PlatformException:
					return codec.EncodeErrorEnvelope(r.Code, services.EncodeErrorEnvelopeOptions{Message: r.Message})
				case noReply:
					return nil
				}
				return codec.EncodeSuccessEnvelope(reply)
			})
		})
	}
	json, standard := services.JSONMethodCodec{}, services.StandardMethodCodec{}
	installPlatform("flutter/undomanager", json)
	installPlatform("flutter/scribe", json)
	installPlatform("flutter/platform", json)
	installPlatform("flutter/textinput", json)
	installPlatform("flutter/processtext", standard)
	installPlatform("flutter/spellcheck", standard)
	showBool := strconv.FormatBool

	// ---- UndoManager.
	l.add("scene undo")
	l.ticks(6)
	services.UndoManager.SetUndoState()
	services.UndoManager.SetUndoState(services.UndoManagerSetUndoStateOptions{CanUndo: true})
	services.UndoManager.SetUndoState(services.UndoManagerSetUndoStateOptions{CanRedo: true})
	services.UndoManager.SetUndoState(services.UndoManagerSetUndoStateOptions{CanUndo: true, CanRedo: true})
	l.add("sent")
	settle()
	services.UndoManager.SetClient(undoClient{l})
	l.add("client " + showBool(services.UndoManager.Client().CanUndo()) + " " + showBool(services.UndoManager.Client().CanRedo()))
	for _, message := range [][]dart.MapEntry[string, any]{
		{{Key: "method", Value: "UndoManagerClient.handleUndo"}, {Key: "args", Value: []any{"undo"}}},
		{{Key: "method", Value: "UndoManagerClient.handleUndo"}, {Key: "args", Value: []any{"redo"}}},
		{{Key: "method", Value: "UndoManagerClient.handleUndo"}, {Key: "args", Value: []any{"sideways"}}},
		{{Key: "method", Value: "UndoManagerClient.other"}, {Key: "args", Value: []any{}}},
	} {
		l.ticks(4)
		testMessenger().HandlePlatformMessage("flutter/undomanager", services.JSONMessageCodec{}.EncodeMessage(message),
			func(reply []byte) { l.add("reply [" + bytesList(reply) + "]") }).Await()
		settle()
	}
	services.UndoManager.SetClient(nil)
	oldOnError := foundation.FlutterErrorOnError
	foundation.FlutterErrorOnError = func(details *foundation.FlutterErrorDetails) {
		library := "null"
		if details.Library != nil {
			library = *details.Library
		}
		l.add("reported " + showValue(details.Exception) + " | " + details.Context.String() + " | " + library)
	}
	failed := "Channel failed"
	replies["UndoManager.setUndoState"] = []any{services.NewPlatformException("error", services.PlatformExceptionOptions{Message: &failed})}
	l.ticks(6)
	services.UndoManager.SetUndoState(services.UndoManagerSetUndoStateOptions{CanUndo: true})
	settle()
	foundation.FlutterErrorOnError = oldOnError

	// ---- Scribe.
	l.add("scene scribe")
	replies["Scribe.isFeatureAvailable"] = []any{true, false, nil}
	replies["Scribe.isStylusHandwritingAvailable"] = []any{false, true, nil}
	for range 3 {
		l.ticks(6)
		trackFuture(l, "isFeatureAvailable", services.Scribe.IsFeatureAvailable(), showBool)
		trackFuture(l, "isStylusHandwritingAvailable", services.Scribe.IsStylusHandwritingAvailable(), showBool)
		settle()
	}
	l.ticks(6)
	l.trackVoid("startStylusHandwriting", services.Scribe.StartStylusHandwriting())
	settle()
	testMessenger().HandlePlatformMessage("flutter/scribe",
		services.JSONMessageCodec{}.EncodeMessage([]dart.MapEntry[string, any]{{Key: "method", Value: "Scribe.unsupportedMessage"}}),
		func(reply []byte) { l.add("scribe reply [" + bytesList(reply) + "]") }).Await()

	// ---- LiveText.
	l.add("scene live text")
	replies["LiveText.isLiveTextInputAvailable"] = []any{nil, true, false}
	for range 3 {
		l.ticks(6)
		trackFuture(l, "isLiveTextInputAvailable", services.LiveText.IsLiveTextInputAvailable(), showBool)
		settle()
	}
	l.ticks(6)
	l.trackVoid("startLiveTextInput", services.LiveText.StartLiveTextInput())
	settle()

	// ---- ProcessText.
	l.add("scene process text")
	service := services.NewDefaultProcessTextService()
	pending := "pending"
	replies["ProcessText.queryTextActions"] = []any{
		[]dart.MapEntry[string, string]{{Key: "com.a.Translate", Value: "Translate"}},
		[]dart.MapEntry[string, string]{{Key: "com.a.Define", Value: "Define"}, {Key: "com.b.Search", Value: "Search"},
			{Key: "com.c.Upper", Value: "UPPER"}},
		// The platform's order, not sorted: Dart lists the actions in it.
		dart.NewLinkedMapOf("com.z.Zap", "Zap", "com.a.Alpha", "Alpha", "com.m.Mid", "Mid", "com.b.Beta", "Beta"),
		[]dart.MapEntry[any, any]{{Key: "com.y.Second", Value: "Second"}, {Key: "com.x.First", Value: "First"}},
		nil,
		services.NewPlatformException("busy", services.PlatformExceptionOptions{Message: &pending}),
		[]any{"not", "a", "map"},
		noReply{},
	}
	showActions := func(actions []services.ProcessTextAction) string {
		parts := make([]string, len(actions))
		for i, a := range actions {
			parts[i] = a.ID + ":" + a.Label
		}
		return strings.Join(parts, ";")
	}
	for range 8 {
		l.ticks(8)
		trackFuture(l, "queryTextActions", service.QueryTextActions(), showActions)
		settle()
	}
	replies["ProcessText.processTextAction"] = []any{"FLUTTER", nil, services.NewPlatformException("x")}
	for _, readOnly := range []bool{false, true, false} {
		l.ticks(6)
		trackFuture(l, "processTextAction", service.ProcessTextAction("com.c.Upper", "flutter 🐦", readOnly), func(s *string) string {
			if s == nil {
				return "null"
			}
			return *s
		})
		settle()
	}

	// ---- SpellCheck.
	l.add("scene spell check")
	spell := services.NewDefaultSpellCheckService()
	span := func(start, end int, suggestions ...string) []dart.MapEntry[string, any] {
		return []dart.MapEntry[string, any]{{Key: "startIndex", Value: start}, {Key: "endIndex", Value: end},
			{Key: "suggestions", Value: suggestions}}
	}
	replies["SpellCheck.initiateSpellCheck"] = []any{
		[]any{span(0, 3, "the", "tea"), span(4, 8, "word")},
		[]any{span(0, 3, "the", "tea"), span(4, 8, "word")},
		[]any{span(2, 3, "x")},
		[]any{},
		services.NewPlatformException("pending"),
		"not a list",
		noReply{},
	}
	script, country := "Hant", "TW"
	for _, req := range []struct {
		locale ui.Locale
		text   string
	}{
		{ui.NewLocale("en", "US"), "teh wrod"},
		{ui.NewLocale("en", "US"), "teh wrod"},
		{ui.NewLocale("pt", "BR"), "teh wrod"},
		{ui.NewLocale("fr"), "bonjour"},
		{ui.NewLocaleFromSubtags(ui.LocaleFromSubtagsOptions{LanguageCode: "zh", ScriptCode: &script, CountryCode: &country}), "x"},
		{ui.NewLocale("en"), "y"},
		{ui.NewLocale("en"), "z"},
	} {
		l.ticks(8)
		trackFuture(l, "fetch", spell.FetchSpellCheckSuggestions(req.locale, req.text), showSpans)
		settle()
		saved := "null"
		if spell.LastSavedResults != nil {
			saved = spell.LastSavedResults.String()
		}
		l.add("saved " + saved)
	}
	s := func(start, end int, w ...string) services.SuggestionSpan {
		word := "w"
		if len(w) > 0 {
			word = w[0]
		}
		return services.NewSuggestionSpan(ui.NewTextRange(start, end), []string{word})
	}
	type spans = []services.SuggestionSpan
	for _, pair := range [][2]spans{
		{{}, {}},
		{{s(0, 1, "old")}, {}},
		{{}, {s(0, 1, "new")}},
		{{s(0, 1, "old"), s(5, 6, "old")}, {s(0, 2, "new"), s(3, 4, "new"), s(9, 9, "new")}},
		{{s(4, 5, "old")}, {s(1, 2, "new"), s(4, 9, "new"), s(7, 8, "new")}},
	} {
		l.add("merge " + showSpans(services.DefaultSpellCheckServiceMergeResults(pair[0], pair[1])))
	}
	l.add("equal " + showBool(s(1, 2).Equal(s(1, 2))) + " " + showBool(s(1, 2).Equal(s(1, 3))) + " " +
		showBool(s(1, 2, "a").Equal(s(1, 2, "b"))))
	l.add("results " + showBool(services.NewSpellCheckResults("t", spans{s(1, 2)}).Equal(services.NewSpellCheckResults("t", spans{s(1, 2)}))) +
		" " + showBool(services.NewSpellCheckResults("t", spans{s(1, 2)}).Equal(services.NewSpellCheckResults("u", spans{s(1, 2)}))))
	l.add(services.NewSpellCheckResults("multi\nline", spans{s(0, 1), s(2, 3, "two words")}).String())
}
